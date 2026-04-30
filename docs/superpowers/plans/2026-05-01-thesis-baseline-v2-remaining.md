# Thesis Baseline v2 — Remaining Work

This file documents what was deferred from the 2026-05-01 implementation
session of the spec at
`docs/superpowers/specs/2026-05-01-thesis-baseline-v2-design.md`.

## What landed in this session (branch `chore/thesis-baseline-v2-spec`)

- ✅ Migrations 0005 (alerts.reason_code, algorithm_version) and 0006 (algorithm_runs).
- ✅ New `internal/baseline/` Go package with all 6 layers, 36 unit tests
  (preprocessor / 4 estimators / time-window / streak gate / condition profile /
  decision rule / safety / Analyze orchestrator).
- ✅ `metrics` service rewired to call `baseline.Analyze`, persists
  `algorithm_runs` row on every invocation and `alerts.reason_code +
  algorithm_version` on alerts. Old `internal/metrics/baseline.go` deleted.
- ✅ Three new metrics-package integration tests (alert reason code,
  cold-start suppression, condition-profile narrowing).
- ✅ Bug fixes:
  - `auth.generateInviteCode` now returns an error (no silent zero-byte fallback).
  - `Register` retries on invite-code UNIQUE collisions (5 attempts).
  - Email collision still 400; other DB errors still 500.
- ✅ Frontend: `Alert` DTO carries `reason_code` and `algorithm_version`;
  i18n dictionary has 10 reason-code entries × 3 languages (ru/kk/en);
  alerts page renders localized reason with graceful fallback.
- ✅ `cmd/algo-runner` JSON-line bridge binary (used by the Python harness).

All checks pass: `go vet`, `go test -race -p 1 -count=2`, `npx tsc --noEmit`,
`npm run lint`, `npm test`, `next build`.

## What remains (next session)

### 1. Python evaluation harness — `evaluation/`

Spec §4. Expected layout:

```
evaluation/
  pyproject.toml          # uv-managed
  Makefile                # eval | eval-clean | eval-stretch | parity
  README.md
  src/
    __init__.py
    simulator.py          # vitals generator with planted anomalies
    archetypes.py         # 4 personas: healthy_70, hypertension_75, t2d_78, copd_80
    algorithms.py         # adapters → cmd/algo-runner subprocess
    runner.py             # JSON-line driver
    metrics.py            # precision/recall/F1/AUROC/lead-time/false-alarm-rate
    plots.py              # matplotlib figures
    report.py             # generates REPORT.md
  tests/                  # pytest
  data/                   # generated csv (gitignored)
  figures/                # generated png (gitignored)
  results/                # csv summary tables (committed; thesis archive)
  REPORT.md               # committee-ready, generated
  anomaly-spec.yaml       # planted-anomaly definitions
```

Subagent prompt for next session: "Build the Python evaluation harness in
`evaluation/` per spec §4. The Go production binary `backend/cmd/algo-runner`
is built and works — see the smoke tests in this remaining-work file. Use
the JSON schemas already shipped in `backend/internal/baseline/baseline.go`
(Input + Result types). Driver should spawn algo-runner once per algorithm
sweep with a long-lived stdin/stdout pipe."

### 2. Parity fixture + parity test

Spec §6.1. `backend/internal/baseline/testdata/parity_v2.jsonl` — 200
inputs with expected outputs. Generated once by a Go test that emits via
`json.Encoder`; replayed by both Go and Python and compared key-by-key
modulo float tolerance (1e-6 absolute).

### 3. CI changes

Spec §7. `.github/workflows/ci.yml` gains:

- `algorithm-parity` job: builds algo-runner, runs `python -m
  evaluation.runner.parity` against `testdata/parity_v2.jsonl`.
- `evaluation-smoke` job: runs `make eval` in 1-archetype, 100-sample mode,
  asserts `ewma_mad_condition` F1 ≥ 0.5.

### 4. Stretch goal C — BIDMC real-data validation

Spec §4.4. `evaluation/src/realdata.py` adapter for BIDMC PPG Dataset,
gated behind `make eval-stretch`.

## Known bugs documented but out of scope for thesis track

Tracked in `docs/superpowers/known-bugs.md` (file does not exist yet —
create on next touch):

- **medications.go TZ bug**: `start_date <= CURRENT_DATE` fails when the
  server's local TZ is past midnight UTC but PG session is UTC. Reproduced
  on 2026-05-01 at 02:30 Asia/Almaty (PG at 21:30 UTC on prev day) — test
  `TestMedicationsCRUDForPatient` returned 0 schedule items. CI passes
  because GH runners are UTC. Fix: store medications in UTC and use
  `(now() AT TIME ZONE 'utc')::date` consistently, or add a `users.tz`
  column.
- Hard-coded `'ru-RU'` locale in patient dashboard time formatter
  (`app/patient/page.tsx:119`) ignores `lang`.
- JWT in localStorage: XSS risk for medical PII; should migrate to
  HttpOnly cookies.
- No rate-limiting on `/api/auth/login` (brute-force).
- No graceful shutdown on SIGTERM in `cmd/server`.
- "Logout" only clears localStorage — JWT still valid until expiry.
- `metrics.itoa` is a manual reimplementation of `strconv.Itoa`; switch to stdlib.

## How to resume

1. `git checkout chore/thesis-baseline-v2-spec` (this branch).
2. Run `make up` to bring up Postgres.
3. `cd backend && go test -race ./...` — verify the green baseline first.
4. Use the spec at `docs/superpowers/specs/2026-05-01-thesis-baseline-v2-design.md`
   as the contract; this remaining-work file as the index of what's left.
5. Pick up at the Python evaluation harness (Section 1 above).
