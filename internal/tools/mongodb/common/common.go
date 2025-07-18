package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

// helper function to convert a parameter to JSON formatted string.
func ConvertParamToJSON(param any) (string, error) {
	jsonData, err := json.Marshal(param)
	if err != nil {
		return "", fmt.Errorf("failed to marshal param to JSON: %w", err)
	}
	return string(jsonData), nil
}

func ParsePayloadTemplate(params tools.Parameters, payload string, paramsMap map[string]any) (string, error) {
	// Create a map for request body parameters
	cleanParamsMap := make(map[string]any)
	for _, p := range params {
		k := p.GetName()
		v, ok := paramsMap[k]
		if !ok {
			return "", fmt.Errorf("missing parameter %s", k)
		}
		cleanParamsMap[k] = v
	}

	// Create a FuncMap to format array parameters
	funcMap := template.FuncMap{
		"json": ConvertParamToJSON,
	}
	templ, err := template.New("template").Funcs(funcMap).Parse(payload)
	if err != nil {
		return "", fmt.Errorf("error parsing: %s", err)
	}
	var result bytes.Buffer
	err = templ.Execute(&result, cleanParamsMap)
	if err != nil {
		return "", fmt.Errorf("error replacing payload: %s", err)
	}
	return result.String(), nil
}

func GetUpdate(params tools.Parameters, payload string, paramsMap map[string]any) (string, error) {
	// Create a map for request body parameters
	cleanParamsMap := make(map[string]any)
	for _, p := range params {
		k := p.GetName()
		v, ok := paramsMap[k]
		if !ok {
			return "", fmt.Errorf("missing update parameter %s", k)
		}
		cleanParamsMap[k] = v
	}

	// Create a FuncMap to format array parameters
	funcMap := template.FuncMap{
		"json": ConvertParamToJSON,
	}
	templ, err := template.New("filter").Funcs(funcMap).Parse(payload)
	if err != nil {
		return "", fmt.Errorf("error parsing filter: %s", err)
	}
	var result bytes.Buffer
	err = templ.Execute(&result, cleanParamsMap)
	if err != nil {
		return "", fmt.Errorf("error replacing filter payload: %s", err)
	}
	return result.String(), nil
}
