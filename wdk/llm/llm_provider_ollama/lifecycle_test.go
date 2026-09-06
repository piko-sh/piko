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
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"sync/atomic"
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

	transport := cloneTransport(http.DefaultTransport)

	closeContext, closeCancel := context.WithCancelCause(context.Background())
	t.Cleanup(func() {
		closeCancel(errors.New("test finished"))
	})

	return &ollamaProvider{
		client:                api.NewClient(u, &http.Client{Transport: transport}),
		transport:             transport,
		defaultModel:          Model("llama3.2"),
		defaultEmbeddingModel: Model("all-minilm"),
		config:                Config{Host: server.URL}.WithDefaults(),
		closeContext:          closeContext,
		closeCancel:           closeCancel,
		started:               true,
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
			if !assert.True(t, ok) {
				return
			}

			enc := json.NewEncoder(w)
			assert.NoError(t, enc.Encode(api.ChatResponse{
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
			if !assert.True(t, ok) {
				return
			}

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

type startupServer struct {
	server       *httptest.Server
	release      chan struct{}
	versionCalls atomic.Int32
	failFirst    atomic.Int32
	block        bool
}

func newStartupServer(t *testing.T, block bool, failFirst int32) *startupServer {
	t.Helper()

	startup := &startupServer{release: make(chan struct{}), block: block}
	startup.failFirst.Store(failFirst)

	startup.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ollamaVersionPath:
			startup.versionCalls.Add(1)
			if startup.block {
				select {
				case <-startup.release:
				case <-r.Context().Done():
					return
				}
			}
			if startup.failFirst.Add(-1) >= 0 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = io.WriteString(w, `{"version":"0.0.0"}`)
		case "/api/show":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(api.ShowResponse{
				Details:   api.ModelDetails{Family: "bert"},
				ModelInfo: map[string]any{"bert.embedding_length": 384},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(startup.server.Close)
	return startup
}

func newStartupTestProvider(t *testing.T, host string) *ollamaProvider {
	t.Helper()

	p, err := newProvider(Config{Host: host, AutoStart: new(false), ProbeTimeout: 5 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = p.Close(context.Background())
	})
	return p
}

func TestNewProvider_DoesNoNetworkWork(t *testing.T) {
	t.Parallel()

	startup := newStartupServer(t, false, 0)

	p := newStartupTestProvider(t, startup.server.URL)

	assert.Zero(t, startup.versionCalls.Load())
	assert.False(t, p.started)
	assert.Nil(t, p.imageFetcher)
}

func TestNewProvider_RejectsInvalidHost(t *testing.T) {
	t.Parallel()

	_, err := newProvider(Config{Host: "http://[::1]:namedport"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parsing ollama host URL")
}

func TestOllamaProvider_Start_ReachableServer(t *testing.T) {
	t.Parallel()

	startup := newStartupServer(t, false, 0)
	p := newStartupTestProvider(t, startup.server.URL)

	require.NoError(t, p.Start(t.Context()))
	require.NoError(t, p.Start(t.Context()))

	assert.Equal(t, int32(1), startup.versionCalls.Load())
	assert.Equal(t, 384, p.EmbeddingDimensions())
	assert.Nil(t, p.process)
}

func TestOllamaProvider_Start_ConcurrentCallersShareOneAttempt(t *testing.T) {
	t.Parallel()

	startup := newStartupServer(t, true, 0)
	p := newStartupTestProvider(t, startup.server.URL)

	const callers = 4
	results := make(chan error, callers)
	for range callers {
		go func() {
			results <- p.Start(t.Context())
		}()
	}

	require.Eventually(t, func() bool {
		return startup.versionCalls.Load() == 1
	}, 5*time.Second, 5*time.Millisecond)
	close(startup.release)

	for range callers {
		require.NoError(t, <-results)
	}
	assert.Equal(t, int32(1), startup.versionCalls.Load())
}

func TestOllamaProvider_Start_CallerCancellationLeavesStartupRunning(t *testing.T) {
	t.Parallel()

	startup := newStartupServer(t, true, 0)
	p := newStartupTestProvider(t, startup.server.URL)

	cause := errors.New("request abandoned")
	ctx, cancel := context.WithCancelCause(t.Context())
	result := make(chan error, 1)
	go func() {
		result <- p.Start(ctx)
	}()

	require.Eventually(t, func() bool {
		return startup.versionCalls.Load() == 1
	}, 5*time.Second, 5*time.Millisecond)
	cancel(cause)

	err := <-result
	require.Error(t, err)
	assert.ErrorIs(t, err, cause)

	close(startup.release)
	require.NoError(t, p.Start(t.Context()))
	assert.Equal(t, int32(1), startup.versionCalls.Load())
}

func TestOllamaProvider_Start_RetriesAfterFailure(t *testing.T) {
	t.Parallel()

	startup := newStartupServer(t, false, 1)
	p := newStartupTestProvider(t, startup.server.URL)

	err := p.Start(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AutoStart is disabled")
	assert.Contains(t, err.Error(), "unexpected status 503")

	require.NoError(t, p.Start(t.Context()))
	assert.Equal(t, int32(2), startup.versionCalls.Load())
}

func TestOllamaProvider_Start_AfterClose(t *testing.T) {
	t.Parallel()

	startup := newStartupServer(t, false, 0)
	p := newStartupTestProvider(t, startup.server.URL)
	require.NoError(t, p.Close(t.Context()))

	assert.ErrorIs(t, p.Start(t.Context()), errProviderClosed)

	_, err := p.Complete(t.Context(), &llm_dto.CompletionRequest{})
	assert.ErrorIs(t, err, errProviderClosed)

	_, err = p.Stream(t.Context(), &llm_dto.CompletionRequest{})
	assert.ErrorIs(t, err, errProviderClosed)

	_, err = p.Embed(t.Context(), &llm_dto.EmbeddingRequest{})
	assert.ErrorIs(t, err, errProviderClosed)

	_, err = p.ListModels(t.Context())
	assert.ErrorIs(t, err, errProviderClosed)
	assert.Zero(t, startup.versionCalls.Load())
}

func TestOllamaProvider_Close_CancelsStartup(t *testing.T) {
	t.Parallel()

	startup := newStartupServer(t, true, 0)
	p := newStartupTestProvider(t, startup.server.URL)

	result := make(chan error, 1)
	go func() {
		result <- p.Start(t.Context())
	}()

	require.Eventually(t, func() bool {
		return startup.versionCalls.Load() == 1
	}, 5*time.Second, 5*time.Millisecond)

	require.NoError(t, p.Close(t.Context()))

	err := <-result
	require.Error(t, err)
	assert.ErrorIs(t, err, errProviderClosed)
}

func TestOllamaProvider_RecordStartup_StopsProcessStartedAfterClose(t *testing.T) {
	t.Parallel()

	p := &ollamaProvider{closed: true}
	process := &managedProcess{
		command:         &exec.Cmd{Process: &os.Process{Pid: os.Getpid()}},
		done:            make(chan struct{}),
		stopGracePeriod: time.Hour,
		reapTimeout:     time.Hour,
	}
	process.interrupt = func(*exec.Cmd) error {
		close(process.done)
		return nil
	}
	process.kill = func(*exec.Cmd) error {
		return errors.New("kill is not expected")
	}

	attempt := &startupAttempt{done: make(chan struct{})}
	attempt.err = p.recordStartup(process, nil)
	require.ErrorIs(t, attempt.err, errProviderClosed)
	require.NoError(t, process.Stop(t.Context()))

	assert.Nil(t, p.process)
	assert.False(t, p.started)
}

func TestOllamaProvider_StartAndClose_ManagedServer(t *testing.T) {
	config := fakeOllamaConfig(t, fakeOllamaModeServe)

	provider, err := NewOllamaProvider(config)
	require.NoError(t, err)

	require.NoError(t, provider.Start(t.Context()))
	provider.lifecycleMutex.Lock()
	process := provider.process
	provider.lifecycleMutex.Unlock()
	require.NotNil(t, process)

	require.NoError(t, provider.Close(t.Context()))
	requireExited(t, process)
	assert.Nil(t, provider.process)
}

func TestOllamaProvider_Complete_ReturnsResponse(t *testing.T) {
	t.Parallel()

	p := newLifecycleTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/show":
			_ = json.NewEncoder(w).Encode(api.ShowResponse{Modelfile: "FROM llama3.2"})
		case "/api/chat":
			_ = json.NewEncoder(w).Encode(api.ChatResponse{
				Message:         api.Message{Role: "assistant", Content: "hello there"},
				Done:            true,
				DoneReason:      "stop",
				PromptEvalCount: 3,
				EvalCount:       2,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	response, err := p.Complete(t.Context(), &llm_dto.CompletionRequest{
		Messages: []llm_dto.Message{{Role: llm_dto.RoleUser, Content: "hi"}},
	})
	require.NoError(t, err)
	require.Len(t, response.Choices, 1)
	assert.Equal(t, "hello there", response.Choices[0].Message.Content)
	assert.Equal(t, 5, response.Usage.TotalTokens)
}

func TestOllamaProvider_Embed_ReturnsVectors(t *testing.T) {
	t.Parallel()

	p := newLifecycleTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/show":
			_ = json.NewEncoder(w).Encode(api.ShowResponse{Modelfile: "FROM all-minilm"})
		case "/api/embed":
			_ = json.NewEncoder(w).Encode(api.EmbedResponse{
				Model:           "all-minilm",
				Embeddings:      [][]float32{{0.1, 0.2, 0.3}},
				PromptEvalCount: 4,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	response, err := p.Embed(t.Context(), &llm_dto.EmbeddingRequest{Input: []string{"hello"}})
	require.NoError(t, err)
	require.Len(t, response.Embeddings, 1)
	assert.Len(t, response.Embeddings[0].Vector, 3)
	assert.Equal(t, 3, p.EmbeddingDimensions())
}

func TestOllamaProvider_ReturnsPanicsAsErrors(t *testing.T) {
	t.Parallel()

	p := newLifecycleTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	completion, err := p.Complete(t.Context(), nil)
	assert.Nil(t, completion)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ollama completion panicked")
	assert.NotContains(t, err.Error(), "goroutine")

	embedding, err := p.Embed(t.Context(), nil)
	assert.Nil(t, embedding)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ollama embedding panicked")
}

func TestOllamaProvider_ListModels(t *testing.T) {
	t.Parallel()

	p := newLifecycleTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(api.ListResponse{
			Models: []api.ListModelResponse{{Name: "llama3.2:latest", Model: "llama3.2:latest"}},
		})
	})

	models, err := p.ListEmbeddingModels(t.Context())
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "llama3.2:latest", models[0].ID)
}
