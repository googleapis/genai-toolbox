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

	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/googleapis/genai-toolbox/internal/util"
)

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
