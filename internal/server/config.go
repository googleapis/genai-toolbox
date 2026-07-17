// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	yaml "github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"
	"github.com/goccy/go-yaml/parser"
	"github.com/goccy/go-yaml/token"
	"github.com/googleapis/mcp-toolbox/internal/auth"
	"github.com/googleapis/mcp-toolbox/internal/auth/generic"
	"github.com/googleapis/mcp-toolbox/internal/auth/google"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels"
	"github.com/googleapis/mcp-toolbox/internal/embeddingmodels/gemini"
	"github.com/googleapis/mcp-toolbox/internal/prompts"
	"github.com/googleapis/mcp-toolbox/internal/resources"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"github.com/yosida95/uritemplate/v3"
)

type ServerConfig struct {
	// Server version
	Version string
	// Address is the address of the interface the server will listen on.
	Address string
	// Port is the port the server will listen on.
	Port int
	// CertFile is the path to tls certificate file
	CertFile string
	// KeyFile is the path to TLS key file
	KeyFile string
	// SourceConfigs defines what sources of data are available for tools.
	SourceConfigs SourceConfigs
	// AuthServiceConfigs defines what sources of authentication are available for tools.
	AuthServiceConfigs AuthServiceConfigs
	// EmbeddingModelConfigs defines a models used to embed parameters.
	EmbeddingModelConfigs EmbeddingModelConfigs
	// ToolConfigs defines what tools are available.
	ToolConfigs ToolConfigs
	// ToolsetConfigs defines what tools are available.
	ToolsetConfigs ToolsetConfigs
	// PromptConfigs defines what prompts are available
	PromptConfigs PromptConfigs
	// PromptsetConfigs defines what prompts are available
	PromptsetConfigs PromptsetConfigs
	// ResourceConfigs defines what resources are available.
	ResourceConfigs ResourceConfigs
	// ResourceTemplateConfigs defines what resource templates are available.
	ResourceTemplateConfigs ResourceTemplateConfigs
	// IgnoreUnknownTools logs warnings and skips unknown/unsupported tool types instead of failing to start.
	IgnoreUnknownTools bool
	// LoggingFormat defines whether structured loggings are used.
	LoggingFormat logFormat
	// LogLevel defines the levels to log.
	LogLevel StringLevel
	// TelemetryGCP defines whether GCP exporter is used.
	TelemetryGCP bool
	// TelemetryOTLP defines OTLP collector url for telemetry exports.
	TelemetryOTLP string
	// TelemetryGCPProject defines the Google Cloud project ID to use for telemetry exports.
	TelemetryGCPProject string
	// TelemetryServiceName defines the value of service.name resource attribute.
	TelemetryServiceName string
	// SQLCommenter enables prepending SQLCommenter-format comments to SQL statements.
	SQLCommenter bool
	// Stdio indicates if Toolbox is listening via MCP stdio.
	Stdio bool
	// DisableReload indicates if the user has disabled dynamic reloading for Toolbox.
	DisableReload bool
	// UI indicates if Toolbox UI endpoints (/ui) are available.
	UI bool
	// EnableAPI indicates if the /api endpoint is enabled.
	EnableAPI bool
	// ToolboxUrl specifies the URL to advertise in the MCP PRM file as the resource field.
	ToolboxUrl string
	// McpPrmFile specifies the path to a manual Protected Resource Metadata (PRM) JSON file. If provided, overrides auto-generation.
	McpPrmFile string
	// Specifies a list of origins permitted to access this server.
	AllowedOrigins []string
	// Specifies a list of hosts permitted to access this server.
	AllowedHosts []string
	// UserAgentMetadata specifies additional metadata to append to the User-Agent string.
	UserAgentMetadata []string
	// PollInterval sets the polling frequency for configuration file updates.
	PollInterval int
	// HttpMaxRequestBytes caps MCP HTTP request bodies. Zero uses the default.
	HttpMaxRequestBytes int64
	// EnableDraftSpecs allow users to opt-in and test upcoming draft MCP specs.
	EnableDraftSpecs bool
}

type logFormat string

// String is used by both fmt.Print and by Cobra in help text
func (f *logFormat) String() string {
	if string(*f) != "" {
		return strings.ToLower(string(*f))
	}
	return "standard"
}

