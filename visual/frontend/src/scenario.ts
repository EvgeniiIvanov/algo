// Парсинг сценария операций стека на клиенте (зеркало visual/server/stack.go):
// ловим ошибки ввода до Run; сервер перепроверяет — он авторитетен.
// ёмкость и значения push — грамматика Go strconv.Atoi, реализованная
// в общем модуле decimalInteger.ts (см. parseDecimalInteger):
// клиент не пропустит формы, которые отвергнет сервер.

import { parseDecimalInteger } from "./decimalInteger";

/** Одна операция сценария стека: push со значением или pop. */
export interface StackOp {
  isPush: boolean;
  value: number;
}

/** Ограничения входа — как на сервере (visual/server/stack.go). */
export const MAX_STACK_CAPACITY = 100;
export const MAX_STACK_OPS = 500;

/** Адаптер общего parseDecimalInteger под контракт сценария: null при любой
 * ошибке (синтаксис или небезопасный диапазон); контекстные сообщения
 * собирают caller'ы ниже. */
function parseDecimalInt(raw: string): number | null {
  const result = parseDecimalInteger(raw);
  return result.ok ? result.value : null;
}

/** Разобрать ёмкость стека: целое от 1 до MAX_STACK_CAPACITY. */
export function parseStackCapacity(raw: string): number {
  const trimmed = raw.trim();
  const n = parseDecimalInt(trimmed);
  if (n === null) {
    throw new Error(`не удалось разобрать «${trimmed}» как целое число: ёмкость стека — одно целое, например 5`);
  }
  if (n < 1 || n > MAX_STACK_CAPACITY) {
    throw new Error(`ёмкость стека должна быть от 1 до ${MAX_STACK_CAPACITY}, получено ${n}`);
  }
  return n;
}

/**
 * Разобрать сценарий операций: одна на строку — «push N» или «pop».
 * Пустые строки пропускаются. Бросает Error с понятным сообщением.
 * Проверяет и смысл: pop из пустого стека — ошибка ввода;
 * push в полный — легален (переполнение покажет кадр).
 */
export function parseStackScenario(raw: string, capacity: number): StackOp[] {
  const ops: StackOp[] = [];
  const lines = raw.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const fields = lines[i]
      .trim()
      .split(/\s+/)
      .filter((f) => f.length > 0);
    if (fields.length === 0) continue; // пустая строка — не операция
    const lineNo = i + 1;

    if (fields[0] === "push") {
      if (fields.length !== 2) {
        throw new Error(
          `строка ${lineNo}: push требует ровно один аргумент — целое число, например «push 5»`,
        );
      }
      const v = parseDecimalInt(fields[1]);
      if (v === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: push N — значение одним целым`,
        );
      }
      ops.push({ isPush: true, value: v });
    } else if (fields[0] === "pop") {
      if (fields.length !== 1) {
        throw new Error(`строка ${lineNo}: pop не принимает аргументов`);
      }
      ops.push({ isPush: false, value: 0 });
    } else {
      throw new Error(
        `строка ${lineNo}: неизвестная операция «${fields[0]}»: ожидается «push N» или «pop»`,
      );
    }
  }

  if (ops.length === 0) {
    throw new Error("сценарий пуст: нужна хотя бы одна операция (push N или pop)");
  }
  if (ops.length > MAX_STACK_OPS) {
    throw new Error(`сценарий слишком длинный: ${ops.length} операций, максимум ${MAX_STACK_OPS}`);
  }

  // Смысловая проверка прогоном по длине стека.
  let length = 0;
  for (let i = 0; i < ops.length; i++) {
    if (ops[i].isPush) {
      if (length < capacity) length++;
      continue;
    }
    if (length === 0) {
      throw new Error(`операция №${i + 1}: pop из пустого стека — нечего снимать`);
    }
    length--;
  }
  return ops;
}
