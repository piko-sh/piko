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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/capabilities/capabilities_dto"
	"piko.sh/piko/internal/daemon/daemon_dto"
	"piko.sh/piko/internal/ratelimiter/ratelimiter_dto"
	"piko.sh/piko/internal/registry/registry_domain"
	"piko.sh/piko/internal/registry/registry_dto"
	"piko.sh/piko/internal/security/security_domain"
	"piko.sh/piko/internal/security/security_dto"
	"piko.sh/piko/internal/templater/templater_domain"
	"piko.sh/piko/internal/templater/templater_dto"
	"piko.sh/piko/wdk/clock"
)

func TestFormatCacheKey(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		hash     uint64
	}{
		{
			name:     "formats zero hash",
			hash:     0,
			expected: "page:0000000000000000",
		},
		{
			name:     "formats small hash",
			hash:     255,
			expected: "page:00000000000000ff",
		},
		{
			name:     "formats medium hash",
			hash:     0xdeadbeef,
			expected: "page:00000000deadbeef",
		},
		{
			name:     "formats max uint64 hash",
			hash:     0xffffffffffffffff,
			expected: "page:ffffffffffffffff",
		},
		{
			name:     "formats arbitrary hash",
			hash:     0x123456789abcdef0,
			expected: "page:123456789abcdef0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatCacheKey(tc.hash)

			assert.Equal(t, tc.expected, result)
			assert.Len(t, result, cacheKeyLen)
		})
	}
}

func TestFormatJITETag(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		hash     uint64
	}{
		{
			name:     "formats zero hash",
			hash:     0,
			expected: `"jit-0000000000000000"`,
		},
		{
			name:     "formats small hash",
			hash:     255,
			expected: `"jit-00000000000000ff"`,
		},
		{
			name:     "formats medium hash",
			hash:     0xdeadbeef,
			expected: `"jit-00000000deadbeef"`,
		},
		{
			name:     "formats max uint64 hash",
			hash:     0xffffffffffffffff,
			expected: `"jit-ffffffffffffffff"`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatJITETag(tc.hash)

			assert.Equal(t, tc.expected, result)
			assert.Len(t, result, jitETagLen)
		})
	}
}

func TestGetCacheControlHeader(t *testing.T) {
	testCases := []struct {
		name     string
		expected string
		maxAge   int
	}{
		{
			name:     "returns precomputed header for zero",
			maxAge:   0,
			expected: "public, max-age=0",
		},
		{
			name:     "returns precomputed header for 60 seconds",
			maxAge:   60,
			expected: "public, max-age=60",
		},
		{
			name:     "returns precomputed header for 1 hour",
			maxAge:   3600,
			expected: "public, max-age=3600",
		},
		{
			name:     "returns precomputed header for 1 day",
			maxAge:   86400,
			expected: "public, max-age=86400",
		},
		{
			name:     "returns precomputed header for 1 week",
			maxAge:   604800,
			expected: "public, max-age=604800",
		},
		{
			name:     "computes header for non-standard age",
			maxAge:   12345,
			expected: "public, max-age=12345",
		},
		{
			name:     "computes header for another non-standard age",
			maxAge:   999,
			expected: "public, max-age=999",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := getCacheControlHeader(tc.maxAge)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestDetermineCompression(t *testing.T) {
	testCases := []struct {
		name               string
		acceptEncoding     string
		expectedEncoding   string
		expectedCapability capabilities_dto.Capability
	}{
		{
			name:               "returns brotli for br encoding",
			acceptEncoding:     "br",
			expectedEncoding:   "br",
			expectedCapability: capabilities_dto.CapabilityCompressBrotli,
		},
		{
			name:               "returns brotli for br with quality",
			acceptEncoding:     "br;q=1.0, gzip;q=0.8",
			expectedEncoding:   "br",
			expectedCapability: capabilities_dto.CapabilityCompressBrotli,
		},
		{
			name:               "returns gzip when only gzip is present",
			acceptEncoding:     "gzip",
			expectedEncoding:   "gzip",
			expectedCapability: capabilities_dto.CapabilityCompressGzip,
		},
		{
			name:               "returns gzip for gzip with deflate",
			acceptEncoding:     "gzip, deflate",
			expectedEncoding:   "gzip",
			expectedCapability: capabilities_dto.CapabilityCompressGzip,
		},
		{
			name:               "returns empty for identity only",
			acceptEncoding:     "identity",
			expectedEncoding:   "",
			expectedCapability: "",
		},
		{
			name:               "returns empty for empty header",
			acceptEncoding:     "",
			expectedEncoding:   "",
			expectedCapability: "",
		},
		{
			name:               "prefers brotli over gzip when both present",
			acceptEncoding:     "gzip, br",
			expectedEncoding:   "br",
			expectedCapability: capabilities_dto.CapabilityCompressBrotli,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoding, capability := determineCompression(tc.acceptEncoding)

			assert.Equal(t, tc.expectedEncoding, encoding)
			assert.Equal(t, tc.expectedCapability, capability)
		})
	}
}

func TestSelectBestVariantForRequest(t *testing.T) {
	t.Run("selects brotli variant when client accepts brotli", func(t *testing.T) {
		var brotliTags registry_dto.Tags
		brotliTags.SetByName("contentEncoding", "br")

		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "gzip-variant", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("contentEncoding", "gzip")
					return tags
				}()},
				{VariantID: "brotli-variant", MetadataTags: brotliTags},
			},
		}

		result := selectBestVariantForRequest(artefact, "br, gzip")

		require.NotNil(t, result.variant)
		assert.Equal(t, "brotli", result.variantName)
	})

	t.Run("selects gzip variant when client accepts only gzip", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "gzip-variant", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("contentEncoding", "gzip")
					return tags
				}()},
				{VariantID: "brotli-variant", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("contentEncoding", "br")
					return tags
				}()},
			},
		}

		result := selectBestVariantForRequest(artefact, "gzip")

		require.NotNil(t, result.variant)
		assert.Equal(t, "gzip", result.variantName)
	})

	t.Run("falls back to minified-html when no compression accepted", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "minified-variant", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("type", "minified-html")
					return tags
				}()},
			},
		}

		result := selectBestVariantForRequest(artefact, "identity")

		require.NotNil(t, result.variant)
		assert.Equal(t, "minified-html", result.variantName)
	})

	t.Run("falls back to source variant when no other variants match", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "source"},
			},
		}

		result := selectBestVariantForRequest(artefact, "identity")

		require.NotNil(t, result.variant)
		assert.Equal(t, "source", result.variantName)
	})

	t.Run("returns empty result when no variants available", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{},
		}

		result := selectBestVariantForRequest(artefact, "br, gzip")

		assert.Nil(t, result.variant)
		assert.Empty(t, result.variantName)
	})
}

