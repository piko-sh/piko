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

package config_domain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

// FileResolver resolves placeholders like "file:path/to/your/file".
// This resolver contains no internal cache, as caching is handled centrally by
// the Loader.
//
// When a sandbox is provided, all paths are relative to the sandbox root and
// access is restricted to within the sandbox. When no sandbox is provided,
// the resolver operates with direct filesystem access (less secure).
type FileResolver struct {
	// sandboxFactory creates sandboxes when no sandbox is set. When non-nil
	// and sandbox is nil, this factory is used instead of safedisk.NewNoOpSandbox.
	sandboxFactory safedisk.Factory

	// sandbox provides restricted file system access; nil uses direct file I/O.
	sandbox safedisk.Sandbox
}

var _ BatchResolver = (*FileResolver)(nil)

// NewFileResolver creates a new file content resolver for loading
// configuration from files.
//
// When sandbox is nil, the resolver operates with direct filesystem access.
// For improved security, provide a sandbox to restrict file access.
//
// Takes sandbox (safedisk.Sandbox) which restricts file access to permitted
// paths, or nil for unrestricted access.
//
// Returns *FileResolver which resolves file paths to their contents.
func NewFileResolver(sandbox safedisk.Sandbox) *FileResolver {
	return &FileResolver{sandbox: sandbox}
}

// NewFileResolverWithFactory creates a new file content resolver with an
// optional sandbox factory. The factory is used to create sandboxes when no
// sandbox is directly injected, falling back to safedisk.NewNoOpSandbox when
// the factory is nil.
//
// Takes sandbox (safedisk.Sandbox) which restricts file access to permitted
// paths, or nil for unrestricted access.
// Takes factory (safedisk.Factory) which creates sandboxes when sandbox is
// nil.
//
// Returns *FileResolver which resolves file paths to their contents.
func NewFileResolverWithFactory(sandbox safedisk.Sandbox, factory safedisk.Factory) *FileResolver {
	return &FileResolver{sandbox: sandbox, sandboxFactory: factory}
}

// GetPrefix returns the prefix used to identify file-based paths.
//
// Returns string which is the literal prefix "file:".
func (*FileResolver) GetPrefix() string {
	return "file:"
}

