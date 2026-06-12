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
	"fmt"
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"reflect"
	"runtime"
	"slices"
	"strings"
)

const (
	// errFmtEvaluatingFile is the format string for wrapping errors during file evaluation.
	errFmtEvaluatingFile = "evaluating file: %w"

	// errFmtCompilingFile is the format string for wrapping errors during file compilation.
	errFmtCompilingFile = "compiling file: %w"

	// minBuildSuffixParts is the threshold for considering a filename to have both GOOS and
	// GOARCH suffix components (e.g. "foo_linux_amd64").
	minBuildSuffixParts = 3
)

var (
	// statementForbiddenIdents names predeclared identifiers that cannot appear as a bare
	// expression statement when invoked in CallExpr form.
	//
	// Three categories share this constraint. Value-returning builtins listed in the Go
	// specification (append, cap, complex, imag, len, make, max, min, new, real) require
	// their result to be used. Predeclared type identifiers (int, int64, string, ...) used
	// in type-conversion form (e.g. `int(x)`) yield a value of the named type, which
	// go/types treats as "evaluated but not used" if dropped on the floor. The `error`,
	// `any`, and `comparable` interface types also serve as type-conversion targets when
	// written as `T(expr)`.
	//
	// A call to one of these as the trailing statement of the synthetic _eval_ function
	// still needs the `_ = call(...)` wrap to satisfy go/types' "value of type X is not
	// used" check; every other call (regular functions, methods, void builtins like
	// println/panic/ print/recover, function literals invoked inline) is a valid statement
	// and is left alone. The unsafe.* family uses SelectorExpr and is left to the existing
	// fallback path; misuse is rare and the resulting type-check error is clear.
	statementForbiddenIdents = map[string]struct{}{
		// Value-returning builtins (Go spec).
		"append":  {},
		"cap":     {},
		"complex": {},
		"imag":    {},
		"len":     {},
		"make":    {},
		"max":     {},
		"min":     {},
		"new":     {},
		"real":    {},
		// Predeclared numeric and integer types.
		"int":        {},
		"int8":       {},
		"int16":      {},
		"int32":      {},
		"int64":      {},
		"uint":       {},
		"uint8":      {},
		"uint16":     {},
		"uint32":     {},
		"uint64":     {},
		"uintptr":    {},
		"byte":       {},
		"rune":       {},
		"float32":    {},
		"float64":    {},
		"complex64":  {},
		"complex128": {},
		// Predeclared boolean, string, and interface types.
		"bool":       {},
		"string":     {},
		"error":      {},
		"any":        {},
		"comparable": {},
	}

	// knownGOOSTags enumerates the GOOS values Go recognises as filename-suffix build
	// constraints. Keep in sync with go/build/syslist.go; this list rarely changes between
	// Go releases.
	knownGOOSTags = map[string]bool{
		"aix":       true,
		"android":   true,
		"darwin":    true,
		"dragonfly": true,
		"freebsd":   true,
		"hurd":      true,
		"illumos":   true,
		"ios":       true,
		"js":        true,
		"linux":     true,
		"nacl":      true,
		"netbsd":    true,
		"openbsd":   true,
		"plan9":     true,
		"solaris":   true,
		"wasip1":    true,
		"windows":   true,
		"zos":       true,
	}

	// knownGOARCHTags enumerates the GOARCH values Go recognises as filename-suffix build
	// constraints. Keep in sync with go/build/syslist.go.
	knownGOARCHTags = map[string]bool{
		"386":         true,
		"amd64":       true,
		"amd64p32":    true,
		"arm":         true,
		"arm64":       true,
		"arm64be":     true,
		"armbe":       true,
		"loong64":     true,
		"mips":        true,
		"mips64":      true,
		"mips64le":    true,
		"mips64p32":   true,
		"mips64p32le": true,
		"mipsle":      true,
		"ppc":         true,
		"ppc64":       true,
		"ppc64le":     true,
		"riscv":       true,
		"riscv64":     true,
		"s390":        true,
		"s390x":       true,
		"sparc":       true,
		"sparc64":     true,
		"wasm":        true,
	}
)

