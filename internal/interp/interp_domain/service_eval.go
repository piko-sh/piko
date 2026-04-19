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
)

// compileOrderedPackages iterates over topologically sorted packages,
// compiling each one and collecting init functions and func tables.
//
// Takes order ([]string) which is the topologically sorted list of
// package import paths.
// Takes parsed (map[string]*parsedPackage) which maps import paths
// to their parsed package data.
// Takes rootFunction (*CompiledFunction) which is the root compiled
// function for the program.
// Takes crossPackageMethods (map[string]uint16) which tracks method
// indices shared across packages.
//
// Returns the collected init function indices, the main package func
// table, the last compiled func table, and any compilation error.
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

		if pkg.relPath != "" {
			s.bridgePackageExports(importPath, result)
		}

		if pkg.relPath == "" {
			mainFuncTable = make(map[string]uint16, len(result.funcTable))
			maps.Copy(mainFuncTable, result.funcTable)
		}
	}

	return allInitFuncs, mainFuncTable, lastFuncTable, nil
}

// parseAndFilterFiles parses source files in deterministic order and
// filters them by //go:build constraints.
//
// Takes sources (map[string]string) which maps filenames to source
// code strings.
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

// newFileSetCompiler creates a compiler for a file-set or eval
// compilation that needs global variable tracking.
//
// Takes ctx (context.Context) which is the compilation context.
// Takes rootFunction (*CompiledFunction) which is the root compiled
// function for the file set.
// Takes info (*types.Info) which holds the type-checking
// information.
//
// Returns *compiler configured with global variable tracking.
func (s *Service) newFileSetCompiler(ctx context.Context, rootFunction *CompiledFunction, info *types.Info) *compiler {
	c := &compiler{
		fileSet:            s.fileSet,
		info:               info,
		function:           rootFunction,
		scopes:             newScopeStack("<root>"),
		funcTable:          make(map[string]uint16),
		rootFunction:       rootFunction,
		symbols:            s.symbols,
		globalVars:         make(map[string]globalVariableInfo),
		globals:            s.globals,
		debugEnabled:       s.config != nil && s.config.debugInfo,
		features:           s.features,
		maxLiteralElements: s.maxLiteralElements(),
	}
	c.initDebugInfo(ctx, nil)
	return c
}

// maxLiteralElements returns the configured max literal element count,
// or 0 if not set.
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
// Takes rootFunction (*CompiledFunction) which provides the function
// table for cross-function calls.
// Takes initFunction (*CompiledFunction) which is the init function
// to execute.
//
// Returns error when the init function execution fails.
func (s *Service) executeInitFunc(ctx context.Context, rootFunction *CompiledFunction, initFunction *CompiledFunction) error {
	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	vm.functions = rootFunction.functions
	vm.rootFunction = rootFunction
	vm.ensureCallStack()
	defer vm.releaseArena()
	vm.pushFrame(initFunction)
	if _, err := vm.run(0); err != nil {
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

	result, evalErr := c.compileAndRunEvalBody(ctx, s, file, info, lastExpr, hasResult)
	if evalErr != nil {
		return nil, fmt.Errorf(errFmtEvaluatingFile, evalErr)
	}
	return result, nil
}

// compileAndRunVarInits compiles and immediately executes package-level
// variable initialisers. No-op when no global variables are registered.
//
// Takes c (*compiler) which holds the registered global variables.
// Takes files ([]*ast.File) which are the parsed AST files with
// variable declarations.
//
// Returns error when compilation or execution of variable initialisers
// fails.
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
	vm.ensureCallStack()
	defer vm.releaseArena()
	vm.pushFrame(variableInitialisationFunction)
	if _, err := vm.run(0); err != nil {
		return fmt.Errorf(errVarinitFmt, err)
	}
	vm.globals.materialiseStrings(vm.arena)
	return nil
}

// compileNonEvalFuncDecls compiles all function declarations in the
// file except the synthetic _eval_ function.
//
// Takes decls ([]ast.Decl) which are the AST declarations to compile.
//
// Returns error when compilation of any function declaration fails.
func (c *compiler) compileNonEvalFuncDecls(ctx context.Context, decls []ast.Decl) error {
	for _, declaration := range decls {
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

// compileAndRunEvalBody finds the _eval_ function, compiles its body,
// and executes it.
//
// Takes s (*Service) which is the interpreter service for VM
// creation.
// Takes file (*ast.File) which contains the _eval_ function.
// Takes info (*types.Info) which holds type-checking information.
// Takes lastExpr (ast.Expr) which is the last expression for
// result extraction.
// Takes hasResult (bool) which is true when the last statement was
// an expression.
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
	functionDeclaration := findEvalFuncDecl(file)
	if functionDeclaration == nil {
		return nil, nil
	}

	c.scopes.pushScope()

	lastLocation, err := c.compileStmtList(ctx, functionDeclaration.Body.List)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}

	if hasResult {
		lastLocation = c.coerceEvalBoolResult(ctx, info, lastExpr, lastLocation)
		c.function.resultKinds = []registerKind{lastLocation.kind}
		c.emitMoveToRegisterZero(ctx, lastLocation)
	}

	if err := c.scopes.overflowError(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}
	c.function.numRegisters = c.scopes.peakRegisters()
	c.function.optimise()
	c.scopes.popScope()

	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
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

// evalMixed handles code that mixes declarations (func, var, type,
// const) with executable statements, separating declarations from
// statements and placing them at the appropriate scope levels.
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
	result, evalErr := s.doEvalFile(ctx, file)
	if evalErr != nil {
		return nil, fmt.Errorf("evaluating mixed source: %w", evalErr)
	}
	return result, nil
}

