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

package provider_mock

import (
	"context"
	"sync"
	"time"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/wdk/clock"
)

const (
	// DefaultMockPromptTokens is the default number of prompt tokens in mock
	// responses.
	DefaultMockPromptTokens = 10

	// DefaultMockCompletionTokens is the default number of completion tokens in
	// mock responses.
	DefaultMockCompletionTokens = 5

	// DefaultMockTotalTokens is the default total token count for mock responses.
	DefaultMockTotalTokens = 15

	// DefaultMockContextWindow is the default context window size for mock models.
	DefaultMockContextWindow = 8192

	// DefaultMockMaxOutputTokens is the default maximum output tokens for mock
	// models.
	DefaultMockMaxOutputTokens = 4096
)

// MockProvider is a test double that implements LLMProviderPort. It allows
// tests to set responses, errors, and check which methods were called.
type MockProvider struct {
	// clock provides time operations for timestamps and delays.
	clock clock.Clock

	// err is the error to return from Complete calls; nil means success.
	err error

	// streamErr is the error to return during streaming operations.
	streamErr error

	// modelsErr is the error to return from ListModels; nil means success.
	modelsErr error

	// response holds the canned completion response returned by Complete.
	response *llm_dto.CompletionResponse

	// defaultModel is the model name returned by DefaultModel.
	defaultModel string

	// streamCalls records all streaming completion requests for test verification.
	streamCalls []llm_dto.CompletionRequest

	// completeCalls stores all Complete requests for test verification.
	completeCalls []llm_dto.CompletionRequest

	// models stores model information returned by ListModels.
	models []llm_dto.ModelInfo

	// streamChunks holds the chunks to return during streaming responses.
	streamChunks []llm_dto.StreamChunk

	// responseDelay is the time to wait before returning responses; 0 means no
	// delay.
	responseDelay time.Duration

	// mu guards concurrent access to the provider's mutable state.
	mu sync.RWMutex

	// supportsStreaming indicates whether the mock provider reports streaming
	// capability in model listings.
	supportsStreaming bool

	// supportsStructuredOutput indicates whether the mock provider supports
	// structured output in responses.
	supportsStructuredOutput bool

	// supportsTools indicates whether the mock provider supports tool calling.
	supportsTools bool

	// supportsPenalties indicates whether the mock provider supports frequency
	// and presence penalties.
	supportsPenalties bool

	// supportsSeed indicates whether the mock provider supports deterministic
	// seed.
	supportsSeed bool

	// supportsParallelToolCalls indicates whether the mock provider supports
	// parallel tool calls.
	supportsParallelToolCalls bool

	// supportsMessageName indicates whether the mock provider supports the
	// name field on messages.
	supportsMessageName bool

	// closed indicates whether the provider has been shut down.
	closed bool
}

var _ llm_domain.LLMProviderPort = (*MockProvider)(nil)

// MockProviderOption is a function that sets up a MockProvider.
type MockProviderOption func(*MockProvider)

// SetResponse configures the response to return from Complete.
//
// Takes response (*llm_dto.CompletionResponse) which will be returned.
//
// Safe for concurrent use.
func (m *MockProvider) SetResponse(response *llm_dto.CompletionResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.response = response
}

// SetError configures an error to return from Complete.
//
// Takes err (error) which will be returned instead of a response.
//
// Safe for concurrent use.
func (m *MockProvider) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// SetStreamChunks configures the chunks to emit during streaming.
//
// Takes chunks ([]llm_dto.StreamChunk) which will be emitted in order.
//
// Safe for concurrent use; protected by mutex.
func (m *MockProvider) SetStreamChunks(chunks []llm_dto.StreamChunk) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamChunks = chunks
}

// SetStreamError configures an error to return from Stream.
//
// Takes err (error) which will cause the stream to emit an error event.
//
// Safe for concurrent use.
func (m *MockProvider) SetStreamError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamErr = err
}

// SetModels configures the models returned by ListModels.
//
// Takes models ([]llm_dto.ModelInfo) which will be returned.
//
// Safe for concurrent use.
func (m *MockProvider) SetModels(models []llm_dto.ModelInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.models = models
}

// SetModelsError configures an error to return from ListModels.
//
// Takes err (error) which will be returned.
//
// Safe for concurrent use.
func (m *MockProvider) SetModelsError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modelsErr = err
}

// SetResponseDelay configures a delay before returning responses.
//
// Takes delay (time.Duration) which is the delay to add.
//
// Safe for concurrent use.
func (m *MockProvider) SetResponseDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responseDelay = delay
}

