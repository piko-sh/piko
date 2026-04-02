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

package templater_adapters

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestPageEntry_GetStyling(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.StyleBlock = ".my-class { color: red; }"
	assert.Equal(t, ".my-class { color: red; }", pe.GetStyling())
}

func TestPageEntry_GetStyling_Empty(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Equal(t, "", pe.GetStyling())
}

func TestPageEntry_GetAssetRefs(t *testing.T) {
	t.Parallel()

	refs := []templater_dto.AssetRef{
		{Kind: "svg", Path: "/icon.svg"},
		{Kind: "css", Path: "/style.css"},
	}
	pe := &PageEntry{}
	pe.AssetRefs = refs
	assert.Equal(t, refs, pe.GetAssetRefs())
}

func TestPageEntry_GetAssetRefs_Nil(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Nil(t, pe.GetAssetRefs())
}

func TestPageEntry_GetCustomTags(t *testing.T) {
	t.Parallel()

	tags := []string{"my-button", "my-card"}
	pe := &PageEntry{}
	pe.CustomTags = tags
	assert.Equal(t, tags, pe.GetCustomTags())
}

func TestPageEntry_GetCustomTags_Nil(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Nil(t, pe.GetCustomTags())
}

func TestPageEntry_GetHasCachePolicy(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.HasCachePolicy = true
		assert.True(t, pe.GetHasCachePolicy())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.HasCachePolicy = false
		assert.False(t, pe.GetHasCachePolicy())
	})
}

func TestPageEntry_GetHasMiddleware(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.HasMiddleware = true
		assert.True(t, pe.GetHasMiddleware())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		assert.False(t, pe.GetHasMiddleware())
	})
}

func TestPageEntry_GetHasSupportedLocales(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.HasSupportedLocales = true
		assert.True(t, pe.GetHasSupportedLocales())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		assert.False(t, pe.GetHasSupportedLocales())
	})
}

func TestPageEntry_GetOriginalPath(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.OriginalSourcePath = "pages/about.pk"
	assert.Equal(t, "pages/about.pk", pe.GetOriginalPath())
}

func TestPageEntry_GetRoutePattern(t *testing.T) {
	t.Parallel()

	t.Run("single pattern", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.RoutePatterns = map[string]string{"": "/about"}
		assert.Equal(t, "/about", pe.GetRoutePattern())
	})

	t.Run("empty patterns", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.RoutePatterns = map[string]string{}
		assert.Equal(t, "", pe.GetRoutePattern())
	})

	t.Run("nil patterns", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		assert.Equal(t, "", pe.GetRoutePattern())
	})
}

func TestPageEntry_GetRoutePatterns(t *testing.T) {
	t.Parallel()

	patterns := map[string]string{"en": "/about", "fr": "/a-propos"}
	pe := &PageEntry{}
	pe.RoutePatterns = patterns
	assert.Equal(t, patterns, pe.GetRoutePatterns())
}

func TestPageEntry_GetI18nStrategy(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.I18nStrategy = "prefix_except_default"
	assert.Equal(t, "prefix_except_default", pe.GetI18nStrategy())
}

func TestPageEntry_GetMiddlewareFuncName(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.MiddlewareFuncName = "AuthMiddleware"
	assert.Equal(t, "AuthMiddleware", pe.GetMiddlewareFuncName())
}

func TestPageEntry_GetCachePolicyFuncName(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.CachePolicyFuncName = "MyCachePolicy"
	assert.Equal(t, "MyCachePolicy", pe.GetCachePolicyFuncName())
}

func TestPageEntry_GetIsE2EOnly(t *testing.T) {
	t.Parallel()

	t.Run("true", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.IsE2EOnly = true
		assert.True(t, pe.GetIsE2EOnly())
	})

	t.Run("false", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		assert.False(t, pe.GetIsE2EOnly())
	})
}

