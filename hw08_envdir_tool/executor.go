package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

// RunCmd runs a command + arguments (cmd) with environment variables from env.
func RunCmd(cmd []string, env Environment) (returnCode int) {
	if len(cmd) == 0 {
		return 1
	}

	command := exec.Command(cmd[0], cmd[1:]...) //nolint:gosec // the command is provided by the user by design
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Env = environmentVariables(env)

	err := command.Run()
	if err == nil {
		return 0
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}

	return 1
}

func environmentVariables(env Environment) []string {
	values := make(map[string]string)

	for _, variable := range os.Environ() {
		key, value, ok := strings.Cut(variable, "=")
		if ok {
			values[key] = value
		}
	}

	for key, value := range env {
		if strings.Contains(key, "=") {
			continue
		}
		if value.NeedRemove {
			delete(values, key)
			continue
		}
		values[key] = value.Value
	}

	result := make([]string, 0, len(values))
	for key, value := range values {
		result = append(result, key+"="+value)
	}

	return result
}