// validate logging format flag
func (f *logFormat) Set(v string) error {
	switch strings.ToLower(v) {
	case "standard", "json":
		*f = logFormat(v)
		return nil
	default:
		return fmt.Errorf(`log format must be one of "standard", or "json"`)
	}
}

// Type is used in Cobra help text
func (f *logFormat) Type() string {
	return "logFormat"
}

type StringLevel string

// String is used by both fmt.Print and by Cobra in help text
func (s *StringLevel) String() string {
	if string(*s) != "" {
		return strings.ToLower(string(*s))
	}
	return "info"
}

// validate log level flag
func (s *StringLevel) Set(v string) error {
	switch strings.ToLower(v) {
	case "debug", "info", "warn", "error":
		*s = StringLevel(v)
		return nil
	default:
		return fmt.Errorf(`log level must be one of "debug", "info", "warn", or "error"`)
	}
}

// Type is used in Cobra help text
func (s *StringLevel) Type() string {
	return "stringLevel"
}

type SourceConfigs map[string]sources.SourceConfig
type AuthServiceConfigs map[string]auth.AuthServiceConfig
type EmbeddingModelConfigs map[string]embeddingmodels.EmbeddingModelConfig
type ToolConfigs map[string]tools.ToolConfig
type ToolsetConfigs map[string]tools.ToolsetConfig
type PromptConfigs map[string]prompts.PromptConfig
type PromptsetConfigs map[string]prompts.PromptsetConfig
type ResourceConfigs map[string]resources.ResourceConfig
type ResourceTemplateConfigs map[string]resources.ResourceTemplateConfig