func TestPageEntry_SetASTFunc(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Nil(t, pe.astFunc)

	registryFunction := func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return nil, templater_dto.InternalMetadata{}, nil
	}
	pe.SetASTFunc(registryFunction)
	assert.NotNil(t, pe.astFunc)
}

func TestPageEntry_SetCachePolicyFunc(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Nil(t, pe.cachePolicyFunc)

	registryFunction := func(_ *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{Enabled: true}
	}
	pe.SetCachePolicyFunc(registryFunction)
	assert.NotNil(t, pe.cachePolicyFunc)
	result := pe.GetCachePolicy(nil)
	assert.True(t, result.Enabled)
}

func TestPageEntry_SetMiddlewareFunc(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Nil(t, pe.middlewareFunc)

	registryFunction := func() []func(http.Handler) http.Handler {
		return []func(http.Handler) http.Handler{}
	}
	pe.SetMiddlewareFunc(registryFunction)
	assert.NotNil(t, pe.middlewareFunc)
	result := pe.GetMiddlewares()
	assert.Empty(t, result)
}

func TestPageEntry_SetSupportedLocalesFunc(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Nil(t, pe.supportedLocalesFunc)

	registryFunction := func() []string {
		return []string{"en", "fr"}
	}
	pe.SetSupportedLocalesFunc(registryFunction)
	assert.NotNil(t, pe.supportedLocalesFunc)
	result := pe.GetSupportedLocales()
	assert.Equal(t, []string{"en", "fr"}, result)
}

func TestPageEntry_GetIsPage(t *testing.T) {
	t.Parallel()

	t.Run("page route", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.RoutePatterns = map[string]string{"en": "/about"}
		assert.True(t, pe.GetIsPage())
	})

	t.Run("partial route", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.RoutePatterns = map[string]string{"": "/_piko/partial/card"}
		assert.False(t, pe.GetIsPage())
	})

	t.Run("empty route", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.RoutePatterns = map[string]string{"": ""}
		assert.False(t, pe.GetIsPage())
	})

	t.Run("no routes", func(t *testing.T) {
		t.Parallel()
		pe := &PageEntry{}
		pe.RoutePatterns = map[string]string{}
		assert.False(t, pe.GetIsPage())
	})
}

func TestPageEntry_GetJSScriptMetas_Empty(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	assert.Nil(t, pe.GetJSScriptMetas())
}

func TestPageEntry_InitCachedJSScriptMetas(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{
		jsArtefactToPartialName: map[string]string{
			"js-partial-1": "my-modal",
		},
	}
	pe.JSArtefactIDs = []string{"js-page-1", "js-partial-1"}
	pe.initialiseCachedJSScriptMetas()

	metas := pe.GetJSScriptMetas()
	require.Len(t, metas, 2)
	assert.Equal(t, "/_piko/assets/js-page-1", metas[0].URL)
	assert.Equal(t, "", metas[0].PartialName)
	assert.Equal(t, "/_piko/assets/js-partial-1", metas[1].URL)
	assert.Equal(t, "my-modal", metas[1].PartialName)
}

func TestPageEntry_InitCachedJSScriptMetas_NoIDs(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.initialiseCachedJSScriptMetas()
	assert.Nil(t, pe.GetJSScriptMetas())
}

func TestPageEntry_GetStaticMetadata(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{
		supportedLocalesFunc: func() []string { return []string{"en"} },
	}
	pe.AssetRefs = []templater_dto.AssetRef{{Kind: "svg", Path: "/icon.svg"}}
	pe.CustomTags = []string{"my-tag"}
	pe.initialiseCachedStaticMetadata()

	meta := pe.GetStaticMetadata()
	require.NotNil(t, meta)
	assert.Equal(t, []templater_dto.AssetRef{{Kind: "svg", Path: "/icon.svg"}}, meta.AssetRefs)
	assert.Equal(t, []string{"my-tag"}, meta.CustomTags)
	assert.Equal(t, []string{"en"}, meta.SupportedLocales)
}

