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

package bigqueryaskdatainsights

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	bigqueryds "github.com/googleapis/genai-toolbox/internal/sources/bigquery"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"golang.org/x/oauth2"
)

const kind string = "bigquery-ask-data-insights"

const instructions = `**INSTRUCTIONS - FOLLOW THESE RULES:**
1. **CONTENT:** Your answer should present the supporting data and then provide a conclusion based on that data.
2. **OUTPUT FORMAT:** Your entire response MUST be in plain text format ONLY.
3. **NO CHARTS:** You are STRICTLY FORBIDDEN from generating any charts, graphs, images, or any other form of visualization.`

func init() {
	if !tools.Register(kind, newConfig) {
		panic(fmt.Sprintf("tool kind %q already registered", kind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	BigQueryClient() *bigqueryapi.Client
	BigQueryTokenSource() oauth2.TokenSource
	GetMaxQueryResultRows() int
}

type BQTableReference struct {
	ProjectID string `json:"projectId"`
	DatasetID string `json:"datasetId"`
	TableID   string `json:"tableId"`
}

// Structs for building the JSON payload
type UserMessage struct {
	Text string `json:"text"`
}
type Message struct {
	UserMessage UserMessage `json:"userMessage"`
}
type BQDatasource struct {
	TableReferences []BQTableReference `json:"tableReferences"`
}
type DatasourceReferences struct {
	BQ BQDatasource `json:"bq"`
}
type ImageOptions struct {
	NoImage map[string]any `json:"noImage"`
}
type ChartOptions struct {
	Image ImageOptions `json:"image"`
}
type Options struct {
	Chart ChartOptions `json:"chart"`
}
type InlineContext struct {
	DatasourceReferences DatasourceReferences `json:"datasourceReferences"`
	Options              Options              `json:"options"`
}

type CAPayload struct {
	Project       string        `json:"project"`
	Messages      []Message     `json:"messages"`
	InlineContext InlineContext `json:"inlineContext"`
}

// validate compatible sources are still compatible
var _ compatibleSource = &bigqueryds.Source{}

var compatibleSources = [...]string{bigqueryds.SourceKind}

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigKind() string {
	return kind
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	// verify source exists
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}

	// verify the source is compatible
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be one of %q", kind, compatibleSources)
	}

	userQueryParameter := tools.NewStringParameter("user_query_with_context", "The user's question, potentially including conversation history and system instructions for context.")
	tableRefsParameter := tools.NewStringParameter("table_references", `A JSON string of a list of BigQuery tables to use as context. Each object in the list must contain 'projectId', 'datasetId', and 'tableId'. Example: '[{"projectId": "my-gcp-project", "datasetId": "my_dataset", "tableId": "my_table"}]'`)

	parameters := tools.Parameters{userQueryParameter, tableRefsParameter}

	mcpManifest := tools.McpManifest{
		Name:        cfg.Name,
		Description: cfg.Description,
		InputSchema: parameters.McpManifest(),
	}

	// finish tool setup
	t := Tool{
		Name:               cfg.Name,
		Kind:               kind,
		Parameters:         parameters,
		AuthRequired:       cfg.AuthRequired,
		Client:             s.BigQueryClient(),
		TokenSource:        s.BigQueryTokenSource(),
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
		MaxQueryResultRows: s.GetMaxQueryResultRows(),
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name               string           `yaml:"name"`
	Kind               string           `yaml:"kind"`
	AuthRequired       []string         `yaml:"authRequired"`
	Parameters         tools.Parameters `yaml:"parameters"`
	Client             *bigqueryapi.Client
	TokenSource        oauth2.TokenSource
	manifest           tools.Manifest
	mcpManifest        tools.McpManifest
	MaxQueryResultRows int
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues) (any, error) {
	// Get credentials for the API call
	if t.TokenSource == nil {
		return nil, fmt.Errorf("authentication error: found credentials but they are missing a valid token source")
	}

	token, err := t.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token from credentials: %w", err)
	}

	// Extract parameters from the map
	mapParams := params.AsMap()
	userQuery, _ := mapParams["user_query_with_context"].(string)

	finalQueryText := fmt.Sprintf("%s\n**User Query and Context:**\n%s", instructions, userQuery)

	tableRefsJSON, _ := mapParams["table_references"].(string)
	var tableRefs []BQTableReference
	if tableRefsJSON != "" {
		if err := json.Unmarshal([]byte(tableRefsJSON), &tableRefs); err != nil {
			return nil, fmt.Errorf("failed to parse 'table_references' JSON string: %w", err)
		}
	}

	// Construct URL, headers, and payload
	projectID := t.Client.Project()
	location := t.Client.Location
	if location == "" {
		location = "us"
	}
	caURL := fmt.Sprintf("https://geminidataanalytics.googleapis.com/v1alpha/projects/%s/locations/%s:chat", projectID, location)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token.AccessToken),
		"Content-Type":  "application/json",
	}

	payload := CAPayload{
		Project:  fmt.Sprintf("projects/%s", projectID),
		Messages: []Message{{UserMessage: UserMessage{Text: finalQueryText}}},
		InlineContext: InlineContext{
			DatasourceReferences: DatasourceReferences{
				BQ: BQDatasource{TableReferences: tableRefs},
			},
			Options: Options{Chart: ChartOptions{Image: ImageOptions{NoImage: map[string]any{}}}},
		},
	}

	// Call the streaming API
	response, err := getStream(caURL, payload, headers, t.MaxQueryResultRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from conversational analytics API: %w", err)
	}

	return response, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (tools.ParamValues, error) {
	return tools.ParseParams(t.Parameters, data, claims)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func getStream(url string, payload CAPayload, headers map[string]string, maxRows int) ([]map[string]any, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned non-200 status: %d %s", resp.StatusCode, string(body))
	}

	var acc strings.Builder
	var messages []map[string]any
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		if line == "[{" {
			acc.WriteString("{")
		} else if line == "}]" {
			acc.WriteString("}")
		} else if line == "," {
			continue
		} else {
			acc.WriteString(line)
		}

		jsonStr := acc.String()
		var dataJSON map[string]any
		if err := json.Unmarshal([]byte(jsonStr), &dataJSON); err != nil {
			continue
		}

		// Successfully parsed a JSON object, now handle it
		var newMessage map[string]any
		if sysMsg, ok := dataJSON["systemMessage"].(map[string]any); ok {
			if text, ok := sysMsg["text"].(map[string]any); ok {
				newMessage = handleTextResponse(text)
			} else if schema, ok := sysMsg["schema"].(map[string]any); ok {
				newMessage = handleSchemaResponse(schema)
			} else if data, ok := sysMsg["data"].(map[string]any); ok {
				newMessage = handleDataResponse(data, maxRows)
			}
		} else if errData, ok := dataJSON["error"].(map[string]any); ok {
			newMessage = handleError(errData)
		}

		messages = appendMessage(messages, newMessage)
		acc.Reset()
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return messages, nil
}

