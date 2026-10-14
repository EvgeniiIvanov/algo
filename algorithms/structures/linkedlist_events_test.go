package structures_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// spyLinkedEmitter — тестовый Emitter: собирает события односвязного
// списка в слайс для проверки трейса.
type spyLinkedEmitter []structures.LinkedEvent

func (s *spyLinkedEmitter) Emit(event structures.LinkedEvent) { *s = append(*s, event) }

// Prepend/Append стреляют типизированными событиями шагов: узел встаёт
// в голову/хвост (LinkedInsert), вокруг него переключается связь next
// (LinkedLink). В пустом списке связь подключать не к чему — события нет.
func TestLinkedListEventsPrependAppend(t *testing.T) {
	want := []structures.LinkedEvent{
		structures.LinkedInsert{Index: 0, Value: 5}, // prepend в пустой список
		structures.LinkedInsert{Index: 0, Value: 3}, // prepend в голову
		structures.LinkedLink{Index: 0},             // next нового головы → старая голова
		structures.LinkedLink{Index: 1},             // next хвоста 5 → новый узел
		structures.LinkedInsert{Index: 2, Value: 7}, // append в хвост
	}

	l := structures.NewLinkedList()
	var spy spyLinkedEmitter

	l.PrependWithEmitter(5, &spy)
	l.PrependWithEmitter(3, &spy)
	l.AppendWithEmitter(7, &spy)

	if !reflect.DeepEqual([]structures.LinkedEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.LinkedEvent(spy), want)
	}
	// И содержимое после сценария согласовано с трейсом.
	if got := drainFront(t, l); !equalInts(got, []int{3, 5, 7}) {
		t.Fatalf("содержимое после сценария = %v, ожидалось [3 5 7]", got)
	}
}

// Вставка в середину: обход с визитом каждого пройденного узла
// (LinkedVisit), затем переключение связи узла перед местом вставки
// (LinkedLink) и вставка узла (LinkedInsert).
func TestLinkedListEventsInsertMiddle(t *testing.T) {
	// Список [3 5] (3 — голова). Вставка 9 на позицию 1:
	// обход дошёл до узла 0, его next переключили, узел встал на позицию 1.
	want := []structures.LinkedEvent{
		structures.LinkedVisit{Index: 0, Value: 3},
		structures.LinkedLink{Index: 0},
		structures.LinkedInsert{Index: 1, Value: 9},
	}

	l := structures.NewLinkedList()
	l.Prepend(5)
	l.Prepend(3)
	var spy spyLinkedEmitter

	if err := l.InsertAtWithEmitter(1, 9, &spy); err != nil {
		t.Fatalf("InsertAtWithEmitter(1, 9): %v", err)
	}

	if !reflect.DeepEqual([]structures.LinkedEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.LinkedEvent(spy), want)
	}
	if got := drainFront(t, l); !equalInts(got, []int{3, 9, 5}) {
		t.Fatalf("содержимое после сценария = %v, ожидалось [3 9 5]", got)
	}
}

// Вставка в голову и в хвост через InsertAtWithEmitter не ходит по узлам
// — те же события, что у Prepend/Append, без визитов.
func TestLinkedListEventsInsertAtEnds(t *testing.T) {
	l := structures.NewLinkedList()
	l.Prepend(5)
	var spy spyLinkedEmitter

	if err := l.InsertAtWithEmitter(0, 3, &spy); err != nil { // голова
		t.Fatalf("InsertAtWithEmitter(0, 3): %v", err)
	}
	if err := l.InsertAtWithEmitter(2, 7, &spy); err != nil { // хвост
		t.Fatalf("InsertAtWithEmitter(2, 7): %v", err)
	}

	want := []structures.LinkedEvent{
		structures.LinkedInsert{Index: 0, Value: 3},
		structures.LinkedLink{Index: 0},
		structures.LinkedLink{Index: 1},
		structures.LinkedInsert{Index: 2, Value: 7},
	}
	if !reflect.DeepEqual([]structures.LinkedEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.LinkedEvent(spy), want)
	}
}

// Удаление из середины: визиты по пути, переключение связи через
// удалённый узел, уход узла (LinkedRemove с его значением). Удаление
// головы — O(1) без обхода: только LinkedRemove.
func TestLinkedListEventsRemove(t *testing.T) {
	want := []structures.LinkedEvent{
		structures.LinkedVisit{Index: 0, Value: 3}, // путь к узлу перед удаляемым
		structures.LinkedLink{Index: 0},            // next узла 3 перепрыгивает удаляемый
		structures.LinkedRemove{Index: 1, Value: 9},
		structures.LinkedRemove{Index: 0, Value: 3}, // голова — без визитов
	}

	l := structures.NewLinkedList()
	l.Append(3)
	l.Append(9)
	l.Append(5)
	var spy spyLinkedEmitter

	got, err := l.RemoveAtWithEmitter(1, &spy)
	if err != nil || got != 9 {
		t.Fatalf("RemoveAtWithEmitter(1) = (%d, %v), ожидалось (9, nil)", got, err)
	}
	got, err = l.RemoveAtWithEmitter(0, &spy)
	if err != nil || got != 3 {
		t.Fatalf("RemoveAtWithEmitter(0) = (%d, %v), ожидалось (3, nil)", got, err)
	}

	if !reflect.DeepEqual([]structures.LinkedEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.LinkedEvent(spy), want)
	}
	if got := drainFront(t, l); !equalInts(got, []int{5}) {
		t.Fatalf("содержимое после сценария = %v, ожидалось [5]", got)
	}
}

