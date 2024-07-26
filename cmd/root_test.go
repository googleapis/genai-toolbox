// Copyright 2024 Google LLC
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

package cmd_test

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/cmd"
)

func TestVersion(t *testing.T) {
	data, err := os.ReadFile("version.txt")
	if err != nil {
		t.Fatalf("failed to read version.txt: %v", err)
	}
	want := strings.TrimSpace(string(data))

	// run command with flag
	b := bytes.NewBufferString("")
	cmd := cmd.NewCommand()
	cmd.SetArgs([]string{"--version"})
	cmd.SetOut(b)

	err = cmd.Execute()
	if err != nil {
		t.Fatalf("unable to execute command: %q", err)
	}

	out, err := io.ReadAll(b)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)

	if !strings.Contains(got, want) {
		t.Errorf("cli did not return correct version: want %q, got %q", want, got)
	}
}
