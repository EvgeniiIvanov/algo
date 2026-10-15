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

// doRunHashTable — запрос Run хеш-таблицы.
func doRunHashTable(t *testing.T, buckets, script string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/run/hash-table", nil)
	q := req.URL.Query()
	if buckets != "" {
		q.Set("buckets", buckets)
	}
	if script != "" {
		q.Set("script", script)
	}
	req.URL.RawQuery = q.Encode()

	resp := httptest.NewRecorder()
	NewMux().ServeHTTP(resp, req)

	return resp
}

// doRunHashTableSSE — успешный Run хеш-таблицы: базовые проверки SSE-ответа
// + кадры.
func doRunHashTableSSE(t *testing.T, buckets, script string) []frame.Frame {
	t.Helper()

	resp := doRunHashTable(t, buckets, script)
	assertSSE(t, resp)
	return readSSEFrames(t, resp)
}

// dumpChains — строки «кN=key:value;…» по списку цепочек: плоское зеркало
// содержимого снимка для сравнения после JSON-кругосветки. Расположение
// записей — от головы цепочки.
func dumpChains(buckets [][]render.HashTableEntry) string {
	var b strings.Builder
	for i, chain := range buckets {
		fmt.Fprintf(&b, "к%d=", i)
		for _, e := range chain {
			fmt.Fprintf(&b, "%d:%d;", e.Key, e.Value)
		}
		b.WriteString("|")
	}
	return b.String()
}

// dumpWireChains — то же по снимку, прошедшему JSON-кругосветку (map).
func dumpWireChains(t *testing.T, s map[string]any) string {
	t.Helper()

	raw, ok := s["buckets"].([]any)
	if !ok {
		t.Fatalf("снимок без поля buckets: %#v", s)
	}
	var b strings.Builder
	for i, bucket := range raw {
		fmt.Fprintf(&b, "к%d=", i)
		for _, item := range bucket.([]any) {
			e := item.(map[string]any)
			fmt.Fprintf(&b, "%v:%v;", e["key"], e["value"])
		}
		b.WriteString("|")
	}
	return b.String()
}

