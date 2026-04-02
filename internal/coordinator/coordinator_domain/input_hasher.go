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

package coordinator_domain

// This file provides input hashing for cache key generation.

import (
	"context"
	"encoding/hex"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/cespare/xxhash/v2"
	"golang.org/x/sync/errgroup"
	"piko.sh/piko/internal/annotator/annotator_domain"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/resolver/resolver_domain"
	"piko.sh/piko/internal/sfcparser"
	"piko.sh/piko/wdk/safedisk"
)

// hashingFileReadWorkers is the number of workers that read and hash files at
// the same time.
const hashingFileReadWorkers = 16

// fileHashResult holds the path, hash, and content of a single hashed file.
type fileHashResult struct {
	// path is the file path used as a key for storing hash and content results.
	path string

	// hash is the computed hash value for the file.
	hash string

	// content holds the raw file bytes for content-based analysis.
	content []byte
}

// hashAndReadFile reads a file and computes its xxhash, using a cached hash
// when available and valid but still reading the file content in all cases.
//
// Takes path (string) which specifies the file path to read and hash.
// Takes modTime (time.Time) which is the file modification time used for
// cache lookups.
//
// Returns hash (string) which is the hex-encoded xxhash digest.
// Returns content ([]byte) which contains the raw file bytes.
// Returns err (error) when the file cannot be read.
func (s *coordinatorService) hashAndReadFile(
	ctx context.Context,
	path string,
	modTime time.Time,
) (hash string, content []byte, err error) {
	if s.fileHashCache != nil {
		if cachedHash, found := s.fileHashCache.Get(ctx, path, modTime); found {
			content, err = s.fsReader.ReadFile(ctx, path)
			if err != nil {
				return "", nil, fmt.Errorf("failed to read cached file %s: %w", path, err)
			}

			verifier := xxhash.New()
			_, _ = verifier.Write(content)
			actualHash := hex.EncodeToString(verifier.Sum(nil))
			if actualHash != cachedHash {
				s.fileHashCache.Set(ctx, path, modTime, actualHash)
				return actualHash, content, nil
			}

			return cachedHash, content, nil
		}
	}

	content, err = s.fsReader.ReadFile(ctx, path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	hasher := xxhash.New()
	_, _ = hasher.Write(content)
	hash = hex.EncodeToString(hasher.Sum(nil))

	if s.fileHashCache != nil {
		s.fileHashCache.Set(ctx, path, modTime, hash)
	}

	return hash, content, nil
}

// processFileForHash gets file information and computes its hash.
// This is used by computeFileHashesWithCache for parallel file processing.
//
// Takes path (string) which specifies the file path to process.
//
// Returns fileHashResult which contains the path, hash, and file content.
// Returns error when the file cannot be read or its information cannot be
// retrieved.
func (s *coordinatorService) processFileForHash(ctx context.Context, path string) (fileHashResult, error) {
	info, err := s.statFile(ctx, path)
	if err != nil {
		return fileHashResult{}, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	hash, content, err := s.hashAndReadFile(ctx, path, info.ModTime())
	if err != nil {
		return fileHashResult{}, fmt.Errorf("hashing and reading file %s: %w", path, err)
	}

	return fileHashResult{path: path, hash: hash, content: content}, nil
}

// statFile returns file info for a path, using the sandbox when available
// or falling back to creating a temporary sandbox for the file's directory.
//
// Takes path (string) which specifies the file path to stat.
//
// Returns fs.FileInfo which contains the file metadata.
// Returns error when the sandbox cannot be created or the file cannot be
// accessed.
func (s *coordinatorService) statFile(ctx context.Context, path string) (fs.FileInfo, error) {
	baseDir := s.resolver.GetBaseDir()

	if s.baseDirSandbox != nil && baseDir != "" && strings.HasPrefix(path, baseDir) {
		relPath, err := filepath.Rel(baseDir, path)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			return s.baseDirSandbox.Stat(relPath)
		}
	}

	directory := filepath.Dir(path)
	var sandbox safedisk.Sandbox
	var err error
	if s.sandboxFactory != nil {
		sandbox, err = s.sandboxFactory.Create("stat file", directory, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(directory, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox for %s: %w", directory, err)
	}
	defer func() {
		if closeErr := sandbox.Close(); closeErr != nil {
			_, wl := logger_domain.From(ctx, log)
			wl.Warn("Failed to close sandbox", logger_domain.Error(closeErr), logger_domain.String("dir", directory))
		}
	}()

	return sandbox.Stat(filepath.Base(path))
}

// calculateInputHash creates a stable hash from all relevant inputs.
// It uses the file hash cache (if available) to improve performance via
// stat-then-read.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the source
// files to include in the hash.
// Takes buildOpts (*buildOptions) which may contain a resolver override.
//
// Returns string which is the hex-encoded hash of all inputs.
// Returns map[string][]byte which contains the source file contents by path.
// Returns error when file discovery or hashing fails.
//
// Action files in actions/ are included as part of Go file hashing, so action
// changes are automatically detected.
func (s *coordinatorService) calculateInputHash(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	buildOpts *buildOptions,
) (string, map[string][]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.calculateInputHash")
	defer span.End()
	startTime := time.Now()

	resolver := s.getEffectiveResolver(buildOpts)
	sortedPaths, err := s.discoverAndSortAllSourcePaths(ctx, entryPoints, resolver)
	if err != nil {
		return "", nil, err
	}

	fileHashes, allSourceContents, err := s.computeFileHashesWithCache(ctx, sortedPaths)
	if err != nil {
		return "", nil, err
	}

	hasher := xxhash.New()
	for _, path := range sortedPaths {
		_, _ = hasher.WriteString(path)
		_, _ = hasher.WriteString(fileHashes[path])
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	duration := time.Since(startTime)
	inputHashDuration.Record(ctx, float64(duration.Milliseconds()))
	l.Trace("Input hash calculated", logger_domain.String("hash", hash), logger_domain.Int("fileCount", len(sortedPaths)))

	return hash, allSourceContents, nil
}

// getEffectiveResolver returns the resolver to use for this build.
// If buildOpts contains a resolver override, it is used; otherwise the
// coordinator's default resolver is returned.
//
// Takes buildOpts (*buildOptions) which may contain a resolver override.
//
// Returns resolver_domain.ResolverPort which is the resolver to use.
func (s *coordinatorService) getEffectiveResolver(buildOpts *buildOptions) resolver_domain.ResolverPort {
	if buildOpts != nil && buildOpts.Resolver != nil {
		return buildOpts.Resolver
	}
	return s.resolver
}

// discoverAndSortAllSourcePaths finds all .pk and .go files relevant to
// the build.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the starting
// points for dependency discovery.
// Takes resolver (ResolverPort) which provides path resolution.
//
// Returns []string which contains deduplicated, sorted paths to all source
// files.
// Returns error when component or Go file discovery fails.
func (s *coordinatorService) discoverAndSortAllSourcePaths(
	ctx context.Context,
	entryPoints []annotator_dto.EntryPoint,
	resolver resolver_domain.ResolverPort,
) ([]string, error) {
	discoverer := &hasherDependencyDiscoverer{
		resolver: resolver,
		fsReader: s.fsReader,
	}

	allComponentPaths, err := discoverer.Discover(ctx, getEntrypointPaths(entryPoints))
	if err != nil {
		return nil, fmt.Errorf("failed to discover component paths: %w", err)
	}

	goFilePaths, err := s.collectOriginalGoFilePaths(ctx, resolver)
	if err != nil {
		return nil, fmt.Errorf("failed to discover go file paths: %w", err)
	}

	allSourceFiles := make(map[string]struct{}, len(allComponentPaths)+len(goFilePaths))
	for _, path := range allComponentPaths {
		allSourceFiles[path] = struct{}{}
	}
	for _, path := range goFilePaths {
		allSourceFiles[path] = struct{}{}
	}

	sortedPaths := make([]string, 0, len(allSourceFiles))
	for path := range allSourceFiles {
		sortedPaths = append(sortedPaths, path)
	}
	slices.Sort(sortedPaths)
	return sortedPaths, nil
}

// computeFileHashesWithCache computes xxhash digests for all
// paths using a stat-then-read method. It returns both the hashes
// for cache key calculation and the file contents for later
// processing.
//
// Takes paths ([]string) which lists the file paths to hash.
//
// Returns map[string]string which maps file paths to their
// xxhash digests.
// Returns map[string][]byte which maps file paths to their contents.
// Returns error when reading or hashing any file fails.
//
// For each file:
//   - Calls os.Stat to get the modification time (fast, metadata only).
//   - Checks fileHashCache with the path and modification time.
//   - If cache hit: skips file read and uses the cached hash.
//   - If cache miss: reads the file, computes the hash, and updates the cache.
//
// In incremental builds where only one or two files change out of many, this
// reduces I/O from reading all files to doing stat calls plus reading only
// the changed files.
func (s *coordinatorService) computeFileHashesWithCache(ctx context.Context, paths []string) (map[string]string, map[string][]byte, error) {
	g, gCtx := errgroup.WithContext(ctx)
	jobs := make(chan string, len(paths))
	results := make(chan fileHashResult, len(paths))

	s.startHashWorkers(gCtx, g, jobs, results)
	s.enqueueHashJobs(gCtx, jobs, paths)

	if err := g.Wait(); err != nil {
		return nil, nil, fmt.Errorf("computing file hashes: %w", err)
	}
	close(results)

	return collectHashResults(results, len(paths))
}

// startHashWorkers starts workers to process file hash jobs.
//
// Takes g (*errgroup.Group) which manages the workers and
// collects errors.
// Takes jobs (<-chan string) which provides file paths for
// workers to process.
// Takes results (chan<- fileHashResult) which receives the
// computed hash results from each worker.
func (s *coordinatorService) startHashWorkers(ctx context.Context, g *errgroup.Group, jobs <-chan string, results chan<- fileHashResult) {
	for range hashingFileReadWorkers {
		g.Go(func() error {
			return s.hashWorkerLoop(ctx, jobs, results)
		})
	}
}

// hashWorkerLoop processes files from the jobs channel and sends results.
//
// Takes jobs (<-chan string) which provides file paths to process.
// Takes results (chan<- fileHashResult) which receives the computed hashes.
//
// Returns error when file processing fails or the context is cancelled.
func (s *coordinatorService) hashWorkerLoop(ctx context.Context, jobs <-chan string, results chan<- fileHashResult) error {
	for path := range jobs {
		result, err := s.processFileForHash(ctx, path)
		if err != nil {
			return fmt.Errorf("processing file for hash %s: %w", path, err)
		}
		select {
		case results <- result:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

// enqueueHashJobs sends file paths to the jobs channel for workers to process.
//
// Takes jobs (chan<- string) which receives paths for workers to process.
// Takes paths ([]string) which contains the file paths to send.
//
// Safe for concurrent use. Spawns a goroutine that sends paths until all
// are sent or the context is cancelled, then closes the jobs channel.
func (*coordinatorService) enqueueHashJobs(ctx context.Context, jobs chan<- string, paths []string) {
	go func() {
		defer close(jobs)
		for _, path := range paths {
			select {
			case jobs <- path:
			case <-ctx.Done():
				return
			}
		}
	}()
}

// collectOriginalGoFilePaths walks the project directory to find all relevant
// .go files.
//
// Takes resolver (ResolverPort) which provides the base directory path.
//
// Returns []string which contains absolute paths to discovered Go files.
// Returns error when the directory walk fails.
func (s *coordinatorService) collectOriginalGoFilePaths(ctx context.Context, resolver resolver_domain.ResolverPort) ([]string, error) {
	baseDir := resolver.GetBaseDir()
	if baseDir == "" {
		return nil, nil
	}

	sandbox, needsClose, err := s.getOrCreateSandbox(baseDir)
	if err != nil {
		return nil, fmt.Errorf("obtaining sandbox for Go file collection: %w", err)
	}
	if needsClose {
		defer func() {
			if closeErr := sandbox.Close(); closeErr != nil {
				_, wl := logger_domain.From(ctx, log)
				wl.Warn("Failed to close sandbox", logger_domain.Error(closeErr), logger_domain.String("dir", baseDir))
			}
		}()
	}

	fi, err := sandbox.Stat(".")
	if err != nil || !fi.IsDir() {
		return nil, nil
	}

	var goFilePaths []string
	err = sandbox.WalkDir(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return s.handleDirEntry(d.Name())
		}
		if isRelevantGoFile(d.Name()) {
			goFilePaths = append(goFilePaths, filepath.Join(baseDir, path))
		}
		return nil
	})
	return goFilePaths, err
}

// collectAllPKFilePaths walks the project tree and returns all .pk file paths.
//
// Unlike the BFS-based hasherDependencyDiscoverer.Discover(), this finds all
// .pk files regardless of which entry points are being built. This is used by
// the introspection hash so that the hash remains stable whether a full build
// or a targeted rebuild (with a subset of entry points) is performed.
//
// Takes resolver (ResolverPort) which provides the base directory.
//
// Returns []string which contains absolute paths to all .pk files.
// Returns error when the directory walk fails.
func (s *coordinatorService) collectAllPKFilePaths(ctx context.Context, resolver resolver_domain.ResolverPort) ([]string, error) {
	baseDir := resolver.GetBaseDir()
	if baseDir == "" {
		return nil, nil
	}

	sandbox, needsClose, err := s.getOrCreateSandbox(baseDir)
	if err != nil {
		return nil, fmt.Errorf("obtaining sandbox for PK file collection: %w", err)
	}
	if needsClose {
		defer func() {
			if closeErr := sandbox.Close(); closeErr != nil {
				_, wl := logger_domain.From(ctx, log)
				wl.Warn("Failed to close sandbox", logger_domain.Error(closeErr), logger_domain.String("dir", baseDir))
			}
		}()
	}

	fi, err := sandbox.Stat(".")
	if err != nil || !fi.IsDir() {
		return nil, nil
	}

	var pkFilePaths []string
	err = sandbox.WalkDir(".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return s.handleDirEntry(d.Name())
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".pk") {
			pkFilePaths = append(pkFilePaths, filepath.Join(baseDir, path))
		}
		return nil
	})
	return pkFilePaths, err
}

// getOrCreateSandbox returns the existing baseDirSandbox if it matches the
// requested baseDir, or creates a temporary one.
//
// Takes baseDir (string) which specifies the directory path for the sandbox.
//
// Returns safedisk.Sandbox which is the existing or newly created sandbox.
// Returns bool which indicates a new sandbox was created and must be closed
// by the caller.
// Returns error when the sandbox cannot be created.
func (s *coordinatorService) getOrCreateSandbox(baseDir string) (safedisk.Sandbox, bool, error) {
	if s.baseDirSandbox != nil && s.baseDirSandboxPath == baseDir {
		return s.baseDirSandbox, false, nil
	}
	var sandbox safedisk.Sandbox
	var err error
	if s.sandboxFactory != nil {
		sandbox, err = s.sandboxFactory.Create("base dir fallback", baseDir, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(baseDir, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to create sandbox for baseDir: %w", err)
	}
	return sandbox, true, nil
}

// handleDirEntry checks if a directory should be skipped during traversal.
//
// Takes name (string) which is the directory name to check.
//
// Returns error when the directory should be skipped (returns fs.SkipDir for
// hidden directories, vendor, dist, and node_modules directories).
func (*coordinatorService) handleDirEntry(name string) error {
	if len(name) > 1 && name[0] == '.' {
		return fs.SkipDir
	}
	switch name {
	case "vendor", "dist", "node_modules":
		return fs.SkipDir
	default:
		return nil
	}
}

// calculateIntrospectionHash creates a deterministic hash from script blocks
// and .go files. This is the Tier 1 hash that determines if expensive type
// introspection results can be reused.
//
// Takes buildOpts (*buildOptions) which provides build configuration for
// resolving source paths.
//
// Returns string which is the computed hash of introspection-affecting content.
// Returns map[string]string which maps file paths to their individual hashes.
// Returns error when file discovery or hashing fails.
//
// The key insight is that type introspection (packages.Load()) only depends on
// the <script> blocks from .pk files and all .go files in the project.
// Changes to <template>, <style>, or <i18n> blocks do not affect type
// introspection. By hashing only the introspection-affecting content, we can
// detect when template-only changes occur and skip the expensive introspection
// phase.
//
// Action files in actions/ are included as part of Go file hashing, so action
// changes are automatically detected.
func (s *coordinatorService) calculateIntrospectionHash(
	ctx context.Context,
	_ []annotator_dto.EntryPoint,
	buildOpts *buildOptions,
) (string, map[string]string, error) {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "CoordinatorService.calculateIntrospectionHash")
	defer span.End()
	startTime := time.Now()

	resolver := s.getEffectiveResolver(buildOpts)

	pkPaths, err := s.collectAllPKFilePaths(ctx, resolver)
	if err != nil {
		return "", nil, fmt.Errorf("collecting pk file paths for introspection hash: %w", err)
	}
	goPaths, err := s.collectOriginalGoFilePaths(ctx, resolver)
	if err != nil {
		return "", nil, fmt.Errorf("collecting go file paths for introspection hash: %w", err)
	}

	allSourceFiles := make(map[string]struct{}, len(pkPaths)+len(goPaths))
	for _, path := range pkPaths {
		allSourceFiles[path] = struct{}{}
	}
	for _, path := range goPaths {
		allSourceFiles[path] = struct{}{}
	}
	sortedPaths := make([]string, 0, len(allSourceFiles))
	for path := range allSourceFiles {
		sortedPaths = append(sortedPaths, path)
	}
	slices.Sort(sortedPaths)

	_, allSourceContents, err := s.computeFileHashesWithCache(ctx, sortedPaths)
	if err != nil {
		return "", nil, err
	}

	hasher := xxhash.New()

	scriptHashes, err := hashIntrospectionContent(hasher, sortedPaths, allSourceContents)
	if err != nil {
		return "", nil, err
	}

	hash := hex.EncodeToString(hasher.Sum(nil))
	duration := time.Since(startTime)
	introspectionHashDuration.Record(ctx, float64(duration.Milliseconds()))
	l.Trace("Introspection hash calculated",
		logger_domain.String("hash", hash),
		logger_domain.Int("pk_file_count", len(scriptHashes)),
		logger_domain.Int("total_file_count", len(sortedPaths)))

	return hash, scriptHashes, nil
}

// hasherDependencyDiscoverer finds all .pk file paths for an input hasher.
// It focuses on speed and path collection rather than detailed error messages.
type hasherDependencyDiscoverer struct {
	// resolver handles path resolution for module and import lookups.
	resolver resolver_domain.ResolverPort

	// fsReader reads files from the filesystem to extract import statements.
	fsReader annotator_domain.FSReaderPort
}

// Discover finds all .pk files reachable from the given entry points.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for file reading and import resolution.
// Takes entryPointPaths ([]string) which specifies the starting paths to
// search from.
//
// Returns []string which contains all discovered file paths.
// Returns error when entry points cannot be processed or imports fail.
func (d *hasherDependencyDiscoverer) Discover(ctx context.Context, entryPointPaths []string) ([]string, error) {
	queue := make([]string, 0, len(entryPointPaths))
	visited := make(map[string]bool)

	if err := d.enqueueEntryPoints(ctx, entryPointPaths, &queue, visited); err != nil {
		return nil, fmt.Errorf("enqueuing entry points for dependency discovery: %w", err)
	}

	if err := d.processImportQueue(ctx, &queue, visited); err != nil {
		return nil, fmt.Errorf("processing import queue for dependency discovery: %w", err)
	}

	return queue, nil
}

// enqueueEntryPoints resolves entry point paths and adds them to the queue.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for path resolution.
// Takes paths ([]string) which contains the entry point paths to process.
// Takes queue (*[]string) which receives the resolved paths for processing.
// Takes visited (map[string]bool) which tracks paths already processed.
//
// Returns error when an entry point cannot be resolved.
func (d *hasherDependencyDiscoverer) enqueueEntryPoints(ctx context.Context, paths []string, queue *[]string, visited map[string]bool) error {
	moduleName := d.resolver.GetModuleName()
	for _, path := range paths {
		resolvedPath, err := d.resolveEntryPoint(ctx, path, moduleName)
		if err != nil {
			return fmt.Errorf("resolving entry point %q: %w", path, err)
		}
		if !visited[resolvedPath] {
			*queue = append(*queue, resolvedPath)
			visited[resolvedPath] = true
		}
	}
	return nil
}

// processImportQueue discovers imports from each file in the queue.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for file reading and import resolution.
// Takes queue (*[]string) which holds the file paths to process.
// Takes visited (map[string]bool) which tracks already processed files.
//
// Returns error when import discovery fails for any file in the queue.
func (d *hasherDependencyDiscoverer) processImportQueue(ctx context.Context, queue *[]string, visited map[string]bool) error {
	for i := 0; i < len(*queue); i++ {
		currentPath := (*queue)[i]
		if err := d.discoverImportsFromFile(ctx, currentPath, queue, visited); err != nil {
			return fmt.Errorf("discovering imports from %s: %w", currentPath, err)
		}
	}
	return nil
}

// discoverImportsFromFile extracts and resolves imports from a single file.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for file reading and import resolution.
// Takes currentPath (string) which is the file path to extract imports from.
// Takes queue (*[]string) which collects resolved paths for processing.
// Takes visited (map[string]bool) which tracks paths already found.
//
// Returns error when import extraction or path resolution fails.
func (d *hasherDependencyDiscoverer) discoverImportsFromFile(ctx context.Context, currentPath string, queue *[]string, visited map[string]bool) error {
	imports, err := d.extractPKImports(ctx, currentPath)
	if err != nil {
		return fmt.Errorf("extracting imports from %s: %w", currentPath, err)
	}
	for _, importPath := range imports {
		resolved, err := d.resolver.ResolvePKPath(ctx, importPath, currentPath)
		if err != nil {
			return fmt.Errorf("failed to resolve import '%s' in '%s': %w", importPath, currentPath, err)
		}
		if !visited[resolved] {
			*queue = append(*queue, resolved)
			visited[resolved] = true
		}
	}
	return nil
}

// resolveEntryPoint resolves a single entry point path to an absolute path.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for path resolution.
// Takes path (string) which is the entry point path to resolve.
// Takes moduleName (string) which is the module name for path prefixing.
//
// Returns string which is the resolved absolute path.
// Returns error when the path cannot be resolved.
func (d *hasherDependencyDiscoverer) resolveEntryPoint(ctx context.Context, path, moduleName string) (string, error) {
	pathToResolve := path
	if !strings.HasPrefix(path, moduleName+"/") {
		pathToResolve = filepath.ToSlash(filepath.Join(moduleName, path))
	}

	if filepath.IsAbs(pathToResolve) {
		return pathToResolve, nil
	}

	resolved, err := d.resolver.ResolvePKPath(ctx, pathToResolve, "")
	if err != nil {
		return "", fmt.Errorf("cannot resolve entry point '%s': %w", path, err)
	}
	return resolved, nil
}

// extractPKImports reads a .pk file and returns its .pk import paths.
//
// Takes ctx (context.Context) which carries cancellation and deadline signals
// for file reading.
// Takes currentPath (string) which specifies the path to the .pk file to read.
//
// Returns []string which contains the import paths ending in .pk found in the
// file's Go script section.
// Returns error when reading the file, parsing the SFC, or parsing Go imports
// fails.
func (d *hasherDependencyDiscoverer) extractPKImports(ctx context.Context, currentPath string) ([]string, error) {
	fileData, err := d.fsReader.ReadFile(ctx, currentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file during hash discovery '%s': %w", currentPath, err)
	}

	sfcResult, err := sfcparser.Parse(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SFC for hash discovery '%s': %w", currentPath, err)
	}

	goScript, ok := sfcResult.GoScript()
	if !ok || strings.TrimSpace(goScript.Content) == "" {
		return nil, nil
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", goScript.Content, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to parse imports for hash discovery '%s': %w", currentPath, err)
	}

	var imports []string
	for _, spec := range file.Imports {
		pathVal := strings.Trim(spec.Path.Value, `"`)
		if strings.HasSuffix(strings.ToLower(pathVal), ".pk") {
			imports = append(imports, pathVal)
		}
	}
	return imports, nil
}

// collectHashResults gathers results from a channel into maps.
//
// Takes results (<-chan fileHashResult) which yields file hash results to
// collect.
// Takes capacity (int) which sets the starting size for the result maps.
//
// Returns map[string]string which maps file paths to their hash strings.
// Returns map[string][]byte which maps file paths to their file contents.
// Returns error which is always nil in this version.
func collectHashResults(results <-chan fileHashResult, capacity int) (map[string]string, map[string][]byte, error) {
	fileHashes := make(map[string]string, capacity)
	fileContents := make(map[string][]byte, capacity)
	for result := range results {
		fileHashes[result.path] = result.hash
		fileContents[result.path] = result.content
	}
	return fileHashes, fileContents, nil
}

// isRelevantGoFile checks if a filename is a Go source file that is not a test.
//
// Takes name (string) which is the filename to check.
//
// Returns bool which is true if the file ends with .go but not _test.go.
func isRelevantGoFile(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// getEntrypointPaths extracts the path strings from a list of entry points.
//
// Takes entryPoints ([]annotator_dto.EntryPoint) which contains the entry
// points to get paths from.
//
// Returns []string which contains the path from each entry point.
func getEntrypointPaths(entryPoints []annotator_dto.EntryPoint) []string {
	paths := make([]string, len(entryPoints))
	for i, ep := range entryPoints {
		paths[i] = ep.Path
	}
	return paths
}

// hashIntrospectionContent hashes source files that affect type introspection.
//
// For .pk files, only the <script> block is hashed. For .go files, the entire
// content is hashed.
//
// Takes hasher (*xxhash.Digest) which accumulates the hash state.
// Takes sortedPaths ([]string) which lists the file paths in sorted order.
// Takes allSourceContents (map[string][]byte) which maps paths to file
// contents.
//
// Returns map[string]string which maps file paths to script hashes for cache
// validation.
// Returns error when script extraction fails for a .pk file.
func hashIntrospectionContent(
	hasher *xxhash.Digest,
	sortedPaths []string,
	allSourceContents map[string][]byte,
) (map[string]string, error) {
	scriptHashes := make(map[string]string)

	for _, path := range sortedPaths {
		if strings.HasSuffix(strings.ToLower(path), ".pk") {
			scriptContent, scriptHash, err := extractScriptBlockContent(path, allSourceContents[path])
			if err != nil {
				return nil, fmt.Errorf("failed to extract script from '%s': %w", path, err)
			}

			if scriptHash != "" {
				scriptHashes[path] = scriptHash
			}

			_, _ = hasher.WriteString(path)
			_, _ = hasher.WriteString(scriptContent)
		} else if strings.HasSuffix(strings.ToLower(path), ".go") {
			_, _ = hasher.WriteString(path)
			_, _ = hasher.Write(allSourceContents[path])
		}
	}

	return scriptHashes, nil
}

// extractScriptBlockContent parses a .pk file and extracts only the <script>
// block content, returning the script text, its xxhash, and any parsing
// error, or empty strings when no script block is present.
//
// Takes filePath (string) which identifies the file for hash
// calculation.
// Takes fileContent ([]byte) which contains the raw .pk file bytes to
// parse.
//
// Returns scriptContent (string) which is the extracted Go script block
// text.
// Returns scriptHash (string) which is the hex-encoded xxhash of the
// script content.
// Returns err (error) when the SFC cannot be parsed.
func extractScriptBlockContent(filePath string, fileContent []byte) (scriptContent string, scriptHash string, err error) {
	sfcResult, err := sfcparser.Parse(fileContent)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse SFC: %w", err)
	}

	goScript, hasGoScript := sfcResult.GoScript()
	if !hasGoScript {
		return "", "", nil
	}

	scriptContent = goScript.Content

	scriptHasher := xxhash.New()
	_, _ = scriptHasher.WriteString(filePath)
	_, _ = scriptHasher.WriteString(scriptContent)
	scriptHash = hex.EncodeToString(scriptHasher.Sum(nil))

	return scriptContent, scriptHash, nil
}
