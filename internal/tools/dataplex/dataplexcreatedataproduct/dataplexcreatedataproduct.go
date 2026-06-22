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

	yaml "github.com/goccy/go-yaml"
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
	actual := Config{ConfigBase: tools.ConfigBase{Name: name}}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type compatibleSource interface {
	CreateDataProduct(
		ctx context.Context,
		locationId string,
		dataProductId string,
		displayName string,
		description string,
		ownerEmails []string,
		accessGroups []dataplex.AccessGroup,
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

func (cfg Config) Initialize() (tools.Tool, error) {
	locationId := parameters.NewStringParameter("locationId", "Required. The location ID (e.g. 'us', 'us-central1') where the Data Product should be created.")
	dataProductId := parameters.NewStringParameterWithRequired("dataProductId", "Optional. The unique ID of the Data Product to create. If not specified, the backend will auto-generate an ID.", false)
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

	params := parameters.Parameters{locationId, dataProductId, displayName, description, ownerEmails, accessGroups}

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
	locationId, ok := paramsMap["locationId"].(string)
	if !ok || locationId == "" {
		return nil, util.NewAgentError("locationId is required and must be a string", nil)
	}

	dataProductId, _ := paramsMap["dataProductId"].(string)

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

	resp, err := source.CreateDataProduct(ctx, locationId, dataProductId, displayName, description, ownerEmails, accessGroups)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	return resp, nil
}
