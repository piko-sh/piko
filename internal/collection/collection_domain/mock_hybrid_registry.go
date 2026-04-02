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
	"sync/atomic"

	"piko.sh/piko/internal/collection/collection_dto"
)

// MockHybridRegistry is a test double for HybridRegistryPort that returns
// zero values from nil function fields and tracks call counts atomically.
type MockHybridRegistry struct {
	// RegisterFunc is the function called by Register.
	RegisterFunc func(ctx context.Context, providerName, collectionName string, blob []byte, etag string, config collection_dto.HybridConfig)

	// GetBlobFunc is the function called by GetBlob.
	GetBlobFunc func(ctx context.Context, providerName, collectionName string) (blob []byte, needsRevalidation bool)

	// GetETagFunc is the function called by GetETag.
	GetETagFunc func(providerName, collectionName string) string

	// HasFunc is the function called by Has.
	HasFunc func(providerName, collectionName string) bool

	// ListFunc is the function called by List.
	ListFunc func() []string

	// TriggerRevalidationFunc is the function called by
	// TriggerRevalidation.
	TriggerRevalidationFunc func(ctx context.Context, providerName, collectionName string)

	// RegisterCallCount tracks how many times Register
	// was called.
	RegisterCallCount int64

	// GetBlobCallCount tracks how many times GetBlob was
	// called.
	GetBlobCallCount int64

	// GetETagCallCount tracks how many times GetETag was
	// called.
	GetETagCallCount int64

	// HasCallCount tracks how many times Has was called.
	HasCallCount int64

	// ListCallCount tracks how many times List was
	// called.
	ListCallCount int64

	// TriggerRevalidationCallCount tracks how many times
	// TriggerRevalidation was called.
	TriggerRevalidationCallCount int64
}

var _ HybridRegistryPort = (*MockHybridRegistry)(nil)

// Register delegates to RegisterFunc if set.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes providerName (string) which identifies the provider by name.
// Takes collectionName (string) which identifies the collection by name.
// Takes blob ([]byte) which is the collection data blob.
// Takes etag (string) which is the ETag for cache validation.
// Takes config (collection_dto.HybridConfig) which provides the
// hybrid configuration.
//
// Does nothing if RegisterFunc is nil.
func (m *MockHybridRegistry) Register(ctx context.Context, providerName, collectionName string, blob []byte, etag string, config collection_dto.HybridConfig) {
	atomic.AddInt64(&m.RegisterCallCount, 1)
	if m.RegisterFunc != nil {
		m.RegisterFunc(ctx, providerName, collectionName, blob, etag, config)
	}
}

// GetBlob delegates to GetBlobFunc if set.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes providerName (string) which identifies the provider by name.
// Takes collectionName (string) which identifies the collection by name.
//
// Returns (nil, false) if GetBlobFunc is nil.
func (m *MockHybridRegistry) GetBlob(ctx context.Context, providerName, collectionName string) ([]byte, bool) {
	atomic.AddInt64(&m.GetBlobCallCount, 1)
	if m.GetBlobFunc != nil {
		return m.GetBlobFunc(ctx, providerName, collectionName)
	}
	return nil, false
}

// GetETag delegates to GetETagFunc if set.
//
// Takes providerName (string) which identifies the provider by name.
// Takes collectionName (string) which identifies the collection by name.
//
// Returns "" if GetETagFunc is nil.
func (m *MockHybridRegistry) GetETag(providerName, collectionName string) string {
	atomic.AddInt64(&m.GetETagCallCount, 1)
	if m.GetETagFunc != nil {
		return m.GetETagFunc(providerName, collectionName)
	}
	return ""
}

// Has delegates to HasFunc if set.
//
// Takes providerName (string) which identifies the provider by name.
// Takes collectionName (string) which identifies the collection by name.
//
// Returns false if HasFunc is nil.
func (m *MockHybridRegistry) Has(providerName, collectionName string) bool {
	atomic.AddInt64(&m.HasCallCount, 1)
	if m.HasFunc != nil {
		return m.HasFunc(providerName, collectionName)
	}
	return false
}

// List delegates to ListFunc if set.
//
// Returns nil if ListFunc is nil.
func (m *MockHybridRegistry) List() []string {
	atomic.AddInt64(&m.ListCallCount, 1)
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return nil
}

// TriggerRevalidation delegates to TriggerRevalidationFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes providerName (string) which identifies the provider by name.
// Takes collectionName (string) which identifies the collection by name.
//
// Does nothing if TriggerRevalidationFunc is nil.
func (m *MockHybridRegistry) TriggerRevalidation(ctx context.Context, providerName, collectionName string) {
	atomic.AddInt64(&m.TriggerRevalidationCallCount, 1)
	if m.TriggerRevalidationFunc != nil {
		m.TriggerRevalidationFunc(ctx, providerName, collectionName)
	}
}
