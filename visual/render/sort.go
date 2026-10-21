package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Роли подсветки затронутых позиций в SortSnapshot.
const (
	RoleCompare = "compare" // позиция сравниваемой сейчас пары (или элемента, сравниваемого с ключом)
	RoleSwapped = "swap"    // позиция обмениваемой сейчас пары
	RoleMin     = "min"     // позиция текущего минимума прохода (selection sort)
	RoleShift   = "shift"   // позиция, в которую сейчас сдвинулось значение (insertion sort)
	RoleInsert  = "insert"  // позиция, куда вставляется ключ (insertion sort)
)

// Сентинелы состояния вставки в SortSnapshot: шаг без удерживаемого ключа
// и «дырки» — стартовый кадр, кадры bubble/selection, кадр после вставки.
// Gap — индекс, поэтому -1 однозначен; Key — значение, а не индекс, его
// смысл определён только при Gap >= 0: без дырки значение рендером
// игнорируется.
const (
	noPendingKey = 0  // значение ключа при Gap = -1 (не рисуется)
	noGap        = -1 // дырки на шаге нет
)

// SortHighlight — подсветка одной позиции с ролью (что с ней случилось на шаге).
type SortHighlight struct {
	Index int    `json:"index"` // индекс затронутой позиции массива
	Role  string `json:"role"`  // роль: RoleCompare / RoleSwapped / RoleMin / RoleShift / RoleInsert
}

