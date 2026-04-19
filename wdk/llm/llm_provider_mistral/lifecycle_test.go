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

package llm_provider_mistral

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/safeerror"
)

func TestProvider_Close_Idempotent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	require.NoError(t, p.Close(context.Background()))
	require.NoError(t, p.Close(context.Background()))
	require.NoError(t, p.Close(context.Background()))
}

func TestProvider_Close_DrainsStreamGoroutines(t *testing.T) {
	streamReady := make(chan struct{})
	allowFinish := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		_, _ = fmt.Fprint(w, "data: {\"id\":\"cmpl-1\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hi\"}}]}\n\n")
		flusher.Flush()
		close(streamReady)

		<-allowFinish
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	streamChannel, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.NoError(t, err)

	<-streamReady

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- p.Close(context.Background())
	}()

	close(allowFinish)

	for range streamChannel {
	}

	select {
	case err := <-closeDone:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return after stream completed")
	}
}

func TestProvider_OversizeBody_ReturnsTruncationError(t *testing.T) {
	bigBody := strings.Repeat("x", maxLLMResponseBytes+1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, bigBody)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	_, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.Error(t, err)
}

func TestProvider_OversizeOKBody_ReturnsTruncationError(t *testing.T) {
	bigBody := strings.Repeat("x", maxLLMResponseBytes+1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, bigBody)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	_, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded")
}

func TestProvider_ClientError_WrappedInSafeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"invalid api key"}`)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	_, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.Error(t, err)

	var safeErr safeerror.Error
	require.True(t, errors.As(err, &safeErr), "expected error to satisfy safeerror.Error")
	assert.Equal(t, "mistral request rejected", safeErr.SafeMessage())
}

func TestProvider_BodyDrainedOnError(t *testing.T) {
	bytesRead := atomic.Int64{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.(http.Flusher).Flush()

		body := strings.Repeat("a", 1024)
		n, _ := io.WriteString(w, body)
		bytesRead.Add(int64(n))
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	_, err := p.Complete(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.Error(t, err)
	assert.Greater(t, bytesRead.Load(), int64(0), "server should have written body bytes")
}

func TestReadBoundedBody_HitsCap(t *testing.T) {
	body := strings.NewReader(strings.Repeat("y", maxLLMResponseBytes+10))
	data, err := readBoundedBody(body)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errResponseTruncated))
	assert.Equal(t, maxLLMResponseBytes, len(data))
}

func TestReadBoundedBody_AtCap(t *testing.T) {
	body := strings.NewReader(strings.Repeat("y", maxLLMResponseBytes))
	data, err := readBoundedBody(body)
	require.NoError(t, err)
	assert.Equal(t, maxLLMResponseBytes, len(data))
}

func TestReadBoundedBody_BelowCap(t *testing.T) {
	body := strings.NewReader("hello world")
	data, err := readBoundedBody(body)
	require.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestMistralProvider_ClientHasTimeout(t *testing.T) {
	p := newTestProvider(t)

	require.NotNil(t, p.client, "underlying HTTP client should be set")
	assert.Greater(t, p.client.Timeout, time.Duration(0),
		"mistral HTTP client should carry a top-level timeout")
	assert.Equal(t, httpClientTimeout, p.client.Timeout,
		"timeout should match the configured constant")
}

func TestMistralProvider_PopulatesRetryAfterFrom429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"error":"rate limited"}`)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	_, err := p.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"hello"},
	})
	require.Error(t, err)

	var providerErr *llm_domain.ProviderError
	require.True(t, errors.As(err, &providerErr), "expected provider error in chain")
	assert.Equal(t, http.StatusTooManyRequests, providerErr.StatusCode)
	assert.Equal(t, 30*time.Second, providerErr.RetryAfter,
		"Retry-After header should populate ProviderError.RetryAfter")
	assert.True(t, providerErr.IsRetryable(), "429 should be classified as retryable")
}

func TestMistralProvider_PopulatesRetryAfterFromStreamRateLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "45")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"error":"overloaded"}`)
	}))
	defer server.Close()

	p := newTestProviderWithServer(t, server)

	_, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.Error(t, err)

	var providerErr *llm_domain.ProviderError
	require.True(t, errors.As(err, &providerErr), "expected provider error in chain")
	assert.Equal(t, http.StatusServiceUnavailable, providerErr.StatusCode)
	assert.Equal(t, 45*time.Second, providerErr.RetryAfter)
	assert.True(t, providerErr.IsRetryable(), "503 should be classified as retryable")
}
