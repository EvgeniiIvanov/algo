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

// tokenWords возвращает слова из токенов splitTokens: для проверок,
// которым нужны позиции слов (отрицание рядом со словом об ошибке),
// а не границы клауз.
func tokenWords(toks []wordTok) []string {
	words := make([]string, len(toks))
	for i, tk := range toks {
		words[i] = tk.word
	}
	return words
}

// wordTok — слово текста вместе с признаками сильного разделителя
// клаузы перед словом: newClause отмечает любую сильную границу,
// colon — что этой границей было двоеточие. Двоеточие двухранольно:
// оно вводит перечисление, управляемое предыдущим утверждением об
// ошибке («…возникает на вызовы: Insert…»), или отделяет завершённое
// перечисление от заметки про Len() («…и Delete: Len() при этом…»);
// различение по месту применения — в lenEnumeratedWithError и
// lenFirstListedAfterColon.
type wordTok struct {
	word      string
	newClause bool
	colon     bool
}

// isClauseBreak сообщает, сильный ли это разделитель клаузы: точка с
// запятой, точка, тире или двоеточие. За пределами клаузы
// перечисление исключений не продолжается; запятая клаузу не
// заканчивает — внутри клаузы она разделяет элементы перечисления.
// Двоеточие двухранольно (вводит перечисление или отделяет его от
// заметки про Len()); различение — в lenEnumeratedWithError и
// lenFirstListedAfterColon.
func isClauseBreak(r rune) bool {
	switch r {
	case ';', '.', '—', '–', ':':
		return true
	}
	return false
}

// splitTokens разбивает текст на слова, как splitWords, и помечает
// каждое слово признаком сильной границы клаузы перед ним. Второй
// результат сообщает, стоит ли сильный разделитель после последнего
// слова: перечисление, дойдя до него, закончилось, не назвав Len()
// («…кроме Delete; Len() при этом…»).
func splitTokens(text string) (toks []wordTok, tailBreak bool) {
	var b strings.Builder
	inWord := false
	pendingBreak := false
	pendingColon := false
	flush := func() {
		if inWord {
			toks = append(toks, wordTok{
				word:      strings.ToLower(b.String()),
				newClause: pendingBreak,
				colon:     pendingColon,
			})
			b.Reset()
			inWord = false
			pendingBreak = false
			pendingColon = false
		}
	}
	for _, r := range text {
		if unicode.IsLetter(r) {
			b.WriteRune(r)
			inWord = true
			continue
		}
		flush()
		if isClauseBreak(r) {
			pendingBreak = true
			pendingColon = r == ':'
		}
	}
	flush()
	return toks, pendingBreak
}

// universalCallWording сообщает, утверждает ли текст, что ошибка
// возникает на любой/все/каждый вызов, метод, обращение или операцию —
// «на любой вызов», «все методы», «каждый вызов», «произвольную
// операцию». Квантор и существительное согласуются падежом, поэтому
// они ищутся по основам, а не точными фразами. Связка квантора с
// существительным ищется на расстоянии до пяти слов, если промежуточные
// слова служебные, — так распознаются «любой из этих вызовов» и «все
// без исключения вызовы», а указательные определители при
// существительном связку не разрывают: «любой такой вызов», «все
// подобные вызовы», «каждый указанный вызов». Ограниченное
// утверждение ошибкой на каждый вызов не считается: знаменательное
// слово между квантором и существительным разрывает связку («все
// четыре метода — Insert, Search, StartsWith и Delete — …»), а
// оговорка «(вызов/метод) с ошибкой …» (как в SPEC), придаточное
// «(вызов/метод), который возвращает/даёт ошибку, …» и явное
// исключение «любой метод, кроме Len()» сужают утверждение до
// методов с результатом-ошибкой.
func universalCallWording(text string) bool {
	quantifierStems := []string{"люб", "всех", "всем", "все", "кажд", "всяк", "произвольн"}
	nounStems := []string{"вызов", "метод", "обращени", "операци"}
	toks, _ := splitTokens(text)
	words := make([]string, len(toks))
	for i, tk := range toks {
		words[i] = tk.word
	}
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
			if qualifiedByErrorWords(words, j) || excludedByExceptClause(toks, j) {
				continue
			}
			return true
		}
	}
	return false
}

// qualifiedByErrorWords сообщает, оговорено ли существительное
// words[nounIdx] сужением «с ошибкой …» («каждый метод с ошибкой в
// результате»), причастием «возвращающий ошибку …» или придаточным
// «который/которые возвращает/даёт ошибку …» среди ближайших слов
// после него.
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
		case k >= nounIdx+2 && hasStem(words[k-2], []string{"котор"}) &&
			hasStem(prev, predicateStems):
			return true // «методы, которые возвращают ошибку»
		}
	}
	return false
}

// functionWordsExact и functionWordStems — служебные слова, которые
// могут стоять между квантором и существительным, не разрывая их
// связку: «любой из этих вызовов», «все без исключения вызовы», а
// также указательные определители «любой такой вызов», «все подобные
// вызовы», «каждый указанный вызов».
var functionWordsExact = map[string]bool{
	"из": true, "без": true, "с": true, "со": true,
	"на": true, "для": true, "по": true, "при": true,
	"о": true, "об": true, "от": true, "до": true,
}

var functionWordStems = []string{"эт", "исключени", "так", "подобн", "указан"}

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

// errMethodStems — основы имён четырёх методов с результатом-ошибкой
// (SPEC, «Минимум API»): Insert, Search, StartsWith и Delete.
var errMethodStems = []string{"insert", "search", "startswith", "delete"}

