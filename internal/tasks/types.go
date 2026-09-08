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

package tasks

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	TaskStatusWorking   TaskStatus = "working"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

// TaskProgress represents the progress of a task.
type TaskProgress struct {
	Total    float64 `json:"total,omitempty"`
	Progress float64 `json:"progress,omitempty"`
}

// TaskError represents an error that occurred during task execution.
type TaskError struct {
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// TaskStatusResult is returned by the AsyncTask when queried for status.
type TaskStatusResult struct {
	Status        TaskStatus    `json:"status"`
	Progress      *TaskProgress `json:"progress,omitempty"`
	StatusMessage string        `json:"statusMessage,omitempty"`
	Result        any           `json:"result,omitempty"`
	Error         *TaskError    `json:"error,omitempty"`
}

// CreateTaskResult is the payload returned when a task is initially created.
type CreateTaskResult struct {
	ResultType     string `json:"resultType"` // Should be "task"
	Status         TaskStatus `json:"status"`     // Typically "working"
	TaskID         string `json:"taskId"`
	PollIntervalMs int64  `json:"pollIntervalMs,omitempty"`
	TTLMs          int64  `json:"ttlMs,omitempty"`
}
