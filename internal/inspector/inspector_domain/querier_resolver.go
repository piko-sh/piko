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

// This file focuses on the logic for resolving package aliases and type names
// from various contexts, including file-scoped imports and local aliases.

import (
	goast "go/ast"
	"path/filepath"
	"slices"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// findNamedType locates a type DTO from the cache by resolving a package
// alias from the perspective of an importer.
//
// Takes typeName (string) which specifies the name of the type to find.
// Takes typePackageAlias (string) which specifies the package alias used in the
// importing file.
// Takes importerPackagePath (string) which specifies the package path of the
// importing code.
// Takes importerFilePath (string) which specifies the file path of the
// importing code.
//
// Returns *inspector_dto.Type which is the resolved type, or nil if not found.
// Returns string which is the resolved package path.
func (ti *TypeQuerier) findNamedType(typeName, typePackageAlias, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
	packagePath, pathFound := ti.resolvePackagePath(typePackageAlias, importerPackagePath, importerFilePath)

	if !pathFound {
		return ti.findTypeThroughAliasFallback(typeName, typePackageAlias, importerPackagePath)
	}

	return ti.lookupTypeInPackage(typeName, typePackageAlias, packagePath)
}

// findTypeThroughAliasFallback handles the case where an alias refers to the
// current package. For example, dto_alias.TransactionDto where dto_alias is an
// import alias for the dtos package, but we are trying to resolve it from
// within that same package.
//
// Takes typeName (string) which is the name of the type to find.
// Takes typePackageAlias (string) which is the import alias used for the package.
// Takes importerPackagePath (string) which is the package path being imported.
//
// Returns *inspector_dto.Type which is the resolved type, or nil if not found.
// Returns string which is the package alias, or empty if not found.
//
// Searches packages in sorted order for consistent results.
func (ti *TypeQuerier) findTypeThroughAliasFallback(typeName, typePackageAlias, importerPackagePath string) (*inspector_dto.Type, string) {
	pkgPaths := make([]string, 0, len(ti.typeData.Packages))
	for packagePath := range ti.typeData.Packages {
		pkgPaths = append(pkgPaths, packagePath)
	}
	slices.Sort(pkgPaths)

	for _, packagePath := range pkgPaths {
		pkg := ti.typeData.Packages[packagePath]
		if namedType := ti.findTypeViaFileImports(pkg, typeName, typePackageAlias, importerPackagePath); namedType != nil {
			return namedType, typePackageAlias
		}
	}
	return nil, ""
}

// findTypeViaFileImports checks if a package's file imports contain an alias
// pointing to the target path.
//
// Iterates over file imports in sorted order for deterministic results.
//
// Takes pkg (*inspector_dto.Package) which provides the package to search.
// Takes typeName (string) which specifies the type name to find.
// Takes typePackageAlias (string) which is the import alias to match.
// Takes targetPackagePath (string) which is the expected import path.
//
// Returns *inspector_dto.Type which is the found type, or nil if not found.
func (ti *TypeQuerier) findTypeViaFileImports(pkg *inspector_dto.Package, typeName, typePackageAlias, targetPackagePath string) *inspector_dto.Type {
	filePaths := make([]string, 0, len(pkg.FileImports))
	for filePath := range pkg.FileImports {
		filePaths = append(filePaths, filePath)
	}
	slices.Sort(filePaths)

	for _, filePath := range filePaths {
		fileImports := pkg.FileImports[filePath]
		if aliasPath, found := fileImports[typePackageAlias]; found && aliasPath == targetPackagePath {
			if targetPackage, ok := ti.typeData.Packages[targetPackagePath]; ok && targetPackage != nil && targetPackage.NamedTypes != nil {
				if namedType, typeFound := targetPackage.NamedTypes[typeName]; typeFound {
					return namedType
				}
			}
		}
	}
	return nil
}

// lookupTypeInPackage looks up a type in the resolved package and handles
// alias access detection.
//
// Takes typeName (string) which is the name of the type to look up.
// Takes typePackageAlias (string) which is the import
// alias used to access the type.
// Takes packagePath (string) which is the full import path of the package.
//
// Returns *inspector_dto.Type which is the resolved type, or nil if not found.
// Returns string which is the package alias, or empty if the type was not found.
func (ti *TypeQuerier) lookupTypeInPackage(typeName, typePackageAlias, packagePath string) (*inspector_dto.Type, string) {
	pkg, pkgExists := ti.typeData.Packages[packagePath]
	if !pkgExists || pkg == nil || pkg.NamedTypes == nil {
		return nil, ""
	}

	namedType, typeFound := pkg.NamedTypes[typeName]
	if !typeFound {
		return nil, ""
	}

	if typePackageAlias != pkg.Name {
		aliasAccessType := *namedType
		aliasAccessType.IsAlias = true
		return &aliasAccessType, typePackageAlias
	}

	return namedType, typePackageAlias
}

// resolvePackagePath determines the canonical package import path from a
// given alias or name. It tries resolution strategies in order of priority.
//
// Takes aliasToResolve (string) which is the package alias or name to resolve.
// Takes importerPackagePath (string) which is the import path of the package where
// the lookup is happening.
// Takes importerFilePath (string) which is the file path used for file-scoped
// import resolution.
//
// Returns string which is the resolved canonical package import path.
// Returns bool which indicates whether resolution succeeded.
func (ti *TypeQuerier) resolvePackagePath(aliasToResolve, importerPackagePath, importerFilePath string) (string, bool) {
	importerPackage, ok := ti.getImporterPackage(importerPackagePath)
	if !ok {
		return "", false
	}

	if path, ok := ti.resolveFromFileScope(aliasToResolve, importerPackage, importerFilePath); ok {
		return path, true
	}

	if path, ok := ti.resolveWithFallbacks(aliasToResolve, importerPackage, importerPackagePath); ok {
		return path, true
	}

	return "", false
}

// getImporterPackage retrieves the package DTO for the given importer path,
// handling the special case for no-gomod projects where the path may be
// represented by the "command-line-arguments" package.
//
// Takes importerPackagePath (string) which is the import path of the package to
// retrieve.
//
// Returns *inspector_dto.Package which is the package metadata if found.
// Returns bool which indicates whether the package was found.
func (ti *TypeQuerier) getImporterPackage(importerPackagePath string) (*inspector_dto.Package, bool) {
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return nil, false
	}
	importerPackage, ok := ti.typeData.Packages[importerPackagePath]
	if !ok {
		if filepath.IsAbs(importerPackagePath) {
			if cmdLinePackage, exists := ti.typeData.Packages["command-line-arguments"]; exists {
				return cmdLinePackage, true
			}
		}
		return nil, false
	}
	return importerPackage, true
}

