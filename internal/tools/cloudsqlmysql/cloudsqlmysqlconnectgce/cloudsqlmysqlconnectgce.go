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

// Package cloudsqlmysqlconnectgce provides a tool that connects a Cloud SQL
// for MySQL instance to a Google Compute Engine VM: it validates network
// reachability, recommends a connection method, and emits ready-to-paste
// setup steps and an optional language-specific code snippet.
package cloudsqlmysqlconnectgce

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/googleapis/mcp-toolbox/internal/util/cloudsqlconnect"
	"github.com/googleapis/mcp-toolbox/internal/util/parameters"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1"
)

const resourceType string = "cloud-sql-mysql-connect-gce"

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
	GetService(context.Context, string) (*sqladmin.Service, error)
	UseClientAuthorization() bool
}

// Config defines the configuration for the cloud-sql-mysql-connect-gce tool.
type Config struct {
	tools.ConfigBase `yaml:",inline"`
	Type             string                 `yaml:"type" validate:"required"`
	Source           string                 `yaml:"source" validate:"required"`
	Annotations      *tools.ToolAnnotations `yaml:"annotations,omitempty"`
}

var _ tools.ToolConfig = Config{}

// ToolConfigType returns the type of the tool.
func (cfg Config) ToolConfigType() string { return resourceType }

// Initialize initializes the tool from the configuration.
func (cfg Config) Initialize(context.Context) (tools.Tool, error) {
	if cfg.Description == "" {
		cfg.Description = "Helps connect a Cloud SQL MySQL instance to a GCE VM. " +
			"Validates network connectivity, recommends the best connection method " +
			"(Auth Proxy, Connector, or Direct Private IP), and provides setup " +
			"instructions and optional code snippets."
	}
	allParameters := buildParams()
	return Tool{
		BaseTool: tools.NewBaseTool(
			cfg,
			tools.GetAnnotationsOrDefault(cfg.Annotations, tools.NewReadOnlyAnnotations),
			tools.Manifest{Description: cfg.Description, Parameters: allParameters.Manifest(), AuthRequired: cfg.AuthRequired},
			allParameters,
		),
	}, nil
}

func buildParams() parameters.Parameters {
	return parameters.Parameters{
		parameters.NewStringParameter(
			"instance_connection_name",
			"Cloud SQL instance connection name in the format: project:region:instance",
		),
		parameters.NewStringParameter(
			"vm_name",
			"Name of the GCE VM instance to connect from",
		),
		parameters.NewStringParameter(
			"vm_zone",
			"Zone of the VM (optional - will auto-discover if not provided)",
			parameters.WithStringDefault(""),
		),
		parameters.NewStringParameter(
			"database_name",
			"Database name to connect to (defaults to 'mysql')",
			parameters.WithStringDefault("mysql"),
		),
		parameters.NewStringParameter(
			"language",
			"Programming language for code snippet generation: python, nodejs, java, go (optional)",
			parameters.WithStringDefault(""),
		),
	}
}

// Tool represents the cloud-sql-mysql-connect-gce tool.
type Tool struct {
	tools.BaseTool[Config]
}

func (t Tool) GetSourceName() string { return t.Cfg.Source }

func (t Tool) ToConfig() tools.ToolConfig { return t.Cfg }

func (t Tool) ValidateSource(source sources.Source) error {
	_, ok := source.(compatibleSource)
	if !ok {
		return fmt.Errorf("invalid source for %q tool: source %q is not a compatible type", t.Cfg.Type, t.Cfg.Source)
	}
	return nil
}

