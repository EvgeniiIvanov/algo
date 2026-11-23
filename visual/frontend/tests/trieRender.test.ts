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

function trieFrame(data: TrieSnapshot, explanation = "пояснение шага"): Frame {
  return { step: 1, kind: "trie", explanation, data } as Frame;
}

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
    expect(
      isTrieFrame(trieFrame({ ...startSnapshot(), edgeCheck: { rune: "к", state: "nope" } })),
    ).toBe(false);
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

  it.each(results)("рисует чип результата %s только на result-кадре", (result) => {
    const { ok, containerEl } = renderToDOM(
      trieFrame({ ...startSnapshot("search кот"), result }),
    );
    expect(ok).toBe(true);
    const chip = containerEl.querySelector(".trie-result");
    expect(chip?.textContent).toBe(`Результат: ${result}`);
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
      "Результат: not-found",
    );
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
