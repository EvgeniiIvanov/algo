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

// Тест порядка модулей block-1 (тикет 16): hash-tables стоит после
// structures и до модулей сортировок, если они уже добавлены.
func TestEmbeddedBankBlock1HashTablesOrder(t *testing.T) {
	bank, err := EmbeddedBank()
	if err != nil {
		t.Fatalf("EmbeddedBank: %v", err)
	}
	var block1 *Block
	for i := range bank.Blocks {
		if bank.Blocks[i].ID == "block-1" {
			block1 = &bank.Blocks[i]
			break
		}
	}
	if block1 == nil {
		t.Fatal("в банке нет блока block-1")
	}
	structuresIdx, hashIdx := -1, -1
	for i, m := range block1.Modules {
		switch m.ID {
		case "structures":
			structuresIdx = i
		case "hash-tables":
			hashIdx = i
		}
	}
	if structuresIdx == -1 {
		t.Fatal("в block-1 нет модуля structures")
	}
	if hashIdx == -1 {
		t.Fatal("в block-1 нет модуля hash-tables")
	}
	if hashIdx < structuresIdx {
		t.Errorf("модуль hash-tables (индекс %d) стоит раньше structures (индекс %d), ожидалось после", hashIdx, structuresIdx)
	}
	// ID модулей сортировок ещё не зафиксированы; когда они появятся,
	// hash-tables должна остаться перед ними — опознаём по подстроке
	// "sort" в id (sorting, merge-sort, quick-sort и т. п.).
	for i, m := range block1.Modules {
		if i < hashIdx && strings.Contains(m.ID, "sort") {
			t.Errorf("модуль сортировок %q (индекс %d) стоит раньше hash-tables (индекс %d)", m.ID, i, hashIdx)
		}
	}
}
