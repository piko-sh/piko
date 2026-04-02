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
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestMockManifestRunnerPort_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m templater_domain.MockManifestRunnerPort

	ctx := context.Background()
	page := templater_dto.PageDefinition{}
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	astResult, meta, styling, err := m.RunPage(ctx, page, request)
	assert.Nil(t, astResult)
	assert.Equal(t, templater_dto.InternalMetadata{}, meta)
	assert.Empty(t, styling)
	assert.NoError(t, err)

	astResult2, meta2, styling2, err2 := m.RunPartial(ctx, page, request)
	assert.Nil(t, astResult2)
	assert.Equal(t, templater_dto.InternalMetadata{}, meta2)
	assert.Empty(t, styling2)
	assert.NoError(t, err2)

	astResult3, meta3, styling3, err3 := m.RunPartialWithProps(ctx, page, request, nil)
	assert.Nil(t, astResult3)
	assert.Equal(t, templater_dto.InternalMetadata{}, meta3)
	assert.Empty(t, styling3)
	assert.NoError(t, err3)

	entry, err4 := m.GetPageEntry(ctx, "pages/home.pk")
	assert.Nil(t, entry)
	assert.NoError(t, err4)
}

func TestMockManifestRunnerPort_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &templater_domain.MockManifestRunnerPort{
		RunPageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
			return nil, templater_dto.InternalMetadata{}, "", nil
		},
		RunPartialFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
			return nil, templater_dto.InternalMetadata{}, "", nil
		},
		RunPartialWithPropsFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
			return nil, templater_dto.InternalMetadata{}, "", nil
		},
		GetPageEntryFunc: func(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
			return nil, nil
		},
	}

	const goroutines = 50

	ctx := context.Background()
	page := templater_dto.PageDefinition{}
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			_, _, _, _ = m.RunPage(ctx, page, request)
			_, _, _, _ = m.RunPartial(ctx, page, request)
			_, _, _, _ = m.RunPartialWithProps(ctx, page, request, nil)
			_, _ = m.GetPageEntry(ctx, "pages/home.pk")
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RunPageCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RunPartialCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.RunPartialWithPropsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetPageEntryCallCount))
}

func TestMockManifestRunnerPort_RunPage(t *testing.T) {
	t.Parallel()

	t.Run("nil RunPageFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockManifestRunnerPort{}
		ctx := context.Background()
		astResult, meta, styling, err := m.RunPage(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Nil(t, astResult)
		assert.Equal(t, templater_dto.InternalMetadata{}, meta)
		assert.Empty(t, styling)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPageCallCount))
	})

	t.Run("delegates to RunPageFunc", func(t *testing.T) {
		t.Parallel()
		expectedAST := &ast_domain.TemplateAST{}
		expectedMeta := templater_dto.InternalMetadata{CustomTags: []string{"x-tag"}}
		m := &templater_domain.MockManifestRunnerPort{
			RunPageFunc: func(ctx context.Context, pd templater_dto.PageDefinition, r *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
				require.NotNil(t, ctx)
				return expectedAST, expectedMeta, "body{}", nil
			},
		}
		ctx := context.Background()
		astResult, meta, styling, err := m.RunPage(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Same(t, expectedAST, astResult)
		assert.Equal(t, expectedMeta, meta)
		assert.Equal(t, "body{}", styling)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPageCallCount))
	})

	t.Run("propagates error from RunPageFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("run page failed")
		m := &templater_domain.MockManifestRunnerPort{
			RunPageFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
				return nil, templater_dto.InternalMetadata{}, "", expectedErr
			},
		}
		ctx := context.Background()
		astResult, _, _, err := m.RunPage(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Nil(t, astResult)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPageCallCount))
	})
}

func TestMockManifestRunnerPort_RunPartial(t *testing.T) {
	t.Parallel()

	t.Run("nil RunPartialFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockManifestRunnerPort{}
		ctx := context.Background()
		astResult, meta, styling, err := m.RunPartial(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Nil(t, astResult)
		assert.Equal(t, templater_dto.InternalMetadata{}, meta)
		assert.Empty(t, styling)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPartialCallCount))
	})

	t.Run("delegates to RunPartialFunc", func(t *testing.T) {
		t.Parallel()
		expectedAST := &ast_domain.TemplateAST{}
		expectedMeta := templater_dto.InternalMetadata{CustomTags: []string{"x-partial"}}
		m := &templater_domain.MockManifestRunnerPort{
			RunPartialFunc: func(ctx context.Context, pd templater_dto.PageDefinition, r *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
				require.NotNil(t, ctx)
				return expectedAST, expectedMeta, "div{}", nil
			},
		}
		ctx := context.Background()
		astResult, meta, styling, err := m.RunPartial(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Same(t, expectedAST, astResult)
		assert.Equal(t, expectedMeta, meta)
		assert.Equal(t, "div{}", styling)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPartialCallCount))
	})

	t.Run("propagates error from RunPartialFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("run partial failed")
		m := &templater_domain.MockManifestRunnerPort{
			RunPartialFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
				return nil, templater_dto.InternalMetadata{}, "", expectedErr
			},
		}
		ctx := context.Background()
		astResult, _, _, err := m.RunPartial(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil))
		assert.Nil(t, astResult)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPartialCallCount))
	})
}