// compileOrderedPackages iterates over topologically sorted packages, compiling each one
// and collecting init functions and func tables.
//
// Takes order ([]string) which is the topologically sorted list of package import paths.
// Takes parsed (map[string]*parsedPackage) which maps import paths to their parsed
// package data.
// Takes rootFunction (*CompiledFunction) which is the root compiled function for the
// program.
// Takes crossPackageMethods (map[string]uint16) which tracks method indices shared across
// packages.
//
// Returns the collected init function indices, the main package func table, the last
// compiled func table, and any compilation error.
func (s *Service) compileOrderedPackages(
	ctx context.Context,
	order []string,
	parsed map[string]*parsedPackage,
	rootFunction *CompiledFunction,
	crossPackageMethods map[string]uint16,
) (allInitFuncs []uint16, mainFuncTable map[string]uint16, lastFuncTable map[string]uint16, err error) {
	interpretedPaths := make(map[string]bool, len(parsed))
	for importPath := range parsed {
		interpretedPaths[importPath] = true
	}

	for _, importPath := range order {
		pkg := parsed[importPath]
		result, compileErr := s.compileSinglePackage(ctx, pkg, rootFunction, crossPackageMethods, interpretedPaths)
		if compileErr != nil {
			return allInitFuncs, mainFuncTable, lastFuncTable, compileErr
		}

		allInitFuncs = append(allInitFuncs, result.initFunctionIndices...)
		collectCrossPackageMethods(result.funcTable, crossPackageMethods)
		lastFuncTable = result.funcTable

		if pkg.packageName != "main" {
			s.bridgePackageExports(importPath, result)
		} else {
			publishExternalMethods(s.globals, result.rootFunction)
		}

		if pkg.packageName == "main" {
			mainFuncTable = make(map[string]uint16, len(result.funcTable))
			maps.Copy(mainFuncTable, result.funcTable)
		}
	}

	return allInitFuncs, mainFuncTable, lastFuncTable, nil
}

// parseAndFilterFiles parses source files in deterministic order and filters them by
// //go:build constraints.
//
// Takes sources (map[string]string) which maps filenames to source code strings.
//
// Returns []*ast.File which are the parsed and filtered AST files.
// Returns error when parsing fails or no files pass the filter.
func (s *Service) parseAndFilterFiles(sources map[string]string) ([]*ast.File, error) {
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
	for _, file := range allFiles {
		if s.shouldIncludeFile(file) {
			files = append(files, file)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("%w: all files excluded by build constraints", errParse)
	}
	return files, nil
}

// newFileSetCompiler creates a compiler for a file-set or eval compilation that needs
// global variable tracking.
//
// Takes ctx (context.Context) which is the compilation context.
// Takes rootFunction (*CompiledFunction) which is the root compiled function for the file
// set.
// Takes info (*types.Info) which holds the type-checking information.
//
// Returns *compiler configured with global variable tracking.
func (s *Service) newFileSetCompiler(ctx context.Context, rootFunction *CompiledFunction, info *types.Info) *compiler {
	s.applyResourceLimits(rootFunction)
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
	return c
}

// applyResourceLimits stamps the service-configured per-function caps (constant pool,
// specialisations, methods) onto rootFunction so that every CompiledFunction propagating
// from it observes the configured limits.
//
// Takes rootFunction (*CompiledFunction) which receives the cap values.
func (s *Service) applyResourceLimits(rootFunction *CompiledFunction) {
	if s.config == nil || rootFunction == nil {
		return
	}
	if s.config.maxConstantPoolSize > 0 {
		rootFunction.maxConstantPoolSize = s.config.maxConstantPoolSize
	}
	if s.config.maxSpecialisations > 0 {
		rootFunction.maxSpecialisations = s.config.maxSpecialisations
	}
	if s.config.maxMethods > 0 {
		rootFunction.maxMethods = s.config.maxMethods
	}
}

// maxExpressionDepth returns the configured expression-depth ceiling, or 0 to keep the
// package default in force.
//
// Returns int which is the configured ceiling, or 0 when unset.
func (s *Service) maxExpressionDepth() int {
	if s.config != nil {
		return s.config.maxExpressionDepth
	}
	return 0
}

// maxLiteralElements returns the configured max literal element count, or 0 if not set.
//
// Returns int which is the configured limit, or 0 for unlimited.
func (s *Service) maxLiteralElements() int {
	if s.config != nil {
		return s.config.maxLiteralElements
	}
	return 0
}

// executeInitFunc runs a single init function in its own VM, ensuring the arena is
// released on init function exit rather than when the caller returns.
//
// Takes rootFunction (*CompiledFunction) which provides the function table for
// cross-function calls.
// Takes initFunction (*CompiledFunction) which is the init function to execute.
//
// Returns error when the init function execution fails.
func (s *Service) executeInitFunc(ctx context.Context, rootFunction *CompiledFunction, initFunction *CompiledFunction) error {
	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.applyVMOutputs(vm)
	vm.functions = rootFunction.functions
	vm.rootFunction = rootFunction
	vm.ensureCallStack()
	defer vm.releaseArena()
	defer vm.finishWatcher()
	vm.pushFrame(initFunction)
	if _, err := vm.runGuarded(0); err != nil {
		return fmt.Errorf("init: %w", err)
	}
	vm.globals.materialiseStrings(vm.arena)
	return nil
}

// evalExpr evaluates a single expression.
//
// Takes code (string) which contains the Go expression source.
//
// Returns any which is the result of evaluating the expression.
// Returns error when parsing, type-checking, or execution fails.
func (s *Service) evalExpr(ctx context.Context, code string) (any, error) {
	wrapped := "package main\nvar _ = " + code
	if expressionAST, parseErr := parser.ParseExprFrom(s.fileSet, evalFileName, code, 0); parseErr == nil {
		if ident, ok := expressionAST.(*ast.Ident); ok && ident.Name == "nil" {
			wrapped = "package main\nvar _ any = " + code
		}
	}
	file, err := parser.ParseFile(s.fileSet, evalFileName, wrapped, 0)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errParse, err)
	}

	info := s.newTypesInfo()
	conf := s.newTypesConfig()

	_, err = conf.Check(mainPackageName, s.fileSet, []*ast.File{file}, info)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errTypeCheck, s.enrichTypeCheckError(err, []*ast.File{file}, nil))
	}

	declaration, ok := file.Decls[0].(*ast.GenDecl)
	if !ok {
		return nil, fmt.Errorf(errChainFmt, errCompilation, fmt.Errorf("expected GenDecl, got %T", file.Decls[0]))
	}
	spec, ok := declaration.Specs[0].(*ast.ValueSpec)
	if !ok {
		return nil, fmt.Errorf(errChainFmt, errCompilation, fmt.Errorf("expected ValueSpec, got %T", declaration.Specs[0]))
	}
	typedExpr := spec.Values[0]

	compiledFunction, err := compileEvalExpression(ctx, s.fileSet, info, typedExpr, s.symbols, s.features, s.maxLiteralElements())
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}

	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.applyVMOutputs(vm)
	result, execErr := vm.execute(compiledFunction)
	s.recordCost(vm)
	return result, execErr
}

