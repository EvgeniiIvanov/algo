// Frame Renderer сценария BST: события bst-операций → кадры kind="tree"
// (ADR-0001 + ADR-0002). Дерево начинается с пустого корня; сценарий —
// команды insert/search/delete, валидированные маршрутом до Run.
//
// Каждый кадр — полный актуальный снимок: состояние дерева после события,
// активный узел шага и пояснение на русском. Снимок переиспользует форму
// TreeSnapshot обходов (пустые visited/queue), поэтому фронтенд-рендер
// дерева работает для BST без изменений; поле order несёт единственное
// значение BSTSnapshotOrder. Идентификаторы узлов — пути от корня
// (trees.NodeID): при изменении топологии путь-идентификатор пересчитывается
// в новом снимке, сохранение идентичности между кадрами не требуется.
package render

import (
	"fmt"
	"strings"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Команды сценария BST — общие для маршрута, рендерера и фронтенда.
const (
	BSTOpInsert = "insert"
	BSTOpSearch = "search"
	BSTOpDelete = "delete"
)

// BSTSnapshotOrder — значение поля order в снимке kind="tree" у сценария
// BST: у обходов поле задаёт порядок обхода, здесь — единственный допустимый
// маркер семейства кадров BST.
const BSTSnapshotOrder = "bst"

// BSTCommand — одна команда сценария BST: действие Op над ключом Key.
type BSTCommand struct {
	Op  string // BSTOpInsert / BSTOpSearch / BSTOpDelete
	Key int    // ключ операции — целое число, валидированное маршрутом
}

// BSTRun — Frame Renderer сценария BST: прогоняет команды над деревом с
// пустым корнем в режиме наблюдения (ADR-0001) и на каждое событие отдаёт
// кадр в emit. Валидация сценария (синтаксис, диапазон ключей, лимиты
// команд и живых узлов) живёт в маршруте — до Run ничего недопустимого
// остаться не может, поэтому ошибки здесь не возвращаются.
func BSTRun(commands []BSTCommand, emit func(frame.Frame)) {
	b := &bstFrames{emit: emit}
	b.startFrame(commands)

	for _, cmd := range commands {
		b.subject = bstSubject(cmd)
		switch cmd.Op {
		case BSTOpInsert:
			b.root, _ = trees.InsertWithEmitter(b.root, cmd.Key, b)
		case BSTOpSearch:
			_, _ = trees.SearchWithEmitter(b.root, cmd.Key, b)
		case BSTOpDelete:
			b.root, _ = trees.DeleteWithEmitter(b.root, cmd.Key, b)
		}
	}
}

// bstSubject — человеческое название текущей команды для пояснений.
func bstSubject(cmd BSTCommand) string {
	switch cmd.Op {
	case BSTOpInsert:
		return fmt.Sprintf("Вставка ключа %d", cmd.Key)
	case BSTOpSearch:
		return fmt.Sprintf("Поиск ключа %d", cmd.Key)
	default:
		return fmt.Sprintf("Удаление ключа %d", cmd.Key)
	}
}

// bstFrames — накапливает состояние между событиями (текущее дерево и
// команда) и строит кадры. Реализует algorithms.Emitter[trees.BSTEvent].
type bstFrames struct {
	root    *trees.Node // дерево сценария; меняется структурными операциями
	subject string      // название текущей команды ("Вставка ключа 5")
	step    int
	emit    func(frame.Frame)
}

// startFrame — исходный кадр Run: пустое дерево и обозрение сценария;
// активного узла нет, имя шага пустое (как у стартового кадра обходов).
func (b *bstFrames) startFrame(commands []BSTCommand) {
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = fmt.Sprintf("%s %d", cmd.Op, cmd.Key)
	}
	b.frame(nil, "", "", fmt.Sprintf(
		"Сценарий BST: %s. Команд: %d. Дерево начинается с пустого корня; после каждого события алгоритма — актуальный снимок дерева.",
		strings.Join(names, " → "), len(commands)))
}

