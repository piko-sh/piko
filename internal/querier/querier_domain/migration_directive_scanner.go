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

// scanMigrationReadOnlyOverrides scans migration content for piko.migration(readonly:
// ...) directives preceding CREATE FUNCTION statements.
//
// Takes content (string) which is the migration SQL content.
// Takes commentPrefix (string) which is the SQL comment prefix (e.g. "--").
//
// Returns map[string]*bool which maps lowercase function names to their read-only
// override values.
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
				continue
			}

			pendingReadOnly = nil
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

// parseReadOnlyDirectiveValue parses the readonly keyword from a
// `piko.migration(readonly: true|false, ...)` directive comment body.
//
// Other keyword arguments (e.g. no_transaction) are ignored here.
//
// Takes comment (string) which is the comment body text.
//
// Returns readOnly (bool) which is the parsed boolean value.
// Returns matched (bool) which is true if a readonly keyword was found.
func parseReadOnlyDirectiveValue(comment string) (readOnly bool, matched bool) {
	return migrationDirectiveBool(comment, "readonly")
}

// migrationDirectiveBool returns the boolean value of the given keyword in a
// `piko.migration(...)` comment body.
//
// matched is false when the comment is not a migration directive, the keyword is absent,
// or its value is not a boolean literal.
//
// Takes comment (string) which is the comment body text.
// Takes keyword (string) which is the keyword whose boolean value to read.
//
// Returns value (bool) which is the parsed boolean value.
// Returns matched (bool) which is true when the keyword was found with a boolean value.
func migrationDirectiveBool(comment, keyword string) (value bool, matched bool) {
	inner, ok := migrationDirectiveInner(comment)
	if !ok {
		return false, false
	}
	for _, segment := range splitTopLevelKeywordArgumentSegments(inner) {
		colonIndex := strings.Index(segment, ":")
		if colonIndex <= 0 {
			continue
		}
		if strings.TrimSpace(segment[:colonIndex]) != keyword {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(segment[colonIndex+1:])) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

// scanMisplacedQueryDirectives returns the 1-based line numbers of any `piko.query(...)`
// directive in migration content. A query header has no meaning in a migration file, so
// the caller warns and ignores it.
//
// Takes content (string) which is the migration SQL content.
// Takes commentPrefix (string) which is the SQL comment prefix (e.g. "--").
//
// Returns []int which holds the 1-based line numbers of misplaced piko.query directives.
func scanMisplacedQueryDirectives(content, commentPrefix string) []int {
	var lineNumbers []int
	for index, line := range strings.Split(content, "\n") {
		body, hasPrefix := strings.CutPrefix(strings.TrimSpace(line), commentPrefix)
		if !hasPrefix {
			continue
		}
		if _, isQuery := cutDirectiveCallPrefix(strings.TrimSpace(body), "piko.query"); isQuery {
			lineNumbers = append(lineNumbers, index+1)
		}
	}
	return lineNumbers
}

// migrationDirectiveInner returns the parenthesised argument text of a
// `piko.migration(...)` comment body.
//
// The scanners here read a single physical line at a time, so a directive split across
// continuation lines is not recognised; the full lexer in directive_parser.go handles
// that case for query files.
//
// Takes comment (string) which is the comment body text.
//
// Returns string which is the parenthesised argument text.
// Returns bool which is true when the comment was a piko.migration directive.
func migrationDirectiveInner(comment string) (string, bool) {
	afterCall, ok := cutDirectiveCallPrefix(comment, "piko.migration")
	if !ok {
		return "", false
	}
	inner, ok := strings.CutSuffix(afterCall, ")")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(inner), true
}

// cutDirectiveCallPrefix strips a `<callName>(` opener from the front of a single-line
// directive body, tolerating whitespace before the opening parenthesis.
//
// Whitespace between the call name and the parenthesis is tolerated, so a body such as
// "piko.column (table.col)" still matches.
//
// Takes body (string) which is a trimmed single-line directive body.
// Takes callName (string) which is the dotted directive name without its parenthesis.
//
// Returns string which is the text following the opening parenthesis.
// Returns bool which is true when the body opened the named call.
func cutDirectiveCallPrefix(body, callName string) (string, bool) {
	afterName, ok := strings.CutPrefix(body, callName)
	if !ok {
		return "", false
	}
	afterParen, ok := strings.CutPrefix(strings.TrimLeft(afterName, " \t"), "(")
	if !ok {
		return "", false
	}
	return afterParen, true
}

// splitTopLevelKeywordArgumentSegments splits a keyword argument list at top-level
// commas.
//
// Commas inside bracketed list literals, parenthesised expression values, and
// brace-delimited typed parameter forms are preserved. Commas inside single- or
// double-quoted string values are also preserved: a quoted span is copied verbatim into
// the current segment, honouring backslash and SQL-standard doubled-quote escapes so a
// comma inside the quotes never truncates the value. Trailing empty segments produced by
// a trailing comma are discarded.
//
// Takes input (string) which is the comma-separated keyword argument list.
//
// Returns []string which holds the top-level segments.
func splitTopLevelKeywordArgumentSegments(input string) []string {
	var segments []string
	var current strings.Builder
	bracketDepth := 0
	parenDepth := 0
	braceDepth := 0
	index := 0
	for index < len(input) {
		character := input[index]
		switch character {
		case '\'', '"':
			nextIndex, _ := skipStringLiteral(input, index, character)
			current.WriteString(input[index:nextIndex])
			index = nextIndex
			continue
		case '[':
			bracketDepth++
			current.WriteByte(character)
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
			current.WriteByte(character)
		case '(':
			parenDepth++
			current.WriteByte(character)
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			current.WriteByte(character)
		case '{':
			braceDepth++
			current.WriteByte(character)
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			current.WriteByte(character)
		case ',':
			segments = appendSplitSegment(segments, &current, bracketDepth+parenDepth+braceDepth, character)
		default:
			current.WriteByte(character)
		}
		index++
	}
	if remainder := strings.TrimSpace(current.String()); remainder != "" {
		segments = append(segments, remainder)
	}
	return segments
}

// appendSplitSegment commits the current accumulated segment when the comma falls outside
// any nested grouping.
//
// When totalDepth is non-zero the comma is inside a list literal, parenthesised value, or
// brace form, and is treated as a literal byte written into current.
//
// Takes segments ([]string) which holds the segments committed so far.
// Takes current (*strings.Builder) which accumulates the segment being built.
// Takes totalDepth (int) which is the combined bracket, paren, and brace nesting depth.
// Takes character (byte) which is the comma byte under consideration.
//
// Returns []string which is the updated segment slice.
func appendSplitSegment(segments []string, current *strings.Builder, totalDepth int, character byte) []string {
	if totalDepth != 0 {
		current.WriteByte(character)
		return segments
	}
	segment := strings.TrimSpace(current.String())
	if segment != "" {
		segments = append(segments, segment)
	}
	current.Reset()
	return segments
}

// extractCreateFunctionName extracts the function name from a CREATE FUNCTION or CREATE
// PROCEDURE statement line.
//
// Takes line (string) which is the SQL statement line.
//
// Returns string which is the lowercase function name, or empty if the line is not a
// CREATE FUNCTION statement.
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

// findFunctionKeywordIndex returns the index of FUNCTION or PROCEDURE when it is the
// object keyword of a CREATE statement.
//
// The keyword must be the first word after the CREATE prefix modifiers (OR, REPLACE,
// TEMP, TEMPORARY), so a statement whose object is TRIGGER, TABLE, VIEW, etc. is rejected
// even when it later mentions FUNCTION (e.g. CREATE TRIGGER ... EXECUTE FUNCTION foo()).
//
// Takes upperWords ([]string) which holds the uppercased tokens of the statement.
//
// Returns int which is the index of the keyword, or -1 if not found.
func findFunctionKeywordIndex(upperWords []string) int {
	if len(upperWords) < 2 || upperWords[0] != "CREATE" {
		return -1
	}

	index := 1
	for index < len(upperWords) && isCreatePrefixModifier(upperWords[index]) {
		index++
	}
	if index >= len(upperWords) {
		return -1
	}
	if upperWords[index] == "FUNCTION" || upperWords[index] == "PROCEDURE" {
		return index
	}
	return -1
}

// isCreatePrefixModifier reports whether an uppercased word is one of the modifiers that
// may appear between CREATE and the object keyword.
//
// Takes word (string) which is an uppercased statement token.
//
// Returns bool which is true when the word is a CREATE prefix modifier.
func isCreatePrefixModifier(word string) bool {
	switch word {
	case "OR", "REPLACE", "TEMP", "TEMPORARY":
		return true
	default:
		return false
	}
}

// cleanFunctionName strips parentheses, schema prefixes, and quotes from a raw function
// name token.
//
// Takes raw (string) which is the raw function name token.
//
// Returns string which is the cleaned, lowercase function name.
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

// migrationColumnOverride captures the parsed shape of a single migration-level `--
// piko.column(table.col, type: ..., go_type: ..., nullable: ...)` directive.
//
// The catalogue builder applies these to Column entries after the engine's DDL
// interpretation builds the table.
type migrationColumnOverride struct {
	// Nullable overrides the column's nullability when set.
	Nullable *bool

	// Table is the lowercase table name the override targets.
	Table string

	// Column is the lowercase column name the override targets.
	Column string

	// SQLType overrides the column's SQL type when set.
	SQLType string

	// GoType overrides the column's generated Go type when set.
	GoType string
}

// scanMigrationColumnOverrides scans a migration file's content for `--
// piko.column(table.col, ...)` directives and returns one entry per directive found. The
// directives may sit above a CREATE TABLE or anywhere else; the catalogue builder later
// resolves them against the columns produced by the engine's DDL pass.
//
// Takes content (string) which is the migration file content.
// Takes commentPrefix (string) which is the engine's line-comment prefix.
//
// Returns []migrationColumnOverride which holds one entry per parsed directive. Returns
// nil when no directives are found.
func scanMigrationColumnOverrides(content string, commentPrefix string) []migrationColumnOverride {
	var overrides []migrationColumnOverride
	for line := range strings.SplitSeq(content, "\n") {
		override, ok := parseMigrationColumnOverrideLine(line, commentPrefix)
		if !ok {
			continue
		}
		overrides = append(overrides, override)
	}
	return overrides
}

// parseMigrationColumnOverrideLine attempts to parse a single line as a `--
// piko.column(table.col, ...)` directive.
//
// Centralising the per-line parse keeps scanMigrationColumnOverrides within the
// cognitive-complexity budget.
//
// Takes line (string) which is the migration line to inspect.
// Takes commentPrefix (string) which is the engine's line-comment prefix.
//
// Returns migrationColumnOverride which is the parsed override.
// Returns bool which is true when the line matched the directive, in which case the
// caller keeps the override and otherwise skips the line.
func parseMigrationColumnOverrideLine(line, commentPrefix string) (migrationColumnOverride, bool) {
	trimmed := strings.TrimSpace(line)
	afterPrefix, hasPrefix := strings.CutPrefix(trimmed, commentPrefix)
	if !hasPrefix {
		return migrationColumnOverride{}, false
	}
	body := strings.TrimSpace(afterPrefix)
	afterCall, hasCallPrefix := cutDirectiveCallPrefix(body, "piko.column")
	if !hasCallPrefix {
		return migrationColumnOverride{}, false
	}
	inner, hasCallSuffix := strings.CutSuffix(afterCall, ")")
	if !hasCallSuffix {
		return migrationColumnOverride{}, false
	}
	segments := splitTopLevelKeywordArgumentSegments(strings.TrimSpace(inner))
	if len(segments) == 0 {
		return migrationColumnOverride{}, false
	}

	positional := strings.TrimSpace(segments[0])
	dotIndex := strings.LastIndex(positional, ".")
	if dotIndex <= 0 || dotIndex >= len(positional)-1 {
		return migrationColumnOverride{}, false
	}

	override := migrationColumnOverride{
		Table:  strings.ToLower(positional[:dotIndex]),
		Column: strings.ToLower(positional[dotIndex+1:]),
	}
	for _, segment := range segments[1:] {
		applyMigrationColumnOverrideKeyword(segment, &override)
	}
	return override, true
}

// applyMigrationColumnOverrideKeyword reads one `key: value` segment from a column
// override directive and writes it onto override.
//
// Keys that the migration scanner does not recognise are ignored so unrelated keyword
// arguments do not cause hard failures here.
//
// Takes segment (string) which is one `key: value` directive segment.
// Takes override (*migrationColumnOverride) which receives the parsed value.
func applyMigrationColumnOverrideKeyword(segment string, override *migrationColumnOverride) {
	colonIndex := strings.Index(segment, ":")
	if colonIndex <= 0 {
		return
	}
	key := strings.TrimSpace(segment[:colonIndex])
	value := strings.TrimSpace(segment[colonIndex+1:])
	value = strings.Trim(value, "\"'")
	switch key {
	case "type":
		override.SQLType = value
	case "go_type":
		override.GoType = value
	case "nullable":
		switch strings.ToLower(value) {
		case "true":
			override.Nullable = new(true)
		case "false":
			override.Nullable = new(false)
		}
	}
}

// applyMigrationReadOnlyOverride sets the data access level on a CREATE FUNCTION mutation
// if a matching read-only override exists.
//
// Takes mutation (*querier_dto.CatalogueMutation) which is the mutation to update.
// Takes overrides (map[string]*bool) which maps function names to their read-only
// override values.
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
