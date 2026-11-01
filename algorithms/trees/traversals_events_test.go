// Тесты событий обходов (ADR-0001): с Emitter последовательность DFSVisit
// восстанавливает порядок посещений, включая узлы — события различают
// повторяющиеся значения путём от корня.
package trees_test

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms"
	"github.com/EvgeniiIvanov/algo/algorithms/trees"
)

// spyTreeEmitter — тестовый Emitter: собирает события обхода в слайс
// для проверки последовательности шагов.
type spyTreeEmitter []trees.DFSEvent

func (s *spyTreeEmitter) Emit(event trees.DFSEvent) { *s = append(*s, event) }

// spyLevelEmitter — тестовый Emitter обхода по уровням: собирает события
// LevelOrderVisit (с состоянием очереди) в слайс.
type spyLevelEmitter []trees.LevelOrderEvent

func (s *spyLevelEmitter) Emit(event trees.LevelOrderEvent) { *s = append(*s, event) }

// Известный порядок событий симметричного дерева (см. traversals_test.go):
// один DFSVisit на каждый узел, путь различает одинаковые значения.
func TestInOrderEvents(t *testing.T) {
	var em spyTreeEmitter
	got := trees.InOrderWithEmitter(symTree(), &em)

	wantResult := []int{4, 2, 7, 5, 1, 3, 6}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("InOrderWithEmitter вернул %v, ожидается %v", got, wantResult)
	}
	want := []trees.DFSEvent{
		trees.DFSVisit{Path: "LL", Value: 4},
		trees.DFSVisit{Path: "L", Value: 2},
		trees.DFSVisit{Path: "LRL", Value: 7},
		trees.DFSVisit{Path: "LR", Value: 5},
		trees.DFSVisit{Path: "", Value: 1},
		trees.DFSVisit{Path: "R", Value: 3},
		trees.DFSVisit{Path: "RR", Value: 6},
	}
	if !reflect.DeepEqual([]trees.DFSEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.DFSEvent(em), want)
	}
}

func TestPreOrderEvents(t *testing.T) {
	var em spyTreeEmitter
	got := trees.PreOrderWithEmitter(symTree(), &em)

	wantResult := []int{1, 2, 4, 5, 7, 3, 6}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("PreOrderWithEmitter вернул %v, ожидается %v", got, wantResult)
	}
	want := []trees.DFSEvent{
		trees.DFSVisit{Path: "", Value: 1}, // корень
		trees.DFSVisit{Path: "L", Value: 2},
		trees.DFSVisit{Path: "LL", Value: 4},
		trees.DFSVisit{Path: "LR", Value: 5},
		trees.DFSVisit{Path: "LRL", Value: 7},
		trees.DFSVisit{Path: "R", Value: 3},
		trees.DFSVisit{Path: "RR", Value: 6},
	}
	if !reflect.DeepEqual([]trees.DFSEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.DFSEvent(em), want)
	}
}

func TestPostOrderEvents(t *testing.T) {
	var em spyTreeEmitter
	got := trees.PostOrderWithEmitter(symTree(), &em)

	wantResult := []int{4, 7, 5, 2, 6, 3, 1}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("PostOrderWithEmitter вернул %v, ожидается %v", got, wantResult)
	}
	want := []trees.DFSEvent{
		trees.DFSVisit{Path: "LL", Value: 4},
		trees.DFSVisit{Path: "LRL", Value: 7},
		trees.DFSVisit{Path: "LR", Value: 5},
		trees.DFSVisit{Path: "L", Value: 2},
		trees.DFSVisit{Path: "RR", Value: 6},
		trees.DFSVisit{Path: "R", Value: 3},
		trees.DFSVisit{Path: "", Value: 1}, // корень
	}
	if !reflect.DeepEqual([]trees.DFSEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.DFSEvent(em), want)
	}
}

