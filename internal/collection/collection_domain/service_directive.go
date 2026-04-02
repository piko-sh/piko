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
	"maps"
	"time"

	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// expandStaticCollection creates virtual entry points for each content item.
//
// Takes provider (CollectionProvider) which supplies the static content.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which specifies
// the collection name, base path, and layout settings.
//
// Returns []*collection_dto.CollectionEntryPoint which contains one entry
// point per content item, each with initial props and routing details.
// Returns error when fetching static content fails.
func (s *collectionService) expandStaticCollection(
	ctx context.Context,
	provider CollectionProvider,
	directive *collection_dto.CollectionDirectiveInfo,
) ([]*collection_dto.CollectionEntryPoint, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Expanding static collection",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, directive.CollectionName))

	if err := s.configureContentSource(ctx, provider, directive); err != nil {
		return nil, fmt.Errorf("configuring content source for collection %q: %w", directive.CollectionName, err)
	}

	l.Trace("Calling provider FetchStaticContent",
		logger_domain.String("collection", directive.CollectionName))
	items, err := provider.FetchStaticContent(ctx, directive.CollectionName)
	if err != nil {
		return nil, fmt.Errorf("fetching static content: %w", err)
	}

	l.Internal("Fetched static content",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, directive.CollectionName),
		logger_domain.Int(logKeyItemCount, len(items)))

	entryPoints := make([]*collection_dto.CollectionEntryPoint, len(items))
	for i := range items {
		entryPoints[i] = &collection_dto.CollectionEntryPoint{
			InitialProps: map[string]any{
				"page":       convertItemMetadata(&items[i]),
				"contentAST": items[i].ContentAST,
				"excerptAST": items[i].ExcerptAST,
				"rawContent": items[i].RawContent,
			},
			Path:                 directive.LayoutPath,
			RoutePatternOverride: items[i].URL,
			DynamicCollection:    directive.CollectionName,
			DynamicProvider:      provider.Name(),
			Locale:               items[i].Locale,
			TranslationKey:       items[i].TranslationKey,
			IsPage:               true,
			IsVirtual:            true,
			IsDynamic:            false,
			IsHybrid:             false,
		}

		l.Trace("Created virtual entry point",
			logger_domain.String("url", items[i].URL),
			logger_domain.String("locale", items[i].Locale),
			logger_domain.String("translation_key", items[i].TranslationKey))
	}

	return entryPoints, nil
}

// expandDynamicCollection creates a single dynamic route handler.
//
// Takes provider (CollectionProvider) which supplies the dynamic content.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which defines the
// route settings.
//
// Returns []*collection_dto.CollectionEntryPoint which contains a single
// virtual entry point for a dynamic route.
// Returns error when the collection cannot be expanded.
func (*collectionService) expandDynamicCollection(
	ctx context.Context,
	provider CollectionProvider,
	directive *collection_dto.CollectionDirectiveInfo,
) ([]*collection_dto.CollectionEntryPoint, error) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Creating dynamic collection route",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, directive.CollectionName),
		logger_domain.String("route", directive.RoutePath))

	entryPoint := &collection_dto.CollectionEntryPoint{
		InitialProps:         nil,
		Path:                 directive.LayoutPath,
		RoutePatternOverride: directive.RoutePath,
		DynamicCollection:    directive.CollectionName,
		DynamicProvider:      provider.Name(),
		Locale:               "",
		TranslationKey:       "",
		IsPage:               true,
		IsVirtual:            true,
		IsDynamic:            true,
		IsHybrid:             false,
	}

	return []*collection_dto.CollectionEntryPoint{entryPoint}, nil
}

// expandHybridCollection creates both static snapshots and dynamic
// revalidation.
//
// Hybrid mode (ISR) provides zero-latency initial responses while keeping
// content fresh. Falls back to static collection expansion when hybrid
// content cannot be fetched.
//
// Takes provider (CollectionProvider) which supplies the collection data.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which specifies
// the collection configuration.
//
// Returns []*collection_dto.CollectionEntryPoint which contains the generated
// entry points for the hybrid collection.
// Returns error when content preparation fails and static fallback also fails.
func (s *collectionService) expandHybridCollection(
	ctx context.Context,
	provider CollectionProvider,
	directive *collection_dto.CollectionDirectiveInfo,
) ([]*collection_dto.CollectionEntryPoint, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Expanding hybrid collection (ISR)",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, directive.CollectionName))

	if err := s.configureContentSource(ctx, provider, directive); err != nil {
		return nil, fmt.Errorf("configuring content source for hybrid collection %q: %w", directive.CollectionName, err)
	}

	items, etag, blob, err := s.fetchAndPrepareHybridContent(ctx, provider, directive)
	if err != nil {
		return s.expandStaticCollection(ctx, provider, directive)
	}

	s.registerHybridDirectiveSnapshot(ctx, provider.Name(), directive.CollectionName, blob, etag, directive)
	return s.createHybridEntryPoints(ctx, items, directive, provider.Name())
}

