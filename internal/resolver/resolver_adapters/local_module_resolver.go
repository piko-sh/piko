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
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/wdk/safedisk"
)

var _ resolver_domain.ResolverPort = (*LocalModuleResolver)(nil)

// LocalModuleResolver implements the ResolverPort interface as a driven adapter.
// It resolves Piko component and asset import paths within the local project's
// module by interacting with the file system and discovering Go module
// information to enforce module-absolute imports.
type LocalModuleResolver struct {
	// sandboxFactory creates sandboxes when needed. When non-nil, this factory
	// is used instead of safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// startDir is the directory where the search for go.mod starts.
	startDir string

	// baseDir is the directory containing go.mod; empty if no go.mod was found.
	baseDir string

	// moduleName is the Go module name from go.mod; empty if no go.mod was found.
	moduleName string
}

// NewLocalModuleResolver creates a new local module resolver.
//
// The startDir is the directory from which to begin searching for the go.mod
// file, typically the project's root or current working directory.
//
// Takes startDir (string) which specifies the directory to start searching
// from for the go.mod file.
//
// Returns *LocalModuleResolver which is ready to detect the local module.
func NewLocalModuleResolver(startDir string) *LocalModuleResolver {
	return &LocalModuleResolver{
		startDir:   startDir,
		baseDir:    "",
		moduleName: "",
	}
}

// NewLocalModuleResolverWithFactory creates a new local module resolver with
// an optional sandbox factory.
//
// Takes startDir (string) which specifies the directory to start searching
// from for the go.mod file.
// Takes factory (safedisk.Factory) which creates sandboxes when needed.
//
// Returns *LocalModuleResolver which is ready to detect the local module.
func NewLocalModuleResolverWithFactory(startDir string, factory safedisk.Factory) *LocalModuleResolver {
	return &LocalModuleResolver{
		sandboxFactory: factory,
		startDir:       startDir,
		baseDir:        "",
		moduleName:     "",
	}
}

// DetectLocalModule finds the project's root by locating the go.mod file and
// parsing it to determine the module's name and base directory. This method
// must be called before any resolution can occur.
//
// Returns error when the go.mod file cannot be found or parsed.
func (lmr *LocalModuleResolver) DetectLocalModule(ctx context.Context) error {
	ctx, span, l := log.Span(ctx, "LocalModuleResolver.DetectLocalModule",
		logger_domain.String("startDir", lmr.startDir),
	)
	defer span.End()

	moduleDetectionCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		moduleDetectionDuration.Record(ctx, float64(duration.Milliseconds()))
	}()

	modFile, err := findGoMod(lmr.startDir)
	if err != nil {
		moduleDetectionErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to find go.mod file while searching upwards from start directory")
		return fmt.Errorf("error finding go.mod: %w", err)
	}

	if modFile == "" {
		absStartDir, absErr := filepath.Abs(lmr.startDir)
		if absErr != nil {
			moduleDetectionErrorCount.Add(ctx, 1)
			return fmt.Errorf("cannot determine absolute path for start directory: %w", absErr)
		}
		lmr.baseDir = absStartDir
		lmr.moduleName = ""
		l.Internal("No go.mod found, using start directory as base",
			logger_domain.String("baseDir", lmr.baseDir))
		return nil
	}

	lmr.baseDir = filepath.Dir(modFile)
	l.Internal("Found go.mod file", logger_domain.String("path", modFile), logger_domain.String("baseDir", lmr.baseDir))

	name, err := readModuleName(modFile, lmr.sandboxFactory)
	if err != nil {
		moduleDetectionErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to read module name from go.mod")
		return fmt.Errorf("cannot parse module name from %s: %w", modFile, err)
	}
	lmr.moduleName = name

	l.Internal("Detected local module information", logger_domain.String("moduleName", lmr.moduleName), logger_domain.String("baseDir", lmr.baseDir))
	return nil
}

// GetModuleName returns the module name found in the go.mod file.
//
// Returns string which is the module name.
func (lmr *LocalModuleResolver) GetModuleName() string {
	return lmr.moduleName
}

// GetBaseDir returns the absolute path to the directory that contains the
// go.mod file.
//
// Returns string which is the base directory path.
func (lmr *LocalModuleResolver) GetBaseDir() string {
	return lmr.baseDir
}

