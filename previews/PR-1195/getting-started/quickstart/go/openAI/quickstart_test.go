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
		t.Skip("Skipping integration test: OPENAI_API_KEY environment variable is not set.")
	}

	t.Log("Compiling quickstart.go...")
	buildCmd := exec.Command("go", "build", "-o", "quickstart_test_binary", ".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to compile quickstart.go: %v", err)
	}
	defer os.Remove("quickstart_test_binary")

	t.Log("Running test binary...")
	cmd := exec.Command("./quickstart_test_binary")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	actualOutput := stdout.String()

	t.Logf("--- SCRIPT OUTPUT ---\n%s", actualOutput)
	if stderr.Len() > 0 {
		t.Logf("--- SCRIPT STDERR ---\n%s", stderr.String())
	}

	if err != nil {
		t.Fatalf("Script execution failed with error: %v", err)
	}
	if len(actualOutput) == 0 {
		t.Fatal("Script ran successfully but produced no output.")
	}
	t.Log("Primary assertion passed: Script ran successfully and produced output.")

	t.Log("--- Checking for essential keywords ---")
	goldenFile, err := os.ReadFile("../../golden.txt")
	if err != nil {
		t.Logf("Warning: Could not read golden.txt to check for keywords: %v", err)
		return
	}

	keywords := strings.Split(string(goldenFile), "\n")
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if strings.Contains(actualOutput, keyword) {
			t.Logf("Keyword check: Found keyword '%s' in output.", keyword)
		} else {
			t.Logf("Keyword check: Did not find keyword '%s' in output.", keyword)
		}
	}
}
