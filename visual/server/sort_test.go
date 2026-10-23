// Тесты шва 2 для простых сортировок (см. docs/SPEC.md): вход →
// SSE-поток → последовательность кадров. Покрывают emitter, Frame Renderer
// и HTTP-слой одной проверкой — и отказ на невалидном входе/лимите до
// начала SSE. Общие каркасы (запрос, валидация входа, тривиальные входы,
// сверка с рендерером) вынесены в хелперы ниже; сценарии кадров каждой
// сортировки — свои, их покрытие не ослаблено.
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

// doRunSort — запрос Run простой сортировки по её маршруту
// ("bubble-sort", "selection-sort", "insertion-sort").
func doRunSort(t *testing.T, route, values string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/"+route, nil)
	q := req.URL.Query()
	q.Set("values", values)
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunSortSSE — успешный Run простой сортировки: базовые проверки
// SSE-ответа + кадры.
func doRunSortSSE(t *testing.T, route, values string) []frame.Frame {
	t.Helper()

	resp := doRunSort(t, route, values)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// assertStepsInOrder — шаги нумеруются с 1 и идут по порядку.
func assertStepsInOrder(t *testing.T, frames []frame.Frame) {
	t.Helper()

	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидалось %d", i, f.Step, i+1)
		}
	}
}

// assertSortValidation — общий контракт входа на каждом маршруте
// сортировок: невалидные значения отвергаются 400 с JSON-ошибкой
// до начала SSE.
func assertSortValidation(t *testing.T, route string) {
	t.Helper()

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
			resp := doRunSort(t, route, tt.values)
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

func TestRunBubbleSortValidation(t *testing.T)    { assertSortValidation(t, "bubble-sort") }
func TestRunSelectionSortValidation(t *testing.T) { assertSortValidation(t, "selection-sort") }
func TestRunInsertionSortValidation(t *testing.T) { assertSortValidation(t, "insertion-sort") }
func TestRunMergeSortValidation(t *testing.T)     { assertSortValidation(t, "merge-sort") }

// runSortTrivialInputs — прогон тривиальных входов: пустой и одноэлементный
// массивы — легальные входы, SSE-поток из одного стартового кадра без
// сравнений. Возвращает кадры обоих потоков ([пустой вход, одноэлементный])
// для дополнительных проверок конкретной сортировки.
func runSortTrivialInputs(t *testing.T, route string) [][]frame.Frame {
	t.Helper()

	var runs [][]frame.Frame
	for _, tc := range []struct {
		name  string
		input string
		want  []int
	}{
		{"пустой массив", "", []int{}},
		{"один элемент", "-7", []int{-7}},
	} {
		frames := doRunSortSSE(t, route, tc.input)
		if len(frames) != 1 {
			t.Fatalf("%s: кадров = %d, ожидался 1:\n%v", tc.name, len(frames), frames)
		}
		assertSortSnapshot(t, frames[0], tc.want, nil, "")
		runs = append(runs, frames)
	}
	return runs
}

// assertSortMatchesDirectRenderer — рендерер и транспорт не должны
// расходиться: кадры из эндпоинта совпадают с кадрами, собранными напрямую
// у рендерера (framesOf) на том же входе.
func assertSortMatchesDirectRenderer(t *testing.T, route, raw string, framesOf func([]int) []frame.Frame) {
	t.Helper()

	frames := doRunSortSSE(t, route, raw)
	want := framesOf(mustInts(t, raw))

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

// sortedInput — «0,1,…,n-1»: уже упорядоченный вход. У каждой сортировки
// он даёт конечный поток кадров, а граница лимита всё равно честная
// (почему конечный — в комментарии каждого теста лимита).
func sortedInput(n int) string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = fmt.Sprintf("%d", i)
	}
	return strings.Join(parts, ",")
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
	frames := doRunSortSSE(t, "bubble-sort", "3,1,2")
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

	assertStepsInOrder(t, frames)
}

// Пустой массив — легальный вход: SSE-поток состоит из одного стартового
// кадра с пустым рядом и завершается; одноэлементный вход — кадр исходного
// состояния без сравнений.
func TestRunBubbleSortEmptyInput(t *testing.T) { runSortTrivialInputs(t, "bubble-sort") }

// Граница лимита длины: 100 элементов принимаются (SSE-поток завершён),
// 101-й элемент отвергается до начала потока. Вход уже отсортирован:
// ровно n-1 сравнений, обменов нет — поток короткий, а граница честная.
func TestRunBubbleSortLengthLimit(t *testing.T) {
	frames := doRunSortSSE(t, "bubble-sort", sortedInput(100))
	if want := 100; len(frames) != want {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), want)
	}
	pair := func(index int, role string) []render.SortHighlight {
		return []render.SortHighlight{{Index: index, Role: role}, {Index: index + 1, Role: role}}
	}
	assertSortSnapshot(t, frames[len(frames)-1], mustInts(t, sortedInput(100)), pair(98, render.RoleCompare), "сравнение 98 99")

	resp := doRunSort(t, "bubble-sort", sortedInput(101))
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
	frames := doRunSortSSE(t, "bubble-sort", "-1,-1,0")
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
	assertSortMatchesDirectRenderer(t, "bubble-sort", "3,1,2", render.BubbleFrames)
}

// Сквозной сценарий теста: сортировка [3,1,2] — один стартовый кадр и
// по кадру на каждое событие (сравнение кандидата с текущим минимумом,
// финальный обмен прохода). Кадр сравнения подсвечивает минимум и
// кандидата разными ролями; итог — упорядоченный массив.
func TestRunSelectionSortScenario(t *testing.T) {
	frames := doRunSortSSE(t, "selection-sort", "3,1,2")
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

	assertStepsInOrder(t, frames)
}

