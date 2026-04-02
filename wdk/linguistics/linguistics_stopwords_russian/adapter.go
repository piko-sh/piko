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

package linguistics_stopwords_russian

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "russian"

// stopWords contains common Russian stop words.
var stopWords = map[string]bool{
	"я": true, "ты": true, "он": true, "она": true, "мы": true, "вы": true, "они": true,
	"меня": true, "тебя": true, "его": true, "её": true, "нас": true, "вас": true, "их": true,
	"в": true, "на": true, "с": true, "к": true, "о": true, "от": true, "для": true, "по": true, "из": true, "за": true, "у": true,
	"и": true, "а": true, "но": true, "или": true, "что": true, "чтобы": true, "если": true,
	"быть": true, "есть": true, "был": true, "была": true, "были": true, "будет": true,
	"это": true, "этот": true, "эта": true, "эти": true, "тот": true, "та": true, "те": true,
	"не": true, "нет": true, "да": true, "как": true, "так": true, "ещё": true, "уже": true,
}

// Provider provides Russian stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the Russian stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "russian".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new Russian stop words provider instance. Use this with
// linguistics_domain.RegisterStopWordsProviderFactory for explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides Russian stop
// words filtering.
// Returns error when the provider cannot be initialised.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a Russian stop words provider.
//
// Returns *Provider which is ready to use.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
