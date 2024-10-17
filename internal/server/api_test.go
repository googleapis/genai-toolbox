package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

var _ tools.Tool = &MockTool{}

type MockTool struct {
	Name        string
	Description string
	Params      []tools.Parameter
}

func (t MockTool) Invoke([]any) (string, error) {
	return "", nil
}

func (t MockTool) ParseParams(data map[string]any) ([]any, error) {
	return tools.ParseParams(t.Params, data)
}

func (t MockTool) Manifest() tools.Manifest {
	return tools.Manifest{Description: t.Description, Parameters: t.Params}
}

func TestToolsetEndpoint(t *testing.T) {
	// Set up resources to test against
	tool1 := MockTool{
		Name:   "no_params",
		Params: []tools.Parameter{},
	}
	tool2 := MockTool{
		Name: "some_params",
		Params: []tools.Parameter{
			{
				Name:        "param1",
				Type:        "int",
				Description: "This is the first parameter.",
			},
			{
				Name:        "param2",
				Type:        "string",
				Description: "This is the second parameter.",
			},
		},
	}
	toolsMap := map[string]tools.Tool{tool1.Name: tool1, tool2.Name: tool2}

	toolsets := make(map[string]tools.Toolset)
	for name, l := range map[string][]string{
		"":           {tool1.Name, tool2.Name},
		"tool1_only": {tool1.Name},
		"tool2_only": {tool2.Name},
	} {
		tc := tools.ToolsetConfig{Name: name, ToolNames: l}
		m, err := tc.Initialize("0.0.0", toolsMap)
		if err != nil {
			t.Fatalf("unable to initialize toolset %q: %s", name, err)
		}
		toolsets[name] = m
	}

	server := Server{tools: toolsMap, toolsets: toolsets}
	r, err := apiRouter(&server)
	if err != nil {
		t.Fatalf("unable to initalize router: %s", err)
	}
	ts := httptest.NewServer(r)
	defer ts.Close()

	// wantRepsonse is a struct for checks against test cases
	type wantResponse struct {
		statusCode int
		isErr      bool
		version    string
		tools      []string
	}

	testCases := []struct {
		name        string
		toolsetName string
		want        wantResponse
	}{
		{
			name:        "'default' manifest",
			toolsetName: "",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    "0.0.0",
			},
		},
		{
			name:        "invalid toolset name",
			toolsetName: "some_imaginary_toolset",
			want: wantResponse{
				statusCode: http.StatusNotFound,
				isErr:      true,
			},
		},
		{
			name:        "single toolset",
			toolsetName: "tool1_only",
			want: wantResponse{
				statusCode: http.StatusOK,
				version:    "0.0.0",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body, err := testRequest(ts, http.MethodGet, fmt.Sprintf("/toolset/%s", tc.toolsetName), nil)
			if err != nil {
				t.Fatalf("unexpected error during request: %s", err)
			}

			if resp.StatusCode != tc.want.statusCode {
				t.Logf("response body: %s", body)
				t.Fatalf("unexpected status code: want %d, got %d", tc.want.statusCode, resp.StatusCode)
			}
			if tc.want.isErr {
				// skip the rest of the checks if this is an error case
				return
			}
			var m tools.ToolsetManifest
			err = json.Unmarshal(body, &m)
			if err != nil {
				t.Fatalf("unable to parse ToolsetManifest: %s", err)
			}
			// Check the version is correct
			if m.ServerVersion != tc.want.version {
				t.Fatalf("unexpected ServerVersion: want %q, got %q", tc.want.version, m.ServerVersion)
			}
			// validate that the tools in the toolset are correct
			for _, name := range tc.want.tools {
				_, ok := m.ToolsManifest[name]
				if !ok {
					t.Errorf("%q tool not found in manfiest", name)
				}
			}
		})
	}
}

func testRequest(ts *httptest.Server, method, path string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to send request: %w", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read request body: %w", err)
	}
	defer resp.Body.Close()

	return resp, respBody, nil
}
