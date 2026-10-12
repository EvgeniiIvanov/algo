# Визуализатор — фронтенд

Тупой рендерер кадров (ADR-0002): Vite + ванильный TypeScript, без фреймворка.
Поддерживаемые kind'ы кадров: `array` (бинарный поиск) и `structure` (фиксированные
Stack и Queue — ряд ячеек на всю ёмкость; для стека указатель top, для очереди head/tail,
подсветки added/removed/shifted).
Подробности — в [docs/tickets/03-ts-player.md](../../docs/tickets/03-ts-player.md),
[docs/tickets/08-stack-vertical-slice.md](../../docs/tickets/08-stack-vertical-slice.md)
и [docs/tickets/09-queue-vertical-slice.md](../../docs/tickets/09-queue-vertical-slice.md).
Валидация ввода бинарного поиска описана в [docs/tickets/13-binary-search-integer-input.md](../../docs/tickets/13-binary-search-integer-input.md).

## Запуск в dev

Два слоя работают независимо: фронт не знает про бэкенд, кроме SSE-эндпоинта `/api/run/*`.

```sh
# 1. API-бэкенд (из корня репозитория)
go run ./cmd/web -addr :8080

# 2. Фронт (в соседнем терминале, из этой папки)
pnpm install --frozen-lockfile   # Node 22+; включает esbuild через pnpm-workspace.yaml
pnpm dev       # http://localhost:5173 — проксирует /api → :8080
```

Другой адрес бэкенда — переменная `ALGO_BACKEND`, по умолчанию `http://localhost:8080`.

## Сборка

```sh
pnpm build    # → dist/
```

Go-сервер предоставляет API; раздача собранного `dist/` из Go не входит в локальный dev-сценарий и остаётся вне scope v1.

## Тесты

```sh
pnpm test
```

Шов фронта — `tests/` (см. [docs/SPEC.md](../../docs/SPEC.md), «Четвёртый шов»). Покрыто:
- числовой парсер бинарного поиска и общая грамматика десятичных целых (`binarysearch.test.ts`, `decimalInteger.test.ts`);
- парсер SSE-кадров (`sse.test.ts`);
- плеер (`player.test.ts`): play/pause, seek, stepForward/Backward, setSpeed, subscribe, load, **append** (без сброса позиции), регрессия на авто-паузу в конце;
- классификация кадра как массива (`render.test.ts`);
- классификация и DOM-рендер кадра `structure` (`structure.test.ts`, `queueRender.test.ts`): ячейки, подсветки, указатели top/head/tail;
- парсинг сценариев стека и очереди, ёмкости (`scenario.test.ts`, `queueScenario.test.ts`): ошибки ввода до Run;
- сетевой клиент + плеер (`stream.test.ts`): успешный поток, HTTP 400 с понятным сообщением, сетевая ошибка, **AbortError при stop() не показывается как ошибка**, кадры из стрима не сбрасывают позицию пользователя.

## Структура

```
src/
  types.ts          # Frame, ArraySnapshot, StructureSnapshot — зеркало Go-кадров
  decimalInteger.ts # общая грамматика десятичных целых и safe-integer проверка
  binarysearch.ts   # разбор входа бинарного поиска
  sse.ts            # parseFrame, buildRunURL/buildStackRunURL/buildQueueRunURL, parseError
  scenario.ts       # парсинг сценариев стека и очереди (зеркало серверной валидации)
  player.ts         # плеер: состояние кадров + таймер + подписки
  stream.ts         # SSE-клиент поверх плеера
  render.ts         # тупой рендерер кадра в DOM (ветки array и structure)
  main.ts           # связка формы, плеера, SSE и рендера
  style.css         # стили
tests/
  binarysearch.test.ts
  decimalInteger.test.ts
  queueRender.test.ts
  queueScenario.test.ts
  player.test.ts
  render.test.ts
  structure.test.ts
  scenario.test.ts
  sse.test.ts
  stream.test.ts
```
