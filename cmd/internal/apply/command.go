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

package apply

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/cmd/internal"
	"github.com/googleapis/genai-toolbox/internal/log"
	"github.com/spf13/cobra"
)

type applyCmd struct {
	*cobra.Command
	port    int
	address string
}

func NewCommand(opts *internal.ToolboxOptions) *cobra.Command {
	cmd := &applyCmd{}
	cmd.Command = &cobra.Command{
		Use:   "apply",
		Short: "Apply configuration to the toolbox server",
		Long:  "Apply configuration to the toolbox server",
	}
	flags := cmd.Flags()
	internal.ConfigFileFlags(flags, opts)
	flags.StringVarP(&cmd.address, "address", "a", "127.0.0.1", "Address of the server that is running on.")
	flags.IntVarP(&cmd.port, "port", "p", 5000, "Port of the server that is listening on.")
	cmd.RunE = func(*cobra.Command, []string) error { return runApply(cmd, opts) }
	return cmd.Command
}

// using this type allow O(1) lookups and deletes
type primitives map[string]map[string]struct{}

func NewPrimitives() primitives {
	return make(primitives)
}

// Helper to fetch and populate the map
func (p primitives) Load(ctx context.Context, address string, port int) error {
	kinds := []string{"source", "authservice", "embeddingmodel", "tool", "toolset", "prompt"}

	for _, kind := range kinds {
		list, err := getByPrimitiveRequest(ctx, address, port, kind)
		if err != nil {
			return fmt.Errorf("error getting %s primitives: %w", kind, err)
		}

		p[kind] = make(map[string]struct{})
		for _, name := range list {
			p[kind][name] = struct{}{}
		}
	}
	return nil
}

func (p primitives) Exists(kind, name string) bool {
	names, ok := p[strings.ToLower(kind)]
	if !ok {
		return false
	}
	_, exists := names[name]
	return exists
}

func (p primitives) Remove(kind, name string) {
	if names, ok := p[strings.ToLower(kind)]; ok {
		delete(names, name)
	}
}

// adminRequest is a generic helper for admin api requests.
func adminRequest(ctx context.Context, method, url string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("error marshaling body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("request creation failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, string(bytes.TrimSpace(respBody)))
	}
	return respBody, nil
}

// getByPrimitiveRequest sends a GET request to the admin endpoint
func getByPrimitiveRequest(ctx context.Context, address string, port int, kind string) ([]string, error) {
	url := fmt.Sprintf("http://%s:%d/admin/%s", address, port, kind)

	respBody, err := adminRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var resList []string
	if err := json.Unmarshal(respBody, &resList); err != nil {
		return nil, fmt.Errorf("could not unmarshal response as json: %w", err)
	}
	return resList, nil
}

// getPrimitiveByName sends a GET request (by primitive name) to the admin endpoint
func getPrimitiveByName(ctx context.Context, address string, port int, kind, name string) (map[string]any, error) {
	url := fmt.Sprintf("http://%s:%d/admin/%s/%s", address, port, kind, name)

	respBody, err := adminRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var body map[string]any
	if err := json.Unmarshal(respBody, &body); err != nil {
		return nil, fmt.Errorf("erorr unmarshaling response body: %w", err)
	}
	return body, nil
}

// applyPrimitive sends a PUT request to the admin endpoint
func applyPrimitive(ctx context.Context, address string, port int, kind, name string, data map[string]any) error {
	url := fmt.Sprintf("http://%s:%d/admin/%s/%s", address, port, kind, name)

	_, err := adminRequest(ctx, http.MethodPut, url, data)
	if err != nil {
		return err
	}
	return nil
}

func runApply(cmd *applyCmd, opts *internal.ToolboxOptions) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	ctx, shutdown, err := opts.Setup(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	filePaths, _, err := opts.GetCustomConfigFiles(ctx)
	if err != nil {
		errMsg := fmt.Errorf("failed to retrieve config files: %w", err)
		opts.Logger.ErrorContext(ctx, errMsg.Error())
		return errMsg
	}

	// GET all the primitive lists in the server
	p := NewPrimitives()
	if err := p.Load(ctx, cmd.address, cmd.port); err != nil {
		return err
	}

	opts.Logger.InfoContext(ctx, "starting apply sequence", "count", len(filePaths))
	for _, filePath := range filePaths {
		if err := processFile(ctx, opts.Logger, filePath, p, cmd.address, cmd.port); err != nil {
			opts.Logger.ErrorContext(ctx, err.Error())
			return err
		}
	}
	opts.Logger.InfoContext(ctx, "Done applying")
	return nil
}

func processFile(ctx context.Context, logger log.Logger, path string, p primitives, address string, port int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("unable to open file at %q: %w", path, err)
	}
	defer f.Close() // Safe closure

	decoder := yaml.NewDecoder(f)
	// loop through documents with the `---` separator
	for {
		var doc map[string]any
		if err := decoder.Decode(&doc); err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("unable to decode YAML document: %w", err)
		}

		if len(doc) == 0 {
			continue
		}

		kind, kOk := doc["kind"].(string)
		name, nOk := doc["name"].(string)
		if !kOk || !nOk || kind == "" || name == "" {
			logger.WarnContext(ctx, fmt.Sprintf("invalid primitive schema: missing metadata in %s: kind and name are required", path))
			continue
		}

		delete(doc, "kind")

		if p.Exists(kind, name) {
			p.Remove(kind, name)
			remoteBody, err := getPrimitiveByName(ctx, address, port, kind, name)
			if err != nil {
				return err
			}
			localJSON, err := json.Marshal(doc)
			if err != nil {
				return fmt.Errorf("failed to marshal local config for %s: %w", name, err)
			}
			remoteJSON, err := json.Marshal(remoteBody)
			if err != nil {
				return fmt.Errorf("failed to marshal remote config for %s: %w", name, err)
			}

			if bytes.Equal(localJSON, remoteJSON) {
				logger.DebugContext(ctx, "skipping: no changes detected", "kind", kind, "name", name)
				continue
			}
			logger.DebugContext(ctx, "change detected, updating resource", "kind", kind, "name", name)
		}

		// TODO: check --prune flag: if prune, delete primitives that are left
		// in the primitive list
		// TODO: check for --dry-run flag.

		if err := applyPrimitive(ctx, address, port, kind, name, doc); err != nil {
			return err
		}
	}
	return nil
}
