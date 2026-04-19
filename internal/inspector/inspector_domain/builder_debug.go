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

// This file provides debug utilities for the TypeBuilder, allowing inspection
// of the final, serialised TypeData artefact. It sanitises file paths and
// filters output to keep debug information focused and portable, which is
// critical for creating stable regression tests.

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"piko.sh/piko/internal/json"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// DumpFormat specifies the output format for type data dumps.
type DumpFormat int

const (
	// DumpFormatReadable produces a human-friendly, sorted text output, ideal for
	// golden file diffs.
	DumpFormatReadable DumpFormat = iota

	// DumpFormatJSON produces a prettified JSON representation of the DTO, ideal
	// for programmatic assertions.
	DumpFormatJSON
)

const (
	// pathSeparator is the leading slash stripped when cleaning file paths.
	pathSeparator = "/"

	// modCachePlaceholder replaces the machine-specific GOMODCACHE path in
	// sanitised output, making golden files portable across environments.
	modCachePlaceholder = "$GOMODCACHE"

	// gorootPlaceholder replaces the machine-specific GOROOT path in sanitised
	// output, making golden files portable across Go installations.
	gorootPlaceholder = "$GOROOT"
)

// DumpOptions holds settings for changing how DumpTypeData writes its output.
type DumpOptions struct {
	// SanitisePathPrefix is the path prefix to remove from type paths in output.
	SanitisePathPrefix string

	// SanitiseModCachePrefix is the GOMODCACHE path prefix to replace with
	// $GOMODCACHE in output. This makes golden files portable across machines
	// with different module cache locations.
	SanitiseModCachePrefix string

	// SanitiseGorootPrefix is the GOROOT path prefix to replace with $GOROOT
	// in output. This makes golden files portable across Go installations
	// where the standard library may be at different filesystem locations.
	SanitiseGorootPrefix string

	// FilterPackagePrefixes lists package path prefixes to include in the output.
	FilterPackagePrefixes []string

	// Format specifies the output format for the dump; defaults to DumpFormatJSON.
	Format DumpFormat
}

// DumpTypeData generates a string representation of the built TypeData
// artefact.
//
// Should only be called after a successful Build() operation.
// It provides multiple formats and options for debugging and testing.
//
// Takes opts (DumpOptions) which specifies the output format and filtering
// options.
//
// Returns string which contains the formatted TypeData representation.
// Returns error when the builder has not been successfully run or
// serialisation fails.
//
// Safe for concurrent use; holds a read lock during execution.
func (m *TypeBuilder) DumpTypeData(opts DumpOptions) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentCacheKey == "" {
		return "", errors.New("cannot dump TypeData: builder has not been successfully run")
	}
	typeData, exists := m.typeDataByKey[m.currentCacheKey]
	if !exists || typeData == nil {
		return "", errors.New("cannot dump TypeData: builder has not been successfully run")
	}

	originalJSON, err := json.Marshal(typeData)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary copy of TypeData: %w", err)
	}
	var dtoCopy inspector_dto.TypeData
	if err := json.Unmarshal(originalJSON, &dtoCopy); err != nil {
		return "", fmt.Errorf("failed to create temporary copy of TypeData: %w", err)
	}

	filteredDTO := filterDTO(&dtoCopy, opts.FilterPackagePrefixes)

	if opts.SanitisePathPrefix != "" {
		sanitiseDTO(filteredDTO, opts.SanitisePathPrefix, opts.SanitiseModCachePrefix, opts.SanitiseGorootPrefix)
	}

	switch opts.Format {
	case DumpFormatJSON:
		bytes, err := json.ConfigStd.MarshalIndent(filteredDTO, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal final DTO to JSON: %w", err)
		}
		return string(bytes), nil
	case DumpFormatReadable:
		return dumpReadable(filteredDTO), nil
	default:
		return "", errors.New("unknown dump format requested")
	}
}