// Регрессия учебной точки сравнения: когда кандидат становится новым
// минимумом, кадр обязан показывать роль "min" на его позиции — тот
// минимум, о котором говорит пояснение, — а прежний минимум — участником
// сравнения ("compare"), а не наоборот.
func TestRunSelectionSortCompareFrameShowsNewMinimum(t *testing.T) {
	frames := doRunSortSSE(t, "selection-sort", "5,2")
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

// Минимум вдали от начала прохода ([5,4,3,1]): кадры сравнения, сменившего
// минимум, подсвечивают прежний минимум ролью "compare" и новый — ролью
// "min" на его позиции; финальный обмен прохода перемещает новый минимум
// на первую неотсортированную позицию (0 ↔ 3, не соседние).
func TestRunSelectionSortNonAdjacentMinimum(t *testing.T) {
	frames := doRunSortSSE(t, "selection-sort", "5,4,3,1")
	if len(frames) != 9 {
		t.Fatalf("кадров = %d, ожидалось 9:\n%v", len(frames), frames)
	}

	start := []int{5, 4, 3, 1}
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

	// Проход 0: минимум последовательно уходит вправо — 4, затем 3,
	// затем 1; каждый кадр сравнения показывает новый минимум на его
	// позиции и прежний минимум участником сравнения.
	assertSortSnapshot(t, frames[0], start, nil, "")
	assertSortSnapshot(t, frames[1], start, compareMove(0, 1), "сравнение 0 1")
	assertSortSnapshot(t, frames[2], start, compareMove(1, 2), "сравнение 1 2")
	assertSortSnapshot(t, frames[3], start, compareMove(2, 3), "сравнение 2 3")
	// Пояснение сравнения несёт значение нового минимума из события.
	if !strings.Contains(frames[1].Explanation, "минимум переходит на позицию 1 (значение 4)") {
		t.Errorf("пояснение смены минимума = %q, ожидается переход минимума со значением", frames[1].Explanation)
	}
	// Финальный обмен прохода: минимум 1 с позиции 3 уходит на позицию 0.
	assertSortSnapshot(t, frames[4], []int{1, 4, 3, 5}, swapPair(0, 3), "обмен 0 3")

	// Конец сортировки: последний кадр — сравнение без смены минимума
	// на уже упорядоченном массиве.
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, []int{1, 3, 4, 5},
		[]render.SortHighlight{
			{Index: 2, Role: render.RoleMin},
			{Index: 3, Role: render.RoleCompare},
		},
		"сравнение 2 3")
}

// Пустой массив — легальный вход: SSE-поток состоит из одного стартового
// кадра с пустым рядом и завершается; одноэлементный вход — кадр исходного
// состояния без сравнений.
func TestRunSelectionSortEmptyInput(t *testing.T) { runSortTrivialInputs(t, "selection-sort") }

// Граница лимита длины: 100 элементов принимаются (SSE-поток завершён),
// 101-й элемент отвергается до начала потока. Вход уже отсортирован:
// selection sort всегда досматривает остаток целиком — граница честная,
// поток конечный.
func TestRunSelectionSortLengthLimit(t *testing.T) {
	frames := doRunSortSSE(t, "selection-sort", sortedInput(100))
	// На каждом проходе i: (n-1-i) сравнений, обменов нет —
	// 1 стартовый кадр + 99+98+...+1 = 4950 кадров сравнений.
	if want := 1 + 99*100/2; len(frames) != want {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), want)
	}
	assertSortSnapshot(t, frames[len(frames)-1], mustInts(t, sortedInput(100)),
		[]render.SortHighlight{{Index: 98, Role: render.RoleMin}, {Index: 99, Role: render.RoleCompare}},
		"сравнение 98 99")

	resp := doRunSort(t, "selection-sort", sortedInput(101))
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
	frames := doRunSortSSE(t, "selection-sort", "-1,-1,0")
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
	assertSortMatchesDirectRenderer(t, "selection-sort", "3,1,2", render.SelectionFrames)
}

// assertSortKeyState сверяет состояние вставки в снимке (после
// JSON-кругосветки): индекс «дырки» и значение удерживаемого ключа.
// Шаги без дырки — gap -1, ключ не задан (0).
func assertSortKeyState(t *testing.T, f frame.Frame, gap, key int) {
	t.Helper()

	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	if got := s["gap"].(float64); got != float64(gap) {
		t.Fatalf("кадр шага %d: gap = %v, ожидалось %d\nпояснение: %s", f.Step, got, gap, f.Explanation)
	}
	if got := s["key"].(float64); got != float64(key) {
		t.Fatalf("кадр шага %d: key = %v, ожидалось %d\nпояснение: %s", f.Step, got, key, f.Explanation)
	}
}

