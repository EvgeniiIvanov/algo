package trees_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
)

// Неверный UTF-8 — один и тот же байт во всех операциях.
var invalidUTF8 = string([]byte{0xFF})

func TestTrieEmptyTrie(t *testing.T) {
	trie := trees.NewTrie()

	if got := trie.Len(); got != 0 {
		t.Fatalf("Len() = %d, ожидалось 0", got)
	}
	if found, err := trie.Search("а"); found || err != nil {
		t.Fatalf("Search в пустом Trie = (%v, %v), ожидалось (false, nil)", found, err)
	}
	// Пустой префикс соответствует корню — true даже в пустом Trie.
	if found, err := trie.StartsWith(""); !found || err != nil {
		t.Fatalf(`StartsWith("") в пустом Trie = (%v, %v), ожидалось (true, nil)`, found, err)
	}
	if found, err := trie.StartsWith("а"); found || err != nil {
		t.Fatalf("StartsWith в пустом Trie = (%v, %v), ожидалось (false, nil)", found, err)
	}
	if deleted, err := trie.Delete("а"); deleted || err != nil {
		t.Fatalf("Delete в пустом Trie = (%v, %v), ожидалось (false, nil)", deleted, err)
	}
}

func TestTrieInsertAndSearch(t *testing.T) {
	trie := trees.NewTrie()

	inserted, err := trie.Insert("а")
	if !inserted || err != nil {
		t.Fatalf(`Insert("а") = (%v, %v), ожидалось (true, nil)`, inserted, err)
	}
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}

	// Повторная вставка идемпотентна: слово не добавлено, Trie и Len не меняются.
	inserted, err = trie.Insert("а")
	if inserted || err != nil {
		t.Fatalf("повторный Insert = (%v, %v), ожидалось (false, nil)", inserted, err)
	}
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len после повторной вставки = %d, ожидалось 1", got)
	}

	if found, err := trie.Search("а"); !found || err != nil {
		t.Fatalf(`Search("а") = (%v, %v), ожидалось (true, nil)`, found, err)
	}
	// Существующий путь без отметки конца слова — не слово.
	if found, err := trie.Search(""); found || err != nil {
		t.Fatalf(`Search("") = (%v, %v), ожидалось (false, nil)`, found, err)
	}
}

func TestTrieBranchingAndSharedPrefixes(t *testing.T) {
	trie := trees.NewTrie()
	for _, w := range []string{"абв", "абд", "аб", "а"} {
		if inserted, err := trie.Insert(w); !inserted || err != nil {
			t.Fatalf("Insert(%q) = (%v, %v), ожидалось (true, nil)", w, inserted, err)
		}
	}
	if got := trie.Len(); got != 4 {
		t.Fatalf("Len() = %d, ожидалось 4", got)
	}
	for _, w := range []string{"абв", "абд", "аб", "а"} {
		if found, err := trie.Search(w); !found || err != nil {
			t.Fatalf("Search(%q) = (%v, %v), ожидалось (true, nil)", w, found, err)
		}
	}
	// Общие префиксы — существующие пути, но не слова, если не сохранены.
	for _, p := range []string{"абвг", "аг"} {
		if found, err := trie.Search(p); found || err != nil {
			t.Fatalf("Search(%q) = (%v, %v), ожидалось (false, nil)", p, found, err)
		}
	}
	// Префиксы существуют как пути, независимо от отметки конца слова.
	for _, p := range []string{"а", "аб", "абв", ""} {
		if found, err := trie.StartsWith(p); !found || err != nil {
			t.Fatalf(`StartsWith(%q) = (%v, %v), ожидалось (true, nil)`, p, found, err)
		}
	}
	if found, err := trie.StartsWith("аг"); found || err != nil {
		t.Fatalf(`StartsWith("аг") = (%v, %v), ожидалось (false, nil)`, found, err)
	}
}

func TestTrieEmptyWord(t *testing.T) {
	trie := trees.NewTrie()

	if _, err := trie.Insert(""); !errors.Is(err, trees.ErrInsertEmptyWord) {
		t.Fatalf(`Insert("") err = %v, ожидалось ErrInsertEmptyWord`, err)
	}
	if _, err := trie.Delete(""); !errors.Is(err, trees.ErrDeleteEmptyWord) {
		t.Fatalf(`Delete("") err = %v, ожидалось ErrDeleteEmptyWord`, err)
	}
	if got := trie.Len(); got != 0 {
		t.Fatalf("Len() = %d, ожидалось 0", got)
	}
}

