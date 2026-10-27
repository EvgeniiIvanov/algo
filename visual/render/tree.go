// Frame Renderer обходов бинарного дерева: снимок kind="tree" (ADR-0002).
// Кадр самодостаточен: полная топология узлов, стабильные идентификаторы
// (путь от корня, trees.NodeID — не зависят от значения), активный узел,
// уже посещённые узлы с порядковыми номерами и пояснение на русском.
// Запуск — с исходного кадра, далее по кадру на каждое событие DFSVisit.
package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Порядки обхода — общие для маршрута, снимка и фронтенда.
const (
	TreeOrderInOrder   = "in-order"
	TreeOrderPreOrder  = "pre-order"
	TreeOrderPostOrder = "post-order"

	treeOrderUnknownMsg = "неизвестный порядок обхода: ожидается in-order, pre-order или post-order"
)

// TreeNodeData — один узел в снимке: значение и связи соседей по ID
// ("" — потомка нет). Идентификатор — путь от корня (trees.NodeID),
// поэтому повторяющиеся значения остаются различимыми узлами.
type TreeNodeData struct {
	ID      string `json:"id"`
	Value   int    `json:"value"`
	LeftID  string `json:"leftId"`  // "" — левого потомка нет
	RightID string `json:"rightId"` // "" — правого потомка нет
}

// TreeVisit — уже посещённый узел с его номером в порядке обхода (с 1).
type TreeVisit struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// TreeSnapshot — снимок дерева в кадре kind="tree": полная топология,
// активный узел текущего шага и посещённая последовательность. Форма
// произвольная: снимок не предполагает свойств BST. Пустое дерево —
// пустой список Nodes и пустой RootID.
type TreeSnapshot struct {
	Order     string         `json:"order"`     // порядок обхода: in-order / pre-order / post-order
	Nodes     []TreeNodeData `json:"nodes"`     // все узлы дерева: полный снимок топологии (не nil: пустое дерево — [])
	RootID    string         `json:"rootId"`    // идентификатор корня; "" — пустое дерево
	ActiveID  string         `json:"activeId"`  // посещаемый узел шага; "" — стартовый кадр
	Visited   []TreeVisit    `json:"visited"`   // посещённые узлы с порядковыми номерами, с 1 (не nil)
	Operation string         `json:"operation"` // имя шага ("старт", "посещение LR"); пусто — стартовый кадр
}

// TreeTraversalRun — Frame Renderer глубинных обходов: прогоняет обход
// дерева порядка order (in-order, pre-order, post-order) в режиме
// наблюдения (ADR-0001) и строит кадры по событиям.
// Порядок валидируется маршрутом до Run; здесь неизвестный порядок —
// ошибка конструктора, кадры не создаются.
func TreeTraversalRun(root *trees.Node, order string, emit func(frame.Frame)) error {
	name, ok := treeOrderName(order)
	if !ok {
		return fmt.Errorf("%s", treeOrderUnknownMsg)
	}

	r := &treeFrames{
		order: order,
		name:  name,
		nodes: collectTreeNodes(root),
		emit:  emit,
	}
	r.frame(fmt.Sprintf("Обход дерева: %s. Узлов в дереве: %d. Каждый шаг — посещение очередного узла.", name, len(r.nodes)))

	switch order {
	case TreeOrderInOrder:
		trees.InOrderWithEmitter(root, r)
	case TreeOrderPreOrder:
		trees.PreOrderWithEmitter(root, r)
	default:
		trees.PostOrderWithEmitter(root, r)
	}
	return nil
}

// TreeFrames — все кадры обхода сразу (удобно для тестов).
func TreeFrames(root *trees.Node, order string) ([]frame.Frame, error) {
	var frames []frame.Frame
	if err := TreeTraversalRun(root, order, func(f frame.Frame) { frames = append(frames, f) }); err != nil {
		return nil, err
	}
	return frames, nil
}

