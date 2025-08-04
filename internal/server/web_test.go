package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-goquery/goquery"
)

// TestWebEndpoint tests the routes defined in webRouter mounted under /ui.
func TestWebEndpoint(t *testing.T) {
	mainRouter := chi.NewRouter()
	webR, err := webRouter()
	if err != nil {
		t.Fatalf("Failed to create webRouter: %v", err)
	}
	mainRouter.Mount("/ui", webR)

	ts := httptest.NewServer(mainRouter)
	defer ts.Close()

	testCases := []struct {
		name            string
		path            string
		wantStatus      int
		wantContentType string
		wantPageTitle   string
	}{
		{
			name:            "web index page",
			path:            "/ui",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
			wantPageTitle:   "Toolbox UI",
		},
		{
			name:            "web index page with trailing slash",
			path:            "/ui/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
			wantPageTitle:   "Toolbox UI",
		},
		{
			name:            "web tools page",
			path:            "/ui/tools",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
			wantPageTitle:   "Tools View",
		},
		{
			name:            "web tools page with trailing slash",
			path:            "/ui/tools/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
			wantPageTitle:   "Tools View",
		},
		{
			name:            "web toolsets page",
			path:            "/ui/toolsets",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
			wantPageTitle:   "Toolsets View",
		},
		{
			name:            "web toolsets page with trailing slash",
			path:            "/ui/toolsets/",
			wantStatus:      http.StatusOK,
			wantContentType: "text/html",
			wantPageTitle:   "Toolsets View",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqURL := ts.URL + tc.path
			req, err := http.NewRequest(http.MethodGet, reqURL, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			client := ts.Client()
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.wantStatus {
				body, _ := io.ReadAll(resp.Body)
				t.Fatalf("Unexpected status code for %s: got %d, want %d, body: %s", tc.path, resp.StatusCode, tc.wantStatus, string(body))
			}

			contentType := resp.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, tc.wantContentType) {
				t.Errorf("Unexpected Content-Type header for %s: got %s, want prefix %s", tc.path, contentType, tc.wantContentType)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}

			gotPageTitle := doc.Find("title").Text()
			if gotPageTitle != tc.wantPageTitle {
				t.Errorf("Unexpected page title for %s: got %q, want %q", tc.path, gotPageTitle, tc.wantPageTitle)
			}

			pageURL := resp.Request.URL
			verifyLinkedResources(t, ts, pageURL, doc)
		})
	}
}

// verifyLinkedResources checks that resources linked in the HTML are served correctly.
func verifyLinkedResources(t *testing.T, ts *httptest.Server, pageURL *url.URL, doc *goquery.Document) {
	t.Helper()

	selectors := map[string]string{
		"stylesheet": "link[rel=stylesheet]",
		"script":     "script[src]",
	}

	attrMap := map[string]string{
		"stylesheet": "href",
		"script":     "src",
	}

	foundResource := false
	for resourceType, selector := range selectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			foundResource = true
			attrName := attrMap[resourceType]
			resourcePath, exists := s.Attr(attrName)
			if !exists || resourcePath == "" {
				t.Errorf("Resource element %s is missing attribute %s on page %s", selector, attrName, pageURL.String())
				return
			}

			// Resolve the URL relative to the page URL
			resURL, err := url.Parse(resourcePath)
			if err != nil {
				t.Errorf("Failed to parse resource path %q on page %s: %v", resourcePath, pageURL.String(), err)
				return
			}
			absoluteResourceURL := pageURL.ResolveReference(resURL)

			// Skip external hosts
			if absoluteResourceURL.Host != pageURL.Host {
				t.Logf("Skipping resource on different host: %s", absoluteResourceURL.String())
				return
			}

			resp, err := ts.Client().Get(absoluteResourceURL.String())
			if err != nil {
				t.Errorf("Failed to GET %s resource %s: %v", resourceType, absoluteResourceURL.String(), err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Resource %s %s: expected status OK (200), but got %d", resourceType, absoluteResourceURL.String(), resp.StatusCode)
			}
		})
	}

	if !foundResource {
		t.Logf("No stylesheet or script resources found to check on page %s", pageURL.String())
	}
}
