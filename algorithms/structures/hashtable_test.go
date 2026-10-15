package structures_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
)

// Тесты шва 1 (см. docs/SPEC.md): публичный API хеш-таблицы в quiet mode —
// поведение операций, инварианты, граничные случаи. Коллизии достигаются
// детерминированным modulo-хешированием: при 5 корзинах ключи 1, 6 и 11
// всегда попадают в корзину 1.

func newTable(t *testing.T, nBuckets int) *structures.HashTable {
	t.Helper()

	table, err := structures.NewHashTable(nBuckets)
	if err != nil {
		t.Fatalf("NewHashTable(%d): %v", nBuckets, err)
	}
	return table
}

// Некорректное число корзин при создании — ошибка, а не паника:
// отрицательное и нулевое значения.
func TestNewHashTableInvalidBuckets(t *testing.T) {
	for _, n := range []int{-10, -1, 0} {
		table, err := structures.NewHashTable(n)
		if !errors.Is(err, structures.ErrHashTableInvalidBuckets) {
			t.Fatalf("NewHashTable(%d): ошибка = %v, ожидалась ErrHashTableInvalidBuckets", n, err)
		}
		if table != nil {
			t.Fatalf("NewHashTable(%d): таблица = %v, ожидалось nil", n, table)
		}
	}
}

// Пустая таблица: Len = 0, поиск и удаление — промахи (не ошибки).
func TestEmptyHashTable(t *testing.T) {
	table := newTable(t, 5)

	if got := table.Len(); got != 0 {
		t.Fatalf("Len пустой таблицы = %d, ожидалось 0", got)
	}
	if got, ok := table.Get(1); ok || got != 0 {
		t.Fatalf("Get по пустой таблице = (%d, %v), ожидалось (0, false)", got, ok)
	}
	if table.Delete(1) {
		t.Fatalf("Delete по пустой таблице = true, ожидалось false")
	}
}

// Put/Get по ключам разлетаются по корзинам и остаются доступны.
func TestHashTablePutGet(t *testing.T) {
	table := newTable(t, 5)

	table.Put(1, 10)
	table.Put(2, 20)
	table.Put(-3, 30) // отрицательный ключ — легален, как любое целое
	if got := table.Len(); got != 3 {
		t.Fatalf("Len = %d, ожидалось 3", got)
	}
	for _, tt := range []struct{ key, want int }{{1, 10}, {2, 20}, {-3, 30}} {
		if got, ok := table.Get(tt.key); !ok || got != tt.want {
			t.Fatalf("Get(%d) = (%d, %v), ожидалось (%d, true)", tt.key, got, ok, tt.want)
		}
	}
}

// Повторный Put существующего ключа обновляет значение, не увеличивая Len.
func TestHashTablePutUpdates(t *testing.T) {
	table := newTable(t, 5)

	table.Put(1, 10)
	table.Put(1, 11)
	table.Put(1, 12)
	if got := table.Len(); got != 1 {
		t.Fatalf("Len после трёх put одного ключа = %d, ожидалось 1", got)
	}
	if got, ok := table.Get(1); !ok || got != 12 {
		t.Fatalf("Get(1) = (%d, %v), ожидалось (12, true)", got, ok)
	}
}

// Коллизия: ключи 1 и 6 при 5 корзинах попадают в одну, обе записи доступны.
func TestHashTableCollision(t *testing.T) {
	table := newTable(t, 5)

	table.Put(1, 10)
	table.Put(6, 20)
	table.Put(11, 30) // тройная коллизия в корзине 1
	if got := table.Len(); got != 3 {
		t.Fatalf("Len = %d, ожидалось 3", got)
	}
	for _, tt := range []struct{ key, want int }{{1, 10}, {6, 20}, {11, 30}} {
		if got, ok := table.Get(tt.key); !ok || got != tt.want {
			t.Fatalf("Get(%d) = (%d, %v), ожидалось (%d, true)", tt.key, got, ok, tt.want)
		}
	}
	// Обновление при коллизии находит свою запись.
	table.Put(6, 21)
	if got, ok := table.Get(6); !ok || got != 21 {
		t.Fatalf("Get(6) после обновления = (%d, %v), ожидалось (21, true)", got, ok)
	}
	if got := table.Len(); got != 3 {
		t.Fatalf("Len после обновления = %d, ожидалось 3", got)
	}
}

