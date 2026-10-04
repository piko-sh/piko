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

package llm_provider_ollama

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShortDigest(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{name: "strips sha256 prefix", input: "sha256:2af3b81862c6abcdef1234567890", expect: "2af3b81862c6"},
		{name: "without prefix", input: "2af3b81862c6abcdef1234567890", expect: "2af3b81862c6"},
		{name: "short digest unchanged", input: "abc123", expect: "abc123"},
		{name: "empty string", input: "", expect: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, shortDigest(tt.input))
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name   string
		expect string
		input  int64
	}{
		{name: "bytes", input: 500, expect: "500 B"},
		{name: "kilobytes", input: 1536, expect: "1.5 KB"},
		{name: "megabytes", input: 1048576, expect: "1.0 MB"},
		{name: "gigabytes", input: 1610612736, expect: "1.5 GB"},
		{name: "large megabytes", input: 637_700_000, expect: "608.2 MB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, formatBytes(tt.input))
		})
	}
}

type modelServer struct {
	installed  map[string]string
	pullDigest string
	mu         sync.Mutex
	pullCalls  atomic.Int32
	pullFails  bool
}

func (m *modelServer) handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/api/show":
		var request api.ShowRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		m.mu.Lock()
		_, found := m.installed[request.Model]
		_, foundLatest := m.installed[request.Model+":latest"]
		m.mu.Unlock()
		found = found || foundLatest
		if !found {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"model not found"}`)
			return
		}
		_ = json.NewEncoder(w).Encode(api.ShowResponse{Modelfile: "FROM " + request.Model})
	case "/api/pull":
		m.pullCalls.Add(1)
		if m.pullFails {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = io.WriteString(w, `{"error":"registry unavailable"}`)
			return
		}
		var request api.PullRequest
		_ = json.NewDecoder(r.Body).Decode(&request)
		encoder := json.NewEncoder(w)
		_ = encoder.Encode(api.ProgressResponse{Status: "pulling manifest"})
		_ = encoder.Encode(api.ProgressResponse{Status: "downloading", Digest: "sha256:layer1", Total: 2048, Completed: 1024})
		_ = encoder.Encode(api.ProgressResponse{Status: "downloading", Digest: "sha256:layer1", Total: 2048, Completed: 2048})
		_ = encoder.Encode(api.ProgressResponse{Status: "success"})
		m.mu.Lock()
		m.installed[request.Model] = m.pullDigest
		m.mu.Unlock()
	case "/api/tags":
		m.mu.Lock()
		models := make([]api.ListModelResponse, 0, len(m.installed))
		for name, digest := range m.installed {
			models = append(models, api.ListModelResponse{Name: name, Model: name, Digest: digest})
		}
		m.mu.Unlock()
		_ = json.NewEncoder(w).Encode(api.ListResponse{Models: models})
	case ollamaVersionPath:
		_, _ = io.WriteString(w, `{"version":"0.0.0"}`)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func newModelTestProvider(t *testing.T, models *modelServer, autoPull bool) *ollamaProvider {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(models.handle))
	t.Cleanup(server.Close)

	p, err := newProvider(Config{
		Host:                  server.URL,
		AutoStart:             new(false),
		AutoPull:              new(autoPull),
		DefaultModel:          ModelWithDigest("llama3.2", "abcdef123456"),
		DefaultEmbeddingModel: Model("all-minilm"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = p.Close(context.Background())
	})
	return p
}

func TestEnsureModel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		installed     map[string]string
		ref           ModelRef
		name          string
		model         string
		wantErrText   string
		pullDigest    string
		wantPullCalls int32
		autoPull      bool
		pullFails     bool
	}{
		{
			name:      "installed model without digest",
			installed: map[string]string{"llama3.2": "abcdef123456"},
			model:     "llama3.2",
			ref:       Model("llama3.2"),
		},
		{
			name:      "installed model with matching digest",
			installed: map[string]string{"llama3.2:latest": "sha256:abcdef1234567890"},
			model:     "llama3.2",
			ref:       ModelWithDigest("llama3.2", "abcdef123456"),
		},
		{
			name:        "installed model with mismatching digest",
			installed:   map[string]string{"llama3.2": "sha256:999999999999"},
			model:       "llama3.2",
			ref:         ModelWithDigest("llama3.2", "abcdef123456"),
			wantErrText: "digest mismatch",
		},
		{
			name:        "missing model with auto-pull disabled",
			installed:   map[string]string{},
			model:       "llama3.2",
			ref:         Model("llama3.2"),
			wantErrText: "AutoPull is disabled",
		},
		{
			name:          "missing model is pulled",
			installed:     map[string]string{},
			model:         "llama3.2",
			ref:           ModelWithDigest("llama3.2", "abcdef123456"),
			autoPull:      true,
			pullDigest:    "abcdef1234567890",
			wantPullCalls: 1,
		},
		{
			name:          "failed pull is reported",
			installed:     map[string]string{},
			model:         "llama3.2",
			ref:           Model("llama3.2"),
			autoPull:      true,
			pullFails:     true,
			wantErrText:   "pulling model",
			wantPullCalls: 1,
		},
		{
			name:          "pulled model with mismatching digest is refused",
			installed:     map[string]string{},
			model:         "llama3.2",
			ref:           ModelWithDigest("llama3.2", "abcdef123456"),
			autoPull:      true,
			pullDigest:    "999999999999",
			wantErrText:   "digest mismatch",
			wantPullCalls: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			models := &modelServer{
				installed:  testCase.installed,
				pullFails:  testCase.pullFails,
				pullDigest: testCase.pullDigest,
			}
			p := newModelTestProvider(t, models, testCase.autoPull)

			err := p.ensureModel(t.Context(), testCase.model, testCase.ref)

			assert.Equal(t, testCase.wantPullCalls, models.pullCalls.Load())
			if testCase.wantErrText == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantErrText)
		})
	}
}

func TestVerifyModelDigest_ModelMissingFromList(t *testing.T) {
	t.Parallel()

	models := &modelServer{installed: map[string]string{"other": "abc"}}
	p := newModelTestProvider(t, models, false)

	err := p.verifyModelDigest(t.Context(), "llama3.2", ModelWithDigest("llama3.2", "abcdef123456"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in model list")
}

func TestOllamaProvider_EnsureModels(t *testing.T) {
	t.Parallel()

	models := &modelServer{installed: map[string]string{
		"llama3.2":   "abcdef1234567890",
		"all-minilm": "123456789abc",
	}}
	server := httptest.NewServer(http.HandlerFunc(models.handle))
	t.Cleanup(server.Close)

	provider, err := NewOllamaProvider(Config{
		Host:                  server.URL,
		AutoStart:             new(false),
		AutoPull:              new(false),
		DefaultModel:          ModelWithDigest("llama3.2", "abcdef123456"),
		DefaultEmbeddingModel: Model("all-minilm"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = provider.Close(context.Background())
	})

	require.NoError(t, provider.EnsureModels(t.Context()))
	assert.Equal(t, "llama3.2", provider.DefaultModel())
}

func TestOllamaProvider_EnsureModels_ReportsFailures(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		installed   map[string]string
		name        string
		wantErrText string
	}{
		{name: "missing completion model", installed: map[string]string{}, wantErrText: "ensuring completion model"},
		{
			name:        "missing embedding model",
			installed:   map[string]string{"llama3.2": "abcdef1234567890"},
			wantErrText: "ensuring embedding model",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			models := &modelServer{installed: testCase.installed}
			p := newModelTestProvider(t, models, false)

			err := (&OllamaProvider{ollamaProvider: p}).EnsureModels(t.Context())
			require.Error(t, err)
			assert.Contains(t, err.Error(), testCase.wantErrText)
		})
	}
}

func TestNewOllamaProvider_RejectsInvalidHost(t *testing.T) {
	t.Parallel()

	provider, err := NewOllamaProvider(Config{Host: "http://[::1]:namedport"})
	require.Error(t, err)
	assert.Nil(t, provider)
}