func UnmarshalPrimitiveConfig(ctx context.Context, raw []byte) (SourceConfigs, AuthServiceConfigs, EmbeddingModelConfigs, ToolConfigs, ToolsetConfigs, PromptConfigs, ResourceConfigs, ResourceTemplateConfigs, error) {
	// prepare configs map
	var sourceConfigs SourceConfigs
	var authServiceConfigs AuthServiceConfigs
	var embeddingModelConfigs EmbeddingModelConfigs
	var toolConfigs ToolConfigs
	var toolsetConfigs ToolsetConfigs
	var promptConfigs PromptConfigs
	var resourceConfigs ResourceConfigs
	var resourceTemplateConfigs ResourceTemplateConfigs
	// promptset configs is not yet supported
	seenResourceURIs := make(map[string]string)

	file, err := parser.ParseBytes(raw, 0)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to parse YAML: %s", yaml.FormatError(err, false, false))
	}

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	for index, doc := range file.Docs {
		if doc == nil || doc.Body == nil {
			continue
		}
		docIndex := index + 1
		var resource map[string]any
		if err := decoder.DecodeFromNodeContext(ctx, doc.Body, &resource); err != nil {
			if len(file.Docs) > 1 {
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: %s", docIndex, yaml.FormatError(err, false, false))
			}
			return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("unable to decode YAML document: %s", yaml.FormatError(err, false, false))
		}
		var kind, name string
		var ok bool
		if kind, ok = resource["kind"].(string); !ok {
			if len(file.Docs) > 1 {
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("%s missing 'kind' field or it is not a string", formatDocLocation(docIndex, keyToken(doc.Body, "kind"), doc.Body))
			}
			return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("missing 'kind' field or it is not a string: %v", resource)
		}
		if name, ok = resource["name"].(string); !ok {
			if len(file.Docs) > 1 {
				fallbackToken := keyToken(doc.Body, "name")
				if fallbackToken == nil {
					fallbackToken = keyToken(doc.Body, "kind")
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("%s missing 'name' field or it is not a string", formatDocLocation(docIndex, fallbackToken, doc.Body))
			}
			return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("missing 'name' field or it is not a string")
		}
		// remove 'kind' from map for strict unmarshaling
		delete(resource, "kind")

		switch kind {
		case "source":
			c, err := UnmarshalYAMLSourceConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if sourceConfigs == nil {
				sourceConfigs = make(SourceConfigs)
			}
			sourceConfigs[name] = c
		case "authService":
			c, err := UnmarshalYAMLAuthServiceConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if authServiceConfigs == nil {
				authServiceConfigs = make(AuthServiceConfigs)
			}
			authServiceConfigs[name] = c
		case "tool":
			c, err := UnmarshalYAMLToolConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if c == nil {
				continue
			}
			if toolConfigs == nil {
				toolConfigs = make(ToolConfigs)
			}
			toolConfigs[name] = c
		case "toolset":
			c, err := UnmarshalYAMLToolsetConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if toolsetConfigs == nil {
				toolsetConfigs = make(ToolsetConfigs)
			}
			toolsetConfigs[name] = c
		case "embeddingModel":
			c, err := UnmarshalYAMLEmbeddingModelConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if embeddingModelConfigs == nil {
				embeddingModelConfigs = make(EmbeddingModelConfigs)
			}
			embeddingModelConfigs[name] = c
		case "prompt":
			c, err := UnmarshalYAMLPromptConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if promptConfigs == nil {
				promptConfigs = make(PromptConfigs)
			}
			promptConfigs[name] = c
		case "resource":
			c, err := UnmarshalYAMLResourceConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if resourceConfigs == nil {
				resourceConfigs = make(ResourceConfigs)
			}
			if _, exists := resourceConfigs[name]; exists {
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("duplicate resource name: %q", name)
			}
			if c.GetURI() != "" {
				if existingName, exists := seenResourceURIs[c.GetURI()]; exists {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("duplicate resource URI %q used by resources %q and %q", c.GetURI(), existingName, name)
				}
				seenResourceURIs[c.GetURI()] = name
			}
			resourceConfigs[name] = c
		case "resourceTemplate":
			c, err := UnmarshalYAMLResourceTemplateConfig(ctx, name, resource)
			if err != nil {
				if len(file.Docs) > 1 {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("document %d: error unmarshaling %s %q: %w", docIndex, kind, name, err)
				}
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error unmarshaling %s: %w", kind, err)
			}
			if resourceTemplateConfigs == nil {
				resourceTemplateConfigs = make(ResourceTemplateConfigs)
			}
			if _, exists := resourceTemplateConfigs[name]; exists {
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("duplicate resourceTemplate name: %q", name)
			}
			if c.GetURITemplate() != "" {
				if existingName, exists := seenResourceURIs[c.GetURITemplate()]; exists {
					return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("duplicate resource URI %q used by resources %q and %q", c.GetURITemplate(), existingName, name)
				}
				seenResourceURIs[c.GetURITemplate()] = name
			}
			resourceTemplateConfigs[name] = c
		default:
			if len(file.Docs) > 1 {
				return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("%s invalid kind %q", formatDocLocation(docIndex, keyToken(doc.Body, "kind"), doc.Body), kind)
			}
			return nil, nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("invalid kind %s", kind)
		}
	}
	return sourceConfigs, authServiceConfigs, embeddingModelConfigs, toolConfigs, toolsetConfigs, promptConfigs, resourceConfigs, resourceTemplateConfigs, nil
}

func UnmarshalYAMLSourceConfig(ctx context.Context, name string, r map[string]any) (sources.SourceConfig, error) {
	resourceType, ok := r["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'type' field or it is not a string")
	}
	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %w", err)
	}
	sourceConfig, err := sources.DecodeConfig(ctx, resourceType, name, dec)
	if err != nil {
		return nil, err
	}
	return sourceConfig, nil
}

