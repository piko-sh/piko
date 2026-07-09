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
		name              string
		disableHTTPCache  bool
		foundByStorageKey bool
		variant           *registry_dto.Variant
		want              string
	}{

		{"content-addressed stays long-lived", false, true, nil, cacheControlLongLived},
		{"content-addressed long-lived even for image", false, true, imageVariant, cacheControlLongLived},

		{"stable-URL JS bundle revalidates", false, false, jsVariant, jitCacheControl},
		{"stable-URL CSS revalidates", false, false, cssVariant, jitCacheControl},

		{"transpiled .ts source revalidates despite linguist mime", false, false, tsSourceVariant, jitCacheControl},
		{"transpiled .mts source revalidates despite video mime", false, false, mtsSourceVariant, jitCacheControl},
		{"unidentified stable-URL variant revalidates", false, false, nil, jitCacheControl},

		{"stable-URL image uses bounded media tier", false, false, imageVariant, cacheControlMutableAsset},

		{"dev mode no-cache for content-addressed", true, true, nil, cacheControlNoCache},
		{"dev mode no-cache for stable JS", true, false, jsVariant, cacheControlNoCache},
		{"dev mode no-cache for image", true, false, imageVariant, cacheControlNoCache},
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
		maxAgeSeconds int
		want          string
	}{
		{"zero disables caching with no-cache", 0, cacheControlNoCache},
		{"negative disables caching with no-cache", -1, cacheControlNoCache},
		{"one minute serves public with revalidation window", 60, "public, max-age=60, stale-while-revalidate=86400"},
		{"ten minutes serves public with revalidation window", 600, "public, max-age=600, stale-while-revalidate=86400"},
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
	assert.Equal(t, cacheControlLongLived, cacheControlForMode(false, cacheControlLongLived))
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
	assert.False(t, isCosmeticMediaVariant(&registry_dto.Variant{MimeType: ""}))
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
