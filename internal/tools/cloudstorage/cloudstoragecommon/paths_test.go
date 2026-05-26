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

package cloudstoragecommon

import (
	"path/filepath"
	"testing"
)

func TestValidateLocalPath(t *testing.T) {
	base := t.TempDir()
	dotPath := base + string(filepath.Separator) + "." + string(filepath.Separator) + "a" + string(filepath.Separator) + "b" + string(filepath.Separator) + "c"
	containsDotsPath := filepath.Join(base, "foo..bar", "baz")

	tcs := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: filepath.Join(base, "out.bin"), want: filepath.Join(base, "out.bin")},
		{in: dotPath, want: filepath.Join(base, "a", "b", "c")},
		{in: containsDotsPath, want: containsDotsPath},

		{in: "", wantErr: true},
		{in: "relative/path", wantErr: true},
		{in: "../escape", wantErr: true},
		{in: "/legit/../../etc/passwd", wantErr: true},
		{in: `C:\legit\..\secret.txt`, wantErr: true},
		{in: `C:/legit/../secret.txt`, wantErr: true},
		{in: "..", wantErr: true},
	}
	for _, tc := range tcs {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ValidateLocalPath(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