// Сквозной сценарий теста: сортировка [3,1,2] — один стартовый кадр и по
// кадру на каждое событие (сравнение с ключом, сдвиг, вставка). Кадр сдвига
// показывает пустую «дырку» и удерживаемый ключ; итог — упорядоченный
// массив.
func TestRunInsertionSortScenario(t *testing.T) {
	frames := doRunSortSSE(t, "insertion-sort", "3,1,2")
	if len(frames) != 8 {
		t.Fatalf("кадров = %d, ожидалось 8:\n%v", len(frames), frames)
	}

	start := []int{3, 1, 2}
	hole := []int{3, 3, 2} // после сдвига: 3 стоит дважды, дырка на позиции 0
	sorted := []int{1, 2, 3}

	assertSortSnapshot(t, frames[0], start, nil, "") // старт: исходный массив
	assertSortKeyState(t, frames[0], -1, 0)
	// Ключ 1: 3 больше ключа — сдвиг, затем вставка в позицию 0.
	// Дырка стартует на месте ключа (позиция 1) и уходит влево.
	assertSortSnapshot(t, frames[1], start, []render.SortHighlight{{Index: 0, Role: render.RoleCompare}}, "сравнение 0 с ключом")
	assertSortKeyState(t, frames[1], 1, 1)
	assertSortSnapshot(t, frames[2], hole, []render.SortHighlight{{Index: 1, Role: render.RoleShift}}, "сдвиг 0 → 1")
	assertSortKeyState(t, frames[2], 0, 1)
	assertSortSnapshot(t, frames[3], []int{1, 3, 2}, []render.SortHighlight{{Index: 0, Role: render.RoleInsert}}, "вставка на позицию 0")
	assertSortKeyState(t, frames[3], -1, 0)
	// Ключ 2: 3 больше — сдвиг; 1 меньше — сканирование остановлено; вставка в позицию 1.
	assertSortSnapshot(t, frames[4], []int{1, 3, 2}, []render.SortHighlight{{Index: 1, Role: render.RoleCompare}}, "сравнение 1 с ключом")
	assertSortKeyState(t, frames[4], 2, 2)
	assertSortSnapshot(t, frames[5], []int{1, 3, 3}, []render.SortHighlight{{Index: 2, Role: render.RoleShift}}, "сдвиг 1 → 2")
	assertSortKeyState(t, frames[5], 1, 2)
	assertSortSnapshot(t, frames[6], []int{1, 3, 3}, []render.SortHighlight{{Index: 0, Role: render.RoleCompare}}, "сравнение 0 с ключом")
	assertSortKeyState(t, frames[6], 1, 2)
	assertSortSnapshot(t, frames[7], sorted, []render.SortHighlight{{Index: 1, Role: render.RoleInsert}}, "вставка на позицию 1")
	assertSortKeyState(t, frames[7], -1, 0)

	// Пояснения отражают учебные точки: больший ключа сдвигается,
	// меньший останавливает сканирование, ключ вставляется в дырку.
	if !strings.Contains(frames[1].Explanation, "больше ключа") {
		t.Errorf("пояснение сравнения с большим = %q, ожидается «больше ключа»", frames[1].Explanation)
	}
	if !strings.Contains(frames[6].Explanation, "меньше ключа") {
		t.Errorf("пояснение сравнения с меньшим = %q, ожидается «меньше ключа»", frames[6].Explanation)
	}
	if !strings.Contains(frames[7].Explanation, "вставляем") {
		t.Errorf("пояснение вставки = %q, ожидается упоминание вставки", frames[7].Explanation)
	}

	assertStepsInOrder(t, frames)
}

// Обратный вход [3,2,1]: каждый ключ проходит весь префикс — цепочка
// сдвигов до позиции 0, финальная вставка в начало; дырка в кадрах
// доходит до начала ряда.
func TestRunInsertionSortReverseShiftsToStart(t *testing.T) {
	frames := doRunSortSSE(t, "insertion-sort", "3,2,1")
	if len(frames) != 9 {
		t.Fatalf("кадров = %d, ожидалось 9:\n%v", len(frames), frames)
	}

	assertSortSnapshot(t, frames[0], []int{3, 2, 1}, nil, "")
	// Ключ 2: сдвиг 0 → 1, вставка в позицию 0.
	assertSortSnapshot(t, frames[1], []int{3, 2, 1}, []render.SortHighlight{{Index: 0, Role: render.RoleCompare}}, "сравнение 0 с ключом")
	assertSortKeyState(t, frames[1], 1, 2)
	assertSortSnapshot(t, frames[2], []int{3, 3, 1}, []render.SortHighlight{{Index: 1, Role: render.RoleShift}}, "сдвиг 0 → 1")
	assertSortKeyState(t, frames[2], 0, 2)
	assertSortSnapshot(t, frames[3], []int{2, 3, 1}, []render.SortHighlight{{Index: 0, Role: render.RoleInsert}}, "вставка на позицию 0")
	// Ключ 1: два сдвига (дырка доходит до позиции 0), вставка в начало.
	assertSortSnapshot(t, frames[4], []int{2, 3, 1}, []render.SortHighlight{{Index: 1, Role: render.RoleCompare}}, "сравнение 1 с ключом")
	assertSortKeyState(t, frames[4], 2, 1)
	assertSortSnapshot(t, frames[5], []int{2, 3, 3}, []render.SortHighlight{{Index: 2, Role: render.RoleShift}}, "сдвиг 1 → 2")
	assertSortKeyState(t, frames[5], 1, 1)
	assertSortSnapshot(t, frames[6], []int{2, 3, 3}, []render.SortHighlight{{Index: 0, Role: render.RoleCompare}}, "сравнение 0 с ключом")
	assertSortKeyState(t, frames[6], 1, 1)
	assertSortSnapshot(t, frames[7], []int{2, 2, 3}, []render.SortHighlight{{Index: 1, Role: render.RoleShift}}, "сдвиг 0 → 1")
	assertSortKeyState(t, frames[7], 0, 1)
	assertSortSnapshot(t, frames[8], []int{1, 2, 3}, []render.SortHighlight{{Index: 0, Role: render.RoleInsert}}, "вставка на позицию 0")
	assertSortKeyState(t, frames[8], -1, 0)
}

