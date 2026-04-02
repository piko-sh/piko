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

package registry_domain

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/registry/registry_dto"
)

// MockMetadataCache is a test double for MetadataCache where nil
// function fields return zero values and call counts are tracked
// atomically.
type MockMetadataCache struct {
	// GetFunc is the function called by Get.
	GetFunc func(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error)

	// GetMultipleFunc is the function called by
	// GetMultiple.
	GetMultipleFunc func(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, []string)

	// SetFunc is the function called by Set.
	SetFunc func(ctx context.Context, artefact *registry_dto.ArtefactMeta)

	// SetMultipleFunc is the function called by
	// SetMultiple.
	SetMultipleFunc func(ctx context.Context, artefacts []*registry_dto.ArtefactMeta)

	// DeleteFunc is the function called by Delete.
	DeleteFunc func(ctx context.Context, artefactID string)

	// CloseFunc is the function called by Close.
	CloseFunc func(ctx context.Context) error

	// GetCallCount tracks how many times Get was called.
	GetCallCount int64

	// GetMultipleCallCount tracks how many times
	// GetMultiple was called.
	GetMultipleCallCount int64

	// SetCallCount tracks how many times Set was called.
	SetCallCount int64

	// SetMultipleCallCount tracks how many times
	// SetMultiple was called.
	SetMultipleCallCount int64

	// DeleteCallCount tracks how many times Delete was
	// called.
	DeleteCallCount int64

	// CloseCallCount tracks how many times Close was
	// called.
	CloseCallCount int64
}

// Get retrieves artefact metadata by ID.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to look up.
//
// Returns (*ArtefactMeta, error), or (nil, nil) if GetFunc is nil.
func (m *MockMetadataCache) Get(ctx context.Context, artefactID string) (*registry_dto.ArtefactMeta, error) {
	atomic.AddInt64(&m.GetCallCount, 1)
	if m.GetFunc != nil {
		return m.GetFunc(ctx, artefactID)
	}
	return nil, nil
}

// GetMultiple retrieves metadata for multiple artefacts.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactIDs ([]string) which lists the artefact IDs to look up.
//
// Returns ([]*ArtefactMeta, []string), or (nil, nil) if GetMultipleFunc
// is nil.
func (m *MockMetadataCache) GetMultiple(ctx context.Context, artefactIDs []string) ([]*registry_dto.ArtefactMeta, []string) {
	atomic.AddInt64(&m.GetMultipleCallCount, 1)
	if m.GetMultipleFunc != nil {
		return m.GetMultipleFunc(ctx, artefactIDs)
	}
	return nil, nil
}

// Set stores artefact metadata in the cache.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefact (*registry_dto.ArtefactMeta) which is the metadata to cache.
func (m *MockMetadataCache) Set(ctx context.Context, artefact *registry_dto.ArtefactMeta) {
	atomic.AddInt64(&m.SetCallCount, 1)
	if m.SetFunc != nil {
		m.SetFunc(ctx, artefact)
	}
}

// SetMultiple stores multiple artefact metadata entries in the cache.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefacts ([]*registry_dto.ArtefactMeta) which
// are the metadata entries to cache.
func (m *MockMetadataCache) SetMultiple(ctx context.Context, artefacts []*registry_dto.ArtefactMeta) {
	atomic.AddInt64(&m.SetMultipleCallCount, 1)
	if m.SetMultipleFunc != nil {
		m.SetMultipleFunc(ctx, artefacts)
	}
}

// Delete removes artefact metadata from the cache by ID.
//
// Takes ctx (context.Context) which carries deadlines and cancellation signals.
// Takes artefactID (string) which identifies the artefact to remove.
func (m *MockMetadataCache) Delete(ctx context.Context, artefactID string) {
	atomic.AddInt64(&m.DeleteCallCount, 1)
	if m.DeleteFunc != nil {
		m.DeleteFunc(ctx, artefactID)
	}
}

// Close shuts down the cache.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
//
// Returns error, or nil if CloseFunc is nil.
func (m *MockMetadataCache) Close(ctx context.Context) error {
	atomic.AddInt64(&m.CloseCallCount, 1)
	if m.CloseFunc != nil {
		return m.CloseFunc(ctx)
	}
	return nil
}
