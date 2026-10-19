package sort_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

// spySelectionEmitter — тестовый Emitter: собирает события selection sort
// в слайс для проверки последовательности шагов.
type spySelectionEmitter []sort.SelectionEvent

func (s *spySelectionEmitter) Emit(event sort.SelectionEvent) { *s = append(*s, event) }

// Известный сценарий [3,1,2]: первый проход — минимум 1 уходит в начало
// обменом, второй — минимум 2 на позицию 1; массив упорядочен.
// Событие сравнения несёт текущий минимум прохода (Min/MinValue),
// обмен — значения пары до обмена.
func TestSelectionSortEventsScenario(t *testing.T) {
	nums := []int{3, 1, 2}
	var spy spySelectionEmitter

	sort.SelectionSortWithEmitter(nums, &spy)

	want := []sort.SelectionEvent{
		// Проход 0: 1 меньше текущего минимума 3 — минимум переходит на
		// позицию 1; 2 минимум не смещает; обмен позиций 0 и 1.
		sort.SelectionCompare{Pass: 0, Min: 0, MinValue: 3, Scanned: 1, ScanValue: 1},
		sort.SelectionCompare{Pass: 0, Min: 1, MinValue: 1, Scanned: 2, ScanValue: 2},
		sort.SelectionSwap{Pass: 0, PassValue: 3, Min: 1, MinValue: 1},
		// Проход 1: 2 меньше текущего минимума 3 — минимум на позиции 2;
		// обмен позиций 1 и 2.
		sort.SelectionCompare{Pass: 1, Min: 1, MinValue: 3, Scanned: 2, ScanValue: 2},
		sort.SelectionSwap{Pass: 1, PassValue: 3, Min: 2, MinValue: 2},
	}
	if !reflect.DeepEqual([]sort.SelectionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.SelectionEvent(spy), want)
	}
	// После сортировки содержимое согласовано с трейсом: результат —
	// отсортированный исходный слайс.
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после SelectionSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// Уже упорядоченный вход: каждый проход оставляет минимум на первом месте —
// только сравнения, ни одного обмена (граница minIdx != i из черновика).
func TestSelectionSortEventsAlreadySorted(t *testing.T) {
	nums := []int{1, 2, 3}
	var spy spySelectionEmitter

	sort.SelectionSortWithEmitter(nums, &spy)

	want := []sort.SelectionEvent{
		sort.SelectionCompare{Pass: 0, Min: 0, MinValue: 1, Scanned: 1, ScanValue: 2},
		sort.SelectionCompare{Pass: 0, Min: 0, MinValue: 1, Scanned: 2, ScanValue: 3},
		sort.SelectionCompare{Pass: 1, Min: 1, MinValue: 2, Scanned: 2, ScanValue: 3},
	}
	if !reflect.DeepEqual([]sort.SelectionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.SelectionEvent(spy), want)
	}
}

// Дубликаты минимум не смещают: строгое «меньше» — кандидат, равный
// текущему минимуму, порождает только сравнение (стандартный вариант
// неустойчив, но выбор минимума детерминирован).
func TestSelectionSortEventsDuplicates(t *testing.T) {
	nums := []int{5, 5}
	var spy spySelectionEmitter

	sort.SelectionSortWithEmitter(nums, &spy)

	want := []sort.SelectionEvent{
		sort.SelectionCompare{Pass: 0, Min: 0, MinValue: 5, Scanned: 1, ScanValue: 5},
	}
	if !reflect.DeepEqual([]sort.SelectionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.SelectionEvent(spy), want)
	}
}

// Минимум уже на месте своего прохода: обменов нет, а последующий проход
// свой минимум находит и обменивает.
func TestSelectionSortEventsMinInPlace(t *testing.T) {
	nums := []int{1, 3, 2}
	var spy spySelectionEmitter

	sort.SelectionSortWithEmitter(nums, &spy)

	want := []sort.SelectionEvent{
		// Проход 0: минимум 1 уже на позиции 0 — только сравнения, без обмена.
		sort.SelectionCompare{Pass: 0, Min: 0, MinValue: 1, Scanned: 1, ScanValue: 3},
		sort.SelectionCompare{Pass: 0, Min: 0, MinValue: 1, Scanned: 2, ScanValue: 2},
		// Проход 1: минимум 2 на позиции 2 — обмен с позицией 1.
		sort.SelectionCompare{Pass: 1, Min: 1, MinValue: 3, Scanned: 2, ScanValue: 2},
		sort.SelectionSwap{Pass: 1, PassValue: 3, Min: 2, MinValue: 2},
	}
	if !reflect.DeepEqual([]sort.SelectionEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.SelectionEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после SelectionSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// Граничные входы: пустой и одноэлементный массивы не порождают событий.
func TestSelectionSortEventsTrivial(t *testing.T) {
	var spy spySelectionEmitter
	sort.SelectionSortWithEmitter([]int{}, &spy)
	if len(spy) != 0 {
		t.Errorf("пустой вход: событий %d, ожидалось 0", len(spy))
	}

	spy = nil
	sort.SelectionSortWithEmitter([]int{7}, &spy)
	if len(spy) != 0 {
		t.Errorf("одноэлементный вход: событий %d, ожидалось 0", len(spy))
	}
}

// Quiet mode WithEmitter-версии: без эмиттера работает как обычная сортировка.
func TestSelectionSortWithEmitterQuietMode(t *testing.T) {
	nums := []int{-1, 3, 2}
	sort.SelectionSortWithEmitter(nums, nil)

	if got := nums; !reflect.DeepEqual(got, []int{-1, 2, 3}) {
		t.Errorf("SelectionSortWithEmitter без эмиттера = %v, ожидалось [-1 2 3]", got)
	}
}
