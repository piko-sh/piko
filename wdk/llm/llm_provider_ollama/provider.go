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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ollama/ollama/api"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/safeerror"
	"piko.sh/piko/internal/security/security_adapters"
	"piko.sh/piko/wdk/goroutine"
	"piko.sh/piko/wdk/logger"
	"piko.sh/piko/wdk/safeconv"
)

const (
	// closeDrainTimeout bounds the time Close waits for active stream goroutines to drain
	// before returning a timeout error.
	closeDrainTimeout = 30 * time.Second

	// embeddingDimensionLookupTimeout bounds the Show request used to learn the default
	// embedding model's vector dimension.
	embeddingDimensionLookupTimeout = 5 * time.Second
)

var (
	// errProviderClosed is returned when the provider is used after Close.
	errProviderClosed = errors.New("ollama provider is closed")

	// errTooManyImages is returned when a request references more images by URL than the
	// configured limit.
	errTooManyImages = errors.New("too many image URLs in request")

	// errMalformedContentPart is returned when an image content part carries no image.
	errMalformedContentPart = errors.New("malformed image content part")

	// errUnsupportedImageScheme is returned when an image URL is not http or https.
	errUnsupportedImageScheme = errors.New("image URL scheme is not http or https")
)

// ollamaProvider implements llm_domain.LLMProviderPort and
// llm_domain.EmbeddingProviderPort for Ollama.
type ollamaProvider struct {
	// closeContext is the provider-level context whose cancellation signals background
	// stream goroutines to exit.
	closeContext context.Context

	// closeCancel cancels closeContext on Close to signal in-flight stream goroutines to
	// wind down.
	closeCancel context.CancelCauseFunc

	// client is the Ollama API client.
	client *api.Client

	// transport is the HTTP transport used by the client, kept for cleanup.
	transport *http.Transport

	// process is non-nil once the provider has spawned the Ollama server; guarded by
	// lifecycleMutex.
	process *managedProcess

	// startup is the startup attempt in progress, or nil when none is running; guarded by
	// lifecycleMutex.
	startup *startupAttempt

	// imageFetcher is an HTTP client used to download URL-referenced images. Only set when
	// Config.ImageFetch is non-nil.
	imageFetcher *http.Client

	// defaultModel is the model reference to use for completions.
	defaultModel ModelRef

	// defaultEmbeddingModel is the model reference to use for embeddings.
	defaultEmbeddingModel ModelRef

	// config holds the provider configuration settings.
	config Config

	// backgroundWaitGroup tracks active stream and startup goroutines so Close can wait for
	// them to drain.
	backgroundWaitGroup sync.WaitGroup

	// lifecycleMutex guards process, startup, started and closed, and orders goroutine
	// registration on backgroundWaitGroup before Close waits on it.
	lifecycleMutex sync.Mutex

	// embeddingDim caches the vector dimension reported by the embedding model. Populated
	// eagerly from Show during construction, or lazily from the first Embed response.
	embeddingDim atomic.Int32

	// closeOnce guards Close so it is idempotent.
	closeOnce sync.Once

	// started is set once the server has been found reachable or started; guarded by
	// lifecycleMutex.
	started bool

	// closed is set when Close begins; guarded by lifecycleMutex.
	closed bool
}

// startupAttempt records one run of the startup sequence shared by every caller waiting
// on it.
type startupAttempt struct {
	// err is the outcome; it is written before done is closed and must only be read after
	// done is closed.
	err error

	// done is closed when the attempt finishes.
	done chan struct{}
}

var (
	_ llm_domain.LLMProviderPort = (*ollamaProvider)(nil)

	_ llm_domain.EmbeddingProviderPort = (*ollamaProvider)(nil)
)

// Complete sends a completion request to Ollama.
//
// Starts the provider first when it has not been started.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the prompt and model
// settings.
//
// Returns *llm_dto.CompletionResponse which contains the generated completion.
// Returns error when startup, image fetching or the API request fails, or when the call
// panics.
func (p *ollamaProvider) Complete(
	ctx context.Context, request *llm_dto.CompletionRequest,
) (response *llm_dto.CompletionResponse, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			response, err = nil, panicError(ctx, "completion", recovered)
		}
	}()

	ctx, l := logger.From(ctx, log)
	completeCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		completeDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	model, ref := p.resolveModel(request.Model, p.defaultModel)

	chatRequest, err := p.prepareChat(ctx, request, model, ref)
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		return nil, err
	}
	chatRequest.Stream = new(bool)

	l.Debug("Sending Ollama completion request",
		logger.String("model", model),
		logger.Int("message_count", len(request.Messages)),
	)

	var chatResp api.ChatResponse

	err = p.client.Chat(ctx, chatRequest, func(chunk api.ChatResponse) error {
		chatResp = chunk
		return nil
	})
	if err != nil {
		completeErrorCount.Add(ctx, 1)
		wrapped := fmt.Errorf("ollama completion failed: %w", wrapError(err))
		return nil, sanitiseProviderError(wrapped, "ollama request rejected")
	}

	return p.convertChatResponse(&chatResp, model), nil
}

