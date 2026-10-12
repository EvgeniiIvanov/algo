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

// doRunQueue — запрос Run очереди.
func doRunQueue(t *testing.T, capacity, script string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/queue", nil)
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

// doRunQueueSSE — успешный Run очереди: базовые проверки SSE-ответа + кадры.
func doRunQueueSSE(t *testing.T, capacity, script string) []frame.Frame {
	t.Helper()

	resp := doRunQueue(t, capacity, script)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// assertQueueSnapshot сверяет снимок очереди в кадре.
func assertQueueSnapshot(t *testing.T, f frame.Frame, values []int, capacity, head, tail int, operation string) {
	t.Helper()

	if f.Kind != frame.KindStructure {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindStructure)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	if f.Data.(map[string]any)["structure"] != render.StructureQueue {
		t.Fatalf("кадр шага %d: structure = %v, ожидается %q", f.Step, f.Data.(map[string]any)["structure"], render.StructureQueue)
	}
	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	want := fmt.Sprintf("values=%v capacity=%d head=%d tail=%d top=%d operation=%q",
		values, capacity, head, tail, -1, operation)
	got := fmt.Sprintf("values=%v capacity=%v head=%v tail=%v top=%v operation=%q",
		s["values"], s["capacity"], s["head"], s["tail"], s["top"], s["operation"])
	if got != want {
		t.Fatalf("кадр шага %d: снимок = %s, ожидается %s\nпояснение: %s", f.Step, got, want, f.Explanation)
	}
}

// Сквозной FIFO-сценарий: запись, снятие со сдвигом и переполнение —
// каждый шаг своим кадром.
func TestRunQueueScenario(t *testing.T) {
	frames := doRunQueueSSE(t, "3", "enqueue 5\nenqueue 7\ndequeue\nenqueue 9\nenqueue 11\nenqueue 13")
	if len(frames) != 7 {
		t.Fatalf("кадров = %d, ожидается 7:\n%v", len(frames), frames)
	}

	assertQueueSnapshot(t, frames[0], []int{}, 3, -1, -1, "")                 // старт
	assertQueueSnapshot(t, frames[1], []int{5}, 3, 0, 0, "enqueue 5")         // enqueue 5
	assertQueueSnapshot(t, frames[2], []int{5, 7}, 3, 0, 1, "enqueue 7")      // enqueue 7
	assertQueueSnapshot(t, frames[3], []int{7}, 3, 0, 0, "dequeue")           // dequeue → сняли 5
	assertQueueSnapshot(t, frames[4], []int{7, 9}, 3, 0, 1, "enqueue 9")      // enqueue 9
	assertQueueSnapshot(t, frames[5], []int{7, 9, 11}, 3, 0, 2, "enqueue 11") // enqueue 11 — очередь полна
	assertQueueSnapshot(t, frames[6], []int{7, 9, 11}, 3, 0, 2, "enqueue 13") // переполнение

	// Подсветки: enqueue — added на новой хвостовой ячейке,
	// dequeue — сдвинувшиеся ячейки со shifted и освободившаяся — removed.
	assertHighlight(t, frames[1], 0, render.RoleAdded)
	assertHighlight(t, frames[2], 1, render.RoleAdded)
	assertQueueDequeueHighlights(t, frames[3], 2)
	assertHighlight(t, frames[4], 1, render.RoleAdded)
	assertNoHighlights(t, frames[0])
	assertNoHighlights(t, frames[6]) // переполнение ничего не трогает

	// Пояснения отражают смысл шага.
	if !strings.Contains(frames[3].Explanation, "сняли 5") {
		t.Errorf("пояснение шага 4 = %q, ожидается «сняли 5»", frames[3].Explanation)
	}
	if !strings.Contains(frames[3].Explanation, "сдвинулись влево") {
		t.Errorf("пояснение шага 4 = %q, ожидается упоминание сдвига влево", frames[3].Explanation)
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

// assertQueueDequeueHighlights сверяет подсветки шага dequeue:
// shifted на ячейках со сдвинувшимися значениями + removed на освободившейся.
func assertQueueDequeueHighlights(t *testing.T, f frame.Frame, oldLen int) {
	t.Helper()

	s := f.Data.(map[string]any)
	highlights, ok := s["highlights"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: highlights не массив: %#v", f.Step, s["highlights"])
	}
	if len(highlights) != 2 {
		t.Fatalf("кадр шага %d: подсветок = %d, ожидается 2 (shifted + removed)", f.Step, len(highlights))
	}

	roles := map[string]float64{}
	for _, h := range highlights {
		hm := h.(map[string]any)
		roles[hm["role"].(string)] = hm["index"].(float64)
	}
	// Сдвинувшиеся значения теперь в ячейке 0, освободилась ячейка oldLen-1.
	if roles[render.RoleShifted] != 0 {
		t.Fatalf("кадр шага %d: подсветка shifted = %v, ожидается индекс 0", f.Step, roles[render.RoleShifted])
	}
	if roles[render.RoleRemoved] != float64(oldLen-1) {
		t.Fatalf("кадр шага %d: подсветка removed = %v, ожидается индекс %d", f.Step, roles[render.RoleRemoved], oldLen-1)
	}
}

func TestRunQueueValidation(t *testing.T) {
	tests := []struct {
		name     string
		capacity string
		script   string
		want     string
	}{
		{"нет capacity", "", "enqueue 5", "capacity обязателен"},
		{"мусор в capacity", "три", "enqueue 5", "не удалось разобрать"},
		{"нулевая ёмкость", "0", "enqueue 5", "от 1 до"},
		{"отрицательная ёмкость", "-2", "enqueue 5", "от 1 до"},
		{"нет script", "3", "", "script обязателен"},
		{"пустой сценарий", "3", "\n  \n", "сценарий пуст"},
		{"неизвестная операция", "3", "jump 5", "неизвестная операция"},
		{"enqueue без значения", "3", "enqueue", "enqueue требует"},
		{"enqueue с лишним аргументом", "3", "enqueue 5 7", "enqueue требует"},
		{"мусор в значении enqueue", "3", "enqueue пять", "не удалось разобрать"},
		{"dequeue с аргументом", "3", "dequeue 5", "не принимает аргументов"},
		{"dequeue из пустой очереди", "3", "dequeue", "пустой очереди"},
		{"dequeue после опустошения", "3", "enqueue 5\ndequeue\ndequeue", "пустой очереди"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunQueue(t, tt.capacity, tt.script)
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

// Enqueue в полную очередь — не ошибка входа, а учебный кадр переполнения.
func TestRunQueueOverflowIsNotValidationError(t *testing.T) {
	frames := doRunQueueSSE(t, "1", "enqueue 5\nenqueue 9")
	if len(frames) != 3 {
		t.Fatalf("кадров = %d, ожидается 3:\n%v", len(frames), frames)
	}
	assertQueueSnapshot(t, frames[2], []int{5}, 1, 0, 0, "enqueue 9")
	assertNoHighlights(t, frames[2])
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта
// совпадают с кадрами, собранными напрямую у рендерера.
func TestRunQueueMatchesDirectRenderer(t *testing.T) {
	ops := []render.QueueOp{
		{IsEnqueue: true, Value: 5},
		{IsEnqueue: true, Value: 7},
		{IsEnqueue: false},
		{IsEnqueue: true, Value: 9},
	}
	frames := doRunQueueSSE(t, "2", "enqueue 5\nenqueue 7\ndequeue\nenqueue 9")
	want, err := render.QueueFrames(2, ops)
	if err != nil {
		t.Fatalf("QueueFrames(2): %v", err)
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
