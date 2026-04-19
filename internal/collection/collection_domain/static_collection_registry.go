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
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"piko.sh/piko/internal/ast/ast_adapters"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/cache/cache_domain"
	"piko.sh/piko/internal/collection/collection_adapters"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/collection/collection_schema"
	coll_fb "piko.sh/piko/internal/collection/collection_schema/collection_schema_gen"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// logKeyCollection is the log attribute key for collection names.
	logKeyCollection = "collection"

	// errFmtGettingCollectionBlob is the format string used when a collection
	// blob cannot be retrieved from the registry.
	errFmtGettingCollectionBlob = "getting collection blob for %q: %w"

	// metadataKeyURL is the metadata key carrying the original URL of a
	// collection item. Read by metadataToContentItems so downstream
	// navigation/sitemap builders can surface the URL on ContentItem.URL.
	metadataKeyURL = "URL"

	// rootIndexSlug is the canonical slug for an index file at a collection
	// root. Lookups whose normalised path is empty resolve to this slug so
	// requests like "/" or "/<collection>/" reach the root index.
	rootIndexSlug = "index"

	// pathSeparator is the forward slash used in slug paths.
	pathSeparator = "/"

	// maxLookupSlugBytes caps the length of a runtime lookup key. Defends
	// against pathological URL captures from being used as map keys or log
	// fields.
	maxLookupSlugBytes = 1024

	// MaxStaticCollectionItems caps the number of items decoded from an
	// embedded FlatBuffer blob. Mirrors the encoder cap as defence-in-depth
	// so a malformed or hostile embedded blob cannot allocate unbounded
	// slices and maps at registration time.
	MaxStaticCollectionItems = 100_000

	// logKeyItemIndex is the log attribute key for the position of an item
	// inside a collection blob.
	logKeyItemIndex = "item_index"
)

var (
	_ StaticCollectionRegistryPort = (*defaultStaticCollectionRegistry)(nil)

	_ ASTDecoderPort = (*defaultASTDecoder)(nil)

	// staticItemsByPointer maps the backing-array address of a pre-decoded items
	// slice to its owning staticCollectionData. This enables TryGetCachedNavigation
	// to detect whether a []map[string]any slice is the same instance returned by
	// GetStaticCollectionItems without requiring callers to change their code.
	//
	staticItemsByPointer sync.Map

	// staticCollectionRegistry stores all static collection data with pre-decoded
	// metadata.
	//
	// Generated code populates this during package initialisation via //go:embed
	// directives. Each collection is stored as a compact FlatBuffer binary for AST
	// lookups, with metadata pre-decoded at registration time so that runtime
	// access is a zero-allocation field read.
	//
	// Thread-safety: Safe for concurrent reads after initialisation.
	staticCollectionRegistry = struct {
		collections map[string]*staticCollectionData
		encoder     CollectionEncoderPort
		mu          sync.RWMutex
	}{
		collections: make(map[string]*staticCollectionData),
		encoder:     collection_adapters.NewFlatBufferEncoder(),
	}
)

// cachedAST holds the decoded content and excerpt ASTs for a single item.
// The renderer never mutates decoded AST nodes, so these are safe to share
// across concurrent requests.
type cachedAST struct {
	// content holds the decoded main content AST for the item.
	content *ast_domain.TemplateAST

	// excerpt holds the decoded excerpt AST for the item, or nil if absent.
	excerpt *ast_domain.TemplateAST
}

// staticCollectionData holds the raw FlatBuffer blob and pre-decoded derived
// data for a single static collection.
//
// Metadata is decoded eagerly at registration time so that runtime lookups
// are zero-allocation field accesses. ASTs are lazily decoded and cached
// per route since the renderer treats them as read-only.
type staticCollectionData struct {
	// navigation stores lazily initialised navigation trees keyed by config.
	// Static collection data is immutable, so each config produces a single
	// tree that lives for the process lifetime.
	navigation sync.Map

	// astCache stores decoded ASTs per item so repeated requests for the
	// same slug avoid FlatBuffer decoding entirely. Keys are slug strings,
	// values are *cachedAST.
	astCache sync.Map

	// slugIndex maps item slugs to indices in items for O(1) metadata lookup.
	// Items are keyed by slug (not URL) so the same collection item can be
	// rendered at multiple URL prefixes by different consuming pages.
	slugIndex map[string]int

	// blob is the raw FlatBuffer binary for single-item AST lookups.
	blob []byte

	// items holds pre-decoded metadata for every item in the collection.
	items []map[string]any
}

