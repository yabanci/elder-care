"""Top-level CLI: smoke | eval | eval-stretch.

Usage:
    python -m eldercare_eval.cli smoke         # 1 archetype, 100 samples — CI-safe
    python -m eldercare_eval.cli eval          # full sweep — all archetypes × metrics × algorithms
    python -m eldercare_eval.cli eval-stretch  # currently a stub; future BIDMC validation
"""
from __future__ import annotations

import csv
import sys
from dataclasses import asdict
from datetime import datetime, timezone
from pathlib import Path

from .algorithms import ALGORITHMS_GO, ALGORITHMS_PY
from .archetypes import ALL_ARCHETYPES, EVALUATED_METRICS, HEALTHY_70
from .metrics import EvalRow, evaluate
from .plots import f1_by_algorithm, far_by_condition, lead_time
from .report import write as write_report
from .runner import AlgoInput, AlgoRunner, algo_runner
from .simulator import Series, simulate

REPO_ROOT = Path(__file__).resolve().parents[3]
RESULTS_DIR = REPO_ROOT / "evaluation" / "results"
FIGURES_DIR = REPO_ROOT / "evaluation" / "figures"
REPORT_PATH = REPO_ROOT / "evaluation" / "REPORT.md"


def _series_to_inputs(s: Series, profile, history_window: int = 60) -> list[AlgoInput]:
    """Convert a series into one AlgoInput per sample, where history is the
    preceding values (excluding the current). Mirrors the production
    behaviour: a metric is recorded, the algorithm sees prior samples + the
    new value."""
    from .runner import Reading
    inputs = []
    for i in range(len(s.values)):
        start = max(0, i - history_window)
        history = [
            Reading(value=s.values[j], measured_at=s.timestamps[j])
            for j in range(start, i)
        ]
        inputs.append(AlgoInput(
            kind=s.metric,
            value=s.values[i],
            history=history,
            profile=profile,
            now=s.timestamps[i],
        ))
    return inputs


def _run_python_algo(algo_id: str, inputs: list[AlgoInput]):
    fn = ALGORITHMS_PY[algo_id]
    return [fn(inp) for inp in inputs]


def _evaluate_one(
    runner: AlgoRunner,
    archetype,
    metric: str,
    n_days: int,
    seed: int,
) -> list[EvalRow]:
    series = simulate(archetype, metric, n_days=n_days, seed=seed)
    spec = archetype.metrics[metric]

    rows: list[EvalRow] = []

    # Python comparators (v1 baselines).
    inputs_default = _series_to_inputs(series, profile=archetype.profile)
    inputs_no_profile = _series_to_inputs(series, profile=type(archetype.profile)())  # empty profile

    for algo_id in ALGORITHMS_PY:
        results = _run_python_algo(algo_id, inputs_no_profile)
        for threshold in ("warning", "critical"):
            rows.append(evaluate(series, results, algo_id, threshold, spec.samples_per_day))

    # Go comparators.
    for algo in ALGORITHMS_GO:
        inputs = inputs_default if algo.use_condition else inputs_no_profile
        # algo.run mutates estimator/profile internally; we just call it.
        results = [algo.run(runner, inp) for inp in inputs]
        for threshold in ("warning", "critical"):
            rows.append(evaluate(series, results, algo.id, threshold, spec.samples_per_day))

    return rows


