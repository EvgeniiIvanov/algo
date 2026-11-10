// Frame Renderer сценария BST: события bst-операций → кадры kind="tree"
// (ADR-0001 + ADR-0002). Дерево начинается с пустого корня; сценарий —
// команды insert/search/delete, валидированные маршрутом до Run.
//
// Каждый кадр — полный актуальный снимок живого дерева: состояние после
// события, активный узел шага и пояснение на русском. Снимок переиспользует
// форму TreeSnapshot обходов (пустые visited/queue), поэтому фронтенд-рендер
// дерева работает для BST без изменений; поле order несёт маркер
// BSTSnapshotOrder — часть контракта TreeSnapshot. Идентификаторы узлов —
// пути от корня (trees.NodeID): при изменении топологии путь-идентификатор
// пересчитывается в новом снимке, сохранение идентичности между кадрами
// не требуется.
//
// Рендерер не повторяет структурную логику BST на копии дерева (ADR-0001):
// снимок всегда читается из живого дерева. События структурных правок
// стреляют раньше, чем библиотека завершит правку (навеска нового узла и
// перезапись потомка происходят при раскрутке рекурсии после возврата
// из вызова), поэтому кадр структурного события откладывается до момента
// применения правки — deferFrame/flush: к первому следующему событию или
// к возврату операции.
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

// BSTSnapshotOrder — значение поля order в снимке TreeSnapshot у сценария
// BST: у обходов поле задаёт порядок обхода, здесь — единственный допустимый
// маркер семейства кадров BST, зафиксированный в контракте TreeSnapshot.
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
		b.key = cmd.Key
		switch cmd.Op {
		case BSTOpInsert:
			b.root, _ = trees.InsertWithEmitter(b.root, cmd.Key, b)
		case BSTOpSearch:
			_, _ = trees.SearchWithEmitter(b.root, cmd.Key, b)
		case BSTOpDelete:
			b.root, _ = trees.DeleteWithEmitter(b.root, cmd.Key, b)
		}
		// Структурные правки библиотека завершает при раскрутке рекурсии,
		// а новый корень операции возвращается сюда: к моменту возврата
		// живое дерево отражает все объявления событий — отложенные кадры
		// структурных событий выпускаются (см. flush).
		b.flush()
		b.cleanup = false
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

// bstFrames — накапливает состояние между событиями (текущее дерево,
// команда и отложенный кадр структурного события) и строит кадры.
// Реализует algorithms.Emitter[trees.BSTEvent].
type bstFrames struct {
	root     *trees.Node       // дерево сценария; меняется структурными операциями
	subject  string            // название текущей команды ("Вставка ключа 5")
	key      int               // ключ текущей команды — для пояснений завершения удаления
	cleanup  bool              // после BSTSuccessor: до конца операции идёт вырезание преемника
	deferred *bstDeferredFrame // структурное событие: кадр ждёт применения правки
	step     int
	emit     func(frame.Frame)
}

// startFrame — исходный кадр Run: пустое дерево и обозрение сценария;
// активного узла нет, имя шага пустое (как у стартового кадра обходов).
func (b *bstFrames) startFrame(commands []BSTCommand) {
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = fmt.Sprintf("%s %d", cmd.Op, cmd.Key)
	}
	b.frame("", "", fmt.Sprintf(
		"Сценарий BST: %s. Команд: %d. Дерево начинается с пустого корня; после каждого события алгоритма — актуальный снимок дерева.",
		strings.Join(names, " → "), len(commands)))
}

