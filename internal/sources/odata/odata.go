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

package odata

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"log/slog"

	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

// SourceType is the type identifier for the OData source.
const SourceType string = "odata"

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
	}
}

// validate interface
var _ sources.SourceConfig = Config{}

type AuthConfig struct {
	Type              string `yaml:"type" validate:"omitempty,oneof=basic bearer x509"` // "basic", "bearer", or "x509"
	Username          string `yaml:"username"`                                          // For basic
	Password          string `yaml:"password"`                                          // For basic
	Token             string `yaml:"token"`                                             // For bearer
	ClientCert        string `yaml:"clientCert"`                                        // For x509
	ClientKey         string `yaml:"clientKey"`                                         // For x509
	ClientKeyPassword string `yaml:"clientKeyPassword"`                                 // Optional for encrypted keys
	CACert            string `yaml:"caCert"`                                            // For x509
}

type CompatibilityConfig struct {
	UrlQuoting          bool `yaml:"urlQuoting"`          // Double single-quotes in string URL parameters for OData v2
	UseTunnelingHeaders bool `yaml:"useTunnelingHeaders"` // POST with X-HTTP-Method header for PATCH/MERGE
}

type Config struct {
	Name                   string              `yaml:"name" validate:"required"`
	Type                   string              `yaml:"type" validate:"required"`
	BaseURL                string              `yaml:"baseUrl" validate:"required"`
	Timeout                string              `yaml:"timeout"`
	DefaultHeaders         map[string]string   `yaml:"headers"`
	QueryParams            map[string]string   `yaml:"queryParams"`
	DisableSslVerification bool                `yaml:"disableSslVerification"`
	Auth                   AuthConfig          `yaml:"auth"` // OData specific auth
	UseClientOauth         string              `yaml:"useClientOauth"`
	AuthStrategy           string              `yaml:"authStrategy"`
	Compatibility          CompatibilityConfig `yaml:"compatibility"`
}

func newConfig(ctx context.Context, name string, decoder *yaml.Decoder) (sources.SourceConfig, error) {
	actual := Config{Name: name, Timeout: "30s"}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return nil, err
	}
	return actual, nil

}

func (c Config) SourceConfigType() string {
	return SourceType
}

