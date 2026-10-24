package quiz

import (
	"strings"
	"testing"
)

func TestEmbeddedBankValid(t *testing.T) {
	bank, err := EmbeddedBank()
	if err != nil {
		t.Fatalf("EmbeddedBank: %v", err)
	}
	if len(bank.Blocks) == 0 {
		t.Fatal("во встроенном банке нет блоков")
	}
	if got := bank.Blocks[0].Title; !strings.Contains(got, "Фундамент") {
		t.Errorf("первый блок %q, ожидали «Блок 1. Фундамент»", got)
	}
}

func TestEmbeddedBankModuleQuestionCounts(t *testing.T) {
	bank, err := EmbeddedBank()
	if err != nil {
		t.Fatalf("EmbeddedBank: %v", err)
	}
	// На модуль — от 5 до 8 вопросов (тикет 04); жёсткие числа не
	// фиксируем, чтобы набор вопросов мог расти без правки теста.
	for _, b := range bank.Blocks {
		for _, m := range b.Modules {
			if n := len(m.Questions); n < minQuestionsPerModule || n > maxQuestionsPerModule {
				t.Errorf("блок %q, модуль %q: вопросов %d, ожидали %d–%d",
					b.ID, m.ID, n, minQuestionsPerModule, maxQuestionsPerModule)
			}
		}
	}
}

// Границы размера набора на модуль из тикета 04.
const (
	minQuestionsPerModule = 5
	maxQuestionsPerModule = 8
)

func TestValidateRejectsTwoCorrect(t *testing.T) {
	// Формат «ровно один правильный» выражается индексом Correct, поэтому
	// второй правильный смоделирован выходом за диапазон.
	Q1 := Question{Text: "q?", Options: []string{"a", "b", "c", "d"}, Correct: -1, Explanation: "потому что"}
	Q2 := Question{Text: "q?", Options: []string{"a", "b", "c", "d"}, Correct: 4, Explanation: "потому что"}
	for _, q := range []Question{Q1, Q2} {
		if err := q.validate(); err == nil {
			t.Errorf("ожидали ошибку для Correct=%d", q.Correct)
		}
	}
}

func TestValidateRejectsBadFormat(t *testing.T) {
	cases := map[string]Question{
		"пустой текст":   {Options: []string{"a", "b", "c", "d"}, Correct: 0, Explanation: "потому что"},
		"3 варианта":     {Text: "q?", Options: []string{"a", "b", "c"}, Correct: 0, Explanation: "потому что"},
		"5 вариантов":    {Text: "q?", Options: []string{"a", "b", "c", "d", "e"}, Correct: 0, Explanation: "потому что"},
		"нет пояснения":  {Text: "q?", Options: []string{"a", "b", "c", "d"}, Correct: 0},
		"пустой вариант": {Text: "q?", Options: []string{"a", "b", "  ", "d"}, Correct: 0, Explanation: "потому что"},
	}
	for name, q := range cases {
		if err := q.validate(); err == nil {
			t.Errorf("%s: ожидали ошибку", name)
		}
	}
}

func TestValidateAcceptsGoodQuestion(t *testing.T) {
	q := Question{Text: "q?", Options: []string{"a", "b", "c", "d"}, Correct: 1, Explanation: "потому что"}
	if err := q.validate(); err != nil {
		t.Errorf("хороший вопрос не прошёл валидацию: %v", err)
	}
}

func TestValidateRejectsModuleWithoutQuestions(t *testing.T) {
	bank := &Bank{Blocks: []Block{{ID: "b1", Title: "Блок", Modules: []Module{{ID: "m1", Title: "Модуль"}}}}}
	if err := bank.Validate(); err == nil {
		t.Error("ожидали ошибку для модуля без вопросов")
	}
}

func TestValidateRejectsModuleWithoutID(t *testing.T) {
	bank := &Bank{Blocks: []Block{{
		ID:    "b1",
		Title: "Блок",
		Modules: []Module{{
			Title: "Модуль без привязки",
			Questions: []Question{{
				Text:        "q?",
				Options:     []string{"a", "b", "c", "d"},
				Correct:     0,
				Explanation: "потому что",
			}},
		}},
	}}}
	if err := bank.Validate(); err == nil {
		t.Error("ожидали ошибку для вопроса без привязки к модулю")
	}
}