// resolveFromFileScope attempts to resolve a package alias using the import
// map of the specific file where the lookup is occurring.
//
// Takes aliasToResolve (string) which is the package alias to look up.
// Takes importerPackage (*inspector_dto.Package) which contains the file imports.
// Takes importerFilePath (string) which identifies the file for scoped lookup.
//
// Returns string which is the resolved import path.
// Returns bool which indicates whether the alias was found.
func (ti *TypeQuerier) resolveFromFileScope(aliasToResolve string, importerPackage *inspector_dto.Package, importerFilePath string) (string, bool) {
	if importerFilePath == "" {
		return "", false
	}

	fileImports, fileFound := importerPackage.FileImports[importerFilePath]
	if !fileFound {
		return "", false
	}

	if resolvedPath, aliasFound := fileImports[aliasToResolve]; aliasFound {
		return resolvedPath, true
	}

	return ti.resolveByImportedPackageName(fileImports, aliasToResolve)
}

// resolveByImportedPackageName searches a file's imports to see if the alias
// matches the actual name of an imported package.
//
// Takes fileImports (map[string]string) which maps import aliases to package
// paths.
// Takes aliasToResolve (string) which is the package name to search for.
//
// Returns string which is the package path if found, or empty string.
// Returns bool which indicates whether a matching package was found.
//
// Note: iterates over imports in sorted order to ensure deterministic resolution
// when multiple packages share the same name. Without sorting, map iteration
// order is random and could cause different packages to be selected across runs.
func (ti *TypeQuerier) resolveByImportedPackageName(fileImports map[string]string, aliasToResolve string) (string, bool) {
	aliases := make([]string, 0, len(fileImports))
	for alias := range fileImports {
		aliases = append(aliases, alias)
	}
	slices.Sort(aliases)

	for _, alias := range aliases {
		importedPackagePath := fileImports[alias]
		importedPackageData, pkgDataFound := ti.typeData.Packages[importedPackagePath]
		if pkgDataFound && importedPackageData.Name == aliasToResolve {
			return importedPackagePath, true
		}
	}
	return "", false
}

