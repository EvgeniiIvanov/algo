// Тупой рендерер кадра в DOM (ADR-0002):
// кадр самодостаточен, состояния между кадрами не накапливаем.
// На вход Frame — на выход мутация двух DOM-узлов (пояснение + контейнер).
//
// Поддерживаем kind="array" (бинарный поиск), kind="structure"
// (стек "stack", очередь "queue", список "list", односвязный список
// "linked-list" — узлы со стрелками next), kind="hash-table"
// (корзины с цепочками записей), kind="sort" (маршруты сортировок —
// bubble sort и далее) и kind="tree" (бинарное дерево, обходы
// in-/pre-/post-order). Новые kind'ы — новая ветка без правок
// существующих.

import {
  KIND_ARRAY,
  KIND_HASH_TABLE,
  KIND_SORT,
  KIND_STRUCTURE,
  KIND_TREE,
  type ArraySnapshot,
  type Frame,
  type HashEntry,
  type HashEntryHighlight,
  type HashEntryRemoval,
  type HashTableSnapshot,
  type SortHighlight,
  type SortRange,
  type SortSnapshot,
  type StructureSnapshot,
  type TreeSnapshot,
  type TreeNodeData,
} from "./types";

/** Внутренний помощник: бросает, если снимок не похож на снимок массива. */
function asArraySnapshot(data: unknown): ArraySnapshot {
  if (data === null || typeof data !== "object") {
    throw new Error("снимок не объект");
  }
  const obj = data as Record<string, unknown>;
  if (!Array.isArray(obj.values)) {
    throw new Error("снимок массива: поле values должно быть массивом");
  }
  for (const v of obj.values) {
    if (typeof v !== "number") {
      throw new Error("снимок массива: все elements.values — числа");
    }
  }
  for (const field of ["low", "high", "mid", "foundIndex"] as const) {
    if (typeof obj[field] !== "number" || !Number.isInteger(obj[field] as number)) {
      throw new Error(`снимок массива: поле ${field} должно быть целым числом`);
    }
  }
  return obj as unknown as ArraySnapshot;
}

/** Внутренний помощник: бросает, если снимок не похож на снимок структуры. */
function asStructureSnapshot(data: unknown): StructureSnapshot {
  if (data === null || typeof data !== "object") {
    throw new Error("снимок не объект");
  }
  const obj = data as Record<string, unknown>;
  if (typeof obj.structure !== "string") {
    throw new Error("снимок структуры: поле structure должно быть строкой");
  }
  if (
    typeof obj.capacity !== "number" ||
    !Number.isInteger(obj.capacity) ||
    obj.capacity < 0
  ) {
    throw new Error(
      "снимок структуры: поле capacity должно быть целым числом от 0: у фиксированных структур это заданная ёмкость, у списка — текущее число ячеек",
    );
  }
  if (!Array.isArray(obj.values)) {
    throw new Error("снимок структуры: поле values должно быть массивом");
  }
  for (const v of obj.values) {
    if (typeof v !== "number") {
      throw new Error("снимок структуры: все values — числа");
    }
  }
  if (!Array.isArray(obj.highlights)) {
    throw new Error("снимок структуры: поле highlights должно быть массивом");
  }
  for (const h of obj.highlights as unknown[]) {
    if (h === null || typeof h !== "object") {
      throw new Error("снимок структуры: подсветка должна быть объектом {index, role}");
    }
    const hl = h as Record<string, unknown>;
    if (typeof hl.index !== "number" || !Number.isInteger(hl.index)) {
      throw new Error("снимок структуры: поле index подсветки должно быть целым числом");
    }
    if (typeof hl.role !== "string") {
      throw new Error("снимок структуры: поле role подсветки должно быть строкой");
    }
  }
  if (typeof obj.top !== "number" || !Number.isInteger(obj.top)) {
    throw new Error("снимок структуры: поле top должно быть целым числом");
  }
  // Указатели очереди head/tail — опциональные: старые снимки стека их не несут.
  for (const field of ["head", "tail"] as const) {
    if (obj[field] !== undefined && (typeof obj[field] !== "number" || !Number.isInteger(obj[field]))) {
      throw new Error(`снимок структуры: поле ${field} должно быть целым числом`);
    }
  }
  if (typeof obj.operation !== "string") {
    throw new Error("снимок структуры: поле operation должно быть строкой");
  }
  // Согласованность по типу структуры: у списка (на слайсе и на ссылках)
  // ёмкость — это число существующих ячеек/узлов и обязана совпадать с
  // числом значений (пустых список не рисует); у stack/queue заданная
  // ёмкость положительна — их серверный контракт не допускает нулевого ряда.
  if (obj.structure === "list" || obj.structure === "linked-list") {
    if (obj.capacity !== obj.values.length) {
      throw new Error(
        "снимок списка: поле capacity должно совпадать с числом значений — пустых ячеек список не рисует",
      );
    }
  } else if (obj.structure === "stack" || obj.structure === "queue") {
    if (obj.capacity === 0) {
      throw new Error(
        "снимок структуры: у stack и queue заданная ёмкость положительна — нулевой ряд не валиден",
      );
    }
  }
  return obj as unknown as StructureSnapshot;
}

