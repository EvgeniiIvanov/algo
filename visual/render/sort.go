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
	RoleMin     = "min"     // позиция текущего минимума прохода (selection sort)
)

// SortHighlight — подсветка одной позиции с ролью (что с ней случилось на шаге).
type SortHighlight struct {
	Index int    `json:"index"` // индекс затронутой позиции массива
	Role  string `json:"role"`  // роль: RoleCompare / RoleSwapped / RoleMin
}

// SortSnapshot — снимок массива сортировки: Data кадра kind="sort".
// Самодостаточен (ADR-0002): весь массив копируется в каждый кадр, чтобы
// клиент рисовал шаг как есть, не накапливая состояние. Отдельный снимок
// и kind: снимок бинарного поиска (kind="array") не меняется.
type SortSnapshot struct {
	Values     []int           `json:"values"`     // весь массив на шаге (копия, не nil)
	Highlights []SortHighlight `json:"highlights"` // подсветки сравниваемых/обмениваемых позиций и минимума (пустые шаги — [])
	Operation  string          `json:"operation"`  // имя шага ("сравнение 0 1", "обмен 1 2"); пусто — стартовый кадр
}

// BubbleRun — Frame Renderer bubble sort: запускает сортировку в режиме
// наблюдения и на каждое событие отдаёт кадр в emit. Вход уже валидирован
// маршрутом (лимит длины, целые числа), ошибок здесь нет.
func BubbleRun(values []int, emit func(frame.Frame)) {
	r := &bubbleFrames{sortFrames: *newSortFrames(values, emit)}

	r.frame(fmt.Sprintf("Начинаем сортировку пузырьком: массив из %d элементов упорядочивается по возрастанию обменами соседних пар. Уже упорядоченные соседи не обмениваются, поэтому после прохода без обменов сортировка завершается досрочно.",
		len(values)))

	sort.BubbleSortWithEmitter(values, r)
}

// BubbleFrames — все кадры сортировки сразу (удобно для тестов).
func BubbleFrames(values []int) []frame.Frame {
	var frames []frame.Frame
	BubbleRun(values, func(f frame.Frame) { frames = append(frames, f) })
	return frames
}

// SelectionRun — Frame Renderer selection sort: запускает сортировку в режиме
// наблюдения и на каждое событие отдаёт кадр в emit. Вход уже валидирован
// маршрутом (лимит длины, целые числа), ошибок здесь нет.
func SelectionRun(values []int, emit func(frame.Frame)) {
	r := &selectionFrames{sortFrames: *newSortFrames(values, emit)}

	r.frame(fmt.Sprintf("Начинаем сортировку выбором: массив из %d элементов упорядочивается по возрастанию — на каждом проходе ищем минимум неотсортированной части и меняем его местами с её первым элементом. Стандартный вариант неустойчив: равные минимуму элементы могут менять порядок.",
		len(values)))

	sort.SelectionSortWithEmitter(values, r)
}

// SelectionFrames — все кадры сортировки сразу (удобно для тестов).
func SelectionFrames(values []int) []frame.Frame {
	var frames []frame.Frame
	SelectionRun(values, func(f frame.Frame) { frames = append(frames, f) })
	return frames
}

// sortFrames — общая основа рендереров сортировок: зеркало массива,
// подсветки и имя текущего шага, сборка самодостаточного кадра.
// События конкретных алгоритмов разбирают обёртки ниже — у каждой свой
// Emit, по одному событию кадр (ADR-0002).
type sortFrames struct {
	mirror     []int           // массив по событиям (чистое зеркало, кадр за шагом)
	highlights []SortHighlight // подсветки текущего шага
	operation  string          // имя текущего шага
	step       int
	emit       func(frame.Frame)
}

// newSortFrames — основа по исходному массиву: кадры не зависят
// от будущего шага, зеркало строится по событиям.
func newSortFrames(values []int, emit func(frame.Frame)) *sortFrames {
	return &sortFrames{
		mirror: append([]int{}, values...),
		emit:   emit,
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

// bubbleFrames — обёртка-эмиттер событий bubble sort над общей основой.
type bubbleFrames struct{ sortFrames }

// Emit — одно событие алгоритма → один кадр (ADR-0002). Зеркало меняется
// по детерминированным индексам события Swap; values и подсветки в кадре
// всегда согласованы.
func (r *bubbleFrames) Emit(ev sort.BubbleEvent) {
	switch e := ev.(type) {
	case sort.BubbleCompare:
		r.highlights = []SortHighlight{
			{Index: e.Left, Role: RoleCompare},
			{Index: e.Right, Role: RoleCompare},
		}
		r.operation = fmt.Sprintf("сравнение %d %d", e.Left, e.Right)
		if e.LeftValue > e.RightValue {
			r.frame(fmt.Sprintf("Сравниваем позиции %d и %d: %d больше %d — пара убывает, следом обмен.",
				e.Left, e.Right, e.LeftValue, e.RightValue))
		} else {
			r.frame(fmt.Sprintf("Сравниваем позиции %d и %d: %d не больше %d — порядок сохраняется, обмена нет.",
				e.Left, e.Right, e.LeftValue, e.RightValue))
		}

	case sort.BubbleSwap:
		r.mirror[e.Left], r.mirror[e.Right] = e.RightValue, e.LeftValue
		r.highlights = []SortHighlight{
			{Index: e.Left, Role: RoleSwapped},
			{Index: e.Right, Role: RoleSwapped},
		}
		r.operation = fmt.Sprintf("обмен %d %d", e.Left, e.Right)
		r.frame(fmt.Sprintf("Обменяли позиции %d и %d: %d и %d поменялись местами.",
			e.Left, e.Right, e.LeftValue, e.RightValue))
	}
}

// selectionFrames — обёртка-эмиттер событий selection sort над общей основой.
type selectionFrames struct{ sortFrames }

// Emit — одно событие алгоритма → один кадр (ADR-0002). На сравнении видно
// два индекса — текущий минимум (role "min") и проверяемого кандидата;
// на обмене — пара «первый неотсортированный / найденный минимум».
func (r *selectionFrames) Emit(ev sort.SelectionEvent) {
	switch e := ev.(type) {
	case sort.SelectionCompare:
		r.highlights = []SortHighlight{
			{Index: e.Min, Role: RoleMin},
			{Index: e.Scanned, Role: RoleCompare},
		}
		r.operation = fmt.Sprintf("сравнение %d %d", e.Min, e.Scanned)
		if e.ScanValue < e.MinValue {
			r.frame(fmt.Sprintf("Сравниваем позицию %d с текущим минимумом на позиции %d: %d меньше — минимум переходит на позицию %d.",
				e.Scanned, e.Min, e.ScanValue, e.Scanned))
		} else {
			r.frame(fmt.Sprintf("Сравниваем позицию %d с текущим минимумом на позиции %d: %d не меньше %d — минимум остаётся на позиции %d.",
				e.Scanned, e.Min, e.ScanValue, e.MinValue, e.Min))
		}

	case sort.SelectionSwap:
		r.mirror[e.Pass], r.mirror[e.Min] = e.MinValue, e.PassValue
		r.highlights = []SortHighlight{
			{Index: e.Pass, Role: RoleSwapped},
			{Index: e.Min, Role: RoleSwapped},
		}
		r.operation = fmt.Sprintf("обмен %d %d", e.Pass, e.Min)
		r.frame(fmt.Sprintf("Минимум остатка найден: меняем позиции %d и %d — значения %d и %d поменялись местами.",
			e.Pass, e.Min, e.PassValue, e.MinValue))
	}
}