// SetSupportsStreaming configures the streaming capability flag.
//
// Takes supported (bool) which is whether streaming is supported.
//
// Safe for concurrent use.
func (m *MockProvider) SetSupportsStreaming(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsStreaming = supported
}

// SetSupportsStructuredOutput configures the structured output capability flag.
//
// Takes supported (bool) which is whether structured output is supported.
//
// Safe for concurrent use.
func (m *MockProvider) SetSupportsStructuredOutput(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsStructuredOutput = supported
}

// SetSupportsTools configures the tools capability flag.
//
// Takes supported (bool) which is whether tools are supported.
//
// Safe for concurrent use.
func (m *MockProvider) SetSupportsTools(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsTools = supported
}

// SetSupportsPenalties configures the penalties capability flag.
//
// Takes supported (bool) which is whether penalties are supported.
//
// Safe for concurrent use.
func (m *MockProvider) SetSupportsPenalties(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsPenalties = supported
}

// SetSupportsSeed configures the seed capability flag.
//
// Takes supported (bool) which is whether seed is supported.
//
// Safe for concurrent use.
func (m *MockProvider) SetSupportsSeed(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsSeed = supported
}

// SetSupportsParallelToolCalls configures the parallel tool calls capability
// flag.
//
// Takes supported (bool) which is whether parallel tool calls are supported.
//
// Safe for concurrent use.
func (m *MockProvider) SetSupportsParallelToolCalls(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsParallelToolCalls = supported
}

// SetSupportsMessageName configures the message name capability flag.
//
// Takes supported (bool) which is whether message names are supported.
//
// Safe for concurrent use.
func (m *MockProvider) SetSupportsMessageName(supported bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.supportsMessageName = supported
}

// GetCompleteCalls returns all requests passed to Complete.
//
// Returns []llm_dto.CompletionRequest containing recorded calls.
//
// Safe for concurrent use. Returns a copy to prevent data races.
func (m *MockProvider) GetCompleteCalls() []llm_dto.CompletionRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]llm_dto.CompletionRequest, len(m.completeCalls))
	copy(calls, m.completeCalls)
	return calls
}

// GetStreamCalls returns all requests passed to Stream.
//
// Returns []llm_dto.CompletionRequest containing recorded calls.
//
// Safe for concurrent use.
func (m *MockProvider) GetStreamCalls() []llm_dto.CompletionRequest {
	m.mu.RLock()
	defer m.mu.RUnlock()
	calls := make([]llm_dto.CompletionRequest, len(m.streamCalls))
	copy(calls, m.streamCalls)
	return calls
}

// WasClosed reports whether Close was called.
//
// Returns bool which is true if Close was called.
//
// Safe for concurrent use.
func (m *MockProvider) WasClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

// Reset clears all recorded calls and resets to defaults.
//
// Safe for concurrent use.
func (m *MockProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.completeCalls = nil
	m.streamCalls = nil
	m.err = nil
	m.streamErr = nil
	m.closed = false
}

// Complete implements LLMProviderPort.Complete.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// parameters.
//
// Returns *llm_dto.CompletionResponse which contains the mock response.
// Returns error when context is cancelled or a configured error is set.
//
// Safe for concurrent use. Uses a mutex to protect access to the mock state.
func (m *MockProvider) Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error) {
	completeCount.Add(ctx, 1)

	m.mu.Lock()
	m.completeCalls = append(m.completeCalls, *request)
	response := m.response
	err := m.err
	delay := m.responseDelay
	m.mu.Unlock()

	if delay > 0 {
		timer := m.clock.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-timer.C():
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if err != nil {
		return nil, err
	}

	if response != nil {
		return new(*response), nil
	}

	return nil, nil
}

// streamState holds the state needed for stream goroutine execution.
type streamState struct {
	// streamErr is an error to send before closing the stream; nil means no error.
	streamErr error

	// clk provides time operations for scheduling delays.
	clk clock.Clock

	// chunks holds the content fragments to emit during streaming.
	chunks []llm_dto.StreamChunk

	// delay is the time to wait between stream events.
	delay time.Duration
}

// Stream implements LLMProviderPort.Stream.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the completion
// request to stream.
//
// Returns <-chan llm_dto.StreamEvent which yields stream events as they
// become available.
// Returns error when the stream cannot be started.
//
// Spawns a goroutine to run the stream loop until the context is cancelled.
func (m *MockProvider) Stream(ctx context.Context, request *llm_dto.CompletionRequest) (<-chan llm_dto.StreamEvent, error) {
	streamCount.Add(ctx, 1)

	state := m.captureStreamState(request)
	events := make(chan llm_dto.StreamEvent)

	go m.runStreamLoop(ctx, events, state)

	return events, nil
}

// SupportsStreaming implements LLMProviderPort.SupportsStreaming.
//
// Returns bool which indicates whether the provider supports streaming.
//
// Safe for concurrent use.
func (m *MockProvider) SupportsStreaming() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsStreaming
}