// Embed generates embeddings for the given input texts.
//
// Starts the provider first when it has not been started.
//
// Takes request (*llm_dto.EmbeddingRequest) which contains the embedding parameters.
//
// Returns *llm_dto.EmbeddingResponse which contains the generated embeddings.
// Returns error when startup or the request fails, or when the call panics.
func (p *ollamaProvider) Embed(
	ctx context.Context, request *llm_dto.EmbeddingRequest,
) (response *llm_dto.EmbeddingResponse, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			response, err = nil, panicError(ctx, "embedding", recovered)
		}
	}()

	ctx, l := logger.From(ctx, log)
	embedCount.Add(ctx, 1)
	start := time.Now()

	defer func() {
		embedDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	model, ref := p.resolveModel(request.Model, p.defaultEmbeddingModel)

	if err = p.ensureStarted(ctx); err != nil {
		embedErrorCount.Add(ctx, 1)
		return nil, err
	}
	if err = p.ensureModel(ctx, model, ref); err != nil {
		embedErrorCount.Add(ctx, 1)
		return nil, err
	}

	l.Debug("Sending Ollama embedding request",
		logger.String("model", model),
		logger.Int("input_count", len(request.Input)),
	)

	embedResp, err := p.client.Embed(ctx, &api.EmbedRequest{
		Model: model,
		Input: request.Input,
	})
	if err != nil {
		embedErrorCount.Add(ctx, 1)
		wrapped := fmt.Errorf("ollama embedding failed: %w", wrapError(err))
		return nil, sanitiseProviderError(wrapped, "ollama embedding rejected")
	}

	embeddings := make([]llm_dto.Embedding, len(embedResp.Embeddings))
	for i, vec := range embedResp.Embeddings {
		f32 := make([]float32, len(vec))
		for j, v := range vec {
			f32[j] = float32(v)
		}
		embeddings[i] = llm_dto.NewFloat32Embedding(i, f32)
	}

	if p.embeddingDim.Load() == 0 && len(embeddings) > 0 {
		p.embeddingDim.Store(safeconv.IntToInt32(len(embeddings[0].Vector)))
	}

	usage := llm_dto.NewEmbeddingUsage(embedResp.PromptEvalCount, embedResp.PromptEvalCount)
	return llm_dto.NewEmbeddingResponse(model, embeddings, usage), nil
}

// ListModels returns available models from the Ollama server.
//
// Starts the provider first when it has not been started.
//
// Returns []llm_dto.ModelInfo which contains model metadata.
// Returns error when startup or the API request fails.
func (p *ollamaProvider) ListModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	if err := p.ensureStarted(ctx); err != nil {
		return nil, err
	}
	response, err := p.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing ollama models: %w", wrapError(err))
	}

	result := make([]llm_dto.ModelInfo, len(response.Models))
	for i := range response.Models {
		m := &response.Models[i]
		result[i] = llm_dto.ModelInfo{
			ID:                       m.Name,
			Name:                     m.Name,
			Provider:                 "ollama",
			Created:                  m.ModifiedAt.Unix(),
			SupportsStreaming:        true,
			ContextWindow:            0,
			MaxOutputTokens:          0,
			SupportsTools:            false,
			SupportsStructuredOutput: false,
			SupportsVision:           false,
		}
	}
	return result, nil
}

// ListEmbeddingModels returns available models from the Ollama server. Ollama does not
// distinguish between completion and embedding models in its listing API, so this returns
// all available models.
//
// Returns []llm_dto.ModelInfo which contains model metadata.
// Returns error when the API request fails.
func (p *ollamaProvider) ListEmbeddingModels(ctx context.Context) ([]llm_dto.ModelInfo, error) {
	return p.ListModels(ctx)
}

// EmbeddingDimensions returns the cached vector dimension for the default embedding
// model.
//
// The value is learned when the provider starts (if the model is already pulled) or after
// the first successful Embed call.
//
// Returns int which is the vector dimension, or 0 if not yet known.
func (p *ollamaProvider) EmbeddingDimensions() int {
	return int(p.embeddingDim.Load())
}

// SupportsStreaming reports whether the provider supports streaming.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsStreaming() bool {
	return true
}

// SupportsStructuredOutput reports whether the provider supports structured output.
//
// Returns bool which is false.
func (*ollamaProvider) SupportsStructuredOutput() bool {
	return false
}

// SupportsTools reports whether the provider supports tool calling.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsTools() bool {
	return true
}

// SupportsPenalties reports whether the provider supports frequency and presence
// penalties.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsPenalties() bool { return true }

// SupportsSeed reports whether the provider supports deterministic seed.
//
// Returns bool which is true.
func (*ollamaProvider) SupportsSeed() bool { return true }

// SupportsParallelToolCalls reports whether the provider supports parallel tool calls.
//
// Returns bool which is false.
func (*ollamaProvider) SupportsParallelToolCalls() bool { return false }