// Повторяющиеся значения: события идут по узлам (пути LL.. "LR"?), не по
// значениям — последовательность путей на том же дереве узнаваема.
func TestEventsDistinguishRepeatedValues(t *testing.T) {
	root := trees.NewNode(2)
	l := trees.NewNode(7)
	r := trees.NewNode(2)
	ll := trees.NewNode(7)
	rr := trees.NewNode(2)
	root.Left, root.Right = l, r
	l.Left = ll
	r.Right = rr

	var em spyTreeEmitter
	got := trees.InOrderWithEmitter(root, &em)
	if !reflect.DeepEqual(got, []int{7, 7, 2, 2, 2}) {
		t.Fatalf("InOrderWithEmitter вернул %v", got)
	}
	want := []trees.DFSEvent{
		trees.DFSVisit{Path: "LL", Value: 7},
		trees.DFSVisit{Path: "L", Value: 7},
		trees.DFSVisit{Path: "", Value: 2}, // корень, значение 2
		trees.DFSVisit{Path: "R", Value: 2},
		trees.DFSVisit{Path: "RR", Value: 2},
	}
	if !reflect.DeepEqual([]trees.DFSEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.DFSEvent(em), want)
	}
}

// Пустое дерево: событий нет, результат пустой; путь корня даёт его ID.
func TestEventsEmptyTreeAndNodeID(t *testing.T) {
	if got := trees.InOrderWithEmitter(nil, &spyTreeEmitter{}); len(got) != 0 {
		t.Fatalf("пустое дерево: результат %v, ожидается пустой", got)
	}
	var em spyTreeEmitter
	_ = trees.PreOrderWithEmitter(trees.NewNode(5), &em)
	if len(em) != 1 || em[0] != (trees.DFSVisit{Path: "", Value: 5}) {
		t.Fatalf("одноузловое дерево: события %#v", []trees.DFSEvent(em))
	}
	if trees.NodeID("") != "root" || trees.NodeID("LR") != "LR" {
		t.Fatalf("NodeID: %q и %q", trees.NodeID(""), trees.NodeID("LR"))
	}
	// Заглушка для алгоритмов.Emitter — проверяем статически, что тип
	// событий удовлетворяет общему контракту (не даём лишних импортов умереть).
	var _ algorithms.Emitter[trees.DFSEvent] = (*spyTreeEmitter)(nil)
}

// Level-order на симметричном дереве: порядок обхода — уровни сверху вниз
// слева направо, а каждое событие несёт состояние FIFO-очереди после шага
// (посещённый уже снят, дети уже добавлены).
func TestLevelOrderEvents(t *testing.T) {
	var em spyLevelEmitter
	got := trees.LevelOrderWithEmitter(symTree(), &em)

	wantResult := []int{1, 2, 3, 4, 5, 6, 7}
	if !reflect.DeepEqual(got, wantResult) {
		t.Fatalf("LevelOrderWithEmitter вернул %v, ожидается %v", got, wantResult)
	}
	want := []trees.LevelOrderEvent{
		trees.LevelOrderVisit{Path: "", Value: 1, Queue: []string{"L", "R"}},
		trees.LevelOrderVisit{Path: "L", Value: 2, Queue: []string{"R", "LL", "LR"}},
		trees.LevelOrderVisit{Path: "R", Value: 3, Queue: []string{"LL", "LR", "RR"}},
		trees.LevelOrderVisit{Path: "LL", Value: 4, Queue: []string{"LR", "RR"}},
		trees.LevelOrderVisit{Path: "LR", Value: 5, Queue: []string{"RR", "LRL"}},
		trees.LevelOrderVisit{Path: "RR", Value: 6, Queue: []string{"LRL"}},
		trees.LevelOrderVisit{Path: "LRL", Value: 7, Queue: []string{}},
	}
	if !reflect.DeepEqual([]trees.LevelOrderEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.LevelOrderEvent(em), want)
	}
	// Заглушка для контракта эмиттера — как у DFS-обходов.
	var _ algorithms.Emitter[trees.LevelOrderEvent] = (*spyLevelEmitter)(nil)
}

