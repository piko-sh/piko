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
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"maps"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/safeerror"
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

	// errChainMessageFmt is the format string for wrapping errors with a sentinel and a
	// message string.
	errChainMessageFmt = "%w: %s: %w"

	// safeMessageEvaluationFailed is the user-safe summary returned to LSP / REPL callers
	// when Eval fails. The underlying go/types or compiler diagnostic stays in the wrapped
	// cause for developer builds; production surfaces only this generic line.
	safeMessageEvaluationFailed = "evaluation failed"

	// safeMessageEvaluationCancelled is the user-safe summary returned when Eval is
	// cancelled or its deadline elapses.
	safeMessageEvaluationCancelled = "evaluation cancelled"

	// safeMessageCompilationFailed is the user-safe summary returned when Compile fails.
	// Mirrors safeMessageEvaluationFailed for the compile-only entrypoint.
	safeMessageCompilationFailed = "compilation failed"

	// safeMessageCompilationCancelled is the user-safe summary returned when Compile is
	// cancelled or its deadline elapses.
	safeMessageCompilationCancelled = "compilation cancelled"

	// safeMessageSubmissionFailed is the user-safe summary returned by Session.Submit when
	// any pipeline step (preprocess, classify, re-check, compile, execute) fails.
	safeMessageSubmissionFailed = "session submission failed"

	// defaultMaxExecutionTime is the maximum duration for any single evaluation when no
	// explicit limit is configured. Prevents untrusted code from running indefinitely.
	defaultMaxExecutionTime = 15 * time.Minute

	// defaultMaxAllocSize is the maximum element count for a single allocation when no
	// explicit limit is configured.
	defaultMaxAllocSize = 1 << 30

	// defaultMaxGoroutines is the maximum concurrent goroutines for interpreted code when no
	// explicit limit is configured.
	defaultMaxGoroutines int32 = 10_000

	// defaultMaxOutputSize is the maximum bytes print/println may write when no explicit
	// limit is configured.
	defaultMaxOutputSize = 256 << 20

	// errFmtCompilingFileSet is the format string for wrapping errors during file set
	// compilation.
	errFmtCompilingFileSet = "compiling file set: %w"

	// errFmtCompilingProgram is the format string for wrapping errors during program
	// compilation.
	errFmtCompilingProgram = "compiling program: %w"
)

// serviceConfig holds optional configuration for the interpreter service.
type serviceConfig struct {
	// bytecodeStore is an optional port for persisting compiled bytecode.
	bytecodeStore BytecodeStorePort

	// clock is the time source interpreted code observes through time.Now, time.Since,
	// time.Sleep, time.NewTimer, time.NewTicker.
	//
	// nil selects WallClock so interpreted code observes the host process's real time. Hosts
	// that need deterministic replay or test-controllable time install a virtual clock via
	// WithClock.
	clock Clock

	// capabilityHook is consulted before every gated native operation.
	//
	// nil means no gating (permissive default). Production hosts running untrusted code must
	// install a hook via WithCapabilityHook or Service.SetCapabilityHook.
	capabilityHook CapabilityHook

	// arenaFactory is an optional factory for creating register arenas.
	arenaFactory func() *RegisterArena

	// compilationSnapshotCallback is called at the end of CompileProgram with the compiled
	// output so far, regardless of whether compilation succeeded or failed partway through.
	// This enables bytecode emission for debugging even when a later package fails to
	// compile.
	compilationSnapshotCallback func(*CompiledFileSet)

	// env holds environment variable overrides for interpreted code.
	env map[string]string

	// deniedImports holds import paths a script may never import.
	deniedImports map[string]struct{}

	// allowedImports restricts which external packages a script may import.
	allowedImports map[string]struct{}

	// debugger is an optional debugger to attach to each VM.
	debugger *Debugger

	// costTable is the per-opcode cost table for cost metering. Nil means use the default
	// cost table.
	costTable *CostTable

	// envOnce guards one-time application of environment overrides.
	envOnce *sync.Once

	// buildTags holds additional build tags for constraint evaluation.
	buildTags []string

	// maxLiteralElements is the maximum number of elements in a single composite literal
	// (slice, array, map). Zero means no limit.
	maxLiteralElements int

	// maxExecutionTime is the maximum duration for any single evaluation.
	maxExecutionTime time.Duration

	// maxOutputSize is the maximum bytes print and println may write.
	maxOutputSize int

	// maxSourceSize is the maximum total source code size in bytes accepted for compilation.
	// Zero means no limit.
	maxSourceSize int

	// maxStringSize is the maximum string length in bytes that a concatenation may produce.
	// Zero means no limit.
	maxStringSize int

	// maxAllocSize is the maximum element count for a single allocation.
	maxAllocSize int

	// maxCallDepth is the maximum call stack depth before overflow.
	maxCallDepth int

	// maxSpecialisations caps the number of generic-function specialisations registered per
	// generic callee. Zero selects defaultMaxSpecialisations (1000).
	maxSpecialisations int

	// costBudget is the maximum total computation cost for a single execution. Zero means
	// cost metering is disabled.
	costBudget int64

	// verifierIterationLimit caps the number of dataflow iterations the bytecode verifier
	// may perform per function. Zero selects the verifier's built-in default.
	verifierIterationLimit int

	// maxArenaSizeBytes mirrors maxArenaBytes under the WithMaxArenaSizeBytes alias for
	// explicit limit naming. Zero preserves the existing arena budget setting.
	maxArenaSizeBytes uint64

	// maxExpressionDepth caps the recursion depth when compiling nested expressions. Zero
	// selects defaultMaxExpressionDepth (1024).
	maxExpressionDepth int

	// maxMethods caps the size of CompiledFunction.methodTable on the root function. Zero
	// selects defaultMaxMethods (10000).
	maxMethods int

	// maxArenaBytes caps the cumulative arena bytes per Execute.
	//
	// Zero selects the default budget (defaultMaxArenaBytes). The limit surfaces as
	// errArenaBudgetExceeded.
	maxArenaBytes uint64

	// maxConstantPoolSize caps each per-function constant pool (int, float, string, bool,
	// uint, complex, general, type, call-site). Zero selects defaultMaxConstantPoolSize
	// (65535).
	maxConstantPoolSize int

	// maxGoroutines is the maximum concurrent goroutines for interpreted code.
	maxGoroutines int32

	// yieldInterval is the number of instructions between runtime.Gosched() calls for
	// cooperative scheduling.
	yieldInterval uint32

	// features controls which Go language constructs are allowed during compilation. Zero
	// value means InterpFeaturesAll.
	features InterpFeature

	// bytecodeVerificationDisabled opts out of the post-compilation bytecode verifier.
	// Verification is on by default; set this via WithBytecodeVerification(false) to skip
	// it.
	bytecodeVerificationDisabled bool

	// debugInfo enables debug information generation during compilation.
	debugInfo bool

	// forceGoDispatch forces the pure Go dispatch loop on all architectures.
	forceGoDispatch bool

	// safeMode enables the runtime guard mode for untrusted code (the WithSafeMode option),
	// distinct from the "safe" build tag. When set, execution is routed onto the pure Go
	// dispatch loop so per-instruction guards can run; the ASM fast path is left untouched,
	// so default fast mode keeps zero added overhead.
	safeMode bool
}

