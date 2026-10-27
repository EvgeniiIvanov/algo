// Обработчик Run обхода бинарного дерева: вход — вложенный JSON дерева
// (`tree`) и порядок (`order`). Тонкий обработчик по шаблону handleQueue:
// парсинг → валидация → runStreaming. Невалидный JSON, дерево больше
// лимита и неизвестный порядок отвергаются до SSE (400 вместо кадров).
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/EvgeniiIvanov/algo/algorithms/trees"
	"github.com/EvgeniiIvanov/algo/visual/frame"
	"github.com/EvgeniiIvanov/algo/visual/render"
)

// maxTreeNodes — лимит визуального Run: дерево должно оставаться
// обсервимым, поток кадров — разумного размера. Лимит только для
// визуализации: библиотечные обходы им не ограничены.
const maxTreeNodes = 100

// handleTreeTraversal — Run обходов in/pre/post-order: на выходе
// SSE-поток кадров kind="tree" до конца обхода (исходный кадр + кадр
// на каждое посещение). Обход по уровням — отдельный маршрут (тикет 32).
func handleTreeTraversal(w http.ResponseWriter, r *http.Request) {
	root, err := parseTreeJSON(r)
	if err != nil {
		writeError(w, err)
		return
	}
	order, err := parseTreeOrder(r.URL.Query().Get("order"))
	if err != nil {
		writeError(w, err)
		return
	}

	runStreaming(w, r, func(emit func(frame.Frame)) {
		// Вход валидирован до Run: порядок известен, ошибка конструктора
		// здесь недостижима.
		_ = render.TreeTraversalRun(root, order, emit)
	})
}

// treeOrders — множество валидных порядков для проверки входа.
var treeOrders = map[string]bool{
	render.TreeOrderInOrder:   true,
	render.TreeOrderPreOrder:  true,
	render.TreeOrderPostOrder: true,
}

// parseTreeJSON разбирает параметр tree — вложенный JSON дерева
// (value, left, right). Отсутствующий или null потомок — пустая ветвь,
// верхний null — пустое дерево. Numbers разбираются как json.Number
// (UseNumber): целое значение узла проверяется точно по литералу,
// без float64-округления. Валидирует значения и лимит узлов.
func parseTreeJSON(r *http.Request) (*trees.Node, error) {
	raw := r.URL.Query().Get("tree")
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("параметр tree обязателен: вложенный JSON дерева, например {\"value\":1,\"left\":{\"value\":2}}")
	}

	var parsed any
	dec := json.NewDecoder(strings.NewReader(raw))
	dec.UseNumber() // числа остаются литералами: точная проверка целых
	if err := dec.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("не удалось разобрать JSON дерева: %v", err)
	}
	// Строго одно JSON-значение: после дерева — только конец ввода
	// (пробелы допустимы). Повторный Decode возвращает io.EOF на чистом
	// хвосте; лишнее значение или мусор — ошибка, а не молчаливое игнор.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, fmt.Errorf("не удалось разобрать JSON дерева: %v", err)
		}
		return nil, fmt.Errorf("лишнее содержимое после JSON дерева: ожидается строго один JSON-объект")
	}

	var root *trees.Node
	count := 0
	if err := buildTree(parsed, &root, &count, ""); err != nil {
		return nil, err
	}
	return root, nil
}

// buildTree — рекурсивная проверка parsed как узла дерева: map с целым
// value и опциональными потомками; null — это nil-узел. context — сегмент
// сообщения об ошибке с путём узла ("": корень, дальше "(путь L)" и т.д.).
// count накапливает число узлов.
func buildTree(parsed any, dest **trees.Node, count *int, context string) error {
	if parsed == nil {
		*dest = nil // верхний null или null-потомок — пустая ветвь
		return nil
	}
	obj, ok := parsed.(map[string]any)
	if !ok {
		return fmt.Errorf("узел дерева%s должен быть объектом {value, left, right} или null", treeContext(context))
	}

	value, err := treeValue(obj["value"])
	if err != nil {
		return fmt.Errorf("значение узла дерева%s: %v", treeContext(context), err)
	}
	node := trees.NewNode(value)
	*count++
	if *count > maxTreeNodes {
		return fmt.Errorf("дерево слишком большое: разрешено не более %d узлов; сократите вход", maxTreeNodes)
	}
	*dest = node
	if err := buildTree(obj["left"], &node.Left, count, context+"L"); err != nil {
		return err
	}
	if err := buildTree(obj["right"], &node.Right, count, context+"R"); err != nil {
		return err
	}
	return nil
}

