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

package llm_domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"piko.sh/piko/internal/llm/llm_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// DefaultBufferSize is the default number of messages to keep in the buffer.
	DefaultBufferSize = 20

	// DefaultTokenLimit is the token limit used when none is specified.
	DefaultTokenLimit = 4000

	// DefaultSummaryBufferSize is the number of messages to store before a summary
	// is created.
	DefaultSummaryBufferSize = 10

	// TokenOverheadPerMessage is the estimated token count added per message for
	// role and structure metadata.
	TokenOverheadPerMessage = 4

	// CharactersPerToken is the average number of characters per token used to
	// estimate token counts from text length.
	CharactersPerToken = 4

	// ImageTokenEstimateLowDetail is the estimated token count for low detail images.
	ImageTokenEstimateLowDetail = 85

	// errFmtLoadingConversation is the format string used when a conversation
	// cannot be loaded from the memory store.
	errFmtLoadingConversation = "loading conversation %q: %w"
)

// Summariser is a minimal interface for LLM completion used by SummaryMemory.
// It allows testing without requiring a full Service implementation.
type Summariser interface {
	// Complete sends a completion request to generate a summary.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes request (*llm_dto.CompletionRequest) which
	// contains the completion parameters.
	//
	// Returns *llm_dto.CompletionResponse containing the model's response.
	// Returns error when the request fails.
	Complete(ctx context.Context, request *llm_dto.CompletionRequest) (*llm_dto.CompletionResponse, error)
}

// MemoryStorePort defines the interface for conversation memory persistence.
// It is a driven port in the hexagonal architecture pattern.
type MemoryStorePort interface {
	// Load retrieves the conversation state for the given ID.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes conversationID (string) which identifies the conversation.
	//
	// Returns *llm_dto.ConversationState containing the conversation state.
	// Returns error when the state cannot be loaded.
	Load(ctx context.Context, conversationID string) (*llm_dto.ConversationState, error)

	// Save persists the conversation state.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes state (*llm_dto.ConversationState) which is the state to persist.
	//
	// Returns error when the state cannot be saved.
	Save(ctx context.Context, state *llm_dto.ConversationState) error

	// Delete removes the conversation state.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes conversationID (string) which identifies the conversation.
	//
	// Returns error when the state cannot be deleted.
	Delete(ctx context.Context, conversationID string) error

	// List returns conversation IDs matching the pattern.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes pattern (string) which is the pattern to match (supports * wildcard).
	//
	// Returns []string containing matching conversation IDs.
	// Returns error when the list cannot be retrieved.
	List(ctx context.Context, pattern string) ([]string, error)
}

// Memory defines the interface for conversation memory management.
type Memory interface {
	// Load retrieves the conversation state for the given ID.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes conversationID (string) which identifies the conversation.
	//
	// Returns *llm_dto.ConversationState containing the conversation state.
	// Returns error when the state cannot be loaded.
	Load(ctx context.Context, conversationID string) (*llm_dto.ConversationState, error)

	// AddMessage adds a message to the conversation and persists it.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes conversationID (string) which identifies the conversation.
	// Takes message (llm_dto.Message) which is the message to add.
	//
	// Returns error when the message cannot be added.
	AddMessage(ctx context.Context, conversationID string, message llm_dto.Message) error

	// GetMessages retrieves the messages for a conversation.
	// The messages returned may be filtered or summarised based on the memory type.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes conversationID (string) which identifies the conversation.
	//
	// Returns []llm_dto.Message containing the conversation messages.
	// Returns error when the messages cannot be retrieved.
	GetMessages(ctx context.Context, conversationID string) ([]llm_dto.Message, error)

	// Clear removes all messages from a conversation.
	//
	// Takes ctx (context.Context) which controls cancellation and timeouts.
	// Takes conversationID (string) which identifies the conversation.
	//
	// Returns error when the conversation cannot be cleared.
	Clear(ctx context.Context, conversationID string) error
}

var (
	// ErrConversationNotFound indicates the conversation was not found.
	ErrConversationNotFound = errors.New("conversation not found")
)

