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
	"net/http"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/i18n/i18n_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestMockPageEntryView_ZeroValueIsUsable(t *testing.T) {
	t.Parallel()

	var m templater_domain.MockPageEntryView

	assert.False(t, m.GetHasMiddleware())
	assert.Empty(t, m.GetMiddlewareFuncName())
	assert.False(t, m.GetHasCachePolicy())
	assert.Equal(t, templater_dto.CachePolicy{}, m.GetCachePolicy(nil))
	assert.Empty(t, m.GetCachePolicyFuncName())
	assert.Nil(t, m.GetMiddlewares())
	assert.False(t, m.GetIsPage())
	assert.Empty(t, m.GetRoutePattern())
	assert.Nil(t, m.GetRoutePatterns())
	assert.Empty(t, m.GetI18nStrategy())
	assert.Empty(t, m.GetOriginalPath())

	astRoot, meta := m.GetASTRoot(nil)
	assert.Nil(t, astRoot)
	assert.Equal(t, templater_dto.InternalMetadata{}, meta)

	astRoot2, meta2 := m.GetASTRootWithProps(nil, nil)
	assert.Nil(t, astRoot2)
	assert.Equal(t, templater_dto.InternalMetadata{}, meta2)

	assert.Empty(t, m.GetStyling())
	assert.Nil(t, m.GetAssetRefs())
	assert.Nil(t, m.GetCustomTags())
	assert.Nil(t, m.GetSupportedLocales())
	assert.Nil(t, m.GetLocalStore())
	assert.Nil(t, m.GetJSScriptMetas())
	assert.False(t, m.GetIsE2EOnly())
	assert.Nil(t, m.GetStaticMetadata())
}

func TestMockPageEntryView_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	m := &templater_domain.MockPageEntryView{
		GetHasMiddlewareFunc:       func() bool { return true },
		GetMiddlewareFuncNameFunc:  func() string { return "mw" },
		GetHasCachePolicyFunc:      func() bool { return true },
		GetCachePolicyFunc:         func(_ *templater_dto.RequestData) templater_dto.CachePolicy { return templater_dto.CachePolicy{} },
		GetCachePolicyFuncNameFunc: func() string { return "cp" },
		GetMiddlewaresFunc:         func() []func(http.Handler) http.Handler { return nil },
		GetIsPageFunc:              func() bool { return true },
		GetRoutePatternFunc:        func() string { return "/" },
		GetRoutePatternsFunc:       func() map[string]string { return map[string]string{"GET": "/"} },
		GetI18nStrategyFunc:        func() string { return "prefix" },
		GetOriginalPathFunc:        func() string { return "pages/home.pk" },
		GetASTRootFunc: func(_ *templater_dto.RequestData) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
			return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}
		},
		GetASTRootWithPropsFunc: func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
			return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}
		},
		GetStylingFunc:          func() string { return "css" },
		GetAssetRefsFunc:        func() []templater_dto.AssetRef { return nil },
		GetCustomTagsFunc:       func() []string { return nil },
		GetSupportedLocalesFunc: func() []string { return nil },
		GetLocalStoreFunc:       func() *i18n_domain.Store { return nil },
		GetJSScriptMetasFunc:    func() []templater_dto.JSScriptMeta { return nil },
		GetIsE2EOnlyFunc:        func() bool { return false },
		GetStaticMetadataFunc:   func() *templater_dto.InternalMetadata { return nil },
	}

	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			m.GetHasMiddleware()
			m.GetMiddlewareFuncName()
			m.GetHasCachePolicy()
			m.GetCachePolicy(nil)
			m.GetCachePolicyFuncName()
			m.GetMiddlewares()
			m.GetIsPage()
			m.GetRoutePattern()
			m.GetRoutePatterns()
			m.GetI18nStrategy()
			m.GetOriginalPath()
			m.GetASTRoot(nil)
			m.GetASTRootWithProps(nil, nil)
			m.GetStyling()
			m.GetAssetRefs()
			m.GetCustomTags()
			m.GetSupportedLocales()
			m.GetLocalStore()
			m.GetJSScriptMetas()
			m.GetIsE2EOnly()
			m.GetStaticMetadata()
		}()
	}

	wg.Wait()

	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetHasMiddlewareCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetMiddlewareFuncNameCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetHasCachePolicyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetCachePolicyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetCachePolicyFuncNameCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetMiddlewaresCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetIsPageCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetRoutePatternCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetRoutePatternsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetI18nStrategyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetOriginalPathCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetASTRootCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetASTRootWithPropsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetStylingCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetAssetRefsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetCustomTagsCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetSupportedLocalesCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetLocalStoreCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetJSScriptMetasCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetIsE2EOnlyCallCount))
	assert.Equal(t, int64(goroutines), atomic.LoadInt64(&m.GetStaticMetadataCallCount))
}

