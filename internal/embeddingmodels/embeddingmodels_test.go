// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package embeddingmodels

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFormatVectorForPgvector(t *testing.T) {
	tcs := []struct {
		name string
		in   []float32
		want any
	}{
		{name: "empty", in: []float32{}, want: "[]"},
		{name: "nil", in: nil, want: "[]"},
		{name: "values", in: []float32{0.1, -0.5, 1.25}, want: "[0.1, -0.5, 1.25]"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatVectorForPgvector(tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected formatted value (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFormatVectorForClickHouse(t *testing.T) {
	tcs := []struct {
		name string
		in   []float32
		want any
	}{
		{name: "empty", in: []float32{}, want: []float32{}},
		{name: "nil", in: nil, want: []float32{}},
		{name: "values", in: []float32{0.1, -0.5, 1.25}, want: []float32{0.1, -0.5, 1.25}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatVectorForClickHouse(tc.in)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("unexpected formatted value (-want +got):\n%s", diff)
			}
		})
	}
}
