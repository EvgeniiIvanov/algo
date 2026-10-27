// Пакет trees — общая модель бинарного дерева и глубинные обходы
// (in-order, pre-order, post-order): quiet mode и режим наблюдения (ADR-0001).
//
// Модель не навязывает свойства BST: значения узлов могут повторяться,
// отсутствующий потомок — nil. Учебный порядок выбран рекурсивным —
// три порядка отличаются только моментом «посещения» текущего узла
// и читаются напрямую из рекурсии.
package trees

// Node — узел бинарного дерева: целочисленное значение и потомки.
// Отсутствующий потомок — nil; дерево не обязано быть BST, значения
// могут повторяться (идентичность узла не выводится из значения).
type Node struct {
	Value int
	Left  *Node
	Right *Node
}

// NewNode — узел без потомков; связи задаются после создания.
func NewNode(value int) *Node {
	return &Node{Value: value}
}

// InOrder — симметричный обход: левое поддерево, узел, правое поддерево.
// Пустое дерево даёт пустой результат.
func InOrder(n *Node) []int {
	if n == nil {
		return []int{}
	}
	result := InOrder(n.Left)
	result = append(result, n.Value)
	result = append(result, InOrder(n.Right)...)
	return result
}

// PreOrder — прямой обход: узел, левое поддерево, правое поддерево.
func PreOrder(n *Node) []int {
	if n == nil {
		return []int{}
	}
	result := []int{n.Value}
	result = append(result, PreOrder(n.Left)...)
	result = append(result, PreOrder(n.Right)...)
	return result
}

// PostOrder — обратный обход: левое поддерево, правое поддерево, узел.
func PostOrder(n *Node) []int {
	if n == nil {
		return []int{}
	}
	result := PostOrder(n.Left)
	result = append(result, PostOrder(n.Right)...)
	result = append(result, n.Value)
	return result
}