func TestPageEntry_GetASTRoot_NilFunc(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.OriginalSourcePath = "pages/broken.pk"
	pe.PackagePath = "mymod/pages/broken"

	ast, meta := pe.GetASTRoot(nil)
	require.NotNil(t, ast)
	assert.Len(t, ast.RootNodes, 1)
	assert.Contains(t, ast.RootNodes[0].InnerHTML, "pages/broken.pk")
	assert.Equal(t, templater_dto.InternalMetadata{}, meta)
}

func TestPageEntry_GetASTRootWithProps_NilFunc(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.OriginalSourcePath = "partials/missing.pk"
	pe.PackagePath = "mymod/partials/missing"

	ast, meta := pe.GetASTRootWithProps(nil, nil)
	require.NotNil(t, ast)
	assert.Len(t, ast.RootNodes, 1)
	assert.Contains(t, ast.RootNodes[0].InnerHTML, "partials/missing.pk")
	assert.Equal(t, templater_dto.InternalMetadata{}, meta)
}

func TestPageEntry_GetASTRoot_WithFunc(t *testing.T) {
	t.Parallel()

	expectedAST := &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{NodeType: ast_domain.NodeElement, TagName: "div"},
		},
	}
	expectedMeta := templater_dto.InternalMetadata{
		CustomTags: []string{"test"},
	}

	pe := &PageEntry{}
	pe.SetASTFunc(func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return expectedAST, expectedMeta, nil
	})

	ast, meta := pe.GetASTRoot(nil)
	assert.Equal(t, expectedAST, ast)
	assert.Equal(t, expectedMeta, meta)
}

func TestPageEntry_GetASTRootWithProps_WithFunc(t *testing.T) {
	t.Parallel()

	var capturedProps any
	pe := &PageEntry{}
	pe.SetASTFunc(func(_ *templater_dto.RequestData, props any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		capturedProps = props
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	})

	myProps := map[string]string{"key": "value"}
	_, _ = pe.GetASTRootWithProps(nil, myProps)
	assert.Equal(t, myProps, capturedProps)
}

func TestPageEntry_LinkFuncs_WithIsolatedRegistry(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/pages/home"

	registry.RegisterASTFunc(packagePath, func(_ *templater_dto.RequestData, _ any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	})

	pe := &PageEntry{
		registry:             registry,
		supportedLocalesFunc: nil,
	}
	pe.PackagePath = packagePath
	pe.LinkFuncs()

	assert.NotNil(t, pe.astFunc)
	assert.NotNil(t, pe.cachePolicyFunc)
	assert.NotNil(t, pe.middlewareFunc)
	assert.NotNil(t, pe.supportedLocalesFunc)
}

func TestPageEntry_LinkFuncs_UsesDefaultRegistryWhenNil(t *testing.T) {
	t.Parallel()

	pe := &PageEntry{}
	pe.PackagePath = "some/unregistered/package"
	pe.LinkFuncs()

	assert.NotNil(t, pe.cachePolicyFunc)
	assert.NotNil(t, pe.middlewareFunc)
	assert.NotNil(t, pe.supportedLocalesFunc)
}

func TestManifestStore_GetKeys(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		keys: []string{"pages/index.pk", "pages/about.pk", "partials/card.pk"},
	}
	assert.Equal(t, []string{"pages/index.pk", "pages/about.pk", "partials/card.pk"}, store.GetKeys())
}

func TestManifestStore_GetKeys_Empty(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{
		keys: []string{},
	}
	assert.Empty(t, store.GetKeys())
}

