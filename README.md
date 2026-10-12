# algo — обучающий проект: визуализатор алгоритмов на Go

Учебный проект с двумя целями: прокачать алгоритмы по структурированной программе
(~35–40 ч) и поработать с ИИ-агентами в парадигме SDD (интервью → спека → тикеты →
имплементация → ревью). Учебная программа — в [PLAN.md](PLAN.md), фактическое
завершение модулей отмечается в [PROGRESS.md](PROGRESS.md).

## Слои проекта

Три независимых слоя (граница слоёв — ADR-0001):

| Слой | Путь | Что это |
|------|------|---------|
| Алгоритмы | `algorithms/` | Чистая Go-библиотека, без UI; quiet mode + опциональный Emitter |
| Визуализатор | `visual/` | Go-сервер (SSE, кадры-снапшоты) + TS-фронтенд (Vite, без фреймворка) |
| Квиз | `quiz/`, `cmd/quiz` | Интерактивный CLI-квиз по учебной программе |

Точки входа — `cmd/web` (сервер API) и `cmd/quiz`.

## Prerequisites

- **Go 1.27** (см. `go.mod`)
- **Node 22** и **pnpm** — версионирование pnpm зафиксировано полем
  `packageManager` в `visual/frontend/package.json` (pnpm 12.9.1, через corepack);
  Node 22 — то, что использует CI
- `golangci-lint` для локального паритета с CI

## Быстрый старт

```sh
make frontend-install   # pnpm install --frozen-lockfile в visual/frontend (lockfile не меняется)
```

### Визуализатор: два терминала

API и dev-фронтенд работают независимо и запускаются **в отдельных терминалах**:

```sh
make run-api            # терминал 1: Go API на :8080 (адрес: go run ./cmd/web -addr :8080)
make run-frontend       # терминал 2: Vite dev-сервер на http://localhost:5173
```

Vite проксирует `/api` на Go-бэкенд. Другой адрес бэкенда — переменная
`ALGO_BACKEND` (по умолчанию `http://localhost:8080`), читается в
`visual/frontend/vite.config.ts` при старте dev-сервера.

Go-сервер предоставляет только API; раздача собранного `dist/` из Go не входит
в локальный dev-сценарий и остаётся вне scope v1 (сборка кладёт статику в
`visual/frontend/dist/`).

### CLI-квиз

```sh
make run-quiz           # интерактивный CLI: go run ./cmd/quiz
```

## Проверки из корня

Команды повторяют то, что гоняет CI (`.github/workflows/ci.yml`):

| Цель | Что делает |
|------|------------|
| `make lint` | gofmt-check + `go vet ./...` + `golangci-lint run ./...` + frontend eslint |
| `make test` | `go test -race ./...` + frontend vitest |
| `make build` | `go build ./...` + frontend build (`tsc --noEmit` + `vite build`) |
| `make check` | Полный CI-паритет: frozen install + все линтеры, тесты и сборка, fail-fast |

Frontend-шаги внутри `make` выполняются в `visual/frontend`; запускать что-то
вручную из этой папки не обязательно. Список всех целей — `make help`.

## Документация

- [PLAN.md](PLAN.md) — цели, принятые решения, учебная программа
- [PROGRESS.md](PROGRESS.md) — чек-лист завершения модулей
- [GLOSSARY.md](GLOSSARY.md) — канонический словарь терминов проекта
- [docs/SPEC.md](docs/SPEC.md) — требования активного учебного модуля
- [docs/WORKFLOWS.md](docs/WORKFLOWS.md) — сценарии процессов (спека → тикеты → реализация → ревью)
- [docs/adr/](docs/adr/) — архитектурные решения
