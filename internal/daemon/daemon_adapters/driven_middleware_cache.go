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

package daemon_adapters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzip"
	"github.com/cespare/xxhash/v2"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/capabilities/capabilities_dto"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/goroutine"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	// htmlBufferPoolSize is the size in bytes to pre-allocate for HTML page
	// buffers.
	htmlBufferPoolSize = 8192

	// mapCarrierPoolSize is the initial size for MapCarrier maps in the pool.
	mapCarrierPoolSize = 16

	// cacheKeyLen is the total length of a cache key: "page:" (5) plus 16 hex
	// digits equals 21.
	cacheKeyLen = 21

	// jitETagLen is the total length of a JIT ETag.
	// Format: quote + "jit-" (5) + 16 hex digits + quote = 22.
	jitETagLen = 22

	// hexDigitCount is the number of hex digits for hash representation
	// (64-bit / 4 bits per digit).
	hexDigitCount = 16

	// prefixLen is the byte count before the hex digits in key formats.
	prefixLen = 5

	// hexNibbleMask selects the lowest 4 bits to extract one hex digit.
	hexNibbleMask = 0xf

	// hexNibbleShift is the number of bits to shift to get the next hex digit.
	hexNibbleShift = 4

	// pipeResponseWriterHeaderSize is the default number of headers to set aside
	// space for in a pipe response writer.
	pipeResponseWriterHeaderSize = 8
)

var (
	// errEmptyBody is returned when a handler returns an empty body with a 200 OK
	// status.
	errEmptyBody = errors.New("handler returned empty body on 200 OK")

	// errHandlerNonSuccess is returned when the upstream handler returns a non-200
	// status code.
	errHandlerNonSuccess = errors.New("upstream handler returned non-200 status code")

	// brotliWriterPool is a pool of brotli writers for reuse across compress
	// operations. It uses quality level 4 and window size 20.
	brotliWriterPool = sync.Pool{
		New: func() any {
			opts := brotli.WriterOptions{
				Quality: 4,
				LGWin:   20,
			}
			return brotli.NewWriterOptions(nil, opts)
		},
	}

	// gzipWriterPool is a pool of gzip writers to reduce memory allocation.
	gzipWriterPool = sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
			return w
		},
	}

	// htmlBufferPool is a sync.Pool of pre-grown byte buffers for HTML page rendering.
	htmlBufferPool = sync.Pool{
		New: func() any {
			b := &bytes.Buffer{}
			b.Grow(htmlBufferPoolSize)
			return b
		},
	}

	// cacheControlHeaders provides pre-computed Cache-Control header
	// values for common max-age durations. Avoids fmt.Sprintf allocation
	// on every cache hit response.
	cacheControlHeaders = func() map[int]string {
		commonAges := []int{0, 60, 300, 600, 900, 1800, 3600, 7200, 14400, 28800, 43200, 86400, 604800}
		m := make(map[int]string, len(commonAges))
		for _, age := range commonAges {
			m[age] = fmt.Sprintf("public, max-age=%d", age)
		}
		return m
	}()

	// mapCarrierPool provides reusable MapCarrier maps for OTel context
	// propagation. Pre-sized for typical HTTP header count.
	mapCarrierPool = sync.Pool{
		New: func() any {
			return make(propagation.MapCarrier, mapCarrierPoolSize)
		},
	}

	// xxhashDigestPool provides reusable xxhash Digest instances for cache key
	// generation. It avoids intermediate string allocations from concatenation
	// in generateCacheArtefactID.
	xxhashDigestPool = sync.Pool{
		New: func() any {
			return xxhash.New()
		},
	}

	// nullSeparator is a pre-allocated byte slice for cache key component
	// separation. Avoids allocation per-separator in generateCacheArtefactID.
	nullSeparator = []byte{0}

	// hexTable holds the lowercase hexadecimal digit lookup table for cache key encoding.
	hexTable = []byte("0123456789abcdef")

	// pageCacheProfiles is pre-built at init time to avoid per-request allocations.
	// Used for background artefact persistence with "local_disk_cache" storage.
	pageCacheProfiles = func() []registry_dto.NamedProfile {
		var brotliDeps registry_dto.Dependencies
		brotliDeps.Add("minified_html")
		var brotliTags registry_dto.Tags
		brotliTags.SetByName("type", "cached-page")
		brotliTags.SetByName(logFieldContentEnc, encodingBrotli)
		brotliTags.SetByName("storageBackendId", "local_disk_cache")
		brotliTags.SetByName("mimeType", contentTypeHTML)
		var gzipDeps registry_dto.Dependencies
		gzipDeps.Add("minified_html")
		var gzipTags registry_dto.Tags
		gzipTags.SetByName("type", "cached-page")
		gzipTags.SetByName(logFieldContentEnc, encodingGzip)
		gzipTags.SetByName("storageBackendId", "local_disk_cache")
		gzipTags.SetByName("mimeType", contentTypeHTML)
		return []registry_dto.NamedProfile{
			{
				Name: "brotli_variant",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: capabilities_dto.CapabilityCompressBrotli.String(),
					Params:         registry_dto.ProfileParams{},
					ResultingTags:  brotliTags,
					DependsOn:      brotliDeps,
				},
			},
			{
				Name: "gzip_variant",
				Profile: registry_dto.DesiredProfile{
					Priority:       registry_dto.PriorityWant,
					CapabilityName: capabilities_dto.CapabilityCompressGzip.String(),
					Params:         registry_dto.ProfileParams{},
					ResultingTags:  gzipTags,
					DependsOn:      gzipDeps,
				},
			},
		}
	}()

	// compressedResponseWriterPool provides reusable compressedResponseWriter
	// structs.
	compressedResponseWriterPool = sync.Pool{
		New: func() any {
			return &compressedResponseWriter{}
		},
	}

	// pipeResponseWriterPool is a sync.Pool of pipe response writers for cache middleware.
	pipeResponseWriterPool = sync.Pool{
		New: func() any {
			return &pipeResponseWriter{
				header: make(http.Header, pipeResponseWriterHeaderSize),
			}
		},
	}
)

