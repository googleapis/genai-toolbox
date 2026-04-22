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

package sapodata

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"container/list"

	"github.com/goccy/go-yaml"
	"github.com/googleapis/mcp-toolbox/internal/sources"
	"github.com/googleapis/mcp-toolbox/internal/tools"
	"github.com/googleapis/mcp-toolbox/internal/util"
	"go.opentelemetry.io/otel/trace"
)

const SourceType string = "sap-odata"

func init() {
	if !sources.Register(SourceType, newConfig) {
		panic(fmt.Sprintf("source type %q already registered", SourceType))
	}
}

type AuthConfig struct {
	Type              string `yaml:"type"`              // "basic" or "bearer"
	Username          string `yaml:"username"`          // For basic
	Password          string `yaml:"password"`          // For basic
	Token             string `yaml:"token"`             // For bearer
	ClientCert        string `yaml:"clientCert"`        // For x509
	ClientKey         string `yaml:"clientKey"`         // For x509
	ClientKeyPassword string `yaml:"clientKeyPassword"` // Optional for encrypted keys
	CACert            string `yaml:"caCert"`            // For x509
}

type Config struct {
	Name                   string            `yaml:"name" validate:"required"`
	Type                   string            `yaml:"type" validate:"required"`
	BaseURL                string            `yaml:"baseUrl" validate:"required"`
	Timeout                string            `yaml:"timeout"`
	DefaultHeaders         map[string]string `yaml:"headers"`
	QueryParams            map[string]string `yaml:"queryParams"`
	DisableSslVerification bool              `yaml:"disableSslVerification"`
	Auth                   AuthConfig        `yaml:"auth"` // SAP specific auth
	UseClientOauth         string            `yaml:"useClientOauth"`
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

	ua, _ := util.UserAgentFromContext(ctx)
	if existingUA, ok := c.DefaultHeaders["User-Agent"]; ok {
		ua = ua + " " + existingUA
	}
	c.DefaultHeaders["User-Agent"] = ua

	// Force JSON for OData v2/v4
	c.DefaultHeaders["Accept"] = "application/json"
	c.DefaultHeaders["X-Requested-With"] = "XMLHttpRequest"

	source := &Source{
		Config:       c,
		client:       &client,
		sessionCache: newSessionCache(1000, 30*time.Minute),
	}

	if c.UseClientOauth != "" {
		if c.UseClientOauth == "true" || c.UseClientOauth == "Authorization" {
			source.authTokenHeaderName = "Authorization"
		} else {
			source.authTokenHeaderName = c.UseClientOauth
		}
	}

	// Fetch metadata (schema) at initialization.
	// We no longer fetch CSRF globally here, as CSRF is user-specific.
	if err := source.fetchMetadata(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize SAP OData metadata: %v", err)
	}

	return source, nil
}

type UserSession struct {
	CsrfToken string
	Jar       http.CookieJar
	ExpiresAt time.Time
}

type lruItem struct {
	key     string
	session *UserSession
}

type sessionLRUCache struct {
	maxItems int
	ttl      time.Duration
	ll       *list.List
	cache    map[string]*list.Element
	mu       sync.RWMutex
}

func newSessionCache(maxItems int, ttl time.Duration) *sessionLRUCache {
	return &sessionLRUCache{
		maxItems: maxItems,
		ttl:      ttl,
		ll:       list.New(),
		cache:    make(map[string]*list.Element),
	}
}

func (c *sessionLRUCache) Get(key string) *UserSession {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, hit := c.cache[key]; hit {
		item := ele.Value.(*lruItem)
		if time.Now().After(item.session.ExpiresAt) {
			c.ll.Remove(ele)
			delete(c.cache, key)
			return nil
		}
		c.ll.MoveToFront(ele)
		return item.session
	}
	return nil
}

func (c *sessionLRUCache) Set(key string, session *UserSession) {
	c.mu.Lock()
	defer c.mu.Unlock()

	session.ExpiresAt = time.Now().Add(c.ttl)

	if ele, hit := c.cache[key]; hit {
		c.ll.MoveToFront(ele)
		ele.Value.(*lruItem).session = session
		return
	}

	ele := c.ll.PushFront(&lruItem{key: key, session: session})
	c.cache[key] = ele

	if c.maxItems != 0 && c.ll.Len() > c.maxItems {
		c.removeOldest()
	}
}

func (c *sessionLRUCache) removeOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		item := ele.Value.(*lruItem)
		delete(c.cache, item.key)
	}
}

func (c *sessionLRUCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ele, hit := c.cache[key]; hit {
		c.ll.Remove(ele)
		delete(c.cache, key)
	}
}

