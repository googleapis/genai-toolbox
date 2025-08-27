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

package jsonrpc

import (
	"reflect"
	"testing"
)

func TestNewError(t *testing.T) {
	var id interface{} = "foo"
	code := 111
	message := "foo bar"
	want := JSONRPCError{
		Jsonrpc: "2.0",
		Id:      "foo",
		Error: Error{
			Code:    111,
			Message: "foo bar",
		},
	}

	got := NewError(id, code, message, nil)
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("unexpected error: got %+v, want %+v", got, want)
	}
}
