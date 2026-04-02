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

package linguistics_phonetic_mock

import (
	"maps"
	"sync"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// MockPhoneticEncoder is a thread-safe, configurable mock implementation of
// PhoneticEncoderPort for testing. It supports call tracking, custom encode
// mappings, and pass-through behaviour.
type MockPhoneticEncoder struct {
	// encodeFunc allows custom encoding behaviour when set.
	encodeFunc func(word string) string

	// language is the language code this encoder uses.
	language string

	// encodeMappings stores word to phonetic code pairs for test output.
	encodeMappings map[string]string

	// encodeCalls records all words passed to Encode for test verification.
	encodeCalls []string

	// mu protects the mock's internal state during concurrent access.
	mu sync.RWMutex

	// passThrough when true returns words unchanged, like NoOpPhoneticEncoder.
	passThrough bool
}

var _ linguistics_domain.PhoneticEncoderPort = (*MockPhoneticEncoder)(nil)

// NewWithMappings creates a mock encoder with pre-configured encode mappings.
//
// Takes language (string) which specifies the language to report.
// Takes mappings (map[string]string) which maps input words to their phonetic
// codes.
//
// Returns *MockPhoneticEncoder configured with the provided mappings.
func NewWithMappings(language string, mappings map[string]string) *MockPhoneticEncoder {
	m := New(language)
	maps.Copy(m.encodeMappings, mappings)
	return m
}

// Encode returns a phonetic code using configured mappings or custom function.
//
// The mock looks up the word in encodeMappings first. If not found:
//   - If passThrough is true, returns the word unchanged
//   - If encodeFunc is set, calls that function
//   - Otherwise returns an empty string
//
// All calls are recorded for later verification via GetEncodeCalls.
//
// Takes word (string) which is the word to encode.
//
// Returns string which is the phonetic code.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockPhoneticEncoder) Encode(word string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.encodeCalls = append(m.encodeCalls, word)

	if code, exists := m.encodeMappings[word]; exists {
		return code
	}

	if m.encodeFunc != nil {
		return m.encodeFunc(word)
	}

	if m.passThrough {
		return word
	}

	return ""
}

// GetLanguage returns the language this encoder is configured for.
//
// Returns string which is the language code.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) GetLanguage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.language
}

// SetEncodeMapping configures a specific word-to-code mapping.
//
// Takes word (string) which is the input word to match.
// Takes code (string) which is the phonetic code to return for that word.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) SetEncodeMapping(word, code string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.encodeMappings[word] = code
}

// SetEncodeMappings replaces all encode mappings with the provided map.
//
// Takes mappings (map[string]string) which maps words to their phonetic codes.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) SetEncodeMappings(mappings map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.encodeMappings = make(map[string]string, len(mappings))
	maps.Copy(m.encodeMappings, mappings)
}

// SetPassThrough configures whether words should pass through unchanged
// when no mapping exists.
//
// Takes passThrough (bool) which when true causes Encode to return the
// input word unchanged if no mapping exists.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) SetPassThrough(passThrough bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.passThrough = passThrough
}

// SetEncodeFunc sets a custom function to be called by Encode when no mapping
// exists.
//
// Takes override (func(string) string) which processes words not in the mapping.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) SetEncodeFunc(override func(string) string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.encodeFunc = override
}

// SetLanguage changes the language reported by GetLanguage.
//
// Takes language (string) which is the new language to report.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) SetLanguage(language string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.language = language
}

// GetEncodeCalls returns a copy of all words passed to Encode.
//
// Returns []string which contains all words that were encoded, in order.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) GetEncodeCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]string, len(m.encodeCalls))
	copy(callsCopy, m.encodeCalls)
	return callsCopy
}

// GetEncodeCallCount returns the number of times Encode was called.
//
// Returns int which is the total number of Encode calls.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) GetEncodeCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.encodeCalls)
}

// GetLastEncodeCall returns the most recent word passed to Encode.
//
// Returns string which is the last word encoded, or empty if no calls made.
// Returns bool which is true if a call was recorded, false otherwise.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) GetLastEncodeCall() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.encodeCalls) == 0 {
		return "", false
	}
	return m.encodeCalls[len(m.encodeCalls)-1], true
}

// Reset clears all recorded calls and configured mappings, preparing the
// mock for a new test case.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.encodeCalls = nil
	m.encodeMappings = make(map[string]string)
	m.encodeFunc = nil
	m.passThrough = false
}

// ResetCalls clears only the recorded calls, keeping configured mappings.
//
// Safe for concurrent use.
func (m *MockPhoneticEncoder) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.encodeCalls = nil
}

// New creates a mock phonetic encoder for the given language.
//
// Takes language (string) which sets the language returned by GetLanguage.
//
// Returns *MockPhoneticEncoder which is ready for use in tests.
func New(language string) *MockPhoneticEncoder {
	return &MockPhoneticEncoder{
		language:       language,
		encodeMappings: make(map[string]string),
		encodeCalls:    nil,
		passThrough:    false,
		encodeFunc:     nil,
		mu:             sync.RWMutex{},
	}
}
