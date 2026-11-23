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

package captcha_provider_recaptcha_v3

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/captcha/captcha_domain"
	"piko.sh/piko/internal/captcha/captcha_dto"
)

type stubTransport struct {
	body string
}

func (s *stubTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.Body != nil {
		_, _ = io.Copy(io.Discard, request.Body)
		_ = request.Body.Close()
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Request:    request,
	}, nil
}

func newTestProvider(t *testing.T, serverURL string, options ...Option) *provider {
	t.Helper()

	testProvider, err := newProvider(Config{
		SiteKey:   "test-site-key",
		SecretKey: "test-secret-key",
	}, serverURL, options...)
	require.NoError(t, err)

	return testProvider
}

func TestNewProvider_Valid(t *testing.T) {
	t.Parallel()

	provider, err := NewProvider(Config{
		SiteKey:   "site-key",
		SecretKey: "secret-key",
	})

	require.NoError(t, err)
	assert.NotNil(t, provider)
}

func TestNewProvider_EmptySiteKey(t *testing.T) {
	t.Parallel()

	_, err := NewProvider(Config{
		SiteKey:   "",
		SecretKey: "secret-key",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSiteKeyEmpty)
}

func TestNewProvider_EmptySecretKey(t *testing.T) {
	t.Parallel()

	_, err := NewProvider(Config{
		SiteKey:   "site-key",
		SecretKey: "",
	})

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSecretKeyEmpty)
}

func TestProvider_Type(t *testing.T) {
	t.Parallel()

	testProvider := newTestProvider(t, "http://localhost")

	assert.Equal(t, captcha_dto.ProviderTypeRecaptchaV3, testProvider.Type())
}

func TestProvider_SiteKey(t *testing.T) {
	t.Parallel()

	testProvider := newTestProvider(t, "http://localhost")

	assert.Equal(t, "test-site-key", testProvider.SiteKey())
}

func TestProvider_ScriptURL(t *testing.T) {
	t.Parallel()

	testProvider := newTestProvider(t, "http://localhost")

	assert.Equal(t, "https://www.google.com/recaptcha/api.js?render=test-site-key", testProvider.ScriptURL())
}

func TestProvider_Verify_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":true,"hostname":"example.com","challenge_ts":"2026-01-01T00:00:00Z","score":0.9,"action":"submit"}`))
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{
		Token: "valid-token",
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "example.com", response.Hostname)
	assert.False(t, response.Timestamp.IsZero())
	require.NotNil(t, response.Score)
	assert.InDelta(t, 0.9, *response.Score, 0.001)
	assert.Equal(t, "submit", response.Action)
}

func TestProvider_Verify_Failure(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":false,"error-codes":["invalid-input-response"]}`))
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{
		Token: "invalid-token",
	})

	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "invalid-input-response")
}

func TestProvider_Verify_EmptyToken(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatal("server should not be called when token is empty")
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{})

	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "missing-input-response")
}

func TestProvider_Verify_WithScore(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":true,"score":0.9,"action":"submit"}`))
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{
		Token: "valid-token",
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	require.NotNil(t, response.Score)
	assert.InDelta(t, 0.9, *response.Score, 0.001)
}

func TestProvider_Verify_LowScore(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":true,"score":0.1}`))
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{
		Token: "low-score-token",
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	require.NotNil(t, response.Score)
	assert.InDelta(t, 0.1, *response.Score, 0.001)
}

func TestProvider_Verify_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	_, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{
		Token: "some-token",
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, captcha_dto.ErrProviderUnavailable))
}

func TestProvider_Verify_MalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{invalid`))
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	_, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{
		Token: "some-token",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing recaptcha response")
}

func TestProvider_HealthCheck(t *testing.T) {
	t.Parallel()

	testProvider := newTestProvider(t, "http://localhost")

	err := testProvider.HealthCheck(context.Background())

	assert.NoError(t, err)
}

func TestProvider_VerifyNilRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		t.Fatal("server should not be called when request is nil")
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(t.Context(), nil)

	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "missing-input-response")
}

func TestProvider_MalformedTimestamp(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":true,"score":0.9,"challenge_ts":"garbage"}`))
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: "some-token",
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.True(t, response.Timestamp.IsZero())
}

func TestProvider_Verify_ResponseSizeLimit(t *testing.T) {
	t.Parallel()

	const limit = 256
	verdictOfSize := func(size int) string {
		const prefix, suffix = `{"success":true,"score":0.9,"hostname":"`, `"}`
		return prefix + strings.Repeat("a", size-len(prefix)-len(suffix)) + suffix
	}

	testCases := []struct {
		name        string
		body        string
		wantSuccess bool
		wantErr     bool
	}{
		{name: "a response of exactly the limit is accepted", body: verdictOfSize(limit), wantSuccess: true},
		{name: "a response over the limit is refused", body: verdictOfSize(limit + 1), wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				_, _ = writer.Write([]byte(tc.body))
			}))
			defer server.Close()

			testProvider := newTestProvider(t, server.URL, WithMaxResponseBytes(limit), WithVerifyTimeout(5*time.Second))
			response, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{Token: "token"})

			if tc.wantErr {
				require.ErrorIs(t, err, captcha_dto.ErrProviderUnavailable)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantSuccess, response.Success)
		})
	}
}

