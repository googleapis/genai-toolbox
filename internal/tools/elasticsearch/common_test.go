// Copyright 2025 Google LLC
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

package elasticsearch

import (
	"reflect"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestRetrieveIndices(t *testing.T) {
	type args struct {
		params tools.ParamValues
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "single index",
			args: args{
				params: tools.ParamValues{
					{"index", "my-index"},
				},
			},
			want:    []string{"my-index"},
			wantErr: false,
		},
		{
			name: "multiple indices",
			args: args{
				params: tools.ParamValues{
					{"indices", []any{"index1", "index2"}},
				},
			},
			want:    []string{"index1", "index2"},
			wantErr: false,
		},
		{
			name: "missing indices",
			args: args{
				params: tools.ParamValues{
					{"missing", "my-index"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid index type",
			args: args{
				params: tools.ParamValues{
					{"indices", "not-an-array"},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid single index type",
			args: args{
				params: tools.ParamValues{
					{"index", 12345},
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RetrieveIndices(tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("RetrieveIndices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RetrieveIndices() got = %v, want %v", got, tt.want)
			}
		})
	}
}
