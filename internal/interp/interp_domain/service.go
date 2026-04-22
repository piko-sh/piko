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
	"go/parser"
	"go/token"
	"go/types"
	"maps"
	"reflect"
	"sync/atomic"
	"time"
	"unsafe"

	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// evalFileName is the synthetic filename used for parsed eval snippets.
	evalFileName = "eval.go"

	// pkgUnsafe is the import path for the unsafe package.
	pkgUnsafe = "unsafe"

	// mainPackageName is the package name used for eval compilations.
	mainPackageName = "main"

	// newlineSep is the newline separator used when joining source lines.
	newlineSep = "\n"

	// errChainFmt is the format string for wrapping errors with a sentinel.
	errChainFmt = "%w: %w"

	// errVarinitFmt is the format string for variable initialiser errors.
	errVarinitFmt = "varinit: %w"

	// errChainMessageFmt is the format string for wrapping errors with a
	// sentinel and a message string.
	errChainMessageFmt = "%w: %s: %w"

	// defaultMaxExecutionTime is the maximum duration for any single
	// evaluation when no explicit limit is configured. Prevents
	// untrusted code from running indefinitely.
	defaultMaxExecutionTime = 15 * time.Minute

	// defaultMaxAllocSize is the maximum element count for a single
	// allocation when no explicit limit is configured.
	defaultMaxAllocSize = 1 << 30

	// defaultMaxGoroutines is the maximum concurrent goroutines for
	// interpreted code when no explicit limit is configured.
	defaultMaxGoroutines int32 = 10_000

	// defaultMaxOutputSize is the maximum bytes print/println may
	// write when no explicit limit is configured.
	defaultMaxOutputSize = 256 << 20

	// errFmtCompilingFileSet is the format string for wrapping
	// errors during file set compilation.
	errFmtCompilingFileSet = "compiling file set: %w"

	// errFmtCompilingProgram is the format string for wrapping
	// errors during program compilation.
	errFmtCompilingProgram = "compiling program: %w"
)

// serviceConfig holds optional configuration for the interpreter service.
type serviceConfig struct {
	// bytecodeStore is an optional port for persisting compiled bytecode.
	bytecodeStore BytecodeStorePort

	// arenaFactory is an optional factory for creating register arenas.
	arenaFactory func() *RegisterArena

	// compilationSnapshotCallback is called at the end of CompileProgram
	// with the compiled output so far, regardless of whether compilation
	// succeeded or failed partway through. This enables bytecode emission
	// for debugging even when a later package fails to compile.
	compilationSnapshotCallback func(*CompiledFileSet)

	// env holds environment variable overrides for interpreted code.
	env map[string]string

	// debugger is an optional debugger to attach to each VM.
	debugger *Debugger

	// costTable is the per-opcode cost table for cost metering. Nil
	// means use the default cost table.
	costTable *CostTable

	// buildTags holds additional build tags for constraint evaluation.
	buildTags []string

	// maxExecutionTime is the maximum duration for any single evaluation.
	maxExecutionTime time.Duration

	// costBudget is the maximum total computation cost for a single
	// execution. Zero means cost metering is disabled.
	costBudget int64

	// maxAllocSize is the maximum element count for a single allocation.
	maxAllocSize int

	// maxCallDepth is the maximum call stack depth before overflow.
	maxCallDepth int

	// maxOutputSize is the maximum bytes print and println may write.
	maxOutputSize int

	// maxSourceSize is the maximum total source code size in bytes
	// accepted for compilation. Zero means no limit.
	maxSourceSize int

	// maxStringSize is the maximum string length in bytes that a
	// concatenation may produce. Zero means no limit.
	maxStringSize int

	// maxLiteralElements is the maximum number of elements in a
	// single composite literal (slice, array, map). Zero means no
	// limit.
	maxLiteralElements int

	// maxGoroutines is the maximum concurrent goroutines for
	// interpreted code.
	maxGoroutines int32

	// features controls which Go language constructs are allowed
	// during compilation. Zero value means InterpFeaturesAll.
	features InterpFeature

	// yieldInterval is the number of instructions between
	// runtime.Gosched() calls for cooperative scheduling.
	yieldInterval uint32

	// envApplied tracks whether environment overrides have been
	// applied.
	envApplied bool

	// forceGoDispatch forces the pure Go dispatch loop on all
	// architectures.
	forceGoDispatch bool

	// debugInfo enables debug information generation during compilation.
	debugInfo bool
}

