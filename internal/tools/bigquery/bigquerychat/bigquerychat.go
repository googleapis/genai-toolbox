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

package bigquerychat

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

const kind string = "bigquery-chat"

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

type ChatPayload struct {
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
	chatURL := fmt.Sprintf("https://geminidataanalytics.googleapis.com/v1alpha/projects/%s/locations/%s:chat", projectID, location)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", token.AccessToken),
		"Content-Type":  "application/json",
	}

	payload := ChatPayload{
		Project:  fmt.Sprintf("projects/%s", projectID),
		Messages: []Message{{UserMessage: UserMessage{Text: userQuery}}},
		InlineContext: InlineContext{
			DatasourceReferences: DatasourceReferences{
				BQ: BQDatasource{TableReferences: tableRefs},
			},
			Options: Options{Chart: ChartOptions{Image: ImageOptions{NoImage: map[string]any{}}}},
		},
	}

	// Call the streaming API
	response, err := getStream(chatURL, payload, headers, t.MaxQueryResultRows)

	if err != nil {
		return nil, fmt.Errorf("failed to get response from chat API: %w", err)
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

func getProperty(data map[string]any, fieldName string, defaultValue string) string {
	if val, ok := data[fieldName]; ok {
		if val == nil {
			return defaultValue
		}
		return fmt.Sprintf("%v", val)
	}
	return defaultValue
}

func getStream(url string, payload ChatPayload, headers map[string]string, maxRows int) (string, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned non-200 status: %d %s", resp.StatusCode, string(body))
	}

	var acc strings.Builder
	var messages []string
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
		if sysMsg, ok := dataJSON["systemMessage"].(map[string]any); ok {
			if text, ok := sysMsg["text"].(map[string]any); ok {
				messages = appendMessage(messages, handleTextResponse(text))
			} else if schema, ok := sysMsg["schema"].(map[string]any); ok {
				messages = appendMessage(messages, handleSchemaResponse(schema))
			} else if data, ok := sysMsg["data"].(map[string]any); ok {
				messages = appendMessage(messages, handleDataResponse(data, maxRows))
			}
		} else if errData, ok := dataJSON["error"].(map[string]any); ok {
			messages = appendMessage(messages, handleError(errData))
		}

		acc.Reset()
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading stream: %w", err)
	}

	return strings.Join(messages, "\n\n"), nil
}

func formatSectionTitle(text string) string {
	return fmt.Sprintf("## %s", text)
}

func formatBqTableRef(tableRef map[string]any) string {
	return fmt.Sprintf("%s.%s.%s",
		getProperty(tableRef, "projectId", ""),
		getProperty(tableRef, "datasetId", ""),
		getProperty(tableRef, "tableId", ""),
	)
}

func formatSchemaAsMarkdown(data map[string]any) string {
	fieldsVal, ok := data["fields"].([]any)
	if !ok || len(fieldsVal) == 0 {
		return "No schema fields found."
	}

	headers := []string{"Column", "Type", "Description", "Mode"}
	var table strings.Builder
	table.WriteString(fmt.Sprintf("| %s |\n", strings.Join(headers, " | ")))
	table.WriteString(fmt.Sprintf("|%s|\n", strings.Repeat("---|", len(headers))))

	for _, fieldVal := range fieldsVal {
		if field, ok := fieldVal.(map[string]any); ok {
			row := []string{
				getProperty(field, "name", ""),
				getProperty(field, "type", ""),
				getProperty(field, "description", "-"),
				getProperty(field, "mode", ""),
			}
			table.WriteString(fmt.Sprintf("| %s |\n", strings.Join(row, " | ")))
		}
	}
	return table.String()
}

func formatDatasourceAsMarkdown(datasource map[string]any) string {
	var sourceName string
	if ref, ok := datasource["bigqueryTableReference"].(map[string]any); ok {
		sourceName = formatBqTableRef(ref)
	}

	var schema map[string]any
	if s, ok := datasource["schema"].(map[string]any); ok {
		schema = s
	}

	schemaMarkdown := formatSchemaAsMarkdown(schema)
	return fmt.Sprintf("**Source:** `%s`\n%s", sourceName, schemaMarkdown)
}

