package main

import (
	"fmt"
	"os"

	"github.com/seanhooo/gitid/internal/app"
)

func main() {
	program, err := app.New()
	if err == nil {
		err = program.Run(os.Args[1:])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, app.Red("gitid: "+err.Error(), os.Stderr))
		os.Exit(1)
	}
}
