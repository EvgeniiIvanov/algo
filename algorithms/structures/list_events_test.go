package structures_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// spyListEmitter — тестовый Emitter: собирает события списка в слайс
// для проверки трейса.
type spyListEmitter []structures.ListEvent

func (s *spyListEmitter) Emit(event structures.ListEvent) { *s = append(*s, event) }

// Операции с эмиттером стреляют типизированными событиями шагов:
// куда вставили значение, с какой позиции удалили и по скольку значений
// сдвинулось вправо/влево.
func TestListEventsOrder(t *testing.T) {
	// Сценарий: insert 0 5 (пустой список), insert 0 3 (голова — сдвиг 5 вправо),
	// insert 1 9 (середина — сдвиг 5 вправо), remove 1 (удалили 9 — сдвиг [5] влево).
	// Ожидаемый трейс выписан вручную как независимый источник правды.
	want := []structures.ListEvent{
		structures.ListInsert{Index: 0, Value: 5, Shifted: 0},
		structures.ListInsert{Index: 0, Value: 3, Shifted: 1},
		structures.ListInsert{Index: 1, Value: 9, Shifted: 1},
		structures.ListRemove{Index: 1, Value: 9, Shifted: 1},
	}

	l := structures.NewList()
	var spy spyListEmitter

	if err := l.InsertWithEmitter(0, 5, &spy); err != nil {
		t.Fatalf("InsertWithEmitter(0, 5): %v", err)
	}
	if err := l.InsertWithEmitter(0, 3, &spy); err != nil {
		t.Fatalf("InsertWithEmitter(0, 3): %v", err)
	}
	if err := l.InsertWithEmitter(1, 9, &spy); err != nil {
		t.Fatalf("InsertWithEmitter(1, 9): %v", err)
	}
	got, err := l.RemoveAtWithEmitter(1, &spy)
	if err != nil || got != 9 {
		t.Fatalf("RemoveAtWithEmitter(1) = (%d, %v), ожидалось (9, nil)", got, err)
	}

	if !reflect.DeepEqual([]structures.ListEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.ListEvent(spy), want)
	}
	// И содержимое после сценария согласовано с трейсом.
	if values := listValues(t, l); !equalInts(values, []int{3, 5}) {
		t.Fatalf("содержимое после сценария = %v, ожидалось [3 5]", values)
	}
}

// Вставка в хвост списка не даёт сдвига в событии, удаление последнего
// элемента — тоже.
func TestListEventsNoShiftOnTail(t *testing.T) {
	want := []structures.ListEvent{
		structures.ListInsert{Index: 1, Value: 7, Shifted: 0},
		structures.ListRemove{Index: 1, Value: 7, Shifted: 0},
	}

	l := structures.NewList()
	if err := l.Insert(0, 5); err != nil {
		t.Fatalf("Insert(0, 5): %v", err)
	}
	var spy spyListEmitter

	if err := l.InsertWithEmitter(1, 7, &spy); err != nil {
		t.Fatalf("InsertWithEmitter(1, 7): %v", err)
	}
	if got, err := l.RemoveAtWithEmitter(1, &spy); err != nil || got != 7 {
		t.Fatalf("RemoveAtWithEmitter(1) = (%d, %v), ожидалось (7, nil)", got, err)
	}

	if !reflect.DeepEqual([]structures.ListEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.ListEvent(spy), want)
	}
}

// Ошибка индекса — та же, что у тихой версии, и событий не даёт.
func TestListInvalidIndexNoEvent(t *testing.T) {
	l := structures.NewList()
	var spy spyListEmitter

	if err := l.InsertWithEmitter(1, 5, &spy); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("InsertWithEmitter(1, 5) в пустой список: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if _, err := l.RemoveAtWithEmitter(0, &spy); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAtWithEmitter(0) пустого списка: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if len(spy) != 0 {
		t.Fatalf("ошибки индекса не должны давать событий, получено %v", []structures.ListEvent(spy))
	}
}

// Без эмиттера (quiet mode) операции работают как обычно, событий не требуют.
func TestListWithEmitterQuietMode(t *testing.T) {
	l := structures.NewList()
	if err := l.InsertWithEmitter(0, 5, nil); err != nil {
		t.Fatalf("InsertWithEmitter(0, 5, nil): %v", err)
	}
	if err := l.InsertWithEmitter(0, 3, nil); err != nil {
		t.Fatalf("InsertWithEmitter(0, 3, nil): %v", err)
	}
	if err := l.InsertWithEmitter(-1, 9, nil); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("InsertWithEmitter(-1, 9, nil): ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if got, err := l.RemoveAtWithEmitter(0, nil); err != nil || got != 3 {
		t.Fatalf("RemoveAtWithEmitter(0, nil) = (%d, %v), ожидалось (3, nil)", got, err)
	}
	if values := listValues(t, l); !equalInts(values, []int{5}) {
		t.Fatalf("содержимое = %v, ожидалось [5]", values)
	}
}
