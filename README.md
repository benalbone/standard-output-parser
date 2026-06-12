# Standard Output Parser

Standard Output Parser (`sop`) extracts barcode values from awkward text and
Excel files and converts them into a canonical list:

```text
'0000000000000','0000000000001','0000000000002'
```

It preserves source order, leading zeroes, and duplicate occurrences. Values
containing 8 or 13 digits are considered standard; every other standalone
numeric value is retained and reported as a warning.

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
sop [--output PATH] [--force] [--no-copy] <input-file>
sop --version
```

Examples:

```bash
sop products.xlsx
sop --no-copy barcodes.csv
sop --output formatted.txt products.xlsx
sop --output formatted.txt --force products.xlsx
```

The formatted result is written to standard output and copied to the clipboard
by default. Counts and warnings are written to standard error, so the result
can be redirected safely:

```bash
sop --no-copy products.xlsx > formatted.txt
```

Options:

- `--output PATH` also saves the result with one trailing newline.
- `--force` allows an existing output file to be overwritten.
- `--no-copy` disables clipboard copying.
- `--version` prints the installed version and exits.

## Supported Input

Input is detected from its contents rather than restricted by filename
extension:

- Excel Open XML workbooks, including `.xlsx` and `.xlsm`. SOP searches the
  first 50 rows of each sheet for Barcode, EAN, GTIN, or UPC column headings.
- CSV, TSV, pipe-delimited exports, and other readable text files.
- UTF-8 and UTF-16 text.

Legacy binary `.xls` files should be saved as `.xlsx` or CSV. Other binary
formats are rejected instead of being scanned for unrelated internal numbers.

## Clipboard Support

- macOS: `pbcopy`
- Windows: `clip`
- Linux: `wl-copy`, `xclip`, or `xsel`

Clipboard failure is reported as a warning and does not discard standard
output or a requested output file.

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
  -ldflags="-s -w -X main.version=0.1.0" \
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
