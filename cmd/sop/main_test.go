package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsCopiesAndKeepsDefaultDiagnosticsQuiet(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.csv")
	if err := os.WriteFile(input, []byte("00000000|123456|123456789012|00000000"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var copied string
	code := run([]string{input}, strings.NewReader(""), &stdout, &stderr, func(text string) error {
		copied = text
		return nil
	})

	want := "'00000000','0123456789012','00000000'"
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.TrimSuffix(stdout.String(), "\n") != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
	if !strings.Contains(stderr.String(), "Copied 3 formatted barcodes to the clipboard.") {
		t.Fatalf("stderr missing copy confirmation:\n%s", stderr.String())
	}
	for _, hidden := range []string{
		"Found 3 supported barcodes",
		"Skipped",
		"normalized",
		"duplicate '00000000'",
		"'123456', has 6 digits",
	} {
		if strings.Contains(stderr.String(), hidden) {
			t.Fatalf("stderr unexpectedly contains %q:\n%s", hidden, stderr.String())
		}
	}
}

func TestRunShowsWarningsWhenRequested(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.csv")
	if err := os.WriteFile(input, []byte("00000000|123456|123456789012|00000000"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--warnings", input}, strings.NewReader(""), &stdout, &stderr, func(string) error {
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	for _, expected := range []string{
		"Found 3 supported barcodes.",
		"Normalized 1 12-digit barcode by adding a leading zero.",
		"Skipped 1 unsupported numeric value.",
		"Copied 3 formatted barcodes to the clipboard.",
		"Info: item 3, '123456789012', was normalized to '0123456789012'.",
		"duplicate '00000000' appears 2 times",
		"'123456', has 6 digits (expected 8, 12, or 13); skipped",
	} {
		if !strings.Contains(stderr.String(), expected) {
			t.Fatalf("stderr missing %q:\n%s", expected, stderr.String())
		}
	}
}

func TestRunNoCopyStaysQuietByDefault(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input.csv")
	if err := os.WriteFile(input, []byte("00000000|123456"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--no-copy", input}, strings.NewReader(""), &stdout, &stderr, func(string) error {
		t.Fatal("clipboard should not be called")
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.String() != "'00000000'\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPrintsDevelopmentVersion(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr, func(string) error {
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

	code := run([]string{"--version", "input.txt"}, strings.NewReader(""), &stdout, &stderr, func(string) error {
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
		strings.NewReader(""),
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
		strings.NewReader(""),
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
		strings.NewReader(""),
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
	if err := os.WriteFile(input, []byte("Barcode,Description\n123456,Short code"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{input}, strings.NewReader(""), &stdout, &stderr, func(string) error {
		t.Fatal("clipboard should not be called")
		return nil
	})

	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "No supported 8-, 12-, or 13-digit barcode values were found") {
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
	code := run([]string{input}, strings.NewReader(""), &stdout, &stderr, func(string) error {
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

func TestRunReadsStandardInput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	var copied string

	code := run(
		[]string{"-"},
		strings.NewReader("00000000|1234567890123"),
		&stdout,
		&stderr,
		func(text string) error {
			copied = text
			return nil
		},
	)

	const want = "'00000000','1234567890123'"
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.String() != want+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want+"\n")
	}
	if copied != want {
		t.Fatalf("copied = %q, want %q", copied, want)
	}
	if !strings.Contains(stderr.String(), "Copied 2 formatted barcodes to the clipboard.") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunReportsStandardInputReadFailure(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"-"}, failingReader{}, &stdout, &stderr, func(string) error {
		t.Fatal("clipboard should not be called")
		return nil
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "read standard input: test read failure") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("test read failure")
}