// CacheMiddlewareConfig configures the cache middleware behaviour.
type CacheMiddlewareConfig struct {
	// StreamCompressionLevel sets the compression level for streamed responses.
	// A value of 0 uses the default level.
	StreamCompressionLevel int

	// CacheWriteConcurrency limits the number of cache writes that can run at
	// once; defaults to defaultCacheWriteConcurrency if <= 0.
	CacheWriteConcurrency int
}

// jitResult holds the output of a just-in-time response generation.
type jitResult struct {
	// Encoding specifies the content encoding for the response, such as gzip or
	// br.
	Encoding string

	// ETag is the entity tag used for cache validation with If-None-Match headers.
	ETag string

	// CacheControl is the value for the Cache-Control HTTP header in the response.
	CacheControl string

	// Content holds the response body bytes to write to the client.
	Content []byte

	// StatusCode is the HTTP status code to send in the response.
	StatusCode int
}

// CacheMiddleware provides HTTP response caching for page and partial requests.
// It uses singleflight to coalesce concurrent requests for the same cache key.
type CacheMiddleware struct {
	// registryService stores and fetches cached artefacts and their variants.
	registryService registry_domain.RegistryService

	// capabilityService provides compression for cache entries.
	capabilityService capabilities_domain.CapabilityService

	// manifest provides read-only access to page entries for cache policy lookup.
	manifest templater_domain.ManifestStoreView

	// requestGroup prevents duplicate concurrent requests for the same artefact.
	requestGroup singleflight.Group

	// routeMap maps URL route patterns to their manifest cache keys.
	routeMap map[string]string

	// writeLimiter limits the number of concurrent disk writes.
	writeLimiter chan struct{}

	// config holds the cache middleware settings.
	config CacheMiddlewareConfig
}

// NewCacheMiddleware creates a new cache middleware instance with the given
// configuration.
//
// Takes config (CacheMiddlewareConfig) which specifies cache behaviour settings.
// Takes manifest (templater_domain.ManifestStoreView) which provides access to
// page manifest entries.
// Takes registryService (registry_domain.RegistryService) which handles registry
// operations.
// Takes capabilityService (capabilities_domain.CapabilityService) which provides
// capability checking.
// Takes partialServePath (string) which is the URL prefix for partial routes.
//
// Returns *CacheMiddleware which is ready for use as HTTP middleware.
func NewCacheMiddleware(
	config CacheMiddlewareConfig,
	manifest templater_domain.ManifestStoreView,
	registryService registry_domain.RegistryService,
	capabilityService capabilities_domain.CapabilityService,
	partialServePath string,
) *CacheMiddleware {
	if config.CacheWriteConcurrency <= 0 {
		config.CacheWriteConcurrency = defaultCacheWriteConcurrency
	}
	if config.StreamCompressionLevel == 0 {
		config.StreamCompressionLevel = defaultStreamCompressionLevel
	}

	routeMap := make(map[string]string)
	for _, key := range manifest.GetKeys() {
		entry, ok := manifest.GetPageEntry(key)
		if !ok || entry.GetRoutePattern() == "" {
			continue
		}

		sanitisedRoutePattern := strings.TrimSpace(entry.GetRoutePattern())

		if entry.GetIsPage() {
			routeMap[sanitisedRoutePattern] = key
		} else {
			prefixedPattern := fmt.Sprintf("%s%s", partialServePath, sanitisedRoutePattern)
			routeMap[prefixedPattern] = key
		}
	}

	return &CacheMiddleware{
		registryService:   registryService,
		capabilityService: capabilityService,
		manifest:          manifest,
		requestGroup:      singleflight.Group{},
		routeMap:          routeMap,
		writeLimiter:      make(chan struct{}, config.CacheWriteConcurrency),
		config:            config,
	}
}

// Handle wraps the given handler with caching logic.
//
// It serves cached responses when available and caches new responses based on
// the page's cache policy.
//
// Takes next (http.Handler) which is the handler to wrap with caching.
//
// Returns http.Handler which serves cached responses or passes requests to
// next.
func (m *CacheMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, contextChanged := extractOTelContextFromRequest(r)
		ctx, l := logger_domain.From(ctx, log)

		request := r
		if contextChanged {
			request = r.WithContext(ctx)
		}

		manifestKey, ok := m.lookupManifestKey(ctx)
		if !ok {
			next.ServeHTTP(w, request)
			return
		}

		policy, ok := m.getCachePolicy(r, manifestKey)
		if !ok {
			handleNonCacheableStream(w, request, next)
			return
		}

		if !policy.Enabled || policy.NoStore || !policy.Static {
			l.Trace("Static page cache policy disabled, NoStore set, or not a static page. Streaming response.",
				logger_domain.Bool("enabled", policy.Enabled),
				logger_domain.Bool("noStore", policy.NoStore),
				logger_domain.Bool("static", policy.Static))
			handleNonCacheableStream(w, request, next)
			return
		}

		artefactID := generateCacheArtefactID(r, policy)

		if m.tryCacheHit(ctx, w, r, artefactID, policy.MaxAgeSeconds) {
			return
		}

		m.generateAndServeResponse(ctx, w, r, next, artefactID)
	})
}

