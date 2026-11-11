// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): парсинг сценария BST
// и построение Run URL — ошибки ввода ловятся до Run. Зеркало серверной
// валидации (visual/server/bst.go): грамматика строки, безопасный диапазон
// целых ±(2^53−1), лимит 500 команд и лимит 100 живых узлов.
import { describe, expect, it } from "vitest";
import { buildBSTRunURL } from "../src/sse";
import { MAX_TREE_NODES } from "../src/tree";
import { MAX_BST_OPS, parseBSTScenario } from "../src/scenario";

describe("parseBSTScenario — удачный разбор", () => {
  it("разбирает сценарий с insert, search и delete", () => {
    expect(parseBSTScenario("insert 5\ninsert 3\nsearch 7\ndelete 3")).toEqual([
      { kind: "insert", key: 5 },
      { kind: "insert", key: 3 },
      { kind: "search", key: 7 },
      { kind: "delete", key: 3 },
    ]);
  });

  it("пропускает пустые строки и триммирует пробелы вокруг команды", () => {
    expect(parseBSTScenario("\n\ninsert 5\n \ndelete 5\n\n")).toEqual([
      { kind: "insert", key: 5 },
      { kind: "delete", key: 5 },
    ]);
    expect(parseBSTScenario("  search   -4  ")).toEqual([
      { kind: "search", key: -4 },
    ]);
  });

  it("принимает отрицательные ключи и границы безопасного диапазона", () => {
    expect(parseBSTScenario("insert -9007199254740991")).toEqual([
      { kind: "insert", key: -Number.MAX_SAFE_INTEGER },
    ]);
    expect(parseBSTScenario("search 9007199254740991")).toEqual([
      { kind: "search", key: Number.MAX_SAFE_INTEGER },
    ]);
  });

  it("дубликат вставки и промахи search/delete — легальные шаги, не ошибки", () => {
    expect(() =>
      parseBSTScenario("insert 5\ninsert 5\nsearch 9\ndelete 9"),
    ).not.toThrow();
  });
});

describe("parseBSTScenario — ошибки ввода до Run", () => {
  it("отвергает неизвестную команду с номером строки", () => {
    expect(() => parseBSTScenario("insert 5\nupsert 7")).toThrow(
      /строка 2.*неизвестная команда «upsert»/,
    );
  });

  it("отвергает команду без числа", () => {
    expect(() => parseBSTScenario("insert")).toThrow(
      /команда «insert»: не хватает целого числа/,
    );
    expect(() => parseBSTScenario("delete 5\nsearch")).toThrow(
      /строка 2: команда «search»: не хватает целого числа/,
    );
  });

  it("отвергает лишние аргументы", () => {
    expect(() => parseBSTScenario("insert 5 6")).toThrow(/лишние аргументы/);
    expect(() => parseBSTScenario("search 1 2")).toThrow(/лишние аргументы/);
  });

  it("отвергает дробное, экспоненту и мусор вместо целого", () => {
    expect(() => parseBSTScenario("insert 1.5")).toThrow(
      /не удалось разобрать «1\.5»/,
    );
    expect(() => parseBSTScenario("insert 1e3")).toThrow(
      /не удалось разобрать «1e3»/,
    );
    expect(() => parseBSTScenario("delete семь")).toThrow(
      /не удалось разобрать «семь»/,
    );
  });

  it("отвергает значения вне безопасного диапазона ±(2^53−1)", () => {
    expect(() => parseBSTScenario("insert 9007199254740992")).toThrow(
      /вне безопасного диапазона/,
    );
    expect(() => parseBSTScenario("search -9007199254740993")).toThrow(
      /вне безопасного диапазона/,
    );
    // Целое за int64 тоже отвергается — на клиенте той же границей
    // Number.MAX_SAFE_INTEGER (осознанный отказ от int64-паритета,
    // см. decimalInteger.ts).
    expect(() => parseBSTScenario("insert 9223372036854775808")).toThrow(
      /вне безопасного диапазона/,
    );
  });

  it("отвергает пустой сценарий (только пустые строки)", () => {
    expect(() => parseBSTScenario("")).toThrow(/сценарий пуст/);
    expect(() => parseBSTScenario("\n \n\t\n")).toThrow(/сценарий пуст/);
  });
});

describe("parseBSTScenario — лимит команд", () => {
  const script = (n: number) =>
    Array.from({ length: n }, (_, i) => `search ${i}`).join("\n");

  it("ровно 500 команд принимается", () => {
    expect(parseBSTScenario(script(MAX_BST_OPS))).toHaveLength(MAX_BST_OPS);
  });

  it("501-я команда отвергается с номером первой лишней строки", () => {
    expect(() => parseBSTScenario(script(MAX_BST_OPS + 1))).toThrow(
      new RegExp(
        `слишком длинный.*не более ${MAX_BST_OPS} команд.*на строке ${MAX_BST_OPS + 1}`,
      ),
    );
  });

  it("лимит считается по непустым командам: пустые строки не занимают места", () => {
    const scriptWithBlanks = "\n" + script(MAX_BST_OPS) + "\n\n";
    expect(() => parseBSTScenario(scriptWithBlanks)).not.toThrow();
  });
});

describe("parseBSTScenario — лимит живых узлов", () => {
  const inserts = (n: number, start = 0) =>
    Array.from({ length: n }, (_, i) => `insert ${start + i}`).join("\n");

  it("ровно 100 живых узлов принимается", () => {
    expect(() => parseBSTScenario(inserts(MAX_TREE_NODES))).not.toThrow();
  });

  it("101-й уникальный узел отвергается до Run", () => {
    expect(() => parseBSTScenario(inserts(MAX_TREE_NODES + 1))).toThrow(
      new RegExp(
        `лимит живых узлов.*не более ${MAX_TREE_NODES}.*команда №${MAX_TREE_NODES + 1} \\(insert ${MAX_TREE_NODES}\\)`,
      ),
    );
  });

  it("повторная вставка дубликата не растит дерево", () => {
    const script = Array.from(
      { length: MAX_TREE_NODES + 1 },
      () => "insert 5",
    ).join("\n");
    expect(() => parseBSTScenario(script)).not.toThrow();
  });

  it("удаление освобождает место под новые вставки", () => {
    const script = [
      inserts(MAX_TREE_NODES),
      "delete 0",
      "insert 1000",
    ].join("\n");
    expect(() => parseBSTScenario(script)).not.toThrow();
    // А без удаления 101-й узел не влезает.
    expect(() =>
      parseBSTScenario([inserts(MAX_TREE_NODES), "insert 1000"].join("\n")),
    ).toThrow(/лимит живых узлов/);
  });

  it("промахи search/delete не занимают место", () => {
    const script = [
      inserts(MAX_TREE_NODES),
      "search -1\ndelete -1\nsearch -2\ndelete -2",
      "delete 0",
      "insert 1000",
    ].join("\n");
    expect(() => parseBSTScenario(script)).not.toThrow();
  });
});

describe("buildBSTRunURL", () => {
  it("строит URL маршрута /api/run/bst с параметром script", () => {
    // URLSearchParams кодирует пробел плюсом, перенос строки — %0A.
    expect(buildBSTRunURL("insert 5")).toBe(
      "/api/run/bst?script=insert+5",
    );
    // Сценарий едет как есть: переносы строк сохраняются кодированием.
    expect(buildBSTRunURL("insert 5\nsearch 7")).toBe(
      "/api/run/bst?script=insert+5%0Asearch+7",
    );
  });
});