// filterDTO returns a new TypeData with only the packages that match the given
// prefixes.
//
// Takes td (*inspector_dto.TypeData) which provides the source type data.
// Takes prefixes ([]string) which lists the package path prefixes to match.
//
// Returns *inspector_dto.TypeData which holds only the matching packages.
func filterDTO(td *inspector_dto.TypeData, prefixes []string) *inspector_dto.TypeData {
	if len(prefixes) == 0 {
		return td
	}

	filtered := &inspector_dto.TypeData{
		Packages:      make(map[string]*inspector_dto.Package),
		FileToPackage: make(map[string]string),
	}

	for packagePath, pkg := range td.Packages {
		include := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(packagePath, prefix) {
				include = true
				break
			}
		}
		if include {
			filtered.Packages[packagePath] = pkg
			for filePath := range pkg.FileImports {
				filtered.FileToPackage[filePath] = packagePath
			}
		}
	}
	return filtered
}

// sanitisePath removes a prefix from a file path and converts it to forward
// slashes. It replaces GOROOT and module cache paths with $GOROOT and
// $GOMODCACHE placeholders respectively.
//
// Takes path (string) which is the file path to sanitise.
// Takes prefix (string) which is the primary prefix to strip from the path.
// Takes modCachePrefix (string) which is the GOMODCACHE prefix to replace.
// Takes gorootPrefix (string) which is the GOROOT prefix to replace.
//
// Returns string which is the sanitised path with forward slashes.
func sanitisePath(path, prefix, modCachePrefix, gorootPrefix string) string {
	if path == "" {
		return ""
	}
	if after, ok := strings.CutPrefix(path, prefix); ok {
		after = strings.TrimPrefix(after, pathSeparator)
		return filepath.ToSlash(after)
	}
	if gorootPrefix != "" {
		if after, ok := strings.CutPrefix(path, gorootPrefix); ok {
			after = strings.TrimPrefix(after, pathSeparator)
			return gorootPlaceholder + "/" + filepath.ToSlash(after)
		}
	}
	if modCachePrefix != "" {
		if after, ok := strings.CutPrefix(path, modCachePrefix); ok {
			after = strings.TrimPrefix(after, pathSeparator)
			return modCachePlaceholder + "/" + filepath.ToSlash(after)
		}
	}
	return filepath.ToSlash(path)
}

// sanitiseDTO replaces absolute file paths with relative paths in the given
// TypeData structure. Changes are made in place.
//
// Takes td (*inspector_dto.TypeData) which is the type data to update.
// Takes prefix (string) which is the path prefix to remove from file paths.
// Takes modCachePrefix (string) which is the GOMODCACHE prefix to replace.
// Takes gorootPrefix (string) which is the GOROOT prefix to replace.
func sanitiseDTO(td *inspector_dto.TypeData, prefix, modCachePrefix, gorootPrefix string) {
	for _, pkg := range td.Packages {
		sanitisePackageDTO(pkg, prefix, modCachePrefix, gorootPrefix)
	}

	sanitisedFTP := make(map[string]string, len(td.FileToPackage))
	for path, pkg := range td.FileToPackage {
		sanitisedFTP[sanitisePath(path, prefix, modCachePrefix, gorootPrefix)] = pkg
	}
	td.FileToPackage = sanitisedFTP
}

// sanitisePackageDTO removes a path prefix from all file paths in a package.
//
// Takes pkg (*inspector_dto.Package) which is the package to update in place.
// Takes prefix (string) which is the path prefix to remove from all file paths.
// Takes modCachePrefix (string) which is the GOMODCACHE prefix to replace.
// Takes gorootPrefix (string) which is the GOROOT prefix to replace.
func sanitisePackageDTO(pkg *inspector_dto.Package, prefix, modCachePrefix, gorootPrefix string) {
	sanitisedFileImports := make(map[string]map[string]string, len(pkg.FileImports))
	for path, imports := range pkg.FileImports {
		sanitisedFileImports[sanitisePath(path, prefix, modCachePrefix, gorootPrefix)] = imports
	}
	pkg.FileImports = sanitisedFileImports

	for _, typ := range pkg.NamedTypes {
		typ.DefinedInFilePath = sanitisePath(typ.DefinedInFilePath, prefix, modCachePrefix, gorootPrefix)
		for _, field := range typ.Fields {
			field.DefinitionFilePath = sanitisePath(field.DefinitionFilePath, prefix, modCachePrefix, gorootPrefix)
		}
		for _, method := range typ.Methods {
			method.DefinitionFilePath = sanitisePath(method.DefinitionFilePath, prefix, modCachePrefix, gorootPrefix)
		}
	}

	for _, inspectedFunction := range pkg.Funcs {
		inspectedFunction.DefinitionFilePath = sanitisePath(inspectedFunction.DefinitionFilePath, prefix, modCachePrefix, gorootPrefix)
	}

	for _, v := range pkg.Variables {
		v.DefinedInFilePath = sanitisePath(v.DefinedInFilePath, prefix, modCachePrefix, gorootPrefix)
	}
}