// assertHashSnapshot сверяет снимок хеш-таблицы в кадре: число корзин,
// содержимое каждой цепочки (от головы записи), выбранная корзина,
// подсветки записей, число записей и имя операции. Надгробия передаются
// последним опциональным аргументом (одно значение — один HashRemove-кадр);
// без аргумента кадр обязан нести пустой массив removed (`[]`, не null —
// строгий контракт фронтенда требует массив).
func assertHashSnapshot(t *testing.T, f frame.Frame, buckets [][]render.HashTableEntry, highlightBucket int, highlights []render.HashTableHighlight, operation string, removed ...render.HashTableRemoval) {
	t.Helper()

	if len(removed) > 1 {
		t.Fatalf("надгробия передаются одним значением, получено %d аргументов", len(removed))
	}
	wantRemoved := []render.HashTableRemoval{}
	if len(removed) == 1 {
		wantRemoved = removed[0:1]
	}

	if f.Kind != frame.KindHashTable {
		t.Fatalf("кадр шага %d: kind = %q, ожидается %q", f.Step, f.Kind, frame.KindHashTable)
	}
	if f.Explanation == "" {
		t.Fatalf("кадр шага %d: пустое пояснение", f.Step)
	}
	s, ok := f.Data.(map[string]any) // после JSON-кругосветки снимок приходит как map
	if !ok {
		t.Fatalf("кадр шага %d: data не является снимком: %#v", f.Step, f.Data)
	}

	entries := 0
	for _, chain := range buckets {
		entries += len(chain)
	}

	if s["bucketCount"].(float64) != float64(len(buckets)) {
		t.Fatalf("кадр шага %d: bucketCount = %v, ожидалось %d", f.Step, s["bucketCount"], len(buckets))
	}
	if s["entryCount"].(float64) != float64(entries) {
		t.Fatalf("кадр шага %d: entryCount = %v, ожидалось %d", f.Step, s["entryCount"], entries)
	}
	if s["highlightBucket"].(float64) != float64(highlightBucket) {
		t.Fatalf("кадр шага %d: highlightBucket = %v, ожидалось %d", f.Step, s["highlightBucket"], highlightBucket)
	}
	if s["operation"].(string) != operation {
		t.Fatalf("кадр шага %d: operation = %q, ожидалось %q\nпояснение: %s", f.Step, s["operation"], operation, f.Explanation)
	}
	if got, want := dumpWireChains(t, s), dumpChains(buckets); got != want {
		t.Fatalf("кадр шага %d: корзины = %s, ожидалось %s\nпояснение: %s", f.Step, got, want, f.Explanation)
	}

	gotHigh := s["highlights"].([]any)
	if len(gotHigh) != len(highlights) {
		t.Fatalf("кадр шага %d: подсветок %d, ожидалось %d (%v)", f.Step, len(gotHigh), len(highlights), gotHigh)
	}
	for i, h := range highlights {
		hl := gotHigh[i].(map[string]any)
		if hl["bucket"].(float64) != float64(h.Bucket) || hl["position"].(float64) != float64(h.Position) || hl["role"].(string) != h.Role {
			t.Fatalf("кадр шага %d: подсветка %d = %v, ожидается {bucket:%d position:%d role:%q}", f.Step, i, hl, h.Bucket, h.Position, h.Role)
		}
	}

	// Надгробия шага: удалённая запись вне подсветок — её бывшая позиция,
	// ключ и значение проверяются на настоящей проволочной форме кадра.
	// Поле removed обязано быть массивом даже без удаления: JSON null не
	// проходит фронтенд-контракт и ломает каждый реальный прогон.
	gotRemoved, hasRemoved := s["removed"].([]any)
	if !hasRemoved {
		t.Fatalf("кадр шага %d: removed = %v, ожидается JSON-массив (пустой [] на шаге без удаления)", f.Step, s["removed"])
	}
	if len(wantRemoved) == 0 {
		if len(gotRemoved) > 0 {
			t.Fatalf("кадр шага %d: removed не пуст: %v", f.Step, gotRemoved)
		}
		return
	}
	if len(gotRemoved) != len(wantRemoved) {
		t.Fatalf("кадр шага %d: removed = %v, ожидалось %d надгроб(ий/я)", f.Step, s["removed"], len(wantRemoved))
	}
	for i, want := range wantRemoved {
		gone := gotRemoved[i].(map[string]any)
		if gone["bucket"].(float64) != float64(want.Bucket) || gone["position"].(float64) != float64(want.Position) ||
			gone["key"].(float64) != float64(want.Key) || gone["value"].(float64) != float64(want.Value) {
			t.Fatalf("кадр шага %d: надгробие %d = %v, ожидается {bucket:%d position:%d key:%d value:%d}", f.Step, i, gone, want.Bucket, want.Position, want.Key, want.Value)
		}
	}
}

// bucket5 — пять пустых корзин; с одной записанной цепочкой — в корзине 1.
func emptyBuckets(n int) [][]render.HashTableEntry {
	return make([][]render.HashTableEntry, n)
}

// withChain — копия таблицы корзин с перезаписанной цепочкой корзины i.
func withChain(buckets [][]render.HashTableEntry, i int, chain ...render.HashTableEntry) [][]render.HashTableEntry {
	out := make([][]render.HashTableEntry, len(buckets))
	for k := range buckets {
		out[k] = append([]render.HashTableEntry{}, buckets[k]...)
	}
	out[i] = chain
	return out
}

