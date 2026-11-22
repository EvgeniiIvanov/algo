// Обработчик Run сценария Trie: вход — URL-кодированный параметр script,
// по одной команде `insert W`, `search W`, `prefix W` или `delete W` на
// строку. Тонкий обработчик по шаблону handleBST: парсинг → валидация →
// runStreaming. Синтаксис, валидный UTF-8, лимиты команд, длин rune и живых
// rune-узлов проверяются до начала SSE: некорректный сценарий даёт ошибку
// (400) без частично начатого потока.
package server

import (
	"fmt"
	"net/http"
	"strings"
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

// maxTrieScriptRaw — верхняя граница размера параметра script: худший
// легальный сценарий — 500 команд по ~100 рун до 4 байт UTF-8 каждая
// (≈ 204 КБ с обвязкой команды); граница отсекает чрезмерно крупные запросы
// до какого-либо разбора.
const maxTrieScriptRaw = 1 << 18

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
		// Сценарий валидирован до Run: TrieRun ошибки входа не даёт.
		_ = render.TrieRun(commands, emit)
	})
}

// parseTrieScript — разбирает и валидирует параметр script: пустые строки
// и строки только из пробелов игнорируются, каждая непустая строка — ровно
// «действие W» из четырёх допустимых действий; аргумент — один непустой
// Unicode-токен без пробелов из валидного UTF-8 длиной не более 100 rune
// (Unicode scalar values, не UTF-8 байты). Пустой сценарий и превышение
// лимита команд отвергаются.
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
	if len(raw) > maxTrieScriptRaw {
		return nil, fmt.Errorf("сценарий слишком длинный: разрешено не более %d символов параметра; сократите сценарий", maxTrieScriptRaw)
	}

	lines := strings.Split(raw, "\n")
	commands := make([]render.TrieCommand, 0, len(lines))
	for i, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue // пустые строки и строки из пробелов игнорируются
		}
		cmd, err := parseTrieLine(fields, i+1)
		if err != nil {
			return nil, err
		}
		// Лимит проверяется на лету: сценарий-перегрузка не копит команды.
		if len(commands) == maxTrieCommands {
			return nil, fmt.Errorf("сценарий слишком длинный: разрешено не более %d команд; первую лишнюю команду видим на строке %d", maxTrieCommands, i+1)
		}
		commands = append(commands, cmd)
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("сценарий пуст: пустые строки игнорируются, нужен хотя бы один непустой шаг (insert W, search W, prefix W или delete W)")
	}
	return commands, nil
}

// parseTrieLine — одна непустая строка сценария: «insert/search/prefix/
// delete W». Информативные ошибки ведут номер строки, демонстрирующий первое
// отклонение. Пустой аргумент сюда не доходит: strings.Fields рвёт строку
// по пробелам, так что аргумент — всегда непустой токен.
func parseTrieLine(fields []string, line int) (render.TrieCommand, error) {
	verb := fields[0]
	op, ok := trieOpsByVerb[verb]
	if !ok {
		return render.TrieCommand{}, fmt.Errorf("неизвестная команда %q на строке %d: ожидается insert, search, prefix или delete", verb, line)
	}
	if len(fields) == 1 {
		return render.TrieCommand{}, fmt.Errorf("команда %q на строке %d: не хватает аргумента — ожидается «%s W», например %s кот", verb, line, verb, verb)
	}
	if len(fields) > 2 {
		return render.TrieCommand{}, fmt.Errorf("команда на строке %d: лишние аргументы — ожидается действие и один непустой аргумент, например «%s кот»", line, verb)
	}
	word := fields[1]
	if !utf8.ValidString(word) {
		return render.TrieCommand{}, fmt.Errorf("аргумент команды на строке %d — невалидный UTF-8: аргумент — последовательность Unicode scalar values", line)
	}
	runes := []rune(word)
	if len(runes) > maxTrieRunes {
		return render.TrieCommand{}, fmt.Errorf("аргумент %q на строке %d длиннее %d rune (Unicode scalar values): длина %d",
			word, line, maxTrieRunes, len(runes))
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