// newServiceConfig returns a serviceConfig with its one-time guards initialised.
//
// The empty literal is kept separate from the field assignment so exhaustruct does not
// require every field to be set.
//
// Returns *serviceConfig which has its envOnce guard ready for use.
func newServiceConfig() *serviceConfig {
	config := &serviceConfig{}
	config.envOnce = new(sync.Once)
	return config
}

// Option configures the interpreter service.
type Option func(*serviceConfig)

// Service is the public entry point for the bytecode interpreter. It parses, type-checks,
// compiles, and executes Go source code via its Eval, Compile, Execute, CompileFileSet,
// ExecuteEntrypoint, ExecuteInits, CompileProgram, and EvalFile methods.
//
// Lifecycle: construct one Service per logical interpreter with NewService, configure it
// via Option functions, populate symbols with UseSymbols or UseSymbolProviders, then call
// any combination of the evaluation methods. Clone derives a sibling that shares symbols
// but owns independent execution state. Reset clears state for reuse.
//
// Thread-safety: a Service is safe for concurrent reads of its symbol registry, which is
// treated as immutable after setup. The remaining state (fileSet, globals, config,
// stderrWriter, limits) is not protected by locks and must not be mutated concurrently
// with an in-flight evaluation. Concurrent independent evaluations must use distinct
// Services (typically via Clone). The lastCostUsed counter is atomic so callers may read
// it from any goroutine.
//
// Allowed call sequence: NewService, optional Option-style mutators (SetStderr,
// UseSymbols, UseSymbolProviders, RegisterPackage, SetCompilationSnapshot), then any mix
// of Eval, Compile, Execute, EvalFile, CompileFileSet, ExecuteEntrypoint, ExecuteInits,
// and CompileProgram. LastCostUsed reports the cost consumed by the most recent
// execution.
type Service struct {
	// stderrWriter overrides the writer for print/println output; when nil, VMs fall back to
	// os.Stderr, and REPL and notebook hosts set this to capture output for display in their
	// own UI.
	stderrWriter io.Writer

	// fileSet is the go/token position table reused across every evaluation handled by this
	// Service.
	fileSet *token.FileSet

	// symbols holds pre-registered native symbols available to interpreted code via import.
	symbols *SymbolRegistry

	// globals holds package-level variables shared between successive evaluations.
	globals *globalStore

	// config holds optional service configuration assembled from Option functions passed to
	// NewService.
	config *serviceConfig

	// pendingVarBridges accumulates one entry per non-main interpreted package compiled by
	// Service.CompileProgram so the post-init snapshot path in
	// Service.finalisePendingVarBridges can register each package's exported-var values in
	// the symbol registry once Service.ExecuteInits has run.
	//
	// Cleared at the end of every successful ExecuteInits call.
	pendingVarBridges []pendingVarBridge

	// limits holds resource constraints threaded into each VM created by this Service.
	limits vmLimits

	// lastCostUsed holds the total computation cost consumed by the most recent execution,
	// read via LastCostUsed.
	lastCostUsed atomic.Int64

	// features controls which Go language constructs are allowed during compilation.
	features InterpFeature
}

