package structures

import (
	"errors"
	"fmt"
)

var (
	ErrQueueFull             = errors.New("queue overflow")
	ErrQueueEmpty            = errors.New("queue underflow")
	ErrQueueNegativeCapacity = errors.New("negative queue capacity")
)

type Queue struct {
	data []int
	max  int
}

// NewQueue создаёт очередь фиксированной ёмкости max. Отрицательная ёмкость —
// ошибка, а не паника и не тихое исправление.
func NewQueue(max int) (*Queue, error) {
	if max < 0 {
		return nil, fmt.Errorf("длина очереди %d отрицательна: %w", max, ErrQueueNegativeCapacity)
	}
	return &Queue{
		data: make([]int, 0, max),
		max:  max,
	}, nil
}

func (q *Queue) Enqueue(v int) error {
	if len(q.data) == q.max {
		return ErrQueueFull
	}
	q.data = append(q.data, v)
	return nil
}

func (q *Queue) Dequeue() (int, error) {
	if len(q.data) == 0 {
		return 0, ErrQueueEmpty
	}
	v := q.data[0]
	copy(q.data, q.data[1:])        // делаем копию со сдвигом элементов влево
	q.data = q.data[:len(q.data)-1] // отрезаем лишнее справа
	return v, nil
}

func (q *Queue) Peek() (int, error) {
	if len(q.data) == 0 {
		return 0, ErrQueueEmpty
	}
	return q.data[0], nil
}

func (q *Queue) Len() int {
	return len(q.data)
}

func (q *Queue) IsEmpty() bool {
	return len(q.data) == 0
}

func (q *Queue) IsFull() bool {
	return len(q.data) == q.max
}

func (q *Queue) Clear() {
	q.data = q.data[:0]
}
