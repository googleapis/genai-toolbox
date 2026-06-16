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

package dataplexcreatedataproduct

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/sources/dataplex"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
)

const resourceType string = "dataplex-create-data-product"

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
	CreateDataProduct(
		ctx context.Context,
		name string,
		displayName string,
		description string,
		ownerEmails []string,
		accessGroups []dataplex.AccessGroup,
	) (string, error)
}

type Config struct {
	Name         string                 `yaml:"name" validate:"required"`
	Type         string                 `yaml:"type" validate:"required"`
	Source       string                 `yaml:"source" validate:"required"`
	Description  string                 `yaml:"description"`
	AuthRequired []string               `yaml:"authRequired"`
	Annotations  *tools.ToolAnnotations `yaml:"annotations,omitempty"`

	ScopesRequired []string `yaml:"scopesRequired"`
}

// validate interface
var _ tools.ToolConfig = Config{}

func (cfg Config) ToolConfigType() string {
	return resourceType
}

func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	name := parameters.NewStringParameter("name", "Required. The resource name of the Data Product in the format projects/{project}/locations/{location}/dataProducts/{dataProduct}.")
	displayName := parameters.NewStringParameter("displayName", "Required. The display name of the Data Product.")
	description := parameters.NewStringParameterWithRequired("description", "Optional. The description of the Data Product.", false)
	ownerEmails := parameters.NewArrayParameter(
		"ownerEmails",
		"Required. The list of owner emails for the Data Product.",
		parameters.NewStringParameter("email", "Owner email address"),
	)
	accessGroups := parameters.NewArrayParameterWithRequired(
		"accessGroups",
		"Optional. List of access groups to associate with the Data Product.",
		false,
		parameters.NewMapParameter("accessGroup", "Access Group details (id, displayName, description, googleGroup, serviceAccount)", ""),
	)

	params := parameters.Parameters{name, displayName, description, ownerEmails, accessGroups}

	t := Tool{
		Config:     cfg,
		Parameters: params,
		manifest: tools.Manifest{
			Description:  cfg.Description,
			Parameters:   params.Manifest(),
			AuthRequired: cfg.AuthRequired,
		},
	}
	return t, nil
}

type Tool struct {
	Config
	Parameters parameters.Parameters
	manifest   tools.Manifest
}

func (t Tool) GetName() string {
	return t.Name
}

func (t Tool) GetDescription() string {
	return t.Description
}

func (t Tool) GetAuthRequired() []string {
	return t.AuthRequired
}

func (t Tool) GetAnnotations() *tools.ToolAnnotations {
	return tools.GetAnnotationsOrDefault(t.Annotations, tools.NewDestructiveAnnotations)
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Source, t.Name, t.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()
	name, ok := paramsMap["name"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'name' parameter: %v", paramsMap["name"]), nil)
	}

	parts := strings.Split(name, "/")
	if len(parts) < 6 || parts[0] != "projects" || parts[2] != "locations" || parts[4] != "dataProducts" {
		err = fmt.Errorf("invalid name format: must be in the form projects/{project}/locations/{location}/dataProducts/{dataProduct}")
		return nil, util.NewAgentError(err.Error(), err)
	}

	displayName, ok := paramsMap["displayName"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'displayName' parameter: %v", paramsMap["displayName"]), nil)
	}

	description, _ := paramsMap["description"].(string)

	rawOwners, ok := paramsMap["ownerEmails"].([]any)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'ownerEmails' parameter: %v", paramsMap["ownerEmails"]), nil)
	}
	var ownerEmails []string
	for _, o := range rawOwners {
		email, ok := o.(string)
		if !ok {
			return nil, util.NewAgentError(fmt.Sprintf("invalid owner email type: expected string, got %T", o), nil)
		}
		ownerEmails = append(ownerEmails, email)
	}

	var accessGroups []dataplex.AccessGroup
	if rawGroups, ok := paramsMap["accessGroups"].([]any); ok {
		for _, rawG := range rawGroups {
			gMap, ok := rawG.(map[string]any)
			if !ok {
				return nil, util.NewAgentError(fmt.Sprintf("invalid accessGroup item: expected map, got %T", rawG), nil)
			}
			id, _ := gMap["id"].(string)
			dispName, _ := gMap["displayName"].(string)
			desc, _ := gMap["description"].(string)
			googleGroup, _ := gMap["googleGroup"].(string)
			serviceAccount, _ := gMap["serviceAccount"].(string)

			if id == "" {
				return nil, util.NewAgentError("access group 'id' is required", nil)
			}

			if dispName == "" {
				return nil, util.NewAgentError("access group 'displayName' is required", nil)
			}

			if googleGroup == "" && serviceAccount == "" {
				return nil, util.NewAgentError("at least one of access group 'googleGroup' or 'serviceAccount' is required", nil)
			}

			accessGroups = append(accessGroups, dataplex.AccessGroup{
				ID:             id,
				DisplayName:    dispName,
				Description:    desc,
				GoogleGroup:    googleGroup,
				ServiceAccount: serviceAccount,
			})
		}
	}

	opName, err := source.CreateDataProduct(ctx, name, displayName, description, ownerEmails, accessGroups)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	return map[string]string{
		"operationName": opName,
	}, nil
}

func (t Tool) EmbedParams(ctx context.Context, paramValues parameters.ParamValues, embeddingModelsMap map[string]embeddingmodels.EmbeddingModel) (parameters.ParamValues, error) {
	return parameters.EmbedParams(ctx, t.Parameters, paramValues, embeddingModelsMap, nil)
}

func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return tools.IsAuthorized(t.AuthRequired, verifiedAuthServices)
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	return false, nil
}

func (t Tool) GetAuthTokenHeaderName(resourceMgr tools.SourceProvider) (string, error) {
	return "Authorization", nil
}

func (t Tool) GetParameters() parameters.Parameters {
	return t.Parameters
}

func (t Tool) GetScopesRequired() []string {
	return t.ScopesRequired
}
