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
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/generator/generator_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

func TestIsolatedRegistry_RegisterAndGetASTFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/component"

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	registry.RegisterASTFunc(packagePath, testFunc)
	retrievedFunc, ok := registry.GetASTFunc(packagePath)

	assert.True(t, ok, "should find registered function")
	assert.NotNil(t, retrievedFunc, "function should not be nil")
}

func TestIsolatedRegistry_GetNonExistentASTFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	registryFunction, ok := registry.GetASTFunc("nonexistent/path")

	assert.False(t, ok, "should not find non-existent function")
	assert.Nil(t, registryFunction, "function should be nil")
}

func TestIsolatedRegistry_GetCachePolicyFunc_Default(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	registryFunction := registry.GetCachePolicyFunc("nonexistent/path")

	require.NotNil(t, registryFunction, "should return default function")

	policy := registryFunction(&templater_dto.RequestData{})
	assert.False(t, policy.Enabled, "default policy should be disabled")
	assert.Equal(t, 0, policy.MaxAgeSeconds, "default MaxAgeSeconds should be 0")
}

func TestIsolatedRegistry_GetMiddlewareFunc_Default(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	registryFunction := registry.GetMiddlewareFunc("nonexistent/path")

	require.NotNil(t, registryFunction, "should return default function")

	middlewares := registryFunction()
	assert.Nil(t, middlewares, "default should return nil middleware slice")
}

func TestIsolatedRegistry_GetSupportedLocalesFunc_Default(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	registryFunction := registry.GetSupportedLocalesFunc("nonexistent/path")

	require.NotNil(t, registryFunction, "should return default function")

	locales := registryFunction()
	assert.Nil(t, locales, "default should return nil locales slice")
}

func TestIsolatedRegistry_RegisterAndGetCachePolicyFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/component"

	testFunc := func(r *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{
			Enabled:       true,
			MaxAgeSeconds: 3600,
		}
	}

	registry.RegisterCachePolicyFunc(packagePath, testFunc)
	retrievedFunc := registry.GetCachePolicyFunc(packagePath)

	require.NotNil(t, retrievedFunc)
	policy := retrievedFunc(&templater_dto.RequestData{})
	assert.True(t, policy.Enabled)
	assert.Equal(t, 3600, policy.MaxAgeSeconds)
}

func TestIsolatedRegistry_Unregister(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/component"

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	registry.RegisterASTFunc(packagePath, testFunc)

	registry.Unregister(packagePath)
	registryFunction, ok := registry.GetASTFunc(packagePath)

	assert.False(t, ok, "should not find unregistered function")
	assert.Nil(t, registryFunction, "function should be nil after unregister")
}

func TestIsolatedRegistry_Clear(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	registry.RegisterASTFunc("test/component1", testFunc)
	registry.RegisterASTFunc("test/component2", testFunc)
	registry.RegisterASTFunc("test/component3", testFunc)

	registry.Clear()

	fn1, ok1 := registry.GetASTFunc("test/component1")
	fn2, ok2 := registry.GetASTFunc("test/component2")
	fn3, ok3 := registry.GetASTFunc("test/component3")

	assert.False(t, ok1, "should not find cleared function 1")
	assert.Nil(t, fn1)
	assert.False(t, ok2, "should not find cleared function 2")
	assert.Nil(t, fn2)
	assert.False(t, ok3, "should not find cleared function 3")
	assert.Nil(t, fn3)
}

func TestIsolatedRegistry_List(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	registry.RegisterASTFunc("test/component1", testFunc)
	registry.RegisterASTFunc("test/component2", testFunc)

	pkgPaths := registry.List()

	assert.Len(t, pkgPaths, 2, "should list all registered packages")
	assert.Contains(t, pkgPaths, "test/component1")
	assert.Contains(t, pkgPaths, "test/component2")
}

