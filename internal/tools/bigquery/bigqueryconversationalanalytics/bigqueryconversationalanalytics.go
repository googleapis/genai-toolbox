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
	"time"

	bigqueryapi "cloud.google.com/go/bigquery"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"golang.org/x/oauth2"
)

const resourceType string = "bigquery-conversational-analytics"

const gdaURLFormat = "https://geminidataanalytics.googleapis.com/v1beta/projects/%s/locations/%s:chat"

const instructions = `**INSTRUCTIONS - FOLLOW THESE RULES:**
1. **CONTENT:** Your answer should present the supporting data and then provide a conclusion based on that data.
2. **OUTPUT FORMAT:** Your entire response MUST be in plain text format ONLY.
3. **NO CHARTS:** You are STRICTLY FORBIDDEN from generating any charts, graphs, images, or any other form of visualization.`

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
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
	BigQueryTokenSourceWithScope(ctx context.Context, scopes []string) (oauth2.TokenSource, error)
	BigQueryProject() string
	BigQueryLocation() string
	GetMaxQueryResultRows() int
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
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

type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Type         string   `yaml:"type" validate:"required"`
	Source       string   `yaml:"source" validate:"required"`
	Description  string   `yaml:"description" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
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
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", resourceType, cfg.Source)
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

	// finish tool setup
	t := Tool{
		Config:      cfg,
		Parameters:  params,
		manifest:    tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	Parameters  parameters.Parameters `yaml:"parameters"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	var tokenStr string

	// Get credentials for the API call
	if source.UseClientAuthorization() {
		// Use client-side access token
		if accessToken == "" {
			return nil, util.NewClientServerError("tool is configured for client OAuth but no token was provided in the request header", http.StatusUnauthorized, nil)
		}
		tokenStr, err = accessToken.ParseBearerToken()
		if err != nil {
			return nil, util.NewClientServerError("error parsing access token", http.StatusUnauthorized, err)
		}
	} else {
		// Get a token source for the Gemini Data Analytics API.
		tokenSource, err := source.BigQueryTokenSourceWithScope(ctx, nil)
		if err != nil {
			return nil, util.NewClientServerError("failed to get token source", http.StatusInternalServerError, err)
		}

		// Use cloud-platform token source for Gemini Data Analytics API
		if tokenSource == nil {
			return nil, util.NewClientServerError("cloud-platform token source is missing", http.StatusInternalServerError, nil)
		}
		token, err := tokenSource.Token()
		if err != nil {
			return nil, util.NewClientServerError("failed to get token from cloud-platform token source", http.StatusInternalServerError, err)
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
			return nil, util.NewAgentError("failed to parse 'table_references' JSON string", err)
		}
	}

	if len(source.BigQueryAllowedDatasets()) > 0 {
		for _, tableRef := range tableRefs {
			if !source.IsDatasetAllowed(tableRef.ProjectID, tableRef.DatasetID) {
				return nil, util.NewAgentError(fmt.Sprintf("access to dataset '%s.%s' (from table '%s') is not allowed", tableRef.ProjectID, tableRef.DatasetID, tableRef.TableID), nil)
			}
		}
	}

	// Construct URL, headers, and payload
	projectID := source.BigQueryProject()
	location := source.BigQueryLocation()
	if location == "" {
		location = "us"
	}
	caURL := fmt.Sprintf(gdaURLFormat, projectID, location)

	headers := map[string]string{
		source.GetAuthTokenHeaderName(): fmt.Sprintf("Bearer %s", tokenStr),
		"Content-Type":                  "application/json",
		"X-Goog-API-Client":             util.GDAClientID,
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
		ClientIdEnum: util.GDAClientID,
	}

	// Call the streaming API
	response, err := getStream(caURL, payload, headers, source.GetMaxQueryResultRows())
	if err != nil {
		// getStream wraps network errors or non-200 responses
		return nil, util.NewClientServerError("failed to get response from conversational analytics API", http.StatusInternalServerError, err)
	}

	return response, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
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

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
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

	client := &http.Client{Timeout: 330 * time.Second}
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
	dataMsgIdx := -1

	// The response is a JSON array, so we read the opening bracket.
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return "", nil // Empty response is valid
		}
		return "", fmt.Errorf("error reading start of json array: %w", err)
	}

	for decoder.More() {
		var rawMsg json.RawMessage
		if err := decoder.Decode(&rawMsg); err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("error decoding raw message: %w", err)
		}

		var msg map[string]any
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			return "", fmt.Errorf("error unmarshaling raw message: %w", err)
		}

		var processedMsg map[string]any
		if dataResult := extractDataResult(msg); dataResult != nil {
			// 1. If it's a data result, format it.
			processedMsg = formatDataRetrieved(dataResult, maxRows)
			if dataMsgIdx >= 0 {
				// Replace previous data with a placeholder. Intermediate data results in a
				// stream are redundant and consume unnecessary tokens.
				messages[dataMsgIdx] = map[string]any{"Data Retrieved": "Intermediate result omitted"}
			}
			dataMsgIdx = len(messages)
		} else if sm, ok := msg["systemMessage"].(map[string]any); ok {
			// 2. If it's a system message, unwrap it.
			processedMsg = sm
		} else {
			// 3. Otherwise (e.g. error), pass it through raw.
			processedMsg = msg
		}

		if processedMsg != nil {
			messages = append(messages, processedMsg)
		}
	}

	var acc strings.Builder
	for i, msg := range messages {
		jsonBytes, err := json.Marshal(msg)
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

// extractDataResult attempts to find the result.data deep inside the generic map.
func extractDataResult(msg map[string]any) map[string]any {
	sm, ok := msg["systemMessage"].(map[string]any)
	if !ok {
		return nil
	}
	data, ok := sm["data"].(map[string]any)
	if !ok {
		return nil
	}
	result, ok := data["result"].(map[string]any)
	if !ok {
		return nil
	}
	if _, hasData := result["data"].([]any); hasData {
		return result
	}
	return nil
}

// formatDataRetrieved transforms the raw result map into the simplified Toolbox format.
func formatDataRetrieved(result map[string]any, maxRows int) map[string]any {
	rawData, _ := result["data"].([]any)

	var fields []any
	if schema, ok := result["schema"].(map[string]any); ok {
		if f, ok := schema["fields"].([]any); ok {
			fields = f
		}
	}

	var headers []string
	for _, f := range fields {
		if fm, ok := f.(map[string]any); ok {
			if name, ok := fm["name"].(string); ok {
				headers = append(headers, name)
			}
		}
	}

	totalRows := len(rawData)
	numToDisplay := totalRows
	if numToDisplay > maxRows {
		numToDisplay = maxRows
	}

	var rows [][]any
	for _, r := range rawData[:numToDisplay] {
		if rm, ok := r.(map[string]any); ok {
			var row []any
			for _, h := range headers {
				row = append(row, rm[h])
			}
			rows = append(rows, row)
		}
	}

	summary := fmt.Sprintf("Showing all %d rows.", totalRows)
	if totalRows > maxRows {
		summary = fmt.Sprintf("Showing the first %d of %d total rows.", numToDisplay, totalRows)
	}

	return map[string]any{
		"Data Retrieved": map[string]any{
			"headers": headers,
			"rows":    rows,
			"summary": summary,
		},
	}
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return "", err
	}
	return source.GetAuthTokenHeaderName(), nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