func TestValidateRejectsDuplicateBlock(t *testing.T) {
	b := Block{ID: "b1", Title: "Блок", Modules: []Module{{
		ID:    "m1",
		Title: "Модуль",
		Questions: []Question{{
			Text:        "q?",
			Options:     []string{"a", "b", "c", "d"},
			Correct:     0,
			Explanation: "потому что",
		}},
	}}}
	bank := &Bank{Blocks: []Block{b, b}}
	if err := bank.Validate(); err == nil {
		t.Error("ожидали ошибку для дубликата блока")
	}
}

func TestParseBankMalformedJSON(t *testing.T) {
	_, err := ParseBank([]byte("{не json"))
	if err == nil {
		t.Fatal("ожидали ошибку разбора")
	}
}

// findBlock1 возвращает указатель на блок block-1 встроенного банка
// или завершает тест, если такого блока нет.
func findBlock1(t *testing.T, bank *Bank) *Block {
	t.Helper()
	for i := range bank.Blocks {
		if bank.Blocks[i].ID == "block-1" {
			return &bank.Blocks[i]
		}
	}
	t.Fatal("в банке нет блока block-1")
	return nil
}

// moduleIndexByID возвращает индекс модуля с данным id или -1, если
// модуля нет. Дубликаты id исключены валидацией банка.
func moduleIndexByID(modules []Module, id string) int {
	for i, m := range modules {
		if m.ID == id {
			return i
		}
	}
	return -1
}

// embeddedBlock1WithModules загружает встроенный банк, находит в нём
// block-1 и убеждается, что все перечисленные модули присутствуют.
// Возвращает блок и индексы модулей в порядке их перечисления.
func embeddedBlock1WithModules(t *testing.T, ids ...string) (*Block, []int) {
	t.Helper()
	bank, err := EmbeddedBank()
	if err != nil {
		t.Fatalf("EmbeddedBank: %v", err)
	}
	block1 := findBlock1(t, bank)
	indices := make([]int, len(ids))
	for i, id := range ids {
		indices[i] = moduleIndexByID(block1.Modules, id)
		if indices[i] == -1 {
			t.Fatalf("в block-1 нет модуля %s", id)
		}
	}
	return block1, indices
}

// assertModulesInOrder проверяет, что модули с перечисленными id идут
// в списке модулей блока в том же порядке, что и в списке id.
func assertModulesInOrder(t *testing.T, modules []Module, ids ...string) {
	t.Helper()
	prevIdx, prevID := -1, ""
	for _, id := range ids {
		idx := moduleIndexByID(modules, id)
		if idx == -1 {
			t.Fatalf("в блоке нет модуля %s", id)
		}
		if prevIdx != -1 && idx < prevIdx {
			t.Errorf("модуль %s (индекс %d) стоит раньше модуля %s (индекс %d), ожидалось после", id, idx, prevID, prevIdx)
		}
		prevIdx, prevID = idx, id
	}
}

// Тест порядка модулей block-1 (тикет 16): hash-tables стоит после
// structures и перед модулями сортировок. ID модулей сортировок
// (simple-sorts, divide-and-conquer-sorts) зафиксированы в банке.
func TestEmbeddedBankBlock1HashTablesOrder(t *testing.T) {
	block1, _ := embeddedBlock1WithModules(t,
		"structures", "hash-tables", "simple-sorts", "divide-and-conquer-sorts")
	assertModulesInOrder(t, block1.Modules,
		"structures", "hash-tables", "simple-sorts", "divide-and-conquer-sorts")
}

// Тест порядка модулей block-1 (тикет 22): модуль simple-sorts стоит
// в блоке сразу после hash-tables.
func TestEmbeddedBankBlock1SimpleSortsAfterHashTables(t *testing.T) {
	_, indices := embeddedBlock1WithModules(t, "hash-tables", "simple-sorts")
	hashIdx, simpleSortsIdx := indices[0], indices[1]
	if simpleSortsIdx != hashIdx+1 {
		t.Errorf("модуль simple-sorts (индекс %d) должен стоять сразу после hash-tables (индекс %d)", simpleSortsIdx, hashIdx)
	}
}

// Тест порядка модулей block-1 (тикет 28): модуль divide-and-conquer-sorts
// стоит в блоке после simple-sorts (немедленного соседства не требуется —
// между ними могут появиться новые модули).
func TestEmbeddedBankBlock1DivideAndConquerAfterSimpleSorts(t *testing.T) {
	block1, _ := embeddedBlock1WithModules(t, "simple-sorts", "divide-and-conquer-sorts")
	assertModulesInOrder(t, block1.Modules, "simple-sorts", "divide-and-conquer-sorts")
}
