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

// doRunLinkedList — запрос Run односвязного списка.
func doRunLinkedList(t *testing.T, script string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/linked-list", nil)
	q := req.URL.Query()
	if script != "" {
		q.Set("script", script)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunLinkedListSSE — успешный Run односвязного списка: базовые проверки
// SSE-ответа + кадры.
func doRunLinkedListSSE(t *testing.T, script string) []frame.Frame {
	t.Helper()

	resp := doRunLinkedList(t, script)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// assertLinkedSnapshot сверяет снимок односвязного списка в кадре: ёмкость =
// числа узлов, head — узел 0, tail — последний (у пустого списка оба -1),
// top не показывается.
func assertLinkedSnapshot(t *testing.T, f frame.Frame, values []int, operation string) {
	t.Helper()

	if f.Kind != frame.KindStructure {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindStructure)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	if f.Data.(map[string]any)["structure"] != render.StructureLinkedList {
		t.Fatalf("кадр шага %d: structure = %v, ожидается %q",
			f.Step, f.Data.(map[string]any)["structure"], render.StructureLinkedList)
	}
	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}
	head, tail := -1, -1
	if len(values) > 0 {
		head, tail = 0, len(values)-1
	}
	want := fmt.Sprintf("values=%v capacity=%d top=%d head=%d tail=%d operation=%q",
		values, len(values), -1, head, tail, operation)
	got := fmt.Sprintf("values=%v capacity=%v top=%v head=%v tail=%v operation=%q",
		s["values"], s["capacity"], s["top"], s["head"], s["tail"], s["operation"])
	if got != want {
		t.Fatalf("кадр шага %d: снимок = %s, ожидается %s\nпояснение: %s", f.Step, got, want, f.Explanation)
	}
}

// Сквозной сценарий тикета: prepend/append (показывают O(1) на концах),
// вставка по позиции с видимым обходом, удаление головы и поиск значения
// — обход узел за узлом виден в кадрах.
func TestRunLinkedListScenario(t *testing.T) {
	frames := doRunLinkedListSSE(t, "prepend 4\nappend 7\ninsert 1 9\nremove 0\nfind 7")
	if len(frames) != 11 {
		t.Fatalf("кадров = %d, ожидается 11:\n%v", len(frames), frames)
	}

	// Старт → prepend → append (связь + вставка) → insert (визит,
	// связь, вставка) → remove 0 → find (два визита + найдено).
	assertLinkedSnapshot(t, frames[0], []int{}, "")                  // старт: пустой список
	assertLinkedSnapshot(t, frames[1], []int{4}, "prepend 4")        // новый узел в голове
	assertLinkedSnapshot(t, frames[2], []int{4}, "append 7")         // связь хвоста 4 → новый узел
	assertLinkedSnapshot(t, frames[3], []int{4, 7}, "append 7")      // узел вставлен после хвоста
	assertLinkedSnapshot(t, frames[4], []int{4, 7}, "insert 1 9")    // визит узла 0 по пути к позиции
	assertLinkedSnapshot(t, frames[5], []int{4, 7}, "insert 1 9")    // связь узла 0 переключена
	assertLinkedSnapshot(t, frames[6], []int{4, 9, 7}, "insert 1 9") // узел 9 встал между 4 и 7
	assertLinkedSnapshot(t, frames[7], []int{9, 7}, "remove 0")      // голова удалена, 9 осталась на месте
	assertLinkedSnapshot(t, frames[8], []int{9, 7}, "find 7")        // визит узла 0 (9)
	assertLinkedSnapshot(t, frames[9], []int{9, 7}, "find 7")        // визит узла 1 (7)
	assertLinkedSnapshot(t, frames[10], []int{9, 7}, "find 7")       // найдено

	// Подсветки ролей: добавлен/перелинкован/посещён/удалён/найден.
	assertListHighlights(t, frames[1], [][2]any{{0, render.RoleAdded}})
	assertListHighlights(t, frames[2], [][2]any{{0, render.RoleRelinked}})
	assertListHighlights(t, frames[3], [][2]any{{1, render.RoleAdded}})
	assertListHighlights(t, frames[4], [][2]any{{0, render.RoleVisited}})
	assertListHighlights(t, frames[5], [][2]any{{0, render.RoleRelinked}})
	assertListHighlights(t, frames[6], [][2]any{{1, render.RoleAdded}})
	assertListHighlights(t, frames[7], [][2]any{{0, render.RoleRemoved}})
	assertListHighlights(t, frames[8], [][2]any{{0, render.RoleVisited}})
	assertListHighlights(t, frames[9], [][2]any{{1, render.RoleVisited}})
	assertListHighlights(t, frames[10], [][2]any{{1, render.RoleFound}})

	// Пояснения отражают учебные точки: O(1) на концах, обход при вставке
	// по позиции и при поиске.
	if !strings.Contains(frames[1].Explanation, "O(1)") {
		t.Errorf("пояснение prepend = %q, ожидается O(1)", frames[1].Explanation)
	}
	if !strings.Contains(frames[6].Explanation, "O(n)") {
		t.Errorf("пояснение insert = %q, ожидается упоминание O(n)", frames[6].Explanation)
	}
	if !strings.Contains(frames[10].Explanation, "найдено на позиции 1") {
		t.Errorf("пояснение find = %q, ожидается found на позиции 1", frames[10].Explanation)
	}
	// Шаги нумеруются с 1 и идут по порядку.
	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидается %d", i, f.Step, i+1)
		}
	}
}

