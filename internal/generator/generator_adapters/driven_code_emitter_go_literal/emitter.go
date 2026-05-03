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

package driven_code_emitter_go_literal

import (
	"bytes"
	"context"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"golang.org/x/tools/imports"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultPrinterTabwidth is the tab width for formatting generated code.
	defaultPrinterTabwidth = 4

	// defaultFilePermissions is the file mode used when writing generated files.
	// Uses 0640: owner rw, group r, others none.
	defaultFilePermissions = 0640

	// defaultBufferCapacity is the initial capacity for pooled buffers.
	// 32KB provides headroom for larger generated files, avoiding repeated
	// slice growth during formatting while maintaining reasonable memory use.
	defaultBufferCapacity = 32 * 1024
)

// formatterBufferPool provides reusable buffers for go/printer output,
// significantly reducing GC pressure during code generation.
var formatterBufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, defaultBufferCapacity))
	},
}

// Emitter provides a way to produce Go code from annotated source files.
// It implements CodeEmitterPort for use by the generator and coordinator.
type Emitter interface {
	// EmitCode generates output code from the given annotation
	// result.
	//
	// Takes annotationResult (*annotator_dto.AnnotationResult)
	// which contains the parsed annotations to process.
	// Takes request (generator_dto.GenerateRequest) which
	// specifies generation options.
	//
	// Returns []byte which contains the generated code.
	// Returns []*ast_domain.Diagnostic which contains any
	// warnings or issues found.
	// Returns error when code generation fails.
	EmitCode(
		ctx context.Context,
		annotationResult *annotator_dto.AnnotationResult,
		request generator_dto.GenerateRequest,
	) ([]byte, []*ast_domain.Diagnostic, error)
}

// emitter holds the state for a single EmitCode operation. It implements
// CodeEmitterPort and manages temporary variable counters while delegating
// AST construction to specialised sub-emitters.
type emitter struct {
	// AnnotationResult holds the parsed annotation data used for code generation.
	AnnotationResult *annotator_dto.AnnotationResult

	// ctx holds the state that changes during code generation.
	ctx *EmitterContext

	// astBuilder builds AST nodes for generated code.
	astBuilder *astBuilder

	// staticEmitter builds variable and init declarations for hoisted static nodes.
	staticEmitter *staticEmitter

	// prerenderer renders static nodes to HTML bytes at generation time.
	// May be nil, in which case prerendering is disabled.
	prerenderer generator_domain.StaticPrerenderer

	// config holds the code generation settings.
	config EmitterConfig
}

// EmitCode generates a Go source file from an annotation result. It is the
// main entry point for the emitter and orchestrates the entire code generation
// process, collecting any internal diagnostics along the way.
//
// Takes result (*annotator_dto.AnnotationResult) which contains the parsed
// annotations and metadata for code generation.
// Takes request (generator_dto.GenerateRequest) which specifies the generation
// settings including paths, package name, and virtual instances.
//
// Returns []byte which contains the formatted Go source code.
// Returns []*ast_domain.Diagnostic which contains any warnings or issues found
// during generation.
// Returns error when the main component validation fails, AST building fails,
// or code formatting fails.
func (em *emitter) EmitCode(
	ctx context.Context,
	result *annotator_dto.AnnotationResult,
	request generator_dto.GenerateRequest,
) ([]byte, []*ast_domain.Diagnostic, error) {
	ctx, span, l := log.Span(ctx, "EmitCode", logger_domain.String("sourcePath", request.SourcePath))
	defer span.End()

	CodeEmissionCount.Add(ctx, 1)

	absBaseDir := request.BaseDir
	if absBaseDirResolved, err := filepath.Abs(request.BaseDir); err == nil {
		absBaseDir = absBaseDirResolved
	}

	em.config = EmitterConfig{
		VirtualInstances:          request.VirtualInstances,
		CanonicalGoPackagePath:    request.CanonicalGoPackagePath,
		BaseDir:                   absBaseDir,
		PackageName:               request.PackageName,
		SourcePath:                request.SourcePath,
		HashedName:                request.HashedName,
		ModuleName:                request.ModuleName,
		IsPage:                    request.IsPage,
		HasClientScript:           result.ClientScript != "",
		SourcePathHasClientScript: buildSourcePathClientScriptMap(result),
		EnablePrerendering:        request.EnablePrerendering,
		EnableStaticHoisting:      request.EnableStaticHoisting,
		StripHTMLComments:         request.StripHTMLComments,
		EnableDwarfLineDirectives: request.EnableDwarfLineDirectives,
	}
	em.ctx = NewEmitterContext()
	em.AnnotationResult = result

	em.resetState(ctx)
	defer em.cleanup()

	mainComponent, err := em.validateMainComponent(request.HashedName, result)
	if err != nil {
		return nil, nil, fmt.Errorf("validating main component %q: %w", request.HashedName, err)
	}

	fileSet := token.NewFileSet()
	fileAST, allDiags, err := em.buildFileAST(ctx, request, result, mainComponent)
	if err != nil {
		return nil, allDiags, fmt.Errorf("building file AST for %q: %w", request.SourcePath, err)
	}

	generatedBytes, err := em.formatAndVerify(request, fileSet, fileAST)
	if err != nil {
		CodeEmissionErrorCount.Add(ctx, 1)
		l.Error("Failed to format or verify generated code.", logger_domain.Error(err))
		return nil, allDiags, fmt.Errorf("formatting generated code for %q: %w", request.SourcePath, err)
	}

	l.Trace("Successfully generated Go code.", logger_domain.String("source", request.SourcePath))
	return generatedBytes, allDiags, nil
}