// parseMixedSource classifies mixed code into imports, declarations,
// and statements, then reconstructs and parses a valid Go source file.
//
// Takes code (string) which is the mixed Go source code to parse.
//
// Returns *ast.File which is the reconstructed and parsed AST file.
// Returns error when the reconstructed source fails to parse.
func (s *Service) parseMixedSource(code string) (*ast.File, error) {
	cl := classifyLines(strings.Split(code, newlineSep))
	src := buildMixedSource(cl)

	file, err := parser.ParseFile(s.fileSet, evalFileName, src, 0)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errParse, err)
	}
	return file, nil
}

// rewriteLastExprStmt transforms the last expression statement in the
// _eval_ function into a blank assignment (_ = expr), preventing
// go/types from rejecting standalone expressions as unused values.
//
// Takes file (*ast.File) which contains the _eval_ function.
//
// Returns ast.Expr which is the original expression for type lookup.
// Returns bool which is true when a rewrite was performed.
func (*Service) rewriteLastExprStmt(file *ast.File) (ast.Expr, bool) {
	for _, declaration := range file.Decls {
		functionDeclaration, ok := declaration.(*ast.FuncDecl)
		if !ok || functionDeclaration.Name.Name != evalFuncName || functionDeclaration.Body == nil {
			continue
		}

		statements := functionDeclaration.Body.List
		if len(statements) == 0 {
			return nil, false
		}

		last, ok := statements[len(statements)-1].(*ast.ExprStmt)
		if !ok {
			return nil, false
		}

		functionDeclaration.Body.List[len(statements)-1] = &ast.AssignStmt{
			Lhs: []ast.Expr{ast.NewIdent("_")},
			Tok: token.ASSIGN,
			Rhs: []ast.Expr{last.X},
		}
		return last.X, true
	}
	return nil, false
}

// compileExpression compiles a single expression.
//
// Takes code (string) which contains the Go expression source.
//
// Returns *CompiledFunction which is the compiled expression.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) compileExpression(ctx context.Context, code string) (*CompiledFunction, error) {
	wrapped := "package main\nvar _ = " + code
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
// Takes variableInitialisationFunction (*CompiledFunction) which is
// the compiled variable initialiser to execute.
//
// Returns error when execution of the initialiser function fails.
func (s *Service) executeVarInitFunction(ctx context.Context, variableInitialisationFunction *CompiledFunction) error {
	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	vm.ensureCallStack()
	defer vm.releaseArena()
	vm.pushFrame(variableInitialisationFunction)
	if _, err := vm.run(0); err != nil {
		return fmt.Errorf(errVarinitFmt, err)
	}
	vm.globals.materialiseStrings(vm.arena)
	return nil
}

// compileEvalFunction finds and compiles the _eval_ function body.
//
// Takes file (*ast.File) which contains the _eval_ function.
// Takes info (*types.Info) which holds type-checking information.
// Takes lastExpr (ast.Expr) which is the last expression for
// result extraction.
// Takes hasResult (bool) which is true when the last statement was
// an expression.
// Takes evalFunction (*CompiledFunction) which is the compiled function
// shell to populate.
//
// Returns *CompiledFunction which is the compiled eval function, or
// evalFunction when no _eval_ function exists.
// Returns error when compilation fails.
func (c *compiler) compileEvalFunction(ctx context.Context,
	file *ast.File,
	info *types.Info,
	lastExpr ast.Expr,
	hasResult bool,
	evalFunction *CompiledFunction,
) (*CompiledFunction, error) {
	functionDeclaration := findEvalFuncDecl(file)
	if functionDeclaration == nil {
		return evalFunction, nil
	}

	c.scopes.pushScope()

	lastLocation, err := c.compileStmtList(ctx, functionDeclaration.Body.List)
	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}

	if hasResult {
		lastLocation = c.coerceEvalBoolResult(ctx, info, lastExpr, lastLocation)
		c.function.resultKinds = []registerKind{lastLocation.kind}
		c.emitMoveToRegisterZero(ctx, lastLocation)
	}

	if err := c.scopes.overflowError(); err != nil {
		return nil, fmt.Errorf("compiling eval function: %w", err)
	}
	c.function.numRegisters = c.scopes.peakRegisters()
	c.function.optimise()
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
	compiled, compileErr := s.compileFile(ctx, file)
	if compileErr != nil {
		return nil, fmt.Errorf("compiling mixed source: %w", compileErr)
	}
	return compiled, nil
}

// shouldIncludeFile evaluates //go:build constraints in the file's
// comments.
//
// Takes file (*ast.File) which is the parsed AST file to evaluate.
//
// Returns true when no constraint exists or the constraint matches
// the configured build tags.
func (s *Service) shouldIncludeFile(file *ast.File) bool {
	for _, cg := range file.Comments {
		if cg.Pos() >= file.Package {
			break
		}
		for _, c := range cg.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expression, err := constraint.Parse(c.Text)
			if err != nil {
				continue
			}
			return expression.Eval(s.buildTagMatcher())
		}
	}
	return true
}

// buildTagMatcher returns a predicate that reports whether a build
// tag is active. The default set includes the current GOOS, GOARCH,
// and Go version, plus any user-provided tags from WithBuildTags.
//
// Returns func(string) bool which reports whether a given tag is active.
func (s *Service) buildTagMatcher() func(string) bool {
	tags := make(map[string]bool)
	tags[runtime.GOOS] = true
	tags[runtime.GOARCH] = true

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

// applyEnvOverrides patches the symbol registry's "os" package so
// that Getenv, LookupEnv, Environ, Setenv, and Unsetenv operate on
// the configured environment map instead of the host process.
func (s *Service) applyEnvOverrides() {
	if s.config == nil || len(s.config.env) == 0 || s.config.envApplied {
		return
	}
	s.config.envApplied = true

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
