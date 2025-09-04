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

package memoryrepo

import (
	"reflect"
	"sort"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/repository"
)

// sortData sorts the datas based on name
func sortData(datas []repository.Resource) []repository.Resource {
	sort.Slice(datas, func(i, j int) bool {
		return datas[i].Name < datas[j].Name // Sorts by Name in ascending order
	})
	return datas
}

func TestRepository(t *testing.T) {
	sourceR, _, _, _ := New()

	mockSource := repository.Resource{Name: "my-source", Type: "source-type", Configuration: `{"type": "source-type", "host": "127.0.0.1"}`, IsActive: true}
	mockSource2 := repository.Resource{Name: "my-source2", Type: "source-type", Configuration: `{"type": "source-type", "host": "127.0.0.1"}`, IsActive: true}

	// run test for Create
	tcsCreate := []struct {
		name      string
		data      any
		isErr     bool
		errString string
	}{
		{
			name: "create mockSource",
			data: mockSource,
		},
		{
			name: "create mockSource2",
			data: mockSource2,
		},
		{
			name:      "insert entity with same name",
			data:      mockSource,
			isErr:     true,
			errString: "name my-source already exists",
		},
	}
	for _, tc := range tcsCreate {
		t.Run(tc.name, func(t *testing.T) {
			err := sourceR.Create(tc.data.(repository.Resource))
			if tc.isErr {
				if err == nil {
					t.Fatalf("should be throwing an error")
				}
				if err.Error() != tc.errString {
					t.Fatalf("unexpected error string: got %s, want %s", err, tc.errString)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}
		})
	}

	// test Get()
	tcsGet := []struct {
		name       string
		sourceName string
		data       any
		isErr      bool
		errString  string
	}{
		{
			name:       "get my-source",
			sourceName: "my-source",
			data:       mockSource,
		},
		{
			name:       "get nonexisting",
			sourceName: "nonexisting",
			isErr:      true,
			errString:  "unable to retrieve data: nonexisting",
		},
	}
	for _, tc := range tcsGet {
		t.Run(tc.name, func(t *testing.T) {
			d, err := sourceR.Get(tc.sourceName)
			if tc.isErr {
				if err == nil {
					t.Fatalf("should be throwing an error")
				}
				if err.Error() != tc.errString {
					t.Fatalf("unexpected error string: got %s, want %s", err, tc.errString)
				}
			} else {
				if !reflect.DeepEqual(d, tc.data) {
					t.Fatalf("unexpected data: got %+v, want %+v", d, tc.data)
				}
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}
		})
	}

	// test GetAll()
	allMocks := []repository.Resource{mockSource, mockSource2}
	datas, err := sourceR.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(allMocks, sortData(datas)) {
		t.Fatalf("unexpected error: got %+v, want %+v", allMocks, datas)
	}

	// test Update()
	mockSource2New := mockSource2
	mockSource2New.IsActive = false
	err = sourceR.Update(mockSource2New)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// check updated repo
	allMocks = []repository.Resource{mockSource, mockSource2New}
	datas, err = sourceR.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(allMocks, sortData(datas)) {
		t.Fatalf("unexpected error: got %+v, want %+v", allMocks, datas)
	}

	// test Delete()
	err = sourceR.Delete("my-source2")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// check updated repo
	allMocks = []repository.Resource{mockSource}
	datas, err = sourceR.GetAll()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(allMocks, sortData(datas)) {
		t.Fatalf("unexpected error: got %+v, want %+v", allMocks, datas)
	}
}
