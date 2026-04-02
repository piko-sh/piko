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
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/ast/ast_domain"
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
