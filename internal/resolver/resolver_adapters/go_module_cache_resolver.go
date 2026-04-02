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

package resolver_adapters

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

var _ resolver_domain.ResolverPort = (*GoModuleCacheResolver)(nil)

const (
	// logKeyModulePath is the log field key for the module path.
	logKeyModulePath = "modulePath"

	// logKeyModuleDir is the log field key for the module folder path.
	logKeyModuleDir = "moduleDir"

	// pathSeparator is the separator used to split module import paths.
	pathSeparator = "/"
)

// GoModuleCacheResolver implements ResolverPort to resolve Piko component
// import paths from external Go modules in the module cache ($GOMODCACHE).
// It reads the project's go.mod to find required modules and matches import
// paths against them.
type GoModuleCacheResolver struct {
	// dirCache maps module paths to their directory locations in the cache.
	dirCache map[string]string

	// workingDir is the directory containing go.mod for packages.Load calls.
	workingDir string

	// knownModules holds the sorted list of module paths from go.mod; protected by
	// mu.
	knownModules []string

	// mu guards access to knownModules and dirCache.
	mu sync.RWMutex
}

// NewGoModuleCacheResolver creates a new Go module cache resolver.
//
// The resolver starts with an empty cache. The cache fills as modules are
// found during the build process.
//
// Before use, you must call DetectLocalModule or create the resolver with
// NewGoModuleCacheResolverWithWorkingDir to set up the module list.
//
// Returns *GoModuleCacheResolver which is ready for use after calling
// DetectLocalModule.
func NewGoModuleCacheResolver() *GoModuleCacheResolver {
	return &GoModuleCacheResolver{
		dirCache:     make(map[string]string),
		knownModules: nil,
		workingDir:   "",
		mu:           sync.RWMutex{},
	}
}

// NewGoModuleCacheResolverWithWorkingDir creates a new Go module cache resolver
// with a specific working directory for packages.Load operations. This is
// useful in test scenarios where the go.mod is in a non-standard location.
//
// This constructor automatically loads the known modules from the go.mod file
// in the specified working directory.
//
// Takes workingDir (string) which specifies the directory containing go.mod.
//
// Returns *GoModuleCacheResolver which is the configured resolver ready for
// use.
// Returns error when go.mod cannot be found or parsed.
func NewGoModuleCacheResolverWithWorkingDir(workingDir string) (*GoModuleCacheResolver, error) {
	resolver := &GoModuleCacheResolver{
		dirCache:     make(map[string]string),
		knownModules: nil,
		workingDir:   workingDir,
		mu:           sync.RWMutex{},
	}

	if err := resolver.loadKnownModulesFromGoMod(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load modules from go.mod: %w", err)
	}

	return resolver, nil
}

// DetectLocalModule loads the list of known modules from the project's go.mod
// file. This must be called before any path resolution can occur, as it builds
// the list of module boundaries used for import path parsing.
//
// Returns error when the go.mod file cannot be read or parsed.
func (gmcr *GoModuleCacheResolver) DetectLocalModule(ctx context.Context) error {
	return gmcr.loadKnownModulesFromGoMod(ctx)
}

// GetModuleName returns an empty string as this resolver does not manage a
// local module.
//
// Returns string which is always empty for this resolver.
func (*GoModuleCacheResolver) GetModuleName() string {
	return ""
}

// GetBaseDir returns an empty string as this resolver does not have a local
// base directory.
//
// Returns string which is always empty for this resolver type.
func (*GoModuleCacheResolver) GetBaseDir() string {
	return ""
}

// ConvertEntryPointPathToManifestKey returns the path as-is for external
// components. External components use their full module path as the manifest
// key (e.g., "github.com/ui/lib/components/button.pk") to distinguish them
// from local components and prevent naming conflicts.
//
// Takes entryPointPath (string) which is the full module path of the external
// component.
//
// Returns string which is the unmodified path to use as the manifest key.
func (*GoModuleCacheResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	return entryPointPath
}

