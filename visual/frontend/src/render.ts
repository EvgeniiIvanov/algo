// Тупой рендерер кадра в DOM (ADR-0002):
// кадр самодостаточен, состояния между кадрами не накапливаем.
// На вход Frame — на выход мутация двух DOM-узлов (пояснение + массив).
//
// Поддерживаем сейчас только kind="array" (бинарный поиск).
// Новые kind'ы — новая ветка без правок существующих.

import { KIND_ARRAY, type ArraySnapshot, type Frame } from "./types";

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

/**
 * Отрисовать кадр в DOM. Возвращает true, если кадр был отрисован;
 * false — если kind не поддерживается (тогда узлы остаются нетронутыми).
 */
export function renderFrame(
  frame: Frame,
  explanationEl: HTMLElement,
  arrayEl: HTMLElement,
): boolean {
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