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
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

type mockASTCache struct {
	store map[string]*ast_domain.CachedASTEntry
	mu    sync.RWMutex
}

func newMockASTCache() *mockASTCache {
	return &mockASTCache{
		store: make(map[string]*ast_domain.CachedASTEntry),
	}
}

func (m *mockASTCache) Get(_ context.Context, key string) (*ast_domain.CachedASTEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	entry, ok := m.store[key]
	if !ok {
		return nil, ast_domain.ErrCacheMiss
	}
	return entry, nil
}

func (m *mockASTCache) Set(_ context.Context, key string, entry *ast_domain.CachedASTEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[key] = entry
	return nil
}

func (m *mockASTCache) SetWithTTL(_ context.Context, key string, entry *ast_domain.CachedASTEntry, _ time.Duration) error {
	return m.Set(context.Background(), key, entry)
}

func (m *mockASTCache) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, key)
	return nil
}

func TestCachedASTIndependenceFromOriginal(t *testing.T) {
	t.Parallel()

	arena := ast_domain.GetArena()
	freshAST := ast_domain.GetTemplateAST()
	freshAST.SetArena(arena)
	freshAST.RootNodes = arena.GetRootNodesSlice(1)

	node := arena.GetNode()
	node.NodeType = ast_domain.NodeElement
	node.TagName = "div"

	dw := arena.GetDirectWriter()
	dw.SetName("class")
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = append((*bufferPointer)[:0], "nav-item"...)
	dw.AppendPooledBytes(bufferPointer)
	_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
	node.AttributeWriters = append(node.AttributeWriters, dw)

	freshAST.RootNodes = append(freshAST.RootNodes, node)

	originalDW := freshAST.RootNodes[0].AttributeWriters[0]
	var originalOutput []byte
	originalOutput = originalDW.WriteTo(originalOutput)
	require.Equal(t, "nav-item", string(originalOutput), "Original should have correct value")

	clonedAST := freshAST.DeepClone()

	clonedDW := clonedAST.RootNodes[0].AttributeWriters[0]
	var clonedOutput []byte
	clonedOutput = clonedDW.WriteTo(clonedOutput)
	require.Equal(t, "nav-item", string(clonedOutput), "Cloned AST should have correct value")

	ast_domain.PutTree(freshAST)

	for range 10 {
		newBufPtr := ast_domain.GetByteBuf()
		*newBufPtr = append((*newBufPtr)[:0], "text-emphasis"...)
		ast_domain.PutByteBuf(newBufPtr)
	}

	var finalClonedOutput []byte
	finalClonedOutput = clonedDW.WriteTo(finalClonedOutput)
	assert.Equal(t, "nav-item", string(finalClonedOutput),
		"Cloned AST should retain correct value after original is returned to pool")

	ast_domain.PutTree(clonedAST)
}

func TestConcurrentCacheAccessWithPoolReuse(t *testing.T) {
	t.Parallel()

	const goroutines = 20
	const iterations = 100

	for range goroutines {
		buffer := ast_domain.GetByteBuf()
		ast_domain.PutByteBuf(buffer)
	}

	cache := newMockASTCache()

	arena := ast_domain.GetArena()
	initialAST := ast_domain.GetTemplateAST()
	initialAST.SetArena(arena)
	initialAST.RootNodes = arena.GetRootNodesSlice(1)

	node := arena.GetNode()
	node.NodeType = ast_domain.NodeElement
	node.TagName = "span"

	dw := arena.GetDirectWriter()
	dw.SetName("class")
	bufferPointer := ast_domain.GetByteBuf()
	*bufferPointer = append((*bufferPointer)[:0], "test-class"...)
	dw.AppendPooledBytes(bufferPointer)
	_, node.AttributeWriters = arena.GetAttrWriterSlice(1)
	node.AttributeWriters = append(node.AttributeWriters, dw)

	initialAST.RootNodes = append(initialAST.RootNodes, node)

	entry := &ast_domain.CachedASTEntry{
		AST:      initialAST.DeepClone(),
		Metadata: "{}",
	}
	_ = cache.Set(context.Background(), "test-key", entry)

	ast_domain.PutTree(initialAST)

	var wg sync.WaitGroup
	errorChan := make(chan error, goroutines*iterations)

	for g := range goroutines {
		goroutineID := g
		wg.Go(func() {
			for i := range iterations {

				cached, err := cache.Get(context.Background(), "test-key")
				if err != nil {
					errorChan <- err
					continue
				}

				clone := cached.AST.DeepClone()

				if len(clone.RootNodes) > 0 && len(clone.RootNodes[0].AttributeWriters) > 0 {
					var output []byte
					output = clone.RootNodes[0].AttributeWriters[0].WriteTo(output)
					if string(output) != "test-class" {
						t.Errorf("Corruption detected: expected 'test-class', got '%s' (goroutine %d, iteration %d)",
							string(output), goroutineID, i)
					}
				}

				ast_domain.PutTree(clone)

				bufferPointer := ast_domain.GetByteBuf()
				*bufferPointer = append((*bufferPointer)[:0], "corrupted-value-longer"...)
				ast_domain.PutByteBuf(bufferPointer)
			}
		})
	}

	wg.Wait()
	close(errorChan)

	for err := range errorChan {
		if err != nil && !errors.Is(err, ast_domain.ErrCacheMiss) {
			t.Error(err)
		}
	}
}

