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
	LeftChosen bool // исход сравнения: true — берём левого кандидата
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

// MergeSortWithEmitter — MergeSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// MergeSort — сортировка по возрастанию на месте с точки зрения
// вызывающего кода. На каждое сравнение голов половин стреляет один
// MergeCompare (с исходом LeftChosen); каждая запись значения в массив —
// MergeWrite.
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

	mergeSortRange(nums, buf, origins, 0, n, em)
}

// mergeSortRange — рекурсивный спуск (top-down): сортируем половины,
// потом сливаем. hi — исключительная граница, как в черновике пользователя.
func mergeSortRange(nums, buf []int, origins []bool, lo, hi int, em algorithms.Emitter[MergeEvent]) {
	if hi-lo <= 1 {
		return
	}
	mid := (lo + hi) / 2
	mergeSortRange(nums, buf, origins, lo, mid, em)
	mergeSortRange(nums, buf, origins, mid, hi, em)
	mergeRange(nums, buf, origins, lo, mid, hi, em)
}

// mergeRange — слияние отсортированных половин [Lo, Mid) и [Mid, Hi) в буфер
// и перенос результата обратно в nums.
func mergeRange(nums, buf []int, origins []bool, lo, mid, hi int, em algorithms.Emitter[MergeEvent]) {
	i, j, k := lo, mid, lo
	for i < mid && j < hi {
		fromLeft := nums[i] <= nums[j] // равенство — берём левую половину
		if em != nil {
			em.Emit(MergeCompare{
				Lo: lo, Mid: mid, Hi: hi,
				Left: i, LeftValue: nums[i],
				Right: j, RightValue: nums[j],
				LeftChosen: fromLeft,
			})
		}
		if fromLeft {
			buf[k] = nums[i]
			if origins != nil {
				origins[k] = true
			}
			i++
		} else {
			buf[k] = nums[j]
			if origins != nil {
				origins[k] = false
			}
			j++
		}
		k++
	}
	for i < mid { // хвост левой половины
		buf[k] = nums[i]
		if origins != nil {
			origins[k] = true
		}
		i++
		k++
	}
	for j < hi { // хвост правой половины
		buf[k] = nums[j]
		if origins != nil {
			origins[k] = false
		}
		j++
		k++
	}
	// Перенос результата слияния в массив — по элементу, чтобы запись
	// каждой позиции была видимым шагом.
	for k := lo; k < hi; k++ {
		nums[k] = buf[k]
		if em != nil {
			em.Emit(MergeWrite{
				Lo: lo, Mid: mid, Hi: hi,
				Index:    k,
				Value:    nums[k],
				FromLeft: origins[k],
			})
		}
	}
}
