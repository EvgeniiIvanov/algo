package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// StructureStack — идентичность стека в снимке StructureSnapshot.
const StructureStack = "stack"

// Роли подсветки затронутых позиций в StructureSnapshot.
const (
	RoleAdded   = "added"   // ячейка записана этой операцией
	RoleRemoved = "removed" // ячейка освобождена этой операцией
)

// Highlight — подсветка одной позиции с ролью (что с ней случилось на шаге).
type Highlight struct {
	Index int    `json:"index"` // индекс затронутой ячейки
	Role  string `json:"role"`  // роль: RoleAdded / RoleRemoved
}

// StructureSnapshot — снимок линейной структуры в кадре kind="structure"
// (ADR-0002). Один снимок на все структуры модуля; переключение рендера —
// по полю Structure. У стека кадр несёт и ёмкость: виден весь ряд ячеек,
// включая пустые — переполнение читается глазами.
type StructureSnapshot struct {
	Structure  string      `json:"structure"`  // идентичность структуры: "stack" (позже queue/list/linked-list)
	Capacity   int         `json:"capacity"`   // ёмкость структуры
	Values     []int       `json:"values"`     // содержимое от дна к вершине
	Highlights []Highlight `json:"highlights"` // подсветки затронутых позиций с ролью
	Top        int         `json:"top"`        // указатель top — индекс вершины; -1 — стек пуст
	Operation  string      `json:"operation"`  // имя операции шага ("push 5", "pop"); пусто — стартовый кадр
}

// StackOp — одна операция сценария стека: push со значением или pop.
type StackOp struct {
	IsPush bool // true — push значения Value, false — pop
	Value  int  // значение для push
}

// StackRun — Frame Renderer стека: прогоняет сценарий операций над стеком
// фиксированной ёмкости и на каждый шаг отдаёт кадр в emit.
// Сценарий должен быть валидным (pop только из непустого стека) — валидация
// входа живёт в маршруте; переполнение — легальное событие сценария.
// Отрицательная ёмкость — ошибка конструктора, кадры не создаются.
func StackRun(capacity int, ops []StackOp, emit func(frame.Frame)) error {
	stack, err := structures.New(capacity)
	if err != nil {
		return err
	}

	r := &stackFrames{
		stack:    stack,
		capacity: capacity,
		changed:  -1,
		emit:     emit,
	}

	r.frame(fmt.Sprintf("Создаём пустой стек фиксированной ёмкости %d. Впереди операций: %d.",
		capacity, len(ops)))

	for _, op := range ops {
		if op.IsPush {
			r.operation = fmt.Sprintf("push %d", op.Value)
			// Переполнение — не сбой Run, а учебное событие: кадр строит Emit.
			_ = r.stack.PushWithEmitter(op.Value, r)
			continue
		}
		r.operation = "pop"
		// Сценарий валидирован до Run: pop из пустого стека сюда не доходит.
		_, _ = r.stack.PopWithEmitter(r)
	}
	return nil
}

// StackFrames — все кадры сценария сразу (удобно для тестов).
func StackFrames(capacity int, ops []StackOp) ([]frame.Frame, error) {
	var frames []frame.Frame
	if err := StackRun(capacity, ops, func(f frame.Frame) { frames = append(frames, f) }); err != nil {
		return nil, err
	}
	return frames, nil
}

// stackFrames — накапливает состояние между событиями и строит кадры.
// Реализует algorithms.Emitter[structures.StackEvent].
type stackFrames struct {
	stack     *structures.Stack
	values    []int // зеркало содержимого из событий (кадр не зависит от будущего)
	capacity  int
	operation string // имя текущей операции ("push 5", "pop")
	changed   int    // индекс подсвеченной ячейки; -1 — на этом шаге нет
	role      string // роль подсветки (RoleAdded / RoleRemoved)
	step      int
	emit      func(frame.Frame)
}

// Emit — одно событие операции → один кадр (ADR-0002).
func (r *stackFrames) Emit(ev structures.StackEvent) {
	switch e := ev.(type) {
	case structures.StackPush:
		r.values = append(r.values, e.Value)
		r.changed, r.role = e.Index, RoleAdded
		r.frame(fmt.Sprintf("%s: положили %d в ячейку %d — теперь это вершина стека.",
			r.operation, e.Value, e.Index))

	case structures.StackPop:
		r.values = r.values[:e.Index]
		r.changed, r.role = e.Index, RoleRemoved
		r.frame(fmt.Sprintf("%s: сняли %d с вершины (ячейка %d освобождена).",
			r.operation, e.Value, e.Index))

	case structures.StackOverflow:
		r.changed, r.role = -1, ""
		r.frame(fmt.Sprintf("%s: стек полон (ёмкость %d) — переполнение, значение %d не записано.",
			r.operation, r.capacity, e.Value))
	}
}

// frame собирает самодостаточный кадр: копия содержимого + подсветки + top.
// Шаги нумеруются с 1: стартовый кадр, затем по кадру на каждое событие.
func (r *stackFrames) frame(explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	highlights := []Highlight{}
	if r.changed >= 0 {
		highlights = append(highlights, Highlight{Index: r.changed, Role: r.role})
	}
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindStructure,
		Explanation: explanation,
		Data: StructureSnapshot{
			Structure:  StructureStack,
			Capacity:   r.capacity,
			Values:     append([]int{}, r.values...), // не nil: пустое содержимое — [], не null
			Highlights: highlights,
			Top:        len(r.values) - 1,
			Operation:  r.operation,
		},
	})
}
