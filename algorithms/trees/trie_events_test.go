package trees_test

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
)

// spyTrieEmitter — тестовый Emitter: собирает события Trie в слайс
// для проверки последовательности шагов.
type spyTrieEmitter []trees.TrieEvent

func (s *spyTrieEmitter) Emit(event trees.TrieEvent) { *s = append(*s, event) }

// bPtr — указатель на bool для проверки TrieVisit.EndFlag (nil — флаг не менялся).
func bPtr(v bool) *bool { return &v }

// wantEvents сравнивает собранные события с ожидаемой последовательностью.
func wantEvents(t *testing.T, spy spyTrieEmitter, want []trees.TrieEvent, context string) {
	t.Helper()
	if !reflect.DeepEqual([]trees.TrieEvent(spy), want) {
		t.Fatalf("события %s:\n got: %v\nwant: %v", context, []trees.TrieEvent(spy), want)
	}
}

func TestTrieEventsInsertIntoEmpty(t *testing.T) {
	trie := trees.NewTrie()
	var spy spyTrieEmitter

	inserted, err := trie.InsertWithEmitter("аб", &spy)
	if !inserted || err != nil {
		t.Fatalf(`InsertWithEmitter("аб") = (%v, %v), ожидалось (true, nil)`, inserted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		// Вставка в пустой Trie: каждое ребро создаётся.
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeCreated},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeCreated, EndFlag: bPtr(true)},
		trees.TrieInserted{},
	}, `Insert("аб") в пустой Trie`)
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
}

func TestTrieEventsInsertSharedPrefix(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}
	var spy spyTrieEmitter

	inserted, err := trie.InsertWithEmitter("абд", &spy)
	if !inserted || err != nil {
		t.Fatalf(`InsertWithEmitter("абд") = (%v, %v), ожидалось (true, nil)`, inserted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		// Общий префикс «аб» переиспользуется, новое ребро только по «д».
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'д', Edge: trees.EdgeCreated, EndFlag: bPtr(true)},
		trees.TrieInserted{},
	}, `Insert("абд") по общему префиксу`)
	if got := trie.Len(); got != 2 {
		t.Fatalf("Len() = %d, ожидалось 2", got)
	}
}

func TestTrieEventsDuplicateInsert(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("аб"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}
	var spy spyTrieEmitter

	inserted, err := trie.InsertWithEmitter("аб", &spy)
	if inserted || err != nil {
		t.Fatalf("повторный InsertWithEmitter = (%v, %v), ожидалось (false, nil)", inserted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		// Повторная вставка: посещения на все руны (reused), флаг конца слова
		// уже установлен и не меняется, pruning нет.
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TrieAlreadyExisted{},
	}, `повторный Insert("аб")`)
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
}

func TestTrieEventsSearchFound(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("аб"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}
	var spy spyTrieEmitter

	found, err := trie.SearchWithEmitter("аб", &spy)
	if !found || err != nil {
		t.Fatalf(`SearchWithEmitter("аб") = (%v, %v), ожидалось (true, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TrieFound{},
	}, `Search("аб")`)
}

func TestTrieEventsSearchMissMissingEdge(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}
	var spy spyTrieEmitter

	found, err := trie.SearchWithEmitter("аг", &spy)
	if found || err != nil {
		t.Fatalf(`SearchWithEmitter("аг") = (%v, %v), ожидалось (false, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		// Промах по отсутствующему ребру: проверки до первой отсутствующей
		// руны включительно, последнее событие — missing с проверяемой руной.
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'г', Edge: trees.EdgeMissing},
		trees.TrieNotFound{},
	}, `Search("аг") по отсутствующему ребру`)
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
}

