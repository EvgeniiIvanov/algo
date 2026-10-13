// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// парсинг сценария операций списка — ошибки ввода ловим до Run.
// Зеркало серверной валидации (visual/server/list.go): сообщения совпадают.
import { describe, expect, it } from "vitest";
import { parseListScenario } from "../src/scenario";

describe("parseListScenario", () => {
  it("разбирает сценарий insert/remove, пропуская пустые строки", () => {
    const ops = parseListScenario("insert 0 5\n\n  \ninsert 1 7\nremove 0\n");
    expect(ops).toEqual([
      { isInsert: true, index: 0, value: 5 },
      { isInsert: true, index: 1, value: 7 },
      { isInsert: false, index: 0, value: 0 },
    ]);
  });

  it("отклоняет неизвестную операцию с номером строки", () => {
    expect(() => parseListScenario("insert 0 5\njump 1")).toThrow(
      /строка 2.*неизвестная операция/,
    );
  });

  it("отклоняет insert без аргументов, с одним аргументом и с лишними", () => {
    expect(() => parseListScenario("insert")).toThrow(/insert требует/);
    expect(() => parseListScenario("insert 5")).toThrow(/insert требует/);
    expect(() => parseListScenario("insert 1 9 7")).toThrow(/insert требует/);
  });

  it("индекс и значение insert следуют грамматике strconv.Atoi", () => {
    expect(parseListScenario("insert +0 -2")).toEqual([
      { isInsert: true, index: 0, value: -2 },
    ]);
    expect(() => parseListScenario("insert один 9")).toThrow(
      /не удалось разобрать «один» как целое число/,
    );
    expect(() => parseListScenario("insert 1 1e2")).toThrow(
      /не удалось разобрать «1e2» как целое число/,
    );
  });

  it("отклоняет remove без индекса, с лишним аргументом и с мусором", () => {
    expect(() => parseListScenario("remove")).toThrow(/remove требует/);
    expect(() => parseListScenario("remove 0 3")).toThrow(/remove требует/);
    expect(() => parseListScenario("remove ноль")).toThrow(
      /не удалось разобрать «ноль» как целое число/,
    );
  });

  it("отклоняет пустой сценарий", () => {
    expect(() => parseListScenario("  \n\n")).toThrow(/сценарий пуст/);
  });

  it("отклоняет insert по индексу больше текущей длины", () => {
    expect(() => parseListScenario("insert 1 5")).toThrow(
      /операция №1: insert 1 в список длины 0 — индекс вне диапазона 0\.\.0/,
    );
    expect(() => parseListScenario("insert 0 5\ninsert 2 7")).toThrow(
      /операция №2: insert 2 в список длины 1 — индекс вне диапазона 0\.\.1/,
    );
  });

  it("отклоняет отрицательный индекс", () => {
    expect(() => parseListScenario("insert -1 5")).toThrow(/индекс вне диапазона/);
    expect(() => parseListScenario("insert 0 5\nremove -1")).toThrow(
      /индекс вне диапазона/,
    );
  });

  it("отклоняет remove из пустого списка и по индексу за хвостом", () => {
    expect(() => parseListScenario("remove 0")).toThrow(/пустого списка/);
    expect(() => parseListScenario("insert 0 5\nremove 1")).toThrow(
      /операция №2: remove 1 в списке длины 1 — индекс вне диапазона 0\.\.0/,
    );
  });

  it("разрешает вставку в хвост по индексу, равному длине", () => {
    const ops = parseListScenario("insert 0 5\ninsert 1 7\ninsert 2 9");
    expect(ops).toHaveLength(3);
  });

  it("отклоняет список длиннее MAX_LIST_LENGTH", () => {
    const script = Array.from({ length: 101 }, () => "insert 0 1").join("\n");
    expect(() => parseListScenario(script)).toThrow(
      /вырастет до 101 значени.*максимум 100/,
    );
  });

  it("отклоняет слишком длинный сценарий", () => {
    const script = Array.from({ length: 501 }, () => "remove 0").join("\n");
    expect(() => parseListScenario(script)).toThrow(/слишком длинный/);
  });
});
