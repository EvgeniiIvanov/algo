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
	"strconv"
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
// treeScenario.test.ts) через parseTreeInput. Fixture фиксирует один и
// тот же вердикт accept/reject у обоих швов: принимают — отдаёт SSE;
// отвергают — 400. Текущие отвергающие кейсы fixture несут один дефект
// за раз, и их подстроки/префиксы ошибок clientError/serverError
// совпадают; при этом полные тексты диагностик швов не обязаны
// совпадать: Go добавляет первопричину json.Unmarshal для не-null
// неверного типа поля, а порядок первых диагностик при нескольких
// одновременных дефектах не гарантируется — фиксируется сам отказ.
// Подробнее о fixture — в его поле "comment".
func TestTreeTraversalParityFixtures(t *testing.T) {
	raw, err := os.ReadFile("../testdata/tree-input-parity.json")
	if err != nil {
		t.Fatalf("не удалось прочитать общий fixture: %v", err)
	}
	var fixtures struct {
		Comment string              `json:"comment"`
		Cases   []parityFixtureCase `json:"cases"`
	}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatalf("fixture не разбирается: %v", err)
	}
	if len(fixtures.Cases) == 0 {
		t.Fatalf("fixture пуст — ожидаются общие кейсы паритета")
	}
	for _, tc := range fixtures.Cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Accept == nil {
				t.Fatalf("в кейсе fixture отсутствует поле accept")
			}
			wantPath, wantValue, assertNode, err := validateEffectiveNodeContract(
				tc.Name, *tc.Accept, tc.AssertEffectiveNode, tc.ExpectedNode,
			)
			if err != nil {
				t.Fatalf("контракт fixture нарушен: %v", err)
			}
			if !*tc.Accept {
				resp := doRunTree(t, tc.Tree, "in-order")
				if resp.Code != http.StatusBadRequest {
					t.Fatalf("статус = %d, ожидается 400: %s", resp.Code, resp.Body.String())
				}
				// Ожидание ошибки обязано быть заполнено в fixture, иначе кейс
				// ничего не проверяет; сравниваем с обязательной сутью сообщения.
				if tc.ServerError == "" {
					t.Fatalf("у отвергающего кейса пустое ожидание ошибки сервера в fixture")
				}
				if got := resp.Body.String(); !strings.Contains(got, tc.ServerError) {
					t.Fatalf("тело ошибки = %s, ожидается упоминание %q", got, tc.ServerError)
				}
				return
			}
			frames := doRunTreeSSE(t, tc.Tree, "in-order")
			if !assertNode {
				return
			}
			// Стартовый кадр — полный снимок топологии: проверяем
			// фактический результат разбора, а не только вердикт.
			assertNodeInSnapshot(t, frames[0], wantPath, wantValue)
		})
	}
}