// Повторяющиеся значения у level-order: события идут по узлам (путям),
// в очереди различимы идентификаторами, а не значениями.
func TestLevelOrderEventsDistinguishRepeatedValues(t *testing.T) {
	root := repeatedValuesTree()

	var em spyLevelEmitter
	got := trees.LevelOrderWithEmitter(root, &em)
	if !reflect.DeepEqual(got, []int{2, 7, 2, 7, 2}) {
		t.Fatalf("LevelOrderWithEmitter вернул %v", got)
	}
	want := []trees.LevelOrderEvent{
		trees.LevelOrderVisit{Path: "", Value: 2, Queue: []string{"L", "R"}},
		trees.LevelOrderVisit{Path: "L", Value: 7, Queue: []string{"R", "LL"}},
		trees.LevelOrderVisit{Path: "R", Value: 2, Queue: []string{"LL", "RR"}},
		trees.LevelOrderVisit{Path: "LL", Value: 7, Queue: []string{"RR"}},
		trees.LevelOrderVisit{Path: "RR", Value: 2, Queue: []string{}},
	}
	if !reflect.DeepEqual([]trees.LevelOrderEvent(em), want) {
		t.Fatalf("события = %#v, ожидается %#v", []trees.LevelOrderEvent(em), want)
	}
}

// Пустое дерево: событий нет, результат пустой; одноузловое — один визит,
// очередь опустела сразу (корень снят, детей нет).
func TestLevelOrderEventsEmptyAndSingleNode(t *testing.T) {
	if got := trees.LevelOrderWithEmitter(nil, &spyLevelEmitter{}); len(got) != 0 {
		t.Fatalf("пустое дерево: результат %v, ожидается пустой", got)
	}
	var em spyLevelEmitter
	_ = trees.LevelOrderWithEmitter(trees.NewNode(5), &em)
	want := []trees.LevelOrderEvent{
		trees.LevelOrderVisit{Path: "", Value: 5, Queue: []string{}},
	}
	if !reflect.DeepEqual([]trees.LevelOrderEvent(em), want) {
		t.Fatalf("одноузловое дерево: события %#v, ожидается %#v", []trees.LevelOrderEvent(em), want)
	}
}

