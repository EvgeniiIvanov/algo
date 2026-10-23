package sort_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

// spyQuickEmitter — тестовый Emitter: собирает события quicksort в слайс
// для проверки последовательности шагов.
type spyQuickEmitter []sort.QuickEvent

func (s *spyQuickEmitter) Emit(event sort.QuickEvent) { *s = append(*s, event) }

// Сценарий [2,4,1,3]: partition(0..3) вокруг pivot 3 — 1 уходит в блок
// меньших обменом 1↔2, pivot встаёт на позицию 2; затем рекурсия слева
// partition(0..1) вокруг pivot 1 (значение читается уже из
// переупорядоченного массива) — pivot встаёт на позицию 0. На сравнение
// с pivot — один QuickCompare (с текущей границей блока меньших); каждый
// фактически состоявшийся обмен — QuickSwap; завершение partition —
// QuickPlaced с финальной позицией pivot. Перестановки элемента с самим
// собой (j=0 в partition(0..3)) не порождают ни обмена, ни события.
func TestQuickSortEventsScenario(t *testing.T) {
	r := sort.QuickRange{Lo: 0, Hi: 3}    // partition [0..3] вокруг pivot 3
	left := sort.QuickRange{Lo: 0, Hi: 1} // partition [0..1] вокруг pivot 1
	nums := []int{2, 4, 1, 3}
	var spy spyQuickEmitter

	sort.QuickSortWithEmitter(nums, &spy)

	want := []sort.QuickEvent{
		// Partition [0..3]: pivot — последний элемент, значение 3.
		sort.QuickPivot{QuickRange: r, PivotValue: 3},
		// 2 < 3: уйдёт влево (обмен с собой на j=0 пропущен — границы блоков
		// ещё нет, Boundary = -1).
		sort.QuickCompare{QuickRange: r, Boundary: -1, J: 0, JValue: 2, PivotValue: 3},
		// 4 не меньше 3 — остаётся справа; блок меньших вырос до позиции 0.
		sort.QuickCompare{QuickRange: r, Boundary: 0, J: 1, JValue: 4, PivotValue: 3},
		// 1 < 3 — уходит влево обменом с позицией 1 (значение 4).
		sort.QuickCompare{QuickRange: r, Boundary: 0, J: 2, JValue: 1, PivotValue: 3},
		sort.QuickSwap{QuickRange: r, Left: 1, Right: 2, LeftValue: 4, RightValue: 1},
		// Partition завершена: pivot 3 — на финальной позиции 2.
		sort.QuickPlaced{QuickRange: r, Index: 2, Value: 3},
		// Рекурсия слева partition(0..1): pivot — последний элемент
		// в уже переупорядоченном массиве, значение 1.
		sort.QuickPivot{QuickRange: left, PivotValue: 1},
		// 2 не меньше 1 — остаётся справа.
		sort.QuickCompare{QuickRange: left, Boundary: -1, J: 0, JValue: 2, PivotValue: 1},
		// Pivot 1 занимает финальную позицию 0 (обмен 0↔1).
		sort.QuickPlaced{QuickRange: left, Index: 0, Value: 1},
	}
	if !reflect.DeepEqual([]sort.QuickEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.QuickEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3, 4}) {
		t.Errorf("слайс после QuickSortWithEmitter = %v, ожидалось [1 2 3 4]", got)
	}
}

// Строгое сравнение из черновика: дубликаты pivot остаются справа.
// Для [2,2]: pivot — nums[1] = 2; единственный кандидат 2 не строго меньше —
// остаётся справа, обменов нет.
func TestQuickSortEventsDuplicatesStayRight(t *testing.T) {
	r := sort.QuickRange{Lo: 0, Hi: 1}
	nums := []int{2, 2}
	var spy spyQuickEmitter

	sort.QuickSortWithEmitter(nums, &spy)

	want := []sort.QuickEvent{
		sort.QuickPivot{QuickRange: r, PivotValue: 2},
		sort.QuickCompare{QuickRange: r, Boundary: -1, J: 0, JValue: 2, PivotValue: 2},
		sort.QuickPlaced{QuickRange: r, Index: 0, Value: 2},
	}
	if !reflect.DeepEqual([]sort.QuickEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.QuickEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{2, 2}) {
		t.Errorf("слайс после QuickSortWithEmitter = %v, ожидалось [2 2]", got)
	}
}

// Вырожденный вход — уже упорядоченный массив: каждый pivot — максимум
// диапазона, все кандидаты строго меньше и стоят подряд (обмен с собой
// пропускается), поэтому в потоке нет ни одного QuickSwap; pivot ставится
// последним обменом с собой — фиксируется только QuickPlaced.
func TestQuickSortEventsSortedDegenerate(t *testing.T) {
	full := sort.QuickRange{Lo: 0, Hi: 2}
	head := sort.QuickRange{Lo: 0, Hi: 1}
	nums := []int{1, 2, 3}
	var spy spyQuickEmitter

	sort.QuickSortWithEmitter(nums, &spy)

	want := []sort.QuickEvent{
		sort.QuickPivot{QuickRange: full, PivotValue: 3},
		sort.QuickCompare{QuickRange: full, Boundary: -1, J: 0, JValue: 1, PivotValue: 3},
		sort.QuickCompare{QuickRange: full, Boundary: 0, J: 1, JValue: 2, PivotValue: 3},
		sort.QuickPlaced{QuickRange: full, Index: 2, Value: 3},
		sort.QuickPivot{QuickRange: head, PivotValue: 2},
		sort.QuickCompare{QuickRange: head, Boundary: -1, J: 0, JValue: 1, PivotValue: 2},
		sort.QuickPlaced{QuickRange: head, Index: 1, Value: 2},
	}
	if !reflect.DeepEqual([]sort.QuickEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.QuickEvent(spy), want)
	}
	for _, ev := range spy {
		if _, ok := ev.(sort.QuickSwap); ok {
			t.Errorf("на уже упорядоченном входе обменов не ожидается, получено %T", ev)
		}
	}
}