// Устойчивость (учебная точка): элемент, равный ключу, не сдвигается —
// для [5,5] проход завершается видимым безсдвиговым кадром вставки
// «ключ уже на месте».
func TestRunInsertionSortEqualKeyDoesNotShift(t *testing.T) {
	frames := doRunSortSSE(t, "insertion-sort", "5,5")
	if len(frames) != 3 {
		t.Fatalf("кадров = %d, ожидалось 3 (сравнение + безсдвиговая вставка):\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{5, 5}, nil, "")
	assertSortSnapshot(t, frames[1], []int{5, 5},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}},
		"сравнение 0 с ключом")
	assertSortKeyState(t, frames[1], 1, 5)
	if !strings.Contains(frames[1].Explanation, "порядок равных сохраняется") {
		t.Errorf("пояснение сравнения равных = %q, ожидается устойчивость", frames[1].Explanation)
	}
	// Безсдвиговый проход завершается явным кадром: ключ на своём месте.
	assertSortSnapshot(t, frames[2], []int{5, 5},
		[]render.SortHighlight{{Index: 1, Role: render.RoleInsert}},
		"вставка на позицию 1")
	assertSortKeyState(t, frames[2], -1, 0)
	if !strings.Contains(frames[2].Explanation, "уже на месте") {
		t.Errorf("пояснение безсдвиговой вставки = %q, ожидается «уже на месте»", frames[2].Explanation)
	}
}

// Пустой массив — легальный вход: SSE-поток состоит из одного стартового
// кадра с пустым рядом и завершается; одноэлементный вход — кадр исходного
// состояния без сравнений. Вставки на тривиальных входах нет: дырки нет,
// ключа нет.
func TestRunInsertionSortEmptyInput(t *testing.T) {
	runs := runSortTrivialInputs(t, "insertion-sort")
	for _, frames := range runs {
		assertSortKeyState(t, frames[0], -1, 0)
	}
}

// Граница лимита длины: 100 элементов принимаются (SSE-поток завершён),
// 101-й элемент отвергается до начала потока. Вход уже отсортирован:
// каждый ключ сразу встаёт на своё место — n-1 сравнений и n-1
// безсдвиговых вставок, поток конечный, а граница честная.
func TestRunInsertionSortLengthLimit(t *testing.T) {
	frames := doRunSortSSE(t, "insertion-sort", sortedInput(100))
	if want := 1 + 99 + 99; len(frames) != want {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), want)
	}
	// Последний кадр — безсдвиговая вставка последнего ключа.
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, mustInts(t, sortedInput(100)),
		[]render.SortHighlight{{Index: 99, Role: render.RoleInsert}},
		"вставка на позицию 99")
	assertSortKeyState(t, last, -1, 0)
	if !strings.Contains(last.Explanation, "уже на месте") {
		t.Errorf("пояснение безсдвиговой вставки = %q, ожидается «уже на месте»", last.Explanation)
	}

	resp := doRunSort(t, "insertion-sort", sortedInput(101))
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
// в кадрах: вход уже упорядочен — сравнения равных и меньших, затем
// безсдвиговые вставки.
func TestRunInsertionSortNegativesAndDuplicates(t *testing.T) {
	frames := doRunSortSSE(t, "insertion-sort", "-1,-1,0")
	if len(frames) != 5 {
		t.Fatalf("кадров = %d, ожидалось 5:\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{-1, -1, 0}, nil, "")
	// Отрицательный ключ — легальное значение удерживаемого ключа.
	assertSortSnapshot(t, frames[1], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}},
		"сравнение 0 с ключом")
	assertSortKeyState(t, frames[1], 1, -1)
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 2, Role: render.RoleInsert}},
		"вставка на позицию 2")
	assertSortKeyState(t, last, -1, 0)
	if !strings.Contains(last.Explanation, "уже на месте") {
		t.Errorf("пояснение безсдвиговой вставки = %q, ожидается «уже на месте»", last.Explanation)
	}
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта совпадают
// с кадрами, собранными напрямую у рендерера.
func TestRunInsertionSortMatchesDirectRenderer(t *testing.T) {
	assertSortMatchesDirectRenderer(t, "insertion-sort", "3,1,2", render.InsertionFrames)
}

// Сортировки без удерживаемого ключа не несут состояния вставки: кадры
// bubble sort имеют пустые gap/key — контракт снимка общий, семантика
// прежних визуализаций не меняется.
func TestRunBubbleSortFramesHaveNoInsertionState(t *testing.T) {
	frames := doRunSortSSE(t, "bubble-sort", "3,1,2")
	if len(frames) == 0 {
		t.Fatal("кадров нет")
	}
	assertSortKeyState(t, frames[0], -1, 0)
}

// midPtr — указатель на значение границы половин для ожидаемых диапазонов
// merge sort в assertSortRanges.
func midPtr(v int) *int { return &v }

