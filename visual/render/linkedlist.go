package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// StructureLinkedList — идентичность односвязного списка в снимке
// StructureSnapshot.
const StructureLinkedList = "linked-list"

// Роли подсветки односвязного списка (свои для этой структуры; роли
// фиксированных структур объявлены в stack.go и list.go).
const (
	RoleVisited  = "visited"  // узел посещён обходом (find/insert/remove по позиции)
	RoleRelinked = "relinked" // у узла переключена связь next
	RoleFound    = "found"    // узел с искомым значением найден
)

// Виды операций сценария односвязного списка.
const (
	LinkedOpPrepend = "prepend"
	LinkedOpAppend  = "append"
	LinkedOpInsert  = "insert"
	LinkedOpRemove  = "remove"
	LinkedOpFind    = "find"
)

// LinkedOp — одна операция сценария односвязного списка.
type LinkedOp struct {
	Kind  string // вид операции: LinkedOpPrepend / Append / Insert / Remove / Find
	Index int    // позиция для insert (0..Len) и remove (0..Len-1)
	Value int    // значение для prepend/append/insert/find
}

// LinkedRun — Frame Renderer односвязного списка: прогоняет сценарий
// операций над списком на указателях и отдаёт кадр на каждое событие.
// Сценарий должен быть валидным (индексы в пределах, remove только из
// непустого списка) — валидация входа живёт в маршруте. Список не имеет
// фиксированной ёмкости, поэтому ошибок конструктора здесь нет; отсутствие
// значения в find — успешный промах (-1), а не ошибка.
func LinkedRun(ops []LinkedOp, emit func(frame.Frame)) error {
	r := &linkedFrames{
		list:   structures.NewLinkedList(),
		values: []int{},
		emit:   emit,
	}

	r.frame(fmt.Sprintf("Создаём пустой односвязный список: узлы связаны указателями next, память не непрерывна. Впереди операций: %d.",
		len(ops)))

	for _, op := range ops {
		switch op.Kind {
		case LinkedOpPrepend:
			r.operation = fmt.Sprintf("prepend %d", op.Value)
			r.list.PrependWithEmitter(op.Value, r)
		case LinkedOpAppend:
			r.operation = fmt.Sprintf("append %d", op.Value)
			r.list.AppendWithEmitter(op.Value, r)
		case LinkedOpInsert:
			r.operation = fmt.Sprintf("insert %d %d", op.Index, op.Value)
			// Сценарий валидирован до Run: индекс вне 0..Len сюда не доходит.
			_ = r.list.InsertAtWithEmitter(op.Index, op.Value, r)
		case LinkedOpRemove:
			r.operation = fmt.Sprintf("remove %d", op.Index)
			// Сценарий валидирован до Run: удаление по несуществующему индексу
			// сюда не доходит.
			_, _ = r.list.RemoveAtWithEmitter(op.Index, r)
		case LinkedOpFind:
			r.operation = fmt.Sprintf("find %d", op.Value)
			// Промах — нормальный результат поиска: Find возвращает -1.
			_ = r.list.FindWithEmitter(op.Value, r)
		}
	}
	return nil
}

// LinkedFrames — все кадры сценария сразу (удобно для тестов).
func LinkedFrames(ops []LinkedOp) ([]frame.Frame, error) {
	var frames []frame.Frame
	if err := LinkedRun(ops, func(f frame.Frame) { frames = append(frames, f) }); err != nil {
		return nil, err
	}
	return frames, nil
}

// linkedFrames — накапливает состояние между событиями и строит кадры.
// Реализует algorithms.Emitter[structures.LinkedEvent]: каждое событие —
// один кадр (ADR-0002); визиты и переключения связей — тоже кадры, чтобы
// обход узел за узлом был виден глазами.
type linkedFrames struct {
	list       *structures.LinkedList
	values     []int       // зеркало логического порядка значений (кадр не зависит от будущего)
	operation  string      // имя текущей операции ("prepend 4", "find 7")
	highlights []Highlight // подсветки текущего шага
	step       int
	emit       func(frame.Frame)
}

