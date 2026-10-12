package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// StructureQueue — идентичность очереди в снимке StructureSnapshot.
const StructureQueue = "queue"

// QueueOp — одна операция сценария очереди: enqueue со значением или dequeue.
type QueueOp struct {
	IsEnqueue bool // true — enqueue значения Value, false — dequeue
	Value     int  // значение для enqueue
}

// QueueRun — Frame Renderer очереди: прогоняет сценарий операций над очередью
// фиксированной ёмкости и на каждый шаг отдаёт кадр в emit.
// Сценарий должен быть валидным (dequeue только из непустой очереди) —
// валидация входа живёт в маршруте; переполнение — легальное событие сценария.
// Отрицательная ёмкость — ошибка конструктора, кадры не создаются.
func QueueRun(capacity int, ops []QueueOp, emit func(frame.Frame)) error {
	queue, err := structures.NewQueue(capacity)
	if err != nil {
		return err
	}

	r := &queueFrames{
		queue:    queue,
		capacity: capacity,
		emit:     emit,
	}

	r.frame(fmt.Sprintf("Создаём пустую очередь фиксированной ёмкости %d. Впереди операций: %d.",
		capacity, len(ops)))

	for _, op := range ops {
		if op.IsEnqueue {
			r.operation = fmt.Sprintf("enqueue %d", op.Value)
			// Переполнение — не сбой Run, а учебное событие: кадр строит Emit.
			_ = r.queue.EnqueueWithEmitter(op.Value, r)
			continue
		}
		r.operation = "dequeue"
		// Сценарий валидирован до Run: dequeue из пустой очереди сюда не доходит.
		_, _ = r.queue.DequeueWithEmitter(r)
	}
	return nil
}

// QueueFrames — все кадры сценария сразу (удобно для тестов).
func QueueFrames(capacity int, ops []QueueOp) ([]frame.Frame, error) {
	var frames []frame.Frame
	if err := QueueRun(capacity, ops, func(f frame.Frame) { frames = append(frames, f) }); err != nil {
		return nil, err
	}
	return frames, nil
}

// queueFrames — накапливает состояние между событиями и строит кадры.
// Реализует algorithms.Emitter[structures.QueueEvent].
type queueFrames struct {
	queue      *structures.Queue
	values     []int // зеркало содержимого из событий (голова → хвост; кадр не зависит от будущего)
	capacity   int
	operation  string      // имя текущей операции ("enqueue 5", "dequeue")
	highlights []Highlight // подсветки текущего шага
	step       int
	emit       func(frame.Frame)
}

// Emit — одно событие операции → один кадр (ADR-0002).
func (r *queueFrames) Emit(ev structures.QueueEvent) {
	switch e := ev.(type) {
	case structures.QueueEnqueue:
		r.values = append(r.values, e.Value)
		r.highlights = []Highlight{{Index: e.Index, Role: RoleAdded}}
		r.frame(fmt.Sprintf("%s: добавили %d в конец очереди (ячейка %d).",
			r.operation, e.Value, e.Index))

	case structures.QueueDequeue:
		old := r.values
		r.values = old[1:] // голова снята, остальные значения — ушли на ячейку влево
		hs := []Highlight{}
		// Все оставшиеся значения сместились влево: каждая их ячейка изменилась.
		for i := range r.values {
			hs = append(hs, Highlight{Index: i, Role: RoleShifted})
		}
		// Освободилась дальняя от головы ячейка — та, где раньше сидел хвост.
		if len(old) > 0 {
			hs = append(hs, Highlight{Index: len(old) - 1, Role: RoleRemoved})
		}
		r.highlights = hs

		noun := fmt.Sprintf("%d значений", len(r.values))
		if len(r.values) == 1 {
			noun = "1 значение"
		}
		r.frame(fmt.Sprintf("%s: сняли %d с головы (ячейка 0); оставшиеся %s сдвинулись влево — наивная dequeue стоит O(n).",
			r.operation, e.Value, noun))

	case structures.QueueOverflow:
		r.highlights = nil
		r.frame(fmt.Sprintf("%s: очередь полна (ёмкость %d) — переполнение, значение %d не записано.",
			r.operation, r.capacity, e.Value))
	}
}

// frame собирает самодостаточный кадр: копия содержимого + подсветки + указатели.
// Модель очереди — со сдвигом: после dequeue всё уезжает влево, голова всегда
// в ячейке 0, поэтому Head — 0 или -1, Tail — последняя занятая ячейка или -1.
// Шаги нумеруются с 1: стартовый кадр, затем по кадру на каждое событие.
func (r *queueFrames) frame(explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	length := len(r.values)
	head, tail := -1, -1
	if length > 0 {
		head, tail = 0, length-1
	}
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindStructure,
		Explanation: explanation,
		Data: StructureSnapshot{
			Structure:  StructureQueue,
			Capacity:   r.capacity,
			Values:     append([]int{}, r.values...), // не nil: пустое содержимое — [], не null
			Highlights: append([]Highlight{}, r.highlights...),
			Top:        -1, // у очереди указатель стека не показывается
			Head:       head,
			Tail:       tail,
			Operation:  r.operation,
		},
	})
}