type recordedCacheWrite struct {
	cause   error
	carrier *daemon_dto.PikoRequestCtx
	key     string
	ttl     time.Duration
}

type gatedRecordingASTCache struct {
	*mockASTCache
	gate   chan struct{}
	writes chan recordedCacheWrite
}

func (c *gatedRecordingASTCache) SetWithTTL(
	ctx context.Context, key string, entry *ast_domain.CachedASTEntry, ttl time.Duration,
) error {
	<-c.gate
	c.writes <- recordedCacheWrite{
		cause:   context.Cause(ctx),
		carrier: daemon_dto.PikoRequestCtxFromContext(ctx),
		key:     key,
		ttl:     ttl,
	}
	return c.Set(ctx, key, entry)
}

func (*gatedRecordingASTCache) Shutdown(context.Context) {}

func TestCachingManifestRunner_BackgroundWriteOutlivesTheRequest(t *testing.T) {
	cache := &gatedRecordingASTCache{
		mockASTCache: newMockASTCache(),
		gate:         make(chan struct{}),
		writes:       make(chan recordedCacheWrite, 1),
	}
	entry := &PageEntry{}
	entry.OriginalSourcePath = "pages/cached.pk"
	entry.HasCachePolicy = true
	entry.SetCachePolicyFunc(func(*templater_dto.RequestData) templater_dto.CachePolicy {
		return templater_dto.CachePolicy{Enabled: true, MaxAgeSeconds: 90}
	})
	next := &mockManifestRunner{
		getPageEntryFunction: func(context.Context, string) (templater_domain.PageEntryView, error) {
			return entry, nil
		},
		runPageFunction: func(context.Context, templater_dto.PageDefinition, *http.Request) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
			return &ast_domain.TemplateAST{}, templater_dto.InternalMetadata{Title: "Cached"}, "", nil
		},
	}
	runner := NewCachingManifestRunner(next, cache)

	pctx := daemon_dto.AcquirePikoRequestCtx()
	pctx.Locale = "en"
	requestCtx, cancelRequest := context.WithCancelCause(daemon_dto.WithPikoRequestCtx(context.Background(), pctx))
	request := newRequestWithChiCtx(http.MethodGet, "/cached")
	request = request.WithContext(daemon_dto.WithPikoRequestCtx(request.Context(), pctx))
	pageDef := templater_dto.PageDefinition{OriginalPath: "pages/cached.pk", NormalisedPath: "/cached"}

	_, metadata, _, err := runner.RunPage(requestCtx, pageDef, request)
	require.NoError(t, err)
	assert.Equal(t, "Cached", metadata.Title)

	cancelRequest(errors.New("client went away"))
	daemon_dto.ReleasePikoRequestCtx(pctx)
	close(cache.gate)

	write := <-cache.writes
	assert.Equal(t, 90*time.Second, write.ttl, "the TTL comes from the page's cache policy")
	require.NoError(t, write.cause, "the cache write must outlive the request")
	require.NotNil(t, write.carrier)
	assert.NotSame(t, pctx, write.carrier, "the cache write must not read the pooled carrier")
	assert.Equal(t, "en", write.carrier.Locale)
	assert.NotEmpty(t, write.key)
}
