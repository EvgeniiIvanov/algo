// Классический несжатый Trie для Unicode-слов: переходы по отдельным rune
// (одно Unicode scalar value, не байты), узел помечен флагом конца слова.
// Сравнение rune точное: без нормализации Unicode и без смены регистра.
//
// Все операции живут в одном месте с необязательным эмиттером: тихий режим —
// тот же обход с nil-эмиттером (ADR-0001), режим наблюдения стреляет теми же
// шагами в Emitter (TrieEvent, см. trie_events.go). Поведение и результаты
// у обоих режимов одинаковые.
package trees

import (
	"unicode/utf8"

	"github.com/EvgeniiIvanov/algo/algorithms"
)

// NodeTrie — узел Trie: переходы по рунам в map, isEnd — отметка конца слова.
// Терминальный узел не обязан быть листом: слово-префикс может лежать на пути
// к более длинным словам.
type NodeTrie struct {
	children map[rune]*NodeTrie
	isEnd    bool
}

// Trie — префиксное дерево с корнем-заглушкой и счётчиком разных слов.
type Trie struct {
	root *NodeTrie
	size int
}

// NewTrie создаёт пустой Trie.
func NewTrie() *Trie {
	return &Trie{root: &NodeTrie{children: make(map[rune]*NodeTrie)}}
}

// Len возвращает число разных сохранённых слов.
func (t *Trie) Len() int {
	return t.size
}

// Insert добавляет слово и сообщает, добавлено ли оно: повторная вставка
// идемпотентна — дерево и Len не меняются, результат false. Пустое слово
// и невалидный UTF-8 — ошибки до любых изменений и событий.
func (t *Trie) Insert(word string) (inserted bool, err error) {
	return t.insert(word, nil)
}

// Search ищет точное слово: true только если путь по всем рунам существует
// и конечный узел отмечен концом слова. Пустое слово — валидный запрос,
// всегда false (пустое слово не хранится).
func (t *Trie) Search(word string) (found bool, err error) {
	return t.search(word, nil)
}

// StartsWith проверяет существование пути префикса: идёт ли от корня путь
// по рунам префикса, независимо от того, образует ли он конец слова.
// Пустой префикс соответствует корню, поэтому StartsWith("") — всегда true,
// в том числе в пустом Trie.
func (t *Trie) StartsWith(prefix string) (found bool, err error) {
	return t.startsWith(prefix, nil)
}

// Delete удаляет слово и сообщает, удалено ли оно: промах — не ошибка,
// дерево не меняется, результат false. Удаляются только больше не нужные
// узлы (pruning): ветвления и сохранённые слова-префиксы остаются.
func (t *Trie) Delete(word string) (deleted bool, err error) {
	return t.delete(word, nil)
}

func (t *Trie) insert(word string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	if word == "" {
		return false, ErrInsertEmptyWord
	}
	if !utf8.ValidString(word) {
		return false, ErrInvalidUTF8
	}
	runes := []rune(word)
	node := t.root
	inserted := false
	for i, char := range runes {
		next, exists := node.children[char]
		state := EdgeReused
		if !exists {
			next = &NodeTrie{children: make(map[rune]*NodeTrie)}
			node.children[char] = next
			state = EdgeCreated
		}
		// На последней руне то же событие посещения несёт итоговое состояние
		// флага конца слова — но только когда оно меняется этой вставкой.
		var endFlag *bool
		if i == len(runes)-1 && !next.isEnd {
			next.isEnd = true
			t.size++
			inserted = true
			endFlag = boolPtr(true)
		}
		node = next
		emitTrie(em, TrieVisit{Rune: char, Edge: state, EndFlag: endFlag})
	}
	if inserted {
		emitTrie(em, TrieInserted{})
		return true, nil
	}
	emitTrie(em, TrieAlreadyExisted{})
	return false, nil
}

func (t *Trie) search(word string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	if !utf8.ValidString(word) {
		return false, ErrInvalidUTF8
	}
	node := t.root
	for _, char := range word {
		next, exists := node.children[char]
		if !exists {
			// Первая отсутствующая руна: узел не создаётся, обход прекращается.
			emitTrie(em, TrieVisit{Rune: char, Edge: EdgeMissing})
			emitTrie(em, TrieNotFound{})
			return false, nil
		}
		emitTrie(em, TrieVisit{Rune: char, Edge: EdgeReused})
		node = next
	}
	if node.isEnd {
		emitTrie(em, TrieFound{})
		return true, nil
	}
	emitTrie(em, TrieNotFound{})
	return false, nil
}

func (t *Trie) startsWith(prefix string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	if !utf8.ValidString(prefix) {
		return false, ErrInvalidUTF8
	}
	node := t.root
	for _, char := range prefix {
		next, exists := node.children[char]
		if !exists {
			emitTrie(em, TrieVisit{Rune: char, Edge: EdgeMissing})
			emitTrie(em, TriePrefixNotFound{})
			return false, nil
		}
		emitTrie(em, TrieVisit{Rune: char, Edge: EdgeReused})
		node = next
	}
	emitTrie(em, TriePrefixFound{})
	return true, nil
}

func (t *Trie) delete(word string, em algorithms.Emitter[TrieEvent]) (bool, error) {
	if word == "" {
		return false, ErrDeleteEmptyWord
	}
	if !utf8.ValidString(word) {
		return false, ErrInvalidUTF8
	}
	runes := []rune(word)
	node := t.root
	// Путь от корня по узлам слова — для прунинга снизу вверх.
	path := []*NodeTrie{node}
	deleted := false
	for i, char := range runes {
		next, exists := node.children[char]
		if !exists {
			emitTrie(em, TrieVisit{Rune: char, Edge: EdgeMissing})
			emitTrie(em, TrieNotDeleted{})
			return false, nil
		}
		// Последнее событие посещения удаляемого слова снимает флаг конца слова.
		if i == len(runes)-1 && next.isEnd {
			next.isEnd = false
			t.size--
			deleted = true
			endFlag := boolPtr(false)
			emitTrie(em, TrieVisit{Rune: char, Edge: EdgeReused, EndFlag: endFlag})
		} else {
			emitTrie(em, TrieVisit{Rune: char, Edge: EdgeReused})
		}
		node = next
		path = append(path, node)
	}
	if !deleted {
		// Путь по всем рунам существует, но конечный узел — не конец слова.
		emitTrie(em, TrieNotDeleted{})
		return false, nil
	}
	// Pruning: снизу вверх стираем узлы, которые больше не ведут ни к какому
	// концу слова; на первом живом узле (конец слова или есть потомки) стоп.
	for i := len(path) - 1; i > 0; i-- {
		cur := path[i]
		if cur.isEnd || len(cur.children) > 0 {
			break
		}
		emitTrie(em, TriePrune{Rune: runes[i-1], Depth: i})
		delete(path[i-1].children, runes[i-1])
	}
	emitTrie(em, TrieDeleted{})
	return true, nil
}

// boolPtr — вспомогательница для события TrieVisit.EndFlag (nil — флаг не менялся).
func boolPtr(v bool) *bool { return &v }