/** Является ли кадр снимком массива с допустимыми полями. */
export function isArrayFrame(frame: Frame): boolean {
  if (frame.kind !== KIND_ARRAY) return false;
  try {
    asArraySnapshot(frame.data);
    return true;
  } catch {
    return false;
  }
}

/** Является ли кадр снимком линейной структуры с допустимыми полями. */
export function isStructureFrame(frame: Frame): boolean {
  if (frame.kind !== KIND_STRUCTURE) return false;
  try {
    asStructureSnapshot(frame.data);
    return true;
  } catch {
    return false;
  }
}

/** Внутренний помощник: бросает, если снимок не похож на снимок хеш-таблицы. */
function asHashTableSnapshot(data: unknown): HashTableSnapshot {
  if (data === null || typeof data !== "object") {
    throw new Error("снимок не объект");
  }
  const obj = data as Record<string, unknown>;
  if (typeof obj.bucketCount !== "number" || !Number.isInteger(obj.bucketCount) || obj.bucketCount < 1) {
    throw new Error(
      "снимок хеш-таблицы: поле bucketCount должно быть положительным целым — нулевых таблиц не существует",
    );
  }
  if (!Array.isArray(obj.buckets) || obj.buckets.length !== obj.bucketCount) {
    throw new Error(
      "снимок хеш-таблицы: поле buckets должно быть массивом длины bucketCount — каждая корзина обязана присутствовать в кадре",
    );
  }
  for (const chain of obj.buckets) {
    if (!Array.isArray(chain)) {
      throw new Error("снимок хеш-таблицы: каждая корзина — массив записей");
    }
    for (const e of chain) {
      if (e === null || typeof e !== "object") {
        throw new Error("снимок хеш-таблицы: запись должна быть объектом {key, value}");
      }
      const entry = e as Record<string, unknown>;
      for (const field of ["key", "value"] as const) {
        if (typeof entry[field] !== "number" || !Number.isInteger(entry[field])) {
          throw new Error(`снимок хеш-таблицы: поле ${field} записи должно быть целым числом`);
        }
      }
    }
  }
  if (typeof obj.highlightBucket !== "number" || !Number.isInteger(obj.highlightBucket)) {
    throw new Error("снимок хеш-таблицы: поле highlightBucket должно быть целым числом");
  }
  if (obj.highlightBucket < -1 || obj.highlightBucket >= obj.bucketCount) {
    throw new Error(
      "снимок хеш-таблицы: highlightBucket вне диапазона корзин (0..bucketCount-1, -1 — нет выбора)",
    );
  }
  if (!Array.isArray(obj.highlights)) {
    throw new Error("снимок хеш-таблицы: поле highlights должно быть массивом");
  }
  for (const h of obj.highlights) {
    if (h === null || typeof h !== "object") {
      throw new Error("снимок хеш-таблицы: подсветка должна быть объектом {bucket, position, role}");
    }
    const hl = h as HashEntryHighlight;
    if (typeof hl.bucket !== "number" || !Number.isInteger(hl.bucket)) {
      throw new Error("снимок хеш-таблицы: поле bucket подсветки должно быть целым числом");
    }
    if (typeof hl.position !== "number" || !Number.isInteger(hl.position)) {
      throw new Error("снимок хеш-таблицы: поле position подсветки должно быть целым числом");
    }
    if (typeof hl.role !== "string") {
      throw new Error("снимок хеш-таблицы: поле role подсветки должно быть строкой");
    }
  }
  // Надгробия шага: удалённые записи на бывших позициях (не входят в
  // buckets — иначе живой снимок был бы искажён). Отсутствие поля — ошибка:
  // снимок от рендерера всегда несёт массив, пусть и пустой.
  if (!Array.isArray(obj.removed)) {
    throw new Error(
      "снимок хеш-таблицы: поле removed должно быть массивом надгробий (пустым, если шаг без удаления)",
    );
  }
  for (const item of obj.removed) {
    if (item === null || typeof item !== "object") {
      throw new Error("снимок хеш-таблицы: надгробие должно быть объектом {bucket, position, key, value}");
    }
    const gone = item as Record<string, unknown>;
    for (const field of ["bucket", "position", "key", "value"] as const) {
      if (typeof gone[field] !== "number" || !Number.isInteger(gone[field])) {
        throw new Error(`снимок хеш-таблицы: поле ${field} надгробия должно быть целым числом`);
      }
    }
    // bucket ведёт в массив корзин рендера: вне диапазона он рисовать не может.
    const removal = item as HashEntryRemoval;
    if (removal.bucket < 0 || removal.bucket >= obj.bucketCount) {
      throw new Error(
        "снимок хеш-таблицы: поле bucket надгробия вне диапазона корзин (0..bucketCount-1)",
      );
    }
  }
  let entries = 0;
  for (const chain of obj.buckets as unknown[][]) {
    entries += chain.length;
  }
  if (obj.entryCount !== entries) {
    throw new Error(
      "снимок хеш-таблицы: поле entryCount должно совпадать с числом записей во всех цепочках",
    );
  }
  if (typeof obj.operation !== "string") {
    throw new Error("снимок хеш-таблицы: поле operation должно быть строкой");
  }
  return obj as unknown as HashTableSnapshot;
}

