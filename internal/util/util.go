// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-playground/validator/v10"
	yaml "github.com/goccy/go-yaml"
	"github.com/googleapis/genai-toolbox/internal/log"
)

var _ yaml.InterfaceUnmarshalerContext = &DelayedUnmarshaler{}

// DelayedUnmarshaler is struct that saves the provided unmarshal function
// passed to UnmarshalYAML so it can be re-used later once the target interface
// is known.
type DelayedUnmarshaler struct {
	unmarshal func(interface{}) error
}

func (d *DelayedUnmarshaler) UnmarshalYAML(ctx context.Context, unmarshal func(interface{}) error) error {
	d.unmarshal = unmarshal
	return nil
}

func (d *DelayedUnmarshaler) Unmarshal(v interface{}) error {
	if d.unmarshal == nil {
		return fmt.Errorf("nothing to unmarshal")
	}
	return d.unmarshal(v)
}

type contextKey string

// UserAgentKey is the key used to store userAgent within context
const UserAgentKey contextKey = "userAgent"

func NewStrictDecoder(v interface{}) (*yaml.Decoder, error) {
	b, err := yaml.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("fail to marshal %q: %w", v, err)
	}

	dec := yaml.NewDecoder(
		bytes.NewReader(b),
		yaml.Strict(),
		yaml.Validator(validator.New()),
	)
	return dec, nil
}

// loggerKey is the key used to store logger within context
const loggerKey contextKey = "logger"

// WithLogger adds a logger into the context as a value
func WithLogger(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext retreives the logger or return an error
func LoggerFromContext(ctx context.Context) (log.Logger, error) {
	if logger, ok := ctx.Value(loggerKey).(log.Logger); ok {
		return logger, nil
	}
	return nil, fmt.Errorf("unable to retrieve logger")
}
