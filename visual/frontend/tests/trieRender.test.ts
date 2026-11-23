// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): рендер кадров kind="trie"
// — снимки зеркалят серверные кадры (visual/render/trie.go): произвольное
// ветвление, отметки конца слова, активный путь из существующих узлов,
// edgeCheck ({rune, state} в кадрах посещения/проверки, null в стартовых/
// pruning/result-кадрах) и result (null до кадра результата операции,
// одно значение закрытого множества в кадре результата). Идентификаторы —
// frame-local пути от корня по рунам; DOM ниже публичного renderer-
// интерфейса не проверяем.
import { describe, expect, it } from "vitest";
import { isTrieFrame, renderFrame } from "../src/render";
import type { Frame, TrieSnapshot } from "../src/types";
import { makeFrame, renderToDOM } from "./renderTestHelpers";

/** Кадр kind="trie" с типизированным снимком — обёртка общего makeFrame. */
function trieFrame(data: TrieSnapshot, explanation = "пояснение шага"): Frame {
  return makeFrame("trie", data, explanation);
}

/** Ячейка узла Trie по идентификатору пути (id кадра, не руна). */
function trieCell(
  containerEl: HTMLElement,
  id: string,
): HTMLDivElement | null {
  return containerEl.querySelector<HTMLDivElement>(
    `.trie-node[data-id="${id}"]`,
  );
}

const ID_K = "root/U+043A"; // к
const ID_O = "root/U+043A/U+043E"; // о
const ID_T = "root/U+043A/U+043E/U+0442"; // т
const ID_D = "root/U+043A/U+043E/U+0434"; // д
const ID_M = "root/U+043A/U+043E/U+043C"; // м
const ID_X = "root/U+0445"; // х
const ID_EMOJI = "root/U+1F600"; // 😀

const node = (id: string, isEnd: boolean, children: string[]) => ({
  id,
  isEnd,
  children,
});

/** Стартовый кадр Run: пустое дерево, только корень-заглушка. */
function startSnapshot(operation = ""): TrieSnapshot {
  return {
    nodes: [node("root", false, [])],
    rootId: "root",
    activePath: [],
    edgeCheck: null,
    result: null,
    operation,
  };
}

/** Полное дерево после insert кот, insert код: ветвление на узле «о». */
function catCodeTreeSnapshot(): TrieSnapshot {
  return {
    nodes: [
      node("root", false, [ID_K]),
      node(ID_K, false, [ID_O]),
      node(ID_O, false, [ID_D, ID_T]),
      node(ID_D, true, []),
      node(ID_T, true, []),
    ],
    rootId: "root",
    activePath: [],
    edgeCheck: null,
    result: null,
    operation: "insert код",
  };
}

