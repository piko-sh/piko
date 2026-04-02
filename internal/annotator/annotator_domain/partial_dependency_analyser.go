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

// Analyses dependencies between partial components to determine optimal expansion order and detect circular references.
// Builds a dependency graph and performs topological sorting to ensure partials are expanded in the correct sequence.

import (
	"fmt"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
)

// PartialDependencyAnalyser detects circular dependencies in partial reload
// chains. It analyses client scripts across components to build a dependency
// graph and detect cycles that would cause infinite reload loops.
type PartialDependencyAnalyser struct {
	// dependencies maps each partial alias to the set of partials it reloads.
	dependencies map[string]map[string]bool

	// aliasToComponent maps a partial alias to its component info for diagnostics.
	aliasToComponent map[string]*componentInfo
}

// componentInfo holds details about a component for use in error messages.
type componentInfo struct {
	// sourcePath is the path to the file where the component is defined.
	sourcePath string

	// clientScript is the JavaScript code for client-side rendering.
	clientScript string

	// location is the position of this component in the source code.
	location ast_domain.Location
}

// NewPartialDependencyAnalyser creates an analyser for finding circular
// dependencies in partial reloads.
//
// Returns *PartialDependencyAnalyser which is ready to track and check
// component dependencies.
func NewPartialDependencyAnalyser() *PartialDependencyAnalyser {
	return &PartialDependencyAnalyser{
		dependencies:     make(map[string]map[string]bool),
		aliasToComponent: make(map[string]*componentInfo),
	}
}

// AnalyseVirtualModule builds the dependency graph from all components in the
// virtual module and returns any circular dependency diagnostics.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which contains all parsed
// components.
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the entry point
// component for diagnostic positioning.
//
// Returns a slice of diagnostics for any detected circular dependencies.
func (pda *PartialDependencyAnalyser) AnalyseVirtualModule(
	virtualModule *annotator_dto.VirtualModule,
	mainComponent *annotator_dto.VirtualComponent,
) []*ast_domain.Diagnostic {
	if virtualModule == nil {
		return nil
	}

	pda.buildDependencyGraph(virtualModule)

	cycles := pda.detectCycles()

	if len(cycles) == 0 {
		return nil
	}

	return pda.cyclesToDiagnostics(cycles, mainComponent)
}

// buildDependencyGraph finds partial reload dependencies from all client
// scripts.
//
// Takes virtualModule (*annotator_dto.VirtualModule) which holds the
// components to check for reload dependencies.
func (pda *PartialDependencyAnalyser) buildDependencyGraph(virtualModule *annotator_dto.VirtualModule) {
	for _, comp := range virtualModule.ComponentsByHash {
		pda.processComponent(comp)
	}
}

// processComponent extracts dependencies from a single component.
//
// Takes comp (*annotator_dto.VirtualComponent) which is the component to
// process.
func (pda *PartialDependencyAnalyser) processComponent(
	comp *annotator_dto.VirtualComponent,
) {
	if comp.Source == nil {
		return
	}

	pda.registerImportAliases(comp)

	if comp.Source.ClientScript == "" {
		return
	}

	sourceName := pda.getSourceName(comp)
	importAliases := pda.collectImportAliases(comp)
	pda.recordDependencies(comp, sourceName, importAliases)

	pda.aliasToComponent[sourceName] = &componentInfo{
		sourcePath:   comp.Source.SourcePath,
		clientScript: comp.Source.ClientScript,
		location:     ast_domain.Location{},
	}
}

// registerImportAliases records all import aliases from a component.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the source
// component with import statements to record.
func (pda *PartialDependencyAnalyser) registerImportAliases(comp *annotator_dto.VirtualComponent) {
	for _, imp := range comp.Source.PikoImports {
		if imp.Alias == "" || imp.Alias == "_" {
			continue
		}
		pda.aliasToComponent[imp.Alias] = &componentInfo{
			sourcePath:   comp.Source.SourcePath,
			location:     imp.Location,
			clientScript: "",
		}
	}
}

// getSourceName returns the display name for a component.
//
// Takes comp (*annotator_dto.VirtualComponent) which is the component to get
// the name for.
//
// Returns string which is the partial name if set, otherwise the hashed name.
func (*PartialDependencyAnalyser) getSourceName(comp *annotator_dto.VirtualComponent) string {
	if comp.PartialName != "" {
		return comp.PartialName
	}
	return comp.HashedName
}

