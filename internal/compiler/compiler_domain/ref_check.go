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

package compiler_domain

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_dto"
	"piko.sh/piko/internal/jsscan"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
	"piko.sh/piko/internal/sfcparser"
)

const (
	// componentContextName is the name of the component context in component scripts.
	componentContextName = "pkc"

	// componentRefsProperty is the property of the component context that holds its refs.
	componentRefsProperty = "refs"

	// refAttributeName is the name of the p-ref directive when it is bound dynamically.
	refAttributeName = "p-ref"

	// maxComponentScriptTokens caps how many tokens of a component script are checked.
	maxComponentScriptTokens = 1 << 20

	// maxRefDiagnostics caps how many unresolved ref warnings one component reports.
	maxRefDiagnostics = 50
)

// findUnresolvedComponentRefs reports pkc.refs names in a component script that no p-ref
// in the component's own template declares. A component's refs hold only the elements its
// template renders, so content slotted in from outside never counts.
//
// Names the script assigns, as in pkc.refs.name = element, count as declared. Computed
// names are not checked, and nothing is checked when the template binds p-ref.
//
// Takes script (string) which is the component script.
// Takes scriptStart (sfcparser.Location) which is where the script content begins in the
// source file.
// Takes tree (*ast_domain.TemplateAST) which is the component's template.
// Takes sourceIdentifier (string) which identifies the component in diagnostics.
//
// Returns []compiler_dto.CompilationDiagnostic which holds a warning per unresolved ref.
func findUnresolvedComponentRefs(
	script string,
	scriptStart sfcparser.Location,
	tree *ast_domain.TemplateAST,
	sourceIdentifier string,
) []compiler_dto.CompilationDiagnostic {
	if script == "" || tree == nil {
		return nil
	}
	tokens, _ := jsscan.Tokens(script, maxComponentScriptTokens)
	reads := jsscan.FindPropertyReads(tokens, componentContextName, componentRefsProperty)
	if len(reads) == 0 {
		return nil
	}
	declared, isDynamic := templateRefNames(tree)
	if isDynamic {
		return nil
	}
	for _, read := range reads {
		if read.Assigned {
			declared[read.Value] = struct{}{}
		}
	}
	candidates := slices.Sorted(maps.Keys(declared))
	lines := jsscan.NewLineIndex(script)
	var diagnostics []compiler_dto.CompilationDiagnostic
	for _, read := range reads {
		if _, found := declared[read.Value]; found {
			continue
		}
		diagnostics = append(diagnostics, newUnresolvedRefDiagnostic(read, lines, scriptStart, candidates, sourceIdentifier))
		if len(diagnostics) == maxRefDiagnostics {
			break
		}
	}
	return diagnostics
}

// newUnresolvedRefDiagnostic builds the warning for one pkc.refs name that no p-ref
// declares, placed in the component's source file.
//
// Takes read (jsscan.PropertyRead) which is the unresolved ref.
// Takes lines (*jsscan.LineIndex) which places offsets within the script.
// Takes scriptStart (sfcparser.Location) which is where the script content begins.
// Takes candidates ([]string) which holds the declared names, for a suggestion.
// Takes sourceIdentifier (string) which identifies the component.
//
// Returns compiler_dto.CompilationDiagnostic which is the warning.
func newUnresolvedRefDiagnostic(
	read jsscan.PropertyRead,
	lines *jsscan.LineIndex,
	scriptStart sfcparser.Location,
	candidates []string,
	sourceIdentifier string,
) compiler_dto.CompilationDiagnostic {
	line, column := lines.Position(read.Start)
	if line == 1 {
		column += max(scriptStart.Column, 1) - 1
	}
	message := fmt.Sprintf("pkc.refs.%s matches no element with p-ref=%q in the component's template, so it is always undefined",
		read.Value, read.Value)
	if suggestion := linguistics_domain.ClosestName(read.Value, candidates); suggestion != "" {
		message += fmt.Sprintf(". Did you mean '%s'?", suggestion)
	}
	return compiler_dto.CompilationDiagnostic{
		Severity:         compiler_dto.DiagnosticSeverityWarning,
		Message:          message,
		SourceIdentifier: sourceIdentifier,
		Code:             annotator_dto.CodeUnresolvedRef,
		Line:             max(scriptStart.Line, 1) + line - 1,
		Column:           column,
	}
}

// templateRefNames collects the p-ref names a template declares.
//
// Takes tree (*ast_domain.TemplateAST) which is the template.
//
// Returns map[string]struct{} which holds the names.
// Returns bool which is true when the template binds p-ref, so any name is possible.
func templateRefNames(tree *ast_domain.TemplateAST) (map[string]struct{}, bool) {
	names := make(map[string]struct{})
	isDynamic := false
	tree.Walk(func(node *ast_domain.TemplateNode) bool {
		if node.DirRef != nil && node.DirRef.RawExpression != "" {
			names[node.DirRef.RawExpression] = struct{}{}
		}
		if _, bound := node.Binds[refAttributeName]; bound {
			isDynamic = true
		}
		for index := range node.DynamicAttributes {
			if strings.EqualFold(node.DynamicAttributes[index].Name, refAttributeName) {
				isDynamic = true
			}
		}
		return true
	})
	return names, isDynamic
}