// excludedByExceptClause сообщает, сужает ли явная оговорка «кроме …»,
// «за исключением …» или «но не …» среди ближайших слов после
// toks[nounIdx] универсальное утверждение до точного: перечисление
// исключений обязано назвать именно Len() и не вправе назвать ни один
// из методов с результатом-ошибкой. «любой метод, кроме Len()»
// утверждает ошибку не на каждый вызов; исключение чужого метода
// («каждый метод, кроме Delete, …») Len() не оговаривает, а
// исключение ошибочного метода («кроме Delete и Len()») делает
// утверждение неверным — ошибка на этот метод как раз возникает.
func excludedByExceptClause(toks []wordTok, nounIdx int) bool {
	for k := nounIdx + 1; k <= nounIdx+3 && k < len(toks); k++ {
		switch {
		case toks[k].word == "кроме":
			return exceptedLen(toks, k+1)
		case toks[k].word == "за" && k+1 < len(toks) && hasStem(toks[k+1].word, []string{"исключен"}):
			return exceptedLen(toks, k+2)
		case toks[k].word == "но" && k+1 < len(toks) && toks[k+1].word == "не":
			return exceptedLen(toks, k+2)
		}
	}
	return false
}

// exceptedLen сообщает, сужает ли перечисление исключений, начинающееся
// со слова toks[idx], универсальное утверждение корректно: Len() в
// перечислении быть обязан, а ни один из четырёх методов с
// результатом-ошибкой назван быть не вправе. Перечисление начинается
// строго после маркера оговорки: сильный разделитель клаузы
// («кроме Delete; Len() при этом…») или чужое слово заканчивают его,
// и Len() за пределами перечисления оговорки не считается; запятая и
// союз «и» внутри перечисления разделяют элементы. Длинные
// перечисления чужих методов оговоркой Len() не считаются — чужое
// слово останавливает обход.
func exceptedLen(toks []wordTok, idx int) bool {
	hasLen, hasErrMethod := false, false
	for k := idx; k < len(toks) && !toks[k].newClause; k++ {
		w := toks[k].word
		if w == "и" {
			continue
		}
		if hasStem(w, []string{"len"}) {
			hasLen = true
			continue
		}
		if hasStem(w, errMethodStems) {
			hasErrMethod = true
			continue
		}
		break // чужое слово — перечисление закончилось
	}
	return hasLen && !hasErrMethod
}

// lenWordingSafe сообщает, корректно ли клауза про
// ErrUninitializedTrie обходит Len(): каждое упоминание Len() не вправе
// включать метод в утверждающие ошибку, а хотя бы одно упоминание
// обязано быть оговорено — нулём («Len() при этом возвращает 0»,
// «возвращает ноль», «= 0»), отрицанием ошибки («Len() ошибки не
// возвращает», «Len() не возвращает ErrUninitializedTrie») или
// оговоркой-исключением, называющей Len() («любой метод, кроме
// Len()»). Дополнительные упоминания без собственной оговорки
// допустимы, если ошибку они не утверждают; подробности — в
// lenNoteSafe.
func lenWordingSafe(clause string) bool {
	return lenNoteSafe(clause, "")
}

// lenNoteSafe сообщает, корректно ли связанное окно из клаузы про
// ErrUninitializedTrie и следующего за ней предложения оговаривает
// Len(): SPEC не требует делить одно предложение, поэтому оговорка про
// Len() может стоять и отдельным предложением («…даже вместо ошибок
// пустого слова. Len() при этом возвращает 0»). Каждое упоминание
// Len() не вправе включать метод в утверждающие ошибку — ни словами
// после упоминания («Len() возвращает ошибку», «Len() возвращает
// ошибку не всегда» — наречие меры отрицание не снимает), ни
// перечислением методов, получающих ошибку
// («…Insert, Search, StartsWith и Len()»). Обязательна хотя бы одна
// действующая оговорка в окне — ноль, отрицание ошибки или
// исключение, называющее Len(); дополнительные упоминания без
// собственной оговорки допустимы, пока ошибку не утверждают: контраст
// «Len() на неинициализированном Trie возвращает 0, а на
// инициализированном — число слов» или самостоятельная заметка о
// сложности Len(). Контекст упоминания не переходит границу
// предложения: оговорка «кроме …» из клаузы не засчитывается
// упоминанию Len() в следующем предложении.
func lenNoteSafe(clause, next string) bool {
	handled := false
	for _, part := range []string{clause, next} {
		rest := part
		for {
			i := strings.Index(rest, "Len()")
			if i == -1 {
				break
			}
			before, after := rest[:i], rest[i+len("Len()"):]
			if lenClaimedErroneous(after) || lenEnumeratedWithError(before) {
				return false
			}
			if lenMentionHandled(before, after) {
				handled = true
			}
			rest = after
		}
	}
	return handled
}

// predicateStems — основы сказуемых, связывающих «не» или «который» с
// названием ошибки: «не возвращает ошибку», «ошибки не даёт»,
// «которые завершаются ошибкой».
var predicateStems = []string{
	"возвращ", "дающ", "даёт", "дают",
	"отвеча", "заверш", "заканчив", "явля",
}

// negatedErrorClaim сообщает, снимает ли отрицание утверждение об
// ошибке, названной словом words[errIdx]: «не возвращает ошибку»,
// «ошибки не возвращает», «возвращает не ошибку», «не возвращает
// ErrUninitializedTrie», «ошибки нет». Отрицание обязано примыкать к
// утверждению через сказуемое; наречие меры между «не» и ошибкой
// («ошибку не всегда», «не всегда возвращает ошибку») утверждение не
// снимает — ошибка возможна, и утверждение остаётся.
func negatedErrorClaim(words []string, errIdx int) bool {
	if errIdx > 0 && words[errIdx-1] == "не" {
		return true // «не ошибку», «не ErrUninitializedTrie»
	}
	if errIdx > 1 && words[errIdx-2] == "не" && hasStem(words[errIdx-1], predicateStems) {
		return true // «не возвращает ошибку»
	}
	if errIdx+1 < len(words) && words[errIdx+1] == "не" &&
		errIdx+2 < len(words) && hasStem(words[errIdx+2], predicateStems) {
		return true // «ошибки не возвращает»
	}
	if errIdx+1 < len(words) && words[errIdx+1] == "нет" {
		return true // «ошибки нет»
	}
	return false
}

