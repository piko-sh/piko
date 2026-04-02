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

package linguistics_stemmer_english

import (
	"github.com/kljensen/snowball"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this stemmer.
const Language = "english"

var _ linguistics_domain.StemmerPort = (*Stemmer)(nil)

// Stemmer provides English word stemming using the Snowball algorithm.
// It implements the linguistics_domain.StemmerPort interface.
type Stemmer struct{}

// Stem reduces a word to its root form using the Snowball algorithm.
//
// Examples:
//   - "running" -> "run"
//   - "flies" -> "fli"
//   - "argument" -> "argu"
//
// Takes word (string) which is the word to stem. It should be lowercase with
// diacritics removed.
//
// Returns string which is the stemmed word, or the original word if stemming
// fails.
func (*Stemmer) Stem(word string) string {
	stemmed, err := snowball.Stem(word, Language, true)
	if err != nil {
		return word
	}
	return stemmed
}

// GetLanguage returns the language this stemmer is configured for.
//
// Returns string which is the language code.
func (*Stemmer) GetLanguage() string {
	return Language
}

// Factory creates a new English stemmer instance.
//
// Use this with linguistics_domain.RegisterStemmerFactory for explicit
// registration.
//
// Returns linguistics_domain.StemmerPort which is the configured stemmer.
// Returns error when the stemmer cannot be created.
func Factory() (linguistics_domain.StemmerPort, error) {
	return New()
}

// New creates a new English Snowball stemmer.
//
// Returns *Stemmer which is ready to use for stemming English words.
// Returns error which is always nil for this stemmer.
func New() (*Stemmer, error) {
	return &Stemmer{}, nil
}

func init() {
	linguistics_domain.RegisterStemmerFactory(Language, Factory)
}
