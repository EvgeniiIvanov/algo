// Тест общей токенизации строк сценария (src/scenarioFields.ts): поля —
// по whitespace-семантике Go strings.Fields / unicode.IsSpace, которой
// зеркалят все парсеры сценариев фронта. Кейсы согласованы с серверными
// тестами trie_test.go (TestTrieRunUnicodeWhitespaceSSE): NBSP U+00A0 и
// идеографический пробел U+3000 разделяют, CRLF корректен, строка только
// из Unicode whitespace пуста. Расхождения с JS \s — BOM U+FEFF (в Go не
// пробел) и NEL U+0085 (в Go пробел) — зафиксированы здесь же.
import { describe, expect, it } from "vitest";
import { splitScenarioFields } from "../src/scenarioFields";

describe("splitScenarioFields — базовая грамматика", () => {
  it("режет по одиночным и подряд идущим пробелам", () => {
    expect(splitScenarioFields("insert кот")).toEqual(["insert", "кот"]);
    expect(splitScenarioFields("  push\t 5 ")).toEqual(["push", "5"]);
  });

  it("пустая строка и строка из пробелов дают []", () => {
    expect(splitScenarioFields("")).toEqual([]);
    expect(splitScenarioFields("   \t \n ")).toEqual([]);
  });
});

describe("splitScenarioFields — паритет с Go unicode.IsSpace", () => {
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
