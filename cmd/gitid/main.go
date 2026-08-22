package main

import (
	"fmt"
	"os"

	"github.com/seanho/gitid/internal/app"
)

func main() {
	program, err := app.New()
	if err == nil {
		err = program.Run(os.Args[1:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gitid:", err)
		os.Exit(1)
	}
}
