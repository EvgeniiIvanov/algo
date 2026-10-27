// Типы кадров — зеркало visual/frame.Frame на стороне бэкенда.
// Источник истины: visual/frame/frame.go (ADR-0002). Меняется вместе с ним.

/** Известные `kind` кадров. Другие принимаем как `unknown`, не падаем. */
export const KIND_ARRAY = "array" as const;
export const KIND_STRUCTURE = "structure" as const;
export const KIND_HASH_TABLE = "hash-table" as const;
export const KIND_SORT = "sort" as const;
export const KIND_TREE = "tree" as const;
export type ArrayKind = typeof KIND_ARRAY;
export type StructureKind = typeof KIND_STRUCTURE;
export type HashTableKind = typeof KIND_HASH_TABLE;
export type SortKind = typeof KIND_SORT;
export type TreeKind = typeof KIND_TREE;
export type Kind =
  | ArrayKind
  | StructureKind
  | HashTableKind
  | SortKind
  | TreeKind
  | (string & { readonly __brand?: unique symbol });

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
   * Роль: "added" (узел/ячейка добавлены) / "removed" (узел/ячейка
   * освобождены) / "shifted" (значение сдвинулось из соседней ячейки) /
   * "visited" (узел посещён обходом — односвязный список) /
   * "relinked" (связь next узла переключена — односвязный список) /
   * "found" (узел с искомым значением найден — односвязный список).
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

/**
 * Подсветка одной записи цепочки хеш-таблицы с ролью — что с ней случилось
 * на шаге кадра kind="hash-table". Роли: "visited" (запись посещена обходом
 * цепочки) / "added" (запись вставлена во главу цепочки) /
 * "updated" (значение существующего ключа переписано) /
 * "found" (запись с искомым ключом найдена). Удалённая запись в подсветках
 * не участвует: в снимке её уже нет — она несётся надгробием в `removed`
 * (см. HashEntryRemoval), иначе роль указывала бы на соседа, вставшего
 * на её место.
 */
export interface HashEntryHighlight {
  /** Корзина, в которой лежит запись. */
  bucket: number;
  /** Позиция записи в цепочке (0 — голова цепочки). */
  position: number;
  role: string;
}

/**
 * Запись, удалённая на этом шаге: надгробие на бывшей позиции цепочки.
 * Живые записи снимка (buckets) её уже не содержат — надгробие несёт ключ
 * и значение, чтобы учащийся видел, что именно удалилось и где стояло;
 * renderer рисует его между соседями по бывшей позиции.
 */
export interface HashEntryRemoval {
  /** Корзина, из которой удалили запись. */
  bucket: number;
  /** Бывшая позиция записи в цепочке (0 — голова). */
  position: number;
  /** Ключ удалённой записи. */
  key: number;
  /** Удалённое значение. */
  value: number;
}

/** Одна запись цепочки корзины хеш-таблицы. */
export interface HashEntry {
  key: number;
  value: number;
}

/**
 * Снимок хеш-таблицы — `Data` кадра kind="hash-table" (зеркало
 * render.HashTableSnapshot). Самодостаточен (ADR-0002): все корзины
 * с их цепочками идут в каждом кадре — от головы записи.
 */
export interface HashTableSnapshot {
  /** Число корзин, заданное при создании таблицы. */
  bucketCount: number;
  /** Содержимое каждой корзины: цепочка записей от головы. */
  buckets: HashEntry[][];
  /** Корзина, выбранная хешированием на шаге; -1 — шаг без выбора. */
  highlightBucket: number;
  /** Подсветки затронутых живых записей с ролью. */
  highlights: HashEntryHighlight[];
  /** Надгробия шага: удалённые записи на бывших позициях; пустой массив — шагов удаления нет. */
  removed: HashEntryRemoval[];
  /** Число записей во всей таблице. */
  entryCount: number;
  /** Имя операции шага ("put 6 20", "get 9"); пусто — стартовый кадр. */
  operation: string;
}

/**
 * Подсветка одной позиции массива сортировки с ролью — что с ней случилось
 * на шаге кадра kind="sort". Роли: "compare" (позиция сравниваемой сейчас
 * пары; у selection sort — обычно проверяемый кандидат, а при сравнении,
 * сменившем минимум, — прежний минимум; у insertion sort — элемент
 * отсортированного префикса, сравниваемый с ключом; у merge sort — голова
 * одной из сравниваемых половин; у quicksort — текущий элемент,
 * сравниваемый с pivot) / "swap" (позиция обмениваемой сейчас
 * пары) / "min" (текущий минимум прохода selection sort — в том числе
 * новый минимум сразу после сравнения) / "shift" (позиция, в которую
 * сейчас сдвинулось значение — приёмник сдвига; источник становится
 * «дыркой» и помечается полем gap, не ролью, insertion sort) / "insert"
 * (позиция, куда вставляется ключ — insertion sort) / "write" (позиция,
 * в которую сейчас записано значение при слиянии — merge sort) /
 * "pivot" (позиция pivot: выбранного для partition и занимаемой финальной
 * позиции — quicksort) / "boundary" (позиция границы i: конец блока
 * строго меньших значений в partition — quicksort).
 */
