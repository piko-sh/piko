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

package linguistics_domain

// BigramAnalyserPort analyses text for character bigram frequency patterns.
// Implementations provide language-specific bigram frequency tables for
// detecting gibberish or random text.
type BigramAnalyserPort interface {
	// BigramFrequencyRatio returns the ratio of uncommon character bigrams
	// to total bigrams in the text. Higher values indicate more random or
	// nonsensical character patterns.
	//
	// Returns the ratio (0.0 to 1.0) and whether analysis was performed
	// (false when the text is too short).
	BigramFrequencyRatio(text string) (ratio float64, analysed bool)

	// GetLanguage returns the language this analyser is configured for.
	GetLanguage() string
}