// lookupManifestKey finds the manifest key for the current route pattern.
//
// Returns string which is the manifest key if found, or empty if not.
// Returns bool which is true when a mapping exists for the route.
func (m *CacheMiddleware) lookupManifestKey(ctx context.Context) (string, bool) {
	ctx, l := logger_domain.From(ctx, log)
	routeCtx := chi.RouteContext(ctx)
	pattern := strings.TrimSpace(routeCtx.RoutePattern())
	manifestKey, ok := m.routeMap[pattern]
	if !ok {
		l.Trace("No cache mapping for route pattern, passing through",
			logger_domain.String("pattern", routeCtx.RoutePattern()))
		return "", false
	}
	return manifestKey, true
}

// getCachePolicy retrieves the cache policy for a page entry.
//
// Takes r (*http.Request) which provides the incoming request data.
// Takes manifestKey (string) which identifies the page entry in the manifest.
//
// Returns templater_dto.CachePolicy which contains the resolved caching rules.
// Returns bool which indicates whether the policy was successfully retrieved.
func (m *CacheMiddleware) getCachePolicy(
	r *http.Request,
	manifestKey string,
) (templater_dto.CachePolicy, bool) {
	ctx := r.Context()
	_, l := logger_domain.From(ctx, log)
	pageEntry, ok := m.manifest.GetPageEntry(manifestKey)
	if !ok {
		return templater_dto.CachePolicy{}, false
	}

	reqData, err := templater_domain.ParseRequestData(r, "")
	if err != nil {
		l.Error("Failed to parse request data for cache policy; serving uncached",
			logger_domain.Error(err))
		return templater_dto.CachePolicy{}, false
	}

	policy := pageEntry.GetCachePolicy(reqData)
	reqData.Release()
	return policy, true
}

// tryCacheHit checks for a valid cached artefact and serves it if found.
//
// Takes w (http.ResponseWriter) which receives the cached response if found.
// Takes r (*http.Request) which provides the request context for serving.
// Takes artefactID (string) which identifies the artefact to look up.
// Takes maxAgeSeconds (int) which specifies the maximum cache age in seconds.
//
// Returns bool which is true if a cached response was served.
func (m *CacheMiddleware) tryCacheHit(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	artefactID string,
	maxAgeSeconds int,
) bool {
	ctx, l := logger_domain.From(ctx, log)
	artefact, err := m.registryService.GetArtefact(ctx, artefactID)
	if err != nil {
		l.Trace("Cache miss, generating content")
		w.Header()[headerXCacheStatus] = headerValCacheStatusMiss
		cacheMissCount.Add(ctx, 1)
		return false
	}

	if time.Since(artefact.UpdatedAt) <= time.Duration(maxAgeSeconds)*time.Second {
		l.Trace("Cache hit, serving from cache")
		w.Header()[headerXCacheStatus] = headerValCacheStatusHit
		cacheHitCount.Add(ctx, 1)
		m.serveFromCache(w, r.WithContext(ctx), artefact, maxAgeSeconds)
		return true
	}

	l.Trace("Cache stale, regenerating",
		logger_domain.Duration("age", time.Since(artefact.UpdatedAt)),
		logger_domain.Duration("maxAge", time.Duration(maxAgeSeconds)*time.Second))
	w.Header()[headerXCacheStatus] = headerValCacheStatusStale
	cacheMissCount.Add(ctx, 1)
	return false
}

// generateAndServeResponse creates a new response using singleflight
// protection and sends it to the client.
//
// Takes w (http.ResponseWriter) which receives the generated response.
// Takes r (*http.Request) which provides the incoming request details.
// Takes next (http.Handler) which generates the original response.
// Takes artefactID (string) which identifies the cached item.
func (m *CacheMiddleware) generateAndServeResponse(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	artefactID string,
) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Generating response with singleflight protection")
	generationStartTime := time.Now()
	result, err, _ := m.requestGroup.Do(artefactID, func() (any, error) {
		return m.generateAndCacheResponse(r.WithContext(ctx), next, artefactID)
	})
	generationDuration := time.Since(generationStartTime)
	cacheGenerationDuration.Record(ctx, float64(generationDuration.Milliseconds()))

	if err != nil {
		if errors.Is(err, errEmptyBody) {
			l.Trace("Empty body returned, sending 204 No Content")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		l.Error("Handler/cache generation failed", logger_domain.Error(err))
		http.Error(w, "Could not generate page", http.StatusInternalServerError)
		return
	}

	m.serveJITResult(w, r.WithContext(ctx), result.(*jitResult))
}

