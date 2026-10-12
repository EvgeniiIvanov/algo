# Единая точка входа для dev-команд (тикет 14). Все цели запускаются из корня репозитория.
# Проверки повторяют то, что гоняет CI (.github/workflows/ci.yml); make check
# дополнительно включает go build ./... — отдельного шага сборки Go в CI нет.

FRONTEND_DIR := visual/frontend

.PHONY: help frontend-install gofmt-check go-vet go-lint go-test go-build \
        frontend-lint frontend-test frontend-build \
        lint test build check run-api run-frontend run-quiz

## Показать список целей и кратко описать их
help:
	@echo "Доступные цели (запуск из корня репозитория):"
	@echo "  make help             — этот список"
	@echo "  make frontend-install — установить зависимости фронтенда (pnpm install --frozen-lockfile)"
	@echo "  make run-api          — запустить Go API (go run ./cmd/web, :8080) в отдельном терминале"
	@echo "  make run-frontend     — запустить dev-сервер Vite (:5173) в отдельном терминале"
	@echo "  make run-quiz         — запустить CLI-квиз"
	@echo "  make lint             — gofmt-check + go vet + golangci-lint + frontend eslint"
	@echo "  make test             — go test -race ./... + frontend vitest"
	@echo "  make build            — go build ./... + frontend build (tsc + vite)"
	@echo "  make check            — frozen install, затем lint → test → build, fail-fast (как в CI, плюс go build)"

## Установить зависимости фронтенда (не меняет pnpm-lock.yaml)
frontend-install:
	cd $(FRONTEND_DIR) && pnpm install --frozen-lockfile

# --- Go-проверки (паритет с CI) ---

## gofmt: падаем, если есть неотформатированные файлы
gofmt-check:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
	  echo "Не отформатированы (gofmt):"; \
	  echo "$$unformatted"; \
	  exit 1; \
	fi

## go vet ./...
go-vet:
	go vet ./...

## golangci-lint run ./...
go-lint:
	golangci-lint run ./...

## go test -race ./...
go-test:
	go test -race ./...

## go build ./...
go-build:
	go build ./...

# --- Frontend-проверки (выполняются в visual/frontend) ---

## frontend eslint
frontend-lint:
	cd $(FRONTEND_DIR) && pnpm lint

## frontend vitest (run)
frontend-test:
	cd $(FRONTEND_DIR) && pnpm test

## frontend build (tsc --noEmit + vite build → dist/)
frontend-build:
	cd $(FRONTEND_DIR) && pnpm build

# --- Запуск компонентов ---

## Запустить Go API-сервер (по умолчанию :8080)
run-api:
	go run ./cmd/web

## Запустить dev-сервер фронтенда (Vite, http://localhost:5173)
run-frontend:
	cd $(FRONTEND_DIR) && pnpm dev

## Запустить CLI-квиз
run-quiz:
	go run ./cmd/quiz

# --- Составные цели ---

## Линтеры Go и фронтенда (паритет с job lint + шаг eslint в CI)
lint: gofmt-check go-vet go-lint frontend-lint

## Тесты Go и фронтенда (паритет с job test)
test: go-test frontend-test

## Сборка Go и фронтенда
build: go-build frontend-build

## Полная проверка: frozen install, затем lint → test → build (как в CI, плюс go build).
## check зависит только от frontend-install, поэтому frontend-install всегда
## завершён до первой проверки, даже при make -j check. Последовательные $(MAKE)
## в рецепте дают fail-fast по факту: неуспешная строка останавливает рецепт.
check: frontend-install
	$(MAKE) lint
	$(MAKE) test
	$(MAKE) build
