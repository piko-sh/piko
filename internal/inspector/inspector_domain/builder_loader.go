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

package inspector_domain

// This file is responsible for wrapping the complex 'golang.org/x/tools/go/packages'
// library, providing a clean interface for loading and type-checking Go packages
// from a virtual source code overlay.

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/quickpackages"
)

// standardLoaderMode is the set of packages.Load mode flags used by the
// standard fallback loader. This replicates what quickpackages provides
// internally but uses the Go team's maintained loading pipeline.
const standardLoaderMode = packages.NeedName |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedImports |
	packages.NeedDeps |
	packages.NeedModule |
	packages.NeedFiles

// loadPackagesFromSource wraps the Go packages loader as a pure function that
// takes all configuration and source data as arguments. This is the core logic
// for the default builderPackageLoader implementation.
//
// Takes inspectorConfig (inspector_dto.Config) which specifies the
// package loading settings including module name and build options.
// Takes overlay (map[string][]byte) which provides in-memory file
// contents to use instead of reading from disk.
//
// Returns []*packages.Package which contains the loaded and validated packages.
// Returns error when packages cannot be loaded or contain errors.
func loadPackagesFromSource(ctx context.Context, inspectorConfig inspector_dto.Config, overlay map[string][]byte) ([]*packages.Package, error) {
	ctx, span, l := log.Span(ctx, "loadPackagesFromSource")
	defer span.End()

	packagesConfig := buildPackagesConfig(inspectorConfig, overlay)
	patterns := getLoadPatterns(inspectorConfig.ModuleName, overlay)

	l.Internal("Calling packages.Load with final configuration...")
	BuilderPackageLoadCount.Add(ctx, 1)
	startTime := time.Now()

	var loadedPackages []*packages.Package
	var err error
	if inspectorConfig.UseStandardLoader {
		l.Internal("Using standard packages.Load (fallback mode)")
		loadedPackages, err = loadWithStandardLoader(packagesConfig, patterns)
	} else {
		loadedPackages, err = quickpackages.Load(packagesConfig, patterns...)
	}

	duration := time.Since(startTime)
	BuilderPackageLoadDuration.Record(ctx, float64(duration.Milliseconds()))

	if err != nil {
		l.Error("packages.Load returned an immediate error", logger_domain.Error(err))
		BuilderPackageLoadErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("loading packages: %w", err)
	}
	l.Internal("packages.Load call completed.")

	if err := aggregatePackageErrors(ctx, loadedPackages); err != nil {
		BuilderPackageLoadErrorCount.Add(ctx, 1)
		return nil, fmt.Errorf("aggregating package errors: %w", err)
	}

	loadedPackages = filterValidPackages(ctx, loadedPackages)

	return loadedPackages, nil
}

// loadWithStandardLoader uses golang.org/x/tools/go/packages.Load
// as a stable fallback.
//
// Takes cfg (*packages.Config) which holds the package loading
// configuration.
// Takes patterns ([]string) which lists the package patterns to
// load.
//
// Returns []*packages.Package which contains the loaded packages.
// Returns error when package loading fails.
func loadWithStandardLoader(cfg *packages.Config, patterns []string) ([]*packages.Package, error) {
	cfg.Mode = standardLoaderMode
	return packages.Load(cfg, patterns...)
}

// buildPackagesConfig creates the base settings for package loading. It is a
// pure helper function that takes all needed data as arguments.
//
// When using quickpackages, Mode/ParseFile/Fset are handled internally.
// When using the standard loader, loadWithStandardLoader sets the Mode.
//
// Takes inspectorConfig (inspector_dto.Config) which sets the base folder, build
// flags, and environment values for package loading.
// Takes overlay (map[string][]byte) which provides file contents held in
// memory that replace files on disk.
//
// Returns *packages.Config which is ready for use with quickpackages.Load.
func buildPackagesConfig(inspectorConfig inspector_dto.Config, overlay map[string][]byte) *packages.Config {
	overlayCopy := make(map[string][]byte, len(overlay))
	maps.Copy(overlayCopy, overlay)

	packagesConfig := &packages.Config{
		Dir:        inspectorConfig.BaseDir,
		Env:        os.Environ(),
		BuildFlags: inspectorConfig.BuildFlags,
		Overlay:    overlayCopy,
	}

	if inspectorConfig.GOCACHE != "" {
		packagesConfig.Env = append(packagesConfig.Env, "GOCACHE="+inspectorConfig.GOCACHE)
	}
	if inspectorConfig.GOMODCACHE != "" {
		packagesConfig.Env = append(packagesConfig.Env, "GOMODCACHE="+inspectorConfig.GOMODCACHE)
	}
	if inspectorConfig.GOOS != "" {
		packagesConfig.Env = append(packagesConfig.Env, "GOOS="+inspectorConfig.GOOS)
	}
	if inspectorConfig.GOARCH != "" {
		packagesConfig.Env = append(packagesConfig.Env, "GOARCH="+inspectorConfig.GOARCH)
	}

	packagesConfig.Env = append(packagesConfig.Env, "GOTOOLCHAIN=auto", "GOWORK=off")

	return packagesConfig
}

