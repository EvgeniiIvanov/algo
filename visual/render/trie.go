// Frame Renderer сценария Trie: события trie-операций → кадры kind="trie"
// (ADR-0001 + ADR-0002). Trie начинается с пустого корня; сценарий — команды
// insert/search/prefix/delete, валидированные маршрутом до Run.
//
// Каждый кадр — полный самодостаточный снимок Trie: топология с произвольным
// ветвлением, отметки конца слова, активный путь текущей операции, проверка
// ребра (edgeCheck) и результат команды (result) — плюс имя операции и
// пояснение на русском. Идентификаторы узлов — frame-local пути от корня по
// rune-меткам (Trie Node ID, GLOSSARY): межкадровая стабильность не обещается,
// L/R-контракт бинарного NodeID не переиспользуется.
//
// Рендерер не заглядывает во внутренности библиотечного Trie (map переходов
// NodeTrie не экспортирован) и держит зеркало топологии из событий (ADR-0001:
// события достаточны для восстановления) — как зеркало корзин у хеш-таблицы.
// Зеркало применяется в момент события, поэтому кадры pruning показывают уже
// срезанные узлы: «ставшие ненужными узлы исчезают».
package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Команды сценария Trie — общие для маршрута, рендерера и фронтенда.
const (
	TrieOpInsert = "insert"
	TrieOpSearch = "search"
	TrieOpPrefix = "prefix"
	TrieOpDelete = "delete"
)

// TrieResult — итог команды в кадре результата операции: закрытое множество
// значений, одно на исход валидной команды.
type TrieResult string

const (
	TrieResultInsert         TrieResult = "insert"
	TrieResultAlreadyExisted TrieResult = "already-existed"
	TrieResultFound          TrieResult = "found"
	TrieResultNotFound       TrieResult = "not-found"
	TrieResultPrefixFound    TrieResult = "prefix-found"
	TrieResultPrefixNotFound TrieResult = "prefix-not-found"
	TrieResultDeleted        TrieResult = "deleted"
	TrieResultNotDeleted     TrieResult = "not-deleted"
)

// trieRootID — идентификатор корня в снимке (корень-заглушка есть всегда,
// в том числе в пустом Trie).
const trieRootID = "root"

// TrieCommand — одна команда сценария Trie: действие Op над словом Word
// (для prefix — префиксом).
type TrieCommand struct {
	Op   string // TrieOpInsert / TrieOpSearch / TrieOpPrefix / TrieOpDelete
	Word string // непустой аргумент в валидном UTF-8, валидирован маршрутом
}

// TrieNodeData — один узел в снимке: идентификатор (frame-local путь от
// корня), отметка конца слова и идентификаторы потомков по возрастанию кода
// руны (детерминированный порядок снимка).
type TrieNodeData struct {
	ID       string   `json:"id"`
	IsEnd    bool     `json:"isEnd"`
	Children []string `json:"children"` // идентификаторы потомков; не nil: у листа — []
}

// TrieEdgeCheck — проверка ребра по руне в кадре посещения/проверки: Rune —
// JSON-строка ровно из одной руны (не числовой код), State — состояние ребра
// библиотечного события (reused / created / missing).
type TrieEdgeCheck struct {
	Rune  string `json:"rune"`
	State string `json:"state"`
}

// TrieSnapshot — снимок Trie в кадре kind="trie". Самодостаточен (ADR-0002):
// полная топология копируется в каждый кадр. ActivePath — активный путь
// текущего шага от корня (только существующие узлы: при missing — до родителя
// отсутствующего ребра; корень включён всегда); в кадрах pruning и результата
// операции активного пути нет ([]). EdgeCheck не nil только в кадре
// посещения/проверки; Result не nil только в кадре результата операции —
// nil маршалится как JSON null.
type TrieSnapshot struct {
	Nodes      []TrieNodeData `json:"nodes"`      // все узлы снимка (корень первым); не nil
	RootID     string         `json:"rootId"`     // всегда "root": корень есть и в пустом Trie
	ActivePath []string       `json:"activePath"` // активный путь шага; не nil: пустой путь — []
	EdgeCheck  *TrieEdgeCheck `json:"edgeCheck"`  // nil — шаг без проверки ребра: JSON null
	Result     *TrieResult    `json:"result"`     // nil до кадра результата: JSON null
	Operation  string         `json:"operation"`  // имя операции шага ("insert кот"); пусто — стартовый кадр
}

