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

package capabilities_domain

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockCapabilityService_Register(t *testing.T) {
	t.Parallel()

	t.Run("nil RegisterFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCapabilityService{
			RegisterFunc:      nil,
			ExecuteFunc:       nil,
			RegisterCallCount: 0,
			ExecuteCallCount:  0,
		}

		err := mock.Register("compress", nil)

		require.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterCallCount))
	})

	t.Run("delegates to RegisterFunc", func(t *testing.T) {
		t.Parallel()

		var capturedName string
		var capturedFunction CapabilityFunc

		capabilityFunction := func(_ context.Context, _ io.Reader, _ CapabilityParams) (io.Reader, error) {
			return nil, nil
		}

		mock := &MockCapabilityService{
			RegisterFunc: func(name string, f CapabilityFunc) error {
				capturedName = name
				capturedFunction = f
				return nil
			},
			ExecuteFunc:       nil,
			RegisterCallCount: 0,
			ExecuteCallCount:  0,
		}

		err := mock.Register("minify", capabilityFunction)

		require.NoError(t, err)
		assert.Equal(t, "minify", capturedName)
		assert.NotNil(t, capturedFunction)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterCallCount))
	})

	t.Run("propagates error from RegisterFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockCapabilityService{
			RegisterFunc: func(_ string, _ CapabilityFunc) error {
				return errors.New("registration failed")
			},
			ExecuteFunc:       nil,
			RegisterCallCount: 0,
			ExecuteCallCount:  0,
		}

		err := mock.Register("broken", nil)

		require.Error(t, err)
		assert.Equal(t, "registration failed", err.Error())
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterCallCount))
	})
}

func TestMockCapabilityService_Execute(t *testing.T) {
	t.Parallel()

	t.Run("nil ExecuteFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCapabilityService{
			RegisterFunc:      nil,
			ExecuteFunc:       nil,
			RegisterCallCount: 0,
			ExecuteCallCount:  0,
		}

		reader, err := mock.Execute(context.Background(), "compress", nil, nil)

		require.NoError(t, err)
		assert.Nil(t, reader)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ExecuteCallCount))
	})

	t.Run("delegates to ExecuteFunc", func(t *testing.T) {
		t.Parallel()

		var capturedCtx context.Context
		var capturedName string
		var capturedParams CapabilityParams

		expectedOutput := bytes.NewBufferString("compressed-output")

		mock := &MockCapabilityService{
			RegisterFunc: nil,
			ExecuteFunc: func(ctx context.Context, capabilityName string, _ io.Reader, params CapabilityParams) (io.Reader, error) {
				capturedCtx = ctx
				capturedName = capabilityName
				capturedParams = params
				return expectedOutput, nil
			},
			RegisterCallCount: 0,
			ExecuteCallCount:  0,
		}

		ctx := context.Background()
		params := CapabilityParams{"level": "9"}

		reader, err := mock.Execute(ctx, "compress", nil, params)

		require.NoError(t, err)
		assert.Equal(t, expectedOutput, reader)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, "compress", capturedName)
		assert.Equal(t, CapabilityParams{"level": "9"}, capturedParams)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ExecuteCallCount))
	})

	t.Run("propagates error from ExecuteFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockCapabilityService{
			RegisterFunc: nil,
			ExecuteFunc: func(_ context.Context, _ string, _ io.Reader, _ CapabilityParams) (io.Reader, error) {
				return nil, errors.New("execution failed")
			},
			RegisterCallCount: 0,
			ExecuteCallCount:  0,
		}

		reader, err := mock.Execute(context.Background(), "missing", nil, nil)

		require.Error(t, err)
		assert.Equal(t, "execution failed", err.Error())
		assert.Nil(t, reader)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ExecuteCallCount))
	})
}

func TestMockCapabilityService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockCapabilityService

	err := mock.Register("cap", nil)
	require.NoError(t, err)

	reader, err := mock.Execute(context.Background(), "cap", nil, nil)
	require.NoError(t, err)
	assert.Nil(t, reader)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.RegisterCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ExecuteCallCount))
}

func TestMockCapabilityService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockCapabilityService{
		RegisterFunc:      nil,
		ExecuteFunc:       nil,
		RegisterCallCount: 0,
		ExecuteCallCount:  0,
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for range goroutines {
		go func() {
			defer wg.Done()
			_ = mock.Register("concurrent", nil)
		}()
		go func() {
			defer wg.Done()
			_, _ = mock.Execute(context.Background(), "concurrent", nil, nil)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.RegisterCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ExecuteCallCount))
}