func TestFindVariantByTag(t *testing.T) {
	t.Run("finds variant with matching tag", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "variant-1", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("encoding", "gzip")
					return tags
				}()},
				{VariantID: "variant-2", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("encoding", "brotli")
					return tags
				}()},
			},
		}

		result := findVariantByTag(artefact, "encoding", "brotli")

		require.NotNil(t, result)
		assert.Equal(t, "variant-2", result.VariantID)
	})

	t.Run("returns nil when tag not found", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "variant-1", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("encoding", "gzip")
					return tags
				}()},
			},
		}

		result := findVariantByTag(artefact, "encoding", "brotli")

		assert.Nil(t, result)
	})

	t.Run("returns nil when key not present", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{
				{VariantID: "variant-1", MetadataTags: func() registry_dto.Tags {
					var tags registry_dto.Tags
					tags.SetByName("type", "cached")
					return tags
				}()},
			},
		}

		result := findVariantByTag(artefact, "encoding", "gzip")

		assert.Nil(t, result)
	})

	t.Run("returns nil for empty variants", func(t *testing.T) {
		artefact := &registry_dto.ArtefactMeta{
			ActualVariants: []registry_dto.Variant{},
		}

		result := findVariantByTag(artefact, "encoding", "gzip")

		assert.Nil(t, result)
	})
}

func TestHTMLBufferPool(t *testing.T) {
	t.Run("getHTMLBuffer returns reset buffer", func(t *testing.T) {
		buffer := getHTMLBuffer()

		require.NotNil(t, buffer)
		assert.Equal(t, 0, buffer.Len())
		assert.GreaterOrEqual(t, buffer.Cap(), htmlBufferPoolSize)

		releaseHTMLBuffer(buffer)
	})

	t.Run("releaseHTMLBuffer resets buffer before returning to pool", func(t *testing.T) {
		buffer := getHTMLBuffer()
		buffer.WriteString("test content")
		require.Equal(t, 12, buffer.Len())

		releaseHTMLBuffer(buffer)

		buf2 := getHTMLBuffer()
		assert.Equal(t, 0, buf2.Len())

		releaseHTMLBuffer(buf2)
	})

	t.Run("buffers can be reused multiple times", func(t *testing.T) {
		for range 10 {
			buffer := getHTMLBuffer()
			buffer.WriteString("iteration content")
			assert.Greater(t, buffer.Len(), 0)
			releaseHTMLBuffer(buffer)
		}
	})
}

func TestPipeResponseWriter(t *testing.T) {
	t.Run("newPipeResponseWriter initialises with correct defaults", func(t *testing.T) {
		pr, pw := io.Pipe()
		defer func() { _ = pr.Close() }()
		defer func() { _ = pw.Close() }()

		prw := newPipeResponseWriter(pw)

		assert.NotNil(t, prw)
		assert.NotNil(t, prw.header)
		assert.Equal(t, http.StatusOK, prw.statusCode)
	})

	t.Run("Header returns modifiable header map", func(t *testing.T) {
		pr, pw := io.Pipe()
		defer func() { _ = pr.Close() }()
		defer func() { _ = pw.Close() }()

		prw := newPipeResponseWriter(pw)

		prw.Header().Set("Content-Type", "text/html")

		assert.Equal(t, "text/html", prw.Header().Get("Content-Type"))
	})

	t.Run("WriteHeader records status code", func(t *testing.T) {
		pr, pw := io.Pipe()
		defer func() { _ = pr.Close() }()
		defer func() { _ = pw.Close() }()

		prw := newPipeResponseWriter(pw)

		prw.WriteHeader(http.StatusNotFound)

		assert.Equal(t, http.StatusNotFound, prw.statusCode)
	})

	t.Run("Write writes to underlying pipe", func(t *testing.T) {
		pr, pw := io.Pipe()
		defer func() { _ = pr.Close() }()

		prw := newPipeResponseWriter(pw)

		go func() {
			n, err := prw.Write([]byte("test data"))
			assert.NoError(t, err)
			assert.Equal(t, 9, n)
			_ = pw.Close()
		}()

		data, err := io.ReadAll(pr)
		assert.NoError(t, err)
		assert.Equal(t, "test data", string(data))
	})
}

