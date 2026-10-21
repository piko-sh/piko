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
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/safedisk"
)

const (
	// packageStateWarningMessage is logged once for each project package that dev-i compiles
	// from source while the server binary also links it.
	packageStateWarningMessage = "dev-i compiles this project package from source, so its package variables " +
		"are a separate copy from the server binary's: values main sets before Run are not visible to pages. " +
		"Pass the value through the request, or register the package as native symbols with " +
		"piko extract --include-project"
)

// hostMainPackagePath returns the import path of the running binary's main package, or ""
// when the binary carries no build information.
//
// Returns string which is the main package's import path.
func hostMainPackagePath() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Path
}

// warnSeparatePackageState logs, once per package and process, that a project package
// dev-i compiles from source is also linked into the server binary and declares exported
// package variables. The two copies do not share those variables, so a value the host
// sets in its copy is invisible to pages.
//
// Takes sandbox (safedisk.Sandbox) which reads the host's sources from the project root.
// Takes userPackages (map[string]map[string]string) which maps the relative directories
// of the project packages compiled from source to their sources.
// Takes hostMainPath (string) which is the import path of the binary's main package.
func (o *InterpretedBuildOrchestrator) warnSeparatePackageState(
	ctx context.Context,
	sandbox safedisk.Sandbox,
	userPackages map[string]map[string]string,
	hostMainPath string,
) {
	_, l := logger_domain.From(ctx, log)
	modulePrefix := o.moduleName + "/"
	o.hostImportsOnce.Do(func() {
		if !strings.HasPrefix(hostMainPath, modulePrefix) {
			return
		}
		o.hostImports = localImportClosure(sandbox, strings.TrimPrefix(hostMainPath, modulePrefix), modulePrefix)
	})
	for _, relativeDirectory := range slices.Sorted(maps.Keys(userPackages)) {
		if !o.hostImports[relativeDirectory] {
			continue
		}
		variables := exportedPackageVariables(userPackages[relativeDirectory])
		if len(variables) == 0 {
			continue
		}
		if _, warned := o.packageStateWarned.LoadOrStore(relativeDirectory, true); warned {
			continue
		}
		l.Warn(packageStateWarningMessage,
			logger_domain.String(fieldPackagePath, relativeDirectory),
			logger_domain.Strings("variables", variables))
	}
}

// localImportClosure returns the relative directories of every project package that the
// package in mainDirectory imports, directly or through other project packages. Test
// files are skipped; every other Go file counts, whatever its build constraints.
//
// Takes sandbox (safedisk.Sandbox) which reads sources relative to the project root.
// Takes mainDirectory (string) which is the starting package's relative directory.
// Takes modulePrefix (string) which is the module path followed by a slash.
//
// Returns map[string]bool which holds the imported packages' relative directories.
func localImportClosure(sandbox safedisk.Sandbox, mainDirectory, modulePrefix string) map[string]bool {
	imported := make(map[string]bool)
	pending := []string{mainDirectory}
	for len(pending) > 0 {
		directory := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for _, relative := range localImportsOfDirectory(sandbox, directory, modulePrefix) {
			if !imported[relative] {
				imported[relative] = true
				pending = append(pending, relative)
			}
		}
	}
	return imported
}

// localImportsOfDirectory returns the relative directories of the project packages that
// the non-test Go files in directory import. An unreadable directory or file contributes
// nothing.
//
// Takes sandbox (safedisk.Sandbox) which reads sources relative to the project root.
// Takes directory (string) which is the package's relative directory.
// Takes modulePrefix (string) which is the module path followed by a slash.
//
// Returns []string which holds the imported relative directories, possibly repeated.
func localImportsOfDirectory(sandbox safedisk.Sandbox, directory, modulePrefix string) []string {
	entries, err := sandbox.ReadDir(directory)
	if err != nil {
		return nil
	}
	var imports []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, readErr := sandbox.ReadFile(filepath.Join(directory, name))
		if readErr != nil {
			continue
		}
		for _, importPath := range parseLocalImportPaths(string(content), modulePrefix) {
			imports = append(imports, strings.TrimPrefix(importPath, modulePrefix))
		}
	}
	return imports
}

// exportedPackageVariables returns the sorted names of the exported package-level
// variables that files declare. A file that does not parse contributes nothing.
//
// Takes files (map[string]string) which maps file names to Go source.
//
// Returns []string which holds the variable names, or nil when there are none.
func exportedPackageVariables(files map[string]string) []string {
	var names []string
	fileSet := token.NewFileSet()
	for name, source := range files {
		file, err := parser.ParseFile(fileSet, name, source, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		names = append(names, exportedVariablesOfFile(file)...)
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// exportedVariablesOfFile returns the names of the exported package-level variables that
// file declares, in declaration order.
//
// Takes file (*ast.File) which is the parsed file.
//
// Returns []string which holds the variable names.
func exportedVariablesOfFile(file *ast.File) []string {
	var names []string
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, spec := range general.Specs {
			if valueSpec, isValue := spec.(*ast.ValueSpec); isValue {
				names = append(names, exportedNames(valueSpec.Names)...)
			}
		}
	}
	return names
}

// exportedNames returns the exported names among idents, in order.
//
// Takes idents ([]*ast.Ident) which are the declared names.
//
// Returns []string which holds the exported names.
func exportedNames(idents []*ast.Ident) []string {
	var names []string
	for _, ident := range idents {
		if ident.IsExported() {
			names = append(names, ident.Name)
		}
	}
	return names
}