func TestNewProvider_AcceptsOptions(t *testing.T) {
	t.Parallel()

	created, err := NewProvider(Config{SiteKey: "site", SecretKey: "secret"},
		WithVerifyTimeout(time.Second), WithMaxResponseBytes(1024))

	require.NoError(t, err)
	assert.NotNil(t, created)
}

func TestProvider_Verify_ScoreHandling(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		body         string
		testKeyScore *float64
		wantScore    float64
		wantErr      error
	}{
		{name: "zero score stays a bot", body: `{"success":true,"score":0}`, testKeyScore: nil, wantScore: 0, wantErr: nil},
		{name: "missing score is rejected", body: `{"success":true}`, testKeyScore: nil, wantScore: 0, wantErr: ErrScoreMissing},
		{name: "test key score fills a missing score", body: `{"success":true,"hostname":"testkey.google.com"}`, testKeyScore: new(0.7), wantScore: 0.7, wantErr: nil},
		{name: "test key score ignores other hostnames", body: `{"success":true,"hostname":"example.com"}`, testKeyScore: new(0.7), wantScore: 0, wantErr: ErrScoreMissing},
		{name: "test key score never overrides a real score", body: `{"success":true,"score":0}`, testKeyScore: new(0.7), wantScore: 0, wantErr: nil},
		{name: "declined response without a score", body: `{"success":false}`, testKeyScore: nil, wantScore: 0, wantErr: nil},
		{name: "out of range score is clamped", body: `{"success":true,"score":1.5}`, testKeyScore: nil, wantScore: 1, wantErr: nil},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(http.StatusOK)
				_, _ = writer.Write([]byte(testCase.body))
			}))
			t.Cleanup(server.Close)

			testProvider, err := newProvider(Config{SiteKey: "site", SecretKey: "secret", TestKeyScore: testCase.testKeyScore}, server.URL)
			require.NoError(t, err)

			response, err := testProvider.Verify(t.Context(), &captcha_dto.VerifyRequest{Token: "token"})
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
				assert.Nil(t, response)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, response.Score)
			assert.InDelta(t, testCase.wantScore, *response.Score, 0.0001)
		})
	}
}

func TestNewProvider_RejectsOutOfRangeTestKeyScore(t *testing.T) {
	t.Parallel()

	for _, score := range []float64{-0.1, 1.1, math.NaN()} {
		_, err := NewProvider(Config{SiteKey: "site", SecretKey: "secret", TestKeyScore: new(score)})
		require.ErrorIs(t, err, ErrTestKeyScoreOutOfRange)
	}
}

func TestCaptchaService_VerifyWithProvider_RecaptchaTestKeys(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		testKeyScore *float64
		name         string
		body         string
		wantErrs     []error
		wantScore    float64
	}{
		{
			name:         "test keys pass with the configured score and requested action",
			body:         `{"success":true,"hostname":"testkey.google.com"}`,
			testKeyScore: new(0.9),
			wantScore:    0.9,
			wantErrs:     nil,
		},
		{
			name:         "test keys without a test key score fail as misconfigured",
			body:         `{"success":true,"hostname":"testkey.google.com"}`,
			testKeyScore: nil,
			wantErrs:     []error{ErrScoreMissing, captcha_dto.ErrProviderMisconfigured},
		},
		{
			name:         "a v2 secret is not rescued by the test key score",
			body:         `{"success":true,"hostname":"example.com"}`,
			testKeyScore: new(0.9),
			wantErrs:     []error{ErrScoreMissing, captcha_dto.ErrProviderMisconfigured},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			recaptcha, err := NewProvider(
				Config{TestKeyScore: testCase.testKeyScore, SiteKey: "site", SecretKey: "secret"},
				WithTransport(&stubTransport{body: testCase.body}),
			)
			require.NoError(t, err)

			service, err := captcha_domain.NewCaptchaService(captcha_dto.DefaultServiceConfig())
			require.NoError(t, err)
			require.NoError(t, service.RegisterProvider(t.Context(), "recaptcha", recaptcha))

			response, err := service.VerifyWithProvider(t.Context(), "recaptcha", "token", "192.0.2.1", "contact", 0.5)
			if testCase.wantErrs != nil {
				for _, wantErr := range testCase.wantErrs {
					require.ErrorIs(t, err, wantErr)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, response.Score)
			assert.InDelta(t, testCase.wantScore, *response.Score, 0.0001)
			assert.Equal(t, "contact", response.Action)
		})
	}
}
