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
	"sync"
	"time"

	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// defaultReadTimeout is the time limit for reading a complete request.
	defaultReadTimeout = 5 * time.Second

	// defaultWriteTimeout is the longest time allowed to write a response.
	defaultWriteTimeout = 10 * time.Second

	// defaultIdleTimeout is the maximum time to wait for the next request on a
	// keep-alive connection.
	defaultIdleTimeout = 120 * time.Second

	// defaultReadHeaderTimeout is the time allowed to read request headers.
	defaultReadHeaderTimeout = 2 * time.Second

	// defaultMaxHeaderBytes is the maximum size for request headers (1 MB).
	defaultMaxHeaderBytes = 1 << 20

	// defaultCacheWriteConcurrency is the default number of concurrent cache write
	// operations.
	defaultCacheWriteConcurrency = 10

	// defaultStreamCompressionLevel is the default compression level for streaming
	// responses.
	defaultStreamCompressionLevel = 4

	// headerContentType is the HTTP header name for specifying the content type.
	headerContentType = "Content-Type"

	// headerContentEncoding is the HTTP header key for specifying content
	// encoding.
	headerContentEncoding = "Content-Encoding"

	// headerContentLength is the HTTP header name for specifying response size.
	headerContentLength = "Content-Length"

	// headerCacheControl is the HTTP header name for cache control settings.
	headerCacheControl = "Cache-Control"

	// headerETag is the HTTP header name for entity tags used for cache control.
	// Canonical MIME form is "Etag" (CanonicalMIMEHeaderKey lowercases after
	// the first character of each hyphen-separated segment).
	headerETag = "Etag"

	// headerIfNoneMatch is the HTTP header for conditional requests using ETags.
	headerIfNoneMatch = "If-None-Match"

	// headerVary is the HTTP header name for content negotiation hints.
	headerVary = "Vary"

	// headerLink is the HTTP header name for link relations.
	headerLink = "Link"

	// encodingBrotli is the Content-Encoding header value for Brotli compression.
	encodingBrotli = "br"

	// encodingGzip is the gzip content encoding value for HTTP headers.
	encodingGzip = "gzip"

	// jitCacheControl is the Cache-Control value for JIT-generated responses.
	// These responses are served immediately but must be revalidated.
	jitCacheControl = "public, max-age=0, must-revalidate"

	// cacheControlNoCache forces ETag revalidation on every request. Used when
	// DisableHTTPCache is true (dev mode).
	cacheControlNoCache = "public, no-cache"

	// cacheControlMutableAsset is the prod Cache-Control for assets that change
	// when a project is updated (theme, sitemaps, robots.txt, HLS playlists,
	// embedded frontend). Fresh for one hour, stale-while-revalidate for one day.
	cacheControlMutableAsset = "public, max-age=3600, stale-while-revalidate=86400"

	// cacheControlLongLived is the prod Cache-Control for user artefacts (images,
	// media). Fresh for one day, stale-while-revalidate for one week.
	cacheControlLongLived = "public, max-age=86400, stale-while-revalidate=604800"

	// cacheControlImmutable is the Cache-Control for content-addressed assets
	// that never change (video chunks). Used regardless of dev/prod mode.
	cacheControlImmutable = "public, max-age=31536000, immutable"

	// contentTypeJSON is the MIME type for JSON content in HTTP responses.
	contentTypeJSON = "application/json"

	// contentTypeHTML is the MIME type for HTML content with UTF-8 encoding.
	contentTypeHTML = "text/html; charset=utf-8"

	// contentTypeMPEGURL is the MIME type for HLS playlist responses.
	contentTypeMPEGURL = "application/x-mpegURL"

	// contentTypeTextPlain is the MIME type for plain text responses.
	contentTypeTextPlain = "text/plain; charset=utf-8"

	// contentTypeCSS is the MIME type for CSS files with UTF-8 encoding.
	contentTypeCSS = "text/css; charset=utf-8"

	// contentTypeXML is the MIME type for XML content with UTF-8 encoding.
	contentTypeXML = "application/xml; charset=utf-8"

	// errMessageInternalServer is the error message returned to clients for internal
	// server errors.
	errMessageInternalServer = "Internal Server Error"

	// methodGET is the HTTP GET method string.
	methodGET = "GET"

	// methodPOST is the HTTP POST method used for page and partial routes.
	methodPOST = "POST"

	// variantSource is the variant identifier for original source content.
	variantSource = "source"

	// hlsVariantPrefix is the prefix added to quality names to form HLS variant
	// IDs.
	hlsVariantPrefix = "hls_"

	// hlsDefaultSegmentDuration is the default duration in seconds for HLS
	// segments.
	hlsDefaultSegmentDuration = 10.0

	// logFieldQuality is the log field key for video quality level.
	logFieldQuality = "quality"

	// logFieldChunkID is the log field key for chunk identifiers.
	logFieldChunkID = "chunkID"

	// msgVariantNotFound is the log message used when a video variant cannot be
	// found.
	msgVariantNotFound = "Variant not found"

	// logFieldPath is the log field key for request URL paths.
	logFieldPath = "path"

	// logFieldMethod is the log field key for the HTTP request method.
	logFieldMethod = "method"

	// logFieldArtefactID is the log field key for artefact identifiers.
	logFieldArtefactID = "artefactID"

	// logFieldVariantID is the log field key for variant IDs.
	logFieldVariantID = "variantID"

	// logFieldError is the key for error messages in structured logging.
	logFieldError = "error"

	// logFieldURL is the logging field key for the request URL.
	logFieldURL = "url"

	// logFieldOriginalPath is the log field key for the original template path.
	logFieldOriginalPath = "originalPath"

	// logFieldRoutePattern is the log field key for the URL route pattern.
	logFieldRoutePattern = "routePattern"

	// logFieldChiPattern is the log field key for the chi-translated route
	// pattern (e.g. "/docs/*" derived from "/docs/{slug:.+}").
	logFieldChiPattern = "chiPattern"

	// logFieldLocale is the log field key for the request locale.
	logFieldLocale = "locale"

	// logFieldI18nStrategy is the log field key for the i18n routing strategy.
	logFieldI18nStrategy = "i18nStrategy"

	// logFieldIsE2EOnly is the log field key indicating whether a route is
	// only registered when e2e mode is enabled.
	logFieldIsE2EOnly = "isE2EOnly"

	// logFieldActionCount is the log field key for the number of actions.
	logFieldActionCount = "actionCount"

	// logFieldActionName is the log field key for the action name.
	logFieldActionName = "actionName"

	// logFieldBytesWritten is the log field key for the number of bytes written.
	logFieldBytesWritten = "bytesWritten"

	// logFieldSelectedVar is the span attribute key for the selected variant name.
	logFieldSelectedVar = "selectedVariant"

	// logFieldETagMatch is the tracing attribute key that records whether the
	// client's ETag matched the cached resource.
	logFieldETagMatch = "etagMatch"

	// logFieldContentEnc is the log field key for content encoding.
	logFieldContentEnc = "contentEncoding"

	// logFieldHandler is the log field key for the handler name.
	logFieldHandler = "handler"

	// logFieldRouter is the log field key for HTTP router operations.
	logFieldRouter = "router"

	// logFieldHTTPBuilder is the log field name for HTTPRouterBuilder entries.
	logFieldHTTPBuilder = "HTTPRouterBuilder"

	// logFieldCacheFile is the log field value for the cache middleware file.
	logFieldCacheFile = "driven_middleware_cache.go"

	// logFieldHTTPFile is the file name used for log field identification in HTTP
	// handlers.
	logFieldHTTPFile = "driven_http.go"

	// logFieldHTTPRouterFile is the filename used in structured log entries for
	// HTTP router operations.
	logFieldHTTPRouterFile = "driven_http_router.go"

	// logFieldProfileName is the log field key for the image profile name.
	logFieldProfileName = "profileName"

	// headerXCacheStatus is the canonical MIME form of the cache status header.
	headerXCacheStatus = "X-Cache-Status"

	// headerXPPResponseSupport is the canonical MIME form of the response
	// support negotiation header. CanonicalMIMEHeaderKey("X-PP-Response-Support")
	// lowercases "PP" to "Pp".
	headerXPPResponseSupport = "X-Pp-Response-Support"
)

