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

package lsp_domain

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_adapters"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

// ErrNoModuleFound is returned when no go.mod file is found while searching up
// the directory tree from a file path.
var ErrNoModuleFound = errors.New("no go.mod file found")

// ModuleContext holds the resolver and cached entry points for a single Go
// module. Each module (identified by its go.mod location) gets its own context
// to ensure correct import resolution.
type ModuleContext struct {
	// ModuleRoot is the absolute path to the directory that contains go.mod.
	ModuleRoot string

	// ModuleName is the Go module name from go.mod (for example a GitHub-hosted
	// path of the form "example.com/org/app").
	ModuleName string

	// Resolver provides import path resolution for this module.
	Resolver resolver_domain.ResolverPort

	// PathsConfig holds the path settings used for entry point discovery.
	PathsConfig *config.PathsConfig

	// sandboxFactory creates sandboxes for filesystem access. If nil, a
	// no-op sandbox is created as a fallback.
	sandboxFactory safedisk.Factory

	// rootSandbox provides file system access for the module root folder.
	// If nil, a sandbox is created when needed.
	rootSandbox safedisk.Sandbox

	// entryPoints caches the entry points found for this module.
	entryPoints []annotator_dto.EntryPoint

	// entryPointsValid indicates whether the cached entry points are still valid.
	entryPointsValid bool

	// mu guards access to the entry points cache.
	mu sync.RWMutex
}

// ModuleContextOption configures a ModuleContext during construction.
type ModuleContextOption func(*ModuleContext)

// NewModuleContext creates a new module context for the given module root.
// It sets up a chained resolver that can handle both local paths and external
// Go module dependencies.
//
// Takes moduleRoot (string) which is the absolute path to the directory that
// contains go.mod.
// Takes basePathsConfig (*config.PathsConfig) which provides the base path
// settings to clone for this module.
// Takes opts (...ModuleContextOption) which provides optional configuration
// such as WithModuleSandbox for testing.
//
// Returns *ModuleContext which is ready to use for path resolution.
// Returns error when the module cannot be detected or the resolver fails to
// initialise.
func NewModuleContext(ctx context.Context, moduleRoot string, basePathsConfig *config.PathsConfig, opts ...ModuleContextOption) (*ModuleContext, error) {
	ctx, l := logger_domain.From(ctx, log)

	localResolver := resolver_adapters.NewLocalModuleResolver(moduleRoot)
	cacheResolver := resolver_adapters.NewGoModuleCacheResolver()
	resolver := resolver_adapters.NewChainedResolver(localResolver, cacheResolver)

	if err := resolver.DetectLocalModule(ctx); err != nil {
		return nil, fmt.Errorf("failed to detect module at %s: %w", moduleRoot, err)
	}

	moduleConfig := clonePathsConfigForModule(basePathsConfig, moduleRoot)

	mc := &ModuleContext{
		ModuleRoot:  moduleRoot,
		ModuleName:  resolver.GetModuleName(),
		Resolver:    resolver,
		PathsConfig: moduleConfig,
		rootSandbox: nil,
	}

	for _, opt := range opts {
		opt(mc)
	}

	l.Debug("Created module context",
		logger_domain.String("moduleRoot", moduleRoot),
		logger_domain.String("moduleName", mc.ModuleName))

	return mc, nil
}

// GetEntryPoints returns the cached entry points for this module, discovering
// them if necessary.
//
// Returns []annotator_dto.EntryPoint which contains all .pk files in the
// module's pages, emails, and partials directories.
// Returns error when entry point discovery fails.
//
// Safe for concurrent use.
func (mc *ModuleContext) GetEntryPoints(ctx context.Context) ([]annotator_dto.EntryPoint, error) {
	ctx, l := logger_domain.From(ctx, log)

	mc.mu.RLock()
	if mc.entryPointsValid {
		eps := mc.entryPoints
		mc.mu.RUnlock()
		return eps, nil
	}
	mc.mu.RUnlock()

	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.entryPointsValid {
		return mc.entryPoints, nil
	}

	eps, err := mc.discoverEntryPoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovering entry points for module %s: %w", mc.ModuleRoot, err)
	}

	mc.entryPoints = eps
	mc.entryPointsValid = true

	l.Debug("Discovered entry points for module",
		logger_domain.String("moduleRoot", mc.ModuleRoot),
		logger_domain.Int("count", len(eps)))

	return mc.entryPoints, nil
}

// InvalidateEntryPoints marks the cached entry points as stale, forcing
// rediscovery on the next call to GetEntryPoints.
//
// Safe for concurrent use.
func (mc *ModuleContext) InvalidateEntryPoints() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.entryPointsValid = false
	mc.entryPoints = nil
}