// defaultStaticCollectionRegistry implements StaticCollectionRegistryPort by
// wrapping the package-level global registry, so the interface can be used for
// DI while maintaining backward compatibility with generated code that uses
// global functions.
type defaultStaticCollectionRegistry struct {
	// encoder decodes collection items from stored blobs.
	encoder CollectionEncoderPort

	// astDecoder decodes AST bytes into AST nodes.
	astDecoder ASTDecoderPort
}

// Register implements StaticCollectionRegistryPort.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes collectionName (string) which identifies the collection to register.
// Takes data ([]byte) which contains the collection content.
func (*defaultStaticCollectionRegistry) Register(ctx context.Context, collectionName string, data []byte) {
	RegisterStaticCollectionBlob(ctx, collectionName, data)
}

// GetItem retrieves a single item from the named collection by slug.
// Implements StaticCollectionRegistryPort.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes collectionName (string) which identifies the collection to search.
// Takes slug (string) which is the item's canonical identifier (e.g.
// "anthropic" for content/integrations/anthropic.md).
//
// Returns *CollectionItemResult which contains the decoded item data.
// Returns error when the collection cannot be loaded or the item is not found.
func (r *defaultStaticCollectionRegistry) GetItem(ctx context.Context, collectionName, slug string) (*CollectionItemResult, error) {
	data, err := getCollectionData(collectionName)
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingCollectionBlob, collectionName, err)
	}

	metadata, contentAST, excerptAST, err := getItemFromData(ctx, r.encoder, r.astDecoder, data, collectionName, slug)
	if err != nil {
		return nil, fmt.Errorf("getting item from collection %q at slug %q: %w", collectionName, slug, err)
	}

	_, l := logger_domain.From(ctx, log)
	l.Trace("GetItem: item found successfully",
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String(logKeySlug, slug))

	return &CollectionItemResult{
		Metadata:   metadata,
		ContentAST: contentAST,
		ExcerptAST: excerptAST,
	}, nil
}

// GetAllItems returns all items from the named static collection.
// Implements StaticCollectionRegistryPort.
//
// Takes collectionName (string) which identifies the collection to retrieve.
//
// Returns []map[string]any which contains the metadata for each item.
// Returns error when the collection cannot be loaded or has a schema mismatch.
func (*defaultStaticCollectionRegistry) GetAllItems(collectionName string) ([]map[string]any, error) {
	return GetStaticCollectionItems(collectionName)
}

// Has reports whether a static collection is registered with the given name.
// Implements StaticCollectionRegistryPort.
//
// Takes collectionName (string) which is the name of the collection to check.
//
// Returns bool which is true if the collection exists, false otherwise.
func (*defaultStaticCollectionRegistry) Has(collectionName string) bool {
	return HasStaticCollection(collectionName)
}

// List returns all registered static collection names.
// Implements StaticCollectionRegistryPort.
//
// Returns []string which contains the names of all registered collections.
//
// Safe for concurrent use.
func (*defaultStaticCollectionRegistry) List() []string {
	staticCollectionRegistry.mu.RLock()
	defer staticCollectionRegistry.mu.RUnlock()
	return listStaticCollectionNames()
}

// defaultASTDecoder implements ASTDecoderPort using the ast_adapters package.
type defaultASTDecoder struct{}

// Decode implements ASTDecoderPort.
//
// Takes ctx (context.Context) which carries logging and cancellation context.
// Takes data ([]byte) which contains the encoded AST representation.
//
// Returns *ast_domain.TemplateAST which is the decoded template AST.
// Returns error when the data cannot be decoded.
func (*defaultASTDecoder) Decode(ctx context.Context, data []byte) (*ast_domain.TemplateAST, error) {
	return ast_adapters.DecodeAST(ctx, data)
}

// DecodeForRender implements ASTDecoderPort.
//
// Takes ctx (context.Context) which carries logging and cancellation context.
// Takes data ([]byte) which contains the encoded AST representation.
//
// Returns *ast_domain.TemplateAST which is the decoded template AST with
// location and range fields omitted.
// Returns error when the data cannot be decoded.
func (*defaultASTDecoder) DecodeForRender(ctx context.Context, data []byte) (*ast_domain.TemplateAST, error) {
	return ast_adapters.DecodeASTForRender(ctx, data)
}

