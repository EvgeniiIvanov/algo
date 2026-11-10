// Тесты шва 2 для сценария BST (см. docs/SPEC.md): вход → SSE-поток →
// последовательность кадров kind="tree". Покрываются успешные и неуспешные
// операции, все случаи удаления (лист, один потомок, два потомка через
// inorder-successor, корень), финальная структура и отказ от некорректного
// сценария/лимитов до SSE.
package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// doRunBST — запрос Run сценария BST; пустой script — параметр не задан.
func doRunBST(t *testing.T, script string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/bst", nil)
	q := req.URL.Query()
	if script != "" {
		q.Set("script", script)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunBSTSSE — успешный Run: базовые проверки SSE-ответа + кадры.
func doRunBSTSSE(t *testing.T, script string) []frame.Frame {
	t.Helper()

	resp := doRunBST(t, script)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// wantBSTNode — ожидание топологии кадра по значениям; nil-потомок —
// пустая ветвь.
type wantBSTNode struct {
	value int
	left  *wantBSTNode
	right *wantBSTNode
}

// assertBSTFrames — общие проверки последовательности кадров: шаги с 1 по
// порядку, kind="tree", непустые пояснения, форма снимка TreeSnapshot
// (order, пустые visited/queue — совместимость с обходами), активный узел
// и идентификаторы среди живых узлов снимка.
func assertBSTFrames(t *testing.T, frames []frame.Frame) {
	t.Helper()

	for i, f := range frames {
		if f.Step != i+1 {
			t.Fatalf("кадр %d: step = %d, ожидается %d", i, f.Step, i+1)
		}
		if f.Kind != frame.KindTree {
			t.Fatalf("кадр %d: kind = %q, ожидается %q", i, f.Kind, frame.KindTree)
		}
		if f.Explanation == "" {
			t.Fatalf("кадр %d: пустое пояснение", i)
		}
		data, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок — map
		if !ok {
			t.Fatalf("кадр %d: data не является снимком: %#v", i, f.Data)
		}
		if order, ok := data["order"].(string); !ok || order != "bst" {
			t.Fatalf("кадр %d: order = %v, ожидается %q", i, data["order"], "bst")
		}
		for _, empty := range []string{"visited", "queue"} {
			list, ok := data[empty].([]any)
			if !ok {
				t.Fatalf("кадр %d: поле %q не массив: %#v", i, empty, data[empty])
			}
			if len(list) != 0 {
				t.Fatalf("кадр %d: поле %q = %v, у BST всегда пустое", i, empty, list)
			}
		}
		// Идентификаторы снимка ссылаются на живые узлы (проверки зеркальны
		// валидации фронта): activeId/rootId — узел или "".
		ids := bstNodeIDs(t, data)
		for _, field := range []string{"activeId", "rootId"} {
			id, _ := data[field].(string)
			if id != "" && !ids[id] {
				t.Fatalf("кадр %d: поле %q = %q не является идентификатором узла снимка", i, field, id)
			}
		}
		for id, node := range bstNodesByID(t, data) {
			for _, side := range []string{"leftId", "rightId"} {
				link, _ := node[side].(string)
				if link != "" && !ids[link] {
					t.Fatalf("кадр %d: узел %q ссылается %q на отсутствующего потомка", i, id, side)
				}
			}
		}
	}
}

// bstNodesByID — узлы снимка кадра по идентификаторам (после JSON-кругосветки).
func bstNodesByID(t *testing.T, data map[string]any) map[string]map[string]any {
	t.Helper()

	raw, ok := data["nodes"].([]any)
	if !ok {
		t.Fatalf("поле nodes не массив: %#v", data["nodes"])
	}
	byID := make(map[string]map[string]any, len(raw))
	for _, e := range raw {
		node, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("узел снимка не объект: %#v", e)
		}
		id, _ := node["id"].(string)
		byID[id] = node
	}
	return byID
}

// bstNodeIDs — множество идентификаторов снимка кадра.
func bstNodeIDs(t *testing.T, data map[string]any) map[string]bool {
	t.Helper()

	ids := make(map[string]bool)
	for id := range bstNodesByID(t, data) {
		ids[id] = true
	}
	return ids
}

// bstNodeValue — значение узла снимка по идентификатору.
func bstNodeValue(t *testing.T, data map[string]any, id string) int {
	t.Helper()

	node, ok := bstNodesByID(t, data)[id]
	if !ok {
		t.Fatalf("узел %q отсутствует в снимке", id)
	}
	v, ok := node["value"].(float64)
	if !ok {
		t.Fatalf("узел %q: value не число: %#v", id, node["value"])
	}
	return int(v)
}

// bstLink — идентификатор потомка узла по стороне.
func bstLink(node map[string]any, side string) string {
	link, _ := node[side].(string)
	return link
}

// assertBSTTopology — сверяет снимок кадра с ожидаемой структурой дерева.
func assertBSTTopology(t *testing.T, f frame.Frame, want *wantBSTNode) {
	t.Helper()

	data, ok := f.Data.(map[string]any)
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	rootID, _ := data["rootId"].(string)
	byID := bstNodesByID(t, data)
	if want == nil {
		if rootID != "" {
			t.Fatalf("кадр шага %d: rootId = %q, дерево пустое", f.Step, rootID)
		}
		if len(byID) != 0 {
			t.Fatalf("кадр шага %d: узлов в снимке %d, ожидается 0 (пустое дерево)", f.Step, len(byID))
		}
		return
	}
	if rootID == "" {
		t.Fatalf("кадр шага %d: rootId пуст, ожидается корень со значением %d", f.Step, want.value)
	}
	assertBSTTopologyFrom(t, f, data, rootID, want)
}

// assertBSTTopologyFrom — рекурсивная часть assertBSTTopology: узел id и
// его поддеревья совпадают с ожиданием.
func assertBSTTopologyFrom(t *testing.T, f frame.Frame, data map[string]any, id string, want *wantBSTNode) {
	t.Helper()

	node, ok := bstNodesByID(t, data)[id]
	if !ok {
		t.Fatalf("кадр шага %d: узел %q отсутствует в снимке", f.Step, id)
	}
	if got := bstNodeValue(t, data, id); got != want.value {
		t.Fatalf("кадр шага %d: узел %q имеет значение %d, ожидается %d", f.Step, id, got, want.value)
	}
	for side, wantChild := range map[string]*wantBSTNode{"leftId": want.left, "rightId": want.right} {
		id2 := bstLink(node, side)
		if wantChild == nil {
			if id2 != "" {
				t.Fatalf("кадр шага %d: узел %q имеет %s = %q, ожидается пустая ветвь", f.Step, id, side, id2)
			}
			continue
		}
		if id2 == "" {
			t.Fatalf("кадр шага %d: у узла %q пустая ветвь %s, ожидается потомок со значением %d", f.Step, id, side, wantChild.value)
		}
		assertBSTTopologyFrom(t, f, data, id2, wantChild)
	}
}

// bstActive — идентификатор активного узла кадра.
func bstActive(t *testing.T, f frame.Frame) string {
	t.Helper()

	data, ok := f.Data.(map[string]any)
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	active, _ := data["activeId"].(string)
	return active
}

// bstFramesCount — ожидаемое число кадров с понятным сообщением.
func bstFramesCount(t *testing.T, frames []frame.Frame, want int) {
	t.Helper()

	if len(frames) != want {
		t.Fatalf("кадров = %d, ожидается %d", len(frames), want)
	}
}

// bstExplanationHas — у последнего кадра проверяется содержимое пояснения.
func bstExplanationHas(t *testing.T, f frame.Frame, substrings ...string) {
	t.Helper()

	for _, s := range substrings {
		if !strings.Contains(f.Explanation, s) {
			t.Fatalf("пояснение кадра шага %d = %q, ожидается упоминание %q", f.Step, f.Explanation, s)
		}
	}
}

func TestBSTRunInsertScenarioSSE(t *testing.T) {
	frames := doRunBSTSSE(t, "insert 5\ninsert 3\ninsert 7")
	assertBSTFrames(t, frames)
	bstFramesCount(t, frames, 6)

	// Исходный кадр: пустое дерево, активного узла нет.
	assertBSTTopology(t, frames[0], nil)
	if active := bstActive(t, frames[0]); active != "" {
		t.Fatalf("исходный кадр: activeId = %q, ожидается пустая строка", active)
	}

	// Вставка в пустое дерево — один кадр: узел 5 становится корнем (BSTInserted
	// сразу, без сравнений).
	assertBSTTopology(t, frames[1], &wantBSTNode{value: 5})
	if active := bstActive(t, frames[1]); active != "root" {
		t.Fatalf("вставка 5: activeId = %q, ожидается \"root\"", active)
	}
	bstExplanationHas(t, frames[1], "Вставка ключа 5", "становится корнем")

	// Вставка 3: сравнение с корнем влево, затем новый узел на пути «L».
	bstExplanationHas(t, frames[2], "сравниваем с узлом 5", "спускаемся влево")
	if active := bstActive(t, frames[2]); active != "root" {
		t.Fatalf("сравнение вставки 3: activeId = %q, ожидается \"root\"", active)
	}
	bstExplanationHas(t, frames[3], "новый узел 3", "путь «L»")
	if active := bstActive(t, frames[3]); active != "L" {
		t.Fatalf("вставка 3: activeId = %q, ожидается \"L\"", active)
	}

	// Вставка 7 — зеркально вправо; финальное дерево 5(3, 7).
	bstExplanationHas(t, frames[4], "сравниваем с узлом 5", "спускаемся вправо")
	assertBSTTopology(t, frames[5], &wantBSTNode{value: 5,
		left:  &wantBSTNode{value: 3},
		right: &wantBSTNode{value: 7}})
}

func TestBSTRunSearchHitAndMissSSE(t *testing.T) {
	// Сценарий: вставки строят дерево 16(8, 24), затем успех и промах поиска.
	frames := doRunBSTSSE(t, "insert 16\ninsert 8\ninsert 24\nsearch 24\nsearch 9")
	assertBSTFrames(t, frames)
	// 1 старт + 1 вставка 16 + 2 + 2 + 3 (поиск 24: два сравнения и
	// находка) + 3 (поиск 9: два сравнения и пустое место).
	bstFramesCount(t, frames, 12)

	// Успешный поиск 24: сравнение с корнем вправо, равенство в узле 24
	// и «найден» на пути «R».
	bstExplanationHas(t, frames[6], "спускаемся вправо")
	bstExplanationHas(t, frames[7], "сравниваем с узлом 24", "ключ равен значению узла")
	bstExplanationHas(t, frames[8], "ключ 24 найден", "путь «R»")
	if active := bstActive(t, frames[8]); active != "R" {
		t.Fatalf("найденный узел 24: activeId = %q, ожидается \"R\"", active)
	}

	// Промах поиска 9: два сравнения (влево от корня, вправо от узла 8) и
	// пустое место — подсветки нет, промах назван штатным исходом.
	bstExplanationHas(t, frames[10], "сравниваем с узлом 8", "спускаемся вправо")
	bstExplanationHas(t, frames[11], "пустого места", "ключа 9 в дереве нет", "штатный исход")
	if active := bstActive(t, frames[11]); active != "" {
		t.Fatalf("промах поиска 9: activeId = %q, ожидается пустая строка", active)
	}
}

func TestBSTRunDuplicateInsertSSE(t *testing.T) {
	frames := doRunBSTSSE(t, "insert 5\ninsert 3\ninsert 5")
	assertBSTFrames(t, frames)
	// 1 старт + 1 + 2 + сравнение с равенством и дубликат (2).
	bstFramesCount(t, frames, 6)

	bstExplanationHas(t, frames[5], "уже держит ключ 5", "Дерево не меняется")
	// Дерево не изменилось: 2 узла.
	assertBSTTopology(t, frames[5], &wantBSTNode{value: 5, left: &wantBSTNode{value: 3}})
}

// TestBSTRunDeleteCasesSSE — все случаи удаления и промах: финальная
// структура после каждого сценария и пояснения значимых шагов.
func TestBSTRunDeleteCasesSSE(t *testing.T) {
	leaf5 := &wantBSTNode{value: 5}
	cases := []struct {
		name      string
		script    string
		wantFinal *wantBSTNode
		wantLast  string // обязательная подстрока пояснения последнего кадра
	}{
		{
			name:      "лист",
			script:    "insert 5\ninsert 3\ninsert 7\ndelete 3",
			wantFinal: &wantBSTNode{value: 5, right: &wantBSTNode{value: 7}},
			wantLast:  "удалён",
		},
		{
			name:      "один потомок — левый",
			script:    "insert 5\ninsert 3\ninsert 1\ndelete 3",
			wantFinal: &wantBSTNode{value: 5, left: &wantBSTNode{value: 1}},
			wantLast:  "удалён",
		},
		{
			name:      "один потомок — правый",
			script:    "insert 5\ninsert 7\ninsert 9\ndelete 7",
			wantFinal: &wantBSTNode{value: 5, right: &wantBSTNode{value: 9}},
			wantLast:  "удалён",
		},
		{
			// Преемник корня-листа: дерево 10(5, 20(15(R17), 30));
			// ключ 10 исчезает, корень теперь 15, у 20 левые — 17.
			name:   "два потомка — преемник не лист",
			script: "insert 10\ninsert 5\ninsert 20\ninsert 15\ninsert 30\ninsert 17\ndelete 10",
			wantFinal: &wantBSTNode{value: 15, left: &wantBSTNode{value: 5},
				right: &wantBSTNode{value: 20, left: &wantBSTNode{value: 17}, right: &wantBSTNode{value: 30}}},
			wantLast: "удалён",
		},
		{
			name:      "корень — единственный узел",
			script:    "insert 5\ndelete 5",
			wantFinal: nil,
			wantLast:  "удалён",
		},
		{
			name:      "корень с одним потомком",
			script:    "insert 5\ninsert 9\ndelete 5",
			wantFinal: &wantBSTNode{value: 9},
			wantLast:  "удалён",
		},
		{
			name:   "корень с двумя потомками",
			script: "insert 5\ninsert 3\ninsert 8\ninsert 7\ninsert 9\ndelete 5",
			wantFinal: &wantBSTNode{value: 7, left: &wantBSTNode{value: 3},
				right: &wantBSTNode{value: 8, right: &wantBSTNode{value: 9}}},
			wantLast: "удалён",
		},
		{
			// Преемник — сам правый потомок: у правого поддерева нет левой
			// ветви, минимум стоит в её корне (путь «R»).
			name:      "два потомка — преемник сам правый потомок",
			script:    "insert 5\ninsert 3\ninsert 7\ndelete 5",
			wantFinal: &wantBSTNode{value: 7, left: &wantBSTNode{value: 3}},
			wantLast:  "удалён",
		},
		{
			name:      "промах удаления",
			script:    "insert 5\ndelete 4",
			wantFinal: leaf5,
			wantLast:  "штатный исход",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frames := doRunBSTSSE(t, tc.script)
			assertBSTFrames(t, frames)
			assertBSTTopology(t, frames[len(frames)-1], tc.wantFinal)
			bstExplanationHas(t, frames[len(frames)-1], tc.wantLast)
		})
	}
}

// TestBSTRunTwoChildrenDeleteStepsSSE — значимые промежуточные шаги
// удаления узла с двумя детьми: поиск минимума правого поддерева,
// выбор преемника и перенос значения, затем вырезание преемника.
func TestBSTRunTwoChildrenDeleteStepsSSE(t *testing.T) {
	frames := doRunBSTSSE(t, "insert 10\ninsert 5\ninsert 20\ninsert 15\ninsert 30\ninsert 17\ndelete 10")
	assertBSTFrames(t, frames)

	// Шаги событий отличаем по имени шага снимка: поиск преемника —
	// MinStep, выбор преемника — Successor (объяснения пересекаются
	// словами, поле operation — точный маркер).
	stepName := func(f frame.Frame) string {
		data, ok := f.Data.(map[string]any)
		if !ok {
			t.Fatalf("кадр шага %d: data не снимок: %#v", f.Step, f.Data)
		}
		op, _ := data["operation"].(string)
		return op
	}
	var minSteps, successorFrame []frame.Frame
	for _, f := range frames {
		if strings.HasPrefix(stepName(f), "поиск преемника") {
			minSteps = append(minSteps, f)
		}
		if strings.HasPrefix(stepName(f), "преемник «") {
			successorFrame = append(successorFrame, f)
		}
	}
	if len(minSteps) != 2 {
		t.Fatalf("шагов поиска минимума = %d, ожидается 2 (узлы 20 и 15)", len(minSteps))
	}
	bstExplanationHas(t, minSteps[0], "узел 20 (путь «R»)")
	bstExplanationHas(t, minSteps[1], "узел 15 (путь «RL»)")
	if active := bstActive(t, minSteps[1]); active != "RL" {
		t.Fatalf("преемник на пути «RL»: activeId = %q", active)
	}

	// Выбор преемника: значение 15 переходит узлу, которое держало 10;
	// в самом кадре снимок уже показывает дерево после переноса.
	if len(successorFrame) != 1 {
		t.Fatalf("кадров выбора преемника = %d, ожидается 1", len(successorFrame))
	}
	bstExplanationHas(t, successorFrame[0], "inorder-successor", "узел 15 (путь «RL»)", "Значение 15")
	data, ok := successorFrame[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("кадр выбора преемника: data не снимок: %#v", successorFrame[0].Data)
	}
	if got := bstNodeValue(t, data, "root"); got != 15 {
		t.Fatalf("в кадре выбора преемника корень = %d, ожидается 15 (значение перенесено)", got)
	}

	// Финальная структура: 10 удалён, преемник вырезан.
	assertBSTTopology(t, frames[len(frames)-1], &wantBSTNode{value: 15, left: &wantBSTNode{value: 5},
		right: &wantBSTNode{value: 20, left: &wantBSTNode{value: 17}, right: &wantBSTNode{value: 30}}})
}

// TestBSTRunMixedScenarioSSE — сценарий со вперемешку успешными/неуспешными
// операциями: состояние дерева воспроизводится через финальный снимок.
func TestBSTRunMixedScenarioSSE(t *testing.T) {
	script := strings.Join([]string{
		"insert 5", "insert 3", "insert 8", "insert 1", "insert 4", "insert 7", "insert 9",
		"delete 8", "search 4", "delete 5", "insert 6", "search 10",
	}, "\n")
	frames := doRunBSTSSE(t, script)
	assertBSTFrames(t, frames)

	// Поиск 4 — успех; 10 — промах.
	var found, missed bool
	for _, f := range frames {
		if strings.Contains(f.Explanation, "ключ 4 найден") {
			found = true
		}
		if strings.Contains(f.Explanation, "ключа 10 в дереве нет") {
			missed = true
		}
	}
	if !found || !missed {
		t.Fatalf("ожидается кадр успеха поиска 4 и кадра промаха 10: found=%v missed=%v", found, missed)
	}

	// Финал: 5 и 8 удалены, 6 добавлен — дерево 7(3(1, 4(R6)), 9).
	assertBSTTopology(t, frames[len(frames)-1], &wantBSTNode{value: 7,
		left: &wantBSTNode{value: 3, left: &wantBSTNode{value: 1},
			right: &wantBSTNode{value: 4, right: &wantBSTNode{value: 6}}},
		right: &wantBSTNode{value: 9}})
}

func TestBSTRunBlankLinesIgnoredSSE(t *testing.T) {
	frames := doRunBSTSSE(t, "\n\ninsert 5\n \ndelete 5\n\n")
	assertBSTFrames(t, frames)
	// 1 старт + вставка (узел становится корнем) + сравнение + удаление.
	bstFramesCount(t, frames, 4)
	assertBSTTopology(t, frames[len(frames)-1], nil)
}

// TestBSTRunSafeRangeBoundarySSE — границы безопасного диапазона
// JavaScript: ±(2^53−1) допустимы (2^53 отвергается валидацией — см. ниже).
func TestBSTRunSafeRangeBoundarySSE(t *testing.T) {
	frames := doRunBSTSSE(t, "insert 9007199254740991\nsearch -9007199254740991")
	assertBSTFrames(t, frames)
	bstFramesCount(t, frames, 4) // старт + вставка + сравнение + промах
	bstExplanationHas(t, frames[3], "ключа -9007199254740991 в дереве нет")

	data, ok := frames[1].Data.(map[string]any)
	if !ok {
		t.Fatalf("кадр вставки: data не снимок: %#v", frames[1].Data)
	}
	if got := bstNodeValue(t, data, "root"); got != 9007199254740991 {
		t.Fatalf("корень = %d, ожидается 9007199254740991 (верхняя граница диапазона)", got)
	}
}

// TestBSTRunNodeLimitBoundarySSE — лимит живых узлов применяется ровно:
// 100 узлов допустимы, 101-й отвергается до SSE.
func TestBSTRunNodeLimitBoundarySSE(t *testing.T) {
	t.Run("100 узлов принимаются", func(t *testing.T) {
		script := bstInsertsScript(t, 100)
		frames := doRunBSTSSE(t, script)
		assertBSTFrames(t, frames)
		// Кадр каждой вставки: первая — узел-корень, далее по кадру на
		// каждое сравнение; точное число событий проверяют отдельные
		// сценарии, здесь важен сам факт прогона на границе лимита.
		bstExplanationHas(t, frames[1], "становится корнем")
		last := frames[len(frames)-1]
		data, ok := last.Data.(map[string]any)
		if !ok {
			t.Fatalf("финальный кадр: data не снимок: %#v", last.Data)
		}
		if got := len(bstNodesByID(t, data)); got != 100 {
			t.Fatalf("узлов в финальном снимке = %d, ожидается 100", got)
		}
	})

	t.Run("101 узел отвергается до SSE", func(t *testing.T) {
		script := bstInsertsScript(t, 101)
		resp := doRunBST(t, script)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("статус = %d, ожидается 400: %s", resp.Code, resp.Body.String())
		}
		if strings.HasPrefix(resp.Header().Get("Content-Type"), "text/event-stream") {
			t.Fatalf("отказ должен случиться до SSE, получен поток: %q", resp.Body.String())
		}
		body := resp.Body.String()
		for _, want := range []string{"лимит живых узлов", "№101"} {
			if !strings.Contains(body, want) {
				t.Fatalf("тело ошибки = %s, ожидается упоминание %q", body, want)
			}
		}
	})
}