// doEvalFile evaluates a parsed file containing statements.
//
// Takes file (*ast.File) which is the parsed AST file to evaluate.
//
// Returns any which is the result of evaluating the last expression.
// Returns error when type-checking, compilation, or execution fails.
func (s *Service) doEvalFile(ctx context.Context, file *ast.File) (any, error) {
	lastExpr, hasResult := s.rewriteLastExprStmt(file)

	info := s.newTypesInfo()
	conf := s.newTypesConfig()

	if _, err := conf.Check(mainPackageName, s.fileSet, []*ast.File{file}, info); err != nil {
		return nil, fmt.Errorf(errChainFmt, errTypeCheck, s.enrichTypeCheckError(err, []*ast.File{file}, nil))
	}

	if hasResult && !expressionYieldsValue(info, lastExpr) {
		hasResult = false
	}

	evalFunction := &CompiledFunction{name: "<eval>"}
	c := s.newFileSetCompiler(ctx, evalFunction, info)

	c.registerPackageLevelVarsFromDecls(ctx, file.Decls)

	if err := s.compileAndRunVarInits(ctx, c, []*ast.File{file}); err != nil {
		return nil, fmt.Errorf(errFmtEvaluatingFile, err)
	}

	if err := c.compileNonEvalFuncDecls(ctx, file.Decls); err != nil {
		return nil, fmt.Errorf(errFmtEvaluatingFile, err)
	}

	for _, initIndex := range c.initFunctionIndices {
		if err := s.executeInitFunc(ctx, evalFunction, evalFunction.functions[initIndex]); err != nil {
			return nil, fmt.Errorf(errFmtEvaluatingFile, err)
		}
	}

	result, evalError := c.compileAndRunEvalBody(ctx, s, file, info, lastExpr, hasResult)
	if evalError != nil {
		return nil, fmt.Errorf(errFmtEvaluatingFile, evalError)
	}
	return result, nil
}

// compileAndRunVarInits compiles and immediately executes package-level variable
// initialisers. No-op when no global variables are registered.
//
// Takes c (*compiler) which holds the registered global variables.
// Takes files ([]*ast.File) which are the parsed AST files with variable declarations.
//
// Returns error when compilation or execution of variable initialisers fails.
func (s *Service) compileAndRunVarInits(ctx context.Context, c *compiler, files []*ast.File) error {
	variableInitialisationFunction, err := c.compileVariableInitFunction(ctx, files)
	if err != nil {
		return fmt.Errorf("compiling variable initialisers: %w", err)
	}
	if variableInitialisationFunction == nil {
		return nil
	}

	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.applyVMOutputs(vm)
	vm.ensureCallStack()
	defer vm.releaseArena()
	defer vm.finishWatcher()
	vm.pushFrame(variableInitialisationFunction)
	if _, err := vm.runGuarded(0); err != nil {
		return fmt.Errorf(errVarinitFmt, err)
	}
	vm.globals.materialiseStrings(vm.arena)
	return nil
}

// compileNonEvalFuncDecls compiles all function declarations in the file except the
// synthetic _eval_ function.
//
// Takes declarations ([]ast.Decl) which are the AST declarations to compile.
//
// Returns error when compilation of any function declaration fails.
func (c *compiler) compileNonEvalFuncDecls(ctx context.Context, declarations []ast.Decl) error {
	for _, declaration := range declarations {
		functionDeclaration, ok := declaration.(*ast.FuncDecl)
		if !ok || functionDeclaration.Name.Name == evalFuncName {
			continue
		}
		if err := c.compileFuncDecl(ctx, functionDeclaration); err != nil {
			return fmt.Errorf(errChainFmt, errCompilation, err)
		}
	}
	return nil
}