// Emit — один шаг алгоритма → один кадр (ADR-0002). Активный узел и
// пояснение задаются по типу события; снимок строится по живому дереву
// на момент выпуска кадра (см. frame/deferFrame). Событие структурной
// правки (BSTInserted, BSTSuccessor, BSTRemoved) объявляет изменение,
// которое библиотека завершит позже, — его кадр откладывается; остальные
// события строят кадр сразу. Активным остаётся только живой узел: у
// пустого места промаха (BSTNotFound) и удалённого узла (BSTRemoved)
// подсветки нет.
func (b *bstFrames) Emit(ev trees.BSTEvent) {
	// Предыдущее структурное событие к моменту нового уже применено
	// (например, перенос значения преемника выполняется сразу после
	// BSTSuccessor): его кадр выпускается перед кадром текущего события.
	b.flush()

	switch e := ev.(type) {
	case trees.BSTCompare:
		b.frame(trees.NodeID(e.Path), b.compareStep(e), b.explainCompare(e))

	case trees.BSTInserted:
		b.deferFrame(trees.NodeID(e.Path), "вставка «"+trees.NodeID(e.Path)+"»", b.explainInserted(e))

	case trees.BSTDuplicate:
		b.frame(trees.NodeID(e.Path), "дубликат «"+trees.NodeID(e.Path)+"»", fmt.Sprintf(
			"%s: узел на пути «%s» уже держит ключ %d. Дерево не меняется: повторные ключи не хранятся, новый узел не добавлен.",
			b.subject, treePathLabel(e.Path), e.Key))

	case trees.BSTFound:
		b.frame(trees.NodeID(e.Path), "найдено «"+trees.NodeID(e.Path)+"»", fmt.Sprintf(
			"%s: ключ %d найден (путь «%s»).",
			b.subject, e.Key, treePathLabel(e.Path)))

	case trees.BSTNotFound:
		b.frame("", "промах", b.explainNotFound(e))

	case trees.BSTRemoved:
		if b.cleanup {
			// Вырезание узла-преемника — шаг завершения удаления узла
			// с двумя потомками, а не удаление первоначального ключа:
			// имя шага и пояснение называют преемника.
			b.deferFrame("", "вырезание преемника «"+trees.NodeID(e.Path)+"»", b.explainSuccessorRemoved(e))
		} else {
			b.deferFrame("", "удаление «"+trees.NodeID(e.Path)+"»", b.explainRemoved(e))
		}

	case trees.BSTMinStep:
		b.frame(trees.NodeID(e.Path), "поиск преемника «"+trees.NodeID(e.Path)+"»", fmt.Sprintf(
			"%s: ищем inorder-successor — минимум правого поддерева: узел %d (путь «%s»); спуск идёт влево, пока есть левый потомок.",
			b.subject, e.Value, treePathLabel(e.Path)))

	case trees.BSTSuccessor:
		// Дальше до конца операции — сравнения и вырезание преемника:
		// их пояснения говорят о преемнике, а не о повторном удалении ключа.
		b.cleanup = true
		b.deferFrame(trees.NodeID(e.Path), "преемник «"+trees.NodeID(e.SuccessorPath)+"»", fmt.Sprintf(
			"%s: у узла (путь «%s») два потомка — inorder-successor: узел %d (путь «%s»), минимум правого поддерева. Значение %d переходит на место удаляемого ключа; узел-преемник будет удалён на следующих шагах.",
			b.subject, treePathLabel(e.Path), e.Value, treePathLabel(e.SuccessorPath), e.Value))
	}
}

// compareStep — имя шага сравнения для поля operation снимка. В фазе
// вырезания преемника шаг помечается отдельно: сравнение ищет узел-
// преемник в правом поддереве, а не повторяет удаление ключа.
func (b *bstFrames) compareStep(e trees.BSTCompare) string {
	if b.cleanup {
		return "вырезание преемника: сравнение с «" + trees.NodeID(e.Path) + "»"
	}
	return "сравнение с «" + trees.NodeID(e.Path) + "»"
}

// explainCompare — пояснение сравнения: с каким узлом сравнили и куда
// идём дальше (у равенства перехода нет — исход решится следующим шагом).
// В фазе вырезания преемника сравнение ищет узел-преемник по его значению:
// пояснение называет его, чтобы кадр не выглядел удалением первоначального
// ключа.
func (b *bstFrames) explainCompare(e trees.BSTCompare) string {
	if b.cleanup {
		return fmt.Sprintf("%s: вырезаем узел-преемник %d — сравниваем с узлом %d (путь «%s»): %s.",
			b.subject, e.Key, e.Value, treePathLabel(e.Path), bstBranchPhrase(e, true))
	}
	return fmt.Sprintf("%s: сравниваем с узлом %d (путь «%s»): %s.",
		b.subject, e.Value, treePathLabel(e.Path), bstBranchPhrase(e, false))
}

