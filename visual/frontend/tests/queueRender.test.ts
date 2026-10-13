// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// рендер кадра kind="structure" для очереди — та же ветка рендера, что у
// стека: ряд ячеек, подсветки ролей и метки указателей head/tail.
// Ветка стека при этом не меняется: у снимка стека head/tail отсутствуют.
import { describe, expect, it } from "vitest";
import { isStructureFrame, renderFrame } from "../src/render";
import { StructureSnapshot } from "../src/types";

function structureFrame(data: unknown) {
  return { step: 1, kind: "structure", explanation: "пояснение шага", data } as never;
}

/** Отрендерить кадр в свежие DOM-узлы и вернуть их. */
function renderToDOM(frame: never): { ok: boolean; explanationEl: HTMLElement; containerEl: HTMLElement } {
  const explanationEl = document.createElement("div");
  const containerEl = document.createElement("div");
  const ok = renderFrame(frame, explanationEl, containerEl);
  return { ok, explanationEl, containerEl };
}

const baseSnapshot: StructureSnapshot = {
  structure: "queue",
  capacity: 3,
  values: [7, 9],
  highlights: [
    { index: 0, role: "shifted" },
    { index: 1, role: "shifted" },
    { index: 2, role: "removed" },
  ],
  top: -1,
  head: 0,
  tail: 1,
  operation: "enqueue 9",
};

const startSnapshot: StructureSnapshot = {
  structure: "queue",
  capacity: 3,
  values: [],
  highlights: [],
  top: -1,
  head: -1,
  tail: -1,
  operation: "",
};

const oneElementSnapshot: StructureSnapshot = {
  structure: "queue",
  capacity: 2,
  values: [7],
  highlights: [
    { index: 0, role: "shifted" },
    { index: 1, role: "removed" },
  ],
  top: -1,
  head: 0,
  tail: 0,
  operation: "dequeue",
};

const stackSnapshot: StructureSnapshot = {
  structure: "stack",
  capacity: 3,
  values: [5, 7],
  highlights: [],
  top: 1,
  operation: "push 7",
};

describe("isStructureFrame очередь", () => {
  it("принимает корректный снимок очереди с head/tail", () => {
    expect(isStructureFrame(structureFrame(baseSnapshot))).toBe(true);
  });

  it("принимает снимок очереди без полей head/tail (они опциональны)", () => {
    const without: Record<string, unknown> = { ...baseSnapshot };
    delete without.head;
    delete without.tail;
    expect(isStructureFrame(structureFrame(without))).toBe(true);
  });

  it("отвергает нулевую заданную ёмкость очереди", () => {
    expect(isStructureFrame(structureFrame({ ...baseSnapshot, capacity: 0 }))).toBe(false);
  });

  it("отвергает снимок с дробным head", () => {
    expect(isStructureFrame(structureFrame({ ...baseSnapshot, head: 0.5 }))).toBe(false);
  });
});

describe("renderFrame очередь", () => {
  it("рисует ряд ячеек на всю ёмкость: значения, пустые и подсветки ролей", () => {
    const { ok, containerEl } = renderToDOM(structureFrame(baseSnapshot));

    expect(ok).toBe(true);
    const cells = containerEl.querySelectorAll(".cell") as NodeListOf<HTMLElement>;
    expect(cells).toHaveLength(3);
    expect(cells[0].textContent).toBe("7");
    expect(cells[1].textContent).toBe("9");
    expect(cells[2].classList.contains("empty")).toBe(true);
    expect(cells[0].classList.contains("shifted")).toBe(true);
    expect(cells[1].classList.contains("shifted")).toBe(true); // оба оставшихся значения сдвинулись влево
    expect(cells[2].classList.contains("removed")).toBe(true);
  });

  it("ставит метки head и tail под их ячейками", () => {
    const { containerEl } = renderToDOM(structureFrame(baseSnapshot));

    const cells = containerEl.querySelectorAll(".cell") as NodeListOf<HTMLElement>;
    expect(cells[0].dataset.pointers).toBe("head");
    expect(cells[1].dataset.pointers).toBe("tail");
    expect(cells[2].dataset.pointers).toBeUndefined();
  });

  it("единственный элемент — одновременно голова и хвост: обе метки на одной ячейке", () => {
    const { containerEl } = renderToDOM(structureFrame(oneElementSnapshot));

    const cells = containerEl.querySelectorAll(".cell") as NodeListOf<HTMLElement>;
    expect(cells[0].dataset.pointers).toBe("head · tail");
    expect(cells[1].dataset.pointers).toBeUndefined();
  });

  it("стартовый кадр пустой очереди: head/tail роли нет, меток нет", () => {
    const { containerEl } = renderToDOM(structureFrame(startSnapshot));

    const cells = containerEl.querySelectorAll(".cell") as NodeListOf<HTMLElement>;
    expect(cells).toHaveLength(3);
    expect(containerEl.querySelectorAll("[data-pointers]")).toHaveLength(0);
    for (const cell of cells) {
      expect(cell.classList.contains("empty")).toBe(true);
    }
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Исходное состояние");
  });

  it("снимок стека рисуется как раньше: класс top, меток head/tail нет", () => {
    const { ok, containerEl } = renderToDOM(structureFrame(stackSnapshot));

    expect(ok).toBe(true);
    const cells = containerEl.querySelectorAll(".cell") as NodeListOf<HTMLElement>;
    expect(cells[1].classList.contains("top")).toBe(true);
    expect(containerEl.querySelectorAll("[data-pointers]")).toHaveLength(0);
  });
});
