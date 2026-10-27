// Обработчик Run обхода бинарного дерева: вход — вложенный JSON дерева
// (`tree`) и порядок (`order`). Тонкий обработчик по шаблону handleQueue:
// парсинг → валидация → runStreaming. Невалидный JSON, дерево больше
// лимита и неизвестный порядок отвергаются до SSE (400 вместо кадров).
package server

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// maxTreeNodes — лимит визуального Run: дерево должно оставаться
// обсервимым, поток кадров — разумного размера. Лимит только для
// визуализации: библиотечные обходы им не ограничены.
const maxTreeNodes = 100

// handleTreeTraversal — Run обходов in/pre/post-order: на выходе
// SSE-поток кадров kind="tree" до конца обхода (исходный кадр + кадр
// на каждое посещение). Обход по уровням — отдельный маршрут (тикет 32).
func handleTreeTraversal(w http.ResponseWriter, r *http.Request) {
	root, err := parseTreeJSON(r)
	if err != nil {
		writeError(w, err)
		return
	}
	order, err := parseTreeOrder(r.URL.Query().Get("order"))
	if err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Вход валидирован до Run: порядок известен, ошибка конструктора
		// здесь недостижима.
		_ = render.TreeTraversalRun(root, order, emit)
	})
}

// treeOrders — множество валидных порядков для проверки входа.
var treeOrders = map[string]bool{
	render.TreeOrderInOrder:   true,
	render.TreeOrderPreOrder:  true,
	render.TreeOrderPostOrder: true,
}

// parseTreeJSON разбирает параметр tree — вложенный JSON дерева
// (value, left, right). Отсутствующий или null потомок — пустая ветвь,
// верхний null — пустое дерево. Numbers разбираются как json.Number
// (UseNumber): целое значение узла проверяется точно по литералу,
// без float64-округления. Валидирует значения и лимит узлов.
func parseTreeJSON(r *http.Request) (*trees.Node, error) {
	raw := r.URL.Query().Get("tree")
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("параметр tree обязателен: вложенный JSON дерева, например {\"value\":1,\"left\":{\"value\":2}}")
	}

	var parsed any
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber() // числа остаются литералами: точная проверка целых
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("не удалось разобрать JSON дерева: %v", err)
	}

	var root *trees.Node
	count := 0
	if err := buildTree(parsed, &root, &count, ""); err != nil {
		return nil, err
	}
	return root, nil
}

// buildTree — рекурсивная проверка parsed как узла дерева: map с целым
// value и опциональными потомками; null — это nil-узел. context — сегмент
// сообщения об ошибке с путём узла ("": корень, дальше "(путь L)" и т.д.).
// count накапливает число узлов.
func buildTree(parsed any, dest **trees.Node, count *int, context string) error {
	if parsed == nil {
		*dest = nil // верхний null или null-потомок — пустая ветвь
		return nil
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return fmt.Errorf("узел дерева%s должен быть объектом {value, left, right} или null", treeContext(context))
	}

	value, err := treeValue(obj["value"])
	if err != nil {
		return fmt.Errorf("значение узла дерева%s: %v", treeContext(context), err)
	}
	node := trees.NewNode(value)
	*count++
	if *count > maxTreeNodes {
		return fmt.Errorf("дерево слишком большое: разрешено не более %d узлов; сократите вход", maxTreeNodes)
	}
	*dest = node
	if err := buildTree(obj["left"], &node.Left, count, context+"L"); err != nil {
		return err
	}
	if err := buildTree(obj["right"], &node.Right, count, context+"R"); err != nil {
		return err
	}
	return nil
}

// treeValue проверяет значение узла: JSON-число, целое и представимое
// в int без округления. Дубликат проверки диапазона через float64 не
// подходит: 2^63 представимо как float64, но не как int64 — округление
// пропустило бы 9223372036854775808 с переполнением int.
func treeValue(v any) (int, error) {
	num, ok := v.(json.Number)
	if !ok {
		return 0, fmt.Errorf("поле value обязательно — целое число")
	}
	// Литерал JSON-числа точно как big.Float (любая запись: 1, 1.5,
	// 1e2): показывает, целое ли число и помещается ли оно в int64.
	bf, _, err := big.ParseFloat(num.String(), 10, 128, big.ToNearestEven)
	if err != nil {
		return 0, fmt.Errorf("значение %s не число: value — целое число", num.String())
	}
	if !bf.IsInt() {
		return 0, fmt.Errorf("значение %s не целое: value — целое число", num.String())
	}
	n, _ := bf.Int(nil)
	if !n.IsInt64() {
		return 0, fmt.Errorf("значение %s за пределами целого (int64): value — целое число", num.String())
	}
	return int(n.Int64()), nil
}

// treeContext — сегмент сообщения об ошибке: путь узла в живой форме,
// как у клиента: корень — «(корень)», дальше — «(путь L)», «(путь LR)».
func treeContext(p string) string {
	if p == "" {
		return "(корень)"
	}
	return "(путь " + p + ")"
}

// parseTreeOrder разбирает параметр order — один из трёх порядков обхода.
func parseTreeOrder(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("параметр order обязателен: порядок обхода — in-order, pre-order или post-order")
	}
	if !treeOrders[raw] {
		return "", fmt.Errorf("неизвестный порядок обхода %q: ожидается in-order, pre-order или post-order", raw)
	}
	return raw, nil
}
