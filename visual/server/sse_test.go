// Тесты шва 2 (см. docs/SPEC.md): вход → SSE-поток → последовательность кадров.
// Одна проверка покрывает emitter, Frame Renderer и HTTP-слой.
package server

import (
	"bufio"
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

// readSSEFrames читает SSE-поток и возвращает кадры из событий data.
func readSSEFrames(t *testing.T, resp *httptest.ResponseRecorder) []frame.Frame {
	t.Helper()

	var frames []frame.Frame
	scanner := bufio.NewScanner(bytes.NewReader(resp.Body.Bytes()))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue // пустые строки — разделители SSE-событий
		}
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			t.Fatalf("в SSE-потоке мусор вне события data: %q", line)
		}
		var f frame.Frame
		if err := json.Unmarshal([]byte(data), &f); err != nil {
			t.Fatalf("кадр не разбирается как JSON: %v\n%s", err, data)
		}
		frames = append(frames, f)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("чтение SSE-потока: %v", err)
	}
	return frames
}

// doRun — запрос Run.
func doRun(t *testing.T, values, target string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/binary-search", nil)
	q := req.URL.Query()
	if values != "" {
		q.Set("values", values)
	}
	if target != "" {
		q.Set("target", target)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunSSE — успешный Run: базовые проверки SSE-ответа + кадры из потока.
func doRunSSE(t *testing.T, values, target string) ([]frame.Frame, *httptest.ResponseRecorder) {
	t.Helper()

	resp := doRun(t, values, target)
	assertSSE(t, resp)
	return readSSEFrames(t, resp), resp
}

// assertSSE проверяет базовые свойства успешного SSE-ответа.
func assertSSE(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()

	if resp.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидается 200", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, ожидается text/event-stream", got)
	}
}

// assertSnapshot сверяет снимок массива в кадре.
func assertSnapshot(t *testing.T, f frame.Frame, values []int, low, high, mid, found int) {
	t.Helper()

	if f.Kind != frame.KindArray {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindArray)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	want := fmt.Sprintf("values=%v low=%d high=%d mid=%d foundIndex=%d", values, low, high, mid, found)
	got := fmt.Sprintf("values=%v low=%v high=%v mid=%v foundIndex=%v",
		s["values"], s["low"], s["high"], s["mid"], s["foundIndex"])
	if got != want {
		t.Fatalf("кадр шага %d: снимок = %s, ожидается %s\nпояснение: %s", f.Step, got, want, f.Explanation)
	}
}

func TestRunBinarySearchFound(t *testing.T) {
	frames, _ := doRunSSE(t, "1,3,5,7,9", "7")
	if len(frames) != 5 {
		t.Fatalf("кадров = %d, ожидается 5:\n%v", len(frames), frames)
	}

	assertSnapshot(t, frames[0], []int{1, 3, 5, 7, 9}, 0, 4, -1, -1) // старт
	assertSnapshot(t, frames[1], []int{1, 3, 5, 7, 9}, 0, 4, 2, -1)  // сравнение с серединой
	assertSnapshot(t, frames[2], []int{1, 3, 5, 7, 9}, 3, 4, -1, -1) // сдвиг границ вправо
	assertSnapshot(t, frames[3], []int{1, 3, 5, 7, 9}, 3, 4, 3, -1)  // сравнение с серединой
	assertSnapshot(t, frames[4], []int{1, 3, 5, 7, 9}, 3, 4, -1, 3)  // найдено

	// Пояснения отражают смысл шага.
	if !strings.Contains(frames[2].Explanation, "правой половине") {
		t.Errorf("пояснение шага 3 = %q, ожидается упоминание правой половины", frames[2].Explanation)
	}
	if !strings.Contains(frames[4].Explanation, "найдено") {
		t.Errorf("пояснение шага 5 = %q, ожидается «найдено»", frames[4].Explanation)
	}
	// Шаги нумеруются с 1 и идут по порядку.
	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидается %d", i, f.Step, i+1)
		}
	}
}

func TestRunBinarySearchNotFound(t *testing.T) {
	frames, _ := doRunSSE(t, "1,3,5", "4")

	if len(frames) != 6 {
		t.Fatalf("кадров = %d, ожидается 6:\n%v", len(frames), frames)
	}

	last := frames[len(frames)-1]
	assertSnapshot(t, last, []int{1, 3, 5}, 2, 1, -1, -1) // границы пересеклись, итог
	if !strings.Contains(last.Explanation, "нет") {
		t.Errorf("пояснение последнего кадра = %q, ожидается сообщение об отсутствии", last.Explanation)
	}
	// Предпоследний кадр — схлопывание границ.
	assertSnapshot(t, frames[len(frames)-2], []int{1, 3, 5}, 2, 1, -1, -1)
}

func TestRunValidation(t *testing.T) {
	tests := []struct {
		name   string
		values string
		target string
		want   string
	}{
		{"неотсортированный массив", "5,1,3", "3", "отсортирован"},
		{"мусор в values", "1,x,3", "3", "не удалось разобрать"},
		{"мусор в target", "1,3,5", "семь", "не удалось разобрать"},
		{"нет values", "", "3", "values обязателен"},
		{"нет target", "1,3,5", "", "target обязателен"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRun(t, tt.values, tt.target)
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

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта
// совпадают с кадрами, собранными напрямую у рендерера.
func TestRunMatchesDirectRenderer(t *testing.T) {
	frames, _ := doRunSSE(t, "2,4,6,8", "8")
	want := render.BinarySearchFrames([]int{2, 4, 6, 8}, 8)

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

// toWire приводит кадр к виду, в котором его получает клиент (JSON → any).
func toWire(t *testing.T, f frame.Frame) frame.Frame {
	t.Helper()
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("маршалинг кадра: %v", err)
	}
	var out frame.Frame
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("разбор кадра: %v", err)
	}
	return out
}
