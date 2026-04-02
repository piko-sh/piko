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

// Discovers and builds the component dependency graph by recursively parsing
// templates and resolving partial references. Detects circular dependencies,
// handles .pikoignore patterns, and produces a complete graph of all components
// in the project.

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"

	"github.com/cespare/xxhash/v2"
	"golang.org/x/sync/errgroup"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

const (
	// pikoFileExtension is the file extension for Piko template files.
	pikoFileExtension = ".pk"

	// shortHashLength is the number of hex characters kept in short hashes.
	shortHashLength = 8

	// defaultPackagePrefix is the prefix added to package names that start with a
	// non-letter character.
	defaultPackagePrefix = "p"

	// defaultPackageName is the fallback package name used when no name is given.
	defaultPackageName = "p_default_pkg_name"
)

// GraphBuilder finds and parses all .pk components to build a complete
// dependency graph. It is the first stage in the compilation pipeline and
// creates the initial, Piko-native view of the project.
type GraphBuilder struct {
	// resolver resolves file paths and looks up module names.
	resolver resolver_domain.ResolverPort

	// fsReader reads files during dependency graph traversal.
	fsReader FSReaderPort

	// cache stores parsed components to avoid reading the same file twice.
	cache ComponentCachePort

	// pathsConfig holds path settings used to build dependency graphs.
	pathsConfig AnnotatorPathsConfig

	// faultTolerant allows the builder to continue when parse errors occur.
	// When true, broken components are registered in PathToHashedName so that
	// dependent files can still resolve imports (used by LSP).
	faultTolerant bool
}

// NewGraphBuilder creates a new GraphBuilder with the given dependencies.
//
// Takes resolver (ResolverPort) which finds links between components.
// Takes fsReader (FSReaderPort) which reads files from the filesystem.
// Takes cache (ComponentCachePort) which stores component data for reuse.
// Takes pathsConfig (AnnotatorPathsConfig) which holds the path settings.
// Takes faultTolerant (bool) which enables fault-tolerant mode for LSP usage.
//
// Returns *GraphBuilder which is ready to build dependency graphs.
func NewGraphBuilder(resolver resolver_domain.ResolverPort, fsReader FSReaderPort, cache ComponentCachePort, pathsConfig AnnotatorPathsConfig, faultTolerant bool) *GraphBuilder {
	return &GraphBuilder{
		resolver:      resolver,
		fsReader:      fsReader,
		cache:         cache,
		pathsConfig:   pathsConfig,
		faultTolerant: faultTolerant,
	}
}

// Build walks the component dependency graph from one or more entry points,
// checks for cycles, and returns a single ComponentGraph with all found
// components and dependencies. Parsing is done in parallel for speed.
//
// Takes entryPointPaths ([]string) which specifies the starting points for
// graph traversal.
//
// Returns *annotator_dto.ComponentGraph which contains all found components
// and their dependencies.
// Returns []*ast_domain.Diagnostic which contains any warnings or errors found
// during traversal.
// Returns error when no entry points are provided or discovery fails.
func (gb *GraphBuilder) Build(ctx context.Context, entryPointPaths []string) (*annotator_dto.ComponentGraph, []*ast_domain.Diagnostic, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "GraphBuilder.Build", logger_domain.Strings("entryPoints", entryPointPaths))
	defer span.End()

	if len(entryPointPaths) == 0 {
		return nil, nil, errors.New("GraphBuilder.Build requires at least one entry point path")
	}

	l.Internal("[GraphBuilder] Phase 1: Discovering all component paths...")
	allPaths, discoveryDiags, err := gb.DiscoverAllPaths(ctx, entryPointPaths)
	if err != nil {
		return nil, discoveryDiags, err
	}
	l.Internal("[GraphBuilder] Discovery complete.", logger_domain.Int("unique_components", len(allPaths)))

	l.Internal("[GraphBuilder] Phase 2: Parsing all components in parallel...")
	componentGraph, parseDiags, err := gb.parseAllPaths(ctx, allPaths)
	if err != nil {
		return nil, append(discoveryDiags, parseDiags...), err
	}
	l.Internal("[GraphBuilder] Parallel parsing complete.")

	allDiagnostics := make([]*ast_domain.Diagnostic, 0, len(discoveryDiags)+len(parseDiags))
	allDiagnostics = append(allDiagnostics, discoveryDiags...)
	allDiagnostics = append(allDiagnostics, parseDiags...)
	if ast_domain.HasErrors(allDiagnostics) {
		return componentGraph, allDiagnostics, nil
	}

	l.Internal("[GraphBuilder] Phase 3: Checking for circular dependencies...")
	cycleDiags := gb.detectCycles(ctx, componentGraph)
	allDiagnostics = append(allDiagnostics, cycleDiags...)
	l.Internal("[GraphBuilder] Finished graph build process.")

	return componentGraph, allDiagnostics, nil
}