// SupportsMessageName reports whether the provider supports the name field on messages.
//
// Returns bool which is false.
func (*ollamaProvider) SupportsMessageName() bool { return false }

// Close releases resources. Cancels in-flight stream and startup goroutines, waits for
// them to drain within a bounded timeout, releases idle HTTP connections, and terminates
// any managed Ollama subprocess.
//
// Returns error when the close drain exceeds its bounded wait or process termination
// fails.
//
// Safe for concurrent use through closeOnce. Marks the provider closed so no new startup
// begins, cancels closeContext via closeCancel to signal active goroutines, then waits on
// backgroundWaitGroup before stopping the managed process.
func (p *ollamaProvider) Close(ctx context.Context) error {
	var closeErr error
	p.closeOnce.Do(func() {
		p.lifecycleMutex.Lock()
		p.closed = true
		p.lifecycleMutex.Unlock()

		if p.closeCancel != nil {
			p.closeCancel(errProviderClosed)
		}

		closeErr = p.waitForBackgroundWork(ctx)

		if p.transport != nil {
			p.transport.CloseIdleConnections()
		}
		if p.imageFetcher != nil {
			p.imageFetcher.CloseIdleConnections()
		}

		p.lifecycleMutex.Lock()
		process := p.process
		p.process = nil
		p.lifecycleMutex.Unlock()

		if processErr := process.Stop(ctx); processErr != nil {
			closeErr = errors.Join(closeErr, processErr)
		}
	})
	return closeErr
}

// Start makes sure an Ollama server is answering, spawning a managed instance when
// AutoStart allows it, and learns the default embedding model's vector dimension.
//
// Requests start the provider on demand, so calling Start is optional; call it at
// application startup to fail fast and avoid first-request latency. Concurrent callers
// share one startup attempt. The attempt is not tied to ctx. A caller whose ctx ends
// stops waiting, while the attempt itself is bounded by StartupTimeout and Close. A
// failed attempt is retried by the next caller.
//
// Returns error when the server cannot be reached or started, when ctx ends first, or
// when the provider is closed.
func (p *ollamaProvider) Start(ctx context.Context) error {
	return p.ensureStarted(ctx)
}

// DefaultModel returns the name of the default model.
//
// Implements LLMProviderPort.DefaultModel.
//
// Returns string which is the configured default model name.
func (p *ollamaProvider) DefaultModel() string {
	return p.defaultModel.Name
}

// prepareChat starts the provider when needed, ensures the model is available and builds
// the chat request.
//
// Takes request (*llm_dto.CompletionRequest) which is the completion to convert.
// Takes model (string) which is the resolved model name.
// Takes ref (ModelRef) which carries the digest to verify, if any.
//
// Returns *api.ChatRequest which is ready to send.
// Returns error when startup, the model check or the request conversion fails.
func (p *ollamaProvider) prepareChat(
	ctx context.Context, request *llm_dto.CompletionRequest, model string, ref ModelRef,
) (*api.ChatRequest, error) {
	if err := p.ensureStarted(ctx); err != nil {
		return nil, err
	}
	if err := p.ensureModel(ctx, model, ref); err != nil {
		return nil, err
	}
	return p.buildChatRequest(ctx, request, model)
}

// ensureStarted waits for the provider to be started, beginning a startup attempt when
// none is running.
//
// Returns error when the attempt fails, ctx ends first, or the provider is closed.
//
// Safe for concurrent use. Holds lifecycleMutex only to inspect or begin an attempt,
// never while waiting for it.
func (p *ollamaProvider) ensureStarted(ctx context.Context) error {
	p.lifecycleMutex.Lock()
	if p.closed {
		p.lifecycleMutex.Unlock()
		return errProviderClosed
	}
	if p.started {
		p.lifecycleMutex.Unlock()
		return nil
	}
	attempt := p.startup
	if attempt == nil {
		attempt = &startupAttempt{err: nil, done: make(chan struct{})}
		p.startup = attempt
		startupContext := context.WithoutCancel(ctx)
		p.backgroundWaitGroup.Go(func() {
			p.runStartup(startupContext, attempt)
		})
	}
	p.lifecycleMutex.Unlock()

	select {
	case <-attempt.done:
		return attempt.err
	case <-ctx.Done():
		return fmt.Errorf("waiting for ollama provider to start: %w", context.Cause(ctx))
	}
}

// runStartup performs one startup attempt and publishes its outcome.
//
// The attempt is detached from the caller's cancellation and is cancelled when the
// provider closes.
//
// Takes attempt (*startupAttempt) which receives the outcome.
func (p *ollamaProvider) runStartup(ctx context.Context, attempt *startupAttempt) {
	defer close(attempt.done)

	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(errors.New("ollama provider startup finished"))
	if p.closeContext != nil {
		stopWatching := context.AfterFunc(p.closeContext, func() {
			cancel(context.Cause(p.closeContext))
		})
		defer stopWatching()
	}

	process, err := p.startServer(ctx)
	attempt.err = p.recordStartup(process, err)
	if attempt.err == nil {
		return
	}
	if stopErr := process.Stop(ctx); stopErr != nil {
		attempt.err = errors.Join(attempt.err, stopErr)
	}
}

