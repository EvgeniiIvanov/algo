// Обработчик Run обхода бинарного дерева: вход — вложенный JSON дерева
// (`tree`) и порядок (`order`). Тонкий обработчик по шаблону handleQueue:
// парсинг → валидация → runStreaming. Невалидный JSON, дерево больше
// лимита и неизвестный порядок отвергаются до SSE (400 вместо кадров).
package server

import (
	"encoding/json"
	"fmt"
	"math"
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
// верхний null — пустое дерево. Валидирует значения (целые числа) и
// лимит узлов.
func parseTreeJSON(r *http.Request) (*trees.Node, error) {
	raw := r.URL.Query().Get("tree")
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("параметр tree обязателен: вложенный JSON дерева, например {\"value\":1,\"left\":{\"value\":2}}")
	}

	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
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
// value и опциональными потомками; null — это nil-узел. count накапливает
// число узлов.
func buildTree(parsed any, dest **trees.Node, count *int, context string) error {
	if parsed == nil {
		*dest = nil // верхний null или null-потомок — пустая ветвь
		return nil
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return fmt.Errorf("узел дерева%s должен быть объектом {value, left, right} или null", context)
	}

	value, err := treeValue(obj["value"])
	if err != nil {
		return fmt.Errorf("узел дерева%s: %v", context, err)
	}

	node := trees.NewNode(value)
	*count++
	if *count > maxTreeNodes {
		return fmt.Errorf("дерево слишком большое: больше %d узлов, максимум %d; сократите вход", *count, maxTreeNodes)
	}
	*dest = node
	if err := buildTree(obj["left"], &node.Left, count, ", левый потомок (путь "+path(context+"L")+")"); err != nil {
		return err
	}
	if err := buildTree(obj["right"], &node.Right, count, ", правый потомок (путь "+path(context+"R")+")"); err != nil {
		return err
	}
	return nil
}

// treeValue проверяет значение узла: число в JSON, целое.
func treeValue(v any) (int, error) {
	num, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("поле value обязательно — целое число")
	}
	if num != math.Trunc(num) || math.IsInf(num, 0) {
		return 0, fmt.Errorf("значение %g не целое: value — целое число", num)
	}
	if num > math.MaxInt64 || num < math.MinInt64 {
		return 0, fmt.Errorf("значение %g за пределами целого: value — целое число", num)
	}
	return int(num), nil
}

// path — путь узла в сообщении об ошибке: корень — «корень».
func path(p string) string {
	if p == "" {
		return "корень"
	}
	return p
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
