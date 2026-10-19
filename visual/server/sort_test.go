// Тесты шва 2 для bubble sort и selection sort (см. docs/SPEC.md): вход →
// SSE-поток → последовательность кадров. Покрывают emitter, Frame Renderer
// и HTTP-слой одной проверкой — и отказ на невалидном входе/лимите до
// начала SSE.
package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// doRunBubble — запрос Run bubble sort.
func doRunBubble(t *testing.T, values string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/bubble-sort", nil)
	q := req.URL.Query()
	q.Set("values", values)
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunBubbleSSE — успешный Run bubble sort: базовые проверки SSE-ответа
// + кадры.
func doRunBubbleSSE(t *testing.T, values string) []frame.Frame {
	t.Helper()

	resp := doRunBubble(t, values)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// assertSortSnapshot сверяет снимок сортировки в кадре (после
// JSON-кругосветки): содержимое массива, подсветки шага и имя операции.
func assertSortSnapshot(t *testing.T, f frame.Frame, values []int, highlights []render.SortHighlight, operation string) {
	t.Helper()

	if f.Kind != frame.KindSort {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindSort)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}

	gotValues := s["values"].([]any)
	if len(gotValues) != len(values) {
		t.Fatalf("кадр шага %d: значений %d, ожидалось %d: %v\nпояснение: %s",
			f.Step, len(gotValues), len(values), gotValues, f.Explanation)
	}
	for i, v := range values {
		if gotValues[i].(float64) != float64(v) {
			t.Fatalf("кадр шага %d: values[%d] = %v, ожидалось %d\nпояснение: %s",
				f.Step, i, gotValues[i], v, f.Explanation)
		}
	}

	gotHigh := s["highlights"].([]any)
	if len(gotHigh) != len(highlights) {
		t.Fatalf("кадр шага %d: подсветок %d, ожидалось %d (%v)", f.Step, len(gotHigh), len(highlights), gotHigh)
	}
	for i, want := range highlights {
		hl := gotHigh[i].(map[string]any)
		if hl["index"].(float64) != float64(want.Index) || hl["role"].(string) != want.Role {
			t.Fatalf("кадр шага %d: подсветка %d = %v, ожидается {index:%d role:%q}",
				f.Step, i, hl, want.Index, want.Role)
		}
	}

	if gotOp := s["operation"].(string); gotOp != operation {
		t.Fatalf("кадр шага %d: operation = %q, ожидалось %q\nпояснение: %s",
			f.Step, gotOp, operation, f.Explanation)
	}
}

// Сквозной сценарий теста: сортировка [3,1,2] — один стартовый кадр и
// по кадру на каждое событие (сравнение, обмен). Итог — упорядоченный
// массив; последний проход без обменов виден как сравнение с «обмена нет».
func TestRunBubbleSortScenario(t *testing.T) {
	frames := doRunBubbleSSE(t, "3,1,2")
	if len(frames) != 6 {
		t.Fatalf("кадров = %d, ожидалось 6:\n%v", len(frames), frames)
	}

	start := []int{3, 1, 2}
	afterSwap01 := []int{1, 3, 2}
	sorted := []int{1, 2, 3}
	pair := func(index int, role string) []render.SortHighlight {
		return []render.SortHighlight{
			{Index: index, Role: role},
			{Index: index + 1, Role: role},
		}
	}

	assertSortSnapshot(t, frames[0], start, nil, "") // старт: исходный массив
	assertSortSnapshot(t, frames[1], start, pair(0, render.RoleCompare), "сравнение 0 1")
	assertSortSnapshot(t, frames[2], afterSwap01, pair(0, render.RoleSwapped), "обмен 0 1")
	assertSortSnapshot(t, frames[3], afterSwap01, pair(1, render.RoleCompare), "сравнение 1 2")
	assertSortSnapshot(t, frames[4], sorted, pair(1, render.RoleSwapped), "обмен 1 2")
	// Второй проход: обменов нет — кадр сравнения хранит отсортированный массив.
	assertSortSnapshot(t, frames[5], sorted, pair(0, render.RoleCompare), "сравнение 0 1")

	// Пояснения отражают учебные точки: убывающую пару, сохранённый порядок
	// и итоговый массив.
	if !strings.Contains(frames[1].Explanation, "пара убывает") {
		t.Errorf("пояснение сравнения = %q, ожидается упоминание убывающей пары", frames[1].Explanation)
	}
	if !strings.Contains(frames[5].Explanation, "обмена нет") {
		t.Errorf("пояснение сравнения без обмена = %q, ожидается «обмена нет»", frames[5].Explanation)
	}
	if !strings.Contains(frames[4].Explanation, "поменялись местами") {
		t.Errorf("пояснение обмена = %q, ожидается упоминание обмена", frames[4].Explanation)
	}

	// Шаги нумеруются с 1 и идут по порядку.
	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидалось %d", i, f.Step, i+1)
		}
	}
}

