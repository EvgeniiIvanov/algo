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

// guaranteeStems — основы слов о гарантии или обещании: «гаранти-»
// (гарантия, гарантированный, гарантировать), «обеща-» (обещать,
// обещается).
var guaranteeStems = []string{"гаранти", "обеща"}

// complexityStems — основы слов о сложности поиска: «сложност-»
// (сложность), «логарифм-» (логарифмический) и запись оценки «log»
// (O(log n)).
var complexityStems = []string{"сложност", "логарифм", "log"}

// giveVerbStems — основы личных форм глагола «дать», которыми
// выражается обещание результата: «не даёт гарантии», «гарантии не
// давал». Список покрывает формы «даёт/дает» (настоящее, ед. ч.),
// «дают» (настоящее, мн. ч.), «даст» и «дадут» (будущее), «дали» и
// «давал» (прошедшее); остальных форм («дать», «даём», «давать» и
// других) в списке нет. Отрицание при такой форме снимает обещание.
var giveVerbStems = []string{"даёт", "дает", "даст", "дадут", "дают", "дали", "давал"}

// copulaForms — формы глагола-связки «являться»: в обороте «не является
// гарантией» отрицание стоит при связке, а связка — при слове
// гарантии.
var copulaForms = []string{"является", "являются"}

// negationIntensifiers — усиления между «нет»/«не даёт» и словом
// гарантии: «нет никакой гарантии», «нет никаких гарантий», «не даётся
// никакая гарантия». Усиление экзистенциальное отрицание не разрывает.
var negationIntensifiers = []string{"никакой", "никаких", "никакая"}

// negatesComplexityGuarantee сообщает, есть ли в тексте предложение, где
// отрицание связано именно с обещанием гарантии сложности, а само
// предложение говорит о сложности. Отрицание распознаётся в формах, где
// оно примыкает к обещанию: «не» при слове гарантии или обещания («не
// гарантия», «сложность не гарантирована», «не обещается», «не
// гарантирует»); «не» при связке перед словом гарантии («не является
// гарантией»); «не» при глаголе «дать», снимающее обещание в своей
// клаузе: обещание — дополнение глагола («не даёт гарантии сложности»,
// «не даёт никакой гарантии», «не дают гарантии») или — в инверсии,
// где глагол закрывает клаузу, — названо до глагола
// («гарантированной логарифмической сложности оно не даёт»); усиленное
// обещание ищется и за обстоятельством-оговоркой «в … случае»
// («не даёт в общем случае никакой гарантии»); при страдательной форме
// «не даётся» снятым обещанием служит само подлежащее («гарантия
// сложности не даётся»), а «не даётся никакая гарантия» узнаёт тот же
// пост-глагольный поиск с усилением, что и «не даёт никакой гарантии»,
// — отдельной ветви для него нет; экзистенциальное «нет» —
// рядом со словом гарантии («нет гарантии»), через усиление («нет
// никакой гарантии») или в инверсии, где между гарантией и «нет» стоит
// зависимая сложность («гарантии сложности нет»). Поэтому
// формулировки «это не гарантия сложности», «сложность не
// гарантирована», «гарантированной логарифмической сложности оно не
// даёт», «не даёт никакой гарантии сложности», «не даёт в общем случае
// никакой гарантии сложности», «гарантия сложности не даётся», «не
// даётся никакая гарантия сложности», «не дают гарантии», «нет никакой
// гарантии», «не является гарантией», «гарантии сложности нет»,
// «гарантированной сложности поиск не даёт в общем случае» считаются
// эквивалентными: точный порядок слов роли не играет, и
// переформулировка урока проверку не ломает. Связь отрицания с
// обещанием ищется в своей клаузе: за сильной границей клаузы («не
// даёт отсечения; гарантия … есть») слово гарантии отрицание не
// поддерживает, и «не» после сильной границы начинает новую клаузу —
// гарантия из предыдущей клаузы ему не подчиняется («…гарантии
// равномерности; не даётся это бесплатно»). Единственное смягчение —
// тире, и то лишь в локальном порядке «подлежащее — сказуемое»,
// когда сказуемое стоит сразу после тире и закрывает клаузу,
// открытую тире («Гарантия сложности — не даётся»): это прямой
// порядок слов, а не инверсия; между тире и сказуемым других слов
// быть не должно («Гарантия сложности — это не даётся бесплатно» не
// распознаётся). Отрицание при другом слове
// проверку не проходит, даже если гарантия и сложность стоят рядом:
// «гарантия сложности не важна — важна практика» отрицание гарантии не
// выражает; не проходит и действительное «не даёт» с другим
// дополнением — «гарантия сложности не даёт выигрыша» говорит о
// подлежащем-гарантии, а не снимает её, как и с оговоркой перед
// дополнением («гарантия сложности не даёт в общем случае выигрыша»);
// так же не проходят
// голое «не» в другом предложении и утверждение гарантии без отрицания.
func negatesComplexityGuarantee(text string) bool {
	for _, s := range sentences(text) {
		toks, _ := splitTokens(s)
		if !anyTokHasStem(toks, complexityStems) {
			continue
		}
		for i, tk := range toks {
			switch tk.word {
			case "не":
				// «Не» отрицает примыкающее слово гарантии:
				// «не гарантия», «не гарантирована», «не
				// обещается».
				if hasAdjacentStem(toks, i, guaranteeStems) {
					return true
				}
				// «Не является гарантией»: отрицание стоит
				// при связке, а связка — при слове
				// гарантии.
				if i+2 < len(toks) &&
					hasStem(toks[i+1].word, copulaForms) &&
					hasStem(toks[i+2].word, guaranteeStems) {
					return true
				}
				// «Не даёт …», «не даются …»: отрицание
				// глагола «дать» снимает обещание той
				// формы, которую глагол принимает; за
				// границей клаузы связь отрицания с
				// обещанием теряется.
				if i+1 < len(toks) &&
					hasStem(toks[i+1].word, giveVerbStems) {
					// Страдательная форма: слово гарантии
					// в клаузе — подлежащее, и отрицание
					// снимает его («гарантия сложности
					// не даётся»). Тире перед сказуемым
					// допустимо лишь в локальном порядке
					// «подлежащее — сказуемое»: сказуемое
					// стоит сразу после тире и закрывает
					// клаузу, открытую тире («Гарантия
					// сложности — не даётся», в том числе
					// с двоеточием после сказуемого), и
					// giveObjectEmpty это проверяет;
					// промежуточные слова между тире и
					// сказуемым («Гарантия сложности —
					// это не даётся бесплатно») и
					// продолжение клаузы за сказуемым
					// связку с подлежащим не доказывают.
					// Усиленное обещание после «не
					// даётся» отдельной ветви не имеет —
					// его узнаёт общий пост-глагольный
					// поиск с усилением ниже.
					if reflexiveGiveForm(toks[i+1].word) &&
						stemBeforeInClause(toks, i, guaranteeStems) &&
						(!toks[i].dash || giveObjectEmpty(toks, i+1)) {
						return true
					}
					// Действительная форма: отрицание
					// снимает названное глаголом
					// дополнение. Обещание стоит сразу за
					// глаголом («не даёт гарантии») или
					// через усиление («не даёт никакой
					// гарантии»); в инверсии обещание
					// стоит до глагола, а сам глагол
					// закрывает клаузу — подлежащее
					// («оно») разрывает дополнение и
					// глагол: «гарантированной
					// логарифмической сложности оно не
					// даёт».
					if i+2 < len(toks) &&
						sameClause(toks, i, i+2) &&
						hasStem(toks[i+2].word, guaranteeStems) {
						return true
					}
					if i+3 < len(toks) &&
						sameClause(toks, i, i+3) &&
						hasStem(toks[i+2].word, negationIntensifiers) &&
						hasStem(toks[i+3].word, guaranteeStems) {
						return true
					}
					// Усиленное обещание может стоять и за
					// обстоятельством-оговоркой, которая
					// слот дополнения не заполняет: «не
					// даёт в общем случае никакой
					// гарантии».
					if hedge := caseHedgeLen(toks, i+1); hedge > 0 {
						k := i + 2 + hedge
						if k+1 < len(toks) &&
							sameClause(toks, i, k+1) &&
							hasStem(toks[k].word, negationIntensifiers) &&
							hasStem(toks[k+1].word, guaranteeStems) {
							return true
						}
					}
					// В инверсии обещание стоит до глагола,
					// если у глагола дополнения нет: клауза
					// кончается за ним или за оговоркой
					// («гарантированной сложности поиск не
					// даёт в общем случае»). Если же за
					// глаголом в клаузе названо другое
					// дополнение, отрицание относится к
					// нему, а не к гарантии: «гарантия
					// сложности не даёт выигрыша» —
					// гарантия тут подлежащее, и её
					// отрицание не снимается.
					if giveObjectEmpty(toks, i+1) &&
						stemBeforeInClause(toks, i, guaranteeStems) &&
						stemBeforeInClause(toks, i, complexityStems) {
						return true
					}
				}
			case "нет":
				// «Нет» — экзистенциальное отрицание: рядом
				// со словом гарантии («нет гарантии»), через
				// усиление («нет никакой гарантии») или в
				// инверсии, где между гарантией и «нет»
				// стоит зависимая сложность («гарантии
				// сложности нет»). Связь ищется в своей
				// клаузе; тире допустимо лишь в локальном
				// порядке «подлежащее — сказуемое»:
				// сказуемое «нет» открывает клаузу сразу
				// за тире («гарантия сложности — нет»), а
				// не за другой сильной границей клаузы —
				// инверсию проверяет
				// inversionGuaranteeComplexity.
				if hasAdjacentStem(toks, i, guaranteeStems) {
					return true
				}
				if i+2 < len(toks) &&
					sameClause(toks, i, i+2) &&
					hasStem(toks[i+1].word, negationIntensifiers) &&
					hasStem(toks[i+2].word, guaranteeStems) {
					return true
				}
				if inversionGuaranteeComplexity(toks, i) {
					return true
				}
			}
		}
	}
	return false
}

