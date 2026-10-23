// Парсер SSE-кадров от бэкенда (visual/server/server.go, ADR-0003).
// Один вход — сырая порция из fetch-stream с одним или несколькими
// `data: {...}\n\n` событиями. Один выход — структура Frame.
//
// Шов не опускается ниже: нас не интересует, как fetch читает поток,
// нас интересует только парсинг одной порции данных.
// См. tests/sse.test.ts.

import type { Frame } from "./types";

/** Один кадр из SSE-порции. Бросает Error, если данных нет или JSON кривой. */
export function parseFrame(chunk: string): Frame {
  // Нас интересует только тело `data:`. Прочие строки (event:, id:, retry:,
  // комментарии `:`) игнорируем — бэкенд их не шлёт.
  const lines = chunk.split(/\r?\n/);
  const dataLines: string[] = [];
  for (const line of lines) {
    if (line.startsWith("data:")) {
      // После "data:" — один пробел (RFC), остальное — тело события.
      dataLines.push(line.slice("data:".length).replace(/^ /, ""));
    }
  }
  if (dataLines.length === 0) {
    throw new Error("SSE-порция не содержит события data");
  }
  const payload = dataLines.join("\n");
  let raw: unknown;
  try {
    raw = JSON.parse(payload);
  } catch (err) {
    throw new Error(`кадр не разбирается как JSON: ${(err as Error).message}`, { cause: err });
  }
  return validateFrame(raw);
}

function validateFrame(value: unknown): Frame {
  if (value === null || typeof value !== "object") {
    throw new Error("кадр не объект");
  }
  const obj = value as Record<string, unknown>;
  if (typeof obj.step !== "number" || !Number.isInteger(obj.step) || obj.step < 1) {
    throw new Error("обязательное поле step должно быть положительным целым");
  }
  if (typeof obj.kind !== "string") {
    throw new Error("обязательное поле kind должно быть строкой");
  }
  if (typeof obj.explanation !== "string") {
    throw new Error("обязательное поле explanation должно быть строкой");
  }
  if (obj.data === null || typeof obj.data !== "object") {
    throw new Error("обязательное поле data должно быть объектом");
  }
  return obj as unknown as Frame;
}

/**
 * Построить URL запроса Run для бэкенда.
 * Вынесено из main.ts, чтобы было что тестировать и переиспользовать.
 */
export function buildRunURL(algorithm: string, values: number[], target: number): string {
  const params = new URLSearchParams({
    values: values.join(","),
    target: String(target),
  });
  return `/api/run/${algorithm}?${params.toString()}`;
}

/**
 * Построить URL запуска Run стека: ёмкость + сырой текст сценария.
 * Сценарий едет как есть — парсинг и валидация авторитетны на сервере.
 */
export function buildStackRunURL(capacity: number, script: string): string {
  const params = new URLSearchParams({
    capacity: String(capacity),
    script,
  });
  return `/api/run/stack?${params.toString()}`;
}

/**
 * Построить URL запуска Run очереди: ёмкость + сырой текст сценария.
 * Сценарий едет как есть — парсинг и валидация авторитетны на сервере.
 */
export function buildQueueRunURL(capacity: number, script: string): string {
  const params = new URLSearchParams({
    capacity: String(capacity),
    script,
  });
  return `/api/run/queue?${params.toString()}`;
}

/**
 * Построить URL запуска Run списка: сырой текст сценария. Список растущий
 * без заданной ёмкости, поэтому параметра capacity нет.
 * Сценарий едет как есть — парсинг и валидация авторитетны на сервере.
 */
export function buildListRunURL(script: string): string {
  const params = new URLSearchParams({ script });
  return `/api/run/list?${params.toString()}`;
}

/**
 * Построить URL запуска Run односвязного списка: сырой текст сценария.
 * Ёмкости нет — список на указателях растёт без заданной ёмкости.
 * Сценарий едет как есть — парсинг и валидация авторитетны на сервере.
 */
export function buildLinkedListRunURL(script: string): string {
  const params = new URLSearchParams({ script });
  return `/api/run/linked-list?${params.toString()}`;
}

/**
 * Построить URL запуска Run хеш-таблицы: число корзин + сырой текст
 * сценария. Сценарий едет как есть — парсинг и валидация авторитетны
 * на сервере.
 */
export function buildHashTableRunURL(buckets: number, script: string): string {
  const params = new URLSearchParams({
    buckets: String(buckets),
    script,
  });
  return `/api/run/hash-table?${params.toString()}`;
}

/** Канонический список маршрутов простых сортировок: единый источник имён
 * для URL-билдера ниже и для выбора алгоритма в main.ts (ALGORITHMS,
 * блок ввода, диспетчер Run). Общий вход `values`, различается только путь. */
export const SORT_ALGORITHMS = [
  "bubble-sort",
  "selection-sort",
  "insertion-sort",
  "merge-sort",
  "quick-sort",
] as const;

/** Маршрут простой сортировки — элемент канонического списка. */
export type SortAlgorithm = (typeof SORT_ALGORITHMS)[number];

/** Страж типа: выбран ли маршрут простой сортировки. */
export function isSortAlgorithm(value: string): value is SortAlgorithm {
  return (SORT_ALGORITHMS as readonly string[]).includes(value);
}

/**
 * Построить URL запуска Run простой сортировки: массив целых чисел через
 * запятую на маршруте выбранной сортировки. Пустой массив едет как есть —
 * сервер авторитетен и считает его валидным.
 */
export function buildSortRunURL(algorithm: SortAlgorithm, values: number[]): string {
  const params = new URLSearchParams({ values: values.join(",") });
  return `/api/run/${algorithm}?${params.toString()}`;
}

/**
 * Разобрать тело ошибки от бэкенда (application/json, поле "error").
 * Если тело не JSON или формат не тот — возвращает null, и вызывающий
 * решает, как показать ошибку.
 */
export function parseError(body: string): string | null {
  try {
    const obj = JSON.parse(body) as Record<string, unknown>;
    if (typeof obj.error === "string") return obj.error;
    return null;
  } catch {
    return null;
  }
}