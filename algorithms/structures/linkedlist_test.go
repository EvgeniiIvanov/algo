package structures_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// Список только с публичным API: содержимое добираем снятием с головы.
// Это сам по себе тест цепочки: если узел выпал из связности, PopFront
// вернёт не то значение или закончится раньше времени.
func drainFront(t *testing.T, l *structures.LinkedList) []int {
	t.Helper()

	var values []int
	for !l.IsEmpty() {
		v, err := l.PopFront()
		if err != nil {
			t.Fatalf("PopFront на непустом списке: %v", err)
		}
		// Len обязан убывать ровно с каждым снятием: обход от головы
		// даёт ровно Len узлов, тайл/хвост не теряются.
		values = append(values, v)
	}
	return values
}

// drainBack — то же с хвоста: проверяет работу с tail (PopBack ходит к
// предпоследнему узлу по цепочке next).
func drainBack(t *testing.T, l *structures.LinkedList) []int {
	t.Helper()

	var values []int
	for !l.IsEmpty() {
		v, err := l.PopBack()
		if err != nil {
			t.Fatalf("PopBack на непустом списке: %v", err)
		}
		values = append(values, v)
	}
	return values
}

// Новый список пуст: Len = 0, IsEmpty = true, Find ничего не находит,
// снятие с любого конца — ошибка, а не паника.
func TestLinkedListEmpty(t *testing.T) {
	l := structures.NewLinkedList()

	if l.Len() != 0 || !l.IsEmpty() {
		t.Fatalf("новый список: Len = %d, IsEmpty = %v, ожидались 0 и true", l.Len(), l.IsEmpty())
	}
	if got := l.Find(5); got != -1 {
		t.Fatalf("Find в пустом списке = %d, ожидалось -1", got)
	}
	if _, err := l.PopFront(); !errors.Is(err, structures.ErrListEmpty) {
		t.Fatalf("PopFront пустого списка: ошибка = %v, ожидалась ErrListEmpty", err)
	}
	if _, err := l.PopBack(); !errors.Is(err, structures.ErrListEmpty) {
		t.Fatalf("PopBack пустого списка: ошибка = %v, ожидалась ErrListEmpty", err)
	}
}

// Prepend/Append: порядок значениям даёт обход, а не место в памяти —
// голова впереди, хвост позади, независимо от порядка вставки.
func TestLinkedListPrependAppend(t *testing.T) {
	l := structures.NewLinkedList()
	l.Prepend(5)
	l.Prepend(3)
	l.Append(7)
	l.Append(9)

	if l.Len() != 4 || l.IsEmpty() {
		t.Fatalf("Len = %d, IsEmpty = %v, ожидались 4 и false", l.Len(), l.IsEmpty())
	}
	// Обход от головы: 3 → 5 → 7 → 9. Ровно Len узлов, порядок preserved.
	if got := drainFront(t, l); !equalInts(got, []int{3, 5, 7, 9}) {
		t.Fatalf("содержимое = %v, ожидалось [3 5 7 9]", got)
	}
}

// Инвариант связности после серии операций со всей территорией позиций:
// обход от головы даёт ровно Len узлов и в ожидаемом порядке; тайл
// поддерживается — Append после удалений продолжает цепочку.
func TestLinkedListConnectivityInvariant(t *testing.T) {
	l := structures.NewLinkedList()
	steps := []struct {
		name     string
		op       func() error
		wantVals []int
	}{
		{"prepend 5", func() error { l.Prepend(5); return nil }, []int{5}},
		{"append 7", func() error { l.Append(7); return nil }, []int{5, 7}},
		{"insert 1 9", func() error { return l.InsertAt(1, 9) }, []int{5, 9, 7}},
		{"remove 2 (хвост)", func() error { return l.RemoveAt(2) }, []int{5, 9}},
		{"append 11", func() error { l.Append(11); return nil }, []int{5, 9, 11}},
		{"remove 0 (голова)", func() error { return l.RemoveAt(0) }, []int{9, 11}},
		{"insert 1 13", func() error { return l.InsertAt(1, 13) }, []int{9, 13, 11}},
	}
	for _, st := range steps {
		if err := st.op(); err != nil {
			t.Fatalf("%s: %v", st.name, err)
		}
		if l.Len() != len(st.wantVals) {
			t.Fatalf("после %s Len = %d, ожидалось %d", st.name, l.Len(), len(st.wantVals))
		}
	}
	// Обход от головы в конце серии и без потерь узлов.
	if got := drainFront(t, l); !equalInts(got, []int{9, 13, 11}) {
		t.Fatalf("обход от головы = %v, ожидался [9 13 11]", got)
	}
}

// Tail-цепочка: удаление хвоста через RemoveAt, затем Append и снятие
// с хвоста — с поддерживаемым tail новая вставка не теряется.
func TestLinkedListTailMaintenance(t *testing.T) {
	l := structures.NewLinkedList()
	l.Append(5)
	l.Append(7)
	l.Append(9)
	if err := l.RemoveAt(2); err != nil { // удалили хвост 9
		t.Fatalf("RemoveAt(2): %v", err)
	}
	if got := drainBack(t, l); !equalInts(got, []int{7, 5}) {
		t.Fatalf("снятие с хвоста = %v, ожидалось [7 5]", got)
	}
	if l.Len() != 0 || !l.IsEmpty() {
		t.Fatalf("Len = %d, IsEmpty = %v, ожидались 0 и true", l.Len(), l.IsEmpty())
	}
}