// DiscoverAllPaths performs a fast, sequential BFS to find all unique file
// paths reachable from the given entry points.
//
// This method orchestrates the discovery process in three steps: resolve
// initial entry points, traverse the dependency graph via BFS, and return
// the complete set of discovered paths.
//
// Takes entryPointPaths ([]string) which specifies the starting file paths
// for discovery.
//
// Returns []string which contains all unique file paths discovered.
// Returns []*ast_domain.Diagnostic which contains any issues found during
// discovery.
// Returns error when the dependency graph traversal fails.
func (gb *GraphBuilder) DiscoverAllPaths(ctx context.Context, entryPointPaths []string) ([]string, []*ast_domain.Diagnostic, error) {
	queue, visited, diagnostics := gb.resolveAndQueueEntryPoints(ctx, entryPointPaths)
	queue, err := gb.traverseDependencyGraph(ctx, queue, visited, &diagnostics)
	return queue, diagnostics, err
}

// resolveAndQueueEntryPoints resolves entry point paths and creates the
// initial queue for breadth-first search.
//
// Takes entryPointPaths ([]string) which contains paths to resolve. These may
// be absolute or relative paths.
//
// Returns []string which is the initial queue of resolved absolute paths.
// Returns map[string]bool which tracks visited paths to prevent duplicates.
// Returns []*ast_domain.Diagnostic which contains any resolution errors.
func (gb *GraphBuilder) resolveAndQueueEntryPoints(ctx context.Context, entryPointPaths []string) ([]string, map[string]bool, []*ast_domain.Diagnostic) {
	queue := make([]string, 0, len(entryPointPaths))
	visited := make(map[string]bool)
	var diagnostics []*ast_domain.Diagnostic

	for _, path := range entryPointPaths {
		var resolvedPath string
		var err error

		if filepath.IsAbs(path) {
			resolvedPath = path
		} else {
			resolvedPath, err = gb.resolver.ResolvePKPath(ctx, path, "")
			if err != nil {
				message := fmt.Sprintf("Cannot resolve entry point: %v", err)
				location := ast_domain.Location{Line: 1, Column: 1, Offset: 0}
				diagnostic := ast_domain.NewDiagnosticWithCode(ast_domain.Error, message, path, annotator_dto.CodeGraphBuildError, location, "command-line argument")
				diagnostics = append(diagnostics, diagnostic)
				continue
			}
		}

		if !visited[resolvedPath] {
			queue = append(queue, resolvedPath)
			visited[resolvedPath] = true
		}
	}

	return queue, visited, diagnostics
}

