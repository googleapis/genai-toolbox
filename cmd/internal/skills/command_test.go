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

package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/cmd/internal"
	_ "github.com/googleapis/genai-toolbox/internal/sources/sqlite"
	_ "github.com/googleapis/genai-toolbox/internal/tools/sqlite/sqlitesql"
	"github.com/spf13/cobra"
)

func invokeCommand(args []string) (string, error) {
	parentCmd := &cobra.Command{Use: "toolbox"}

	buf := new(bytes.Buffer)
	opts := internal.NewToolboxOptions(internal.WithIOStreams(buf, buf))
	internal.PersistentFlags(parentCmd, opts)

	cmd := NewCommand(opts)
	parentCmd.AddCommand(cmd)
	parentCmd.SetArgs(args)

	err := parentCmd.Execute()
	return buf.String(), err
}

func TestGenerateSkill(t *testing.T) {
	// Create a temporary directory for tests
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "skills")

	// Create a tools.yaml file with a sqlite tool
	toolsFileContent := `
sources:
  my-sqlite:
    kind: sqlite
    database: test.db
tools:
  hello-sqlite:
    kind: sqlite-sql
    source: my-sqlite
    description: "hello tool"
    statement: "SELECT 'hello' as greeting"
`

	toolsFilePath := filepath.Join(tmpDir, "tools.yaml")
	if err := os.WriteFile(toolsFilePath, []byte(toolsFileContent), 0644); err != nil {
		t.Fatalf("failed to write tools file: %v", err)
	}

	args := []string{
		"skills-generate",
		"--tools-file", toolsFilePath,
		"--output-dir", outputDir,
		"--name", "hello-sqlite",
		"--description", "hello tool",
	}

	got, err := invokeCommand(args)
	if err != nil {
		t.Fatalf("command failed: %v\nOutput: %s", err, got)
	}

	// Verify generated directory structure
	skillPath := filepath.Join(outputDir, "hello-sqlite")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatalf("skill directory not created: %s", skillPath)
	}

	// Check SKILL.md
	skillMarkdown := filepath.Join(skillPath, "SKILL.md")
	content, err := os.ReadFile(skillMarkdown)
	if err != nil {
		t.Fatalf("failed to read SKILL.md: %v", err)
	}

	expectedFrontmatter := `---
name: hello-sqlite
description: hello tool
---`
	if !strings.HasPrefix(string(content), expectedFrontmatter) {
		t.Errorf("SKILL.md does not have expected frontmatter format.\nExpected prefix:\n%s\nGot:\n%s", expectedFrontmatter, string(content))
	}

	if !strings.Contains(string(content), "## Usage") {
		t.Errorf("SKILL.md does not contain '## Usage' section")
	}

	if !strings.Contains(string(content), "## Scripts") {
		t.Errorf("SKILL.md does not contain '## Scripts' section")
	}

	if !strings.Contains(string(content), "### hello-sqlite") {
		t.Errorf("SKILL.md does not contain '### hello-sqlite' tool header")
	}

	// Check script file
	scriptFilename := "hello-sqlite.js"
	scriptPath := filepath.Join(skillPath, "scripts", scriptFilename)
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Fatalf("script file not created: %s", scriptPath)
	}

	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read script file: %v", err)
	}
	if !strings.Contains(string(scriptContent), "hello-sqlite") {
		t.Errorf("script file does not contain expected tool name")
	}

	// Check assets
	assetPath := filepath.Join(skillPath, "assets", "hello-sqlite.yaml")
	if _, err := os.Stat(assetPath); os.IsNotExist(err) {
		t.Fatalf("asset file not created: %s", assetPath)
	}
	assetContent, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("failed to read asset file: %v", err)
	}
	if !strings.Contains(string(assetContent), "hello-sqlite") {
		t.Errorf("asset file does not contain expected tool name")
	}
}

func TestGenerateSkill_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "skills")

	args := []string{
		"skills-generate",
		"--output-dir", outputDir,
		"--name", "test",
		"--description", "test",
	}

	_, err := invokeCommand(args)
	if err == nil {
		t.Fatal("expected command to fail when no configuration is provided and tools.yaml is missing")
	}

	// Should not have created the directory if no config was processed
	if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
		t.Errorf("output directory should not have been created")
	}
}

func TestGenerateSkill_MissingArguments(t *testing.T) {
	tmpDir := t.TempDir()
	toolsFilePath := filepath.Join(tmpDir, "tools.yaml")
	if err := os.WriteFile(toolsFilePath, []byte("tools: {}"), 0644); err != nil {
		t.Fatalf("failed to write tools file: %v", err)
	}

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing name",
			args: []string{"skills-generate", "--tools-file", toolsFilePath, "--description", "test"},
		},
		{
			name: "missing description",
			args: []string{"skills-generate", "--tools-file", toolsFilePath, "--name", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := invokeCommand(tt.args)
			if err == nil {
				t.Fatalf("expected command to fail due to missing arguments, but it succeeded\nOutput: %s", got)
			}
		})
	}
}
