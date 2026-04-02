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

package security_domain

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/healthprobe/healthprobe_dto"
	"piko.sh/piko/internal/security/security_dto"
)

func TestMockCSRFTokenService_GenerateCSRFPair(t *testing.T) {
	t.Parallel()

	t.Run("nil GenerateCSRFPairFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc:      nil,
			ValidateCSRFPairFunc:      nil,
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		buffer := &bytes.Buffer{}

		pair, err := mock.GenerateCSRFPair(w, r, buffer)

		require.NoError(t, err)
		assert.Equal(t, security_dto.CSRFPair{}, pair)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateCSRFPairCallCount))
	})

	t.Run("delegates to GenerateCSRFPairFunc", func(t *testing.T) {
		t.Parallel()

		expected := security_dto.CSRFPair{
			RawEphemeralToken: "ephemeral-token-abc",
			ActionToken:       []byte("action-token-xyz"),
		}

		var capturedW http.ResponseWriter
		var capturedR *http.Request
		var capturedBuf *bytes.Buffer

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc: func(w http.ResponseWriter, r *http.Request, buffer *bytes.Buffer) (security_dto.CSRFPair, error) {
				capturedW = w
				capturedR = r
				capturedBuf = buffer
				return expected, nil
			},
			ValidateCSRFPairFunc:      nil,
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/form", nil)
		buffer := bytes.NewBufferString("seed")

		pair, err := mock.GenerateCSRFPair(w, r, buffer)

		require.NoError(t, err)
		assert.Equal(t, expected, pair)
		assert.Same(t, w, capturedW)
		assert.Same(t, r, capturedR)
		assert.Same(t, buffer, capturedBuf)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateCSRFPairCallCount))
	})

	t.Run("propagates error from GenerateCSRFPairFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc: func(_ http.ResponseWriter, _ *http.Request, _ *bytes.Buffer) (security_dto.CSRFPair, error) {
				return security_dto.CSRFPair{}, errors.New("token generation failed")
			},
			ValidateCSRFPairFunc:      nil,
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		buffer := &bytes.Buffer{}

		pair, err := mock.GenerateCSRFPair(w, r, buffer)

		require.Error(t, err)
		assert.Equal(t, "token generation failed", err.Error())
		assert.Equal(t, security_dto.CSRFPair{}, pair)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateCSRFPairCallCount))
	})
}

func TestMockCSRFTokenService_ValidateCSRFPair(t *testing.T) {
	t.Parallel()

	t.Run("nil ValidateCSRFPairFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc:      nil,
			ValidateCSRFPairFunc:      nil,
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		r := httptest.NewRequest(http.MethodPost, "/submit", nil)

		valid, err := mock.ValidateCSRFPair(r, "ephemeral-token", []byte("action-token"))

		require.NoError(t, err)
		assert.False(t, valid)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ValidateCSRFPairCallCount))
	})

	t.Run("delegates to ValidateCSRFPairFunc", func(t *testing.T) {
		t.Parallel()

		var capturedR *http.Request
		var capturedEphemeral string
		var capturedAction []byte

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc: nil,
			ValidateCSRFPairFunc: func(r *http.Request, rawEphemeralTokenFromRequest string, actionToken []byte) (bool, error) {
				capturedR = r
				capturedEphemeral = rawEphemeralTokenFromRequest
				capturedAction = actionToken
				return true, nil
			},
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		r := httptest.NewRequest(http.MethodPost, "/api/action", nil)
		ephemeral := "eph-token-42"
		action := []byte("act-token-99")

		valid, err := mock.ValidateCSRFPair(r, ephemeral, action)

		require.NoError(t, err)
		assert.True(t, valid)
		assert.Same(t, r, capturedR)
		assert.Equal(t, ephemeral, capturedEphemeral)
		assert.Equal(t, action, capturedAction)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ValidateCSRFPairCallCount))
	})

	t.Run("propagates error from ValidateCSRFPairFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc: nil,
			ValidateCSRFPairFunc: func(_ *http.Request, _ string, _ []byte) (bool, error) {
				return false, errors.New("validation failed")
			},
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		r := httptest.NewRequest(http.MethodPost, "/submit", nil)

		valid, err := mock.ValidateCSRFPair(r, "token", []byte("action"))

		require.Error(t, err)
		assert.Equal(t, "validation failed", err.Error())
		assert.False(t, valid)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ValidateCSRFPairCallCount))
	})
}

