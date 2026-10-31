// Обходы бинарного дерева с типизированными событиями (ADR-0001).
//
// Quiet-версии (node.go) остаются обычными функциями библиотеки; здесь —
// их режим наблюдения: рекурсия и результат те же, перед посещением узла
// в эмиттер стреляется событие DFSVisit (глубинные обходы) или
// LevelOrderVisit с состоянием FIFO-очереди (обход по уровням).
// Идентичность узла не выводится из значения: узел задаётся путём от
// корня (Path, символы L/R), поэтому узлы с повторяющимися значениями
// различимы.

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

// LevelOrderEvent — маркер-интерфейс событий обхода по уровням: вне пакета
// свои события реализовать нельзя.
type LevelOrderEvent interface{ isLevelOrderEvent() }

// LevelOrderVisit — узел снят с головы FIFO-очереди и посещён.
// Path — путь узла от корня (NodeID), как в DFSVisit; повторяющиеся
// значения остаются различимыми. Queue — состояние FIFO-очереди сразу
// после этого шага: посещённый узел уже снят с головы, его существующие
// потомки уже добавлены в конец; идентификаторы идут от головы к хвосту.
// Снимок — своя копия: очередь обхода дальше живёт своей жизнью.
type LevelOrderVisit struct {
	Path  string   // путь узла от корня ("": корень, "L", "LR", ...)
	Value int      // значение посещённого узла
	Queue []string // очередь после шага (путь к хвосту; пустая — очередь опустела)
}

func (LevelOrderVisit) isLevelOrderEvent() {}

// LevelOrderWithEmitter — LevelOrder в режиме наблюдения: на каждый
// посещённый узел в эмиттер уходит LevelOrderVisit с состоянием очереди.
// Поведение и результат — те же, что у тихого LevelOrder.
func LevelOrderWithEmitter(n *Node, em algorithms.Emitter[LevelOrderEvent]) []int {
	return levelOrder(n, em)
}

// levelOrder — общий цикл тихого и наблюдающего режимов обхода по
// уровням: FIFO-очередь с индексом головы (черновик автора — перемещение
// head вместо пересборки слайса — сохранён как есть). Учебная идея:
// узел посещается, когда доходит до головы очереди, его потомки
// становятся в конец — сначала левый, потом правый, поэтому уровни
// идут слева направо.
func levelOrder(n *Node, em algorithms.Emitter[LevelOrderEvent]) []int {
	if n == nil {
		return []int{}
	}
	// Очередь хранит пары узел/путь: идентичность узла не выводится
	// из значения (значения могут повторяться).
	queue := []queueNode{{node: n, path: ""}}
	head := 0
	result := []int{}

	for head < len(queue) {
		cur := queue[head]
		head++

		if cur.node.Left != nil {
			queue = append(queue, queueNode{node: cur.node.Left, path: cur.path + "L"})
		}
		if cur.node.Right != nil {
			queue = append(queue, queueNode{node: cur.node.Right, path: cur.path + "R"})
		}

		result = append(result, cur.node.Value)
		if em != nil {
			em.Emit(LevelOrderVisit{
				Path:  cur.path,
				Value: cur.node.Value,
				Queue: queuePaths(queue, head),
			})
		}
	}
	return result
}

// queueNode — элемент FIFO-очереди обхода по уровням: узел и его путь
// от корня (идентичность узла не выводится из значения).
type queueNode struct {
	node *Node
	path string
}

// queuePaths — оставшиеся в очереди пути от позиции head до хвоста:
// свежий слайс, чтобы событие не aliased на живую очередь обхода.
func queuePaths(queue []queueNode, head int) []string {
	paths := []string{}
	for i := head; i < len(queue); i++ {
		paths = append(paths, queue[i].path)
	}
	return paths
}
