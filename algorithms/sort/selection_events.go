package sort

import "github.com/EvgeniiIvanov/algo/algorithms"

// SelectionEvent — маркер-интерфейс событий selection sort (sealed): вне
// пакета свои события реализовать нельзя. Типы названы по алгоритму —
// Selection*, по образцу Bubble*, чтобы события сортировок спокойно
// жили в этом же пакете.
type SelectionEvent interface{ isSelectionEvent() }

// SelectionCompare — сравнение проверяемого элемента Scanned с текущим
// минимумом прохода Min (Min < Scanned). Значения — на момент сравнения:
// их достаточно, чтобы построить кадр и показать текущий минимум, не
// зная внутренностей алгоритма.
type SelectionCompare struct {
	Pass      int // индекс первой неотсортированной позиции (начало прохода)
	Min       int // индекс текущего минимума прохода
	MinValue  int // значение текущего минимума на момент сравнения
	Scanned   int // индекс проверяемого элемента
	ScanValue int // значение проверяемого элемента
}

// SelectionSwap — обмен первого неотсортированного элемента Pass с найденным
// минимумом остатка Min (Pass < Min): финальный обмен прохода. Значения —
// до обмена: шаг «после обмена» восстанавливается по индексам. Проход,
// у которого минимум уже стоит на первом месте (Min == Pass), обмена
// не порождает.
type SelectionSwap struct {
	Pass      int // индекс первой неотсортированной позиции
	PassValue int // значение на позиции Pass до обмена
	Min       int // индекс найденного минимума остатка
	MinValue  int // значение минимума до обмена
}

func (SelectionCompare) isSelectionEvent() {}
func (SelectionSwap) isSelectionEvent()    {}

// SelectionSortWithEmitter — SelectionSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// SelectionSort — сортировка по возрастанию на месте. На каждое сравнение
// кандидата с текущим минимумом стреляет один SelectionCompare;
// состоявшийся финальный обмен прохода — ещё и SelectionSwap. Здесь живёт
// единственная копия цикла алгоритма — тихий SelectionSort делегирует
// сюда с nil эмиттером.
//
// Идея (учебный черновик пользователя): каждый проход ищет минимум
// неотсортированной части — начиная с позиции i, кандидаты j от i+1;
// строго меньший кандидат становится новым минимумом (nums[j] < nums[minIdx]),
// равные выбор не смещают — стандартный вариант неустойчив. Найденный
// минимум меняется местами с первым элементом части; если он уже там
// (minIdx == i), проход обходится без обмена.
func SelectionSortWithEmitter(nums []int, em algorithms.Emitter[SelectionEvent]) {
	n := len(nums)

	for i := 0; i < n-1; i++ {
		minIdx := i
		for j := i + 1; j < n; j++ {
			emitSelection(em, SelectionCompare{
				Pass:      i,
				Min:       minIdx,
				MinValue:  nums[minIdx],
				Scanned:   j,
				ScanValue: nums[j],
			})

			if nums[j] < nums[minIdx] {
				minIdx = j
			}
		}
		if minIdx != i {
			passValue, minValue := nums[i], nums[minIdx] // значения до обмена
			nums[i], nums[minIdx] = minValue, passValue
			emitSelection(em, SelectionSwap{Pass: i, PassValue: passValue, Min: minIdx, MinValue: minValue})
		}
	}
}

// emitSelection вызывает эмиттер, если он передан (внутренний nil-check —
// деталь реализации).
func emitSelection(em algorithms.Emitter[SelectionEvent], event SelectionEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
