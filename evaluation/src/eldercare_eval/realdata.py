"""Real-data adapter: BIDMC PPG and Respiration Dataset (PhysioNet).

PhysioNet's BIDMC dataset contains 53 ICU recordings of ~8 min each,
with HR / Pulse / Respiratory rate / SpO2 sampled at 1 Hz. We
downsample to a home-monitoring cadence (one reading every 4 hours,
sampled by stratified mean within each window) so the v2 algorithm
sees the same shape of input it sees in production.

Event labels (oracle ground truth for evaluation):
- HR    > 120 bpm    → critical (tachycardia)
- HR    < 50 bpm     → critical (bradycardia)
- SpO2  < 92 %       → critical (desaturation)

This is a stretch goal: the cadence mismatch between hospital ICU
recording (1 Hz × 8 min) and home monitoring (4/day) means the eval
is illustrative, not definitive. The slide for defense should be
explicit about this — see `report.py` integration.
"""
from __future__ import annotations

import csv
import statistics
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from pathlib import Path

from .runner import AlgoInput, AlgoRunner, Profile, Reading
from .simulator import AnomalyLabel, AnomalyType, Series

REPO_ROOT = Path(__file__).resolve().parents[3]
BIDMC_DIR = REPO_ROOT / "evaluation" / "data" / "bidmc"

# Map BIDMC CSV columns to our metric kinds. Pulse is the same as HR for
# ElderCare purposes; Resp is not a tracked metric so we ignore it.
COLUMN_MAP = {
    "HR": "pulse",
    "SpO2": "spo2",
}

# Per-metric event thresholds (clinical, used as oracle truth labels).
EVENT_THRESHOLDS = {
    "pulse": {"high": 120.0, "low": 50.0},
    "spo2": {"low": 92.0},
}

# Home-cadence: 4 readings per day = once every 6 hours (strict grid).
HOME_HOURS_PER_SAMPLE = 6.0


@dataclass
class RealdataResult:
    patient_id: str
    metric: str
    n_samples: int
    n_events: int
    severity_thresholds: dict
    detected: int
    missed: int
    sensitivity: float


def _parse_bidmc_csv(path: Path) -> dict[str, list[float]]:
    """Returns {metric: [values...]} sampled at 1 Hz. Drops NaN/inf and
    physiologically impossible values (HR ≤ 0, SpO2 outside [0, 100])."""
    import math
    out: dict[str, list[float]] = {m: [] for m in set(COLUMN_MAP.values())}
    with path.open() as f:
        reader = csv.reader(f)
        header = [h.strip() for h in next(reader)]
        col_to_metric: dict[int, str] = {}
        for idx, name in enumerate(header):
            base = name.split("[")[0].strip()
            if base in COLUMN_MAP:
                col_to_metric[idx] = COLUMN_MAP[base]
        for row in reader:
            for idx, metric in col_to_metric.items():
                if idx >= len(row):
                    continue
                raw = row[idx].strip()
                if not raw or raw.lower() in {"nan", "-nan", "inf", "-inf"}:
                    continue
                try:
                    v = float(raw)
                except ValueError:
                    continue
                if math.isnan(v) or math.isinf(v):
                    continue
                if metric == "spo2" and not (0 <= v <= 100):
                    continue
                if metric == "pulse" and v <= 0:
                    continue
                out[metric].append(v)
    return out


