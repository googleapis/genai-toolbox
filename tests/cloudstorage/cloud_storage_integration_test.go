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

package cloudstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/uuid"
	"github.com/googleapis/mcp-toolbox/internal/testutils"
	"github.com/googleapis/mcp-toolbox/tests"
	"google.golang.org/api/iterator"
)

var (
	CloudStorageSourceType = "cloud-storage"
	CloudStorageProject    = os.Getenv("CLOUD_STORAGE_PROJECT")
)

const (
	helloObject  = "seed/hello.txt"
	jsonObject   = "seed/nested/data.json"
	largeObject  = "seed/large.bin"
	binaryObject = "seed/binary.bin"
	helloBody    = "hello world"
	jsonBody     = `{"foo":"bar"}`
	// largeObjectSize is > the 8 MiB read cap so we can assert the size-limit
	// agent-error path on the read_object tool.
	largeObjectSize = (8 << 20) + 1024
)

func getCloudStorageVars(t *testing.T) map[string]any {
	if CloudStorageProject == "" {
		t.Fatal("'CLOUD_STORAGE_PROJECT' not set")
	}
	return map[string]any{
		"type":    CloudStorageSourceType,
		"project": CloudStorageProject,
	}
}

func TestCloudStorageToolEndpoints(t *testing.T) {
	sourceConfig := getCloudStorageVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Fatalf("unable to create Cloud Storage client: %s", err)
	}
	defer client.Close()

	// Bucket names must be globally unique and match [a-z0-9_.-]{3,63}.
	bucketName := "toolbox-it-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:20]
	t.Logf("Using test bucket %q", bucketName)

	teardown := setupCloudStorageTestData(t, ctx, client, CloudStorageProject, bucketName)
	defer teardown(t)

	toolsFile := getCloudStorageToolsConfig(sourceConfig)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, "--enable-api")
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, waitCancel := context.WithTimeout(ctx, 10*time.Second)
	defer waitCancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}

	tests.RunToolGetTestByName(t, "my_list_objects",
		map[string]any{
			"my_list_objects": map[string]any{
				"description":  "List objects in a Cloud Storage bucket.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket to list objects from.",
						"name":         "bucket",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Filter results to objects whose names begin with this prefix.",
						"name":         "prefix",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Delimiter used to group object names (typically '/'). When set, common prefixes are returned as 'prefixes'.",
						"name":         "delimiter",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Maximum number of objects to return per page. A value of 0 uses the API default (1000); values above 1000 are rejected.",
						"name":         "max_results",
						"required":     false,
						"type":         "integer",
						"default":      float64(0),
					},
					map[string]any{
						"authServices": []any{},
						"description":  "A previously-returned page token for retrieving the next page of results.",
						"name":         "page_token",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "my_read_object",
		map[string]any{
			"my_read_object": map[string]any{
				"description":  "Read a Cloud Storage object.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket containing the object.",
						"name":         "bucket",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Full object name (path) within the bucket, e.g. 'path/to/file.txt'.",
						"name":         "object",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Optional HTTP byte range, e.g. 'bytes=0-999' (first 1000 bytes), 'bytes=-500' (last 500 bytes), or 'bytes=500-' (from byte 500 to end). Empty reads the full object.",
						"name":         "range",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
				},
			},
		},
	)

	runCloudStorageListObjectsTest(t, bucketName)
	runCloudStorageReadObjectTest(t, bucketName)
}

func getCloudStorageToolsConfig(sourceConfig map[string]any) map[string]any {
	return map[string]any{
		"sources": map[string]any{
			"my_instance": sourceConfig,
		},
		"tools": map[string]any{
			"my_list_objects": map[string]any{
				"type":        "cloud-storage-list-objects",
				"source":      "my_instance",
				"description": "List objects in a Cloud Storage bucket.",
			},
			"my_read_object": map[string]any{
				"type":        "cloud-storage-read-object",
				"source":      "my_instance",
				"description": "Read a Cloud Storage object.",
			},
		},
	}
}