// Emit — одно событие списка → один кадр (ADR-0002).
func (r *linkedFrames) Emit(ev structures.LinkedEvent) {
	switch e := ev.(type) {
	case structures.LinkedVisit:
		// Обход дошёл до узла; содержимое не изменилось, подсвечиваем узел.
		r.highlights = []Highlight{{Index: e.Index, Role: RoleVisited}}
		r.frame(fmt.Sprintf("%s: идём по цепочке next узел за узлом — посетили узел на позиции %d со значением %d.",
			r.operation, e.Index, e.Value))

	case structures.LinkedLink:
		// Связь next узла переключена; содержимое не изменилось.
		r.highlights = []Highlight{{Index: e.Index, Role: RoleRelinked}}
		r.frame(fmt.Sprintf("%s: переключили связь next у узла на позиции %d — сами узлы в памяти не двигаем.",
			r.operation, e.Index))

	case structures.LinkedInsert:
		// Узел встал на позицию e.Index: в логическом порядке добавляем
		// значение между соседями.
		old := r.values
		fresh := make([]int, 0, len(old)+1)
		fresh = append(fresh, old[:e.Index]...)
		fresh = append(fresh, e.Value)
		fresh = append(fresh, old[e.Index:]...)
		r.values = fresh
		r.highlights = []Highlight{{Index: e.Index, Role: RoleAdded}}

		switch {
		case r.startsWith(LinkedOpPrepend):
			r.frame(fmt.Sprintf("%s: добавили %d в голову — нового узла достаточно, обхода нет, O(1).",
				r.operation, e.Value))
		case r.startsWith(LinkedOpAppend):
			r.frame(fmt.Sprintf("%s: добавили %d в хвост — хвостовой узел вел прямо в конец, O(1).",
				r.operation, e.Value))
		default:
			r.frame(fmt.Sprintf("%s: вставили %d на позицию %d — сама вставка O(1), но переход к месту занял обход по узлам: суммарно O(n).",
				r.operation, e.Value, e.Index))
		}

	case structures.LinkedRemove:
		// Узел удалён с позиции e.Index: остальные узлы НЕ сдвигаются —
		// это не слайс, связь переключена через освободившееся место.
		old := r.values
		fresh := make([]int, 0, len(old)-1)
		fresh = append(fresh, old[:e.Index]...)
		fresh = append(fresh, old[e.Index+1:]...)
		r.values = fresh
		r.highlights = []Highlight{{Index: e.Index, Role: RoleRemoved}}

		if e.Index == 0 {
			r.frame(fmt.Sprintf("%s: удалили %d с головы — сдвигается только head, обхода нет, O(1).",
				r.operation, e.Value))
		} else {
			r.frame(fmt.Sprintf("%s: удалили %d с позиции %d — до удаления дошли обходом по узлам (O(n)), сами другие узлы не двигались.",
				r.operation, e.Value, e.Index))
		}

	case structures.LinkedFound:
		// Нашли узел с искомым значением и сразу остановились.
		r.highlights = []Highlight{{Index: e.Index, Role: RoleFound}}
		r.frame(fmt.Sprintf("%s: значение найдено на позиции %d — узлы до него обошли по одному, чем глубже узел, тем дольше поиск.",
			r.operation, e.Index))

	case structures.LinkedNotFound:
		// Прошли весь список и не нашли: успешный промах, -1 — не ошибка.
		r.highlights = []Highlight{}
		r.frame(fmt.Sprintf("%s: значения нет в списке — обошли все %d узлов, поиск вернул -1 (промах — нормальный результат).",
			r.operation, len(r.values)))
	}
}

// startsWith — вычисляет вид операции по имени текущей: помогает
// объяснению вставки отличать prepend/append от вставки по позиции,
// не ведя отдельное поле-вид.
func (r *linkedFrames) startsWith(kind string) bool {
	return len(r.operation) >= len(kind) && r.operation[:len(kind)] == kind
}

// frame собирает самодостаточный кадр: копия логического порядка +
// подсветки. Ёмкость = текущему числу узлов (0 — пустой список): пустых
// ячеек у списка не рисуется, как у списка на слайсе; внутреннее
// «устройство памяти» слайса тут неприменимо — узлы инспектируются по одному.
// Head указывает на узел 0, Tail — на последний (0..Len-1); у пустого
// списка оба -1. Top списку не принадлежит — стоит в -1.
// Шаги нумеруются с 1: стартовый кадр, затем по кадру на каждое событие.
func (r *linkedFrames) frame(explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	head, tail := -1, -1
	if len(r.values) > 0 {
		head, tail = 0, len(r.values)-1
	}
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindStructure,
		Explanation: explanation,
		Data: StructureSnapshot{
			Structure:  StructureLinkedList,
			Capacity:   len(r.values),
			Values:     append([]int{}, r.values...), // не nil: пустое содержимое — [], не null
			Highlights: append([]Highlight{}, r.highlights...),
			Top:        -1, // у списка указатель стека не показывается
			Head:       head,
			Tail:       tail,
			Operation:  r.operation,
		},
	})
}
