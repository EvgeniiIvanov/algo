// Парсинг сценария операций стека, очереди и списка на клиенте (зеркало
// visual/server/stack.go, visual/server/queue.go и visual/server/list.go):
// ловим ошибки ввода до Run; сервер перепроверяет — он авторитетен.
// ёмкость и значения push/enqueue — грамматика Go strconv.Atoi, реализованная
// в общем модуле decimalInteger.ts (см. parseDecimalInteger):
// клиент не пропустит формы, которые отвергнет сервер.

import { parseDecimalInteger } from "./decimalInteger";

/** Одна операция сценария стека: push со значением или pop. */
export interface StackOp {
  isPush: boolean;
  value: number;
}

/** Одна операция сценария очереди: enqueue со значением или dequeue. */
export interface QueueOp {
  isEnqueue: boolean;
  value: number;
}

/** Одна операция сценария списка: вставка на индекс или удаление по индексу. */
export interface ListOp {
  isInsert: boolean;
  index: number;
  value: number;
}

/** Ограничения входа — как на сервере (visual/server/stack.go, queue.go, list.go). */
export const MAX_STACK_CAPACITY = 100;
export const MAX_STACK_OPS = 500;
export const MAX_QUEUE_CAPACITY = 100;
export const MAX_QUEUE_OPS = 500;
export const MAX_LIST_OPS = 500;
export const MAX_LIST_LENGTH = 100;

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

/** Разобрать ёмкость очереди: целое от 1 до MAX_QUEUE_CAPACITY. */
export function parseQueueCapacity(raw: string): number {
  const trimmed = raw.trim();
  const n = parseDecimalInt(trimmed);
  if (n === null) {
    throw new Error(`не удалось разобрать «${trimmed}» как целое число: ёмкость очереди — одно целое, например 5`);
  }
  if (n < 1 || n > MAX_QUEUE_CAPACITY) {
    throw new Error(`ёмкость очереди должна быть от 1 до ${MAX_QUEUE_CAPACITY}, получено ${n}`);
  }
  return n;
}

/**
 * Разобрать сценарий операций очереди: одна на строку — «enqueue N»
 * или «dequeue». Пустые строки пропускаются. Бросает Error с понятным
 * сообщением. Проверяет и смысл: dequeue из пустой очереди — ошибка ввода;
 * enqueue в полную — легален (переполнение покажет кадр).
 */
export function parseQueueScenario(raw: string, capacity: number): QueueOp[] {
  const ops: QueueOp[] = [];
  const lines = raw.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const fields = lines[i]
      .trim()
      .split(/\s+/)
      .filter((f) => f.length > 0);
    if (fields.length === 0) continue; // пустая строка — не операция
    const lineNo = i + 1;

    if (fields[0] === "enqueue") {
      if (fields.length !== 2) {
        throw new Error(
          `строка ${lineNo}: enqueue требует ровно один аргумент — целое число, например «enqueue 5»`,
        );
      }
      const v = parseDecimalInt(fields[1]);
      if (v === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: enqueue N — значение одним целым`,
        );
      }
      ops.push({ isEnqueue: true, value: v });
    } else if (fields[0] === "dequeue") {
      if (fields.length !== 1) {
        throw new Error(`строка ${lineNo}: dequeue не принимает аргументов`);
      }
      ops.push({ isEnqueue: false, value: 0 });
    } else {
      throw new Error(
        `строка ${lineNo}: неизвестная операция «${fields[0]}»: ожидается «enqueue N» или «dequeue»`,
      );
    }
  }

  if (ops.length === 0) {
    throw new Error("сценарий пуст: нужна хотя бы одна операция (enqueue N или dequeue)");
  }
  if (ops.length > MAX_QUEUE_OPS) {
    throw new Error(`сценарий слишком длинный: ${ops.length} операций, максимум ${MAX_QUEUE_OPS}`);
  }

  // Смысловая проверка прогоном по длине очереди.
  let length = 0;
  for (let i = 0; i < ops.length; i++) {
    if (ops[i].isEnqueue) {
      if (length < capacity) length++;
      continue;
    }
    if (length === 0) {
      throw new Error(`операция №${i + 1}: dequeue из пустой очереди — нечего снимать`);
    }
    length--;
  }
  return ops;
}

/**
 * Разобрать сценарий операций списка: одна на строку — «insert I V»
 * (индекс 0..текущая длина включительно) или «remove I» (индекс
 * 0..текущая длина-1). Пустые строки пропускаются. Бросает Error
 * с понятным сообщением. Список растёт без ёмкости, но не длиннее
 * MAX_LIST_LENGTH — ряд ячеек должен оставаться обозримым (зеркало
 * server/list.go).
 */
export function parseListScenario(raw: string): ListOp[] {
  const ops: ListOp[] = [];
  const lines = raw.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const fields = lines[i]
      .trim()
      .split(/\s+/)
      .filter((f) => f.length > 0);
    if (fields.length === 0) continue; // пустая строка — не операция
    const lineNo = i + 1;

    if (fields[0] === "insert") {
      if (fields.length !== 3) {
        throw new Error(
          `строка ${lineNo}: insert требует ровно два аргумента — индекс и значение, например «insert 1 9»`,
        );
      }
      const index = parseDecimalInt(fields[1]);
      if (index === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: insert I V — индекс и значение целыми`,
        );
      }
      const v = parseDecimalInt(fields[2]);
      if (v === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[2]}» как целое число: insert I V — индекс и значение целыми`,
        );
      }
      ops.push({ isInsert: true, index, value: v });
    } else if (fields[0] === "remove") {
      if (fields.length !== 2) {
        throw new Error(
          `строка ${lineNo}: remove требует ровно один аргумент — индекс, например «remove 0»`,
        );
      }
      const index = parseDecimalInt(fields[1]);
      if (index === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: remove I — индекс одним целым`,
        );
      }
      ops.push({ isInsert: false, index, value: 0 });
    } else {
      throw new Error(
        `строка ${lineNo}: неизвестная операция «${fields[0]}»: ожидается «insert I V» или «remove I»`,
      );
    }
  }

  if (ops.length === 0) {
    throw new Error("сценарий пуст: нужна хотя бы одна операция (insert I V или remove I)");
  }
  if (ops.length > MAX_LIST_OPS) {
    throw new Error(`сценарий слишком длинный: ${ops.length} операций, максимум ${MAX_LIST_OPS}`);
  }

  // Смысловая проверка прогоном по текущей длине списка.
  let length = 0;
  for (let i = 0; i < ops.length; i++) {
    const op = ops[i];
    if (op.isInsert) {
      if (op.index < 0 || op.index > length) {
        throw new Error(
          `операция №${i + 1}: insert ${op.index} в список длины ${length} — индекс вне диапазона 0..${length}`,
        );
      }
      if (length + 1 > MAX_LIST_LENGTH) {
        throw new Error(
          `операция №${i + 1}: список вырастет до ${length + 1} значений — максимум ${MAX_LIST_LENGTH}; сократите сценарий`,
        );
      }
      length++;
      continue;
    }
    if (length === 0) {
      throw new Error(`операция №${i + 1}: remove из пустого списка — нечего удалять`);
    }
    if (op.index < 0 || op.index >= length) {
      throw new Error(
        `операция №${i + 1}: remove ${op.index} в списке длины ${length} — индекс вне диапазона 0..${length - 1}`,
      );
    }
    length--;
  }
  return ops;
}
