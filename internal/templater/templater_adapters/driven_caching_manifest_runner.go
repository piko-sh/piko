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
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/internal/json"
	"github.com/cespare/xxhash/v2"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// cacheKeyHexDigits is the number of hex digits for hash representation
	// (64-bit / 4 bits per digit).
	cacheKeyHexDigits = 16

	// cacheKeyNibbleMask selects the lowest 4 bits to extract one hex digit.
	cacheKeyNibbleMask = 0xf

	// cacheKeyNibbleShift is the number of bits to shift when extracting each
	// hex digit.
	cacheKeyNibbleShift = 4
)

// CachingManifestRunner is a decorator that adds multi-level caching to a
// ManifestRunnerPort. It intercepts calls to RunPage and RunPartial, checking
// the cache before delegating to the next runner on a cache miss.
type CachingManifestRunner struct {
	// next is the wrapped runner called when the cache does not contain the result.
	next templater_domain.ManifestRunnerPort

	// cache stores template AST entries with support for multi-level caching.
	cache ast_domain.ASTCacheService
}

var (
	_ templater_domain.ManifestRunnerPort = (*CachingManifestRunner)(nil)

	// cacheKeyHasherPool provides reusable xxhash Digest instances for cache key
	// generation, avoiding per-request allocations.
	cacheKeyHasherPool = sync.Pool{
		New: func() any {
			return xxhash.New()
		},
	}

	// cacheKeyNullSeparator is a pre-allocated byte for cache key component
	// separation, avoiding allocation per-separator.
	cacheKeyNullSeparator = []byte{0}

	// cacheKeyHexTable maps nibble values to hex characters.
	cacheKeyHexTable = []byte("0123456789abcdef")
)

// missFunction is a callback type used by RunPage and RunPartial methods.
// It keeps the runWithCache method simpler and provides type safety.
type missFunction func(
	context.Context,
	templater_dto.PageDefinition,
	*http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error)

// RunPage returns a page from the cache if it is available.
//
// If the page is not in the cache or cannot be cached, RunPage calls the
// wrapped runner and stores the result in the cache.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
//
// Returns *ast_domain.TemplateAST which is the parsed template structure.
// Returns templater_dto.InternalMetadata which contains internal page metadata.
// Returns string which is the cache key used for this page.
// Returns error when the page cannot be rendered or retrieved from the cache.
func (c *CachingManifestRunner) RunPage(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return c.runWithCache(ctx, pageDef, request, c.next.RunPage)
}

// RunPartial attempts to serve a partial from the cache, following the same
// logic as RunPage.
//
// Takes pageDef (templater_dto.PageDefinition) which defines the partial to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
//
// Returns *ast_domain.TemplateAST which contains the parsed template tree.
// Returns templater_dto.InternalMetadata which holds rendering metadata.
// Returns string which is the cache key used for this partial.
// Returns error when cache lookup or partial rendering fails.
func (c *CachingManifestRunner) RunPartial(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return c.runWithCache(ctx, pageDef, request, c.next.RunPartial)
}

// RunPartialWithProps bypasses caching for prop-driven renders and delegates
// directly to the next runner. Caching with arbitrary props is non-trivial and
// outside current scope; this preserves correctness.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes props (any) which contains the properties to pass to the partial.
//
// Returns *ast_domain.TemplateAST which is the parsed template structure.
// Returns templater_dto.InternalMetadata which contains rendering metadata.
// Returns string which is the rendered output.
// Returns error when the underlying runner fails.
func (c *CachingManifestRunner) RunPartialWithProps(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	props any,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	return c.next.RunPartialWithProps(ctx, pageDef, request, props)
}

// GetPageEntry delegates to the wrapped runner to retrieve a page entry.
//
// Takes manifestKey (string) which identifies the page entry to retrieve.
//
// Returns templater_domain.PageEntryView which contains the page entry data.
// Returns error when the wrapped runner fails to retrieve the entry.
func (c *CachingManifestRunner) GetPageEntry(ctx context.Context, manifestKey string) (templater_domain.PageEntryView, error) {
	return c.next.GetPageEntry(ctx, manifestKey)
}

