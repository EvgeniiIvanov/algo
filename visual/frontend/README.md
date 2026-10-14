# Визуализатор — фронтенд

Тупой рендерер кадров (ADR-0002): Vite + ванильный TypeScript, без фреймворка.
Поддерживаемые kind'ы кадров: `array` (бинарный поиск) и `structure`:
фиксированные Stack и Queue — ряд ячеек на всю ёмкость (для стека указатель top,
для очереди head/tail); растущий List на слайсе — только заполненные ячейки,
надгробие при удалении; односвязный LinkedList — узлы со стрелками next и
разрывами между ними (память не непрерывна), метка head, роли подсветки
added/removed/shifted/visited/relinked/found.
Подробности — в [docs/tickets/03-ts-player.md](../../docs/tickets/03-ts-player.md),
[docs/tickets/08-stack-vertical-slice.md](../../docs/tickets/08-stack-vertical-slice.md),
[docs/tickets/09-queue-vertical-slice.md](../../docs/tickets/09-queue-vertical-slice.md) и
[docs/tickets/11-linked-list-vertical-slice.md](../../docs/tickets/11-linked-list-vertical-slice.md).
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
- классификация и DOM-рендер кадра `structure` вида stack (`structure.test.ts`): ячейки, подсветки, указатели top/head/tail;
- рендер вида queue (`queueRender.test.ts`) и вида list (`listRender.test.ts`): последовательность ячеек, надгробие при удалении;
- рендер вида linked-list (`linkedListRender.test.ts`): узлы со стрелками next, разрывы, метка head, роли visited/relinked/found, надгробие;
- парсинг сценариев stack/queue/list/linked-list, ёмкости (`scenario.test.ts`, `queueScenario.test.ts`, `listScenario.test.ts`, `linkedListScenario.test.ts`): ошибки ввода до Run;
- сетевой клиент + плеер (`stream.test.ts`): успешный поток, HTTP 400 с понятным сообщением, сетевая ошибка, **AbortError при stop() не показывается как ошибка**, кадры из стрима не сбрасывают позицию пользователя.

## Структура

```
src/
  types.ts          # Frame, ArraySnapshot, StructureSnapshot — зеркало Go-кадров
  decimalInteger.ts # общая грамматика десятичных целых и safe-integer проверка
  binarysearch.ts   # разбор входа бинарного поиска
  sse.ts            # parseFrame, buildRunURL/buildStackRunURL/buildQueueRunURL/buildListRunURL/buildLinkedListRunURL, parseError
  scenario.ts       # парсинг сценариев stack/queue/list/linked-list (зеркало серверной валидации)
  player.ts         # плеер: состояние кадров + таймер + подписки
  stream.ts         # SSE-клиент поверх плеера
  render.ts         # тупой рендерер кадра в DOM (ветки array и structure; linked-list — узлы со стрелками)
  main.ts           # связка формы, плеера, SSE и рендера
  style.css         # стили
tests/
  binarysearch.test.ts
  decimalInteger.test.ts
  linkedListRender.test.ts
  linkedListScenario.test.ts
  listRender.test.ts
  listScenario.test.ts
  queueRender.test.ts
  queueScenario.test.ts
  player.test.ts
  render.test.ts
  structure.test.ts
  scenario.test.ts
  sse.test.ts
  stream.test.ts
```