// validateEffectiveNodeContract — рантайм-охрана coherent-контракта
// маркера assertEffectiveNode и ожидания expectedNode в кейсе fixture:
//   - маркер, если присутствует, обязан быть булевым;
//   - маркер true требует принимающего кейса с корректным
//     expectedNode (путь — точный идентификатор root или непустая
//     строка из символов L/R, без обрезки пробелов; значение —
//     знаковое десятичное целое в строке в пределах |v| ≤ 2^53−1);
//   - expectedNode без assertEffectiveNode: true — ошибка fixture;
//   - отвергающий кейс не несёт ни маркера, ни ожидания;
//   - обычные принимающие кейсы не маркируются и expectedNode не несут;
//   - отсутствие поля и явный JSON null различаются: null отклоняется
//     как некорректное значение и для маркера, и для ожидания (как в
//     TS-валидаторе, где typeof null !== "boolean" и node === null).
//
// Проверка общая с TS-консьюмером fixture; ожидание передаётся сырым
// JSON-полем (presence-aware: nil/пусто — поле отсутствует) и
// возвращает путь и значение ожидания и признак назначенной проверки
// снимка.
func validateEffectiveNodeContract(caseName string, accept bool, markerRaw json.RawMessage, expectRaw json.RawMessage) (string, int64, bool, error) {
	assert := false
	if len(markerRaw) > 0 {
		if strings.TrimSpace(string(markerRaw)) == "null" {
			return "", 0, false, fmt.Errorf("кейс «%s»: маркер assertEffectiveNode не булево (%s)", caseName, markerRaw)
		}
		var marker bool
		if err := json.Unmarshal(markerRaw, &marker); err != nil {
			return "", 0, false, fmt.Errorf("кейс «%s»: маркер assertEffectiveNode не булево (%s)", caseName, markerRaw)
		}
		assert = marker
	}
	expectPresent := len(strings.TrimSpace(string(expectRaw))) > 0
	if !accept {
		if len(markerRaw) > 0 {
			return "", 0, false, fmt.Errorf("кейс «%s»: отвергающий кейс не должен нести маркер assertEffectiveNode", caseName)
		}
		if expectPresent {
			return "", 0, false, fmt.Errorf("кейс «%s»: отвергающий кейс не должен нести expectedNode", caseName)
		}
		return "", 0, false, nil
	}
	if expectPresent && !assert {
		return "", 0, false, fmt.Errorf("кейс «%s»: expectedNode без assertEffectiveNode: true — ошибка fixture", caseName)
	}
	if assert && !expectPresent {
		return "", 0, false, fmt.Errorf("кейс «%s»: у кейса с assertEffectiveNode отсутствует expectedNode", caseName)
	}
	if !assert {
		return "", 0, false, nil
	}
	expect, err := decodeExpectedNode(expectRaw)
	if err != nil {
		return "", 0, false, fmt.Errorf("кейс «%s»: %w", caseName, err)
	}
	if expect.null {
		return "", 0, false, fmt.Errorf("кейс «%s»: expectedNode — null вместо объекта ожидания", caseName)
	}
	path := expect.node.Path
	if !isValidNodePath(path) {
		return "", 0, false, fmt.Errorf("кейс «%s»: путь в expectedNode — не «root» и не путь из символов L/R (без пробелов и прочих символов): %q", caseName, path)
	}
	if !isSignedDecimalInteger(expect.node.Value) {
		return "", 0, false, fmt.Errorf("кейс «%s»: value в expectedNode не знаковое десятичное целое в строке: %q", caseName, expect.node.Value)
	}
	v, err := strconv.ParseInt(expect.node.Value, 10, 64)
	if err != nil || v > maxSafeInteger || v < -maxSafeInteger {
		return "", 0, false, fmt.Errorf("кейс «%s»: value в expectedNode вне безопасного диапазона (|v| ≤ 2^53−1): %q", caseName, expect.node.Value)
	}
	return path, v, true, nil
}

// parityExpectedNode — тестовое ожидание эффективного узла снимка
// (метаданные fixture, не вход дерева). Value хранится строкой со
// знаковым десятичным целым: это исключает лексическое расхождение
// Go json.Number.Int64() и TS Number.isInteger() (1e3, 1.0 и т.п.).
// Ключи path/value — точные lowercase: decodeExpectedNode разбирает
// ожидание case-sensitive (struct-теги json дают нечувствительность к
// регистру).
type parityExpectedNode struct {
	Path  string `json:"path"`
	Value string `json:"value"`
}

// parityExpectation — результат разбора сырого поля expectedNode:
// отсутствие поля (nil из decodeExpectedNode) и явный JSON null
// (нен-nil, флаг null) — разные ситуации. null не материализуется как
// валидное ожидание.
type parityExpectation struct {
	null bool
	node parityExpectedNode
}

// decodeExpectedNode — presence-aware разбор сырого поля expectedNode:
// nil/пусто — поле отсутствует (nil), обрезанный null — явный JSON
// null (флаг null), иначе — объект ожидания. Явно недопустимые формы
// (строка, массив) дают ошибку разбора с конкретной формой. Ключи
// разбираются точно (case-sensitive): читаются только точные lowercase
// path и value (map[string]json.RawMessage, без struct-тегов — те
// нечувствительны к регистру); посторонние ключи любого JSON-типа
// игнорируются, как в TS-консьюмере, а присутствующим path/value
// требуется строковое значение (ключи вроде Path не читаются вовсе —
// их проверка выполняет валидатор на пустом поле).
// Присутствие и null различаются здесь, а не через nil-указатель:
// декодирование выполняет валидатор на уже разобранном fixture-кейсе.
func decodeExpectedNode(raw json.RawMessage) (*parityExpectation, error) {
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 {
		return nil, nil
	}
	if trimmed == "null" {
		return &parityExpectation{null: true}, nil
	}
	if strings.HasPrefix(trimmed, "\"") {
		return nil, fmt.Errorf("expectedNode — строка вместо объекта ожидания")
	}
	if strings.HasPrefix(trimmed, "[") {
		return nil, fmt.Errorf("expectedNode — массив вместо объекта ожидания")
	}
	var node map[string]json.RawMessage
	if err := json.Unmarshal(raw, &node); err != nil {
		return nil, fmt.Errorf("expectedNode — некорректный JSON в объекте ожидания: %w", err)
	}
	var expect parityExpectation
	// Точные lowercase-ключи: map-lookup case-sensitive (struct-теги
	// json — нет, поэтому map). Ключи отсутствуют — поля остаются
	// пустыми, их отвергнет валидатор; присутствующий ключ обязан быть
	// JSON-строкой (null не засчитывается строкой лишь у path/value —
	// посторонние ключи с null допустимы, они игнорируются).
	if err := expectStringField(node, "path", &expect.node.Path); err != nil {
		return nil, err
	}
	if err := expectStringField(node, "value", &expect.node.Value); err != nil {
		return nil, err
	}
	return &expect, nil
}

