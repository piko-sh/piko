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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestGlobalRegisterASTFunc(t *testing.T) {
	packagePath := "global_test/ast_func"
	defer templater_domain.Unregister(packagePath)

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	templater_domain.RegisterASTFunc(packagePath, testFunc)
	registryFunction, ok := templater_domain.GetASTFunc(packagePath)

	assert.True(t, ok, "should find registered global AST function")
	assert.NotNil(t, registryFunction)
}

func TestGlobalGetASTFunc_NotFound(t *testing.T) {
	registryFunction, ok := templater_domain.GetASTFunc("global_test/nonexistent_ast")

	assert.False(t, ok)
	assert.Nil(t, registryFunction)
}

func TestGlobalRegisterCachePolicyFunc(t *testing.T) {
	packagePath := "global_test/cache_policy"
	defer templater_domain.Unregister(packagePath)

	testFunc := func(r *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{
			Enabled:       true,
			MaxAgeSeconds: 600,
		}
	}

	templater_domain.RegisterCachePolicyFunc(packagePath, testFunc)
	registryFunction := templater_domain.GetCachePolicyFunc(packagePath)

	require.NotNil(t, registryFunction)
	policy := registryFunction(&templater_dto.RequestData{})
	assert.True(t, policy.Enabled)
	assert.Equal(t, 600, policy.MaxAgeSeconds)
}

func TestGlobalGetCachePolicyFunc_Default(t *testing.T) {
	registryFunction := templater_domain.GetCachePolicyFunc("global_test/nonexistent_cache")

	require.NotNil(t, registryFunction, "should return default function")
	policy := registryFunction(&templater_dto.RequestData{})
	assert.False(t, policy.Enabled)
	assert.Equal(t, 0, policy.MaxAgeSeconds)
}

func TestGlobalRegisterMiddlewareFunc(t *testing.T) {
	packagePath := "global_test/middleware"
	defer templater_domain.Unregister(packagePath)

	testFunc := func() []func(http.Handler) http.Handler {
		return []func(http.Handler) http.Handler{
			func(next http.Handler) http.Handler {
				return next
			},
		}
	}

	templater_domain.RegisterMiddlewareFunc(packagePath, testFunc)
	registryFunction := templater_domain.GetMiddlewareFunc(packagePath)

	require.NotNil(t, registryFunction)
	middlewares := registryFunction()
	assert.Len(t, middlewares, 1)
}

func TestGlobalGetMiddlewareFunc_Default(t *testing.T) {
	registryFunction := templater_domain.GetMiddlewareFunc("global_test/nonexistent_mw")

	require.NotNil(t, registryFunction, "should return default function")
	middlewares := registryFunction()
	assert.Nil(t, middlewares)
}

func TestGlobalRegisterSupportedLocalesFunc(t *testing.T) {
	packagePath := "global_test/locales"
	defer templater_domain.Unregister(packagePath)

	testFunc := func() []string {
		return []string{"en_GB", "fr_FR", "de_DE"}
	}

	templater_domain.RegisterSupportedLocalesFunc(packagePath, testFunc)
	registryFunction := templater_domain.GetSupportedLocalesFunc(packagePath)

	require.NotNil(t, registryFunction)
	locales := registryFunction()
	assert.Equal(t, []string{"en_GB", "fr_FR", "de_DE"}, locales)
}

func TestGlobalGetSupportedLocalesFunc_Default(t *testing.T) {
	registryFunction := templater_domain.GetSupportedLocalesFunc("global_test/nonexistent_locales")

	require.NotNil(t, registryFunction, "should return default function")
	locales := registryFunction()
	assert.Nil(t, locales)
}

func TestGlobalUnregister(t *testing.T) {
	packagePath := "global_test/unregister"

	testASTFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}
	testCacheFunc := func(r *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{Enabled: true}
	}

	templater_domain.RegisterASTFunc(packagePath, testASTFunc)
	templater_domain.RegisterCachePolicyFunc(packagePath, testCacheFunc)

	registryFunction, ok := templater_domain.GetASTFunc(packagePath)
	assert.True(t, ok)
	assert.NotNil(t, registryFunction)

	templater_domain.Unregister(packagePath)

	fn2, ok2 := templater_domain.GetASTFunc(packagePath)
	assert.False(t, ok2, "should not find unregistered AST function")
	assert.Nil(t, fn2)

	cacheFunction := templater_domain.GetCachePolicyFunc(packagePath)
	policy := cacheFunction(&templater_dto.RequestData{})
	assert.False(t, policy.Enabled, "should return default disabled policy after unregister")
}

func TestGlobalClear(t *testing.T) {
	pkgPath1 := "global_test/clear_1"
	pkgPath2 := "global_test/clear_2"

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	templater_domain.RegisterASTFunc(pkgPath1, testFunc)
	templater_domain.RegisterASTFunc(pkgPath2, testFunc)

	templater_domain.Clear()

	fn1, ok1 := templater_domain.GetASTFunc(pkgPath1)
	fn2, ok2 := templater_domain.GetASTFunc(pkgPath2)

	assert.False(t, ok1)
	assert.Nil(t, fn1)
	assert.False(t, ok2)
	assert.Nil(t, fn2)
}

