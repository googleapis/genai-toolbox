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

package classifier

import (
	"regexp"
	"sort"
	"strings"
)

// QueryType represents the classification of a Cypher query
type QueryType int

const (
	ReadQuery QueryType = iota
	WriteQuery
)

func (qt QueryType) String() string {
	if qt == ReadQuery {
		return "READ"
	}
	return "WRITE"
}

// QueryClassification represents the result of query classification
type QueryClassification struct {
	Type        QueryType
	Confidence  float64
	WriteTokens []string
	ReadTokens  []string
	HasSubquery bool
	Error       error
}

// QueryClassifier classifies Cypher queries into read or write operations
type QueryClassifier struct {
	writeKeywords          map[string]struct{}
	readKeywords           map[string]struct{}
	writeProcedures        map[string]struct{}
	readProcedures         map[string]struct{}
	multiWordWriteKeywords []string
	multiWordReadKeywords  []string
	commentPattern         *regexp.Regexp
	stringLiteralPattern   *regexp.Regexp
	procedureCallPattern   *regexp.Regexp
	subqueryPattern        *regexp.Regexp
	whitespacePattern      *regexp.Regexp
	tokenSplitPattern      *regexp.Regexp
}

// NewQueryClassifier creates a new QueryClassifier instance
func NewQueryClassifier() *QueryClassifier {
	c := &QueryClassifier{
		writeKeywords:        make(map[string]struct{}),
		readKeywords:         make(map[string]struct{}),
		writeProcedures:      make(map[string]struct{}),
		readProcedures:       make(map[string]struct{}),
		commentPattern:       regexp.MustCompile(`(?m)//.*?$|/\*[\s\S]*?\*/`),
		stringLiteralPattern: regexp.MustCompile(`'[^']*'|"[^"]*"`),
		procedureCallPattern: regexp.MustCompile(`(?i)\bCALL\s+([a-zA-Z0-9_.]+)`),
		subqueryPattern:      regexp.MustCompile(`(?i)\bCALL\s*\{`),
		whitespacePattern:    regexp.MustCompile(`\s+`),
		tokenSplitPattern:    regexp.MustCompile(`[\s,(){}[\]]+`),
	}

	writeKeywordsList := []string{
		"CREATE", "MERGE", "DELETE", "DETACH DELETE", "SET", "REMOVE", "FOREACH",
		"CREATE INDEX", "DROP INDEX", "CREATE CONSTRAINT", "DROP CONSTRAINT",
	}
	readKeywordsList := []string{
		"MATCH", "OPTIONAL MATCH", "WITH", "WHERE", "RETURN", "ORDER BY", "SKIP", "LIMIT",
		"UNION", "UNION ALL", "UNWIND", "CASE", "WHEN", "THEN", "ELSE", "END",
		"SHOW", "PROFILE", "EXPLAIN",
	}
	writeProceduresList := []string{
		"apoc.create", "apoc.merge", "apoc.refactor", "apoc.atomic", "apoc.trigger",
		"apoc.periodic.commit", "apoc.load.jdbc", "apoc.load.json", "apoc.load.csv",
		"apoc.export", "apoc.import", "db.create", "db.drop", "db.index.create",
		"db.constraints.create", "dbms.security.create", "gds.graph.create", "gds.graph.drop",
	}
	readProceduresList := []string{
		"apoc.meta", "apoc.help", "apoc.version", "apoc.text", "apoc.math", "apoc.coll",
		"apoc.path", "apoc.algo", "apoc.date", "db.labels", "db.propertyKeys",
		"db.relationshipTypes", "db.schema", "db.indexes", "db.constraints",
		"dbms.components", "dbms.listConfig", "gds.graph.list", "gds.util",
	}

	c.populateKeywords(writeKeywordsList, c.writeKeywords, &c.multiWordWriteKeywords)
	c.populateKeywords(readKeywordsList, c.readKeywords, &c.multiWordReadKeywords)
	c.populateProcedures(writeProceduresList, c.writeProcedures)
	c.populateProcedures(readProceduresList, c.readProcedures)

	return c
}

func (c *QueryClassifier) populateKeywords(keywords []string, keywordMap map[string]struct{}, multiWord *[]string) {
	for _, kw := range keywords {
		if strings.Contains(kw, " ") {
			*multiWord = append(*multiWord, kw)
		}
		keywordMap[strings.ReplaceAll(kw, " ", "_")] = struct{}{}
	}
	sort.SliceStable(*multiWord, func(i, j int) bool {
		return len((*multiWord)[i]) > len((*multiWord)[j])
	})
}

func (c *QueryClassifier) populateProcedures(procedures []string, procedureMap map[string]struct{}) {
	for _, proc := range procedures {
		procedureMap[proc] = struct{}{}
	}
}