// expectStringField — точный lowercase-lookup ключа в разобранном
// объекте ожидания: отсутствие ключа оставляет поле пустым, присутствующий
// ключ обязан быть JSON-строкой (ошибка типа поля, без анализа
// содержимого — до грамматики path/value; для одиночного дефекта поля
// стабильный префикс диагностики — имя поля и «не строка» — выровнен
// с TS-консьюмером fixture, но полные тексты не обязаны совпадать:
// Go оборачивает первопричину json.Unmarshal; при нескольких
// некорректных полях порядок первой диагностики не фиксируется).
// Массив, число, объект, булево отвергает json.Unmarshal; явный JSON null
// приходится проверять по сырому литералу: json.Unmarshal(null, &string)
// оставил бы поле пустым без ошибки.
func expectStringField(node map[string]json.RawMessage, key string, dst *string) error {
	raw, ok := node[key]
	if !ok {
		return nil
	}
	if strings.TrimSpace(string(raw)) == "null" {
		// Явный raw null — собственная ошибка типа поля: json.Unmarshal
		// (null, &string) не даёт ни ошибки, ни записи поля. Сообщение —
		// то же слово «не строка», что и у прочих не-строк.
		return fmt.Errorf("expectedNode — %s в объекте ожидания не строка", key)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		// Сохраняем первопричину: несоответствие типа поля выдаёт
		// json.Unmarshal («cannot unmarshal … into Go value of type string»).
		return fmt.Errorf("expectedNode — %s в объекте ожидания не строка: %w", key, err)
	}
	return nil
}

// isValidNodePath — грамматика пути expectedNode: точный идентификатор
// root или непустая строка из символов L/R. Без обрезки пробелов:
// дополненный пробелами, BOM или иными символами путь отвергается
// (каждый байт обязан быть 'L' или 'R'). Та же грамматика проверяется
// в TS-консьюмере fixture.
func isValidNodePath(path string) bool {
	if path == "" {
		return false
	}
	if path == "root" {
		return true
	}
	for i := 0; i < len(path); i++ {
		if path[i] != 'L' && path[i] != 'R' {
			return false
		}
	}
	return true
}

// parityFixtureCase — прозрачная копия структуры кейса fixture
// (visual/testdata/tree-input-parity.json), чтобы оба консьюмера
// читали одни и те же поля.
type parityFixtureCase struct {
	Name   string `json:"name"`
	Tree   string `json:"tree"`
	Accept *bool  `json:"accept"` // указатель: отсутствие поля — ошибка fixture, а не «отвергается»
	// AssertEffectiveNode — маркер назначенной проверки снимка.
	AssertEffectiveNode json.RawMessage `json:"assertEffectiveNode"`
	ClientError         string          `json:"clientError"`
	ServerError         string          `json:"serverError"`
	// ExpectedNode — сырой JSON ожидания (presence-aware): nil/пусто —
	// поле отсутствует, обрезанное null — явный JSON null, иначе —
	// объект ожидания; отсутствие поля и null различаются.
	ExpectedNode json.RawMessage `json:"expectedNode"`
}

// maxSafeInteger — граница безопасного целого TS (2^53−1): тестовое
// значение expectedNode обязано в неё укладываться, чтобы оба
// консьюмера сравнивали одно и то же число.
const maxSafeInteger = 1<<53 - 1