// CollectionItemResult holds the data for a single collection item.
type CollectionItemResult struct {
	// Metadata holds key-value pairs from the collection item's JSON data.
	Metadata map[string]any

	// ContentAST is the parsed template AST for the item's main content.
	ContentAST *ast_domain.TemplateAST

	// ExcerptAST is the parsed template AST for the item excerpt; nil if no
	// excerpt.
	ExcerptAST *ast_domain.TemplateAST
}

// navigationConfigKey identifies a unique navigation configuration for use as
// a sync.Map key.
type navigationConfigKey struct {
	// locale is the locale filter for navigation building.
	locale string

	// defaultOrder is the fallback sort order when items lack an explicit weight.
	defaultOrder int

	// includeHidden controls whether hidden items appear in the tree.
	includeHidden bool

	// groupBySection controls whether items are grouped by section heading.
	groupBySection bool
}

// NewDefaultStaticCollectionRegistry creates a StaticCollectionRegistryPort
// that wraps the package-level global registry.
//
// This is the standard way to obtain a StaticCollectionRegistryPort for
// production use. For testing, create a mock implementation instead.
//
// Returns StaticCollectionRegistryPort which provides access to the global
// collection registry.
func NewDefaultStaticCollectionRegistry() StaticCollectionRegistryPort {
	return &defaultStaticCollectionRegistry{
		encoder:    collection_adapters.NewFlatBufferEncoder(),
		astDecoder: &defaultASTDecoder{},
	}
}

// NewDefaultASTDecoder creates an ASTDecoderPort using the standard AST
// adapters.
//
// Returns ASTDecoderPort which is the default decoder ready for use.
func NewDefaultASTDecoder() ASTDecoderPort {
	return &defaultASTDecoder{}
}

// RegisterStaticCollectionBlob registers a binary blob for a static collection
// and eagerly decodes all item metadata for zero-cost runtime access.
//
// This is called by generated code in init() functions (from //go:embed
// directives) to register the embedded binary data. Metadata is decoded once
// here so that GetStaticCollectionItems and GetStaticCollectionItem avoid
// per-request JSON parsing.
//
// Takes collectionName (string) which identifies the collection
// (e.g., "docs", "blog").
// Takes data ([]byte) which contains the FlatBuffer binary blob
// (embedded via //go:embed).
//
// The blob is not copied (zero-copy registration). The byte slice points to
// read-only memory in the executable.
//
// Safe for concurrent use during package initialisation. Uses a mutex to
// protect the registry.
func RegisterStaticCollectionBlob(ctx context.Context, collectionName string, data []byte) {
	ctx, l := logger_domain.From(ctx, log)
	items, slugIndex := decodeAllItemMetadata(ctx, data)

	staticCollectionRegistry.mu.Lock()
	defer staticCollectionRegistry.mu.Unlock()

	collData := &staticCollectionData{
		blob:      data,
		items:     items,
		slugIndex: slugIndex,
	}
	staticCollectionRegistry.collections[collectionName] = collData

	if ptr := sliceDataPointer(items); ptr != 0 {
		staticItemsByPointer.Store(ptr, collData)
	}
	l.Internal("Static collection blob registered",
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.Int("blob_size", len(data)),
		logger_domain.Int("items", len(items)))
}

// GetStaticCollectionItem retrieves a single item from a static collection by
// slug.
//
// Metadata is returned from the pre-decoded slug index (zero allocation).
// ASTs are decoded per-request from FlatBuffer bytes because the rendering
// pipeline may mutate nodes.
//
// Takes collectionName (string) which is the collection identifier.
// Takes slug (string) which is the item's slug (e.g. "anthropic" for
// content/integrations/anthropic.md). Generated page code reads this from the
// matched URL parameter (defaults to {slug}; configurable via p-param).
//
// Returns metadata (map[string]any) which is the pre-decoded JSON metadata.
// Returns contentAST (*ast_domain.TemplateAST) which is the decoded
// content AST from FlatBuffer bytes.
// Returns excerptAST (*ast_domain.TemplateAST) which is the decoded
// excerpt AST from FlatBuffer bytes, or nil if not present.
// Returns err (error) when the collection or slug is not found.
func GetStaticCollectionItem(ctx context.Context, collectionName, slug string) (metadata map[string]any, contentAST *ast_domain.TemplateAST, excerptAST *ast_domain.TemplateAST, err error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("GetStaticCollectionItem called",
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String(logKeySlug, slug))

	data, err := getCollectionData(collectionName)
	if err != nil {
		l.Error("Failed to get collection data",
			logger_domain.String(logKeyCollection, collectionName),
			logger_domain.Error(err))
		return nil, nil, nil, fmt.Errorf(errFmtGettingCollectionBlob, collectionName, err)
	}

	metadata, contentAST, excerptAST, err = getItemFromData(
		ctx, staticCollectionRegistry.encoder, &defaultASTDecoder{}, data, collectionName, slug)
	if err != nil {
		return nil, nil, nil, err
	}

	l.Trace("GetStaticCollectionItem: item found successfully",
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String(logKeySlug, slug))

	return metadata, contentAST, excerptAST, nil
}

