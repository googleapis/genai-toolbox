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
	"strings"

	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
	"github.com/thlib/go-timezone-local/tzlocal"
)

const (
	DimensionsFields = "fields(dimensions(name,type,label,label_short,description,synonyms,tags,hidden))"
	FiltersFields    = "fields(filters(name,type,label,label_short,description,synonyms,tags,hidden))"
	MeasuresFields   = "fields(measures(name,type,label,label_short,description,synonyms,tags,hidden))"
	ParametersFields = "fields(parameters(name,type,label,label_short,description,synonyms,tags,hidden))"
)

// ExtractLookerFieldProperties extracts common properties from Looker field objects.
func ExtractLookerFieldProperties(ctx context.Context, fields *[]v4.LookmlModelExploreField, showHiddenFields bool) ([]any, error) {
	data := make([]any, 0)

	// Handle nil fields pointer
	if fields == nil {
		return data, nil
	}

	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		// This should ideally not happen if the context is properly set up.
		// Log and return an empty map or handle as appropriate for your error strategy.
		return data, fmt.Errorf("error getting logger from context in ExtractLookerFieldProperties: %v", err)
	}

	for _, v := range *fields {
		logger.DebugContext(ctx, "Got response element of %v\n", v)
		if v.Name != nil && strings.HasSuffix(*v.Name, "_raw") {
			continue
		}
		if !showHiddenFields && v.Hidden != nil && *v.Hidden {
			continue
		}
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
		if v.Description != nil {
			vMap["description"] = *v.Description
		}
		if v.Tags != nil {
			vMap["tags"] = *v.Tags
		}
		if v.Synonyms != nil {
			vMap["synonyms"] = *v.Synonyms
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

func GetFieldParameters() tools.Parameters {
	modelParameter := tools.NewStringParameter("model", "The model containing the explore.")
	exploreParameter := tools.NewStringParameter("explore", "The explore containing the fields.")
	return tools.Parameters{modelParameter, exploreParameter}
}

func GetQueryParameters() tools.Parameters {
	modelParameter := tools.NewStringParameter("model", "The model containing the explore.")
	exploreParameter := tools.NewStringParameter("explore", "The explore to be queried.")
	fieldsParameter := tools.NewArrayParameter("fields",
		"The fields to be retrieved.",
		tools.NewStringParameter("field", "A field to be returned in the query"),
	)
	filtersParameter := tools.NewMapParameterWithDefault("filters",
		map[string]any{},
		"The filters for the query",
		"",
	)
	pivotsParameter := tools.NewArrayParameterWithDefault("pivots",
		[]any{},
		"The query pivots (must be included in fields as well).",
		tools.NewStringParameter("pivot_field", "A field to be used as a pivot in the query"),
	)
	sortsParameter := tools.NewArrayParameterWithDefault("sorts",
		[]any{},
		"The sorts like \"field.id desc 0\".",
		tools.NewStringParameter("sort_field", "A field to be used as a sort in the query"),
	)
	limitParameter := tools.NewIntParameterWithDefault("limit", 500, "The row limit.")
	tzParameter := tools.NewStringParameterWithRequired("tz", "The query timezone.", false)

	return tools.Parameters{
		modelParameter,
		exploreParameter,
		fieldsParameter,
		filtersParameter,
		pivotsParameter,
		sortsParameter,
		limitParameter,
		tzParameter,
	}
}

func ProcessFieldArgs(ctx context.Context, params tools.ParamValues) (*string, *string, error) {
	mapParams := params.AsMap()
	model, ok := mapParams["model"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("'model' must be a string, got %T", mapParams["model"])
	}
	explore, ok := mapParams["explore"].(string)
	if !ok {
		return nil, nil, fmt.Errorf("'explore' must be a string, got %T", mapParams["explore"])
	}
	return &model, &explore, nil
}

func ProcessQueryArgs(ctx context.Context, params tools.ParamValues) (*v4.WriteQuery, error) {
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get logger from ctx: %s", err)
	}

	logger.DebugContext(ctx, "params = ", params)
	paramsMap := params.AsMap()

	f, err := tools.ConvertAnySliceToTyped(paramsMap["fields"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert fields to array of strings: %s", err)
	}
	fields := f.([]string)
	filters := paramsMap["filters"].(map[string]any)
	// Sometimes filters come as "'field.id'": "expression" so strip extra ''
	for k, v := range filters {
		if len(k) > 0 && k[0] == '\'' && k[len(k)-1] == '\'' {
			delete(filters, k)
			filters[k[1:len(k)-1]] = v
		}
	}
	p, err := tools.ConvertAnySliceToTyped(paramsMap["pivots"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert pivots to array of strings: %s", err)
	}
	pivots := p.([]string)
	s, err := tools.ConvertAnySliceToTyped(paramsMap["sorts"].([]any), "string")
	if err != nil {
		return nil, fmt.Errorf("can't convert sorts to array of strings: %s", err)
	}
	sorts := s.([]string)
	limit := fmt.Sprintf("%v", paramsMap["limit"].(int))

	var tz string
	if paramsMap["tz"] != nil {
		tz = paramsMap["tz"].(string)
	} else {
		tzname, err := tzlocal.RuntimeTZ()
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error getting local timezone: %s", err))
			tzname = "Etc/UTC"
		}
		tz = tzname
	}

	wq := v4.WriteQuery{
		Model:         paramsMap["model"].(string),
		View:          paramsMap["explore"].(string),
		Fields:        &fields,
		Pivots:        &pivots,
		Filters:       &filters,
		Sorts:         &sorts,
		QueryTimezone: &tz,
		Limit:         &limit,
	}
	return &wq, nil
}