func TestManifestStore_GetPageEntry(t *testing.T) {
	t.Parallel()

	pageEntry := &PageEntry{}
	pageEntry.OriginalSourcePath = "pages/index.pk"

	partialEntry := &PageEntry{}
	partialEntry.OriginalSourcePath = "partials/card.pk"

	emailEntry := &PageEntry{}
	emailEntry.OriginalSourcePath = "emails/welcome.pk"

	store := &ManifestStore{
		pages:    map[string]*PageEntry{"pages/index.pk": pageEntry},
		partials: map[string]*PageEntry{"partials/card.pk": partialEntry},
		emails:   map[string]*PageEntry{"emails/welcome.pk": emailEntry},
	}

	t.Run("find page", func(t *testing.T) {
		t.Parallel()
		entry, found := store.GetPageEntry("pages/index.pk")
		assert.True(t, found)
		assert.Equal(t, "pages/index.pk", entry.GetOriginalPath())
	})

	t.Run("find partial", func(t *testing.T) {
		t.Parallel()
		entry, found := store.GetPageEntry("partials/card.pk")
		assert.True(t, found)
		assert.Equal(t, "partials/card.pk", entry.GetOriginalPath())
	})

	t.Run("find email", func(t *testing.T) {
		t.Parallel()
		entry, found := store.GetPageEntry("emails/welcome.pk")
		assert.True(t, found)
		assert.Equal(t, "emails/welcome.pk", entry.GetOriginalPath())
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		entry, found := store.GetPageEntry("nonexistent.pk")
		assert.False(t, found)
		assert.Nil(t, entry)
	})
}

func TestWithBaseDir(t *testing.T) {
	t.Parallel()

	store := &ManifestStore{}
	opt := WithBaseDir("/my/project")
	opt(store)
	assert.Equal(t, "/my/project", store.baseDir)
}

func TestWithRegistry(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	store := &ManifestStore{}
	opt := withRegistry(registry)
	opt(store)
	assert.Equal(t, registry, store.registry)
}

func TestNewSimpleRenderer(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	require.NotNil(t, renderer)
}

func TestSimpleRenderer_RenderPage(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	var buffer bytes.Buffer

	params := templater_domain.RenderPageParams{
		Writer: &buffer,
		PageDefinition: templater_dto.PageDefinition{
			NormalisedPath: "/about",
		},
		TemplateAST: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
			},
		},
		Metadata: &templater_dto.InternalMetadata{},
	}

	err := renderer.RenderPage(context.Background(), params)
	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "Page: /about")
	assert.Contains(t, output, "AST RootNodes: 1")
}

func TestSimpleRenderer_RenderPartial(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	var buffer bytes.Buffer

	params := templater_domain.RenderPageParams{
		Writer: &buffer,
		PageDefinition: templater_dto.PageDefinition{
			NormalisedPath: "/_piko/partial/card",
		},
		TemplateAST: nil,
		Metadata:    &templater_dto.InternalMetadata{},
	}

	err := renderer.RenderPartial(context.Background(), params)
	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "Partial: /_piko/partial/card")
	assert.Contains(t, output, "AST RootNodes: 0")
}

func TestSimpleRenderer_RenderEmail(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	var buffer bytes.Buffer

	params := templater_domain.RenderEmailParams{
		Writer:   &buffer,
		PageID:   "emails/welcome",
		Metadata: &templater_dto.InternalMetadata{},
		TemplateAST: &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeElement, TagName: "div"},
				{NodeType: ast_domain.NodeText, TextContent: "Hello"},
			},
		},
		Styling: ".email { color: blue; }",
	}

	err := renderer.RenderEmail(context.Background(), params)
	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "Email: emails/welcome")
	assert.Contains(t, output, "AST RootNodes: 2")
	assert.Contains(t, output, "Styling: 23 bytes")
}

func TestSimpleRenderer_RenderEmail_NilAST(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	var buffer bytes.Buffer

	params := templater_domain.RenderEmailParams{
		Writer:      &buffer,
		PageID:      "emails/test",
		Metadata:    &templater_dto.InternalMetadata{},
		TemplateAST: nil,
	}

	err := renderer.RenderEmail(context.Background(), params)
	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "AST RootNodes: 0")
}

func TestSimpleRenderer_CollectMetadata(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	headers, _, err := renderer.CollectMetadata(context.Background(), request, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, headers)
}