// GetStaticCollectionItems returns the pre-decoded metadata for all items in a
// static collection.
//
// This returns the slice that was materialised at registration time. No JSON
// parsing occurs at call time.
//
// Takes collectionName (string) which identifies the collection to retrieve.
//
// Returns []map[string]any which contains the metadata for each item in the
// collection. The returned slice must not be mutated by callers.
// Returns error when the collection is not found.
func GetStaticCollectionItems(collectionName string) ([]map[string]any, error) {
	data, err := getCollectionData(collectionName)
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingCollectionBlob, collectionName, err)
	}

	return data.items, nil
}

// GetStaticCollectionNavigation returns a lazily initialised navigation tree
// for the named static collection and configuration.
//
// The first call for a given collection and config pair builds the tree; all
// subsequent calls return the same instance. This is not a cache - it is
// deferred initialisation of derived data from immutable inputs.
//
// Takes ctx (context.Context) which carries deadlines and request-scoped
// values.
// Takes collectionName (string) which identifies the collection.
// Takes config (collection_dto.NavigationConfig) which controls tree building.
//
// Returns *collection_dto.NavigationGroups which contains the navigation
// trees.
// Returns error when the collection is not found.
func GetStaticCollectionNavigation(
	ctx context.Context,
	collectionName string,
	config collection_dto.NavigationConfig,
) (*collection_dto.NavigationGroups, error) {
	data, err := getCollectionData(collectionName)
	if err != nil {
		return nil, fmt.Errorf(errFmtGettingCollectionBlob, collectionName, err)
	}

	return getOrBuildNavigation(ctx, data, config), nil
}

// TryGetCachedNavigation returns cached navigation for items if they belong to
// a known static collection, or false when the caller must rebuild.
//
// Takes ctx (context.Context) which carries request-scoped values.
// Takes items ([]map[string]any) which is the metadata slice to identify.
// Takes config (collection_dto.NavigationConfig) which controls tree building.
//
// Returns *collection_dto.NavigationGroups which is the cached tree, or nil.
// Returns bool which is true if the cache was hit.
func TryGetCachedNavigation(
	ctx context.Context,
	items []map[string]any,
	config collection_dto.NavigationConfig,
) (*collection_dto.NavigationGroups, bool) {
	ptr := sliceDataPointer(items)
	if ptr == 0 {
		return nil, false
	}
	value, ok := staticItemsByPointer.Load(ptr)
	if !ok {
		return nil, false
	}
	data, valid := value.(*staticCollectionData)
	if !valid {
		return nil, false
	}
	return getOrBuildNavigation(ctx, data, config), true
}

// HasStaticCollection checks if a static collection is registered.
//
// Use it to decide whether to use static or dynamic collection fetching
// in search operations.
//
// Takes collectionName (string) which is the collection identifier.
//
// Returns bool which is true if the collection is registered.
//
// Safe for concurrent use by multiple goroutines.
func HasStaticCollection(collectionName string) bool {
	staticCollectionRegistry.mu.RLock()
	defer staticCollectionRegistry.mu.RUnlock()

	_, exists := staticCollectionRegistry.collections[collectionName]
	return exists
}

