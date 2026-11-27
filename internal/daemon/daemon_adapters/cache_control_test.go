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
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
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

func TestSetVariantCacheHeadersMergesVary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		existing string
		want     string
	}{
		{name: "app token is kept", existing: "Origin", want: "Origin, Accept-Encoding"},
		{name: "same token is not repeated", existing: "accept-encoding", want: "accept-encoding"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			header := http.Header{}
			header.Set(headerVary, testCase.existing)

			setVariantCacheHeaders(header, `"etag-1"`, jitCacheControl)

			assert.Equal(t, []string{testCase.want}, header.Values(headerVary))
		})
	}
}

func TestIfNoneMatchSatisfied(t *testing.T) {
	t.Parallel()

	const etag = `"abc123"`

	testCases := []struct {
		name        string
		ifNoneMatch string
		etag        string
		want        bool
	}{
		{name: "exact strong tag matches", ifNoneMatch: etag, etag: etag, want: true},
		{name: "weak form of the tag matches", ifNoneMatch: `W/"abc123"`, etag: etag, want: true},
		{name: "weak served tag matches its strong form", ifNoneMatch: etag, etag: `W/"abc123"`, want: true},
		{name: "tag later in a list matches", ifNoneMatch: `"old", W/"older", "abc123"`, etag: etag, want: true},
		{name: "list without spaces matches", ifNoneMatch: `"old","abc123"`, etag: etag, want: true},
		{name: "wildcard matches any representation", ifNoneMatch: "*", etag: etag, want: true},
		{name: "tag from an earlier release does not match", ifNoneMatch: `"old"`, etag: etag, want: false},
		{name: "tag that only shares a prefix does not match", ifNoneMatch: `"abc1234"`, etag: etag, want: false},
		{name: "tag that only shares a suffix does not match", ifNoneMatch: `"xabc123"`, etag: etag, want: false},
		{name: "comma inside a quoted tag is part of the tag", ifNoneMatch: `"x,abc123"`, etag: etag, want: false},
		{name: "unquoted value does not match", ifNoneMatch: "abc123", etag: etag, want: false},
		{name: "unterminated tag does not match", ifNoneMatch: `"abc123`, etag: etag, want: false},
		{name: "malformed member stops the scan", ifNoneMatch: `junk, "abc123"`, etag: etag, want: false},
		{name: "empty header does not match", ifNoneMatch: "", etag: etag, want: false},
		{name: "separators only do not match", ifNoneMatch: " , ", etag: etag, want: false},
		{name: "missing served tag never matches", ifNoneMatch: "*", etag: "", want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ifNoneMatchSatisfied(tc.ifNoneMatch, tc.etag))
		})
	}
}

func TestCutEntityTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		list       string
		wantOpaque string
		wantRest   string
		wantOK     bool
	}{
		{name: "strong tag", list: `"a", "b"`, wantOpaque: `"a"`, wantRest: `, "b"`, wantOK: true},
		{name: "weak tag loses its prefix", list: `W/"a"`, wantOpaque: `"a"`, wantRest: "", wantOK: true},
		{name: "empty tag", list: `""`, wantOpaque: `""`, wantRest: "", wantOK: true},
		{name: "unquoted", list: "a", wantOK: false},
		{name: "unterminated", list: `"a`, wantOK: false},
		{name: "bare weak prefix", list: "W/", wantOK: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opaque, rest, ok := cutEntityTag(tc.list)

			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantOpaque, opaque)
			assert.Equal(t, tc.wantRest, rest)
		})
	}
}

