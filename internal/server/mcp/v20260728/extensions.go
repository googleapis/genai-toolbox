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

package v20260728

import "context"

type extensionsKey struct{}
type serverExtensionsKey struct{}

// ClientExtensions holds the set of extension URIs supported by the client.
type ClientExtensions map[string]bool

// CapabilitiesProvider provides access to client extension maps.
type CapabilitiesProvider interface {
	GetExtensions() map[string]any
}

// WithClientExtensions attaches the client's supported extensions to the context.
func WithClientExtensions(ctx context.Context, exts ClientExtensions) context.Context {
	return context.WithValue(ctx, extensionsKey{}, exts)
}

// ClientExtensionsFromContext retrieves the client's extension map from the context.
func ClientExtensionsFromContext(ctx context.Context) ClientExtensions {
	exts, _ := ctx.Value(extensionsKey{}).(ClientExtensions)
	return exts
}

// SupportsExtension returns true if the client explicitly declared support for the given extension URI.
func SupportsExtension(ctx context.Context, uri string) bool {
	exts := ClientExtensionsFromContext(ctx)
	return exts != nil && exts[uri]
}

// ExtractClientExtensions extracts supported extension URIs from standard capability maps.
func ExtractClientExtensions(extensions map[string]any) ClientExtensions {
	exts := make(ClientExtensions)
	if extensions == nil {
		return exts
	}
	for k, v := range extensions {
		if v == nil {
			continue
		}
		if b, ok := v.(bool); ok {
			if b {
				exts[k] = true
			}
			continue
		}
		// Any non-nil, non-false setting object (e.g., {}, map, struct) indicates support.
		exts[k] = true
	}
	return exts
}

// WithClientCapabilities extracts client extensions from capabilities and attaches them to the context.
func WithClientCapabilities(ctx context.Context, caps CapabilitiesProvider) context.Context {
	if caps == nil || caps.GetExtensions() == nil {
		return ctx
	}
	return WithClientExtensions(ctx, ExtractClientExtensions(caps.GetExtensions()))
}

// WithServerExtensions adds server standard extensions into the context.
func WithServerExtensions(ctx context.Context, exts []string) context.Context {
	return context.WithValue(ctx, serverExtensionsKey{}, exts)
}

// ServerExtensionsFromContext retrieves server standard extensions from context.
func ServerExtensionsFromContext(ctx context.Context) ([]string, bool) {
	if exts, ok := ctx.Value(serverExtensionsKey{}).([]string); ok {
		return exts, true
	}
	return nil, false
}

// GetServerExtensions returns extensions advertised by this server from context.
func GetServerExtensions(ctx context.Context) map[string]interface{} {
	exts, _ := ServerExtensionsFromContext(ctx)
	if len(exts) == 0 {
		return nil
	}
	res := make(map[string]interface{}, len(exts))
	for _, uri := range exts {
		res[uri] = map[string]any{}
	}
	return res
}
