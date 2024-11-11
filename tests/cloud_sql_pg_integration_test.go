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

package tests

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/server"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqlpg"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/postgressql"
)

func TestCloudsqlConnection(t *testing.T) {
	sourceConfigs := map[string]sources.SourceConfig{
		"my-pg-instance": cloudsqlpg.Config{
			Name:     "my-pg-instance",
			Kind:     cloudsqlpg.SourceKind,
			Project:  os.Getenv("PROJECT_ID"),
			Region:   os.Getenv("REGION"),
			Instance: os.Getenv("INSTANCE_ID"),
			User:     os.Getenv("cloud_sql_pg_user"),
			Password: os.Getenv("cloud_sql_pg_pass"),
			Database: os.Getenv("DATABASE_ID")}}
	toolConfigs := server.ToolConfigs{
		"tool1": postgressql.Config{
			Name:        "tool1",
			Kind:        cloudsqlpg.SourceKind,
			Source:      "my-pg-instance",
			Description: "description1",
			Statement:   "SELECT * FROM postgres",
			Parameters:  tools.Parameters{tools.NewStringParameter("str-param", "String parameter")},
		},
		"tool2": postgressql.Config{
			Name:        "tool2",
			Kind:        cloudsqlpg.SourceKind,
			Source:      "my-pg-instance",
			Description: "description2",
			Parameters:  tools.Parameters{tools.NewIntParameter("int-param", "Integer parameter")},
		},
	}

	toolsetConfigs := server.ToolsetConfigs{
		"toolset1": tools.ToolsetConfig{Name: "toolset1", ToolNames: []string{"tool1", "tool2"}},
	}

	cfg := server.ServerConfig{
		Version:        "1.0.0",
		Address:        "127.0.0.1",
		Port:           5000,
		SourceConfigs:  sourceConfigs,
		ToolConfigs:    toolConfigs,
		ToolsetConfigs: toolsetConfigs,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Unable to initialize server! %v", err)
	}
	errCh := make(chan error)
	go func() {
		err := s.ListenAndServe(ctx)
		defer close(errCh)
		if err != nil {
			errCh <- err
		}
	}()

	// Test tool invocation request
	resp, err := http.Post("http://127.0.0.1:5000/api/tool/tool2", "application/json", nil)
	if err != nil {
		t.Fatalf("Error sending POST request /api/tool/tool2: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Error handling POST request /api/tool/tool2: %s", err)
	}

	// Test toolset manifest request
	resp, err = http.Get("http://127.0.0.1:5000/api/toolset/")
	if err != nil {
		t.Fatalf("Error sending GET request /api/toolset/: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Error handling GET request /api/toolset/: %s", err)
	}

}
