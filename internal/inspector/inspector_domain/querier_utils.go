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

// This file contains the core business logic and helper utilities for the Go source code querier.

import (
	"fmt"
	goast "go/ast"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"piko.sh/piko/internal/inspector/inspector_dto"
)

// GetAllPackages returns all packages from the TypeData.
//
// This is used for package-wide searches when the caller needs to iterate
// over all known packages.
//
// Returns map[string]*inspector_dto.Package which contains all packages keyed
// by their import path, or an empty map if no packages are available.
func (ti *TypeQuerier) GetAllPackages() map[string]*inspector_dto.Package {
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return make(map[string]*inspector_dto.Package)
	}
	return ti.typeData.Packages
}

// FindRenderReturnType finds the return type of a component's Render
// function.
//
// It requires both the package path and the specific file path of the
// component to correctly resolve the function within the proper file-scoped
// context.
//
// Takes componentPackagePath (string) which specifies the package path of the
// component.
// Takes componentFilePath (string) which specifies the file path of the
// component.
//
// Returns goast.Expr which is the return type expression, or nil if the
// querier is uninitialised or the component cannot be found.
func (ti *TypeQuerier) FindRenderReturnType(componentPackagePath, componentFilePath string) goast.Expr {
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return nil
	}

	pkg, ok := ti.typeData.Packages[componentPackagePath]
	if !ok {
		return nil
	}
	localPackageName := pkg.Name

	return ti.FindFuncReturnType(
		localPackageName,
		"Render",
		componentPackagePath,
		componentFilePath,
	)
}

// GetImportsForFile returns a map of all imports for a specific file.
// The map contains alias to full package path mappings, plus identity mappings
// for convenience lookups.
//
// Takes importerPackagePath (string) which identifies the package containing the
// file.
// Takes importerFilePath (string) which specifies the file to get imports for.
//
// Returns map[string]string which maps import aliases and paths to their full
// package paths. Returns an empty map when the file is not found or state is
// uninitialised.
func (ti *TypeQuerier) GetImportsForFile(importerPackagePath, importerFilePath string) map[string]string {
	if ti.typeData == nil || ti.typeData.Packages == nil || importerFilePath == "" {
		return make(map[string]string)
	}
	pkg, ok := ti.typeData.Packages[importerPackagePath]
	if !ok || pkg.FileImports == nil {
		return make(map[string]string)
	}
	fileImports, found := pkg.FileImports[importerFilePath]
	if !found {
		return make(map[string]string)
	}

	importsCopy := make(map[string]string, len(fileImports)+2)

	for alias, path := range fileImports {
		importsCopy[alias] = path
		importsCopy[path] = path
	}

	if pkg.Path != "" {
		importsCopy[pkg.Path] = pkg.Path
		importsCopy[pkg.Name] = pkg.Path
	}

	return importsCopy
}

// FindPropsType searches the local script ASTs for a type Props struct.
//
// Takes filePath (string) which specifies which file's AST to search.
//
// Returns goast.Expr which is the Props type expression, or nil if not found.
func (ti *TypeQuerier) FindPropsType(filePath string) goast.Expr {
	file, ok := ti.localPackageFiles[filePath]
	if !ok {
		return nil
	}

	var propsType goast.Expr
	goast.Inspect(file, func(n goast.Node) bool {
		if ts, ok := n.(*goast.TypeSpec); ok && ts.Name.Name == "Props" {
			propsType = ts.Name
			return false
		}
		return true
	})
	return propsType
}