// traverseDependencyGraph performs a BFS traversal of the component dependency
// graph.
//
// For each file in the queue, it reads the file, extracts imports, resolves
// them, and adds newly discovered files to the queue. The queue grows as new
// dependencies are discovered. Any resolution errors are captured as
// diagnostics.
//
// Takes queue ([]string) which contains the initial file paths to process.
// Takes visited (map[string]bool) which tracks already discovered paths.
// Takes diagnostics (*[]*ast_domain.Diagnostic) which collects resolution
// errors.
//
// Returns []string which is the complete queue of all discovered paths.
// Returns error when a file cannot be read.
func (gb *GraphBuilder) traverseDependencyGraph(ctx context.Context, queue []string, visited map[string]bool, diagnostics *[]*ast_domain.Diagnostic) ([]string, error) {
	for i := 0; i < len(queue); i++ {
		if ctx.Err() != nil {
			return queue, ctx.Err()
		}

		currentPath := queue[i]
		data, err := gb.fsReader.ReadFile(ctx, currentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file during discovery %s: %w", currentPath, err)
		}
		_, pikoImports, _ := getPikoImports(data)

		for _, imp := range pikoImports {
			resolved, err := gb.resolver.ResolvePKPath(ctx, imp.Path, currentPath)
			if err != nil {
				detailedMessage := fmt.Sprintf("GraphBuilder failed to resolve import '%s' from file '%s'. Resolver error: %v", imp.Path, currentPath, err)
				diagnostic := ast_domain.NewDiagnosticWithCode(ast_domain.Error, detailedMessage, imp.Path, annotator_dto.CodeUnresolvedImport, imp.Location, currentPath)
				*diagnostics = append(*diagnostics, diagnostic)
				continue
			}
			if !visited[resolved] {
				queue = append(queue, resolved)
				visited[resolved] = true
			}
		}
	}

	return queue, nil
}

// parseResult holds the output from parsing a single file during concurrent
// processing.
type parseResult struct {
	// err holds any error from parsing; nil means success.
	err error

	// parsedComponent holds the parsed documentation component; nil if parsing
	// failed.
	parsedComponent *annotator_dto.ParsedComponent

	// path is the file path of the parsed source file.
	path string

	// source holds the original file bytes being parsed.
	source []byte
}

// parseAllPaths parses the given file paths and builds the component graph.
//
// Takes allPaths ([]string) which contains the file paths to parse.
//
// Returns *annotator_dto.ComponentGraph which is the built component graph.
// Returns []*ast_domain.Diagnostic which contains any issues found.
// Returns error when the parsing workers fail to start.
func (gb *GraphBuilder) parseAllPaths(ctx context.Context, allPaths []string) (*annotator_dto.ComponentGraph, []*ast_domain.Diagnostic, error) {
	resultsChan, err := gb.runParsingWorkers(ctx, allPaths)
	if err != nil {
		return nil, nil, fmt.Errorf("running parsing workers: %w", err)
	}
	return gb.aggregateParseResults(ctx, resultsChan)
}