// Сквозной сценарий тикета: коллизия и обход цепочки (put), попадание (get),
// промах get, удаление из цепочки, промах delete — все операции читаются
// в кадрах. Порядок кадров: по кадру на каждое событие (выбор корзины,
// визиты записей, вставка/удаление/результат), старт — один кадр.
func TestRunHashTableScenario(t *testing.T) {
	frames := doRunHashTableSSE(t, "5", "put 1 10\nput 6 20\nget 1\nget 9\ndelete 6\ndelete 9")
	if len(frames) != 17 {
		t.Fatalf("кадров = %d, ожидалось 17:\n%v", len(frames), frames)
	}

	// Состояние корзин по ходу сценария.
	start := emptyBuckets(5)
	afterPut1 := withChain(start, 1, render.HashTableEntry{Key: 1, Value: 10})
	afterPut6 := withChain(start, 1, render.HashTableEntry{Key: 6, Value: 20}, render.HashTableEntry{Key: 1, Value: 10})
	afterDelete6 := withChain(start, 1, render.HashTableEntry{Key: 1, Value: 10})

	assertHashSnapshot(t, frames[0], start, -1, nil, "") // старт: все корзины пусты

	// put 1 10: корзина 1 пуста — выбор корзины и сразу вставка.
	assertHashSnapshot(t, frames[1], start, 1, nil, "put 1 10")
	assertHashSnapshot(t, frames[2], afterPut1, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleAdded}}, "put 1 10")

	// put 6 20: коллизия — визит записи 1, вставка во главу цепочки.
	assertHashSnapshot(t, frames[3], afterPut1, 1, nil, "put 6 20")
	assertHashSnapshot(t, frames[4], afterPut1, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleVisited}}, "put 6 20")
	assertHashSnapshot(t, frames[5], afterPut6, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleAdded}}, "put 6 20")

	// get 1: визиты записей 6 (позиция 0) и 1 (позиция 1), попали.
	assertHashSnapshot(t, frames[6], afterPut6, 1, nil, "get 1")
	assertHashSnapshot(t, frames[7], afterPut6, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleVisited}}, "get 1")
	assertHashSnapshot(t, frames[8], afterPut6, 1, []render.HashTableHighlight{{Bucket: 1, Position: 1, Role: render.RoleVisited}}, "get 1")
	assertHashSnapshot(t, frames[9], afterPut6, 1, []render.HashTableHighlight{{Bucket: 1, Position: 1, Role: render.RoleFound}}, "get 1")

	// get 9: корзина 4 пуста — выбор корзины и сразу промах.
	assertHashSnapshot(t, frames[10], afterPut6, 4, nil, "get 9")
	assertHashSnapshot(t, frames[11], afterPut6, 4, nil, "get 9")

	// delete 6: визит записи (6 — голова цепочки), удаление головы.
	// Регрессия прежнего бага подсветки: удалённой записи в снимке нет,
	// кадр несёт надгробие с её ключом/значением, сосед не красится ролью
	// removed.
	assertHashSnapshot(t, frames[12], afterPut6, 1, nil, "delete 6")
	assertHashSnapshot(t, frames[13], afterPut6, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleVisited}}, "delete 6")
	assertHashSnapshot(t, frames[14], afterDelete6, 1, nil, "delete 6", render.HashTableRemoval{Bucket: 1, Position: 0, Key: 6, Value: 20})

	// delete 9: корзина 4 пуста — выбор и промах.
	assertHashSnapshot(t, frames[15], afterDelete6, 4, nil, "delete 9")
	assertHashSnapshot(t, frames[16], afterDelete6, 4, nil, "delete 9")

	// Пояснения отражают учебные точки: расчёт индекса, коллизию, попадание,
	// штатный промах, целость записей цепочки при удалении.
	if !strings.Contains(frames[1].Explanation, "%") || !strings.Contains(frames[1].Explanation, "корзину 1") {
		t.Errorf("пояснение выбора корзины = %q, ожидается расчёт остатка через %%", frames[1].Explanation)
	}
	if !strings.Contains(frames[5].Explanation, "коллизия") {
		t.Errorf("пояснение вставки при коллизии = %q, ожидается слово «коллизия»", frames[5].Explanation)
	}
	if !strings.Contains(frames[9].Explanation, "найден") {
		t.Errorf("пояснение попадания = %q, ожидается «найден»", frames[9].Explanation)
	}
	if !strings.Contains(frames[11].Explanation, "промах") {
		t.Errorf("пояснение промаха get = %q, ожидается «промах»", frames[11].Explanation)
	}
	if !strings.Contains(frames[14].Explanation, "остались доступны") {
		t.Errorf("пояснение удаления = %q, ожидается, что прочие записи цепочки целы", frames[14].Explanation)
	}
	// Шаги нумеруются с 1 и идут по порядку.
	for i, f := range frames {
		if f.Step != i+1 {
			t.Errorf("кадр %d: step = %d, ожидается %d", i, f.Step, i+1)
		}
	}
}

