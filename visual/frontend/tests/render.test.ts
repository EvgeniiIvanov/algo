// Тест рендерера (см. docs/SPEC.md, шов фронтенда):
// проверяем классификацию и парсинг снимка массива —
// именно здесь легко сломать, если формат на бэкенде разъедется с фронтом.
import { describe, expect, it } from "vitest";
import { isArrayFrame } from "../src/render";
import type { Frame } from "../src/types";

describe("isArrayFrame", () => {
  it("принимает корректный снимок массива", () => {
    const f: Frame = {
      step: 1,
      kind: "array",
      explanation: "x",
      data: { values: [1, 2, 3], low: 0, high: 2, mid: -1, foundIndex: -1 },
    };
    expect(isArrayFrame(f)).toBe(true);
  });

  it("отвергает неизвестный kind", () => {
    const f = {
      step: 1,
      kind: "graph",
      explanation: "x",
      data: { values: [1], low: 0, high: 0, mid: -1, foundIndex: -1 },
    } as unknown as Frame;
    expect(isArrayFrame(f)).toBe(false);
  });

  it("отвергает kind=array, но data — не массив", () => {
    const f = {
      step: 1,
      kind: "array",
      explanation: "x",
      data: { values: "не массив", low: 0, high: 0, mid: -1, foundIndex: -1 },
    } as unknown as Frame;
    expect(isArrayFrame(f)).toBe(false);
  });

  it("отвергает снимок с нечисловыми значениями", () => {
    const f = {
      step: 1,
      kind: "array",
      explanation: "x",
      data: { values: [1, "x" as unknown as number], low: 0, high: 1, mid: -1, foundIndex: -1 },
    };
    expect(isArrayFrame(f)).toBe(false);
  });

  it("отвергает снимок без обязательного числового поля", () => {
    const f = {
      step: 1,
      kind: "array",
      explanation: "x",
      data: { values: [1, 2, 3], low: 0, high: 2, mid: -1 }, // нет foundIndex
    } as unknown as Frame;
    expect(isArrayFrame(f)).toBe(false);
  });
});