// Resolve reads the content of a single file from the filesystem.
//
// When a sandbox is set, the path is treated as relative to the sandbox root.
// When no sandbox is set, the path is used directly (absolute or relative to
// the current working directory).
//
// Takes value (string) which specifies the file path to read.
//
// Returns string which contains the trimmed file content.
// Returns error when the file does not exist or cannot be read.
func (r *FileResolver) Resolve(ctx context.Context, value string) (string, error) {
	filePath := filepath.Clean(value)

	var data []byte
	var err error

	if r.sandbox != nil {
		data, err = r.sandbox.ReadFile(filePath)
	} else {
		data, err = r.readFileDirect(ctx, filePath)
	}

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("file not found: %q", filePath)
		}
		return "", fmt.Errorf("failed to read file %q: %w", filePath, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// ResolveBatch reads multiple files with bounded parallelism.
//
// It uses the single Resolve method, and the central Loader's cache handles
// deduplication of reads for the same file path.
//
// Takes values ([]string) which specifies the file paths to resolve.
//
// Returns map[string]string which maps input paths to their resolved content.
// Returns error when any file cannot be read.
func (r *FileResolver) ResolveBatch(ctx context.Context, values []string) (map[string]string, error) {
	numWorkers := min(runtime.GOMAXPROCS(0), len(values))
	if numWorkers == 0 {
		return make(map[string]string), nil
	}

	jobs := make(chan string, len(values))
	results := make(chan fileResolveResult, len(values))

	uniqueValues := r.startWorkersAndFeedJobs(ctx, numWorkers, jobs, results, values)
	resolved, err := r.collectResults(results, uniqueValues)
	if err != nil {
		return nil, fmt.Errorf("collecting batch file resolution results: %w", err)
	}

	return r.mapResultsToInput(resolved, values), nil
}

// readFileDirect reads a file directly from the filesystem without sandboxing.
// This is a fallback when no sandbox is configured.
//
// Takes filePath (string) which specifies the path to the file to read.
//
// Returns []byte which contains the file contents.
// Returns error when the sandbox cannot be created or the file cannot be read.
func (r *FileResolver) readFileDirect(ctx context.Context, filePath string) ([]byte, error) {
	parentDir := filepath.Dir(filePath)
	if parentDir == "" {
		parentDir = "."
	}

	var sandbox safedisk.Sandbox
	var err error
	if r.sandboxFactory != nil {
		sandbox, err = r.sandboxFactory.Create("config-file", parentDir, safedisk.ModeReadOnly)
	} else {
		sandbox, err = safedisk.NewNoOpSandbox(parentDir, safedisk.ModeReadOnly)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox for %q: %w", parentDir, err)
	}
	defer func() {
		if closeErr := sandbox.Close(); closeErr != nil {
			_, l := logger_domain.From(ctx, log)
			l.Warn("Failed to close sandbox", logger_domain.Error(closeErr), logger_domain.String("dir", parentDir))
		}
	}()

	baseName := filepath.Base(filePath)
	return sandbox.ReadFile(baseName)
}

// startWorkersAndFeedJobs starts workers and feeds unique values to them.
//
// Takes numWorkers (int) which specifies the number of workers to start.
// Takes jobs (chan string) which receives values to process.
// Takes results (chan fileResolveResult) which receives worker results.
// Takes values ([]string) which provides the values to deduplicate and feed.
//
// Returns map[string]struct{} which contains the unique values that were fed.
//
// Spawns numWorkers goroutines that process jobs until the channel closes.
// Spawns an additional goroutine that closes results when all workers finish.
func (r *FileResolver) startWorkersAndFeedJobs(
	ctx context.Context,
	numWorkers int,
	jobs chan string,
	results chan fileResolveResult,
	values []string,
) map[string]struct{} {
	var wg sync.WaitGroup

	for range numWorkers {
		wg.Go(func() {
			r.processJobs(ctx, jobs, results)
		})
	}

	uniqueValues := r.feedUniqueJobs(ctx, jobs, values)

	go func() {
		wg.Wait()
		close(results)
	}()

	return uniqueValues
}

// processJobs reads keys from the jobs channel, resolves each one, and
// sends the result until the channel closes or the context is cancelled.
//
// Takes jobs (chan string) which provides the keys to resolve.
// Takes results (chan fileResolveResult) which receives the resolved values.
func (r *FileResolver) processJobs(
	ctx context.Context,
	jobs chan string,
	results chan fileResolveResult,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case key, ok := <-jobs:
			if !ok {
				return
			}
			content, err := r.Resolve(ctx, key)
			select {
			case results <- fileResolveResult{err: err, key: key, content: content}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// feedUniqueJobs deduplicates values and sends them to the jobs channel,
// closing it when all values are sent or the context is cancelled.
//
// Takes jobs (chan string) which receives the deduplicated values.
// Takes values ([]string) which contains the raw values to deduplicate and send.
//
// Returns map[string]struct{} which contains the set of unique values sent.
func (*FileResolver) feedUniqueJobs(
	ctx context.Context,
	jobs chan string,
	values []string,
) map[string]struct{} {
	uniqueValues := make(map[string]struct{})
	for _, v := range values {
		if _, exists := uniqueValues[v]; !exists {
			uniqueValues[v] = struct{}{}
			select {
			case jobs <- v:
			case <-ctx.Done():
				close(jobs)
				return uniqueValues
			}
		}
	}
	close(jobs)
	return uniqueValues
}

// collectResults reads all results from the channel and collects any errors.
//
// Takes results (chan fileResolveResult) which provides the resolution results
// to collect.
// Takes uniqueValues (map[string]struct{...}) which sets the expected size for
// the resolved map.
//
// Returns map[string]string which maps keys to their resolved content.
// Returns error when any resolution fails, with all errors joined together.
func (*FileResolver) collectResults(results chan fileResolveResult, uniqueValues map[string]struct{}) (map[string]string, error) {
	var allErrors []error
	resolved := make(map[string]string, len(uniqueValues))

	for fileResult := range results {
		if fileResult.err != nil {
			allErrors = append(allErrors, fmt.Errorf("value %q: %w", fileResult.key, fileResult.err))
		} else {
			resolved[fileResult.key] = fileResult.content
		}
	}

	if len(allErrors) > 0 {
		return nil, errors.Join(allErrors...)
	}
	return resolved, nil
}

// mapResultsToInput maps resolved values back to the original input order.
//
// Takes resolved (map[string]string) which contains the resolved path mappings.
// Takes values ([]string) which specifies the original input order to preserve.
//
// Returns map[string]string which contains only the values from the original
// input, preserving their order.
func (*FileResolver) mapResultsToInput(resolved map[string]string, values []string) map[string]string {
	finalResults := make(map[string]string, len(values))
	for _, v := range values {
		finalResults[v] = resolved[v]
	}
	return finalResults
}

// fileResolveResult holds the outcome of resolving a single file.
type fileResolveResult struct {
	// err holds any error that occurred during file resolution.
	err error

	// key is the original value string used to look up and store the result.
	key string

	// content is the resolved value for the key.
	content string
}
