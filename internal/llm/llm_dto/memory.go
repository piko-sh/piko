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

package llm_dto

import "time"

// MemoryType identifies the strategy for managing conversation memory.
type MemoryType string

const (
	// DefaultBufferSize is the default number of messages to keep in buffer
	// memory.
	DefaultBufferSize = 20

	// DefaultWindowTokenLimit is the default token limit for window memory.
	DefaultWindowTokenLimit = 4000

	// DefaultSummaryBufferSize is the default number of messages to keep before
	// summarising.
	DefaultSummaryBufferSize = 10

	// MemoryTypeBuffer stores the most recent N messages in memory.
	MemoryTypeBuffer MemoryType = "buffer"

	// MemoryTypeSummary uses LLM summarisation to compress older messages.
	MemoryTypeSummary MemoryType = "summary"

	// MemoryTypeWindow keeps messages within a token limit.
	MemoryTypeWindow MemoryType = "window"

	// defaultSummaryPrompt is the default prompt used for summarising conversation
	// history.
	defaultSummaryPrompt = `Summarise the following conversation history concisely.
Focus on key facts, decisions, and context that would be important for continuing the conversation.
Keep the summary brief but comprehensive.
Conversation:
{{.Messages}}
Summary:`
)

// MemoryConfig configures conversation memory behaviour.
type MemoryConfig struct {
	// Type specifies the memory strategy to use.
	Type MemoryType

	// SummaryModel is the model to use for summarisation; only used with Summary
	// type.
	SummaryModel string

	// SummaryPrompt is the prompt template for summarisation (for Summary type).
	// If empty, a default prompt will be used.
	SummaryPrompt string

	// BufferSize is the maximum number of messages to keep before summarisation.
	BufferSize int

	// TokenLimit is the maximum number of tokens to keep for Window type.
	TokenLimit int
}

// ConversationState represents the current state of a conversation's memory.
type ConversationState struct {
	// CreatedAt is when the conversation was first created.
	CreatedAt time.Time

	// UpdatedAt is when the conversation was last changed.
	UpdatedAt time.Time

	// Summary holds a condensed version of older messages when the token count
	// grows too large; nil means no summary exists.
	Summary *string

	// ID is the unique identifier for this conversation.
	ID string

	// Messages holds the list of messages in this conversation.
	Messages []Message

	// TokenCount is the estimated number of tokens in the current messages.
	TokenCount int
}

// NewConversationState creates a new ConversationState with the given ID.
//
// Takes id (string) which is the unique identifier for the conversation.
//
// Returns *ConversationState initialised with empty messages.
func NewConversationState(id string) *ConversationState {
	now := time.Now()
	return &ConversationState{
		ID:        id,
		Messages:  make([]Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage appends a message to the conversation and updates the timestamp.
//
// Takes message (Message) which is the message to append.
func (s *ConversationState) AddMessage(message Message) {
	s.Messages = append(s.Messages, message)
	s.UpdatedAt = time.Now()
}

// MessageCount returns the number of messages in the conversation.
//
// Returns int which is the message count.
func (s *ConversationState) MessageCount() int {
	return len(s.Messages)
}

// HasSummary reports whether the conversation has a summary.
//
// Returns bool which is true if a summary exists.
func (s *ConversationState) HasSummary() bool {
	return s.Summary != nil && *s.Summary != ""
}

// Clear removes all messages and the summary from the conversation state.
func (s *ConversationState) Clear() {
	s.Messages = make([]Message, 0)
	s.Summary = nil
	s.TokenCount = 0
	s.UpdatedAt = time.Now()
}

// DefaultBufferMemoryConfig returns a MemoryConfig with sensible defaults
// for buffer memory.
//
// Returns *MemoryConfig which has BufferSize set to 20 messages.
func DefaultBufferMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		Type:       MemoryTypeBuffer,
		BufferSize: DefaultBufferSize,
	}
}

// DefaultWindowMemoryConfig returns a MemoryConfig with sensible defaults
// for window memory.
//
// Returns *MemoryConfig which has a TokenLimit of 4000 tokens.
func DefaultWindowMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		Type:       MemoryTypeWindow,
		TokenLimit: DefaultWindowTokenLimit,
	}
}

// DefaultSummaryMemoryConfig returns a MemoryConfig with sensible defaults
// for summary memory.
//
// Takes model (string) which is the model to use for summarisation.
//
// Returns *MemoryConfig which is configured for summary memory.
func DefaultSummaryMemoryConfig(model string) *MemoryConfig {
	return &MemoryConfig{
		Type:          MemoryTypeSummary,
		SummaryModel:  model,
		BufferSize:    DefaultSummaryBufferSize,
		SummaryPrompt: defaultSummaryPrompt,
	}
}
