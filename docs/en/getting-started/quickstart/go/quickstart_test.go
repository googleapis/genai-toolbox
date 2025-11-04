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
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuickstartSample(t *testing.T) {
	framework := os.Getenv("ORCH_NAME")
	if framework == "" {
		t.Skip("Skipping test: ORCH_NAME environment variable is not set.")
	}

	t.Logf("--- Testing: %s ---", framework)

	if framework == "openAI" {
		if os.Getenv("OPENAI_API_KEY") == "" {
			t.Skip("Skipping test: OPENAI_API_KEY environment variable is not set for openAI framework.")
		}
	} else if framework == "adkgo" {
		t.Skip("Skipping test: ADK Go module is not publicly available")
	} else {
		if os.Getenv("GOOGLE_API_KEY") == "" {
			t.Skipf("Skipping test for %s: GOOGLE_API_KEY environment variable is not set.", framework)
		}
	}

	sampleDir := filepath.Join(".", framework)
	if _, err := os.Stat(sampleDir); os.IsNotExist(err) {
		t.Fatalf("Test setup failed: directory for framework '%s' not found.", framework)
	}

	cmd := exec.Command("go", "run", ".")
	cmd.Dir = sampleDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	actualOutput := stdout.String()

	if err != nil {
		t.Fatalf("Script execution failed with error: %v\n--- STDERR ---\n%s", err, stderr.String())
	}
	if len(actualOutput) == 0 {
		t.Fatal("Script ran successfully but produced no output.")
	}

	goldenFile, err := os.ReadFile("../golden.txt")
	if err != nil {
		t.Fatalf("Could not read golden.txt to check for keywords: %v", err)
	}

	keywords := strings.Split(string(goldenFile), "\n")
	var missingKeywords []string
	outputLower := strings.ToLower(actualOutput)

	for _, keyword := range keywords {
		kw := strings.TrimSpace(keyword)
		if kw != "" && !strings.Contains(outputLower, strings.ToLower(kw)) {
			missingKeywords = append(missingKeywords, kw)
		}
	}

	if len(missingKeywords) > 0 {
		t.Fatalf("FAIL: The following keywords were missing from the output: [%s]", strings.Join(missingKeywords, ", "))
	}
}
