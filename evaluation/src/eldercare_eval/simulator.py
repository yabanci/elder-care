"""Synthetic vitals generator with planted, ground-truth-labeled anomalies.

Each call to `simulate(...)` returns a deterministic time series for one
archetype × metric, plus the indices of injected anomalies (with severity
labels) used as ground truth in evaluation.

Component model per series:
  baseline + diurnal_amp · sin(2π · t/24h) + drift_random_walk + Gaussian_noise
  + state-correlated noise (agitation periods elevate AND noise the readings)
  + planted anomalies of four orthogonal types

Anomaly types (orthogonal so they are separable in the report):
  POINT      single reading 4·σ above baseline; 1 sample
  CONTEXTUAL 2·σ above patient mean but absolute value still under static safety
  COLLECTIVE 3-day drift to 2·σ off baseline (e.g., glucose decompensation)
  INVERSE    SpO2 dip 2·σ below baseline (down-only metric)
"""
from __future__ import annotations

import math
import random
from dataclasses import dataclass
from datetime import datetime, timedelta, timezone
from enum import Enum

from .archetypes import Archetype, MetricSpec
from .runner import Reading


class AnomalyType(str, Enum):
    POINT = "point"
    CONTEXTUAL = "contextual"
    COLLECTIVE = "collective"
    INVERSE = "inverse"


@dataclass(frozen=True)
class AnomalyLabel:
    type: AnomalyType
    start_idx: int
    end_idx: int  # inclusive
    severity_truth: str  # "warning" or "critical" (oracle-defined)


@dataclass
class Series:
    archetype_id: str
    metric: str
    timestamps: list[datetime]
    values: list[float]
    anomalies: list[AnomalyLabel]  # ground truth


def _diurnal(hour_of_day: float, amp: float) -> float:
    # Peak around 14:00, trough around 02:00 — clinically reasonable for BP/HR.
    return amp * math.sin((hour_of_day - 8.0) / 24.0 * 2.0 * math.pi)


def simulate(
    archetype: Archetype,
    metric: str,
    n_days: int,
    seed: int = 42,
    plant_anomalies: bool = True,
    now: datetime | None = None,
) -> Series:
    """Generate one synthetic vitals series for the given archetype and metric.

    Returns a deterministic Series given the same (archetype, metric, n_days, seed).
    """
    spec: MetricSpec = archetype.metrics[metric]
    if now is None:
        now = datetime(2026, 5, 1, tzinfo=timezone.utc)
    rng = random.Random(seed * 10_000 + hash((archetype.id, metric)) % 10_000)

    n_samples = spec.samples_per_day * n_days
    spacing_hours = 24.0 / spec.samples_per_day if spec.samples_per_day else 24.0

    timestamps: list[datetime] = []
    values: list[float] = []

    # Random-walk drift in mean across the simulated period.
    drift = 0.0

    for i in range(n_samples):
        # Time goes from oldest → newest (i=0 is oldest).
        ts = now - timedelta(hours=spacing_hours * (n_samples - 1 - i))
        timestamps.append(ts)
        drift += rng.gauss(0, spec.drift_sigma)
        diurnal_term = _diurnal(ts.hour + ts.minute / 60.0, spec.diurnal_amp)
        noise = rng.gauss(0, spec.std)
        v = spec.mean + diurnal_term + drift + noise
        values.append(v)

    # State-correlated noise: occasional agitation blocks add modest extra
    # noise. Too aggressive a setting would swamp the planted anomalies and
    # give a misleadingly pessimistic eval; too weak hides the failure mode
    # where mean+SD inflates more than robust estimators. 2% probability
    # with σ amplitude is the calibration point for this MVP. Burn-in (first
    # 30 samples) is held clean so the baseline can stabilize.
    block = max(spec.samples_per_day, 1)
    for start in range(30, n_samples, block):
        if rng.random() < 0.02:
            for j in range(start, min(start + block, n_samples)):
                values[j] += rng.gauss(spec.std * 0.3, spec.std * 1.0)

    anomalies: list[AnomalyLabel] = []
    if plant_anomalies and n_samples > 30:
        anomalies = _inject_anomalies(values, spec, rng, n_samples, metric)

    return Series(
        archetype_id=archetype.id,
        metric=metric,
        timestamps=timestamps,
        values=values,
        anomalies=anomalies,
    )


def _inject_anomalies(
    values: list[float],
    spec: MetricSpec,
    rng: random.Random,
    n_samples: int,
    metric: str,
) -> list[AnomalyLabel]:
    """Inject 4 anomaly types at random positions, return their ground-truth labels."""
    labels: list[AnomalyLabel] = []
    used: set[int] = set()

    def free_index(min_idx: int = 30, span: int = 1) -> int:
        for _ in range(50):
            i = rng.randint(min_idx, n_samples - span - 1)
            if not any((i + d) in used for d in range(-span, span + 1)):
                for d in range(span):
                    used.add(i + d)
                return i
        return min_idx

    # Anomaly magnitudes are sized so the algorithm has a fair chance to
    # detect them above measurement + agitation noise (calibrated jointly
    # with the noise levels above). Multiple identical magnitudes across
    # archetypes keep the comparison fair.

    # POINT — single reading well outside personal norm. Replace value
    # outright (rather than add) so the anomaly is unambiguous.
    i = free_index()
    values[i] = spec.mean + 5 * spec.std
    labels.append(AnomalyLabel(AnomalyType.POINT, i, i, "critical"))

    # CONTEXTUAL — 3σ above patient mean but ideally still under any static
    # safety bound (so static thresholds cannot detect it).
    i = free_index()
    values[i] = spec.mean + 3 * spec.std
    labels.append(AnomalyLabel(AnomalyType.CONTEXTUAL, i, i, "warning"))

    # COLLECTIVE — 3-day drift; size depends on cadence. End reaches +3σ.
    span = max(3 * spec.samples_per_day, 3)
    start = free_index(min_idx=30, span=span)
    for j in range(start, min(start + span, n_samples)):
        ramp = (j - start + 1) / span
        values[j] += ramp * 3.0 * spec.std
    labels.append(AnomalyLabel(AnomalyType.COLLECTIVE, start, min(start + span - 1, n_samples - 1), "warning"))

    # INVERSE — low excursion. SpO2 only flags the down side, but other
    # metrics also benefit from low-side detection.
    i = free_index()
    values[i] = spec.mean - 4 * spec.std
    labels.append(AnomalyLabel(AnomalyType.INVERSE, i, i, "warning"))

    return labels
