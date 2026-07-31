// Copyright 2026 Google LLC
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
package lookercreatemergequery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/looker/lookercommon"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"

	"github.com/looker-open-source/sdk-codegen/go/rtl"
	v4 "github.com/looker-open-source/sdk-codegen/go/sdk/v4"
)

const resourceType string = "looker-create-merge-query"

// sourceQueryKeys are the keys accepted inside a single `source_queries`
// entry. Anything else is rejected so that a misspelled key surfaces as an
// error instead of being silently dropped from the query.
var sourceQueryKeys = []string{
	"name",
	"model",
	"explore",
	"fields",
	"filters",
	"pivots",
	"sorts",
	"limit",
	"filter_expression",
	"dynamic_fields",
	"merge_fields",
	"tz",
}

func init() {
	if !tools.Register(resourceType, newConfig) {
		panic(fmt.Sprintf("tool type %q already registered", resourceType))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (tools.ToolConfig, error) {
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	UseClientAuthorization() bool
	GetAuthTokenHeaderName() string
	LookerApiSettings() *rtl.ApiSettings
	GetLookerSDK(context.Context, string) (*v4.LookerSDK, error)
	GetHostURL(context.Context, *v4.LookerSDK) (string, error)
}

type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(context.Context) (tools.Tool, error) {
	if cfg.Description == "" {
		return nil, fmt.Errorf("description is required for tool %q", cfg.Name)
	}

	sourceQueriesParameter := parameters.NewArrayParameter(
		"source_queries",
		"The queries whose results will be merged, one per explore. Order is "+
			"significant: the first entry is the primary query and the results "+
			"of every later entry are joined onto it, like a SQL left outer "+
			"join. Each entry is an object with the required keys `model`, "+
			"`explore` and `fields`, plus the optional keys `name`, `filters`, "+
			"`pivots`, `sorts`, `limit`, `filter_expression`, "+
			"`dynamic_fields`, `tz` and `merge_fields`. `merge_fields` "+
			"declares how the entry lines up with the merged results: it is an "+
			"array of objects with `source_field_name` (a field of this entry) "+
			"and `field_name` (the field of the primary query it maps onto), "+
			"e.g. [{\"source_field_name\": \"events.event_date\", "+
			"\"field_name\": \"orders.created_date\"}].",
		parameters.NewMapParameter("source_query", "A query to be merged, keyed by query attribute.", ""),
	)
	pivotsParameter := parameters.NewArrayParameter(
		"pivots",
		"The pivots of the merged results. Names refer to fields of the merged results.",
		parameters.NewStringParameter("pivot_field", "A field to be used as a pivot in the merged results"),
		parameters.WithArrayDefault([]any{}),
	)
	sortsParameter := parameters.NewArrayParameter(
		"sorts",
		"The sorts of the merged results like \"field.id desc 0\".",
		parameters.NewStringParameter("sort_field", "A field to be used as a sort in the merged results"),
		parameters.WithArrayDefault([]any{}),
	)
	limitParameter := parameters.NewIntParameter("limit", "The row limit of the merged results.", parameters.WithIntDefault(500))
	columnLimitParameter := parameters.NewIntParameter("column_limit", "An optional column limit for the merged results.", parameters.WithIntRequired(false))
	totalParameter := parameters.NewBooleanParameter("total", "Whether to include a totals row in the merged results.", parameters.WithBooleanDefault(false))
	dynamicFieldsParameter := parameters.NewArrayParameter(
		"dynamic_fields",
		"An optional array of dynamic fields (table calculations, custom measures, custom dimensions) computed over the merged results.",
		parameters.NewMapParameter("dynamic_field", "A dynamic field definition", ""),
		parameters.WithArrayDefault([]any{}),
	)
	visConfigParameter := parameters.NewMapParameter("vis_config", "The visualization config for the merged results", "", parameters.WithMapDefault(map[string]any{}))

	allParameters := parameters.Parameters{
		sourceQueriesParameter,
		pivotsParameter,
		sortsParameter,
		limitParameter,
		columnLimitParameter,
		totalParameter,
		dynamicFieldsParameter,
		visConfigParameter,
	}

	// finish tool setup
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewWriteAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
			allParameters,
		),
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) GetSourceName() string {
	return t.Cfg.Source
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) ValidateSource(source sources.Source) error {
	_, ok := source.(compatibleSource)
	if !ok {
		return fmt.Errorf("invalid source for %q tool: source %q is not a compatible type", t.Cfg.Type, t.Cfg.Source)
	}
	return nil
}

// stripQuotes removes a single layer of matching wrapping quotes, mirroring the
// leniency of lookercommon.ProcessQueryArgs for filter keys and values.
func stripQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return s
}

