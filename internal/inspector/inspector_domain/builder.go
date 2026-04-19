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

// This file defines the TypeBuilder, which orchestrates the parsing,
// caching, and serialisation of Go source code type information. It is designed
// to be highly testable through the use of dependency injection.

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	goast "go/ast"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultMaxWorkers is the default number of workers used to parse files at the
	// same time.
	defaultMaxWorkers = 4

	// resolveTypeRecursionGuard is the maximum depth for resolving types.
	resolveTypeRecursionGuard = 20

	// listSeparator is the separator used between items in type lists.
	listSeparator = ", "
)

// builderCacheKeyGenerator defines how to create a stable cache key.
type builderCacheKeyGenerator interface {
	// Generates documentation output from the analysed source files.
	//
	// Takes config (inspector_dto.Config) which specifies the generation settings.
	// Takes sourceContents (map[string][]byte) which provides the source file
	// contents keyed by file path.
	// Takes scriptHashes (map[string]string) which maps file paths to their
	// content hashes.
	//
	// Returns string which is the generated documentation.
	// Returns error when generation fails.
	Generate(ctx context.Context, config inspector_dto.Config, sourceContents map[string][]byte, scriptHashes map[string]string) (string, error)
}

// builderSourceParser defines the contract for parsing Go source files into ASTs.
type builderSourceParser interface {
	// Parse processes source files and returns their AST representations.
	//
	// Takes sourceContents (map[string][]byte) which maps file paths to their
	// source code.
	// Takes maxWorkers (int) which limits the number of concurrent parsers.
	//
	// Returns map[string]*goast.File which maps file paths to their parsed ASTs.
	// Returns error when parsing fails for any source file.
	Parse(ctx context.Context, sourceContents map[string][]byte, maxWorkers int) (map[string]*goast.File, error)
}

// builderPackageLoader defines the interface for loading Go packages from source.
type builderPackageLoader interface {
	// Load loads Go packages using the given configuration and file overlays.
	//
	// Takes config (inspector_dto.Config) which specifies how packages are loaded.
	// Takes overlay (map[string][]byte) which provides in-memory file contents.
	//
	// Returns []*packages.Package which contains the loaded package data.
	// Returns error when package loading fails.
	Load(ctx context.Context, config inspector_dto.Config, overlay map[string][]byte) ([]*packages.Package, error)
}

// builderEncoder defines how package data is converted into a DTO.
type builderEncoder interface {
	// Encode converts the given packages into a type data representation.
	//
	// Takes pkgs ([]*packages.Package) which contains the packages to encode.
	// Takes moduleName (string) which is the user's module path for filtering
	// internal packages from external dependencies.
	//
	// Returns *inspector_dto.TypeData which holds the encoded type information.
	// Returns error when encoding fails.
	Encode(pkgs []*packages.Package, moduleName string) (*inspector_dto.TypeData, error)
}

// TypeBuilder orchestrates the building and caching of Go source code type
// information. It supports full mode (using go/packages) and lite mode
// (using go/parser only), and is safe for concurrent use.
type TypeBuilder struct {
	// provider is the optional cache provider for loading and saving type data.
	provider TypeDataProvider

	// parser parses source files into ASTs; defaults to defaultParser if nil.
	parser builderSourceParser

	// loader loads Go packages from source files for type checking.
	loader builderPackageLoader

	// encoder converts loaded packages into data transfer objects.
	encoder builderEncoder

	// keyGen generates cache keys for type building results.
	keyGen builderCacheKeyGenerator

	// sandboxFactory creates sandboxes with validated paths. Passed to the
	// default key generator when no custom keyGen is provided.
	sandboxFactory safedisk.Factory

	// querierByKey stores TypeQuerier instances by their cache key.
	querierByKey map[string]*TypeQuerier

	// typeDataByKey stores type data for each module, keyed by cache key.
	typeDataByKey map[string]*inspector_dto.TypeData

	// liteBuilder handles type building in lite mode; nil when lite mode is off.
	liteBuilder *LiteBuilder

	// currentCacheKey holds the cache key for the current or last build.
	currentCacheKey string

	// config holds the inspector settings for the current module.
	config inspector_dto.Config

	// mu guards access to the per-module caches during concurrent use.
	mu sync.RWMutex

	// liteMode enables a lightweight parsing mode that skips go/packages.
	liteMode bool
}

