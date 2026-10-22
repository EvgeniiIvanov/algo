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
		// Деление [0,3) — рекурсия сверху вниз, кадр деления до спуска
		// в половины.
		sort.MergeSplit{Lo: 0, Mid: 1, Hi: 3},
		// Слияние [1,3): сравнение 1 и 2 — левая половина не больше, берём
		// её; победитель уходит на позицию слияния 1.
		sort.MergeSplit{Lo: 1, Mid: 2, Hi: 3},
		sort.MergeCompare{Lo: 1, Mid: 2, Hi: 3, Left: 1, LeftValue: 1,
			Right: 2, RightValue: 2, Dest: 1, LeftChosen: true},
		sort.MergeWrite{Lo: 1, Mid: 2, Hi: 3, Index: 1, Value: 1, FromLeft: true},
		// Хвост правой половины — только запись.
		sort.MergeWrite{Lo: 1, Mid: 2, Hi: 3, Index: 2, Value: 2, FromLeft: false},
		// Слияние [0,3): 3 больше 1 — берём правую; 3 больше 2 — снова правая;
		// 3 — хвост левой половины. Сравнения идут до переносимого
		// обратно в массив результата — на них массив ещё исходный,
		// записи же — след целиком за двумя сравнениями.
		sort.MergeCompare{Lo: 0, Mid: 1, Hi: 3, Left: 0, LeftValue: 3,
			Right: 1, RightValue: 1, Dest: 0, LeftChosen: false},
		sort.MergeCompare{Lo: 0, Mid: 1, Hi: 3, Left: 0, LeftValue: 3,
			Right: 2, RightValue: 2, Dest: 1, LeftChosen: false},
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
		sort.MergeSplit{Lo: 0, Mid: 1, Hi: 2},
		sort.MergeCompare{Lo: 0, Mid: 1, Hi: 2, Left: 0, LeftValue: 5,
			Right: 1, RightValue: 5, Dest: 0, LeftChosen: true},
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

// Split-события: по одному на каждый рекурсивный диапазон (внутренний узел
// рекурсии), сверху вниз — кадр деления диапазона виден раньше активности
// его половин; граница — именно середина диапазона.
func TestMergeSortEventsSplitBeforeChildren(t *testing.T) {
	nums := []int{4, 3, 2, 1, 0}
	var spy spyMergeEmitter

	sort.MergeSortWithEmitter(nums, &spy)

	// Порядок split'ов (прямой обход рекурсии) для длины 5.
	wantSplits := []sort.MergeSplit{
		{Lo: 0, Mid: 2, Hi: 5},
		{Lo: 0, Mid: 1, Hi: 2},
		{Lo: 2, Mid: 3, Hi: 5},
		{Lo: 3, Mid: 4, Hi: 5},
	}
	var gotSplits []sort.MergeSplit
	var firstEventIndex = map[int]int{} // позиция первого события диапазона
	for i, ev := range spy {
		switch e := ev.(type) {
		case sort.MergeSplit:
			gotSplits = append(gotSplits, e)
			firstEventIndex[e.Lo*100+e.Hi] = i
		}
	}
	if !reflect.DeepEqual(gotSplits, wantSplits) {
		t.Fatalf("последовательность делений:\n получено %v\n ожидалось %v", gotSplits, wantSplits)
	}
	// Родительский split всегда раньше split'ов своих дочерних диапазонов
	// (кадр деления виден до спуска в половины; половины-синглтоны
	// split'ов не порождают).
	parents := []struct{ lo, mid, hi int }{{0, 2, 5}, {2, 3, 5}}
	for _, parent := range parents {
		p := firstEventIndex[parent.lo*100+parent.hi]
		for _, child := range []struct{ lo, hi int }{
			{parent.lo, parent.mid}, {parent.mid, parent.hi},
		} {
			ci, ok := firstEventIndex[child.lo*100+child.hi]
			if child.hi-child.lo <= 1 {
				continue // синглтон: рекурсия не спускается, событий нет
			}
			if !ok || ci < p {
				t.Fatalf("деление дочернего диапазона [%d,%d) не идёт после деления [%d,%d)", child.lo, child.hi, parent.lo, parent.hi)
			}
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