func TestGlobalList(t *testing.T) {

	templater_domain.Clear()

	pkgPath1 := "global_test/list_1"
	pkgPath2 := "global_test/list_2"
	defer templater_domain.Clear()

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	templater_domain.RegisterASTFunc(pkgPath1, testFunc)
	templater_domain.RegisterASTFunc(pkgPath2, testFunc)

	pkgPaths := templater_domain.List()

	assert.Contains(t, pkgPaths, pkgPath1)
	assert.Contains(t, pkgPaths, pkgPath2)
}

func TestIsolatedRegistry_RegisterAndGetMiddlewareFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/middleware_component"

	testFunc := func() []func(http.Handler) http.Handler {
		return []func(http.Handler) http.Handler{
			func(next http.Handler) http.Handler { return next },
			func(next http.Handler) http.Handler { return next },
		}
	}

	registry.RegisterMiddlewareFunc(packagePath, testFunc)
	registryFunction := registry.GetMiddlewareFunc(packagePath)

	require.NotNil(t, registryFunction)
	middlewares := registryFunction()
	assert.Len(t, middlewares, 2)
}

func TestIsolatedRegistry_RegisterAndGetSupportedLocalesFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/locales_component"

	testFunc := func() []string {
		return []string{"en_US", "es_ES"}
	}

	registry.RegisterSupportedLocalesFunc(packagePath, testFunc)
	registryFunction := registry.GetSupportedLocalesFunc(packagePath)

	require.NotNil(t, registryFunction)
	locales := registryFunction()
	assert.Equal(t, []string{"en_US", "es_ES"}, locales)
}

func TestIsolatedRegistry_List_AllFunctionTypes(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	astFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return nil, templater_dto.InternalMetadata{}, nil
	}
	cacheFunc := func(r *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{}
	}
	mwFunc := func() []func(http.Handler) http.Handler { return nil }
	localesFunc := func() []string { return nil }

	registry.RegisterASTFunc("pkg/ast_only", astFunc)
	registry.RegisterCachePolicyFunc("pkg/cache_only", cacheFunc)
	registry.RegisterMiddlewareFunc("pkg/mw_only", mwFunc)
	registry.RegisterSupportedLocalesFunc("pkg/locales_only", localesFunc)

	pkgPaths := registry.List()

	assert.Len(t, pkgPaths, 4)
	assert.Contains(t, pkgPaths, "pkg/ast_only")
	assert.Contains(t, pkgPaths, "pkg/cache_only")
	assert.Contains(t, pkgPaths, "pkg/mw_only")
	assert.Contains(t, pkgPaths, "pkg/locales_only")
}

func TestIsolatedRegistry_List_Empty(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	pkgPaths := registry.List()

	assert.Empty(t, pkgPaths)
}

func TestIsolatedRegistry_Unregister_RemovesAllFunctionTypes(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/full_component"

	astFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return nil, templater_dto.InternalMetadata{}, nil
	}
	cacheFunc := func(r *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{Enabled: true}
	}
	mwFunc := func() []func(http.Handler) http.Handler {
		return []func(http.Handler) http.Handler{func(h http.Handler) http.Handler { return h }}
	}
	localesFunc := func() []string { return []string{"en_GB"} }

	registry.RegisterASTFunc(packagePath, astFunc)
	registry.RegisterCachePolicyFunc(packagePath, cacheFunc)
	registry.RegisterMiddlewareFunc(packagePath, mwFunc)
	registry.RegisterSupportedLocalesFunc(packagePath, localesFunc)

	_, ok := registry.GetASTFunc(packagePath)
	assert.True(t, ok)

	registry.Unregister(packagePath)

	_, ok = registry.GetASTFunc(packagePath)
	assert.False(t, ok, "AST func should be gone")

	cacheFunction := registry.GetCachePolicyFunc(packagePath)
	policy := cacheFunction(&templater_dto.RequestData{})
	assert.False(t, policy.Enabled, "cache policy should return default")

	middlewareFunction := registry.GetMiddlewareFunc(packagePath)
	assert.Nil(t, middlewareFunction(), "middleware should return default nil")

	localeFunction := registry.GetSupportedLocalesFunc(packagePath)
	assert.Nil(t, localeFunction(), "locales should return default nil")

	assert.Empty(t, registry.List())
}

func TestIsolatedRegistry_Clear_RemovesAllFunctionTypes(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	mwFunc := func() []func(http.Handler) http.Handler { return nil }
	localesFunc := func() []string { return []string{"en_GB"} }

	registry.RegisterMiddlewareFunc("pkg/mw", mwFunc)
	registry.RegisterSupportedLocalesFunc("pkg/locales", localesFunc)

	assert.Len(t, registry.List(), 2)

	registry.Clear()

	assert.Empty(t, registry.List())
}

func TestIsolatedRegistry_OverwriteRegistration(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/overwrite"

	firstFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TextContent: "first"}},
		}, templater_dto.InternalMetadata{}, nil
	}

	secondFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{
			RootNodes: []*ast_domain.TemplateNode{{TextContent: "second"}},
		}, templater_dto.InternalMetadata{}, nil
	}

	registry.RegisterASTFunc(packagePath, firstFunc)
	registry.RegisterASTFunc(packagePath, secondFunc)

	registryFunction, ok := registry.GetASTFunc(packagePath)
	require.True(t, ok)

	ast, _, _ := registryFunction(nil, nil)
	assert.Equal(t, "second", ast.RootNodes[0].TextContent, "second registration should overwrite first")
}
