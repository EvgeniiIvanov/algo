// Парсинг входа обхода бинарного дерева на клиенте (зеркало
// visual/server/tree.go): ловим ошибки ввода до Run; сервер перепроверяет —
// он авторитетен. Вложенный JSON дерева (value, left, right): отсутствующий
// или null потомок — пустая ветвь, верхний null — пустое дерево.
// Лимит узлов — MAX_TREE_NODES, как на сервере.

/** Ограничения входа — как на сервере (visual/server/tree.go). */
export const MAX_TREE_NODES = 100;

/** Порядки обхода — как на сервере (render.TreeOrder*). */
export const TREE_ORDERS = ["in-order", "pre-order", "post-order"] as const;
export type TreeOrder = (typeof TREE_ORDERS)[number];

/** Страж типа: выбран порядок обхода дерева. */
export function isTreeOrder(value: string): value is TreeOrder {
  return (TREE_ORDERS as readonly string[]).includes(value);
}

/** Разобрать порядок обхода: бросает Error для неизвестного или пустого. */
export function parseTreeOrder(raw: string): TreeOrder {
  const trimmed = raw.trim();
  if (trimmed === "") {
    throw new Error("выберите порядок обхода: in-order, pre-order или post-order");
  }
  if (!isTreeOrder(trimmed)) {
    throw new Error(
      `неизвестный порядок обхода «${trimmed}»: ожидается in-order, pre-order или post-order`,
    );
  }
  return trimmed;
}

/**
 * Проверить вложенный JSON дерева (зеркало parseTreeJSON сервера):
 * верхний null — пустое дерево; каждый узел — объект {value, left, right}
 * с целым значением; отсутствующий или null потомок — пустая ветвь;
 * больше MAX_TREE_NODES узлов — ошибка. Бросает Error с понятным
 * сообщением. Числа за пределами точного диапазона JavaScript клиент
 * отвергает сам, а не молча округляет (см. decimalInteger.ts).
 */
export function parseTreeInput(raw: string): void {
  if (raw.trim() === "") {
    throw new Error(
      "введите дерево — вложенный JSON, например {\"value\":1,\"left\":{\"value\":2}}; null — пустое дерево",
    );
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch (err) {
    throw new Error(`не удалось разобрать JSON дерева: ${(err as Error).message}`, { cause: err });
  }
  buildTreeInput(parsed, "", 0);
}

/**
 * Рекурсивная проверка узла (зеркало buildTree на сервере): value —
 * целое в пределах безопасного диапазона, потомки — вложенные узлы
 * или null/без ключа. count — число узлов выше по дереву; путь —
 * символы L/R от корня ("" — корень) для сообщений об ошибках.
 * Возвращает число узлов в поддереве.
 */
function buildTreeInput(parsed: unknown, branch: string, count: number): number {
  if (parsed === null) {
    return 0; // верхний null или null-потомок — пустая ветвь
  }
  if (typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error(
      `узел дерева${treeContext(branch)} должен быть объектом {value, left, right} или null`,
    );
  }
  const obj = parsed as Record<string, unknown>;
  const value = obj.value;
  if (typeof value !== "number" || !Number.isInteger(value)) {
    throw new Error(
      `значение узла${treeContext(branch)} не целое: value — целое число`,
    );
  }
  if (Math.abs(value) > Number.MAX_SAFE_INTEGER) {
    throw new Error(
      `значение узла${treeContext(branch)} за пределами точного диапазона JavaScript: поддерживаются целые от -${Number.MAX_SAFE_INTEGER} до ${Number.MAX_SAFE_INTEGER}`,
    );
  }
  if (count + 1 > MAX_TREE_NODES) {
    throw new Error(
      `дерево слишком большое: разрешено не более ${MAX_TREE_NODES} узлов; сократите вход`,
    );
  }
  // Отсутствующий ключ и явный null равнозначны — оба пустая ветвь.
  const leftCount =
    obj.left === undefined || obj.left === null
      ? 0
      : buildTreeInput(obj.left, branch + "L", count + 1);
  const rightCount =
    obj.right === undefined || obj.right === null
      ? 0
      : buildTreeInput(obj.right, branch + "R", count + 1 + leftCount);
  return 1 + leftCount + rightCount;
}

/** Сегмент сообщения об ошибке: путь узла от корня ("" — корень). */
function treeContext(branch: string): string {
  return branch === "" ? " (корень)" : ` (путь ${branch})`;
}
