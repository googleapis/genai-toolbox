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

package bigqueryconversationalanalytics

import (
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
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"golang.org/x/oauth2"
)

const kind string = "bigquery-conversational-analytics"

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
	BigQueryTokenSourceWithScope(ctx context.Context, scope string) (oauth2.TokenSource, error)
	BigQueryProject() string
	BigQueryLocation() string
	GetMaxQueryResultRows() int
	UseClientAuthorization() bool
	IsDatasetAllowed(projectID, datasetID string) bool
	BigQueryAllowedDatasets() []string
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
	ClientIdEnum  string        `json:"clientIdEnum"`
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

	allowedDatasets := s.BigQueryAllowedDatasets()
	tableRefsDescription := `A JSON string of a list of BigQuery tables to use as context. Each object in the list must contain 'projectId', 'datasetId', and 'tableId'. Example: '[{"projectId": "my-gcp-project", "datasetId": "my_dataset", "tableId": "my_table"}]'.`
	if len(allowedDatasets) > 0 {
		datasetIDs := []string{}
		for _, ds := range allowedDatasets {
			datasetIDs = append(datasetIDs, fmt.Sprintf("`%s`", ds))
		}
		tableRefsDescription += fmt.Sprintf(" The tables must only be from datasets in the following list: %s.", strings.Join(datasetIDs, ", "))
	}
	userQueryParameter := parameters.NewStringParameter("user_query_with_context", "The user's question, potentially including conversation history and system instructions for context.")
	tableRefsParameter := parameters.NewStringParameter("table_references", tableRefsDescription)

	params := parameters.Parameters{userQueryParameter, tableRefsParameter}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// Get cloud-platform token source for Gemini Data Analytics API during initialization
	var bigQueryTokenSourceWithScope oauth2.TokenSource
	if !s.UseClientAuthorization() {
		ctx := context.Background()
		ts, err := s.BigQueryTokenSourceWithScope(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return nil, fmt.Errorf("failed to get cloud-platform token source: %w", err)
		}
		bigQueryTokenSourceWithScope = ts
	}

	// finish tool setup
	t := Tool{
		Config:             cfg,
		Project:            s.BigQueryProject(),
		Location:           s.BigQueryLocation(),
		Parameters:         params,
		Client:             s.BigQueryClient(),
		UseClientOAuth:     s.UseClientAuthorization(),
		TokenSource:        bigQueryTokenSourceWithScope,
		manifest:           tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:        mcpManifest,
		MaxQueryResultRows: s.GetMaxQueryResultRows(),
		IsDatasetAllowed:   s.IsDatasetAllowed,
		AllowedDatasets:    allowedDatasets,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	UseClientOAuth bool                  `yaml:"useClientOAuth"`
	Parameters     parameters.Parameters `yaml:"parameters"`

	Project            string
	Location           string
	Client             *bigqueryapi.Client
	TokenSource        oauth2.TokenSource
	manifest           tools.Manifest
	mcpManifest        tools.McpManifest
	MaxQueryResultRows int
	IsDatasetAllowed   func(projectID, datasetID string) bool
	AllowedDatasets    []string
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	var tokenStr string
	var err error

	// Get credentials for the API call
	if t.UseClientOAuth {
		// Use client-side access token
		if accessToken == "" {
			return nil, fmt.Errorf("tool is configured for client OAuth but no token was provided in the request header: %w", util.ErrUnauthorized)
		}
		tokenStr, err = accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
	} else {
		// Use cloud-platform token source for Gemini Data Analytics API
		if t.TokenSource == nil {
			return nil, fmt.Errorf("cloud-platform token source is missing")
		}
		token, err := t.TokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to get token from cloud-platform token source: %w", err)
		}
		tokenStr = token.AccessToken
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

	if len(t.AllowedDatasets) > 0 {
		for _, tableRef := range tableRefs {
			if !t.IsDatasetAllowed(tableRef.ProjectID, tableRef.DatasetID) {
				return nil, fmt.Errorf("access to dataset '%s.%s' (from table '%s') is not allowed", tableRef.ProjectID, tableRef.DatasetID, tableRef.TableID)
			}
		}
	}

	// Construct URL, headers, and payload
	projectID := t.Project
	location := t.Location
	if location == "" {
		location = "us"
	}
	caURL := fmt.Sprintf("https://geminidataanalytics.googleapis.com/v1alpha/projects/%s/locations/%s:chat", projectID, location)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", tokenStr),
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
		ClientIdEnum: "GENAI_TOOLBOX",
	}

	// Call the streaming API
	response, err := getStream(caURL, payload, headers, t.MaxQueryResultRows)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from conversational analytics API: %w", err)
	}

	return response, nil
}