/** Является ли кадр снимком хеш-таблицы с допустимыми полями. */
export function isHashTableFrame(frame: Frame): boolean {
  if (frame.kind !== KIND_HASH_TABLE) return false;
  try {
    asHashTableSnapshot(frame.data);
    return true;
  } catch {
    return false;
  }
}

/** Внутренний помощник: бросает, если снимок не похож на снимок сортировки. */
function asSortSnapshot(data: unknown): SortSnapshot {
  if (data === null || typeof data !== "object") {
    throw new Error("снимок не объект");
  }
  const obj = data as Record<string, unknown>;
  if (!Array.isArray(obj.values)) {
    throw new Error("снимок сортировки: поле values должно быть массивом");
  }
  for (const v of obj.values) {
    if (typeof v !== "number") {
      throw new Error("снимок сортировки: все values — числа");
    }
  }
  if (!Array.isArray(obj.highlights)) {
    throw new Error("снимок сортировки: поле highlights должно быть массивом");
  }
  for (const h of obj.highlights as unknown[]) {
    if (h === null || typeof h !== "object") {
      throw new Error("снимок сортировки: подсветка должна быть объектом {index, role}");
    }
    const hl = h as Record<string, unknown>;
    if (typeof hl.index !== "number" || !Number.isInteger(hl.index)) {
      throw new Error("снимок сортировки: поле index подсветки должно быть целым числом");
    }
    if (typeof hl.role !== "string") {
      throw new Error("снимок сортировки: поле role подсветки должно быть строкой");
    }
  }
  // Подсветка ведёт в массив рендера: индекс вне диапазона рисовать нечем.
  for (const hl of obj.highlights as SortHighlight[]) {
    if (hl.index < 0 || hl.index >= obj.values.length) {
      throw new Error(
        "снимок сортировки: индекс подсветки вне диапазона массива (0..values.length-1)",
      );
    }
  }
  if (typeof obj.operation !== "string") {
    throw new Error("снимок сортировки: поле operation должно быть строкой");
  }
  // Состояние вставки insertion sort: «дырка» — валидный индекс массива
  // или -1 (шага без дырки); ключ — значение, осмысленное при дырке.
  // Допустимы ровно две формы снимка: старый без обоих полей (кадры
  // bubble/selection и кадры до тикета 21 — нормализуются к шагу без
  // вставки, чтобы код рендера видел поля явно) и новый с обоими полями.
  // Частичная пара (только gap или только key) — искажённый снимок,
  // как и присутствующие кривые значения.
  const hasGap = obj.gap !== undefined;
  const hasKey = obj.key !== undefined;
  if (hasGap !== hasKey) {
    throw new Error(
      "снимок сортировки: поля gap и key идут парой — частичное состояние вставки не читается",
    );
  }
  let gap = -1;
  if (hasGap) {
    if (
      typeof obj.gap !== "number" ||
      !Number.isInteger(obj.gap) ||
      obj.gap < -1 ||
      obj.gap >= (obj.values as number[]).length
    ) {
      throw new Error(
        "снимок сортировки: поле gap должно быть индексом «дырки» вставки (-1 или позиция в массиве)",
      );
    }
    gap = obj.gap;
  }
  let key = 0;
  if (hasKey) {
    if (typeof obj.key !== "number" || !Number.isInteger(obj.key)) {
      throw new Error(
        "снимок сортировки: поле key должно быть целым числом (значение удерживаемого ключа; игнорируется при gap = -1)",
      );
    }
    key = obj.key;
  }
  // Нормализованный снимок: gap/key заданы явно — рендер не различает
  // старые и новые кадры.
  // Активные диапазоны сортировок: старый снимок без поля читается как
  // пустой список; кривые значения — ошибка.
  let ranges: SortRange[] = [];
  if (obj.ranges !== undefined) {
    if (!Array.isArray(obj.ranges)) {
      throw new Error("снимок сортировки: поле ranges должно быть массивом активных диапазонов");
    }
    ranges = (obj.ranges as unknown[]).map((r) => {
      if (r === null || typeof r !== "object") {
        throw new Error("снимок сортировки: активный диапазон должен быть объектом {lo, hi, mid?}");
      }
      const rg = r as Record<string, unknown>;
      for (const field of ["lo", "hi"] as const) {
        if (typeof rg[field] !== "number" || !Number.isInteger(rg[field])) {
          throw new Error(`снимок сортировки: поле ${field} активного диапазона должно быть целым числом`);
        }
      }
      const range = rg as unknown as SortRange;
      if (range.lo < 0 || range.hi > (obj.values as number[]).length || range.lo >= range.hi) {
        throw new Error(
          "снимок сортировки: активный диапазон должен быть непустым отрезком внутри массива",
        );
      }
      // Граница половин mid несёт только merge sort; у partition quicksort
      // её нет — поле отсутствует. Если mid задан, он обязан делить
      // диапазон на непустые части.
      if (range.mid !== undefined) {
        if (typeof range.mid !== "number" || !Number.isInteger(range.mid)) {
          throw new Error(
            "снимок сортировки: поле mid активного диапазона должно быть целым числом",
          );
        }
        if (range.mid <= range.lo || range.mid >= range.hi) {
          throw new Error(
            "снимок сортировки: граница половин mid должна делить активный диапазон на непустые части",
          );
        }
      }
      return range;
    });
  }
  return { ...obj, gap, key, ranges } as unknown as SortSnapshot;
}