func setupCloudStorageTestData(t *testing.T, ctx context.Context, client *storage.Client, project, bucket string) func(*testing.T) {
	bkt := client.Bucket(bucket)
	if err := bkt.Create(ctx, project, &storage.BucketAttrs{Location: "US"}); err != nil {
		t.Fatalf("failed to create bucket %q: %v", bucket, err)
	}

	writeSeed := func(name, contentType, body string) {
		w := bkt.Object(name).NewWriter(ctx)
		w.ContentType = contentType
		if _, err := io.WriteString(w, body); err != nil {
			_ = w.Close()
			t.Fatalf("failed to write seed object %q: %v", name, err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("failed to close writer for seed object %q: %v", name, err)
		}
	}

	writeSeed(helloObject, "text/plain", helloBody)
	writeSeed(jsonObject, "application/json", jsonBody)

	// Seed an oversize object to exercise the read-size cap.
	large := bytes.Repeat([]byte{'A'}, largeObjectSize)
	lw := bkt.Object(largeObject).NewWriter(ctx)
	lw.ContentType = "application/octet-stream"
	if _, err := lw.Write(large); err != nil {
		_ = lw.Close()
		t.Fatalf("failed to write seed object %q: %v", largeObject, err)
	}
	if err := lw.Close(); err != nil {
		t.Fatalf("failed to close writer for seed object %q: %v", largeObject, err)
	}

	// Seed a small binary (non-UTF-8) object to exercise the
	// ErrBinaryContent path on read_object.
	binary := []byte{0xff, 0xfe, 0xfd, 0xfc}
	bw := bkt.Object(binaryObject).NewWriter(ctx)
	bw.ContentType = "application/octet-stream"
	if _, err := bw.Write(binary); err != nil {
		_ = bw.Close()
		t.Fatalf("failed to write seed object %q: %v", binaryObject, err)
	}
	if err := bw.Close(); err != nil {
		t.Fatalf("failed to close writer for seed object %q: %v", binaryObject, err)
	}

	return func(t *testing.T) {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		it := bkt.Objects(cleanupCtx, nil)
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				t.Logf("cleanup: iterator error, aborting object delete loop: %v", err)
				break
			}
			if delErr := bkt.Object(attrs.Name).Delete(cleanupCtx); delErr != nil {
				t.Logf("cleanup: failed to delete object %q: %v", attrs.Name, delErr)
			}
		}
		if err := bkt.Delete(cleanupCtx); err != nil {
			t.Logf("cleanup: failed to delete bucket %q: %v", bucket, err)
		}
	}
}

// invokeTool POSTs to the tool invoke endpoint and returns the parsed `result`
// string (which is itself a JSON-encoded payload). On non-200 responses, the
// full body is returned as the error.
func invokeTool(t *testing.T, toolName, requestBody string) (string, int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:5000/api/tool/%s/invoke", toolName)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(requestBody))
	if err != nil {
		t.Fatalf("unable to create request: %s", err)
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("unable to send request: %s", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return string(bodyBytes), resp.StatusCode
	}
	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("failed to parse response JSON: %s (body=%s)", err, string(bodyBytes))
	}
	result, _ := body["result"].(string)
	return result, resp.StatusCode
}

func runCloudStorageListObjectsTest(t *testing.T, bucket string) {
	fakeBucket := "toolbox-it-does-not-exist-" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	tcs := []struct {
		name           string
		body           string
		wantSubstrings []string
	}{
		{
			name:           "list with prefix",
			body:           fmt.Sprintf(`{"bucket": %q, "prefix": "seed/"}`, bucket),
			wantSubstrings: []string{helloObject, jsonObject},
		},
		{
			name:           "empty prefix and delimiter lists all objects",
			body:           fmt.Sprintf(`{"bucket": %q, "prefix": "", "delimiter": ""}`, bucket),
			wantSubstrings: []string{helloObject, jsonObject},
		},
		{
			name:           "empty page_token behaves as first page",
			body:           fmt.Sprintf(`{"bucket": %q, "prefix": "seed/", "page_token": ""}`, bucket),
			wantSubstrings: []string{helloObject, jsonObject},
		},
		{
			name:           "list with delimiter returns prefixes",
			body:           fmt.Sprintf(`{"bucket": %q, "prefix": "seed/", "delimiter": "/"}`, bucket),
			wantSubstrings: []string{helloObject, `"seed/nested/"`},
		},
		{
			name:           "missing bucket parameter returns agent error",
			body:           `{}`,
			wantSubstrings: []string{"bucket"},
		},
		{
			name:           "max_results above 1000 returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "max_results": 1001}`, bucket),
			wantSubstrings: []string{"max_results", "1000"},
		},
		{
			name:           "nonexistent bucket returns error",
			body:           fmt.Sprintf(`{"bucket": %q}`, fakeBucket),
			wantSubstrings: []string{fakeBucket},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_list_objects", tc.body)
			if status != http.StatusOK {
				t.Fatalf("unexpected status %d: %s", status, result)
			}
			for _, want := range tc.wantSubstrings {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got %s", want, result)
				}
			}
		})
	}

	// Pagination is inherently two-step (fetch page one, reuse its token for
	// page two), so it doesn't fit the single-request table above.
	t.Run("pagination via max_results and page_token", func(t *testing.T) {
		result, status := invokeTool(t, "my_list_objects",
			fmt.Sprintf(`{"bucket": %q, "prefix": "seed/", "max_results": 1}`, bucket))
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		token := extractStringField(t, result, "nextPageToken")
		if token == "" {
			t.Fatalf("expected non-empty nextPageToken, got %s", result)
		}

		result2, status := invokeTool(t, "my_list_objects",
			fmt.Sprintf(`{"bucket": %q, "prefix": "seed/", "page_token": %q}`, bucket, token))
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result2)
		}
		combined := result + result2
		if !strings.Contains(combined, helloObject) || !strings.Contains(combined, jsonObject) {
			t.Errorf("expected both %q and %q across paginated results, got page1=%s page2=%s",
				helloObject, jsonObject, result, result2)
		}
	})
}

