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

package cloudgda

// QueryDataRequest represents the JSON body for the queryData API
type QueryDataRequest struct {
	Parent            string             `json:"parent"`
	Prompt            string             `json:"prompt"`
	Context           *QueryDataContext  `json:"context,omitempty"`
	GenerationOptions *GenerationOptions `json:"generationOptions,omitempty"`
}

// QueryDataContext reflects the proto definition for the query context.
type QueryDataContext struct {
	DatasourceReferences *DatasourceReferences `json:"datasourceReferences,omitempty" yaml:"datasourceReferences,omitempty"`
}

// DatasourceReferences reflects the proto definition for datasource references, using a oneof.
type DatasourceReferences struct {
	SpannerReference  *SpannerReference  `json:"spannerReference,omitempty" yaml:"spannerReference,omitempty"`
	AlloyDBReference  *AlloyDBReference  `json:"alloydb,omitempty" yaml:"alloydb,omitempty"`
	CloudSQLReference *CloudSQLReference `json:"cloudSqlReference,omitempty" yaml:"cloudSqlReference,omitempty"`
}

// SpannerReference reflects the proto definition for Spanner database reference.
type SpannerReference struct {
	DatabaseReference     *SpannerDatabaseReference `json:"databaseReference,omitempty" yaml:"databaseReference,omitempty"`
	AgentContextReference *AgentContextReference    `json:"agentContextReference,omitempty" yaml:"agentContextReference,omitempty"`
}

// SpannerDatabaseReference reflects the proto definition for a Spanner database reference.
type SpannerDatabaseReference struct {
	Engine     SpannerEngine `json:"engine,omitempty" yaml:"engine,omitempty"`
	ProjectID  string        `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	Region     string        `json:"region,omitempty" yaml:"region,omitempty"`
	InstanceID string        `json:"instanceId,omitempty" yaml:"instanceId,omitempty"`
	DatabaseID string        `json:"databaseId,omitempty" yaml:"databaseId,omitempty"`
	TableIDs   []string      `json:"tableIds,omitempty" yaml:"tableIds,omitempty"`
}

// SpannerEngine represents the engine of the Spanner instance.
type SpannerEngine string

const (
	SpannerEngineUnspecified SpannerEngine = "ENGINE_UNSPECIFIED"
	SpannerEngineGoogleSQL   SpannerEngine = "GOOGLE_SQL"
	SpannerEnginePostgreSQL  SpannerEngine = "POSTGRESQL"
)

// AlloyDBReference reflects the proto definition for an AlloyDB database reference.
type AlloyDBReference struct {
	DatabaseReference     *AlloyDBDatabaseReference `json:"databaseReference,omitempty" yaml:"databaseReference,omitempty"`
	AgentContextReference *AgentContextReference    `json:"agentContextReference,omitempty" yaml:"agentContextReference,omitempty"`
}

// AlloyDBDatabaseReference reflects the proto definition for an AlloyDB database reference.
type AlloyDBDatabaseReference struct {
	ProjectID  string   `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	Region     string   `json:"region,omitempty" yaml:"region,omitempty"`
	ClusterID  string   `json:"clusterId,omitempty" yaml:"clusterId,omitempty"`
	InstanceID string   `json:"instanceId,omitempty" yaml:"instanceId,omitempty"`
	DatabaseID string   `json:"databaseId,omitempty" yaml:"databaseId,omitempty"`
	TableIDs   []string `json:"tableIds,omitempty" yaml:"tableIds,omitempty"`
}

// CloudSQLReference reflects the proto definition for a Cloud SQL database reference.
type CloudSQLReference struct {
	DatabaseReference     *CloudSQLDatabaseReference `json:"databaseReference,omitempty" yaml:"databaseReference,omitempty"`
	AgentContextReference *AgentContextReference     `json:"agentContextReference,omitempty" yaml:"agentContextReference,omitempty"`
}

// CloudSQLDatabaseReference reflects the proto definition for a Cloud SQL database reference.
type CloudSQLDatabaseReference struct {
	Engine     CloudSQLEngine `json:"engine,omitempty" yaml:"engine,omitempty"`
	ProjectID  string         `json:"projectId,omitempty" yaml:"projectId,omitempty"`
	Region     string         `json:"region,omitempty" yaml:"region,omitempty"`
	InstanceID string         `json:"instanceId,omitempty" yaml:"instanceId,omitempty"`
	DatabaseID string         `json:"databaseId,omitempty" yaml:"databaseId,omitempty"`
	TableIDs   []string       `json:"tableIds,omitempty" yaml:"tableIds,omitempty"`
}

// CloudSQLEngine represents the engine of the Cloud SQL instance.
type CloudSQLEngine string

const (
	CloudSQLEngineUnspecified CloudSQLEngine = "ENGINE_UNSPECIFIED"
	CloudSQLEnginePostgreSQL  CloudSQLEngine = "POSTGRESQL"
	CloudSQLEngineMySQL       CloudSQLEngine = "MYSQL"
)

// AgentContextReference reflects the proto definition for agent context.
type AgentContextReference struct {
	ContextSetID string `json:"contextSetId,omitempty" yaml:"contextSetId,omitempty"`
}

// GenerationOptions reflects the proto definition for generation options.
type GenerationOptions struct {
	GenerateQueryResult            bool `json:"generateQueryResult" yaml:"generateQueryResult"`
	GenerateNaturalLanguageAnswer  bool `json:"generateNaturalLanguageAnswer" yaml:"generateNaturalLanguageAnswer"`
	GenerateExplanation            bool `json:"generateExplanation" yaml:"generateExplanation"`
	GenerateDisambiguationQuestion bool `json:"generateDisambiguationQuestion" yaml:"generateDisambiguationQuestion"`
}