// lenClaimedErroneous сообщает, утверждает ли фрагмент после
// упоминания Len(), что метод возвращает ErrUninitializedTrie или
// ошибку: слово об ошибке без примыкающего отрицания «не»/«нет»
// включает Len() в возвращающие ошибку методы, чего на
// неинициализированном Trie не бывает (SPEC, «Минимум API»).
// Сканирование ограничено клаузой: сильный разделитель заканчивает
// окно утверждения, поэтому ошибка из следующей клаузы упоминанию
// не приписывается, а расстояние внутри клаузы не ограничено —
// между Len() и ошибкой могут стоять несколько слов («Len() на
// неинициализированном Trie возвращает ошибку»). Подлежащее
// отслеживается: другое имя метода, соединённое с Len() союзом «и»
// или предлогом «с»/«со», продолжает совместное подлежащее
// («Len() и Delete возвращают ошибку»), а после знаменательных слов
// означает смену подлежащего — ошибка дальше относится уже к нему.
// Противительные «а» и «но» exemption не дают сами по себе:
// продолжение утверждения про Len() («Len() возвращает 0, а также
// ошибку», «…а иногда ErrUninitializedTrie», «…а при пустом слове
// возвращает ошибку») оставляет ошибку при Len(). Исключение —
// порядок «ошибка, затем сказуемое, затем метод» («…возвращает 0,
// а ошибку возвращает Delete», равнозначное «…но ошибку возвращает
// Delete»): за словом об ошибке следует сказуемое и новое
// подлежащее-метод, значит, ошибка назначена ему, а не Len()
// (безобидные вставки «всегда», «метод» связку не меняют).
// Первое слово об ошибке решает: с отрицанием («Len() не возвращает
// ErrUninitializedTrie», «Len() ошибки не возвращает») утверждение
// снято, без него Len() включён в утверждающие ошибку методы.
func lenClaimedErroneous(after string) bool {
	toks, _ := splitTokens(after)
	words := tokenWords(toks)
	subjectChanged := false
	for k := 0; k < len(toks); k++ {
		if toks[k].newClause {
			return false // ошибка из следующей клаузы упоминанию не принадлежит
		}
		if isErrorWord(toks[k].word) {
			if subjectChanged && methodFollowsErrorPredicate(toks, k) {
				return false // ошибка назначена новому подлежащему, не Len()
			}
			return !negatedErrorClaim(words, k)
		}
		if isMethodToken(toks[k].word) && !jointSubject(toks, k) {
			return false // подлежащее сменилось: ошибка не про Len()
		}
		if toks[k].word == "а" || toks[k].word == "но" {
			subjectChanged = true
		}
	}
	return false
}

// methodFollowsErrorPredicate сообщает, стоит ли за словом об ошибке
// toks[errIdx] сказуемое и подлежащее-метод в той же клаузе: оборот
// «а ошибку возвращает Delete» назначает ошибку новому подлежащему,
// а не Len(). Между ошибкой и методом допускаются слова-сказуемые
// (predicateStems) и безобидные вставки (errorSubjectFillers):
// «а ошибку всегда возвращает Delete», «а ошибку возвращает метод
// Delete». Другое знаменательное слово («а также ошибку, которую
// возвращает Delete») означает, что оборот продолжает утверждение
// про Len(), и exemption не засчитывается. Подлежащим не может стать
// сам Len(): «а ошибку возвращает Len» как раз утверждает ошибку для
// Len(), поэтому len-основа exemption не даёт. Проверяется весь
// координированный список подлежащих: за первым подлежащим в той же
// клаузе вправе стоять только союзы «и»/«или», вставки и имена
// методов, и Len в этом списке («а ошибку возвращает Delete и Len»)
// означает, что ошибка утверждена и ему, — exemption не даётся.
// Свежая заметка про Len() после списка подлежащих отделяется
// противительным «а»/«но» или сильной границей: «и Len() при этом…»
// сразу за списком считается продолжением списка подлежащих.
func methodFollowsErrorPredicate(toks []wordTok, errIdx int) bool {
	subject := -1
	for k := errIdx + 1; k < len(toks) && !toks[k].newClause; k++ {
		w := toks[k].word
		if hasStem(w, predicateStems) || errorSubjectFillers[w] {
			continue
		}
		if isMethodToken(w) && !hasStem(w, []string{"len"}) {
			subject = k
		}
		break
	}
	if subject == -1 {
		return false
	}
	for k := subject + 1; k < len(toks) && !toks[k].newClause; k++ {
		w := toks[k].word
		switch {
		case w == "и" || w == "или" || errorSubjectFillers[w]:
			continue
		case isMethodToken(w):
			if hasStem(w, []string{"len"}) {
				return false // Len — координированное подлежащее
			}
			continue
		}
		break // чужое слово — список подлежащих закончился
	}
	return true
}

// errorSubjectFillers — безобидные вставки между словом об ошибке и
// подлежащим-методом в обороте «а ошибку возвращает …»: наречие
// «всегда» («а ошибку всегда возвращает Delete») и существительное
// «метод» («а ошибку возвращает метод Delete») связку сказуемого
// с подлежащим не меняют. Прочие знаменательные слова exemption
// не дают.
var errorSubjectFillers = map[string]bool{
	"всегда": true,
	"метод":  true,
}

// jointConnectors — служебные слова, соединяющие Len() с другим
// именем метода в совместное подлежащее: «Len() и Delete»,
// «Len() с Delete». Ищутся точным совпадением: короткие основы
// вроде «и» по префиксу ловили бы знаменательные слова («именно»).
var jointConnectors = map[string]bool{
	"и": true, "с": true, "со": true,
}