// assertSortRanges сверяет активные диапазоны шага в снимке (после
// JSON-кругосветки); у шагов без активного диапазона — пустой список.
func assertSortRanges(t *testing.T, f frame.Frame, ranges []render.SortRange) {
	t.Helper()

	s, ok := f.Data.(map[string]any)
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	gotRaw, ok := s["ranges"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: ranges не список: %v", f.Step, s["ranges"])
	}
	if len(gotRaw) != len(ranges) {
		t.Fatalf("кадр шага %d: диапазонов %d, ожидалось %d", f.Step, len(gotRaw), len(ranges))
	}
	for i, want := range ranges {
		rg := gotRaw[i].(map[string]any)
		got := fmt.Sprintf("lo=%v hi=%v", rg["lo"], rg["hi"])
		expr := fmt.Sprintf("lo=%d hi=%d", want.Lo, want.Hi)
		switch {
		case want.Mid != nil:
			got += fmt.Sprintf(" mid=%v", rg["mid"])
			expr += fmt.Sprintf(" mid=%d", *want.Mid)
		default:
			if _, present := rg["mid"]; present {
				t.Fatalf("кадр шага %d: диапазон %d не должен нести mid", f.Step, i)
			}
		}
		if got != expr {
			t.Fatalf("кадр шага %d: диапазон %d = %s, ожидалось %s", f.Step, i, got, expr)
		}
	}
}

// Сквозной сценарий теста: сортировка [3,1,2] — один стартовый кадр, кадры
// делений рекурсивных диапазонов и по кадру на каждое сравнение/запись.
// Рекурсия делит [0,3) как [0,1) и [1,3), поэтому сначала делится [0,3),
// затем [1,3); после сортировки половин сначала сливается пара 1–2, затем
// весь диапазон; сравнения идут до записей (результат слияния живёт
// в буфере до переноса), кадр каждой записи показывает обновлённую
// позицию; итог — упорядоченный массив.
func TestRunMergeSortScenario(t *testing.T) {
	frames := doRunSortSSE(t, "merge-sort", "3,1,2")
	if len(frames) != 11 {
		t.Fatalf("кадров = %d, ожидалось 11:\n%v", len(frames), frames)
	}

	start := []int{3, 1, 2}
	sorted := []int{1, 2, 3}

	assertSortSnapshot(t, frames[0], start, nil, "") // старт: исходный массив
	assertSortRanges(t, frames[0], nil)

	// Деление рекурсивных диапазонов до спуска в их половины; зеркальное
	// состояние делений не меняется — меняется только разметка диапазона.
	assertSortSnapshot(t, frames[1], start, nil, "деление 0..2")
	assertSortRanges(t, frames[1], []render.SortRange{{Lo: 0, Mid: midPtr(1), Hi: 3}})
	assertSortSnapshot(t, frames[2], start, nil, "деление 1..2")
	assertSortRanges(t, frames[2], []render.SortRange{{Lo: 1, Mid: midPtr(2), Hi: 3}})
	if !strings.Contains(frames[1].Explanation, "Делим диапазон [0, 3)") {
		t.Errorf("пояснение деления = %q, ожидается деление [0, 3) пополам", frames[1].Explanation)
	}

	// Слияние [1,3): сравнение голов, потом две записи в порядке позиций.
	assertSortSnapshot(t, frames[3], start,
		[]render.SortHighlight{{Index: 1, Role: render.RoleCompare}, {Index: 2, Role: render.RoleCompare}},
		"сравнение 1 2")
	assertSortRanges(t, frames[3], []render.SortRange{{Lo: 1, Mid: midPtr(2), Hi: 3}})
	assertSortSnapshot(t, frames[4], []int{3, 1, 2},
		[]render.SortHighlight{{Index: 1, Role: render.RoleWrite}}, "запись 1")
	assertSortSnapshot(t, frames[5], []int{3, 1, 2},
		[]render.SortHighlight{{Index: 2, Role: render.RoleWrite}}, "запись 2")
	// Кадры записи — приёмник слияния: разметки половин источника на них нет.
	assertSortRanges(t, frames[4], nil)
	assertSortRanges(t, frames[5], nil)

	// Слияние [0,3): два сравнения по неизменному префиксу и три записи.
	assertSortSnapshot(t, frames[6], []int{3, 1, 2},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 1, Role: render.RoleCompare}},
		"сравнение 0 1")
	assertSortRanges(t, frames[6], []render.SortRange{{Lo: 0, Mid: midPtr(1), Hi: 3}})
	assertSortSnapshot(t, frames[7], []int{3, 1, 2},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 2, Role: render.RoleCompare}},
		"сравнение 0 2")
	assertSortSnapshot(t, frames[8], []int{1, 1, 2},
		[]render.SortHighlight{{Index: 0, Role: render.RoleWrite}}, "запись 0")
	assertSortSnapshot(t, frames[9], []int{1, 2, 2},
		[]render.SortHighlight{{Index: 1, Role: render.RoleWrite}}, "запись 1")
	assertSortSnapshot(t, frames[10], sorted,
		[]render.SortHighlight{{Index: 2, Role: render.RoleWrite}}, "запись 2")
	assertSortRanges(t, frames[10], nil)

	// Пояснения отражают учебные точки: выбор из левой половины, из правой
	// и итоговую запись слияния.
	if !strings.Contains(frames[3].Explanation, "берём значение из левой половины") {
		t.Errorf("пояснение сравнения = %q, ожидается выбор левой половины", frames[3].Explanation)
	}
	if !strings.Contains(frames[6].Explanation, "из правой половины") {
		t.Errorf("пояснение сравнения = %q, ожидается выбор правой половины", frames[6].Explanation)
	}
	if !strings.Contains(frames[10].Explanation, "из левой половины") {
		t.Errorf("пояснение записи = %q, ожидается упоминание источника", frames[10].Explanation)
	}

	assertStepsInOrder(t, frames)
}