func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.Parameters, data, claims)
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

func (t Tool) RequiresClientAuthorization() bool {
	return t.UseClientOAuth
}

// StreamMessage represents a single message object from the streaming API response.
type StreamMessage struct {
	SystemMessage *SystemMessage `json:"systemMessage,omitempty"`
	Error         *ErrorResponse `json:"error,omitempty"`
}

// SystemMessage contains different types of system-generated content.
type SystemMessage struct {
	Text   *TextResponse   `json:"text,omitempty"`
	Schema *SchemaResponse `json:"schema,omitempty"`
	Data   *DataResponse   `json:"data,omitempty"`
}

// TextResponse contains textual parts of a message.
type TextResponse struct {
	Parts []string `json:"parts"`
}

// SchemaResponse contains schema-related information.
type SchemaResponse struct {
	Query  *SchemaQuery  `json:"query,omitempty"`
	Result *SchemaResult `json:"result,omitempty"`
}

// SchemaQuery holds the question that prompted a schema lookup.
type SchemaQuery struct {
	Question string `json:"question"`
}

// SchemaResult contains the datasources with their schemas.
type SchemaResult struct {
	Datasources []Datasource `json:"datasources"`
}

// Datasource represents a data source with its reference and schema.
type Datasource struct {
	BigQueryTableReference *BQTableReference `json:"bigqueryTableReference,omitempty"`
	Schema                 *BQSchema         `json:"schema,omitempty"`
}

// BQSchema defines the structure of a BigQuery table.
type BQSchema struct {
	Fields []BQField `json:"fields"`
}

// BQField describes a single column in a BigQuery table.
type BQField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Mode        string `json:"mode"`
}

// DataResponse contains data-related information, like queries and results.
type DataResponse struct {
	Query        *DataQuery  `json:"query,omitempty"`
	GeneratedSQL string      `json:"generatedSql,omitempty"`
	Result       *DataResult `json:"result,omitempty"`
}

// DataQuery holds information about a data retrieval query.
type DataQuery struct {
	Name     string `json:"name"`
	Question string `json:"question"`
}

// DataResult contains the schema and rows of a query result.
type DataResult struct {
	Schema BQSchema         `json:"schema"`
	Data   []map[string]any `json:"data"`
}

// ErrorResponse represents an error message from the API.
type ErrorResponse struct {
	Code    float64 `json:"code"` // JSON numbers are float64 by default
	Message string  `json:"message"`
}

func getStream(url string, payload CAPayload, headers map[string]string, maxRows int) (string, error) {
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

	var messages []map[string]any
	decoder := json.NewDecoder(resp.Body)

	// The response is a JSON array, so we read the opening bracket.
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return "", nil // Empty response is valid
		}
		return "", fmt.Errorf("error reading start of json array: %w", err)
	}

	for decoder.More() {
		var msg StreamMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error decoding stream message: %w", err)
		}

		var newMessage map[string]any
		if msg.SystemMessage != nil {
			if msg.SystemMessage.Text != nil {
				newMessage = handleTextResponse(msg.SystemMessage.Text)
			} else if msg.SystemMessage.Schema != nil {
				newMessage = handleSchemaResponse(msg.SystemMessage.Schema)
			} else if msg.SystemMessage.Data != nil {
				newMessage = handleDataResponse(msg.SystemMessage.Data, maxRows)
			}
		} else if msg.Error != nil {
			newMessage = handleError(msg.Error)
		}
		messages = appendMessage(messages, newMessage)
	}

	var acc strings.Builder
	for i, msg := range messages {
		jsonBytes, err := json.MarshalIndent(msg, "", "  ")
		if err != nil {
			return "", fmt.Errorf("error marshalling message: %w", err)
		}
		acc.Write(jsonBytes)
		if i < len(messages)-1 {
			acc.WriteString("\n")
		}
	}

	return acc.String(), nil
}