// readAndCompressStream reads content from stream, optionally compressing it.
//
// Takes stream (io.Reader) which provides the content to read.
// Takes encoding (string) which specifies the compression encoding to apply.
// Takes capability (capabilities_dto.Capability) which identifies the
// compression capability to use.
//
// Returns []byte which contains the read content, compressed if encoding was
// specified.
// Returns error when reading fails or compression cannot be applied.
func (m *CacheMiddleware) readAndCompressStream(
	ctx context.Context,
	stream io.Reader, encoding string, capability capabilities_dto.Capability,
) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	if encoding == "" {
		content, err := io.ReadAll(stream)
		if err != nil {
			l.Error("Failed to read from uncompressed stream", logger_domain.Error(err))
			return nil, fmt.Errorf("failed to read uncompressed stream: %w", err)
		}
		return content, nil
	}

	l.Trace("Preparing JIT compression stream", logger_domain.String("encoding", encoding))
	compressedStream, err := m.capabilityService.Execute(ctx, capability.String(), stream, nil)
	if err != nil {
		l.Error("Failed to create compression capability stream", logger_domain.Error(err))
		return nil, fmt.Errorf("failed to start compression stream: %w", err)
	}
	content, err := io.ReadAll(compressedStream)
	if err != nil {
		l.Error("Failed to read from compressed stream", logger_domain.Error(err))
		return nil, fmt.Errorf("failed to read compressed stream: %w", err)
	}
	return content, nil
}

// generateAndCacheResponse generates a response by calling the next handler
// and caches the result for future requests.
//
// Takes r (*http.Request) which provides the incoming request context.
// Takes next (http.Handler) which is the upstream handler to generate the
// response.
// Takes artefactID (string) which identifies the cache entry for storage.
//
// Returns *jitResult which contains the compressed response ready for the
// client.
// Returns error when the upstream handler returns a non-200 status or an empty
// body.
//
// Spawns goroutines to capture the upstream response and to persist the
// artefact in the background.
func (m *CacheMiddleware) generateAndCacheResponse(
	r *http.Request, next http.Handler, artefactID string,
) (*jitResult, error) {
	ctx := r.Context()
	ctx, l := logger_domain.From(ctx, log)

	pr, pw := io.Pipe()
	statusChan := make(chan int, 1)
	go func() {
		defer func() { _ = pw.Close() }()
		defer goroutine.RecoverPanic(ctx, "daemon.cacheMiddlewareUpstream")
		recorder := getPipeResponseWriter(pw)
		next.ServeHTTP(recorder, r.WithContext(ctx))
		statusChan <- recorder.statusCode
		releasePipeResponseWriter(recorder)
	}()

	rawHTMLBuffer := getHTMLBuffer()
	bufferHandedOff := false
	defer func() {
		if !bufferHandedOff {
			releaseHTMLBuffer(rawHTMLBuffer)
		}
	}()

	streamForCompression := io.TeeReader(pr, rawHTMLBuffer)
	encodingForUser, capabilityToUse := determineCompression(r.Header.Get("Accept-Encoding"))

	contentForUser, err := m.readAndCompressStream(ctx, streamForCompression, encodingForUser, capabilityToUse)
	if err != nil {
		return nil, fmt.Errorf("reading and compressing response stream: %w", err)
	}

	if finalStatusCode := <-statusChan; finalStatusCode != http.StatusOK {
		l.Warn("Upstream handler returned non-200 status code", logger_domain.Int("status", finalStatusCode))
		return nil, errHandlerNonSuccess
	}
	if rawHTMLBuffer.Len() == 0 {
		l.Warn("Handler returned empty body")
		return nil, errEmptyBody
	}

	l.Trace("Scheduling background persistence of artefact", logger_domain.Int("size", rawHTMLBuffer.Len()))
	bufferHandedOff = true
	go m.persistArtefactInBackground(ctx, artefactID, r.URL.String(), rawHTMLBuffer)

	return &jitResult{
		Content: contentForUser, Encoding: encodingForUser,
		ETag:         formatJITETag(xxhash.Sum64(contentForUser)),
		StatusCode:   http.StatusOK,
		CacheControl: jitCacheControl,
	}, nil
}

// persistArtefactInBackground saves rendered HTML to the cache.
//
// Takes parentCtx (context.Context) which provides tracing values from the
// original request. Cancellation is detached so persistence completes
// independently.
// Takes artefactID (string) which identifies the cache entry.
// Takes sourcePath (string) which is the original request path.
// Takes rawHTML (*bytes.Buffer) which contains the rendered HTML to cache.
//
// The buffer is returned to the pool after use. Uses a write limiter to
// control disk writes.
func (m *CacheMiddleware) persistArtefactInBackground(parentCtx context.Context, artefactID, sourcePath string, rawHTML *bytes.Buffer) {
	defer func() {
		rawHTML.Reset()
		htmlBufferPool.Put(rawHTML)
	}()
	defer goroutine.RecoverPanic(context.WithoutCancel(parentCtx), "daemon.persistArtefactInBackground")

	m.writeLimiter <- struct{}{}
	defer func() { <-m.writeLimiter }()

	ctx, cancel := context.WithTimeoutCause(context.WithoutCancel(parentCtx), 30*time.Second,
		errors.New("cache middleware shutdown exceeded 30s timeout"))
	defer cancel()

	ctx, l := logger_domain.From(ctx, log)
	const blobStoreID = "local_disk_cache"

	_, err := m.registryService.UpsertArtefact(ctx, artefactID, sourcePath, rawHTML, blobStoreID, pageCacheProfiles)
	if err != nil {
		l.Error("Background cache persistence failed",
			logger_domain.String(logFieldArtefactID, artefactID),
			logger_domain.Error(err))
	} else {
		l.Trace("Successfully upserted cache artefact. Orchestrator will process variants.",
			logger_domain.String(logFieldArtefactID, artefactID))
	}
}

