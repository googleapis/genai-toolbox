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
	) (string, string, error)
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
	locationId := parameters.NewStringParameter("locationId", "The location ID (e.g. 'us', 'us-central1') where the Data Product should be created.")
	dataProductId := parameters.NewStringParameter(
		"dataProductId",
		"Optional. The unique ID of the Data Product to create. If not specified, the backend will auto-generate an ID.",
		parameters.WithStringRequired(false),
	)
	displayName := parameters.NewStringParameter("displayName", "The display name of the Data Product.")
	description := parameters.NewStringParameter(
		"description",
		"Optional. The description of the Data Product.",
		parameters.WithStringRequired(false),
	)
	ownerEmails := parameters.NewArrayParameter(
		"ownerEmails",
		"The list of owner emails for the Data Product.",
		parameters.NewStringParameter("email", "Owner email address"),
	)
	accessGroups := parameters.NewArrayParameter(
		"accessGroups",
		"Optional. List of access groups to associate with the Data Product.",
		parameters.NewMapParameter("accessGroup", "Access Group details (id, displayName, description, googleGroup, serviceAccount)", ""),
		parameters.WithArrayRequired(false),
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
	prodLocID, ok := paramsMap["locationId"].(string)
	if !ok || prodLocID == "" {
		return nil, util.NewAgentError("locationId is required and must be a non-empty string", nil)
	}

	var prodID string
	if val, exists := paramsMap["dataProductId"]; exists && val != nil {
		var ok bool
		prodID, ok = val.(string)
		if !ok {
			return nil, util.NewAgentError("dataProductId must be a string", nil)
		}
	}

	displayName, ok := paramsMap["displayName"].(string)
	if !ok || displayName == "" {
		return nil, util.NewAgentError("displayName is required and must be a non-empty string", nil)
	}

	var description string
	if val, exists := paramsMap["description"]; exists && val != nil {
		var ok bool
		description, ok = val.(string)
		if !ok {
			return nil, util.NewAgentError("description must be a string", nil)
		}
	}

	rawOwners, ok := paramsMap["ownerEmails"].([]any)
	if !ok || len(rawOwners) == 0 {
		return nil, util.NewAgentError("ownerEmails is required and must be a non-empty array of strings", nil)
	}
	var ownerEmails []string
	for _, o := range rawOwners {
		email, ok := o.(string)
		if !ok || email == "" {
			return nil, util.NewAgentError("each item in ownerEmails must be a non-empty string", nil)
		}
		ownerEmails = append(ownerEmails, email)
	}

	var accessGroups []dataplex.AccessGroup
	if val, exists := paramsMap["accessGroups"]; exists && val != nil {
		rawGroups, ok := val.([]any)
		if !ok {
			return nil, util.NewAgentError("accessGroups must be an array", nil)
		}
		for _, rawG := range rawGroups {
			gMap, ok := rawG.(map[string]any)
			if !ok {
				return nil, util.NewAgentError("each access group in accessGroups must be an object", nil)
			}
			id, ok := gMap["id"].(string)
			if !ok || id == "" {
				return nil, util.NewAgentError("access group 'id' is required and must be a non-empty string", nil)
			}
			dispName, ok := gMap["displayName"].(string)
			if !ok || dispName == "" {
				return nil, util.NewAgentError("access group 'displayName' is required and must be a non-empty string", nil)
			}

			var desc string
			if dVal, dExists := gMap["description"]; dExists && dVal != nil {
				var dOk bool
				desc, dOk = dVal.(string)
				if !dOk {
					return nil, util.NewAgentError("access group 'description' must be a string", nil)
				}
			}

			var googleGroup string
			if gVal, gExists := gMap["googleGroup"]; gExists && gVal != nil {
				var gOk bool
				googleGroup, gOk = gVal.(string)
				if !gOk {
					return nil, util.NewAgentError("access group 'googleGroup' must be a string", nil)
				}
			}

			var serviceAccount string
			if sVal, sExists := gMap["serviceAccount"]; sExists && sVal != nil {
				var sOk bool
				serviceAccount, sOk = sVal.(string)
				if !sOk {
					return nil, util.NewAgentError("access group 'serviceAccount' must be a string", nil)
				}
			}

			if googleGroup == "" && serviceAccount == "" {
				return nil, util.NewAgentError("at least one of access group 'googleGroup' or 'serviceAccount' must be a non-empty string", nil)
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

	opLocID, opID, err := source.CreateDataProduct(ctx, prodLocID, prodID, displayName, description, ownerEmails, accessGroups)
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	return map[string]string{
		"locationId":  opLocID,
		"operationId": opID,
	}, nil
}
