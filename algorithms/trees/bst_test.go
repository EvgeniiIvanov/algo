// Тесты шва 1 — публичный API BST-операций (см. docs/SPEC.md): инвариант
// после последовательностей операций, пустое дерево, дубликат, промахи,
// удаление корня и все три случая удаления. Внутренняя реализация
// (рекурсия, путь в событиях) не проверяется — только внешнее поведение.
// Тесты из внешнего пакета trees_test: проверяется только то, что
// доступно пользователю библиотеки.
package trees_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
)

// assertBSTInvariant — строгий BST-инвариант: все значения левого
// поддерева узла строго меньше его значения, правого — строго больше.
// Границы lo/hi (nil — границы нет) проверяют поддеревья целиком.
func assertBSTInvariant(t *testing.T, n *trees.Node, lo, hi *int) {
	t.Helper()
	if n == nil {
		return
	}
	if lo != nil && n.Value <= *lo {
		t.Fatalf("узел %d нарушает нижнюю границу %d", n.Value, *lo)
	}
	if hi != nil && n.Value >= *hi {
		t.Fatalf("узел %d нарушает верхнюю границу %d", n.Value, *hi)
	}
	assertBSTInvariant(t, n.Left, lo, &n.Value)
	assertBSTInvariant(t, n.Right, &n.Value, hi)
}

// mustBST — проверка инварианта после операций (упрощает чтение тестов).
func mustBST(t *testing.T, n *trees.Node) {
	t.Helper()
	assertBSTInvariant(t, n, nil, nil)
}

// buildBST — дерево из последовательности вставок в пустое дерево;
// после каждой вставки инвариант проверяется.
func buildBST(t *testing.T, keys ...int) *trees.Node {
	t.Helper()
	var root *trees.Node
	for _, key := range keys {
		inserted := false
		root, inserted = insertKey(t, root, key)
		if !inserted {
			t.Fatalf("вставка %d не выполнена", key)
		}
		mustBST(t, root)
	}
	return root
}

// insertKey — одна вставка через публичный API (обёртка для читаемости).
func insertKey(t *testing.T, root *trees.Node, key int) (*trees.Node, bool) {
	t.Helper()
	root, inserted := trees.Insert(root, key)
	mustBST(t, root)
	return root, inserted
}

// deleteKey — одно удаление через публичный API: признак сравнивается
// с ожидаемым, инвариант проверяется всегда.
func deleteKey(t *testing.T, root *trees.Node, key int, wantDeleted bool) *trees.Node {
	t.Helper()
	newRoot, deleted := trees.Delete(root, key)
	if deleted != wantDeleted {
		t.Fatalf("Delete(%d): deleted = %v, ожидается %v", key, deleted, wantDeleted)
	}
	mustBST(t, newRoot)
	return newRoot
}

func TestInsertEmptyTreeCreatesRoot(t *testing.T) {
	root, inserted := trees.Insert(nil, 5)
	if !inserted {
		t.Fatal("вставка в пустое дерево: inserted = false, ожидается true")
	}
	if root == nil || root.Value != 5 || root.Left != nil || root.Right != nil {
		t.Fatalf("вставка в пустое дерево: корень = %+v, ожидается узел 5 без потомков", root)
	}
	mustBST(t, root)
}

// Вставка в непустое дерево не меняет корень: тот же указатель.
func TestInsertKeepsRootPointer(t *testing.T) {
	root := buildBST(t, 5, 3)
	newRoot, inserted := trees.Insert(root, 7)
	if !inserted {
		t.Fatal("вставка 7: inserted = false, ожидается true")
	}
	if newRoot != root {
		t.Fatal("вставка в непустое дерево вернула другой корень")
	}
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{3, 5, 7}) {
		t.Fatalf("InOrder = %v, ожидается [3 5 7]", got)
	}
}

// Дубликат: дерево не меняется, вставка сообщает, что не выполнена.
func TestInsertDuplicate(t *testing.T) {
	root := buildBST(t, 5, 3, 7)
	left, right := root.Left, root.Right

	newRoot, inserted := trees.Insert(root, 5)
	if inserted {
		t.Fatal("повторная вставка 5: inserted = true, ожидается false")
	}
	if newRoot != root || newRoot.Left != left || newRoot.Right != right {
		t.Fatal("повторная вставка изменила дерево")
	}
	if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{3, 5, 7}) {
		t.Fatalf("InOrder после дубликата = %v, ожидается [3 5 7]", got)
	}
}

