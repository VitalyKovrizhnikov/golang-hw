package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: go-envdir <dir> <command> [args...]")
		os.Exit(2)
	}

	env, err := ReadDir(os.Args[1])
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	os.Exit(RunCmd(os.Args[2:], env))
}
