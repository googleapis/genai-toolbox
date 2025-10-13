// Copyright 2025 Google LLC
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

// Copyright 2025 Google LLC
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

package prompts

import (
	"fmt"

	"github.com/googleapis/genai-toolbox/internal/tools"
)

// Message represents a single message in a prompt, with a role and content.
type Message struct {
	Role    string `yaml:"role,omitempty"`
	Content string `yaml:"content"`
}

func (m *Message) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Use a type alias to prevent an infinite recursion loop. The alias
	// has the same fields but lacks the UnmarshalYAML method.
	type messageAlias Message
	var alias messageAlias
	if err := unmarshal(&alias); err != nil {
		return err
	}

	*m = Message(alias)
	if m.Role == "" {
		m.Role = "user"
	}
	if m.Role != "user" && m.Role != "assistant" {
		return fmt.Errorf("invalid role %q: must be 'user' or 'assistant'", m.Role)
	}
	return nil
}

// SubstituteMessages takes a slice of Messages and a set of parameter values,
// and returns a new slice with all template variables resolved.
func SubstituteMessages(messages []Message, arguments Arguments, argValues tools.ParamValues) ([]Message, error) {
	substitutedMessages := make([]Message, 0, len(messages))
	argsMap := argValues.AsMap()

	var parameters tools.Parameters
	for _, arg := range arguments {
		parameters = append(parameters, arg)
	}

	for _, msg := range messages {
		substitutedContent, err := tools.ResolveTemplateParams(parameters, msg.Content, argsMap)
		if err != nil {
			return nil, fmt.Errorf("error substituting params for message: %w", err)
		}

		substitutedMessages = append(substitutedMessages, Message{
			Role:    msg.Role,
			Content: substitutedContent,
		})
	}

	return substitutedMessages, nil
}
