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

package linguistics_domain //nolint:dupl // parallel typed API per registry

// StemmerFactory creates a stemmer for a given language.
// Each language adapter registers its own factory under the language name.
type StemmerFactory = factoryFunc[StemmerPort]

// stemmerRegistry holds the registered stemmer factories keyed by language.
var stemmerRegistry = newRegistry[StemmerPort]("stemmer")

// RegisterStemmerFactory registers a stemmer factory for a language.
// This should be called explicitly at application startup to register the
// stemmers needed for the application.
//
// Takes language (string) which is the language this factory provides
// (e.g., "english", "french").
// Takes factory (StemmerFactory) which creates the stemmer.
func RegisterStemmerFactory(language string, factory StemmerFactory) {
	stemmerRegistry.register(language, factory)
}

// CreateStemmer creates a stemmer for the specified language.
// If no factory is registered for the language or creation fails, a
// NoOpStemmer is returned instead.
//
// Takes language (string) which is the language to create a stemmer for
// (e.g., "english", "french").
//
// Returns StemmerPort which is the created stemmer, or a NoOpStemmer if no
// factory is registered for the language.
func CreateStemmer(language string) StemmerPort {
	factory, ok := getStemmerFactory(language)
	if !ok {
		return NewNoOpStemmer(language)
	}

	stemmer, err := factory()
	if err != nil {
		return NewNoOpStemmer(language)
	}

	return stemmer
}

// RegisteredStemmerFactories returns the names of languages that have stemmer
// factories registered. Use it for debugging and checking what is available.
//
// Returns []string which contains the language names of all registered
// factories.
func RegisteredStemmerFactories() []string {
	return stemmerRegistry.registeredNames()
}

// getStemmerFactory retrieves a registered stemmer factory for the given
// language.
//
// Takes language (string) which is the language identifier.
//
// Returns StemmerFactory which is the factory for that language, or nil if not
// found.
// Returns bool which is true if a factory was found, false otherwise.
func getStemmerFactory(language string) (StemmerFactory, bool) {
	return stemmerRegistry.get(language)
}