// jointSubject сообщает, является ли метод toks[k] совместным
// подлежащим с упомянутым Len(): между упоминанием и методом
// обязаны стоять только союзы и предлоги совместности. Знаменательное
// слово («Len() просто считает слова, а Insert возвращает ошибку»)
// или противительный союз «а» означают смену подлежащего: ошибка
// дальше к Len() не относится.
func jointSubject(toks []wordTok, k int) bool {
	for j := 0; j < k; j++ {
		if !jointConnectors[toks[j].word] {
			return false
		}
	}
	return true
}

// lenEnumeratedWithError сообщает, включает ли перечисление методов,
// которым текст утверждает ErrUninitializedTrie или ошибку, данное
// упоминание Len(): «ErrUninitializedTrie возникает на вызовы Insert,
// Search, StartsWith и Len()». От упоминания идём назад по элементам
// перечисления — именам методов и союзу «и»; сильный разделитель
// клаузы или чужое слово останавливают обход: граница или другой
// контекст разрывают связь с ошибкой («…пустого слова; Len() при
// этом…», «…Delete, а Len() при этом…»). Сильный разделитель прямо
// перед упоминанием тоже заканчивает перечисление раньше него —
// Len() за границей («…Insert, Search и Delete; Len() при этом…»)
// к перечислению не относится, но завершающее двоеточие
// двухранольно: оно либо вводит перечисление под управлением
// утверждения об ошибке, первым элементом которого стало данное
// упоминание («ErrUninitializedTrie возникает на вызовы: Len(),
// Search и Delete возвращают ошибку»), либо отделяет завершённое
// перечисление от заметки про Len() — различение в
// lenFirstListedAfterColon. Если набран хотя бы один метод и обход
// дошёл до слова об ошибке — либо ошибка названа тремя словами до
// остановки, не пересекая сильной границы, — Len() включён в
// утверждающие ошибку методы, чего на неинициализированном Trie не
// бывает (SPEC, «Минимум API»).
func lenEnumeratedWithError(before string) bool {
	toks, tailBreak := splitTokens(before)
	if tailBreak {
		return lenFirstListedAfterColon(before, toks)
	}
	methods := 0
	for k := len(toks) - 1; k >= 0; k-- {
		if toks[k].newClause {
			// Двоеточие перед перечислением не разрывает его связь
			// с ошибкой, только если оно структурно вводит
			// перечисление, управляемое утверждением об ошибке;
			// остальные сильные разделители — разрывают.
			if !toks[k].colon || !colonIntroducesErrorList(toks, k) {
				return false // перечисление из предыдущей клаузы
			}
			continue
		}
		w := toks[k].word
		if w == "и" {
			continue
		}
		if isMethodToken(w) {
			methods++
			continue
		}
		if isErrorWord(w) {
			return methods > 0
		}
		// чужое слово — ошибка может быть названа рядом, до
		// остановки и без пересечения сильной границы.
		if errorWordBefore(toks, k, shortEnumWindow) {
			return methods > 0
		}
		return false
	}
	return false
}

// listIntroStems — основы существительных, вводящих перечисление
// методов: «на вызовы: …», «на вызовы методов: …», «при операциях:
// …», «при обращениях: …».
var listIntroStems = []string{"вызов", "метод", "операци", "обращени"}

// colonIntroducesErrorList сообщает, вводит ли двоеточие перед
// позицией idx (toks[idx] — первый токен после двоеточия; idx ==
// len(toks) — конец текста) перечисление методов, управляемое
// утверждением об ошибке из той же клаузы перед двоеточием.
// Признаки структурные, а не расстояние в словах: непосредственно
// перед двоеточием стоит существительное-вводящее (listIntroStems),
// а в клаузе перед ним названо слово об ошибке — «…ErrUninitializedTrie
// до проверки входа на вызовы методов: …». Вводящее существительное
// может стоять далеко от ошибки, поэтому слово об ошибке ищется до
// сильной границы клаузы: длинная, но верная преамбула утверждения
// перечисление не отменяет. Отрицание снимает управляющее утверждение
// («не возвращает ошибку ни один из методов: …») — перечисление
// ошибочных методов не вводится. Между словом об ошибке и двоеточием
// не вправе стоять имена методов: начавшееся до двоеточия перечисление
// к нему завершилось, и двоеточие тогда лишь отделяет его от
// продолжения или заметки про Len() («…Insert, Search и Delete: Len()
// при этом возвращает 0»).
func colonIntroducesErrorList(toks []wordTok, idx int) bool {
	if idx == 0 || idx > len(toks) || !hasStem(toks[idx-1].word, listIntroStems) {
		return false // вводящего перечисление существительного нет
	}
	errIdx := -1
	for k := idx - 1; k >= 0 && !toks[k].newClause; k-- {
		if isErrorWord(toks[k].word) {
			errIdx = k
			break
		}
	}
	if errIdx == -1 || negatedErrorClaim(tokenWords(toks), errIdx) {
		return false
	}
	for k := errIdx + 1; k < idx; k++ {
		if isMethodToken(toks[k].word) {
			return false // перечисление завершилось до двоеточия
		}
	}
	return true
}

// shortEnumWindow — сколько слов перед чужим словом просматривается
// в поисках слова об ошибке: «ErrUninitializedTrie возникает на
// вызовы Insert, …» — три слова от чужого слова до ошибки.
const shortEnumWindow = 3

// lenFirstListedAfterColon сообщает, вводит ли завершающее before
// двоеточие перечисление, первым элементом которого стало данное
// упоминание Len(): «ErrUninitializedTrie возникает на вызовы:
// Len(), Search и Delete возвращают ошибку» — перечисление под
// управлением утверждения об ошибке, и Len() включён в получающие
// ошибку методы. Точка, точка с запятой и тире перечисления не
// вводят — Len() за ними к предыдущей клаузе не относится.
// Структурные признаки вводящего перечисление двоеточия — в
// colonIntroducesErrorList.
func lenFirstListedAfterColon(before string, toks []wordTok) bool {
	if trimmed := strings.TrimRight(before, " \t\n"); !strings.HasSuffix(trimmed, ":") {
		return false // точка, точка с запятой и тире перечисление не вводят
	}
	return colonIntroducesErrorList(toks, len(toks))
}