// treeOrderName — человеческое имя порядка на русском; второе значение —
// валиден ли порядок.
func treeOrderName(order string) (string, bool) {
	switch order {
	case TreeOrderInOrder:
		return "in-order (левое поддерево, узел, правое поддерево)", true
	case TreeOrderPreOrder:
		return "pre-order (узел, левое поддерево, правое поддерево)", true
	case TreeOrderPostOrder:
		return "post-order (левое поддерево, правое поддерево, узел)", true
	}
	return "", false
}

// treePathLabel — путь узла в живой форме для пояснений: корень — «корень»,
// остальные — путь буквами L/R (e.g. «LR»).
func treePathLabel(path string) string {
	if path == "" {
		return "корень"
	}
	return path
}

// collectTreeNodes — полный снимок топологии один раз до обхода: узлы
// с идентификаторами по пути от корня (trees.NodeID); корень первым,
// дальше в глубину слева направо.
func collectTreeNodes(root *trees.Node) []TreeNodeData {
	var nodes []TreeNodeData
	collectFrom(root, "", &nodes)
	return nodes
}

func collectFrom(n *trees.Node, path string, nodes *[]TreeNodeData) {
	if n == nil {
		return
	}
	id := trees.NodeID(path)
	node := TreeNodeData{ID: id, Value: n.Value}
	if n.Left != nil {
		node.LeftID = trees.NodeID(path + "L")
	}
	if n.Right != nil {
		node.RightID = trees.NodeID(path + "R")
	}
	*nodes = append(*nodes, node)
	collectFrom(n.Left, path+"L", nodes)
	collectFrom(n.Right, path+"R", nodes)
}

// treeFrames — накапливает посещённую последовательность между событиями
// и строит кадры. Реализует algorithms.Emitter[trees.DFSEvent].
type treeFrames struct {
	order    string         // порядок обхода, как в маршруте
	name     string         // человеческое имя порядка для пояснений
	nodes    []TreeNodeData // полный снимок топологии, собранный один раз
	activeID string         // активный узел текущего шага; "" — стартовый кадр
	visited  []TreeVisit    // посещённая последовательность
	step     int
	emit     func(frame.Frame)
}

// Emit — одно событие посещения → один кадр (ADR-0002).
func (r *treeFrames) Emit(ev trees.DFSEvent) {
	switch e := ev.(type) {
	case trees.DFSVisit:
		id := trees.NodeID(e.Path)
		r.activeID = id
		r.visited = append(r.visited, TreeVisit{ID: id, Position: len(r.visited) + 1})
		r.frame(fmt.Sprintf("%s: посещаем узел %d (путь от корня «%s») — %d-й по порядку.",
			r.orderPrefix(), e.Value, treePathLabel(e.Path), len(r.visited)))
	}
}

// orderPrefix — человеческое имя порядка с большой буквы для начала фразы.
func (r *treeFrames) orderPrefix() string {
	switch r.order {
	case TreeOrderInOrder:
		return "In-order"
	case TreeOrderPreOrder:
		return "Pre-order"
	default:
		return "Post-order"
	}
}

// frame собирает самодостаточный кадр: полная топология + активный узел +
// посещённые узлы. Шаги нумеруются с 1: стартовый кадр, затем по кадру
// на каждое событие.
func (r *treeFrames) frame(explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	rootID := ""
	operation := ""
	for _, n := range r.nodes {
		if n.ID == trees.NodeID("") {
			rootID = n.ID
		}
	}
	if r.activeID != "" {
		operation = "посещение " + r.activeID
	}
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindTree,
		Explanation: explanation,
		Data: TreeSnapshot{
			Order:     r.order,
			Nodes:     append([]TreeNodeData{}, r.nodes...), // не nil: пустое дерево — []
			RootID:    rootID,
			ActiveID:  r.activeID,
			Visited:   append([]TreeVisit{}, r.visited...), // не nil: пустая последовательность — []
			Operation: operation,
		},
	})
}
