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

package ast_domain

// Defines port interfaces for AST caching that external adapters implement to store and retrieve parsed templates.
// Provides ASTCache and ASTCacheService contracts with support for TTL-based expiration and request-specific metadata bundling.

import (
	"context"
	"errors"
	"time"
)

// CachedASTEntry is the encodable container for a cached page rendering.
// It bundles the generated AST with its corresponding request-specific metadata.
type CachedASTEntry struct {
	// AST is the parsed template abstract syntax tree.
	AST *TemplateAST

	// Metadata holds extra data about the cached entry as a JSON string.
	Metadata string
}

// ErrCacheMiss is returned when a cache lookup finds no value for the key.
var ErrCacheMiss = errors.New("cache: key not found")

// ASTCache provides a way to store and retrieve TemplateAST objects.
type ASTCache interface {
	// Get retrieves a TemplateAST from the cache.
	//
	// Takes key (string) which identifies the cached template.
	//
	// Returns *TemplateAST which is the cached template.
	// Returns error when the key is not found (ErrCacheMiss) or retrieval fails.
	Get(ctx context.Context, key string) (*TemplateAST, error)

	// Set stores a TemplateAST in the cache.
	//
	// Takes key (string) which identifies the cached template.
	// Takes ast (*TemplateAST) which is the parsed template to store.
	//
	// Returns error when the cache operation fails.
	Set(ctx context.Context, key string, ast *TemplateAST) error

	// SetWithTTL stores a TemplateAST in the cache with a custom time-to-live.
	//
	// Takes key (string) which identifies the cached template.
	// Takes cache (*TemplateAST) which is the parsed template to store.
	// Takes ttl (time.Duration) which sets how long the entry should be kept.
	//
	// Returns error when the cache operation fails.
	//
	// Cache types that do not support TTLs should fall back to their standard
	// Set behaviour.
	SetWithTTL(ctx context.Context, key string, cache *TemplateAST, ttl time.Duration) error

	// Delete removes a TemplateAST from the cache.
	//
	// Takes key (string) which identifies the template to remove.
	//
	// Returns error when the deletion fails.
	Delete(ctx context.Context, key string) error
}

// ASTCacheService defines the contract for a cache that stores CachedASTEntry
// bundles. It extends ASTCache by including request-specific metadata alongside
// the AST.
type ASTCacheService interface {
	// Get retrieves a TemplateAST from the cache.
	//
	// Takes key (string) which identifies the cached item.
	//
	// Returns *CachedASTEntry which contains the cached template.
	// Returns error when the item is not found (ErrCacheMiss).
	Get(ctx context.Context, key string) (*CachedASTEntry, error)

	// Set stores a TemplateAST entry in the cache.
	//
	// Takes key (string) which identifies the cached entry.
	// Takes ast (*CachedASTEntry) which contains the template AST to store.
	//
	// Returns error when the cache operation fails.
	Set(ctx context.Context, key string, ast *CachedASTEntry) error

	// SetWithTTL stores a TemplateAST in the cache with a custom time-to-live.
	//
	// Takes key (string) which identifies the cache entry.
	// Takes cache (*CachedASTEntry) which is the entry to store.
	// Takes ttl (time.Duration) which sets how long the entry stays valid.
	//
	// Returns error when the storage fails.
	//
	// Cache types that do not support TTLs should fall back to their normal Set
	// behaviour.
	SetWithTTL(ctx context.Context, key string, cache *CachedASTEntry, ttl time.Duration) error

	// Delete removes a TemplateAST from the cache.
	//
	// Takes key (string) which identifies the template to remove.
	//
	// Returns error when the deletion fails.
	Delete(ctx context.Context, key string) error

	// Shutdown stops the server in a controlled way, allowing current requests
	// to finish.
	Shutdown(ctx context.Context)
}
