// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// парсер SSE-кадров — единственная точка контакта с бэкендом по сети.
// Тест читает сырой поток и сверяет разобранные кадры с ожидаемыми.
// Это шов, не опускаемся ниже — внутрь fetch не заглядываем.
import { describe, expect, it } from "vitest";
import { parseFrame } from "../src/sse";

describe("parseFrame", () => {
  it("разбирает одно событие data в кадр", () => {
    const wire =
      'data: {"step":1,"kind":"array","explanation":"старт","data":{"values":[1,3,5],"low":0,"high":2,"mid":-1,"foundIndex":-1}}\n\n';
    const frame = parseFrame(wire);

    expect(frame).toEqual({
      step: 1,
      kind: "array",
      explanation: "старт",
      data: { values: [1, 3, 5], low: 0, high: 2, mid: -1, foundIndex: -1 },
    });
  });

  it("принимает строку без хвостового \\n\\n", () => {
    const wire =
      'data: {"step":2,"kind":"array","explanation":"шаг","data":{"values":[1],"low":0,"high":0,"mid":0,"foundIndex":-1}}';
    const frame = parseFrame(wire);

    expect(frame.step).toBe(2);
    expect(frame.kind).toBe("array");
  });

  it("бросает ошибку на событие без data (например, server-sent ping)", () => {
    const wire = ": ping\n\n";
    expect(() => parseFrame(wire)).toThrow(/data/i);
  });

  it("бросает ошибку на невалидный JSON", () => {
    const wire = "data: {не json}\n\n";
    expect(() => parseFrame(wire)).toThrow();
  });

  it("бросает ошибку на JSON без обязательного поля step", () => {
    const wire = 'data: {"kind":"array","explanation":"без шага","data":{}}\n\n';
    expect(() => parseFrame(wire)).toThrow(/step/i);
  });
});