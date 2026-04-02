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

package linguistics_stopwords_french

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "french"

// stopWords contains common French stop words (mots vides) organised by
// grammatical category. Words are in normalised form (without accents) as input
// is expected to be pre-normalised.
var stopWords = map[string]bool{
	"le": true, "la": true, "les": true, "l": true,

	"un": true, "une": true, "des": true, "du": true,

	"au": true, "aux": true,

	"je": true, "tu": true, "il": true, "elle": true, "on": true,
	"nous": true, "vous": true, "ils": true, "elles": true,

	"me": true, "te": true, "se": true, "lui": true, "leur": true,

	"mon": true, "ma": true, "mes": true,
	"ton": true, "ta": true, "tes": true,
	"son": true, "sa": true, "ses": true,

	"notre": true, "nos": true, "votre": true, "vos": true, "leurs": true,

	"ce": true, "cet": true, "cette": true, "ces": true, "ceci": true, "cela": true,

	"qui": true, "que": true, "quoi": true, "dont": true, "ou": true,
	"lequel": true, "laquelle": true, "lesquels": true, "lesquelles": true,
	"comment": true, "pourquoi": true, "quand": true,

	"de": true, "a": true, "en": true, "pour": true, "par": true,
	"avec": true, "dans": true, "sur": true, "sous": true, "sans": true,
	"vers": true, "chez": true, "entre": true, "depuis": true, "pendant": true,

	"et": true, "mais": true, "donc": true, "car": true,
	"si": true, "comme": true, "lorsque": true, "puisque": true, "quoique": true, "parce": true,

	"suis": true, "es": true, "est": true, "sommes": true, "etes": true, "sont": true,
	"etre": true, "ete": true, "etais": true, "etait": true,

	"ai": true, "as": true, "avons": true, "avez": true, "ont": true,
	"avoir": true, "eu": true, "avais": true, "avait": true, "avaient": true,

	"vais": true, "vas": true, "va": true, "allons": true, "allez": true, "vont": true,

	"fais": true, "fait": true, "faisons": true, "faites": true, "font": true, "faire": true,

	"peux": true, "peut": true, "peuvent": true, "pouvoir": true,
	"veux": true, "veut": true, "veulent": true, "vouloir": true,

	"ne": true, "n": true, "pas": true, "plus": true, "jamais": true,
	"rien": true, "personne": true, "aucun": true, "aucune": true,

	"tres": true, "moins": true, "peu": true, "beaucoup": true,
	"trop": true, "assez": true, "tant": true, "tout": true, "tous": true, "toute": true,

	"ici": true, "maintenant": true, "alors": true,
	"encore": true, "toujours": true, "deja": true, "souvent": true,
	"bien": true, "seulement": true,

	"autre": true, "autres": true, "meme": true, "quelque": true, "quelques": true, "certain": true,

	"aussi": true, "ainsi": true, "puis": true, "ensuite": true, "enfin": true, "non": true,
}

// Provider supplies French stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the French stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "french".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new French stop words provider instance.
//
// Use this with linguistics_domain.RegisterStopWordsProviderFactory for
// explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides French
// stop words.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a new French stop words provider.
//
// Returns *Provider which is a ready-to-use French stop words provider.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
