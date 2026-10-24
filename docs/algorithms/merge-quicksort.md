# Merge sort и quicksort

Реализация: `algorithms/sort/` (`MergeSort`, `QuickSort` и их `*WithEmitter`
-варианты). Оба сортируют по возрастанию, на месте с точки зрения вызывающего
кода, для слайса `[]int` — но варианты алгоритмов и цены по памяти разные.

Реализованные варианты: merge sort — **top-down stable** (рекурсивное деление
диапазона пополам, устойчивое слияние через буфер), quicksort — **Lomuto**
(pivot — последний элемент диапазона, строго меньшие значения переносятся
влево).

## Merge sort (top-down, stable)

Рекурсивный спуск: диапазон `[Lo, Hi)` делится пополам границей `Mid`
(левая половина `[Lo, Mid)`, правая `[Mid, Hi)` — верхняя граница
**исключительная**), половины сортируются, затем сливаются через один общий
буфер размера n (память O(n), не зависит от глубины рекурсии). Слияние
устойчивое: при равных головах первым берётся элемент **левой** половины
(`nums[i] <= nums[j]` — нестрогое `<=` как раз ради устойчивости).

### Рекурсивный спуск

```mermaid
flowchart TD
    MsStart(["mergeSortRange(Lo, Mid, Hi)"]) --> MsBase{"Hi - Lo ≤ 1?"}
    MsBase -- "да: пусто или один элемент" --> MsReturn([возврат])
    MsBase -- "нет: MergeSplit — диапазон делится" --> MsLeft["mergeSortRange(Lo, (Lo+Mid)/2, Mid)<br/>левая половина"]
    MsLeft --> MsRight["mergeSortRange(Mid, (Mid+Hi)/2, Hi)<br/>правая половина"]
    MsRight --> MsMerge["mergeRange(Lo, Mid, Hi)<br/>слияние отсортированных половин"]
    MsMerge --> MsReturn
```

### Слияние двух половин

```mermaid
flowchart TD
    MrStart(["mergeRange(Lo, Mid, Hi)"]) --> MrInit["i = Lo — голова левой<br/>j = Mid — голова правой<br/>k = Lo — позиция буфера"]
    MrInit --> MrHeads{"i < Mid и j < Hi?"}
    MrHeads -- "да" --> MrCompare{"nums[i] ≤ nums[j]?"}
    MrCompare -- "да: равенство — берём левую<br/>(устойчивость), MergeCompare" --> MrTakeLeft["take(i, слева): буфер[k] = nums[i]"]
    MrCompare -- "нет: берём правого кандидата,<br/>MergeCompare" --> MrTakeRight["take(j, справа): буфер[k] = nums[j]"]
    MrTakeLeft --> MrHeads
    MrTakeRight --> MrHeads
    MrHeads -- "нет" --> MrTail{"оставался ли хвост?"}
    MrTail -- "в левой половине" --> MrFlushLeft["дописать хвост левой в буфер"]
    MrTail -- "в правой половине" --> MrFlushRight["дописать хвост правой в буфер"]
    MrTail -- "нет: буфер полон" --> MrCopyBack["перенос буфера в nums по элементу<br/>MergeWrite на каждую позицию [Lo..Hi)"]
    MrFlushLeft --> MrCopyBack
    MrFlushRight --> MrCopyBack
    MrCopyBack --> MrReturn([конец слияния])
```

События режима наблюдения: `MergeSplit` (деление диапазона, до спуска
в половины), `MergeCompare` (сравнение голов двух отсортированных половин,
несёт исход `LeftChosen` и позицию `Dest`), `MergeWrite` (видимая запись
значения в массив на шаге переноса буфера, с пометкой `FromLeft`).
На сравнении и копировании в буфер массив ещё не меняется — события
`MergeWrite` размечают переносимые значения, шаги восстановимы.

Свойства: устойчивость, худший/средний/лучший случай O(n log n), память O(n)
(один общий буфер).

## Quicksort (Lomuto, pivot — последний)

Рекурсия по **включительному** диапазону `lo..hi` (в отличие от merge, у
которого верхняя граница исключительная). `partition` берёт pivot `nums[hi]`,
строго меньшие значения через обмен заносятся в растущий блок `[lo..i]`,
затем pivot ставится на финальную позицию `i+1`; рекурсия сортирует
`[lo..p-1]` и `[p+1..hi]`.

```mermaid
flowchart TD
    QsStart(["QuickSort(nums)"]) --> QsRecurse["quickSort(nums, 0, len(nums)-1)"]
    QsRecurse --> QsBase{"lo ≥ hi?"}
    QsBase -- "да: отрезок пуст или одноэлементный" --> QsReturn([возврат])
    QsBase -- "нет" --> QsPartition["partition(nums, lo, hi)"]
    QsPartition --> QsPivot["QuickPivot: pivot = nums[hi]<br/>i = lo - 1"]
    QsPivot --> QsScanInit["j = lo"]
    QsScanInit --> QsBounds{"j < hi?"}
    QsBounds -- "да" --> QsCompare{"nums[j] < pivot?<br/>QuickCompare"}
    QsCompare -- "да: строго меньшее, QuickSwap при i ≠ j" --> QsSwap["i++; обмен nums[i] ↔ nums[j]"]
    QsSwap --> QsNext["j++"]
    QsCompare -- "нет: не меньшее остаётся справа" --> QsNext
    QsNext --> QsBounds
    QsBounds -- "нет: сканирование окончено" --> QsPlace["обмен nums[i+1] ↔ nums[hi]<br/>QuickPlaced: pivot на позиции i+1 = p"]
    QsPlace --> QsLeft["quickSort(nums, lo, p-1)<br/>левая часть"]
    QsLeft --> QsRight["quickSort(nums, p+1, hi)<br/>правая часть"]
    QsRight --> QsReturn
```

Детали реализованного варианта:

- **Строгое** сравнение `nums[j] < pivot`: равные pivot значения остаются
  справа (в блок «строго меньших» они не входят).
- Обмен при `i == j` (строго меньшие стоят подряд) — перестановка позиции
  с собой; он пропускается и не эмитится, тихий результат тот же.
- Вариант **неустойчив**: дальние обмены при partition и финальное размещение
  pivot могут переносить значения через другие равные им значения и менять
  их взаимный порядок.
- Среднее время O(n log n), худшее O(n²) — в частности на уже упорядоченных
  входах при выбранном pivot. Дополнительная память — это глубина стека
  рекурсии: **O(log n) в среднем** и **O(n) в худшем** (при Lomuto с pivot —
  последним элементом худшая глубина рекурсии достигается на уже
  упорядоченных входах, когда partition не делит диапазон пополам).

## Сравнение вариантов

| Свойство | Merge sort | Quicksort |
|----------|------------|-----------|
| Устойчивость | да | нет |
| Среднее время | O(n log n) | O(n log n) |
| Худшее время | O(n log n) | O(n²) |
| Память | O(n) (буфер) | O(log n) в среднем, O(n) в худшем (глубина рекурсии) |
| Границы диапазона | `[Lo, Hi)`, верхняя исключительная | `[lo..hi]`, включительные |
