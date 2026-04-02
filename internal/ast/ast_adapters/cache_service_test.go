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

package ast_adapters

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/maypok86/otter/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
)

type mockL2Cache struct {
	storage map[string]*ast_domain.TemplateAST
	mu      sync.RWMutex
}

func newMockL2Cache() *mockL2Cache {
	return &mockL2Cache{
		storage: make(map[string]*ast_domain.TemplateAST),
	}
}

func (m *mockL2Cache) Get(ctx context.Context, key string) (*ast_domain.TemplateAST, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ast, found := m.storage[key]
	if !found {
		return nil, ast_domain.ErrCacheMiss
	}

	if ast.ExpiresAtUnixNano != nil {
		if time.Now().UnixNano() > *ast.ExpiresAtUnixNano {
			return nil, ast_domain.ErrCacheMiss
		}
	}

	return ast, nil
}

func (m *mockL2Cache) Set(ctx context.Context, key string, ast *ast_domain.TemplateAST) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := ast.DeepClone()
	clone.ExpiresAtUnixNano = nil
	m.storage[key] = clone
	return nil
}

func (m *mockL2Cache) SetWithTTL(ctx context.Context, key string, ast *ast_domain.TemplateAST, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	clone := ast.DeepClone()
	clone.ExpiresAtUnixNano = new(time.Now().Add(ttl).UnixNano())
	m.storage[key] = clone
	return nil
}

func (m *mockL2Cache) Delete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.storage, key)
	return nil
}

func newTestAST(id string) *ast_domain.TemplateAST {
	return &ast_domain.TemplateAST{
		RootNodes: []*ast_domain.TemplateNode{
			{TagName: "div", Attributes: []ast_domain.HTMLAttribute{{Name: "id", Value: id}}},
		},
	}
}

func newTestEntry(id string) *ast_domain.CachedASTEntry {
	return &ast_domain.CachedASTEntry{
		AST: newTestAST(id),
	}
}

func TestASTCacheService_New(t *testing.T) {
	t.Run("should create service with valid config", func(t *testing.T) {
		tempDir := t.TempDir()
		config := ASTCacheConfig{
			L1CacheCapacity: 100,
			L1CacheTTL:      1 * time.Hour,
			L2CacheBaseDir:  tempDir,
		}

		service, err := NewASTCacheService(context.Background(), config)
		require.NoError(t, err)
		t.Cleanup(func() { service.Shutdown(context.Background()) })
		require.NotNil(t, service)
		require.NotNil(t, service.cache)
	})

	t.Run("should fail with invalid config", func(t *testing.T) {
		testCases := []struct {
			name        string
			errContains string
			config      ASTCacheConfig
		}{
			{
				name:        "zero capacity",
				config:      ASTCacheConfig{L1CacheCapacity: 0, L1CacheTTL: time.Hour, L2CacheBaseDir: t.TempDir()},
				errContains: "L1CacheCapacity must be positive",
			},
			{
				name:        "zero TTL",
				config:      ASTCacheConfig{L1CacheCapacity: 100, L1CacheTTL: 0, L2CacheBaseDir: t.TempDir()},
				errContains: "L1CacheTTL must be a positive duration",
			},
			{
				name:        "missing L2 directory",
				config:      ASTCacheConfig{L1CacheCapacity: 100, L1CacheTTL: time.Hour, L2CacheBaseDir: ""},
				errContains: "L2CacheBaseDir must be provided",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				service, err := NewASTCacheService(context.Background(), tc.config)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Nil(t, service)
			})
		}
	})
}

func setupService(t *testing.T, l1TTL time.Duration) (*ASTCacheService, *mockL2Cache) {
	t.Helper()

	mockL2 := newMockL2Cache()
	l1Cache := otter.Must(&otter.Options[string, *ast_domain.TemplateAST]{
		MaximumSize:      100,
		ExpiryCalculator: otter.ExpiryWriting[string, *ast_domain.TemplateAST](l1TTL),
	})
	t.Cleanup(func() { l1Cache.StopAllGoroutines() })

	multiLevelCache := newMultiLevelCache(l1Cache, mockL2, l1TTL, nil)
	service := &ASTCacheService{cache: multiLevelCache}

	return service, mockL2
}

