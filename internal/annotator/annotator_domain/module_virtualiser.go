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

package annotator_domain

// Creates virtual Go modules from component templates by generating
// synthetic Go source files for type inspection. Transforms template
// scripts into valid Go code with proper package structures, enabling
// the Go type checker to analyse component types.

import (
	"context"
	"fmt"
	goast "go/ast"
	"go/printer"
	"go/token"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

const (
	// pikoFacadePath is the import path for the Piko facade package.
	pikoFacadePath = "piko.sh/piko"

	// pikoFacadeAlias is the import alias for the Piko facade package.
	pikoFacadeAlias = "piko"

	// renderFuncName is the name of the render function in components.
	renderFuncName = "Render"

	// cachePolicyFuncName is the name of the cache policy function.
	cachePolicyFuncName = "CachePolicy"

	// requestDataStruct is the name of the struct type that holds request data.
	requestDataStruct = "RequestData"

	// noPropsStruct is the struct name used when a component has no properties.
	noPropsStruct = "NoProps"

	// noResponseStruct is the type name for methods that return no response body.
	noResponseStruct = "NoResponse"

	// metadataStruct is the name of the Metadata struct type in generated code.
	metadataStruct = "Metadata"

	// cachePolicyStruct is the name of the cache policy settings struct type.
	cachePolicyStruct = "CachePolicy"

	// errorInterface is the name of the built-in error interface type.
	errorInterface = "error"

	// requestParamName is the HTTP request parameter name in handler functions.
	requestParamName = "r"

	// propsParamName is the name of the props parameter in render functions.
	propsParamName = "props"

	// sideEffectImportName is the blank identifier used for side-effect imports.
	sideEffectImportName = "_"
)

// dynamicParamRegex matches chi-style {paramName} patterns in filenames.
// Used to replace {slug} with actual values when generating manifest keys.
var dynamicParamRegex = regexp.MustCompile(`\{([a-zA-Z_][a-zA-Z0-9_]*)\}`)

// ModuleVirtualiser converts a Piko project structure into a virtual
// Go module.
type ModuleVirtualiser struct {
	// resolver finds and loads Go modules from import paths.
	resolver resolver_domain.ResolverPort

	// pathsConfig holds the path settings for the module virtualiser.
	pathsConfig AnnotatorPathsConfig
}

// virtualisationContext holds the state for a single Virtualise operation.
type virtualisationContext struct {
	// graph holds the parsed component dependency graph used during
	// virtualisation.
	graph *annotator_dto.ComponentGraph

	// originalGoFiles holds the original Go file contents before any changes.
	originalGoFiles map[string][]byte

	// virtualModule is the module being built during virtualisation.
	virtualModule *annotator_dto.VirtualModule

	// resolver provides path resolution for virtualisation tasks.
	resolver resolver_domain.ResolverPort

	// pathsConfig holds path settings used to find file paths.
	pathsConfig AnnotatorPathsConfig

	// entryPoints holds the entry points to check for virtualisation.
	entryPoints []annotator_dto.EntryPoint
}

// NewModuleVirtualiser creates a new ModuleVirtualiser with the given
// settings.
//
// Takes resolver (ResolverPort) which provides module resolution.
// Takes pathsConfig (AnnotatorPathsConfig) which specifies the path settings.
//
// Returns *ModuleVirtualiser which is ready for use.
func NewModuleVirtualiser(resolver resolver_domain.ResolverPort, pathsConfig AnnotatorPathsConfig) *ModuleVirtualiser {
	return &ModuleVirtualiser{
		resolver:    resolver,
		pathsConfig: pathsConfig,
	}
}

// Virtualise is the main entry point for this stage. It transforms a
// ComponentGraph into a VirtualModule.
//
// Takes graph (*annotator_dto.ComponentGraph) which contains the parsed
// component relationships.
// Takes originalGoFiles (map[string][]byte) which maps file paths to their
// original source content.
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the entry
// points for virtualisation.
//
// Returns *annotator_dto.VirtualModule which contains the virtualised module
// with source overlays and component mappings.
// Returns error when boilerplate injection, AST rewriting, or source assembly
// fails.
func (mv *ModuleVirtualiser) Virtualise(
	ctx context.Context,
	graph *annotator_dto.ComponentGraph,
	originalGoFiles map[string][]byte,
	entryPoints []annotator_dto.EntryPoint,
) (*annotator_dto.VirtualModule, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "ModuleVirtualiser.Virtualise")
	defer span.End()

	l.Internal("--- [VIRTUALISER START] ---")
	l.Internal("Received component graph.", logger_domain.Int("components", len(graph.Components)), logger_domain.Int("go_files", len(originalGoFiles)))

	vCtx := &virtualisationContext{
		graph:           graph,
		originalGoFiles: originalGoFiles,
		resolver:        mv.resolver,
		pathsConfig:     mv.pathsConfig,
		entryPoints:     entryPoints,
		virtualModule: &annotator_dto.VirtualModule{
			SourceOverlay:      make(map[string][]byte, len(originalGoFiles)+len(graph.Components)),
			ComponentsByGoPath: make(map[string]*annotator_dto.VirtualComponent, len(graph.Components)),
			ComponentsByHash:   make(map[string]*annotator_dto.VirtualComponent, len(graph.Components)),
			Graph:              graph,
		},
	}

	if err := vCtx.injectDefaultBoilerplateFuncs(ctx); err != nil {
		return nil, fmt.Errorf("injecting default boilerplate functions: %w", err)
	}
	vCtx.createVirtualComponents(ctx)
	if err := vCtx.rewriteAllScriptASTs(ctx); err != nil {
		return nil, fmt.Errorf("rewriting script ASTs: %w", err)
	}
	if err := vCtx.assembleSourceOverlay(ctx); err != nil {
		return nil, fmt.Errorf("assembling source overlay: %w", err)
	}

	l.Internal("--- [VIRTUALISER END] ---", logger_domain.Int("overlay_files", len(vCtx.virtualModule.SourceOverlay)))
	return vCtx.virtualModule, nil
}