describe("isTrieFrame: снимки Trie", () => {
  it("принимает стартовый, посещение и result-кадры", () => {
    expect(isTrieFrame(trieFrame(startSnapshot()))).toBe(true);
    expect(
      isTrieFrame(
        trieFrame({
          ...startSnapshot("insert кот"),
          nodes: [node("root", false, [ID_K]), node(ID_K, false, [])],
          activePath: ["root", ID_K],
          edgeCheck: { rune: "к", state: "created" },
        }),
      ),
    ).toBe(true);
    expect(isTrieFrame(trieFrame({ ...startSnapshot(), result: "insert" }))).toBe(
      true,
    );
  });

  it("принимает кадр missing: активный путь до родителя, потомка нет", () => {
    expect(
      isTrieFrame(
        trieFrame({
          ...startSnapshot("search ком"),
          nodes: [
            node("root", false, [ID_K]),
            node(ID_K, false, [ID_O]),
            node(ID_O, false, [ID_T]),
            node(ID_T, true, []),
          ],
          activePath: ["root", ID_K, ID_O],
          edgeCheck: { rune: "м", state: "missing" },
        }),
      ),
    ).toBe(true);
  });

  it("отвергает edgeCheck с неизвестным состоянием", () => {
    const bad = {
      ...startSnapshot(),
      edgeCheck: { rune: "к", state: "nope" },
    } as unknown as TrieSnapshot;
    expect(isTrieFrame(trieFrame(bad))).toBe(false);
  });

  it("отвергает edgeCheck.rune не из одной руны и одиночную surrogate", () => {
    expect(
      isTrieFrame(trieFrame({ ...startSnapshot(), edgeCheck: { rune: "ко", state: "reused" } })),
    ).toBe(false);
    expect(
      isTrieFrame(
        trieFrame({ ...startSnapshot(), edgeCheck: { rune: "\uD800", state: "reused" } }),
      ),
    ).toBe(false);
  });

  it("отвергает result вне закрытого множества", () => {
    const won = { ...startSnapshot(), result: "won" } as unknown as TrieSnapshot;
    expect(isTrieFrame(trieFrame(won))).toBe(false);
  });

  it("отвергает неканонические идентификаторы и surrogate-половину", () => {
    expect(
      isTrieFrame(trieFrame({ ...startSnapshot(), nodes: [node("root/u+043a", false, [])] })),
    ).toBe(false);
    expect(
      isTrieFrame(trieFrame({ ...startSnapshot(), nodes: [node("root/U+D83D", false, [])] })),
    ).toBe(false);
    // Минимальная ширина 4: U+043A валиден, трёхзначный U+43A — нет.
    expect(
      isTrieFrame(
        trieFrame({
          ...startSnapshot(),
          nodes: [node("root", false, ["root/U+043A"]), node("root/U+043A", false, [])],
        }),
      ),
    ).toBe(true);
    expect(
      isTrieFrame(trieFrame({ ...startSnapshot(), nodes: [node("root/U+43A", false, [])] })),
    ).toBe(false);
  });

  it("отвергает не-минимальные записи кода руны: U+00041 — не метка U+0041", () => {
    const treeWith = (id: string): TrieSnapshot => ({
      ...startSnapshot(),
      nodes: [node("root", false, [id]), node(id, false, [])],
    });
    // Пять hex-цифр при четырёхзначном коде: не-минимальная запись.
    expect(isTrieFrame(trieFrame(treeWith("root/U+00041")))).toBe(false);
    // То же у символа вне BMP: U+01F600 — не каноническая запись 😀.
    expect(isTrieFrame(trieFrame(treeWith("root/U+01F600")))).toBe(false);
    // Канонические записи принимаются: ширина ровно 4 и полный код вне BMP.
    expect(isTrieFrame(trieFrame(treeWith("root/U+0041")))).toBe(true);
    expect(isTrieFrame(trieFrame(treeWith(ID_EMOJI)))).toBe(true);
  });

  it("принимает канонические идентификаторы и supplementary-руны", () => {
    const treeWith = (id: string): TrieSnapshot => ({
      ...startSnapshot(),
      nodes: [node("root", false, [id]), node(id, false, [])],
    });
    // Максимальный скаляр U+10FFFF каноничен; граница не спутана с лимитом
    // живых узлов или лимитом рун.
    expect(isTrieFrame(trieFrame(treeWith("root/U+10FFFF")))).toBe(true);
    // Кириллица — минимальная ширина 4.
    expect(isTrieFrame(trieFrame(treeWith("root/U+0041/U+0042")))).toBe(true);
  });

  it("отвергает ссылку на отсутствующего потомка и разрыв активного пути", () => {
    expect(
      isTrieFrame(trieFrame({ ...startSnapshot(), nodes: [node("root", false, [ID_K])] })),
    ).toBe(false);
    expect(
      isTrieFrame(trieFrame({ ...startSnapshot(), activePath: [ID_K] })),
    ).toBe(false);
  });
});

