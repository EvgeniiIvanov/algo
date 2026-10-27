// Тесты шва 2 для обходов дерева (см. docs/SPEC.md): вход → SSE-поток →
// последовательность кадров kind="tree". Покрываются все три порядка на
// известном дереве, пустое дерево и отказ от невалидного входа/лимита до SSE.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
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
		{"не целое на глубине 2", `{"value":1,"left":{"value":2,"left":{"value":2.5}}}`, "in-order", "(путь LL)"},
		// 2^63 представимо как float64, но не как int64: округление через
		// float64 пропускало бы 9223372036854775808 с переполнением int.
		{"значение больше int64", `{"value":9223372036854775808}`, "in-order", "за пределами целого"},
		{"значение меньше -int64", `{"value":-9223372036854775809}`, "pre-order", "за пределами целого"},
		{"экспонента не целое", `{"value":1e-2}`, "in-order", "не целое"},
		{"неизвестный порядок", knownTreeJSON, "breadth-first", "неизвестный порядок"},
		{"лимит узлов", deepTreeLikeJSON(101), "in-order", "разрешено не более 100 узлов"},
		// D1: строго одно JSON-значение — лишнее содержимое после дерева
		// отвергается, а не молча игнорируется вторым Decode.
		{"конкатенированные JSON-значения", `{"value":1}{"value":2}`, "in-order", "лишнее содержимое после JSON дерева"},
		{"мусор после значения", `{"value":1} x`, "in-order", "не удалось разобрать JSON дерева"},
		// D2: огромная экспонента и длинный литерал — в диапазон не
		// проверяем материализацией: отсекаем по границам до вычислений.
		// Показатель в записи int остаётся MaxInt/MinInt (strconv.Atoi
		// их принимает): арифметика на границах переполняется, поэтому
		// первый отсечной признак — сама граница показателя, до E/k.
		{"показатель MaxInt64", `{"value":1e9223372036854775807}`, "in-order", "за пределами целого"},
		{"показатель MinInt64", `{"value":1e-9223372036854775808}`, "in-order", "не целое"},
		// 1e4000000000 не дотягивает до границы int64-представления int,
		// но int64 в принципе не бывает выше 19 значащих цифр: значение
		// ограничено раньше, чем переполнились бы E := exp - len(frac).
		{"большая экспонента внутри int", `{"value":1e4000000000}`, "in-order", "за пределами целого"},
		{"большая отрицательная экспонента внутри int", `{"value":1e-4000000000}`, "in-order", "не целое"},
		{"дробная вариация у границы", `{"value":1.0e9223372036854775807}`, "in-order", "за пределами целого"},
		{"дробная вариация ниже границы", `{"value":1.0e-9223372036854775808}`, "in-order", "не целое"},
		{"огромная положительная экспонента", `{"value":1e500000000}`, "in-order", "за пределами целого"},
		{"огромная отрицательная экспонента", `{"value":1e-500000000}`, "in-order", "не целое"},
		{"слишком длинный литерал", `{"value":1000000000000000000000` + strings.Repeat("0", 5000) + `}`, "in-order", "слишком длинное"},
		// D3: точная проверка целых — длинная дробь не округляется до 1
		// (big.Float конечной точности принял бы её как целое).
		{"длинная дробь не целое", `{"value":1.0000000000000000000000000000000000000000000000001}`, "in-order", "не целое"},
		{"маленькая дробь не целое", `{"value":0.0000001}`, "in-order", "не целое"},
		// Компенсированная экспонента всё же обязана укладываться в int64:
		// одна значащая цифра с показателем 4096 после вычитания len(frac)
		// даёт 1e4096 — заведомо вне int64.
		{"компенсированная экспонента вне int64", `{"value":0.1e4097}`, "in-order", "за пределами целого"},
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

