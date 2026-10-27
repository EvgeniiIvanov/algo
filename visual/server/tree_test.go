// Тесты шва 2 для обходов дерева (см. docs/SPEC.md): вход → SSE-поток →
// последовательность кадров kind="tree". Покрываются все три порядка на
// известном дереве, пустое дерево и отказ от невалидного входа/лимита до SSE.
package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// doRunTree — запрос Run обхода дерева.
func doRunTree(t *testing.T, tree, order string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/tree-traversal", nil)
	q := req.URL.Query()
	if tree != "" {
		q.Set("tree", tree)
	}
	if order != "" {
		q.Set("order", order)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunTreeSSE — успешный Run: базовые проверки SSE-ответа + кадры.
func doRunTreeSSE(t *testing.T, tree, order string) []frame.Frame {
	t.Helper()

	resp := doRunTree(t, tree, order)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// knownTree — известное асимметричное дерево с повторяющимся значением:
//
//	     1
//	   /   \
//	  2     2
//	 /       \
//	4         5
const knownTreeJSON = `{"value":1,"left":{"value":2,"left":{"value":4}},"right":{"value":2,"right":{"value":5}}}`

// assertTreeFrame — базовые проверки кадра kind="tree" на известном дереве.
func assertTreeFrame(t *testing.T, f frame.Frame, step int, activeID string) {
	t.Helper()

	if f.Kind != frame.KindTree {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindTree)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	if f.Step != step {
		t.Fatalf("номер шага = %d, ожидается %d", f.Step, step)
	}
	data, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок — map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	if got, ok := data["activeId"].(string); !ok || got != activeID {
		t.Fatalf("кадр шага %d: activeId = %v, ожидается %q", f.Step, data["activeId"], activeID)
	}
	if nodes, ok := data["nodes"].([]any); !ok || len(nodes) != 5 {
		t.Fatalf("кадр шага %d: узлов в снимке %v, ожидается 5", f.Step, len(mustSliceAny(t, data, "nodes")))
	}
	if got, ok := data["order"].(string); !ok || got == "" {
		t.Fatalf("кадр шага %d: пустой порядок в снимке", f.Step)
	}
	if _, ok := data["rootId"].(string); !ok {
		t.Fatalf("кадр шага %d: rootId не строка", f.Step)
	}
	if visited, ok := data["visited"].([]any); !ok || len(visited) != step-1 {
		t.Fatalf("кадр шага %d: посещённых узлов %v, ожидается %d", f.Step, len(mustSliceAny(t, data, "visited")), step-1)
	}
}

func mustSliceAny(t *testing.T, data map[string]any, key string) []any {
	t.Helper()
	s, ok := data[key].([]any)
	if !ok {
		t.Fatalf("поле %q не массив: %#v", key, data[key])
	}
	return s
}

// assertTreePathOrder — на известном дереве проверяет последовательность
// активных узлов по путям от корня ("": корень).
func assertTreePathOrder(t *testing.T, frames []frame.Frame, wantPaths []string) {
	t.Helper()

	if len(frames) != len(wantPaths)+1 {
		t.Fatalf("кадров = %d, ожидается %d (стартовый + по кадру на посещение)", len(frames), len(wantPaths)+1)
	}
	// Стартовый кадр: полный снимок, активного узла нет.
	assertTreeFrame(t, frames[0], 1, "")
	for i, want := range wantPaths {
		assertTreeFrame(t, frames[i+1], i+2, want)
	}
}

func TestTreeTraversalInOrderSSE(t *testing.T) {
	// In-order на известном дереве: 4 (LL), 2 (L), 1 (root), 2 (R), 5 (RR).
	frames := doRunTreeSSE(t, knownTreeJSON, "in-order")
	assertTreePathOrder(t, frames, []string{"LL", "L", "root", "R", "RR"})

	if !strings.Contains(frames[1].Explanation, "4") {
		t.Fatalf("первый шаг не упоминает значение 4: %q", frames[1].Explanation)
	}
}

func TestTreeTraversalPreOrderSSE(t *testing.T) {
	// Pre-order: корень, левое поддерево (2, 4), правое (2, 5).
	frames := doRunTreeSSE(t, knownTreeJSON, "pre-order")
	assertTreePathOrder(t, frames, []string{"root", "L", "LL", "R", "RR"})
}

func TestTreeTraversalPostOrderSSE(t *testing.T) {
	// Post-order: 4 (LL), 2 (L), 5 (RR), 2 (R), корень (root) — последним.
	frames := doRunTreeSSE(t, knownTreeJSON, "post-order")
	assertTreePathOrder(t, frames, []string{"LL", "L", "RR", "R", "root"})
}

func TestTreeTraversalEmptyTreeSSE(t *testing.T) {
	frames := doRunTreeSSE(t, "null", "in-order")
	if len(frames) != 1 {
		t.Fatalf("кадров = %d, ожидается 1 (только исходный кадр)", len(frames))
	}
	data, ok := frames[0].Data.(map[string]any)
	if !ok {
		t.Fatalf("data не является снимком: %#v", frames[0].Data)
	}
	if len(mustSliceAny(t, data, "nodes")) != 0 {
		t.Fatalf("узлов в снимке пустого дерева %d, ожидается 0", len(mustSliceAny(t, data, "nodes")))
	}
	if got := data["rootId"].(string); got != "" {
		t.Fatalf("rootId пустого дерева = %q, ожидается пустая строка", got)
	}
	if len(mustSliceAny(t, data, "visited")) != 0 {
		t.Fatalf("посещённых узлов %d, ожидается 0", len(mustSliceAny(t, data, "visited")))
	}
	if data["activeId"].(string) != "" {
		t.Fatalf("activeId = %q, ожидается пустая строка", data["activeId"])
	}
}

// deepTreeLikeJSON — JSON цепочки из n узлов влево: {"value":1,"left":..."left":{"value":n}}.
func deepTreeLikeJSON(n int) string {
	body := fmt.Sprintf(`{"value":%d}`, n)
	for i := n - 1; i >= 1; i-- {
		body = fmt.Sprintf(`{"value":%d,"left":%s}`, i, body)
	}
	return body
}

func TestTreeTraversalRejections(t *testing.T) {
	tests := []struct {
		name  string
		tree  string
		order string
		want  string
	}{
		{"нет параметра tree", "", "in-order", "параметр tree обязателен"},
		{"нет параметра order", knownTreeJSON, "", "параметр order обязателен"},
		{"незакрытый JSON", `{"value":1,`, "in-order", "не удалось разобрать JSON"},
		{"не объект вместо узла", `[1,2]`, "in-order", "должен быть объектом"},
		{"значение не число", `{"value":"1"}`, "pre-order", "value обязательно"},
		{"значение не целое", `{"value":1.5}`, "post-order", "не целое"},
		{"неизвестный порядок", knownTreeJSON, "breadth-first", "неизвестный порядок"},
		{"лимит узлов", deepTreeLikeJSON(101), "in-order", "максимум 100"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunTree(t, tt.tree, tt.order)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("статус = %d, ожидается 400: %s", resp.Code, resp.Body.String())
			}
			if resp.Header().Get("Content-Type") == "text/event-stream" {
				t.Fatalf("отказ должен случиться до SSE, получен поток: %q", resp.Body.String())
			}
			if got := resp.Body.String(); !strings.Contains(got, tt.want) {
				t.Fatalf("тело ошибки = %s, ожидается упоминание %q", got, tt.want)
			}
		})
	}
}