// ConvertEntryPointPathToManifestKey implements the ResolverPort interface.
// It strips the module name prefix from an entry point path to create a
// project-relative manifest key.
//
// This method is the authoritative converter between the two path formats used
// in the Piko system:
//   - Build-time: module-absolute (e.g., "github.com/org/app/pages/index.pk")
//   - Runtime: project-relative (e.g., "pages/index.pk")
//
// The conversion uses the actual module name from go.mod, making it reliable
// against any valid Go module name format, including those with slashes.
//
// Takes entryPointPath (string) which is the module-absolute path to convert.
//
// Returns string which is the project-relative manifest key, or the original
// path if the module prefix does not match.
func (lmr *LocalModuleResolver) ConvertEntryPointPathToManifestKey(entryPointPath string) string {
	prefix := lmr.moduleName + "/"

	if result, found := strings.CutPrefix(entryPointPath, prefix); found {
		return result
	}

	return entryPointPath
}

// ResolvePKPath resolves a Piko component import path to an absolute file
// system path.
//
// It supports module-absolute paths and the @ alias. Relative paths and
// project-relative paths are not supported.
//
// Resolution order:
//  1. @ alias (e.g., "@/partials/card.pk") - expanded to module-absolute path
//  2. Module-absolute (e.g., "mymodule/partials/card.pk") - resolved via
//     module detection
//
// Takes importPath (string) which is the import path to resolve.
// Takes containingFilePath (string) which is the absolute path of the file
// containing the import statement, used to resolve the @ alias.
//
// Returns string which is the absolute file system path to the component.
// Returns error when the path format is invalid, the module cannot be
// detected, or the resolved path is not a .pk file.
func (lmr *LocalModuleResolver) ResolvePKPath(ctx context.Context, importPath string, containingFilePath string) (string, error) {
	ctx, span, l := log.Span(ctx, "LocalModuleResolver.ResolvePKPath",
		logger_domain.String(logKeyImportPath, importPath),
		logger_domain.String("containingFilePath", containingFilePath),
	)
	defer span.End()
	pathResolutionCount.Add(ctx, 1)

	if lmr.isRemote(importPath) {
		l.Trace("Resolving remote PK path", logger_domain.String("url", importPath))
		return lmr.fetchRemotePK(ctx, importPath)
	}

	expandedPath, err := ExpandModuleAlias(importPath, containingFilePath)
	if err != nil {
		pathResolutionErrorCount.Add(ctx, 1)
		return "", fmt.Errorf("expanding module alias for PK path %q: %w", importPath, err)
	}

	var resolvedPath string

	if lmr.moduleName == "" || !strings.HasPrefix(expandedPath, lmr.moduleName+"/") {
		pathResolutionErrorCount.Add(ctx, 1)
		if lmr.moduleName != "" {
			examplePath := lmr.moduleName + "/path/to/your/component.pk"
			return "", fmt.Errorf(
				"invalid component import path: %q. Piko components must be imported using their full "+
					"module path (e.g., 'import comp %q') or the @ alias (e.g., 'import comp \"@/...\"')",
				importPath,
				examplePath,
			)
		}
		return "", fmt.Errorf("cannot resolve module path '%s': no local module detected", importPath)
	}

	resolvedPath, err = lmr.resolveModulePathInternal(ctx, expandedPath)
	if err != nil {
		pathResolutionErrorCount.Add(ctx, 1)
		return "", fmt.Errorf("resolving PK module path %q: %w", importPath, err)
	}

	if !strings.HasSuffix(strings.ToLower(resolvedPath), ".pk") {
		err := fmt.Errorf("cannot resolve component import '%s': resolved path is not a .pk file (%s)", importPath, resolvedPath)
		pathResolutionErrorCount.Add(ctx, 1)
		return "", err
	}

	return resolvedPath, nil
}

