package importer

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"
	"unicode"

	"github.com/benalbone/standard-output-parser/internal/converter"
)

const maxHeaderRow = 50

type workbookDocument struct {
	Sheets []workbookSheet `xml:"sheets>sheet"`
}

type workbookSheet struct {
	Name string `xml:"name,attr"`
	ID   string `xml:"id,attr"`
}

type relationshipsDocument struct {
	Relationships []relationship `xml:"Relationship"`
}

type relationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type sharedStringsDocument struct {
	Items []stringItem `xml:"si"`
}

type stylesDocument struct {
	NumberFormats []numberFormat `xml:"numFmts>numFmt"`
	CellFormats   []cellFormat   `xml:"cellXfs>xf"`
}

type numberFormat struct {
	ID   int    `xml:"numFmtId,attr"`
	Code string `xml:"formatCode,attr"`
}

type cellFormat struct {
	NumberFormatID int `xml:"numFmtId,attr"`
}

type stringItem struct {
	Text string    `xml:"t"`
	Runs []textRun `xml:"r"`
}

type textRun struct {
	Text string `xml:"t"`
}

type worksheetRow struct {
	Number int             `xml:"r,attr"`
	Cells  []worksheetCell `xml:"c"`
}

type worksheetCell struct {
	Reference string       `xml:"r,attr"`
	Type      string       `xml:"t,attr"`
	Style     int          `xml:"s,attr"`
	Value     string       `xml:"v"`
	Inline    inlineString `xml:"is"`
}

type inlineString struct {
	Text string    `xml:"t"`
	Runs []textRun `xml:"r"`
}

type barcodeColumn struct {
	Index  int
	Header string
}

func extractXLSX(data []byte) ([]string, []string, bool, error) {
	reader, err := zipReader(data)
	if err != nil {
		return nil, nil, true, err
	}

	files := make(map[string]*zip.File, len(reader.File))
	for _, file := range reader.File {
		files[path.Clean(file.Name)] = file
	}
	if files["xl/workbook.xml"] == nil {
		return nil, nil, false, nil
	}

	var workbook workbookDocument
	if err := decodeXMLFile(files["xl/workbook.xml"], &workbook); err != nil {
		return nil, nil, true, fmt.Errorf("read Excel workbook metadata: %w", err)
	}

	var relationships relationshipsDocument
	if err := decodeXMLFile(files["xl/_rels/workbook.xml.rels"], &relationships); err != nil {
		return nil, nil, true, fmt.Errorf("read Excel worksheet relationships: %w", err)
	}
	targets := make(map[string]string, len(relationships.Relationships))
	for _, relation := range relationships.Relationships {
		targets[relation.ID] = workbookTarget(relation.Target)
	}

	sharedStrings, err := readSharedStrings(files["xl/sharedStrings.xml"])
	if err != nil {
		return nil, nil, true, err
	}
	numberFormats, err := readNumberFormats(files["xl/styles.xml"])
	if err != nil {
		return nil, nil, true, err
	}

	var values []string
	var sources []string
	for _, sheet := range workbook.Sheets {
		target := targets[sheet.ID]
		if target == "" {
			continue
		}
		file := files[target]
		if file == nil {
			return nil, nil, true, fmt.Errorf("worksheet %q is missing from the Excel workbook", sheet.Name)
		}

		sheetValues, columns, err := readWorksheet(file, sharedStrings, numberFormats)
		if err != nil {
			return nil, nil, true, fmt.Errorf("read worksheet %q: %w", sheet.Name, err)
		}
		values = append(values, sheetValues...)
		for _, column := range columns {
			sources = append(sources, fmt.Sprintf(
				"%s!%s (%s)",
				sheet.Name,
				columnName(column.Index),
				column.Header,
			))
		}
	}

	return values, sources, true, nil
}

func readWorksheet(
	file *zip.File,
	sharedStrings []string,
	numberFormats map[int]string,
) ([]string, []barcodeColumn, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, nil, err
	}
	defer reader.Close()

	decoder := xml.NewDecoder(reader)
	var headerRow int
	var columns []barcodeColumn
	var values []string

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "row" {
			continue
		}

		var row worksheetRow
		if err := decoder.DecodeElement(&row, &start); err != nil {
			return nil, nil, err
		}

		if headerRow == 0 && row.Number <= maxHeaderRow {
			for _, cell := range row.Cells {
				value := cellText(cell, sharedStrings, numberFormats)
				if isBarcodeHeader(value) {
					columns = append(columns, barcodeColumn{
						Index:  cellColumnIndex(cell.Reference),
						Header: strings.TrimSpace(value),
					})
				}
			}
			if len(columns) > 0 {
				headerRow = row.Number
			}
			continue
		}
		if headerRow == 0 || row.Number <= headerRow {
			continue
		}

		cells := make(map[int]string, len(row.Cells))
		for _, cell := range row.Cells {
			cells[cellColumnIndex(cell.Reference)] = cellText(cell, sharedStrings, numberFormats)
		}
		for _, column := range columns {
			values = append(values, cellBarcodeValues(cells[column.Index])...)
		}
	}

	return values, columns, nil
}

