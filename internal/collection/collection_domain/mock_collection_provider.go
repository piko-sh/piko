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
	"sync/atomic"

	"piko.sh/piko/internal/collection/collection_dto"
)

// MockCollectionProvider is a test double for CollectionProvider that returns
// zero values from nil function fields and tracks call counts atomically.
type MockCollectionProvider struct {
	// NameFunc is the function called by Name.
	NameFunc func() string

	// TypeFunc is the function called by Type.
	TypeFunc func() ProviderType

	// DiscoverCollectionsFunc is the function called by
	// DiscoverCollections.
	DiscoverCollectionsFunc func(ctx context.Context, config collection_dto.ProviderConfig) ([]collection_dto.CollectionInfo, error)

	// ValidateTargetTypeFunc is the function called by
	// ValidateTargetType.
	ValidateTargetTypeFunc func(targetType ast.Expr) error

	// FetchStaticContentFunc is the function called by
	// FetchStaticContent.
	FetchStaticContentFunc func(ctx context.Context, collectionName string, source collection_dto.ContentSource) ([]collection_dto.ContentItem, error)

	// GenerateRuntimeFetcherFunc is the function called
	// by GenerateRuntimeFetcher.
	GenerateRuntimeFetcherFunc func(ctx context.Context, collectionName string, targetType ast.Expr, options collection_dto.FetchOptions) (*collection_dto.RuntimeFetcherCode, error)

	// ComputeETagFunc is the function called by
	// ComputeETag.
	ComputeETagFunc func(ctx context.Context, collectionName string, source collection_dto.ContentSource) (string, error)

	// ValidateETagFunc is the function called by
	// ValidateETag.
	ValidateETagFunc func(ctx context.Context, collectionName string, expectedETag string, source collection_dto.ContentSource) (currentETag string, changed bool, err error)

	// GenerateRevalidatorFunc is the function called by
	// GenerateRevalidator.
	GenerateRevalidatorFunc func(ctx context.Context, collectionName string, targetType ast.Expr, config collection_dto.HybridConfig) (*collection_dto.RuntimeFetcherCode, error)

	// NameCallCount tracks how many times Name was
	// called.
	NameCallCount int64

	// TypeCallCount tracks how many times Type was
	// called.
	TypeCallCount int64

	// DiscoverCollectionsCallCount tracks how many times
	// DiscoverCollections was called.
	DiscoverCollectionsCallCount int64

	// ValidateTargetTypeCallCount tracks how many times
	// ValidateTargetType was called.
	ValidateTargetTypeCallCount int64

	// FetchStaticContentCallCount tracks how many times
	// FetchStaticContent was called.
	FetchStaticContentCallCount int64

	// GenerateRuntimeFetcherCallCount tracks how many
	// times GenerateRuntimeFetcher was called.
	GenerateRuntimeFetcherCallCount int64

	// ComputeETagCallCount tracks how many times
	// ComputeETag was called.
	ComputeETagCallCount int64

	// ValidateETagCallCount tracks how many times
	// ValidateETag was called.
	ValidateETagCallCount int64

	// GenerateRevalidatorCallCount tracks how many times
	// GenerateRevalidator was called.
	GenerateRevalidatorCallCount int64
}

var _ CollectionProvider = (*MockCollectionProvider)(nil)

// Name delegates to NameFunc if set.
//
// Returns "" if NameFunc is nil.
func (m *MockCollectionProvider) Name() string {
	atomic.AddInt64(&m.NameCallCount, 1)
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return ""
}

// Type delegates to TypeFunc if set.
//
// Returns "" if TypeFunc is nil.
func (m *MockCollectionProvider) Type() ProviderType {
	atomic.AddInt64(&m.TypeCallCount, 1)
	if m.TypeFunc != nil {
		return m.TypeFunc()
	}
	return ""
}

// DiscoverCollections delegates to DiscoverCollectionsFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes config (collection_dto.ProviderConfig) which provides the
// provider configuration.
//
// Returns (nil, nil) if DiscoverCollectionsFunc is nil.
func (m *MockCollectionProvider) DiscoverCollections(ctx context.Context, config collection_dto.ProviderConfig) ([]collection_dto.CollectionInfo, error) {
	atomic.AddInt64(&m.DiscoverCollectionsCallCount, 1)
	if m.DiscoverCollectionsFunc != nil {
		return m.DiscoverCollectionsFunc(ctx, config)
	}
	return nil, nil
}

