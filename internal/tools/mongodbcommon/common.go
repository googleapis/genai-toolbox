package mongodbcommon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"text/template"
)

// helper function to convert a parameter to JSON formatted string.
func ConvertParamToJSON(param any) (string, error) {
	jsonData, err := json.Marshal(param)
	if err != nil {
		return "", fmt.Errorf("failed to marshal param to JSON: %w", err)
	}
	return string(jsonData), nil
}

func GetFilter(filterParams tools.Parameters, filterPayload string, paramsMap map[string]any) (string, error) {
	// Create a map for request body parameters
	filterParamsMap := make(map[string]any)
	for _, p := range filterParams {
		k := p.GetName()
		v, ok := paramsMap[k]
		if !ok {
			return "", fmt.Errorf("missing filter parameter %s", k)
		}
		filterParamsMap[k] = v
	}

	// Create a FuncMap to format array parameters
	funcMap := template.FuncMap{
		"json": ConvertParamToJSON,
	}
	templ, err := template.New("filter").Funcs(funcMap).Parse(filterPayload)
	if err != nil {
		return "", fmt.Errorf("error parsing filter: %s", err)
	}
	var result bytes.Buffer
	err = templ.Execute(&result, filterParamsMap)
	if err != nil {
		return "", fmt.Errorf("error replacing filter payload: %s", err)
	}
	return result.String(), nil
}