type Source struct {
	Config
	client              *http.Client
	metadata            *ODataMetadata
	sessionCache        *sessionLRUCache
	authTokenHeaderName string
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

func (s *Source) IsClientOauthEnabled() bool {
	return s.UseClientOauth != "" && s.UseClientOauth != "false"
}

func (s *Source) GetAuthTokenHeaderName() string {
	if s.authTokenHeaderName == "" {
		return "Authorization"
	}
	return s.authTokenHeaderName
}

func (s *Source) Metadata() *ODataMetadata {
	return s.metadata
}

// injectAuth adds the configured auth to the HTTP request.
// It prioritizes the provided accessToken (if useClientOauth is enabled)
// over the source-level static credentials.
func (s *Source) injectAuth(req *http.Request, accessToken tools.AccessToken) {
	if s.IsClientOauthEnabled() && accessToken != "" {
		// Identity B: Use the pass-through token from the human user (OAuth)
		tokenStr := string(accessToken)
		if !strings.HasPrefix(strings.ToLower(tokenStr), "bearer ") {
			tokenStr = "Bearer " + tokenStr
		}
		req.Header.Set("Authorization", tokenStr)
		return
	}

	if req.Header.Get("Authorization") != "" {
		return // Manual override already present
	}

	// Identity A: Fallback to Source-Level Technical Identity
	switch s.Auth.Type {
	case "x509":
		// No-op at the HTTP header level.
		// Authentication happens natively at the TLS layer via the specialized http.Client.
		return
	case "basic":
		req.SetBasicAuth(s.Auth.Username, s.Auth.Password)
	case "bearer":
		if s.Auth.Token != "" {
			req.Header.Set("Authorization", "Bearer "+s.Auth.Token)
		}
	}
}

// hashAuth creates a secure, anonymous key for the session cache
func hashAuth(authHeader string) string {
	if authHeader == "" {
		return "anonymous"
	}
	hash := sha256.Sum256([]byte(authHeader))
	return hex.EncodeToString(hash[:])
}

// fetchMetadata performs a GET to $metadata to fetch and parse the schema
func (s *Source) fetchMetadata(ctx context.Context) error {
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
	s.injectAuth(req, "")

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
	s.metadata = meta

	return nil
}

// fetchUserCsrf preemptively fires a HEAD request for a specific user to grab their CSRF token & cookies
func (s *Source) fetchUserCsrf(req *http.Request, authKey string) (*UserSession, error) {
	fetchReq, err := http.NewRequest("HEAD", s.BaseURL, nil)
	if err != nil {
		return nil, err
	}

	// Copy auth and default headers
	s.injectAuth(fetchReq, tools.AccessToken(req.Header.Get("Authorization")))
	for k, v := range s.DefaultHeaders {
		fetchReq.Header.Set(k, v)
	}
	fetchReq.Header.Set("X-CSRF-Token", "Fetch")

	jar, _ := cookiejar.New(nil)
	fetchClient := &http.Client{
		Timeout:   s.client.Timeout,
		Transport: s.client.Transport,
		Jar:       jar,
	}

	resp, err := fetchClient.Do(fetchReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	token := resp.Header.Get("X-CSRF-Token")
	if token == "" || token == "Required" {
		return nil, fmt.Errorf("failed to fetch valid CSRF token. status: %d", resp.StatusCode)
	}

	session := &UserSession{
		CsrfToken: token,
		Jar:       jar, // The jar automatically captured the Set-Cookie headers
	}

	s.sessionCache.Set(authKey, session)

	return session, nil
}

// RunSAPRequest attaches auth and CSRF headers before executing
func (s *Source) RunSAPRequest(req *http.Request, accessToken tools.AccessToken) (any, error) {
	// 1. Inject Authentication
	s.injectAuth(req, accessToken)

	authHeader := req.Header.Get("Authorization")
	sessionKey := hashAuth(authHeader)

	// 2. Multi-Tenant CSRF & Session Management
	session := s.sessionCache.Get(sessionKey)

	// If modifying data (POST/PUT/DELETE/etc.) and no session exists, fetch it dynamically
	if req.Method != "GET" && req.Method != "HEAD" {
		if session == nil {
			var err error
			session, err = s.fetchUserCsrf(req, sessionKey)
			if err != nil {
				return nil, fmt.Errorf("csrf setup failed: %v", err)
			}
		}
		// Inject the user's specific CSRF token
		req.Header.Set("X-CSRF-Token", session.CsrfToken)
	}

	// 3. Inject User's Cookies (if they have an active session)
	if session != nil && session.Jar != nil {
		cookies := session.Jar.Cookies(req.URL)
		for _, c := range cookies {
			req.AddCookie(c)
		}
	}

	// Inject Default Headers
	for k, v := range s.DefaultHeaders {
		if req.Header.Get(k) == "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sap http execution failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Clear session cache on 403 to force a new CSRF token fetch next time
		if resp.StatusCode == 403 && req.Method != "GET" && req.Method != "HEAD" {
			s.sessionCache.Remove(sessionKey)
		}
		// Log response for debugging a 400 Bad Request
		return nil, fmt.Errorf("sap http error %d: %s", resp.StatusCode, string(body))
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
