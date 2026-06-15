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

package cssinliner

import (
	"hash/maphash"
	"sync"
	"sync/atomic"

	"piko.sh/piko/internal/esbuild/css_ast"
	es_logger "piko.sh/piko/internal/esbuild/logger"
)

const (
	// parseCacheMaxEntries bounds the number of cached parses.
	//
	// This stops a long-lived owner (e.g. the dev-interpreted watch daemon, which holds one
	// Processor across many rebuilds and edits) accreting stale parses without limit. On
	// overflow the cache is wiped wholesale; content hashing makes it self-rebuilding, so a
	// wipe only costs a re-parse of the sheets still in use.
	parseCacheMaxEntries = 10000
)

var (
	// cssHashSeed seeds the content hash. The cache is process-local and keys never leave
	// the process or get persisted, so a per-process random seed is sufficient (and the hit
	// path compares the full content anyway, so the hash is only ever a bucket index).
	cssHashSeed = maphash.MakeSeed()
)

// parseCacheEntry is one cached CSS parse.
//
// It holds the read-only master AST (never handed out directly, every consumer receives a
// CloneAST copy), the source content (compared on every hit so a hash collision can never
// serve the wrong AST), and the raw esbuild parse messages (re-offset to the caller's
// location so diagnostics survive a cache hit).
type parseCacheEntry struct {
	// master is the read-only parsed AST, cloned before being handed to any consumer.
	master *css_ast.AST

	// content is the source CSS this entry was parsed from, compared on every hit.
	content string

	// diags are the raw esbuild parse messages, re-offset to the caller's location.
	diags []es_logger.Msg
}

// parseCache is a process-level, content-addressed cache of parsed CSS. A shared
// design-system or partial stylesheet (imported across many pages) is parsed once and
// reused, instead of re-parsed per page, the dominant CSS allocation in a full build.
//
// It lives on the Processor and is shared by its WithResolver copies, so every entry is
// produced with one parser-options configuration; a Processor with different options owns
// a different cache, which keeps content the only thing the key needs to capture. Keying
// on content also makes the cache self-invalidating across rebuilds.
type parseCache struct {
	// m maps a content hash to its *parseCacheEntry.
	m sync.Map

	// entries counts cached entries, used to enforce parseCacheMaxEntries.
	entries atomic.Int64
}

// newParseCache returns a new, empty parse cache.
//
// Returns *parseCache which is ready for concurrent use.
func newParseCache() *parseCache { return &parseCache{} }

// hashCSSContent returns the seeded hash of the given CSS content.
//
// Takes content (string) which is the CSS source to hash.
//
// Returns uint64 which is the content hash used as the cache key.
func hashCSSContent(content string) uint64 {
	return maphash.String(cssHashSeed, content)
}

// get returns the cached entry whose content exactly matches, or nil.
//
// A hash hit whose stored content differs (a collision) is treated as a miss, so the
// master AST handed back is always the parse of this exact content, never a colliding
// sheet's.
//
// Takes content (string) which is the CSS source to look up.
//
// Returns *parseCacheEntry which is the matching entry, or nil on a miss.
func (c *parseCache) get(content string) *parseCacheEntry {
	if v, ok := c.m.Load(hashCSSContent(content)); ok {
		if entry, isEntry := v.(*parseCacheEntry); isEntry && entry != nil && entry.content == content {
			return entry
		}
	}
	return nil
}

// put records entry under content's hash.
//
// It returns the entry now authoritative for that content. If another goroutine cached
// the identical content first, that winner is returned (so all consumers clone one
// master). A hash collision with different content overwrites: collisions between
// distinct build-time sheets are vanishingly rare, so a replace is cheaper than
// per-bucket chaining and still correct because get re-verifies content.
//
// Takes content (string) which is the CSS source the entry was parsed from.
//
// Takes entry (*parseCacheEntry) which is the parse to cache for that content.
//
// Returns *parseCacheEntry which is the entry now authoritative for the content.
func (c *parseCache) put(content string, entry *parseCacheEntry) *parseCacheEntry {
	key := hashCSSContent(content)
	if existing, loaded := c.m.LoadOrStore(key, entry); loaded {
		if winner, isEntry := existing.(*parseCacheEntry); isEntry && winner != nil && winner.content == content {
			return winner
		}
		c.m.Store(key, entry)
		return entry
	}
	if c.entries.Add(1) > parseCacheMaxEntries {
		c.m.Clear()
		c.entries.Store(0)
	}
	return entry
}
