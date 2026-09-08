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