// Устойчивость (учебная точка): при равных кандидатах берётся значение
// из левой половины — первым сравнением и первой записью диапазона.
// Для [5,5] обе половины длины 1.
func TestRunMergeSortEqualTakesLeft(t *testing.T) {
	frames := doRunSortSSE(t, "merge-sort", "5,5")
	if len(frames) != 5 {
		t.Fatalf("кадров = %d, ожидалось 5 (деление, сравнение, две записи):\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{5, 5}, nil, "")
	assertSortSnapshot(t, frames[1], []int{5, 5}, nil, "деление 0..1")
	assertSortRanges(t, frames[1], []render.SortRange{{Lo: 0, Mid: midPtr(1), Hi: 2}})
	assertSortSnapshot(t, frames[2], []int{5, 5},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 1, Role: render.RoleCompare}},
		"сравнение 0 1")
	if !strings.Contains(frames[2].Explanation, "равны") {
		t.Errorf("пояснение сравнения равных = %q, ожидается упоминание равенства", frames[2].Explanation)
	}
	if !strings.Contains(frames[3].Explanation, "из левой половины") {
		t.Errorf("пояснение записи = %q, ожидается первое значение из левой половины", frames[3].Explanation)
	}
	assertSortSnapshot(t, frames[4], []int{5, 5},
		[]render.SortHighlight{{Index: 1, Role: render.RoleWrite}}, "запись 1")
}

// Пустой массив — легальный вход: SSE-поток состоит из одного стартового
// кадра с пустым рядом и завершается; одноэлементный вход — кадр исходного
// состояния без сравнений.
func TestRunMergeSortEmptyInput(t *testing.T) { runSortTrivialInputs(t, "merge-sort") }

// Граница лимита длины: 100 элементов принимаются (SSE-поток завершён),
// 101-й элемент отвергается до начала потока. Вход уже отсортирован, кадры
// считаются независимой формулой: каждый элемент левой половины слива
// выигрывает первым же сравнением, поэтому сравнений — длина левой части
// каждой слия; записей — длина каждой слия целиком.
func TestRunMergeSortLengthLimit(t *testing.T) {
	frames := doRunSortSSE(t, "merge-sort", sortedInput(100))
	if len(frames) != 1+mergeSortedEvents(100) {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), 1+mergeSortedEvents(100))
	}
	// Последний кадр — запись последнего элемента рекурсивного диапазона.
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, mustInts(t, sortedInput(100)),
		[]render.SortHighlight{{Index: 99, Role: render.RoleWrite}}, "запись 99")
	// Кадр записи — приёмник слияния: разметки половин источника на нём нет.
	assertSortRanges(t, last, nil)

	resp := doRunSort(t, "merge-sort", sortedInput(101))
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

// mergeSortedEvents — независимая формула для числа кадров merge sort
// минус стартовый на уже отсортированном входе длины n: каждый рекурсивный
// диапазон добавляет один кадр деления (только для n > 1), длина левой
// половины — кадры сравнений (левая сливается, выигрывая каждый раз с
// первым сравнением), длина диапазона — кадры записей (записываются все
// элементы): E(n) = E(nL) + E(nR) + 1 + nL + n, E(1) = 0. nL = n/2 — как
// в рекурсии.
func mergeSortedEvents(n int) int {
	if n <= 1 {
		return 0
	}
	l := n / 2
	return mergeSortedEvents(l) + mergeSortedEvents(n-l) + 1 + l + n
}

// Отрицательные значения и дубликаты сортируются на месте и читаются
// в кадрах: сравнения равных идут в начале слияний, итоговый массив
// упорядочен.
func TestRunMergeSortNegativesAndDuplicates(t *testing.T) {
	frames := doRunSortSSE(t, "merge-sort", "-1,-1,0")
	if len(frames) != 10 {
		t.Fatalf("кадров = %d, ожидалось 10:\n%v", len(frames), frames)
	}
	assertSortSnapshot(t, frames[0], []int{-1, -1, 0}, nil, "")
	// Деления рекурсивных диапазонов: [0,3), затем [1,3).
	assertSortSnapshot(t, frames[1], []int{-1, -1, 0}, nil, "деление 0..2")
	assertSortSnapshot(t, frames[2], []int{-1, -1, 0}, nil, "деление 1..2")
	// Слияние [1,3): сравнение -1 и 0 — берём левую.
	assertSortSnapshot(t, frames[3], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 1, Role: render.RoleCompare}, {Index: 2, Role: render.RoleCompare}},
		"сравнение 1 2")
	if !strings.Contains(frames[3].Explanation, "берём значение из левой половины") {
		t.Errorf("пояснение сравнения = %q, ожидается выбор левой половины", frames[3].Explanation)
	}
	// Слияние [0,3): сравнение одинаковых -1 — равные, берём левую.
	assertSortSnapshot(t, frames[6], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 1, Role: render.RoleCompare}},
		"сравнение 0 1")
	if !strings.Contains(frames[6].Explanation, "равны") {
		t.Errorf("пояснение сравнения дубликатов = %q, ожидается равенство кандидатов", frames[6].Explanation)
	}
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 2, Role: render.RoleWrite}}, "запись 2")
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта совпадают
// с кадрами, собранными напрямую у рендерера.
func TestRunMergeSortMatchesDirectRenderer(t *testing.T) {
	assertSortMatchesDirectRenderer(t, "merge-sort", "3,1,2", render.MergeFrames)
}

