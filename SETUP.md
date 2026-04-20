# ElderCare — руководство по установке и запуску

Подробная инструкция по поднятию системы на локальной машине, разворачиванию демо-данных и решению типовых проблем.

---

## 1. Требования

| Компонент | Версия | Зачем |
|---|---|---|
| **Go** | ≥ 1.25 | бэкенд |
| **Node.js** | ≥ 20 | фронтенд |
| **npm** | ≥ 10 | пакетный менеджер фронта |
| **Docker Desktop** | любая свежая | PostgreSQL в контейнере |
| **Docker Compose** | plugin (v2) | входит в Docker Desktop |
| **make** | любая | удобные команды |
| **git** | ≥ 2.30 | клонирование |

### Проверить версии

```bash
go version        # go1.25.x
node --version    # v20+ (или v22)
npm --version     # 10+
docker --version  # Docker version 24+
docker compose version
make --version
```

### macOS (Homebrew)

```bash
brew install go node docker make
# или Docker Desktop:  https://www.docker.com/products/docker-desktop/
```

### Ubuntu/Debian

```bash
sudo apt update
sudo apt install -y make nodejs npm
# Go:  https://go.dev/dl/
# Docker: https://docs.docker.com/engine/install/
```

---

## 2. Клонирование и структура

```bash
git clone <URL> elder-care
cd elder-care
```

Структура репозитория:

```
elder-care/
├── backend/                 Go API (порт 8090)
│   ├── cmd/
│   │   ├── server/          основной HTTP-сервер
│   │   └── seed/            скрипт демо-данных
│   ├── internal/
│   │   ├── auth/            JWT, регистрация, логин
│   │   ├── config/          загрузка .env
│   │   ├── db/              подключение к Postgres + миграции
│   │   │   └── migrations/  *.sql (0001_init.sql и т. д.)
│   │   ├── httpx/           общие error-хелперы
│   │   ├── links/           связи пациент ↔ врач/родственник
│   │   ├── medications/     лекарства + расписание
│   │   ├── messages/        чат
│   │   └── metrics/         показатели + baseline-алгоритм
│   ├── go.mod
│   └── go.sum
├── frontend/                Next.js 14 (порт 3100)
│   ├── app/                 роуты App Router
│   │   ├── login/           /login
│   │   ├── register/        /register
│   │   ├── patient/         панель пациента
│   │   └── care/            панель врача/родственника
│   ├── components/          Shell, AuthGate, карточки, графики
│   ├── lib/                 api.ts, metric-meta.ts
│   ├── package.json
│   └── tailwind.config.ts
├── docker-compose.yml       Postgres 16
├── Makefile                 команды up/migrate/seed/backend/frontend/check
├── .env.example             шаблон переменных окружения
├── README.md                обзор
└── SETUP.md                 этот файл
```

---

## 3. Настройка окружения

### 3.1 Создать `.env` из шаблона

```bash
cp .env.example .env
```

Содержимое `.env` (достаточно для локального запуска):

```ini
POSTGRES_USER=eldercare
POSTGRES_PASSWORD=eldercare_dev
POSTGRES_DB=eldercare
POSTGRES_HOST=localhost
POSTGRES_PORT=5433

DATABASE_URL=postgres://eldercare:eldercare_dev@localhost:5433/eldercare?sslmode=disable

JWT_SECRET=change_me_in_prod_please_use_long_random_string
JWT_TTL_HOURS=168

SERVER_ADDR=:8090
CORS_ORIGIN=http://localhost:3100

NEXT_PUBLIC_API_URL=http://localhost:8090
```

**Что здесь настраивается:**

- `POSTGRES_PORT=5433` — нестандартный порт, чтобы не конфликтовать с локально установленным Postgres, если он у вас есть.
- `SERVER_ADDR=:8090` — API слушает на 8090 (не 8080, потому что 8080 часто занят Docker Desktop).
- `CORS_ORIGIN=http://localhost:3100` — фронт на 3100 (3000 часто занят другими проектами/Docker).
- `JWT_SECRET` — **в продакшене обязательно заменить** на длинную случайную строку (например `openssl rand -hex 48`).