// ResetStaticCollectionRegistry clears the static collection registry for test
// isolation.
//
// Should only be called from tests. Clears all registered static collections
// and their pre-decoded data so that tests start with a clean state.
//
// Safe for use from any goroutine, but not concurrently with other registry
// operations.
func ResetStaticCollectionRegistry() {
	staticCollectionRegistry.mu.Lock()
	defer staticCollectionRegistry.mu.Unlock()
	staticCollectionRegistry.collections = make(map[string]*staticCollectionData)
	staticItemsByPointer = sync.Map{}
}

// decodeAllItemMetadata decodes every item's JSON metadata from a FlatBuffer
// blob and builds a slug index for O(1) lookups.
//
// The FlatBuffer "route" field holds the item's slug (the schema field is named
// "route" for legacy reasons; semantically it is now a slug). Items are looked
// up by slug at runtime so multiple consuming pages can render the same item
// at different URL prefixes. The decoded item count is capped against
// MaxStaticCollectionItems so a malformed or hostile embedded blob cannot
// allocate unbounded slices and maps at registration time. Items whose slug
// fails structural validation, or whose metadata fails to unmarshal, are
// dropped with a warning log so silent corruption does not propagate.
//
// Takes ctx (context.Context) which carries the logger for warning diagnostics.
// Takes blob ([]byte) which contains the encoded collection data.
//
// Returns []map[string]any which holds pre-decoded metadata for each item.
// Returns map[string]int which maps item slugs to their indices in the items
// slice.
func decodeAllItemMetadata(ctx context.Context, blob []byte) ([]map[string]any, map[string]int) {
	_, l := logger_domain.From(ctx, log)

	payload, err := collection_schema.Unpack(blob)
	if err != nil {
		l.Warn("Skipping collection blob: schema version mismatch", logger_domain.Error(err))
		return nil, nil
	}

	coll := coll_fb.GetRootAsStaticCollectionFB(payload, 0)
	itemsLength := coll.ItemsLength()
	if itemsLength > MaxStaticCollectionItems {
		l.Warn("Truncating collection blob: item count exceeds cap",
			logger_domain.Int("blob_items", itemsLength),
			logger_domain.Int("max_items", MaxStaticCollectionItems))
		itemsLength = MaxStaticCollectionItems
	}

	items := make([]map[string]any, 0, itemsLength)
	slugIndex := make(map[string]int, itemsLength)

	for i := range itemsLength {
		item := &coll_fb.ContentItemFB{}
		if !coll.Items(item, i) {
			l.Warn("Skipping collection item: FlatBuffer Items returned false",
				logger_domain.Int(logKeyItemIndex, i))
			continue
		}

		metadataJSON := item.MetadataJsonBytes()
		var metadata map[string]any
		if err := cache_domain.CacheAPI.Unmarshal(metadataJSON, &metadata); err != nil {
			l.Warn("Skipping collection item: metadata JSON unmarshal failed",
				logger_domain.Int(logKeyItemIndex, i),
				logger_domain.Error(err))
			continue
		}
		if metadata == nil {
			metadata = make(map[string]any)
		}

		slug := string(item.Route())
		if slug == "" {
			l.Warn("Skipping collection item: empty slug",
				logger_domain.Int(logKeyItemIndex, i))
			continue
		}
		if !utf8.ValidString(slug) {
			l.Warn("Skipping collection item: slug is not valid UTF-8",
				logger_domain.Int(logKeyItemIndex, i))
			continue
		}
		slugIndex[slug] = len(items)

		items = append(items, metadata)
	}

	return items, slugIndex
}

