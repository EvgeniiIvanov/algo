// Обработчик Run сценария BST: вход — URL-кодированный параметр script,
// по одной команде `insert N`, `search N` или `delete N` на строку.
// Тонкий обработчик по шаблону handleTreeTraversal: парсинг → валидация →
// runStreaming. Синтаксис, диапазон целых, лимиты команд и живых узлов
// проверяются до начала SSE: некорректный сценарий даёт ошибку (400)
// без частично начатого потока.
package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// maxBSTCommands — лимит команд сценария визуального Run: поток кадров
// остаётся разумного размера. Лимит только для визуализации: библиотечные
// операции BST им не ограничены.
const maxBSTCommands = 500

// maxBSTScriptRaw — верхняя граница размера параметра script: 500 команд
// по ~25 символов каждая укладываются с запасом; граница отсекает чрезмерно
// крупные запросы до какого-либо разбора.
const maxBSTScriptRaw = 1 << 16

// maxBSTSafeInteger — граница безопасного целого JavaScript (2^53−1):
// значения ключей сценария в диапазоне [−2^53+1; 2^53−1] — фронтенд
// зеркально проверяет тот же диапазон (SPEC).
const maxBSTSafeInteger = 1<<53 - 1

// bstOpsByVerb — допустимые действия сценария. Значения — те же строки
// команд, что общие для рендерера и фронтенда (render.BSTOp*).
var bstOpsByVerb = map[string]string{
	render.BSTOpInsert: render.BSTOpInsert,
	render.BSTOpSearch: render.BSTOpSearch,
	render.BSTOpDelete: render.BSTOpDelete,
}

// handleBST — Run сценария BST: на выходе SSE-поток кадров kind="tree"
// (исходный кадр + кадр на каждое событие алгоритма).
func handleBST(w http.ResponseWriter, r *http.Request) {
	commands, err := parseBSTScript(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkBSTNodeLimit(commands); err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		render.BSTRun(commands, emit)
	})
}

// parseBSTScript — разбирает и валидирует параметр script: пустые строки
// игнорируются, каждая непустая строка — ровно «действие целое» из трёх
// допустимых действий; ключ — десятичное целое в безопасном диапазоне
// [−2^53+1; 2^53−1]. Пустой сценарий и превышение лимита команд отвергаются.
func parseBSTScript(r *http.Request) ([]render.BSTCommand, error) {
	script, present := r.URL.Query()["script"]
	if !present {
		return nil, fmt.Errorf("параметр script обязателен: по одной команде на строку — insert N, search N или delete N, например insert 5")
	}
	if len(script) != 1 {
		return nil, fmt.Errorf("параметр script задан %d раз: ожидается один параметр script", len(script))
	}
	raw := script[0]
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("сценарий пуст: пустые строки игнорируются, нужен хотя бы один непустой шаг (insert N, search N или delete N)")
	}
	if len(raw) > maxBSTScriptRaw {
		return nil, fmt.Errorf("сценарий слишком длинный: разрешено не более %d символов параметра, лимит — %d команд; сократите сценарий", maxBSTScriptRaw, maxBSTCommands)
	}

	lines := strings.Split(raw, "\n")
	commands := make([]render.BSTCommand, 0, len(lines))
	for i, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 0 {
			continue // пустые строки игнорируются
		}
		cmd, err := parseBSTLine(fields, i+1)
		if err != nil {
			return nil, err
		}
		// Лимит проверяется на лету: сценарий-перегрузка не копит команды.
		if len(commands) == maxBSTCommands {
			return nil, fmt.Errorf("сценарий слишком длинный: разрешено не более %d команд; первую лишнюю команду видим на строке %d", maxBSTCommands, i+1)
		}
		commands = append(commands, cmd)
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("сценарий пуст: пустые строки игнорируются, нужен хотя бы один непустой шаг (insert N, search N или delete N)")
	}
	return commands, nil
}

// parseBSTLine — одна непустая строка сценария: «insert/search/delete N».
// Информативные ошибки ведут номер строки, демонстрирующий первое отклонение.
func parseBSTLine(fields []string, line int) (render.BSTCommand, error) {
	verb := fields[0]
	op, ok := bstOpsByVerb[verb]
	if !ok {
		return render.BSTCommand{}, fmt.Errorf("неизвестная команда %q на строке %d: ожидается insert, search или delete", verb, line)
	}
	if len(fields) == 1 {
		return render.BSTCommand{}, fmt.Errorf("команда %q на строке %d: не хватает целого числа — ожидается «%s N», например %s 5", verb, line, verb, verb)
	}
	if len(fields) > 2 {
		return render.BSTCommand{}, fmt.Errorf("команда на строке %d: лишние аргументы — ожидается действие и одно целое число, например «%s 5»", line, verb)
	}
	key, err := bstScriptKey(fields[1], line)
	if err != nil {
		return render.BSTCommand{}, err
	}
	return render.BSTCommand{Op: op, Key: key}, nil
}

// bstScriptKey — проверка ключа команды: десятичное целое, помещающееся
// в int64 и в безопасный диапазон JavaScript. Порядок диагностик: сначала
// грамматика/переполнение int64 (strconv), затем безопасный диапазон
// (значения в (−2^53, 2^53) как int64 корректны, а для фронтенда опасны).
func bstScriptKey(number string, line int) (int, error) {
	key, err := strconv.ParseInt(number, 10, 64)
	if err != nil {
		if errors.Is(err, strconv.ErrRange) {
			return 0, fmt.Errorf("значение %s на строке %d за пределами целого (int64): ключ — десятичное целое", number, line)
		}
		return 0, fmt.Errorf("не удалось разобрать %q как целое число на строке %d: ключ — десятичное целое", number, line)
	}
	if key > maxBSTSafeInteger || key < -maxBSTSafeInteger {
		return 0, fmt.Errorf("значение %d на строке %d вне безопасного диапазона: допустимо от −(2^53−1) до 2^53−1", key, line)
	}
	return int(key), nil
}

// checkBSTNodeLimit — холостой прогон сценария тихими операциями
// (ADR-0001: семантика одна): считаем живые узлы после каждой команды и
// отвергаем превышение лимита визуализации до начала SSE. Промахи и
// дубликаты протекают как легальные шаги — на счётчик они не дают иного
// эффекта, чем дают в Run.
func checkBSTNodeLimit(commands []render.BSTCommand) error {
	var root *trees.Node
	for i, cmd := range commands {
		switch cmd.Op {
		case render.BSTOpInsert:
			root, _ = trees.Insert(root, cmd.Key)
		case render.BSTOpDelete:
			root, _ = trees.Delete(root, cmd.Key)
		}
		if n := countTreeNodes(root); n > maxTreeNodes {
			return fmt.Errorf("сценарий превышает лимит живых узлов: разрешено не более %d, а команда №%d (%s %d) даёт %d узлов; сократите сценарий",
				maxTreeNodes, i+1, cmd.Op, cmd.Key, n)
		}
	}
	return nil
}

// countTreeNodes — число узлов дерева (рекурсивно), для проверки лимита.
func countTreeNodes(n *trees.Node) int {
	if n == nil {
		return 0
	}
	return 1 + countTreeNodes(n.Left) + countTreeNodes(n.Right)
}
