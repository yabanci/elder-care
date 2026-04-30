# ElderCare Thesis Baseline v2 — Design Spec

**Date:** 2026-05-01
**Status:** Draft (pre-implementation plan)
**Owner:** Arsen Ozhetov (master's thesis defense)
**Successor of:** `backend/internal/metrics/baseline.go::Analyze` (v1, will be removed)

## 1. Goal

Strengthen the personal-baseline algorithm and add a labeled evaluation harness that produces committee-grade evidence for two thesis claims:

- **Claim A** — Personalized rolling baseline reduces false alarms vs. static safety thresholds, without losing sensitivity to clinically meaningful events.
- **Claim C** — Condition-aware adaptive thresholds (parameterized by `users.chronic_conditions`) further reduce false alarms in chronic-disease patients vs. one-size-fits-all baselines.

Both claims are evaluated against synthetic data of literature-grounded distributions; an optional real-data slide (BIDMC PPG) backs the synthetic findings on real noise.

## 2. Non-goals

Out of scope for this spec; tracked separately:

- Multivariate joint detection (BP+pulse correlations).
- Push / email / SMS notifications on alerts.
- Doctor-side medication prescribing.
- Migration of JWT storage from `localStorage` to HttpOnly cookies.
- Production observability stack (Prometheus, structured logs, request IDs).
- Real-time chat / WebSocket.
- Rate-limiting, JWT refresh / revocation.
- Mobile-app polish, accessibility audit beyond existing baseline.

## 3. Architecture

### 3.1 Algorithm — `backend/internal/baseline/`

Replaces `backend/internal/metrics/baseline.go::Analyze`. Old code is deleted (no feature flag — ablations are run in the eval harness, not the production product).

The algorithm is a six-layer pipeline:

```
Input: kind, value, timestamped history, patient profile
  ↓
1. Preprocessor          Hampel filter on history (rejects single-point outliers
                         in the baseline window so a previous spike does not
                         poison the current baseline)
  ↓
2. Estimator             Pluggable: MeanStd | MedianMAD | EWMA | EWMA_MAD
  ↓
3. TimeAwareWindow       Last N days (default 30), capped at K samples (default 60).
                         For EWMA, weights decay by time-delta with 7-day half-life.
  ↓
4. StableStreakGate      Personal mode active iff ≥10 readings in last 14 days;
                         otherwise → cold_start fallback (safety bounds only).
  ↓
5. ConditionProfile      Keyword-parsed from chronic_conditions; modulates safety
                         bounds and z-thresholds. Multi-profile composition takes
                         the narrower bound per metric.
  ↓
6. DecisionRule          Dual threshold (z≥2 → warning, z≥3 → critical).
                         Direction-aware for unidirectional metrics (SpO2 only
                         flags on the low side).
  ↓
SafetyOverride           Absolute clinical bounds always fire (existing v1 behavior,
                         unchanged).
  ↓
Output: severity, reason_code, mean, std, z_score, estimator,
        used_history, history_size, algorithm_version
```

**Production default** in `cmd/server`: estimator `EWMA_MAD`, `condition_aware=true`, `algorithm_version="v2"`. The `MeanStd` estimator is kept as an importable layer (used only in eval ablations).

### 3.2 Estimator semantics

- `MeanStd` — mean + **population** standard deviation (1/N). Faithfully reproduces v1-as-shipped for the ablation comparator. Kept for ablation only.
- `MedianMAD` — median + 1.4826·MAD as σ-equivalent. Robust to historical outliers.
- `EWMA` — exponentially weighted mean with time-aware decay (half-life 7 days). Variance is **sample** (1/(N−1)) to match standard z-score conventions.
- `EWMA_MAD` — EWMA mean + EWMA-of-absolute-deviations. Production default. Sample variance.

### 3.3 Stable-streak gate rationale

The current v1 algorithm activates personal baseline as soon as 5 readings exist. In practice, a new patient logs all 5 during the onboarding session — std collapses to near zero, and any subsequent reading fires a critical alert. The gate (`≥10 readings spread across ≥14 days`) eliminates this cold-start false-alarm storm. Below threshold, the algorithm returns `reason_code=cold_start` and falls back to safety bounds only.

### 3.4 Condition profile

`users.chronic_conditions` is free-text. Parsed case-insensitively for known multilingual keywords (ru / kk / en):

| Profile | Keywords |
|---|---|
| `hypertension` | `гипертония`, `артериальная гипертензия`, `гипертензия`, `гипертония 1`, `гипертония 2`, `hypertension`, `гипертониялық`, `қан қысымы` |
| `t2d` (type 2 diabetes) | `диабет 2`, `сахарный диабет`, `сд2`, `t2d`, `diabetes`, `қант диабеті` |
| `copd` | `хобл`, `copd`, `өкпенің созылмалы` |
| `none` | default |

A patient may match multiple profiles; thresholds compose by taking the **narrower** bound per metric (more sensitive). Initial threshold table (tunable):

| Metric | Default warn high | hypertension | t2d | copd |
|---|---|---|---|---|
| `bp_sys` warn high | 150 | 140 | 145 | — |
| `bp_dia` warn high | 95 | 90 | — | — |
| `glucose` warn high | 10.0 | — | 9.0 | — |
| `glucose` warn low | 4.0 | — | 4.5 | — |
| `spo2` warn low | 93 | — | — | 95 |
| `pulse` warn high | 110 | 105 | — | 115 |

### 3.5 Schema changes (additive, reversible)

**Migration `0005_alert_reason_codes.sql`:**
```sql
ALTER TABLE alerts
    ADD COLUMN reason_code        TEXT NOT NULL DEFAULT 'legacy',
    ADD COLUMN algorithm_version  TEXT NOT NULL DEFAULT 'v1';
CREATE INDEX idx_alerts_reason_code ON alerts(reason_code);
-- Rollback:
-- DROP INDEX idx_alerts_reason_code;
-- ALTER TABLE alerts DROP COLUMN reason_code, DROP COLUMN algorithm_version;
```

**Migration `0006_algorithm_runs.sql`:**
```sql
CREATE TABLE algorithm_runs (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    metric_id          UUID NOT NULL REFERENCES health_metrics(id) ON DELETE CASCADE,
    kind               TEXT NOT NULL,
    value              DOUBLE PRECISION NOT NULL,
    estimator          TEXT NOT NULL,
    mean_used          DOUBLE PRECISION,
    std_used           DOUBLE PRECISION,
    z_score            DOUBLE PRECISION,
    severity           TEXT NOT NULL,
    reason_code        TEXT NOT NULL,
    used_history       BOOLEAN NOT NULL,
    history_size       INTEGER NOT NULL,
    algorithm_version  TEXT NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_algorithm_runs_patient_kind ON algorithm_runs(patient_id, kind, created_at DESC);
-- Rollback:
-- DROP TABLE algorithm_runs;
```

`algorithm_runs` records **every** invocation, not just alerts. Rationale: gives the offline harness a real-world replay corpus, and lets the demo dashboard show "last run did/didn't fire" telemetry in defense.

### 3.6 Reason-code taxonomy

Stable i18n keys, used by the new `alerts.reason_code` column. Frontend `lib/i18n.ts` adds entries in ru / kk / en for each.

| Code | Meaning |
|---|---|
| `safety_below_min` | Absolute clinical minimum violated |
| `safety_above_max` | Absolute clinical maximum violated |
| `safety_warn_low` | Warning band, low side |
| `safety_warn_high` | Warning band, high side |
| `baseline_warn_z2` | Personal baseline z ≥ 2 |
| `baseline_crit_z3` | Personal baseline z ≥ 3 |
| `condition_warn` | Condition-narrowed bound, warn |
| `condition_crit` | Condition-narrowed bound, crit |
| `cold_start` | Informational; baseline not yet active |
| `legacy` | Pre-migration rows (unparsed) |

Replaces inline Russian strings such as `"значительное отклонение от личной нормы (z≥3)"` — those are removed from Go code and live only in the i18n dictionary.

## 4. Evaluation harness — `evaluation/`

Python 3.12, project-managed with `uv`. Lives only in `evaluation/`; no production runtime depends on Python. The Go production code is what's measured — Python only orchestrates and plots.

```
evaluation/
  pyproject.toml              uv-managed
  Makefile                    eval | eval-clean | eval-stretch | parity
  README.md
  src/
    simulator.py              vitals generator with planted anomalies
    archetypes.py             healthy_70 | hypertension_75 | t2d_78 | copd_80
    algorithms.py             algorithm adapters → cmd/algo-runner subprocess
    runner.py                 long-lived JSON-line subprocess driver
    metrics.py                precision/recall/F1/AUROC/lead-time/false-alarm-rate
    plots.py                  matplotlib figures
    report.py                 generates REPORT.md
  tests/                      pytest: simulator validity + adapter smoke
  data/                       generated csv (gitignored)
  figures/                    generated png (gitignored)
  results/                    csv summary tables (committed; thesis archive)
  REPORT.md                   committee-ready summary, generated
```

### 4.1 Algorithm runner — `backend/cmd/algo-runner/`

Thin Go binary, reads JSON-line input from stdin, writes one JSON-line output per call. Reuses `internal/baseline/` directly — no duplicate algorithm code. Python harness spawns it once per algorithm sweep with a persistent stdin/stdout pipe (avoids per-invocation startup cost; ~100k invocations per full eval run).

Input schema:
```json
{"kind": "bp_sys", "value": 168.0, "history": [{"value": 132.1, "measured_at": "2026-04-01T08:00:00Z"}, ...], "profile": ["hypertension"], "estimator": "ewma_mad", "now": "2026-05-01T08:00:00Z"}
```

Output schema:
```json
{"severity": "critical", "reason_code": "baseline_crit_z3", "mean": 131.4, "std": 4.2, "z_score": 8.7, "estimator": "ewma_mad", "used_history": true, "history_size": 47, "algorithm_version": "v2"}
```

### 4.2 Synthetic generator design

Per archetype × metric, generate `D` days of readings at the metric's natural cadence (BP: 3/day morning + evening + bedtime; pulse: hourly when wearable; glucose: 4/day pre-/post-meal; etc.).

Components composed per series:
- **Baseline** — persona-specific mean (literature: AHA/ESC for BP by age/condition, ATP for glucose).
- **Diurnal cycle** — sinusoidal, metric-specific amplitude (BP +10/-5 mmHg AM/night).
- **Slow drift** — random walk with small σ_drift; simulates seasonal change.
- **Measurement noise** — Gaussian, metric-specific σ.
- **State-correlated noise** — agitation periods inject elevated *and* noisier readings. This is the failure mode that makes naive baselines look better than they are; without it, the eval is not credible.

Anomaly injection (4 orthogonal types, ground-truth labels):
- **Point** — single reading 4σ above baseline.
- **Contextual** — 2σ above patient mean but absolute value still within static safety bounds (the case static thresholds miss).
- **Collective** — 3-day drift to 2σ off baseline (e.g., glucose creeping up = decompensation).
- **Inverse** — SpO2 dip 1σ below baseline.

Anomaly *parameters* are committed to `evaluation/anomaly-spec.yaml`; the algorithm under test never sees them. Anomaly *types* used in evaluation include some not used during algorithm tuning (`held_out_anomaly_types: [collective_with_diurnal_phase_shift]`) — defends against the committee critique that the generator was overfit to the algorithm's strengths.

### 4.3 Comparators

| ID | Description | Maps to |
|---|---|---|
| `static` | Current `SafetyLimits` only | Claim A baseline |
| `mean_std` | Old v1 `Analyze` (mean+SD over last 30, no preprocessing, no streak gate) | Claim A baseline |
| `median_mad` | Robust estimator only | Claim A ablation |
| `ewma` | Time-aware EWMA mean+SD | Claim A ablation |
| `ewma_mad` | Production v2 estimator (no condition profile) | Claim A main |
| `ewma_mad_condition` | v2 + condition profile | Claim C main |

Each algorithm runs on each archetype × metric. Aggregated metrics:
- Precision / recall / F1 per metric per algorithm per condition.
- AUROC.
- Mean detection lead-time on collective anomalies (samples between drift onset and first alert).
- False-alarm rate per simulated patient-week.

### 4.4 Stretch goal C — real-data validation

Gated behind `make eval-stretch`. Optional, does not block defense.

Dataset: BIDMC PPG and Respiration Dataset (53 patients, 8-min ICU segments). Adapter downsamples HR / SpO2 to home-reading cadence (every 4-6h via stratified sampling). Event labels: clinical-threshold violations (HR > 120 bpm, SpO2 < 92%) flagged from raw stream. Run `ewma_mad` and `ewma_mad_condition` on resampled streams; report sensitivity. Single defense slide.

**Risk acknowledged in the report**: cadence mismatch (BIDMC is hospital-paced, not home-paced). Do not overclaim.

## 5. Bug fixes in scope (algorithm path only)

Limited to bugs on the algorithm path; everything else goes to backlog.

1. Inline Russian alert reason strings → i18n `reason_code` (closes localization gap; required by Section 3.6).
2. `auth.generateInviteCode` ignores `crypto/rand.Read` error → log + retry.
3. Invite-code `UNIQUE` violation retry loop (5 attempts, then 500). 8-hex collision space is small enough for this to bite.
4. `metrics/baseline.go::meanStd` divides by N (population variance). New estimators (`MedianMAD`, `EWMA`, `EWMA_MAD`) use sample variance (N−1, when N ≥ 2) to match standard z-score conventions. The `MeanStd` estimator kept for ablation **deliberately retains population variance** to faithfully reproduce v1-as-shipped. Documented in v1 → v2 release notes.

Other bugs (medication TZ, hard-coded `ru-RU`, JWT in localStorage, no rate-limit, no graceful shutdown, fake logout) → `docs/superpowers/known-bugs.md`. Not part of this spec.

## 6. Test strategy

### 6.1 Backend (Go)

Per-layer unit tests under `internal/baseline/`:
- `preprocessor_test.go` — Hampel rejects 1-of-N outlier; passes through clean series.
- `estimator_test.go` — golden-output table tests per estimator (committed `testdata/*.json`).
- `time_window_test.go` — date-math off-by-one; samples cap.
- `streak_gate_test.go` — 8 readings → cold_start; 11 readings → personal.
- `condition_test.go` — ru/kk/en keyword tables; multi-profile composition takes narrower bound.
- `decision_test.go` — z thresholds, direction-aware unidirectional.
- `safety_test.go` — existing v1 cases, regression-frozen.

Integration test:
- `analyze_integration_test.go` — `Analyze` end-to-end on a fixture history per archetype, asserts `severity + reason_code + algorithm_version`.

Parity test:
- `testdata/parity_v2.jsonl` — 200 inputs with expected outputs.
- Replayed by `cmd/algo-runner` from the Python harness via `evaluation/src/runner.py::parity()`.
- CI fails if Go and Python disagree on any output.

### 6.2 Python (eval)

- pytest: simulator anomaly types are detectable at oracle thresholds (sanity).
- pytest: each algorithm adapter returns the JSON output schema the harness expects.
- pytest: aggregate-metrics functions on tiny synthetic ground truth.

## 7. CI changes

`.github/workflows/ci.yml` gains:

- `algorithm-parity` job — runs Go test suite, builds `algo-runner`, then runs `python -m evaluation.runner parity` in `evaluation/`. Required for merge.
- `evaluation-smoke` job — runs `make eval` in 1-archetype, 100-sample mode (~10s), asserts `ewma_mad_condition` F1 ≥ 0.5 on the smoke set. Regression guard, not validation.

Defense-grade evaluation (`make eval` in full mode, ~5–10 min) runs locally / in a manual one-off CI job — not on every PR.

## 8. Defense deliverables

1. `evaluation/REPORT.md` — committee-ready summary, generated by `report.py`.
2. Ablation bar chart (CSV + PNG): F1 contribution of each layer (preprocessor / estimator / streak gate / condition profile) on the held-out anomaly set.
3. ROC curves per metric, six algorithms.
4. Detection-lead-time bar chart on collective anomalies, six algorithms.
5. False-alarm rate by condition profile (validates Claim C).
6. Optional: real-data sensitivity slide from `make eval-stretch`.

## 9. Risk register

| Risk | Mitigation |
|---|---|
| Synthetic generator encodes the algorithm's assumptions (circularity attack) | Literature-grounded params; held-out anomaly types not used during tuning; explicit ablation table |
| Cross-language Go/Python parity drift | Parity test in CI; 200-row fixture |
| Defense-date scope creep | Stretch goal gated behind a separate Make target, droppable without touching main contributions |
| Patient timezone assumed Almaty in medication "today" | Documented; deferred (not algorithm-path) |
| Condition keyword parser misses a synonym | Test fixtures cover ru/kk/en common variants; profile defaults to `none` on miss (safe fallback) |
| `cold_start` patients get no personal alerts at all | Safety overrides still fire; `cold_start` is a labeled state in telemetry, not silence |
| `algorithm_runs` table grows unbounded | Retention policy: `DELETE WHERE created_at < now() - interval '90 days'` in a future migration; out of scope here |
| Sample-vs-population variance change shifts v1 ablation numbers | Numbers are reported per algorithm; the comparison is internally consistent |

## 10. Open questions

None at spec-write time; the spec is self-contained. Implementation may surface tuning questions (exact half-life for EWMA, exact streak-gate counts) — those are tuned on the synthetic ablation set and reported, not pre-decided.

## 11. Implementation milestones (planning hint, not the plan)

The implementation plan will be drafted next via the writing-plans skill. Likely milestone shape:

1. New `internal/baseline/` package with all six layers + tests.
2. Migrations 0005, 0006.
3. `metrics.create` rewired to use new package; old `Analyze` removed.
4. Frontend i18n dictionary entries for reason codes.
5. `cmd/algo-runner` binary + parity fixture.
6. `evaluation/` Python project scaffolded; simulator + algorithm adapters.
7. Eval metrics + plots + REPORT.md generation.
8. CI jobs.
9. Stretch: real-data adapter for BIDMC.
10. Bug fixes 5.2–5.4 (small, can land alongside 1).

## 12. References (to be cited in thesis)

- AHA / ACC 2017 Hypertension Guidelines (BP norms by age, condition).
- ATP / GINA (glucose target ranges by diabetes type).
- BIDMC PPG and Respiration Dataset, PhysioNet (real-data stretch).
- Hampel, F. R. (1974). The influence curve and its role in robust estimation.
- Roberts, S. (1959). Control chart tests based on geometric moving averages (EWMA).
