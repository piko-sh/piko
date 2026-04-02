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

package templater_domain_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestMockTemplaterService_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m templater_domain.MockTemplaterService

	ctx := context.Background()
	page := templater_dto.PageDefinition{}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	websiteConfig := &config.WebsiteConfig{}

	result, err := m.ProbePage(ctx, page, request, websiteConfig)
	assert.Nil(t, result)
	assert.NoError(t, err)

	err = m.RenderPage(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        io.Discard,
		Response:      httptest.NewRecorder(),
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})
	assert.NoError(t, err)

	result2, err2 := m.ProbePartial(ctx, page, request, websiteConfig)
	assert.Nil(t, result2)
	assert.NoError(t, err2)

	err = m.RenderPartial(ctx, templater_domain.RenderRequest{
		Page:          page,
		Writer:        io.Discard,
		Response:      httptest.NewRecorder(),
		Request:       request,
		IsFragment:    false,
		WebsiteConfig: websiteConfig,
	})
	assert.NoError(t, err)

	m.SetRunner(nil)
}

func TestMockTemplaterService_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &templater_domain.MockTemplaterService{
		ProbePageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
			return nil, nil
		},
		RenderPageFunc: func(_ context.Context, _ templater_domain.RenderRequest) error {
			return nil
		},
		ProbePartialFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
			return nil, nil
		},
		RenderPartialFunc: func(_ context.Context, _ templater_domain.RenderRequest) error {
			return nil
		},
		SetRunnerFunc: func(_ templater_domain.ManifestRunnerPort) {},
	}

	const goroutines = 50
	ctx := context.Background()
	page := templater_dto.PageDefinition{}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	websiteConfig := &config.WebsiteConfig{}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _ = m.ProbePage(ctx, page, request, websiteConfig)
			_ = m.RenderPage(ctx, templater_domain.RenderRequest{
				Page:          page,
				Writer:        io.Discard,
				Response:      httptest.NewRecorder(),
				Request:       request,
				IsFragment:    false,
				WebsiteConfig: websiteConfig,
			})
			_, _ = m.ProbePartial(ctx, page, request, websiteConfig)
			_ = m.RenderPartial(ctx, templater_domain.RenderRequest{
				Page:          page,
				Writer:        io.Discard,
				Response:      httptest.NewRecorder(),
				Request:       request,
				IsFragment:    false,
				WebsiteConfig: websiteConfig,
			})
			m.SetRunner(nil)
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ProbePageCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RenderPageCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.ProbePartialCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RenderPartialCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.SetRunnerCallCount))
}

func TestMockTemplaterService_ProbePage(t *testing.T) {
	t.Parallel()

	t.Run("nil ProbePageFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockTemplaterService{}
		ctx := context.Background()
		page := templater_dto.PageDefinition{}
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		websiteConfig := &config.WebsiteConfig{}

		result, err := m.ProbePage(ctx, page, request, websiteConfig)
		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ProbePageCallCount))
	})

	t.Run("delegates to ProbePageFunc", func(t *testing.T) {
		t.Parallel()
		expected := &templater_dto.PageProbeResult{}
		m := &templater_domain.MockTemplaterService{
			ProbePageFunc: func(ctx context.Context, page templater_dto.PageDefinition, request *http.Request, websiteConfig *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				require.NotNil(t, ctx)
				return expected, nil
			},
		}
		ctx := context.Background()
		result, err := m.ProbePage(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), &config.WebsiteConfig{})
		assert.Same(t, expected, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ProbePageCallCount))
	})

	t.Run("propagates error from ProbePageFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("probe failed")
		m := &templater_domain.MockTemplaterService{
			ProbePageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				return nil, expectedErr
			},
		}
		ctx := context.Background()
		result, err := m.ProbePage(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), &config.WebsiteConfig{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ProbePageCallCount))
	})
}

func TestMockTemplaterService_RenderPage(t *testing.T) {
	t.Parallel()

	t.Run("nil RenderPageFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockTemplaterService{}
		ctx := context.Background()
		err := m.RenderPage(ctx, templater_domain.RenderRequest{
			Page:          templater_dto.PageDefinition{},
			Writer:        io.Discard,
			Response:      httptest.NewRecorder(),
			Request:       httptest.NewRequest(http.MethodGet, "/", nil),
			IsFragment:    false,
			WebsiteConfig: &config.WebsiteConfig{},
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPageCallCount))
	})

	t.Run("delegates to RenderPageFunc", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		m := &templater_domain.MockTemplaterService{
			RenderPageFunc: func(_ context.Context, req templater_domain.RenderRequest) error {
				_, writeErr := req.Writer.Write([]byte("rendered"))
				return writeErr
			},
		}
		ctx := context.Background()
		err := m.RenderPage(ctx, templater_domain.RenderRequest{
			Page:          templater_dto.PageDefinition{},
			Writer:        &buffer,
			Response:      httptest.NewRecorder(),
			Request:       httptest.NewRequest(http.MethodGet, "/", nil),
			IsFragment:    false,
			WebsiteConfig: &config.WebsiteConfig{},
		})
		assert.NoError(t, err)
		assert.Equal(t, "rendered", buffer.String())
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPageCallCount))
	})

	t.Run("propagates error from RenderPageFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("render failed")
		m := &templater_domain.MockTemplaterService{
			RenderPageFunc: func(_ context.Context, _ templater_domain.RenderRequest) error {
				return expectedErr
			},
		}
		ctx := context.Background()
		err := m.RenderPage(ctx, templater_domain.RenderRequest{
			Page:          templater_dto.PageDefinition{},
			Writer:        io.Discard,
			Response:      httptest.NewRecorder(),
			Request:       httptest.NewRequest(http.MethodGet, "/", nil),
			IsFragment:    false,
			WebsiteConfig: &config.WebsiteConfig{},
		})
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPageCallCount))
	})
}

