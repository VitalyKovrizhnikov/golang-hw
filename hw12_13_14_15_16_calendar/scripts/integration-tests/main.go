package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

func main() {
	testStatus := run(exec.Command(
		"docker",
		"compose",
		"-f", "deployments/docker-compose.yaml",
		"--profile", "integration",
		"run", "--rm", "-T", "--build", "integration-tests",
	))
	downStatus := run(exec.Command(
		"docker",
		"compose",
		"-f", "deployments/docker-compose.yaml",
		"--profile", "integration",
		"down", "-v", "--remove-orphans",
	))

	if testStatus != 0 {
		os.Exit(testStatus)
	}

	os.Exit(downStatus)
}

func run(cmd *exec.Cmd) int {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	fmt.Fprintf(os.Stderr, "%s: %v\n", cmd.String(), err)

	return 1
}