// TestTreeTraversalParityFixtures — общий fixture паритета клиент/сервер
// (visual/testdata/tree-input-parity.json): тот же набор кейсов, что
// гоняет тест публичного шва фронта (visual/frontend/tests/
// treeScenario.test.ts) через parseTreeInput. Оба шва обязаны приходить
// к одному вердикту: принимают — отдаёт SSE; отвергают — 400 с одной и
// той же сутью сообщения. Подробнее о fixture — в его поле "comment".
func TestTreeTraversalParityFixtures(t *testing.T) {
	raw, err := os.ReadFile("../testdata/tree-input-parity.json")
	if err != nil {
		t.Fatalf("не удалось прочитать общий fixture: %v", err)
	}
	var fixtures struct {
		Comment string `json:"comment"`
		Cases   []struct {
			Name        string `json:"name"`
			Tree        string `json:"tree"`
			Accept      bool   `json:"accept"`
			ClientError string `json:"clientError"`
			ServerError string `json:"serverError"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatalf("fixture не разбирается: %v", err)
	}
	if len(fixtures.Cases) == 0 {
		t.Fatalf("fixture пуст — ожидаются общие кейсы паритета")
	}
	for _, tc := range fixtures.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			resp := doRunTree(t, tc.Tree, "in-order")
			if tc.Accept {
				if resp.Code != http.StatusOK {
					t.Fatalf("статус = %d, ожидается 200: %s", resp.Code, resp.Body.String())
				}
				assertSSE(t, resp)
				return
			}
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("статус = %d, ожидается 400: %s", resp.Code, resp.Body.String())
			}
			if tc.ServerError != "" && !strings.Contains(resp.Body.String(), tc.ServerError) {
				t.Fatalf("ответ = %q, ожидается упоминание %q", resp.Body.String(), tc.ServerError)
			}
		})
	}
}

// TestTreeTraversalIntegerBoundariesSSE — целые числа на границах int64 и
// целые JSON-числа в ненатуральной записи (экспонента) проходят в Run по
// SSE: сервер проверяет литерал точно, без float64-округления. Значение
// проверяем по пояснению шага корня: к этом моменту снимок уже прошёл
// JSON-кругосветку, где int64 max/min теряют точность, а строка — нет.
func TestTreeTraversalIntegerBoundariesSSE(t *testing.T) {
	tests := []struct {
		name       string
		tree       string
		wantInText string
	}{
		{"int64 max", `{"value":9223372036854775807}`, "9223372036854775807"},
		{"int64 min", `{"value":-9223372036854775808}`, "-9223372036854775808"},
		{"экспонента, целое", `{"value":1e2}`, "100"},
		// D1: после дерева допускается трейлинг-пробел — конец ввода.
		{"пробел после дерева", `{"value":1}   `, "1"},
		// D3: целые в десятичной и экспоненциальной записи проходят
		// побайтово точно, без округления big.Float.
		{"десятичное целое 1.0", `{"value":1.0}`, "1"},
		{"десятичное целое 1.0e2", `{"value":1.0e2}`, "100"},
		{"минус нуль", `{"value":-0}`, "0"},
		{"экспонента 1e+2", `{"value":1e+2}`, "100"},
		// Нуль — нуль при любом показателе, даже за пределами int:
		// значение не материализуется и не требует арифметики экспоненты.
		{"нуль с огромным показателем", `{"value":0e99999999999999999999999999}`, "0"},
		// Компенсированная экспонента: дробь из 4089 цифр (4088 нулей и
		// единица) с показателем 4097 — это ровно 100000000 (1e8); литерал
		// ровно 4096 символов, без отсечке по грубой границе |exp| —
		// регрессия на ложный отказ по exp ≤ maxNumberLiteral.
		{"компенсированная экспонента", `{"value":0.` + strings.Repeat("0", 4088) + `1e4097}`, "100000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frames := doRunTreeSSE(t, tt.tree, "pre-order")
			data, ok := frames[1].Data.(map[string]any)
			if !ok {
				t.Fatalf("кадр шага 2: data не является снимком: %#v", frames[1].Data)
			}
			if activeID, _ := data["activeId"].(string); activeID != "root" {
				t.Fatalf("activeId = %q, ожидается root", activeID)
			}
			if !strings.Contains(frames[1].Explanation, tt.wantInText) {
				t.Fatalf("пояснение = %q, ожидается значение %q", frames[1].Explanation, tt.wantInText)
			}
		})
	}
}