// Option configures the interpreter service.
type Option func(*serviceConfig)

// Service implements the core interpreter logic. It parses, type-checks,
// compiles, and executes Go source code.
type Service struct {
	// fileSet is reused across evaluations within the same interpreter.
	fileSet *token.FileSet

	// symbols holds pre-registered native symbols.
	symbols *SymbolRegistry

	// globals holds package-level variables.
	globals *globalStore

	// config holds optional service configuration.
	config *serviceConfig

	// limits holds resource constraints threaded into each VM.
	limits vmLimits

	// features controls which Go language constructs are allowed.
	features InterpFeature

	// lastCostUsed holds the total computation cost consumed by the most recent execution.
	lastCostUsed atomic.Int64
}

// NewService creates a new interpreter service.
//
// Takes opts (Option variadic) which configure build tags, environment
// variables, and other interpreter behaviour.
//
// Returns *Service which is ready to evaluate code.
func NewService(opts ...Option) *Service {
	config := &serviceConfig{}
	for _, opt := range opts {
		opt(config)
	}

	symbols := NewSymbolRegistry(nil)
	symbols.ProtectPackage(pkgUnsafe)

	features := config.features
	if features == 0 {
		features = InterpFeaturesAll
	}

	s := &Service{
		fileSet:  token.NewFileSet(),
		globals:  newGlobalStore(),
		symbols:  symbols,
		config:   config,
		features: features,
	}
	s.limits = s.buildLimits()
	return s
}

// Eval evaluates Go source code and returns the result.
//
// The source can be a single expression, a statement, or a complete
// Go source file. For expressions, the expression's value is returned.
// For statements, nil is returned.
//
// Takes code (string) which contains the Go source code to evaluate.
//
// Returns any which is the result of evaluating the code.
// Returns error when parsing, type-checking, compilation, or execution
// fails.
func (s *Service) Eval(ctx context.Context, code string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	if err := s.checkSourceSize(len(code)); err != nil {
		return nil, err
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	s.applyEnvOverrides()

	_, err := parser.ParseExprFrom(s.fileSet, evalFileName, code, 0)
	if err == nil {
		result, evalErr := s.evalExpr(ctx, code)
		if evalErr != nil {
			return nil, fmt.Errorf("evaluating expression: %w", evalErr)
		}
		return result, nil
	}

	wrappedCode := "package main\nfunc _eval_() {\n" + code + "\n}"
	file, parseErr := parser.ParseFile(s.fileSet, evalFileName, wrappedCode, 0)
	if parseErr == nil {
		result, evalErr := s.doEvalFile(ctx, file)
		if evalErr == nil {
			return result, nil
		}

		if mixedResult, mixedErr := s.evalMixed(ctx, code); mixedErr == nil {
			return mixedResult, nil
		}
		return nil, fmt.Errorf("evaluating expression: %w", evalErr)
	}

	result, evalErr := s.evalMixed(ctx, code)
	if evalErr != nil {
		return nil, fmt.Errorf("evaluating expression: %w", evalErr)
	}
	return result, nil
}

// Compile parses, type-checks, and compiles Go source code into a
// CompiledFunction without executing it. The returned function can be
// executed multiple times via Execute.
//
// Takes code (string) which contains the Go source code to compile.
//
// Returns *CompiledFunction which is the compiled representation.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) Compile(ctx context.Context, code string) (*CompiledFunction, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	if err := s.checkSourceSize(len(code)); err != nil {
		return nil, err
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	_, err := parser.ParseExprFrom(s.fileSet, evalFileName, code, 0)
	if err == nil {
		result, compileErr := s.compileExpression(ctx, code)
		if compileErr != nil {
			return nil, fmt.Errorf("compiling source: %w", compileErr)
		}
		return result, nil
	}

	wrappedCode := "package main\nfunc _eval_() {\n" + code + "\n}"
	file, parseErr := parser.ParseFile(s.fileSet, evalFileName, wrappedCode, 0)
	if parseErr == nil {
		result, compileErr := s.compileFile(ctx, file)
		if compileErr == nil {
			return result, nil
		}

		if mixedResult, mixedErr := s.compileMixed(ctx, code); mixedErr == nil {
			return mixedResult, nil
		}
		return nil, fmt.Errorf("compiling source: %w", compileErr)
	}

	result, compileErr := s.compileMixed(ctx, code)
	if compileErr != nil {
		return nil, fmt.Errorf("compiling source: %w", compileErr)
	}
	return result, nil
}

