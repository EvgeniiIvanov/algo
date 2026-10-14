// События односвязного списка (ADR-0001): каждое событие — атомарный шаг
// операции, достаточный для построения кадра без знания внутренностей
// списка (указателей head/tail, полей узлов).
package structures

import "github.com/EvgeniiIvanov/algo/algorithms"

// LinkedEvent — маркер-интерфейс событий односвязного списка (sealed):
// вне пакета свои события реализовать нельзя.
type LinkedEvent interface{ isLinkedEvent() }

// LinkedVisit — обход посетил узел на позиции Index (нумерация с головы,
// 0 — голова). Список пересчитаемой памяти не имеет: единственный способ
// дойти до узла — пройти по цепочке next узел за узлом, и каждый пройденный
// узел — шаг операции.
type LinkedVisit struct {
	Index int // логическая позиция посещённого узла
	Value int // значение в посещённом узле
}

// LinkedLink — переключена связь next узла на позиции Index: после этой
// операции его next ведёт на следующий справа узел (или срезает удалённый).
// Никакие узлы при этом не двигаются в памяти — меняется только указатель.
type LinkedLink struct {
	Index int // позиция узла, чей указатель next переключён
}

// LinkedInsert — узел со значением Value встал на позицию Index
// (0 — голова). Соседние узлы не двигаются — вокруг него переключаются связи.
type LinkedInsert struct {
	Index int // позиция нового узла
	Value int // вставленное значение
}

// LinkedRemove — узел со значением Value удалён с позиции Index
// (0 — голова). Оставшиеся узлы не сдвигаются: связь предыдущего
// переключается через удалённый, его позиции логические.
type LinkedRemove struct {
	Index int // логическая позиция удалённого узла
	Value int // удалённое значение
}

// LinkedFound — обход дошёл до узла с искомым значением и остановился.
type LinkedFound struct {
	Index int // позиция найденного узла
}

// LinkedNotFound — обход дошёл до конца списка и искомого значения
// не встретил: результат поиска — -1 (не ошибка).
type LinkedNotFound struct{}

func (LinkedVisit) isLinkedEvent()    {}
func (LinkedLink) isLinkedEvent()     {}
func (LinkedInsert) isLinkedEvent()   {}
func (LinkedRemove) isLinkedEvent()   {}
func (LinkedFound) isLinkedEvent()    {}
func (LinkedNotFound) isLinkedEvent() {}

// PrependWithEmitter — Prepend в режиме наблюдения (см. ADR-0001): узел
// встаёт в голову (LinkedInsert), при непустом списке его next подключается
// к старой голове (LinkedLink). Поведение и порядок в списке — те же,
// что у тихого Prepend.
func (l *LinkedList) PrependWithEmitter(v int, em algorithms.Emitter[LinkedEvent]) {
	l.Prepend(v)
	emitLinked(em, LinkedInsert{Index: 0, Value: v})
	if l.size > 1 {
		// Next нового головы ведёт на старую голову (теперь на позиции 1).
		emitLinked(em, LinkedLink{Index: 0})
	}
}

// AppendWithEmitter — Append в режиме наблюдения: при непустом списке
// next старого хвоста переключается на новый узел (LinkedLink), узел
// встаёт в хвост (LinkedInsert). Поведение — как у тихого Append.
func (l *LinkedList) AppendWithEmitter(v int, em algorithms.Emitter[LinkedEvent]) {
	if l.size == 0 {
		l.Append(v)
		emitLinked(em, LinkedInsert{Index: 0, Value: v})
		return
	}
	tail := l.size - 1
	l.Append(v)
	emitLinked(em, LinkedLink{Index: tail})
	emitLinked(em, LinkedInsert{Index: tail + 1, Value: v})
}

// PopFrontWithEmitter — PopFront в режиме наблюдения: голова уходит
// (LinkedRemove), обходимой цепочки до него не было — это и есть O(1).
// Пустой список — только ошибка, без события, как у тихого PopFront.
func (l *LinkedList) PopFrontWithEmitter(em algorithms.Emitter[LinkedEvent]) (int, error) {
	v, err := l.PopFront()
	if err != nil {
		return 0, err
	}
	emitLinked(em, LinkedRemove{Index: 0, Value: v})
	return v, nil
}