func TestTrieDeletePruning(t *testing.T) {
	trie := trees.NewTrie()
	for _, w := range []string{"абв", "абд"} {
		if _, err := trie.Insert(w); err != nil {
			t.Fatalf("Insert(%q) err = %v, ожидалось nil", w, err)
		}
	}

	// Удаление одного из двух слов на ветвлении: ветвление и соседнее слово остаются.
	if deleted, err := trie.Delete("абв"); !deleted || err != nil {
		t.Fatalf(`Delete("абв") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("абв"); found || err != nil {
		t.Fatalf("Search удалённого слова = (%v, %v), ожидалось (false, nil)", found, err)
	}
	if found, err := trie.Search("абд"); !found || err != nil {
		t.Fatalf("Search оставшегося слова = (%v, %v), ожидалось (true, nil)", found, err)
	}
	if found, err := trie.StartsWith("аб"); !found || err != nil {
		t.Fatalf(`StartsWith("аб") после удаления = (%v, %v), ожидалось (true, nil)`, found, err)
	}

	// Удаление последнего слова счищает всю ставшую ненужной ветвь: от корня
	// пути больше нет.
	if deleted, err := trie.Delete("абд"); !deleted || err != nil {
		t.Fatalf(`Delete("абд") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	if got := trie.Len(); got != 0 {
		t.Fatalf("Len() = %d, ожидалось 0", got)
	}
	if found, err := trie.StartsWith("а"); found || err != nil {
		t.Fatalf(`StartsWith("а") в опустевшем Trie = (%v, %v), ожидалось (false, nil)`, found, err)
	}
}

func TestTrieDeleteWordPrefixOfLonger(t *testing.T) {
	trie := trees.NewTrie()
	for _, w := range []string{"аб", "абв"} {
		if _, err := trie.Insert(w); err != nil {
			t.Fatalf("Insert(%q) err = %v, ожидалось nil", w, err)
		}
	}

	// Слово-префикс более длинного слова: удаляется отметка конца слова,
	// путь к более длинному слову остаётся.
	if deleted, err := trie.Delete("аб"); !deleted || err != nil {
		t.Fatalf(`Delete("аб") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("аб"); found || err != nil {
		t.Fatalf("Search удалённого префикса = (%v, %v), ожидалось (false, nil)", found, err)
	}
	if found, err := trie.Search("абв"); !found || err != nil {
		t.Fatalf("Search более длинного слова = (%v, %v), ожидалось (true, nil)", found, err)
	}
	if found, err := trie.StartsWith("аб"); !found || err != nil {
		t.Fatalf(`StartsWith("аб") = (%v, %v), ожидалось (true, nil)`, found, err)
	}

	// Удаление более длинного слова сохраняет сохранённое слово-префикс.
	if _, err := trie.Insert("аб"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}
	if deleted, err := trie.Delete("абв"); !deleted || err != nil {
		t.Fatalf(`Delete("абв") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("аб"); !found || err != nil {
		t.Fatalf("Search сохранённого префикса = (%v, %v), ожидалось (true, nil)", found, err)
	}
	if found, err := trie.StartsWith("абв"); found || err != nil {
		t.Fatalf(`StartsWith("абв") после pruning = (%v, %v), ожидалось (false, nil)`, found, err)
	}
}

func TestTrieDeleteMisses(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}

	// Промах по отсутствующему ребру — не ошибка, Trie и Len не меняются.
	deleted, err := trie.Delete("аб")
	if deleted || err != nil {
		t.Fatalf(`Delete("аб") = (%v, %v), ожидалось (false, nil)`, deleted, err)
	}
	// Промах по существующему пути без отметки конца слова — тоже.
	deleted, err = trie.Delete("абвг")
	if deleted || err != nil {
		t.Fatalf(`Delete("абвг") = (%v, %v), ожидалось (false, nil)`, deleted, err)
	}
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("абв"); !found || err != nil {
		t.Fatalf("Search после промахов Delete = (%v, %v), ожидалось (true, nil)", found, err)
	}
}

func TestTrieLenAfterSeries(t *testing.T) {
	trie := trees.NewTrie()
	words := []string{"аб", "абв", "б", "ба", "😀", "🍎б"}

	for i, w := range words {
		if _, err := trie.Insert(w); err != nil {
			t.Fatalf("Insert(%q) err = %v, ожидалось nil", w, err)
		}
		if got := trie.Len(); got != i+1 {
			t.Fatalf("Len после вставки %q = %d, ожидалось %d", w, got, i+1)
		}
	}
	if _, err := trie.Insert("аб"); err != nil {
		t.Fatalf("повторный Insert err = %v, ожидалось nil", err)
	}
	if got := trie.Len(); got != len(words) {
		t.Fatalf("Len после дубликата = %d, ожидалось %d", got, len(words))
	}

	for _, w := range []string{"аб", "😀", "ба"} {
		if _, err := trie.Delete(w); err != nil {
			t.Fatalf("Delete(%q) err = %v, ожидалось nil", w, err)
		}
	}
	if got := trie.Len(); got != len(words)-3 {
		t.Fatalf("Len после удалений = %d, ожидалось %d", got, len(words)-3)
	}
}

func TestTrieInvalidUTF8(t *testing.T) {
	trie := trees.NewTrie()
	if _, err := trie.Insert("абв"); err != nil {
		t.Fatalf("Insert err = %v, ожидалось nil", err)
	}

	// Невалидный UTF-8 — различимая ошибка, не промах Search и не «найденный
	// префикс» StartsWith: Trie и Len не меняются.
	inserted, err := trie.Insert(invalidUTF8)
	if inserted || !errors.Is(err, trees.ErrInvalidUTF8) {
		t.Fatalf("Insert с невалидным UTF-8 = (%v, %v), ожидалось (false, ErrInvalidUTF8)", inserted, err)
	}
	found, err := trie.Search(invalidUTF8)
	if found || !errors.Is(err, trees.ErrInvalidUTF8) {
		t.Fatalf("Search с невалидным UTF-8 = (%v, %v), ожидалось (false, ErrInvalidUTF8)", found, err)
	}
	found, err = trie.StartsWith(invalidUTF8)
	if found || !errors.Is(err, trees.ErrInvalidUTF8) {
		t.Fatalf("StartsWith с невалидным UTF-8 = (%v, %v), ожидалось (false, ErrInvalidUTF8)", found, err)
	}
	deleted, err := trie.Delete(invalidUTF8)
	if deleted || !errors.Is(err, trees.ErrInvalidUTF8) {
		t.Fatalf("Delete с невалидным UTF-8 = (%v, %v), ожидалось (false, ErrInvalidUTF8)", deleted, err)
	}

	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("абв"); !found || err != nil {
		t.Fatalf("Search после ошибок = (%v, %v), ожидалось (true, nil)", found, err)
	}
}

func TestTrieNonBMPIsOneRune(t *testing.T) {
	trie := trees.NewTrie()

	// Символ вне BMP (😀, U+1F600) — одна руна: слово из одной руны хранится
	// и находится, а его «половины» UTF-16 — другие слова.
	inserted, err := trie.Insert("😀")
	if !inserted || err != nil {
		t.Fatalf(`Insert("😀") = (%v, %v), ожидалось (true, nil)`, inserted, err)
	}
	if got := trie.Len(); got != 1 {
		t.Fatalf("Len() = %d, ожидалось 1", got)
	}
	if found, err := trie.Search("😀"); !found || err != nil {
		t.Fatalf(`Search("😀") = (%v, %v), ожидалось (true, nil)`, found, err)
	}
	if found, err := trie.StartsWith("😀"); !found || err != nil {
		t.Fatalf(`StartsWith("😀") = (%v, %v), ожидалось (true, nil)`, found, err)
	}
	deleted, err := trie.Delete("😀")
	if !deleted || err != nil {
		t.Fatalf(`Delete("😀") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	if got := trie.Len(); got != 0 {
		t.Fatalf("Len() = %d, ожидалось 0", got)
	}
}

func TestTrieUninitialized(t *testing.T) {
	// nil-приёмник и нулевое значение Trie{} — не паника: все операции
	// отклоняются ErrUninitializedTrie, Len() — 0.
	uninitialized := []*trees.Trie{nil, {}}
	for i, trie := range uninitialized {
		if got := trie.Len(); got != 0 {
			t.Fatalf("trie %d: Len() = %d, ожидалось 0", i, got)
		}
		if inserted, err := trie.Insert("а"); inserted || !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: Insert = (%v, %v), ожидалось (false, ErrUninitializedTrie)", i, inserted, err)
		}
		if found, err := trie.Search("а"); found || !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: Search = (%v, %v), ожидалось (false, ErrUninitializedTrie)", i, found, err)
		}
		if found, err := trie.StartsWith("а"); found || !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: StartsWith = (%v, %v), ожидалось (false, ErrUninitializedTrie)", i, found, err)
		}
		if deleted, err := trie.Delete("а"); deleted || !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: Delete = (%v, %v), ожидалось (false, ErrUninitializedTrie)", i, deleted, err)
		}
		// Ошибка неинициализированного Trie — раньше проверки входа: не
		// ErrInsertEmptyWord и не ErrInvalidUTF8.
		if _, err := trie.Insert(""); !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: Insert(\"\") err = %v, ожидалось ErrUninitializedTrie", i, err)
		}
		if _, err := trie.Insert(invalidUTF8); !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: Insert с невалидным UTF-8 err = %v, ожидалось ErrUninitializedTrie", i, err)
		}
		if _, err := trie.Delete(""); !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: Delete(\"\") err = %v, ожидалось ErrUninitializedTrie", i, err)
		}
		if _, err := trie.Search(invalidUTF8); !errors.Is(err, trees.ErrUninitializedTrie) {
			t.Fatalf("trie %d: Search с невалидным UTF-8 err = %v, ожидалось ErrUninitializedTrie", i, err)
		}
	}
}

func TestTrieExactComparison(t *testing.T) {
	trie := trees.NewTrie()
	// Регистр и NFC/NFD-форма символа («é»-руна U+00E9 против «e»+U+0301,
	// «ё»-руна U+0451 против «е»+U+0308) — разные последовательности rune,
	// слова различны.
	words := []string{"Apple", "apple", "é", "e\u0301", "ёлка", "е\u0308лка"}

	for _, w := range words {
		if inserted, err := trie.Insert(w); !inserted || err != nil {
			t.Fatalf("Insert(%q) = (%v, %v), ожидалось (true, nil)", w, inserted, err)
		}
	}
	if got := trie.Len(); got != len(words) {
		t.Fatalf("Len() = %d, ожидалось %d", got, len(words))
	}
	for _, w := range words {
		if found, err := trie.Search(w); !found || err != nil {
			t.Fatalf("Search(%q) = (%v, %v), ожидалось (true, nil)", w, found, err)
		}
	}
	// Взаимные промахи: нормализации и смены регистра нет.
	if found, err := trie.Search("APPLE"); found || err != nil {
		t.Fatalf("Search(\"APPLE\") = (%v, %v), ожидалось (false, nil)", found, err)
	}
	if found, err := trie.Search("ё\u0308лка"); found || err != nil {
		t.Fatalf("Search(\"ё\\u0308лка\") = (%v, %v), ожидалось (false, nil)", found, err)
	}
	if found, err := trie.StartsWith("APP"); found || err != nil {
		t.Fatalf("StartsWith(\"APP\") = (%v, %v), ожидалось (false, nil)", found, err)
	}

	// Точность распространяется и на удаление: удаляется ровно своё слово.
	deleted, err := trie.Delete("apple")
	if !deleted || err != nil {
		t.Fatalf(`Delete("apple") = (%v, %v), ожидалось (true, nil)`, deleted, err)
	}
	if found, err := trie.Search("Apple"); !found || err != nil {
		t.Fatalf("Search(\"Apple\") = (%v, %v), ожидалось (true, nil)", found, err)
	}
	if got := trie.Len(); got != len(words)-1 {
		t.Fatalf("Len() = %d, ожидалось %d", got, len(words)-1)
	}
}
