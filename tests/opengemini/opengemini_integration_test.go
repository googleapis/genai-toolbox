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
package opengemini

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	opengeminisrc "github.com/googleapis/genai-toolbox/internal/sources/opengemini"
	"github.com/googleapis/genai-toolbox/internal/testutils"
	"github.com/googleapis/genai-toolbox/internal/tools"
	_ "github.com/googleapis/genai-toolbox/internal/tools/opengemini/opengeminisql"
	"github.com/googleapis/genai-toolbox/tests"
	"github.com/openGemini/opengemini-client-go/opengemini"
)

var (
	OpenGeminiSourceKind      = "opengemini"
	OpenGeminiToolKind        = "opengemini-sql"
	OpenGeminiDatabase        = os.Getenv("OPENGEMINI_DATABASE")
	OpenGeminiRetentionPolicy = os.Getenv("OPENGEMINI_RETENTIONPOLICY")
	OpenGeminiHost            = os.Getenv("OPENGEMINI_HOST")
	OpenGeminiPort            = os.Getenv("OPENGEMINI_PORT")
	OpenGeminiUser            = os.Getenv("OPENGEMINI_USER")
	OpenGeminiPass            = os.Getenv("OPENGEMINI_PASS")
	OpenGeminiToken           = os.Getenv("OPENGEMINI_Token")
	/*
		0, 1, 2
		0 means no authorization is required

		1 means user and password authorization is required

		2 means token authorization is required
	*/
	OpenGeminiAuthType = os.Getenv("OPENGEMINI_AUTHTYPE")
)

func getOpenGeminiVars(t *testing.T) map[string]any {
	// OpenGeminiDatabase = "db9"
	// OpenGeminiRetentionPolicy = "autogen"
	// OpenGeminiHost = "127.0.0.1"
	// OpenGeminiPort = "8086"
	// OpenGeminiAuthType = "0"
	switch "" {
	case OpenGeminiDatabase:
		t.Fatal("'OPENGEMINI_DATABASE' not set")
	case OpenGeminiRetentionPolicy:
		t.Fatal("'OPENGEMINI_RETENTIONPOLICY' not set")
	case OpenGeminiHost:
		t.Fatal("'OPENGEMINI_HOST' not set")
	case OpenGeminiPort:
		t.Fatal("'OPENGEMINI_PORT' not set")
	case OpenGeminiAuthType:
		t.Fatal("'OPENGEMINI_AUTHTYPE' not set")
		// case OpenGeminiUser:
		// 	t.Fatal("'OPENGEMINI_USER' not set")
		// case OpenGeminiPass:
		// 	t.Fatal("'OPENGEMINI_PASS' not set")
	}

	return map[string]any{
		"kind":            OpenGeminiSourceKind,
		"host":            OpenGeminiHost,
		"port":            OpenGeminiPort,
		"database":        OpenGeminiDatabase,
		"retentionpolicy": OpenGeminiRetentionPolicy,
		"user":            OpenGeminiUser,
		"password":        OpenGeminiPass,
	}
}

func initOpenGeminiClient(host, port, user, pass, token string) (opengemini.Client, error) {
	var authConfig *opengemini.AuthConfig

	tp, err := strconv.Atoi(OpenGeminiAuthType)
	if err != nil {
		return nil, fmt.Errorf("incorrect authtype: %s", err)
	}

	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("incorrect port: %s", err)
	}

	if tp == opengeminisrc.AuthTypePwd || len(user) != 0 {
		authConfig = &opengemini.AuthConfig{
			AuthType: opengemini.AuthTypePassword,
			Username: user,
			Password: pass,
		}
	} else if tp == opengeminisrc.AuthTypeToken || len(token) != 0 {
		authConfig = &opengemini.AuthConfig{
			AuthType: opengemini.AuthTypeToken,
			Token:    token,
		}
	}

	config := &opengemini.Config{
		Addresses: []opengemini.Address{
			{
				Host: host,
				Port: p,
			},
		},
		AuthConfig: authConfig,
	}

	client, err := opengemini.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to opengemini client: %w", err)
	}

	if err = client.Ping(0); err != nil {
		return nil, fmt.Errorf("unable to connect to opengemini: %s", err)
	}

	return client, nil
}