// Invoke validates connectivity between a Cloud SQL MySQL instance and a
// GCE VM, then returns connection recommendations.
func (t Tool) Invoke(ctx context.Context, s sources.Source, params parameters.ParamValues, accessToken tools.AccessToken) (any, util.ToolboxError) {
	source, ok := s.(compatibleSource)
	if !ok {
		return nil, util.NewClientServerError("source used is not compatible with the tool", http.StatusInternalServerError, nil)
	}

	sqlService, err := source.GetService(ctx, string(accessToken))
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}

	computeService, err := getComputeService(ctx)
	if err != nil {
		return nil, util.NewClientServerError("failed to initialize Compute Engine service", http.StatusInternalServerError, err)
	}

	paramsMap := params.AsMap()

	connName, ok := paramsMap["instance_connection_name"].(string)
	if !ok || connName == "" {
		return nil, util.NewAgentError("missing or empty 'instance_connection_name' parameter", nil)
	}

	vmName, ok := paramsMap["vm_name"].(string)
	if !ok || vmName == "" {
		return nil, util.NewAgentError("missing or empty 'vm_name' parameter", nil)
	}
	if err := cloudsqlconnect.ValidateGCEResourceName(vmName, "vm_name"); err != nil {
		return nil, util.NewAgentError(err.Error(), err)
	}

	vmZone, _ := paramsMap["vm_zone"].(string)
	if vmZone != "" {
		if err := cloudsqlconnect.ValidateGCEResourceName(vmZone, "vm_zone"); err != nil {
			return nil, util.NewAgentError(err.Error(), err)
		}
	}

	dbName, _ := paramsMap["database_name"].(string)
	if dbName == "" {
		dbName = cloudsqlconnect.DefaultDatabaseName(cloudsqlconnect.MySQL)
	}
	if err := cloudsqlconnect.ValidateDatabaseName(dbName); err != nil {
		return nil, util.NewAgentError(err.Error(), err)
	}

	language, _ := paramsMap["language"].(string)
	if err := cloudsqlconnect.ValidateLanguage(language); err != nil {
		return nil, util.NewAgentError(err.Error(), err)
	}

	project, region, instanceName, err := cloudsqlconnect.ValidateInstanceConnectionName(connName)
	if err != nil {
		return nil, util.NewAgentError(err.Error(), err)
	}

	sqlInstance, err := sqlService.Instances.Get(project, instanceName).Context(ctx).Do()
	if err != nil {
		return nil, util.ProcessGcpError(err)
	}
	sqlInfo := extractSQLInfo(sqlInstance)

	if err := cloudsqlconnect.AssertEngine(cloudsqlconnect.MySQL, instanceName, sqlInfo.DatabaseVersion); err != nil {
		return nil, util.NewAgentError(err.Error(), err)
	}

	var vmInstance *compute.Instance
	if vmZone == "" {
		vmInstance, vmZone, err = findVM(ctx, computeService, project, vmName)
		if err != nil {
			return nil, util.ProcessGcpError(err)
		}
	} else {
		vmInstance, err = computeService.Instances.Get(project, vmZone, vmName).Context(ctx).Do()
		if err != nil {
			return nil, util.ProcessGcpError(err)
		}
	}
	vmInfo := extractVMInfo(vmInstance, vmZone)

	validation := cloudsqlconnect.ValidateGCEConnection(sqlInfo, vmInfo)
	sameVPC := cloudsqlconnect.IsSameVPC(sqlInfo.VPCNetwork, vmInfo.VPCNetwork)
	primary, alternatives := cloudsqlconnect.GetGCERecommendations(sqlInfo, vmInfo, sameVPC)

	port := cloudsqlconnect.GetDatabasePort(cloudsqlconnect.MySQL)
	setupSteps := cloudsqlconnect.GenerateGCESetupSteps(primary.Method, connName, port, sqlInfo.PrivateIPAddress, vmInfo.ServiceAccount)
	envConfig := cloudsqlconnect.GenerateEnvironmentConfig(
		primary.Method, cloudsqlconnect.ComputeGCE, connName, port,
		sqlInfo.PrivateIPAddress, dbName, project,
	)
	connStrings := cloudsqlconnect.BuildConnectionStrings(primary.Method, cloudsqlconnect.MySQL, sqlInfo, dbName, connName)

	result := &cloudsqlconnect.ConnectResult{
		InstanceConnectionName: connName,
		Project:                project,
		Region:                 region,
		DatabaseType:           cloudsqlconnect.MySQL,
		DatabaseVersion:        sqlInfo.DatabaseVersion,
		ComputeType:            cloudsqlconnect.ComputeGCE,
		ComputeResource:        vmName,
		ComputeLocation:        vmZone,
		Validation:             *validation,
		RecommendedMethod:      primary,
		AlternativeMethods:     alternatives,
		ConnectionStrings:      connStrings,
		EnvironmentConfig:      envConfig,
		SetupSteps:             setupSteps,
		AvailableLanguages:     cloudsqlconnect.AvailableLanguages,
		RequiredIAMRoles:       []string{"roles/cloudsql.client"},
		RequiredAPIs:           []string{"sqladmin.googleapis.com", "compute.googleapis.com"},
	}

	if language != "" {
		lang := cloudsqlconnect.Language(strings.ToLower(language))
		result.CodeSnippet = cloudsqlconnect.GenerateCodeSnippet(
			lang, primary.Method, cloudsqlconnect.MySQL,
			connName, dbName, port, sqlInfo.PrivateIPAddress,
		)
	}

	return result, nil
}

