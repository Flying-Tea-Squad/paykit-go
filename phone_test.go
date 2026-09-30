package paykit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatKenyanPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Valid local formats with leading zero
		{"07 prefix", "0700000001", "254700000001", false},
		{"07 with spaces", " 0700000001 ", "254700000001", false},
		{"01 prefix (Airtel)", "0100000001", "254100000001", false},
		{"01 with spaces", " 0100000001 ", "254100000001", false},

		// Valid local formats without leading zero
		{"7 prefix no zero", "700000001", "254700000001", false},
		{"1 prefix no zero (Airtel)", "100000001", "254100000001", false},

		// Valid E.164 without +
		{"2547 E.164", "254700000001", "254700000001", false},
		{"2541 E.164", "254100000001", "254100000001", false},

		// Valid full E.164 with +
		{"+2547 full E.164", "+254700000001", "254700000001", false},
		{"+2541 full E.164", "+254100000001", "254100000001", false},
		{"+2547 with spaces", " +254700000001 ", "254700000001", false},

		// Valid formatted with delimiters (spaces, hyphens, dots)
		{"with hyphens", "0700-000-001", "254700000001", false},
		{"with spaces", "0712 345 678", "254712345678", false},
		{"with dots", "0712.345.678", "254712345678", false},
		{"with mixed delimiters", "07-00.000 001", "254700000001", false},

		// All valid Safaricom prefixes (70-79)
		{"70 prefix", "0700000001", "254700000001", false},
		{"71 prefix", "0710000001", "254710000001", false},
		{"72 prefix", "0720000001", "254720000001", false},
		{"73 prefix", "0730000001", "254730000001", false},
		{"74 prefix", "0740000001", "254740000001", false},
		{"75 prefix", "0750000001", "254750000001", false},
		{"76 prefix", "0760000001", "254760000001", false},
		{"77 prefix", "0770000001", "254770000001", false},
		{"78 prefix", "0780000001", "254780000001", false},
		{"79 prefix", "0790000001", "254790000001", false},

		// All valid Airtel/Telkom prefixes (10-19)
		{"10 prefix", "0100000001", "254100000001", false},
		{"11 prefix", "0110000001", "254110000001", false},
		{"12 prefix", "0120000001", "254120000001", false},
		{"13 prefix", "0130000001", "254130000001", false},
		{"14 prefix", "0140000001", "254140000001", false},
		{"15 prefix", "0150000001", "254150000001", false},
		{"16 prefix", "0160000001", "254160000001", false},
		{"17 prefix", "0170000001", "254170000001", false},
		{"18 prefix", "0180000001", "254180000001", false},
		{"19 prefix", "0190000001", "254190000001", false},

		// Invalid: wrong prefix (not Kenyan mobile)
		{"02 prefix invalid", "0200000001", "", true},
		{"03 prefix invalid", "0300000001", "", true},
		{"04 prefix invalid", "0400000001", "", true},
		{"05 prefix invalid", "0500000001", "", true},
		{"06 prefix invalid", "0600000001", "", true},
		{"08 prefix invalid", "0800000001", "", true},
		{"09 prefix invalid", "0900000001", "", true},
		{"20 prefix invalid", "0200000001", "", true},

		// Invalid: wrong length
		{"too short (8 digits)", "070000001", "", true},
		{"too long (11 digits)", "07000000001", "", true},
		{"way too long", "0700000000001", "", true},

		// Invalid: non-numeric
		{"letters", "0700abc001", "", true},
		{"plus in middle", "07+0000001", "", true},

		// Invalid: empty/whitespace
		{"empty string", "", "", true},
		{"whitespace only", "   ", "", true},

		// Invalid: international non-Kenyan
		{"US number", "+15551234567", "", true},
		{"UK number", "+447700900123", "", true},
		{"Tanzania number", "+255700000001", "", true},
		{"Uganda number", "+256700000001", "", true},

		// Invalid: landline prefixes (not mobile)
		{"landline 020", "0200000001", "", true},
		{"landline 041", "0410000001", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := FormatKenyanPhone(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, "", result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestIsValidKenyanPhone(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid 07", "0700000001", true},
		{"valid 01", "0100000001", true},
		{"valid 2547", "254700000001", true},
		{"valid +2547", "+254700000001", true},
		{"valid with hyphens", "0700-000-001", true},
		{"valid with spaces", "0712 345 678", true},
		{"valid with dots", "0712.345.678", true},
		{"invalid prefix", "0200000001", false},
		{"invalid length", "070000001", false},
		{"empty", "", false},
		{"US number", "+15551234567", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidKenyanPhone(tt.input))
		})
	}
}

func TestFormatKenyanPhone_RoundTrip(t *testing.T) {
	// Test that valid numbers normalize consistently regardless of input format
	original := "254700000001"
	formats := []string{
		"0700000001",
		"700000001",
		"254700000001",
		"+254700000001",
		" 0700000001 ",
		" +254700000001 ",
		"0700-000-001",
		"07-00.000 001",
	}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			result, err := FormatKenyanPhone(format)
			require.NoError(t, err)
			assert.Equal(t, original, result)
		})
	}
}
