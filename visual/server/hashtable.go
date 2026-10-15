package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// Ограничения входа Run хеш-таблицы — как у линейных структур:
// не более 500 операций и не более 100 одновременно хранимых записей,
// чтобы поток кадров оставался обозримым (кадр даёт и выбор корзины, и
// каждый визит обхода цепочки). Это safety bound одного визуального Run:
// библиотечная structures.HashTable лимитов не имеет.
const (
	maxHashTableOps     = 500
	maxHashTableEntries = 100
)

// Число корзин ограничено сверху, чтобы bucket-массив оставался обозримым;
// ноль и отрицательные значения отвергает и библиотечный конструктор —
// здесь они ловятся раньше, понятным сообщением об ошибке входа.
const (
	minHashTableBuckets = 1
	maxHashTableBuckets = 100
)

// handleHashTable — Run сценария операций над хеш-таблицей: на выходе
// SSE-поток кадров до конца сценария. Тонкий обработчик по шаблону
// handleLinkedList: парсинг → валидация → runStreaming.
func handleHashTable(w http.ResponseWriter, r *http.Request) {
	nBuckets, ops, err := parseHashTableInput(r)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := checkHashTableScript(ops); err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Сценарий валидирован до Run: HashTableRun ошибки входа не даёт.
		_ = render.HashTableRun(nBuckets, ops, emit)
	})
}

// parseHashTableInput разбирает параметры buckets и script: buckets —
// число корзин (одна строка), script — сценарий операций по одной на строку:
// «put <key> <value>», «get <key>» или «delete <key>». Пустые строки
// пропускаются. Промахи get/delete не проверяются по содержимому таблицы:
// отсутствующий ключ — легальный шаг сценария, кадр его покажет.
func parseHashTableInput(r *http.Request) (int, []render.HashTableOp, error) {
	q := r.URL.Query()

	rawBuckets := q.Get("buckets")
	if strings.TrimSpace(rawBuckets) == "" {
		return 0, nil, fmt.Errorf("параметр buckets обязателен: число корзин одним целым, например buckets=5")
	}
	nBuckets, err := strconv.Atoi(rawBuckets)
	if err != nil {
		return 0, nil, fmt.Errorf("не удалось разобрать %q как целое число: buckets — число корзин одним целым", rawBuckets)
	}
	if nBuckets < minHashTableBuckets || nBuckets > maxHashTableBuckets {
		return 0, nil, fmt.Errorf("число корзин должно быть от %d до %d, получено %d",
			minHashTableBuckets, maxHashTableBuckets, nBuckets)
	}

	if !q.Has("script") {
		return 0, nil, fmt.Errorf("параметр script обязателен: операции по одной на строку, например «put 1 10», «get 1», «delete 6»")
	}

	var ops []render.HashTableOp
	for i, line := range strings.Split(q.Get("script"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue // пустая строка — не операция
		}
		switch fields[0] {
		case render.HashTableOpPut:
			if len(fields) != 3 {
				return 0, nil, fmt.Errorf("строка %d: put требует ровно два аргумента — ключ и значение, например «put 1 10»", i+1)
			}
			key, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: put K V — ключ и значение целыми", i+1, fields[1])
			}
			value, err := strconv.Atoi(fields[2])
			if err != nil {
				return 0, nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: put K V — ключ и значение целыми", i+1, fields[2])
			}
			ops = append(ops, render.HashTableOp{Kind: render.HashTableOpPut, Key: key, Value: value})
		case render.HashTableOpGet:
			if len(fields) != 2 {
				return 0, nil, fmt.Errorf("строка %d: get требует ровно один аргумент — ключ, например «get 1»", i+1)
			}
			key, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: get K — ключ одним целым", i+1, fields[1])
			}
			ops = append(ops, render.HashTableOp{Kind: render.HashTableOpGet, Key: key})
		case render.HashTableOpDelete:
			if len(fields) != 2 {
				return 0, nil, fmt.Errorf("строка %d: delete требует ровно один аргумент — ключ, например «delete 6»", i+1)
			}
			key, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, nil, fmt.Errorf("строка %d: не удалось разобрать %q как целое число: delete K — ключ одним целым", i+1, fields[1])
			}
			ops = append(ops, render.HashTableOp{Kind: render.HashTableOpDelete, Key: key})
		default:
			return 0, nil, fmt.Errorf("строка %d: неизвестная операция %q: ожидается «put K V», «get K» или «delete K»",
				i+1, fields[0])
		}
	}

	if len(ops) == 0 {
		return 0, nil, fmt.Errorf("сценарий пуст: нужна хотя бы одна операция (put K V, get K или delete K)")
	}
	if len(ops) > maxHashTableOps {
		return 0, nil, fmt.Errorf("сценарий слишком длинный: %d операций, максимум %d", len(ops), maxHashTableOps)
	}
	return nBuckets, ops, nil
}

// checkHashTableScript проверяет сценарий прогоном по числу одновременно
// хранимых записей: put нового ключа растит таблицу (лимит 100 записей),
// повторный put того же ключа обновляет значение и места не занимает,
// delete существующего ключа освобождает запись — эти инварианты повторяют
// поведение самой таблицы и не зависят от её внутренностей. get/delete
// отсутствующего ключа пропускаем всегда: промах — легальный шаг сценария.
func checkHashTableScript(ops []render.HashTableOp) error {
	keys := make(map[int]struct{}) // ключи, которые будут храниться к началу операции
	for i, op := range ops {
		switch op.Kind {
		case render.HashTableOpPut:
			if _, exists := keys[op.Key]; !exists {
				if len(keys)+1 > maxHashTableEntries {
					return fmt.Errorf("операция №%d: в таблице будет %d записей — максимум %d; сократите сценарий",
						i+1, len(keys)+1, maxHashTableEntries)
				}
				keys[op.Key] = struct{}{}
			}
		case render.HashTableOpDelete:
			delete(keys, op.Key) // промах тоже нормален: нечего освобождать
		case render.HashTableOpGet:
			// Промах — не ошибка входа; визиты обхода попадут в кадры.
		}
	}
	return nil
}
