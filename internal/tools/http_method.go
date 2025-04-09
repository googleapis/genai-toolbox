// Copyright 2025 Google LLC
//
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
package tools

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// HTTPMethod is a string of a valid HTTP method (e.g "GET")
type HTTPMethod string

// isValidHTTPMethod checks if the input string matches one of the method constants defined in the net/http package
func isValidHTTPMethod(method string) bool {

	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete,
		http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace,
		http.MethodConnect:
		return true
	}
	return false
}

func (i *HTTPMethod) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	var httpMethod string
	if err := unmarshal(&httpMethod); err != nil {
		return fmt.Errorf(`error unmarshalling HTTP method: %s`, err)
	}
	httpMethod = strings.ToUpper(httpMethod)
	if !isValidHTTPMethod(httpMethod) {
		return fmt.Errorf(`%s is not a valid http method`, httpMethod)
	}
	*i = HTTPMethod(httpMethod)
	return nil
}
