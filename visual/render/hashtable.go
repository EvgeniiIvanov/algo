package render

import (
	"fmt"

	"github.com/EvgeniiIvanov/algo/algorithms/structures"
	"github.com/EvgeniiIvanov/algo/visual/frame"
)

// Виды операций сценария хеш-таблицы.
const (
	HashTableOpPut    = "put"
	HashTableOpGet    = "get"
	HashTableOpDelete = "delete"
)

// Дополнительная роль подсветки хеш-таблицы: значение существующего ключа
// обновлено. Остальные роли — общие правила пакета (RoleAdded, RoleRemoved,
// RoleVisited, RoleFound объявлены в stack.go и linkedlist.go).
const RoleUpdated = "updated"

// HashTableOp — одна операция сценария хеш-таблицы.
type HashTableOp struct {
	Kind  string // вид операции: HashTableOpPut / Get / Delete
	Key   int    // ключ операции
	Value int    // значение для put
}

// HashTableRun — Frame Renderer хеш-таблицы: прогоняет сценарий операций
// put/get/delete над таблицей с заданным числом корзин и отдаёт кадр на
// каждое событие. Валидация входа живёт в маршруте (пределы 1..100 корзин,
// ≤100 записей, ≤500 операций; get/delete по чужому ключу — легальный
// промах, а не ошибка входа). Ошибка здесь — только защитная от сценария,
// просочившегося мимо валидации маршрута: некорректное число корзин или
// неизвестный вид операции.
func HashTableRun(buckets int, ops []HashTableOp, emit func(frame.Frame)) error {
	table, err := structures.NewHashTable(buckets)
	if err != nil {
		return fmt.Errorf("хеш-таблица: %w", err)
	}

	r := &hashTableFrames{
		mirror:          make([][]HashTableEntry, buckets),
		highlightBucket: -1,
		emit:            emit,
	}

	r.frame(fmt.Sprintf("Создаём хеш-таблицу: %d корзин, коллизии разрешаются цепочками. Впереди операций: %d.",
		buckets, len(ops)))

	for _, op := range ops {
		// Ключ операции запоминаем: пояснение выбора корзины отсчитывает
		// его остаток от деления на число корзин.
		r.operationKey = op.Key
		switch op.Kind {
		case HashTableOpPut:
			r.operation = fmt.Sprintf("%s %d %d", HashTableOpPut, op.Key, op.Value)
			table.PutWithEmitter(op.Key, op.Value, r)
		case HashTableOpGet:
			r.operation = fmt.Sprintf("%s %d", HashTableOpGet, op.Key)
			// Промах — нормальный результат поиска, а не ошибка.
			_, _ = table.GetWithEmitter(op.Key, r)
		case HashTableOpDelete:
			r.operation = fmt.Sprintf("%s %d", HashTableOpDelete, op.Key)
			_ = table.DeleteWithEmitter(op.Key, r)
		default:
			// Недостижимо: parseHashTableInput маршрута пропускает только известные виды.
			return fmt.Errorf("неизвестный вид операции %q", op.Kind)
		}
	}
	return nil
}

// HashTableFrames — все кадры сценария сразу (удобно для тестов).
func HashTableFrames(buckets int, ops []HashTableOp) ([]frame.Frame, error) {
	var frames []frame.Frame
	if err := HashTableRun(buckets, ops, func(f frame.Frame) { frames = append(frames, f) }); err != nil {
		return nil, err
	}
	return frames, nil
}

// HashTableEntry — запись цепочки одной корзины в снимке кадра.
type HashTableEntry struct {
	Key   int `json:"key"`   // ключ записи
	Value int `json:"value"` // значение записи
}

// HashTableHighlight — подсветка одной записи цепочки с ролью: что с ней
// случилось на шаге. Роли — записи, которые в снапшоте есть; удалённая
// запись в снапшоте не живёт и подсветкой не отмечается — она несётся
// надгробием в Removed (см. HashTableRemoval).
type HashTableHighlight struct {
	Bucket   int    `json:"bucket"`   // корзина записи
	Position int    `json:"position"` // позиция записи в цепочке (0 — голова)
	Role     string `json:"role"`     // роль подсветки
}

// HashTableRemoval — запись, удалённая на этом шаге: надгробие на бывшей
// позиции в цепочке. Самого снапшота корзин её уже нет, а подсветка role
// "removed" на живой записи указывала бы на соседа, вставшего на её место:
// надгробие несёт ключ и значение, чтобы учащийся видел, что именно
// удалилось и где оно стояло.
type HashTableRemoval struct {
	Bucket   int `json:"bucket"`   // корзина, из которой удалили запись
	Position int `json:"position"` // бывшая позиция записи в цепочке (0 — голова)
	Key      int `json:"key"`      // ключ удалённой записи
	Value    int `json:"value"`    // удалённое значение
}