// Execute runs a pre-compiled function and returns its result.
//
// Takes compiledFunction (*CompiledFunction) which is the compiled
// function to run.
//
// Returns any which is the result of executing the function.
// Returns error when execution fails.
func (s *Service) Execute(ctx context.Context, compiledFunction *CompiledFunction) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()

	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.attachDebuggerToVM(vm)
	result, err := vm.execute(compiledFunction)
	s.recordCost(vm)
	return result, err
}

// EvalFile parses a complete Go source file, compiles all declarations,
// and executes the named entrypoint function. This is a convenience
// wrapper around CompileFileSet + ExecuteEntrypoint for single-file use.
//
// The source must be a valid Go file with a package clause. The
// entrypoint must name a function declared in the file.
//
// Takes source (string) which is the complete Go source file.
// Takes entrypoint (string) which is the function name to execute.
//
// Returns any which is the entrypoint function's return value.
// Returns error when parsing, type-checking, compilation, or execution
// fails.
func (s *Service) EvalFile(ctx context.Context, source string, entrypoint string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()

	cfs, err := s.CompileFileSet(ctx, map[string]string{"main.go": source})
	if err != nil {
		return nil, fmt.Errorf("evaluating file: %w", err)
	}
	result, entryErr := s.ExecuteEntrypoint(ctx, cfs, entrypoint)
	if entryErr != nil {
		return nil, fmt.Errorf("evaluating file: %w", entryErr)
	}
	return result, nil
}

// funcDeclEntry pairs a function declaration with its compiled shell,
// used between the register and compile passes.
type funcDeclEntry struct {
	// declaration is the parsed function declaration AST node.
	declaration *ast.FuncDecl

	// compiledFunction is the compiled function shell for this declaration.
	compiledFunction *CompiledFunction
}

// parsedPackage holds the parsed and filtered files for a single
// package within a multi-package compilation.
type parsedPackage struct {
	// importPath is the fully qualified import path for this package.
	importPath string

	// relPath is the relative path within the module.
	relPath string

	// packageName is the declared package name from the source files.
	packageName string

	// files holds the parsed and build-tag-filtered AST files.
	files []*ast.File
}

// CompileFileSet parses and type-checks multiple Go source files as a
// single package, returning a CompiledFileSet that can be executed
// multiple times via ExecuteEntrypoint.
//
// Takes sources (map[string]string) where keys are filenames (used for
// error reporting and deterministic ordering) and values are source
// code strings.
//
// Returns *CompiledFileSet which holds all compiled functions.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) CompileFileSet(ctx context.Context, sources map[string]string) (*CompiledFileSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	totalSourceBytes := 0
	for _, src := range sources {
		totalSourceBytes += len(src)
	}
	if err := s.checkSourceSize(totalSourceBytes); err != nil {
		return nil, err
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	s.applyEnvOverrides()

	files, err := s.parseAndFilterFiles(sources)
	if err != nil {
		return nil, fmt.Errorf(errFmtCompilingFileSet, err)
	}

	info := s.newTypesInfo()
	conf := s.newTypesConfig()
	if _, err := conf.Check(mainPackageName, s.fileSet, files, info); err != nil {
		enriched := s.enrichTypeCheckError(err, files, nil)
		return nil, fmt.Errorf(errChainFmt, errTypeCheck, enriched)
	}

	rootFunction := &CompiledFunction{name: "<fileset>"}
	c := s.newFileSetCompiler(ctx, rootFunction, info)

	c.registerPackageLevelVarsFromFiles(ctx, files)

	if err := c.twoPassCompileFuncs(ctx, files, ""); err != nil {
		return nil, fmt.Errorf(errFmtCompilingFileSet, err)
	}

	variableInitialisationFunction, err := c.compileVariableInitFunction(ctx, files)
	if err != nil {
		return nil, fmt.Errorf(errFmtCompilingFileSet, err)
	}

	if !s.features.Has(InterpFeatureRecursion) {
		if err := detectRecursion(rootFunction); err != nil {
			return nil, fmt.Errorf(errFmtCompilingFileSet, err)
		}
	}

	entrypoints := make(map[string]uint16, len(c.funcTable))
	maps.Copy(entrypoints, c.funcTable)

	return &CompiledFileSet{
		root:                 rootFunction,
		entrypoints:          entrypoints,
		initFunctionIndices:  c.initFunctionIndices,
		variableInitFunction: variableInitialisationFunction,
	}, nil
}