// Отрицательный ключ: остаток от деления нормализуется — кадр выбора корзины
// показывает нормализацию, запись доступна для поиска.
func TestRunHashTableNegativeKey(t *testing.T) {
	frames := doRunHashTableSSE(t, "5", "put -4 40\nget -4")
	if len(frames) != 6 {
		t.Fatalf("кадров = %d, ожидалось 6:\n%v", len(frames), frames)
	}

	start := emptyBuckets(5)
	afterPut := withChain(start, 1, render.HashTableEntry{Key: -4, Value: 40})

	assertHashSnapshot(t, frames[0], start, -1, nil, "")
	assertHashSnapshot(t, frames[1], start, 1, nil, "put -4 40") // выбор корзины до вставки
	assertHashSnapshot(t, frames[2], afterPut, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleAdded}}, "put -4 40")
	assertHashSnapshot(t, frames[3], afterPut, 1, nil, "get -4")
	assertHashSnapshot(t, frames[4], afterPut, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleVisited}}, "get -4")
	assertHashSnapshot(t, frames[5], afterPut, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleFound}}, "get -4")

	if !strings.Contains(frames[1].Explanation, "нормализуем") || !strings.Contains(frames[1].Explanation, "-4 + 5 = 1") {
		t.Errorf("пояснение нормализации = %q, ожидается «-4 + 5 = 1»", frames[1].Explanation)
	}
}

// Обновление существующего ключа: число записей не растёт — кадр объясняет
// переписывание значения на месте.
func TestRunHashTableUpdate(t *testing.T) {
	frames := doRunHashTableSSE(t, "5", "put 1 10\nput 1 11")
	if len(frames) != 6 {
		t.Fatalf("кадров = %d, ожидалось 6:\n%v", len(frames), frames)
	}

	updated := withChain(emptyBuckets(5), 1, render.HashTableEntry{Key: 1, Value: 11})
	assertHashSnapshot(t, frames[4], withChain(emptyBuckets(5), 1, render.HashTableEntry{Key: 1, Value: 10}), 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleVisited}}, "put 1 11")
	assertHashSnapshot(t, frames[5], updated, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleUpdated}}, "put 1 11")
	if !strings.Contains(frames[5].Explanation, "не изменилось") {
		t.Errorf("пояснение обновления = %q, ожидалась неизменность числа записей", frames[5].Explanation)
	}
}

// Промах delete по непустой, но чужой корзине — легальный шаг сценария.
func TestRunHashTableDeleteMiss(t *testing.T) {
	frames := doRunHashTableSSE(t, "5", "put 1 10\ndelete 3")
	if len(frames) != 5 {
		t.Fatalf("кадров = %d, ожидалось 5:\n%v", len(frames), frames)
	}

	state := withChain(emptyBuckets(5), 1, render.HashTableEntry{Key: 1, Value: 10})
	assertHashSnapshot(t, frames[3], state, 3, nil, "delete 3")
	assertHashSnapshot(t, frames[4], state, 3, nil, "delete 3")
	if !strings.Contains(frames[4].Explanation, "промах") {
		t.Errorf("пояснение delete-промаха = %q, ожидается «промах»", frames[4].Explanation)
	}
}

