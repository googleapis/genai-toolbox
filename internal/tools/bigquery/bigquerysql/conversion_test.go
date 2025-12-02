package bigquerysql

import (
	"math/big"
	"reflect"
	"testing"
)

func TestNormalizeValue(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "big.Rat 1/3 (NUMERIC scale 9)",
			input:    new(big.Rat).SetFrac64(1, 3), // 0.33333333333...
			expected: "0.33333333333333333333333333333333333333", // FloatString(38)
		},
		{
			name:     "big.Rat 19/2 (9.5)",
			input:    new(big.Rat).SetFrac64(19, 2),
			expected: "9.5",
		},
		{
			name:     "big.Rat 12341/10 (1234.1)",
			input:    new(big.Rat).SetFrac64(12341, 10),
			expected: "1234.1",
		},
		{
			name:     "big.Rat 10/1 (10)",
			input:    new(big.Rat).SetFrac64(10, 1),
			expected: "10",
		},
		{
			name:     "string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "int",
			input:    123,
			expected: 123,
		},
		{
			name: "nested slice of big.Rat",
			input: []any{
				new(big.Rat).SetFrac64(19, 2),
				new(big.Rat).SetFrac64(1, 4),
			},
			expected: []any{"9.5", "0.25"},
		},
		{
			name: "nested map of big.Rat",
			input: map[string]any{
				"val1": new(big.Rat).SetFrac64(19, 2),
				"val2": new(big.Rat).SetFrac64(1, 2),
			},
			expected: map[string]any{
				"val1": "9.5",
				"val2": "0.5",
			},
		},
		{
			name: "complex nested structure",
			input: map[string]any{
				"list": []any{
					map[string]any{
						"rat": new(big.Rat).SetFrac64(3, 2),
					},
				},
			},
			expected: map[string]any{
				"list": []any{
					map[string]any{
						"rat": "1.5",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeValue(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("normalizeValue() = %v, want %v", got, tt.expected)
			}
		})
	}
}
