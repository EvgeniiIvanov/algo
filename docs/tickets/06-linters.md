# 06: Линтеры в общем CI — errcheck в Go-коде + ESLint для TS-фронтенда

**What to build:** навести порядок с линтерами по итогам ревью тикета 03. (1) В Go-сервере: golangci-lint нашёл 2 errcheck-замечания в `visual/server/server.go` — не проверены `fmt.Fprintf` в SSE-стриме и `json.Encode` в error-handler'е. Код уже слит с тикетом 02, а CI на PR гоняет golangci-lint и покраснеет. (2) У TS-фронтенда нет линтера вообще: `tsc --noEmit` проверяет только типы. Добавить ESLint (typescript-eslint strict, flat config) с запуском локально и в CI.

**Blocked by:** None (can start immediately).

**Status:** done

## Acceptance Criteria

- [x] `golangci-lint run ./...` локально — 0 замечаний по Go-коду
- [x] ESLint настроен в `visual/frontend` (typescript-eslint strict, flat config), `pnpm lint` — 0 ошибок
- [x] В CI добавлен frontend-job: pnpm install → eslint → vitest → build (tsc + vite)
- [x] Все проверки зелёные локально

## Notes

- Порядок: сначала errcheck в Go (мелочь: ошибка записи = разорванное соединение), затем ESLint.
- Не упарываться: правила по умолчанию, без кастомных конфигов — добавим при первой боли.
