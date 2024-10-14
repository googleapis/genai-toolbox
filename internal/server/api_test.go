package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

func TestToolsetManifest(t *testing.T) {

	manifestsMap := make(map[string][]byte)
	toolManifest := tools.ToolManifest{Description: "description", Parameters: []tools.Parameter{tools.Parameter{Name: "name", Type: "type", Description: "description"}}}
	manifestsMap[""] = []byte(`{"tool1": "description"}`)
	server := &Server{
		manifests: manifestsMap,
	}
	server.conf.Version = "2.0.0"

	req := httptest.NewRequest("GET", "/toolset/", nil)
	w := httptest.NewRecorder()
	toolsetHandler(server)(w, req)

	// Check the response status code
	if w.Code != http.StatusOK {
		t.Fatalf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response tools.ToolsetManifest
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Error decoding response body: %v", err)
	}
	if response.ServerVersion != server.conf.Version {
		t.Fatalf("Expected ServerVersion '%s', got '%s'", serverVersion, response.ServerVersion)
	}

	if response.ToolsManifest != expectedManifest {
		t.Fatalf("Expected Tools '%s', got '%s'", expectedTools, response.Tools)
	}
}
