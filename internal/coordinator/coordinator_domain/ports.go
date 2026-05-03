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

import (
	"context"
	"time"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/resolver/resolver_domain"
)

// CoordinatorService is the primary driving port for the coordinator service.
// It defines capabilities for managing the Piko project's build lifecycle,
// acting as a long-lived, stateful service that orchestrates expensive build
// operations with caching and concurrency management.
type CoordinatorService interface {
	// Subscribe registers a listener for new successful build results.
	//
	// Takes name (string) which identifies the subscriber for observability.
	//
	// Returns <-chan BuildNotification which yields build notifications.
	// Returns UnsubscribeFunc which must be called to prevent resource leaks.
	Subscribe(name string) (<-chan BuildNotification, UnsubscribeFunc)

	// GetResult returns the last successful build result synchronously.
	//
	// On first run when no build has completed, blocks and triggers
	// the initial build. Subsequent calls return the latest result from memory
	// without triggering a new build. Simplifies downstream consumers by
	// guaranteeing a valid result without forcing them to handle the initial
	// cold start case.
	//
	// Actions are auto-discovered from the actions/ directory during annotation.
	//
	// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the build
	// entry points.
	// Takes opts (...BuildOption) which configures build behaviour.
	//
	// Returns *annotator_dto.ProjectAnnotationResult which contains the build
	// result.
	// Returns error when the build fails.
	GetResult(
		ctx context.Context,
		entryPoints []annotator_dto.EntryPoint,
		opts ...BuildOption,
	) (*annotator_dto.ProjectAnnotationResult, error)

	// GetOrBuildProject retrieves a cached build result or triggers a new build.
	//
	// The primary synchronous, blocking method for consumers that need a
	// build result to proceed with their work (e.g., a web server rendering a
	// page). Guarantees returning a valid build result for the current state
	// of the project's source files. First attempts to find a valid result
	// in the cache. If a cache miss occurs (due to changed files or an empty
	// cache), triggers a new build and blocks until that build completes.
	//
	// Actions are auto-discovered from the actions/ directory during annotation.
	//
	// Takes entryPoints ([]annotator_dto.EntryPoint) which specifies the files
	// to build.
	// Takes opts (...BuildOption) which configures build behaviour.
	//
	// Returns *annotator_dto.ProjectAnnotationResult which contains the build
	// output.
	// Returns error when the build fails or is cancelled.
	//
	// Protected against concurrent builds for the same inputs
	// (cache stampedes) using an internal singleflight mechanism.
	GetOrBuildProject(
		ctx context.Context,
		entryPoints []annotator_dto.EntryPoint,
		opts ...BuildOption,
	) (*annotator_dto.ProjectAnnotationResult, error)

	// RequestRebuild schedules a build to run after a short debounce period.
	//
	// Non-blocking: completes straight away. When called many times in quick
	// succession, the debounce timer resets, so only one build runs for a burst
	// of events. The build runs in the background.
	//
	// Actions are auto-discovered from the actions/ directory during annotation.
	//
	// Takes entryPoints ([]annotator_dto.EntryPoint) which lists the files to
	// build.
	// Takes opts (...BuildOption) which configures the build behaviour.
	RequestRebuild(
		ctx context.Context,
		entryPoints []annotator_dto.EntryPoint,
		opts ...BuildOption,
	)

	// GetLastSuccessfulBuild is a fast, non-blocking, read-only method that
	// returns the most recent successfully completed build result from memory.
	//
	// It never triggers a build. This is ideal for consumers that need access to
	// the project's state without incurring the cost of a potential build, such
	// as serving slightly stale content while a new build triggered by
	// RequestRebuild is running in the background.
	//
	// Returns *ProjectAnnotationResult which is the most recent successful build.
	// Returns bool which indicates whether a successful build was available.
	GetLastSuccessfulBuild() (*annotator_dto.ProjectAnnotationResult, bool)

	// Invalidate clears the build cache.
	//
	// Forces the next call to GetOrBuildProject to do a full build without
	// using the cache. Does not start a build on its own. Use to
	// force a refresh, typically in tests or admin tools. For most cases
	// where a trigger starts a rebuild, use RequestRebuild instead.
	//
	// Returns error when the cache cannot be cleared.
	Invalidate(ctx context.Context) error

	// Shutdown performs a graceful shutdown of the coordinator service, ensuring
	// that background goroutines are terminated cleanly. This should be called
	// once when the application is exiting.
	//
	// Takes ctx (context.Context) which carries logging context for shutdown
	// operations.
	Shutdown(ctx context.Context)
}