// collectImportAliases builds a set of valid import aliases.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the source
// component containing piko imports to scan.
//
// Returns map[string]bool which contains the non-blank, non-underscore import
// aliases as keys.
func (*PartialDependencyAnalyser) collectImportAliases(comp *annotator_dto.VirtualComponent) map[string]bool {
	importAliases := make(map[string]bool)
	for _, imp := range comp.Source.PikoImports {
		if imp.Alias != "" && imp.Alias != "_" {
			importAliases[imp.Alias] = true
		}
	}
	return importAliases
}

// recordDependencies stores which partials a component reloads.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the component
// to check.
// Takes sourceName (string) which identifies the source partial.
// Takes importAliases (map[string]bool) which holds valid import aliases.
func (pda *PartialDependencyAnalyser) recordDependencies(
	comp *annotator_dto.VirtualComponent,
	sourceName string,
	importAliases map[string]bool,
) {
	reloadedAliases := extractReloadedPartials(comp.Source.ClientScript)

	for alias := range reloadedAliases {
		if !importAliases[alias] {
			continue
		}
		if pda.dependencies[sourceName] == nil {
			pda.dependencies[sourceName] = make(map[string]bool)
		}
		pda.dependencies[sourceName][alias] = true
	}
}

// detectCycles finds all cycles in the dependency graph using depth-first
// search.
//
// Returns [][]string which contains all cycles found, where each cycle is a
// list of partial names.
func (pda *PartialDependencyAnalyser) detectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	path := make([]string, 0)

	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for neighbour := range pda.dependencies[node] {
			if !visited[neighbour] {
				dfs(neighbour)
			} else if recStack[neighbour] {
				cycle := extractCycle(path, neighbour)
				if len(cycle) > 0 {
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
	}

	for node := range pda.dependencies {
		if !visited[node] {
			dfs(node)
		}
	}

	return cycles
}

// cyclesToDiagnostics turns detected cycles into diagnostic messages.
//
// Takes cycles ([][]string) which contains the detected dependency cycles.
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the
// backup source path for diagnostics.
//
// Returns []*ast_domain.Diagnostic which contains error diagnostics for each
// unique cycle, or nil if no cycles exist.
func (pda *PartialDependencyAnalyser) cyclesToDiagnostics(
	cycles [][]string,
	mainComponent *annotator_dto.VirtualComponent,
) []*ast_domain.Diagnostic {
	if len(cycles) == 0 {
		return nil
	}

	diagnostics := make([]*ast_domain.Diagnostic, 0, len(cycles))

	seen := make(map[string]bool)

	for _, cycle := range cycles {
		normalisedCycle := normaliseCycle(cycle)
		cycleKey := strings.Join(normalisedCycle, " → ")

		if seen[cycleKey] {
			continue
		}
		seen[cycleKey] = true

		message := fmt.Sprintf(
			"Circular partial reload dependency detected: %s. "+
				"This will cause infinite reload loops at runtime.",
			cycleKey,
		)

		var location ast_domain.Location
		var sourcePath string
		if len(cycle) > 0 {
			if info, ok := pda.aliasToComponent[cycle[0]]; ok {
				location = info.location
				sourcePath = info.sourcePath
			}
		}

		if sourcePath == "" && mainComponent != nil && mainComponent.Source != nil {
			sourcePath = mainComponent.Source.SourcePath
		}

		diagnostics = append(diagnostics, ast_domain.NewDiagnosticWithCode(
			ast_domain.Error,
			message,
			cycleKey,
			annotator_dto.CodeCircularDependency,
			location,
			sourcePath,
		))
	}

	return diagnostics
}

// extractReloadedPartials finds all partial aliases used in reloadPartial and
// reloadGroup calls within a client script.
//
// Takes script (string) which contains the client script to parse.
//
// Returns map[string]bool which contains the set of partial aliases found.
func extractReloadedPartials(script string) map[string]bool {
	result := make(map[string]bool)

	extractPartialCallsToMap(script, "reloadPartial", result)

	extractReloadGroupCallsToMap(script, result)

	return result
}

// extractPartialCallsToMap finds calls to a named function that take a string
// argument and adds those string values to the result map.
//
// Takes script (string) which contains the source code to search.
// Takes functionName (string) which specifies the function name to look for.
// Takes result (map[string]bool) which stores the found partial aliases.
func extractPartialCallsToMap(script, functionName string, result map[string]bool) {
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
				result[alias] = true
			}
			index += endIndex + 1
		}
	}
}

