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

/* Configurations for RunToolInvokeTest()  */

// InvokeTestConfig represents the various configuration options for RunToolInvokeTest()
type InvokeTestConfig struct {
	select1Want              string
	myToolId3NameAliceWant   string
	myToolById4Want          string
	nullWant                 string
	supportOptionalNullParam bool
	supportArrayParam        bool
}

type InvokeTestOption func(*InvokeTestConfig)

// NewInvokeTestConfig creates a new InvokeTestConfig instances with options.
// If the source config differs from the default values, use the associated function to
// update these values.
// e.g. invokeTestConfigs := NewInvokeTestConfig(
//
//	   WithSelect1Want("custom select1 response"),
//	)
func NewInvokeTestConfig(options ...InvokeTestOption) *InvokeTestConfig {
	// default values for InvokeTestConfig
	invokeTestOption := &InvokeTestConfig{
		select1Want:              "",
		myToolId3NameAliceWant:   "[{\"id\":1,\"name\":\"Alice\"},{\"id\":3,\"name\":\"Sid\"}]",
		myToolById4Want:          "[{\"id\":4,\"name\":null}]",
		nullWant:                 "null",
		supportOptionalNullParam: true,
		supportArrayParam:        true,
	}

	// Apply provided options
	for _, option := range options {
		option(invokeTestOption)
	}

	return invokeTestOption
}

// WithSelect1Want represents the response value for select 1 statement.
func WithSelect1Want(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.select1Want = s
	}
}

// WithMyToolId3NameAliceWant represents the response value for my-tool with id=3 and name=Alice.
func WithMyToolId3NameAliceWant(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.myToolId3NameAliceWant = s
	}
}

// WithMyToolById4Want represents the response value for my-tool-by-id with id=4.
// This response includes a null value column.
func WithMyToolById4Want(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.myToolById4Want = s
	}
}

// WithNullWant represents a response value of null string.
func WithNullWant(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.nullWant = s
	}
}

// DisableOptionalNullParamTest disables tests for optional null parameters.
func DisableOptionalNullParamTest() InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.supportOptionalNullParam = false
	}
}

// DisableArrayTest disables tests for sources that do not support array.
func DisableArrayTest() InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.supportArrayParam = false
	}
}

/* Configurations for RunMCPToolCallMethod()  */

// MCPTestConfig represents the various configuration options for mcp tool call tests.
type MCPTestConfig struct {
	myToolId3NameAliceWant string
	myFailToolWant         string
}

type McpTestOption func(*MCPTestConfig)

// NewMCPTestConfig creates a new ExecuteSqlTestConfig instances with options.
// If the source config differs from the default values, use the associated function to
// update these values.
// e.g. mcpTestConfigs := NewMCPTestConfig(
//
//	    WithMcpMyToolId3NameAliceWant("custom my-tool response"),
//	)
func NewMCPTestConfig(options ...McpTestOption) *MCPTestConfig {
	// default values for MCPTestConfig
	mcpTestOption := &MCPTestConfig{
		myToolId3NameAliceWant: `{"jsonrpc":"2.0","id":"my-tool","result":{"content":[{"type":"text","text":"{\"id\":1,\"name\":\"Alice\"}"},{"type":"text","text":"{\"id\":3,\"name\":\"Sid\"}"}]}}`,
		myFailToolWant:         "",
	}

	// Apply provided options
	for _, option := range options {
		option(mcpTestOption)
	}

	return mcpTestOption
}

// WithMcpMyToolId3NameAliceWant represents the response value for my-tool with id=3 and name=Alice.
func WithMcpMyToolId3NameAliceWant(s string) McpTestOption {
	return func(c *MCPTestConfig) {
		c.myToolId3NameAliceWant = s
	}
}

// WithMyFailToolWant respresents the response value for my-fail-tool.
func WithMyFailToolWant(s string) McpTestOption {
	return func(c *MCPTestConfig) {
		c.myFailToolWant = s
	}
}

/* Configurations for RunExecuteSqlToolInvokeTest()  */

// ExecuteSqlTestConfig represents the various configuration options for RunExecuteSqlToolInvokeTest()
type ExecuteSqlTestConfig struct {
	select1Statement     string
	createTableStatement string
	select1Want          string
}

type ExecuteSqlOption func(*ExecuteSqlTestConfig)

