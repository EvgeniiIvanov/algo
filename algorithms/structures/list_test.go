package structures_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// Инвариант порядка значений после серии вставок и удалений: содержимое
// следуем через Get в порядке следования.
func listValues(t *testing.T, l *structures.List) []int {
	t.Helper()

	values := make([]int, l.Len())
	for i := range values {
		v, err := l.Get(i)
		if err != nil {
			t.Fatalf("Get(%d): %v", i, err)
		}
		values[i] = v
	}
	return values
}

// Новый список пуст, Len = 0, любой Get — ошибка индекса, не паника.
func TestListEmpty(t *testing.T) {
	l := structures.NewList()

	if l.Len() != 0 {
		t.Fatalf("Len нового списка = %d, ожидался 0", l.Len())
	}
	if _, err := l.Get(0); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("Get(0) пустого списка: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
}

// Вставка в пустой список: единственный допустимый индекс — 0 (и он же хвост).
// Регрессия границы Insert: i == Len разрешено, без этого нельзя добавить
// даже первый элемент.
func TestListInsertIntoEmpty(t *testing.T) {
	l := structures.NewList()

	if err := l.Insert(0, 5); err != nil {
		t.Fatalf("Insert(0, 5) в пустой список: %v", err)
	}
	if got := listValues(t, l); len(got) != 1 || got[0] != 5 {
		t.Fatalf("содержимое = %v, ожидалось [5]", got)
	}
	if l.Len() != 1 {
		t.Fatalf("Len = %d, ожидался 1", l.Len())
	}
}

// Вставка в голову сдвигает весь хвост вправо: значения сохраняют порядок.
func TestListInsertHead(t *testing.T) {
	l := structures.NewList()
	if err := l.Insert(0, 5); err != nil {
		t.Fatalf("Insert(0, 5): %v", err)
	}

	if err := l.Insert(0, 3); err != nil {
		t.Fatalf("Insert(0, 3) в голову: %v", err)
	}
	if got := listValues(t, l); !equalInts(got, []int{3, 5}) {
		t.Fatalf("содержимое = %v, ожидалось [3 5]", got)
	}
}

// Вставка в середину: новые значения встанут между старыми.
func TestListInsertMiddle(t *testing.T) {
	l := structures.NewList()
	for _, step := range []struct{ i, v int }{{0, 5}, {1, 7}, {1, 9}} {
		if err := l.Insert(step.i, step.v); err != nil {
			t.Fatalf("Insert(%d, %d): %v", step.i, step.v, err)
		}
	}
	if got := listValues(t, l); !equalInts(got, []int{5, 9, 7}) {
		t.Fatalf("содержимое = %v, ожидалось [5 9 7]", got)
	}
}

// Вставка в хвост: индекс Len — допустимая верхняя граница (регрессия границы
// Insert в аргументе i == len(data)).
func TestListInsertTail(t *testing.T) {
	l := structures.NewList()
	if err := l.Insert(0, 5); err != nil {
		t.Fatalf("Insert(0, 5): %v", err)
	}

	if err := l.Insert(1, 7); err != nil {
		t.Fatalf("Insert(1, 7) в хвост: %v", err)
	}
	if got := listValues(t, l); !equalInts(got, []int{5, 7}) {
		t.Fatalf("содержимое = %v, ожидалось [5 7]", got)
	}
}

// Вставка по индексу вне диапазона — ошибка, состояние не меняется.
func TestListInsertOutOfRange(t *testing.T) {
	l := structures.NewList()
	if err := l.Insert(0, 5); err != nil {
		t.Fatalf("Insert(0, 5): %v", err)
	}

	if err := l.Insert(-1, 9); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("Insert(-1, 9): ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if err := l.Insert(2, 9); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("Insert(2, 9) в список длины 1: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if got := listValues(t, l); !equalInts(got, []int{5}) {
		t.Fatalf("после неуспешных вставок содержимое = %v, ожидалось [5]", got)
	}
}

// Get по всем позициям непустого списка; возвращает значения без снятия.
func TestListGet(t *testing.T) {
	l := structures.NewList()
	for _, step := range []struct{ i, v int }{{0, 5}, {1, 7}} {
		if err := l.Insert(step.i, step.v); err != nil {
			t.Fatalf("Insert(%d, %d): %v", step.i, step.v, err)
		}
	}

	for i, want := range []int{5, 7} {
		got, err := l.Get(i)
		if err != nil || got != want {
			t.Fatalf("Get(%d) = (%d, %v), ожидалось (%d, nil)", i, got, err, want)
		}
	}
	if l.Len() != 2 {
		t.Fatalf("Get снял элемент: Len = %d, ожидался 2", l.Len())
	}
}

// Удаление последнего элемента: список становится пустым, Len = 0.
func TestListRemoveLast(t *testing.T) {
	l := structures.NewList()
	if err := l.Insert(0, 5); err != nil {
		t.Fatalf("Insert(0, 5): %v", err)
	}

	if err := l.RemoveAt(0); err != nil {
		t.Fatalf("RemoveAt(0) последнего элемента: %v", err)
	}
	if l.Len() != 0 {
		t.Fatalf("после удаления последнего Len = %d, ожидался 0", l.Len())
	}
	if _, err := l.Get(0); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("Get(0) опустевшего списка: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
}

// Удаление из середины: значения правее уезжают влево, порядок сохраняется.
func TestListRemoveMiddle(t *testing.T) {
	l := structures.NewList()
	for _, step := range []struct{ i, v int }{{0, 5}, {1, 9}, {2, 7}} {
		if err := l.Insert(step.i, step.v); err != nil {
			t.Fatalf("Insert(%d, %d): %v", step.i, step.v, err)
		}
	}

	if err := l.RemoveAt(1); err != nil {
		t.Fatalf("RemoveAt(1): %v", err)
	}
	if got := listValues(t, l); !equalInts(got, []int{5, 7}) {
		t.Fatalf("содержимое = %v, ожидалось [5 7]", got)
	}
}

// RemoveAt из пустого списка и по индексу вне диапазона — ошибки,
// состояние не меняется.
func TestListRemoveOutOfRange(t *testing.T) {
	l := structures.NewList()
	if err := l.Insert(0, 5); err != nil {
		t.Fatalf("Insert(0, 5): %v", err)
	}

	if err := l.RemoveAt(-1); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAt(-1): ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if err := l.RemoveAt(1); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAt(1) в список длины 1: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if got := listValues(t, l); !equalInts(got, []int{5}) {
		t.Fatalf("после неуспешных удалений содержимое = %v, ожидалось [5]", got)
	}

	if err := structures.NewList().RemoveAt(0); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAt(0) пустого списка: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
}

// Len отражает длину содержимого после вставок и удалений.
func TestListLen(t *testing.T) {
	l := structures.NewList()
	steps := []struct{ i, v, wantLen int }{
		{0, 5, 1},
		{1, 7, 2},
		{0, 3, 3},
	}
	for _, st := range steps {
		if err := l.Insert(st.i, st.v); err != nil {
			t.Fatalf("Insert(%d, %d): %v", st.i, st.v, err)
		}
		if l.Len() != st.wantLen {
			t.Fatalf("после Insert(%d, %d) Len = %d, ожидался %d", st.i, st.v, l.Len(), st.wantLen)
		}
	}
	if err := l.RemoveAt(1); err != nil {
		t.Fatalf("RemoveAt(1): %v", err)
	}
	if l.Len() != 2 {
		t.Fatalf("после RemoveAt Len = %d, ожидался 2", l.Len())
	}
}

// equalInts — поверхностное сравнение срезов целых.
func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