// getItemFromData retrieves a single item using the pre-decoded route index
// for metadata and the AST cache for decoded content trees.
//
// On the first request for a given slug the ASTs are decoded from the
// FlatBuffer blob and stored in the cache. Subsequent requests return the
// cached trees with zero allocations.
//
// Takes encoder (CollectionEncoderPort) which decodes items from the blob.
// Takes decoder (ASTDecoderPort) which decodes AST bytes.
// Takes data (*staticCollectionData) which holds the pre-decoded collection.
// Takes collectionName (string) which identifies the collection for logging.
// Takes slug (string) which identifies the item to look up.
//
// Returns metadata (map[string]any) which is the pre-decoded metadata.
// Returns contentAST (*ast_domain.TemplateAST) which is the decoded content.
// Returns excerptAST (*ast_domain.TemplateAST) which is the decoded excerpt.
// Returns err (error) when the item cannot be found.
func getItemFromData(
	ctx context.Context,
	encoder CollectionEncoderPort,
	decoder ASTDecoderPort,
	data *staticCollectionData,
	collectionName, slug string,
) (metadata map[string]any, contentAST *ast_domain.TemplateAST, excerptAST *ast_domain.TemplateAST, err error) {
	slug = normaliseLookupSlug(slug)

	metadata = resolveMetadata(encoder, data, slug)

	if value, ok := data.astCache.Load(slug); ok {
		if cached, valid := value.(*cachedAST); valid {
			return metadata, cached.content, cached.excerpt, nil
		}
	}

	_, contentASTBytes, excerptASTBytes, lookupErr := lookupItemWithEncoder(
		ctx, encoder, data.blob, collectionName, slug)
	if lookupErr != nil {
		return nil, nil, nil, lookupErr
	}

	content, excerpt, err := decodeASTPair(ctx, decoder, contentASTBytes, excerptASTBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	data.astCache.Store(slug, &cachedAST{content: content, excerpt: excerpt})

	return metadata, content, excerpt, nil
}

// normaliseLookupSlug aligns a runtime lookup key with stored slugs.
//
// Strips a leading slash, trims a trailing slash (chi catch-all captures may
// include either), resolves an empty result to rootIndexSlug so the root
// index page is reachable, and caps length at maxLookupSlugBytes so a
// hostile caller cannot flood log fields or map keys.
//
// Takes slug (string) which is the raw lookup key from the request.
//
// Returns string which is the normalised slug ready for slugIndex / encoder
// lookups.
func normaliseLookupSlug(slug string) string {
	slug = strings.TrimPrefix(slug, pathSeparator)
	slug = strings.TrimSuffix(slug, pathSeparator)
	if slug == "" {
		return rootIndexSlug
	}
	if len(slug) > maxLookupSlugBytes {
		slug = slug[:maxLookupSlugBytes]
	}
	return slug
}

// resolveMetadata looks up pre-decoded metadata for an item by slug. The
// previous implementation accepted URL-style paths and tried trailing-slash /
// /index variants; lookups are now slug-only so a single direct map check
// suffices.
//
// Takes encoder (CollectionEncoderPort) which decodes items from the blob.
// Takes data (*staticCollectionData) which holds the pre-decoded collection.
// Takes slug (string) which identifies the collection item to look up.
//
// Returns map[string]any which is the metadata for the slug, or nil if not
// found.
func resolveMetadata(encoder CollectionEncoderPort, data *staticCollectionData, slug string) map[string]any {
	if index, found := data.slugIndex[slug]; found {
		return data.items[index]
	}

	metadataJSON, _, _, _ := encoder.DecodeCollectionItem(data.blob, slug)
	if metadataJSON == nil {
		return nil
	}

	var metadata map[string]any
	_ = cache_domain.CacheAPI.Unmarshal(metadataJSON, &metadata)
	return metadata
}

// decodeASTPair decodes content and excerpt AST byte slices into their
// respective TemplateAST structures.
//
// Takes ctx (context.Context) which carries logging and cancellation context.
// Takes decoder (ASTDecoderPort) which decodes AST bytes.
// Takes contentBytes ([]byte) which contains the encoded content AST.
// Takes excerptBytes ([]byte) which contains the encoded excerpt AST.
//
// Returns *ast_domain.TemplateAST which is the decoded content AST.
// Returns *ast_domain.TemplateAST which is the decoded excerpt AST.
// Returns error when decoding either AST fails.
func decodeASTPair(ctx context.Context, decoder ASTDecoderPort, contentBytes, excerptBytes []byte) (contentAST *ast_domain.TemplateAST, excerptAST *ast_domain.TemplateAST, err error) {
	if len(contentBytes) > 0 {
		contentAST, err = decoder.DecodeForRender(ctx, contentBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode content AST: %w", err)
		}
	}

	if len(excerptBytes) > 0 {
		excerptAST, err = decoder.DecodeForRender(ctx, excerptBytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to decode excerpt AST: %w", err)
		}
	}

	return contentAST, excerptAST, nil
}

// getCollectionData retrieves the pre-decoded data for a named collection.
//
// Takes collectionName (string) which identifies the collection to retrieve.
//
// Returns *staticCollectionData which contains the blob and pre-decoded items.
// Returns error when the collection name is not found in the registry.
//
// Safe for concurrent use by multiple goroutines.
func getCollectionData(collectionName string) (*staticCollectionData, error) {
	staticCollectionRegistry.mu.RLock()
	data, exists := staticCollectionRegistry.collections[collectionName]
	staticCollectionRegistry.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("static collection %q not found; available collections: %v",
			collectionName, listStaticCollectionNames())
	}

	return data, nil
}