// Коллизия на промахе: искомый ключ попадает в занятую корзину, вся цепочка
// обходится до конца — визиты каждой записи видны, промах штатен.
func TestRunHashTableGetMissAlongChain(t *testing.T) {
	frames := doRunHashTableSSE(t, "5", "put 1 10\nput 6 20\nget 11")
	if len(frames) != 10 {
		t.Fatalf("кадров = %d, ожидалось 10:\n%v", len(frames), frames)
	}

	chain := withChain(emptyBuckets(5), 1, render.HashTableEntry{Key: 6, Value: 20}, render.HashTableEntry{Key: 1, Value: 10})
	assertHashSnapshot(t, frames[7], chain, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleVisited}}, "get 11")
	assertHashSnapshot(t, frames[8], chain, 1, []render.HashTableHighlight{{Bucket: 1, Position: 1, Role: render.RoleVisited}}, "get 11")
	assertHashSnapshot(t, frames[9], chain, 1, nil, "get 11")
}

func TestRunHashTableValidation(t *testing.T) {
	tests := []struct {
		name    string
		buckets string
		script  string
		want    string
	}{
		{"нет buckets", "", "put 1 10", "buckets обязателен"},
		{"мусор в buckets", "пять", "put 1 10", "не удалось разобрать"},
		{"нулевой buckets", "0", "put 1 10", "от 1 до 100"},
		{"отрицательный buckets", "-3", "put 1 10", "от 1 до 100"},
		{"за пределом buckets", "101", "put 1 10", "от 1 до 100"},
		{"нет script", "5", "", "script обязателен"},
		{"пустой сценарий", "5", "\n  \n", "сценарий пуст"},
		{"неизвестная операция", "5", "set 1 10", "неизвестная операция"},
		{"put без аргументов", "5", "put 1", "put требует"},
		{"put с лишним аргументом", "5", "put 1 10 20", "put требует"},
		{"мусор в ключе put", "5", "put один 10", "не удалось разобрать"},
		{"мусор в значении put", "5", "put 1 десять", "не удалось разобрать"},
		{"get без аргумента", "5", "get", "get требует"},
		{"get с лишним аргументом", "5", "get 1 2", "get требует"},
		{"мусор в ключе get", "5", "get семь", "не удалось разобрать"},
		{"delete без аргумента", "5", "delete", "delete требует"},
		{"delete с лишним аргументом", "5", "delete 1 2", "delete требует"},
		{"мусор в ключе delete", "5", "delete ноль", "не удалось разобрать"},
		{"слишком длинный сценарий", "5", strings.Repeat("put 1 10\n", 501), "слишком длинный"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := doRunHashTable(t, tt.buckets, tt.script)
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

// Регрессия бага подсветки удаления из середины цепочки: роль removed на
// живых записях указывала на выживший сосед, вставший на бывшую позицию
// (или не показывала ничего у хвоста). Теперь кадр удаления несёт
// надгробие с ключом/значением удалённой записи на настоящей проволочной
// форме (снимок после удаления), а соседи остаются неподкрашенными.
func TestRunHashTableRemoveMidChain(t *testing.T) {
	frames := doRunHashTableSSE(t, "5", "put 1 10\nput 6 20\nput -4 40\ndelete 6")
	// Кадры: старт + put 1 (выбор, вставка) + put 6 (выбор, визит, вставка)
	// + put -4 (выбор, два визита, вставка) + delete 6 (выбор, два визита,
	// удаление — всего 14).
	if len(frames) != 14 {
		t.Fatalf("кадров = %d, ожидалось 14:\n%v", len(frames), frames)
	}

	chain := withChain(emptyBuckets(5), 1,
		render.HashTableEntry{Key: -4, Value: 40},
		render.HashTableEntry{Key: 6, Value: 20},
		render.HashTableEntry{Key: 1, Value: 10},
	)
	afterDelete := withChain(emptyBuckets(5), 1,
		render.HashTableEntry{Key: -4, Value: 40},
		render.HashTableEntry{Key: 1, Value: 10},
	)

	// Перед удалением: выбор корзины, визиты записи -4 (позиция 0) и
	// записи 6 (позиция 1 — бывшая позиция удаляемой).
	assertHashSnapshot(t, frames[10], chain, 1, nil, "delete 6")
	assertHashSnapshot(t, frames[11], chain, 1, []render.HashTableHighlight{{Bucket: 1, Position: 0, Role: render.RoleVisited}}, "delete 6")
	assertHashSnapshot(t, frames[12], chain, 1, []render.HashTableHighlight{{Bucket: 1, Position: 1, Role: render.RoleVisited}}, "delete 6")
	// Кадр удаления: в снимке соседей (-4, 1), ни одна не красится removed;
	// надгробие на бывшей позиции 1 несёт удалённую пару 6:20.
	assertHashSnapshot(t, frames[13], afterDelete, 1, nil, "delete 6",
		render.HashTableRemoval{Bucket: 1, Position: 1, Key: 6, Value: 20},
	)
	if !strings.Contains(frames[13].Explanation, "остались доступны") || !strings.Contains(frames[13].Explanation, "ключ 6") {
		t.Errorf("пояснение удаления = %q, ожидается удалённый ключ и целость соседей", frames[13].Explanation)
	}
}

// Точная граница лимита операций: ровно 500 принимается (SSE-поток идёт
// до конца), 501-я операция отвергается до старта потока.
func TestRunHashTableOpsLimitBoundary(t *testing.T) {
	script := func(n int) string {
		// get по разным ключам: каждый — легальный промах, лимит записей не трогается.
		lines := make([]string, 0, n)
		for k := 0; k < n; k++ {
			lines = append(lines, fmt.Sprintf("get %d", k))
		}
		return strings.Join(lines, "\n")
	}

	accepted := script(maxHashTableOps)
	if resp := doRunHashTable(t, "3", accepted); resp.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидается 200: %d операций на границе допустимы", resp.Code, maxHashTableOps)
	}
	// Поток доходит до конца: на операцию — выбор корзины и промах, плюс стартовый кадр.
	frames := doRunHashTableSSE(t, "3", accepted)
	if want := 2*maxHashTableOps + 1; len(frames) != want {
		t.Fatalf("кадров = %d, ожидалось %d", len(frames), want)
	}

	over := script(maxHashTableOps + 1)
	resp := doRunHashTable(t, "3", over)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("статус = %d, ожидается 400: %d операций за границей", resp.Code, maxHashTableOps+1)
	}
	var got map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело ошибки не JSON: %v\n%s", err, resp.Body.String())
	}
	if !strings.Contains(got["error"], fmt.Sprintf("максимум %d", maxHashTableOps)) {
		t.Errorf("ошибка = %q, ожидается упоминание максимума %d", got["error"], maxHashTableOps)
	}
}