func UnmarshalYAMLAuthServiceConfig(ctx context.Context, name string, r map[string]any) (auth.AuthServiceConfig, error) {
	resourceType, ok := r["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'type' field or it is not a string")
	}
	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}

	switch resourceType {
	case google.AuthServiceType:
		actual := google.Config{Name: name}
		if err := dec.DecodeContext(ctx, &actual); err != nil {
			return nil, fmt.Errorf("unable to parse as %s: %w", name, err)
		}
		if !actual.McpEnabled {
			if actual.Audience != "" {
				return nil, fmt.Errorf("`audience` is not allowed when `mcpEnabled` is false")
			}
			if len(actual.ScopesRequired) > 0 {
				return nil, fmt.Errorf("`scopesRequired` is not allowed when `mcpEnabled` is false")
			}
		}
		return actual, nil
	case generic.AuthServiceType:
		actual := generic.Config{Name: name}
		if err := dec.DecodeContext(ctx, &actual); err != nil {
			return nil, fmt.Errorf("unable to parse as %s: %w", name, err)
		}
		if !actual.McpEnabled {
			if actual.IntrospectionEndpoint != "" {
				return nil, fmt.Errorf("`introspectionEndpoint` is not allowed when `mcpEnabled` is false")
			}
			if actual.IntrospectionMethod != "" {
				return nil, fmt.Errorf("`introspectionMethod` is not allowed when `mcpEnabled` is false")
			}
			if actual.IntrospectionParamName != "" {
				return nil, fmt.Errorf("`introspectionParamName` is not allowed when `mcpEnabled` is false")
			}
			if len(actual.ScopesRequired) > 0 {
				return nil, fmt.Errorf("`scopesRequired` is not allowed when `mcpEnabled` is false")
			}
		}
		return actual, nil
	default:
		return nil, fmt.Errorf("%s is not a valid type of auth service", resourceType)
	}
}

func UnmarshalYAMLEmbeddingModelConfig(ctx context.Context, name string, r map[string]any) (embeddingmodels.EmbeddingModelConfig, error) {
	resourceType, ok := r["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'type' field or it is not a string")
	}
	if resourceType != gemini.EmbeddingModelType {
		return nil, fmt.Errorf("%s is not a valid type of embedding model", resourceType)
	}
	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}
	actual := gemini.Config{Name: name}
	if err := dec.DecodeContext(ctx, &actual); err != nil {
		return nil, fmt.Errorf("unable to parse as %q: %w", name, err)
	}
	return actual, nil
}

func UnmarshalYAMLToolConfig(ctx context.Context, name string, r map[string]any) (tools.ToolConfig, error) {
	resourceType, ok := r["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing 'type' field or it is not a string")
	}
	// `authRequired` and `useClientOAuth` cannot be specified together
	if r["authRequired"] != nil && r["useClientOAuth"] == true {
		return nil, fmt.Errorf("`authRequired` and `useClientOAuth` are mutually exclusive. Choose only one authentication method")
	}
	// Make `authRequired` an empty list instead of nil for Tool manifest
	if r["authRequired"] == nil {
		r["authRequired"] = []string{}
	}

	// Parse scopesRequired if present
	if rawScopes, ok := r["scopesRequired"]; ok {
		if scopesList, ok := rawScopes.([]any); ok {
			var scopes []string
			for _, s := range scopesList {
				if str, ok := s.(string); ok {
					scopes = append(scopes, str)
				}
			}
			r["scopesRequired"] = scopes
		} else {
			return nil, fmt.Errorf("scopesRequired must be a list of strings")
		}
	}

	// validify parameter references
	if rawParams, ok := r["parameters"]; ok {
		if paramsList, ok := rawParams.([]any); ok {
			// Turn params into a map
			validParamNames := make(map[string]bool)
			for _, rawP := range paramsList {
				if pMap, ok := rawP.(map[string]any); ok {
					if pName, ok := pMap["name"].(string); ok && pName != "" {
						validParamNames[pName] = true
					}
				}
			}

			// Validate references
			for i, rawP := range paramsList {
				pMap, ok := rawP.(map[string]any)
				if !ok {
					continue
				}

				pName, _ := pMap["name"].(string)
				refName, _ := pMap["valueFromParam"].(string)

				if refName != "" {
					// Check if the referenced parameter exists
					if !validParamNames[refName] {
						return nil, fmt.Errorf("tool %q config error: parameter %q (index %d) references '%q' in the 'valueFromParam' field, which is not a defined parameter", name, pName, i, refName)
					}

					// Check for self-reference
					if refName == pName {
						return nil, fmt.Errorf("tool %q config error: parameter %q cannot copy value from itself", name, pName)
					}
				}
			}
		}
	}

	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}
	toolCfg, err := tools.DecodeConfig(ctx, resourceType, name, dec)
	if err != nil {
		if errors.Is(err, tools.ErrUnknownToolType) && util.IgnoreUnknownToolsFromContext(ctx) {
			l, logErr := util.LoggerFromContext(ctx)
			if logErr == nil {
				l.WarnContext(ctx, fmt.Sprintf("Skipping unknown tool type %q for tool %q", resourceType, name))
			}
			return nil, nil
		}
		return nil, err
	}
	return toolCfg, nil
}

