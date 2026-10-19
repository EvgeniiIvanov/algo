package sort

import "github.com/EvgeniiIvanov/algo/algorithms"

// InsertionEvent — маркер-интерфейс событий insertion sort (sealed): вне
// пакета свои события реализовать нельзя. Типы названы по алгоритму —
// Insertion*, по образцу Bubble* и Selection*, чтобы события сортировок
// спокойно жили в этом же пакете.
type InsertionEvent interface{ isInsertionEvent() }

// InsertionOutcome — исход сравнения элемента префикса с ключом. Событие
// несёт его как есть: рендерер ветвит по нему, не повторяя сравнение
// из значений (по образцу SelectionCompare с NewMin).
type InsertionOutcome int

const (
	// InsertionGreater — элемент строго больше ключа: сдвинется вправо.
	InsertionGreater InsertionOutcome = iota
	// InsertionEqual — элемент равен ключу: не сдвигается, порядок равных
	// сохраняется (устойчивость).
	InsertionEqual
	// InsertionLess — элемент меньше ключа: сканирование остановлено.
	InsertionLess
)

// InsertionCompare — сравнение ключа с элементом отсортированного префикса
// на позиции Pos (сканирование справа налево). Значения — на момент
// сравнения; исход сравнения несёт само событие (Outcome), а «дырка» —
// позиция Gap, куда вернётся ключ при остановке сканирования: рендерер
// строит кадр из события как есть — без повторного вывода результата
// сравнения из значений.
type InsertionCompare struct {
	Pos      int              // индекс элемента префикса, сравниваемого с ключом
	PosValue int              // значение элемента на позиции Pos на момент сравнения
	Key      int              // значение ключа, удерживаемого вне массива на время прохода
	Gap      int              // индекс «дырки» на момент сравнения (Pos + 1)
	Outcome  InsertionOutcome // исход сравнения: Greater — сдвиг, Equal — пропуск равного, Less — остановка
}

// InsertionShift — сдвиг значения, строго большего ключа, из позиции From
// в позицию To (= From+1): место для ключа освобождается. После сдвига
// «дырка» переходит на From — это позиция Gap события: прежнее содержимое
// From — устаревшая копия, кадр помечает её пустой.
type InsertionShift struct {
	From  int // индекс, из которого значение уходит вправо
	To    int // индекс, в который значение записывается (From + 1)
	Value int // сдвигаемое значение (новое содержимое позиции To)
	Key   int // значение ключа, для которого освобождается место
	Gap   int // индекс «дырки» после сдвига (= From)
}

// InsertionWrite — вставка ключа в позицию Index: видимый финальный шаг
// каждого прохода. Moved различает вставку в «дырку» после сдвигов (true)
// и безсдвиговый проход, когда ключ уже на своём месте (false) — рендерер
// не восстанавливает это из истории событий.
type InsertionWrite struct {
	Index int  // позиция вставки ключа
	Value int  // значение ключа
	Moved bool // ключ сместился влево от своей исходной позиции (были сдвиги)
}

func (InsertionCompare) isInsertionEvent() {}
func (InsertionShift) isInsertionEvent()   {}
func (InsertionWrite) isInsertionEvent()   {}

// InsertionSortWithEmitter — InsertionSort в режиме наблюдения: каждый шаг
// сопровождается событием в em (ADR-0001). Результат тот же, что у тихого
// InsertionSort — сортировка по возрастанию на месте. На каждое сравнение
// элемента префикса с ключом стреляет один InsertionCompare (с исходом
// Outcome и «дыркой» Gap); каждый состоявшийся сдвиг — ещё и InsertionShift;
// каждый проход завершается InsertionWrite — в том числе проход без сдвигов,
// когда ключ уже на своём месте (Moved = false). Здесь живёт единственная
// копия цикла алгоритма — тихий InsertionSort делегирует сюда с nil
// эмиттером.
//
// Идея (учебный черновик пользователя): ключ удерживается вне массива,
// отсортированный префикс сканируется справа налево, и только элементы,
// строго большие ключа (nums[j] > key), сдвигаются вправо — равные ключу
// не сдвигаются, поэтому сортировка устойчива. Затем ключ записывается
// в освободившуюся позицию. Чтобы сравнение, на котором сканирование
// останавливается, тоже стреляло событием, проверка nums[j] > key вынесена
// из условия цикла в тело — сами сравнения и записи массива те же, что
// в черновике.
//
// Эмиссия охраняется проверкой em != nil на месте вызова: тихий режим
// не выполняет преобразование события в интерфейс и потому не аллоцирует
// вовсе (проверка внутри общего хелпера не помогла бы — боксинг случается
// на вызове, до nil-check).
func InsertionSortWithEmitter(nums []int, em algorithms.Emitter[InsertionEvent]) {
	for i := 1; i < len(nums); i++ {
		key := nums[i]
		j := i - 1
		for j >= 0 {
			outcome := InsertionLess
			switch {
			case nums[j] > key: // строгое «больше» — единственный повод сдвигать
				outcome = InsertionGreater
			case nums[j] == key:
				outcome = InsertionEqual
			}
			if em != nil {
				em.Emit(InsertionCompare{Pos: j, PosValue: nums[j], Key: key, Gap: j + 1, Outcome: outcome})
			}
			if outcome != InsertionGreater {
				break
			}
			v := nums[j] // значение строго больше ключа — уходит вправо
			nums[j+1] = v
			if em != nil {
				em.Emit(InsertionShift{From: j, To: j + 1, Value: v, Key: key, Gap: j})
			}
			j--
		}
		nums[j+1] = key
		if em != nil {
			em.Emit(InsertionWrite{Index: j + 1, Value: key, Moved: j+1 != i})
		}
	}
}
