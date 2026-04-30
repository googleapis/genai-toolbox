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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	helloObject    = "seed/hello.txt"
	jsonObject     = "seed/nested/data.json"
	largeObject    = "seed/large.bin"
	binaryObject   = "seed/binary.bin"
	downloadObject = "seed/download.txt"
	helloBody      = "hello world"
	jsonBody       = `{"foo":"bar"}`
	downloadBody   = "download-me"
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
						"description":  "Maximum number of objects to return per page. A value of 0 uses the API default (1000); negative values and values above 1000 are rejected.",
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

	tests.RunToolGetTestByName(t, "my_list_buckets",
		map[string]any{
			"my_list_buckets": map[string]any{
				"description":  "List Cloud Storage buckets in the project.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Project ID to list buckets in. When empty, the source's configured project is used.",
						"name":         "project",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Filter results to buckets whose names begin with this prefix.",
						"name":         "prefix",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Maximum number of buckets to return per page. A value of 0 uses the API default (1000); negative values and values above 1000 are rejected.",
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
	tests.RunToolGetTestByName(t, "my_get_object_metadata",
		map[string]any{
			"my_get_object_metadata": map[string]any{
				"description":  "Get metadata for a Cloud Storage object.",
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
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "my_download_object",
		map[string]any{
			"my_download_object": map[string]any{
				"description":  "Download a Cloud Storage object to a local file.",
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
						"description":  "Absolute local filesystem path where the object will be written. Relative paths and paths containing '..' are rejected.",
						"name":         "destination",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "If true, overwrite the destination when it already exists. If false (default), the tool returns an error when the destination exists.",
						"name":         "overwrite",
						"required":     false,
						"type":         "boolean",
						"default":      false,
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "my_upload_object",
		map[string]any{
			"my_upload_object": map[string]any{
				"description":  "Upload a local file to a Cloud Storage object.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket to upload into.",
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
						"description":  "Absolute local filesystem path of the file to upload. Relative paths and paths containing '..' are rejected.",
						"name":         "source",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "MIME type to record on the uploaded object. When empty, it is inferred from the source file's extension; if that fails, Cloud Storage auto-detects from the first 512 bytes of content.",
						"name":         "content_type",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "my_write_object",
		map[string]any{
			"my_write_object": map[string]any{
				"description":  "Write text content to a Cloud Storage object.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket to write into.",
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
						"description":  "Text content to write to the Cloud Storage object.",
						"name":         "content",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "MIME type to record on the written object. When empty, Cloud Storage auto-detects from the first 512 bytes of content.",
						"name":         "content_type",
						"required":     false,
						"type":         "string",
						"default":      "",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "my_copy_object",
		map[string]any{
			"my_copy_object": map[string]any{
				"description":  "Copy a Cloud Storage object.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket containing the source object.",
						"name":         "source_bucket",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Full source object name (path) within the source bucket, e.g. 'path/to/file.txt'.",
						"name":         "source_object",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket to copy into.",
						"name":         "destination_bucket",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Full destination object name (path) within the destination bucket, e.g. 'path/to/file.txt'.",
						"name":         "destination_object",
						"required":     true,
						"type":         "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "my_move_object",
		map[string]any{
			"my_move_object": map[string]any{
				"description":  "Move a Cloud Storage object within the same bucket.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket containing the object to move.",
						"name":         "bucket",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Full source object name (path) within the bucket, e.g. 'path/to/file.txt'.",
						"name":         "source_object",
						"required":     true,
						"type":         "string",
					},
					map[string]any{
						"authServices": []any{},
						"description":  "Full destination object name (path) within the same bucket, e.g. 'path/to/file.txt'.",
						"name":         "destination_object",
						"required":     true,
						"type":         "string",
					},
				},
			},
		},
	)
	tests.RunToolGetTestByName(t, "my_delete_object",
		map[string]any{
			"my_delete_object": map[string]any{
				"description":  "Delete a Cloud Storage object.",
				"authRequired": []any{},
				"parameters": []any{
					map[string]any{
						"authServices": []any{},
						"description":  "Name of the Cloud Storage bucket containing the object to delete.",
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
				},
			},
		},
	)

	runCloudStorageListObjectsTest(t, bucketName)
	runCloudStorageReadObjectTest(t, bucketName)
	runCloudStorageListBucketsTest(t, bucketName)
	runCloudStorageGetObjectMetadataTest(t, bucketName)
	runCloudStorageDownloadObjectTest(t, bucketName)
	runCloudStorageUploadObjectTest(ctx, t, client, bucketName)
	runCloudStorageWriteObjectTest(ctx, t, client, bucketName)
	runCloudStorageCopyObjectTest(ctx, t, client, bucketName)
	runCloudStorageMoveObjectTest(ctx, t, client, bucketName)
	runCloudStorageDeleteObjectTest(ctx, t, client, bucketName)
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
			"my_list_buckets": map[string]any{
				"type":        "cloud-storage-list-buckets",
				"source":      "my_instance",
				"description": "List Cloud Storage buckets in the project.",
			},
			"my_get_object_metadata": map[string]any{
				"type":        "cloud-storage-get-object-metadata",
				"source":      "my_instance",
				"description": "Get metadata for a Cloud Storage object.",
			},
			"my_download_object": map[string]any{
				"type":        "cloud-storage-download-object",
				"source":      "my_instance",
				"description": "Download a Cloud Storage object to a local file.",
			},
			"my_upload_object": map[string]any{
				"type":        "cloud-storage-upload-object",
				"source":      "my_instance",
				"description": "Upload a local file to a Cloud Storage object.",
			},
			"my_write_object": map[string]any{
				"type":        "cloud-storage-write-object",
				"source":      "my_instance",
				"description": "Write text content to a Cloud Storage object.",
			},
			"my_copy_object": map[string]any{
				"type":        "cloud-storage-copy-object",
				"source":      "my_instance",
				"description": "Copy a Cloud Storage object.",
			},
			"my_move_object": map[string]any{
				"type":        "cloud-storage-move-object",
				"source":      "my_instance",
				"description": "Move a Cloud Storage object within the same bucket.",
			},
			"my_delete_object": map[string]any{
				"type":        "cloud-storage-delete-object",
				"source":      "my_instance",
				"description": "Delete a Cloud Storage object.",
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
	writeSeed(downloadObject, "text/plain", downloadBody)

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
			name:           "negative max_results returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "max_results": -1}`, bucket),
			wantSubstrings: []string{"max_results", "must be"},
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

func runCloudStorageListBucketsTest(t *testing.T, bucket string) {
	tcs := []struct {
		name             string
		body             string
		wantSubstrings   []string
		unwantSubstrings []string
	}{
		{
			name:           "list with matching prefix finds the test bucket",
			body:           fmt.Sprintf(`{"prefix": %q}`, bucket[:10]),
			wantSubstrings: []string{bucket},
		},
		{
			name:             "list with non-matching prefix omits the test bucket",
			body:             `{"prefix": "toolbox-it-definitely-not-a-real-prefix-"}`,
			unwantSubstrings: []string{bucket},
		},
		{
			name:           "explicit project override returns the test bucket",
			body:           fmt.Sprintf(`{"project": %q, "prefix": %q}`, CloudStorageProject, bucket[:10]),
			wantSubstrings: []string{bucket},
		},
		{
			name:           "max_results above 1000 returns agent error",
			body:           `{"max_results": 1001}`,
			wantSubstrings: []string{"max_results", "1000"},
		},
		{
			name:           "negative max_results returns agent error",
			body:           `{"max_results": -1}`,
			wantSubstrings: []string{"max_results", "must be"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_list_buckets", tc.body)
			if status != http.StatusOK {
				t.Fatalf("unexpected status %d: %s", status, result)
			}
			for _, want := range tc.wantSubstrings {
				if !strings.Contains(result, want) {
					t.Errorf("expected result to contain %q, got %s", want, result)
				}
			}
			for _, unwant := range tc.unwantSubstrings {
				if strings.Contains(result, unwant) {
					t.Errorf("did not expect result to contain %q, got %s", unwant, result)
				}
			}
		})
	}
}

func runCloudStorageGetObjectMetadataTest(t *testing.T, bucket string) {
	tcs := []struct {
		name           string
		body           string
		wantSubstrings []string
	}{
		{
			name:           "metadata for hello.txt",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q}`, bucket, helloObject),
			wantSubstrings: []string{`"ContentType":"text/plain"`, `"Size":11`, `"Name":"seed/hello.txt"`},
		},
		{
			name:           "metadata for JSON object",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q}`, bucket, jsonObject),
			wantSubstrings: []string{`"ContentType":"application/json"`},
		},
		{
			name:           "missing bucket returns agent error",
			body:           `{"object": "x"}`,
			wantSubstrings: []string{"bucket"},
		},
		{
			name:           "missing object returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q}`, bucket),
			wantSubstrings: []string{"object"},
		},
		{
			name:           "nonexistent object returns error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "does/not/exist.bin"}`, bucket),
			wantSubstrings: []string{"does/not/exist.bin"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_get_object_metadata", tc.body)
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
}

func runCloudStorageDownloadObjectTest(t *testing.T, bucket string) {
	t.Run("happy path writes expected bytes", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "downloaded.txt")
		body := fmt.Sprintf(`{"bucket": %q, "object": %q, "destination": %q}`, bucket, downloadObject, dest)
		result, status := invokeTool(t, "my_download_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		if ct := extractStringField(t, result, "contentType"); ct != "text/plain" {
			t.Errorf("contentType = %q, want text/plain (raw %s)", ct, result)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("failed to read downloaded file: %v", err)
		}
		if string(got) != downloadBody {
			t.Errorf("downloaded content = %q, want %q", string(got), downloadBody)
		}
	})

	t.Run("overwrite=false on existing file returns agent error", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "existing.txt")
		if err := os.WriteFile(dest, []byte("pre-existing"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		body := fmt.Sprintf(`{"bucket": %q, "object": %q, "destination": %q}`, bucket, downloadObject, dest)
		result, status := invokeTool(t, "my_download_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		if !strings.Contains(result, "overwrite") && !strings.Contains(result, "exists") {
			t.Errorf("expected error referencing overwrite/exists, got %s", result)
		}
		got, err := os.ReadFile(dest)
		if err != nil || string(got) != "pre-existing" {
			t.Errorf("destination was modified or unreadable: %q, %v", string(got), err)
		}
	})

	t.Run("overwrite=true replaces existing file", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "existing.txt")
		if err := os.WriteFile(dest, []byte("pre-existing"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}
		body := fmt.Sprintf(`{"bucket": %q, "object": %q, "destination": %q, "overwrite": true}`, bucket, downloadObject, dest)
		result, status := invokeTool(t, "my_download_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("failed to read downloaded file: %v", err)
		}
		if string(got) != downloadBody {
			t.Errorf("downloaded content = %q, want %q", string(got), downloadBody)
		}
	})

	tcs := []struct {
		name           string
		body           string
		wantSubstrings []string
	}{
		{
			name:           "relative destination returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q, "destination": "relative/out.bin"}`, bucket, downloadObject),
			wantSubstrings: []string{"destination"},
		},
		{
			name:           "destination with traversal returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q, "destination": "/tmp/../etc/passwd"}`, bucket, downloadObject),
			wantSubstrings: []string{"destination"},
		},
		{
			name:           "missing destination returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": %q}`, bucket, downloadObject),
			wantSubstrings: []string{"destination"},
		},
		{
			name:           "nonexistent object returns error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "does/not/exist.bin", "destination": "/tmp/nope-should-not-be-created-%s.bin"}`, bucket, uuid.New().String()[:8]),
			wantSubstrings: []string{"does/not/exist.bin"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_download_object", tc.body)
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
}

func runCloudStorageUploadObjectTest(ctx context.Context, t *testing.T, client *storage.Client, bucket string) {
	// Seed a local file that the explicit-content-type and MIME-auto-detect
	// cases both read from.
	srcDir := t.TempDir()
	csvPath := filepath.Join(srcDir, "data.csv")
	if err := os.WriteFile(csvPath, []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	binPath := filepath.Join(srcDir, "blob.unknownext")
	if err := os.WriteFile(binPath, []byte("<html>hi</html>"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	t.Run("upload with explicit content_type", func(t *testing.T) {
		obj := "uploaded/explicit.bin"
		body := fmt.Sprintf(`{"bucket": %q, "object": %q, "source": %q, "content_type": "application/octet-stream"}`,
			bucket, obj, csvPath)
		result, status := invokeTool(t, "my_upload_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		if ct := extractStringField(t, result, "contentType"); ct != "application/octet-stream" {
			t.Errorf("contentType = %q, want application/octet-stream (raw %s)", ct, result)
		}
		attrs, err := client.Bucket(bucket).Object(obj).Attrs(ctx)
		if err != nil {
			t.Fatalf("expected uploaded object to exist: %v", err)
		}
		if attrs.ContentType != "application/octet-stream" {
			t.Errorf("GCS ContentType = %q, want application/octet-stream", attrs.ContentType)
		}
	})

	t.Run("upload infers MIME from .csv extension", func(t *testing.T) {
		obj := "uploaded/inferred.csv"
		body := fmt.Sprintf(`{"bucket": %q, "object": %q, "source": %q}`, bucket, obj, csvPath)
		result, status := invokeTool(t, "my_upload_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		// mime.TypeByExtension(".csv") is "text/csv" on most systems but may
		// carry a charset suffix — assert containment rather than equality.
		if ct := extractStringField(t, result, "contentType"); !strings.Contains(ct, "csv") {
			t.Errorf("contentType = %q, want to contain 'csv' (raw %s)", ct, result)
		}
	})

	t.Run("upload with unknown extension lets GCS auto-detect", func(t *testing.T) {
		obj := "uploaded/unknown.bin"
		body := fmt.Sprintf(`{"bucket": %q, "object": %q, "source": %q}`, bucket, obj, binPath)
		result, status := invokeTool(t, "my_upload_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		attrs, err := client.Bucket(bucket).Object(obj).Attrs(ctx)
		if err != nil {
			t.Fatalf("expected uploaded object to exist: %v", err)
		}
		// GCS always records *something* — verify the bytes landed regardless
		// of whichever content type the server ended up storing.
		if attrs.Size != int64(len("<html>hi</html>")) {
			t.Errorf("object size = %d, want %d", attrs.Size, len("<html>hi</html>"))
		}
	})

	tcs := []struct {
		name           string
		body           string
		wantSubstrings []string
	}{
		{
			name:           "missing source returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "uploaded/nope.bin"}`, bucket),
			wantSubstrings: []string{"source"},
		},
		{
			name:           "relative source returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "uploaded/nope.bin", "source": "relative/path"}`, bucket),
			wantSubstrings: []string{"source"},
		},
		{
			name:           "nonexistent local source returns error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "uploaded/nope.bin", "source": "/tmp/definitely-not-a-real-source-%s"}`, bucket, uuid.New().String()[:8]),
			wantSubstrings: []string{"source", "no such"},
		},
		{
			name:           "missing bucket returns agent error",
			body:           fmt.Sprintf(`{"object": "uploaded/nope.bin", "source": %q}`, csvPath),
			wantSubstrings: []string{"bucket"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_upload_object", tc.body)
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
}

func runCloudStorageWriteObjectTest(ctx context.Context, t *testing.T, client *storage.Client, bucket string) {
	tcs := []struct {
		name            string
		body            string
		object          string
		wantContent     string
		wantContentType string
		wantSize        int64
		wantSubstrings  []string
	}{
		{
			name:            "write object with explicit content_type",
			object:          "written/explicit.txt",
			wantContent:     "written from tool",
			wantContentType: "text/plain",
			wantSize:        int64(len("written from tool")),
		},
		{
			name:            "write empty object",
			object:          "written/empty.txt",
			wantContent:     "",
			wantContentType: "text/plain",
			wantSize:        0,
		},
		{
			name:           "missing bucket returns agent error",
			body:           `{"object": "written/nope.txt", "content": "x"}`,
			wantSubstrings: []string{"bucket"},
		},
		{
			name:           "missing object returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "content": "x"}`, bucket),
			wantSubstrings: []string{"object"},
		},
		{
			name:           "missing content returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "written/nope.txt"}`, bucket),
			wantSubstrings: []string{"content"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			body := tc.body
			if body == "" {
				body = fmt.Sprintf(`{"bucket": %q, "object": %q, "content": %q, "content_type": %q}`,
					bucket, tc.object, tc.wantContent, tc.wantContentType)
			}
			result, status := invokeTool(t, "my_write_object", body)
			if status != http.StatusOK {
				t.Fatalf("unexpected status %d: %s", status, result)
			}
			if tc.object != "" {
				if ct := extractStringField(t, result, "contentType"); ct != tc.wantContentType {
					t.Errorf("contentType = %q, want %q (raw %s)", ct, tc.wantContentType, result)
				}
				got, attrs := readGCSObject(t, ctx, client, bucket, tc.object)
				if got != tc.wantContent {
					t.Errorf("written content = %q, want %q", got, tc.wantContent)
				}
				if attrs.ContentType != tc.wantContentType {
					t.Errorf("GCS ContentType = %q, want %q", attrs.ContentType, tc.wantContentType)
				}
				if attrs.Size != tc.wantSize {
					t.Errorf("object size = %d, want %d", attrs.Size, tc.wantSize)
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

func runCloudStorageCopyObjectTest(ctx context.Context, t *testing.T, client *storage.Client, bucket string) {
	t.Run("copy creates destination and leaves source", func(t *testing.T) {
		dest := "copied/hello.txt"
		body := fmt.Sprintf(`{"source_bucket": %q, "source_object": %q, "destination_bucket": %q, "destination_object": %q}`,
			bucket, helloObject, bucket, dest)
		result, status := invokeTool(t, "my_copy_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		got, attrs := readGCSObject(t, ctx, client, bucket, dest)
		if got != helloBody {
			t.Errorf("copied content = %q, want %q", got, helloBody)
		}
		if attrs.ContentType != "text/plain" {
			t.Errorf("copied ContentType = %q, want text/plain", attrs.ContentType)
		}
		if !gcsObjectExists(t, ctx, client, bucket, helloObject) {
			t.Errorf("expected source object %q to remain after copy", helloObject)
		}
	})

	tcs := []struct {
		name           string
		body           string
		wantSubstrings []string
	}{
		{
			name:           "missing source_bucket returns agent error",
			body:           fmt.Sprintf(`{"source_object": %q, "destination_bucket": %q, "destination_object": "copied/nope.txt"}`, helloObject, bucket),
			wantSubstrings: []string{"source_bucket"},
		},
		{
			name:           "missing source_object returns agent error",
			body:           fmt.Sprintf(`{"source_bucket": %q, "destination_bucket": %q, "destination_object": "copied/nope.txt"}`, bucket, bucket),
			wantSubstrings: []string{"source_object"},
		},
		{
			name:           "missing destination_bucket returns agent error",
			body:           fmt.Sprintf(`{"source_bucket": %q, "source_object": %q, "destination_object": "copied/nope.txt"}`, bucket, helloObject),
			wantSubstrings: []string{"destination_bucket"},
		},
		{
			name:           "missing destination_object returns agent error",
			body:           fmt.Sprintf(`{"source_bucket": %q, "source_object": %q, "destination_bucket": %q}`, bucket, helloObject, bucket),
			wantSubstrings: []string{"destination_object"},
		},
		{
			name:           "nonexistent source object returns error",
			body:           fmt.Sprintf(`{"source_bucket": %q, "source_object": "does/not/exist.txt", "destination_bucket": %q, "destination_object": "copied/nope.txt"}`, bucket, bucket),
			wantSubstrings: []string{"does/not/exist.txt"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_copy_object", tc.body)
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
}

func runCloudStorageMoveObjectTest(ctx context.Context, t *testing.T, client *storage.Client, bucket string) {
	t.Run("move atomically renames within bucket", func(t *testing.T) {
		src := "move/source.txt"
		dest := "move/destination.txt"
		writeGCSObject(t, ctx, client, bucket, src, "text/plain", "move me")
		body := fmt.Sprintf(`{"bucket": %q, "source_object": %q, "destination_object": %q}`,
			bucket, src, dest)
		result, status := invokeTool(t, "my_move_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		got, _ := readGCSObject(t, ctx, client, bucket, dest)
		if got != "move me" {
			t.Errorf("moved content = %q, want %q", got, "move me")
		}
		if gcsObjectExists(t, ctx, client, bucket, src) {
			t.Errorf("expected source object %q to be gone after move", src)
		}
	})

	tcs := []struct {
		name           string
		body           string
		wantSubstrings []string
	}{
		{
			name:           "missing bucket returns agent error",
			body:           `{"source_object": "move/nope.txt", "destination_object": "move/nope2.txt"}`,
			wantSubstrings: []string{"bucket"},
		},
		{
			name:           "missing source_object returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "destination_object": "move/nope2.txt"}`, bucket),
			wantSubstrings: []string{"source_object"},
		},
		{
			name:           "missing destination_object returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q, "source_object": "move/nope.txt"}`, bucket),
			wantSubstrings: []string{"destination_object"},
		},
		{
			name:           "nonexistent source object returns error",
			body:           fmt.Sprintf(`{"bucket": %q, "source_object": "move/does-not-exist.txt", "destination_object": "move/nope2.txt"}`, bucket),
			wantSubstrings: []string{"move/does-not-exist.txt"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_move_object", tc.body)
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
}

func runCloudStorageDeleteObjectTest(ctx context.Context, t *testing.T, client *storage.Client, bucket string) {
	t.Run("delete removes object", func(t *testing.T) {
		obj := "delete/delete-me.txt"
		writeGCSObject(t, ctx, client, bucket, obj, "text/plain", "delete me")
		body := fmt.Sprintf(`{"bucket": %q, "object": %q}`, bucket, obj)
		result, status := invokeTool(t, "my_delete_object", body)
		if status != http.StatusOK {
			t.Fatalf("unexpected status %d: %s", status, result)
		}
		if !strings.Contains(result, `"deleted":true`) {
			t.Errorf("expected deleted confirmation, got %s", result)
		}
		if gcsObjectExists(t, ctx, client, bucket, obj) {
			t.Errorf("expected object %q to be deleted", obj)
		}
	})

	tcs := []struct {
		name           string
		body           string
		wantSubstrings []string
	}{
		{
			name:           "missing bucket returns agent error",
			body:           `{"object": "delete/nope.txt"}`,
			wantSubstrings: []string{"bucket"},
		},
		{
			name:           "missing object returns agent error",
			body:           fmt.Sprintf(`{"bucket": %q}`, bucket),
			wantSubstrings: []string{"object"},
		},
		{
			name:           "missing object in storage returns agent-visible error",
			body:           fmt.Sprintf(`{"bucket": %q, "object": "delete/does-not-exist.txt"}`, bucket),
			wantSubstrings: []string{"object", "does-not-exist"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result, status := invokeTool(t, "my_delete_object", tc.body)
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
}

func writeGCSObject(t *testing.T, ctx context.Context, client *storage.Client, bucket, object, contentType, body string) {
	t.Helper()
	w := client.Bucket(bucket).Object(object).NewWriter(ctx)
	w.ContentType = contentType
	if _, err := io.WriteString(w, body); err != nil {
		_ = w.Close()
		t.Fatalf("failed to write object %q: %v", object, err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer for object %q: %v", object, err)
	}
}

func readGCSObject(t *testing.T, ctx context.Context, client *storage.Client, bucket, object string) (string, *storage.ReaderObjectAttrs) {
	t.Helper()
	r, err := client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		t.Fatalf("failed to open object %q: %v", object, err)
	}
	defer r.Close()
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read object %q: %v", object, err)
	}
	return string(data), &r.Attrs
}

func gcsObjectExists(t *testing.T, ctx context.Context, client *storage.Client, bucket, object string) bool {
	t.Helper()
	_, err := client.Bucket(bucket).Object(object).Attrs(ctx)
	if err == nil {
		return true
	}
	if errors.Is(err, storage.ErrObjectNotExist) {
		return false
	}
	t.Fatalf("failed checking object %q existence: %v", object, err)
	return false
}
