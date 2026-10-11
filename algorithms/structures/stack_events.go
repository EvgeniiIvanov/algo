// Пакет structures — структуры данных Блока 1 (стек, очередь, список,
// связный список). Каждая структура работает в тихом режиме (quiet mode)
// и опционально стреляет типизированными событиями операций в Emitter
// (ADR-0001): этого достаточно, чтобы визуализатор построил кадр,
// не зная внутренностей структуры.
package structures

import "github.com/EvgeniiIvanov/algo/algorithms"

// StackEvent — маркер-интерфейс событий стека (sealed):
// вне пакета свои события реализовать нельзя.
type StackEvent interface{ isStackEvent() }

// StackPush — запись значения по индексу: ячейка стала новой вершиной.
type StackPush struct {
	Index int // индекс записанной ячейки
	Value int // записанное значение
}

// StackPop — удаление значения с вершины: ячейка освобождена.
type StackPop struct {
	Index int // индекс освобождённой ячейки
	Value int // снятое значение
}

// StackOverflow — неуспешный push в полный стек: значение не записано,
// содержимое не изменилось (переполнение — штатная ситуация
// фиксированной ёмкости, а не паника).
type StackOverflow struct {
	Value int // значение, которое пытались записать
}

func (StackPush) isStackEvent()     {}
func (StackPop) isStackEvent()      {}
func (StackOverflow) isStackEvent() {}

// PushWithEmitter — Push в режиме наблюдения (см. ADR-0001): успешная запись
// сопровождается событием StackPush, переполнение — событием StackOverflow.
// Поведение и ошибки — те же, что у тихого Push.
func (s *Stack) PushWithEmitter(v int, em algorithms.Emitter[StackEvent]) error {
	if err := s.Push(v); err != nil {
		emitStack(em, StackOverflow{Value: v})
		return err
	}
	emitStack(em, StackPush{Index: s.sp - 1, Value: v})
	return nil
}

// PopWithEmitter — Pop в режиме наблюдения: успешное снятие сопровождается
// событием StackPop. Pop из пустого стека события не даёт — только ошибку,
// как тихий Pop.
func (s *Stack) PopWithEmitter(em algorithms.Emitter[StackEvent]) (int, error) {
	v, err := s.Pop()
	if err != nil {
		return 0, err
	}
	emitStack(em, StackPop{Index: s.sp, Value: v})
	return v, nil
}

// emitStack вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitStack(em algorithms.Emitter[StackEvent], event StackEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
