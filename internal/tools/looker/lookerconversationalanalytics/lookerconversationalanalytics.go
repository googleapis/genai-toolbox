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

package lookerconversationalanalytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	lookerds "github.com/googleapis/genai-toolbox/internal/sources/looker"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"github.com/looker-open-source/sdk-codegen/go/rtl"
	"golang.org/x/oauth2"
)

const kind string = "looker-conversational-analytics"

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
	GetApiSettings() *rtl.ApiSettings
	GoogleCloudTokenSourceWithScope(ctx context.Context, scope string) (oauth2.TokenSource, error)
	GoogleCloudProject() string
	GoogleCloudLocation() string
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
}

// Structs for building the JSON payload
type UserMessage struct {
	Text string `json:"text"`
}
type Message struct {
	UserMessage UserMessage `json:"userMessage"`
}
type LookerExploreReference struct {
	LookerInstanceUri string `json:"lookerInstanceUri"`
	LookmlModel       string `json:"lookmlModel"`
	Explore           string `json:"explore"`
}
type LookerExploreReferences struct {
	ExploreReferences []LookerExploreReference `json:"exploreReferences"`
	Credentials       Credentials              `json:"credentials,omitzero"`
}
type SecretBased struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}
type TokenBased struct {
	AccessToken string `json:"accessToken"`
}
type OAuthCredentials struct {
	Secret SecretBased `json:"secret,omitzero"`
	Token  TokenBased  `json:"token,omitzero"`
}
type Credentials struct {
	OAuth OAuthCredentials `json:"oauth"`
}
type DatasourceReferences struct {
	Looker LookerExploreReferences `json:"looker"`
}
type ImageOptions struct {
	NoImage map[string]any `json:"noImage"`
}
type ChartOptions struct {
	Image ImageOptions `json:"image"`
}
type Python struct {
	Enabled bool `json:"enabled"`
}
type AnalysisOptions struct {
	Python Python `json:"python"`
}
type ConversationOptions struct {
	Chart    ChartOptions    `json:"chart,omitzero"`
	Analysis AnalysisOptions `json:"analysis,omitzero"`
}
type InlineContext struct {
	SystemInstruction    string               `json:"systemInstruction"`
	DatasourceReferences DatasourceReferences `json:"datasourceReferences"`
	Options              ConversationOptions  `json:"options"`
}
type CAPayload struct {
	Messages      []Message     `json:"messages"`
	InlineContext InlineContext `json:"inlineContext"`
	ClientIdEnum  string        `json:"clientIdEnum"`
}

// validate compatible sources are still compatible
var _ compatibleSource = &lookerds.Source{}

