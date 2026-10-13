package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// Ограничения входа Run списка: длина сценария — чтобы поток кадров был
// разумного размера; максимальная длина списка — чтобы ряд ячеек оставался
// обозримым (список растущий, но слишком длинный ряд не читается).
const (
	maxListOps    = 500
	maxListLength = 100
)

// handleList — Run сценария операций над списком на слайсе:
// на выходе SSE-поток кадров до конца сценария. Тонкий обработчик
// по шаблону handleQueue: парсинг → валидация → runStreaming.
func handleList(w http.ResponseWriter, r *http.Request) {
	ops, err := parseListScript(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkListScript(ops); err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Сценарий валидирован до Run: ListRun ошибки входа не даёт.
		_ = render.ListRun(ops, emit)
	})
}

// parseListScript разбирает параметр script — сценарий операций, по одной
// на строку: «insert I V» (индекс 0..текущая длина включительно) или
// «remove I» (индекс 0..текущая длина-1). Пустые строки пропускаются.
func parseListScript(r *http.Request) ([]render.ListOp, error) {
	q := r.URL.Query()
	if !q.Has("script") {
		return nil, fmt.Errorf("параметр script обязателен: операции по одной на строку, например «insert 1 9», «remove 0»")
	}
	raw := q.Get("script")

	var ops []render.ListOp
	for i, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue // пустая строка — не операция
		}
		switch fields[0] {
		case "insert":
			if len(fields) != 3 {
				return nil, fmt.Errorf("строка %d: insert требует ровно два аргумента — индекс и значение, например «insert 1 9»", i+1)
			}
			index, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: insert I V — индекс и значение целыми", i+1, fields[1])
			}
			v, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: insert I V — индекс и значение целыми", i+1, fields[2])
			}
			ops = append(ops, render.ListOp{IsInsert: true, Index: index, Value: v})
		case "remove":
			if len(fields) != 2 {
				return nil, fmt.Errorf("строка %d: remove требует ровно один аргумент — индекс, например «remove 0»", i+1)
			}
			index, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: remove I — индекс одним целым", i+1, fields[1])
			}
			ops = append(ops, render.ListOp{IsInsert: false, Index: index})
		default:
			return nil, fmt.Errorf("строка %d: неизвестная операция %q: ожидается «insert I V» или «remove I»", i+1, fields[0])
		}
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("сценарий пуст: нужна хотя бы одна операция (insert I V или remove I)")
	}
	if len(ops) > maxListOps {
		return nil, fmt.Errorf("сценарий слишком длинный: %d операций, максимум %d", len(ops), maxListOps)
	}
	return ops, nil
}

// checkListScript проверяет сценарий прогоном по текущей длине списка:
// вставка допускает индекс 0..длина включительно, удаление — 0..длина-1;
// удалить из пустого списка или по индексу вне диапазона — ошибка входа.
func checkListScript(ops []render.ListOp) error {
	length := 0
	for i, op := range ops {
		if op.IsInsert {
			if op.Index < 0 || op.Index > length {
				return fmt.Errorf("операция №%d: insert %d в список длины %d — индекс вне диапазона 0..%d",
					i+1, op.Index, length, length)
			}
			if length+1 > maxListLength {
				return fmt.Errorf("операция №%d: список вырастет до %d значений — максимум %d; сократите сценарий",
					i+1, length+1, maxListLength)
			}
			length++
			continue
		}
		if length == 0 {
			return fmt.Errorf("операция №%d: remove из пустого списка — нечего удалять", i+1)
		}
		if op.Index < 0 || op.Index >= length {
			return fmt.Errorf("операция №%d: remove %d в списке длины %d — индекс вне диапазона 0..%d",
				i+1, op.Index, length, length-1)
		}
		length--
	}
	return nil
}