// NewService creates a new interpreter service.
//
// Takes opts (Option variadic) which configure build tags, environment variables, and
// other interpreter behaviour.
//
// Returns *Service which is ready to evaluate code.
func NewService(opts ...Option) *Service {
	config := newServiceConfig()
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

// SetStderr overrides the writer for print/println output.
//
// Pass nil to restore the default (os.Stderr). The change applies to every subsequent VM
// created by this Service, including those spun up by Eval, ExecuteEntrypoint, and
// Session.Submit.
//
// Takes writer (io.Writer) which receives print/println output, or nil to reset.
func (s *Service) SetStderr(writer io.Writer) {
	s.stderrWriter = writer
}

// Eval evaluates Go source code and returns the result.
//
// The source can be a single expression, a statement, or a complete Go source file. For
// expressions, the expression's value is returned. For statements, nil is returned.
//
// Takes code (string) which contains the Go source code to evaluate.
//
// Returns any which is the result of evaluating the code.
// Returns error when parsing, type-checking, compilation, or execution fails.
func (s *Service) Eval(ctx context.Context, code string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, safeerror.NewError(safeMessageEvaluationCancelled, fmt.Errorf(errChainFmt, errExecutionCancelled, err))
	}
	if err := s.checkSourceSize(len(code)); err != nil {
		return nil, safeerror.NewError(safeMessageEvaluationFailed, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	s.applyEnvOverrides()

	_, err := parser.ParseExprFrom(s.fileSet, evalFileName, code, 0)
	if err == nil {
		result, evalErr := s.evalExpr(ctx, code)
		if evalErr != nil {
			return nil, safeerror.NewError(safeMessageEvaluationFailed, fmt.Errorf("evaluating expression: %w", evalErr))
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
		return nil, safeerror.NewError(safeMessageEvaluationFailed, fmt.Errorf("evaluating expression: %w", evalErr))
	}

	result, evalErr := s.evalMixed(ctx, code)
	if evalErr != nil {
		return nil, safeerror.NewError(safeMessageEvaluationFailed, fmt.Errorf("evaluating expression: %w", evalErr))
	}
	return result, nil
}

// Compile parses, type-checks, and compiles Go source code into a CompiledFunction
// without executing it. The returned function can be executed multiple times via Execute.
//
// Takes code (string) which contains the Go source code to compile.
//
// Returns *CompiledFunction which is the compiled representation.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) Compile(ctx context.Context, code string) (*CompiledFunction, error) {
	if err := ctx.Err(); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationCancelled, fmt.Errorf(errChainFmt, errExecutionCancelled, err))
	}
	if err := s.checkSourceSize(len(code)); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	_, err := parser.ParseExprFrom(s.fileSet, evalFileName, code, 0)
	if err == nil {
		result, compileErr := s.compileExpression(ctx, code)
		if compileErr != nil {
			return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf("compiling source: %w", compileErr))
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
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf("compiling source: %w", compileErr))
	}

	result, compileErr := s.compileMixed(ctx, code)
	if compileErr != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf("compiling source: %w", compileErr))
	}
	return result, nil
}

// Execute runs a pre-compiled function and returns its result.
//
// Takes compiledFunction (*CompiledFunction) which is the compiled function to run.
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
	s.applyVMOutputs(vm)
	s.attachDebuggerToVM(vm)
	result, err := vm.execute(compiledFunction)
	s.recordCost(vm)
	return result, err
}

// EvalFile parses a complete Go source file, compiles all declarations, and executes the
// named entrypoint function. This is a convenience wrapper around CompileFileSet +
// ExecuteEntrypoint for single-file use.
//
// The source must be a valid Go file with a package clause. The entrypoint must name a
// function declared in the file.
//
// Takes source (string) which is the complete Go source file.
// Takes entrypoint (string) which is the function name to execute.
//
// Returns any which is the entrypoint function's return value.
// Returns error when parsing, type-checking, compilation, or execution fails.
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

// funcDeclEntry pairs a function declaration with its compiled shell, used between the
// register and compile passes.
type funcDeclEntry struct {
	// declaration is the parsed function declaration AST node.
	declaration *ast.FuncDecl

	// compiledFunction is the compiled function shell for this declaration.
	compiledFunction *CompiledFunction
}

// parsedPackage holds the parsed and filtered files for a single package within a
// multi-package compilation.
type parsedPackage struct {
	// importPath is the fully qualified import path for the package.
	importPath string

	// relPath is the relative path within the module.
	relPath string

	// packageName is the declared package name from the source files.
	packageName string

	// files holds the parsed and build-tag-filtered AST files.
	files []*ast.File
}