// PopBackWithEmitter — PopBack в режиме наблюдения: путь к предпоследнему
// узлу — обход от головы (LinkedVisit по узлам 0..Len-2), затем переключение
// связи (LinkedLink) и уход хвоста (LinkedRemove). Единственный узел
// уходит без обхода. Пустой список — только ошибка, без события.
func (l *LinkedList) PopBackWithEmitter(em algorithms.Emitter[LinkedEvent]) (int, error) {
	if l.size == 1 {
		return l.PopFrontWithEmitter(em)
	}
	if l.size > 1 {
		cur := l.head
		for k := 0; k < l.size-1; k++ {
			emitLinked(em, LinkedVisit{Index: k, Value: cur.val})
			if k < l.size-2 {
				cur = cur.next
			}
		}
		// cur — предпоследний узел: его next уходит на nil, хвост двигается.
		emitLinked(em, LinkedLink{Index: l.size - 2})
	}
	v, err := l.PopBack()
	if err != nil {
		return 0, err // недостижимо: размер проверен выше
	}
	emitLinked(em, LinkedRemove{Index: l.size, Value: v})
	return v, nil
}

// InsertAtWithEmitter — InsertAt в режиме наблюдения. Вставка в середину
// требует переходов по узлам 0..i-1 (LinkedVisit), затем переключается связь
// узла перед местом вставки (LinkedLink) и узел встаёт на позицию i
// (LinkedInsert). Вставка в голову/хвост — как у Prepend/AppendWithEmitter:
// обхода там нет. Поведение и ошибки — те же, что у тихого InsertAt;
// события ошибка индекса не даёт.
func (l *LinkedList) InsertAtWithEmitter(i, v int, em algorithms.Emitter[LinkedEvent]) error {
	if i < 0 || i > l.size {
		return ErrIndexOutOfRange
	}
	if i == 0 {
		l.PrependWithEmitter(v, em)
		return nil
	}
	if i == l.size {
		l.AppendWithEmitter(v, em)
		return nil
	}

	// Обход до узла i-1 (предыдущего для места вставки): посещаем
	// узлы 0..i-1 — каждый пройденный узел есть шаг.
	cur := l.head
	for k := 0; k < i; k++ {
		emitLinked(em, LinkedVisit{Index: k, Value: cur.val})
		if k < i-1 {
			cur = cur.next
		}
	}
	// cur — узел i-1: его next направляется на новый узел.
	emitLinked(em, LinkedLink{Index: i - 1})
	if err := l.InsertAt(i, v); err != nil {
		return err // недостижимо: границы проверены выше
	}
	emitLinked(em, LinkedInsert{Index: i, Value: v})
	return nil
}

// RemoveAtWithEmitter — RemoveAt в режиме наблюдения: возвращает удалённое
// значение (его берёт событие LinkedRemove). Удаление головы не требует
// обхода — один шаг LinkedRemove; удаление середины/хвоста — обход до узла
// i-1 (LinkedVisit), переключение связи (LinkedLink) и уход узла (LinkedRemove).
// Ошибка индекса события не даёт, как у тихого RemoveAt.
func (l *LinkedList) RemoveAtWithEmitter(i int, em algorithms.Emitter[LinkedEvent]) (int, error) {
	if i < 0 || i >= l.size {
		return 0, ErrIndexOutOfRange
	}
	if i == 0 {
		return l.PopFrontWithEmitter(em)
	}

	// Обход до узла i-1: посещаем узлы 0..i-1.
	cur := l.head
	for k := 0; k <= i-1; k++ {
		emitLinked(em, LinkedVisit{Index: k, Value: cur.val})
		if k < i-1 {
			cur = cur.next
		}
	}
	victim := cur.next.val // значение удалённого узла (cur.next — сам victim)
	emitLinked(em, LinkedLink{Index: i - 1})
	if err := l.RemoveAt(i); err != nil {
		return 0, err // недостижимо: границы проверены выше
	}
	emitLinked(em, LinkedRemove{Index: i, Value: victim})
	return victim, nil
}

// FindWithEmitter — Find в режиме наблюдения: обход от головы с визитом
// каждого проверенного узла (LinkedVisit) и финальным результатом поиска —
// LinkedFound на первом узле с искомым значением или LinkedNotFound при
// промахе. Отсутствие значения — не ошибка, а нормальный результат: возврат
// -1, тот же, что у тихого Find.
func (l *LinkedList) FindWithEmitter(value int, em algorithms.Emitter[LinkedEvent]) int {
	cur := l.head
	for k := 0; k < l.size; k++ {
		emitLinked(em, LinkedVisit{Index: k, Value: cur.val})
		if cur.val == value {
			emitLinked(em, LinkedFound{Index: k})
			return k
		}
		cur = cur.next
	}
	emitLinked(em, LinkedNotFound{})
	return -1
}

// emitLinked вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitLinked(em algorithms.Emitter[LinkedEvent], event LinkedEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
