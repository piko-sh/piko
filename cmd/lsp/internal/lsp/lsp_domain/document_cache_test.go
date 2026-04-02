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

package lsp_domain

import (
	"fmt"
	"sync"
	"testing"

	"go.lsp.dev/protocol"
)

func TestDocumentCache_GetNonExistent_ReturnsNotFound(t *testing.T) {
	cache := NewDocumentCache()

	content, found := cache.Get("file:///nonexistent.pk")

	if found {
		t.Error("expected found=false for nonexistent document")
	}
	if content != nil {
		t.Errorf("expected nil content, got %v", content)
	}
}

func TestDocumentCache_Set_StoresContent(t *testing.T) {
	cache := NewDocumentCache()
	uri := protocol.DocumentURI("file:///test.pk")
	expectedContent := []byte("<template>hello</template>")

	cache.Set(uri, expectedContent)

	content, found := cache.Get(uri)
	if !found {
		t.Error("expected found=true after Set")
	}
	if string(content) != string(expectedContent) {
		t.Errorf("expected content %q, got %q", expectedContent, content)
	}
}

func TestDocumentCache_Set_OverwritesExisting(t *testing.T) {
	cache := NewDocumentCache()
	uri := protocol.DocumentURI("file:///test.pk")
	initialContent := []byte("<template>initial</template>")
	updatedContent := []byte("<template>updated</template>")

	cache.Set(uri, initialContent)
	cache.Set(uri, updatedContent)

	content, found := cache.Get(uri)
	if !found {
		t.Error("expected found=true after Set")
	}
	if string(content) != string(updatedContent) {
		t.Errorf("expected updated content %q, got %q", updatedContent, content)
	}
}

func TestDocumentCache_Delete_RemovesEntry(t *testing.T) {
	cache := NewDocumentCache()
	uri := protocol.DocumentURI("file:///test.pk")
	content := []byte("<template>test</template>")

	cache.Set(uri, content)
	cache.Delete(uri)

	_, found := cache.Get(uri)
	if found {
		t.Error("expected found=false after Delete")
	}
}

func TestDocumentCache_Delete_NonExistent_NoError(t *testing.T) {
	cache := NewDocumentCache()

	cache.Delete("file:///nonexistent.pk")
}

func TestDocumentCache_ConcurrentSetGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	cache := NewDocumentCache()
	uri := protocol.DocumentURI("file:///test.pk")
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				content := fmt.Appendf(nil, "content-%d-%d", id, j)
				cache.Set(uri, content)
			}
		}(i)
	}

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				_, _ = cache.Get(uri)
			}
		}()
	}

	wg.Wait()
}

func TestDocumentCache_ConcurrentSetDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	cache := NewDocumentCache()
	uri := protocol.DocumentURI("file:///test.pk")
	const goroutines = 50
	const iterations = 500

	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := range goroutines {
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				content := fmt.Appendf(nil, "content-%d-%d", id, j)
				cache.Set(uri, content)
			}
		}(i)
	}

	for range goroutines {
		go func() {
			defer wg.Done()
			for range iterations {
				cache.Delete(uri)
			}
		}()
	}

	wg.Wait()
}

func TestDocumentCache_ConcurrentMultipleURIs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping race condition test in short mode")
	}

	cache := NewDocumentCache()
	const numURIs = 10
	const goroutinesPerURI = 20
	const iterations = 200

	uris := make([]protocol.DocumentURI, numURIs)
	for i := range numURIs {
		uris[i] = protocol.DocumentURI(fmt.Sprintf("file:///test%d.pk", i))
	}

	var wg sync.WaitGroup
	wg.Add(numURIs * goroutinesPerURI * 3)

	for _, uri := range uris {

		for i := range goroutinesPerURI {
			go func(id int) {
				defer wg.Done()
				for j := range iterations {
					content := fmt.Appendf(nil, "content-%d-%d", id, j)
					cache.Set(uri, content)
				}
			}(i)
		}

		for range goroutinesPerURI {
			go func() {
				defer wg.Done()
				for range iterations {
					_, _ = cache.Get(uri)
				}
			}()
		}

		for range goroutinesPerURI {
			go func() {
				defer wg.Done()
				for range iterations {
					cache.Delete(uri)
				}
			}()
		}
	}

	wg.Wait()
}

func TestDocumentCache_GetAllURIs_Empty(t *testing.T) {
	cache := NewDocumentCache()
	uris := cache.GetAllURIs()
	if len(uris) != 0 {
		t.Errorf("expected empty, got %d URIs", len(uris))
	}
}

func TestDocumentCache_GetAllURIs_ReturnsAllStored(t *testing.T) {
	cache := NewDocumentCache()

	uri1 := protocol.DocumentURI("file:///a.pk")
	uri2 := protocol.DocumentURI("file:///b.pk")
	uri3 := protocol.DocumentURI("file:///c.pk")

	cache.Set(uri1, []byte("aaa"))
	cache.Set(uri2, []byte("bbb"))
	cache.Set(uri3, []byte("ccc"))

	uris := cache.GetAllURIs()
	if len(uris) != 3 {
		t.Fatalf("expected 3 URIs, got %d", len(uris))
	}

	found := make(map[protocol.DocumentURI]bool)
	for _, u := range uris {
		found[u] = true
	}
	for _, expected := range []protocol.DocumentURI{uri1, uri2, uri3} {
		if !found[expected] {
			t.Errorf("missing URI %s in result", expected)
		}
	}
}

func TestDocumentCache_GetAllURIs_AfterDelete(t *testing.T) {
	cache := NewDocumentCache()

	uri1 := protocol.DocumentURI("file:///a.pk")
	uri2 := protocol.DocumentURI("file:///b.pk")

	cache.Set(uri1, []byte("aaa"))
	cache.Set(uri2, []byte("bbb"))
	cache.Delete(uri1)

	uris := cache.GetAllURIs()
	if len(uris) != 1 {
		t.Fatalf("expected 1 URI after delete, got %d", len(uris))
	}
	if uris[0] != uri2 {
		t.Errorf("expected %s, got %s", uri2, uris[0])
	}
}