// TrieRun — Frame Renderer сценария Trie: прогоняет команды над Trie
// с пустым корнем в режиме наблюдения (ADR-0001) и на каждое событие отдаёт
// кадр в emit. Валидация сценария (синтаксис, валидный UTF-8, лимиты команд,
// длины и живых узлов) живёт в маршруте — до Run ничего недопустимого
// остаться не может. Ошибка здесь — только защитная от сценария,
// просочившегося мимо валидации маршрута: неизвестный вид операции или
// отклонение библиотеки.
func TrieRun(commands []TrieCommand, emit func(frame.Frame)) error {
	t := trees.NewTrie()
	r := &trieFrames{emit: emit}
	r.startFrame(commands)

	for _, cmd := range commands {
		r.beginCommand(cmd)
		var err error
		switch cmd.Op {
		case TrieOpInsert:
			_, err = t.InsertWithEmitter(cmd.Word, r)
		case TrieOpSearch:
			_, err = t.SearchWithEmitter(cmd.Word, r)
		case TrieOpPrefix:
			_, err = t.StartsWithWithEmitter(cmd.Word, r)
		case TrieOpDelete:
			_, err = t.DeleteWithEmitter(cmd.Word, r)
		default:
			// Недостижимо: parseTrieScript маршрута пропускает только известные виды.
			return fmt.Errorf("неизвестный вид операции %q", cmd.Op)
		}
		if err != nil {
			return fmt.Errorf("команда %s %s: %w", cmd.Op, cmd.Word, err)
		}
	}
	return nil
}

// TrieFrames — все кадры сценария сразу (удобно для тестов).
func TrieFrames(commands []TrieCommand) ([]frame.Frame, error) {
	var frames []frame.Frame
	if err := TrieRun(commands, func(f frame.Frame) { frames = append(frames, f) }); err != nil {
		return nil, err
	}
	return frames, nil
}

// trieMirrorNode — узел зеркала Trie: рендерер восстанавливает топологию
// из событий (ADR-0001) и не заглядывает во внутренности библиотеки.
type trieMirrorNode struct {
	children map[rune]*trieMirrorNode
	isEnd    bool
}

// trieFrames — накапливает состояние между событиями (зеркало Trie, активный
// путь и контекст текущей команды) и строит кадры. Реализует
// algorithms.Emitter[trees.TrieEvent]: одно событие — один кадр (ADR-0002).
type trieFrames struct {
	mirror     *trieMirrorNode // корень зеркала Trie (заглушка, без отметки)
	cur        *trieMirrorNode // узел зеркала, до которого дошёл обход команды
	path       []rune          // руны команды, пройденные до существующих узлов
	activePath []string        // активный путь текущего шага: идентификаторы от корня
	operation  string          // имя операции текущей команды ("insert кот")
	subject    string          // название команды для пояснений ("Вставка слова «кот»")
	word       string          // слово текущей команды — для пояснений
	step       int             // нумерация кадров с 1
	emit       func(frame.Frame)
}

// startFrame — исходный кадр Run: пустое дерево и обозрение сценария;
// активного пути нет, имя шага пустое (как у стартового кадра обходов).
func (r *trieFrames) startFrame(commands []TrieCommand) {
	r.mirror = &trieMirrorNode{children: map[rune]*trieMirrorNode{}}
	r.cur = r.mirror
	r.activePath = []string{}
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Op + " " + cmd.Word
	}
	r.frame(nil, nil, fmt.Sprintf(
		"Сценарий Trie: %s. Команд: %d. Дерево начинается с пустого корня; после каждого события алгоритма — актуальный снимок дерева.",
		strings.Join(names, " → "), len(commands)))
}

// beginCommand — контекст новой команды: зеркало обхода — с корня, активный
// путь начинается с корня, имя операции и пояснительное название заданы.
func (r *trieFrames) beginCommand(cmd TrieCommand) {
	r.cur = r.mirror
	r.path = nil
	r.activePath = []string{trieRootID}
	r.operation = cmd.Op + " " + cmd.Word
	r.subject = trieSubject(cmd)
	r.word = cmd.Word
}