func UnmarshalYAMLToolsetConfig(ctx context.Context, name string, r map[string]any) (tools.ToolsetConfig, error) {
	var toolsetConfig tools.ToolsetConfig
	toolList, ok := r["tools"].([]any)
	if !ok {
		return toolsetConfig, fmt.Errorf("tools is missing or not a list of strings: %v", r)
	}
	justTools := map[string]any{"tools": toolList}
	dec, err := util.NewStrictDecoder(justTools)
	if err != nil {
		return toolsetConfig, fmt.Errorf("error creating decoder: %s", err)
	}
	var raw map[string][]string
	if err := dec.DecodeContext(ctx, &raw); err != nil {
		return toolsetConfig, fmt.Errorf("unable to unmarshal tools: %s", err)
	}
	return tools.ToolsetConfig{Name: name, ToolNames: raw["tools"]}, nil
}

func UnmarshalYAMLPromptConfig(ctx context.Context, name string, r map[string]any) (prompts.PromptConfig, error) {
	// Look for the 'type' field. If it's not present, typeStr will be an
	// empty string, which prompts.DecodeConfig will correctly default to "custom".
	var resourceType string
	if typeVal, ok := r["type"]; ok {
		var isString bool
		resourceType, isString = typeVal.(string)
		if !isString {
			return nil, fmt.Errorf("invalid 'type' field for prompt %q (must be a string)", name)
		}
	}
	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}

	// Use the central registry to decode the prompt based on its type.
	promptCfg, err := prompts.DecodeConfig(ctx, resourceType, name, dec)
	if err != nil {
		return nil, err
	}
	return promptCfg, nil
}

func UnmarshalYAMLResourceConfig(ctx context.Context, name string, r map[string]any) (resources.ResourceConfig, error) {
	var resourceType string
	if typeVal, ok := r["type"]; ok {
		var isString bool
		resourceType, isString = typeVal.(string)
		if !isString {
			return nil, fmt.Errorf("invalid 'type' field for resource %q (must be a string)", name)
		}
		resourceType = strings.ToLower(resourceType)
	} else {
		return nil, fmt.Errorf("missing required 'type' field for resource %q", name)
	}

	if uriVal, ok := r["uri"]; ok {
		if uriStr, isString := uriVal.(string); !isString {
			return nil, fmt.Errorf("invalid 'uri' field for resource %q (must be a string)", name)
		} else {
			parsed, err := url.ParseRequestURI(uriStr)
			if err != nil || parsed.Scheme == "" {
				return nil, fmt.Errorf("invalid 'uri' field for resource %q: must be a valid RFC-compliant absolute URI with a scheme", name)
			}

			// Normalize scheme and host to lowercase for consistent comparison and usage
			parsed.Scheme = strings.ToLower(parsed.Scheme)
			parsed.Host = strings.ToLower(parsed.Host)

			// Scheme whitelisting
			if resourceType == "file" && parsed.Scheme != "file" {
				return nil, fmt.Errorf("invalid scheme for file resource %q: must be 'file'", name)
			}

			// Update the map with the normalized URI so that unmarshaling and
			// collision detection see the normalized form (e.g. lowercase scheme and host)
			r["uri"] = parsed.String()
		}
	} else {
		switch resourceType {
		case "file":
			r["uri"] = fmt.Sprintf("file://%s", name)
		case "text":
			r["uri"] = fmt.Sprintf("text://%s", name)
		default:
			return nil, fmt.Errorf("missing required 'uri' field for resource %q", name)
		}
	}
	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}

	resCfg, err := resources.DecodeConfig(ctx, resourceType, name, dec)
	if err != nil {
		return nil, err
	}
	return resCfg, nil
}

