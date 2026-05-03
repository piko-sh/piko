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

package linguistics_stopwords_mock

import (
	"maps"
	"slices"
	"sync"

	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

var _ linguistics_domain.StopWordsProviderPort = (*MockStopWordsProvider)(nil)

// MockStopWordsProvider is a configurable stop words provider for testing.
// It allows tests to define custom stop word sets and track calls.
type MockStopWordsProvider struct {
	// getStopWordsFunc is an optional custom function for GetStopWords.
	getStopWordsFunc func(language string) map[string]bool

	// stopWords maps language codes to their sets of stop words.
	stopWords map[string]map[string]bool

	// calls records the languages passed to GetStopWords.
	calls []string

	// mu protects concurrent access to all fields.
	mu sync.RWMutex

	// passThrough when true returns empty maps instead of using stopWords.
	passThrough bool
}

// GetStopWords returns the stop words for the specified language,
// delegating to a custom function if set and falling back to the
// configured stop words or an empty map.
//
// Takes language (string) which specifies the language code.
//
// Returns map[string]bool which contains the stop words.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) GetStopWords(language string) map[string]bool {
	m.mu.Lock()
	m.calls = append(m.calls, language)
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.getStopWordsFunc != nil {
		return m.getStopWordsFunc(language)
	}

	if m.passThrough {
		return make(map[string]bool)
	}

	if words, ok := m.stopWords[language]; ok {
		return maps.Clone(words)
	}

	return make(map[string]bool)
}

// SupportedLanguages returns the list of languages that have stop words
// configured.
//
// Returns []string which contains the configured language codes.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) SupportedLanguages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	languages := make([]string, 0, len(m.stopWords))
	for lang := range m.stopWords {
		languages = append(languages, lang)
	}
	slices.Sort(languages)
	return languages
}

// SetStopWords configures the stop words for a specific language.
//
// Takes language (string) which is the language code.
// Takes words (map[string]bool) which contains the stop words to set.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) SetStopWords(language string, words map[string]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopWords[language] = words
}

// SetPassThrough enables or disables pass-through mode.
// When enabled, GetStopWords returns empty maps regardless of configuration.
//
// Takes enabled (bool) which controls pass-through mode.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) SetPassThrough(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.passThrough = enabled
}

// SetGetStopWordsFunc sets a custom function to handle GetStopWords calls.
// When set, the override is called instead of using the configured stop words.
//
// Takes override (func(string) map[string]bool) which is the custom function.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) SetGetStopWordsFunc(override func(language string) map[string]bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.getStopWordsFunc = override
}

// GetCalls returns all language codes passed to GetStopWords.
//
// Returns []string which contains the recorded language codes in order.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) GetCalls() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.calls))
	copy(result, m.calls)
	return result
}

// ResetCalls clears the recorded calls.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) ResetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls = make([]string, 0)
}

// Reset clears all configuration and recorded calls.
//
// Safe for concurrent use.
func (m *MockStopWordsProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.stopWords = make(map[string]map[string]bool)
	m.calls = make([]string, 0)
	m.passThrough = false
	m.getStopWordsFunc = nil
}

// New creates a new mock stop words provider for testing.
//
// Returns *MockStopWordsProvider which can be set up to return test data.
func New() *MockStopWordsProvider {
	return &MockStopWordsProvider{
		stopWords: make(map[string]map[string]bool),
		calls:     make([]string, 0),
	}
}
