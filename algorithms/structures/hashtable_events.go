// События хеш-таблицы (ADR-0001): каждое событие — атомарный шаг операции,
// достаточный для построения кадра без знания внутренностей таблицы
// (slide-массива корзин, полей записей). Шаги операции: выбор корзины
// хешированием, обход записей цепочки, вставка/обновление/удаление и
// результат поиска (найдено/не найдено).
package structures

import "github.com/EvgeniiIvanov/algo/algorithms"

// HashEvent — маркер-интерфейс событий хеш-таблицы (sealed): вне пакета
// свои события реализовать нельзя.
type HashEvent interface{ isHashEvent() }

// HashBucket — хеширование выбрало корзину Index для ключа текущей операции:
// любые чтения и изменения пойдут в цепочку этой корзины, остальные не
// затрагиваются. Индекс уже нормализован в диапазон 0..числоКорзин-1; Raw —
// тот же ключ до нормализации (отрицательный остаток при отрицательном
// ключе): визуализатор объясняет расчёт по факту события, не повторяя
// формулу остатка библиотеки — она живёт в bucketOf в одном экземпляре.
type HashBucket struct {
	Index int // нормализованный индекс выбранной корзины
	Raw   int // сырой остаток ключа % число-корзин до нормализации
}

// HashVisit — обход цепочки корзины Bucket дошёл до записи Position
// (нумерация с головы цепочки, 0 — голова): сверяем её ключ с искомым.
// Единственный способ найти запись — пройти цепочку звено за звеном,
// и каждая посещённая запись — шаг операции, хорошо видимый при коллизии.
type HashVisit struct {
	Bucket   int // корзина, чью цепочку обходим
	Position int // позиция записи в цепочке (0 — голова)
	Key      int // ключ посещённой записи
	Value    int // значение посещённой записи
}

// HashInsert — новая пара записана в корзину Bucket. Запись встаёт во главу
// цепочки (позиция 0): соседние записи не двигаются, обход Get/Delete
// добирается до них уже по новой голове.
type HashInsert struct {
	Bucket   int // корзина, в которую вставили запись
	Position int // позиция записи после вставки (0 — голова цепочки)
	Key      int // ключ вставленной записи
	Value    int // значение вставленной записи
}

// HashUpdate — значение существующего ключа переписано вместо вставки:
// число записей таблицы не изменилось, а запись осталась на своей позиции.
type HashUpdate struct {
	Bucket   int // корзина, где лежит запись
	Position int // позиция записи в цепочке (0 — голова)
	Key      int // ключ обновлённой записи
	Value    int // новое значение записи
}

// HashRemove — запись удалена из цепочки корзины: связи соседних записей
// перепрыгнули через удалённую, остальные записи остались доступны.
type HashRemove struct {
	Bucket   int // корзина, из которой удалили запись
	Position int // позиция, которую запись занимала в цепочке
	Key      int // ключ удалённой записи
	Value    int // удалённое значение
}

// HashFound — поиск успешен: запись с искомым ключом найдена в корзине.
type HashFound struct {
	Bucket   int // корзина, где нашёлся ключ
	Position int // позиция записи в цепочке (0 — голова)
	Key      int // искомый ключ
	Value    int // найденное значение
}

// HashNotFound — цепочка пройдена до конца, искомого ключа нет: промах,
// штатный результат поиска (не ошибка) — как для Get, так и для Delete.
type HashNotFound struct {
	Bucket int // корзина, чью цепочку прошли до конца
	Key    int // ключ, который не нашёлся
}

func (HashBucket) isHashEvent()   {}
func (HashVisit) isHashEvent()    {}
func (HashInsert) isHashEvent()   {}
func (HashUpdate) isHashEvent()   {}
func (HashRemove) isHashEvent()   {}
func (HashFound) isHashEvent()    {}
func (HashNotFound) isHashEvent() {}

// PutWithEmitter — Put в режиме наблюдения (см. ADR-0001): выбор корзины
// (HashBucket) и обход записей цепочки (HashVisit) до поиска ключа, затем
// сама вставка или обновление через тихий Put — мутация живёт в одном
// месте, у тихой версии. Новая запись встаёт во главу
// цепочки (HashInsert), обновление переписывает значение на месте
// (HashUpdate). Поведение и результат — те же, что у тихого Put.
func (h *HashTable) PutWithEmitter(key, value int, em algorithms.Emitter[HashEvent]) {
	idx := h.bucketOf(key)
	emitHash(em, HashBucket{Index: idx, Raw: h.rawRemainder(key)})

	// Обход существующей цепочки для событий: находим ключ и его позицию.
	position := 0
	existing := false
	for cur := h.buckets[idx]; cur != nil; cur = cur.next {
		emitHash(em, HashVisit{Bucket: idx, Position: position, Key: cur.key, Value: cur.value})
		if cur.key == key {
			existing = true
			break
		}
		position++
	}

	h.Put(key, value) // сама мутация — код тихой версии; внутри он обходит цепочку ещё раз
	if existing {
		emitHash(em, HashUpdate{Bucket: idx, Position: position, Key: key, Value: value})
	} else {
		// Новая запись всегда становится головой цепочки — позиция 0.
		emitHash(em, HashInsert{Bucket: idx, Position: 0, Key: key, Value: value})
	}
}

// GetWithEmitter — Get в режиме наблюдения: выбор корзины (HashBucket),
// обход цепочки с HashVisit каждой проверенной записи и результат —
// HashFound на найденной записи или HashNotFound при промахе. Промах — не
// ошибка, результат тот же, что у тихого Get: (0, false).
func (h *HashTable) GetWithEmitter(key int, em algorithms.Emitter[HashEvent]) (int, bool) {
	idx := h.bucketOf(key)
	emitHash(em, HashBucket{Index: idx, Raw: h.rawRemainder(key)})

	position := 0
	for cur := h.buckets[idx]; cur != nil; cur = cur.next {
		emitHash(em, HashVisit{Bucket: idx, Position: position, Key: cur.key, Value: cur.value})
		if cur.key == key {
			emitHash(em, HashFound{Bucket: idx, Position: position, Key: key, Value: cur.value})
			return cur.value, true
		}
		position++
	}
	emitHash(em, HashNotFound{Bucket: idx, Key: key})
	return 0, false
}

// DeleteWithEmitter — Delete в режиме наблюдения: выбор корзины (HashBucket),
// обход цепочки с HashVisit проверенных записей и результат — HashRemove
// при попадании (тихий Delete делает мутацию) или HashNotFound при промахе.
// Возвращает признак удаления, как тихий Delete.
func (h *HashTable) DeleteWithEmitter(key int, em algorithms.Emitter[HashEvent]) bool {
	idx := h.bucketOf(key)
	emitHash(em, HashBucket{Index: idx, Raw: h.rawRemainder(key)})

	// Обход цепочки для событий; позиция находит найденный ключ.
	position := 0
	victim := (*hashEntry)(nil)
	for cur := h.buckets[idx]; cur != nil; cur = cur.next {
		emitHash(em, HashVisit{Bucket: idx, Position: position, Key: cur.key, Value: cur.value})
		if cur.key == key {
			victim = cur
			break
		}
		position++
	}

	if victim == nil {
		emitHash(em, HashNotFound{Bucket: idx, Key: key})
		return false
	}
	value := victim.value
	h.Delete(key) // сама мутация — код тихой версии
	emitHash(em, HashRemove{Bucket: idx, Position: position, Key: key, Value: value})
	return true
}

// emitHash вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitHash(em algorithms.Emitter[HashEvent], event HashEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
