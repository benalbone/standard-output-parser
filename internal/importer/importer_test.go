package importer

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestConvertPlainTextRegardlessOfFilename(t *testing.T) {
	result, info, err := Convert([]byte("00000000|123456789012|1234567890123|42"))
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"00000000", "0123456789012", "1234567890123"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("values = %#v, want %#v", result.Values, want)
	}
	if len(result.Normalized) != 1 || result.Normalized[0].Value != "0123456789012" {
		t.Fatalf("unexpected normalized values: %#v", result.Normalized)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].Value != "42" {
		t.Fatalf("unexpected skipped values: %#v", result.Skipped)
	}
	if info.Format != "plain text" {
		t.Fatalf("format = %q", info.Format)
	}
}

func TestConvertUTF16Text(t *testing.T) {
	units := utf16.Encode([]rune("00000000\r\n1234567890123"))
	data := []byte{0xff, 0xfe}
	for _, unit := range units {
		var encoded [2]byte
		binary.LittleEndian.PutUint16(encoded[:], unit)
		data = append(data, encoded[:]...)
	}

	result, _, err := Convert(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"00000000", "1234567890123"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("values = %#v, want %#v", result.Values, want)
	}
}

func TestConvertXLSXReadsOnlyBarcodeColumns(t *testing.T) {
	data := testWorkbook(t, true)
	result, info, err := Convert(data)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"00000000", "5012345678901", "00000000"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("values = %#v, want %#v", result.Values, want)
	}
	if info.Format != "Excel workbook" {
		t.Fatalf("format = %q", info.Format)
	}
	wantSources := []string{"Products!B (Barcode)"}
	if !reflect.DeepEqual(info.Sources, wantSources) {
		t.Fatalf("sources = %#v, want %#v", info.Sources, wantSources)
	}
	if len(result.Duplicates) != 1 || result.Duplicates[0].Value != "00000000" {
		t.Fatalf("unexpected duplicates: %#v", result.Duplicates)
	}
	if len(result.Skipped) != 1 || result.Skipped[0].Value != "123456" {
		t.Fatalf("unexpected skipped values: %#v", result.Skipped)
	}
}

func TestConvertXLSXRequiresBarcodeHeader(t *testing.T) {
	_, _, err := Convert(testWorkbook(t, false))
	if err == nil || !strings.Contains(err.Error(), "no column headed") {
		t.Fatalf("expected missing barcode header error, got %v", err)
	}
}

func TestConvertRejectsUnknownBinary(t *testing.T) {
	_, _, err := Convert([]byte{0x01, 0x00, 0x02, 0x03})
	if err == nil || !strings.Contains(err.Error(), "unsupported binary") {
		t.Fatalf("expected binary error, got %v", err)
	}
}

func testWorkbook(t *testing.T, barcodeHeader bool) []byte {
	t.Helper()

	headerIndex := 0
	if barcodeHeader {
		headerIndex = 1
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	files := map[string]string{
		"xl/workbook.xml": `<?xml version="1.0"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"
 xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
 <sheets><sheet name="Products" sheetId="1" r:id="rId1"/></sheets>
</workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
 <Relationship Id="rId1" Target="worksheets/sheet1.xml"/>
</Relationships>`,
		"xl/sharedStrings.xml": `<?xml version="1.0"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
 <si><t>Quantity</t></si>
 <si><t>Barcode</t></si>
 <si><t>Description</t></si>
 <si><t>Code</t></si>
</sst>`,
		"xl/styles.xml": `<?xml version="1.0"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
 <numFmts count="1"><numFmt numFmtId="164" formatCode="00000000"/></numFmts>
 <cellXfs count="2"><xf numFmtId="0"/><xf numFmtId="164"/></cellXfs>
</styleSheet>`,
		"xl/worksheets/sheet1.xml": `<?xml version="1.0"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
 <sheetData>
  <row r="1">
   <c r="A1" t="s"><v>0</v></c>
   <c r="B1" t="s"><v>` + string(rune('0'+headerIndex)) + `</v></c>
   <c r="C1" t="s"><v>2</v></c>
  </row>
  <row r="2"><c r="A2"><v>12</v></c><c r="B2" s="1"><v>0</v></c><c r="C2" t="s"><v>3</v></c></row>
  <row r="3"><c r="A3"><v>99</v></c><c r="B3"><v>5012345678901</v></c></row>
  <row r="4"><c r="A4"><v>7</v></c><c r="B4" t="inlineStr"><is><t>123456 | 00000000</t></is></c></row>
 </sheetData>
</worksheet>`,
	}

	for name, contents := range files {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
