// Copyright 2025 Google LLC
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

package util

import (
	"strings"
	"testing"
)

func TestValidateCollectionPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "valid root collection",
			path:    "users",
			wantErr: false,
		},
		{
			name:    "valid subcollection",
			path:    "users/user123/posts",
			wantErr: false,
		},
		{
			name:    "valid deeply nested",
			path:    "users/user123/posts/post456/comments",
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "collection path cannot be empty",
		},
		{
			name:    "even segments (document path)",
			path:    "users/user123",
			wantErr: true,
			errMsg:  "must have an odd number of segments",
		},
		{
			name:    "absolute path",
			path:    "projects/my-project/databases/(default)/documents/users",
			wantErr: true,
			errMsg:  "path must be relative",
		},
		{
			name:    "reserved prefix __",
			path:    "__users",
			wantErr: true,
			errMsg:  "collection ID cannot start with '__'",
		},
		{
			name:    "dot segment",
			path:    "users/./posts",
			wantErr: true,
			errMsg:  "segment cannot be '.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCollectionPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCollectionPath(%q) expected error but got none", tt.path)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCollectionPath(%q) error = %v, want error containing %q", tt.path, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCollectionPath(%q) unexpected error: %v", tt.path, err)
				}
			}
		})
	}
}

func TestValidateDocumentPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
		errMsg  string
	}{
		// Valid cases
		{
			name:    "valid root document",
			path:    "users/user123",
			wantErr: false,
		},
		{
			name:    "valid nested document",
			path:    "users/user123/posts/post456",
			wantErr: false,
		},
		{
			name:    "valid deeply nested",
			path:    "users/user123/posts/post456/comments/comment789",
			wantErr: false,
		},

		// Invalid cases
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errMsg:  "document path cannot be empty",
		},
		{
			name:    "odd segments (collection path)",
			path:    "users",
			wantErr: true,
			errMsg:  "must have an even number of segments",
		},
		{
			name:    "absolute path",
			path:    "projects/my-project/databases/(default)/documents/users/user123",
			wantErr: true,
			errMsg:  "path must be relative",
		},
		{
			name:    "reserved prefix __",
			path:    "users/__user123",
			wantErr: true,
			errMsg:  "document ID cannot start with '__'",
		},
		{
			name:    "double dot segment",
			path:    "users/..",
			wantErr: true,
			errMsg:  "segment cannot be '.'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDocumentPath(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateDocumentPath(%q) expected error but got none", tt.path)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateDocumentPath(%q) error = %v, want error containing %q", tt.path, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateDocumentPath(%q) unexpected error: %v", tt.path, err)
				}
			}
		})
	}
}
