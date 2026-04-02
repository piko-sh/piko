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

package linguistics_stopwords_swedish

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "swedish"

// stopWords contains common Swedish stop words (stoppord).
var stopWords = map[string]bool{
	"en": true, "ett": true, "den": true, "det": true, "de": true,
	"jag": true, "du": true, "han": true, "hon": true, "vi": true, "ni": true,
	"mig": true, "dig": true, "honom": true, "henne": true, "oss": true, "er": true, "dem": true,
	"min": true, "din": true, "hans": true, "hennes": true, "sin": true, "vår": true, "deras": true,
	"mitt": true, "ditt": true, "sitt": true, "vårt": true, "ert": true,
	"mina": true, "dina": true, "sina": true, "våra": true, "era": true,
	"i": true, "på": true, "till": true, "från": true, "med": true, "av": true, "för": true, "om": true, "vid": true, "efter": true, "under": true, "över": true, "mellan": true,
	"och": true, "eller": true, "men": true, "så": true, "att": true, "när": true, "där": true, "hur": true, "vad": true, "vem": true, "vilken": true,
	"är": true, "var": true, "varit": true, "vara": true, "blir": true, "blev": true, "blivit": true, "bli": true,
	"har": true, "hade": true, "haft": true, "ha": true,
	"kan": true, "kunde": true, "kunnat": true, "kunna": true,
	"ska": true, "skulle": true, "skall": true,
	"vill": true, "ville": true, "velat": true, "vilja": true,
	"måste": true, "får": true, "fick": true, "fått": true,
	"denna": true, "dette": true, "dessa": true,
	"inte": true, "ingen": true, "inget": true, "inga": true,
	"också": true, "bara": true, "redan": true, "än": true, "nu": true, "här": true,
	"mycket": true, "mer": true, "mest": true, "alla": true, "allt": true, "annat": true,
	"sig": true, "som": true, "sedan": true, "dock": true, "ju": true, "nog": true,
}

// Provider provides Swedish stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the Swedish stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "swedish".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new Swedish stop words provider instance. Use this with
// linguistics_domain.RegisterStopWordsProviderFactory for explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides Swedish stop
// words.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a new Swedish stop words provider.
//
// Returns *Provider which is ready for use.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
