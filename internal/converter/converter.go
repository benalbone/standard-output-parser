package converter

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Duplicate describes a value that appears more than once in the input.
type Duplicate struct {
	Value         string
	Count         int
	FirstPosition int
}

// SkippedValue describes a standalone numeric value ignored because its length is unsupported.
type SkippedValue struct {
	Value    string
	Length   int
	Position int
}

// NormalizedValue describes a 12-digit barcode converted to a 13-digit EAN by adding a leading zero.
type NormalizedValue struct {
	Original string
	Value    string
	Position int
}

// Result contains the canonical output and diagnostic details for the source data.
type Result struct {
	Values     []string
	Output     string
	Duplicates []Duplicate
	Skipped    []SkippedValue
	Normalized []NormalizedValue
}

// Convert extracts supported standalone ASCII barcode sequences and formats them for output.
func Convert(input string) Result {
	return FromValues(Extract(input))
}

// Extract returns standalone ASCII digit sequences in source order.
func Extract(input string) []string {
	return extract(input)
}

// FromValues filters and formats already-extracted values and builds diagnostic metadata.
func FromValues(values []string) Result {
	result := Result{}
	if len(values) == 0 {
		return result
	}

	counts := make(map[string]int)
	firstPositions := make(map[string]int)
	var output strings.Builder

	for index, value := range values {
		position := index + 1

		formatted, normalized := supportedBarcode(value)
		if formatted == "" {
			result.Skipped = append(result.Skipped, SkippedValue{
				Value:    value,
				Length:   len(value),
				Position: position,
			})
			continue
		}
		if normalized {
			result.Normalized = append(result.Normalized, NormalizedValue{
				Original: value,
				Value:    formatted,
				Position: position,
			})
		}

		if len(result.Values) > 0 {
			output.WriteByte(',')
		}
		output.WriteByte('\'')
		output.WriteString(formatted)
		output.WriteByte('\'')

		result.Values = append(result.Values, formatted)
		counts[formatted]++
		if _, exists := firstPositions[formatted]; !exists {
			firstPositions[formatted] = len(result.Values)
		}
	}

	duplicateAdded := make(map[string]bool)
	for _, value := range result.Values {
		if counts[value] > 1 && !duplicateAdded[value] {
			result.Duplicates = append(result.Duplicates, Duplicate{
				Value:         value,
				Count:         counts[value],
				FirstPosition: firstPositions[value],
			})
			duplicateAdded[value] = true
		}
	}

	result.Output = output.String()
	return result
}

func supportedBarcode(value string) (string, bool) {
	switch len(value) {
	case 8, 13:
		return value, false
	case 12:
		return "0" + value, true
	default:
		return "", false
	}
}

func extract(input string) []string {
	var values []string

	for index := 0; index < len(input); {
		if !isASCIIDigit(input[index]) {
			_, size := utf8.DecodeRuneInString(input[index:])
			if size == 0 {
				size = 1
			}
			index += size
			continue
		}

		start := index
		for index < len(input) && isASCIIDigit(input[index]) {
			index++
		}

		if hasAlphanumericBefore(input, start) || hasAlphanumericAfter(input, index) {
			continue
		}
		values = append(values, input[start:index])
	}

	return values
}

func hasAlphanumericBefore(input string, index int) bool {
	if index == 0 {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(input[:index])
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

func hasAlphanumericAfter(input string, index int) bool {
	if index == len(input) {
		return false
	}
	r, _ := utf8.DecodeRuneInString(input[index:])
	return unicode.IsLetter(r) || unicode.IsNumber(r)
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}