func TestMockPageEntryView_GetHasMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("nil GetHasMiddlewareFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetHasMiddleware()
		assert.False(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetHasMiddlewareCallCount))
	})

	t.Run("delegates to GetHasMiddlewareFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetHasMiddlewareFunc: func() bool { return true },
		}
		got := m.GetHasMiddleware()
		assert.True(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetHasMiddlewareCallCount))
	})
}

func TestMockPageEntryView_GetMiddlewareFuncName(t *testing.T) {
	t.Parallel()

	t.Run("nil GetMiddlewareFuncNameFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetMiddlewareFuncName()
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMiddlewareFuncNameCallCount))
	})

	t.Run("delegates to GetMiddlewareFuncNameFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetMiddlewareFuncNameFunc: func() string { return "authMiddleware" },
		}
		got := m.GetMiddlewareFuncName()
		assert.Equal(t, "authMiddleware", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMiddlewareFuncNameCallCount))
	})
}

func TestMockPageEntryView_GetHasCachePolicy(t *testing.T) {
	t.Parallel()

	t.Run("nil GetHasCachePolicyFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetHasCachePolicy()
		assert.False(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetHasCachePolicyCallCount))
	})

	t.Run("delegates to GetHasCachePolicyFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetHasCachePolicyFunc: func() bool { return true },
		}
		got := m.GetHasCachePolicy()
		assert.True(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetHasCachePolicyCallCount))
	})
}

func TestMockPageEntryView_GetCachePolicy(t *testing.T) {
	t.Parallel()

	t.Run("nil GetCachePolicyFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetCachePolicy(nil)
		assert.Equal(t, templater_dto.CachePolicy{}, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCachePolicyCallCount))
	})

	t.Run("delegates to GetCachePolicyFunc", func(t *testing.T) {
		t.Parallel()
		expected := templater_dto.CachePolicy{Key: "user-123"}
		m := &templater_domain.MockPageEntryView{
			GetCachePolicyFunc: func(r *templater_dto.RequestData) templater_dto.CachePolicy {
				require.NotNil(t, r)
				return expected
			},
		}
		rd := NewTestRequestData()
		got := m.GetCachePolicy(rd)
		assert.Equal(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCachePolicyCallCount))
	})
}

func TestMockPageEntryView_GetCachePolicyFuncName(t *testing.T) {
	t.Parallel()

	t.Run("nil GetCachePolicyFuncNameFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetCachePolicyFuncName()
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCachePolicyFuncNameCallCount))
	})

	t.Run("delegates to GetCachePolicyFuncNameFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetCachePolicyFuncNameFunc: func() string { return "UserCachePolicy" },
		}
		got := m.GetCachePolicyFuncName()
		assert.Equal(t, "UserCachePolicy", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCachePolicyFuncNameCallCount))
	})
}

func TestMockPageEntryView_GetMiddlewares(t *testing.T) {
	t.Parallel()

	t.Run("nil GetMiddlewaresFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetMiddlewares()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMiddlewaresCallCount))
	})

	t.Run("delegates to GetMiddlewaresFunc", func(t *testing.T) {
		t.Parallel()
		mw := func(next http.Handler) http.Handler { return next }
		m := &templater_domain.MockPageEntryView{
			GetMiddlewaresFunc: func() []func(http.Handler) http.Handler {
				return []func(http.Handler) http.Handler{mw}
			},
		}
		got := m.GetMiddlewares()
		require.Len(t, got, 1)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetMiddlewaresCallCount))
	})
}

func TestMockPageEntryView_GetIsPage(t *testing.T) {
	t.Parallel()

	t.Run("nil GetIsPageFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetIsPage()
		assert.False(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetIsPageCallCount))
	})

	t.Run("delegates to GetIsPageFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetIsPageFunc: func() bool { return true },
		}
		got := m.GetIsPage()
		assert.True(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetIsPageCallCount))
	})
}

func TestMockPageEntryView_GetRoutePattern(t *testing.T) {
	t.Parallel()

	t.Run("nil GetRoutePatternFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetRoutePattern()
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetRoutePatternCallCount))
	})

	t.Run("delegates to GetRoutePatternFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetRoutePatternFunc: func() string { return "/home" },
		}
		got := m.GetRoutePattern()
		assert.Equal(t, "/home", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetRoutePatternCallCount))
	})
}