// getLoadPatterns returns the load patterns for packages.Load.
//
// When moduleName is set, returns the standard Go module pattern to load all
// packages. When moduleName is empty, returns the file paths from the overlay
// for projects without a go.mod file.
//
// Takes moduleName (string) which is the module name, or empty if there is
// none.
// Takes overlay (map[string][]byte) which holds the file contents to load.
//
// Returns []string which contains the patterns or file paths to load.
func getLoadPatterns(moduleName string, overlay map[string][]byte) []string {
	if moduleName != "" {
		return []string{"./..."}
	}

	filePaths := make([]string, 0, len(overlay))
	for path := range overlay {
		filePaths = append(filePaths, path)
	}
	return filePaths
}

// aggregatePackageErrors checks all loaded packages and gathers any errors
// into one combined error.
//
// Only errors from root (initial) packages are treated as fatal. For
// dependency packages, only ListError (missing modules, version conflicts)
// is aggregated. TypeError and ParseError from dependencies are skipped
// because quickpackages does not run the CGo preprocessor, so CGo
// dependency packages commonly have undefined-type errors that are harmless
// and will cascade to root packages if they genuinely matter.
//
// Takes loadedPackages ([]*packages.Package) which contains the root packages
// to check (including their transitive dependency graph).
//
// Returns error when one or more packages contain errors.
func aggregatePackageErrors(ctx context.Context, loadedPackages []*packages.Package) error {
	ctx, l := logger_domain.From(ctx, log)
	var allErrors []string

	rootPaths := make(map[string]struct{}, len(loadedPackages))
	for _, pkg := range loadedPackages {
		rootPaths[pkg.PkgPath] = struct{}{}
	}

	packages.Visit(loadedPackages, nil, func(pkg *packages.Package) {
		if len(pkg.Errors) > 0 {
			l.Internal("Found potential errors in loaded package", logger_domain.String("package", pkg.PkgPath), logger_domain.Int("error_count", len(pkg.Errors)))
		}
		_, isRoot := rootPaths[pkg.PkgPath]
		for _, err := range pkg.Errors {
			message := err.Error()
			if isIgnorablePackageError(message) {
				l.Internal("Ignoring benign package error",
					logger_domain.String("package", pkg.PkgPath),
					logger_domain.String("error", message))
				continue
			}

			if !isRoot && err.Kind != packages.ListError {
				l.Internal("Skipping non-root package error",
					logger_domain.String("package", pkg.PkgPath),
					logger_domain.String("error", message),
					logger_domain.String("kind", errorKindName(err.Kind)))
				continue
			}
			allErrors = append(allErrors, message)
			l.Warn("Package error detail", logger_domain.String("package", pkg.PkgPath), logger_domain.String("error", message))
		}
	})

	if len(allErrors) > 0 {
		l.Error("Found one or more errors across all loaded packages.", logger_domain.Int("total_errors", len(allErrors)))
		return fmt.Errorf("errors found during package loading: %s", strings.Join(allErrors, "; "))
	}

	l.Internal("No errors found in any loaded packages.")
	return nil
}

// errorKindName returns a human-readable name for a
// packages.ErrorKind value.
//
// Takes kind (packages.ErrorKind) which is the error kind to name.
//
// Returns string which is the human-readable name of the error
// kind.
func errorKindName(kind packages.ErrorKind) string {
	switch kind {
	case packages.ListError:
		return "ListError"
	case packages.ParseError:
		return "ParseError"
	case packages.TypeError:
		return "TypeError"
	default:
		return "UnknownError"
	}
}

// filterValidPackages removes packages that could not be loaded properly from
// the result set. This handles cases where packages.Load returns package
// objects with empty names due to build constraints excluding all files or
// directories having no Go files.
//
// Takes pkgs ([]*packages.Package) which is the loaded package list.
//
// Returns []*packages.Package with invalid packages removed.
func filterValidPackages(ctx context.Context, pkgs []*packages.Package) []*packages.Package {
	ctx, l := logger_domain.From(ctx, log)
	valid := make([]*packages.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg.Name == "" {
			l.Internal("Filtering out package with empty name",
				logger_domain.String("package", pkg.PkgPath))
			continue
		}
		valid = append(valid, pkg)
	}
	return valid
}

// isIgnorablePackageError checks whether an error message is a harmless
// package loading error that can be safely ignored.
//
// Ignored errors include:
//   - "and not used": benign in virtual Go files where imports may be added for
//     type resolution. Matches both "imported and not used" and
//     "imported as X and not used" formats.
//   - "no Go files in": a directory was included as a pattern but contains no
//     Go source files. This occurs when the project root or a scaffold directory
//     has no compilable files.
//   - "build constraints exclude all Go files in": a directory has Go files but
//     all are excluded by build tags. Common in test fixtures and scaffold
//     directories.
//
// Takes message (string) which is the error message to check.
//
// Returns bool which is true if the error can be safely ignored.
func isIgnorablePackageError(message string) bool {
	if strings.Contains(message, "and not used") {
		return true
	}
	if strings.Contains(message, "no Go files in") {
		return true
	}
	if strings.Contains(message, "build constraints exclude all Go files in") {
		return true
	}

	if strings.Contains(message, "undefined: _C") || strings.Contains(message, "undefined: _Ctype") {
		return true
	}
	return false
}
