# Визуализатор — фронтенд

Тупой рендерер кадров бинарного поиска (ADR-0002): Vite + ванильный TypeScript,
без фреймворка. Подробности — в [docs/tickets/03-ts-player.md](../../docs/tickets/03-ts-player.md).

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

Шов фронта — `tests/`. Покрыто:
- парсер SSE-кадров (`sse.test.ts`);
- плеер (`player.test.ts`): play/pause, seek, stepForward/Backward, setSpeed, subscribe, load;
- классификация кадра как массива (`render.test.ts`);
- сетевой клиент + плеер (`stream.test.ts`): успешный поток, HTTP 400 с понятным сообщением, сетевая ошибка.

## Структура

```
src/
  types.ts     # Frame, ArraySnapshot — зеркало visual/frame/frame.go
  sse.ts       # parseFrame, buildRunURL, parseError
  player.ts    # плеер: состояние кадров + таймер + подписки
  stream.ts    # SSE-клиент поверх плеера
  render.ts    # тупой рендерер кадра в DOM
  main.ts      # связка: форма → плеер → render
  style.css    # базовые стили
tests/
  sse.test.ts
  player.test.ts
  render.test.ts
  stream.test.ts
```