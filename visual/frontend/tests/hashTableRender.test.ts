// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): рендер кадра
// kind="hash-table" — ряды-корзины, цепочки с коллизиями, подсветки визитов/
// попаданий/промахов/вставок/обновлений/удалений. Существующие рендеры
// (ячеек структур и узлов linked-list) остаются нетронутыми и проверяются
// своими файлами.
import { describe, expect, it } from "vitest";
import { isHashTableFrame, renderFrame } from "../src/render";
import type { Frame, HashTableSnapshot } from "../src/types";

function hashFrame(data: unknown): Frame {
  return { step: 1, kind: "hash-table", explanation: "пояснение шага", data } as Frame;
}

/** Отрендерить кадр в свежие DOM-узлы и вернуть их. */
function renderToDOM(frame: Frame): { ok: boolean; explanationEl: HTMLElement; containerEl: HTMLElement } {
  const explanationEl = document.createElement("div");
  const containerEl = document.createElement("div");
  const ok = renderFrame(frame, explanationEl, containerEl);
  return { ok, explanationEl, containerEl };
}

/** Снимок: 3 корзины, в корзине 1 цепочка из двух записей (коллизия). */
function collisionSnapshot(highlights: HashTableSnapshot["highlights"], operation: string): HashTableSnapshot {
  return {
    bucketCount: 3,
    buckets: [[], [{ key: 6, value: 20 }, { key: 1, value: 10 }], []],
    highlightBucket: 1,
    highlights,
    removed: [],
    entryCount: 2,
    operation,
  };
}

describe("isHashTableFrame", () => {
  it("принимает снимок с пустыми корзинами и выбранным bucket, а также -1 без выбора", () => {
    expect(
      isHashTableFrame(
        hashFrame({
          bucketCount: 2,
          buckets: [[], []],
          highlightBucket: 1,
          highlights: [],
          removed: [],
          entryCount: 0,
          operation: "",
        }),
      ),
    ).toBe(true);
  });

  it("отвергает снимки со сломанными полями", () => {
    const base = collisionSnapshot([], "get 1");
    expect(isHashTableFrame(hashFrame({ ...base, bucketCount: 0 }))).toBe(false);
    expect(isHashTableFrame(hashFrame({ ...base, buckets: [[], []] }))).toBe(false); // длина ≠ bucketCount
    expect(isHashTableFrame(hashFrame({ ...base, entryCount: 5 }))).toBe(false);
    expect(isHashTableFrame(hashFrame({ ...base, highlightBucket: 7 }))).toBe(false);
    expect(isHashTableFrame(hashFrame({ ...base, highlightBucket: -2 }))).toBe(false);
    expect(isHashTableFrame(hashFrame({ ...base, operation: 5 }))).toBe(false);
    // Отрицательный индекс легален: -1 — «корзина не выбрана».
    expect(isHashTableFrame(hashFrame({ ...base, highlightBucket: -1 }))).toBe(true);
  });

  it("отвергает сломанные надгробия removed", () => {
    const base = collisionSnapshot([], "delete 6");
    expect(isHashTableFrame(hashFrame({ ...base, removed: undefined }))).toBe(false); // поля нет
    expect(isHashTableFrame(hashFrame({ ...base, removed: null }))).toBe(false);
    expect(isHashTableFrame(hashFrame({ ...base, removed: { bucket: 1 } }))).toBe(false); // не массив
    expect(isHashTableFrame(hashFrame({ ...base, removed: [{ bucket: 1, position: 0, key: "6", value: 20 }] }))).toBe(false); // ключ не число
    expect(isHashTableFrame(hashFrame({ ...base, removed: [{ bucket: 7, position: 0, key: 6, value: 20 }] }))).toBe(false); // корзины 7 нет
    expect(
      isHashTableFrame(hashFrame({ ...base, removed: [{ bucket: 1, position: 0, key: 6, value: 20 }] })),
    ).toBe(true);
  });
});

