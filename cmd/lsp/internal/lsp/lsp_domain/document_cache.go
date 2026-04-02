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
	"sync"

	"go.lsp.dev/protocol"
)

// DocumentCache is a thread-safe in-memory store for open documents.
type DocumentCache struct {
	// content maps document URIs to their file contents.
	content map[protocol.DocumentURI][]byte

	// mu guards access to the content map.
	mu sync.Mutex
}

// NewDocumentCache creates a new cache for storing parsed document data.
//
// Returns *DocumentCache which is an empty cache ready for use.
func NewDocumentCache() *DocumentCache {
	return &DocumentCache{
		content: make(map[protocol.DocumentURI][]byte),
		mu:      sync.Mutex{},
	}
}

// Get retrieves a document's content from the cache by its URI.
//
// Takes uri (protocol.DocumentURI) which identifies the document to retrieve.
//
// Returns []byte which contains the document content if found.
// Returns bool which indicates whether the document was found in the cache.
//
// Safe for concurrent use.
func (c *DocumentCache) Get(uri protocol.DocumentURI) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	content, found := c.content[uri]
	return content, found
}

// Set stores a document's content in the cache.
//
// Takes uri (protocol.DocumentURI) which identifies the document.
// Takes content ([]byte) which is the content to store.
//
// Safe for concurrent use.
func (c *DocumentCache) Set(uri protocol.DocumentURI, content []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.content[uri] = content
}

// Delete removes a document from the cache.
//
// Takes uri (protocol.DocumentURI) which identifies the document to remove.
//
// Safe for concurrent use.
func (c *DocumentCache) Delete(uri protocol.DocumentURI) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.content, uri)
}

// GetAllURIs returns a copy of all document URIs currently in the cache.
//
// Returns []protocol.DocumentURI which contains all cached document URIs.
//
// Safe for concurrent use.
func (c *DocumentCache) GetAllURIs() []protocol.DocumentURI {
	c.mu.Lock()
	defer c.mu.Unlock()
	uris := make([]protocol.DocumentURI, 0, len(c.content))
	for uri := range c.content {
		uris = append(uris, uri)
	}
	return uris
}