// compileAndRunEvalBody finds the _eval_ function, compiles its body, and executes it.
//
// Takes s (*Service) which is the interpreter service for VM creation.
// Takes file (*ast.File) which contains the _eval_ function.
// Takes info (*types.Info) which holds type-checking information.
// Takes lastExpr (ast.Expr) which is the last expression for result extraction.
// Takes hasResult (bool) which is true when the last statement was an expression.
//
// Returns any which is the result, or nil when no _eval_ exists.
// Returns error when compilation or execution fails.
func (c *compiler) compileAndRunEvalBody(
	ctx context.Context,
	s *Service,
	file *ast.File,
	info *types.Info,
	lastExpr ast.Expr,
	hasResult bool,
) (any, error) {
	functionDeclaration := findEvalFunctionDeclaration(file)
	if functionDeclaration == nil {
		return nil, nil
	}

	c.scopes.pushScope()
	c.closureCapturedNames = collectClosureCapturedNamesAll(functionDeclaration.Body)
	c.typedSliceLocals = classifyTypedSliceLocals(c, functionDeclaration.Body)

	lastLocation, err := c.compileStmtList(ctx, functionDeclaration.Body.List)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}

	if hasResult {
		lastLocation = c.coerceEvalBoolResult(ctx, info, lastExpr, lastLocation)
		c.function.resultKinds = []registerKind{lastLocation.kind}
		c.emitMoveToRegisterZero(ctx, lastLocation)
	}

	if err := c.resourceError(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}
	c.function.numRegisters = c.scopes.peakRegisters()
	if err := c.function.optimise(ctx); err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}
	c.scopes.popScope()

	if err := runBytecodeInliner(ctx, c.function); err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}

	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.applyVMOutputs(vm)
	result, err := vm.execute(c.function)
	s.recordCost(vm)
	return result, err
}

// newTypesInfo creates a fresh types.Info for type checking.
//
// Returns *types.Info with all maps initialised.
func (*Service) newTypesInfo() *types.Info {
	return &types.Info{
		Types:      make(map[ast.Expr]types.TypeAndValue),
		Defs:       make(map[*ast.Ident]types.Object),
		Uses:       make(map[*ast.Ident]types.Object),
		Instances:  make(map[*ast.Ident]types.Instance),
		Selections: make(map[*ast.SelectorExpr]*types.Selection),
	}
}

// newTypesConfig creates a types.Config for type checking.
//
// Returns *types.Config configured with the service's symbol importer.
func (s *Service) newTypesConfig() *types.Config {
	conf := &types.Config{
		Sizes: types.SizesFor("gc", "amd64"),
	}
	if s.symbols != nil {
		conf.Importer = s.symbols
	}
	return conf
}

// evalMixed handles code that mixes declarations (func, var, type, const) with executable
// statements, separating declarations from statements and placing them at the appropriate
// scope levels.
//
// Takes code (string) which is the mixed Go source code to evaluate.
//
// Returns any which is the result of evaluating the statements.
// Returns error when parsing, type-checking, or execution fails.
func (s *Service) evalMixed(ctx context.Context, code string) (any, error) {
	file, err := s.parseMixedSource(code)
	if err != nil {
		return nil, fmt.Errorf("evaluating mixed source: %w", err)
	}
	if err := s.checkFileImports(nil, file); err != nil {
		return nil, fmt.Errorf("evaluating mixed source: %w", err)
	}
	result, evalError := s.doEvalFile(ctx, file)
	if evalError != nil {
		return nil, fmt.Errorf("evaluating mixed source: %w", evalError)
	}
	return result, nil
}

// parseMixedSource classifies mixed code into imports, declarations, and statements, then
// reconstructs and parses a valid Go source file.
//
// Takes code (string) which is the mixed Go source code to parse.
//
// Returns *ast.File which is the reconstructed and parsed AST file.
// Returns error when the reconstructed source fails to parse.
func (s *Service) parseMixedSource(code string) (*ast.File, error) {
	cl := classifyLines(strings.Split(code, newlineSep))
	source := buildMixedSource(cl)

	file, err := parser.ParseFile(s.fileSet, evalFileName, source, 0)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errParse, err)
	}
	return file, nil
}