### 3.2 (Опционально) `.env.local` для фронта

Если нужно переопределить API URL только для фронта, создайте `frontend/.env.local`:

```bash
cp frontend/.env.local.example frontend/.env.local
```

---

## 4. Запуск — быстрый путь

Всё в одну цепочку, из корня репозитория:

```bash
make up          # поднимает Postgres в Docker
make seed        # накатывает миграции + заполняет демо-данными
make backend &   # стартует Go API на :8090 (в фоне)
make frontend    # стартует Next.js на :3100 (эта команда блокирующая)
```

Откройте **http://localhost:3100/login** и войдите одним из демо-аккаунтов:

| Роль | Email | Пароль | Что увидите |
|---|---|---|---|
| Пациент | `patient@demo.kz` | `demo1234` | дашборд с 7 метриками, лекарства, оповещения, чат |
| Врач | `doctor@demo.kz` | `demo1234` | список пациентов, их графики, активные алерты |
| Родственник | `family@demo.kz` | `demo1234` | то же что и врач, но в роли «семья» |

**Демо-пациент имеет invite-код `ELDER001`** — если вы зарегистрируетесь новым врачом/родственником, используйте этот код на странице «Добавить» для подключения.

---

## 5. Запуск — пошаговый путь

Если `make` недоступен или нужна тонкая настройка.

### 5.1 Поднять Postgres

```bash
docker compose up -d db
```

Проверить, что контейнер здоров:

```bash
docker compose ps
# STATUS должен быть "Up ... (healthy)"

docker compose exec db pg_isready -U eldercare
# accepting connections
```

### 5.2 Накатить миграции

Миграции лежат в `backend/internal/db/migrations/` (встроены в бинарник через `//go:embed`) и применяются автоматически при старте сервера. Чтобы применить их **отдельно**:

```bash
cd backend
go run ./cmd/server --migrate-only
```

### 5.3 Заполнить демо-данными

```bash
cd backend
go run ./cmd/seed
```

Что делает seed:

- **Очищает** все таблицы (это dev-утилита, не запускать в проде).
- Создаёт трёх пользователей: пациент, врач, родственник.
- Связывает врача и родственника с пациентом.
- Генерирует **30 дней истории** по 7 метрикам (пульс, АД систолическое/диастолическое, глюкоза, температура, вес, SpO₂).
- Вставляет **преднамеренные выбросы 2 дня назад**, чтобы сработал baseline-алгоритм и создались критические алерты.
- Создаёт 3 лекарства с расписанием приёма.
- Вставляет 4 демо-сообщения между врачом, пациентом и родственником.

### 5.4 Запустить backend

```bash
cd backend
go run ./cmd/server
# server listening on :8090
```

Проверка:

```bash
curl http://localhost:8090/health
# {"status":"ok"}
```

### 5.5 Запустить frontend

В **новом терминале**:

```bash
cd frontend
npm install          # в первый раз
npm run dev          # next dev -p 3100
```

Откройте **http://localhost:3100**.

---

## 6. Что делать с системой

### Как пациент

1. Войти под `patient@demo.kz` / `demo1234`.
2. На главной увидеть последние показатели, сегодняшние лекарства, активные оповещения.
3. Через «Быстрый ввод» добавить новое измерение (пульс, АД, глюкоза, температура). Если значение выбивается из вашей личной нормы более чем на 2σ — появится алерт.
4. Нажать «Принял ✓» на плашке лекарства, чтобы залогировать приём.
5. Во вкладке «Оповещения» посмотреть и подтвердить алерты.
6. В «Чат» написать врачу или родственнику.
7. Свой **invite-код** показывается внизу главной — сообщите его близким для подключения.

### Как врач / родственник

1. Войти под `doctor@demo.kz` или `family@demo.kz`.
2. На главной — список подключённых пациентов.
3. Открыть пациента — увидеть его показатели, графики динамики, активные алерты.
4. Вкладка «Оповещения» — консолидированная лента по всем пациентам.
5. Чтобы подключить нового пациента — вкладка «Добавить», ввести его invite-код.

