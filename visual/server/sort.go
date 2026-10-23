package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Ограничение входа Run сортировки: чем массив длиннее, тем легче теряется
// обозримость ряда ячеек и потока кадров. Это safety bound одного визуального
// Run — библиотечные сортировки (sort.BubbleSort, sort.SelectionSort, ...)
// лимитов не имеют.
const maxSortValues = 100

// handleSort — Run сортировки на общем входе (массив целых чисел),
// шаблон тонкого обработчика тоже — парсинг → валидация → runStreaming.
// Различается только Frame Renderer алгоритма (run), он же задаёт маршрут
// при регистрации; сопоставление «событие → кадр» и пояснения живут
// в visual/render/sort.go.
func handleSort(run func(values []int, emit func(frame.Frame))) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values, err := parseSortInput(r)
		if err != nil {
			writeError(w, err)
			return
		}

		runStreaming(w, r, func(emit func(frame.Frame)) {
			run(values, emit)
		})
	}
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