// registerPackageLevelVarsFromFiles scans all files for package-level
// var declarations and registers them in the compiler.
//
// Takes files ([]*ast.File) which are the parsed AST files to scan.
func (c *compiler) registerPackageLevelVarsFromFiles(ctx context.Context, files []*ast.File) {
	for _, file := range files {
		c.registerPackageLevelVarsFromDecls(ctx, file.Decls)
	}
}

// registerPackageLevelVarsFromDecls scans declarations for package-
// level var specs and registers them in the compiler.
//
// Takes decls ([]ast.Decl) which are the AST declarations to scan.
func (c *compiler) registerPackageLevelVarsFromDecls(ctx context.Context, decls []ast.Decl) {
	for _, declaration := range decls {
		genDecl, ok := declaration.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}
		for _, spec := range genDecl.Specs {
			if vs, ok := spec.(*ast.ValueSpec); ok {
				c.registerPackageLevelVar(ctx, vs)
			}
		}
	}
}

// twoPassCompileFuncs performs a two-pass compilation: first register
// all function declarations across files so cross-file references
// resolve, then compile all function bodies.
//
// Takes files ([]*ast.File) which are the parsed AST files to compile.
// Takes packageLabel (string) which is included in error messages for
// multi-package compilations and is empty for single-package ones.
//
// Returns error when registration or body compilation fails.
func (c *compiler) twoPassCompileFuncs(ctx context.Context, files []*ast.File, packageLabel string) error {
	var entries []funcDeclEntry
	for _, file := range files {
		for _, declaration := range file.Decls {
			functionDeclaration, ok := declaration.(*ast.FuncDecl)
			if !ok {
				continue
			}
			cf, err := c.registerFuncDecl(ctx, functionDeclaration)
			if err != nil {
				return c.wrapCompileError(ctx, err, packageLabel)
			}
			entries = append(entries, funcDeclEntry{declaration: functionDeclaration, compiledFunction: cf})
		}
	}

	for _, entry := range entries {
		if err := c.compileFuncBody(ctx, entry.declaration, entry.compiledFunction); err != nil {
			return c.wrapCompileError(ctx, err, packageLabel)
		}
	}
	return nil
}

// wrapCompileError wraps an error with errCompilation and an optional
// package label for multi-package compilations.
//
// Takes err (error) which is the original compilation error.
// Takes packageLabel (string) which is the package label for error
// context, or empty for single-package compilations.
//
// Returns error wrapped with errCompilation and the package label.
func (*compiler) wrapCompileError(_ context.Context, err error, packageLabel string) error {
	if packageLabel != "" {
		return fmt.Errorf(errChainMessageFmt, errCompilation, packageLabel, err)
	}
	return fmt.Errorf(errChainFmt, errCompilation, err)
}

