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
// уровням. Очередь — кольцевой буфер: снятые с головы слоты освобождаются
// сразу, обработанный префикс не накапливается, живых элементов всегда
// не больше пары соседних уровней. Учебная идея черновика сохранена:
// узел посещается, когда доходит до головы FIFO-очереди, его потомки
// становятся в конец — сначала левый, потом правый, поэтому уровни
// идут слева направо.
func levelOrder(n *Node, em algorithms.Emitter[LevelOrderEvent]) []int {
	if n == nil {
		return []int{}
	}
	queue := newNodeQueue()
	queue.push(queueNode{node: n, path: ""})
	result := []int{}

	for queue.size > 0 {
		cur := queue.pop()
		if cur.node.Left != nil {
			queue.push(queueNode{node: cur.node.Left, path: cur.path + "L"})
		}
		if cur.node.Right != nil {
			queue.push(queueNode{node: cur.node.Right, path: cur.path + "R"})
		}
		result = append(result, cur.node.Value)
		if em != nil {
			em.Emit(LevelOrderVisit{
				Path:  cur.path,
				Value: cur.node.Value,
				Queue: queue.paths(),
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

// nodeQueue — FIFO-очередь обхода по уровням на кольцевом буфере.
// Освобождённые слоты очищаются сразу при снятии с головы, чтобы
// обработанные узлы не удерживались; буфер удваивается только при
// переполнении живыми элементами, поэтому ёмкость соизмерима с шириной
// дерева. size — число живых элементов; buf ненулевой всегда.
type nodeQueue struct {
	buf  []queueNode // кольцевой буфер (сделан с запасом, нулевые слоты — свободны)
	head int         // индекс головы в buf
	size int         // живые элементы, подряд с позиции head (с заворачиванием)
}

// newNodeQueue — пустая очередь с минимальным запасом ёмкости.
func newNodeQueue() *nodeQueue {
	return &nodeQueue{buf: make([]queueNode, 4)}
}

// push — поставить элемент в конец очереди; при полном живом буфере
// ёмкость удваивается (старые элементы переписываются с головы новой).
func (q *nodeQueue) push(n queueNode) {
	if q.size == len(q.buf) {
		q.grow()
	}
	q.buf[(q.head+q.size)%len(q.buf)] = n
	q.size++
}

// pop — снять элемент с головы очереди; слот очищается, чтобы уже
// обработанную пару (включая указатель на узел) не удерживал буфер.
func (q *nodeQueue) pop() queueNode {
	cur := q.buf[q.head]
	q.buf[q.head] = queueNode{} // освободившийся слот — нулевой
	q.head = (q.head + 1) % len(q.buf)
	q.size--
	return cur
}

// grow — удвоить буфер, переписав живые элементы подряд от головы;
// после роста head обнуляется, свободного хвоста нет.
func (q *nodeQueue) grow() {
	buf := make([]queueNode, 2*len(q.buf))
	for i := 0; i < q.size; i++ {
		buf[i] = q.buf[(q.head+i)%len(q.buf)]
	}
	q.buf, q.head = buf, 0
}

// paths — живые элементы от головы к хвосту путями от корня: свежий
// слайс, чтобы событие было независимым снимком очереди обхода.
func (q *nodeQueue) paths() []string {
	paths := []string{}
	for i := 0; i < q.size; i++ {
		paths = append(paths, q.buf[(q.head+i)%len(q.buf)].path)
	}
	return paths
}
