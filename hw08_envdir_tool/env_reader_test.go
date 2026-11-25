package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()

	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestReadDirTestdata(t *testing.T) {
	env, err := ReadDir("testdata/env")
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	expected := Environment{
		"HELLO": {Value: `"hello"`},
		"BAR":   {Value: "bar"},
		"FOO":   {Value: "   foo\nwith new line"},
		"UNSET": {NeedRemove: true},
		"EMPTY": {Value: ""},
	}

	if len(env) != len(expected) {
		t.Fatalf("ReadDir() returned %d variables, want %d", len(env), len(expected))
	}

	for name, want := range expected {
		got, ok := env[name]
		if !ok {
			t.Errorf("ReadDir() did not return %q", name)
			continue
		}
		if got != want {
			t.Errorf("%s = %+v, want %+v", name, got, want)
		}
	}
}

func TestReadDirEdgeCases(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "TRIMMED", "value \t \nsecond line")
	writeTestFile(t, dir, "NULS", "a\x00b\x00c")
	writeTestFile(t, dir, "EMPTY_LINE", "\nsecond line")
	writeTestFile(t, dir, "ZERO", "")

	subdir := filepath.Join(dir, "SUBDIR")
	if err := os.Mkdir(subdir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, subdir, "IGNORED", "value")

	env, err := ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	expected := Environment{
		"TRIMMED":    {Value: "value"},
		"NULS":       {Value: "a\nb\nc"},
		"EMPTY_LINE": {Value: ""},
		"ZERO":       {NeedRemove: true},
	}

	if len(env) != len(expected) {
		t.Fatalf("ReadDir() returned %d variables, want %d: %+v", len(env), len(expected), env)
	}

	for name, want := range expected {
		if got := env[name]; got != want {
			t.Errorf("%s = %+v, want %+v", name, got, want)
		}
	}
}

func TestReadDirInvalidName(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "BAD=NAME", "value")

	env, err := ReadDir(dir)
	if err == nil {
		t.Fatalf("ReadDir() error = nil, want error; env = %+v", env)
	}
}
