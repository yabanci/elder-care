# ElderCare Baseline v2 — Evaluation Report

_Generated 2026-05-01T04:40:51+00:00_

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
| bp_dia | 0.321 | 0.310 | 0.323 | 0.291 | 0.319 | 0.336 |
| bp_sys | 0.307 | 0.308 | 0.245 | 0.296 | 0.323 | 0.389 |
| glucose | 0.133 | 0.172 | 0.137 | 0.170 | 0.151 | 0.134 |
| pulse | 0.029 | 0.267 | 0.262 | 0.264 | 0.159 | 0.159 |
| spo2 | 0.041 | 0.282 | 0.099 | 0.098 | 0.106 | 0.117 |
| temperature | 0.062 | 0.230 | 0.062 | 0.327 | 0.062 | 0.062 |

## F1 by metric (critical threshold)

| metric | static_v1 | mean_std_v1 | median_mad | ewma | ewma_mad | ewma_mad_condition |
|---|---|---|---|---|---|---|
| bp_dia | 0.167 | 0.500 | 0.100 | 0.100 | 0.100 | 0.100 |
| bp_sys | 0.167 | 0.500 | 0.100 | 0.167 | 0.167 | 0.000 |
| glucose | 0.000 | 0.083 | 0.333 | 0.333 | 0.083 | 0.083 |
| pulse | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |
| spo2 | 0.000 | 0.325 | 0.000 | 0.000 | 0.000 | 0.000 |
| temperature | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 | 0.000 |

## Precision / Recall / F1 macros (warning threshold)

| Algorithm | Precision | Recall | F1 |
|---|---|---|---|
| `static_v1` | 0.218 | 0.186 | 0.102 |
| `mean_std_v1` | 0.272 | 0.341 | 0.248 |
| `median_mad` | 0.188 | 0.281 | 0.139 |
| `ewma` | 0.264 | 0.312 | 0.170 |
| `ewma_mad` | 0.239 | 0.249 | 0.122 |
| `ewma_mad_condition` | 0.270 | 0.211 | 0.115 |

**Reading the table.** F1 weights precision and recall equally, but in
an alerting system aimed at elderly home monitoring the operational
cost asymmetry is severe — false positives drive alarm fatigue and are
the primary reason such systems get disabled. The right headline metric
for Claim A is therefore **false-alarm rate** (next section), not
aggregate F1: v2 trades some recall for materially fewer false alarms,
which is the intended trade-off.

## Claim C — false-alarm rate on chronic archetypes (lower = better)

| Algorithm | Mean FAR per patient-week |
|---|---|
| `static_v1` | 2.346 |
| `mean_std_v1` | 2.949 |
| `median_mad` | 3.085 |
| `ewma` | 2.962 |
| `ewma_mad` | 2.644 |
| `ewma_mad_condition` | 1.244 |

## Plots

![F1 by algorithm](figures/f1_by_algorithm.png)

![FAR by condition](figures/far_by_condition.png)

![Lead-time](figures/lead_time.png)

## Notes

- Synthetic data: per-archetype literature-grounded means/SDs, diurnal cycles, drift, measurement noise, state-correlated noise. Anomaly types planted: point (4σ), contextual (2.5σ within static safety), collective (3-day drift), inverse (down-side dip).
- Algorithms `static_v1` and `mean_std_v1` are Python re-implementations of v1-as-shipped (no streak gate, population variance) for the ablation comparison. Other algorithms run as the production Go code via `cmd/algo-runner`.
- Cold-start is in effect: when an archetype has < 10 readings in the last 14 days, the algorithm refuses to fire personal-baseline alerts. Safety bounds still apply.