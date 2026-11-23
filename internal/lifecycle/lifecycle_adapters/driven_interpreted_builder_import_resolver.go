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
	"errors"
	"fmt"
	"iter"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
)

var (
	// errAmbiguousImport is returned when an import matches more than one component.
	errAmbiguousImport = errors.New("import matches more than one component")
)

// importResolver resolves piko imports to the components of a set of artefacts.
//
// An import of the project's own module names a project-relative path and matches only
// the component at exactly that path. Any other import matches a component at exactly its
// relative path, or else the one component whose path ends with it on a path boundary, so
// "card.pk" never matches "mycard.pk".
type importResolver struct {
	// packageByRelativePath maps each component's project-relative source path to its
	// canonical Go package path.
	packageByRelativePath map[string]string

	// moduleName is the project's Go module path.
	moduleName string

	// relativePaths holds every component's project-relative source path in sorted order, so
	// suffix matches do not depend on map iteration.
	relativePaths []string
}

// resolve finds the one component an import names.
//
// Takes importPath (string) which is the piko import path.
//
// Returns relativePath (string) which is the component's project-relative source path, or
// empty when no component matches.
// Returns packagePath (string) which is the component's canonical Go package path, or
// empty when no component matches.
// Returns err (error) which wraps errAmbiguousImport when several components match.
func (r *importResolver) resolve(importPath string) (relativePath, packagePath string, err error) {
	matches := r.matches(importPath)
	switch len(matches) {
	case 0:
		return "", "", nil
	case 1:
		return matches[0], r.packageByRelativePath[matches[0]], nil
	default:
		return "", "", fmt.Errorf("%w: %s", errAmbiguousImport, strings.Join(matches, ", "))
	}
}

// matches lists the project-relative source paths of every component an import could
// name.
//
// Takes importPath (string) which is the piko import path.
//
// Returns []string which holds one path for an exact match, every boundary suffix match
// in sorted order otherwise, or nothing when no component matches.
func (r *importResolver) matches(importPath string) []string {
	relativePath, local := r.importRelativePath(importPath)
	if relativePath == "" {
		return nil
	}
	if _, exact := r.packageByRelativePath[relativePath]; exact {
		return []string{relativePath}
	}
	if local {
		return nil
	}
	var suffixMatches []string
	for _, candidate := range r.relativePaths {
		if strings.HasSuffix(candidate, "/"+relativePath) {
			suffixMatches = append(suffixMatches, candidate)
		}
	}
	return suffixMatches
}

// importRelativePath gets the relative path from a piko import path.
//
// Takes importPath (string) which is the full import path to process.
//
// Returns relativePath (string) which is the project-relative path for an import of this
// module, or for another module the path after its first element.
// Returns local (bool) which is true when the import names this module.
func (r *importResolver) importRelativePath(importPath string) (relativePath string, local bool) {
	if r.moduleName != "" {
		if rel, isLocal := strings.CutPrefix(importPath, r.moduleName+"/"); isLocal {
			return rel, true
		}
	}
	if _, rel, ok := strings.Cut(importPath, "/"); ok {
		return rel, false
	}
	return importPath, false
}

// newImportResolver indexes the components of artefacts by project-relative source path.
//
// Takes projectRoot (string) which is the directory source paths are relative to.
// Takes moduleName (string) which is the project's Go module path.
// Takes artefacts (iter.Seq[*generator_dto.GeneratedArtefact]) which yields the artefacts
// whose components imports may name.
//
// Returns *importResolver which resolves imports against those components.
func newImportResolver(projectRoot, moduleName string, artefacts iter.Seq[*generator_dto.GeneratedArtefact]) *importResolver {
	packageByRelativePath := make(map[string]string)
	for artefact := range artefacts {
		component, _ := generator_domain.GetMainComponent(artefact.Result)
		if component == nil {
			continue
		}
		packageByRelativePath[componentRelativePath(projectRoot, component.Source.SourcePath)] = component.CanonicalGoPackagePath
	}
	return &importResolver{
		packageByRelativePath: packageByRelativePath,
		moduleName:            moduleName,
		relativePaths:         slices.Sorted(maps.Keys(packageByRelativePath)),
	}
}

// componentRelativePath returns a component source path relative to the project root,
// with forward slashes.
//
// Takes projectRoot (string) which is the directory to make the path relative to.
// Takes sourcePath (string) which is the component's source path.
//
// Returns string which is the relative path, or sourcePath itself when it cannot be made
// relative.
func componentRelativePath(projectRoot, sourcePath string) string {
	relativePath, err := filepath.Rel(projectRoot, sourcePath)
	if err != nil {
		relativePath = sourcePath
	}
	return filepath.ToSlash(relativePath)
}
