package sort_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

// spyInsertionEmitter — тестовый Emitter: собирает события сортировки
// в слайс для проверки последовательности шагов.
type spyInsertionEmitter []sort.InsertionEvent

func (s *spyInsertionEmitter) Emit(event sort.InsertionEvent) { *s = append(*s, event) }

// Известный сценарий [3,1,2]: ключ 1 вдвигается в префикс [3] (сдвиг
// и вставка в позицию 0), ключ 2 — в префикс [1,3]: сдвиг тройки,
// сравнение с единицей останавливает сканирование, вставка в позицию 1.
// Каждый проход завершается вставкой, «дырка» в событиях видна явно.
func TestInsertionSortEventsScenario(t *testing.T) {
	nums := []int{3, 1, 2}
	var spy spyInsertionEmitter

	sort.InsertionSortWithEmitter(nums, &spy)

	want := []sort.InsertionEvent{
		// Проход ключа 1: 3 больше ключа — сдвиг вправо, затем вставка
		// в позицию 0; сканирование стартует с дырки на месте ключа (позиция 1).
		sort.InsertionCompare{Pos: 0, PosValue: 3, Key: 1, Gap: 1, Outcome: sort.InsertionGreater},
		sort.InsertionShift{From: 0, To: 1, Value: 3, Key: 1, Gap: 0},
		sort.InsertionWrite{Index: 0, Value: 1, Moved: true},
		// Проход ключа 2: 3 больше ключа — сдвиг; 1 меньше — сканирование
		// остановлено, ключ вставляется в дырку на позиции 1.
		sort.InsertionCompare{Pos: 1, PosValue: 3, Key: 2, Gap: 2, Outcome: sort.InsertionGreater},
		sort.InsertionShift{From: 1, To: 2, Value: 3, Key: 2, Gap: 1},
		sort.InsertionCompare{Pos: 0, PosValue: 1, Key: 2, Gap: 1, Outcome: sort.InsertionLess},
		sort.InsertionWrite{Index: 1, Value: 2, Moved: true},
	}
	if !reflect.DeepEqual([]sort.InsertionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.InsertionEvent(spy), want)
	}
	// После сортировки содержимое согласовано с трейсом: результат —
	// отсортированный исходный слайс.
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после InsertionSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// Обратный вход [3,2,1]: каждый ключ проходит весь префикс — цепочка
// сдвигов до начала массива, вставка в позицию 0.
func TestInsertionSortEventsReverse(t *testing.T) {
	nums := []int{3, 2, 1}
	var spy spyInsertionEmitter

	sort.InsertionSortWithEmitter(nums, &spy)

	want := []sort.InsertionEvent{
		// Ключ 2: 3 больше — сдвиг, вставка в позицию 0.
		sort.InsertionCompare{Pos: 0, PosValue: 3, Key: 2, Gap: 1, Outcome: sort.InsertionGreater},
		sort.InsertionShift{From: 0, To: 1, Value: 3, Key: 2, Gap: 0},
		sort.InsertionWrite{Index: 0, Value: 2, Moved: true},
		// Ключ 1: 3 и 2 больше — два сдвига, вставка в позицию 0.
		sort.InsertionCompare{Pos: 1, PosValue: 3, Key: 1, Gap: 2, Outcome: sort.InsertionGreater},
		sort.InsertionShift{From: 1, To: 2, Value: 3, Key: 1, Gap: 1},
		sort.InsertionCompare{Pos: 0, PosValue: 2, Key: 1, Gap: 1, Outcome: sort.InsertionGreater},
		sort.InsertionShift{From: 0, To: 1, Value: 2, Key: 1, Gap: 0},
		sort.InsertionWrite{Index: 0, Value: 1, Moved: true},
	}
	if !reflect.DeepEqual([]sort.InsertionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.InsertionEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после InsertionSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// Уже упорядоченный вход: ровно n-1 сравнений и n-1 безсдвиговых вставок —
// каждый ключ сразу встаёт на своё место, и проход завершается видимым
// событием вставки (Moved = false), а не молчанием.
func TestInsertionSortEventsAlreadySorted(t *testing.T) {
	nums := []int{1, 2, 3}
	var spy spyInsertionEmitter

	sort.InsertionSortWithEmitter(nums, &spy)

	want := []sort.InsertionEvent{
		sort.InsertionCompare{Pos: 0, PosValue: 1, Key: 2, Gap: 1, Outcome: sort.InsertionLess},
		sort.InsertionWrite{Index: 1, Value: 2, Moved: false},
		sort.InsertionCompare{Pos: 1, PosValue: 2, Key: 3, Gap: 2, Outcome: sort.InsertionLess},
		sort.InsertionWrite{Index: 2, Value: 3, Moved: false},
	}
	if !reflect.DeepEqual([]sort.InsertionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.InsertionEvent(spy), want)
	}
}

// Устойчивость: элемент, равный ключу, не сдвигается — исход сравнения
// Equal, сдвига нет, а проход завершается безсдвиговой вставкой
// (ключ возвращается на свою же позицию).
func TestInsertionSortEventsDuplicates(t *testing.T) {
	nums := []int{5, 5}
	var spy spyInsertionEmitter

	sort.InsertionSortWithEmitter(nums, &spy)

	want := []sort.InsertionEvent{
		sort.InsertionCompare{Pos: 0, PosValue: 5, Key: 5, Gap: 1, Outcome: sort.InsertionEqual},
		sort.InsertionWrite{Index: 1, Value: 5, Moved: false},
	}
	if !reflect.DeepEqual([]sort.InsertionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.InsertionEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{5, 5}) {
		t.Errorf("слайс после InsertionSortWithEmitter = %v, ожидалось [5 5]", got)
	}
}

// Граничные входы: пустой и одноэлементный массивы не порождают событий.
func TestInsertionSortEventsTrivial(t *testing.T) {
	var spy spyInsertionEmitter
	sort.InsertionSortWithEmitter([]int{}, &spy)
	if len(spy) != 0 {
		t.Errorf("пустой вход: событий %d, ожидалось 0", len(spy))
	}

	spy = nil
	sort.InsertionSortWithEmitter([]int{7}, &spy)
	if len(spy) != 0 {
		t.Errorf("одноэлементный вход: событий %d, ожидалось 0", len(spy))
	}
}

// Quiet mode WithEmitter-версии: без эмиттера работает как обычная сортировка.
func TestInsertionSortWithEmitterQuietMode(t *testing.T) {
	nums := []int{-1, 3, 2}
	sort.InsertionSortWithEmitter(nums, nil)

	if got := nums; !reflect.DeepEqual(got, []int{-1, 2, 3}) {
		t.Errorf("InsertionSortWithEmitter без эмиттера = %v, ожидалось [-1 2 3]", got)
	}
}
