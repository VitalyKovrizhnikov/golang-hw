package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return data
}

func TestCopy(t *testing.T) {
	oldOutput := progressOutput
	progressOutput = io.Discard
	t.Cleanup(func() {
		progressOutput = oldOutput
	})

	tests := []struct {
		name     string
		expected string
		offset   int64
		limit    int64
	}{
		{name: "offset0_limit0", expected: "testdata/out_offset0_limit0.txt"},
		{name: "offset0_limit10", expected: "testdata/out_offset0_limit10.txt", limit: 10},
		{name: "offset0_limit1000", expected: "testdata/out_offset0_limit1000.txt", limit: 1000},
		{name: "offset0_limit10000", expected: "testdata/out_offset0_limit10000.txt", limit: 10000},
		{name: "offset100_limit1000", expected: "testdata/out_offset100_limit1000.txt", offset: 100, limit: 1000},
		{name: "offset6000_limit1000", expected: "testdata/out_offset6000_limit1000.txt", offset: 6000, limit: 1000},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			to := filepath.Join(t.TempDir(), "out.txt")

			if err := Copy("testdata/input.txt", to, tc.offset, tc.limit); err != nil {
				t.Fatalf("Copy() error = %v", err)
			}

			expected := readFile(t, tc.expected)
			actual := readFile(t, to)
			if !bytes.Equal(expected, actual) {
				t.Errorf("Copy() = %d bytes, expected %d bytes", len(actual), len(expected))
			}
		})
	}
}

func TestCopyErrors(t *testing.T) {
	oldOutput := progressOutput
	progressOutput = io.Discard
	t.Cleanup(func() {
		progressOutput = oldOutput
	})

	inputSize := int64(len(readFile(t, "testdata/input.txt")))

	t.Run("offset exceeds file size", func(t *testing.T) {
		to := filepath.Join(t.TempDir(), "out.txt")

		err := Copy("testdata/input.txt", to, inputSize+1, 0)
		if !errors.Is(err, ErrOffsetExceedsFileSize) {
			t.Fatalf("error = %v, want %v", err, ErrOffsetExceedsFileSize)
		}
	})

	t.Run("unsupported file", func(t *testing.T) {
		to := filepath.Join(t.TempDir(), "out.txt")

		err := Copy("testdata", to, 0, 0)
		if !errors.Is(err, ErrUnsupportedFile) {
			t.Fatalf("error = %v, want %v", err, ErrUnsupportedFile)
		}
	})

	t.Run("negative offset", func(t *testing.T) {
		to := filepath.Join(t.TempDir(), "out.txt")

		err := Copy("testdata/input.txt", to, -1, 0)
		if !errors.Is(err, ErrInvalidOffset) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidOffset)
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		to := filepath.Join(t.TempDir(), "out.txt")

		err := Copy("testdata/input.txt", to, 0, -1)
		if !errors.Is(err, ErrInvalidLimit) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidLimit)
		}
	})

	t.Run("same source and destination", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "file.txt")
		if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}

		err := Copy(path, path, 0, 0)
		if !errors.Is(err, ErrSameFile) {
			t.Fatalf("error = %v, want %v", err, ErrSameFile)
		}
	})
}

func TestCopyEmptyFile(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "empty.txt")
	to := filepath.Join(dir, "out.txt")

	if err := os.WriteFile(from, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Copy(from, to, 0, 0); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	if data := readFile(t, to); len(data) != 0 {
		t.Errorf("Copy() produced %d bytes, want 0", len(data))
	}
}

func TestCopyTruncatesDestination(t *testing.T) {
	to := filepath.Join(t.TempDir(), "out.txt")
	if err := os.WriteFile(to, []byte("stale content that must be removed"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Copy("testdata/input.txt", to, 0, 10); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	expected := readFile(t, "testdata/out_offset0_limit10.txt")
	actual := readFile(t, to)
	if !bytes.Equal(expected, actual) {
		t.Errorf("Copy() = %q, want %q", actual, expected)
	}
}

func TestCopyProgress(t *testing.T) {
	var output bytes.Buffer

	oldOutput := progressOutput
	progressOutput = &output
	t.Cleanup(func() {
		progressOutput = oldOutput
	})

	to := filepath.Join(t.TempDir(), "out.txt")
	if err := Copy("testdata/input.txt", to, 0, 1000); err != nil {
		t.Fatalf("Copy() error = %v", err)
	}

	if !strings.Contains(output.String(), "%") {
		t.Errorf("progress output %q does not contain a percentage", output.String())
	}
}