var compatibleSources = [...]string{lookerds.SourceKind}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Kind         string                 `yaml:"kind" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description" validate:"required"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`
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

	if s.GoogleCloudProject() == "" {
		return nil, fmt.Errorf("project must be defined for source to use with %q tool", kind)
	}

	userQueryParameter := parameters.NewStringParameter("user_query_with_context", "The user's question, potentially including conversation history and system instructions for context.")

	exploreRefsDescription := `An Array of at least one and up to 5 explore references like [{'model': 'MODEL_NAME', 'explore': 'EXPLORE_NAME'}]`
	exploreRefsParameter := parameters.NewArrayParameter(
		"explore_references",
		exploreRefsDescription,
		parameters.NewMapParameter(
			"explore_reference",
			"An explore reference like {'model': 'MODEL_NAME', 'explore': 'EXPLORE_NAME'}",
			"",
		),
	)

	params := parameters.Parameters{userQueryParameter, exploreRefsParameter}

	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, cfg.Annotations)

	// Get cloud-platform token source for Gemini Data Analytics API during initialization
	ctx := context.Background()
	ts, err := s.GoogleCloudTokenSourceWithScope(ctx, "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to get cloud-platform token source: %w", err)
	}

	// finish tool setup
	t := Tool{
		Config:              cfg,
		ApiSettings:         s.GetApiSettings(),
		Project:             s.GoogleCloudProject(),
		Location:            s.GoogleCloudLocation(),
		Parameters:          params,
		UseClientOAuth:      s.UseClientAuthorization(),
		AuthTokenHeaderName: s.GetAuthTokenHeaderName(),
		TokenSource:         ts,
		manifest:            tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:         mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	ApiSettings         *rtl.ApiSettings
	UseClientOAuth      bool `yaml:"useClientOAuth"`
	AuthTokenHeaderName string
	Parameters          parameters.Parameters `yaml:"parameters"`
	Project             string
	Location            string
	TokenSource         oauth2.TokenSource
	manifest            tools.Manifest
	mcpManifest         tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	var tokenStr string
	var err error

	// Get credentials for the API call
	// Use cloud-platform token source for Gemini Data Analytics API
	if t.TokenSource == nil {
		return nil, fmt.Errorf("cloud-platform token source is missing")
	}
	token, err := t.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token from cloud-platform token source: %w", err)
	}
	tokenStr = token.AccessToken

	// Extract parameters from the map
	mapParams := params.AsMap()
	userQuery, _ := mapParams["user_query_with_context"].(string)
	exploreReferences, _ := mapParams["explore_references"].([]any)

	ler := make([]LookerExploreReference, 0)
	for _, er := range exploreReferences {
		ler = append(ler, LookerExploreReference{
			LookerInstanceUri: t.ApiSettings.BaseUrl,
			LookmlModel:       er.(map[string]any)["model"].(string),
			Explore:           er.(map[string]any)["explore"].(string),
		})
	}
	oauth_creds := OAuthCredentials{}
	if t.UseClientOAuth {
		oauth_creds.Token = TokenBased{AccessToken: string(accessToken)}
	} else {
		oauth_creds.Secret = SecretBased{ClientId: t.ApiSettings.ClientId, ClientSecret: t.ApiSettings.ClientSecret}
	}

	lers := LookerExploreReferences{
		ExploreReferences: ler,
		Credentials: Credentials{
			OAuth: oauth_creds,
		},
	}

	// Construct URL, headers, and payload
	projectID := t.Project
	location := t.Location
	caURL := fmt.Sprintf("https://geminidataanalytics.googleapis.com/v1beta/projects/%s/locations/%s:chat", url.PathEscape(projectID), url.PathEscape(location))

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", tokenStr),
		"Content-Type":  "application/json",
	}

	payload := CAPayload{
		Messages: []Message{{UserMessage: UserMessage{Text: userQuery}}},
		InlineContext: InlineContext{
			SystemInstruction: instructions,
			DatasourceReferences: DatasourceReferences{
				Looker: lers,
			},
			Options: ConversationOptions{Chart: ChartOptions{Image: ImageOptions{NoImage: map[string]any{}}}},
		},
		ClientIdEnum: "GENAI_TOOLBOX",
	}

	// Call the streaming API
	response, err := getStream(ctx, caURL, payload, headers)
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
}

// SystemMessage contains different types of system-generated content.
type SystemMessage struct {
	Text     *TextMessage     `json:"text,omitempty"`
	Schema   *SchemaMessage   `json:"schema,omitempty"`
	Data     *DataMessage     `json:"data,omitempty"`
	Analysis *AnalysisMessage `json:"analysis,omitempty"`
	Error    *ErrorMessage    `json:"error,omitempty"`
}

// TextMessage contains textual parts of a message.
type TextMessage struct {
	Parts []string `json:"parts"`
}

// SchemaMessage contains schema-related information.
type SchemaMessage struct {
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
	LookerExploreReference LookerExploreReference `json:"lookerExploreReference"`
}

// DataMessage contains data-related information, like queries and results.
type DataMessage struct {
	GeneratedLookerQuery *LookerQuery `json:"generatedLookerQuery,omitempty"`
	Result               *DataResult  `json:"result,omitempty"`
}

type LookerQuery struct {
	Model   string   `json:"model"`
	Explore string   `json:"explore"`
	Fields  []string `json:"fields"`
	Filters []Filter `json:"filters,omitempty"`
	Sorts   []string `json:"sorts,omitempty"`
	Limit   string   `json:"limit,omitempty"`
}

type Filter struct {
	Field string `json:"field,omitempty"`
	Value string `json:"value,omitempty"`
}

// DataResult contains the schema and rows of a query result.
type DataResult struct {
	Data []map[string]any `json:"data"`
}

type AnalysisQuery struct {
	Question        string   `json:"question,omitempty"`
	DataResultNames []string `json:"dataResultNames,omitempty"`
}
type AnalysisEvent struct {
	PlannerReasoning      string `json:"plannerReasoning,omitempty"`
	CoderInstructions     string `json:"coderInstructions,omitempty"`
	Code                  string `json:"code,omitempty"`
	ExecutionOutput       string `json:"executionOutput,omitempty"`
	ExecutionError        string `json:"executionError,omitempty"`
	ResultVegaChartJson   string `json:"resultVegaChartJson,omitempty"`
	ResultNaturalLanguage string `json:"resultNaturalLanguage,omitempty"`
	ResultCsvData         string `json:"resultCsvData,omitempty"`
	ResultReferenceData   string `json:"resultReferenceData,omitempty"`
	Error                 string `json:"error,omitempty"`
}
type AnalysisMessage struct {
	Query         AnalysisQuery `json:"query,omitempty"`
	ProgressEvent AnalysisEvent `json:"progressEvent,omitempty"`
}

