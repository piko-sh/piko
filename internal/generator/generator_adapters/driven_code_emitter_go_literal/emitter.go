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
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultPrinterTabwidth is the tab width for the initial AST print. go/format.Source
	// re-canonicalises the output afterwards, so this only needs to produce valid Go.
	defaultPrinterTabwidth = 4

	// defaultFilePermissions is the file mode used when writing generated files. Uses 0640:
	// owner rw, group r, others none.
	defaultFilePermissions = 0640

	// defaultBufferCapacity is the initial capacity for pooled buffers. 32KB provides
	// headroom for larger generated files, avoiding repeated slice growth during formatting
	// while maintaining reasonable memory use.
	defaultBufferCapacity = 32 * 1024

	// pikoFacadePackagePath is the import path of the piko facade. Its types appear in every
	// signature this emitter writes, so the generated file always imports it under
	// facadePackageName whatever the user's script chose to call it.
	pikoFacadePackagePath = "piko.sh/piko"

	// buildASTDeclName is the function every generated component declares, built by
	// createBuildASTFunctionSignature.
	buildASTDeclName = "BuildAST"

	// customTagsDeclName is the package-level variable buildCustomTagsStaticVar declares.
	customTagsDeclName = "customTags"

	// fetcherNamePrefix begins the name of every generated collection fetcher function.
	fetcherNamePrefix = "fetchCollection"

	// tempVarNamePrefix begins the name of every generated temporary variable.
	tempVarNamePrefix = "tempVar"

	// staticNodeNamePrefix begins the name of every hoisted static node variable.
	staticNodeNamePrefix = "staticNode_"

	// staticAttrsNamePrefix begins the name of every hoisted static attribute variable.
	staticAttrsNamePrefix = "staticAttrs_"

	// loopIterNamePrefix begins the name of every extracted p-for collection variable.
	loopIterNamePrefix = "loopIter_"
)

var (
	// formatterBufferPool provides reusable buffers for go/printer output, significantly
	// reducing GC pressure during code generation.
	formatterBufferPool = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(make([]byte, 0, defaultBufferCapacity))
		},
	}

	// emitterImportAliases maps each package the emitter references to its qualifier.
	emitterImportAliases = map[string]string{
		"cmp":                       "",
		"fmt":                       "",
		"html":                      "",
		"strconv":                   "",
		"sort":                      "",
		"piko.sh/piko/wdk/runtime":  runtimePackageName,
		"piko.sh/piko/wdk/safeconv": "",
		pikoFacadePackagePath:       facadePackageName,
	}

	// emitterReservedQualifiers maps each qualifier the emitter's own references use onto
	// the package path it must resolve to.
	emitterReservedQualifiers = buildEmitterReservedQualifiers()

	// reservedGeneratedDeclNames holds the package-level names the emitter declares for
	// every component. A script declaring one of these produces a file with the name
	// declared twice, which does not compile.
	reservedGeneratedDeclNames = map[string]struct{}{
		buildASTDeclName:   {},
		customTagsDeclName: {},
	}

	// reservedGeneratedDeclPrefixes holds the prefixes of the counted names the emitter
	// generates. A name is reserved when a run of digits is all that follows the prefix.
	reservedGeneratedDeclPrefixes = []string{
		fetcherNamePrefix,
		tempVarNamePrefix,
		staticNodeNamePrefix,
		staticAttrsNamePrefix,
		loopIterNamePrefix,
	}
)

// buildEmitterReservedQualifiers inverts emitterImportAliases into the qualifier-keyed
// form the user script import check needs.
//
// Returns map[string]string which maps each reserved qualifier to its package path.
func buildEmitterReservedQualifiers() map[string]string {
	qualifiers := make(map[string]string, len(emitterImportAliases))
	for path, alias := range emitterImportAliases {
		qualifiers[resolvedQualifier(alias, path)] = path
	}
	return qualifiers
}

// Emitter provides a way to produce Go code from annotated source files. It implements
// CodeEmitterPort for use by the generator and coordinator.
type Emitter interface {
	// EmitCode generates output code from the given annotation result.
	//
	// Takes annotationResult (*annotator_dto.AnnotationResult) which contains the parsed
	// annotations to process.
	// Takes request (generator_dto.GenerateRequest) which specifies generation options.
	//
	// Returns []byte which contains the generated code.
	// Returns []*ast_domain.Diagnostic which contains any warnings or issues found.
	// Returns error when code generation fails.
	EmitCode(
		ctx context.Context,
		annotationResult *annotator_dto.AnnotationResult,
		request generator_dto.GenerateRequest,
	) ([]byte, []*ast_domain.Diagnostic, error)
}

