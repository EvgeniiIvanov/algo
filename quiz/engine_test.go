package quiz

import (
	"bytes"
	"testing"
)

// Банк из двух модулей по одному вопросу — для сценариев сессии.
func testBank(t *testing.T) *Bank {
	t.Helper()
	return &Bank{Blocks: []Block{{
		ID:    "block-1",
		Title: "Блок 1. Фундамент",
		Modules: []Module{
			{
				ID:    "big-o",
				Title: "Big O",
				Questions: []Question{{
					Text:        "Сложность доступа по индексу?",
					Options:     []string{"O(1)", "O(log n)", "O(n)", "O(n²)"},
					Correct:     0,
					Explanation: "Индекс считается за константу.",
				}},
			},
			{
				ID:    "binary-search",
				Title: "Бинарный поиск",
				Questions: []Question{{
					Text:        "Что делает бинарный поиск на каждом шаге?",
					Options:     []string{"Отбрасывает половину", "Меняет соседей", "Делит на n частей", "Идёт с начала"},
					Correct:     0,
					Explanation: "Сравнение с серединой отсекает половину области поиска.",
				}},
			},
		},
	}}}
}

// allRight — сценарий: выбор блока 1, все ответы правильные, выход.
func allRight() string {
	return "1\n1\n1\nq\n"
}

// oneWrongOneRight — сценарий: первый ответ неверный, второй верный, выход.
func oneWrongOneRight() string {
	return "1\n2\n1\nq\n"
}

// runEngine прогоняет движок со скриптованным вводом и возвращает вывод.
func runEngine(t *testing.T, bank *Bank, input string) string {
	t.Helper()
	var out bytes.Buffer
	err := Run(bytes.NewBufferString(input), &out, bank)
	if err != nil {
		t.Fatalf("Run: %v\nвывод:\n%s", err, out.String())
	}
	return out.String()
}

func TestEngineAllCorrect(t *testing.T) {
	out := runEngine(t, testBank(t), allRight())

	want := "Big O: 1/1\n  Бинарный поиск: 1/1"
	if !bytes.Contains([]byte(out), []byte(want)) {
		t.Errorf("нет сводки %q в выводе:\n%s", want, out)
	}
}

func TestEngineCounting(t *testing.T) {
	out := runEngine(t, testBank(t), oneWrongOneRight())

	want := "Блок 1. Фундамент: 1/2"
	if !bytes.Contains([]byte(out), []byte(want)) {
		t.Errorf("нет сводки %q в выводе:\n%s", want, out)
	}
}

func TestEngineEmptySession(t *testing.T) {
	out := runEngine(t, testBank(t), "q\n") // выход сразу, без выбора блока

	if !bytes.Contains([]byte(out), []byte("Ни один блок не пройден")) {
		t.Errorf("нет сообщения о пустой сессии:\n%s", out)
	}
}

func TestEngineQuitMidBlock(t *testing.T) {
	// Выход после первого вопроса: посчитан только отвеченный.
	out := runEngine(t, testBank(t), "1\n1\nq\n")

	want := "Блок 1. Фундамент: 1/1"
	if !bytes.Contains([]byte(out), []byte(want)) {
		t.Errorf("нет сводки %q в выводе:\n%s", want, out)
	}
}

func TestEngineInvalidInputDoesNotCount(t *testing.T) {
	// «99» на выбор блока, «0» на ответ — мусор не должен попасть в счёт.
	out := runEngine(t, testBank(t), "99\n1\n0\n1\n1\nq\n")

	want := "Блок 1. Фундамент: 2/2"
	if !bytes.Contains([]byte(out), []byte(want)) {
		t.Errorf("нет сводки %q в выводе:\n%s", want, out)
	}
}

func TestEngineWrongAnswerShowsExplanation(t *testing.T) {
	// Сценарий: ответ «2» на вопрос с правильным ответом «1».
	out := runEngine(t, testBank(t), "1\n2\nq\n")

	want := "Пояснение: Индекс считается за константу."
	if !bytes.Contains([]byte(out), []byte(want)) {
		t.Errorf("нет пояснения %q после неверного ответа:\n%s", want, out)
	}
}

func TestEngineReplayResetsCounters(t *testing.T) {
	// Блок проходится дважды: сначала оба ответа верные (2/2),
	// потом оба неверные. Сводка должна показывать только второй
	// проход, а не сумму двух: 0/2, а не 2/4.
	out := runEngine(t, testBank(t), "1\n1\n1\n1\n2\n2\nq\n")

	want := "Блок 1. Фундамент: 0/2"
	if !bytes.Contains([]byte(out), []byte(want)) {
		t.Errorf("нет сводки %q в выводе:\n%s", want, out)
	}
}

func TestEngineQuitWhileAsking(t *testing.T) {
	// Выход на вопросе бинарного поиска (q вместо ответа): учтён неотвеченный.
	out := runEngine(t, testBank(t), "1\n1\nq\nq\n")

	want := "Блок 1. Фундамент: 1/1\n  Big O: 1/1\n  Бинарный поиск: 0/0"
	if !bytes.Contains([]byte(out), []byte(want)) {
		t.Errorf("нет сводки %q в выводе:\n%s", want, out)
	}
}

func TestEngineInvalidBlock(t *testing.T) {
	// Движок обязан проверять банку перед запуском.
	bank := testBank(t)
	bank.Blocks[0].Modules[0].Questions[0].Correct = 9

	var out bytes.Buffer
	err := Run(bytes.NewBufferString("1\n"), &out, bank)
	if err == nil {
		t.Fatal("ожидали ошибку на некорректный банке, получили nil")
	}
}
