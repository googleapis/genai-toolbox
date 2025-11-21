// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package common

import (
	"testing"
	"time"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestExtractBatchDetails_Success(t *testing.T) {
	batchName := "projects/my-project/locations/us-central1/batches/my-batch"
	projectID, location, batchID, err := ExtractBatchDetails(batchName)
	if err != nil {
		t.Errorf("ExtractBatchDetails() error = %v, want no error", err)
		return
	}
	wantProject := "my-project"
	wantLocation := "us-central1"
	wantBatchID := "my-batch"
	if projectID != wantProject {
		t.Errorf("ExtractBatchDetails() projectID = %v, want %v", projectID, wantProject)
	}
	if location != wantLocation {
		t.Errorf("ExtractBatchDetails() location = %v, want %v", location, wantLocation)
	}
	if batchID != wantBatchID {
		t.Errorf("ExtractBatchDetails() batchID = %v, want %v", batchID, wantBatchID)
	}
}

func TestExtractBatchDetails_Failure(t *testing.T) {
	batchName := "invalid-name"
	_, _, _, err := ExtractBatchDetails(batchName)
	wantErr := "failed to parse batch name: invalid-name"
	if err == nil || err.Error() != wantErr {
		t.Errorf("ExtractBatchDetails() error = %v, want %v", err, wantErr)
	}
}

func TestBatchConsoleURL(t *testing.T) {
	got := BatchConsoleURL("my-project", "us-central1", "my-batch")
	want := "https://console.cloud.google.com/dataproc/batches/us-central1/my-batch/summary?project=my-project"
	if got != want {
		t.Errorf("BatchConsoleURL() = %v, want %v", got, want)
	}
}

func TestBatchLogsURL(t *testing.T) {
	startTime := time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 10, 1, 6, 0, 0, 0, time.UTC)
	got := BatchLogsURL("my-project", "us-central1", "my-batch", startTime, endTime)
	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_batch%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.location%3D%22us-central1%22" +
		"%0Aresource.labels.batch_id%3D%22my-batch%22" +
		"%0Atimestamp%3E%3D%222025-10-01T04%3A59%3A00Z%22" + // Minus 1 minute
		"%0Atimestamp%3C%3D%222025-10-01T06%3A10%3A00Z%22" + // Plus 10 minutes
		"&project=my-project" +
		"&resource=cloud_dataproc_batch%2Fbatch_id%2Fmy-batch"
	if got != want {
		t.Errorf("BatchLogsURL() = %v, want %v", got, want)
	}
}

func TestBatchConsoleURLFromProto(t *testing.T) {
	batchPb := &dataprocpb.Batch{
		Name: "projects/my-project/locations/us-central1/batches/my-batch",
	}
	got, err := BatchConsoleURLFromProto(batchPb)
	if err != nil {
		t.Fatalf("BatchConsoleURLFromProto() error = %v", err)
	}
	want := "https://console.cloud.google.com/dataproc/batches/us-central1/my-batch/summary?project=my-project"
	if got != want {
		t.Errorf("BatchConsoleURLFromProto() = %v, want %v", got, want)
	}
}

func TestBatchLogsURLFromProto(t *testing.T) {
	createTime := time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC)
	stateTime := time.Date(2025, 10, 1, 6, 0, 0, 0, time.UTC)
	batchPb := &dataprocpb.Batch{
		Name:       "projects/my-project/locations/us-central1/batches/my-batch",
		CreateTime: timestamppb.New(createTime),
		StateTime:  timestamppb.New(stateTime),
	}
	got, err := BatchLogsURLFromProto(batchPb)
	if err != nil {
		t.Fatalf("BatchLogsURLFromProto() error = %v", err)
	}
	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_batch%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.location%3D%22us-central1%22" +
		"%0Aresource.labels.batch_id%3D%22my-batch%22" +
		"%0Atimestamp%3E%3D%222025-10-01T04%3A59%3A00Z%22" + // Minus 1 minute
		"%0Atimestamp%3C%3D%222025-10-01T06%3A10%3A00Z%22" + // Plus 10 minutes
		"&project=my-project" +
		"&resource=cloud_dataproc_batch%2Fbatch_id%2Fmy-batch"
	if got != want {
		t.Errorf("BatchLogsURLFromProto() = %v, want %v", got, want)
	}
}