// emitter holds the state for a single EmitCode operation. It implements CodeEmitterPort
// and manages temporary variable counters while delegating AST construction to
// specialised sub-emitters.
type emitter struct {
	// AnnotationResult holds the parsed annotation data used for code generation.
	AnnotationResult *annotator_dto.AnnotationResult

	// guardedKeys caches the conditionally-guarded invocation keys for the current
	// annotation result. It is computed once per emission and read by the hoist, loop, and
	// conditional render paths so the annotated tree is not re-walked at every scope.
	guardedKeys map[string]bool

	// ctx holds the state that changes during code generation.
	ctx *EmitterContext

	// astBuilder builds AST nodes for generated code.
	astBuilder *astBuilder

	// staticEmitter builds variable and init declarations for hoisted static nodes.
	staticEmitter *staticEmitter

	// prerenderer renders static nodes to HTML bytes at generation time. May be nil, in
	// which case prerendering is disabled.
	prerenderer generator_domain.StaticPrerenderer

	// config holds the code generation settings.
	config EmitterConfig
}

// EmitCode generates a Go source file from an annotation result. It is the main entry
// point for the emitter and orchestrates the entire code generation process, collecting
// any internal diagnostics along the way.
//
// Takes result (*annotator_dto.AnnotationResult) which contains the parsed annotations
// and metadata for code generation.
// Takes request (generator_dto.GenerateRequest) which specifies the generation settings
// including paths, package name, and virtual instances.
//
// Returns []byte which contains the formatted Go source code.
// Returns []*ast_domain.Diagnostic which contains any warnings or issues found during
// generation.
// Returns error when the main component validation fails, AST building fails, or code
// formatting fails.
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
		FormatGeneratedCode:       request.FormatGeneratedCode,
	}
	em.ctx = NewEmitterContext()
	em.AnnotationResult = result
	em.guardedKeys = nil

	em.registerEmitterImports()

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
	if fileAST == nil {
		return nil, allDiags, nil
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

// conditionallyGuardedKeys returns the conditionally-guarded invocation keys for the
// current annotation result, computing them once on first use and caching them for the
// emission. The annotated tree is immutable during emission, so the cached set is safe to
// reuse across the hoist, loop, and conditional render paths.
//
// Returns map[string]bool which is the set of conditionally-guarded invocation keys, or
// nil when no annotation result is present.
func (em *emitter) conditionallyGuardedKeys() map[string]bool {
	if em.guardedKeys == nil && em.AnnotationResult != nil {
		em.guardedKeys = collectConditionallyGuardedKeys(em.AnnotationResult.AnnotatedAST)
	}
	return em.guardedKeys
}

// validateMainComponent checks that the main component exists in the result.
//
// Takes hashedName (string) which identifies the component to find.
// Takes result (*annotator_dto.AnnotationResult) which contains the virtual module with
// component mappings.
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

// diagnosticError wraps a diagnostic as an error, implementing the error interface.
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
// Takes request (generator_dto.GenerateRequest) which provides the generation settings
// including the package name.
// Takes result (*annotator_dto.AnnotationResult) which provides the annotated components
// and custom tags.
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the root component for
// code generation.
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

	if diagnostic := em.checkUserScriptCollisions(result, mainComponent); diagnostic != nil {
		return nil, append(allDiags, diagnostic), nil
	}

	if err := em.addBoilerplateAndUserCode(fileAST, mainComponent); err != nil {
		return nil, allDiags, err
	}

	customTagsDecl, customTagsVarName := buildCustomTagsStaticVar(result.CustomTags)
	if customTagsDecl != nil {
		fileAST.Decls = append(fileAST.Decls, customTagsDecl)
	}
	em.ctx.customTagsVarName = customTagsVarName

	buildASTDiags := em.generateBuildASTFunction(ctx, request, result, fileAST)
	allDiags = append(allDiags, buildASTDiags...)

	em.addFetcherDeclarations(fileAST)

	err := em.addStaticAndInitFunctions(result, fileAST)
	if err != nil {
		return nil, allDiags, fmt.Errorf("adding static and init functions: %w", err)
	}

	if err := em.addImportBlock(result, mainComponent, fileAST); err != nil {
		return nil, allDiags, err
	}

	return fileAST, allDiags, nil
}

