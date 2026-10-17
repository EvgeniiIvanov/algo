// Парсинг сценария операций стека, очереди, списка на слайсе,
// односвязного списка и входа массива сортировки на клиенте (зеркало
// visual/server/stack.go, visual/server/queue.go, visual/server/list.go,
// visual/server/linkedlist.go и visual/server/sort.go): ловим ошибки
// ввода до Run; сервер перепроверяет — он авторитетен.
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
export const MAX_LINKED_OPS = 500;
export const MAX_LINKED_LENGTH = 100;
/** Ограничения входа хеш-таблицы — как на сервере (visual/server/hashtable.go). */
export const MAX_HASH_TABLE_BUCKETS = 100;
export const MAX_HASH_TABLE_OPS = 500;
export const MAX_HASH_TABLE_ENTRIES = 100;
/** Ограничения входа сортировки — как на сервере (visual/server/sort.go). */
export const MAX_SORT_VALUES = 100;

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

/** Вид операции сценария односвязного списка. */
export type LinkedListOpKind = "prepend" | "append" | "insert" | "remove" | "find";

/** Одна операция сценария односвязного списка. */
export interface LinkedListOp {
  kind: LinkedListOpKind;
  index: number; // позиция для insert (0..Len) и remove (0..Len-1)
  value: number; // значение для prepend/append/insert/find
}

/**
 * Разобрать сценарий операций односвязного списка: одна на строку —
 * «prepend N», «append N», «insert I N» (индекс 0..текущая длина
 * включительно), «remove I» (индекс 0..текущая длина-1) или «find N».
 * Пустые строки пропускаются. Бросает Error с понятным сообщением.
 * find не проверяется по длине отсутствующего значения — оно легальный
 * промах (кадр покажет обход и возврат -1). Список растёт без ёмкости,
 * но не длиннее MAX_LINKED_LENGTH — цепочка узлов должна оставаться
 * обозримой (зеркало server/linkedlist.go).
 */
export function parseLinkedListScenario(raw: string): LinkedListOp[] {
  const ops: LinkedListOp[] = [];
  const lines = raw.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const fields = lines[i]
      .trim()
      .split(/\s+/)
      .filter((f) => f.length > 0);
    if (fields.length === 0) continue; // пустая строка — не операция
    const lineNo = i + 1;

    if (fields[0] === "prepend" || fields[0] === "append") {
      if (fields.length !== 2) {
        throw new Error(
          `строка ${lineNo}: ${fields[0]} требует ровно один аргумент — целое число, например «${fields[0]} 4»`,
        );
      }
      const v = parseDecimalInt(fields[1]);
      if (v === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: ${fields[0]} N — значение одним целым`,
        );
      }
      ops.push({ kind: fields[0], index: 0, value: v });
    } else if (fields[0] === "insert") {
      if (fields.length !== 3) {
        throw new Error(
          `строка ${lineNo}: insert требует ровно два аргумента — индекс и значение, например «insert 1 9»`,
        );
      }
      const index = parseDecimalInt(fields[1]);
      if (index === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: insert I N — индекс и значение целыми`,
        );
      }
      const v = parseDecimalInt(fields[2]);
      if (v === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[2]}» как целое число: insert I N — индекс и значение целыми`,
        );
      }
      ops.push({ kind: "insert", index, value: v });
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
      ops.push({ kind: "remove", index, value: 0 });
    } else if (fields[0] === "find") {
      if (fields.length !== 2) {
        throw new Error(
          `строка ${lineNo}: find требует ровно один аргумент — значение, например «find 7»`,
        );
      }
      const v = parseDecimalInt(fields[1]);
      if (v === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: find N — значение одним целым`,
        );
      }
      ops.push({ kind: "find", index: 0, value: v });
    } else {
      throw new Error(
        `строка ${lineNo}: неизвестная операция «${fields[0]}»: ожидается «prepend N», «append N», «insert I N», «remove I» или «find N»`,
      );
    }
  }

  if (ops.length === 0) {
    throw new Error(
      "сценарий пуст: нужна хотя бы одна операция (prepend N, append N, insert I N, remove I или find N)",
    );
  }
  if (ops.length > MAX_LINKED_OPS) {
    throw new Error(
      `сценарий слишком длинный: ${ops.length} операций, максимум ${MAX_LINKED_OPS}`,
    );
  }

  // Смысловая проверка прогоном по числу узлов.
  let length = 0;
  for (let i = 0; i < ops.length; i++) {
    const op = ops[i];
    if (op.kind === "prepend" || op.kind === "append") {
      if (length + 1 > MAX_LINKED_LENGTH) {
        throw new Error(
          `операция №${i + 1}: список вырастет до ${length + 1} узлов — максимум ${MAX_LINKED_LENGTH}; сократите сценарий`,
        );
      }
      length++;
      continue;
    }
    if (op.kind === "insert") {
      if (op.index < 0 || op.index > length) {
        throw new Error(
          `операция №${i + 1}: insert ${op.index} в список длины ${length} — индекс вне диапазона 0..${length}`,
        );
      }
      if (length + 1 > MAX_LINKED_LENGTH) {
        throw new Error(
          `операция №${i + 1}: список вырастет до ${length + 1} узлов — максимум ${MAX_LINKED_LENGTH}; сократите сценарий`,
        );
      }
      length++;
      continue;
    }
    if (op.kind === "remove") {
      if (length === 0) {
        throw new Error(`операция №${i + 1}: remove из пустого списка — нечего удалять`);
      }
      if (op.index < 0 || op.index >= length) {
        throw new Error(
          `операция №${i + 1}: remove ${op.index} в списке длины ${length} — индекс вне диапазона 0..${length - 1}`,
        );
      }
      length--;
      continue;
    }
    // find: значение, отсутствующее в списке, — легальный промах,
    // кадр покажет обход и возврат -1.
  }
  return ops;
}

