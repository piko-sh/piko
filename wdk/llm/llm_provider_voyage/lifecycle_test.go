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

package llm_provider_voyage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/safeerror"
)

func TestVoyageProvider_Close_Idempotent(t *testing.T) {
	provider, err := NewVoyageProvider(Config{APIKey: "test-key"})
	require.NoError(t, err)

	require.NoError(t, provider.Close(context.Background()))
	require.NoError(t, provider.Close(context.Background()))
	require.NoError(t, provider.Close(context.Background()))
}

func TestVoyageProvider_OversizeBody(t *testing.T) {
	bigBody := strings.Repeat("y", maxLLMResponseBytes+1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, bigBody)
	}))
	defer server.Close()

	provider, err := NewVoyageProvider(Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)

	_, err = provider.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"hi"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded")
}

func TestVoyageProvider_ClientError_WrappedInSafeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"unauthorized"}`)
	}))
	defer server.Close()

	provider, err := NewVoyageProvider(Config{APIKey: "test-key", BaseURL: server.URL})
	require.NoError(t, err)

	_, err = provider.Embed(context.Background(), &llm_dto.EmbeddingRequest{
		Input: []string{"hi"},
	})
	require.Error(t, err)

	var safeErr safeerror.Error
	require.True(t, errors.As(err, &safeErr))
	assert.Equal(t, "voyage request rejected", safeErr.SafeMessage())
}

func TestVoyageReadBoundedBody(t *testing.T) {
	body := strings.NewReader(strings.Repeat("y", maxLLMResponseBytes+10))
	data, err := readBoundedBody(body)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errResponseTruncated))
	assert.Equal(t, maxLLMResponseBytes, len(data))
}