// validateMainComponent checks that the main component exists in the result.
//
// Takes hashedName (string) which identifies the component to find.
// Takes result (*annotator_dto.AnnotationResult) which contains the virtual
// module with component mappings.
//
// Returns *annotator_dto.VirtualComponent which is the found component.
// Returns error when no component matches the given hash.
func (*emitter) validateMainComponent(
	hashedName string,
	result *annotator_dto.AnnotationResult,
) (*annotator_dto.VirtualComponent, error) {
	mainComponent, ok := result.VirtualModule.ComponentsByHash[hashedName]
	if mainComponent == nil || !ok {
		diagnostic := ast_domain.NewDiagnostic(
			ast_domain.Error,
			"Internal Emitter Error: Virtual component for hash not found: "+hashedName,
			"emitter",
			ast_domain.Location{},
			"",
		)
		return nil, fmt.Errorf("virtual component for hash '%s' not found: %w", hashedName, &diagnosticError{diagnostic})
	}
	return mainComponent, nil
}

// diagnosticError wraps a diagnostic as an error, implementing the error
// interface.
type diagnosticError struct {
	// diagnostic holds the diagnostic details for this error.
	diagnostic *ast_domain.Diagnostic
}

// Error returns the diagnostic message, implementing the error interface.
//
// Returns string which contains the diagnostic message text.
func (e *diagnosticError) Error() string {
	return e.diagnostic.Message
}

// buildFileAST builds the complete Go file AST.
//
// Takes request (generator_dto.GenerateRequest) which provides the generation
// settings including the package name.
// Takes result (*annotator_dto.AnnotationResult) which provides the annotated
// components and custom tags.
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the root
// component for code generation.
//
// Returns *goast.File which is the built AST ready for rendering.
// Returns []*ast_domain.Diagnostic which contains any warnings or issues found.
// Returns error when static or init function generation fails.
func (em *emitter) buildFileAST(
	ctx context.Context,
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
	mainComponent *annotator_dto.VirtualComponent,
) (*goast.File, []*ast_domain.Diagnostic, error) {
	fileAST := &goast.File{
		Name:  cachedIdent(request.PackageName),
		Decls: []goast.Decl{},
	}

	allDiags := make([]*ast_domain.Diagnostic, 0, defaultDiagnosticCapacity)

	em.addBoilerplateAndUserCode(fileAST, mainComponent)

	customTagsDecl, customTagsVarName := buildCustomTagsStaticVar(result.CustomTags)
	if customTagsDecl != nil {
		fileAST.Decls = append(fileAST.Decls, customTagsDecl)
	}
	em.ctx.customTagsVarName = customTagsVarName

	buildASTDiags := em.generateBuildASTFunction(ctx, request, result, fileAST)
	allDiags = append(allDiags, buildASTDiags...)

	em.addFetcherDeclarations(fileAST)

	em.addImportBlock(result, mainComponent, fileAST)

	err := em.addStaticAndInitFunctions(result, fileAST)
	if err != nil {
		return nil, allDiags, fmt.Errorf("adding static and init functions: %w", err)
	}

	return fileAST, allDiags, nil
}

// addBoilerplateAndUserCode adds standard acknowledgements and user script
// code to the file.
//
// Takes fileAST (*goast.File) which is the target file to modify.
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the
// user code to copy.
func (em *emitter) addBoilerplateAndUserCode(fileAST *goast.File, mainComponent *annotator_dto.VirtualComponent) {
	fileAST.Decls = append(fileAST.Decls, buildBoilerplateVarAcks()...)
	copyUserCode(fileAST, mainComponent, em)
}

// generateBuildASTFunction creates the BuildAST function when an annotated AST
// exists.
//
// Takes request (generator_dto.GenerateRequest) which specifies the generation
// settings.
// Takes result (*annotator_dto.AnnotationResult) which contains the annotated
// AST data.
// Takes fileAST (*goast.File) which is the file to add the function to.
//
// Returns []*ast_domain.Diagnostic which contains any issues found during
// generation, or nil if AnnotatedAST is nil.
func (em *emitter) generateBuildASTFunction(
	ctx context.Context,
	request generator_dto.GenerateRequest,
	result *annotator_dto.AnnotationResult,
	fileAST *goast.File,
) []*ast_domain.Diagnostic {
	if result.AnnotatedAST == nil {
		return nil
	}

	buildASTFunc, funcDiags := em.astBuilder.buildASTFunction(ctx, request, result)
	fileAST.Decls = append(fileAST.Decls, buildASTFunc)
	return funcDiags
}