/** Вид операции сценария хеш-таблицы. */
export type HashTableOpKind = "put" | "get" | "delete";

/** Одна операция сценария хеш-таблицы. */
export interface HashTableOp {
  kind: HashTableOpKind;
  key: number; // ключ операции
  value: number; // значение для put (в get/delete не используется)
}

/**
 * Разобрать число корзин хеш-таблицы: целое от 1 до MAX_HASH_TABLE_BUCKETS.
 * Бросает Error с понятным сообщением. Зеркало серверной проверки
 * (visual/server/hashtable.go): ноль и отрицательные значения отвергаются
 * и фронтендом, и библиотечным конструктором.
 */
export function parseHashTableBucketCount(raw: string): number {
  const trimmed = raw.trim();
  const n = parseDecimalInt(trimmed);
  if (n === null) {
    throw new Error(
      `не удалось разобрать «${trimmed}» как целое число: число корзин — одно целое, например 5`,
    );
  }
  if (n < 1 || n > MAX_HASH_TABLE_BUCKETS) {
    throw new Error(
      `число корзин должно быть от 1 до ${MAX_HASH_TABLE_BUCKETS}, получено ${n}`,
    );
  }
  return n;
}

/**
 * Разобрать сценарий операций хеш-таблицы: одна на строку — «put K V»
 * (ключ и значение), «get K» или «delete K». Пустые строки пропускаются.
 * Бросает Error с понятным сообщением. Промахи get/delete не проверяются
 * по содержимому таблицы: отсутствующий ключ — легальный шаг сценария,
 * кадр его покажет. Лимит одновременно хранимых записей проверяется
 * прогоном: put нового ключа растит таблицу (не более 100 записей),
 * повторный put того же ключа обновляет значение и места не занимает,
 * delete существующего ключа освобождает запись.
 */