// InsertAt: границы 0..Len включительно; вне — ErrIndexOutOfRange,
// состояние не меняется.
func TestLinkedListInsertAt(t *testing.T) {
	l := structures.NewLinkedList()
	if err := l.InsertAt(0, 5); err != nil {
		t.Fatalf("InsertAt(0, 5) в пустой список: %v", err)
	}
	l.Append(7)
	if err := l.InsertAt(2, 9); err != nil { // хвост (индекс = Len)
		t.Fatalf("InsertAt(2, 9) в хвост: %v", err)
	}
	if err := l.InsertAt(1, 3); err != nil { // середина
		t.Fatalf("InsertAt(1, 3) в середину: %v", err)
	}
	if got := drainFront(t, l); !equalInts(got, []int{5, 3, 7, 9}) {
		t.Fatalf("содержимое = %v, ожидалось [5 3 7 9]", got)
	}

	l = structures.NewLinkedList()
	l.Append(5)
	if err := l.InsertAt(-1, 9); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("InsertAt(-1, 9): ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if err := l.InsertAt(2, 9); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("InsertAt(2, 9) в список длины 1: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if got := drainFront(t, l); !equalInts(got, []int{5}) {
		t.Fatalf("после неуспешных вставок содержимое = %v, ожидалось [5]", got)
	}
}

// RemoveAt: голова, середина, хвост; границы 0..Len-1, вне — ошибка;
// узлы не сдвигаются — порядок остальных сохраняется.
func TestLinkedListRemoveAt(t *testing.T) {
	l := structures.NewLinkedList()
	for _, v := range []int{5, 9, 7, 3} {
		l.Append(v)
	}
	if err := l.RemoveAt(0); err != nil { // голова — O(1)
		t.Fatalf("RemoveAt(0): %v", err)
	}
	if err := l.RemoveAt(1); err != nil { // середина
		t.Fatalf("RemoveAt(1): %v", err)
	}
	if err := l.RemoveAt(1); err != nil { // хвост
		t.Fatalf("RemoveAt(1) хвоста: %v", err)
	}
	if l.Len() != 1 {
		t.Fatalf("Len = %d, ожидался 1", l.Len())
	}
	if got := drainFront(t, l); !equalInts(got, []int{9}) {
		t.Fatalf("содержимое = %v, ожидалось [9]", got)
	}

	l = structures.NewLinkedList()
	l.Append(5)
	if err := l.RemoveAt(-1); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAt(-1): ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if err := l.RemoveAt(1); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAt(1) в список длины 1: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if err := structures.NewLinkedList().RemoveAt(0); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAt(0) пустого списка: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if got := drainFront(t, l); !equalInts(got, []int{5}) {
		t.Fatalf("после неуспешных удалений содержимое = %v, ожидалось [5]", got)
	}
}

// PopBack: у односвязного списка ход к предпоследнему узлу — обход
// от головы (учебная цена O(n)); снятие последнего узла опустошает список.
func TestLinkedListPopBack(t *testing.T) {
	l := structures.NewLinkedList()
	for _, v := range []int{5, 7, 9} {
		l.Append(v)
	}
	want := []int{9, 7, 5}
	for i, w := range want {
		got, err := l.PopBack()
		if err != nil || got != w {
			t.Fatalf("PopBack №%d = (%d, %v), ожидалось (%d, nil)", i+1, got, err, w)
		}
		if l.Len() != len(want)-i-1 {
			t.Fatalf("после PopBack №%d Len = %d, ожидалось %d", i+1, l.Len(), len(want)-i-1)
		}
	}
	if _, err := l.PopBack(); !errors.Is(err, structures.ErrListEmpty) {
		t.Fatalf("PopBack опустевшего списка: ошибка = %v, ожидалась ErrListEmpty", err)
	}
}

// Find по значению: возвращает позицию ПЕРВОГО узла с искомым значением;
// значения в списке нет — -1, и это не ошибка.
func TestLinkedListFind(t *testing.T) {
	l := structures.NewLinkedList()
	if got := l.Find(7); got != -1 {
		t.Fatalf("Find в пустом списке = %d, ожидалось -1", got)
	}

	for _, v := range []int{3, 5, 7, 5} {
		l.Append(v)
	}
	for _, tc := range []struct{ value, want int }{
		{3, 0},
		// Дубликат: первый узел с этим значением (позиция 1, не 3).
		{5, 1},
		{7, 2},
		{9, -1},
	} {
		if got := l.Find(tc.value); got != tc.want {
			t.Fatalf("Find(%d) = %d, ожидалось %d", tc.value, got, tc.want)
		}
	}
	// Поиск не меняет список.
	if got := drainFront(t, l); !equalInts(got, []int{3, 5, 7, 5}) {
		t.Fatalf("содержимое после Find = %v, ожидалось [3 5 7 5]", got)
	}
}