def _write_results_csv(rows: list[EvalRow], path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", newline="") as f:
        if not rows:
            return
        w = csv.DictWriter(f, fieldnames=list(asdict(rows[0]).keys()))
        w.writeheader()
        for r in rows:
            w.writerow(asdict(r))


def _ensure_dirs() -> None:
    RESULTS_DIR.mkdir(parents=True, exist_ok=True)
    FIGURES_DIR.mkdir(parents=True, exist_ok=True)


def cmd_smoke(args: list[str]) -> int:
    """Tiny end-to-end run for CI regression guard."""
    _ensure_dirs()
    rows: list[EvalRow] = []
    with algo_runner() as runner:
        for metric in ("pulse", "bp_sys", "glucose", "spo2"):
            rows.extend(_evaluate_one(runner, HEALTHY_70, metric, n_days=25, seed=1))

    _write_results_csv(rows, RESULTS_DIR / "smoke.csv")
    f1 = {}
    for r in rows:
        if r.threshold == "warning":
            f1.setdefault(r.algorithm, []).append(r.f1)
    avg = {k: (sum(v) / len(v) if v else 0.0) for k, v in f1.items()}
    print("smoke F1 (warning):", {k: f"{v:.2f}" for k, v in avg.items()})

    # Smoke is a regression guard, not validation. We require non-trivial F1
    # on at least one v2 algorithm (so we know the pipeline produces
    # meaningful detections) AND that v2 strictly beats static_v1.
    SMOKE_MIN_F1 = 0.10
    target = max(avg.get("ewma_mad", 0.0), avg.get("ewma_mad_condition", 0.0))
    static_baseline = avg.get("static_v1", 0.0)
    if target < SMOKE_MIN_F1:
        print(f"FAIL: best v2 F1={target:.3f} below smoke floor {SMOKE_MIN_F1}")
        return 1
    if target < static_baseline:
        print(f"FAIL: best v2 F1={target:.3f} did not beat static_v1={static_baseline:.3f}")
        return 1
    print(f"PASS: best v2 F1={target:.3f} beats floor and static_v1={static_baseline:.3f}")
    return 0


def cmd_eval(args: list[str]) -> int:
    """Full evaluation: all archetypes × all evaluated metrics × all algorithms."""
    _ensure_dirs()
    n_days = 60
    rows: list[EvalRow] = []
    started = datetime.now(timezone.utc)

    with algo_runner() as runner:
        for archetype in ALL_ARCHETYPES:
            for metric in EVALUATED_METRICS:
                if metric == "weight":
                    continue  # no clinically-meaningful baseline thresholds
                rows.extend(_evaluate_one(runner, archetype, metric, n_days=n_days, seed=42))

    elapsed = (datetime.now(timezone.utc) - started).total_seconds()
    print(f"completed full eval in {elapsed:.1f}s; {len(rows)} rows")

    _write_results_csv(rows, RESULTS_DIR / "eval_full.csv")
    f1_by_algorithm(rows, FIGURES_DIR / "f1_by_algorithm.png")
    far_by_condition(rows, FIGURES_DIR / "far_by_condition.png")
    lead_time(rows, FIGURES_DIR / "lead_time.png")
    write_report(rows, REPORT_PATH, FIGURES_DIR)
    print(f"wrote {REPORT_PATH}")
    return 0


def cmd_eval_stretch(args: list[str]) -> int:
    """Stretch: real-data validation on BIDMC PPG and Respiration Dataset.

    Looks for `bidmc_*.csv` files in `evaluation/data/bidmc/`. If none
    exist, prints download instructions and exits 0 (non-blocking).
    Appends the summary block to REPORT.md so the section appears next
    to Claim A and Claim C results.
    """
    from . import realdata

    files = realdata.discover_files()
    if not files:
        print("No BIDMC files found at evaluation/data/bidmc/.")
        print("Fetch with:")
        print("  mkdir -p evaluation/data/bidmc && \\")
        print("  for i in 01 02 03 04 05; do \\")
        print("    curl -sL 'https://physionet.org/files/bidmc/1.0.0/bidmc_csv/bidmc_${i}_Numerics.csv' \\")
        print("      -o evaluation/data/bidmc/bidmc_${i}.csv; done")
        return 0

    _ensure_dirs()
    from .runner import Profile
    with algo_runner() as runner:
        rows = realdata.evaluate_bidmc(runner, profile_for_threshold=Profile())

    print(f"BIDMC eval: {len(rows)} (patient, metric) rows")
    summary = realdata.summarize(rows)

    # Append (or replace if already present) the BIDMC section in REPORT.md.
    existing = REPORT_PATH.read_text() if REPORT_PATH.exists() else ""
    marker = "## Stretch goal C — real-data validation (BIDMC)"
    if marker in existing:
        # Replace from marker to end of file (BIDMC always last section).
        existing = existing.split(marker)[0]
    REPORT_PATH.write_text(existing.rstrip() + "\n\n" + summary)
    print(f"appended BIDMC summary to {REPORT_PATH}")
    return 0


def main(argv: list[str] | None = None) -> int:
    argv = argv if argv is not None else sys.argv[1:]
    if not argv:
        print(__doc__)
        return 2
    cmd, *rest = argv
    table = {"smoke": cmd_smoke, "eval": cmd_eval, "eval-stretch": cmd_eval_stretch}
    if cmd not in table:
        print(f"unknown command: {cmd}", file=sys.stderr)
        print(__doc__, file=sys.stderr)
        return 2
    return table[cmd](rest)


if __name__ == "__main__":
    raise SystemExit(main())
