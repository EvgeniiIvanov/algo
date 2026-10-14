// Тест шва фронта (docs/SPEC.md, «Четвёртый шов»): парсинг сценария
// односвязного списка — ошибки ввода ловятся до Run. Зеркало серверной
// валидации (visual/server/linkedlist.go): грамматика и смысловые границы.
import { describe, expect, it } from "vitest";
import { MAX_LINKED_LENGTH, MAX_LINKED_OPS, parseLinkedListScenario } from "../src/scenario";

describe("parseLinkedListScenario — удачный разбор", () => {
  it("разбирает сценарий тикета целиком", () => {
    expect(parseLinkedListScenario("prepend 4\nappend 7\ninsert 1 9\nremove 0\nfind 7")).toEqual([
      { kind: "prepend", index: 0, value: 4 },
      { kind: "append", index: 0, value: 7 },
      { kind: "insert", index: 1, value: 9 },
      { kind: "remove", index: 0, value: 0 },
      { kind: "find", index: 0, value: 7 },
    ]);
  });

  it("пропускает пустые строки и допускает отрицательные значения — целые числа", () => {
    expect(parseLinkedListScenario("\n\nprepend -4\n find 0 \n")).toEqual([
      { kind: "prepend", index: 0, value: -4 },
      { kind: "find", index: 0, value: 0 },
    ]);
  });

  it("поиск: отсутствующее значение валидно — промах легален, не ошибка", () => {
    expect(() => parseLinkedListScenario("prepend 4\nfind 9")).not.toThrow();
  });
});

describe("parseLinkedListScenario — ошибки ввода до Run", () => {
  it("отвергает неизвестную операцию", () => {
    expect(() => parseLinkedListScenario("pop")).toThrow(/строка 1.*неизвестная операция «pop»/);
  });

  it("требует один аргумент у prepend/append/find", () => {
    expect(() => parseLinkedListScenario("prepend")).toThrow(/prepend требует/);
    expect(() => parseLinkedListScenario("append 1 2")).toThrow(/append требует/);
    expect(() => parseLinkedListScenario("find")).toThrow(/find требует/);
  });

  it("требует ровно два аргумента у insert", () => {
    expect(() => parseLinkedListScenario("insert 5")).toThrow(/insert требует/);
    expect(() => parseLinkedListScenario("insert 1 9 7")).toThrow(/insert требует/);
  });

  it("требует ровно один аргумент у remove", () => {
    expect(() => parseLinkedListScenario("remove")).toThrow(/remove требует/);
    expect(() => parseLinkedListScenario("remove 0 3")).toThrow(/remove требует/);
  });

  it("отвергает мусор вместо целых чисел", () => {
    expect(() => parseLinkedListScenario("prepend четыре")).toThrow(/не удалось разобрать «четыре»/);
    expect(() => parseLinkedListScenario("insert один 9")).toThrow(/не удалось разобрать «один»/);
    expect(() => parseLinkedListScenario("insert 1 девять")).toThrow(/не удалось разобрать «девять»/);
    expect(() => parseLinkedListScenario("remove ноль")).toThrow(/не удалось разобрать «ноль»/);
    expect(() => parseLinkedListScenario("find семь")).toThrow(/не удалось разобрать «семь»/);
  });

  it("insert: индекс вне 0..текущая длина — ошибка ввода", () => {
    expect(() => parseLinkedListScenario("insert 1 9")).toThrow(/вне диапазона 0\.\.0/);
    expect(() => parseLinkedListScenario("prepend 4\ninsert 2 9")).toThrow(/вне диапазона 0\.\.1/);
    expect(() => parseLinkedListScenario("insert -1 9")).toThrow(/вне диапазона/);
    expect(() => parseLinkedListScenario("prepend 4\ninsert 1 9")).not.toThrow(); // граница допустима
  });

  it("remove: из пустого списка или по индексу за хвостом — ошибка ввода", () => {
    expect(() => parseLinkedListScenario("remove 0")).toThrow(/remove из пустого списка/);
    expect(() => parseLinkedListScenario("prepend 4\nremove 0\nremove 0")).toThrow(/пустого списка/);
    expect(() => parseLinkedListScenario("prepend 4\nremove 1")).toThrow(/вне диапазона 0\.\.0/);
    expect(() => parseLinkedListScenario("prepend 4\nappend 7\nremove 1")).not.toThrow();
  });

  it("ограничения визуального Run: длина сценария и длины списка", () => {
    const longOps = Array.from({ length: MAX_LINKED_OPS + 1 }, (_, i) => `prepend ${i}`).join("\n");
    expect(() => parseLinkedListScenario(longOps)).toThrow(
      new RegExp(`слишком длинный.*максимум ${MAX_LINKED_OPS}`),
    );

    const tooLong = Array.from({ length: MAX_LINKED_LENGTH + 1 }, () => "append 1").join("\n");
    expect(() => parseLinkedListScenario(tooLong)).toThrow(
      new RegExp(`вырастет до ${MAX_LINKED_LENGTH + 1} узлов.*максимум ${MAX_LINKED_LENGTH}`),
    );
  });
});