// SupportsStructuredOutput reports whether the provider supports structured
// output. Implements LLMProviderPort.SupportsStructuredOutput.
//
// Returns bool which is true if structured output is supported.
//
// Safe for concurrent use.
func (m *MockProvider) SupportsStructuredOutput() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsStructuredOutput
}

// SupportsTools implements LLMProviderPort.SupportsTools.
//
// Returns bool which indicates whether the provider supports tool calling.
//
// Safe for concurrent use.
func (m *MockProvider) SupportsTools() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsTools
}

// SupportsPenalties implements LLMProviderPort.SupportsPenalties.
//
// Returns bool which indicates whether the provider supports penalties.
//
// Safe for concurrent use.
func (m *MockProvider) SupportsPenalties() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsPenalties
}

// SupportsSeed implements LLMProviderPort.SupportsSeed.
//
// Returns bool which indicates whether the provider supports seed.
//
// Safe for concurrent use.
func (m *MockProvider) SupportsSeed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsSeed
}

// SupportsParallelToolCalls implements LLMProviderPort.SupportsParallelToolCalls.
//
// Returns bool which indicates whether the provider supports parallel tool
// calls.
//
// Safe for concurrent use.
func (m *MockProvider) SupportsParallelToolCalls() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsParallelToolCalls
}

// SupportsMessageName implements LLMProviderPort.SupportsMessageName.
//
// Returns bool which indicates whether the provider supports message names.
//
// Safe for concurrent use.
func (m *MockProvider) SupportsMessageName() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.supportsMessageName
}

// ListModels implements LLMProviderPort.ListModels.
//
// Returns []llm_dto.ModelInfo which contains the configured mock models or
// defaults.
// Returns error when a mock error has been set.
//
// Safe for concurrent use. Uses a read lock to protect access to mock state.
func (m *MockProvider) ListModels(_ context.Context) ([]llm_dto.ModelInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.modelsErr != nil {
		return nil, m.modelsErr
	}

	if m.models != nil {
		models := make([]llm_dto.ModelInfo, len(m.models))
		copy(models, m.models)
		return models, nil
	}

	return []llm_dto.ModelInfo{
		{
			ID:                       "mock-model",
			Name:                     "Mock Model",
			Provider:                 "mock",
			ContextWindow:            DefaultMockContextWindow,
			MaxOutputTokens:          DefaultMockMaxOutputTokens,
			SupportsStreaming:        m.supportsStreaming,
			SupportsTools:            m.supportsTools,
			SupportsStructuredOutput: m.supportsStructuredOutput,
		},
	}, nil
}

// Close implements LLMProviderPort.Close.
//
// Returns error when the provider cannot be closed.
//
// Safe for concurrent use.
func (m *MockProvider) Close(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// DefaultModel implements LLMProviderPort.DefaultModel.
//
// Returns string which is the default model name.
//
// Safe for concurrent use.
func (m *MockProvider) DefaultModel() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultModel
}

// captureStreamState captures the current provider state for stream execution.
//
// Takes request (*llm_dto.CompletionRequest) which specifies the
// completion request to record.
//
// Returns streamState which contains the captured chunks, error, delay, and
// clock for stream processing.
//
// Safe for concurrent use; protects access with a mutex lock.
func (m *MockProvider) captureStreamState(request *llm_dto.CompletionRequest) streamState {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamCalls = append(m.streamCalls, *request)
	return streamState{
		chunks:    m.streamChunks,
		streamErr: m.streamErr,
		delay:     m.responseDelay,
		clk:       m.clock,
	}
}

// runStreamLoop executes the main stream loop, emitting events until
// completion.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream events.
// Takes state (streamState) which holds the current streaming state.
func (m *MockProvider) runStreamLoop(ctx context.Context, events chan<- llm_dto.StreamEvent, state streamState) {
	defer close(events)

	if cancelled := m.handleStreamDelay(ctx, events, state); cancelled {
		return
	}

	if state.streamErr != nil {
		events <- llm_dto.NewErrorEvent(state.streamErr)
		return
	}

	if cancelled := m.emitStreamChunks(ctx, events, state.chunks); cancelled {
		return
	}

	m.emitDoneEvent(events)
}

