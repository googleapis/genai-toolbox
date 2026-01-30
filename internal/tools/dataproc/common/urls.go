// Copyright 2026 Google LLC
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
	"time"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
)

const (
	logTimeBufferBefore = 1 * time.Minute
	logTimeBufferAfter  = 10 * time.Minute
)

// ClusterConsoleURLFromProto builds a URL to the Google Cloud Console linking to the cluster monitoring page.
func ClusterConsoleURLFromProto(clusterPb *dataprocpb.Cluster, region string) string {
	return ClusterConsoleURL(clusterPb.ProjectId, region, clusterPb.ClusterName)
}

// ClusterLogsURLFromProto builds a URL to the Google Cloud Console showing Cloud Logging for the given cluster.
func ClusterLogsURLFromProto(clusterPb *dataprocpb.Cluster, region string) string {
	var startTime, endTime time.Time

	if clusterPb.Status != nil {
		state := clusterPb.GetStatus().GetState()
		stateStartTime := clusterPb.GetStatus().GetStateStartTime().AsTime()

		// If the cluster is in a terminal or stopping state, we bound the logs
		// to the hour preceding the state change to help with debugging.
		switch state {
		case dataprocpb.ClusterStatus_ERROR,
			dataprocpb.ClusterStatus_DELETING,
			dataprocpb.ClusterStatus_STOPPED,
			dataprocpb.ClusterStatus_STOPPING:
			if !stateStartTime.IsZero() {
				endTime = stateStartTime
				startTime = stateStartTime.Add(-1 * time.Hour)
			}
		}
	}

	return ClusterLogsURL(clusterPb.ProjectId, region, clusterPb.ClusterName, clusterPb.ClusterUuid, startTime, endTime)
}

// ClusterConsoleURL builds a URL to the Google Cloud Console linking to the cluster monitoring page.
func ClusterConsoleURL(projectID, region, clusterName string) string {
	return fmt.Sprintf("https://console.cloud.google.com/dataproc/clusters/%s/monitoring?region=%s&project=%s", clusterName, region, projectID)
}

// ClusterLogsURL builds a URL to the Google Cloud Console showing Cloud Logging for the given cluster and time range.
func ClusterLogsURL(projectID, region, clusterName, clusterUUID string, startTime, endTime time.Time) string {
	advancedFilterTemplate := `resource.type="cloud_dataproc_cluster"
resource.labels.project_id=%s
resource.labels.region=%s
resource.labels.cluster_name=%s`
	// Use %q to quote and escape the strings.
	advancedFilter := fmt.Sprintf(advancedFilterTemplate, fmt.Sprintf("%q", projectID), fmt.Sprintf("%q", region), fmt.Sprintf("%q", clusterName))

	if clusterUUID != "" {
		advancedFilter += fmt.Sprintf("\nresource.labels.cluster_uuid=%q", clusterUUID)
	}

	if !startTime.IsZero() {
		actualStart := startTime.Add(-1 * logTimeBufferBefore)
		advancedFilter += fmt.Sprintf("\ntimestamp>=\"%s\"", actualStart.Format(time.RFC3339Nano))
	}
	if !endTime.IsZero() {
		actualEnd := endTime.Add(logTimeBufferAfter)
		advancedFilter += fmt.Sprintf("\ntimestamp<=\"%s\"", actualEnd.Format(time.RFC3339Nano))
	}

	v := url.Values{}
	v.Add("resource", "cloud_dataproc_cluster/cluster_name/"+clusterName)
	v.Add("advancedFilter", advancedFilter)
	v.Add("project", projectID)

	return "https://console.cloud.google.com/logs/viewer?" + v.Encode()
}
