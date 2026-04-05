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
	"go/ast"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/collection/collection_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// generateHybridAnnotation creates annotation for hybrid (ISR) collections.
//
// Hybrid mode combines static generation with runtime revalidation:
//   - StaticCollectionLiteral: For immediate rendering (like static mode)
//   - DynamicCollectionInfo: With HybridMode=true for revalidation handling
//
// This enables zero-latency initial responses while keeping content fresh.
//
// Takes provider (CollectionProvider) which supplies the collection data.
// Takes collectionName (string) which identifies the collection to generate.
// Takes targetTypeExpr (ast.Expr) which specifies the target slice element type.
// Takes options (*collection_dto.FetchOptions) which controls fetch behaviour.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the hybrid metadata.
// Returns error when building the slice literal fails.
func (s *collectionService) generateHybridAnnotation(
	ctx context.Context,
	provider CollectionProvider,
	collectionName string,
	targetTypeExpr ast.Expr,
	options *collection_dto.FetchOptions,
) (*ast_domain.GoGeneratorAnnotation, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Generating hybrid collection annotation",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, collectionName))

	source := s.defaultContentSource()
	processedItems, etag, err := s.prepareHybridContent(ctx, provider, collectionName, options, source)
	if err != nil {
		return s.generateStaticCollectionAnnotation(ctx, provider, collectionName, targetTypeExpr, options)
	}

	if err := s.registerHybridSnapshot(ctx, provider, collectionName, processedItems, etag); err != nil {
		l.Warn("Failed to register hybrid snapshot, falling back to static mode",
			logger_domain.String(logKeyProvider, provider.Name()),
			logger_domain.String(logKeyCollection, collectionName),
			logger_domain.Error(err))
		return s.generateStaticCollectionAnnotation(ctx, provider, collectionName, targetTypeExpr, options)
	}

	sliceLiteral, err := s.buildSliceLiteral(ctx, targetTypeExpr, processedItems)
	if err != nil {
		return nil, fmt.Errorf("building slice literal for hybrid: %w", err)
	}

	return s.finaliseHybridAnnotation(ctx, hybridAnnotationParams{
		ProviderName:   provider.Name(),
		CollectionName: collectionName,
		TargetTypeExpr: targetTypeExpr,
		Etag:           etag,
		SliceLiteral:   sliceLiteral,
		ProcessedItems: processedItems,
		Options:        options,
	})
}

// prepareHybridContent fetches, filters, and computes ETag for hybrid content.
//
// Takes provider (CollectionProvider) which supplies the content source.
// Takes collectionName (string) which identifies the collection to fetch.
// Takes options (*collection_dto.FetchOptions) which specifies query filters.
//
// Returns []collection_dto.ContentItem which contains the filtered content.
// Returns string which is the computed ETag for cache validation.
// Returns error when content fetching or ETag computation fails.
func (s *collectionService) prepareHybridContent(
	ctx context.Context,
	provider CollectionProvider,
	collectionName string,
	options *collection_dto.FetchOptions,
	source collection_dto.ContentSource,
) ([]collection_dto.ContentItem, string, error) {
	ctx, l := logger_domain.From(ctx, log)
	items, err := s.fetchOrCacheStaticContent(ctx, provider, collectionName, source)
	if err != nil {
		return nil, "", fmt.Errorf("fetching static content for hybrid collection %q: %w", collectionName, err)
	}

	processedItems := s.applyQueryOptions(items, options)
	l.Trace("Applied query options for hybrid collection",
		logger_domain.Int("original_count", len(items)),
		logger_domain.Int("filtered_count", len(processedItems)))

	etag, err := provider.ComputeETag(ctx, collectionName, source)
	if err != nil {
		l.Warn("Failed to compute ETag, falling back to static mode",
			logger_domain.String(logKeyProvider, provider.Name()),
			logger_domain.String(logKeyCollection, collectionName),
			logger_domain.Error(err))
		return nil, "", fmt.Errorf("computing ETag for hybrid collection %q: %w", collectionName, err)
	}

	l.Trace("Computed ETag for hybrid collection",
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String("etag", etag))

	return processedItems, etag, nil
}

// hybridAnnotationParams bundles the data parameters for finaliseHybridAnnotation.
type hybridAnnotationParams struct {
	// TargetTypeExpr specifies the target slice element type expression.
	TargetTypeExpr ast.Expr

	// SliceLiteral holds the static slice literal expression for immediate rendering.
	SliceLiteral ast.Expr

	// Options controls fetch behaviour such as caching and filtering.
	Options *collection_dto.FetchOptions

	// ProviderName identifies the data provider that supplies the collection.
	ProviderName string

	// CollectionName identifies the collection within the provider.
	CollectionName string

	// Etag holds the computed ETag for cache validation of the snapshot.
	Etag string

	// ProcessedItems contains the filtered and processed content items.
	ProcessedItems []collection_dto.ContentItem
}

