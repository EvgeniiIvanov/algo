// Тесты событий BST-операций (ADR-0001): с Emitter последовательность
// событий восстанавливает сравнения с выбранной ветвью, исход операции
// и структурные изменения — включая поиск inorder-successor и удаление
// узла-преемника. Пути узлов — та же система координат, что у обходов
// (NodeID); внутренняя реализация по событиям не проверяется.
package trees_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms"
	"github.com/EvgeniiIvanov/algo/algorithms/trees"
)

// spyBSTEmitter — тестовый Emitter: собирает события BST-операций
// в слайс для проверки последовательности шагов.
type spyBSTEmitter []trees.BSTEvent

func (s *spyBSTEmitter) Emit(event trees.BSTEvent) { *s = append(*s, event) }

// Вставка 5, 3, 7 в пустое дерево: сравнение на каждом существующем узле
// с выбранной ветвью, вставка — на путь пустого места.
func TestInsertEvents(t *testing.T) {
	var em spyBSTEmitter
	var root *trees.Node
	for _, key := range []int{5, 3, 7} {
		root, _ = trees.InsertWithEmitter(root, key, &em)
	}

	want := []trees.BSTEvent{
		trees.BSTInserted{Path: "", Key: 5},
		trees.BSTCompare{Path: "", Key: 3, Value: 5, Branch: "L"},
		trees.BSTInserted{Path: "L", Key: 3},
		trees.BSTCompare{Path: "", Key: 7, Value: 5, Branch: "R"},
		trees.BSTInserted{Path: "R", Key: 7},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.BSTEvent(em), want)
	}
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{3, 5, 7}) {
		t.Fatalf("InOrder = %v, ожидается [3 5 7]", got)
	}
	// Заглушка для алгоритмов.Emitter: тип событий удовлетворяет общему
	// контракту.
	var _ algorithms.Emitter[trees.BSTEvent] = (*spyBSTEmitter)(nil)
}

// Повторная вставка: сравнение нашло существующий узел, вставки нет.
func TestInsertDuplicateEvents(t *testing.T) {
	var em spyBSTEmitter
	root, _ := trees.InsertWithEmitter(nil, 5, &em)
	em = nil // интересует только вторая операция
	root, _ = trees.InsertWithEmitter(root, 5, &em)

	want := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 5, Value: 5, Branch: ""},
		trees.BSTDuplicate{Path: "", Key: 5},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.BSTEvent(em), want)
	}
	if root == nil || root.Value != 5 || root.Left != nil || root.Right != nil {
		t.Fatalf("дерево после дубликата = %+v, ожидается один узел 5", root)
	}
}

// Поиск: сравнения с ветвями и исход — найден или промах (с местом,
// где ключ был бы). Пустое дерево — сразу промах.
func TestSearchEvents(t *testing.T) {
	root := buildBST(t, 5, 3, 7)

	var em spyBSTEmitter
	node, found := trees.SearchWithEmitter(root, 7, &em)
	if !found || node != root.Right {
		t.Fatalf("SearchWithEmitter(7): found = %v, узел = %+v", found, node)
	}
	wantFound := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 7, Value: 5, Branch: "R"},
		trees.BSTCompare{Path: "R", Key: 7, Value: 7, Branch: ""},
		trees.BSTFound{Path: "R", Key: 7},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), wantFound) {
		t.Fatalf("события поиска = %#v, ожидается %#v", []trees.BSTEvent(em), wantFound)
	}

	em = nil
	if _, found = trees.SearchWithEmitter(root, 4, &em); found {
		t.Fatal("SearchWithEmitter(4): found = true, ожидается промах")
	}
	wantMiss := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 4, Value: 5, Branch: "L"},
		trees.BSTCompare{Path: "L", Key: 4, Value: 3, Branch: "R"},
		trees.BSTNotFound{Path: "LR", Key: 4},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), wantMiss) {
		t.Fatalf("события промаха = %#v, ожидается %#v", []trees.BSTEvent(em), wantMiss)
	}

	em = nil
	if _, found = trees.SearchWithEmitter(nil, 5, &em); found {
		t.Fatal("SearchWithEmitter в пустом дереве: found = true")
	}
	wantEmpty := []trees.BSTEvent{trees.BSTNotFound{Path: "", Key: 5}}
	if !reflect.DeepEqual([]trees.BSTEvent(em), wantEmpty) {
		t.Fatalf("события пустого дерева = %#v, ожидается %#v", []trees.BSTEvent(em), wantEmpty)
	}
}

