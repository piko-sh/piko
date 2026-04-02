// Copyright 2026 PolitePixels Limited
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

// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package querier_domain

import (
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// scanMigrationReadOnlyOverrides scans migration content for
// piko.readonly directives preceding CREATE FUNCTION
// statements.
//
// Takes content (string) which is the migration SQL content.
// Takes commentPrefix (string) which is the SQL comment
// prefix (e.g. "--").
//
// Returns map[string]*bool which maps lowercase function
// names to their read-only override values.
func scanMigrationReadOnlyOverrides(content string, commentPrefix string) map[string]*bool {
	result := make(map[string]*bool)
	lines := strings.Split(content, "\n")
	var pendingReadOnly *bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, commentPrefix) {
			commentBody := strings.TrimSpace(trimmed[len(commentPrefix):])
			if value, matched := parseReadOnlyDirectiveValue(commentBody); matched {
				pendingReadOnly = &value
			}
			continue
		}

		if trimmed == "" {
			continue
		}

		if pendingReadOnly != nil {
			if name := extractCreateFunctionName(trimmed); name != "" {
				result[name] = pendingReadOnly
			}
			pendingReadOnly = nil
		}
	}

	return result
}

// parseReadOnlyDirectiveValue parses a piko.readonly
// directive from a comment body.
//
// Takes comment (string) which is the comment body text.
//
// Returns readOnly (bool) which is the parsed boolean value.
// Returns matched (bool) which is true if a valid directive
// was found.
func parseReadOnlyDirectiveValue(comment string) (readOnly bool, matched bool) {
	if !strings.HasPrefix(comment, "piko.readonly") {
		return false, false
	}
	rest := strings.TrimSpace(comment[len("piko.readonly"):])

	if rest == "" {
		return true, true
	}
	if rest == "(true)" {
		return true, true
	}
	if rest == "(false)" {
		return false, true
	}
	return false, false
}

// extractCreateFunctionName extracts the function name from
// a CREATE FUNCTION or CREATE PROCEDURE statement line.
//
// Takes line (string) which is the SQL statement line.
//
// Returns string which is the lowercase function name, or
// empty if the line is not a CREATE FUNCTION statement.
func extractCreateFunctionName(line string) string {
	upper := strings.ToUpper(line)
	words := strings.Fields(upper)
	functionIndex := findFunctionKeywordIndex(words)
	if functionIndex == -1 {
		return ""
	}

	originalWords := strings.Fields(line)
	if functionIndex+1 >= len(originalWords) {
		return ""
	}

	return cleanFunctionName(originalWords[functionIndex+1])
}

// findFunctionKeywordIndex returns the index of FUNCTION
// or PROCEDURE in a CREATE statement's uppercased words.
//
// Takes upperWords ([]string) which holds the uppercased
// tokens of the statement.
//
// Returns int which is the index of the keyword, or -1 if
// not found.
func findFunctionKeywordIndex(upperWords []string) int {
	if len(upperWords) < 2 || upperWords[0] != "CREATE" {
		return -1
	}

	for index, word := range upperWords[1:] {
		if word == "FUNCTION" || word == "PROCEDURE" {
			return index + 1
		}
	}
	return -1
}

// cleanFunctionName strips parentheses, schema prefixes,
// and quotes from a raw function name token.
//
// Takes raw (string) which is the raw function name token.
//
// Returns string which is the cleaned, lowercase function
// name.
func cleanFunctionName(raw string) string {
	name := raw
	if index := strings.IndexByte(name, '('); index >= 0 {
		name = name[:index]
	}
	if index := strings.LastIndexByte(name, '.'); index >= 0 {
		name = name[index+1:]
	}
	name = strings.Trim(name, "\"'`")
	return strings.ToLower(name)
}

// applyMigrationReadOnlyOverride sets the data access level
// on a CREATE FUNCTION mutation if a matching read-only
// override exists.
//
// Takes mutation (*querier_dto.CatalogueMutation) which is
// the mutation to update.
// Takes overrides (map[string]*bool) which maps function
// names to their read-only override values.
func applyMigrationReadOnlyOverride(
	mutation *querier_dto.CatalogueMutation,
	overrides map[string]*bool,
) {
	if mutation.Kind != querier_dto.MutationCreateFunction {
		return
	}
	if mutation.FunctionSignature == nil {
		return
	}
	name := strings.ToLower(mutation.FunctionSignature.Name)
	override, exists := overrides[name]
	if !exists {
		return
	}
	if *override {
		mutation.FunctionSignature.DataAccess = querier_dto.DataAccessReadOnly
	} else {
		mutation.FunctionSignature.DataAccess = querier_dto.DataAccessModifiesData
	}
}
