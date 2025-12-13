// Copyright 2025 Google LLC
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
	"regexp"
	"strconv"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"github.com/googleapis/genai-toolbox/internal/sources/cloudsqladmin"
	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/util/parameters"
	"google.golang.org/api/sqladmin/v1"
)

const kind string = "cloud-sql-restore-backup"

var _ tools.ToolConfig = Config{}
var backupDRRegex = regexp.MustCompile(`^projects/([^/]+)/locations/([^/]+)/backupVaults/([^/]+)/dataSources/([^/]+)/backups/([^/]+)$`)

// Config defines the configuration for the restore-backup tool.
type Config struct {
	Name         string   `yaml:"name" validate:"required"`
	Kind         string   `yaml:"kind" validate:"required"`
	Description  string   `yaml:"description"`
	Source       string   `yaml:"source" validate:"required"`
	AuthRequired []string `yaml:"authRequired"`
}

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

// ToolConfigKind returns the kind of the tool.
func (cfg Config) ToolConfigKind() string {
	return kind
}

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(srcs map[string]sources.Source) (tools.Tool, error) {
	rawS, ok := srcs[cfg.Source]
	if !ok {
		return nil, fmt.Errorf("no source named %q configured", cfg.Source)
	}
	s, ok := rawS.(*cloudsqladmin.Source)
	if !ok {
		return nil, fmt.Errorf("invalid source for %q tool: source kind must be `cloud-sql-admin`", kind)
	}

	project := s.DefaultProject
	var targetProjectParam parameters.Parameter
	if project != "" {
		targetProjectParam = parameters.NewStringParameterWithDefault("target_project", project, "The GCP project ID. This is pre-configured; do not ask for it unless the user explicitly provides a different one.")
	} else {
		targetProjectParam = parameters.NewStringParameter("target_project", "The project ID")
	}

	allParameters := parameters.Parameters{
		targetProjectParam,
		parameters.NewStringParameter("target_instance", "Cloud SQL instance ID of the target instance. This does not include the project ID."),
		parameters.NewStringParameter("backup_id", "Identifier of the backup being restored. Can be a BackupRun ID, backup name, or BackupDR backup name."),
		parameters.NewStringParameterWithRequired("source_project", "GCP project ID of the instance that the backup belongs to. Only required if the backup being restored is a standard backup.", false),
		parameters.NewStringParameterWithRequired("source_instance", "Cloud SQL instance ID of the instance that the backup belongs to. Only required if the backup being restored is a standard backup.", false),
	}
	paramManifest := allParameters.Manifest()

	description := cfg.Description
	if description == "" {
		description = "Restores a backup on a Cloud SQL instance. The call returns a Cloud SQL Operation object. Call wait_for_operation tool after this, make sure to use multiplier as 4 to poll the operation status till it is marked DONE."
	}

	mcpManifest := tools.GetMcpManifest(cfg.Name, description, cfg.AuthRequired, allParameters, nil)

	return Tool{
		Config:      cfg,
		Source:      s,
		AllParams:   allParameters,
		manifest:    tools.Manifest{Description: description, Parameters: paramManifest, AuthRequired: cfg.AuthRequired},
		mcpManifest: mcpManifest,
	}, nil
}

// Tool represents the restore-backup tool.
type Tool struct {
	Config
	Source      *cloudsqladmin.Source
	AllParams   parameters.Parameters `yaml:"allParams"`
	manifest    tools.Manifest
	mcpManifest tools.McpManifest
}

func (t Tool) ToConfig() tools.ToolConfig {
	return t.Config
}

func (t Tool) Invoke(ctx context.Context, resourceMgr tools.SourceProvider, params parameters.ParamValues, accessToken tools.AccessToken) (any, error) {
	paramsMap := params.AsMap()

	targetProject, ok := paramsMap["target_project"].(string)
	if !ok {
		return nil, fmt.Errorf("error casting 'target_project' parameter: %v", paramsMap["target_project"])
	}
	targetInstance, ok := paramsMap["target_instance"].(string)
	if !ok {
		return nil, fmt.Errorf("error casting 'target_instance' parameter: %v", paramsMap["target_instance"])
	}
	backupID, ok := paramsMap["backup_id"].(string)
	if !ok {
		return nil, fmt.Errorf("error casting 'backup_id' parameter: %v", paramsMap["backup_id"])
	}

	request := &sqladmin.InstancesRestoreBackupRequest{}

	// There are 3 scenarios for the backup identifier:
	// 1. The identifier is an int64 containing the timestamp of the BackupRun.
	//    This is used to restore standard backups, and the RestoreBackupContext
	//    field should be populated with the backup ID and source instance info.
	// 2. The identifier is a string of the format
	//    'projects/{project-id}/locations/{location}/backupVaults/{backupvault}/dataSources/{datasource}/backups/{backup-uid}'.
	//    This is used to restore BackupDR backups, and the BackupdrBackup field
	//    should be populated.
	// 3. The identifer is a string of the format
	//    'projects/{project-id}/backups/{backup-uid}'. This is used to restore
	//    project level backups, and the Backup field should be populated.
	if backupRunID, err := strconv.ParseInt(backupID, 10, 64); err == nil {
		// If backup_id is a BackupRun ID, it is expected that sourceProject
		// and source_instance are also provided.
		sourceProject, ok := paramsMap["source_project"].(string)
		if !ok {
			return nil, fmt.Errorf("error casting 'source_project' parameter: %v", paramsMap["source_project"])
		}
		sourceInstance, ok := paramsMap["source_instance"].(string)
		if !ok {
			return nil, fmt.Errorf("error casting 'source_instance' parameter: %v", paramsMap["source_instance"])
		}
		request.RestoreBackupContext = &sqladmin.RestoreBackupContext{
			Project:     sourceProject,
			InstanceId:  sourceInstance,
			BackupRunId: backupRunID,
		}
	} else if isBackupDR(backupID) {
		request.BackupdrBackup = backupID
	} else {
		request.Backup = backupID
	}

	service, err := t.Source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, err
	}

	resp, err := service.Instances.RestoreBackup(targetProject, targetInstance, request).Do()
	if err != nil {
		return nil, fmt.Errorf("error restoring backup: %w", err)
	}

	return resp, nil
}

// ParseParams parses the parameters for the tool.
func (t Tool) ParseParams(data map[string]any, claims map[string]map[string]any) (parameters.ParamValues, error) {
	return parameters.ParseParams(t.AllParams, data, claims)
}

// Manifest returns the tool's manifest.
func (t Tool) Manifest() tools.Manifest {
	return t.manifest
}

// McpManifest returns the tool's MCP manifest.
func (t Tool) McpManifest() tools.McpManifest {
	return t.mcpManifest
}

// Authorized checks if the tool is authorized.
func (t Tool) Authorized(verifiedAuthServices []string) bool {
	return true
}

func (t Tool) RequiresClientAuthorization(resourceMgr tools.SourceProvider) bool {
	return t.Source.UseClientAuthorization()
}

func (t Tool) GetAuthTokenHeaderName() string {
	return "Authorization"
}

func isBackupDR(backupID string) bool {
	return backupDRRegex.MatchString(backupID)
}