// rewriteLastExprStmt rewrites the trailing eval statement for go/types.
//
// The default transformation wraps the trailing expression as `_ = expr`, silencing
// "evaluated but not used" for value-bearing expressions so the synthetic _eval_ function
// passes go/types. The predeclared identifier `nil` is untyped, so the rewrite emits `_ =
// any(nil)` instead, an explicit type conversion that gives the literal a concrete type
// while keeping the original ident node in the tree for downstream info.Types lookups.
// Function calls are valid statements regardless of their return type; wrapping a void
// call as `_ = call(...)` would fail with "(no value) used as value", so calls are left
// as bare ExprStmt nodes. The post-typecheck refinement in doEvalFile / compileFile
// downgrades hasResult to false for void calls so the runtime does not extract a
// non-existent register value.
//
// Takes file (*ast.File) which contains the _eval_ function.
//
// Returns ast.Expr which is the original expression for type lookup; nil when no rewrite
// was performed.
// Returns bool which is true when the last statement was an expression statement we
// recognised; the effective hasResult downstream may still flip to false after
// typechecking detects a void result.
func (*Service) rewriteLastExprStmt(file *ast.File) (ast.Expr, bool) {
	for _, declaration := range file.Decls {
		functionDeclaration, ok := declaration.(*ast.FuncDecl)
		if !ok || functionDeclaration.Name.Name != evalFuncName || functionDeclaration.Body == nil {
			continue
		}
		return rewriteEvalBodyTail(functionDeclaration.Body)
	}
	return nil, false
}

// compileExpression compiles a single expression.
//
// The default wrapping is `package main\nvar _ = <code>`; for the bare predeclared
// identifier `nil` the assignee type is set to `any` so the literal acquires a concrete
// type and the assignment type-checks. Without this, `var _ = nil` fails with "use of
// untyped nil in assignment to _ identifier".
//
// Void function calls as standalone expressions are not handled here (they would still
// fail "(no value) used as value" under `var _ any = ...`); Service.Eval falls back to
// evalMixed when compileExpression returns an error, and the fixed rewriteLastExprStmt
// handles them through that path.
//
// Takes code (string) which contains the Go expression source.
//
// Returns *CompiledFunction which is the compiled expression.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) compileExpression(ctx context.Context, code string) (*CompiledFunction, error) {
	wrapped := "package main\nvar _ = " + code
	if expressionAST, parseErr := parser.ParseExprFrom(s.fileSet, evalFileName, code, 0); parseErr == nil {
		if ident, ok := expressionAST.(*ast.Ident); ok && ident.Name == "nil" {
			wrapped = "package main\nvar _ any = " + code
		}
	}
	file, err := parser.ParseFile(s.fileSet, evalFileName, wrapped, 0)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errParse, err)
	}

	info := s.newTypesInfo()
	conf := s.newTypesConfig()

	_, err = conf.Check(mainPackageName, s.fileSet, []*ast.File{file}, info)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errTypeCheck, s.enrichTypeCheckError(err, []*ast.File{file}, nil))
	}

	declaration, ok := file.Decls[0].(*ast.GenDecl)
	if !ok {
		return nil, fmt.Errorf(errChainFmt, errCompilation, fmt.Errorf("expected GenDecl, got %T", file.Decls[0]))
	}
	spec, ok := declaration.Specs[0].(*ast.ValueSpec)
	if !ok {
		return nil, fmt.Errorf(errChainFmt, errCompilation, fmt.Errorf("expected ValueSpec, got %T", declaration.Specs[0]))
	}
	typedExpr := spec.Values[0]

	compiled, compileErr := compileEvalExpression(ctx, s.fileSet, info, typedExpr, s.symbols, s.features, s.maxLiteralElements())
	if compileErr != nil {
		return nil, fmt.Errorf("compiling expression: %w", compileErr)
	}
	return compiled, nil
}

// compileFile compiles a parsed file containing statements.
//
// Takes file (*ast.File) which is the parsed AST file to compile.
//
// Returns *CompiledFunction which is the compiled file.
// Returns error when type-checking or compilation fails.
func (s *Service) compileFile(ctx context.Context, file *ast.File) (*CompiledFunction, error) {
	lastExpr, hasResult := s.rewriteLastExprStmt(file)

	info := s.newTypesInfo()
	conf := s.newTypesConfig()

	if _, err := conf.Check(mainPackageName, s.fileSet, []*ast.File{file}, info); err != nil {
		return nil, fmt.Errorf(errChainFmt, errTypeCheck, s.enrichTypeCheckError(err, []*ast.File{file}, nil))
	}

	if hasResult && !expressionYieldsValue(info, lastExpr) {
		hasResult = false
	}

	evalFunction := &CompiledFunction{name: "<eval>"}
	c := s.newFileSetCompiler(ctx, evalFunction, info)

	c.registerPackageLevelVarsFromDecls(ctx, file.Decls)

	variableInitialisationFunction, err := c.compileVariableInitFunction(ctx, []*ast.File{file})
	if err != nil {
		return nil, fmt.Errorf(errFmtCompilingFile, err)
	}
	if variableInitialisationFunction != nil {
		if err := s.executeVarInitFunction(ctx, variableInitialisationFunction); err != nil {
			return nil, fmt.Errorf(errFmtCompilingFile, err)
		}
		evalFunction.variableInitFunction = variableInitialisationFunction
	}

	if err := c.compileNonEvalFuncDecls(ctx, file.Decls); err != nil {
		return nil, fmt.Errorf(errFmtCompilingFile, err)
	}

	compiled, compileErr := c.compileEvalFunction(ctx, file, info, lastExpr, hasResult, evalFunction)
	if compileErr != nil {
		return nil, fmt.Errorf(errFmtCompilingFile, compileErr)
	}
	return compiled, nil
}