// injectDefaultBoilerplateFuncs adds missing Render and CachePolicy functions
// to each component. If a component lacks either function, a safe default is
// inserted into its Go AST before type analysis runs.
//
// Takes ctx (context.Context) which carries cancellation and logging.
//
// Returns error when injection fails.
func (vc *virtualisationContext) injectDefaultBoilerplateFuncs(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	l.Internal("[VIRTUALISER-STEP 1/4] Injecting default boilerplate functions...")

	for _, parsedComp := range vc.graph.Components {
		if parsedComp.Script == nil {
			const defaultPackageName = "piko_default"
			parsedComp.Script = &annotator_dto.ParsedScript{
				PropsTypeExpression:        nil,
				RenderReturnTypeExpression: nil,
				AST:                        &goast.File{Name: goast.NewIdent(defaultPackageName)},
				Fset:                       token.NewFileSet(),
				ProvisionalGoPackagePath:   "",
				GoPackageName:              defaultPackageName,
				MiddlewaresFuncName:        "",
				CachePolicyFuncName:        "",
				SupportedLocalesFuncName:   "",
				ScriptStartLocation:        ast_domain.Location{Line: 0, Column: 0, Offset: 0},
				HasMiddleware:              false,
				HasCachePolicy:             false,
				HasSupportedLocales:        false,
			}
		}

		scriptAST := parsedComp.Script.AST

		dtoAlias := ensureImportAndGetAlias(scriptAST, pikoFacadePath, pikoFacadeAlias)

		if !hasFuncDecl(scriptAST, renderFuncName) {
			defaultRender := buildDefaultRenderDecl(dtoAlias, parsedComp.Script.PropsTypeExpression)
			scriptAST.Decls = append(scriptAST.Decls, defaultRender)
			l.Trace("Injected default Render function.", logger_domain.String("component", parsedComp.SourcePath))
		}

		if !hasFuncDecl(scriptAST, cachePolicyFuncName) {
			defaultCachePolicy := buildDefaultCachePolicyDecl(dtoAlias)
			scriptAST.Decls = append(scriptAST.Decls, defaultCachePolicy)
			l.Trace("Injected default CachePolicy function.", logger_domain.String("component", parsedComp.SourcePath))
		}
	}
	return nil
}

// createVirtualComponents builds virtual components from the parsed graph and
// converts their paths to a standard format.
func (vc *virtualisationContext) createVirtualComponents(ctx context.Context) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("[VIRTUALISER-STEP 2/4] Creating virtual components and canonicalising paths...")
	metadata := vc.collectEntryPointMetadata(ctx)

	moduleName := vc.resolver.GetModuleName()
	baseDir := vc.resolver.GetBaseDir()

	for hashedName, parsedComp := range vc.graph.Components {
		virtualComp := vc.buildVirtualComponent(ctx, hashedName, parsedComp, metadata, moduleName, baseDir)
		vc.virtualModule.ComponentsByHash[hashedName] = virtualComp
	}
}

