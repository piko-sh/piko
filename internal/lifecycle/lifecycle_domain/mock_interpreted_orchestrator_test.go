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

package lifecycle_domain

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
)

func TestMockInterpretedOrchestrator_BuildRunner(t *testing.T) {
	t.Parallel()

	t.Run("nil BuildRunnerFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockInterpretedOrchestrator{}

		runner, err := m.BuildRunner(context.Background(), &annotator_dto.ProjectAnnotationResult{})
		assert.NoError(t, err)
		assert.Nil(t, runner)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.BuildRunnerCallCount))
	})

	t.Run("delegates to BuildRunnerFunc", func(t *testing.T) {
		t.Parallel()

		expectedResult := &annotator_dto.ProjectAnnotationResult{}
		var receivedResult *annotator_dto.ProjectAnnotationResult

		m := &MockInterpretedOrchestrator{
			BuildRunnerFunc: func(_ context.Context, result *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error) {
				receivedResult = result
				return nil, nil
			},
		}

		runner, err := m.BuildRunner(context.Background(), expectedResult)
		require.NoError(t, err)
		assert.Nil(t, runner)
		assert.Same(t, expectedResult, receivedResult)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.BuildRunnerCallCount))
	})

	t.Run("propagates error from BuildRunnerFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("build runner creation failed")
		m := &MockInterpretedOrchestrator{
			BuildRunnerFunc: func(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error) {
				return nil, expectedErr
			},
		}

		runner, err := m.BuildRunner(context.Background(), &annotator_dto.ProjectAnnotationResult{})
		assert.Nil(t, runner)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.BuildRunnerCallCount))
	})
}

func TestMockInterpretedOrchestrator_MarkDirty(t *testing.T) {
	t.Parallel()

	t.Run("nil MarkDirtyFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockInterpretedOrchestrator{}

		err := m.MarkDirty(context.Background(), &annotator_dto.ProjectAnnotationResult{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.MarkDirtyCallCount))
	})

	t.Run("delegates to MarkDirtyFunc", func(t *testing.T) {
		t.Parallel()

		expectedResult := &annotator_dto.ProjectAnnotationResult{}
		var receivedResult *annotator_dto.ProjectAnnotationResult

		m := &MockInterpretedOrchestrator{
			MarkDirtyFunc: func(_ context.Context, result *annotator_dto.ProjectAnnotationResult) error {
				receivedResult = result
				return nil
			},
		}

		err := m.MarkDirty(context.Background(), expectedResult)
		assert.NoError(t, err)
		assert.Same(t, expectedResult, receivedResult)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.MarkDirtyCallCount))
	})

	t.Run("propagates error from MarkDirtyFunc", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("mark dirty failed")
		m := &MockInterpretedOrchestrator{
			MarkDirtyFunc: func(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
				return expectedErr
			},
		}

		err := m.MarkDirty(context.Background(), &annotator_dto.ProjectAnnotationResult{})
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.MarkDirtyCallCount))
	})
}

func TestMockInterpretedOrchestrator_IsInitialised(t *testing.T) {
	t.Parallel()

	t.Run("nil IsInitialisedFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		m := &MockInterpretedOrchestrator{}

		result := m.IsInitialised()
		assert.False(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.IsInitialisedCallCount))
	})

	t.Run("delegates to IsInitialisedFunc returning true", func(t *testing.T) {
		t.Parallel()

		m := &MockInterpretedOrchestrator{
			IsInitialisedFunc: func() bool {
				return true
			},
		}

		result := m.IsInitialised()
		assert.True(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.IsInitialisedCallCount))
	})

	t.Run("delegates to IsInitialisedFunc returning false", func(t *testing.T) {
		t.Parallel()

		m := &MockInterpretedOrchestrator{
			IsInitialisedFunc: func() bool {
				return false
			},
		}

		result := m.IsInitialised()
		assert.False(t, result)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.IsInitialisedCallCount))
	})
}

func TestMockInterpretedOrchestrator_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	m := &MockInterpretedOrchestrator{}

	runner, err := m.BuildRunner(context.Background(), &annotator_dto.ProjectAnnotationResult{})
	assert.NoError(t, err)
	assert.Nil(t, runner)

	err = m.MarkDirty(context.Background(), &annotator_dto.ProjectAnnotationResult{})
	assert.NoError(t, err)

	initialised := m.IsInitialised()
	assert.False(t, initialised)

	assert.Equal(t, int64(1), atomic.LoadInt64(&m.BuildRunnerCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.MarkDirtyCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&m.IsInitialisedCallCount))
}

func TestMockInterpretedOrchestrator_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	const goroutines = 50

	m := &MockInterpretedOrchestrator{
		BuildRunnerFunc: func(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) (templater_domain.ManifestRunnerPort, error) {
			return nil, nil
		},
		MarkDirtyFunc: func(_ context.Context, _ *annotator_dto.ProjectAnnotationResult) error {
			return nil
		},
		IsInitialisedFunc: func() bool {
			return true
		},
	}

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.BuildRunner(context.Background(), &annotator_dto.ProjectAnnotationResult{})
		}()
		go func() {
			defer wg.Done()
			_ = m.MarkDirty(context.Background(), &annotator_dto.ProjectAnnotationResult{})
		}()
		go func() {
			defer wg.Done()
			_ = m.IsInitialised()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.BuildRunnerCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.MarkDirtyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.IsInitialisedCallCount))
}
