// Copyright 2026 Google LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import "fmt"

type ErrorCategory string

const (
	CategoryAgent  ErrorCategory = "AGENT_ERROR"
	CategoryServer ErrorCategory = "SERVER_ERROR"
)

// ToolboxError is the interface all custom errors must satisfy
type ToolboxError interface {
	error
	Category() ErrorCategory
}

// Agent Errors return 200 to the sender
type AgentError struct {
	Msg   string
	Cause error
}

func (e *AgentError) Error() string { return e.Msg }

func (e *AgentError) Category() ErrorCategory { return CategoryAgent }

func (e *AgentError) Unwrap() error { return e.Cause }

func NewAgentError(msg string, args ...any) *AgentError {
	return &AgentError{Msg: fmt.Sprintf(msg, args...)}
}

// Server Errors return specified error code
type ServerError struct {
	Msg   string
	Code  int
	Cause error
}

func (e *ServerError) Error() string { return fmt.Sprintf("%s: %v", e.Msg, e.Cause) }

func (e *ServerError) Category() ErrorCategory { return CategoryServer }

func (e *ServerError) Unwrap() error { return e.Cause }

func NewServerError(msg string, code int, cause error) *ServerError {
	return &ServerError{Msg: msg, Code: code, Cause: cause}
}
