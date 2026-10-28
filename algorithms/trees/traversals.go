// Обходы бинарного дерева с типизированными событиями (ADR-0001).
//
// Quiet-версии (node.go) остаются обычными функциями библиотеки; здесь —
// их режим наблюдения: рекурсия и результат те же, перед посещением узла
// в эмиттер стреляется событие DFSVisit. Идентичность узла не выводится
// из значения: узел задаётся путём от корня (Path, символы L/R), поэтому
// узлы с повторяющимися значениями различимы.

package trees

import "github.com/EvgeniiIvanov/algo/algorithms"

// DFSEvent — маркер-интерфейс событий глубинных обходов бинарного дерева:
// вне пакета свои события реализовать нельзя.
type DFSEvent interface{ isDFSEvent() }

// DFSVisit — обход дошёл до узла: свершилось «посещение» узла в смысле
// выбранного порядка (in-order — после левого поддерева и т.д.).
// Path — путь узла от корня символами L (левый потомок) и R (правый);
// у корня путь пустой. Один узел — один Path: повторяющиеся значения
// не мешают отличать узлы.
type DFSVisit struct {
	Path  string // путь узла от корня ("": корень, "L", "LR", ...)
	Value int    // значение посещённого узла
}

func (DFSVisit) isDFSEvent() {}

// NodeID — стабильный идентификатор узла по пути от корня. Не зависит от
// значения узла; общий для событий и снимков kind="tree".
func NodeID(path string) string {
	if path == "" {
		return "root"
	}
	return path
}

// InOrderWithEmitter — InOrder в режиме наблюдения: перед посещением
// каждого узла в эмиттер уходит DFSVisit. Поведение и результат — те же,
// что у тихого InOrder.
func InOrderWithEmitter(n *Node, em algorithms.Emitter[DFSEvent]) []int {
	return inOrder(n, em, "")
}

// PreOrderWithEmitter — PreOrder в режиме наблюдения: перед посещением
// каждого узла в эмиттер уходит DFSVisit. Поведение и результат — те же,
// что у тихого PreOrder.
func PreOrderWithEmitter(n *Node, em algorithms.Emitter[DFSEvent]) []int {
	return preOrder(n, em, "")
}

// PostOrderWithEmitter — PostOrder в режиме наблюдения: перед посещением
// каждого узла в эмиттер уходит DFSVisit. Поведение и результат — те же,
// что у тихого PostOrder.
func PostOrderWithEmitter(n *Node, em algorithms.Emitter[DFSEvent]) []int {
	return postOrder(n, em, "")
}

// inOrder — рекурсивный InOrder с путём текущего узла от корня.
func inOrder(n *Node, em algorithms.Emitter[DFSEvent], path string) []int {
	if n == nil {
		return []int{}
	}
	result := inOrder(n.Left, em, path+"L")
	emitDFS(em, DFSVisit{Path: path, Value: n.Value})
	result = append(result, n.Value)
	result = append(result, inOrder(n.Right, em, path+"R")...)
	return result
}

// preOrder — рекурсивный PreOrder с путём текущего узла от корня.
func preOrder(n *Node, em algorithms.Emitter[DFSEvent], path string) []int {
	if n == nil {
		return []int{}
	}
	result := []int{n.Value}
	emitDFS(em, DFSVisit{Path: path, Value: n.Value})
	result = append(result, preOrder(n.Left, em, path+"L")...)
	result = append(result, preOrder(n.Right, em, path+"R")...)
	return result
}

// postOrder — рекурсивный PostOrder с путём текущего узла от корня.
func postOrder(n *Node, em algorithms.Emitter[DFSEvent], path string) []int {
	if n == nil {
		return []int{}
	}
	result := postOrder(n.Left, em, path+"L")
	result = append(result, postOrder(n.Right, em, path+"R")...)
	emitDFS(em, DFSVisit{Path: path, Value: n.Value})
	result = append(result, n.Value)
	return result
}

// emitDFS вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitDFS(em algorithms.Emitter[DFSEvent], event DFSEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