describe("renderFrame хеш-таблица", () => {
  it("стартовый кадр: ряд пустых корзин, подпись «Исходное состояние»", () => {
    const { ok, containerEl } = renderToDOM(
      hashFrame({
        bucketCount: 2,
        buckets: [[], []],
        highlightBucket: -1,
        highlights: [],
        removed: [],
        entryCount: 0,
        operation: "",
      }),
    );

    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".hash-table-row")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".hash-entry.empty")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".hash-entry.filled")).toHaveLength(0);
    expect(containerEl.querySelector(".selected")).toBeNull();
    expect(containerEl.querySelector(".operation")?.textContent).toBe("Исходное состояние");
  });

  it("коллизия: цепочка из двух записей в одном ряду со стрелкой next", () => {
    const { containerEl } = renderToDOM(hashFrame(collisionSnapshot([], "put 6 20")));

    const row = containerEl.querySelector('[data-bucket="1"]');
    expect(row).not.toBeNull();
    const entries = row?.querySelectorAll(".hash-entry.filled") ?? [];
    expect(entries).toHaveLength(2); // обе записи доступны
    expect(entries[0].textContent).toBe("6:20"); // голова цепочки — последняя вставка
    expect(entries[1].textContent).toBe("1:10");
    expect(row?.querySelectorAll(".arrow")).toHaveLength(1); // одна связь next между записями
    // Корзина 0 и 2 пусты: по одному пустому маркеру в каждом ряду.
    expect(containerEl.querySelector('[data-bucket="0"] .hash-entry.empty')).not.toBeNull();
    expect(containerEl.querySelector('[data-bucket="2"] .hash-entry.empty')).not.toBeNull();
  });

  it("выбранная корзина подчёркивается целиком, записи — своей ролью", () => {
    const { containerEl: visitedRow } = renderToDOM(
      hashFrame(collisionSnapshot([{ bucket: 1, position: 0, role: "visited" }], "get 1")),
    );
    const selected = visitedRow.querySelector(".hash-table-row.selected");
    expect(selected).not.toBeNull();
    expect(selected?.getAttribute("data-bucket")).toBe("1");
    const visited = selected?.querySelector('[data-position="0"]');
    expect(visited?.classList.contains("visited")).toBe(true);
    // Чужая запись цепочки без роли.
    expect(selected?.querySelector('[data-position="1"]')?.classList.contains("visited")).toBe(false);

    // Попадание: роль found на записи с искомым ключом.
    const { containerEl: foundRow } = renderToDOM(
      hashFrame(collisionSnapshot([{ bucket: 1, position: 1, role: "found" }], "get 1")),
    );
    expect(foundRow.querySelector('[data-bucket="1"] [data-position="1"]')?.classList.contains("found")).toBe(true);

    // Обновление: роль updated, цепочка и число записей прежние.
    const { containerEl: updatedRow } = renderToDOM(
      hashFrame(collisionSnapshot([{ bucket: 1, position: 0, role: "updated" }], "put 6 21")),
    );
    expect(updatedRow.querySelector('[data-bucket="1"] [data-position="0"]')?.classList.contains("updated")).toBe(true);
  });

  it("оба результата поиска: попадание и промах рисуются как кадры", () => {
    // Попадание: роль found уже проверена выше; промах — подсветок записей
    // нет, выбранная корзина пуста.
    const miss: HashTableSnapshot = {
      bucketCount: 2,
      buckets: [[], []],
      highlightBucket: 1,
      highlights: [],
      removed: [],
      entryCount: 0,
      operation: "get 9",
    };
    const { containerEl } = renderToDOM(hashFrame(miss));

    expect(containerEl.querySelectorAll(".hash-table-row.selected")).toHaveLength(1);
    expect(containerEl.querySelectorAll(".hash-entry.filled")).toHaveLength(0); // ключ не нашёлся
    expect(containerEl.querySelectorAll(".added, .visited, .found, .removed, .updated")).toHaveLength(0);  });

  it("удаление из головы цепочки: надгробие на бывшей позиции, сосед не красится removed", () => {
    // Настоящая форма кадра после удаления (как в SSE-тесте):
    // цепочка [6:20, 1:10], delete 6 → снимок хранит только [1:10],
    // удалённая пара живёт надгробием со своей бывшей позицией.
    const afterHeadRemoval: HashTableSnapshot = {
      ...collisionSnapshot([], "delete 6"),
      buckets: [[], [{ key: 1, value: 10 }], []],
      entryCount: 1,
      removed: [{ bucket: 1, position: 0, key: 6, value: 20 }],
    };
    const { containerEl } = renderToDOM(hashFrame(afterHeadRemoval));

    const row = containerEl.querySelector('[data-bucket="1"]');
    expect(row).not.toBeNull();
    // Надгробие: удалённая пара 6:20 на бывшей позиции 0.
    const ghost = row?.querySelector(".hash-entry.removed");
    expect(ghost?.textContent).toBe("6:20");
    expect(ghost?.getAttribute("data-bucket")).toBe("1");
    expect(ghost?.getAttribute("data-position")).toBe("0");
    // Запись 1, вставшая на позицию 0 после удаления, доступна и без роли removed.
    const survivor = row?.querySelector(".hash-entry.filled");
    expect(survivor?.textContent).toBe("1:10");
    expect(survivor?.classList.contains("removed")).toBe(false);
    // Стрелка между надгробием и соседом — прерывистая (цепочка идёт мимо).
    expect(row?.querySelectorAll(".arrow.skip")).toHaveLength(1);
  });

  it("удаление из середины цепочки: надгробие между соседями, соседи живы", () => {
    // Ровно случай регрессии: удаляется запись на позиции 1, и прежняя
    // подсветка removed красила бы выжившую запись, вставшую на её место.
    const afterMidRemoval: HashTableSnapshot = {
      bucketCount: 3,
      buckets: [[], [{ key: -4, value: 40 }, { key: 1, value: 10 }], []],
      highlightBucket: 1,
      highlights: [],
      entryCount: 2,
      operation: "delete 6",
      removed: [{ bucket: 1, position: 1, key: 6, value: 20 }],
    };
    const { containerEl } = renderToDOM(hashFrame(afterMidRemoval));

    const row = containerEl.querySelector('[data-bucket="1"]');
    const ghostAt = [...(row?.querySelectorAll(".hash-entry.removed") ?? [])];
    expect(ghostAt).toHaveLength(1); // только надгробие красится removed
    expect(ghostAt[0].textContent).toBe("6:20");
    // Оба соседа живы и доступны: до и после надгробия.
    const filled = row?.querySelectorAll(".hash-entry.filled") ?? [];
    expect(filled).toHaveLength(2);
    expect(filled[0].textContent).toBe("-4:40");
    expect(filled[1].textContent).toBe("1:10");
    expect(filled[0].classList.contains("removed")).toBe(false);
    expect(filled[1].classList.contains("removed")).toBe(false);
  });

  it("удаление единственной записи корзины: надгробие вместо пустого маркера", () => {
    const afterLastRemoval: HashTableSnapshot = {
      bucketCount: 2,
      buckets: [[], []],
      highlightBucket: 1,
      highlights: [],
      entryCount: 0,
      operation: "delete 5",
      removed: [{ bucket: 1, position: 0, key: 5, value: 50 }],
    };
    const { containerEl } = renderToDOM(hashFrame(afterLastRemoval));

    const row = containerEl.querySelector('[data-bucket="1"]');
    expect(row?.querySelector(".hash-entry.removed")?.textContent).toBe("5:50");
    // Корзина 0 пуста и без надгробий — пустой маркер на месте.
    expect(containerEl.querySelector('[data-bucket="0"] .hash-entry.empty')).not.toBeNull();
    // Шаг без надгробий в пустой корзине рисуется по-старому.
    expect(row?.querySelectorAll(".hash-entry.empty")).toHaveLength(0);
  });

  it("шаг без удаления не рисует надгробий (removed пуст)", () => {
    const { containerEl } = renderToDOM(hashFrame(collisionSnapshot([{ bucket: 1, position: 1, role: "found" }], "get 1")));
    expect(containerEl.querySelectorAll(".hash-entry.removed")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".arrow.skip")).toHaveLength(0);
  });

  it("слои кроме хеш-таблицы рендерятся как раньше", () => {
    // Кадр структуры рисуется прежней веткой, корзин не появляется.
    const explanationEl = document.createElement("div");
    const containerEl = document.createElement("div");
    const ok = renderFrame(
      { step: 1, kind: "structure", explanation: "шаг", data: {
        structure: "stack",
        capacity: 2,
        values: [5],
        highlights: [],
        top: 0,
        operation: "push 5",
      } } as Frame,
      explanationEl,
      containerEl,
    );
    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".cell")).toHaveLength(2);
    expect(containerEl.querySelectorAll(".hash-table-row")).toHaveLength(0);

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
