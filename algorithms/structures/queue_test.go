package structures_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// mustQueue — обёртка для тестов: NewQueue успешен, ошибки не ждём.
func mustQueue(t *testing.T, size int) *structures.Queue {
	t.Helper()
	q, err := structures.NewQueue(size)
	if err != nil {
		t.Fatalf("NewQueue(%d): %v", size, err)
	}
	return q
}

// Инвариант FIFO: элементы выходят в порядке поступления.
func TestQueueFIFO(t *testing.T) {
	q := mustQueue(t, 3)
	for _, v := range []int{1, 2, 3} {
		if err := q.Enqueue(v); err != nil {
			t.Fatalf("Enqueue(%d) вернул ошибку %v, ожидалась запись", v, err)
		}
	}
	for i, want := range []int{1, 2, 3} {
		got, err := q.Dequeue()
		if err != nil {
			t.Fatalf("Dequeue №%d вернул ошибку %v, ожидалось значение %d", i+1, err, want)
		}
		if got != want {
			t.Fatalf("Dequeue №%d = %d, ожидалось %d (нарушен FIFO)", i+1, got, want)
		}
	}
	if !q.IsEmpty() {
		t.Fatalf("после снятия всех элементов очередь должна быть пуста, Len = %d", q.Len())
	}
}

// Dequeue и Peek из пустой очереди — ошибка, а не паника; состояние не меняется.
func TestQueueEmptyErrors(t *testing.T) {
	q := mustQueue(t, 2)

	if _, err := q.Dequeue(); !errors.Is(err, structures.ErrQueueEmpty) {
		t.Fatalf("Dequeue из пустой очереди: ошибка = %v, ожидалась ErrQueueEmpty", err)
	}
	if _, err := q.Peek(); !errors.Is(err, structures.ErrQueueEmpty) {
		t.Fatalf("Peek пустой очереди: ошибка = %v, ожидалась ErrQueueEmpty", err)
	}
	if q.Len() != 0 {
		t.Fatalf("после неуспешных Dequeue/Peek Len = %d, ожидался 0", q.Len())
	}
}

// Enqueue в полную очередь — ошибка переполнения; содержимое не меняется.
func TestQueueOverflow(t *testing.T) {
	q := mustQueue(t, 2)
	if err := q.Enqueue(1); err != nil {
		t.Fatalf("Enqueue(1): %v", err)
	}
	if err := q.Enqueue(2); err != nil {
		t.Fatalf("Enqueue(2): %v", err)
	}

	if err := q.Enqueue(3); !errors.Is(err, structures.ErrQueueFull) {
		t.Fatalf("Enqueue в полную очередь: ошибка = %v, ожидалась ErrQueueFull", err)
	}
	if q.Len() != 2 {
		t.Fatalf("после переполнения Len = %d, ожидался 2", q.Len())
	}
	if head, _ := q.Peek(); head != 1 {
		t.Fatalf("после переполнения голова = %d, ожидалась 1", head)
	}
}

// Peek возвращает голову, не снимая её.
func TestQueuePeekKeepsValue(t *testing.T) {
	q := mustQueue(t, 2)
	_ = q.Enqueue(5)
	_ = q.Enqueue(7)

	for range 2 {
		got, err := q.Peek()
		if err != nil || got != 5 {
			t.Fatalf("Peek = (%d, %v), ожидалось (5, nil)", got, err)
		}
	}
	if q.Len() != 2 {
		t.Fatalf("Peek снял элемент: Len = %d, ожидался 2", q.Len())
	}
}

// Флаги и размеры отражают заполненность очереди.
func TestQueueLenFlags(t *testing.T) {
	q := mustQueue(t, 2)
	if q.Len() != 0 || !q.IsEmpty() || q.IsFull() {
		t.Fatalf("новая очередь: Len=%d IsEmpty=%v IsFull=%v, ожидалось 0/true/false",
			q.Len(), q.IsEmpty(), q.IsFull())
	}

	_ = q.Enqueue(1)
	if q.Len() != 1 || q.IsEmpty() || q.IsFull() {
		t.Fatalf("после одного enqueue: Len=%d IsEmpty=%v IsFull=%v, ожидалось 1/false/false",
			q.Len(), q.IsEmpty(), q.IsFull())
	}

	_ = q.Enqueue(2)
	if q.Len() != 2 || !q.IsFull() {
		t.Fatalf("полная очередь: Len=%d IsFull=%v, ожидалось 2/true", q.Len(), q.IsFull())
	}
}