// entryPointMetadata holds details about entry points found during
// code analysis.
type entryPointMetadata struct {
	// isPage maps file paths to true if they are page entry points.
	isPage map[string]bool

	// isPublic tracks which paths are marked as public entry points.
	isPublic map[string]bool

	// isEmail tracks which entry points are email components.
	isEmail map[string]bool

	// isPdf tracks which entry points are PDF components.
	isPdf map[string]bool

	// isE2EOnly tracks paths that are only used for end-to-end tests.
	isE2EOnly map[string]bool

	// isErrorPage tracks which entry points are error pages (e.g., !404.pk).
	isErrorPage map[string]bool

	// errorStatusCode maps file paths to the HTTP status code the
	// error page handles.
	errorStatusCode map[string]int

	// errorStatusCodeMin maps file paths to range lower bounds
	// (e.g., 400 for !400-499.pk).
	errorStatusCodeMin map[string]int

	// errorStatusCodeMax maps file paths to range upper bounds
	// (e.g., 499 for !400-499.pk).
	errorStatusCodeMax map[string]int

	// isCatchAllError tracks which error pages are catch-all (!error.pk).
	isCatchAllError map[string]bool

	// virtualInstances maps source file paths to their virtual page instances.
	virtualInstances map[string][]annotator_dto.VirtualPageInstance
}

// collectEntryPointMetadata gathers metadata for all entry points.
//
// Returns *entryPointMetadata which holds the flags and virtual page instances
// for each entry point path.
func (vc *virtualisationContext) collectEntryPointMetadata(ctx context.Context) *entryPointMetadata {
	baseDir := vc.resolver.GetBaseDir()
	meta := &entryPointMetadata{
		isPage:             make(map[string]bool),
		isPublic:           make(map[string]bool),
		isEmail:            make(map[string]bool),
		isPdf:              make(map[string]bool),
		isE2EOnly:          make(map[string]bool),
		isErrorPage:        make(map[string]bool),
		errorStatusCode:    make(map[string]int),
		errorStatusCodeMin: make(map[string]int),
		errorStatusCodeMax: make(map[string]int),
		isCatchAllError:    make(map[string]bool),
		virtualInstances:   make(map[string][]annotator_dto.VirtualPageInstance),
	}

	for _, ep := range vc.entryPoints {
		resolvedPath, err := vc.resolver.ResolvePKPath(ctx, ep.Path, "")
		if err != nil {
			continue
		}
		vc.recordEntryPointFlags(meta, resolvedPath, ep)
		vc.recordVirtualPageSource(meta, resolvedPath, ep, baseDir)
	}
	return meta
}

// recordEntryPointFlags stores the flags from an entry point into the metadata
// maps.
//
// Takes meta (*entryPointMetadata) which holds the flag maps to update.
// Takes resolvedPath (string) which is the key for the metadata maps.
// Takes ep (annotator_dto.EntryPoint) which provides the flag values.
func (*virtualisationContext) recordEntryPointFlags(meta *entryPointMetadata, resolvedPath string, ep annotator_dto.EntryPoint) {
	if ep.IsPage {
		meta.isPage[resolvedPath] = true
	}
	if ep.IsPublic {
		meta.isPublic[resolvedPath] = true
	}
	if ep.IsEmail {
		meta.isEmail[resolvedPath] = true
	}
	if ep.IsPdf {
		meta.isPdf[resolvedPath] = true
	}
	if ep.IsE2EOnly {
		meta.isE2EOnly[resolvedPath] = true
	}
	if ep.IsErrorPage {
		meta.isErrorPage[resolvedPath] = true
		meta.errorStatusCode[resolvedPath] = ep.ErrorStatusCode
		meta.errorStatusCodeMin[resolvedPath] = ep.ErrorStatusCodeMin
		meta.errorStatusCodeMax[resolvedPath] = ep.ErrorStatusCodeMax
		meta.isCatchAllError[resolvedPath] = ep.IsCatchAllError
	}
}

