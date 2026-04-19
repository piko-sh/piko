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

package annotator_domain

// Validates PK event handlers by checking that referenced functions exist in
// the client script exports. Provides compile-time safety for p-on event
// bindings by verifying function names and async compatibility.

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// maxExportsToDisplay is the maximum number of export names to show in error
// messages before shortening the list to "and N more".
const maxExportsToDisplay = 5

// PKValidator validates PK client-side JavaScript integration.
// It verifies that p-on event handlers reference valid exported functions
// from the client script block.
type PKValidator struct {
	// clientExports stores the parsed export data from the client script.
	clientExports *ClientScriptExports

	// usedHandlers tracks which exported functions are used as event handlers.
	usedHandlers map[string]bool

	// usedPartials tracks partials used in reloadPartial or reloadGroup calls.
	usedPartials map[string]bool

	// renderedPartials tracks which partials have been rendered in the template.
	renderedPartials map[string]bool

	// importedPartials tracks partials that were imported via PikoImports.
	importedPartials map[string]bool

	// sfcSourcePath is the file path to the PK file, used in error messages.
	sfcSourcePath string

	// clientScript holds the raw client script content for analysis.
	clientScript string
}

// NewPKValidator creates a validator for a PK component.
//
// Takes clientScript (string) which is the raw JavaScript or TypeScript code.
// Takes sfcPath (string) which is the path to the PK file.
//
// Returns *PKValidator which may have nil clientExports if the script is
// empty or cannot be parsed.
func NewPKValidator(clientScript string, sfcPath string) *PKValidator {
	var exports *ClientScriptExports
	if clientScript != "" {
		exports = AnalyseClientScript(clientScript, sfcPath)
	}

	validator := &PKValidator{
		clientExports:    exports,
		usedHandlers:     make(map[string]bool),
		usedPartials:     make(map[string]bool),
		renderedPartials: make(map[string]bool),
		importedPartials: make(map[string]bool),
		sfcSourcePath:    sfcPath,
		clientScript:     clientScript,
	}

	if clientScript != "" {
		validator.analysePartialUsage(clientScript)
	}

	return validator
}

// HasClientScript reports whether the component has a parseable client script.
//
// Returns bool which is true when the validator has exported client functions.
func (v *PKValidator) HasClientScript() bool {
	return v != nil && v.clientExports != nil && len(v.clientExports.ExportedFunctions) > 0
}

// ValidateEventHandler checks if a p-on directive references a valid exported
// function from the client script.
//
// For directives without a modifier, the expression should be a call to an
// exported function. Extracts the function name and validates it exists.
//
// Takes directive (*ast_domain.Directive) which is the event directive to
// validate.
// Takes ctx (*AnalysisContext) which receives any diagnostic messages.
func (v *PKValidator) ValidateEventHandler(directive *ast_domain.Directive, ctx *AnalysisContext) {
	if v == nil || directive == nil {
		return
	}

	handlerName := extractHandlerName(directive)
	if handlerName == "" {
		return
	}

	v.usedHandlers[handlerName] = true

	if v.clientExports == nil {
		return
	}

	if !v.clientExports.HasExport(handlerName) {
		if isLikelyGoFunction(handlerName) {
			return
		}

		message := fmt.Sprintf(
			"Event handler '%s' not found in client script exports. "+
				"Did you forget to export it? Available exports: %s",
			handlerName,
			formatAvailableExports(v.clientExports),
		)
		ctx.addDiagnostic(
			ast_domain.Error,
			message,
			directive.RawExpression,
			directive.Location,
			directive.GoAnnotations,
			annotator_dto.CodeClientScriptError,
		)
	}
}

// ReportUnusedExports does nothing. Exists because top-level functions are
// now auto-exported, so utility functions called by event handlers are valid
// and expected.
func (*PKValidator) ReportUnusedExports(_ *AnalysisContext, _ ast_domain.Location) {
}

// RegisterImportedPartials records the partial aliases imported in this
// component. The semantic analyser calls this when processing PikoImports.
//
// Takes aliases ([]string) which are the partial import aliases to register.
func (v *PKValidator) RegisterImportedPartials(aliases []string) {
	if v == nil {
		return
	}
	for _, alias := range aliases {
		v.importedPartials[alias] = true
	}
}