// addFetcherDeclarations adds the dynamic collection fetcher functions to the
// file.
//
// Takes fileAST (*goast.File) which receives the fetcher declarations.
func (em *emitter) addFetcherDeclarations(fileAST *goast.File) {
	fileAST.Decls = append(fileAST.Decls, em.ctx.fetcherDecls...)
}

// addImportBlock builds the import block and adds it to the start of the file.
//
// Takes result (*annotator_dto.AnnotationResult) which holds the annotation
// data.
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the main
// component being processed.
// Takes fileAST (*goast.File) which is the AST to add the import block to.
func (em *emitter) addImportBlock(
	result *annotator_dto.AnnotationResult,
	mainComponent *annotator_dto.VirtualComponent,
	fileAST *goast.File,
) {
	importDecl := em.buildImportBlock(result, mainComponent)
	if importDecl != nil {
		fileAST.Decls = append([]goast.Decl{importDecl}, fileAST.Decls...)
	}
}

// addStaticAndInitFunctions adds static declarations and a registration init
// function to the file AST.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotation
// data for building the registration function.
// Takes fileAST (*goast.File) which is the target file to add declarations to.
//
// Returns error when the registration init function cannot be built.
func (em *emitter) addStaticAndInitFunctions(result *annotator_dto.AnnotationResult, fileAST *goast.File) error {
	em.appendStaticDeclarations(fileAST)

	registrationInitFunc, err := em.buildRegistrationInitFunction(result)
	if err != nil {
		return fmt.Errorf("building registration init function: %w", err)
	}
	fileAST.Decls = append(fileAST.Decls, registrationInitFunc)
	return nil
}

// addImport registers an import and handles alias conflicts.
// If the requested alias is already used by a different package, a unique alias
// is created (e.g., "dto_1", "dto_2") to prevent Go build errors.
//
// Takes canonicalPath (string) which specifies the full import path to register.
// Takes alias (string) which specifies the preferred short name for the import.
func (em *emitter) addImport(canonicalPath, alias string) {
	if canonicalPath == "" {
		return
	}
	if canonicalPath == em.config.CanonicalGoPackagePath {
		return
	}

	if _, exists := em.ctx.requiredImports[canonicalPath]; exists {
		return
	}

	finalAlias := alias

	if existingPath, aliasUsed := em.ctx.usedAliases[alias]; aliasUsed && existingPath != canonicalPath {
		em.ctx.aliasCtr++
		finalAlias = fmt.Sprintf("%s_%d", alias, em.ctx.aliasCtr)

		for {
			if _, stillUsed := em.ctx.usedAliases[finalAlias]; !stillUsed {
				break
			}
			em.ctx.aliasCtr++
			finalAlias = fmt.Sprintf("%s_%d", alias, em.ctx.aliasCtr)
		}
	}

	em.ctx.requiredImports[canonicalPath] = finalAlias
	if finalAlias != "" {
		em.ctx.usedAliases[finalAlias] = canonicalPath
	}
}

// getImportAlias returns the alias for a given package path.
// This means type expressions use the correct alias when import name
// conflicts have been resolved.
//
// Takes canonicalPath (string) which is the import path to look up.
//
// Returns string which is the alias for the package, or empty if not found.
func (em *emitter) getImportAlias(canonicalPath string) string {
	return em.ctx.requiredImports[canonicalPath]
}

// nextFetcherName generates a unique name for a collection fetcher function.
//
// This guarantees that multiple GetCollection calls in the same component
// produce uniquely named fetcher functions.
//
// Returns string which is a unique function name (e.g. "fetchCollection1").
func (em *emitter) nextFetcherName() string {
	em.ctx.fetcherCtr++
	return fmt.Sprintf("fetchCollection%d", em.ctx.fetcherCtr)
}

// addFetcherDeclaration adds a fetcher function to the file's declarations.
// These functions are placed at file level, alongside the BuildAST function.
//
// Takes declaration (goast.Decl) which is the function declaration to add.
func (em *emitter) addFetcherDeclaration(declaration goast.Decl) {
	em.ctx.fetcherDecls = append(em.ctx.fetcherDecls, declaration)
}

// resetState clears the emitter and prepares it for new code output.
func (em *emitter) resetState(ctx context.Context) {
	em.astBuilder = getAstBuilder(ctx, em)
}

// cleanup returns all pooled emitters back to their pools.
func (em *emitter) cleanup() {
	if em.astBuilder != nil {
		putAstBuilder(em.astBuilder)
		em.astBuilder = nil
	}
}

// appendStaticDeclarations adds the var() and init() blocks for hoisted static
// nodes to the file AST.
//
// Takes fileAST (*goast.File) which receives the generated declarations.
func (em *emitter) appendStaticDeclarations(fileAST *goast.File) {
	if staticVarDecl := em.staticEmitter.buildDeclarations(); staticVarDecl != nil {
		fileAST.Decls = append(fileAST.Decls, staticVarDecl)
	}
	if initFunc := em.staticEmitter.buildInitFunction(); initFunc != nil {
		fileAST.Decls = append(fileAST.Decls, initFunc)
	}
}