// bstBranchPhrase — фраза перехода после сравнения. В фазе вырезания
// преемника равенство означает, что узел-преемник найден: следующим шагом
// его вырежут.
func bstBranchPhrase(e trees.BSTCompare, cleanup bool) string {
	switch e.Branch {
	case "L":
		return fmt.Sprintf("%d < %d — спускаемся влево", e.Key, e.Value)
	case "R":
		return fmt.Sprintf("%d > %d — спускаемся вправо", e.Key, e.Value)
	default:
		if cleanup {
			return "ключ равен значению узла — нашли узел-преемник, следующим шагом вырежем его"
		}
		return fmt.Sprintf("ключ равен значению узла (%d), исход решится следующим шагом", e.Value)
	}
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

// explainRemoved — пояснение структурного изменения удаления: узел
// замещён единственным потомком или пустым местом.
func (b *bstFrames) explainRemoved(e trees.BSTRemoved) string {
	return fmt.Sprintf("%s: узел %d (путь «%s») удалён — на его месте теперь его единственный потомок или пустое место.",
		b.subject, e.Key, treePathLabel(e.Path))
}

// explainSuccessorRemoved — пояснение вырезания узла-преемника: шаг
// завершения удаления узла с двумя потомками; перенесённое значение
// преемника остаётся в дереве на месте первоначального ключа.
func (b *bstFrames) explainSuccessorRemoved(e trees.BSTRemoved) string {
	return fmt.Sprintf("%s: узел-преемник %d (путь «%s») вырезан из правого поддерева — на его месте его единственный потомок или пустое место. Значение %d не исчезло из дерева: оно перенесено на место ключа %d.",
		b.subject, e.Key, treePathLabel(e.Path), e.Key, b.key)
}

// bstDeferredFrame — отложенный кадр структурного события: активный узел,
// имя шага и пояснение известны в момент события, снимок дерева строится
// в момент применения правки.
type bstDeferredFrame struct {
	activeID    string
	operation   string
	explanation string
}

// deferFrame — отложить кадр структурного события (BSTInserted,
// BSTSuccessor, BSTRemoved): событие стреляет раньше, чем библиотека
// завершит структурную правку — новый узел навешивается на родителя и
// потомок перезаписывается при раскрутке рекурсии после возврата из
// вызова. Рендерер не повторяет эти правки на копии дерева (ADR-0001):
// кадр просто выпускается позже, когда живое дерево уже отражает
// объявленное изменение.
func (b *bstFrames) deferFrame(activeID, operation, explanation string) {
	b.deferred = &bstDeferredFrame{activeID: activeID, operation: operation, explanation: explanation}
}

// flush — выпустить отложенный кадр структурного события. Момент выпуска:
// первое следующее событие (перенос значения преемника выполняется сразу
// после BSTSuccessor, поэтому промежуточный шаг — значение перенесено,
// узел-преемник ещё жив — виден в кадре первого сравнения вложенного
// удаления) или возврат операции, когда правка наверняка завершена
// (см. BSTRun).
func (b *bstFrames) flush() {
	if b.deferred == nil {
		return
	}
	d := b.deferred
	b.deferred = nil
	b.frame(d.activeID, d.operation, d.explanation)
}

// frame — собирает самодостаточный кадр (ADR-0002): полный снимок живого
// дерева на момент вызова + подсветки + пояснение. Момент вызова выбран так,
// чтобы снимок показывал дерево после объявленного событием изменения
// (см. deferFrame/flush). Шаги нумеруются с 1: исходный кадр, затем по кадру
// на каждое событие. Активный узел пуст — подсветки нет; имя шага пусто —
// исходный кадр.
func (b *bstFrames) frame(activeID, operation, explanation string) {
	if b.emit == nil {
		return
	}
	b.step++

	nodes := append([]TreeNodeData{}, collectTreeNodes(b.root)...) // не nil: пустое дерево — []
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