// CompileFileSet parses and type-checks multiple Go source files as a single package,
// returning a CompiledFileSet that can be executed multiple times via ExecuteEntrypoint.
//
// Takes sources (map[string]string) where keys are filenames (used for error reporting
// and deterministic ordering) and values are source code strings.
//
// Returns *CompiledFileSet which holds all compiled functions.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) CompileFileSet(ctx context.Context, sources map[string]string) (*CompiledFileSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationCancelled, fmt.Errorf(errChainFmt, errExecutionCancelled, err))
	}
	totalSourceBytes := 0
	for _, source := range sources {
		totalSourceBytes += len(source)
	}
	if err := s.checkSourceSize(totalSourceBytes); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	s.applyEnvOverrides()

	files, err := s.parseAndValidateFileSet(sources)
	if err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf(errFmtCompilingFileSet, err))
	}

	info := s.newTypesInfo()
	conf := s.newTypesConfig()
	if _, err := conf.Check(mainPackageName, s.fileSet, files, info); err != nil {
		enriched := s.enrichTypeCheckError(err, files, nil)
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf(errChainFmt, errTypeCheck, enriched))
	}

	rootFunction := &CompiledFunction{name: "<fileset>"}
	c := s.newFileSetCompiler(ctx, rootFunction, info)

	c.registerPackageLevelVarsFromFiles(ctx, files)

	if err := c.twoPassCompileFuncs(ctx, files, ""); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf(errFmtCompilingFileSet, err))
	}

	variableInitialisationFunction, err := c.compileVariableInitFunction(ctx, files)
	if err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf(errFmtCompilingFileSet, err))
	}

	if err := s.runPostCompilationChecks(ctx, rootFunction, errFmtCompilingFileSet); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, err)
	}

	entrypoints := make(map[string]uint16, len(c.funcTable))
	maps.Copy(entrypoints, c.funcTable)

	pretouchCompileTimeDerivedState(rootFunction)

	return &CompiledFileSet{
		root:                 rootFunction,
		entrypoints:          entrypoints,
		initFunctionIndices:  c.initFunctionIndices,
		variableInitFunction: variableInitialisationFunction,
	}, nil
}

// registerPackageLevelVarsFromFiles scans all files for package-level var declarations
// and registers them in the compiler.
//
// Takes files ([]*ast.File) which are the parsed AST files to scan.
func (c *compiler) registerPackageLevelVarsFromFiles(ctx context.Context, files []*ast.File) {
	for _, file := range files {
		c.registerPackageLevelVarsFromDecls(ctx, file.Decls)
	}
}

// registerPackageLevelVarsFromDecls scans declarations for package- level var specs and
// registers them in the compiler.
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

// twoPassCompileFuncs performs a two-pass compilation: first register all function
// declarations across files so cross-file references resolve, then compile all function
// bodies.
//
// Takes files ([]*ast.File) which are the parsed AST files to compile.
// Takes packageLabel (string) which is included in error messages for multi-package
// compilations and is empty for single-package ones.
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

// wrapCompileError wraps an error with errCompilation and an optional package label for
// multi-package compilations.
//
// Takes err (error) which is the original compilation error.
// Takes packageLabel (string) which is the package label for error context, or empty for
// single-package compilations.
//
// Returns error wrapped with errCompilation and the package label.
func (*compiler) wrapCompileError(_ context.Context, err error, packageLabel string) error {
	if packageLabel != "" {
		return fmt.Errorf(errChainMessageFmt, errCompilation, packageLabel, err)
	}
	return fmt.Errorf(errChainFmt, errCompilation, err)
}

// compileVariableInitFunction compiles package-level variable initialisers from all files
// into a dedicated function.
//
// Takes files ([]*ast.File) which are the parsed AST files with variable declarations.
//
// Returns *CompiledFunction which is the initialiser function, or nil when no global
// variables exist.
// Returns error when compilation of any initialiser fails.
func (c *compiler) compileVariableInitFunction(ctx context.Context, files []*ast.File) (*CompiledFunction, error) {
	if len(c.globalVariables) == 0 {
		return nil, nil
	}

	initFunction := &CompiledFunction{name: "<varinit>"}
	savedFunction := c.function
	c.function = initFunction
	c.scopes.pushScope()

	err := c.compileVarInitSpecs(ctx, files)

	c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DrillTier3), uint8(subOpTier3ReturnVoid))
	if overflowErr := c.resourceError(); overflowErr != nil {
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

// compileVarInitSpecs compiles each package-level variable initialiser in Go's required
// dependency order.
//
// Go's spec mandates that package vars be initialised so each initialiser sees the
// fully-initialised values of every var it depends on, regardless of file order. go/types
// computes the correct order during type-checking and exposes it as types.Info.InitOrder;
// we use it directly here.
//
// When InitOrder is unavailable (eval mode, custom info object, ...) we fall back to
// file-declaration order, which resolves simple cases correctly; elaborate dependency
// orderings require the InitOrder path.
//
// Takes files ([]*ast.File) which are the parsed AST files. The fallback path walks
// these; the InitOrder path indexes into them by position.
//
// Returns error when any variable initialiser fails to compile.
func (c *compiler) compileVarInitSpecs(ctx context.Context, files []*ast.File) error {
	if c.info != nil && len(c.info.InitOrder) > 0 {
		if err := c.compileVarInitFromInitOrder(ctx, files); err == nil {
			return nil
		}
	}
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

// compileVarInitFromInitOrder walks types.Info.InitOrder and emits each initialiser by
// locating its source ast.ValueSpec via the first LHS variable's position.
//
// Building a position -> ValueSpec index over every file makes the per-initialiser lookup
// O(1) after one O(n) scan. The total cost is linear in the number of package-level var
// declarations.
//
// Takes files ([]*ast.File) which are the parsed AST files providing the value specs the
// initialiser references.
//
// Returns error when any initialiser fails to compile, when an initialiser references a
// position not present in the index (which would indicate a desynchronised go/types
// result), or when sticky compile errors surface from the resource accounting path.
func (c *compiler) compileVarInitFromInitOrder(ctx context.Context, files []*ast.File) error {
	specsByPos := indexValueSpecsByPosition(files)
	for _, initialiser := range c.info.InitOrder {
		if len(initialiser.Lhs) == 0 {
			continue
		}
		spec, ok := specsByPos[initialiser.Lhs[0].Pos()]
		if !ok {
			return fmt.Errorf("compiling variable init specs: no value spec for %s", initialiser.Lhs[0].Name())
		}
		watermark := c.scopes.alloc.snapshot()
		if err := c.compilePackageLevelVarInit(ctx, spec); err != nil {
			return fmt.Errorf("compiling variable init specs: %w", err)
		}
		c.scopes.restoreWatermark(watermark)
	}
	return nil
}

// compileVarDeclInit compiles variable initialisers from a single declaration. Non-var
// declarations are silently skipped.
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

// ExecuteEntrypoint runs a named function from a pre-compiled file set, executing init
// functions first (in source order) before the entrypoint.
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

	if err := s.runVariableInits(ctx, cfs); err != nil {
		return nil, err
	}

	for _, initIndex := range cfs.initFunctionIndices {
		if err := s.executeInitFunc(ctx, cfs.root, cfs.root.functions[initIndex]); err != nil {
			return nil, fmt.Errorf("executing entrypoint: %w", err)
		}
	}

	s.finalisePendingVarBridges()
	return s.runEntrypointFunction(ctx, cfs, cfs.root.functions[index])
}

// ExecuteInits runs variable initialisers and init functions from a pre-compiled file set
// without requiring a named entrypoint. This is useful when the compiled code only needs
// its init side-effects (such as registering functions into a global registry).
//
// Takes cfs (*CompiledFileSet) which is the compiled file set whose init functions are to
// execute.
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
		s.applyVMOutputs(vm)
		vm.functions = cfs.root.functions
		vm.rootFunction = cfs.root
		vm.ensureCallStack()
		defer vm.releaseArena()
		defer vm.finishWatcher()
		vm.pushFrame(cfs.variableInitFunction)
		if _, err := vm.runGuarded(0); err != nil {
			return fmt.Errorf(errVarinitFmt, err)
		}
		vm.globals.materialiseStrings(vm.arena)
	}

	for _, initIndex := range cfs.initFunctionIndices {
		function := cfs.root.functions[initIndex]
		if err := s.executeInitFunc(ctx, cfs.root, function); err != nil {
			return fmt.Errorf("executing init functions: %w", err)
		}
	}

	s.finalisePendingVarBridges()
	return nil
}

