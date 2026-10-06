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
    throw new Error(`кадр не разбирается как JSON: ${(err as Error).message}`);
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