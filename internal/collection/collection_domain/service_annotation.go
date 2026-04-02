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

// generateStaticCollectionAnnotation creates an annotation for static
// collections.
//
// Takes provider (CollectionProvider) which supplies the collection data.
// Takes collectionName (string) which identifies the collection to fetch.
// Takes targetTypeExpr (ast.Expr) which specifies the target slice type.
// Takes options (*collection_dto.FetchOptions) which controls filtering,
// sorting, and pagination.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the generated
// slice literal for the static collection.
// Returns error when fetching content fails or building the slice literal
// fails.
func (s *collectionService) generateStaticCollectionAnnotation(
	ctx context.Context,
	provider CollectionProvider,
	collectionName string,
	targetTypeExpr ast.Expr,
	options *collection_dto.FetchOptions,
) (*ast_domain.GoGeneratorAnnotation, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Generating static collection annotation",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, collectionName))

	items, err := s.fetchOrCacheStaticContent(ctx, provider, collectionName)
	if err != nil {
		return nil, fmt.Errorf("fetching static content for collection %q: %w", collectionName, err)
	}

	processedItems := s.applyQueryOptions(items, options)
	l.Internal("Applied query options",
		logger_domain.Int("original_count", len(items)),
		logger_domain.Int("filtered_count", len(processedItems)))

	sliceLiteral, err := s.buildSliceLiteral(ctx, targetTypeExpr, processedItems)
	if err != nil {
		return nil, fmt.Errorf("building slice literal: %w", err)
	}

	annotation := s.buildStaticAnnotation(targetTypeExpr, sliceLiteral, processedItems)
	s.logStaticAnnotationDiagnostics(ctx, annotation, processedItems)

	return annotation, nil
}

// fetchOrCacheStaticContent fetches static content from provider with caching.
//
// Takes provider (CollectionProvider) which supplies the static content.
// Takes collectionName (string) which identifies the collection to fetch.
//
// Returns []collection_dto.ContentItem which contains the cached or freshly
// fetched content items.
// Returns error when the provider fails to fetch the content.
func (s *collectionService) fetchOrCacheStaticContent(
	ctx context.Context,
	provider CollectionProvider,
	collectionName string,
) ([]collection_dto.ContentItem, error) {
	ctx, l := logger_domain.From(ctx, log)
	if cachedItems, ok := s.getCachedContent(provider.Name(), collectionName); ok {
		l.Trace("Using cached static content",
			logger_domain.String(logKeyProvider, provider.Name()),
			logger_domain.String(logKeyCollection, collectionName),
			logger_domain.Int(logKeyItemCount, len(cachedItems)))
		return cachedItems, nil
	}

	items, err := provider.FetchStaticContent(ctx, collectionName)
	if err != nil {
		return nil, fmt.Errorf("fetching static content: %w", err)
	}

	s.setCachedContent(provider.Name(), collectionName, items)

	l.Internal("Fetched and cached static content",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.Int(logKeyItemCount, len(items)))

	return items, nil
}