// resolveWithFallbacks attempts several lower-priority strategies for
// resolving a package path.
//
// Takes aliasToResolve (string) which is the alias or path to resolve.
// Takes importerPackage (*inspector_dto.Package) which is the package context.
// Takes importerPackagePath (string) which is the fallback path if DTO is empty.
//
// Returns string which is the resolved package path.
// Returns bool which indicates whether resolution succeeded.
func (ti *TypeQuerier) resolveWithFallbacks(aliasToResolve string, importerPackage *inspector_dto.Package, importerPackagePath string) (string, bool) {
	if importerPackage != nil && importerPackage.Name == aliasToResolve {
		if importerPackage.Path != "" {
			return importerPackage.Path, true
		}
		return importerPackagePath, true
	}

	if _, pkgExists := ti.typeData.Packages[aliasToResolve]; pkgExists {
		return aliasToResolve, true
	}

	if filepath.IsAbs(aliasToResolve) && aliasToResolve == importerPackagePath {
		if _, exists := ti.typeData.Packages["command-line-arguments"]; exists {
			return "command-line-arguments", true
		}
	}

	return "", false
}

// findNamedTypeInDotPackage handles looking for a type that may have been
// dot-imported.
//
// Takes typeName (string) which is the name of the type to find.
// Takes importerPackagePath (string) which is the package path performing the
// import.
// Takes importerFilePath (string) which is the file path where the import
// occurs.
//
// Returns *inspector_dto.Type which is the found type, or nil if not found.
// Returns string which is the canonical package name, or empty if not found.
func (ti *TypeQuerier) findNamedTypeInDotPackage(typeName, importerPackagePath, importerFilePath string) (*inspector_dto.Type, string) {
	if importerFilePath == "" {
		return nil, ""
	}

	importerPackage, ok := ti.typeData.Packages[importerPackagePath]
	if !ok {
		return nil, ""
	}

	fileImports, fileFound := importerPackage.FileImports[importerFilePath]
	if !fileFound {
		return nil, ""
	}

	dotImportedPackagePath, isDotImported := fileImports["."]
	if !isDotImported {
		return nil, ""
	}

	dotImportedPackage, ok := ti.typeData.Packages[dotImportedPackagePath]
	if !ok {
		return nil, ""
	}

	namedType, typeFound := dotImportedPackage.NamedTypes[typeName]
	if !typeFound {
		return nil, ""
	}

	return namedType, dotImportedPackage.Name
}

// resolveLocalAlias tries to resolve one level of a type alias in the
// local package files.
//
// Takes expression (goast.Expr) which is the expression to resolve.
// Takes currentFilePath (string) which identifies the file containing
// the type.
//
// Returns goast.Expr which is the resolved expression the alias
// points to.
// Returns string which is the file path for the next step.
// Returns bool which is true when an alias was found and resolved.
func (ti *TypeQuerier) resolveLocalAlias(expression goast.Expr, currentFilePath string) (goast.Expr, string, bool) {
	identifier, isIdent := expression.(*goast.Ident)
	if !isIdent {
		return nil, "", false
	}
	typeName := identifier.Name

	file, ok := ti.localPackageFiles[currentFilePath]
	if !ok {
		return nil, "", false
	}

	var (
		nextExpr    goast.Expr
		foundInFile bool
	)
	goast.Inspect(file, func(n goast.Node) bool {
		ts, ok := n.(*goast.TypeSpec)
		if !ok || ts.Name.Name != typeName {
			return true
		}

		if ts.Assign.IsValid() {
			nextExpr = ts.Type
			foundInFile = true
			return false
		}
		return true
	})

	if foundInFile {
		return nextExpr, currentFilePath, true
	}

	return nil, "", false
}

// ResolvePackageAlias resolves a package alias to its canonical import path
// from a specific file and package context. For example, given alias "url"
// from a file that imports "net/url", this returns "net/url".
//
// Takes aliasToResolve (string) which is the package alias to resolve.
// Takes importerPackagePath (string) which is the package path of the importing
// file.
// Takes importerFilePath (string) which is the file path of the importing
// file.
//
// Returns string which is the canonical import path, or empty if not found.
func (ti *TypeQuerier) ResolvePackageAlias(aliasToResolve, importerPackagePath, importerFilePath string) string {
	canonicalPath, found := ti.resolvePackagePath(aliasToResolve, importerPackagePath, importerFilePath)
	if !found {
		return ""
	}
	return canonicalPath
}
