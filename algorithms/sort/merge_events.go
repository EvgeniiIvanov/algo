package sort

import "github.com/EvgeniiIvanov/algo/algorithms"

// MergeEvent — маркер-интерфейс событий merge sort (sealed): вне пакета
// свои события реализовать нельзя. Типы названы по алгоритму — Merge*,
// по образцу Bubble*, Selection* и Insertion*.
type MergeEvent interface{ isMergeEvent() }

// MergeCompare — сравнение голов двух отсортированных частей при слиянии:
// кандидаты стоят на позициях Left (левая половина [Lo, Mid)) и Right
// (правая половина [Mid, Hi)), всегда Left < Right. Исход сравнения несёт
// само событие: LeftChosen — true, когда берётся значение левой половины
// (в том числе при равенстве — доказательство устойчивого слияния),
// false — правой. Значения кандидатов — на момент сравнения; их достаточно,
// чтобы построить кадр без знания внутренностей алгоритма.
type MergeCompare struct {
	Lo         int  // начало сливаемого диапазона
	Mid        int  // граница половин: левая [Lo, Mid), правая [Mid, Hi)
	Hi         int  // исключительная граница сливаемого диапазона
	Left       int  // индекс кандидата левой половины
	LeftValue  int  // значение кандидата левой половины
	Right      int  // индекс кандидата правой половины (Right > Left)
	RightValue int  // значение кандидата правой половины
	Dest       int  // позиция слияния, куда пойдёт выигравший кандидат
	LeftChosen bool // исход сравнения: true — берём левого кандидата
}

// MergeSplit — рекурсивное деление диапазона [Lo, Hi) пополам границей
// Mid (левая половина [Lo, Mid), правая [Mid, Hi)). Событие стреляет до
// сортировки половин: кадр деления показывает, на какие части массив
// разбит, ещё не изменяя их. Cache пустых диапазонов нет: деление
// срабатывает один раз на каждый рекурсивный диапазон, сверху вниз.
type MergeSplit struct {
	Lo  int // начало делимого диапазона
	Mid int // граница половин: левая [Lo, Mid), правая [Mid, Hi)
	Hi  int // исключительная граница делимого диапазона
}

// MergeWrite — значение вставлено на позицию Index при слиянии
// диапазона [Lo, Hi) с границей Mid. В буфере слияния позиция та же
// (k совпадает с индексом в слайсе), видимой записью она становится
// на шаге копии буфера в массив; событие несёт итоговую запись в массив —
// одно на каждый элемент этого шага, в порядке позиций. FromLeft — из какой
// половины взято значение: при равных кандидатах свидетельство устойчивости —
// левая половина даёт значение первой.
type MergeWrite struct {
	Lo       int  // начало сливаемого диапазона
	Mid      int  // граница половин
	Hi       int  // исключительная граница сливаемого диапазона
	Index    int  // позиция записи в массиве (та же, что и в буфере)
	Value    int  // записанное значение
	FromLeft bool // значение взято из левой половины слияния
}

func (MergeCompare) isMergeEvent() {}
func (MergeWrite) isMergeEvent()   {}
func (MergeSplit) isMergeEvent()   {}

// MergeSortWithEmitter — MergeSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// MergeSort — сортировка по возрастанию на месте с точки зрения
// вызывающего кода. Каждое рекурсивное деление диапазона — MergeSplit,
// на каждое сравнение голов половин — MergeCompare (с исходом LeftChosen
// и позицией Dest), каждая запись значения в массив — MergeWrite.
//
// Идея (учебный черновик пользователя, Ч2.5): рекурсивные половины
// сортируются сверху вниз, затем сливаются через один общий буфер.
// Строгое сравнение НЕ используем: головы сравниваются, меньшая (при
// равенстве — левая) уходит в буфер первой, поэтому сортировка устойчива.
// На сравнении и копировании в буфер массив ещё не меняется: события
// MergeWrite размечают только перенесённые значения — шаги восстановимы.
func MergeSortWithEmitter(nums []int, em algorithms.Emitter[MergeEvent]) {
	n := len(nums)
	if n <= 1 {
		return
	}
	// Общий буфер слияния размера n: память O(n), не зависит от глубины рекурсии.
	buf := make([]int, n)
	// Происхождение каждого значения буфера: левая половина или правая.
	// Нужно только при эмиттере; тихий режим идёт без него полностью.
	var origins []bool
	if em != nil {
		origins = make([]bool, n)
	}

	mergeSortRange(nums, buf, origins, mergeBounds{0, n / 2, n}, em)
}

