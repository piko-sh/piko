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
	"slices"
	"strings"

	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
)

const (
	// varyAnyToken is the Vary value that says the response varies on everything.
	varyAnyToken = "*"

	// varyTokenSeparator separates the tokens in a Vary header value.
	varyTokenSeparator = ","

	// varyJoinedSeparator joins merged Vary tokens into one header value.
	varyJoinedSeparator = ", "

	// varyTokenCapacity is the starting capacity for the merged list of Vary tokens.
	varyTokenCapacity = 8
)

// setDefaultCacheControl gives a response a default Cache-Control unless it already has
// one. A policy that the app, a page or an action set earlier always wins.
//
// An entry set to nil also counts as set, because Go uses nil to leave a header out.
//
// Takes h (http.Header) which is the response header map.
// Takes value ([]string) which is the default policy as a header value.
func setDefaultCacheControl(h http.Header, value []string) {
	if _, set := h[headerCacheControl]; set {
		return
	}
	h[headerCacheControl] = value
}

// setNoStoreCacheControl marks a response as one that no cache may keep, for a page whose
// cache policy sets NoStore. A Cache-Control that the app or the page set earlier wins.
//
// Takes h (http.Header) which is the response header map.
func setNoStoreCacheControl(h http.Header) {
	setDefaultCacheControl(h, headerValCacheNoStore)
}

// mergeVary adds Vary tokens to a response and keeps the tokens the app or the page set
// earlier, in their order and spelling.
//
// Tokens compare without case, so no token is listed twice. A Vary of "*" already covers
// every token, so it is left as it is. An entry set to nil counts as unset, because a
// response that leaves out the tokens given here would let a shared cache serve the wrong
// variant.
//
// Takes h (http.Header) which is the response header map.
// Takes values ([]string) which are the Vary header values to add.
func mergeVary(h http.Header, values []string) {
	existing := h[headerVary]
	if len(existing) == 0 {
		h[headerVary] = values
		return
	}

	tokens := appendVaryTokens(make([]string, 0, varyTokenCapacity), existing)
	if slices.Contains(tokens, varyAnyToken) {
		return
	}
	tokens = appendVaryTokens(tokens, values)
	h[headerVary] = []string{strings.Join(tokens, varyJoinedSeparator)}
}

// appendVaryTokens splits Vary header values into tokens and appends each one that is not
// already listed.
//
// Takes tokens ([]string) which holds the tokens found so far.
// Takes values ([]string) which are the Vary header values to split.
//
// Returns []string which holds the tokens with the new ones appended.
func appendVaryTokens(tokens, values []string) []string {
	for _, value := range values {
		for token := range strings.SplitSeq(value, varyTokenSeparator) {
			token = strings.TrimSpace(token)
			if token == "" || containsTokenFold(tokens, token) {
				continue
			}
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// containsTokenFold reports whether a token is in the list, ignoring case.
//
// Takes tokens ([]string) which is the list to search.
// Takes token (string) which is the token to find.
//
// Returns bool which is true when the list holds the token.
func containsTokenFold(tokens []string, token string) bool {
	return slices.ContainsFunc(tokens, func(listed string) bool {
		return strings.EqualFold(listed, token)
	})
}

// resolveCachePolicy works out a page's cache policy for a request.
//
// The policy sees the request's path, query, cookies and locale, but never its body. The
// body is left for the page, so it is read once, and a large or malformed upload never
// costs anything for a page that does not read it.
//
// Takes r (*http.Request) which is the incoming request.
// Takes pageEntry (templater_domain.PageEntryView) which holds the page's cache policy.
//
// Returns templater_dto.CachePolicy which is the page's policy for this request.
func resolveCachePolicy(r *http.Request, pageEntry templater_domain.PageEntryView) templater_dto.CachePolicy {
	requestData := templater_domain.NewRequestDataFromAction(r)
	defer requestData.Release()
	return pageEntry.GetCachePolicy(requestData)
}

// noStorePolicyMiddleware sends Cache-Control: no-store for a page whose cache policy
// sets NoStore, on routes that the static page cache does not wrap.
//
// Takes next (http.Handler) which serves the page.
// Takes pageEntry (templater_domain.PageEntryView) which holds the page's cache policy.
//
// Returns http.Handler which applies the policy and then calls next.
func noStorePolicyMiddleware(next http.Handler, pageEntry templater_domain.PageEntryView) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if resolveCachePolicy(r, pageEntry).NoStore {
			setNoStoreCacheControl(w.Header())
		}
		next.ServeHTTP(w, r)
	})
}

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
// shared caches key on the negotiated encoding. A Vary set earlier is merged, not
// replaced.
//
// Takes h (http.Header) which is the response header map to populate.
// Takes etag (string) which is the variant's entity tag.
// Takes cacheControl (string) which is the resolved Cache-Control value.
func setVariantCacheHeaders(h http.Header, etag, cacheControl string) {
	h[headerETag] = []string{etag}
	h[headerCacheControl] = []string{cacheControl}
	mergeVary(h, headerValVaryAcceptEncoding)
}

// ifNoneMatchSatisfied reports whether an If-None-Match header names the entity tag of
// the representation being served, so the server can answer 304 Not Modified.
//
// It uses the weak comparison that RFC 9110 requires for If-None-Match. A "W/" prefix on
// either side is ignored, a list matches when any member matches, and "*" matches any
// representation. A malformed list stops the scan, which only costs the client a full
// response.
//
// Takes ifNoneMatch (string) which is the raw If-None-Match header value.
// Takes etag (string) which is the entity tag of the representation being served.
//
// Returns bool which is true when the client already holds this representation.
func ifNoneMatchSatisfied(ifNoneMatch, etag string) bool {
	if ifNoneMatch == "" || etag == "" {
		return false
	}
	if ifNoneMatch == etag {
		return true
	}
	target := strings.TrimPrefix(etag, weakETagPrefix)

	remaining := ifNoneMatch
	for {
		remaining = strings.TrimLeft(remaining, ifNoneMatchSeparators)
		if remaining == "" {
			return false
		}
		if remaining[0] == ifNoneMatchAny {
			return true
		}

		candidate, rest, ok := cutEntityTag(remaining)
		if !ok {
			return false
		}
		if candidate == target {
			return true
		}
		remaining = rest
	}
}

// cutEntityTag splits the first entity tag off the front of an If-None-Match list.
//
// Takes list (string) which starts with an entity tag, weak or strong.
//
// Returns opaque (string) which is the quoted tag without any weak prefix.
// Returns rest (string) which is the text after the tag.
// Returns ok (bool) which is false when the list does not start with a quoted tag.
func cutEntityTag(list string) (opaque, rest string, ok bool) {
	afterOpening, opened := strings.CutPrefix(strings.TrimPrefix(list, weakETagPrefix), entityTagQuote)
	if !opened {
		return "", "", false
	}
	tag, rest, closed := strings.Cut(afterOpening, entityTagQuote)
	if !closed {
		return "", "", false
	}
	return entityTagQuote + tag + entityTagQuote, rest, true
}
