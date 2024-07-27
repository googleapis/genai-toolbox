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
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/spf13/cobra"
)

var (
	// versionString indicates the version of this library.
	//go:embed version.txt
	versionString string
	// metadataString indicates additional build or distribution metadata.
	metadataString string
)

func init() {
	versionString = semanticVersion()
}

// semanticVersion returns the version of the CLI including a compile-time metadata.
func semanticVersion() string {
	v := strings.TrimSpace(versionString)
	if metadataString != "" {
		v += "+" + metadataString
	}
	return v
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := NewCommand().Execute(); err != nil {
		exit := 1
		os.Exit(exit)
	}
}

// Command represents an invocation of the CLI.
type Command struct {
	*cobra.Command

	cfg server.Config
}

// NewCommand returns a Command object representing an invocation of the CLI.
func NewCommand() *Command {
	c := &Command{
		Command: &cobra.Command{
			Use:     "toolbox",
			Version: versionString,
		},
	}

	flags := c.Flags()
	flags.StringVarP(&c.cfg.Address, "address", "a", "127.0.0.1", "Address of the interface the server will listen on.")
	flags.IntVarP(&c.cfg.Port, "port", "p", 5000, "Port the server will listen on.")

	// wrap RunE command so that we have access to original Command object
	c.RunE = func(*cobra.Command, []string) error { return run(c) }

	return c
}

func run(cmd *Command) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	// run server
	s := server.NewServer(cmd.cfg)
	err := s.ListenAndServe(ctx)
	if err != nil {
		return fmt.Errorf("Toolbox crashed with the following error: %w", err)
	}

	return nil
}
