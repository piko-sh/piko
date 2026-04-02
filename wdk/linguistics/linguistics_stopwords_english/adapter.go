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

package linguistics_stopwords_english

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "english"

// stopWords contains common English stop words organised by grammatical category.
var stopWords = map[string]bool{
	"a": true, "an": true, "the": true,

	"and": true, "but": true, "or": true, "nor": true, "for": true, "yet": true, "so": true,

	"in": true, "on": true, "at": true, "to": true, "from": true, "of": true, "with": true,
	"about": true, "by": true, "into": true, "through": true, "during": true, "before": true,
	"after": true, "above": true, "below": true, "between": true, "under": true, "over": true,
	"out": true, "off": true, "up": true, "down": true, "around": true, "against": true,

	"i": true, "you": true, "he": true, "she": true, "it": true, "we": true, "they": true,

	"me": true, "him": true, "her": true, "us": true, "them": true,

	"my": true, "your": true, "his": true, "its": true, "our": true, "their": true, "whose": true,

	"mine": true, "yours": true, "hers": true, "ours": true, "theirs": true, "own": true,

	"myself": true, "yourself": true, "himself": true, "herself": true, "itself": true,
	"ourselves": true, "yourselves": true, "themselves": true,

	"is": true, "am": true, "are": true, "was": true, "were": true, "be": true, "been": true, "being": true,

	"have": true, "has": true, "had": true, "having": true,

	"do": true, "does": true, "did": true, "doing": true,

	"can": true, "could": true, "will": true, "would": true, "shall": true, "should": true,
	"may": true, "might": true, "must": true,

	"this": true, "that": true, "these": true, "those": true,

	"what": true, "which": true, "who": true, "whom": true, "when": true, "where": true, "why": true, "how": true,

	"all": true, "any": true, "some": true, "each": true, "every": true, "no": true, "none": true,
	"many": true, "much": true, "few": true, "little": true, "more": true, "most": true,
	"less": true, "least": true,

	"anyone": true, "anything": true, "someone": true, "something": true, "everyone": true,
	"everything": true, "nobody": true, "nothing": true, "anybody": true, "somebody": true,

	"very": true, "really": true, "just": true, "only": true, "even": true, "still": true,
	"already": true, "always": true, "never": true, "ever": true, "often": true, "sometimes": true,
	"here": true, "there": true, "then": true,

	"now": true, "again": true, "once": true, "twice": true, "further": true, "soon": true,
	"today": true, "tomorrow": true,

	"not": true,

	"than": true, "also": true, "too": true, "both": true, "either": true, "neither": true,
	"other": true, "another": true, "such": true, "same": true, "like": true, "as": true,

	"because": true, "although": true, "if": true, "unless": true, "while": true,
	"whether": true, "until": true, "since": true, "however": true, "therefore": true,
}

// Provider provides English stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the English stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "english".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new English stop words provider instance.
//
// Use this with linguistics_domain.RegisterStopWordsProviderFactory for
// explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which provides English stop
// word filtering.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a new English stop words provider.
//
// Returns *Provider which is ready for use.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