// reflexiveGiveForm сообщает, страдательная ли это форма глагола
// «дать»: возвратная форма с постфиксом «-ся/-сь» («даётся»,
// «даются»). У такой формы слово гарантии перед глаголом —
// подлежащее, поэтому отрицание при глаголе снимает его.
func reflexiveGiveForm(word string) bool {
	return hasStem(word, giveVerbStems) &&
		(strings.HasSuffix(word, "ся") || strings.HasSuffix(word, "сь"))
}

// giveObjectEmpty сообщает, что у глагола toks[v] в своей клаузе нет
// названного дополнения: клауза кончается сразу за глаголом либо за
// обстоятельством-оговоркой «в … случае» (caseHedgeLen), занимающей
// остаток клаузы. Названное дополнение («не даёт выигрыша», «не даёт в
// общем случае выигрыша») слот занимает, и инверсия с обещанием перед
// глаголом засчитана быть не может: «гарантия сложности не даёт
// выигрыша». Для страдательной формы («не даётся») та же проверка
// работает как проверка локальности тире: сказуемое обязано закрывать
// клаузу, чтобы связка с подлежащим через тире засчитывалась
// («Гарантия сложности — не даётся», но не «…— не даётся бесплатно»).
func giveObjectEmpty(toks []wordTok, v int) bool {
	if end := v + 1 + caseHedgeLen(toks, v); end < len(toks) {
		return toks[end].newClause
	}
	return true
}

// caseHedgeLen сообщает длину обстоятельства-оговорки, стоящего
// непосредственно за глаголом toks[v]: «в общем случае», «в худшем
// случае», «в таком случае» — предлог «в», одно слово между ним и
// словом «случая», всего три токена. Такая оговорка слот дополнения не
// заполняет: за ней ищется обещание («не даёт в общем случае никакой
// гарантии») или конец клаузы («не даёт в общем случае»); 0 означает,
// что оговорки за глаголом нет.
func caseHedgeLen(toks []wordTok, v int) int {
	if v+3 < len(toks) && toks[v+1].word == "в" && hasStem(toks[v+3].word, []string{"случа"}) {
		return 3
	}
	return 0
}

