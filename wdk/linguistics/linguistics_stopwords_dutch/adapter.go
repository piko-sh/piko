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

package linguistics_stopwords_dutch

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "dutch"

// stopWords contains common Dutch stop words.
var stopWords = map[string]bool{
	"de": true, "het": true, "een": true,
	"ik": true, "je": true, "jij": true, "u": true, "hij": true, "zij": true, "ze": true, "wij": true, "we": true, "jullie": true,
	"mij": true, "me": true, "jou": true, "hem": true, "haar": true, "ons": true, "hen": true, "hun": true,
	"mijn": true, "jouw": true, "uw": true, "zijn": true, "onze": true,
	"in": true, "op": true, "aan": true, "van": true, "voor": true, "met": true, "naar": true, "om": true, "bij": true, "tot": true, "uit": true, "over": true, "door": true,
	"en": true, "of": true, "maar": true, "want": true, "dus": true, "dat": true, "die": true, "wie": true, "wat": true, "waar": true, "wanneer": true, "hoe": true,
	"is": true, "was": true, "waren": true, "ben": true, "bent": true, "wordt": true, "worden": true, "werd": true, "werden": true,
	"heeft": true, "hebben": true, "had": true, "hadden": true, "kan": true, "kunnen": true, "kon": true, "konden": true,
	"zal": true, "zullen": true, "zou": true, "zouden": true, "moet": true, "moeten": true,
	"dit": true, "deze": true,
	"niet": true, "geen": true, "wel": true, "ook": true, "nog": true, "al": true, "er": true, "hier": true, "daar": true,
	"nu": true, "dan": true, "zo": true, "als": true, "meer": true, "veel": true, "te": true, "zeer": true,
}

// Provider supplies Dutch stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the Dutch stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "dutch".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new Dutch stop words provider instance.
//
// Use this with linguistics_domain.RegisterStopWordsProviderFactory for
// explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which is the provider.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a new Dutch stop words provider.
//
// Returns *Provider which provides Dutch stop word filtering.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