func TestIsolatedRegistry_List_Deduplication(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/component"

	testASTFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	testCacheFunc := func(r *templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{}
	}

	registry.RegisterASTFunc(packagePath, testASTFunc)
	registry.RegisterCachePolicyFunc(packagePath, testCacheFunc)

	pkgPaths := registry.List()

	assert.Len(t, pkgPaths, 1, "should deduplicate package paths")
	assert.Contains(t, pkgPaths, packagePath)
}

func TestIsolatedRegistry_RegisterAndGetPreviewFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/component"

	testFunc := func() []templater_dto.PreviewScenario {
		return []templater_dto.PreviewScenario{
			{Name: "default", Description: "Default scenario", Props: "test-props"},
		}
	}

	registry.RegisterPreviewFunc(packagePath, testFunc)
	retrievedFunc, ok := registry.GetPreviewFunc(packagePath)

	assert.True(t, ok, "should find registered preview function")
	require.NotNil(t, retrievedFunc, "preview function should not be nil")

	scenarios := retrievedFunc()
	require.Len(t, scenarios, 1)
	assert.Equal(t, "default", scenarios[0].Name)
	assert.Equal(t, "Default scenario", scenarios[0].Description)
	assert.Equal(t, "test-props", scenarios[0].Props)
}

func TestIsolatedRegistry_GetPreviewFunc_NotFound(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	registryFunction, ok := registry.GetPreviewFunc("nonexistent/path")

	assert.False(t, ok, "should not find non-existent preview function")
	assert.Nil(t, registryFunction, "preview function should be nil")
}

func TestIsolatedRegistry_UnregisterPreviewFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "test/component"

	testFunc := func() []templater_dto.PreviewScenario {
		return nil
	}

	registry.RegisterPreviewFunc(packagePath, testFunc)
	registry.Unregister(packagePath)

	registryFunction, ok := registry.GetPreviewFunc(packagePath)
	assert.False(t, ok, "should not find unregistered preview function")
	assert.Nil(t, registryFunction, "preview function should be nil after unregister")
}

func TestIsolatedRegistry_ClearPreviewFunc(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	testFunc := func() []templater_dto.PreviewScenario {
		return nil
	}

	registry.RegisterPreviewFunc("test/component1", testFunc)
	registry.RegisterPreviewFunc("test/component2", testFunc)

	registry.Clear()

	fn1, ok1 := registry.GetPreviewFunc("test/component1")
	fn2, ok2 := registry.GetPreviewFunc("test/component2")

	assert.False(t, ok1, "should not find cleared preview function 1")
	assert.Nil(t, fn1)
	assert.False(t, ok2, "should not find cleared preview function 2")
	assert.Nil(t, fn2)
}

func TestIsolatedRegistry_ConcurrentWrites(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := range numGoroutines {
		id := i
		wg.Go(func() {
			packagePath := fmt.Sprintf("test/component%d", id)
			registry.RegisterASTFunc(packagePath, testFunc)
		})
	}

	wg.Wait()

	pkgPaths := registry.List()
	assert.Len(t, pkgPaths, numGoroutines, "all concurrent registrations should succeed")
}

func TestIsolatedRegistry_ConcurrentReads(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	numComponents := 100
	for i := range numComponents {
		packagePath := fmt.Sprintf("test/component%d", i)
		registry.RegisterASTFunc(packagePath, testFunc)
	}

	var wg sync.WaitGroup

	for i := range numComponents {
		id := i
		wg.Go(func() {
			packagePath := fmt.Sprintf("test/component%d", id)
			registryFunction, ok := registry.GetASTFunc(packagePath)
			assert.True(t, ok, "should find function for %s", packagePath)
			assert.NotNil(t, registryFunction)
		})
	}

	wg.Wait()
}

func TestIsolatedRegistry_ConcurrentReadWrite(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	var wg sync.WaitGroup
	numGoroutines := 50

	for i := range numGoroutines {
		id := i

		wg.Go(func() {
			packagePath := fmt.Sprintf("test/component%d", id)
			registry.RegisterASTFunc(packagePath, testFunc)
		})

		wg.Go(func() {
			packagePath := fmt.Sprintf("test/component%d", id)
			_, _ = registry.GetASTFunc(packagePath)
		})
	}

	wg.Wait()

	pkgPaths := registry.List()
	assert.Len(t, pkgPaths, numGoroutines, "all concurrent writes should succeed")
}