// GetAllFieldsAndMethods returns a sorted list of all exported field and
// method names for a given type.
//
// Takes baseType (goast.Expr) which is the type expression to analyse.
// Takes importerPackagePath (string) which is the package path for resolution.
// Takes importerFilePath (string) which provides file-scoped context to
// correctly resolve the base type.
//
// Returns []string which contains the sorted field and method names, or nil
// if the type cannot be resolved.
func (ti *TypeQuerier) GetAllFieldsAndMethods(baseType goast.Expr, importerPackagePath, importerFilePath string) []string {
	if baseType == nil {
		return nil
	}

	resolvedBaseType := ti.ResolveToUnderlyingAST(baseType, importerFilePath)
	namedType, _ := ti.ResolveExprToNamedType(resolvedBaseType, importerPackagePath, importerFilePath)
	if namedType == nil {
		return nil
	}

	candidates := make(map[string]struct{})
	for _, field := range namedType.Fields {
		candidates[field.Name] = struct{}{}
	}
	for _, method := range namedType.Methods {
		candidates[method.Name] = struct{}{}
	}

	if len(candidates) == 0 {
		return nil
	}

	result := make([]string, 0, len(candidates))
	for name := range candidates {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

// FindFileWithImportAlias searches through all files in a package to find which
// file contains the specified import alias that maps to the given canonical
// package path. Determines the correct file context for resolving aliased types.
//
// Takes packagePath (string) which specifies the package to search within.
// Takes alias (string) which is the import alias to find.
// Takes canonicalPath (string) which is the expected target package path.
//
// Returns string which is the file path containing the alias, or empty if
// not found.
func (ti *TypeQuerier) FindFileWithImportAlias(packagePath, alias, canonicalPath string) string {
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return ""
	}

	pkg, ok := ti.typeData.Packages[packagePath]
	if !ok || pkg.FileImports == nil {
		return ""
	}

	for filePath, fileImports := range pkg.FileImports {
		if targetPath, found := fileImports[alias]; found && targetPath == canonicalPath {
			return filePath
		}
	}

	return ""
}

// FindPackagePathForTypeDTO locates the package path that owns the provided
// Type DTO. The PackagePath is stored directly on the Type DTO for O(1)
// lookup.
//
// Takes target (*inspector_dto.Type) which is the type to look up.
//
// Returns string which is the package path, or empty if target is nil.
func (*TypeQuerier) FindPackagePathForTypeDTO(target *inspector_dto.Type) string {
	if target == nil {
		return ""
	}
	return target.PackagePath
}

// getNamedTypeByPackageAndName looks up a Type DTO by its package path and type
// name.
//
// Takes packagePath (string) which specifies the import path of the package.
// Takes typeName (string) which specifies the name of the type to find.
//
// Returns *inspector_dto.Type which is the matching type, or nil if not found.
func (ti *TypeQuerier) getNamedTypeByPackageAndName(packagePath, typeName string) *inspector_dto.Type {
	if ti == nil || ti.typeData == nil || ti.typeData.Packages == nil || packagePath == "" || typeName == "" {
		return nil
	}
	if pkg, ok := ti.typeData.Packages[packagePath]; ok && pkg != nil {
		if t, ok := pkg.NamedTypes[typeName]; ok {
			return t
		}
	}
	return nil
}

// Debug returns a formatted string slice detailing the current state of the
// TypeQuerier. It delegates the formatting of each section to helpers.
//
// Takes importerPackagePath (string) which specifies the package path to focus on.
// Takes importerFilePath (string) which specifies the file path to focus on.
//
// Returns []string which contains the formatted debug output lines.
func (ti *TypeQuerier) Debug(importerPackagePath, importerFilePath string) []string {
	var lines []string
	lines = append(lines, "--- TYPE INSPECTOR STATE ---", "")
	lines = ti.appendGlobalContext(lines)
	lines = ti.appendFocusedContext(lines, importerPackagePath, importerFilePath)
	lines = append(lines, "", "--- END INSPECTOR STATE ---")
	return lines
}

// appendGlobalContext adds details about all loaded packages to the debug
// output.
//
// Takes lines ([]string) which is the existing output to add to.
//
// Returns []string which is the updated output with package details added.
func (ti *TypeQuerier) appendGlobalContext(lines []string) []string {
	lines = append(lines, ">> GLOBAL CONTEXT: All Loaded Packages")
	if ti.typeData == nil || len(ti.typeData.Packages) == 0 {
		return append(lines, "  (None)")
	}

	pkgPaths := make([]string, 0, len(ti.typeData.Packages))
	for path := range ti.typeData.Packages {
		pkgPaths = append(pkgPaths, path)
	}
	slices.Sort(pkgPaths)

	for _, path := range pkgPaths {
		lines = append(lines, fmt.Sprintf("  - %s", path))
	}
	return lines
}

// appendFocusedContext adds detailed context for a given package and file.
//
// Takes lines ([]string) which is the existing context lines to append to.
// Takes importerPackagePath (string) which specifies the package path to focus on.
// Takes importerFilePath (string) which specifies the file path for scoped
// imports.
//
// Returns []string which contains the original lines with focused context
// appended.
func (ti *TypeQuerier) appendFocusedContext(lines []string, importerPackagePath, importerFilePath string) []string {
	if importerPackagePath == "" {
		return lines
	}

	lines = append(lines, "", fmt.Sprintf(">> FOCUSED CONTEXT FOR PACKAGE: %s", importerPackagePath))
	if importerFilePath != "" {
		lines = append(lines, fmt.Sprintf(">> FOCUSED CONTEXT FOR FILE:    %s", importerFilePath))
	}

	pkgData, ok := ti.typeData.Packages[importerPackagePath]
	if !ok {
		return append(lines, "  (Could not find cached package data for this path)")
	}

	lines = appendPackageInfo(lines, pkgData)
	lines = appendFileScopedImports(lines, pkgData, importerFilePath)
	lines = appendDeclarations(lines, pkgData)
	lines = appendLiveASTImports(ti.localPackageFiles, lines, importerFilePath)
	return lines
}

// PackagePathForFile returns the package import path for a given file path.
// This is the public entry point for callers who need to determine the correct
// package context after resolving through type aliases.
//
// Takes filePath (string) which specifies the file to look up.
//
// Returns string which is the package import path for the file.
func (ti *TypeQuerier) PackagePathForFile(filePath string) string {
	return ti.lookupPackagePathForFile(filePath)
}

// GetFilesForPackage returns all files that belong to the given package path.
// Use it to find a valid file context when you need to interpret type
// expressions from a specific package.
//
// Takes packagePath (string) which specifies the package import path to look up.
//
// Returns []string which contains the file paths belonging to the package, or
// nil if the package is not found.
func (ti *TypeQuerier) GetFilesForPackage(packagePath string) []string {
	if ti == nil || ti.typeData == nil || ti.typeData.Packages == nil {
		return nil
	}

	pkg, ok := ti.typeData.Packages[packagePath]
	if !ok || pkg == nil || pkg.FileImports == nil {
		return nil
	}

	files := make([]string, 0, len(pkg.FileImports))
	for filePath := range pkg.FileImports {
		files = append(files, filePath)
	}
	return files
}

// lookupPackagePathForFile returns the package import path that owns the given
// file path.
//
// It first attempts to find a matching package by consulting the pre-computed
// file-to-package map (O(1)). If not found, it falls back to deriving the path
// from the querier's BaseDir and ModuleName.
//
// Takes filePath (string) which is the absolute path to a Go source file.
//
// Returns string which is the package import path, or empty if not found.
func (ti *TypeQuerier) lookupPackagePathForFile(filePath string) string {
	if filePath == "" {
		return ""
	}

	if ti != nil && ti.typeData != nil && ti.typeData.FileToPackage != nil {
		if packagePath, ok := ti.typeData.FileToPackage[filePath]; ok {
			return packagePath
		}
	}

	if ti != nil && ti.Config.BaseDir != "" && ti.Config.ModuleName != "" {
		rel, err := filepath.Rel(ti.Config.BaseDir, filepath.Dir(filePath))
		if err == nil {
			rel = filepath.ToSlash(rel)
			rel = strings.Trim(rel, "/.")
			if rel == "" {
				return ti.Config.ModuleName
			}
			return ti.Config.ModuleName + "/" + rel
		}
	}
	return ""
}

// DebugDTO returns a structured dump of the entire TypeData artefact for
// inspection. It discovers all packages, sorts them for deterministic output,
// and calls DebugPackageDTO for each one.
//
// Returns map[string][]string which maps each canonical package path to its
// detailed string dump.
func (ti *TypeQuerier) DebugDTO() map[string][]string {
	if ti.typeData == nil || ti.typeData.Packages == nil {
		return map[string][]string{
			"error": {"TypeData or Packages map is nil."},
		}
	}

	allDumps := make(map[string][]string)

	pkgPaths := make([]string, 0, len(ti.typeData.Packages))
	for path := range ti.typeData.Packages {
		pkgPaths = append(pkgPaths, path)
	}
	slices.Sort(pkgPaths)

	for _, path := range pkgPaths {
		allDumps[path] = ti.DebugPackageDTO(path)
	}

	return allDumps
}

// DebugPackageDTO returns a detailed string representation of a package's
// data as stored in the TypeData cache.
//
// Acts as a debugging tool for verifying the output of the serialisation stage.
// It formats every piece of information, including types, fields, methods,
// functions, and file-scoped import maps.
//
// Takes packagePath (string) which specifies the import path of the package to
// inspect.
//
// Returns []string which contains formatted debug lines, or an error message
// if the package is not found.
func (ti *TypeQuerier) DebugPackageDTO(packagePath string) []string {
	var lines []string
	lines = append(lines, fmt.Sprintf("--- DEBUG DTO for package: %s ---", packagePath))

	pkg, ok := ti.typeData.Packages[packagePath]
	if !ok {
		lines = append(lines, "  ERROR: Package not found in TypeData.", "--- END DTO DEBUG ---")
		return lines
	}

	lines = append(lines,
		"[Package Info]",
		fmt.Sprintf("  - Name:    %s", pkg.Name),
		fmt.Sprintf("  - Path:    %s", pkg.Path),
	)
	if pkg.Version != "" {
		lines = append(lines, fmt.Sprintf("  - Version: %s", pkg.Version))
	}
	lines = append(lines, "", "[File-Scoped Imports]")
	lines = append(lines, formatFileImportsLines(pkg.FileImports)...)
	lines = append(lines, "", "[Named Types]")
	lines = append(lines, formatNamedTypesReadable(pkg)...)
	lines = append(lines, "", "[Package-Level Functions]")
	lines = append(lines, formatFuncsReadable(pkg.Funcs)...)
	lines = append(lines, "--- END DTO DEBUG ---")
	return lines
}

// DeconstructTypeExpr extracts the core parts from a type expression
// AST.
//
// It removes wrapper types such as pointers, arrays, channels, and
// generics to find the base type. If the expression is an unqualified
// identifier, pkgAlias will be empty.
//
// Takes expression (goast.Expr) which is the type expression to
// deconstruct.
//
// Returns typeName (string) which is the base type name.
// Returns pkgAlias (string) which is the package alias, or empty if
// local.
// Returns ok (bool) which indicates whether deconstruction succeeded.
func DeconstructTypeExpr(expression goast.Expr) (typeName, pkgAlias string, ok bool) {
	for {
		switch t := expression.(type) {
		case *goast.StarExpr:
			expression = t.X
		case *goast.ArrayType:
			expression = t.Elt
		case *goast.MapType:
			return "map", "", true
		case *goast.ChanType:
			expression = t.Value
		case *goast.Ident:
			return t.Name, "", true
		case *goast.SelectorExpr:
			if pkgIdent, isIdent := t.X.(*goast.Ident); isIdent {
				return t.Sel.Name, pkgIdent.Name, true
			}
			return "", "", false
		case *goast.IndexExpr:
			expression = t.X
		case *goast.IndexListExpr:
			expression = t.X
		case *goast.ParenExpr:
			expression = t.X

		case *goast.FuncType:
			return "function", "", true
		case *goast.InterfaceType:
			return "interface{}", "", true
		case *goast.StructType:
			return "struct", "", true

		default:
			return "", "", false
		}
	}
}

// ToExportedName converts a string to its exported Go form by making the first
// letter upper case (e.g., "name" becomes "Name").
//
// Takes s (string) which is the name to convert.
//
// Returns string which is the input with its first letter capitalised.
func ToExportedName(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

// appendPackageInfo adds the package name and version to the debug output.
//
// Takes lines ([]string) which holds the existing output lines.
// Takes pkgData (*inspector_dto.Package) which holds the package details.
//
// Returns []string which contains the updated lines with package information.
func appendPackageInfo(lines []string, pkgData *inspector_dto.Package) []string {
	lines = append(lines, fmt.Sprintf("  - Package Name: %s", pkgData.Name))
	if pkgData.Version != "" {
		lines = append(lines, fmt.Sprintf("  - Version: %s", pkgData.Version))
	}
	return lines
}

// appendFileScopedImports adds stored import data to the debug output.
//
// Takes lines ([]string) which is the current output to add to.
// Takes pkgData (*inspector_dto.Package) which holds the stored import data.
// Takes importerFilePath (string) which marks the current file in the output.
//
// Returns []string which is the output with formatted import details added.
func appendFileScopedImports(lines []string, pkgData *inspector_dto.Package, importerFilePath string) []string {
	lines = append(lines, "  - File-Scoped Imports (from cached data):")
	if len(pkgData.FileImports) == 0 {
		return append(lines, "    (None)")
	}

	filePaths := make([]string, 0, len(pkgData.FileImports))
	for path := range pkgData.FileImports {
		filePaths = append(filePaths, path)
	}
	slices.Sort(filePaths)

	for _, path := range filePaths {
		contextMarker := ""
		if path == importerFilePath {
			contextMarker = "  <-- CURRENT CONTEXT"
		}
		lines = append(lines, fmt.Sprintf("    --- File: %s%s ---", path, contextMarker))

		importMap := pkgData.FileImports[path]
		aliases := make([]string, 0, len(importMap))
		for alias := range importMap {
			aliases = append(aliases, alias)
		}
		slices.Sort(aliases)

		for _, alias := range aliases {
			lines = append(lines, fmt.Sprintf("      * %-20s -> %s", `"`+alias+`"`, importMap[alias]))
		}
	}
	return lines
}

// appendDeclarations adds cached type and function declarations to the debug
// output.
//
// Takes lines ([]string) which holds the existing debug output lines.
// Takes pkgData (*inspector_dto.Package) which provides the cached
// declarations.
//
// Returns []string which contains the lines with declarations appended.
func appendDeclarations(lines []string, pkgData *inspector_dto.Package) []string {
	lines = append(lines, "  - Declarations (from cached data):")
	var declsFound bool

	if len(pkgData.NamedTypes) > 0 {
		typeNames := make([]string, 0, len(pkgData.NamedTypes))
		for name := range pkgData.NamedTypes {
			typeNames = append(typeNames, name)
		}
		slices.Sort(typeNames)
		for _, name := range typeNames {
			lines = append(lines, fmt.Sprintf("    * type %s ...", name))
		}
		declsFound = true
	}

	if len(pkgData.Funcs) > 0 {
		functionNames := make([]string, 0, len(pkgData.Funcs))
		for name := range pkgData.Funcs {
			functionNames = append(functionNames, name)
		}
		slices.Sort(functionNames)
		for _, name := range functionNames {
			lines = append(lines, fmt.Sprintf("    * func %s(...)", name))
		}
		declsFound = true
	}

	if !declsFound {
		lines = append(lines, "    (None)")
	}
	return lines
}

// appendLiveASTImports adds import details from the current live AST to the
// debug output.
//
// Takes localPackageFiles (map[string]*goast.File) which maps file paths to
// their parsed AST data.
// Takes lines ([]string) which is the current debug output to add to.
// Takes importerFilePath (string) which names the file to get imports from.
//
// Returns []string which is the updated debug output with import details added.
func appendLiveASTImports(localPackageFiles map[string]*goast.File, lines []string, importerFilePath string) []string {
	if importerFilePath == "" {
		return lines
	}

	file, ok := localPackageFiles[importerFilePath]
	if !ok {
		return lines
	}

	lines = append(lines, "  - Live AST Imports (from current file):")
	if len(file.Imports) == 0 {
		return append(lines, "    (No imports in AST)")
	}

	for _, imp := range file.Imports {
		var alias string
		if imp.Name != nil {
			alias = imp.Name.Name
		} else {
			alias = "(default)"
		}
		lines = append(lines, fmt.Sprintf("    * import %-15s %s", `"`+alias+`"`, imp.Path.Value))
	}
	return lines
}
