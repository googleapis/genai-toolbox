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

package cmd

import (
	"bytes"
	_ "embed"
	"os"
	"strings"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/spf13/cobra"
)

func invokeCommand(args []string) (*Command, string, error) {
	c := NewCommand()

	// Keep the test output quiet
	c.SilenceUsage = true
	c.SilenceErrors = true

	// Capture output
	buf := new(bytes.Buffer)
	c.SetOut(buf)
	c.SetErr(buf)
	c.SetArgs(args)

	// Disable execute behavior
	c.RunE = func(*cobra.Command, []string) error {
		return nil
	}

	err := c.Execute()

	return c, buf.String(), err
}

func TestVersion(t *testing.T) {
	data, err := os.ReadFile("version.txt")
	if err != nil {
		t.Fatalf("failed to read version.txt: %v", err)
	}
	want := strings.TrimSpace(string(data))

	_, got, err := invokeCommand([]string{"--version"})
	if err != nil {
		t.Fatalf("error invoking command: %s", err)
	}

	if !strings.Contains(got, want) {
		t.Errorf("cli did not return correct version: want %q, got %q", want, got)
	}
}

func TestFlags(t *testing.T) {
	tcs := []struct {
		desc string
		args []string
		want server.Config
	}{
		{
			desc: "default values",
			args: []string{},
			want: server.Config{
				Address: "127.0.0.1",
				Port:    5000,
			},
		},
		{
			desc: "address short",
			args: []string{"-a", "127.0.1.1"},
			want: server.Config{
				Address: "127.0.1.1",
				Port:    5000,
			},
		},
		{
			desc: "address long",
			args: []string{"--address", "0.0.0.0"},
			want: server.Config{
				Address: "0.0.0.0",
				Port:    5000,
			},
		},
		{
			desc: "port short",
			args: []string{"-p", "5052"},
			want: server.Config{
				Address: "127.0.0.1",
				Port:    5052,
			},
		},
		{
			desc: "port long",
			args: []string{"--port", "5050"},
			want: server.Config{
				Address: "127.0.0.1",
				Port:    5050,
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			c, _, err := invokeCommand(tc.args)
			if err != nil {
				t.Fatalf("unexpected error invoking command: %s", err)
			}

			if c.cfg != tc.want {
				t.Fatalf("got %v, want %v", c.cfg, tc.want)
			}
		})
	}
}