// Emit — один шаг алгоритма → один кадр (ADR-0002). Активный узел и
// пояснение задаются по типу события; снимок строится по актуальному
// состоянию дерева после события (см. frameNodes). Активным остаётся
// только живой узел: у пустого места промаха (BSTNotFound) и удалённого
// узла (BSTRemoved) подсветки нет.
func (b *bstFrames) Emit(ev trees.BSTEvent) {
	switch e := ev.(type) {
	case trees.BSTCompare:
		b.frame(ev, trees.NodeID(e.Path), bstCompareStep(e), b.explainCompare(e))

	case trees.BSTInserted:
		b.frame(ev, trees.NodeID(e.Path), "вставка «"+trees.NodeID(e.Path)+"»", b.explainInserted(e))

	case trees.BSTDuplicate:
		b.frame(ev, trees.NodeID(e.Path), "дубликат «"+trees.NodeID(e.Path)+"»", fmt.Sprintf(
			"%s: узел на пути «%s» уже держит ключ %d. Дерево не меняется: повторные ключи не хранятся, новый узел не добавлен.",
			b.subject, treePathLabel(e.Path), e.Key))

	case trees.BSTFound:
		b.frame(ev, trees.NodeID(e.Path), "найдено «"+trees.NodeID(e.Path)+"»", fmt.Sprintf(
			"%s: ключ %d найден (путь «%s»).",
			b.subject, e.Key, treePathLabel(e.Path)))

	case trees.BSTNotFound:
		b.frame(ev, "", "промах", b.explainNotFound(e))

	case trees.BSTRemoved:
		b.frame(ev, "", "удаление «"+trees.NodeID(e.Path)+"»", fmt.Sprintf(
			"%s: узел %d (путь «%s») удалён — на его месте теперь его единственный потомок или пустое место.",
			b.subject, e.Key, treePathLabel(e.Path)))

	case trees.BSTMinStep:
		b.frame(ev, trees.NodeID(e.Path), "поиск преемника «"+trees.NodeID(e.Path)+"»", fmt.Sprintf(
			"%s: ищем inorder-successor — минимум правого поддерева: узел %d (путь «%s»); спуск идёт влево, пока есть левый потомок.",
			b.subject, e.Value, treePathLabel(e.Path)))

	case trees.BSTSuccessor:
		b.frame(ev, trees.NodeID(e.Path), "преемник «"+trees.NodeID(e.SuccessorPath)+"»", fmt.Sprintf(
			"%s: у узла (путь «%s») два потомка — inorder-successor: узел %d (путь «%s»), минимум правого поддерева. Значение %d переходит на место удаляемого ключа; узел-преемник будет удалён на следующих шагах.",
			b.subject, treePathLabel(e.Path), e.Value, treePathLabel(e.SuccessorPath), e.Value))
	}
}

// bstCompareStep — имя шага сравнения для поля operation снимка.
func bstCompareStep(e trees.BSTCompare) string {
	return "сравнение с «" + trees.NodeID(e.Path) + "»"
}

// explainCompare — пояснение сравнения: с каким узлом сравнили и куда
// идём дальше (у равенства перехода нет — исход решится следующим шагом).
func (b *bstFrames) explainCompare(e trees.BSTCompare) string {
	var where string
	switch e.Branch {
	case "L":
		where = fmt.Sprintf("%d < %d — спускаемся влево", e.Key, e.Value)
	case "R":
		where = fmt.Sprintf("%d > %d — спускаемся вправо", e.Key, e.Value)
	default:
		where = fmt.Sprintf("ключ равен значению узла (%d), исход решится следующим шагом", e.Value)
	}
	return fmt.Sprintf("%s: сравниваем с узлом %d (путь «%s»): %s.",
		b.subject, e.Value, treePathLabel(e.Path), where)
}

// explainInserted — пояснение структурного изменения вставки.
func (b *bstFrames) explainInserted(e trees.BSTInserted) string {
	if e.Path == "" {
		return fmt.Sprintf("%s: дерево пустое — новый узел %d становится корнем.", b.subject, e.Key)
	}
	return fmt.Sprintf("%s: пустое место достигнуто — добавляем новый узел %d (путь «%s»).",
		b.subject, e.Key, treePathLabel(e.Path))
}

// explainNotFound — промах поиска или удаления: штатный исход, а не
// ошибка сценария (SPEC); вывод уточняется текущей операцией.
func (b *bstFrames) explainNotFound(e trees.BSTNotFound) string {
	var outcome string
	switch subject := b.subject; {
	case strings.HasPrefix(subject, "Поиск"):
		outcome = "Промах — штатный исход поиска."
	default:
		outcome = "Дерево не меняется: промах — штатный исход удаления."
	}
	if e.Path == "" {
		return fmt.Sprintf("%s: дерево пустое — ключа %d в дереве нет. %s",
			b.subject, e.Key, outcome)
	}
	return fmt.Sprintf("%s: спуск дошёл до пустого места (путь «%s») — ключа %d в дереве нет. %s",
		b.subject, treePathLabel(e.Path), e.Key, outcome)
}