func TestNewRateLimitMiddleware(t *testing.T) {
	t.Run("creates middleware with default clock", func(t *testing.T) {
		config := security_dto.RateLimitValues{
			Enabled: true,
			Global: security_dto.RateLimitTierValues{
				RequestsPerMinute: 100,
				BurstSize:         10,
			},
		}
		mockService := &security_domain.MockRateLimitService{}

		m := newRateLimitMiddleware(config, mockService)

		assert.NotNil(t, m)
		assert.NotNil(t, m.clock)
		assert.Equal(t, config, m.config)
	})

	t.Run("applies custom clock option", func(t *testing.T) {
		config := security_dto.RateLimitValues{}
		mockService := &security_domain.MockRateLimitService{}
		mockClock := clock.NewMockClock(time.Unix(1700000000, 0))

		m := newRateLimitMiddleware(config, mockService, withRateLimitClock(mockClock))

		assert.Equal(t, mockClock, m.clock)
	})
}

func TestIsExemptPath(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		exemptPaths []string
		expected    bool
	}{
		{
			name:        "returns true for exact match",
			exemptPaths: []string{"/health", "/metrics"},
			path:        "/health",
			expected:    true,
		},
		{
			name:        "returns true for prefix match",
			exemptPaths: []string{"/api/internal"},
			path:        "/api/internal/status",
			expected:    true,
		},
		{
			name:        "returns false for non-matching path",
			exemptPaths: []string{"/health", "/metrics"},
			path:        "/api/users",
			expected:    false,
		},
		{
			name:        "returns false for empty exempt list",
			exemptPaths: []string{},
			path:        "/health",
			expected:    false,
		},
		{
			name:        "returns false for partial non-prefix match",
			exemptPaths: []string{"/health"},
			path:        "/healthcheck",
			expected:    true,
		},
		{
			name:        "handles root path",
			exemptPaths: []string{"/"},
			path:        "/anything",
			expected:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &rateLimitMiddleware{
				config: security_dto.RateLimitValues{
					ExemptPaths: tc.exemptPaths,
				},
			}

			result := m.isExemptPath(tc.path)

			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSetRateLimitHeaders(t *testing.T) {
	t.Run("sets all headers for allowed request", func(t *testing.T) {
		m := &rateLimitMiddleware{}
		w := httptest.NewRecorder()
		result := ratelimiter_dto.Result{
			Allowed:   true,
			Limit:     100,
			Remaining: 99,
			ResetAt:   time.Unix(1700000000, 0),
		}

		m.setRateLimitHeaders(w, result)

		assert.Equal(t, "100", w.Header().Get(headerRateLimitLimit))
		assert.Equal(t, "99", w.Header().Get(headerRateLimitRemaining))
		assert.Equal(t, "1700000000", w.Header().Get(headerRateLimitReset))
		assert.Empty(t, w.Header().Get(headerRetryAfter))
	})

	t.Run("sets Retry-After header for denied request", func(t *testing.T) {
		m := &rateLimitMiddleware{}
		w := httptest.NewRecorder()
		result := ratelimiter_dto.Result{
			Allowed:    false,
			Limit:      100,
			Remaining:  0,
			ResetAt:    time.Unix(1700000060, 0),
			RetryAfter: 30 * time.Second,
		}

		m.setRateLimitHeaders(w, result)

		assert.Equal(t, "100", w.Header().Get(headerRateLimitLimit))
		assert.Equal(t, "0", w.Header().Get(headerRateLimitRemaining))
		assert.Equal(t, "1700000060", w.Header().Get(headerRateLimitReset))
		assert.Equal(t, "30", w.Header().Get(headerRetryAfter))
	})
}

func TestCheckRateLimit(t *testing.T) {
	t.Run("returns service result when check succeeds", func(t *testing.T) {
		var capturedKey string
		mockService := &security_domain.MockRateLimitService{
			CheckLimitFunc: func(_ context.Context, key string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
				capturedKey = key
				return ratelimiter_dto.Result{
					Allowed:   true,
					Limit:     100,
					Remaining: 50,
					ResetAt:   time.Unix(1700000060, 0),
				}, nil
			},
		}
		m := &rateLimitMiddleware{
			clock:   clock.RealClock(),
			service: mockService,
		}
		tier := security_dto.RateLimitTierValues{
			RequestsPerMinute: 100,
			BurstSize:         10,
		}

		result := m.checkRateLimit(context.Background(), "192.168.1.1", "global", tier)

		assert.True(t, result.Allowed)
		assert.Equal(t, 100, result.Limit)
		assert.Equal(t, 50, result.Remaining)
		assert.Equal(t, "ratelimit:global:192.168.1.1", capturedKey)
	})

	t.Run("returns denied result when service returns error (fail closed)", func(t *testing.T) {
		mockClock := clock.NewMockClock(time.Unix(1700000000, 0))
		mockService := &security_domain.MockRateLimitService{
			CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
				return ratelimiter_dto.Result{}, assert.AnError
			},
		}
		m := &rateLimitMiddleware{
			clock:   mockClock,
			service: mockService,
		}
		tier := security_dto.RateLimitTierValues{
			RequestsPerMinute: 100,
			BurstSize:         10,
		}

		result := m.checkRateLimit(context.Background(), "192.168.1.1", "action", tier)

		assert.False(t, result.Allowed)
		assert.Equal(t, 100, result.Limit)
		assert.Equal(t, 0, result.Remaining)
		assert.Equal(t, time.Minute, result.RetryAfter)
	})

	t.Run("constructs correct key with keySuffix", func(t *testing.T) {
		var capturedKey string
		mockService := &security_domain.MockRateLimitService{
			CheckLimitFunc: func(_ context.Context, key string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
				capturedKey = key
				return ratelimiter_dto.Result{Allowed: true}, nil
			},
		}
		m := &rateLimitMiddleware{
			clock:   clock.RealClock(),
			service: mockService,
		}
		tier := security_dto.RateLimitTierValues{RequestsPerMinute: 100}

		m.checkRateLimit(context.Background(), "10.0.0.1", "login", tier)

		assert.Equal(t, "ratelimit:login:10.0.0.1", capturedKey)
	})
}

func TestHandler_AllowedRequest_PassesThrough(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     100,
				Remaining: 99,
				ResetAt:   time.Unix(1700000060, 0),
			}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: true,
		Global: security_dto.RateLimitTierValues{
			RequestsPerMinute: 100,
			BurstSize:         10,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(recorder, request)

	assert.True(t, handlerCalled)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "100", recorder.Header().Get(headerRateLimitLimit))
	assert.Equal(t, "99", recorder.Header().Get(headerRateLimitRemaining))
}

