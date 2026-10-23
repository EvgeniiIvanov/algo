package sort

import "github.com/EvgeniiIvanov/algo/algorithms"

// QuickEvent — маркер-интерфейс событий quicksort (sealed): вне пакета
// свои события реализовать нельзя. Типы названы по алгоритму — Quick*,
// по образцу Bubble*, Selection*, Insertion* и Merge*.
type QuickEvent interface{ isQuickEvent() }

// QuickRange — включительный диапазон partition: [Lo..Hi]; pivot стоит на
// позиции Hi (последний элемент диапазона). В отличие от MergeRange
// (исключительная верхняя граница), границы черновика quicksort
// включительные с обеих сторон.
type QuickRange struct {
	Lo int // включительное начало диапазона partition
	Hi int // включительная позиция pivot — конец диапазона
}

// QuickPivot — partition выбрала pivot: значение PivotValue на позиции Hi,
// на момент события это последний элемент диапазона. Событие стреляет до
// сканирования: кадр выбора показывает, относительно какого значения
// пойдёт расклад. Значений выбора достаточно, чтобы построить кадр без
// знания внутренностей алгоритма.
type QuickPivot struct {
	QuickRange     // диапазон partition
	PivotValue int // значение pivot (последний элемент диапазона)
}

// QuickCompare — текущий элемент J сравнивается с pivot. Значения — на
// момент сравнения; исход считывается со значений: строго меньше
// PivotValue — значение уйдёт в левую часть, иначе останется справа.
// Boundary — текущая граница i: конец блока строго меньших значений
// и место, куда встанет следующее строго меньшее; при i = Lo-1 блока
// ещё нет.
type QuickCompare struct {
	QuickRange     // диапазон partition
	Boundary   int // граница i — конец блока строго меньших (может быть меньше Lo)
	J          int // позиция сравниваемого элемента
	JValue     int // значение сравниваемого элемента
	PivotValue int // значение pivot
}

// QuickSwap — строго меньшее значение ушло в блок меньших: обмен позиций
// Left и Right. Left — растущая граница i (позиция, куда встаёт строго
// меньшее), Right — источник j. Значения — до обмена; зеркальное состояние
// после шага восстанавливается по индексам, как у BubbleSwap.
type QuickSwap struct {
	QuickRange     // диапазон partition
	Left       int // позиция i, куда встаёт строго меньшее значение
	Right      int // источник j, откуда берётся строго меньшее значение
	LeftValue  int // значение левой позиции до обмена
	RightValue int // значение правой позиции до обмена
}

// QuickPlaced — partition завершена: pivot value Value занял финальную
// позицию Index (i+1): слева от неё — строго меньшие значения диапазона,
// справа — не меньшие. Когда Index == Hi, pivot уже стоял на месте —
// событие всё равно фиксирует завершение partition (обменом был бы
// перестановкой элемента с самим собой).
type QuickPlaced struct {
	QuickRange     // диапазон partition: Hi — прежняя позиция pivot
	Index      int // финальная позиция pivot
	Value      int // значение pivot
}

func (QuickPivot) isQuickEvent()   {}
func (QuickCompare) isQuickEvent() {}
func (QuickSwap) isQuickEvent()    {}
func (QuickPlaced) isQuickEvent()  {}

// QuickSortWithEmitter — QuickSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// QuickSort — сортировка по возрастанию на месте с точки зрения
// вызывающего кода. Каждая partition стреляет QuickPivot (выбор pivot),
// QuickCompare на каждое сравнение с pivot, QuickSwap на каждый обмен
// строго меньшего значения влево и QuickPlaced — на финальное размещение
// pivot; затем рекурсия спускается в левую и правую части.
//
// Идея (учебный черновик пользователя, Ч2.5): рекурсия по включительному
// диапазону lo..hi, pivot — nums[hi], строго меньшие значения уходят
// влево. Строгое сравнение (`< pivot`) сохранено из черновика: равные
// pivot значения остаются справа (по правилу partition не меньшие не
// входят в блок строго меньших). Вариант неустойчив: обмены при
// partition (перенос строго меньших на границу блока) и финальное
// размещение pivot обменивают позиции с равными значениями и могут
// изменить их исходный относительный порядок. Доработка черновика, идея и границы
// сохранены: обмен элемента с самим собой (i == j, когда строго меньшие
// стоят подряд) не выполняется и не эмитится — это шаг без эффекта
// (перестановка позиции с собой); тихий результат тот же, кадры без
// пустых обменов чище.
func QuickSortWithEmitter(nums []int, em algorithms.Emitter[QuickEvent]) {
	quickSort(nums, 0, len(nums)-1, em)
}

// quickSort — рекурсивный спуск черновика: диапазон [lo..hi] сортируется
// partition вокруг pivot nums[hi], затем сортируются обе части.
func quickSort(nums []int, lo, hi int, em algorithms.Emitter[QuickEvent]) {
	if lo >= hi {
		return
	}
	p := partition(nums, lo, hi, em)
	quickSort(nums, lo, p-1, em)
	quickSort(nums, p+1, hi, em)
}

// partition — Lomuto partition черновика: pivot nums[hi], строго меньшие
// значения через обмен заносятся в блок [lo..i]. Обмен при i == j —
// перестановка позиции с собой; он пропускается (и не эмитится), тихий
// результат идентичен черновику.
func partition(nums []int, lo, hi int, em algorithms.Emitter[QuickEvent]) int {
	if em != nil {
		em.Emit(QuickPivot{QuickRange: QuickRange{Lo: lo, Hi: hi}, PivotValue: nums[hi]})
	}
	pivot := nums[hi]
	i := lo - 1
	for j := lo; j < hi; j++ {
		v := nums[j]
		if em != nil {
			em.Emit(QuickCompare{
				QuickRange: QuickRange{Lo: lo, Hi: hi},
				Boundary:   i,
				J:          j,
				JValue:     v,
				PivotValue: pivot,
			})
		}
		if v < pivot {
			i++
			if i != j {
				leftValue := nums[i] // до обмена: это значение уедет на позицию j
				nums[i], nums[j] = v, leftValue
				if em != nil {
					em.Emit(QuickSwap{
						QuickRange: QuickRange{Lo: lo, Hi: hi},
						Left:       i,
						Right:      j,
						LeftValue:  leftValue,
						RightValue: v,
					})
				}
			}
		}
	}
	nums[i+1], nums[hi] = nums[hi], nums[i+1]
	if em != nil {
		em.Emit(QuickPlaced{
			QuickRange: QuickRange{Lo: lo, Hi: hi},
			Index:      i + 1,
			Value:      pivot,
		})
	}
	return i + 1
}
