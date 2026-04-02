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

package linguistics_stopwords_hungarian

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this provider.
const Language = "hungarian"

// stopWords contains common Hungarian stop words.
var stopWords = map[string]bool{
	"a": true, "az": true, "egy": true,
	"én": true, "te": true, "ő": true, "mi": true, "ti": true, "ők": true,
	"engem": true, "téged": true, "őt": true, "minket": true, "titeket": true, "őket": true,
	"nekem": true, "neked": true, "neki": true, "nekünk": true, "nektek": true, "nekik": true,
	"enyém": true, "tiéd": true, "övé": true, "miénk": true, "tiétek": true, "övék": true,
	"ban": true, "ben": true, "ból": true, "ből": true, "ba": true, "be": true,
	"on": true, "en": true, "ön": true, "ról": true, "ről": true, "ra": true, "re": true,
	"nál": true, "nél": true, "hoz": true, "hez": true, "höz": true,
	"tól": true, "től": true, "ig": true, "ért": true, "val": true, "vel": true,
	"alatt": true, "fölött": true, "között": true, "mellett": true, "mögött": true, "előtt": true, "után": true,
	"és": true, "vagy": true, "de": true, "hogy": true, "ha": true, "mint": true, "mert": true, "amikor": true, "ahol": true, "ami": true, "aki": true,
	"van": true, "volt": true, "lesz": true, "lett": true, "lenni": true,
	"vagyok": true, "vagyunk": true, "vagytok": true, "vannak": true,
	"voltam": true, "voltál": true, "voltunk": true, "voltatok": true, "voltak": true,
	"ez": true, "ezek": true, "azok": true, "itt": true, "ott": true,
	"ki": true, "hol": true, "mikor": true, "hogyan": true, "miért": true, "melyik": true, "mennyi": true,
	"nem": true, "is": true, "még": true, "már": true, "csak": true, "meg": true, "el": true, "fel": true, "le": true,
	"igen": true, "sem": true, "minden": true, "más": true, "sok": true, "kevés": true, "nagy": true, "kicsi": true,
	"új": true, "régi": true, "jó": true, "rossz": true, "így": true, "úgy": true, "olyan": true, "ilyen": true,
	"majd": true, "most": true, "akkor": true, "tehát": true, "pedig": true, "hiszen": true, "persze": true,
}

// Provider provides Hungarian stop words for text analysis.
// It implements the linguistics_domain.StopWordsProviderPort interface.
type Provider struct{}

// GetStopWords returns the Hungarian stop words.
//
// Takes _ (string) which is ignored for this single-language provider.
//
// Returns map[string]bool containing the stop words.
func (*Provider) GetStopWords(_ string) map[string]bool {
	return stopWords
}

// SupportedLanguages returns the languages supported by this provider.
//
// Returns []string containing only "hungarian".
func (*Provider) SupportedLanguages() []string {
	return []string{Language}
}

var _ linguistics_domain.StopWordsProviderPort = (*Provider)(nil)

// Factory creates a new Hungarian stop words provider instance.
// Use this with linguistics_domain.RegisterStopWordsProviderFactory
// for explicit registration.
//
// Returns linguistics_domain.StopWordsProviderPort which is the provider
// instance for Hungarian stop words.
// Returns error when the provider cannot be created.
func Factory() (linguistics_domain.StopWordsProviderPort, error) {
	return New()
}

// New creates a Hungarian stop words provider.
//
// Returns *Provider which is ready to use.
// Returns error which is always nil for this provider.
func New() (*Provider, error) {
	return &Provider{}, nil
}

func init() { linguistics_domain.RegisterStopWordsProviderFactory(Language, Factory) }
