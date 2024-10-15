package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestToolsetManifest(t *testing.T) {
	tool1Manifest := tools.ToolManifest{
		Description: "description1",
		Parameters:  []tools.Parameter{{Name: "param1", Type: "type", Description: "description1"}},
	}
	tool2Manifest := tools.ToolManifest{
		Description: "description2",
		Parameters:  []tools.Parameter{{Name: "param2", Type: "type", Description: "description2"}},
	}

	tests := []struct {
		name              string
		toolsetName       string
		sourceConfigs     sources.Configs
		toolConfigs       tools.Configs
		toolsetConfigs    tools.ToolsetConfigs
		wantStatusCode    int
		wantServerVersion string
		wantManifests     []tools.ToolManifest
		wantErr           bool
	}{
		{
			name:        "test all tool manifest",
			toolsetName: "",
			sourceConfigs: sources.Configs{
				"my-pg-instance": sources.CloudSQLPgConfig{Name: "my-pg-instance", Kind: sources.CloudSQLPgKind},
			},
			toolConfigs: tools.Configs{
				"tool1": tools.CloudSQLPgGenericConfig{
					Name:        "tool1",
					Kind:        tools.CloudSQLPgSQLGenericKind,
					Source:      "my-pg-instance",
					Description: "description1",
					Parameters:  []tools.Parameter{{Name: "param1", Type: "type", Description: "description1"}},
				},
				"tool2": tools.CloudSQLPgGenericConfig{
					Name:        "tool2",
					Kind:        tools.CloudSQLPgSQLGenericKind,
					Source:      "my-pg-instance",
					Description: "description2",
					Parameters:  []tools.Parameter{{Name: "param2", Type: "type", Description: "description2"}},
				},
			},
			toolsetConfigs: tools.ToolsetConfigs{
				"toolset1": tools.ToolsetConfig{Name: "toolset1", ToolNames: []string{"tool1", "tool2"}},
			},
			wantStatusCode:    http.StatusOK,
			wantServerVersion: "1.0.0",
			wantManifests:     []tools.ToolManifest{tool1Manifest, tool2Manifest},
			wantErr:           false,
		},
		{
			name:        "test invalid toolset name",
			toolsetName: "nonExistentToolset",
			sourceConfigs: sources.Configs{
				"my-pg-instance": sources.CloudSQLPgConfig{Name: "my-pg-instance", Kind: sources.CloudSQLPgKind},
			},
			toolConfigs: tools.Configs{
				"tool1": tools.CloudSQLPgGenericConfig{
					Name:        "tool1",
					Kind:        tools.CloudSQLPgSQLGenericKind,
					Source:      "my-pg-instance",
					Description: "description1",
					Parameters:  []tools.Parameter{{Name: "param1", Type: "type", Description: "description1"}},
				},
			},
			toolsetConfigs: tools.ToolsetConfigs{
				"toolset1": tools.ToolsetConfig{Name: "toolset1", ToolNames: []string{"tool1"}},
			},
			wantStatusCode: http.StatusNotFound,
			wantErr:        true,
		},
		{
			name:        "test one toolset",
			toolsetName: "toolset1",
			sourceConfigs: sources.Configs{
				"my-pg-instance": sources.CloudSQLPgConfig{Name: "my-pg-instance", Kind: sources.CloudSQLPgKind},
			},
			toolConfigs: tools.Configs{
				"tool1": tools.CloudSQLPgGenericConfig{
					Name:        "tool1",
					Kind:        tools.CloudSQLPgSQLGenericKind,
					Source:      "my-pg-instance",
					Description: "description1",
					Parameters:  []tools.Parameter{{Name: "param1", Type: "type", Description: "description1"}},
				},
			},
			toolsetConfigs: tools.ToolsetConfigs{
				"toolset1": tools.ToolsetConfig{Name: "toolset1", ToolNames: []string{"tool1"}},
			},
			wantServerVersion: "1.0.0",
			wantStatusCode:    http.StatusOK,
			wantManifests:     []tools.ToolManifest{tool1Manifest},
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				Version:        "1.0.0",
				Address:        "127.0.0.1",
				Port:           5000,
				SourceConfigs:  tt.sourceConfigs,
				ToolConfigs:    tt.toolConfigs,
				ToolsetConfigs: tt.toolsetConfigs,
			}
			s, err := NewServer(cfg)
			if err != nil {
				t.Fatalf("Unable to initialize server!")
			}
			w := httptest.NewRecorder()
			chiCtx := chi.NewRouteContext()
			r := httptest.NewRequest("GET", "/toolset/"+tt.toolsetName, nil)
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
			chiCtx.URLParams.Add("toolsetName", fmt.Sprintf("%v", tt.toolsetName))

			handler := toolsetHandler(s)
			handler(w, r)

			// Check for error cases
			if tt.wantErr {
				if w.Code != tt.wantStatusCode {
					t.Fatalf("Expected status code %d for error case, got %d", tt.wantStatusCode, w.Code)
				}
				return
			}

			if w.Code != tt.wantStatusCode {
				t.Fatalf("Expected status code %d, got %d", tt.wantStatusCode, w.Code)
			}

			var response tools.ToolsetManifest
			err = json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Error decoding response body: %v", err)
			}

			if response.ServerVersion != tt.wantServerVersion {
				t.Fatalf("Expected ServerVersion '%s', got '%s'", tt.wantServerVersion, response.ServerVersion)
			}

			if diff := cmp.Diff(response.ToolsManifest, tt.wantManifests); diff != "" {
				t.Fatalf("Expected ToolsManifests '%+v', got '%+v'", tt.wantManifests, response.ToolsManifest)
			}
		})
	}
}
