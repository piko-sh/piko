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

package templater_adapters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/build"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/trace"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// generatedGoVariantID is the variant identifier for generated Go code.
	generatedGoVariantID = "go_generated"

	// logFieldArtefactID is the field name for artefact IDs in log messages.
	logFieldArtefactID = "artefactID"

	// logFieldArtefactIDKey is the logging field key for artefact IDs.
	logFieldArtefactIDKey = "artefact_id"

	// logFieldDir is the log field key for directory paths.
	logFieldDir = "dir"

	// logFieldOpen is the operation name used in fs.PathError when opening fails.
	logFieldOpen = "open"

	// logFieldPackagePath is the log field key for package path values.
	logFieldPackagePath = "pkg_path"

	// logFieldRealPath is the log field key for resolved file system paths.
	logFieldRealPath = "real_path"

	// generatedGoFilename is the filename used for virtual Go files in the VFS.
	generatedGoFilename = "generated.go"

	// srcDirName is the subdirectory name for Go source files within paths.
	srcDirName = "src"

	// srcPathPrefix is the src folder name with a trailing path separator.
	srcPathPrefix = srcDirName + pathSeparator

	// srcPathSuffix is the path suffix to trim from GOPATH and GOROOT paths.
	srcPathSuffix = pathSeparator + srcDirName

	// defaultFileSizeEstimate is the assumed file size in bytes for generated
	// files when the actual size is not available.
	defaultFileSizeEstimate = 1024

	// virtualDirPermissions is the file mode for virtual directories.
	virtualDirPermissions = 0755

	// virtualFilePermissions is the permission bits for virtual files in the
	// registry.
	virtualFilePermissions = 0644
)

// RegistryVFSAdapter provides a virtual filesystem for the Go compiler that
// serves generated code from the registry instead of the real filesystem.
type RegistryVFSAdapter struct {
	// ctx carries logging context for trace and request ID
	// propagation, retained on the struct because the
	// adapter implements fs.FS and go/build.Context
	// callbacks whose signatures cannot accept a context
	// parameter.
	ctx context.Context

	// registryService retrieves and fetches artefact data from the registry.
	registryService registry_domain.RegistryService

	// projectSandbox provides sandboxed filesystem access to the project root.
	// When non-nil, user package resolution uses this instead of raw os calls.
	projectSandbox safedisk.Sandbox

	// pathMap maps package paths to artefact IDs.
	pathMap map[string]string

	// freshArtefacts is an in-memory cache of freshly generated code (artefactID
	// -> content).
	freshArtefacts map[string][]byte

	// goPath is the virtual GOPATH prefix used to resolve file paths.
	goPath string

	// goRoot is the path prefix for standard library packages.
	goRoot string

	// modulePath is the Go module path for the project (for example a
	// GitHub-hosted module path).
	modulePath string

	// projectRoot is the absolute path to the project root folder.
	projectRoot string

	// mu guards access to pathMap and fileCache for safe concurrent use.
	mu sync.RWMutex
}

// NewRegistryVFSAdapter creates a virtual filesystem adapter for serving
// generated code. The modulePath and projectSandbox parameters enable the VFS
// to redirect user package imports (imports that match the module path but are
// not generated code) to the real filesystem via a sandboxed reader.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes registryService (registry_domain.RegistryService) which provides access
// to the code registry.
// Takes goPath (string) which specifies the virtual GOPATH root.
// Takes goRoot (string) which specifies the virtual GOROOT root.
// Takes modulePath (string) which identifies the module for import redirection.
// Takes projectSandbox (safedisk.Sandbox) which provides sandboxed filesystem
// access to the project root. May be nil to disable user package resolution.
//
// Returns *RegistryVFSAdapter which is the configured adapter ready for use.
// Returns error when registryService is nil or when goPath or goRoot are empty.
func NewRegistryVFSAdapter(
	ctx context.Context,
	registryService registry_domain.RegistryService,
	goPath, goRoot string,
	modulePath string,
	projectSandbox safedisk.Sandbox,
) (*RegistryVFSAdapter, error) {
	if registryService == nil {
		return nil, errors.New("registryService cannot be nil")
	}
	if goPath == "" {
		return nil, errors.New("virtual GOPATH cannot be empty")
	}
	if goRoot == "" {
		return nil, errors.New("virtual GOROOT cannot be empty")
	}

	var projectRoot string
	if projectSandbox != nil {
		projectRoot = projectSandbox.Root()
	}

	return &RegistryVFSAdapter{
		ctx:             ctx,
		registryService: registryService,
		pathMap:         make(map[string]string),
		freshArtefacts:  make(map[string][]byte),
		goPath:          filepath.ToSlash(filepath.Join(goPath, srcDirName)),
		goRoot:          filepath.ToSlash(filepath.Join(goRoot, srcDirName)),
		modulePath:      modulePath,
		projectRoot:     projectRoot,
		projectSandbox:  projectSandbox,
		mu:              sync.RWMutex{},
	}, nil
}

// UpdateMap replaces the path-to-artefact mapping with a new snapshot.
//
// Takes newMap (map[string]string) which provides the complete mapping of
// package paths to artefact IDs.
//
// Safe for concurrent use; protected by a mutex.
func (vfs *RegistryVFSAdapter) UpdateMap(newMap map[string]string) {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	vfs.pathMap = newMap
}