// compileVariableInitFunction compiles package-level variable
// initialisers from all files into a dedicated function.
//
// Takes files ([]*ast.File) which are the parsed AST files with
// variable declarations.
//
// Returns *CompiledFunction which is the initialiser function, or nil
// when no global variables exist.
// Returns error when compilation of any initialiser fails.
func (c *compiler) compileVariableInitFunction(ctx context.Context, files []*ast.File) (*CompiledFunction, error) {
	if len(c.globalVars) == 0 {
		return nil, nil
	}

	initFunction := &CompiledFunction{name: "<varinit>"}
	savedFunction := c.function
	c.function = initFunction
	c.scopes.pushScope()

	err := c.compileVarInitSpecs(ctx, files)

	c.function.emit(opReturnVoid, 0, 0, 0)
	if overflowErr := c.scopes.overflowError(); overflowErr != nil {
		err = overflowErr
	}
	initFunction.numRegisters = c.scopes.peakRegisters()
	c.scopes.popScope()
	c.function = savedFunction

	if err != nil {
		return nil, fmt.Errorf(errChainFmt, errCompilation, err)
	}
	return initFunction, nil
}

// compileVarInitSpecs walks all files and compiles each package-level
// variable initialiser.
//
// Takes files ([]*ast.File) which are the parsed AST files to process.
//
// Returns error when any variable initialiser fails to compile.
func (c *compiler) compileVarInitSpecs(ctx context.Context, files []*ast.File) error {
	for _, file := range files {
		for _, declaration := range file.Decls {
			watermark := c.scopes.alloc.snapshot()
			if err := c.compileVarDeclInit(ctx, declaration); err != nil {
				return fmt.Errorf("compiling variable init specs: %w", err)
			}
			c.scopes.restoreWatermark(watermark)
		}
	}
	return nil
}

// compileVarDeclInit compiles variable initialisers from a single
// declaration. Non-var declarations are silently skipped.
//
// Takes declaration (ast.Decl) which is the AST declaration to compile.
//
// Returns error when compilation of any variable initialiser fails.
func (c *compiler) compileVarDeclInit(ctx context.Context, declaration ast.Decl) error {
	genDecl, ok := declaration.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return nil
	}
	for _, spec := range genDecl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		if err := c.compilePackageLevelVarInit(ctx, vs); err != nil {
			return fmt.Errorf("compiling variable declaration: %w", err)
		}
	}
	return nil
}

// ExecuteEntrypoint runs a named function from a pre-compiled file set,
// executing init functions first (in source order) before the entrypoint.
//
// Takes cfs (*CompiledFileSet) which is the compiled file set.
// Takes entrypoint (string) which is the function name to execute.
//
// Returns any which is the entrypoint function's return value.
// Returns error when the entrypoint is not found or execution fails.
func (s *Service) ExecuteEntrypoint(ctx context.Context, cfs *CompiledFileSet, entrypoint string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()

	index, ok := cfs.entrypoints[entrypoint]
	if !ok {
		return nil, fmt.Errorf("%w: %q", errEntrypointNotFound, entrypoint)
	}

	if cfs.variableInitFunction != nil {
		vm := newVM(ctx, s.globals, s.symbols)
		vm.limits = s.limits
		vm.functions = cfs.root.functions
		vm.rootFunction = cfs.root
		vm.ensureCallStack()
		defer vm.releaseArena()
		vm.pushFrame(cfs.variableInitFunction)
		if _, err := vm.run(0); err != nil {
			return nil, fmt.Errorf(errVarinitFmt, err)
		}
	}

	for _, initIndex := range cfs.initFunctionIndices {
		if err := s.executeInitFunc(ctx, cfs.root, cfs.root.functions[initIndex]); err != nil {
			return nil, fmt.Errorf("executing entrypoint: %w", err)
		}
	}

	entrypointFunction := cfs.root.functions[index]
	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.attachDebuggerToVM(vm)
	vm.functions = cfs.root.functions
	vm.rootFunction = cfs.root
	arena := GetRegisterArena()
	vm.arena = arena
	vm.callStack = arena.frameStack()
	vm.sizeArenaFromFunctions(cfs.root)
	vm.asmCallInfoTables, _ = buildASMCallInfoTables(entrypointFunction, vm.functions)
	vm.asmCallInfoBases = arena.CallInfoBases()
	vm.asmDispatchSaves = arena.dispatchSaves()
	if table := vm.asmCallInfoTables[entrypointFunction]; len(table) > 0 {
		vm.asmCallInfoBases[0] = uintptr(unsafe.Pointer(&table[0]))
	}
	vm.pushFrame(entrypointFunction)
	result, err := vm.runDispatched(0)
	vm.callStack = nil
	vm.asmCallInfoTables = nil
	vm.asmCallInfoBases = nil
	vm.asmDispatchSaves = nil
	PutRegisterArena(arena)
	return result, err
}