// Удаление листа: сравнения до узла, затем удаление.
func TestDeleteLeafEvents(t *testing.T) {
	root := buildBST(t, 5, 3, 7)

	var em spyBSTEmitter
	root, _ = trees.DeleteWithEmitter(root, 3, &em)

	want := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 3, Value: 5, Branch: "L"},
		trees.BSTCompare{Path: "L", Key: 3, Value: 3, Branch: ""},
		trees.BSTRemoved{Path: "L", Key: 3},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.BSTEvent(em), want)
	}
	if root.Left != nil {
		t.Fatalf("после удаления листа: левый потомок корня = %+v, ожидается nil", root.Left)
	}
}

// Промах удаления: сравнения до пустого места, исход — не найден,
// структурных изменений нет.
func TestDeleteMissingEvents(t *testing.T) {
	root := buildBST(t, 5, 3, 7)

	var em spyBSTEmitter
	root, _ = trees.DeleteWithEmitter(root, 4, &em)

	want := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 4, Value: 5, Branch: "L"},
		trees.BSTCompare{Path: "L", Key: 4, Value: 3, Branch: "R"},
		trees.BSTNotFound{Path: "LR", Key: 4},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.BSTEvent(em), want)
	}
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{3, 5, 7}) {
		t.Fatalf("InOrder = %v, ожидается [3 5 7]", got)
	}
}

// Удаление единственного узла: сравнение и удаление корня, дерево пустеет.
func TestDeleteRootSingleEvents(t *testing.T) {
	root := buildBST(t, 5)

	var em spyBSTEmitter
	root, _ = trees.DeleteWithEmitter(root, 5, &em)

	want := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 5, Value: 5, Branch: ""},
		trees.BSTRemoved{Path: "", Key: 5},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.BSTEvent(em), want)
	}
	if root != nil {
		t.Fatalf("корень = %+v, ожидается nil", root)
	}
}

// Удаление корня с двумя потомками: поиск преемника начинается с правого
// потомка, значение преемника переходит корню, узел-преемник удалён.
func TestDeleteRootTwoChildrenEvents(t *testing.T) {
	root := buildBST(t, 10, 5, 20)

	var em spyBSTEmitter
	root, _ = trees.DeleteWithEmitter(root, 10, &em)

	want := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 10, Value: 10, Branch: ""},
		trees.BSTMinStep{Path: "R", Value: 20},
		trees.BSTSuccessor{Path: "", SuccessorPath: "R", Value: 20},
		trees.BSTCompare{Path: "R", Key: 20, Value: 20, Branch: ""},
		trees.BSTRemoved{Path: "R", Key: 20},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.BSTEvent(em), want)
	}
	if root == nil || root.Value != 20 || root.Left.Value != 5 {
		t.Fatalf("корень после удаления = %+v, ожидается узел 20 с левым 5", root)
	}
}

