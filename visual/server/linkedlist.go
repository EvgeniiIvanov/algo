package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// Ограничения входа Run односвязного списка — как у списка на слайсе:
// длина сценария — чтобы поток кадров был разумного размера (кадр даёт
// и каждый визит обхода), максимальная длина списка — чтобы цепочка
// узлов оставалась обозримой. Это safety bound одного визуального Run:
// библиотечная structures.LinkedList растёт без ограничения.
const (
	maxLinkedOps    = 500
	maxLinkedLength = 100
)

// handleLinkedList — Run сценария операций над односвязным списком:
// на выходе SSE-поток кадров до конца сценария. Тонкий обработчик
// по шаблону handleList: парсинг → валидация → runStreaming.
func handleLinkedList(w http.ResponseWriter, r *http.Request) {
	ops, err := parseLinkedScript(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkLinkedScript(ops); err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Сценарий валидирован до Run: LinkedRun ошибки входа не даёт.
		_ = render.LinkedRun(ops, emit)
	})
}

// parseLinkedScript разбирает параметр script — сценарий операций, по одной
// на строку: «prepend N», «append N», «insert I N» (индекс 0..текущая длина
// включительно), «remove I» (индекс 0..текущая длина-1) или «find N»
// (значение). Пустые строки пропускаются. find не валидируется по списку:
// отсутствующее значение — успешный промах, а не ошибка входа.
func parseLinkedScript(r *http.Request) ([]render.LinkedOp, error) {
	q := r.URL.Query()
	if !q.Has("script") {
		return nil, fmt.Errorf("параметр script обязателен: операции по одной на строку, например «prepend 4», «insert 1 9», «find 7»")
	}
	raw := q.Get("script")

	var ops []render.LinkedOp
	for i, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue // пустая строка — не операция
		}
		switch fields[0] {
		case render.LinkedOpPrepend, render.LinkedOpAppend:
			if len(fields) != 2 {
				return nil, fmt.Errorf("строка %d: %s требует ровно один аргумент — целое число, например «%s 4»",
					i+1, fields[0], fields[0])
			}
			v, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: %s N — значение одним целым",
					i+1, fields[1], fields[0])
			}
			ops = append(ops, render.LinkedOp{Kind: fields[0], Value: v})
		case render.LinkedOpInsert:
			if len(fields) != 3 {
				return nil, fmt.Errorf("строка %d: insert требует ровно два аргумента — индекс и значение, например «insert 1 9»", i+1)
			}
			index, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: insert I N — индекс и значение целыми", i+1, fields[1])
			}
			v, err := strconv.Atoi(fields[2])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: insert I N — индекс и значение целыми", i+1, fields[2])
			}
			ops = append(ops, render.LinkedOp{Kind: render.LinkedOpInsert, Index: index, Value: v})
		case render.LinkedOpRemove:
			if len(fields) != 2 {
				return nil, fmt.Errorf("строка %d: remove требует ровно один аргумент — индекс, например «remove 0»", i+1)
			}
			index, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: remove I — индекс одним целым", i+1, fields[1])
			}
			ops = append(ops, render.LinkedOp{Kind: render.LinkedOpRemove, Index: index})
		case render.LinkedOpFind:
			if len(fields) != 2 {
				return nil, fmt.Errorf("строка %d: find требует ровно один аргумент — значение, например «find 7»", i+1)
			}
			v, err := strconv.Atoi(fields[1])
			if err != nil {
				return nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: find N — значение одним целым", i+1, fields[1])
			}
			ops = append(ops, render.LinkedOp{Kind: render.LinkedOpFind, Value: v})
		default:
			return nil, fmt.Errorf("строка %d: неизвестная операция %q: ожидается «prepend N», «append N», «insert I N», «remove I» или «find N»",
				i+1, fields[0])
		}
	}

	if len(ops) == 0 {
		return nil, fmt.Errorf("сценарий пуст: нужна хотя бы одна операция (prepend N, append N, insert I N, remove I или find N)")
	}
	if len(ops) > maxLinkedOps {
		return nil, fmt.Errorf("сценарий слишком длинный: %d операций, максимум %d", len(ops), maxLinkedOps)
	}
	return ops, nil
}

// checkLinkedScript проверяет сценарий прогоном по текущей длине списка
// (числу узлов): prepend/append растят список, вставка допускает индекс
// 0..длина включительно, удаление — 0..длина-1; удалить из пустого списка
// или по индексу вне диапазона — ошибка входа. find пропускаем всегда:
// отсутствующее значение — легальный промах, кадр его покажет.
func checkLinkedScript(ops []render.LinkedOp) error {
	length := 0
	for i, op := range ops {
		switch op.Kind {
		case render.LinkedOpPrepend, render.LinkedOpAppend:
			if length+1 > maxLinkedLength {
				return fmt.Errorf("операция №%d: список вырастет до %d узлов — максимум %d; сократите сценарий",
					i+1, length+1, maxLinkedLength)
			}
			length++
			continue
		case render.LinkedOpInsert:
			if op.Index < 0 || op.Index > length {
				return fmt.Errorf("операция №%d: insert %d в список длины %d — индекс вне диапазона 0..%d",
					i+1, op.Index, length, length)
			}
			if length+1 > maxLinkedLength {
				return fmt.Errorf("операция №%d: список вырастет до %d узлов — максимум %d; сократите сценарий",
					i+1, length+1, maxLinkedLength)
			}
			length++
			continue
		case render.LinkedOpRemove:
			if length == 0 {
				return fmt.Errorf("операция №%d: remove из пустого списка — нечего удалять", i+1)
			}
			if op.Index < 0 || op.Index >= length {
				return fmt.Errorf("операция №%d: remove %d в списке длины %d — индекс вне диапазона 0..%d",
					i+1, op.Index, length, length-1)
			}
			length--
			continue
		case render.LinkedOpFind:
			// Промах — не ошибка входа; визиты обхода попадут в кадры.
			continue
		default:
			// Недостижимо: parseLinkedScript пропускает только известные виды.
			return fmt.Errorf("операция №%d: неизвестный вид операции %q", i+1, op.Kind)
		}
	}
	return nil
}