/** Является ли кадр снимком сортировки с допустимыми полями. */
export function isSortFrame(frame: Frame): boolean {
  if (frame.kind !== KIND_SORT) return false;
  try {
    asSortSnapshot(frame.data);
    return true;
  } catch {
    return false;
  }
}

/**
 * Ветка kind="tree": бинарное дерево (обходы in/pre/post-order).
 * Полная топология redraw целиком (кадр самодостаточен), вложенная
 * структура: узел — ячейка со значением, потомки — ветки L и R ниже,
 * отступ растёт с глубиной. Подсветки: активный узел (activeId) —
 * класс active, посещённые — класс visited с номером позиции.
 */
function asTreeSnapshot(data: unknown): TreeSnapshot {
  if (data === null || typeof data !== "object") {
    throw new Error("снимок не объект");
  }
  const obj = data as Record<string, unknown>;
  if (typeof obj.order !== "string") {
    throw new Error("снимок дерева: поле order должно быть строкой");
  }
  if (!Array.isArray(obj.nodes)) {
    throw new Error("снимок дерева: поле nodes должно быть массивом");
  }
  const byId = new Map<string, unknown>();
  for (const n of obj.nodes) {
    if (n === null || typeof n !== "object") {
      throw new Error("снимок дерева: каждый узел — объект {id, value, leftId, rightId}");
    }
    const node = n as Record<string, unknown>;
    for (const field of ["id", "leftId", "rightId"] as const) {
      if (typeof node[field] !== "string") {
        throw new Error(`снимок дерева: поле ${field} узла должно быть строкой`);
      }
    }
    if (typeof node.value !== "number" || !Number.isInteger(node.value)) {
      throw new Error("снимок дерева: поле value узла должно быть целым числом");
    }
    if (byId.has(node.id as string)) {
      throw new Error("снимок дерева: идентификаторы узлов должны быть уникальными");
    }
    byId.set(node.id as string, n);
  }
  for (const n of obj.nodes as TreeNodeData[]) {
    for (const link of [n.leftId, n.rightId]) {
      if (link !== "" && !byId.has(link)) {
        throw new Error(
          `снимок дерева: узел ${n.id} ссылается на отсутствующего потомка ${link}`,
        );
      }
    }
  }
  if (typeof obj.rootId !== "string") {
    throw new Error("снимок дерева: поле rootId должно быть строкой");
  }
  if (typeof obj.activeId !== "string") {
    throw new Error("снимок дерева: поле activeId должно быть строкой");
  }
  if (obj.activeId !== "" && !byId.has(obj.activeId)) {
    throw new Error(
      "снимок дерева: activeId должен быть идентификатором узла дерева",
    );
  }
  if (!Array.isArray(obj.visited)) {
    throw new Error("снимок дерева: поле visited должно быть массивом");
  }
  for (const v of obj.visited) {
    if (v === null || typeof v !== "object") {
      throw new Error("снимок дерева: посещение должно быть объектом {id, position}");
    }
    const visit = v as Record<string, unknown>;
    if (typeof visit.id !== "string" || !byId.has(visit.id)) {
      throw new Error(
        "снимок дерева: поле id посещения должно быть идентификатором узла дерева",
      );
    }
    if (
      typeof visit.position !== "number" ||
      !Number.isInteger(visit.position) ||
      visit.position < 1
    ) {
      throw new Error(
        "снимок дерева: поле position посещения должно быть положительным целым",
      );
    }
  }
  if (typeof obj.operation !== "string") {
    throw new Error("снимок дерева: поле operation должно быть строкой");
  }
  return obj as unknown as TreeSnapshot;
}

/** Является ли кадр снимком дерева с допустимыми полями. */
export function isTreeFrame(frame: Frame): boolean {
  if (frame.kind !== KIND_TREE) return false;
  try {
    asTreeSnapshot(frame.data);
    return true;
  } catch {
    return false;
  }
}