// executeVarInitFunction runs a compiled variable initialiser function.
//
// Takes variableInitialisationFunction (*CompiledFunction) which is the compiled variable
// initialiser to execute.
//
// Returns error when execution of the initialiser function fails.
func (s *Service) executeVarInitFunction(ctx context.Context, variableInitialisationFunction *CompiledFunction) error {
	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.applyVMOutputs(vm)
	vm.ensureCallStack()
	defer vm.releaseArena()
	defer vm.finishWatcher()
	vm.pushFrame(variableInitialisationFunction)
	if _, err := vm.runGuarded(0); err != nil {
		return fmt.Errorf(errVarinitFmt, err)
	}
	vm.globals.materialiseStrings(vm.arena)
	return nil
}

// compileEvalFunction finds and compiles the _eval_ function body.
//
// Takes file (*ast.File) which contains the _eval_ function.
// Takes info (*types.Info) which holds type-checking information.
// Takes lastExpr (ast.Expr) which is the last expression for result extraction.
// Takes hasResult (bool) which is true when the last statement was an expression.
// Takes evalFunction (*CompiledFunction) which is the compiled function shell to
// populate.
//
// Returns *CompiledFunction which is the compiled eval function, or evalFunction when no
// _eval_ function exists.
// Returns error when compilation fails.
func (c *compiler) compileEvalFunction(ctx context.Context,
	file *ast.File,
	info *types.Info,
	lastExpr ast.Expr,
	hasResult bool,
	evalFunction *CompiledFunction,
) (*CompiledFunction, error) {
	functionDeclaration := findEvalFunctionDeclaration(file)
	if functionDeclaration == nil {
		return evalFunction, nil
	}

	c.scopes.pushScope()
	c.typedSliceLocals = classifyTypedSliceLocals(c, functionDeclaration.Body)

	lastLocation, err := c.compileStmtList(ctx, functionDeclaration.Body.List)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}

	if hasResult {
		lastLocation = c.coerceEvalBoolResult(ctx, info, lastExpr, lastLocation)
		c.function.resultKinds = []registerKind{lastLocation.kind}
		c.emitMoveToRegisterZero(ctx, lastLocation)
	}

	if err := c.resourceError(); err != nil {
		return nil, fmt.Errorf("compiling eval function: %w", err)
	}
	c.function.numRegisters = c.scopes.peakRegisters()
	if err := c.function.optimise(ctx); err != nil {
		return nil, fmt.Errorf("compiling eval function: %w", err)
	}
	c.scopes.popScope()
	return c.function, nil
}

// compileMixed compiles code mixing declarations with statements.
//
// Takes code (string) which is the mixed Go source code to compile.
//
// Returns *CompiledFunction which is the compiled mixed code.
// Returns error when parsing or compilation fails.
func (s *Service) compileMixed(ctx context.Context, code string) (*CompiledFunction, error) {
	file, err := s.parseMixedSource(code)
	if err != nil {
		return nil, fmt.Errorf("compiling mixed source: %w", err)
	}
	if err := s.checkFileImports(nil, file); err != nil {
		return nil, fmt.Errorf("compiling mixed source: %w", err)
	}
	compiled, compileErr := s.compileFile(ctx, file)
	if compileErr != nil {
		return nil, fmt.Errorf("compiling mixed source: %w", compileErr)
	}
	return compiled, nil
}

// shouldIncludeFile evaluates Go build constraints for a file.
//
// The result decides whether the file participates in the current compilation. Three
// sources contribute, evaluated in order: the modern `//go:build EXPR` directive
// (preferred form), the legacy `// +build EXPR` directive (still emitted by older
// packages such as github.com/google/uuid's node_js.go), and the filename's GOOS/GOARCH
// suffix convention (foo_linux.go, foo_amd64.go, foo_linux_amd64.go). A file is included
// if every constraint matches the active tag set; files with no constraints at all are
// always included.
//
// Takes file (*ast.File) which is the parsed Go source file.
//
// Returns bool which is true when the file participates in the build.
func (s *Service) shouldIncludeFile(file *ast.File) bool {
	matcher := s.buildTagMatcher()
	if !buildConstraintCommentsAccept(file, matcher) {
		return false
	}
	filename := s.fileSet.Position(file.Package).Filename
	if filename == "" {
		filename = file.Name.Name
	}
	return filenameMatchesBuildSuffix(filename, matcher)
}