// BuildResultCachePort is the driven port for caching complete build results.
// This stores the final ProjectAnnotationResult after both introspection and
// annotation phases have completed, decoupling the coordinator service from
// the specific caching implementation.
type BuildResultCachePort interface {
	// Get retrieves a build result from the cache using the given key.
	//
	// Takes key (string) which is the cache key to look up.
	//
	// Returns *ProjectAnnotationResult which is the cached build result.
	// Returns error when the lookup fails. Returns ErrCacheMiss if the item is not
	// in the cache.
	Get(ctx context.Context, key string) (*annotator_dto.ProjectAnnotationResult, error)

	// Set stores a build result in the cache.
	//
	// Takes key (string) which identifies the cached result.
	// Takes result (*annotator_dto.ProjectAnnotationResult) which contains the
	// data to store.
	//
	// Returns error when the cache operation fails.
	Set(ctx context.Context, key string, result *annotator_dto.ProjectAnnotationResult) error

	// Clear removes all entries from the cache.
	//
	// Returns error when the operation fails. Does not return an error if the
	// cache is already empty.
	Clear(ctx context.Context) error
}

// IntrospectionCachePort is the driven port for caching type introspection
// results (Tier 1). This is the first tier of the two-tier caching system that
// stores the expensive results of Phase 1 (buildUnifiedGraph + virtualiseModule
// + initialiseTypeResolver).
//
// Phase 1 introspection is 100-1000x more expensive than Phase 2 annotation
// because it invokes packages.Load() for full Go type analysis. However, it
// only depends on the <script> blocks from .pk files and all .go files in
// the project.
//
// When only <template>, <style>, or <i18n> blocks change, the cached
// introspection data can be reused, skipping directly to Phase 2 and achieving
// 5-10x performance improvement.
type IntrospectionCachePort interface {
	// Get retrieves an introspection cache entry using the introspection hash.
	//
	// Takes key (string) which is the introspection hash to look up.
	//
	// Returns *IntrospectionCacheEntry which contains the cached data.
	// Returns error when the lookup fails. Returns ErrCacheMiss if the item is not
	// found. Implementations must validate the entry before returning, including
	// version checks and nil checks.
	Get(ctx context.Context, key string) (*IntrospectionCacheEntry, error)

	// Set stores an introspection cache entry.
	//
	// Takes key (string) which identifies the cache entry.
	// Takes entry (*IntrospectionCacheEntry) which is the data to store.
	//
	// Returns error when the entry is not valid (ErrInvalidCacheEntry).
	Set(ctx context.Context, key string, entry *IntrospectionCacheEntry) error

	// Clear removes all entries from the introspection cache.
	//
	// Returns error when the cache cannot be cleared.
	Clear(ctx context.Context) error
}

// FileHashCachePort is the driven port for caching file content hashes keyed
// by modification time. It implements coordinator_domain.FileHashCachePort and
// collection_domain.HybridPersistencePort.
//
// This provides a persistent cache to optimise the coordinator's hash
// calculation logic, implementing a "stat-then-read" optimisation that
// reduces disk I/O.
//
// Reading all source files to compute hashes is expensive I/O. Using os.Stat()
// to check modification times is fast (metadata-only). This cache
// allows the coordinator to skip reading files whose ModTime has not changed
// since the last hash was computed.
//
// For incremental builds where only 1-2 files changed out of 60+, this reduces
// hash calculation time from 50-200ms to <5ms.
type FileHashCachePort interface {
	// Get retrieves the cached hash for a file if its modification time matches.
	//
	// Takes path (string) which is the file path to look up in the cache.
	// Takes modTime (time.Time) which is the expected modification time.
	//
	// Returns hash (string) which is the cached hash value.
	// Returns found (bool) which is true if the cache contains an entry with a
	// matching modification time, or false if not found or the time differs.
	Get(ctx context.Context, path string, modTime time.Time) (hash string, found bool)

	// Set stores or updates the hash for a file with its change time.
	// Call this after you have calculated a new hash from the file's content.
	//
	// Takes path (string) which is the file path to store the hash for.
	// Takes modTime (time.Time) which is when the file was last changed.
	// Takes hash (string) which is the hash value to store.
	Set(ctx context.Context, path string, modTime time.Time, hash string)

	// Load reads the cache from persistent storage into memory. This should be
	// called during coordinator initialisation and should handle a missing cache
	// file gracefully.
	//
	// Returns error when reading or parsing the cache file fails.
	Load(ctx context.Context) error

	// Persist saves the cache from memory to storage. Call this during shutdown
	// to keep the cache for the next restart.
	//
	// Returns error when the write to storage fails.
	Persist(ctx context.Context) error
}

// UnsubscribeFunc is a function that stops receiving build notifications.
type UnsubscribeFunc func()

// BuildOption configures how a build is started.
type BuildOption func(*buildOptions)

// buildOptions configures build behaviour. Fields are ordered for optimal
// memory alignment.
type buildOptions struct {
	// Resolver overrides the coordinator's default resolver for this build,
	// enabling per-module resolution in LSP contexts where files from
	// different modules may be analysed.
	Resolver resolver_domain.ResolverPort

	// InspectionCacheHints maps file paths to script hashes for cache checking.
	InspectionCacheHints map[string]string

	// CausationID links related build operations for tracing.
	CausationID string

	// ChangedFiles lists the files that have changed; used with SkipInspection.
	ChangedFiles []string

	// SkipInspection skips full inspection when true, checking only changed files.
	SkipInspection bool

	// FaultTolerant allows annotation to continue despite errors when true.
	FaultTolerant bool
}

