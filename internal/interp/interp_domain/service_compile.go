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

package interp_domain

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/types"
	"maps"
	"path"
	"reflect"
	"slices"
	"strings"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// undefinedPrefix is the prefix that go/types uses for "undefined" error messages (e.g.
	// "undefined: content_domain.AnnotatedField").
	undefinedPrefix = "undefined: "

	// couldNotImportPrefix is the prefix that go/types uses when the Importer returns an
	// error for a requested import. It is a stable part of the go/types public error format;
	// a change in upstream Go silently disables enrichment of these messages until the
	// prefix is updated here.
	couldNotImportPrefix = "could not import "

	// notRegisteredMarker is the surface text of errPackageNotInRegistry as it appears
	// inside the Importer's wrapped error message. The marker stays in sync with the
	// fmt.Errorf call at symbol_registry.go:(*SymbolRegistry).Import; enrichTypeCheckError
	// falls back to the un-enriched error when the marker no longer matches.
	notRegisteredMarker = "not registered with interpreter"
)

// parseAllPackages parses and filters files for every package in a multi-package
// compilation.
//
// Takes modulePath (string) which is the module path prefix.
// Takes packages (map[string]map[string]string) which maps relative package paths to
// filename-to-source maps.
//
// Returns map[string]*parsedPackage which maps import paths to their parsed package data.
// Returns error when any package fails to parse.
func (s *Service) parseAllPackages(modulePath string, packages map[string]map[string]string) (map[string]*parsedPackage, error) {
	parsed := make(map[string]*parsedPackage, len(packages))
	for relPath, sources := range packages {
		importPath := modulePath
		if relPath != "" {
			importPath = modulePath + "/" + relPath
		}

		pkg, err := s.parseSinglePackage(importPath, relPath, sources)
		if err != nil {
			return nil, fmt.Errorf("parsing packages: %w", err)
		}
		if pkg != nil {
			parsed[importPath] = pkg
		}
	}
	return parsed, nil
}

// parseSinglePackage parses a single package's source files and filters by build
// constraints.
//
// Takes importPath (string) which is the fully qualified import path.
// Takes relPath (string) which is the relative path within the module.
// Takes sources (map[string]string) which maps filenames to source.
//
// Returns *parsedPackage which holds the filtered AST files, or nil when all files are
// excluded by build constraints.
// Returns error when any file fails to parse.
func (s *Service) parseSinglePackage(importPath, relPath string, sources map[string]string) (*parsedPackage, error) {
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	slices.Sort(names)

	allFiles := make([]*ast.File, 0, len(sources))
	for _, name := range names {
		file, err := parser.ParseFile(s.fileSet, name, sources[name], parser.ParseComments)
		if err != nil {
			return nil, fmt.Errorf(errChainMessageFmt, errParse, name, err)
		}
		allFiles = append(allFiles, file)
	}

	files := make([]*ast.File, 0, len(allFiles))
	for _, f := range allFiles {
		if s.shouldIncludeFile(f) {
			files = append(files, f)
		}
	}

	if len(files) == 0 {
		return nil, nil
	}

	return &parsedPackage{
		importPath:  importPath,
		relPath:     relPath,
		packageName: files[0].Name.Name,
		files:       files,
	}, nil
}