// TypeBuilderOption is a function that configures a TypeBuilder.
type TypeBuilderOption func(*TypeBuilder)

// defaultKeyGenerator implements builderCacheKeyGenerator with default behaviour.
type defaultKeyGenerator struct {
	// sandboxFactory creates sandboxes with validated paths. When set, the
	// factory is preferred over NewNoOpSandbox for fallback sandbox creation.
	sandboxFactory safedisk.Factory
}

// Generate generates a cache key from the given configuration and sources.
//
// Takes config (inspector_dto.Config) which specifies the inspection settings.
// Takes sources (map[string][]byte) which contains the source files to hash.
// Takes scriptHashes (map[string]string) which provides pre-computed hashes for
// scripts.
//
// Returns string which is the generated cache key.
// Returns error when key generation fails.
func (g *defaultKeyGenerator) Generate(_ context.Context, config inspector_dto.Config, sources map[string][]byte, scriptHashes map[string]string) (string, error) {
	return generateCacheKey(config, sources, scriptHashes, g.sandboxFactory)
}

// defaultParser implements builderSourceParser using the standard Go parser.
type defaultParser struct{}

// Parse converts source files into abstract syntax trees.
//
// Takes sources (map[string][]byte) which maps file paths to their contents.
// Takes maxWorkers (int) which limits how many workers parse files at once.
//
// Returns map[string]*goast.File which maps file paths to their parsed trees.
// Returns error when parsing fails for any source file.
func (*defaultParser) Parse(_ context.Context, sources map[string][]byte, maxWorkers int) (map[string]*goast.File, error) {
	return parseSourceContents(sources, maxWorkers)
}

// defaultLoader implements builderPackageLoader using the standard loader.
type defaultLoader struct{}

// Load uses the Go packages library to load and type-check packages from
// source.
//
// Takes config (inspector_dto.Config) which specifies the package loading
// settings.
// Takes overlay (map[string][]byte) which provides in-memory file contents to
// use instead of reading from disk.
//
// Returns []*packages.Package which contains the loaded and type-checked
// packages.
// Returns error when package loading or type-checking fails.
func (*defaultLoader) Load(ctx context.Context, config inspector_dto.Config, overlay map[string][]byte) ([]*packages.Package, error) {
	return loadPackagesFromSource(ctx, config, overlay)
}

// defaultEncoder implements builderEncoder with standard formatting.
type defaultEncoder struct{}

// Encode converts loaded packages into a portable TypeData DTO.
//
// Takes pkgs ([]*packages.Package) which contains the parsed Go packages to
// process.
// Takes moduleName (string) which is the user's module path for filtering
// internal packages from external dependencies.
//
// Returns *inspector_dto.TypeData which holds the encoded type information.
// Returns error when extraction or encoding fails.
func (*defaultEncoder) Encode(pkgs []*packages.Package, moduleName string) (*inspector_dto.TypeData, error) {
	return extractAndEncode(pkgs, moduleName)
}

// NewTypeBuilder creates a new TypeBuilder using functional options.
// If key parts (parser, loader, and so on) are not given via options, it uses
// default values that are ready for production.
//
// Takes config (inspector_dto.Config) which sets the configuration, including
// the maximum number of parse workers.
// Takes opts (...TypeBuilderOption) which provides optional functional options
// to change the builder's dependencies.
//
// Returns *TypeBuilder which is ready for use after applying all options and
// defaults.
func NewTypeBuilder(config inspector_dto.Config, opts ...TypeBuilderOption) *TypeBuilder {
	if config.MaxParseWorkers == nil {
		config.MaxParseWorkers = new(defaultMaxWorkers)
	}

	m := &TypeBuilder{
		provider:        nil,
		parser:          nil,
		loader:          nil,
		encoder:         nil,
		keyGen:          nil,
		typeDataByKey:   make(map[string]*inspector_dto.TypeData),
		querierByKey:    make(map[string]*TypeQuerier),
		liteBuilder:     nil,
		currentCacheKey: "",
		config:          config,
		mu:              sync.RWMutex{},
		liteMode:        false,
	}

	for _, opt := range opts {
		opt(m)
	}

	if m.keyGen == nil {
		m.keyGen = &defaultKeyGenerator{sandboxFactory: m.sandboxFactory}
	}
	if m.parser == nil {
		m.parser = &defaultParser{}
	}
	if m.loader == nil {
		m.loader = &defaultLoader{}
	}
	if m.encoder == nil {
		m.encoder = &defaultEncoder{}
	}

	return m
}

