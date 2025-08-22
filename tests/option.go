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
	myToolId3NameAliceWant   string
	myToolById4Want          string
	nullWant                 string
	supportOptionalNullParam bool
	supportArrayParam        bool
	supportClientAuth        bool
}

type InvokeTestOption func(*InvokeTestConfig)

// WithMyToolId3NameAliceWant represents the response value for my-tool with id=3 and name=Alice.
// e.g. tests.RunToolInvokeTest(t, select1Want, tests.WithMyToolId3NameAliceWant("custom"))
func WithMyToolId3NameAliceWant(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.myToolId3NameAliceWant = s
	}
}

// WithMyToolById4Want represents the response value for my-tool-by-id with id=4.
// This response includes a null value column.
// e.g. tests.RunToolInvokeTest(t, select1Want, tests.WithMyToolById4Want("custom"))
func WithMyToolById4Want(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.myToolById4Want = s
	}
}

// WithNullWant represents a response value of null string.
// e.g. tests.RunToolInvokeTest(t, select1Want, tests.WithNullWant("custom"))
func WithNullWant(s string) InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.nullWant = s
	}
}

// DisableOptionalNullParamTest disables tests for optional null parameters.
// e.g. tests.RunToolInvokeTest(t, select1Want, tests.DisableOptionalNullParamTest())
func DisableOptionalNullParamTest() InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.supportOptionalNullParam = false
	}
}

// DisableArrayTest disables tests for sources that do not support array.
// e.g. tests.RunToolInvokeTest(t, select1Want, tests.DisableArrayTest())
func DisableArrayTest() InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.supportArrayParam = false
	}
}

// DisableClientAuthTest disables tests for sources that do not support client auth.
// e.g. tests.RunToolInvokeTest(t, select1Want, tests.DisableClientAuthTest())
func EnableClientAuthTest() InvokeTestOption {
	return func(c *InvokeTestConfig) {
		c.supportClientAuth = true
	}
}

/* Configurations for RunMCPToolCallMethod()  */

// MCPTestConfig represents the various configuration options for mcp tool call tests.
type MCPTestConfig struct {
	myToolId3NameAliceWant string
	supportClientAuth      bool
}

type McpTestOption func(*MCPTestConfig)

// WithMcpMyToolId3NameAliceWant represents the response value for my-tool with id=3 and name=Alice.
// e.g. tests.RunMCPToolCallMethod(t, mcpMyFailToolWant, tests.WithMcpMyToolId3NameAliceWant("custom"))
func WithMcpMyToolId3NameAliceWant(s string) McpTestOption {
	return func(c *MCPTestConfig) {
		c.myToolId3NameAliceWant = s
	}
}

func EnableMcpClientAuthTest() McpTestOption {
	return func(c *MCPTestConfig) {
		c.supportClientAuth = true
	}
}

/* Configurations for RunExecuteSqlToolInvokeTest()  */

// ExecuteSqlTestConfig represents the various configuration options for RunExecuteSqlToolInvokeTest()
type ExecuteSqlTestConfig struct {
	select1Statement string
}

type ExecuteSqlOption func(*ExecuteSqlTestConfig)

// WithSelect1Statement represents the database's statement for `SELECT 1`.
// e.g. tests.RunExecuteSqlToolInvokeTest(t, createTableStatement, select1Want, tests.WithSelect1Statement("custom"))
func WithSelect1Statement(s string) ExecuteSqlOption {
	return func(c *ExecuteSqlTestConfig) {
		c.select1Statement = s
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

// WithDdlWant represents the response value of ddl statements.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithDdlWant("custom"))
func WithDdlWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.ddlWant = s
	}
}

// WithSelectAllWant represents the response value of select-templateParams-tool.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithSelectAllWant("custom"))
func WithSelectAllWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectAllWant = s
	}
}

// WithTmplSelectId1Want represents the response value of select-templateParams-combined-tool with id=1.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithTmplSelectId1Want("custom"))
func WithTmplSelectId1Want(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectId1Want = s
	}
}

// WithSelectEmptyWant represents the response value of select-templateParams-combined-tool with no results.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithSelectEmptyWant("custom"))
func WithSelectEmptyWant(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.selectEmptyWant = s
	}
}

// WithInsert1Want represents the response value of insert-table-templateParams-tool.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithInsert1Want("custom"))
func WithInsert1Want(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.insert1Want = s
	}
}

// WithNameFieldArray represents fields array parameter for select-fields-templateParams-tool.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithNameFieldArray("custom"))
func WithNameFieldArray(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.nameFieldArray = s
	}
}

// WithNameColFilter represents the columnFilter parameter for select-filter-templateParams-combined-tool.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithNameColFilter("custom"))
func WithNameColFilter(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.nameColFilter = s
	}
}

// WithCreateColArray represents the columns array parameter for create-table-templateParams-tool.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.WithCreateColArray("custom"))
func WithCreateColArray(s string) TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.createColArray = s
	}
}

// DisableDdlTest disables tests for ddl statements for sources that do not support ddl.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.DisableDdlTest())
func DisableDdlTest() TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.supportDdl = false
	}
}

// DisableInsertTest disables tests of insert statements for sources that do not support insert.
// e.g. tests.RunToolInvokeWithTemplateParameters(t, tableNameTemplateParam, tests.DisableInsertTest())
func DisableInsertTest() TemplateParamOption {
	return func(c *TemplateParameterTestConfig) {
		c.supportInsert = false
	}
}
