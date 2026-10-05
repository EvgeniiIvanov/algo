package quiz

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// Quiz Bank хранится рядом с пакетом и встраивается в бинарник
// (PLAN.md, решение 7: вопросы — embedded JSON по блокам).
//
//go:embed bank/bank.json
var bankJSON []byte

// EmbeddedBank разбирает встроенный банк и валидирует его перед каждым
// запуском: malformed JSON или битый формат вопросов видны сразу.
func EmbeddedBank() (*Bank, error) {
	bank, err := ParseBank(bankJSON)
	if err != nil {
		return nil, err
	}
	if err := bank.Validate(); err != nil {
		return nil, fmt.Errorf("quiz: встроенный банк не прошёл валидацию: %w", err)
	}
	return bank, nil
}

// ParseBank разбирает JSON в Bank.
func ParseBank(data []byte) (*Bank, error) {
	var bank Bank
	if err := json.Unmarshal(data, &bank); err != nil {
		return nil, fmt.Errorf("quiz: не удалось разобрать банк: %w", err)
	}
	return &bank, nil
}
