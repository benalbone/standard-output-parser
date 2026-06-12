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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, clipboard.Copy))
}

func run(args []string, stdout, stderr io.Writer, copyToClipboard func(string) error) int {
	flags := flag.NewFlagSet("sop", flag.ContinueOnError)
	flags.SetOutput(stderr)

	var outputPath string
	var force bool
	var noCopy bool
	var showVersion bool
	flags.StringVar(&outputPath, "output", "", "save the formatted result to this file")
	flags.BoolVar(&force, "force", false, "overwrite an existing output file")
	flags.BoolVar(&noCopy, "no-copy", false, "do not copy the result to the clipboard")
	flags.BoolVar(&showVersion, "version", false, "print the SOP version and exit")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sop [--output PATH] [--force] [--no-copy] <input-file>")
		fmt.Fprintln(stderr, "       sop --version")
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
	data, err := os.ReadFile(inputPath)
	if err != nil {
		fmt.Fprintf(stderr, "sop: read %s: %v\n", inputPath, err)
		return 1
	}

	result, importInfo, err := importer.Convert(data)
	if err != nil {
		fmt.Fprintf(stderr, "sop: import %s: %v\n", inputPath, err)
		return 1
	}
	if outputPath != "" {
		if err := writeResult(outputPath, result.Output, force); err != nil {
			fmt.Fprintf(stderr, "sop: save %s: %v\n", outputPath, err)
			return 1
		}
	}

	if result.Output != "" {
		fmt.Fprintln(stdout, result.Output)
	}

	var clipboardErr error
	if !noCopy && result.Output != "" {
		clipboardErr = copyToClipboard(result.Output)
	}

	printSummary(stderr, result, importInfo, outputPath, noCopy, clipboardErr)
	return 0
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
) {
	if len(importInfo.Sources) > 0 {
		fmt.Fprintf(
			writer,
			"Imported %s barcode column%s: %s.\n",
			importInfo.Format,
			plural(len(importInfo.Sources)),
			strings.Join(importInfo.Sources, ", "),
		)
	}
	total := len(result.Values)
	fmt.Fprintf(
		writer,
		"Found %d barcode%s: %d standard, %d nonstandard.\n",
		total,
		plural(total),
		result.StandardCount(),
		len(result.Nonstandard),
	)

	if outputPath != "" {
		fmt.Fprintf(writer, "Saved formatted output to %s.\n", outputPath)
	}
	if total == 0 {
		fmt.Fprintln(writer, "Warning: no standalone numeric values were found; the clipboard was not changed.")
	} else if noCopy {
		fmt.Fprintln(writer, "Clipboard copy disabled.")
	} else if clipboardErr != nil {
		fmt.Fprintf(writer, "Warning: %v. The formatted output is still available on stdout.\n", clipboardErr)
	} else {
		fmt.Fprintln(writer, "Copied formatted output to the clipboard.")
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
	for _, warning := range result.Nonstandard {
		fmt.Fprintf(
			writer,
			"Warning: item %d, '%s', has %d digits (expected 8 or 13).\n",
			warning.Position,
			warning.Value,
			warning.Length,
		)
	}
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
