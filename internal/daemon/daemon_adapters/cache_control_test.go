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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/registry/registry_dto"
)

func TestCacheControlForArtefact(t *testing.T) {
	t.Parallel()

	jsVariant := &registry_dto.Variant{MimeType: "application/javascript"}
	cssVariant := &registry_dto.Variant{MimeType: "text/css; charset=utf-8"}
	tsSourceVariant := &registry_dto.Variant{MimeType: "text/vnd.trolltech.linguist; charset=utf-8"}
	mtsSourceVariant := &registry_dto.Variant{MimeType: "video/mp2t"}
	imageVariant := &registry_dto.Variant{MimeType: "image/avif"}

	testCases := []struct {
		variant           *registry_dto.Variant
		name              string
		want              string
		disableHTTPCache  bool
		foundByStorageKey bool
	}{

		{nil, "content-addressed is immutable", cacheControlImmutable, false, true},
		{imageVariant, "content-addressed is immutable even for image", cacheControlImmutable, false, true},

		{jsVariant, "stable-URL JS bundle revalidates", jitCacheControl, false, false},
		{cssVariant, "stable-URL CSS revalidates", jitCacheControl, false, false},

		{tsSourceVariant, "transpiled .ts source revalidates despite linguist mime", jitCacheControl, false, false},
		{mtsSourceVariant, "transpiled .mts source revalidates despite video mime", jitCacheControl, false, false},
		{nil, "unidentified stable-URL variant revalidates", jitCacheControl, false, false},

		{imageVariant, "stable-URL image uses bounded media tier", cacheControlMutableAsset, false, false},

		{nil, "dev mode no-cache for content-addressed", cacheControlNoCache, true, true},
		{jsVariant, "dev mode no-cache for stable JS", cacheControlNoCache, true, false},
		{imageVariant, "dev mode no-cache for image", cacheControlNoCache, true, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, cacheControlForArtefact(tc.disableHTTPCache, tc.foundByStorageKey, tc.variant))
		})
	}
}

func TestSitemapCacheControlValue(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		want          string
		maxAgeSeconds int
	}{
		{"zero disables caching with no-cache", cacheControlNoCache, 0},
		{"negative disables caching with no-cache", cacheControlNoCache, -1},
		{"one minute serves public with revalidation window", "public, max-age=60, stale-while-revalidate=86400", 60},
		{"ten minutes serves public with revalidation window", "public, max-age=600, stale-while-revalidate=86400", 600},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, sitemapCacheControlValue(tc.maxAgeSeconds))
		})
	}
}

func TestCacheControlForMode_DisabledReturnsNoCache(t *testing.T) {
	t.Parallel()

	assert.Equal(t, cacheControlNoCache, cacheControlForMode(true, cacheControlMutableAsset))
}

func TestCacheControlForMode_EnabledReturnsProdValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, cacheControlMutableAsset, cacheControlForMode(false, cacheControlMutableAsset))
	assert.Equal(t, cacheControlImmutable, cacheControlForMode(false, cacheControlImmutable))
}

func TestIsCosmeticMediaVariant(t *testing.T) {
	t.Parallel()

	assert.True(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: "image/png"}))
	assert.True(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: "image/avif"}))
	assert.True(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: "image/svg+xml"}))

	assert.False(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: "application/javascript"}))
	assert.False(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: "text/css; charset=utf-8"}))
	assert.False(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: "video/mp2t"}))
	assert.False(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: "application/manifest+json"}))
	assert.False(t, isCosmeticMediaVariant(&registry_dto.Variant{}))
	assert.False(t, isCosmeticMediaVariant(nil))
}

func TestSetVariantCacheHeaders(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	setVariantCacheHeaders(header, `"etag-1"`, jitCacheControl)

	assert.Equal(t, `"etag-1"`, header.Get(headerETag))
	assert.Equal(t, jitCacheControl, header.Get(headerCacheControl))
	assert.Equal(t, headerAcceptEncoding, header.Get(headerVary))
}