// compileSinglePackage type-checks and compiles a single package, sharing the provided
// root function and seeding with cross-package method entries from previously compiled
// packages.
//
// Takes pkg (*parsedPackage) which is the parsed package to compile.
// Takes rootFunction (*CompiledFunction) which is the shared root function.
// Takes crossPackageMethods (map[string]uint16) which holds method entries from
// previously compiled packages.
// Takes interpretedPaths (map[string]bool) which holds import paths of packages compiled
// from source in this batch, used to distinguish them from native packages when enriching
// type-check errors.
//
// Returns *packageCompileResult which holds the compiled output.
// Returns error when type-checking or compilation fails.
func (s *Service) compileSinglePackage(
	ctx context.Context,
	pkg *parsedPackage,
	rootFunction *CompiledFunction,
	crossPackageMethods map[string]uint16,
	interpretedPaths map[string]bool,
) (result *packageCompileResult, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			result = nil
			err = fmt.Errorf("%w: %s: %v", errCompilePanic, pkg.importPath, recovered)
		}
	}()
	info := s.newTypesInfo()
	conf := s.newTypesConfig()
	typesPackage, err := conf.Check(pkg.importPath, s.fileSet, pkg.files, info)
	if err != nil {
		enriched := s.enrichTypeCheckError(err, pkg.files, interpretedPaths)
		return nil, fmt.Errorf(errChainMessageFmt, errTypeCheck, pkg.importPath, enriched)
	}

	s.applyResourceLimits(rootFunction)
	c := s.newPackageCompiler(ctx, rootFunction, info, crossPackageMethods)

	c.registerPackageLevelVarsFromFiles(ctx, pkg.files)

	if err := c.twoPassCompileFuncs(ctx, pkg.files, pkg.importPath); err != nil {
		return nil, fmt.Errorf("compiling package %s: %w", pkg.importPath, err)
	}

	variableInitialisationFunction, err := c.compileVariableInitFunction(ctx, pkg.files)
	if err != nil {
		return nil, fmt.Errorf("compiling package %s variable init: %w", pkg.importPath, err)
	}

	initIndices := c.initFunctionIndices
	if variableInitialisationFunction != nil {
		varInitIndex := safeconv.IntToUint16(len(rootFunction.functions))
		rootFunction.functions = append(rootFunction.functions, variableInitialisationFunction)
		initIndices = append([]uint16{varInitIndex}, initIndices...)
	}

	exportedVars := collectExportedVars(ctx, c, typesPackage)

	return &packageCompileResult{
		funcTable:           c.funcTable,
		initFunctionIndices: initIndices,
		info:                info,
		typesPackage:        typesPackage,
		rootFunction:        rootFunction,
		exportedVars:        exportedVars,
	}, nil
}

// newPackageCompiler constructs the per-package compiler used by compileSinglePackage,
// threading the Service-level configuration and merging any cross-package method indices
// already discovered.
//
// Takes ctx (context.Context) which carries cancellation.
// Takes rootFunction (*CompiledFunction) which is the package root.
// Takes info (*types.Info) which carries the go/types data.
// Takes crossPackageMethods (map[string]uint16) which seeds the funcTable with
// already-resolved cross-package methods.
//
// Returns *compiler which is the configured package compiler.
func (s *Service) newPackageCompiler(ctx context.Context, rootFunction *CompiledFunction, info *types.Info, crossPackageMethods map[string]uint16) *compiler {
	c := &compiler{
		fileSet:            s.fileSet,
		info:               info,
		function:           rootFunction,
		scopes:             newScopeStack("<root>"),
		funcTable:          make(map[string]uint16),
		rootFunction:       rootFunction,
		symbols:            s.symbols,
		globalVariables:    make(map[string]globalVariableInfo),
		globals:            s.globals,
		debugEnabled:       s.config != nil && s.config.debugInfo,
		features:           s.features,
		maxLiteralElements: s.maxLiteralElements(),
		maxExpressionDepth: s.maxExpressionDepth(),
	}
	c.initDebugInfo(ctx, nil)
	maps.Copy(c.funcTable, crossPackageMethods)
	return c
}

// bridgePackageExports registers exported functions and constants from a compiled
// non-main package into the symbol registry so that subsequent packages can import them.
//
// Takes importPath (string) which is the package's import path.
// Takes result (*packageCompileResult) which holds the compiled package output.
func (s *Service) bridgePackageExports(importPath string, result *packageCompileResult) {
	exports := collectPackageExports(result)

	s.symbols.RegisterPackage(importPath, exports)
	s.symbols.RegisterTypesPackage(importPath, result.typesPackage)

	if len(result.exportedVars) > 0 {
		s.pendingVarBridges = append(s.pendingVarBridges, pendingVarBridge{
			importPath: importPath,
			vars:       result.exportedVars,
		})
	}

	publishExternalMethods(s.globals, result.rootFunction)
}

// finalisePendingVarBridges reads the post-init value of every recorded exported var from
// globalStore and merges them into the symbol registry under each package's import path.
//
// Called from Service.ExecuteInits after all init functions (including each package's
// synthesised <varinit>) have executed, so the values are guaranteed to reflect the
// user's initialisers rather than zero values.
func (s *Service) finalisePendingVarBridges() {
	if len(s.pendingVarBridges) == 0 {
		return
	}
	for _, pending := range s.pendingVarBridges {
		for _, varExport := range pending.vars {
			value, ok := s.globals.snapshotVarAs(varExport.slot, varExport.reflectType)
			if !ok {
				continue
			}
			if !varExport.storage.IsValid() || !varExport.storage.CanSet() {
				continue
			}
			varExport.storage.Set(value)
		}
	}
	s.pendingVarBridges = nil
}