// UpdateFreshArtefacts updates the in-memory cache of freshly
// generated artefacts, giving the interpreter access to
// generated code that has not yet been stored in the registry.
// The cache maps artefact IDs (source paths) to their
// generated Go code content.
//
// Takes artefacts ([]*generator_dto.GeneratedArtefact) which contains the
// freshly generated artefacts to cache.
// Takes projectRoot (string) which is the absolute path to the project root
// for computing relative paths.
//
// Returns error which is currently always nil.
//
// Safe for concurrent use. Access is serialised by an internal
// mutex.
func (vfs *RegistryVFSAdapter) UpdateFreshArtefacts(artefacts []*generator_dto.GeneratedArtefact, projectRoot string) error {
	vfs.mu.Lock()
	defer vfs.mu.Unlock()

	vfs.freshArtefacts = make(map[string][]byte)

	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("Updating VFS fresh artefacts cache...", logger_domain.Int("artefact_count", len(artefacts)))

	for _, artefact := range artefacts {
		vc, _ := generator_domain.GetMainComponent(artefact.Result)
		if vc == nil {
			continue
		}

		relPath, err := filepath.Rel(projectRoot, vc.Source.SourcePath)
		if err != nil {
			l.Error("Failed to compute relative path for fresh artefact",
				logger_domain.String("absolutePath", vc.Source.SourcePath),
				logger_domain.String("projectRoot", projectRoot),
				logger_domain.Error(err))
			continue
		}
		relPath = filepath.ToSlash(relPath)

		vfs.freshArtefacts[relPath] = artefact.Content

		l.Trace("Cached fresh artefact",
			logger_domain.String(logFieldArtefactIDKey, relPath),
			logger_domain.String("canonical_path", vc.CanonicalGoPackagePath),
			logger_domain.Int("content_size", len(artefact.Content)))
	}

	l.Trace("VFS fresh artefacts cache updated successfully", logger_domain.Int("cached_artefacts", len(vfs.freshArtefacts)))
	return nil
}

// OpenFile opens a file from the virtual file system, serving generated code
// from the registry. It implements the build.Context.OpenFile interface for
// the Go compiler.
//
// Takes path (string) which specifies the file path to open.
//
// Returns io.ReadCloser which provides access to the file content.
// Returns error when the path is not found in the VFS map or registry.
//
// Safe for concurrent use; uses a read lock when accessing the path map.
func (vfs *RegistryVFSAdapter) OpenFile(ctx context.Context, path string) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "RegistryVFSAdapter.OpenFile",
		logger_domain.String(logFieldPath, path),
	)
	defer span.End()

	l.Trace("[VFS-OPEN] Resolving import path",
		logger_domain.String("interp_path", path),
		logger_domain.String("goPath_prefix", vfs.goPath),
		logger_domain.String("goRoot_prefix", vfs.goRoot))

	packagePath := vfs.normalisePath(path)
	l.Trace("[VFS-OPEN] Path normalised",
		logger_domain.String("interp_path", path),
		logger_domain.String("normalised_pkg_path", packagePath))

	vfs.mu.RLock()
	artefactID, ok := vfs.pathMap[packagePath]
	if !ok {
		vfs.mu.RUnlock()
		if relPath := vfs.resolveUserPackagePath(packagePath, path); relPath != "" {
			l.Trace("[VFS-OPEN] Resolved user package via project sandbox",
				logger_domain.String(logFieldPackagePath, packagePath),
				logger_domain.String(logFieldRealPath, relPath))
			return vfs.projectSandbox.Open(relPath)
		}
		l.Warn("[VFS-OPEN] Path not found in VFS map",
			logger_domain.String("interp_path", path),
			logger_domain.String("normalised_pkg_path", packagePath),
			logger_domain.Int("available_mappings", len(vfs.pathMap)))
		return nil, os.ErrNotExist
	}

	l.Trace("[VFS-OPEN] Path found in VFS map",
		logger_domain.String("normalised_pkg_path", packagePath),
		logger_domain.String(logFieldArtefactIDKey, artefactID))

	if content := vfs.checkFreshCache(ctx, artefactID, packagePath); content != nil {
		vfs.mu.RUnlock()
		return io.NopCloser(bytes.NewReader(content)), nil
	}
	vfs.mu.RUnlock()

	return vfs.fetchFromRegistry(ctx, span, artefactID)
}

// GetBuildContext creates a custom build.Context configured to use this VFS
// adapter for file resolution. This context is designed for use with
// incremental interpreters.
//
// Returns *build.Context which is configured for GOPATH resolution with VFS
// file operations and real filesystem fallback.
func (vfs *RegistryVFSAdapter) GetBuildContext() *build.Context {
	customContext := build.Default
	customContext.GOPATH = strings.TrimSuffix(vfs.goPath, srcPathSuffix)
	customContext.GOROOT = strings.TrimSuffix(vfs.goRoot, srcPathSuffix)
	customContext.Dir = ""

	customContext.UseAllFiles = true
	customContext.CgoEnabled = false

	customContext.OpenFile = vfs.buildContextOpenFile
	customContext.IsDir = vfs.buildContextIsDir
	customContext.HasSubdir = vfs.buildContextHasSubdir
	customContext.ReadDir = vfs.buildContextReadDir

	return &customContext
}