// trieSubject — человеческое название команды для пояснений.
func trieSubject(cmd TrieCommand) string {
	switch cmd.Op {
	case TrieOpInsert:
		return "Вставка слова «" + cmd.Word + "»"
	case TrieOpSearch:
		return "Поиск слова «" + cmd.Word + "»"
	case TrieOpPrefix:
		return "Проверка префикса «" + cmd.Word + "»"
	default:
		return "Удаление слова «" + cmd.Word + "»"
	}
}

// Emit — одно событие Trie-операции → один кадр (ADR-0002). Зеркало
// применяется в момент события: узел created к кадру уже создан (библиотека
// навешивает его до события посещения), срезанный pruning-узел к кадру уже
// срезан (библиотека стреляет событием до правки — кадр показывает
// исчезнувший узел).
func (r *trieFrames) Emit(ev trees.TrieEvent) {
	switch e := ev.(type) {
	case trees.TrieVisit:
		r.visit(e)
	case trees.TriePrune:
		r.prune(e)
	case trees.TrieInserted:
		r.result(TrieResultInsert, fmt.Sprintf("%s: слово «%s» добавлено в Trie.", r.subject, r.word))
	case trees.TrieAlreadyExisted:
		r.result(TrieResultAlreadyExisted, fmt.Sprintf(
			"%s: слово «%s» уже хранится в Trie — повторная вставка ничего не меняет: дерево и число слов прежние. Повторная вставка — допустимый шаг сценария.",
			r.subject, r.word))
	case trees.TrieFound:
		r.result(TrieResultFound, fmt.Sprintf(
			"%s: слово «%s» найдено — путь по всем рунам существует, узел отмечен концом слова.", r.subject, r.word))
	case trees.TrieNotFound:
		r.result(TrieResultNotFound, fmt.Sprintf(
			"%s: слова «%s» в Trie нет — промах, штатный исход поиска.", r.subject, r.word))
	case trees.TriePrefixFound:
		r.result(TrieResultPrefixFound, fmt.Sprintf(
			"%s: путь по всем рунам префикса существует.", r.subject))
	case trees.TriePrefixNotFound:
		r.result(TrieResultPrefixNotFound, fmt.Sprintf(
			"%s: пути по рунам префикса в Trie нет — промах, штатный исход.", r.subject))
	case trees.TrieDeleted:
		r.result(TrieResultDeleted, fmt.Sprintf(
			"%s: слово «%s» удалено из Trie — отметка конца слова снята, дерево обновлено.",
			r.subject, r.word))
	case trees.TrieNotDeleted:
		r.result(TrieResultNotDeleted, fmt.Sprintf(
			"%s: слова «%s» в Trie нет (путь отсутствует или узел не отмечен концом слова) — удалять нечего, дерево не изменилось. Промах — штатный исход удаления.",
			r.subject, r.word))
	}
}

// visit — шаг rune-обращения: спуск по существующему/создаваемому ребру или
// проверка отсутствующего ребра (узел не создаётся, активный путь остаётся
// у родителя).
func (r *trieFrames) visit(e trees.TrieVisit) {
	check := &TrieEdgeCheck{Rune: string(e.Rune), State: string(e.Edge)}
	var explanation string

	switch e.Edge {
	case trees.EdgeReused, trees.EdgeCreated:
		if e.Edge == trees.EdgeCreated {
			r.cur.children[e.Rune] = &trieMirrorNode{children: map[rune]*trieMirrorNode{}}
		}
		r.cur = r.cur.children[e.Rune]
		r.path = append(r.path, e.Rune)
		id := trieNodeID(r.path)
		r.activePath = append(r.activePath, id)
		if e.EndFlag != nil {
			r.cur.isEnd = *e.EndFlag
		}
		explanation = r.explainVisit(e)
	default: // trees.EdgeMissing: отсутствующее ребро — попытка без перехода
		explanation = r.explainVisit(e)
	}
	r.frame(check, nil, explanation)
}