// recordVirtualPageSource stores virtual page source data for an entry point.
//
// Takes meta (*entryPointMetadata) which holds the metadata to update.
// Takes resolvedPath (string) which is the key for storing instances.
// Takes ep (annotator_dto.EntryPoint) which provides the virtual page source.
// Takes baseDir (string) which is the base folder for path resolution.
func (vc *virtualisationContext) recordVirtualPageSource(meta *entryPointMetadata, resolvedPath string, ep annotator_dto.EntryPoint, baseDir string) {
	if ep.VirtualPageSource == nil || ep.VirtualPageSource.InitialProps == nil {
		return
	}
	instance := vc.createVirtualInstance(ep, baseDir)
	meta.virtualInstances[resolvedPath] = append(meta.virtualInstances[resolvedPath], instance)
}

// buildVirtualComponent creates a virtual component from parsed
// component data.
//
// Takes hashedName (string) which is the unique name for the component.
// Takes parsedComp (*annotator_dto.ParsedComponent) which holds the parsed
// component data.
// Takes meta (*entryPointMetadata) which provides page, public, and email
// flags for the component.
// Takes moduleName (string) which is the Go module path.
// Takes baseDir (string) which is the base folder for output paths.
//
// Returns *annotator_dto.VirtualComponent which is the fully built virtual
// component ready for code generation.
func (vc *virtualisationContext) buildVirtualComponent(
	ctx context.Context,
	hashedName string,
	parsedComp *annotator_dto.ParsedComponent,
	meta *entryPointMetadata,
	moduleName, baseDir string,
) *annotator_dto.VirtualComponent {
	isPage := meta.isPage[parsedComp.SourcePath]
	isPublic := meta.isPublic[parsedComp.SourcePath]
	if parsedComp.VisibilityOverride != nil {
		isPublic = *parsedComp.VisibilityOverride
	}
	isEmail := meta.isEmail[parsedComp.SourcePath]
	isPdf := meta.isPdf[parsedComp.SourcePath]
	isE2EOnly := meta.isE2EOnly[parsedComp.SourcePath]

	isErrorPage := meta.isErrorPage[parsedComp.SourcePath]
	targetSubDir, partialName, partialSrc := vc.resolveComponentPaths(ctx, parsedComp, baseDir, isPage, isEmail, isPdf, isErrorPage)

	return &annotator_dto.VirtualComponent{
		Source:                 parsedComp,
		RewrittenScriptAST:     nil,
		HashedName:             hashedName,
		CanonicalGoPackagePath: filepath.ToSlash(filepath.Join(moduleName, targetSubDir, hashedName)),
		VirtualGoFilePath:      filepath.Join(baseDir, targetSubDir, hashedName, "generated.go"),
		PartialName:            partialName,
		PartialSrc:             partialSrc,
		IsPage:                 isPage,
		IsPublic:               isPublic,
		IsEmail:                isEmail,
		IsPdf:                  isPdf,
		IsE2EOnly:              isE2EOnly,
		IsErrorPage:            meta.isErrorPage[parsedComp.SourcePath],
		ErrorStatusCode:        meta.errorStatusCode[parsedComp.SourcePath],
		ErrorStatusCodeMin:     meta.errorStatusCodeMin[parsedComp.SourcePath],
		ErrorStatusCodeMax:     meta.errorStatusCodeMax[parsedComp.SourcePath],
		IsCatchAllError:        meta.isCatchAllError[parsedComp.SourcePath],
		VirtualInstances:       meta.virtualInstances[parsedComp.SourcePath],
	}
}

// resolveComponentPaths finds the target folder and partial paths for a
// component based on its type.
//
// Takes parsedComp (*annotator_dto.ParsedComponent) which is the parsed
// component to find paths for.
// Takes baseDir (string) which is the base folder for path resolution.
// Takes isPage (bool) which indicates if the component is a page.
// Takes isEmail (bool) which indicates if the component is an email.
// Takes isPdf (bool) which indicates if the component is a PDF.
// Takes isErrorPage (bool) which indicates if the component is an error page
// (e.g. !404.pk, !500.pk). Error pages live in the pages directory but are not
// routable pages, so they are compiled into the pages target folder.
//
// Returns targetSubDir (string) which is the target subfolder for output.
// Returns partialName (string) which is the partial template name.
// Returns partialSrc (string) which is the partial source path.
func (vc *virtualisationContext) resolveComponentPaths(
	ctx context.Context, parsedComp *annotator_dto.ParsedComponent,
	baseDir string, isPage, isEmail, isPdf, isErrorPage bool,
) (targetSubDir, partialName, partialSrc string) {
	if isEmail {
		return config.CompiledEmailsTargetDir, "", ""
	}
	if isPdf {
		return config.CompiledPdfsTargetDir, "", ""
	}
	if isPage || isErrorPage {
		return config.CompiledPagesTargetDir, "", ""
	}
	return vc.resolvePartialPaths(ctx, parsedComp.SourcePath, baseDir)
}

