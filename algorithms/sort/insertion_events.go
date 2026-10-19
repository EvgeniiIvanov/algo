package sort

import "github.com/EvgeniiIvanov/algo/algorithms"

// InsertionEvent — маркер-интерфейс событий insertion sort (sealed): вне
// пакета свои события реализовать нельзя. Типы названы по алгоритму —
// Insertion*, по образцу Bubble* и Selection*, чтобы события сортировок
// спокойно жили в этом же пакете.
type InsertionEvent interface{ isInsertionEvent() }

// InsertionCompare — сравнение ключа с элементом отсортированного префикса
// на позиции Pos (сканирование справа налево). Значения — на момент
// сравнения; исход сравнения несёт само событие (Shifts): рендерер строит
// кадр из события как есть — без повторного вывода результата сравнения
// из значений.
type InsertionCompare struct {
	Pos      int  // индекс элемента префикса, сравниваемого с ключом
	PosValue int  // значение элемента на позиции Pos на момент сравнения
	Key      int  // значение ключа, удерживаемого вне массива на время прохода
	Shifts   bool // исход сравнения: PosValue строго больше ключа — элемент сдвинется вправо
}

// InsertionShift — сдвиг значения, строго большего ключа, из позиции From
// в позицию To (= From+1): место для ключа освобождается. From — позиция,
// которая после сдвига становится «дыркой» — кандидатом на вставку ключа.
type InsertionShift struct {
	From  int // индекс, из которого значение уходит вправо
	To    int // индекс, в который значение записывается (From + 1)
	Value int // сдвигаемое значение (новое содержимое позиции To)
	Key   int // значение ключа, для которого освобождается место
}

// InsertionWrite — вставка ключа в освободившуюся позицию: финальный шаг
// прохода. Вставка без сдвигов (ключ уже на своём месте) массив не меняет
// и события не порождает — по образцу selection sort, чей проход
// с минимумом на первом месте обходится без события обмена.
type InsertionWrite struct {
	Index int // позиция, в которую вставляется ключ
	Value int // значение ключа
}

func (InsertionCompare) isInsertionEvent() {}
func (InsertionShift) isInsertionEvent()   {}
func (InsertionWrite) isInsertionEvent()   {}

// InsertionSortWithEmitter — InsertionSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// InsertionSort — сортировка по возрастанию на месте. На каждое сравнение
// элемента префикса с ключом стреляет один InsertionCompare; каждый
// состоявшийся сдвиг — ещё и InsertionShift; вставка ключа в освободившуюся
// позицию — InsertionWrite. Здесь живёт единственная копия цикла алгоритма —
// тихий InsertionSort делегирует сюда с nil эмиттером.
//
// Идея (учебный черновик пользователя): ключ удерживается вне массива,
// отсортированный префикс сканируется справа налево, и только элементы,
// строго большие ключа (nums[j] > key), сдвигаются вправо — равные ключу
// не сдвигаются, поэтому сортировка устойчива. Затем ключ записывается
// в освободившуюся позицию. Чтобы сравнение, на котором сканирование
// останавливается, тоже стреляло событием, проверка nums[j] > key вынесена
// из условия цикла в тело — сами сравнения и записи массива те же, что
// в черновике.
func InsertionSortWithEmitter(nums []int, em algorithms.Emitter[InsertionEvent]) {
	for i := 1; i < len(nums); i++ {
		key := nums[i]
		j := i - 1
		for j >= 0 {
			shift := nums[j] > key
			emitInsertion(em, InsertionCompare{Pos: j, PosValue: nums[j], Key: key, Shifts: shift})
			if !shift {
				break
			}
			v := nums[j] // значение строго больше ключа — уходит вправо
			nums[j+1] = v
			emitInsertion(em, InsertionShift{From: j, To: j + 1, Value: v, Key: key})
			j--
		}
		nums[j+1] = key
		if j+1 != i { // без сдвигов вставка не меняет массив — кадра нет
			emitInsertion(em, InsertionWrite{Index: j + 1, Value: key})
		}
	}
}

// emitInsertion вызывает эмиттер, если он передан (внутренний nil-check —
// деталь реализации).
func emitInsertion(em algorithms.Emitter[InsertionEvent], event InsertionEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