describe("renderFrame: стартовый и посещенческие кадры", () => {
  it("рисует пустое Trie корнем-заглушкой без чипов edgeCheck/result", () => {
    const { ok, explanationEl, containerEl } = renderToDOM(
      trieFrame(startSnapshot(), "Сценарий Trie: insert кот. Дерево начинается с пустого корня."),
    );
    expect(ok).toBe(true);
    expect(explanationEl.textContent).toContain("Сценарий Trie");
    expect(trieCell(containerEl, "root")?.textContent).toBe("root");
    expect(containerEl.querySelectorAll(".trie-edge-check")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".trie-result")).toHaveLength(0);
  });

  it("подсвечивает активный путь и чипом показывает проверку ребра created", () => {
    const { ok, containerEl } = renderToDOM(
      trieFrame({
        ...startSnapshot("insert кот"),
        nodes: [node("root", false, [ID_K]), node(ID_K, false, [])],
        activePath: ["root", ID_K],
        edgeCheck: { rune: "к", state: "created" },
      }),
    );
    expect(ok).toBe(true);
    expect(trieCell(containerEl, ID_K)?.classList.contains("active")).toBe(true);
    expect(trieCell(containerEl, "root")?.classList.contains("active")).toBe(true);
    expect(containerEl.querySelector(".trie-edge-check")?.textContent).toBe(
      "Проверка ребра: «к» — новое ребро — узел создан",
    );
  });

  it("рисует произвольное ветвление и отметки конца слова", () => {
    const { ok, containerEl } = renderToDOM(trieFrame(catCodeTreeSnapshot()));
    expect(ok).toBe(true);
    // Все пять узлов нарисованы, у узла «о» два потомка (т и д).
    expect(trieCell(containerEl, "root")?.textContent).toBe("root");
    expect(trieCell(containerEl, ID_K)?.textContent).toBe("к");
    expect(trieCell(containerEl, ID_O)?.textContent).toBe("о");
    expect(trieCell(containerEl, ID_D)?.textContent).toBe("д •");
    expect(trieCell(containerEl, ID_T)?.textContent).toBe("т •");
    expect(trieCell(containerEl, ID_D)?.classList.contains("trie-end")).toBe(true);
    expect(trieCell(containerEl, ID_O)?.classList.contains("trie-end")).toBe(false);
  });

  it("кадр missing: путь из существующих узлов, отсутствующий потомок не рисуется", () => {
    const snapshot: TrieSnapshot = {
      ...catCodeTreeSnapshot(),
      operation: "search ком",
      activePath: ["root", ID_K, ID_O],
      edgeCheck: { rune: "м", state: "missing" },
    };
    const { ok, containerEl } = renderToDOM(trieFrame(snapshot));
    expect(ok).toBe(true);
    expect(trieCell(containerEl, ID_O)?.classList.contains("active")).toBe(true);
    expect(trieCell(containerEl, ID_M)).toBeNull();
    expect(containerEl.querySelectorAll(".trie-node.active")).toHaveLength(3);
    expect(containerEl.querySelector(".trie-edge-check")?.textContent).toBe(
      "Проверка ребра: «м» — ребра нет — спуск невозможен",
    );
  });

  it("промах префикса по отсутствующему ребру от корня: путь — только корень", () => {
    const { ok, containerEl } = renderToDOM(
      trieFrame({
        ...catCodeTreeSnapshot(),
        operation: "prefix пер",
        activePath: ["root"],
        edgeCheck: { rune: "х", state: "missing" },
      }),
    );
    expect(ok).toBe(true);
    expect(trieCell(containerEl, "root")?.classList.contains("active")).toBe(true);
    expect(trieCell(containerEl, ID_X)).toBeNull();
    expect(containerEl.querySelectorAll(".trie-node.active")).toHaveLength(1);
  });

  it("кадр pruning: активного пути нет, срезанный узел исчез из топологии", () => {
    const snapshot: TrieSnapshot = {
      ...startSnapshot("delete кот"),
      nodes: [
        node("root", false, [ID_K]),
        node(ID_K, false, [ID_O]),
        node(ID_O, false, []), // т уже срезан
      ],
      edgeCheck: null,
    };
    const { ok, containerEl } = renderToDOM(trieFrame(snapshot));
    expect(ok).toBe(true);
    expect(trieCell(containerEl, ID_T)).toBeNull();
    expect(trieCell(containerEl, ID_O)).not.toBeNull();
    expect(containerEl.querySelectorAll(".trie-node.active")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".trie-edge-check")).toHaveLength(0);
  });

  it("перерисовывает снимок целиком: идентификаторы не переживают кадр", () => {
    const first = renderToDOM(trieFrame(catCodeTreeSnapshot()));
    expect(trieCell(first.containerEl, ID_T)).not.toBeNull();
    // Кадр после полного pruning: только корень, тот же контейнер.
    const replaced = renderFrame(
      trieFrame(startSnapshot("delete кот")),
      first.explanationEl,
      first.containerEl,
    );
    expect(replaced).toBe(true);
    expect(trieCell(first.containerEl, ID_T)).toBeNull();
    expect(trieCell(first.containerEl, ID_K)).toBeNull();
    expect(trieCell(first.containerEl, "root")).not.toBeNull();
  });
});