// addBoilerplateAndUserCode adds standard acknowledgements and user script code to the
// file.
//
// Takes fileAST (*goast.File) which is the target file to modify.
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the user code to
// copy.
//
// Returns error when the script declares a name the emitter generates itself.
func (em *emitter) addBoilerplateAndUserCode(
	fileAST *goast.File,
	mainComponent *annotator_dto.VirtualComponent,
) error {
	fileAST.Decls = append(fileAST.Decls, buildBoilerplateVarAcks()...)
	return copyUserCode(fileAST, mainComponent, em)
}

// generateBuildASTFunction creates the BuildAST function when an annotated AST exists.
//
// Takes request (generator_dto.GenerateRequest) which specifies the generation settings.
// Takes result (*annotator_dto.AnnotationResult) which contains the annotated AST data.
// Takes fileAST (*goast.File) which is the file to add the function to.
//
// Returns []*ast_domain.Diagnostic which contains any issues found during generation, or
// nil if AnnotatedAST is nil.
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

// addFetcherDeclarations adds the dynamic collection fetcher functions to the file.
//
// Takes fileAST (*goast.File) which receives the fetcher declarations.
func (em *emitter) addFetcherDeclarations(fileAST *goast.File) {
	fileAST.Decls = append(fileAST.Decls, em.ctx.fetcherDecls...)
}

// addImportBlock builds the import block and adds it to the start of the file.
//
// Takes result (*annotator_dto.AnnotationResult) which holds the annotation data.
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the main component being
// processed.
// Takes fileAST (*goast.File) which is the AST to add the import block to.
//
// Returns error when a script import binds a qualifier the emitter's own references use.
func (em *emitter) addImportBlock(
	result *annotator_dto.AnnotationResult,
	mainComponent *annotator_dto.VirtualComponent,
	fileAST *goast.File,
) error {
	importDecl, err := em.buildImportBlock(result, mainComponent, fileAST)
	if err != nil {
		return err
	}
	if importDecl != nil {
		fileAST.Decls = append([]goast.Decl{importDecl}, fileAST.Decls...)
	}
	return nil
}

// addStaticAndInitFunctions adds static declarations and a registration init function to
// the file AST.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotation data for
// building the registration function.
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

// addImport registers an import and handles alias conflicts. If the requested alias is
// already used by a different package, a unique alias is created (e.g., "dto_1", "dto_2")
// to prevent Go build errors.
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
		finalAlias = goastutil.DisambiguateIdentifier(alias, em.ctx.usedAliases)
	}

	em.ctx.requiredImports[canonicalPath] = finalAlias
	if finalAlias != "" {
		em.ctx.usedAliases[finalAlias] = canonicalPath
	}
}

// registerEmitterImports records the imports the emitter's own generated references need,
// before any import discovered while building the AST can claim their qualifiers.
//
// The facade is the one that has to be registered rather than only forced into the import
// block: the annotator lets a script import it under any alias it likes, and that alias
// is the one the script's own code uses. Registering the canonical qualifier here means
// the import block carries both, so "func BuildAST(r *piko.RequestData, ...)" resolves
// whatever the script called the package.
func (em *emitter) registerEmitterImports() {
	em.addImport(pikoFacadePackagePath, facadePackageName)
}

// getImportAlias returns the alias for a given package path. This means type expressions
// use the correct alias when import name conflicts have been resolved.
//
// Takes canonicalPath (string) which is the import path to look up.
//
// Returns string which is the alias for the package, or empty if not found.
func (em *emitter) getImportAlias(canonicalPath string) string {
	return em.ctx.requiredImports[canonicalPath]
}

// nextFetcherName generates a unique name for a collection fetcher function.
//
// This guarantees that multiple GetCollection calls in the same component produce
// uniquely named fetcher functions.
//
// Returns string which is a unique function name (e.g. "fetchCollection1").
func (em *emitter) nextFetcherName() string {
	em.ctx.fetcherCtr++
	return fetcherNamePrefix + strconv.FormatInt(em.ctx.fetcherCtr, 10)
}

// addFetcherDeclaration adds a fetcher function to the file's declarations. These
// functions are placed at file level, alongside the BuildAST function.
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

// appendStaticDeclarations adds the var() and init() blocks for hoisted static nodes to
// the file AST.
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