// Tools naming validation is added in the MCP v2025-11-25, but we'll be
// implementing it across Toolbox
// Tool names SHOULD be between 1 and 128 characters in length (inclusive).
// Tool names SHOULD be considered case-sensitive.
// The following SHOULD be the only allowed characters: uppercase and lowercase ASCII letters (A-Z, a-z), digits (0-9), underscore (_), hyphen (-), and dot (.)
// Tool names SHOULD NOT contain spaces, commas, or other special characters.
// Tool names SHOULD be unique within a server.
func NameValidation(name string) error {
	strLen := len(name)
	if strLen < 1 || strLen > 128 {
		return fmt.Errorf("resource name SHOULD be between 1 and 128 characters in length (inclusive)")
	}
	validChars := regexp.MustCompile("^[a-zA-Z0-9_.-]+$")
	isValid := validChars.MatchString(name)
	if !isValid {
		return fmt.Errorf("invalid character for resource name; only uppercase and lowercase ASCII letters (A-Z, a-z), digits (0-9), underscore (_), hyphen (-), and dot (.) is allowed")
	}
	return nil
}

func formatDocLocation(docIndex int, keyToken *token.Token, body ast.Node) string {
	line, column := 0, 0
	if keyToken != nil {
		line = keyToken.Position.Line
		column = keyToken.Position.Column
	} else if body != nil && body.GetToken() != nil {
		line = body.GetToken().Position.Line
		column = body.GetToken().Position.Column
	}
	if line > 0 && column > 0 {
		return fmt.Sprintf("document %d (line %d, column %d):", docIndex, line, column)
	}
	return fmt.Sprintf("document %d:", docIndex)
}

func keyToken(body ast.Node, key string) *token.Token {
	if body == nil {
		return nil
	}
	mapping, ok := body.(*ast.MappingNode)
	if !ok {
		return nil
	}
	iter := mapping.MapRange()
	for iter.Next() {
		mapKey := iter.Key()
		if mapKey == nil {
			continue
		}
		token := mapKey.GetToken()
		if token != nil && token.Value == key {
			return token
		}
	}
	return nil
}

func UnmarshalYAMLResourceTemplateConfig(ctx context.Context, name string, r map[string]any) (resources.ResourceTemplateConfig, error) {
	resourceType := "file"
	if typeVal, ok := r["type"]; ok {
		var isString bool
		resourceType, isString = typeVal.(string)
		if !isString {
			return nil, fmt.Errorf("invalid 'type' field for resourceTemplate %q (must be a string)", name)
		}
		resourceType = strings.ToLower(resourceType)
	}

	if resourceType != "file" {
		return nil, fmt.Errorf("invalid 'type' field for resourceTemplate %q: only 'file' is supported", name)
	}

	if uriVal, ok := r["uriTemplate"]; ok {
		if uriStr, isString := uriVal.(string); !isString {
			return nil, fmt.Errorf("invalid 'uriTemplate' field for resourceTemplate %q (must be a string)", name)
		} else {
			// Validate RFC 6570 compliance
			tmpl, err := uritemplate.New(uriStr)
			if err != nil {
				return nil, fmt.Errorf("invalid RFC 6570 uriTemplate %q: %w", name, err)
			}

			// Enforce only 'path' is allowed as a variable
			for _, varName := range tmpl.Varnames() {
				if varName != "path" {
					return nil, fmt.Errorf("invalid uriTemplate %q: only the 'path' variable is supported (found %q)", name, varName)
				}
			}

			// Strip all {variables} to validate the base URI structure natively
			re := regexp.MustCompile(`\{[^}]+\}`)
			parseableURI := re.ReplaceAllString(uriStr, "dummy")
			parsed, err := url.ParseRequestURI(parseableURI)
			if err != nil || parsed.Scheme == "" {
				return nil, fmt.Errorf("invalid 'uriTemplate' field for resourceTemplate %q: must be a valid RFC-compliant absolute URI with a scheme", name)
			}

			// Must be a URI starting with file:// per user requirements
			if resourceType == "file" && parsed.Scheme != "file" {
				return nil, fmt.Errorf("invalid scheme for file resource template %q: must be 'file'", name)
			}

			// Update the map with the normalized URI
			r["uriTemplate"] = uriStr
		}
	} else {
		return nil, fmt.Errorf("missing required 'uriTemplate' field for resourceTemplate %q", name)
	}

	dec, err := util.NewStrictDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("error creating decoder: %s", err)
	}

	resCfg, err := resources.DecodeTemplateConfig(ctx, resourceType, name, dec)
	if err != nil {
		return nil, err
	}
	return resCfg, nil
}
