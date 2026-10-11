package structures_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// spyEmitter — тестовый Emitter: собирает события в слайс для проверки трейса.
type spyEmitter []structures.StackEvent

func (s *spyEmitter) Emit(event structures.StackEvent) { *s = append(*s, event) }

// Операции с эмиттером стреляют типизированными событиями шагов:
// что записали/сняли и по какому индексу.
func TestStackEventsOrder(t *testing.T) {
	// Сценарий: push 5, push 7, pop.
	// Ожидаемый трейс выписан вручную как независимый источник правды:
	// 5 пишется в ячейку 0, 7 — в ячейку 1, pop снимает 7 с ячейки 1.
	want := []structures.StackEvent{
		structures.StackPush{Index: 0, Value: 5},
		structures.StackPush{Index: 1, Value: 7},
		structures.StackPop{Index: 1, Value: 7},
	}

	s := mustStack(t, 3)
	var spy spyEmitter

	if err := s.PushWithEmitter(5, &spy); err != nil {
		t.Fatalf("PushWithEmitter(5): %v", err)
	}
	if err := s.PushWithEmitter(7, &spy); err != nil {
		t.Fatalf("PushWithEmitter(7): %v", err)
	}
	got, err := s.PopWithEmitter(&spy)
	if err != nil || got != 7 {
		t.Fatalf("PopWithEmitter = (%d, %v), ожидалось (7, nil)", got, err)
	}

	if !reflect.DeepEqual([]structures.StackEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.StackEvent(spy), want)
	}
}

// Push в полный стек — событие StackOverflow и та же ошибка, что у тихого Push.
func TestStackOverflowEvent(t *testing.T) {
	s := mustStack(t, 1)
	var spy spyEmitter

	if err := s.PushWithEmitter(5, &spy); err != nil {
		t.Fatalf("PushWithEmitter(5): %v", err)
	}
	if err := s.PushWithEmitter(9, &spy); !errors.Is(err, structures.ErrStackOverflow) {
		t.Fatalf("PushWithEmitter в полный стек: ошибка = %v, ожидалась ErrStackOverflow", err)
	}

	want := []structures.StackEvent{
		structures.StackPush{Index: 0, Value: 5},
		structures.StackOverflow{Value: 9},
	}
	if !reflect.DeepEqual([]structures.StackEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.StackEvent(spy), want)
	}
	// Переполнение не изменило содержимое.
	if top, _ := s.Peek(); top != 5 {
		t.Fatalf("после переполнения вершина = %d, ожидалась 5", top)
	}
}

// Pop из пустого стека — только ошибка, без события.
func TestStackPopEmptyNoEvent(t *testing.T) {
	s := mustStack(t, 1)
	var spy spyEmitter

	if _, err := s.PopWithEmitter(&spy); !errors.Is(err, structures.ErrStackUnderflow) {
		t.Fatalf("PopWithEmitter из пустого стека: ошибка = %v, ожидалась ErrStackUnderflow", err)
	}
	if len(spy) != 0 {
		t.Fatalf("Pop из пустого стека не должен давать событий, получено %v", []structures.StackEvent(spy))
	}
}

// Без эмиттера (quiet mode) операции работают как обычно, событий не требуют.
func TestStackWithEmitterQuietMode(t *testing.T) {
	s := mustStack(t, 2)
	if err := s.PushWithEmitter(5, nil); err != nil {
		t.Fatalf("PushWithEmitter(5, nil): %v", err)
	}
	if err := s.PushWithEmitter(7, nil); err != nil {
		t.Fatalf("PushWithEmitter(7, nil): %v", err)
	}
	if err := s.PushWithEmitter(9, nil); !errors.Is(err, structures.ErrStackOverflow) {
		t.Fatalf("PushWithEmitter(9, nil) в полный стек: ошибка = %v, ожидалась ErrStackOverflow", err)
	}
	got, err := s.PopWithEmitter(nil)
	if err != nil || got != 7 {
		t.Fatalf("PopWithEmitter(nil) = (%d, %v), ожидалось (7, nil)", got, err)
	}
	if s.Len() != 1 {
		t.Fatalf("Len = %d, ожидался 1", s.Len())
	}
}
