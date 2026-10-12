// События очереди (ADR-0001): каждое событие — атомарный шаг операции,
// достаточный для построения кадра без знания внутренностей очереди.
package structures

import "github.com/EvgeniiIvanov/algo/algorithms"

// QueueEvent — маркер-интерфейс событий очереди (sealed):
// вне пакета свои события реализовать нельзя.
type QueueEvent interface{ isQueueEvent() }

// QueueEnqueue — запись значения в конец очереди: ячейка стала хвостом.
type QueueEnqueue struct {
	Index int // индекс записанной ячейки (новый хвост)
	Value int // записанное значение
}

// QueueDequeue — снятие значения с головы: оставшиеся значения сдвинулись
// влево на одну ячейку (цена наивной dequeue на слайсе — O(n)).
type QueueDequeue struct {
	Value   int // снятое значение
	Shifted int // сколько значений сдвинулось влево (0 — очередь стала пустой)
}

// QueueOverflow — неуспешный enqueue в полную очередь: значение не записано,
// содержимое не изменилось (переполнение — штатная ситуация
// фиксированной ёмкости, а не паника).
type QueueOverflow struct {
	Value int // значение, которое пытались записать
}

func (QueueEnqueue) isQueueEvent()  {}
func (QueueDequeue) isQueueEvent()  {}
func (QueueOverflow) isQueueEvent() {}

// EnqueueWithEmitter — Enqueue в режиме наблюдения (см. ADR-0001): успешная
// запись сопровождается событием QueueEnqueue, переполнение — QueueOverflow.
// Поведение и ошибки — те же, что у тихого Enqueue.
func (q *Queue) EnqueueWithEmitter(v int, em algorithms.Emitter[QueueEvent]) error {
	if err := q.Enqueue(v); err != nil {
		emitQueue(em, QueueOverflow{Value: v})
		return err
	}
	emitQueue(em, QueueEnqueue{Index: q.Len() - 1, Value: v})
	return nil
}

// DequeueWithEmitter — Dequeue в режиме наблюдения: успешное снятие
// сопровождается событием QueueDequeue. Dequeue из пустой очереди события
// не даёт — только ошибку, как тихий Dequeue.
func (q *Queue) DequeueWithEmitter(em algorithms.Emitter[QueueEvent]) (int, error) {
	v, err := q.Dequeue()
	if err != nil {
		return 0, err
	}
	emitQueue(em, QueueDequeue{Value: v, Shifted: q.Len()})
	return v, nil
}

// emitQueue вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitQueue(em algorithms.Emitter[QueueEvent], event QueueEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
