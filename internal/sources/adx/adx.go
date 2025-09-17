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

package adx

import (
	"context"
	"fmt"
	"io"
	"log"
	"reflect"
	"strings"

	"github.com/Azure/azure-kusto-go/kusto"
	"github.com/Azure/azure-kusto-go/kusto/kql"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/sources"
	"go.opentelemetry.io/otel/trace"
)

const SourceKind string = "adx"

// validate interface
var _ sources.SourceConfig = Config{}

// AuthMode represents Azure authentication modes for ADX
type AuthMode string

const (
	// AuthModeDefault uses DefaultAzureCredential (Azure CLI, Environment, etc.)
	AuthModeDefault AuthMode = "default"
	// AuthModeClientSecret uses ClientSecretCredential (tenant/client/secret)
	AuthModeClientSecret AuthMode = "client_secret"
	// AuthModeDelegated uses token-based authentication
	AuthModeDelegated AuthMode = "delegated"
	// AuthModeDeviceCode uses DeviceCodeCredential for interactive login
	AuthModeDeviceCode AuthMode = "device_code"
	// AuthModeManagedIdentity uses ManagedIdentityCredential
	AuthModeManagedIdentity AuthMode = "managed_identity"
	// AuthModeBrowser uses InteractiveBrowserCredential for browser-based login
	AuthModeBrowser AuthMode = "browser"
	
	// Legacy aliases for backward compatibility
	authModeExplicit = "explicit" // maps to client_secret
	authModeDCR      = "dcr"      // maps to device_code
	authModeMI       = "mi"       // maps to managed_identity
)

func init() {
	if !sources.Register(SourceKind, newConfig) {
		panic(fmt.Sprintf("source kind %q already registered", SourceKind))
	}
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil
}

type Config struct {
	Name            string   `yaml:"name" validate:"required"`
	Kind            string   `yaml:"kind" validate:"required"`
	ClusterURI      string   `yaml:"cluster_uri" validate:"required"`
	Database        string   `yaml:"database" validate:"required"`
	AuthMode        AuthMode `yaml:"auth_mode"`
	TenantID        string   `yaml:"tenant_id,omitempty"`
	ClientID        string   `yaml:"client_id,omitempty"`
	ClientSecret    string   `yaml:"client_secret,omitempty"`
	AccessToken     string   `yaml:"access_token,omitempty"`
	ManagedIdentity string   `yaml:"managed_identity,omitempty"` // Client ID for user-assigned managed identity
	RedirectURL     string   `yaml:"redirect_url,omitempty"`     // Redirect URL for browser authentication
}

func (r Config) SourceConfigKind() string {
	return SourceKind
}

func (r Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	// Default auth mode if not specified
	if r.AuthMode == "" {
		r.AuthMode = AuthModeDefault
	}

	// Handle legacy mode mappings
	switch string(r.AuthMode) {
	case authModeExplicit:
		r.AuthMode = AuthModeClientSecret
	case authModeDCR:
		r.AuthMode = AuthModeDeviceCode
	case authModeMI:
		r.AuthMode = AuthModeManagedIdentity
	}

	// Create Kusto connection string based on auth mode
	var kcsb *kusto.ConnectionStringBuilder
	
	switch r.AuthMode {
	case AuthModeDefault:
		// Use Azure CLI authentication
		kcsb = kusto.NewConnectionStringBuilder(r.ClusterURI).WithAzCli()
	case AuthModeClientSecret:
		if r.TenantID == "" || r.ClientID == "" || r.ClientSecret == "" {
			return nil, fmt.Errorf("tenant_id, client_id, and client_secret are required for client_secret auth mode")
		}
		// Use application authentication with client secret
		kcsb = kusto.NewConnectionStringBuilder(r.ClusterURI).WithAadAppKey(r.ClientID, r.ClientSecret, r.TenantID)
	case AuthModeDelegated:
		if r.AccessToken == "" {
			return nil, fmt.Errorf("access_token is required for delegated auth mode")
		}
		// Use user token
		kcsb = kusto.NewConnectionStringBuilder(r.ClusterURI).WithAadAppKey("", r.AccessToken, "")
	case AuthModeDeviceCode:
		if r.ClientID == "" || r.TenantID == "" {
			return nil, fmt.Errorf("client_id and tenant_id are required for device_code auth mode")
		}
		// Use device code credential
		cred, err := azidentity.NewDeviceCodeCredential(&azidentity.DeviceCodeCredentialOptions{
			TenantID: r.TenantID,
			ClientID: r.ClientID,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create device code credential: %w", err)
		}
		kcsb = kusto.NewConnectionStringBuilder(r.ClusterURI).WithTokenCredential(cred)
	case AuthModeBrowser:
		// Use interactive browser credential
		var cred azcore.TokenCredential
		var err error
		
		options := &azidentity.InteractiveBrowserCredentialOptions{}
		if r.TenantID != "" {
			options.TenantID = r.TenantID
		}
		if r.ClientID != "" {
			options.ClientID = r.ClientID
		}
		if r.RedirectURL != "" {
			options.RedirectURL = r.RedirectURL
		}
		
		cred, err = azidentity.NewInteractiveBrowserCredential(options)
		if err != nil {
			return nil, fmt.Errorf("failed to create interactive browser credential: %w", err)
		}
		
		kcsb = kusto.NewConnectionStringBuilder(r.ClusterURI).WithTokenCredential(cred)
	case AuthModeManagedIdentity:
		// Use managed identity authentication
		var cred azcore.TokenCredential
		var err error
		
		if r.ManagedIdentity != "" {
			// User-assigned managed identity with client ID
			cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
				ID: azidentity.ClientID(r.ManagedIdentity),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create user-assigned managed identity credential: %w", err)
			}
		} else {
			// System-assigned managed identity
			cred, err = azidentity.NewManagedIdentityCredential(nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create system-assigned managed identity credential: %w", err)
			}
		}
		
		kcsb = kusto.NewConnectionStringBuilder(r.ClusterURI).WithTokenCredential(cred)
	default:
		return nil, fmt.Errorf("unsupported auth mode: %s", r.AuthMode)
	}
	
	client, err := kusto.New(kcsb)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kusto client: %w", err)
	}

	s := &Source{
		Name:     r.Name,
		Kind:     SourceKind,
		Client:   client,
		Database: r.Database,
	}
	return s, nil
}

