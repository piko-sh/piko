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
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-chi/chi/v5"
	"golang.org/x/sync/singleflight"
	"piko.sh/piko/internal/capabilities/capabilities_domain"
	"piko.sh/piko/internal/capabilities/capabilities_dto"

	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/daemon/daemon_frontend"
	"piko.sh/piko/internal/logger/logger_domain"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/goroutine"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const (
	// htmlBufferPoolSize is the size in bytes to pre-allocate for HTML page buffers.
	htmlBufferPoolSize = 8192

	// mapCarrierPoolSize is the initial size for MapCarrier maps in the pool.
	mapCarrierPoolSize = 16

	// cacheKeyLen is the total length of a cache key: "page:" (5) plus 16 hex digits equals
	// 21.
	cacheKeyLen = 21

	// jitETagLen is the total length of a JIT ETag. Format: quote + "jit-" (5) + 16 hex
	// digits + quote = 22.
	jitETagLen = 22

	// hexDigitCount is the number of hex digits for hash representation (64-bit / 4 bits per
	// digit).
	hexDigitCount = 16

	// prefixLen is the byte count before the hex digits in key formats.
	prefixLen = 5

	// hexNibbleMask selects the lowest 4 bits to extract one hex digit.
	hexNibbleMask = 0xf

	// hexNibbleShift is the number of bits to shift to get the next hex digit.
	hexNibbleShift = 4

	// pipeResponseWriterHeaderSize is the default number of headers to set aside space for
	// in a pipe response writer.
	pipeResponseWriterHeaderSize = 8

	// htmlBufferMaxRetainedCapacity caps the pooled HTML buffer capacity; buffers larger
	// than this are dropped on release.
	htmlBufferMaxRetainedCapacity = 1 << 20

	// pipeResponseWriterMaxRetainedHeaders caps the number of header keys a pooled
	// pipeResponseWriter may retain.
	pipeResponseWriterMaxRetainedHeaders = 64

	// maxStoredPageLinkBytes caps the size of the page's Link headers kept with a cached
	// page. Longer headers are not kept, so a cache hit sends none.
	maxStoredPageLinkBytes = 8 << 10

	// pageLinkTagName is the custom tag on a cached page's first profile that keeps the
	// page's own Link header values, one per line. A page cached before the tag existed has
	// no such tag and is sent without them.
	pageLinkTagName = "pageLinkHeader"

	// pageLinkSeparator joins the stored Link header values. A header value cannot hold a
	// line break, so it never appears inside a value.
	pageLinkSeparator = "\n"

	// backgroundPersistTimeout bounds how long one background persistence of a rendered page
	// may take once it holds a write slot.
	backgroundPersistTimeout = 30 * time.Second
)

var (
	// errEmptyBody is returned when a handler returns an empty body with a 200 OK status.
	errEmptyBody = errors.New("handler returned empty body on 200 OK")

	// errHandlerNonSuccess is returned when the upstream handler returns a non-200 status
	// code.
	errHandlerNonSuccess = errors.New("upstream handler returned non-200 status code")

	// errPageLinksTooLong is returned when the page's Link headers are too long to keep with
	// the cached page.
	errPageLinksTooLong = errors.New("page Link headers are too long to store")

	// htmlBufferPool is a sync.Pool of pre-grown byte buffers for HTML page rendering.
	htmlBufferPool = sync.Pool{
		New: func() any {
			b := &bytes.Buffer{}
			b.Grow(htmlBufferPoolSize)
			return b
		},
	}

	// staticPageCacheControlValues holds the static page Cache-Control value for common
	// MaxAgeSeconds settings, so most responses do not need fmt.Sprintf.
	staticPageCacheControlValues = func() map[int]string {
		commonAges := []int{0, 60, 300, 600, 900, 1800, 3600, 7200, 14400, 28800, 43200, 86400, 604800}
		m := make(map[int]string, len(commonAges))
		for _, age := range commonAges {
			m[age] = fmt.Sprintf(staticPageCacheControlFormat, age)
		}
		return m
	}()

	// mapCarrierPool provides reusable MapCarrier maps for OTel context propagation.
	// Pre-sized for typical HTTP header count.
	mapCarrierPool = sync.Pool{
		New: func() any {
			return make(propagation.MapCarrier, mapCarrierPoolSize)
		},
	}

	// xxhashDigestPool provides reusable xxhash Digest instances for cache key generation.
	// It avoids intermediate string allocations from concatenation in
	// generateCacheArtefactID.
	xxhashDigestPool = sync.Pool{
		New: func() any {
			return xxhash.New()
		},
	}

	// nullSeparator is a pre-allocated byte slice for cache key component separation. Avoids
	// allocation per-separator in generateCacheArtefactID.
	nullSeparator = []byte{0}

	// hexTable holds the lowercase hexadecimal digit lookup table for cache key encoding.
	hexTable = []byte("0123456789abcdef")

	// pageCacheProfiles is pre-built at init time to avoid per-request allocations. Used for
	// background artefact persistence with "local_disk_cache" storage.
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

	// pipeResponseWriterPool is a sync.Pool of pipe response writers for cache middleware.
	pipeResponseWriterPool = sync.Pool{
		New: func() any {
			return &pipeResponseWriter{
				header:     make(http.Header, pipeResponseWriterHeaderSize),
				Writer:     nil,
				statusCode: 0,
			}
		},
	}
)