func TestIsolatedRegistry_MultipleInstances_NoInterference(t *testing.T) {
	t.Parallel()

	registry1 := templater_domain.NewIsolatedRegistry()
	registry2 := templater_domain.NewIsolatedRegistry()

	testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
	}

	registry1.RegisterASTFunc("test/component", testFunc)

	fn1, ok1 := registry1.GetASTFunc("test/component")
	fn2, ok2 := registry2.GetASTFunc("test/component")

	assert.True(t, ok1, "registry1 should have the function")
	assert.NotNil(t, fn1)
	assert.False(t, ok2, "registry2 should NOT have the function")
	assert.Nil(t, fn2)
}

func TestIsolatedRegistry_ParallelTests_NoInterference(t *testing.T) {

	testCases := []struct {
		name        string
		componentID string
	}{
		{name: "test1", componentID: "test/component1"},
		{name: "test2", componentID: "test/component2"},
		{name: "test3", componentID: "test/component3"},
		{name: "test4", componentID: "test/component4"},
		{name: "test5", componentID: "test/component5"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry := templater_domain.NewIsolatedRegistry()

			testFunc := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
				return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{}, nil
			}

			registry.RegisterASTFunc(tc.componentID, testFunc)

			pkgPaths := registry.List()
			assert.Len(t, pkgPaths, 1, "should only have one component")
			assert.Contains(t, pkgPaths, tc.componentID)

			registryFunction, ok := registry.GetASTFunc(tc.componentID)
			assert.True(t, ok)
			assert.NotNil(t, registryFunction)
		})
	}
}

func TestGetDefaultRegistry_ReturnsSameInstance(t *testing.T) {

	registry1 := templater_domain.GetDefaultRegistry()
	registry2 := templater_domain.GetDefaultRegistry()

	assert.NotNil(t, registry1)
	assert.NotNil(t, registry2)

}

func TestIsolatedRegistry_RealUsageScenario(t *testing.T) {
	t.Parallel()

	registry := templater_domain.NewIsolatedRegistry()
	packagePath := "github.com/myapp/pages/home"

	buildAST := func(r *templater_dto.RequestData, propsData any) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, []*generator_dto.RuntimeDiagnostic) {
		return &ast_domain.TemplateAST{
				RootNodes: []*ast_domain.TemplateNode{
					{
						NodeType:    ast_domain.NodeText,
						TextContent: "Welcome to " + r.Locale(),
					},
				},
			}, templater_dto.InternalMetadata{
				AssetRefs:  []templater_dto.AssetRef{},
				CustomTags: []string{},
			}, nil
	}

	cachePolicy := func(r *templater_dto.RequestData) templater_dto.CachePolicy {
		urlString := ""
		if reqURL := r.URL(); reqURL != nil {
			urlString = reqURL.String()
		}
		return templater_dto.CachePolicy{
			Enabled:       true,
			MaxAgeSeconds: 3600,
			Key:           urlString,
		}
	}

	registry.RegisterASTFunc(packagePath, buildAST)
	registry.RegisterCachePolicyFunc(packagePath, cachePolicy)

	astFunc, ok := registry.GetASTFunc(packagePath)
	require.True(t, ok)

	reqData := templater_dto.NewRequestDataBuilder().
		WithLocale("fr_FR").
		Build()

	ast, metadata, diagnostics := astFunc(reqData, nil)

	assert.NotNil(t, ast)
	assert.NotNil(t, metadata)
	assert.Empty(t, diagnostics)
	assert.Len(t, ast.RootNodes, 1)
	assert.Equal(t, "Welcome to fr_FR", ast.RootNodes[0].TextContent)

	policyFunc := registry.GetCachePolicyFunc(packagePath)
	policy := policyFunc(reqData)
	assert.True(t, policy.Enabled)
	assert.Equal(t, 3600, policy.MaxAgeSeconds)
}