func (t Tool) RequiresClientAuthorization(source sources.Source) (bool, error) {
	s, ok := source.(compatibleSource)
	if !ok {
		return false, fmt.Errorf("invalid source for %q tool: source %q is not a compatible type", t.Cfg.Type, t.Cfg.Source)
	}
	return s.UseClientAuthorization(), nil
}

// computeService is lazily initialized on first use and shared across
// invocations: *compute.Service is goroutine-safe and rebuilding it per
// call would re-pay token-source + service-discovery costs.
var (
	computeOnce    sync.Once
	computeService *compute.Service
	computeErr     error
)

func getComputeService(ctx context.Context) (*compute.Service, error) {
	computeOnce.Do(func() {
		computeService, computeErr = compute.NewService(ctx, option.WithScopes(compute.ComputeReadonlyScope))
	})
	return computeService, computeErr
}

func extractSQLInfo(inst *sqladmin.DatabaseInstance) *cloudsqlconnect.CloudSQLInstanceInfo {
	info := &cloudsqlconnect.CloudSQLInstanceInfo{
		Name:            inst.Name,
		Project:         inst.Project,
		Region:          inst.Region,
		ConnectionName:  inst.ConnectionName,
		DatabaseVersion: inst.DatabaseVersion,
		DatabaseType:    cloudsqlconnect.ParseDatabaseType(inst.DatabaseVersion),
	}
	for _, ip := range inst.IpAddresses {
		switch ip.Type {
		case "PRIMARY":
			info.PublicIPAddress = ip.IpAddress
			info.PublicIPEnabled = true
		case "PRIVATE":
			info.PrivateIPAddress = ip.IpAddress
			info.PrivateIPEnabled = true
		}
	}
	if inst.Settings != nil && inst.Settings.IpConfiguration != nil {
		info.VPCNetwork = inst.Settings.IpConfiguration.PrivateNetwork
		info.RequireSSL = inst.Settings.IpConfiguration.RequireSsl
	}
	return info
}

func extractVMInfo(inst *compute.Instance, zone string) *cloudsqlconnect.GCEInstanceInfo {
	info := &cloudsqlconnect.GCEInstanceInfo{Name: inst.Name, Zone: zone}
	if len(inst.NetworkInterfaces) > 0 {
		ni := inst.NetworkInterfaces[0]
		info.InternalIP = ni.NetworkIP
		info.VPCNetwork = cloudsqlconnect.ExtractNetworkName(ni.Network)
		info.Subnetwork = cloudsqlconnect.ExtractNetworkName(ni.Subnetwork)
		for _, ac := range ni.AccessConfigs {
			if ac.NatIP != "" {
				info.ExternalIP = ac.NatIP
				info.HasExternalIP = true
				break
			}
		}
	}
	if len(inst.ServiceAccounts) > 0 {
		info.ServiceAccount = inst.ServiceAccounts[0].Email
	}
	return info
}

// findVM resolves a VM by name across all zones in a project.
func findVM(ctx context.Context, service *compute.Service, project, vmName string) (*compute.Instance, string, error) {
	var foundInstances []*compute.Instance
	var foundZones []string

	const stopPaging stringErr = "stop-paging"

	req := service.Instances.AggregatedList(project).Filter(fmt.Sprintf("name eq %q", vmName))
	err := req.Pages(ctx, func(page *compute.InstanceAggregatedList) error {
		for zone, list := range page.Items {
			for _, instance := range list.Instances {
				if instance.Name != vmName {
					continue
				}
				foundInstances = append(foundInstances, instance)
				foundZones = append(foundZones, cloudsqlconnect.ExtractNetworkName(zone))
				if len(foundInstances) > 1 {
					return stopPaging
				}
			}
		}
		return nil
	})
	if err != nil && err != stopPaging {
		return nil, "", fmt.Errorf("failed to search for VM: %w", err)
	}

	switch len(foundInstances) {
	case 0:
		return nil, "", fmt.Errorf("VM %q not found in project %q", vmName, project)
	case 1:
		return foundInstances[0], foundZones[0], nil
	default:
		return nil, "", fmt.Errorf("multiple VMs named %q found in zones: %v - please specify vm_zone parameter", vmName, foundZones)
	}
}

type stringErr string

func (e stringErr) Error() string { return string(e) }
