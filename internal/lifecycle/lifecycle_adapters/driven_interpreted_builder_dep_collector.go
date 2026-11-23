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
	"maps"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

// collectDependencies creates a dependency collector and returns all dependencies for a
// component.
//
// Takes targetRelativePath (string) which specifies the component to collect dependencies
// for.
//
// Returns []*generator_dto.GeneratedArtefact which contains the component and all its
// transitive dependencies.
// Returns error when dependency collection fails.
func (o *InterpretedBuildOrchestrator) collectDependencies(
	targetRelativePath string,
) ([]*generator_dto.GeneratedArtefact, error) {
	collector := &dependencyCollector{
		orchestrator: o,
		resolver:     newImportResolver(o.projectRoot, o.moduleName, maps.Values(o.artefactByPackagePath)),
		visited:      make(map[string]bool),
		result:       make([]*generator_dto.GeneratedArtefact, 0),
	}
	return collector.collect(targetRelativePath)
}

// dependencyCollector holds the state for gathering build dependencies.
type dependencyCollector struct {
	// orchestrator holds a reference to the parent build orchestrator's shared state.
	orchestrator *InterpretedBuildOrchestrator

	// resolver resolves imports and relative paths to the orchestrator's components.
	resolver *importResolver

	// visited tracks paths already processed to prevent collecting them twice.
	visited map[string]bool

	// result holds the collected generated artefacts.
	result []*generator_dto.GeneratedArtefact
}

// collect recursively collects a component and its dependencies.
//
// Takes relativePath (string) which specifies the relative path to the component.
//
// Returns []*generator_dto.GeneratedArtefact which contains the collected component and
// all its dependencies.
// Returns error when recursive collection fails.
func (collector *dependencyCollector) collect(relativePath string) ([]*generator_dto.GeneratedArtefact, error) {
	if err := collector.collectRecursive(relativePath); err != nil {
		return nil, fmt.Errorf("collecting dependencies for %q: %w", relativePath, err)
	}
	return collector.result, nil
}

// collectRecursive is the recursive part of dependency collection.
//
// Takes relativePath (string) which specifies the path to collect.
//
// Returns error when the artefact cannot be found or dependencies fail to collect.
func (collector *dependencyCollector) collectRecursive(relativePath string) error {
	if collector.visited[relativePath] {
		return nil
	}
	collector.visited[relativePath] = true

	artefact, virtualComponent, err := collector.findArtefactByRelativePath(relativePath)
	if err != nil {
		return fmt.Errorf("finding artefact for %q: %w", relativePath, err)
	}

	if err := collector.collectImportDependencies(virtualComponent); err != nil {
		return fmt.Errorf("collecting import dependencies for %q: %w", relativePath, err)
	}

	collector.result = append(collector.result, artefact)
	return nil
}

// findArtefactByRelativePath finds an artefact by its relative path.
//
// Takes relativePath (string) which specifies the path relative to the project root.
//
// Returns *generator_dto.GeneratedArtefact which is the matching artefact.
// Returns *annotator_dto.VirtualComponent which is the main component of the artefact.
// Returns error when no artefact matches the given relative path.
func (collector *dependencyCollector) findArtefactByRelativePath(
	relativePath string,
) (*generator_dto.GeneratedArtefact, *annotator_dto.VirtualComponent, error) {
	packagePath, found := collector.resolver.packageByRelativePath[relativePath]
	if !found {
		return nil, nil, fmt.Errorf("could not find artefact for component: %s", relativePath)
	}
	artefact := collector.orchestrator.artefactByPackagePath[packagePath]
	if artefact == nil {
		return nil, nil, fmt.Errorf("could not find artefact for component: %s", relativePath)
	}
	component, _ := generator_domain.GetMainComponent(artefact.Result)
	if component == nil {
		return nil, nil, fmt.Errorf("could not find main component for: %s", relativePath)
	}
	return artefact, component, nil
}

// collectImportDependencies collects all dependencies from a component's piko imports.
//
// Takes virtualComponent (*annotator_dto.VirtualComponent) which provides the component
// whose imports should be processed.
//
// Returns error when an import matches more than one component or recursive dependency
// collection fails.
func (collector *dependencyCollector) collectImportDependencies(virtualComponent *annotator_dto.VirtualComponent) error {
	for _, pikoImport := range virtualComponent.Source.PikoImports {
		dependencyRelativePath, _, err := collector.resolver.resolve(pikoImport.Path)
		if err != nil {
			return fmt.Errorf("resolving import %q: %w", pikoImport.Path, err)
		}
		if dependencyRelativePath == "" {
			continue
		}
		if err := collector.collectRecursive(dependencyRelativePath); err != nil {
			return fmt.Errorf("collecting recursive dependency %q: %w", dependencyRelativePath, err)
		}
	}
	return nil
}
