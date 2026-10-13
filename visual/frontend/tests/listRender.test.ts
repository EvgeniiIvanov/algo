// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// рендер кадра kind="structure" для списка — та же ветка рендера, что у
// стека и очереди: подсветки ролей и подпись операции. Ряд ячеек у списка
// рисуется только из заполненных ячеек: capacity = len(values), в том числе
// ноль на пустом списке — пустой кадр без пустых ячеек и указателей.
import { describe, expect, it } from "vitest";
import { isStructureFrame, renderFrame } from "../src/render";
import type { Frame, StructureSnapshot } from "../src/types";

function structureFrame(data: unknown): Frame {
  return { step: 1, kind: "structure", explanation: "пояснение шага", data } as Frame;
}

/** Отрендерить кадр в свежие DOM-узлы и вернуть их. */
function renderToDOM(frame: Frame): { ok: boolean; explanationEl: HTMLElement; containerEl: HTMLElement } {
  const explanationEl = document.createElement("div");
  const containerEl = document.createElement("div");
  const ok = renderFrame(frame, explanationEl, containerEl);
  return { ok, explanationEl, containerEl };
}

const startSnapshot: StructureSnapshot = {
  structure: "list",
  capacity: 0,
  values: [],
  highlights: [],
  top: -1,
  head: -1,
  tail: -1,
  operation: "",
};

const insertHeadSnapshot: StructureSnapshot = {
  structure: "list",
  capacity: 2,
  values: [3, 5],
  highlights: [
    { index: 0, role: "added" },
    { index: 1, role: "shifted" },
  ],
  top: -1,
  head: -1,
  tail: -1,
  operation: "insert 0 3",
};

const removeMiddleSnapshot: StructureSnapshot = {
  structure: "list",
  capacity: 3,
  values: [3, 5, 7],
  highlights: [
    { index: 1, role: "shifted" },
    { index: 2, role: "shifted" },
    { index: 3, role: "removed" },
  ],
  top: -1,
  head: -1,
  tail: -1,
  operation: "remove 1",
};

const stackSnapshot: StructureSnapshot = {
  structure: "stack",
  capacity: 3,
  values: [5, 7],
  highlights: [],
  top: 1,
  operation: "push 7",
};

describe("isStructureFrame список", () => {
  it("принимает снимок списка с нулевой ёмкостью (пустой список)", () => {
    expect(isStructureFrame(structureFrame(startSnapshot))).toBe(true);
  });

  it("принимает снимок списка без полей head/tail (они опциональны)", () => {
    const without: Record<string, unknown> = { ...insertHeadSnapshot };
    delete without.head;
    delete without.tail;
    expect(isStructureFrame(structureFrame(without))).toBe(true);
  });

  it("отвергает отрицательную ёмкость", () => {
    expect(isStructureFrame(structureFrame({ ...startSnapshot, capacity: -1 }))).toBe(false);
  });
});

describe("renderFrame список", () => {
  it("стартовый кадр пустого списка: корректный пустой кадр без ячеек и меток", () => {
    const { ok, containerEl } = renderToDOM(structureFrame(startSnapshot));

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell")).toHaveLength(0); // нулевой ряд — только существующие значения
    expect(containerEl.querySelectorAll(".empty")).toHaveLength(0); // пустых ячеек списка не рисуем
    expect(containerEl.querySelectorAll("[data-pointers]")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Исходное состояние");
  });

  it("рисует только существующие значения и подсвечивает роли вставки со сдвигом", () => {
    const { explanationEl, containerEl } = renderToDOM(structureFrame(insertHeadSnapshot));

    expect(explanationEl.textContent).toBe("пояснение шага");
    const cells = containerEl.querySelectorAll(".cell");
    expect(cells).toHaveLength(2); // capacity = values.length, без пустых ячеек
    expect(cells[0].textContent).toBe("3");
    expect(cells[1].textContent).toBe("5");
    expect(cells[0].classList.contains("added")).toBe(true);
    expect(cells[1].classList.contains("shifted")).toBe(true);
    expect(containerEl.querySelectorAll(".empty")).toHaveLength(0);
    expect(containerEl.querySelectorAll("[data-pointers]")).toHaveLength(0); // указатели списку не нужны
  });

  it("кадр удаления из середины: роли shifted и removed на ячейках", () => {
    const { containerEl } = renderToDOM(structureFrame(removeMiddleSnapshot));

    const cells = containerEl.querySelectorAll(".cell");
    expect(cells).toHaveLength(3);
    expect(cells[1].classList.contains("shifted")).toBe(true);
    expect(cells[2].classList.contains("shifted")).toBe(true);
    // Освободившаяся ячейка за пределами новых значений: подсветка удалена,
    // третьей ячейки в ряду нет — рисуются только существующие значения.
    expect(containerEl.querySelectorAll(".removed")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toContain("remove 1");
  });

  it("снимок стека рисуется как раньше: класс top, ряд на всю ёмкость", () => {
    const { containerEl } = renderToDOM(structureFrame(stackSnapshot));

    const cells = containerEl.querySelectorAll(".cell");
    expect(cells).toHaveLength(3); // фиксированная ёмкость, пустая ячейка видна
    expect(cells[2].classList.contains("empty")).toBe(true);
    expect(cells[1].classList.contains("top")).toBe(true);
  });
});
