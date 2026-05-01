# ElderCare — информационная система мониторинга здоровья пожилых людей

Магистерская диссертация. Система непрерывного мониторинга здоровья с персонализированной детекцией отклонений и многоуровневой коммуникацией (пациент ↔ врач ↔ родственник).

## Стек

- **Backend:** Go 1.25, Gin, PostgreSQL 16, JWT (Bearer + опциональный HttpOnly cookie), Web Push (VAPID), SSE live alerts, per-IP rate-limit, graceful shutdown
- **Mobile:** Flutter 3.41+ (Material 3, GoRouter, Riverpod, Dio + Bearer interceptor, fl_chart, SSE через HTTP chunked stream); 3 роли × ru/kk/en
- **Eval harness:** Python 3.12 (numpy, matplotlib) — гоняет production Go-алгоритм через JSON-line subprocess; синтетический generator + BIDMC real-data адаптер (PhysioNet)
- **Dev:** Docker Compose (Postgres + backend); Flutter SDK для мобилки

## Научная новизна

Персональный базовый алгоритм v2 — алерты триггерятся по 6-слойному пайплайну, а не по статичным порогам:

1. **Hampel preprocessor** — отбраковывает изолированные выбросы из истории, чтобы прошлая аномалия не отравляла baseline.
2. **Estimator** (выбираемый): `MeanStd` | `MedianMAD` | `EWMA` | `EWMA_MAD` (production default).
3. **TimeAwareWindow** — окно за последние 30 дней (cap 60 сэмплов), с временным распадом весов в EWMA.
4. **Stable-streak gate** — персональный baseline активен только при ≥10 замерах за 14 дней, иначе cold-start fallback.
5. **Condition profile** — `chronic_conditions` парсится по ключевым словам (ru/kk/en); расширяет границы для хронических пациентов (Claim C).
6. **Decision rule** — двухпороговая (z≥2 warn, z≥3 critical), направление-чувствительная для unidirectional метрик (SpO2).

Безопасные клинические границы (`SafetyOverride`) всегда срабатывают первыми и шунтируют остальной пайплайн.

**Защищаемые тезисы:**
- **Claim A** — персональный baseline снижает false-alarms по сравнению со статическими порогами без потери чувствительности.
- **Claim C** — condition-aware пороги дополнительно снижают false-alarms у хронических пациентов на ~47% (см. `evaluation/REPORT.md`).

## Быстрый старт

```bash
cp .env.example .env
make up          # поднять Postgres
make migrate     # накатить схему
make seed        # демо-данные
make backend     # Go API на :8080
make mobile      # Flutter run на текущем устройстве (chrome / android emulator)
```

Демо-аккаунты после `make seed`:

| Роль | Email | Пароль |
|---|---|---|
| Пациент | patient@demo.kz | demo1234 |
| Врач | doctor@demo.kz | demo1234 |
| Родственник | family@demo.kz | demo1234 |

## Архитектура

```
elder-care/
  backend/                       Go API
    cmd/server/                  main entrypoint (graceful shutdown)
    cmd/seed/                    seed script (v2 reason codes)
    cmd/algo-runner/             JSON-line bridge for the eval harness
    internal/
      auth/                      JWT, регистрация, логин (rate-limited)
      baseline/                  v2 алгоритм (6 слоёв, 39+ тестов, parity-fixture)
      metrics/                   показатели, alerts, algorithm_runs telemetry
      medications/               напоминания (UTC date math)
      links/                     связи пациент-врач-семья
      messages/                  чат
      plans/                     еженедельное расписание
      db/                        подключение, миграции
      httpx/                     общие хелперы (rate limit, error wrapping)
  mobile/                        Flutter app (Material 3, ru/kk/en, 3 роли)
  evaluation/                    Python harness (simulator, comparators, report)
  docs/superpowers/              specs + plans
```

## make check

```bash
make check   # go vet + go test + flutter analyze + flutter test
```

## Evaluation harness

Защитные графики и таблицы для thesis:

```bash
cd evaluation
make parity   # replay parity_v2.jsonl через Go algo-runner; проверка дрейфа
make smoke    # tiny end-to-end (1 archetype, 100 samples) — CI guard
make eval     # full sweep — все archetypes × metrics × algorithms; пишет REPORT.md и figures/
make clean    # уборка venv и сгенерированных артефактов
```

Свежий REPORT.md и `evaluation/results/eval_full.csv` — committee-ready summary.

## Безопасность и production-hardening

- JWT в HttpOnly cookie (Secure при HTTPS, SameSite=Lax); Bearer header — fallback для скриптовых клиентов.
- Per-IP rate limit 5 req/min на `/api/auth/login` и `/api/auth/register`.
- Graceful shutdown 15s по SIGTERM/SIGINT.
- Web Push на критические alerts через VAPID (передаётся пациенту + всем привязанным caregivers).
- `govulncheck` зелёный (0 уязвимостей в Go-deps).
- Полный аудит: см. `docs/superpowers/AUDIT.md`.