// formatAndVerify prints the AST to a byte slice, formats it with gofmt-style rules, and
// can check that the result is valid Go syntax.
//
// Takes request (generator_dto.GenerateRequest) which provides the source path and
// settings for checking the output.
// Takes fset (*token.FileSet) which holds position data for the AST.
// Takes fileAST (*goast.File) which is the AST to format and check.
//
// Returns []byte which contains the formatted Go source code.
// Returns error when formatting fails or the syntax check finds a problem.
//
// Uses a pooled buffer to reduce memory use. Set request.VerifyGeneratedCode to false to
// skip syntax checking for faster builds.
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

	formattedBytes := insertImportGroupBlankLine(buffer.Bytes())
	if em.config.FormatGeneratedCode {
		canonical, ferr := format.Source(formattedBytes)
		if ferr != nil {
			return nil, fmt.Errorf("canonicalising generated Go code for %s: %w", request.SourcePath, ferr)
		}
		formattedBytes = canonical
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
	c := em.ctx.tempVarCtr.Add(1)
	return tempVarNamePrefix + strconv.FormatInt(c, 10)
}

// nextStaticVarName creates a unique name for a static node variable.
//
// Returns string which is the generated variable name.
func (em *emitter) nextStaticVarName() string {
	c := em.ctx.staticVarCtr.Add(1)
	return staticNodeNamePrefix + strconv.FormatInt(c, 10)
}

// nextStaticAttrVarName returns a unique name for a static attribute slice variable.
//
// Returns string which is the generated variable name.
func (em *emitter) nextStaticAttrVarName() string {
	c := em.ctx.staticAttrVarCtr.Add(1)
	return staticAttrsNamePrefix + strconv.FormatInt(c, 10)
}

// nextLoopIterName returns a unique name for a loop variable. These names are used to
// store p-for collection values, which allows correct slice capacity calculation and
// prevents expressions from being evaluated twice.
//
// Returns string which is the generated loop variable name.
func (em *emitter) nextLoopIterName() string {
	c := em.ctx.loopIterCtr.Add(1)
	return loopIterNamePrefix + strconv.FormatInt(c, 10)
}

// buildImportBlock builds an import declaration block for the generated output.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotated components
// to gather imports from.
// Takes mainComponent (*annotator_dto.VirtualComponent) which supplies the hashed name
// used to find partial imports.
//
// Returns *goast.GenDecl which contains the merged import declaration, or nil if no
// imports are needed.
// Returns error when a script import binds a qualifier the emitter's own references use.
func (em *emitter) buildImportBlock(
	result *annotator_dto.AnnotationResult,
	mainComponent *annotator_dto.VirtualComponent,
	fileAST *goast.File,
) (*goast.GenDecl, error) {
	importSet := make(map[string]goast.Spec)

	addStdImports(importSet)
	addUserScriptImports(importSet, mainComponent)
	addPartialImports(importSet, result, mainComponent.HashedName)
	addPartialScriptImports(importSet, result, mainComponent.HashedName)

	specs := em.collectCandidateImportSpecs(importSet)
	kept := pruneUnreferencedSpecs(specs, collectUsedQualifiers(fileAST))
	if len(kept) == 0 {
		return nil, nil
	}
	return buildImportDeclFromSpecs(kept), nil
}

// checkUserScriptCollisions reports the first name a user script binds that the emitter
// generates for itself.
//
// Both collisions are reported as diagnostics rather than as errors, because the
// generator drops the diagnostics it was given whenever an error is also returned, and a
// user writing an ordinary Go declaration deserves the same formatted, located message
// the annotator gives them for a template mistake.
//
// Takes result (*annotator_dto.AnnotationResult) which supplies the invoked partials.
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the component being
// emitted.
//
// Returns *ast_domain.Diagnostic describing the first collision, or nil when none exist.
func (em *emitter) checkUserScriptCollisions(
	result *annotator_dto.AnnotationResult,
	mainComponent *annotator_dto.VirtualComponent,
) *ast_domain.Diagnostic {
	if diagnostic := checkReservedImportAliases(result, mainComponent, em); diagnostic != nil {
		return diagnostic
	}

	if mainComponent == nil || mainComponent.RewrittenScriptAST == nil {
		return nil
	}

	return checkReservedUserDeclNames(mainComponent, em, buildUserDeclLineMap(mainComponent, em))
}

// checkReservedImportAliases reports a reserved qualifier bound by any script whose
// imports are merged into the generated file.
//
// Takes result (*annotator_dto.AnnotationResult) which supplies the invoked partials.
// Takes mainComponent (*annotator_dto.VirtualComponent) which is the component being
// emitted.
// Takes em (*emitter) which supplies path computation for the reported location.
//
// Returns *ast_domain.Diagnostic describing the first colliding import, or nil when none
// collide.
func checkReservedImportAliases(
	result *annotator_dto.AnnotationResult,
	mainComponent *annotator_dto.VirtualComponent,
	em *emitter,
) *ast_domain.Diagnostic {
	if diagnostic := checkReservedUserImportAliases(mainComponent, em); diagnostic != nil {
		return diagnostic
	}

	if result == nil || result.VirtualModule == nil || mainComponent == nil {
		return nil
	}

	for _, invocation := range result.UniqueInvocations {
		if invocation.PartialHashedName == mainComponent.HashedName {
			continue
		}

		partial := result.VirtualModule.ComponentsByHash[invocation.PartialHashedName]
		if diagnostic := checkReservedUserImportAliases(partial, em); diagnostic != nil {
			return diagnostic
		}
	}

	return nil
}