func getOpenGeminiParamToolInfo(measurementName string) (string, string, string, string, string, string, string, []any) {
	createStatement := fmt.Sprintf("CREATE MEASUREMENT %s (id tag, temp float64, rate float64) WITH ENGINETYPE = columnstore PRIMARYKEY id", measurementName)
	insertStatement := "" // insertStatement := fmt.Sprintf("INSERT %s,id=$id temp=$temp,rate=$rate", measurementName)  // openGemini not support
	toolStatement := fmt.Sprintf("SELECT * FROM %s WHERE temp=$temp OR rate=$rate", measurementName)
	idParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE id=$id", measurementName)
	tempParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE temp=$temp", measurementName)
	rateParamStatement := fmt.Sprintf("SELECT * FROM %s WHERE rate=$rate", measurementName)
	arrayToolStatement := fmt.Sprintf("SELECT * FROM %s WHERE temp=$temp AND rate=$rate", measurementName)
	params := []any{map[string]any{"id": "centos-1", "temp": 44.7, "rate": 75.4}, map[string]any{"id": "ubuntu-2", "temp": 37.2, "rate": 17.0}}

	return createStatement, insertStatement, toolStatement, idParamStatement, tempParamStatement, rateParamStatement, arrayToolStatement, params
}

func getOpenGeminiTmplToolStatement() (string, string) {
	tmplSelectCombined := "SELECT * FROM {{.tableName}} WHERE id=$id"
	tmplSelectFilterCombined := "SELECT * FROM {{.tableName}} WHERE {{.columnFilter}}=$columnFilter"
	return tmplSelectCombined, tmplSelectFilterCombined
}

func SetupOpenGeminiTable(t *testing.T, ctx context.Context, client opengemini.Client, createStatement, insertStatement, tableName string, params []any) func(*testing.T) {
	// create measurement
	_, err := client.Query(opengemini.Query{
		Database:        OpenGeminiDatabase,
		Command:         createStatement,
		RetentionPolicy: OpenGeminiRetentionPolicy,
	})
	if err != nil {
		t.Fatalf("unable to create test measurement %s: %s", tableName, err)
	}

	// Insert test data
	for _, param := range params {
		point := &opengemini.Point{
			Measurement: tableName,
		}
		p := param.(map[string]any)
		for k, v := range p {
			if k == "id" {
				point.AddTag(k, v.(string))
			} else {
				point.AddField(k, v)
			}
		}
		err := client.WritePoint(OpenGeminiDatabase, point, func(err error) {
			if err != nil {
				fmt.Printf("write point failed for %s", err)
			}
		})
		if err != nil {
			t.Fatalf("unable to insert test data: %s", err)
		}
	}

	return func(t *testing.T) {
		// tear down test
		err := client.DropMeasurement(OpenGeminiDatabase, OpenGeminiRetentionPolicy, tableName)
		if err != nil {
			t.Errorf("Teardown failed: %s", err)
		}
	}
}

