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
	"fmt"
	"net/url"
	"regexp"
	"time"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
)

const (
	logTimeBufferBefore = 1 * time.Minute
	logTimeBufferAfter  = 10 * time.Minute
)

var batchFullNameRegex = regexp.MustCompile(`projects/(?P<project>[^/]+)/locations/(?P<location>[^/]+)/batches/(?P<batch_id>[^/]+)`)

// Extract BatchDetails extracts the project ID, location, and batch ID from a fully qualified batch name.
func ExtractBatchDetails(batchName string) (projectID, location, batchID string, err error) {
	matches := batchFullNameRegex.FindStringSubmatch(batchName)
	if len(matches) < 4 {
		return "", "", "", fmt.Errorf("failed to parse batch name: %s", batchName)
	}
	return matches[1], matches[2], matches[3], nil
}

// BatchConsoleURLFromProto builds a URL to the Google Cloud Console linking to the batch summary page.
func BatchConsoleURLFromProto(batchPb *dataprocpb.Batch) (string, error) {
	projectID, location, batchID, err := ExtractBatchDetails(batchPb.GetName())
	if err != nil {
		return "", err
	}
	return BatchConsoleURL(projectID, location, batchID), nil
}

// BatchLogsURLFromProto builds a URL to the Google Cloud Console showing Cloud Logging for the given batch and time range.
func BatchLogsURLFromProto(batchPb *dataprocpb.Batch) (string, error) {
	projectID, location, batchID, err := ExtractBatchDetails(batchPb.GetName())
	if err != nil {
		return "", err
	}
	createTime := batchPb.GetCreateTime().AsTime()
	stateTime := batchPb.GetStateTime().AsTime()
	return BatchLogsURL(projectID, location, batchID, createTime, stateTime), nil
}

// BatchConsoleURL builds a URL to the Google Cloud Console linking to the batch summary page.
func BatchConsoleURL(projectID, location, batchID string) string {
	return fmt.Sprintf("https://console.cloud.google.com/dataproc/batches/%s/%s/summary?project=%s", location, batchID, projectID)
}

// BatchLogsURL builds a URL to the Google Cloud Console showing Cloud Logging for the given batch and time range.
//
// The implementation adds some buffer before and after the provided times.
func BatchLogsURL(projectID, location, batchID string, startTime, endTime time.Time) string {
	advancedFilterTemplate := `resource.type="cloud_dataproc_batch"
resource.labels.project_id="%s"
resource.labels.location="%s"
resource.labels.batch_id="%s"`
	advancedFilter := fmt.Sprintf(advancedFilterTemplate, projectID, location, batchID)
	if !startTime.IsZero() {
		actualStart := startTime.Add(-1 * logTimeBufferBefore)
		advancedFilter += fmt.Sprintf("\ntimestamp>=\"%s\"", actualStart.Format(time.RFC3339Nano))
	}
	if !endTime.IsZero() {
		actualEnd := endTime.Add(logTimeBufferAfter)
		advancedFilter += fmt.Sprintf("\ntimestamp<=\"%s\"", actualEnd.Format(time.RFC3339Nano))
	}

	v := url.Values{}
	v.Add("resource", "cloud_dataproc_batch/batch_id/"+batchID)
	v.Add("advancedFilter", advancedFilter)
	v.Add("project", projectID)

	return "https://console.cloud.google.com/logs/viewer?" + v.Encode()
}