// inversionGuaranteeComplexity сообщает, что перед позицией i стоят два
// слова — гарантия (i-2) и её зависимая сложность (i-1): инверсия
// «гарантии сложности нет». Связь ищется в своей клаузе: гарантия и
// зависимая сложность стоят в одной клаузе, а «нет» либо продолжает
// её, либо — единственное допускаемое тире — открывает свою клаузу
// сразу за тире в локальном порядке «подлежащее — сказуемое»
// («гарантия сложности — нет»). Тире между гарантией и её зависимой
// сложностью («у отсечения гарантии — сложности нет») и любая другая
// сильная граница клаузы («у отсечения гарантии; сложности нет»)
// связь не доказывают, и инверсия не засчитывается.
func inversionGuaranteeComplexity(toks []wordTok, i int) bool {
	return i >= 2 &&
		hasStem(toks[i-2].word, guaranteeStems) &&
		hasStem(toks[i-1].word, complexityStems) &&
		sameClause(toks, i-2, i-1) &&
		(!toks[i].newClause || toks[i].dash)
}

// hasAdjacentStem сообщает, есть ли у позиции i слово-сосед (слева или
// справа) с одной из основ.
func hasAdjacentStem(toks []wordTok, i int, stems []string) bool {
	return (i > 0 && hasStem(toks[i-1].word, stems)) ||
		(i+1 < len(toks) && hasStem(toks[i+1].word, stems))
}

// anyTokHasStem сообщает, начинается ли хотя бы одно слово токенов с
// одной из основ.
func anyTokHasStem(toks []wordTok, stems []string) bool {
	for _, tk := range toks {
		if hasStem(tk.word, stems) {
			return true
		}
	}
	return false
}

// sameClause сообщает, лежат ли два токена в одной клаузе: между ними
// нет токена с сильной границей клаузы перед ним (признак newClause).
// Запятая клаузу не кончает — новую клаузу открывает только токен,
// перед которым стоял сильный разделитель.
func sameClause(toks []wordTok, i, j int) bool {
	if i < 0 || j < 0 || i >= len(toks) || j >= len(toks) {
		return false
	}
	lo, hi := i, j
	if lo > hi {
		lo, hi = hi, lo
	}
	for k := lo + 1; k <= hi; k++ {
		if toks[k].newClause {
			return false
		}
	}
	return true
}