func TestMockTemplaterService_ProbePartial(t *testing.T) {
	t.Parallel()

	t.Run("nil ProbePartialFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockTemplaterService{}
		ctx := context.Background()
		result, err := m.ProbePartial(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), &config.WebsiteConfig{})
		assert.Nil(t, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ProbePartialCallCount))
	})

	t.Run("delegates to ProbePartialFunc", func(t *testing.T) {
		t.Parallel()
		expected := &templater_dto.PageProbeResult{}
		m := &templater_domain.MockTemplaterService{
			ProbePartialFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				return expected, nil
			},
		}
		ctx := context.Background()
		result, err := m.ProbePartial(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), &config.WebsiteConfig{})
		assert.Same(t, expected, result)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ProbePartialCallCount))
	})

	t.Run("propagates error from ProbePartialFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("partial probe failed")
		m := &templater_domain.MockTemplaterService{
			ProbePartialFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ *config.WebsiteConfig) (*templater_dto.PageProbeResult, error) {
				return nil, expectedErr
			},
		}
		ctx := context.Background()
		result, err := m.ProbePartial(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), &config.WebsiteConfig{})
		assert.Nil(t, result)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.ProbePartialCallCount))
	})
}

func TestMockTemplaterService_RenderPartial(t *testing.T) {
	t.Parallel()

	t.Run("nil RenderPartialFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockTemplaterService{}
		ctx := context.Background()
		err := m.RenderPartial(ctx, templater_domain.RenderRequest{
			Page:          templater_dto.PageDefinition{},
			Writer:        io.Discard,
			Response:      httptest.NewRecorder(),
			Request:       httptest.NewRequest(http.MethodGet, "/", nil),
			IsFragment:    false,
			WebsiteConfig: &config.WebsiteConfig{},
		})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPartialCallCount))
	})

	t.Run("delegates to RenderPartialFunc", func(t *testing.T) {
		t.Parallel()
		var buffer bytes.Buffer
		m := &templater_domain.MockTemplaterService{
			RenderPartialFunc: func(_ context.Context, req templater_domain.RenderRequest) error {
				_, writeErr := req.Writer.Write([]byte("partial"))
				return writeErr
			},
		}
		ctx := context.Background()
		err := m.RenderPartial(ctx, templater_domain.RenderRequest{
			Page:          templater_dto.PageDefinition{},
			Writer:        &buffer,
			Response:      httptest.NewRecorder(),
			Request:       httptest.NewRequest(http.MethodGet, "/", nil),
			IsFragment:    false,
			WebsiteConfig: &config.WebsiteConfig{},
		})
		assert.NoError(t, err)
		assert.Equal(t, "partial", buffer.String())
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPartialCallCount))
	})

	t.Run("propagates error from RenderPartialFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("partial render failed")
		m := &templater_domain.MockTemplaterService{
			RenderPartialFunc: func(_ context.Context, _ templater_domain.RenderRequest) error {
				return expectedErr
			},
		}
		ctx := context.Background()
		err := m.RenderPartial(ctx, templater_domain.RenderRequest{
			Page:          templater_dto.PageDefinition{},
			Writer:        io.Discard,
			Response:      httptest.NewRecorder(),
			Request:       httptest.NewRequest(http.MethodGet, "/", nil),
			IsFragment:    false,
			WebsiteConfig: &config.WebsiteConfig{},
		})
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPartialCallCount))
	})
}

func TestMockTemplaterService_SetRunner(t *testing.T) {
	t.Parallel()

	t.Run("nil SetRunnerFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockTemplaterService{}
		m.SetRunner(nil)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SetRunnerCallCount))
	})

	t.Run("delegates to SetRunnerFunc", func(t *testing.T) {
		t.Parallel()
		var captured templater_domain.ManifestRunnerPort
		runner := &templater_domain.MockManifestRunnerPort{}
		m := &templater_domain.MockTemplaterService{
			SetRunnerFunc: func(r templater_domain.ManifestRunnerPort) {
				captured = r
			},
		}
		m.SetRunner(runner)
		assert.Same(t, runner, captured)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.SetRunnerCallCount))
	})
}
