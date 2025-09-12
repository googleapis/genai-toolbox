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

package bigquerycommon

import (
	"fmt"
	"strings"
	"unicode"
)

// parserState defines the state of the SQL parser's state machine.
type parserState int

const (
	stateNormal parserState = iota
	// String states
	stateInSingleQuoteString
	stateInDoubleQuoteString
	stateInTripleSingleQuoteString
	stateInTripleDoubleQuoteString
	stateInRawSingleQuoteString
	stateInRawDoubleQuoteString
	stateInRawTripleSingleQuoteString
	stateInRawTripleDoubleQuoteString
	// Comment states
	stateInSingleLineCommentDash
	stateInSingleLineCommentHash
	stateInMultiLineComment
)

// SQL statement verbs
const (
	verbCreate = "create"
	verbAlter  = "alter"
	verbDrop   = "drop"
	verbSelect = "select"
	verbInsert = "insert"
	verbUpdate = "update"
	verbDelete = "delete"
	verbMerge  = "merge"
)

var tableFollowsKeywords = map[string]bool{
	"from":   true,
	"join":   true,
	"update": true,
	"into":   true, // INSERT INTO, MERGE INTO
	"table":  true, // CREATE TABLE, ALTER TABLE
	"using":  true, // MERGE ... USING
	"insert": true, // INSERT my_table
	"merge":  true, // MERGE my_table
}

var tableContextExitKeywords = map[string]bool{
	"where":  true,
	"group":  true, // GROUP BY
	"having": true,
	"order":  true, // ORDER BY
	"limit":  true,
	"window": true,
	"on":     true, // JOIN ... ON
	"set":    true, // UPDATE ... SET
	"when":   true, // MERGE ... WHEN
}

// TableParser is the main entry point for parsing a SQL string to find all referenced table IDs.
// It handles multi-statement SQL, comments, and recursive parsing of EXECUTE IMMEDIATE statements.
func TableParser(sql, defaultProjectID string) ([]string, error) {
	tableIDSet := make(map[string]struct{})
	visitedSQLs := make(map[string]struct{})
	if _, err := parseSQL(sql, defaultProjectID, tableIDSet, visitedSQLs, false); err != nil {
		return nil, err
	}

	tableIDs := make([]string, 0, len(tableIDSet))
	for id := range tableIDSet {
		tableIDs = append(tableIDs, id)
	}
	return tableIDs, nil
}

