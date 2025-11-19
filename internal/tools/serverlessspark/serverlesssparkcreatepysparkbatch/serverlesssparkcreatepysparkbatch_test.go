// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not- use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package serverlesssparkcreatepysparkbatch_test

import (
	"testing"

	"github.com/googleapis/genai-toolbox/internal/tools"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/createbatch"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/serverlesssparkcreatepysparkbatch"
	"github.com/googleapis/genai-toolbox/internal/tools/serverlessspark/testutils"
)

func TestParseFromYaml(t *testing.T) {
	testutils.RunParseFromYAMLTests(t, "serverless-spark-create-pyspark-batch", func(c createbatch.Config) tools.ToolConfig {
		return serverlesssparkcreatepysparkbatch.Config{Config: c}
	})
}
