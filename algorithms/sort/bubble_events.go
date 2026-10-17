package sort

import "github.com/EvgeniiIvanov/algo/algorithms"

// Event — маркер-интерфейс событий bubble sort (sealed): вне пакета
// свои события реализовать нельзя.
type Event interface{ isSortEvent() }

// Compare — сравнение соседних элементов на позициях Left и Right
// (Right = Left+1). Значения пары — на момент сравнения; их достаточно,
// чтобы построить кадр без знания внутренностей алгоритма.
type Compare struct {
	Left       int // индекс левого элемента пары
	Right      int // индекс правого элемента пары
	LeftValue  int // значение левого элемента
	RightValue int // значение правого элемента
}

// Swap — обмен строго убывающей соседней пары Left и Right. Значения те же,
// что и в предыдущем сравнении этой пары — до обмена: шаг «после обмена»
// восстанавливается по индексам, порядок восстановления не искажает.
type Swap struct {
	Left       int // индекс левого элемента пары
	Right      int // индекс правого элемента пары
	LeftValue  int // значение левого элемента до обмена
	RightValue int // значение правого элемента до обмена
}

func (Compare) isSortEvent() {}
func (Swap) isSortEvent()    {}

// BubbleSortWithEmitter — BubbleSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// BubbleSort — сортировка по возрастанию на месте. На каждое сравнение
// соседей стреляет один Compare; состоявшийся обмен — ещё и Swap.
func BubbleSortWithEmitter(nums []int, em algorithms.Emitter[Event]) {
	n := len(nums)
	for i := 0; i < n-1; i++ {
		swapped := false
		for j := 0; j < n-1-i; j++ {
			left, right := nums[j], nums[j+1]

			emit(em, Compare{Left: j, Right: j + 1, LeftValue: left, RightValue: right})

			if left > right {
				nums[j], nums[j+1] = right, left
				swapped = true
				emit(em, Swap{Left: j, Right: j + 1, LeftValue: left, RightValue: right})
			}
		}
		if !swapped {
			break
		}
	}
}

// emit вызывает эмиттер, если он передан (внутренний nil-check — деталь реализации).
func emit(em algorithms.Emitter[Event], event Event) {
	if em == nil {
		return
	}
	em.Emit(event)
}
