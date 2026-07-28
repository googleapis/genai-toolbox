// Copyright 2026 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestConvertNumbers(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want any
	}{
		{"integer", json.Number("123"), int64(123)},
		{"negative integer", json.Number("-7"), int64(-7)},
		{"decimal", json.Number("1.5"), float64(1.5)},
		// Exponent-form numbers contain no ".", so deciding the type on the
		// presence of a "." alone sent them to Int64, which rejects exponent
		// syntax and returned an error. json.Marshal emits this form for
		// large or small floats (e.g. 1e-07, 1e+21), so it is reachable.
		{"positive exponent", json.Number("1e5"), float64(100000)},
		{"uppercase exponent", json.Number("1E5"), float64(100000)},
		{"negative exponent", json.Number("1e-07"), float64(1e-07)},
		{"large exponent beyond int64", json.Number("1e+21"), float64(1e21)},
		// An integer literal larger than int64 also falls back to float
		// instead of erroring.
		{"integer overflowing int64", json.Number("99999999999999999999"), float64(99999999999999999999)},
		{"string passthrough", "hello", "hello"},
		{"bool passthrough", true, true},
		{
			"nested map and slice",
			map[string]any{"a": json.Number("2e3"), "b": []any{json.Number("4")}},
			map[string]any{"a": float64(2000), "b": []any{int64(4)}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ConvertNumbers(tc.in)
			if err != nil {
				t.Fatalf("ConvertNumbers(%v) returned error: %v", tc.in, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ConvertNumbers(%v) = %#v (%T), want %#v (%T)", tc.in, got, got, tc.want, tc.want)
			}
		})
	}
}
