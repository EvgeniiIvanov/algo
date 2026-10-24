# Простые сортировки: bubble, selection, insertion

Реализация: `algorithms/sort/` (`BubbleSort`, `SelectionSort`, `InsertionSort` и их
`*WithEmitter`-варианты). Все три — по возрастанию, на месте, для слайса `[]int`.

## Bubble sort

Реализованный вариант: проходы слева направо сравнивают соседние элементы;
каждая строго убывающая пара (`nums[j] > nums[j+1]`) меняется местами, и максимум
остатка «всплывает» в конец. Равные соседи (строгое `>`) не обмениваются, поэтому
сортировка **устойчива**. Отсортированные проходы сокращают необработанный
диапазон на `i` справа.

```mermaid
flowchart TD
    BubStart([BubbleSort nums]) --> BubInitI["i = 0"]
    BubInitI --> BubOuter{"i < n - 1?"}
    BubOuter -- "нет" --> BubEnd([конец: массив отсортирован])
    BubOuter -- "да" --> BubInitJ["swapped = false<br/>j = 0"]
    BubInitJ --> BubInner{"j < n - 1 - i?"}
    BubInner -- "нет" --> BubSwapped{"swapped?"}
    BubSwapped -- "нет: проход без обменов" --> BubEnd
    BubSwapped -- "да" --> BubIncI["i = i + 1"]
    BubIncI --> BubOuter
    BubInner -- "да" --> BubCompare["Compare: nums[j] и nums[j+1]"]
    BubCompare --> BubPair{"nums[j] > nums[j+1]?"}
    BubPair -- "да: строго убывающая пара" --> BubSwap["Swap: обмен пары<br/>swapped = true"]
    BubPair -- "нет: пара упорядочена (≤)" --> BubIncJ["j = j + 1"]
    BubSwap --> BubIncJ
    BubIncJ --> BubInner
```

Сложность: худший и средний случай O(n²), лучший — O(n) на уже упорядоченном
входе за счёт досрочного выхода: проход без обменов завершает работу.

## Selection sort

Реализованный вариант — стандартный **неустойчивый**: на каждом проходе ищется
минимум неотсортированной части (кандидаты от `i+1`, минимум смещает строгий
`<` — равные выбор не смещают) и меняется местами с первым элементом части.
Обмен дальними позициями может менять порядок равных — отсюда неустойчивость.

```mermaid
flowchart TD
    SelStart([SelectionSort nums]) --> SelInitI["i = 0"]
    SelInitI --> SelOuter{"i < n - 1?"}
    SelOuter -- "нет" --> SelEnd([конец])
    SelOuter -- "да" --> SelInitMin["minIdx = i<br/>j = i + 1"]
    SelInitMin --> SelInner{"j < n?"}
    SelInner -- "да" --> SelCompare{"nums[j] < nums[minIdx]?"}
    SelCompare -- "да: новый минимум" --> SelUpdateMin["minIdx = j"]
    SelCompare -- "нет: минимум остаётся" --> SelIncJ["j = j + 1"]
    SelUpdateMin --> SelIncJ
    SelIncJ --> SelInner
    SelInner -- "нет: минимум найден" --> SelMinDiff{"minIdx ≠ i?"}
    SelMinDiff -- "да" --> SelSwap["Swap: nums[i] ↔ nums[minIdx]"]
    SelMinDiff -- "нет: минимум уже на месте, без обмена" --> SelIncI["i = i + 1"]
    SelSwap --> SelIncI
    SelIncI --> SelOuter
```

Сложность всегда O(n²) сравнений — даже на упорядоченном входе; обменов не
больше n−1.

## Insertion sort

Реализованный вариант — классический **устойчивый**: ключ `nums[i]` удерживается
вне массива, отсортированный префикс `[0..i-1]` сканируется **справа налево**;
только элементы **строго больше** ключа сдвигаются вправо, равные ключу
не сдвигаются — порядок равных сохраняется. Затем ключ записывается
в освободившуюся позицию («дырку»).

```mermaid
flowchart TD
    InsStart([InsertionSort nums]) --> InsInitI["i = 1"]
    InsInitI --> InsOuter{"i < n?"}
    InsOuter -- "нет" --> InsEnd([конец])
    InsOuter -- "да" --> InsInitKey["key = nums[i]<br/>j = i - 1"]
    InsInitKey --> InsBack{"j >= 0?"}
    InsBack -- "нет: префикс кончился" --> InsPlace["nums[j + 1] = key<br/>Write: вставка ключа"]
    InsBack -- "да" --> InsCompare{"Outcome: nums[j] и key"}
    InsCompare -- "nums[j] > key" --> InsShift["Shift: nums[j + 1] = nums[j]<br/>j = j - 1"]
    InsShift --> InsBack
    InsCompare -- "nums[j] == key: равный не сдвигается" --> InsPlace
    InsCompare -- "nums[j] < key" --> InsPlace
    InsPlace --> InsIncI["i = i + 1"]
    InsIncI --> InsOuter
```

Сложность: худший и средний O(n²), лучший — O(n) на упорядоченном входе
(каждый проход без сдвигов). Обучающее событие `InsertionEqual` (равный не
сдвигается) построено именно на устойчивом варианте — см.
`insertion_events.go`.

## Сравнение вариантов

| Свойство | Bubble | Selection | Insertion |
|----------|--------|-----------|-----------|
| Устойчивость | да | нет | да |
| Лучший случай | O(n) | O(n²) | O(n) |
| На месте | да | да | да |

События всех трёх сортировок (режим наблюдения, ADR-0001) описаны
в комментариях `*_events.go`; здесь приводится только алгоритмическая логика.