// Open implements the fs.FS interface.
// This is the main entry point for the interpreter's file system access.
//
// Takes name (string) which specifies the path to open.
//
// Returns fs.File which provides access to the requested file or folder.
// Returns error when the path cannot be found or opened.
//
// Safe for concurrent use; takes a read lock during path lookup.
func (vfs *RegistryVFSAdapter) Open(name string) (fs.File, error) {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[VFS-FS] Open called",
		logger_domain.String("name", name))

	packagePath := vfs.normalisePathForFS(name)

	l.Trace("[VFS-FS] Normalised path",
		logger_domain.String("original", name),
		logger_domain.String("normalised", packagePath))

	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	if artefactID, ok := vfs.pathMap[packagePath]; ok {
		l.Trace("[VFS-FS] Path is a package, serving file content",
			logger_domain.String(logFieldPath, packagePath),
			logger_domain.String(logFieldArtefactIDKey, artefactID))
		return vfs.openPackageAsFile(name, artefactID)
	}

	if vfs.isParentDirectory(packagePath) {
		l.Trace("[VFS-FS] Returning virtual directory for parent path",
			logger_domain.String(logFieldPath, packagePath))
		return &virtualDir{
			name:    filepath.Base(packagePath),
			path:    packagePath,
			vfs:     vfs,
			entries: nil,
			offset:  0,
			closed:  false,
		}, nil
	}

	if relPath := vfs.resolveUserPackagePath(packagePath, name); relPath != "" {
		l.Trace("[VFS-FS] Resolved user package via project sandbox",
			logger_domain.String(logFieldPackagePath, packagePath),
			logger_domain.String(logFieldRealPath, relPath))
		return vfs.projectSandbox.Open(relPath)
	}

	l.Trace("[VFS-FS] Path not in VFS, trying real filesystem",
		logger_domain.String(logFieldPath, packagePath))
	//nolint:gosec // trusted build system path
	return os.Open(name)
}

// Stat implements the fs.StatFS interface.
//
// Takes name (string) which is the path to check.
//
// Returns fs.FileInfo which holds file details for the given path.
// Returns error when the path does not exist in either the virtual or real
// file system.
//
// Safe for concurrent use; protected by a read lock on the VFS mutex.
func (vfs *RegistryVFSAdapter) Stat(name string) (fs.FileInfo, error) {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[VFS-STAT] Stat called",
		logger_domain.String("name", name))

	packagePath := vfs.normalisePathForFS(name)

	l.Trace("[VFS-STAT] Normalised path",
		logger_domain.String("original", name),
		logger_domain.String("normalised", packagePath))

	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	if vfs.isDirectory(packagePath) {
		l.Trace("[VFS-STAT] Returning directory info",
			logger_domain.String(logFieldPath, packagePath))
		return &virtualFileInfo{
			name:  filepath.Base(packagePath),
			size:  0,
			isDir: true,
		}, nil
	}

	artefactID, ok := vfs.pathMap[packagePath]
	if !ok {
		if relPath := vfs.resolveUserPackagePath(packagePath, name); relPath != "" {
			l.Trace("[VFS-STAT] Resolved user package via project sandbox",
				logger_domain.String(logFieldPackagePath, packagePath),
				logger_domain.String(logFieldRealPath, relPath))
			return vfs.projectSandbox.Stat(relPath)
		}
		l.Trace("[VFS-STAT] Not in VFS, trying real filesystem",
			logger_domain.String(logFieldPath, packagePath))
		return os.Stat(name)
	}

	var size int64
	if content, hasFresh := vfs.freshArtefacts[artefactID]; hasFresh {
		size = int64(len(content))
	} else {
		size = defaultFileSizeEstimate
	}

	l.Trace("[VFS-STAT] Returning file info",
		logger_domain.String(logFieldArtefactIDKey, artefactID),
		logger_domain.Int64("size", size))

	return &virtualFileInfo{
		name:  generatedGoFilename,
		size:  size,
		isDir: false,
	}, nil
}

// ReadDir implements fs.ReadDirFS interface.
// Lists all files/subdirectories in the given directory.
//
// Takes name (string) which specifies the directory path to list.
//
// Returns []fs.DirEntry which contains the directory entries.
// Returns error when reading the directory fails.
//
// Safe for concurrent use; protected by a read lock on the VFS mutex.
func (vfs *RegistryVFSAdapter) ReadDir(name string) ([]fs.DirEntry, error) {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[VFS-READDIR] ReadDir called",
		logger_domain.String("name", name))

	packagePath := vfs.normalisePathForFS(name)

	l.Trace("[VFS-READDIR] Normalised path",
		logger_domain.String("original", name),
		logger_domain.String("normalised", packagePath))

	vfs.mu.RLock()
	defer vfs.mu.RUnlock()

	if entries, ok := vfs.readDirPackage(packagePath); ok {
		return entries, nil
	}

	entries := vfs.listSubdirectories(packagePath)

	if len(entries) == 0 {
		return vfs.readDirFallback(packagePath, name)
	}

	l.Trace("[VFS-READDIR] Returning directory entries",
		logger_domain.String(logFieldPath, packagePath),
		logger_domain.Int("count", len(entries)))

	return entries, nil
}

// checkFreshCache looks for a recent artefact in the memory cache.
//
// Takes ctx (context.Context) which carries the logger and tracing data.
// Takes artefactID (string) which identifies the artefact to find.
// Takes packagePath (string) which specifies the package path for logging.
//
// Returns []byte which contains the cached content, or nil if not found.
func (vfs *RegistryVFSAdapter) checkFreshCache(ctx context.Context, artefactID, packagePath string) []byte {
	ctx, l := logger_domain.From(ctx, log)
	if freshContent, hasFresh := vfs.freshArtefacts[artefactID]; hasFresh {
		l.Trace("VFS serving fresh artefact from in-memory cache",
			logger_domain.String("packagePath", packagePath),
			logger_domain.String(logFieldArtefactID, artefactID),
			logger_domain.Int("content_size", len(freshContent)))
		return freshContent
	}
	return nil
}