// ExecuteInits runs variable initialisers and init functions from a
// pre-compiled file set without requiring a named entrypoint. This is
// useful when the compiled code only needs its init side-effects (such
// as registering functions into a global registry).
//
// Takes cfs (*CompiledFileSet) which is the compiled file set whose
// init functions will be executed.
//
// Returns error when a variable initialiser or init function fails.
func (s *Service) ExecuteInits(ctx context.Context, cfs *CompiledFileSet) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()

	if cfs.variableInitFunction != nil {
		vm := newVM(ctx, s.globals, s.symbols)
		vm.limits = s.limits
		vm.ensureCallStack()
		defer vm.releaseArena()
		vm.pushFrame(cfs.variableInitFunction)
		if _, err := vm.run(0); err != nil {
			return fmt.Errorf(errVarinitFmt, err)
		}
		vm.globals.materialiseStrings(vm.arena)
	}

	for _, initIndex := range cfs.initFunctionIndices {
		fn := cfs.root.functions[initIndex]
		if err := s.executeInitFunc(ctx, cfs.root, fn); err != nil {
			return fmt.Errorf("executing init functions: %w", err)
		}
	}

	return nil
}

// CompileProgram compiles multiple packages from source, automatically
// resolving import dependencies and wiring cross-package calls via
// the symbol registry.
//
// Takes modulePath (string) which is the module path (e.g. "testpkg").
// Takes packages (map[string]map[string]string) which maps relative
// package paths to filename-to-source maps.
//
// Returns *CompiledFileSet which contains all compiled functions.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) CompileProgram(ctx context.Context, modulePath string, packages map[string]map[string]string) (*CompiledFileSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf(errChainFmt, errExecutionCancelled, err)
	}
	if err := s.checkSourceSize(countSourceBytes(packages)); err != nil {
		return nil, err
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	s.applyEnvOverrides()

	parsed, err := s.parseAllPackages(modulePath, packages)
	if err != nil {
		return nil, fmt.Errorf(errFmtCompilingProgram, err)
	}

	order, err := topoSort(buildDependencyGraph(parsed))
	if err != nil {
		return nil, fmt.Errorf(errFmtCompilingProgram, err)
	}

	rootFunction := &CompiledFunction{name: "<program>"}
	crossPackageMethods := make(map[string]uint16)
	var allInitFuncs []uint16
	var mainFuncTable map[string]uint16
	var lastFuncTable map[string]uint16

	if s.config != nil && s.config.compilationSnapshotCallback != nil {
		defer func() {
			entrypoints := resolveEntrypoints(mainFuncTable, lastFuncTable)
			s.config.compilationSnapshotCallback(&CompiledFileSet{
				root:                rootFunction,
				entrypoints:         entrypoints,
				initFunctionIndices: allInitFuncs,
			})
		}()
	}

	allInitFuncs, mainFuncTable, lastFuncTable, err = s.compileOrderedPackages(
		ctx, order, parsed, rootFunction, crossPackageMethods,
	)
	if err != nil {
		return nil, fmt.Errorf(errFmtCompilingProgram, err)
	}

	if !s.features.Has(InterpFeatureRecursion) {
		if err := detectRecursion(rootFunction); err != nil {
			return nil, fmt.Errorf(errFmtCompilingProgram, err)
		}
	}

	return &CompiledFileSet{
		root:                rootFunction,
		entrypoints:         resolveEntrypoints(mainFuncTable, lastFuncTable),
		initFunctionIndices: allInitFuncs,
	}, nil
}

