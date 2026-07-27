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

// ClientExtensions holds the set of experimental extension URIs supported by the client.
type ClientExtensions map[string]bool

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

// ExtractClientExtensions extracts client experimental extensions from a capabilities map.
// This is typically called with request metadata such as meta.MetaClientCapabilities.Experimental.
func ExtractClientExtensions(experimental map[string]any) ClientExtensions {
	exts := make(ClientExtensions)
	if experimental != nil {
		for k, v := range experimental {
			if enabled, ok := v.(bool); ok && enabled {
				exts[k] = true
			}
		}
	}
	return exts
}