### Зарегистрировать нового пациента и подключиться к нему

1. В другом браузере (или incognito) откройте `/register` → выберите роль «Пациент» → заполните форму.
2. На главной новый пациент увидит свой invite-код (8 символов в верхнем регистре).
3. Вернитесь в первый браузер под `doctor@demo.kz` → «Добавить» → введите этот код → готово.

---

## 7. Проверка качества кода

```bash
make check
```

Что запускается:

- `go vet ./...` — статика Go
- `go test ./...` — юнит-тесты бэкенда (в том числе тесты baseline-алгоритма)
- `npm run lint` — ESLint фронтенда
- `npx tsc --noEmit` — типовая проверка TypeScript

Все четыре должны проходить зелёными. Если нет — коммит не пушить.

---

## 8. API — основные эндпоинты

Базовый URL: `http://localhost:8090`

| Метод | Путь | Роль | Описание |
|---|---|---|---|
| POST | `/api/auth/register` | публичный | регистрация |
| POST | `/api/auth/login` | публичный | логин, возвращает JWT |
| GET | `/api/me` | любая | текущий пользователь |
| GET | `/api/metrics/summary` | patient | последнее значение каждой метрики |
| GET | `/api/metrics` | patient | история (опц. `?kind=pulse`) |
| POST | `/api/metrics` | patient | добавить измерение + триггер baseline |
| GET | `/api/alerts` | patient | список алертов |
| POST | `/api/alerts/:id/acknowledge` | patient | подтвердить алерт |
| GET | `/api/medications/today` | patient | расписание лекарств на сегодня |
| POST | `/api/medications/:id/log` | patient | отметить приём (taken/missed/skipped) |
| GET | `/api/caregivers` | patient | список подключённых врачей/родни |
| GET | `/api/patients` | doctor/family | список пациентов |
| POST | `/api/patients/link` | doctor/family | подключиться по invite-коду |
| GET | `/api/patients/:id/metrics` | doctor/family | метрики пациента |
| GET | `/api/patients/:id/alerts` | doctor/family | алерты пациента |
| POST | `/api/messages` | все связанные | отправить сообщение |
| GET | `/api/messages/:otherID` | все связанные | переписка |

Все приватные эндпоинты требуют `Authorization: Bearer <token>`.

Пример с curl:

```bash
TOKEN=$(curl -s -X POST http://localhost:8090/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"patient@demo.kz","password":"demo1234"}' \
  | jq -r .token)

curl -H "Authorization: Bearer $TOKEN" http://localhost:8090/api/alerts | jq
```

---

## 9. Научная новизна — baseline-алгоритм

Ключевая фича для защиты диссертации. Логика в `backend/internal/metrics/baseline.go`, тесты — `baseline_test.go`.

### Как работает

При добавлении нового измерения:

1. Сначала проверяются **абсолютные медицинские границы** (`SafetyLimits` в `baseline.go`). Например, пульс > 140 или < 40 → немедленно critical независимо от истории.
2. Если абсолютных триггеров нет и у пациента ≥ 5 предыдущих измерений той же метрики — считается **личный baseline** (скользящие mean и stddev по последним 30 значениям).
3. Считается z-score: `z = |value - mean| / std`.
4. Правила severity:
   - `z < 2` — норма, алерт не создаётся;
   - `2 ≤ z < 3` — **warning** («отклонение от личной нормы»);
   - `z ≥ 3` — **critical** («значительное отклонение от личной нормы»).

### Почему это важно

Дефолтные системы мониторинга срабатывают только на универсальные пороги (АД > 140/90 и т. д.). Они **пропускают индивидуальные изменения** — например, у гипотоника АД 125 — это критический скачок, а у гипертоника норма. Персональный baseline видит **относительную динамику** и ловит ранние признаки ухудшения.

### Как воспроизвести в demo

