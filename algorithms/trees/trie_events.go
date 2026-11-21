// События Trie-операций (ADR-0001): каждое событие — атомарный шаг операции,
// достаточный для восстановления пошагового обхода по rune, повторного
// использования или создания ребра, отметки конца слова и pruning — без
// знания внутренностей дерева (map переходов NodeTrie).
//
// Гранулярность: каждое rune-обращение даёт не более одного события
// посещения/проверки (TrieVisit) с состоянием ребра; не более одного события
// на каждый узел pruning (TriePrune); каждая валидная операция — ровно одно
// событие результата. Ошибочные вызовы (пустое слово, невалидный UTF-8)
// возвращают ошибку до любых событий.
package trees

import "github.com/EvgeniiIvanov/algo/algorithms"

// TrieEvent — маркер-интерфейс событий Trie (sealed): вне пакета свои
// события реализовать нельзя.
type TrieEvent interface{ isTrieEvent() }

// EdgeState — состояние ребра по руне в событии посещения/проверки:
// reused — существующее ребро, created — новое ребро, создаваемое вставкой,
// missing — попытка поиска/проверки префикса/удаления пройти отсутствующее
// ребро (узел не создаётся). Значения совпадают с формой edgeCheck кадров.
type EdgeState string

const (
	EdgeReused  EdgeState = "reused"
	EdgeCreated EdgeState = "created"
	EdgeMissing EdgeState = "missing"
)

// TrieVisit — одно rune-обращение операции: проверка ребра по руне и переход
// по нему (или отказ при missing). Rune — ровно одна руна (одно Unicode
// scalar value: символы вне BMP — одна руна, не два байта и не две половины).
// EndFlag — итоговое состояние флага конца слова: непуст только когда операция
// меняет флаг на последней руне слова — установка при вставке (true) и снятие
// при удалении (false); отдельного события отметки конца слова нет.
type TrieVisit struct {
	Rune    rune      // проверенная руна
	Edge    EdgeState // состояние ребра по этой руне
	EndFlag *bool     // итоговое состояние флага конца слова, когда оно меняется; nil — не менялся
}

// TriePrune — узел срезан при удалении слова: он больше не ведёт ни к какому
// концу слова. Rune — метка ребра, по которому узел был доступен от родителя
// (ребро удаляется), Depth — глубина срезанного узла (корень — 0).
type TriePrune struct {
	Rune  rune // метка ребра к срезанному узлу
	Depth int  // глубина срезанного узла (корень — 0)
}

// TrieInserted — слово добавлено: дерево выросло, Len увеличился.
type TrieInserted struct{}

// TrieAlreadyExisted — повторная вставка: слово уже сохранено, дерево и Len
// не изменились (идемпотентность), результат сообщает, что слово не добавлено.
type TrieAlreadyExisted struct{}

// TrieFound — точное слово найдено: путь по всем рунам существует и конечный
// узел отмечен концом слова.
type TrieFound struct{}

// TrieNotFound — промах точного поиска: отсутствующая руна на пути или путь
// есть, но конечный узел не отмечен концом слова. Штатный исход, не ошибка.
type TrieNotFound struct{}

// TriePrefixFound — путь префикса существует (в том числе пустой префикс —
// всегда: он соответствует корню).
type TriePrefixFound struct{}

// TriePrefixNotFound — пути префикса нет: первая отсутствующая руна.
type TriePrefixNotFound struct{}

// TrieDeleted — слово удалено: флаг конца слова снят, ставшие ненужными узлы
// срезаны (pruning), Len уменьшился.
type TrieDeleted struct{}

// TrieNotDeleted — промах удаления: отсутствующая руна на пути или конечный
// узел не конец слова. Дерево и Len не изменились, результат сообщает, что
// слово не найдено и не удалено; промах — не ошибка.
type TrieNotDeleted struct{}

func (TrieVisit) isTrieEvent()          {}
func (TriePrune) isTrieEvent()          {}
func (TrieInserted) isTrieEvent()       {}
func (TrieAlreadyExisted) isTrieEvent() {}
func (TrieFound) isTrieEvent()          {}
func (TrieNotFound) isTrieEvent()       {}
func (TriePrefixFound) isTrieEvent()    {}
func (TriePrefixNotFound) isTrieEvent() {}
func (TrieDeleted) isTrieEvent()        {}
func (TrieNotDeleted) isTrieEvent()     {}

// InsertWithEmitter — Insert в режиме наблюдения: события посещения на все
// руны слова (состояния reused/created), на последней руне — итоговое
// состояние флага конца слова, когда оно меняется, и результат
// (TrieInserted/TrieAlreadyExisted). Поведение и результат — те же, что у
// тихого Insert.
func (t *Trie) InsertWithEmitter(word string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	return t.insert(word, em)
}

// SearchWithEmitter — Search в режиме наблюдения: события посещения/проверки
// до первой отсутствующей руны включительно (или на все руны) и результат
// (TrieFound/TrieNotFound). Результат — тот же, что у тихого Search.
func (t *Trie) SearchWithEmitter(word string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	return t.search(word, em)
}

// StartsWithWithEmitter — StartsWith в режиме наблюдения: события
// посещения/проверки префикса и результат (TriePrefixFound/
// TriePrefixNotFound). Результат — тот же, что у тихого StartsWith.
func (t *Trie) StartsWithWithEmitter(prefix string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	return t.startsWith(prefix, em)
}

// DeleteWithEmitter — Delete в режиме наблюдения: события посещения/проверки,
// снятие флага конца слова в последнем событии посещения удаляемого слова,
// события pruning (TriePrune) и результат (TrieDeleted/TrieNotDeleted).
// Поведение и результат — те же, что у тихого Delete.
func (t *Trie) DeleteWithEmitter(word string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	return t.delete(word, em)
}

// emitTrie вызывает эмиттер, если он передан (nil-check — деталь реализации).
func emitTrie(em algorithms.Emitter[TrieEvent], event TrieEvent) {
	if em == nil {
		return
	}
	em.Emit(event)
}
