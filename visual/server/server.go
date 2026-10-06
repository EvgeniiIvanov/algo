// Пакет server — HTTP-слой визуализатора: приём Run и стрим кадров по SSE
// (ADR-0003). Транспорт знает только про frame.Frame: новый алгоритм — это
// новый маршрут поверх тех же хелперов, правок транспорта не нужно.
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// NewMux — маршруты визуализатора.
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/run/binary-search", handleBinarySearch)
	return mux
}

// handleBinarySearch — Run бинарного поиска: вход (массив + искомое значение),
// на выходе SSE-поток кадров до конца выполнения.
func handleBinarySearch(w http.ResponseWriter, r *http.Request) {
	values, target, err := parseRunInput(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkSorted(values); err != nil {
		writeError(w, err)
		return
	}

	// Кадры стримятся по мере выполнения: рендерер работает в горутине,
	// каждый кадр сразу уходит клиенту.
	frames := make(chan frame.Frame)
	go func() {
		defer close(frames)
		render.BinarySearch(values, target, func(f frame.Frame) {
			select {
			case frames <- f:
			case <-r.Context().Done(): // клиент отключился — не зависаем
			}
		})
	}()

	streamFrames(w, r, frames)
}

// streamFrames — SSE-транспорт (ADR-0003): каждый кадр — событие data,
// после последнего кадра поток завершается.
func streamFrames(w http.ResponseWriter, r *http.Request, frames <-chan frame.Frame) {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, fmt.Errorf("стриминг не поддерживается этим транспортом"))
		return
	}

	for f := range frames {
		data, err := json.Marshal(f)
		if err != nil {
			writeError(w, err) // заголовки уже отправлены — ошибка уйдёт в поток
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
}

// parseRunInput разбирает вход Run из query-параметров: values — целые числа
// через запятую, target — искомое значение.
func parseRunInput(r *http.Request) ([]int, int, error) {
	raw := r.URL.Query().Get("values")
	if strings.TrimSpace(raw) == "" {
		return nil, 0, fmt.Errorf("параметр values обязателен: целые числа через запятую, например values=1,3,5,7,9")
	}

	var values []int
	for _, part := range strings.Split(raw, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, 0, fmt.Errorf("не удалось разобрать %q как целое число: values — целые числа через запятую", part)
		}
		values = append(values, v)
	}

	rawTarget := r.URL.Query().Get("target")
	if strings.TrimSpace(rawTarget) == "" {
		return nil, 0, fmt.Errorf("параметр target обязателен: искомое значение, например target=7")
	}
	target, err := strconv.Atoi(strings.TrimSpace(rawTarget))
	if err != nil {
		return nil, 0, fmt.Errorf("не удалось разобрать %q как целое число: target — одно целое число", rawTarget)
	}

	return values, target, nil
}

// checkSorted проверяет предусловие бинарного поиска — массив по возрастанию.
func checkSorted(values []int) error {
	for i := 1; i < len(values); i++ {
		if values[i] < values[i-1] {
			return fmt.Errorf("массив должен быть отсортирован по возрастанию: элемент %d на позиции %d больше элемента %d на позиции %d",
				values[i], i, values[i-1], i-1)
		}
	}
	return nil
}

// writeError — понятная ошибка вместо кадров с мусором (400).
func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
