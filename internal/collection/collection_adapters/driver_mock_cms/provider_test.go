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

package driver_mock_cms_test

import (
	"context"
	goast "go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/collection/collection_adapters/driver_mock_cms"
	"piko.sh/piko/internal/collection/collection_domain"
	"piko.sh/piko/internal/collection/collection_dto"
)

func TestMockCMSProvider_Name(t *testing.T) {
	provider := driver_mock_cms.NewMockCMSProvider("mock-cms")

	assert.Equal(t, "mock-cms", provider.Name())
}

func TestMockCMSProvider_Type(t *testing.T) {
	provider := driver_mock_cms.NewMockCMSProvider("mock-cms")

	assert.Equal(t, collection_domain.ProviderTypeDynamic, provider.Type())
}

func TestMockCMSProvider_FetchStaticContent_ReturnsError(t *testing.T) {
	provider := driver_mock_cms.NewMockCMSProvider("mock-cms")
	ctx := context.Background()

	_, err := provider.FetchStaticContent(ctx, "blog")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support static fetching")
}

func TestMockCMSProvider_ValidateTargetType_AcceptsAnyType(t *testing.T) {
	provider := driver_mock_cms.NewMockCMSProvider("mock-cms")

	testCases := []struct {
		typeExpr goast.Expr
		name     string
	}{
		{
			name:     "Simple identifier",
			typeExpr: goast.NewIdent("Post"),
		},
		{
			name: "Selector expression",
			typeExpr: &goast.SelectorExpr{
				X:   goast.NewIdent("models"),
				Sel: goast.NewIdent("Post"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := provider.ValidateTargetType(tc.typeExpr)
			assert.NoError(t, err, "Mock CMS should accept any type")
		})
	}
}

func TestMockCMSProvider_GenerateRuntimeFetcher(t *testing.T) {
	provider := driver_mock_cms.NewMockCMSProvider("mock-cms")
	ctx := context.Background()

	targetType := goast.NewIdent("Post")
	options := collection_dto.FetchOptions{
		Locale: "en",
		Cache: &collection_dto.CacheConfig{
			Strategy: "cache-first",
			TTL:      120,
		},
	}

	fetcherCode, err := provider.GenerateRuntimeFetcher(ctx, "blog", targetType, options)

	require.NoError(t, err)
	require.NotNil(t, fetcherCode)

	assert.NotNil(t, fetcherCode.FetcherFunc, "Should have fetcher function AST")
	assert.NotEmpty(t, fetcherCode.RequiredImports, "Should specify required imports")
	assert.Contains(t, fetcherCode.RequiredImports, "context")
	assert.Contains(t, fetcherCode.RequiredImports, "piko.sh/piko/wdk/runtime")
	assert.Equal(t, "cache-first", fetcherCode.CacheStrategy)
	assert.NotNil(t, fetcherCode.RetryConfig)

	fetcherFunction := fetcherCode.FetcherFunc
	assert.NotNil(t, fetcherFunction.Name, "Function should have a name")
	assert.NotNil(t, fetcherFunction.Type, "Function should have a type")
	assert.NotNil(t, fetcherFunction.Type.Params, "Function should have parameters")
	assert.NotNil(t, fetcherFunction.Type.Results, "Function should have return values")
	assert.NotNil(t, fetcherFunction.Body, "Function should have a body")

	assert.Len(t, fetcherFunction.Type.Params.List, 1, "Should have one parameter")
	param := fetcherFunction.Type.Params.List[0]
	assert.Len(t, param.Names, 1, "Parameter should have a name")
	assert.Equal(t, "ctx", param.Names[0].Name)

	assert.Len(t, fetcherFunction.Type.Results.List, 2, "Should have two return values")

	assert.NotEmpty(t, fetcherFunction.Body.List, "Function body should have statements")
}

func TestMockCMSProvider_DiscoverCollections(t *testing.T) {
	provider := driver_mock_cms.NewMockCMSProvider("mock-cms")
	ctx := context.Background()

	config := collection_dto.ProviderConfig{
		BasePath:      "/test",
		Locales:       []string{"en", "fr"},
		DefaultLocale: "en",
	}

	collections, err := provider.DiscoverCollections(ctx, config)

	require.NoError(t, err)
	assert.Empty(t, collections, "Mock CMS returns empty collection list")
}

func TestMockCMSRuntimeProvider_Name(t *testing.T) {
	runtime := driver_mock_cms.NewMockCMSRuntimeProvider("mock-cms")

	assert.Equal(t, "mock-cms", runtime.Name())
}

func TestMockCMSRuntimeProvider_Fetch_WithNoData(t *testing.T) {
	runtime := driver_mock_cms.NewMockCMSRuntimeProvider("mock-cms")
	ctx := context.Background()

	var results []map[string]any
	err := runtime.Fetch(ctx, "blog", &collection_dto.FetchOptions{}, &results)

	require.NoError(t, err)
	assert.Empty(t, results, "Should return empty slice when no mock data set")
}

func TestMockCMSRuntimeProvider_Fetch_WithMockData(t *testing.T) {
	runtime := driver_mock_cms.NewMockCMSRuntimeProvider("mock-cms")

	mockData := []byte(`[
		{"Title": "Hello World", "Slug": "hello"},
		{"Title": "Second Post", "Slug": "second"}
	]`)
	runtime.SetMockData("blog", mockData)

	ctx := context.Background()
	var results []map[string]any
	err := runtime.Fetch(ctx, "blog", &collection_dto.FetchOptions{}, &results)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Hello World", results[0]["Title"])
	assert.Equal(t, "hello", results[0]["Slug"])
	assert.Equal(t, "Second Post", results[1]["Title"])
	assert.Equal(t, "second", results[1]["Slug"])
}

func TestMockCMSRuntimeProvider_ClearMockData(t *testing.T) {
	runtime := driver_mock_cms.NewMockCMSRuntimeProvider("mock-cms")

	mockData := []byte(`[{"Title": "Test"}]`)
	runtime.SetMockData("blog", mockData)

	runtime.ClearMockData()

	ctx := context.Background()
	var results []map[string]any
	err := runtime.Fetch(ctx, "blog", &collection_dto.FetchOptions{}, &results)

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestMockCMSRuntimeProvider_Fetch_InvalidJSON(t *testing.T) {
	runtime := driver_mock_cms.NewMockCMSRuntimeProvider("mock-cms")

	invalidJSON := []byte(`{this is not valid json}`)
	runtime.SetMockData("blog", invalidJSON)

	ctx := context.Background()
	var results []map[string]any
	err := runtime.Fetch(ctx, "blog", &collection_dto.FetchOptions{}, &results)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshalling")
}