// finaliseHybridAnnotation builds the final annotation with hybrid metadata.
//
// Takes params (hybridAnnotationParams) which bundles the provider name,
// collection name, type expression, ETag, slice literal, processed items,
// and fetch options.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the complete hybrid
// annotation.
// Returns error when annotation building fails.
func (s *collectionService) finaliseHybridAnnotation(
	ctx context.Context,
	params hybridAnnotationParams,
) (*ast_domain.GoGeneratorAnnotation, error) {
	dynamicInfo := s.buildHybridDynamicInfo(params.ProviderName, params.CollectionName, params.TargetTypeExpr, params.Etag, new(s.buildHybridConfigFromOptions(params.Options)))
	annotation := s.buildHybridAnnotation(params.TargetTypeExpr, params.SliceLiteral, dynamicInfo, params.ProcessedItems)
	s.logHybridAnnotationDiagnostics(ctx, annotation, dynamicInfo, params.ProcessedItems)
	return annotation, nil
}

// registerHybridSnapshot encodes content and registers it with the hybrid
// registry.
//
// Takes provider (CollectionProvider) which supplies the collection source.
// Takes collectionName (string) which identifies the collection to register.
// Takes items ([]collection_dto.ContentItem) which contains the content to
// encode.
// Takes etag (string) which provides the version tag for the snapshot.
//
// Returns error when encoding fails.
func (s *collectionService) registerHybridSnapshot(
	ctx context.Context,
	provider CollectionProvider,
	collectionName string,
	items []collection_dto.ContentItem,
	etag string,
) error {
	_, l := logger_domain.From(ctx, log)
	blob, err := s.encoder.EncodeCollection(items)
	if err != nil {
		return fmt.Errorf("encoding hybrid content: %w", err)
	}

	config := collection_dto.DefaultHybridConfig()

	s.hybridRegistry.Register(
		ctx,
		provider.Name(),
		collectionName,
		blob,
		etag,
		config,
	)

	l.Trace("Registered hybrid snapshot",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.String("etag", etag),
		logger_domain.Int("blob_size", len(blob)))

	return nil
}

// buildHybridConfigFromOptions builds a HybridConfig from fetch options.
//
// Takes options (*collection_dto.FetchOptions) which specifies the cache and
// fetch settings to use.
//
// Returns collection_dto.HybridConfig which contains the default settings
// with any values from options applied.
func (*collectionService) buildHybridConfigFromOptions(options *collection_dto.FetchOptions) collection_dto.HybridConfig {
	config := collection_dto.DefaultHybridConfig()

	if options == nil {
		return config
	}

	if options.Cache != nil {
		ttl := options.Cache.GetTTLDuration()
		if ttl > 0 {
			config.RevalidationTTL = ttl
		}

		if options.Cache.Strategy == "stale-while-revalidate" {
			config.StaleIfError = true
		}
		if options.Cache.Strategy == "no-cache" {
			config.RevalidationTTL = 0
		}
	}

	return config
}

// buildHybridDynamicInfo creates DynamicCollectionInfo with hybrid mode fields.
//
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which specifies the collection to configure.
// Takes targetType (ast.Expr) which defines the type for the collection items.
// Takes etag (string) which provides the snapshot version identifier.
// Takes hybridConfig (*collection_dto.HybridConfig) which contains hybrid mode
// settings.
//
// Returns *collection_dto.DynamicCollectionInfo which is configured for hybrid
// mode with fetcher and revalidator set to nil.
func (*collectionService) buildHybridDynamicInfo(
	providerName string,
	collectionName string,
	targetType ast.Expr,
	etag string,
	hybridConfig *collection_dto.HybridConfig,
) *collection_dto.DynamicCollectionInfo {
	return &collection_dto.DynamicCollectionInfo{
		FetcherCode:     nil,
		TargetType:      targetType,
		ProviderName:    providerName,
		CollectionName:  collectionName,
		HybridMode:      true,
		HybridConfig:    hybridConfig,
		SnapshotETag:    etag,
		RevalidatorCode: nil,
	}
}