// explainVisit — пояснение rune-обращения по состоянию ребра и, когда флаг
// конца слова меняется этим шагом, — по итоговой отметке.
func (r *trieFrames) explainVisit(e trees.TrieVisit) string {
	var text string
	switch e.Edge {
	case trees.EdgeReused:
		text = fmt.Sprintf("%s: руна «%s» уже есть в дереве — спускаемся по существующему ребру.",
			r.subject, string(e.Rune))
	case trees.EdgeCreated:
		text = fmt.Sprintf("%s: руны «%s» в дереве нет — создаём новое ребро и узел.",
			r.subject, string(e.Rune))
	default: // trees.EdgeMissing
		text = fmt.Sprintf("%s: руны «%s» в дереве нет — спуск невозможен, дальше пути нет.",
			r.subject, string(e.Rune))
	}
	switch {
	case e.EndFlag != nil && *e.EndFlag:
		text += fmt.Sprintf(" Узел отмечен концом слова: слово «%s» сохранено.", r.word)
	case e.EndFlag != nil:
		text += fmt.Sprintf(" Отметка конца слова снята: слово «%s» больше не хранится.", r.word)
	}
	return text
}

// prune — узел срезан при удалении слова. Событие несёт метку ребра и
// глубину срезанного узла: родитель — узел на пути удаляемого слова с
// глубиной Depth−1. Кадр показывает уже срезанный узел: зеркало применяется
// в момент события, активного пути у шага pruning нет.
func (r *trieFrames) prune(e trees.TriePrune) {
	parent := r.nodeAt(r.path[:e.Depth-1])
	delete(parent.children, e.Rune)
	r.activePath = []string{}
	r.frame(nil, nil, fmt.Sprintf(
		"%s: срезаем узел по ребру «%s» — он больше не ведёт ни к одному концу слова.",
		r.subject, string(e.Rune)))
}

// nodeAt — узел зеркала по пути рун (корень — пустой путь).
func (r *trieFrames) nodeAt(path []rune) *trieMirrorNode {
	node := r.mirror
	for _, ru := range path {
		node = node.children[ru]
	}
	return node
}

// result — кадр результата операции: единственный кадр команды с заполненным
// result; активного пути у итогового кадра нет.
func (r *trieFrames) result(res TrieResult, explanation string) {
	r.activePath = []string{}
	r.frame(nil, &res, explanation)
}

// frame — собирает самодостаточный кадр (ADR-0002): полный снимок зеркала +
// активный путь + edgeCheck/result + пояснение. Шаги нумеруются с 1:
// стартовый кадр, затем по кадру на каждое событие.
func (r *trieFrames) frame(edgeCheck *TrieEdgeCheck, result *TrieResult, explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindTrie,
		Explanation: explanation,
		Data: TrieSnapshot{
			Nodes:      trieSnapshotNodes(r.mirror),
			RootID:     trieRootID,
			ActivePath: append([]string{}, r.activePath...), // не nil: пустой путь — []
			EdgeCheck:  edgeCheck,
			Result:     result,
			Operation:  r.operation,
		},
	})
}

// trieSnapshotNodes — полный снимок топологии зеркала: корень первым, дальше
// в глубину по рунам в порядке возрастания кода (детерминированный порядок).
func trieSnapshotNodes(root *trieMirrorNode) []TrieNodeData {
	var nodes []TrieNodeData
	collectTrieNodes(root, nil, &nodes)
	return nodes
}

func collectTrieNodes(node *trieMirrorNode, path []rune, nodes *[]TrieNodeData) {
	sorted := make([]rune, 0, len(node.children))
	for ru := range node.children {
		sorted = append(sorted, ru)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	data := TrieNodeData{
		ID:       trieNodeID(path),
		IsEnd:    node.isEnd,
		Children: []string{}, // не nil: у листа — []
	}
	for _, ru := range sorted {
		data.Children = append(data.Children, trieNodeID(append(append([]rune{}, path...), ru)))
	}
	*nodes = append(*nodes, data)
	for _, ru := range sorted {
		collectTrieNodes(node.children[ru], append(append([]rune{}, path...), ru), nodes)
	}
}

// trieNodeID — frame-local идентификатор узла Trie (GLOSSARY: Trie Node ID):
// «root» для корня, далее по сегменту на руну — «U+» и шестнадцатеричный код
// руны в верхнем регистре минимальной ширины 4 (длиннее при необходимости:
// символы вне BMP — один сегмент по полному коду, не surrogate-половина).
func trieNodeID(path []rune) string {
	var b strings.Builder
	b.WriteString(trieRootID)
	for _, ru := range path {
		fmt.Fprintf(&b, "/U+%04X", ru)
	}
	return b.String()
}
