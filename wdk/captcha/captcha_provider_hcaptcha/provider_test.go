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

package captcha_provider_hcaptcha

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/captcha/captcha_dto"
)

type urlRewriteTransport struct {
	base      http.RoundTripper
	targetURL string
}

func (transport *urlRewriteTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request.URL, _ = url.Parse(transport.targetURL)
	return transport.base.RoundTrip(request)
}

func newTestProvider(t *testing.T, serverURL string) *provider {
	t.Helper()
	return &provider{
		httpClient: &http.Client{
			Transport: &urlRewriteTransport{
				base:      http.DefaultTransport,
				targetURL: serverURL,
			},
		},
		config: Config{
			SiteKey:   "test-site-key",
			SecretKey: "test-secret-key",
		},
	}
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

	assert.Equal(t, captcha_dto.ProviderTypeHCaptcha, testProvider.Type())
}

func TestProvider_SiteKey(t *testing.T) {
	t.Parallel()

	testProvider := newTestProvider(t, "http://localhost")

	assert.Equal(t, "test-site-key", testProvider.SiteKey())
}

func TestProvider_ScriptURL(t *testing.T) {
	t.Parallel()

	testProvider := newTestProvider(t, "http://localhost")

	assert.Equal(t, "https://js.hcaptcha.com/1/api.js", testProvider.ScriptURL())
}

func TestProvider_Verify_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":true,"hostname":"example.com","challenge_ts":"2026-01-01T00:00:00Z"}`))
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

	response, err := testProvider.Verify(context.Background(), &captcha_dto.VerifyRequest{
		Token: "",
	})

	require.NoError(t, err)
	assert.False(t, response.Success)
	assert.Contains(t, response.ErrorCodes, "missing-input-response")
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
	assert.Contains(t, err.Error(), "parsing hcaptcha response")
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

func TestProvider_EnterpriseScoreNormalisation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":true,"score":0.3}`))
	}))
	defer server.Close()

	testProvider := newTestProvider(t, server.URL)

	response, err := testProvider.Verify(t.Context(), &captcha_dto.VerifyRequest{
		Token: "enterprise-token",
	})

	require.NoError(t, err)
	assert.True(t, response.Success)
	require.NotNil(t, response.Score)
	assert.InDelta(t, 0.7, *response.Score, 0.001)
}

func TestProvider_MalformedTimestamp(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"success":true,"challenge_ts":"not-a-date"}`))
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
