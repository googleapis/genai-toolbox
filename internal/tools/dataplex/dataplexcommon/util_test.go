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

package dataplexcommon

import (
	"testing"
)

func TestNormalizeResourcePath(t *testing.T) {
	projectID := "test-project"

	tests := []struct {
		name         string
		resourcePath string
		want         string
	}{
		{
			name:         "empty path",
			resourcePath: "",
			want:         "",
		},
		{
			name:         "gs URI",
			resourcePath: "gs://my-bucket/data/file.csv",
			want:         "//storage.googleapis.com/projects/test-project/buckets/my-bucket",
		},
		{
			name:         "storage buckets prefix",
			resourcePath: "//storage.googleapis.com/buckets/my-bucket/other/path",
			want:         "//storage.googleapis.com/projects/test-project/buckets/my-bucket",
		},
		{
			name:         "fully qualified GCS path",
			resourcePath: "//storage.googleapis.com/projects/other-proj/buckets/my-bucket",
			want:         "//storage.googleapis.com/projects/other-proj/buckets/my-bucket",
		},
		{
			name:         "fully qualified BigQuery path",
			resourcePath: "//bigquery.googleapis.com/projects/p/datasets/d/tables/t",
			want:         "//bigquery.googleapis.com/projects/p/datasets/d/tables/t",
		},
		{
			name:         "projects prefixed BigQuery path",
			resourcePath: "projects/p/datasets/d/tables/t",
			want:         "//bigquery.googleapis.com/projects/p/datasets/d/tables/t",
		},
		{
			name:         "dataset.table shorthand",
			resourcePath: "my_dataset.my_table",
			want:         "//bigquery.googleapis.com/projects/test-project/datasets/my_dataset/tables/my_table",
		},
		{
			name:         "project.dataset.table shorthand",
			resourcePath: "other-proj.my_dataset.my_table",
			want:         "//bigquery.googleapis.com/projects/other-proj/datasets/my_dataset/tables/my_table",
		},
		{
			name:         "unrecognized raw string",
			resourcePath: "raw_string_without_dots",
			want:         "raw_string_without_dots",
		},
		{
			name:         "dataplex entry path",
			resourcePath: "projects/my-project/locations/us-central1/entryGroups/my-group/entries/my-entry",
			want:         "projects/my-project/locations/us-central1/entryGroups/my-group/entries/my-entry",
		},
		{
			name:         "dataplex datascan path",
			resourcePath: "projects/my-project/locations/us-central1/dataScans/my-scan",
			want:         "projects/my-project/locations/us-central1/dataScans/my-scan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeResourcePath(tt.resourcePath, projectID)
			if got != tt.want {
				t.Errorf("NormalizeResourcePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildResourceURI(t *testing.T) {
	tests := []struct {
		name               string
		resourceType       string
		resourceProjectID  string
		resourceLocationID string
		resourceDatasetID  string
		resourceID         string
		want               string
	}{
		{
			name:              "BIGQUERY_DATASET",
			resourceType:      "BIGQUERY_DATASET",
			resourceProjectID: "proj-1",
			resourceDatasetID: "ds-1",
			want:              "//bigquery.googleapis.com/projects/proj-1/datasets/ds-1",
		},
		{
			name:              "BIGQUERY_TABLE",
			resourceType:      "BIGQUERY_TABLE",
			resourceProjectID: "proj-1",
			resourceDatasetID: "ds-1",
			resourceID:        "tbl-1",
			want:              "//bigquery.googleapis.com/projects/proj-1/datasets/ds-1/tables/tbl-1",
		},
		{
			name:               "GEMINI_ENTERPRISE_AGENT_PLATFORM_DATASET",
			resourceType:       "GEMINI_ENTERPRISE_AGENT_PLATFORM_DATASET",
			resourceProjectID:  "proj-1",
			resourceLocationID: "us-central1",
			resourceID:         "ds-id",
			want:               "//aiplatform.googleapis.com/projects/proj-1/locations/us-central1/datasets/ds-id",
		},
		{
			name:         "CLOUD_STORAGE_BUCKET",
			resourceType: "CLOUD_STORAGE_BUCKET",
			resourceID:   "my-bucket",
			want:         "//storage.googleapis.com/projects/_/buckets/my-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildResourceURI(tt.resourceType, tt.resourceProjectID, tt.resourceLocationID, tt.resourceDatasetID, tt.resourceID)
			if got != tt.want {
				t.Errorf("BuildResourceURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
