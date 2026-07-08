// Copyright 2026 Google LLC
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

package dataplexcreatedataasset

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/tools/dataplex/dataplexcommon"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "dataplex-create-data-asset"

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
	CreateDataAsset(
		ctx context.Context,
		locationID string,
		dataProductID string,
		dataAssetID string,
		resourceType string,
		resourceProjectID string,
		resourceLocationID string,
		resourceDatasetID string,
		resourceID string,
		labels map[string]string,
		accessGroupConfigs map[string][]string,
	) (map[string]string, error)
}

type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(ctx context.Context) (tools.Tool, error) {
	locationID := parameters.NewStringParameter("locationId", "The location ID (e.g. 'us', 'us-central1') where the parent Data Product is located.")
	dataProductID := parameters.NewStringParameter("dataProductId", "The unique ID of the parent Data Product.")
	dataAssetID := parameters.NewStringParameter("dataAssetId", "The unique ID of the Data Asset to create.")

	allowedTypes := make([]any, len(dataplexcommon.SupportedResourceTypes))
	quotedTypes := make([]string, len(dataplexcommon.SupportedResourceTypes))
	for i, v := range dataplexcommon.SupportedResourceTypes {
		allowedTypes[i] = v
		quotedTypes[i] = fmt.Sprintf("\"%s\"", v)
	}
	resourceTypeDesc := "The type of resource to be represented by this data asset. Supported values are: " + strings.Join(quotedTypes, ", ") + "."
	resourceType := parameters.NewStringParameter(
		"resourceType",

		resourceTypeDesc,
		parameters.WithStringAllowedValues(allowedTypes),
	)

	resourceProjectID := parameters.NewStringParameter("resourceProjectId", "Optional. The project ID of the resource. This is NOT required if the resource is a 'CLOUD_STORAGE_BUCKET'.", parameters.WithStringRequired(false))
	resourceLocationID := parameters.NewStringParameter("resourceLocationId", "Optional. The location of the resource. This is required if the resource type is 'GEMINI_ENTERPRISE_AGENT_PLATFORM_DATASET' or 'GEMINI_ENTERPRISE_AGENT_PLATFORM_MODEL'.", parameters.WithStringRequired(false))
	resourceDatasetID := parameters.NewStringParameter("resourceDatasetId", "Optional. The dataset ID of the resource. This is required if the resource type is 'BIGQUERY_*' or 'GEMINI_ENTERPRISE_AGENT_PLATFORM_DATASET'.", parameters.WithStringRequired(false))
	resourceID := parameters.NewStringParameter("resourceId", "Optional. The resource ID of the resource.", parameters.WithStringRequired(false))
	labels := parameters.NewMapParameter("labels", "Optional. The labels associated with the Data Asset.", "string", parameters.WithMapRequired(false))
	accessGroupConfigs := parameters.NewMapParameter("accessGroupConfigs", "Optional. Map of access group configurations to associate with the Data Asset. Each key represents the access group ID, and the value is a list of string IAM role names (e.g. {'test-group': ['roles/bigquery.dataViewer']}). To find the list of supported roles that can be granted on the resource, refer to the IAM Roles documentation or use the roles:queryGrantableRoles API method (https://cloud.google.com/iam/docs/reference/rest/v1/roles/queryGrantableRoles).", "", parameters.WithMapRequired(false))

	params := parameters.Parameters{locationID, dataProductID, dataAssetID, resourceType, resourceProjectID, resourceLocationID, resourceDatasetID, resourceID, labels, accessGroupConfigs}

	t := Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewDestructiveAnnotations),
			tools.Manifest{
				Description:  cfg.Description,
				Parameters:   params.Manifest(),
				AuthRequired: cfg.AuthRequired,
			},
			params,
		),
	}
	return t, nil
}

type Tool struct {
	tools.BaseTool[Config]
}

var _ tools.Tool = Tool{}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	locationID, ok := paramsMap["locationId"].(string)
	if !ok || locationID == "" {
		return nil, util.NewAgentError("locationId is required and must be a string", nil)
	}

	dataProductID, ok := paramsMap["dataProductId"].(string)
	if !ok || dataProductID == "" {
		return nil, util.NewAgentError("dataProductId is required and must be a string", nil)
	}

	dataAssetID, ok := paramsMap["dataAssetId"].(string)
	if !ok || dataAssetID == "" {
		return nil, util.NewAgentError("dataAssetId is required and must be a string", nil)
	}

	resourceType, ok := paramsMap["resourceType"].(string)
	if !ok || resourceType == "" {
		return nil, util.NewAgentError("resourceType is required and must be a string", nil)
	}

	var resourceProjectID string
	if val, exists := paramsMap["resourceProjectId"]; exists && val != nil {
		sVal, ok := val.(string)
		if !ok {
			return nil, util.NewAgentError("resourceProjectId must be a string", nil)
		}
		resourceProjectID = sVal
	}

	var resourceLocationID string
	if val, exists := paramsMap["resourceLocationId"]; exists && val != nil {
		sVal, ok := val.(string)
		if !ok {
			return nil, util.NewAgentError("resourceLocationId must be a string", nil)
		}
		resourceLocationID = sVal
	}

	var resourceDatasetID string
	if val, exists := paramsMap["resourceDatasetId"]; exists && val != nil {
		sVal, ok := val.(string)
		if !ok {
			return nil, util.NewAgentError("resourceDatasetId must be a string", nil)
		}
		resourceDatasetID = sVal
	}

	var resourceID string
	if val, exists := paramsMap["resourceId"]; exists && val != nil {
		sVal, ok := val.(string)
		if !ok {
			return nil, util.NewAgentError("resourceId must be a string", nil)
		}
		resourceID = sVal
	}

	var labels map[string]string
	if rawLabels, ok := paramsMap["labels"].(map[string]any); ok {
		labels = make(map[string]string)
		for k, v := range rawLabels {
			sVal, ok := v.(string)
			if !ok {
				return nil, util.NewAgentError(fmt.Sprintf("invalid label value type for key %q: expected string, got %T", k, v), nil)
			}
			labels[k] = sVal
		}
	}

	accessGroupConfigs := make(map[string][]string)
	if rawConfigs, ok := paramsMap["accessGroupConfigs"].(map[string]any); ok {
		for k, v := range rawConfigs {
			rawRoles, ok := v.([]any)
			if !ok {
				return nil, util.NewAgentError(fmt.Sprintf("invalid accessGroupConfigs value type for key %q: expected array, got %T", k, v), nil)
			}
			var roles []string
			for _, r := range rawRoles {
				sRole, ok := r.(string)
				if !ok {
					return nil, util.NewAgentError(fmt.Sprintf("invalid iamRole in group %q: expected string, got %T", k, r), nil)
				}
				roles = append(roles, sRole)
			}
			accessGroupConfigs[k] = roles
		}
	}

	resp, err := source.CreateDataAsset(ctx, locationID, dataProductID, dataAssetID, resourceType, resourceProjectID, resourceLocationID, resourceDatasetID, resourceID, labels, accessGroupConfigs)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	return resp, nil
}
