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

package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/googleapis/mcp-toolbox/internal/server/mcp/jsonrpc"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/idtoken"
)

const (
	authModeIDToken     = "id-token"
	authModeAccessToken = "access-token"
	authModeNone        = "none"

	defaultAuthHeader = "Authorization"
	defaultScope      = "https://www.googleapis.com/auth/cloud-platform"
)

type proxyConfig struct {
	target     string
	authMode   string
	authHeader string
	audience   string
	scopes     []string
}

type tokenSourceFactory interface {
	TokenSource(context.Context, proxyConfig) (oauth2.TokenSource, error)
}

type CommandOptions struct {
	In      io.Reader
	Out     io.Writer
	Version string
	Setup   func(context.Context) (context.Context, func(context.Context) error, error)
}

type defaultTokenSourceFactory struct{}

func (defaultTokenSourceFactory) TokenSource(ctx context.Context, cfg proxyConfig) (oauth2.TokenSource, error) {
	switch cfg.authMode {
	case authModeIDToken:
		audience := cfg.audience
		if audience == "" {
			u, err := parseTarget(cfg.target)
			if err != nil {
				return nil, err
			}
			audience = targetOrigin(u)
		}
		return idtoken.NewTokenSource(ctx, audience)
	case authModeAccessToken:
		scopes := cfg.scopes
		if len(scopes) == 0 {
			scopes = []string{defaultScope}
		}
		return googleoauth.DefaultTokenSource(ctx, scopes...)
	case authModeNone:
		return nil, nil
	default:
		return nil, fmt.Errorf(`auth mode must be one of "%s", "%s", or "%s"`, authModeIDToken, authModeAccessToken, authModeNone)
	}
}

func NewCommand(opts CommandOptions) *cobra.Command {
	return newCommand(opts, defaultTokenSourceFactory{})
}

