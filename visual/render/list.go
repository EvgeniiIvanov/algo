package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// StructureList — идентичность списка в снимке StructureSnapshot.
const StructureList = "list"

// ListOp — одна операция сценария списка: вставка значения на индекс
// или удаление по индексу.
type ListOp struct {
	IsInsert bool // true — вставка Value на позицию Index, false — удаление по Index
	Index    int  // индекс операции (вставка: 0..Len; удаление: 0..Len-1)
	Value    int  // значение для вставки
}

// ListRun — Frame Renderer списка на слайсе: прогоняет сценарий операций
// над растущим списком и на каждый шаг отдаёт кадр в emit. Сценарий должен
// быть валидным (индексы в пределах текущей длины) — валидация входа живёт
// в маршруте. Список не имеет фиксированной ёмкости, поэтому ошибок
// конструктора и переполнения здесь нет: вставка в хвост разрешена всегда.
func ListRun(ops []ListOp, emit func(frame.Frame)) error {
	r := &listFrames{
		list:   structures.NewList(),
		values: []int{},
		emit:   emit,
	}

	r.frame(fmt.Sprintf("Создаём пустой список: растёт без фиксированной ёмкости. Впереди операций: %d.",
		len(ops)))

	for _, op := range ops {
		if op.IsInsert {
			r.operation = fmt.Sprintf("insert %d %d", op.Index, op.Value)
			// Сценарий валидирован до Run: индекс вне 0..Len сюда не доходит.
			_ = r.list.InsertWithEmitter(op.Index, op.Value, r)
			continue
		}
		r.operation = fmt.Sprintf("remove %d", op.Index)
		// Сценарий валидирован до Run: удаление по несуществующему индексу
		// сюда не доходит.
		_, _ = r.list.RemoveAtWithEmitter(op.Index, r)
	}
	return nil
}

// ListFrames — все кадры сценария сразу (удобно для тестов).
func ListFrames(ops []ListOp) ([]frame.Frame, error) {
	var frames []frame.Frame
	if err := ListRun(ops, func(f frame.Frame) { frames = append(frames, f) }); err != nil {
		return nil, err
	}
	return frames, nil
}

// listFrames — накапливает состояние между событиями и строит кадры.
// Реализует algorithms.Emitter[structures.ListEvent].
type listFrames struct {
	list       *structures.List
	values     []int       // зеркало содержимого из событий (кадр не зависит от будущего)
	operation  string      // имя текущей операции ("insert 1 9", "remove 0")
	highlights []Highlight // подсветки текущего шага
	step       int
	emit       func(frame.Frame)
}

// Emit — одно событие операции → один кадр (ADR-0002). Подсветки строятся
// из зеркала значений и счётчиков сдвига события: сдвинувшиеся значения
// определяются по их текущим ячейкам, не по внутренностям слайса.
func (r *listFrames) Emit(ev structures.ListEvent) {
	switch e := ev.(type) {
	case structures.ListInsert:
		// Вставка на позицию e.Index: значения с e.Index... уехали на ячейку вправо.
		old := r.values
		fresh := make([]int, 0, len(old)+1)
		fresh = append(fresh, old[:e.Index]...)
		fresh = append(fresh, e.Value)
		fresh = append(fresh, old[e.Index:]...)
		r.values = fresh

		hs := []Highlight{{Index: e.Index, Role: RoleAdded}}
		for i := e.Index + 1; i <= e.Index+e.Shifted; i++ {
			// Ячейки со вставленными правее старыми значениями.
			hs = append(hs, Highlight{Index: i, Role: RoleShifted})
		}
		r.highlights = hs

		if e.Shifted == 0 {
			r.frame(fmt.Sprintf("%s: вставили %d в конец (ячейка %d) — хвост, сдвигов нет.",
				r.operation, e.Value, e.Index))
		} else {
			// Позиция может быть любой — голова, середина, вставка перед
			// существующими: важно само событие со сдвигом, не место в списке.
			noun, verb := shiftedWords(e.Shifted)
			r.frame(fmt.Sprintf("%s: вставили %d на позицию %d; %s %s вправо — операция со сдвигом стоит O(n).",
				r.operation, e.Value, e.Index, noun, verb))
		}

	case structures.ListRemove:
		// Удаление с позиции e.Index: значения правее уехали на ячейку влево.
		old := r.values
		fresh := make([]int, 0, len(old)-1)
		fresh = append(fresh, old[:e.Index]...)
		fresh = append(fresh, old[e.Index+1:]...)
		r.values = fresh

		hs := []Highlight{}
		for i := e.Index; i < e.Index+e.Shifted; i++ {
			// Сдвинувшиеся значения теперь в ячейках e.Index..e.Index+Shifted-1.
			hs = append(hs, Highlight{Index: i, Role: RoleShifted})
		}
		// Освободилась дальняя от головы ячейка — так, где раньше сидел хвост.
		hs = append(hs, Highlight{Index: len(old) - 1, Role: RoleRemoved})
		r.highlights = hs

		if e.Shifted == 0 {
			r.frame(fmt.Sprintf("%s: удалили %d (позиция %d, ячейка %d освобождена) — удалили последний элемент, сдвигов нет.",
				r.operation, e.Value, e.Index, len(old)-1))
		} else {
			noun, verb := shiftedWords(e.Shifted)
			r.frame(fmt.Sprintf("%s: удалили %d (позиция %d); %s %s влево — операция со сдвигом стоит O(n).",
				r.operation, e.Value, e.Index, noun, verb))
		}
	}
}

// shiftedWords — по-русски согласованная пара «N значений/значения/значение»
// с глаголом «сдвинулись/сдвинулось»: счётчик сдвига события переводится в
// точную фразу без привязки к месту операции в списке.
func shiftedWords(n int) (noun, verb string) {
	if n%100 >= 11 && n%100 <= 14 {
		return fmt.Sprintf("%d значений", n), "сдвинулись"
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%d значение", n), "сдвинулось"
	case 2, 3, 4:
		return fmt.Sprintf("%d значения", n), "сдвинулись"
	default:
		return fmt.Sprintf("%d значений", n), "сдвинулись"
	}
}

// frame собирает самодостаточный кадр: копия содержимого + подсветки.
// У растущего списка фиксированной ёмкости нет: Capacity несёт текущее число
// заполненных ячеек (= len(values), 0 — пустой список); внутреннюю ёмкость
// слайса по спеке не показываем, пустых ячеек в кадре не рисуем. Указатели
// top/head/tail списку не нужны — все три стоят в -1.
// Шаги нумеруются с 1: стартовый кадр, затем по кадру на каждое событие.
func (r *listFrames) frame(explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindStructure,
		Explanation: explanation,
		Data: StructureSnapshot{
			Structure:  StructureList,
			Capacity:   len(r.values),
			Values:     append([]int{}, r.values...), // не nil: пустое содержимое — [], не null
			Highlights: append([]Highlight{}, r.highlights...),
			Top:        -1, // у списка указатели других структур не показываются
			Head:       -1,
			Tail:       -1,
			Operation:  r.operation,
		},
	})
}
