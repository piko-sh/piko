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

package memory_memory

import (
	"context"
	"strings"
	"sync"

	"piko.sh/piko/internal/llm/llm_domain"
	"piko.sh/piko/internal/llm/llm_dto"
)

// Store is an in-memory implementation of MemoryStorePort.
// It stores conversations in memory and is suitable for development,
// testing, and single-instance deployments.
type Store struct {
	// conversations maps conversation IDs to their state.
	conversations map[string]*llm_dto.ConversationState

	// mu guards access to all mutable fields in Store.
	mu sync.RWMutex
}

// Load retrieves the conversation state for the given ID.
//
// Takes conversationID (string) which identifies the conversation.
//
// Returns *llm_dto.ConversationState which contains a deep copy of the
// conversation state.
// Returns error when the conversation is not found.
//
// Safe for concurrent use; protected by a read lock.
func (s *Store) Load(_ context.Context, conversationID string) (*llm_dto.ConversationState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.conversations[conversationID]
	if !ok {
		return nil, llm_domain.ErrConversationNotFound
	}

	return copyState(state), nil
}

// Save persists the conversation state.
//
// Takes state (*llm_dto.ConversationState) which is the state to persist.
//
// Returns error when the state cannot be saved.
//
// Safe for concurrent use. Holds a mutex while storing.
func (s *Store) Save(_ context.Context, state *llm_dto.ConversationState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conversations[state.ID] = copyState(state)
	return nil
}

// Delete removes the conversation state.
//
// Takes conversationID (string) which identifies the conversation.
//
// Returns error when the state cannot be deleted.
//
// Safe for concurrent use.
func (s *Store) Delete(_ context.Context, conversationID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.conversations, conversationID)
	return nil
}

// List returns conversation IDs matching the pattern.
// The pattern supports * as a wildcard that matches any characters.
//
// Takes pattern (string) which is the pattern to match.
//
// Returns []string which contains matching conversation IDs.
// Returns error when the list cannot be retrieved.
//
// Safe for concurrent use.
func (s *Store) List(_ context.Context, pattern string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var ids []string
	for id := range s.conversations {
		if matchPattern(pattern, id) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// Size returns the number of conversations stored.
//
// Returns int which is the number of conversations.
//
// Safe for concurrent use.
func (s *Store) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.conversations)
}

// Clear removes all conversations from the store.
//
// Safe for concurrent use.
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conversations = make(map[string]*llm_dto.ConversationState)
}

// New creates a new in-memory conversation store.
//
// Returns *Store which is ready for use.
func New() *Store {
	return &Store{
		conversations: make(map[string]*llm_dto.ConversationState),
	}
}

// copyState creates a deep copy of a ConversationState.
//
// Takes state (*llm_dto.ConversationState) which is the state to copy.
//
// Returns *llm_dto.ConversationState which is a new independent copy of the
// input, or nil if the input is nil.
func copyState(state *llm_dto.ConversationState) *llm_dto.ConversationState {
	if state == nil {
		return nil
	}

	copied := &llm_dto.ConversationState{
		ID:         state.ID,
		TokenCount: state.TokenCount,
		CreatedAt:  state.CreatedAt,
		UpdatedAt:  state.UpdatedAt,
	}

	if state.Summary != nil {
		copied.Summary = new(*state.Summary)
	}

	if state.Messages != nil {
		copied.Messages = make([]llm_dto.Message, len(state.Messages))
		for i, message := range state.Messages {
			copied.Messages[i] = copyMessage(message)
		}
	}

	return copied
}

// copyMessage creates a deep copy of a Message.
//
// Takes message (llm_dto.Message) which is the message to copy.
//
// Returns llm_dto.Message which is a new message with all fields copied.
func copyMessage(message llm_dto.Message) llm_dto.Message {
	copied := llm_dto.Message{
		Role:    message.Role,
		Content: message.Content,
	}

	if message.Name != nil {
		copied.Name = new(*message.Name)
	}

	if message.ToolCallID != nil {
		copied.ToolCallID = new(*message.ToolCallID)
	}

	if message.ContentParts != nil {
		copied.ContentParts = make([]llm_dto.ContentPart, len(message.ContentParts))
		for i, part := range message.ContentParts {
			copied.ContentParts[i] = copyContentPart(part)
		}
	}

	if message.ToolCalls != nil {
		copied.ToolCalls = make([]llm_dto.ToolCall, len(message.ToolCalls))
		for i, tc := range message.ToolCalls {
			copied.ToolCalls[i] = copyToolCall(tc)
		}
	}

	return copied
}

// copyContentPart creates a deep copy of a ContentPart.
//
// Takes part (llm_dto.ContentPart) which is the content part to copy.
//
// Returns llm_dto.ContentPart which is an independent copy of the input.
func copyContentPart(part llm_dto.ContentPart) llm_dto.ContentPart {
	copied := llm_dto.ContentPart{
		Type: part.Type,
	}

	if part.Text != nil {
		copied.Text = new(*part.Text)
	}

	if part.ImageURL != nil {
		copied.ImageURL = &llm_dto.ImageURL{
			URL: part.ImageURL.URL,
		}
		if part.ImageURL.Detail != nil {
			copied.ImageURL.Detail = new(*part.ImageURL.Detail)
		}
	}

	if part.ImageData != nil {
		copied.ImageData = &llm_dto.ImageData{
			MIMEType: part.ImageData.MIMEType,
			Data:     part.ImageData.Data,
		}
	}

	return copied
}

// copyToolCall creates a deep copy of a ToolCall.
//
// Takes tc (llm_dto.ToolCall) which is the tool call to copy.
//
// Returns llm_dto.ToolCall which is a new instance with copied values.
func copyToolCall(tc llm_dto.ToolCall) llm_dto.ToolCall {
	return llm_dto.ToolCall{
		ID:   tc.ID,
		Type: tc.Type,
		Function: llm_dto.FunctionCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		},
	}
}

// matchPattern checks if a string matches a pattern with * wildcards.
//
// Takes pattern (string) which specifies the pattern to match against.
// Takes s (string) which is the string to test.
//
// Returns bool which is true if the string matches the pattern.
func matchPattern(pattern, s string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return pattern == s
	}

	if parts[0] != "" && !strings.HasPrefix(s, parts[0]) {
		return false
	}

	if parts[len(parts)-1] != "" && !strings.HasSuffix(s, parts[len(parts)-1]) {
		return false
	}

	position := len(parts[0])
	for i := 1; i < len(parts)-1; i++ {
		if parts[i] == "" {
			continue
		}
		index := strings.Index(s[position:], parts[i])
		if index < 0 {
			return false
		}
		position += index + len(parts[i])
	}

	return true
}