func formatBqTableRef(tableRef map[string]any) string {
	projectID, _ := tableRef["projectId"].(string)
	datasetID, _ := tableRef["datasetId"].(string)
	tableID, _ := tableRef["tableId"].(string)
	return fmt.Sprintf("%s.%s.%s", projectID, datasetID, tableID)
}

func formatSchemaAsDict(data map[string]any) map[string]any {
	headers := []string{"Column", "Type", "Description", "Mode"}
	fieldsVal, ok := data["fields"].([]any)
	if !ok {
		return map[string]any{"headers": headers, "rows": []any{}}
	}

	var rows [][]any
	for _, fieldVal := range fieldsVal {
		if field, ok := fieldVal.(map[string]any); ok {
			name, _ := field["name"].(string)
			typ, _ := field["type"].(string)
			desc, _ := field["description"].(string)
			mode, _ := field["mode"].(string)
			rows = append(rows, []any{name, typ, desc, mode})
		}
	}
	return map[string]any{"headers": headers, "rows": rows}
}

func formatDatasourceAsDict(datasource map[string]any) map[string]any {
	var sourceName string
	if ref, ok := datasource["bigqueryTableReference"].(map[string]any); ok {
		sourceName = formatBqTableRef(ref)
	}

	var schema map[string]any
	if s, ok := datasource["schema"].(map[string]any); ok {
		schema = formatSchemaAsDict(s)
	}

	return map[string]any{"source_name": sourceName, "schema": schema}
}