// buildStaticAnnotation creates a GoGeneratorAnnotation struct for static
// collections.
//
// Takes targetTypeExpr (ast.Expr) which specifies the type of elements in the
// collection.
// Takes sliceLiteral (ast.Expr) which provides the slice literal expression.
// Takes processedItems ([]collection_dto.ContentItem) which contains the
// pre-processed collection items.
//
// Returns *ast_domain.GoGeneratorAnnotation which represents the annotation
// configured for static collection generation.
func (s *collectionService) buildStaticAnnotation(
	targetTypeExpr ast.Expr,
	sliceLiteral ast.Expr,
	processedItems []collection_dto.ContentItem,
) *ast_domain.GoGeneratorAnnotation {
	resolvedType := s.createSliceTypeInfo(targetTypeExpr)

	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
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
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// logStaticAnnotationDiagnostics logs debug information about a static
// collection annotation.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes annotation (*ast_domain.GoGeneratorAnnotation) which is the annotation
// to log details for.
// Takes processedItems ([]collection_dto.ContentItem) which are the items that
// were processed from the annotation.
func (*collectionService) logStaticAnnotationDiagnostics(
	ctx context.Context,
	annotation *ast_domain.GoGeneratorAnnotation,
	processedItems []collection_dto.ContentItem,
) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Collection annotation created successfully",
		logger_domain.Int(logKeyItemCount, len(processedItems)),
		logger_domain.Bool("has_literal", annotation.StaticCollectionLiteral != nil))

	l.Trace("Collection annotation diagnostics",
		logger_domain.Bool("is_collection_call", annotation.IsCollectionCall),
		logger_domain.Bool("has_literal", annotation.StaticCollectionLiteral != nil),
		logger_domain.Bool("has_data", len(annotation.StaticCollectionData) > 0),
		logger_domain.Bool("is_static", annotation.IsStatic),
		logger_domain.Bool("is_structurally_static", annotation.IsStructurallyStatic),
		logger_domain.String("resolved_type", fmt.Sprintf("%T", annotation.ResolvedType)))

	if annotation.StaticCollectionLiteral != nil {
		astString := fmt.Sprintf("%#v", annotation.StaticCollectionLiteral)
		if len(astString) > maxASTPreviewLength {
			astString = astString[:maxASTPreviewLength] + "..."
		}
		l.Trace("Collection AST preview", logger_domain.String("ast", astString))
	}
}

// createSliceTypeInfo creates type information for a slice type.
//
// Takes targetTypeExpr (ast.Expr) which specifies the element type for the
// slice.
//
// Returns *ast_domain.ResolvedTypeInfo which contains the constructed slice
// type expression with empty package information.
func (*collectionService) createSliceTypeInfo(targetTypeExpr ast.Expr) *ast_domain.ResolvedTypeInfo {
	sliceTypeExpr := &ast.ArrayType{
		Lbrack: 0,
		Len:    nil,
		Elt:    targetTypeExpr,
	}

	return &ast_domain.ResolvedTypeInfo{
		TypeExpression:       sliceTypeExpr,
		PackageAlias:         "",
		CanonicalPackagePath: "",
	}
}

// generateDynamicAnnotation creates an annotation for runtime data fetching.
//
// For dynamic providers, this generates a blueprint that the Generator uses
// to emit a runtime fetcher function and its call.
//
// Takes provider (CollectionProvider) which supplies the data source.
// Takes collectionName (string) which identifies the collection to fetch.
// Takes targetType (ast.Expr) which specifies the expected result type.
// Takes options (*collection_dto.FetchOptions) which controls fetch behaviour.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the fetcher
// blueprint for code generation.
// Returns error when the provider fails to generate the runtime fetcher.
func (s *collectionService) generateDynamicAnnotation(
	ctx context.Context,
	provider CollectionProvider,
	collectionName string,
	targetType ast.Expr,
	options *collection_dto.FetchOptions,
) (*ast_domain.GoGeneratorAnnotation, error) {
	ctx, l := logger_domain.From(ctx, log)
	l.Internal("Generating dynamic collection annotation",
		logger_domain.String(logKeyProvider, provider.Name()),
		logger_domain.String(logKeyCollection, collectionName))

	fetcherCode, err := provider.GenerateRuntimeFetcher(ctx, collectionName, targetType, *options)
	if err != nil {
		return nil, fmt.Errorf("generating runtime fetcher: %w", err)
	}

	dynamicInfo := buildDynamicCollectionInfo(fetcherCode, targetType, provider.Name(), collectionName)

	annotation := s.buildDynamicAnnotation(targetType, dynamicInfo)
	s.logDynamicAnnotationDiagnostics(ctx, annotation, dynamicInfo, provider.Name(), collectionName, fetcherCode)

	return annotation, nil
}

