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
	"fmt"
	"regexp"
	"strings"
)

// Regular expressions for validating Firestore paths
var (
	// Pattern to detect absolute paths (those starting with "projects/")
	absolutePathRegex = regexp.MustCompile(`^projects/[^/]+/databases/[^/]+/documents/`)
)

// PathType represents the type of Firestore path
type PathType int

const (
	CollectionPath PathType = iota
	DocumentPath
)

// ValidateCollectionPath validates that a path is a valid Firestore collection path.
// Collection paths must have an odd number of segments (collection/doc/collection)
func ValidateCollectionPath(path string) error {
	return validatePath(path, CollectionPath)
}

// ValidateDocumentPath validates that a path is a valid Firestore document path.
// Document paths must have an even number of segments (collection/doc or collection/doc/collection/doc)
func ValidateDocumentPath(path string) error {
	return validatePath(path, DocumentPath)
}

// validatePath is the common validation function for both collection and document paths
func validatePath(path string, pathType PathType) error {
	pathTypeName := "document"
	if pathType == CollectionPath {
		pathTypeName = "collection"
	}

	// Check for empty path
	if path == "" {
		return fmt.Errorf("%s path cannot be empty", pathTypeName)
	}

	// Check if it's an absolute path
	if absolutePathRegex.MatchString(path) {
		example := "users/userId"
		if pathType == CollectionPath {
			example = "users"
		}
		return fmt.Errorf("path must be relative (e.g., '%s'), not absolute (matching pattern: ^projects/[^/]+/databases/[^/]+/documents/)", example)
	}

	// Split the path using strings.Split to preserve empty segments
	segments := strings.Split(path, "/")

	// Check for empty result
	if len(segments) == 0 {
		return fmt.Errorf("%s path cannot be empty or contain only slashes", pathTypeName)
	}

	// Check segment count based on path type
	segmentCount := len(segments)
	if pathType == CollectionPath && segmentCount%2 == 0 {
		// Collection paths must have an odd number of segments
		return fmt.Errorf("invalid collection path: must have an odd number of segments (e.g., 'collection' or 'collection/doc/subcollection'), got %d segments", segmentCount)
	} else if pathType == DocumentPath && segmentCount%2 != 0 {
		// Document paths must have an even number of segments
		return fmt.Errorf("invalid document path: must have an even number of segments (e.g., 'collection/doc'), got %d segments", segmentCount)
	}

	// Validate each segment
	for i, segment := range segments {
		isCollectionSegment := (i % 2) == 0
		if err := validateSegment(segment, isCollectionSegment); err != nil {
			return fmt.Errorf("invalid segment at position %d (%s): %w", i+1, segment, err)
		}
	}

	return nil
}

// validateSegment validates a single path segment
func validateSegment(segment string, isCollection bool) error {
	segmentType := "document ID"
	if isCollection {
		segmentType = "collection ID"
	}

	// Check for empty segment
	if segment == "" {
		return fmt.Errorf("segment cannot be empty")
	}

	// Check for whitespace-only segment
	if strings.TrimSpace(segment) == "" {
		return fmt.Errorf("segment cannot be only whitespace")
	}

	// Check for single or double period
	if segment == "." || segment == ".." {
		return fmt.Errorf("segment cannot be '.' or '..'")
	}

	// Check for reserved prefix
	if strings.HasPrefix(segment, "__") {
		return fmt.Errorf("%s cannot start with '__' (reserved prefix)", segmentType)
	}

	return nil
}

// IsAbsolutePath checks if a path is an absolute Firestore path
func IsAbsolutePath(path string) bool {
	return absolutePathRegex.MatchString(path)
}

// IsRelativePath checks if a path is a relative Firestore path
func IsRelativePath(path string) bool {
	return path != "" && !IsAbsolutePath(path)
}