func runCloudStorageReadObjectTest(t *testing.T, bucket string) {
	tcs := []struct {
		name            string
		body            string
		wantContent     string
		wantContentType string
		wantSubstrings  []string
	}{
		{
			name:            "read full object",
			body:            fmt.Sprintf(`{"bucket": %q, "object": %q}`, bucket, helloObject),
			wantContent:     helloBody,
			wantContentType: "text/plain",
		},
		{
			name:        "read range bytes=0-4",
			body:        fmt.Sprintf(`{"bucket": %q, "object": %q, "range": "bytes=0-4"}`, bucket, helloObject),
			wantContent: "hello",
		},
		{
			name:        "read suffix range bytes=-5",
			body:        fmt.Sprintf(`{"bucket": %q, "object": %q, "range": "bytes=-5"}`, bucket, helloObject),
			wantContent: "world",
		},
		{
			name:        "read open-ended range bytes=6-",
			body:        fmt.Sprintf(`{"bucket": %q, "object": %q, "range": "bytes=6-"}`, bucket, helloObject),
			wantContent: "world",
		},
		{
			name:        "oversize read narrowed by range succeeds",
			body:        fmt.Sprintf(`{"bucket": %q, "object": %q, "range": "bytes=0-9"}`, bucket, largeObject),
			wantContent: "AAAAAAAAAA",
		},
		{
			name:           "missing object parameter returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q}`, bucket),
			wantSubstrings: []string{"object"},
		},
		{
			name:           "nonexistent object returns error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "does/not/exist.bin"}`, bucket),
			wantSubstrings: []string{"does/not/exist.bin"},
		},
		{
			name:           "invalid range returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q, "range": "garbage"}`, bucket, helloObject),
			wantSubstrings: []string{"range"},
		},
		{
			name:           "oversize read returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q}`, bucket, largeObject),
			wantSubstrings: []string{"size limit"},
		},
		{
			name:           "binary object returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q}`, bucket, binaryObject),
			wantSubstrings: []string{"UTF-8"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_read_object", tc.body)
			if status != http.StatusOK {
				t.Fatalf("unexpected status %d: %s", status, result)
			}
			if tc.wantContent != "" {
				if got := extractStringField(t, result, "content"); got != tc.wantContent {
					t.Errorf("expected content %q, got %q (raw %s)", tc.wantContent, got, result)
				}
			}
			if tc.wantContentType != "" {
				if got := extractStringField(t, result, "contentType"); got != tc.wantContentType {
					t.Errorf("expected contentType %q, got %q (raw %s)", tc.wantContentType, got, result)
				}
			}
			for _, want := range tc.wantSubstrings {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got %s", want, result)
				}
			}
		})
	}
}

// extractStringField pulls a top-level string field out of a JSON-encoded result
// string (the kind the tool invoke API wraps in the `result` property).
func extractStringField(t *testing.T, result, field string) string {
	t.Helper()
	var parsed map[string]any
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("failed to parse tool result JSON: %s (raw=%s)", err, result)
	}
	v, _ := parsed[field].(string)
	return v
}