// ResolveCSSPath resolves a CSS import path to an absolute file system path.
// It supports relative paths, module-absolute paths, and the @ alias.
//
// Takes importPath (string) which specifies the CSS import to resolve.
// Takes containingDir (string) which is the directory of the file containing
// the @import statement, used for @ alias expansion.
//
// Returns string which is the absolute file system path to the CSS file.
// Returns error when the path format is invalid or does not end with .css.
func (lmr *LocalModuleResolver) ResolveCSSPath(ctx context.Context, importPath string, containingDir string) (string, error) {
	ctx, span, _ := log.Span(ctx, "LocalModuleResolver.ResolveCSSPath",
		logger_domain.String(logKeyImportPath, importPath),
		logger_domain.String("containingDir", containingDir),
	)
	defer span.End()
	pathResolutionCount.Add(ctx, 1)

	var resolvedPath string
	var err error

	if hasModuleAliasPrefix(importPath) {
		syntheticContainingFile := filepath.Join(containingDir, "style.css")
		expandedPath, expandErr := ExpandModuleAlias(importPath, syntheticContainingFile)
		if expandErr != nil {
			pathResolutionErrorCount.Add(ctx, 1)
			return "", fmt.Errorf("expanding module alias for CSS path %q: %w", importPath, expandErr)
		}
		resolvedPath, err = lmr.resolveModulePathInternal(ctx, expandedPath)
		if err != nil {
			pathResolutionErrorCount.Add(ctx, 1)
			return "", fmt.Errorf("resolving CSS module path %q: %w", importPath, err)
		}
	} else if strings.HasPrefix(importPath, lmr.moduleName) {
		resolvedPath, err = lmr.resolveModulePathInternal(ctx, importPath)
		if err != nil {
			pathResolutionErrorCount.Add(ctx, 1)
			return "", fmt.Errorf("resolving CSS module path %q: %w", importPath, err)
		}
	} else if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
		resolvedPath = filepath.Join(containingDir, filepath.FromSlash(importPath))
	} else {
		err = fmt.Errorf(
			"invalid CSS import path '%s': path must be relative (e.g., './styles.css'), an absolute module path (e.g., '%s/styles/theme.css'), or use the @ alias (e.g., '@/styles/theme.css')",
			importPath, lmr.moduleName)
		pathResolutionErrorCount.Add(ctx, 1)
		return "", err
	}

	if !strings.HasSuffix(strings.ToLower(resolvedPath), ".css") {
		err = fmt.Errorf("cannot resolve stylesheet import '%s': resolved path is not a .css file (%s)", importPath, resolvedPath)
		pathResolutionErrorCount.Add(ctx, 1)
		return "", err
	}

	return resolvedPath, nil
}

// ResolveAssetPath resolves an asset path to an absolute file system path.
// It supports module-absolute paths and the @ alias for piko:svg, piko:img,
// piko:video, and pml-img src attributes.
//
// Takes importPath (string) which is the asset path to resolve, either as a
// full module path (e.g. "mymodule/lib/icons/arrow.svg") or using the @ alias
// (e.g. "@/lib/icons/arrow.svg").
// Takes containingFilePath (string) which is the absolute path of the
// component file containing the asset reference, used to resolve the @ alias
// to the correct module.
//
// Returns string which is the absolute file system path to the asset.
// Returns error when the module alias cannot be expanded or the path cannot
// be resolved.
func (lmr *LocalModuleResolver) ResolveAssetPath(ctx context.Context, importPath string, containingFilePath string) (string, error) {
	ctx, span, l := log.Span(ctx, "LocalModuleResolver.ResolveAssetPath",
		logger_domain.String(logKeyImportPath, importPath),
		logger_domain.String("containingFilePath", containingFilePath),
	)
	defer span.End()
	pathResolutionCount.Add(ctx, 1)

	expandedPath, err := ExpandModuleAlias(importPath, containingFilePath)
	if err != nil {
		pathResolutionErrorCount.Add(ctx, 1)
		return "", fmt.Errorf("expanding module alias for asset path %q: %w", importPath, err)
	}

	resolvedPath, err := lmr.resolveModulePathInternal(ctx, expandedPath)
	if err != nil {
		pathResolutionErrorCount.Add(ctx, 1)
		if !strings.HasPrefix(expandedPath, lmr.moduleName) {
			examplePath := lmr.moduleName + "/lib/path/to/asset.svg"
			err = fmt.Errorf(
				"invalid asset path: \"%s\". Assets must use their full module path (e.g., 'src=\"%s\"') or the @ alias (e.g., 'src=\"@/lib/path/to/asset.svg\"')",
				importPath,
				examplePath,
			)
		}
		l.Trace("Asset path resolution failed", logger_domain.Error(err))
		return "", fmt.Errorf("resolving asset module path %q: %w", importPath, err)
	}

	l.Trace("Resolved asset path",
		logger_domain.String(logKeyImportPath, importPath),
		logger_domain.String("expandedPath", expandedPath),
		logger_domain.String("resolvedPath", resolvedPath),
	)

	return resolvedPath, nil
}

// GetModuleDir returns an error as this resolver does not support external
// modules. Use GoModuleCacheResolver for external module paths.
//
// Returns string which is always empty.
// Returns error when called, as this resolver does not support this operation.
func (*LocalModuleResolver) GetModuleDir(_ context.Context, _ string) (string, error) {
	return "", errors.New("local resolver does not support external module resolution; use GoModuleCacheResolver")
}

