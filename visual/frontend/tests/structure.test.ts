// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// рендер кадра kind="structure" (стек) — классификация снимка
// и DOM-отрисовка ряда ячеек с подсветками и указателем top.
import { describe, expect, it } from "vitest";
import { isStructureFrame, renderFrame } from "../src/render";
import type { Frame, StructureSnapshot } from "../src/types";

function structureFrame(data: unknown): Frame {
  return { step: 1, kind: "structure", explanation: "пояснение шага", data } as Frame;
}

const baseSnapshot: StructureSnapshot = {
  structure: "stack",
  capacity: 3,
  values: [5, 7],
  highlights: [{ index: 1, role: "added" }],
  top: 1,
  operation: "push 7",
};

/** Отрендерить кадр в свежие DOM-узлы и вернуть их. */
function renderToDOM(frame: Frame): { ok: boolean; explanationEl: HTMLElement; containerEl: HTMLElement } {
  const explanationEl = document.createElement("div");
  const containerEl = document.createElement("div");
  const ok = renderFrame(frame, explanationEl, containerEl);
  return { ok, explanationEl, containerEl };
}

describe("isStructureFrame", () => {
  it("принимает корректный снимок структуры", () => {
    expect(isStructureFrame(structureFrame(baseSnapshot))).toBe(true);
  });

  it("принимает пустой стек: без значений, подсветок и top", () => {
    const empty: StructureSnapshot = {
      structure: "stack",
      capacity: 3,
      values: [],
      highlights: [],
      top: -1,
      operation: "",
    };
    expect(isStructureFrame(structureFrame(empty))).toBe(true);
  });

  it("отвергает другой kind", () => {
    const f = { ...structureFrame(baseSnapshot), kind: "array" } as Frame;
    expect(isStructureFrame(f)).toBe(false);
  });

  it("отвергает снимок с values не массивом", () => {
    expect(isStructureFrame(structureFrame({ ...baseSnapshot, values: "мусор" }))).toBe(false);
  });

  it("отвергает снимок без capacity", () => {
    const noCapacity = {
      structure: "stack",
      values: [5],
      highlights: [],
      top: 0,
      operation: "push 5",
    };
    expect(isStructureFrame(structureFrame(noCapacity))).toBe(false);
  });

  it("отвергает нулевую заданную ёмкость стека", () => {
    expect(isStructureFrame(structureFrame({ ...baseSnapshot, capacity: 0 }))).toBe(false);
  });

  it("отвергает подсветку без роли", () => {
    expect(isStructureFrame(structureFrame({ ...baseSnapshot, highlights: [{ index: 1 }] }))).toBe(
      false,
    );
  });

  it("отвергает снимок без top или operation", () => {
    const noTop = { structure: "stack", capacity: 3, values: [5], highlights: [], operation: "x" };
    expect(isStructureFrame(structureFrame(noTop))).toBe(false);
    const noOp = { structure: "stack", capacity: 3, values: [5], highlights: [], top: 0 };
    expect(isStructureFrame(structureFrame(noOp))).toBe(false);
  });
});

describe("renderFrame kind=structure", () => {
  it("рисует ряд ячеек на всю ёмкость: заполненные и пустые видны", () => {
    const { ok, explanationEl, containerEl } = renderToDOM(structureFrame(baseSnapshot));

    expect(ok).toBe(true);
    expect(explanationEl.textContent).toBe("пояснение шага");

    const cells = containerEl.querySelectorAll(".cell");
    expect(cells).toHaveLength(3); // вся ёмкость, не только значения
    expect(cells[0].textContent).toBe("5");
    expect(cells[1].textContent).toBe("7");
    expect(cells[2].textContent).toBe(""); // пустая ячейка видна
    expect(cells[0].classList.contains("filled")).toBe(true);
    expect(cells[2].classList.contains("empty")).toBe(true);
  });

  it("подсвечивает затронутую ячейку ролью, помечает top и показывает операцию", () => {
    const { containerEl } = renderToDOM(structureFrame(baseSnapshot));

    const cells = containerEl.querySelectorAll(".cell");
    expect(cells[1].classList.contains("added")).toBe(true);
    expect(cells[1].classList.contains("top")).toBe(true);
    expect(cells[0].classList.contains("added")).toBe(false);
    expect(containerEl.querySelector(".operation")?.textContent).toContain("push 7");
  });

  it("кадр pop: освобождённая ячейка пуста и подсвечена removed, top сместился", () => {
    const popped: StructureSnapshot = {
      ...baseSnapshot,
      values: [5],
      highlights: [{ index: 1, role: "removed" }],
      top: 0,
      operation: "pop",
    };
    const { containerEl } = renderToDOM(structureFrame(popped));

    const cells = containerEl.querySelectorAll(".cell");
    expect(cells[1].classList.contains("removed")).toBe(true);
    expect(cells[1].classList.contains("empty")).toBe(true);
    expect(cells[0].classList.contains("top")).toBe(true);
  });

  it("кадр переполнения: содержимое без изменений, подсветок нет", () => {
    const overflow: StructureSnapshot = {
      ...baseSnapshot,
      values: [5, 7, 9],
      highlights: [],
      top: 2,
      operation: "push 13",
    };
    const { containerEl } = renderToDOM(structureFrame(overflow));

    const cells = containerEl.querySelectorAll(".cell");
    expect(containerEl.querySelectorAll(".added, .removed")).toHaveLength(0);
    expect(cells[2].textContent).toBe("9");
    expect(cells[2].classList.contains("top")).toBe(true);
  });

  it("стартовый кадр пустого стека: все ячейки пустые, top нигде нет", () => {
    const start: StructureSnapshot = {
      structure: "stack",
      capacity: 2,
      values: [],
      highlights: [],
      top: -1,
      operation: "",
    };
    const { containerEl } = renderToDOM(structureFrame(start));

    const cells = containerEl.querySelectorAll(".cell");
    expect(cells).toHaveLength(2);
    expect(containerEl.querySelectorAll(".empty")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".top")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Исходное состояние");
  });

  it("неизвестный kind не рисуется и не трогает DOM", () => {
    const f = { step: 1, kind: "graph", explanation: "x", data: {} } as unknown as Frame;
    const { ok, containerEl } = renderToDOM(f);

    expect(ok).toBe(false);
    expect(containerEl.childElementCount).toBe(0);
  });
});