// fetchFromRegistry retrieves an artefact from the registry service.
//
// Takes span (trace.Span) which provides tracing context.
// Takes artefactID (string) which identifies the artefact to fetch.
//
// Returns io.ReadCloser which provides access to the artefact's Go variant
// data.
// Returns error when the artefact cannot be found or lacks the required Go
// variant.
func (vfs *RegistryVFSAdapter) fetchFromRegistry(
	ctx context.Context,
	span trace.Span,
	artefactID string,
) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("VFS cache miss. Retrieving artefact from registry.",
		logger_domain.String(logFieldArtefactID, artefactID))

	artefact, err := vfs.registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		l.ReportError(span, err, "Failed to get artefact from registry for VFS open.",
			logger_domain.String(logFieldArtefactID, artefactID))
		return nil, fmt.Errorf("VFS: failed to get artefact '%s': %w", artefactID, err)
	}

	goVariant := vfs.findGoVariant(artefact)
	if goVariant == nil {
		err := fmt.Errorf("artefact '%s' found but is missing its '%s' variant", artefactID, generatedGoVariantID)
		l.ReportError(span, err, "Could not find generated Go variant for artefact.")
		return nil, err
	}

	return vfs.readVariantData(ctx, span, artefactID, goVariant)
}

// findGoVariant locates the generated Go variant in an artefact.
//
// Takes artefact (*registry_dto.ArtefactMeta) which contains the variants to
// search.
//
// Returns *registry_dto.Variant which is the Go variant, or nil if not found.
func (*RegistryVFSAdapter) findGoVariant(artefact *registry_dto.ArtefactMeta) *registry_dto.Variant {
	for i := range artefact.ActualVariants {
		if artefact.ActualVariants[i].VariantID == generatedGoVariantID {
			return &artefact.ActualVariants[i]
		}
	}
	return nil
}

// readVariantData reads the data for a variant and returns it as a stream.
//
// Takes span (trace.Span) which provides tracing context for the operation.
// Takes artefactID (string) which identifies the artefact being read.
// Takes goVariant (*registry_dto.Variant) which specifies which variant to
// read.
//
// Returns io.ReadCloser which provides the variant data as a readable stream.
// Returns error when the variant data cannot be fetched or read.
func (vfs *RegistryVFSAdapter) readVariantData(
	ctx context.Context,
	span trace.Span,
	artefactID string,
	goVariant *registry_dto.Variant,
) (io.ReadCloser, error) {
	ctx, l := logger_domain.From(ctx, log)
	stream, err := vfs.registryService.GetVariantData(ctx, goVariant)
	if err != nil {
		l.ReportError(span, err, "Failed to get variant data stream.",
			logger_domain.String("variantID", goVariant.VariantID),
			logger_domain.String("storageKey", goVariant.StorageKey))
		return nil, fmt.Errorf("VFS: failed to get data for variant '%s': %w", goVariant.VariantID, err)
	}

	data, err := io.ReadAll(stream)
	_ = stream.Close()
	if err != nil {
		return nil, fmt.Errorf("VFS: failed to read variant data stream: %w", err)
	}

	l.Trace("Successfully providing generated code stream to interpreter.",
		logger_domain.String(logFieldArtefactID, artefactID),
		logger_domain.Int("size", len(data)))

	return io.NopCloser(bytes.NewReader(data)), nil
}

// normalisePath converts a file path to its standard package path.
//
// Takes path (string) which is the file path to convert.
//
// Returns string which is the directory part of the path with the virtual
// GOPATH, GOROOT, or .piko-gopath prefixes removed.
func (vfs *RegistryVFSAdapter) normalisePath(path string) string {
	p := filepath.ToSlash(path)

	var packagePath string
	if result, found := strings.CutPrefix(p, vfs.goPath); found {
		packagePath = result
	} else if result, found := strings.CutPrefix(p, vfs.goRoot); found {
		packagePath = result
	} else if _, after, found := strings.Cut(p, ".piko-gopath/src/"); found {
		packagePath = after
	} else {
		return p
	}

	packagePath = strings.Trim(packagePath, pathSeparator)
	packagePath = strings.TrimPrefix(packagePath, srcPathPrefix)
	packagePath = strings.TrimPrefix(packagePath, srcDirName)
	packagePath = strings.Trim(packagePath, pathSeparator)

	return filepath.Dir(packagePath)
}

// buildContextOpenFile implements build.Context.OpenFile for the interpreter.
// It checks the VFS first, then falls back to the real file system.
//
// Takes path (string) which specifies the file path to open.
//
// Returns io.ReadCloser which provides access to the file contents.
// Returns error when the file cannot be opened from either source.
func (vfs *RegistryVFSAdapter) buildContextOpenFile(path string) (io.ReadCloser, error) {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[BUILD-CTX] OpenFile called by interpreter",
		logger_domain.String("requested_path", path))

	file, err := vfs.OpenFile(vfs.ctx, path)
	if err == nil {
		l.Trace("[BUILD-CTX] VFS served file successfully", logger_domain.String(logFieldPath, path))
		return file, nil
	}

	if !errors.Is(err, os.ErrNotExist) {
		l.Error("[BUILD-CTX] VFS returned non-NotExist error",
			logger_domain.String(logFieldPath, path),
			logger_domain.Error(err))
		return nil, fmt.Errorf("opening file %q from VFS: %w", path, err)
	}

	l.Trace("[BUILD-CTX] VFS miss, trying real filesystem", logger_domain.String(logFieldPath, path))
	//nolint:gosec // trusted module system path
	realFile, realErr := os.Open(path)
	if realErr == nil {
		l.Trace("[BUILD-CTX] Real filesystem served file", logger_domain.String(logFieldPath, path))
	} else {
		l.Warn("[BUILD-CTX] Real filesystem also failed",
			logger_domain.String(logFieldPath, path),
			logger_domain.Error(realErr))
	}
	return realFile, realErr
}