// parseSQL is the core recursive function that processes SQL strings.
// It uses a state machine to find table names and recursively parse EXECUTE IMMEDIATE.
func parseSQL(sql, defaultProjectID string, tableIDSet map[string]struct{}, visitedSQLs map[string]struct{}, inSubquery bool) (int, error) {
	// Prevent infinite recursion.
	if _, ok := visitedSQLs[sql]; ok {
		return len(sql), nil
	}
	visitedSQLs[sql] = struct{}{}

	state := stateNormal
	expectingTable := false
	var lastTableKeyword, lastToken, statementVerb string
	runes := []rune(sql)

	for i := 0; i < len(runes); {
		char := runes[i]
		remaining := sql[i:]

		switch state {
		case stateNormal:
			if strings.HasPrefix(remaining, "--") {
				state = stateInSingleLineCommentDash
				i += 2
				continue
			}
			if strings.HasPrefix(remaining, "#") {
				state = stateInSingleLineCommentHash
				i++
				continue
			}
			if strings.HasPrefix(remaining, "/*") {
				state = stateInMultiLineComment
				i += 2
				continue
			}
			if char == '(' {
				if expectingTable {
					// The subquery starts after '('.
					consumed, err := parseSQL(remaining[1:], defaultProjectID, tableIDSet, visitedSQLs, true)
					if err != nil {
						return 0, err
					}
					// Advance i by the length of the subquery + the opening parenthesis.
					// The recursive call returns what it consumed, including the closing parenthesis.
					i += consumed + 1
					// For most keywords, we expect only one table. `from` can have multiple "tables" (subqueries).
					if lastTableKeyword != "from" {
						expectingTable = false
					}
					continue
				}
			}
			if char == ')' {
				if inSubquery {
					return i + 1, nil
				}
			}

			if char == ';' {
				statementVerb = ""
				lastToken = ""
				i++
				continue

			}

			// Raw strings must be checked before regular strings.
			if strings.HasPrefix(remaining, "r'''") || strings.HasPrefix(remaining, "R'''") {
				state = stateInRawTripleSingleQuoteString
				i += 4
				continue
			}
			if strings.HasPrefix(remaining, `r"""`) || strings.HasPrefix(remaining, `R"""`) {
				state = stateInRawTripleDoubleQuoteString
				i += 4
				continue
			}
			if strings.HasPrefix(remaining, "r'") || strings.HasPrefix(remaining, "R'") {
				state = stateInRawSingleQuoteString
				i += 2
				continue
			}
			if strings.HasPrefix(remaining, `r"`) || strings.HasPrefix(remaining, `R"`) {
				state = stateInRawDoubleQuoteString
				i += 2
				continue
			}
			if strings.HasPrefix(remaining, "'''") {
				state = stateInTripleSingleQuoteString
				i += 3
				continue
			}
			if strings.HasPrefix(remaining, `"""`) {
				state = stateInTripleDoubleQuoteString
				i += 3
				continue
			}
			if char == '\'' {
				state = stateInSingleQuoteString
				i++
				continue
			}
			if char == '"' {
				state = stateInDoubleQuoteString
				i++
				continue
			}

			if unicode.IsLetter(char) || char == '`' {
				parts, consumed, err := parseIdentifierSequence(remaining)
				if err != nil {
					return 0, err
				}
				if consumed == 0 {
					i++
					continue
				}

				if len(parts) == 1 {
					keyword := strings.ToLower(parts[0])
					if keyword == "call" {
						return 0, fmt.Errorf("CALL is not allowed when dataset restrictions are in place, as the called procedure's contents cannot be safely analyzed")
					}
					if lastToken == "execute" && keyword == "immediate" {
						return 0, fmt.Errorf("EXECUTE IMMEDIATE is not allowed when dataset restrictions are in place, as its contents cannot be safely analyzed")
					}

					// The statementVerb is set by the first DML/DDL keyword and is used to identify context.
					if keyword == "procedure" || keyword == "function" {
						if lastToken == "create" || lastToken == "create or replace" {
							return 0, fmt.Errorf("unanalyzable statements like '%s %s' are not allowed", strings.ToUpper(lastToken), strings.ToUpper(keyword))
						}
					}

					switch keyword {
					case verbCreate, verbAlter, verbDrop, verbSelect, verbInsert, verbUpdate, verbDelete, verbMerge:
						if statementVerb == "" {
							statementVerb = keyword
						}
					}

					if statementVerb == verbCreate || statementVerb == verbAlter || statementVerb == verbDrop {
						if keyword == "schema" || keyword == "dataset" {
							return 0, fmt.Errorf("dataset-level operations like '%s %s' are not allowed when dataset restrictions are in place", strings.ToUpper(statementVerb), strings.ToUpper(keyword))
						}
					}

					if _, ok := tableFollowsKeywords[keyword]; ok {
						expectingTable = true
						lastTableKeyword = keyword
					} else if _, ok := tableContextExitKeywords[keyword]; ok {
						expectingTable = false
						lastTableKeyword = ""
					}
					if lastToken == "create" && keyword == "or" {
						lastToken = "create or"
					} else if lastToken == "create or" && keyword == "replace" {
						lastToken = "create or replace"
					} else {
						lastToken = keyword
					}
				} else if len(parts) >= 2 {
					// This is a multi-part identifier. If we were expecting a table, this is it.
					if expectingTable {
						tableID, err := formatTableID(parts, defaultProjectID)
						if err != nil {
							return 0, err
						}
						if tableID != "" {
							tableIDSet[tableID] = struct{}{}
						}
						// For most keywords, we expect only one table.
						if lastTableKeyword != "from" {
							expectingTable = false
						}
					}
					lastToken = ""
				}

				i += consumed
				continue
			}
			i++

		case stateInSingleQuoteString:
			if char == '\\' {
				i += 2 // Skip backslash and the escaped character.
				continue
			}
			if char == '\'' {
				state = stateNormal
			}
			i++
		case stateInDoubleQuoteString:
			if char == '\\' {
				i += 2 // Skip backslash and the escaped character.
				continue
			}
			if char == '"' {
				state = stateNormal
			}
			i++
		case stateInTripleSingleQuoteString:
			if strings.HasPrefix(remaining, "'''") {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		case stateInTripleDoubleQuoteString:
			if strings.HasPrefix(remaining, `"""`) {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		case stateInSingleLineCommentDash, stateInSingleLineCommentHash:
			if char == '\n' {
				state = stateNormal
			}
			i++
		case stateInMultiLineComment:
			if strings.HasPrefix(remaining, "*/") {
				state = stateNormal
				i += 2
			} else {
				i++
			}
		case stateInRawSingleQuoteString:
			if char == '\'' {
				state = stateNormal
			}
			i++
		case stateInRawDoubleQuoteString:
			if char == '"' {
				state = stateNormal
			}
			i++
		case stateInRawTripleSingleQuoteString:
			if strings.HasPrefix(remaining, "'''") {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		case stateInRawTripleDoubleQuoteString:
			if strings.HasPrefix(remaining, `"""`) {
				state = stateNormal
				i += 3
			} else {
				i++
			}
		}
	}

	if inSubquery {
		return 0, fmt.Errorf("unclosed subquery parenthesis")
	}
	return len(sql), nil
}

// parseIdentifierSequence parses a sequence of dot-separated identifiers.
// It returns the parts of the identifier, the number of characters consumed, and an error.
func parseIdentifierSequence(s string) ([]string, int, error) {
	var parts []string
	var totalConsumed int

	for {
		remaining := s[totalConsumed:]
		trimmed := strings.TrimLeftFunc(remaining, unicode.IsSpace)
		totalConsumed += len(remaining) - len(trimmed)
		current := s[totalConsumed:]

		if len(current) == 0 {
			break
		}

		var part string
		var consumed int

		if current[0] == '`' {
			end := strings.Index(current[1:], "`")
			if end == -1 {
				return nil, 0, fmt.Errorf("unclosed backtick identifier")
			}
			part = current[1 : end+1]
			consumed = end + 2
		} else if len(current) > 0 && unicode.IsLetter(rune(current[0])) {
			end := strings.IndexFunc(current, func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '_' && r != '-'
			})
			if end == -1 {
				part = current
				consumed = len(current)
			} else {
				part = current[:end]
				consumed = end
			}
		} else {
			break
		}

		if current[0] == '`' && strings.Contains(part, ".") {
			// This handles cases like `project.dataset.table` but not `project.dataset`.table.
			// If the character after the quoted identifier is not a dot, we treat it as a full name.
			if len(current) <= consumed || current[consumed] != '.' {
				parts = append(parts, strings.Split(part, ".")...)
				totalConsumed += consumed
				break
			}
		}

		parts = append(parts, strings.Split(part, ".")...)
		totalConsumed += consumed

		if len(s) <= totalConsumed || s[totalConsumed] != '.' {
			break
		}
		totalConsumed++
	}
	return parts, totalConsumed, nil
}

func formatTableID(parts []string, defaultProjectID string) (string, error) {
	if len(parts) < 2 || len(parts) > 3 {
		// Not a table identifier (could be a CTE, column, etc.).
		// Return the consumed length so the main loop can skip this identifier.
		return "", nil
	}

	var tableID string
	if len(parts) == 3 { // project.dataset.table
		tableID = strings.Join(parts, ".")
	} else { // dataset.table
		if defaultProjectID == "" {
			return "", fmt.Errorf("query contains table '%s' without project ID, and no default project ID is provided", strings.Join(parts, "."))
		}
		tableID = fmt.Sprintf("%s.%s", defaultProjectID, strings.Join(parts, "."))
	}

	return tableID, nil
}
