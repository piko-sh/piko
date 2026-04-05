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

package collection_domain

import (
	"context"
	"go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

// ProviderType indicates how a provider fetches and handles data.
// Static fetches all data at build time, Dynamic fetches at runtime, and
// Hybrid uses build-time data with runtime updates.
type ProviderType string

const (
	// ProviderTypeStatic indicates a provider whose data is fetched at build time.
	// All content is embedded in the binary, resulting in zero runtime overhead.
	ProviderTypeStatic ProviderType = "static"

	// ProviderTypeDynamic indicates all data is fetched at RUNTIME.
	//
	// Characteristics:
	//   - GenerateRuntimeFetcher() is called during build
	//   - GetCollection() is replaced with a runtime API call
	//   - Runtime overhead (API latency + caching)
	//   - Ideal for: E-commerce products, user-generated content, real-time data
	//
	ProviderTypeDynamic ProviderType = "dynamic"

	// ProviderTypeHybrid indicates data is fetched at both build and runtime.
	//
	// Characteristics:
	//   - FetchStaticContent() creates a snapshot at build time
	//   - GenerateRuntimeFetcher() creates revalidation code
	//   - GetCollection() returns snapshot immediately, revalidates in background
	//   - Balanced: Fast initial load + eventual freshness
	//   - Ideal for: Team pages, testimonials, product catalogues
	//
	// This implements Incremental Static Regeneration (ISR) pattern.
	ProviderTypeHybrid ProviderType = "hybrid"
)

// CollectionProvider is the primary DRIVEN port for data source adapters.
// It implements collection_domain.CollectionProvider and hybridCapableProvider.
//
// Every data source (markdown files, CMS, database, etc.) must implement this
// interface to integrate with the collection system.
//
// Design Philosophy:
//   - Clean contract: Framework doesn't know about provider internals
//   - Type declaration: Provider explicitly declares its behaviour
//   - Flexible: Supports static, dynamic, and hybrid data models
//
// Implementation Guide:
//   - Implement Name() and Type() first (required)
//   - For static providers: Implement FetchStaticContent()
//   - For dynamic providers: Implement GenerateRuntimeFetcher() and
//     RuntimeProvider
//   - For hybrid providers: Implement both
//
// See: internal/collection/design.md for detailed implementation guidance.
type CollectionProvider interface {
	// Name returns the unique name for this provider.
	//
	// This name is used in user code to refer to the provider:
	// <template p-collection:provider="NAME">
	// data.GetCollection(..., data.WithProvider("NAME"))
	// Use lowercase names with hyphens between words (e.g. "markdown",
	// "headless-cms", "postgres", "contentful").
	//
	// Returns string which is the provider name.
	Name() string

	// Type returns how this provider's data should be handled.
	//
	// This determines which methods the framework will call and how it will
	// generate code.
	//
	// Returns ProviderType which is one of ProviderTypeStatic,
	// ProviderTypeDynamic, or ProviderTypeHybrid.
	Type() ProviderType

	// DiscoverCollections scans the provider's data source and returns
	// information about available collections.
	//
	// This is called during project initialisation and validation.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - config: Provider configuration from piko.config.yaml
	//
	// Returns:
	//   - A slice of CollectionInfo describing each discovered collection
	//   - An error if discovery fails
	DiscoverCollections(ctx context.Context, config collection_dto.ProviderConfig) ([]collection_dto.CollectionInfo, error)

	// ValidateTargetType checks if a user's target struct is compatible with
	// this provider's data.
	//
	// This is called during type resolution for GetCollection() calls to ensure
	// the user's struct can be populated from the provider's data.
	//
	// Parameters:
	//   - targetType: Go AST expression for the user's struct type
	//
	// Returns:
	//   - nil if the type is valid
	//   - An error describing the incompatibility
	//
	// This is for validation only. Actual field mapping happens later.
	ValidateTargetType(targetType ast.Expr) error

	// FetchStaticContent retrieves all content from a collection at BUILD TIME.
	//
	// This method is called for Static and Hybrid providers during the build
	// process. It must return ALL items in the collection, as they will be
	// embedded in the compiled binary.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - collectionName: The collection to fetch (e.g., "blog", "products")
	//   - source: Where the content files are located (local or external module)
	//
	// Returns:
	//   - A slice of ContentItem (one per item in the collection)
	//   - An error if fetching fails
	//
	// For Dynamic providers: Should return an error indicating this operation is
	// not supported.
	FetchStaticContent(ctx context.Context, collectionName string, source collection_dto.ContentSource) ([]collection_dto.ContentItem, error)

	// GenerateRuntimeFetcher generates Go code for fetching data at RUNTIME.
	//
	// This method is called for Dynamic and Hybrid providers during the build
	// process. It must generate a complete, compilable Go function that will fetch
	// data when the compiled application runs.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - collectionName: The collection to fetch (e.g., "products")
	//   - targetType: Go AST expression for the user's target struct
	//   - options: Fetch options (locale, caching, filters, etc.)
	//
	// Returns:
	//   - RuntimeFetcherCode containing the generated function and metadata
	//   - An error if code generation fails
	//
	// The generated function should:
	//   1. Check cache (if caching enabled)
	//   2. Fetch fresh data from the source
	//   3. Transform data to match targetType
	//   4. Handle errors gracefully
	//   5. Update cache
	//
	// For Static providers: Should return an error indicating this operation is
	// not supported.
	GenerateRuntimeFetcher(
		ctx context.Context,
		collectionName string,
		targetType ast.Expr,
		options collection_dto.FetchOptions,
	) (*collection_dto.RuntimeFetcherCode, error)

	//
	// These methods support Incremental Static Regeneration (ISR). They are used
	// by hybrid providers to enable ETag-based staleness detection and background
	// revalidation.
	//
	// Implementation is OPTIONAL for pure static or pure dynamic providers.
	// Default implementations that return errors are acceptable.
	// ComputeETag computes a content fingerprint for hybrid mode staleness
	// detection.
	//
	// This method is called at build time to create an ETag that represents the
	// current state of the collection. At runtime, this ETag is compared against
	// the current content to detect changes.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - collectionName: The collection to compute ETag for
	//   - source: Where the content files are located (local or external module)
	//
	// Returns:
	//   - An ETag string (format varies by provider)
	//   - An error if computation fails
	//
	// ETag format conventions:
	//   - Markdown: "md-{xxhash64 hex}" (e.g., "md-a1b2c3d4e5f67890")
	//   - CMS/API: Pass through the provider's ETag header
	//   - Database: "db-{hash of row versions}"
	//
	// For pure Static/Dynamic providers: May return ("", ErrNotSupported)
	ComputeETag(ctx context.Context, collectionName string, source collection_dto.ContentSource) (string, error)

	// ValidateETag checks if the current content matches an expected ETag.
	//
	// This method is called at runtime during background revalidation to determine
	// if content has changed since the last check.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - collectionName: The collection to validate
	//   - expectedETag: The ETag from the last known good state
	//
	// Returns:
	//   - currentETag: The current ETag (same as expected if unchanged)
	//   - changed: true if the content has changed
	//   - error: If validation fails
	//
	// This method should be efficient:
	//   - For files: Check modification times (avoid reading content)
	//   - For APIs: Use conditional requests (If-None-Match header)
	//   - For databases: Check row versions without full scan
	//
	// For pure Static/Dynamic providers: May return ("", false, ErrNotSupported)
	ValidateETag(ctx context.Context, collectionName string, expectedETag string, source collection_dto.ContentSource) (currentETag string, changed bool, err error)

	// GenerateRevalidator generates Go code for runtime ETag validation and
	// refresh.
	//
	// This method is called at build time for hybrid providers to generate the
	// code that will run in background goroutines to revalidate stale content.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - collectionName: The collection to revalidate
	//   - targetType: Go AST expression for the user's target struct
	//   - config: Hybrid configuration (TTL, stale-if-error, etc.)
	//
	// Returns:
	//   - RuntimeFetcherCode containing the revalidation function
	//   - An error if code generation fails
	//
	// The generated function should:
	//   1. Call ValidateETag() to check for changes
	//   2. If unchanged: Return early (no-op)
	//   3. If changed: Fetch fresh content via FetchStaticContent()
	//   4. Serialise new content to FlatBuffer
	//   5. Update the hybrid registry
	//
	// For pure Static/Dynamic providers: May return (nil, ErrNotSupported)
	GenerateRevalidator(
		ctx context.Context,
		collectionName string,
		targetType ast.Expr,
		config collection_dto.HybridConfig,
	) (*collection_dto.RuntimeFetcherCode, error)
}

// RuntimeProvider defines the interface for providers that operate at runtime.
// It implements collection_domain.RuntimeProvider and runtime.RuntimeProvider.
//
// Dynamic and Hybrid providers must implement this interface AND register it
// with the runtime when the application starts.
//
// Design Philosophy:
//   - Separation: Build-time (CollectionProvider) and runtime (RuntimeProvider)
//   - Simplicity: Runtime interface is much simpler (no code generation)
//   - Performance: Providers control caching and optimisation
//
// Implementation Guide:
//  1. Implement this interface in a separate file (e.g., runtime_provider.go)
//  2. Register with pikoruntime.RegisterProvider() in main.go
//  3. Implement efficient caching strategy
//  4. Handle errors gracefully (use fallbacks if available)
type RuntimeProvider interface {
	// Name returns the identifier that matches the build-time provider.
	//
	// The runtime uses this name to find the correct provider to call.
	//
	// Returns string which is the provider name.
	Name() string

	// Fetch retrieves live data and unmarshals it into the target slice.
	//
	// Parameters:
	//   - ctx: Request context (for cancellation, tracing, etc.)
	//   - collectionName: The collection to fetch
	//   - options: Fetch options (locale, caching, filters, etc.)
	//   - target: Pointer to a slice of the user's struct type (e.g., *[]Post)
	//
	// Returns:
	//   - nil on success (target slice is populated)
	//   - An error if fetching fails
	//
	// Implementation notes:
	//   - Check cache first (if caching enabled)
	//   - Make API call / database query
	//   - Use reflection to populate target slice
	//   - Update cache on successful fetch
	//   - Log errors for observability
	Fetch(
		ctx context.Context,
		collectionName string,
		options *collection_dto.FetchOptions,
		target any,
	) error
}

// CollectionEncoderPort is the driven port for encoding collection data
// to binary format. It abstracts the encoding mechanism from domain logic,
// allowing the collection hexagon to pack data into a binary blob suitable for
// embedding via //go:embed.
//
// Design philosophy:
//   - Decoupling: Domain does not depend on FlatBuffers directly.
//   - Zero-copy: Implementations must produce formats optimised for direct
//     memory access.
//   - Sortable: Binary format must support efficient lookups via binary search.
//
// Implementation guide:
//   - Implement in collection_adapters/flatbuffer_encoder.go.
//   - Sort items by route before encoding (required for binary search).
//   - Use FlatBuffers schema from collection_schema/collection.fbs.
//   - Return raw bytes suitable for direct embedding.
type CollectionEncoderPort interface {
	// EncodeCollection packs a full collection into a binary blob.
	//
	// This method transforms a slice of ContentItems into a binary format that:
	//   1. Can be embedded directly into the compiled binary (via //go:embed)
	//   2. Supports O(log n) lookups by route at runtime
	//   3. Allows lazy decoding of individual items
	//   4. Minimises memory overhead (zero-copy access)
	//
	// Parameters:
	//   - items: Slice of content items to encode
	//
	// Returns:
	//   - Binary blob containing the encoded collection
	//   - An error if encoding fails
	//
	// Implementation requirements:
	//   - MUST sort items by route (ascending) before encoding
	//   - MUST encode metadata as JSON bytes
	//   - MUST encode ASTs using ast_adapters.EncodeAST()
	//   - MUST produce a format compatible with DecodeCollectionItem()
	EncodeCollection(items []collection_dto.ContentItem) ([]byte, error)

	// DecodeCollectionItem extracts a single item from an encoded
	// collection.
	//
	// This method performs a binary search lookup in the blob and returns the raw
	// data for a specific route without decoding the entire collection.
	//
	// Takes blob ([]byte) which is the encoded collection blob.
	// Takes route (string) which is the route to look up (e.g., "/docs/actions").
	//
	// Returns metadataJSON ([]byte) which is the raw metadata JSON bytes.
	// Returns contentAST ([]byte) which is the raw content AST bytes.
	// Returns excerptAST ([]byte) which is the raw excerpt AST bytes (may be nil).
	// Returns err (error) when the route is not found or blob is invalid.
	//
	// Implementations must use binary search on sorted routes (O(log n)), must not
	// decode the entire collection, and must return raw bytes for lazy
	// decoding. The runtime registry handles final decoding into
	// domain objects.
	DecodeCollectionItem(blob []byte, route string) (metadataJSON, contentAST, excerptAST []byte, err error)
}

// ProviderRegistryPort is the DRIVER port for managing providers.
//
// This interface defines how the framework manages the collection of registered
// providers. The actual implementation (adapter) lives in collection_adapters.
//
// Design Philosophy:
//   - Simple CRUD operations
//   - Thread-safe (implementations must handle concurrency)
//   - Immutable after initialisation (register during startup only)
type ProviderRegistryPort interface {
	// Register adds a provider to the registry.
	//
	// Parameters:
	//   - provider: The provider to register
	//
	// Returns:
	//   - An error if registration fails (e.g., duplicate name)
	//
	// Thread-safety: Must be safe to call from multiple goroutines during startup.
	Register(provider CollectionProvider) error

	// Get retrieves a provider by name.
	//
	// Takes name (string) which is the provider's unique identifier.
	//
	// Returns CollectionProvider which is the requested provider.
	// Returns bool which is true if found, false if not.
	//
	// Concurrency: Must be safe to call at the same time as Register.
	Get(name string) (CollectionProvider, bool)

	// List returns all registered provider names.
	//
	// Intended for diagnostics, validation, and error messages.
	//
	// Returns []string which contains the provider names.
	//
	// Concurrency: Must be safe to call from multiple goroutines.
	List() []string

	// Has checks whether a provider with the given name is registered.
	//
	// Takes name (string) which is the provider name to look up.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	//
	// This method is safe to call from multiple goroutines.
	Has(name string) bool
}

// CollectionService is the primary DRIVING port for the Collection Hexagon.
//
// This is the main entry point that other hexagons (Coordinator, Annotator) use
// to interact with the collection system.
//
// Integration Points:
//   - Coordinator: Calls ProcessCollectionDirective() during build
//     orchestration
//   - Annotator: Calls ProcessGetCollectionCall() during type resolution
type CollectionService interface {
	// ProcessCollectionDirective expands a p-collection directive into entry
	// points.
	//
	// This method is called by the Coordinator when it encounters a .pk file with
	// a p-collection directive. It orchestrates the provider to generate virtual
	// entry points for each content item.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - directive: Parsed information from the p-collection directive
	//
	// Returns:
	//   - A slice of virtual entry points (one per content item for static
	//     providers)
	//   - An error if expansion fails
	//
	// Workflow:
	//   1. Look up provider by name
	//   2. Check provider type
	//   3. For Static: Call FetchStaticContent(), create entry point per item
	//   4. For Dynamic: Create single dynamic route entry point
	//   5. For Hybrid: Do both
	//
	// See: internal/collection/design.md Section 9.1 for integration details
	ProcessCollectionDirective(
		ctx context.Context,
		directive *collection_dto.CollectionDirectiveInfo,
	) ([]*collection_dto.CollectionEntryPoint, error)

	// ProcessGetCollectionCall handles data.GetCollection() in user code.
	//
	// This method is called by the Annotator's TypeResolver when it encounters
	// a GetCollection() call. It receives semantic information extracted from the
	// Piko AST and generates the appropriate annotation for the Generator.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - collectionName: Name of the collection (e.g., "blog", "products")
	//   - targetTypeName: Name of the target struct type (e.g., "Post", "Product")
	//   - targetTypeExpr: Go AST expression for the target type (for validation)
	//   - options: Fetch options (provider, locale, filters, sorting, pagination)
	//
	// Returns:
	//   - A GoGeneratorAnnotation with instructions for the Generator
	//   - An error if processing fails
	//
	// Workflow:
	//   1. Look up provider from options (or use default)
	//   2. Validate target type compatibility with provider
	//   3. For Static: Fetch content, generate static slice literal annotation
	//   4. For Dynamic: Generate runtime fetch call annotation
	//   5. For Hybrid: Generate hybrid annotation (snapshot + revalidation)
	//
	// See: internal/collection/design.md Section 9.2 for integration details
	ProcessGetCollectionCall(
		ctx context.Context,
		collectionName string,
		targetTypeName string,
		targetTypeExpr ast.Expr,
		options any,
	) (*ast_domain.GoGeneratorAnnotation, error)

	// ValidateConfiguration checks all provider configurations at startup.
	//
	// This method is called during application initialisation to ensure:
	//   - All referenced providers are registered
	//   - Provider configurations are valid
	//   - Collections exist and are accessible
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeouts
	//   - config: The complete project configuration
	//
	// Returns:
	//   - nil if all configurations are valid
	//   - An error describing any validation failures
	ValidateConfiguration(ctx context.Context, config *Config) error

	// Close releases resources held by the service.
	//
	// This closes any sandboxes created for external module content sources
	// during directive processing.
	//
	// Returns error when resource cleanup fails.
	Close() error
}

// CollectionDirectiveInfo contains parsed information from a p-collection
// directive. It is passed from the Coordinator to the CollectionService.
type CollectionDirectiveInfo struct {
	// CacheConfig holds cache settings for dynamic providers; nil uses defaults.
	CacheConfig *collection_dto.CacheConfig

	// Filters contains extra options for the provider.
	Filters map[string]any

	// ProviderName is the name of the collection provider used to fetch data.
	ProviderName string

	// CollectionName is the name of the collection to fetch, such as "blog" or
	// "products".
	CollectionName string

	// LayoutPath is the path to the .pk file that serves as the layout template
	// for generated pages. Example: "pages/blog/{slug}.pk".
	LayoutPath string

	// RoutePath is the base route path for generated pages.
	//
	// For static providers, each item gets its own route under this path.
	// For dynamic providers, this becomes a dynamic route pattern.
	RoutePath string
}

// EntryPoint represents a single page to be built.
//
// This is what the CollectionService returns to the Coordinator.
//
// Design Note: This is a simplified version. The actual EntryPoint type will
// be imported from annotator_dto in the real implementation.
type EntryPoint struct {
	// InitialProps contains pre-computed props for virtual entry points.
	//
	// For static collections: Contains the content item's metadata and AST
	// For dynamic collections: Empty (props fetched at runtime)
	InitialProps map[string]any

	// Path is the file path to the .pk layout file.
	Path string

	// RoutePatternOverride is the custom route for this entry point.
	RoutePatternOverride string

	// DynamicCollection is the collection name for dynamic routes.
	DynamicCollection string

	// DynamicProvider is the provider name for dynamic routes.
	DynamicProvider string

	// Locale is the language and region setting for this entry point.
	Locale string

	// TranslationKey groups related entry points across different locales.
	TranslationKey string

	// IsPage indicates whether this entry point is a page.
	IsPage bool

	// IsVirtual indicates this entry point was generated, not a real file.
	IsVirtual bool

	// IsDynamic indicates whether this route fetches data at run time.
	IsDynamic bool
}

// Config represents the collection-specific project configuration.
//
// This configuration is extracted from the main project configuration
// (piko.config.yaml) and passed to the collection service for validation at
// startup.
//
// Design Philosophy:
//   - Explicit declaration: All collections and providers must be declared
//   - Fail-fast: Invalid configuration fails at startup, not at runtime
//   - Type-safe: Structured configuration prevents typos and misconfiguration
type Config struct {
	// Providers maps provider names to their settings.
	//
	// Keys must match registered provider names (e.g. "markdown", "headless-cms").
	Providers map[string]ProviderConfigEntry

	// Collections maps collection names to their settings.
	// Keys are collection names such as "blog" or "docs".
	Collections map[string]CollectionConfigEntry

	// DefaultProvider is the name of the provider to use when none is given.
	//
	// This provider is used when GetCollection is called without WithProvider.
	// Must be registered in the provider registry.
	//
	// Default: "markdown"
	DefaultProvider string
}

// ProviderConfigEntry contains configuration for a specific provider.
type ProviderConfigEntry struct {
	// Custom holds settings specific to this provider as key-value pairs.
	Custom map[string]any

	// BasePath is the base folder for this provider.
	// For markdown providers, this is the project root containing the "content"
	// folder.
	BasePath string

	// Enabled indicates whether this provider should be registered. Default: true.
	Enabled bool
}

// CollectionConfigEntry contains configuration for a specific collection.
type CollectionConfigEntry struct {
	// Provider is the name of the provider to use for this collection.
	// If empty, Config.DefaultProvider is used.
	Provider string

	// Enabled indicates whether this collection is active. Default is true.
	Enabled bool
}

// HybridRegistryPort manages hybrid collection caches for ISR (Incremental
// Static Regeneration).
//
// This port abstracts the hybrid registry, allowing tests to inject mock
// implementations that don't rely on global state. The hybrid registry stores
// build-time snapshots and manages background revalidation for hybrid
// collections.
//
// Implements collection_adapters.hybridRegistryAccessor and HybridRegistryPort
// interfaces.
//
// Thread-safety: All implementations must be safe for concurrent use.
type HybridRegistryPort interface {
	// Register stores a build-time snapshot for runtime use.
	//
	// This is called from generated init() functions to register the
	// embedded FlatBuffer blob and its ETag for hybrid mode operation.
	//
	// Parameters:
	//   - ctx: Context for logging and trace propagation
	//   - providerName: The provider that generated this snapshot
	//   - collectionName: The collection this snapshot belongs to
	//   - blob: The FlatBuffer-serialised content (from //go:embed)
	//   - etag: The content fingerprint at build time
	//   - config: Hybrid mode configuration (TTL, stale-if-error, etc.)
	Register(ctx context.Context, providerName, collectionName string, blob []byte, etag string, config collection_dto.HybridConfig)

	// GetBlob returns the current FlatBuffer blob and whether revalidation is
	// needed.
	//
	// Parameters:
	//   - ctx: Context for logging and trace propagation
	//
	// Returns:
	//   - blob: The current FlatBuffer blob (nil if not registered)
	//   - needsRevalidation: True if TTL has expired and revalidation should run
	GetBlob(ctx context.Context, providerName, collectionName string) (blob []byte, needsRevalidation bool)

	// GetETag returns the current ETag for a hybrid collection.
	//
	// Takes providerName (string) which identifies the data provider.
	// Takes collectionName (string) which specifies the collection to query.
	//
	// Returns string which is the ETag value for debugging and monitoring.
	GetETag(providerName, collectionName string) string

	// Has checks whether a hybrid collection is registered.
	//
	// Takes providerName (string) which identifies the provider.
	// Takes collectionName (string) which identifies the collection.
	//
	// Returns bool which is true if the collection is registered, false otherwise.
	Has(providerName, collectionName string) bool

	// List returns all registered hybrid collection keys.
	//
	// Keys use the format "provider:collection".
	//
	// Returns []string which contains all registered collection keys.
	List() []string

	// TriggerRevalidation starts background revalidation for a hybrid collection.
	//
	// Takes providerName (string) which identifies the data provider.
	// Takes collectionName (string) which identifies the collection to revalidate.
	//
	// The function returns at once; revalidation runs in the background.
	// Concurrent calls are combined into a single revalidation.
	TriggerRevalidation(ctx context.Context, providerName, collectionName string)
}

// HybridPersistencePort manages persistent storage of hybrid collection state.
//
// This port enables hybrid collections to survive process restarts by
// persisting their current state (blob, ETag, timestamps) to disk. Without
// persistence, revalidated content would be lost on restart, forcing
// unnecessary re-fetches.
//
// Design Philosophy:
//   - Atomic writes: Prevents corruption from process crashes mid-write
//   - Graceful degradation: Missing cache file is not an error (cold start)
//   - JSON format: Human-readable for debugging, efficient enough for cache
//     size
//
// Thread-safety: All implementations must be safe for concurrent use.
type HybridPersistencePort interface {
	// Load reads persisted hybrid state from storage into the registry.
	//
	// This should be called during application startup to restore hybrid
	// collection state from the previous run.
	//
	// Graceful degradation:
	//   - Missing file: Not an error, registry starts empty (cold start)
	//   - Corrupted file: Log warning, registry starts empty
	//   - Read error: Return error for caller to handle
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//
	// Returns:
	//   - nil on success (including when file doesn't exist)
	//   - An error if loading fails
	Load(ctx context.Context) error

	// Persist writes current hybrid state from the registry to storage.
	//
	// This should be called during graceful shutdown to preserve the
	// current state for the next run.
	//
	// Atomic write strategy:
	//   1. Write to temporary file
	//   2. Rename temp file to target (atomic on POSIX)
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//
	// Returns:
	//   - nil on success
	//   - An error if persisting fails
	Persist(ctx context.Context) error
}

// StaticCollectionRegistryPort manages static collection blobs for runtime
// access.
//
// This port abstracts the static collection registry, allowing tests to inject
// mock implementations. The registry stores FlatBuffer blobs registered via
// //go:embed directives in generated code.
//
// Design Philosophy:
//   - Enables isolated testing of static collection lookups
//   - Breaks dependency on package-level global state
//   - Allows testing without actual FlatBuffer serialisation
//
// Thread-safety: All implementations must be safe for concurrent use.
type StaticCollectionRegistryPort interface {
	// Register stores a binary blob for a static collection.
	//
	// This is called by generated code in init() functions (from //go:embed
	// directives) to register the embedded binary data for runtime access.
	//
	// Takes ctx (context.Context) which carries logging context for
	// trace/request ID propagation.
	// Takes collectionName (string) which identifies the collection
	// (e.g., "docs", "blog").
	// Takes data ([]byte) which is the FlatBuffer binary blob embedded via
	// //go:embed.
	Register(ctx context.Context, collectionName string, data []byte)

	// GetItem retrieves a single item from a static collection by route.
	//
	// This performs an O(log n) binary search lookup in the FlatBuffer blob.
	//
	// Parameters:
	//   - ctx: Context for logging and trace propagation
	//   - collectionName: The collection identifier
	//   - route: The URL/route to look up (e.g., "/docs/actions")
	//
	// Returns:
	//   - CollectionItemResult containing metadata and ASTs
	//   - An error if the collection or route is not found
	GetItem(ctx context.Context, collectionName, route string) (*CollectionItemResult, error)

	// GetAllItems retrieves all items from a static collection.
	//
	// This is used for search operations that need to scan all items.
	//
	// Parameters:
	//   - collectionName: The collection identifier
	//
	// Returns:
	//   - A slice of metadata maps (excluding ASTs for efficiency)
	//   - An error if the collection is not found
	GetAllItems(collectionName string) ([]map[string]any, error)

	// Has checks whether a static collection with the given name is registered.
	//
	// Takes collectionName (string) which is the name to look for.
	//
	// Returns bool which is true if the collection exists.
	Has(collectionName string) bool

	// List returns all registered static collection names.
	List() []string
}

// RuntimeProviderRegistryPort manages runtime providers for dynamic data
// fetching.
//
// This port abstracts the runtime provider registry, allowing tests to inject
// mock implementations. Runtime providers are registered during application
// startup and handle data fetching for dynamic and hybrid collections.
//
// Design Philosophy:
//   - Enables isolated testing of runtime provider lookup and invocation
//   - Breaks dependency on package-level global state
//   - Allows verification of provider registration and fetch interactions
//
// Thread-safety: All implementations must be safe for concurrent use.
type RuntimeProviderRegistryPort interface {
	// Register adds a runtime provider to the registry.
	//
	// Parameters:
	//   - provider: The runtime provider to register
	//
	// Returns:
	//   - An error if provider name conflicts with existing provider
	Register(provider RuntimeProvider) error

	// Get retrieves a runtime provider by name.
	//
	// Takes name (string) which is the unique identifier for the provider.
	//
	// Returns RuntimeProvider which is the requested provider.
	// Returns error when the provider is not found.
	Get(name string) (RuntimeProvider, error)

	// List returns all registered runtime provider names.
	List() []string

	// Has checks whether a runtime provider with the given name exists.
	//
	// Takes name (string) which is the provider name to look for.
	//
	// Returns bool which is true if the provider exists, false otherwise.
	Has(name string) bool

	// Fetch is a convenience method that looks up a provider and fetches data.
	//
	// This is the runtime entry point for dynamic collections.
	//
	// Parameters:
	//   - ctx: Request context for cancellation and tracing
	//   - providerName: Name of the provider to use
	//   - collectionName: Name of the collection to fetch
	//   - options: Fetch options (locale, filters, cache config, etc.)
	//   - target: Pointer to slice to populate (e.g., *[]Post)
	//
	// Returns:
	//   - An error if fetch fails or provider not found
	Fetch(ctx context.Context, providerName, collectionName string, options *collection_dto.FetchOptions, target any) error
}

// ASTDecoderPort abstracts AST decoding from domain logic.
//
// This port enables testing of collection item decoding without
// depending on the actual ast_adapters implementation.
//
// Design Philosophy:
//   - Decouples domain code from ast_adapters
//   - Enables isolated testing of decoding logic
//   - Follows hexagonal architecture boundaries
type ASTDecoderPort interface {
	// Decode converts FlatBuffer bytes into a TemplateAST.
	//
	// Parameters:
	//   - ctx: context for logging and cancellation propagation
	//   - data: FlatBuffer-encoded AST bytes
	//
	// Returns:
	//   - The decoded TemplateAST
	//   - An error if decoding fails
	Decode(ctx context.Context, data []byte) (*ast_domain.TemplateAST, error)

	// DecodeForRender converts FlatBuffer bytes into a TemplateAST optimised
	// for rendering. Location and range fields are skipped since the renderer
	// never reads them.
	//
	// Parameters:
	//   - ctx: context for logging and cancellation propagation
	//   - data: FlatBuffer-encoded AST bytes
	//
	// Returns:
	//   - The decoded TemplateAST
	//   - An error if decoding fails
	DecodeForRender(ctx context.Context, data []byte) (*ast_domain.TemplateAST, error)
}

// SearchIndexLoaderPort abstracts loading search indexes from the search
// adapter layer.
//
// This port enables testing search functionality without requiring actual
// FlatBuffer indexes to be registered in the global search index registry.
// It decouples the collection domain from search_adapters and follows
// hexagonal architecture boundaries.
type SearchIndexLoaderPort interface {
	// GetIndex retrieves a search index reader for a collection and search mode.
	//
	// Parameters:
	//   - collectionName: The collection to search (e.g., "docs", "blog")
	//   - searchMode: The search mode ("fast" or "smart")
	//
	// Returns:
	//   - A search_domain.IndexReaderPort for querying the index
	//   - An error if the index is not found or cannot be loaded
	GetIndex(collectionName, searchMode string) (any, error)
}

// CollectionItemsLoaderPort abstracts loading collection items for search
// result hydration. This port enables testing without requiring actual
// FlatBuffer blobs in the global static collection registry.
type CollectionItemsLoaderPort interface {
	// GetAllItems retrieves all items from a static collection.
	//
	// Parameters:
	//   - collectionName: The collection identifier
	//
	// Returns:
	//   - A slice of metadata maps (excluding ASTs for efficiency)
	//   - An error if the collection is not found
	GetAllItems(collectionName string) ([]map[string]any, error)
}

// SearchServicePort defines the contract for searching collections.
//
// This is the primary entry point for search functionality, abstracting all
// search-related dependencies behind a single interface.
//
// Design Philosophy:
//   - Single responsibility: searching collections
//   - Enables isolated testing without search infrastructure
//   - Decouples collection domain from search implementation details
type SearchServicePort interface {
	// Search executes a search query against a static collection.
	//
	// Takes collectionName (string) which identifies the collection to search.
	// Takes currentPageData (map[string]any) which contains current page data if
	// called from a collection page, or nil otherwise.
	// Takes config (SearchConfig) which specifies search parameters including
	// query, fields, and scoring thresholds.
	// Takes searchMode (string) which sets the search mode ("fast" or "smart").
	//
	// Returns []SearchResult which contains matching documents sorted by
	// relevance.
	// Returns error when the collection is not found or search fails.
	Search(
		ctx context.Context,
		collectionName string,
		currentPageData map[string]any,
		config SearchConfig,
		searchMode string,
	) ([]SearchResult, error)
}
