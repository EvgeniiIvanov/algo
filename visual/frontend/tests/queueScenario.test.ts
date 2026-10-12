// Тест шва фронта (см. docs/SPEC.md, шов фронтенда):
// парсинг сценария операций очереди и её ёмкости — ошибки ввода ловим до Run.
// Зеркало серверной валидации (visual/server/queue.go): сообщения совпадают.
import { describe, expect, it } from "vitest";
import { parseQueueCapacity, parseQueueScenario } from "../src/scenario";

describe("parseQueueCapacity", () => {
  it("разбирает положительное целое", () => {
    expect(parseQueueCapacity("5")).toBe(5);
    expect(parseQueueCapacity(" 12 ")).toBe(12);
  });

  it("принимает формы со знаком и ведущими нулями, как Go strconv.Atoi", () => {
    expect(parseQueueCapacity("+2")).toBe(2);
    expect(parseQueueCapacity("007")).toBe(7);
  });

  it("отклоняет мусор, дробные и пустую строку", () => {
    expect(() => parseQueueCapacity("три")).toThrow(/целое/);
    expect(() => parseQueueCapacity("2.5")).toThrow(/целое/);
    expect(() => parseQueueCapacity("1e2")).toThrow(/целое/);
    expect(() => parseQueueCapacity("")).toThrow(/целое/);
  });

  it("отклоняет ёмкость вне диапазона", () => {
    expect(() => parseQueueCapacity("0")).toThrow(/от 1 до/);
    expect(() => parseQueueCapacity("-2")).toThrow(/от 1 до/);
    expect(() => parseQueueCapacity("101")).toThrow(/от 1 до/);
  });
});

describe("parseQueueScenario", () => {
  it("разбирает сценарий enqueue/dequeue, пропуская пустые строки", () => {
    const ops = parseQueueScenario("enqueue 5\n\n  \nenqueue 7\ndequeue\n", 3);
    expect(ops).toEqual([
      { isEnqueue: true, value: 5 },
      { isEnqueue: true, value: 7 },
      { isEnqueue: false, value: 0 },
    ]);
  });

  it("отклоняет неизвестную операцию с номером строки", () => {
    expect(() => parseQueueScenario("enqueue 5\njump 1", 3)).toThrow(
      /строка 2.*неизвестная операция/,
    );
  });

  it("отклоняет enqueue без значения, с лишним аргументом и с мусором", () => {
    expect(() => parseQueueScenario("enqueue", 3)).toThrow(/enqueue требует/);
    expect(() => parseQueueScenario("enqueue 5 7", 3)).toThrow(/enqueue требует/);
    expect(() => parseQueueScenario("enqueue пять", 3)).toThrow(
      /не удалось разобрать «пять» как целое число/,
    );
  });

  it("значения enqueue следуют грамматике strconv.Atoi: со знаком и ведущими нулями можно, 0x/экспоненту/дроби — нет", () => {
    expect(parseQueueScenario("enqueue +2", 3)).toEqual([{ isEnqueue: true, value: 2 }]);
    expect(parseQueueScenario("enqueue -1", 3)).toEqual([{ isEnqueue: true, value: -1 }]);
    expect(parseQueueScenario("enqueue 007", 3)).toEqual([{ isEnqueue: true, value: 7 }]);

    expect(() => parseQueueScenario("enqueue 1e2", 3)).toThrow(
      /не удалось разобрать «1e2» как целое число/,
    );
    expect(() => parseQueueScenario("enqueue 0x10", 3)).toThrow(
      /не удалось разобрать «0x10» как целое число/,
    );
  });

  it("отклоняет dequeue с аргументом", () => {
    expect(() => parseQueueScenario("dequeue 3", 3)).toThrow(/dequeue не принимает/);
  });

  it("отклоняет пустой сценарий", () => {
    expect(() => parseQueueScenario("  \n\n", 3)).toThrow(/сценарий пуст/);
  });

  it("отклоняет dequeue из пустой очереди", () => {
    expect(() => parseQueueScenario("dequeue", 3)).toThrow(/пустой очереди/);
    expect(() => parseQueueScenario("enqueue 5\ndequeue\ndequeue", 3)).toThrow(
      /операция №3.*пустой очереди/,
    );
  });

  it("разрешает enqueue в полную очередь — переполнение показывает кадр, а не ошибку", () => {
    const ops = parseQueueScenario("enqueue 1\nenqueue 2", 1);
    expect(ops).toHaveLength(2);
  });

  it("отклоняет слишком длинный сценарий", () => {
    const script = Array.from({ length: 501 }, () => "dequeue").join("\n");
    expect(() => parseQueueScenario(script, 3)).toThrow(/слишком длинный/);
  });
});
