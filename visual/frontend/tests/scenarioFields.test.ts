// Тесты токенизации строк сценария (src/scenarioFields.ts). Два набора
// семантики под разные швы (см. шапку модуля):
//
// - splitScenarioFields — whitespace по Go unicode.IsSpace / strings.Fields,
//   которой зеркалит парсер сценария Trie; кейсы согласованы с серверными
//   тестами trie_test.go (TestTrieRunUnicodeWhitespaceSSE): NBSP U+00A0
//   и идеографический пробел U+3000 разделяют, CRLF корректен, строка
//   только из Unicode whitespace пуста. Расхождения с JS \s — BOM U+FEFF
//   (в Go не пробел) и NEL U+0085 (в Go пробел) — зафиксированы здесь же.
//
// - splitLegacyScenarioFields — прежняя JS-токенизация (trim + \s) шести
//   исторических парсеров (стек, очередь, список, односвязный список,
//   хеш-таблица, BST): наблюдаемое поведение этих швов не менялось (AC19),
//   здесь оно и держится как регрессия.
import { describe, expect, it } from "vitest";
import {
  splitLegacyScenarioFields,
  splitScenarioFields,
} from "../src/scenarioFields";

describe("splitScenarioFields — базовая грамматика (Trie)", () => {
  it("режет по одиночным и подряд идущим пробелам", () => {
    expect(splitScenarioFields("insert кот")).toEqual(["insert", "кот"]);
    expect(splitScenarioFields("  push\t 5 ")).toEqual(["push", "5"]);
  });

  it("пустая строка и строка из пробелов дают []", () => {
    expect(splitScenarioFields("")).toEqual([]);
    expect(splitScenarioFields("   \t \n ")).toEqual([]);
  });
});

describe("splitScenarioFields — паритет с Go unicode.IsSpace (Trie)", () => {
  it("NBSP U+00A0 и идеографический пробел U+3000 разделяют поля", () => {
    expect(splitScenarioFields("insert\u00A0кот")).toEqual(["insert", "кот"]);
    expect(splitScenarioFields("insert\u3000кот")).toEqual(["insert", "кот"]);
  });

  it("NEL U+0085 — пробел в Go, в JS \\s пробельного нет", () => {
    expect(splitScenarioFields("insert\u0085кот")).toEqual(["insert", "кот"]);
    expect(splitScenarioFields("\u0085insert\u0085кот\u0085")).toEqual([
      "insert",
      "кот",
    ]);
  });

  it("BOM U+FEFF — не пробел в Go: приклеивается к токену", () => {
    expect(splitScenarioFields("\uFEFFinsert кот")).toEqual([
      "\uFEFFinsert",
      "кот",
    ]);
    expect(splitScenarioFields("insert\uFEFFкот")).toEqual(["insert\uFEFFкот"]);
  });

  it("U+200B (zero width space) — не пробел: часть токена", () => {
    expect(splitScenarioFields("insert\u200Bкот")).toEqual(["insert\u200Bкот"]);
  });

  it("CRLF: \\r перед переносом строки — пробел, не часть токена", () => {
    expect(splitScenarioFields("insert кот\r")).toEqual(["insert", "кот"]);
    expect(splitScenarioFields("delete\tкот\r")).toEqual(["delete", "кот"]);
  });

  it("строка только из Unicode whitespace даёт []", () => {
    expect(splitScenarioFields("\u00A0\u3000\u0085\u2028")).toEqual([]);
  });
});

describe("splitLegacyScenarioFields — прежняя JS-токенизация исторических парсеров", () => {
  it("режет по одиночным и подряд идущим пробелам", () => {
    expect(splitLegacyScenarioFields("insert кот")).toEqual(["insert", "кот"]);
    expect(splitLegacyScenarioFields("  push\t 5 ")).toEqual(["push", "5"]);
  });

  it("пустая строка и строка из пробелов дают []", () => {
    expect(splitLegacyScenarioFields("")).toEqual([]);
    expect(splitLegacyScenarioFields("   \t \n ")).toEqual([]);
  });

  it("регрессия: BOM U+FEFF — пробельный в JS, отделяется от токена", () => {
    expect(splitLegacyScenarioFields("\uFEFFpush 5")).toEqual(["push", "5"]);
    expect(splitLegacyScenarioFields("insert\uFEFFкот")).toEqual([
      "insert",
      "кот",
    ]);
  });

  it("регрессия: NEL U+0085 — не пробельный в JS, часть токена", () => {
    expect(splitLegacyScenarioFields("insert\u0085кот")).toEqual([
      "insert\u0085кот",
    ]);
    expect(splitLegacyScenarioFields("\u0085insert\u0085кот")).toEqual([
      "\u0085insert\u0085кот",
    ]);
  });

  it("NBSP U+00A0 и идеографический пробел U+3000 разделяют и в JS \\s", () => {
    expect(splitLegacyScenarioFields("put\u00A01\u00A010")).toEqual([
      "put",
      "1",
      "10",
    ]);
    expect(splitLegacyScenarioFields("insert\u3000кот")).toEqual([
      "insert",
      "кот",
    ]);
  });

  it("U+200B (zero width space) — не пробел и в JS \\s: часть токена", () => {
    expect(splitLegacyScenarioFields("insert\u200Bкот")).toEqual([
      "insert\u200Bкот",
    ]);
  });

  it("CRLF: \\r — пробельный в JS \\s, не часть токена", () => {
    expect(splitLegacyScenarioFields("insert кот\r")).toEqual([
      "insert",
      "кот",
    ]);
  });

  it("строка только из JS-пробелов даёт []", () => {
    expect(splitLegacyScenarioFields("\u00A0\u3000\u2028\uFEFF")).toEqual([]);
  });
});