// Поиск: найденный узел — именно тот узел дерева, промах и пустое
// дерево — определённые результаты, не ошибки.
func TestSearch(t *testing.T) {
	root := buildBST(t, 5, 3, 7, 1, 9)

	node, found := trees.Search(root, 9)
	if !found || node != root.Right.Right {
		t.Fatalf("Search(9): found = %v, узел = %+v", found, node)
	}
	node, found = trees.Search(root, 5)
	if !found || node != root {
		t.Fatalf("Search(5): found = %v, узел = %+v", found, node)
	}
	if node, found = trees.Search(root, 4); found || node != nil {
		t.Fatalf("Search(4): found = %v, узел = %+v, ожидается промах", found, node)
	}
	if node, found = trees.Search(nil, 5); found || node != nil {
		t.Fatalf("Search в пустом дереве: found = %v, узел = %+v", found, node)
	}
}

// Удаление отсутствующего ключа: дерево не меняется, промах — не ошибка.
func TestDeleteMissing(t *testing.T) {
	root := buildBST(t, 5, 3, 7)
	newRoot := deleteKey(t, root, 4, false)
	if newRoot != root {
		t.Fatal("промах удаления изменил корень")
	}
	if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{3, 5, 7}) {
		t.Fatalf("InOrder после промаха = %v, ожидается [3 5 7]", got)
	}
	if got := deleteKey(t, nil, 5, false); got != nil {
		t.Fatalf("удаление из пустого дерева: корень = %+v, ожидается nil", got)
	}
}

// Случай 1 удаления — лист: на его месте пустота.
func TestDeleteLeaf(t *testing.T) {
	root := buildBST(t, 5, 3, 7)
	newRoot := deleteKey(t, root, 3, true)
	if newRoot.Left != nil {
		t.Fatalf("после удаления листа 3: левый потомок корня = %+v, ожидается nil", newRoot.Left)
	}
	if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{5, 7}) {
		t.Fatalf("InOrder = %v, ожидается [5 7]", got)
	}
}

// Случай 2 удаления — один потомок (левый или правый): на место узла
// встаёт его поддерево.
func TestDeleteOneChild(t *testing.T) {
	t.Run("только левый потомок", func(t *testing.T) {
		root := buildBST(t, 5, 3, 1)
		newRoot := deleteKey(t, root, 3, true)
		if newRoot.Left == nil || newRoot.Left.Value != 1 {
			t.Fatalf("после удаления 3: левый потомок корня = %+v, ожидается узел 1", newRoot.Left)
		}
		if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{1, 5}) {
			t.Fatalf("InOrder = %v, ожидается [1 5]", got)
		}
	})
	t.Run("только правый потомок", func(t *testing.T) {
		root := buildBST(t, 5, 7, 6)
		newRoot := deleteKey(t, root, 7, true)
		if newRoot.Right == nil || newRoot.Right.Value != 6 {
			t.Fatalf("после удаления 7: правый потомок корня = %+v, ожидается узел 6", newRoot.Right)
		}
		if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{5, 6}) {
			t.Fatalf("InOrder = %v, ожидается [5 6]", got)
		}
	})
}