// countSourceBytes totals the byte length across all source files in
// a multi-package compilation input.
//
// Takes packages (map[string]map[string]string) which maps package
// paths to filename-to-source maps.
//
// Returns int which is the total byte count across all source files.
func countSourceBytes(packages map[string]map[string]string) int {
	total := 0
	for _, files := range packages {
		for _, src := range files {
			total += len(src)
		}
	}
	return total
}

// packageCompileResult holds the output of compiling a single package
// within a multi-package compilation.
type packageCompileResult struct {
	// funcTable maps function names to their indices in the root function.
	funcTable map[string]uint16

	// info holds the type-checking results for this package.
	info *types.Info

	// typesPackage is the types.Package produced by type-checking.
	typesPackage *types.Package

	// rootFunction is the root compiled function containing all sub-functions.
	rootFunction *CompiledFunction

	// initFunctionIndices holds indices of init functions in source order.
	initFunctionIndices []uint16
}

// UseSymbols sets the pre-registered native symbols for import
// resolution, protecting the "unsafe" package from override.
//
// Takes symbols (*SymbolRegistry) which is the registry to use.
func (s *Service) UseSymbols(symbols *SymbolRegistry) {
	symbols.ProtectPackage(pkgUnsafe)
	s.symbols = symbols
}

// UseSymbolProviders builds a SymbolRegistry from one or more symbol
// providers.
//
// Later providers override earlier ones for the same package/symbol.
// The "unsafe" package is always protected.
//
// Takes providers (SymbolProviderPort variadic) which are the symbol
// providers to compose.
func (s *Service) UseSymbolProviders(providers ...SymbolProviderPort) {
	composite := newCompositeSymbolProvider(providers...)
	exports := composite.Exports()
	delete(exports, pkgUnsafe)
	s.symbols = NewSymbolRegistry(exports)
	s.symbols.ProtectPackage(pkgUnsafe)

	for _, p := range providers {
		if tp, ok := p.(TypesPackageProviderPort); ok {
			for path, pkg := range tp.TypesPackages() {
				s.symbols.RegisterTypesPackage(path, pkg)
			}
		}
	}

	s.symbols.SynthesiseAll()
}

// RegisterPackage registers symbols under a package path in the symbol
// registry. This is useful for creating package aliases by registering
// the same symbol set under a shorter import path.
//
// Takes packagePath (string) which is the import path to register.
// Takes symbols (map[string]reflect.Value) which maps symbol names to
// their reflected values.
func (s *Service) RegisterPackage(packagePath string, symbols map[string]reflect.Value) {
	s.symbols.RegisterPackage(packagePath, symbols)
}

// HasRegisteredPackage reports whether the given import path is
// available in the symbol registry.
//
// Takes importPath (string) which is the full Go import path to check.
//
// Returns true if the package is already available via the symbol
// registry.
func (s *Service) HasRegisteredPackage(importPath string) bool {
	return s.symbols.HasPackage(importPath)
}

// Reset clears the interpreter state for reuse.
func (s *Service) Reset() {
	s.fileSet = token.NewFileSet()
	s.globals.reset()
}

// Clone creates a copy of the service sharing symbols but with
// fresh execution state.
//
// Returns *Service which is a new service with shared symbols and
// independent state.
func (s *Service) Clone() *Service {
	cloned := &Service{
		fileSet:  token.NewFileSet(),
		symbols:  s.symbols,
		globals:  newGlobalStore(),
		config:   s.config,
		features: s.features,
	}
	cloned.limits = cloned.buildLimits()
	return cloned
}

// SetCompilationSnapshot sets a callback that receives a snapshot of
// the compiled output at the end of CompileProgram, regardless of
// whether compilation succeeded or failed. This creates a private
// copy of the service config so that the callback does not affect
// other clones sharing the same golden config.
//
// Takes callback (func(*CompiledFileSet)) which receives the
// snapshot.
func (s *Service) SetCompilationSnapshot(callback func(*CompiledFileSet)) {
	if s.config == nil {
		s.config = &serviceConfig{}
	} else {
		s.config = new(*s.config)
	}
	s.config.compilationSnapshotCallback = callback
}

