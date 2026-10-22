// Тупой рендерер кадра в DOM (ADR-0002):
// кадр самодостаточен, состояния между кадрами не накапливаем.
// На вход Frame — на выход мутация двух DOM-узлов (пояснение + контейнер).
//
// Поддерживаем kind="array" (бинарный поиск), kind="structure"
// (стек "stack", очередь "queue", список "list", односвязный список
// "linked-list" — узлы со стрелками next), kind="hash-table"
// (корзины с цепочками записей) и kind="sort" (простые сортировки,
// bubble sort и далее). Новые kind'ы — новая ветка без правок
// существующих.

import {
  KIND_ARRAY,
  KIND_HASH_TABLE,
  KIND_SORT,
  KIND_STRUCTURE,
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
  // Активные диапазоны merge sort: старый снимок без поля читается как
  // пустой список; кривые значения — ошибка.
  let ranges: SortRange[] = [];
  if (obj.ranges !== undefined) {
    if (!Array.isArray(obj.ranges)) {
      throw new Error("снимок сортировки: поле ranges должно быть массивом активных диапазонов");
    }
    ranges = (obj.ranges as unknown[]).map((r) => {
      if (r === null || typeof r !== "object") {
        throw new Error("снимок сортировки: активный диапазон должен быть объектом {lo, mid, hi}");
      }
      const rg = r as Record<string, unknown>;
      for (const field of ["lo", "mid", "hi"] as const) {
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
      if (range.mid <= range.lo || range.mid >= range.hi) {
        throw new Error(
          "снимок сортировки: граница половин mid должна делить активный диапазон на непустые части",
        );
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
 * Отрисовать кадр в DOM. Возвращает true, если кадр был отрисован;
 * false — если kind не поддерживается (тогда узлы остаются нетронутыми).
 */
export function renderFrame(
  frame: Frame,
  explanationEl: HTMLElement,
  arrayEl: HTMLElement,
): boolean {
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
 * минимум; у insertion sort — элемент префикса, сравниваемый с ключом),
 * "swap" — обмениваемая, "min" — текущий минимум прохода selection sort
 * (в том числе новый минимум сразу после сравнения), "shift" — позиция,
 * в которую сейчас сдвинулось значение (insertion sort), "insert" —
 * позиция, куда вставляется ключ (insertion sort), "write" — позиция,
 * в которую сейчас записано значение при слиянии (merge sort). Активные
 * диапазоны merge sort (snap.ranges) размечают ряд: ячейки левой половины
 * получают класс in-left, правой — in-right; кадры записи диапазоны не несут
 * (позиция записи — приёмник слияния, а не половина источника), поэтому
 * разметки половин на них нет. Временное состояние
 * вставки insertion sort (пока ключ удерживается вне массива) рисуется
 * из полей снимка — видимое состояние задают gap и key вместе: ячейка
 * на позиции gap — пустая «дырка» (значение массива под ней — исходное
 * значение ключа до сдвигов или устаревшая копия после них — не
 * рисуется), над рядом — чип со значением удерживаемого
 * ключа (snap.key). Все простые сортировки
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
  const sideByIndex = new Map<number, "in-left" | "in-right">();
  for (const range of snap.ranges) {
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
    }
    row.appendChild(cell);
  }
  wrap.appendChild(row);

  containerEl.replaceChildren(wrap);
}