// buildHybridAnnotation creates a GoGeneratorAnnotation for hybrid collections.
//
// Unlike pure static or dynamic modes, hybrid annotations contain both:
//   - StaticCollectionLiteral: For immediate zero-latency rendering
//   - DynamicCollectionInfo: With hybrid mode fields for revalidation
//
// Takes targetTypeExpr (ast.Expr) which specifies the target slice type.
// Takes sliceLiteral (ast.Expr) which provides the static slice literal.
// Takes dynamicInfo (*collection_dto.DynamicCollectionInfo) which contains
// the dynamic collection configuration for revalidation.
// Takes processedItems ([]collection_dto.ContentItem) which holds the
// pre-processed content items for static rendering.
//
// Returns *ast_domain.GoGeneratorAnnotation which is configured for hybrid
// collection mode with both static and dynamic capabilities.
func (s *collectionService) buildHybridAnnotation(
	targetTypeExpr ast.Expr,
	sliceLiteral ast.Expr,
	dynamicInfo *collection_dto.DynamicCollectionInfo,
	processedItems []collection_dto.ContentItem,
) *ast_domain.GoGeneratorAnnotation {
	resolvedType := s.createSliceTypeInfo(targetTypeExpr)

	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   dynamicInfo,
		StaticCollectionLiteral: sliceLiteral,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            resolvedType,
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      nil,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    convertItemsToAny(processedItems),
		Srcset:                  nil,
		Stringability:           0,
		IsStatic:                true,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    true,
		IsPointerToStringable:   false,
		IsCollectionCall:        true,
		IsHybridCollection:      true,
		IsMapAccess:             false,
	}
}

// logHybridAnnotationDiagnostics logs debug details about a hybrid collection
// annotation.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes annotation (*ast_domain.GoGeneratorAnnotation) which holds the parsed
// annotation data to check.
// Takes dynamicInfo (*collection_dto.DynamicCollectionInfo) which provides
// the dynamic collection settings including hybrid mode options.
// Takes processedItems ([]collection_dto.ContentItem) which holds the items
// that were processed for this collection.
func (*collectionService) logHybridAnnotationDiagnostics(
	ctx context.Context,
	annotation *ast_domain.GoGeneratorAnnotation,
	dynamicInfo *collection_dto.DynamicCollectionInfo,
	processedItems []collection_dto.ContentItem,
) {
	_, l := logger_domain.From(ctx, log)

	if dynamicInfo == nil {
		l.Warn("logHybridAnnotationDiagnostics called with nil dynamicInfo")
		return
	}

	l.Internal("Hybrid collection annotation created successfully",
		logger_domain.String(logKeyProvider, dynamicInfo.ProviderName),
		logger_domain.String(logKeyCollection, dynamicInfo.CollectionName),
		logger_domain.Int(logKeyItemCount, len(processedItems)),
		logger_domain.String("etag", dynamicInfo.SnapshotETag))

	l.Trace("Hybrid collection annotation diagnostics",
		logger_domain.Bool("is_collection_call", annotation.IsCollectionCall),
		logger_domain.Bool("is_hybrid_collection", annotation.IsHybridCollection),
		logger_domain.Bool("has_static_literal", annotation.StaticCollectionLiteral != nil),
		logger_domain.Bool("has_dynamic_info", annotation.DynamicCollectionInfo != nil),
		logger_domain.Bool("is_static", annotation.IsStatic),
		logger_domain.Bool("is_structurally_static", annotation.IsStructurallyStatic),
		logger_domain.String("resolved_type", fmt.Sprintf("%T", annotation.ResolvedType)))

	l.Trace("Hybrid mode fields",
		logger_domain.Bool("hybrid_mode", dynamicInfo.HybridMode),
		logger_domain.String("snapshot_etag", dynamicInfo.SnapshotETag),
		logger_domain.Bool("has_hybrid_config", dynamicInfo.HybridConfig != nil))

	if dynamicInfo.HybridConfig != nil {
		l.Trace("Hybrid config details",
			logger_domain.Duration("revalidation_ttl", dynamicInfo.HybridConfig.RevalidationTTL),
			logger_domain.Bool("stale_if_error", dynamicInfo.HybridConfig.StaleIfError),
			logger_domain.String("etag_source", dynamicInfo.HybridConfig.ETagSource))
	}

	if annotation.StaticCollectionLiteral != nil {
		astString := fmt.Sprintf("%#v", annotation.StaticCollectionLiteral)
		if len(astString) > maxASTPreviewLength {
			astString = astString[:maxASTPreviewLength] + "..."
		}
		l.Trace("Hybrid collection AST preview", logger_domain.String("ast", astString))
	}
}