// buildContextIsDir implements build.Context.IsDir for the interpreter.
// It checks VFS first, then falls back to the real filesystem.
//
// Takes path (string) which specifies the directory path to check.
//
// Returns bool which is true if the path exists and is a directory.
func (vfs *RegistryVFSAdapter) buildContextIsDir(path string) bool {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[BUILD-CTX] IsDir called", logger_domain.String(logFieldPath, path))

	info, err := vfs.Stat(path)
	if err == nil && info.IsDir() {
		l.Trace("[BUILD-CTX] VFS reports directory exists", logger_domain.String(logFieldPath, path))
		return true
	}

	info, err = os.Stat(path)
	if err == nil && info.IsDir() {
		l.Trace("[BUILD-CTX] Real filesystem reports directory exists", logger_domain.String(logFieldPath, path))
		return true
	}

	l.Trace("[BUILD-CTX] Directory not found", logger_domain.String(logFieldPath, path))
	return false
}

// buildContextHasSubdir implements build.Context.HasSubdir for the interpreter.
// It checks the VFS first, then falls back to the real filesystem.
//
// Takes root (string) which is the base directory path.
// Takes directory (string) which is the subdirectory to check for.
//
// Returns string which is the full path to the subdirectory if found.
// Returns bool which indicates whether the subdirectory exists.
func (vfs *RegistryVFSAdapter) buildContextHasSubdir(root, directory string) (string, bool) {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[BUILD-CTX] HasSubdir called",
		logger_domain.String("root", root),
		logger_domain.String(logFieldDir, directory))

	fullPath := filepath.Join(root, directory)

	info, err := vfs.Stat(fullPath)
	if err == nil && info.IsDir() {
		l.Trace("[BUILD-CTX] VFS found subdir", logger_domain.String(logFieldPath, fullPath))
		return fullPath, true
	}

	info, err = os.Stat(fullPath)
	if err == nil && info.IsDir() {
		l.Trace("[BUILD-CTX] Real filesystem found subdir", logger_domain.String(logFieldPath, fullPath))
		return fullPath, true
	}

	return "", false
}

// buildContextReadDir implements build.Context.ReadDir for the interpreter.
// It tries the VFS first, then falls back to the real file system.
//
// Takes directory (string) which specifies the directory path to read.
//
// Returns []os.FileInfo which contains the directory entries.
// Returns error when both the VFS and real file system fail to read the
// directory.
func (vfs *RegistryVFSAdapter) buildContextReadDir(directory string) ([]os.FileInfo, error) {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[BUILD-CTX] ReadDir called", logger_domain.String(logFieldDir, directory))

	entries, err := vfs.ReadDir(directory)
	if err == nil {
		l.Trace("[BUILD-CTX] VFS ReadDir succeeded",
			logger_domain.String(logFieldDir, directory),
			logger_domain.Int("entries", len(entries)))
		return convertDirEntriesToFileInfos(entries), nil
	}

	entries, err = os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("reading directory %q from real filesystem: %w", directory, err)
	}
	l.Trace("[BUILD-CTX] Real filesystem ReadDir succeeded",
		logger_domain.String(logFieldDir, directory),
		logger_domain.Int("entries", len(entries)))
	return convertDirEntriesToFileInfos(entries), nil
}

// openPackageAsFile returns a package's generated.go content as a virtual file.
//
// Takes name (string) which is the file path being requested.
// Takes artefactID (string) which identifies the package to fetch.
//
// Returns fs.File which provides access to the generated.go content.
// Returns error when the artefact cannot be fetched or has no Go variant.
func (vfs *RegistryVFSAdapter) openPackageAsFile(name, artefactID string) (fs.File, error) {
	_, l := logger_domain.From(vfs.ctx, log)
	if content, hasFresh := vfs.freshArtefacts[artefactID]; hasFresh {
		l.Trace("[VFS-FS] Serving from fresh cache",
			logger_domain.String(logFieldArtefactIDKey, artefactID),
			logger_domain.Int("size", len(content)))
		return &virtualFile{
			name:    generatedGoFilename,
			content: content,
			reader:  bytes.NewReader(content),
			closed:  false,
		}, nil
	}

	l.Trace("[VFS-FS] Fetching from registry",
		logger_domain.String(logFieldArtefactIDKey, artefactID))

	artefact, err := vfs.registryService.GetArtefact(vfs.ctx, artefactID)
	if err != nil {
		return nil, &fs.PathError{Op: logFieldOpen, Path: name, Err: err}
	}

	goVariant := vfs.findGoVariant(artefact)
	if goVariant == nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fmt.Errorf("missing %s variant", generatedGoVariantID),
		}
	}

	stream, err := vfs.registryService.GetVariantData(vfs.ctx, goVariant)
	if err != nil {
		return nil, &fs.PathError{Op: logFieldOpen, Path: name, Err: err}
	}

	data, err := io.ReadAll(stream)
	_ = stream.Close()
	if err != nil {
		return nil, &fs.PathError{Op: logFieldOpen, Path: name, Err: err}
	}

	return &virtualFile{
		name:    generatedGoFilename,
		content: data,
		reader:  bytes.NewReader(data),
		closed:  false,
	}, nil
}