export interface SortHighlight {
  index: number;
  role: string;
}

/**
 * Активный диапазон шага сортировки: [lo, hi) сейчас используется
 * алгоритмом. Merge sort несёт mid — границу половин слияния; quicksort
 * несёт диапазон partition [lo, hi) без mid (у partition нет границы
 * половин). Другие сортировки диапазонов не несут.
 */
export interface SortRange {
  lo: number;
  /** Граница половин слияния: левая часть [lo, mid), правая [mid, hi) — только merge sort. */
  mid?: number;
  hi: number;
}

/**
 * Снимок массива сортировки — `Data` кадра kind="sort" (зеркало
 * render.SortSnapshot). Самодостаточен (ADR-0002): весь массив идёт
 * в каждом кадре, клиент ничего не докапливает.
 */
export interface SortSnapshot {
  /**
   * Весь массив на шаге (копия бэкенда). При gap >= 0 значение
   * values[gap] — исходное значение ключа до сдвигов, а после сдвигов —
   * устаревшая копия сдвинутого значения: в обоих случаях рендер его
   * скрывает; видимое состояние вставки задают gap и key вместе,
   * а не values в одиночку.
   */
  values: number[];
  /** Подсветки сравниваемых/обмениваемых/сдвигаемых позиций, минимума и вставки. */
  highlights: SortHighlight[];
  /** Имя шага ("сравнение 0 1", "обмен 1 2", "сдвиг 0 → 1"); пусто — стартовый кадр. */
  operation: string;
  /**
   * Индекс «дырки» — пустой позиции, ожидающей удерживаемый ключ
   * (insertion sort, пока ключ вне массива); -1 — дырки на шаге нет
   * (стартовый кадр, bubble/selection, шаг после вставки).
   */
  gap: number;
  /**
   * Значение удерживаемого ключа (insertion sort); осмыслен только
   * при gap >= 0, иначе 0 и рендером игнорируется.
   */
  key: number;
  /**
   * Активные диапазоны шага (merge sort: сливаемый отрезок [lo, hi)
   * с границей половин mid; quicksort: partition [lo, hi) без mid);
   * у других сортировок — пустой массив.
   * Поле обратно совместимо: старые снимки без ranges читаются как [].
   */
  ranges: SortRange[];
}

/**
 * Снимок бинарного дерева — `Data` кадра kind="tree" (зеркало
 * render.TreeSnapshot). Самодостаточен (ADR-0002): полная топология узлов
 * в каждом кадре, преподаваемое состояние — активный узел и посещённая
 * последовательность. Идентификаторы — путь узла от корня
 * (["root", "L", "LR"...]), не зависят от значения: узлы с одинаковыми
 * значениями различимы.
 */
export interface TreeSnapshot {
  /** Порядок обхода: "in-order" / "pre-order" / "post-order". */
  order: string;
  /** Все узлы дерева: полный снимок топологии (пустой массив — пустое дерево). */
  nodes: TreeNodeData[];
  /** Идентификатор корня; "" — пустое дерево. */
  rootId: string;
  /** Посещаемый узел шага; "" — стартовый кадр. */
  activeId: string;
  /** Уже посещённые узлы с порядковыми номерами, с 1. */
  visited: TreeVisit[];
  /** Имя шага ("старт", "посещение LR"); пусто — стартовый кадр. */
  operation: string;
}

/** Один узел дерева в снимке: значение и связи соседей по ID ("" — потомка нет). */
export interface TreeNodeData {
  id: string;
  value: number;
  leftId: string;
  rightId: string;
}

/** Уже посещённый узел с его номером в порядке обхода (с 1). */
export interface TreeVisit {
  id: string;
  position: number;
}

/** Кадр визуализации. Самодостаточен (ADR-0002). */
export interface Frame {
  /** Порядковый номер шага, начиная с 1. */
  step: number;
  kind: Kind;
  /** Пояснение на русском. */
  explanation: string;
  /** Снимок структуры. У каждого рендерера свой тип. */
  data:
    | ArraySnapshot
    | StructureSnapshot
    | HashTableSnapshot
    | SortSnapshot
    | TreeSnapshot
    | { readonly [key: string]: unknown };
}