// Пустой массив — легальный вход: SSE-поток состоит из одного стартового
// кадра с пустым рядом и завершается.
func TestRunBubbleSortEmptyInput(t *testing.T) {
	frames := doRunBubbleSSE(t, "")
	if len(frames) != 1 {
		t.Fatalf("кадров = %d, ожидался 1 (пустой массив сортировать не надо):\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{}, nil, "")

	// Одноэлементный вход: кадр исходного состояния без сравнений.
	frames = doRunBubbleSSE(t, "-7")
	if len(frames) != 1 {
		t.Fatalf("кадров = %d, ожидался 1 (один элемент отсортирован):\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{-7}, nil, "")
}

func TestRunBubbleSortValidation(t *testing.T) {
	tests := []struct {
		name   string
		values string
		want   string
	}{
		{"мусор в values", "1,x,3", "не удалось разобрать"},
		{"пустая компонента", "1,,3", "не удалось разобрать"},
		{"запятая в конце", "1,3,", "не удалось разобрать"},
		{"дробное значение", "1.5,2", "не удалось разобрать"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunBubble(t, tt.values)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("статус = %d, ожидается 400", resp.Code)
			}
			var got map[string]string
			if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
				t.Fatalf("тело ошибки не JSON: %v\n%s", err, resp.Body.String())
			}
			if !strings.Contains(got["error"], tt.want) {
				t.Errorf("ошибка = %q, ожидается упоминание %q", got["error"], tt.want)
			}
		})
	}
}

// Граница лимита длины: 100 элементов принимаются (SSE-поток завершён),
// 101-й элемент отвергается до начала потока.
func TestRunBubbleSortLengthLimit(t *testing.T) {
	input := func(n int) string {
		// Уже отсортированные данные: ровно n-1 сравнений, обменов нет —
		// поток короткий, а граница всё равно честная.
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf("%d", i)
		}
		return strings.Join(parts, ",")
	}

	frames := doRunBubbleSSE(t, input(100))
	if want := 100; len(frames) != want {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), want)
	}
	pair := func(index int, role string) []render.SortHighlight {
		return []render.SortHighlight{{Index: index, Role: role}, {Index: index + 1, Role: role}}
	}
	assertSortSnapshot(t, frames[len(frames)-1], mustInts(t, input(100)), pair(98, render.RoleCompare), "сравнение 98 99")

	resp := doRunBubble(t, input(101))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("статус = %d, ожидается 400: лимит превышен на 101-м элементе", resp.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело ошибки не JSON: %v\n%s", err, resp.Body.String())
	}
	if !strings.Contains(got["error"], "максимум 100") {
		t.Errorf("ошибка = %q, ожидается упоминание максимума 100", got["error"])
	}
}

// Отрицательные значения и дубликаты сортируются на месте и читаются
// в кадрах: вход уже упорядочен — два сравнения без обменов.
func TestRunBubbleSortNegativesAndDuplicates(t *testing.T) {
	frames := doRunBubbleSSE(t, "-1,-1,0")
	if len(frames) != 3 {
		t.Fatalf("кадров = %d, ожидалось 3:\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{-1, -1, 0}, nil, "")
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 1, Role: render.RoleCompare}, {Index: 2, Role: render.RoleCompare}},
		"сравнение 1 2")
	if !strings.Contains(last.Explanation, "порядок сохраняется") {
		t.Errorf("пояснение сравнения дубликатов = %q, ожидается сохранение порядка", last.Explanation)
	}
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта совпадают
// с кадрами, собранными напрямую у рендерера.
func TestRunBubbleSortMatchesDirectRenderer(t *testing.T) {
	frames := doRunBubbleSSE(t, "3,1,2")
	want := render.BubbleFrames([]int{3, 1, 2})

	if len(frames) != len(want) {
		t.Fatalf("кадров = %d, рендерер даёт %d", len(frames), len(want))
	}
	for i := range want {
		// Обе стороны прогоняем через JSON-кругосветку: снимок из потока
		// приходит как map, порядок ключей не должен влиять на сравнение.
		aw, _ := json.Marshal(toWire(t, frames[i]))
		bw, _ := json.Marshal(toWire(t, want[i]))
		if !bytes.Equal(aw, bw) {
			t.Errorf("кадр %d отличается:\nэндпоинт %s\nрендерер %s", i, aw, bw)
		}
	}
}

// mustInts — разбор «a,b,c» с гарантией: вызывается на валидированных входах.
func mustInts(t *testing.T, raw string) []int {
	t.Helper()
	parts := strings.Split(raw, ",")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			t.Fatalf("mustInts: %q: %v", p, err)
		}
		out = append(out, v)
	}
	return out
}

