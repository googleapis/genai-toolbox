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

package piipolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// matchClaimValue safely checks if a claim value contains the target string,
// handling cases where the claim is an array of strings or interfaces.
func matchClaimValue(claimVal any, target string) bool {
	switch v := claimVal.(type) {
	case string:
		return v == target
	case []string:
		return slices.Contains(v, target)
	case []any:
		for _, item := range v {
			if fmt.Sprintf("%v", item) == target {
				return true
			}
		}
		return false
	default:
		return fmt.Sprintf("%v", claimVal) == target
	}
}

// ApplyPolicy evaluates the PII policy against the provided data given the user's claims.
func ApplyPolicy(ctx context.Context, config Config, claims map[string]any, data any) (any, error) {
	if len(config.Rules) == 0 {
		return data, nil // Nothing to do
	}

	action := determineAction(config, claims)
	return applyPolicyWithAction(ctx, config.Rules, action, data)
}

func determineAction(config Config, claims map[string]any) Action {
	defaultAction := Action(MaskFull) // Fail-closed baseline

	for _, tier := range config.Tiers {
		if tier.Name == config.DefaultTier {
			defaultAction = tier.Action
		}

		if claims != nil && len(tier.MatchClaims) > 0 {
			matched := true
			for k, v := range tier.MatchClaims {
				claimVal, ok := claims[k]
				if !ok || !matchClaimValue(claimVal, v) {
					matched = false
					break
				}
			}
			if matched {
				return tier.Action
			}
		}
	}

	return defaultAction
}

func applyPolicyWithAction(ctx context.Context, rules []Rule, action Action, data any) (any, error) {
	switch v := data.(type) {
	case string:
		return applyToString(v, rules, action)
	case map[string]any:
		return applyToMap(ctx, rules, action, v)
	case []map[string]any:
		return applyToMapSlice(ctx, rules, action, v)
	case []any:
		var out []any
		for _, el := range v {
			res, err := applyPolicyWithAction(ctx, rules, action, el)
			if err != nil {
				return nil, err
			}
			out = append(out, res)
		}
		return out, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool, nil:
		return data, nil
	default:
		// Attempt to convert struct to map/slice using JSON
		bytes, err := json.Marshal(data)
		if err == nil {
			var parsed any
			if err := json.Unmarshal(bytes, &parsed); err == nil {
				// Prevent infinite loop if type doesn't change
				if fmt.Sprintf("%T", parsed) != fmt.Sprintf("%T", data) {
					return applyPolicyWithAction(ctx, rules, action, parsed)
				}
			}
		}
		// Unsupported type, return as is
		return data, nil
	}
}

func applyToString(data string, rules []Rule, action Action) (string, error) {
	if action == Unmask {
		return data, nil
	}
	result := data
	for _, rule := range rules {
		if rule.Pattern == "" || rule.CompiledRegex == nil {
			continue // Column-based rule, skip for unstructured text
		}

		result = rule.CompiledRegex.ReplaceAllStringFunc(result, func(match string) string {
			return applyActionToString(action, match)
		})
	}
	return result, nil
}

func applyToMapSlice(ctx context.Context, rules []Rule, action Action, data []map[string]any) ([]map[string]any, error) {
	var out []map[string]any
	for _, m := range data {
		res, err := applyToMap(ctx, rules, action, m)
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

func applyToMap(ctx context.Context, rules []Rule, action Action, data map[string]any) (map[string]any, error) {
	if action == Unmask {
		out := make(map[string]any, len(data))
		for k, v := range data {
			out[k] = v
		}
		return out, nil
	}

	out := make(map[string]any, len(data))

	columnRules := make(map[string]bool)
	for _, rule := range rules {
		if rule.Column != "" {
			columnRules[rule.Column] = true
		}
	}

	for k, v := range data {
		if columnRules[k] {
			if action == DenyField {
				out[k] = "[DENIED]"
				continue
			}
			if strVal, ok := v.(string); ok {
				out[k] = applyActionToString(action, strVal)
			} else {
				out[k] = "***"
			}
		} else {
			res, err := applyPolicyWithAction(ctx, rules, action, v)
			if err != nil {
				return nil, err
			}
			out[k] = res
		}
	}
	return out, nil
}

func applyActionToString(action Action, val string) string {
	switch action {
	case MaskFull:
		return strings.Repeat("*", utf8.RuneCountInString(val))
	case MaskPartial:
		runes := []rune(val)
		if len(runes) <= 2 {
			return strings.Repeat("*", len(runes))
		}
		visibleLen := len(runes) / 2
		return string(runes[:visibleLen]) + strings.Repeat("*", len(runes)-visibleLen)
	case DenyField:
		return "[DENIED]"
	default:
		return val // Unmask or unknown
	}
}
