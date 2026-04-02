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

package linguistics_stemmer_german

import (
	"github.com/dchest/stemmer/german"
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for the German stemmer.
const Language = "german"

var _ linguistics_domain.StemmerPort = (*Stemmer)(nil)

// Stemmer provides German word stemming using the dchest/stemmer library.
// It implements the linguistics_domain.StemmerPort interface.
type Stemmer struct{}

// Stem reduces a word to its root form.
//
// Examples:
//   - "laufen" -> "lauf"
//   - "hauser" -> "haus"
//
// Takes word (string) which is the word to stem.
//
// Returns string which is the stemmed word.
func (*Stemmer) Stem(word string) string {
	return german.Stemmer.Stem(word)
}

// GetLanguage returns the language this stemmer is configured for.
//
// Returns string which is the language code for the stemmer.
func (*Stemmer) GetLanguage() string {
	return Language
}

// Factory creates a new German stemmer instance.
//
// Use this with linguistics_domain.RegisterStemmerFactory for explicit
// registration.
//
// Returns linguistics_domain.StemmerPort which is the configured stemmer.
// Returns error when the stemmer cannot be created.
func Factory() (linguistics_domain.StemmerPort, error) {
	return New()
}

// New creates a new German stemmer.
//
// Returns *Stemmer which is ready for use.
// Returns error which is always nil for this stemmer.
func New() (*Stemmer, error) {
	return &Stemmer{}, nil
}

func init() {
	linguistics_domain.RegisterStemmerFactory(Language, Factory)
}