/**
 * Ветка kind="tree": рисует полное дерево одним вложенным блоком.
 * Корень сверху, ветки L и R с отступами — форма читается глазами;
 * активный узел — класс active, посещённые — visited с номером позиции
 * (метка data.position). Пустое дерево рисуется пустым маркером.
 */
function renderTree(snap: TreeSnapshot, containerEl: HTMLElement): void {
  const wrap = document.createElement("div");
  wrap.className = "tree";

  const caption = document.createElement("div");
  caption.className = "operation";
  caption.textContent =
    snap.operation === "" ? "Исходное состояние" : `Шаг: ${snap.operation}`;
  wrap.appendChild(caption);

  if (snap.rootId !== "") {
    const root = snap.nodes.find((n) => n.id === snap.rootId) as TreeNodeData;
    wrap.appendChild(treeNode(snap, root));
  } else {
    const empty = document.createElement("div");
    empty.className = "tree-node tree-empty";
    empty.textContent = "Дерево пустое";
    wrap.appendChild(empty);
  }

  containerEl.replaceChildren(wrap);
}

/** Построить DOM-блок одного узла с ветками L и R. */
function treeNode(snap: TreeSnapshot, node: TreeNodeData): HTMLElement {
  const block = document.createElement("div");
  block.className = "tree-node-block";

  const cell = document.createElement("div");
  cell.className = "cell tree-node";
  cell.dataset.id = node.id;
  cell.textContent = String(node.value);
  if (snap.activeId === node.id) cell.classList.add("active");
  const visit = snap.visited.find((v) => v.id === node.id);
  if (visit !== undefined) {
    cell.classList.add("visited");
    cell.dataset.visitedPosition = String(visit.position);
  }
  block.appendChild(cell);

  const children = document.createElement("div");
  children.className = "tree-children";
  appendTreeBranch(snap, node.leftId, "L", children);
  appendTreeBranch(snap, node.rightId, "R", children);
  block.appendChild(children);
  return block;
}

/** Одна ветка: метка L или R, вложенный узел или пустой маркер. */
function appendTreeBranch(
  snap: TreeSnapshot,
  childId: string,
  side: string,
  childrenEl: HTMLElement,
): void {
  const branch = document.createElement("div");
  branch.className = "tree-branch";
  branch.dataset.side = side;

  const label = document.createElement("div");
  label.className = "tree-branch-label";
  label.textContent = side;
  branch.appendChild(label);

  if (childId === "") {
    const empty = document.createElement("div");
    empty.className = "tree-node tree-empty";
    empty.textContent = "∅";
    branch.appendChild(empty);
  } else {
    const node = snap.nodes.find((n) => n.id === childId) as TreeNodeData;
    branch.appendChild(treeNode(snap, node));
  }
  childrenEl.appendChild(branch);
}

/**
 * Отрисовать кадр в DOM. Возвращает true, если кадр был отрисован;
 * false — если kind не поддерживается (тогда узлы остаются нетронутыми).
 */
export function renderFrame(
  frame: Frame,
  explanationEl: HTMLElement,
  arrayEl: HTMLElement,
): boolean {
  if (frame.kind === KIND_TREE) {
    const snap = asTreeSnapshot(frame.data);
    explanationEl.textContent = frame.explanation;
    renderTree(snap, arrayEl);
    return true;
  }
  if (frame.kind === KIND_STRUCTURE) {
    const snap = asStructureSnapshot(frame.data);
    explanationEl.textContent = frame.explanation;
    renderStructure(snap, arrayEl);
    return true;
  }
  if (frame.kind === KIND_HASH_TABLE) {
    const snap = asHashTableSnapshot(frame.data);
    explanationEl.textContent = frame.explanation;
    renderHashTable(snap, arrayEl);
    return true;
  }
  if (frame.kind === KIND_SORT) {
    const snap = asSortSnapshot(frame.data);
    explanationEl.textContent = frame.explanation;
    renderSort(snap, arrayEl);
    return true;
  }
  if (frame.kind !== KIND_ARRAY) return false;
  const snap = asArraySnapshot(frame.data);

  explanationEl.textContent = frame.explanation;

  // Пересоздаём массив целиком: кадр самодостаточен, состояние в DOM
  // не накапливаем. Дёшево — до сотен элементов на одной анимации.
  arrayEl.replaceChildren();
  for (let i = 0; i < snap.values.length; i++) {
    const cell = document.createElement("div");
    cell.className = "cell";
    cell.dataset.index = String(i);
    cell.textContent = String(snap.values[i]);

    if (i < snap.low || i > snap.high) cell.classList.add("out");
    if (i === snap.mid) cell.classList.add("mid");
    if (i === snap.foundIndex) cell.classList.add("found");
    if (snap.low !== -1 && snap.high !== -1 && snap.low <= i && i <= snap.high) {
      cell.classList.add("in-range");
    }

    arrayEl.appendChild(cell);
  }
  return true;
}