// isSignedDecimalInteger — общая грамматика знакового десятичного
// целого для expectedNode.value: необязательный минус, затем 0 или
// ненулевая цифра с любым хвостом десятичных цифр (ведущие нули и
// лексические формы вроде 1e3 или 1.0 не допускаются).
// Та же грамматика проверяется в TS-консьюмере fixture.
func isSignedDecimalInteger(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '-' {
		s = s[1:]
	}
	if s == "0" {
		return true
	}
	if s == "" || s[0] == '0' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// TestValidateEffectiveNodeContract — рантайм-охрана coherent-контракта
// маркера/ожидания: таблица нарушений и корректных форм кейса fixture.
func TestValidateEffectiveNodeContract(t *testing.T) {
	tests := []struct {
		name       string
		accept     bool
		markerRaw  string // "" — маркер отсутствует
		expect     *parityExpectedNode
		expectRaw  string // "" — ожидание отсутствует или задаётся expect; сырая JSON-форма для кейсов формы/регистра ключей
		expectNull bool   // true — в кейсе явный expectedNode: null
		wantPath   string
		wantValue  int64
		wantAssert bool
		wantErr    string // "" — ошибки нет
	}{
		{
			name:      "обычный принимающий кейс — без маркера и ожидания",
			accept:    true,
			wantPath:  "",
			wantValue: 0,
		},
		{
			name:       "назначенный кейс — маркер true и корректное ожидание",
			accept:     true,
			markerRaw:  "true",
			expect:     &parityExpectedNode{Path: "L", Value: "3"},
			wantPath:   "L",
			wantValue:  3,
			wantAssert: true,
		},
		{
			name:       "value 0 — корректное значение",
			accept:     true,
			markerRaw:  "true",
			expect:     &parityExpectedNode{Path: "L", Value: "0"},
			wantPath:   "L",
			wantValue:  0,
			wantAssert: true,
		},
		{
			name:       "value -0 — корректное значение",
			accept:     true,
			markerRaw:  "true",
			expect:     &parityExpectedNode{Path: "L", Value: "-0"},
			wantPath:   "L",
			wantValue:  0,
			wantAssert: true,
		},
		{
			name:       "путь из символов L/R — корректный путь",
			accept:     true,
			markerRaw:  "true",
			expect:     &parityExpectedNode{Path: "LRRL", Value: "-7"},
			wantPath:   "LRRL",
			wantValue:  -7,
			wantAssert: true,
		},
		{
			name:      "путь, дополненный пробелами, отвергается без обрезки",
			accept:    true,
			markerRaw: "true",
			expect:    &parityExpectedNode{Path: " root ", Value: "-7"},
			wantErr:   "путь в expectedNode",
		},
		{
			name:      "путь с BOM отвергается",
			accept:    true,
			markerRaw: "true",
			expect:    &parityExpectedNode{Path: "\uFEFFL", Value: "-7"},
			wantErr:   "путь в expectedNode",
		},
		{
			name:      "отвергающий кейс не несёт маркер",
			accept:    false,
			markerRaw: "true",
			wantErr:   "не должен нести маркер",
		},
		{
			name:      "отвергающий кейс не несёт ожидание",
			accept:    false,
			expect:    &parityExpectedNode{Path: "L", Value: "3"},
			wantPath:  "",
			wantValue: 0,
			wantErr:   "не должен нести expectedNode",
		},
		{
			name:       "отвергающий кейс несёт null вместо expectedNode",
			accept:     false,
			expectNull: true,
			wantErr:    "не должен нести expectedNode",
		},
		{
			name:      "маркер не булево",
			accept:    true,
			markerRaw: `"true"`,
			wantErr:   "маркер assertEffectiveNode не булево",
		},
		{
			name:      "маркер null не булево",
			accept:    true,
			markerRaw: "null",
			wantErr:   "маркер assertEffectiveNode не булево",
		},
		{
			name:      "маркер true без expectedNode",
			accept:    true,
			markerRaw: "true",
			wantErr:   "отсутствует expectedNode",
		},
		{
			name:    "expectedNode без маркера true",
			accept:  true,
			expect:  &parityExpectedNode{Path: "L", Value: "3"},
			wantErr: "expectedNode без assertEffectiveNode: true",
		},
		{
			name:       "null вместо expectedNode без маркера true",
			accept:     true,
			expectNull: true,
			wantErr:    "expectedNode без assertEffectiveNode: true",
		},
		{
			name:      "маркер false с expectedNode",
			accept:    true,
			markerRaw: "false",
			expect:    &parityExpectedNode{Path: "L", Value: "3"},
			wantErr:   "expectedNode без assertEffectiveNode: true",
		},
		{
			name:       "null вместо объекта ожидания",
			accept:     true,
			markerRaw:  "true",
			expectNull: true,
			wantErr:    "null вместо объекта ожидания",
		},
		{
			name:      "пустой путь в ожидании отвергается",
			accept:    true,
			markerRaw: "true",
			expect:    &parityExpectedNode{Path: "   ", Value: "3"},
			wantErr:   "путь в expectedNode",
		},
		{
			name:      "value не знаковое десятичное целое",
			accept:    true,
			markerRaw: "true",
			expect:    &parityExpectedNode{Path: "L", Value: "1.0"},
			wantErr:   "не знаковое десятичное целое",
		},
		{
			name:      "value вне безопасного диапазона",
			accept:    true,
			markerRaw: "true",
			expect:    &parityExpectedNode{Path: "L", Value: "9007199254740993"},
			wantErr:   "вне безопасного диапазона",
		},
		{
			name:      "expectedNode — строка вместо объекта ожидания",
			accept:    true,
			markerRaw: "true",
			expectRaw: `"L"`,
			wantErr:   "строка вместо объекта ожидания",
		},
		{
			name:      "expectedNode — массив вместо объекта ожидания",
			accept:    true,
			markerRaw: "true",
			expectRaw: `["L","3"]`,
			wantErr:   "массив вместо объекта ожидания",
		},
		{
			name:      "ключ Path (не lowercase) отвергается",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"Path":"L","value":"3"}`,
			wantErr:   "путь в expectedNode",
		},
		{
			name:      "ключ Value (не lowercase) отвергается",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"path":"L","Value":"3"}`,
			wantErr:   "не знаковое десятичное целое",
		},
		{
			// Некорректный тип поля ожидания: поле присутствует, но это
			// не JSON-строка — сырая форма (массив, число) отвергается
			// expectStringField как ошибка типа поля, до грамматики.
			// Не-строки здесь работают по-разному у швов, и это не баг:
			// в Go коэрции нет — encoding/json сам отвергает массив при
			// декодировании в string (скрытая коэрция строк в Go-тесте
			// невозможна), поэтому Go-кейс фиксирует проверку типа
			// сырого JSON-поля. TS-кейс, наоборот, дискриминирующий
			// именно из-за коэрции JS: ["L"] приводится к строке "L";
			// если бы typeof-guard был снят, кейс «прошёл» бы грамматику
			// пути. Зеркалит TS-кейс validateParityFixture.
			name:      "path в expectedNode — массив вместо строки",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"path":["L"],"value":"3"}`,
			wantErr:   "path в объекте ожидания не строка",
		},
		{
			name:      "path в expectedNode — число вместо строки",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"path":3,"value":"3"}`,
			wantErr:   "path в объекте ожидания не строка",
		},
		{
			// Явный JSON null в path при json.Unmarshal(null, &string) не
			// даёт ни записи поля, ни ошибки — expectStringField отсекает
			// null по сырому литералу как ошибку типа поля, так что null
			// отвергается с той же диагностикой, что и прочие не-строки
			// (в Go null никуда не приводится — проверка именно сырая).
			name:      "path в expectedNode — null вместо строки",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"path":null,"value":"3"}`,
			wantErr:   "path в объекте ожидания не строка",
		},
		{
			name:      "value в expectedNode — число вместо строки",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"path":"L","value":3}`,
			wantErr:   "value в объекте ожидания не строка",
		},
		{
			// В Go коэрции массива в число/строку нет: encoding/json
			// отвергает ["3"] при декодировании в string как ошибку типа
			// поля, до грамматики value — тест фиксирует проверку типа
			// сырого JSON-поля, а не случайную форму значения.
			// Дискриминирующий подбор ["3"] (коэрция TS: и String(["3"]),
			// и Number(["3"]) прошли бы грамматику, если бы
			// typeof-guard был снят) относится к TS-кейсу
			// validateParityFixture.
			name:      "value в expectedNode — массив вместо строки",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"path":"L","value":["3"]}`,
			wantErr:   "value в объекте ожидания не строка",
		},
		{
			// Явный JSON null в value тоже отсекается expectStringField
			// по сырому литералу как ошибка типа поля — до грамматики
			// value: json.Unmarshal(null, &string) оставил бы пустое поле
			// без ошибки (в Go null никуда не приводится — проверка
			// именно сырая).
			name:      "value в expectedNode — null вместо строки",
			accept:    true,
			markerRaw: "true",
			expectRaw: `{"path":"L","value":null}`,
			wantErr:   "value в объекте ожидания не строка",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Кейс собирается как настоящий JSON и разбирается тем же
			// декодером, что и fixture-файл: присутствие/отсутствие
			// expectedNode и явный null проходят через json.Unmarshal
			// в parityFixtureCase.ExpectedNode (json.RawMessage), а не
			// через ручную конструкцию обёртки.
			cj := map[string]any{
				"name":        tc.name,
				"tree":        `{"value":1}`,
				"accept":      tc.accept,
				"clientError": "не целое",
			}
			if tc.markerRaw != "" {
				cj["assertEffectiveNode"] = json.RawMessage(tc.markerRaw)
			}
			if raw := expectJSONRaw(tc.expect, tc.expectNull); raw != "" {
				cj["expectedNode"] = json.RawMessage(raw)
			}
			if tc.expectRaw != "" {
				cj["expectedNode"] = json.RawMessage(tc.expectRaw)
			}
			data, err := json.Marshal(cj)
			if err != nil {
				t.Fatalf("кейс fixture не собирается: %v", err)
			}
			var c parityFixtureCase
			if err := json.Unmarshal(data, &c); err != nil {
				t.Fatalf("кейс fixture не разбирается: %v", err)
			}
			if c.Accept == nil {
				t.Fatalf("в кейсе fixture отсутствует поле accept")
			}
			path, value, assert, err := validateEffectiveNodeContract(tc.name, *c.Accept, c.AssertEffectiveNode, c.ExpectedNode)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("ошибка = %v, ожидается упоминание %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if assert != tc.wantAssert {
				t.Fatalf("признак назначенной проверки = %v, ожидается %v", assert, tc.wantAssert)
			}
			if path != tc.wantPath {
				t.Fatalf("путь = %q, ожидается %q", path, tc.wantPath)
			}
			if value != tc.wantValue {
				t.Fatalf("значение = %d, ожидается %d", value, tc.wantValue)
			}
		})
	}
}

// expectJSONRaw — JSON-представление поля expectedNode для кейса
// таблицы: nil/false — поле отсутствует, "null" — явный null, иначе —
// объект ожидания.
func expectJSONRaw(expect *parityExpectedNode, expectNull bool) string {
	switch {
	case expectNull:
		return "null"
	case expect != nil:
		data, err := json.Marshal(expect)
		if err != nil {
			panic(err)
		}
		return string(data)
	default:
		return ""
	}
}

// TestIsSignedDecimalInteger — грамматика тестового значения
// expectedNode.value, общая с TS-консьюмером fixture.
func TestIsSignedDecimalInteger(t *testing.T) {
	valid := []string{"0", "-0", "3", "-7", "123", "-123", "9007199254740991"}
	invalid := []string{"", "-", "1.0", "1e3", "+3", "01", "-01", " 3", "3 ", "3. "}
	for _, s := range valid {
		if !isSignedDecimalInteger(s) {
			t.Fatalf("%q должно быть корректным знаковым десятичным целым", s)
		}
	}
	for _, s := range invalid {
		if isSignedDecimalInteger(s) {
			t.Fatalf("%q не должно быть корректным знаковым десятичным целым", s)
		}
	}
}

// от корня (id) и ровно данным значением.
func assertNodeInSnapshot(t *testing.T, f frame.Frame, path string, value int64) {
	t.Helper()

	data, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок — map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	nodes := mustSliceAny(t, data, "nodes")
	for _, node := range nodes {
		nodeData, ok := node.(map[string]any)
		if !ok {
			t.Fatalf("кадр шага %d: элемент nodes не объект: %#v", f.Step, node)
		}
		id, _ := nodeData["id"].(string)
		if id != path {
			continue
		}
		num, ok := nodeData["value"].(float64)
		if !ok {
			t.Fatalf("узел пути %q: value не число: %#v", path, nodeData["value"])
		}
		if int64(num) != value || num != float64(int64(num)) {
			t.Fatalf("узел пути %q: value = %v, ожидается %d", path, nodeData["value"], value)
		}
		return
	}
	t.Fatalf("узла пути %q нет в снимке кадра шага %d: %d узлов", path, f.Step, len(nodes))
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