// errorWordBefore сообщает, стоит ли слово об ошибке в пределах window
// слов перед позицией idx, не пересекая другой сильной границы клаузы,
// и не снято ли его утверждение отрицанием: «ошибку не возвращает:
// Len()» утверждения об ошибке не содержит. Первое найденное слово об
// ошибке решает. Это помощник короткого взгляда назад перед чужим
// словом в lenEnumeratedWithError (shortEnumWindow); связь вводящего
// двоеточия с ошибкой проверяется структурно — в colonIntroducesErrorList.
func errorWordBefore(toks []wordTok, idx, window int) bool {
	for d := 1; d <= window && idx-d >= 0; d++ {
		if toks[idx-d].newClause {
			break
		}
		if isErrorWord(toks[idx-d].word) {
			return !negatedErrorClaim(tokenWords(toks), idx-d)
		}
	}
	return false
}

// methodStems — основы имён всех пяти методов Trie (SPEC, «Минимум
// API»): элементами перечислений методов могут быть только они.
var methodStems = []string{"insert", "search", "startswith", "delete", "len"}

// isMethodToken сообщает, является ли слово именем метода Trie.
func isMethodToken(w string) bool {
	return hasStem(w, methodStems)
}

// isErrorWord сообщает, называет ли слово ошибку: ErrUninitializedTrie
// или производное от «ошибка».
func isErrorWord(w string) bool {
	return w == "erruninitializedtrie" || hasStem(w, []string{"ошибк"})
}

// lenMentionHandled сообщает, оговорено ли данное упоминание Len():
// нулём рядом («Len() при этом возвращает 0», «возвращает ноль»,
// «= 0» — отдельно стоящий «0», но не цифра внутри числа вроде «10»),
// отрицанием ошибки — после упоминания («Len() ошибки не возвращает»,
// «Len() не возвращает ErrUninitializedTrie») или перед ним, когда
// упоминание входит в отрицаемое перечисление («ошибку не возвращают:
// Insert, Search и Len()»), — или оговоркой-исключением, называющей
// именно Len() («любой метод, кроме Len()», «за исключением Len()»,
// «но не Len()», «кроме Delete и Len()»).
// Ассоциация направленная: оговорка про чужой метод («кроме Delete;
// Len() при этом просто считает слова») упоминание не оговаривает.
func lenMentionHandled(before, after string) bool {
	if lenExceptHandled(before) {
		return true
	}
	if negatedErrorBeforeMention(before) {
		return true
	}
	words := splitWords(after)
	for k := 0; k <= 4 && k < len(words); k++ {
		switch {
		case hasStem(words[k], []string{"нул", "нол"}):
			return true
		case isErrorWord(words[k]):
			if negatedErrorClaim(words, k) {
				return true
			}
		}
	}
	return mentionsZero(after[:segmentEnd(after)])
}

// lenExceptHandled сообщает, называет ли оговорка-исключение в before
// именно Len(): ищем последний маркер оговорки («кроме», «за
// исключением», «но не») и проверяем перечисление от маркера до
// упоминания — между ними обязаны стоять только имена методов и союз
// «и», без сильного разделителя клаузы и без чужих слов. Тогда Len() —
// элемент перечисления: «кроме Len()», «кроме Delete и Len()».
// Сильный разделитель («кроме Delete; Len() при этом…») или чужое
// слово («кроме Delete, поэтому Len()…») заканчивают перечисление
// раньше, и оговорка чужого метода упоминание Len() не оговаривает.
func lenExceptHandled(before string) bool {
	toks, tailBreak := splitTokens(before)
	start := -1
	for i := 0; i < len(toks); i++ {
		switch {
		case toks[i].word == "кроме":
			start = i + 1
		case toks[i].word == "за" && i+1 < len(toks) && hasStem(toks[i+1].word, []string{"исключен"}):
			start = i + 2
		case toks[i].word == "но" && i+1 < len(toks) && toks[i+1].word == "не":
			start = i + 2
		}
	}
	if start == -1 {
		return false
	}
	for k := start; k < len(toks); k++ {
		if toks[k].newClause {
			return false // перечисление закончилось до упоминания
		}
		w := toks[k].word
		if w == "и" || isMethodToken(w) {
			continue
		}
		return false // чужое слово — Len() вне перечисления
	}
	return !tailBreak // сильный разделитель перед Len() начинает новую клаузу
}

// negatedErrorBeforeMention сообщает, снимает ли отрицание
// утверждение об ошибке, названное перед упоминанием Len() в том же
// перечислении: «ошибку не возвращает: Len()» —
// перечисление под управлением отрицательного утверждения, и Len()
// ошибки не получает. Первое слово об ошибке решает. Отрицание
// оговаривает упоминание, только если упоминание входит в
// перечисление, управляемое этим отрицанием (см.
// mentionInNegatedEnumeration): противительный оборот со сменой
// подлежащего («Insert не возвращает ошибку, а Len() просто считает
// слова») отрицанию Len() не подчиняется. Отрицание из предыдущей
// клаузы (сильный разделитель в конце before) упоминание не
// оговаривает; двоеточие — исключение: оно не разрывает связь
// отрицания с перечислением и тогда, когда Len() — его первый
// элемент («ошибку не возвращает: Len()»). Остальные сильные
// разделители упоминание от отрицания отгораживают.
func negatedErrorBeforeMention(before string) bool {
	toks, tailBreak := splitTokens(before)
	if tailBreak && !strings.HasSuffix(strings.TrimRight(before, " \t\n"), ":") {
		return false // отрицание из предыдущей клаузы; двоеточие — исключение
	}
	words := tokenWords(toks)
	for k := len(toks) - 1; k >= 0; k-- {
		if toks[k].newClause {
			if !toks[k].colon {
				return false // отрицание из предыдущей клаузы
			}
			continue
		}
		if isErrorWord(toks[k].word) {
			return negatedErrorClaim(words, k) && mentionInNegatedEnumeration(toks, k)
		}
	}
	return false
}