func TestMockPageEntryView_GetRoutePatterns(t *testing.T) {
	t.Parallel()

	t.Run("nil GetRoutePatternsFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetRoutePatterns()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetRoutePatternsCallCount))
	})

	t.Run("delegates to GetRoutePatternsFunc", func(t *testing.T) {
		t.Parallel()
		expected := map[string]string{"GET": "/home", "POST": "/home"}
		m := &templater_domain.MockPageEntryView{
			GetRoutePatternsFunc: func() map[string]string { return expected },
		}
		got := m.GetRoutePatterns()
		assert.Equal(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetRoutePatternsCallCount))
	})
}

func TestMockPageEntryView_GetI18nStrategy(t *testing.T) {
	t.Parallel()

	t.Run("nil GetI18nStrategyFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetI18nStrategy()
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetI18nStrategyCallCount))
	})

	t.Run("delegates to GetI18nStrategyFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetI18nStrategyFunc: func() string { return "prefix" },
		}
		got := m.GetI18nStrategy()
		assert.Equal(t, "prefix", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetI18nStrategyCallCount))
	})
}

func TestMockPageEntryView_GetOriginalPath(t *testing.T) {
	t.Parallel()

	t.Run("nil GetOriginalPathFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetOriginalPath()
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetOriginalPathCallCount))
	})

	t.Run("delegates to GetOriginalPathFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetOriginalPathFunc: func() string { return "pages/home.pk" },
		}
		got := m.GetOriginalPath()
		assert.Equal(t, "pages/home.pk", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetOriginalPathCallCount))
	})
}

func TestMockPageEntryView_GetASTRoot(t *testing.T) {
	t.Parallel()

	t.Run("nil GetASTRootFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		astRoot, meta := m.GetASTRoot(nil)
		assert.Nil(t, astRoot)
		assert.Equal(t, templater_dto.InternalMetadata{}, meta)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetASTRootCallCount))
	})

	t.Run("delegates to GetASTRootFunc", func(t *testing.T) {
		t.Parallel()
		expectedAST := &ast_domain.TemplateAST{}
		expectedMeta := templater_dto.InternalMetadata{CustomTags: []string{"x-tag"}}
		m := &templater_domain.MockPageEntryView{
			GetASTRootFunc: func(r *templater_dto.RequestData) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
				require.NotNil(t, r)
				return expectedAST, expectedMeta
			},
		}
		rd := NewTestRequestData()
		astRoot, meta := m.GetASTRoot(rd)
		assert.Same(t, expectedAST, astRoot)
		assert.Equal(t, expectedMeta, meta)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetASTRootCallCount))
	})
}

func TestMockPageEntryView_GetASTRootWithProps(t *testing.T) {
	t.Parallel()

	t.Run("nil GetASTRootWithPropsFunc returns zero values", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		astRoot, meta := m.GetASTRootWithProps(nil, nil)
		assert.Nil(t, astRoot)
		assert.Equal(t, templater_dto.InternalMetadata{}, meta)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetASTRootWithPropsCallCount))
	})

	t.Run("delegates to GetASTRootWithPropsFunc", func(t *testing.T) {
		t.Parallel()
		expectedAST := &ast_domain.TemplateAST{}
		expectedMeta := templater_dto.InternalMetadata{CustomTags: []string{"x-email"}}
		props := map[string]string{"name": "test"}
		m := &templater_domain.MockPageEntryView{
			GetASTRootWithPropsFunc: func(r *templater_dto.RequestData, p any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata) {
				require.NotNil(t, r)
				assert.Equal(t, props, p)
				return expectedAST, expectedMeta
			},
		}
		rd := NewTestRequestData()
		astRoot, meta := m.GetASTRootWithProps(rd, props)
		assert.Same(t, expectedAST, astRoot)
		assert.Equal(t, expectedMeta, meta)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetASTRootWithPropsCallCount))
	})
}

func TestMockPageEntryView_GetStyling(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStylingFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetStyling()
		assert.Empty(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStylingCallCount))
	})

	t.Run("delegates to GetStylingFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetStylingFunc: func() string { return "body { colour: red; }" },
		}
		got := m.GetStyling()
		assert.Equal(t, "body { colour: red; }", got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStylingCallCount))
	})
}