// formatAndVerify prints the AST to a byte slice, formats it with gofmt-style
// rules, and can check that the result is valid Go syntax.
//
// Takes request (generator_dto.GenerateRequest) which provides the source path and
// settings for checking the output.
// Takes fset (*token.FileSet) which holds position data for the AST.
// Takes fileAST (*goast.File) which is the AST to format and check.
//
// Returns []byte which contains the formatted Go source code.
// Returns error when formatting fails or the syntax check finds a problem.
//
// Uses a pooled buffer to reduce memory use. Set request.VerifyGeneratedCode to
// false to skip syntax checking for faster builds.
func (em *emitter) formatAndVerify(request generator_dto.GenerateRequest, fset *token.FileSet, fileAST *goast.File) ([]byte, error) {
	buffer, ok := formatterBufferPool.Get().(*bytes.Buffer)
	if !ok {
		buffer = new(bytes.Buffer)
	}
	buffer.Reset()
	defer formatterBufferPool.Put(buffer)

	_, _ = buffer.WriteString(generator_dto.AnalysisBuildConstraint)
	_, _ = buffer.WriteString("/* Code generated by piko; DO NOT EDIT. */\n\n")
	printerConfig := printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: defaultPrinterTabwidth}

	if err := printerConfig.Fprint(buffer, fset, fileAST); err != nil {
		return nil, fmt.Errorf("failed to format generated Go code for %s: %w", request.SourcePath, err)
	}

	formattedBytes, err := imports.Process(request.SourcePath, buffer.Bytes(), nil)
	if err != nil {
		formattedBytes = make([]byte, buffer.Len())
		copy(formattedBytes, buffer.Bytes())
	}

	formattedBytes = injectUserCodeLineDirectives(formattedBytes, em.ctx.userCodeLineDirectives)
	formattedBytes = dedentLineDirectives(formattedBytes)

	if request.VerifyGeneratedCode {
		if err := verifyGeneratedCode(request, formattedBytes); err != nil {
			return nil, fmt.Errorf("verifying generated code for %q: %w", request.SourcePath, err)
		}
	}
	return formattedBytes, nil
}

// nextTempName generates a unique name for a temporary variable.
//
// Returns string which is the generated name.
func (em *emitter) nextTempName() string {
	c := atomic.AddInt64(&em.ctx.tempVarCtr, 1)
	return "tempVar" + strconv.FormatInt(c, 10)
}

// nextStaticVarName creates a unique name for a static node variable.
//
// Returns string which is the generated variable name.
func (em *emitter) nextStaticVarName() string {
	c := atomic.AddInt64(&em.ctx.staticVarCtr, 1)
	return "staticNode_" + strconv.FormatInt(c, 10)
}

// nextStaticAttrVarName returns a unique name for a static attribute slice
// variable.
//
// Returns string which is the generated variable name.
func (em *emitter) nextStaticAttrVarName() string {
	c := atomic.AddInt64(&em.ctx.staticAttrVarCtr, 1)
	return "staticAttrs_" + strconv.FormatInt(c, 10)
}

// nextLoopIterName returns a unique name for a loop variable. These names are
// used to store p-for collection values, which allows correct slice capacity
// calculation and prevents expressions from being evaluated twice.
//
// Returns string which is the generated loop variable name.
func (em *emitter) nextLoopIterName() string {
	c := atomic.AddInt64(&em.ctx.loopIterCtr, 1)
	return "loopIter_" + strconv.FormatInt(c, 10)
}

// buildImportBlock builds an import declaration block for the generated output.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotated
// components to gather imports from.
// Takes mainComponent (*annotator_dto.VirtualComponent) which supplies the
// hashed name used to find partial imports.
//
// Returns *goast.GenDecl which contains the merged import declaration, or nil
// if no imports are needed.
func (em *emitter) buildImportBlock(result *annotator_dto.AnnotationResult, mainComponent *annotator_dto.VirtualComponent) *goast.GenDecl {
	importSet := make(map[string]goast.Spec)

	addStdImports(importSet)
	addUserScriptImports(importSet, mainComponent)
	addPartialImports(importSet, result, mainComponent.HashedName)
	addPartialScriptImports(importSet, result, mainComponent.HashedName)

	for path, alias := range em.ctx.requiredImports {
		if _, exists := importSet[path]; !exists {
			spec := &goast.ImportSpec{Path: strLit(path)}
			if alias != "" {
				spec.Name = cachedIdent(alias)
			}
			importSet[path] = spec
		}
	}

	if len(importSet) == 0 {
		return nil
	}

	return buildImportDecl(importSet)
}