// startServer makes sure a server answers and learns the embedding dimension.
//
// Returns *managedProcess which is the server spawned by this call, or nil when one was
// already running.
// Returns error when the server cannot be reached or started, or when startup panics.
func (p *ollamaProvider) startServer(ctx context.Context) (process *managedProcess, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = panicError(ctx, "startup", recovered)
		}
	}()

	process, err = ensureServerRunning(ctx, &p.config)
	if err != nil {
		return nil, err
	}
	p.tryPopulateEmbeddingDim(ctx, p.defaultEmbeddingModel.Name)
	return process, nil
}

// recordStartup records the outcome of a startup attempt.
//
// Takes process (*managedProcess) which is the server the attempt spawned, or nil.
// Takes err (error) which is the attempt's failure, or nil on success.
//
// Returns error which is the outcome to report to waiting callers; when it is non-nil the
// caller must stop process, because the provider does not keep it.
//
// Safe for concurrent use. Holds lifecycleMutex for bookkeeping only; the caller stops
// any unrecorded process after the lock is released.
func (p *ollamaProvider) recordStartup(process *managedProcess, err error) error {
	p.lifecycleMutex.Lock()
	defer p.lifecycleMutex.Unlock()

	p.startup = nil
	if err != nil {
		return err
	}
	if p.closed {
		return errProviderClosed
	}
	p.process = process
	p.started = true
	return nil
}

// goBackground runs task on a goroutine tracked by backgroundWaitGroup, unless the
// provider is closed.
//
// Takes task (func()) which is the work to run.
//
// Returns error which is errProviderClosed when Close has begun.
//
// Serialises registration of the goroutine under lifecycleMutex so it cannot race with
// Close during the backgroundWaitGroup wait.
func (p *ollamaProvider) goBackground(task func()) error {
	p.lifecycleMutex.Lock()
	defer p.lifecycleMutex.Unlock()

	if p.closed {
		return errProviderClosed
	}
	p.backgroundWaitGroup.Go(task)
	return nil
}

// waitForBackgroundWork waits for stream and startup goroutines to finish, bounded by
// closeDrainTimeout and ctx.
//
// Returns error when the wait ends before every goroutine has finished.
//
// Spawns a goroutine to wait for tracked background work. The caller may return on
// cancellation or timeout before that goroutine finishes.
func (p *ollamaProvider) waitForBackgroundWork(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer goroutine.RecoverPanic(ctx, "llm.ollamaProvider.Close.wait")
		p.backgroundWaitGroup.Wait()
	}()

	waitContext, cancel := context.WithTimeoutCause(ctx, closeDrainTimeout,
		fmt.Errorf("ollama provider close drain exceeded %s", closeDrainTimeout))
	defer cancel()

	select {
	case <-done:
		return nil
	case <-waitContext.Done():
		return fmt.Errorf("ollama provider close timed out: %w", context.Cause(waitContext))
	}
}

// tryPopulateEmbeddingDim queries the Ollama Show API for the given model and caches the
// embedding dimension if found.
//
// If the model is not yet pulled or the server does not report the dimension, the cached
// value remains 0 and is learned from the first Embed response.
//
// Takes ctx (context.Context) which bounds the Show API call.
// Takes model (string) which is the model name to query for its embedding dimension.
func (p *ollamaProvider) tryPopulateEmbeddingDim(ctx context.Context, model string) {
	if p.embeddingDim.Load() != 0 {
		return
	}

	ctx, cancel := context.WithTimeoutCause(ctx, embeddingDimensionLookupTimeout,
		fmt.Errorf("ollama embedding dimension lookup exceeded %s", embeddingDimensionLookupTimeout))
	defer cancel()

	response, err := p.client.Show(ctx, &api.ShowRequest{Model: model})
	if err != nil {
		_, l := logger.From(ctx, log)
		l.Internal("Embedding dimension lookup skipped",
			logger.String(logKeyModel, model),
			logger.Error(err),
		)
		return
	}

	if dim := embeddingDimFromShow(response); dim > 0 {
		p.embeddingDim.Store(safeconv.IntToInt32(dim))
	}
}

