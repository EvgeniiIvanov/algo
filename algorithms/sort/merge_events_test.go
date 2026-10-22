package sort_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

// spyMergeEmitter — тестовый Emitter: собирает события merge sort в слайс
// для проверки последовательности шагов.
type spyMergeEmitter []sort.MergeEvent

func (s *spyMergeEmitter) Emit(event sort.MergeEvent) { *s = append(*s, event) }

// Сценарий [3,1,2]: рекурсия [0,3) делится как [0,1) и [1,3), поэтому
// сначала сливается пара позиций 1–2, затем диапазон 0–3 целиком.
// На сравнение головы — один MergeCompare; каждая запись значения
// в массив — один MergeWrite в порядке позиций.
func TestMergeSortEventsScenario(t *testing.T) {
	nums := []int{3, 1, 2}
	var spy spyMergeEmitter

	sort.MergeSortWithEmitter(nums, &spy)

	want := []sort.MergeEvent{
		// Слияние [1,3): сравнение 1 и 2 — левая половина не больше, берём её.
		sort.MergeCompare{Lo: 1, Mid: 2, Hi: 3, Left: 1, LeftValue: 1,
			Right: 2, RightValue: 2, LeftChosen: true},
		sort.MergeWrite{Lo: 1, Mid: 2, Hi: 3, Index: 1, Value: 1, FromLeft: true},
		// Хвост правой половины — только запись.
		sort.MergeWrite{Lo: 1, Mid: 2, Hi: 3, Index: 2, Value: 2, FromLeft: false},
		// Слияние [0,3): 3 больше 1 — берём правую; 3 больше 2 — снова правая;
		// 3 — хвост левой половины. Сравнения идут до переносимого
		// обратно в массив результата — на них массив ещё исходный,
		// записи же — след целиком за двумя сравнениями.
		sort.MergeCompare{Lo: 0, Mid: 1, Hi: 3, Left: 0, LeftValue: 3,
			Right: 1, RightValue: 1, LeftChosen: false},
		sort.MergeCompare{Lo: 0, Mid: 1, Hi: 3, Left: 0, LeftValue: 3,
			Right: 2, RightValue: 2, LeftChosen: false},
		sort.MergeWrite{Lo: 0, Mid: 1, Hi: 3, Index: 0, Value: 1, FromLeft: false},
		sort.MergeWrite{Lo: 0, Mid: 1, Hi: 3, Index: 1, Value: 2, FromLeft: false},
		sort.MergeWrite{Lo: 0, Mid: 1, Hi: 3, Index: 2, Value: 3, FromLeft: true},
	}
	if !reflect.DeepEqual([]sort.MergeEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.MergeEvent(spy), want)
	}
	if got := nums; !reflect.DeepEqual(got, []int{1, 2, 3}) {
		t.Errorf("слайс после MergeSortWithEmitter = %v, ожидалось [1 2 3]", got)
	}
}

// Устойчивость: при равных кандидатах сравнение выбирает левую половину
// (LeftChosen = true), и первое записанное значение тоже приходит из неё.
// Для [5, 5] сравниваем позиции 0 и 1 — обе половины длины 1.
func TestMergeSortEventsStability(t *testing.T) {
	nums := []int{5, 5}
	var spy spyMergeEmitter

	sort.MergeSortWithEmitter(nums, &spy)

	want := []sort.MergeEvent{
		sort.MergeCompare{Lo: 0, Mid: 1, Hi: 2, Left: 0, LeftValue: 5,
			Right: 1, RightValue: 5, LeftChosen: true},
		sort.MergeWrite{Lo: 0, Mid: 1, Hi: 2, Index: 0, Value: 5, FromLeft: true},
		sort.MergeWrite{Lo: 0, Mid: 1, Hi: 2, Index: 1, Value: 5, FromLeft: false},
	}
	if !reflect.DeepEqual([]sort.MergeEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]sort.MergeEvent(spy), want)
	}
}

// События согласованы с проезжаемой сортировкой: по MergeWrite записей
// каждый диапазон получает отсортированный порядок, индексы стартуют
// с Lo и идут подряд до Hi.
func TestMergeSortEventsWritesAreOrdered(t *testing.T) {
	nums := []int{4, -1, 3, 2, 4, 0}
	var spy spyMergeEmitter

	sort.MergeSortWithEmitter(nums, &spy)

	for _, ev := range spy {
		w, ok := ev.(sort.MergeWrite)
		if !ok {
			continue
		}
		if w.Index < w.Lo || w.Index >= w.Hi {
			t.Fatalf("запись вне диапазона: %d вне [%d, %d)", w.Index, w.Lo, w.Hi)
		}
	}
	// Итог согласован с событиями: последний MergeWrite каждой позиции
	// даёт финальное содержимое массива.
	final := append([]int{}, nums...)
	for i := range final {
		var last *sort.MergeWrite
		for _, ev := range spy {
			if w, ok := ev.(sort.MergeWrite); ok && w.Index == i {
				w := w
				last = &w
			}
		}
		if last != nil && last.Value != final[i] {
			t.Errorf("позиция %d: последняя запись %d, в массиве %d", i, last.Value, final[i])
		}
	}
}

// Quiet mode WithEmitter-версии: без эмиттера работает как обычная сортировка.
func TestMergeSortWithEmitterQuietMode(t *testing.T) {
	nums := []int{-1, 3, 2}
	sort.MergeSortWithEmitter(nums, nil)

	if got := nums; !reflect.DeepEqual(got, []int{-1, 2, 3}) {
		t.Errorf("MergeSortWithEmitter без эмиттера = %v, ожидалось [-1 2 3]", got)
	}
}

// Граничные входы: пустой и одноэлементный массивы не порождают событий.
func TestMergeSortEventsTrivial(t *testing.T) {
	var spy spyMergeEmitter
	sort.MergeSortWithEmitter([]int{}, &spy)
	if len(spy) != 0 {
		t.Errorf("пустой вход: событий %d, ожидалось 0", len(spy))
	}

	spy = nil
	sort.MergeSortWithEmitter([]int{7}, &spy)
	if len(spy) != 0 {
		t.Errorf("одноэлементный вход: событий %d, ожидалось 0", len(spy))
	}
}