// buildRegistrationInitFunction generates the init() function responsible for
// calling the central registry to make this component's functions discoverable
// at runtime.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotation
// data containing component metadata and script configuration.
//
// Returns goast.Decl which is the generated init() function declaration.
// Returns error when the main component cannot be found in the result.
func (*emitter) buildRegistrationInitFunction(result *annotator_dto.AnnotationResult) (goast.Decl, error) {
	mainComponent, err := generator_domain.GetMainComponent(result)
	if err != nil {
		return nil, fmt.Errorf("getting main component for registration: %w", err)
	}

	var statements []goast.Stmt

	pkgPathLit := strLit(mainComponent.CanonicalGoPackagePath)

	createRegisterCall := func(functionName string, handlerName string) goast.Stmt {
		return &goast.ExprStmt{
			X: &goast.CallExpr{
				Fun: &goast.SelectorExpr{X: cachedIdent(runtimePackageName), Sel: cachedIdent(functionName)},
				Args: []goast.Expr{
					pkgPathLit,
					cachedIdent(handlerName),
				},
			},
		}
	}

	statements = append(statements, createRegisterCall("RegisterASTFunc", "BuildAST"))

	if mainComponent.Source.Script.HasCachePolicy {
		statements = append(statements, buildCachePolicyRegisterCall(
			pkgPathLit,
			mainComponent.Source.Script.CachePolicyFuncName,
		))
	}
	if mainComponent.Source.Script.HasMiddleware {
		statements = append(statements, createRegisterCall("RegisterMiddlewareFunc", mainComponent.Source.Script.MiddlewaresFuncName))
	}
	if mainComponent.Source.Script.HasSupportedLocales {
		statements = append(statements, createRegisterCall("RegisterSupportedLocalesFunc", mainComponent.Source.Script.SupportedLocalesFuncName))
	}
	if mainComponent.Source.Script.HasPreview {
		statements = append(statements, createRegisterCall("RegisterPreviewFunc", mainComponent.Source.Script.PreviewFuncName))
	}

	initFunc := &goast.FuncDecl{
		Name: cachedIdent("init"),
		Type: &goast.FuncType{Params: &goast.FieldList{}},
		Body: &goast.BlockStmt{List: statements},
	}

	return initFunc, nil
}

// NewEmitter creates a new emitter for Go code literals.
//
// Takes ctx (context.Context) which provides the base context for logging in
// pool initialisation paths.
//
// Returns Emitter which is ready to output Go code literals.
func NewEmitter(_ context.Context) Emitter {
	return &emitter{}
}

// NewEmitterWithPrerenderer creates a new emitter with a prerenderer
// for static HTML generation.
//
// Takes prerenderer (generator_domain.StaticPrerenderer) which
// renders static nodes to HTML bytes at generation time. May be
// nil to disable prerendering.
//
// Returns Emitter which is ready to output Go code literals with
// prerendering.
func NewEmitterWithPrerenderer(_ context.Context, prerenderer generator_domain.StaticPrerenderer) Emitter {
	return &emitter{
		prerenderer: prerenderer,
	}
}

// addStdImports adds the standard library imports needed by generated code.
//
// Takes importSet (map[string]goast.Spec) which receives the import
// entries to add.
func addStdImports(importSet map[string]goast.Spec) {
	stdImports := map[string]string{
		"cmp":                       "",
		"fmt":                       "",
		"html":                      "",
		"strconv":                   "",
		"sort":                      "",
		"piko.sh/piko/wdk/runtime":  runtimePackageName,
		"piko.sh/piko/wdk/safeconv": "",
	}
	for path, alias := range stdImports {
		spec := &goast.ImportSpec{Path: strLit(path)}
		if alias != "" {
			spec.Name = cachedIdent(alias)
		}
		importSet[path] = spec
	}
}

// addUserScriptImports adds imports from the user's script block to the set.
//
// Takes importSet (map[string]goast.Spec) which collects the import specs.
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the
// rewritten script AST to extract imports from.
func addUserScriptImports(importSet map[string]goast.Spec, mainComponent *annotator_dto.VirtualComponent) {
	if mainComponent == nil || mainComponent.RewrittenScriptAST == nil {
		return
	}
	for _, declaration := range mainComponent.RewrittenScriptAST.Decls {
		if impDecl, ok := declaration.(*goast.GenDecl); ok && impDecl.Tok == token.IMPORT {
			for _, spec := range impDecl.Specs {
				if impSpec, ok := spec.(*goast.ImportSpec); ok {
					path := strings.Trim(impSpec.Path.Value, `"`)
					importSet[path] = impSpec
				}
			}
		}
	}
}

// addPartialImports adds an import statement for each unique partial used in
// the template.
//
// Takes importSet (map[string]goast.Spec) which collects the import specs to
// add.
// Takes result (*annotator_dto.AnnotationResult) which provides the partial
// calls and virtual module data.
// Takes currentComponentHash (string) which identifies the current component
// to skip self-imports.
func addPartialImports(importSet map[string]goast.Spec, result *annotator_dto.AnnotationResult, currentComponentHash string) {
	for _, invocation := range result.UniqueInvocations {
		if invocation.PartialHashedName == currentComponentHash {
			continue
		}

		vc, ok := result.VirtualModule.ComponentsByHash[invocation.PartialHashedName]
		if !ok {
			continue
		}
		path := vc.CanonicalGoPackagePath
		spec := &goast.ImportSpec{
			Name: cachedIdent(vc.HashedName),
			Path: strLit(path),
		}
		importSet[path] = spec
	}
}

