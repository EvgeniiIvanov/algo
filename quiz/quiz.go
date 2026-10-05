package quiz

import (
	"fmt"
	"strings"
)

// Количество вариантов ответа — по формату проекта: 4 на каждый вопрос.
const optionsCount = 4

// Question — вопрос квиза: текст, 4 варианта, ровно один правильный
// (индекс Correct), пояснение. Привязка к модулю задаётся вложенностью
// в Module (см. GLOSSARY.md «Quiz Question»).
type Question struct {
	Text        string   `json:"text"`
	Options     []string `json:"options"`
	Correct     int      `json:"correct"`
	Explanation string   `json:"explanation"`
}

// Module — учебный модуль, к которому привязаны вопросы квиза.
type Module struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Questions []Question `json:"questions"`
}

// Block — блок программы: группа модулей (Блок 1 «Фундамент» и т. д.).
type Block struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Modules []Module `json:"modules"`
}

// Bank — Quiz Bank: вопросы, сгруппированные по блокам и модулям.
type Bank struct {
	Blocks []Block `json:"blocks"`
}

// questionCount считает суммарное число вопросов в блоке.
func (b *Block) questionCount() int {
	n := 0
	for i := range b.Modules {
		n += len(b.Modules[i].Questions)
	}
	return n
}

// Validate проверяет банк целиком: непустые id и названия, привязку
// вопросов к модулям и формат каждого вопроса (4 варианта, ровно один
// правильный, пояснение). Вызывается движком перед каждым запуском.
func (b *Bank) Validate() error {
	if len(b.Blocks) == 0 {
		return fmt.Errorf("в банке нет блоков")
	}
	blockIDs := map[string]bool{}
	for _, blk := range b.Blocks {
		if strings.TrimSpace(blk.ID) == "" {
			return fmt.Errorf("блок без id")
		}
		if strings.TrimSpace(blk.Title) == "" {
			return fmt.Errorf("блок %q без названия", blk.ID)
		}
		if blockIDs[blk.ID] {
			return fmt.Errorf("дубликат блока %q", blk.ID)
		}
		blockIDs[blk.ID] = true
		if len(blk.Modules) == 0 {
			return fmt.Errorf("блок %q без модулей", blk.ID)
		}
		moduleIDs := map[string]bool{}
		for _, m := range blk.Modules {
			if strings.TrimSpace(m.ID) == "" {
				return fmt.Errorf("блок %q: модуль без id (привязка к модулю обязательна)", blk.ID)
			}
			if moduleIDs[m.ID] {
				return fmt.Errorf("блок %q: дубликат модуля %q", blk.ID, m.ID)
			}
			moduleIDs[m.ID] = true
			if len(m.Questions) == 0 {
				return fmt.Errorf("модуль %q без вопросов", m.ID)
			}
			for i := range m.Questions {
				if err := m.Questions[i].validate(); err != nil {
					return fmt.Errorf("модуль %q, вопрос %d: %w", m.ID, i+1, err)
				}
			}
		}
	}
	return nil
}

// validate проверяет формат одного вопроса: текст, 4 варианта,
// ровно один правильный, пояснение.
func (q Question) validate() error {
	if strings.TrimSpace(q.Text) == "" {
		return fmt.Errorf("пустой текст вопроса")
	}
	if len(q.Options) != optionsCount {
		return fmt.Errorf("вариантов %d, нужно ровно %d", len(q.Options), optionsCount)
	}
	for _, opt := range q.Options {
		if strings.TrimSpace(opt) == "" {
			return fmt.Errorf("пустой вариант ответа")
		}
	}
	if q.Correct < 0 || q.Correct >= len(q.Options) {
		return fmt.Errorf("правильный ответ (индекс %d) вне диапазона вариантов", q.Correct)
	}
	if strings.TrimSpace(q.Explanation) == "" {
		return fmt.Errorf("нет пояснения к вопросу")
	}
	return nil
}