var (
	// pageDefPool is a sync.Pool for reusing PageDefinition structs to reduce
	// allocations in high-throughput page request handling.
	pageDefPool = sync.Pool{
		New: func() any {
			return &templater_dto.PageDefinition{}
		},
	}

	// headerValContentTypeHTML is a pre-allocated header value
	// slice for zero-alloc direct map assignment.
	//
	// Assigned to the response header map via h[key] = slice,
	// bypassing Header.Set() and its per-call []string +
	// CanonicalMIMEHeaderKey allocations. The stdlib only reads
	// these slices during response writing, never mutating them,
	// so sharing across concurrent requests is safe.
	headerValContentTypeHTML = []string{contentTypeHTML}

	// headerValContentTypeMPEGURL is a pre-allocated header value
	// slice for HLS playlist content type.
	headerValContentTypeMPEGURL = []string{contentTypeMPEGURL}

	// headerValContentTypeText is a pre-allocated header value
	// slice for plain text content type.
	headerValContentTypeText = []string{contentTypeTextPlain}

	// headerValEncodingBrotli is a pre-allocated header value slice for Brotli content encoding.
	headerValEncodingBrotli = []string{encodingBrotli}

	// headerValEncodingGzip is a pre-allocated header value slice for gzip content encoding.
	headerValEncodingGzip = []string{encodingGzip}

	// headerValCacheImmutable is a pre-allocated header value slice for immutable cache control.
	headerValCacheImmutable = []string{cacheControlImmutable}

	// headerValFragmentPatch is a pre-allocated header value
	// slice for fragment-patch response support.
	headerValFragmentPatch = []string{"fragment-patch"}

	// headerValCacheStatusMiss is a pre-allocated header value slice for a cache miss status.
	headerValCacheStatusMiss = []string{"MISS"}

	// headerValCacheStatusHit is a pre-allocated header value slice for a cache hit status.
	headerValCacheStatusHit = []string{"HIT"}

	// headerValCacheStatusStale is a pre-allocated header value slice for a stale cache status.
	headerValCacheStatusStale = []string{"STALE"}

	// headerValVaryCachePage is the Vary header for cached page responses
	// without i18n. Accept-Encoding because different compressed variants
	// are served.
	headerValVaryCachePage = []string{"Accept-Encoding"}

	// headerValVaryCachePageI18n is the Vary header for cached page
	// responses with i18n enabled. Includes Accept-Language because the
	// cache key varies by locale.
	headerValVaryCachePageI18n = []string{"Accept-Encoding, Accept-Language"}
)
