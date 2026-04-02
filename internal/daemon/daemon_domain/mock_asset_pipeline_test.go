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

	"piko.sh/piko/internal/annotator/annotator_dto"
)

func TestMockAssetPipeline_ProcessBuildResult(t *testing.T) {
	t.Parallel()

	t.Run("nil ProcessBuildResultFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockAssetPipeline{}

		err := mock.ProcessBuildResult(context.Background(), &annotator_dto.ProjectAnnotationResult{})

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessBuildResultCallCount))
	})

	t.Run("delegates to ProcessBuildResultFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCtx context.Context
		var capturedResult *annotator_dto.ProjectAnnotationResult

		mock := &MockAssetPipeline{
			ProcessBuildResultFunc: func(ctx context.Context, result *annotator_dto.ProjectAnnotationResult) error {
				capturedCtx = ctx
				capturedResult = result
				return nil
			},
		}

		ctx := context.Background()
		result := &annotator_dto.ProjectAnnotationResult{}

		err := mock.ProcessBuildResult(ctx, result)

		require.NoError(t, err)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, result, capturedResult)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessBuildResultCallCount))
	})

	t.Run("propagates error from ProcessBuildResultFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("pipeline processing failed")
		mock := &MockAssetPipeline{
			ProcessBuildResultFunc: func(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
				return expectedErr
			},
		}

		err := mock.ProcessBuildResult(context.Background(), &annotator_dto.ProjectAnnotationResult{})

		require.Error(t, err)
		assert.Equal(t, expectedErr.Error(), err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessBuildResultCallCount))
	})
}

func TestMockAssetPipeline_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockAssetPipeline

	err := mock.ProcessBuildResult(context.Background(), &annotator_dto.ProjectAnnotationResult{})

	require.NoError(t, err)
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ProcessBuildResultCallCount))
}

func TestMockAssetPipeline_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockAssetPipeline{}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.ProcessBuildResult(context.Background(), &annotator_dto.ProjectAnnotationResult{})
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ProcessBuildResultCallCount))
}

func TestMockAssetPipeline_ImplementsAssetPipelinePort(t *testing.T) {
	t.Parallel()

	var mock MockAssetPipeline
	var _ AssetPipelinePort = &mock
}
