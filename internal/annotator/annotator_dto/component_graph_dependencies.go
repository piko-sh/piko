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

package annotator_dto

import (
	"path/filepath"
	"strings"
)

// BuildReverseDependencyMapFromGraph creates a map from import-relative paths
// to the project-relative paths of components that depend on them.
//
// For example, if pages/login.pk imports components/card.pk, the returned map
// will contain: "components/card.pk" -> ["pages/login.pk"].
//
// Takes graph (*ComponentGraph) which provides the parsed components and their
// imports.
// Takes projectRoot (string) which is the absolute path to the project root,
// used to convert SourcePath to relative paths.
//
// Returns map[string][]string which maps each import-relative path to the list
// of project-relative paths of components that depend on it.
func BuildReverseDependencyMapFromGraph(
	graph *ComponentGraph,
	projectRoot string,
) map[string][]string {
	reverseDeps := make(map[string][]string)

	for _, component := range graph.Components {
		relativePath, err := filepath.Rel(projectRoot, component.SourcePath)
		if err != nil {
			continue
		}
		relativePath = filepath.ToSlash(relativePath)

		for _, pikoImport := range component.PikoImports {
			importRelativePath := extractImportRelativePath(pikoImport.Path)
			reverseDeps[importRelativePath] = append(reverseDeps[importRelativePath], relativePath)
		}
	}

	return reverseDeps
}

// GetTransitiveDependents performs a breadth-first search through the reverse
// dependency map to find all components transitively affected by a change to
// the component at changedPath.
//
// Takes reverseDeps (map[string][]string) which maps each path to the paths of
// components that depend on it.
// Takes changedPath (string) which is the project-relative path of the changed
// component.
//
// Returns []string which contains the project-relative paths of all
// transitively affected components, not including changedPath itself. Safe for
// cyclic graphs.
func GetTransitiveDependents(
	reverseDeps map[string][]string,
	changedPath string,
) []string {
	visited := make(map[string]bool)
	queue := []string{changedPath}
	var affected []string

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range reverseDeps[current] {
			if visited[dep] {
				continue
			}
			visited[dep] = true
			affected = append(affected, dep)
			queue = append(queue, dep)
		}
	}

	return affected
}

// FilterEntryPointsByRelativePaths returns only the entry points whose paths
// match the given project-relative paths. Each relative path is prefixed with
// moduleName + "/" before matching against EntryPoint.Path.
//
// Takes entryPoints ([]EntryPoint) which is the full set of entry points to
// filter.
// Takes relPaths ([]string) which contains the project-relative paths to keep.
// Takes moduleName (string) which is prepended to each relPath for matching.
//
// Returns []EntryPoint which contains only the matching entry points.
func FilterEntryPointsByRelativePaths(
	entryPoints []EntryPoint,
	relPaths []string,
	moduleName string,
) []EntryPoint {
	modulePrefix := moduleName + "/"

	pathSet := make(map[string]bool, len(relPaths))
	for _, p := range relPaths {
		pathSet[modulePrefix+p] = true
	}

	var filtered []EntryPoint
	for _, ep := range entryPoints {
		if pathSet[ep.Path] {
			filtered = append(filtered, ep)
		}
	}

	return filtered
}

// extractImportRelativePath gets the relative path from a piko import path by
// stripping the module name prefix (everything before the first slash).
//
// Takes importPath (string) which is the full import path to process.
//
// Returns string which is the part after the first slash, or the original path
// if no slash is found.
func extractImportRelativePath(importPath string) string {
	parts := strings.SplitN(importPath, "/", 2)
	if len(parts) > 1 {
		return filepath.ToSlash(parts[1])
	}
	return filepath.ToSlash(importPath)
}