func (c *QueryClassifier) Classify(query string) QueryClassification {
	result := QueryClassification{
		Type:       ReadQuery,
		Confidence: 1.0,
	}

	normalizedQuery := c.normalizeQuery(query)
	if normalizedQuery == "" {
		return result
	}

	result.HasSubquery = c.subqueryPattern.MatchString(normalizedQuery)
	procedures := c.extractProcedureCalls(normalizedQuery)
	sanitizedQuery := c.stringLiteralPattern.ReplaceAllString(normalizedQuery, "STRING_LITERAL")
	unifiedQuery := c.unifyMultiWordKeywords(sanitizedQuery)
	tokens := c.extractTokens(unifiedQuery)

	for _, token := range tokens {
		upperToken := strings.ToUpper(token)
		originalToken := strings.ReplaceAll(upperToken, "_", " ")

		if _, isWrite := c.writeKeywords[upperToken]; isWrite {
			result.WriteTokens = append(result.WriteTokens, originalToken)
			result.Type = WriteQuery
		} else if _, isRead := c.readKeywords[upperToken]; isRead {
			result.ReadTokens = append(result.ReadTokens, originalToken)
		}
	}

	for _, proc := range procedures {
		if c.isWriteProcedure(proc) {
			result.WriteTokens = append(result.WriteTokens, "CALL "+proc)
			result.Type = WriteQuery
		} else if c.isReadProcedure(proc) {
			result.ReadTokens = append(result.ReadTokens, "CALL "+proc)
		} else {
			if strings.Contains(proc, ".get") || strings.Contains(proc, ".list") ||
				strings.Contains(proc, ".show") || strings.Contains(proc, ".meta") {
				result.ReadTokens = append(result.ReadTokens, "CALL "+proc)
			} else {
				result.WriteTokens = append(result.WriteTokens, "CALL "+proc)
				result.Type = WriteQuery
				result.Confidence = 0.8
			}
		}
	}

	if result.HasSubquery && c.hasWriteInSubquery(unifiedQuery) {
		result.Type = WriteQuery
		found := false
		for _, t := range result.WriteTokens {
			if t == "WRITE_IN_SUBQUERY" {
				found = true
				break
			}
		}
		if !found {
			result.WriteTokens = append(result.WriteTokens, "WRITE_IN_SUBQUERY")
		}
	}

	if len(result.WriteTokens) > 0 && len(result.ReadTokens) > 0 {
		result.Confidence = 0.9
	}

	return result
}

func (c *QueryClassifier) unifyMultiWordKeywords(query string) string {
	upperQuery := strings.ToUpper(query)
	allMultiWord := append(c.multiWordWriteKeywords, c.multiWordReadKeywords...)

	for _, kw := range allMultiWord {
		placeholder := strings.ReplaceAll(kw, " ", "_")
		upperQuery = strings.ReplaceAll(upperQuery, kw, placeholder)
	}
	return upperQuery
}

func (c *QueryClassifier) normalizeQuery(query string) string {
	query = c.commentPattern.ReplaceAllString(query, " ")
	query = c.whitespacePattern.ReplaceAllString(query, " ")
	return strings.TrimSpace(query)
}

func (c *QueryClassifier) extractTokens(query string) []string {
	tokens := c.tokenSplitPattern.Split(query, -1)
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if token != "" {
			result = append(result, token)
		}
	}
	return result
}

func (c *QueryClassifier) extractProcedureCalls(query string) []string {
	matches := c.procedureCallPattern.FindAllStringSubmatch(query, -1)
	procedures := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			procedures = append(procedures, strings.ToLower(match[1]))
		}
	}
	return procedures
}

func (c *QueryClassifier) isWriteProcedure(procedure string) bool {
	procedure = strings.ToLower(procedure)
	for wp := range c.writeProcedures {
		if strings.HasPrefix(procedure, wp) {
			return true
		}
	}
	return false
}

func (c *QueryClassifier) isReadProcedure(procedure string) bool {
	procedure = strings.ToLower(procedure)
	for rp := range c.readProcedures {
		if strings.HasPrefix(procedure, rp) {
			return true
		}
	}
	return false
}

func (c *QueryClassifier) hasWriteInSubquery(unifiedQuery string) bool {
	loc := c.subqueryPattern.FindStringIndex(unifiedQuery)
	if loc == nil {
		return false
	}

	subqueryContent := unifiedQuery[loc[0]:]
	openBraces := 0
	startIndex := -1
	endIndex := -1

	for i, char := range subqueryContent {
		if char == '{' {
			if openBraces == 0 {
				startIndex = i + 1
			}
			openBraces++
		} else if char == '}' {
			openBraces--
			if openBraces == 0 {
				endIndex = i
				break
			}
		}
	}

	var block string
	if startIndex != -1 {
		if endIndex != -1 {
			// Found a complete block
			block = subqueryContent[startIndex:endIndex]
		} else {
			// Found an opening brace but no closing one; check the rest of the string
			block = subqueryContent[startIndex:]
		}

		for writeOp := range c.writeKeywords {
			re := regexp.MustCompile(`\b` + writeOp + `\b`)
			if re.MatchString(block) {
				return true
			}
		}
	}

	return false
}

func (c *QueryClassifier) AddWriteProcedure(pattern string) {
	if pattern != "" {
		c.writeProcedures[strings.ToLower(pattern)] = struct{}{}
	}
}

func (c *QueryClassifier) AddReadProcedure(pattern string) {
	if pattern != "" {
		c.readProcedures[strings.ToLower(pattern)] = struct{}{}
	}
}
