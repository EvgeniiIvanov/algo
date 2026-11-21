package trees

import "errors"

var (
	ErrInsertEmptyWord = errors.New("insert empty word")
	ErrDeleteEmptyWord = errors.New("delete empty word")
	ErrInvalidUTF8     = errors.New("word is not valid UTF-8")
	// ErrUninitializedTrie — операция над неинициализированным Trie:
	// nil-приёмник или нулевое значение Trie{} без корня (создавайте Trie
	// через NewTrie). Возвращается до проверки входа и до любых событий.
	ErrUninitializedTrie = errors.New("trie is not initialized")
)