// FindModuleBoundary is not supported by the local resolver.
// Module boundary detection requires the GoModuleCacheResolver which has
// access to the full go.mod dependency list.
//
// Returns modulePath (string) which is always empty.
// Returns subpath (string) which is always empty.
// Returns error which always indicates this operation is unsupported.
func (*LocalModuleResolver) FindModuleBoundary(_ context.Context, _ string) (modulePath string, subpath string, err error) {
	return "", "", errors.New("local resolver does not support module boundary detection; use GoModuleCacheResolver")
}

// resolveModulePathInternal is the shared core logic for resolving
// module-absolute paths. It does not check file types, which is left to the
// public methods.
//
// Takes importPath (string) which is the module-absolute path to resolve.
//
// Returns string which is the resolved absolute file system path.
// Returns error when module information is missing or the path does not belong
// to the local module.
func (lmr *LocalModuleResolver) resolveModulePathInternal(_ context.Context, importPath string) (string, error) {
	if lmr.moduleName == "" || lmr.baseDir == "" {
		return "", errors.New("cannot resolve path: module information is missing. Was DetectLocalModule called and did it succeed")
	}

	if relativePath, found := strings.CutPrefix(importPath, lmr.moduleName+"/"); found {
		absolutePath := filepath.Join(lmr.baseDir, filepath.FromSlash(relativePath))
		return absolutePath, nil
	}

	return "", fmt.Errorf("cannot resolve module path '%s': not in local module '%s'", importPath, lmr.moduleName)
}

// isRemote checks if an import path is a URL for a remote component.
//
// Takes path (string) which is the import path to check.
//
// Returns bool which is true if the path starts with http:// or https://.
func (*LocalModuleResolver) isRemote(path string) bool {
	return strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://")
}

// fetchRemotePK is a placeholder for future remote component downloading.
//
// Takes url (string) which specifies the remote location to fetch from.
//
// Returns string which will contain the fetched content when implemented.
// Returns error when called, as remote fetching is not yet supported.
func (*LocalModuleResolver) fetchRemotePK(ctx context.Context, url string) (string, error) {
	ctx, l := logger_domain.From(ctx, log)
	err := errors.New("remote component fetching is not yet implemented")
	l.Error("Attempted to fetch remote PK", logger_domain.String("url", url), logger_domain.Error(err))
	pathResolutionErrorCount.Add(ctx, 1)
	return "", err
}

// findGoMod searches upward through the directory tree to find a go.mod file.
//
// Takes start (string) which is the directory or file path to begin from.
//
// Returns string which is the path to the go.mod file, or empty if not found.
// Returns error when the start path is invalid or cannot be accessed.
func findGoMod(start string) (string, error) {
	directory, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("invalid start path for go.mod search: %w", err)
	}

	startInfo, err := os.Stat(directory) //nolint:gosec // upward walk for go.mod
	if err != nil {
		return "", fmt.Errorf("cannot stat start path '%s': %w", directory, err)
	}
	if !startInfo.IsDir() {
		directory = filepath.Dir(directory)
	}

	for {
		modPath := filepath.Join(directory, "go.mod")
		info, err := os.Stat(modPath) //nolint:gosec // upward walk for go.mod

		if err == nil && !info.IsDir() {
			return modPath, nil
		}

		if err != nil && !os.IsNotExist(err) {
			return "", fmt.Errorf("checking for go.mod at %q: %w", modPath, err)
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			return "", nil
		}
		directory = parent
	}
}

// readModuleName reads a go.mod file and returns the module name from the
// "module ..." line.
//
// Takes modFile (string) which is the path to the go.mod file.
//
// Returns string which is the module name found in the file.
// Returns error when the file cannot be opened, read, or does not contain a
// module line.
func readModuleName(modFile string, factory safedisk.Factory) (string, error) {
	modDir := filepath.Dir(modFile)
	var sandbox safedisk.Sandbox
	var sErr error
	if factory != nil {
		sandbox, sErr = factory.Create("go-mod", modDir, safedisk.ModeReadOnly)
	} else {
		sandbox, sErr = safedisk.NewNoOpSandbox(modDir, safedisk.ModeReadOnly)
	}
	if sErr != nil {
		return "", fmt.Errorf("creating sandbox for go.mod at %q: %w", modFile, sErr)
	}
	defer func() { _ = sandbox.Close() }()

	data, err := sandbox.ReadFile("go.mod")
	if err != nil {
		return "", fmt.Errorf("reading go.mod file %q: %w", modFile, err)
	}

	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("scanning go.mod file %q: %w", modFile, err)
	}

	return "", fmt.Errorf("no 'module' line found in %s", modFile)
}
