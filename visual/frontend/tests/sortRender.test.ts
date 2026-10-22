// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): рендер кадра
// kind="sort" — ряд ячеек всего массива с подсветками сравнения/обмена.
// Существующие рендеры (массив бинарного поиска, ячейки структур,
// узлы linked-list, корзины хеш-таблицы) остаются нетронутыми.
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
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

  it("подсвечивает запись слияния merge sort и размечает активные половины диапазона", () => {
    // Кадр слияния [0,3) с границей 1: запись на позицию 0, левая половина —
    // ячейка 0, правая — ячейки 1–2.
    const { ok, explanationEl, containerEl } = renderToDOM(
      sortFrame(
        snapshot({
          values: [1, 2, 2],
          highlights: [{ index: 0, role: "write" }],
          operation: "запись 0",
          ranges: [{ lo: 0, mid: 1, hi: 3 }],
        }),
      ),
    );

    expect(ok).toBe(true);
    expect(explanationEl.textContent).toBe("пояснение шага");
    expect(containerEl.querySelectorAll(".cell.write")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".cell.in-left")).toHaveLength(1); // левая половина [0, 1)
    expect(containerEl.querySelectorAll(".cell.in-right")).toHaveLength(2); // правая [1, 3)
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Шаг: запись 0");
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

// Каскад CSS разметки половин и ролей шага: правила .cell.in-left/.cell.in-right
// и роли .cell.compare/.cell.write имеют равную специфичность — победителя
// каскада задаёт порядок правил в файле. Регрессия: разметка половин стоит
// до ролей, и роли шага с белым текстом не теряют фон в пользу фоновой
// разметки половин (иначе белый текст на светлом фоне нечитаем).
describe("style.css — каскад ролей поверх разметки половин merge sort", () => {
  const css = readFileSync(resolve(process.cwd(), "src/style.css"), "utf8");
  const ruleIndex = (selector: string): number => css.indexOf(`${selector} {`);

  it("разметка половин стоит до ролей шага — роли шага выигрывают каскад", () => {
    const inLeft = ruleIndex(".cell.in-left");
    const inRight = ruleIndex(".cell.in-right");
    const compare = ruleIndex(".cell.compare");
    const write = ruleIndex(".cell.write");
    for (const at of [inLeft, inRight]) {
      expect(at).toBeGreaterThan(-1); // все правила присутствуют
      expect(compare).toBeGreaterThan(at);
      expect(write).toBeGreaterThan(at);
    }
  });

  it("роли шага с белым текстом задают несветлый фон — поверх светлой разметки половин читаемы", () => {
    for (const at of [ruleIndex(".cell.compare"), ruleIndex(".cell.write")]) {
      const body = css.slice(at, css.indexOf("}", at));
      expect(body).toContain("color: white");
      // Светлые фоны разметки половин не должны повторяться в ролях шага.
      expect(body).not.toContain("#eef2ff");
      expect(body).not.toContain("#fdf2f8");
    }
  });
});
