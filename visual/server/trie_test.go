// Тесты шва 2 для сценария Trie (см. docs/SPEC.md): вход → SSE-поток →
// последовательность кадров kind="trie". Покрываются сценарий с общим
// префиксом, точный поиск и промах, проверка префикса, удаление с pruning,
// все варианты закрытого множества result, no-op-сценарии (промахи по
// отсутствующему ребру и по существующему пути без конца слова), Unicode
// вне BMP, форма edgeCheck/result и отказ от некорректного сценария и
// лимитов до SSE.
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// trieResultValues — закрытое множество значений result снимка Trie.
var trieResultValues = map[string]bool{
	"insert":           true,
	"already-existed":  true,
	"found":            true,
	"not-found":        true,
	"prefix-found":     true,
	"prefix-not-found": true,
	"deleted":          true,
	"not-deleted":      true,
}

// trieEdgeStates — закрытое множество состояний ребра в edgeCheck.
var trieEdgeStates = map[string]bool{
	"reused":  true,
	"created": true,
	"missing": true,
}

// doRunTrie — запрос Run сценария Trie; пустой script — параметр не задан.
func doRunTrie(t *testing.T, script string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/trie", nil)
	q := req.URL.Query()
	if script != "" {
		q.Set("script", script)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunTrieRaw — запрос с сырым query: для невалидного UTF-8 (%FF должен
// доехать байтом 0xFF, а не текстом «%FF»).
func doRunTrieRaw(t *testing.T, rawQuery string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/trie?"+rawQuery, nil)
	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunTrieSSE — успешный Run: базовые проверки SSE-ответа + кадры.
func doRunTrieSSE(t *testing.T, script string) []frame.Frame {
	t.Helper()

	resp := doRunTrie(t, script)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// trieData — снимок Trie кадра после JSON-кругосветки (map).
func trieData(t *testing.T, f frame.Frame) map[string]any {
	t.Helper()

	data, ok := f.Data.(map[string]any)
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	return data
}

// assertTrieFrames — общий прогон последовательности кадров: шаги с 1 по
// порядку и контрактные проверки каждого кадра.
func assertTrieFrames(t *testing.T, frames []frame.Frame) {
	t.Helper()

	for i, f := range frames {
		if f.Step != i+1 {
			t.Fatalf("кадр %d: step = %d, ожидается %d", i, f.Step, i+1)
		}
		assertTrieFrame(t, f)
	}
}

// assertTrieFrame — контрактные проверки одного кадра kind="trie": kind,
// непустое пояснение, форма снимка (nodes/rootId/activePath/edgeCheck/
// result/operation), уникальные канонические идентификаторы, согласованность
// children и активного пути, форма edgeCheck (null или {rune, state};
// синонима status нет) и принадлежность result закрытому множеству.
func assertTrieFrame(t *testing.T, f frame.Frame) {
	t.Helper()

	if f.Kind != frame.KindTrie {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindTrie)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	data := trieData(t, f)

	// Все поля снимка присутствуют в проволочной форме: null тоже контракт.
	for _, key := range []string{"nodes", "rootId", "activePath", "edgeCheck", "result", "operation"} {
		if _, has := data[key]; !has {
			t.Fatalf("кадр шага %d: в снимке нет поля %q", f.Step, key)
		}
	}
	if rootID, _ := data["rootId"].(string); rootID != "root" {
		t.Fatalf("кадр шага %d: rootId = %q, ожидается %q", f.Step, rootID, "root")
	}
	if _, ok := data["operation"].(string); !ok {
		t.Fatalf("кадр шага %d: operation не строка: %#v", f.Step, data["operation"])
	}

	// Узлы: уникальные идентификаторы, булевы отметки конца слова,
	// потомки — массив ссылок на живые узлы (продолжения пути родителя).
	rawNodes, ok := data["nodes"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: nodes не массив: %#v", f.Step, data["nodes"])
	}
	ids := map[string]bool{}
	paths := map[string]string{} // id → путь родителя (id без последнего сегмента)
	for _, e := range rawNodes {
		node, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("кадр шага %d: узел снимка не объект: %#v", f.Step, e)
		}
		id, _ := node["id"].(string)
		if id == "" || ids[id] {
			t.Fatalf("кадр шага %d: id узла пуст или не уникален: %q", f.Step, id)
		}
		ids[id] = true
		if _, ok := node["isEnd"].(bool); !ok {
			t.Fatalf("кадр шага %d: узел %q: isEnd не булево: %#v", f.Step, id, node["isEnd"])
		}
		children, ok := node["children"].([]any)
		if !ok {
			t.Fatalf("кадр шага %d: узел %q: children не массив: %#v", f.Step, id, node["children"])
		}
		paths[id] = trieParentPath(id)
		for _, c := range children {
			child, ok := c.(string)
			if !ok {
				t.Fatalf("кадр шага %d: узел %q: элемент children не строка: %#v", f.Step, id, c)
			}
			if !strings.HasPrefix(child, id+"/") {
				t.Fatalf("кадр шага %d: узел %q: потомок %q не продолжение пути родителя", f.Step, id, child)
			}
		}
	}

	// children каждого узла — по возрастанию кода руны (детерминированный
	// порядок снимка) и только на живые узлы.
	for _, e := range rawNodes {
		node := e.(map[string]any)
		id, _ := node["id"].(string)
		prev := ""
		for _, c := range node["children"].([]any) {
			child := c.(string)
			if !ids[child] {
				t.Fatalf("кадр шага %d: узел %q ссылается %q на отсутствующего потомка", f.Step, id, child)
			}
			if prev != "" && trieLastRuneHex(t, prev) >= trieLastRuneHex(t, child) {
				t.Fatalf("кадр шага %d: узел %q: потомки не по возрастанию кода руны: %q перед %q", f.Step, id, prev, child)
			}
			prev = child
		}
	}

	// Активный путь — массив существующих узлов, последовательные продолжения.
	rawPath, ok := data["activePath"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: activePath не массив: %#v", f.Step, data["activePath"])
	}
	prev := ""
	for _, p := range rawPath {
		id, ok := p.(string)
		if !ok || !ids[id] {
			t.Fatalf("кадр шага %d: activePath содержит %v — не идентификатор узла снимка", f.Step, p)
		}
		if prev != "" && !strings.HasPrefix(id, prev+"/") {
			t.Fatalf("кадр шага %d: activePath не последовательный: %q после %q", f.Step, id, prev)
		}
		prev = id
	}

	// edgeCheck: null или объект {rune, state}; синонимичного поля status нет.
	edge := data["edgeCheck"]
	if edge != nil {
		m, ok := edge.(map[string]any)
		if !ok {
			t.Fatalf("кадр шага %d: edgeCheck не объект: %#v", f.Step, edge)
		}
		if _, has := m["status"]; has {
			t.Fatalf("кадр шага %d: edgeCheck содержит синонимичное поле status: %#v", f.Step, m)
		}
		if len(m) != 2 {
			t.Fatalf("кадр шага %d: edgeCheck = %#v, ожидается ровно {rune, state}", f.Step, m)
		}
		runeStr, _ := m["rune"].(string)
		if utf8.RuneCountInString(runeStr) != 1 {
			t.Fatalf("кадр шага %d: edgeCheck.rune = %q — ожидается строка ровно из одной руны", f.Step, runeStr)
		}
		state, _ := m["state"].(string)
		if !trieEdgeStates[state] {
			t.Fatalf("кадр шага %d: edgeCheck.state = %q вне закрытого множества", f.Step, state)
		}
	}

	// result: null или значение закрытого множества.
	res := data["result"]
	if res != nil {
		s, ok := res.(string)
		if !ok || !trieResultValues[s] {
			t.Fatalf("кадр шага %d: result = %#v вне закрытого множества", f.Step, data["result"])
		}
	}
}

// trieParentPath — путь родителя из идентификатора (без последнего сегмента);
// для корня пустая строка.
func trieParentPath(id string) string {
	if i := strings.LastIndex(id, "/"); i >= 0 {
		return id[:i]
	}
	return ""
}

// trieLastRuneHex — числовой код руны из последнего сегмента идентификатора
// (для проверки порядка потомков).
func trieLastRuneHex(t *testing.T, id string) int {
	t.Helper()

	seg := id
	if i := strings.LastIndex(id, "/"); i >= 0 {
		seg = id[i+1:]
	}
	var code int
	if _, err := fmt.Sscanf(seg, "U+%X", &code); err != nil {
		t.Fatalf("идентификатор %q не в канонической форме U+<hex>: %v", id, err)
	}
	return code
}

// trieID — ожидаемая форма идентификатора узла: «root» и сегмент «U+<hex>
// в верхнем регистре минимальной ширины 4» на каждую руну. Форма зашита
// в тесте явно — это контракт идентификаторов, а не вызов кода рендерера.
func trieID(word string) string {
	var b strings.Builder
	b.WriteString("root")
	for _, ru := range word {
		fmt.Fprintf(&b, "/U+%04X", ru)
	}
	return b.String()
}

// assertTrieTopology — снимок кадра содержит ровно узлы всех префиксов слов
// words, отметки конца слова — у точных слов, а children согласованы
// с топологией.
func assertTrieTopology(t *testing.T, f frame.Frame, words ...string) {
	t.Helper()

	want := make(map[string]bool, len(words))
	for _, w := range words {
		want[w] = true
	}
	assertTrieNodes(t, f, want)
}

// assertTrieNodes — снимок кадра содержит ровно узлы замыкания по префиксам
// перечисленных путей; отметка конца слова — только у путей со значением true
// (промежуточные состояния снятия флага/несрезанных узлов задаются явно).
func assertTrieNodes(t *testing.T, f frame.Frame, want map[string]bool) {
	t.Helper()

	data := trieData(t, f)
	// Замыкание по префиксам: каждый узел пути существует, отметка — false,
	// затем точные слова помечаются своим значением (двумя проходами, чтобы
	// промежуточные руны длинных слов не затирали отметки коротких слов).
	wantNodes := map[string]bool{"root": false}
	wantChildren := map[string]map[string]bool{"root": {}}
	for word := range want {
		prev := "root"
		for i := 1; i <= len([]rune(word)); i++ {
			id := trieID(string([]rune(word)[:i]))
			if _, seen := wantNodes[id]; !seen {
				wantNodes[id] = false
			}
			if wantChildren[prev] == nil {
				wantChildren[prev] = map[string]bool{}
			}
			wantChildren[prev][id] = true
			if _, seen := wantChildren[id]; !seen {
				wantChildren[id] = map[string]bool{}
			}
			prev = id
		}
	}
	for word, isEnd := range want {
		wantNodes[trieID(word)] = isEnd
	}

	gotNodes := map[string]bool{}
	for _, e := range data["nodes"].([]any) {
		node, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("кадр шага %d: узел снимка не объект: %#v", f.Step, e)
		}
		id := node["id"].(string)
		gotNodes[id] = true
		if got, want := node["isEnd"].(bool), wantNodes[id]; got != want {
			t.Fatalf("кадр шага %d: узел %q: isEnd = %v, ожидается %v", f.Step, id, got, want)
		}
		gotChildren := map[string]bool{}
		for _, c := range node["children"].([]any) {
			gotChildren[c.(string)] = true
		}
		if !maps.Equal(gotChildren, wantChildren[id]) {
			t.Fatalf("кадр шага %d: узел %q: children = %v, ожидается %v", f.Step, id, gotChildren, wantChildren[id])
		}
	}
	wantKeys := map[string]bool{}
	for id := range wantNodes {
		wantKeys[id] = true
	}
	if !maps.Equal(gotNodes, wantKeys) {
		t.Fatalf("кадр шага %d: узлы снимка = %v, ожидается %v", f.Step, gotNodes, wantKeys)
	}
}

// assertEdgeCheck — в кадре объект {rune, state} с ожидаемыми значениями.
func assertEdgeCheck(t *testing.T, f frame.Frame, runeStr, state string) {
	t.Helper()

	m, ok := trieData(t, f)["edgeCheck"].(map[string]any)
	if !ok {
		t.Fatalf("кадр шага %d: edgeCheck = %#v, ожидается объект {rune, state}", f.Step, trieData(t, f)["edgeCheck"])
	}
	if got, _ := m["rune"].(string); got != runeStr {
		t.Fatalf("кадр шага %d: edgeCheck.rune = %q, ожидается %q", f.Step, got, runeStr)
	}
	if got, _ := m["state"].(string); got != state {
		t.Fatalf("кадр шага %d: edgeCheck.state = %q, ожидается %q", f.Step, got, state)
	}
}

// assertEdgeCheckNull — в кадре ровно edgeCheck: null.
func assertEdgeCheckNull(t *testing.T, f frame.Frame) {
	t.Helper()

	if got := trieData(t, f)["edgeCheck"]; got != nil {
		t.Fatalf("кадр шага %d: edgeCheck = %v, ожидается null", f.Step, got)
	}
}

// assertResult — в кадре result равен ожидаемому значению закрытого множества.
func assertResult(t *testing.T, f frame.Frame, want string) {
	t.Helper()

	got, ok := trieData(t, f)["result"].(string)
	if !ok {
		t.Fatalf("кадр шага %d: result = %#v, ожидается %q", f.Step, trieData(t, f)["result"], want)
	}
	if got != want {
		t.Fatalf("кадр шага %d: result = %q, ожидается %q", f.Step, got, want)
	}
}

// assertResultNull — в кадре ровно result: null.
func assertResultNull(t *testing.T, f frame.Frame) {
	t.Helper()

	if got := trieData(t, f)["result"]; got != nil {
		t.Fatalf("кадр шага %d: result = %v, ожидается null", f.Step, got)
	}
}

// countResultFrames — число кадров с заполненным result.
func countResultFrames(frames []frame.Frame) int {
	n := 0
	for _, f := range frames {
		if trieResultOf(f) != "" {
			n++
		}
	}
	return n
}

// trieResultOf — значение result кадра ("" — null).
func trieResultOf(f frame.Frame) string {
	data, ok := f.Data.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := data["result"].(string)
	return s
}

// trieExplanationHas — у кадра проверяется содержимое пояснения.
func trieExplanationHas(t *testing.T, f frame.Frame, substrings ...string) {
	t.Helper()

	for _, s := range substrings {
		if !strings.Contains(f.Explanation, s) {
			t.Fatalf("пояснение кадра шага %d = %q, ожидается упоминание %q", f.Step, f.Explanation, s)
		}
	}
}

// assertActivePath — активный путь кадра равен ожидаемым идентификаторам.
func assertActivePath(t *testing.T, f frame.Frame, want ...string) {
	t.Helper()

	raw, _ := trieData(t, f)["activePath"].([]any)
	got := make([]string, len(raw))
	for i, p := range raw {
		got[i] = p.(string)
	}
	if len(got) != len(want) {
		t.Fatalf("кадр шага %d: activePath = %v, ожидается %v", f.Step, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("кадр шага %d: activePath = %v, ожидается %v", f.Step, got, want)
		}
	}
}

// TestTrieRunScenarioSSE — сценарий с общим префиксом, точным поиском
// и промахом, проверкой префикса: кадры посещения несут edgeCheck с руной
// и состоянием ребра, активный путь растёт (при missing — только до
// родителя), result — null до кадра результата.
func TestTrieRunScenarioSSE(t *testing.T) {
	// к=U+043A о=U+043E т=U+0442 д=U+0434 р=U+0440
	const (
		idRoot = "root"
		idК    = "root/U+043A"
		idКо   = "root/U+043A/U+043E"
		idКот  = "root/U+043A/U+043E/U+0442"
		idКод  = "root/U+043A/U+043E/U+0434"
	)

	frames := doRunTrieSSE(t, "insert кот\ninsert код\nsearch кот\nsearch крот\nprefix ко")
	assertTrieFrames(t, frames)
	if len(frames) != 19 { // 1 старт + (3+1) + (3+1) + (3+1) + (2+1) + (2+1)
		t.Fatalf("кадров = %d, ожидается 19", len(frames))
	}

	// Стартовый кадр: пустое Trie, активного пути нет.
	assertEdgeCheckNull(t, frames[0])
	assertResultNull(t, frames[0])
	assertActivePath(t, frames[0])
	assertTrieTopology(t, frames[0])
	if op, _ := trieData(t, frames[0])["operation"].(string); op != "" {
		t.Fatalf("стартовый кадр: operation = %q, ожидается пустая строка", op)
	}
	trieExplanationHas(t, frames[0], "Сценарий Trie:", "Дерево начинается с пустого корня")

	// Вставка «кот» в пустое Trie: все три ребра created, на последней руне —
	// отметка конца слова (в том же событии посещения).
	for i, runeStr := range []string{"к", "о", "т"} {
		f := frames[1+i]
		assertEdgeCheck(t, f, runeStr, "created")
		assertResultNull(t, f)
		trieExplanationHas(t, f, "создаём новое ребро и узел", fmt.Sprintf("руны «%s»", runeStr))
	}
	assertActivePath(t, frames[3], idRoot, idК, idКо, idКот)
	trieExplanationHas(t, frames[3], "Узел отмечен концом слова")
	assertTrieTopology(t, frames[3], "кот")

	// Кадр результата вставки: единственный на команду с result,
	// edgeCheck null, активного пути нет.
	assertResult(t, frames[4], "insert")
	assertEdgeCheckNull(t, frames[4])
	assertActivePath(t, frames[4])
	trieExplanationHas(t, frames[4], "добавлено в Trie")

	// Вставка «код» по общему префиксу: к и о — reused, д — created.
	assertEdgeCheck(t, frames[5], "к", "reused")
	assertEdgeCheck(t, frames[6], "о", "reused")
	assertEdgeCheck(t, frames[7], "д", "created")
	assertActivePath(t, frames[7], idRoot, idК, idКо, idКод)
	assertTrieTopology(t, frames[7], "кот", "код")
	assertResult(t, frames[8], "insert")
	assertEdgeCheckNull(t, frames[8])

	// Точный поиск «кот»: все руны reused, результат found.
	for i, runeStr := range []string{"к", "о", "т"} {
		assertEdgeCheck(t, frames[9+i], runeStr, "reused")
	}
	assertActivePath(t, frames[11], idRoot, idК, idКо, idКот)
	assertResult(t, frames[12], "found")
	assertEdgeCheckNull(t, frames[12])

	// Промах поиска «крот» (к-р-о-т): вторая руна «р» отсутствует — событие
	// проверки с состоянием missing; отсутствующий потомок не в топологии
	// и не имеет идентификатора, активный путь — только до родителя (к).
	assertEdgeCheck(t, frames[13], "к", "reused")
	assertEdgeCheck(t, frames[14], "р", "missing")
	assertActivePath(t, frames[14], idRoot, idК)
	if ids := trieNodeIDsOf(t, frames[14]); ids[trieID("кор")] {
		t.Fatalf("кадр шага %d: отсутствующий потомок «кор» присутствует в снимке", frames[14].Step)
	}
	assertTrieTopology(t, frames[14], "кот", "код")
	assertResult(t, frames[15], "not-found")
	assertEdgeCheckNull(t, frames[15])
	trieExplanationHas(t, frames[15], "штатный исход поиска")

	// Проверка префикса «ко»: обе руны reused, результат prefix-found.
	assertEdgeCheck(t, frames[16], "к", "reused")
	assertEdgeCheck(t, frames[17], "о", "reused")
	assertResult(t, frames[18], "prefix-found")
	assertEdgeCheckNull(t, frames[18])

	// Ровно один кадр результата на каждую команду сценария.
	if n := countResultFrames(frames); n != 5 {
		t.Fatalf("кадров с result = %d, ожидается по одному на каждую из 5 команд", n)
	}
}

// trieNodeIDsOf — множество идентификаторов узлов снимка кадра.
func trieNodeIDsOf(t *testing.T, f frame.Frame) map[string]bool {
	t.Helper()

	ids := map[string]bool{}
	for _, e := range trieData(t, f)["nodes"].([]any) {
		ids[e.(map[string]any)["id"].(string)] = true
	}
	return ids
}

// TestTrieRunDeletePrunesBranchSSE — удаление слова, узлы которого лежат
// на пути к более длинному слову: pruning останавливается на ветвлении,
// кадр срезанного узла показывает уже исчезнувший узел.
func TestTrieRunDeletePrunesBranchSSE(t *testing.T) {
	frames := doRunTrieSSE(t, "insert кот\ninsert код\ndelete кот")
	assertTrieFrames(t, frames)
	if len(frames) != 14 { // 1 старт + 4 + 4 + (3 посещения + 1 pruning + 1 результат)
		t.Fatalf("кадров = %d, ожидается 14", len(frames))
	}

	// Последнее событие посещения снимает флаг конца слова: узел «кот»
	// ещё в дереве, но уже не конец слова.
	assertTrieNodes(t, frames[11], map[string]bool{"кот": false, "код": true})
	trieExplanationHas(t, frames[11], "Отметка конца слова снята")

	// Кадр pruning: только «кот» срезан (ветвление «ко» живо — на нём
	// висит «код»), edgeCheck и result — null, активного пути нет.
	assertEdgeCheckNull(t, frames[12])
	assertResultNull(t, frames[12])
	assertActivePath(t, frames[12])
	assertTrieTopology(t, frames[12], "код")
	trieExplanationHas(t, frames[12], "срезаем узел", "ребру «т»")

	// Кадр результата удаления: итог читается из снимка.
	assertResult(t, frames[13], "deleted")
	assertEdgeCheckNull(t, frames[13])
	assertTrieTopology(t, frames[13], "код")
}

// TestTrieRunDeletePrunesToEmptySSE — удаление единственного слова срезает
// всю цепочку: по кадру на каждый срезанный узел, финальный снимок пуст.
func TestTrieRunDeletePrunesToEmptySSE(t *testing.T) {
	frames := doRunTrieSSE(t, "insert кот\ndelete кот")
	assertTrieFrames(t, frames)
	if len(frames) != 12 { // 1 старт + 4 + 3 посещения + 3 pruning + 1 результат
		t.Fatalf("кадров = %d, ожидается 12", len(frames))
	}

	// Pruning снизу вверх: по кадру на каждый исчезающий узел
	// (промежуточные узлы цепочки не были концами слов).
	assertTrieNodes(t, frames[8], map[string]bool{"ко": false})
	assertTrieNodes(t, frames[9], map[string]bool{"к": false})
	assertTrieNodes(t, frames[10], nil)
	for i := 8; i <= 10; i++ {
		assertEdgeCheckNull(t, frames[i])
		assertResultNull(t, frames[i])
		assertActivePath(t, frames[i])
	}
	assertResult(t, frames[11], "deleted")
	assertEdgeCheckNull(t, frames[11])
	assertTrieTopology(t, frames[11])
}

// TestTrieRunResultVariantsSSE — каждый вариант закрытого множества result
// отдельным случаем, включая no-op-значения; промежуточные кадры несут
// result: null, кадр результата — значение и edgeCheck: null.
func TestTrieRunResultVariantsSSE(t *testing.T) {
	cases := []struct {
		name      string
		script    string
		commands  int
		want      string
		wantWords []string // топология последнего кадра
	}{
		{"insert", "insert кот", 1, "insert", []string{"кот"}},
		{"already-existed", "insert кот\ninsert кот", 2, "already-existed", []string{"кот"}},
		{"found", "insert кот\nsearch кот", 2, "found", []string{"кот"}},
		{"not-found — промах по отсутствующему ребру", "insert кот\nsearch крот", 2, "not-found", []string{"кот"}},
		{"prefix-found", "insert кот\nprefix ко", 2, "prefix-found", []string{"кот"}},
		{"prefix-not-found", "insert кот\nprefix п", 2, "prefix-not-found", []string{"кот"}},
		{"deleted", "insert кот\ndelete кот", 2, "deleted", nil},
		{"not-deleted — промах по отсутствующему ребру", "insert кот\ndelete крот", 2, "not-deleted", []string{"кот"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			frames := doRunTrieSSE(t, tc.script)
			assertTrieFrames(t, frames)

			// Ровно один кадр результата на команду; последний кадр —
			// результат последней команды, промежуточные кадры её не несут.
			resultIdx := []int{}
			for i, f := range frames {
				if trieResultOf(f) != "" {
					resultIdx = append(resultIdx, i)
				}
			}
			if len(resultIdx) != tc.commands {
				t.Fatalf("кадров с result = %d, ожидается %d", len(resultIdx), tc.commands)
			}
			if resultIdx[len(resultIdx)-1] != len(frames)-1 {
				t.Fatalf("последний кадр не кадр результата: result-кадры на позициях %v из %d кадров", resultIdx, len(frames))
			}
			last := frames[len(frames)-1]
			assertResult(t, last, tc.want)
			assertEdgeCheckNull(t, last)
			assertTrieTopology(t, last, tc.wantWords...)

			// Кадры последней команды до её результата несут result: null.
			start := 0
			if len(resultIdx) > 1 {
				start = resultIdx[len(resultIdx)-2] + 1
			}
			for _, f := range frames[start : len(frames)-1] {
				assertResultNull(t, f)
			}
		})
	}
}

// TestTrieRunNoOpPathsSSE — no-op-сценарии: промахи по существующему пути,
// конечный узел которого не отмечен как конец слова (search, delete),
// и повторная вставка. События посещения/проверки и результат сверяются,
// топология не меняется, pruning нет.
func TestTrieRunNoOpPathsSSE(t *testing.T) {
	t.Run("search по существующему пути без конца слова", func(t *testing.T) {
		frames := doRunTrieSSE(t, "insert кот\nsearch ко")
		assertTrieFrames(t, frames)
		if len(frames) != 8 { // 1 старт + 4 + 2 посещения + 1 результат
			t.Fatalf("кадров = %d, ожидается 8", len(frames))
		}
		assertEdgeCheck(t, frames[5], "к", "reused")
		assertEdgeCheck(t, frames[6], "о", "reused")
		assertTrieTopology(t, frames[6], "кот")
		assertResult(t, frames[7], "not-found")
		assertEdgeCheckNull(t, frames[7])
		trieExplanationHas(t, frames[7], "штатный исход поиска")
	})

	t.Run("delete по существующему пути без конца слова", func(t *testing.T) {
		frames := doRunTrieSSE(t, "insert кот\ndelete ко")
		assertTrieFrames(t, frames)
		// Ровно 8 кадров — pruning нет, флаг конца слова не менялся.
		if len(frames) != 8 {
			t.Fatalf("кадров = %d, ожидается 8", len(frames))
		}
		assertEdgeCheck(t, frames[5], "к", "reused")
		assertEdgeCheck(t, frames[6], "о", "reused")
		assertTrieTopology(t, frames[6], "кот")
		assertResult(t, frames[7], "not-deleted")
		assertEdgeCheckNull(t, frames[7])
		trieExplanationHas(t, frames[7], "штатный исход удаления", "дерево не изменилось")
	})

	t.Run("повторная вставка", func(t *testing.T) {
		frames := doRunTrieSSE(t, "insert кот\ninsert кот")
		assertTrieFrames(t, frames)
		if len(frames) != 9 { // 1 старт + 4 + 4
			t.Fatalf("кадров = %d, ожидается 9", len(frames))
		}
		// Повторная вставка: все руны reused, флаг конца слова не меняется.
		assertEdgeCheck(t, frames[5], "к", "reused")
		assertEdgeCheck(t, frames[6], "о", "reused")
		assertEdgeCheck(t, frames[7], "т", "reused")
		assertTrieTopology(t, frames[7], "кот")
		assertResult(t, frames[8], "already-existed")
		assertEdgeCheckNull(t, frames[8])
		trieExplanationHas(t, frames[8], "уже хранится в Trie", "допустимый шаг сценария")
	})
}

// TestTrieRunMissOnEmptyTrieSSE — промах поиска в пустом Trie: единственное
// событие проверки отсутствующей руны и результат, узлы не создаются.
func TestTrieRunMissOnEmptyTrieSSE(t *testing.T) {
	frames := doRunTrieSSE(t, "search кот")
	assertTrieFrames(t, frames)
	if len(frames) != 3 { // 1 старт + 1 проверка + 1 результат
		t.Fatalf("кадров = %d, ожидается 3", len(frames))
	}
	assertEdgeCheck(t, frames[1], "к", "missing")
	assertActivePath(t, frames[1], "root") // родитель отсутствующего ребра — корень
	assertTrieTopology(t, frames[1])
	assertResult(t, frames[2], "not-found")
	assertEdgeCheckNull(t, frames[2])
}

// TestTrieRunNonBMPRuneSSE — символ вне BMP: edgeCheck.rune — JSON-строка
// ровно из одной руны «😀», идентификатор узла — root/U+1F600 (полный hex-код
// руны, не surrogate-половина U+D83D).
func TestTrieRunNonBMPRuneSSE(t *testing.T) {
	if got := trieID("😀"); got != "root/U+1F600" {
		t.Fatalf("ожидаемый идентификатор руны 😀 = %q, получено %q", "root/U+1F600", got)
	}

	frames := doRunTrieSSE(t, "insert 😀\ninsert 😀😀")
	assertTrieFrames(t, frames)
	if len(frames) != 6 { // 1 старт + (1+1) + (2+1)
		t.Fatalf("кадров = %d, ожидается 6", len(frames))
	}

	assertEdgeCheck(t, frames[1], "😀", "created")
	assertActivePath(t, frames[1], "root", "root/U+1F600")
	assertResult(t, frames[2], "insert")

	// Вторая вставка: первая руна reused, вторая — created (полный путь
	// из полного кода руны, не surrogate-половины).
	assertEdgeCheck(t, frames[3], "😀", "reused")
	assertEdgeCheck(t, frames[4], "😀", "created")
	assertActivePath(t, frames[4], "root", "root/U+1F600", "root/U+1F600/U+1F600")

	assertResult(t, frames[5], "insert")
	assertTrieTopology(t, frames[5], "😀", "😀😀")
	for _, f := range frames {
		if strings.Contains(fmt.Sprintf("%v", trieNodeIDsOf(t, f)), "D83D") {
			t.Fatalf("кадр шага %d: в идентификаторах surrogate-половина U+D83D", f.Step)
		}
	}
}

// TestTrieRunBlankLinesIgnoredSSE — пустые строки и строки из пробелов
// игнорируются.
func TestTrieRunBlankLinesIgnoredSSE(t *testing.T) {
	frames := doRunTrieSSE(t, "\n\n insert кот \n \t\ndelete кот\n\n")
	assertTrieFrames(t, frames)
	if len(frames) != 12 { // как «insert кот\ndelete кот»
		t.Fatalf("кадров = %d, ожидается 12", len(frames))
	}
	assertTrieTopology(t, frames[len(frames)-1])
}

// TestTrieRunNodeLimitSSE — лимит живых rune-узлов применяется к пиковому
// размеру Trie: максимум по состоянию после каждой команды, а не только
// к финальному.
func TestTrieRunNodeLimitSSE(t *testing.T) {
	word100 := strings.Repeat("а", 100)

	t.Run("100 живых rune-узлов принимаются", func(t *testing.T) {
		frames := doRunTrieSSE(t, "insert "+word100)
		assertTrieFrames(t, frames)
		last := frames[len(frames)-1]
		assertTrieTopology(t, last, word100)
		if got := len(trieNodeIDsOf(t, last)); got != 101 { // корень + 100 rune-узлов
			t.Fatalf("узлов в финальном снимке = %d, ожидается 101", got)
		}
	})

	t.Run("101 узел отвергается до SSE", func(t *testing.T) {
		script := "insert " + word100 + "\ninsert б"
		resp := doRunTrie(t, script)
		assertRejectedBeforeSSE(t, resp, "лимит живых rune-узлов", "№2")
	})

	t.Run("пик больше 100 при финале не больше 100 отвергается", func(t *testing.T) {
		// После второй команды пик 101 (финал сценария был бы 100):
		// лимит сверяется после каждой команды.
		script := "insert " + word100 + "\ninsert б\ndelete б"
		resp := doRunTrie(t, script)
		assertRejectedBeforeSSE(t, resp, "лимит живых rune-узлов", "№2")
	})

	t.Run("500 команд принимаются", func(t *testing.T) {
		script := strings.Repeat("insert а\n", 500)
		frames := doRunTrieSSE(t, script)
		assertTrieFrames(t, frames)
		// 1 старт + по 2 кадра на команду (повторные вставки — легальные шаги).
		if len(frames) != 1001 {
			t.Fatalf("кадров = %d, ожидается 1001", len(frames))
		}
		assertTrieTopology(t, frames[len(frames)-1], "а")
	})

	t.Run("501 команда отвергается до SSE", func(t *testing.T) {
		script := strings.Repeat("insert а\n", 501)
		resp := doRunTrie(t, script)
		assertRejectedBeforeSSE(t, resp, "не более 500 команд")
	})
}

// TestTrieRunLengthBoundarySSE — граница лимита длины изолирована от лимита
// живых узлов: search из 100 повторных 😀 в пустом Trie принимается и
// доводится до кадра результата not-found без создания узлов, из 101 —
// отвергается до SSE.
func TestTrieRunLengthBoundarySSE(t *testing.T) {
	t.Run("100 рун принимаются", func(t *testing.T) {
		script := "search " + strings.Repeat("😀", 100)
		frames := doRunTrieSSE(t, script)
		assertTrieFrames(t, frames)
		if len(frames) != 3 { // 1 старт + 1 проверка отсутствующей руны + 1 результат
			t.Fatalf("кадров = %d, ожидается 3", len(frames))
		}
		assertEdgeCheck(t, frames[1], "😀", "missing")
		assertActivePath(t, frames[1], "root") // родитель отсутствующего ребра — корень
		assertResult(t, frames[2], "not-found")
		assertEdgeCheckNull(t, frames[2])
		assertTrieTopology(t, frames[2]) // узлов не создано
	})

	t.Run("101 руна отвергается до SSE", func(t *testing.T) {
		script := "search " + strings.Repeat("😀", 101)
		resp := doRunTrie(t, script)
		assertRejectedBeforeSSE(t, resp, "длиннее 100 rune")
	})
}

// assertRejectedBeforeSSE — ответ 400 с JSON-ошибкой, поток не начат;
// тело ошибки содержит все перечисленные подстроки.
func assertRejectedBeforeSSE(t *testing.T, resp *httptest.ResponseRecorder, want ...string) {
	t.Helper()

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("статус = %d, ожидается 400: %s", resp.Code, resp.Body.String())
	}
	if strings.HasPrefix(resp.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("отказ должен случиться до SSE, получен поток: %q", resp.Body.String())
	}
	var got map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело ошибки не JSON: %v\n%s", err, resp.Body.String())
	}
	for _, s := range want {
		if !strings.Contains(got["error"], s) {
			t.Fatalf("тело ошибки = %s, ожидается упоминание %q", got["error"], s)
		}
	}
}

// TestTrieRunValidation — синтаксис, аргументы, невалидный UTF-8 и лимиты
// отвергаются до SSE, без частично начатого потока.
func TestTrieRunValidation(t *testing.T) {
	tooLong := strings.Repeat("а", 101)
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{"нет параметра script", "", "параметр script обязателен"},
		{"только пустые строки", "\n \n\t\n", "сценарий пуст"},
		{"неизвестная команда", "upsert кот", "неизвестная команда"},
		{"команда без аргумента", "insert", "не хватает аргумента"},
		{"лишние аргументы", "insert кот пёс", "лишние аргументы"},
		{"слово длиннее 100 rune", "insert " + tooLong, "длиннее 100 rune"},
		{"префикс длиннее 100 rune", "prefix " + tooLong, "длиннее 100 rune"},
		{"ошибка указывает строку", "insert кот\nupsert пёс\ndelete кот", "на строке 2"},
		{"лимит команд", strings.Repeat("insert а\n", 501), "не более 500 команд"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertRejectedBeforeSSE(t, doRunTrie(t, tt.script), tt.want)
		})
	}

	t.Run("невалидный UTF-8", func(t *testing.T) {
		// Сыро закодированный невалидный байт %FF доезжает байтом 0xFF:
		// аргумент — не последовательность Unicode scalar values.
		assertRejectedBeforeSSE(t, doRunTrieRaw(t, "script=insert%20%FF"), "невалидный UTF-8")
	})

	t.Run("параметр script задан дважды", func(t *testing.T) {
		assertRejectedBeforeSSE(t, doRunTrieRaw(t, "script=insert%20%D0%B0&script=delete%20%D0%B0"), "задан 2 раз")
	})
}