func newCommand(opts CommandOptions, tokenFactory tokenSourceFactory) *cobra.Command {
	cfg := proxyConfig{
		authMode:   authModeIDToken,
		authHeader: defaultAuthHeader,
		scopes:     []string{defaultScope},
	}

	cmd := &cobra.Command{
		Use:   "proxy",
		Short: "Proxy MCP stdio to a remote HTTP MCP server",
		Long: `Proxy MCP stdio to a remote HTTP MCP server.

The proxy reads newline-delimited MCP JSON-RPC messages from stdin, forwards
them to the remote HTTP MCP endpoint, injects Application Default Credentials
on upstream requests, and writes JSON-RPC responses back to stdout.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProxy(cmd, opts, cfg, tokenFactory)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&cfg.target, "target", "", "Remote HTTP MCP endpoint to proxy to, for example https://service.run.app/mcp.")
	flags.StringVar(&cfg.authMode, "auth-mode", authModeIDToken, fmt.Sprintf("Authentication mode for upstream requests. Allowed: %q, %q, %q.", authModeIDToken, authModeAccessToken, authModeNone))
	flags.StringVar(&cfg.authHeader, "auth-header", defaultAuthHeader, "Header used to send the upstream credential.")
	flags.StringVar(&cfg.audience, "audience", "", "Audience for ID tokens. Defaults to the target origin.")
	flags.StringSliceVar(&cfg.scopes, "scope", []string{defaultScope}, "OAuth scopes for access tokens. Can be specified multiple times.")
	_ = cmd.MarkFlagRequired("target")

	return cmd
}

func runProxy(cmd *cobra.Command, opts CommandOptions, cfg proxyConfig, tokenFactory tokenSourceFactory) error {
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(signals)
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-signals:
			cancel()
		}
	}()

	setup := opts.Setup
	if setup == nil {
		setup = func(ctx context.Context) (context.Context, func(context.Context) error, error) {
			return ctx, func(context.Context) error { return nil }, nil
		}
	}

	ctx, shutdown, err := setup(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	target, err := parseTarget(cfg.target)
	if err != nil {
		return err
	}
	if cfg.authHeader == "" {
		return fmt.Errorf("auth header must not be empty")
	}

	tokenSource, err := tokenFactory.TokenSource(ctx, cfg)
	if err != nil {
		return fmt.Errorf("unable to initialize upstream credentials: %w", err)
	}

	userAgent := "genai-toolbox/" + opts.Version
	if opts.Version == "" {
		userAgent = "genai-toolbox/dev"
	}
	in := opts.In
	if in == nil {
		in = cmd.InOrStdin()
	}
	out := opts.Out
	if out == nil {
		out = cmd.OutOrStdout()
	}

	p := &stdioProxy{
		target:      target,
		client:      http.DefaultClient,
		tokenSource: tokenSource,
		authHeader:  cfg.authHeader,
		userAgent:   userAgent,
	}

	return p.Start(ctx, in, out)
}

func parseTarget(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, fmt.Errorf("target must not be empty")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid target: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("target must use http or https")
	}
	if u.Host == "" {
		return nil, fmt.Errorf("target must include a host")
	}
	return u, nil
}

func targetOrigin(u *url.URL) string {
	origin := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}
	return origin.String()
}

type stdioProxy struct {
	target      *url.URL
	client      *http.Client
	tokenSource oauth2.TokenSource
	authHeader  string
	userAgent   string

	sessionID       string
	protocolVersion string
}

type rpcEnvelope struct {
	Jsonrpc string `json:"jsonrpc"`
	Method  string `json:"method"`
	Id      any    `json:"id,omitempty"`
	Params  struct {
		ProtocolVersion string `json:"protocolVersion,omitempty"`
	} `json:"params,omitempty"`
}

func (p *stdioProxy) Start(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				if len(bytes.TrimSpace(line)) == 0 {
					return nil
				}
			} else {
				return fmt.Errorf("unable to read stdin: %w", err)
			}
		}

		if len(bytes.TrimSpace(line)) > 0 {
			if err := p.forwardMessage(ctx, bytes.TrimSpace(line), out); err != nil {
				return err
			}
		}

		if err == io.EOF {
			return nil
		}
	}
}

func (p *stdioProxy) forwardMessage(ctx context.Context, body []byte, out io.Writer) error {
	env, err := parseRPCEnvelope(body)
	if err != nil {
		return writeRPCError(out, nil, jsonrpc.PARSE_ERROR, fmt.Sprintf("parse error: %v", err))
	}

	if env.Jsonrpc != jsonrpc.JSONRPC_VERSION {
		return writeRPCError(out, env.Id, jsonrpc.INVALID_REQUEST, "invalid json-rpc version")
	}
	if env.Method == "" {
		return writeRPCError(out, env.Id, jsonrpc.METHOD_NOT_FOUND, "method not found")
	}

	initializeProtocolVersion := ""
	if env.Method == "initialize" && env.Params.ProtocolVersion != "" {
		initializeProtocolVersion = env.Params.ProtocolVersion
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.target.String(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Content-Type", "application/json")
	if p.userAgent != "" {
		req.Header.Set("User-Agent", p.userAgent)
	}
	if p.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", p.sessionID)
	}
	if p.protocolVersion != "" {
		req.Header.Set("MCP-Protocol-Version", p.protocolVersion)
	}
	if p.tokenSource != nil {
		tok, err := p.tokenSource.Token()
		if err != nil {
			return fmt.Errorf("unable to retrieve upstream credential: %w", err)
		}
		req.Header.Set(p.authHeader, tok.Type()+" "+tok.AccessToken)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("upstream request failed: %w", err)
	}
	defer resp.Body.Close()

	if sessionID := resp.Header.Get("Mcp-Session-Id"); sessionID != "" {
		p.sessionID = sessionID
	}

	expectsResponse := env.Id != nil
	if err := p.writeUpstreamResponse(ctx, out, resp, env.Id, expectsResponse); err != nil {
		return err
	}
	if p.protocolVersion == "" && initializeProtocolVersion != "" {
		p.protocolVersion = initializeProtocolVersion
	}
	return nil
}

func parseRPCEnvelope(body []byte) (rpcEnvelope, error) {
	var env rpcEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return rpcEnvelope{}, err
	}
	return env, nil
}

func (p *stdioProxy) writeUpstreamResponse(ctx context.Context, out io.Writer, resp *http.Response, id any, expectsResponse bool) error {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/event-stream") {
		return p.writeSSEResponse(ctx, out, resp.Body, expectsResponse)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read upstream response: %w", err)
	}
	body = bytes.TrimSpace(body)

	if resp.StatusCode == http.StatusAccepted || resp.StatusCode == http.StatusNoContent || len(body) == 0 {
		return nil
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if expectsResponse && json.Valid(body) {
			return p.writeResponseLine(out, body)
		}
		if expectsResponse {
			return writeRPCError(out, id, rpcErrorCodeForStatus(resp.StatusCode), fmt.Sprintf("upstream returned HTTP status %s", resp.Status))
		}
		return fmt.Errorf("upstream returned HTTP status %s", resp.Status)
	}

	if !expectsResponse {
		return nil
	}
	if !json.Valid(body) {
		return writeRPCError(out, id, jsonrpc.INTERNAL_ERROR, "upstream returned a non-JSON MCP response")
	}

	return p.writeResponseLine(out, body)
}

func (p *stdioProxy) writeSSEResponse(ctx context.Context, out io.Writer, body io.Reader, expectsResponse bool) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 1024), 10*1024*1024)

	var event string
	var data []string
	flush := func() error {
		defer func() {
			event = ""
			data = nil
		}()
		if !expectsResponse || len(data) == 0 {
			return nil
		}
		if event != "" && event != "message" {
			return nil
		}

		payload := []byte(strings.Join(data, "\n"))
		payload = bytes.TrimSpace(payload)
		if len(payload) == 0 {
			return nil
		}
		if !json.Valid(payload) {
			return fmt.Errorf("upstream returned a non-JSON MCP SSE message")
		}
		return p.writeResponseLine(out, payload)
	}

	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil
		}

		line := scanner.Text()
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimPrefix(value, " ")

		switch field {
		case "event":
			event = value
		case "data":
			data = append(data, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("unable to read upstream SSE response: %w", err)
	}
	if len(data) > 0 {
		return flush()
	}
	return nil
}

func (p *stdioProxy) writeResponseLine(out io.Writer, body []byte) error {
	var compact bytes.Buffer
	if err := json.Compact(&compact, body); err != nil {
		return err
	}

	var response struct {
		Result struct {
			ProtocolVersion string `json:"protocolVersion,omitempty"`
		} `json:"result,omitempty"`
	}
	if err := json.Unmarshal(compact.Bytes(), &response); err == nil && response.Result.ProtocolVersion != "" {
		p.protocolVersion = response.Result.ProtocolVersion
	}

	compact.WriteByte('\n')
	_, err := out.Write(compact.Bytes())
	return err
}

func writeRPCError(out io.Writer, id any, code int, message string) error {
	res := jsonrpc.NewError(id, code, message, nil)
	body, err := json.Marshal(res)
	if err != nil {
		return err
	}
	body = append(body, '\n')
	_, err = out.Write(body)
	return err
}

func rpcErrorCodeForStatus(status int) int {
	switch status {
	case http.StatusUnauthorized:
		return jsonrpc.UNAUTHORIZED
	case http.StatusForbidden:
		return jsonrpc.FORBIDDEN
	default:
		return jsonrpc.INTERNAL_ERROR
	}
}
