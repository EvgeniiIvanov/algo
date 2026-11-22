// Обработчик Run сценария Trie: вход — URL-кодированный параметр script,
// по одной команде `insert W`, `search W`, `prefix W` или `delete W` на
// строку. Тонкий обработчик по шаблону handleBST: парсинг → валидация →
// runStreaming. Синтаксис, валидный UTF-8, лимиты команд, длин rune и живых
// rune-узлов проверяются до начала SSE: некорректный сценарий даёт ошибку
// (400) без частично начатого потока.
package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// Ограничения входа Run Trie — как у BST и хеш-таблицы: не более 500 команд,
// не более 100 rune (Unicode scalar values) в слове/префиксе и не более 100
// живых rune-узлов (корень не считается), чтобы дерево оставалось обозримым,
// а поток кадров — разумного размера. Лимит живых узлов применяется к
// пиковому размеру Trie — максимуму числа живых узлов по состоянию после
// каждой команды сценария. Лимиты только для визуализации: библиотечный
// trees.Trie им не ограничен.
const (
	maxTrieCommands = 500
	maxTrieRunes    = 100
	maxTrieNodes    = 100
)

// trieOpsByVerb — допустимые действия сценария. Значения — те же строки
// команд, что общие для рендерера и фронтенда (render.TrieOp*).
var trieOpsByVerb = map[string]string{
	render.TrieOpInsert: render.TrieOpInsert,
	render.TrieOpSearch: render.TrieOpSearch,
	render.TrieOpPrefix: render.TrieOpPrefix,
	render.TrieOpDelete: render.TrieOpDelete,
}

// handleTrie — Run сценария Trie: на выходе SSE-поток кадров kind="trie"
// (исходный кадр + кадр на каждое событие алгоритма).
func handleTrie(w http.ResponseWriter, r *http.Request) {
	commands, err := parseTrieScript(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkTrieNodeLimit(commands); err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Preflight-валидация маршрута уже выполнялась до старта SSE, так что
		// ошибка рендерера здесь — защитный инвариантный fallback: известными
		// входами недостижима. Поток уже начат — статус не поменять, скрыть
		// ошибку нельзя, она записывается в журнал.
		if err := render.TrieRun(commands, emit); err != nil {
			log.Printf("сценарий прошёл валидацию, но рендерер его отклонил: %v", err)
		}
	})
}

// parseTrieScript — разбирает и валидирует параметр script: пустые строки
// и строки только из пробелов игнорируются, каждая непустая строка — ровно
// «действие W» из четырёх допустимых действий; аргумент — один непустой
// Unicode-токен без пробелов из валидного UTF-8 длиной не более 100 rune
// (Unicode scalar values, не UTF-8 байты). Пустой сценарий и превышение
// лимита команд отвергаются. Отдельного лимита размера параметра нет:
// произвольный whitespace (разделители, игнорируемые строки) сценарий
// не ограничивает, перегрузка отсекается на лету лимитами команд и длин
// rune до начала SSE, а полный размер запроса ограничивает сам транспорт
// HTTP-сервера (MaxHeaderBytes и его конфигурация) — разбор параметра
// линейный.
func parseTrieScript(r *http.Request) ([]render.TrieCommand, error) {
	script, present := r.URL.Query()["script"]
	if !present {
		return nil, fmt.Errorf("параметр script обязателен: по одной команде на строку — insert W, search W, prefix W или delete W, например insert кот")
	}
	if len(script) != 1 {
		return nil, fmt.Errorf("параметр script задан %d раз: ожидается один параметр script", len(script))
	}
	raw := script[0]
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("сценарий пуст: пустые строки игнорируются, нужен хотя бы один непустой шаг (insert W, search W, prefix W или delete W)")
	}

	// Ёмкость ограничена лимитом команд: больше команда не накопится.
	// Строки разбираются итерацией без strings.Split: подстроки делят память
	// с параметром, так что аллокации не растут пропорционально числу
	// игнорируемых пустых строк.
	commands := make([]render.TrieCommand, 0, maxTrieCommands)
	lineNo := 0
	for rest := raw; rest != ""; {
		line := rest
		if i := strings.IndexByte(rest, '\n'); i >= 0 {
			line, rest = rest[:i], rest[i+1:]
		} else {
			rest = ""
		}
		lineNo++
		// До трёх полей без массива пропорционально числу токенов: строка
		// с сотнями тысяч токенов не порождает срез «поле на токен».
		verb, word, extra := trieSplit3(line)
		if verb == "" {
			continue // пустые строки и строки из пробелов игнорируются
		}
		cmd, err := parseTrieLine(verb, word, extra, lineNo)
		if err != nil {
			return nil, err
		}
		// Лимит проверяется на лету: сценарий-перегрузка не копит команды.
		if len(commands) == maxTrieCommands {
			return nil, fmt.Errorf("сценарий слишком длинный: разрешено не более %d команд; первую лишнюю команду видим на строке %d", maxTrieCommands, lineNo)
		}
		commands = append(commands, cmd)
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("сценарий пуст: пустые строки игнорируются, нужен хотя бы один непустой шаг (insert W, search W, prefix W или delete W)")
	}
	return commands, nil
}