// SetConfig updates the builder's configuration. This is used by the LSP to
// reconfigure the builder for different modules without recreating it.
//
// The builder keeps per-module caches, so changing the configuration does not
// clear existing cached data. Each module's data is stored by its unique cache
// key (based on BaseDir, ModuleName, and source contents).
//
// Takes config (inspector_dto.Config) which provides the new configuration.
//
// Safe for concurrent use. Acquires the internal mutex.
func (m *TypeBuilder) SetConfig(config inspector_dto.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.MaxParseWorkers == nil {
		config.MaxParseWorkers = new(defaultMaxWorkers)
	}

	config.UseStandardLoader = m.config.UseStandardLoader

	configChanged := m.config.BaseDir != config.BaseDir || m.config.ModuleName != config.ModuleName
	m.config = config

	if configChanged {
		_, l := logger_domain.From(context.Background(), log)
		l.Internal("TypeBuilder config updated",
			logger_domain.String("baseDir", config.BaseDir),
			logger_domain.String("moduleName", config.ModuleName),
			logger_domain.Int("cached_modules", len(m.querierByKey)))
	}
}

// Build performs the main analysis and type-building process. It is
// idempotent per module (cache key).
//
// Takes sourceContents (map[string][]byte) which provides virtual Go source
// files from ModuleVirtualiser.
// Takes scriptHashes (map[string]string) which maps .pk file paths to SHA1
// hashes of their script block content. This is critical for cache
// invalidation when scripts change, as sourceContents only contains stub
// files that do not reflect script changes. In lite mode, scriptHashes is
// ignored and caching is disabled.
//
// Returns error when the build process fails.
//
// Not safe for concurrent use. Acquires the internal mutex for the duration
// of the build.
func (m *TypeBuilder) Build(ctx context.Context, sourceContents map[string][]byte, scriptHashes map[string]string) error {
	ctx, l := logger_domain.From(ctx, log)
	ctx, span, l := l.Span(ctx, "Build", logger_domain.Int("source_file_count", len(sourceContents)))
	defer span.End()

	BuilderBuildCount.Add(ctx, 1)
	startTime := time.Now()
	defer func() { BuilderBuildDuration.Record(ctx, float64(time.Since(startTime).Milliseconds())) }()

	l.Internal("--- [INSPECTOR MANAGER START] --- Starting Build Process ---")
	defer l.Internal("--- [INSPECTOR MANAGER END] --- Build Process Finished ---")

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.liteMode {
		return m.buildLite(ctx, sourceContents)
	}

	cacheKey := m.generateCacheKeyForBuild(ctx, sourceContents, scriptHashes)

	if cacheKey != "" {
		if _, exists := m.querierByKey[cacheKey]; exists {
			m.currentCacheKey = cacheKey
			l.Internal("[INSPECTOR] Build skipped: module already cached in memory.",
				logger_domain.String("cacheKey", cacheKey[:min(8, len(cacheKey))]))
			return nil
		}
	}

	if cacheKey != "" && m.buildFromCache(ctx, cacheKey, sourceContents) {
		m.currentCacheKey = cacheKey
		return nil
	}

	if err := m.buildFromSource(ctx, cacheKey, sourceContents); err != nil {
		BuilderBuildErrorCount.Add(ctx, 1)
		l.ReportError(span, err, "Failed to build from source")
		return fmt.Errorf("building type data from source: %w", err)
	}

	effectiveKey := cmp.Or(cacheKey, "__no_cache_key__")
	m.currentCacheKey = effectiveKey

	if cacheKey != "" {
		m.saveToCache(ctx, cacheKey, m.typeDataByKey[cacheKey])
	}

	return nil
}

