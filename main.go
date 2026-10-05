package main

import (
	"fmt"
)

func main() {
	fmt.Println("Some algorithms examples")

	fmt.Println("E.g., integer array")
	numbers := IntArray{3, 5, 1, 4, 2}
	fmt.Println("Before sort:", numbers)
	numbers.Sort()
	fmt.Println("After sort:", numbers)

	num := 2
	fmt.Printf("And lets find index of number %d in this array...\n", num)
	index := numbers.Search(num)
	if index != -1 {
		fmt.Printf("Found at index %d\n", index)
	} else {
		fmt.Println("Not found")
	}
}

type Array[T any] interface {
	Sort()
	Search(x T) int
}

type IntArray []int

// Bubble Sort for integet array
func (ia *IntArray) Sort() {
	n := len(*ia)

	for i := 0; i < n-1; i++ {
		swapped := false
		for j := 0; j < n-i-1; j++ {
			if (*ia)[j] > (*ia)[j+1] {
				(*ia)[j], (*ia)[j+1] = (*ia)[j+1], (*ia)[j]
				swapped = true
			}
		}

		if !swapped {
			break
		}
	}
}

// Binary Search algorithms for integer array
func (ia IntArray) Search(t int) int {
	low := 0
	high := len(ia) - 1

	for low <= high {
		mid := low + (high-low)/2

		if ia[mid] == t {
			return mid
		} else if ia[mid] < t {
			low = mid + 1
		} else {
			high = mid - 1
		}
	}
	return -1
}

type StringArray []string

// Implement later
