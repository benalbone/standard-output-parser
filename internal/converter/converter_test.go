package converter

import (
	"reflect"
	"testing"
)

func TestConvertCommonInputFormats(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		values []string
		output string
	}{
		{
			name:   "quoted csv and crlf",
			input:  "\"00000000\",\"0000000000001\"\r\n\"12345678\",\"123456789012\"\r\n",
			values: []string{"00000000", "0000000000001", "12345678", "0123456789012"},
			output: "'00000000','0000000000001','12345678','0123456789012'",
		},
		{
			name:   "pipes whitespace tabs and mixed delimiters",
			input:  "00000000|0000000000001  12345678\t9999999999999;\n87654321",
			values: []string{"00000000", "0000000000001", "12345678", "9999999999999", "87654321"},
			output: "'00000000','0000000000001','12345678','9999999999999','87654321'",
		},
		{
			name:   "leading zeros",
			input:  "00000001,0000000000002",
			values: []string{"00000001", "0000000000002"},
			output: "'00000001','0000000000002'",
		},
		{
			name:   "empty",
			input:  "",
			values: nil,
			output: "",
		},
		{
			name:   "no numeric tokens",
			input:  "Barcode,Description\nABC,Example",
			values: nil,
			output: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Convert(test.input)
			if !reflect.DeepEqual(result.Values, test.values) {
				t.Fatalf("values = %#v, want %#v", result.Values, test.values)
			}
			if result.Output != test.output {
				t.Fatalf("output = %q, want %q", result.Output, test.output)
			}
		})
	}
}

func TestConvertIgnoresDigitsEmbeddedInWords(t *testing.T) {
	result := Convert("header SKU12345678ABC 12345678 abc1234567890123 cafe12345678 café12345678")

	want := []string{"12345678"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("values = %#v, want %#v", result.Values, want)
	}
}

func TestConvertFiltersUnsupportedLengthsAndNormalizesTwelveDigits(t *testing.T) {
	result := Convert("1,123456,12345678,123456789012,1234567890123,12345678901234")

	want := []string{"12345678", "0123456789012", "1234567890123"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("values = %#v, want %#v", result.Values, want)
	}
	if result.Output != "'12345678','0123456789012','1234567890123'" {
		t.Fatalf("output = %q", result.Output)
	}

	wantSkipped := []SkippedValue{
		{Value: "1", Length: 1, Position: 1},
		{Value: "123456", Length: 6, Position: 2},
		{Value: "12345678901234", Length: 14, Position: 6},
	}
	if !reflect.DeepEqual(result.Skipped, wantSkipped) {
		t.Fatalf("skipped = %#v, want %#v", result.Skipped, wantSkipped)
	}

	wantNormalized := []NormalizedValue{
		{Original: "123456789012", Value: "0123456789012", Position: 4},
	}
	if !reflect.DeepEqual(result.Normalized, wantNormalized) {
		t.Fatalf("normalized = %#v, want %#v", result.Normalized, wantNormalized)
	}
}

func TestConvertReportsDuplicatesAfterNormalization(t *testing.T) {
	result := Convert("123456789012|0123456789012")

	wantValues := []string{"0123456789012", "0123456789012"}
	if !reflect.DeepEqual(result.Values, wantValues) {
		t.Fatalf("values = %#v, want %#v", result.Values, wantValues)
	}

	wantDuplicates := []Duplicate{
		{Value: "0123456789012", Count: 2, FirstPosition: 1},
	}
	if !reflect.DeepEqual(result.Duplicates, wantDuplicates) {
		t.Fatalf("duplicates = %#v, want %#v", result.Duplicates, wantDuplicates)
	}
}

func TestConvertPreservesAndReportsDuplicates(t *testing.T) {
	result := Convert("00000000|1234567890123|00000000|00000001|1234567890123|00000000")

	wantValues := []string{
		"00000000",
		"1234567890123",
		"00000000",
		"00000001",
		"1234567890123",
		"00000000",
	}
	if !reflect.DeepEqual(result.Values, wantValues) {
		t.Fatalf("values = %#v, want %#v", result.Values, wantValues)
	}

	wantDuplicates := []Duplicate{
		{Value: "00000000", Count: 3, FirstPosition: 1},
		{Value: "1234567890123", Count: 2, FirstPosition: 2},
	}
	if !reflect.DeepEqual(result.Duplicates, wantDuplicates) {
		t.Fatalf("duplicates = %#v, want %#v", result.Duplicates, wantDuplicates)
	}
}

func TestConvertTreatsSymbolsAsSeparators(t *testing.T) {
	result := Convert("'12345678'/\"1234567890123\"_87654321")

	want := []string{"12345678", "1234567890123", "87654321"}
	if !reflect.DeepEqual(result.Values, want) {
		t.Fatalf("values = %#v, want %#v", result.Values, want)
	}
}
