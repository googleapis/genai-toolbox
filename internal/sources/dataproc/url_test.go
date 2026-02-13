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

package dataproc

import (
	"testing"
	"time"

	"cloud.google.com/go/dataproc/v2/apiv1/dataprocpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestClusterConsoleURL(t *testing.T) {
	got := ClusterConsoleURL("my-project", "us-central1", "my-cluster")
	want := "https://console.cloud.google.com/dataproc/clusters/my-cluster/monitoring?region=us-central1&project=my-project"
	if got != want {
		t.Errorf("ClusterConsoleURL() = %v, want %v", got, want)
	}
}

func TestClusterLogsURL(t *testing.T) {
	startTime := time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 10, 1, 6, 0, 0, 0, time.UTC)

	got := ClusterLogsURL("my-project", "us-central1", "my-cluster", "my-uuid", startTime, endTime)
	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_cluster%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.region%3D%22us-central1%22" +
		"%0Aresource.labels.cluster_name%3D%22my-cluster%22" +
		"%0Aresource.labels.cluster_uuid%3D%22my-uuid%22" +
		"%0Atimestamp%3E%3D%222025-10-01T04%3A59%3A00Z%22" + // Minus 1 minute buffer from 5:00
		"%0Atimestamp%3C%3D%222025-10-01T06%3A10%3A00Z%22" + // Plus 10 minutes buffer from 6:00
		"&project=my-project" +
		"&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"
	if got != want {
		t.Errorf("ClusterLogsURL() = %v, want %v", got, want)
	}
}

func TestClusterLogsURLFromProto_Running(t *testing.T) {
	clusterPb := &dataprocpb.Cluster{
		ProjectId:   "my-project",
		ClusterName: "my-cluster",
		ClusterUuid: "my-uuid",
		Status: &dataprocpb.ClusterStatus{
			State:          dataprocpb.ClusterStatus_RUNNING,
			StateStartTime: timestamppb.New(time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC)),
		},
	}
	got := ClusterLogsURLFromProto(clusterPb, "us-central1")
	// Expect no timestamps
	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_cluster%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.region%3D%22us-central1%22" +
		"%0Aresource.labels.cluster_name%3D%22my-cluster%22" +
		"%0Aresource.labels.cluster_uuid%3D%22my-uuid%22" +
		"&project=my-project" +
		"&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"

	if got != want {
		t.Errorf("ClusterLogsURLFromProto(RUNNING) = %v, want %v", got, want)
	}
}

func TestClusterLogsURLFromProto_Error(t *testing.T) {
	stateTime := time.Date(2025, 10, 1, 6, 0, 0, 0, time.UTC)
	clusterPb := &dataprocpb.Cluster{
		ProjectId:   "my-project",
		ClusterName: "my-cluster",
		ClusterUuid: "my-uuid",
		Status: &dataprocpb.ClusterStatus{
			State:          dataprocpb.ClusterStatus_ERROR,
			StateStartTime: timestamppb.New(stateTime),
		},
	}
	got := ClusterLogsURLFromProto(clusterPb, "us-central1")

	// Start Time: stateTime - 1h = 05:00. Plus buffer (-1m) = 04:59:00.
	// End Time: stateTime = 06:00. Plus buffer (+10m) = 06:10:00.

	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_cluster%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.region%3D%22us-central1%22" +
		"%0Aresource.labels.cluster_name%3D%22my-cluster%22" +
		"%0Aresource.labels.cluster_uuid%3D%22my-uuid%22" +
		"%0Atimestamp%3E%3D%222025-10-01T04%3A59%3A00Z%22" +
		"%0Atimestamp%3C%3D%222025-10-01T06%3A10%3A00Z%22" +
		"&project=my-project" +
		"&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"

	if got != want {
		t.Errorf("ClusterLogsURLFromProto(ERROR) = %v, want %v", got, want)
	}
}