// mergeBounds — границы одного рекурсивного диапазона merge sort: [Lo, Mid)
// — левая половина, [Mid, Hi) — правая. Именованная форма сырой тройки
// lo/mid/hi вместо повторения трёх индексов через весь стек вызовов:
// событиям свои структуры, а внутренний перенос — только этой парой полей.
type mergeBounds struct {
	Lo, Mid, Hi int
}

// mergeSortRange — рекурсивный спуск (top-down): диапазон делится пополам,
// половины сортируются, затем сливаются. Деление диапазона — отдельное
// событие MergeSplit: кадр деления виден до спуска в его половины.
func mergeSortRange(nums, buf []int, origins []bool, b mergeBounds, em algorithms.Emitter[MergeEvent]) {
	if b.Hi-b.Lo <= 1 {
		return
	}
	if em != nil {
		em.Emit(MergeSplit(b)) // у MergeSplit те же поля границ — conversion правомерен
	}
	mergeSortRange(nums, buf, origins, mergeBounds{b.Lo, (b.Lo + b.Mid) / 2, b.Mid}, em)
	mergeSortRange(nums, buf, origins, mergeBounds{b.Mid, (b.Mid + b.Hi) / 2, b.Hi}, em)
	mergeRange(nums, buf, origins, b, em)
}

// mergeRange — слияние отсортированных половин диапазона в буфер и перенос
// результата обратно в nums. Форма «буфер + происхождение + позиция»
// выделена в take: четыре ветки записи (основной цикл и хвосты двух
// половин) не повторяют её по отдельности.
func mergeRange(nums, buf []int, origins []bool, b mergeBounds, em algorithms.Emitter[MergeEvent]) {
	i, j, k := b.Lo, b.Mid, b.Lo
	// take — перенос значения кандидата src в буфер на текущую позицию k
	// с пометкой происхождения и сдвигом соответствующей головы половины.
	take := func(src int, fromLeft bool) {
		buf[k] = nums[src]
		if origins != nil {
			origins[k] = fromLeft
		}
		if fromLeft {
			i++
		} else {
			j++
		}
		k++
	}
	for i < b.Mid && j < b.Hi {
		fromLeft := nums[i] <= nums[j] // равенство — берём левую половину
		if em != nil {
			em.Emit(MergeCompare{
				Lo: b.Lo, Mid: b.Mid, Hi: b.Hi,
				Left: i, LeftValue: nums[i],
				Right: j, RightValue: nums[j],
				Dest:       k,
				LeftChosen: fromLeft,
			})
		}
		if fromLeft {
			take(i, true)
		} else {
			take(j, false)
		}
	}
	for i < b.Mid { // хвост левой половины
		take(i, true)
	}
	for j < b.Hi { // хвост правой половины
		take(j, false)
	}
	// Перенос результата слияния в массив — по элементу, чтобы запись
	// каждой позиции была видимым шагом.
	for k := b.Lo; k < b.Hi; k++ {
		nums[k] = buf[k]
		if em != nil {
			em.Emit(MergeWrite{
				Lo: b.Lo, Mid: b.Mid, Hi: b.Hi,
				Index:    k,
				Value:    nums[k],
				FromLeft: origins[k],
			})
		}
	}
}