// discoverEntryPoints walks the module's source directories and collects all
// .pk entry points.
//
// Returns []annotator_dto.EntryPoint which contains all entry points found.
// Returns error when walking the directory fails.
func (mc *ModuleContext) discoverEntryPoints(ctx context.Context) ([]annotator_dto.EntryPoint, error) {
	paths := mc.PathsConfig
	sourceDirs := map[string]sourceDirInfo{
		"page":    {Dir: *paths.PagesSourceDir, IsPage: true, IsPublic: true},
		"email":   {Dir: *paths.EmailsSourceDir, IsEmail: true, IsPublic: true},
		"partial": {Dir: *paths.PartialsSourceDir, IsPage: false, IsPublic: true},
	}

	var entryPoints []annotator_dto.EntryPoint
	for kind, info := range sourceDirs {
		discovered, err := mc.discoverEntryPointsInDir(ctx, kind, info)
		if err != nil {
			return nil, fmt.Errorf("discovering %s entry points: %w", kind, err)
		}
		entryPoints = append(entryPoints, discovered...)
	}

	return entryPoints, nil
}

// discoverEntryPointsInDir walks a single source directory and collects entry
// points.
//
// Takes kind (string) which describes the directory type for error messages.
// Takes info (sourceDirInfo) which provides the directory path and metadata.
//
// Returns []annotator_dto.EntryPoint which contains the discovered entry
// points.
// Returns error when walking fails.
func (mc *ModuleContext) discoverEntryPointsInDir(ctx context.Context, kind string, info sourceDirInfo) ([]annotator_dto.EntryPoint, error) {
	_, l := logger_domain.From(ctx, log)

	if info.Dir == "" {
		return nil, nil
	}

	absDir := filepath.Join(mc.ModuleRoot, info.Dir)
	if !mc.sourceDirectoryExists(ctx, info.Dir, absDir) {
		return nil, nil
	}

	var entryPoints []annotator_dto.EntryPoint
	err := filepath.WalkDir(absDir, func(currentPath string, d fs.DirEntry, err error) error {
		if err != nil {
			l.Warn("Error during entry point discovery walk",
				logger_domain.Error(err),
				logger_domain.String("path", currentPath))
			return nil
		}

		if !isValidEntryPointFile(d) {
			return nil
		}

		ep := mc.createEntryPoint(currentPath, info)
		entryPoints = append(entryPoints, ep)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk %s directory '%s': %w", kind, absDir, err)
	}

	return entryPoints, nil
}

// sourceDirectoryExists checks if a source directory exists using a sandbox.
//
// Takes relDir (string) which specifies the relative directory to check.
// Takes absDir (string) which specifies the absolute directory for logging.
//
// Returns bool which is true if the directory exists.
func (mc *ModuleContext) sourceDirectoryExists(ctx context.Context, relDir, absDir string) bool {
	_, l := logger_domain.From(ctx, log)

	sandbox := mc.rootSandbox
	if sandbox == nil {
		var err error
		if mc.sandboxFactory != nil {
			sandbox, err = mc.sandboxFactory.Create("lsp-module-source", mc.ModuleRoot, safedisk.ModeReadOnly)
		} else {
			sandbox, err = safedisk.NewNoOpSandbox(mc.ModuleRoot, safedisk.ModeReadOnly)
		}
		if err != nil {
			l.Warn("Failed to create sandbox for module root", logger_domain.Error(err))
			return false
		}
		defer func() { _ = sandbox.Close() }()
	}

	_, statErr := sandbox.Stat(relDir)
	if errors.Is(statErr, fs.ErrNotExist) {
		l.Debug("Source directory does not exist, skipping walk",
			logger_domain.String("dir", absDir))
		return false
	}
	return true
}

// createEntryPoint creates an entry point from a discovered file path.
//
// Takes currentPath (string) which is the discovered file path.
// Takes info (sourceDirInfo) which contains metadata about the source
// directory.
//
// Returns annotator_dto.EntryPoint which contains the full import path and
// flags.
func (mc *ModuleContext) createEntryPoint(currentPath string, info sourceDirInfo) annotator_dto.EntryPoint {
	relPath, err := filepath.Rel(mc.ModuleRoot, currentPath)
	if err != nil {
		relPath = currentPath
	}

	fullImportPath := filepath.ToSlash(filepath.Join(mc.ModuleName, relPath))

	return annotator_dto.EntryPoint{
		Path:     fullImportPath,
		IsPage:   info.IsPage,
		IsEmail:  info.IsEmail,
		IsPublic: info.IsPublic,
	}
}

// WithModuleSandbox sets a custom sandbox for the module context, letting
// mock sandboxes stand in for testing filesystem operations.
//
// Takes sandbox (safedisk.Sandbox) which provides filesystem access for the
// module root directory.
//
// Returns ModuleContextOption which configures the context with the given
// sandbox.
func WithModuleSandbox(sandbox safedisk.Sandbox) ModuleContextOption {
	return func(mc *ModuleContext) {
		mc.rootSandbox = sandbox
	}
}

// WithModuleSandboxFactory sets a factory for creating sandboxes in the module
// context.
//
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access.
//
// Returns ModuleContextOption which configures the context with the given
// factory.
func WithModuleSandboxFactory(factory safedisk.Factory) ModuleContextOption {
	return func(mc *ModuleContext) {
		mc.sandboxFactory = factory
	}
}

// FindGoModRoot searches upward from the given file path to find the nearest
// go.mod file.
//
// Takes filePath (string) which is the starting point for the search.
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access. When nil, a no-op sandbox is used as a fallback.
//
// Returns string which is the absolute path to the folder containing go.mod.
// Returns error when no go.mod file is found or the path is not valid.
func FindGoModRoot(ctx context.Context, filePath string, factory safedisk.Factory) (string, error) {
	_, l := logger_domain.From(ctx, log)

	directory, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("invalid path for go.mod search: %w", err)
	}

	info, err := os.Stat(directory)
	if err != nil {
		return "", fmt.Errorf("cannot stat path '%s': %w", directory, err)
	}
	if !info.IsDir() {
		directory = filepath.Dir(directory)
	}

	for {
		found, walkErr := tryReadGoMod(directory, l, factory)
		if walkErr != nil {
			return "", walkErr
		}
		if found {
			return directory, nil
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", ErrNoModuleFound
		}
		directory = parent
	}
}