func TestHandler_DeniedRequest_Returns429(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:    false,
				Limit:      100,
				Remaining:  0,
				ResetAt:    time.Unix(1700000060, 0),
				RetryAfter: 30 * time.Second,
			}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: true,
		Global: security_dto.RateLimitTierValues{
			RequestsPerMinute: 100,
			BurstSize:         10,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	handlerCalled := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
	})

	request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(recorder, request)

	assert.False(t, handlerCalled, "Handler should not be called when rate limited")
	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Rate limit exceeded")
	assert.Equal(t, "text/plain; charset=utf-8", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "30", recorder.Header().Get(headerRetryAfter))
}

func TestHandler_ExemptPath_BypassesRateLimit(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{Allowed: false}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:     true,
		ExemptPaths: []string{"/health", "/metrics"},
		Global: security_dto.RateLimitTierValues{
			RequestsPerMinute: 1,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(recorder, request)

	assert.True(t, handlerCalled, "Handler should be called for exempt paths")
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestHandler_HeadersDisabled_NoRateLimitHeaders(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     100,
				Remaining: 99,
				ResetAt:   time.Unix(1700000060, 0),
			}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: false,
		Global: security_dto.RateLimitTierValues{
			RequestsPerMinute: 100,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	m.Handler(next).ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Empty(t, recorder.Header().Get(headerRateLimitLimit))
	assert.Empty(t, recorder.Header().Get(headerRateLimitRemaining))
}

func TestActionHandler_AllowedRequest_ReturnsTrue(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     50,
				Remaining: 49,
				ResetAt:   time.Unix(1700000060, 0),
			}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: true,
		Actions: security_dto.RateLimitTierValues{
			RequestsPerMinute: 50,
			BurstSize:         5,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "192.168.1.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	allowed := m.ActionHandler(recorder, request, nil)

	assert.True(t, allowed)
	assert.Equal(t, "50", recorder.Header().Get(headerRateLimitLimit))
	assert.Equal(t, "49", recorder.Header().Get(headerRateLimitRemaining))
}

func TestActionHandler_DeniedRequest_ReturnsFalse(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:    false,
				Limit:      50,
				Remaining:  0,
				ResetAt:    time.Unix(1700000060, 0),
				RetryAfter: 15 * time.Second,
			}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: true,
		Actions: security_dto.RateLimitTierValues{
			RequestsPerMinute: 50,
			BurstSize:         5,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "192.168.1.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	allowed := m.ActionHandler(recorder, request, nil)

	assert.False(t, allowed)
	assert.Equal(t, http.StatusTooManyRequests, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Rate limit exceeded")
	assert.Equal(t, "15", recorder.Header().Get(headerRetryAfter))
}

func TestActionHandler_WithOverride_UsesCustomSettings(t *testing.T) {
	t.Parallel()

	var capturedKey string
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, key string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			capturedKey = key
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     200,
				Remaining: 199,
				ResetAt:   time.Unix(1700000060, 0),
			}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: true,
		Actions: security_dto.RateLimitTierValues{
			RequestsPerMinute: 50,
			BurstSize:         5,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	override := &security_dto.RateLimitOverride{
		RequestsPerMinute: 200,
		BurstSize:         20,
		KeySuffix:         "login",
	}

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "192.168.1.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	allowed := m.ActionHandler(recorder, request, override)

	assert.True(t, allowed)

	assert.Equal(t, "ratelimit:login:192.168.1.1", capturedKey)
}

func TestActionHandler_OverridePartialFields(t *testing.T) {
	t.Parallel()

	var capturedKey string
	var capturedLimit int
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, key string, limit int, _ time.Duration) (ratelimiter_dto.Result, error) {
			capturedKey = key
			capturedLimit = limit
			return ratelimiter_dto.Result{Allowed: true}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: false,
		Actions: security_dto.RateLimitTierValues{
			RequestsPerMinute: 50,
			BurstSize:         5,
		},
	}
	m := &rateLimitMiddleware{
		clock:   clock.RealClock(),
		service: mockService,
		config:  config,
	}

	override := &security_dto.RateLimitOverride{
		RequestsPerMinute: 100,
	}

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	allowed := m.ActionHandler(recorder, request, override)

	assert.True(t, allowed)

	assert.Equal(t, "ratelimit:action:10.0.0.1", capturedKey)

	assert.Equal(t, 100, capturedLimit)
}

func TestActionHandler_HeadersDisabled_NoHeaders(t *testing.T) {
	t.Parallel()

	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{
				Allowed:   true,
				Limit:     50,
				Remaining: 49,
				ResetAt:   time.Unix(1700000060, 0),
			}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: false,
		Actions: security_dto.RateLimitTierValues{
			RequestsPerMinute: 50,
		},
	}
	m := newRateLimitMiddleware(config, mockService)

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	allowed := m.ActionHandler(recorder, request, nil)

	assert.True(t, allowed)
	assert.Empty(t, recorder.Header().Get(headerRateLimitLimit))
	assert.Empty(t, recorder.Header().Get(headerRateLimitRemaining))
}

func TestWithRateLimitClock_SetsCustomClock(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(fixedTime)
	config := security_dto.RateLimitValues{}

	m := newRateLimitMiddleware(config, nil, withRateLimitClock(mockClock))

	assert.Equal(t, mockClock, m.clock)
}

func TestWithRateLimitClock_OverridesDefaultClock(t *testing.T) {
	t.Parallel()

	config := security_dto.RateLimitValues{}

	m1 := newRateLimitMiddleware(config, nil)
	assert.NotNil(t, m1.clock)

	mockClock := clock.NewMockClock(time.Unix(0, 0))
	m2 := newRateLimitMiddleware(config, nil, withRateLimitClock(mockClock))
	assert.Equal(t, mockClock, m2.clock)
}

func TestExtractOTelContextFromRequest_AlreadyExtracted_ReturnsSameContext(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("traceparent", "00-abc123-def456-01")

	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		OtelExtracted: true,
	})
	request = request.WithContext(ctx)

	resultCtx, changed := extractOTelContextFromRequest(request)

	assert.False(t, changed, "Should not change context when already extracted")
	assert.Equal(t, ctx, resultCtx, "Should return the same context")
}