// resolvePartialPaths works out the target folder and partial paths from a
// source file path.
//
// Takes sourcePath (string) which is the full path to the partial source file.
// Takes baseDir (string) which is the base folder for path working out.
//
// Returns targetSubDir (string) which is the compiled partials target folder.
// Returns partialName (string) which is the partial name without extension.
// Returns partialSrc (string) which is the full serve path for the partial.
func (vc *virtualisationContext) resolvePartialPaths(ctx context.Context, sourcePath, baseDir string) (targetSubDir, partialName, partialSrc string) {
	targetSubDir = config.CompiledPartialsTargetDir
	relToPartialsDir := vc.calculateRelativePartialPath(ctx, sourcePath, baseDir)

	slashPath := filepath.ToSlash(relToPartialsDir)
	partialName = strings.TrimSuffix(slashPath, ".pk")
	partialSrc = path.Join(vc.pathsConfig.PartialServePath, partialName)
	return targetSubDir, partialName, partialSrc
}

// calculateRelativePartialPath works out a relative path from the partials
// source folder to the given source path.
//
// Takes sourcePath (string) which is the full path to the partial file.
// Takes baseDir (string) which is the base folder for path calculation.
//
// Returns string which is the relative path from the partials folder.
func (vc *virtualisationContext) calculateRelativePartialPath(ctx context.Context, sourcePath, baseDir string) string {
	partialsSourceBase := filepath.Join(baseDir, vc.pathsConfig.PartialsSourceDir)
	prefixToTrim := partialsSourceBase + string(filepath.Separator)

	if relativePath, found := strings.CutPrefix(sourcePath, prefixToTrim); found {
		return relativePath
	}

	_, warnL := logger_domain.From(ctx, log)
	warnL.Warn("Partial component is outside of the configured PartialsSourceDir. Its URL path may be unexpected.",
		logger_domain.String("partial_path", sourcePath),
		logger_domain.String("partials_dir", partialsSourceBase),
	)
	relativePath, _ := filepath.Rel(baseDir, sourcePath)
	return relativePath
}

// createVirtualInstance builds a VirtualPageInstance from an entry point that
// has a VirtualPageSource.
//
// Takes ep (annotator_dto.EntryPoint) which provides the entry point data.
// Takes baseDir (string) which specifies the base directory for path
// calculations.
//
// Returns annotator_dto.VirtualPageInstance which contains the manifest key,
// route, and initial properties from the entry point.
func (vc *virtualisationContext) createVirtualInstance(ep annotator_dto.EntryPoint, baseDir string) annotator_dto.VirtualPageInstance {
	vps := ep.VirtualPageSource

	manifestKey := vc.calculateVirtualManifestKey(vps, baseDir)

	route := vc.calculateVirtualRoute(vps)

	return annotator_dto.VirtualPageInstance{
		ManifestKey:  manifestKey,
		Route:        route,
		InitialProps: vps.InitialProps,
	}
}

// calculateVirtualManifestKey creates a manifest key for a virtual page.
// For collection pages, it builds a unique key from the full URL path.
//
// Takes vps (*annotator_dto.VirtualPageSource) which holds the virtual page
// data including initial props with page metadata.
// Takes baseDir (string) which is the base directory for working out relative
// paths in the fallback case.
//
// Returns string which is the manifest key in the format "pages/path.pk".
func (*virtualisationContext) calculateVirtualManifestKey(vps *annotator_dto.VirtualPageSource, baseDir string) string {
	if page, ok := vps.InitialProps["page"].(map[string]any); ok {
		if url, ok := page[collection_dto.MetaKeyURL].(string); ok {
			url = strings.TrimPrefix(url, "/")
			return "pages/" + url + ".pk"
		}
	}

	relativePath, err := filepath.Rel(baseDir, vps.TemplatePath)
	if err != nil {
		relativePath = vps.TemplatePath
	}

	slug := ""
	if page, ok := vps.InitialProps["page"].(map[string]any); ok {
		if s, ok := page[collection_dto.MetaKeySlug].(string); ok {
			slug = s
		}
	}

	if slug == "" {
		return filepath.ToSlash(relativePath)
	}

	key := filepath.ToSlash(relativePath)
	key = dynamicParamRegex.ReplaceAllString(key, slug)
	return key
}

