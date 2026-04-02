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

package linguistics_stemmer_mock

import (
	"maps"
	"sync"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// MockStemmer is a thread-safe, configurable mock implementation of
// StemmerPort for testing. It supports call tracking, custom stem mappings,
// and pass-through behaviour.
type MockStemmer struct {
	// stemFunc allows custom stem behaviour when set.
	stemFunc func(word string) string

	// language is the language this stemmer is set up for.
	language string

	// stemMappings maps words to their stems for predictable test output.
	stemMappings map[string]string

	// stemCalls records all words passed to Stem for test verification.
	stemCalls []string

	// mu protects the mock's internal state from concurrent access.
	mu sync.RWMutex

	// passThrough when true returns words unchanged, like NoOpStemmer.
	passThrough bool
}

var _ linguistics_domain.StemmerPort = (*MockStemmer)(nil)

// NewWithMappings creates a mock stemmer with pre-configured stem mappings.
//
// Takes language (string) which specifies the language to report.
// Takes mappings (map[string]string) which maps input words to their stems.
//
// Returns *MockStemmer configured with the provided mappings.
func NewWithMappings(language string, mappings map[string]string) *MockStemmer {
	m := New(language)
	maps.Copy(m.stemMappings, mappings)
	return m
}

// Stem reduces a word to its stem using configured mappings or custom function.
//
// The mock looks up the word in stemMappings first. If not found:
//   - If passThrough is true, returns the word unchanged
//   - If stemFunc is set, calls that function
//   - Otherwise returns the word unchanged
//
// All calls are recorded for later verification via GetStemCalls.
//
// Takes word (string) which is the word to stem.
//
// Returns string which is the stemmed word.
//
// Safe for concurrent use; protected by a mutex.
func (m *MockStemmer) Stem(word string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stemCalls = append(m.stemCalls, word)

	if stem, exists := m.stemMappings[word]; exists {
		return stem
	}

	if m.stemFunc != nil {
		return m.stemFunc(word)
	}

	return word
}

// GetLanguage returns the language this stemmer is configured for.
//
// Returns string which is the language code.
//
// Safe for concurrent use.
func (m *MockStemmer) GetLanguage() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.language
}

// SetStemMapping configures a specific word-to-stem mapping.
//
// Takes word (string) which is the input word to match.
// Takes stem (string) which is the stem to return for that word.
//
// Safe for concurrent use.
func (m *MockStemmer) SetStemMapping(word, stem string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stemMappings[word] = stem
}

// SetStemMappings replaces all stem mappings with the provided map.
//
// Takes mappings (map[string]string) which maps words to their stems.
//
// Safe for concurrent use.
func (m *MockStemmer) SetStemMappings(mappings map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stemMappings = make(map[string]string, len(mappings))
	maps.Copy(m.stemMappings, mappings)
}

// SetPassThrough configures whether words should pass through unchanged
// when no mapping exists.
//
// Takes passThrough (bool) which when true causes Stem to return the
// input word unchanged if no mapping exists.
//
// Safe for concurrent use.
func (m *MockStemmer) SetPassThrough(passThrough bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.passThrough = passThrough
}

// SetStemFunc sets a custom function to be called by Stem when no mapping
// exists.
//
// Takes override (func(string) string) which processes words not in the mapping.
//
// Safe for concurrent use.
func (m *MockStemmer) SetStemFunc(override func(string) string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stemFunc = override
}

// SetLanguage changes the language reported by GetLanguage.
//
// Takes language (string) which is the new language to report.
//
// Safe for concurrent use.
func (m *MockStemmer) SetLanguage(language string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.language = language
}

// GetStemCalls returns a copy of all words passed to Stem.
//
// Returns []string which contains all words that were stemmed, in order.
//
// Safe for concurrent use.
func (m *MockStemmer) GetStemCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	callsCopy := make([]string, len(m.stemCalls))
	copy(callsCopy, m.stemCalls)
	return callsCopy
}

// GetStemCallCount returns the number of times Stem was called.
//
// Returns int which is the total number of Stem calls.
//
// Safe for concurrent use.
func (m *MockStemmer) GetStemCallCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.stemCalls)
}

// GetLastStemCall returns the most recent word passed to Stem.
//
// Returns string which is the last word stemmed, or empty if no calls made.
// Returns bool which is true if a call was recorded, false otherwise.
//
// Safe for concurrent use.
func (m *MockStemmer) GetLastStemCall() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.stemCalls) == 0 {
		return "", false
	}
	return m.stemCalls[len(m.stemCalls)-1], true
}

// Reset clears all recorded calls and configured mappings, preparing the
// mock for a new test case.
//
// Safe for concurrent use.
func (m *MockStemmer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stemCalls = nil
	m.stemMappings = make(map[string]string)
	m.stemFunc = nil
	m.passThrough = false
}

// ResetCalls clears only the recorded calls, keeping configured mappings.
//
// Safe for concurrent use.
func (m *MockStemmer) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stemCalls = nil
}

// New creates a new mock stemmer for the given language.
//
// Takes language (string) which sets the language returned by GetLanguage.
//
// Returns *MockStemmer which is ready for use in tests.
func New(language string) *MockStemmer {
	return &MockStemmer{
		language:     language,
		stemMappings: make(map[string]string),
		stemCalls:    nil,
		passThrough:  false,
		stemFunc:     nil,
		mu:           sync.RWMutex{},
	}
}