// trieASCIISpace — ASCII whitespace по таблице strings.Fields ('\t', '\n',
// '\v', '\f', '\r', ' '): для байтов < utf8.RuneSelf проверка без декодирования.
var trieASCIISpace = [utf8.RuneSelf]bool{
	'\t': true, '\n': true, '\v': true, '\f': true, '\r': true, ' ': true,
}

// trieSpaceSize — whitespace ли байт на позиции i и сколько байт он занимает:
// ASCII — по таблице strings.Fields, прочее — по unicode.IsSpace после
// декодирования руны (в том числе CRLF и Unicode whitespace вроде NBSP).
// Невалидные байты полем не рвут — как в strings.Fields (RuneError не
// whitespace); их ловит utf8.ValidString на аргументе.
func trieSpaceSize(line string, i int) (space bool, size int) {
	c := line[i]
	if c < utf8.RuneSelf {
		return trieASCIISpace[c], 1
	}
	r, n := utf8.DecodeRuneInString(line[i:])
	return unicode.IsSpace(r), n
}

// trieSkipSpace — индекс первого не-whitespace байта с позиции i.
func trieSkipSpace(line string, i int) int {
	for i < len(line) {
		space, size := trieSpaceSize(line, i)
		if !space {
			return i
		}
		i += size
	}
	return i
}

// trieFieldEnd — индекс конца поля, начавшегося с позиции i (первый
// whitespace-байт или конец строки).
func trieFieldEnd(line string, i int) int {
	for i < len(line) {
		space, size := trieSpaceSize(line, i)
		if space {
			return i
		}
		i += size
	}
	return i
}

// trieSplit3 — распознаёт первые три whitespace-разделённых поля строки
// без массива пропорционально числу полей (в отличие от strings.Fields,
// строящего срез «поле на токен»): verb, аргумент и только факт лишних
// полей — сканирование останавливается на первом байте третьего поля.
// Семантика whitespace — как у strings.Fields (см. trieSpaceSize). Пустая
// строка или строка из пробелов даёт пустой verb.
func trieSplit3(line string) (verb, word string, extra bool) {
	i := trieSkipSpace(line, 0)
	if i == len(line) {
		return "", "", false
	}
	j := trieFieldEnd(line, i)
	verb = line[i:j]

	i = trieSkipSpace(line, j)
	if i == len(line) {
		return verb, "", false
	}
	j = trieFieldEnd(line, i)
	word = line[i:j]

	// Лишние поля: достаточно первого байта третьего поля, дальше не сканируем.
	return verb, word, trieSkipSpace(line, j) < len(line)
}