// buildChatRequest converts a CompletionRequest to an Ollama ChatRequest.
//
// Image URLs are counted against the configured limit before any is fetched, and a
// warning reports image URLs skipped because fetching is disabled.
//
// Takes ctx (context.Context) which controls cancellation for image fetches.
// Takes request (*llm_dto.CompletionRequest) which contains the completion settings.
// Takes model (string) which specifies the model to use.
//
// Returns *api.ChatRequest which is ready for the Ollama API.
// Returns error when the request references too many image URLs or an image cannot be
// decoded or fetched.
func (p *ollamaProvider) buildChatRequest(
	ctx context.Context, request *llm_dto.CompletionRequest, model string,
) (*api.ChatRequest, error) {
	if err := p.checkImageURLs(ctx, request.Messages); err != nil {
		return nil, err
	}

	messages := make([]api.Message, len(request.Messages))
	for i, message := range request.Messages {
		converted, err := p.convertMessage(ctx, message)
		if err != nil {
			return nil, fmt.Errorf("converting message %d: %w", i, err)
		}
		messages[i] = converted
	}

	chatRequest := &api.ChatRequest{
		Model:    model,
		Messages: messages,
	}

	if len(request.Tools) > 0 {
		chatRequest.Tools = convertTools(request.Tools)
		chatRequest.Think = &api.ThinkValue{Value: false}
	}

	if options := buildChatOptions(request); len(options) > 0 {
		chatRequest.Options = options
	}

	if request.ResponseFormat != nil {
		switch request.ResponseFormat.Type { //nolint:exhaustive // exhaustive case-set intentionally partial; missing entries are no-ops
		case llm_dto.ResponseFormatJSONObject, llm_dto.ResponseFormatJSONSchema:
			chatRequest.Format = json.RawMessage(`"json"`)
		}
	}

	return chatRequest, nil
}

// checkImageURLs enforces the per-request image URL limit before anything is fetched, and
// warns when image URLs will be skipped because fetching is disabled.
//
// Takes messages ([]llm_dto.Message) which are the request messages to inspect.
//
// Returns error which wraps errTooManyImages when the limit is exceeded.
func (p *ollamaProvider) checkImageURLs(ctx context.Context, messages []llm_dto.Message) error {
	count := countImageURLParts(messages)
	if count == 0 {
		return nil
	}

	if p.imageFetcher == nil {
		_, l := logger.From(ctx, log)
		l.Warn("Skipping image URLs because Ollama image fetching is disabled",
			logger.Int("skipped_image_count", count),
		)
		return nil
	}

	limit := p.imageFetchSettings().MaxImages
	if count > limit {
		return fmt.Errorf("%w: %d image URLs exceed the limit of %d", errTooManyImages, count, limit)
	}
	return nil
}

// imageFetchSettings returns the image fetch configuration with defaults applied.
//
// Returns ImageFetchConfig which holds the effective limits.
func (p *ollamaProvider) imageFetchSettings() ImageFetchConfig {
	if p.config.ImageFetch == nil {
		return ImageFetchConfig{}.withDefaults()
	}
	return p.config.ImageFetch.withDefaults()
}

// convertMessage converts a single llm_dto.Message to an Ollama api.Message, handling
// multimodal content parts, tool calls, and tool results.
//
// Takes message (llm_dto.Message) which is the message to convert.
//
// Returns api.Message which is the converted Ollama message.
// Returns error when an image part cannot be decoded or fetched.
func (p *ollamaProvider) convertMessage(ctx context.Context, message llm_dto.Message) (api.Message, error) {
	m := api.Message{
		Role:    string(message.Role),
		Content: message.Content,
	}

	if len(message.ContentParts) > 0 {
		text, images, err := p.convertContentParts(ctx, message.ContentParts)
		if err != nil {
			return api.Message{}, err
		}
		m.Content = text
		m.Images = images
	}

	if message.Role == llm_dto.RoleAssistant && len(message.ToolCalls) > 0 {
		m.ToolCalls = convertDTOToolCalls(ctx, message.ToolCalls)
	}

	if message.Role == llm_dto.RoleTool && message.ToolCallID != nil {
		m.ToolCallID = *message.ToolCallID
	}

	return m, nil
}

// convertContentParts extracts text and images from multimodal content parts.
//
// Image URL parts are skipped when image fetching is disabled.
//
// Takes parts ([]llm_dto.ContentPart) which contains the content parts to process.
//
// Returns string which is the concatenated text content.
// Returns []api.ImageData which holds decoded image bytes.
// Returns error when an image part is malformed or cannot be decoded or fetched.
func (p *ollamaProvider) convertContentParts(
	ctx context.Context, parts []llm_dto.ContentPart,
) (string, []api.ImageData, error) {
	var textParts []string
	var images []api.ImageData

	for i, part := range parts {
		switch part.Type {
		case llm_dto.ContentPartTypeText:
			if part.Text != nil {
				textParts = append(textParts, *part.Text)
			}
		case llm_dto.ContentPartTypeImageData:
			image, err := decodeImageData(part)
			if err != nil {
				return "", nil, fmt.Errorf("content part %d: %w", i, err)
			}
			images = append(images, image)
		case llm_dto.ContentPartTypeImageURL:
			if p.imageFetcher == nil {
				continue
			}
			image, err := p.fetchImagePart(ctx, part)
			if err != nil {
				return "", nil, fmt.Errorf("content part %d: %w", i, err)
			}
			images = append(images, image)
		}
	}

	return strings.Join(textParts, ""), images, nil
}

