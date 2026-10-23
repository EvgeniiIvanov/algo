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

/** Снимок сортировки: массив [3,1,2], сравнивается пара 0..1. Переопределяемые
 * поля передаются как переопределения; явные undefined отфильтровываются,
 * чтобы Partial-переопределения не «затирали» обязательные поля снимка. */
function snapshot(overrides?: Partial<SortSnapshot>): SortSnapshot {
  const base: SortSnapshot = {
    values: [3, 1, 2],
    highlights: [
      { index: 0, role: "compare" },
      { index: 1, role: "compare" },
    ],
    operation: "сравнение 0 1",
    gap: -1, // шаг без состояния вставки (стартовый кадр, bubble/selection)
    key: 0,
    ranges: [],
  };
  if (overrides === undefined) return base;
  const clean = Object.fromEntries(
    Object.entries(overrides).filter(([, v]) => v !== undefined),
  );
  return { ...base, ...clean } as SortSnapshot;
}

describe("isSortFrame", () => {
  it("принимает снимок с ролями compare/swap и без подсветок", () => {
    expect(isSortFrame(sortFrame(snapshot()))).toBe(true);
    expect(
      isSortFrame(sortFrame({ values: [1, 2, 3], highlights: [], operation: "", gap: -1, key: 0 })),
    ).toBe(true);
  });

  it("принимает состояние вставки insertion sort: дырка в массиве и удерживаемый ключ", () => {
    expect(isSortFrame(sortFrame(snapshot({ values: [3, 3, 2], gap: 0, key: 1 })))).toBe(true);
  });

  it("принимает пустой массив — легальный вход сортировки", () => {
    expect(
      isSortFrame(sortFrame({ values: [], highlights: [], operation: "", gap: -1, key: 0 })),
    ).toBe(true);
  });

  it("принимает активные диапазоны merge sort", () => {
    expect(
      isSortFrame(
        sortFrame(
          snapshot({
            values: [1, 3, 2],
            ranges: [{ lo: 0, mid: 1, hi: 3 }],
          }),
        ),
      ),
    ).toBe(true);
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
    expect(isSortFrame(sortFrame({ ...base, gap: 3 }))).toBe(false); // дырка вне массива
    expect(isSortFrame(sortFrame({ ...base, gap: -2 }))).toBe(false); // дырка не -1 и не индекс
    expect(isSortFrame(sortFrame({ ...base, gap: "0" }))).toBe(false); // дырка не число
    expect(isSortFrame(sortFrame({ ...base, key: "5" }))).toBe(false); // ключ не число
    expect(isSortFrame(sortFrame({ ...base, key: undefined, gap: 0 }))).toBe(false); // gap без key
    expect(isSortFrame(sortFrame({ ...base, gap: undefined, key: 5 }))).toBe(false); // key без gap
  });

  it("отвергает кривые активные диапазоны merge sort", () => {
    const base = snapshot({ values: [1, 3, 2] });
    expect(isSortFrame(sortFrame({ ...base, ranges: {} }))).toBe(false); // не массив
    expect(isSortFrame(sortFrame({ ...base, ranges: [{ lo: "0", mid: 1, hi: 3 }] }))).toBe(false); // lo не число
    expect(isSortFrame(sortFrame({ ...base, ranges: [{ lo: 0, mid: 1, hi: 5 }] }))).toBe(false); // hi вне массива
    expect(isSortFrame(sortFrame({ ...base, ranges: [{ lo: 2, mid: 1, hi: 3 }] }))).toBe(false); // lo после hi
    expect(isSortFrame(sortFrame({ ...base, ranges: [{ lo: 0, mid: 0, hi: 3 }] }))).toBe(false); // mid не делит пополам
    expect(isSortFrame(sortFrame({ ...base, ranges: [{ lo: 0, mid: 3, hi: 3 }] }))).toBe(false); // mid на границе
  });

  it("старый снимок без ranges читается как шаг без активных диапазонов", () => {
    const oldFrame = {
      step: 1,
      kind: "sort",
      explanation: "шаг",
      data: { values: [3, 1, 2], highlights: [], operation: "" },
    } as unknown as Frame;

    expect(isSortFrame(oldFrame)).toBe(true);
  });

  it("принимает диапазон partition quicksort без границы половин mid", () => {
    expect(
      isSortFrame(
        sortFrame(
          snapshot({
            values: [2, 1, 3, 4],
            ranges: [{ lo: 0, hi: 4 }],
          }),
        ),
      ),
    ).toBe(true);
    // Пустой отрезок диапазонов по-прежнему невалиден.
    expect(
      isSortFrame(
        sortFrame(snapshot({ values: [2, 1], ranges: [{ lo: 0, hi: 0 }] })),
      ),
    ).toBe(false);
    const part = snapshot({ values: [2, 1] });
    // Кривой mid у диапазона, у которого его быть не должно, — ошибка.
    expect(
      isSortFrame(sortFrame({ ...part, ranges: [{ lo: 0, hi: 2, mid: "1" }] })),
    ).toBe(false);
    expect(
      isSortFrame(sortFrame({ ...part, ranges: [{ lo: 0, hi: 2, mid: 0 }] })),
    ).toBe(false);
  });

  it("некорректный снимок вызывает ошибку — ветка рендера не рисует мусор", () => {
    expect(() => renderToDOM(sortFrame({ values: "целиком не массив" }))).toThrow();
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
          operation: "обмен 1 2",
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
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: обмен 1 2");
  });

  it("стартовый кадр без подсветок: подпись «Исходное состояние»", () => {
    const { containerEl } = renderToDOM(sortFrame(snapshot({ highlights: [], operation: "" })));

    expect(containerEl.querySelector(".operation")?.textContent).toBe("Исходное состояние");
    expect(containerEl.querySelectorAll(".cell.compare, .cell.swap")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".cell.filled")).toHaveLength(3);
  });

  it("подсвечивает текущий минимум прохода selection sort отдельной ролью", () => {
    // Кадр сравнения selection sort: позиция 0 — текущий минимум (min),
    // позиция 1 — проверяемый кандидат (compare).
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [3, 1, 2],
          highlights: [
            { index: 0, role: "min" },
            { index: 1, role: "compare" },
          ],
          operation: "сравнение 0 1",
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.min")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.compare")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.swap")).toHaveLength(0);
  });

  it("подсвечивает сдвиг и вставку insertion sort отдельными ролями", () => {
    // Кадр вставки insertion sort: позиция 2 — куда встал ключ (insert);
    // на кадре сдвига источником и приёмником выступает роль shift.
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [1, 3, 3],
          highlights: [
            { index: 1, role: "shift" },
            { index: 2, role: "shift" },
            { index: 0, role: "insert" },
          ],
          operation: "вставка на позицию 0",
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.shift")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".cell.insert")).toHaveLength(1);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: вставка на позицию 0");
  });

  it("подсвечивает запись слияния merge sort без разметки половин источника", () => {
    // Кадр записи слияния [0,3) с границей 1: запись на позицию 0.
    // Позиция записи — приёмник слияния: половинные классы in-left/in-right
    // на кадре записи не рисуются, иначе выход слияния выглядел бы как
    // левая/правая половина источника вопреки пояснению («взято из левой»).
    // Источник читается в тексте пояснения, а роль write отмечает ячейку.
    const { ok, explanationEl, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [1, 2, 2],
          highlights: [{ index: 0, role: "write" }],
          operation: "запись 0",
          ranges: [],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(explanationEl.textContent).toBe("пояснение шага");
    expect(containerEl.querySelectorAll(".cell.write")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.in-left, .cell.in-right")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: запись 0");
  });

  it("подсвечивает сравнение слияния на фоне половин диапазона", () => {
    // Кадр сравнения слияния [0,3) с границей 1: сравниваемые головы
    // 0 (левая половина) и 1 (правая) подсвечены ролями compare, а сам
    // диапазон размечен половинами.
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [1, 2, 2],
          highlights: [
            { index: 0, role: "compare" },
            { index: 1, role: "compare" },
          ],
          operation: "сравнение 0 1",
          ranges: [{ lo: 0, mid: 1, hi: 3 }],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.compare")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".cell.in-left")).toHaveLength(1); // левая половина [0, 1)
    expect(containerEl.querySelectorAll(".cell.in-right")).toHaveLength(2); // правая [1, 3)
  });

  it("рисует кадр деления merge sort: обе половины размечены, зеркальный массив цел", () => {
    // Кадр split-события диапазона [0,3) с границей 1: деления без подсветок
    // ролей, левая половина [0,1) и правая [1,3) видны разметкой половин.
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [3, 1, 2],
          highlights: [],
          operation: "деление 0..2",
          ranges: [{ lo: 0, mid: 1, hi: 3 }],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.filled")).toHaveLength(3); // деление не меняет массив
    expect(containerEl.querySelectorAll(".cell.in-left")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.in-right")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".cell.compare, .cell.write")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: деление 0..2");
  });

  it("подсвечивает pivot, сравнение и границу на фоне разметки partition", () => {
    // Кадр сравнения quicksort: элемент на позиции 1 сравнивается с pivot
    // на позиции 3, граница блока строго меньших — позиция 0. Диапазон
    // partition [0, 4) идёт без mid: разметки половин на нём нет, а ячейки
    // диапазона получают фоновую разметку in-partition.
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [2, 4, 1, 3],
          highlights: [
            { index: 1, role: "compare" },
            { index: 3, role: "pivot" },
            { index: 0, role: "boundary" },
          ],
          operation: "сравнение 1 с pivot",
          ranges: [{ lo: 0, hi: 4 }],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.pivot")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.boundary")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.compare")).toHaveLength(1);
    // Разметка половин (merge sort) на partition-диапазоне не рисуется.
    expect(containerEl.querySelectorAll(".cell.in-left, .cell.in-right")).toHaveLength(0);
    // Активный диапазон partition: все четыре ячейки размечены in-partition.
    const partitionCells = containerEl.querySelectorAll<HTMLElement>(".cell.in-partition");
    expect(partitionCells).toHaveLength(4);
    expect(partitionCells[0]?.dataset.index).toBe("0");
    expect(partitionCells[3]?.dataset.index).toBe("3");
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: сравнение 1 с pivot");
  });

  it("разметка partition покрывает диапазон кадра обмена, не расползаясь за его границы", () => {
    // Кадр обмена внутри partition [1, 4): диапазон уже сжался — в разметку
    // попадают ровно ячейки [1, 4), а не весь массив.
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [2, 1, 4, 3],
          highlights: [
            { index: 1, role: "boundary" },
            { index: 2, role: "swap" },
            { index: 3, role: "pivot" },
          ],
          operation: "обмен 1 2",
          ranges: [{ lo: 1, hi: 4 }],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.in-partition")).toHaveLength(3);
    expect(containerEl.querySelector(".cell[data-index='0']")?.className).not.toContain("in-partition");
  });

  it("подсвечивает обмен и границы блока меньших на фоне pivot", () => {
    // Кадр обмена quicksort: строго меньшее ушло на позицию 1 (новая
    // граница boundary), участник обмена — позиция 2, pivot — позиция 3.
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [2, 1, 4, 3],
          highlights: [
            { index: 1, role: "boundary" },
            { index: 2, role: "swap" },
            { index: 3, role: "pivot" },
          ],
          operation: "обмен 1 2",
          ranges: [{ lo: 0, hi: 4 }],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.swap")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.boundary")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.pivot")).toHaveLength(1);
    // Обмен идёт внутри активной partition — весь её диапазон размечен.
    expect(containerEl.querySelectorAll(".cell.in-partition")).toHaveLength(4);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: обмен 1 2");
  });

  it("очищает разметку partition на кадре размещения pivot", () => {
    const explanationEl = document.createElement("div");
    const containerEl = document.createElement("div");

    // Сначала кадр partition с разметкой диапазона [0, 4).
    const partitionFrame = sortFrame(
      snapshot({
        values: [2, 1, 4, 3],
        highlights: [{ index: 3, role: "pivot" }],
        operation: "pivot 3",
        ranges: [{ lo: 0, hi: 4 }],
      }),
    );
    expect(renderFrame(partitionFrame, explanationEl, containerEl)).toBe(true);
    expect(containerEl.querySelectorAll(".cell.in-partition")).toHaveLength(4);

    // Затем кадр QuickPlaced: диапазона partition больше нет — разметка
    // не должна остаться от предыдущего кадра.
    const placedFrame = sortFrame(
      snapshot({
        values: [2, 1, 3, 4],
        highlights: [{ index: 2, role: "pivot" }],
        operation: "pivot на позиции 2",
        ranges: [],
      }),
    );
    expect(renderFrame(placedFrame, explanationEl, containerEl)).toBe(true);
    expect(containerEl.querySelectorAll(".cell.in-partition")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".cell.pivot")).toHaveLength(1);
  });

  it("рисует финальную позицию pivot без диапазона partition", () => {
    // Кадр QuickPlaced: pivot занял финальную позицию 2, диапазона
    // partition больше нет — завершённая partition границ не несёт.
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [2, 1, 3, 4],
          highlights: [{ index: 2, role: "pivot" }],
          operation: "pivot на позиции 2",
          ranges: [],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.pivot")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.boundary, .cell.swap")).toHaveLength(0);
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: pivot на позиции 2");
  });

  it("старый снимок без ranges остаётся валидным и рисуется без разметки половин", () => {
    const oldFrame = {
      step: 1,
      kind: "sort",
      explanation: "шаг",
      data: { values: [3, 1, 2], highlights: [], operation: "" },
    } as unknown as Frame;

    expect(isSortFrame(oldFrame)).toBe(true);
    const { ok, containerEl } = renderToDOM(oldFrame);
    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.in-left, .cell.in-right")).toHaveLength(0);
  });

  it("пустой массив рисуется без ячеек", () => {
    const { ok, containerEl } = renderToDOM(
      sortFrame({ values: [], highlights: [], operation: "", gap: -1, key: 0 }),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell")).toHaveLength(0);
  });

  it("рисует пустую дырку вставки и чип удерживаемого ключа insertion sort", () => {
    // Кадр сдвига insertion sort: значение уехало из позиции 0 в позицию 1,
    // дырка на позиции 0 ждёт ключ 1 (значение массива под дыркой —
    // устаревшая копия, она не рисуется).
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [3, 3, 2],
          highlights: [{ index: 1, role: "shift" }],
          operation: "сдвиг 0 → 1",
          gap: 0,
          key: 1,
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.gap")).toHaveLength(1);
    const gapCell = containerEl.querySelector<HTMLElement>(".cell.gap");
    expect(gapCell?.textContent).toBe(""); // дырка пуста: копия под ней не рисуется
    expect(gapCell?.dataset.index).toBe("0");
    expect(containerEl.querySelectorAll(".cell.filled")).toHaveLength(2); // только значения вне дырки
    expect(containerEl.querySelector(".sort-key")?.textContent).toBe("Ключ вне массива: 1");
  });

  it("после вставки дырки и чипа нет — ключ уже в массиве", () => {
    const { ok, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [1, 3, 2],
          highlights: [{ index: 0, role: "insert" }],
          operation: "вставка на позицию 0",
          gap: -1,
          key: 0,
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.gap")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".cell.filled")).toHaveLength(3);
    expect(containerEl.querySelector(".sort-key")).toBeNull();
  });

  it("старый снимок без gap/key остаётся валидным и рисуется без UI вставки", () => {
    // Кадры bubble/selection и кадры до тикета 21 не несут полей gap/key:
    // нормализация читает их как шаг без вставки — ряд ячеек прежний.
    const oldFrame = { step: 1, kind: "sort", explanation: "шаг", data: { values: [3, 1, 2], highlights: [], operation: "" } } as unknown as Frame;

    expect(isSortFrame(oldFrame)).toBe(true);

    const { ok, containerEl } = renderToDOM(oldFrame);
    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell.filled")).toHaveLength(3); // весь ряд заполнен
    expect(containerEl.querySelectorAll(".cell.gap")).toHaveLength(0); // дырки нет
    expect(containerEl.querySelector(".sort-key")).toBeNull(); // чипа ключа нет
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