func formatBqTableRef(tableRef *BQTableReference) string {
	return fmt.Sprintf("%s.%s.%s", tableRef.ProjectID, tableRef.DatasetID, tableRef.TableID)
}

func formatSchemaAsDict(data *BQSchema) map[string]any {
	headers := []string{"Column", "Type", "Description", "Mode"}
	if data == nil {
		return map[string]any{"headers": headers, "rows": []any{}}
	}

	var rows [][]any
	for _, field := range data.Fields {
		rows = append(rows, []any{field.Name, field.Type, field.Description, field.Mode})
	}
	return map[string]any{"headers": headers, "rows": rows}
}

func formatDatasourceAsDict(datasource *Datasource) map[string]any {
	var sourceName string
	if datasource.BigQueryTableReference != nil {
		sourceName = formatBqTableRef(datasource.BigQueryTableReference)
	}

	var schema map[string]any
	if datasource.Schema != nil {
		schema = formatSchemaAsDict(datasource.Schema)
	}

	return map[string]any{"source_name": sourceName, "schema": schema}
}

func handleTextResponse(resp *TextResponse) map[string]any {
	return map[string]any{"Answer": strings.Join(resp.Parts, "")}
}

func handleSchemaResponse(resp *SchemaResponse) map[string]any {
	if resp.Query != nil {
		return map[string]any{"Question": resp.Query.Question}
	}
	if resp.Result != nil {
		var formattedSources []map[string]any
		for _, ds := range resp.Result.Datasources {
			formattedSources = append(formattedSources, formatDatasourceAsDict(&ds))
		}
		return map[string]any{"Schema Resolved": formattedSources}
	}
	return nil
}

func handleDataResponse(resp *DataResponse, maxRows int) map[string]any {
	if resp.Query != nil {
		return map[string]any{
			"Retrieval Query": map[string]any{
				"Query Name": resp.Query.Name,
				"Question":   resp.Query.Question,
			},
		}
	}
	if resp.GeneratedSQL != "" {
		return map[string]any{"SQL Generated": resp.GeneratedSQL}
	}
	if resp.Result != nil {
		var headers []string
		for _, f := range resp.Result.Schema.Fields {
			headers = append(headers, f.Name)
		}

		totalRows := len(resp.Result.Data)
		var compactRows [][]any
		numRowsToDisplay := totalRows
		if numRowsToDisplay > maxRows {
			numRowsToDisplay = maxRows
		}

		for _, rowVal := range resp.Result.Data[:numRowsToDisplay] {
			var rowValues []any
			for _, header := range headers {
				rowValues = append(rowValues, rowVal[header])
			}
			compactRows = append(compactRows, rowValues)
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

func handleError(resp *ErrorResponse) map[string]any {
	return map[string]any{
		"Error": map[string]any{
			"Code":    int(resp.Code),
			"Message": resp.Message,
		},
	}
}

func appendMessage(messages []map[string]any, newMessage map[string]any) []map[string]any {
	if newMessage == nil {
		return messages
	}
	if len(messages) > 0 {
		if _, ok := messages[len(messages)-1]["Data Retrieved"]; ok {
			messages = messages[:len(messages)-1]
		}
	}
	return append(messages, newMessage)
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
