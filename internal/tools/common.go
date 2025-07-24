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
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"text/template"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`)

func IsValidName(s string) bool {
	return validName.MatchString(s)
}

// ConvertAnySliceToTyped a []any to typed slice ([]string, []int, []float etc.)
func ConvertAnySliceToTyped(s []any, itemType string) (any, error) {
	var typedSlice any
	switch itemType {
	case "string":
		tempSlice := make([]string, len(s))
		for j, item := range s {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be string, got %T", j, item)
			}
			tempSlice[j] = s
		}
		typedSlice = tempSlice
	case "integer":
		tempSlice := make([]int64, len(s))
		for j, item := range s {
			i, ok := item.(int)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be integer, got %T", j, item)
			}
			tempSlice[j] = int64(i)
		}
		typedSlice = tempSlice
	case "float":
		tempSlice := make([]float64, len(s))
		for j, item := range s {
			f, ok := item.(float64)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be float, got %T", j, item)
			}
			tempSlice[j] = f
		}
		typedSlice = tempSlice
	case "boolean":
		tempSlice := make([]bool, len(s))
		for j, item := range s {
			b, ok := item.(bool)
			if !ok {
				return nil, fmt.Errorf("expected item at index %d to be boolean, got %T", j, item)
			}
			tempSlice[j] = b
		}
		typedSlice = tempSlice
	}
	return typedSlice, nil
}

// convertParamToJSON  is a Go template helper function to convert a parameter to JSON formatted string.
func convertParamToJSON(param any) (string, error) {
	jsonData, err := json.Marshal(param)
	if err != nil {
		return "", fmt.Errorf("failed to marshal param to JSON: %w", err)
	}
	return string(jsonData), nil
}

// PopulateTemplateWithJSON populate a Go template with a custom `json` array formatter
func PopulateTemplateWithJSON(templateName, templateString string, data map[string]any) (string, error) {
	funcMap := template.FuncMap{
		"json": convertParamToJSON,
	}

	tmpl, err := template.New(templateName).Funcs(funcMap).Parse(templateString)
	if err != nil {
		return "", fmt.Errorf("error parsing template '%s': %w", templateName, err)
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("error executing template '%s': %w", templateName, err)
	}
	return result.String(), nil
}

// CheckDuplicateParameters verify there are no duplicate parameter names
func CheckDuplicateParameters(ps Parameters) error {
	seenNames := make(map[string]bool)
	for _, p := range ps {
		pName := p.GetName()
		if _, exists := seenNames[pName]; exists {
			return fmt.Errorf("parameter name must be unique across all parameter fields. Duplicate parameter: %s", pName)
		}
		seenNames[pName] = true
	}
	return nil
}