func TestExtractOTelContextFromRequest_NotExtracted_ExtractsAndMarks(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{})
	request = request.WithContext(ctx)

	resultCtx, changed := extractOTelContextFromRequest(request)

	assert.True(t, changed, "Should indicate context was changed")
	assert.NotNil(t, resultCtx)

	pctx := daemon_dto.PikoRequestCtxFromContext(resultCtx)
	assert.NotNil(t, pctx)
	assert.True(t, pctx.OtelExtracted)
}

func TestExtractOTelContextFromRequest_NoHeaders_StillMarksExtracted(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{})
	request = request.WithContext(ctx)

	resultCtx, changed := extractOTelContextFromRequest(request)

	assert.True(t, changed, "Should indicate context was changed even without headers")
	pctx := daemon_dto.PikoRequestCtxFromContext(resultCtx)
	assert.NotNil(t, pctx)
	assert.True(t, pctx.OtelExtracted)
}

func TestPageDefPool_GetReturnsBPageDefinition(t *testing.T) {
	t.Parallel()

	definition := pageDefPool.Get()
	require.NotNil(t, definition)
	_, ok := definition.(*templater_dto.PageDefinition)
	assert.True(t, ok, "Pool should return *PageDefinition")
	pageDefPool.Put(definition)
}

func TestPageDefPool_ReturnsToPool(t *testing.T) {
	t.Parallel()

	definition, ok := pageDefPool.Get().(*templater_dto.PageDefinition)
	if !ok {
		t.Fatal("expected *templater_dto.PageDefinition")
	}
	require.NotNil(t, definition)

	pageDefPool.Put(definition)

	def2 := pageDefPool.Get()
	assert.NotNil(t, def2)
	pageDefPool.Put(def2)
}

