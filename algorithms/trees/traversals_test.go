// Тесты шва 1 — публичный API обходов (см. docs/SPEC.md): порядок
// возвращённых значений на заданных формах дерева. Внутренний способ
// реализации (рекурсия) не проверяется — только внешнее поведение.
// Тесты из внешнего пакета trees_test: проверяется только то, что
// доступно пользователю библиотеки.
package trees_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
)

// buildSymmetric — асимметричное учебное дерево 7 узлов:
//
//	     1
//	   /   \
//	  2     3
//	 / \     \
//	4   5     6
//	   /
//	  7
func buildSymmetric() *trees.Node {
	n1 := trees.NewNode(1)
	n2 := trees.NewNode(2)
	n3 := trees.NewNode(3)
	n4 := trees.NewNode(4)
	n5 := trees.NewNode(5)
	n6 := trees.NewNode(6)
	n7 := trees.NewNode(7)
	n1.Left, n1.Right = n2, n3
	n2.Left, n2.Right = n4, n5
	n3.Right = n6
	n5.Left = n7
	return n1
}

func TestInOrder(t *testing.T) {
	tests := []struct {
		name string
		root *trees.Node
		want []int
	}{
		{"пустое дерево", nil, []int{}},
		{"одноузловое", trees.NewNode(5), []int{5}},
		// Лево-корень-право: 4,2,5,7 — но у 5 есть левый 7.
		// Симметричное дерево: 4,2,7,5,1,3,6.
		{"асимметричное", buildSymmetric(), []int{4, 2, 7, 5, 1, 3, 6}},
		{"перекошенное вправо", skewedRight(), []int{1, 2, 3}},
		{"перекошенное влево", skewedLeft(), []int{3, 2, 1}},
		{"повторяющиеся значения", buildDuplicates(), []int{7, 7, 2, 2, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trees.InOrder(tt.root)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("trees.InOrder = %v, ожидается %v", got, tt.want)
			}
		})
	}
}

func TestPreOrder(t *testing.T) {
	tests := []struct {
		name string
		root *trees.Node
		want []int
	}{
		{"пустое дерево", nil, []int{}},
		{"одноузловое", trees.NewNode(5), []int{5}},
		{"асимметричное", buildSymmetric(), []int{1, 2, 4, 5, 7, 3, 6}},
		{"перекошенное вправо", skewedRight(), []int{1, 2, 3}},
		{"перекошенное влево", skewedLeft(), []int{1, 2, 3}},
		{"повторяющиеся значения", buildDuplicates(), []int{2, 7, 7, 2, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trees.PreOrder(tt.root)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("trees.PreOrder = %v, ожидается %v", got, tt.want)
			}
		})
	}
}

func TestPostOrder(t *testing.T) {
	tests := []struct {
		name string
		root *trees.Node
		want []int
	}{
		{"пустое дерево", nil, []int{}},
		{"одноузловое", trees.NewNode(5), []int{5}},
		{"асимметричное", buildSymmetric(), []int{4, 7, 5, 2, 6, 3, 1}},
		{"перекошенное вправо", skewedRight(), []int{3, 2, 1}},
		{"перекошенное влево", skewedLeft(), []int{3, 2, 1}},
		{"повторяющиеся значения", buildDuplicates(), []int{7, 7, 2, 2, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trees.PostOrder(tt.root)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("trees.PostOrder = %v, ожидается %v", got, tt.want)
			}
		})
	}
}

// buildDuplicates — дерево с повторяющимися значениями:
//
//	     2
//	   /   \
//	  7     2
//	 /       \
//	7         2
//
// Порядок относится к узлам, а не к значениям: одинаковые значения
// идут в порядке, заданном формой дерева.
func buildDuplicates() *trees.Node {
	root := trees.NewNode(2)
	l := trees.NewNode(7)
	r := trees.NewNode(2)
	ll := trees.NewNode(7)
	rr := trees.NewNode(2)
	root.Left, root.Right = l, r
	l.Left = ll
	r.Right = rr
	return root
}

// skewedRight — перекошенное дерево из трёх узлов вправо.
func skewedRight() *trees.Node {
	n1 := trees.NewNode(1)
	n2 := trees.NewNode(2)
	n3 := trees.NewNode(3)
	n1.Right = n2
	n2.Right = n3
	return n1
}

// skewedLeft — перекошенное дерево из трёх узлов влево.
func skewedLeft() *trees.Node {
	n1 := trees.NewNode(1)
	n2 := trees.NewNode(2)
	n3 := trees.NewNode(3)
	n1.Left = n2
	n2.Left = n3
	return n1
}
