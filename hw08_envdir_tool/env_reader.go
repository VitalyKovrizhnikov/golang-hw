package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Environment map[string]EnvValue

// EnvValue helps to distinguish between empty files and files with the first empty line.
type EnvValue struct {
	Value      string
	NeedRemove bool
}

// ReadDir reads a specified directory and returns map of env variables.
// Variables represented as files where filename is name of variable, file first line is a value.
func ReadDir(dir string) (Environment, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	env := make(Environment, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.ContainsRune(name, '=') {
			return nil, fmt.Errorf("invalid environment variable name %q", name)
		}

		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}

		env[name] = parseEnvValue(data)
	}

	return env, nil
}

func parseEnvValue(data []byte) EnvValue {
	if len(data) == 0 {
		return EnvValue{NeedRemove: true}
	}

	if index := bytes.IndexByte(data, '\n'); index >= 0 {
		data = data[:index]
	}
	data = bytes.ReplaceAll(data, []byte{0}, []byte{'\n'})

	return EnvValue{Value: strings.TrimRight(string(data), " \t")}
}
