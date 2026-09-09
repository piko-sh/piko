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

package inspector_domain

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

const (
	// gorootPathPlaceholder replaces the Go installation prefix in generated paths.
	gorootPathPlaceholder = "$GOROOT"

	// modcachePathPlaceholder replaces the module cache prefix in generated paths.
	modcachePathPlaceholder = "$GOMODCACHE"
)

// sanitiseTypeDataPaths rewrites absolute file paths in generated type data so that they
// carry no information about the machine that produced them.
//
// Takes typeData (*inspector_dto.TypeData) which is mutated in place.
func sanitiseTypeDataPaths(typeData *inspector_dto.TypeData) {
	if typeData == nil {
		return
	}

	replacer := newPathSanitiser()
	if replacer == nil {
		return
	}

	if len(typeData.FileToPackage) > 0 {
		rewritten := make(map[string]string, len(typeData.FileToPackage))
		for path, packagePath := range typeData.FileToPackage {
			rewritten[replacer(path)] = packagePath
		}
		typeData.FileToPackage = rewritten
	}

	for _, pkg := range typeData.Packages {
		if pkg == nil {
			continue
		}
		sanitisePackagePaths(pkg, replacer)
	}
}

// sanitisePackagePaths rewrites every recorded file path within one package.
//
// Takes pkg (*inspector_dto.Package) which is mutated in place.
// Takes replacer (func(string) string) which rewrites a single path.
func sanitisePackagePaths(pkg *inspector_dto.Package, replacer func(string) string) {
	if len(pkg.FileImports) > 0 {
		rewritten := make(map[string]map[string]string, len(pkg.FileImports))
		for path, imports := range pkg.FileImports {
			rewritten[replacer(path)] = imports
		}
		pkg.FileImports = rewritten
	}

	for _, namedType := range pkg.NamedTypes {
		if namedType == nil {
			continue
		}
		namedType.DefinedInFilePath = replacer(namedType.DefinedInFilePath)
		for index := range namedType.Fields {
			namedType.Fields[index].DefinitionFilePath = replacer(namedType.Fields[index].DefinitionFilePath)
		}
		for _, method := range namedType.Methods {
			if method != nil {
				method.DefinitionFilePath = replacer(method.DefinitionFilePath)
			}
		}
	}

	for _, function := range pkg.Funcs {
		if function != nil {
			function.DefinitionFilePath = replacer(function.DefinitionFilePath)
		}
	}

	for _, variable := range pkg.Variables {
		if variable != nil {
			variable.DefinedInFilePath = replacer(variable.DefinedInFilePath)
		}
	}
}

// newPathSanitiser builds a rewriter for the current machine's Go layout.
//
// Returns func(string) string which rewrites one path, or nil when no prefixes are known.
func newPathSanitiser() func(string) string {
	type prefixReplacement struct {
		prefix      string
		placeholder string
	}

	var replacements []prefixReplacement

	if goroot := build.Default.GOROOT; goroot != "" {
		replacements = append(replacements, prefixReplacement{
			prefix:      filepath.Clean(goroot) + string(filepath.Separator),
			placeholder: gorootPathPlaceholder + "/",
		})
	}

	modcache := os.Getenv("GOMODCACHE")
	if modcache == "" {
		if gopath := build.Default.GOPATH; gopath != "" {
			modcache = filepath.Join(filepath.Clean(gopath), "pkg", "mod")
		}
	}
	if modcache != "" {
		replacements = append(replacements, prefixReplacement{
			prefix:      filepath.Clean(modcache) + string(filepath.Separator),
			placeholder: modcachePathPlaceholder + "/",
		})
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" && home != string(filepath.Separator) {
		replacements = append(replacements, prefixReplacement{
			prefix:      filepath.Clean(home) + string(filepath.Separator),
			placeholder: "$HOME/",
		})
	}

	if len(replacements) == 0 {
		return nil
	}

	return func(path string) string {
		if path == "" {
			return path
		}
		for _, replacement := range replacements {
			if strings.HasPrefix(path, replacement.prefix) {
				return replacement.placeholder + filepath.ToSlash(strings.TrimPrefix(path, replacement.prefix))
			}
		}
		return path
	}
}
