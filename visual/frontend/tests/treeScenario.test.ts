// Тест шва фронта: разбор входа обхода дерева и порядка — валидный,
// невалидный ввод и лимит узлов (зеркало visual/server/tree.go).
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
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

describe("parseTreeInput: точные value-литералы (париитет с сервером)", () => {
  // Компенсированная экспонента: «0.» + 4088 нулей + «1e4097» — ровно
  // 100000000 (1e8), литерал ровно 4096 символов; JSON.parse даёт точное
  // 100000000, а сервер принимает тот же вход через SSE.
  const compensated = `{"value":0.${"0".repeat(4088)}1e4097}`;

  it("принимает компенсированную экспоненту (ровно 4096 символов)", () => {
    expect(() => parseTreeInput(compensated)).not.toThrow();
  });

  it("принимает обычные целые формы: 1.0, 1e2, 1.0e2, -0", () => {
    for (const tree of [
      `{"value":1.0}`,
      `{"value":1e2}`,
      `{"value":1.0e2}`,
      `{"value":-0}`,
      `{"value": 1 , "left" : {"value":0.50e3} }`,
    ]) {
      expect(() => parseTreeInput(tree)).not.toThrow();
    }
  });

  it("принимает безопасную целую экспоненту вне int64 — клиентская политика точности", () => {
    // 1e30 — целое, но за Number.MAX_SAFE_INTEGER: клиент отвергает сам.
    expect(() => parseTreeInput(`{"value":1e30}`)).toThrow(/точного диапазона/);
    expect(() => parseTreeInput(`{"value":1e12}`)).not.toThrow();
  });

  it("отвергает точную нецелую дробь, которая округляется до целого", () => {
    const lit = "1." + "0".repeat(48) + "1";
    expect(() => parseTreeInput(`{"value":${lit}}`)).toThrow(/не целое/);
    // и на глубине дерева — литерал не проходит ни в каком узле
    expect(() =>
      parseTreeInput(`{"value":1,"right":{"value":${lit}}}`),
    ).toThrow(/не целое/);
  });

  it("отвергает литерал длиннее 4096 символов — как сервер", () => {
    const tooLong = "1" + "0".repeat(4097);
    expect(() => parseTreeInput(`{"value":${tooLong}}`)).toThrow(
      /слишком длинное/,
    );
  });

  it("отвергает дробный хвост после показателя — не целое", () => {
    expect(() => parseTreeInput(`{"value":1e-4000}`)).toThrow(/не целое/);
  });

  it("игнорирует посторонние поля и вложенные структуры — как сервер", () => {
    const lit = "1." + "0".repeat(48) + "1";
    expect(() =>
      parseTreeInput(
        `{"meta":{"value":${lit}},"xs":[{"value":${lit}}],"s":"${lit}","value":1}`,
      ),
    ).not.toThrow();
  });

  it("читает экранированный ключ value так же, как JSON.parse", () => {
    expect(() =>
      parseTreeInput(`{"\\u0076alue":${"1." + "0".repeat(48) + "1"}}`),
    ).toThrow(/не целое/);
  });

  it("при повторении ключа value проверяет последнюю запись", () => {
    expect(() =>
      parseTreeInput(`{"value":${"1." + "0".repeat(48) + "1"},"value":2}`),
    ).not.toThrow();
    expect(() =>
      parseTreeInput(`{"value":2,"value":${"1." + "0".repeat(48) + "1"}}`),
    ).toThrow(/не целое/);
  });
});

// Общие кейсы паритета клиент/сервер: один и тот же fixture-файл
// (visual/testdata/tree-input-parity.json) прогоняется и здесь, и в
// visual/server/tree_test.go — оба публичных шва проверяют одни и те же
// representative-входы (числовые литералы и повторные ключи).
type ParityCase = { name: string; tree: string } & (
  | { accept: true }
  // Отвергающие кейсы обязаны нести непустое ожидание ошибки клиента:
  // пропуск поля ловится уже на уровне типов.
  | { accept: false; clientError: string }
);

type ParityFixture = {
  comment: string;
  cases: ParityCase[];
};

const parityCases = (JSON.parse(
  readFileSync(
    join(dirname(fileURLToPath(import.meta.url)), "../../testdata/tree-input-parity.json"),
    "utf8",
  ),
) as ParityFixture).cases;

// Рантайм-охрана контракта fixture: набор не пуст, а у отвергающих кейсов
// ожидание ошибки непусто (типовая проверка дополняется проверкой значения).
if (parityCases.length === 0) {
  throw new Error(
    "общий fixture пуст — ожидаются общие кейсы паритета клиент/сервер",
  );
}
for (const testCase of parityCases) {
  if (!testCase.accept && testCase.clientError.trim() === "") {
    throw new Error(
      `кейс «${testCase.name}»: пустое ожидание ошибки клиента у отвергающего кейса`,
    );
  }
}

describe("parseTreeInput: паритет с сервером (общий fixture)", () => {
  it.each(parityCases)("%s", (testCase) => {
    if (testCase.accept) {
      expect(() => parseTreeInput(testCase.tree)).not.toThrow();
    } else {
      expect(() => parseTreeInput(testCase.tree)).toThrow(
        testCase.clientError,
      );
    }
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
