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

package searchdicominstances

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/cloudhealthcare/common"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"google.golang.org/api/googleapi"
)

const resourceType string = "cloud-healthcare-search-dicom-instances"
const (
	studyInstanceUIDKey       = "StudyInstanceUID"
	patientNameKey            = "PatientName"
	patientIDKey              = "PatientID"
	accessionNumberKey        = "AccessionNumber"
	referringPhysicianNameKey = "ReferringPhysicianName"
	studyDateKey              = "StudyDate"
	seriesInstanceUIDKey      = "SeriesInstanceUID"
	modalityKey               = "Modality"
	sopInstanceUIDKey         = "SOPInstanceUID"
)

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
	AllowedDICOMStores() map[string]struct{}
	UseClientAuthorization() bool
	SearchDICOM(string, string, string, string, []googleapi.CallOption) (any, error)
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

	params := buildParams(false)
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
			params,
		),
	}, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) Invoke(ctx context.Context, primitiveMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
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

	opts, err := common.ParseDICOMSearchParameters(params, []string{sopInstanceUIDKey, patientNameKey, patientIDKey, accessionNumberKey, referringPhysicianNameKey, studyDateKey, modalityKey})
	if err != nil {
		return nil, util.NewAgentError("failed to parse DICOM search parameters", err)
	}
	paramsMap := params.AsMap()
	dicomWebPath := "instances"
	if studyInstanceUID, ok := paramsMap[studyInstanceUIDKey]; ok {
		id, ok := studyInstanceUID.(string)
		if !ok {
			return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter; expected a string", studyInstanceUIDKey), nil)
		}
		if id != "" {
			dicomWebPath = fmt.Sprintf("studies/%s/instances", id)
		}
	}
	if seriesInstanceUID, ok := paramsMap[seriesInstanceUIDKey]; ok {
		id, ok := seriesInstanceUID.(string)
		if !ok {
			return nil, util.NewAgentError(fmt.Sprintf("invalid '%s' parameter; expected a string", seriesInstanceUIDKey), nil)
		}
		if id != "" {
			if dicomWebPath != "instances" {
				dicomWebPath = fmt.Sprintf("%s/series/%s/instances", strings.TrimSuffix(dicomWebPath, "/instances"), id)
			} else {
				opts = append(opts, googleapi.QueryParameter(seriesInstanceUIDKey, id))
			}
		}
	}
	resp, err := source.SearchDICOM(t.Cfg.Type, storeID, dicomWebPath, tokenStr, opts)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

func (t Tool) RequiresClientAuthorization(primitiveMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](primitiveMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}
	return source.UseClientAuthorization(), nil
}

// buildParams builds the tool's parameters. When the source pins exactly one store
// (singleStore), the store param is omitted; otherwise it is included.
func buildParams(singleStore bool) parameters.Parameters {
	params := parameters.Parameters{
		parameters.NewStringParameter(studyInstanceUIDKey, "The UID of the DICOM study", parameters.WithStringDefault("")),
		parameters.NewStringParameter(patientNameKey, "The name of the patient", parameters.WithStringDefault("")),
		parameters.NewStringParameter(patientIDKey, "The ID of the patient", parameters.WithStringDefault("")),
		parameters.NewStringParameter(accessionNumberKey, "The accession number of the series", parameters.WithStringDefault("")),
		parameters.NewStringParameter(referringPhysicianNameKey, "The name of the referring physician", parameters.WithStringDefault("")),
		parameters.NewStringParameter(studyDateKey, "The date of the study in the format `YYYYMMDD`. You can also specify a date range in the format `YYYYMMDD-YYYYMMDD`", parameters.WithStringDefault("")),
		parameters.NewStringParameter(seriesInstanceUIDKey, "The UID of the DICOM series", parameters.WithStringDefault("")),
		parameters.NewStringParameter(modalityKey, "The modality of the series", parameters.WithStringDefault("")),
		parameters.NewStringParameter(sopInstanceUIDKey, "The UID of the SOP instance.", parameters.WithStringDefault("")),
		parameters.NewBooleanParameter(common.EnablePatientNameFuzzyMatchingKey, `Whether to enable fuzzy matching for patient names. Fuzzy matching will perform tokenization and normalization of both the value of PatientName in the query and the stored value. It will match if any search token is a prefix of any stored token. For example, if PatientName is "John^Doe", then "jo", "Do" and "John Doe" will all match. However "ohn" will not match`, parameters.WithBooleanDefault(false)),
		parameters.NewArrayParameter(common.IncludeAttributesKey, "List of attributeIDs, such as DICOM tag IDs or keywords. Set to [\"all\"] to return all available tags.", parameters.NewStringParameter("attributeID", "The attributeID to include. Set to 'all' to return all available tags"), parameters.WithArrayDefault([]any{})),
	}
	if !singleStore {
		params = append(params, parameters.NewStringParameter(common.StoreKey, "The DICOM store ID to get details for."))
	}
	return params
}

// resolveParams builds the tool's parameters using the source's configured FHIR/DICOM stores.
func (t Tool) resolveParams(srcs map[string]sources.Source) (parameters.Parameters, error) {
	s, err := tools.GetCompatibleSourceFromMap[compatibleSource](srcs, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, err
	}

	return buildParams(len(s.AllowedDICOMStores()) == 1), nil
}

// GetParameters returns the tool's parameters, resolved against the source.
func (t Tool) GetParameters(srcs map[string]sources.Source) (parameters.Parameters, error) {
	return t.resolveParams(srcs)
}

// Manifest returns the tool's manifest, resolved against the source.
func (t Tool) Manifest(srcs map[string]sources.Source) (tools.Manifest, error) {
	allParameters, err := t.resolveParams(srcs)
	if err != nil {
		return tools.Manifest{}, err
	}
	return tools.Manifest{Description: t.Cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: t.Cfg.AuthRequired}, nil
}