// asStringSlice converts a JSON array of strings into a []string.
func asStringSlice(key string, v any) ([]string, error) {
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("%q must be an array of strings, got %T", key, v)
	}
	out := make([]string, 0, len(raw))
	for i, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%q element #%d must be a string, got %T", key, i, item)
		}
		out = append(out, s)
	}
	return out, nil
}

// asRowLimit renders a row limit as the string Looker expects. Numbers arriving
// from JSON may be typed as int, int64 or float64 depending on how the value
// was decoded, so all three are accepted alongside a bare string.
func asRowLimit(key string, v any) (string, error) {
	switch n := v.(type) {
	case string:
		if _, err := strconv.Atoi(n); err != nil {
			return "", fmt.Errorf("%q must be an integer, got %q", key, n)
		}
		return n, nil
	case int:
		return strconv.Itoa(n), nil
	case int64:
		return strconv.FormatInt(n, 10), nil
	case float64:
		if n != float64(int64(n)) {
			return "", fmt.Errorf("%q must be an integer, got %v", key, n)
		}
		return strconv.FormatInt(int64(n), 10), nil
	default:
		return "", fmt.Errorf("%q must be an integer, got %T", key, v)
	}
}

// asMergeFields converts the `merge_fields` entry of a source query into the
// SDK representation of a field mapping.
func asMergeFields(v any) (*[]v4.MergeFields, error) {
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("\"merge_fields\" must be an array of objects, got %T", v)
	}
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]v4.MergeFields, 0, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("\"merge_fields\" element #%d must be an object, got %T", i, item)
		}
		mf := v4.MergeFields{}
		for key, val := range m {
			s, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("\"merge_fields\" element #%d key %q must be a string, got %T", i, key, val)
			}
			switch key {
			case "field_name":
				mf.FieldName = &s
			case "source_field_name":
				mf.SourceFieldName = &s
			default:
				return nil, fmt.Errorf("\"merge_fields\" element #%d has unknown key %q, expected \"field_name\" or \"source_field_name\"", i, key)
			}
		}
		if mf.FieldName == nil || mf.SourceFieldName == nil {
			return nil, fmt.Errorf("\"merge_fields\" element #%d requires both \"field_name\" and \"source_field_name\"", i)
		}
		out = append(out, mf)
	}
	return &out, nil
}

// marshalDynamicFields renders an array of dynamic field definitions as the
// JSON string Looker expects, or nil when there are none.
func marshalDynamicFields(v any) (*string, error) {
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("\"dynamic_fields\" must be an array of objects, got %T", v)
	}
	if len(raw) == 0 {
		return nil, nil
	}
	jsonBytes, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("error marshaling dynamic_fields: %w", err)
	}
	jsonStr := string(jsonBytes)
	return &jsonStr, nil
}

// sourceQuery holds one parsed `source_queries` entry: the query to create in
// Looker plus the metadata describing how it joins into the merged results.
type sourceQuery struct {
	name        string
	model       string
	explore     string
	writeQuery  *v4.WriteQuery
	mergeFields *[]v4.MergeFields
}

