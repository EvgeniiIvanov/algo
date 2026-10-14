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

/**
 * Подсветка одной позиции с ролью — что с ней случилось на шаге.
 * Роли: "added" (ячейка записана) / "removed" (ячейка освобождена) /
 * "shifted" (значение сдвинулось из соседней ячейки) /
 * "visited" (узел посещён обходом — односвязный список) /
 * "relinked" (связь next узла переключена — односвязный список) /
 * "found" (узел с искомым значением найден).
 */
export interface StructureHighlight {
  index: number;
  /**
   * Роль: "added" (ячейка записана) / "removed" (ячейка освобождена) /
   * "shifted" (значение сдвинулось из соседней ячейки).
   */
  role: string;
}

/**
 * Снимок линейной структуры — `Data` кадра kind="structure"
 * (зеркало render.StructureSnapshot). Один снимок на все структуры модуля;
 * переключение рендера — по полю structure.
 */
export interface StructureSnapshot {
  /** Идентичность структуры: "stack" / "queue" / "list" / "linked-list". */
  structure: string;
  /**
   * Сколько ячеек рисовать: у фиксированных структур (stack/queue) — заданная
   * ёмкость, весь ряд с пустыми ячейками; у растущего списка — текущее число
   * заполненных ячеек (совпадает с values.length, 0 — пустой список).
   */
  capacity: number;
  /** Содержимое в логическом порядке (стек: дно → вершина, очередь и список: голова → хвост). */
  values: number[];
  /** Подсветки затронутых позиций с ролью. */
  highlights: StructureHighlight[];
  /** Указатель top — индекс вершины стека; -1 — пуста или это очередь. */
  top: number;
  /**
   * Указатель head — индекс головы очереди; у односвязного списка — узел 0.
   * -1 — пуста или это стек.
   */
  head?: number;
  /** Указатель tail — индекс хвоста очереди/списка; -1 — пуста или это стек. */
  tail?: number;
  /** Имя операции шага ("push 5", "enqueue 3", "pop"); пусто — стартовый кадр. */
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