func TestTrieEventsSearchMissExistingPath(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}
	var spy spyTrieEmitter

	// Существующий путь, конечный узел не отмечен концом слова: посещения на
	// все руны слова и один результат, без мутаций.
	found, err := trie.SearchWithEmitter("аб", &spy)
	if found || err != nil {
		t.Fatalf(`SearchWithEmitter("аб") = (%v, %v), ожидалось (false, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TrieNotFound{},
	}, `Search("аб") по существующему пути без конца слова`)
}

func TestTrieEventsStartsWith(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}

	var spy spyTrieEmitter
	found, err := trie.StartsWithWithEmitter("аб", &spy)
	if !found || err != nil {
		t.Fatalf(`StartsWithWithEmitter("аб") = (%v, %v), ожидалось (true, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TriePrefixFound{},
	}, `StartsWith("аб")`)

	spy = nil
	found, err = trie.StartsWithWithEmitter("аг", &spy)
	if found || err != nil {
		t.Fatalf(`StartsWithWithEmitter("аг") = (%v, %v), ожидалось (false, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'г', Edge: trees.EdgeMissing},
		trees.TriePrefixNotFound{},
	}, `StartsWith("аг")`)
}

func TestTrieEventsEmptyQueries(t *testing.T) {
	// Даже в пустом Trie пустые запросы валидны: ровно одно событие
	// результата, событий посещения нет.
	empty := trees.NewTrie()

	var spy spyTrieEmitter
	found, err := empty.SearchWithEmitter("", &spy)
	if found || err != nil {
		t.Fatalf(`SearchWithEmitter("") = (%v, %v), ожидалось (false, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{trees.TrieNotFound{}}, `Search("") в пустом Trie`)

	spy = nil
	found, err = empty.StartsWithWithEmitter("", &spy)
	if !found || err != nil {
		t.Fatalf(`StartsWithWithEmitter("") = (%v, %v), ожидалось (true, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{trees.TriePrefixFound{}}, `StartsWith("") в пустом Trie`)
}

func TestTrieEventsErrorsBeforeEvents(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}

	// Ошибочные вызовы возвращают ошибку до любых событий: ни посещений,
	// ни результата, Trie и Len не меняются.
	var spy spyTrieEmitter
	if _, err := trie.InsertWithEmitter("", &spy); err == nil {
		t.Fatalf(`InsertWithEmitter("") err = nil, ожидалась ошибка`)
	}
	wantEvents(t, spy, nil, `Insert("")`)
	if _, err := trie.InsertWithEmitter(invalidUTF8, &spy); err == nil {
		t.Fatalf("InsertWithEmitter с невалидным UTF-8 err = nil, ожидалась ошибка")
	}
	wantEvents(t, spy, nil, "Insert с невалидным UTF-8")

	spy = nil
	if _, err := trie.DeleteWithEmitter("", &spy); err == nil {
		t.Fatalf(`DeleteWithEmitter("") err = nil, ожидалась ошибка`)
	}
	wantEvents(t, spy, nil, `Delete("")`)
	if _, err := trie.DeleteWithEmitter(invalidUTF8, &spy); err == nil {
		t.Fatalf("DeleteWithEmitter с невалидным UTF-8 err = nil, ожидалась ошибка")
	}
	wantEvents(t, spy, nil, "Delete с невалидным UTF-8")

	spy = nil
	if _, err := trie.SearchWithEmitter(invalidUTF8, &spy); err == nil {
		t.Fatalf("SearchWithEmitter с невалидным UTF-8 err = nil, ожидалась ошибка")
	}
	wantEvents(t, spy, nil, "Search с невалидным UTF-8")
	if _, err := trie.StartsWithWithEmitter(invalidUTF8, &spy); err == nil {
		t.Fatalf("StartsWithWithEmitter с невалидным UTF-8 err = nil, ожидалась ошибка")
	}
	wantEvents(t, spy, nil, "StartsWith с невалидным UTF-8")

	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
}

func TestTrieEventsDeleteWithPruning(t *testing.T) {
	trie := trees.NewTrie()
	for _, w := range []string{"абв", "абд"} {
		if _, err := trie.Insert(w); err != nil {
			t.Fatalf("Insert(%q) err = %v, ожидалось nil", w, err)
		}
	}
	var spy spyTrieEmitter

	// Удаление на ветвлении: флаг конца слова снимается в последнем событии
	// посещения; pruning срезает только ставший ненужным узел «в».
	deleted, err := trie.DeleteWithEmitter("абв", &spy)
	if !deleted || err != nil {
		t.Fatalf(`DeleteWithEmitter("абв") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'в', Edge: trees.EdgeReused, EndFlag: bPtr(false)},
		trees.TriePrune{Rune: 'в', Depth: 3},
		trees.TrieDeleted{},
	}, `Delete("абв") на ветвлении`)
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}

	// Удаление последнего слова счищает всю ветвь до корня: по одному
	// событию на каждый срезанный узел, снизу вверх.
	spy = nil
	deleted, err = trie.DeleteWithEmitter("абд", &spy)
	if !deleted || err != nil {
		t.Fatalf(`DeleteWithEmitter("абд") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'д', Edge: trees.EdgeReused, EndFlag: bPtr(false)},
		trees.TriePrune{Rune: 'д', Depth: 3},
		trees.TriePrune{Rune: 'б', Depth: 2},
		trees.TriePrune{Rune: 'а', Depth: 1},
		trees.TrieDeleted{},
	}, `Delete("абд") с прунингом до корня`)
	if got := trie.Len(); got != 0 {
		t.Fatalf("Len() = %d, ожидалось 0", got)
	}
}

func TestTrieEventsDeletePrefixWord(t *testing.T) {
	trie := trees.NewTrie()
	for _, w := range []string{"аб", "абв"} {
		if _, err := trie.Insert(w); err != nil {
			t.Fatalf("Insert(%q) err = %v, ожидалось nil", w, err)
		}
	}
	var spy spyTrieEmitter

	// Удаление слова-префикса: флаг снимается, pruning нет — узел «б» ещё
	// ведёт к слову «абв».
	deleted, err := trie.DeleteWithEmitter("аб", &spy)
	if !deleted || err != nil {
		t.Fatalf(`DeleteWithEmitter("аб") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused, EndFlag: bPtr(false)},
		trees.TrieDeleted{},
	}, `Delete("аб") — слово-префикс`)
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("абв"); !found || err != nil {
		t.Fatalf("Search оставшегося слова = (%v, %v), ожидалось (true, nil)", found, err)
	}
}