// doRunSelection — запрос Run selection sort.
func doRunSelection(t *testing.T, values string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/selection-sort", nil)
	q := req.URL.Query()
	q.Set("values", values)
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunSelectionSSE — успешный Run selection sort: базовые проверки
// SSE-ответа + кадры.
func doRunSelectionSSE(t *testing.T, values string) []frame.Frame {
	t.Helper()

	resp := doRunSelection(t, values)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// Сквозной сценарий теста: сортировка [3,1,2] — один стартовый кадр и
// по кадру на каждое событие (сравнение кандидата с текущим минимумом,
// финальный обмен прохода). Кадр сравнения подсвечивает минимум и
// кандидата разными ролями; итог — упорядоченный массив.
func TestRunSelectionSortScenario(t *testing.T) {
	frames := doRunSelectionSSE(t, "3,1,2")
	if len(frames) != 6 {
		t.Fatalf("кадров = %d, ожидалось 6:\n%v", len(frames), frames)
	}

	start := []int{3, 1, 2}
	afterSwap01 := []int{1, 3, 2}
	sorted := []int{1, 2, 3}
	// Сравнение без смены минимума: минимум остаётся на прежней позиции.
	compareKeep := func(min, scanned int) []render.SortHighlight {
		return []render.SortHighlight{
			{Index: min, Role: render.RoleMin},
			{Index: scanned, Role: render.RoleCompare},
		}
	}
	// Сравнение со сменой минимума: кадр показывает новый минимум
	// на проверяемой позиции, прежний — участником сравнения.
	compareMove := func(oldMin, newMin int) []render.SortHighlight {
		return []render.SortHighlight{
			{Index: oldMin, Role: render.RoleCompare},
			{Index: newMin, Role: render.RoleMin},
		}
	}
	swapPair := func(a, b int) []render.SortHighlight {
		return []render.SortHighlight{
			{Index: a, Role: render.RoleSwapped},
			{Index: b, Role: render.RoleSwapped},
		}
	}

	assertSortSnapshot(t, frames[0], start, nil, "") // старт: исходный массив
	// Проход 0: кандидат 1 меньше минимума 3 — минимум переходит на позицию 1.
	assertSortSnapshot(t, frames[1], start, compareMove(0, 1), "сравнение 0 1")
	// Кандидат 2 минимум 1 не смещает.
	assertSortSnapshot(t, frames[2], start, compareKeep(1, 2), "сравнение 1 2")
	assertSortSnapshot(t, frames[3], afterSwap01, swapPair(0, 1), "обмен 0 1")
	// Проход 1: кандидат 2 меньше минимума 3 — минимум на позиции 2.
	assertSortSnapshot(t, frames[4], afterSwap01, compareMove(1, 2), "сравнение 1 2")
	assertSortSnapshot(t, frames[5], sorted, swapPair(1, 2), "обмен 1 2")

	// Пояснения отражают учебные точки: смену минимума, сохранение
	// минимума и финальный обмен прохода.
	if !strings.Contains(frames[1].Explanation, "минимум переходит на позицию 1") {
		t.Errorf("пояснение смены минимума = %q, ожидается переход минимума", frames[1].Explanation)
	}
	if !strings.Contains(frames[2].Explanation, "минимум остаётся на позиции 1") {
		t.Errorf("пояснение сравнения без смены = %q, ожидается сохранение минимума", frames[2].Explanation)
	}
	if !strings.Contains(frames[3].Explanation, "поменялись местами") {
		t.Errorf("пояснение обмена = %q, ожидается упоминание обмена", frames[3].Explanation)
	}

	// Шаги нумеруются с 1 и идут по порядку.
	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидалось %d", i, f.Step, i+1)
		}
	}
}

// Регрессия учебной точки сравнения: когда кандидат становится новым
// минимумом, кадр обязан показывать роль "min" на его позиции — тот
// минимум, о котором говорит пояснение, — а прежний минимум — участником
// сравнения ("compare"), а не наоборот.
func TestRunSelectionSortCompareFrameShowsNewMinimum(t *testing.T) {
	frames := doRunSelectionSSE(t, "5,2")
	if len(frames) != 3 {
		t.Fatalf("кадров = %d, ожидалось 3:\n%v", len(frames), frames)
	}

	// Кандидат 2 на позиции 1 меньше минимума 5: минимум теперь на позиции 1.
	assertSortSnapshot(t, frames[1], []int{5, 2},
		[]render.SortHighlight{
			{Index: 0, Role: render.RoleCompare},
			{Index: 1, Role: render.RoleMin},
		},
		"сравнение 0 1")
	if !strings.Contains(frames[1].Explanation, "минимум переходит на позицию 1") {
		t.Errorf("пояснение смены минимума = %q, ожидается переход минимума", frames[1].Explanation)
	}

	// Финальный обмен перемещает новый минимум на первую позицию прохода.
	assertSortSnapshot(t, frames[2], []int{2, 5},
		[]render.SortHighlight{
			{Index: 0, Role: render.RoleSwapped},
			{Index: 1, Role: render.RoleSwapped},
		},
		"обмен 0 1")
}