// isParentDirectory checks if the path is a parent directory of any packages
// in pathMap. Unlike isDirectory, this returns false for exact package paths.
//
// Takes packagePath (string) which specifies the path to check.
//
// Returns bool which is true if packagePath is a parent of any registered package.
func (vfs *RegistryVFSAdapter) isParentDirectory(packagePath string) bool {
	prefix := packagePath
	if prefix != "" && prefix != "." {
		prefix = prefix + pathSeparator
	} else {
		prefix = ""
	}

	for candidatePath := range vfs.pathMap {
		if strings.HasPrefix(candidatePath, prefix) {
			return true
		}
	}

	return false
}

// resolveUserPackagePath checks if a package path belongs to the user's
// project and returns the matching path relative to the project sandbox.
//
// Takes packagePath (string) which is the package import path to resolve.
// Takes originalName (string) which is the original file or folder name.
//
// Returns string which is the resolved path relative to the project sandbox,
// or empty if the path does not belong to the user's project.
func (vfs *RegistryVFSAdapter) resolveUserPackagePath(packagePath, originalName string) string {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[VFS-USER-PKG] Checking user package resolution",
		logger_domain.String(logFieldPackagePath, packagePath),
		logger_domain.String("original_name", originalName),
		logger_domain.String("module_path", vfs.modulePath),
		logger_domain.String("project_root", vfs.projectRoot))

	if vfs.modulePath == "" || vfs.projectSandbox == nil {
		l.Trace("[VFS-USER-PKG] Skipping - module path or project sandbox not configured")
		return ""
	}

	if !strings.HasPrefix(packagePath, vfs.modulePath) {
		l.Trace("[VFS-USER-PKG] Skipping - package path doesn't match module path",
			logger_domain.String(logFieldPackagePath, packagePath),
			logger_domain.String("module_path", vfs.modulePath))
		return ""
	}

	relPath := strings.TrimPrefix(packagePath, vfs.modulePath)
	relPath = strings.TrimPrefix(relPath, pathSeparator)

	l.Trace("[VFS-USER-PKG] Extracted relative path",
		logger_domain.String("rel_path", relPath))

	baseName := filepath.Base(originalName)
	if strings.HasSuffix(baseName, ".go") {
		filePath := filepath.Join(relPath, baseName)
		l.Trace("[VFS-USER-PKG] Checking file path",
			logger_domain.String(logFieldRealPath, filePath))
		if _, err := vfs.projectSandbox.Stat(filePath); err == nil {
			l.Trace("[VFS-USER-PKG] Found file", logger_domain.String(logFieldPath, filePath))
			return filePath
		}
	}

	l.Trace("[VFS-USER-PKG] Checking directory path",
		logger_domain.String("dir_path", relPath))
	if info, err := vfs.projectSandbox.Stat(relPath); err == nil && info.IsDir() {
		l.Trace("[VFS-USER-PKG] Found directory", logger_domain.String(logFieldPath, relPath))
		return relPath
	}

	l.Trace("[VFS-USER-PKG] Path not found on filesystem",
		logger_domain.String("checked_dir", relPath))
	return ""
}

// readDirPackage handles ReadDir for a package directory that exists in
// pathMap.
//
// Takes packagePath (string) which specifies the package path to read.
//
// Returns []fs.DirEntry which contains the directory entries for the package.
// Returns bool which indicates whether the path is a known package.
func (vfs *RegistryVFSAdapter) readDirPackage(packagePath string) ([]fs.DirEntry, bool) {
	artefactID, ok := vfs.pathMap[packagePath]
	if !ok {
		return nil, false
	}

	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[VFS-READDIR] Found package in VFS",
		logger_domain.String(logFieldPath, packagePath),
		logger_domain.String(logFieldArtefactIDKey, artefactID))

	var size int64 = defaultFileSizeEstimate
	if content, hasFresh := vfs.freshArtefacts[artefactID]; hasFresh {
		size = int64(len(content))
	}

	return []fs.DirEntry{
		&virtualDirEntry{
			name:  generatedGoFilename,
			isDir: false,
			info: &virtualFileInfo{
				name:  generatedGoFilename,
				size:  size,
				isDir: false,
			},
		},
	}, true
}

// listSubdirectories returns virtual directory entries for packages that start
// with the given path.
//
// Takes packagePath (string) which specifies the package path prefix
// to search for.
//
// Returns []fs.DirEntry which contains the subdirectory entries found.
func (vfs *RegistryVFSAdapter) listSubdirectories(packagePath string) []fs.DirEntry {
	entries := make([]fs.DirEntry, 0, len(vfs.pathMap))
	seenDirs := make(map[string]bool)

	prefix := packagePath
	if prefix != "" && prefix != "." {
		prefix = prefix + pathSeparator
	} else {
		prefix = ""
	}

	for candidatePath := range vfs.pathMap {
		if !strings.HasPrefix(candidatePath, prefix) {
			continue
		}

		nextDir := vfs.extractNextPathComponent(candidatePath, prefix)
		if nextDir == "" || seenDirs[nextDir] {
			continue
		}

		seenDirs[nextDir] = true
		entries = append(entries, &virtualDirEntry{
			name:  nextDir,
			isDir: true,
			info: &virtualFileInfo{
				name:  nextDir,
				size:  0,
				isDir: true,
			},
		})
	}

	return entries
}