var _ sources.Source = &Source{}

type Source struct {
	Name     string `yaml:"name"`
	Kind     string `yaml:"kind"`
	Client   *kusto.Client
	Database string
}

func (s *Source) SourceKind() string {
	return SourceKind
}

func (s *Source) KustoClient() *kusto.Client {
	return s.Client
}

func (s *Source) GetDatabase() string {
	return s.Database
}

// ExecuteQuery executes a KQL query
func (s *Source) ExecuteQuery(ctx context.Context, query string) ([]map[string]interface{}, error) {
	// Clean the query
	cleanQuery := strings.TrimSpace(query)

	// Debug logging: print the query being sent to ADX
	log.Printf("DEBUG: Executing ADX query:\n%s", cleanQuery)

	// Create KQL statement
	stmt := kql.New("").AddUnsafe(cleanQuery)

	// Execute the query
	iter, err := s.Client.Query(ctx, s.Database, stmt)
	if err != nil {
		// Provide verbose error details for query execution failures
		errorDetails := fmt.Sprintf("Error type: %T, Error message: %v", err, err)
		// Try to get more details using error unwrapping if available
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			if cause := unwrapper.Unwrap(); cause != nil {
				errorDetails += fmt.Sprintf(", Underlying error: %T: %v", cause, cause)
			}
		}
		// Also include the full error string to get all available details
		errorDetails += fmt.Sprintf(", Full error string: %s", err.Error())
		log.Printf("DEBUG: Failed to execute ADX query - %s", errorDetails)
		return nil, fmt.Errorf("failed to execute query - %s", errorDetails)
	}
	defer iter.Stop()

	var results []map[string]interface{}
	for {
		row, inlineErr, finalErr := iter.NextRowOrError()
		if finalErr != nil {
			// EOF is the normal way to signal end of iteration, not an error
			if finalErr == io.EOF {
				break
			}
			// Provide verbose error details for debugging
			errorDetails := fmt.Sprintf("Error type: %T, Error message: %v", finalErr, finalErr)
			// Try to get more details using error unwrapping if available
			if unwrapper, ok := finalErr.(interface{ Unwrap() error }); ok {
				if cause := unwrapper.Unwrap(); cause != nil {
					errorDetails += fmt.Sprintf(", Underlying error: %T: %v", cause, cause)
				}
			}
			// Also try to convert to string to get all available details
			errorDetails += fmt.Sprintf(", Full error string: %s", finalErr.Error())
			log.Printf("DEBUG: Final error during ADX query iteration - %s", errorDetails)
			return nil, fmt.Errorf("final error during iteration - %s", errorDetails)
		}
		if inlineErr != nil {
			// Provide verbose error details for inline errors as well
			errorDetails := fmt.Sprintf("Error type: %T, Error message: %v", inlineErr, inlineErr)
			// For inline errors, convert to string to get full details
			errorDetails += fmt.Sprintf(", Full error string: %s", inlineErr.Error())
			log.Printf("DEBUG: Inline error during ADX query iteration - %s", errorDetails)
			return nil, fmt.Errorf("inline error during iteration - %s", errorDetails)
		}
		if row == nil {
			break
		}

		// Convert row to map using actual column names
		rowMap := make(map[string]interface{})
		columnNames := row.ColumnNames()
		
		for i, val := range row.Values {
			var colName string
			if i < len(columnNames) {
				colName = columnNames[i]
			} else {
				// Fallback to index-based name if column names are not available
				colName = fmt.Sprintf("Column_%d", i)
			}
			
			// Extract the actual value from the Kusto value object
			// The Kusto value when JSON marshaled produces a structure with Value and Valid fields
			// We want to extract just the underlying value
			var actualValue interface{}
			
			if val != nil {
				// Use reflection to access the underlying value
				// The Kusto values likely have a Value field that contains the actual data
				valReflect := reflect.ValueOf(val)
				
				// Handle pointer types
				for valReflect.Kind() == reflect.Ptr && !valReflect.IsNil() {
					valReflect = valReflect.Elem()
				}
				
				// Try to access a "Value" field if it exists
				if valReflect.Kind() == reflect.Struct && valReflect.FieldByName("Value").IsValid() {
					valueField := valReflect.FieldByName("Value")
					actualValue = valueField.Interface()
				} else {
					// Fallback to string representation if we can't access the Value field
					actualValue = val.String()
				}
			}
			
			rowMap[colName] = actualValue
		}
		results = append(results, rowMap)
	}

	return results, nil
}