// Тест шва фронта: разбор входа обхода дерева и порядка — валидный,
// невалидный ввод и лимит узлов (зеркало visual/server/tree.go).
import { describe, expect, it } from "vitest";
import {
  buildTreeTraversalRunURL,
} from "../src/sse";
import {
  MAX_TREE_NODES,
  isTreeOrder,
  parseTreeInput,
  parseTreeOrder,
} from "../src/tree";

describe("parseTreeInput", () => {
  it("принимает корректное асимметричное дерево с повторяющимися значениями", () => {
    expect(() =>
      parseTreeInput(
        `{"value":1,"left":{"value":2,"left":{"value":4}},"right":{"value":2,"right":{"value":5}}}`,
      ),
    ).not.toThrow();
  });

  it("принимает null — пустое дерево", () => {
    expect(() => parseTreeInput("null")).not.toThrow();
  });

  it("принимает потомков без ключей left/right", () => {
    expect(() => parseTreeInput(`{"value":7}`)).not.toThrow();
  });

  it("принимает явные null-потомки", () => {
    expect(() =>
      parseTreeInput(`{"value":7,"left":null,"right":null}`),
    ).not.toThrow();
  });

  it("пустой ввод — ошибка", () => {
    expect(() => parseTreeInput("   ")).toThrow(/введите дерево/);
  });

  it("мусор вместо JSON — ошибка", () => {
    expect(() => parseTreeInput("{value: 1}")).toThrow(/JSON/);
  });

  it("массив вместо узла — ошибка", () => {
    expect(() => parseTreeInput("[1,2]")).toThrow(/должен быть объектом/);
  });

  it("строка вместо узла — ошибка", () => {
    expect(() => parseTreeInput(`"1"`)).toThrow(/должен быть объектом/);
  });

  it("значение не целое — ошибка", () => {
    expect(() => parseTreeInput(`{"value":1.5}`)).toThrow(/не целое/);
  });

  it("значение не число — ошибка", () => {
    expect(() => parseTreeInput(`{"value":"1"}`)).toThrow(/не целое/);
  });

  it("значение за пределами безопасного диапазона — ошибка", () => {
    expect(() =>
      parseTreeInput(`{"value":${Number.MAX_SAFE_INTEGER + 1}}`),
    ).toThrow(/точного диапазона/);
  });

  it("ввод с null у потомка — ошибка", () => {
    expect(() => parseTreeInput(`{"value":1,"left":"2"}`)).toThrow(
      /должен быть объектом/,
    );
  });

  it("дерево из 100 узлов проходит", () => {
    // Цепочка из 100 узлов влево, корень — узел №100.
    let body = `{"value":${MAX_TREE_NODES}}`;
    for (let v = MAX_TREE_NODES - 1; v >= 1; v--) {
      body = `{"value":${v},"left":${body}}`;
    }
    expect(() => parseTreeInput(body)).not.toThrow();
  });

  it("дерево из 101 узла — ошибка лимита", () => {
    let body = `{"value":${MAX_TREE_NODES + 1}}`;
    for (let v = MAX_TREE_NODES; v >= 1; v--) {
      body = `{"value":${v},"left":${body}}`;
    }
    expect(() => parseTreeInput(body)).toThrow(/разрешено не более 100 узлов/);
  });
});

describe("parseTreeOrder", () => {
  it.each(["in-order", "pre-order", "post-order"] as const)(
    "принимает порядок %s",
    (order) => {
      expect(parseTreeOrder(order)).toBe(order);
    },
  );

  it("пустой порядок — ошибка", () => {
    expect(() => parseTreeOrder("")).toThrow(/выберите порядок/);
  });

  it("неизвестный порядок — ошибка", () => {
    expect(() => parseTreeOrder("breadth-first")).toThrow(
      /неизвестный порядок/,
    );
  });

  it("isTreeOrder различает валидный порядок", () => {
    expect(isTreeOrder("in-order")).toBe(true);
    expect(isTreeOrder("breadth-first")).toBe(false);
  });
});

describe("buildTreeTraversalRunURL", () => {
  it("строит URL с параметрами tree и order", () => {
    expect(
      buildTreeTraversalRunURL(`{"value":1}`, "pre-order"),
    ).toBe(`/api/run/tree-traversal?tree=${encodeURIComponent(`{"value":1}`)}&order=pre-order`);
  });
});