func TestMockCSRFTokenService_Name(t *testing.T) {
	t.Parallel()

	t.Run("nil NameFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc:      nil,
			ValidateCSRFPairFunc:      nil,
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		name := mock.Name()

		assert.Equal(t, "", name)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NameCallCount))
	})

	t.Run("delegates to NameFunc", func(t *testing.T) {
		t.Parallel()

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc: nil,
			ValidateCSRFPairFunc: nil,
			NameFunc: func() string {
				return "csrf-service"
			},
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		name := mock.Name()

		assert.Equal(t, "csrf-service", name)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NameCallCount))
	})
}

func TestMockCSRFTokenService_Check(t *testing.T) {
	t.Parallel()

	t.Run("nil CheckFunc returns zero values", func(t *testing.T) {
		t.Parallel()

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc:      nil,
			ValidateCSRFPairFunc:      nil,
			NameFunc:                  nil,
			CheckFunc:                 nil,
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		ctx := context.Background()
		status := mock.Check(ctx, healthprobe_dto.CheckTypeReadiness)

		assert.Equal(t, healthprobe_dto.Status{}, status)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CheckCallCount))
	})

	t.Run("delegates to CheckFunc", func(t *testing.T) {
		t.Parallel()

		expected := healthprobe_dto.Status{
			Name:    "csrf-service",
			State:   healthprobe_dto.StateHealthy,
			Message: "all good",
		}

		var capturedCtx context.Context
		var capturedCheckType healthprobe_dto.CheckType

		mock := &MockCSRFTokenService{
			GenerateCSRFPairFunc: nil,
			ValidateCSRFPairFunc: nil,
			NameFunc:             nil,
			CheckFunc: func(ctx context.Context, checkType healthprobe_dto.CheckType) healthprobe_dto.Status {
				capturedCtx = ctx
				capturedCheckType = checkType
				return expected
			},
			GenerateCSRFPairCallCount: 0,
			ValidateCSRFPairCallCount: 0,
			NameCallCount:             0,
			CheckCallCount:            0,
		}

		ctx := context.WithValue(context.Background(), csrfTestContextKey{}, "test-val")
		status := mock.Check(ctx, healthprobe_dto.CheckTypeLiveness)

		assert.Equal(t, expected, status)
		assert.Equal(t, ctx, capturedCtx)
		assert.Equal(t, healthprobe_dto.CheckTypeLiveness, capturedCheckType)
		assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CheckCallCount))
	})
}

type csrfTestContextKey struct{}

func TestMockCSRFTokenService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var mock MockCSRFTokenService

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	buffer := &bytes.Buffer{}

	pair, err := mock.GenerateCSRFPair(w, r, buffer)
	require.NoError(t, err)
	assert.Equal(t, security_dto.CSRFPair{}, pair)

	valid, err := mock.ValidateCSRFPair(r, "token", []byte("action"))
	require.NoError(t, err)
	assert.False(t, valid)

	name := mock.Name()
	assert.Equal(t, "", name)

	ctx := context.Background()
	status := mock.Check(ctx, healthprobe_dto.CheckTypeReadiness)
	assert.Equal(t, healthprobe_dto.Status{}, status)

	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.GenerateCSRFPairCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.ValidateCSRFPairCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.NameCallCount))
	assert.Equal(t, int64(1), atomic.LoadInt64(&mock.CheckCallCount))
}

func TestMockCSRFTokenService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	mock := &MockCSRFTokenService{
		GenerateCSRFPairFunc:      nil,
		ValidateCSRFPairFunc:      nil,
		NameFunc:                  nil,
		CheckFunc:                 nil,
		GenerateCSRFPairCallCount: 0,
		ValidateCSRFPairCallCount: 0,
		NameCallCount:             0,
		CheckCallCount:            0,
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 4)

	for range goroutines {
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			buffer := &bytes.Buffer{}
			_, _ = mock.GenerateCSRFPair(w, r, buffer)
		}()
		go func() {
			defer wg.Done()
			r := httptest.NewRequest(http.MethodPost, "/submit", nil)
			_, _ = mock.ValidateCSRFPair(r, "token", []byte("action"))
		}()
		go func() {
			defer wg.Done()
			_ = mock.Name()
		}()
		go func() {
			defer wg.Done()
			ctx := context.Background()
			_ = mock.Check(ctx, healthprobe_dto.CheckTypeReadiness)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.GenerateCSRFPairCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.ValidateCSRFPairCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.NameCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&mock.CheckCallCount))
}
