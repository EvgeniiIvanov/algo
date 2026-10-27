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
  // Отвергающие кейсы обязаны нести непустое ожидание ошибки клиента;
  // контракт проверяется в рантайме (см. validateParityFixture), тип
  // лишь фиксирует, что у отвергающего кейса поле есть.
  | { accept: false; clientError: string }
) & {
  // Явный маркер назначенной проверки эффективного узла: требует
  // корректный expectedNode — это проверяет рантайм-охрана
  // validateParityFixture. Сам снимок SSE проверяет серверный
  // консьюмер fixture. Оба консьюмера применяют один coherent-контракт
  // (см. describe ниже): маркер, если присутствует, обязан быть
  // булевым; маркер true требует принимающего кейса с корректным
  // expectedNode; expectedNode без assertEffectiveNode: true — ошибка
  // fixture; отвергающий кейс не несёт ни маркера, ни ожидания.
  assertEffectiveNode?: boolean;
  // Тестовое представление value — строка со знаковым десятичным
  // целым (например "3"): это метаданные fixture, а не вход дерева.
  // Строка исключает лексическое расхождение Go json.Number.Int64() и
  // TS Number.isInteger() (1e3, 1.0 и т.п.); общая грамматика
  // проверяется ниже, значение затем сравнивается с числом узла
  // SSE-снимка серверным консьюмером fixture.
  expectedNode?: { path: string; value: string };
};

type ParityFixture = {
  comment: string;
  cases: ParityCase[];
};

// fixturePath — общий fixture паритета клиент/сервер.
const fixturePath = join(
  dirname(fileURLToPath(import.meta.url)),
  "../../testdata/tree-input-parity.json",
);

// maxSafeInteger — граница безопасного целого TS (2^53−1): тестовое
// значение expectedNode обязано в неё укладываться, чтобы оба
// консьюмера (TS и Go) сравнивали одно и то же число.
const maxSafeInteger = Number.MAX_SAFE_INTEGER;

// isSignedDecimalInteger — общая грамматика знакового десятичного
// целого для expectedNode.value: необязательный минус, затем 0 или
// ненулевая цифра с любым хвостом десятичных цифр (ведущие нули и
// лексические формы вроде 1e3 или 1.0 не допускаются). Та же
// грамматика проверяется в Go-консьюмере fixture.
function isSignedDecimalInteger(value: string): boolean {
  return /^-?(0|[1-9]\d*)$/.test(value);
}

// validateExpectation — общая проверка корректного expectedNode
// назначенного кейса: непустой после обрезки пробелов путь и значение —
// знаковое десятичное целое в строке в пределах |v| ≤ 2^53−1.
// Отсутствие поля (undefined) и явный null — разные ситуации: null
// отклоняется как некорректная форма ожидания. Возвращает обрезанный
// путь — так же, как это делает Go-консьюмер fixture.
function validateExpectation(byName: string, node: unknown): {
  path: string;
  value: string;
} {
  if (node === null) {
    throw new Error(
      `${byName}: у кейса с assertEffectiveNode expectedNode — null вместо объекта ожидания`,
    );
  }
  if (typeof node !== "object" || Array.isArray(node)) {
    throw new Error(
      `${byName}: у кейса с assertEffectiveNode отсутствует expectedNode`,
    );
  }
  const { path, value } = node as Record<string, unknown>;
  if (typeof path !== "string" || path.trim() === "") {
    throw new Error(
      `${byName}: у кейса с assertEffectiveNode пустой путь в expectedNode`,
    );
  }
  if (
    typeof value !== "string" ||
    !isSignedDecimalInteger(value) ||
    !Number.isInteger(Number(value)) ||
    Math.abs(Number(value)) > maxSafeInteger
  ) {
    throw new Error(
      `${byName}: у кейса с assertEffectiveNode value в expectedNode не знаковое десятичное целое в безопасном диапазоне`,
    );
  }
  return { path: path.trim(), value };
}

