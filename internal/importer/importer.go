package importer

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf16"

	"github.com/benalbone/standard-output-parser/internal/converter"
)

var (
	zipMagic = []byte{'P', 'K', 0x03, 0x04}
	oleMagic = []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}
)

// Info describes how values were imported from a source file.
type Info struct {
	Format  string
	Sources []string
}

// Convert detects the input content, extracts barcode values, and formats them.
func Convert(data []byte) (converter.Result, Info, error) {
	if bytes.HasPrefix(data, zipMagic) {
		values, sources, matched, err := extractXLSX(data)
		if err != nil {
			return converter.Result{}, Info{}, err
		}
		if !matched {
			return converter.Result{}, Info{}, fmt.Errorf(
				"this ZIP archive is not an Excel workbook; extract it or provide a plain-text or Excel file",
			)
		}
		if len(sources) == 0 {
			return converter.Result{}, Info{}, fmt.Errorf(
				"Excel workbook has no column headed Barcode, EAN, GTIN, or UPC",
			)
		}
		return converter.FromValues(values), Info{
			Format:  "Excel workbook",
			Sources: sources,
		}, nil
	}

	if bytes.HasPrefix(data, oleMagic) {
		return converter.Result{}, Info{}, fmt.Errorf(
			"legacy Microsoft Office binary files are not supported; save the workbook as .xlsx or CSV",
		)
	}
	if looksBinary(data) {
		return converter.Result{}, Info{}, fmt.Errorf(
			"unsupported binary file; provide an Excel workbook or a file containing readable text",
		)
	}

	text := decodeText(data)
	return converter.Convert(text), Info{Format: "plain text"}, nil
}

func looksBinary(data []byte) bool {
	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}

	if bytes.HasPrefix(sample, []byte{0xff, 0xfe}) || bytes.HasPrefix(sample, []byte{0xfe, 0xff}) {
		return false
	}
	return bytes.IndexByte(sample, 0) >= 0
}

func decodeText(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}):
		return string(data[3:])
	case bytes.HasPrefix(data, []byte{0xff, 0xfe}):
		return decodeUTF16(data[2:], binary.LittleEndian)
	case bytes.HasPrefix(data, []byte{0xfe, 0xff}):
		return decodeUTF16(data[2:], binary.BigEndian)
	default:
		return string(data)
	}
}

func decodeUTF16(data []byte, order binary.ByteOrder) string {
	units := make([]uint16, 0, len(data)/2)
	for len(data) >= 2 {
		units = append(units, order.Uint16(data[:2]))
		data = data[2:]
	}
	return string(utf16.Decode(units))
}

func zipReader(data []byte) (*zip.Reader, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open Excel workbook: %w", err)
	}
	return reader, nil
}