// MarkPartialRendered records that a partial has been rendered in the
// template. This prevents false warnings about unused imports when a partial
// is rendered but not used in reloadPartial() calls.
//
// Takes alias (string) which is the import alias of the rendered partial.
func (v *PKValidator) MarkPartialRendered(alias string) {
	if v == nil || alias == "" {
		return
	}
	v.renderedPartials[alias] = true
}

// ReportOrphanedPartials adds warnings for partials that are imported but never
// used. This includes partials that are neither rendered in the template nor
// referenced in reloadPartial() or reloadGroup() calls.
//
// Takes ctx (*AnalysisContext) which receives warning diagnostics.
// Takes scriptLocation (ast_domain.Location) which is the location for
// diagnostics.
func (v *PKValidator) ReportOrphanedPartials(ctx *AnalysisContext, scriptLocation ast_domain.Location) {
	if v == nil || len(v.importedPartials) == 0 {
		return
	}

	for alias := range v.importedPartials {
		isRendered := v.renderedPartials[alias]
		isUsedInReload := v.usedPartials[alias]

		if !isRendered && !isUsedInReload {
			message := fmt.Sprintf(
				"Partial '%s' is imported but never used (not rendered in template or referenced in reloadPartial/reloadGroup calls)",
				alias,
			)
			ctx.addDiagnostic(
				ast_domain.Warning,
				message,
				alias,
				scriptLocation,
				nil,
				annotator_dto.CodeClientScriptError,
			)
		}
	}
}

// analysePartialUsage scans the client script for reloadPartial and
// reloadGroup calls and records which partial aliases are used.
//
// Takes script (string) which contains the client script to analyse.
func (v *PKValidator) analysePartialUsage(script string) {
	if v == nil || script == "" {
		return
	}

	v.extractPartialCalls(script, "reloadPartial")

	v.extractReloadGroupCalls(script)
}

// extractPartialCalls finds calls like reloadPartial('alias') or
// reloadPartial("alias") in the given script text.
//
// Takes script (string) which is the source text to search.
// Takes functionName (string) which is the function name to match.
func (v *PKValidator) extractPartialCalls(script, functionName string) {
	patterns := []string{
		functionName + "('",
		functionName + "(\"",
	}

	for _, pattern := range patterns {
		quoteChar := pattern[len(pattern)-1]
		index := 0
		for {
			foundIndex := strings.Index(script[index:], pattern)
			if foundIndex == -1 {
				break
			}
			index += foundIndex + len(pattern)

			endIndex := strings.IndexByte(script[index:], quoteChar)
			if endIndex == -1 {
				break
			}
			alias := script[index : index+endIndex]
			if alias != "" && isValidPartialAlias(alias) {
				v.usedPartials[alias] = true
			}
			index += endIndex + 1
		}
	}
}

// extractReloadGroupCalls finds calls like reloadGroup(['alias1', 'alias2'])
// and extracts the aliases from them.
//
// Takes script (string) which contains the JavaScript code to search.
func (v *PKValidator) extractReloadGroupCalls(script string) {
	patterns := []string{"reloadGroup([", "reloadGroup( ["}

	for _, pattern := range patterns {
		v.extractReloadGroupForPattern(script, pattern)
	}
}

// extractReloadGroupForPattern finds all matches of a pattern in the script
// and extracts aliases from each match.
//
// Takes script (string) which is the script content to search.
// Takes pattern (string) which is the pattern to match against.
func (v *PKValidator) extractReloadGroupForPattern(script, pattern string) {
	index := 0
	for {
		foundIndex := strings.Index(script[index:], pattern)
		if foundIndex == -1 {
			break
		}
		index += foundIndex + len(pattern)

		endIndex := v.findClosingBracket(script, index)
		if endIndex > index {
			arrayContent := script[index : endIndex-1]
			v.extractAliasesFromArray(arrayContent)
		}

		index = endIndex
	}
}

// findClosingBracket finds the position after the matching closing bracket.
//
// Takes script (string) which contains the text to search.
// Takes startIndex (int) which is the position just after the opening bracket.
//
// Returns int which is the position after the closing bracket, or startIndex if
// no matching bracket is found.
func (*PKValidator) findClosingBracket(script string, startIndex int) int {
	bracketDepth := 1
	endIndex := startIndex

	for endIndex < len(script) && bracketDepth > 0 {
		switch script[endIndex] {
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		}
		endIndex++
	}

	if bracketDepth != 0 {
		return startIndex
	}
	return endIndex
}