func TestClusterLogsURL_Escaping(t *testing.T) {
	startTime := time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 10, 1, 6, 0, 0, 0, time.UTC)

	// Input contains a double quote which should be escaped.
	clusterName := `my-cluster" OR root`
	got := ClusterLogsURL("my-project", "us-central1", clusterName, "my-uuid", startTime, endTime)

	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_cluster%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.region%3D%22us-central1%22" +
		// "my-cluster\" OR root" encoded
		"%0Aresource.labels.cluster_name%3D%22my-cluster%5C%22+OR+root%22" +
		"%0Aresource.labels.cluster_uuid%3D%22my-uuid%22" +
		"%0Atimestamp%3E%3D%222025-10-01T04%3A59%3A00Z%22" +
		"%0Atimestamp%3C%3D%222025-10-01T06%3A10%3A00Z%22" +
		"&project=my-project" +
		"&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster%22+OR+root"

	if got != want {
		t.Errorf("ClusterLogsURL_Escaping() = \n%v\nwant \n%v", got, want)
	}
}

func TestJobConsoleURL(t *testing.T) {
	got := JobConsoleURL("my-project", "us-central1", "my-job")
	want := "https://console.cloud.google.com/dataproc/jobs/my-job?region=us-central1&project=my-project"
	if got != want {
		t.Errorf("JobConsoleURL() = %v, want %v", got, want)
	}
}

func TestJobLogsURL(t *testing.T) {
	startTime := time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 10, 1, 6, 0, 0, 0, time.UTC)

	got := JobLogsURL("my-project", "us-central1", "my-cluster", "my-job", startTime, endTime)
	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_cluster%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.region%3D%22us-central1%22" +
		"%0Aresource.labels.cluster_name%3D%22my-cluster%22" +
		"%0Alabels.job_id%3D%22my-job%22" +
		"%0Atimestamp%3E%3D%222025-10-01T04%3A59%3A00Z%22" +
		"%0Atimestamp%3C%3D%222025-10-01T06%3A10%3A00Z%22" +
		"&project=my-project" +
		"&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"
	if got != want {
		t.Errorf("JobLogsURL() = %v, want %v", got, want)
	}
}

func TestJobLogsURLFromProto(t *testing.T) {
	startTime := time.Date(2025, 10, 1, 5, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 10, 1, 6, 0, 0, 0, time.UTC)

	jobPb := &dataprocpb.Job{
		Reference: &dataprocpb.JobReference{
			ProjectId: "my-project",
			JobId:     "my-job",
		},
		Placement: &dataprocpb.JobPlacement{
			ClusterName: "my-cluster",
		},
		StatusHistory: []*dataprocpb.JobStatus{
			{
				State:          dataprocpb.JobStatus_PENDING,
				StateStartTime: timestamppb.New(startTime),
			},
			{
				State:          dataprocpb.JobStatus_RUNNING,
				StateStartTime: timestamppb.New(startTime.Add(1 * time.Minute)),
			},
		},
		Status: &dataprocpb.JobStatus{
			State:          dataprocpb.JobStatus_DONE,
			StateStartTime: timestamppb.New(endTime),
		},
	}

	got, err := JobLogsURLFromProto(jobPb, "us-central1")
	if err != nil {
		t.Fatalf("JobLogsURLFromProto() error = %v", err)
	}

	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_cluster%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.region%3D%22us-central1%22" +
		"%0Aresource.labels.cluster_name%3D%22my-cluster%22" +
		"%0Alabels.job_id%3D%22my-job%22" +
		"%0Atimestamp%3E%3D%222025-10-01T04%3A59%3A00Z%22" +
		"%0Atimestamp%3C%3D%222025-10-01T06%3A10%3A00Z%22" +
		"&project=my-project" +
		"&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"

	if got != want {
		t.Errorf("JobLogsURLFromProto() = %v, want %v", got, want)
	}
}

func TestJobLogsURL_Escaping(t *testing.T) {
	// Input contains a double quote which should be escaped.
	jobID := `my-job" OR root`
	got := JobLogsURL("my-project", "us-central1", "my-cluster", jobID, time.Time{}, time.Time{})

	want := "https://console.cloud.google.com/logs/viewer?advancedFilter=" +
		"resource.type%3D%22cloud_dataproc_cluster%22" +
		"%0Aresource.labels.project_id%3D%22my-project%22" +
		"%0Aresource.labels.region%3D%22us-central1%22" +
		"%0Aresource.labels.cluster_name%3D%22my-cluster%22" +
		// "my-job\" OR root" encoded
		"%0Alabels.job_id%3D%22my-job%5C%22+OR+root%22" +
		"&project=my-project" +
		"&resource=cloud_dataproc_cluster%2Fcluster_name%2Fmy-cluster"

	if got != want {
		t.Errorf("JobLogsURL_Escaping() = \n%v\nwant \n%v", got, want)
	}
}