func TestMockManifestRunnerPort_RunPartialWithProps(t *testing.T) {
	t.Parallel()

	t.Run("nil RunPartialWithPropsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockManifestRunnerPort{}
		ctx := context.Background()
		astResult, meta, styling, err := m.RunPartialWithProps(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), nil)
		assert.Nil(t, astResult)
		assert.Equal(t, templater_dto.InternalMetadata{}, meta)
		assert.Empty(t, styling)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPartialWithPropsCallCount))
	})

	t.Run("delegates to RunPartialWithPropsFunc", func(t *testing.T) {
		t.Parallel()
		expectedAST := &ast_domain.TemplateAST{}
		expectedMeta := templater_dto.InternalMetadata{CustomTags: []string{"x-email"}}
		props := map[string]string{"name": "test"}
		m := &templater_domain.MockManifestRunnerPort{
			RunPartialWithPropsFunc: func(ctx context.Context, pd templater_dto.PageDefinition, r *http.Request, p any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
				require.NotNil(t, ctx)
				assert.Equal(t, props, p)
				return expectedAST, expectedMeta, "span{}", nil
			},
		}
		ctx := context.Background()
		astResult, meta, styling, err := m.RunPartialWithProps(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), props)
		assert.Same(t, expectedAST, astResult)
		assert.Equal(t, expectedMeta, meta)
		assert.Equal(t, "span{}", styling)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPartialWithPropsCallCount))
	})

	t.Run("propagates error from RunPartialWithPropsFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("run partial with props failed")
		m := &templater_domain.MockManifestRunnerPort{
			RunPartialWithPropsFunc: func(_ context.Context, _ templater_dto.PageDefinition, _ *http.Request, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
				return nil, templater_dto.InternalMetadata{}, "", expectedErr
			},
		}
		ctx := context.Background()
		astResult, _, _, err := m.RunPartialWithProps(ctx, templater_dto.PageDefinition{}, httptest.NewRequest(http.MethodGet, "/", nil), nil)
		assert.Nil(t, astResult)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.RunPartialWithPropsCallCount))
	})
}

func TestMockManifestRunnerPort_GetPageEntry(t *testing.T) {
	t.Parallel()

	t.Run("nil GetPageEntryFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockManifestRunnerPort{}
		ctx := context.Background()
		entry, err := m.GetPageEntry(ctx, "pages/home.pk")
		assert.Nil(t, entry)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetPageEntryCallCount))
	})

	t.Run("delegates to GetPageEntryFunc", func(t *testing.T) {
		t.Parallel()
		expectedEntry := &templater_domain.MockPageEntryView{
			GetOriginalPathFunc: func() string { return "pages/home.pk" },
		}
		m := &templater_domain.MockManifestRunnerPort{
			GetPageEntryFunc: func(ctx context.Context, key string) (templater_domain.PageEntryView, error) {
				require.NotNil(t, ctx)
				require.Equal(t, "pages/home.pk", key)
				return expectedEntry, nil
			},
		}
		ctx := context.Background()
		entry, err := m.GetPageEntry(ctx, "pages/home.pk")
		assert.NoError(t, err)
		assert.Equal(t, "pages/home.pk", entry.GetOriginalPath())
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetPageEntryCallCount))
	})

	t.Run("propagates error from GetPageEntryFunc", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("page entry not found")
		m := &templater_domain.MockManifestRunnerPort{
			GetPageEntryFunc: func(_ context.Context, _ string) (templater_domain.PageEntryView, error) {
				return nil, expectedErr
			},
		}
		ctx := context.Background()
		entry, err := m.GetPageEntry(ctx, "missing.pk")
		assert.Nil(t, entry)
		assert.ErrorIs(t, err, expectedErr)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetPageEntryCallCount))
	})
}