/**
 * Ветка kind="structure": ряд ячеек — сколько задано capacity, подсветка
 * затронутой ячейки ролью, метки указателей (top у стека, head/tail у
 * очереди). Переключение по snap.structure: "stack", "queue" и "list"
 * рисуются одинаковыми ячейками — у фиксированных структур ряд на всю
 * ёмкость (пустые видны), у списка capacity = len(values), рисуются только
 * заполненные ячейки. У списка при удалении к ряду добавляется одна
 * временная пустая ячейка-надгробие (см. ниже). "linked-list" рисуется
 * своей разновидностью (см. ниже): узлы со стрелками next, разрывы между
 * узлами, указатель head. Новые структуры — новые варианты внутри ветки.
 */
function renderStructure(snap: StructureSnapshot, containerEl: HTMLElement): void {
  const wrap = document.createElement("div");
  wrap.className = "structure";

  const caption = document.createElement("div");
  caption.className = "operation";
  caption.textContent = snap.operation === "" ? "Исходное состояние" : `Операция: ${snap.operation}`;
  wrap.appendChild(caption);

  const row = document.createElement("div");
  if (snap.structure === "linked-list") {
    // Односвязный список: память не непрерывна — узлы со стрелками next
    // и разрывами между ними, никаких адресов.
    row.className = "structure-row linked-list-row";
    renderLinkedListRow(snap, row);
    wrap.appendChild(row);
    containerEl.replaceChildren(wrap);
    return;
  }
  row.className = "structure-row";

  const roleByIndex = new Map(snap.highlights.map((h) => [h.index, h.role]));
  for (let i = 0; i < snap.capacity; i++) {
    const cell = document.createElement("div");
    cell.className = "cell";
    cell.dataset.index = String(i);
    if (i < snap.values.length) {
      cell.textContent = String(snap.values[i]);
      cell.classList.add("filled");
    } else {
      cell.classList.add("empty");
    }
    const role = roleByIndex.get(i);
    if (role !== undefined) cell.classList.add(role); // "added" / "removed" / "shifted"
    if (i === snap.top) cell.classList.add("top");

    // Указатели очереди: метки head/tail под их ячейками (могут совпасть).
    const pointers: string[] = [];
    if (snap.head !== undefined && snap.head !== -1 && i === snap.head) pointers.push("head");
    if (snap.tail !== undefined && snap.tail !== -1 && i === snap.tail) pointers.push("tail");
    if (pointers.length > 0) cell.dataset.pointers = pointers.join(" · ");

    row.appendChild(cell);
  }

  // Временная ячейка-надгробие при удалении из списка: освободившийся индекс
  // находится за концом нового ряда (index === values.length = capacity), так
  // что без этой ячейки роль removed осталась бы невидимой. Рисуем один
  // пустой маркер операции — он не входит в values/capacity снимка и не
  // расширяет capacity; у stack/queue надгробий нет — их освободившиеся
  // ячейки всегда попадают в фиксированный ряд.
  if (snap.structure === "list") {
    for (const h of snap.highlights) {
      if (h.role !== "removed" || h.index !== snap.values.length) continue;
      const ghost = document.createElement("div");
      ghost.className = "cell removed";
      ghost.dataset.index = String(h.index);
      row.appendChild(ghost);
    }
  }

  wrap.appendChild(row);

  containerEl.replaceChildren(wrap);
}

/**
 * Ряд односвязного списка: узлы со стрелкой next между соседями и разрывами
 * (память не непрерывна — это не ряд ячеек слайса); ни адресов памяти, ни
 * скрытой ёмкости — рисуются только существующие узлы (capacity =
 * values.length). Свои роли: visited (обход), relinked (переключение next),
 * found (найдено); added/removed работают как у остальных структур.
 * При удалении рисуется ячейка-надгробие на освободившейся логической
 * позиции: соседние с ней стрелки прерывисты — цепочка идёт мимо узла.
 * Указатели: метка head под узлом 0, tail — под последним узлом.
 */
function renderLinkedListRow(snap: StructureSnapshot, row: HTMLElement): void {
  const roleByIndex = new Map(snap.highlights.map((h) => [h.index, h.role]));
  const ghosts = new Set(
    snap.highlights.filter((h) => h.role === "removed").map((h) => h.index),
  );

  // Ячейка-надгробие на позиции i: узел удалён, его бывшая позиция
  // находится между новым узлами i-1 и i (в конце ряда — был хвост,
  // в начале — была голова).
  const items: Array<{ kind: "ghost" | "node"; index: number }> = [];
  for (let i = 0; i <= snap.values.length; i++) {
    if (ghosts.has(i)) items.push({ kind: "ghost", index: i });
    if (i < snap.values.length) items.push({ kind: "node", index: i });
  }

  items.forEach((item, k) => {
    if (k > 0) {
      const arrow = document.createElement("div");
      arrow.className = "arrow";
      // Стрелка к надгробию или от него: цепочка идёт мимо удалённого узла.
      if (items[k - 1].kind === "ghost" || item.kind === "ghost") {
        arrow.classList.add("skip");
      }
      row.appendChild(arrow);
    }

    const el = document.createElement("div");
    el.dataset.index = String(item.index);
    if (item.kind === "ghost") {
      el.className = "cell removed";
    } else {
      el.className = "cell filled";
      el.textContent = String(snap.values[item.index]);
      const role = roleByIndex.get(item.index);
      if (role !== undefined) el.classList.add(role); // added/visited/relinked/found/removed
      const pointers: string[] = [];
      if (snap.head !== undefined && snap.head !== -1 && item.index === snap.head) {
        pointers.push("head");
      }
      if (snap.tail !== undefined && snap.tail !== -1 && item.index === snap.tail) {
        pointers.push("tail");
      }
      if (pointers.length > 0) el.dataset.pointers = pointers.join(" · ");
    }
    row.appendChild(el);
  });
}

