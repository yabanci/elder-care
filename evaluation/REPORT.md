# ElderCare Baseline v2 — Evaluation Report

_Generated 2026-04-30T21:48:42+00:00_

## Algorithms compared

| ID | Description |
|---|---|
| `static_v1` | v1 static safety thresholds only (Claim A baseline) |
| `mean_std_v1` | v1 mean + population SD over last 30 values (Claim A baseline) |
| `median_mad` | Robust estimator (median + 1.4826·MAD), no condition profile |
| `ewma` | Time-decayed EWMA mean + sample variance, no condition profile |
| `ewma_mad` | Production v2 estimator without condition profile (Claim A main) |
| `ewma_mad_condition` | Production v2 + condition-aware thresholds (Claim C main) |

## F1 by metric (warning threshold)

| metric | static_v1 | mean_std_v1 | median_mad | ewma | ewma_mad | ewma_mad_condition |
|---|---|---|---|---|---|---|
| bp_dia | 0.292 | 0.320 | 0.302 | 0.320 | 0.343 | 0.382 |
| bp_sys | 0.258 | 0.313 | 0.279 | 0.309 | 0.307 | 0.297 |
| glucose | 0.107 | 0.201 | 0.155 | 0.203 | 0.136 | 0.117 |
| pulse | 0.121 | 0.278 | 0.285 | 0.288 | 0.304 | 0.304 |
| spo2 | 0.039 | 0.296 | 0.068 | 0.074 | 0.085 | 0.123 |
| temperature | 0.091 | 0.436 | 0.175 | 0.394 | 0.141 | 0.141 |

## F1 by metric (critical threshold)

| metric | static_v1 | mean_std_v1 | median_mad | ewma | ewma_mad | ewma_mad_condition |
|---|---|---|---|---|---|---|
| bp_dia | 0.167 | 0.350 | 0.071 | 0.100 | 0.125 | 0.125 |
| bp_sys | 0.250 | 0.542 | 0.167 | 0.167 | 0.250 | 0.000 |
| glucose | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| pulse | 0.000 | 0.417 | 0.250 | 0.250 | 0.250 | 0.250 |
| spo2 | 0.000 | 0.325 | 0.000 | 0.000 | 0.000 | 0.000 |
| temperature | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |

## Macro-F1 across all (archetype × metric) combinations

| Algorithm | Macro-F1 |
|---|---|
| `static_v1` | 0.110 |
| `mean_std_v1` | 0.290 |
| `median_mad` | 0.146 |
| `ewma` | 0.175 |
| `ewma_mad` | 0.162 |
| `ewma_mad_condition` | 0.145 |

## Claim C — false-alarm rate on chronic archetypes (lower = better)

| Algorithm | Mean FAR per patient-week |
|---|---|
| `static_v1` | 1.944 |
| `mean_std_v1` | 2.515 |
| `median_mad` | 2.651 |
| `ewma` | 2.534 |
| `ewma_mad` | 2.210 |
| `ewma_mad_condition` | 1.167 |

## Plots

![F1 by algorithm](figures/f1_by_algorithm.png)

![FAR by condition](figures/far_by_condition.png)

![Lead-time](figures/lead_time.png)

## Notes

- Synthetic data: per-archetype literature-grounded means/SDs, diurnal cycles, drift, measurement noise, state-correlated noise. Anomaly types planted: point (4σ), contextual (2.5σ within static safety), collective (3-day drift), inverse (down-side dip).
- Algorithms `static_v1` and `mean_std_v1` are Python re-implementations of v1-as-shipped (no streak gate, population variance) for the ablation comparison. Other algorithms run as the production Go code via `cmd/algo-runner`.
- Cold-start is in effect: when an archetype has < 10 readings in the last 14 days, the algorithm refuses to fire personal-baseline alerts. Safety bounds still apply.

## Stretch goal C — real-data validation (BIDMC)

_BIDMC PPG and Respiration Dataset — 53 ICU recordings; downsampled from 1 Hz to ~8 home-cadence readings per file. Oracle thresholds: HR > 120 / < 50 bpm, SpO2 < 92%._ **Caveat**: ICU cadence ≠ home cadence; results illustrative.

| metric | patients | total events | detected | sensitivity |
|---|---|---|---|---|
| `pulse` | 18 | 8 | 8 | 100.00% |
| `spo2` | 18 | 0 | 0 | n/a |