// runWithCache contains the core, shared logic for the caching strategy.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes missFunction (missFunction) which handles cache misses by generating fresh
// content.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns templater_dto.InternalMetadata which contains internal page metadata.
// Returns string which is the styling configuration for the page.
// Returns error when page entry lookup, request parsing, or rendering fails.
func (c *CachingManifestRunner) runWithCache(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	missFunction missFunction,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	pageEntry, err := c.GetPageEntry(ctx, pageDef.OriginalPath)
	if err != nil {
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("getting page entry for %q: %w", pageDef.OriginalPath, err)
	}

	reqData, err := templater_domain.ParseRequestData(request, "")
	if err != nil {
		l.Error("Failed to parse request data for cache policy",
			logger_domain.Error(err),
			logger_domain.String("path", pageDef.OriginalPath))
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("parsing request data for cache policy on %q: %w", pageDef.OriginalPath, err)
	}
	defer reqData.Release()

	policy := pageEntry.GetCachePolicy(reqData)

	if !policy.Enabled || policy.NoStore {
		l.Trace("Caching disabled for this page, bypassing",
			logger_domain.String("path", pageDef.OriginalPath))
		return missFunction(ctx, pageDef, request)
	}

	cacheKey := generateCacheKey(request, pageEntry)

	cachedEntry, err := c.cache.Get(ctx, cacheKey)

	if err == nil {
		l.Trace("AST cache hit",
			logger_domain.String("path", pageDef.OriginalPath),
			logger_domain.String(logFieldCacheKey, cacheKey))

		var metadata templater_dto.InternalMetadata
		if err := json.UnmarshalString(cachedEntry.Metadata, &metadata); err != nil {
			l.Error("Corrupt metadata in cache; invalidating entry and regenerating",
				logger_domain.Error(err),
				logger_domain.String(logFieldCacheKey, cacheKey))
			_ = c.cache.Delete(ctx, cacheKey)
			return c.handleCacheMiss(ctx, pageDef, request, missFunction, cacheKey, pageEntry)
		}

		return cachedEntry.AST.DeepClone(), metadata, pageEntry.GetStyling(), nil
	}

	return c.processCacheMiss(ctx, err, pageDef, request, missFunction, cacheKey, pageEntry)
}

// processCacheMiss handles cache miss cases, telling apart expected misses
// from errors.
//
// Takes err (error) which is the cache lookup result to check.
// Takes pageDef (templater_dto.PageDefinition) which defines the page to
// render.
// Takes request (*http.Request) which provides the incoming HTTP request.
// Takes missFunction (missFunction) which creates fresh content on cache miss.
// Takes cacheKey (string) which identifies the cache entry.
// Takes pageEntry (templater_domain.PageEntryView) which provides page
// metadata.
//
// Returns *ast_domain.TemplateAST which is the parsed template tree.
// Returns templater_dto.InternalMetadata which contains rendering metadata.
// Returns string which is the rendered content.
// Returns error when content creation fails.
func (c *CachingManifestRunner) processCacheMiss(
	ctx context.Context,
	err error,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	missFunction missFunction,
	cacheKey string,
	pageEntry templater_domain.PageEntryView,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	if !errors.Is(err, ast_domain.ErrCacheMiss) {
		l.Error("Error retrieving from cache; falling back to generating fresh content",
			logger_domain.Error(err),
			logger_domain.String(logFieldCacheKey, cacheKey))
		return missFunction(ctx, pageDef, request)
	}

	return c.handleCacheMiss(ctx, pageDef, request, missFunction, cacheKey, pageEntry)
}

// handleCacheMiss creates, encodes, and caches a new entry when the cache
// does not contain the requested page.
//
// Takes pageDef (templater_dto.PageDefinition) which specifies the page to
// render.
// Takes request (*http.Request) which provides the HTTP request context.
// Takes missFunction (missFunction) which creates fresh content when called.
// Takes cacheKey (string) which identifies the cache entry.
// Takes pageEntry (templater_domain.PageEntryView) which provides cache policy
// settings.
//
// Returns *ast_domain.TemplateAST which contains the newly created AST.
// Returns templater_dto.InternalMetadata which contains the page metadata.
// Returns string which contains the styling data.
// Returns error when the wrapped runner fails to create content.
//
// Concurrent goroutine is started to write the entry to the cache in the
// background. The goroutine uses its own context with a 10-second timeout
// so the write can finish even if the request context is cancelled.
func (c *CachingManifestRunner) handleCacheMiss(
	ctx context.Context,
	pageDef templater_dto.PageDefinition,
	request *http.Request,
	missFunction missFunction,
	cacheKey string,
	pageEntry templater_domain.PageEntryView,
) (*ast_domain.TemplateAST, templater_dto.InternalMetadata, string, error) {
	ctx, l := logger_domain.From(ctx, log)

	l.Trace("AST cache miss",
		logger_domain.String(logFieldPath, pageDef.OriginalPath),
		logger_domain.String(logFieldCacheKey, cacheKey))

	freshAST, metadata, styling, err := missFunction(ctx, pageDef, request)
	if err != nil {
		l.Error("Failed to generate fresh AST on cache miss",
			logger_domain.Error(err),
			logger_domain.String(logFieldCacheKey, cacheKey))
		return nil, templater_dto.InternalMetadata{}, "", fmt.Errorf("generating fresh AST on cache miss for %q: %w", pageDef.OriginalPath, err)
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		l.Error("Failed to encode metadata for caching; serving uncached response",
			logger_domain.Error(err),
			logger_domain.String(logFieldCacheKey, cacheKey))
		return freshAST, metadata, styling, nil
	}

	clonedForCache := freshAST.DeepClone()
	go c.backgroundCacheWrite(ctx, request, cacheKey, pageEntry, clonedForCache, metadataBytes)

	return freshAST, metadata, styling, nil
}

