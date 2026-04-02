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

package lifecycle_adapters

import (
	"fmt"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

// topologicallySortArtefacts sorts artefacts by their dependencies.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// artefacts to sort.
//
// Returns []*generator_dto.GeneratedArtefact which contains the artefacts in
// dependency order.
// Returns error when a dependency cycle is found.
func (o *InterpretedBuildOrchestrator) topologicallySortArtefacts(
	artefacts []*generator_dto.GeneratedArtefact,
) ([]*generator_dto.GeneratedArtefact, error) {
	sorter := &topologicalSorter{
		orchestrator:          o,
		artefactByPackagePath: make(map[string]*generator_dto.GeneratedArtefact),
		adjacency:             make(map[string][]string),
		inDegree:              make(map[string]int),
		allPaths:              make([]string, 0, len(artefacts)),
	}
	return sorter.sort(artefacts)
}

// topologicalSorter holds the state needed to sort build artefacts by their
// dependencies using topological sorting.
type topologicalSorter struct {
	// orchestrator holds the parent orchestrator for accessing project root.
	orchestrator *InterpretedBuildOrchestrator

	// artefactByPackagePath maps canonical Go package paths to their generated
	// artefacts.
	artefactByPackagePath map[string]*generator_dto.GeneratedArtefact

	// adjacency maps each package path to its list of dependent packages.
	adjacency map[string][]string

	// inDegree tracks how many unprocessed dependencies each package path has.
	inDegree map[string]int

	// allPaths holds all package paths in the order they were added.
	allPaths []string
}

// sort performs the topological sort using Kahn's algorithm.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// items to sort by their dependencies.
//
// Returns []*generator_dto.GeneratedArtefact which contains the artefacts in
// dependency order.
// Returns error when a dependency cycle is detected or validation fails.
func (sorter *topologicalSorter) sort(artefacts []*generator_dto.GeneratedArtefact) ([]*generator_dto.GeneratedArtefact, error) {
	sorter.buildArtefactMap(artefacts)
	sorter.buildDependencyGraph()
	sorted := sorter.executeKahnsAlgorithm()
	return sorter.validateAndReturn(sorted)
}

// buildArtefactMap creates a map from package paths to artefacts.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which provides the
// generated artefacts to index by their package path.
func (sorter *topologicalSorter) buildArtefactMap(artefacts []*generator_dto.GeneratedArtefact) {
	for _, artefact := range artefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component != nil {
			sorter.artefactByPackagePath[component.CanonicalGoPackagePath] = artefact
		}
	}
}

// buildDependencyGraph builds the adjacency list and in-degree map for all
// packages.
func (sorter *topologicalSorter) buildDependencyGraph() {
	for packagePath, artefact := range sorter.artefactByPackagePath {
		sorter.initialiseGraphNode(packagePath)
		sorter.processArtefactImports(packagePath, artefact)
	}
}

// initialiseGraphNode adds a node to the dependency graph.
//
// Takes packagePath (string) which specifies the package path to add as a node.
func (sorter *topologicalSorter) initialiseGraphNode(packagePath string) {
	sorter.allPaths = append(sorter.allPaths, packagePath)
	sorter.inDegree[packagePath] = 0
	if _, ok := sorter.adjacency[packagePath]; !ok {
		sorter.adjacency[packagePath] = []string{}
	}
}

// processArtefactImports processes all piko imports for an artefact.
//
// Takes packagePath (string) which identifies the package being processed.
// Takes artefact (*generator_dto.GeneratedArtefact) which contains the artefact
// with piko imports to process.
func (sorter *topologicalSorter) processArtefactImports(packagePath string, artefact *generator_dto.GeneratedArtefact) {
	component, _ := generator_domain.GetMainComponent(artefact.Result)
	if component == nil {
		return
	}

	for _, pikoImport := range component.Source.PikoImports {
		sorter.processImport(packagePath, pikoImport.Path)
	}
}

// processImport handles a single import and updates the dependency graph.
//
// Takes packagePath (string) which is the package being processed.
// Takes importPath (string) which is the import path to resolve.
func (sorter *topologicalSorter) processImport(packagePath, importPath string) {
	importRelativePath := sorter.extractRelativePath(importPath)
	dependencyPackagePath := sorter.findDependencyPackagePath(importRelativePath)

	if dependencyPackagePath == "" || sorter.artefactByPackagePath[dependencyPackagePath] == nil {
		return
	}

	sorter.adjacency[dependencyPackagePath] = append(sorter.adjacency[dependencyPackagePath], packagePath)
	sorter.inDegree[packagePath]++
}

