package search

import "github.com/EvgeniiIvanov/algo/algorithms"

// Event — маркер-интерфейс событий бинарного поиска (sealed):
// вне пакета свои события реализовать нельзя.
type Event interface{ isSearchEvent() }

// Compare — сравнение искомого значения со значением в середине диапазона.
type Compare struct {
	Mid   int // индекс середины
	Value int // значение в середине
}

// Bounds — сдвиг границ поиска после сравнения.
// На финальном промахе границы схлопываются (Low может стать больше High).
type Bounds struct {
	Low  int
	High int
}

// Found — элемент найден.
type Found struct {
	Index int
}

// NotFound — элемент отсутствует в массиве.
type NotFound struct{}

func (Compare) isSearchEvent()  {}
func (Bounds) isSearchEvent()   {}
func (Found) isSearchEvent()    {}
func (NotFound) isSearchEvent() {}

// SearchWithEmitter — Search в режиме наблюдения: каждый шаг
// сопровождается событием в em (см. ADR-0001). Результат тот же,
// что у тихого Search.
func SearchWithEmitter(data []int, target int, em algorithms.Emitter[Event]) int {
	low, high := 0, len(data)-1

	for low <= high {
		mid := low + (high-low)/2

		emit(em, Compare{Mid: mid, Value: data[mid]})

		switch {
		case data[mid] == target:
			emit(em, Found{Index: mid})
			return mid
		case data[mid] < target:
			low = mid + 1
		default:
			high = mid - 1
		}

		emit(em, Bounds{Low: low, High: high})
	}

	emit(em, NotFound{})
	return -1
}

// emit вызывает эмиттер, если он передан (внутренний nil-check — деталь реализации).
func emit(em algorithms.Emitter[Event], event Event) {
	if em == nil {
		return
	}
	em.Emit(event)
}
