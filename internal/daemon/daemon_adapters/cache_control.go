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
	"fmt"
	"net/http"
	"strings"

	"piko.sh/piko/internal/registry/registry_dto"
)

// cacheControlForArtefact derives the Cache-Control policy for a served artefact variant.
//
// Takes disableHTTPCache (bool) which indicates whether HTTP caching is disabled (dev
// mode).
// Takes foundByStorageKey (bool) which is true when the URL is content-addressed.
// Takes variant (*registry_dto.Variant) which is the variant being served.
//
// Returns string which is the resolved Cache-Control header value.
func cacheControlForArtefact(disableHTTPCache, foundByStorageKey bool, variant *registry_dto.Variant) string {
	switch {
	case foundByStorageKey:
		return cacheControlForMode(disableHTTPCache, cacheControlImmutable)
	case isCosmeticMediaVariant(variant):
		return cacheControlForMode(disableHTTPCache, cacheControlMutableAsset)
	default:
		return cacheControlForMode(disableHTTPCache, jitCacheControl)
	}
}

// sitemapCacheControlValue builds the production Cache-Control value for sitemap and
// robots responses from the resolved max-age. A non-positive value (an operator setting
// CacheMaxAgeSeconds to 0 to disable caching) yields no-cache; otherwise it serves fresh
// for max-age with a one-day stale-while-revalidate window so crawlers can revalidate
// cheaply.
//
// Takes maxAgeSeconds (int) which is the configured sitemap cache duration in seconds.
//
// Returns string which is the Cache-Control value to use in production mode.
func sitemapCacheControlValue(maxAgeSeconds int) string {
	if maxAgeSeconds <= 0 {
		return cacheControlNoCache
	}
	return fmt.Sprintf("public, max-age=%d, stale-while-revalidate=86400", maxAgeSeconds)
}

// cacheControlForMode returns the appropriate Cache-Control header value, using no-cache
// to force ETag revalidation when HTTP caching is disabled (dev mode) and the provided
// production value when enabled (prod mode).
//
// Takes disableHTTPCache (bool) which indicates whether HTTP caching is disabled.
// Takes prodValue (string) which is the Cache-Control value to use in production mode.
//
// Returns string which is the resolved Cache-Control header value.
func cacheControlForMode(disableHTTPCache bool, prodValue string) string {
	if disableHTTPCache {
		return cacheControlNoCache
	}
	return prodValue
}

// isCosmeticMediaVariant reports whether a stable-URL variant is an image whose staleness
// has only cosmetic impact, letting it use the bounded mutable-asset tier rather than
// must-revalidate.
//
// Takes variant (*registry_dto.Variant) which is the variant being served.
//
// Returns bool which is true when the variant is cosmetic image media.
func isCosmeticMediaVariant(variant *registry_dto.Variant) bool {
	return variant != nil && strings.HasPrefix(variant.MimeType, "image/")
}

// setVariantCacheHeaders writes the caching headers shared by the 200 and 304 responses
// for a served variant: its ETag, the resolved Cache-Control policy, and Vary so that
// shared caches key on the negotiated encoding.
//
// Takes h (http.Header) which is the response header map to populate.
// Takes etag (string) which is the variant's entity tag.
// Takes cacheControl (string) which is the resolved Cache-Control value.
func setVariantCacheHeaders(h http.Header, etag, cacheControl string) {
	h[headerETag] = []string{etag}
	h[headerCacheControl] = []string{cacheControl}
	h[headerVary] = []string{headerAcceptEncoding}
}
