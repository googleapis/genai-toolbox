package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-goquery/goquery"
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
		wantPageTitle   string
	}{
		{
			name:            "web index page GET",
			method:          http.MethodGet,
			path:            "/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html; charset=utf-8",
			wantPageTitle:   "Toolbox UI",
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

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}
			gotPageTitle := doc.Find("title").Text()

			if gotPageTitle != tc.wantPageTitle {
				t.Errorf("Unexpected page title: got %q, want %q", gotPageTitle, tc.wantPageTitle)
			}
		})
	}
}