// LastCostUsed returns the total computation cost consumed by the most
// recent execution when cost metering is enabled.
//
// Returns int64 which is the cost consumed, or 0 when cost metering is
// disabled.
func (s *Service) LastCostUsed() int64 {
	return s.lastCostUsed.Load()
}

// buildLimits constructs vmLimits from the service configuration.
// Each call creates a fresh resourceTracker so that cloned services
// and separate evaluations do not share counters.
//
// Returns vmLimits configured from the service settings.
func (s *Service) buildLimits() vmLimits {
	limits := vmLimits{
		maxAllocSize:  defaultMaxAllocSize,
		maxGoroutines: defaultMaxGoroutines,
		maxOutputSize: defaultMaxOutputSize,
	}
	if s.config != nil {
		s.applyConfigLimits(&limits)
	}
	limits.tracker = &resourceTracker{}
	return limits
}

// applyConfigLimits copies non-zero config values into the given
// vmLimits, enabling cost accounting and yield when configured.
//
// Takes limits (*vmLimits) which is the limits struct to populate
// from the service config.
func (s *Service) applyConfigLimits(limits *vmLimits) {
	limits.arenaFactory = s.config.arenaFactory
	limits.maxCallDepth = s.config.maxCallDepth
	limits.forceGoDispatch = s.config.forceGoDispatch
	if s.config.maxAllocSize > 0 {
		limits.maxAllocSize = s.config.maxAllocSize
	}
	if s.config.maxGoroutines > 0 {
		limits.maxGoroutines = s.config.maxGoroutines
	}
	if s.config.maxOutputSize > 0 {
		limits.maxOutputSize = s.config.maxOutputSize
	}
	if s.config.costBudget > 0 {
		limits.costBudget = s.config.costBudget
		limits.forceGoDispatch = true
		if s.config.costTable != nil {
			limits.costTable = s.config.costTable
		} else {
			limits.costTable = &defaultCostTable
		}
	}
	if s.config.maxStringSize > 0 {
		limits.maxStringSize = s.config.maxStringSize
	}
	if s.config.yieldInterval > 0 {
		limits.yieldInterval = s.config.yieldInterval
		limits.forceGoDispatch = true
	}
}

// attachDebuggerToVM attaches the configured debugger (if any) to a
// VM.
//
// Takes vm (*VM) which is the virtual machine to attach the debugger
// to.
func (s *Service) attachDebuggerToVM(vm *VM) {
	if s.config != nil && s.config.debugger != nil {
		s.config.debugger.attachToVM(vm)
	}
}

// applyMaxExecutionTime wraps ctx with the configured maximum execution
// time, if set. The cancel function must be deferred by the caller.
//
// Returns context.Context with the deadline applied and context.CancelFunc
// that must be deferred.
func (s *Service) applyMaxExecutionTime(ctx context.Context) (context.Context, context.CancelFunc) {
	limit := defaultMaxExecutionTime
	if s.config != nil && s.config.maxExecutionTime > 0 {
		limit = s.config.maxExecutionTime
	}
	return context.WithTimeoutCause(ctx, limit, errExecutionCancelled)
}

// checkSourceSize returns an error if the total source code size
// exceeds the configured maximum.
//
// Takes totalBytes (int) which is the total source code size in
// bytes to validate.
//
// Returns error when the size exceeds the configured limit, or nil.
func (s *Service) checkSourceSize(totalBytes int) error {
	if s.config != nil && s.config.maxSourceSize > 0 && totalBytes > s.config.maxSourceSize {
		return fmt.Errorf("%w: %d bytes exceeds limit %d", errSourceSizeLimit, totalBytes, s.config.maxSourceSize)
	}
	return nil
}

// recordCost stores the cost consumed by a VM execution into the
// service for later retrieval via LastCostUsed. When cost metering
// is disabled (costBudget == 0) this is a no-op to avoid any
// overhead on the hot path.
//
// Takes vm (*VM) which is the virtual machine whose cost to record.
func (s *Service) recordCost(vm *VM) {
	if vm.limits.costBudget > 0 {
		s.lastCostUsed.Store(vm.limits.costBudget - vm.costRemaining)
	}
}
