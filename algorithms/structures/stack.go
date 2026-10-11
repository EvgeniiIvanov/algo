package structures

import (
	"errors"
	"fmt"
)

var (
	ErrStackOverflow         = errors.New("stack overflow")
	ErrStackUnderflow        = errors.New("stack underflow")
	ErrStackNegativeCapacity = errors.New("negative stack capacity")
)

type Stack struct {
	data []int // резервируем для стека
	sp   int   // указатель
}

// New создаёт стек фиксированной ёмкости size. Отрицательная ёмкость —
// ошибка, а не паника и не тихое исправление: объем задаёт вызывающий,
// и он же разбирается со своей ошибкой.
func New(size int) (*Stack, error) {
	if size < 0 {
		return nil, fmt.Errorf("ёмкость стека %d отрицательна: %w", size, ErrStackNegativeCapacity)
	}
	return &Stack{
		data: make([]int, size),
		sp:   0,
	}, nil
}

func (s *Stack) Push(v int) error {
	if s.sp == len(s.data) {
		return ErrStackOverflow
	}
	s.data[s.sp] = v
	s.sp++
	return nil
}

func (s *Stack) Pop() (int, error) {
	if s.sp == 0 {
		return 0, ErrStackUnderflow
	}
	s.sp--
	v := s.data[s.sp]
	return v, nil
}

func (s *Stack) Peek() (int, error) {
	if s.sp == 0 {
		return 0, ErrStackUnderflow
	}
	return s.data[s.sp-1], nil
}

func (s *Stack) Len() int {
	return s.sp
}

func (s *Stack) Cap() int {
	return len(s.data)
}

func (s *Stack) IsEmpty() bool {
	return s.sp == 0
}

func (s *Stack) IsFull() bool {
	return s.sp == len(s.data)
}

func (s *Stack) Clear() {
	s.sp = 0
}