// CompileProgram compiles multiple packages from source, automatically resolving import
// dependencies and wiring cross-package calls via the symbol registry.
//
// Takes modulePath (string) which is the module path (e.g. "testpkg").
// Takes packages (map[string]map[string]string) which maps relative package paths to
// filename-to-source maps.
//
// Returns *CompiledFileSet which contains all compiled functions.
// Returns error when parsing, type-checking, or compilation fails.
func (s *Service) CompileProgram(ctx context.Context, modulePath string, packages map[string]map[string]string) (*CompiledFileSet, error) {
	if err := ctx.Err(); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationCancelled, fmt.Errorf(errChainFmt, errExecutionCancelled, err))
	}
	if err := s.checkSourceSize(countSourceBytes(packages)); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, err)
	}
	ctx, cancel := s.applyMaxExecutionTime(ctx)
	defer cancel()
	ctx, _ = logger_domain.From(ctx, log)

	s.applyEnvOverrides()

	parsed, err := s.parseAndValidateImports(modulePath, packages)
	if err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf(errFmtCompilingProgram, err))
	}

	order, err := topoSort(buildDependencyGraph(parsed))
	if err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf(errFmtCompilingProgram, err))
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
		return nil, safeerror.NewError(safeMessageCompilationFailed, fmt.Errorf(errFmtCompilingProgram, err))
	}

	if err := s.runPostCompilationChecks(ctx, rootFunction, errFmtCompilingProgram); err != nil {
		return nil, safeerror.NewError(safeMessageCompilationFailed, err)
	}

	pretouchCompileTimeDerivedState(rootFunction)

	return &CompiledFileSet{
		root:                rootFunction,
		entrypoints:         resolveEntrypoints(mainFuncTable, lastFuncTable),
		initFunctionIndices: allInitFuncs,
	}, nil
}

// packageCompileResult holds the output of compiling a single package within a
// multi-package compilation.
type packageCompileResult struct {
	// funcTable maps function names to their indices in the root function.
	funcTable map[string]uint16

	// info holds the type-checking results for the package.
	info *types.Info

	// typesPackage is the types.Package produced by type-checking.
	typesPackage *types.Package

	// rootFunction is the root compiled function containing all sub-functions.
	rootFunction *CompiledFunction

	// initFunctionIndices holds indices of init functions in source order.
	initFunctionIndices []uint16

	// exportedVars lists exported package-level vars and their slots.
	//
	// Pairs each exported var with the global-store slot the runtime stores its value in.
	// Cross-package readers (for example main reading `uuid.NameSpaceURL`) cannot see vars
	// at bridge time because var initialisers run inside Service.ExecuteInits only after
	// compilation completes, so slots are recorded here and the post-init values are
	// snapshotted into the symbol registry from Service.finalisePendingVarBridges.
	exportedVars []packageVarExport
}