func TestASTCacheService_Lifecycle(t *testing.T) {
	ctx := context.Background()
	defaultL1TTL := 1 * time.Hour
	ast1 := newTestAST("ast1")

	t.Run("Set and Get should work", func(t *testing.T) {
		service, mockL2 := setupService(t, defaultL1TTL)
		entry1 := newTestEntry("ast1")

		err := service.Set(ctx, "key1", entry1)
		require.NoError(t, err)

		l2Entry, l2Err := mockL2.Get(ctx, "key1")
		require.NoError(t, l2Err)
		assert.Equal(t, "ast1", l2Entry.RootNodes[0].Attributes[0].Value)
		assert.Nil(t, l2Entry.ExpiresAtUnixNano)

		retrievedAST, err := service.Get(ctx, "key1")
		require.NoError(t, err)
		require.NotNil(t, retrievedAST)
		assert.Equal(t, "ast1", retrievedAST.AST.RootNodes[0].Attributes[0].Value)
	})

	t.Run("SetWithTTL and Get should work", func(t *testing.T) {
		service, mockL2 := setupService(t, defaultL1TTL)
		customTTL := 500 * time.Millisecond
		entry1 := newTestEntry("ast1")

		err := service.SetWithTTL(ctx, "key1", entry1, customTTL)
		require.NoError(t, err)

		l2Entry, l2Err := mockL2.Get(ctx, "key1")
		require.NoError(t, l2Err)
		require.NotNil(t, l2Entry.ExpiresAtUnixNano)
		assert.WithinDuration(t, time.Now().Add(customTTL), time.Unix(0, *l2Entry.ExpiresAtUnixNano), 50*time.Millisecond)

		retrievedAST, err := service.Get(ctx, "key1")
		require.NoError(t, err)
		assert.Equal(t, "ast1", retrievedAST.AST.RootNodes[0].Attributes[0].Value)

		time.Sleep(customTTL + 50*time.Millisecond)
		_, err = service.Get(ctx, "key1")
		assert.ErrorIs(t, err, ast_domain.ErrCacheMiss)
	})

	t.Run("Get should fall back to L2 and populate L1", func(t *testing.T) {
		service, mockL2 := setupService(t, defaultL1TTL)

		err := mockL2.Set(ctx, "key1", ast1)
		require.NoError(t, err)

		retrievedAST, err := service.Get(ctx, "key1")
		require.NoError(t, err)
		assert.Equal(t, "ast1", retrievedAST.AST.RootNodes[0].Attributes[0].Value)

		multiCache, ok := service.cache.(*multiLevelCache)
		require.True(t, ok, "cache should be *multiLevelCache")
		_, found := multiCache.l1Cache.GetIfPresent("key1")
		assert.True(t, found, "L1 cache should be populated after a successful L2 fetch")
	})

	t.Run("Delete should remove from both caches", func(t *testing.T) {
		service, mockL2 := setupService(t, defaultL1TTL)
		entry1 := newTestEntry("ast1")

		err := service.Set(ctx, "key1", entry1)
		require.NoError(t, err)

		_, getErr := service.Get(ctx, "key1")
		require.NoError(t, getErr)
		_, l2GetErr := mockL2.Get(ctx, "key1")
		require.NoError(t, l2GetErr)

		err = service.Delete(ctx, "key1")
		require.NoError(t, err)

		_, getErr = service.Get(ctx, "key1")
		assert.ErrorIs(t, getErr, ast_domain.ErrCacheMiss)
		_, l2GetErr = mockL2.Get(ctx, "key1")
		assert.ErrorIs(t, l2GetErr, ast_domain.ErrCacheMiss)
	})
}