// extractReloadGroupCallsToMap finds reloadGroup calls in a script and adds
// the aliases to the result map.
//
// Takes script (string) which contains the source code to search.
// Takes result (map[string]bool) which stores the found aliases.
func extractReloadGroupCallsToMap(script string, result map[string]bool) {
	patterns := []string{"reloadGroup([", "reloadGroup( ["}

	for _, pattern := range patterns {
		extractReloadGroupForPattern(script, pattern, result)
	}
}

// extractReloadGroupForPattern finds all matches of a pattern in the script
// and extracts aliases from the bracketed content that follows each match.
//
// Takes script (string) which contains the source text to search.
// Takes pattern (string) which specifies the text to find.
// Takes result (map[string]bool) which stores the found aliases.
func extractReloadGroupForPattern(script, pattern string, result map[string]bool) {
	index := 0
	for {
		foundIndex := strings.Index(script[index:], pattern)
		if foundIndex == -1 {
			break
		}
		index += foundIndex + len(pattern)

		endIndex := findMatchingBracket(script, index)
		if endIndex > index {
			arrayContent := script[index : endIndex-1]
			extractAliasesFromArrayToMap(arrayContent, result)
		}

		index = endIndex
	}
}

// findMatchingBracket finds the position after the matching closing bracket.
//
// Takes script (string) which contains the text to search.
// Takes startIndex (int) which is the position after the opening bracket.
//
// Returns int which is the position after the closing bracket, or startIndex if
// no matching bracket is found.
func findMatchingBracket(script string, startIndex int) int {
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

// extractAliasesFromArrayToMap parses a comma-separated string and adds valid
// aliases to a map.
//
// Takes content (string) which holds comma-separated values wrapped in single
// or double quotes.
// Takes result (map[string]bool) which stores the valid aliases found.
func extractAliasesFromArrayToMap(content string, result map[string]bool) {
	for part := range strings.SplitSeq(content, ",") {
		part = strings.TrimSpace(part)

		if len(part) >= 2 && part[0] == '\'' && part[len(part)-1] == '\'' {
			alias := part[1 : len(part)-1]
			if alias != "" && isValidPartialAlias(alias) {
				result[alias] = true
			}
			continue
		}

		if len(part) >= 2 && part[0] == '"' && part[len(part)-1] == '"' {
			alias := part[1 : len(part)-1]
			if alias != "" && isValidPartialAlias(alias) {
				result[alias] = true
			}
		}
	}
}

// extractCycle finds the cycle portion from a path starting at the target node.
//
// Takes path ([]string) which is the list of nodes visited so far.
// Takes target (string) which is the node where the cycle starts.
//
// Returns []string which holds the cycle from target back to itself, or nil
// if the target is not in the path.
func extractCycle(path []string, target string) []string {
	for i, node := range path {
		if node == target {
			cycle := make([]string, len(path)-i+1)
			copy(cycle, path[i:])
			cycle[len(cycle)-1] = target
			return cycle
		}
	}
	return nil
}

// normaliseCycle rotates a cycle so it starts with the smallest element in
// alphabetical order. This helps find duplicate cycles that are the same but
// were found from different starting points.
//
// Takes cycle ([]string) which is the cycle to put in standard form.
//
// Returns []string which is the cycle starting with its smallest element.
func normaliseCycle(cycle []string) []string {
	if len(cycle) <= 1 {
		return cycle
	}

	uniquePart := cycle[:len(cycle)-1]

	minIndex := 0
	for i := 1; i < len(uniquePart); i++ {
		if uniquePart[i] < uniquePart[minIndex] { //nolint:gosec // loop bounded
			minIndex = i
		}
	}

	result := make([]string, len(uniquePart)+1)
	for i := range len(uniquePart) {
		result[i] = uniquePart[(minIndex+i)%len(uniquePart)]
	}
	result[len(uniquePart)] = result[0]

	return result
}
