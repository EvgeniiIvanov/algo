// Типы кадров — зеркало visual/frame.Frame на стороне бэкенда.
// Источник истины: visual/frame/frame.go (ADR-0002). Меняется вместе с ним.

/** Известные `kind` кадров. Другие принимаем как `unknown`, не падаем. */
export const KIND_ARRAY = "array" as const;
export type ArrayKind = typeof KIND_ARRAY;
export type Kind = ArrayKind | (string & { readonly __brand?: unique symbol });

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

/** Кадр визуализации. Самодостаточен (ADR-0002). */
export interface Frame {
  /** Порядковый номер шага, начиная с 1. */
  step: number;
  kind: Kind;
  /** Пояснение на русском. */
  explanation: string;
  /** Снимок структуры. У каждого рендерера свой тип — здесь только массив. */
  data: ArraySnapshot | { readonly [key: string]: unknown };
}