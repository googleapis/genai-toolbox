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
package lookercommon

import (
	"context"
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/util"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const (
	DimensionsFields = "fields(dimensions(name,type,label,label_short))"
	FiltersFields    = "fields(filters(name,type,label,label_short))"
	MeasuresFields   = "fields(measures(name,type,label,label_short))"
	ParametersFields = "fields(parameters(name,type,label,label_short))"
)

// ExtractLookerFieldProperties extracts common properties from Looker field objects.
func ExtractLookerFieldProperties(ctx context.Context, fields *[]v4.LookmlModelExploreField) ([]any, error) {
	var data []any

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		// This should ideally not happen if the context is properly set up.
		// Log and return an empty map or handle as appropriate for your error strategy.
		return data, fmt.Errorf("error getting logger from context in ExtractLookerFieldProperties: %v", err)
	}

	for _, v := range *fields {
		logger.DebugContext(ctx, "Got response element of %v\n", v)
		vMap := make(map[string]any)
		if v.Name != nil {
			vMap["name"] = *v.Name
		}
		if v.Type != nil {
			vMap["type"] = *v.Type
		}
		if v.Label != nil {
			vMap["label"] = *v.Label
		}
		if v.LabelShort != nil {
			vMap["label_short"] = *v.LabelShort
		}
		logger.DebugContext(ctx, "Converted to %v\n", vMap)
		data = append(data, vMap)
	}

	return data, nil
}

// CheckLookerExploreFields checks if the Fields object in LookmlModelExplore is nil before accessing its sub-fields.
func CheckLookerExploreFields(resp *v4.LookmlModelExplore) error {
	if resp == nil || resp.Fields == nil {
		return fmt.Errorf("looker API response or its fields object is nil")
	}
	return nil
}