// checkReservedUserImportAliases reports a script import that binds a qualifier the
// emitter's own generated references use.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the script to check.
// Takes em (*emitter) which supplies path computation for the reported location.
//
// Returns *ast_domain.Diagnostic describing the first colliding import, or nil when none
// collide.
func checkReservedUserImportAliases(comp *annotator_dto.VirtualComponent, em *emitter) *ast_domain.Diagnostic {
	if comp == nil || comp.RewrittenScriptAST == nil {
		return nil
	}

	importLines := buildUserImportLineMap(comp, em)

	for _, spec := range userImportSpecs(comp.RewrittenScriptAST) {
		qualifier := importSpecQualifier(spec)
		path := importSpecPath(spec)

		reservedPath, reserved := emitterReservedQualifiers[qualifier]
		if !reserved || reservedPath == path {
			continue
		}

		return ast_domain.NewDiagnostic(
			ast_domain.Error,
			fmt.Sprintf(
				"Script import %q uses the name %q, which piko's generated code uses for %q. "+
					"Give the import a different alias.",
				path, qualifier, reservedPath,
			),
			qualifier,
			ast_domain.Location{Line: importLines[qualifier], Column: 1},
			userCodeSourcePath(comp, em),
		)
	}

	return nil
}

// userImportSpecs lists the import specs of a script AST in source order.
//
// Takes file (*goast.File) which is the script AST to read.
//
// Returns []*goast.ImportSpec which are the script's imports.
func userImportSpecs(file *goast.File) []*goast.ImportSpec {
	var specs []*goast.ImportSpec
	for _, declaration := range file.Decls {
		genDecl, isGen := declaration.(*goast.GenDecl)
		if !isGen || genDecl.Tok != token.IMPORT {
			continue
		}
		for _, spec := range genDecl.Specs {
			if impSpec, ok := spec.(*goast.ImportSpec); ok {
				specs = append(specs, impSpec)
			}
		}
	}
	return specs
}

// buildUserImportLineMap maps each qualifier a script imports under to its .pk line.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the script AST and source
// location data.
// Takes em (*emitter) which is present when source locations are available at all.
//
// Returns map[string]int which maps qualifiers to line numbers, or nil when source
// location data is unavailable.
func buildUserImportLineMap(comp *annotator_dto.VirtualComponent, em *emitter) map[string]int {
	if em == nil || comp.Source == nil || comp.Source.Script == nil {
		return nil
	}

	script := comp.Source.Script
	if script.ScriptStartLocation.Line <= 0 || script.Fset == nil || script.AST == nil {
		return nil
	}

	startLine := script.ScriptStartLocation.Line
	result := make(map[string]int)

	for _, spec := range userImportSpecs(script.AST) {
		if !spec.Pos().IsValid() {
			continue
		}
		virtualLine := script.Fset.Position(spec.Pos()).Line
		result[importSpecQualifier(spec)] = startLine + virtualLine - 1
	}

	return result
}

// collectCandidateImportSpecs flattens importSet into the candidate import-spec slice.
//
// Takes importSet (map[string]goast.Spec) which holds one collected spec per path.
//
// Returns []*goast.ImportSpec which is the candidate set, with any synthesised specs.
func (em *emitter) collectCandidateImportSpecs(importSet map[string]goast.Spec) []*goast.ImportSpec {
	specs := make([]*goast.ImportSpec, 0, len(importSet)+len(em.ctx.requiredImports))
	for path, spec := range importSet {
		impSpec, ok := spec.(*goast.ImportSpec)
		if !ok {
			continue
		}
		specs = append(specs, impSpec)

		if reqAlias, ok := em.ctx.requiredImports[path]; ok {
			if importSpecQualifier(impSpec) != resolvedQualifier(reqAlias, path) {
				specs = append(specs, synthesisedImportSpec(path, reqAlias))
			}
		}
	}
	for path, alias := range em.ctx.requiredImports {
		if _, exists := importSet[path]; exists {
			continue
		}
		spec := &goast.ImportSpec{Path: strLit(path)}
		if alias != "" {
			spec.Name = cachedIdent(alias)
		}
		specs = append(specs, spec)
	}
	return specs
}

