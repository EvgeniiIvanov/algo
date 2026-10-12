package structures_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// spyQueueEmitter — тестовый Emitter: собирает события очереди в слайс
// для проверки трейса.
type spyQueueEmitter []structures.QueueEvent

func (s *spyQueueEmitter) Emit(event structures.QueueEvent) { *s = append(*s, event) }

// Операции с эмиттером стреляют типизированными событиями шагов:
// что записали в конец, что сняли с головы и сколько элементов сдвинулось.
func TestQueueEventsOrder(t *testing.T) {
	// Сценарий: enqueue 5, enqueue 7, dequeue.
	// Ожидаемый трейс выписан вручную как независимый источник правды:
	// 5 и 7 встают в ячейки 0 и 1, dequeue снимает 5, а 7 сдвигается влево.
	want := []structures.QueueEvent{
		structures.QueueEnqueue{Index: 0, Value: 5},
		structures.QueueEnqueue{Index: 1, Value: 7},
		structures.QueueDequeue{Value: 5, Shifted: 1},
	}

	q := mustQueue(t, 3)
	var spy spyQueueEmitter

	if err := q.EnqueueWithEmitter(5, &spy); err != nil {
		t.Fatalf("EnqueueWithEmitter(5): %v", err)
	}
	if err := q.EnqueueWithEmitter(7, &spy); err != nil {
		t.Fatalf("EnqueueWithEmitter(7): %v", err)
	}
	got, err := q.DequeueWithEmitter(&spy)
	if err != nil || got != 5 {
		t.Fatalf("DequeueWithEmitter = (%d, %v), ожидалось (5, nil)", got, err)
	}

	if !reflect.DeepEqual([]structures.QueueEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.QueueEvent(spy), want)
	}
}

// Enqueue в полную очередь — событие QueueOverflow и та же ошибка,
// что у тихого Enqueue.
func TestQueueOverflowEvent(t *testing.T) {
	q := mustQueue(t, 1)
	var spy spyQueueEmitter

	if err := q.EnqueueWithEmitter(5, &spy); err != nil {
		t.Fatalf("EnqueueWithEmitter(5): %v", err)
	}
	if err := q.EnqueueWithEmitter(9, &spy); !errors.Is(err, structures.ErrQueueFull) {
		t.Fatalf("EnqueueWithEmitter в полную очередь: ошибка = %v, ожидалась ErrQueueFull", err)
	}

	want := []structures.QueueEvent{
		structures.QueueEnqueue{Index: 0, Value: 5},
		structures.QueueOverflow{Value: 9},
	}
	if !reflect.DeepEqual([]structures.QueueEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v", []structures.QueueEvent(spy), want)
	}
	// Переполнение не изменило содержимое.
	if head, _ := q.Peek(); head != 5 {
		t.Fatalf("после переполнения голова = %d, ожидалась 5", head)
	}
}

// Dequeue из пустой очереди — только ошибка, без события.
func TestQueueDequeueEmptyNoEvent(t *testing.T) {
	q := mustQueue(t, 1)
	var spy spyQueueEmitter

	if _, err := q.DequeueWithEmitter(&spy); !errors.Is(err, structures.ErrQueueEmpty) {
		t.Fatalf("DequeueWithEmitter из пустой очереди: ошибка = %v, ожидалась ErrQueueEmpty", err)
	}
	if len(spy) != 0 {
		t.Fatalf("Dequeue из пустой очереди не должен давать событий, получено %v", []structures.QueueEvent(spy))
	}
}

// Без эмиттера (quiet mode) операции работают как обычно, событий не требуют.
func TestQueueWithEmitterQuietMode(t *testing.T) {
	q := mustQueue(t, 2)
	if err := q.EnqueueWithEmitter(5, nil); err != nil {
		t.Fatalf("EnqueueWithEmitter(5, nil): %v", err)
	}
	if err := q.EnqueueWithEmitter(7, nil); err != nil {
		t.Fatalf("EnqueueWithEmitter(7, nil): %v", err)
	}
	if err := q.EnqueueWithEmitter(9, nil); !errors.Is(err, structures.ErrQueueFull) {
		t.Fatalf("EnqueueWithEmitter(9, nil) в полную очередь: ошибка = %v, ожидалась ErrQueueFull", err)
	}
	if got, err := q.DequeueWithEmitter(nil); err != nil || got != 5 {
		t.Fatalf("DequeueWithEmitter(nil) = (%d, %v), ожидалось (5, nil)", got, err)
	}
	if q.Len() != 1 {
		t.Fatalf("Len = %d, ожидался 1", q.Len())
	}
}