// HashTableSnapshot — снимок хеш-таблицы: Data кадра kind="hash-table".
// Самодостаточен (ADR-0002): все корзины с их цепочками копируются в каждый
// кадр, чтобы клиент рисовал кадр как есть, не накапливая состояние.
type HashTableSnapshot struct {
	BucketCount     int                  `json:"bucketCount"`     // число корзин (задаётся при создании)
	Buckets         [][]HashTableEntry   `json:"buckets"`         // содержимое каждой корзины: цепочка от головы
	HighlightBucket int                  `json:"highlightBucket"` // корзина, выбранная хешированием; -1 — шаг без выбора
	Highlights      []HashTableHighlight `json:"highlights"`      // подсветки затронутых живых записей
	// Надгробия шага: удалённые записи на бывших позициях; шагов удаления
	// нет — присылается пустой массив, в JSON он обязан быть `[]` (строгий
	// контракт фронтенда требует массив при парсинге снимка).
	Removed    []HashTableRemoval `json:"removed"`
	EntryCount int                `json:"entryCount"` // число записей во всей таблице
	Operation  string             `json:"operation"`  // имя операции шага ("put 6 20", "get 9")
}

// hashTableFrames — накапливает состояние между событиями и строит кадры.
// Реализует algorithms.Emitter[structures.HashEvent]: одно событие — один
// кадр (ADR-0002), включая выбор корзины и визит каждой записи цепочки.
// Содержимое корзин держится зеркалом из событий: рендерер не заглядывает
// во внутренности библиотечной таблицы.
type hashTableFrames struct {
	mirror          [][]HashTableEntry // содержимое корзин (первая запись — голова цепочки)
	entryCount      int                // число записей во всей таблице
	highlightBucket int                // корзина, выбранная на текущем шаге (-1 — нет)
	highlights      []HashTableHighlight
	removals        []HashTableRemoval // надгробия текущего шага (ставит HashRemove)
	operation       string             // имя текущей операции ("put 6 20", "get 9")
	operationKey    int                // ключ текущей операции (для пояснения выбора корзины)
	step            int
	emit            func(frame.Frame)
}

// Emit — одно событие хеш-таблицы → один кадр (ADR-0002).
func (r *hashTableFrames) Emit(ev structures.HashEvent) {
	switch e := ev.(type) {
	case structures.HashBucket:
		// Корзина выбрана, содержимое не изменилось; шаг начинается с
		// чистого листа — и по подсветкам, и по надгробиям.
		r.highlightBucket = e.Index
		r.highlights = []HashTableHighlight{}
		r.removals = nil
		r.frame(r.bucketExplanation(e))

	case structures.HashVisit:
		// Обход дошёл до записи цепочки; содержимое не изменилось.
		r.highlights = []HashTableHighlight{{Bucket: e.Bucket, Position: e.Position, Role: RoleVisited}}
		r.frame(fmt.Sprintf("%s: обход цепочки корзины %d: посетили запись на позиции %d (ключ %d, значение %d).",
			r.operation, e.Bucket, e.Position, e.Key, e.Value))

	case structures.HashInsert:
		// Новая запись встала во главу цепочки (позиция 0): зеркало корзины
		// можно дополнить prepend'ом.
		chain := r.mirror[e.Bucket]
		fresh := make([]HashTableEntry, 0, len(chain)+1)
		fresh = append(fresh, HashTableEntry{Key: e.Key, Value: e.Value})
		fresh = append(fresh, chain...)
		r.mirror[e.Bucket] = fresh
		r.entryCount++
		r.highlights = []HashTableHighlight{{Bucket: e.Bucket, Position: e.Position, Role: RoleAdded}}

		if len(chain) == 0 {
			r.frame(fmt.Sprintf("%s: корзина %d была пуста — запись (ключ %d, значение %d) стала первой в цепочке.",
				r.operation, e.Bucket, e.Key, e.Value))
		} else {
			r.frame(fmt.Sprintf("%s: коллизия — ключ %d попал в занятую корзину %d; запись (ключ %d, значение %d) встала во главу цепочки, остальные записи остались доступны.",
				r.operation, e.Key, e.Bucket, e.Key, e.Value))
		}

	case structures.HashUpdate:
		// Значение существующего ключа переписано; длина цепочки та же.
		r.mirror[e.Bucket][e.Position].Value = e.Value
		r.highlights = []HashTableHighlight{{Bucket: e.Bucket, Position: e.Position, Role: RoleUpdated}}
		r.frame(fmt.Sprintf("%s: ключ %d уже был в корзине %d (позиция %d) — переписали значение на %d; число записей таблицы не изменилось.",
			r.operation, e.Key, e.Bucket, e.Position, e.Value))

	case structures.HashRemove:
		// Запись удалена: в зеркале её уже нет, у записей после неё позиции
		// стали на одну раньше. Роль removed на живой записи указывала бы
		// на соседа, вставшего на её место, — удалённая пара живёт в кадре
		// надгробием на бывшей позиции (Removed), а не подсветкой.
		chain := r.mirror[e.Bucket]
		fresh := make([]HashTableEntry, 0, len(chain)-1)
		fresh = append(fresh, chain[:e.Position]...)
		fresh = append(fresh, chain[e.Position+1:]...)
		r.mirror[e.Bucket] = fresh
		r.entryCount--
		r.highlights = []HashTableHighlight{}
		r.removals = []HashTableRemoval{{Bucket: e.Bucket, Position: e.Position, Key: e.Key, Value: e.Value}}
		r.frame(fmt.Sprintf("%s: удалили запись (ключ %d, значение %d) из корзины %d (позиция %d) — связи перепрыгнули через удалённую, остальные записи цепочки остались доступны.",
			r.operation, e.Key, e.Value, e.Bucket, e.Position))

	case structures.HashFound:
		// Ключ найден на позиции e.Position — обход останавливается здесь.
		r.highlights = []HashTableHighlight{{Bucket: e.Bucket, Position: e.Position, Role: RoleFound}}
		r.frame(fmt.Sprintf("%s: ключ %d найден в корзине %d (запись на позиции %d): значение %d — до попадания пришлось обойти %d запис(ей) цепочки.",
			r.operation, e.Key, e.Bucket, e.Position, e.Value, e.Position+1))

	case structures.HashNotFound:
		// Цепочка пройдена до конца, ключа нет: успешный промах, не ошибка.
		r.highlights = []HashTableHighlight{}
		r.frame(fmt.Sprintf("%s: ключа %d среди записей корзины %d нет — цепочка пройдена до конца, это промах (не ошибка операции).",
			r.operation, e.Key, e.Bucket))
	}
}