// addPartialScriptImports adds Go imports from embedded partials' script blocks.
// This means when partial template code is inlined into a parent,
// any Go package imports used in the partial's template expressions are available.
//
// Takes importSet (map[string]goast.Spec) which collects the import specs to add.
// Takes result (*annotator_dto.AnnotationResult) which provides the partial calls
// and virtual module data.
// Takes currentComponentHash (string) which identifies the current component to
// skip self-imports.
func addPartialScriptImports(importSet map[string]goast.Spec, result *annotator_dto.AnnotationResult, currentComponentHash string) {
	for _, invocation := range result.UniqueInvocations {
		if invocation.PartialHashedName == currentComponentHash {
			continue
		}

		vc := result.VirtualModule.ComponentsByHash[invocation.PartialHashedName]
		if vc == nil || vc.RewrittenScriptAST == nil {
			continue
		}

		extractImportsFromAST(importSet, vc.RewrittenScriptAST)
	}
}

// extractImportsFromAST collects import specs from a Go AST file and adds
// them to the import set.
//
// Takes importSet (map[string]goast.Spec) which collects the import specs.
// Takes file (*goast.File) which is the parsed Go file to extract imports from.
func extractImportsFromAST(importSet map[string]goast.Spec, file *goast.File) {
	for _, declaration := range file.Decls {
		impDecl, ok := declaration.(*goast.GenDecl)
		if !ok || impDecl.Tok != token.IMPORT {
			continue
		}

		addImportSpecsToSet(importSet, impDecl.Specs)
	}
}

// addImportSpecsToSet adds import specs to a set, skipping any that already
// exist.
//
// Takes importSet (map[string]goast.Spec) which collects the import specs.
// Takes specs ([]goast.Spec) which contains the import specs to add.
func addImportSpecsToSet(importSet map[string]goast.Spec, specs []goast.Spec) {
	for _, spec := range specs {
		impSpec, ok := spec.(*goast.ImportSpec)
		if !ok {
			continue
		}

		path := strings.Trim(impSpec.Path.Value, `"`)
		if _, exists := importSet[path]; exists {
			continue
		}

		importSet[path] = impSpec
	}
}

// buildImportDecl creates an import declaration from a set of import specs.
//
// Takes importSet (map[string]goast.Spec) which maps import paths to their
// spec values.
//
// Returns *goast.GenDecl which holds the import specs sorted by path.
func buildImportDecl(importSet map[string]goast.Spec) *goast.GenDecl {
	sortedPaths := make([]string, 0, len(importSet))
	for path := range importSet {
		sortedPaths = append(sortedPaths, path)
	}
	slices.Sort(sortedPaths)

	sortedSpecs := make([]goast.Spec, len(sortedPaths))
	for i, path := range sortedPaths {
		sortedSpecs[i] = importSet[path]
	}

	return &goast.GenDecl{
		Tok:    token.IMPORT,
		Lparen: 1,
		Specs:  sortedSpecs,
	}
}

// userCodeLineDirective holds a pending //line directive to be injected before
// a user declaration during post-processing.
type userCodeLineDirective struct {
	// declSignature is a unique prefix of the declaration line (e.g. "func Render(")
	// used to locate the declaration in the formatted output.
	declSignature string

	// directive is the full //line directive text (e.g. "//line pages/main.pk:37").
	directive string
}

// copyUserCode moves all declarations except imports from the user's script
// into the target AST. When source location data is available, it records
// //line directive metadata for post-processing by injectUserCodeLineDirectives.
//
// The line mapping uses the ORIGINAL script AST (Source.Script.AST) rather than
// RewrittenScriptAST because the rewriter's deepCopyASTFile discards the
// FileSet used for re-parsing, making RewrittenScriptAST positions unresolvable.
// Auto-generated declarations (default Render, CachePolicy) have Pos()=0 in the
// original AST and are correctly excluded from //line emission.
//
// Takes fileAST (*goast.File) which is the target AST to add declarations to.
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the
// rewritten script with its declarations to copy.
// Takes em (*emitter) which provides path computation for //line directives.
func copyUserCode(fileAST *goast.File, mainComponent *annotator_dto.VirtualComponent, em *emitter) {
	if mainComponent == nil || mainComponent.RewrittenScriptAST == nil {
		return
	}

	userDeclLines := buildUserDeclLineMap(mainComponent, em)

	for _, declaration := range mainComponent.RewrittenScriptAST.Decls {
		if genDecl, isGen := declaration.(*goast.GenDecl); isGen && genDecl.Tok == token.IMPORT {
			continue
		}

		if userDeclLines != nil {
			name, sig := declNameAndSignature(declaration)
			if pkLine, ok := userDeclLines[name]; ok && sig != "" {
				relPath := em.computeRelativePath(mainComponent.Source.SourcePath)
				em.ctx.userCodeLineDirectives = append(em.ctx.userCodeLineDirectives, userCodeLineDirective{
					declSignature: sig,
					directive:     em.formatLineDirective(relPath, pkLine, 0),
				})
			}
		}

		fileAST.Decls = append(fileAST.Decls, declaration)
	}
}

