# Standard Output Parser

Standard Output Parser (`sop`) extracts supported barcode values from awkward
text and Excel files and converts them into a canonical list:

```text
'0000000000000','0000000000001','0000000000002'
```

It preserves source order, leading zeroes, and duplicate occurrences. SOP keeps
8- and 13-digit barcodes as-is, normalizes 12-digit barcodes by adding a leading
zero, and ignores every other standalone numeric length.

## Install

With Homebrew:

```bash
brew install benalbone/tap/sop
```

From source:

```bash
git clone https://github.com/benalbone/standard-output-parser.git
cd standard-output-parser
go build -trimpath -o sop ./cmd/sop
```

## Usage

```text
sop [--column] [--output PATH] [--force] [--no-copy] [--show-output] [--warnings] <input-file|->
sop --version
```

Examples:

```bash
sop products.xlsx
sop --column products.xlsx
sop --show-output barcodes.csv
sop --show-output --no-copy barcodes.csv
sop --warnings products.xlsx
sop --output formatted.txt products.xlsx
sop --output formatted.txt --force products.xlsx
pbpaste | sop -
cat barcodes.txt | sop --show-output --no-copy -
```

By default, the formatted result is copied to the clipboard and SOP prints only
a short confirmation such as:

```text
Copied 1952 formatted barcodes to the clipboard.
```

Use `--show-output` when the canonical barcode list should also be written to
standard output. Routine diagnostics remain hidden unless `--warnings` is
supplied:

```bash
sop --show-output products.xlsx
sop --show-output --no-copy products.xlsx > formatted.txt
```

Use `--column` to keep the single quotes and commas while placing each barcode
on its own line. This layout is used consistently for the clipboard,
`--show-output`, and `--output`:

```text
'00000000',
'0123456789012',
'5012345678901'
```

With `--warnings`, detailed counts, Excel import details, 12-digit
normalization notes, duplicate notices, and skipped-value warnings are written
to standard error:

```bash
sop --warnings products.xlsx
```

Use `-` as the input argument to read from standard input. This allows text
from the clipboard, another command, an archive tool, or an export pipeline to
be processed without first creating a temporary file:

```bash
pbpaste | sop -
unzip -p exports.zip products.csv | sop --show-output --no-copy -
```

Options:

- `--column` places each quoted, comma-separated barcode on its own line.
- `--output PATH` also saves the result with one trailing newline.
- `--force` allows an existing output file to be overwritten.
- `--no-copy` disables clipboard copying.
- `--show-output` prints the canonical barcode list to standard output.
- `--warnings` shows counts, normalization notes, duplicate notices, and
  skipped-value warnings.
- `--version` prints the installed version and exits.

## Barcode Rules

SOP only outputs these standalone ASCII digit sequences:

- 8 digits, unchanged.
- 12 digits, with a leading `0` added.
- 13 digits, unchanged.

Every other standalone numeric value is ignored. Numbers embedded inside words
are ignored before length rules are applied.

Example:

```text
12345678
123456789012
5012345678901
123456
```

Output:

```text
'12345678','0123456789012','5012345678901'
```

## Supported Input

Input is detected from its contents rather than restricted by filename
extension:

- Excel Open XML workbooks, including `.xlsx` and `.xlsm`. SOP searches the
  first 50 rows of each sheet for Barcode, EAN, GTIN, or UPC column headings.
- CSV, TSV, pipe-delimited exports, and other readable text files.
- UTF-8 and UTF-16 text.
- Standard input when `-` is supplied as the input argument.

Legacy binary `.xls` files should be saved as `.xlsx` or CSV. Other binary
formats are rejected instead of being scanned for unrelated internal numbers.

## Clipboard Support

- macOS: `pbcopy`
- Windows: `clip`
- Linux: `wl-copy`, `xclip`, or `xsel`

Clipboard failure is always reported, even when `--warnings` is not supplied.
Use `--show-output` or `--output` when the formatted result must remain
available even if no supported clipboard command is installed.

## Development

```bash
go test ./...
go vet ./...
CGO_ENABLED=0 go build -trimpath -o dist/sop ./cmd/sop
```

Development builds report:

```text
sop dev
```

Release builds inject the version:

```bash
go build -trimpath \
  -ldflags="-s -w -X main.version=0.6.0" \
  -o dist/sop ./cmd/sop
```

## Releasing

Feature and bug-fix commits are merged before release preparation. When a
useful set of changes is ready:

1. Create `release/vX.Y.Z` from current `main`.
2. Move the relevant changelog entries from `Unreleased` into a dated version.
3. Run the tests, vet, static build, version check, and workbook smoke test.
4. Commit only release metadata as `Prepare vX.Y.Z`.
5. Merge the release pull request.
6. Tag the exact merge commit and create the GitHub release.
7. Update the Homebrew formula in a separate repository and pull request.

Published tags are permanent and must never be reused or moved.

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Licence

[MIT](LICENSE)
