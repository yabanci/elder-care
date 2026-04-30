# ElderCare Evaluation Harness

Offline evaluation rig for the v2 personal-baseline algorithm. Generates
synthetic vitals with planted anomalies, runs the production Go algorithm
via the `cmd/algo-runner` JSON-line bridge, and reports precision /
recall / F1 / AUROC / detection lead-time / false-alarm-rate per metric
and per algorithm comparator.

The Python here only orchestrates — the algorithm being measured is the
real Go code under `backend/internal/baseline/`. Defends against the
"prototype isn't your product" critique.

## Quick start

```bash
make parity   # replay parity_v2.jsonl through the Go binary; CI-safe
make smoke    # tiny end-to-end run (1 archetype, 100 samples) — F1 regression guard
make eval     # full sweep; writes results/ and figures/
make clean    # nuke venv + outputs
```

Outputs:

- `results/*.csv` — committed; thesis archive.
- `figures/*.png` — gitignored; regenerated.
- `REPORT.md` — committee summary, regenerated.

## Layout

```
evaluation/
  pyproject.toml
  requirements.txt
  Makefile
  src/eldercare_eval/
    __init__.py
    archetypes.py        # 4 patient personas (healthy_70 / hyper_75 / t2d_78 / copd_80)
    simulator.py         # vitals generator with planted anomalies
    runner.py            # JSON-line subprocess driver around algo-runner
    algorithms.py        # 6 comparators (static / mean_std / median_mad / ewma / ewma_mad / ewma_mad_condition)
    metrics.py           # precision / recall / F1 / AUROC / lead-time / FAR
    plots.py             # matplotlib figures
    report.py            # generates REPORT.md
    parity.py            # CLI: replay parity_v2.jsonl, exit non-zero on drift
    cli.py               # CLI: smoke | eval | eval-stretch
  tests/                 # pytest unit tests
  data/                  # generated CSV (gitignored)
  figures/               # generated PNG (gitignored)
  results/               # committed summary CSVs
  REPORT.md              # committee-ready, generated
```

## CI

Two jobs in `.github/workflows/ci.yml`:

- `algorithm-parity` — runs `make parity`. Required on every PR.
- `evaluation-smoke` — runs `make smoke`. Required on every PR.

Full `make eval` is *not* in CI — too long; run locally before defense.