/**
 * Ветка kind="hash-table": каждый ряд — одна корзина; внутри ряда метка
 * индекса корзины и цепочка записей «ключ:значение» со стрелками next
 * (метод цепочек: столько записей, сколько ключей попали в корзину).
 * Пустая корзина рисуется пустым маркером — она сравнивается глазами
 * с соседями, как ряд ячеек у фиксированных структур. Подсветки:
 * выбранная хешированием корзина подчёркивается целиком (selected),
 * живые записи — своей ролью (visited/added/updated/found). Удалённые
 * записи в снимке не живут: они рисуются надгробиями из removed на своих
 * бывших позициях, соседние стрелки — прерывистые (цепочка идёт мимо).
 * Ряды не перетасовываются: индекс корзины — адрес статики, не позиция
 * обхода.
 */
function renderHashTable(snap: HashTableSnapshot, containerEl: HTMLElement): void {
  const wrap = document.createElement("div");
  wrap.className = "hash-table";

  const caption = document.createElement("div");
  caption.className = "operation";
  caption.textContent = snap.operation === "" ? "Исходное состояние" : `Операция: ${snap.operation}`;
  wrap.appendChild(caption);

  const roleByEntry = new Map(
    snap.highlights.map((h) => [`${h.bucket}:${h.position}`, h.role] as const),
  );

  // Надгробия по корзинам: бывшая позиция → удалённая пара.
  const tombstones = new Map<number, Map<number, HashEntryRemoval>>();
  for (const removal of snap.removed) {
    let byBucket = tombstones.get(removal.bucket);
    if (!byBucket) {
      byBucket = new Map();
      tombstones.set(removal.bucket, byBucket);
    }
    byBucket.set(removal.position, removal);
  }

  for (let i = 0; i < snap.bucketCount; i++) {
    const row = document.createElement("div");
    row.className = "hash-table-row";
    row.dataset.bucket = String(i);
    if (i === snap.highlightBucket) row.classList.add("selected");

    const label = document.createElement("div");
    label.className = "bucket-label";
    label.textContent = String(i);
    row.appendChild(label);

    const chain = snap.buckets[i];
    const ghosts = tombstones.get(i);

    // Элементы ряда через сцепку позиций: надгробия встают на свои бывшие
    // позиции, живые записи — по позициям снимка (после удаления они
    // сдвинуты, поэтому соседом надгробия оказывается вставший рядом).
    type RowItem =
      | { kind: "ghost"; removal: HashEntryRemoval }
      | { kind: "empty" }
      | { kind: "entry"; entry: HashEntry; position: number };
    const items: RowItem[] = [];
    if (chain.length === 0 && (ghosts === undefined || ghosts.size === 0)) {
      items.push({ kind: "empty" });
    }
    for (let position = 0; position <= chain.length; position++) {
      const ghost = ghosts?.get(position);
      if (ghost !== undefined) items.push({ kind: "ghost", removal: ghost });
      if (position < chain.length) {
        items.push({ kind: "entry", entry: chain[position], position });
      }
    }

    items.forEach((item, k) => {
      if (k > 0) {
        const arrow = document.createElement("div");
        arrow.className = "arrow";
        // Стрелка к надгробию или от него: цепочка идёт мимо удалённой записи.
        if (items[k - 1].kind === "ghost" || item.kind === "ghost") {
          arrow.classList.add("skip");
        }
        row.appendChild(arrow);
      }
      if (item.kind === "ghost") {
        // Надгробие не входит в buckets: data-position — бывшая позиция
        // удалённой записи, живые номера позиций ряда оно не считает.
        const cell = document.createElement("div");
        cell.className = "hash-entry removed";
        cell.textContent = `${item.removal.key}:${item.removal.value}`;
        cell.dataset.bucket = String(i);
        cell.dataset.position = String(item.removal.position);
        row.appendChild(cell);
        return;
      }
      if (item.kind === "empty") {
        const empty = document.createElement("div");
        empty.className = "hash-entry empty";
        row.appendChild(empty);
        return;
      }
      const cell = document.createElement("div");
      cell.className = "hash-entry filled";
      cell.textContent = `${item.entry.key}:${item.entry.value}`;
      cell.dataset.bucket = String(i);
      cell.dataset.position = String(item.position);
      const role = roleByEntry.get(`${i}:${item.position}`);
      if (role !== undefined) cell.classList.add(role); // visited/added/updated/found
      row.appendChild(cell);
    });

    wrap.appendChild(row);
  }

  containerEl.replaceChildren(wrap);
}