// mentionInNegatedEnumeration сообщает, входит ли упоминание Len()
// в перечисление, управляемое отрицательным утверждением об ошибке
// toks[errIdx]: после слова об ошибке допускаются только «не»,
// слова-сказуемые (predicateStems), вставка «метод…», союз «и» и
// имена методов. Противительное «а» («Insert не возвращает ошибку,
// а Len()…») или другое знаменательное слово означают смену
// подлежащего — отрицание к упоминанию Len() не относится; сильная
// граница, кроме двоеточия, перечисление заканчивает раньше него.
func mentionInNegatedEnumeration(toks []wordTok, errIdx int) bool {
	for k := errIdx + 1; k < len(toks); k++ {
		if toks[k].newClause && !toks[k].colon {
			return false // перечисление закончилось до упоминания
		}
		w := toks[k].word
		switch {
		case w == "не" || w == "и" || hasStem(w, predicateStems) ||
			hasStem(w, []string{"метод"}) || isMethodToken(w):
			continue
		}
		return false
	}
	return true
}

// mentionsZero сообщает, называет ли фрагмент нулевое значение
// отдельно стоящей цифрой «0» («= 0», «возвращает 0»): «0» внутри
// числа вроде «10» или «100» нулём не считается.
func mentionsZero(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' {
			continue
		}
		prevDigit := i > 0 && isASCIIDigit(s[i-1])
		nextDigit := i+1 < len(s) && isASCIIDigit(s[i+1])
		if !prevDigit && !nextDigit {
			return true
		}
	}
	return false
}