func TestASTCacheService_TTLSynchronisation(t *testing.T) {
	ctx := context.Background()
	ast1 := newTestAST("ast1")

	t.Run("L1 TTL should be capped by L2 TTL when L2 is shorter", func(t *testing.T) {
		l1DefaultTTL := 1 * time.Hour
		service, mockL2 := setupService(t, l1DefaultTTL)
		l2RemainingTTL := 100 * time.Millisecond

		err := mockL2.SetWithTTL(ctx, "key1", ast1, l2RemainingTTL)
		require.NoError(t, err)

		_, err = service.Get(ctx, "key1")
		require.NoError(t, err)

		multiCache, ok := service.cache.(*multiLevelCache)
		require.True(t, ok, "cache should be *multiLevelCache")
		entry, found := multiCache.l1Cache.GetEntry("key1")
		require.True(t, found)
		assert.LessOrEqual(t, entry.ExpiresAfter(), l2RemainingTTL, "L1 TTL should be capped by the shorter L2 TTL")
		assert.Greater(t, entry.ExpiresAfter(), time.Duration(0), "L1 TTL should be positive")
	})

	t.Run("L1 TTL should be capped by its own default when L2 is longer", func(t *testing.T) {
		l1DefaultTTL := 100 * time.Millisecond
		service, mockL2 := setupService(t, l1DefaultTTL)
		l2RemainingTTL := 1 * time.Hour

		err := mockL2.SetWithTTL(ctx, "key1", ast1, l2RemainingTTL)
		require.NoError(t, err)

		_, err = service.Get(ctx, "key1")
		require.NoError(t, err)

		multiCache, ok := service.cache.(*multiLevelCache)
		require.True(t, ok, "cache should be *multiLevelCache")
		entry, found := multiCache.l1Cache.GetEntry("key1")
		require.True(t, found)
		assert.LessOrEqual(t, entry.ExpiresAfter(), l1DefaultTTL, "L1 TTL should be capped by its own default TTL")
		assert.Greater(t, entry.ExpiresAfter(), time.Duration(0), "L1 TTL should be positive")
	})

	t.Run("L1 uses its default TTL when L2 entry never expires", func(t *testing.T) {
		l1DefaultTTL := 100 * time.Millisecond
		service, mockL2 := setupService(t, l1DefaultTTL)

		err := mockL2.Set(ctx, "key1", ast1)
		require.NoError(t, err)

		_, err = service.Get(ctx, "key1")
		require.NoError(t, err)

		multiCache, ok := service.cache.(*multiLevelCache)
		require.True(t, ok, "cache should be *multiLevelCache")
		entry, found := multiCache.l1Cache.GetEntry("key1")
		require.True(t, found)
		assert.LessOrEqual(t, entry.ExpiresAfter(), l1DefaultTTL, "L1 TTL should be its default")
	})
}