func TestSimpleRenderer_RenderASTToPlainText(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()

	t.Run("nil AST", func(t *testing.T) {
		t.Parallel()
		text, err := renderer.RenderASTToPlainText(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "", text)
	})

	t.Run("text nodes", func(t *testing.T) {
		t.Parallel()
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{NodeType: ast_domain.NodeText, TextContent: "Hello "},
				{NodeType: ast_domain.NodeText, TextContent: "World"},
			},
		}
		text, err := renderer.RenderASTToPlainText(context.Background(), ast)
		require.NoError(t, err)
		assert.Equal(t, "Hello World", text)
	})

	t.Run("element with children", func(t *testing.T) {
		t.Parallel()
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "inside div"},
					},
				},
			},
		}
		text, err := renderer.RenderASTToPlainText(context.Background(), ast)
		require.NoError(t, err)
		assert.Equal(t, "inside div", text)
	})

	t.Run("fragment with children", func(t *testing.T) {
		t.Parallel()
		ast := &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeFragment,
					Children: []*ast_domain.TemplateNode{
						{NodeType: ast_domain.NodeText, TextContent: "fragment text"},
					},
				},
			},
		}
		text, err := renderer.RenderASTToPlainText(context.Background(), ast)
		require.NoError(t, err)
		assert.Equal(t, "fragment text", text)
	})

	t.Run("empty AST", func(t *testing.T) {
		t.Parallel()
		ast := &ast_domain.TemplateAST{}
		text, err := renderer.RenderASTToPlainText(context.Background(), ast)
		require.NoError(t, err)
		assert.Equal(t, "", text)
	})
}

func TestSimpleRenderer_GetLastEmailAssetRequests(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	requests := renderer.GetLastEmailAssetRequests()
	assert.Empty(t, requests)
}

func TestNewDrivenRenderer(t *testing.T) {
	t.Parallel()

	renderer := NewDrivenRenderer(nil)
	require.NotNil(t, renderer)
}

func TestNewCachingManifestRunner(t *testing.T) {
	t.Parallel()

	runner := NewCachingManifestRunner(nil, nil)
	require.NotNil(t, runner)
}

func TestNewCompiledManifestRunner(t *testing.T) {
	t.Parallel()

	runner := NewCompiledManifestRunner(nil, nil, "en")
	require.NotNil(t, runner)
}

func TestNewInterpretedManifestRunner(t *testing.T) {
	t.Parallel()

	cache := make(map[string]*PageEntry)
	runner := NewInterpretedManifestRunner(nil, cache, nil, "en")
	require.NotNil(t, runner)
}

func TestNewInterpretedManifestStoreView(t *testing.T) {
	t.Parallel()

	runner := &InterpretedManifestRunner{
		progCache: make(map[string]*PageEntry),
	}
	view := NewInterpretedManifestStoreView(runner)
	require.NotNil(t, view)
}

func TestInterpretedManifestStoreView_GetKeys(t *testing.T) {
	t.Parallel()

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/b.pk": {},
			"pages/a.pk": {},
			"pages/c.pk": {},
		},
	}
	view := NewInterpretedManifestStoreView(runner)
	keys := view.GetKeys()
	assert.Equal(t, []string{"pages/a.pk", "pages/b.pk", "pages/c.pk"}, keys)
}

func TestInterpretedManifestStoreView_GetKeys_Empty(t *testing.T) {
	t.Parallel()

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{},
	}
	view := NewInterpretedManifestStoreView(runner)
	keys := view.GetKeys()
	assert.Empty(t, keys)
}

func TestInterpretedManifestStoreView_GetPageEntry(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/home.pk"

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/home.pk": entry,
		},
	}
	view := NewInterpretedManifestStoreView(runner)

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		pe, found := view.GetPageEntry("pages/home.pk")
		assert.True(t, found)
		assert.Equal(t, entry, pe)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		pe, found := view.GetPageEntry("nonexistent.pk")
		assert.False(t, found)
		assert.Nil(t, pe)
	})
}