// ResolvePKPath resolves a Piko component import path from an external Go
// module. It uses the go/packages API to locate the module in the Go module
// cache, then constructs the absolute path to the .pk file within that module.
//
// The @ alias is supported and will be expanded using the containing file's
// module.
//
// Takes importPath (string) which is the import path to resolve.
// Takes containingFilePath (string) which is the absolute path of the file
// containing the import statement, used to resolve the @ alias.
//
// Returns string which is the absolute path to the resolved .pk file.
// Returns error when the import path cannot be parsed, the module cannot be
// located, or the .pk file does not exist.
func (gmcr *GoModuleCacheResolver) ResolvePKPath(ctx context.Context, importPath string, containingFilePath string) (string, error) {
	ctx, span, _ := log.Span(ctx, "GoModuleCacheResolver.ResolvePKPath",
		logger_domain.String("importPath", importPath),
		logger_domain.String("containingFilePath", containingFilePath),
	)
	defer span.End()

	goModuleCacheResolutionCount.Add(ctx, 1)
	defer gmcr.recordResolutionDuration(ctx, time.Now())

	expandedPath, err := ExpandModuleAlias(importPath, containingFilePath)
	if err != nil {
		goModuleCacheResolutionErrorCount.Add(ctx, 1)
		return "", fmt.Errorf("expanding module alias for import %q: %w", importPath, err)
	}

	modulePath, filePathInModule, err := gmcr.parseImportPath(ctx, expandedPath)
	if err != nil {
		return "", fmt.Errorf("parsing import path %q: %w", expandedPath, err)
	}

	moduleDir, err := gmcr.resolveModuleDir(ctx, modulePath)
	if err != nil {
		goModuleCacheResolutionErrorCount.Add(ctx, 1)
		return "", fmt.Errorf("resolving module directory for %q: %w", modulePath, err)
	}

	absolutePath, err := gmcr.constructAndValidatePKPath(ctx, moduleDir, filePathInModule, importPath)
	if err != nil {
		return "", fmt.Errorf("constructing PK path for import %q: %w", importPath, err)
	}

	return absolutePath, nil
}

// ResolveCSSPath is not implemented for this resolver.
//
// CSS resolution from external Go modules is not currently supported, but may
// be added in future if there is a use case for it.
//
// Returns string which is always empty.
// Returns error which is always non-nil, showing the feature is unavailable.
func (*GoModuleCacheResolver) ResolveCSSPath(_ context.Context, _ string, _ string) (string, error) {
	return "", errors.New("CSS resolution from Go module cache is not yet implemented")
}

// ResolveAssetPath returns an error as asset resolution from external Go
// modules is not yet supported.
//
// Returns string which is always empty.
// Returns error when called, as this feature is not implemented.
func (*GoModuleCacheResolver) ResolveAssetPath(_ context.Context, _ string, _ string) (string, error) {
	return "", errors.New("asset resolution from Go module cache is not yet implemented")
}

// GetModuleDir implements ResolverPort.GetModuleDir.
// It finds the directory for a Go module in the local module cache.
//
// Takes modulePath (string) which is the Go module path
// (e.g. "piko.sh/piko").
//
// Returns string which is the full path to the module directory.
// Returns error when the module cannot be found or has not been downloaded.
func (gmcr *GoModuleCacheResolver) GetModuleDir(ctx context.Context, modulePath string) (string, error) {
	return gmcr.resolveModuleDir(ctx, modulePath)
}

// FindModuleBoundary implements ResolverPort.FindModuleBoundary.
// It splits an import path into the module path and subpath using the known
// modules from go.mod.
//
// Takes importPath (string) which is a full import path to split.
//
// Returns modulePath (string) which is the Go module portion.
// Returns subpath (string) which is the path within the module.
// Returns error when the import path does not match any known module.
func (gmcr *GoModuleCacheResolver) FindModuleBoundary(_ context.Context, importPath string) (modulePath string, subpath string, err error) {
	return gmcr.findModulePath(importPath)
}

// loadKnownModulesFromGoMod reads the project's go.mod file and extracts all
// required module paths. These are stored in sorted order (longest first) to
// allow greedy prefix matching during import path resolution.
//
// This gives accurate module boundary detection without guesswork, and works
// with any module path format.
//
// Returns error when the go.mod file cannot be read or parsed.
//
// Safe for concurrent use. Acquires the receiver's mutex when storing the
// parsed module list.
func (gmcr *GoModuleCacheResolver) loadKnownModulesFromGoMod(ctx context.Context) error {
	_, l := logger_domain.From(ctx, log)

	goModPath := filepath.Join(gmcr.workingDir, "go.mod")
	if gmcr.workingDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		goModPath = filepath.Join(cwd, "go.mod")
	}

	//nolint:gosec // trusted module system path
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return fmt.Errorf("failed to read go.mod at '%s': %w", goModPath, err)
	}

	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return fmt.Errorf("failed to parse go.mod at '%s': %w", goModPath, err)
	}

	modules := make([]string, 0, len(modFile.Require))
	for _, request := range modFile.Require {
		modules = append(modules, request.Mod.Path)
	}

	slices.SortFunc(modules, func(a, b string) int {
		return cmp.Compare(len(b), len(a))
	})

	gmcr.mu.Lock()
	gmcr.knownModules = modules
	gmcr.mu.Unlock()

	l.Internal("Loaded known modules from go.mod",
		logger_domain.Int("moduleCount", len(modules)),
		logger_domain.String("goModPath", goModPath),
	)

	return nil
}