// fetchImagePart downloads the image referenced by a URL content part.
//
// Takes part (llm_dto.ContentPart) which contains the image URL to fetch.
//
// Returns api.ImageData which is the downloaded bytes.
// Returns error when the part has no URL or the fetch fails.
func (p *ollamaProvider) fetchImagePart(ctx context.Context, part llm_dto.ContentPart) (api.ImageData, error) {
	if part.ImageURL == nil || part.ImageURL.URL == "" {
		return nil, fmt.Errorf("%w: image URL part has no URL", errMalformedContentPart)
	}
	data, err := p.fetchImage(ctx, part.ImageURL.URL)
	if err != nil {
		return nil, fmt.Errorf("fetching image: %w", err)
	}
	return data, nil
}

// fetchImage downloads an image from the given URL, respecting the configured size limit.
//
// Takes imageURL (string) which is the http or https URL of the image to download.
//
// Returns []byte which holds the raw image bytes.
// Returns error when the URL is not http or https, the fetch fails or reaches a
// non-public address, the response status is not OK, or the image exceeds the configured
// size limit.
func (p *ollamaProvider) fetchImage(ctx context.Context, imageURL string) ([]byte, error) {
	if p.imageFetcher == nil {
		return nil, errors.New("image fetching is not enabled")
	}

	parsed, err := url.Parse(imageURL)
	if err != nil {
		return nil, fmt.Errorf("parsing image URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("%w: %q", errUnsupportedImageScheme, parsed.Scheme)
	}

	settings := p.imageFetchSettings()

	ctx, cancel := context.WithTimeoutCause(ctx, settings.Timeout,
		fmt.Errorf("image fetch from %s exceeded %s", parsed.Host, settings.Timeout),
	)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating image request: %w", err)
	}

	response, err := p.imageFetcher.Do(request)
	if err != nil {
		if cause := context.Cause(ctx); cause != nil {
			return nil, errors.Join(err, cause)
		}
		return nil, err
	}
	defer func() {
		_ = drainAndClose(response.Body)
	}()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d from %s", response.StatusCode, parsed.Host)
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, settings.MaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("reading image body: %w", err)
	}
	if int64(len(data)) > settings.MaxBytes {
		return nil, fmt.Errorf("image exceeds maximum size of %d bytes", settings.MaxBytes)
	}

	return data, nil
}