// packageVarExport pairs an exported var's name with the global-store slot the compiler
// allocated for it, plus the var's Go type and the settable reflect.Value the symbol
// registry advertises for it.
//
// The advertised storage is allocated eagerly at bridge time so downstream packages can
// resolve the symbol during type-check and bake a captured reflect.Value into their
// constant pool. The storage starts at the type's zero value; the post-init finaliser
// rewrites it via reflect.Value.Set, and existing bytecode that captured the same storage
// location observes the update on its next opLoadGeneralConst read.
//
// This is what makes the basic case work in a single CompileProgram call (where lib is
// type-checked, compiled, then main is compiled against the registry before any init
// runs).
type packageVarExport struct {
	// reflectType is the Go reflect.Type the bridged value should carry, used to wrap
	// primitive-bank values (int, uint, ...) in the var's declared Go type rather than the
	// bank's raw int64.
	reflectType reflect.Type

	// storage is the settable reflect.Value advertised in the symbol registry. The post-init
	// finaliser writes the current global-store value into this address; any compiled
	// bytecode that captured this Value reads through the same backing pointer.
	storage reflect.Value

	// name is the exported identifier (without package qualifier).
	name string

	// slot is the global-store coordinates the var's value lives at.
	slot globalVariableInfo
}

// pendingVarBridge ties one package's exported-var slot list to its import path. The
// Service accumulates these during compilation and resolves them after
// Service.ExecuteInits runs the package's variable-init function.
type pendingVarBridge struct {
	// importPath identifies the symbol-registry package the bridged vars belong to.
	importPath string

	// vars carries the slot+type entries recorded for each exported var by
	// compileSinglePackage.
	vars []packageVarExport
}

// UseSymbols sets the pre-registered native symbols for import resolution, protecting the
// "unsafe" package from override.
//
// Takes symbols (*SymbolRegistry) which is the registry to use.
func (s *Service) UseSymbols(symbols *SymbolRegistry) {
	symbols.ProtectPackage(pkgUnsafe)
	s.symbols = symbols
	s.applyClockOverrides()
}

// UseSymbolProviders builds a SymbolRegistry from one or more symbol providers.
//
// Later providers override earlier ones for the same package/symbol. The "unsafe" package
// is always protected.
//
// Takes providers (SymbolProviderPort variadic) which are the symbol providers to
// compose.
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
	s.applyClockOverrides()
}

// RegisterPackage registers symbols under a package path in the symbol registry. This is
// useful for creating package aliases by registering the same symbol set under a shorter
// import path.
//
// Takes packagePath (string) which is the import path to register.
// Takes symbols (map[string]reflect.Value) which maps symbol names to their reflected
// values.
func (s *Service) RegisterPackage(packagePath string, symbols map[string]reflect.Value) {
	s.symbols.RegisterPackage(packagePath, symbols)
}

// HasRegisteredPackage reports whether the given import path is available in the symbol
// registry.
//
// Takes importPath (string) which is the full Go import path to check.
//
// Returns true if the package is already available via the symbol registry.
func (s *Service) HasRegisteredPackage(importPath string) bool {
	return s.symbols.HasPackage(importPath)
}

// Reset clears the interpreter state for reuse.
//
// The token.FileSet is replaced with a fresh one and all package-level globals are
// cleared, returning the global store to single-threaded mode. The symbol registry and
// configuration are preserved.
func (s *Service) Reset() {
	s.fileSet = token.NewFileSet()
	s.globals.reset()
}

// Clone creates a copy of the service sharing symbols but with fresh execution state.
//
// Returns *Service which is a new service with shared symbols and independent state.
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

// SetCompilationSnapshot sets a callback that receives a snapshot of the compiled output
// at the end of CompileProgram, regardless of whether compilation succeeded or failed.
// The Service's config is replaced with a private copy so the callback does not propagate
// to clones sharing the original config.
//
// Takes callback (func(*CompiledFileSet)) which receives the snapshot.
func (s *Service) SetCompilationSnapshot(callback func(*CompiledFileSet)) {
	if s.config == nil {
		s.config = newServiceConfig()
	} else {
		s.config = new(*s.config)
	}
	s.config.compilationSnapshotCallback = callback
}

// SetCapabilityHook installs or replaces the CapabilityHook on this Service. Pass nil to
// revert to the permissive default.
//
// The hook is consulted before every gated native operation (file open, network dial,
// process spawn, environment access, native function dispatch). Changes take effect for
// subsequent evaluations only; in-flight executions continue using the hook they were
// started with.
//
// This setter mirrors WithCapabilityHook for hosts (such as REPLs and CLIs) that resolve
// the hook policy after Service construction.
//
// Takes hook (CapabilityHook) which is the new hook, or nil to clear.
func (s *Service) SetCapabilityHook(hook CapabilityHook) {
	if s.config == nil {
		s.config = newServiceConfig()
	}
	s.config.capabilityHook = hook
	s.limits.capabilityHook = hook
}

// CapabilityHook returns the currently installed CapabilityHook, or nil when no hook has
// been configured. The returned value reflects the configured hook, not the permissive
// fallback used at dispatch time.
//
// Returns CapabilityHook which is the configured hook, or nil.
func (s *Service) CapabilityHook() CapabilityHook {
	if s.config == nil {
		return nil
	}
	return s.config.capabilityHook
}

// LastCostUsed returns the total computation cost consumed by the most recent execution
// when cost metering is enabled.
//
// Returns int64 which is the cost consumed, or 0 when cost metering is disabled.
func (s *Service) LastCostUsed() int64 {
	return s.lastCostUsed.Load()
}

