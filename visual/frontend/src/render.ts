// Тупой рендерер кадра в DOM (ADR-0002):
// кадр самодостаточен, состояния между кадрами не накапливаем.
// На вход Frame — на выход мутация двух DOM-узлов (пояснение + контейнер).
//
// Поддерживаем kind="array" (бинарный поиск) и kind="structure"
// (стек "stack", очередь "queue", список "list", односвязный список
// "linked-list" — узлы со стрелками next).
// Новые kind'ы — новая ветка без правок существующих.

import {
  KIND_ARRAY,
  KIND_STRUCTURE,
  type ArraySnapshot,
  type Frame,
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