// stemBeforeInClause сообщает, стоит ли в той же клаузе, что и токен i,
// более ранний токен с одной из основ: отрицанный глагол «не даёт»
// снимает названное в клаузе обещание независимо от порядка слов —
// «гарантированной сложности оно не даёт». Если токен i сам отмечен
// признаком newClause, он — первое слово своей клаузы («…гарантии
// равномерности; не даётся это бесплатно»), и опираться на слово
// гарантии из предыдущей клаузы нельзя; исключение — тире, но только
// в локальном порядке «подлежащее — сказуемое»: тире отделяет
// сказуемое от подлежащего, а сказуемое закрывает клаузу, открытую
// тире («Гарантия сложности — не даётся» — это прямой порядок слов, а
// не инверсия). Промежуточных слов между тире и сказуемым быть не
// должно: если клаузу после тире открывает не сказуемое, а другое
// слово («Гарантия сложности — это не даётся бесплатно»), тире границу
// не смягчает — поиск дальше токена i не идёт, и любая граница внутри
// него связь разрывает. Локальность узора за границей проверяет
// вызывающий код через giveObjectEmpty. Токен
// с признаком newClause сам ещё принадлежит клаузе i, а слова раньше
// него — уже нет, поэтому поиск влево на сильной границе
// заканчивается.
func stemBeforeInClause(toks []wordTok, i int, stems []string) bool {
	if toks[i].newClause && !toks[i].dash {
		return false
	}
	for j := i - 1; j >= 0; j-- {
		if hasStem(toks[j].word, stems) {
			return true
		}
		if toks[j].newClause {
			return false
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
// colon — что этой границей было двоеточие, dash — что тире.
// Двоеточие двухранольно: оно вводит перечисление, управляемое
// предыдущим утверждением об ошибке («…возникает на вызовы:
// Insert…»), или отделяет завершённое перечисление от заметки про
// Len() («…и Delete: Len() при этом…»); различение по месту
// применения — в lenEnumeratedWithError и lenFirstListedAfterColon.
// Тире из сильных границ самое мягкое, но его исключение узкое: тире
// смягчает границу лишь в локальном порядке «подлежащее — сказуемое»,
// когда сказуемое стоит сразу после тире и закрывает клаузу,
// открытую тире («гарантия сложности — нет», «Гарантия сложности —
// не даётся»); проверяют это inversionGuaranteeComplexity и
// stemBeforeInClause вместе с giveObjectEmpty. Для произвольных слов
// тире остаётся сильной границей, а последовательность разделителей
// жёсткая в любом порядке: тире смягчает только одиночное тире
// (splitTokens).
type wordTok struct {
	word      string
	newClause bool
	colon     bool
	dash      bool
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
// каждое слово признаком сильной границы клаузы перед ним и её видом:
// двоеточие (colon) и тире (dash). Тире — единственный мягкий
// разделитель, и мягким остаётся лишь одиночное тире: любой более
// жёсткий разделитель последовательности («; —», «: —», но и «— ;»,
// «— :») задаёт жёсткую границу, тире её ни перед ним, ни после него
// не смягчает, и повторное тире («— —») границу тоже не смягчает.
// Второй результат сообщает,
// стоит ли сильный разделитель после последнего слова: перечисление,
// дойдя до него, закончилось, не назвав Len() («…кроме Delete; Len()
// при этом…»).
func splitTokens(text string) (toks []wordTok, tailBreak bool) {
	var b strings.Builder
	inWord := false
	pendingBreak := false
	pendingColon := false
	pendingDash := false
	flush := func() {
		if inWord {
			toks = append(toks, wordTok{
				word:      strings.ToLower(b.String()),
				newClause: pendingBreak,
				colon:     pendingColon,
				dash:      pendingDash,
			})
			b.Reset()
			inWord = false
			pendingBreak = false
			pendingColon = false
			pendingDash = false
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
			isDash := r == '—' || r == '–'
			if !pendingBreak || pendingDash {
				// Первый разделитель последовательности
				// задаёт вид границы; если это тире, то
				// следующий более жёсткий разделитель
				// («— ;», «— :») границу возвращает к
				// жёсткой, а тире после жёсткого
				// разделителя («; —», «: —») её не
				// смягчает.
				if pendingDash && isDash {
					// Повторное тире: смягчает только
					// одиночное тире, дубль делает
					// границу жёсткой («— —»).
					pendingDash = false
				} else {
					pendingColon = r == ':'
					pendingDash = isDash
				}
			}
			pendingBreak = true
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
			if j < 0 || j >= len(words) || j == i || !hasStem(words[j], callNounStems) {
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
// клаузе вправе стоять только союзы «и»/«или», предлоги совместности
// «с»/«со» (jointConnectors), вставки и имена методов, и Len в этом
// списке («а ошибку возвращает Delete и Len», «а ошибку возвращает
// Delete с Len()») означает, что ошибка утверждена и ему, —
// exemption не даётся.
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
		case jointConnectors[w] || w == "или" || errorSubjectFillers[w]:
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
// Используются в jointSubject, в methodFollowsErrorPredicate
// («а ошибку возвращает Delete с Len()») и в обратном обходе
// перечисления в lenEnumeratedWithError («…Insert, Search и Delete
// с Len()» — Len в совместном подлежащем перечисления).
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
// перечисления — именам методов, союзу «и» и предлогам совместности
// «с»/«со» (jointConnectors); сильный разделитель
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
// lenFirstListedAfterColon. Если обход пересёк подтверждённое
// двоеточие, связь перечисления с ошибкой уже проверена структурно
// (colonIntroducesErrorList), и упоминание — элемент перечисления
// (первый или последующий), включённый в утверждающие ошибку
// методы, чего на неинициализированном Trie не бывает (SPEC,
// «Минимум API»); искать слово об ошибке дальше не нужно, за
// вводящим существительным его уже нашёл colonIntroducesErrorList.
// Без двоеточия ошибка должна быть названа рядом: либо обход дошёл
// до слова об ошибке, либо она стоит не дальше трёх слов до чужого
// слова, остановившего обход, — иначе связь с ошибкой не доказана.
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
			return true // упоминание — элемент перечисления под ошибкой
		}
		w := toks[k].word
		if jointConnectors[w] {
			continue // союз «и», предлоги «с»/«со» — перечисление продолжается
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

// callNounStems — основы существительных, называющих обращение к
// методам Trie: «вызов», «метод», «операция», «обращение». Один
// список на две проверки: в universalCallWording — существительное
// при кванторе («на любой вызов», «все методы»), в
// colonIntroducesErrorList — существительное, вводящее перечисление
// методов: «на вызовы: …», «на вызовы методов: …», «при операциях:
// …», «при обращениях: …».
var callNounStems = []string{"вызов", "метод", "обращени", "операци"}

// additiveAForms — слова, превращающие противительное «а» в добавочное
// или уточняющее: «а также», «а именно» и «а иногда» продолжают
// перечисление объектов того же утверждения, а не начинают новое
// («ErrUninitializedTrie возникает при пустом слове, а иногда на
// вызовы: …»). Прочие формы «а» противительные: они начинают новое
// утверждение, которому двоеточие подчиняется лишь при поправке.
// Список узкий: каждое слово обязано и само по себе продолжать
// утверждение об ошибке после «а» в реальных формулировках, поэтому
// расширение — только с новой оговоркой-примером.
var additiveAForms = map[string]bool{
	"также":  true,
	"именно": true,
	"иногда": true,
}

// colonIntroducesErrorList сообщает, вводит ли двоеточие перед
// позицией idx (toks[idx] — первый токен после двоеточия; idx ==
// len(toks) — конец текста) перечисление методов, управляемое
// утверждением об ошибке из той же клаузы перед двоеточием.
// Признаки структурные, а не расстояние в словах: непосредственно
// перед двоеточием стоит существительное-вводящее (callNounStems) или
// уточняющий оборот «а именно:» после такого существительного
// («…на вызовы, а именно: Len(), Search и Delete…»), а в клаузе перед
// ним названо слово об ошибке — «…ErrUninitializedTrie
// до проверки входа на вызовы методов: …». Вводящее существительное
// может стоять далеко от ошибки, поэтому слово об ошибке ищется до
// сильной границы клаузы: длинная, но верная преамбула утверждения
// перечисление не отменяет. Противительные «а»/«но» отделяют
// утверждение об ошибке от вводящего существительного, когда после
// противительного начинается отдельное утверждение, и двоеточие ему
// не подчиняется («…ErrUninitializedTrie возник до проверки входа, а
// сложность операций: Len() при этом возвращает 0» — заметка о
// сложности перечисление ошибочных методов не вводит). «Но» отделяет
// всегда, а «а» — только со сменой утверждения: добавочные «а также»
// и «а иногда», уточняющее «а именно» продолжают перечисление
// объектов того же утверждения («…ErrUninitializedTrie возникает
// при пустом слове, а иногда на вызовы: …»), а поправительное
// «не X, а Y»
// («…ErrUninitializedTrie возникает не на слове, а на вызовы: …»)
// отрицает первый объект общего сказуемого — «не» стоит между словом
// об ошибке и «а», не непосредственно после него, где «не» снимало бы
// само утверждение («ErrUninitializedTrie не возникает, а сложность
// операций: …»). При нескольких противительных «а» двоеточие
// управляется ближайшим к нему «а», и отрицание из сегмента более
// ранней поправки ему не одалживается (см.
// negationBetweenErrorAndAdversative). Отрицание снимает
// управляющее утверждение
// («не возвращает ошибку ни один из методов: …») — перечисление
// ошибочных методов не вводится. Между словом об ошибке и двоеточием
// не вправе стоять имена методов: начавшееся до двоеточия перечисление
// к нему завершилось, и двоеточие тогда лишь отделяет его от
// продолжения или заметки про Len() («…Insert, Search и Delete: Len()
// при этом возвращает 0»).
func colonIntroducesErrorList(toks []wordTok, idx int) bool {
	if idx <= 0 || idx > len(toks) {
		return false
	}
	nounIdx := idx - 1
	// Уточняющий оборот «а именно:» непосредственно после вводящего
	// существительного продолжает то же перечисление: «…на вызовы,
	// а именно: Len(), Search и Delete…». Без существительного
	// («…а именно: Len()…») двоеточие перечисление ошибочных
	// методов не вводит.
	if nounIdx >= 1 && toks[nounIdx].word == "именно" && toks[nounIdx-1].word == "а" {
		nounIdx -= 2
	}
	if nounIdx < 0 || !hasStem(toks[nounIdx].word, callNounStems) {
		return false // вводящего перечисление существительного нет
	}
	errIdx := -1
	adversative := -1 // ближайшее к двоеточию противительное «а»
	for k := idx - 1; k >= 0 && !toks[k].newClause; k-- {
		if toks[k].word == "но" {
			break // противительное «но» отделяет отдельное утверждение
		}
		if toks[k].word == "а" {
			// Добавочные «а также»/«а иногда» и уточняющее
			// «а именно» продолжают утверждение об ошибке.
			if k+1 < idx && additiveAForms[toks[k+1].word] {
				continue
			}
			if adversative == -1 {
				adversative = k // ближайшее к двоеточию «а» — именно им двоеточие и управляется
			}
			continue // поправительное ли это «а» — решаем после слова об ошибке
		}
		if isErrorWord(toks[k].word) {
			errIdx = k
			break
		}
	}
	if errIdx == -1 || negatedErrorClaim(tokenWords(toks), errIdx) {
		return false
	}
	if adversative != -1 && !negationBetweenErrorAndAdversative(toks, errIdx, adversative) {
		return false // «а» начинает отдельное утверждение
	}
	for k := errIdx + 1; k < idx; k++ {
		if isMethodToken(toks[k].word) {
			return false // перечисление завершилось до двоеточия
		}
	}
	return true
}

// negationBetweenErrorAndAdversative сообщает, является ли
// противительное «а» toks[advIdx] поправительным оборотом
// «не X, а Y» при утверждении об ошибке toks[errIdx]: «не» обязано
// стоять в сегменте между словом об ошибке и «а» — в поправительном
// обороте оно отрицает первый объект общего сказуемого
// («ErrUninitializedTrie возникает не на слове, а на вызовы»).
// «Не» непосредственно после слова об ошибке снимает само
// утверждение («ErrUninitializedTrie не возникает, а сложность
// операций: …»), поэтому таким «а» перечисление не вводится.
// При нескольких противительных «а» двоеточие управляется ближайшим
// к нему «а», и сегмент этого «а» заканчивается на предыдущем
// противительном: «не» из более ранней поправки («…не при невалидном
// UTF-8, а на неинициализированном Trie, а сложность операций: …»)
// ближнему «а» не одалживается, и его отдельное утверждение
// перечисление ошибочных методов не вводит. Добавочные
// «а также»/«а иногда» и уточняющее «а именно» продолжают
// перечисление объектов того же утверждения (роль для
// colonIntroducesErrorList не меняется), но через них «не» тоже
// не одалживается: отрицание принадлежит сегменту до добавочного
// «а» («…не при невалидном UTF-8, а иногда на неинициализированном
// Trie, а сложность операций: …»), поэтому поиск «не» на нём
// заканчивается.
func negationBetweenErrorAndAdversative(toks []wordTok, errIdx, advIdx int) bool {
	for k := advIdx - 1; k > errIdx+1; k-- {
		if toks[k].word == "а" {
			// Сегмент ближнего «а» закончился: на предыдущем
			// противительном или добавочном/уточняющем «а»
			// («а также», «а иногда», «а именно») — «не»
			// из-за него ближнему «а» не одалживается.
			return false
		}
		if toks[k].word == "не" {
			return true
		}
	}
	return false
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
		// Противительное «а» отделяет от утверждения об ошибке
		// самостоятельную заметку о сложности: двоеточие после
		// «сложность операций» перечисление ошибочных методов
		// не вводит, и Len() к утверждению об ошибке не относится,
		// несмотря на вводящее существительное «операций».
		"ErrUninitializedTrie возник до проверки входа, а сложность операций: Len() при этом возвращает 0",
		// Отрицание снимает само утверждение об ошибке («не
		// возникает»), поэтому и после противительного «а» заметка
		// о сложности перечисление ошибочных методов не вводит —
		// как и без отрицания.
		"ErrUninitializedTrie не возникает, а сложность операций: Len() при этом возвращает 0",
		// Несколько противительных «а»: двоеточие управляется
		// ближайшим «а», начинающим отдельную заметку о сложности,
		// и «не» из более ранней поправки («не при невалидном
		// UTF-8, а на неинициализированном Trie») ему не
		// одалживается — заметка о сложности перечисление
		// ошибочных методов не вводит.
		"ErrUninitializedTrie возникает не при невалидном UTF-8, а на неинициализированном Trie, а сложность операций: Len() при этом возвращает 0",
		// Тот же перенос отрицания, когда первая поправка
		// продолжена добавочным «а иногда»: «не» из сегмента
		// «не …, а иногда …» ближнему противительному «а» через
		// добавочное «а» не одалживается, и заметка о сложности
		// перечисление ошибочных методов не вводит.
		"ErrUninitializedTrie возникает не при невалидном UTF-8, а иногда на неинициализированном Trie, а сложность операций: Len() при этом возвращает 0",
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
		// Противительное «а», продолжающее или уточняющее то же
		// утверждение об ошибке, двоеточию не мешает: добавочное
		// «а также», уточняющее «а именно» и поправительное
		// «не …, а …» ведут перечисление под тем же утверждением,
		// и Len() в нём не оговорён.
		"ErrUninitializedTrie возникает до проверки входа, а также на вызовы: Len(), Search и Delete возвращают ошибку",
		// Добавочное «а иногда» тоже продолжает то же утверждение
		// об ошибке: двоеточие после «вызовы» вводит перечисление
		// под ним, и Len() внутри перечисления не оговорён. Заметка
		// «Len() при этом возвращает 0» здесь — проверка различает:
		// сочти код «а иногда» противительным, двоеточие осталось бы
		// без управляющего утверждения, упоминание вышло бы из
		// перечисления, и позднейшая оговорка сделала бы окно
		// безопасным — тест распознал бы поломку.
		"ErrUninitializedTrie возникает при пустом слове, а иногда на вызовы: Insert, Search, Len() и Delete; Len() при этом возвращает 0",
		// Добавочное «а иногда» продолжает то же управляемое
		// утверждение об ошибке и после поправки «не …»: отрицание
		// в сегменте поправки («не при невалидном UTF-8») новую
		// поправку не начинает, двоеточие после «вызовы» остаётся
		// под тем же утверждением об ошибке, и Len() внутри
		// перечисления не оговорён.
		"ErrUninitializedTrie возникает не при невалидном UTF-8, а иногда на вызовы: Insert, Search, Len() и Delete; Len() при этом возвращает 0",
		"ErrUninitializedTrie возникает не на слове, а именно на вызовы: Len(), Search и Delete возвращают ошибку",
		"ErrUninitializedTrie возникает не на слове, а на вызовы: Len(), Search и Delete возвращают ошибку",
		// Уточняющий оборот «а именно:» после вводящего
		// существительного продолжает то же перечисление под
		// утверждением об ошибке: Len() — его первый элемент, и
		// позднейшая верная заметка про Len() это не спасает.
		"ErrUninitializedTrie возникает на вызовы, а именно: Len(), Search и Delete возвращают ошибку, но Len() при этом возвращает 0",
		// Len() внутри перечисления «а именно:» — собственной
		// оговорки (ноль, отрицание, исключение) у упоминания нет.
		"ErrUninitializedTrie возникает на вызовы, а именно: Insert, Search, Len() и Delete возвращают ошибку",
		// Поправительный оборот «не …, а …» с одним противительным
		// «а»: двоеточие управляется именно им, и «не» в его
		// сегменте — часть поправки («не при пустом Trie, а на
		// вызовы»), а не снятие утверждения, поэтому перечисление
		// остаётся под тем же утверждением об ошибке, и Len() в нём
		// не оговорён. Заметка «Len() при этом возвращает 0» здесь —
		// проверка различает: сочти код поправительное «а» началом
		// отдельного утверждения, двоеточие осталось бы без
		// управляющего утверждения, упоминание вышло бы из
		// перечисления, и позднейшая оговорка сделала бы окно
		// безопасным — тест распознал бы поломку.
		"ErrUninitializedTrie возникает не при пустом Trie, а на вызовы: Insert, Search, Len() и Delete; Len() при этом возвращает 0",
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
		// Совместное подлежащее через предлог совместности:
		// «Delete с Len()»/«Delete со Len()» утверждает ошибку и
		// для Len(), exemption не действует.
		"Len() возвращает 0, а ошибку возвращает Delete с Len()",
		"Len() возвращает 0, а ошибку возвращает Delete со Len()",
		// Предлог совместности «с»/«со» в перечислении ошибочных
		// методов: «Delete с Len()»/«Delete со Len()» включают Len()
		// в возвращающие ошибку методы, как и союз «и».
		"ErrUninitializedTrie возвращают Insert, Search и Delete с Len()",
		"ErrUninitializedTrie возвращают Insert, Search и Delete со Len()",
		// Ассоциация перечисления с ошибкой распространяется на все
		// элементы, в том числе через вводящее существительное
		// «методов»: Len() внутри перечисления, и позднейшая верная
		// заметка про Len() это не спасает.
		"ErrUninitializedTrie до проверки входа на вызовы методов: Insert, Search, Len(), Delete; Len() при этом возвращает 0",
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

// Тест порядка модулей Блока 2 (тикет 44): модуль kd-dom-ast присутствует
// в block-2 и идёт после trie, замыкая цепочку
// tree-traversals → bst → bst-balancing → b-trees → trie → kd-dom-ast.
func TestEmbeddedBankBlock2KdDomAstAfterTrie(t *testing.T) {
	block2 := embeddedBlock(t, "block-2")
	assertModulesInOrder(t, block2.Modules,
		"tree-traversals", "bst", "bst-balancing", "b-trees", "trie", "kd-dom-ast")
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

// Регрессия тикета 44: два семантически нагруженных вопроса модуля
// kd-dom-ast. Вопрос про сложность поиска ближайшего соседа не вправе
// обещать гарантированную O(log n): отсечение зависит от данных и формы
// дерева, и в худшем случае осматриваются все точки. Вопрос про общее у
// трёх структур обязан закреплять границу аналогии — общее только
// древовидное представление, а не семантика или тип ветвления. Проверки
// идут по коротким маркерам у публичного шва банка, а не повторяют
// формулировки целиком; маркеры уникальны в модуле, чтобы похожий новый
// вопрос не нарушал проверку.
func TestEmbeddedBankKdDomAstQuestionSemantics(t *testing.T) {
	questions := embeddedModule(t, "block-2", "kd-dom-ast").Questions

	// Вопрос про отсечение: правильный ответ отрицает гарантию и
	// называет худший случай, а пояснение проговаривает и то, что
	// отсечение может помочь, и то, что гарантированной сложности оно
	// не даёт. Отсутствие гарантии проверяется по связи отрицания со
	// словом гарантии в предложении про сложность, а не по точной
	// фразе: переформулировка урока с другим порядком слов не должна
	// ломать проверку, а отрицание при другом слове — не считается.
	// Распознавание формулировок — в negatesComplexityGuarantee и его
	// табличном тесте.
	q := questionByMarker(t, "kd-dom-ast", questions, "Гарантирует")
	correct := q.Options[q.Correct]
	if !containsAny(correct, "Нет") {
		t.Errorf("правильный ответ должен отрицать гарантию O(log n): %q", correct)
	}
	if !containsAny(correct, "худш") || !containsAny(correct, "n точек") {
		t.Errorf("правильный ответ должен называть худший случай со всеми точками: %q", correct)
	}
	if !containsAny(q.Explanation, "может помочь") {
		t.Errorf("пояснение должно проговаривать, что отсечение может помочь: %q", q.Explanation)
	}
	if !negatesComplexityGuarantee(q.Explanation) {
		t.Errorf("пояснение должно отрицать гарантированную сложность, а не просто содержать «не»: %q", q.Explanation)
	}

	// Вопрос про границу аналогии: правильный ответ сводит общее трёх
	// структур к древовидной форме и отмечает различие семантики и
	// ветвления; уравнивать структуры через BST-семантику нельзя.
	q = questionByMarker(t, "kd-dom-ast", questions, "граница аналогии")
	correct = q.Options[q.Correct]
	if !containsAny(correct, "древовидн") {
		t.Errorf("правильный ответ должен сводить общее к древовидной форме: %q", correct)
	}
	if !containsAny(correct, "семантик") || !containsAny(correct, "ветвлен") {
		t.Errorf("правильный ответ должен отмечать различие семантики и типа ветвления: %q", correct)
	}
	if !containsAny(q.Explanation, "не переносит") {
		t.Errorf("пояснение должно оговаривать, что аналогия не переносит свойства структур: %q", q.Explanation)
	}
}

// Регрессия тикета 44 (ревью): вопросы KD о разбиении области обязаны
// приписывать деление внутреннему узлу, а не любому узлу, и отличать
// лист, который область не делит, — у узла без потомков нет
// разделяющей плоскости. Проверки идут по коротким маркерам у
// публичного шва банка; маркеры уникальны в модуле, чтобы похожий
// новый вопрос не нарушал проверку.
func TestEmbeddedBankKdDomAstPartitionSemantics(t *testing.T) {
	questions := embeddedModule(t, "block-2", "kd-dom-ast").Questions

	// Вопрос про хранение и спуск: правильный ответ и пояснение
	// приписывают деление области внутреннему узлу, а лист отличают
	// — он область не делит.
	q := questionByMarker(t, "kd-dom-ast", questions, "шаге спуска")
	correct := q.Options[q.Correct]
	// Деление названо утвердительно и связано с внутренним узлом: в
	// правильном ответе стоит точная связка «внутренний узел делит
	// область». Отрицательные и переадресованные варианты связку не
	// содержат: «внутренний узел не делит область» (отрицание при
	// сказуемом разрывает связку), «не внутренний узел делит
	// область» (отрицание при подлежащем) и «лист делит область»
	// (деление приписано листу).
	if !containsAny(correct, "внутренний узел делит область") ||
		containsAny(correct, "не внутренний узел", "лист делит") {
		t.Errorf("правильный ответ должен утвердительно связывать деление области с внутренним узлом: %q", correct)
	}
	if !containsAny(q.Explanation, "внутренн") {
		t.Errorf("пояснение должно приписывать деление области внутреннему узлу: %q", q.Explanation)
	}
	if !containsAny(q.Explanation, "Лист", "лист") || !containsAny(q.Explanation, "не делит") {
		t.Errorf("пояснение должно отличать лист, который область не делит: %q", q.Explanation)
	}

	// Вопрос про выбор координаты: деление области уже в тексте
	// вопроса приписано внутреннему узлу, правильный ответ называет
	// стратегию реализации, а пояснение привязывает пример стратегии
	// к внутренним узлам.
	q = questionByMarker(t, "kd-dom-ast", questions, "выбирается координата")
	if !containsAny(q.Text, "внутренн") || !containsAny(q.Text, "делит") {
		t.Errorf("вопрос должен спрашивать про деление области внутренним узлом: %q", q.Text)
	}
	correct = q.Options[q.Correct]
	if !containsAny(correct, "стратеги") {
		t.Errorf("правильный ответ должен называть стратегию реализации: %q", correct)
	}
	// Деление названо утвердительно и привязано к внутренним узлам
	// одного уровня: в пояснении стоит точная связка «внутренние
	// узлы одного уровня делят». Отрицательные варианты связку не
	// содержат: «внутренние узлы не всегда делят» (отрицание при
	// сказуемом разрывает связку) и «не внутренние узлы … делят»
	// (отрицание при подлежащем).
	if !containsAny(q.Explanation, "внутренние узлы одного уровня делят") ||
		containsAny(q.Explanation, "не внутренние узлы") {
		t.Errorf("пояснение должно привязывать деление по осям к внутренним узлам одного уровня: %q", q.Explanation)
	}
}

// Таблица регрессии проверки отрицания гарантии сложности (тикет 44,
// ревью): отрицание должно быть связано со словом гарантии или
// обещания, а не просто стоять рядом с упоминаниями гарантии и
// сложности. Распознаются формулировки урока и обзора — в том числе
// оборот «не даёт» (с усилением «никакой», в множественном числе «не
// дают», в инверсии и за обстоятельством-оговоркой «в общем случае»),
// страдательная форма «не даётся» (с усилением «никакая» и в прямом
// порядке «подлежащее — сказуемое» при тире) и связка «не является» —
// в своей клаузе, а несвязанное
// отрицание («гарантия сложности не важна»), действительное «не даёт»
// с другим дополнением («гарантия сложности не даёт выигрыша», в том
// числе с оговоркой перед дополнением), отрицание за границей клаузы —
// точкой с запятой, двоеточием или их последовательностью в любом
// порядке («; —», «— ;», «: —», «— :»: тире смягчает только одиночное
// тире), тире не вплотную к сказуемому
// («Гарантия сложности — это не даётся бесплатно»), —
// голое «не» в другом предложении и утвердительная гарантия — нет.
func TestNegatesComplexityGuarantee(t *testing.T) {
	negates := []string{
		// Формулировка из банка: отрицание примыкает к слову
		// гарантии, сложность названа в том же предложении.
		"Отсечение может сильно сократить поиск, но это не гарантия сложности: польза зависит от распределения данных",
		// Формулировка из обзора (docs/algorithms/kd-dom-ast.md):
		// отрицание стоит при глаголе «дать», а обещание названо
		// до него в той же клаузе.
		"Но гарантированной логарифмической сложности оно не даёт: польза отсечения зависит от данных и формы дерева",
		// Обещание стоит сразу за отрицанным глаголом «дать».
		"Отсечение не даёт гарантии сложности: польза зависит от распределения данных",
		// Усиление между отрицанным глаголом и словом гарантии.
		"Отсечение не даёт никакой гарантии сложности: польза зависит от данных и формы дерева",
		"в худшем случае KD-деревья не дают никакой гарантии логарифмической сложности",
		// Усиленное обещание за обстоятельством-оговоркой после
		// глагола: оговорка слот дополнения не заполняет.
		"Отсечение не даёт в общем случае никакой гарантии сложности",
		// Множественная форма действительного глагола: обещание —
		// дополнение после глагола.
		"KD-деревья не дают гарантии логарифмической сложности",
		// Страдательная форма: названное в клаузе слово гарантии —
		// подлежащее, и отрицание снимает его.
		"гарантия сложности не даётся: отсечение зависит от данных и формы дерева",
		// Страдательная форма с согласованным усилением после
		// глагола.
		"в худшем случае не даётся никакая гарантия сложности",
		// Отрицание примыкает к слову обещания; гарантия и
		// сложность стоят перед ним в обратном порядке.
		"поэтому гарантированной логарифмической сложности не обещается",
		// Инверсия без дополнения: клауза кончается за
		// обстоятельством-оговоркой.
		"гарантированной сложности поиск не даёт в общем случае",
		// Отрицание примыкает к слову гарантии после подлежащего.
		"сложность поиска не гарантирована",
		// Отрицание примыкает к слову гарантии без слова
		// «сложность»: оценка O(log n) сама говорит о сложности.
		"Отсечение не гарантирует поиск за O(log n)",
		// Отрицание стоит при связке перед словом гарантии.
		"Отсечение не является гарантией логарифмической сложности",
		// Экзистенциальное «нет» в прямом порядке, через усиление
		// и в инверсии.
		"у отсечения нет гарантии сложности",
		"в худшем случае у отсечения нет никакой гарантии логарифмической сложности",
		"гарантии сложности нет",
		"гарантия сложности — нет",
		// Тире допустимо лишь в локальном порядке «подлежащее —
		// сказуемое»: «Гарантия сложности — не даётся» — это
		// прямой порядок слов (не инверсия), и сказуемое
		// закрывает клаузу, открытую тире.
		"Гарантия сложности — не даётся",
		"гарантия сложности — не даётся: отсечение зависит от данных и формы дерева",
		// Связь отрицания со словом гарантии ищется в своём
		// предложении: голое «не» в соседнем предложении роли не
		// играет.
		"Поиск не линеен. Отсечение не гарантирует логарифмической сложности",
	}
	affirms := []string{
		// Действительное «не даёт» с другим дополнением: гарантия —
		// подлежащее, отрицание относится к выигрышу, а не снимает
		// гарантию. Прежняя проверка по произвольному слову гарантии
		// в клаузе ошибочно пропускала бы такую формулировку.
		"Гарантия сложности не даёт выигрыша: польза отсечения зависит от данных",
		// Оговорка «в общем случае» между глаголом и дополнением
		// слот дополнения не освобождает: дополнение названо после
		// неё.
		"Гарантия сложности не даёт в общем случае выигрыша",
		// «Не» после сильной границы начинает новую клаузу: слово
		// гарантии из предыдущей клаузы отрицанию не подчиняется,
		// и страдательная форма «не даётся» обещания в своей
		// клаузе не снимает.
		"Цикл по осям даёт гарантию равномерности сложности; не даётся это бесплатно",
		"KD-деревья не дают выигрыша по сложности, отсечение лишь помогает",
		// Несвязанное отрицание: «не» отрицает оценку важности, а не
		// гарантию, — прежняя проверка по близости слов ошибочно
		// пропускала бы такую формулировку.
		"Гарантия сложности не важна — важна практика",
		// Отрицание при наречии меры: гарантию оно не снимает.
		"Отсечение не всегда сокращает поиск, но гарантия сложности при этом есть",
		// Голое «не» в предложении без гарантии сложности.
		"Поиск в KD-дереве не линеен, отсечение лишь помогает",
		// Сложность названа, отрицание — тоже, но не при слове
		// гарантии: «сложность не важна» гарантию не отрицает.
		"Сложность поиска не важна, гарантию сложности отсечение даёт",
		// Утвердительная гарантия без отрицания.
		"Отсечение гарантирует логарифмическую сложность",
		"Отсечение даёт гарантированную O(log n)",
		// Страдательная форма без отрицания: гарантия названа, но
		// ничего не снимается.
		"гарантия сложности даётся поиском с отсечением: оно лишь помогает",
		// Обещание стоит за границей клаузы: «не даёт» относится к
		// другому дополнению, а не к гарантии.
		"Отсечение не даёт отсечения; гарантия сложности при этом есть",
		// Гарантия и сложность в одной клаузе, «нет» — за точкой с
		// запятой в следующей: инверсия через границу клаузы не
		// засчитывается, гарантия из предыдущей клаузы с «нет» не
		// связана.
		"у отсечения гарантии; сложности нет",
		// Усиление не переносит «нет» на слово гарантии из другой
		// клаузы.
		"нет никакой пользы — гарантия сложности есть",
		// Тире не смягчает границу для произвольных слов: сказуемое
		// «не даётся» с продолжением клаузы («бесплатно») её не
		// закрывает, и связка с подлежащим через тире не доказана.
		"Гарантия сложности — не даётся бесплатно",
		// Гарантия из предыдущей клаузы: тире перед сказуемым с
		// продолжением («это бесплатно») связку не восстанавливает.
		"Отсечение даёт гарантию равномерности сложности — не даётся это бесплатно",
		// Тире отделяет сказуемое только вплотную: между тире и
		// сказуемым другие слова не допускаются, и связка с
		// подлежащим через тире не доказана.
		"Гарантия сложности — это не даётся",
		"Гарантия сложности — это не даётся бесплатно",
		// Инверсия с «нет» требует того же: тире должно открывать
		// клаузу со сказуемым, а не со словом-связкой.
		"гарантия сложности — это нет",
		// Двоеточие — сильная граница: гарантия из клаузы перед
		// двоеточием отрицанию после него не подчиняется.
		"Цикл по осям даёт гарантию равномерности сложности: не даётся это бесплатно",
		// Последовательность разделителей остаётся жёсткой границей
		// в любом порядке: тире смягчает только одиночное тире,
		// поэтому «— ;» и «— :» жёсткие, как и «; —», «: —».
		"Цикл по осям даёт гарантию равномерности сложности; — не даётся это бесплатно",
		"гарантия сложности — ; не даётся",
		"гарантия сложности — : не даётся",
		"гарантия сложности: — не даётся",
		"гарантии сложности — ; нет",
		// Повторное тире — тоже жёсткая граница: мягким остаётся
		// только одиночное тире, поэтому удвоенное тире «— —»
		// сказуемое с подлежащим через тире не связывает.
		"Гарантия сложности — — не даётся",
		// Двоеточие между гарантией и «нет» разрывает инверсию.
		"у отсечения гарантии: сложности нет",
		// Тире между гарантией и её зависимой сложностью — не
		// порядок «подлежащее — сказуемое»: сказуемое «нет» не
		// открывает клаузу сразу за тире, и инверсия не
		// засчитывается.
		"у отсечения гарантии — сложности нет",
	}
	for _, text := range negates {
		if !negatesComplexityGuarantee(text) {
			t.Errorf("отрицание гарантии сложности не распознано: %q", text)
		}
	}
	for _, text := range affirms {
		if negatesComplexityGuarantee(text) {
			t.Errorf("формулировка без отрицания гарантии распознана как отрицание: %q", text)
		}
	}
}