// GetTypeData returns the serialised type data for the current module.
// Build must be called first.
//
// Returns *inspector_dto.TypeData which contains the serialised type data.
// Returns error when Build has not been called or no data exists for the
// current cache key.
//
// Safe for concurrent use.
func (m *TypeBuilder) GetTypeData() (*inspector_dto.TypeData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.currentCacheKey == "" {
		return nil, errors.New("typeBuilder has not been built yet")
	}
	td, exists := m.typeDataByKey[m.currentCacheKey]
	if !exists {
		return nil, errors.New("typeBuilder has not been built yet")
	}
	return td, nil
}

// GetQuerier returns the query engine for inspecting types in the current module.
//
// Returns *TypeQuerier which is the query engine, or nil if not built.
// Returns bool which is true when the querier is available.
//
// Safe for concurrent use. Build must be called first.
func (m *TypeBuilder) GetQuerier() (*TypeQuerier, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.currentCacheKey == "" {
		return nil, false
	}
	querier, exists := m.querierByKey[m.currentCacheKey]
	if !exists || querier == nil {
		return nil, false
	}
	return querier, true
}

// GenerateCacheKeyForTest is a test-only helper to expose the cache key
// generation logic.
//
// Takes sourceContents (map[string][]byte) which provides the source file
// contents keyed by path.
// Takes scriptHashes (map[string]string) which provides precomputed hashes
// for scripts.
//
// Returns string which is the generated cache key.
// Returns error when key generation fails.
func (m *TypeBuilder) GenerateCacheKeyForTest(ctx context.Context, sourceContents map[string][]byte, scriptHashes map[string]string) (string, error) {
	return m.keyGen.Generate(ctx, m.config, sourceContents, scriptHashes)
}

// IsLiteMode returns true if the builder is configured to use lite mode.
//
// Returns bool which indicates whether lite mode is enabled.
func (m *TypeBuilder) IsLiteMode() bool {
	return m.liteMode
}

// generateCacheKeyForBuild generates a cache key for the current build
// configuration.
//
// Takes sourceContents (map[string][]byte) which provides the source file
// contents for cache key generation.
// Takes scriptHashes (map[string]string) which provides script hashes for
// cache validation.
//
// Returns string which is the generated cache key, or empty if generation
// fails.
func (m *TypeBuilder) generateCacheKeyForBuild(ctx context.Context, sourceContents map[string][]byte, scriptHashes map[string]string) string {
	ctx, l := logger_domain.From(ctx, log)
	cacheKey, err := m.keyGen.Generate(ctx, m.config, sourceContents, scriptHashes)
	if err != nil {
		l.Warn("Failed to generate cache key, proceeding with live build.", logger_domain.Error(err))
		return ""
	}
	return cacheKey
}

// buildFromCache tries to load type data from the cache and set up the
// querier.
//
// Takes cacheKey (string) which identifies the cached type data to load.
// Takes sourceContents (map[string][]byte) which provides the source files to
// parse.
//
// Returns bool which is true when loading succeeds, or false when the cache
// has no data or parsing fails.
func (m *TypeBuilder) buildFromCache(ctx context.Context, cacheKey string, sourceContents map[string][]byte) bool {
	ctx, l := logger_domain.From(ctx, log)
	if m.provider == nil {
		l.Internal("[INSPECTOR] Skipping cache load (no provider).")
		return false
	}

	l.Internal("[INSPECTOR] Attempting to load from cache provider...", logger_domain.String("cacheKey", cacheKey))
	cachedData, err := m.provider.GetTypeData(ctx, cacheKey)
	if err != nil || cachedData == nil {
		l.Internal("[INSPECTOR] CACHE MISS or error.", logger_domain.Error(err))
		return false
	}

	l.Internal("[INSPECTOR] CACHE HIT. Building querier from cached data.")
	allScriptBlocks, err := m.parser.Parse(ctx, sourceContents, *m.config.MaxParseWorkers)
	if err != nil {
		l.Warn("Failed to parse script blocks on cache hit. Forcing live build.", logger_domain.Error(err))
		return false
	}

	m.typeDataByKey[cacheKey] = cachedData
	m.querierByKey[cacheKey] = NewTypeQuerier(allScriptBlocks, cachedData, m.config)
	l.Internal("[INSPECTOR] Successfully built from cache.")
	return true
}

