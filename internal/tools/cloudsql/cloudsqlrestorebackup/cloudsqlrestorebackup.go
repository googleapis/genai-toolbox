// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cloudsqlrestorebackup

import (
	"context"
	"fmt"
	"net/http"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"google.golang.org/api/sqladmin/v1"
)

const resourceType string = "cloud-sql-restore-backup"

var _ tools.ToolConfig = Config{}

type compatibleSource interface {
	GetDefaultProject() string
	GetService(context.Context, string) (*sqladmin.Service, error)
	UseClientAuthorization() bool
	RestoreBackup(ctx context.Context, targetProject, targetInstance, sourceProject, sourceInstance, backupID, accessToken string) (any, error)
}

// Config defines the configuration for the restore-backup tool.
type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
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

// ToolConfigType returns the type of the tool.
func (cfg Config) ToolConfigType() string {
	return resourceType
}

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}
	s, ok := rawS.(compatibleSource)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source %q not compatible", resourceType, cfg.Source)
	}

	project := s.GetDefaultProject()
	var targetProjectParam parameters.Parameter
	if project != "" {
		targetProjectParam = parameters.NewStringParameterWithDefault("target_project", project, "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.")
	} else {
		targetProjectParam = parameters.NewStringParameter("target_project", "The project ID")
	}

	allParameters := parameters.Parameters{
		targetProjectParam,
		parameters.NewStringParameter("target_instance", "Cloud SQL instance ID of the target instance. This does not include the project ID."),
		parameters.NewStringParameter("backup_id", "Identifier of the backup being restored. Can be a BackupRun ID, backup name, or BackupDR backup name. Use the full backup ID as provided, do not try to parse it"),
		parameters.NewStringParameterWithRequired("source_project", "GCP project ID of the instance that the backup belongs to. Only required if the backup_id is a BackupRun ID.", false),
		parameters.NewStringParameterWithRequired("source_instance", "Cloud SQL instance ID of the instance that the backup belongs to. Only required if the backup_id is a BackupRun ID.", false),
	}
	paramManifest := allParameters.Manifest()

	if cfg.Description == "" {
		cfg.Description = "Restores a backup on a Cloud SQL instance."
	}

	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewDestructiveAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
			allParameters,
		),
	}, nil
}

// Tool represents the restore-backup tool.
type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Cfg
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, err)
	}
	paramsMap := params.AsMap()

	targetProject, ok := paramsMap["target_project"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'target_project' parameter: %v", paramsMap["target_project"]), nil)
	}
	targetInstance, ok := paramsMap["target_instance"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'target_instance' parameter: %v", paramsMap["target_instance"]), nil)
	}
	backupID, ok := paramsMap["backup_id"].(string)
	if !ok {
		return nil, util.NewAgentError(fmt.Sprintf("error casting 'backup_id' parameter: %v", paramsMap["backup_id"]), nil)
	}
	sourceProject, _ := paramsMap["source_project"].(string)
	sourceInstance, _ := paramsMap["source_instance"].(string)

	resp, err := source.RestoreBackup(ctx, targetProject, targetInstance, sourceProject, sourceInstance, backupID, string(accessToken))
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	return resp, nil
}

// Authorized checks if the tool is authorized.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) (bool, error) {
	source, err := tools.GetCompatibleSource[compatibleSource](resourceMgr, t.Cfg.Source, t.Cfg.Name, t.Cfg.Type)
	if err != nil {
		return false, err
	}

	return source.UseClientAuthorization(), nil
}