// pruneUnreferencedSpecs drops specs whose qualifier the generated code never references.
//
// This reproduces goimports' per-file unused-import removal. Blank and dot imports are
// kept unconditionally; they are never "unused". It reuses the input slice's array.
//
// Takes specs ([]*goast.ImportSpec) which is the candidate import set to filter.
// Takes used (map[string]struct{}) which is the set of qualifiers the code references.
//
// Returns []*goast.ImportSpec which is the retained subset of specs.
func pruneUnreferencedSpecs(specs []*goast.ImportSpec, used map[string]struct{}) []*goast.ImportSpec {
	kept := specs[:0]
	for _, spec := range specs {
		switch importSpecName(spec) {
		case "_", ".":
			kept = append(kept, spec)
			continue
		}
		if _, ok := used[importSpecQualifier(spec)]; ok {
			kept = append(kept, spec)
		}
	}
	return kept
}

// buildRegistrationInitFunction generates the init() function responsible for calling the
// central registry to make this component's functions discoverable at runtime.
//
// Takes result (*annotator_dto.AnnotationResult) which provides the annotation data
// containing component metadata and script configuration.
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
		statements = append(statements, buildPolicyRegisterCall(
			pkgPathLit,
			"RegisterCachePolicyFunc",
			"CachePolicy",
			mainComponent.Source.Script.CachePolicyFuncName,
		))
	}
	if mainComponent.Source.Script.HasMiddleware {
		statements = append(statements, createRegisterCall("RegisterMiddlewareFunc", mainComponent.Source.Script.MiddlewaresFuncName))
	}
	if mainComponent.Source.Script.HasSupportedLocales {
		statements = append(statements, createRegisterCall("RegisterSupportedLocalesFunc", mainComponent.Source.Script.SupportedLocalesFuncName))
	}
	if mainComponent.Source.Script.HasAuthPolicy {
		statements = append(statements, buildPolicyRegisterCall(
			pkgPathLit,
			"RegisterAuthPolicyFunc",
			"AuthPolicy",
			mainComponent.Source.Script.AuthPolicyFuncName,
		))
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
// Takes ctx (context.Context) which provides the base context for logging in pool
// initialisation paths.
//
// Returns Emitter which is ready to output Go code literals.
func NewEmitter(_ context.Context) Emitter {
	return &emitter{}
}

// NewEmitterWithPrerenderer creates a new emitter with a prerenderer for static HTML
// generation.
//
// Takes prerenderer (generator_domain.StaticPrerenderer) which renders static nodes to
// HTML bytes at generation time. May be nil to disable prerendering.
//
// Returns Emitter which is ready to output Go code literals with prerendering.
func NewEmitterWithPrerenderer(_ context.Context, prerenderer generator_domain.StaticPrerenderer) Emitter {
	return &emitter{
		prerenderer: prerenderer,
	}
}

// addStdImports adds the imports the emitter's own generated references need.
//
// Takes importSet (map[string]goast.Spec) which receives the import entries to add.
func addStdImports(importSet map[string]goast.Spec) {
	for path, alias := range emitterImportAliases {
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
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the rewritten
// script AST to extract imports from.
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

// addPartialImports adds an import statement for each unique partial used in the
// template.
//
// Takes importSet (map[string]goast.Spec) which collects the import specs to add.
// Takes result (*annotator_dto.AnnotationResult) which provides the partial calls and
// virtual module data.
// Takes currentComponentHash (string) which identifies the current component to skip
// self-imports.
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

// addPartialScriptImports adds Go imports from embedded partials' script blocks. This
// means when partial template code is inlined into a parent, any Go package imports used
// in the partial's template expressions are available.
//
// Takes importSet (map[string]goast.Spec) which collects the import specs to add.
// Takes result (*annotator_dto.AnnotationResult) which provides the partial calls and
// virtual module data.
// Takes currentComponentHash (string) which identifies the current component to skip
// self-imports.
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

// extractImportsFromAST collects import specs from a Go AST file and adds them to the
// import set.
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

// addImportSpecsToSet adds import specs to a set, skipping any that already exist.
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

// userCodeLineDirective holds a pending //line directive to be injected before a user
// declaration during post-processing.
type userCodeLineDirective struct {
	// declSignature is a unique prefix of the declaration line (e.g. "func Render(") used to
	// locate the declaration in the formatted output.
	declSignature string

	// directive is the full //line directive text (e.g. "//line pages/main.pk:37").
	directive string
}

// copyUserCode moves all declarations except imports from the user's script into the
// target AST. When source location data is available, it records //line directive
// metadata for post-processing by injectUserCodeLineDirectives.
//
// The line mapping uses the ORIGINAL script AST (Source.Script.AST) rather than
// RewrittenScriptAST because the rewriter's deepCopyASTFile discards the FileSet used for
// re-parsing, making RewrittenScriptAST positions unresolvable. Auto-generated
// declarations (default Render, CachePolicy) have Pos()=0 in the original AST and are
// correctly excluded from //line emission.
//
// Takes fileAST (*goast.File) which is the target AST to add declarations to.
// Takes mainComponent (*annotator_dto.VirtualComponent) which provides the rewritten
// script with its declarations to copy.
// Takes em (*emitter) which provides path computation for //line directives.
//
// Returns error when a declaration cannot be copied.
func copyUserCode(fileAST *goast.File, mainComponent *annotator_dto.VirtualComponent, em *emitter) error {
	if mainComponent == nil || mainComponent.RewrittenScriptAST == nil {
		return nil
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

	return nil
}

// checkReservedUserDeclNames reports a script declaration that collides with a name the
// emitter generates for the same file.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the script to check.
// Takes em (*emitter) which supplies path computation for the reported location.
// Takes userDeclLines (map[string]int) which maps declaration names to .pk line numbers,
// or nil when source locations are unavailable.
//
// Returns *ast_domain.Diagnostic describing the first colliding declaration, or nil when
// none collide.
func checkReservedUserDeclNames(
	comp *annotator_dto.VirtualComponent,
	em *emitter,
	userDeclLines map[string]int,
) *ast_domain.Diagnostic {
	for _, name := range userDeclaredNames(comp.RewrittenScriptAST) {
		if !isReservedGeneratedName(name) {
			continue
		}
		return ast_domain.NewDiagnostic(
			ast_domain.Error,
			fmt.Sprintf(
				"Script declares %q, which piko generates for this component. "+
					"Rename the declaration.",
				name,
			),
			name,
			ast_domain.Location{Line: userDeclLines[name], Column: 1},
			userCodeSourcePath(comp, em),
		)
	}

	return nil
}

// userDeclaredNames lists the package-level names a script declares.
//
// Methods are excluded: their names live on their receiver, not in the file scope, so a
// method may share a name with a generated function. Imports are excluded because the
// emitter builds the import block itself.
//
// Takes file (*goast.File) which is the script AST to read.
//
// Returns []string which are the declared names in source order.
func userDeclaredNames(file *goast.File) []string {
	if file == nil {
		return nil
	}

	names := make([]string, 0, len(file.Decls))
	for _, declaration := range file.Decls {
		switch decl := declaration.(type) {
		case *goast.FuncDecl:
			if decl.Recv == nil && decl.Name != nil {
				names = append(names, decl.Name.Name)
			}
		case *goast.GenDecl:
			if decl.Tok == token.IMPORT {
				continue
			}
			names = append(names, genDeclNames(decl)...)
		}
	}
	return names
}

// genDeclNames lists the names a const, var or type declaration binds.
//
// Takes decl (*goast.GenDecl) which is the declaration to read.
//
// Returns []string which are the bound names in source order.
func genDeclNames(decl *goast.GenDecl) []string {
	var names []string
	for _, spec := range decl.Specs {
		switch typed := spec.(type) {
		case *goast.TypeSpec:
			if typed.Name != nil {
				names = append(names, typed.Name.Name)
			}
		case *goast.ValueSpec:
			for _, ident := range typed.Names {
				names = append(names, ident.Name)
			}
		}
	}
	return names
}

// isReservedGeneratedName reports whether a name is one the emitter generates itself.
//
// Takes name (string) which is the declared name to test.
//
// Returns bool which is true when the emitter may generate that name.
func isReservedGeneratedName(name string) bool {
	if _, reserved := reservedGeneratedDeclNames[name]; reserved {
		return true
	}
	for _, prefix := range reservedGeneratedDeclPrefixes {
		suffix, found := strings.CutPrefix(name, prefix)
		if found && isAllDigits(suffix) {
			return true
		}
	}
	return false
}

// isAllDigits reports whether a string is one or more ASCII digits.
//
// Takes text (string) which is the string to test.
//
// Returns bool which is true when the string is a non-empty run of digits.
func isAllDigits(text string) bool {
	if text == "" {
		return false
	}
	for index := range len(text) {
		if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return true
}

// userCodeSourcePath returns the .pk path a diagnostic about user script code names.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the source path.
// Takes em (*emitter) which supplies the base directory the path is made relative to. May
// be nil, in which case the path is reported as stored.
//
// Returns string which is the path, relative to the project root where one is known.
func userCodeSourcePath(comp *annotator_dto.VirtualComponent, em *emitter) string {
	sourcePath := ""
	if comp != nil && comp.Source != nil {
		sourcePath = comp.Source.SourcePath
	}
	if em == nil {
		return sourcePath
	}
	if sourcePath == "" {
		sourcePath = em.config.SourcePath
	}

	return em.computeRelativePath(sourcePath)
}

// buildUserDeclLineMap builds a map from user-defined declaration names to their absolute
// line numbers in the .pk file.
//
// It uses the ORIGINAL script AST (Source.Script.AST) with its FileSet, since
// RewrittenScriptAST positions come from a discarded FileSet created during deep copy.
// Auto-generated declarations (Pos()=0) are excluded.
//
// Takes comp (*annotator_dto.VirtualComponent) which provides the script AST and source
// location data.
// Takes em (*emitter) which supplies path computation for line directives.
//
// Returns map[string]int which maps declaration names to their line numbers, or nil when
// source location data is unavailable.
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

// declNameAndSignature extracts the name and a unique line prefix signature from a Go
// declaration.
//
// The signature is used to locate the declaration line in formatted output for //line
// directive injection.
//
// Takes decl (goast.Decl) which is the declaration to extract the name and signature
// from.
//
// Returns name (string) which is the declaration's identifier.
// Returns sig (string) which is a unique prefix of the declaration line, or empty if not
// extractable.
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

// injectUserCodeLineDirectives inserts //line directives before user-authored
// declarations in the formatted output.
//
// Each directive is placed on its own line immediately before the line containing the
// declaration signature.
//
// Takes src ([]byte) which is the formatted Go source code.
// Takes directives ([]userCodeLineDirective) which lists the directives to inject before
// their matching declarations.
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
		for i, directive := range slices.Backward(directives) {
			if bytes.HasPrefix(trimmed, []byte(directive.declSignature)) {
				result = append(result, directive.directive...)
				result = append(result, '\n')
				directives = slices.Delete(directives, i, i+1)
			}
		}

		result = append(result, line...)
	}

	return result
}

// dedentLineDirectives strips leading whitespace from //line directive lines.
//
// The Go compiler only recognises //line directives that start at column 1. go/printer
// indents statements inside function bodies, so directives emitted as AST statements end
// up with leading tabs and are silently ignored. This post-processing step moves them
// back to column 1 so they appear in DWARF.
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
// Takes request (generator_dto.GenerateRequest) which provides the source path and
// generation settings.
// Takes generatedBytes ([]byte) which contains the generated Go code to check.
//
// Returns error when the generated code is not valid Go. The broken code is saved to a
// temporary file to help with debugging.
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

// buildPolicyRegisterCall generates the AST for registering a no-argument policy hook
// (CachePolicy, AuthPolicy) with a wrapper that adapts the user's function to the
// registry signature, which receives *RequestData.
//
// The user defines the hook as `func() piko.T`, but the registry expects
// `func(*RequestData) piko.T`. This wrapper bridges the two.
//
// Takes pkgPathLit (goast.Expr) which is the string literal for the package path.
// Takes registerFuncName (string) which is the runtime registration function to call.
// Takes policyTypeName (string) which is the facade type the hook returns.
// Takes policyFuncName (string) which is the name of the user's hook function.
//
// Returns goast.Stmt which is the registration call statement.
func buildPolicyRegisterCall(pkgPathLit goast.Expr, registerFuncName, policyTypeName, policyFuncName string) goast.Stmt {
	return &goast.ExprStmt{
		X: &goast.CallExpr{
			Fun: &goast.SelectorExpr{
				X:   cachedIdent(runtimePackageName),
				Sel: cachedIdent(registerFuncName),
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
									Sel: cachedIdent(policyTypeName),
								},
							}},
						},
					},
					Body: &goast.BlockStmt{
						List: []goast.Stmt{
							&goast.ReturnStmt{
								Results: []goast.Expr{
									&goast.CallExpr{
										Fun: cachedIdent(policyFuncName),
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

// buildSourcePathClientScriptMap creates a map from source file paths to whether those
// files have client scripts, ensuring proper event handler output for nodes that come
// from embedded partials, which may have their own client scripts even when the parent
// page does not.
//
// Takes result (*annotator_dto.AnnotationResult) which provides access to all components
// and their source paths.
//
// Returns map[string]bool which maps source paths to their client script status.
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
