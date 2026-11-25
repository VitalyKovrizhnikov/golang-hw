package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
)

func helperCommand() []string {
	return []string{os.Args[0], "-test.run=TestHelperProcess"}
}

func runWithCapturedStdout(t *testing.T, cmd []string, env Environment) (int, string) {
	t.Helper()

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.Stdout = oldStdout
	}()

	os.Stdout = writer
	code := RunCmd(cmd, env)

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = oldStdout

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}

	return code, string(output)
}

func TestRunCmdExitCode(t *testing.T) {
	env := Environment{
		"GO_WANT_HELPER_PROCESS": {Value: "1"},
		"HELPER_EXIT_CODE":       {Value: "42"},
	}

	if code := RunCmd(helperCommand(), env); code != 42 {
		t.Fatalf("RunCmd() = %d, want 42", code)
	}
}

func TestRunCmdEnvironment(t *testing.T) {
	t.Setenv("HELPER_KEEP", "keep")
	t.Setenv("HELPER_OVERRIDE", "old")
	t.Setenv("HELPER_REMOVE", "remove")

	env := Environment{
		"GO_WANT_HELPER_PROCESS": {Value: "1"},
		"HELPER_ENV_NAMES":       {Value: "HELPER_KEEP,HELPER_OVERRIDE,HELPER_REMOVE"},
		"HELPER_OVERRIDE":        {Value: "new"},
		"HELPER_REMOVE":          {NeedRemove: true},
	}

	code, output := runWithCapturedStdout(t, helperCommand(), env)
	if code != 0 {
		t.Fatalf("RunCmd() = %d, want 0", code)
	}

	expected := "HELPER_KEEP=keep\nHELPER_OVERRIDE=new\nHELPER_REMOVE=\n"
	if output != expected {
		t.Fatalf("output = %q, want %q", output, expected)
	}
}

func TestRunCmdNotFound(t *testing.T) {
	if code := RunCmd([]string{"envdir-command-that-does-not-exist"}, Environment{}); code != 1 {
		t.Fatalf("RunCmd() = %d, want 1", code)
	}
}

func TestRunCmdEmptyCommand(t *testing.T) {
	if code := RunCmd(nil, Environment{}); code != 1 {
		t.Fatalf("RunCmd() = %d, want 1", code)
	}
}

func TestHelperProcess(_ *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	if exitCode := os.Getenv("HELPER_EXIT_CODE"); exitCode != "" {
		code, err := strconv.Atoi(exitCode)
		if err != nil {
			os.Exit(1)
		}
		os.Exit(code)
	}

	for _, name := range strings.Split(os.Getenv("HELPER_ENV_NAMES"), ",") {
		fmt.Printf("%s=%s\n", name, os.Getenv(name))
	}

	os.Exit(0)
}