// buildTagMatcher returns a predicate that reports whether a build
// tag is active. The default set includes the current GOOS, GOARCH,
// the Go version, the synthetic `interp` tag, and any user-provided
// tags from WithBuildTags.
//
// The `interp` tag lets module authors mark files that cannot run
// under a bytecode interpreter - typically anything that touches
// `unsafe.Pointer`, inline assembly, cgo, or other features piko
// does not implement. A file annotated with `// +build !interp`
// (or `//go:build !interp`) is skipped by piko, and the author
// can ship a paired `// +build interp` file with an equivalent
// pure-Go implementation.
//
// Returns func(string) bool which reports whether a given tag is active.
func (s *Service) buildTagMatcher() func(string) bool {
	tags := make(map[string]bool)
	tags[runtime.GOOS] = true
	tags[runtime.GOARCH] = true
	tags["interp"] = true

	version := runtime.Version()
	if strings.HasPrefix(version, "go") {
		tags[version] = true
	}

	if s.config != nil {
		for _, t := range s.config.buildTags {
			tags[t] = true
		}
	}

	return func(tag string) bool {
		return tags[tag]
	}
}

// applyEnvOverrides patches the symbol registry's "os" package so that Getenv, LookupEnv,
// Environ, Setenv, and Unsetenv operate on the configured environment map instead of the
// host process.
func (s *Service) applyEnvOverrides() {
	if s.config == nil || len(s.config.env) == 0 || s.config.envOnce == nil {
		return
	}
	s.config.envOnce.Do(s.installEnvOverrides)
}

// installEnvOverrides performs the registry patch for applyEnvOverrides. It runs under
// the config's sync.Once, so it executes exactly once across all pooled clones sharing
// the config and the underlying symbol registry.
func (s *Service) installEnvOverrides() {
	env := s.config.env

	existing, ok := s.symbols.PackageSymbols("os")
	if !ok {
		return
	}

	patched := make(map[string]reflect.Value, len(existing))
	maps.Copy(patched, existing)

	patched["Getenv"] = reflect.ValueOf(func(key string) string {
		if v, has := env[key]; has {
			return v
		}
		return ""
	})

	patched["LookupEnv"] = reflect.ValueOf(func(key string) (string, bool) {
		v, has := env[key]
		return v, has
	})

	patched["Environ"] = reflect.ValueOf(func() []string {
		result := make([]string, 0, len(env))
		for k, v := range env {
			result = append(result, k+"="+v)
		}
		slices.Sort(result)
		return result
	})

	patched["Setenv"] = reflect.ValueOf(func(key, value string) error {
		env[key] = value
		return nil
	})

	patched["Unsetenv"] = reflect.ValueOf(func(key string) error {
		delete(env, key)
		return nil
	})

	s.symbols.RegisterPackage("os", patched)
}

// rewriteEvalBodyTail inspects the last statement of a piko-eval function body and
// rewrites it so the eval result is observable via a synthetic `_ = expr` assignment when
// the statement is a value- returning expression. Returns the original expression and a
// bool indicating whether the rewrite succeeded.
//
// Takes body (*ast.BlockStmt) which is the eval function's body.
//
// Returns the trailing expression (or nil) and true when the body ended in an expression
// statement piko can observe.
func rewriteEvalBodyTail(body *ast.BlockStmt) (ast.Expr, bool) {
	statements := body.List
	if len(statements) == 0 {
		return nil, false
	}
	last, ok := statements[len(statements)-1].(*ast.ExprStmt)
	if !ok {
		return nil, false
	}
	if call, isCall := last.X.(*ast.CallExpr); isCall {
		if callRequiresResultBinding(call) {
			body.List[len(statements)-1] = makeBindToBlankAssign(last.X)
		}
		return last.X, true
	}
	if ident, isIdent := last.X.(*ast.Ident); isIdent && ident.Name == "nil" {
		body.List[len(statements)-1] = makeBindToBlankAssign(&ast.CallExpr{
			Fun:  ast.NewIdent("any"),
			Args: []ast.Expr{last.X},
		})
		return last.X, true
	}

	body.List[len(statements)-1] = makeBindToBlankAssign(last.X)
	return last.X, true
}

