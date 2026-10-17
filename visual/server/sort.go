package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// Ограничение входа Run сортировки: чем массив длиннее, тем легче теряется
// обозримость ряда ячеек и потока кадров. Это safety bound одного визуального
// Run — библиотечный sort.BubbleSort лимитов не имеет.
const maxSortValues = 100

// handleBubbleSort — Run bubble sort: вход — массив целых чисел, на выходе
// SSE-поток кадров до конца сортировки. Тонкий обработчик по шаблону
// handleList: парсинг → валидация → runStreaming.
func handleBubbleSort(w http.ResponseWriter, r *http.Request) {
	values, err := parseSortInput(r)
	if err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		render.BubbleRun(values, emit)
	})
}

// parseSortInput разбирает параметр values — целые числа через запятую,
// например values=3,1,2. Пустой массив — легальный вход: сортировать
// нечего, кадр покажет пустой ряд. Пустая компонента между запятыми
// и мусор вместо целого отвергаются до начала SSE; длина ограничена
// maxSortValues.
func parseSortInput(r *http.Request) ([]int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("values"))
	if raw == "" {
		return []int{}, nil
	}

	values := []int{}
	for _, part := range strings.Split(raw, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("не удалось разобрать %q как целое число: values — целые числа через запятую, например values=3,1,2", part)
		}
		values = append(values, v)
	}
	if len(values) > maxSortValues {
		return nil, fmt.Errorf("массив слишком длинный: %d элементов, максимум %d; сократите вход", len(values), maxSortValues)
	}
	return values, nil
}