// isASCIIDigit сообщает, десятичная ли это ASCII-цифра.
func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
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
		// Сужение придаточным: ошибку получают только методы,
		// возвращающие ошибку.
		"все методы, которые возвращают ошибку, отвечают ErrUninitializedTrie",
		"каждый вызов, который даёт ошибку, отвечает ErrUninitializedTrie",
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
		// Указательные определители при существительном связку
		// не разрывают.
		"ErrUninitializedTrie возникает на любой такой вызов",
		"ошибку возвращают все подобные вызовы",
		"ErrUninitializedTrie возникает на каждый указанный вызов",
		// Исключение чужого метода Len() не оговаривает.
		"каждый метод, кроме Delete, завершается ErrUninitializedTrie",
		// Исключение ошибочного метода вместе с Len() утверждение
		// не сужает: Delete ошибку как раз возвращает.
		"каждый метод, кроме Delete и Len(), завершается ErrUninitializedTrie",
		// Сильный разделитель заканчивает перечисление исключений:
		// Len() за точкой с запятой к оговорке не относится, и
		// универсальное утверждение остаётся.
		"ErrUninitializedTrie возникает на любой вызов, кроме Delete; Len() при этом возвращает 0",
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
		// Отрицание ошибки по имени ошибки.
		"Len() не возвращает ErrUninitializedTrie",
		// Оговорка-исключение.
		"ошибку возвращает любой метод, кроме Len()",
		// Каждое из двух упоминаний оговорено.
		"Len() ошибки не возвращает, а Len() возвращает ноль",
		// Контраст неинициализированный/инициализированный: одно
		// упоминание оговорено нулём, второе — точный контекст,
		// ошибки не утверждает.
		"Len() на неинициализированном Trie возвращает 0, а Len() на инициализированном — число слов",
		// Второе упоминание — точный контекст без утверждения
		// ошибки; достаточно одной оговорки в окне.
		"Len() ошибки не возвращает; Len() при этом просто считает слова",
		// Перечисление ошибочных методов закончилось сильным
		// разделителем перед Len(): точка с запятой, точка, тире
		// и двоеточие разрывают связь перечисления с ошибкой.
		"ErrUninitializedTrie возвращают Insert, Search и Delete; Len() при этом возвращает 0",
		"ErrUninitializedTrie возвращают Insert, Search и Delete. Len() при этом возвращает 0",
		"ErrUninitializedTrie возвращают Insert, Search и Delete — Len() при этом возвращает 0",
		"ErrUninitializedTrie возвращают Insert, Search и Delete: Len() при этом возвращает 0",
		// Утверждение ошибки в следующей клаузе со сменой подлежащего
		// к Len() не относится: ошибку возвращает Insert.
		"Len() при этом возвращает 0, а Insert на пустом слове возвращает ошибку",
		// Смена подлежащего после противительного «а» при порядке
		// «ошибка, затем сказуемое, затем метод»: ошибку возвращает
		// Delete, а не Len().
		"Len() при этом возвращает 0, а ошибку возвращает Delete",
		// Равнозначный противительный оборот с «но».
		"Len() возвращает 0, но ошибку возвращает Delete",
		// Отрицание ошибки перед перечислением с двоеточием: Len() —
		// единственный метод, не получающий ошибку (SPEC, «Минимум
		// API»: Insert и Search ErrUninitializedTrie возвращают,
		// поэтому перечень не возвращающих ошибку методов длиннее
		// одного элемента был бы ложным). Двоеточие связи отрицания
		// с перечислением не разрывает, даже когда Len() — его
		// первый элемент.
		"ошибку не возвращает: Len()",
		// Двоеточие отделяет завершённое перечисление ошибочных
		// методов от заметки про Len() и при длинной, но верной
		// преамбуле: расстояние до слова об ошибке роли не играет.
		"неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа на вызовы Insert, Search, StartsWith и Delete: Len() при этом возвращает 0",
		// Отрицание находит общий помощник errorWordBefore из обхода
		// перечисления: слово об ошибке стоит перед вводящим
		// двоеточием, отрицание управляет перечислением, и Len() в
		// нём ошибки не получает.
		"ошибку не возвращает: Insert и Len()",
		// Отрицание в коротком взгляде назад: перечисление
		// «Insert и Len()» под отрицательным утверждением, ошибки
		// Len() не получает.
		"ошибку не возвращает Insert, и Len() при этом возвращает 0",
		// Отрицание из предыдущей клаузы (точка с запятой) и после
		// сильной границы посередине упоминание не оговаривают —
		// оговоркой служит ноль после упоминания.
		"ошибку не возвращает Delete; Len() при этом возвращает 0",
		"ошибку не возвращает Delete; при этом Len() возвращает 0",
		// Двоеточие после далёкого слова об ошибке (за пределами окна
		// поиска) отделяет самостоятельную заметку про Len():
		// перечисление ошибочных методов не вводится, и Len() к
		// утверждению об ошибке не относится.
		"ErrUninitializedTrie возникает на неинициализированном Trie: Len() при этом возвращает 0",
		// Оборот «ошибка → сказуемое → подлежащее» с безобидными
		// вставками: наречие «всегда» и существительное «метод»
		// exemption не мешают — ошибку назначает Delete, а не Len().
		"Len() возвращает 0, а ошибку всегда возвращает Delete",
		"Len() возвращает 0, а ошибку возвращает метод Delete",
		// Ошибка в следующей клаузе после точки с запятой упоминанию
		// Len() не принадлежит: ошибку возвращает Insert.
		"Len() возвращает 0; Insert возвращает ошибку",
	}
	erroneous := []string{
		// Len() включён в утверждающие ошибку методы.
		"Insert, Search, StartsWith, Delete и Len() возвращают ErrUninitializedTrie",
		"Len() возвращает ErrUninitializedTrie",
		"Len() возвращает ошибку",
		// Длинное утверждение ошибки в той же клаузе: между Len() и
		// словом об ошибке несколько слов.
		"Len() на неинициализированном Trie возвращает ошибку",
		// Len() назван в перечислении до сильного разделителя:
		// граница после него ассоциацию с ошибкой не разрывает.
		"ErrUninitializedTrie возвращают Insert, Search, StartsWith и Len(); Delete при этом возвращает ошибку",
		"ErrUninitializedTrie возвращают Insert, Search, StartsWith и Len(): Delete при этом возвращает ошибку",
		// Упоминание Len() без нуля, отрицания или исключения не
		// считается оговоркой.
		"Len() при этом просто считает слова",
		// Оговорка «кроме …» называет чужой метод: точка с запятой
		// заканчивает перечисление, и Len() к нему не относится.
		"ErrUninitializedTrie возникает на любой вызов, кроме Delete; Len() при этом просто считает слова",
		// Отрицание при наречии меры утверждение об ошибке не
		// снимает: «не всегда» — значит, иногда возвращает.
		"Len() возвращает ошибку не всегда",
		// «0» внутри числа вроде 10 нулём не считается.
		"Len() возвращает 10",
		// Совместное подлежащее: Len() назван вместе с Delete в
		// утверждающих ошибку методах.
		"Len() и Delete возвращают ErrUninitializedTrie",
		"Len() и Delete возвращают ошибку",
		// Противительные «а»/«но» exemption не дают сами по себе:
		// продолжение утверждения («а также», «а иногда», «а при
		// пустом слове») оставляет ошибку при Len().
		"Len() возвращает 0, а также ошибку",
		"Len() возвращает 0, а иногда ErrUninitializedTrie",
		"Len() возвращает 0, а при пустом слове возвращает ошибку",
		// Двоеточие вводит перечисление, первым элементом которого
		// стал Len(): позднейшая верная заметка про Len() это не
		// спасает — упоминание в перечислении не оговорено.
		"ErrUninitializedTrie возникает на вызовы: Len(), Search и Delete возвращают ошибку",
		"ErrUninitializedTrie возникает на вызовы: Len(), Search и Delete возвращают ошибку, но Len() при этом возвращает 0",
		// Двоеточие вводит перечисление, управляемое предыдущим
		// утверждением об ошибке: Len() внутри перечисления.
		"ErrUninitializedTrie возникает на вызовы: Insert, Search, StartsWith и Len()",
		// То же с длинной, но верной преамбулой утверждения:
		// структурные признаки вводящего двоеточия (существительное
		// «методов» непосредственно перед ним) расстояние до слова
		// об ошибке не отменяют, и позднейшая верная заметка про
		// Len() упоминание в перечислении не спасает.
		"неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа на вызовы методов: Len(), Search и Delete возвращают ошибку, но Len() при этом возвращает 0",
		// То же перечисление с Len() внутри списка, а не первым
		// элементом.
		"ErrUninitializedTrie возникает на вызовы методов: Insert, Search, Len() и Delete возвращают ошибку, но Len() при этом возвращает 0",
		// Отрицание относится к Insert: противительный оборот «а»
		// меняет подлежащее, поэтому отрицание не оговаривает
		// упоминание Len() — собственной оговорки (ноль, отрицание,
		// исключение) у него нет.
		"Insert не возвращает ошибку, а Len() просто считает слова",
		// Оборот «ошибка → сказуемое → подлежащее» не вправе назначить
		// ошибку самому Len(): новый субъект с len-основой exemption
		// не даёт — утверждение остаётся при Len().
		"Len() возвращает ноль, а ошибку возвращает Len",
		// Координированный список подлежащих: «Delete и Len»
		// утверждает ошибку и для Len(), exemption не действует.
		"Len() возвращает 0, а ошибку возвращает Delete и Len",
		// Знаменательное слово между ошибкой и методом продолжает
		// утверждение про Len(): exemption не засчитывается, даже если
		// дальше назван Delete.
		"Len() возвращает 0, а также ошибку, которую возвращает Delete",
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

// Таблица проверяет связанное окно из клаузы про ErrUninitializedTrie
// и следующего за ней предложения: оговорка про Len() не обязана
// делить с ошибкой одно предложение (SPEC, «Минимум API»), но
// требование хотя бы одной оговорки и запрет утверждать ошибку для
// Len() те же, что в lenWordingSafe (тикет 43, ревью).
func TestLenNoteSafe(t *testing.T) {
	accurate := []struct{ clause, next string }{
		{
			// Формулировка из банка: оговорка в той же клаузе.
			clause: "неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа; Len() при этом возвращает 0",
		},
		{
			// Оговорка про Len() в следующем предложении.
			clause: "неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа на вызовы Insert, Search, StartsWith и Delete — даже вместо ошибок пустого слова",
			next:   "Len() при этом возвращает 0",
		},
		{
			// Отрицание ошибки по имени ошибки в отдельном предложении.
			clause: "ErrUninitializedTrie до проверки входа возвращают Insert, Search, StartsWith и Delete",
			next:   "Len() не возвращает ErrUninitializedTrie",
		},
		{
			// Контраст неинициализированный/инициализированный:
			// упоминание в клаузе оговорено нулём, второе — точный
			// контекст, ошибки не утверждает.
			clause: "неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа; Len() при этом возвращает 0",
			next:   "Len() на инициализированном Trie возвращает число слов",
		},
		{
			// Самостоятельная заметка о сложности Len() следующим
			// предложением ошибку не утверждает.
			clause: "неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа; Len() при этом возвращает 0",
			next:   "Len() выполняется за константное время",
		},
		{
			// Второе упоминание — точный контекст без утверждения
			// ошибки; достаточно одной оговорки в окне.
			clause: "неинициализированный Trie отвечает ErrUninitializedTrie до проверки входа",
			next:   "Len() ошибки не возвращает; Len() при этом просто считает слова",
		},
		{
			// Двоеточие отделяет завершённое перечисление ошибочных
			// методов от заметки про Len(): Len() за двоеточием
			// к перечислению не относится.
			clause: "ErrUninitializedTrie возвращают Insert, Search и Delete: Len() при этом возвращает 0",
		},
	}
	erroneous := []struct{ clause, next string }{
		{
			// Len() включён в методы, получающие ошибку, — оговорка
			// следующим предложением это не спасает: упоминание
			// в клаузе не оговорено.
			clause: "ErrUninitializedTrie возникает на вызовы Insert, Search, StartsWith и Len()",
			next:   "Len() при этом возвращает 0",
		},
		{
			// Оговорка «кроме …» из клаузы не засчитывается
			// упоминанию Len() в следующем предложении.
			clause: "ошибку возвращает любой метод, кроме Delete",
			next:   "Len() при этом просто считает слова",
		},
		{
			// Оговорка «кроме …» называет чужой метод: точка с
			// запятой заканчивает перечисление, и Len() к нему
			// не относится.
			clause: "ErrUninitializedTrie возникает на любой вызов, кроме Delete; Len() при этом просто считает слова",
		},
		{
			// Двоеточие вводит перечисление, управляемое предыдущим
			// утверждением об ошибке: Len() внутри перечисления,
			// и оговорка следующим предложением это не спасает.
			clause: "ErrUninitializedTrie возникает на вызовы: Insert, Search, StartsWith и Len()",
			next:   "Len() при этом возвращает 0",
		},
	}
	for _, tc := range accurate {
		if !lenNoteSafe(tc.clause, tc.next) {
			t.Errorf("точная пара предложений про Len() распознана как ошибочная: %q | %q", tc.clause, tc.next)
		}
	}
	for _, tc := range erroneous {
		if lenNoteSafe(tc.clause, tc.next) {
			t.Errorf("небезопасная пара предложений про Len() не распознана: %q | %q", tc.clause, tc.next)
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
	// принимали бы случайные совпадения за клаузу. Клауза изолирована:
	// её проверки (методы, контекст, порядок) не тянут соседние
	// предложения.
	sent := sentences(q.Explanation)
	var clause string
	next := ""
	clauseIdx := -1
	errSentences := 0
	for i, s := range sent {
		if strings.Contains(s, "ErrUninitializedTrie") {
			errSentences++
			clause, clauseIdx = s, i
		}
	}
	if errSentences != 1 {
		t.Fatalf("в пояснении должно быть ровно одно предложение с ErrUninitializedTrie, найдено %d: %q",
			errSentences, q.Explanation)
	}
	if clauseIdx+1 < len(sent) {
		next = sent[clauseIdx+1]
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
	// (SPEC, «Минимум API»): каждое утверждение ошибки для Len()
	// отвергается — словами после упоминания или включением Len() в
	// перечень методов, получающих ошибку, — а в связанном окне
	// обязательна хотя бы одна действующая оговорка: нулём,
	// отрицанием ошибки или оговоркой-исключением. Дополнительные
	// упоминания Len() без собственной оговорки допустимы, пока
	// ошибку они не утверждают. SPEC не требует делить одно
	// предложение: оговорка про Len() может стоять и следующим
	// предложением — проверяем связанное окно из клаузы и следующего
	// за ней предложения. Распознавание формулировок — в
	// lenNoteSafe и его табличных тестах.
	if !containsAny(clause+" "+next, "Len()") {
		t.Fatalf("в клаузе про ErrUninitializedTrie и следующем за ней предложении нет оговорки про Len(): %q | %q",
			clause, next)
	}
	if !lenNoteSafe(clause, next) {
		t.Errorf("клауза и следующее предложение неверно описывают Len() на неинициализированном Trie — включая его в утверждающие ошибку методы или не оговаривая ноль: %q | %q",
			clause, next)
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