func TestMockPageEntryView_GetAssetRefs(t *testing.T) {
	t.Parallel()

	t.Run("nil GetAssetRefsFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetAssetRefs()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetAssetRefsCallCount))
	})

	t.Run("delegates to GetAssetRefsFunc", func(t *testing.T) {
		t.Parallel()
		expected := []templater_dto.AssetRef{{Kind: "svg", Path: "/icons/logo.svg"}}
		m := &templater_domain.MockPageEntryView{
			GetAssetRefsFunc: func() []templater_dto.AssetRef { return expected },
		}
		got := m.GetAssetRefs()
		assert.Equal(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetAssetRefsCallCount))
	})
}

func TestMockPageEntryView_GetCustomTags(t *testing.T) {
	t.Parallel()

	t.Run("nil GetCustomTagsFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetCustomTags()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCustomTagsCallCount))
	})

	t.Run("delegates to GetCustomTagsFunc", func(t *testing.T) {
		t.Parallel()
		expected := []string{"x-header", "x-footer"}
		m := &templater_domain.MockPageEntryView{
			GetCustomTagsFunc: func() []string { return expected },
		}
		got := m.GetCustomTags()
		assert.Equal(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetCustomTagsCallCount))
	})
}

func TestMockPageEntryView_GetSupportedLocales(t *testing.T) {
	t.Parallel()

	t.Run("nil GetSupportedLocalesFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetSupportedLocales()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetSupportedLocalesCallCount))
	})

	t.Run("delegates to GetSupportedLocalesFunc", func(t *testing.T) {
		t.Parallel()
		expected := []string{"en_GB", "de_DE"}
		m := &templater_domain.MockPageEntryView{
			GetSupportedLocalesFunc: func() []string { return expected },
		}
		got := m.GetSupportedLocales()
		assert.Equal(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetSupportedLocalesCallCount))
	})
}

func TestMockPageEntryView_GetLocalStore(t *testing.T) {
	t.Parallel()

	t.Run("nil GetLocalStoreFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetLocalStore()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetLocalStoreCallCount))
	})

	t.Run("delegates to GetLocalStoreFunc", func(t *testing.T) {
		t.Parallel()
		expected := &i18n_domain.Store{}
		m := &templater_domain.MockPageEntryView{
			GetLocalStoreFunc: func() *i18n_domain.Store { return expected },
		}
		got := m.GetLocalStore()
		assert.Same(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetLocalStoreCallCount))
	})
}

func TestMockPageEntryView_GetJSScriptMetas(t *testing.T) {
	t.Parallel()

	t.Run("nil GetJSScriptMetasFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetJSScriptMetas()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetJSScriptMetasCallCount))
	})

	t.Run("delegates to GetJSScriptMetasFunc", func(t *testing.T) {
		t.Parallel()
		expected := []templater_dto.JSScriptMeta{{URL: "/_piko/assets/pk-js/main.js"}}
		m := &templater_domain.MockPageEntryView{
			GetJSScriptMetasFunc: func() []templater_dto.JSScriptMeta { return expected },
		}
		got := m.GetJSScriptMetas()
		assert.Equal(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetJSScriptMetasCallCount))
	})
}

func TestMockPageEntryView_GetIsE2EOnly(t *testing.T) {
	t.Parallel()

	t.Run("nil GetIsE2EOnlyFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetIsE2EOnly()
		assert.False(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetIsE2EOnlyCallCount))
	})

	t.Run("delegates to GetIsE2EOnlyFunc", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{
			GetIsE2EOnlyFunc: func() bool { return true },
		}
		got := m.GetIsE2EOnly()
		assert.True(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetIsE2EOnlyCallCount))
	})
}

func TestMockPageEntryView_GetStaticMetadata(t *testing.T) {
	t.Parallel()

	t.Run("nil GetStaticMetadataFunc returns zero value", func(t *testing.T) {
		t.Parallel()
		m := &templater_domain.MockPageEntryView{}
		got := m.GetStaticMetadata()
		assert.Nil(t, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStaticMetadataCallCount))
	})

	t.Run("delegates to GetStaticMetadataFunc", func(t *testing.T) {
		t.Parallel()
		expected := &templater_dto.InternalMetadata{CustomTags: []string{"x-header"}}
		m := &templater_domain.MockPageEntryView{
			GetStaticMetadataFunc: func() *templater_dto.InternalMetadata { return expected },
		}
		got := m.GetStaticMetadata()
		assert.Same(t, expected, got)
		assert.Equal(t, int64(1), atomic.LoadInt64(&m.GetStaticMetadataCallCount))
	})
}
