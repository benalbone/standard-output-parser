# Changelog

All notable changes to Standard Output Parser are documented here.

The project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.6.0] - 2026-08-24

### Added

- Added `--column` to place each quoted, comma-separated barcode on its own
  line in clipboard, terminal, and saved output.

## [0.5.0] - 2026-08-05

### Added

- Added `--show-output` to print the canonical barcode list to standard output.

### Changed

- Hid the canonical barcode list by default so normal runs print only the
  clipboard confirmation.

## [0.4.0] - 2026-06-16

### Changed

- Restricted output to standalone 8-, 12-, and 13-digit barcode values.
- Normalized standalone 12-digit barcodes by adding a leading zero.
- Skipped unsupported standalone numeric lengths instead of including them in
  formatted output.

## [0.3.0] - 2026-06-16

### Added

- Added `--warnings` to show counts, duplicate notices, and nonstandard-length
  warnings on demand.

### Changed

- Hid routine warning details by default so everyday CLI output stays minimal.

## [0.2.0] - 2026-06-12

### Added

- Standard input support through `sop -` for pipelines and clipboard input.

## [0.1.0] - 2026-06-12

### Added

- Plain-text and Excel barcode extraction.
- Canonical single-quoted, comma-separated output.
- Duplicate and nonstandard-length warnings.
- Clipboard and output-file support.
- Version reporting through `sop --version`.

[Unreleased]: https://github.com/benalbone/standard-output-parser/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/benalbone/standard-output-parser/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/benalbone/standard-output-parser/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/benalbone/standard-output-parser/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/benalbone/standard-output-parser/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/benalbone/standard-output-parser/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/benalbone/standard-output-parser/releases/tag/v0.1.0