// clonePathsConfigForModule creates a copy of the base path settings with
// BaseDir set for a given module root folder.
//
// Takes basePaths (*config.PathsConfig) which is the path settings to copy.
// Takes moduleRoot (string) which is the root folder for the new
// configuration.
//
// Returns *config.PathsConfig which is a new path configuration set up for
// this module.
func clonePathsConfigForModule(basePaths *config.PathsConfig, moduleRoot string) *config.PathsConfig {
	return &config.PathsConfig{
		BaseDir:             &moduleRoot,
		ComponentsSourceDir: basePaths.ComponentsSourceDir,
		PagesSourceDir:      basePaths.PagesSourceDir,
		PartialsSourceDir:   basePaths.PartialsSourceDir,
		EmailsSourceDir:     basePaths.EmailsSourceDir,
		AssetsSourceDir:     basePaths.AssetsSourceDir,
	}
}

// tryReadGoMod checks whether a go.mod file exists in directory and
// can be parsed. It returns true when a valid go.mod was found,
// false when the directory does not contain one, and a non-nil
// error for unexpected failures.
//
// Takes directory (string) which is the directory to check for a
// go.mod file.
// Takes l (logger_domain.Logger) which receives debug and
// warning messages.
// Takes factory (safedisk.Factory) which creates sandboxes for filesystem
// access. When nil, a no-op sandbox is used as a fallback.
//
// Returns bool which is true when a valid go.mod was found.
// Returns error when an unexpected failure occurs.
func tryReadGoMod(directory string, l logger_domain.Logger, factory safedisk.Factory) (bool, error) {
	modPath := filepath.Join(directory, "go.mod")
	modInfo, statErr := os.Stat(modPath) //nolint:gosec // upward walk for go.mod

	if statErr != nil {
		if os.IsNotExist(statErr) {
			return false, nil
		}
		return false, fmt.Errorf("checking for go.mod in %s: %w", directory, statErr)
	}

	if modInfo.IsDir() {
		return false, nil
	}

	var modSandbox safedisk.Sandbox
	var sErr error
	if factory != nil {
		modSandbox, sErr = factory.Create("lsp-module-gomod", directory, safedisk.ModeReadOnly)
	} else {
		modSandbox, sErr = safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	}
	if sErr != nil {
		return false, fmt.Errorf("creating sandbox for go.mod at %s: %w", directory, sErr)
	}
	moduleName, readErr := readModuleNameFromGoMod(modSandbox)
	_ = modSandbox.Close()
	if readErr != nil {
		return false, fmt.Errorf("cannot parse go.mod at %s: %w", modPath, readErr)
	}
	l.Debug("Found go.mod",
		logger_domain.String("path", modPath),
		logger_domain.String("module", moduleName))
	return true, nil
}

// readModuleNameFromGoMod reads the module name from a go.mod file using a
// sandboxed filesystem rooted at the directory containing the go.mod.
//
// Takes sandbox (safedisk.Sandbox) which provides access to the directory
// containing go.mod.
//
// Returns string which is the module name.
// Returns error when the file cannot be read or has no module line.
func readModuleNameFromGoMod(sandbox safedisk.Sandbox) (string, error) {
	data, err := sandbox.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}

	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("scanning go.mod: %w", err)
	}

	return "", errors.New("no 'module' line found in go.mod")
}
