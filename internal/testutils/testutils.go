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

package testutils

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/googleapis/mcp-toolbox/internal/log"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

// MockVersionString is used as a temporary version string in tests
const MockVersionString = "0.0.0"

// formatYaml is a utility function for stripping out tabs in multiline strings
func FormatYaml(in string) []byte {
	// removes any leading indentation(tabs)
	in = strings.ReplaceAll(in, "\n\t", "\n ")
	// converts remaining indentation
	in = strings.ReplaceAll(in, "\t", "  ")
	return []byte(in)
}

// ContextWithNewLogger create a new context with new logger
func ContextWithNewLogger() (context.Context, error) {
	ctx := context.Background()
	logger, err := log.NewStdLogger(os.Stdout, os.Stderr, "info")
	if err != nil {
		return nil, fmt.Errorf("unable to create logger: %s", err)
	}
	return util.WithLogger(ctx, logger), nil
}

// ContextWithUserAgent creates a new context with a specified user agent string.
func ContextWithUserAgent(ctx context.Context, userAgent string) context.Context {
	return util.WithUserAgent(ctx, userAgent)
}

// WaitForString waits until the server logs a single line that matches the provided regex.
// returns the output of whatever the server sent so far.
func WaitForString(ctx context.Context, re *regexp.Regexp, pr io.ReadCloser) (string, error) {
	in := bufio.NewReader(pr)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// read lines in background, sending result of each read over a channel
	// this allows us to use in.ReadString without blocking
	type result struct {
		s   string
		err error
	}
	output := make(chan result)
	go func() {
		defer close(output)
		for {
			select {
			case <-ctx.Done():
				// if the context is canceled, the orig thread will send back the error
				// so we can just exit the goroutine here
				return
			default:
				// otherwise read a line from the output
				s, err := in.ReadString('\n')
				if err != nil {
					output <- result{err: err}
					return
				}
				output <- result{s: s}
				// if that last string matched, exit the goroutine
				if re.MatchString(s) {
					return
				}
			}
		}
	}()

	// collect the output until the ctx is canceled, an error was hit,
	// or match was found (which is indicated the channel is closed)
	var sb strings.Builder
	for {
		select {
		case <-ctx.Done():
			// if ctx is done, return that error
			return sb.String(), ctx.Err()
		case o, ok := <-output:
			if !ok {
				// match was found!
				return sb.String(), nil
			}
			if o.err != nil {
				// error was found!
				return sb.String(), o.err
			}
			sb.WriteString(o.s)
		}
	}
}

var MockTool1 = NewMockTool("no_params", "", []parameters.Parameter{}, false, false)

var MockTool2 = NewMockTool(
	"some_params",
	"",
	parameters.Parameters{
		parameters.NewIntParameter("param1", "This is the first parameter."),
		parameters.NewIntParameter("param2", "This is the second parameter."),
	}, false, false)

var MockTool3 = NewMockTool(
	"array_param", "some description",
	parameters.Parameters{
		parameters.NewArrayParameter("my_array", "this param is an array of strings", parameters.NewStringParameter("my_string", "string item")),
	}, false, false)

var MockTool4 = NewMockTool("unauthorized_tool", "", []parameters.Parameter{}, true, false)

var MockTool5 = NewMockTool("require_client_auth_tool", "", []parameters.Parameter{}, false, true)

var MockPrompt1 = NewMockPrompt("prompt1", "", prompts.Arguments{})

var MockPrompt2 = NewMockPrompt("prompt2", "", prompts.Arguments{
	{Parameter: parameters.NewStringParameter("arg1", "This is the first argument.")},
})

// SetUpResources setups resources to test against
func SetUpResources(t *testing.T, mockTools []MockTool, mockPrompts []MockPrompt) (map[string]tools.Tool, map[string]tools.Toolset, map[string]prompts.Prompt, map[string]prompts.Promptset) {
	toolsMap := make(map[string]tools.Tool)
	var allTools []string
	for _, tool := range mockTools {
		toolsMap[tool.Name] = tool
		allTools = append(allTools, tool.Name)
	}

	toolsets := make(map[string]tools.Toolset)
	if len(allTools) > 0 {
		for name, l := range map[string][]string{
			"":           allTools,
			"tool1_only": {allTools[0]},
			"tool2_only": {allTools[1]},
		} {
			tc := tools.ToolsetConfig{Name: name, ToolNames: l}
			m, err := tc.Initialize(MockVersionString, toolsMap)
			if err != nil {
				t.Fatalf("unable to initialize toolset %q: %s", name, err)
			}
			toolsets[name] = m
		}
	}

	promptsMap := make(map[string]prompts.Prompt)
	var allPrompts []string
	for _, prompt := range mockPrompts {
		promptsMap[prompt.Name] = prompt
		allPrompts = append(allPrompts, prompt.Name)
	}

	promptsets := make(map[string]prompts.Promptset)
	if len(allPrompts) > 0 {
		psc := prompts.PromptsetConfig{Name: "", PromptNames: allPrompts}
		ps, err := psc.Initialize(MockVersionString, promptsMap)
		if err != nil {
			t.Fatalf("unable to initialize default promptset: %s", err)
		}
		promptsets[""] = ps
	}

	return toolsMap, toolsets, promptsMap, promptsets
}
