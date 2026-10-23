package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Роли подсветки затронутых позиций в SortSnapshot.
const (
	RoleCompare  = "compare"  // позиция сравниваемой сейчас пары (или элемента, сравниваемого с ключом/pivot)
	RoleSwapped  = "swap"     // позиция обмениваемой сейчас пары
	RoleMin      = "min"      // позиция текущего минимума прохода (selection sort)
	RoleShift    = "shift"    // позиция, в которую сейчас сдвинулось значение (insertion sort)
	RoleInsert   = "insert"   // позиция, куда вставляется ключ (insertion sort)
	RoleWrite    = "write"    // позиция, в которую сейчас записано значение при слиянии (merge sort)
	RolePivot    = "pivot"    // позиция pivot: выбранного на partition и занимаемой финальной позиции (quicksort)
	RoleBoundary = "boundary" // позиция границы i — конца блока строго меньших значений (quicksort)
)

// SortRange — активный диапазон шага сортировки: [Lo, Hi) сейчас
// используется алгоритмом. Merge sort несёт Mid — границу половин
// слияния; quicksort несёт диапазоны partition [Lo, Hi) без Mid
// (у partition нет границы половин — nil). Другие сортировки активных
// диапазонов не несут.
type SortRange struct {
	Lo  int  `json:"lo"`            // начало активного диапазона
	Mid *int `json:"mid,omitempty"` // граница половин слияния (merge sort); nil — у partition границы половин нет (quicksort)
	Hi  int  `json:"hi"`            // исключительная граница активного диапазона
}

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
	Role  string `json:"role"`  // роль подсветки: RoleCompare / RoleSwapped / RoleMin / RoleShift / RoleInsert / RoleWrite / RolePivot / RoleBoundary

}

