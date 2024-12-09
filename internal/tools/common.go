// Copyright 2024 Google LLC
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

package tools

import (
	"regexp"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`)

func IsValidName(s string) bool {
	return validName.MatchString(s)
}

func Authorized(parameters []Parameter, claims map[string]map[string]any) bool {
	for _, p := range parameters {
		if p.GetAuthSources() == nil {
			// skip non-auth parameters
			continue
		}
		isAuthorized := false
		for _, paramAuthSource := range p.GetAuthSources() {
			if _, ok := claims[paramAuthSource.Name]; ok {
				// param auth source found
				isAuthorized = true
				break
			}
		}
		if !isAuthorized {
			return false
		}
	}
	return true
}
