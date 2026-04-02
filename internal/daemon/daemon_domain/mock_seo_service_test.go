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
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/seo/seo_dto"
)

func TestMockSEOService_GenerateArtefacts(t *testing.T) {
	t.Parallel()

	t.Run("nil GenerateArtefactsFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockSEOService{}

		err := mock.GenerateArtefacts(context.Background(), &seo_dto.ProjectView{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateArtefactsCallCount))
	})

	t.Run("delegates to GenerateArtefactsFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCtx context.Context
		var capturedView *seo_dto.ProjectView

		mock := &MockSEOService{
			GenerateArtefactsFunc: func(ctx context.Context, view *seo_dto.ProjectView) error {
				capturedCtx = ctx
				capturedView = view
				return nil
			},
		}

		ctx := context.Background()
		view := &seo_dto.ProjectView{}

		err := mock.GenerateArtefacts(ctx, view)

		require.NoError(t, err)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, view, capturedView)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateArtefactsCallCount))
	})

	t.Run("propagates error from GenerateArtefactsFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("SEO generation failed")
		mock := &MockSEOService{
			GenerateArtefactsFunc: func(_ context.Context, _ *seo_dto.ProjectView) error {
				return expectedErr
			},
		}

		err := mock.GenerateArtefacts(context.Background(), &seo_dto.ProjectView{})

		require.Error(t, err)
		assert.Equal(t, expectedErr.Error(), err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateArtefactsCallCount))
	})
}

func TestMockSEOService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockSEOService

	err := mock.GenerateArtefacts(context.Background(), &seo_dto.ProjectView{})

	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateArtefactsCallCount))
}

func TestMockSEOService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockSEOService{}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.GenerateArtefacts(context.Background(), &seo_dto.ProjectView{})
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GenerateArtefactsCallCount))
}

func TestMockSEOService_ImplementsSEOServicePort(t *testing.T) {
	t.Parallel()

	var mock MockSEOService
	var _ SEOServicePort = &mock
}