// extractNextPathComponent gets the next folder name after a prefix.
//
// Takes path (string) which is the full path to extract from.
// Takes prefix (string) which is the leading part to remove before extracting.
//
// Returns string which is the first path part after the prefix, or an empty
// string if none remains.
func (*RegistryVFSAdapter) extractNextPathComponent(path, prefix string) string {
	remainder := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remainder, pathSeparator)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// readDirFallback handles ReadDir when no entries exist in the VFS.
// It falls back to user packages or the real file system.
//
// Takes packagePath (string) which specifies the package path to resolve.
// Takes name (string) which specifies the directory name to read.
//
// Returns []fs.DirEntry which contains the directory entries found.
// Returns error when the directory cannot be read.
func (vfs *RegistryVFSAdapter) readDirFallback(packagePath, name string) ([]fs.DirEntry, error) {
	_, l := logger_domain.From(vfs.ctx, log)
	l.Trace("[VFS-READDIR] No entries in VFS, trying user package or real filesystem",
		logger_domain.String(logFieldPath, packagePath))

	if relPath := vfs.resolveUserPackagePath(packagePath, name); relPath != "" {
		l.Trace("[VFS-READDIR] Resolved user package via project sandbox",
			logger_domain.String(logFieldRealPath, relPath))
		return vfs.projectSandbox.ReadDir(relPath)
	}

	return os.ReadDir(name)
}

// isDirectory checks if the given path is a directory in the VFS.
//
// A path is a directory if it exactly matches a package path in pathMap, or
// other package paths start with this path (it is a parent directory).
//
// Takes packagePath (string) which specifies the path to check.
//
// Returns bool which is true if the path is a directory.
func (vfs *RegistryVFSAdapter) isDirectory(packagePath string) bool {
	if _, ok := vfs.pathMap[packagePath]; ok {
		return true
	}

	prefix := packagePath
	if prefix != "" && prefix != "." {
		prefix = prefix + pathSeparator
	} else {
		prefix = ""
	}

	for candidatePath := range vfs.pathMap {
		if strings.HasPrefix(candidatePath, prefix) {
			return true
		}
	}

	return false
}

// normalisePathForFS prepares a filesystem path for VFS lookups.
//
// This works like normalisePath but handles directory paths in a special way.
// It processes both absolute paths (e.g. /path/.piko-gopath/src/pkg) and
// relative paths (e.g. .piko-gopath/src/pkg) that the interpreter may pass.
//
// Takes name (string) which is the path to prepare for filesystem lookup.
//
// Returns string which is the package path with virtual prefixes and source
// directory parts removed.
func (vfs *RegistryVFSAdapter) normalisePathForFS(name string) string {
	p := filepath.ToSlash(name)
	packagePath := vfs.stripVirtualPrefix(p)

	packagePath = strings.Trim(packagePath, pathSeparator)
	packagePath = strings.TrimPrefix(packagePath, srcPathPrefix)
	packagePath = strings.TrimPrefix(packagePath, srcDirName)
	packagePath = strings.Trim(packagePath, pathSeparator)

	if strings.HasSuffix(packagePath, ".go") {
		return filepath.Dir(packagePath)
	}

	return packagePath
}

// stripVirtualPrefix extracts the package path from a virtual filesystem path.
// It handles GOPATH, GOROOT, and relative .piko-gopath prefixes.
//
// Takes p (string) which is the virtual filesystem path to process.
//
// Returns string which is the extracted package path. Returns empty string for
// the root of virtual GOPATH/src, or the original path if no virtual prefix
// is found.
func (vfs *RegistryVFSAdapter) stripVirtualPrefix(p string) string {
	if result, found := strings.CutPrefix(p, vfs.goPath); found {
		return result
	}

	if result, found := strings.CutPrefix(p, vfs.goRoot); found {
		return result
	}

	if _, after, found := strings.Cut(p, ".piko-gopath/src/"); found {
		return after
	}

	if _, after, found := strings.Cut(p, ".piko-gopath/src"); found && after == "" {
		return ""
	}

	return p
}

// virtualFile provides an in-memory file that implements io.ReadCloser.
type virtualFile struct {
	// reader provides read access to the file content.
	reader *bytes.Reader

	// name is the file path used in error messages.
	name string

	// content holds the file data in memory.
	content []byte

	// closed indicates whether the file has been closed.
	closed bool
}

// Stat implements fs.File.Stat.
//
// Returns fs.FileInfo which contains the file's metadata.
// Returns error when the file has been closed.
func (f *virtualFile) Stat() (fs.FileInfo, error) {
	if f.closed {
		return nil, &fs.PathError{Op: "stat", Path: f.name, Err: fs.ErrClosed}
	}
	return &virtualFileInfo{
		name:  f.name,
		size:  int64(len(f.content)),
		isDir: false,
	}, nil
}

// Read implements fs.File.Read.
//
// Takes b ([]byte) which is the buffer to read into.
//
// Returns int which is the number of bytes read.
// Returns error when the file has been closed or on read failure.
func (f *virtualFile) Read(b []byte) (int, error) {
	if f.closed {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrClosed}
	}
	return f.reader.Read(b)
}

// Close implements fs.File.Close.
//
// Returns error which is always nil.
func (f *virtualFile) Close() error {
	f.closed = true
	return nil
}

