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

package searchdicomseries

import (
	"context"
	"fmt"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/embeddingmodels"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/common"
	"github.com/googleapis/genai-toolbox/internal/util"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/googleapi"
)

const resourceType string = "cloud-healthcare-search-dicom-series"
const (
	studyInstanceUIDKey       = "StudyInstanceUID"
	patientNameKey            = "PatientName"
	patientIDKey              = "PatientID"
	accessionNumberKey        = "AccessionNumber"
	referringPhysicianNameKey = "ReferringPhysicianName"
	studyDateKey              = "StudyDate"
	seriesInstanceUIDKey      = "SeriesInstanceUID"
	modalityKey               = "Modality"
)

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
	AllowedDICOMStores() map[string]struct{}
	UseClientAuthorization() bool
	SearchDICOM(string, string, string, string, []googleapi.CallOption) (any, error)
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

	params := parameters.Parameters{
		parameters.NewStringParameterWithDefault(studyInstanceUIDKey, "", "The UID of the DICOM study"),
		parameters.NewStringParameterWithDefault(patientNameKey, "", "The name of the patient"),
		parameters.NewStringParameterWithDefault(patientIDKey, "", "The ID of the patient"),
		parameters.NewStringParameterWithDefault(accessionNumberKey, "", "The accession number of the series"),
		parameters.NewStringParameterWithDefault(referringPhysicianNameKey, "", "The name of the referring physician"),
		parameters.NewStringParameterWithDefault(studyDateKey, "", "The date of the study in the format `YYYYMMDD`. You can also specify a date range in the format `YYYYMMDD-YYYYMMDD`"),
		parameters.NewStringParameterWithDefault(seriesInstanceUIDKey, "", "The UID of the DICOM series"),
		parameters.NewStringParameterWithDefault(modalityKey, "", "The modality of the series"),
		parameters.NewBooleanParameterWithDefault(common.EnablePatientNameFuzzyMatchingKey, false, `Whether to enable fuzzy matching for patient names. Fuzzy matching will perform tokenization and normalization of both the value of PatientName in the query and the stored value. It will match if any search token is a prefix of any stored token. For example, if PatientName is "John^Doe", then "jo", "Do" and "John Doe" will all match. However "ohn" will not match`),
		parameters.NewArrayParameterWithDefault(common.IncludeAttributesKey, []any{}, "List of attributeIDs, such as DICOM tag IDs or keywords. Set to [\"all\"] to return all available tags.", parameters.NewStringParameter("attributeID", "The attributeID to include. Set to 'all' to return all available tags")),
	}
	if len(s.AllowedDICOMStores()) != 1 {
		params = append(params, parameters.NewStringParameter(common.StoreKey, "The DICOM store ID to get details for."))
	}
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

	storeID, err := common.ValidateAndFetchStoreID(params, source.AllowedDICOMStores())
	if err != nil {
		return nil, util.NewAgentError("failed to validate store ID", err)
	}
	var tokenStr string
	if source.UseClientAuthorization() {
		tokenStr, err = accessToken.ParseBearerToken()
		if err != nil {
			return nil, util.NewClientServerError("error parsing access token", http.StatusUnauthorized, err)
		}
	}

	opts, err := common.ParseDICOMSearchParameters(params, []string{seriesInstanceUIDKey, patientNameKey, patientIDKey, accessionNumberKey, referringPhysicianNameKey, studyDateKey, modalityKey})
	if err != nil {
		return nil, util.NewAgentError("failed to parse DICOM search parameters", err)
	}
	paramsMap := params.AsMap()
	dicomWebPath := "series"
	if studyInstanceUID, ok := paramsMap[studyInstanceUIDKey]; ok {
		id, ok := studyInstanceUID.(string)
		if !ok {
			return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter; expected a string", studyInstanceUIDKey), nil)
		}
		if id != "" {
			dicomWebPath = fmt.Sprintf("studies/%s/series", id)
		}
	}
	resp, err := source.SearchDICOM(t.Type, storeID, dicomWebPath, tokenStr, opts)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
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

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}