// buildDynamicAnnotation creates a GoGeneratorAnnotation struct for dynamic
// collections.
//
// Takes targetType (ast.Expr) which specifies the target type expression.
// Takes dynamicInfo (*collection_dto.DynamicCollectionInfo) which provides
// the dynamic collection metadata.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the annotation
// configured for dynamic collection use.
func (s *collectionService) buildDynamicAnnotation(
	targetType ast.Expr,
	dynamicInfo *collection_dto.DynamicCollectionInfo,
) *ast_domain.GoGeneratorAnnotation {
	resolvedType := s.createSliceTypeInfo(targetType)

	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   dynamicInfo,
		StaticCollectionLiteral: nil,
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
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           0,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        true,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// logDynamicAnnotationDiagnostics logs debug details about a dynamic
// collection annotation.
//
// Takes ctx (context.Context) which carries deadlines, cancellation signals,
// and request-scoped values.
// Takes annotation (*ast_domain.GoGeneratorAnnotation) which is the annotation
// to log.
// Takes dynamicInfo (*collection_dto.DynamicCollectionInfo) which holds the
// dynamic collection data.
// Takes providerName (string) which names the collection provider.
// Takes collectionName (string) which names the collection.
// Takes fetcherCode (*collection_dto.RuntimeFetcherCode) which holds the
// runtime fetcher code, or nil if not present.
func (*collectionService) logDynamicAnnotationDiagnostics(
	ctx context.Context,
	annotation *ast_domain.GoGeneratorAnnotation,
	dynamicInfo *collection_dto.DynamicCollectionInfo,
	providerName string,
	collectionName string,
	fetcherCode *collection_dto.RuntimeFetcherCode,
) {
	_, l := logger_domain.From(ctx, log)
	l.Internal("Dynamic collection annotation created successfully",
		logger_domain.String(logKeyProvider, providerName),
		logger_domain.String(logKeyCollection, collectionName),
		logger_domain.Bool("has_fetcher_code", fetcherCode != nil))

	l.Trace("Dynamic collection annotation diagnostics",
		logger_domain.Bool("is_collection_call", annotation.IsCollectionCall),
		logger_domain.Bool("has_fetcher_code", dynamicInfo.FetcherCode != nil),
		logger_domain.Bool("has_fetcher_func", dynamicInfo.FetcherCode.FetcherFunc != nil),
		logger_domain.Bool("is_static", annotation.IsStatic),
		logger_domain.String("provider", dynamicInfo.ProviderName))
}

// convertItemsToAny converts a ContentItem slice to []any for annotation storage.
//
// Takes items ([]collection_dto.ContentItem) which is the slice to convert.
//
// Returns []any which holds the same items as interface values.
func convertItemsToAny(items []collection_dto.ContentItem) []any {
	result := make([]any, len(items))
	for i := range items {
		result[i] = items[i]
	}
	return result
}

// buildDynamicCollectionInfo creates a DynamicCollectionInfo struct.
//
// Takes fetcherCode (*collection_dto.RuntimeFetcherCode) which provides the
// runtime fetcher code.
// Takes targetType (ast.Expr) which specifies the target type expression.
// Takes providerName (string) which identifies the data provider.
// Takes collectionName (string) which names the collection.
//
// Returns *collection_dto.DynamicCollectionInfo which is set up with hybrid
// mode off and optional fields set to nil.
func buildDynamicCollectionInfo(
	fetcherCode *collection_dto.RuntimeFetcherCode,
	targetType ast.Expr,
	providerName string,
	collectionName string,
) *collection_dto.DynamicCollectionInfo {
	return &collection_dto.DynamicCollectionInfo{
		FetcherCode:     fetcherCode,
		TargetType:      targetType,
		ProviderName:    providerName,
		CollectionName:  collectionName,
		HybridMode:      false,
		HybridConfig:    nil,
		SnapshotETag:    "",
		RevalidatorCode: nil,
	}
}
