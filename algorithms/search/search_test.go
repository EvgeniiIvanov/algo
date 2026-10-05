package search_test

import (
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/search"
)

// Бинарный поиск находит индекс элемента в отсортированном массиве.
func TestSearchFindsElement(t *testing.T) {
	data := []int{1, 3, 5, 7, 9, 11}

	if got := search.Search(data, 7); got != 3 {
		t.Fatalf("Search([1,3,5,7,9,11], 7) = %d, ожидался 3", got)
	}
}

// Граничные случаи: отсутствие, пустой вход, один элемент, края массива.
func TestSearchEdgeCases(t *testing.T) {
	data := []int{1, 3, 5, 7, 9, 11}

	cases := []struct {
		name   string
		data   []int
		target int
		want   int
	}{
		{"отсутствующий элемент", data, 4, -1},
		{"отсутствующий за границами слева", data, 0, -1},
		{"отсутствующий за границами справа", data, 12, -1},
		{"пустой вход", []int{}, 5, -1},
		{"nil вместо слайса", nil, 5, -1},
		{"единственный элемент — попадание", []int{5}, 5, 0},
		{"единственный элемент — промах", []int{5}, 3, -1},
		{"первый элемент", data, 1, 0},
		{"последний элемент", data, 11, 5},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := search.Search(tc.data, tc.target); got != tc.want {
				t.Fatalf("Search(%v, %d) = %d, ожидался %d", tc.data, tc.target, got, tc.want)
			}
		})
	}
}
