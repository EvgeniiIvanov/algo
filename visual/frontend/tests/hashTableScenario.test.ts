// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): парсинг сценария
// хеш-таблицы и числа корзин — ошибки ввода ловятся до Run. Зеркало
// серверной валидации (visual/server/hashtable.go): грамматика, целые числа
// и лимиты визуального Run.
import { describe, expect, it } from "vitest";
import {
  MAX_HASH_TABLE_BUCKETS,
  MAX_HASH_TABLE_ENTRIES,
  MAX_HASH_TABLE_OPS,
  parseHashTableBucketCount,
  parseHashTableScenario,
} from "../src/scenario";

describe("parseHashTableBucketCount", () => {
  it("разбирает одно целое в пределах лимита", () => {
    expect(parseHashTableBucketCount("5")).toBe(5);
    expect(parseHashTableBucketCount(" 1 ")).toBe(1);
    expect(parseHashTableBucketCount(String(MAX_HASH_TABLE_BUCKETS))).toBe(MAX_HASH_TABLE_BUCKETS);
  });

  it("отвергает ноль, отрицательное и превысившее лимит", () => {
    expect(() => parseHashTableBucketCount("0")).toThrow(/от 1 до 100/);
    expect(() => parseHashTableBucketCount("-1")).toThrow(/от 1 до 100/);
    expect(() =>
      parseHashTableBucketCount(String(MAX_HASH_TABLE_BUCKETS + 1)),
    ).toThrow(/от 1 до 100/);
    // Отрицательное — целое, но вне допустимых границ: сообщение про границы.
    expect(() => parseHashTableBucketCount("пять")).toThrow(/не удалось разобрать «пять»/);
    expect(() => parseHashTableBucketCount("")).toThrow(/не удалось разобрать/);
  });
});

describe("parseHashTableScenario — удачный разбор", () => {
  it("разбирает сценарий тикета целиком", () => {
    expect(parseHashTableScenario("put 1 10\nput 6 20\nget 1\nget 9\ndelete 6\ndelete 9")).toEqual([
      { kind: "put", key: 1, value: 10 },
      { kind: "put", key: 6, value: 20 },
      { kind: "get", key: 1, value: 0 },
      { kind: "get", key: 9, value: 0 },
      { kind: "delete", key: 6, value: 0 },
      { kind: "delete", key: 9, value: 0 },
    ]);
  });

  it("пропускает пустые строки и допускает отрицательные ключи и значения", () => {
    expect(parseHashTableScenario("\n\nput -4 40\n get -4 \n")).toEqual([
      { kind: "put", key: -4, value: 40 },
      { kind: "get", key: -4, value: 0 },
    ]);
  });

  it("промахи get/delete отсутствующего ключа валидны — не ошибки ввода", () => {
    expect(() => parseHashTableScenario("put 1 10\ndelete 3")).not.toThrow();
    expect(() => parseHashTableScenario("get 99")).not.toThrow();
    // Удаление отсутствующего ключа не растит таблицу и не проверяется по лимиту.
    expect(() => parseHashTableScenario("put 1 10\ndelete 99\ndelete 99")).not.toThrow();
  });
});

describe("parseHashTableScenario — ошибки ввода до Run", () => {
  it("отвергает неизвестную операцию", () => {
    expect(() => parseHashTableScenario("pop")).toThrow(/строка 1.*неизвестная операция «pop»/);
  });

  it("требует ровно два аргумента у put", () => {
    expect(() => parseHashTableScenario("put 1")).toThrow(/put требует/);
    expect(() => parseHashTableScenario("put 1 10 20")).toThrow(/put требует/);
  });

  it("требует один аргумент у get/delete", () => {
    expect(() => parseHashTableScenario("get")).toThrow(/get требует/);
    expect(() => parseHashTableScenario("get 1 2")).toThrow(/get требует/);
    expect(() => parseHashTableScenario("delete")).toThrow(/delete требует/);
    expect(() => parseHashTableScenario("delete 1 2")).toThrow(/delete требует/);
  });

  it("отвергает мусор вместо целых чисел", () => {
    expect(() => parseHashTableScenario("put один 10")).toThrow(/не удалось разобрать «один»/);
    expect(() => parseHashTableScenario("put 1 десять")).toThrow(/не удалось разобрать «десять»/);
    expect(() => parseHashTableScenario("get семь")).toThrow(/не удалось разобрать «семь»/);
    expect(() => parseHashTableScenario("delete ноль")).toThrow(/не удалось разобрать «ноль»/);
  });

  it("ограничения визуального Run: длина сценария и число записей", () => {
    const longOps = Array.from({ length: MAX_HASH_TABLE_OPS + 1 }, (_, i) => `put ${i} 1`).join("\n");
    expect(() => parseHashTableScenario(longOps)).toThrow(
      new RegExp(`слишком длинный.*максимум ${MAX_HASH_TABLE_OPS}`),
    );

    // Лимит записей: 101-й уникальный ключ — ошибка; повторный put того же
    // ключа места не занимает, delete освобождает.
    const tooMany = Array.from({ length: MAX_HASH_TABLE_ENTRIES + 1 }, (_, i) => `put ${i} 1`).join("\n");
    expect(() => parseHashTableScenario(tooMany)).toThrow(
      new RegExp(`будет ${MAX_HASH_TABLE_ENTRIES + 1} записей.*максимум ${MAX_HASH_TABLE_ENTRIES}`),
    );

    const churn: string[] = [];
    for (let i = 0; i < MAX_HASH_TABLE_ENTRIES + 1; i++) {
      churn.push(`put ${i} 1`, "delete 0"); // число записей не превышает лимит
    }
    expect(() => parseHashTableScenario(churn.join("\n"))).not.toThrow();
  });

  it("точная граница лимита операций: ровно 500 принимается, 501 отвергается", () => {
    const script = (n: number) =>
      Array.from({ length: n }, (_, i) => `get ${i}`).join("\n"); // get — легальные промахи

    expect(parseHashTableScenario(script(MAX_HASH_TABLE_OPS))).toHaveLength(MAX_HASH_TABLE_OPS);
    expect(() => parseHashTableScenario(script(MAX_HASH_TABLE_OPS + 1))).toThrow(
      new RegExp(`слишком длинный.*максимум ${MAX_HASH_TABLE_OPS}`),
    );
  });
});