// buildFromSource runs the full build process without using the cache.
//
// Takes cacheKey (string) which identifies where to store the results. May be
// empty if cache key generation failed.
// Takes sourceContents (map[string][]byte) which provides the source files to
// parse and build.
//
// Returns error when parsing, loading, encoding, or validation fails.
func (m *TypeBuilder) buildFromSource(ctx context.Context, cacheKey string, sourceContents map[string][]byte) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[INSPECTOR] Performing full live build.")

	l.Internal("[INSPECTOR-STEP 3a/5] Parsing source contents into ASTs...")
	allScriptBlocks, err := m.parser.Parse(ctx, sourceContents, *m.config.MaxParseWorkers)
	if err != nil {
		return fmt.Errorf("failed to parse source contents into ASTs: %w", err)
	}
	l.Internal("[INSPECTOR] Successfully parsed source contents.", logger_domain.Int("script_blocks_found", len(allScriptBlocks)))

	l.Internal("[INSPECTOR-STEP 3b/5] Loading packages from virtual source overlay...")
	loadedPackages, err := m.loader.Load(ctx, m.config, sourceContents)
	if err != nil {
		return fmt.Errorf("failed to load packages from source: %w", err)
	}
	l.Internal("[INSPECTOR] Successfully loaded Go packages.")

	l.Internal("[INSPECTOR-STEP 3c/5] Encoding live package data...")
	typeData, err := m.encoder.Encode(loadedPackages, m.config.ModuleName)
	if err != nil {
		return fmt.Errorf("failed to encode live package data: %w", err)
	}
	l.Internal("[INSPECTOR] Successfully encoded package data.")

	l.Internal("[INSPECTOR-STEP 3d/5] Validating encoded DTO consistency...")
	if err := validate(typeData); err != nil {
		return fmt.Errorf("encoded TypeData failed validation: %w", err)
	}
	l.Internal("[INSPECTOR] Encoded DTO passed validation.")

	effectiveKey := cmp.Or(cacheKey, "__no_cache_key__")
	m.typeDataByKey[effectiveKey] = typeData
	m.querierByKey[effectiveKey] = NewTypeQuerier(allScriptBlocks, typeData, m.config)
	l.Internal("[INSPECTOR] Inspector built successfully from live source.")

	return nil
}

// saveToCache stores the given type data in the cache.
//
// Takes cacheKey (string) which identifies the cache entry.
// Takes typeData (*inspector_dto.TypeData) which contains the type to store.
func (m *TypeBuilder) saveToCache(ctx context.Context, cacheKey string, typeData *inspector_dto.TypeData) {
	ctx, l := logger_domain.From(ctx, log)
	if m.provider == nil {
		return
	}

	l.Internal("[INSPECTOR] Saving newly generated type data to cache...", logger_domain.String("cacheKey", cacheKey))
	if err := m.provider.SaveTypeData(ctx, cacheKey, typeData); err != nil {
		l.Warn("Failed to save type data to cache.", logger_domain.Error(err))
	} else {
		l.Internal("[INSPECTOR] Cache save successful.")
	}
}

// buildLite runs a lightweight build using the LiteBuilder.
// This skips go/packages and uses only go/parser for AST parsing.
//
// Takes sourceContents (map[string][]byte) which maps file paths to their
// source code bytes.
//
// Returns error when the LiteBuilder is not set up, the build fails, or
// type data cannot be retrieved.
func (m *TypeBuilder) buildLite(ctx context.Context, sourceContents map[string][]byte) error {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("[INSPECTOR-LITE] Starting lite build...")

	if m.liteBuilder == nil {
		return errors.New("lite mode enabled but LiteBuilder not initialised (missing stdlib data?)")
	}

	const liteCacheKey = "__lite_mode__"

	if _, exists := m.querierByKey[liteCacheKey]; exists {
		l.Internal("[INSPECTOR-LITE] Build skipped: manager is already built.")
		m.currentCacheKey = liteCacheKey
		return nil
	}

	if err := m.liteBuilder.Build(ctx, sourceContents); err != nil {
		return fmt.Errorf("lite build failed: %w", err)
	}

	typeData, err := m.liteBuilder.GetTypeData()
	if err != nil {
		return fmt.Errorf("failed to get lite build TypeData: %w", err)
	}

	querier, ok := m.liteBuilder.GetQuerier()
	if !ok {
		return errors.New("lite builder did not produce a querier")
	}

	m.typeDataByKey[liteCacheKey] = typeData
	m.querierByKey[liteCacheKey] = querier
	m.currentCacheKey = liteCacheKey

	l.Internal("[INSPECTOR-LITE] Lite build complete.",
		logger_domain.Int("package_count", len(typeData.Packages)),
	)

	return nil
}

