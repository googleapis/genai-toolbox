//go:build !integration

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
	"errors"
	"io"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandOptions(t *testing.T) {
	w := io.Discard
	tcs := []struct {
		desc    string
		isValid func(*Command) error
		option  Option
	}{
		{
			desc: "with logger",
			isValid: func(c *Command) error {
				if c.outStream != w || c.errStream != w {
					return errors.New("loggers do not match")
				}
				return nil
			},
			option: WithStreams(w, w),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := invokeProxyWithOption(tc.option)
			if err != nil {
				t.Fatal(err)
			}
			if err := tc.isValid(got); err != nil {
				t.Errorf("option did not initialize command correctly: %v", err)
			}
		})
	}
}

func invokeProxyWithOption(o Option) (*Command, error) {
	c := NewCommand(o)
	// Keep the test output quiet
	c.SilenceUsage = true
	c.SilenceErrors = true
	// Disable execute behavior
	c.RunE = func(*cobra.Command, []string) error {
		return nil
	}

	err := c.Execute()
	return c, err
}
