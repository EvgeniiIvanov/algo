// Типы кадров — зеркало visual/frame.Frame на стороне бэкенда.
// Источник истины: visual/frame/frame.go (ADR-0002). Меняется вместе с ним.

/** Известные `kind` кадров. Другие принимаем как `unknown`, не падаем. */
export const KIND_ARRAY = "array" as const;
export const KIND_STRUCTURE = "structure" as const;
export type ArrayKind = typeof KIND_ARRAY;
export type StructureKind = typeof KIND_STRUCTURE;
export type Kind = ArrayKind | StructureKind | (string & { readonly __brand?: unique symbol });

/** Снимок массива — `Data` кадра для бинарного поиска. */
export interface ArraySnapshot {
  values: number[];
  low: number;
  high: number;
  /** -1 — на этом шаге нет сравниваемой середины. */
  mid: number;
  /** -1 — значение ещё не найдено / не найдено вообще. */
  foundIndex: number;
}

/** Подсветка одной позиции с ролью — что с ней случилось на шаге. */
export interface StructureHighlight {
  index: number;
  /** Роль: "added" (ячейка записана) / "removed" (ячейка освобождена). */
  role: string;
}

/**
 * Снимок линейной структуры — `Data` кадра kind="structure"
 * (зеркало render.StructureSnapshot). Один снимок на все структуры модуля;
 * переключение рендера — по полю structure.
 */
export interface StructureSnapshot {
  /** Идентичность структуры: "stack" (позже queue/list/linked-list). */
  structure: string;
  /** Ёмкость: ряд ячеек рисуется целиком, пустые видны. */
  capacity: number;
  /** Содержимое от дна к вершине. */
  values: number[];
  /** Подсветки затронутых позиций с ролью. */
  highlights: StructureHighlight[];
  /** Указатель top — индекс вершины; -1 — структура пуста. */
  top: number;
  /** Имя операции шага ("push 5", "pop"); пусто — стартовый кадр. */
  operation: string;
}

/** Кадр визуализации. Самодостаточен (ADR-0002). */
export interface Frame {
  /** Порядковый номер шага, начиная с 1. */
  step: number;
  kind: Kind;
  /** Пояснение на русском. */
  explanation: string;
  /** Снимок структуры. У каждого рендерера свой тип. */
  data: ArraySnapshot | StructureSnapshot | { readonly [key: string]: unknown };
}
