package sort_test

import (
	"reflect"
	"testing"

	topsort "sort"

	"github.com/EvgeniiIvanov/algo/algorithms/sort"
)

// MergeSort (тихий режим) на публичных входах: результат записывается
// в переданный слайс — проверяем его сам.
func TestMergeSortQuietMode(t *testing.T) {
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
		{"двойная длина не делится по хвосту", []int{4, 3, 2, 1, 6, 5}, []int{1, 2, 3, 4, 5, 6}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := append([]int{}, tt.in...)
			got := tt.in // сортировка на месте с точки зрения вызывающего кода

			sort.MergeSort(got)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sort.MergeSort(%v) = %v, ожидалось %v", in, got, tt.want)
			}
		})
	}
}

// Сортировка на месте: указатель на первый элемент не меняется —
// MergeSort переупорядочивает переданный слайс.
func TestMergeSortInPlace(t *testing.T) {
	nums := []int{3, 1, 2}
	backing := &nums[0] // адрес первого элемента до сортировки

	sort.MergeSort(nums)

	if &nums[0] != backing {
		t.Error("MergeSort создаёт новый массив: должен сортировать переданный на месте")
	}
	if !reflect.DeepEqual(nums, []int{1, 2, 3}) {
		t.Errorf("результат сортировки = %v, ожидалось [1 2 3]", nums)
	}
}

// Устойчивость на большом входе значения одинаковые не различают — она
// зафиксирована по событиям (FromLeft) в merge_events_test.go; здесь —
// совпадение с эталоном на длинном псевдослучайном входе с дубликатами
// и отрицательными значениями.
func TestMergeSortAgainstOracle(t *testing.T) {
	size := 257
	got := make([]int, 0, size)
	for i := range size {
		got = append(got, (i*37)%53-i) // псевдослучайные значения с дубликатами и отрицательными
	}
	want := append([]int{}, got...)
	topsort.Ints(want)

	sort.MergeSort(got)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("MergeSort на случайном входе:\n получено %v\n ожидалось %v", got, want)
	}
}
