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
func TestInsertionSortEventsScenario(t *testing.T) {
	nums := []int{3, 1, 2}
	var spy spyInsertionEmitter

	sort.InsertionSortWithEmitter(nums, &spy)

	want := []sort.InsertionEvent{
		// Проход ключа 1: 3 больше ключа — сдвиг вправо, затем вставка в позицию 0.
		sort.InsertionCompare{Pos: 0, PosValue: 3, Key: 1, Shifts: true},
		sort.InsertionShift{From: 0, To: 1, Value: 3, Key: 1},
		sort.InsertionWrite{Index: 0, Value: 1},
		// Проход ключа 2: 3 больше ключа — сдвиг; 1 не больше — сканирование
		// остановлено, ключ вставляется в позицию 1.
		sort.InsertionCompare{Pos: 1, PosValue: 3, Key: 2, Shifts: true},
		sort.InsertionShift{From: 1, To: 2, Value: 3, Key: 2},
		sort.InsertionCompare{Pos: 0, PosValue: 1, Key: 2, Shifts: false},
		sort.InsertionWrite{Index: 1, Value: 2},
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
		sort.InsertionCompare{Pos: 0, PosValue: 3, Key: 2, Shifts: true},
		sort.InsertionShift{From: 0, To: 1, Value: 3, Key: 2},
		sort.InsertionWrite{Index: 0, Value: 2},
		// Ключ 1: 3 и 2 больше — два сдвига, вставка в позицию 0.
		sort.InsertionCompare{Pos: 1, PosValue: 3, Key: 1, Shifts: true},
		sort.InsertionShift{From: 1, To: 2, Value: 3, Key: 1},
		sort.InsertionCompare{Pos: 0, PosValue: 2, Key: 1, Shifts: true},
		sort.InsertionShift{From: 0, To: 1, Value: 2, Key: 1},
		sort.InsertionWrite{Index: 0, Value: 1},
	}
	if !reflect.DeepEqual([]sort.InsertionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.InsertionEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после InsertionSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// Уже упорядоченный вход: ровно n-1 сравнений, ноль сдвигов и вставок —
// каждый ключ сразу встаёт на своё место, вставка без сдвигов события
// не порождает.
func TestInsertionSortEventsAlreadySorted(t *testing.T) {
	nums := []int{1, 2, 3}
	var spy spyInsertionEmitter

	sort.InsertionSortWithEmitter(nums, &spy)

	want := []sort.InsertionEvent{
		sort.InsertionCompare{Pos: 0, PosValue: 1, Key: 2, Shifts: false},
		sort.InsertionCompare{Pos: 1, PosValue: 2, Key: 3, Shifts: false},
	}
	if !reflect.DeepEqual([]sort.InsertionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.InsertionEvent(spy), want)
	}
}

// Устойчивость: элемент, равный ключу, не сдвигается — строгое «больше»
// в сравнении; для [5,5] равный префикс порождает только сравнение,
// сдвига и вставки нет.
func TestInsertionSortEventsDuplicates(t *testing.T) {
	nums := []int{5, 5}
	var spy spyInsertionEmitter

	sort.InsertionSortWithEmitter(nums, &spy)

	want := []sort.InsertionEvent{
		sort.InsertionCompare{Pos: 0, PosValue: 5, Key: 5, Shifts: false},
	}
	if !reflect.DeepEqual([]sort.InsertionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.InsertionEvent(spy), want)
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