// Growth-тест через публичный API: полное широкое дерево 31 узел из 5
// уровней переполняет начальную 4-слотовую очередь (росты 4→8 и 8→16 при
// заполнении живыми элементами к границам уровней), вставки и снятия идут
// с заворотом по модулю ёмкости — вплоть до съёма листьев с полностью
// обёрнутого 16-слотового буфера. Проверяются результат, полный
// порядок путей и точные состояния FIFO-очереди на критичных шагах
// роста/заворота; приватный nodeQueue напрямую не трогается.
func TestLevelOrderEventsRingBufferGrowth(t *testing.T) {
	root := perfectTree31()

	// Тихий обход: полнота и порядок уровней сохраняются после роста и
	// заворачивания буфера (значения узлов 1..31 по уровням).
	wantValues := make([]int, 31)
	for v := 1; v <= 31; v++ {
		wantValues[v-1] = v
	}
	if got := trees.LevelOrder(root); !reflect.DeepEqual(got, wantValues) {
		t.Fatalf("trees.LevelOrder вернул %v, ожидается 1..31", got)
	}

	var em spyLevelEmitter
	if got := trees.LevelOrderWithEmitter(root, &em); len(got) != 31 {
		t.Fatalf("LevelOrderWithEmitter вернул %d значений, ожидается 31", len(got))
	}

	// Путь узла v: двоичная запись v без ведущей единицы, бит 0 → L,
	// бит 1 → R (пути идут по уровням слева направо).
	pathOf := func(v int) string {
		path := ""
		for _, digit := range strconv.FormatInt(int64(v), 2)[1:] {
			if digit == '0' {
				path += "L"
			} else {
				path += "R"
			}
		}
		return path
	}
	// visitOf — извлечение проверки из события: тестовый эмиттер хранит
	// содержимое как маркер-интерфейс.
	visitOf := func(i int) trees.LevelOrderVisit {
		visit, ok := em[i].(trees.LevelOrderVisit)
		if !ok {
			t.Fatalf("шаг %d: событие не LevelOrderVisit: %#v", i+1, em[i])
		}
		return visit
	}
	for i := range em {
		if want := pathOf(i + 1); visitOf(i).Path != want {
			t.Fatalf("шаг %d: путь = %q, ожидается %q", i+1, visitOf(i).Path, want)
		}
	}

	// Существенная последовательность очереди: границы начальной ёмкости,
	// каждый рост буфера и порядок живых записей после заворотов.
	wantQueues := []struct {
		step  int // индекс события (0 — первый визит)
		queue []string
	}{
		{0, []string{"L", "R"}},
		{1, []string{"R", "LL", "LR"}},
		// Очередь ровно на начальную ёмкость (4), без роста.
		{2, []string{"LL", "LR", "RL", "RR"}},
		// Первый рост (4→8) при постановке LLR: переполнение + вставка.
		{3, []string{"LR", "RL", "RR", "LLL", "LLR"}},
		// Очередь ровно на ёмкость 8.
		{6, []string{"LLL", "LLR", "LRL", "LRR", "RLL", "RLR", "RRL", "RRR"}},
		// Второй рост (8→16): вставка LLLL с заворотом по модулю 8, затем рост.
		{7, []string{"LLR", "LRL", "LRR", "RLL", "RLR", "RRL", "RRR", "LLLL", "LLLR"}},
		// Очередь ровно на ёмкость 16 (все листья пятого уровня).
		{14, []string{"LLLL", "LLLR", "LLRL", "LLRR", "LRLL", "LRLR", "LRRL", "LRRR",
			"RLLL", "RLLR", "RLRL", "RLRR", "RRLL", "RRLR", "RRRL", "RRRR"}},
		// Снятие листьев идёт с головы по обёрнутому кругу буфера: LLLL
		// снята первой, порядок хвоста сохранён.
		{15, []string{"LLLR", "LLRL", "LLRR", "LRLL", "LRLR", "LRRL", "LRRR", "RLLL",
			"RLLR", "RLRL", "RLRR", "RRLL", "RRLR", "RRRL", "RRRR"}},
		// Итог: очередь опустела ровно в порядке постановки листьев —
		// порядок живых записей сохранён через все росты и завороты.
		{30, []string{}},
	}

	for _, c := range wantQueues {
		if !reflect.DeepEqual(visitOf(c.step).Queue, c.queue) {
			t.Fatalf("шаг %d: очередь = %v, ожидается %v", c.step+1, visitOf(c.step).Queue, c.queue)
		}
	}
}

// perfectTree31 — полное широкое дерево 31 узел из 5 уровней: значение
// узла v = его номер по уровням (родитель v/2, левый 2v, правый 2v+1).
// Местная сборка fixture, не общий helper: нужна только этому тесту.
func perfectTree31() *trees.Node {
	nodes := make([]*trees.Node, 32) // индекс — значение узла
	for v := 1; v <= 31; v++ {
		nodes[v] = trees.NewNode(v)
	}
	for v := 2; v <= 31; v++ {
		if v%2 == 0 {
			nodes[v/2].Left = nodes[v]
		} else {
			nodes[v/2].Right = nodes[v]
		}
	}
	return nodes[1]
}

// Тот же узловой состав дерева с повторяющимися значениями, что и в
// DFS-событийных тестах: порядок относится к узлам, а не к значениям.
func repeatedValuesTree() *trees.Node {
	root := trees.NewNode(2)
	l := trees.NewNode(7)
	r := trees.NewNode(2)
	ll := trees.NewNode(7)
	rr := trees.NewNode(2)
	root.Left, root.Right = l, r
	l.Left = ll
	r.Right = rr
	return root
}

func symTree() *trees.Node {
	n1 := trees.NewNode(1)
	n2 := trees.NewNode(2)
	n3 := trees.NewNode(3)
	n4 := trees.NewNode(4)
	n5 := trees.NewNode(5)
	n6 := trees.NewNode(6)
	n7 := trees.NewNode(7)
	n1.Left, n1.Right = n2, n3
	n2.Left, n2.Right = n4, n5
	n3.Right = n6
	n5.Left = n7
	return n1
}