// maxNumberLiteral — верхняя граница длины JSON-числа, которое рассматриваем:
// ни один литерал, помещающийся в int64, не требует и десятой доли этого
// размера (int64 — не более 19 значащих цифр; длиннее — только копейки
// с ведущими нулями или дробные хвосты). Граница отсекает огромные литералы
// до любых вычислений — раньше, чем что-то материализуется в памяти.
const maxNumberLiteral = 4096

// errNotInteger — ошибка «значение не целое»: формулировка повторяется
// в нескольких ветках treeValue, поэтому собрана в одном месте, чтобы
// сообщения не разъезжались.
func errNotInteger(lit string) error {
	return fmt.Errorf("значение %s не целое: value — целое число", lit)
}

// errOutOfInt64 — ошибка «значение за пределами целого»: единая
// формулировка для всех веток treeValue.
func errOutOfInt64(lit string) error {
	return fmt.Errorf("значение %s за пределами целого (int64): value — целое число", lit)
}

// treeValue проверяет значение узла: JSON-число, целое и представимое
// в int64. Проверка точная — по литералу (декодер с UseNumber сохраняет
// исходную запись числа), без float64 и без big.Float: на конечной
// точности длинные дроби округлялись бы (например к 1), а разбор огромной
// экспоненты через big.Float материализовал бы гигантский big.Int.
func treeValue(v any) (int, error) {
	num, ok := v.(json.Number)
	if !ok {
		return 0, fmt.Errorf("поле value обязательно — целое число")
	}
	lit := num.String()
	if len(lit) > maxNumberLiteral {
		return 0, fmt.Errorf("значение %s… слишком длинное (лимит %d символов): value — целое число", lit[:16], maxNumberLiteral)
	}

	// Разбор литерала по структуре JSON-числа (саму грамматику уже
	// гарантировал декодер): [-] цифры [. цифры] [eE [+-] цифры].
	neg := strings.HasPrefix(lit, "-")
	body := strings.TrimLeft(lit, "-+")
	mant, es := body, ""
	if i := strings.IndexAny(body, "eE"); i >= 0 {
		mant, es = body[:i], body[i+1:]
	}
	intPart, frac := mant, ""
	if i := strings.IndexByte(mant, '.'); i >= 0 {
		intPart, frac = mant[:i], mant[i+1:]
	}
	// Значащие цифры (ведущие нули не считаются): "0", "0.0" — нуль
	// при любом показателе, поэтому нуль-проверка до разбора экспоненты:
	// «0e99999999999999999999» — это нуль, а не литерал за пределами int.
	// Значение = 0 × 10^n = 0, никакая арифметика показателя не нужна.
	digits := strings.TrimLeft(intPart+frac, "0")
	if digits == "" {
		return 0, nil
	}

	var err error
	exp := 0
	if es != "" {
		exp, err = strconv.Atoi(es)
		if err != nil {
			// Показатель не помещается в int (грамматику давно гарантировал
			// декодер, числовой формат здесь единственная причина отказа):
			// переполнение вверх — значение заведомо вне int64; переполнение
			// вниз — дробный хвост с астрономическим числом нулей после
			// запятой, замыкающими нулями значащих цифр не покрыть.
			if strings.HasPrefix(es, "-") {
				return 0, errNotInteger(lit)
			}
			return 0, errOutOfInt64(lit)
		}
		// Переполнобезопасные границы показателя — с учётом len(frac) и
		// числа значащих цифр (обе длины — это maxNumberLiteral в худшем
		// случае, так что вся последующая арифметика ходит в диапазоне
		// ±(2·maxNumberLiteral + 19) и не переполняет int):
		//
		//   Сверху: exp > 19+len(frac) даёт E := exp-len(frac) > 19 — после
		//   домножения значащих цифр их заведомо больше 19, то есть значение
		//   ≥ 10^19 — вне int64. Компенсированная запись («0.<тысячи
		//   нулей>1e4097») проходит ниже: экспонента до 19+len(frac)
		//   допустима, даже если exp больше maxNumberLiteral.
		//
		//   Снизу: exp < -(len(frac)+len(digits)) даёт k := len(frac)-exp >
		//   len(digits) ≥ t (замыкающие нули), то есть дробная часть целой
		//   быть не может в принципе.
		//
		// Обе границы сравнивают exp без вычитаний, поэтому int не
		// переполняется раньше времени (Atoi принимает MaxInt/MinInt).
		if exp > 19+len(frac) {
			return 0, errOutOfInt64(lit)
		}
		if exp < -(len(frac) + len(digits)) {
			return 0, errNotInteger(lit)
		}
	}
	t := 0 // замыкающие нули значащих цифр
	for t < len(digits) && digits[len(digits)-1-t] == '0' {
		t++
	}
	// E — порядок при значащих цифрах: значение = digits × 10^E.
	// Отрицательный E — дробная часть: она должна полностью покрываться
	// замыкающими нулями, иначе значение не целое (сравнение по счёту,
	// без материализации длинных дробей).
	E := exp - len(frac)
	if E < 0 {
		k := -E
		if k > t {
			return 0, errNotInteger(lit)
		}
		digits = digits[:len(digits)-k]
		E = 0
	}
	// Неотрицательный E — домножение на степени десяти: 20 значащих
	// разрядов и больше означают ≥ 10^19, что заведомо вне int64
	// (в int64 помещается не более 19 цифр). Граница по счёту цифр —
	// без порождения длинного числа в памяти.
	if len(digits)+E > 19 {
		return 0, errOutOfInt64(lit)
	}
	digits += strings.Repeat("0", E)
	if len(digits) > 19 || (len(digits) == 19 && (neg && digits > "9223372036854775808" || !neg && digits > "9223372036854775807")) {
		return 0, errOutOfInt64(lit)
	}
	// Magnitude гарантированно ≤ 2^63 (проверено сравнением), поэтому
	// разбор беззнаковый: без него максимум int64 по модулю (2^63 для
	// минимального значения) не влезает в ParseInt как положительное число.
	u, err2 := strconv.ParseUint(digits, 10, 64)
	if err2 != nil {
		return 0, errOutOfInt64(lit)
	}
	n := int64(u)
	if neg {
		n = -n
	}
	// int64 → int: проект поддерживает 64-битные платформы — CI гоняет
	// тесты на ubuntu-latest (64-битный GOARCH), локальная разработка
	// тоже 64-битная, и только там поведение гарантируется. На 32-битной
	// сборке int — 32 бита, диапазон сузится; совместимость с ней не
	// заявлена и не поддерживается.
	return int(n), nil
}

// treeContext — сегмент сообщения об ошибке: путь узла в живой форме,
// как у клиента: корень — «(корень)», дальше — «(путь L)», «(путь LR)».
func treeContext(p string) string {
	if p == "" {
		return "(корень)"
	}
	return "(путь " + p + ")"
}

// parseTreeOrder разбирает параметр order — один из трёх порядков обхода.
func parseTreeOrder(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("параметр order обязателен: порядок обхода — in-order, pre-order или post-order")
	}
	if !treeOrders[raw] {
		return "", fmt.Errorf("неизвестный порядок обхода %q: ожидается in-order, pre-order или post-order", raw)
	}
	return raw, nil
}
