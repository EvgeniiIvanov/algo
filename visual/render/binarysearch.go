// Пакет render — Frame Renderer'ы: события алгоритмов → кадры (ADR-0002).
//
// Каждый визуализируемый алгоритм получает свой рендерер. Рендерер держит
// текущее состояние (границы, последнюю середину) и после каждого события
// отдаёт самодостаточный кадр в emit — транспорт о событиях не знает.
package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/search"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Snapshot — снимок массива в кадре бинарного поиска.
// Все поля для отрисовки внутри кадра: клиент ничего не докапливает.
type Snapshot struct {
	Values     []int `json:"values"`     // копия массива (кадр не зависит от будущего)
	Low        int   `json:"low"`        // нижняя граница поиска
	High       int   `json:"high"`       // верхняя граница поиска
	Mid        int   `json:"mid"`        // индекс сравниваемой середины; -1 — на этом шаге нет
	FoundIndex int   `json:"foundIndex"` // индекс найденного элемента; -1 — не найден
}

// BinarySearch — Frame Renderer бинарного поиска: запускает поиск
// и на каждый шаг отдаёт кадр в emit. Возвращает результат поиска,
// как тихий Search.
func BinarySearch(data []int, target int, emit func(frame.Frame)) int {
	b := &binarySearchFrames{
		data:   data,
		target: target,
		low:    0,
		high:   len(data) - 1,
		mid:    -1,
		found:  -1,
		emit:   emit,
	}

	b.frame(fmt.Sprintf("Начинаем поиск значения %d в массиве из %d элементов. Границы: [%d..%d].",
		target, len(data), b.low, b.high))

	return search.SearchWithEmitter(data, target, b)
}

// BinarySearchFrames — все кадры выполнения сразу (удобно для тестов).
func BinarySearchFrames(data []int, target int) []frame.Frame {
	var frames []frame.Frame
	BinarySearch(data, target, func(f frame.Frame) { frames = append(frames, f) })
	return frames
}

// binarySearchFrames — накапливает состояние между событиями и строит кадры.
// Реализует algorithms.Emitter[search.Event].
type binarySearchFrames struct {
	data   []int
	target int
	low    int
	high   int
	mid    int // индекс последней сравненной середины
	found  int
	step   int
	emit   func(frame.Frame)
}

// Emit — один шаг алгоритма → один кадр (ADR-0002).
func (b *binarySearchFrames) Emit(ev search.Event) {
	switch e := ev.(type) {
	case search.Compare:
		b.mid = e.Mid
		b.frame(fmt.Sprintf("Смотрим середину диапазона [%d..%d]: индекс %d, значение %d.",
			b.low, b.high, e.Mid, e.Value))

	case search.Bounds:
		b.low, b.high = e.Low, e.High
		var expl string
		switch {
		case b.low > b.high:
			expl = fmt.Sprintf("Границы пересеклись ([%d..%d]) — диапазон пуст.", b.low, b.high)
		case b.target > b.data[b.mid]:
			expl = fmt.Sprintf("%d больше %d — ищем в правой половине: границы [%d..%d].",
				b.target, b.data[b.mid], b.low, b.high)
		default:
			expl = fmt.Sprintf("%d меньше %d — ищем в левой половине: границы [%d..%d].",
				b.target, b.data[b.mid], b.low, b.high)
		}
		b.mid = -1
		b.frame(expl)

	case search.Found:
		b.found = e.Index
		b.mid = -1
		b.frame(fmt.Sprintf("Значение %d найдено на индексе %d.", b.target, e.Index))

	case search.NotFound:
		b.mid = -1
		b.frame(fmt.Sprintf("Диапазон пуст — значения %d в массиве нет.", b.target))
	}
}

// frame собирает самодостаточный кадр: копия массива + текущие подсветки.
// Шаги нумеруются с 1: стартовый кадр, затем по кадру на каждое событие.
func (b *binarySearchFrames) frame(explanation string) {
	if b.emit == nil {
		return
	}
	b.step++
	b.emit(frame.Frame{
		Step:        b.step,
		Kind:        frame.KindArray,
		Explanation: explanation,
		Data: Snapshot{
			Values:     append([]int(nil), b.data...),
			Low:        b.low,
			High:       b.high,
			Mid:        b.mid,
			FoundIndex: b.found,
		},
	})
}