// dumpReadable creates a sorted, readable text dump of the full TypeData
// structure. It sorts all collections before output to ensure consistent
// results.
//
// Takes td (*inspector_dto.TypeData) which is the type data to dump.
//
// Returns string which is the formatted text, or a placeholder message when td
// is nil or empty.
func dumpReadable(td *inspector_dto.TypeData) string {
	var allLines []string

	if td == nil || len(td.Packages) == 0 {
		return "--- EMPTY OR FILTERED TYPEDATA ---"
	}

	pkgPaths := make([]string, 0, len(td.Packages))
	for path := range td.Packages {
		pkgPaths = append(pkgPaths, path)
	}
	slices.Sort(pkgPaths)

	for i, path := range pkgPaths {
		if i > 0 {
			allLines = append(allLines, "")
		}
		allLines = append(allLines, dumpPackageReadable(td.Packages[path])...)
	}

	return strings.Join(allLines, "\n")
}

// formatFileImportsLines formats file-scoped imports into readable lines.
// This shared helper is used by both dumpPackageReadable and DebugPackageDTO.
//
// Takes fileImports (map[string]map[string]string) which maps file paths to
// their import alias-to-path pairs.
//
// Returns []string which contains the formatted lines ready for display.
func formatFileImportsLines(fileImports map[string]map[string]string) []string {
	lines := make([]string, 0, len(fileImports)*3)
	if len(fileImports) == 0 {
		lines = append(lines, "  (none)")
		return lines
	}

	filePaths := make([]string, 0, len(fileImports))
	for path := range fileImports {
		filePaths = append(filePaths, path)
	}
	slices.Sort(filePaths)

	for _, filePath := range filePaths {
		lines = append(lines, fmt.Sprintf("  > File: %s", filePath))
		importMap := fileImports[filePath]
		if len(importMap) == 0 {
			lines = append(lines, "    (no imports in this file)")
			continue
		}
		aliases := make([]string, 0, len(importMap))
		for alias := range importMap {
			aliases = append(aliases, alias)
		}
		slices.Sort(aliases)
		for _, alias := range aliases {
			lines = append(lines, fmt.Sprintf("    - import %-20s -> %s", `"`+alias+`"`, importMap[alias]))
		}
	}
	return lines
}

// dumpPackageReadable formats a package DTO as readable text lines.
//
// Takes pkg (*inspector_dto.Package) which is the package to format.
//
// Returns []string which contains formatted lines showing the package name,
// version, file imports, named types, and package-level functions.
func dumpPackageReadable(pkg *inspector_dto.Package) []string {
	var lines []string
	lines = append(lines,
		fmt.Sprintf("--- PACKAGE: %s ---", pkg.Path),
		"[Package Info]",
		fmt.Sprintf("  - Name:    %s", pkg.Name),
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

	return lines
}

// formatNamedTypesReadable formats all named types in a package for readable
// output.
//
// Takes pkg (*inspector_dto.Package) which provides the package containing
// named types to format.
//
// Returns []string which contains the formatted type information, with each
// type separated by blank lines.
func formatNamedTypesReadable(pkg *inspector_dto.Package) []string {
	if len(pkg.NamedTypes) == 0 {
		return []string{"  (none)"}
	}

	typeNames := make([]string, 0, len(pkg.NamedTypes))
	for name := range pkg.NamedTypes {
		typeNames = append(typeNames, name)
	}
	slices.Sort(typeNames)

	var lines []string
	for i, name := range typeNames {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, formatSingleTypeReadable(pkg.NamedTypes[name], pkg.Path)...)
	}
	return lines
}