// PopBack: ход к предпоследнему узлу — визиты, затем связь и уход хвоста.
func TestLinkedListEventsPopBack(t *testing.T) {
	want := []structures.LinkedEvent{
		structures.LinkedVisit{Index: 0, Value: 3},
		structures.LinkedVisit{Index: 1, Value: 5},
		structures.LinkedLink{Index: 1}, // next узла 5 → nil, 7 уходит с хвоста
		structures.LinkedRemove{Index: 2, Value: 7},
	}

	l := structures.NewLinkedList()
	l.Append(3)
	l.Append(5)
	l.Append(7)
	var spy spyLinkedEmitter

	got, err := l.PopBackWithEmitter(&spy)
	if err != nil || got != 7 {
		t.Fatalf("PopBackWithEmitter = (%d, %v), ожидалось (7, nil)", got, err)
	}

	if !reflect.DeepEqual([]structures.LinkedEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.LinkedEvent(spy), want)
	}
}

// Find по значению: визит каждого проверенного узла и финал — LinkedFound
// на первом подходящем узле или LinkedNotFound при промахе. Промах — без
// ошибки, результат -1.
func TestLinkedListEventsFind(t *testing.T) {
	// Список [3 5 7]: find 5 — визиты узлов 0 и 1, нашли на позиции 1.
	wantHit := []structures.LinkedEvent{
		structures.LinkedVisit{Index: 0, Value: 3},
		structures.LinkedVisit{Index: 1, Value: 5},
		structures.LinkedFound{Index: 1},
	}
	// find 9 — прошли весь список до конца, промах.
	wantMiss := []structures.LinkedEvent{
		structures.LinkedVisit{Index: 0, Value: 3},
		structures.LinkedVisit{Index: 1, Value: 5},
		structures.LinkedVisit{Index: 2, Value: 7},
		structures.LinkedNotFound{},
	}

	l := structures.NewLinkedList()
	l.Append(3)
	l.Append(5)
	l.Append(7)
	var spy spyLinkedEmitter

	if got := l.FindWithEmitter(5, &spy); got != 1 {
		t.Fatalf("FindWithEmitter(5) = %d, ожидалось 1", got)
	}
	if !reflect.DeepEqual([]structures.LinkedEvent(spy), wantHit) {
		t.Fatalf("трейс успешного поиска:\n получено %v\n ожидалось %v", []structures.LinkedEvent(spy), wantHit)
	}

	spy = nil // чистим шпион между сценариями
	if got := l.FindWithEmitter(9, &spy); got != -1 {
		t.Fatalf("FindWithEmitter(9) = %d, ожидалось -1", got)
	}
	if !reflect.DeepEqual([]structures.LinkedEvent(spy), wantMiss) {
		t.Fatalf("трейс промаха:\n получено %v\n ожидалось %v", []structures.LinkedEvent(spy), wantMiss)
	}
}

// Ошибка индекса — та же, что у тихой версии, и событий не даёт;
// удаление из пустого списка — тоже.
func TestLinkedListInvalidIndexNoEvent(t *testing.T) {
	l := structures.NewLinkedList()
	var spy spyLinkedEmitter

	if err := l.InsertAtWithEmitter(1, 5, &spy); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("InsertAtWithEmitter(1, 5) в пустой список: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if _, err := l.RemoveAtWithEmitter(0, &spy); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("RemoveAtWithEmitter(0) пустого списка: ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if _, err := l.PopFrontWithEmitter(&spy); !errors.Is(err, structures.ErrListEmpty) {
		t.Fatalf("PopFrontWithEmitter пустого списка: ошибка = %v, ожидалась ErrListEmpty", err)
	}
	if _, err := l.PopBackWithEmitter(&spy); !errors.Is(err, structures.ErrListEmpty) {
		t.Fatalf("PopBackWithEmitter пустого списка: ошибка = %v, ожидалась ErrListEmpty", err)
	}
	if len(spy) != 0 {
		t.Fatalf("ошибки не должны давать событий, получено %v", []structures.LinkedEvent(spy))
	}
}

// Без эмиттера (quiet mode) операции с WithEmitter работают как обычно.
func TestLinkedListWithEmitterQuietMode(t *testing.T) {
	l := structures.NewLinkedList()

	l.PrependWithEmitter(5, nil)
	l.AppendWithEmitter(7, nil)
	if err := l.InsertAtWithEmitter(1, 9, nil); err != nil {
		t.Fatalf("InsertAtWithEmitter(1, 9, nil): %v", err)
	}
	if err := l.InsertAtWithEmitter(5, 9, nil); !errors.Is(err, structures.ErrIndexOutOfRange) {
		t.Fatalf("InsertAtWithEmitter(5, 9, nil): ошибка = %v, ожидалась ErrIndexOutOfRange", err)
	}
	if got, err := l.RemoveAtWithEmitter(1, nil); err != nil || got != 9 {
		t.Fatalf("RemoveAtWithEmitter(1, nil) = (%d, %v), ожидалось (9, nil)", got, err)
	}
	if got := l.FindWithEmitter(7, nil); got != 1 {
		t.Fatalf("FindWithEmitter(7, nil) = %d, ожидалось 1", got)
	}
	if got, err := l.PopBackWithEmitter(nil); err != nil || got != 7 {
		t.Fatalf("PopBackWithEmitter(nil) = (%d, %v), ожидалось (7, nil)", got, err)
	}
	if got := drainFront(t, l); !equalInts(got, []int{5}) {
		t.Fatalf("содержимое = %v, ожидалось [5]", got)
	}
}
