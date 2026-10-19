package sort_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

func TestInsertionSortQuietMode(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want []int
	}{
		{"пустой вход", []int{}, []int{}},
		{"один элемент", []int{42}, []int{42}},
		{"два убывающих", []int{2, 1}, []int{1, 2}},
		{"неотсортированный", []int{3, 1, 2}, []int{1, 2, 3}},
		{"уже отсортированный", []int{1, 2, 3, 4}, []int{1, 2, 3, 4}},
		{"обратный", []int{5, 4, 3, 2, 1}, []int{1, 2, 3, 4, 5}},
		{"дубликаты", []int{2, 1, 2, 1, 3}, []int{1, 1, 2, 2, 3}},
		{"отрицательные", []int{-1, -3, 0, -2}, []int{-3, -2, -1, 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := append([]int{}, tt.in...)
			got := tt.in // сортировка на месте: проверяем переданный слайс

			sort.InsertionSort(got)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InsertionSort(%v) = %v, ожидалось %v", in, got, tt.want)
			}
		})
	}
}

// Сортировка на месте: указатель на первый элемент не меняется —
// InsertionSort не выделяет новый массив, а переупорядочивает переданный.
func TestInsertionSortInPlace(t *testing.T) {
	nums := []int{3, 1, 2}
	backing := &nums[0] // адрес первого элемента до сортировки

	sort.InsertionSort(nums)

	if &nums[0] != backing {
		t.Error("InsertionSort создаёт новый массив: сортировка обязана быть на месте")
	}
	if !reflect.DeepEqual(nums, []int{1, 2, 3}) {
		t.Errorf("результат сортировки = %v, ожидалось [1 2 3]", nums)
	}
}
