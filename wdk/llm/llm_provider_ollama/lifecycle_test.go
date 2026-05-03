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
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/llm/llm_dto"
)

func newLifecycleTestProvider(t *testing.T, handler http.HandlerFunc) *ollamaProvider {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	u, err := url.Parse(server.URL)
	require.NoError(t, err)

	transport := http.DefaultTransport.(*http.Transport).Clone()

	closeContext, closeCancel := context.WithCancelCause(context.Background())

	return &ollamaProvider{
		client:                api.NewClient(u, &http.Client{Transport: transport}),
		transport:             transport,
		defaultModel:          Model("llama3.2"),
		defaultEmbeddingModel: Model("all-minilm"),
		config:                Config{}.WithDefaults(),
		closeContext:          closeContext,
		closeCancel:           closeCancel,
	}
}

func TestOllamaProvider_Close_Idempotent(t *testing.T) {
	p := newLifecycleTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	require.NoError(t, p.Close(context.Background()))
	require.NoError(t, p.Close(context.Background()))
	require.NoError(t, p.Close(context.Background()))
}

func TestOllamaProvider_Close_HasCloseContext(t *testing.T) {
	p := newLifecycleTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	require.NotNil(t, p.closeContext, "close context should be initialised")
	require.NotNil(t, p.closeCancel, "close cancel should be initialised")

	select {
	case <-p.closeContext.Done():
		t.Fatal("close context should not be cancelled before Close")
	default:
	}

	require.NoError(t, p.Close(context.Background()))

	select {
	case <-p.closeContext.Done():
	default:
		t.Fatal("close context should be cancelled after Close")
	}
}

func TestOllamaProvider_Close_DrainsActiveStreams(t *testing.T) {
	allowFinish := make(chan struct{})
	firstChunkServed := make(chan struct{}, 1)
	chatHits := make(chan struct{}, 1)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(api.ShowResponse{
				Modelfile: "FROM llama3.2",
			})
			return
		case "/api/chat":
			select {
			case chatHits <- struct{}{}:
			default:
			}
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			require.True(t, ok)

			enc := json.NewEncoder(w)
			require.NoError(t, enc.Encode(api.ChatResponse{
				Message: api.Message{Role: "assistant", Content: "hi"},
			}))
			flusher.Flush()
			select {
			case firstChunkServed <- struct{}{}:
			default:
			}

			select {
			case <-allowFinish:
			case <-r.Context().Done():
				return
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	p := newLifecycleTestProvider(t, handler)

	streamChannel, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.NoError(t, err)

	drained := make(chan struct{})
	go func() {
		defer close(drained)
		for range streamChannel {
		}
	}()

	select {
	case <-firstChunkServed:
	case <-time.After(5 * time.Second):
		close(allowFinish)
		t.Fatal("server never served the first chunk")
	}

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- p.Close(context.Background())
	}()

	select {
	case err := <-closeDone:
		require.NoError(t, err)
	case <-time.After(closeDrainTimeout + 5*time.Second):
		close(allowFinish)
		t.Fatal("Close did not return; stream goroutine appears stuck")
	}

	close(allowFinish)
	<-drained
}

func TestOllamaProvider_Close_CancelsActiveStream(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(api.ShowResponse{
				Modelfile: "FROM llama3.2",
			})
			return
		case "/api/chat":
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			require.True(t, ok)

			enc := json.NewEncoder(w)
			_ = enc.Encode(api.ChatResponse{
				Message: api.Message{Role: "assistant", Content: "first"},
			})
			flusher.Flush()

			<-r.Context().Done()
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	p := newLifecycleTestProvider(t, handler)

	streamChannel, err := p.Stream(context.Background(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.NoError(t, err)

	select {
	case <-streamChannel:
	case <-time.After(2 * time.Second):
		t.Fatal("never received first chunk")
	}

	closeErr := p.Close(context.Background())
	require.NoError(t, closeErr)

	for event := range streamChannel {
		if event.Type == llm_dto.StreamEventError && event.Error != nil {
			assert.True(t, errors.Is(event.Error, context.Canceled) ||
				event.Error.Error() != "")
		}
	}
}