// Чередование enqueue/dequeue: длина и голова отслеживаются корректно.
func TestQueueInterleavedEnqueueDequeue(t *testing.T) {
	q := mustQueue(t, 3)

	steps := []struct {
		op      string
		arg     int
		wantLen int
		wantTop int // голова; игнорируется, когда wantLen == 0
	}{
		{"enqueue", 5, 1, 5},
		{"enqueue", 7, 2, 5},
		{"dequeue", 0, 1, 7},
		{"enqueue", 9, 2, 7},
		{"dequeue", 0, 1, 9},
		{"dequeue", 0, 0, 0},
	}
	for i, st := range steps {
		var err error
		if st.op == "enqueue" {
			err = q.Enqueue(st.arg)
		} else {
			_, err = q.Dequeue()
		}
		if err != nil {
			t.Fatalf("шаг %d (%s): ошибка %v", i+1, st.op, err)
		}
		if q.Len() != st.wantLen {
			t.Fatalf("шаг %d (%s): Len = %d, ожидался %d", i+1, st.op, q.Len(), st.wantLen)
		}
		if st.wantLen > 0 {
			if top, _ := q.Peek(); top != st.wantTop {
				t.Fatalf("шаг %d (%s): голова = %d, ожидалась %d", i+1, st.op, top, st.wantTop)
			}
		}
	}
}

// Повторное использование после Dequeue: слот возвращается службе,
// новые элементы встают в конец, FIFO сохраняется.
func TestQueueReuseAfterDequeue(t *testing.T) {
	q := mustQueue(t, 2)
	_ = q.Enqueue(1)
	_ = q.Enqueue(2)

	if got, _ := q.Dequeue(); got != 1 {
		t.Fatalf("Dequeue = %d, ожидалось 1", got)
	}
	if err := q.Enqueue(3); err != nil {
		t.Fatalf("Enqueue(3) после Dequeue: %v", err)
	}
	// Оставшиеся элементы выходят в правильном порядке.
	for i, want := range []int{2, 3} {
		got, err := q.Dequeue()
		if err != nil || got != want {
			t.Fatalf("Dequeue №%d = (%d, %v), ожидалось (%d, nil)", i+1, got, err, want)
		}
	}
}

// Clear опустошает очередь: элементы переиспользуются с нулевого индекса.
func TestQueueClear(t *testing.T) {
	q := mustQueue(t, 2)
	_ = q.Enqueue(1)
	_ = q.Enqueue(2)

	q.Clear()
	if !q.IsEmpty() || q.Len() != 0 {
		t.Fatalf("после Clear: Len=%d IsEmpty=%v, ожидалось 0/true", q.Len(), q.IsEmpty())
	}
	if _, err := q.Dequeue(); !errors.Is(err, structures.ErrQueueEmpty) {
		t.Fatalf("Dequeue после Clear: ошибка = %v, ожидалась ErrQueueEmpty", err)
	}

	if err := q.Enqueue(9); err != nil {
		t.Fatalf("Enqueue после Clear: %v", err)
	}
	if err := q.Enqueue(3); err != nil {
		t.Fatalf("Enqueue(3) после Clear: %v", err)
	}
	if head, _ := q.Peek(); head != 9 {
		t.Fatalf("голова после Clear и Enqueue(9) = %d, ожидалась 9", head)
	}
	if got, _ := q.Dequeue(); got != 9 {
		t.Fatalf("Dequeue после Clear = %d, ожидалось 9 (нарушен FIFO)", got)
	}
	if got, _ := q.Dequeue(); got != 3 {
		t.Fatalf("Dequeue после Clear = %d, ожидалось 3 (нарушен FIFO)", got)
	}
}

// Очередь нулевой ёмкости: пуста и полна одновременно, любой enqueue — переполнение.
func TestQueueZeroCapacity(t *testing.T) {
	q := mustQueue(t, 0)
	if !q.IsEmpty() || !q.IsFull() || q.Len() != 0 {
		t.Fatalf("очередь нулевой ёмкости: Len=%d IsEmpty=%v IsFull=%v, ожидалось 0/true/true",
			q.Len(), q.IsEmpty(), q.IsFull())
	}
	if err := q.Enqueue(1); !errors.Is(err, structures.ErrQueueFull) {
		t.Fatalf("Enqueue в очередь нулевой ёмкости: ошибка = %v, ожидалась ErrQueueFull", err)
	}
	if _, err := q.Dequeue(); !errors.Is(err, structures.ErrQueueEmpty) {
		t.Fatalf("Dequeue из очереди нулевой ёмкости: ошибка = %v, ожидалась ErrQueueEmpty", err)
	}
}

// Отрицательная ёмкость — явная ошибка, а не паника и не тихое исправление в 0.
func TestQueueNegativeCapacity(t *testing.T) {
	q, err := structures.NewQueue(-5)
	if err == nil {
		t.Fatal("NewQueue(-5) вернул очередь без ошибки, ожидалась ошибка")
	}
	if !errors.Is(err, structures.ErrQueueNegativeCapacity) {
		t.Fatalf("NewQueue(-5): ошибка = %v, ожидалась ErrQueueNegativeCapacity", err)
	}
	if q != nil {
		t.Fatal("NewQueue(-5) вернул ненулевую очередь вместе с ошибкой")
	}
}
