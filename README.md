# ElderCare — информационная система мониторинга здоровья пожилых людей

Магистерская диссертация. Система непрерывного мониторинга здоровья с персонализированной детекцией отклонений и многоуровневой коммуникацией (пациент ↔ врач ↔ родственник).

## Стек

- **Backend:** Go 1.25, Gin, PostgreSQL 16, JWT
- **Frontend:** Next.js 15 (App Router), TypeScript, Tailwind, shadcn/ui, Recharts
- **Dev:** Docker Compose

## Научная новизна

Персональный базовый алгоритм — алерты триггерятся не по статичным порогам, а по отклонениям от индивидуальной скользящей средней и стандартного отклонения пациента.

## Быстрый старт

```bash
cp .env.example .env
make up          # поднять Postgres
make migrate     # накатить схему
make seed        # демо-данные
make backend     # Go API на :8080
make frontend    # Next.js на :3000
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
  backend/           Go API
    cmd/server/      main entrypoint
    cmd/seed/        seed script
    internal/
      auth/          JWT, регистрация, логин
      metrics/       показатели + baseline алгоритм
      medications/   напоминания о лекарствах
      links/         связи пациент-врач-семья
      messages/      чат
      db/            подключение, миграции
      httpx/         общие хелперы
  frontend/          Next.js SPA
  migrations/        SQL-миграции
```

## make check

```bash
make check   # go vet + staticcheck + go test + npm lint + tsc
```
