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

package linguistics_stopwords_german

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "german"

// stopWords contains common German stop words.
var stopWords = map[string]bool{
	"der": true, "die": true, "das": true, "den": true, "dem": true, "des": true,
	"ein": true, "eine": true, "einer": true, "einem": true, "einen": true, "eines": true,
	"ich": true, "du": true, "er": true, "sie": true, "es": true, "wir": true, "ihr": true,
	"mich": true, "dich": true, "ihn": true, "uns": true, "euch": true, "ihnen": true,
	"mir": true, "dir": true, "ihm": true,
	"mein": true, "dein": true, "sein": true, "unser": true, "euer": true,
	"meine": true, "deine": true, "seine": true, "ihre": true, "unsere": true, "eure": true,
	"in": true, "an": true, "auf": true, "aus": true, "bei": true, "mit": true, "nach": true, "von": true, "zu": true, "um": true, "für": true, "über": true, "unter": true,
	"vor": true, "hinter": true, "neben": true, "zwischen": true, "durch": true, "gegen": true, "ohne": true,
	"und": true, "oder": true, "aber": true, "denn": true, "weil": true, "dass": true, "ob": true, "wenn": true, "als": true, "wie": true,
	"ist": true, "sind": true, "war": true, "waren": true, "bin": true, "bist": true, "seid": true, "gewesen": true,
	"hat": true, "haben": true, "hatte": true, "hatten": true, "habe": true, "hast": true, "habt": true,
	"wird": true, "werden": true, "wurde": true, "wurden": true, "werde": true, "wirst": true, "werdet": true,
	"kann": true, "können": true, "konnte": true, "konnten": true,
	"muss": true, "müssen": true, "musste": true, "mussten": true,
	"soll": true, "sollen": true, "sollte": true, "sollten": true,
	"will": true, "wollen": true, "wollte": true, "wollten": true,
	"dieser": true, "diese": true, "dieses": true, "jener": true, "jene": true, "jenes": true,
	"nicht": true, "kein": true, "keine": true, "auch": true, "noch": true, "schon": true, "nur": true, "sehr": true, "mehr": true,
	"hier": true, "dort": true, "da": true, "wo": true, "was": true, "wer": true, "wann": true, "warum": true,
	"so": true, "dann": true, "doch": true, "also": true, "nun": true, "jetzt": true, "immer": true, "wieder": true,
}

// Provider supplies German stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the German stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "german".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new German stop words provider instance. Use this with
// linguistics_domain.RegisterStopWordsProviderFactory for explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides German stop
// words filtering.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a German stop words provider.
//
// Returns *Provider which is ready for use.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
