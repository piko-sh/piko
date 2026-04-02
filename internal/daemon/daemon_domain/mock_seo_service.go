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

package daemon_domain

import (
	"context"
	"sync/atomic"

	"piko.sh/piko/internal/seo/seo_dto"
)

// MockSEOService is a test double for SEOServicePort where nil function
// fields return zero values and call counts are tracked atomically.
type MockSEOService struct {
	// GenerateArtefactsFunc is the function called by
	// GenerateArtefacts.
	GenerateArtefactsFunc func(ctx context.Context, view *seo_dto.ProjectView) error

	// GenerateArtefactsCallCount tracks how many times
	// GenerateArtefacts was called.
	GenerateArtefactsCallCount int64
}

var _ SEOServicePort = (*MockSEOService)(nil)

// GenerateArtefacts creates SEO artefacts for the given project view.
//
// Takes ctx (context.Context) which carries deadlines and
// cancellation signals.
// Takes view (*seo_dto.ProjectView) which is the project view
// to generate artefacts for.
//
// Returns error, or nil if GenerateArtefactsFunc is nil.
func (m *MockSEOService) GenerateArtefacts(ctx context.Context, view *seo_dto.ProjectView) error {
	atomic.AddInt64(&m.GenerateArtefactsCallCount, 1)
	if m.GenerateArtefactsFunc != nil {
		return m.GenerateArtefactsFunc(ctx, view)
	}
	return nil
}