// WithProvider sets the provider for storing and fetching type data.
//
// Takes provider (TypeDataProvider) which handles type data storage and
// retrieval.
//
// Returns TypeBuilderOption which configures the TypeBuilder with the given
// provider.
func WithProvider(provider TypeDataProvider) TypeBuilderOption {
	return func(m *TypeBuilder) {
		m.provider = provider
	}
}

// WithParser sets the source code parser to use.
//
// Takes parser (builderSourceParser) which provides the logic for parsing
// source files.
//
// Returns TypeBuilderOption which configures the TypeBuilder to use the
// given parser.
func WithParser(parser builderSourceParser) TypeBuilderOption {
	return func(m *TypeBuilder) {
		m.parser = parser
	}
}

// WithBuilderPackageLoader sets the Go package loader used to build types.
//
// Takes loader (builderPackageLoader) which provides the logic to load
// packages for type resolution.
//
// Returns TypeBuilderOption which sets up a TypeBuilder with the given loader.
func WithBuilderPackageLoader(loader builderPackageLoader) TypeBuilderOption {
	return func(m *TypeBuilder) {
		m.loader = loader
	}
}

// WithEncoder sets the encoder used to convert type data.
//
// Takes encoder (builderEncoder) which provides the encoding method for type
// data.
//
// Returns TypeBuilderOption which configures a TypeBuilder to use the given
// encoder.
func WithEncoder(encoder builderEncoder) TypeBuilderOption {
	return func(m *TypeBuilder) {
		m.encoder = encoder
	}
}

// WithBuilderCacheKeyGenerator sets the cache key generator for a TypeBuilder.
//
// Takes keyGen (builderCacheKeyGenerator) which is the function used to create
// cache keys.
//
// Returns TypeBuilderOption which sets up a TypeBuilder to use the given key
// generator.
func WithBuilderCacheKeyGenerator(keyGen builderCacheKeyGenerator) TypeBuilderOption {
	return func(m *TypeBuilder) {
		m.keyGen = keyGen
	}
}

// WithBuilderSandboxFactory sets the sandbox factory for the TypeBuilder. The
// factory is passed to the default cache key generator so it can create
// sandboxes through the factory instead of using NewNoOpSandbox directly.
//
// Takes factory (safedisk.Factory) which creates sandboxes with validated
// paths.
//
// Returns TypeBuilderOption which configures the TypeBuilder with the factory.
func WithBuilderSandboxFactory(factory safedisk.Factory) TypeBuilderOption {
	return func(m *TypeBuilder) {
		m.sandboxFactory = factory
	}
}

// WithLiteMode configures the builder to use lite mode, which bypasses
// go/packages. Use it for REPL or WASM cases where go/packages is not
// available.
//
// When lite mode is enabled:
//   - Build only requires sourceContents (scriptHashes is ignored)
//   - Caching is disabled (no provider support)
//   - Only AST parsing is done (no type-checking)
//
// Takes stdlibData (*inspector_dto.TypeData) which contains pre-made TypeData
// for standard library types.
//
// Returns TypeBuilderOption which sets up a TypeBuilder for lite mode.
func WithLiteMode(stdlibData *inspector_dto.TypeData) TypeBuilderOption {
	return func(m *TypeBuilder) {
		m.liteMode = true
		m.liteBuilder = nil
		if stdlibData != nil {
			lb, err := NewLiteBuilder(stdlibData, m.config)
			if err == nil {
				m.liteBuilder = lb
			}
		}
	}
}