func TestOpenGemini(t *testing.T) {
	sourceConfig := getOpenGeminiVars(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	var args []string

	client, err := initOpenGeminiClient(OpenGeminiHost, OpenGeminiPort, OpenGeminiUser, OpenGeminiPass, OpenGeminiToken)
	if err != nil {
		t.Fatalf("unable to connect to opengemini client: %s", err)
	}

	// create table name with UUID
	tableNameParam := "param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	// tableNameTemplateParam := "template_param_table_" + strings.ReplaceAll(uuid.New().String(), "-", "")

	// set up data for param tool
	createParamTableStmt, insertParamTableStmt, paramToolStmt, idParamToolStmt, tempParamStmt, rateParamStmt, arrayToolStmt, paramTestParams := getOpenGeminiParamToolInfo(tableNameParam)
	teardownTable1 := SetupOpenGeminiTable(t, ctx, client, createParamTableStmt, insertParamTableStmt, tableNameParam, paramTestParams)
	defer teardownTable1(t)

	// Write config into a file and pass it to command
	toolsFile := getOpenGeminiToolsConfig(sourceConfig, OpenGeminiToolKind, paramToolStmt, idParamToolStmt, tempParamStmt, arrayToolStmt, rateParamStmt)

	tmplSelectCombined, tmplSelectFilterCombined := getOpenGeminiTmplToolStatement()
	toolsFile = addOpenGeminiTemplateParamConfig(t, toolsFile, OpenGeminiToolKind, tmplSelectCombined, tmplSelectFilterCombined)

	cmd, cleanup, err := tests.StartCmd(ctx, toolsFile, args...)
	if err != nil {
		t.Fatalf("command initialization returned an error: %s", err)
	}
	defer cleanup()

	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := testutils.WaitForString(waitCtx, regexp.MustCompile(`Server ready to serve`), cmd.Out)
	if err != nil {
		t.Logf("toolbox command logs: \n%s", out)
		t.Fatalf("toolbox didn't start successfully: %s", err)
	}
}

func getOpenGeminiToolsConfig(sourceConfig map[string]any, toolKind, paramToolStatement, idParamToolStmt, tempParamToolStmt, arrayToolStmt, rateParamToolStmt string) map[string]any {
	// Write config into a file and pass it to command
	toolsFile := map[string]any{
		"sources": map[string]any{
			"my-instance": sourceConfig,
		},
		"tools": map[string]any{
			"my-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   paramToolStatement,
				"parameters": []any{
					map[string]any{
						"name":        "temp",
						"type":        "float",
						"description": "user ID",
					},
					map[string]any{
						"name":        "rate",
						"type":        "float",
						"description": "user name",
					},
				},
			},
			"my-tool-by-id": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   idParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "id",
						"type":        "string",
						"description": "cpu id",
					},
				},
			},
			"my-tool-by-temp": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   tempParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "temp",
						"type":        "float",
						"description": "the temperature of cpu",
						"required":    false,
					},
				},
			},
			"my-tool-by-rate": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with params.",
				"statement":   rateParamToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "rate",
						"type":        "float",
						"description": "the rate of cpu usage",
						"required":    false,
					},
				},
			},
			"my-array-tool": map[string]any{
				"kind":        toolKind,
				"source":      "my-instance",
				"description": "Tool to test invocation with array params.",
				"statement":   arrayToolStmt,
				"parameters": []any{
					map[string]any{
						"name":        "tempArray",
						"type":        "array",
						"description": "temperature array",
						"items": map[string]any{
							"name":        "temp",
							"type":        "float",
							"description": "temperature",
						},
					},
					map[string]any{
						"name":        "rateArray",
						"type":        "array",
						"description": "rate array",
						"items": map[string]any{
							"name":        "rate",
							"type":        "float",
							"description": "rate",
						},
					},
				},
			},
		},
	}

	return toolsFile
}

func addOpenGeminiTemplateParamConfig(t *testing.T, config map[string]any, toolKind, tmplSelectCombined, tmplSelectFilterCombined string) map[string]any {
	toolsMap, ok := config["tools"].(map[string]any)
	if !ok {
		t.Fatalf("unable to get tools from config")
	}

	toolsMap["select-templateParams-combined-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   tmplSelectCombined,
		"parameters":  []tools.Parameter{tools.NewStringParameter("id", "the id of the cpu")},
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
		},
	}

	toolsMap["select-filter-templateParams-combined-tool"] = map[string]any{
		"kind":        toolKind,
		"source":      "my-instance",
		"description": "Create table tool with template parameters",
		"statement":   tmplSelectFilterCombined,
		"parameters":  []tools.Parameter{tools.NewFloatParameter("temp", "the temperature of the cpu")},
		"templateParameters": []tools.Parameter{
			tools.NewStringParameter("tableName", "some description"),
			tools.NewStringParameter("columnFilter", "some description"),
		},
	}

	config["tools"] = toolsMap
	return config
}
