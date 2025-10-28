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

package searchdicomstudies

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	healthcareds "github.com/googleapis/genai-toolbox/internal/sources/healthcare"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/healthcare/common"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/healthcare/v1"
)

const kind string = "search-dicom-studies"
const (
	studyInstanceUIDKey               = "StudyInstanceUID"
	patientNameKey                    = "PatientName"
	patientIDKey                      = "PatientID"
	accessionNumberKey                = "AccessionNumber"
	referringPhysicianNameKey         = "ReferringPhysicianName"
	studyDateKey                      = "StudyDate"
	enablePatientNameFuzzyMatchingKey = "fuzzymatching"
	includeAttributesKey              = "includefield"
)

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
	Project() string
	Region() string
	DatasetID() string
	AllowedDICOMStores() map[string]struct{}
	Service() *healthcare.Service
	ServiceCreator() healthcareds.HealthcareServiceCreator
	UseClientAuthorization() bool
}

// validate compatible sources are still compatible
var _ compatibleSource = &healthcareds.Source{}

var compatibleSources = [...]string{healthcareds.SourceKind}

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

	parameters := tools.Parameters{
		tools.NewStringParameterWithDefault(studyInstanceUIDKey, "", "The UID of the DICOM study"),
		tools.NewStringParameterWithDefault(patientNameKey, "", "The name of the patient"),
		tools.NewStringParameterWithDefault(patientIDKey, "", "The ID of the patient"),
		tools.NewStringParameterWithDefault(accessionNumberKey, "", "The accession number of the study"),
		tools.NewStringParameterWithDefault(referringPhysicianNameKey, "", "The name of the referring physician"),
		tools.NewStringParameterWithDefault(studyDateKey, "", "The date of the study in the format `YYYYMMDD`. You can also specify a date range in the format `YYYYMMDD-YYYYMMDD`"),
		tools.NewBooleanParameterWithDefault(enablePatientNameFuzzyMatchingKey, false, `Whether to enable fuzzy matching for patient names. Fuzzy matching will perform tokenization and normalization of both the value of PatientName in the query and the stored value. It will match if any search token is a prefix of any stored token. For example, if PatientName is "John^Doe", then "jo", "Do" and "John Doe" will all match. However "ohn" will not match`),
		tools.NewArrayParameterWithDefault(includeAttributesKey, []any{}, "List of attributeIDs, such as DICOM tag IDs or keywords. Set to [\"all\"] to return all available tags.", tools.NewStringParameter("attributeID", "The attributeID to include. Set to 'all' to return all available tags")),
	}
	if len(s.AllowedDICOMStores()) != 1 {
		parameters = append(parameters, tools.NewStringParameter(common.StoreKey, "The DICOM store ID to get details for."))
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, parameters)

	// finish tool setup
	t := Tool{
		Name:           cfg.Name,
		Kind:           kind,
		Parameters:     parameters,
		AuthRequired:   cfg.AuthRequired,
		Project:        s.Project(),
		Region:         s.Region(),
		Dataset:        s.DatasetID(),
		AllowedStores:  s.AllowedDICOMStores(),
		UseClientOAuth: s.UseClientAuthorization(),
		ServiceCreator: s.ServiceCreator(),
		Service:        s.Service(),
		manifest:       tools.Manifest{Description: cfg.Description, Parameters: parameters.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:    mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Name           string           `yaml:"name"`
	Kind           string           `yaml:"kind"`
	AuthRequired   []string         `yaml:"authRequired"`
	UseClientOAuth bool             `yaml:"useClientOAuth"`
	Parameters     tools.Parameters `yaml:"parameters"`

	Project, Region, Dataset string
	AllowedStores            map[string]struct{}
	Service                  *healthcare.Service
	ServiceCreator           healthcareds.HealthcareServiceCreator
	manifest                 tools.Manifest
	mcpManifest              tools.McpManifest
}

func (t Tool) Invoke(ctx context.Context, params tools.ParamValues, accessToken tools.AccessToken) (any, error) {
	storeID, err := common.ValidateAndFetchStoreID(params, t.AllowedStores)
	if err != nil {
		return nil, err
	}

	svc := t.Service
	// Initialize new service if using user OAuth token
	if t.UseClientOAuth {
		tokenStr, err := accessToken.ParseBearerToken()
		if err != nil {
			return nil, fmt.Errorf("error parsing access token: %w", err)
		}
		svc, err = t.ServiceCreator(tokenStr)
		if err != nil {
			return nil, fmt.Errorf("error creating service from OAuth access token: %w", err)
		}
	}

	paramsMap := params.AsMap()
	var opts []googleapi.CallOption
	if attributes, ok := paramsMap[includeAttributesKey]; ok {
		if _, ok := attributes.([]any); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string array", includeAttributesKey)
		}
		attributeIDsSlice, err := tools.ConvertAnySliceToTyped(attributes.([]any), "string")
		if err != nil {
			return nil, fmt.Errorf("can't convert '%s' to array of strings: %s", includeAttributesKey, err)
		}
		attributeIDs := attributeIDsSlice.([]string)
		if len(attributeIDs) != 0 {
			opts = append(opts, googleapi.QueryParameter(includeAttributesKey, strings.Join(attributeIDs, ",")))
		}
	}
	if fuzzymatching, ok := paramsMap[enablePatientNameFuzzyMatchingKey]; ok {
		if _, ok := fuzzymatching.(bool); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a boolean", enablePatientNameFuzzyMatchingKey)
		}
		opts = append(opts, googleapi.QueryParameter(enablePatientNameFuzzyMatchingKey, fmt.Sprintf("%t", fuzzymatching.(bool))))
	}
	if studyInstanceUID, ok := paramsMap[studyInstanceUIDKey]; ok {
		if _, ok := studyInstanceUID.(string); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string", studyInstanceUIDKey)
		}
		if studyInstanceUID.(string) != "" {
			opts = append(opts, googleapi.QueryParameter(studyInstanceUIDKey, studyInstanceUID.(string)))
		}
	}
	if patientName, ok := paramsMap[patientNameKey]; ok {
		if _, ok := patientName.(string); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string", patientNameKey)
		}
		if patientName.(string) != "" {
			opts = append(opts, googleapi.QueryParameter(patientNameKey, patientName.(string)))
		}
	}
	if patientID, ok := paramsMap[patientIDKey]; ok {
		if _, ok := patientID.(string); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string", patientIDKey)
		}
		if patientID.(string) != "" {
			opts = append(opts, googleapi.QueryParameter(patientIDKey, patientID.(string)))
		}
	}
	if accessionNumber, ok := paramsMap[accessionNumberKey]; ok {
		if _, ok := accessionNumber.(string); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string", accessionNumberKey)
		}
		if accessionNumber.(string) != "" {
			opts = append(opts, googleapi.QueryParameter(accessionNumberKey, accessionNumber.(string)))
		}
	}
	if referringPhysicianName, ok := paramsMap[referringPhysicianNameKey]; ok {
		if _, ok := referringPhysicianName.(string); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string", referringPhysicianNameKey)
		}
		if referringPhysicianName.(string) != "" {
			opts = append(opts, googleapi.QueryParameter(referringPhysicianNameKey, referringPhysicianName.(string)))
		}
	}
	if studyDate, ok := paramsMap[studyDateKey]; ok {
		if _, ok := studyDate.(string); !ok {
			return nil, fmt.Errorf("invalid '%s' parameter; expected a string", studyDateKey)
		}
		if studyDate.(string) != "" {
			opts = append(opts, googleapi.QueryParameter(studyDateKey, studyDate.(string)))
		}
	}

	name := fmt.Sprintf("projects/%s/locations/%s/datasets/%s/dicomStores/%s", t.Project, t.Region, t.Dataset, storeID)
	resp, err := svc.Projects.Locations.Datasets.DicomStores.SearchForStudies(name, "studies").Do(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to search dicom studies: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}
	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("search: status %d %s: %s", resp.StatusCode, resp.Status, respBytes)
	}
	if len(respBytes) == 0 {
		return []interface{}{}, nil
	}
	var result []interface{}
	if err := json.Unmarshal([]byte(string(respBytes)), &result); err != nil {
		return nil, fmt.Errorf("could not unmarshal response as list: %w", err)
	}
	return result, nil
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

func (t Tool) RequiresClientAuthorization() bool {
	return t.UseClientOAuth
}