func handleTextResponse(resp map[string]any) string {
	if partsVal, ok := resp["parts"].([]any); ok {
		var parts []string
		for _, p := range partsVal {
			if partStr, ok := p.(string); ok {
				parts = append(parts, partStr)
			}
		}
		return "Answer: " + strings.Join(parts, "")
	}
	return "Answer: Not provided."
}

func handleSchemaResponse(resp map[string]any) string {
	if query, ok := resp["query"].(map[string]any); ok {
		return getProperty(query, "question", "")
	}
	if result, ok := resp["result"].(map[string]any); ok {
		title := formatSectionTitle("Schema Resolved")
		var formattedSources []string
		if datasources, ok := result["datasources"].([]any); ok {
			for _, dsVal := range datasources {
				if ds, ok := dsVal.(map[string]any); ok {
					formattedSources = append(formattedSources, formatDatasourceAsMarkdown(ds))
				}
			}
		}
		return fmt.Sprintf("%s\nData sources:\n%s", title, strings.Join(formattedSources, "\n\n"))
	}
	return ""
}

func handleDataResponse(resp map[string]any, maxRows int) string {
	if query, ok := resp["query"].(map[string]any); ok {
		title := formatSectionTitle("Retrieval Query")
		return fmt.Sprintf("%s\n**Query Name:** %s\n**Question:** %s", title, getProperty(query, "name", "N/A"), getProperty(query, "question", "N/A"))
	}
	if sql, ok := resp["generatedSql"].(string); ok {
		title := formatSectionTitle("SQL Generated")
		return fmt.Sprintf("%s\n```sql\n%s\n```", title, sql)
	}
	if result, ok := resp["result"].(map[string]any); ok {
		title := formatSectionTitle("Data Retrieved")
		schema, _ := result["schema"].(map[string]any)
		dataRows, _ := result["data"].([]any)
		fieldsVal, _ := schema["fields"].([]any)

		var fields []string
		for _, f := range fieldsVal {
			if fieldMap, ok := f.(map[string]any); ok {
				fields = append(fields, getProperty(fieldMap, "name", ""))
			}
		}

		totalRows := len(dataRows)
		headerLine := fmt.Sprintf("| %s |", strings.Join(fields, " | "))
		separatorLine := fmt.Sprintf("|%s|", strings.Repeat("---|", len(fields)))
		tableLines := []string{headerLine, separatorLine}

		numRowsToDisplay := totalRows
		if numRowsToDisplay > maxRows {
			numRowsToDisplay = maxRows
		}

		for _, rowVal := range dataRows[:numRowsToDisplay] {
			if rowDict, ok := rowVal.(map[string]any); ok {
				var rowValues []string
				for _, field := range fields {
					rowValues = append(rowValues, fmt.Sprintf("%v", rowDict[field]))
				}
				tableLines = append(tableLines, fmt.Sprintf("| %s |", strings.Join(rowValues, " | ")))
			}
		}
		tableMarkdown := strings.Join(tableLines, "\n")

		if totalRows > maxRows {
			tableMarkdown += fmt.Sprintf("\n\n... *and %d more rows*.", totalRows-maxRows)
		}
		return fmt.Sprintf("%s\n%s", title, tableMarkdown)
	}
	return ""
}

func handleError(resp map[string]any) string {
	title := formatSectionTitle("Error")
	code := getProperty(resp, "code", "N/A")
	message := getProperty(resp, "message", "No message provided.")
	return fmt.Sprintf("%s\n**Code:** %s\n**Message:** %s", title, code, message)
}

// Go version of _append_message, returns a new slice
func appendMessage(messages []string, newMessage string) []string {
	if newMessage == "" {
		return messages
	}
	if len(messages) > 0 && strings.HasPrefix(messages[len(messages)-1], "## Data Retrieved") {
		// Replace the last element
		messages[len(messages)-1] = newMessage
		return messages
	}
	// Append
	return append(messages, newMessage)
}