export function parseHashTableScenario(raw: string): HashTableOp[] {
  const ops: HashTableOp[] = [];
  const lines = raw.split("\n");
  for (let i = 0; i < lines.length; i++) {
    const fields = lines[i]
      .trim()
      .split(/\s+/)
      .filter((f) => f.length > 0);
    if (fields.length === 0) continue; // пустая строка — не операция
    const lineNo = i + 1;

    if (fields[0] === "put") {
      if (fields.length !== 3) {
        throw new Error(
          `строка ${lineNo}: put требует ровно два аргумента — ключ и значение, например «put 1 10»`,
        );
      }
      const key = parseDecimalInt(fields[1]);
      if (key === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: put K V — ключ и значение целыми`,
        );
      }
      const value = parseDecimalInt(fields[2]);
      if (value === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[2]}» как целое число: put K V — ключ и значение целыми`,
        );
      }
      ops.push({ kind: "put", key, value });
    } else if (fields[0] === "get" || fields[0] === "delete") {
      if (fields.length !== 2) {
        throw new Error(
          `строка ${lineNo}: ${fields[0]} требует ровно один аргумент — ключ, например «${fields[0]} 1»`,
        );
      }
      const key = parseDecimalInt(fields[1]);
      if (key === null) {
        throw new Error(
          `строка ${lineNo}: не удалось разобрать «${fields[1]}» как целое число: ${fields[0]} K — ключ одним целым`,
        );
      }
      ops.push({ kind: fields[0], key, value: 0 });
    } else {
      throw new Error(
        `строка ${lineNo}: неизвестная операция «${fields[0]}»: ожидается «put K V», «get K» или «delete K»`,
      );
    }
  }

  if (ops.length === 0) {
    throw new Error(
      "сценарий пуст: нужна хотя бы одна операция (put K V, get K или delete K)",
    );
  }
  if (ops.length > MAX_HASH_TABLE_OPS) {
    throw new Error(
      `сценарий слишком длинный: ${ops.length} операций, максимум ${MAX_HASH_TABLE_OPS}`,
    );
  }

  // Смысловая проверка прогоном по числу записей таблицы.
  const keys = new Set<number>();
  for (let i = 0; i < ops.length; i++) {
    const op = ops[i];
    if (op.kind === "put") {
      if (!keys.has(op.key)) {
        if (keys.size + 1 > MAX_HASH_TABLE_ENTRIES) {
          throw new Error(
            `операция №${i + 1}: в таблице будет ${keys.size + 1} записей — максимум ${MAX_HASH_TABLE_ENTRIES}; сократите сценарий`,
          );
        }
        keys.add(op.key);
      }
      continue;
    }
    if (op.kind === "delete") {
      keys.delete(op.key); // промах тоже нормален: нечего освобождать
    }
    // get: отсутствующий ключ — легальный промах, кадр его покажет.
  }
  return ops;
}

/**
 * Разобрать массив сортировки: целые числа через запятую (зеркало
 * parseSortInput в visual/server/sort.go). Пустой ввод — допустимый
 * пустой массив: сортировать нечего, кадр покажет пустой ряд. Мусор
 * вместо целого и пустая компонента между запятыми — ошибки до Run;
 * грамматика целого — общий модуль decimalInteger.ts (strconv.Atoi
 * в пределах безопасного целого). Лимит длины — MAX_SORT_VALUES.
 */
export function parseSortValues(raw: string): number[] {
  const trimmed = raw.trim();
  if (trimmed === "") return [];

  const values: number[] = [];
  const parts = raw.split(",");
  for (const part of parts) {
    const result = parseDecimalInteger(part); // парсер триммирует сам
    if (!result.ok) {
      if (result.reason === "unsafe-range") {
        throw new Error(
          `«${result.input}» — за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
        );
      }
      throw new Error(
        `не удалось разобрать «${result.input}» как целое число: массив — целые числа через запятую, например «3,1,2»`,
      );
    }
    values.push(result.value);
  }
  if (values.length > MAX_SORT_VALUES) {
    throw new Error(
      `массив слишком длинный: ${values.length} элементов, максимум ${MAX_SORT_VALUES}; сократите вход`,
    );
  }
  return values;
}
