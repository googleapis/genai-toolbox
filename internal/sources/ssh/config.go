// Copyright 2024 Google LLC
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

package ssh

import (
	"fmt"
	"strconv"
	"strings"
)

// FlexBool is a custom boolean type that can unmarshal from both bool and string values
type FlexBool bool

// UnmarshalYAML implements custom YAML unmarshaling for boolean values that may come as strings
func (fb *FlexBool) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First, try to unmarshal as a boolean
	var boolVal bool
	if err := unmarshal(&boolVal); err == nil {
		*fb = FlexBool(boolVal)
		return nil
	}

	// If that fails, try to unmarshal as a string and convert to boolean
	var strVal string
	if err := unmarshal(&strVal); err != nil {
		return err
	}

	// Handle common string boolean representations
	switch strings.ToLower(strings.TrimSpace(strVal)) {
	case "true", "yes", "1", "on", "enabled":
		*fb = FlexBool(true)
	case "false", "no", "0", "off", "disabled", "":
		*fb = FlexBool(false)
	default:
		// Try standard strconv.ParseBool as fallback
		parsed, err := strconv.ParseBool(strVal)
		if err != nil {
			return fmt.Errorf("cannot parse %q as boolean", strVal)
		}
		*fb = FlexBool(parsed)
	}
	return nil
}

// Config represents SSH tunnel configuration for database sources
type Config struct {
	Enabled        FlexBool `yaml:"enabled"`
	Host          string `yaml:"host"`
	Port          string `yaml:"port,omitempty"`           // defaults to "22"
	User          string `yaml:"user"`
	Password      string `yaml:"password,omitempty"`
	PrivateKeyPath string `yaml:"private_key_path,omitempty"`
	Passphrase    string `yaml:"passphrase,omitempty"`
	LocalPort     int    `yaml:"local_port,omitempty"`     // 0 = auto-assign
	Timeout       string `yaml:"timeout,omitempty"`        // defaults to "30s"
}

// Validate checks if SSH configuration is valid
func (c Config) Validate() error {
	if !bool(c.Enabled) {
		return nil
	}
	if c.Host == "" {
		return fmt.Errorf("ssh.host is required when SSH is enabled")
	}
	if c.User == "" {
		return fmt.Errorf("ssh.user is required when SSH is enabled")
	}
	if c.Password == "" && c.PrivateKeyPath == "" {
		return fmt.Errorf("either ssh.password or ssh.private_key_path must be provided")
	}
	return nil
}

// PortWithDefault returns the SSH port, defaulting to "22" if not specified
func (c Config) PortWithDefault() string {
	if c.Port == "" {
		return "22"
	}
	return c.Port
}

// TimeoutWithDefault returns the timeout, defaulting to "30s" if not specified
func (c Config) TimeoutWithDefault() string {
	if c.Timeout == "" {
		return "30s"
	}
	return c.Timeout
}