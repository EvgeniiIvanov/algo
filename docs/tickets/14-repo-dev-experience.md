# 14: Упростить запуск и проверки проекта

**What to build:** разработчик может быстро понять, как устроен проект, запустить API, фронтенд и CLI-квиз, а также выполнить основные проверки из корня репозитория через короткие команды `make`.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

## Acceptance Criteria

- [ ] Корневой `README.md` кратко описывает назначение проекта и его три слоя, требования к Go/Node/pnpm, быстрый запуск API и фронтенда в отдельных терминалах, запуск квиза и ссылки на `PLAN.md`, `PROGRESS.md`, `GLOSSARY.md` и `docs/WORKFLOWS.md`
- [ ] `Makefile` содержит `help` с перечнем и кратким описанием доступных целей
- [ ] Есть отдельные цели запуска API, dev-фронтенда и CLI-квиза
- [ ] Цели сборки, тестов и линтинга покрывают Go и фронтенд; `check` повторяет проверки CI, включая `gofmt`, `go vet`, `golangci-lint`, `go test -race ./...`, `pnpm install --frozen-lockfile`, frontend lint/test/build
- [ ] Все цели запускаются из корня репозитория, frontend-команды выполняются в `visual/frontend`, установка зависимостей не меняет lockfile
- [ ] Инструкции README согласованы с `Makefile`, текущим CI и локальным способом запуска через Vite + Go API
