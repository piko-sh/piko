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

package linguistics_stopwords_spanish

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "spanish"

// stopWords contains common Spanish stop words.
var stopWords = map[string]bool{
	"el": true, "la": true, "los": true, "las": true, "un": true, "una": true, "unos": true, "unas": true,
	"yo": true, "tú": true, "él": true, "ella": true, "nosotros": true, "vosotros": true, "ellos": true, "ellas": true,
	"me": true, "te": true, "se": true, "nos": true, "os": true,
	"de": true, "en": true, "a": true, "por": true, "para": true, "con": true, "sin": true, "sobre": true,
	"y": true, "o": true, "pero": true, "porque": true, "que": true, "si": true,
	"es": true, "son": true, "ser": true, "estar": true, "ha": true, "hay": true, "fue": true, "sido": true,
	"este": true, "esta": true, "estos": true, "estas": true, "ese": true, "esa": true, "esos": true, "esas": true,
	"como": true, "más": true, "su": true, "sus": true, "del": true, "al": true,
}

// Provider provides Spanish stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the Spanish stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "spanish".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new Spanish stop words provider instance.
//
// Use this with linguistics_domain.RegisterStopWordsProviderFactory for
// explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides the Spanish
// stop words provider.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a Spanish stop words provider.
//
// Returns *Provider which is the configured provider ready for use.
// Returns error when creation fails, though this provider always returns nil.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() {
	linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory)
}