// Удаление корня с двумя потомками и глубоким преемником: полная цепочка
// поиска минимума (левые переходы), замещение и удаление преемника.
func TestDeleteTwoChildrenDeepSuccessorEvents(t *testing.T) {
	root := buildBST(t, 10, 5, 20, 15, 30)

	var em spyBSTEmitter
	root, _ = trees.DeleteWithEmitter(root, 10, &em)

	want := []trees.BSTEvent{
		trees.BSTCompare{Path: "", Key: 10, Value: 10, Branch: ""},
		// Поиск преемника — минимум правого поддерева: 20 → 15.
		trees.BSTMinStep{Path: "R", Value: 20},
		trees.BSTMinStep{Path: "RL", Value: 15},
		trees.BSTSuccessor{Path: "", SuccessorPath: "RL", Value: 15},
		// Удаление узла-преемника: спуск по правому поддереву.
		trees.BSTCompare{Path: "R", Key: 15, Value: 20, Branch: "L"},
		trees.BSTCompare{Path: "RL", Key: 15, Value: 15, Branch: ""},
		trees.BSTRemoved{Path: "RL", Key: 15},
	}
	if !reflect.DeepEqual([]trees.BSTEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.BSTEvent(em), want)
	}
	if got := trees.InOrder(root); !reflect.DeepEqual(got, []int{5, 15, 20, 30}) {
		t.Fatalf("InOrder = %v, ожидается [5 15 20 30]", got)
	}
}

// Quiet- и событийный режимы — одна семантика (ADR-0001): одинаковый
// сценарий операций даёт одинаковые признаки исходов и одинаковое
// итоговое дерево.
func TestBSTEmitterModeMatchesQuiet(t *testing.T) {
	// Сценарий: вставки, дубликат, успешные и безрезультатные поиски
	// и удаления, корень с двумя потомками.
	script := []struct {
		op   string // "insert", "search", "delete"
		key  int
		want bool // ожидаемый признак исхода
	}{
		{"insert", 8, true},
		{"insert", 3, true},
		{"insert", 10, true},
		{"insert", 1, true},
		{"insert", 6, true},
		{"insert", 14, true},
		{"insert", 6, false}, // дубликат
		{"search", 6, true},
		{"search", 99, false},
		{"delete", 3, true}, // два потомка
		{"delete", 1, true}, // лист
		{"delete", 8, true}, // корень с двумя потомками
		{"delete", 99, false},
		{"search", 10, true},
		{"insert", 5, true},
	}

	run := func(withEmitter bool) (outcomes []bool, final []int) {
		var root *trees.Node
		var em spyBSTEmitter
		for _, step := range script {
			var got bool
			switch step.op {
			case "insert":
				if withEmitter {
					root, got = trees.InsertWithEmitter(root, step.key, &em)
				} else {
					root, got = trees.Insert(root, step.key)
				}
			case "search":
				if withEmitter {
					_, got = trees.SearchWithEmitter(root, step.key, &em)
				} else {
					_, got = trees.Search(root, step.key)
				}
			case "delete":
				if withEmitter {
					root, got = trees.DeleteWithEmitter(root, step.key, &em)
				} else {
					root, got = trees.Delete(root, step.key)
				}
			}
			if got != step.want {
				t.Fatalf("%s %d: исход = %v, ожидается %v", step.op, step.key, got, step.want)
			}
			outcomes = append(outcomes, got)
		}
		if withEmitter && len(em) == 0 {
			t.Fatal("событийный режим не записал ни одного события")
		}
		return outcomes, trees.InOrder(root)
	}

	quietOutcomes, quietFinal := run(false)
	emittedOutcomes, emittedFinal := run(true)
	if !reflect.DeepEqual(quietOutcomes, emittedOutcomes) {
		t.Fatalf("исходы режимов различаются: quiet %v, events %v", quietOutcomes, emittedOutcomes)
	}
	if !reflect.DeepEqual(quietFinal, emittedFinal) {
		t.Fatalf("итоговые деревья различаются: quiet %v, events %v", quietFinal, emittedFinal)
	}
	wantFinal := []int{5, 6, 10, 14} // 1 удалён шагом delete 1
	if !reflect.DeepEqual(emittedFinal, wantFinal) {
		t.Fatalf("итоговое дерево = %v, ожидается %v", emittedFinal, wantFinal)
	}
}