// handleStreamDelay waits for the configured delay before continuing.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives error events on
// cancellation.
// Takes state (streamState) which provides the delay duration and clock.
//
// Returns bool which is true if the context was cancelled, false otherwise.
func (*MockProvider) handleStreamDelay(ctx context.Context, events chan<- llm_dto.StreamEvent, state streamState) bool {
	if state.delay <= 0 {
		return false
	}

	timer := state.clk.NewTimer(state.delay)
	select {
	case <-timer.C():
		timer.Stop()
		return false
	case <-ctx.Done():
		timer.Stop()
		events <- llm_dto.NewErrorEvent(ctx.Err())
		return true
	}
}

// emitStreamChunks emits all configured chunks or a default chunk.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream events.
// Takes chunks ([]llm_dto.StreamChunk) which contains the chunks to emit.
//
// Returns bool which indicates whether the context was cancelled.
func (m *MockProvider) emitStreamChunks(ctx context.Context, events chan<- llm_dto.StreamEvent, chunks []llm_dto.StreamChunk) bool {
	if len(chunks) > 0 {
		return m.emitConfiguredChunks(ctx, events, chunks)
	}
	m.emitDefaultChunk(events)
	return false
}

// emitConfiguredChunks emits user-configured chunks to the events channel.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the stream events.
// Takes chunks ([]llm_dto.StreamChunk) which contains the chunks to emit.
//
// Returns bool which is true if the context was cancelled during emission.
func (*MockProvider) emitConfiguredChunks(ctx context.Context, events chan<- llm_dto.StreamEvent, chunks []llm_dto.StreamChunk) bool {
	for _, chunk := range chunks {
		select {
		case events <- llm_dto.NewChunkEvent(&chunk):
		case <-ctx.Done():
			events <- llm_dto.NewErrorEvent(ctx.Err())
			return true
		}
	}
	return false
}

// emitDefaultChunk emits a single chunk from the mock response.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the emitted chunk.
//
// Safe for concurrent use; uses a read lock to access the mock response.
func (m *MockProvider) emitDefaultChunk(events chan<- llm_dto.StreamEvent) {
	m.mu.RLock()
	response := m.response
	m.mu.RUnlock()

	if response == nil || len(response.Choices) == 0 {
		return
	}

	events <- llm_dto.NewChunkEvent(&llm_dto.StreamChunk{
		ID:    response.ID,
		Model: response.Model,
		Delta: &llm_dto.MessageDelta{
			Role:    new(response.Choices[0].Message.Role),
			Content: new(response.Choices[0].Message.Content),
		},
	})
}

// emitDoneEvent sends the final done event with usage information.
//
// Takes events (chan<- llm_dto.StreamEvent) which receives the done event.
//
// Safe for concurrent use. Uses a read lock to access the response safely.
func (m *MockProvider) emitDoneEvent(events chan<- llm_dto.StreamEvent) {
	m.mu.RLock()
	response := m.response
	m.mu.RUnlock()
	events <- llm_dto.NewDoneEvent(response)
}

// WithMockClock sets the clock used for time operations.
// If not set, clock.RealClock() is used.
//
// Takes c (clock.Clock) which provides time operations.
//
// Returns MockProviderOption which applies the clock setting to the provider.
func WithMockClock(c clock.Clock) MockProviderOption {
	return func(m *MockProvider) {
		m.clock = c
	}
}

// New creates a new MockProvider with default settings.
// By default, all capabilities are enabled and a simple response is set.
//
// Takes opts (...MockProviderOption) which are optional functions to change
// the default settings.
//
// Returns *MockProvider which is ready for use in tests.
func New(opts ...MockProviderOption) *MockProvider {
	m := &MockProvider{
		clock:                    clock.RealClock(),
		defaultModel:             "mock-model",
		supportsStreaming:        true,
		supportsStructuredOutput: true,
		supportsTools:            true,
	}
	for _, opt := range opts {
		opt(m)
	}
	m.response = &llm_dto.CompletionResponse{
		ID:      "mock-completion-id",
		Model:   "mock-model",
		Created: m.clock.Now().Unix(),
		Choices: []llm_dto.Choice{
			{
				Index: 0,
				Message: llm_dto.Message{
					Role:    llm_dto.RoleAssistant,
					Content: "This is a mock response.",
				},
				FinishReason: llm_dto.FinishReasonStop,
			},
		},
		Usage: &llm_dto.Usage{
			PromptTokens:     DefaultMockPromptTokens,
			CompletionTokens: DefaultMockCompletionTokens,
			TotalTokens:      DefaultMockTotalTokens,
		},
	}
	return m
}