// buildUserDeclLineMap builds a map from user-defined declaration
// names to their absolute line numbers in the .pk file.
//
// It uses the ORIGINAL script AST (Source.Script.AST) with its
// FileSet, since RewrittenScriptAST positions come from a
// discarded FileSet created during deep copy. Auto-generated
// declarations (Pos()=0) are excluded.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides
// the script AST and source location data.
// Takes em (*emitter) which supplies path computation for line
// directives.
//
// Returns map[string]int which maps declaration names to their
// line numbers, or nil when source location data is unavailable.
func buildUserDeclLineMap(comp *annotator_dto.VirtualComponent, em *emitter) map[string]int {
	if em == nil || comp.Source == nil || comp.Source.Script == nil {
		return nil
	}

	script := comp.Source.Script
	if script.ScriptStartLocation.Line <= 0 || script.Fset == nil {
		return nil
	}

	startLine := script.ScriptStartLocation.Line
	result := make(map[string]int)

	for _, decl := range script.AST.Decls {
		if !decl.Pos().IsValid() {
			continue
		}
		if genDecl, isGen := decl.(*goast.GenDecl); isGen && genDecl.Tok == token.IMPORT {
			continue
		}

		virtualLine := script.Fset.Position(decl.Pos()).Line
		pkLine := startLine + virtualLine - 1
		name, _ := declNameAndSignature(decl)
		if name != "" {
			result[name] = pkLine
		}
	}

	return result
}

// declNameAndSignature extracts the name and a unique line
// prefix signature from a Go declaration.
//
// The signature is used to locate the declaration line in
// formatted output for //line directive injection.
//
// Takes decl (goast.Decl) which is the declaration to extract
// the name and signature from.
//
// Returns name (string) which is the declaration's identifier.
// Returns sig (string) which is a unique prefix of the
// declaration line, or empty if not extractable.
func declNameAndSignature(decl goast.Decl) (name string, sig string) {
	switch d := decl.(type) {
	case *goast.FuncDecl:
		return d.Name.Name, "func " + d.Name.Name + "("
	case *goast.GenDecl:
		if len(d.Specs) == 0 {
			return "", ""
		}
		switch s := d.Specs[0].(type) {
		case *goast.TypeSpec:
			return s.Name.Name, "type " + s.Name.Name + " "
		case *goast.ValueSpec:
			if len(s.Names) > 0 {
				return s.Names[0].Name, "var " + s.Names[0].Name + " "
			}
		}
	}
	return "", ""
}

// injectUserCodeLineDirectives inserts //line directives before
// user-authored declarations in the formatted output.
//
// Each directive is placed on its own line immediately before the
// line containing the declaration signature.
//
// Takes src ([]byte) which is the formatted Go source code.
// Takes directives ([]userCodeLineDirective) which lists the
// directives to inject before their matching declarations.
//
// Returns []byte which is the source with directives inserted.
func injectUserCodeLineDirectives(src []byte, directives []userCodeLineDirective) []byte {
	if len(directives) == 0 {
		return src
	}

	result := make([]byte, 0, len(src)+len(directives)*64)
	remaining := src

	for len(remaining) > 0 {
		newlineIdx := bytes.IndexByte(remaining, '\n')
		var line []byte
		if newlineIdx >= 0 {
			line = remaining[:newlineIdx+1]
			remaining = remaining[newlineIdx+1:]
		} else {
			line = remaining
			remaining = nil
		}

		trimmed := bytes.TrimSpace(line)
		for i := len(directives) - 1; i >= 0; i-- {
			if bytes.HasPrefix(trimmed, []byte(directives[i].declSignature)) {
				result = append(result, directives[i].directive...)
				result = append(result, '\n')
				directives = append(directives[:i], directives[i+1:]...)
			}
		}

		result = append(result, line...)
	}

	return result
}

// dedentLineDirectives strips leading whitespace from //line directive lines.
//
// The Go compiler only recognises //line directives that start at column 1.
// go/printer indents statements inside function bodies, so directives emitted
// as AST statements end up with leading tabs and are silently ignored. This
// post-processing step moves them back to column 1 so they appear in DWARF.
//
// Takes src ([]byte) which is the formatted Go source.
//
// Returns []byte with all //line directive lines dedented to column 1.
func dedentLineDirectives(src []byte) []byte {
	if !bytes.Contains(src, []byte("//line ")) {
		return src
	}

	result := make([]byte, 0, len(src))
	remaining := src

	for len(remaining) > 0 {
		newlineIdx := bytes.IndexByte(remaining, '\n')
		var line []byte
		if newlineIdx >= 0 {
			line = remaining[:newlineIdx+1]
			remaining = remaining[newlineIdx+1:]
		} else {
			line = remaining
			remaining = nil
		}

		trimmed := bytes.TrimLeft(line, " \t")
		if bytes.HasPrefix(trimmed, []byte("//line ")) {
			result = append(result, trimmed...)
		} else {
			result = append(result, line...)
		}
	}

	return result
}

