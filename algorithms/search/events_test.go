package search_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/search"
)

// spyEmitter — тестовый Emitter: собирает события в слайс для проверки трейса.
type spyEmitter []search.Event

func (s *spyEmitter) Emit(event search.Event) { *s = append(*s, event) }

// SearchWithEmitter стреляет типизированными событиями каждого шага.
func TestSearchWithEmitterEventOrder(t *testing.T) {
	// Вход: [1, 3, 5, 7, 9], ищем 7.
	// Ожидаемый трейс выписан вручную как независимый источник правды:
	// середина [0..4] = 2 (значение 5), 5 < 7 → границы [3..4];
	// середина [3..4] = 3 (значение 7) → найдено на индексе 3.
	want := []search.Event{
		search.Compare{Mid: 2, Value: 5},
		search.Bounds{Low: 3, High: 4},
		search.Compare{Mid: 3, Value: 7},
		search.Found{Index: 3},
	}

	var spy spyEmitter
	got := search.SearchWithEmitter([]int{1, 3, 5, 7, 9}, 7, &spy)

	if got != 3 {
		t.Fatalf("результат поиска = %d, ожидался 3", got)
	}
	if !reflect.DeepEqual([]search.Event(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []search.Event(spy), want)
	}
}

// Промах завершается событием NotFound.
func TestSearchWithEmitterMiss(t *testing.T) {
	var spy spyEmitter
	got := search.SearchWithEmitter([]int{1, 3, 5}, 2, &spy)

	if got != -1 {
		t.Fatalf("результат поиска = %d, ожидался -1", got)
	}
	sawNotFound := false
	for _, e := range spy {
		if _, ok := e.(search.NotFound); ok {
			sawNotFound = true
		}
	}
	if !sawNotFound {
		t.Fatalf("ожидали событие NotFound в трейсе, получили %v", []search.Event(spy))
	}
}