// recordResolutionDuration records how long a resolution took.
//
// Takes startTime (time.Time) which marks when the resolution began.
func (*GoModuleCacheResolver) recordResolutionDuration(ctx context.Context, startTime time.Time) {
	duration := time.Since(startTime)
	goModuleCacheResolutionDuration.Record(ctx, float64(duration.Milliseconds()))
}

// parseImportPath extracts the module path and file path from an import path.
//
// Takes importPath (string) which is the full import path to split.
//
// Returns modulePath (string) which is the Go module portion of the
// import path.
// Returns filePathInModule (string) which is the file path within the
// module.
// Returns err (error) when the import path does not match any known
// module.
func (gmcr *GoModuleCacheResolver) parseImportPath(ctx context.Context, importPath string) (modulePath, filePathInModule string, err error) {
	ctx, l := logger_domain.From(ctx, log)
	modulePath, filePathInModule, err = gmcr.findModulePath(importPath)
	if err != nil {
		goModuleCacheResolutionErrorCount.Add(ctx, 1)
		l.Trace("Failed to parse module path from import",
			logger_domain.String("importPath", importPath),
			logger_domain.Error(err),
		)
		return "", "", fmt.Errorf("could not parse module path from import '%s': %w", importPath, err)
	}

	l.Trace("Parsed import path",
		logger_domain.String(logKeyModulePath, modulePath),
		logger_domain.String("filePathInModule", filePathInModule),
	)

	return modulePath, filePathInModule, nil
}

// constructAndValidatePKPath builds the full path and checks it has a .pk
// extension.
//
// Takes moduleDir (string) which is the module folder.
// Takes filePathInModule (string) which is the file path within the module.
// Takes importPath (string) which is used for logging.
//
// Returns string which is the validated full path to the .pk file.
// Returns error when the path does not end with a .pk extension.
func (*GoModuleCacheResolver) constructAndValidatePKPath(ctx context.Context, moduleDir, filePathInModule, importPath string) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	absolutePath := filepath.Join(moduleDir, filepath.FromSlash(filePathInModule))

	if !strings.HasSuffix(strings.ToLower(absolutePath), ".pk") {
		goModuleCacheResolutionErrorCount.Add(ctx, 1)
		return "", fmt.Errorf("resolved path is not a .pk file (%s)", absolutePath)
	}

	l.Trace("Resolved PK path from Go module cache",
		logger_domain.String("importPath", importPath),
		logger_domain.String("absolutePath", absolutePath),
	)

	return absolutePath, nil
}

// resolveModuleDir finds the absolute directory path of a Go module in the
// module cache. Results are cached to avoid repeated calls to packages.Load.
//
// The go/packages API respects the project's go.mod and go.sum files, so the
// correct version of each module is used.
//
// Takes modulePath (string) which specifies the import path of the module.
//
// Returns string which is the absolute path to the module directory.
// Returns error when the module cannot be found or loaded.
func (gmcr *GoModuleCacheResolver) resolveModuleDir(ctx context.Context, modulePath string) (string, error) {
	if directory, ok := gmcr.getCachedModuleDir(ctx, modulePath); ok {
		return directory, nil
	}

	ctx, l := logger_domain.From(ctx, log)
	goModuleCacheMissCount.Add(ctx, 1)
	l.Trace("Go module cache miss, loading module", logger_domain.String(logKeyModulePath, modulePath))

	moduleDir, err := gmcr.loadModuleDirFromCache(ctx, modulePath)
	if err != nil {
		return "", fmt.Errorf("loading module %q from Go cache: %w", modulePath, err)
	}

	gmcr.cacheModuleDir(ctx, modulePath, moduleDir)

	return moduleDir, nil
}

// getCachedModuleDir checks if a module directory is in the cache and returns
// it if found.
//
// Takes modulePath (string) which specifies the module to look up.
//
// Returns string which is the cached directory path, or empty if not found.
// Returns bool which is true when the module was found in the cache.
//
// Safe for concurrent use; protected by a read lock.
func (gmcr *GoModuleCacheResolver) getCachedModuleDir(ctx context.Context, modulePath string) (string, bool) {
	gmcr.mu.RLock()
	defer gmcr.mu.RUnlock()

	if directory, ok := gmcr.dirCache[modulePath]; ok {
		goModuleCacheHitCount.Add(ctx, 1)
		_, l := logger_domain.From(ctx, log)
		l.Trace("Go module cache hit", logger_domain.String(logKeyModulePath, modulePath))
		return directory, true
	}
	return "", false
}