func TestCompressedResponseWriter_DelegatesWriteToCompressor(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	buffer := &bytes.Buffer{}

	crw := &compressedResponseWriter{
		ResponseWriter: recorder,
		compressor:     nopWriteCloser{buffer},
	}

	n, err := crw.Write([]byte("hello world"))

	require.NoError(t, err)
	assert.Equal(t, 11, n)
	assert.Equal(t, "hello world", buffer.String())

	assert.Empty(t, recorder.Body.String())
}

func TestCompressedResponseWriter_HeaderDelegatesToResponseWriter(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	buffer := &bytes.Buffer{}

	crw := &compressedResponseWriter{
		ResponseWriter: recorder,
		compressor:     nopWriteCloser{buffer},
	}

	crw.Header().Set("X-Custom", "test")
	assert.Equal(t, "test", recorder.Header().Get("X-Custom"))
}

func TestGetHTMLBuffer_AlwaysReturnsResetBuffer(t *testing.T) {
	t.Parallel()

	for range 5 {
		buffer := getHTMLBuffer()
		assert.Equal(t, 0, buffer.Len(), "Buffer should be reset")
		buffer.WriteString("some content")
		releaseHTMLBuffer(buffer)
	}
}

func TestGetHTMLBuffer_HasMinimumCapacity(t *testing.T) {
	t.Parallel()

	buffer := getHTMLBuffer()
	defer releaseHTMLBuffer(buffer)

	assert.GreaterOrEqual(t, buffer.Cap(), htmlBufferPoolSize)
}

func TestRateLimitHeaderConstants_Values(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "X-Ratelimit-Limit", headerRateLimitLimit)
	assert.Equal(t, "X-Ratelimit-Remaining", headerRateLimitRemaining)
	assert.Equal(t, "X-Ratelimit-Reset", headerRateLimitReset)
	assert.Equal(t, "Retry-After", headerRetryAfter)
}

func TestCacheConstants_Values(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 8192, htmlBufferPoolSize)
	assert.Equal(t, 16, mapCarrierPoolSize)
	assert.Equal(t, 21, cacheKeyLen)
	assert.Equal(t, 22, jitETagLen)
	assert.Equal(t, 16, hexDigitCount)
	assert.Equal(t, 5, prefixLen)
	assert.Equal(t, uint64(0xf), uint64(hexNibbleMask))
	assert.Equal(t, 4, hexNibbleShift)
	assert.Equal(t, 8, pipeResponseWriterHeaderSize)
}

func TestJitCacheControl_Value(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "public, max-age=0, must-revalidate", jitCacheControl)
}

func TestErrorSentinels_Values(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, errEmptyBody)
	assert.NotNil(t, errHandlerNonSuccess)
	assert.Equal(t, "handler returned empty body on 200 OK", errEmptyBody.Error())
	assert.Equal(t, "upstream handler returned non-200 status code", errHandlerNonSuccess.Error())
}

func TestDefaultConstants_Values(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 10, defaultCacheWriteConcurrency)
	assert.Equal(t, 4, defaultStreamCompressionLevel)
	assert.Equal(t, "Content-Length", headerContentLength)
	assert.Equal(t, "If-None-Match", headerIfNoneMatch)
	assert.Equal(t, "Link", headerLink)
	assert.Equal(t, "public, max-age=0, must-revalidate", jitCacheControl)
	assert.Equal(t, "text/css; charset=utf-8", contentTypeCSS)
	assert.Equal(t, "application/xml; charset=utf-8", contentTypeXML)
	assert.Equal(t, "Internal Server Error", errMessageInternalServer)
	assert.Equal(t, "GET", methodGET)
	assert.Equal(t, "POST", methodPOST)
	assert.Equal(t, "source", variantSource)
	assert.Equal(t, "hls_", hlsVariantPrefix)
	assert.Equal(t, 10.0, hlsDefaultSegmentDuration)
}

func TestLogFieldConstants_Additional(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "quality", logFieldQuality)
	assert.Equal(t, "chunkID", logFieldChunkID)
	assert.Equal(t, "Variant not found", msgVariantNotFound)
	assert.Equal(t, "bytesWritten", logFieldBytesWritten)
	assert.Equal(t, "selectedVariant", logFieldSelectedVar)
	assert.Equal(t, "etagMatch", logFieldETagMatch)
	assert.Equal(t, "contentEncoding", logFieldContentEnc)
	assert.Equal(t, "handler", logFieldHandler)
	assert.Equal(t, "router", logFieldRouter)
	assert.Equal(t, "HTTPRouterBuilder", logFieldHTTPBuilder)
	assert.Equal(t, "driven_middleware_cache.go", logFieldCacheFile)
	assert.Equal(t, "driven_http.go", logFieldHTTPFile)
	assert.Equal(t, "driven_http_router.go", logFieldHTTPRouterFile)
	assert.Equal(t, "profileName", logFieldProfileName)
}

