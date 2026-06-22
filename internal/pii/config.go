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
	"context"
	"fmt"

	yaml "github.com/goccy/go-yaml"
)

type Action string

const (
	ActionUnmask     Action = "unmask"
	ActionPartial    Action = "partial_mask"
	ActionFull       Action = "full_mask"
	ActionDenyField  Action = "deny_field"
)

type MatchClaims map[string][]string

type Tier struct {
	Name        string      `yaml:"name"`
	MatchClaims MatchClaims `yaml:"matchClaims"`
	Action      Action      `yaml:"action"`
}

type Rule struct {
	Type    string `yaml:"type"`
	Pattern string `yaml:"pattern"`
	Column  string `yaml:"column"`
}

type PiiPolicyConfig struct {
	Name        string `yaml:"name"`
	DefaultTier Action `yaml:"defaultTier"`
	Tiers       []Tier `yaml:"tiers"`
	Rules       []Rule `yaml:"rules"`
}

func DecodeConfig(ctx context.Context, name string, decoder *yaml.Decoder) (PiiPolicyConfig, error) {
	actual := PiiPolicyConfig{Name: name}
	if err := decoder.DecodeContext(ctx, &actual); err != nil {
		return PiiPolicyConfig{}, fmt.Errorf("unable to parse as %s: %w", name, err)
	}

	// Default to full_mask if not specified for fail-closed security
	if actual.DefaultTier == "" {
		actual.DefaultTier = ActionFull
	}
	return actual, nil
}