// loadModuleDirFromCache loads a module using go/packages and finds its
// directory path.
//
// Takes modulePath (string) which specifies the module to load.
//
// Returns string which is the directory path of the loaded module.
// Returns error when the package cannot be loaded or extraction fails.
func (gmcr *GoModuleCacheResolver) loadModuleDirFromCache(ctx context.Context, modulePath string) (string, error) {
	config := &packages.Config{
		Context:    ctx,
		Mode:       packages.NeedModule,
		Tests:      false,
		Dir:        gmcr.workingDir,
		Logf:       nil,
		Env:        append(os.Environ(), "GOWORK=off"),
		BuildFlags: nil,
		Fset:       nil,
		ParseFile:  nil,
		Overlay:    nil,
	}

	pkgs, err := packages.Load(config, modulePath)
	if err != nil {
		return "", fmt.Errorf("failed to load package '%s': %w", modulePath, err)
	}

	return gmcr.extractModuleDirFromPackages(pkgs, modulePath)
}

// extractModuleDirFromPackages finds the module directory from loaded packages.
//
// Takes pkgs ([]*packages.Package) which contains the loaded package data.
// Takes modulePath (string) which identifies the module to find.
//
// Returns string which is the module directory path.
// Returns error when the packages contain errors or the module data is missing.
func (*GoModuleCacheResolver) extractModuleDirFromPackages(pkgs []*packages.Package, modulePath string) (string, error) {
	if packages.PrintErrors(pkgs) > 0 {
		return "", fmt.Errorf("errors while loading package '%s'", modulePath)
	}

	if len(pkgs) == 0 || pkgs[0].Module == nil {
		return "", fmt.Errorf(
			"module '%s' not found in module cache. "+
				"Ensure the module is listed in go.mod and run: go mod download %s",
			modulePath, modulePath,
		)
	}

	return pkgs[0].Module.Dir, nil
}

// cacheModuleDir stores a module directory in the cache for future lookups.
//
// Takes modulePath (string) which identifies the module to cache.
// Takes moduleDir (string) which specifies the directory path to store.
//
// Safe for concurrent use; protected by a mutex.
func (gmcr *GoModuleCacheResolver) cacheModuleDir(ctx context.Context, modulePath, moduleDir string) {
	_, l := logger_domain.From(ctx, log)

	gmcr.mu.Lock()
	defer gmcr.mu.Unlock()

	gmcr.dirCache[modulePath] = moduleDir
	l.Trace("Cached module directory",
		logger_domain.String(logKeyModulePath, modulePath),
		logger_domain.String(logKeyModuleDir, moduleDir),
	)
}

// findModulePath splits an import path into the module path and the file path
// within the module. It uses definitive module boundaries from go.mod rather
// than heuristics.
//
// This works correctly with any module path format:
//   - Standard: github.com/org/repo
//   - Custom domains: go.uber.org/zap
//   - Vanity imports: gopkg.in/yaml.v2
//   - Nested modules: github.com/org/repo/submodule
//
// Takes importPath (string) which is the full import path to split
// into module and subpath components.
//
// Returns modulePath (string) which is the matched module portion.
// Returns pathInModule (string) which is the path within the module.
// Returns err (error) when the import path does not match any known
// module from go.mod.
//
// Safe for concurrent use; protected by a read lock.
func (gmcr *GoModuleCacheResolver) findModulePath(importPath string) (modulePath, pathInModule string, err error) {
	gmcr.mu.RLock()
	modules := gmcr.knownModules
	gmcr.mu.RUnlock()

	if modules == nil {
		return "", "", errors.New("module list not initialised. Call DetectLocalModule first")
	}

	for _, module := range modules {
		if strings.HasPrefix(importPath, module+pathSeparator) {
			modulePath = module
			pathInModule = strings.TrimPrefix(importPath, module+pathSeparator)
			return modulePath, pathInModule, nil
		}

		if importPath == module {
			return module, "", nil
		}
	}

	return "", "", fmt.Errorf(
		"import path '%s' does not match any module in go.mod. "+
			"Ensure the module is listed in go.mod and run: go mod download",
		importPath,
	)
}
