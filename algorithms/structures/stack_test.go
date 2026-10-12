package structures_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// mustStack — обёртка для тестов: NewStack успешен, ошибки не ждём.
func mustStack(t *testing.T, size int) *structures.Stack {
	t.Helper()
	s, err := structures.NewStack(size)
	if err != nil {
		t.Fatalf("NewStack(%d): %v", size, err)
	}
	return s
}

// Инвариант LIFO: элементы снимаются в порядке, обратном добавлению.
func TestStackLIFO(t *testing.T) {
	s := mustStack(t, 3)
	for _, v := range []int{1, 2, 3} {
		if err := s.Push(v); err != nil {
			t.Fatalf("Push(%d) вернул ошибку %v, ожидалась запись", v, err)
		}
	}
	for i, want := range []int{3, 2, 1} {
		got, err := s.Pop()
		if err != nil {
			t.Fatalf("Pop №%d вернул ошибку %v, ожидалось значение %d", i+1, err, want)
		}
		if got != want {
			t.Fatalf("Pop №%d = %d, ожидалось %d (нарушен LIFO)", i+1, got, want)
		}
	}
	if !s.IsEmpty() {
		t.Fatalf("после снятия всех элементов стек должен быть пуст, Len = %d", s.Len())
	}
}

// Pop и Peek из пустого стека — ошибка, а не паника; состояние не меняется.
func TestStackEmptyErrors(t *testing.T) {
	s := mustStack(t, 2)

	if _, err := s.Pop(); !errors.Is(err, structures.ErrStackUnderflow) {
		t.Fatalf("Pop из пустого стека: ошибка = %v, ожидалась ErrStackUnderflow", err)
	}
	if _, err := s.Peek(); !errors.Is(err, structures.ErrStackUnderflow) {
		t.Fatalf("Peek пустого стека: ошибка = %v, ожидалась ErrStackUnderflow", err)
	}
	if s.Len() != 0 {
		t.Fatalf("после неуспешных Pop/Peek Len = %d, ожидался 0", s.Len())
	}
}

// Push в полный стек — ошибка переполнения; содержимое не меняется.
func TestStackOverflow(t *testing.T) {
	s := mustStack(t, 2)
	if err := s.Push(1); err != nil {
		t.Fatalf("Push(1): %v", err)
	}
	if err := s.Push(2); err != nil {
		t.Fatalf("Push(2): %v", err)
	}

	if err := s.Push(3); !errors.Is(err, structures.ErrStackOverflow) {
		t.Fatalf("Push в полный стек: ошибка = %v, ожидалась ErrStackOverflow", err)
	}
	if s.Len() != 2 {
		t.Fatalf("после переполнения Len = %d, ожидался 2", s.Len())
	}
	if top, _ := s.Peek(); top != 2 {
		t.Fatalf("после переполнения вершина = %d, ожидалась 2", top)
	}
}

// Peek возвращает вершину, не снимая её.
func TestStackPeekKeepsValue(t *testing.T) {
	s := mustStack(t, 2)
	_ = s.Push(5)
	_ = s.Push(7)

	for range 2 {
		got, err := s.Peek()
		if err != nil || got != 7 {
			t.Fatalf("Peek = (%d, %v), ожидалось (7, nil)", got, err)
		}
	}
	if s.Len() != 2 {
		t.Fatalf("Peek снял элемент: Len = %d, ожидался 2", s.Len())
	}
}

// Флаги и размеры отражают заполненность стека.
func TestStackLenCapFlags(t *testing.T) {
	s := mustStack(t, 2)
	if s.Cap() != 2 || s.Len() != 0 || !s.IsEmpty() || s.IsFull() {
		t.Fatalf("новый стек: Cap=%d Len=%d IsEmpty=%v IsFull=%v, ожидалось 2/0/true/false",
			s.Cap(), s.Len(), s.IsEmpty(), s.IsFull())
	}

	_ = s.Push(1)
	if s.Len() != 1 || s.IsEmpty() || s.IsFull() {
		t.Fatalf("после одного push: Len=%d IsEmpty=%v IsFull=%v, ожидалось 1/false/false",
			s.Len(), s.IsEmpty(), s.IsFull())
	}

	_ = s.Push(2)
	if s.Len() != 2 || !s.IsFull() {
		t.Fatalf("полный стек: Len=%d IsFull=%v, ожидалось 2/true", s.Len(), s.IsFull())
	}
}

