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

package fhirpatientsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	healthcareds "github.com/googleapis/genai-toolbox/internal/sources/cloudhealthcare"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/cloudhealthcare/common"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/healthcare/v1"
)

const kind string = "cloud-healthcare-fhir-patient-search"
const (
	activeKey           = "active"
	cityKey             = "city"
	countryKey          = "country"
	postalCodeKey       = "postalcode"
	stateKey            = "state"
	addressSubstringKey = "addressSubstring"
	birthDateRangeKey   = "birthDateRange"
	deathDateRangeKey   = "deathDateRange"
	deceasedKey         = "deceased"
	emailKey            = "email"
	genderKey           = "gender"
	addressUseKey       = "addressUse"
	nameKey             = "name"
	givenNameKey        = "givenName"
	familyNameKey       = "familyName"
	phoneKey            = "phone"
	languageKey         = "language"
	identifierKey       = "identifier"
	summaryKey          = "summary"
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
	AllowedFHIRStores() map[string]struct{}
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

	params := parameters.Parameters{
		parameters.NewStringParameterWithDefault(activeKey, "", "Whether the patient record is active. Use true or false"),
		parameters.NewStringParameterWithDefault(cityKey, "", "The city of the patient's address"),
		parameters.NewStringParameterWithDefault(countryKey, "", "The country of the patient's address"),
		parameters.NewStringParameterWithDefault(postalCodeKey, "", "The postal code of the patient's address"),
		parameters.NewStringParameterWithDefault(stateKey, "", "The state of the patient's address"),
		parameters.NewStringParameterWithDefault(addressSubstringKey, "", "A substring to search for in any address field"),
		parameters.NewStringParameterWithDefault(birthDateRangeKey, "", "A date range for the patient's birthdate in the format YYYY-MM-DD/YYYY-MM-DD. Omit the first or second date to indicate open-ended ranges (e.g. '/2000-01-01' or '1950-01-01/')"),
		parameters.NewStringParameterWithDefault(deathDateRangeKey, "", "A date range for the patient's death date in the format YYYY-MM-DD/YYYY-MM-DD. Omit the first or second date to indicate open-ended ranges (e.g. '/2000-01-01' or '1950-01-01/')"),
		parameters.NewStringParameterWithDefault(deceasedKey, "", "Whether the patient is deceased. Use true or false"),
		parameters.NewStringParameterWithDefault(emailKey, "", "The patient's email address"),
		parameters.NewStringParameterWithDefault(genderKey, "", "The patient's gender. Must be one of 'male', 'female', 'other', or 'unknown'"),
		parameters.NewStringParameterWithDefault(addressUseKey, "", "The use of the patient's address. Must be one of 'home', 'work', 'temp', 'old', or 'billing'"),
		parameters.NewStringParameterWithDefault(nameKey, "", "The patient's name. Can be a family name, given name, or both"),
		parameters.NewStringParameterWithDefault(givenNameKey, "", "A portion of the given name of the patient"),
		parameters.NewStringParameterWithDefault(familyNameKey, "", "A portion of the family name of the patient"),
		parameters.NewStringParameterWithDefault(phoneKey, "", "The patient's phone number"),
		parameters.NewStringParameterWithDefault(languageKey, "", "The patient's preferred language. Must be a valid BCP-47 code (e.g. 'en-US', 'es')"),
		parameters.NewStringParameterWithDefault(identifierKey, "", "An identifier for the patient"),
		parameters.NewBooleanParameterWithDefault(summaryKey, true, "Requests the server to return a subset of the resource. Return a limited subset of elements from the resource. Enabled by default to reduce response size. Use get-fhir-resource tool to get full resource details (preferred) or set to false to disable."),
	}

	if len(s.AllowedFHIRStores()) != 1 {
		params = append(params, parameters.NewStringParameter(common.StoreKey, "The FHIR store ID to retrieve the resource from."))
	}
	mcpManifest := tools.GetMcpManifest(cfg.Name, cfg.Description, cfg.AuthRequired, params, nil)

	// finish tool setup
	t := Tool{
		Config:         cfg,
		Parameters:     params,
		Project:        s.Project(),
		Region:         s.Region(),
		Dataset:        s.DatasetID(),
		AllowedStores:  s.AllowedFHIRStores(),
		UseClientOAuth: s.UseClientAuthorization(),
		ServiceCreator: s.ServiceCreator(),
		Service:        s.Service(),
		manifest:       tools.Manifest{Description: cfg.Description, Parameters: params.Manifest(), AuthRequired: cfg.AuthRequired},
		mcpManifest:    mcpManifest,
	}
	return t, nil
}

