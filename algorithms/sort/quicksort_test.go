package sort_test

import (
	"reflect"
	"testing"

	topsort "sort"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

// QuickSort (тихий режим) на публичных входах: результат записывается
// в переданный слайс — проверяем его сам.
func TestQuickSortQuietMode(t *testing.T) {
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
		{"все равные", []int{7, 7, 7, 7}, []int{7, 7, 7, 7}},
		{"двойной разворот половин", []int{4, 3, 2, 1, 6, 5}, []int{1, 2, 3, 4, 5, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := append([]int{}, tt.in...)
			got := tt.in // сортировка на месте с точки зрения вызывающего кода

			sort.QuickSort(got)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sort.QuickSort(%v) = %v, ожидалось %v", in, got, tt.want)
			}
		})
	}
}

// Сортировка на месте: указатель на первый элемент не меняется —
// QuickSort переупорядочивает переданный слайс.
func TestQuickSortInPlace(t *testing.T) {
	nums := []int{3, 1, 2}
	backing := &nums[0] // адрес первого элемента до сортировки

	sort.QuickSort(nums)

	if &nums[0] != backing {
		t.Error("QuickSort создаёт новый массив: должен сортировать переданный на месте")
	}
	if !reflect.DeepEqual(nums, []int{1, 2, 3}) {
		t.Errorf("результат сортировки = %v, ожидалось [1 2 3]", nums)
	}
}

// Совпадение с эталоном на длинном псевдослучайном входе с дубликатами
// и отрицательными значениями; сюда же попадает худший случай за счёт
// уже упорядоченного входа с дубликатами.
func TestQuickSortAgainstOracle(t *testing.T) {
	size := 257
	got := make([]int, 0, size)
	for i := range size {
		got = append(got, (i*37)%53-i) // псевдослучайные значения с дубликатами и отрицательными
	}
	want := append([]int{}, got...)
	topsort.Ints(want)

	sort.QuickSort(got)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("QuickSort на случайном входе:\n получено %v\n ожидалось %v", got, want)
	}
}