// frame — собирает самодостаточный кадр (ADR-0002): полный снимок дерева
// после события + подсветки + пояснение. Шаги нумеруются с 1: исходный
// кадр, затем по кадру на каждое событие. Активный узел пуст — подсветки
// нет; имя шага пусто — исходный кадр.
func (b *bstFrames) frame(ev trees.BSTEvent, activeID, operation, explanation string) {
	if b.emit == nil {
		return
	}
	b.step++

	nodes := append([]TreeNodeData{}, b.frameNodes(ev)...) // не nil: пустое дерево — []
	rootID := ""
	for _, n := range nodes {
		if n.ID == trees.NodeID("") {
			rootID = n.ID
			break
		}
	}

	b.emit(frame.Frame{
		Step:        b.step,
		Kind:        frame.KindTree,
		Explanation: explanation,
		Data: TreeSnapshot{
			Order:     BSTSnapshotOrder,
			Nodes:     nodes,
			RootID:    rootID,
			ActiveID:  activeID,
			Visited:   []TreeVisit{},
			Queue:     []string{},
			Operation: operation,
		},
	})
}

// frameNodes — узлы снимка: актуальное состояние дерева после события.
// Событие стреляет раньше, чем библиотека завершит структурную правку
// (рекурсивная операция делает rewire после возврата из вызова — например,
// родитель переприсваивает потомка после BSTRemoved). Поэтому объявленное
// событием изменение применяется к клону дерева: живое дерево продолжает
// работать без вмешательства рендерера, а между событиями библиотека
// успевает завершить правки предыдущего события, так что следующий кадр
// читает уже учтённое состояние. Событие nil — стартовый кадр без правок.
func (b *bstFrames) frameNodes(ev trees.BSTEvent) []TreeNodeData {
	tree := cloneTree(b.root)
	switch e := ev.(type) {
	case trees.BSTInserted:
		tree = insertNodeToClone(tree, e.Path, e.Key)
	case trees.BSTSuccessor:
		markCloneValue(tree, e.Path, e.Value)
	case trees.BSTRemoved:
		tree = replaceCloneNode(tree, e.Path)
	}
	return collectTreeNodes(tree)
}

// cloneTree — полная копия дерева для снимка кадра: структурные правки
// рендерер вносит только в собственной копии, живое дерево не трогает.
func cloneTree(n *trees.Node) *trees.Node {
	if n == nil {
		return nil
	}
	c := trees.NewNode(n.Value)
	c.Left = cloneTree(n.Left)
	c.Right = cloneTree(n.Right)
	return c
}

// insertNodeToClone — применить к клону объявление BSTInserted: в пустом
// месте path появляется новый лист; у корня (path "") дерево пустое —
// новый узел становится корнем.
func insertNodeToClone(tree *trees.Node, path string, key int) *trees.Node {
	if path == "" {
		return trees.NewNode(key)
	}
	parent := nodeAt(tree, parentPath(path))
	if path[len(path)-1] == 'L' {
		parent.Left = trees.NewNode(key)
	} else {
		parent.Right = trees.NewNode(key)
	}
	return tree
}

// markCloneValue — применить к клону объявление BSTSuccessor: значение
// преемника переходит узлу на пути path (в живом дереве та же правка
// выполняется сразу после события).
func markCloneValue(tree *trees.Node, path string, value int) {
	nodeAt(tree, path).Value = value
}

// replaceCloneNode — применить к клону объявление BSTRemoved: узел на
// пути path замещается единственным потомком (лист — пустотой). У корня
// (path "") замещённый узел — сам корень: дерево сжимается к потомку.
func replaceCloneNode(tree *trees.Node, path string) *trees.Node {
	replace := func(n *trees.Node) *trees.Node {
		if n.Left != nil {
			return n.Left
		}
		return n.Right // лист — nil: место пустое
	}
	if path == "" {
		return replace(tree)
	}
	parent := nodeAt(tree, parentPath(path))
	if path[len(path)-1] == 'L' {
		parent.Left = replace(parent.Left)
	} else {
		parent.Right = replace(parent.Right)
	}
	return tree
}

// nodeAt — узел клона на пути path ("": корень). Путь события существует
// в дереве на момент события: события не ссылаются на отсутствующие узлы.
func nodeAt(n *trees.Node, path string) *trees.Node {
	for _, ch := range path {
		if ch == 'L' {
			n = n.Left
		} else {
			n = n.Right
		}
	}
	return n
}

// parentPath — путь родителя: без последнего символа ("" — корень).
func parentPath(path string) string {
	return path[:len(path)-1]
}