// Поиск отсутствующего значения — успешный промах: Run работает как обычно,
// кадры показывают полный обход и объясняют возврат -1.
func TestRunLinkedListFindMiss(t *testing.T) {
	frames := doRunLinkedListSSE(t, "prepend 4\nfind 9")
	if len(frames) != 4 {
		t.Fatalf("кадров = %d, ожидается 4:\n%v", len(frames), frames)
	}

	assertLinkedSnapshot(t, frames[0], []int{}, "")
	assertLinkedSnapshot(t, frames[1], []int{4}, "prepend 4")
	assertLinkedSnapshot(t, frames[2], []int{4}, "find 9") // единственный узел посещён
	assertLinkedSnapshot(t, frames[3], []int{4}, "find 9") // промах

	assertListHighlights(t, frames[2], [][2]any{{0, render.RoleVisited}})
	// На кадре промаха подсветок нет — обход завершён.
	assertListHighlights(t, frames[3], [][2]any{})
	if !strings.Contains(frames[3].Explanation, "-1") {
		t.Errorf("пояснение промаха = %q, ожидается упоминание -1", frames[3].Explanation)
	}
}

func TestRunLinkedListValidation(t *testing.T) {
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{"нет script", "", "script обязателен"},
		{"пустой сценарий", "\n  \n", "сценарий пуст"},
		{"неизвестная операция", "jump 5", "неизвестная операция"},
		{"prepend без аргумента", "prepend", "prepend требует"},
		{"prepend с лишним аргументом", "prepend 1 4", "prepend требует"},
		{"мусор в значении prepend", "prepend четыре", "не удалось разобрать"},
		{"append без аргумента", "append", "append требует"},
		{"insert без аргументов", "insert 9", "insert требует"},
		{"insert с лишним аргументом", "insert 1 9 7", "insert требует"},
		{"мусор в индексе insert", "insert один 9", "не удалось разобрать"},
		{"мусор в значении insert", "insert 1 девять", "не удалось разобрать"},
		{"insert по индексу за хвостом", "insert 1 9", "вне диапазона"},
		{"отрицательный индекс insert", "insert -1 9", "вне диапазона"},
		{"remove без индекса", "remove", "remove требует"},
		{"remove с лишним аргументом", "remove 0 3", "remove требует"},
		{"мусор в индексе remove", "remove ноль", "не удалось разобрать"},
		{"remove из пустого списка", "remove 0", "пустого списка"},
		{"remove после опустошения", "prepend 4\nremove 0\nremove 0", "пустого списка"},
		{"remove по индексу за хвостом", "prepend 4\nremove 1", "вне диапазона"},
		{"find без аргумента", "find", "find требует"},
		{"find с лишним аргументом", "find 7 5", "find требует"},
		{"мусор в значении find", "find семь", "не удалось разобрать"},
		{"слишком длинный сценарий", strings.Repeat("remove 0\n", 501), "слишком длинный"},
		{"список вырастет за пределы", strings.Repeat("prepend 1\n", 101), "вырастет"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunLinkedList(t, tt.script)
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
func TestRunLinkedListMatchesDirectRenderer(t *testing.T) {
	ops := []render.LinkedOp{
		{Kind: render.LinkedOpPrepend, Value: 4},
		{Kind: render.LinkedOpAppend, Value: 7},
		{Kind: render.LinkedOpInsert, Index: 1, Value: 9},
		{Kind: render.LinkedOpRemove, Index: 0},
		{Kind: render.LinkedOpFind, Value: 7},
		{Kind: render.LinkedOpFind, Value: 13},
	}
	frames := doRunLinkedListSSE(t, "prepend 4\nappend 7\ninsert 1 9\nremove 0\nfind 7\nfind 13")
	want, err := render.LinkedFrames(ops)
	if err != nil {
		t.Fatalf("LinkedFrames: %v", err)
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
