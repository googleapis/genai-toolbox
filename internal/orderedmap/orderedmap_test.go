// Copyright 2024 Google LLC
//
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
package orderedmap

import (
	"encoding/json"
	"testing"
)

func TestRow_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		row     Row
		want    string
		wantErr bool
	}{
		{
			name: "Simple row",
			row: Row{
				Columns: []Column{
					{Name: "A", Value: 1},
					{Name: "B", Value: "two"},
					{Name: "C", Value: true},
				},
			},
			want:    `{"A":1,"B":"two","C":true}`,
			wantErr: false,
		},
		{
			name: "Row with different order",
			row: Row{
				Columns: []Column{
					{Name: "C", Value: true},
					{Name: "A", Value: 1},
					{Name: "B", Value: "two"},
				},
			},
			want:    `{"C":true,"A":1,"B":"two"}`,
			wantErr: false,
		},
		{
			name:    "Empty row",
			row:     Row{},
			want:    `{}`,
			wantErr: false,
		},
		{
			name: "Row with nil value",
			row: Row{
				Columns: []Column{
					{Name: "A", Value: 1},
					{Name: "B", Value: nil},
				},
			},
			want:    `{"A":1,"B":null}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.row)
			if (err != nil) != tt.wantErr {
				t.Errorf("Row.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.want {
				t.Errorf("Row.MarshalJSON() = %s, want %s", string(got), tt.want)
			}
		})
	}
}