// configureContentSource sets the content source for a collection provider.
//
// If ContentModulePath is set, the provider reads from an external Go module.
// This takes priority over BasePath. Otherwise, if BasePath is set, the
// provider uses the local file path.
//
// Takes provider (CollectionProvider) which is the provider to set up.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which holds the
// content source settings.
//
// Returns error when the provider does not support module content sourcing or
// when module setup fails.
func (*collectionService) configureContentSource(
	ctx context.Context,
	provider CollectionProvider,
	directive *collection_dto.CollectionDirectiveInfo,
) error {
	ctx, l := logger_domain.From(ctx, log)
	if directive.ContentModulePath != "" {
		configurable, ok := provider.(ContentModuleConfigurable)
		if !ok {
			return fmt.Errorf(
				"provider %q does not support module content sourcing (p-collection-source)",
				provider.Name(),
			)
		}

		if err := configurable.SetContentModulePath(ctx, directive.ContentModulePath); err != nil {
			return fmt.Errorf("configuring module content source: %w", err)
		}

		l.Internal("Configured provider to read from external module",
			logger_domain.String(logKeyProvider, provider.Name()),
			logger_domain.String("module_path", directive.ContentModulePath))
		return nil
	}

	if configurable, ok := provider.(BasePathConfigurable); ok && directive.BasePath != "" {
		configurable.SetBasePath(ctx, directive.BasePath)
		l.Trace("Configured provider base path",
			logger_domain.String("base_path", directive.BasePath))
	}

	return nil
}

// configureProviderBasePath sets the base path if the provider supports it.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes provider (CollectionProvider) which is checked for BasePathConfigurable
// support.
// Takes basePath (string) which specifies the path to set on the provider.
func (*collectionService) configureProviderBasePath(ctx context.Context, provider CollectionProvider, basePath string) {
	_, l := logger_domain.From(ctx, log)
	if configurable, ok := provider.(BasePathConfigurable); ok && basePath != "" {
		configurable.SetBasePath(ctx, basePath)
		l.Trace("Configured provider base path", logger_domain.String("base_path", basePath))
	}
}

// fetchAndPrepareHybridContent fetches content, computes ETag, and encodes
// to FlatBuffer.
//
// Takes provider (CollectionProvider) which supplies the static content.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which specifies
// the collection to fetch.
//
// Returns []collection_dto.ContentItem which contains the fetched items.
// Returns string which is the computed ETag for cache validation.
// Returns []byte which is the FlatBuffer encoded content.
// Returns error when fetching, ETag computation, or encoding fails.
func (s *collectionService) fetchAndPrepareHybridContent(
	ctx context.Context,
	provider CollectionProvider,
	directive *collection_dto.CollectionDirectiveInfo,
) ([]collection_dto.ContentItem, string, []byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	items, err := provider.FetchStaticContent(ctx, directive.CollectionName)
	if err != nil {
		return nil, "", nil, fmt.Errorf("fetching static content for hybrid: %w", err)
	}

	l.Internal("Fetched static content for hybrid snapshot",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, directive.CollectionName),
		logger_domain.Int(logKeyItemCount, len(items)))

	etag, err := provider.ComputeETag(ctx, directive.CollectionName)
	if err != nil {
		l.Warn("Failed to compute ETag, falling back to static mode",
			logger_domain.String(logKeyProvider, provider.Name()),
			logger_domain.String(logKeyCollection, directive.CollectionName),
			logger_domain.Error(err))
		return nil, "", nil, fmt.Errorf("computing ETag for collection %q: %w", directive.CollectionName, err)
	}

	l.Trace("Computed ETag for hybrid collection",
		logger_domain.String(logKeyCollection, directive.CollectionName),
		logger_domain.String("etag", etag))

	blob, err := s.encoder.EncodeCollection(items)
	if err != nil {
		l.Warn("Failed to encode hybrid snapshot, falling back to static mode",
			logger_domain.String(logKeyProvider, provider.Name()),
			logger_domain.String(logKeyCollection, directive.CollectionName),
			logger_domain.Error(err))
		return nil, "", nil, fmt.Errorf("encoding hybrid snapshot for collection %q: %w", directive.CollectionName, err)
	}

	return items, etag, blob, nil
}