// validate interface
var _ tools.Tool = Tool{}

type Tool struct {
	Config
	UseClientOAuth bool                  `yaml:"useClientOAuth"`
	Parameters     parameters.Parameters `yaml:"parameters"`

	Project, Region, Dataset string
	AllowedStores            map[string]struct{}
	Service                  *healthcare.Service
	ServiceCreator           healthcareds.HealthcareServiceCreator
	manifest                 tools.Manifest
	mcpManifest              tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
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

	var summary bool
	var opts []googleapi.CallOption
	for k, v := range params.AsMap() {
		if k == common.StoreKey {
			continue
		}
		if k == summaryKey {
			var ok bool
			summary, ok = v.(bool)
			if !ok {
				return nil, fmt.Errorf("invalid '%s' parameter; expected a boolean", summaryKey)
			}
			continue
		}

		val, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("invalid parameter '%s'; expected a string", k)
		}
		if val == "" {
			continue
		}
		switch k {
		case activeKey, deceasedKey, emailKey, genderKey, phoneKey, languageKey, identifierKey:
			opts = append(opts, googleapi.QueryParameter(k, val))
		case cityKey, countryKey, postalCodeKey, stateKey:
			opts = append(opts, googleapi.QueryParameter("address-"+k, val))
		case addressSubstringKey:
			opts = append(opts, googleapi.QueryParameter("address", val))
		case birthDateRangeKey, deathDateRangeKey:
			key := "birthdate"
			if k == deathDateRangeKey {
				key = "death-date"
			}
			parts := strings.Split(val, "/")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid '%s' format; expected YYYY-MM-DD/YYYY-MM-DD", k)
			}
			var values []string
			if parts[0] != "" {
				values = append(values, "ge"+parts[0])
			}
			if parts[1] != "" {
				values = append(values, "le"+parts[1])
			}
			if len(values) != 0 {
				opts = append(opts, googleapi.QueryParameter(key, values...))
			}
		case addressUseKey:
			opts = append(opts, googleapi.QueryParameter("address-use", val))
		case nameKey:
			parts := strings.Split(val, " ")
			for _, part := range parts {
				opts = append(opts, googleapi.QueryParameter("name", part))
			}
		case givenNameKey:
			opts = append(opts, googleapi.QueryParameter("given", val))
		case familyNameKey:
			opts = append(opts, googleapi.QueryParameter("family", val))
		default:
			return nil, fmt.Errorf("unexpected parameter key %q", k)
		}
	}
	if summary {
		opts = append(opts, googleapi.QueryParameter("_summary", "text"))
	}

	name := fmt.Sprintf("projects/%s/locations/%s/datasets/%s/fhirStores/%s", t.Project, t.Region, t.Dataset, storeID)
	resp, err := svc.Projects.Locations.Datasets.FhirStores.Fhir.SearchType(name, "Patient", &healthcare.SearchResourcesRequest{ResourceType: "Patient"}).Do(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to search patient resources: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}
	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("search: status %d %s: %s", resp.StatusCode, resp.Status, respBytes)
	}
	var jsonMap map[string]interface{}
	if err := json.Unmarshal([]byte(string(respBytes)), &jsonMap); err != nil {
		return nil, fmt.Errorf("could not unmarshal response as json: %w", err)
	}
	return jsonMap, nil
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

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}