// validateParityFixture — рантайм-охрана контракта fixture (JSON.parse
// даёт any, компиляции он не подчиняется): объект, непустой набор кейсов,
// у каждого кейса строковые name/tree и булево accept, у отвергающих —
// непустое clientError. Диагностики — со спецификой кейса.
function validateParityFixture(json: unknown): ParityFixture {
  if (typeof json !== "object" || json === null || Array.isArray(json)) {
    throw new Error(
      "общий fixture: корень не является объектом — ожидаются кейсы паритета",
    );
  }
  const raw = json as { comment?: unknown; cases?: unknown };
  if (!Array.isArray(raw.cases) || raw.cases.length === 0) {
    throw new Error(
      "общий fixture: cases не непустой массив — ожидаются кейсы паритета",
    );
  }
  const cases: ParityCase[] = raw.cases.map((element, index) => {
    if (
      typeof element !== "object" ||
      element === null ||
      Array.isArray(element)
    ) {
      throw new Error(
        `общий fixture: кейс №${index + 1} не объект — ожидаются кейсы паритета`,
      );
    }
    const item = element as Record<string, unknown>;
    if (typeof item.name !== "string" || item.name.trim() === "") {
      throw new Error(
        `общий fixture: кейс №${index + 1}: name не непустая строка`,
      );
    }
    const byName = `общий fixture: кейс «${item.name}»`;
    if (typeof item.tree !== "string" || item.tree.trim() === "") {
      throw new Error(`${byName}: tree не непустая строка`);
    }
    if (item.accept !== true && item.accept !== false) {
      throw new Error(`${byName}: accept не булево (или поле отсутствует)`);
    }
    if (!item.accept) {
      // Coherent-контракт: отвергающий кейс не несёт ни маркера, ни
      // ожидания — у него нет снимка.
      if ("assertEffectiveNode" in item) {
        if (typeof item.assertEffectiveNode !== "boolean") {
          throw new Error(
            `${byName}: маркер assertEffectiveNode не булево`,
          );
        }
        throw new Error(
          `${byName}: отвергающий кейс не должен нести маркер assertEffectiveNode`,
        );
      }
      if ("expectedNode" in item) {
        throw new Error(
          `${byName}: отвергающий кейс не должен нести expectedNode`,
        );
      }
      if (
        typeof item.clientError !== "string" ||
        item.clientError.trim() === ""
      ) {
        throw new Error(
          `${byName}: пустое ожидание ошибки клиента у отвергающего кейса`,
        );
      }
      return {
        name: item.name,
        tree: item.tree,
        accept: false,
        clientError: item.clientError,
      };
    }
    // Рантайм-охрана coherent-контракта маркера/ожидания: маркер, если
    // присутствует, обязан быть булевым; expectedNode без
    // assertEffectiveNode: true — ошибка fixture; маркер true требует
    // корректного expectedNode. Обычные принимающие кейсы expectedNode
    // не требуют.
    if ("assertEffectiveNode" in item) {
      if (typeof item.assertEffectiveNode !== "boolean") {
        throw new Error(`${byName}: маркер assertEffectiveNode не булево`);
      }
      if (!item.assertEffectiveNode) {
        if ("expectedNode" in item) {
          throw new Error(
            `${byName}: expectedNode без assertEffectiveNode: true — ошибка fixture`,
          );
        }
        return { name: item.name, tree: item.tree, accept: true };
      }
      if (!("expectedNode" in item)) {
        throw new Error(
          `${byName}: у кейса с assertEffectiveNode отсутствует expectedNode`,
        );
      }
      const node = validateExpectation(byName, item.expectedNode);
      return {
        name: item.name,
        tree: item.tree,
        accept: true,
        assertEffectiveNode: true,
        expectedNode: node,
      };
    }
    if ("expectedNode" in item) {
      throw new Error(
        `${byName}: expectedNode без assertEffectiveNode: true — ошибка fixture`,
      );
    }
    return { name: item.name, tree: item.tree, accept: true };
  });
  if (typeof raw.comment !== "string" || raw.comment.trim() === "") {
    throw new Error("общий fixture: comment не непустая строка");
  }
  return { comment: raw.comment, cases };
}

const parityCases =
  validateParityFixture(
    JSON.parse(readFileSync(fixturePath, "utf8")) as unknown,
  ).cases;

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

// Таблица coherent-контракта маркера/ожидания: те же правила, что
// проверяет Go-консьюмер fixture (visual/server/tree_test.go →
// validateEffectiveNodeContract).
type ContractCase = {
  name: string;
  accept?: boolean;
  assertEffectiveNode?: unknown;
  expectedNode?: unknown;
  clientError?: string;
  wantErr?: string;
  wantPath?: string;
  wantValue?: string;
};