// bucketExplanation — пояснение выбора корзины хешированием: расчёт ведётся
// от сырого остатка HashBucket.Raw — событие крутит его из библиотеки, и
// рендерер формулу остатка (и нормализацию отрицательного) не повторяет:
// она живёт в bucketOf в одном экземпляре. Здесь только текст объяснения
// по готовым числам: остаток % число-корзин, при отрицательном —
// прибавление числа корзин.
func (r *hashTableFrames) bucketExplanation(e structures.HashBucket) string {
	n := len(r.mirror)
	if e.Raw >= 0 {
		return fmt.Sprintf("%s: хешируем ключ %d: остаток %d %% %d = %d — открываем корзину %d.",
			r.operation, r.operationKey, r.operationKey, n, e.Raw, e.Index)
	}
	return fmt.Sprintf("%s: хешируем ключ %d: остаток %d %% %d = %d отрицательный — нормализуем: %d + %d = %d, открываем корзину %d.",
		r.operation, r.operationKey, r.operationKey, n, e.Raw, e.Raw, n, e.Index, e.Index)
}

// frame собирает самодостаточный кадр: копия всех корзин + подсветки.
// Каждая корзина копируется в новый слайс (в том числе пустая — пустой
// цепочка соответствует [], не null). Шаги нумеруются с 1: стартовый кадр,
// затем по кадру на каждое событие.
func (r *hashTableFrames) frame(explanation string) {
	if r.emit == nil {
		return
	}
	r.step++
	buckets := make([][]HashTableEntry, len(r.mirror))
	for i, chain := range r.mirror {
		buckets[i] = append([]HashTableEntry{}, chain...)
	}
	r.emit(frame.Frame{
		Step:        r.step,
		Kind:        frame.KindHashTable,
		Explanation: explanation,
		Data: HashTableSnapshot{
			BucketCount:     len(r.mirror),
			Buckets:         buckets,
			HighlightBucket: r.highlightBucket,
			Highlights:      append([]HashTableHighlight{}, r.highlights...),
			// removed всегда уходит как массив (`[]` на шаге без удаления,
			// не null): снятие nil в JSON null ломает строгий контракт
			// фронтенда на первом же кадре каждого прогона.
			Removed:    append([]HashTableRemoval{}, r.removals...),
			EntryCount: r.entryCount,
			Operation:  r.operation,
		},
	})
}
