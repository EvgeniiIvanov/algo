// Пакет server — HTTP-слой визуализатора: приём Run и стрим кадров по SSE
// (ADR-0003). Транспорт знает только про frame.Frame: новый алгоритм — это
// новый маршрут поверх того же стриминга, правок streamFrames не нужно.
package server

import (
	"encoding/json"
	"fmt"
	"log"
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
// Здесь только вход алгоритма: вся стриминговая обвязка — в runStreaming.
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

	runStreaming(w, r, func(emit func(frame.Frame)) {
		render.BinarySearch(values, target, emit)
	})
}

// runStreaming — мост «алгоритм → SSE» (ADR-0003): запускает run в горутине
// и стримит его кадры клиенту. Transport-обвязка живёт здесь; новый алгоритм —
// это новый маршрут и тонкий обработчик «парсинг → валидация → runStreaming»,
// стриминг не правится и не копируется.
func runStreaming(w http.ResponseWriter, r *http.Request, run func(emit func(frame.Frame))) {
	frames := make(chan frame.Frame)
	go func() {
		defer close(frames)
		run(func(f frame.Frame) {
			select {
			case frames <- f:
			case <-r.Context().Done(): // клиент отключился — не зависаем
			}
		})
	}()

	streamFrames(w, frames)
}

// streamFrames — SSE-транспорт (ADR-0003): каждый кадр — событие data,
// после последнего кадра поток завершается.
func streamFrames(w http.ResponseWriter, frames <-chan frame.Frame) {
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
			// Кадр, который не маршалится, отдать нельзя: заголовки уже
			// отправлены, WriteHeader не сработает. Логируем и рвём поток —
			// известных входов, ведущих сюда, нет.
			log.Printf("кадр не маршалится: %v", err)
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