def _resample_to_home_cadence(values: list[float], stride_hz: int) -> list[float]:
    """Mean within each home-cadence bin. stride_hz is the source rate
    (samples per hour); we collapse HOME_HOURS_PER_SAMPLE worth of source
    samples into one home reading. For BIDMC at 1 Hz × 8 min (= 480
    samples), one bin would normally span 6×3600=21600 samples — way more
    than we have. To keep something to work with at this dataset scale,
    we instead split the file into N equal bins and average each."""
    if not values:
        return []
    # Aim for ~8 home readings per file (BIDMC files are 8 min long).
    n_bins = 8
    bin_size = max(1, len(values) // n_bins)
    out = []
    for i in range(n_bins):
        chunk = values[i * bin_size : (i + 1) * bin_size]
        if not chunk:
            continue
        out.append(statistics.fmean(chunk))
    return out


def _label_events(values: list[float], metric: str) -> list[AnomalyLabel]:
    """Identify oracle-truth event windows in the down-sampled series."""
    labels: list[AnomalyLabel] = []
    th = EVENT_THRESHOLDS.get(metric, {})
    for i, v in enumerate(values):
        is_event = False
        if "high" in th and v > th["high"]:
            is_event = True
        if "low" in th and v < th["low"]:
            is_event = True
        if is_event:
            labels.append(AnomalyLabel(AnomalyType.POINT, i, i, "critical"))
    return labels


def _to_series(patient_id: str, metric: str, values: list[float], now: datetime) -> Series:
    """Wrap downsampled values as a Series with synthetic timestamps."""
    timestamps = [
        now - timedelta(hours=HOME_HOURS_PER_SAMPLE * (len(values) - 1 - i))
        for i in range(len(values))
    ]
    anomalies = _label_events(values, metric)
    return Series(
        archetype_id=f"bidmc_{patient_id}",
        metric=metric,
        timestamps=timestamps,
        values=values,
        anomalies=anomalies,
    )


def discover_files() -> list[Path]:
    """Return sorted list of bidmc_*.csv in the local data dir."""
    if not BIDMC_DIR.exists():
        return []
    return sorted(BIDMC_DIR.glob("bidmc_*.csv"))


def _series_to_inputs(s: Series, profile: Profile) -> list[AlgoInput]:
    """Same shape as cli._series_to_inputs but local to avoid cycles."""
    inputs = []
    for i in range(len(s.values)):
        history = [
            Reading(value=s.values[j], measured_at=s.timestamps[j])
            for j in range(max(0, i - 60), i)
        ]
        inputs.append(AlgoInput(
            kind=s.metric, value=s.values[i], history=history,
            profile=profile, now=s.timestamps[i],
        ))
    return inputs


def evaluate_bidmc(runner: AlgoRunner, profile_for_threshold: Profile) -> list[RealdataResult]:
    """Run the v2 algorithm over every available BIDMC file × tracked metric.

    Returns one row per (patient, metric). Empty list if no files.
    """
    files = discover_files()
    if not files:
        return []

    out: list[RealdataResult] = []
    now = datetime(2026, 5, 1, tzinfo=timezone.utc)

    for path in files:
        pid = path.stem.replace("bidmc_", "")
        per_metric = _parse_bidmc_csv(path)
        for metric, raw in per_metric.items():
            if not raw:
                continue
            values = _resample_to_home_cadence(raw, stride_hz=1)
            series = _to_series(pid, metric, values, now)

            inputs = _series_to_inputs(series, profile_for_threshold)
            results = [runner.call(inp) for inp in inputs]

            n_events = len(series.anomalies)
            detected = 0
            for ev in series.anomalies:
                # Detected if any algorithm result in the event window has
                # severity warning or critical. BIDMC event labels (HR>120,
                # SpO2<92) may map to either depending on how far past the
                # threshold the value is — the salient question is "did the
                # alert system fire at all", not "what severity tier".
                if any(results[i].severity in ("warning", "critical")
                       for i in range(ev.start_idx, ev.end_idx + 1)):
                    detected += 1

            sensitivity = (detected / n_events) if n_events > 0 else float("nan")
            out.append(RealdataResult(
                patient_id=pid,
                metric=metric,
                n_samples=len(series.values),
                n_events=n_events,
                severity_thresholds=EVENT_THRESHOLDS.get(metric, {}),
                detected=detected,
                missed=n_events - detected,
                sensitivity=sensitivity,
            ))
    return out


def summarize(rows: list[RealdataResult]) -> str:
    """Markdown summary block, intended to be appended to REPORT.md."""
    if not rows:
        return (
            "## Stretch goal C — real-data validation (BIDMC)\n\n"
            "_No BIDMC files found in `evaluation/data/bidmc/`. Run "
            "`make eval-stretch` after fetching the dataset._\n"
        )
    by_metric: dict[str, list[RealdataResult]] = {}
    for r in rows:
        by_metric.setdefault(r.metric, []).append(r)

    lines = [
        "## Stretch goal C — real-data validation (BIDMC)",
        "",
        "_BIDMC PPG and Respiration Dataset — 53 ICU recordings; "
        "downsampled from 1 Hz to ~8 home-cadence readings per file. "
        "Oracle thresholds: HR > 120 / < 50 bpm, SpO2 < 92%._ "
        "**Caveat**: ICU cadence ≠ home cadence; results illustrative.",
        "",
        "| metric | patients | total events | detected | sensitivity |",
        "|---|---|---|---|---|",
    ]
    for metric, group in sorted(by_metric.items()):
        n_pts = len(group)
        total_ev = sum(r.n_events for r in group)
        total_det = sum(r.detected for r in group)
        sens = (total_det / total_ev) if total_ev > 0 else float("nan")
        sens_str = f"{sens:.2%}" if sens == sens else "n/a"
        lines.append(f"| `{metric}` | {n_pts} | {total_ev} | {total_det} | {sens_str} |")

    return "\n".join(lines) + "\n"
