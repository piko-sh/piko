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

package annotator_domain

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

// MockTypeInspectorBuilder is a test double for
// TypeInspectorBuilderPort where nil function fields return zero
// values and call counts are tracked atomically.
type MockTypeInspectorBuilder struct {
	// SetConfigFunc is the function called by SetConfig.
	SetConfigFunc func(config inspector_dto.Config)

	// BuildFunc is the function called by Build.
	BuildFunc func(ctx context.Context, sourceOverlay map[string][]byte, scriptHashes map[string]string) error

	// GetQuerierFunc is the function called by
	// GetQuerier.
	GetQuerierFunc func() (TypeInspectorPort, bool)

	// SetConfigCallCount tracks how many times SetConfig
	// was called.
	SetConfigCallCount int64

	// BuildCallCount tracks how many times Build was
	// called.
	BuildCallCount int64

	// GetQuerierCallCount tracks how many times
	// GetQuerier was called.
	GetQuerierCallCount int64
}

var _ TypeInspectorBuilderPort = (*MockTypeInspectorBuilder)(nil)

// SetConfig delegates to SetConfigFunc if set.
//
// Takes config (inspector_dto.Config) which provides the inspector
// configuration.
//
// Does nothing if SetConfigFunc is nil.
func (m *MockTypeInspectorBuilder) SetConfig(config inspector_dto.Config) {
	atomic.AddInt64(&m.SetConfigCallCount, 1)
	if m.SetConfigFunc != nil {
		m.SetConfigFunc(config)
	}
}

// Build delegates to BuildFunc if set.
//
// Takes ctx (context.Context) which carries deadlines and cancellation
// signals.
// Takes sourceOverlay (map[string][]byte) which maps file paths to
// their overlay source content.
// Takes scriptHashes (map[string]string) which maps file paths to
// their content hashes.
//
// Returns nil if BuildFunc is nil.
func (m *MockTypeInspectorBuilder) Build(ctx context.Context, sourceOverlay map[string][]byte, scriptHashes map[string]string) error {
	atomic.AddInt64(&m.BuildCallCount, 1)
	if m.BuildFunc != nil {
		return m.BuildFunc(ctx, sourceOverlay, scriptHashes)
	}
	return nil
}

// GetQuerier delegates to GetQuerierFunc if set.
//
// Returns (&inspector_domain.MockTypeQuerier{}, true) if
// GetQuerierFunc is nil.
func (m *MockTypeInspectorBuilder) GetQuerier() (TypeInspectorPort, bool) {
	atomic.AddInt64(&m.GetQuerierCallCount, 1)
	if m.GetQuerierFunc != nil {
		return m.GetQuerierFunc()
	}
	return &inspector_domain.MockTypeQuerier{}, true
}
