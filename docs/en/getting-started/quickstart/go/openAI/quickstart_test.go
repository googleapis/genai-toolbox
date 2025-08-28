package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestAgentOutputAndKeywords(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping integration test: GOOGLE_API_KEY environment variable is not set.")
	}

	buildCmd := exec.Command("go", "build", "-o", "quickstart_test_binary", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("FAIL: Failed to compile quickstart.go: %v", err)
	}
	defer os.Remove("quickstart_test_binary")

	cmd := exec.Command("./quickstart_test_binary")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	actualOutput := stdout.String()

	if err != nil {
		t.Fatalf("FAIL: Script execution failed with error: %v\n--- STDERR ---\n%s", err, stderr.String())
	}

	if len(actualOutput) == 0 {
		t.Fatal("FAIL: Script ran successfully but produced no output.")
	}

	goldenFile, err := os.ReadFile("../../golden.txt")
	if err != nil {
		t.Fatalf("FAIL: Could not read golden.txt to check for keywords: %v", err)
	}

	keywords := strings.Split(string(goldenFile), "\n")
	var missingKeywords []string

	outputLower := strings.ToLower(actualOutput)
	for _, keyword := range keywords {
		kw := strings.TrimSpace(keyword)
		if kw == "" {
			continue
		}
		if !strings.Contains(outputLower, strings.ToLower(kw)) {
			missingKeywords = append(missingKeywords, kw)
		}
	}

	if len(missingKeywords) > 0 {
		t.Fatalf("FAIL: The following keywords were missing from the output: [%s]", strings.Join(missingKeywords, ", "))
	}
}
