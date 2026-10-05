package main

import (
	"fmt"
	"os"

	"github.com/EvgeniiIvanov/algo/quiz"
)

func main() {
	bank, err := quiz.EmbeddedBank()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки банка вопросов: %v\n", err)
		os.Exit(1)
	}
	if err := quiz.Run(os.Stdin, os.Stdout, bank); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка квиза: %v\n", err)
		os.Exit(1)
	}
}
