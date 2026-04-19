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

package seo_adapters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPSourceAdapter_FetchURLs_ValidJSONArray(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "application/json", request.Header.Get("Accept"))
		assert.Equal(t, "Piko-SEO-Service/1.0", request.Header.Get("User-Agent"))

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
			{"loc": "/about", "lastmod": "2026-01-15", "changefreq": "monthly", "priority": 0.8},
			{"loc": "/blog/post-one", "lastmod": "2026-03-01", "changefreq": "weekly", "priority": 0.6}
		]`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	urls, err := adapter.FetchURLs(context.Background(), server.URL)
	require.NoError(t, err)
	require.Len(t, urls, 2)

	assert.Equal(t, "/about", urls[0].Location)
	assert.Equal(t, "2026-01-15", urls[0].LastMod)
	assert.Equal(t, "monthly", urls[0].ChangeFreq)
	assert.InDelta(t, 0.8, float64(urls[0].Priority), 0.01)

	assert.Equal(t, "/blog/post-one", urls[1].Location)
	assert.Equal(t, "weekly", urls[1].ChangeFreq)
}

func TestHTTPSourceAdapter_FetchURLs_EmptyArray(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[]`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	urls, err := adapter.FetchURLs(context.Background(), server.URL)
	require.NoError(t, err)
	assert.Empty(t, urls)
}

func TestHTTPSourceAdapter_FetchURLs_ServerError500(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte(`Internal Server Error`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	_, err := adapter.FetchURLs(context.Background(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestHTTPSourceAdapter_FetchURLs_InvalidJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{this is not valid json`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	_, err := adapter.FetchURLs(context.Background(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding JSON response")
}

func TestHTTPSourceAdapter_FetchURLs_ServerNotFound404(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	_, err := adapter.FetchURLs(context.Background(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestHTTPSourceAdapter_FetchURLs_CancelledContext(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[]`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := adapter.FetchURLs(ctx, server.URL)
	require.Error(t, err)
}

func TestHTTPSourceAdapter_FetchURLs_InvalidURL(t *testing.T) {
	t.Parallel()

	adapter := &HTTPSourceAdapter{
		httpClient: http.DefaultClient,
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	_, err := adapter.FetchURLs(context.Background(), "://invalid-url")
	require.Error(t, err)
}

func TestHTTPSourceAdapter_FetchURLs_WithImagesAndVideos(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
			{
				"loc": "/gallery",
				"images": ["https://example.com/img1.jpg", "https://example.com/img2.png"],
				"videos": [
					{
						"title": "Demo Video",
						"description": "A demonstration",
						"thumbnailLoc": "https://example.com/thumb.jpg",
						"duration": 120
					}
				]
			}
		]`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	urls, err := adapter.FetchURLs(context.Background(), server.URL)
	require.NoError(t, err)
	require.Len(t, urls, 1)

	assert.Equal(t, "/gallery", urls[0].Location)
	assert.Len(t, urls[0].Images, 2)
	assert.Equal(t, "https://example.com/img1.jpg", urls[0].Images[0])

	require.Len(t, urls[0].Videos, 1)
	assert.Equal(t, "Demo Video", urls[0].Videos[0].Title)
	assert.Equal(t, 120, urls[0].Videos[0].Duration)
}

func TestHTTPSourceAdapter_FetchURLs_ServiceUnavailable503(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	_, err := adapter.FetchURLs(context.Background(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestHTTPSourceAdapter_FetchURLs_EmptyResponseBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	_, err := adapter.FetchURLs(context.Background(), server.URL)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decoding JSON response")
}

func TestHTTPSourceAdapter_FetchURLs_WithNewsEntry(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[
			{
				"loc": "/news/breaking",
				"news": {
					"publicationName": "The Example Times",
					"publicationLanguage": "en",
					"publicationDate": "2026-03-27",
					"title": "Breaking News Story"
				}
			}
		]`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	urls, err := adapter.FetchURLs(context.Background(), server.URL)
	require.NoError(t, err)
	require.Len(t, urls, 1)

	assert.Equal(t, "/news/breaking", urls[0].Location)
	require.NotNil(t, urls[0].News)
	assert.Equal(t, "The Example Times", urls[0].News.PublicationName)
	assert.Equal(t, "en", urls[0].News.PublicationLanguage)
	assert.Equal(t, "Breaking News Story", urls[0].News.Title)
}

func TestHTTPSourceAdapter_FetchURLs_SingleURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[{"loc": "/"}]`))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient: server.Client(),
		breaker:    newHTTPSourceCircuitBreaker(),
	}

	urls, err := adapter.FetchURLs(context.Background(), server.URL)
	require.NoError(t, err)
	require.Len(t, urls, 1)
	assert.Equal(t, "/", urls[0].Location)
}

func TestHTTPSourceAdapter_FetchURLs_RejectsResponseLargerThanCap(t *testing.T) {
	t.Parallel()

	const responseCap = int64(256)
	padding := strings.Repeat("a", int(responseCap)+512)
	body := `[{"loc": "/oversized", "title": "` + padding + `"}]`

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient:       server.Client(),
		breaker:          newHTTPSourceCircuitBreaker(),
		maxResponseBytes: responseCap,
	}

	_, err := adapter.FetchURLs(context.Background(), server.URL)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSitemapResponseTooLarge)
	assert.Contains(t, err.Error(), "256")
}

func TestHTTPSourceAdapter_FetchURLs_AcceptsResponseAtExactlyCap(t *testing.T) {
	t.Parallel()

	body := []byte(`[{"loc": "/"}]`)

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write(body)
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient:       server.Client(),
		breaker:          newHTTPSourceCircuitBreaker(),
		maxResponseBytes: int64(len(body)),
	}

	urls, err := adapter.FetchURLs(context.Background(), server.URL)
	require.NoError(t, err)
	require.Len(t, urls, 1)
	assert.Equal(t, "/", urls[0].Location)
}

func TestHTTPSourceAdapter_FetchURLs_DisabledCapAcceptsLargeResponse(t *testing.T) {
	t.Parallel()

	padding := strings.Repeat("b", 4096)
	body := `[{"loc": "/big", "title": "` + padding + `"}]`

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(body))
	}))
	defer server.Close()

	adapter := &HTTPSourceAdapter{
		httpClient:       server.Client(),
		breaker:          newHTTPSourceCircuitBreaker(),
		maxResponseBytes: 0,
	}

	urls, err := adapter.FetchURLs(context.Background(), server.URL)
	require.NoError(t, err)
	require.Len(t, urls, 1)
	assert.Equal(t, "/big", urls[0].Location)
}

func TestNewHTTPSourceAdapter_ApplyMaxResponseBytesOption(t *testing.T) {
	t.Parallel()

	adapter, ok := NewHTTPSourceAdapter(WithHTTPSourceMaxResponseBytes(2048)).(*HTTPSourceAdapter)
	require.True(t, ok, "expected concrete *HTTPSourceAdapter")
	assert.Equal(t, int64(2048), adapter.maxResponseBytes)
}

func TestNewHTTPSourceAdapter_DefaultsResponseCap(t *testing.T) {
	t.Parallel()

	adapter, ok := NewHTTPSourceAdapter().(*HTTPSourceAdapter)
	require.True(t, ok, "expected concrete *HTTPSourceAdapter")
	assert.Equal(t, defaultMaxSitemapResponseBytes, adapter.maxResponseBytes)
}

func TestErrSitemapResponseTooLarge_IsSentinel(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(errors.New("preceding"), ErrSitemapResponseTooLarge)
	assert.ErrorIs(t, wrapped, ErrSitemapResponseTooLarge)
}