// extractRelativePath extracts the relative path portion of an import path.
//
// Takes importPath (string) which is the full import path to process.
//
// Returns string which is the path after the first slash, or the original
// path if no slash is present.
func (*topologicalSorter) extractRelativePath(importPath string) string {
	parts := strings.SplitN(importPath, "/", 2)
	if len(parts) > 1 {
		return filepath.ToSlash(parts[1])
	}
	return filepath.ToSlash(importPath)
}

// findDependencyPackagePath finds the package path of an artefact matching the
// import.
//
// Takes importRelativePath (string) which specifies the relative import path
// to search for.
//
// Returns string which is the package path of the matching artefact, or an
// empty string if no match is found.
func (sorter *topologicalSorter) findDependencyPackagePath(importRelativePath string) string {
	for candidatePackagePath, candidateArtefact := range sorter.artefactByPackagePath {
		if sorter.matchesImport(candidateArtefact, importRelativePath) {
			return candidatePackagePath
		}
	}
	return ""
}

// matchesImport checks if an artefact matches the given import path.
//
// Takes artefact (*generator_dto.GeneratedArtefact) which is the artefact
// to check.
// Takes importRelativePath (string) which is the import path to match against.
//
// Returns bool which is true if the artefact's relative path matches or ends
// with the import path.
func (sorter *topologicalSorter) matchesImport(artefact *generator_dto.GeneratedArtefact, importRelativePath string) bool {
	candidateComponent, _ := generator_domain.GetMainComponent(artefact.Result)
	if candidateComponent == nil {
		return false
	}

	candidateRelativePath, err := filepath.Rel(sorter.orchestrator.projectRoot, candidateComponent.Source.SourcePath)
	if err != nil {
		candidateRelativePath = candidateComponent.Source.SourcePath
	}
	candidateRelativePath = filepath.ToSlash(candidateRelativePath)

	return candidateRelativePath == importRelativePath || strings.HasSuffix(candidateRelativePath, importRelativePath)
}

// executeKahnsAlgorithm performs Kahn's algorithm to produce a topologically
// sorted list.
//
// Returns []*generator_dto.GeneratedArtefact which contains the artefacts
// ordered by their dependencies.
func (sorter *topologicalSorter) executeKahnsAlgorithm() []*generator_dto.GeneratedArtefact {
	queue := sorter.initialiseQueue()
	sorted := make([]*generator_dto.GeneratedArtefact, 0, len(sorter.artefactByPackagePath))

	for len(queue) > 0 {
		packagePath := queue[0]
		queue = queue[1:]
		sorted = append(sorted, sorter.artefactByPackagePath[packagePath])
		queue = sorter.processNodeDependents(packagePath, queue)
	}

	return sorted
}

// initialiseQueue creates the starting queue with nodes that have no incoming
// edges.
//
// Returns []string which contains paths that have no dependencies.
func (sorter *topologicalSorter) initialiseQueue() []string {
	var queue []string
	for _, path := range sorter.allPaths {
		if sorter.inDegree[path] == 0 {
			queue = append(queue, path)
		}
	}
	return queue
}

// processNodeDependents decrements in-degrees and adds newly ready nodes to the
// queue.
//
// Takes packagePath (string) which identifies the node whose dependents to
// process.
// Takes queue ([]string) which holds nodes ready for processing.
//
// Returns []string which is the updated queue with any newly ready nodes added.
func (sorter *topologicalSorter) processNodeDependents(packagePath string, queue []string) []string {
	for _, dependent := range sorter.adjacency[packagePath] {
		sorter.inDegree[dependent]--
		if sorter.inDegree[dependent] == 0 {
			queue = append(queue, dependent)
		}
	}
	return queue
}

// validateAndReturn checks the sorted result and returns it if valid.
//
// Takes sorted ([]*generator_dto.GeneratedArtefact) which is the sorted list
// to check.
//
// Returns []*generator_dto.GeneratedArtefact which is the checked sorted list.
// Returns error when a circular dependency is found.
func (sorter *topologicalSorter) validateAndReturn(sorted []*generator_dto.GeneratedArtefact) ([]*generator_dto.GeneratedArtefact, error) {
	if len(sorted) != len(sorter.artefactByPackagePath) {
		return nil, fmt.Errorf("circular dependency detected: %d of %d components could not be sorted",
			len(sorter.artefactByPackagePath)-len(sorted), len(sorter.artefactByPackagePath))
	}
	return sorted, nil
}
