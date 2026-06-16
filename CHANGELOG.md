# Changelog

All notable changes to Standard Output Parser are documented here.

The project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

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

[Unreleased]: https://github.com/benalbone/standard-output-parser/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/benalbone/standard-output-parser/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/benalbone/standard-output-parser/releases/tag/v0.1.0