func handleTextResponse(resp map[string]any) map[string]any {
	var parts []string
	if partsVal, ok := resp["parts"].([]any); ok {
		for _, p := range partsVal {
			if partStr, ok := p.(string); ok {
				parts = append(parts, partStr)
			}
		}
	}
	return map[string]any{"Answer": strings.Join(parts, "")}
}

func handleSchemaResponse(resp map[string]any) map[string]any {
	if query, ok := resp["query"].(map[string]any); ok {
		if question, ok := query["question"].(string); ok {
			return map[string]any{"Question": question}
		}
	}
	if result, ok := resp["result"].(map[string]any); ok {
		var formattedSources []map[string]any
		if datasources, ok := result["datasources"].([]any); ok {
			for _, dsVal := range datasources {
				if ds, ok := dsVal.(map[string]any); ok {
					formattedSources = append(formattedSources, formatDatasourceAsDict(ds))
				}
			}
		}
		return map[string]any{"Schema Resolved": formattedSources}
	}
	return nil
}

func handleDataResponse(resp map[string]any, maxRows int) map[string]any {
	if query, ok := resp["query"].(map[string]any); ok {
		queryName, _ := query["name"].(string)
		question, _ := query["question"].(string)
		return map[string]any{
			"Retrieval Query": map[string]any{
				"Query Name": queryName,
				"Question":   question,
			},
		}
	}
	if sql, ok := resp["generatedSql"].(string); ok {
		return map[string]any{"SQL Generated": sql}
	}
	if result, ok := resp["result"].(map[string]any); ok {
		schema, _ := result["schema"].(map[string]any)
		var dataRows []any
		if data, ok := result["data"]; ok {
			dataRows, _ = data.([]any)
		}
		fieldsVal, _ := schema["fields"].([]any)

		var headers []string
		for _, f := range fieldsVal {
			if fieldMap, ok := f.(map[string]any); ok {
				if name, ok := fieldMap["name"].(string); ok {
					headers = append(headers, name)
				}
			}
		}

		totalRows := len(dataRows)
		var compactRows [][]any
		numRowsToDisplay := totalRows
		if numRowsToDisplay > maxRows {
			numRowsToDisplay = maxRows
		}

		for _, rowVal := range dataRows[:numRowsToDisplay] {
			if rowDict, ok := rowVal.(map[string]any); ok {
				var rowValues []any
				for _, header := range headers {
					rowValues = append(rowValues, rowDict[header])
				}
				compactRows = append(compactRows, rowValues)
			}
		}

		summary := fmt.Sprintf("Showing all %d rows.", totalRows)
		if totalRows > maxRows {
			summary = fmt.Sprintf("Showing the first %d of %d total rows.", numRowsToDisplay, totalRows)
		}

		return map[string]any{
			"Data Retrieved": map[string]any{
				"headers": headers,
				"rows":    compactRows,
				"summary": summary,
			},
		}
	}
	return nil
}

func handleError(resp map[string]any) map[string]any {
	code, _ := resp["code"].(float64) // JSON numbers are float64 by default
	message, _ := resp["message"].(string)
	return map[string]any{
		"Error": map[string]any{
			"Code":    int(code), // Convert to int for cleaner output
			"Message": message,
		},
	}
}

func appendMessage(messages []map[string]any, newMessage map[string]any) []map[string]any {
	if newMessage == nil {
		return messages
	}
	if len(messages) > 0 {
		if _, ok := messages[len(messages)-1]["Data Retrieved"]; ok {
			messages = messages[:len(messages)-1] // Replace last element
		}
	}
	return append(messages, newMessage)
}