func readSharedStrings(file *zip.File) ([]string, error) {
	if file == nil {
		return nil, nil
	}

	var document sharedStringsDocument
	if err := decodeXMLFile(file, &document); err != nil {
		return nil, fmt.Errorf("read Excel shared strings: %w", err)
	}

	values := make([]string, 0, len(document.Items))
	for _, item := range document.Items {
		var text strings.Builder
		text.WriteString(item.Text)
		for _, run := range item.Runs {
			text.WriteString(run.Text)
		}
		values = append(values, text.String())
	}
	return values, nil
}

func readNumberFormats(file *zip.File) (map[int]string, error) {
	if file == nil {
		return nil, nil
	}

	var document stylesDocument
	if err := decodeXMLFile(file, &document); err != nil {
		return nil, fmt.Errorf("read Excel number formats: %w", err)
	}

	formatCodes := map[int]string{
		0: "General",
		1: "0",
	}
	for _, format := range document.NumberFormats {
		formatCodes[format.ID] = format.Code
	}

	styles := make(map[int]string, len(document.CellFormats))
	for index, format := range document.CellFormats {
		styles[index] = formatCodes[format.NumberFormatID]
	}
	return styles, nil
}

func decodeXMLFile(file *zip.File, target any) error {
	if file == nil {
		return fmt.Errorf("required workbook file is missing")
	}
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()
	return xml.NewDecoder(reader).Decode(target)
}

func workbookTarget(target string) string {
	target = strings.TrimPrefix(strings.ReplaceAll(target, "\\", "/"), "/")
	if strings.HasPrefix(target, "xl/") {
		return path.Clean(target)
	}
	return path.Clean(path.Join("xl", target))
}

func cellText(cell worksheetCell, sharedStrings []string, numberFormats map[int]string) string {
	switch cell.Type {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err != nil || index < 0 || index >= len(sharedStrings) {
			return ""
		}
		return sharedStrings[index]
	case "inlineStr":
		var text strings.Builder
		text.WriteString(cell.Inline.Text)
		for _, run := range cell.Inline.Runs {
			text.WriteString(run.Text)
		}
		return text.String()
	case "e":
		return ""
	default:
		return applyNumberFormat(cell.Value, numberFormats[cell.Style])
	}
}

func applyNumberFormat(value, format string) string {
	format = strings.TrimSpace(format)
	if format == "" || !allZeroFormat(format) {
		return value
	}

	number, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil || number < 0 {
		return value
	}
	integer := strconv.FormatFloat(number, 'f', 0, 64)
	if len(integer) >= len(format) {
		return integer
	}
	return strings.Repeat("0", len(format)-len(integer)) + integer
}

func allZeroFormat(format string) bool {
	if format == "" {
		return false
	}
	for _, r := range format {
		if r != '0' {
			return false
		}
	}
	return true
}

func isBarcodeHeader(value string) bool {
	var normalized strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			normalized.WriteRune(r)
		}
	}
	header := normalized.String()
	if strings.Contains(header, "barcode") {
		return true
	}
	switch header {
	case "ean", "ean8", "ean13",
		"gtin", "gtin8", "gtin12", "gtin13", "gtin14",
		"upc", "upca", "upce":
		return true
	default:
		return false
	}
}

func cellBarcodeValues(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if allASCIIDigits(value) {
		return []string{value}
	}

	if number, err := strconv.ParseFloat(value, 64); err == nil && number >= 0 {
		formatted := strconv.FormatFloat(number, 'f', -1, 64)
		if allASCIIDigits(formatted) {
			return []string{formatted}
		}
	}
	return converter.Extract(value)
}

func allASCIIDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func cellColumnIndex(reference string) int {
	index := 0
	for _, r := range reference {
		if r < 'A' || r > 'Z' {
			break
		}
		index = index*26 + int(r-'A'+1)
	}
	return index
}

func columnName(index int) string {
	if index <= 0 {
		return "?"
	}
	var name []byte
	for index > 0 {
		index--
		name = append([]byte{byte('A' + index%26)}, name...)
		index /= 26
	}
	return string(name)
}