// Случай 3 удаления — два потомка: значение узла становится значением
// inorder-successor (минимума правого поддерева), сам преемник удаляется.
func TestDeleteTwoChildren(t *testing.T) {
	t.Run("преемник — лист, у корня", func(t *testing.T) {
		root := buildBST(t, 10, 5, 20, 15, 30)
		newRoot := deleteKey(t, root, 10, true)
		if newRoot == nil || newRoot.Value != 15 {
			t.Fatalf("после удаления корня 10: корень = %+v, ожидается узел 15 (преемник)", newRoot)
		}
		if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{5, 15, 20, 30}) {
			t.Fatalf("InOrder = %v, ожидается [5 15 20 30]", got)
		}
	})
	t.Run("преемник не лист, не корень", func(t *testing.T) {
		root := buildBST(t, 10, 5, 20, 15, 30, 17, 25)
		newRoot := deleteKey(t, root, 20, true)
		// Преемник узла 20 — минимум правого поддерева: 15 (с правым
		// потомком 17). Значение 15 переходит узлу, сам преемник удалён.
		if node, found := trees.Search(newRoot, 20); found || node != nil {
			t.Fatalf("узел 20 должен быть удалён, найден %+v", node)
		}
		if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{5, 10, 15, 17, 25, 30}) {
			t.Fatalf("InOrder = %v, ожидается [5 10 15 17 25 30]", got)
		}
	})
	t.Run("преемник — сам правый потомок", func(t *testing.T) {
		root := buildBST(t, 10, 5, 20)
		newRoot := deleteKey(t, root, 10, true)
		if newRoot == nil || newRoot.Value != 20 {
			t.Fatalf("после удаления корня 10: корень = %+v, ожидается узел 20", newRoot)
		}
		if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{5, 20}) {
			t.Fatalf("InOrder = %v, ожидается [5 20]", got)
		}
	})
}

// Удаление корня: одноузловое дерево пустеет, корень с одним потомком
// замещается потомком, корень с двумя потомками — преемником.
func TestDeleteRoot(t *testing.T) {
	t.Run("единственный узел", func(t *testing.T) {
		root := buildBST(t, 5)
		if got := deleteKey(t, root, 5, true); got != nil {
			t.Fatalf("после удаления единственного узла: корень = %+v, ожидается nil", got)
		}
	})
	t.Run("корень с одним потомком", func(t *testing.T) {
		root := buildBST(t, 5, 7, 9)
		newRoot := deleteKey(t, root, 5, true)
		if newRoot == nil || newRoot.Value != 7 {
			t.Fatalf("после удаления корня 5: корень = %+v, ожидается узел 7", newRoot)
		}
	})
	t.Run("корень с двумя потомками", func(t *testing.T) {
		root := buildBST(t, 5, 3, 8, 7, 9)
		newRoot := deleteKey(t, root, 5, true)
		if newRoot == nil || newRoot.Value != 7 {
			t.Fatalf("после удаления корня 5: корень = %+v, ожидается узел 7 (преемник)", newRoot)
		}
		if got := trees.InOrder(newRoot); !reflect.DeepEqual(got, []int{3, 7, 8, 9}) {
			t.Fatalf("InOrder = %v, ожидается [3 7 8 9]", got)
		}
	})
}

// Последовательность операций на перекошенном дереве: после каждого
// шага инвариант сохраняется, удалённые ключи не находятся.
func TestSequenceOnSkewedTree(t *testing.T) {
	root := buildBST(t, 1, 2, 3, 4, 5) // перекошено вправо
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("InOrder = %v, ожидается [1 2 3 4 5]", got)
	}
	if node, found := trees.Search(root, 4); !found || node.Value != 4 {
		t.Fatalf("Search(4) в перекошенном дереве: found = %v", found)
	}

	root = deleteKey(t, root, 3, true) // середина: один потомок
	root = deleteKey(t, root, 3, false)
	root = deleteKey(t, root, 1, true) // корень с одним потомком
	root = deleteKey(t, root, 5, true) // дальний лист
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{2, 4}) {
		t.Fatalf("InOrder после последовательности = %v, ожидается [2 4]", got)
	}
	if node, found := trees.Search(root, 4); !found || node.Value != 4 {
		t.Fatalf("Search(4) после удалений: found = %v", found)
	}
	if node, found := trees.Search(root, 3); found || node != nil {
		t.Fatalf("удалённый ключ 3 найден: %+v", node)
	}
}

// Перекошенное влево дерево: вставка по убыванию и удаление листа-корня.
func TestSequenceOnLeftSkewedTree(t *testing.T) {
	root := buildBST(t, 5, 4, 3, 2, 1)
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("InOrder = %v, ожидается [1 2 3 4 5]", got)
	}
	root = deleteKey(t, root, 1, true) // дальний лист
	root = deleteKey(t, root, 5, true) // корень с одним потомком
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{2, 3, 4}) {
		t.Fatalf("InOrder = %v, ожидается [2 3 4]", got)
	}
}
