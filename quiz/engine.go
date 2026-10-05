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
// Ввод и вывод подменяемы, поэтому движок тестируется со скриптованным
// вводом; история попыток не хранится.
func Run(in io.Reader, out io.Writer, bank *Bank) error {
	if err := bank.Validate(); err != nil {
		return fmt.Errorf("quiz: банк вопросов не прошёл валидацию: %w", err)
	}

	s := &session{out: out, blocks: map[string]*blockResult{}}
	fmt.Fprintln(s.out, "Квиз по алгоритмам. «q» — выйти.")

	sc := bufio.NewScanner(in)
	for {
		blk, ok := chooseBlock(sc, s.out, bank)
		if !ok {
			break // выход или конец ввода
		}
		s.runBlock(sc, blk)
	}
	s.printSummary()
	return nil
}

// chooseBlock показывает список блоков и читает выбор (номер или «q»).
func chooseBlock(sc *bufio.Scanner, out io.Writer, bank *Bank) (*Block, bool) {
	for {
		fmt.Fprintln(out, "\nБлоки программы:")
		for i := range bank.Blocks {
			b := &bank.Blocks[i]
			fmt.Fprintf(out, "  %d) %s — вопросов: %d\n", i+1, b.Title, b.questionCount())
		}
		fmt.Fprint(out, "Выберите блок (номер, q — выход): ")

		line, ok := readLine(sc)
		if !ok {
			return nil, false
		}
		if isQuit(line) {
			return nil, false
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > len(bank.Blocks) {
			fmt.Fprintln(out, "Нет такого блока, попробуйте ещё раз.")
			continue
		}
		return &bank.Blocks[n-1], true
	}
}

// askAnswer читает номер варианта (1–4); ok=false — выход или конец ввода.
func askAnswer(sc *bufio.Scanner, out io.Writer) (int, bool) {
	for {
		line, ok := readLine(sc)
		if !ok {
			return 0, false
		}
		if isQuit(line) {
			return 0, false
		}
		n, err := strconv.Atoi(line)
		if err != nil || n < 1 || n > optionsCount {
			fmt.Fprint(out, "Введите число от 1 до 4: ")
			continue
		}
		return n - 1, true
	}
}

// session накапливает результаты текущей сессии по затронутым блокам.
type session struct {
	out    io.Writer
	order  []*blockResult          // в порядке выбора
	blocks map[string]*blockResult // по id блока
}

// blockResult — результат по одному блоку и его модулям.
type blockResult struct {
	title   string
	order   []*moduleResult
	modules map[string]*moduleResult // по id модуля
}

// moduleResult — результат по одному модулю.
type moduleResult struct {
	title    string
	correct  int
	answered int
}

// runBlock проводит опрос по блоку: модули по порядку, вопросы по порядку,
// после каждого ответа — пояснение (почему правильно/неправильно).
func (s *session) runBlock(sc *bufio.Scanner, b *Block) {
	res, seen := s.blocks[b.ID]
	if !seen {
		res = &blockResult{title: b.Title, modules: map[string]*moduleResult{}}
		s.blocks[b.ID] = res
		s.order = append(s.order, res)
	}

	fmt.Fprintf(s.out, "\n=== %s ===\n", b.Title)
	for mi := range b.Modules {
		m := &b.Modules[mi]
		mr, seen := res.modules[m.ID]
		if !seen {
			mr = &moduleResult{title: m.Title}
			res.modules[m.ID] = mr
			res.order = append(res.order, mr)
		}

		fmt.Fprintf(s.out, "\nМодуль: %s\n", m.Title)
		for qi := range m.Questions {
			q := &m.Questions[qi]
			fmt.Fprintf(s.out, "\nВопрос %d из %d: %s\n", qi+1, len(m.Questions), q.Text)
			for j, opt := range q.Options {
				fmt.Fprintf(s.out, "  %d) %s\n", j+1, opt)
			}
			fmt.Fprint(s.out, "Ваш ответ (1–4, q — выйти): ")

			ans, ok := askAnswer(sc, s.out)
			if !ok {
				fmt.Fprintln(s.out) // опрос прерван — учитываем только отвеченное
				return
			}
			mr.answered++
			if ans == q.Correct {
				mr.correct++
				fmt.Fprintln(s.out, "✔ Правильно.")
			} else {
				fmt.Fprintf(s.out, "✘ Неправильно. Правильный ответ: %d) %s\n", q.Correct+1, q.Options[q.Correct])
			}
			fmt.Fprintf(s.out, "Пояснение: %s\n", q.Explanation)
		}
	}
}

// printSummary печатает сводку: правильные/всего по каждому затронутому блоку.
func (s *session) printSummary() {
	fmt.Fprintln(s.out, "\n=== Сводка ===")
	if len(s.order) == 0 {
		fmt.Fprintln(s.out, "Ни один блок не пройден.")
		return
	}
	for _, res := range s.order {
		fmt.Fprintf(s.out, "%s: %d/%d\n", res.title, res.correct(), res.answered())
		for _, mr := range res.order {
			fmt.Fprintf(s.out, "  %s: %d/%d\n", mr.title, mr.correct, mr.answered)
		}
	}
}

func (b *blockResult) correct() int {
	n := 0
	for _, mr := range b.order {
		n += mr.correct
	}
	return n
}

func (b *blockResult) answered() int {
	n := 0
	for _, mr := range b.order {
		n += mr.answered
	}
	return n
}

// readLine читает одну строку ввода без завершающих пробелов;
// ok=false — ввод закончился (EOF).
func readLine(sc *bufio.Scanner) (string, bool) {
	if !sc.Scan() {
		return "", false
	}
	return strings.TrimSpace(sc.Text()), true
}

// isQuit распознаёт команду выхода.
func isQuit(line string) bool {
	return strings.EqualFold(line, "q")
}
