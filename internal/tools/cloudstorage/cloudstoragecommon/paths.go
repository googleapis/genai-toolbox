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

package cloudstoragecommon

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateLocalPath enforces the local-filesystem path contract used by
// download_object and upload_object: non-empty, absolute after filepath.Clean,
// and free of ".." components. It returns the cleaned path. OS permissions
// remain the real isolation boundary; this check just prevents obvious
// traversal mistakes and forces callers to be explicit about where they want
// bytes to land.
func ValidateLocalPath(p string) (string, error) {
	if p == "" {
		return "", fmt.Errorf("path is empty")
	}
	// Reject any ".." segment in the raw input. We check the raw input
	// (not just the cleaned output) so that escapes like
	// "/legit/../../etc/passwd" — which filepath.Clean collapses to an
	// innocuous-looking absolute path — are still rejected. Legitimate
	// names that happen to *contain* two dots (e.g. "foo..bar") are fine;
	// only a standalone ".." segment is disallowed.
	for _, seg := range strings.FieldsFunc(p, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if seg == ".." {
			return "", fmt.Errorf("path %q contains '..'", p)
		}
	}
	clean := filepath.Clean(p)
	if !filepath.IsAbs(clean) {
		return "", fmt.Errorf("path %q must be absolute", p)
	}
	return clean, nil
}
