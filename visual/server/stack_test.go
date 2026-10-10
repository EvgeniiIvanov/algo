package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// doRunStack — запрос Run стека.
func doRunStack(t *testing.T, capacity, script string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/stack", nil)
	q := req.URL.Query()
	if capacity != "" {
		q.Set("capacity", capacity)
	}
	if script != "" {
		q.Set("script", script)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunStackSSE — успешный Run стека: базовые проверки SSE-ответа + кадры.
func doRunStackSSE(t *testing.T, capacity, script string) []frame.Frame {
	t.Helper()

	resp := doRunStack(t, capacity, script)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// assertStackSnapshot сверяет снимок структуры в кадре.
func assertStackSnapshot(t *testing.T, f frame.Frame, values []int, capacity, top int, operation string) {
	t.Helper()

	if f.Kind != frame.KindStructure {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindStructure)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	want := fmt.Sprintf("structure=%s values=%v capacity=%d top=%d operation=%q",
		render.StructureStack, values, capacity, top, operation)
	got := fmt.Sprintf("structure=%v values=%v capacity=%v top=%v operation=%q",
		s["structure"], s["values"], s["capacity"], s["top"], s["operation"])
	if got != want {
		t.Fatalf("кадр шага %d: снимок = %s, ожидается %s\nпояснение: %s", f.Step, got, want, f.Explanation)
	}
}

// assertHighlight сверяет единственную подсветку в кадре (индекс + роль).
func assertHighlight(t *testing.T, f frame.Frame, wantIndex int, wantRole string) {
	t.Helper()

	s := f.Data.(map[string]any)
	highlights, ok := s["highlights"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: highlights не массив: %#v", f.Step, s["highlights"])
	}
	if len(highlights) != 1 {
		t.Fatalf("кадр шага %d: подсветок = %d, ожидается 1 (%s индекса %d)",
			f.Step, len(highlights), wantRole, wantIndex)
	}
	h := highlights[0].(map[string]any)
	if h["index"] != float64(wantIndex) || h["role"] != wantRole {
		t.Fatalf("кадр шага %d: подсветка = %v, ожидается {index: %d, role: %q}",
			f.Step, h, wantIndex, wantRole)
	}
}

// assertNoHighlights проверяет, что в кадре нет подсветок.
func assertNoHighlights(t *testing.T, f frame.Frame) {
	t.Helper()

	s := f.Data.(map[string]any)
	highlights, ok := s["highlights"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: highlights не массив: %#v", f.Step, s["highlights"])
	}
	if len(highlights) != 0 {
		t.Fatalf("кадр шага %d: подсветок = %d, ожидается 0", f.Step, len(highlights))
	}
}

// Сквозной сценарий: запись, снятие и переполнение — каждый шаг своим кадром.
func TestRunStackScenario(t *testing.T) {
	frames := doRunStackSSE(t, "3", "push 5\npush 7\npop\npush 9\npush 11\npush 13")
	if len(frames) != 7 {
		t.Fatalf("кадров = %d, ожидается 7:\n%v", len(frames), frames)
	}

	assertStackSnapshot(t, frames[0], []int{}, 3, -1, "")               // старт
	assertStackSnapshot(t, frames[1], []int{5}, 3, 0, "push 5")         // push 5
	assertStackSnapshot(t, frames[2], []int{5, 7}, 3, 1, "push 7")      // push 7
	assertStackSnapshot(t, frames[3], []int{5}, 3, 0, "pop")            // pop → сняли 7
	assertStackSnapshot(t, frames[4], []int{5, 9}, 3, 1, "push 9")      // push 9
	assertStackSnapshot(t, frames[5], []int{5, 9, 11}, 3, 2, "push 11") // push 11 — стек полон
	assertStackSnapshot(t, frames[6], []int{5, 9, 11}, 3, 2, "push 13") // переполнение

	assertHighlight(t, frames[1], 0, render.RoleAdded)
	assertHighlight(t, frames[2], 1, render.RoleAdded)
	assertHighlight(t, frames[3], 1, render.RoleRemoved)
	assertHighlight(t, frames[4], 1, render.RoleAdded)
	assertNoHighlights(t, frames[0])
	assertNoHighlights(t, frames[6]) // переполнение ничего не трогает

	// Пояснения отражают смысл шага.
	if !strings.Contains(frames[3].Explanation, "сняли 7") {
		t.Errorf("пояснение шага 4 = %q, ожидается «сняли 7»", frames[3].Explanation)
	}
	if !strings.Contains(frames[6].Explanation, "переполнение") {
		t.Errorf("пояснение шага 7 = %q, ожидается упоминание переполнения", frames[6].Explanation)
	}
	// Шаги нумеруются с 1 и идут по порядку.
	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидается %d", i, f.Step, i+1)
		}
	}
}

func TestRunStackValidation(t *testing.T) {
	tests := []struct {
		name     string
		capacity string
		script   string
		want     string
	}{
		{"нет capacity", "", "push 5", "capacity обязателен"},
		{"мусор в capacity", "три", "push 5", "не удалось разобрать"},
		{"нулевая ёмкость", "0", "push 5", "от 1 до"},
		{"отрицательная ёмкость", "-2", "push 5", "от 1 до"},
		{"нет script", "3", "", "script обязателен"},
		{"пустой сценарий", "3", "\n  \n", "сценарий пуст"},
		{"неизвестная операция", "3", "jump 5", "неизвестная операция"},
		{"push без значения", "3", "push", "push требует"},
		{"push с лишним аргументом", "3", "push 5 7", "push требует"},
		{"мусор в значении push", "3", "push пять", "не удалось разобрать"},
		{"pop с аргументом", "3", "pop 5", "не принимает аргументов"},
		{"pop из пустого стека", "3", "pop", "пустого стека"},
		{"pop после опустошения", "3", "push 5\npop\npop", "пустого стека"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunStack(t, tt.capacity, tt.script)
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

// Push в полный стек — не ошибка входа, а учебный кадр переполнения.
func TestRunStackOverflowIsNotValidationError(t *testing.T) {
	frames := doRunStackSSE(t, "1", "push 5\npush 9")
	if len(frames) != 3 {
		t.Fatalf("кадров = %d, ожидается 3:\n%v", len(frames), frames)
	}
	assertStackSnapshot(t, frames[2], []int{5}, 1, 0, "push 9")
	assertNoHighlights(t, frames[2])
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта
// совпадают с кадрами, собранными напрямую у рендерера.
func TestRunStackMatchesDirectRenderer(t *testing.T) {
	ops := []render.StackOp{
		{IsPush: true, Value: 5},
		{IsPush: true, Value: 7},
		{IsPush: false},
		{IsPush: true, Value: 9},
	}
	frames := doRunStackSSE(t, "2", "push 5\npush 7\npop\npush 9")
	want := render.StackFrames(2, ops)

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