// convertChatResponse converts an Ollama ChatResponse to a CompletionResponse.
//
// Takes response (*api.ChatResponse) which is the Ollama response.
// Takes model (string) which is the model that was used.
//
// Returns *llm_dto.CompletionResponse with the converted data.
func (*ollamaProvider) convertChatResponse(response *api.ChatResponse, model string) *llm_dto.CompletionResponse {
	message := llm_dto.Message{
		Role:         llm_dto.RoleAssistant,
		Content:      response.Message.Content,
		Name:         nil,
		ToolCallID:   nil,
		ContentParts: nil,
		ToolCalls:    nil,
	}

	finishReason := llm_dto.FinishReasonStop

	if len(response.Message.ToolCalls) > 0 {
		message.ToolCalls = convertOllamaToolCalls(response.Message.ToolCalls)
		finishReason = llm_dto.FinishReasonToolCalls
	}

	return &llm_dto.CompletionResponse{
		Model: model,
		Choices: []llm_dto.Choice{
			{
				Index:        0,
				Message:      message,
				FinishReason: finishReason,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     response.PromptEvalCount,
			CompletionTokens: response.EvalCount,
			TotalTokens:      response.PromptEvalCount + response.EvalCount,
			EstimatedCost:    nil,
			CachedTokens:     0,
		},
		FallbackInfo: nil,
		ID:           "",
		Sources:      nil,
		Created:      0,
	}
}

// newProvider creates a new Ollama provider with the given settings.
//
// No network or process work happens here; the server is reached, or started, by Start or
// on the first request.
//
// Takes config (Config) which contains the provider settings.
//
// Returns *ollamaProvider which is the configured provider that also implements
// llm_domain.EmbeddingProviderPort.
// Returns error when the configuration is not valid or the host URL cannot be parsed.
func newProvider(config Config) (*ollamaProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	config = config.WithDefaults()

	ollamaURL, err := url.Parse(config.Host)
	if err != nil {
		return nil, fmt.Errorf("parsing ollama host URL: %w", err)
	}

	closeContext, closeCancel := context.WithCancelCause(context.Background())

	transport := cloneTransport(http.DefaultTransport)

	p := &ollamaProvider{}
	p.config = config
	p.defaultModel = config.DefaultModel
	p.defaultEmbeddingModel = config.DefaultEmbeddingModel
	p.closeContext = closeContext
	p.closeCancel = closeCancel
	p.transport = transport
	p.client = api.NewClient(ollamaURL, &http.Client{
		Transport: transport,
		Timeout:   config.HTTPTimeout,
	})

	if config.ImageFetch != nil {
		p.imageFetcher = security_adapters.NewPublicHTTPClient(
			security_adapters.WithPublicClientTimeout(config.ImageFetch.Timeout),
			security_adapters.WithPublicClientMaxRedirects(config.ImageFetch.redirectLimit()),
		)
	}

	return p, nil
}

// ensureServerRunning checks that an Ollama server answers at the configured host,
// starting a managed instance when it does not and auto-start is enabled.
//
// Takes config (*Config) which supplies the host, binary path, auto-start setting and
// process timeouts.
//
// Returns *managedProcess which is the started instance, or nil when the server was
// already reachable.
// Returns error when the server is unreachable and auto-start is disabled or fails.
func ensureServerRunning(ctx context.Context, config *Config) (*managedProcess, error) {
	healthURL, err := healthURLFor(config.Host)
	if err != nil {
		return nil, err
	}

	probeErr := probeOnce(ctx, healthURL, config.ProbeTimeout)
	if probeErr == nil {
		return nil, nil
	}

	if !config.autoStartEnabled() {
		return nil, fmt.Errorf("ollama server not reachable at %s and AutoStart is disabled: %w",
			config.Host, probeErr)
	}

	ctx, l := logger.From(ctx, log)
	l.Notice("Ollama server not reachable, starting managed instance",
		logger.String("host", config.Host),
		logger.String("reason", probeErr.Error()),
	)

	process, err := startOllama(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("auto-starting ollama: %w", err)
	}
	return process, nil
}

// sanitiseProviderError wraps a 4xx provider error in a safeerror so HTTP edges can
// sanitise it before returning to the user. Non-4xx errors are returned unchanged so
// retry classification continues to work.
//
// Takes err (error) which is the error to inspect.
// Takes safeMessage (string) which is shown to end users for 4xx errors.
//
// Returns error which is wrapped when err carries a 4xx status code.
func sanitiseProviderError(err error, safeMessage string) error {
	if providerErr, ok := errors.AsType[*llm_domain.ProviderError](err); ok && providerErr.StatusCode >= http.StatusBadRequest && providerErr.StatusCode < http.StatusInternalServerError {
		return safeerror.NewError(safeMessage, err)
	}
	return err
}

// panicError logs a recovered panic with its stack once and returns a stack-free error.
//
// Takes operation (string) which names the provider operation that panicked.
// Takes recovered (any) which is the value passed to panic.
//
// Returns error which describes the panic without the stack.
func panicError(ctx context.Context, operation string, recovered any) error {
	_, l := logger.From(ctx, log)
	l.Warn("Recovered panic in Ollama provider",
		logger.String("operation", operation),
		logger.String("panic", fmt.Sprintf("%v", recovered)),
		logger.String("stack", string(debug.Stack())),
	)
	return fmt.Errorf("ollama %s panicked: %v", operation, recovered)
}

// countImageURLParts counts the image URL content parts across all messages.
//
// Takes messages ([]llm_dto.Message) which are the request messages.
//
// Returns int which is the number of image URL parts.
func countImageURLParts(messages []llm_dto.Message) int {
	count := 0
	for i := range messages {
		for j := range messages[i].ContentParts {
			if messages[i].ContentParts[j].Type == llm_dto.ContentPartTypeImageURL {
				count++
			}
		}
	}
	return count
}

// embeddingDimFromShow extracts the embedding vector dimension from an Ollama
// ShowResponse. The dimension is stored in ModelInfo under the key
// "<family>.embedding_length" where family comes from Details.Family.
//
// Takes response (*api.ShowResponse) which contains the model metadata including family
// and embedding length.
//
// Returns int which is the embedding dimension, or 0 when the dimension cannot be
// determined.
func embeddingDimFromShow(response *api.ShowResponse) int {
	family := response.Details.Family
	if family == "" {
		return 0
	}

	key := family + ".embedding_length"

	v, ok := response.ModelInfo[key]
	if !ok {
		return 0
	}

	f, ok := v.(float64)
	if !ok || f <= 0 {
		return 0
	}

	return int(f)
}

// decodeImageData decodes base64 image data from a content part.
//
// Takes part (llm_dto.ContentPart) which contains the base64 encoded image data.
//
// Returns api.ImageData which is the decoded bytes.
// Returns error when the part has no image data or the data is not valid base64.
func decodeImageData(part llm_dto.ContentPart) (api.ImageData, error) {
	if part.ImageData == nil {
		return nil, fmt.Errorf("%w: image data part has no data", errMalformedContentPart)
	}
	decoded, err := base64.StdEncoding.DecodeString(part.ImageData.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding inline image data: %w", err)
	}
	return decoded, nil
}

// buildChatOptions builds the Ollama options map from the completion request.
//
// Takes request (*llm_dto.CompletionRequest) which provides the generation parameters to
// translate into Ollama options.
//
// Returns map[string]any which holds the Ollama option keys and values.
func buildChatOptions(request *llm_dto.CompletionRequest) map[string]any {
	options := map[string]any{}
	if request.Temperature != nil {
		options["temperature"] = *request.Temperature
	}
	if request.TopP != nil {
		options["top_p"] = *request.TopP
	}
	if request.MaxTokens != nil {
		options["num_predict"] = *request.MaxTokens
	}
	if len(request.Stop) > 0 {
		options["stop"] = request.Stop
	}
	if request.Seed != nil {
		options["seed"] = *request.Seed
	}
	if request.FrequencyPenalty != nil {
		options["frequency_penalty"] = *request.FrequencyPenalty
	}
	if request.PresencePenalty != nil {
		options["presence_penalty"] = *request.PresencePenalty
	}
	maps.Copy(options, request.ProviderOptions)
	return options
}

// convertTools maps DTO tool definitions to Ollama's api.Tools type.
//
// Takes tools ([]llm_dto.ToolDefinition) which contains the tool definitions to convert.
//
// Returns api.Tools which holds the converted Ollama tools.
func convertTools(tools []llm_dto.ToolDefinition) api.Tools {
	out := make(api.Tools, len(tools))
	for i, td := range tools {
		params := api.ToolFunctionParameters{
			Type: "object",
		}

		if td.Function.Parameters != nil {
			params.Required = td.Function.Parameters.Required
			params.Properties = convertSchemaProperties(td.Function.Parameters.Properties)
		}

		description := ""
		if td.Function.Description != nil {
			description = *td.Function.Description
		}

		out[i] = api.Tool{
			Type: "function",
			Function: api.ToolFunction{
				Name:        td.Function.Name,
				Description: description,
				Parameters:  params,
			},
		}
	}
	return out
}

// convertSchemaProperties maps JSONSchema properties to Ollama's ordered
// ToolPropertiesMap, sorting keys alphabetically for deterministic output.
//
// Takes props (map[string]*llm_dto.JSONSchema) which contains the schema properties to
// convert.
//
// Returns *api.ToolPropertiesMap which holds the converted properties, or nil when props
// is empty.
func convertSchemaProperties(props map[string]*llm_dto.JSONSchema) *api.ToolPropertiesMap {
	if len(props) == 0 {
		return nil
	}

	keys := slices.Sorted(maps.Keys(props))

	pm := api.NewToolPropertiesMap()
	for _, k := range keys {
		pm.Set(k, convertSchemaToProperty(props[k]))
	}
	return pm
}

// convertSchemaToProperty recursively converts a single JSONSchema to an Ollama
// ToolProperty.
//
// Takes schema (*llm_dto.JSONSchema) which is the schema to convert.
//
// Returns api.ToolProperty which holds the converted property.
func convertSchemaToProperty(schema *llm_dto.JSONSchema) api.ToolProperty {
	if schema == nil {
		return api.ToolProperty{}
	}

	prop := api.ToolProperty{
		Type: api.PropertyType{schema.Type},
		Enum: schema.Enum,
	}

	if schema.Description != nil {
		prop.Description = *schema.Description
	}

	if len(schema.Properties) > 0 {
		prop.Properties = convertSchemaProperties(schema.Properties)
	}

	if len(schema.AnyOf) > 0 {
		prop.AnyOf = make([]api.ToolProperty, len(schema.AnyOf))
		for i, s := range schema.AnyOf {
			prop.AnyOf[i] = convertSchemaToProperty(s)
		}
	}

	return prop
}

// convertOllamaToolCalls maps Ollama tool calls to the DTO representation.
//
// Takes calls ([]api.ToolCall) which contains the Ollama tool calls to convert.
//
// Returns []llm_dto.ToolCall which holds the converted calls.
func convertOllamaToolCalls(calls []api.ToolCall) []llm_dto.ToolCall {
	out := make([]llm_dto.ToolCall, len(calls))
	for i, tc := range calls {
		out[i] = llm_dto.ToolCall{
			ID:   tc.ID,
			Type: "function",
			Function: llm_dto.FunctionCall{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments.String(),
			},
		}
	}
	return out
}

// convertDTOToolCalls maps DTO tool calls back to Ollama's api.ToolCall for assistant
// messages that contain prior tool calls in the conversation history.
//
// Arguments that are not valid JSON are logged and sent as empty arguments.
//
// Takes calls ([]llm_dto.ToolCall) which contains the DTO tool calls to convert.
//
// Returns []api.ToolCall which holds the converted Ollama calls.
func convertDTOToolCalls(ctx context.Context, calls []llm_dto.ToolCall) []api.ToolCall {
	out := make([]api.ToolCall, len(calls))
	for i, tc := range calls {
		var arguments api.ToolCallFunctionArguments
		if unmarshalError := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); unmarshalError != nil {
			_, l := logger.From(ctx, log)
			l.Warn("Failed to unmarshal tool call arguments",
				logger.String("function", tc.Function.Name),
				logger.Error(unmarshalError))
		}

		out[i] = api.ToolCall{
			ID: tc.ID,
			Function: api.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: arguments,
			},
		}
	}
	return out
}