// processSourceQuery turns a single `source_queries` entry into a WriteQuery.
// The default timezone is left unset so Looker applies the instance default;
// callers wanting a specific timezone pass `tz`.
func processSourceQuery(idx int, raw any) (*sourceQuery, error) {
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("source query #%d must be an object, got %T", idx, raw)
	}

	unknown := []string{}
	for key := range m {
		if !slices.Contains(sourceQueryKeys, key) {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("source query #%d has unknown keys %v, expected any of %v", idx, unknown, sourceQueryKeys)
	}

	model, ok := m["model"].(string)
	if !ok || model == "" {
		return nil, fmt.Errorf("source query #%d requires a non-empty string %q", idx, "model")
	}
	explore, ok := m["explore"].(string)
	if !ok || explore == "" {
		return nil, fmt.Errorf("source query #%d requires a non-empty string %q", idx, "explore")
	}
	rawFields, ok := m["fields"]
	if !ok {
		return nil, fmt.Errorf("source query #%d requires %q", idx, "fields")
	}
	fields, err := asStringSlice("fields", rawFields)
	if err != nil {
		return nil, fmt.Errorf("source query #%d: %w", idx, err)
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("source query #%d requires at least one entry in %q", idx, "fields")
	}

	sq := &sourceQuery{name: explore, model: model, explore: explore}
	if name, ok := m["name"].(string); ok && name != "" {
		sq.name = name
	}

	wq := v4.WriteQuery{
		Model:  model,
		View:   explore,
		Fields: &fields,
	}

	if raw, ok := m["filters"]; ok && raw != nil {
		filters, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("source query #%d: %q must be an object, got %T", idx, "filters", raw)
		}
		// Strip a single layer of wrapping quotes from keys and string values,
		// matching the leniency of the `query` tool.
		processed := make(map[string]any, len(filters))
		for k, v := range filters {
			if s, ok := v.(string); ok {
				processed[stripQuotes(k)] = stripQuotes(s)
				continue
			}
			processed[stripQuotes(k)] = v
		}
		wq.Filters = &processed
	}
	if raw, ok := m["pivots"]; ok && raw != nil {
		pivots, err := asStringSlice("pivots", raw)
		if err != nil {
			return nil, fmt.Errorf("source query #%d: %w", idx, err)
		}
		wq.Pivots = &pivots
	}
	if raw, ok := m["sorts"]; ok && raw != nil {
		sorts, err := asStringSlice("sorts", raw)
		if err != nil {
			return nil, fmt.Errorf("source query #%d: %w", idx, err)
		}
		wq.Sorts = &sorts
	}
	if raw, ok := m["limit"]; ok && raw != nil {
		limit, err := asRowLimit("limit", raw)
		if err != nil {
			return nil, fmt.Errorf("source query #%d: %w", idx, err)
		}
		wq.Limit = &limit
	}
	if raw, ok := m["filter_expression"]; ok && raw != nil {
		fe, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("source query #%d: %q must be a string, got %T", idx, "filter_expression", raw)
		}
		wq.FilterExpression = &fe
	}
	if raw, ok := m["tz"]; ok && raw != nil {
		tz, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("source query #%d: %q must be a string, got %T", idx, "tz", raw)
		}
		wq.QueryTimezone = &tz
	}
	if raw, ok := m["dynamic_fields"]; ok && raw != nil {
		df, err := marshalDynamicFields(raw)
		if err != nil {
			return nil, fmt.Errorf("source query #%d: %w", idx, err)
		}
		wq.DynamicFields = df
	}
	if raw, ok := m["merge_fields"]; ok && raw != nil {
		mf, err := asMergeFields(raw)
		if err != nil {
			return nil, fmt.Errorf("source query #%d: %w", idx, err)
		}
		sq.mergeFields = mf
	}

	sq.writeQuery = &wq
	return sq, nil
}

// processSourceQueries parses the `source_queries` parameter. A merge needs at
// least two queries, so a shorter list is rejected before any query is created
// in Looker.
func processSourceQueries(raw any) ([]*sourceQuery, error) {
	rawSourceQueries, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("'source_queries' must be an array of objects, got %T", raw)
	}
	if len(rawSourceQueries) < 2 {
		return nil, fmt.Errorf("'source_queries' must contain at least two queries to merge, got %d", len(rawSourceQueries))
	}
	sourceQueries := make([]*sourceQuery, 0, len(rawSourceQueries))
	for i, rawSQ := range rawSourceQueries {
		sq, err := processSourceQuery(i, rawSQ)
		if err != nil {
			return nil, err
		}
		sourceQueries = append(sourceQueries, sq)
	}
	return sourceQueries, nil
}

// mergeURL builds a link to the Looker merged results builder, pre-populated
// with the source queries. Looker has no API endpoint that runs a merge query,
// so this URL is how the merged results get viewed.
func mergeURL(hostURL string, slugs []string) string {
	if hostURL == "" || len(slugs) == 0 {
		return ""
	}
	query := url.Values{}
	for _, slug := range slugs {
		query.Add("qids[]", slug)
	}
	return fmt.Sprintf("%s/merge?%s", strings.TrimSuffix(hostURL, "/"), query.Encode())
}