func (c Config) Initialize(ctx context.Context, tracer trace.Tracer) (sources.Source, error) {
	duration, err := time.ParseDuration(c.Timeout)
	if err != nil {
		return nil, fmt.Errorf("unable to parse Timeout string as time.Duration: %s", err)
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.DisableSslVerification,
	}

	if c.Auth.Type == "x509" {
		if c.Auth.ClientCert == "" || c.Auth.ClientKey == "" {
			return nil, fmt.Errorf("x509 auth requires both clientCert and clientKey")
		}
		cert, err := tls.LoadX509KeyPair(c.Auth.ClientCert, c.Auth.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load x509 key pair: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	if c.Auth.CACert != "" {
		caCert, err := os.ReadFile(c.Auth.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read caCert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to parse caCert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	client := http.Client{
		Timeout:   duration,
		Transport: &http.Transport{TLSClientConfig: tlsConfig},
	}

	_, err = url.ParseRequestURI(c.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse BaseUrl %v", err)
	}

	if c.DefaultHeaders == nil {
		c.DefaultHeaders = make(map[string]string)
	}

	ua, err := util.UserAgentFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user agent from context: %w", err)
	}
	if existingUA, ok := c.DefaultHeaders["User-Agent"]; ok {
		ua = ua + " " + existingUA
	}
	c.DefaultHeaders["User-Agent"] = ua

	// Set default Accept and X-Requested-With if not explicitly configured
	if _, ok := c.DefaultHeaders["Accept"]; !ok {
		c.DefaultHeaders["Accept"] = "application/json"
	}
	if _, ok := c.DefaultHeaders["X-Requested-With"]; !ok {
		c.DefaultHeaders["X-Requested-With"] = "XMLHttpRequest"
	}

	var primary AuthStrategy
	switch c.Auth.Type {
	case "basic":
		primary = &BasicAuthStrategy{Username: c.Auth.Username, Password: c.Auth.Password}
	case "bearer":
		primary = &BearerTokenStrategy{Token: c.Auth.Token}
	case "x509":
		primary = &TlsStrategy{}
	default:
		primary = &BasicAuthStrategy{}
	}

	var dynamic AuthStrategy
	headerName := "Authorization"
	if c.UseClientOauth != "" {
		if c.UseClientOauth != "true" && c.UseClientOauth != "Authorization" {
			headerName = c.UseClientOauth
		}
		dynamic = &DynamicUserOauthStrategy{AuthTokenHeaderName: headerName}
	}

	var authStrategy AuthStrategy
	if c.AuthStrategy == "gateway" || c.AuthStrategy == "csrf-token" {
		authStrategy = NewGatewayStrategy(c.BaseURL, c.DefaultHeaders, primary, dynamic, headerName)
	} else {
		if dynamic != nil {
			authStrategy = dynamic
		} else {
			authStrategy = primary
		}
	}

	source := &Source{
		Config:              c,
		client:              &client,
		authStrategy:        authStrategy,
		AuthTokenHeaderName: headerName,
	}

	// Attempt initial metadata fetch at startup; if unauthenticated (e.g. user OAuth token required), lazy-load on first request.
	if err := source.fetchMetadata(ctx, ""); err != nil {
		logger, logErr := util.LoggerFromContext(ctx)
		if logErr != nil {
			return nil, fmt.Errorf("unable to get logger from ctx: %w", logErr)
		}
		noticeMsg := fmt.Sprintf("OData metadata will be lazily fetched on first user request: %v", err)
		logger.InfoContext(ctx, noticeMsg)
	}

	return source, nil
}

type Source struct {
	Config
	mu                  sync.RWMutex
	client              *http.Client
	metadata            *ODataMetadata
	authStrategy        AuthStrategy
	AuthTokenHeaderName string
}

var _ sources.Source = &Source{}

func (s *Source) SourceType() string {
	return SourceType
}

func (s *Source) ToConfig() sources.SourceConfig {
	return s.Config
}

func (s *Source) HttpBaseURL() string {
	return s.BaseURL
}

func (s *Source) UseClientAuthorization() bool {
	return s.UseClientOauth != "" && s.UseClientOauth != "false"
}

func (s *Source) GetAuthTokenHeaderName() string {
	if s.AuthTokenHeaderName == "" {
		return "Authorization"
	}
	return s.AuthTokenHeaderName
}

func (s *Source) Metadata() *ODataMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.metadata
}

func (s *Source) Compatibility() CompatibilityConfig {
	return s.Config.Compatibility
}

// fetchMetadata performs a GET to $metadata to fetch and parse the schema
func (s *Source) fetchMetadata(ctx context.Context, accessToken tools.AccessToken) error {
	metaURL := strings.TrimRight(s.BaseURL, "/") + "/$metadata"
	req, err := http.NewRequestWithContext(ctx, "GET", metaURL, nil)
	if err != nil {
		return err
	}

	for k, v := range s.DefaultHeaders {
		req.Header.Set(k, v)
	}
	// Overwrite Accept header for metadata specifically, as OData $metadata is always XML
	req.Header.Set("Accept", "application/xml")

	if accessToken != "" {
		req.Header.Set(s.GetAuthTokenHeaderName(), string(accessToken))
	}

	if err := s.authStrategy.Authorize(ctx, req, s.client); err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("metadata request failed (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse Metadata XML
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read metadata body: %v", err)
	}

	meta, err := ParseMetadata(bodyBytes)
	if err != nil {
		return fmt.Errorf("failed to parse metadata XML: %v", err)
	}
	s.mu.Lock()
	s.metadata = meta
	s.mu.Unlock()

	return nil
}

// RunODataRequest attaches auth and CSRF headers via AuthStrategy before executing.
func (s *Source) RunODataRequest(req *http.Request, accessToken tools.AccessToken) (any, error) {
	ctx := req.Context()

	if s.Metadata() == nil {
		_ = s.fetchMetadata(ctx, accessToken)
	}

	// 1. Pass context token via header so DynamicUserOauthStrategy can read it
	if accessToken != "" {
		req.Header.Set(s.GetAuthTokenHeaderName(), string(accessToken))
	}

	// 2. Delegate Authentication and Handshake to Strategy
	if err := s.authStrategy.Authorize(ctx, req, s.client); err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	}

	// Inject Default Headers
	for k, v := range s.DefaultHeaders {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http execution failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Evict cached session on 403 CSRF failure
		if resp.StatusCode == 403 && req.Method != "GET" && req.Method != "HEAD" {
			s.authStrategy.Evict(ctx, req)
		}
		return nil, fmt.Errorf("http error %d: %s", resp.StatusCode, string(body))
	}

	// If 204 No Content, return empty
	if resp.StatusCode == 204 || len(body) == 0 {
		return map[string]interface{}{"status": "success"}, nil
	}

	var data any
	if err = json.Unmarshal(body, &data); err != nil {
		return string(body), nil
	}
	return data, nil
}