// BufferMemoryOption is a functional option for configuring [BufferMemory].
type BufferMemoryOption func(*BufferMemory)

// BufferMemory implements the Memory interface using a fixed-size buffer.
// It stores the last N messages and removes older ones when the buffer is full.
type BufferMemory struct {
	// store provides persistent storage for conversation state.
	store MemoryStorePort

	// maxSize is the maximum number of messages to keep; older messages are trimmed.
	maxSize int

	// mu guards concurrent access to the memory store.
	mu sync.RWMutex
}

// NewBufferMemory creates a new BufferMemory.
//
// Takes store (MemoryStorePort) which handles persistence.
// Takes opts (...BufferMemoryOption) which configure the memory behaviour.
// When no [WithBufferSize] option is provided, [DefaultBufferSize] is used.
//
// Returns *BufferMemory ready for use.
func NewBufferMemory(store MemoryStorePort, opts ...BufferMemoryOption) *BufferMemory {
	m := &BufferMemory{
		store:   store,
		maxSize: DefaultBufferSize,
		mu:      sync.RWMutex{},
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Load retrieves the conversation state.
//
// Takes conversationID (string) which identifies the conversation to load.
//
// Returns *llm_dto.ConversationState which contains the loaded conversation.
// Returns error when the conversation cannot be found or loading fails.
//
// Safe for concurrent use; protected by a read lock.
func (m *BufferMemory) Load(ctx context.Context, conversationID string) (*llm_dto.ConversationState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.Load(ctx, conversationID)
}

// AddMessage adds a message to the conversation, trimming old messages if
// necessary.
//
// Takes conversationID (string) which identifies the conversation to update.
// Takes message (llm_dto.Message) which is the message to add.
//
// Returns error when loading or saving the conversation state fails.
//
// Safe for concurrent use; protected by a mutex.
func (m *BufferMemory) AddMessage(ctx context.Context, conversationID string, message llm_dto.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.store.Load(ctx, conversationID)
	if err != nil {
		if !errors.Is(err, ErrConversationNotFound) {
			return fmt.Errorf(errFmtLoadingConversation, conversationID, err)
		}
		state = llm_dto.NewConversationState(conversationID)
	}

	state.AddMessage(message)

	if len(state.Messages) > m.maxSize {
		excess := len(state.Messages) - m.maxSize
		state.Messages = state.Messages[excess:]
	}

	return m.store.Save(ctx, state)
}

// GetMessages retrieves the messages for a conversation.
//
// Takes conversationID (string) which identifies the conversation to retrieve.
//
// Returns []llm_dto.Message which contains a copy of the stored messages.
// Returns error when the underlying store fails to load the conversation.
//
// Safe for concurrent use; protected by a read lock.
func (m *BufferMemory) GetMessages(ctx context.Context, conversationID string) ([]llm_dto.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return loadMessagesFromStore(ctx, m.store, conversationID)
}

// Clear removes all messages from a conversation.
//
// Takes conversationID (string) which identifies the conversation to clear.
//
// Returns error when the store fails to delete the conversation.
//
// Safe for concurrent use.
func (m *BufferMemory) Clear(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Delete(ctx, conversationID)
}

// WindowMemoryOption is a functional option for configuring [WindowMemory].
type WindowMemoryOption func(*WindowMemory)

// WindowMemory implements Memory using a sliding window based on token count.
// It keeps recent messages that fit within the token limit and removes older
// messages when the limit is reached.
type WindowMemory struct {
	// store provides persistent storage for conversation state.
	store MemoryStorePort

	// tokenLimit is the maximum number of tokens allowed in the conversation window.
	tokenLimit int

	// mu guards concurrent access to the conversation store.
	mu sync.RWMutex
}

// NewWindowMemory creates a new WindowMemory.
//
// Takes store (MemoryStorePort) which handles persistence.
// Takes opts (...WindowMemoryOption) which configure the memory behaviour.
// When no [WithTokenLimit] option is provided, [DefaultTokenLimit] is used.
//
// Returns *WindowMemory ready for use.
func NewWindowMemory(store MemoryStorePort, opts ...WindowMemoryOption) *WindowMemory {
	m := &WindowMemory{
		store:      store,
		tokenLimit: DefaultTokenLimit,
		mu:         sync.RWMutex{},
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

// Load retrieves the conversation state.
//
// Takes conversationID (string) which identifies the conversation to load.
//
// Returns *llm_dto.ConversationState which contains the loaded state.
// Returns error when the conversation cannot be found or loading fails.
//
// Safe for concurrent use; protected by a read lock.
func (m *WindowMemory) Load(ctx context.Context, conversationID string) (*llm_dto.ConversationState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.Load(ctx, conversationID)
}

// AddMessage adds a message to the conversation, trimming old messages if the
// token limit is exceeded.
//
// Takes conversationID (string) which identifies the conversation to update.
// Takes message (llm_dto.Message) which is the message to add.
//
// Returns error when the underlying store fails to load or save the state.
//
// Safe for concurrent use; protects access to the store with a mutex.
func (m *WindowMemory) AddMessage(ctx context.Context, conversationID string, message llm_dto.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, err := m.store.Load(ctx, conversationID)
	if err != nil {
		if !errors.Is(err, ErrConversationNotFound) {
			return fmt.Errorf(errFmtLoadingConversation, conversationID, err)
		}
		state = llm_dto.NewConversationState(conversationID)
	}

	state.AddMessage(message)

	m.trimToTokenLimit(state)

	return m.store.Save(ctx, state)
}

// GetMessages retrieves the messages for a conversation.
//
// Takes conversationID (string) which identifies the conversation to retrieve.
//
// Returns []llm_dto.Message which contains a copy of all messages in the
// conversation window.
// Returns error when the underlying store fails to load the conversation.
//
// Safe for concurrent use; protected by a read lock.
func (m *WindowMemory) GetMessages(ctx context.Context, conversationID string) ([]llm_dto.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return loadMessagesFromStore(ctx, m.store, conversationID)
}

// Clear removes all messages from a conversation.
//
// Takes conversationID (string) which identifies the conversation to clear.
//
// Returns error when the underlying store fails to delete the conversation.
//
// Safe for concurrent use.
func (m *WindowMemory) Clear(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Delete(ctx, conversationID)
}

// trimToTokenLimit removes old messages until the token count is within the
// limit.
//
// Takes state (*llm_dto.ConversationState) which holds the messages to trim.
func (m *WindowMemory) trimToTokenLimit(state *llm_dto.ConversationState) {
	tokenCount := m.estimateTokens(state.Messages)
	for tokenCount > m.tokenLimit && len(state.Messages) > 1 {
		state.Messages = state.Messages[1:]
		tokenCount = m.estimateTokens(state.Messages)
	}
	state.TokenCount = tokenCount
}

// estimateTokens provides a rough estimate of token count for messages.
// This uses a simple heuristic of approximately CharactersPerToken characters
// per token.
//
// Takes messages ([]llm_dto.Message) which contains the messages to estimate.
//
// Returns int which is the estimated token count.
func (*WindowMemory) estimateTokens(messages []llm_dto.Message) int {
	return estimateMessageTokens(messages)
}

// SummaryMemory implements the Memory interface using LLM summarisation.
// It keeps recent messages in a buffer and summarises older messages when
// the buffer grows too large.
type SummaryMemory struct {
	// store holds conversation state via the MemoryStorePort interface.
	store MemoryStorePort

	// summariser generates summaries of conversation history.
	summariser Summariser

	// config holds the memory settings including buffer size, summary prompt,
	// and model for generating conversation summaries.
	config llm_dto.MemoryConfig

	// mu guards concurrent access to the underlying store.
	mu sync.RWMutex
}

// NewSummaryMemory creates a new SummaryMemory.
//
// Takes store (MemoryStorePort) which handles persistence.
// Takes summariser (Summariser) which provides LLM for summarisation.
// Takes config (llm_dto.MemoryConfig) which configures the memory behaviour.
//
// Returns *SummaryMemory ready for use.
func NewSummaryMemory(store MemoryStorePort, summariser Summariser, config llm_dto.MemoryConfig) *SummaryMemory {
	if config.BufferSize <= 0 {
		config.BufferSize = DefaultSummaryBufferSize
	}
	if config.SummaryPrompt == "" {
		config.SummaryPrompt = "Summarise the following conversation concisely, keeping key facts and context:\n\n{{.Messages}}\n\nSummary:"
	}
	return &SummaryMemory{
		store:      store,
		summariser: summariser,
		config:     config,
		mu:         sync.RWMutex{},
	}
}

// Load retrieves the conversation state.
//
// Takes conversationID (string) which identifies the conversation to load.
//
// Returns *llm_dto.ConversationState which contains the loaded conversation.
// Returns error when the conversation cannot be found or loading fails.
//
// Safe for concurrent use; protected by a read lock.
func (m *SummaryMemory) Load(ctx context.Context, conversationID string) (*llm_dto.ConversationState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.store.Load(ctx, conversationID)
}

// AddMessage adds a message to the conversation, potentially triggering
// summarisation.
//
// Takes conversationID (string) which identifies the conversation to update.
// Takes message (llm_dto.Message) which is the message to add.
//
// Returns error when the conversation cannot be loaded or saved.
//
// Safe for concurrent use.
func (m *SummaryMemory) AddMessage(ctx context.Context, conversationID string, message llm_dto.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, l := logger_domain.From(ctx, log)

	state, err := m.store.Load(ctx, conversationID)
	if err != nil {
		if !errors.Is(err, ErrConversationNotFound) {
			return fmt.Errorf(errFmtLoadingConversation, conversationID, err)
		}
		state = llm_dto.NewConversationState(conversationID)
	}

	state.AddMessage(message)

	if len(state.Messages) > m.config.BufferSize*2 {
		err = m.summariseOldMessages(ctx, state)
		if err != nil {
			l.Warn("Failed to summarise conversation history",
				logger_domain.Error(err),
			)
		}
	}

	return m.store.Save(ctx, state)
}

// GetMessages retrieves the messages for a conversation.
// If a summary exists, it is prepended as a system message.
//
// Takes conversationID (string) which identifies the conversation to retrieve.
//
// Returns []llm_dto.Message which contains the conversation messages.
// Returns error when the store fails to load the conversation state.
//
// Safe for concurrent use; holds a read lock during execution.
func (m *SummaryMemory) GetMessages(ctx context.Context, conversationID string) ([]llm_dto.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, err := m.store.Load(ctx, conversationID)
	if err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			return []llm_dto.Message{}, nil
		}
		return nil, fmt.Errorf(errFmtLoadingConversation, conversationID, err)
	}

	if state.HasSummary() {
		messages := make([]llm_dto.Message, 0, len(state.Messages)+1)
		messages = append(messages, llm_dto.NewSystemMessage("Previous conversation summary: "+*state.Summary))
		messages = append(messages, state.Messages...)
		return messages, nil
	}

	messages := make([]llm_dto.Message, len(state.Messages))
	copy(messages, state.Messages)
	return messages, nil
}

// Clear removes all messages from a conversation.
//
// Takes conversationID (string) which identifies the conversation to clear.
//
// Returns error when the store fails to delete the conversation.
//
// Safe for concurrent use.
func (m *SummaryMemory) Clear(ctx context.Context, conversationID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.store.Delete(ctx, conversationID)
}

// summariseOldMessages summarises older messages and keeps recent ones.
//
// Takes state (*llm_dto.ConversationState) which contains the messages to
// process.
//
// Returns error when the LLM summarisation request fails.
func (m *SummaryMemory) summariseOldMessages(ctx context.Context, state *llm_dto.ConversationState) error {
	if len(state.Messages) <= m.config.BufferSize {
		return nil
	}

	splitPoint := len(state.Messages) - m.config.BufferSize
	toSummarise := state.Messages[:splitPoint]
	toKeep := state.Messages[splitPoint:]

	var builder strings.Builder
	if state.HasSummary() {
		builder.WriteString("Previous summary: ")
		builder.WriteString(*state.Summary)
		builder.WriteString("\n\n")
	}
	for _, message := range toSummarise {
		builder.WriteString(string(message.Role))
		builder.WriteString(": ")
		builder.WriteString(message.Content)
		builder.WriteString("\n")
	}

	prompt := strings.Replace(m.config.SummaryPrompt, "{{.Messages}}", builder.String(), 1)
	response, err := m.summariser.Complete(ctx, &llm_dto.CompletionRequest{
		Model: m.config.SummaryModel,
		Messages: []llm_dto.Message{
			llm_dto.NewUserMessage(prompt),
		},
		Temperature:       nil,
		MaxTokens:         nil,
		TopP:              nil,
		FrequencyPenalty:  nil,
		PresencePenalty:   nil,
		Stop:              nil,
		Seed:              nil,
		Tools:             nil,
		ToolChoice:        nil,
		ParallelToolCalls: nil,
		ResponseFormat:    nil,
		Stream:            false,
		StreamOptions:     nil,
		ProviderOptions:   nil,
		Metadata:          nil,
	})
	if err != nil {
		return fmt.Errorf("summarising conversation history: %w", err)
	}

	state.Summary = new(response.Content())
	state.Messages = toKeep

	return nil
}

// WithBufferSize sets the maximum number of messages to keep. When omitted,
// [DefaultBufferSize] (20) is used.
//
// Takes size (int) which specifies the maximum buffer capacity.
//
// Returns BufferMemoryOption which configures the buffer size.
func WithBufferSize(size int) BufferMemoryOption {
	return func(m *BufferMemory) {
		if size > 0 {
			m.maxSize = size
		}
	}
}

// WithTokenLimit sets the maximum number of tokens to keep in the window.
//
// When omitted, [DefaultTokenLimit] (4000) is used.
//
// Takes limit (int) which specifies the maximum number of tokens to retain.
//
// Returns WindowMemoryOption which configures the token limit on the window.
func WithTokenLimit(limit int) WindowMemoryOption {
	return func(m *WindowMemory) {
		if limit > 0 {
			m.tokenLimit = limit
		}
	}
}

// loadMessagesFromStore retrieves messages for a conversation from the store.
//
// Takes store (MemoryStorePort) which provides access to conversation state.
// Takes conversationID (string) which identifies the conversation to load.
//
// Returns []llm_dto.Message which contains a copy of the stored messages.
// Returns error when the store fails to load the conversation.
func loadMessagesFromStore(ctx context.Context, store MemoryStorePort, conversationID string) ([]llm_dto.Message, error) {
	state, err := store.Load(ctx, conversationID)
	if err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			return []llm_dto.Message{}, nil
		}
		return nil, fmt.Errorf(errFmtLoadingConversation, conversationID, err)
	}

	messages := make([]llm_dto.Message, len(state.Messages))
	copy(messages, state.Messages)
	return messages, nil
}

// estimateMessageTokens provides a rough estimate of token count for messages.
// This is a package-level helper to reduce receiver usage.
//
// Takes messages ([]llm_dto.Message) which contains the messages to estimate.
//
// Returns int which is the total estimated token count for all messages.
func estimateMessageTokens(messages []llm_dto.Message) int {
	total := 0
	for _, message := range messages {
		total += estimateSingleMessageTokens(message)
	}
	return total
}

// estimateSingleMessageTokens estimates tokens for a single message.
//
// Takes message (llm_dto.Message) which is the message to estimate tokens for.
//
// Returns int which is the estimated token count for the message.
func estimateSingleMessageTokens(message llm_dto.Message) int {
	total := TokenOverheadPerMessage

	if len(message.ContentParts) > 0 {
		for _, part := range message.ContentParts {
			if part.Text != nil {
				total += len(*part.Text) / CharactersPerToken
			}
			if part.ImageURL != nil || part.ImageData != nil {
				total += ImageTokenEstimateLowDetail
			}
		}
	} else {
		total += len(message.Content) / CharactersPerToken
	}

	for _, tc := range message.ToolCalls {
		total += len(tc.Function.Name)/CharactersPerToken + len(tc.Function.Arguments)/CharactersPerToken
	}

	return total
}