// calculateVirtualRoute builds the URL route for a virtual page.
//
// Takes vps (*annotator_dto.VirtualPageSource) which provides the page source
// data, including route overrides and initial properties.
//
// Returns string which is the resolved route, or an empty string if no route
// can be found.
func (*virtualisationContext) calculateVirtualRoute(vps *annotator_dto.VirtualPageSource) string {
	if vps.RouteOverride != "" {
		return vps.RouteOverride
	}

	if page, ok := vps.InitialProps["page"].(map[string]any); ok {
		if url, ok := page[collection_dto.MetaKeyURL].(string); ok {
			return url
		}
	}

	return ""
}

// rewriteAllScriptASTs processes all component script ASTs and rewrites them
// for virtualisation.
//
// Returns error when AST rewriting fails or a duplicate package path is found.
func (vc *virtualisationContext) rewriteAllScriptASTs(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	l.Internal("[VIRTUALISER-STEP 3/4] Rewriting component Go ASTs...")
	for hashedName, virtualComp := range vc.virtualModule.ComponentsByHash {
		if virtualComp.Source.Script == nil || virtualComp.Source.Script.AST == nil {
			continue
		}
		rewriter := newASTRewriter(vc, virtualComp)
		rewrittenAST, shadowedAliases, err := rewriter.rewrite(ctx)
		if err != nil {
			return fmt.Errorf("failed to rewrite AST for '%s': %w", virtualComp.Source.SourcePath, err)
		}
		for _, alias := range shadowedAliases {
			message := fmt.Sprintf("Local variable '%s' shadows Piko import alias; avoid shadowing import aliases to prevent unexpected behaviour", alias)
			diagnostic := ast_domain.NewDiagnosticWithCode(ast_domain.Warning, message, "", annotator_dto.CodeVariableShadowing, ast_domain.Location{}, virtualComp.Source.SourcePath)
			vc.virtualModule.Diagnostics = append(vc.virtualModule.Diagnostics, diagnostic)
		}
		virtualComp.RewrittenScriptAST = rewrittenAST
		virtualComp.PikoAliasToHash = rewriter.pikoAliasToHash
		if _, exists := vc.virtualModule.ComponentsByGoPath[virtualComp.CanonicalGoPackagePath]; exists {
			return fmt.Errorf("internal error: duplicate canonical Go package path detected (%s): %s", hashedName, virtualComp.CanonicalGoPackagePath)
		}
		vc.virtualModule.ComponentsByGoPath[virtualComp.CanonicalGoPackagePath] = virtualComp
	}
	return nil
}

// assembleSourceOverlay copies original Go files and rewritten virtual
// component ASTs into the source overlay.
//
// Returns error when printing a virtual Go file fails.
func (vc *virtualisationContext) assembleSourceOverlay(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)
	l.Internal("[VIRTUALISER-STEP 4/4] Assembling source overlay...")
	maps.Copy(vc.virtualModule.SourceOverlay, vc.originalGoFiles)
	for _, virtualComp := range vc.virtualModule.ComponentsByHash {
		if virtualComp.RewrittenScriptAST == nil {
			continue
		}
		var buffer strings.Builder
		if err := printer.Fprint(&buffer, token.NewFileSet(), virtualComp.RewrittenScriptAST); err != nil {
			return fmt.Errorf("failed to print virtual Go file for '%s': %w", virtualComp.Source.SourcePath, err)
		}
		vc.virtualModule.SourceOverlay[virtualComp.VirtualGoFilePath] = []byte(buffer.String())
	}
	return nil
}

// pkgMemberResult holds the result of extracting a package member pattern.
type pkgMemberResult struct {
	// root is the package or module identifier in a qualified expression.
	root *ast_domain.Identifier

	// member is the name of the accessed package member.
	member string
}

// getModuleRootAndMember finds the root identifier and first member from an
// expression.
//
// For example, for util.FormatUser(state), it returns ("util", "FormatUser").
// For util alone, it returns ("util", "").
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns root (*ast_domain.Identifier) which is the package name, or nil if
// not found.
// Returns member (string) which is the first member name, or empty if there is
// none.
func getModuleRootAndMember(expression ast_domain.Expression) (root *ast_domain.Identifier, member string) {
	root, member, _ = getModuleRootAndMemberWithCallInfo(expression)
	return root, member
}

