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

// LengthWarning describes a barcode whose length is not the standard 8 or 13 digits.
type LengthWarning struct {
	Value    string
	Length   int
	Position int
}

// Result contains the canonical output and diagnostic details for the source data.
type Result struct {
	Values      []string
	Output      string
	Duplicates  []Duplicate
	Nonstandard []LengthWarning
}

// StandardCount returns the number of 8- or 13-digit values in the result.
func (r Result) StandardCount() int {
	return len(r.Values) - len(r.Nonstandard)
}

// Convert extracts standalone ASCII digit sequences and formats them for output.
func Convert(input string) Result {
	return FromValues(Extract(input))
}

// Extract returns standalone ASCII digit sequences in source order.
func Extract(input string) []string {
	return extract(input)
}

// FromValues formats already-extracted values and builds diagnostic metadata.
func FromValues(values []string) Result {
	result := Result{Values: values}
	if len(values) == 0 {
		return result
	}

	counts := make(map[string]int, len(values))
	firstPositions := make(map[string]int, len(values))
	var output strings.Builder

	for index, value := range values {
		position := index + 1
		if index > 0 {
			output.WriteByte(',')
		}
		output.WriteByte('\'')
		output.WriteString(value)
		output.WriteByte('\'')

		counts[value]++
		if _, exists := firstPositions[value]; !exists {
			firstPositions[value] = position
		}
		if len(value) != 8 && len(value) != 13 {
			result.Nonstandard = append(result.Nonstandard, LengthWarning{
				Value:    value,
				Length:   len(value),
				Position: position,
			})
		}
	}

	duplicateAdded := make(map[string]bool)
	for _, value := range values {
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
