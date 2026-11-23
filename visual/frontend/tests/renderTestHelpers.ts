// Общие помощники рендер-тестов: сборка кадра и прогон через публичный
// renderFrame на свежих DOM-узлах. Файл не *.test.ts — витестом не
// собирается, импортируется тестами шва рендера (tree, trie и далее).
import { renderFrame } from "../src/render";
import type { Frame } from "../src/types";

/** Кадр для renderFrame: kind и снимок; шаг и пояснение — по умолчанию. */
export function makeFrame(
  kind: Frame["kind"],
  data: unknown,
  explanation = "пояснение шага",
  step = 1,
): Frame {
  return { step, kind, explanation, data } as Frame;
}

/** Прогнать кадр через публичный renderFrame на свежих DOM-узлах. */
export function renderToDOM(frame: Frame): {
  ok: boolean;
  explanationEl: HTMLElement;
  containerEl: HTMLElement;
} {
  const explanationEl = document.createElement("div");
  const containerEl = document.createElement("div");
  const ok = renderFrame(frame, explanationEl, containerEl);
  return { ok, explanationEl, containerEl };
}