func TestTrieEventsDeleteMisses(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}

	// Промах по отсутствующему ребру: проверки до первой отсутствующей руны
	// включительно и один результат not-deleted, без мутаций и pruning.
	var spy spyTrieEmitter
	deleted, err := trie.DeleteWithEmitter("аг", &spy)
	if deleted || err != nil {
		t.Fatalf(`DeleteWithEmitter("аг") = (%v, %v), ожидалось (false, nil)`, deleted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'г', Edge: trees.EdgeMissing},
		trees.TrieNotDeleted{},
	}, `Delete("аг") по отсутствующему ребру`)

	// Существующий путь без отметки конца слова: посещения на все руны
	// (без флага конца слова), один результат, без мутаций и pruning.
	spy = nil
	deleted, err = trie.DeleteWithEmitter("аб", &spy)
	if deleted || err != nil {
		t.Fatalf(`DeleteWithEmitter("аб") = (%v, %v), ожидалось (false, nil)`, deleted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: 'а', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeReused},
		trees.TrieNotDeleted{},
	}, `Delete("аб") по существующему пути без конца слова`)
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("абв"); !found || err != nil {
		t.Fatalf("Search после промахов = (%v, %v), ожидалось (true, nil)", found, err)
	}
}

func TestTrieEventsOnePerRune(t *testing.T) {
	trie := trees.NewTrie()
	var spy spyTrieEmitter

	// Слово «😀б» — 5 UTF-8 байтов, но ровно 2 руны: по одному событию
	// посещения на каждую руну, включая руну вне BMP.
	word := "😀б"
	inserted, err := trie.InsertWithEmitter(word, &spy)
	if !inserted || err != nil {
		t.Fatalf("InsertWithEmitter(%q) = (%v, %v), ожидалось (true, nil)", word, inserted, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: '😀', Edge: trees.EdgeCreated},
		trees.TrieVisit{Rune: 'б', Edge: trees.EdgeCreated, EndFlag: bPtr(true)},
		trees.TrieInserted{},
	}, "Insert символа вне BMP")
	if runes := utf8.RuneCountInString(word); runes != 2 {
		t.Fatalf("RuneCountInString(%q) = %d, ожидалось 2", word, runes)
	}

	// Промах по символу вне BMP — тоже одно событие проверки на руну.
	spy = nil
	found, err := trie.StartsWithWithEmitter("😀в", &spy)
	if found || err != nil {
		t.Fatalf(`StartsWithWithEmitter("😀в") = (%v, %v), ожидалось (false, nil)`, found, err)
	}
	wantEvents(t, spy, []trees.TrieEvent{
		trees.TrieVisit{Rune: '😀', Edge: trees.EdgeReused},
		trees.TrieVisit{Rune: 'в', Edge: trees.EdgeMissing},
		trees.TriePrefixNotFound{},
	}, "промах по символу вне BMP")
}

func TestTrieEventsSameBehaviorAsQuiet(t *testing.T) {
	// Тихий режим и режим наблюдения — одна семантика: одинаковые результаты
	// и одинаковое итоговое состояние.
	ops := []func(trie *trees.Trie) (bool, error){
		func(t *trees.Trie) (bool, error) { return t.Insert("абв") },
		func(t *trees.Trie) (bool, error) { return t.Insert("аб") },
		func(t *trees.Trie) (bool, error) { return t.Insert("аб") },
		func(t *trees.Trie) (bool, error) { return t.Search("аб") },
		func(t *trees.Trie) (bool, error) { return t.Search("абг") },
		func(t *trees.Trie) (bool, error) { return t.StartsWith("а") },
		func(t *trees.Trie) (bool, error) { return t.StartsWith("б") },
		func(t *trees.Trie) (bool, error) { return t.Delete("абв") },
		func(t *trees.Trie) (bool, error) { return t.Delete("абв") },
		func(t *trees.Trie) (bool, error) { return t.Search("аб") },
	}

	quiet := trees.NewTrie()
	observed := trees.NewTrie()
	for i, op := range ops {
		qGot, qErr := op(quiet)
		oGot, oErr := op(observed)
		if qGot != oGot || !errEqual(qErr, oErr) {
			t.Fatalf("операция %d: quiet = (%v, %v), observed = (%v, %v)", i, qGot, qErr, oGot, oErr)
		}
	}
	if quiet.Len() != observed.Len() {
		t.Fatalf("Len: quiet = %d, observed = %d", quiet.Len(), observed.Len())
	}
	if got := quiet.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
}

func errEqual(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Error() == b.Error()
}