// assertPartitionRange сверяет активный диапазон partition quicksort
// в снимке (после JSON-кругосветки): границы lo..hi включительно,
// границы половин mid у partition нет.
func assertPartitionRange(t *testing.T, f frame.Frame, lo, hi int) {
	t.Helper()

	s, ok := f.Data.(map[string]any)
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	gotRaw, ok := s["ranges"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: ranges не список: %v", f.Step, s["ranges"])
	}
	if len(gotRaw) != 1 {
		t.Fatalf("кадр шага %d: диапазонов %d, ожидался 1", f.Step, len(gotRaw))
	}
	rg := gotRaw[0].(map[string]any)
	if got := rg["lo"].(float64); got != float64(lo) {
		t.Fatalf("кадр шага %d: lo диапазона = %v, ожидалось %d", f.Step, got, lo)
	}
	if got := rg["hi"].(float64); got != float64(hi+1) {
		t.Fatalf("кадр шага %d: hi диапазона = %v, ожидалось %d (включительный hi partition)", f.Step, got, hi+1)
	}
	if _, present := rg["mid"]; present {
		t.Fatalf("кадр шага %d: у диапазона partition не может быть границы половин mid", f.Step)
	}
}

func TestRunQuickSortValidation(t *testing.T) { assertSortValidation(t, "quick-sort") }

// Сквозной сценарий теста: сортировка [2,4,1,3] — один стартовый кадр и по
// кадру на каждое событие: выбор pivot, три сравнения с pivot (первый
// кандидат-самообмен не эмитится), обмен строго меньшего влево, размещение
// pivot, затем partition левой части [0..1]. Итог — упорядоченный массив.
func TestRunQuickSortScenario(t *testing.T) {
	frames := doRunSortSSE(t, "quick-sort", "2,4,1,3")
	if len(frames) != 10 {
		t.Fatalf("кадров = %d, ожидалось 10:\n%v", len(frames), frames)
	}

	start := []int{2, 4, 1, 3}
	sorted := []int{1, 2, 3, 4}

	assertSortSnapshot(t, frames[0], start, nil, "") // старт: исходный массив
	assertSortRanges(t, frames[0], nil)

	// Partition [0..3] вокруг pivot 3 (последний элемент диапазона).
	assertSortSnapshot(t, frames[1], start,
		[]render.SortHighlight{{Index: 3, Role: render.RolePivot}}, "pivot 3")
	assertPartitionRange(t, frames[1], 0, 3)
	if !strings.Contains(frames[1].Explanation, "последний элемент") {
		t.Errorf("пояснение выбора pivot = %q, ожидается «последний элемент»", frames[1].Explanation)
	}

	// 2 < 3: строго меньше; блока меньших ещё нет (boundary не подсвечен).
	assertSortSnapshot(t, frames[2], start,
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 3, Role: render.RolePivot}},
		"сравнение 0 с pivot")
	assertPartitionRange(t, frames[2], 0, 3)
	if !strings.Contains(frames[2].Explanation, "строго меньше") {
		t.Errorf("пояснение сравнения с меньшим = %q, ожидается «строго меньше»", frames[2].Explanation)
	}

	// 4 не меньше 3; блок меньших вырос до позиции 0 — граница подсвечена.
	assertSortSnapshot(t, frames[3], start,
		[]render.SortHighlight{{Index: 1, Role: render.RoleCompare}, {Index: 3, Role: render.RolePivot}, {Index: 0, Role: render.RoleBoundary}},
		"сравнение 1 с pivot")
	if !strings.Contains(frames[3].Explanation, "не меньше pivot") {
		t.Errorf("пояснение сравнения с большим = %q, ожидается «не меньше pivot»", frames[3].Explanation)
	}

	// 1 < 3: уходит влево обменом с позицией 1.
	assertSortSnapshot(t, frames[4], start,
		[]render.SortHighlight{{Index: 2, Role: render.RoleCompare}, {Index: 3, Role: render.RolePivot}, {Index: 0, Role: render.RoleBoundary}},
		"сравнение 2 с pivot")
	assertSortSnapshot(t, frames[5], []int{2, 1, 4, 3},
		[]render.SortHighlight{{Index: 1, Role: render.RoleBoundary}, {Index: 2, Role: render.RoleSwapped}, {Index: 3, Role: render.RolePivot}},
		"обмен 1 2")
	if !strings.Contains(frames[5].Explanation, "блок меньших вырос") {
		t.Errorf("пояснение обмена = %q, ожидается рост блока меньших", frames[5].Explanation)
	}

	// Pivot 3 занимает финальную позицию 2: 4 переезжает в конец.
	assertSortSnapshot(t, frames[6], []int{2, 1, 3, 4},
		[]render.SortHighlight{{Index: 2, Role: render.RolePivot}}, "pivot на позиции 2")
	assertSortRanges(t, frames[6], nil)

	// Рекурсия слева partition [0..1] вокруг pivot 1.
	assertSortSnapshot(t, frames[7], []int{2, 1, 3, 4},
		[]render.SortHighlight{{Index: 1, Role: render.RolePivot}}, "pivot 1")
	assertPartitionRange(t, frames[7], 0, 1)
	assertSortSnapshot(t, frames[8], []int{2, 1, 3, 4},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 1, Role: render.RolePivot}},
		"сравнение 0 с pivot")
	assertPartitionRange(t, frames[8], 0, 1)

	// Pivot 1 занимает финальную позицию 0; правые рекурсии ([2..2] и [3..3])
	// одноэлементны — partition'ов и событий на них нет.
	assertSortSnapshot(t, frames[9], sorted,
		[]render.SortHighlight{{Index: 0, Role: render.RolePivot}}, "pivot на позиции 0")
	if !strings.Contains(frames[9].Explanation, "финальную позицию") {
		t.Errorf("пояснение размещения pivot = %q, ожидается финальная позиция", frames[9].Explanation)
	}

	assertStepsInOrder(t, frames)
}

