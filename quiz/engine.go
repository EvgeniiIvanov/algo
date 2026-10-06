package quiz

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Run — движок квиза (шов 3). Читает ввод построчно, пишет в out:
// выбор блока → вопросы блока по порядку → пояснение после каждого
// ответа → в конце сводка по каждому затронутому блоку. «q» в любой
// момент — выход (текущий блок считается пройденным до места выхода).
// Повторный проход блока пересчитывает его счёт заново, а не копит
// поверх прошлого. Ввод и вывод подменяемы, поэтому движок тестируется
// со скриптованным вводом; история попыток не хранится.
func Run(in io.Reader, out io.Writer, bank *Bank) error {
	if err := bank.Validate(); err != nil {
		return fmt.Errorf("quiz: банк вопросов не прошёл валидацию: %w", err)
	}

	c := &console{in: bufio.NewScanner(in), out: out}
	c.println("Квиз по алгоритмам. «q» — выйти.")

	s := &session{c: c, blocks: map[string]*blockResult{}}
	for {
		blk, ok := chooseBlock(c, bank)
		if !ok {
			break // выход или конец ввода
		}
		s.runBlock(blk)
	}
	s.printSummary()
	return nil
}

// console — ввод/вывод движка в одном месте: сканер ввода + writer вывода.
type console struct {
	in  *bufio.Scanner
	out io.Writer
}

// Печатаем в stdout CLI: ошибки записи в терминал намеренно игнорируем.
func (c *console) printf(format string, args ...any) { _, _ = fmt.Fprintf(c.out, format, args...) }
func (c *console) println(args ...any)               { _, _ = fmt.Fprintln(c.out, args...) }

// line читает одну строку ввода без завершающих пробелов;
// ok=false — ввод закончился (EOF).
func (c *console) line() (string, bool) {
	if !c.in.Scan() {
		return "", false
	}
	return strings.TrimSpace(c.in.Text()), true
}

// chooseBlock показывает список блоков и читает выбор (номер или «q»).
func chooseBlock(c *console, bank *Bank) (*Block, bool) {
	for {
		c.printf("\nБлоки программы:\n")
		for i := range bank.Blocks {
			b := &bank.Blocks[i]
			c.printf("  %d) %s — вопросов: %d\n", i+1, b.Title, b.questionCount())
		}
		c.printf("Выберите блок (номер, q — выход): ")

		line, ok := c.line()
		if !ok {
			return nil, false
		}
		if isQuit(line) {
			return nil, false
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(bank.Blocks) {
			c.println("Нет такого блока, попробуйте ещё раз.")
			continue
		}
		return &bank.Blocks[n-1], true
	}
}

// askAnswer читает номер варианта (1–4); ok=false — выход или конец ввода.
func askAnswer(c *console) (int, bool) {
	for {
		line, ok := c.line()
		if !ok {
			return 0, false
		}
		if isQuit(line) {
			return 0, false
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > optionsCount {
			c.printf("Введите число от 1 до 4: ")
			continue
		}
		return n - 1, true
	}
}

// score — счёт по модулю или блоку: отвечено вопросов и сколько из них
// правильно.
type score struct {
	correct  int
	answered int
}

func (s score) add(o score) score {
	s.correct += o.correct
	s.answered += o.answered
	return s
}

// session накапливает результаты текущей сессии по затронутым блокам.
type session struct {
	c      *console
	order  []*blockResult          // в порядке выбора
	blocks map[string]*blockResult // по id блока
}

// blockResult — результат по одному блоку и его модулям.
type blockResult struct {
	title   string
	order   []*moduleResult
	modules map[string]*moduleResult // по id модуля
}

// total суммирует счёт всех модулей блока.
func (b *blockResult) total() score {
	var t score
	for _, mr := range b.order {
		t = t.add(mr.score)
	}
	return t
}

// reset обнуляет счёт блока перед повторным проходом.
func (b *blockResult) reset() {
	for _, mr := range b.order {
		mr.score = score{}
	}
}

// moduleResult — результат по одному модулю.
type moduleResult struct {
	title string
	score
}

// runBlock проводит опрос по блоку: модули по порядку, вопросы по порядку,
// после каждого ответа — пояснение (почему правильно/неправильно).
func (s *session) runBlock(b *Block) {
	res, seen := s.blocks[b.ID]
	if !seen {
		res = &blockResult{title: b.Title, modules: map[string]*moduleResult{}}
		s.blocks[b.ID] = res
		s.order = append(s.order, res)
	} else {
		res.reset() // повторный проход считается заново
	}

	c := s.c
	c.printf("\n=== %s ===\n", b.Title)
	for mi := range b.Modules {
		m := &b.Modules[mi]
		mr, seen := res.modules[m.ID]
		if !seen {
			mr = &moduleResult{title: m.Title}
			res.modules[m.ID] = mr
			res.order = append(res.order, mr)
		}

		c.printf("\nМодуль: %s\n", m.Title)
		for qi := range m.Questions {
			q := &m.Questions[qi]
			c.printf("\nВопрос %d из %d: %s\n", qi+1, len(m.Questions), q.Text)
			for j, opt := range q.Options {
				c.printf("  %d) %s\n", j+1, opt)
			}
			c.printf("Ваш ответ (1–4, q — выйти): ")

			ans, ok := askAnswer(c)
			if !ok {
				c.println() // опрос прерван — учитываем только отвеченное
				return
			}
			mr.answered++
			if ans == q.Correct {
				mr.correct++
				c.println("✔ Правильно.")
			} else {
				c.printf("✘ Неправильно. Правильный ответ: %d) %s\n", q.Correct+1, q.Options[q.Correct])
			}
			c.printf("Пояснение: %s\n", q.Explanation)
		}
	}
}

// printSummary печатает сводку: правильные/всего по каждому затронутому блоку.
func (s *session) printSummary() {
	c := s.c
	c.println("\n=== Сводка ===")
	if len(s.order) == 0 {
		c.println("Ни один блок не пройден.")
		return
	}
	for _, res := range s.order {
		t := res.total()
		c.printf("%s: %d/%d\n", res.title, t.correct, t.answered)
		for _, mr := range res.order {
			c.printf("  %s: %d/%d\n", mr.title, mr.correct, mr.answered)
		}
	}
}

// isQuit распознаёт команду выхода.
func isQuit(line string) bool {
	return strings.EqualFold(line, "q")
}