// backgroundCacheWrite stores the cloned AST entry in the cache in a
// background goroutine with its own timeout context.
//
// Takes ctx (context.Context) which provides the parent context for
// cancellation-independent background work.
// Takes request (*http.Request) which is re-parsed for cache policy.
// Takes cacheKey (string) which identifies the cache entry.
// Takes pageEntry (templater_domain.PageEntryView) which provides TTL policy.
// Takes clonedAST (*ast_domain.TemplateAST) which is the AST to cache.
// Takes metadataBytes ([]byte) which is the serialised metadata.
func (c *CachingManifestRunner) backgroundCacheWrite(
	ctx context.Context,
	request *http.Request,
	cacheKey string,
	pageEntry templater_domain.PageEntryView,
	clonedAST *ast_domain.TemplateAST,
	metadataBytes []byte,
) {
	bgCtx, cancel := context.WithTimeoutCause(context.WithoutCancel(ctx), 10*time.Second,
		errors.New("background cache operation exceeded 10s timeout"))
	defer cancel()

	bgCtx, bgL := logger_domain.From(bgCtx, log)

	bgL.Trace("Starting background cache write",
		logger_domain.String(logFieldCacheKey, cacheKey))

	reqData, err := templater_domain.ParseRequestData(request, "")
	if err != nil {
		bgL.Error("Failed to parse request data for cache policy in background",
			logger_domain.Error(err),
			logger_domain.String(logFieldCacheKey, cacheKey))
		return
	}
	defer reqData.Release()

	policy := pageEntry.GetCachePolicy(reqData)
	ttl := time.Duration(policy.MaxAgeSeconds) * time.Second

	entryToCache := &ast_domain.CachedASTEntry{
		AST:      clonedAST,
		Metadata: string(metadataBytes),
	}

	if err := c.cache.SetWithTTL(bgCtx, cacheKey, entryToCache, ttl); err != nil {
		bgL.Error("Failed to write to AST cache in background",
			logger_domain.Error(err),
			logger_domain.String(logFieldCacheKey, cacheKey))
	} else {
		bgL.Trace("Background cache write successful",
			logger_domain.String(logFieldCacheKey, cacheKey))
	}
}

// NewCachingManifestRunner creates a new caching decorator.
//
// Takes next (templater_domain.ManifestRunnerPort) which is the runner to fall
// back to when the cache misses.
// Takes cache (ast_domain.ASTCacheService) which provides the configured AST
// cache service.
//
// Returns templater_domain.ManifestRunnerPort which wraps the next runner with
// caching behaviour.
func NewCachingManifestRunner(next templater_domain.ManifestRunnerPort, cache ast_domain.ASTCacheService) templater_domain.ManifestRunnerPort {
	return &CachingManifestRunner{
		next:  next,
		cache: cache,
	}
}

// generateCacheKey creates a stable, unique key for a given request and page.
//
// Takes r (*http.Request) which provides the URL path and query parameters.
// Takes entry (templater_domain.PageEntryView) which provides the page path.
//
// Returns string which is the hex-encoded hash key.
//
// Uses xxhash for speed and to make it clear this is not for cryptographic
// purposes. Uses a pooled hasher and direct hex encoding to avoid allocations.
func generateCacheKey(r *http.Request, entry templater_domain.PageEntryView) string {
	hasher, ok := cacheKeyHasherPool.Get().(*xxhash.Digest)
	if !ok {
		hasher = xxhash.New()
	}

	_, _ = hasher.WriteString(entry.GetOriginalPath())
	_, _ = hasher.Write(cacheKeyNullSeparator)
	_, _ = hasher.WriteString(r.URL.Path)
	_, _ = hasher.WriteString("?")
	_, _ = hasher.WriteString(r.URL.RawQuery)
	_, _ = hasher.Write(cacheKeyNullSeparator)

	if strings.Contains(r.URL.RawQuery, "_f=true") {
		_, _ = hasher.WriteString("fragment")
	} else {
		_, _ = hasher.WriteString("full")
	}

	sum := hasher.Sum64()
	hasher.Reset()
	cacheKeyHasherPool.Put(hasher)

	return formatCacheKeyHex(sum)
}

// formatCacheKeyHex formats a hash value as a 16-character hex string.
// Uses direct hex encoding instead of hex.EncodeToString to avoid memory
// allocations (except the final string conversion).
//
// Takes hash (uint64) which is the value to convert to hexadecimal.
//
// Returns string which is the 16-character hex-encoded hash.
func formatCacheKeyHex(hash uint64) string {
	buffer := make([]byte, cacheKeyHexDigits)
	for i := cacheKeyHexDigits - 1; i >= 0; i-- {
		buffer[i] = cacheKeyHexTable[hash&cacheKeyNibbleMask]
		hash >>= cacheKeyNibbleShift
	}
	return string(buffer)
}