func TestPageCacheProfiles_HasBrotliAndGzip(t *testing.T) {
	t.Parallel()

	require.Len(t, pageCacheProfiles, 2)

	brotliProfile := pageCacheProfiles[0]
	assert.Equal(t, "brotli_variant", brotliProfile.Name)
	assert.NotEmpty(t, brotliProfile.Profile.CapabilityName)
	assert.NotEmpty(t, brotliProfile.Profile.ResultingTags)

	gzipProfile := pageCacheProfiles[1]
	assert.Equal(t, "gzip_variant", gzipProfile.Name)
	assert.NotEmpty(t, gzipProfile.Profile.CapabilityName)
	assert.NotEmpty(t, gzipProfile.Profile.ResultingTags)
}

func TestPageCacheProfiles_BrotliTagsSet(t *testing.T) {
	t.Parallel()

	brotliProfile := pageCacheProfiles[0]
	tags := brotliProfile.Profile.ResultingTags

	contentEnc, ok := tags.GetByName(logFieldContentEnc)
	assert.True(t, ok)
	assert.Equal(t, encodingBrotli, contentEnc)

	typeVal, ok := tags.GetByName("type")
	assert.True(t, ok)
	assert.Equal(t, "cached-page", typeVal)

	storageBackend, ok := tags.GetByName("storageBackendId")
	assert.True(t, ok)
	assert.Equal(t, "local_disk_cache", storageBackend)

	mimeType, ok := tags.GetByName("mimeType")
	assert.True(t, ok)
	assert.Equal(t, contentTypeHTML, mimeType)
}

func TestPageCacheProfiles_GzipTagsSet(t *testing.T) {
	t.Parallel()

	gzipProfile := pageCacheProfiles[1]
	tags := gzipProfile.Profile.ResultingTags

	contentEnc, ok := tags.GetByName(logFieldContentEnc)
	assert.True(t, ok)
	assert.Equal(t, encodingGzip, contentEnc)

	typeVal, ok := tags.GetByName("type")
	assert.True(t, ok)
	assert.Equal(t, "cached-page", typeVal)
}

func TestCacheControlHeaders_PreComputedValues(t *testing.T) {
	t.Parallel()

	expectedAges := []int{0, 60, 300, 600, 900, 1800, 3600, 7200, 14400, 28800, 43200, 86400, 604800}
	assert.Len(t, cacheControlHeaders, len(expectedAges))

	for _, age := range expectedAges {
		_, ok := cacheControlHeaders[age]
		assert.True(t, ok, "Should have pre-computed header for age %d", age)
	}
}

func TestCacheControlHeaders_NonStandardAge_NotPrecomputed(t *testing.T) {
	t.Parallel()

	_, ok := cacheControlHeaders[12345]
	assert.False(t, ok, "Non-standard ages should not be pre-computed")
}

func TestHexTable_Has16Characters(t *testing.T) {
	t.Parallel()

	assert.Len(t, hexTable, 16)
	assert.Equal(t, []byte("0123456789abcdef"), hexTable)
}

func TestNullSeparator_Value(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []byte{0}, nullSeparator)
	assert.Len(t, nullSeparator, 1)
}

func TestNewCacheMiddleware_SetsDefaultStreamCompressionLevel(t *testing.T) {
	t.Parallel()

	config := CacheMiddlewareConfig{
		StreamCompressionLevel: 0,
	}
	manifest := &templater_domain.MockManifestStoreView{}
	registry := &registry_domain.MockRegistryService{}

	mw := NewCacheMiddleware(config, manifest, registry, nil, "")

	assert.Equal(t, defaultStreamCompressionLevel, mw.config.StreamCompressionLevel)
}

func TestNewCacheMiddleware_CustomConcurrency_Preserved(t *testing.T) {
	t.Parallel()

	config := CacheMiddlewareConfig{
		CacheWriteConcurrency: 20,
	}
	manifest := &templater_domain.MockManifestStoreView{}
	registry := &registry_domain.MockRegistryService{}

	mw := NewCacheMiddleware(config, manifest, registry, nil, "")

	assert.Equal(t, 20, mw.config.CacheWriteConcurrency)
}

func TestNewCacheMiddleware_NegativeConcurrency_UsesDefault(t *testing.T) {
	t.Parallel()

	config := CacheMiddlewareConfig{
		CacheWriteConcurrency: -1,
	}
	manifest := &templater_domain.MockManifestStoreView{}
	registry := &registry_domain.MockRegistryService{}

	mw := NewCacheMiddleware(config, manifest, registry, nil, "")

	assert.Equal(t, defaultCacheWriteConcurrency, mw.config.CacheWriteConcurrency)
}

func TestNewCacheMiddleware_EmptyRouteMap(t *testing.T) {
	t.Parallel()

	config := CacheMiddlewareConfig{}
	manifest := &templater_domain.MockManifestStoreView{}
	registry := &registry_domain.MockRegistryService{}

	mw := NewCacheMiddleware(config, manifest, registry, nil, "")

	assert.NotNil(t, mw.routeMap)
	assert.Empty(t, mw.routeMap)
}