func TestInterpretedManifestRunner_GetPageEntryByPath_NoOrchestrator(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/test.pk"

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/test.pk": entry,
		},
		orchestrator: nil,
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		pe, found := runner.GetPageEntryByPath("pages/test.pk")
		assert.True(t, found)
		assert.Equal(t, entry, pe)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		pe, found := runner.GetPageEntryByPath("nonexistent")
		assert.False(t, found)
		assert.Nil(t, pe)
	})
}

func TestGenerateCacheKey(t *testing.T) {
	t.Parallel()

	t.Run("consistent for same input", func(t *testing.T) {
		t.Parallel()

		request := httptest.NewRequest(http.MethodGet, "/about?lang=en", nil)
		entry := &PageEntry{}
		entry.OriginalSourcePath = "pages/about.pk"

		key1 := generateCacheKey(request, entry)
		key2 := generateCacheKey(request, entry)
		assert.Equal(t, key1, key2)
		assert.Len(t, key1, 16)
	})

	t.Run("different for different paths", func(t *testing.T) {
		t.Parallel()

		req1 := httptest.NewRequest(http.MethodGet, "/about", nil)
		req2 := httptest.NewRequest(http.MethodGet, "/home", nil)

		entry := &PageEntry{}
		entry.OriginalSourcePath = "pages/about.pk"

		key1 := generateCacheKey(req1, entry)
		key2 := generateCacheKey(req2, entry)
		assert.NotEqual(t, key1, key2)
	})

	t.Run("different for fragment vs full", func(t *testing.T) {
		t.Parallel()

		reqFull := httptest.NewRequest(http.MethodGet, "/about", nil)
		reqFragment := httptest.NewRequest(http.MethodGet, "/about?_f=true", nil)

		entry := &PageEntry{}
		entry.OriginalSourcePath = "pages/about.pk"

		keyFull := generateCacheKey(reqFull, entry)
		keyFragment := generateCacheKey(reqFragment, entry)
		assert.NotEqual(t, keyFull, keyFragment)
	})

	t.Run("different for different query params", func(t *testing.T) {
		t.Parallel()

		req1 := httptest.NewRequest(http.MethodGet, "/about?page=1", nil)
		req2 := httptest.NewRequest(http.MethodGet, "/about?page=2", nil)

		entry := &PageEntry{}
		entry.OriginalSourcePath = "pages/about.pk"

		key1 := generateCacheKey(req1, entry)
		key2 := generateCacheKey(req2, entry)
		assert.NotEqual(t, key1, key2)
	})
}

func TestSortKeysByRouteSpecificityWithLookup(t *testing.T) {
	t.Parallel()

	entries := map[string]*PageEntry{
		"pages/{slug}.pk": {
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				RoutePatterns: map[string]string{"en": "/{slug}"},
			},
		},
		"pages/index.pk": {
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				RoutePatterns: map[string]string{"en": "/"},
			},
		},
		"pages/docs/api.pk": {
			ManifestPageEntry: generator_dto.ManifestPageEntry{
				RoutePatterns: map[string]string{"en": "/docs/api"},
			},
		},
	}

	keys := []string{"pages/{slug}.pk", "pages/index.pk", "pages/docs/api.pk"}
	sortKeysByRouteSpecificityWithLookup(keys, func(key string) *PageEntry {
		return entries[key]
	})

	require.Len(t, keys, 3)
	assert.Equal(t, "pages/docs/api.pk", keys[0])
	assert.Equal(t, "pages/index.pk", keys[1])
	assert.Equal(t, "pages/{slug}.pk", keys[2])
}

func TestSortKeysByRouteSpecificityWithLookup_NilEntries(t *testing.T) {
	t.Parallel()

	keys := []string{"a", "b", "c"}
	sortKeysByRouteSpecificityWithLookup(keys, func(_ string) *PageEntry {
		return nil
	})

	assert.Equal(t, []string{"a", "b", "c"}, keys)
}