// lookupItemWithEncoder performs a binary search lookup in the encoded
// collection data by slug.
//
// Takes encoder (CollectionEncoderPort) which decodes items from the blob.
// Takes blob ([]byte) which is the encoded collection data to search.
// Takes collectionName (string) which identifies the collection for logging.
// Takes slug (string) which is the item identifier to look up.
//
// Returns metadataJSON ([]byte) which is the raw JSON metadata for the
// matched item.
// Returns contentASTBytes ([]byte) which is the encoded content AST.
// Returns excerptASTBytes ([]byte) which is the encoded excerpt AST.
// Returns err (error) when the item cannot be found.
func lookupItemWithEncoder(ctx context.Context, encoder CollectionEncoderPort, blob []byte, collectionName, slug string) (metadataJSON, contentASTBytes, excerptASTBytes []byte, err error) {
	_, l := logger_domain.From(ctx, log)
	l.Trace("lookupItemWithEncoder: looking up item",
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String(logKeySlug, slug))

	metadataJSON, contentASTBytes, excerptASTBytes, err = encoder.DecodeCollectionItem(blob, slug)
	if err != nil {
		l.Trace("lookupItemWithEncoder: item not found",
			logger_domain.String(logKeyCollection, collectionName),
			logger_domain.String(logKeySlug, slug),
			logger_domain.Error(err))
		return nil, nil, nil, fmt.Errorf("finding item %q in collection %q: %w", slug, collectionName, err)
	}

	return metadataJSON, contentASTBytes, excerptASTBytes, nil
}

// getOrBuildNavigation returns a lazily initialised navigation tree for the
// given data and config. The tree is built once and reused for all subsequent
// calls with the same config.
//
// Takes ctx (context.Context) which carries request-scoped values.
// Takes data (*staticCollectionData) which holds the pre-decoded items.
// Takes config (collection_dto.NavigationConfig) which controls tree building.
//
// Returns *collection_dto.NavigationGroups which contains the navigation
// trees.
func getOrBuildNavigation(
	ctx context.Context,
	data *staticCollectionData,
	config collection_dto.NavigationConfig,
) *collection_dto.NavigationGroups {
	key := navigationConfigKey{
		locale:         config.Locale,
		defaultOrder:   config.DefaultOrder,
		includeHidden:  config.IncludeHidden,
		groupBySection: config.GroupBySection,
	}

	if value, ok := data.navigation.Load(key); ok {
		if groups, valid := value.(*collection_dto.NavigationGroups); valid {
			return groups
		}
	}

	contentItems := metadataToContentItems(data.items)
	builder := NewNavigationBuilder()
	groups := builder.BuildNavigationGroups(ctx, contentItems, config)

	data.navigation.Store(key, groups)
	return groups
}

// metadataToContentItems converts pre-decoded metadata maps into ContentItem
// structs suitable for the NavigationBuilder.
//
// Takes items ([]map[string]any) which contains the pre-decoded metadata.
//
// Returns []collection_dto.ContentItem which holds the content items.
func metadataToContentItems(items []map[string]any) []collection_dto.ContentItem {
	contentItems := make([]collection_dto.ContentItem, 0, len(items))
	for _, metadata := range items {
		item := collection_dto.ContentItem{
			Metadata: metadata,
		}
		if id, ok := metadata["ID"].(string); ok {
			item.ID = id
		}
		if slug, ok := metadata["Slug"].(string); ok {
			item.Slug = slug
		}
		if locale, ok := metadata["Locale"].(string); ok {
			item.Locale = locale
		}
		if url, ok := metadata[metadataKeyURL].(string); ok {
			item.URL = url
		}
		contentItems = append(contentItems, item)
	}
	return contentItems
}

// listStaticCollectionNames returns all registered static collection names.
//
// Returns []string which contains the names of all static collections.
//
// The caller must hold the read lock or call from within a locked section.
func listStaticCollectionNames() []string {
	names := make([]string, 0, len(staticCollectionRegistry.collections))
	for name := range staticCollectionRegistry.collections {
		names = append(names, name)
	}
	return names
}