describe("validateParityFixture: coherent-контракт маркера/ожидания", () => {
  const cases: ContractCase[] = [
    {
      name: "обычный принимающий кейс — без маркера и ожидания",
      accept: true,
    },
    {
      name: "назначенный кейс — маркер true и корректное ожидание (строка-value)",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: { path: "L", value: "3" },
      wantPath: "L",
      wantValue: "3",
    },
    {
      name: "value 0 — корректное значение",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: { path: "L", value: "0" },
      wantPath: "L",
      wantValue: "0",
    },
    {
      name: "value -0 — корректное значение",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: { path: "L", value: "-0" },
      wantPath: "L",
      wantValue: "-0",
    },
    {
      name: "путь в ожидании обрезается пробелами",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: { path: " root ", value: "-7" },
      wantPath: "root",
      wantValue: "-7",
    },
    {
      name: "отвергающий кейс не несёт маркер",
      accept: false,
      assertEffectiveNode: true,
      clientError: "не целое",
      wantErr: "не должен нести маркер",
    },
    {
      name: "отвергающий кейс не несёт ожидание",
      accept: false,
      expectedNode: { path: "L", value: "3" },
      clientError: "не целое",
      wantErr: "не должен нести expectedNode",
    },
    {
      name: "отвергающий кейс несёт null вместо expectedNode",
      accept: false,
      expectedNode: null,
      clientError: "не целое",
      wantErr: "не должен нести expectedNode",
    },
    {
      name: "маркер не булево",
      accept: true,
      assertEffectiveNode: "true",
      wantErr: "маркер assertEffectiveNode не булево",
    },
    {
      name: "маркер null не булево",
      accept: true,
      assertEffectiveNode: null,
      wantErr: "маркер assertEffectiveNode не булево",
    },
    {
      name: "маркер true без expectedNode",
      accept: true,
      assertEffectiveNode: true,
      wantErr: "отсутствует expectedNode",
    },
    {
      name: "expectedNode без маркера true",
      accept: true,
      expectedNode: { path: "L", value: "3" },
      wantErr: "expectedNode без assertEffectiveNode: true",
    },
    {
      name: "null вместо expectedNode без маркера true",
      accept: true,
      expectedNode: null,
      wantErr: "expectedNode без assertEffectiveNode: true",
    },
    {
      name: "маркер false с expectedNode",
      accept: true,
      assertEffectiveNode: false,
      expectedNode: { path: "L", value: "3" },
      wantErr: "expectedNode без assertEffectiveNode: true",
    },
    {
      name: "null вместо объекта ожидания",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: null,
      wantErr: "null вместо объекта ожидания",
    },
    {
      name: "пустой путь в ожидании",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: { path: "   ", value: "3" },
      wantErr: "пустой путь",
    },
    {
      name: "value не знаковое десятичное целое (лексическая форма 1.0)",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: { path: "L", value: "1.0" },
      wantErr: "не знаковое десятичное целое",
    },
    {
      name: "value вне безопасного диапазона (2^53)",
      accept: true,
      assertEffectiveNode: true,
      expectedNode: { path: "L", value: "9007199254740992" },
      wantErr: "не знаковое десятичное целое",
    },
  ];
  const base = { tree: `{"value":1}`, clientError: "не целое" };
  it.each(cases)("%s", (testCase) => {
    const fixture = {
      comment: "тестовый fixture",
      cases: [{ ...base, ...testCase }],
    };
    if (testCase.wantErr !== undefined) {
      expect(() => validateParityFixture(fixture)).toThrow(
        testCase.wantErr,
      );
      return;
    }
    // Удачная валидация обязана вернуть назначенный кейс с маркером
    // true и ожиданием с обрезанным путём; обычный принимающий кейс
    // не несёт ни маркера, ни ожидания.
    const validated = validateParityFixture(fixture);
    const returned = validated.cases[0] as Record<string, unknown>;
    const assigned = testCase.assertEffectiveNode === true;
    expect(returned.assertEffectiveNode).toBe(assigned ? true : undefined);
    if (assigned) {
      expect(returned.expectedNode).toEqual({
        path: testCase.wantPath,
        value: testCase.wantValue,
      });
    } else {
      expect(returned.expectedNode).toBeUndefined();
    }
  });
});

describe("isSignedDecimalInteger", () => {
  it.each(["0", "-0", "3", "-7", "123", "-123", "9007199254740991"])(
    "принимает %s",
    (value) => {
      expect(isSignedDecimalInteger(value)).toBe(true);
    },
  );
  it.each(["", "-", "1.0", "1e3", "+3", "01", "-01", " 3", "3 "])(
    "отвергает %s",
    (value) => {
      expect(isSignedDecimalInteger(value)).toBe(false);
    },
  );
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