// extractAliasesFromArray finds partial aliases in an array literal string.
// It handles both single-quoted ('alias1', 'alias2') and double-quoted
// ("alias1", "alias2") formats, and stores valid aliases in the usedPartials
// map.
//
// Takes content (string) which is the array literal to parse for aliases.
func (v *PKValidator) extractAliasesFromArray(content string) {
	for part := range strings.SplitSeq(content, ",") {
		part = strings.TrimSpace(part)

		if len(part) >= 2 && part[0] == '\'' && part[len(part)-1] == '\'' {
			alias := part[1 : len(part)-1]
			if alias != "" && isValidPartialAlias(alias) {
				v.usedPartials[alias] = true
			}
			continue
		}

		if len(part) >= 2 && part[0] == '"' && part[len(part)-1] == '"' {
			alias := part[1 : len(part)-1]
			if alias != "" && isValidPartialAlias(alias) {
				v.usedPartials[alias] = true
			}
		}
	}
}

// extractHandlerName gets the function name from an event handler directive.
// It handles both simple calls like "handleClick()" and calls with arguments
// like "handleClick(event, data)".
//
// Takes directive (*ast_domain.Directive) which contains the parsed event
// handler expression.
//
// Returns string which is the function name, or empty if the directive is nil
// or has no valid expression.
func extractHandlerName(directive *ast_domain.Directive) string {
	if directive == nil {
		return ""
	}

	if directive.Expression != nil {
		if callExpr, ok := directive.Expression.(*ast_domain.CallExpression); ok {
			if identifier, ok := callExpr.Callee.(*ast_domain.Identifier); ok {
				return identifier.Name
			}
		}
		if identifier, ok := directive.Expression.(*ast_domain.Identifier); ok {
			return identifier.Name
		}
	}

	rawExpr := strings.TrimSpace(directive.RawExpression)
	if rawExpr == "" {
		return ""
	}

	if before, _, found := strings.Cut(rawExpr, "("); found && before != "" {
		return strings.TrimSpace(before)
	}

	return rawExpr
}

// isLikelyGoFunction checks if a name looks like a Go exported function.
// Exported functions in Go start with an uppercase letter.
//
// Takes name (string) which is the function name to check.
//
// Returns bool which is true if the name starts with an uppercase letter.
func isLikelyGoFunction(name string) bool {
	if name == "" {
		return false
	}
	firstChar := name[0]
	return firstChar >= 'A' && firstChar <= 'Z'
}

// isCommonUtilityName checks if a name matches common utility function
// patterns. These are functions that might be exported but used elsewhere, not
// as event handlers.
//
// Takes name (string) which is the function name to check.
//
// Returns bool which is true if the name starts with a common utility prefix
// such as "init", "setup", "create", "get", or "validate".
func isCommonUtilityName(name string) bool {
	utilityPatterns := []string{
		"init", "setup", "configure", "register",
		"create", "build", "make",
		"get", "set", "update",
		"format", "parse", "validate",
		"util", "helper", "factory",
	}

	nameLower := strings.ToLower(name)
	for _, pattern := range utilityPatterns {
		if strings.HasPrefix(nameLower, pattern) {
			return true
		}
	}
	return false
}

// formatAvailableExports formats export names for use in error messages.
//
// Takes exports (*ClientScriptExports) which contains the functions to format.
//
// Returns string which lists up to five export names with a count of any
// remaining, or "(none)" if there are no exports.
func formatAvailableExports(exports *ClientScriptExports) string {
	if exports == nil || len(exports.ExportedFunctions) == 0 {
		return "(none)"
	}

	names := exports.ExportNames()
	if len(names) == 0 {
		return "(none)"
	}

	if len(names) <= maxExportsToDisplay {
		return strings.Join(names, ", ")
	}

	return strings.Join(names[:maxExportsToDisplay], ", ") + fmt.Sprintf(" and %d more", len(names)-maxExportsToDisplay)
}

// isValidPartialAlias checks whether a string is a valid partial alias.
//
// A valid alias must start with a letter or underscore. The rest may contain
// only letters, numbers, or underscores.
//
// Takes alias (string) which is the string to check.
//
// Returns bool which is true if the alias is valid, false otherwise.
func isValidPartialAlias(alias string) bool {
	if alias == "" {
		return false
	}
	first := alias[0]
	if (first < 'a' || first > 'z') && (first < 'A' || first > 'Z') && first != '_' {
		return false
	}
	for i := 1; i < len(alias); i++ {
		c := alias[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