// Лимит одновременно хранимых записей: 101-й уникальный ключ — ошибка ввода
// до Run; 100 — граница допустимого (уникальным ключом считаем значение,
// повторный put того же ключа места не занимает).
func TestRunHashTableEntriesLimit(t *testing.T) {
	ops := func(n int) string {
		lines := make([]string, 0, n)
		for k := 0; k < n; k++ {
			lines = append(lines, fmt.Sprintf("put %d 1", k*3))
		}
		return strings.Join(lines, "\n")
	}

	resp := doRunHashTable(t, "100", ops(101))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("статус = %d, ожидается 400", resp.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело ошибки не JSON: %v\n%s", err, resp.Body.String())
	}
	if !strings.Contains(got["error"], "максимум 100") {
		t.Errorf("ошибка = %q, ожидается упоминание максимума записей", got["error"])
	}

	// Граница: ровно 100 уникальных ключей + повторные их put — валиден
	// (повторный put обновляет, места не занимая).
	if resp := doRunHashTable(t, "100", ops(100)+"\nput 0 2\ndelete 0"); resp.Code != http.StatusOK {
		t.Fatalf("статус = %d, ожидается 200: сценарий на границе лимита допустим", resp.Code)
	}
}

// Рендерер и транспорт не должны расходиться: кадры из эндпоинта совпадают
// с кадрами, собранными напрямую у рендерера.
func TestRunHashTableMatchesDirectRenderer(t *testing.T) {
	ops := []render.HashTableOp{
		{Kind: render.HashTableOpPut, Key: 1, Value: 10},
		{Kind: render.HashTableOpPut, Key: 6, Value: 20},
		{Kind: render.HashTableOpGet, Key: 1},
		{Kind: render.HashTableOpGet, Key: 9},
		{Kind: render.HashTableOpDelete, Key: 6},
		{Kind: render.HashTableOpDelete, Key: 9},
	}
	frames := doRunHashTableSSE(t, "5", "put 1 10\nput 6 20\nget 1\nget 9\ndelete 6\ndelete 9")
	want, err := render.HashTableFrames(5, ops)
	if err != nil {
		t.Fatalf("HashTableFrames: %v", err)
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
			t.Errorf("кадр %d отличается:\nэндпоинт %s\nрендерер %s", i, aw, bw)
		}
	}
}

