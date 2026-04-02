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

package linguistics_domain

// NoOpStemmer is a stemmer that returns words unchanged.
// It implements the StemmerPort interface and is used as the default when
// no stemmer is configured.
//
// Use this when:
//   - No stemming is required for your use case
//   - A language is not supported by available stemmers
//   - Testing tokenisation without stemming side effects
type NoOpStemmer struct {
	// language is the language code for this stemmer.
	language string
}

var _ StemmerPort = (*NoOpStemmer)(nil)

// NewNoOpStemmer creates a no-op stemmer for the specified language.
// The stemmer will return all words unchanged.
//
// Takes language (string) which is the language code to associate with this
// stemmer. The language is normalised using ValidateLanguage.
//
// Returns *NoOpStemmer which implements StemmerPort but performs no stemming.
func NewNoOpStemmer(language string) *NoOpStemmer {
	return &NoOpStemmer{
		language: ValidateLanguage(language),
	}
}

// Stem returns the word unchanged.
// This method satisfies the StemmerPort interface without performing any
// actual stemming transformation.
//
// Takes word (string) which is the word to "stem".
//
// Returns string which is the same word, unchanged.
func (*NoOpStemmer) Stem(word string) string {
	return word
}

// GetLanguage returns the language this stemmer is configured for.
//
// Returns string which is the language code.
func (s *NoOpStemmer) GetLanguage() string {
	return s.language
}
