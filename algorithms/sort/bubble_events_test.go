package sort_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

// spySortEmitter — тестовый Emitter: собирает события сортировки в слайс
// для проверки последовательности шагов.
type spySortEmitter []sort.BubbleEvent

func (s *spySortEmitter) Emit(event sort.BubbleEvent) { *s = append(*s, event) }

// Известный сценарий [3,1,2]: первый проход — два убывающих обмена,
// второй — одно сравнение без обменов, затем досрочное завершение
// (проход без обменов значит, что массив уже упорядочен).
func TestBubbleSortEventsScenario(t *testing.T) {
	nums := []int{3, 1, 2}
	var spy spySortEmitter

	sort.BubbleSortWithEmitter(nums, &spy)

	want := []sort.BubbleEvent{
		// Проход 1: 3 и 1 — пара убывает, обмен; 3 и 2 — обмен.
		sort.BubbleCompare{Left: 0, Right: 1, LeftValue: 3, RightValue: 1},
		sort.BubbleSwap{Left: 0, Right: 1, LeftValue: 3, RightValue: 1},
		sort.BubbleCompare{Left: 1, Right: 2, LeftValue: 3, RightValue: 2},
		sort.BubbleSwap{Left: 1, Right: 2, LeftValue: 3, RightValue: 2},
		// Проход 2: 1 и 2 — порядок сохраняется; обменов нет → выход.
		sort.BubbleCompare{Left: 0, Right: 1, LeftValue: 1, RightValue: 2},
	}
	if !reflect.DeepEqual([]sort.BubbleEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.BubbleEvent(spy), want)
	}
	// После сортировки содержимое согласовано с трейсом: результат —
	// отсортированный исходный слайс.
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после BubbleSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// Уже упорядоченный вход: ровно n-1 сравнений, ноль обменов —
// ранний выход после первого прохода без обменов; равные соседи
// (строгое «больше») обменов не порождают.
func TestBubbleSortEventsAlreadySorted(t *testing.T) {
	nums := []int{1, 2, 3}
	var spy spySortEmitter

	sort.BubbleSortWithEmitter(nums, &spy)

	want := []sort.BubbleEvent{
		sort.BubbleCompare{Left: 0, Right: 1, LeftValue: 1, RightValue: 2},
		sort.BubbleCompare{Left: 1, Right: 2, LeftValue: 2, RightValue: 3},
	}
	if !reflect.DeepEqual([]sort.BubbleEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.BubbleEvent(spy), want)
	}
}

// Дубликаты не обмениваются: пара равных соседей даёт только сравнение.
func TestBubbleSortEventsDuplicates(t *testing.T) {
	nums := []int{5, 5}
	var spy spySortEmitter

	sort.BubbleSortWithEmitter(nums, &spy)

	want := []sort.BubbleEvent{
		sort.BubbleCompare{Left: 0, Right: 1, LeftValue: 5, RightValue: 5},
	}
	if !reflect.DeepEqual([]sort.BubbleEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.BubbleEvent(spy), want)
	}
}

// Граничные входы: пустой и одноэлементный массивы не порождают ни событий.
func TestBubbleSortEventsTrivial(t *testing.T) {
	var spy spySortEmitter
	sort.BubbleSortWithEmitter([]int{}, &spy)
	if len(spy) != 0 {
		t.Errorf("пустой вход: событий %d, ожидалось 0", len(spy))
	}

	spy = nil
	sort.BubbleSortWithEmitter([]int{7}, &spy)
	if len(spy) != 0 {
		t.Errorf("одноэлементный вход: событий %d, ожидалось 0", len(spy))
	}
}

// Quiet mode WithEmitter-версии: без эмиттера работает как обычная сортировка.
func TestBubbleSortWithEmitterQuietMode(t *testing.T) {
	nums := []int{-1, 3, 2}
	sort.BubbleSortWithEmitter(nums, nil)

	if got := nums; !reflect.DeepEqual(got, []int{-1, 2, 3}) {
		t.Errorf("BubbleSortWithEmitter без эмиттера = %v, ожидалось [-1 2 3]", got)
	}
}
