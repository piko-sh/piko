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

// MockRuntimeProvider is a test double for RuntimeProvider that returns zero
// values from nil function fields and tracks call counts atomically.
type MockRuntimeProvider struct {
	// NameFunc is the function called by Name.
	NameFunc func() string

	// FetchFunc is the function called by Fetch.
	FetchFunc func(ctx context.Context, collectionName string, options *collection_dto.FetchOptions, target any) error

	// NameCallCount tracks how many times Name was
	// called.
	NameCallCount int64

	// FetchCallCount tracks how many times Fetch was
	// called.
	FetchCallCount int64
}

var _ RuntimeProvider = (*MockRuntimeProvider)(nil)

// Name delegates to NameFunc if set.
//
// Returns "" if NameFunc is nil.
func (m *MockRuntimeProvider) Name() string {
	atomic.AddInt64(&m.NameCallCount, 1)
	if m.NameFunc != nil {
		return m.NameFunc()
	}
	return ""
}

// Fetch delegates to FetchFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes collectionName (string) which identifies the collection by name.
// Takes options (*collection_dto.FetchOptions) which provides fetch
// configuration options.
// Takes target (any) which is the destination to decode collection data into.
//
// Returns nil if FetchFunc is nil.
func (m *MockRuntimeProvider) Fetch(ctx context.Context, collectionName string, options *collection_dto.FetchOptions, target any) error {
	atomic.AddInt64(&m.FetchCallCount, 1)
	if m.FetchFunc != nil {
		return m.FetchFunc(ctx, collectionName, options, target)
	}
	return nil
}
