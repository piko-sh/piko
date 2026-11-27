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

// partialJSDependencyResolver computes the transitive set of partial JavaScript artefacts
// required by a page component. It uses PikoImports from each component's ParsedComponent
// to traverse the dependency graph.
type partialJSDependencyResolver struct {
	// artefactsByHashedName maps hashed component names to their generated artefacts. Used
	// to find dependencies and look up JSArtefactID values.
	artefactsByHashedName map[string]*generator_dto.GeneratedArtefact

	// partialJSLookup maps hashed partial names to their JS artefact IDs. Only partials that
	// have JS are included.
	partialJSLookup map[string]string

	// importPathToHashedName maps a module import path to its artefact's hashed name.
	importPathToHashedName map[string]string
}

// ResolveForPage computes all partial JS artefact IDs needed by a page. It performs a DFS
// traversal of PikoImports to find all embedded partials that have client-side
// JavaScript.
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

// PartialScriptNames maps the JS artefact ID of every partial script the resolver knows
// to the name of its partial.
//
// Returns map[string]string which maps JS artefact IDs to partial names, or nil when no
// partial has a script.
func (r *partialJSDependencyResolver) PartialScriptNames() map[string]string {
	if len(r.partialJSLookup) == 0 {
		return nil
	}
	names := make(map[string]string, len(r.partialJSLookup))
	for hashedName, jsArtefactID := range r.partialJSLookup {
		names[jsArtefactID] = r.artefactsByHashedName[hashedName].Component.PartialName
	}
	return names
}

// collectDependencies performs DFS to find all partial dependencies. It traverses
// PikoImports recursively, collecting JS artefact IDs along the way.
//
// Takes hashedName (string) which identifies the starting artefact.
// Takes visited (map[string]bool) which tracks visited nodes to avoid cycles.
// Takes artefactIDs (*[]string) which accumulates the collected JS artefact IDs.
// Takes seen (map[string]bool) which tracks already-added artefact IDs to avoid
// duplicates.
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

// newPartialJSDependencyResolver indexes partial scripts, with the last artefact winning
// even when it removes a script.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains all generated
// artefacts for the project.
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

		if isPartialWithScript(artefact) {
			resolver.partialJSLookup[hashedName] = artefact.JSArtefactID
		} else {
			delete(resolver.partialJSLookup, hashedName)
		}
	}

	return resolver
}

// isPartialWithScript reports whether an artefact is a partial that has client-side
// JavaScript.
//
// The decision uses the same client script target as the emitter, not the folder the
// source lives in, so a partial under a custom partials directory or from another module
// is found too.
//
// Takes artefact (*generator_dto.GeneratedArtefact) which is the artefact to check.
//
// Returns bool which is true when the artefact's script is a partial script.
func isPartialWithScript(artefact *generator_dto.GeneratedArtefact) bool {
	if artefact.JSArtefactID == "" {
		return false
	}
	target, err := generator_dto.ClientScriptTargetFor(artefact.Component)
	return err == nil && target.Kind == generator_dto.ClientScriptPartial
}
