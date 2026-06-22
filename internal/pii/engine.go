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

package pii

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/googleapis/mcp-toolbox/internal/util/orderedmap"
)

type Engine struct {
	Config       PiiPolicyConfig
	compiledRule []compiledRule
}

type compiledRule struct {
	Rule    Rule
	Regex   *regexp.Regexp
	IsRegex bool
}

func NewEngine(config PiiPolicyConfig) (*Engine, error) {
	engine := &Engine{
		Config: config,
	}

	for _, rule := range config.Rules {
		cr := compiledRule{Rule: rule}
		if rule.Pattern != "" {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to compile regex for rule %s: %w", rule.Type, err)
			}
			cr.Regex = re
			cr.IsRegex = true
		}
		engine.compiledRule = append(engine.compiledRule, cr)
	}

	return engine, nil
}

func (e *Engine) getAction(claimsFromAuth map[string]map[string]any) Action {
	if len(e.Config.Tiers) == 0 {
		return e.Config.DefaultTier
	}

	for _, tier := range e.Config.Tiers {
		if len(tier.MatchClaims) == 0 {
			return tier.Action
		}

		tierMatched := true
		for claimKey, allowedValues := range tier.MatchClaims {
			keyMatched := false
			for _, claims := range claimsFromAuth {
				if claimVal, ok := claims[claimKey]; ok {
					if cvSlice, ok := claimVal.([]string); ok {
						for _, cv := range cvSlice {
							for _, allowed := range allowedValues {
								if cv == allowed {
									keyMatched = true
									break
								}
							}
							if keyMatched {
								break
							}
						}
					} else if cvStr, ok := claimVal.(string); ok {
						for _, allowed := range allowedValues {
							if cvStr == allowed {
								keyMatched = true
								break
							}
						}
					} else if cvSliceI, ok := claimVal.([]any); ok {
						for _, cv := range cvSliceI {
							if cvStr, ok := cv.(string); ok {
								for _, allowed := range allowedValues {
									if cvStr == allowed {
										keyMatched = true
										break
									}
								}
							}
							if keyMatched {
								break
							}
						}
					}
				}
				if keyMatched {
					break
				}
			}
			if !keyMatched {
				tierMatched = false
				break
			}
		}

		if tierMatched {
			return tier.Action
		}
	}

	return e.Config.DefaultTier
}

type auditStats struct {
	matchedTypes map[string]int
}

func (e *Engine) Mask(data any, claimsFromAuth map[string]map[string]any) (any, error) {
	action := e.getAction(claimsFromAuth)
	
	stats := &auditStats{
		matchedTypes: make(map[string]int),
	}

	// Fast path for unmask
	if action == ActionUnmask {
		slog.Info("PII Policy Audit", "policy", e.Config.Name, "action", action, "reason", "unmask allowed by tier")
		return data, nil
	}

	maskedData, err := e.maskRecursive(data, action, stats)
	
	// Emit audit log
	if len(stats.matchedTypes) > 0 {
		slog.Info("PII Policy Audit", "policy", e.Config.Name, "action", action, "matches", stats.matchedTypes)
	} else {
		slog.Info("PII Policy Audit", "policy", e.Config.Name, "action", action, "reason", "no PII matched")
	}
	
	return maskedData, err
}

func (e *Engine) maskRecursive(data any, action Action, stats *auditStats) (any, error) {
	switch v := data.(type) {
	case string:
		return e.maskString(v, action, stats), nil
	case map[string]any:
		result := make(map[string]any)
		for key, val := range v {
			columnMatched := false
			for _, rule := range e.compiledRule {
				if !rule.IsRegex && rule.Rule.Column != "" && strings.EqualFold(rule.Rule.Column, key) {
					columnMatched = true
					stats.matchedTypes[rule.Rule.Type]++
					if action == ActionDenyField {
						break
					}
					if action == ActionFull {
						result[key] = fmt.Sprintf("[REDACTED:%s]", rule.Rule.Type)
					} else if action == ActionPartial {
						if val == nil {
							result[key] = nil
						} else {
							result[key] = partialMask(fmt.Sprintf("%v", val))
						}
					}
					break
				}
			}
			if columnMatched {
				continue
			}

			maskedVal, _ := e.maskRecursive(val, action, stats)
			result[key] = maskedVal
		}
		return result, nil
	case orderedmap.Row:
		var columns []orderedmap.Column
		for _, col := range v.Columns {
			columnMatched := false
			var maskedCol orderedmap.Column
			for _, rule := range e.compiledRule {
				if !rule.IsRegex && rule.Rule.Column != "" && strings.EqualFold(rule.Rule.Column, col.Name) {
					columnMatched = true
					stats.matchedTypes[rule.Rule.Type]++
					if action == ActionDenyField {
						break
					}
					if action == ActionFull {
						maskedCol = orderedmap.Column{Name: col.Name, Value: fmt.Sprintf("[REDACTED:%s]", rule.Rule.Type)}
					} else if action == ActionPartial {
						if col.Value == nil {
							maskedCol = orderedmap.Column{Name: col.Name, Value: nil}
						} else {
							maskedCol = orderedmap.Column{Name: col.Name, Value: partialMask(fmt.Sprintf("%v", col.Value))}
						}
					}
					break
				}
			}
			if columnMatched {
				if action != ActionDenyField {
					columns = append(columns, maskedCol)
				}
				continue
			}

			maskedVal, _ := e.maskRecursive(col.Value, action, stats)
			columns = append(columns, orderedmap.Column{Name: col.Name, Value: maskedVal})
		}
		return orderedmap.Row{Columns: columns}, nil
	case []any:
		result := make([]any, len(v))
		for i, val := range v {
			maskedVal, _ := e.maskRecursive(val, action, stats)
			result[i] = maskedVal
		}
		return result, nil
	default:
		return v, nil
	}
}

func (e *Engine) maskString(val string, action Action, stats *auditStats) string {
	if action == ActionDenyField {
		action = ActionFull
	}

	result := val
	for _, rule := range e.compiledRule {
		if rule.IsRegex {
			result = rule.Regex.ReplaceAllStringFunc(result, func(match string) string {
				stats.matchedTypes[rule.Rule.Type]++
				if action == ActionFull {
					return fmt.Sprintf("[REDACTED:%s]", rule.Rule.Type)
				} else if action == ActionPartial {
					return partialMask(match)
				}
				return match
			})
		}
	}
	return result
}

func partialMask(val string) string {
	runes := []rune(val)
	if len(runes) <= 2 {
		return strings.Repeat("*", len(runes))
	}
	
	if strings.Contains(val, "@") {
		parts := strings.Split(val, "@")
		if len(parts) == 2 {
			localRunes := []rune(parts[0])
			domain := parts[1]
			
			maskedLocal := ""
			if len(localRunes) > 2 {
				maskedLocal = string(localRunes[0]) + strings.Repeat("*", len(localRunes)-2) + string(localRunes[len(localRunes)-1])
			} else {
				maskedLocal = strings.Repeat("*", len(localRunes))
			}
			return maskedLocal + "@" + domain
		}
	}

	return string(runes[0]) + strings.Repeat("*", len(runes)-2) + string(runes[len(runes)-1])
}
