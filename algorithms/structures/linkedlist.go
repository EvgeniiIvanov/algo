package structures

// LinkedList — односвязный список на указателях: память не непрерывна,
// каждый узел хранит значение и указатель next. Поддерживаем tail и size,
// чтобы вставка в хвост была O(1) без обхода. Доступ/вставка/удаление
// по позиции требуют обхода от головы узел за узлом — O(n).
type LinkedList struct {
	head *node
	tail *node
	size int
}

// node — узел списка: значение и связь со следующим. Позиции узлов
// логические (нумерация с головы, 0 — голова); адреса памяти не показываются
// и визуально не рисуются (out of scope модуля).
type node struct {
	val  int
	next *node
}

// NewLinkedList создаёт пустой список.
func NewLinkedList() *LinkedList {
	return &LinkedList{}
}

// Prepend добавляет значение в голову: новый узел становится головой,
// его next ведёт на старую голову. O(1) — обхода нет, список не двигается.
func (l *LinkedList) Prepend(v int) {
	n := &node{val: v, next: l.head}
	l.head = n
	if l.tail == nil {
		l.tail = n // список был пуст: единственный узел — и голова, и хвост
	}
	l.size++
}

// Append добавляет значение в хвост: next старого хвоста ведёт на новый
// узел, который сам становится хвостом. O(1) за счёт поддерживаемого tail
// — обход от головы не нужен.
func (l *LinkedList) Append(v int) {
	n := &node{val: v}
	if l.head == nil {
		l.head = n
		l.tail = n
	} else {
		l.tail.next = n
		l.tail = n
	}
	l.size++
}

// PopFront снимает значение с головы. Пустой список — ErrListEmpty.
// Голова сдвигается на следующий узел: O(1), хвост правим только когда
// список опустел.
func (l *LinkedList) PopFront() (int, error) {
	if l.head == nil {
		return 0, ErrListEmpty
	}
	v := l.head.val
	l.head = l.head.next
	if l.head == nil {
		l.tail = nil // сняли единственный узел — хвост уходит вместе с головой
	}
	l.size--
	return v, nil
}

// PopBack снимает значение с хвоста. У односвязного списка нет prev:
// путь к предпоследнему узлу — обход от головы, поэтому O(n).
// Пустой список — ErrListEmpty.
func (l *LinkedList) PopBack() (int, error) {
	if l.head == nil {
		return 0, ErrListEmpty
	}
	if l.head == l.tail {
		v := l.head.val
		l.head = nil
		l.tail = nil
		l.size--
		return v, nil
	}
	cur := l.head
	for cur.next != l.tail {
		cur = cur.next
	}
	v := l.tail.val
	cur.next = nil
	l.tail = cur
	l.size--
	return v, nil
}

// InsertAt вставляет значение на позицию i. Допустимый диапазон —
// 0..Len включительно: 0 — Prepend в голову, Len — Append в хвост;
// вставка в середину — после перехода к узлу i-1 (O(n) обхода, сама
// переоценка связи — O(1)). Индекс вне диапазона — ErrIndexOutOfRange,
// а не паника.
func (l *LinkedList) InsertAt(i, v int) error {
	if i < 0 || i > l.size {
		return ErrIndexOutOfRange
	}
	if i == 0 {
		l.Prepend(v)
		return nil
	}
	if i == l.size {
		l.Append(v)
		return nil
	}
	prev := l.head
	for k := 0; k < i-1; k++ {
		prev = prev.next
	}
	n := &node{val: v, next: prev.next} // новый узел держит следующий
	prev.next = n                       // предыдущий переключается на новый
	l.size++
	return nil
}

// RemoveAt удаляет узел на позиции i; допустимый диапазон — 0..Len-1,
// индекс вне диапазона — ErrIndexOutOfRange. Голова удаляется за
// O(1) (двигается только head), середина/хвост — после перехода к узлу i-1
// (O(n) обхода, сама переоценка связи — O(1)).
func (l *LinkedList) RemoveAt(i int) error {
	if i < 0 || i >= l.size {
		return ErrIndexOutOfRange
	}
	if i == 0 {
		_, err := l.PopFront()
		return err
	}
	prev := l.head
	for k := 0; k < i-1; k++ {
		prev = prev.next
	}
	victim := prev.next
	prev.next = victim.next // связь перепрыгивает через удалённый узел
	if victim == l.tail {
		l.tail = prev
	}
	l.size--
	return nil
}

// Find ищет значение (не индекс!) в списке обходом от головы и возвращает
// позицию первого узла с этим значением; значения в списке нет — -1.
// Отсутствие значения — не ошибка, а нормальный результат поиска (как
// в бинарном поиске трейсера): учебная цель — самому увидеть обход узел
// за узлом и его цену O(n).
func (l *LinkedList) Find(value int) int {
	cur := l.head
	for k := 0; k < l.size; k++ {
		if cur.val == value {
			return k
		}
		cur = cur.next
	}
	return -1
}

// Len возвращает число узлов.
func (l *LinkedList) Len() int {
	return l.size
}

// IsEmpty сообщает, пуст ли список.
func (l *LinkedList) IsEmpty() bool {
	return l.size == 0
}