// SortSnapshot — снимок массива сортировки: Data кадра kind="sort".
// Самодостаточен (ADR-0002): весь массив копируется в каждый кадр, чтобы
// клиент рисовал шаг как есть, не накапливая состояние. Отдельный снимок
// и kind: снимок бинарного поиска (kind="array") не меняется.
type SortSnapshot struct {
	Values     []int           `json:"values"`     // весь массив на шаге (копия, не nil)
	Highlights []SortHighlight `json:"highlights"` // подсветки сравниваемых/обмениваемых/сдвигаемых позиций, минимума и вставки (пустые шаги — [])
	Operation  string          `json:"operation"`  // имя шага ("сравнение 0 1", "обмен 1 2", "сдвиг 0 → 1"); пусто — стартовый кадр
	// Ranges — активные диапазоны шага (merge sort: сливаемый отрезок
	// [Lo, Hi) с границей половин Mid; quicksort: активный диапазон
	// partition [Lo, Hi) без Mid); у сортировок без активного диапазона
	// — пустой список. Поле обратно совместимо:
	// прежние снимки без ranges читаются фронтом как пустые.
	Ranges []SortRange `json:"ranges"`
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
	key        int         // значение удерживаемого ключа (insertion sort); осмыслен при gap >= 0
	gap        int         // индекс «дырки» вставки (insertion sort); -1 — дырки нет
	ranges     []SortRange // активный диапазон шага (merge sort — слияние, quicksort — partition); у остальных пустой
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
			Ranges:     append([]SortRange{}, r.ranges...),
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

// MergeRun — Frame Renderer merge sort: запускает сортировку в режиме
// наблюдения и на каждое событие отдаёт кадр в emit. Вход уже валидирован
// маршрутом (лимит длины, целые числа), ошибок здесь нет.
func MergeRun(values []int, emit func(frame.Frame)) {
	r := &mergeFrames{sortFrames: *newSortFrames(values, emit)}

	r.frame(fmt.Sprintf("Начинаем сортировку слиянием: массив из %d элементов упорядочивается по возрастанию — рекурсия делит диапазон пополам, сортирует половины и сливает их через буфер. При равных значениях первым берётся элемент левой половины, поэтому слияние устойчиво; дополнительная память — один буфер размера массива.",
		len(values)))

	sort.MergeSortWithEmitter(values, r)
}

// MergeFrames — все кадры сортировки сразу (удобно для тестов).
func MergeFrames(values []int) []frame.Frame {
	return collectSortFrames(MergeRun, values)
}

// mergeFrames — обёртка-эмиттер событий merge sort над общей основой.
type mergeFrames struct{ sortFrames }

// mergeActiveRange — разметка активного диапазона слияния из события алгоритма
// для снимка: одна именованная конструкция тройки lo/mid/hi вместо её
// повторения в каждой ветке рендерера. Ставится
// на кадрах, показывающих источник — половины слияния (деление,
// сравнение); кадр записи его не несёт — он рисует приёмник.
func mergeActiveRange(r sort.MergeRange) []SortRange {
	mid := r.Mid
	return []SortRange{{Lo: r.Lo, Mid: &mid, Hi: r.Hi}}
}

// Emit — одно событие алгоритма → один кадр (ADR-0002). Зеркало меняется
// только по событиям записи (MergeWrite): деление и сравнения читают ещё
// не перестроенные половины. Активный диапазон слияния — в поле ranges
// снимка; кадр деления показывает обе половины диапазона разметкой
// in-left/in-right; сравнение подсвечивает обе сравниваемые позиции;
// запись подсвечивает позицию, в которой значение появилось в массиве,
// и не размечает половины: выход слияния — приёмник, а не диапазон-источник,
// и метки левой/правой половины ввели бы в заблуждение (FromLeft объясняет
// источник текстом пошагового пояснения).
func (r *mergeFrames) Emit(ev sort.MergeEvent) {
	switch e := ev.(type) {
	case sort.MergeSplit:
		// Кадр деления: зеркало не меняется — половины ещё не сортированы.
		// Обе половины видны как левая/правая разметка одного активного
		// диапазона: ячейки [Lo, Mid) получают in-left, [Mid, Hi) — in-right.
		r.ranges = mergeActiveRange(e.MergeRange)
		r.highlights = nil
		r.operation = fmt.Sprintf("деление %d..%d", e.Lo, e.Hi-1)
		r.frame(fmt.Sprintf("Делим диапазон [%d, %d) пополам: левая половина [%d, %d), правая [%d, %d). Каждую половину алгоритм отсортирует той же рекурсией, затем отсортированные половины будут слиты в порядке возрастания.",
			e.Lo, e.Hi, e.Lo, e.Mid, e.Mid, e.Hi))

	case sort.MergeCompare:
		r.ranges = mergeActiveRange(e.MergeRange)
		r.highlights = []SortHighlight{
			{Index: e.Left, Role: RoleCompare},
			{Index: e.Right, Role: RoleCompare},
		}
		r.operation = fmt.Sprintf("сравнение %d %d", e.Left, e.Right)
		if e.LeftChosen && e.LeftValue == e.RightValue {
			r.frame(fmt.Sprintf("Сравниваем головы половин при слиянии [%d, %d): %d на позиции %d и %d на позиции %d равны — берём значение из левой половины, порядок равных сохраняется (устойчивость); оно пойдёт на позицию %d.",
				e.Lo, e.Hi, e.LeftValue, e.Left, e.RightValue, e.Right, e.Dest))
		} else if e.LeftChosen {
			r.frame(fmt.Sprintf("Сравниваем головы половин при слиянии [%d, %d): %d на позиции %d не больше %d на позиции %d — берём значение из левой половины, оно пойдёт на позицию %d.",
				e.Lo, e.Hi, e.LeftValue, e.Left, e.RightValue, e.Right, e.Dest))
		} else {
			r.frame(fmt.Sprintf("Сравниваем головы половин при слиянии [%d, %d): %d на позиции %d больше %d на позиции %d — берём значение из правой половины, оно пойдёт на позицию %d.",
				e.Lo, e.Hi, e.LeftValue, e.Left, e.RightValue, e.Right, e.Dest))
		}

	case sort.MergeWrite:
		r.mirror[e.Index] = e.Value
		// Разметка половин не ставится: позиция записи — приёмник слияния,
		// а не левая/правая половина источника (см. комментарий Emit выше).
		r.ranges = nil
		r.highlights = []SortHighlight{{Index: e.Index, Role: RoleWrite}}
		r.operation = fmt.Sprintf("запись %d", e.Index)
		side := "правой"
		if e.FromLeft {
			side = "левой"
		}
		r.frame(fmt.Sprintf("Записываем %d на позицию %d: значение взято из %s половины слияния [%d, %d) — сливаемые половины отсортированы, их значения идут в массив в порядке очерёдности.",
			e.Value, e.Index, side, e.Lo, e.Hi))
	}
}

// QuickSortRun — Frame Renderer quicksort: запускает сортировку в режиме
// наблюдения и на каждое событие отдаёт кадр в emit. Вход уже валидирован
// маршрутом (лимит длины, целые числа), ошибок здесь нет.
func QuickSortRun(values []int, emit func(frame.Frame)) {
	r := &quickFrames{sortFrames: *newSortFrames(values, emit)}

	r.frame(fmt.Sprintf("Начинаем быструю сортировку: массив из %d элементов упорядочивается по возрастанию — рекурсия работает по включительным диапазонам lo..hi, выбирает в каждом pivot (последний элемент диапазона) и раскладывает значения относительно него (partition по схеме Lomuto): строго меньшие уходят влево, затем pivot встаёт на финальную позицию между частями. Обмены при partition и финальном размещении pivot могут менять относительный порядок равных значений, поэтому сортировка неустойчива; среднее время O(n log n), худшее — O(n²), в частности на уже упорядоченных входах.",
		len(values)))

	sort.QuickSortWithEmitter(values, r)
}

// QuickFrames — все кадры сортировки сразу (удобно для тестов).
func QuickFrames(values []int) []frame.Frame {
	return collectSortFrames(QuickSortRun, values)
}

// quickFrames — обёртка-эмиттер событий quicksort над общей основой.
type quickFrames struct{ sortFrames }

// quickActiveRange — разметка активного диапазона partition [lo..hi]
// включительно из события quicksort для снимка: SortRange хранит
// исключительную верхнюю границу, поэтому hi переводится в hi+1.
// Границы половин (Mid) у partition нет — поле остаётся nil.
func quickActiveRange(r sort.QuickRange) []SortRange {
	return []SortRange{{Lo: r.Lo, Hi: r.Hi + 1}}
}

// Emit — одно событие алгоритма → один кадр (ADR-0002). Зеркало меняется
// по детерминированным индексам событий Swap/Placed: pivot за сканирование
// не двигается, строго меньшие значения уходят влево. Активный диапазон
// partition — в поле ranges снимка (без границы половин); pivot подсвечен
// на всех кадрах partition, граница блока строго меньших — на кадрах
// сравнения и обмена; кадр размещения pivot диапазона не несёт:
// финальная позиция pivot — граница уже завершённой partition.
func (r *quickFrames) Emit(ev sort.QuickEvent) {
	switch e := ev.(type) {
	case sort.QuickPivot:
		// Кадр выбора pivot: зеркало не меняется — сканирование ещё не шло.
		r.ranges = quickActiveRange(e.QuickRange)
		r.highlights = []SortHighlight{{Index: e.Hi, Role: RolePivot}}
		r.operation = fmt.Sprintf("pivot %d", e.Hi)
		r.frame(fmt.Sprintf("Выбираем pivot для partition [%d..%d]: значение %d на позиции %d — последний элемент диапазона. Строго меньшие значения соберутся слева, не меньшие — справа.",
			e.Lo, e.Hi, e.PivotValue, e.Hi))

	case sort.QuickCompare:
		// Кадр сравнения: элемент j против pivot; если блок строго меньших
		// уже начался, видна и текущая граница i.
		r.ranges = quickActiveRange(e.QuickRange)
		r.highlights = []SortHighlight{
			{Index: e.J, Role: RoleCompare},
			{Index: e.Hi, Role: RolePivot},
		}
		if e.Boundary >= e.Lo {
			r.highlights = append(r.highlights, SortHighlight{Index: e.Boundary, Role: RoleBoundary})
		}
		r.operation = fmt.Sprintf("сравнение %d с pivot", e.J)
		if e.JValue < e.PivotValue {
			r.frame(fmt.Sprintf("Сравниваем позицию %d (%d) с pivot %d: значение строго меньше — оно окажется в левой части partition [%d..%d], следом перенос в блок меньших.",
				e.J, e.JValue, e.PivotValue, e.Lo, e.Hi))
		} else {
			r.frame(fmt.Sprintf("Сравниваем позицию %d (%d) с pivot %d: значение не меньше pivot — оно остаётся в правой части partition [%d..%d] (строгое сравнение: равные pivot значения уходят вправо).",
				e.J, e.JValue, e.PivotValue, e.Lo, e.Hi))
		}

	case sort.QuickSwap:
		// Кадр обмена: строго меньшее значение с позиции j встаёт на
		// границу i, прежнее значение i уходит на j.
		r.mirror[e.Left], r.mirror[e.Right] = e.RightValue, e.LeftValue
		r.ranges = quickActiveRange(e.QuickRange)
		r.highlights = []SortHighlight{
			{Index: e.Left, Role: RoleBoundary},
			{Index: e.Right, Role: RoleSwapped},
			{Index: e.Hi, Role: RolePivot},
		}
		r.operation = fmt.Sprintf("обмен %d %d", e.Left, e.Right)
		r.frame(fmt.Sprintf("Обмен при partition [%d..%d]: строго меньшее %d уходит на позицию %d — блок меньших вырос до неё, а %d переезжает на позицию %d.",
			e.Lo, e.Hi, e.RightValue, e.Left, e.LeftValue, e.Right))

	case sort.QuickPlaced:
		// Кадр размещения pivot на финальной позиции Index: значение,
		// стоявшее там до обмена, переезжает на позицию pivot (Hi);
		// при Index == Hi обмена нет — pivot уже на месте, зеркало цело.
		displaced := r.mirror[e.Index]
		r.mirror[e.Index] = e.Value
		r.mirror[e.Hi] = displaced
		r.ranges = nil
		r.highlights = []SortHighlight{{Index: e.Index, Role: RolePivot}}
		r.operation = fmt.Sprintf("pivot на позиции %d", e.Index)
		if e.Index == e.Hi {
			r.frame(fmt.Sprintf("Partition [%d..%d] завершена: pivot %d уже на финальной позиции %d — слева все значения строго меньше его, справа — не меньше; обмена не потребовалось.",
				e.Lo, e.Hi, e.Value, e.Index))
		} else {
			r.frame(fmt.Sprintf("Partition [%d..%d] завершена: pivot %d занимает финальную позицию %d — слева все значения строго меньше его, справа — не меньше.",
				e.Lo, e.Hi, e.Value, e.Index))
		}
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