// buildBoilerplateVarAcks emits blank-identifier declarations for helper packages.
//
// Returns []goast.Decl which holds the blank identifier declarations.
func buildBoilerplateVarAcks() []goast.Decl {
	type selectorAck struct {
		name   string
		symbol string
	}
	selectorAcks := []selectorAck{
		{"fmt", "Println"},
		{"html", "EscapeString"},
		{"strconv", "FormatInt"},
		{"sort", "Strings"},
		{runtimePackageName, "EvaluateTruthiness"},
		{pkgSafeconv, "IntToInt32"},
	}
	emptyString := &goast.BasicLit{Kind: token.STRING, Value: `""`}
	expressionAcks := []goast.Expr{
		&goast.CallExpr{
			Fun:  &goast.SelectorExpr{X: cachedIdent("cmp"), Sel: cachedIdent("Compare")},
			Args: []goast.Expr{emptyString, emptyString},
		},
		&goast.CompositeLit{
			Type: &goast.SelectorExpr{X: cachedIdent(facadePackageName), Sel: cachedIdent("Metadata")},
		},
	}

	acks := make([]goast.Decl, 0, len(selectorAcks)+len(expressionAcks))

	for _, ack := range selectorAcks {
		acks = append(acks, &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{&goast.ValueSpec{
				Names:  []*goast.Ident{cachedIdent("_")},
				Values: []goast.Expr{&goast.SelectorExpr{X: cachedIdent(ack.name), Sel: cachedIdent(ack.symbol)}},
			}},
		})
	}
	for _, expression := range expressionAcks {
		acks = append(acks, &goast.GenDecl{
			Tok: token.VAR,
			Specs: []goast.Spec{&goast.ValueSpec{
				Names:  []*goast.Ident{cachedIdent("_")},
				Values: []goast.Expr{expression},
			}},
		})
	}

	return acks
}

// verifyGeneratedCode checks that the output bytes are valid Go code.
//
// Takes request (generator_dto.GenerateRequest) which provides the source path
// and generation settings.
// Takes generatedBytes ([]byte) which contains the generated Go code to
// check.
//
// Returns error when the generated code is not valid Go. The broken code is
// saved to a temporary file to help with debugging.
func verifyGeneratedCode(request generator_dto.GenerateRequest, generatedBytes []byte) error {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, request.SourcePath, generatedBytes, parser.AllErrors)
	if err != nil {
		badFileName := "piko_bad_gen_" + filepath.Base(request.SourcePath) + ".go"
		badFilePath := filepath.Join(os.TempDir(), badFileName)

		tempSandbox, sandboxErr := safedisk.NewNoOpSandbox(os.TempDir(), safedisk.ModeReadWrite)
		if sandboxErr == nil {
			_ = tempSandbox.WriteFile(badFileName, generatedBytes, defaultFilePermissions)
			_ = tempSandbox.Close()
		}

		return fmt.Errorf(
			"internal emitter error: produced invalid Go code for %s (written to %s for debugging). Parser error: %w",
			request.SourcePath, badFilePath, err,
		)
	}
	return nil
}

// buildCachePolicyRegisterCall generates the AST for registering a cache
// policy function with a wrapper that adapts the user's no-argument function
// to the CachePolicyFunc signature (which receives *RequestData).
//
// The user defines CachePolicy as func() piko.CachePolicy, but the registry
// expects func(*RequestData) CachePolicy. This wrapper bridges the two.
//
// Takes pkgPathLit (goast.Expr) which is the string literal for the package
// path.
// Takes cachePolicyFuncName (string) which is the name of the user's cache
// policy function.
//
// Returns goast.Stmt which is the registration call statement.
func buildCachePolicyRegisterCall(pkgPathLit goast.Expr, cachePolicyFuncName string) goast.Stmt {
	return &goast.ExprStmt{
		X: &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   cachedIdent(runtimePackageName),
				Sel: cachedIdent("RegisterCachePolicyFunc"),
			},
			Args: []goast.Expr{
				pkgPathLit,
				&goast.FuncLit{
					Type: &goast.FuncType{
						Params: &goast.FieldList{
							List: []*goast.Field{{
								Names: []*goast.Ident{cachedIdent("_")},
								Type: &goast.StarExpr{
									X: &goast.SelectorExpr{
										X:   cachedIdent(facadePackageName),
										Sel: cachedIdent("RequestData"),
									},
								},
							}},
						},
						Results: &goast.FieldList{
							List: []*goast.Field{{
								Type: &goast.SelectorExpr{
									X:   cachedIdent(facadePackageName),
									Sel: cachedIdent("CachePolicy"),
								},
							}},
						},
					},
					Body: &goast.BlockStmt{
						List: []goast.Stmt{
							&goast.ReturnStmt{
								Results: []goast.Expr{
									&goast.CallExpr{
										Fun: cachedIdent(cachePolicyFuncName),
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// buildSourcePathClientScriptMap creates a map from source file paths to
// whether those files have client scripts, ensuring proper event handler output
// for nodes that come from embedded partials, which may have their own client
// scripts even when the parent page does not.
//
// Takes result (*annotator_dto.AnnotationResult) which provides access to all
// components and their source paths.
//
// Returns map[string]bool which maps source paths to their client script
// status.
func buildSourcePathClientScriptMap(result *annotator_dto.AnnotationResult) map[string]bool {
	if result == nil || result.VirtualModule == nil {
		return nil
	}

	m := make(map[string]bool, len(result.VirtualModule.ComponentsByHash))
	for _, vc := range result.VirtualModule.ComponentsByHash {
		if vc == nil || vc.Source == nil {
			continue
		}
		hasClientScript := vc.Source.ClientScript != ""
		m[vc.Source.SourcePath] = hasClientScript
	}
	return m
}