// applyClockOverrides replaces the stdlib time package's Now, Since, Until, Sleep,
// NewTimer, and NewTicker symbols with clock-dispatched versions when a non-default Clock
// has been configured. No-op when the configured clock is nil or WallClock (the default
// behaviour already matches stdlib).
//
// Called after every symbol-registry mutation (UseSymbols, UseSymbolProviders) so the
// override survives registry replacement.
func (s *Service) applyClockOverrides() {
	if s.config == nil || s.config.clock == nil {
		return
	}
	if s.config.clock == WallClock {
		return
	}
	overrides := clockOverrideSymbols(s.config.clock)
	if s.symbols == nil {
		s.symbols = NewSymbolRegistry(SymbolExports{"time": overrides})
		s.symbols.ProtectPackage(pkgUnsafe)
		return
	}
	s.symbols.OverlayPackage("time", overrides)
}

// applyVMOutputs propagates Service-level output settings to a freshly constructed VM.
// Called immediately after newVM in every code path that runs interpreted bytecode.
//
// Takes vm (*VM) which is the VM to configure.
func (s *Service) applyVMOutputs(vm *VM) {
	if s.stderrWriter != nil {
		vm.stderrWriter = s.stderrWriter
	}
}

// runVariableInits executes the variable-initialisation function of a compiled file set
// when present. No-op when the file set has no package-level variables.
//
// Takes ctx (context.Context) for cancellation.
// Takes cfs (*CompiledFileSet) which provides root functions and the optional variable
// initialiser.
//
// Returns error when the initialiser fails to execute.
func (s *Service) runVariableInits(ctx context.Context, cfs *CompiledFileSet) error {
	if cfs.variableInitFunction == nil {
		return nil
	}
	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.applyVMOutputs(vm)
	vm.functions = cfs.root.functions
	vm.rootFunction = cfs.root
	vm.ensureCallStack()
	defer vm.releaseArena()
	defer vm.finishWatcher()
	vm.pushFrame(cfs.variableInitFunction)
	if _, err := vm.runGuarded(0); err != nil {
		return fmt.Errorf(errVarinitFmt, err)
	}
	return nil
}

// runEntrypointFunction executes the resolved entrypoint via the dispatched runner,
// configuring the VM with the register arena and ASM call-info tables.
//
// Takes ctx (context.Context) which is observed for cancellation.
// Takes cfs (*CompiledFileSet) which provides root functions.
// Takes entrypointFunction (*CompiledFunction) which is the function to run.
//
// Returns any which is the entrypoint's return value.
// Returns error when execution fails or a goroutine panic surfaces.
//
// Panics when a recovered panic is not the arena budget error; the original panic is
// rethrown to preserve host-visible behaviour.
func (s *Service) runEntrypointFunction(
	ctx context.Context,
	cfs *CompiledFileSet,
	entrypointFunction *CompiledFunction,
) (result any, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if budgetErr, ok := recovered.(error); ok && errors.Is(budgetErr, errArenaBudgetExceeded) {
				result = nil
				err = budgetErr
				return
			}
			panic(recovered)
		}
	}()
	vm := newVM(ctx, s.globals, s.symbols)
	vm.limits = s.limits
	s.applyVMOutputs(vm)
	s.attachDebuggerToVM(vm)
	defer vm.finishWatcher()
	vm.functions = cfs.root.functions
	vm.rootFunction = cfs.root
	arena := vm.acquireArena()
	vm.arena = arena
	defer func() {
		vm.callStack = nil
		vm.asmCallInfoTables = nil
		vm.asmCallInfoBases = nil
		vm.asmDispatchSaves = nil
		PutRegisterArena(arena)
	}()
	vm.callStack = arena.frameStack()
	vm.sizeArenaFromFunctions(cfs.root)
	vm.asmCallInfoTables = ensureASMCallInfoTables(cfs.root)
	vm.asmCallInfoBases = arena.CallInfoBases()
	vm.asmDispatchSaves = arena.dispatchSaves()
	if table := vm.asmCallInfoTables[entrypointFunction]; len(table) > 0 {
		vm.asmCallInfoBases[0] = uintptr(unsafe.Pointer(&table[0]))
	}
	vm.pushFrame(entrypointFunction)
	result, err = vm.runDispatchedGuarded(0)
	result = materialiseAnyForArena(arena, result)
	for index, allResult := range vm.evalAllResults {
		vm.evalAllResults[index] = materialiseAnyForArena(arena, allResult)
	}
	if err == nil {
		if info := vm.globals.goroutinePanic.Load(); info != nil {
			err = fmt.Errorf("goroutine panicked: %v", info.value)
			result = nil
		}
	}
	return result, err
}

// buildLimits constructs vmLimits from the service configuration. Each call creates a
// fresh resourceTracker so that cloned services and separate evaluations do not share
// counters.
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
	limits.diagnostics = &fastPathDiagnostics{}
	return limits
}

// applyConfigLimits copies non-zero config values into the given vmLimits, enabling cost
// accounting and yield when configured.
//
// Takes limits (*vmLimits) which is the limits struct to populate from the service
// config.
func (s *Service) applyConfigLimits(limits *vmLimits) {
	limits.arenaFactory = s.config.arenaFactory
	limits.maxCallDepth = s.config.maxCallDepth
	limits.forceGoDispatch = s.config.forceGoDispatch
	limits.safeMode = s.config.safeMode
	if s.config.safeMode {
		limits.forceGoDispatch = true
	}
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
	if s.config.maxArenaBytes > 0 {
		limits.maxArenaBytes = s.config.maxArenaBytes
	}
	if s.config.capabilityHook != nil {
		limits.capabilityHook = s.config.capabilityHook
	}
}

