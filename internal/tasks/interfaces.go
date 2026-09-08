package tasks

import (
	"context"

	"github.com/googleapis/mcp-toolbox/internal/sources"
)

// AsyncTask defines the stateless proxy interface for managing long-running tasks.
type AsyncTask interface {
	GetStatus(ctx context.Context, s sources.Source, nativeID string) (*TaskStatusResult, error)
	Cancel(ctx context.Context, s sources.Source, nativeID string) error
}
