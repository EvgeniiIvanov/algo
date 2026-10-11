// Тупой рендерер кадра в DOM (ADR-0002):
// кадр самодостаточен, состояния между кадрами не накапливаем.
// На вход Frame — на выход мутация двух DOM-узлов (пояснение + контейнер).
//
// Поддерживаем kind="array" (бинарный поиск) и kind="structure" (стек).
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
    obj.capacity < 1
  ) {
    throw new Error("снимок структуры: поле capacity должно быть положительным целым");
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
  if (typeof obj.operation !== "string") {
    throw new Error("снимок структуры: поле operation должно быть строкой");
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
 * Ветка kind="structure": ряд ячеек на всю ёмкость (пустые видны),
 * подсветка затронутой ячейки ролью, метка указателя top.
 * Переключение по snap.structure: пока один вариант — "stack" (непрерывные
 * ячейки); новые структуры — новые варианты внутри этой ветки.
 */
function renderStructure(snap: StructureSnapshot, containerEl: HTMLElement): void {
  const wrap = document.createElement("div");
  wrap.className = "structure";

  const caption = document.createElement("div");
  caption.className = "operation";
  caption.textContent = snap.operation === "" ? "Исходное состояние" : `Операция: ${snap.operation}`;
  wrap.appendChild(caption);

  const row = document.createElement("div");
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
    if (role !== undefined) cell.classList.add(role); // "added" / "removed"
    if (i === snap.top) cell.classList.add("top");

    row.appendChild(cell);
  }
  wrap.appendChild(row);

  containerEl.replaceChildren(wrap);
}
