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

package linguistics_stopwords_norwegian

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "norwegian"

// stopWords contains common Norwegian stop words (stoppord).
// Covers both Bokmal and Nynorsk variants.
var stopWords = map[string]bool{
	"en": true, "ei": true, "et": true, "den": true, "det": true, "de": true,
	"jeg": true, "du": true, "han": true, "hun": true, "vi": true, "dere": true,
	"meg": true, "deg": true, "ham": true, "henne": true, "oss": true, "dem": true,
	"min": true, "din": true, "hans": true, "hennes": true, "sin": true, "vår": true, "deres": true,
	"mitt": true, "ditt": true, "sitt": true, "vårt": true,
	"mine": true, "dine": true, "sine": true, "våre": true,
	"i": true, "på": true, "til": true, "fra": true, "med": true, "av": true, "for": true, "om": true, "ved": true, "etter": true, "under": true, "over": true, "mellom": true,
	"og": true, "eller": true, "men": true, "så": true, "at": true, "når": true, "hvor": true, "hvordan": true, "hva": true, "hvem": true, "hvilken": true,
	"er": true, "var": true, "vært": true, "være": true, "blir": true, "ble": true, "blitt": true, "bli": true,
	"har": true, "hadde": true, "hatt": true, "ha": true,
	"kan": true, "kunne": true, "kunnet": true,
	"skal": true, "skulle": true, "skullet": true,
	"vil": true, "ville": true, "villet": true,
	"må": true, "måtte": true, "måttet": true,
	"denne": true, "dette": true, "disse": true,
	"ikke": true, "ingen": true, "intet": true, "inga": true,
	"også": true, "bare": true, "allerede": true, "enn": true, "nå": true, "her": true, "der": true,
	"mye": true, "mer": true, "mest": true, "alle": true, "alt": true, "annet": true,
	"seg": true, "som": true, "siden": true, "da": true, "jo": true, "nok": true,
}

// Provider provides Norwegian stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the Norwegian stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "norwegian".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new Norwegian stop words provider instance.
//
// Use this with linguistics_domain.RegisterStopWordsProviderFactory for
// explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides Norwegian
// stop words.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a Norwegian stop words provider.
//
// Returns *Provider which is ready for use.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