// SortSnapshot — снимок массива сортировки: Data кадра kind="sort".
// Самодостаточен (ADR-0002): весь массив копируется в каждый кадр, чтобы
// клиент рисовал шаг как есть, не накапливая состояние. Отдельный снимок
// и kind: снимок бинарного поиска (kind="array") не меняется.
type SortSnapshot struct {
	Values     []int           `json:"values"`     // весь массив на шаге (копия, не nil)
	Highlights []SortHighlight `json:"highlights"` // подсветки сравниваемых/обмениваемых/сдвигаемых позиций, минимума и вставки (пустые шаги — [])
	Operation  string          `json:"operation"`  // имя шага ("сравнение 0 1", "обмен 1 2", "сдвиг 0 → 1"); пусто — стартовый кадр
	// Gap — индекс «дырки»: пустой позиции, ожидающей удерживаемый ключ
	// (insertion sort, пока ключ вне массива). -1 — дырки на шаге нет.
	// При Gap >= 0 значение Values[Gap] — исходное значение ключа
	// до сдвигов, а после сдвигов — устаревшая копия сдвинутого значения;
	// в обоих случаях фронтенд его скрывает и рисует ячейку пустой —
	// видимое состояние вставки задают Gap и Key вместе, а не Values.
	Gap int `json:"gap"`
	// Key — значение удерживаемого ключа (insertion sort); осмыслен только
	// при Gap >= 0, иначе noPendingKey и рендером игнорируется.
	Key int `json:"key"`
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

// collectSortFrames — общий каркас XFrames-хелперов сортировок (других
// рендереров не покрывает): прогон Run со сбором кадров без стриминга
// (удобно для тестов).
func collectSortFrames(run func([]int, func(frame.Frame)), values []int) []frame.Frame {
	var frames []frame.Frame
	run(values, func(f frame.Frame) { frames = append(frames, f) })
	return frames
}

// BubbleFrames — все кадры сортировки сразу (удобно для тестов).
func BubbleFrames(values []int) []frame.Frame {
	return collectSortFrames(BubbleRun, values)
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
	return collectSortFrames(SelectionRun, values)
}

// InsertionRun — Frame Renderer insertion sort: запускает сортировку в режиме
// наблюдения и на каждое событие отдаёт кадр в emit. Вход уже валидирован
// маршрутом (лимит длины, целые числа), ошибок здесь нет.
func InsertionRun(values []int, emit func(frame.Frame)) {
	r := &insertionFrames{sortFrames: *newSortFrames(values, emit)}

	r.frame(fmt.Sprintf("Начинаем сортировку вставками: массив из %d элементов упорядочивается по возрастанию — берём очередной элемент (ключ) и вставляем его в отсортированный префикс, сдвигая строго большие значения вправо. Равные ключу элементы не сдвигаются, поэтому сортировка устойчива.",
		len(values)))
	sort.InsertionSortWithEmitter(values, r)
}

// InsertionFrames — все кадры сортировки сразу (удобно для тестов).
func InsertionFrames(values []int) []frame.Frame {
	return collectSortFrames(InsertionRun, values)
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
	key        int // значение удерживаемого ключа (insertion sort); осмыслен при gap >= 0
	gap        int // индекс «дырки» вставки (insertion sort); -1 — дырки нет
}

// newSortFrames — основа по исходному массиву: кадры не зависят
// от будущего шага, зеркало строится по событиям. Шаг без вставки
// (стартовый кадр, bubble/selection) не имеет ни ключа, ни дырки.
func newSortFrames(values []int, emit func(frame.Frame)) *sortFrames {
	return &sortFrames{
		mirror: append([]int{}, values...),
		emit:   emit,
		gap:    noGap,
	}
}

// frame собирает самодостаточный кадр: копия массива + подсветки
// + состояние вставки. Шаги нумеруются с 1: стартовый кадр, затем
// по кадру на каждое событие.
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
			Gap:        r.gap,
			Key:        r.key,
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

// Emit — одно событие алгоритма → один кадр (ADR-0002). Результат сравнения
// и минимум прохода берутся из события (NewMin/NewMinValue) как есть: кадр
// показывает тот минимум, о котором говорит пояснение. На сравнении видно
// пару «минимум / проверяемый кандидат», на обмене — пару «первый
// неотсортированный / найденный минимум».
func (r *selectionFrames) Emit(ev sort.SelectionEvent) {
	switch e := ev.(type) {
	case sort.SelectionCompare:
		r.operation = fmt.Sprintf("сравнение %d %d", e.Min, e.Scanned)
		if e.NewMin == e.Scanned {
			// Минимум перешёл на проверяемую позицию: она — новый минимум,
			// прежний минимум — участник сравнения.
			r.highlights = []SortHighlight{
				{Index: e.Min, Role: RoleCompare},
				{Index: e.NewMin, Role: RoleMin},
			}
			r.frame(fmt.Sprintf("Сравниваем позицию %d с текущим минимумом на позиции %d: %d меньше — минимум переходит на позицию %d (значение %d).",
				e.Scanned, e.Min, e.ScanValue, e.NewMin, e.NewMinValue))
		} else {
			r.highlights = []SortHighlight{
				{Index: e.NewMin, Role: RoleMin},
				{Index: e.Scanned, Role: RoleCompare},
			}
			r.frame(fmt.Sprintf("Сравниваем позицию %d с текущим минимумом на позиции %d: %d не меньше %d — минимум остаётся на позиции %d (значение %d).",
				e.Scanned, e.Min, e.ScanValue, e.MinValue, e.NewMin, e.NewMinValue))
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

// insertionFrames — обёртка-эмиттер событий insertion sort над общей основой.
type insertionFrames struct{ sortFrames }

// Emit — одно событие алгоритма → один кадр (ADR-0002). Зеркало меняется
// по детерминированным индексам событий Shift/Write; исход сравнения
// (Outcome) и состояние вставки (ключ, «дырка») берутся из события как
// есть — кадр показывает тот расклад, о котором говорит пояснение.
// Видимое состояние вставки кадр задаёт совместными полями Gap и Key:
// при Gap >= 0 значение Values[Gap] — исходное значение ключа до сдвигов,
// после — устаревшая копия: в обоих случаях рендер скрывает его и рисует
// ячейку пустой, а удерживаемый ключ виден в поле Key.
// На сравнении видна проверяемая позиция префикса, на сдвиге — приёмник
// значения (источник становится дыркой), на вставке — позиция ключа.
func (r *insertionFrames) Emit(ev sort.InsertionEvent) {
	switch e := ev.(type) {
	case sort.InsertionCompare:
		r.key, r.gap = e.Key, e.Gap
		r.highlights = []SortHighlight{{Index: e.Pos, Role: RoleCompare}}
		r.operation = fmt.Sprintf("сравнение %d с ключом", e.Pos)
		switch e.Outcome {
		case sort.InsertionGreater:
			r.frame(fmt.Sprintf("Сравниваем элемент на позиции %d (%d) с ключом %d: он больше ключа — следом сдвиг вправо.",
				e.Pos, e.PosValue, e.Key))
		case sort.InsertionEqual:
			r.frame(fmt.Sprintf("Сравниваем элемент на позиции %d (%d) с ключом %d: он равен ключу — не сдвигается, порядок равных сохраняется, ключ встанет правее.",
				e.Pos, e.PosValue, e.Key))
		case sort.InsertionLess:
			r.frame(fmt.Sprintf("Сравниваем элемент на позиции %d (%d) с ключом %d: он меньше ключа — сканирование остановлено, ключ встанет на пустую позицию %d.",
				e.Pos, e.PosValue, e.Key, e.Gap))
		}

	case sort.InsertionShift:
		r.mirror[e.To] = e.Value
		r.key, r.gap = e.Key, e.Gap
		r.highlights = []SortHighlight{{Index: e.To, Role: RoleShift}}
		r.operation = fmt.Sprintf("сдвиг %d → %d", e.From, e.To)
		r.frame(fmt.Sprintf("Сдвигаем %d с позиции %d на позицию %d: значение больше ключа %d — освобождаем ему место; пустая позиция %d ждёт ключ.",
			e.Value, e.From, e.To, e.Key, e.Gap))

	case sort.InsertionWrite:
		r.mirror[e.Index] = e.Value
		r.key, r.gap = noPendingKey, noGap
		r.highlights = []SortHighlight{{Index: e.Index, Role: RoleInsert}}
		r.operation = fmt.Sprintf("вставка на позицию %d", e.Index)
		if e.Moved {
			r.frame(fmt.Sprintf("Место для ключа найдено: вставляем %d на позицию %d — префикс снова отсортирован.",
				e.Value, e.Index))
		} else {
			r.frame(fmt.Sprintf("Ключ %d уже на месте (позиция %d): сдвиг не нужен — префикс остаётся упорядоченным.",
				e.Value, e.Index))
		}
	}
}