// getModuleRootAndMemberWithCallInfo finds the root identifier and first
// member from an expression, and reports whether the member is being called.
//
// For example:
//   - util.FormatUser(state) returns ("util", "FormatUser", true)
//   - util.SomeValue returns ("util", "SomeValue", false)
//   - util alone returns ("util", "", false)
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns root (*ast_domain.Identifier) which is the package name, or nil if
// not found.
// Returns member (string) which is the first member name, or empty if none
// exists.
// Returns isCall (bool) which is true if the member is used as a function
// call.
func getModuleRootAndMemberWithCallInfo(expression ast_domain.Expression) (root *ast_domain.Identifier, member string, isCall bool) {
	current := expression
	wasCall := false
	for {
		switch n := current.(type) {
		case *ast_domain.Identifier:
			return n, "", false
		case *ast_domain.MemberExpression:
			if rootIdent, memberName, found := tryExtractPackageMemberPattern(n); found {
				return rootIdent, memberName, wasCall
			}
			current = n.Base
			wasCall = false
		case *ast_domain.IndexExpression:
			current = n.Base
		case *ast_domain.CallExpression:
			if result := tryExtractCallExprMember(n); result != nil {
				return result.root, result.member, true
			}
			current = n.Callee
			wasCall = true
		default:
			return nil, "", false
		}
	}
}

// tryExtractCallExprMember checks if a call expression is a direct call to a
// package member (such as pkg.Func()) and extracts the pattern if found.
//
// Takes n (*ast_domain.CallExpression) which is the call expression to check.
//
// Returns *pkgMemberResult which contains the root identifier and member name,
// or nil if the expression is not a package member call.
func tryExtractCallExprMember(n *ast_domain.CallExpression) *pkgMemberResult {
	memberExpr, isMember := n.Callee.(*ast_domain.MemberExpression)
	if !isMember {
		return nil
	}
	rootIdent, memberName, found := tryExtractPackageMemberPattern(memberExpr)
	if !found {
		return nil
	}
	return &pkgMemberResult{root: rootIdent, member: memberName}
}

// tryExtractPackageMemberPattern checks if a MemberExpr matches the pattern
// `pkg.Member` and extracts the root identifier and member name if it does.
//
// Takes n (*ast_domain.MemberExpression) which is the member expression to check.
//
// Returns *ast_domain.Identifier which is the root package identifier, or nil
// if the pattern does not match.
// Returns string which is the member name, or empty if the pattern does not
// match.
// Returns bool which is true if the pattern matched.
func tryExtractPackageMemberPattern(n *ast_domain.MemberExpression) (*ast_domain.Identifier, string, bool) {
	rootIdent, isRoot := n.Base.(*ast_domain.Identifier)
	if !isRoot {
		return nil, "", false
	}

	memberIdent, isMember := n.Property.(*ast_domain.Identifier)
	if !isMember {
		return nil, "", false
	}

	return rootIdent, memberIdent.Name, true
}

// createImportSpec creates an import specification for the given path.
//
// Takes importPath (string) which is the import path to include.
// Takes alias (string) which is the alias name, or empty for no alias.
//
// Returns *goast.ImportSpec which represents the import declaration.
func createImportSpec(importPath, alias string) *goast.ImportSpec {
	spec := &goast.ImportSpec{
		Path: &goast.BasicLit{Kind: token.STRING, Value: fmt.Sprintf(`%q`, importPath)},
	}
	if alias != "" {
		spec.Name = goast.NewIdent(alias)
	}
	return spec
}

// hasFuncDecl checks whether a function with the given name exists in the AST.
//
// Takes scriptAST (*goast.File) which is the parsed Go source file.
// Takes name (string) which is the function name to search for.
//
// Returns bool which is true if a function with the given name was found.
func hasFuncDecl(scriptAST *goast.File, name string) bool {
	for _, declaration := range scriptAST.Decls {
		if functionDeclaration, ok := declaration.(*goast.FuncDecl); ok && functionDeclaration.Name.Name == name {
			return true
		}
	}
	return false
}

