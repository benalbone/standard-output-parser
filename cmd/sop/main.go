package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/benalbone/standard-output-parser/internal/clipboard"
	"github.com/benalbone/standard-output-parser/internal/converter"
	"github.com/benalbone/standard-output-parser/internal/importer"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, clipboard.Copy))
}

func run(
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	copyToClipboard func(string) error,
) int {
	flags := flag.NewFlagSet("sop", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var outputPath string
	var column bool
	var force bool
	var noCopy bool
	var showOutput bool
	var showWarnings bool
	var showVersion bool
	flags.BoolVar(&column, "column", false, "place each formatted barcode on its own line")
	flags.StringVar(&outputPath, "output", "", "save the formatted result to this file")
	flags.BoolVar(&force, "force", false, "overwrite an existing output file")
	flags.BoolVar(&noCopy, "no-copy", false, "do not copy the result to the clipboard")
	flags.BoolVar(&showOutput, "show-output", false, "print the formatted result to standard output")
	flags.BoolVar(&showWarnings, "warnings", false, "show counts, normalization notes, duplicate notices, and skipped values")
	flags.BoolVar(&showVersion, "version", false, "print the SOP version and exit")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sop [--column] [--output PATH] [--force] [--no-copy] [--show-output] [--warnings] <input-file|->")
		fmt.Fprintln(stderr, "       sop --version")
		fmt.Fprintln(stderr, "Use - to read input from standard input.")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if showVersion {
		if flags.NArg() != 0 {
			fmt.Fprintln(stderr, "sop: --version does not accept an input file")
			return 2
		}
		fmt.Fprintf(stdout, "sop %s\n", version)
		return 0
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}
	if force && outputPath == "" {
		fmt.Fprintln(stderr, "sop: --force requires --output")
		return 2
	}

	inputPath := flags.Arg(0)
	data, inputName, err := readInput(inputPath, stdin)
	if err != nil {
		fmt.Fprintf(stderr, "sop: read %s: %v\n", inputName, err)
		return 1
	}

	result, importInfo, err := importer.Convert(data)
	if err != nil {
		fmt.Fprintf(stderr, "sop: import %s: %v\n", inputName, err)
		return 1
	}
	formattedOutput := formatOutput(result, column)
	if outputPath != "" {
		if err := writeResult(outputPath, formattedOutput, force); err != nil {
			fmt.Fprintf(stderr, "sop: save %s: %v\n", outputPath, err)
			return 1
		}
	}

	if showOutput && formattedOutput != "" {
		fmt.Fprintln(stdout, formattedOutput)
	}

	var clipboardErr error
	if !noCopy && formattedOutput != "" {
		clipboardErr = copyToClipboard(formattedOutput)
	}

	printSummary(stderr, result, importInfo, outputPath, noCopy, clipboardErr, showWarnings)
	return 0
}

func formatOutput(result converter.Result, column bool) string {
	if !column || len(result.Values) == 0 {
		return result.Output
	}

	quoted := make([]string, len(result.Values))
	for index, value := range result.Values {
		quoted[index] = "'" + value + "'"
	}
	return strings.Join(quoted, ",\n")
}

func readInput(path string, stdin io.Reader) ([]byte, string, error) {
	if path == "-" {
		data, err := io.ReadAll(stdin)
		return data, "standard input", err
	}

	data, err := os.ReadFile(path)
	return data, path, err
}

func writeResult(path, output string, force bool) error {
	flags := os.O_WRONLY | os.O_CREATE
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	file, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("file already exists (use --force to overwrite)")
		}
		return err
	}

	_, writeErr := io.WriteString(file, output+"\n")
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func printSummary(
	writer io.Writer,
	result converter.Result,
	importInfo importer.Info,
	outputPath string,
	noCopy bool,
	clipboardErr error,
	showWarnings bool,
) {
	if showWarnings && len(importInfo.Sources) > 0 {
		fmt.Fprintf(
			writer,
			"Imported %s barcode column%s: %s.\n",
			importInfo.Format,
			plural(len(importInfo.Sources)),
			strings.Join(importInfo.Sources, ", "),
		)
	}
	total := len(result.Values)
	if showWarnings {
		fmt.Fprintf(
			writer,
			"Found %d supported barcode%s.\n",
			total,
			plural(total),
		)
		if len(result.Normalized) > 0 {
			fmt.Fprintf(
				writer,
				"Normalized %d 12-digit barcode%s by adding a leading zero.\n",
				len(result.Normalized),
				plural(len(result.Normalized)),
			)
		}
		if len(result.Skipped) > 0 {
			fmt.Fprintf(
				writer,
				"Skipped %d unsupported numeric value%s.\n",
				len(result.Skipped),
				plural(len(result.Skipped)),
			)
		}
	}

	if outputPath != "" {
		fmt.Fprintf(writer, "Saved formatted output to %s.\n", outputPath)
	}
	if total == 0 {
		fmt.Fprintln(writer, "No supported 8-, 12-, or 13-digit barcode values were found; the clipboard was not changed.")
	} else if noCopy {
		if showWarnings {
			fmt.Fprintln(writer, "Clipboard copy disabled.")
		}
	} else if clipboardErr != nil {
		fmt.Fprintf(writer, "Warning: %v. The formatted output was not copied; use --show-output or --output to preserve it.\n", clipboardErr)
	} else {
		fmt.Fprintf(writer, "Copied %d formatted barcode%s to the clipboard.\n", total, plural(total))
	}

	if !showWarnings {
		return
	}
	for _, normalized := range result.Normalized {
		fmt.Fprintf(
			writer,
			"Info: item %d, '%s', was normalized to '%s'.\n",
			normalized.Position,
			normalized.Original,
			normalized.Value,
		)
	}
	for _, duplicate := range result.Duplicates {
		fmt.Fprintf(
			writer,
			"Warning: duplicate '%s' appears %d times (first at item %d).\n",
			duplicate.Value,
			duplicate.Count,
			duplicate.FirstPosition,
		)
	}
	for _, skipped := range result.Skipped {
		fmt.Fprintf(
			writer,
			"Warning: item %d, '%s', has %d digits (expected 8, 12, or 13); skipped.\n",
			skipped.Position,
			skipped.Value,
			skipped.Length,
		)
	}
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