// parseTrieLine — одна непустая строка сценария, разобранная trieSplit3
// на действие, аргумент и факт лишних полей: «insert/search/prefix/delete W».
// Информативные ошибки ведут номер строки, демонстрирующий первое отклонение.
// Порядок диагностик: сначала грамматика (вид, отсутствие и лишние поля),
// затем валидный UTF-8 и длина аргумента.
func parseTrieLine(verb, word string, extra bool, line int) (render.TrieCommand, error) {
	op, ok := trieOpsByVerb[verb]
	if !ok {
		return render.TrieCommand{}, fmt.Errorf("неизвестная команда %q на строке %d: ожидается insert, search, prefix или delete", verb, line)
	}
	if word == "" {
		return render.TrieCommand{}, fmt.Errorf("команда %q на строке %d: не хватает аргумента — ожидается «%s W», например %s кот", verb, line, verb, verb)
	}
	if extra {
		return render.TrieCommand{}, fmt.Errorf("команда на строке %d: лишние аргументы — ожидается действие и один непустой аргумент, например «%s кот»", line, verb)
	}
	// Валидный UTF-8 проверяется до подсчёта рун.
	if !utf8.ValidString(word) {
		return render.TrieCommand{}, fmt.Errorf("аргумент команды на строке %d — невалидный UTF-8: аргумент — последовательность Unicode scalar values", line)
	}
	// Длина — одна проверка, две ветки: токен из валидного UTF-8 длиннее
	// 100×utf8.UTFMax байт заведомо длиннее 100 rune и отсекается сразу,
	// без подсчёта и без эха токена в диагностике — большой токен не должен
	// раздувать сообщение об ошибке. В пределах этой границы руны считает
	// utf8.RuneCountInString без аллокации; эхо в ошибке получает только
	// токен не длиннее 400 байт исходного UTF-8 — это максимум размера
	// исходного токена в ветке с %q, а не гарантированный размер
	// экранированного текста.
	if len(word) > maxTrieRunes*utf8.UTFMax {
		return render.TrieCommand{}, fmt.Errorf("аргумент команды на строке %d длиннее %d rune (Unicode scalar values): сократите аргумент", line, maxTrieRunes)
	}
	if n := utf8.RuneCountInString(word); n > maxTrieRunes {
		return render.TrieCommand{}, fmt.Errorf("аргумент %q на строке %d длиннее %d rune (Unicode scalar values): длина %d",
			word, line, maxTrieRunes, n)
	}
	return render.TrieCommand{Op: op, Word: word}, nil
}

// trieNodeCounter — счётчик живых rune-узлов (корень не считается) по
// событиям библиотеки: вставка создаёт ровно один узел на каждое событие
// ребра created, pruning срезает ровно один узел на каждое TriePrune.
// Промахи и повторные вставки топологии не меняют.
type trieNodeCounter struct {
	live int
}

func (c *trieNodeCounter) Emit(ev trees.TrieEvent) {
	switch e := ev.(type) {
	case trees.TrieVisit:
		if e.Edge == trees.EdgeCreated {
			c.live++
		}
	case trees.TriePrune:
		c.live--
	}
}

// checkTrieNodeLimit — холостой прогон сценария в режиме наблюдения
// (ADR-0001: семантика одна): считаем живые rune-узлы после каждой команды
// и отвергаем превышение лимита визуализации до начала SSE. Лимит применяется
// к пиковому размеру Trie — максимуму по состоянию после каждой команды,
// а не только к финальному; внутри команды максимум достигается в её конце,
// поэтому проверки «после команды» точны. Промахи и дубликаты протекают
// как легальные шаги — на счётчик они не дают иного эффекта, чем дают в Run.
func checkTrieNodeLimit(commands []render.TrieCommand) error {
	t := trees.NewTrie()
	c := &trieNodeCounter{}
	for i, cmd := range commands {
		var err error
		switch cmd.Op {
		case render.TrieOpInsert:
			_, err = t.InsertWithEmitter(cmd.Word, c)
		case render.TrieOpSearch:
			_, err = t.SearchWithEmitter(cmd.Word, c)
		case render.TrieOpPrefix:
			_, err = t.StartsWithWithEmitter(cmd.Word, c)
		case render.TrieOpDelete:
			_, err = t.DeleteWithEmitter(cmd.Word, c)
		default:
			// Недостижимо: parseTrieScript пропускает только известные виды.
			return fmt.Errorf("неизвестный вид операции %q", cmd.Op)
		}
		if err != nil {
			return fmt.Errorf("команда №%d (%s %s): %v", i+1, cmd.Op, cmd.Word, err)
		}
		if c.live > maxTrieNodes {
			return fmt.Errorf("сценарий превышает лимит живых rune-узлов: разрешено не более %d, а команда №%d (%s %s) даёт %d узлов; сократите сценарий",
				maxTrieNodes, i+1, cmd.Op, cmd.Word, c.live)
		}
	}
	return nil
}
