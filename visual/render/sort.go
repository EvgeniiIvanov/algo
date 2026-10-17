package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Роли подсветки затронутых позиций в SortSnapshot.
const (
	RoleCompare = "compare" // позиция сравниваемой сейчас пары
	RoleSwapped = "swap"    // позиция обмениваемой сейчас пары
)

// SortHighlight — подсветка одной позиции с ролью (что с ней случилось на шаге).
type SortHighlight struct {
	Index int    `json:"index"` // индекс затронутой позиции массива
	Role  string `json:"role"`  // роль: RoleCompare / RoleSwapped
}

// SortSnapshot — снимок массива сортировки: Data кадра kind="sort".
// Самодостаточен (ADR-0002): весь массив копируется в каждый кадр, чтобы
// клиент рисовал шаг как есть, не накапливая состояние. Отдельный снимок
// и kind: снимок бинарного поиска (kind="array") не меняется.
type SortSnapshot struct {
	Values     []int           `json:"values"`     // весь массив на шаге (копия, не nil)
	Highlights []SortHighlight `json:"highlights"` // подсветки сравниваемых/обмениваемых позиций (пустые шаги — [])
	Operation  string          `json:"operation"`  // имя шага ("compare 0 1", "swap 1 2"); пусто — стартовый кадр
}

// BubbleRun — Frame Renderer bubble sort: запускает сортировку в режиме
// наблюдения и на каждое событие отдаёт кадр в emit. Вход уже валидирован
// маршрутом (лимит длины, целые числа), ошибок здесь нет.
func BubbleRun(values []int, emit func(frame.Frame)) {
	r := &sortFrames{
		// Зеркало массива из событий: кадры не зависят от будущего шага.
		mirror: append([]int{}, values...),
		emit:   emit,
	}

	r.frame(fmt.Sprintf("Начинаем bubble sort: массив из %d элементов упорядочивается по возрастанию обменами соседних пар. Уже упорядоченные соседи не обмениваются, поэтому после прохода без обменов сортировка завершается досрочно.",
		len(values)))

	sort.BubbleSortWithEmitter(values, r)
}

// BubbleFrames — все кадры сортировки сразу (удобно для тестов).
func BubbleFrames(values []int) []frame.Frame {
	var frames []frame.Frame
	BubbleRun(values, func(f frame.Frame) { frames = append(frames, f) })
	return frames
}

// sortFrames — накапливает состояние между событиями и строит кадры.
// Реализует algorithms.Emitter[sort.Event].
type sortFrames struct {
	mirror     []int           // массив по событиям (чистое зеркало, кадр за шагом)
	highlights []SortHighlight // подсветки текущего шага
	operation  string          // имя текущего шага
	step       int
	emit       func(frame.Frame)
}

// Emit — одно событие алгоритма → один кадр (ADR-0002). Зеркало меняется
// по детерминированным индексам события Swap; values и подсветки в кадре
// всегда согласованы.
func (r *sortFrames) Emit(ev sort.Event) {
	switch e := ev.(type) {
	case sort.Compare:
		r.highlights = []SortHighlight{
			{Index: e.Left, Role: RoleCompare},
			{Index: e.Right, Role: RoleCompare},
		}
		r.operation = fmt.Sprintf("compare %d %d", e.Left, e.Right)
		if e.LeftValue > e.RightValue {
			r.frame(fmt.Sprintf("Сравниваем позиции %d и %d: %d больше %d — пара убывает, следом обмен.",
				e.Left, e.Right, e.LeftValue, e.RightValue))
		} else {
			r.frame(fmt.Sprintf("Сравниваем позиции %d и %d: %d не больше %d — порядок сохраняется, обмена нет.",
				e.Left, e.Right, e.LeftValue, e.RightValue))
		}

	case sort.Swap:
		r.mirror[e.Left], r.mirror[e.Right] = e.RightValue, e.LeftValue
		r.highlights = []SortHighlight{
			{Index: e.Left, Role: RoleSwapped},
			{Index: e.Right, Role: RoleSwapped},
		}
		r.operation = fmt.Sprintf("swap %d %d", e.Left, e.Right)
		r.frame(fmt.Sprintf("Обменяли позиции %d и %d: %d и %d поменялись местами.",
			e.Left, e.Right, e.LeftValue, e.RightValue))
	}
}

// frame собирает самодостаточный кадр: копия массива + подсветки.
// Шаги нумеруются с 1: стартовый кадр, затем по кадру на каждое событие.
func (r *sortFrames) frame(explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindSort,
		Explanation: explanation,
		Data: SortSnapshot{
			Values:     append([]int{}, r.mirror...), // не nil: пустое содержимое — [], не null
			Highlights: append([]SortHighlight{}, r.highlights...),
			Operation:  r.operation,
		},
	})
}
