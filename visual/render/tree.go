// Frame Renderer обходов бинарного дерева: снимок kind="tree" (ADR-0002).
// Кадр самодостаточен: полная топология узлов, стабильные идентификаторы
// (путь от корня, trees.NodeID — не зависят от значения), активный узел,
// уже посещённые узлы с порядковыми номерами, FIFO-очередь у обхода по
// уровням и пояснение на русском. Запуск — с исходного кадра, далее по
// кадру на каждое событие посещения (DFSVisit / LevelOrderVisit).
package render

import (
	"fmt"
	"strings"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Порядки обхода — общие для маршрута, снимка и фронтенда.
const (
	TreeOrderInOrder    = "in-order"
	TreeOrderPreOrder   = "pre-order"
	TreeOrderPostOrder  = "post-order"
	TreeOrderLevelOrder = "level-order"

	treeOrderUnknownMsg = "неизвестный порядок обхода: ожидается in-order, pre-order, post-order или level-order"
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
// пустой список Nodes и пустой RootID. Queue — FIFO-очередь обхода
// по уровням (идентификаторы от головы к хвосту); у глубинных порядков
// очередь всегда пустая ([]). Поле Order задаёт порядок обхода, а у
// сценария BST — маркер семейства кадров "bst" (BSTSnapshotOrder):
// visited и queue у BST всегда пустые.
type TreeSnapshot struct {
	Order     string         `json:"order"`     // порядок обхода или маркер семейства кадров: in-order / pre-order / post-order / level-order (обходы), "bst" (BSTSnapshotOrder) — сценарий BST
	Nodes     []TreeNodeData `json:"nodes"`     // все узлы дерева: полный снимок топологии (не nil: пустое дерево — [])
	RootID    string         `json:"rootId"`    // идентификатор корня; "" — пустое дерево
	ActiveID  string         `json:"activeId"`  // посещаемый узел шага; "" — стартовый кадр
	Visited   []TreeVisit    `json:"visited"`   // посещённые узлы с порядковыми номерами, с 1 (не nil)
	Queue     []string       `json:"queue"`     // FIFO-очередь level-order, идентификаторы от головы к хвосту (не nil)
	Operation string         `json:"operation"` // имя шага ("старт", "посещение LR"); пусто — стартовый кадр
}

// TreeTraversalRun — Frame Renderer обходов дерева: прогоняет обход
// порядка order (in-order, pre-order, post-order, level-order) в режиме
// наблюдения (ADR-0001) и строит кадры по событиям. События двух семейств
// (DFSVisit и LevelOrderVisit) адаптируются к общему сборщику кадров.
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
	startText := fmt.Sprintf("Обход дерева: %s. Узлов в дереве: %d. Каждый шаг — посещение очередного узла.", name, len(r.nodes))
	if order == TreeOrderLevelOrder {
		startText = fmt.Sprintf("Обход дерева: %s. Узлов в дереве: %d. Каждый шаг — снять узел с головы FIFO-очереди, посетить его и добавить в конец очереди его потомков.", name, len(r.nodes))
		if root != nil {
			r.queue = []string{trees.NodeID("")} // корень ждёт в очереди с первого кадра
		}
	}
	r.frame(startText)

	switch order {
	case TreeOrderInOrder:
		trees.InOrderWithEmitter(root, treeDFSEmitter{r})
	case TreeOrderPreOrder:
		trees.PreOrderWithEmitter(root, treeDFSEmitter{r})
	case TreeOrderPostOrder:
		trees.PostOrderWithEmitter(root, treeDFSEmitter{r})
	default:
		trees.LevelOrderWithEmitter(root, treeLevelEmitter{r})
	}
	return nil
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
	case TreeOrderLevelOrder:
		return "level-order (по уровням: сверху вниз, слева направо; FIFO-очередь)", true
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

// treeFrames — накапливает состояние между событиями (активный узел,
// посещённая последовательность, очередь level-order) и строит кадры.
// Общий сборщик для двух семейств событий; адаптеры-эмиттеры ниже
// переводят DFSVisit и LevelOrderVisit в изменения этого состояния.
type treeFrames struct {
	order    string         // порядок обхода, как в маршруте
	name     string         // человеческое имя порядка для пояснений
	nodes    []TreeNodeData // полный снимок топологии, собранный один раз
	activeID string         // активный узел текущего шага; "" — стартовый кадр
	visited  []TreeVisit    // посещённая последовательность
	queue    []string       // FIFO-очередь level-order от головы к хвосту; у DFS — пустая
	step     int
	emit     func(frame.Frame)
}

// treeDFSEmitter — адаптер algorithms.Emitter[trees.DFSEvent] к сборщику
// кадров: одно событие посещения → одно изменение состояния и один кадр.
type treeDFSEmitter struct{ f *treeFrames }

func (e treeDFSEmitter) Emit(ev trees.DFSEvent) {
	visit, ok := ev.(trees.DFSVisit)
	if !ok {
		return
	}
	r := e.f
	id := trees.NodeID(visit.Path)
	r.activeID = id
	r.visited = append(r.visited, TreeVisit{ID: id, Position: len(r.visited) + 1})
	r.frame(fmt.Sprintf("%s: посещаем узел %d (путь от корня «%s») — %d-й по порядку.",
		r.orderPrefix(), visit.Value, treePathLabel(visit.Path), len(r.visited)))
}

// treeLevelEmitter — адаптер algorithms.Emitter[trees.LevelOrderEvent]:
// узел снят с головы очереди и посещён, потомки встали в конец; состояние
// очереди снимка берётся из события, пояснение объясняет оба движения.
type treeLevelEmitter struct{ f *treeFrames }

func (e treeLevelEmitter) Emit(ev trees.LevelOrderEvent) {
	visit, ok := ev.(trees.LevelOrderVisit)
	if !ok {
		return
	}
	r := e.f
	id := trees.NodeID(visit.Path)
	r.activeID = id
	r.visited = append(r.visited, TreeVisit{ID: id, Position: len(r.visited) + 1})
	r.queue = append([]string{}, visit.Queue...)
	r.frame(r.levelExplanation(visit))
}

// levelExplanation — пояснение шага обхода по уровням: снятие с головы
// очереди, добавление существующих потомков в конец (отдельно каждый) и
// состояние очереди после шага. Потомки берутся из топологии снимка:
// существующие потомки активного узла — это ровно те, что встали в
// конец очереди на этом шаге.
func (r *treeFrames) levelExplanation(ev trees.LevelOrderVisit) string {
	id := trees.NodeID(ev.Path)
	text := fmt.Sprintf("%s: снимаем с головы FIFO-очереди узел %d (путь от корня «%s») и посещаем его — %d-й по порядку.",
		r.orderPrefix(), ev.Value, treePathLabel(ev.Path), len(r.visited))
	children := childIDs(r.nodes, id)
	switch len(children) {
	case 1:
		text += fmt.Sprintf(" Его потомок «%s» встаёт в конец очереди.", children[0])
	case 2:
		text += fmt.Sprintf(" Его потомки «%s» и «%s» встают в конец очереди.", children[0], children[1])
	}
	if len(r.queue) == 0 {
		return text + " Очередь опустела: все узлы посещены."
	}
	return fmt.Sprintf("%s В очереди сейчас: %s.", text, strings.Join(r.queue, ", "))
}

// childIDs — идентификаторы существующих потомков узла id по топологии
// снимка, слева направо (у level-order это ровно потомки, вставшие в
// конец очереди на этом шаге).
func childIDs(nodes []TreeNodeData, id string) []string {
	for _, n := range nodes {
		if n.ID != id {
			continue
		}
		var children []string
		if n.LeftID != "" {
			children = append(children, n.LeftID)
		}
		if n.RightID != "" {
			children = append(children, n.RightID)
		}
		return children
	}
	return nil
}

// orderPrefix — человеческое имя порядка с большой буквы для начала фразы.
func (r *treeFrames) orderPrefix() string {
	switch r.order {
	case TreeOrderInOrder:
		return "In-order"
	case TreeOrderPreOrder:
		return "Pre-order"
	case TreeOrderLevelOrder:
		return "Level-order"
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
			Queue:     append([]string{}, r.queue...),      // не nil: пустая очередь — []
			Operation: operation,
		},
	})
}
