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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/email/email_dto"
	"piko.sh/piko/internal/render/render_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestMockRendererPort_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m templater_domain.MockRendererPort

	ctx := context.Background()

	links, probeData, err := m.CollectMetadata(ctx, httptest.NewRequest(http.MethodGet, "/", nil), &templater_dto.InternalMetadata{}, &config.WebsiteConfig{})
	assert.Nil(t, links)
	assert.Nil(t, probeData)
	assert.NoError(t, err)

	err = m.RenderPage(ctx, templater_domain.RenderPageParams{})
	assert.NoError(t, err)

	err = m.RenderPartial(ctx, templater_domain.RenderPageParams{})
	assert.NoError(t, err)

	err = m.RenderEmail(ctx, templater_domain.RenderEmailParams{})
	assert.NoError(t, err)

	text, err := m.RenderASTToPlainText(ctx, nil)
	assert.Empty(t, text)
	assert.NoError(t, err)

	requests := m.GetLastEmailAssetRequests()
	assert.Nil(t, requests)
}

func TestMockRendererPort_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &templater_domain.MockRendererPort{
		CollectMetadataFunc: func(_ context.Context, _ *http.Request, _ *templater_dto.InternalMetadata, _ *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
			return nil, nil, nil
		},
		RenderPageFunc: func(_ context.Context, _ templater_domain.RenderPageParams) error {
			return nil
		},
		RenderPartialFunc: func(_ context.Context, _ templater_domain.RenderPageParams) error {
			return nil
		},
		RenderEmailFunc: func(_ context.Context, _ templater_domain.RenderEmailParams) error {
			return nil
		},
		RenderASTToPlainTextFunc: func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
			return "", nil
		},
		GetLastEmailAssetRequestsFunc: func() []*email_dto.EmailAssetRequest {
			return nil
		},
	}

	const goroutines = 50

	ctx := context.Background()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _, _ = m.CollectMetadata(ctx, request, &templater_dto.InternalMetadata{}, &config.WebsiteConfig{})
			_ = m.RenderPage(ctx, templater_domain.RenderPageParams{})
			_ = m.RenderPartial(ctx, templater_domain.RenderPageParams{})
			_ = m.RenderEmail(ctx, templater_domain.RenderEmailParams{})
			_, _ = m.RenderASTToPlainText(ctx, nil)
			m.GetLastEmailAssetRequests()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.CollectMetadataCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RenderPageCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RenderPartialCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RenderEmailCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RenderASTToPlainTextCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetLastEmailAssetRequestsCallCount))
}

func TestMockRendererPort_CollectMetadata(t *testing.T) {
	t.Parallel()

	t.Run("nil CollectMetadataFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockRendererPort{}
		ctx := context.Background()
		links, pd, err := m.CollectMetadata(ctx, httptest.NewRequest(http.MethodGet, "/", nil), &templater_dto.InternalMetadata{}, &config.WebsiteConfig{})
		assert.Nil(t, links)
		assert.Nil(t, pd)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CollectMetadataCallCount))
	})

	t.Run("delegates to CollectMetadataFunc", func(t *testing.T) {
		t.Parallel()
		expected := []render_dto.LinkHeader{{URL: "/style.css", Rel: "preload"}}
		m := &templater_domain.MockRendererPort{
			CollectMetadataFunc: func(ctx context.Context, request *http.Request, meta *templater_dto.InternalMetadata, websiteConfig *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
				require.NotNil(t, ctx)
				require.NotNil(t, request)
				return expected, nil, nil
			},
		}
		ctx := context.Background()
		links, _, err := m.CollectMetadata(ctx, httptest.NewRequest(http.MethodGet, "/", nil), &templater_dto.InternalMetadata{}, &config.WebsiteConfig{})
		assert.Equal(t, expected, links)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CollectMetadataCallCount))
	})

	t.Run("propagates error from CollectMetadataFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("metadata collection failed")
		m := &templater_domain.MockRendererPort{
			CollectMetadataFunc: func(_ context.Context, _ *http.Request, _ *templater_dto.InternalMetadata, _ *config.WebsiteConfig) ([]render_dto.LinkHeader, *render_dto.ProbeData, error) {
				return nil, nil, expectedErr
			},
		}
		ctx := context.Background()
		links, _, err := m.CollectMetadata(ctx, httptest.NewRequest(http.MethodGet, "/", nil), &templater_dto.InternalMetadata{}, &config.WebsiteConfig{})
		assert.Nil(t, links)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.CollectMetadataCallCount))
	})
}

func TestMockRendererPort_RenderPage(t *testing.T) {
	t.Parallel()

	t.Run("nil RenderPageFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockRendererPort{}
		ctx := context.Background()
		err := m.RenderPage(ctx, templater_domain.RenderPageParams{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPageCallCount))
	})

	t.Run("delegates to RenderPageFunc", func(t *testing.T) {
		t.Parallel()
		var capturedParams templater_domain.RenderPageParams
		m := &templater_domain.MockRendererPort{
			RenderPageFunc: func(ctx context.Context, params templater_domain.RenderPageParams) error {
				require.NotNil(t, ctx)
				capturedParams = params
				return nil
			},
		}
		ctx := context.Background()
		params := templater_domain.RenderPageParams{Styling: "body{}", IsFragment: true}
		err := m.RenderPage(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, "body{}", capturedParams.Styling)
		assert.True(t, capturedParams.IsFragment)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPageCallCount))
	})

	t.Run("propagates error from RenderPageFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("render page failed")
		m := &templater_domain.MockRendererPort{
			RenderPageFunc: func(_ context.Context, _ templater_domain.RenderPageParams) error {
				return expectedErr
			},
		}
		ctx := context.Background()
		err := m.RenderPage(ctx, templater_domain.RenderPageParams{})
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPageCallCount))
	})
}