// CacheMiddlewareConfig configures the cache middleware behaviour.
type CacheMiddlewareConfig struct {
	// StreamCompressionLevel sets the compression level for streamed responses. A value of 0
	// uses the default level.
	StreamCompressionLevel int

	// CacheWriteConcurrency limits the number of cache writes that can run at once; defaults
	// to defaultCacheWriteConcurrency if <= 0.
	CacheWriteConcurrency int

	// PersistenceDisabled turns off background persistence of rendered pages.
	PersistenceDisabled bool

	// CSPUsesRequestTokens turns off the static page cache because the Content Security
	// Policy gives each response its own token. Pages carry that token on their script
	// elements, so a stored page would only match the response it was rendered for.
	CSPUsesRequestTokens bool

	// CoreRuntimeBuild is the build of the core runtime the server sends. A cached page
	// carries the SRI hash of that build, so the build is part of every cache key.
	CoreRuntimeBuild daemon_frontend.CoreRuntimeBuild
}

// jitResult holds the output of a just-in-time response generation.
type jitResult struct {
	// ForwardedHeader holds a copy of the page's response headers when the page answered
	// with a status other than 200 OK, such as a redirect or an error page, and is nil
	// otherwise. Such a response is sent on as the page wrote it and is never cached.
	ForwardedHeader http.Header

	// Encoding specifies the content encoding for the response, such as gzip or br.
	Encoding string

	// ETag is the entity tag used for cache validation with If-None-Match headers.
	ETag string

	// Content holds the response body bytes to write to the client.
	Content []byte

	// Links holds the page's own Link header values for a 200 OK page.
	Links []string

	// StatusCode is the HTTP status code to send in the response.
	StatusCode int
}

// sharedWithFollower returns the result as a request may see it when it joined another
// request's render instead of rendering the page itself.
//
// A forwarded response, such as a redirect or an error page, may set cookies that belong
// to the request that rendered it, such as a session, a flash message or a CSRF token.
// Those are left out, so they never reach another visitor. A 200 OK page keeps no
// cookies, so it is returned as it is.
//
// Returns *jitResult which is the same result when it sets no cookies, or a copy without
// them. The receiver is never changed, because other requests share it.
func (result *jitResult) sharedWithFollower() *jitResult {
	if _, setsCookies := result.ForwardedHeader[headerSetCookie]; !setsCookies {
		return result
	}
	follower := *result
	follower.ForwardedHeader = result.ForwardedHeader.Clone()
	delete(follower.ForwardedHeader, headerSetCookie)
	return &follower
}

// upstreamResponse describes what the upstream handler sent once it has finished.
type upstreamResponse struct {
	// forwardedHeader holds a copy of the handler's headers when it answered with a status
	// other than 200 OK, and is nil otherwise.
	forwardedHeader http.Header

	// links holds the handler's own Link header values when it answered 200 OK.
	links []string

	// statusCode is the final status code the handler wrote.
	statusCode int
}

// isForwarded reports whether the handler's response is sent on as it is rather than
// cached.
//
// Returns bool which is true when the handler answered with a status other than 200 OK.
func (u upstreamResponse) isForwarded() bool {
	return u.forwardedHeader != nil
}

// staticPageBody describes a full response with a page from the static page cache.
type staticPageBody struct {
	// encoding is the body's content encoding, or empty when it is not compressed.
	encoding string

	// etag is the page's entity tag, or empty when it has none.
	etag string

	// links holds the page's own Link header values.
	links []string

	// maxAgeSeconds is how long a shared cache may keep the page.
	maxAgeSeconds int
}

// CacheMiddleware provides HTTP response caching for page and partial requests. It uses
// singleflight to coalesce concurrent requests for the same cache key.
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

	// runtimeKey names the framework bundles this server sends by their runtime version and
	// the core runtime build. It is part of every cache key, because a cached page names the
	// bundle URLs and the core SRI hash of the server that rendered it, and the cache may
	// outlive that server.
	runtimeKey string

	// config holds the cache middleware settings.
	config CacheMiddlewareConfig
}

// NewCacheMiddleware creates a new cache middleware instance with the given
// configuration.
//
// Takes config (CacheMiddlewareConfig) which specifies cache behaviour settings.
// Takes manifest (templater_domain.ManifestStoreView) which provides access to page
// manifest entries.
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
		runtimeKey:        cacheRuntimeKey(daemon_frontend.RuntimeVersion(), config.CoreRuntimeBuild),
		config:            config,
	}
}

// Handle wraps the given handler with caching logic.
//
// It serves cached responses when available and caches new responses based on the page's
// cache policy.
//
// Takes next (http.Handler) which is the handler to wrap with caching.
//
// Returns http.Handler which serves cached responses or passes requests to next.
func (m *CacheMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, contextChanged := extractOTelContextFromRequest(r)

		request := r
		if contextChanged {
			request = r.WithContext(ctx)
		}

		manifestKey, ok := m.lookupManifestKey(ctx)
		if !ok {
			next.ServeHTTP(w, request)
			return
		}

		policy, ok := m.getCachePolicy(request, manifestKey)
		if !ok || !m.servesFromCache(ctx, w.Header(), request.Method, policy) {
			handleNonCacheableStream(w, request, next)
			return
		}

		artefactID := generateCacheArtefactID(request, policy, m.runtimeKey)

		if m.tryCacheHit(ctx, w, request, artefactID, policy.MaxAgeSeconds) {
			return
		}

		m.generateAndServeResponse(ctx, w, request, next, artefactID, policy.MaxAgeSeconds)
	})
}