// TestTrieRunMatchesDirectRenderer — рендерер и транспорт не расходятся:
// кадры из эндпоинта совпадают с кадрами, собранными напрямую у рендерера.
func TestTrieRunMatchesDirectRenderer(t *testing.T) {
	script := "insert кот\ninsert код\nsearch кот\ndelete кот\ninsert 😀"
	frames := doRunTrieSSE(t, script)

	want, err := render.TrieFrames([]render.TrieCommand{
		{Op: render.TrieOpInsert, Word: "кот"},
		{Op: render.TrieOpInsert, Word: "код"},
		{Op: render.TrieOpSearch, Word: "кот"},
		{Op: render.TrieOpDelete, Word: "кот"},
		{Op: render.TrieOpInsert, Word: "😀"},
	})
	if err != nil {
		t.Fatalf("рендерер вернул ошибку: %v", err)
	}

	if len(frames) != len(want) {
		t.Fatalf("кадров = %d, рендерер даёт %d", len(frames), len(want))
	}
	for i := range want {
		// Обе стороны прогоняем через JSON-кругосветку: снимок из потока
		// приходит как map, порядок ключей не должен влиять на сравнение.
		aw, _ := json.Marshal(toWire(t, frames[i]))
		bw, _ := json.Marshal(toWire(t, want[i]))
		if !bytes.Equal(aw, bw) {
			t.Errorf("кадр %d отличается: эндпоинт %s, рендерер %s", i, aw, bw)
		}
	}
}
