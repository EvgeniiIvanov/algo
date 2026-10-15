package structures_test

import (
	"reflect"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// spyHashEmitter — тестовый Emitter: собирает события хеш-таблицы в слайс
// для проверки последовательности шагов.
type spyHashEmitter []structures.HashEvent

func (s *spyHashEmitter) Emit(event structures.HashEvent) { *s = append(*s, event) }

// Известный сценарий с коллизией: события шагов — выбор корзины, визиты
// записей цепочки, вставка/обновление/удаление и результат поиска.
// При 5 корзинах ключи 1, 6, 11 и -4 попадают в корзину 1, ключ 9 — в 4.
func TestHashTableEventsScenario(t *testing.T) {
	want := []structures.HashEvent{
		// put 1 10: корзина 1 пуста — только выбор и вставка.
		structures.HashBucket{Index: 1, Raw: 1},
		structures.HashInsert{Bucket: 1, Position: 0, Key: 1, Value: 10},
		// put 6 20: коллизия — визит записи 1, вставка во главу цепочки.
		structures.HashBucket{Index: 1, Raw: 1},
		structures.HashVisit{Bucket: 1, Position: 0, Key: 1, Value: 10},
		structures.HashInsert{Bucket: 1, Position: 0, Key: 6, Value: 20},
		// put -4 40: -4 % 5 = -4 → нормализация (Raw сырой) в корзину 1; выше коллизия.
		structures.HashBucket{Index: 1, Raw: -4},
		structures.HashVisit{Bucket: 1, Position: 0, Key: 6, Value: 20},
		structures.HashVisit{Bucket: 1, Position: 1, Key: 1, Value: 10},
		structures.HashInsert{Bucket: 1, Position: 0, Key: -4, Value: 40},
		// put 1 12: обновление существующего ключа вместо вставки; сверен
		// и ключ 6 — с ним ничего не случилось.
		structures.HashBucket{Index: 1, Raw: 1},
		structures.HashVisit{Bucket: 1, Position: 0, Key: -4, Value: 40},
		structures.HashVisit{Bucket: 1, Position: 1, Key: 6, Value: 20},
		structures.HashVisit{Bucket: 1, Position: 2, Key: 1, Value: 10},
		structures.HashUpdate{Bucket: 1, Position: 2, Key: 1, Value: 12},
		// get 1: визиты до попадания на позиции 2.
		structures.HashBucket{Index: 1, Raw: 1},
		structures.HashVisit{Bucket: 1, Position: 0, Key: -4, Value: 40},
		structures.HashVisit{Bucket: 1, Position: 1, Key: 6, Value: 20},
		structures.HashVisit{Bucket: 1, Position: 2, Key: 1, Value: 12},
		structures.HashFound{Bucket: 1, Position: 2, Key: 1, Value: 12},
		// get 9: корзина 4, цепочка пуста — промах без визитов.
		structures.HashBucket{Index: 4, Raw: 4},
		structures.HashNotFound{Bucket: 4, Key: 9},
		// delete 6: визиты записей -4 и 6, удаление из середины цепочки.
		structures.HashBucket{Index: 1, Raw: 1},
		structures.HashVisit{Bucket: 1, Position: 0, Key: -4, Value: 40},
		structures.HashVisit{Bucket: 1, Position: 1, Key: 6, Value: 20},
		structures.HashRemove{Bucket: 1, Position: 1, Key: 6, Value: 20},
		// delete -4: визит головы цепочки и удаление головы.
		structures.HashBucket{Index: 1, Raw: -4},
		structures.HashVisit{Bucket: 1, Position: 0, Key: -4, Value: 40},
		structures.HashRemove{Bucket: 1, Position: 0, Key: -4, Value: 40},
		// delete 6: ключа больше нет — визитом проходит оставшаяся запись 1,
		// промах после полного обхода цепочки.
		structures.HashBucket{Index: 1, Raw: 1},
		structures.HashVisit{Bucket: 1, Position: 0, Key: 1, Value: 12},
		structures.HashNotFound{Bucket: 1, Key: 6},
	}

	table := newTable(t, 5)
	var spy spyHashEmitter

	table.PutWithEmitter(1, 10, &spy)
	table.PutWithEmitter(6, 20, &spy)
	table.PutWithEmitter(-4, 40, &spy)
	table.PutWithEmitter(1, 12, &spy)
	if got, ok := table.GetWithEmitter(1, &spy); !ok || got != 12 {
		t.Fatalf("GetWithEmitter(1) = (%d, %v), ожидалось (12, true)", got, ok)
	}
	if got, ok := table.GetWithEmitter(9, &spy); ok || got != 0 {
		t.Fatalf("GetWithEmitter(9) = (%d, %v), ожидался промах", got, ok)
	}
	if !table.DeleteWithEmitter(6, &spy) {
		t.Fatalf("DeleteWithEmitter(6) = false, ожидалось true")
	}
	if !table.DeleteWithEmitter(-4, &spy) {
		t.Fatalf("DeleteWithEmitter(-4) = false, ожидалось true")
	}
	if table.DeleteWithEmitter(6, &spy) {
		t.Fatalf("DeleteWithEmitter(6) повторно = true, ожидался промах (false)")
	}

	if !reflect.DeepEqual([]structures.HashEvent(spy), want) {
		t.Fatalf("последовательность событий:\n получено %v\n ожидалось %v",
			[]structures.HashEvent(spy), want)
	}
	// Содержимое после сценария согласовано с трейсом.
	if got := table.Len(); got != 1 {
		t.Fatalf("Len после сценария = %d, ожидалось 1", got)
	}
}

// Без эмиттера (quiet mode) WithEmitter-операции работают как обычные.
func TestHashTableWithEmitterQuietMode(t *testing.T) {
	table := newTable(t, 5)

	table.PutWithEmitter(1, 10, nil)
	table.PutWithEmitter(6, 20, nil)
	if got, ok := table.GetWithEmitter(6, nil); !ok || got != 20 {
		t.Fatalf("GetWithEmitter(6, nil) = (%d, %v), ожидалось (20, true)", got, ok)
	}
	if got, ok := table.GetWithEmitter(9, nil); ok || got != 0 {
		t.Fatalf("GetWithEmitter(9, nil) = (%d, %v), ожидался промах", got, ok)
	}
	if !table.DeleteWithEmitter(1, nil) {
		t.Fatalf("DeleteWithEmitter(1, nil) = false, ожидалось true")
	}
	if table.DeleteWithEmitter(99, nil) {
		t.Fatalf("DeleteWithEmitter(99, nil) = true, ожидалось false")
	}
	if got := table.Len(); got != 1 {
		t.Fatalf("Len = %d, ожидалось 1", got)
	}
}
