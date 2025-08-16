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

package tests

// InvokeTestConfig represents the various configuration options for RunToolInvokeTest()
type InvokeTestConfig struct {
	select1Want      string
	invokeParamWant  string
	invokeIdNullWant string
	nullString       string
	supportNullParam bool
	supportArray     bool
}

type InvokeTestOption func(*InvokeTestConfig)

// NewInvokeTestConfig creates a new InvokeTestConfig instances with options.
func NewInvokeTestConfig(options ...InvokeTestOption) *InvokeTestConfig {
	invokeTestOption := &InvokeTestConfig{
		select1Want:      "",
		invokeParamWant:  "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]",
		invokeIdNullWant: "[{\"id\":4,\"name\":null}]",
		nullString:       "null",
		supportNullParam: true,
		supportArray:     true,
	}

	// Apply provided options
	for _, option := range options {
		option(invokeTestOption)
	}

	return invokeTestOption
}

func WithInvoketestSelect1Want(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.select1Want = s
	}
}

func WithInvokeParamWant(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.invokeParamWant = s
	}
}

func WithInvokeIdNullWant(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.invokeIdNullWant = s
	}
}

func WithNullString(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.nullString = s
	}
}

func WithDisableNullParam() InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.supportNullParam = false
	}
}

func WithDisableArray() InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.supportArray = false
	}
}

// MCPTestConfig represents the various configuration options for mcp tool call tests.
type MCPTestConfig struct {
	invokeParamWant    string
	failInvocationWant string
}

type McpTestOption func(*MCPTestConfig)

// NewMCPTestConfig creates a new ExecuteSqlTestConfig instances with options.
func NewMCPTestConfig(options ...McpTestOption) *MCPTestConfig {
	mcpTestOption := &MCPTestConfig{
		invokeParamWant:    `{"jsonrpc":"2.0","id":"my-tool","result":{"content":[{"type":"text","text":"{\"id\":1,\"name\":\"Alice\"}"},{"type":"text","text":"{\"id\":3,\"name\":\"Sid\"}"}]}}`,
		failInvocationWant: "",
	}

	// Apply provided options
	for _, option := range options {
		option(mcpTestOption)
	}

	return mcpTestOption
}

func WithMcpInvokeParamWant(s string) McpTestOption {
	return func(c *MCPTestConfig) {
		c.invokeParamWant = s
	}
}

func WithFailInvocationWant(s string) McpTestOption {
	return func(c *MCPTestConfig) {
		c.failInvocationWant = s
	}
}

// ExecuteSqlTestConfig represents the various configuration options for RunExecuteSqlToolInvokeTest()
type ExecuteSqlTestConfig struct {
	select1Statement     string
	createTableStatement string
	select1Want          string
}

type ExecuteSqlOption func(*ExecuteSqlTestConfig)

// NewExecuteSqlTestConfig creates a new ExecuteSqlTestConfig instances with options.
func NewExecuteSqlTestConfig(options ...ExecuteSqlOption) *ExecuteSqlTestConfig {
	executeSqlTestOption := &ExecuteSqlTestConfig{
		select1Statement:     `"SELECT 1"`,
		createTableStatement: "",
		select1Want:          "",
	}

	// Apply provided options
	for _, option := range options {
		option(executeSqlTestOption)
	}

	return executeSqlTestOption
}

func WithSelect1Statement(s string) ExecuteSqlOption {
	return func(c *ExecuteSqlTestConfig) {
		c.select1Statement = s
	}
}

func WithCreateTableStatement(s string) ExecuteSqlOption {
	return func(c *ExecuteSqlTestConfig) {
		c.createTableStatement = s
	}
}

func WithExecSqlSelect1Want(s string) ExecuteSqlOption {
	return func(c *ExecuteSqlTestConfig) {
		c.select1Want = s
	}
}

// TemplateParameterTestConfig represents the various configuration options for template parameter tests.
type TemplateParameterTestConfig struct {
	ignoreDdl       bool
	ignoreInsert    bool
	ddlWant         string
	selectAllWant   string
	select1Want     string
	selectEmptyWant string
	nameFieldArray  string
	nameColFilter   string
	createColArray  string
	insert1Want     string
}

type TemplateParamOption func(*TemplateParameterTestConfig)

// NewTemplateParameterTestConfig creates a new TemplateParameterTestConfig instances with options.
func NewTemplateParameterTestConfig(options ...TemplateParamOption) *TemplateParameterTestConfig {
	templateParamOption := &TemplateParameterTestConfig{
		ignoreDdl:       false,
		ignoreInsert:    false,
		ddlWant:         "null",
		selectAllWant:   "[{\"age\":21,\"id\":1,\"name\":\"Alex\"},{\"age\":100,\"id\":2,\"name\":\"Alice\"}]",
		select1Want:     "[{\"age\":21,\"id\":1,\"name\":\"Alex\"}]",
		selectEmptyWant: "null",
		nameFieldArray:  `["name"]`,
		nameColFilter:   "name",
		createColArray:  `["id INT","name VARCHAR(20)","age INT"]`,
		insert1Want:     "null",
	}

	// Apply provided options
	for _, option := range options {
		option(templateParamOption)
	}

	return templateParamOption
}

// WithIgnoreDdl is the option function to configure ignoreDdl.
func WithIgnoreDdl() TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.ignoreDdl = true
	}
}

// WithIgnoreInsert is the option function to configure ignoreInsert.
func WithIgnoreInsert() TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.ignoreInsert = true
	}
}

// WithDdlWant is the option function to configure ddlWant.
func WithDdlWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.ddlWant = s
	}
}

// WithSelectAllWant is the option function to configure selectAllWant.
func WithSelectAllWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectAllWant = s
	}
}

// WithTmplSelect1Want is the option function to configure select1Want.
func WithTmplSelect1Want(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.select1Want = s
	}
}

// WithSelectEmptyWant is the option function to configure selectEmptyWant.
func WithSelectEmptyWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectEmptyWant = s
	}
}

// WithReplaceNameFieldArray is the option function to configure replaceNameFieldArray.
func WithReplaceNameFieldArray(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.nameFieldArray = s
	}
}

// WithReplaceNameColFilter is the option function to configure replaceNameColFilter.
func WithReplaceNameColFilter(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.nameColFilter = s
	}
}

// WithCreateColArray is the option function to configure replaceNameColFilter.
func WithCreateColArray(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.createColArray = s
	}
}

func WithInsert1Want(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.insert1Want = s
	}
}