func TestCompressedResponseWriterPool_GetAndPut(t *testing.T) {
	t.Parallel()

	crw := compressedResponseWriterPool.Get()
	assert.NotNil(t, crw)
	_, ok := crw.(*compressedResponseWriter)
	assert.True(t, ok)
	compressedResponseWriterPool.Put(crw)
}

func TestBrotliWriterPool_GetAndReturn(t *testing.T) {
	t.Parallel()

	w := brotliWriterPool.Get()
	assert.NotNil(t, w)
	brotliWriterPool.Put(w)
}

func TestGzipWriterPool_GetAndReturn(t *testing.T) {
	t.Parallel()

	w := gzipWriterPool.Get()
	assert.NotNil(t, w)
	gzipWriterPool.Put(w)
}

func TestActionHandler_OverrideRequestsPerMinute_PassedToService(t *testing.T) {
	t.Parallel()

	var capturedLimit int
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, limit int, _ time.Duration) (ratelimiter_dto.Result, error) {
			capturedLimit = limit
			return ratelimiter_dto.Result{Allowed: true}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: false,
		Actions: security_dto.RateLimitTierValues{
			RequestsPerMinute: 50,
		},
	}
	m := &rateLimitMiddleware{
		clock:   clock.RealClock(),
		service: mockService,
		config:  config,
	}

	override := &security_dto.RateLimitOverride{
		RequestsPerMinute: 200,
	}

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	allowed := m.ActionHandler(recorder, request, override)

	assert.True(t, allowed)
	assert.Equal(t, 200, capturedLimit)
}

func TestActionHandler_NilOverride_UsesDefaultActionConfig(t *testing.T) {
	t.Parallel()

	var capturedKey string
	var capturedLimit int
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, key string, limit int, _ time.Duration) (ratelimiter_dto.Result, error) {
			capturedKey = key
			capturedLimit = limit
			return ratelimiter_dto.Result{Allowed: true}, nil
		},
	}
	config := security_dto.RateLimitValues{
		Enabled:        true,
		HeadersEnabled: false,
		Actions: security_dto.RateLimitTierValues{
			RequestsPerMinute: 75,
		},
	}
	m := &rateLimitMiddleware{
		clock:   clock.RealClock(),
		service: mockService,
		config:  config,
	}

	request := httptest.NewRequest(http.MethodPost, "/action", nil)
	ctx := daemon_dto.WithPikoRequestCtx(request.Context(), &daemon_dto.PikoRequestCtx{
		ClientIP: "10.0.0.1",
	})
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	allowed := m.ActionHandler(recorder, request, nil)

	assert.True(t, allowed)
	assert.Equal(t, 75, capturedLimit)
	assert.Equal(t, "ratelimit:action:10.0.0.1", capturedKey)
}

func TestSetupBrotliCompressor_SetsContentEncodingHeader(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	compressor, ok := setupBrotliCompressor(context.Background(), recorder)

	assert.True(t, ok)
	assert.NotNil(t, compressor)
	assert.Equal(t, "br", recorder.Header().Get("Content-Encoding"))

	if compressor != nil {
		_ = compressor.Close()
	}
}

func TestSetupGzipCompressor_SetsContentEncodingHeader(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	compressor, ok := setupGzipCompressor(context.Background(), recorder)

	assert.True(t, ok)
	assert.NotNil(t, compressor)
	assert.Equal(t, "gzip", recorder.Header().Get("Content-Encoding"))

	if compressor != nil {
		_ = compressor.Close()
	}
}

func TestCacheMiddlewareConfig_ZeroValues(t *testing.T) {
	t.Parallel()

	config := CacheMiddlewareConfig{}

	assert.Equal(t, 0, config.StreamCompressionLevel)
	assert.Equal(t, 0, config.CacheWriteConcurrency)
}

func TestVariantSelectionResult_EmptyResult(t *testing.T) {
	t.Parallel()

	result := variantSelectionResult{}

	assert.Nil(t, result.variant)
	assert.Empty(t, result.variantName)
}

func TestCheckRateLimit_FailClosed_UsesClockForResetAt(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	mockClock := clock.NewMockClock(fixedTime)
	mockService := &security_domain.MockRateLimitService{
		CheckLimitFunc: func(_ context.Context, _ string, _ int, _ time.Duration) (ratelimiter_dto.Result, error) {
			return ratelimiter_dto.Result{}, assert.AnError
		},
	}
	m := &rateLimitMiddleware{
		clock:   mockClock,
		service: mockService,
	}
	tier := security_dto.RateLimitTierValues{
		RequestsPerMinute: 50,
	}

	result := m.checkRateLimit(context.Background(), "1.2.3.4", "test", tier)

	assert.False(t, result.Allowed)
	assert.Equal(t, 50, result.Limit)
	assert.Equal(t, 0, result.Remaining)
	assert.Equal(t, fixedTime.Add(time.Minute), result.ResetAt)
	assert.Equal(t, time.Minute, result.RetryAfter)
}
