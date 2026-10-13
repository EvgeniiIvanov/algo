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

// doRunList — запрос Run списка.
func doRunList(t *testing.T, script string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/list", nil)
	q := req.URL.Query()
	if script != "" {
		q.Set("script", script)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunListSSE — успешный Run списка: базовые проверки SSE-ответа + кадры.
func doRunListSSE(t *testing.T, script string) []frame.Frame {
	t.Helper()

	resp := doRunList(t, script)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// assertListSnapshot сверяет снимок списка в кадре: у списка ёмкость =
// текущей длине, а указатели всех структур стоят в -1.
func assertListSnapshot(t *testing.T, f frame.Frame, values []int, operation string) {
	t.Helper()

	if f.Kind != frame.KindStructure {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindStructure)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	if f.Data.(map[string]any)["structure"] != render.StructureList {
		t.Fatalf("кадр шага %d: structure = %v, ожидается %q", f.Step, f.Data.(map[string]any)["structure"], render.StructureList)
	}
	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	want := fmt.Sprintf("values=%v capacity=%d top=%d head=%d tail=%d operation=%q",
		values, len(values), -1, -1, -1, operation)
	got := fmt.Sprintf("values=%v capacity=%v top=%v head=%v tail=%v operation=%q",
		s["values"], s["capacity"], s["top"], s["head"], s["tail"], s["operation"])
	if got != want {
		t.Fatalf("кадр шага %d: снимок = %s, ожидается %s\nпояснение: %s", f.Step, got, want, f.Explanation)
	}
}

// Сквозной сценарий: вставка в пустой список, в голову, в середину со сдвигом,
// в хвост и удаление из середины — каждый шаг своим кадром.
func TestRunListScenario(t *testing.T) {
	frames := doRunListSSE(t, "insert 0 5\ninsert 0 3\ninsert 1 9\ninsert 3 7\nremove 1")
	if len(frames) != 6 {
		t.Fatalf("кадров = %d, ожидается 6:\n%v", len(frames), frames)
	}

	assertListSnapshot(t, frames[0], []int{}, "")                     // старт: пустой список
	assertListSnapshot(t, frames[1], []int{5}, "insert 0 5")          // вставка в пустой список (он же хвост)
	assertListSnapshot(t, frames[2], []int{3, 5}, "insert 0 3")       // вставка в голову
	assertListSnapshot(t, frames[3], []int{3, 9, 5}, "insert 1 9")    // середина со сдвигом 5 вправо
	assertListSnapshot(t, frames[4], []int{3, 9, 5, 7}, "insert 3 7") // вставка в хвост, без сдвига
	assertListSnapshot(t, frames[5], []int{3, 5, 7}, "remove 1")      // удалили 9, сдвиг [5 7] влево

	// Подсветки ролей: добавленная, сдвинувшиеся, освободившаяся ячейки.
	assertHighlight(t, frames[1], 0, render.RoleAdded)
	assertListHighlights(t, frames[2], [][2]any{{0, render.RoleAdded}, {1, render.RoleShifted}})
	assertListHighlights(t, frames[3], [][2]any{{1, render.RoleAdded}, {2, render.RoleShifted}})
	assertHighlight(t, frames[4], 3, render.RoleAdded) // хвост записан, но сдвига нет
	// Удаление из середины: значения правее уехали влево (две shifted),
	// освободилась дальняя ячейка.
	assertListHighlights(t, frames[5], [][2]any{
		{1, render.RoleShifted}, {2, render.RoleShifted}, {3, render.RoleRemoved},
	})

	// Пояснения отражают смысл шага.
	if !strings.Contains(frames[3].Explanation, "сдвинулись вправо") {
		t.Errorf("пояснение шага 4 = %q, ожидается упоминание сдвига вправо", frames[3].Explanation)
	}
	if !strings.Contains(frames[4].Explanation, "сдвигов нет") {
		t.Errorf("пояснение шага 5 = %q, ожидается указание, что сдвигов нет", frames[4].Explanation)
	}
	if !strings.Contains(frames[5].Explanation, "удалили 9") || !strings.Contains(frames[5].Explanation, "влево") {
		t.Errorf("пояснение шага 6 = %q, ожидается «удалили 9» и сдвиг влево", frames[5].Explanation)
	}
	// Шаги нумеруются с 1 и идут по порядку.
	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидается %d", i, f.Step, i+1)
		}
	}
}

// assertListHighlights сверяет подсветки кадра: точная последовательность
// пар {индекс, роль} в порядке следования.
func assertListHighlights(t *testing.T, f frame.Frame, want [][2]any) {
	t.Helper()

	s := f.Data.(map[string]any)
	highlights, ok := s["highlights"].([]any)
	if !ok {
		t.Fatalf("кадр шага %d: highlights не массив: %#v", f.Step, s["highlights"])
	}
	if len(highlights) != len(want) {
		t.Fatalf("кадр шага %d: подсветок = %d, ожидается %d (%v)",
			f.Step, len(highlights), len(want), want)
	}
	for i, h := range highlights {
		hm := h.(map[string]any)
		// После JSON-кругосветки индекс приходит как float64,
		// в ожиданиях — целая константа: сравниваем по значению числа.
		if !equalNumbers(hm["index"], want[i][0]) || hm["role"] != want[i][1] {
			t.Fatalf("кадр шага %d: подсветка %d = (%v, %v), ожидается (%v, %v)",
				f.Step, i, hm["index"], hm["role"], want[i][0], want[i][1])
		}
	}
}

// equalNumbers — сравнение чисел, пришедших как int или float64.
func equalNumbers(a, b any) bool {
	af, aok := a.(float64)
	bf, bok := b.(float64)
	if aok && bok {
		return af == bf
	}
	ai, aok := a.(int)
	bi, bok := b.(int)
	if aok && bok {
		return ai == bi
	}
	if aok {
		return float64(ai) == bf
	}
	if bok {
		return af == float64(bi)
	}
	return false
}

func TestRunListValidation(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{"нет script", "", "script обязателен"},
		{"пустой сценарий", "\n  \n", "сценарий пуст"},
		{"неизвестная операция", "jump 5", "неизвестная операция"},
		{"insert без аргументов", "insert 5", "insert требует"},
		{"insert с лишним аргументом", "insert 1 9 7", "insert требует"},
		{"мусор в индексе insert", "insert один 9", "не удалось разобрать"},
		{"мусор в значении insert", "insert 1 девять", "не удалось разобрать"},
		{"remove без индекса", "remove", "remove требует"},
		{"remove с лишним аргументом", "remove 0 3", "remove требует"},
		{"мусор в индексе remove", "remove ноль", "не удалось разобрать"},
		{"insert по индексу за хвостом", "insert 3 9", "вне диапазона"},
		{"отрицательный индекс insert", "insert -1 9", "вне диапазона"},
		{"remove из пустого списка", "remove 0", "пустого списка"},
		{"remove после опустошения", "insert 0 5\nremove 0\nremove 0", "пустого списка"},
		{"remove по индексу за хвостом", "insert 0 5\nremove 1", "вне диапазона"},
		{"слишком длинный сценарий", strings.Repeat("remove 0\n", 501), "слишком длинный"},
		{"список вырастет за пределы", strings.Repeat("insert 0 1\n", 101), "вырастет"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunList(t, tt.script)
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
func TestRunListMatchesDirectRenderer(t *testing.T) {
	ops := []render.ListOp{
		{IsInsert: true, Index: 0, Value: 5},
		{IsInsert: true, Index: 0, Value: 3},
		{IsInsert: true, Index: 1, Value: 9},
		{IsInsert: false, Index: 1},
		{IsInsert: true, Index: 2, Value: 7},
	}
	frames := doRunListSSE(t, "insert 0 5\ninsert 0 3\ninsert 1 9\nremove 1\ninsert 2 7")
	want, err := render.ListFrames(ops)
	if err != nil {
		t.Fatalf("ListFrames: %v", err)
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