// Отрицательный ключ: остаток от деления нормализуется в диапазон индексов,
// поэтому ключи -4 и 1 (и 6) попадают в одну корзину и не теряют друг друга.
func TestHashTableNegativeKey(t *testing.T) {
	table := newTable(t, 5)

	table.Put(-4, 40) // -4 % 5 = -4 → нормализация в корзину 1 — коллизия с 1 и 6
	table.Put(1, 10)
	table.Put(6, 20)
	if got := table.Len(); got != 3 {
		t.Fatalf("Len = %d, ожидалось 3 (все ключи разные, какими бы ни были остатки)", got)
	}
	for _, tt := range []struct{ key, want int }{{-4, 40}, {1, 10}, {6, 20}} {
		if got, ok := table.Get(tt.key); !ok || got != tt.want {
			t.Fatalf("Get(%d) = (%d, %v), ожидалось (%d, true)", tt.key, got, ok, tt.want)
		}
	}
}

// Удаление из середины цепочки не повреждает остальные записи;
// голова цепочки удаляется отдельно; промах delete — не ошибка.
func TestHashTableDeleteFromChain(t *testing.T) {
	table := newTable(t, 5)

	table.Put(1, 10)  // цепочка корзины 1: 1
	table.Put(6, 20)  // 6 → 1
	table.Put(11, 30) // 11 → 6 → 1

	// Середина цепочки.
	if !table.Delete(6) {
		t.Fatalf("Delete(6) = false, ожидалось true (ключ есть)")
	}
	if got := table.Len(); got != 2 {
		t.Fatalf("Len = %d, ожидалось 2", got)
	}
	if _, ok := table.Get(6); ok {
		t.Fatalf("Get(6) попал, ожидался промах")
	}
	if got, ok := table.Get(1); !ok || got != 10 {
		t.Fatalf("запись 1 повреждена удалением 6: Get(1) = (%d, %v), ожидалось (10, true)", got, ok)
	}
	if got, ok := table.Get(11); !ok || got != 30 {
		t.Fatalf("запись 11 повреждена удалением 6: Get(11) = (%d, %v), ожидалось (30, true)", got, ok)
	}

	// Голова оставшейся цепочки (11 → 1: головой стал 11 при вставке 11).
	if !table.Delete(11) {
		t.Fatalf("Delete(11) = false, ожидалось true")
	}
	if _, ok := table.Get(11); ok {
		t.Fatalf("Get(11) попал, ожидался промах")
	}
	if got, ok := table.Get(1); !ok || got != 10 {
		t.Fatalf("хвост цепочки потерян удалением головы: Get(1) = (%d, %v), ожидалось (10, true)", got, ok)
	}

	// Повторное удаление того же ключа и удаление чужого — промахи.
	if table.Delete(11) || table.Delete(99) {
		t.Fatalf("Delete отсутствующего ключа = true, ожидалось false")
	}
	if got := table.Len(); got != 1 {
		t.Fatalf("Len = %d, ожидалось 1", got)
	}
}

// Одна корзина: все ключи живут в единственной цепочке — вырожденный случай,
// коллизия на каждый ключ, поиск линеен (худший случай O(n)).
func TestHashTableSingleBucket(t *testing.T) {
	table := newTable(t, 1)

	for i := 0; i < 5; i++ {
		table.Put(i*10, i)
	}
	if got := table.Len(); got != 5 {
		t.Fatalf("Len = %d, ожидалось 5", got)
	}
	if got, ok := table.Get(40); !ok || got != 4 {
		t.Fatalf("Get(40) = (%d, %v), ожидалось (4, true)", got, ok)
	}
	if !table.Delete(20) {
		t.Fatalf("Delete(20) = false, ожидалось true")
	}
	for _, key := range []int{0, 10, 30, 40} {
		if got, ok := table.Get(key); !ok {
			t.Fatalf("Get(%d) = (%d, %v), ожидался попадание", key, got, ok)
		}
	}
	if _, ok := table.Get(20); ok {
		t.Fatalf("Get(20) попал, ожидался промах")
	}
}