/**
 * Ветка kind="sort": ряд ячеек всего массива (кадр самодостаточен —
 * пересоздаётся целиком, состояние не копится) с ролями подсветки:
 * "compare" — сравниваемая сейчас пара (у selection sort — обычно
 * проверяемый кандидат, а при сравнении, сменившем минимум, — прежний
 * минимум; у insertion sort — элемент префикса, сравниваемый с ключом;
 * у quicksort — текущий элемент, сравниваемый с pivot),
 * "swap" — обмениваемая, "min" — текущий минимум прохода selection sort
 * (в том числе новый минимум сразу после сравнения), "shift" — позиция,
 * в которую сейчас сдвинулось значение (insertion sort), "insert" —
 * позиция, куда вставляется ключ (insertion sort), "write" — позиция,
 * в которую сейчас записано значение при слиянии (merge sort), "pivot" —
 * позиция pivot: выбранного для partition и занимаемой финальной позиции
 * (quicksort), "boundary" — позиция границы i, конца блока строго меньших
 * значений partition (quicksort). Активные диапазоны (snap.ranges)
 * размечают ряд: у merge sort ячейки левой половины получают класс
 * in-left, правой — in-right; кадры записи диапазоны не несут (позиция
 * записи — приёмник слияния, а не половина источника), поэтому разметки
 * половин на них нет. Диапазоны partition quicksort идут без границы
 * половин (mid): их ячейки [lo, hi) получают фоновую разметку
 * in-partition, а границы шага несут роли pivot/boundary; кадр размещения
 * pivot диапазона не несёт — завершённая partition разметки не рисует.
 * Временное состояние
 * вставки insertion sort (пока ключ удерживается вне массива) рисуется
 * из полей снимка — видимое состояние задают gap и key вместе: ячейка
 * на позиции gap — пустая «дырка» (значение массива под ней — исходное
 * значение ключа до сдвигов или устаревшая копия после них — не
 * рисуется), над рядом — чип со значением удерживаемого
 * ключа (snap.key). Все сортировки
 * модуля (bubble sort и далее) идут в эту же ветку по снимку kind="sort".
 */
function renderSort(snap: SortSnapshot, containerEl: HTMLElement): void {
  const wrap = document.createElement("div");
  wrap.className = "sort";

  const caption = document.createElement("div");
  caption.className = "operation";
  caption.textContent = snap.operation === "" ? "Исходное состояние" : `Шаг: ${snap.operation}`;
  wrap.appendChild(caption);

  // Удерживаемый ключ (insertion sort): чип над рядом, пока проход ищет
  // ему место; после вставки (gap = -1) чипа нет — ключ в массиве.
  if (snap.gap >= 0) {
    const keyChip = document.createElement("div");
    keyChip.className = "sort-key";
    keyChip.textContent = `Ключ вне массива: ${snap.key}`;
    wrap.appendChild(keyChip);
  }

  const row = document.createElement("div");
  row.className = "array";
  const roleByIndex = new Map(snap.highlights.map((h) => [h.index, h.role]));
  // Активные диапазоны merge sort: левая/правая половины слияния получают
  // фоновую разметку — видно, какие части массива сейчас в работе.
  // Диапазоны без границы половин — partition quicksort: их ячейки
  // получают разметку in-partition, границы шага несут роли pivot/boundary.
  const sideByIndex = new Map<number, "in-left" | "in-right">();
  const partitionIndices = new Set<number>();
  for (const range of snap.ranges) {
    if (range.mid === undefined) {
      for (let i = range.lo; i < range.hi; i++) partitionIndices.add(i);
      continue;
    }
    for (let i = range.lo; i < range.hi; i++) {
      sideByIndex.set(i, i < range.mid ? "in-left" : "in-right");
    }
  }
  for (let i = 0; i < snap.values.length; i++) {
    const cell = document.createElement("div");
    cell.className = "cell";
    cell.dataset.index = String(i);
    if (i === snap.gap) {
      cell.classList.add("gap"); // «дырка»: значение из неё уехало, здесь ждёт ключ
    } else {
      cell.classList.add("filled");
      cell.textContent = String(snap.values[i]);
      const role = roleByIndex.get(i);
      if (role !== undefined) cell.classList.add(role); // "compare" / "swap" / "min" / "shift" / "insert" / "write"
      const side = sideByIndex.get(i);
      if (side !== undefined) cell.classList.add(side); // активные половины слияния merge sort
      if (partitionIndices.has(i)) cell.classList.add("in-partition"); // активный диапазон partition quicksort
    }
    row.appendChild(cell);
  }
  wrap.appendChild(row);

  containerEl.replaceChildren(wrap);
}