// Дубликаты pivot уходят вправо (строгое сравнение из черновика):
// вход [-1,-1,0] — оба элемента строго меньше pivot 0 и за счёт частых
// самообменов не порождают обменов; сравнение равных в partition [0..1]
// не сдвигает дубликат с правой стороны.
func TestRunQuickSortDuplicates(t *testing.T) {
	frames := doRunSortSSE(t, "quick-sort", "-1,-1,0")
	if len(frames) != 8 {
		t.Fatalf("кадров = %d, ожидалось 8:\n%v", len(frames), frames)
	}

	assertSortSnapshot(t, frames[0], []int{-1, -1, 0}, nil, "")
	assertSortSnapshot(t, frames[1], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 2, Role: render.RolePivot}}, "pivot 2")
	// Оба элемента строго меньше 0 — по кадру сравнения на каждый.
	assertSortSnapshot(t, frames[2], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 2, Role: render.RolePivot}},
		"сравнение 0 с pivot")
	assertSortSnapshot(t, frames[3], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 1, Role: render.RoleCompare}, {Index: 2, Role: render.RolePivot}, {Index: 0, Role: render.RoleBoundary}},
		"сравнение 1 с pivot")
	// Pivot 0 уже на финальной позиции 2 — обмена не потребовалось.
	assertSortSnapshot(t, frames[4], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 2, Role: render.RolePivot}}, "pivot на позиции 2")
	if !strings.Contains(frames[4].Explanation, "обмена не потребовалось") {
		t.Errorf("пояснение размещения = %q, ожидается «обмена не потребовалось»", frames[4].Explanation)
	}
	// Partition [0..1] вокруг pivot -1: сравнение равных значений —
	// дубликат не строго меньше pivot, остаётся справа.
	assertSortSnapshot(t, frames[5], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 1, Role: render.RolePivot}}, "pivot 1")
	assertSortSnapshot(t, frames[6], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 0, Role: render.RoleCompare}, {Index: 1, Role: render.RolePivot}},
		"сравнение 0 с pivot")
	if !strings.Contains(frames[6].Explanation, "не меньше pivot") {
		t.Errorf("пояснение сравнения дубликатов = %q, ожидается «не меньше pivot»", frames[6].Explanation)
	}
	assertSortSnapshot(t, frames[7], []int{-1, -1, 0},
		[]render.SortHighlight{{Index: 0, Role: render.RolePivot}}, "pivot на позиции 0")

	assertStepsInOrder(t, frames)
}

// Вырожденный вход — уже упорядоченный массив: каждый pivot — максимум
// диапазона, строго меньшие стоят подряд (самообмен пропускается),
// обменов в потоке нет совсем.
func TestRunQuickSortSortedDegenerate(t *testing.T) {
	frames := doRunSortSSE(t, "quick-sort", "1,2,3")
	if len(frames) != 8 {
		t.Fatalf("кадров = %d, ожидалось 8:\n%v", len(frames), frames)
	}
	for _, f := range frames {
		s, ok := f.Data.(map[string]any)
		if !ok {
			t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
		}
		if op, ok := s["operation"].(string); ok && strings.HasPrefix(op, "обмен") {
			t.Errorf("на упорядоченном входе обменов не ожидается: кадр %d, операция %q", f.Step, op)
		}
	}
}

// Пустой массив — легальный вход: SSE-поток состоит из одного стартового
// кадра с пустым рядом и завершается; одноэлементный вход — кадр исходного
// состояния без partition.
func TestRunQuickSortEmptyInput(t *testing.T) {
	runs := runSortTrivialInputs(t, "quick-sort")
	for _, frames := range runs {
		assertSortRanges(t, frames[0], nil)
	}
}

// Граница лимита длины: 100 элементов принимаются (SSE-поток завершён),
// 101-й элемент отвергается до начала потока. Вход уже отсортирован —
// худший случай по глубине рекурсии, но кадры считаются независимой
// формулой: на partition длины k приходится 1 pivot + (k-1) сравнение
// + 1 размещение = k+1 событий; E(n) = Σ_{k=2..n}(k+1).
func TestRunQuickSortLengthLimit(t *testing.T) {
	frames := doRunSortSSE(t, "quick-sort", sortedInput(100))
	want := 1 + quickSortedEvents(100)
	if len(frames) != want {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), want)
	}
	// Последний кадр — размещение последнего pivot рекурсии в [0..1].
	last := frames[len(frames)-1]
	assertSortSnapshot(t, last, mustInts(t, sortedInput(100)),
		[]render.SortHighlight{{Index: 1, Role: render.RolePivot}}, "pivot на позиции 1")
	assertSortRanges(t, last, nil)

	resp := doRunSort(t, "quick-sort", sortedInput(101))
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

// quickSortedEvents — независимая формула числа кадров quicksort минус
// стартовый на уже отсортированном входе длины n: partition (0..n-1)
// даёт pivot + (n-1) сравнение + размещение, затем рекурсия в [0..n-2].
func quickSortedEvents(n int) int {
	total := 0
	for k := 2; k <= n; k++ {
		total += k + 1
	}
	return total
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта совпадают
// с кадрами, собранными напрямую у рендерера.
func TestRunQuickSortMatchesDirectRenderer(t *testing.T) {
	assertSortMatchesDirectRenderer(t, "quick-sort", "2,4,1,3", render.QuickFrames)
}