// virtualDir implements fs.ReadDirFile for virtual directories.
type virtualDir struct {
	// name is the directory path used in error messages.
	name string

	// path is the directory path in the virtual file system.
	path string

	// vfs provides access to the filesystem for reading directory entries.
	vfs *RegistryVFSAdapter

	// entries holds cached directory entries; nil until the first ReadDir call.
	entries []fs.DirEntry

	// offset tracks the current read position in entries for ReadDir calls.
	offset int

	// closed indicates whether the directory handle has been closed.
	closed bool
}

// Stat implements fs.File.Stat.
//
// Returns fs.FileInfo which describes this virtual directory.
// Returns error when the directory has been closed.
func (d *virtualDir) Stat() (fs.FileInfo, error) {
	if d.closed {
		return nil, &fs.PathError{Op: "stat", Path: d.name, Err: fs.ErrClosed}
	}
	return &virtualFileInfo{
		name:  d.name,
		size:  0,
		isDir: true,
	}, nil
}

// Read implements fs.File.Read for a directory entry.
//
// Takes []byte which is the buffer to read into.
//
// Returns int which is always zero as directories cannot be read.
// Returns error when called, as directories do not support reading.
func (d *virtualDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{Op: "read", Path: d.name, Err: errors.New("is a directory")}
}

// Close implements fs.File.Close.
//
// Returns error which is always nil.
func (d *virtualDir) Close() error {
	d.closed = true
	return nil
}

// ReadDir implements fs.ReadDirFile.ReadDir.
//
// Takes n (int) which specifies the maximum number of entries to return. If n
// is less than or equal to zero, all remaining entries are returned.
//
// Returns []fs.DirEntry which contains the directory entries.
// Returns error when the directory is closed, reading fails, or there are no
// more entries (io.EOF).
func (d *virtualDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.closed {
		return nil, &fs.PathError{Op: "readdir", Path: d.name, Err: fs.ErrClosed}
	}

	if d.entries == nil {
		entries, err := d.vfs.ReadDir(d.path)
		if err != nil {
			return nil, fmt.Errorf("reading virtual directory %q: %w", d.path, err)
		}
		d.entries = entries
	}

	if n <= 0 {
		result := d.entries[d.offset:]
		d.offset = len(d.entries)
		return result, nil
	}

	if d.offset >= len(d.entries) {
		return nil, io.EOF
	}

	end := min(d.offset+n, len(d.entries))

	result := d.entries[d.offset:end]
	d.offset = end

	if d.offset >= len(d.entries) {
		return result, io.EOF
	}

	return result, nil
}

// virtualFileInfo implements fs.FileInfo for virtual files and directories.
type virtualFileInfo struct {
	// name is the base name of the file.
	name string

	// size is the file size in bytes.
	size int64

	// isDir is true if this entry represents a directory.
	isDir bool
}

// Name returns the base name of the virtual file. Implements fs.FileInfo.
//
// Returns string which is the file's base name.
func (fi *virtualFileInfo) Name() string { return fi.name }

// Size returns the length of the virtual file content in bytes.
//
// Implements fs.FileInfo.Size.
//
// Returns int64 which is the size of the content.
func (fi *virtualFileInfo) Size() int64 { return fi.size }

// Mode returns the file mode bits for this virtual file or directory.
// Implements fs.FileInfo.
//
// Returns fs.FileMode which describes the file type and permission bits.
func (fi *virtualFileInfo) Mode() fs.FileMode {
	if fi.isDir {
		return fs.ModeDir | virtualDirPermissions
	}
	return virtualFilePermissions
}

// ModTime implements fs.FileInfo.ModTime.
//
// Returns time.Time which is the current time.
func (*virtualFileInfo) ModTime() time.Time { return time.Now() }

// IsDir implements fs.FileInfo.IsDir.
//
// Returns bool which is true if this file info represents a directory.
func (fi *virtualFileInfo) IsDir() bool { return fi.isDir }

// Sys implements fs.FileInfo.Sys.
//
// Returns any which is always nil for virtual files.
func (*virtualFileInfo) Sys() any { return nil }

// virtualDirEntry implements fs.DirEntry for directory listings.
type virtualDirEntry struct {
	// info holds the file metadata returned by the Info method.
	info fs.FileInfo

	// name is the file or directory name.
	name string

	// isDir is true if this entry represents a directory.
	isDir bool
}

// Name returns the directory entry's base name. Implements fs.DirEntry.
//
// Returns string which is the name of the file or directory.
func (e *virtualDirEntry) Name() string { return e.name }

// IsDir implements fs.DirEntry.IsDir.
//
// Returns bool which is true if the entry represents a directory.
func (e *virtualDirEntry) IsDir() bool { return e.isDir }

// Type implements fs.DirEntry.Type.
//
// Returns fs.FileMode which is fs.ModeDir for directories or 0 for files.
func (e *virtualDirEntry) Type() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}
	return 0
}

// Info implements fs.DirEntry.Info.
//
// Returns fs.FileInfo which provides the file metadata.
// Returns error which is always nil for this virtual entry.
func (e *virtualDirEntry) Info() (fs.FileInfo, error) { return e.info, nil }

// convertDirEntriesToFileInfos converts directory entries to file info values.
//
// Takes entries ([]fs.DirEntry) which contains the directory entries to
// convert.
//
// Returns []os.FileInfo which contains the file info for each entry that could
// be read. Entries that fail to read are skipped.
func convertDirEntriesToFileInfos(entries []fs.DirEntry) []os.FileInfo {
	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err == nil {
			infos = append(infos, info)
		}
	}
	return infos
}