describe("renderFrame: кадр результата операции и result", () => {
  const results = [
    "insert",
    "already-existed",
    "found",
    "not-found",
    "prefix-found",
    "prefix-not-found",
    "deleted",
    "not-deleted",
  ] as const;

  // Русские подписи итога: enum остаётся сырым в data-result (SRC: render.ts).
  const resultLabels: Record<(typeof results)[number], string> = {
    insert: "слово добавлено",
    "already-existed": "слово уже хранится",
    found: "слово найдено",
    "not-found": "слова нет — не найдено",
    "prefix-found": "префикс найден",
    "prefix-not-found": "префикса нет — не найдено",
    deleted: "слово удалено",
    "not-deleted": "слова нет — не удалено",
  };

  it.each(results)("рисует чип результата %s только на result-кадре", (result) => {
    const { ok, containerEl } = renderToDOM(
      trieFrame({ ...startSnapshot("search кот"), result }),
    );
    expect(ok).toBe(true);
    const chip = containerEl.querySelector(".trie-result");
    expect(chip?.textContent).toBe(`Результат: ${resultLabels[result]}`);
    // Сырой enum закрытого множества — в data-result.
    expect(chip?.getAttribute("data-result")).toBe(result);
  });

  it("на промежуточных снимках result null — чипа нет", () => {
    const { ok, containerEl } = renderToDOM(trieFrame(catCodeTreeSnapshot()));
    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".trie-result")).toHaveLength(0);
  });

  it("search ровно 100 😀 в пустом Trie: missing-кадр и result not-found", () => {
    const visit = renderToDOM(
      trieFrame({
        ...startSnapshot(`search ${"😀".repeat(100)}`),
        activePath: ["root"],
        edgeCheck: { rune: "😀", state: "missing" },
      }),
    );
    expect(visit.ok).toBe(true);
    expect(trieCell(visit.containerEl, "root")?.classList.contains("active")).toBe(true);
    expect(visit.containerEl.querySelector(".trie-edge-check")?.textContent).toBe(
      "Проверка ребра: «😀» — ребра нет — спуск невозможен",
    );
    const final = renderToDOM(
      trieFrame({ ...startSnapshot(`search ${"😀".repeat(100)}`), result: "not-found" }),
    );
    expect(final.containerEl.querySelector(".trie-result")?.textContent).toBe(
      "Результат: слова нет — не найдено",
    );
    expect(
      final.containerEl.querySelector(".trie-result")?.getAttribute("data-result"),
    ).toBe("not-found");
  });

  it("символ вне BMP: узел — root/U+1F600, не surrogate-половина", () => {
    const { ok, containerEl } = renderToDOM(
      trieFrame({
        ...startSnapshot("insert 😀"),
        nodes: [node("root", false, [ID_EMOJI]), node(ID_EMOJI, true, [])],
        activePath: ["root", ID_EMOJI],
        edgeCheck: { rune: "😀", state: "created" },
      }),
    );
    expect(ok).toBe(true);
    expect(trieCell(containerEl, ID_EMOJI)?.textContent).toBe("😀 •");
    expect(trieCell(containerEl, ID_EMOJI)?.classList.contains("active")).toBe(true);
    expect(trieCell(containerEl, "root/U+D83D")).toBeNull();
  });
});

