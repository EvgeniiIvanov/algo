package search_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/search"
)

// рекордер — тестовый Emitter: просто собирает события в слайс.
type рекордер []search.Event

func (r *рекордер) Emit(event search.Event) { *r = append(*r, event) }

// SearchWithEmitter стреляет типизированными событиями каждого шага.
func TestSearchWithEmitterПорядокСобытий(t *testing.T) {
	// Вход: [1, 3, 5, 7, 9], ищем 7.
	// Ожидаемый трейс выписан вручную как независимый источник правды:
	// середина [0..4] = 2 (значение 5), 5 < 7 → границы [3..4];
	// середина [3..4] = 3 (значение 7) → найдено на индексе 3.
	хотим := []search.Event{
		search.Compare{Mid: 2, Value: 5},
		search.Bounds{Low: 3, High: 4},
		search.Compare{Mid: 3, Value: 7},
		search.Found{Index: 3},
	}

	var r рекордер
	got := search.SearchWithEmitter([]int{1, 3, 5, 7, 9}, 7, &r)

	if got != 3 {
		t.Fatalf("результат поиска = %d, ожидался 3", got)
	}
	if !reflect.DeepEqual([]search.Event(r), хотим) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []search.Event(r), хотим)
	}
}

// Промах завершается событием NotFound.
func TestSearchWithEmitterПромах(t *testing.T) {
	var r рекордер
	got := search.SearchWithEmitter([]int{1, 3, 5}, 2, &r)

	if got != -1 {
		t.Fatalf("результат поиска = %d, ожидался -1", got)
	}
	еслиНотФаунд := false
	for _, e := range r {
		if _, ок := e.(search.NotFound); ок {
			еслиНотФаунд = true
		}
	}
	if !еслиНотФаунд {
		t.Fatalf("ожидали событие NotFound в трейсе, получили %v", []search.Event(r))
	}
}
