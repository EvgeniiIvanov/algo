package structures

import (
	"errors"
)

var ErrIndexOutOfRange = errors.New("index out of range")

// List — список на слайсе: растущая структура без фиксированной ёмкости
// (темы роста и амортизации модуля живут здесь). Хранит значения в порядке
// следования; вставка/удаление в середине двигают хвост — O(n) на слайсе.
type List struct {
	data []int
}

// NewList создаёт пустой список.
func NewList() *List {
	return &List{
		data: make([]int, 0),
	}
}

// Insert вставляет значение v на позицию i, сдвигая значения правее на одну
// ячейку. Допустимый диапазон индекса — 0..Len включительно: 0 — вставка в
// голову, Len — вставка в хвост. Индекс вне диапазона — ErrIndexOutOfRange,
// а не паника.
func (l *List) Insert(i, v int) error {
	if i < 0 || i > len(l.data) {
		return ErrIndexOutOfRange
	}
	l.data = append(l.data, 0)
	copy(l.data[i+1:], l.data[i:])
	l.data[i] = v
	return nil
}

// RemoveAt удаляет значение на позиции i; значения правее сдвигаются влево.
// Допустимый диапазон — 0..Len-1; индекс вне диапазона — ErrIndexOutOfRange.
func (l *List) RemoveAt(i int) error {
	if i < 0 || i >= len(l.data) {
		return ErrIndexOutOfRange
	}
	copy(l.data[i:], l.data[i+1:])
	l.data = l.data[:len(l.data)-1]
	return nil
}

// Get возвращает значение на позиции i (0..Len-1) или ErrIndexOutOfRange.
func (l *List) Get(i int) (int, error) {
	if i < 0 || i >= len(l.data) {
		return 0, ErrIndexOutOfRange
	}
	return l.data[i], nil
}

func (l *List) Len() int {
	return len(l.data)
}
