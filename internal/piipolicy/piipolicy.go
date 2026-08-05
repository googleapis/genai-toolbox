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

import "regexp"

type Action string

const (
	Unmask      Action = "unmask"
	MaskPartial Action = "partial_mask"
	MaskFull    Action = "full_mask"
	DenyField   Action = "deny_field"
)

type Tier struct {
	Name        string            `yaml:"name"`
	MatchClaims map[string]string `yaml:"matchClaims"`
	Action      Action            `yaml:"action"`
}

type Rule struct {
	Pattern       string         `yaml:"pattern,omitempty"` // Regex pattern for unstructured text
	CompiledRegex *regexp.Regexp `yaml:"-"`                 // Compiled regex
	Column        string         `yaml:"column,omitempty"`  // Column name for structured data
}

type Config struct {
	Name        string `yaml:"name"`
	DefaultTier string `yaml:"defaultTier"`
	Tiers       []Tier `yaml:"tiers"`
	Rules       []Rule `yaml:"rules"`
}