// Clear опустошает стек: ячейки переиспользуются с нулевого индекса.
func TestStackClear(t *testing.T) {
	s := mustStack(t, 2)
	_ = s.Push(1)
	_ = s.Push(2)

	s.Clear()
	if !s.IsEmpty() || s.Len() != 0 {
		t.Fatalf("после Clear: Len=%d IsEmpty=%v, ожидалось 0/true", s.Len(), s.IsEmpty())
	}
	if _, err := s.Pop(); !errors.Is(err, structures.ErrStackUnderflow) {
		t.Fatalf("Pop после Clear: ошибка = %v, ожидалась ErrStackUnderflow", err)
	}

	if err := s.Push(9); err != nil {
		t.Fatalf("Push после Clear: %v", err)
	}
	if top, _ := s.Peek(); top != 9 {
		t.Fatalf("вершина после Clear и Push(9) = %d, ожидалась 9", top)
	}
}

// Чередование push/pop: длина и вершина отслеживаются корректно.
func TestStackInterleavedPushPop(t *testing.T) {
	s := mustStack(t, 3)

	steps := []struct {
		op      string
		arg     int
		wantLen int
		wantTop int // игнорируется, когда wantLen == 0
	}{
		{"push", 5, 1, 5},
		{"push", 7, 2, 7},
		{"pop", 0, 1, 5},
		{"push", 9, 2, 9},
		{"pop", 0, 1, 5},
		{"pop", 0, 0, 0},
	}
	for i, st := range steps {
		var err error
		if st.op == "push" {
			err = s.Push(st.arg)
		} else {
			_, err = s.Pop()
		}
		if err != nil {
			t.Fatalf("шаг %d (%s): ошибка %v", i+1, st.op, err)
		}
		if s.Len() != st.wantLen {
			t.Fatalf("шаг %d (%s): Len = %d, ожидался %d", i+1, st.op, s.Len(), st.wantLen)
		}
		if st.wantLen > 0 {
			if top, _ := s.Peek(); top != st.wantTop {
				t.Fatalf("шаг %d (%s): вершина = %d, ожидалась %d", i+1, st.op, top, st.wantTop)
			}
		}
	}
}

// Стек нулевой ёмкости: пуст и полон одновременно, любой push — переполнение.
func TestStackZeroCapacity(t *testing.T) {
	s := mustStack(t, 0)
	if !s.IsEmpty() || !s.IsFull() || s.Len() != 0 || s.Cap() != 0 {
		t.Fatalf("стек нулевой ёмкости: Len=%d Cap=%d IsEmpty=%v IsFull=%v, ожидалось 0/0/true/true",
			s.Len(), s.Cap(), s.IsEmpty(), s.IsFull())
	}
	if err := s.Push(1); !errors.Is(err, structures.ErrStackOverflow) {
		t.Fatalf("Push в стек нулевой ёмкости: ошибка = %v, ожидалась ErrStackOverflow", err)
	}
	if _, err := s.Pop(); !errors.Is(err, structures.ErrStackUnderflow) {
		t.Fatalf("Pop из стека нулевой ёмкости: ошибка = %v, ожидалась ErrStackUnderflow", err)
	}
}

// Отрицательная ёмкость — явная ошибка, а не паника и не тихое исправление в 0.
func TestStackNegativeCapacity(t *testing.T) {
	s, err := structures.NewStack(-5)
	if err == nil {
		t.Fatal("NewStack(-5) вернул стек без ошибки, ожидалась ошибка")
	}
	if !errors.Is(err, structures.ErrStackNegativeCapacity) {
		t.Fatalf("NewStack(-5): ошибка = %v, ожидалась ErrStackNegativeCapacity", err)
	}
	if s != nil {
		t.Fatal("NewStack(-5) вернул ненулевой стек вместе с ошибкой")
	}
}