// variantSelectionResult holds the result of choosing the best variant for a
// request.
type variantSelectionResult struct {
	// variant is the chosen variant, or nil if no suitable variant was found.
	variant *registry_dto.Variant

	// variantName is the name of the selected variant for logging and tracing.
	variantName string
}

// serveCachedVariant writes cached data to the HTTP response.
//
// Takes w (http.ResponseWriter) which receives the cached content.
// Takes variant (*registry_dto.Variant) which holds the cached data to serve.
// Takes maxAge (int) which sets how long the cache lasts in seconds.
func (m *CacheMiddleware) serveCachedVariant(
	ctx context.Context,
	w http.ResponseWriter,
	variant *registry_dto.Variant,
	maxAge int,
) {
	ctx, l := logger_domain.From(ctx, log)
	blobStream, err := m.registryService.GetVariantData(ctx, variant)
	if err != nil {
		l.Error("Failed to read blob for cached artefact", logger_domain.Error(err))
		http.Error(w, "Failed to read from cache", http.StatusInternalServerError)
		return
	}

	etag := variant.MetadataTags.Get(registry_dto.TagEtag)

	l.Trace("Preparing response headers",
		logger_domain.Int64("contentLength", variant.SizeBytes),
		logger_domain.Int("maxAge", maxAge))

	h := w.Header()
	h[headerContentType] = headerValContentTypeHTML
	if encoding := variant.MetadataTags.Get(registry_dto.TagContentEncoding); encoding != "" {
		h[headerContentEncoding] = []string{encoding}
	}
	h[headerETag] = []string{etag}
	h[headerCacheControl] = []string{getCacheControlHeader(maxAge)}
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil && pctx.Locale != "" {
		h[headerVary] = headerValVaryCachePageI18n
	} else {
		h[headerVary] = headerValVaryCachePage
	}
	w.WriteHeader(http.StatusOK)

	bytesWritten, err := io.Copy(w, blobStream)
	if err != nil {
		l.Error("Error writing response body", logger_domain.Error(err))
	} else {
		l.Trace("Response successfully written", logger_domain.Int64(logFieldBytesWritten, bytesWritten))
	}
}

// serveFromCache writes the cached artefact response to the client.
//
// Takes w (http.ResponseWriter) which receives the cached response.
// Takes r (*http.Request) which provides headers for selecting the best
// variant.
// Takes artefact (*registry_dto.ArtefactMeta) which holds the cached artefact
// and its variants.
// Takes maxAge (int) which sets the cache duration in seconds.
func (m *CacheMiddleware) serveFromCache(w http.ResponseWriter, r *http.Request, artefact *registry_dto.ArtefactMeta, maxAge int) {
	ctx := r.Context()
	ctx, l := logger_domain.From(ctx, log)

	acceptEncoding := r.Header.Get("Accept-Encoding")
	l.Trace("Finding best variant based on Accept-Encoding",
		logger_domain.String("acceptEncoding", acceptEncoding))

	result := selectBestVariantForRequest(artefact, acceptEncoding)
	if result.variant == nil {
		l.Error("No suitable variant found",
			logger_domain.Int("variantCount", len(artefact.ActualVariants)))
		http.Error(w, "Cache consistency error: no suitable variant found", http.StatusInternalServerError)
		return
	}

	l.Trace("Selected variant", logger_domain.String("variant", result.variantName))

	etag := result.variant.MetadataTags.Get(registry_dto.TagEtag)

	if r.Header.Get(headerIfNoneMatch) == etag {
		l.Trace("ETag matched, returning 304 Not Modified")
		w.WriteHeader(http.StatusNotModified)
		return
	}

	m.serveCachedVariant(ctx, w, result.variant, maxAge)
}

// serveJITResult writes a just-in-time compiled result to the response.
// It checks if the client's ETag matches and returns 304 Not Modified when
// the content has not changed.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes r (*http.Request) which provides headers for ETag matching.
// Takes result (*jitResult) which contains the content and metadata to serve.
func (*CacheMiddleware) serveJITResult(w http.ResponseWriter, r *http.Request, result *jitResult) {
	ctx := r.Context()
	_, l := logger_domain.From(ctx, log)
	if r.Header.Get(headerIfNoneMatch) == result.ETag {
		l.Trace("ETag matched, returning 304 Not Modified")
		w.WriteHeader(http.StatusNotModified)
		return
	}

	l.Trace("Serving JIT result",
		logger_domain.Int("contentLength", len(result.Content)),
		logger_domain.Int("statusCode", result.StatusCode))

	h := w.Header()
	h[headerContentType] = headerValContentTypeHTML
	h[headerContentEncoding] = []string{result.Encoding}
	h[headerETag] = []string{result.ETag}
	h[headerCacheControl] = []string{result.CacheControl}
	if pctx := daemon_dto.PikoRequestCtxFromContext(r.Context()); pctx != nil && pctx.Locale != "" {
		h[headerVary] = headerValVaryCachePageI18n
	} else {
		h[headerVary] = headerValVaryCachePage
	}
	w.WriteHeader(result.StatusCode)

	bytesWritten, err := w.Write(result.Content)
	if err != nil {
		l.Error("Error writing JIT response body", logger_domain.Error(err))
	} else {
		l.Trace("JIT response successfully written", logger_domain.Int(logFieldBytesWritten, bytesWritten))
	}
}

