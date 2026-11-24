package quiz

import (
	"strings"
	"testing"
	"unicode"
)

func TestEmbeddedBankValid(t *testing.T) {
	bank := embeddedBank(t)
	if len(bank.Blocks) == 0 {
		t.Fatal("во встроенном банке нет блоков")
	}
	if got := bank.Blocks[0].Title; !strings.Contains(got, "Фундамент") {
		t.Errorf("первый блок %q, ожидали «Блок 1. Фундамент»", got)
	}
}

func TestEmbeddedBankModuleQuestionCounts(t *testing.T) {
	bank := embeddedBank(t)
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

// findBlock возвращает индекс блока с данным id и указатель на него
// или завершает тест, если такого блока нет.
func findBlock(t *testing.T, bank *Bank, id string) (int, *Block) {
	t.Helper()
	for i := range bank.Blocks {
		if bank.Blocks[i].ID == id {
			return i, &bank.Blocks[i]
		}
	}
	t.Fatalf("в банке нет блока %s", id)
	return 0, nil
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

// embeddedBlock1WithModules находит block-1 во встроенном банке и
// убеждается, что все перечисленные модули присутствуют.
// Возвращает блок и индексы модулей в порядке их перечисления.
func embeddedBlock1WithModules(t *testing.T, ids ...string) (*Block, []int) {
	t.Helper()
	block1 := embeddedBlock(t, "block-1")
	indices := make([]int, len(ids))
	for i, id := range ids {
		indices[i] = moduleIndexByID(block1.Modules, id)
		if indices[i] == -1 {
			t.Fatalf("в block-1 нет модуля %s", id)
		}
	}
	return block1, indices
}

// embeddedBank загружает встроенный банк или завершает тест, если
// загрузить банк не удалось. Единая точка загрузки для всех тестов,
// которым нужен банк целиком; отдельный блок удобнее брать через
// embeddedBlock.
func embeddedBank(t *testing.T) *Bank {
	t.Helper()
	bank, err := EmbeddedBank()
	if err != nil {
		t.Fatalf("EmbeddedBank: %v", err)
	}
	return bank
}

// embeddedBlock загружает встроенный банк и возвращает указатель на
// блок с данным id или завершает тест, если загрузить банк не удалось
// или блока с таким id в нём нет.
func embeddedBlock(t *testing.T, id string) *Block {
	t.Helper()
	bank := embeddedBank(t)
	_, b := findBlock(t, bank, id)
	return b
}

// embeddedModule загружает встроенный банк и возвращает указатель на
// модуль с данным id или завершает тест, если банка, блока или модуля
// с такими id в нём нет.
func embeddedModule(t *testing.T, blockID, moduleID string) *Module {
	t.Helper()
	block := embeddedBlock(t, blockID)
	idx := moduleIndexByID(block.Modules, moduleID)
	if idx == -1 {
		t.Fatalf("в блоке %s нет модуля %s", blockID, moduleID)
	}
	return &block.Modules[idx]
}

// questionByMarker возвращает единственный вопрос модуля, в тексте
// которого встречается маркер, или завершает тест, если маркер не
// совпал ни с одним вопросом либо совпал больше чем с одним:
// неоднозначный маркер нельзя молча сводить к первому совпадению —
// тест мог бы незаметно проверять другой вопрос.
func questionByMarker(t *testing.T, moduleID string, questions []Question, marker string) *Question {
	t.Helper()
	var found []int
	for i := range questions {
		if strings.Contains(questions[i].Text, marker) {
			found = append(found, i)
		}
	}
	switch len(found) {
	case 1:
		return &questions[found[0]]
	case 0:
		t.Fatalf("в модуле %s нет вопроса с маркером %q", moduleID, marker)
	default:
		t.Fatalf("маркер %q совпал с несколькими вопросами модуля %s (индексы %v); уточните маркер",
			marker, moduleID, found)
	}
	return nil
}

// containsAny сообщает, встречается ли в s хотя бы один из фрагментов.
func containsAny(s string, fragments ...string) bool {
	for _, f := range fragments {
		if strings.Contains(s, f) {
			return true
		}
	}
	return false
}

// sentences разбивает текст на предложения: границей служит точка, за
// которой идёт пробел или конец текста. Сокращений с точкой внутри
// предложений в пояснениях банка нет.
func sentences(text string) []string {
	var out []string
	start := 0
	for i := 0; i < len(text); i++ {
		if text[i] != '.' {
			continue
		}
		if i == len(text)-1 || text[i+1] == ' ' || text[i+1] == '\n' {
			out = append(out, strings.TrimSpace(text[start:i+1]))
			start = i + 1
		}
	}
	if start < len(text) {
		out = append(out, strings.TrimSpace(text[start:]))
	}
	return out
}

// segmentAfterCall возвращает фрагмент текста от вызова call до
// ближайшего из других вызовов others или до конца текста: в этом
// фрагменте записан результат вызова («Search("") = false»). Если call
// не встречается, возвращает пустую строку.
func segmentAfterCall(text, call string, others ...string) string {
	start := strings.Index(text, call)
	if start == -1 {
		return ""
	}
	start += len(call)
	end := len(text)
	for _, other := range others {
		if i := strings.Index(text[start:], other); i != -1 && start+i < end {
			end = start + i
		}
	}
	return text[start:end]
}

// segmentEnd возвращает длину первого фрагмента s до разделителя
// (запятая, точка с запятой, точка) или длину s, если разделителя нет.
func segmentEnd(s string) int {
	if i := strings.IndexAny(s, ",;."); i != -1 {
		return i
	}
	return len(s)
}

// splitWords разбивает текст на строчные слова из букв: знаки и цифры
// служат разделителями.
func splitWords(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
	for i, w := range fields {
		fields[i] = strings.ToLower(w)
	}
	return fields
}

// hasStem сообщает, начинается ли слово с одной из основ.
func hasStem(word string, stems []string) bool {
	for _, stem := range stems {
		if strings.HasPrefix(word, stem) {
			return true
		}
	}
	return false
}

// universalCallWording сообщает, утверждает ли текст, что ошибка
// возникает на любой/все/каждый вызов, метод, обращение или операцию —
// «на любой вызов», «все методы», «каждый вызов», «произвольную
// операцию». Квантор и существительное согласуются падежом, поэтому
// они ищутся по основам, а не точными фразами. Связка квантора с
// существительным ищется на расстоянии до пяти слов, если промежуточные
// слова служебные, — так распознаются «любой из этих вызовов» и «все
// без исключения вызовы». Ограниченное утверждение ошибкой на каждый
// вызов не считается: знаменательное слово между квантором и
// существительным разрывает связку («все четыре метода — Insert,
// Search, StartsWith и Delete — …»), а оговорка «(вызов/метод)
// с ошибкой …» (как в SPEC) и явное исключение «любой метод, кроме
// Len()» сужают утверждение до методов с результатом-ошибкой.
func universalCallWording(text string) bool {
	quantifierStems := []string{"люб", "всех", "всем", "все", "кажд", "всяк", "произвольн"}
	nounStems := []string{"вызов", "метод", "обращени", "операци"}
	words := splitWords(text)
	for i, w := range words {
		if !hasStem(w, quantifierStems) {
			continue
		}
		for j := i - 5; j <= i+5; j++ {
			if j < 0 || j >= len(words) || j == i || !hasStem(words[j], nounStems) {
				continue
			}
			if !linkedByFunctionWords(words, i, j) {
				continue
			}
			if qualifiedByErrorWords(words, j) || excludedByExceptClause(words, j) {
				continue
			}
			return true
		}
	}
	return false
}

// qualifiedByErrorWords сообщает, оговорено ли существительное
// words[nounIdx] сужением «с ошибкой …» («каждый метод с ошибкой в
// результате») или причастием «возвращающий ошибку …» среди ближайших
// слов после него.
func qualifiedByErrorWords(words []string, nounIdx int) bool {
	for k := nounIdx + 1; k <= nounIdx+4 && k < len(words); k++ {
		if !strings.HasPrefix(words[k], "ошибк") {
			continue
		}
		switch prev := words[k-1]; {
		case prev == "с" || prev == "со":
			return true
		case strings.HasPrefix(prev, "возвращающ"), strings.HasPrefix(prev, "дающ"):
			return true
		case strings.HasPrefix(prev, "результатом"): // «с результатом-ошибкой»
			return true
		}
	}
	return false
}

// functionWordsExact и functionWordStems — служебные слова, которые
// могут стоять между квантором и существительным, не разрывая их
// связку: «любой из этих вызовов», «все без исключения вызовы».
var functionWordsExact = map[string]bool{
	"из": true, "без": true, "с": true, "со": true,
	"на": true, "для": true, "по": true, "при": true,
	"о": true, "об": true, "от": true, "до": true,
}

var functionWordStems = []string{"эт", "исключени"}

// isFunctionWord сообщает, служебное ли слово; знаменательное слово
// (числительное «четыре», прилагательное) связку разрывает.
func isFunctionWord(w string) bool {
	return functionWordsExact[w] || hasStem(w, functionWordStems)
}

// linkedByFunctionWords сообщает, связывают ли квантор words[qi] и
// существительное words[ni] только служебные слова: знаменательное
// слово между ними разрывает связку, поэтому универсальное
// утверждение об ограниченном множестве («все четыре метода»)
// нарушением не считается.
func linkedByFunctionWords(words []string, qi, ni int) bool {
	lo, hi := qi, ni
	if lo > hi {
		lo, hi = hi, lo
	}
	for k := lo + 1; k < hi; k++ {
		if !isFunctionWord(words[k]) {
			return false
		}
	}
	return true
}

// excludedByExceptClause сообщает, исключено ли существительное
// words[nounIdx] явной оговоркой «кроме …», «за исключением …» или
// «но не …» среди ближайших слов после него: «любой метод, кроме
// Len()» утверждает ошибку не на каждый вызов.
func excludedByExceptClause(words []string, nounIdx int) bool {
	for k := nounIdx + 1; k <= nounIdx+3 && k < len(words); k++ {
		switch {
		case words[k] == "кроме":
			return true
		case words[k] == "за" && k+1 < len(words) && hasStem(words[k+1], []string{"исключен"}):
			return true
		case words[k] == "но" && k+1 < len(words) && words[k+1] == "не":
			return true
		}
	}
	return false
}

// lenWordingSafe сообщает, корректно ли клауза про
// ErrUninitializedTrie обходит Len(): ни одно упоминание Len() не
// вправе включать метод в утверждающие ошибку («…Delete и Len()
// возвращают ErrUninitializedTrie», «Len() возвращает ошибку»), и
// хотя бы одно упоминание обязано оговаривать Len() — нулём («Len()
// при этом возвращает 0», «возвращает ноль», «= 0»), отрицанием
// ошибки («Len() ошибки не возвращает») или оговоркой-исключением
// («любой метод, кроме Len()»). Упоминаний Len() в клаузе быть должно,
// это проверяет вызывающий тест.
func lenWordingSafe(clause string) bool {
	handled := false
	rest := clause
	for {
		i := strings.Index(rest, "Len()")
		if i == -1 {
			break
		}
		before, after := rest[:i], rest[i+len("Len()"):]
		if lenClaimedErroneous(after) {
			return false
		}
		if lenMentionHandled(before, after) {
			handled = true
		}
		rest = after
	}
	return handled
}

// lenClaimedErroneous сообщает, утверждает ли фрагмент после
// упоминания Len(), что метод возвращает ErrUninitializedTrie или
// ошибку: ErrUninitializedTrie рядом с Len() или слово об ошибке без
// отрицания «не» включает Len() в возвращающие ошибку методы, чего на
// неинициализированном Trie не бывает (SPEC, «Минимум API»).
func lenClaimedErroneous(after string) bool {
	words := splitWords(after)
	sawNot, sawErr := false, false
	for k := 0; k <= 3 && k < len(words); k++ {
		switch {
		case words[k] == "erruninitializedtrie":
			return true
		case words[k] == "не":
			sawNot = true
		case hasStem(words[k], []string{"ошибк"}):
			sawErr = true
		}
	}
	return sawErr && !sawNot
}

// lenMentionHandled сообщает, оговорено ли данное упоминание Len():
// нулём рядом («Len() при этом возвращает 0», «возвращает ноль»,
// «= 0»), отрицанием ошибки («Len() ошибки не возвращает») или
// оговоркой-исключением («любой метод, кроме Len()», «за исключением
// Len()», «но не Len()»).
func lenMentionHandled(before, after string) bool {
	beforeWords := splitWords(before)
	if len(beforeWords) > 3 {
		beforeWords = beforeWords[len(beforeWords)-3:]
	}
	for _, w := range beforeWords {
		if w == "кроме" || hasStem(w, []string{"исключен"}) {
			return true
		}
	}
	if len(beforeWords) >= 2 &&
		beforeWords[len(beforeWords)-2] == "но" && beforeWords[len(beforeWords)-1] == "не" {
		return true
	}
	words := splitWords(after)
	sawNot, sawErr := false, false
	for k := 0; k <= 4 && k < len(words); k++ {
		switch {
		case words[k] == "не":
			sawNot = true
		case hasStem(words[k], []string{"ошибк"}):
			sawErr = true
		case hasStem(words[k], []string{"нул", "нол"}):
			return true
		}
	}
	if sawNot && sawErr {
		return true
	}
	if i := segmentEnd(after); strings.Contains(after[:i], "0") {
		return true
	}
	return false
}

// Таблицы проверяют сами детекторы формулировок: регрессия на
// неточную формулировку не должна проходить мимо них молча.
func TestUniversalCallWording(t *testing.T) {
	accurate := []string{
		// Ограниченное множество: числительное разрывает связку
		// квантора с существительным.
		"ErrUninitializedTrie возвращают все четыре метода — Insert, Search, StartsWith и Delete — до проверки входа",
		// Явное исключение Len.
		"ошибку возвращает любой метод, кроме Len()",
		// Сужение до методов с результатом-ошибкой (SPEC).
		"каждый метод с ошибкой в результате — Insert, Search, StartsWith и Delete",
		"ErrUninitializedTrie возвращает любой вызов, возвращающий ошибку",
		"любой из четырёх вызовов завершается ErrUninitializedTrie",
	}
	erroneous := []string{
		// Прежняя неточная формулировка и её варианты.
		"ErrUninitializedTrie возникает на любой вызов",
		"ошибку возвращают все методы",
		"каждая операция заканчивается ErrUninitializedTrie",
		// Универсальное утверждение на расстоянии: служебные слова
		// связку не разрывают.
		"любой из этих вызовов завершается ErrUninitializedTrie",
		"все без исключения вызовы возвращают ErrUninitializedTrie",
	}
	for _, text := range accurate {
		if universalCallWording(text) {
			t.Errorf("точная формулировка распознана как универсальное утверждение: %q", text)
		}
	}
	for _, text := range erroneous {
		if !universalCallWording(text) {
			t.Errorf("универсальное утверждение не распознано: %q", text)
		}
	}
}

func TestLenWordingSafe(t *testing.T) {
	accurate := []string{
		// Формулировка из банка: Len() оговорено нулём.
		"неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа; Len() при этом возвращает 0",
		// Отрицание ошибки и ноль в продолжении клаузы.
		"Len() ошибки не возвращает, а даёт 0",
		"Len() возвращает ноль",
		// Оговорка-исключение.
		"ошибку возвращает любой метод, кроме Len()",
	}
	erroneous := []string{
		// Len() включён в утверждающие ошибку методы.
		"Insert, Search, StartsWith, Delete и Len() возвращают ErrUninitializedTrie",
		"Len() возвращает ErrUninitializedTrie",
		"Len() возвращает ошибку",
		// Упоминание Len() без нуля, отрицания или исключения не
		// считается оговоркой.
		"Len() при этом просто считает слова",
	}
	for _, clause := range accurate {
		if !lenWordingSafe(clause) {
			t.Errorf("точная формулировка про Len() распознана как ошибочная: %q", clause)
		}
	}
	for _, clause := range erroneous {
		if lenWordingSafe(clause) {
			t.Errorf("небезопасная формулировка про Len() не распознана: %q", clause)
		}
	}
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
// structures и перед модулями сортировок. Проверка распознаёт модули
// сортировок универсально по ID, содержащему "sort", чтобы инвариант
// сохранялся при добавлении новых сортировочных модулей.
func TestEmbeddedBankBlock1HashTablesOrder(t *testing.T) {
	block1, _ := embeddedBlock1WithModules(t,
		"structures", "hash-tables", "simple-sorts", "divide-and-conquer-sorts")

	// Универсальная проверка: любой модуль сортировки не стоит перед
	// hash-tables.
	hashIdx := moduleIndexByID(block1.Modules, "hash-tables")
	for i, m := range block1.Modules[:hashIdx] {
		if strings.Contains(m.ID, "sort") {
			t.Errorf("модуль сортировки %s (индекс %d) стоит раньше hash-tables (индекс %d)", m.ID, i, hashIdx)
		}
	}

	// Известные модули сохраняют свой порядок.
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

// Тест порядка блоков и модулей Блока 2 (тикет 33): блок block-2
// («Деревья») идёт после block-1, и первым модулем в нём стоит
// tree-traversals.
func TestEmbeddedBankBlock2StartsWithTreeTraversals(t *testing.T) {
	bank := embeddedBank(t)

	block1Idx, _ := findBlock(t, bank, "block-1")
	block2Idx, block2 := findBlock(t, bank, "block-2")

	if !strings.Contains(block2.Title, "Деревья") {
		t.Errorf("название блока block-2 %q, ожидали «Деревья»", block2.Title)
	}
	if block2Idx <= block1Idx {
		t.Errorf("блок block-2 (индекс %d) должен идти после block-1 (индекс %d)", block2Idx, block1Idx)
	}

	// Первый модуль Блока 2 — tree-traversals; модули Блока 1 в него
	// попадать не должны. Ниже проверяется один представитель block-1
	// (big-o), остальные модули block-1 этим тестом не проверяются.
	if len(block2.Modules) == 0 {
		t.Fatal("в блоке block-2 нет модулей")
	}
	if got := block2.Modules[0].ID; got != "tree-traversals" {
		t.Errorf("первый модуль block-2 %q, ожидали tree-traversals", got)
	}
	if idx := moduleIndexByID(block2.Modules, "big-o"); idx != -1 {
		t.Errorf("модуль big-o из block-1 найден в block-2 (индекс %d)", idx)
	}
}

// Тест порядка модулей Блока 2 (тикет 37): модуль bst присутствует в
// block-2 и идёт после tree-traversals.
func TestEmbeddedBankBlock2BSTAfterTreeTraversals(t *testing.T) {
	block2 := embeddedBlock(t, "block-2")
	assertModulesInOrder(t, block2.Modules, "tree-traversals", "bst")
}

// Тест порядка модулей Блока 2 (тикет 38): модуль bst-balancing
// присутствует в block-2 и идёт после bst.
func TestEmbeddedBankBlock2BSTBalancingAfterBST(t *testing.T) {
	block2 := embeddedBlock(t, "block-2")
	assertModulesInOrder(t, block2.Modules, "tree-traversals", "bst", "bst-balancing")
}

// Тест порядка модулей Блока 2 (тикет 39): модуль b-trees присутствует
// в block-2 и идёт после bst-balancing; остальная цепочка
// tree-traversals → bst → bst-balancing покрыта тестами тикетов 37 и 38.
func TestEmbeddedBankBlock2BTreesAfterBSTBalancing(t *testing.T) {
	block2 := embeddedBlock(t, "block-2")
	assertModulesInOrder(t, block2.Modules, "bst-balancing", "b-trees")
}

// Тест порядка модулей Блока 2 (тикет 43): модуль trie присутствует в
// block-2 и идёт после b-trees, замыкая цепочку
// tree-traversals → bst → bst-balancing → b-trees → trie.
func TestEmbeddedBankBlock2TrieAfterBTrees(t *testing.T) {
	block2 := embeddedBlock(t, "block-2")
	assertModulesInOrder(t, block2.Modules,
		"tree-traversals", "bst", "bst-balancing", "b-trees", "trie")
}

// Регрессия тикета 39 (ревью): семантически нагруженные формулировки
// вопросов модуля b-trees. Связь «k ключей — k + 1 потомок» обязана
// оговаривать внутренний узел (у листа потомков нет), а вопрос о
// расщеплении заполненного узла обязан быть про корень-лист, у которого
// нет родителя: продвинутый разделитель сам становится родителем
// (новым корнем). Проверки идут по коротким маркерам у публичного шва
// банка, а не повторяют формулировки целиком; маркеры уникальны в
// модуле, чтобы похожий новый вопрос не нарушал проверку.
func TestEmbeddedBankBTreesQuestionSemantics(t *testing.T) {
	questions := embeddedModule(t, "block-2", "b-trees").Questions

	// Вопрос о числе потомков оговаривает внутренний узел, а не любой
	// узел: универсальное «k ключей — k + 1 потомок» было бы ложным для
	// листа. Маркер «внутренн» называет тему вопроса (внутренний узел)
	// и уникален в текстах вопросов модуля.
	q := questionByMarker(t, "b-trees", questions, "внутренн")
	if !strings.Contains(q.Text, "потомк") {
		t.Errorf("вопрос про k ключей не спрашивает про потомков: %q", q.Text)
	}
	if got := q.Options[q.Correct]; !strings.Contains(got, "6 потомк") {
		t.Errorf("правильный ответ должен давать k + 1 = 6 потомков, получено: %q", got)
	}
	if !strings.Contains(q.Explanation, "внутренн") {
		t.Errorf("пояснение не оговаривает внутренний узел: %q", q.Explanation)
	}

	// Вопрос о расщеплении заполненного узла описывает корень-лист:
	// продвинутый разделитель сам становится родителем (новым корнем),
	// «продвинется в родителя» для корня было бы неверно.
	q = questionByMarker(t, "b-trees", questions, "корень-лист")
	correct := q.Options[q.Correct]
	if !strings.Contains(correct, "родител") || !strings.Contains(correct, "корн") {
		t.Errorf("правильный ответ должен назначать продвинутый ключ родителем (новым корнем), получено: %q", correct)
	}
	if !strings.Contains(q.Explanation, "новым корн") {
		t.Errorf("пояснение не проговаривает появление нового корня: %q", q.Explanation)
	}
}

// Регрессия тикета 43 (ревью): вопрос trie о пустом слове обязан точно
// описывать поведение неинициализированного Trie (SPEC, «Минимум API»):
// ErrUninitializedTrie возникает на вызовы Insert, Search, StartsWith
// и Delete, причём проверка инициализации предшествует проверкам ввода
// — в том числе вместо ошибок пустого слова, — а Len() при этом
// возвращает 0; прежняя формулировка «на любой вызов» была неточной.
func TestEmbeddedBankTrieEmptyWordQuestionSemantics(t *testing.T) {
	questions := embeddedModule(t, "block-2", "trie").Questions

	// Тематический маркер «операции с пустым словом» называет тему
	// вопроса и уникален в текстах вопросов модуля trie;
	// questionByMarker завершит тест, если маркер перестанет совпадать
	// или станет неоднозначным.
	q := questionByMarker(t, "trie", questions, "операции с пустым словом")

	// Контроль правильного ответа: не просто упоминания вызовов, а их
	// значения — Search("") на пустом Trie даёт false, а StartsWith("")
	// — true (SPEC, «Минимум API»). Значение ищется в тексте сразу после
	// вызова, до следующего упоминания вызова, поэтому перечисление имён
	// без значений проверку не проходит.
	correct := q.Options[q.Correct]
	if !containsAny(correct, `Search("")`, `StartsWith("")`) {
		t.Errorf("правильный ответ должен описывать Search(\"\") и StartsWith(\"\"): %q", correct)
	}
	if part := segmentAfterCall(correct, `Search("")`,
		`StartsWith("")`, `Insert("")`, `Delete("")`); !strings.Contains(part, "false") {
		t.Errorf("правильный ответ должен сообщать, что Search(\"\") даёт false: %q", correct)
	}
	if part := segmentAfterCall(correct, `StartsWith("")`,
		`Search("")`, `Insert("")`, `Delete("")`); !strings.Contains(part, "true") {
		t.Errorf("правильный ответ должен сообщать, что StartsWith(\"\") даёт true: %q", correct)
	}

	// Все проверки семантики неинициализированного Trie — по
	// предложению, называющему ErrUninitializedTrie, а не по хвосту
	// пояснения: Insert и Delete встречаются в именах
	// ErrInsertEmptyWord/ErrDeleteEmptyWord, Search и StartsWith — в
	// предложениях про пустые запросы, поэтому проверки по всему тексту
	// принимали бы случайные совпадения за клаузу.
	var clause string
	errSentences := 0
	for _, s := range sentences(q.Explanation) {
		if strings.Contains(s, "ErrUninitializedTrie") {
			errSentences++
			clause = s
		}
	}
	if errSentences != 1 {
		t.Fatalf("в пояснении должно быть ровно одно предложение с ErrUninitializedTrie, найдено %d: %q",
			errSentences, q.Explanation)
	}

	// Клауза обязана называть Trie неинициализированным — nil-приёмник
	// или нулевое значение Trie{} (SPEC, «Минимум API»): у
	// инициализированного NewTrie() ошибки ErrUninitializedTrie нет,
	// поэтому предложение про инициализированный Trie клаузой не
	// считается. Все дальнейшие проверки метода, порядка и Len идут
	// по этой же клаузе.
	if !containsAny(clause, "неинициализирован", "nil", "нулев") {
		t.Errorf("клауза не называет Trie неинициализированным (nil-приёмник или нулевое значение Trie{}): %q", clause)
	}

	// Ошибка названа для каждого из четырёх методов с
	// результатом-ошибкой (SPEC, «Минимум API»). Имена ошибок пустого
	// слова содержат имена методов (ErrInsertEmptyWord → Insert), поэтому
	// перед проверкой перечня вырезаем их: совпадение внутри имени
	// ошибки не считается упоминанием метода.
	methods := strings.ReplaceAll(clause, "ErrInsertEmptyWord", "")
	methods = strings.ReplaceAll(methods, "ErrDeleteEmptyWord", "")
	for _, op := range []string{"Insert", "Search", "StartsWith", "Delete"} {
		if !strings.Contains(methods, op) {
			t.Errorf("клауза не связывает ErrUninitializedTrie с %s: %q", op, clause)
		}
	}

	// Проверка инициализации предшествует проверкам ввода — в том числе
	// вместо ошибок пустого слова (SPEC, «Минимум API»). Допускаем
	// варианты формулировки порядка, но не его отсутствие.
	if !containsAny(clause,
		"до провер", "прежде провер", "прежде чем провер", "раньше провер",
		"перед провер", "опережает провер", "предшествует провер",
		"до валидац", "прежде валидац", "раньше валидац", "перед валидац") {
		t.Errorf("клауза не оговаривает, что проверка инициализации предшествует проверкам ввода: %q", clause)
	}
	if !containsAny(clause,
		"пустого слова", "пустых слов", "пустым слов", "пустое слов", "EmptyWord") {
		t.Errorf("клауза не оговаривает, что ErrUninitializedTrie возвращается вместо ошибок пустого слова: %q", clause)
	}

	// Len() на неинициализированном Trie возвращает 0, а не ошибку
	// (SPEC, «Минимум API»): упоминание Len() не вправе включать метод
	// в утверждающие ошибку методы и хотя бы одно упоминание обязано
	// оговаривать Len() — нулём, отрицанием ошибки или оговоркой-
	// исключением. Распознавание формулировок — в lenWordingSafe и его
	// табличном тесте.
	if !strings.Contains(clause, "Len()") {
		t.Fatalf("клауза про неинициализированный Trie не упоминает Len(): %q", clause)
	}
	if !lenWordingSafe(clause) {
		t.Errorf("клауза неверно описывает Len() на неинициализированном Trie — включая его в утверждающие ошибку методы или не оговаривая ноль: %q", clause)
	}

	// Негативная проверка на прежнюю неточную формулировку: клауза не
	// вправе утверждать, что ошибка возникает на любой/все/каждый вызов
	// или метод. Универсальное утверждение не спасают ни перечень
	// четырёх методов, ни отдельная оговорка «Len() возвращает 0»:
	// «любой вызов» включал бы и Len(), который ошибки не возвращает.
	if universalCallWording(clause) {
		t.Errorf("клауза не вправе утверждать, что ErrUninitializedTrie возникает на любой/все/каждый вызов или метод: %q", clause)
	}
}