// attachDebuggerToVM attaches the configured debugger (if any) to a VM.
//
// Takes vm (*VM) which is the virtual machine to attach the debugger to.
func (s *Service) attachDebuggerToVM(vm *VM) {
	if s.config != nil && s.config.debugger != nil {
		s.config.debugger.attachToVM(vm)
	}
}

// applyMaxExecutionTime wraps ctx with the configured maximum execution time, if set. The
// cancel function must be deferred by the caller.
//
// Returns context.Context with the deadline applied and context.CancelFunc that must be
// deferred.
func (s *Service) applyMaxExecutionTime(ctx context.Context) (context.Context, context.CancelFunc) {
	limit := defaultMaxExecutionTime
	if s.config != nil && s.config.maxExecutionTime > 0 {
		limit = s.config.maxExecutionTime
	}
	return context.WithTimeoutCause(ctx, limit, errExecutionCancelled)
}

// checkSourceSize returns an error if the total source code size exceeds the configured
// maximum.
//
// Takes totalBytes (int) which is the total source code size in bytes to validate.
//
// Returns error when the size exceeds the configured limit, or nil.
func (s *Service) checkSourceSize(totalBytes int) error {
	if s.config != nil && s.config.maxSourceSize > 0 && totalBytes > s.config.maxSourceSize {
		return fmt.Errorf("%w: %d bytes exceeds limit %d", errSourceSizeLimit, totalBytes, s.config.maxSourceSize)
	}
	return nil
}

// recordCost stores the cost consumed by a VM execution into the service for later
// retrieval via LastCostUsed. When cost metering is disabled (costBudget == 0) this is a
// no-op to avoid any overhead on the hot path.
//
// Takes vm (*VM) which is the virtual machine whose cost to record.
func (s *Service) recordCost(vm *VM) {
	if vm.limits.costBudget > 0 {
		s.lastCostUsed.Store(vm.limits.costBudget - vm.costRemaining)
	}
}

// pretouchCompileTimeDerivedState forces eager construction of every per-function
// structure that would otherwise be built lazily on the first ExecuteEntrypoint call.
// When a fresh Service is executed once with no warmup, any work deferred to "first
// execute" inflates the timed wall clock.
//
// Two pieces of state qualify: asmCallInfoTables on the root (built once per rootFunction
// under a sync.Once and safe to invoke any time after compilation finishes) and
// precomputedAllocCounts per CompiledFunction (pure data derived from numRegisters and
// constant-pool sizes, guarded by a lazy gate so the first Execute need not compute
// them). Both invocations are no-ops on a warm Service; the eager call adds
// O(numFunctions + numCallSites) work to CompileFileSet/CompileProgram in exchange for
// removing the same work from the first Execute.
//
// Takes rootFunction (*CompiledFunction) which is the fileset root produced by the
// compiler.
func pretouchCompileTimeDerivedState(rootFunction *CompiledFunction) {
	if rootFunction == nil {
		return
	}
	ensureASMCallInfoTables(rootFunction)
	rootFunction.ensurePrecomputedAllocCounts()
	for _, child := range rootFunction.functions {
		if child == nil {
			continue
		}
		child.ensurePrecomputedAllocCounts()
	}
}

// indexValueSpecsByPosition indexes every ast.ValueSpec from a declaration list by the
// source position of each of its names. types.Initializer.Lhs uses *types.Var (which
// carries a Pos), so this index lets us recover the originating spec from any LHS var.
//
// Takes files ([]*ast.File) which are the parsed AST files to walk.
//
// Returns map[token.Pos]*ast.ValueSpec keyed by every declared name's position.
func indexValueSpecsByPosition(files []*ast.File) map[token.Pos]*ast.ValueSpec {
	out := make(map[token.Pos]*ast.ValueSpec)
	for _, file := range files {
		for _, declaration := range file.Decls {
			indexValueSpecsFromDecl(declaration, out)
		}
	}
	return out
}

// indexValueSpecsFromDecl indexes value specs from a var declaration.
//
// Records every name position from a single top-level declaration into out, when the
// declaration is a `var` block. No-op for any other declaration kind.
//
// Takes declaration (ast.Decl) which is the top-level declaration to inspect.
// Takes out (map[token.Pos]*ast.ValueSpec) which receives the position-keyed spec
// entries.
func indexValueSpecsFromDecl(declaration ast.Decl, out map[token.Pos]*ast.ValueSpec) {
	genDecl, ok := declaration.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.VAR {
		return
	}
	for _, spec := range genDecl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range vs.Names {
			out[name.Pos()] = vs
		}
	}
}

// countSourceBytes totals the byte length across all source files in a multi-package
// compilation input.
//
// Takes packages (map[string]map[string]string) which maps package paths to
// filename-to-source maps.
//
// Returns int which is the total byte count across all source files.
func countSourceBytes(packages map[string]map[string]string) int {
	total := 0
	for _, files := range packages {
		for _, source := range files {
			total += len(source)
		}
	}
	return total
}