// Пустой массив — легальный вход: SSE-поток состоит из одного стартового
// кадра с пустым рядом и завершается.
func TestRunSelectionSortEmptyInput(t *testing.T) {
	frames := doRunSelectionSSE(t, "")
	if len(frames) != 1 {
		t.Fatalf("кадров = %d, ожидался 1 (пустой массив сортировать не надо):\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{}, nil, "")

	// Одноэлементный вход: кадр исходного состояния без сравнений.
	frames = doRunSelectionSSE(t, "-7")
	if len(frames) != 1 {
		t.Fatalf("кадров = %d, ожидался 1 (один элемент отсортирован):\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{-7}, nil, "")
}

func TestRunSelectionSortValidation(t *testing.T) {
	tests := []struct {
		name   string
		values string
		want   string
	}{
		{"мусор в values", "1,x,3", "не удалось разобрать"},
		{"пустая компонента", "1,,3", "не удалось разобрать"},
		{"запятая в конце", "1,3,", "не удалось разобрать"},
		{"дробное значение", "1.5,2", "не удалось разобрать"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunSelection(t, tt.values)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("статус = %d, ожидается 400", resp.Code)
			}
			var got map[string]string
			if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
				t.Fatalf("тело ошибки не JSON: %v\n%s", err, resp.Body.String())
			}
			if !strings.Contains(got["error"], tt.want) {
				t.Errorf("ошибка = %q, ожидается упоминание %q", got["error"], tt.want)
			}
		})
	}
}

// Граница лимита длины: 100 элементов принимаются (SSE-поток завершён),
// 101-й элемент отвергается до начала потока.
func TestRunSelectionSortLengthLimit(t *testing.T) {
	input := func(n int) string {
		// Уже отсортированные данные: selection sort всегда досматривает
		// остаток целиком — граница честная, поток конечный.
		parts := make([]string, n)
		for i := range parts {
			parts[i] = fmt.Sprintf("%d", i)
		}
		return strings.Join(parts, ",")
	}

	frames := doRunSelectionSSE(t, input(100))
	// На каждом проходе i: (n-1-i) сравнений, обменов нет —
	// 1 стартовый кадр + 99+98+...+1 = 4950 кадров сравнений.
	if want := 1 + 99*100/2; len(frames) != want {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), want)
	}
	assertSortSnapshot(t, frames[len(frames)-1], mustInts(t, input(100)),
		[]render.SortHighlight{{Index: 98, Role: render.RoleMin}, {Index: 99, Role: render.RoleCompare}},
		"сравнение 98 99")

	resp := doRunSelection(t, input(101))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("статус = %d, ожидается 400: лимит превышен на 101-м элементе", resp.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело ошибки не JSON: %v\n%s", err, resp.Body.String())
	}
	if !strings.Contains(got["error"], "максимум 100") {
		t.Errorf("ошибка = %q, ожидается упоминание максимума 100", got["error"])
	}
}

// Отрицательные значения и дубликаты сортируются на месте и читаются
// в кадрах: вход уже упорядочен — сравнения без смены минимума и обменов.
func TestRunSelectionSortNegativesAndDuplicates(t *testing.T) {
	frames := doRunSelectionSSE(t, "-1,-1,0")
	if len(frames) != 4 {
		t.Fatalf("кадров = %d, ожидалось 4:\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{-1, -1, 0}, nil, "")
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 1, Role: render.RoleMin}, {Index: 2, Role: render.RoleCompare}},
		"сравнение 1 2")
	if !strings.Contains(last.Explanation, "минимум остаётся на позиции 1") {
		t.Errorf("пояснение сравнения дубликатов = %q, ожидается сохранение минимума", last.Explanation)
	}
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта совпадают
// с кадрами, собранными напрямую у рендерера.
func TestRunSelectionSortMatchesDirectRenderer(t *testing.T) {
	frames := doRunSelectionSSE(t, "3,1,2")
	want := render.SelectionFrames([]int{3, 1, 2})

	if len(frames) != len(want) {
		t.Fatalf("кадров = %d, рендерер даёт %d", len(frames), len(want))
	}
	for i := range want {
		// Обе стороны прогоняем через JSON-кругосветку: снимок из потока
		// приходит как map, порядок ключей не должен влиять на сравнение.
		aw, _ := json.Marshal(toWire(t, frames[i]))
		bw, _ := json.Marshal(toWire(t, want[i]))
		if !bytes.Equal(aw, bw) {
			t.Errorf("кадр %d отличается:\nэндпоинт %s\nрендерер %s", i, aw, bw)
		}
	}
}
