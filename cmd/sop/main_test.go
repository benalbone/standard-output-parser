package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsCopiesAndSummarises(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.csv")
	if err := os.WriteFile(input, []byte("00000000|123456|00000000"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var copied string
	code := run([]string{input}, &stdout, &stderr, func(text string) error {
		copied = text
		return nil
	})

	want := "'00000000','123456','00000000'"
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSuffix(stdout.String(), "\n") != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
	for _, expected := range []string{
		"Found 3 barcodes: 2 standard, 1 nonstandard.",
		"duplicate '00000000' appears 2 times",
		"'123456', has 6 digits",
	} {
		if !strings.Contains(stderr.String(), expected) {
			t.Fatalf("stderr missing %q:\n%s", expected, stderr.String())
		}
	}
}

func TestRunPrintsDevelopmentVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version"}, &stdout, &stderr, func(string) error {
		t.Fatal("clipboard should not be called")
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.String() != "sop dev\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunRejectsInputWithVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version", "input.txt"}, &stdout, &stderr, func(string) error {
		return nil
	})

	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "--version does not accept an input file") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunSavesWithTrailingNewlineAndRequiresForce(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.txt")
	output := filepath.Join(dir, "output.txt")
	if err := os.WriteFile(input, []byte("12345678"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run(
		[]string{"--output", output, "--no-copy", input},
		&stdout,
		&stderr,
		func(string) error { t.Fatal("clipboard should not be called"); return nil },
	)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "'12345678'\n" {
		t.Fatalf("saved data = %q", data)
	}

	stderr.Reset()
	code = run(
		[]string{"--output", output, "--no-copy", input},
		&stdout,
		&stderr,
		func(string) error { return nil },
	)
	if code != 1 || !strings.Contains(stderr.String(), "use --force") {
		t.Fatalf("expected existing-file failure, code=%d stderr=%s", code, stderr.String())
	}

	stderr.Reset()
	code = run(
		[]string{"--output", output, "--force", "--no-copy", input},
		&stdout,
		&stderr,
		func(string) error { return nil },
	)
	if code != 0 {
		t.Fatalf("forced overwrite failed, code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunDoesNotCopyWhenNoBarcodesAreFound(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(input, []byte("Barcode,Description"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{input}, &stdout, &stderr, func(string) error {
		t.Fatal("clipboard should not be called")
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "clipboard was not changed") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunTreatsClipboardFailureAsWarning(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.txt")
	if err := os.WriteFile(input, []byte("12345678"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{input}, &stdout, &stderr, func(string) error {
		return errors.New("clipboard unavailable")
	})

	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(stderr.String(), "clipboard unavailable") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
	if stdout.String() != "'12345678'\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}
