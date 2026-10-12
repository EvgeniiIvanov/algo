package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// Ограничения входа Run очереди: ёмкость — чтобы ряд ячеек оставался
// обозримым и не выделять гигантские слайсы, длина сценария — чтобы поток
// кадров был разумного размера.
const (
	maxQueueCapacity = 100
	maxQueueOps      = 500
)

// handleQueue — Run сценария операций над очередью фиксированной ёмкости:
// на выходе SSE-поток кадров до конца сценария. Тонкий обработчик
// по шаблону handleStack: парсинг → валидация → runStreaming.
func handleQueue(w http.ResponseWriter, r *http.Request) {
	capacity, err := parseQueueCapacity(r)
	if err != nil {
		writeError(w, err)
		return
	}
	ops, err := parseQueueScript(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkQueueScript(capacity, ops); err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Ёмкость валидирована в parseQueueCapacity (1..maxQueueCapacity):
		// ошибка конструктора здесь недостижима.
		_ = render.QueueRun(capacity, ops, emit)
	})
}

// parseQueueCapacity разбирает параметр capacity — ёмкость очереди.
func parseQueueCapacity(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("capacity")
	if strings.TrimSpace(raw) == "" {
		return 0, fmt.Errorf("параметр capacity обязателен: ёмкость очереди, например capacity=5")
	}
	capacity, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("не удалось разобрать %q как целое число: capacity — одно целое число", raw)
	}
	if capacity < 1 || capacity > maxQueueCapacity {
		return 0, fmt.Errorf("ёмкость очереди должна быть от 1 до %d, получено %d", maxQueueCapacity, capacity)
	}
	return capacity, nil
}

// parseQueueScript разбирает параметр script — сценарий операций, по одной
// на строку: «enqueue N» или «dequeue». Пустые строки пропускаются.
func parseQueueScript(r *http.Request) ([]render.QueueOp, error) {
	q := r.URL.Query()
	if !q.Has("script") {
		return nil, fmt.Errorf("параметр script обязателен: операции по одной на строку, например «enqueue 5», «dequeue»")
	}
	raw := q.Get("script")

	var ops []render.QueueOp
	for i, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue // пустая строка — не операция
		}
		switch fields[0] {
		case "enqueue":
			if len(fields) != 2 {
				return nil, fmt.Errorf("строка %d: enqueue требует ровно один аргумент — целое число, например «enqueue 5»", i+1)
			}
			v, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: enqueue N — значение одним целым", i+1, fields[1])
			}
			ops = append(ops, render.QueueOp{IsEnqueue: true, Value: v})
		case "dequeue":
			if len(fields) != 1 {
				return nil, fmt.Errorf("строка %d: dequeue не принимает аргументов", i+1)
			}
			ops = append(ops, render.QueueOp{})
		default:
			return nil, fmt.Errorf("строка %d: неизвестная операция %q: ожидается «enqueue N» или «dequeue»", i+1, fields[0])
		}
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("сценарий пуст: нужна хотя бы одна операция (enqueue N или dequeue)")
	}
	if len(ops) > maxQueueOps {
		return nil, fmt.Errorf("сценарий слишком длинный: %d операций, максимум %d", len(ops), maxQueueOps)
	}
	return ops, nil
}

// checkQueueScript проверяет сценарий прогоном по длине очереди:
// dequeue из пустой очереди — ошибка входа; enqueue в полную — легален
// (переполнение показывается кадром, а не отказом Run).
func checkQueueScript(capacity int, ops []render.QueueOp) error {
	length := 0
	for i, op := range ops {
		if op.IsEnqueue {
			if length < capacity {
				length++
			}
			continue
		}
		if length == 0 {
			return fmt.Errorf("операция №%d: dequeue из пустой очереди — нечего снимать", i+1)
		}
		length--
	}
	return nil
}