// enrichTypeCheckError inspects a type-check error for "undefined: X.Y" patterns where X
// is a package alias that resolves to a native package registered in the symbol registry.
// When the package is registered but the symbol Y is not, the error is enriched with a
// hint suggesting re-extraction via "piko extract".
//
// Takes err (error) which is the original error from go/types.
// Takes files ([]*ast.File) which are the AST files being type-checked.
// Takes interpretedPaths (map[string]bool) which holds import paths of packages compiled
// from source in this batch - these are excluded from the hint because they are not
// managed by piko extract.
//
// Returns the original error unchanged when no enrichment applies, or a wrapped error
// with a diagnostic hint appended.
func (s *Service) enrichTypeCheckError(err error, files []*ast.File, interpretedPaths map[string]bool) error {
	if s.symbols == nil {
		return err
	}

	var typesErr types.Error
	if !errors.As(err, &typesErr) {
		return err
	}

	msg := typesErr.Msg

	if strings.HasPrefix(msg, couldNotImportPrefix) && strings.Contains(msg, notRegisteredMarker) {
		return enrichMissingPackageError(err, msg)
	}

	if !strings.HasPrefix(msg, undefinedPrefix) {
		return err
	}

	identifier := strings.TrimPrefix(msg, undefinedPrefix)
	packageAlias, symbolName, hasDot := strings.Cut(identifier, ".")
	if !hasDot {
		return err
	}

	aliases := buildImportAliasMap(files)
	fullPath, found := aliases[packageAlias]
	if !found {
		return err
	}

	if interpretedPaths[fullPath] {
		return err
	}

	if !s.symbols.HasPackage(fullPath) {
		return err
	}

	if _, registered := s.symbols.Lookup(fullPath, symbolName); registered {
		return err
	}

	return fmt.Errorf(
		"%w - symbol %q is not registered in the symbol registry for package %q; "+
			"you may need to re-run \"piko extract generate\" to update symbol exports",
		err, symbolName, fullPath,
	)
}

// collectExportedVars walks the package scope and records every exported package-level
// var's name, global-store slot, and Go type.
//
// The package-level vars themselves are not initialised at this point; that work happens
// during Service.ExecuteInits. The returned entries are an i-owe-you: at finalise time
// the Service reads each slot from globalStore and exposes the current value through the
// symbol registry. See Service.finalisePendingVarBridges.
//
// Takes ctx (context.Context) which is forwarded to typeToReflect for generic
// substitution / cancellation.
// Takes c (*compiler) which provides the globalVariables map and the types.Type to
// reflect.Type translator.
// Takes typesPackage (*types.Package) which supplies the scope walked for exported
// identifiers.
//
// Returns the export entries in deterministic name order.
func collectExportedVars(ctx context.Context, c *compiler, typesPackage *types.Package) []packageVarExport {
	if c == nil || typesPackage == nil || typesPackage.Scope() == nil {
		return nil
	}
	scope := typesPackage.Scope()
	names := scope.Names()
	slices.Sort(names)

	exports := make([]packageVarExport, 0, len(names))
	for _, name := range names {
		object := scope.Lookup(name)
		varObject, ok := object.(*types.Var)
		if !ok || !varObject.Exported() {
			continue
		}
		slot, ok := c.globalVariables[name]
		if !ok {
			continue
		}
		reflectType := c.typeToReflect(ctx, varObject.Type())
		if reflectType == nil {
			continue
		}

		storage := reflect.New(reflectType).Elem()
		exports = append(exports, packageVarExport{
			name:        name,
			slot:        slot,
			reflectType: reflectType,
			storage:     storage,
		})
	}
	return exports
}

