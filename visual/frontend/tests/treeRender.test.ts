// Тест шва фронта: рендер кадра kind="tree" — полное дерево с активным
// и посещёнными узлами на несимметричном дереве с повторяющимися
// значениями. DOM ниже публичного renderer-интерфейса не проверяем.
import { describe, expect, it } from "vitest";
import { isTreeFrame, renderFrame } from "../src/render";

function treeFrame(data: unknown) {
  return { step: 2, kind: "tree", explanation: "пояснение шага", data } as never;
}

function renderToDOM(frame: never): {
  ok: boolean;
  explanationEl: HTMLElement;
  containerEl: HTMLElement;
} {
  const explanationEl = document.createElement("div");
  const containerEl = document.createElement("div");
  const ok = renderFrame(frame, explanationEl, containerEl);
  return { ok, explanationEl, containerEl };
}

// Несимметричное дерево с повторяющимся значением (шаги 1–3 из 5):
//
//	        1
//	      /   \
//	     2     2
//	    /       \
//	   4         5
const nodes = [
  { id: "root", value: 1, leftId: "L", rightId: "R" },
  { id: "L", value: 2, leftId: "LL", rightId: "" },
  { id: "R", value: 2, leftId: "", rightId: "RR" },
  { id: "LL", value: 4, leftId: "", rightId: "" },
  { id: "RR", value: 5, leftId: "", rightId: "" },
];
const snapshot = {
  order: "in-order",
  nodes,
  rootId: "root",
  activeId: "L",
  visited: [
    { id: "LL", position: 1 },
    { id: "L", position: 2 },
  ],
  operation: "посещение L",
};

const startSnapshot = {
  order: "in-order",
  nodes,
  rootId: "root",
  activeId: "",
  visited: [],
  operation: "",
};

const emptySnapshot = {
  order: "pre-order",
  nodes: [],
  rootId: "",
  activeId: "",
  visited: [],
  operation: "",
};

describe("isTreeFrame", () => {
  it("принимает корректный снимок дерева", () => {
    expect(isTreeFrame(treeFrame(snapshot))).toBe(true);
  });

  it("принимает пустое дерево", () => {
    expect(isTreeFrame(treeFrame(emptySnapshot))).toBe(true);
  });

  it("отвергает другой kind", () => {
    expect(
      isTreeFrame({
        step: 1,
        kind: "structure",
        explanation: "пояснение",
        data: {},
      } as never),
    ).toBe(false);
  });

  it("отвергает снимок без поля nodes", () => {
    expect(isTreeFrame(treeFrame({ order: "in-order" }))).toBe(false);
  });

  it("отвергает узел с нецелым значением", () => {
    expect(
      isTreeFrame(
        treeFrame({
          ...snapshot,
          nodes: [{ id: "root", value: 1.5, leftId: "", rightId: "" }],
          rootId: "root",
        }),
      ),
    ).toBe(false);
  });

  it("отвергает активный узел, которого нет в дереве", () => {
    expect(isTreeFrame(treeFrame({ ...snapshot, activeId: "XX" }))).toBe(false);
  });

  it("отвергает посещение узла вне дерева", () => {
    expect(
      isTreeFrame(
        treeFrame({ ...snapshot, visited: [{ id: "XX", position: 1 }] }),
      ),
    ).toBe(false);
  });

  it("отвергает посещение с нулевой позицией", () => {
    expect(
      isTreeFrame(
        treeFrame({ ...snapshot, visited: [{ id: "L", position: 0 }] }),
      ),
    ).toBe(false);
  });
});

describe("renderFrame kind=tree", () => {
  it("рисует все узлы с идентификаторами путей, независимо от значения", () => {
    const { ok, explanationEl, containerEl } = renderToDOM(treeFrame(snapshot));
    expect(ok).toBe(true);
    expect(explanationEl.textContent).toBe("пояснение шага");

    const classNames = (id: string): string[] | undefined => {
      const el = containerEl.querySelector<HTMLDivElement>(
        `.tree-node[data-id="${id}"]`,
      );
      if (el === null) return undefined;
      return Array.from(el.classList);
    };

    // У повторяющихся значений 2 два разных узла — L и R.
    expect(classNames("L")).toContain("visited");
    expect(classNames("R")).not.toContain("visited");
    expect(classNames("LL")).toContain("visited");
    expect(classNames("LL")).not.toContain("active");
    expect(classNames("L")).toContain("active");
    expect(containerEl.querySelectorAll(".tree-node:not(.tree-empty)").length).toBe(5);
  });

  it("ставит номер позиции посещённому узлу", () => {
    const { containerEl } = renderToDOM(treeFrame(snapshot));
    const active = containerEl.querySelector<HTMLDivElement>(
      '.tree-node[data-id="L"]',
    );
    expect(active?.dataset.visitedPosition).toBe("2");
    const first = containerEl.querySelector<HTMLDivElement>(
      '.tree-node[data-id="LL"]',
    );
    expect(first?.dataset.visitedPosition).toBe("1");
  });

  it("не помечает активный узел на стартовом кадре", () => {
    const { containerEl } = renderToDOM(treeFrame(startSnapshot));
    expect(
      Array.from(containerEl.querySelectorAll(".tree-node.active")),
    ).toHaveLength(0);
    expect(
      Array.from(containerEl.querySelectorAll(".tree-node.visited")),
    ).toHaveLength(0);
  });

  it("рисует пустые ветви как маркеры ∅", () => {
    const { containerEl } = renderToDOM(treeFrame(snapshot));
    // Пустых ветвей 6: право L, лево R, обе дочерние у LL и у RR.
    expect(containerEl.querySelectorAll(".tree-node.tree-empty").length).toBe(6);
  });

  it("пустое дерево рисуется пустым маркером без узлов", () => {
    const { containerEl } = renderToDOM(treeFrame(emptySnapshot));
    expect(containerEl.querySelectorAll(".tree-node").length).toBe(1);
    expect(
      containerEl.querySelector(".tree-node")?.classList.contains("tree-empty"),
    ).toBe(true);
  });

  it("не рисует кадр другого kind", () => {
    const { ok, containerEl } = renderToDOM({
      step: 1,
      kind: "unknown-kind",
      explanation: "пояснение",
      data: {},
    } as never);
    expect(ok).toBe(false);
    expect(containerEl.children).toHaveLength(0);
  });
});
