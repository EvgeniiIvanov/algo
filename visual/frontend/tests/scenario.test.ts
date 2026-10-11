// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// парсинг сценария операций стека и ёмкости — ошибки ввода ловим до Run.
// Зеркало серверной валидации (visual/server/stack.go): сообщения совпадают.
import { describe, expect, it } from "vitest";
import { parseStackCapacity, parseStackScenario } from "../src/scenario";

describe("parseStackCapacity", () => {
  it("разбирает положительное целое", () => {
    expect(parseStackCapacity("5")).toBe(5);
    expect(parseStackCapacity(" 12 ")).toBe(12);
  });

  it("принимает формы со знаком и ведущими нулями, как Go strconv.Atoi", () => {
    expect(parseStackCapacity("+2")).toBe(2);
    expect(parseStackCapacity("+50")).toBe(50);
    expect(parseStackCapacity("007")).toBe(7);
  });

  it("отклоняет мусор и дробные", () => {
    expect(() => parseStackCapacity("три")).toThrow(/целое/);
    expect(() => parseStackCapacity("2.5")).toThrow(/целое/);
    expect(() => parseStackCapacity("2.0")).toThrow(/целое/);
    expect(() => parseStackCapacity("")).toThrow(/целое/);
  });

  it("отклоняет формы, которые отвергнет Go strconv.Atoi на сервере", () => {
    expect(() => parseStackCapacity("1e2")).toThrow(/целое/);
    expect(() => parseStackCapacity("0x10")).toThrow(/целое/);
    expect(() => parseStackCapacity("1 2")).toThrow(/целое/);
    expect(() => parseStackCapacity(" ")).toThrow(/целое/);
  });

  it("отклоняет ёмкость вне диапазона", () => {
    expect(() => parseStackCapacity("0")).toThrow(/от 1 до/);
    expect(() => parseStackCapacity("-2")).toThrow(/от 1 до/);
    expect(() => parseStackCapacity("101")).toThrow(/от 1 до/);
    // Парсер знака принимает, диапазон отбраковывает: как на сервере.
    expect(() => parseStackCapacity("-7")).toThrow(/от 1 до/);
  });
});

describe("parseStackScenario", () => {
  it("разбирает сценарий push/pop, пропуская пустые строки", () => {
    const ops = parseStackScenario("push 5\n\n  \npush 7\npop\n", 3);
    expect(ops).toEqual([
      { isPush: true, value: 5 },
      { isPush: true, value: 7 },
      { isPush: false, value: 0 },
    ]);
  });

  it("отклоняет неизвестную операцию с номером строки", () => {
    expect(() => parseStackScenario("push 5\njump 1", 3)).toThrow(/строка 2.*неизвестная операция/);
  });

  it("отклоняет push без значения, с лишним аргументом и с мусором", () => {
    expect(() => parseStackScenario("push", 3)).toThrow(/push требует/);
    expect(() => parseStackScenario("push 5 7", 3)).toThrow(/push требует/);
    expect(() => parseStackScenario("push пять", 3)).toThrow(/не удалось разобрать «пять» как целое число/);
  });

  it("значения push следуют грамматике strconv.Atoi: со знаком и ведущими нулями можно, 0x/экспоненту/дроби — нет", () => {
    expect(parseStackScenario("push +2", 3)).toEqual([{ isPush: true, value: 2 }]);
    expect(parseStackScenario("push -1", 3)).toEqual([{ isPush: true, value: -1 }]);
    expect(parseStackScenario("push 007", 3)).toEqual([{ isPush: true, value: 7 }]);

    expect(() => parseStackScenario("push 1e2", 3)).toThrow(/не удалось разобрать «1e2» как целое число/);
    expect(() => parseStackScenario("push 0x10", 3)).toThrow(/не удалось разобрать «0x10» как целое число/);
    expect(() => parseStackScenario("push 2.0", 3)).toThrow(/не удалось разобрать «2.0» как целое число/);
  });

  it("отклоняет pop с аргументом", () => {
    expect(() => parseStackScenario("pop 3", 3)).toThrow(/pop не принимает/);
  });

  it("отклоняет пустой сценарий", () => {
    expect(() => parseStackScenario("  \n\n", 3)).toThrow(/сценарий пуст/);
  });

  it("отклоняет pop из пустого стека", () => {
    expect(() => parseStackScenario("pop", 3)).toThrow(/пустого стека/);
    expect(() => parseStackScenario("push 5\npop\npop", 3)).toThrow(/операция №3.*пустого стека/);
  });

  it("разрешает push в полный стек — переполнение показывает кадр, а не ошибку", () => {
    const ops = parseStackScenario("push 1\npush 2", 1);
    expect(ops).toHaveLength(2);
  });

  it("отклоняет слишком длинный сценарий", () => {
    const script = Array.from({ length: 501 }, () => "pop").join("\n");
    expect(() => parseStackScenario(script, 3)).toThrow(/слишком длинный/);
  });
});