// collectPackageExports gathers a package's exported symbols.
//
// Gathers exported function closures, constants, and var storages into a single
// reflect.Value-keyed map suitable for SymbolRegistry.RegisterPackage. Var storages are
// advertised even before their initialisers have run so a downstream package can capture
// the storage's reflect.Value during type-check; finalisePendingVarBridges rewrites them
// post-init.
//
// Takes result (*packageCompileResult) which holds the compiled package output.
//
// Returns map[string]reflect.Value which keys exports by symbol name.
func collectPackageExports(result *packageCompileResult) map[string]reflect.Value {
	exports := make(map[string]reflect.Value)
	for name, index := range result.funcTable {
		if !ast.IsExported(name) || strings.Contains(name, ".") {
			continue
		}
		closure := &runtimeClosure{function: result.rootFunction.functions[index], rootFunction: result.rootFunction}
		exports[name] = reflect.ValueOf(closure)
	}
	for _, typeObject := range result.info.Defs {
		if typeObject == nil || !typeObject.Exported() {
			continue
		}
		constantDecl, ok := typeObject.(*types.Const)
		if !ok {
			continue
		}
		if _, already := exports[constantDecl.Name()]; already {
			continue
		}
		exports[constantDecl.Name()] = constantToReflectValue(constantDecl.Val())
	}
	for _, varExport := range result.exportedVars {
		exports[varExport.name] = varExport.storage
	}
	return exports
}

// publishExternalMethods copies each "TypeName.MethodName" entry on rootFunction's
// methodTable into the Service-shared external method registry so a later CompileProgram
// for a different package can resolve cross-package adapter dispatch. No-op when there is
// nothing to publish or the globals slot is absent.
//
// Takes globals (*globalStore) which holds the cross-package method registry.
// Takes rootFunction (*CompiledFunction) which carries the local methodTable.
func publishExternalMethods(globals *globalStore, rootFunction *CompiledFunction) {
	if rootFunction == nil || globals == nil || len(rootFunction.methodTable) == 0 {
		return
	}
	for key, methodIndex := range rootFunction.methodTable {
		globals.registerExternalMethod(key, rootFunction, methodIndex)
	}
}

// buildDependencyGraph returns an import graph mapping each package to its local
// (source-defined) dependencies.
//
// Takes parsed (map[string]*parsedPackage) which maps import paths to their parsed
// package data.
//
// Returns map[string][]string which maps each import path to its local dependency import
// paths.
func buildDependencyGraph(parsed map[string]*parsedPackage) map[string][]string {
	deps := make(map[string][]string, len(parsed))
	for importPath, pkg := range parsed {
		var localDeps []string
		for _, file := range pkg.files {
			for _, imp := range file.Imports {
				impPath := strings.Trim(imp.Path.Value, `"`)
				if _, isLocal := parsed[impPath]; isLocal {
					localDeps = append(localDeps, impPath)
				}
			}
		}
		deps[importPath] = localDeps
	}
	return deps
}

// collectCrossPackageMethods copies method entries (names containing ".") from a
// package's funcTable into the cross-package map.
//
// Takes funcTable (map[string]uint16) which is the compiled package's function table.
// Takes crossPackageMethods (map[string]uint16) which is the shared cross-package method
// accumulator.
func collectCrossPackageMethods(funcTable, crossPackageMethods map[string]uint16) {
	for name, index := range funcTable {
		if strings.Contains(name, ".") {
			crossPackageMethods[name] = index
		}
	}
}