// ensureImportAndGetAlias checks if an import exists and adds it if not found.
//
// Takes scriptAST (*goast.File) which is the AST to check and modify.
// Takes importPath (string) which is the import path to find or add.
// Takes defaultAlias (string) which is the alias to use when adding
// the import.
//
// Returns string which is the alias to use for the import.
func ensureImportAndGetAlias(scriptAST *goast.File, importPath, defaultAlias string) string {
	for _, imp := range scriptAST.Imports {
		if strings.Trim(imp.Path.Value, `"`) == importPath {
			if imp.Name != nil {
				return imp.Name.Name
			}
			return defaultAlias
		}
	}
	scriptAST.Imports = append(scriptAST.Imports, createImportSpec(importPath, defaultAlias))
	reconstructModuleImportBlock(scriptAST)
	return defaultAlias
}

// buildDefaultRenderDecl builds the AST for a default Render function.
//
// When propsTypeExpr is provided, it uses that type in the function signature.
// When propsTypeExpr is nil, it uses NoProps as the default type.
//
// Takes dtoAlias (string) which is the package alias for the DTO types.
// Takes propsTypeExpr (goast.Expr) which is the props type to use, or nil for
// the default NoProps type.
//
// Returns *goast.FuncDecl which is the complete function declaration AST.
func buildDefaultRenderDecl(dtoAlias string, propsTypeExpr goast.Expr) *goast.FuncDecl {
	finalPropsType := propsTypeExpr
	if finalPropsType == nil {
		finalPropsType = &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(noPropsStruct)}
	}

	return &goast.FuncDecl{
		Name: goast.NewIdent(renderFuncName),
		Type: &goast.FuncType{
			Params: &goast.FieldList{
				List: []*goast.Field{
					{
						Names: []*goast.Ident{goast.NewIdent(requestParamName)},
						Type: &goast.StarExpr{
							X: &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(requestDataStruct)},
						},
					},
					{
						Names: []*goast.Ident{goast.NewIdent(propsParamName)},
						Type:  finalPropsType,
					},
				},
			},
			Results: &goast.FieldList{
				List: []*goast.Field{
					{Type: &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(noResponseStruct)}},
					{Type: &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(metadataStruct)}},
					{Type: goast.NewIdent(errorInterface)},
				},
			},
		},
		Body: &goast.BlockStmt{
			List: []goast.Stmt{
				&goast.ReturnStmt{
					Results: []goast.Expr{
						&goast.CompositeLit{Type: &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(noResponseStruct)}},
						&goast.CompositeLit{Type: &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(metadataStruct)}},
						goast.NewIdent("nil"),
					},
				},
			},
		},
	}
}

// buildDefaultCachePolicyDecl builds an AST function declaration that returns
// a default cache policy struct.
//
// Takes dtoAlias (string) which is the package alias for the DTO type.
//
// Returns *goast.FuncDecl which is the generated function declaration.
func buildDefaultCachePolicyDecl(dtoAlias string) *goast.FuncDecl {
	return &goast.FuncDecl{
		Name: goast.NewIdent(cachePolicyFuncName),
		Type: &goast.FuncType{
			Params: &goast.FieldList{},
			Results: &goast.FieldList{List: []*goast.Field{
				{Type: &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(cachePolicyStruct)}},
			}},
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.ReturnStmt{Results: []goast.Expr{
				&goast.CompositeLit{Type: &goast.SelectorExpr{X: goast.NewIdent(dtoAlias), Sel: goast.NewIdent(cachePolicyStruct)}},
			}},
		}},
	}
}

// reconstructModuleImportBlock rebuilds the import block for a file.
//
// Takes file (*goast.File) which is the AST file to modify in place.
func reconstructModuleImportBlock(file *goast.File) {
	otherDecls := make([]goast.Decl, 0, len(file.Decls))
	for _, declaration := range file.Decls {
		if gen, ok := declaration.(*goast.GenDecl); ok && gen.Tok == token.IMPORT {
			continue
		}
		otherDecls = append(otherDecls, declaration)
	}
	if len(file.Imports) > 0 {
		importDecl := &goast.GenDecl{
			Tok:    token.IMPORT,
			Lparen: token.Pos(1),
			Specs:  make([]goast.Spec, len(file.Imports)),
		}
		for i, s := range file.Imports {
			importDecl.Specs[i] = s
		}
		file.Decls = append([]goast.Decl{importDecl}, otherDecls...)
	} else {
		file.Decls = otherDecls
	}
}