// servesFromCache decides whether the static page cache serves a request, and marks the
// response as one no cache may keep when the policy sets NoStore.
//
// Only GET and HEAD are served from the cache. Any other method, such as a form POST to a
// static page, must reach the page itself and must never be stored under the URL that GET
// visitors share.
//
// Takes h (http.Header) which is the response header map.
// Takes method (string) which is the request method.
// Takes policy (templater_dto.CachePolicy) which is the page's policy for this request.
//
// Returns bool which is true when the request may be served from the cache.
func (m *CacheMiddleware) servesFromCache(
	ctx context.Context,
	h http.Header,
	method string,
	policy templater_dto.CachePolicy,
) bool {
	_, l := logger_domain.From(ctx, log)
	if policy.NoStore {
		l.Trace("Cache policy sets NoStore. Streaming response that no cache may keep.")
		setNoStoreCacheControl(h)
		return false
	}
	if !policy.Enabled || !policy.Static {
		l.Trace("Static page cache policy disabled or not a static page. Streaming response.",
			logger_domain.Bool("enabled", policy.Enabled),
			logger_domain.Bool("static", policy.Static))
		return false
	}
	if m.config.CSPUsesRequestTokens {
		l.Trace("Static page cache is off because the CSP uses per-request tokens. Streaming response.")
		return false
	}
	if !isCacheableMethod(method) {
		l.Trace("Static page cache serves only GET and HEAD. Streaming response.",
			logger_domain.String(logger_domain.KeyMethod, method))
		return false
	}
	return true
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
// Returns bool which is false when the manifest has no entry for the key.
func (m *CacheMiddleware) getCachePolicy(
	r *http.Request,
	manifestKey string,
) (templater_dto.CachePolicy, bool) {
	pageEntry, ok := m.manifest.GetPageEntry(manifestKey)
	if !ok {
		_, l := logger_domain.From(r.Context(), log)
		l.Debug("Route maps to a page the manifest does not hold; serving uncached",
			logger_domain.String("manifest_key", manifestKey))
		return templater_dto.CachePolicy{}, false
	}
	return resolveCachePolicy(r, pageEntry), true
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

// generateAndServeResponse creates a new response using singleflight protection and sends
// it to the client.
//
// Takes w (http.ResponseWriter) which receives the generated response.
// Takes r (*http.Request) which provides the incoming request details.
// Takes next (http.Handler) which generates the original response.
// Takes artefactID (string) which identifies the cached item.
// Takes maxAgeSeconds (int) which is how long a shared cache may keep the page.
func (m *CacheMiddleware) generateAndServeResponse(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	artefactID string,
	maxAgeSeconds int,
) {
	ctx, l := logger_domain.From(ctx, log)
	l.Trace("Generating response with singleflight protection")
	generationStartTime := time.Now()
	rendered := false
	result, err, _ := m.requestGroup.Do(artefactID, func() (any, error) {
		rendered = true
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

	jit, ok := result.(*jitResult)
	if !ok {
		l.Error("Shared page generation returned an unexpected result",
			logger_domain.String("result_type", fmt.Sprintf("%T", result)))
		http.Error(w, "Could not generate page", http.StatusInternalServerError)
		return
	}
	if !rendered {
		jit = jit.sharedWithFollower()
	}
	m.serveJITResult(w, r.WithContext(ctx), jit, maxAgeSeconds)
}

// readAndCompressStream reads content from stream, optionally compressing it.
//
// Takes stream (io.Reader) which provides the content to read.
// Takes encoding (string) which specifies the compression encoding to apply.
// Takes capability (capabilities_dto.Capability) which identifies the compression
// capability to use.
//
// Returns []byte which contains the read content, compressed if encoding was specified.
// Returns error when reading fails or compression cannot be applied.
func (m *CacheMiddleware) readAndCompressStream(
	ctx context.Context,
	stream io.Reader, encoding string, capability capabilities_dto.Capability,
) ([]byte, error) {
	ctx, l := logger_domain.From(ctx, log)
	if encoding == "" {
		content, err := io.ReadAll(stream)
		if err != nil {
			return nil, fmt.Errorf("reading uncompressed stream: %w", err)
		}
		return content, nil
	}

	l.Trace("Preparing JIT compression stream", logger_domain.String("encoding", encoding))
	compressedStream, err := m.capabilityService.Execute(ctx, capability.String(), stream, nil)
	if err != nil {
		return nil, fmt.Errorf("starting compression stream: %w", err)
	}
	content, err := io.ReadAll(compressedStream)
	if err != nil {
		return nil, fmt.Errorf("reading compressed stream: %w", err)
	}
	return content, nil
}

// generateAndCacheResponse generates a response by calling the next handler and caches
// the result for future requests.
//
// The upstream response is captured by a helper, which every return path waits for, and
// the artefact may be stored in the background.
//
// Takes r (*http.Request) which provides the incoming request context.
// Takes next (http.Handler) which is the upstream handler to generate the response.
// Takes artefactID (string) which identifies the cache entry for storage.
//
// Returns *jitResult which contains the compressed response ready for the client.
// Returns error when the upstream handler fails before it answers, or returns an empty
// body with 200 OK.
func (m *CacheMiddleware) generateAndCacheResponse(
	r *http.Request, next http.Handler, artefactID string,
) (*jitResult, error) {
	ctx := r.Context()
	ctx, l := logger_domain.From(ctx, log)

	pr, pw := io.Pipe()
	upstreamStatus := serveUpstreamIntoPipe(ctx, r, next, pw)

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
		_ = pr.CloseWithError(err)
		<-upstreamStatus
		return nil, fmt.Errorf("reading and compressing response stream: %w", err)
	}

	upstream := <-upstreamStatus
	if upstream.isForwarded() {
		l.Trace("Upstream handler answered without 200 OK; forwarding the response without caching it",
			logger_domain.Int("status", upstream.statusCode))
		return newForwardedResult(upstream, encodingForUser, contentForUser, rawHTMLBuffer.Len()), nil
	}
	if upstream.statusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", errHandlerNonSuccess, upstream.statusCode)
	}
	if rawHTMLBuffer.Len() == 0 {
		l.Warn("Handler returned empty body")
		return nil, errEmptyBody
	}

	if m.config.PersistenceDisabled {
		l.Trace("Registry is read-only; serving just-in-time without persisting the rendered page")
	} else {
		bufferHandedOff = m.startBackgroundPersistence(ctx, artefactID, r.URL.String(), rawHTMLBuffer, upstream.links)
	}

	return &jitResult{
		ForwardedHeader: nil,
		Content:         contentForUser,
		Encoding:        encodingForUser,
		ETag:            formatJITETag(xxhash.Sum64(contentForUser)),
		Links:           upstream.links,
		StatusCode:      http.StatusOK,
	}, nil
}

// startBackgroundPersistence stores a rendered page in the background when a cache write
// slot is free.
//
// The slot is taken before the goroutine starts, so the number of goroutines and of
// buffers they hold never grows past the write limit, however many distinct URLs clients
// ask for. When every slot is busy the page is served without being stored, and a later
// request renders and stores it.
//
// Takes artefactID (string) which identifies the cache entry.
// Takes sourcePath (string) which is the original request path.
// Takes rawHTML (*bytes.Buffer) which contains the rendered HTML to cache.
// Takes links ([]string) which holds the page's own Link header values to keep with it.
//
// Returns bool which is true when the goroutine started and now owns rawHTML.
//
// Spawns a goroutine under a context detached through daemon_dto.DetachRequestContext, so
// the write keeps its trace values while outliving the request. It ends when the write
// finishes or its timeout passes.
func (m *CacheMiddleware) startBackgroundPersistence(
	ctx context.Context, artefactID, sourcePath string, rawHTML *bytes.Buffer, links []string,
) bool {
	ctx, l := logger_domain.From(ctx, log)
	select {
	case m.writeLimiter <- struct{}{}:
	default:
		cachePersistSkippedCount.Add(ctx, 1)
		l.Debug("Serving the page without storing it because every cache write slot is busy",
			logger_domain.String(logFieldArtefactID, artefactID))
		return false
	}

	l.Trace("Scheduling background persistence of artefact", logger_domain.Int("size", rawHTML.Len()))
	go m.persistArtefactInBackground(daemon_dto.DetachRequestContext(ctx), artefactID, sourcePath, rawHTML, links)
	return true
}

// persistArtefactInBackground saves rendered HTML to the cache.
//
// The caller has already taken a cache write slot, which is given back here.
//
// Takes artefactID (string) which identifies the cache entry.
// Takes sourcePath (string) which is the original request path.
// Takes rawHTML (*bytes.Buffer) which contains the rendered HTML to cache.
// Takes links ([]string) which holds the page's own Link header values to keep with it.
//
// The buffer is returned to the pool after use.
func (m *CacheMiddleware) persistArtefactInBackground(
	ctx context.Context, artefactID, sourcePath string, rawHTML *bytes.Buffer, links []string,
) {
	ctx = goroutine.Label(ctx, "daemon.persistArtefactInBackground",
		"artefact_id", artefactID, "path", sourcePath)
	defer func() { <-m.writeLimiter }()
	defer releaseHTMLBuffer(rawHTML)
	defer goroutine.RecoverPanic(ctx, "daemon.persistArtefactInBackground")

	ctx, cancel := context.WithTimeoutCause(ctx, backgroundPersistTimeout,
		errors.New("background artefact persistence exceeded its timeout"))
	defer cancel()

	ctx, l := logger_domain.From(ctx, log)
	const blobStoreID = "local_disk_cache"

	profiles, err := pageCacheProfilesWithLinks(links)
	if err != nil {
		l.Warn("Caching the page without its Link headers",
			logger_domain.String(logFieldArtefactID, artefactID),
			logger_domain.Error(err))
	}

	_, err = m.registryService.UpsertArtefact(ctx, artefactID, sourcePath, rawHTML, blobStoreID, profiles)
	if err != nil {
		l.Error("Background cache persistence failed",
			logger_domain.String(logFieldArtefactID, artefactID),
			logger_domain.Error(err))
	} else {
		l.Trace("Successfully upserted cache artefact. Orchestrator will process variants.",
			logger_domain.String(logFieldArtefactID, artefactID))
	}
}

// variantSelectionResult holds the result of choosing the best variant for a request.
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
// Takes maxAge (int) which is how long a shared cache may keep the page, in seconds.
// Takes links ([]string) which holds the page's own Link header values kept with it.
func (m *CacheMiddleware) serveCachedVariant(
	ctx context.Context,
	w http.ResponseWriter,
	variant *registry_dto.Variant,
	maxAge int,
	links []string,
) {
	ctx, l := logger_domain.From(ctx, log)
	blobStream, err := m.registryService.GetVariantData(ctx, variant)
	if err != nil {
		l.Error("Failed to read blob for cached artefact", logger_domain.Error(err))
		http.Error(w, "Failed to read from cache", http.StatusInternalServerError)
		return
	}
	defer func() {
		if closeErr := blobStream.Close(); closeErr != nil {
			l.Warn("Failed to close the cached page blob", logger_domain.Error(closeErr))
		}
	}()

	writeCachedVariantHeaders(ctx, w, variant, maxAge, links)

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
// Takes r (*http.Request) which provides headers for selecting the best variant.
// Takes artefact (*registry_dto.ArtefactMeta) which holds the cached artefact and its
// variants.
// Takes maxAge (int) which is how long a shared cache may keep the page, in seconds.
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

	etag := variantETag(result.variant)

	if ifNoneMatchSatisfied(r.Header.Get(headerIfNoneMatch), etag) {
		l.Trace("ETag matched, returning 304 Not Modified")
		setStaticPageHeaders(ctx, w.Header(), etag, maxAge)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if r.Method == http.MethodHead {
		l.Trace("Answering HEAD from the cache without reading the page body")
		writeCachedVariantHeaders(ctx, w, result.variant, maxAge, storedPageLinks(artefact))
		return
	}
	m.serveCachedVariant(ctx, w, result.variant, maxAge, storedPageLinks(artefact))
}

// serveJITResult writes a just-in-time compiled result to the response. It checks if the
// client's ETag matches and returns 304 Not Modified when the content has not changed.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes r (*http.Request) which provides headers for ETag matching.
// Takes result (*jitResult) which contains the content and metadata to serve, or the page
// response to forward. It may be shared with other requests, so it is never changed here.
// Takes maxAgeSeconds (int) which is how long a shared cache may keep the page.
func (*CacheMiddleware) serveJITResult(w http.ResponseWriter, r *http.Request, result *jitResult, maxAgeSeconds int) {
	ctx := r.Context()
	_, l := logger_domain.From(ctx, log)
	if result.ForwardedHeader != nil {
		l.Trace("Forwarding the page response", logger_domain.Int("statusCode", result.StatusCode))
		serveForwardedResult(w, r, result)
		return
	}

	if ifNoneMatchSatisfied(r.Header.Get(headerIfNoneMatch), result.ETag) {
		l.Trace("ETag matched, returning 304 Not Modified")
		setStaticPageHeaders(ctx, w.Header(), result.ETag, maxAgeSeconds)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	l.Trace("Serving JIT result",
		logger_domain.Int("contentLength", len(result.Content)),
		logger_domain.Int("statusCode", result.StatusCode))

	setStaticPageBodyHeaders(ctx, w.Header(), staticPageBody{
		links:         result.Links,
		encoding:      result.Encoding,
		etag:          result.ETag,
		maxAgeSeconds: maxAgeSeconds,
	})
	w.WriteHeader(result.StatusCode)
	if r.Method == http.MethodHead {
		return
	}

	bytesWritten, err := w.Write(result.Content)
	if err != nil {
		l.Error("Error writing JIT response body", logger_domain.Error(err))
	} else {
		l.Trace("JIT response successfully written", logger_domain.Int(logFieldBytesWritten, bytesWritten))
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

// WriteHeader records the status code for later retrieval. An informational status such
// as 103 Early Hints is not the final status, so it is not recorded, and a body written
// after it is sent as 200 OK, as net/http does.
//
// Takes statusCode (int) which specifies the HTTP status code to record.
func (w *pipeResponseWriter) WriteHeader(statusCode int) {
	if isInformationalStatus(statusCode) {
		return
	}
	w.statusCode = statusCode
}

// Flush sends any buffered data to the client if the underlying writer supports flushing.
func (w *pipeResponseWriter) Flush() {
	if flusher, ok := w.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// response describes what the handler sent. It copies the headers it needs, all of them
// for a response that is forwarded and the Link values for a 200 OK page, because the
// writer goes back to its pool.
//
// Returns upstreamResponse which holds the final status and the copied headers.
func (w *pipeResponseWriter) response() upstreamResponse {
	if w.statusCode != http.StatusOK {
		return upstreamResponse{forwardedHeader: w.header.Clone(), links: nil, statusCode: w.statusCode}
	}
	return upstreamResponse{forwardedHeader: nil, links: slices.Clone(w.header[headerLink]), statusCode: w.statusCode}
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

// releaseHTMLBuffer returns a buffer to the HTML buffer pool, dropping buffers whose
// capacity exceeds htmlBufferMaxRetainedCapacity.
//
// Takes buffer (*bytes.Buffer) which is the buffer to reset and return.
func releaseHTMLBuffer(buffer *bytes.Buffer) {
	if buffer.Cap() > htmlBufferMaxRetainedCapacity {
		return
	}
	buffer.Reset()
	htmlBufferPool.Put(buffer)
}

// staticPageCacheControl builds the Cache-Control value for a page from the static page
// cache.
//
// A shared cache such as a CDN may keep the page for maxAgeSeconds. A browser must
// revalidate it before every reuse, because the page names the versioned framework bundle
// URLs of the build that rendered it, and a server from a later build answers those URLs
// with 404, so a browser copy that outlived a deploy would load no runtime. A shared
// cache keeps the same risk until its copy expires, so a deploy that changes the runtime
// should purge it.
//
// Takes maxAgeSeconds (int) which is the page's MaxAgeSeconds. A negative value counts as
// zero.
//
// Returns string which is the Cache-Control header value.
func staticPageCacheControl(maxAgeSeconds int) string {
	maxAgeSeconds = max(maxAgeSeconds, 0)
	if value, ok := staticPageCacheControlValues[maxAgeSeconds]; ok {
		return value
	}
	return fmt.Sprintf(staticPageCacheControlFormat, maxAgeSeconds)
}

// setStaticPageBodyHeaders sets the headers for a full response with a page from the
// static page cache. A cache hit and a cache miss both use it, so they send the same
// headers.
//
// Takes h (http.Header) which is the response header map.
// Takes body (staticPageBody) which describes the page being sent.
func setStaticPageBodyHeaders(ctx context.Context, h http.Header, body staticPageBody) {
	h[headerContentType] = headerValContentTypeHTML
	h[headerXPPResponseSupport] = headerValFragmentPatch
	if body.encoding != "" {
		h[headerContentEncoding] = []string{body.encoding}
	}
	if len(body.links) > 0 {
		h[headerLink] = slices.Concat(h[headerLink], body.links)
	}
	setStaticPageHeaders(ctx, h, body.etag, body.maxAgeSeconds)
}

// setStaticPageHeaders sets the validator and cache headers for a page from the static
// page cache.
//
// A full response and a 304 Not Modified both carry them, so a cache that revalidates its
// copy keeps the same policy. A Cache-Control that the app or the page set earlier is
// kept, and a Vary that they set is merged with the page's own.
//
// Takes h (http.Header) which is the response header map.
// Takes etag (string) which is the page's entity tag, or empty when it has none.
// Takes maxAgeSeconds (int) which is how long a shared cache may keep the page.
func setStaticPageHeaders(ctx context.Context, h http.Header, etag string, maxAgeSeconds int) {
	if etag != "" {
		h[headerETag] = []string{etag}
	}
	setDefaultCacheControl(h, []string{staticPageCacheControl(maxAgeSeconds)})
	mergeVary(h, staticPageVary(ctx))
}

// copyPageHeaders copies the headers of a forwarded page response into the client's
// response, as if the page had written them there itself.
//
// A Cache-Control that the app set earlier wins over the page's, a Vary from both is
// merged, and Link values from both are kept. Content-Length is left out, because the
// body may be compressed here.
//
// Takes destination (http.Header) which is the client's response header map.
// Takes source (http.Header) which holds the page's headers. It may be shared with other
// requests, so it is never changed here.
func copyPageHeaders(destination, source http.Header) {
	for key, values := range source {
		switch key {
		case headerCacheControl:
			setDefaultCacheControl(destination, slices.Clone(values))
		case headerVary:
			mergeVary(destination, slices.Clone(values))
		case headerLink:
			destination[key] = slices.Concat(destination[key], values)
		case headerContentLength:
		default:
			destination[key] = slices.Clone(values)
		}
	}
}

// serveForwardedResult sends a page response that is not cached, such as a redirect or an
// error page, with the page's status, headers and body. A HEAD request gets the status
// and headers only.
//
// Takes w (http.ResponseWriter) which receives the response.
// Takes r (*http.Request) which is the request being answered.
// Takes result (*jitResult) which holds the forwarded response. It may be shared with
// other requests, so it is never changed here.
func serveForwardedResult(w http.ResponseWriter, r *http.Request, result *jitResult) {
	ctx := r.Context()
	h := w.Header()
	copyPageHeaders(h, result.ForwardedHeader)
	if result.Encoding != "" {
		h[headerContentEncoding] = []string{result.Encoding}
		mergeVary(h, staticPageVary(ctx))
	}
	w.WriteHeader(result.StatusCode)
	if len(result.Content) == 0 || r.Method == http.MethodHead {
		return
	}
	if _, err := w.Write(result.Content); err != nil {
		_, l := logger_domain.From(ctx, log)
		l.Error("Error writing forwarded response body", logger_domain.Error(err))
	}
}

// newForwardedResult builds the result for a page response that is sent on as it is.
//
// Takes upstream (upstreamResponse) which holds the page's status and headers.
// Takes encoding (string) which is the encoding content was compressed with.
// Takes content ([]byte) which is the body as sent to the client.
// Takes rawLength (int) which is the length of the body the page wrote. When it is zero
// the response has no body, so the encoding is dropped too.
//
// Returns *jitResult which forwards the response.
func newForwardedResult(upstream upstreamResponse, encoding string, content []byte, rawLength int) *jitResult {
	if rawLength == 0 {
		encoding, content = "", nil
	}
	return &jitResult{
		ForwardedHeader: upstream.forwardedHeader,
		Encoding:        encoding,
		ETag:            "",
		Content:         content,
		Links:           nil,
		StatusCode:      upstream.statusCode,
	}
}

// pageCacheProfilesWithLinks returns the profiles to cache a page with, keeping the
// page's own Link header values on the first one so a cache hit can send them.
//
// Takes links ([]string) which holds the page's Link header values.
//
// Returns []registry_dto.NamedProfile which is the shared default when there are no
// links, or when they are too long to keep.
// Returns error when the links are too long to keep.
func pageCacheProfilesWithLinks(links []string) ([]registry_dto.NamedProfile, error) {
	if len(links) == 0 {
		return pageCacheProfiles, nil
	}
	joined := strings.Join(links, pageLinkSeparator)
	if len(joined) > maxStoredPageLinkBytes {
		return pageCacheProfiles, fmt.Errorf("%w: %d bytes, limit %d", errPageLinksTooLong, len(joined), maxStoredPageLinkBytes)
	}

	profiles := slices.Clone(pageCacheProfiles)
	tags := profiles[0].Profile.ResultingTags
	tags.Custom = maps.Clone(tags.Custom)
	tags.SetByName(pageLinkTagName, joined)
	profiles[0].Profile.ResultingTags = tags
	return profiles, nil
}

// storedPageLinks reads the page's own Link header values kept with a cached page.
//
// Takes artefact (*registry_dto.ArtefactMeta) which is the cached page.
//
// Returns []string which holds the Link values, or nil for a page cached without them,
// including one cached before they were kept.
func storedPageLinks(artefact *registry_dto.ArtefactMeta) []string {
	for i := range artefact.DesiredProfiles {
		if joined, ok := artefact.DesiredProfiles[i].Profile.ResultingTags.GetByName(pageLinkTagName); ok && joined != "" {
			return strings.Split(joined, pageLinkSeparator)
		}
	}
	return nil
}

// writeCachedVariantHeaders sends the status and headers of a page from the static page
// cache, without its body.
//
// Takes w (http.ResponseWriter) which receives the headers.
// Takes variant (*registry_dto.Variant) which is the cached page being sent.
// Takes maxAge (int) which is how long a shared cache may keep the page, in seconds.
// Takes links ([]string) which holds the page's own Link header values kept with it.
func writeCachedVariantHeaders(
	ctx context.Context,
	w http.ResponseWriter,
	variant *registry_dto.Variant,
	maxAge int,
	links []string,
) {
	_, l := logger_domain.From(ctx, log)
	l.Trace("Preparing response headers",
		logger_domain.Int64("contentLength", variant.SizeBytes),
		logger_domain.Int("maxAge", maxAge))

	setStaticPageBodyHeaders(ctx, w.Header(), staticPageBody{
		links:         links,
		encoding:      variant.MetadataTags.Get(registry_dto.TagContentEncoding),
		etag:          variantETag(variant),
		maxAgeSeconds: maxAge,
	})
	w.WriteHeader(http.StatusOK)
}

// isCacheableMethod reports whether the static page cache may answer a request method.
//
// Takes method (string) which is the request method.
//
// Returns bool which is true for GET and HEAD.
func isCacheableMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

// isInformationalStatus reports whether a status is an interim 1xx response, which comes
// before the final status. 101 Switching Protocols ends the response, so it is final.
//
// Takes statusCode (int) which is the status to check.
//
// Returns bool which is true for an interim status.
func isInformationalStatus(statusCode int) bool {
	return statusCode >= http.StatusContinue &&
		statusCode < http.StatusOK &&
		statusCode != http.StatusSwitchingProtocols
}

// staticPageVary picks the Vary value for a page from the static page cache.
//
// Returns []string which also names Accept-Language when the route is localised.
func staticPageVary(ctx context.Context) []string {
	if pctx := daemon_dto.PikoRequestCtxFromContext(ctx); pctx != nil && pctx.Locale != "" {
		return headerValVaryCachePageI18n
	}
	return headerValVaryCachePage
}

// formatCacheKey formats a hash as "page:" followed by 16 hex digits. Uses direct hex
// encoding instead of fmt.Sprintf for zero allocations.
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

// variantETag returns a usable ETag for a cached variant. It prefers the stored ETag tag
// and falls back to a strong validator derived from the content hash, so a variant
// persisted without an explicit ETag tag still has a non-empty validator.
//
// Takes variant (*registry_dto.Variant) which provides the cached metadata.
//
// Returns string which is the ETag, or empty when the variant has no usable validator.
func variantETag(variant *registry_dto.Variant) string {
	if etag := variant.MetadataTags.Get(registry_dto.TagEtag); etag != "" {
		return etag
	}
	if variant.ContentHash != "" {
		return `"` + variant.ContentHash + `"`
	}
	return ""
}

// formatJITETag formats a hash as a quoted ETag string: "jit-" followed by 16 hex digits.
// Uses direct hex encoding instead of fmt.Sprintf for zero memory allocations.
//
// Takes hash (uint64) which is the value to encode as hex digits.
//
// Returns string which is the formatted ETag in the form "jit-XXXXXXXXXXXXXXXX".
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

// extractOTelContextFromRequest extracts OpenTelemetry context from request headers.
//
// Uses a pooled MapCarrier to avoid per-request map allocation. If context was already
// extracted by earlier middleware (e.g., createTracingHandler), returns immediately
// without re-parsing headers.
//
// Takes r (*http.Request) which provides the incoming request with headers.
//
// Returns context.Context which contains the extracted OpenTelemetry trace context.
// Returns bool which is true if extraction was performed (context changed), false if
// context was already extracted and unchanged.
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

// cacheRuntimeKey names the framework bundles a server sends, for the static page cache
// key.
//
// Takes version (string) which is the runtime version of this binary, or empty.
// Takes build (daemon_frontend.CoreRuntimeBuild) which is the core runtime build sent.
//
// Returns string which joins the version and the core runtime file name.
func cacheRuntimeKey(version string, build daemon_frontend.CoreRuntimeBuild) string {
	return version + "/" + build.CoreFileName()
}

// generateCacheArtefactID creates a cache key from the request URL, locale, policy key
// and runtime key. Uses a pooled xxhash hasher to avoid extra memory use from string
// joining.
//
// The runtime key keeps a page cached by one build from being served by another, whose
// server would answer its versioned bundle URLs with 404, or send core runtime bytes that
// do not match the page's SRI hash.
//
// Takes r (*http.Request) which provides the URL and context for building the cache key.
// Takes policy (CachePolicy) which provides an optional key to include in the hash.
// Takes runtimeVersion (string) which names the bundles this server sends, or empty.
//
// Returns string which is the formatted cache key based on the hash.
func generateCacheArtefactID(r *http.Request, policy templater_dto.CachePolicy, runtimeVersion string) string {
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

	if runtimeVersion != "" {
		_, _ = hasher.Write(nullSeparator)
		_, _ = hasher.WriteString("runtime=")
		_, _ = hasher.WriteString(runtimeVersion)
	}

	sum := hasher.Sum64()
	hasher.Reset()
	xxhashDigestPool.Put(hasher)

	return formatCacheKey(sum)
}

// selectBestVariantForRequest picks the best variant based on Accept-Encoding. Priority
// order: brotli, gzip, minified-html, source.
//
// Takes artefact (*registry_dto.ArtefactMeta) which holds the available variants to
// choose from.
// Takes acceptEncoding (string) which lists the encodings the client accepts.
//
// Returns variantSelectionResult which holds the chosen variant and its name, or an empty
// result if no suitable variant is found.
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

// findVariantByTag searches for a variant with a given metadata tag value.
//
// Takes artefact (*registry_dto.ArtefactMeta) which holds the variants to search.
// Takes key (string) which is the metadata tag name to match.
// Takes value (string) which is the tag value to find.
//
// Returns *registry_dto.Variant which is the matching variant, or nil if no variant has
// the given tag value.
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
			header:     make(http.Header, pipeResponseWriterHeaderSize),
			Writer:     nil,
			statusCode: 0,
		}
	}
	prw.Writer = pw
	prw.statusCode = http.StatusOK
	clear(prw.header)
	return prw
}

// newPipeResponseWriter creates a new pipeResponseWriter for test compatibility.
//
// Takes pw (*io.PipeWriter) which receives the response body data.
//
// Returns *pipeResponseWriter which wraps the pipe writer with HTTP response handling.
func newPipeResponseWriter(pw *io.PipeWriter) *pipeResponseWriter {
	return &pipeResponseWriter{
		Writer:     pw,
		header:     make(http.Header),
		statusCode: http.StatusOK,
	}
}

// releasePipeResponseWriter returns a pipeResponseWriter to the pool, clearing the header
// map in place. Drops the writer when the header map exceeds
// pipeResponseWriterMaxRetainedHeaders.
//
// Takes prw (*pipeResponseWriter) which is the writer to release.
func releasePipeResponseWriter(prw *pipeResponseWriter) {
	prw.Writer = nil
	if len(prw.header) > pipeResponseWriterMaxRetainedHeaders {
		return
	}
	for k := range prw.header {
		delete(prw.header, k)
	}
	pipeResponseWriterPool.Put(prw)
}

// serveUpstreamIntoPipe runs next on a new goroutine, streaming its body into pw.
//
// Takes r (*http.Request) which is served by next under ctx.
// Takes next (http.Handler) which renders the response.
// Takes pw (*io.PipeWriter) which receives the body and is closed when next returns or
// panics, so the reader always sees the end of the stream.
//
// Returns <-chan upstreamResponse which receives the upstream response exactly once,
// after next has returned and pw is closed. Its status is the code next wrote, or 500
// when it panicked. Receiving from it joins the goroutine. A reader that stops early must
// close the pipe reader first, so a handler waiting to write is released.
func serveUpstreamIntoPipe(ctx context.Context, r *http.Request, next http.Handler, pw *io.PipeWriter) <-chan upstreamResponse {
	status := make(chan upstreamResponse, 1)
	go func() {
		response := upstreamResponse{forwardedHeader: nil, links: nil, statusCode: http.StatusInternalServerError}
		defer func() { status <- response }()
		defer func() { _ = pw.Close() }()
		defer goroutine.RecoverPanic(ctx, "daemon.cacheMiddlewareUpstream")

		recorder := getPipeResponseWriter(pw)
		next.ServeHTTP(recorder, r.WithContext(ctx))
		response = recorder.response()
		releasePipeResponseWriter(recorder)
	}()

	return status
}