// CodeEmitterPort is the driven port for code emission.
//
// It generates executable Go code from annotations, used in dev-i mode where
// the coordinator must provide fully-emitted artefacts to the interpreted
// build orchestrator. In other modes (dev/prod), code emission is handled
// externally by the generator CLI tool.
type CodeEmitterPort interface {
	// EmitCode creates Go code from an annotation result.
	//
	// Takes annotationResult (*annotator_dto.AnnotationResult) which holds the
	// parsed annotation data.
	// Takes request (generator_dto.GenerateRequest) which specifies what to generate.
	//
	// Returns []byte which contains the generated Go code.
	// Returns []*ast_domain.Diagnostic which lists any warnings or issues found.
	// Returns error when code generation fails.
	EmitCode(
		ctx context.Context,
		annotationResult *annotator_dto.AnnotationResult,
		request generator_dto.GenerateRequest,
	) ([]byte, []*ast_domain.Diagnostic, error)
}

// ClientScriptEmitterPort is the driven port for transpiling and
// storing the <script lang="ts"> block of a .pk file as a client-side
// JavaScript artefact.
type ClientScriptEmitterPort interface {
	// EmitJS transpiles and stores a client-side script for a .pk
	// component, returning the artefact ID used to build the script
	// URL on the rendered page.
	//
	// Takes source (string) which is the TypeScript/JavaScript source
	// from the <script> block.
	// Takes pagePath (string) which is the project-relative path of the
	// component without its .pk extension (e.g. "partials/header").
	// Takes moduleName (string) which is the Go module name used for
	// @/ alias resolution in imports.
	// Takes outputDir (string) which is ignored; the registry handles
	// storage.
	// Takes minify (bool) which is ignored; the capabilities pipeline
	// handles minification.
	//
	// Returns artefactID (string) which identifies the stored artefact.
	// Returns error when transpilation or registry storage fails.
	EmitJS(
		ctx context.Context,
		source string,
		pagePath string,
		moduleName string,
		outputDir string,
		minify bool,
	) (artefactID string, err error)
}

// DiagnosticOutputPort defines the driven port for diagnostic output.
// Following hexagonal architecture, it lets the coordinator emit diagnostics
// without knowing the context (CLI outputs ANSI to stderr; LSP is a no-op).
type DiagnosticOutputPort interface {
	// OutputDiagnostics writes formatted diagnostics to the appropriate output
	// channel for the current execution context.
	//
	// Takes diagnostics ([]*ast_domain.Diagnostic) which is the list of
	// diagnostics to output.
	// Takes sourceContents (map[string][]byte) which maps file paths to their
	// source code for formatting with context.
	// Takes isError (bool) which indicates whether these diagnostics contain
	// errors rather than only warnings.
	OutputDiagnostics(diagnostics []*ast_domain.Diagnostic, sourceContents map[string][]byte, isError bool)
}

// WithCausationID sets a causation identifier on a build request, letting a
// trigger tag the request with its own identity.
//
// Takes id (string) which specifies the causation identifier.
//
// Returns BuildOption which sets the causation ID on the build options.
func WithCausationID(id string) BuildOption {
	return func(o *buildOptions) {
		o.CausationID = id
	}
}

// WithFaultTolerance enables fault-tolerant mode where annotation continues
// even when errors occur, so LSP features can work on the valid parts of the
// code.
//
// Returns BuildOption which sets the builder to fault-tolerant mode.
func WithFaultTolerance() BuildOption {
	return func(o *buildOptions) {
		o.FaultTolerant = true
	}
}

// WithResolver sets a custom resolver for path resolution during the build.
// This enables per-module resolution in LSP contexts where files from
// different Go modules may be analysed.
//
// Takes resolver (ResolverPort) which provides path resolution for this build.
//
// Returns BuildOption which sets the resolver override.
func WithResolver(resolver resolver_domain.ResolverPort) BuildOption {
	return func(o *buildOptions) {
		o.Resolver = resolver
	}
}

// withSkipInspection enables the fast rebuild path that reuses cached
// inspection results. Use this when only template, style, or i18n blocks have
// changed, not script blocks.
//
// Takes changedFiles ([]string) which lists the files that have changed.
//
// Returns BuildOption which sets up the build to skip inspection.
func withSkipInspection(changedFiles []string) BuildOption {
	return func(o *buildOptions) {
		o.SkipInspection = true
		o.ChangedFiles = changedFiles
	}
}

// withInspectionCacheHints sets script hashes for files to enable cache lookups
// during inspection.
//
// Takes hints (map[string]string) which maps file paths to their script hashes.
//
// Returns BuildOption which applies the inspection cache hints to the build.
func withInspectionCacheHints(hints map[string]string) BuildOption {
	return func(o *buildOptions) {
		o.InspectionCacheHints = hints
	}
}

// withFullInspection requests a full inspection, which is the default
// behaviour. Use it when the caller wants to be explicit about their
// intent.
//
// Returns BuildOption which sets the build to perform full inspection.
func withFullInspection() BuildOption {
	return func(o *buildOptions) {
		o.SkipInspection = false
	}
}
