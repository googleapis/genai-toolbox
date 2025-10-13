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

package prompts_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/prompts"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestSubstituteMessages(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		arguments := prompts.Arguments{
			{Parameter: tools.NewStringParameter("name", "The name to use.")},
			{Parameter: tools.NewStringParameterWithRequired("location", "The location.", false)},
		}
		messages := []prompts.Message{
			{Role: "user", Content: "Hello, my name is {{.name}} and I am in {{.location}}."},
			{Role: "assistant", Content: "Nice to meet you, {{.name}}!"},
		}
		argValues := tools.ParamValues{
			{Name: "name", Value: "Alice"},
			{Name: "location", Value: "Wonderland"},
		}

		want := []prompts.Message{
			{Role: "user", Content: "Hello, my name is Alice and I am in Wonderland."},
			{Role: "assistant", Content: "Nice to meet you, Alice!"},
		}

		got, err := prompts.SubstituteMessages(messages, arguments, argValues)
		if err != nil {
			t.Fatalf("SubstituteMessages() failed: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("SubstituteMessages() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("FailureInvalidTemplate", func(t *testing.T) {
		arguments := prompts.Arguments{}
		messages := []prompts.Message{
			{Content: "This has an {{.unclosed template"},
		}
		argValues := tools.ParamValues{}

		_, err := prompts.SubstituteMessages(messages, arguments, argValues)
		if err == nil {
			t.Fatal("expected an error for invalid template, but got nil")
		}
		wantErr := "unexpected <template> in operand"
		if !strings.Contains(err.Error(), wantErr) {
			t.Errorf("error mismatch:\n  want to contain: %q\n  got: %q", wantErr, err.Error())
		}
	})
}