// formatSingleTypeReadable formats a single type definition for readable
// output.
//
// Takes t (*inspector_dto.Type) which provides the type information to format.
// Takes packagePath (string) which specifies the package path for method
// formatting.
//
// Returns []string which contains the formatted lines describing the type.
func formatSingleTypeReadable(t *inspector_dto.Type, packagePath string) []string {
	typeKind := "type"
	if t.IsAlias {
		typeKind = "type alias"
	}
	lines := []string{
		fmt.Sprintf("  > %s %s", typeKind, t.Name),
		fmt.Sprintf("    - Defined In:      %s", t.DefinedInFilePath),
		fmt.Sprintf("    - TypeString:      %s", t.TypeString),
		fmt.Sprintf("    - Underlying:      %s", t.UnderlyingTypeString),
		fmt.Sprintf("    - Stringability:   %v", t.Stringability),
	}

	if len(t.TypeParams) > 0 {
		lines = append(lines, fmt.Sprintf("    - Type Params:     [%s]", strings.Join(t.TypeParams, ", ")))
	}

	lines = append(lines, formatFieldsReadable(t.Fields)...)
	lines = append(lines, formatMethodsReadable(t.Methods, t.Name, packagePath)...)

	return lines
}

// formatFieldsReadable formats type fields into lines for display.
//
// Takes fields ([]*inspector_dto.Field) which contains the field data to
// format.
//
// Returns []string which contains the formatted lines ready for display.
func formatFieldsReadable(fields []*inspector_dto.Field) []string {
	if len(fields) == 0 {
		return nil
	}

	lines := []string{"    - Fields:"}
	for _, f := range fields {
		embeddedString := ""
		if f.IsEmbedded {
			embeddedString = " (embedded)"
		}
		lines = append(lines, fmt.Sprintf("      - %-20s %s%s", f.Name, f.TypeString, embeddedString))
		if f.PackagePath != "" {
			lines = append(lines, fmt.Sprintf("        (Canonical Package: %s)", f.PackagePath))
		}
		if f.RawTag != "" {
			lines = append(lines, fmt.Sprintf("        (Tag: `%s`)", f.RawTag))
		}
	}
	return lines
}

// formatMethodsReadable formats a type's methods for readable output.
//
// Takes methods ([]*inspector_dto.Method) which holds the methods to format.
// Takes typeName (string) which is the name of the type that owns the methods.
// Takes packagePath (string) which is the package path for finding promoted
// methods.
//
// Returns []string which holds formatted lines ready for display, or nil if
// methods is empty.
func formatMethodsReadable(methods []*inspector_dto.Method, typeName, packagePath string) []string {
	if len(methods) == 0 {
		return nil
	}

	methodNames := make([]string, 0, len(methods))
	methodMap := make(map[string]*inspector_dto.Method, len(methods))
	for i := range methods {
		m := methods[i]
		methodNames = append(methodNames, m.Name)
		methodMap[m.Name] = m
	}
	slices.Sort(methodNames)

	lines := []string{"    - Methods:"}
	for _, methodName := range methodNames {
		m := methodMap[methodName]
		receiverString := "(T)"
		if m.IsPointerReceiver {
			receiverString = "(*T)"
		}
		lines = append(lines, fmt.Sprintf("      - func %s %-15s %s", receiverString, m.Name, m.Signature.ToSignatureString()))
		if m.DeclaringTypeName != "" && (m.DeclaringTypeName != typeName || m.DeclaringPackagePath != packagePath) {
			lines = append(lines, fmt.Sprintf("        (Promoted from: %s.%s)", m.DeclaringPackagePath, m.DeclaringTypeName))
		}
	}
	return lines
}

// formatFuncsReadable formats package-level functions for readable output.
//
// Takes funcs (map[string]*inspector_dto.Function) which contains the
// functions to format, keyed by name.
//
// Returns []string which contains the formatted function lines sorted by
// name, or a single "(none)" entry if the map is empty.
func formatFuncsReadable(funcs map[string]*inspector_dto.Function) []string {
	if len(funcs) == 0 {
		return []string{"  (none)"}
	}

	functionNames := make([]string, 0, len(funcs))
	for name := range funcs {
		functionNames = append(functionNames, name)
	}
	slices.Sort(functionNames)

	lines := make([]string, 0, len(functionNames))
	for _, name := range functionNames {
		inspectedFunction := funcs[name]
		lines = append(lines, fmt.Sprintf("  - func %-20s %s", name, inspectedFunction.Signature.ToSignatureString()))
	}
	return lines
}
