# Визуализатор — фронтенд

Тупой рендерер кадров (ADR-0002): Vite + ванильный TypeScript, без фреймворка.
Поддерживаемые kind'ы кадров: `array` (бинарный поиск) и `structure` (стек —
ряд ячеек на всю ёмкость, подсветки added/removed, указатель top).
Подробности — в [docs/tickets/03-ts-player.md](../../docs/tickets/03-ts-player.md)
и [docs/tickets/08-stack-vertical-slice.md](../../docs/tickets/08-stack-vertical-slice.md).

## Запуск в dev

Два слоя работают независимо: фронт не знает про бэкенд, кроме SSE-эндпоинта `/api/run/*`.

```sh
# 1. Бэкенд (из корня репозитория)
go run ./cmd/web -addr :8080

# 2. Фронт (в соседнем терминале, из этой папки)
pnpm install   # Node 22+; включает esbuild через pnpm-workspace.yaml
pnpm dev       # http://localhost:5173 — проксирует /api → :8080
```

Другой адрес бэкенда — переменная `ALGO_BACKEND`, по умолчанию `http://localhost:8080`.

## Сборка

```sh
pnpm build    # → dist/
```

`cmd/web` пока не раздаёт `dist/` — это отдельный тикет.

## Тесты

```sh
pnpm test
```

Шов фронта — `tests/` (см. [docs/SPEC.md](../../docs/SPEC.md), «Четвёртый шов»). Покрыто:
- парсер SSE-кадров (`sse.test.ts`);
- плеер (`player.test.ts`): play/pause, seek, stepForward/Backward, setSpeed, subscribe, load, **append** (без сброса позиции), регрессия на авто-паузу в конце;
- классификация кадра как массива (`render.test.ts`);
- классификация и DOM-рендер кадра `structure` (`structure.test.ts`): ряд ячеек на всю ёмкость, подсветки added/removed, указатель top;
- парсинг сценария операций стека и ёмкости (`scenario.test.ts`): ошибки ввода до Run;
- сетевой клиент + плеер (`stream.test.ts`): успешный поток, HTTP 400 с понятным сообщением, сетевая ошибка, **AbortError при stop() не показывается как ошибка**, кадры из стрима не сбрасывают позицию пользователя.

## Структура

```
src/
  types.ts     # Frame, ArraySnapshot, StructureSnapshot — зеркало visual/frame/frame.go
  sse.ts       # parseFrame, buildRunURL, buildStackRunURL, parseError
  scenario.ts  # парсинг сценария операций стека (зеркало серверной валидации)
  player.ts    # плеер: состояние кадров + таймер + подписки
  stream.ts    # SSE-клиент поверх плеера
  render.ts    # тупой рендерер кадра в DOM (ветки array и structure)
  main.ts      # связка: форма → плеер → render
  style.css    # базовые стили
tests/
  sse.test.ts
  player.test.ts
  render.test.ts
  structure.test.ts
  scenario.test.ts
  stream.test.ts
```