// topoSort performs a topological sort of the dependency graph, returning packages in
// dependency order (dependencies before dependents).
//
// Takes deps (map[string][]string) which maps each package import path to its dependency
// import paths.
//
// Returns []string which is the sorted import order.
// Returns error when a dependency cycle is detected; wraps errCyclicImport.
func topoSort(deps map[string][]string) ([]string, error) {
	const (
		unvisited = 0
		visiting  = 1
		visited   = 2

		maxImportGraphDepth = 1024
	)

	state := make(map[string]int, len(deps))
	var order []string

	var visit func(string, int) error
	visit = func(node string, depth int) error {
		if depth > maxImportGraphDepth {
			return fmt.Errorf("%w: import graph depth at %s exceeded %d", errCompileDepthLimit, node, maxImportGraphDepth)
		}
		switch state[node] {
		case visiting:
			return fmt.Errorf("%w: %s", errCyclicImport, node)
		case visited:
			return nil
		}

		state[node] = visiting
		for _, dependency := range deps[node] {
			if err := visit(dependency, depth+1); err != nil {
				return err
			}
		}
		state[node] = visited
		order = append(order, node)
		return nil
	}

	keys := make([]string, 0, len(deps))
	for k := range deps {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	for _, k := range keys {
		if err := visit(k, 0); err != nil {
			return nil, err
		}
	}

	return order, nil
}

// resolveEntrypoints determines the entrypoint function table from the main package,
// falling back to the last compiled package when no explicit main package exists.
//
// Takes mainFuncTable (map[string]uint16) which is the main package's function table, or
// nil if no main package was compiled.
// Takes lastFuncTable (map[string]uint16) which is the last compiled package's function
// table.
//
// Returns map[string]uint16 which maps entrypoint names to function indices, or nil when
// no entrypoints are available.
func resolveEntrypoints(mainFuncTable, lastFuncTable map[string]uint16) map[string]uint16 {
	if mainFuncTable == nil && lastFuncTable != nil {
		mainFuncTable = make(map[string]uint16, len(lastFuncTable))
		maps.Copy(mainFuncTable, lastFuncTable)
	}
	if mainFuncTable == nil {
		return nil
	}
	entrypoints := make(map[string]uint16, len(mainFuncTable))
	maps.Copy(entrypoints, mainFuncTable)
	return entrypoints
}

// constantToReflectValue converts a go/constant.Value to a reflect.Value using the
// constant's kind to select the appropriate Go type.
//
// Takes value (constant.Value) which is the constant to convert.
//
// Returns reflect.Value which holds the converted value, or reflect.ValueOf(nil) for
// unhandled constant kinds.
func constantToReflectValue(value constant.Value) reflect.Value {
	switch value.Kind() {
	case constant.Int:
		if v, ok := constant.Int64Val(value); ok {
			return reflect.ValueOf(int(v))
		}
		return reflect.ValueOf(0)
	case constant.Float:
		if v, ok := constant.Float64Val(value); ok {
			return reflect.ValueOf(v)
		}
		return reflect.ValueOf(0.0)
	case constant.String:
		return reflect.ValueOf(constant.StringVal(value))
	case constant.Bool:
		return reflect.ValueOf(constant.BoolVal(value))
	default:
		return reflect.ValueOf(nil)
	}
}

// buildImportAliasMap constructs a mapping from local package alias to full import path
// by inspecting the import declarations in the provided AST files.
//
// Takes files ([]*ast.File) which are the parsed source files to scan.
//
// Returns map[string]string mapping each local alias to its fully qualified import path.
// Blank imports ("_") and dot imports (".") are excluded.
func buildImportAliasMap(files []*ast.File) map[string]string {
	aliases := make(map[string]string)
	for _, file := range files {
		for _, imp := range file.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)

			if imp.Name != nil {
				name := imp.Name.Name
				if name == "_" || name == "." {
					continue
				}
				aliases[name] = importPath
				continue
			}

			aliases[path.Base(importPath)] = importPath
		}
	}
	return aliases
}

// enrichMissingPackageError appends an actionable hint to the original type-check error
// when the underlying cause is a package missing from the symbol registry. The hint
// directs the user at the discover and generate subcommands, which together resolve the
// vast majority of these failures.
//
// Takes err (error) which is the unwrapped types.Error returned by go/types.
// Takes msg (string) which is the types.Error.Msg, already matched against
// couldNotImportPrefix and notRegisteredMarker.
//
// Returns the original error wrapped with a trailing hint block.
func enrichMissingPackageError(err error, msg string) error {
	importPath := extractMissingPackagePath(msg)
	if importPath == "" {
		return errors.Join(errPackageNotInRegistry, fmt.Errorf(
			"%w\n\nhint: run \"piko extract discover\" to find missing package registrations, "+
				"then \"piko extract generate\" to update the symbol registry",
			err,
		))
	}
	return errors.Join(errPackageNotInRegistry, fmt.Errorf(
		"%w\n\nhint: add %q to piko-symbols.yaml and run \"piko extract generate\", "+
			"or run \"piko extract discover\" to find all missing registrations",
		err, importPath,
	))
}

// extractMissingPackagePath pulls the import path out of the "could not import X (package
// \"X\" not registered with interpreter)" message shape emitted by go/types when the
// Importer returns errPackageNotInRegistry. Returns empty string when the message does
// not match the expected shape.
//
// Takes msg (string) which is the types.Error.Msg text.
//
// Returns the extracted import path or an empty string.
func extractMissingPackagePath(msg string) string {
	if !strings.HasPrefix(msg, couldNotImportPrefix) {
		return ""
	}
	rest := msg[len(couldNotImportPrefix):]
	end := strings.IndexByte(rest, ' ')
	if end <= 0 {
		return ""
	}
	return rest[:end]
}
