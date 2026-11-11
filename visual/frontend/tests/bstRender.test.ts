// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): рендер кадров
// kind="tree" сценария BST — активный путь сравнения, результат шага
// и обновлённая топология на снимках, зеркалящих серверные кадры
// (visual/render/bst.go): order="bst", пустые visited/queue,
// идентификаторы — пути от корня в пределах кадра. DOM ниже публичного
// renderer-интерфейса не проверяем.
import { describe, expect, it } from "vitest";
import { isTreeFrame, renderFrame } from "../src/render";
import type { Frame } from "../src/types";

function bstFrame(data: unknown, explanation = "пояснение шага"): Frame {
  return { step: 1, kind: "tree", explanation, data } as Frame;
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

/** Ячейка узла дерева по идентификатору пути (id кадра, не значение). */
function nodeCell(containerEl: HTMLElement, id: string): HTMLDivElement | null {
  return containerEl.querySelector<HTMLDivElement>(
    `.tree-node[data-id="${id}"]`,
  );
}

const bstOrder = "bst";

// Исходный кадр Run: пустое дерево (зеркало стартового кадра BSTRun).
const startSnapshot = {
  order: bstOrder,
  nodes: [],
  rootId: "",
  activeId: "",
  visited: [],
  queue: [],
  operation: "",
};

// Снимок вставки первого узла: дерево было пустым, узел 5 стал корнем
// (зеркало кадра BSTInserted с пустым путём).
const insertedRootSnapshot = {
  order: bstOrder,
  nodes: [{ id: "root", value: 5, leftId: "", rightId: "" }],
  rootId: "root",
  activeId: "root",
  visited: [],
  queue: [],
  operation: "вставка «root»",
};

// Снимок дерева 5(3, 7) после вставок 5, 3, 7: активный узел — новый
// узел 7 на пути «R» (зеркало кадра BSTInserted).
const insertedTreeSnapshot = {
  order: bstOrder,
  nodes: [
    { id: "root", value: 5, leftId: "L", rightId: "R" },
    { id: "L", value: 3, leftId: "", rightId: "" },
    { id: "R", value: 7, leftId: "", rightId: "" },
  ],
  rootId: "root",
  activeId: "R",
  visited: [],
  queue: [],
  operation: "вставка «R»",
};

// Успешный поиск 7 в дереве 5(3, 7): кадр находки на пути «R».
const foundSnapshot = {
  ...insertedTreeSnapshot,
  activeId: "R",
  operation: "найдено «R»",
};

// Промах поиска 9: спуск дошёл до пустого места пути «RR» — активного
// узла нет (зеркало кадра BSTNotFound).
const missSnapshot = {
  order: bstOrder,
  nodes: [
    { id: "root", value: 5, leftId: "L", rightId: "R" },
    { id: "L", value: 3, leftId: "", rightId: "" },
    { id: "R", value: 7, leftId: "", rightId: "" },
  ],
  rootId: "root",
  activeId: "",
  visited: [],
  queue: [],
  operation: "промах",
};

// Кадр выбора преемника при удалении 5 из дерева 5(3, 8(7, 9)):
// значение 7 преемника (путь «RL») уже перенесено в корень, но узел-
// преемник ещё жив (зеркало кадра BSTSuccessor после переноса).
const successorSnapshot = {
  order: bstOrder,
  nodes: [
    { id: "root", value: 7, leftId: "L", rightId: "R" },
    { id: "L", value: 3, leftId: "", rightId: "" },
    { id: "R", value: 8, leftId: "RL", rightId: "RR" },
    { id: "RL", value: 7, leftId: "", rightId: "" },
    { id: "RR", value: 9, leftId: "", rightId: "" },
  ],
  rootId: "root",
  activeId: "RL",
  visited: [],
  queue: [],
  operation: "преемник «RL»",
};

// Финальный кадр удаления: преемник вырезан, дерево 7(3, 8(_, 9)) —
// новая топология и новый корень (зеркало кадра BSTRemoved).
const removedSnapshot = {
  order: bstOrder,
  nodes: [
    { id: "root", value: 7, leftId: "L", rightId: "R" },
    { id: "L", value: 3, leftId: "", rightId: "" },
    { id: "R", value: 8, leftId: "", rightId: "RR" },
    { id: "RR", value: 9, leftId: "", rightId: "" },
  ],
  rootId: "root",
  activeId: "",
  visited: [],
  queue: [],
  operation: "вырезание преемника «RL»",
};

describe("isTreeFrame: кадры сценария BST", () => {
  it("принимает BST-снимок с маркером order=bst и пустыми visited/queue", () => {
    expect(isTreeFrame(bstFrame(foundSnapshot))).toBe(true);
    expect(isTreeFrame(bstFrame(startSnapshot))).toBe(true);
  });

  it("принимает кадр промаха без активного узла", () => {
    expect(isTreeFrame(bstFrame(missSnapshot))).toBe(true);
  });

  it("отвергает BST-снимок с активным узлом вне дерева", () => {
    expect(isTreeFrame(bstFrame({ ...foundSnapshot, activeId: "XX" }))).toBe(
      false,
    );
  });

  it("отвергает BST-снимок со ссылкой на отсутствующего потомка", () => {
    // Ссылка на вырезанного преемника «RL» остаться в снимке не может:
    // топология после структурного изменения согласована.
    expect(
      isTreeFrame(
        bstFrame({
          ...removedSnapshot,
          nodes: removedSnapshot.nodes.map((n) =>
            n.id === "R" ? { ...n, leftId: "RL" } : n,
          ),
        }),
      ),
    ).toBe(false);
  });
});

describe("renderFrame: кадры вставки BST", () => {
  it("рисует пустое дерево маркером на стартовом кадре", () => {
    const { ok, containerEl } = renderToDOM(bstFrame(startSnapshot));
    expect(ok).toBe(true);
    expect(containerEl.querySelectorAll(".tree-node").length).toBe(1);
    expect(
      containerEl.querySelector(".tree-node")?.classList.contains("tree-empty"),
    ).toBe(true);
  });

  it("выделяет активный узел вставленного корня", () => {
    const { ok, explanationEl, containerEl } = renderToDOM(
      bstFrame(insertedRootSnapshot, "Вставка ключа 5: дерево пустое — новый узел 5 становится корнем."),
    );
    expect(ok).toBe(true);
    expect(explanationEl.textContent).toBe(
      "Вставка ключа 5: дерево пустое — новый узел 5 становится корнем.",
    );
    const root = nodeCell(containerEl, "root");
    expect(root?.textContent).toBe("5");
    expect(root?.classList.contains("active")).toBe(true);
  });

  it("рисует обновлённую топологию после вставки в непустое дерево", () => {
    const { containerEl } = renderToDOM(bstFrame(insertedTreeSnapshot));
    // Все три узла нарисованы, активный — новый узел 7 на пути «R».
    expect(nodeCell(containerEl, "root")?.textContent).toBe("5");
    expect(nodeCell(containerEl, "L")?.textContent).toBe("3");
    const inserted = nodeCell(containerEl, "R");
    expect(inserted?.textContent).toBe("7");
    expect(inserted?.classList.contains("active")).toBe(true);
    expect(nodeCell(containerEl, "L")?.classList.contains("active")).toBe(false);
  });

  it("не рисует подсветку посещённых и очередь — у BST они всегда пустые", () => {
    const { containerEl } = renderToDOM(bstFrame(insertedTreeSnapshot));
    expect(containerEl.querySelectorAll(".tree-node.visited")).toHaveLength(0);
    expect(containerEl.querySelectorAll(".tree-queue-item")).toHaveLength(0);
  });
});

describe("renderFrame: кадры поиска BST", () => {
  it("выделяет найденный узел на пути сравнения", () => {
    const { ok, containerEl } = renderToDOM(
      bstFrame(foundSnapshot, "Поиск ключа 7: ключ 7 найден (путь «R»)."),
    );
    expect(ok).toBe(true);
    const found = nodeCell(containerEl, "R");
    expect(found?.textContent).toBe("7");
    expect(found?.classList.contains("active")).toBe(true);
    // Соседний узел 3 на пути «L» не подсвечен: путь сравнения — «R».
    expect(nodeCell(containerEl, "L")?.classList.contains("active")).toBe(false);
  });

  it("рисует промах без подсветки активного узла", () => {
    const { ok, explanationEl, containerEl } = renderToDOM(
      bstFrame(missSnapshot, "Поиск ключа 9: спуск дошёл до пустого места (путь «RR») — ключа 9 в дереве нет. Промах — штатный исход поиска."),
    );
    expect(ok).toBe(true);
    expect(explanationEl.textContent).toContain("штатный исход");
    expect(containerEl.querySelectorAll(".tree-node.active")).toHaveLength(0);
    // Дерево при промахе не меняется: все узлы живы.
    expect(nodeCell(containerEl, "root")?.textContent).toBe("5");
    expect(nodeCell(containerEl, "R")?.textContent).toBe("7");
  });
});

describe("renderFrame: кадры удаления BST с двумя потомками", () => {
  it("рисует выбор преемника: перенесённое значение у корня и живой узел-преемник", () => {
    const { containerEl } = renderToDOM(bstFrame(successorSnapshot));
    // Активный узел — преемник на пути «RL», ещё жив в снимке.
    const successor = nodeCell(containerEl, "RL");
    expect(successor?.textContent).toBe("7");
    expect(successor?.classList.contains("active")).toBe(true);
    // Значение 7 уже перенесено в корень — одинаковые значения 7 у двух
    // разных узлов различаются идентификаторами путей.
    expect(nodeCell(containerEl, "root")?.textContent).toBe("7");
    expect(containerEl.querySelectorAll(".tree-node:not(.tree-empty)").length).toBe(5);
  });

  it("рисует новую топологию после вырезания преемника", () => {
    const { containerEl } = renderToDOM(bstFrame(removedSnapshot));
    // Корень теперь держит 7, преемника «RL» в дереве больше нет.
    expect(nodeCell(containerEl, "root")?.textContent).toBe("7");
    expect(nodeCell(containerEl, "RL")).toBeNull();
    // Правый потомок 8 потерял левую ветвь, правая ветвь 9 жива.
    expect(nodeCell(containerEl, "R")?.textContent).toBe("8");
    expect(nodeCell(containerEl, "RR")?.textContent).toBe("9");
    expect(containerEl.querySelectorAll(".tree-node:not(.tree-empty)").length).toBe(4);
  });

  it("перерисовывает дерево целиком: идентификаторы не обязаны переживать кадр", () => {
    // Кадр до удаления и кадр после: один и тот же идентификатор «root»
    // в разных кадрах — разные узлы; рендер каждый раз строит дерево из
    // снимка, не опираясь на предыдущий кадр.
    const before = renderToDOM(bstFrame(insertedRootSnapshot));
    expect(nodeCell(before.containerEl, "root")?.textContent).toBe("5");
    expect(nodeCell(before.containerEl, "root")?.classList.contains("active")).toBe(true);

    const after = renderToDOM(bstFrame(removedSnapshot));
    // Тот же контейнер — кадр after рисуется заново: без узла «root»
    // со значением 5 из предыдущего кадра.
    expect(nodeCell(after.containerEl, "root")?.textContent).toBe("7");
    expect(nodeCell(after.containerEl, "root")?.classList.contains("active")).toBe(false);
    // Последовательность кадров в одном контейнере: предыдущая топология
    // полностью заменена.
    const sequential = renderToDOM(bstFrame(insertedRootSnapshot));
    const replaced = renderFrame(bstFrame(removedSnapshot), sequential.explanationEl, sequential.containerEl);
    expect(replaced).toBe(true);
    expect(nodeCell(sequential.containerEl, "root")?.textContent).toBe("7");
    expect(sequential.containerEl.querySelectorAll(".tree-node:not(.tree-empty)").length).toBe(4);
  });
});