// Регрессия проволочной формы: nil-надгробия сериализовались как JSON null,
// а фронтенд требует массив — каждый реальный прогон падал на первом же
// кадре. Здесь настоящий SSE-кадр (data-строка потока) разбирается как
// JSON-документ: поле removed обязано присутствовать всегда — пустой `[]`
// на шагах без удаления и надгробие в кадре удаления.
func TestRunHashTableWireRemovedArray(t *testing.T) {
	resp := doRunHashTable(t, "5", "put 1 10\ndelete 1")
	assertSSE(t, resp)

	type wireFrame struct {
		Step int            `json:"step"`
		Data map[string]any `json:"data"`
	}
	var frames []wireFrame
	scan := bufio.NewScanner(bytes.NewReader(resp.Body.Bytes()))
	for scan.Scan() {
		payload, ok := strings.CutPrefix(scan.Text(), "data: ")
		if !ok {
			continue // пустые строки и служебные поля SSE не несут кадра
		}
		var f wireFrame
		if err := json.Unmarshal([]byte(payload), &f); err != nil {
			t.Fatalf("кадр не разбирается как JSON: %v\n%s", err, payload)
		}
		frames = append(frames, f)
	}
	if err := scan.Err(); err != nil {
		t.Fatalf("чтение SSE-потока: %v", err)
	}

	// put 1 10: старт, выбор корзины, вставка; delete 1: выбор, визит,
	// удаление — всего 6 кадров.
	if len(frames) != 6 {
		t.Fatalf("кадров = %d, ожидалось 6", len(frames))
	}

	removedOf := func(step int, data map[string]any) []any {
		t.Helper()
		raw, present := data["removed"]
		if !present {
			t.Fatalf("кадр шага %d: в снимке нет поля removed: %v", step, data)
		}
		arr, ok := raw.([]any)
		if !ok {
			t.Fatalf("кадр шага %d: removed = %v, ожидается JSON-массив (не null)", step, raw)
		}
		return arr
	}

	// Шаги без удаления — стартовый, выбор корзины, вставка, визит — несут
	// removed как пустой массив, не null и не отсутствующее поле.
	for _, i := range []int{0, 1, 2, 3, 4} {
		got := removedOf(frames[i].Step, frames[i].Data)
		if len(got) != 0 {
			t.Fatalf("кадр шага %d: removed = %v, ожидается пустой массив", frames[i].Step, got)
		}
	}

	// Кадр удаления по-прежнему несёт надгробие в том же поле removed.
	got := removedOf(frames[5].Step, frames[5].Data)
	if len(got) != 1 {
		t.Fatalf("кадр шага %d: removed = %v, ожидалось 1 надгробие", frames[5].Step, got)
	}
	gone := got[0].(map[string]any)
	if gone["bucket"].(float64) != 1 || gone["position"].(float64) != 0 ||
		gone["key"].(float64) != 1 || gone["value"].(float64) != 10 {
		t.Fatalf("кадр шага %d: надгробие = %v, ожидается {bucket:1 position:0 key:1 value:10}", frames[5].Step, gone)
	}
}