func TestMockRendererPort_RenderPartial(t *testing.T) {
	t.Parallel()

	t.Run("nil RenderPartialFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockRendererPort{}
		ctx := context.Background()
		err := m.RenderPartial(ctx, templater_domain.RenderPageParams{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPartialCallCount))
	})

	t.Run("delegates to RenderPartialFunc", func(t *testing.T) {
		t.Parallel()
		var capturedParams templater_domain.RenderPageParams
		m := &templater_domain.MockRendererPort{
			RenderPartialFunc: func(ctx context.Context, params templater_domain.RenderPageParams) error {
				require.NotNil(t, ctx)
				capturedParams = params
				return nil
			},
		}
		ctx := context.Background()
		params := templater_domain.RenderPageParams{Styling: "div{}", IsFragment: false}
		err := m.RenderPartial(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, "div{}", capturedParams.Styling)
		assert.False(t, capturedParams.IsFragment)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPartialCallCount))
	})

	t.Run("propagates error from RenderPartialFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("render partial failed")
		m := &templater_domain.MockRendererPort{
			RenderPartialFunc: func(_ context.Context, _ templater_domain.RenderPageParams) error {
				return expectedErr
			},
		}
		ctx := context.Background()
		err := m.RenderPartial(ctx, templater_domain.RenderPageParams{})
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderPartialCallCount))
	})
}

func TestMockRendererPort_RenderEmail(t *testing.T) {
	t.Parallel()

	t.Run("nil RenderEmailFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockRendererPort{}
		ctx := context.Background()
		err := m.RenderEmail(ctx, templater_domain.RenderEmailParams{})
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderEmailCallCount))
	})

	t.Run("delegates to RenderEmailFunc", func(t *testing.T) {
		t.Parallel()
		var capturedParams templater_domain.RenderEmailParams
		m := &templater_domain.MockRendererPort{
			RenderEmailFunc: func(ctx context.Context, params templater_domain.RenderEmailParams) error {
				require.NotNil(t, ctx)
				capturedParams = params
				return nil
			},
		}
		ctx := context.Background()
		params := templater_domain.RenderEmailParams{PageID: "welcome-email", Styling: "table{}"}
		err := m.RenderEmail(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, "welcome-email", capturedParams.PageID)
		assert.Equal(t, "table{}", capturedParams.Styling)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderEmailCallCount))
	})

	t.Run("propagates error from RenderEmailFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("render email failed")
		m := &templater_domain.MockRendererPort{
			RenderEmailFunc: func(_ context.Context, _ templater_domain.RenderEmailParams) error {
				return expectedErr
			},
		}
		ctx := context.Background()
		err := m.RenderEmail(ctx, templater_domain.RenderEmailParams{})
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderEmailCallCount))
	})
}

func TestMockRendererPort_RenderASTToPlainText(t *testing.T) {
	t.Parallel()

	t.Run("nil RenderASTToPlainTextFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockRendererPort{}
		ctx := context.Background()
		text, err := m.RenderASTToPlainText(ctx, nil)
		assert.Empty(t, text)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderASTToPlainTextCallCount))
	})

	t.Run("delegates to RenderASTToPlainTextFunc", func(t *testing.T) {
		t.Parallel()
		ast := &ast_domain.TemplateAST{}
		m := &templater_domain.MockRendererPort{
			RenderASTToPlainTextFunc: func(ctx context.Context, templateAST *ast_domain.TemplateAST) (string, error) {
				require.NotNil(t, ctx)
				require.Same(t, ast, templateAST)
				return "plain text content", nil
			},
		}
		ctx := context.Background()
		text, err := m.RenderASTToPlainText(ctx, ast)
		assert.Equal(t, "plain text content", text)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderASTToPlainTextCallCount))
	})

	t.Run("propagates error from RenderASTToPlainTextFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("plain text rendering failed")
		m := &templater_domain.MockRendererPort{
			RenderASTToPlainTextFunc: func(_ context.Context, _ *ast_domain.TemplateAST) (string, error) {
				return "", expectedErr
			},
		}
		ctx := context.Background()
		text, err := m.RenderASTToPlainText(ctx, nil)
		assert.Empty(t, text)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RenderASTToPlainTextCallCount))
	})
}

func TestMockRendererPort_GetLastEmailAssetRequests(t *testing.T) {
	t.Parallel()

	t.Run("nil GetLastEmailAssetRequestsFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockRendererPort{}
		got := m.GetLastEmailAssetRequests()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetLastEmailAssetRequestsCallCount))
	})

	t.Run("delegates to GetLastEmailAssetRequestsFunc", func(t *testing.T) {
		t.Parallel()
		expected := []*email_dto.EmailAssetRequest{
			{SourcePath: "assets/images/logo.png"},
		}
		m := &templater_domain.MockRendererPort{
			GetLastEmailAssetRequestsFunc: func() []*email_dto.EmailAssetRequest {
				return expected
			},
		}
		got := m.GetLastEmailAssetRequests()
		require.Len(t, got, 1)
		assert.Equal(t, "assets/images/logo.png", got[0].SourcePath)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetLastEmailAssetRequestsCallCount))
	})
}