После `make seed` зайдите под `patient@demo.kz` → вкладка «Оповещения» — увидите 5-6 алертов с пояснением типа «значительное отклонение от личной нормы (z≥3)» и сравнением `Значение: 37.9 °C · Норма ≈ 36.6 ± 0.3`.

---

## 10. Траблшутинг

### `EADDRINUSE: address already in use :::3100` (или :::8090)

Порт занят. Найти процесс:

```bash
lsof -i :3100 -P
lsof -i :8090 -P
```

Либо убить его, либо сменить порт в `.env` (`SERVER_ADDR`, `CORS_ORIGIN`, `NEXT_PUBLIC_API_URL`) и в `frontend/package.json` (`next dev -p <новый>`).

### «Загружаем…» висит на главной / 404 на `/_next/static/chunks/...`

Бывает когда вы запустили `npm run build` параллельно с работающим `npm run dev` — build перезаписывает кеш dev-сервера.

```bash
cd frontend
# остановить оба процесса
rm -rf .next
npm run dev
```

И обновить страницу через **Cmd/Ctrl + Shift + R**.

### `missing required env: DATABASE_URL`

Забыли `cp .env.example .env` или запускаете `go run` из неправильной директории. `cmd/server` ищет `.env` сначала в `../` (корень репо), потом в текущей папке.

### Postgres не стартует (`docker compose up -d db` висит)

```bash
docker compose down -v    # удалить volume
docker compose up -d db
docker compose logs -f db
```

Если Docker Desktop не запущен — откройте его вручную.

### `recharts` / React peer-deps конфликт при `npm install`

Стек зафиксирован на Next 14 + React 18 именно чтобы recharts ставился без флагов. Если видите ошибку ERESOLVE — проверьте, что в `frontend/package.json` **не появилось** `react@19-rc`. Если появилось (например, после ручного обновления):

```bash
cd frontend
rm -rf node_modules package-lock.json
# восстановите версии из git
git checkout package.json
npm install
```

### JWT expired / 401 на `/api/me`

Токен живёт `JWT_TTL_HOURS` часов (дефолт 168 = 7 дней). Если истёк — фронт автоматически чистит localStorage и редиректит на `/login`. Если зациклилось на `/login` — откройте DevTools → Application → Local Storage → очистите вручную.

### База «засеялась», но алертов нет

Проверить:

```bash
docker compose exec db psql -U eldercare -d eldercare -c "SELECT kind, severity, COUNT(*) FROM alerts GROUP BY kind, severity;"
```

Должно быть 5-6 алертов. Если 0 — seed не отработал до конца (ищите ошибки в выводе `go run ./cmd/seed`).

---

## 11. Остановка и очистка

```bash
# Ctrl+C в терминалах с backend и frontend

# остановить Postgres
docker compose down

# полностью удалить данные БД
docker compose down -v

# очистить кеш фронта
rm -rf frontend/.next frontend/node_modules
```

---

## 12. Деплой — краткие заметки

Этот MVP оптимизирован под локальную защиту диссертации, но контуры деплоя:

- **Backend** — собрать `go build -o server ./cmd/server`, запустить как systemd-сервис или в Docker. `JWT_SECRET` обязательно поменять.
- **Frontend** — `npm run build && npm start` (SSR) либо статический экспорт + nginx.
- **Postgres** — managed (Supabase, Neon, Yandex Cloud) или самохостинг, бэкапы обязательны.
- **HTTPS** — через nginx/Caddy перед бэком и фронтом, единый домен с путями `/api/*` и `/*` или два субдомена.
- **CORS** — обновить `CORS_ORIGIN` на продовый домен фронта.

---

## 13. Полезные ссылки

- Основной README: `README.md`
- Миграции БД: `backend/internal/db/migrations/`
- Тесты baseline-алгоритма: `backend/internal/metrics/baseline_test.go`
- Палитра и стили: `frontend/tailwind.config.ts`, `frontend/app/globals.css`

Вопросы/баги — смотрите логи backend (stdout) и frontend (в терминале `npm run dev`).
