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

package util

import "context"

type extensionsKey struct{}

// ClientExtensions holds the set of extension URIs supported by the client.
type ClientExtensions map[string]bool

// CapabilitiesProvider is implemented by client capabilities structs across all MCP protocol versions.
type CapabilitiesProvider interface {
	GetExtensions() map[string]any
	GetExperimental() map[string]any
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

// ExtractClientExtensions extracts supported extension URIs from standard extensions
// and experimental capabilities maps.
// According to the MCP specification, extensions may be advertised under
// clientCapabilities.extensions or clientCapabilities.experimental, with values
// represented either as boolean flags or as settings objects (where an empty
// object {} indicates default support).
func ExtractClientExtensions(extensions map[string]any, experimental map[string]any) ClientExtensions {
	exts := make(ClientExtensions)
	extract := func(m map[string]any) {
		if m == nil {
			return
		}
		for k, v := range m {
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
	}
	extract(extensions)
	extract(experimental)
	return exts
}

// WithClientCapabilities extracts client extensions from standard and experimental
// capabilities and attaches them to the context. This shared helper can be called
// by handlers across any protocol version.
func WithClientCapabilities(ctx context.Context, caps CapabilitiesProvider) context.Context {
	if caps == nil || (caps.GetExtensions() == nil && caps.GetExperimental() == nil) {
		return ctx
	}
	return WithClientExtensions(ctx, ExtractClientExtensions(caps.GetExtensions(), caps.GetExperimental()))
}