describe("renderFrame: метки рун многосегментных идентификаторов", () => {
  const ID_P = "root/U+043F"; // п
  const ID_R = "root/U+043F/U+0440"; // р

  it("многосегментный путь (root/U+043F/U+0440): метка — руна последнего сегмента", () => {
    const { ok, containerEl } = renderToDOM(
      trieFrame({
        ...startSnapshot("insert при"),
        nodes: [
          node("root", false, [ID_P]),
          node(ID_P, false, [ID_R]),
          node(ID_R, true, []),
        ],
        activePath: ["root", ID_P, ID_R],
        edgeCheck: { rune: "р", state: "created" },
      }),
    );
    expect(ok).toBe(true);
    expect(trieCell(containerEl, "root")?.textContent).toBe("root");
    expect(trieCell(containerEl, ID_P)?.textContent).toBe("п");
    expect(trieCell(containerEl, ID_R)?.textContent).toBe("р •");
    expect(trieCell(containerEl, ID_R)?.classList.contains("trie-end")).toBe(true);
  });
});

// No-op-сценарии спеки на рендере: промахи по отсутствующему ребру
// (search, prefix, delete), существующий путь без отметки конца слова
// (search, delete) и точный поиск найденного — с активным путём из
// существующих узлов, edgeCheck по форме бэкенда и result: null до кадра
// результата операции.
describe("renderFrame: no-op-сценарии — промахи и точный поиск", () => {
  /** Дерево после insert кот: путь к-о-т, «т» — конец слова. */
  function catTreeSnapshot(operation = ""): TrieSnapshot {
    return {
      ...startSnapshot(operation),
      nodes: [
        node("root", false, [ID_K]),
        node(ID_K, false, [ID_O]),
        node(ID_O, false, [ID_T]),
        node(ID_T, true, []),
      ],
    };
  }

  it("search по полностью существующему пути без конца слова: reused-кадры, result null, итог not-found", () => {
    // search «ко» при сохранённом «кот»: путь к-о есть, «о» — не конец слова.
    const visitK = renderToDOM(
      trieFrame({
        ...catTreeSnapshot("search ко"),
        activePath: ["root", ID_K],
        edgeCheck: { rune: "к", state: "reused" },
      }),
    );
    expect(visitK.ok).toBe(true);
    expect(trieCell(visitK.containerEl, ID_K)?.classList.contains("active")).toBe(true);
    expect(trieCell(visitK.containerEl, ID_O)?.classList.contains("active")).toBe(false);
    expect(visitK.containerEl.querySelector(".trie-edge-check")?.textContent).toBe(
      "Проверка ребра: «к» — ребро уже есть — спускаемся",
    );
    // Промежуточный снимок: result null — чипа итога нет.
    expect(visitK.containerEl.querySelectorAll(".trie-result")).toHaveLength(0);

    const visitO = renderToDOM(
      trieFrame({
        ...catTreeSnapshot("search ко"),
        activePath: ["root", ID_K, ID_O],
        edgeCheck: { rune: "о", state: "reused" },
      }),
    );
    expect(visitO.ok).toBe(true);
    expect(trieCell(visitO.containerEl, ID_O)?.classList.contains("active")).toBe(true);
    expect(trieCell(visitO.containerEl, ID_T)?.classList.contains("active")).toBe(false);
    expect(visitO.containerEl.querySelectorAll(".trie-result")).toHaveLength(0);

    const final = renderToDOM(trieFrame({ ...catTreeSnapshot("search ко"), result: "not-found" }));
    expect(final.ok).toBe(true);
    expect(final.containerEl.querySelectorAll(".trie-edge-check")).toHaveLength(0);
    expect(final.containerEl.querySelectorAll(".trie-node.active")).toHaveLength(0);
    expect(final.containerEl.querySelector(".trie-result")?.textContent).toBe(
      "Результат: слова нет — не найдено",
    );
    expect(final.containerEl.querySelector(".trie-result")?.getAttribute("data-result")).toBe(
      "not-found",
    );
  });

  it("delete по существующему пути без конца слова: reused-кадры, result null, итог not-deleted", () => {
    // delete «ко» при сохранённом «кот»: путь есть, удалять нечего.
    const visitO = renderToDOM(
      trieFrame({
        ...catTreeSnapshot("delete ко"),
        activePath: ["root", ID_K, ID_O],
        edgeCheck: { rune: "о", state: "reused" },
      }),
    );
    expect(visitO.ok).toBe(true);
    expect(trieCell(visitO.containerEl, ID_O)?.classList.contains("active")).toBe(true);
    expect(visitO.containerEl.querySelector(".trie-edge-check")?.textContent).toBe(
      "Проверка ребра: «о» — ребро уже есть — спускаемся",
    );
    expect(visitO.containerEl.querySelectorAll(".trie-result")).toHaveLength(0);

    const final = renderToDOM(trieFrame({ ...catTreeSnapshot("delete ко"), result: "not-deleted" }));
    expect(final.ok).toBe(true);
    expect(final.containerEl.querySelectorAll(".trie-edge-check")).toHaveLength(0);
    expect(final.containerEl.querySelector(".trie-result")?.textContent).toBe(
      "Результат: слова нет — не удалено",
    );
    expect(final.containerEl.querySelector(".trie-result")?.getAttribute("data-result")).toBe(
      "not-deleted",
    );
  });

  it("delete промах по отсутствующему ребру: путь до родителя, result null, итог not-deleted", () => {
    // delete «ком» на дереве кот/код: м отсутствует в топологии.
    const visitMissing = renderToDOM(
      trieFrame({
        ...catCodeTreeSnapshot(),
        operation: "delete ком",
        activePath: ["root", ID_K, ID_O],
        edgeCheck: { rune: "м", state: "missing" },
      }),
    );
    expect(visitMissing.ok).toBe(true);
    expect(trieCell(visitMissing.containerEl, ID_O)?.classList.contains("active")).toBe(true);
    expect(trieCell(visitMissing.containerEl, ID_M)).toBeNull();
    expect(visitMissing.containerEl.querySelectorAll(".trie-node.active")).toHaveLength(3);
    expect(visitMissing.containerEl.querySelector(".trie-edge-check")?.textContent).toBe(
      "Проверка ребра: «м» — ребра нет — спуск невозможен",
    );
    expect(visitMissing.containerEl.querySelectorAll(".trie-result")).toHaveLength(0);

    const final = renderToDOM(
      trieFrame({
        ...catCodeTreeSnapshot(),
        operation: "delete ком",
        result: "not-deleted",
      }),
    );
    expect(final.ok).toBe(true);
    expect(final.containerEl.querySelectorAll(".trie-edge-check")).toHaveLength(0);
    expect(final.containerEl.querySelector(".trie-result")?.textContent).toBe(
      "Результат: слова нет — не удалено",
    );
    expect(final.containerEl.querySelector(".trie-result")?.getAttribute("data-result")).toBe(
      "not-deleted",
    );
  });

  it("точный поиск найденного: активный путь на visit-кадрах, result null, итог found", () => {
    // search «кот» по сохранённому «кот»: три reused-кадра, затем found.
    const visitT = renderToDOM(
      trieFrame({
        ...catTreeSnapshot("search кот"),
        activePath: ["root", ID_K, ID_O, ID_T],
        edgeCheck: { rune: "т", state: "reused" },
      }),
    );
    expect(visitT.ok).toBe(true);
    expect(visitT.containerEl.querySelectorAll(".trie-node.active")).toHaveLength(4);
    expect(trieCell(visitT.containerEl, ID_T)?.classList.contains("active")).toBe(true);
    expect(visitT.containerEl.querySelector(".trie-edge-check")?.textContent).toBe(
      "Проверка ребра: «т» — ребро уже есть — спускаемся",
    );
    expect(visitT.containerEl.querySelectorAll(".trie-result")).toHaveLength(0);

    const final = renderToDOM(trieFrame({ ...catTreeSnapshot("search кот"), result: "found" }));
    expect(final.ok).toBe(true);
    expect(final.containerEl.querySelectorAll(".trie-edge-check")).toHaveLength(0);
    expect(final.containerEl.querySelector(".trie-result")?.textContent).toBe(
      "Результат: слово найдено",
    );
    expect(final.containerEl.querySelector(".trie-result")?.getAttribute("data-result")).toBe(
      "found",
    );
  });
});