// ValidateTargetType delegates to ValidateTargetTypeFunc if set.
//
// Takes targetType (ast.Expr) which is the AST expression for the target type.
//
// Returns nil if ValidateTargetTypeFunc is nil.
func (m *MockCollectionProvider) ValidateTargetType(targetType ast.Expr) error {
	atomic.AddInt64(&m.ValidateTargetTypeCallCount, 1)
	if m.ValidateTargetTypeFunc != nil {
		return m.ValidateTargetTypeFunc(targetType)
	}
	return nil
}

// FetchStaticContent delegates to FetchStaticContentFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes collectionName (string) which identifies the collection by name.
//
// Returns (nil, nil) if FetchStaticContentFunc is nil.
func (m *MockCollectionProvider) FetchStaticContent(ctx context.Context, collectionName string, source collection_dto.ContentSource) ([]collection_dto.ContentItem, error) {
	atomic.AddInt64(&m.FetchStaticContentCallCount, 1)
	if m.FetchStaticContentFunc != nil {
		return m.FetchStaticContentFunc(ctx, collectionName, source)
	}
	return nil, nil
}

// GenerateRuntimeFetcher delegates to GenerateRuntimeFetcherFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes collectionName (string) which identifies the collection by name.
// Takes targetType (ast.Expr) which is the AST expression for the target type.
// Takes options (collection_dto.FetchOptions) which provides fetch
// configuration options.
//
// Returns (nil, nil) if GenerateRuntimeFetcherFunc is nil.
func (m *MockCollectionProvider) GenerateRuntimeFetcher(
	ctx context.Context, collectionName string,
	targetType ast.Expr, options collection_dto.FetchOptions,
) (*collection_dto.RuntimeFetcherCode, error) {
	atomic.AddInt64(&m.GenerateRuntimeFetcherCallCount, 1)
	if m.GenerateRuntimeFetcherFunc != nil {
		return m.GenerateRuntimeFetcherFunc(ctx, collectionName, targetType, options)
	}
	return nil, nil
}

// ComputeETag delegates to ComputeETagFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes collectionName (string) which identifies the collection by name.
//
// Returns ("", nil) if ComputeETagFunc is nil.
func (m *MockCollectionProvider) ComputeETag(ctx context.Context, collectionName string, source collection_dto.ContentSource) (string, error) {
	atomic.AddInt64(&m.ComputeETagCallCount, 1)
	if m.ComputeETagFunc != nil {
		return m.ComputeETagFunc(ctx, collectionName, source)
	}
	return "", nil
}

// ValidateETag delegates to ValidateETagFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes collectionName (string) which identifies the collection by name.
// Takes expectedETag (string) which is the ETag value to validate against.
//
// Returns ("", false, nil) if ValidateETagFunc is nil.
func (m *MockCollectionProvider) ValidateETag(ctx context.Context, collectionName string, expectedETag string, source collection_dto.ContentSource) (string, bool, error) {
	atomic.AddInt64(&m.ValidateETagCallCount, 1)
	if m.ValidateETagFunc != nil {
		return m.ValidateETagFunc(ctx, collectionName, expectedETag, source)
	}
	return "", false, nil
}

// GenerateRevalidator delegates to GenerateRevalidatorFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes collectionName (string) which identifies the collection by name.
// Takes targetType (ast.Expr) which is the AST expression for the target type.
// Takes config (collection_dto.HybridConfig) which provides the
// hybrid revalidation configuration.
//
// Returns (nil, nil) if GenerateRevalidatorFunc is nil.
func (m *MockCollectionProvider) GenerateRevalidator(ctx context.Context, collectionName string, targetType ast.Expr, config collection_dto.HybridConfig) (*collection_dto.RuntimeFetcherCode, error) {
	atomic.AddInt64(&m.GenerateRevalidatorCallCount, 1)
	if m.GenerateRevalidatorFunc != nil {
		return m.GenerateRevalidatorFunc(ctx, collectionName, targetType, config)
	}
	return nil, nil
}