// makeBindToBlankAssign produces the AST for `_ = expr` assignment statement used by
// rewriteEvalBodyTail to observe a trailing expression's value during piko's eval
// transform.
//
// Takes expression (ast.Expr) which is the right-hand-side expression.
//
// Returns the synthetic assignment statement.
func makeBindToBlankAssign(expression ast.Expr) *ast.AssignStmt {
	return &ast.AssignStmt{
		Lhs: []ast.Expr{ast.NewIdent("_")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{expression},
	}
}

// callRequiresResultBinding reports whether a call needs `_ = call(...)` wrapping.
//
// The wrapping makes go/types accept the call as a top-level statement. The decision is
// purely syntactic and covers two patterns the Go spec forbids in statement context:
// calls to predeclared identifiers in statementForbiddenIdents (builtins like
// len/cap/make and named types like int/string, where Fun is an *ast.Ident whose name
// matches), and type conversions written with a composite-type literal at the callee
// position such as []byte(s), (*T)(p), map[K]V(m), or chan int(c), where Fun is one of
// ArrayType, MapType, ChanType, StructType, InterfaceType, FuncType, StarExpr, or
// ParenExpr. Regular function/method calls are valid statements and are left alone.
// Shadowed builtins (e.g. a local `len := func(...){...}`) are detected as forbidden
// under the Ident heuristic; the resulting rewrite still type-checks because the user
// function returns a value.
//
// Takes call (*ast.CallExpr) which is the trailing call expression.
//
// Returns bool, true when the call must be wrapped to typecheck.
func callRequiresResultBinding(call *ast.CallExpr) bool {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		_, ok := statementForbiddenIdents[fun.Name]
		return ok
	case *ast.ArrayType,
		*ast.MapType,
		*ast.ChanType,
		*ast.StructType,
		*ast.InterfaceType,
		*ast.FuncType,
		*ast.StarExpr,
		*ast.ParenExpr:

		return true
	default:
		return false
	}
}

// expressionYieldsValue reports whether the expression yields a value.
//
// Treats void function calls (info.Types[expr].Type is the zero-length tuple) and
// expressions with no recorded type as not yielding a value (the missing-type case is
// defensive: the caller already passed a non-nil lastExpr captured by
// rewriteLastExprStmt, so the entry should normally be present).
//
// Used by doEvalFile and compileFile to refine the initial hasResult flag returned by
// rewriteLastExprStmt: a CallExpr whose signature returns nothing must downgrade
// hasResult to false so the bytecode compiler does not coerce or emit a move from an
// undefined result location.
//
// Takes info (*types.Info) populated by a successful Check pass.
// Takes expression (ast.Expr) which is the trailing expression of the _eval_ function
// captured by rewriteLastExprStmt.
//
// Returns bool, true when the expression yields a value.
func expressionYieldsValue(info *types.Info, expression ast.Expr) bool {
	if expression == nil {
		return false
	}
	typeAndValue, ok := info.Types[expression]
	if !ok {
		return false
	}
	if typeAndValue.Type == nil {
		return false
	}
	if tuple, isTuple := typeAndValue.Type.(*types.Tuple); isTuple {
		return tuple.Len() > 0
	}
	return true
}

// buildConstraintCommentsAccept evaluates every //go:build (or +build) comment appearing
// before the package clause against matcher. Returns false as soon as any constraint
// rejects the configured build tag set.
//
// Takes file (*ast.File) which provides the comment groups to inspect.
// Takes matcher (func(string) bool) which reports whether a tag is active.
//
// Returns bool which is true when every constraint accepts the tag set.
func buildConstraintCommentsAccept(file *ast.File, matcher func(string) bool) bool {
	for _, cg := range file.Comments {
		if cg.Pos() >= file.Package {
			break
		}
		for _, c := range cg.List {
			if !constraint.IsGoBuild(c.Text) && !constraint.IsPlusBuild(c.Text) {
				continue
			}
			expression, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}
			if !expression.Eval(matcher) {
				return false
			}
		}
	}
	return true
}

// filenameMatchesBuildSuffix evaluates the GOOS/GOARCH suffix on a filename.
//
// The function decodes the implicit build constraint encoded in a Go file's basename.
// Names like foo.go always match; foo_linux.go matches linux only; foo_amd64.go matches
// amd64 only; foo_linux_amd64.go requires both linux and amd64. The `_test` suffix is
// stripped before constraint matching. A filename whose final underscore-separated token
// is not a known GOOS or GOARCH name is treated as unconstrained.
//
// Takes filename (string) which is the file's basename or full path.
// Takes matcher (func(string) bool) which reports whether a tag is active.
//
// Returns bool which is true when the encoded constraint accepts the active tag set.
func filenameMatchesBuildSuffix(filename string, matcher func(string) bool) bool {
	base := filename
	if idx := strings.LastIndexAny(base, "/\\"); idx >= 0 {
		base = base[idx+1:]
	}
	base = strings.TrimSuffix(base, ".go")
	base = strings.TrimSuffix(base, "_test")

	parts := strings.Split(base, "_")
	if len(parts) < 2 {
		return true
	}
	last := parts[len(parts)-1]
	var prev string
	if len(parts) >= minBuildSuffixParts {
		prev = parts[len(parts)-2]
	}

	if prev != "" && knownGOOSTags[prev] && knownGOARCHTags[last] {
		return matcher(prev) && matcher(last)
	}
	if knownGOOSTags[last] || knownGOARCHTags[last] {
		return matcher(last)
	}
	return true
}
