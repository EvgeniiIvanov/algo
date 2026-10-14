// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): рендер разновидности
// linked-list кадра kind="structure" — узлы со стрелками next, разрывы
// между узлами, указатель head, подсветка узла обхода. Слайсовые
// рендеры (stack/queue/list) проверяются в своих файлах и остаются нетронутыми.
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

const emptySnapshot: StructureSnapshot = {
  structure: "linked-list",
  capacity: 0,
  values: [],
  highlights: [],
  top: -1,
  head: -1,
  tail: -1,
  operation: "",
};

const appendedSnapshot: StructureSnapshot = {
  // Сценарий: prepend 4, append 7 (кадр связи). Узлы: 4 → 7.
  structure: "linked-list",
  capacity: 2,
  values: [4, 7],
  highlights: [{ index: 0, role: "relinked" }],
  top: -1,
  head: 0,
  tail: 1,
  operation: "append 7",
};

const visitingSnapshot: StructureSnapshot = {
  // Вставка по позиции: посещён узел 0 со значением 4.
  structure: "linked-list",
  capacity: 2,
  values: [4, 7],
  highlights: [{ index: 0, role: "visited" }],
  top: -1,
  head: 0,
  tail: 1,
  operation: "insert 1 9",
};

const foundSnapshot: StructureSnapshot = {
  // find 7: узел найден на позиции 1.
  structure: "linked-list",
  capacity: 2,
  values: [4, 7],
  highlights: [{ index: 1, role: "found" }],
  top: -1,
  head: 0,
  tail: 1,
  operation: "find 7",
};

const removeHeadSnapshot: StructureSnapshot = {
  // remove 0: голова 4 удалена; надгробие на бывшей позиции 0.
  structure: "linked-list",
  capacity: 1,
  values: [7],
  highlights: [{ index: 0, role: "removed" }],
  top: -1,
  head: 0,
  tail: 0,
  operation: "remove 0",
};

const stackSnapshot: StructureSnapshot = {
  structure: "stack",
  capacity: 3,
  values: [5, 7],
  highlights: [],
  top: 1,
  operation: "push 7",
};

describe("isStructureFrame связный список", () => {
  it("принимает снимок linked-list с нулевой ёмкостью (пустой список)", () => {
    expect(isStructureFrame(structureFrame(emptySnapshot))).toBe(true);
  });

  it("принимает снимок linked-list без полей head/tail (они опциональны)", () => {
    const without: Record<string, unknown> = { ...appendedSnapshot };
    delete without.head;
    delete without.tail;
    expect(isStructureFrame(structureFrame(without))).toBe(true);
  });

  it("отвергает снимок linked-list, где capacity не совпадает с числом значений", () => {
    expect(isStructureFrame(structureFrame({ ...appendedSnapshot, capacity: 3 }))).toBe(false);
    expect(isStructureFrame(structureFrame({ ...emptySnapshot, values: [1], capacity: 0 }))).toBe(false);
  });
});

describe("renderFrame связный список", () => {
  it("стартовый кадр пустого списка: ни узлов, ни стрелок, подпись «Исходное состояние»", () => {
    const { ok, containerEl } = renderToDOM(structureFrame(emptySnapshot));

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".arrow")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Исходное состояние");
  });

  it("рисует существующие узлы со стрелкой next между соседями и меткой head", () => {
    const { explanationEl, containerEl } = renderToDOM(structureFrame(appendedSnapshot));

    expect(explanationEl.textContent).toBe("пояснение шага");
    const cells = containerEl.querySelectorAll(".cell.filled");
    expect(cells).toHaveLength(2);
    expect(cells[0].textContent).toBe("4");
    expect(cells[1].textContent).toBe("7");
    // Один разрыв на два узла — память не непрерывна, но цепочка связна.
    expect(containerEl.querySelectorAll(".arrow")).toHaveLength(1);
    // Метка head под узлом 0 (совпадает с tail на единственном узле).
    const headCell = containerEl.querySelector('[data-pointers="head"]');
    expect(headCell).not.toBeNull();
    expect(headCell?.textContent).toBe("4");
    const tailCell = containerEl.querySelector('[data-pointers="tail"]');
    expect(tailCell?.textContent).toBe("7");
    // Роль relinked: узел 0 подсвечен переключением связи.
    expect(cells[0].classList.contains("relinked")).toBe(true);
  });

  it("подсветка узла обхода (visited) и найденного узла (found)", () => {
    const { containerEl: visitedRow } = renderToDOM(structureFrame(visitingSnapshot));
    const visitedCells = visitedRow.querySelectorAll(".cell.filled");
    expect(visitedCells[0].classList.contains("visited")).toBe(true);
    expect(visitedCells[1].classList.contains("visited")).toBe(false);

    const { containerEl: foundRow } = renderToDOM(structureFrame(foundSnapshot));
    const foundCells = foundRow.querySelectorAll(".cell.filled");
    expect(foundCells[1].classList.contains("found")).toBe(true);
  });

  it("удаление головы: узлы без сдвига и ячейка-надгробие на бывшей позиции 0", () => {
    const { containerEl } = renderToDOM(structureFrame(removeHeadSnapshot));

    const cells = containerEl.querySelectorAll(".cell") as NodeListOf<HTMLElement>;
    expect(cells).toHaveLength(2); // один живой узел + надгробие
    // Надгробие стоит на бывшей позиции 0 (перед оставшимся узлом).
    expect(cells[0].classList.contains("removed")).toBe(true);
    expect(cells[0].textContent).toBe("");
    expect(cells[1].textContent).toBe("7");
    // Стрелки: живой узел один, полноценных стрелок нет; надгробие соединено
    // со следующим узлом прерывистой стрелкой.
    expect(containerEl.querySelectorAll(".arrow:not(.skip)")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".arrow.skip")).toHaveLength(1);
  });

  it("слайсовый снимок рисуется как раньше: ряд ячеек без стрелок", () => {
    const { containerEl } = renderToDOM(
      structureFrame({
        structure: "list",
        capacity: 2,
        values: [3, 5],
        highlights: [],
        top: -1,
        head: -1,
        tail: -1,
        operation: "insert 1 5",
      }),
    );

    expect(containerEl.querySelectorAll(".cell")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".arrow")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".linked-list-row")).toHaveLength(0);
  });

  it("снимок стека рисуется как раньше: класс top, ряд на всю ёмкость", () => {
    const { containerEl } = renderToDOM(structureFrame(stackSnapshot));

    const cells = containerEl.querySelectorAll(".cell");
    expect(cells).toHaveLength(3); // фиксированная ёмкость, пустая ячейка видна
    expect(cells[2].classList.contains("empty")).toBe(true); // только 2 значения из 3
    expect(cells[1].classList.contains("top")).toBe(true);
  });
});
