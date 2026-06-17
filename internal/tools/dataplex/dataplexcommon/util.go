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
	"fmt"
	"strings"
)

// NormalizeResourcePath normalizes shorthand BigQuery table/dataset names or Cloud Storage
// bucket URIs into fully-qualified Dataplex resource URIs.
func NormalizeResourcePath(resourcePath, projectID string) string {
	resourcePath = strings.TrimSpace(resourcePath)
	if resourcePath == "" {
		return ""
	}

	// 1. Cloud Storage (GCS) normalization
	if strings.HasPrefix(resourcePath, "gs://") {
		bucketName := strings.TrimPrefix(resourcePath, "gs://")
		bucketName = strings.Split(bucketName, "/")[0]
		return fmt.Sprintf("//storage.googleapis.com/projects/%s/buckets/%s", projectID, bucketName)
	} else if strings.HasPrefix(resourcePath, "//storage.googleapis.com/buckets/") {
		bucketName := strings.TrimPrefix(resourcePath, "//storage.googleapis.com/buckets/")
		bucketName = strings.Split(bucketName, "/")[0]
		return fmt.Sprintf("//storage.googleapis.com/projects/%s/buckets/%s", projectID, bucketName)
	} else if strings.HasPrefix(resourcePath, "//storage.googleapis.com/projects/") {
		return resourcePath
	}

	// 2. BigQuery fully-qualified prefix check
	if strings.HasPrefix(resourcePath, "//bigquery.googleapis.com/") {
		return resourcePath
	}

	// 3. BigQuery relative projects/ path check
	if strings.HasPrefix(resourcePath, "projects/") {
		if strings.Contains(resourcePath, "/datasets/") {
			return "//bigquery.googleapis.com/" + resourcePath
		}
		// Return as-is for other projects/ paths (like Dataplex entries, Datascans, etc)
		return resourcePath
	}

	// 4. Dot-separated BigQuery shorthand (project.dataset.table or dataset.table)
	parts := strings.Split(resourcePath, ".")
	if len(parts) == 3 {
		return fmt.Sprintf("//bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", parts[0], parts[1], parts[2])
	} else if len(parts) == 2 {
		return fmt.Sprintf("//bigquery.googleapis.com/projects/%s/datasets/%s/tables/%s", projectID, parts[0], parts[1])
	}

	return resourcePath
}
