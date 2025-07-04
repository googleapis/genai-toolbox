package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebEndpoint(t *testing.T) {
	router, err := webRouter()
	if err != nil {
		t.Fatalf("Failed to create webRouter: %v", err)
	}

	ts := httptest.NewServer(router)
	defer ts.Close()

	testCases := []struct {
		name            string
		method          string
		path            string
		wantStatus      int
		wantContentType string
		wantBodySubstr  string
	}{
		{
			name:            "web index page GET",
			method:          http.MethodGet,
			path:            "/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html; charset=utf-8",
			wantBodySubstr:  "<title>Toolbox UI</title>",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := ts.URL + tc.path
			req, err := http.NewRequest(tc.method, url, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if resp.StatusCode != tc.wantStatus {
				t.Errorf("Unexpected status code: got %d, want %d, body: %s", resp.StatusCode, tc.wantStatus, string(body))
			}

			if contentType := resp.Header.Get("Content-Type"); contentType != tc.wantContentType {
				t.Errorf("Unexpected Content-Type header: got %s, want %s", contentType, tc.wantContentType)
			}

			if !strings.Contains(string(body), tc.wantBodySubstr) {
				t.Errorf("Unexpected response body: got %q, want to contain %q", string(body), tc.wantBodySubstr)
			}
		})
	}
}
