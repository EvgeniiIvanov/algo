// Бинарное дерево поиска (BST): вставка, поиск и удаление уникальных
// целочисленных ключей поверх общей модели Node.
//
// Инвариант строгий: у каждого узла все значения левого поддерева меньше
// его значения, правого — строго больше; значения узла и есть ключи.
// Повторные ключи не хранятся: повторная вставка существующего ключа
// не меняет дерево. Операции возвращают корень дерева после операции —
// вставка в пустое дерево и удаление корня меняют корень, — а также
// признак исхода: вставлен/удалён или нет. Дубликат и промах — не ошибки,
// а определённые результаты (ADR-0001: quiet mode и режим наблюдения
// имеют одну семантику — общее рекурсивное ядро принимает необязательный
// эмиттер, тихие версии передают nil).
//
// Путь узла от корня — строка из букв L (левый потомок) и R (правый),
// как в обходах: он же идентификатор узла в событиях (NodeID).
package trees

import "github.com/EvgeniiIvanov/algo/algorithms"

// Insert — вставка ключа в BST: возвращает корень дерева после вставки
// (у пустого дерева это новый узел) и признак, вставлен ли ключ.
// Повторная вставка существующего ключа не меняет дерево: inserted=false.
func Insert(root *Node, key int) (*Node, bool) {
	return insert(root, key, "", nil)
}

// Search — поиск узла с ключом: возвращает найденный узел (nil, если
// ключа нет) и признак находки. Пустое дерево и промах — не ошибки.
func Search(root *Node, key int) (*Node, bool) {
	return search(root, key, nil)
}

// Delete — удаление ключа: возвращает корень дерева после удаления
// (у одноузлового дерева это nil) и признак, удалён ли ключ. Удаление
// отсутствующего ключа не меняет дерево: deleted=false. Узел с двумя
// потомками замещается значением inorder-successor — минимума правого
// поддерева, после чего сам узел-преемник удаляется из этого поддерева.
func Delete(root *Node, key int) (*Node, bool) {
	return remove(root, key, "", nil)
}

// insert — общее рекурсивное ядро вставки: спуск по строгому порядку до
// пустого места, туда встаёт новый узел. path — путь текущего узла от
// корня операции; новый узел встаёт на путь пустого места.
func insert(n *Node, key int, path string, em algorithms.Emitter[BSTEvent]) (*Node, bool) {
	if n == nil {
		emitBST(em, BSTInserted{Path: path, Key: key})
		return NewNode(key), true
	}
	emitBST(em, BSTCompare{Path: path, Key: key, Value: n.Value, Branch: branchOf(key, n.Value)})
	switch {
	case key < n.Value:
		left, inserted := insert(n.Left, key, path+"L", em)
		n.Left = left
		return n, inserted
	case key > n.Value:
		right, inserted := insert(n.Right, key, path+"R", em)
		n.Right = right
		return n, inserted
	default:
		// Ключ уже есть: дерево не меняем, вставка не выполнена.
		emitBST(em, BSTDuplicate{Path: path, Key: key})
		return n, false
	}
}

// search — общее ядро поиска: спуск от корня, пока есть куда идти.
// Итеративный цикл черновика сохранён; путь наращивается символами L/R.
func search(root *Node, key int, em algorithms.Emitter[BSTEvent]) (*Node, bool) {
	n := root
	path := ""
	for n != nil {
		emitBST(em, BSTCompare{Path: path, Key: key, Value: n.Value, Branch: branchOf(key, n.Value)})
		switch {
		case key == n.Value:
			emitBST(em, BSTFound{Path: path, Key: key})
			return n, true
		case key < n.Value:
			n, path = n.Left, path+"L"
		default:
			n, path = n.Right, path+"R"
		}
	}
	// Пустое дерево или спуск в пустое место: ключа нет, промах — не ошибка.
	emitBST(em, BSTNotFound{Path: path, Key: key})
	return nil, false
}

// remove — общее рекурсивное ядро удаления: спуск по строгому порядку,
// найденный узел удаляется по случаям removeFound. path — путь текущего
// узла от корня операции: события удаления узла-преемника несут пути в
// той же системе координат, что и события внешней операции.
func remove(n *Node, key int, path string, em algorithms.Emitter[BSTEvent]) (*Node, bool) {
	if n == nil {
		emitBST(em, BSTNotFound{Path: path, Key: key})
		return nil, false
	}
	emitBST(em, BSTCompare{Path: path, Key: key, Value: n.Value, Branch: branchOf(key, n.Value)})
	switch {
	case key < n.Value:
		left, deleted := remove(n.Left, key, path+"L", em)
		n.Left = left
		return n, deleted
	case key > n.Value:
		right, deleted := remove(n.Right, key, path+"R", em)
		n.Right = right
		return n, deleted
	default:
		return removeFound(n, key, path, em)
	}
}

// removeFound — удаление найденного узла по трём случаям: лист или один
// потомок — на место узла встаёт единственный потомок (или пустота);
// два потомка — значение узла заменяется значением inorder-successor
// (минимума правого поддерева), после чего преемник удаляется из правого
// поддерева рекурсивным remove: у преемника нет левого потомка, поэтому
// там сработает случай «нет левого потомка».
func removeFound(n *Node, key int, path string, em algorithms.Emitter[BSTEvent]) (*Node, bool) {
	switch {
	case n.Left == nil:
		// Лист или только правый потомок: на место узла встаёт правое
		// поддерево (пустое место — для листа).
		emitBST(em, BSTRemoved{Path: path, Key: key})
		return n.Right, true
	case n.Right == nil:
		// Только левый потомок: на место узла встаёт левое поддерево.
		emitBST(em, BSTRemoved{Path: path, Key: key})
		return n.Left, true
	default:
		succ, succPath := minNode(n.Right, path+"R", em)
		emitBST(em, BSTSuccessor{Path: path, SuccessorPath: succPath, Value: succ.Value})
		n.Value = succ.Value
		// Узел-преемник удаляется из правого поддерева: путь считается
		// от корня операции, ключ — значение преемника. Признак заведомо
		// true: преемник — минимум поддерева, он не может не найтись.
		right, _ := remove(n.Right, succ.Value, path+"R", em)
		n.Right = right
		return n, true
	}
}

// minNode — минимум непустого поддерева: спуск влево до узла без левого
// потомка. На каждый посещённый узел уходит BSTMinStep; последний шаг
// цепочки — сам преемник. Возвращает узел-минимум и его путь.
func minNode(n *Node, path string, em algorithms.Emitter[BSTEvent]) (*Node, string) {
	for {
		emitBST(em, BSTMinStep{Path: path, Value: n.Value})
		if n.Left == nil {
			return n, path
		}
		n, path = n.Left, path+"L"
	}
}

// branchOf — ветвь сравнения ключей: "L" — ключ меньше узла (спуск
// влево), "R" — больше (вправо); равенство — "" (перехода нет: ключ
// в этом узле).
func branchOf(key, value int) string {
	switch {
	case key < value:
		return "L"
	case key > value:
		return "R"
	default:
		return ""
	}
}