// runParsingWorkers parses files in parallel using a worker pool.
//
// Takes allPaths ([]string) which contains the file paths to parse.
//
// Returns <-chan *parseResult which yields parse results for each file.
// Returns error when a worker fails.
func (gb *GraphBuilder) runParsingWorkers(ctx context.Context, allPaths []string) (<-chan *parseResult, error) {
	g, gCtx := errgroup.WithContext(ctx)
	jobs := make(chan string, len(allPaths))
	for _, path := range allPaths {
		jobs <- path
	}
	close(jobs)

	numWorkers := min(runtime.NumCPU(), len(allPaths))

	resultsChan := make(chan *parseResult, len(allPaths))

	for range numWorkers {
		g.Go(func() error {
			for path := range jobs {
				select {
				case <-gCtx.Done():
					return gCtx.Err()
				default:
					parsedComponent, source, err := gb.readAndParse(gCtx, path)
					resultsChan <- &parseResult{
						path:            path,
						parsedComponent: parsedComponent,
						source:          source,
						err:             err,
					}
				}
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("waiting for parsing workers: %w", err)
	}
	close(resultsChan)
	return resultsChan, nil
}

// aggregateParseResults collects parse results from the channel and builds a
// component graph.
//
// Takes ctx (context.Context) which carries the logger.
// Takes resultsChan (<-chan *parseResult) which provides parsed components.
//
// Returns *annotator_dto.ComponentGraph which contains the built graph.
// Returns []*ast_domain.Diagnostic which holds any warnings found.
// Returns error when a fatal parse error occurs.
func (gb *GraphBuilder) aggregateParseResults(ctx context.Context, resultsChan <-chan *parseResult) (*annotator_dto.ComponentGraph, []*ast_domain.Diagnostic, error) {
	componentGraph := &annotator_dto.ComponentGraph{
		Components:        make(map[string]*annotator_dto.ParsedComponent),
		PathToHashedName:  make(map[string]string),
		HashedNameToPath:  make(map[string]string),
		AllSourceContents: make(map[string][]byte),
	}
	var diagnostics []*ast_domain.Diagnostic

	for result := range resultsChan {
		resultDiags, fatalErr := gb.processParseResultError(result)
		if fatalErr != nil {
			return nil, nil, fatalErr
		}
		diagnostics = append(diagnostics, resultDiags...)

		if result.source != nil {
			componentGraph.AllSourceContents[result.path] = result.source
		}
		if result.parsedComponent != nil {
			gb.registerParsedComponent(componentGraph, result)
		} else if gb.faultTolerant && result.err != nil {
			gb.registerBrokenComponent(ctx, componentGraph, result.path)
		}
	}

	return componentGraph, diagnostics, nil
}

// processParseResultError converts parse errors into diagnostics.
//
// Takes result (*parseResult) which contains the parse outcome to check.
//
// Returns []*ast_domain.Diagnostic which holds any diagnostics extracted from
// known error types.
// Returns error when the error is not a known diagnostic type.
func (*GraphBuilder) processParseResultError(result *parseResult) ([]*ast_domain.Diagnostic, error) {
	if result.err == nil {
		return nil, nil
	}

	if diagErr, ok := errors.AsType[*ParseDiagnosticError](result.err); ok {
		return diagErr.Diagnostics, nil
	}

	if scriptErr, ok := errors.AsType[*scriptBlockParseError](result.err); ok {
		diagnostic := ast_domain.NewDiagnosticWithCode(ast_domain.Error, scriptErr.Error(), "", annotator_dto.CodeClientScriptError, ast_domain.Location{Line: 0, Column: 0, Offset: 0}, result.path)
		return []*ast_domain.Diagnostic{diagnostic}, nil
	}

	return nil, result.err
}

// registerParsedComponent adds a parsed component to the graph with computed
// metadata such as source path, external status, and module import path.
//
// Takes graph (*annotator_dto.ComponentGraph) which stores all components.
// Takes result (*parseResult) which contains the parsed component data.
func (gb *GraphBuilder) registerParsedComponent(graph *annotator_dto.ComponentGraph, result *parseResult) {
	baseDir := gb.resolver.GetBaseDir()
	isExternal := !strings.HasPrefix(result.path, baseDir)

	result.parsedComponent.SourcePath = result.path
	result.parsedComponent.IsExternal = isExternal
	result.parsedComponent.ModuleImportPath = gb.buildModuleImportPath(result.path, baseDir, isExternal)

	relativePath, _ := filepath.Rel(baseDir, result.path)
	hash := buildAliasFromPath(relativePath)
	graph.PathToHashedName[result.path] = hash
	graph.HashedNameToPath[hash] = hash
	graph.Components[hash] = result.parsedComponent
}

// registerBrokenComponent registers a stub entry for a component that failed
// to parse. Dependent files can still resolve the import path in
// fault-tolerant mode (LSP), preventing cascading "could not find hash"
// errors.
//
// Takes ctx (context.Context) which carries the logger.
// Takes graph (*annotator_dto.ComponentGraph) which stores all components.
// Takes path (string) which is the absolute path to the broken component.
func (gb *GraphBuilder) registerBrokenComponent(ctx context.Context, graph *annotator_dto.ComponentGraph, path string) {
	baseDir := gb.resolver.GetBaseDir()
	relativePath, _ := filepath.Rel(baseDir, path)
	hash := buildAliasFromPath(relativePath)

	graph.PathToHashedName[path] = hash
	graph.HashedNameToPath[hash] = path

	graph.Components[hash] = &annotator_dto.ParsedComponent{
		SourcePath:       path,
		IsExternal:       !strings.HasPrefix(path, baseDir),
		ModuleImportPath: gb.buildModuleImportPath(path, baseDir, !strings.HasPrefix(path, baseDir)),
		ComponentType:    gb.determineComponentType(ctx, path),
	}
}

// buildModuleImportPath builds the full import path for a module.
//
// Takes path (string) which is the file system path to the module.
// Takes baseDir (string) which is the base directory for working out relative
// paths.
// Takes isExternal (bool) which indicates whether the module is external.
//
// Returns string which is the full import path.
func (gb *GraphBuilder) buildModuleImportPath(path, baseDir string, isExternal bool) string {
	if isExternal {
		return extractModuleImportPath(path)
	}
	relativePath, _ := filepath.Rel(baseDir, path)
	moduleName := gb.resolver.GetModuleName()
	return moduleName + "/" + filepath.ToSlash(relativePath)
}

// readAndParse reads a file and parses it into a component.
//
// Uses the cache adapter to store and fetch parsed results.
//
// Takes path (string) which is the file path to read and parse.
//
// Returns *annotator_dto.ParsedComponent which is the parsed component.
// In fault-tolerant mode, a partial component may be returned along with an
// error to allow dependent files to still resolve imports.
// Returns []byte which is the raw file content.
// Returns error when reading or parsing fails.
func (gb *GraphBuilder) readAndParse(ctx context.Context, path string) (*annotator_dto.ParsedComponent, []byte, error) {
	fileData, err := gb.fsReader.ReadFile(ctx, path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading file %q: %w", path, err)
	}

	cacheKey := generateCacheKey(path, fileData)

	loader := func(ctx context.Context) (*annotator_dto.ParsedComponent, error) {
		component, _, parseErr := ParsePK(ctx, fileData, path)

		if parseErr != nil {
			if gb.faultTolerant && component != nil {
				component.ComponentType = gb.determineComponentType(ctx, path)
				return component, parseErr
			}
			return nil, parseErr
		}

		component.ComponentType = gb.determineComponentType(ctx, path)

		pmlUsageDiags := validatePMLUsage(component)
		if len(pmlUsageDiags) > 0 && component.Template != nil {
			component.Template.Diagnostics = append(component.Template.Diagnostics, pmlUsageDiags...)
		}

		return component, nil
	}

	parsedComponent, err := gb.cache.GetOrSet(ctx, cacheKey, loader)
	if err != nil {
		if gb.faultTolerant && parsedComponent != nil {
			return parsedComponent, fileData, err
		}
		return nil, fileData, err
	}

	return parsedComponent, fileData, nil
}

// determineComponentType finds the type of a component based on its file path.
//
// Takes ctx (context.Context) which carries the logger.
// Takes absolutePath (string) which is the full path to the component file.
//
// Returns string which is the component type: "page", "email", "partial",
// or "component" as the default.
func (gb *GraphBuilder) determineComponentType(ctx context.Context, absolutePath string) string {
	_, l := logger_domain.From(ctx, log)
	baseDir := gb.resolver.GetBaseDir()

	pagesDir := filepath.Join(baseDir, gb.pathsConfig.PagesSourceDir)
	emailsDir := filepath.Join(baseDir, gb.pathsConfig.EmailsSourceDir)
	pdfsDir := filepath.Join(baseDir, gb.pathsConfig.PdfsSourceDir)
	partialsDir := filepath.Join(baseDir, gb.pathsConfig.PartialsSourceDir)

	e2ePagesDir := filepath.Join(baseDir, gb.pathsConfig.E2ESourceDir, "pages")
	e2ePartialsDir := filepath.Join(baseDir, gb.pathsConfig.E2ESourceDir, "partials")

	if gb.pathsConfig.EmailsSourceDir != "" && strings.HasPrefix(absolutePath, emailsDir) {
		return "email"
	}
	if gb.pathsConfig.PdfsSourceDir != "" && strings.HasPrefix(absolutePath, pdfsDir) {
		return "pdf"
	}
	if strings.HasPrefix(absolutePath, pagesDir) || strings.HasPrefix(absolutePath, e2ePagesDir) {
		return "page"
	}
	if strings.HasPrefix(absolutePath, partialsDir) || strings.HasPrefix(absolutePath, e2ePartialsDir) {
		return "partial"
	}

	if gb.pathsConfig.PagesSourceDir == "" || gb.pathsConfig.PartialsSourceDir == "" {
		l.Warn("Component type determination attempted with incomplete configuration",
			logger_domain.String("path", absolutePath),
			logger_domain.String("pages_dir", gb.pathsConfig.PagesSourceDir),
			logger_domain.String("partials_dir", gb.pathsConfig.PartialsSourceDir),
			logger_domain.String("emails_dir", gb.pathsConfig.EmailsSourceDir),
			logger_domain.String("pdfs_dir", gb.pathsConfig.PdfsSourceDir),
		)
	}

	return "component"
}

// detectCycles checks the graph for circular dependencies.
//
// Takes graph (*annotator_dto.ComponentGraph) which is the component graph to
// check.
//
// Returns []*ast_domain.Diagnostic which contains any cycle errors found.
func (gb *GraphBuilder) detectCycles(ctx context.Context, graph *annotator_dto.ComponentGraph) []*ast_domain.Diagnostic {
	detector := &cycleDetector{
		graph:       graph,
		resolver:    gb.resolver,
		visiting:    make(map[string]bool),
		visited:     make(map[string]bool),
		diagnostics: make([]*ast_domain.Diagnostic, 0),
	}
	for hashedName := range graph.Components {
		if !detector.visited[hashedName] {
			detector.dfs(ctx, hashedName, nil)
		}
	}
	return detector.diagnostics
}

// cycleDetector uses a depth-first search to find circular dependencies in the
// graph.
type cycleDetector struct {
	// graph is the dependency graph used to detect cycles between components.
	graph *annotator_dto.ComponentGraph

	// resolver resolves import paths to file paths.
	resolver resolver_domain.ResolverPort

	// visiting tracks packages in the current DFS path to detect import cycles.
	visiting map[string]bool

	// visited tracks which components have been fully checked by the DFS.
	visited map[string]bool

	// diagnostics collects any problems found during cycle detection.
	diagnostics []*ast_domain.Diagnostic
}

// dfs searches for circular dependencies using depth-first traversal.
//
// Takes ctx (context.Context) which carries cancellation and tracing.
// Takes hashedName (string) which is the unique name of the component to check.
// Takes path ([]string) which holds the current path for cycle reporting.
func (cd *cycleDetector) dfs(ctx context.Context, hashedName string, path []string) {
	if ctx.Err() != nil {
		return
	}

	cd.visiting[hashedName] = true
	cd.visited[hashedName] = true

	currentPath := make([]string, len(path)+1)
	copy(currentPath, path)
	currentPath[len(path)] = cd.graph.HashedNameToPath[hashedName]
	component, ok := cd.graph.Components[hashedName]
	if !ok {
		return
	}

	for _, pikoImport := range component.PikoImports {
		resolvedPath, err := cd.resolver.ResolvePKPath(ctx, pikoImport.Path, component.SourcePath)
		if err != nil {
			continue
		}
		depHashedName, ok := cd.graph.PathToHashedName[resolvedPath]
		if !ok {
			continue
		}

		if cd.visiting[depHashedName] {
			cyclePath := make([]string, len(currentPath)+1)
			copy(cyclePath, currentPath)
			cyclePath[len(currentPath)] = resolvedPath
			message := fmt.Sprintf("Circular dependency detected: %s", strings.Join(cyclePath, " -> "))
			diagnostic := ast_domain.NewDiagnosticWithCode(
				ast_domain.Error,
				message,
				pikoImport.Path,
				annotator_dto.CodeCircularDependency,
				pikoImport.Location,
				component.SourcePath,
			)
			cd.diagnostics = append(cd.diagnostics, diagnostic)
			continue
		}
		if !cd.visited[depHashedName] {
			cd.dfs(ctx, depHashedName, currentPath)
		}
	}
	cd.visiting[hashedName] = false
}

// SanitiseForPackageName cleans a string to make it a valid Go package name
// part.
//
// Takes name (string) which is the raw string to clean.
//
// Returns string which is a valid package name part with only lowercase
// letters, digits, and underscores.
func SanitiseForPackageName(name string) string {
	lowerName := strings.ToLower(name)
	var builder strings.Builder
	builder.Grow(len(lowerName))
	lastWasSeparator := true

	for _, r := range lowerName {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			_, _ = builder.WriteRune(r)
			lastWasSeparator = false
		} else if !lastWasSeparator {
			_, _ = builder.WriteRune('_')
			lastWasSeparator = true
		}
	}

	result := strings.Trim(builder.String(), "_")
	if result == "" {
		return defaultPackageName
	}
	if !unicode.IsLetter(rune(result[0])) {
		return defaultPackagePrefix + result
	}
	return result
}

// generateCacheKey creates a hash from a file path and its content. Including
// the path means that if a file is renamed, even with the same content, it will
// be treated as new and parsed again.
//
// Takes path (string) which is the file location.
// Takes content ([]byte) which is the raw file data.
//
// Returns string which is the hex-encoded hash of the path and content.
func generateCacheKey(path string, content []byte) string {
	hasher := xxhash.New()
	_, _ = hasher.WriteString(path)
	_, _ = hasher.Write(content)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// getPikoImports parses a file to extract .pk imports.
//
// Takes data ([]byte) which contains the raw file content to parse.
//
// Returns Sources which contains the separated source parts.
// Returns []annotator_dto.PikoImport which lists the found Piko imports.
// Returns error when the file cannot be parsed or has syntax errors.
func getPikoImports(data []byte) (Sources, []annotator_dto.PikoImport, error) {
	_, srcs, err := parseAndSeparateSFC(data)
	if err != nil {
		return srcs, nil, err
	}
	if isEffectivelyEmpty(srcs.ScriptSource) {
		return srcs, nil, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", srcs.ScriptSource, parser.ImportsOnly)
	if err != nil {
		return srcs, nil, &scriptBlockParseError{reason: err.Error()}
	}

	pikoImports, _, _ := separateImports(file.Decls, fset)

	return srcs, pikoImports, nil
}

// buildAliasFromPath creates a stable, unique alias from a file path.
//
// The alias is safe to use as a Go identifier. It removes special characters,
// replaces path separators with underscores, and adds a short hash to ensure
// the result is unique.
//
// Takes relativePath (string) which is the file path to convert.
//
// Returns string which is a cleaned alias suitable for use as a Go identifier.
func buildAliasFromPath(relativePath string) string {
	cleanedPath := strings.ReplaceAll(filepath.ToSlash(relativePath), "/", "_")
	cleanedPath = strings.TrimSuffix(cleanedPath, pikoFileExtension)
	cleanedPath = strings.ReplaceAll(cleanedPath, "{", "")
	cleanedPath = strings.ReplaceAll(cleanedPath, "}", "")
	cleanedPath = strings.ReplaceAll(cleanedPath, "-", "_")

	return fmt.Sprintf("%s_%s", SanitiseForPackageName(cleanedPath), shortHash(relativePath))
}

// shortHash creates a fixed-length hex string from the xxhash of an input.
// It uses xxhash for speed and to show this is not for cryptographic use.
//
// Takes txt (string) which is the input to hash.
//
// Returns string which is the shortened hexadecimal hash.
func shortHash(txt string) string {
	hash := xxhash.Sum64String(txt)
	return fmt.Sprintf("%016x", hash)[:shortHashLength]
}

// extractModuleImportPath gets the module import path from a GOMODCACHE path
// by removing the pkg/mod/ prefix and version suffix.
//
// Takes gomodcachePath (string) which is the full path within GOMODCACHE.
//
// Returns string which is the cleaned module import path, or the original
// path if it cannot be parsed.
func extractModuleImportPath(gomodcachePath string) string {
	_, pathAfterMod, found := strings.Cut(gomodcachePath, "pkg/mod/")
	if !found {
		return gomodcachePath
	}

	atIndex := strings.Index(pathAfterMod, "@")
	if atIndex == -1 {
		return filepath.ToSlash(pathAfterMod)
	}

	slashAfterAt := strings.Index(pathAfterMod[atIndex:], "/")
	if slashAfterAt == -1 {
		return filepath.ToSlash(pathAfterMod[:atIndex])
	}

	modulePrefix := pathAfterMod[:atIndex]
	pathSuffix := pathAfterMod[atIndex+slashAfterAt:]
	return filepath.ToSlash(modulePrefix + pathSuffix)
}