// ErrorResponse represents an error message from the API.
type ErrorMessage struct {
	Text string `json:"text"`
}

func getStream(ctx context.Context, url string, payload CAPayload, headers map[string]string) ([]map[string]any, error) {
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

	var messages []map[string]any
	decoder := json.NewDecoder(resp.Body)

	// The response is a JSON array, so we read the opening bracket.
	if _, err := decoder.Token(); err != nil {
		if err == io.EOF {
			return nil, nil // Empty response is valid
		}
		return nil, fmt.Errorf("error reading start of json array: %w", err)
	}

	for decoder.More() {
		var msg StreamMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error decoding stream message: %w", err)
		}

		var newMessage map[string]any
		if msg.SystemMessage != nil {
			if msg.SystemMessage.Text != nil {
				newMessage = handleTextResponse(ctx, msg.SystemMessage.Text)
			} else if msg.SystemMessage.Schema != nil {
				newMessage = handleSchemaResponse(ctx, msg.SystemMessage.Schema)
			} else if msg.SystemMessage.Data != nil {
				newMessage = handleDataResponse(ctx, msg.SystemMessage.Data)
			} else if msg.SystemMessage.Analysis != nil {
				newMessage = handleAnalysisResponse(ctx, msg.SystemMessage.Analysis)
			} else if msg.SystemMessage.Error != nil {
				newMessage = handleError(ctx, msg.SystemMessage.Error)
			}
			messages = appendMessage(messages, newMessage)
		}
	}

	return messages, nil
}

func formatDatasourceAsDict(ctx context.Context, datasource *Datasource) map[string]any {
	logger, _ := util.LoggerFromContext(ctx)
	logger.DebugContext(ctx, "Datasource %s", *datasource)
	ds := make(map[string]any)
	ds["model"] = datasource.LookerExploreReference.LookmlModel
	ds["explore"] = datasource.LookerExploreReference.Explore
	ds["lookerInstanceUri"] = datasource.LookerExploreReference.LookerInstanceUri
	return map[string]any{"Datasource": ds}
}

func handleAnalysisResponse(ctx context.Context, resp *AnalysisMessage) map[string]any {
	logger, _ := util.LoggerFromContext(ctx)
	jsonData, err := json.Marshal(*resp)
	if err != nil {
		logger.ErrorContext(ctx, "error marshaling struct: %w", err)
		return map[string]any{"Analysis": "error"}
	}
	return map[string]any{"Analysis": jsonData}
}

func handleTextResponse(ctx context.Context, resp *TextMessage) map[string]any {
	logger, _ := util.LoggerFromContext(ctx)
	logger.DebugContext(ctx, "Text Response: %s", strings.Join(resp.Parts, ""))
	return map[string]any{"Answer": strings.Join(resp.Parts, "")}
}

func handleSchemaResponse(ctx context.Context, resp *SchemaMessage) map[string]any {
	if resp.Query != nil {
		return map[string]any{"Question": resp.Query.Question}
	}
	if resp.Result != nil {
		var formattedSources []map[string]any
		for _, ds := range resp.Result.Datasources {
			formattedSources = append(formattedSources, formatDatasourceAsDict(ctx, &ds))
		}
		return map[string]any{"Schema Resolved": formattedSources}
	}
	return nil
}

func handleDataResponse(ctx context.Context, resp *DataMessage) map[string]any {
	if resp.GeneratedLookerQuery != nil {
		logger, _ := util.LoggerFromContext(ctx)
		jsonData, err := json.Marshal(resp.GeneratedLookerQuery)
		if err != nil {
			logger.ErrorContext(ctx, "error marshaling struct: %w", err)
			return map[string]any{"Retrieval Query": "error"}
		}
		return map[string]any{
			"Retrieval Query": jsonData,
		}
	}
	if resp.Result != nil {

		return map[string]any{
			"Data Retrieved": resp.Result.Data,
		}
	}
	return nil
}

func handleError(ctx context.Context, resp *ErrorMessage) map[string]any {
	logger, _ := util.LoggerFromContext(ctx)
	logger.DebugContext(ctx, "Error Response: %s", resp.Text)
	return map[string]any{
		"Error": map[string]any{
			"Message": resp.Text,
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
	return t.AuthTokenHeaderName
}