// registerHybridDirectiveSnapshot saves a snapshot to the hybrid registry.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which identifies the collection.
// Takes blob ([]byte) which contains the snapshot data.
// Takes etag (string) which provides the version tag for the snapshot.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which specifies
// the hybrid settings.
func (s *collectionService) registerHybridDirectiveSnapshot(
	ctx context.Context,
	providerName, collectionName string,
	blob []byte,
	etag string,
	directive *collection_dto.CollectionDirectiveInfo,
) {
	_, l := logger_domain.From(ctx, log)
	hybridConfig := buildHybridConfigFromDirective(directive)
	s.hybridRegistry.Register(ctx, providerName, collectionName, blob, etag, hybridConfig)

	l.Internal("Registered hybrid snapshot",
		logger_domain.String(logKeyProvider, providerName),
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String("etag", etag),
		logger_domain.Int("blob_size", len(blob)))
}

// createHybridEntryPoints creates entry points with IsHybrid=true.
//
// Takes items ([]collection_dto.ContentItem) which provides the content items
// to convert into entry points.
// Takes directive (*collection_dto.CollectionDirectiveInfo) which specifies
// the collection configuration including layout path.
// Takes providerName (string) which identifies the dynamic provider.
//
// Returns []*collection_dto.CollectionEntryPoint which contains the created
// hybrid entry points ready for rendering.
// Returns error when entry point creation fails.
func (*collectionService) createHybridEntryPoints(
	ctx context.Context,
	items []collection_dto.ContentItem,
	directive *collection_dto.CollectionDirectiveInfo,
	providerName string,
) ([]*collection_dto.CollectionEntryPoint, error) {
	_, l := logger_domain.From(ctx, log)
	entryPoints := make([]*collection_dto.CollectionEntryPoint, len(items))
	for i := range items {
		entryPoints[i] = &collection_dto.CollectionEntryPoint{
			InitialProps: map[string]any{
				"page":       convertItemMetadata(&items[i]),
				"contentAST": items[i].ContentAST,
				"excerptAST": items[i].ExcerptAST,
				"rawContent": items[i].RawContent,
			},
			Path:                 directive.LayoutPath,
			RoutePatternOverride: items[i].URL,
			DynamicCollection:    directive.CollectionName,
			DynamicProvider:      providerName,
			Locale:               items[i].Locale,
			TranslationKey:       items[i].TranslationKey,
			IsPage:               true,
			IsVirtual:            true,
			IsDynamic:            false,
			IsHybrid:             true,
		}

		l.Trace("Created hybrid entry point",
			logger_domain.String("url", items[i].URL),
			logger_domain.String("locale", items[i].Locale))
	}

	return entryPoints, nil
}

// buildHybridConfigFromDirective builds a HybridConfig from directive settings.
//
// Takes directive (*collection_dto.CollectionDirectiveInfo) which contains the
// cache settings to apply.
//
// Returns collection_dto.HybridConfig which has default values replaced by any
// settings from the directive.
func buildHybridConfigFromDirective(directive *collection_dto.CollectionDirectiveInfo) collection_dto.HybridConfig {
	config := collection_dto.DefaultHybridConfig()

	if directive.CacheConfig != nil {
		if directive.CacheConfig.TTL > 0 {
			config.RevalidationTTL = time.Duration(directive.CacheConfig.TTL) * time.Second
		}
		if directive.CacheConfig.Strategy == "no-cache" {
			config.RevalidationTTL = 0
		}
	}

	return config
}

// convertItemMetadata converts a ContentItem into a metadata map for use in
// templates.
//
// Takes item (*collection_dto.ContentItem) which is the content item to
// convert.
//
// Returns map[string]any which contains the item metadata with standard keys
// for template use.
func convertItemMetadata(item *collection_dto.ContentItem) map[string]any {
	metadata := make(map[string]any, len(item.Metadata)+9)
	maps.Copy(metadata, item.Metadata)

	metadata[collection_dto.MetaKeyID] = item.ID
	metadata[collection_dto.MetaKeySlug] = item.Slug
	metadata[collection_dto.MetaKeyLocale] = item.Locale
	metadata[collection_dto.MetaKeyTranslationKey] = item.TranslationKey
	metadata[collection_dto.MetaKeyURL] = item.URL
	metadata[collection_dto.MetaKeyReadingTime] = item.ReadingTime
	metadata[collection_dto.MetaKeyCreatedAt] = item.CreatedAt
	metadata[collection_dto.MetaKeyUpdatedAt] = item.UpdatedAt
	metadata[collection_dto.MetaKeyPublishedAt] = item.PublishedAt

	return metadata
}