// NewExecuteSqlTestConfig creates a new ExecuteSqlTestConfig instances with options.
// If the source config differs from the default values, use the associated function to
// update these values.
// e.g. executeSqlTestConfigs := NewExecuteSqlTestConfig(
//
//	    WithExecSqlSelect1Statement("custom select 1 statement"),
//	)
func NewExecuteSqlTestConfig(options ...ExecuteSqlOption) *ExecuteSqlTestConfig {
	// default values for ExecuteSqlTestConfig
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

// WithExecSqlSelect1Statement represents the database's statement for `SELECT 1`.
func WithExecSqlSelect1Statement(s string) ExecuteSqlOption {
	return func(c *ExecuteSqlTestConfig) {
		c.select1Statement = s
	}
}

// WithCreateTableStatement represents the database's statement for creating a new table.
func WithCreateTableStatement(s string) ExecuteSqlOption {
	return func(c *ExecuteSqlTestConfig) {
		c.createTableStatement = s
	}
}

// WithExecSqlSelect1Want represents the response value for select 1 statement.
func WithExecSqlSelect1Want(s string) ExecuteSqlOption {
	return func(c *ExecuteSqlTestConfig) {
		c.select1Want = s
	}
}

/* Configurations for RunToolInvokeWithTemplateParameters()  */

// TemplateParameterTestConfig represents the various configuration options for template parameter tests.
type TemplateParameterTestConfig struct {
	ddlWant         string
	selectAllWant   string
	selectId1Want   string
	selectEmptyWant string
	insert1Want     string

	nameFieldArray string
	nameColFilter  string
	createColArray string

	supportDdl    bool
	supportInsert bool
}

type TemplateParamOption func(*TemplateParameterTestConfig)

// NewTemplateParameterTestConfig creates a new TemplateParameterTestConfig instances with options.
// If the source config differs from the default values, use the associated function to
// update these values.
// e.g. templateParamTestConfigs := NewTemplateParameterTestConfig(
//
//	    WithDdlWant("custom ddl response"),
//	)
func NewTemplateParameterTestConfig(options ...TemplateParamOption) *TemplateParameterTestConfig {
	// default values for TemplateParameterTestConfig
	templateParamOption := &TemplateParameterTestConfig{
		ddlWant:         "null",
		selectAllWant:   "[{\"age\":21,\"id\":1,\"name\":\"Alex\"},{\"age\":100,\"id\":2,\"name\":\"Alice\"}]",
		selectId1Want:   "[{\"age\":21,\"id\":1,\"name\":\"Alex\"}]",
		selectEmptyWant: "null",
		insert1Want:     "null",

		nameFieldArray: `["name"]`,
		nameColFilter:  "name",
		createColArray: `["id INT","name VARCHAR(20)","age INT"]`,

		supportDdl:    true,
		supportInsert: true,
	}

	// Apply provided options
	for _, option := range options {
		option(templateParamOption)
	}

	return templateParamOption
}

// WithDdlWant represents the response value of ddl statements.
func WithDdlWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.ddlWant = s
	}
}

// WithSelectAllWant represents the response value of select-templateParams-tool.
func WithSelectAllWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectAllWant = s
	}
}

// WithTmplSelectId1Want represents the response value of select-templateParams-combined-tool with id=1.
func WithTmplSelectId1Want(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectId1Want = s
	}
}

// WithSelectEmptyWant represents the response value of select-templateParams-combined-tool with no results.
func WithSelectEmptyWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectEmptyWant = s
	}
}

// WithInsert1Want represents the response value of insert-table-templateParams-tool.
func WithInsert1Want(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.insert1Want = s
	}
}

// WithNameFieldArray represents fields array parameter for select-fields-templateParams-tool.
func WithNameFieldArray(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.nameFieldArray = s
	}
}

// WithNameColFilter represents the columnFilter parameter for select-filter-templateParams-combined-tool.
func WithNameColFilter(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.nameColFilter = s
	}
}

// WithCreateColArray represents the columns array parameter for create-table-templateParams-tool.
func WithCreateColArray(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.createColArray = s
	}
}

// DisableDdlTest disables tests for ddl statements for sources that do not support ddl.
func DisableDdlTest() TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.supportDdl = false
	}
}

// DisableInsertTest disables tests of insert statements for sources that do not support insert.
func DisableInsertTest() TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.supportInsert = false
	}
}
