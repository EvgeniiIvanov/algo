// События списка (ADR-0001): каждое событие — атомарный шаг операции,
// достаточный для построения кадра без знания внутренностей списка.
package structures

import "github.com/EvgeniiIvanov/algo/algorithms"

// ListEvent — маркер-интерфейс событий списка (sealed):
// вне пакета свои события реализовать нельзя.
type ListEvent interface{ isListEvent() }

// ListInsert — вставка значения на позицию: существующие значения правее
// сдвинулись на одну ячейку вправо. Shifted — сколько значений сдвинулось
// (0 — вставка в хвост, сдвига нет).
type ListInsert struct {
	Index   int // позиция, куда записано значение
	Value   int // вставленное значение
	Shifted int // сколько существующих значений сдвинулось вправо
}

// ListRemove — удаление значения по индексу: значения правее сдвинулись
// на одну ячейку влево. Shifted — сколько значений сдвинулось
// (0 — удалён последний элемент, сдвига нет).
type ListRemove struct {
	Index   int // позиция, с которой удалено значение
	Value   int // удалённое значение
	Shifted int // сколько значений сдвинулось влево
}

func (ListInsert) isListEvent() {}
func (ListRemove) isListEvent() {}

// InsertWithEmitter — Insert в режиме наблюдения (см. ADR-0001): успешная
// вставка сопровождается событием ListInsert с числом сдвинувшихся значений.
// Поведение и ошибки — те же, что у тихого Insert; ошибка индекса события
// не даёт (как тихая Insert).
func (l *List) InsertWithEmitter(i, v int, em algorithms.Emitter[ListEvent]) error {
	if err := l.Insert(i, v); err != nil {
		return err
	}
	// После вставки длина выросла на 1: правее записанной ячейки
	// осталось l.Len()-1-i значений, каждое уехало на ячейку вправо.
	emitList(em, ListInsert{Index: i, Value: v, Shifted: l.Len() - 1 - i})
	return nil
}

// RemoveAtWithEmitter — RemoveAt в режиме наблюдения: возвращает удалённое
// значение (его берёт событие ListRemove, поэтому механизм шире тихого
// RemoveAt, возвращающего только ошибку). Ошибка индекса события не даёт.
func (l *List) RemoveAtWithEmitter(i int, em algorithms.Emitter[ListEvent]) (int, error) {
	v, err := l.Get(i)
	if err != nil {
		return 0, err
	}
	if err := l.RemoveAt(i); err != nil {
		return 0, err // недостижимо: Get и RemoveAt проверяют одни и те же границы
	}
	// После удаления длина стала меньше на 1: значения правее позиции i
	// уехали на ячейку влево, их l.Len()-i штук.
	emitList(em, ListRemove{Index: i, Value: v, Shifted: l.Len() - i})
	return v, nil
}

// emitList вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitList(em algorithms.Emitter[ListEvent], event ListEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