func (t Tool) Invoke(ctx context.Context, s sources.Source, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, ok := s.(compatibleSource)
	if !ok {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, nil)
	}
	logger, err := util.LoggerFromContext(ctx)
	if err != nil {
		return nil, util.NewClientServerError("unable to get logger from ctx", http.StatusInternalServerError, err)
	}
	logger.DebugContext(ctx, "params = ", params)
	paramsMap := params.AsMap()

	sourceQueries, err := processSourceQueries(paramsMap["source_queries"])
	if err != nil {
		return nil, util.NewAgentError("error building merge query request", err)
	}

	sdk, err := source.GetLookerSDK(ctx, string(accessToken))
	if err != nil {
		return nil, util.NewClientServerError("error getting sdk", http.StatusInternalServerError, err)
	}

	// Merge queries reference saved query objects, so each source query has to
	// be created first. Query creation is idempotent: an identical query
	// returns the existing object rather than a duplicate.
	mergeSourceQueries := make([]v4.MergeQuerySourceQuery, 0, len(sourceQueries))
	sourceQueryData := make([]any, 0, len(sourceQueries))
	slugs := make([]string, 0, len(sourceQueries))
	for i, sq := range sourceQueries {
		if escErr := lookercommon.EscapeUnquotedParameterFilters(ctx, sdk, sq.writeQuery, source.LookerApiSettings()); escErr != nil {
			logger.WarnContext(ctx, "skipping unquoted-parameter escape, metadata lookup failed", "error", escErr)
		}
		qresp, err := sdk.CreateQuery(*sq.writeQuery, "id,slug", source.LookerApiSettings())
		if err != nil {
			if strings.Contains(err.Error(), "status=401") {
				return nil, util.NewClientServerError("unauthorized error", http.StatusUnauthorized, err)
			}
			return nil, util.ProcessGeneralError(fmt.Errorf("error creating query for source query #%d (%s/%s): %w", i, sq.model, sq.explore, err))
		}
		name := sq.name
		mergeSourceQueries = append(mergeSourceQueries, v4.MergeQuerySourceQuery{
			Name:        &name,
			QueryId:     qresp.Id,
			MergeFields: sq.mergeFields,
		})

		data := map[string]any{
			"name":    name,
			"model":   sq.model,
			"explore": sq.explore,
		}
		if qresp.Id != nil {
			data["query_id"] = *qresp.Id
		}
		if qresp.Slug != nil {
			data["slug"] = *qresp.Slug
			slugs = append(slugs, *qresp.Slug)
		}
		sourceQueryData = append(sourceQueryData, data)
	}

	pivots, err := asStringSlice("pivots", paramsMap["pivots"])
	if err != nil {
		return nil, util.NewAgentError("error building merge query request", err)
	}
	sorts, err := asStringSlice("sorts", paramsMap["sorts"])
	if err != nil {
		return nil, util.NewAgentError("error building merge query request", err)
	}
	limit, err := asRowLimit("limit", paramsMap["limit"])
	if err != nil {
		return nil, util.NewAgentError("error building merge query request", err)
	}
	dynamicFields, err := marshalDynamicFields(paramsMap["dynamic_fields"])
	if err != nil {
		return nil, util.NewAgentError("error building merge query request", err)
	}
	total, ok := paramsMap["total"].(bool)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("'total' must be a boolean, got %T", paramsMap["total"]), nil)
	}

	wmq := v4.WriteMergeQuery{
		SourceQueries: &mergeSourceQueries,
		Pivots:        &pivots,
		Sorts:         &sorts,
		Limit:         &limit,
		Total:         &total,
		DynamicFields: dynamicFields,
	}
	if raw, ok := paramsMap["column_limit"]; ok && raw != nil {
		columnLimit, err := asRowLimit("column_limit", raw)
		if err != nil {
			return nil, util.NewAgentError("error building merge query request", err)
		}
		wmq.ColumnLimit = &columnLimit
	}
	if visConfig, ok := paramsMap["vis_config"].(map[string]any); ok && len(visConfig) > 0 {
		wmq.VisConfig = &visConfig
	}

	resp, err := sdk.CreateMergeQuery(wmq, "id,result_maker_id", source.LookerApiSettings())
	if err != nil {
		if strings.Contains(err.Error(), "status=401") {
			return nil, util.NewClientServerError("unauthorized error", http.StatusUnauthorized, err)
		}
		return nil, util.ProcessGeneralError(err)
	}
	logger.DebugContext(ctx, "resp = ", resp)

	hostURL, err := source.GetHostURL(ctx, sdk)
	if err != nil {
		logger.WarnContext(ctx, "failed to dynamically resolve public host URL, utilizing fallback", "error", err)
	}

	data := make(map[string]any)
	if resp.Id != nil {
		data["id"] = *resp.Id
	}
	if resp.ResultMakerId != nil {
		data["result_maker_id"] = *resp.ResultMakerId
	}
	data["source_queries"] = sourceQueryData
	if u := mergeURL(hostURL, slugs); u != "" {
		data["url"] = u
	}
	logger.DebugContext(ctx, "data = ", data)

	return data, nil
}

func (t Tool) RequiresClientAuthorization(source sources.Source) (bool, error) {
	s, ok := source.(compatibleSource)
	if !ok {
		return false, fmt.Errorf("invalid source for %q tool: source %q is not a compatible type", t.Cfg.Type, t.Cfg.Source)
	}
	return s.UseClientAuthorization(), nil
}

func (t Tool) GetAuthTokenHeaderName(source sources.Source) (string, error) {
	s, ok := source.(compatibleSource)
	if !ok {
		return "", fmt.Errorf("invalid source for %q tool: source %q is not a compatible type", t.Cfg.Type, t.Cfg.Source)
	}
	return s.GetAuthTokenHeaderName(), nil
}
