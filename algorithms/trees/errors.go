package trees

import "errors"

var (
	ErrInsertEmptyWord = errors.New("insert empty word")
	ErrDeleteEmptyWord = errors.New("delete empty word")
	ErrInvalidUTF8     = errors.New("word is not valid UTF-8")
)
