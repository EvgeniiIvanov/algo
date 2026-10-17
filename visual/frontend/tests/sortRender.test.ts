// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): рендер кадра
// kind="sort" — ряд ячеек всего массива с подсветками сравнения/обмена.
// Существующие рендеры (массив бинарного поиска, ячейки структур,
// узлы linked-list, корзины хеш-таблицы) остаются нетронутыми.
import { describe, expect, it } from "vitest";
import { isSortFrame, renderFrame } from "../src/render";
import type { Frame, SortSnapshot } from "../src/types";

/** Кадр kind="sort" с произвольным снимком (для проверки валидации). */
function sortFrame(data: unknown): Frame {
  return { step: 1, kind: "sort", explanation: "пояснение шага", data } as Frame;
}

/** Отрендерить кадр в свежие DOM-узлы и вернуть их. */
function renderToDOM(frame: Frame): {
  ok: boolean;
  explanationEl: HTMLElement;
  containerEl: HTMLElement;
} {
  const explanationEl = document.createElement("div");
  const containerEl = document.createElement("div");
  const ok = renderFrame(frame, explanationEl, containerEl);
  return { ok, explanationEl, containerEl };
}

/** Снимок сортировки: массив [3,1,2], сравнивается пара 0..1. */
function snapshot(overrides?: Partial<SortSnapshot>): SortSnapshot {
  return {
    values: [3, 1, 2],
    highlights: [
      { index: 0, role: "compare" },
      { index: 1, role: "compare" },
    ],
    operation: "compare 0 1",
    ...overrides,
  };
}

describe("isSortFrame", () => {
  it("принимает снимок с ролями compare/swap и без подсветок", () => {
    expect(isSortFrame(sortFrame(snapshot()))).toBe(true);
    expect(
      isSortFrame(sortFrame({ values: [1, 2, 3], highlights: [], operation: "" })),
    ).toBe(true);
  });

  it("принимает пустой массив — легальный вход сортировки", () => {
    expect(isSortFrame(sortFrame({ values: [], highlights: [], operation: "" }))).toBe(true);
  });

  it("отвергает сломанные снимки", () => {
    const base = snapshot();
    expect(isSortFrame(sortFrame(null))).toBe(false);
    expect(isSortFrame(sortFrame({ ...base, values: "3,1,2" }))).toBe(false); // не массив
    expect(isSortFrame(sortFrame({ ...base, values: [3, "1", 2] }))).toBe(false); // не число
    expect(isSortFrame(sortFrame({ ...base, highlights: {} }))).toBe(false); // не массив
    expect(isSortFrame(sortFrame({ ...base, highlights: [{ index: "0", role: "compare" }] }))).toBe(
      false,
    ); // индекс не число
    expect(isSortFrame(sortFrame({ ...base, highlights: [{ index: 0 }] }))).toBe(false); // нет role
    expect(isSortFrame(sortFrame({ ...base, highlights: [{ index: 3, role: "compare" }] }))).toBe(
      false,
    ); // индекс вне массива
    expect(isSortFrame(sortFrame({ ...base, highlights: [{ index: -1, role: "compare" }] }))).toBe(
      false,
    ); // отрицательный индекс
    expect(isSortFrame(sortFrame({ ...base, operation: 5 }))).toBe(false); // operation не строка
  });

  it("отвергает чужие kind'ы", () => {
    const f = { step: 1, kind: "array", explanation: "x", data: snapshot() } as unknown as Frame;
    expect(isSortFrame(f)).toBe(false);
  });
});

describe("renderFrame — kind=\"sort\"", () => {
  it("рисует ряд ячеек массива целиком и подсвечивает роли шага", () => {
    const { ok, explanationEl, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [1, 3, 2],
          highlights: [
            { index: 1, role: "swap" },
            { index: 2, role: "swap" },
          ],
          operation: "swap 1 2",
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(explanationEl.textContent).toBe("пояснение шага");
    const cells = containerEl.querySelectorAll(".cell.filled");
    expect(cells).toHaveLength(3); // весь массив самодостаточно
    expect(cells[0].textContent).toBe("1");
    expect(cells[1].textContent).toBe("3");
    expect(cells[2].textContent).toBe("2");
    expect(containerEl.querySelectorAll(".cell.swap")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".cell.compare")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: swap 1 2");
  });

  it("стартовый кадр без подсветок: подпись «Исходное состояние»", () => {
    const { containerEl } = renderToDOM(sortFrame(snapshot({ highlights: [], operation: "" })));

    expect(containerEl.querySelector(".operation")?.textContent).toBe("Исходное состояние");
    expect(containerEl.querySelectorAll(".cell.compare, .cell.swap")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".cell.filled")).toHaveLength(3);
  });

  it("пустой массив рисуется без ячеек", () => {
    const { ok, containerEl } = renderToDOM(sortFrame({ values: [], highlights: [], operation: "" }));

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell")).toHaveLength(0);
  });

  it("некорректный снимок вызывает ошибку — ветка рендера не рисует мусор", () => {
    expect(() => renderToDOM(sortFrame({ values: "целиком не массив" }))).toThrow();
  });

  it("существующие рендеры продолжают работать без изменений", () => {
    // kind="array" бинарного поиска — прежняя ветка: ячейки, out/mid/found.
    const explanationEl = document.createElement("div");
    const containerEl = document.createElement("div");
    const ok = renderFrame(
      {
        step: 1,
        kind: "array",
        explanation: "шаг",
        data: {
          values: [1, 3, 5],
          low: 0,
          high: 2,
          mid: 1,
          foundIndex: -1,
        },
      } as Frame,
      explanationEl,
      containerEl,
    );
    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell")).toHaveLength(3);
    expect(containerEl.querySelectorAll(".cell.mid")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".sort")).toHaveLength(0);

    // Незнакомый kind — кадр не отрисован, DOM не тронут.
    const before = containerEl.childNodes.length;
    const untouched = renderFrame(
      { step: 1, kind: "unknown", explanation: "?", data: {} } as Frame,
      explanationEl,
      containerEl,
    );
    expect(untouched).toBe(false);
    expect(containerEl.childNodes.length).toBe(before);
  });
});