// Второй вырожденный случай — обратный вход: каждый pivot — минимум
// своего диапазона, обменов во время сканирования нет, pivot переносится
// финальным обменом в начало.
func TestQuickSortEventsReverse(t *testing.T) {
	full := sort.QuickRange{Lo: 0, Hi: 2}
	tail := sort.QuickRange{Lo: 1, Hi: 2}
	nums := []int{3, 2, 1}
	var spy spyQuickEmitter

	sort.QuickSortWithEmitter(nums, &spy)

	want := []sort.QuickEvent{
		// Partition(0..2): pivot 1 — оба кандидата не строго меньше;
		// финальная позиция pivot — 0 (обмен 0↔2 выпиливает 3 в конец).
		sort.QuickPivot{QuickRange: full, PivotValue: 1},
		sort.QuickCompare{QuickRange: full, Boundary: -1, J: 0, JValue: 3, PivotValue: 1},
		sort.QuickCompare{QuickRange: full, Boundary: -1, J: 1, JValue: 2, PivotValue: 1},
		sort.QuickPlaced{QuickRange: full, Index: 0, Value: 1},
		// Рекурсия справа partition(1..2): pivot — nums[2] = 3.
		sort.QuickPivot{QuickRange: tail, PivotValue: 3},
		sort.QuickCompare{QuickRange: tail, Boundary: 0, J: 1, JValue: 2, PivotValue: 3},
		sort.QuickPlaced{QuickRange: tail, Index: 2, Value: 3},
	}
	if !reflect.DeepEqual([]sort.QuickEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.QuickEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после QuickSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// События согласованы с границами partition: сравнения и обмены живут
// внутри [Lo..Hi), финальная позиция pivot — в [Lo..Hi].
func TestQuickSortEventsRangesConsistent(t *testing.T) {
	nums := []int{4, -1, 3, -1, 0, 2}
	var spy spyQuickEmitter

	sort.QuickSortWithEmitter(nums, &spy)

	for _, ev := range spy {
		switch e := ev.(type) {
		case sort.QuickPivot:
			if e.Hi < e.Lo {
				t.Fatalf("пустой диапазон partition: %v", e)
			}
		case sort.QuickCompare:
			if e.J < e.Lo || e.J >= e.Hi {
				t.Fatalf("сравнение вне диапазона: j=%d вне [%d..%d]", e.J, e.Lo, e.Hi)
			}
			if e.Boundary < e.Lo-1 || e.Boundary >= e.Hi {
				t.Fatalf("граница вне диапазона: boundary=%d в partition [%d..%d]", e.Boundary, e.Lo, e.Hi)
			}
		case sort.QuickSwap:
			if e.Left < e.Lo || e.Left >= e.Hi || e.Right <= e.Left || e.Right >= e.Hi {
				t.Fatalf("обмен вне диапазона: %d и %d вне [%d..%d)", e.Left, e.Right, e.Lo, e.Hi)
			}
		case sort.QuickPlaced:
			if e.Index < e.Lo || e.Index > e.Hi {
				t.Fatalf("финальная позиция pivot вне диапазона: %d вне [%d..%d]", e.Index, e.Lo, e.Hi)
			}
		}
	}
	if got, want := nums, []int{-1, -1, 0, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("итог сортировки = %v, ожидалось %v", got, want)
	}
}

// Quiet mode WithEmitter-версии: без эмиттера работает как обычная сортировка.
func TestQuickSortWithEmitterQuietMode(t *testing.T) {
	nums := []int{-1, 3, 2}
	sort.QuickSortWithEmitter(nums, nil)

	if got := nums; !reflect.DeepEqual(got, []int{-1, 2, 3}) {
		t.Errorf("QuickSortWithEmitter без эмиттера = %v, ожидалось [-1 2 3]", got)
	}
}

// Граничные входы: пустой и одноэлементный массивы не порождают событий.
func TestQuickSortEventsTrivial(t *testing.T) {
	var spy spyQuickEmitter
	sort.QuickSortWithEmitter([]int{}, &spy)
	if len(spy) != 0 {
		t.Errorf("пустой вход: событий %d, ожидалось 0", len(spy))
	}

	spy = nil
	sort.QuickSortWithEmitter([]int{7}, &spy)
	if len(spy) != 0 {
		t.Errorf("одноэлементный вход: событий %d, ожидалось 0", len(spy))
	}
}