func TestMergeVary(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		existing []string
		values   []string
		want     []string
	}{
		{name: "no earlier Vary", values: []string{"Accept-Encoding"}, want: []string{"Accept-Encoding"}},
		{name: "earlier Vary set to nil", existing: nil, values: []string{"Accept-Encoding"}, want: []string{"Accept-Encoding"}},
		{
			name:     "earlier token kept first",
			existing: []string{"Origin"},
			values:   []string{"Accept-Encoding, Accept-Language"},
			want:     []string{"Origin, Accept-Encoding, Accept-Language"},
		},
		{
			name:     "token in another case is not repeated",
			existing: []string{"accept-encoding"},
			values:   []string{"Accept-Encoding, Accept-Language"},
			want:     []string{"accept-encoding, Accept-Language"},
		},
		{
			name:     "earlier lines and repeats are folded together",
			existing: []string{"Origin, Cookie", "origin", " , Cookie "},
			values:   []string{"Accept-Encoding"},
			want:     []string{"Origin, Cookie, Accept-Encoding"},
		},
		{name: "any token is left alone", existing: []string{"Origin, *"}, values: []string{"Accept-Encoding"}, want: []string{"Origin, *"}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			h := http.Header{}
			if testCase.existing != nil {
				h[headerVary] = testCase.existing
			}

			mergeVary(h, testCase.values)

			assert.Equal(t, testCase.want, h[headerVary])
		})
	}
}

func newRoutedRequest(t *testing.T, path string) *http.Request {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
}

func TestNoStorePolicyMiddleware(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		appCacheControl  string
		wantCacheControl string
		policy           templater_dto.CachePolicy
	}{
		{name: "NoStore", policy: templater_dto.CachePolicy{NoStore: true}, wantCacheControl: cacheControlNoStore},
		{name: "no NoStore keeps the page default", policy: templater_dto.CachePolicy{Enabled: true}, wantCacheControl: cacheControlPrivateNoCache},
		{
			name:             "app Cache-Control wins over NoStore",
			policy:           templater_dto.CachePolicy{NoStore: true},
			appCacheControl:  "private, max-age=10",
			wantCacheControl: "private, max-age=10",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			entry := &templater_domain.MockPageEntryView{
				GetCachePolicyFunc: func(*templater_dto.RequestData) templater_dto.CachePolicy { return testCase.policy },
			}
			page := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				setDefaultCacheControl(w.Header(), headerValCachePrivateNoCache)
				w.WriteHeader(http.StatusOK)
			})
			recorder := httptest.NewRecorder()
			if testCase.appCacheControl != "" {
				recorder.Header().Set("Cache-Control", testCase.appCacheControl)
			}

			noStorePolicyMiddleware(page, entry).ServeHTTP(recorder, newRoutedRequest(t, "/account"))

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Equal(t, []string{testCase.wantCacheControl}, recorder.Header().Values("Cache-Control"))
		})
	}
}

type readRecordingBody struct {
	io.Reader
	read bool
}

func (body *readRecordingBody) Read(p []byte) (int, error) {
	body.read = true
	return body.Reader.Read(p)
}

func (*readRecordingBody) Close() error { return nil }

func TestResolveCachePolicyLeavesTheBodyForThePage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "url-encoded form", contentType: "application/x-www-form-urlencoded", body: "name=Ada"},
		{name: "multipart form", contentType: "multipart/form-data; boundary=edge", body: "--edge\r\nContent-Disposition: form-data; name=\"name\"\r\n\r\nAda\r\n--edge--\r\n"},
		{name: "malformed multipart form", contentType: "multipart/form-data; boundary=edge", body: "not multipart"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var seenQuery string
			entry := &templater_domain.MockPageEntryView{
				GetCachePolicyFunc: func(requestData *templater_dto.RequestData) templater_dto.CachePolicy {
					seenQuery = requestData.QueryParam("variant")
					return templater_dto.CachePolicy{NoStore: true}
				},
			}
			body := &readRecordingBody{Reader: strings.NewReader(testCase.body)}
			request := newRoutedRequest(t, "/account?variant=b")
			request.Method = http.MethodPost
			request.Body = body
			request.Header.Set("Content-Type", testCase.contentType)

			policy := resolveCachePolicy(request, entry)

			assert.True(t, policy.NoStore)
			assert.Equal(t, "b", seenQuery, "the policy still sees the query")
			assert.False(t, body.read, "the policy must not read the request body")
		})
	}
}
