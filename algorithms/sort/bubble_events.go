package sort

import "github.com/EvgeniiIvanov/algo/algorithms"

// BubbleEvent — маркер-интерфейс событий bubble sort (sealed): вне пакета
// свои события реализовать нельзя. Типы названы по алгоритму — Bubble*,
// чтобы события будущих сортировок (Selection*, Insertion* и т.п.)
// спокойно жили в этом же пакете.
type BubbleEvent interface{ isBubbleEvent() }

// BubbleCompare — сравнение соседних элементов на позициях Left и Right
// (Right = Left+1). Значения пары — на момент сравнения; их достаточно,
// чтобы построить кадр без знания внутренностей алгоритма.
type BubbleCompare struct {
	Left       int // индекс левого элемента пары
	Right      int // индекс правого элемента пары
	LeftValue  int // значение левого элемента
	RightValue int // значение правого элемента
}

// BubbleSwap — обмен строго убывающей соседней пары Left и Right. Значения
// те же, что и в предыдущем сравнении этой пары — до обмена: шаг
// «после обмена» восстанавливается по индексам, порядок восстановления
// не искажает.
type BubbleSwap struct {
	Left       int // индекс левого элемента пары
	Right      int // индекс правого элемента пары
	LeftValue  int // значение левого элемента до обмена
	RightValue int // значение правого элемента до обмена
}

func (BubbleCompare) isBubbleEvent() {}
func (BubbleSwap) isBubbleEvent()    {}

// BubbleSortWithEmitter — BubbleSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// BubbleSort — сортировка по возрастанию на месте. На каждое сравнение
// соседей стреляет один BubbleCompare; состоявшийся обмен — ещё и BubbleSwap.
// Здесь живёт единственная копия цикла алгоритма — тихий BubbleSort
// делегирует сюда с nil эмиттером.
//
// Идея (учебный черновик пользователя): проходы слева направо сравнивают
// соседние элементы; каждая строго убывающая пара (nums[j] > nums[j+1])
// меняется местами, и максимум остатка «всплывает» в конец. Равные соседи
// не обмениваются (строгое «больше»), поэтому сортировка устойчива.
// После прохода без обменов массив уже упорядочен — работа завершается
// досрочно.
func BubbleSortWithEmitter(nums []int, em algorithms.Emitter[BubbleEvent]) {
	n := len(nums)
	for i := 0; i < n-1; i++ {
		swapped := false
		for j := 0; j < n-1-i; j++ {
			left, right := nums[j], nums[j+1]

			emitBubble(em, BubbleCompare{Left: j, Right: j + 1, LeftValue: left, RightValue: right})

			if left > right {
				nums[j], nums[j+1] = right, left
				swapped = true
				emitBubble(em, BubbleSwap{Left: j, Right: j + 1, LeftValue: left, RightValue: right})
			}
		}
		if !swapped {
			break
		}
	}
}

// emitBubble вызывает эмиттер, если он передан (внутренний nil-check —
// деталь реализации).
func emitBubble(em algorithms.Emitter[BubbleEvent], event BubbleEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
