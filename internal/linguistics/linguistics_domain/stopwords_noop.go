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

// NoOpStopWordsProvider implements StopWordsProviderPort and returns empty maps.
// It is the default when no stop words provider is set, allowing the system to
// work without stop word filtering.
type NoOpStopWordsProvider struct{}

var _ StopWordsProviderPort = (*NoOpStopWordsProvider)(nil)

// NewNoOpStopWordsProvider creates a stop words provider that does nothing.
// The provider returns empty stop word maps for all languages.
//
// Returns *NoOpStopWordsProvider which implements StopWordsProviderPort but
// provides no stop words.
func NewNoOpStopWordsProvider() *NoOpStopWordsProvider {
	return &NoOpStopWordsProvider{}
}

// GetStopWords returns an empty map for any language. Satisfies the
// StopWordsProviderPort interface without providing any actual stop words.
//
// Returns map[string]bool which is always an empty map.
func (*NoOpStopWordsProvider) GetStopWords(_ string) map[string]bool {
	return make(map[string]bool)
}

// SupportedLanguages returns an empty slice.
// This provider does not support any languages.
//
// Returns []string which is always an empty slice.
func (*NoOpStopWordsProvider) SupportedLanguages() []string {
	return []string{}
}
