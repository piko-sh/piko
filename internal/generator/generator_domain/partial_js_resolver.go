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

package generator_domain

import (
	"piko.sh/piko/internal/generator/generator_dto"
)

// partialJSDependencyResolver computes the transitive set of partial JavaScript
// artefacts required by a page component. It uses PikoImports from each
// component's ParsedComponent to traverse the dependency graph.
type partialJSDependencyResolver struct {
	// artefactsByHashedName maps hashed component names to their generated
	// artefacts. Used to find dependencies and look up JSArtefactID values.
	artefactsByHashedName map[string]*generator_dto.GeneratedArtefact

	// partialJSLookup maps hashed partial names to their JS artefact IDs.
	// Only partials that have JS are included.
	partialJSLookup map[string]string

	// importPathToHashedName maps a module import path to its artefact's hashed name.
	importPathToHashedName map[string]string
}

// ResolveForPage computes all partial JS artefact IDs needed by a page.
// It performs a DFS traversal of PikoImports to find all embedded partials
// that have client-side JavaScript.
//
// Takes pageHashedName (string) which identifies the page component.
//
// Returns []string containing unique, deduplicated JS artefact IDs.
// Returns nil if there are no partial JS dependencies.
func (r *partialJSDependencyResolver) ResolveForPage(pageHashedName string) []string {
	visited := make(map[string]bool)
	seen := make(map[string]bool)
	var artefactIDs []string

	r.collectDependencies(pageHashedName, visited, &artefactIDs, seen)

	if len(artefactIDs) == 0 {
		return nil
	}
	return artefactIDs
}

// collectDependencies performs DFS to find all partial dependencies.
// It traverses PikoImports recursively, collecting JS artefact IDs along the
// way.
//
// Takes hashedName (string) which identifies the starting artefact.
// Takes visited (map[string]bool) which tracks visited nodes to avoid cycles.
// Takes artefactIDs (*[]string) which accumulates the collected JS artefact
// IDs.
// Takes seen (map[string]bool) which tracks already-added artefact IDs to
// avoid duplicates.
func (r *partialJSDependencyResolver) collectDependencies(
	hashedName string,
	visited map[string]bool,
	artefactIDs *[]string,
	seen map[string]bool,
) {
	if visited[hashedName] {
		return
	}
	visited[hashedName] = true

	artefact, ok := r.artefactsByHashedName[hashedName]
	if !ok || artefact.Component == nil || artefact.Component.Source == nil {
		return
	}

	for _, pikoImport := range artefact.Component.Source.PikoImports {
		importPath := pikoImport.Path

		importedHashedName, found := r.importPathToHashedName[importPath]
		if !found {
			continue
		}

		if jsArtefactID, hasJS := r.partialJSLookup[importedHashedName]; hasJS {
			if !seen[jsArtefactID] {
				*artefactIDs = append(*artefactIDs, jsArtefactID)
				seen[jsArtefactID] = true
			}
		}

		r.collectDependencies(importedHashedName, visited, artefactIDs, seen)
	}
}

// newPartialJSDependencyResolver creates a resolver from the given artefacts.
// It builds internal lookup maps for efficient dependency resolution.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains all
// generated artefacts for the project.
//
// Returns *partialJSDependencyResolver which is ready to resolve dependencies.
func newPartialJSDependencyResolver(artefacts []*generator_dto.GeneratedArtefact) *partialJSDependencyResolver {
	resolver := &partialJSDependencyResolver{
		artefactsByHashedName:  make(map[string]*generator_dto.GeneratedArtefact),
		partialJSLookup:        make(map[string]string),
		importPathToHashedName: make(map[string]string),
	}

	for _, artefact := range artefacts {
		if artefact.Component == nil {
			continue
		}

		vc := artefact.Component
		hashedName := vc.HashedName

		resolver.artefactsByHashedName[hashedName] = artefact

		if vc.Source != nil && vc.Source.ModuleImportPath != "" {
			resolver.importPathToHashedName[vc.Source.ModuleImportPath] = hashedName
		}

		if vc.Source != nil && vc.Source.ComponentType == "partial" && artefact.JSArtefactID != "" {
			resolver.partialJSLookup[hashedName] = artefact.JSArtefactID
		}
	}

	return resolver
}