// compressedResponseWriter wraps an http.ResponseWriter with compression.
// It implements io.Writer.
type compressedResponseWriter struct {
	http.ResponseWriter

	// compressor writes compressed data to the underlying response writer.
	compressor io.WriteCloser
}

// Write writes compressed data to the underlying response writer.
//
// Takes p ([]byte) which contains the data to compress and write.
//
// Returns int which is the number of bytes written.
// Returns error when the write to the underlying compressor fails.
func (w *compressedResponseWriter) Write(p []byte) (int, error) {
	return w.compressor.Write(p)
}

// Flush sends any buffered compressed data to the client. Required for
// HTTP/2 features such as 103 Early Hints that need to flush intermediate
// responses before the main body.
func (w *compressedResponseWriter) Flush() {
	if flusher, ok := w.compressor.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// pipeResponseWriter implements io.Writer and captures HTTP response data.
type pipeResponseWriter struct {
	io.Writer

	// header stores the HTTP response headers.
	header http.Header

	// statusCode holds the HTTP status code; defaults to 200 OK.
	statusCode int
}

// Header returns the header map for the response.
//
// Returns http.Header which contains the response headers.
func (w *pipeResponseWriter) Header() http.Header {
	return w.header
}

// Write writes data to the underlying pipe writer.
//
// Takes p ([]byte) which contains the data to write.
//
// Returns int which is the number of bytes written.
// Returns error when the write fails.
func (w *pipeResponseWriter) Write(p []byte) (int, error) {
	return w.Writer.Write(p)
}

// WriteHeader records the status code for later retrieval.
//
// Takes statusCode (int) which specifies the HTTP status code to record.
func (w *pipeResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
}

// Flush sends any buffered data to the client if the underlying writer
// supports flushing.
func (w *pipeResponseWriter) Flush() {
	if flusher, ok := w.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// getHTMLBuffer gets a buffer from the pool for HTML content.
//
// Returns *bytes.Buffer which is a reset buffer ready for use.
func getHTMLBuffer() *bytes.Buffer {
	buffer, ok := htmlBufferPool.Get().(*bytes.Buffer)
	if !ok {
		buffer = &bytes.Buffer{}
		buffer.Grow(htmlBufferPoolSize)
	}
	buffer.Reset()
	return buffer
}

// releaseHTMLBuffer returns a buffer to the HTML buffer pool.
//
// Takes buffer (*bytes.Buffer) which is the buffer to reset and return.
func releaseHTMLBuffer(buffer *bytes.Buffer) {
	buffer.Reset()
	htmlBufferPool.Put(buffer)
}

// getCacheControlHeader returns a Cache-Control header value for the given
// maximum age. It uses stored values for common ages and falls back to
// fmt.Sprintf for other values.
//
// Takes maxAge (int) which specifies the cache length in seconds.
//
// Returns string which is the formatted Cache-Control header value.
func getCacheControlHeader(maxAge int) string {
	if h, ok := cacheControlHeaders[maxAge]; ok {
		return h
	}
	return fmt.Sprintf("public, max-age=%d", maxAge)
}

// formatCacheKey formats a hash as "page:" followed by 16 hex digits.
// Uses direct hex encoding instead of fmt.Sprintf for zero allocations.
//
// Takes hash (uint64) which is the value to encode as hexadecimal.
//
// Returns string which is the formatted cache key.
func formatCacheKey(hash uint64) string {
	buffer := make([]byte, cacheKeyLen)
	copy(buffer, "page:")
	for i := hexDigitCount - 1; i >= 0; i-- {
		buffer[prefixLen+i] = hexTable[hash&hexNibbleMask]
		hash >>= hexNibbleShift
	}
	return string(buffer)
}

// formatJITETag formats a hash as a quoted ETag string: "jit-" followed by
// 16 hex digits. Uses direct hex encoding instead of fmt.Sprintf for zero
// memory allocations.
//
// Takes hash (uint64) which is the value to encode as hex digits.
//
// Returns string which is the formatted ETag in the form
// "jit-XXXXXXXXXXXXXXXX".
func formatJITETag(hash uint64) string {
	buffer := make([]byte, jitETagLen)
	copy(buffer, `"jit-`)
	for i := hexDigitCount - 1; i >= 0; i-- {
		buffer[prefixLen+i] = hexTable[hash&hexNibbleMask]
		hash >>= hexNibbleShift
	}
	buffer[jitETagLen-1] = '"'
	return string(buffer)
}

// extractOTelContextFromRequest extracts OpenTelemetry context from request
// headers.
//
// Uses a pooled MapCarrier to avoid per-request map allocation. If context was
// already extracted by earlier middleware (e.g., createTracingHandler), returns
// immediately without re-parsing headers.
//
// Takes r (*http.Request) which provides the incoming request with headers.
//
// Returns context.Context which contains the extracted OpenTelemetry trace
// context.
// Returns bool which is true if extraction was performed (context changed),
// false if context was already extracted and unchanged.
func extractOTelContextFromRequest(r *http.Request) (context.Context, bool) {
	ctx := r.Context()

	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil && pctx.OtelExtracted {
		return ctx, false
	}

	carrier, ok := mapCarrierPool.Get().(propagation.MapCarrier)
	if !ok {
		carrier = make(propagation.MapCarrier, mapCarrierPoolSize)
	}
	clear(carrier)

	for k, v := range r.Header {
		if len(v) > 0 {
			carrier.Set(k, v[0])
		}
	}
	result := otel.GetTextMapPropagator().Extract(ctx, carrier)
	mapCarrierPool.Put(carrier)

	if pctx := daemon_dto.PikoRequestCtxFromContext(result); pctx != nil {
		pctx.OtelExtracted = true
	}

	return result, true
}

// generateCacheArtefactID creates a cache key from the request URL, locale,
// and policy key. Uses a pooled xxhash hasher to avoid extra memory use from
// string joining.
//
// Takes r (*http.Request) which provides the URL and context for building the
// cache key.
// Takes policy (CachePolicy) which provides an optional key to include in the
// hash.
//
// Returns string which is the formatted cache key based on the hash.
func generateCacheArtefactID(r *http.Request, policy templater_dto.CachePolicy) string {
	hasher, ok := xxhashDigestPool.Get().(*xxhash.Digest)
	if !ok {
		hasher = xxhash.New()
	}

	_, _ = hasher.WriteString(r.URL.String())

	if pctx := daemon_dto.PikoRequestCtxFromContext(r.Context()); pctx != nil && pctx.Locale != "" {
		_, _ = hasher.Write(nullSeparator)
		_, _ = hasher.WriteString("locale=")
		_, _ = hasher.WriteString(pctx.Locale)
	}

	if policy.Key != "" {
		_, _ = hasher.Write(nullSeparator)
		_, _ = hasher.WriteString(policy.Key)
	}

	sum := hasher.Sum64()
	hasher.Reset()
	xxhashDigestPool.Put(hasher)

	return formatCacheKey(sum)
}

// determineCompression determines the encoding and capability based on the
// Accept-Encoding header.
//
// Takes acceptEncoding (string) which is the value of the Accept-Encoding
// header from the HTTP request.
//
// Returns string which is the encoding name (e.g. "br", "gzip") or empty if
// no supported encoding is found.
// Returns capabilities_dto.Capability which indicates the compression
// capability to use.
func determineCompression(acceptEncoding string) (string, capabilities_dto.Capability) {
	if strings.Contains(acceptEncoding, encodingBrotli) {
		return "br", capabilities_dto.CapabilityCompressBrotli
	}
	if strings.Contains(acceptEncoding, "gzip") {
		return "gzip", capabilities_dto.CapabilityCompressGzip
	}
	return "", ""
}

// selectBestVariantForRequest picks the best variant based on Accept-Encoding.
// Priority order: brotli, gzip, minified-html, source.
//
// Takes artefact (*registry_dto.ArtefactMeta) which holds the available
// variants to choose from.
// Takes acceptEncoding (string) which lists the encodings the client accepts.
//
// Returns variantSelectionResult which holds the chosen variant and its name,
// or an empty result if no suitable variant is found.
func selectBestVariantForRequest(
	artefact *registry_dto.ArtefactMeta,
	acceptEncoding string,
) variantSelectionResult {
	if strings.Contains(acceptEncoding, encodingBrotli) {
		if v := findVariantByTag(artefact, logFieldContentEnc, encodingBrotli); v != nil {
			return variantSelectionResult{variant: v, variantName: "brotli"}
		}
	}

	if strings.Contains(acceptEncoding, encodingGzip) {
		if v := findVariantByTag(artefact, logFieldContentEnc, encodingGzip); v != nil {
			return variantSelectionResult{variant: v, variantName: encodingGzip}
		}
	}

	if v := findVariantByTag(artefact, "type", "minified-html"); v != nil {
		return variantSelectionResult{variant: v, variantName: "minified-html"}
	}

	if v := findVariantByID(artefact.ActualVariants, variantSource); v != nil {
		return variantSelectionResult{variant: v, variantName: variantSource}
	}

	return variantSelectionResult{}
}

// getCompressedResponseWriter gets a compressedResponseWriter from the pool.
//
// Takes w (http.ResponseWriter) which is the underlying response writer.
// Takes compressor (io.WriteCloser) which compresses the response data.
//
// Returns *compressedResponseWriter which wraps the response writer with
// compression support.
func getCompressedResponseWriter(w http.ResponseWriter, compressor io.WriteCloser) *compressedResponseWriter {
	crw, ok := compressedResponseWriterPool.Get().(*compressedResponseWriter)
	if !ok {
		crw = &compressedResponseWriter{}
	}
	crw.ResponseWriter = w
	crw.compressor = compressor
	return crw
}

// releaseCompressedResponseWriter returns a compressedResponseWriter to the
// pool for reuse.
//
// Takes crw (*compressedResponseWriter) which is the writer to release.
func releaseCompressedResponseWriter(crw *compressedResponseWriter) {
	crw.ResponseWriter = nil
	crw.compressor = nil
	compressedResponseWriterPool.Put(crw)
}

// setupBrotliCompressor gets a brotli writer from the pool and sets the
// response headers for brotli compression.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes w (http.ResponseWriter) which receives the compression headers.
//
// Returns io.WriteCloser which wraps the response writer with brotli
// compression.
// Returns bool which shows whether the compressor was set up with success.
func setupBrotliCompressor(ctx context.Context, w http.ResponseWriter) (io.WriteCloser, bool) {
	w.Header()[headerContentEncoding] = headerValEncodingBrotli
	bw, ok := brotliWriterPool.Get().(*brotli.Writer)
	if !ok {
		_, l := logger_domain.From(ctx, log)
		l.Error("Failed to get brotli writer from pool")
		return nil, false
	}
	bw.Reset(w)
	return bw, true
}

// setupGzipCompressor gets a gzip writer from a pool and sets the response
// headers for gzip encoding.
//
// Takes ctx (context.Context) which carries logging context for trace/request
// ID propagation.
// Takes w (http.ResponseWriter) which receives the compression headers.
//
// Returns io.WriteCloser which wraps the response with gzip compression.
// Returns bool which indicates whether the writer was obtained from the pool.
func setupGzipCompressor(ctx context.Context, w http.ResponseWriter) (io.WriteCloser, bool) {
	w.Header()[headerContentEncoding] = headerValEncodingGzip
	gw, ok := gzipWriterPool.Get().(*gzip.Writer)
	if !ok {
		_, l := logger_domain.From(ctx, log)
		l.Error("Failed to get gzip writer from pool")
		return nil, false
	}
	gw.Reset(w)
	return gw, true
}

// releaseCompressor returns a compressor to its pool for reuse.
//
// Takes compressor (io.WriteCloser) which is the compressor to return.
// Takes isBrotli (bool) which indicates whether this is a Brotli or gzip
// compressor.
func releaseCompressor(compressor io.WriteCloser, isBrotli bool) {
	_ = compressor.Close()
	if isBrotli {
		(compressor.(*brotli.Writer)).Reset(nil)
		brotliWriterPool.Put(compressor)
	} else {
		(compressor.(*gzip.Writer)).Reset(nil)
		gzipWriterPool.Put(compressor)
	}
}

// handleNonCacheableStream serves a response with streaming compression.
//
// When the client accepts Brotli or Gzip encoding, the response is compressed
// on the fly. When no compression is requested, the response is served
// uncompressed.
//
// Takes w (http.ResponseWriter) which receives the compressed response.
// Takes r (*http.Request) which provides the incoming request and headers.
// Takes next (http.Handler) which is the handler to serve the actual content.
func handleNonCacheableStream(w http.ResponseWriter, r *http.Request, next http.Handler) {
	acceptEncoding := r.Header.Get("Accept-Encoding")

	var compressor io.WriteCloser
	var isBrotli, ok bool

	switch {
	case strings.Contains(acceptEncoding, encodingBrotli):
		compressor, ok = setupBrotliCompressor(r.Context(), w)
		isBrotli = true
	case strings.Contains(acceptEncoding, encodingGzip):
		compressor, ok = setupGzipCompressor(r.Context(), w)
	default:
		next.ServeHTTP(w, r)
		return
	}

	if !ok {
		next.ServeHTTP(w, r)
		return
	}
	defer releaseCompressor(compressor, isBrotli)

	wrappedWriter := getCompressedResponseWriter(w, compressor)
	defer releaseCompressedResponseWriter(wrappedWriter)

	next.ServeHTTP(wrappedWriter, r)
}

// findVariantByTag searches for a variant with a given metadata tag value.
//
// Takes artefact (*registry_dto.ArtefactMeta) which holds the variants to
// search.
// Takes key (string) which is the metadata tag name to match.
// Takes value (string) which is the tag value to find.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if no
// variant has the given tag value.
func findVariantByTag(artefact *registry_dto.ArtefactMeta, key, value string) *registry_dto.Variant {
	for i := range artefact.ActualVariants {
		v := &artefact.ActualVariants[i]
		if tagValue, ok := v.MetadataTags.GetByName(key); ok && tagValue == value {
			return v
		}
	}
	return nil
}

// getPipeResponseWriter gets a pipeResponseWriter from the pool.
//
// Takes pw (*io.PipeWriter) which is the underlying writer for the response.
//
// Returns *pipeResponseWriter which is ready to use with status set to OK.
func getPipeResponseWriter(pw *io.PipeWriter) *pipeResponseWriter {
	prw, ok := pipeResponseWriterPool.Get().(*pipeResponseWriter)
	if !ok {
		prw = &pipeResponseWriter{
			header: make(http.Header, pipeResponseWriterHeaderSize),
		}
	}
	prw.Writer = pw
	prw.statusCode = http.StatusOK
	clear(prw.header)
	return prw
}

// newPipeResponseWriter creates a new pipeResponseWriter for test
// compatibility.
//
// Takes pw (*io.PipeWriter) which receives the response body data.
//
// Returns *pipeResponseWriter which wraps the pipe writer with HTTP response
// handling.
func newPipeResponseWriter(pw *io.PipeWriter) *pipeResponseWriter {
	return &pipeResponseWriter{
		Writer:     pw,
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

// releasePipeResponseWriter returns a pipeResponseWriter to the pool.
//
// Takes prw (*pipeResponseWriter) which is the writer to release.
func releasePipeResponseWriter(prw *pipeResponseWriter) {
	prw.Writer = nil
	pipeResponseWriterPool.Put(prw)
}