func TestASTCacheService_WithRealL2Cache(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := ASTCacheConfig{
		L1CacheCapacity: 10,
		L1CacheTTL:      500 * time.Millisecond,
		L2CacheBaseDir:  tempDir,
	}

	service, err := NewASTCacheService(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(func() { service.Shutdown(context.Background()) })
	require.NotNil(t, service)

	entry1 := newTestEntry("real-l2")

	err = service.SetWithTTL(ctx, "real-key", entry1, 100*time.Millisecond)
	require.NoError(t, err)

	retrieved, err := service.Get(ctx, "real-key")
	require.NoError(t, err)
	assert.Equal(t, "real-l2", retrieved.AST.RootNodes[0].Attributes[0].Value)

	time.Sleep(150 * time.Millisecond)

	_, err = service.Get(ctx, "real-key")
	assert.ErrorIs(t, err, ast_domain.ErrCacheMiss)

	multiCache, ok := service.cache.(*multiLevelCache)
	require.True(t, ok, "cache should be *multiLevelCache")
	l2Cache, ok := multiCache.l2Cache.(*fbsFileCache)
	require.True(t, ok, "l2Cache should be *fbsFileCache")
	filePath := l2Cache.getFilePath("real-key")

	deadline := time.Now().Add(500 * time.Millisecond)
	var statErr error
	for {
		_, statErr = os.Stat(filePath)
		if os.IsNotExist(statErr) || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	assert.True(t, os.IsNotExist(statErr), "The physical cache file should have been deleted by the self-healing Get")
}

func TestASTCacheService_ComplexASTDiskRoundTrip(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	config := ASTCacheConfig{
		L1CacheCapacity: 10,
		L1CacheTTL:      1 * time.Hour,
		L2CacheBaseDir:  tempDir,
	}

	service, err := NewASTCacheService(context.Background(), config)
	require.NoError(t, err)
	t.Cleanup(func() { service.Shutdown(context.Background()) })

	sourcePath := "/app/pages/dashboard.pk"
	metadataJSON := `{"title":"Dashboard","customTags":["card","chart"]}`

	original := &ast_domain.CachedASTEntry{
		AST: &ast_domain.TemplateAST{
			SourcePath: &sourcePath,
			Metadata:   &metadataJSON,
			Tidied:     true,
			SourceSize: 12345,
			RootNodes: []*ast_domain.TemplateNode{
				{
					NodeType: ast_domain.NodeElement,
					TagName:  "div",
					Attributes: []ast_domain.HTMLAttribute{
						{Name: "id", Value: "app"},
						{Name: "class", Value: "container mx-auto"},
					},
					Children: []*ast_domain.TemplateNode{
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "h1",
							DirText: &ast_domain.Directive{
								Type:          ast_domain.DirectiveText,
								RawExpression: "state.Title",
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "p",
							DirIf: &ast_domain.Directive{
								Type:          ast_domain.DirectiveIf,
								RawExpression: "state.ShowDescription",
							},
							RichText: []ast_domain.TextPart{
								{IsLiteral: true, Literal: "Welcome, "},
								{IsLiteral: false, RawExpression: "state.UserName"},
								{IsLiteral: true, Literal: "!"},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "p",
							DirElse: &ast_domain.Directive{
								Type: ast_domain.DirectiveElse,
							},
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "No description available"},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "ul",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "class", Value: "item-list"},
							},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "li",
									DirFor: &ast_domain.Directive{
										Type:          ast_domain.DirectiveFor,
										Arg:           "item",
										RawExpression: "item in state.Items",
									},
									DirKey: &ast_domain.Directive{
										Type:          ast_domain.DirectiveKey,
										RawExpression: "item.ID",
									},
									DirShow: &ast_domain.Directive{
										Type:          ast_domain.DirectiveShow,
										RawExpression: "item.Visible",
									},
									RichText: []ast_domain.TextPart{
										{IsLiteral: false, RawExpression: "item.Name"},
									},
									DynamicAttributes: []ast_domain.DynamicAttribute{
										{Name: "data-id", RawExpression: "item.ID"},
									},
								},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "button",
							OnEvents: map[string][]ast_domain.Directive{
								"click": {
									{Type: ast_domain.DirectiveOn, Arg: "click", RawExpression: "handleClick", EventModifiers: []string{"prevent"}},
								},
								"mouseover": {
									{Type: ast_domain.DirectiveOn, Arg: "mouseover", RawExpression: "handleHover", IsStaticEvent: true},
								},
							},
							Binds: map[string]*ast_domain.Directive{
								"disabled": {Type: ast_domain.DirectiveBind, Arg: "disabled", RawExpression: "state.IsLoading"},
								"class":    {Type: ast_domain.DirectiveBind, Arg: "class", RawExpression: "state.BtnClass"},
							},
							Children: []*ast_domain.TemplateNode{
								{NodeType: ast_domain.NodeText, TextContent: "Submit"},
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "input",
							Attributes: []ast_domain.HTMLAttribute{
								{Name: "type", Value: "text"},
								{Name: "name", Value: "search"},
							},
							DirModel: &ast_domain.Directive{
								Type:          ast_domain.DirectiveModel,
								RawExpression: "state.SearchQuery",
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "span",
							DirHTML: &ast_domain.Directive{
								Type:          ast_domain.DirectiveHTML,
								RawExpression: "state.RichContent",
							},
						},
						{
							NodeType: ast_domain.NodeElement,
							TagName:  "div",
							DirRef: &ast_domain.Directive{
								Type:          ast_domain.DirectiveRef,
								RawExpression: "container",
							},
							Children: []*ast_domain.TemplateNode{
								{
									NodeType: ast_domain.NodeElement,
									TagName:  "section",
									DirSlot: &ast_domain.Directive{
										Type:          ast_domain.DirectiveSlot,
										RawExpression: "header",
									},
									Children: []*ast_domain.TemplateNode{
										{NodeType: ast_domain.NodeText, TextContent: "Slot content"},
									},
								},
							},
						},
						{NodeType: ast_domain.NodeComment, TextContent: "End of dashboard"},
					},
				},
			},
		},
		Metadata: metadataJSON,
	}

	err = service.Set(ctx, "complex-key", original)
	require.NoError(t, err)

	multiCache, ok := service.cache.(*multiLevelCache)
	require.True(t, ok)
	multiCache.l1Cache.Invalidate("complex-key")

	retrieved, err := service.Get(ctx, "complex-key")
	require.NoError(t, err)
	require.NotNil(t, retrieved)
	require.NotNil(t, retrieved.AST)

	ast := retrieved.AST
	require.NotNil(t, ast.SourcePath)
	assert.Equal(t, sourcePath, *ast.SourcePath)
	assert.True(t, ast.Tidied)
	assert.Equal(t, int64(12345), ast.SourceSize)

	require.Len(t, ast.RootNodes, 1)
	root := ast.RootNodes[0]
	assert.Equal(t, "div", root.TagName)
	require.Len(t, root.Attributes, 2)
	assert.Equal(t, "id", root.Attributes[0].Name)
	assert.Equal(t, "app", root.Attributes[0].Value)
	assert.Equal(t, "class", root.Attributes[1].Name)
	assert.Equal(t, "container mx-auto", root.Attributes[1].Value)

	require.Len(t, root.Children, 9)

	h1 := root.Children[0]
	assert.Equal(t, "h1", h1.TagName)
	require.NotNil(t, h1.DirText)
	assert.Equal(t, "state.Title", h1.DirText.RawExpression)

	pIf := root.Children[1]
	assert.Equal(t, "p", pIf.TagName)
	require.NotNil(t, pIf.DirIf)
	assert.Equal(t, "state.ShowDescription", pIf.DirIf.RawExpression)
	require.Len(t, pIf.RichText, 3)
	assert.True(t, pIf.RichText[0].IsLiteral)
	assert.Equal(t, "Welcome, ", pIf.RichText[0].Literal)
	assert.False(t, pIf.RichText[1].IsLiteral)
	assert.Equal(t, "state.UserName", pIf.RichText[1].RawExpression)
	assert.True(t, pIf.RichText[2].IsLiteral)
	assert.Equal(t, "!", pIf.RichText[2].Literal)

	pElse := root.Children[2]
	require.NotNil(t, pElse.DirElse)
	require.Len(t, pElse.Children, 1)
	assert.Equal(t, "No description available", pElse.Children[0].TextContent)

	ul := root.Children[3]
	assert.Equal(t, "ul", ul.TagName)
	require.Len(t, ul.Children, 1)
	li := ul.Children[0]
	require.NotNil(t, li.DirFor)
	assert.Equal(t, "item", li.DirFor.Arg)
	assert.Equal(t, "item in state.Items", li.DirFor.RawExpression)
	require.NotNil(t, li.DirKey)
	assert.Equal(t, "item.ID", li.DirKey.RawExpression)
	require.NotNil(t, li.DirShow)
	assert.Equal(t, "item.Visible", li.DirShow.RawExpression)
	require.Len(t, li.DynamicAttributes, 1)
	assert.Equal(t, "data-id", li.DynamicAttributes[0].Name)
	assert.Equal(t, "item.ID", li.DynamicAttributes[0].RawExpression)

	button := root.Children[4]
	assert.Equal(t, "button", button.TagName)
	require.Contains(t, button.OnEvents, "click")
	require.Len(t, button.OnEvents["click"], 1)
	assert.Equal(t, "handleClick", button.OnEvents["click"][0].RawExpression)
	assert.Equal(t, []string{"prevent"}, button.OnEvents["click"][0].EventModifiers)
	require.Contains(t, button.OnEvents, "mouseover")
	require.Len(t, button.OnEvents["mouseover"], 1)
	assert.True(t, button.OnEvents["mouseover"][0].IsStaticEvent)
	require.Contains(t, button.Binds, "disabled")
	assert.Equal(t, "state.IsLoading", button.Binds["disabled"].RawExpression)
	require.Contains(t, button.Binds, "class")
	assert.Equal(t, "state.BtnClass", button.Binds["class"].RawExpression)
	require.Len(t, button.Children, 1)
	assert.Equal(t, "Submit", button.Children[0].TextContent)

	input := root.Children[5]
	assert.Equal(t, "input", input.TagName)
	require.NotNil(t, input.DirModel)
	assert.Equal(t, "state.SearchQuery", input.DirModel.RawExpression)

	span := root.Children[6]
	require.NotNil(t, span.DirHTML)
	assert.Equal(t, "state.RichContent", span.DirHTML.RawExpression)

	refDiv := root.Children[7]
	require.NotNil(t, refDiv.DirRef)
	assert.Equal(t, "container", refDiv.DirRef.RawExpression)
	require.Len(t, refDiv.Children, 1)
	section := refDiv.Children[0]
	require.NotNil(t, section.DirSlot)
	assert.Equal(t, "header", section.DirSlot.RawExpression)

	comment := root.Children[8]
	assert.Equal(t, ast_domain.NodeComment, comment.NodeType)
	assert.Equal(t, "End of dashboard", comment.TextContent)

	assert.Equal(t, metadataJSON, retrieved.Metadata)
}