func TestInterpretedManifestRunner_LookupPageEntry(t *testing.T) {
	t.Parallel()

	entry := &PageEntry{}
	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/home.pk": entry,
		},
	}

	t.Run("found", func(t *testing.T) {
		t.Parallel()
		pe, found := runner.lookupPageEntry("pages/home.pk")
		assert.True(t, found)
		assert.Equal(t, entry, pe)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		pe, found := runner.lookupPageEntry("pages/missing.pk")
		assert.False(t, found)
		assert.Nil(t, pe)
	})
}

func TestInterpretedManifestRunner_TriggerJITCompilation_NilOrchestrator(t *testing.T) {
	t.Parallel()

	runner := &InterpretedManifestRunner{
		orchestrator: nil,
	}

	runner.triggerJITCompilation(context.Background(), "pages/test.pk")
}

func TestPartialJSArtefactIDsToSlice_AlreadyCovered(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []string{"id-1"}, partialJSArtefactIDsToSlice("id-1"))
	assert.Nil(t, partialJSArtefactIDsToSlice(""))
}

func TestSimpleRenderer_RenderDebug_NilAST(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	var buffer bytes.Buffer

	params := templater_domain.RenderPageParams{
		Writer: &buffer,
		PageDefinition: templater_dto.PageDefinition{
			NormalisedPath: "/test",
		},
		TemplateAST: nil,
		Metadata:    &templater_dto.InternalMetadata{},
	}

	err := renderer.RenderPage(context.Background(), params)
	require.NoError(t, err)
	assert.Contains(t, buffer.String(), "AST RootNodes: 0")
}

func TestSimpleRenderer_RenderPage_WithMetadata(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	var buffer bytes.Buffer

	metadata := &templater_dto.InternalMetadata{
		CustomTags: []string{"my-element"},
	}

	params := templater_domain.RenderPageParams{
		Writer: &buffer,
		PageDefinition: templater_dto.PageDefinition{
			NormalisedPath: "/metadata-test",
		},
		TemplateAST: nil,
		Metadata:    metadata,
	}

	err := renderer.RenderPage(context.Background(), params)
	require.NoError(t, err)
	output := buffer.String()
	assert.Contains(t, output, "Page: /metadata-test")
	assert.Contains(t, output, "SnippetData:")
	assert.Contains(t, output, "my-element")
}

func TestSimpleRenderer_RenderPage_WriterError(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	w := &failWriter{}

	params := templater_domain.RenderPageParams{
		Writer: w,
		PageDefinition: templater_dto.PageDefinition{
			NormalisedPath: "/fail",
		},
		Metadata: &templater_dto.InternalMetadata{},
	}

	err := renderer.RenderPage(context.Background(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writing page render output")
}

func TestSimpleRenderer_RenderEmail_WriterError(t *testing.T) {
	t.Parallel()

	renderer := newSimpleRenderer()
	w := &failWriter{}

	params := templater_domain.RenderEmailParams{
		Writer:   w,
		PageID:   "test",
		Metadata: &templater_dto.InternalMetadata{},
	}

	err := renderer.RenderEmail(context.Background(), params)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "writing email render output")
}

func TestInterpretedManifestRunner_GetKeys_NoOrchestrator(t *testing.T) {
	t.Parallel()

	runner := &InterpretedManifestRunner{
		progCache: map[string]*PageEntry{
			"pages/b.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/b"},
				},
			},
			"pages/a.pk": {
				ManifestPageEntry: generator_dto.ManifestPageEntry{
					RoutePatterns: map[string]string{"en": "/a"},
				},
			},
		},
		orchestrator: nil,
	}

	keys := runner.GetKeys()
	require.Len(t, keys, 2)

	assert.Equal(t, "pages/a.pk", keys[0])
	assert.Equal(t, "pages/b.pk", keys[1])
}

type failWriter struct{}

func (w *failWriter) Write(_ []byte) (int, error) {
	return 0, errors.New("write error")
}
