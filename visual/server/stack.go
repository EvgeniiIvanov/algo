package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// Ограничения входа Run стека: ёмкость — чтобы ряд ячеек оставался обозримым
// и не выделять гигантские слайсы, длина сценария — чтобы поток кадров
// был разумного размера.
const (
	maxStackCapacity = 100
	maxStackOps      = 500
)

// handleStack — Run сценария операций над стеком фиксированной ёмкости:
// на выходе SSE-поток кадров до конца сценария. Тонкий обработчик
// по шаблону handleBinarySearch: парсинг → валидация → runStreaming.
func handleStack(w http.ResponseWriter, r *http.Request) {
	capacity, err := parseStackCapacity(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ops, err := parseStackScript(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkStackScript(capacity, ops); err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Ёмкость валидирована в parseStackCapacity (1..maxStackCapacity):
		// ошибка конструктора здесь недостижима.
		_ = render.StackRun(capacity, ops, emit)
	})
}

// parseStackCapacity разбирает параметр capacity — ёмкость стека.
func parseStackCapacity(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("capacity")
	if strings.TrimSpace(raw) == "" {
		return 0, fmt.Errorf("параметр capacity обязателен: ёмкость стека, например capacity=5")
	}
	capacity, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("не удалось разобрать %q как целое число: capacity — одно целое число", raw)
	}
	if capacity < 1 || capacity > maxStackCapacity {
		return 0, fmt.Errorf("ёмкость стека должна быть от 1 до %d, получено %d", maxStackCapacity, capacity)
	}
	return capacity, nil
}

// parseStackScript разбирает параметр script — сценарий операций, по одной
// на строку: «push N» или «pop». Пустые строки пропускаются.
func parseStackScript(r *http.Request) ([]render.StackOp, error) {
	q := r.URL.Query()
	if !q.Has("script") {
		return nil, fmt.Errorf("параметр script обязателен: операции по одной на строку, например «push 5», «pop»")
	}
	raw := q.Get("script")

	var ops []render.StackOp
	for i, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue // пустая строка — не операция
		}
		switch fields[0] {
		case "push":
			if len(fields) != 2 {
				return nil, fmt.Errorf("строка %d: push требует ровно один аргумент — целое число, например «push 5»", i+1)
			}
			v, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: push N — значение одним целым", i+1, fields[1])
			}
			ops = append(ops, render.StackOp{IsPush: true, Value: v})
		case "pop":
			if len(fields) != 1 {
				return nil, fmt.Errorf("строка %d: pop не принимает аргументов", i+1)
			}
			ops = append(ops, render.StackOp{})
		default:
			return nil, fmt.Errorf("строка %d: неизвестная операция %q: ожидается «push N» или «pop»", i+1, fields[0])
		}
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("сценарий пуст: нужна хотя бы одна операция (push N или pop)")
	}
	if len(ops) > maxStackOps {
		return nil, fmt.Errorf("сценарий слишком длинный: %d операций, максимум %d", len(ops), maxStackOps)
	}
	return ops, nil
}

// checkStackScript проверяет сценарий прогоном по длине стека:
// pop из пустого стека — ошибка входа; push в полный — легален
// (переполнение показывается кадром, а не отказом Run).
func checkStackScript(capacity int, ops []render.StackOp) error {
	length := 0
	for i, op := range ops {
		if op.IsPush {
			if length < capacity {
				length++
			}
			continue
		}
		if length == 0 {
			return fmt.Errorf("операция №%d: pop из пустого стека — нечего снимать", i+1)
		}
		length--
	}
	return nil
}