// bstInsertsScript — сценарий из n последовательных вставок возрастающих
// ключей (перекошенное вправо дерево ровно из n живых узлов).
func bstInsertsScript(t *testing.T, n int) string {
	t.Helper()

	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "insert %d\n", i)
	}
	return b.String()
}

// TestBSTRunValidation — синтаксис, диапазон значений и лимит команд
// отвергаются до SSE, без частично начатого потока.
func TestBSTRunValidation(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{"нет параметра script", "", "параметр script обязателен"},
		{"только пустые строки", "\n \n\t\n", "сценарий пуст"},
		{"неизвестная команда", "upsert 5", "неизвестная команда"},
		{"команда без числа", "insert", "не хватает целого числа"},
		{"лишние аргументы", "insert 5 6", "лишние аргументы"},
		{"дробное число", "insert 1.5", "не удалось разобрать"},
		{"экспонента", "insert 1e3", "не удалось разобрать"},
		{"мусор вместо числа", "delete семь", "не удалось разобрать"},
		{"значение больше int64", "insert 9223372036854775808", "за пределами целого"},
		{"значение меньше -int64", "delete -9223372036854775809", "за пределами целого"},
		{"выше безопасного диапазона", "insert 9007199254740993", "вне безопасного диапазона"},
		{"ниже безопасного диапазона", "search -9007199254740993", "вне безопасного диапазона"},
		{"ошибка указывает строку", "insert 5\nupsert 7\ndelete 1", "на строке 2"},
		{"лимит команд", strings.Repeat("insert 1\n", 501), "не более 500 команд"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunBST(t, tt.script)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("статус = %d, ожидается 400: %s", resp.Code, resp.Body.String())
			}
			// Content-Type по префиксу: операции до SSE не должны начинать
			// поток (см. одноимённую проверку тестов обходов дерева).
			if strings.HasPrefix(resp.Header().Get("Content-Type"), "text/event-stream") {
				t.Fatalf("отказ должен случиться до SSE, получен поток: %q", resp.Body.String())
			}
			if got := resp.Body.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("тело ошибки = %s, ожидается упоминание %q", got, tt.want)
			}
		})
	}
}
