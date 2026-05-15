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

// BigramAnalyserFactory creates a bigram analyser for a given language.
type BigramAnalyserFactory = factoryFunc[BigramAnalyserPort]

// bigramAnalyserRegistry holds all registered bigram analyser factories.
var bigramAnalyserRegistry = newRegistry[BigramAnalyserPort]("bigram_analyser")

// RegisterBigramAnalyserFactory registers a bigram analyser factory for a
// language.
//
// Takes language (string) which is the language code.
// Takes factory (BigramAnalyserFactory) which creates the analyser.
func RegisterBigramAnalyserFactory(language string, factory BigramAnalyserFactory) {
	bigramAnalyserRegistry.register(language, factory)
}

// CreateBigramAnalyser creates a bigram analyser for the specified language.
//
// Takes language (string) which is the language code to look up.
//
// Returns BigramAnalyserPort which is the analyser, or nil if no factory is
// registered.
func CreateBigramAnalyser(language string) BigramAnalyserPort {
	factory, ok := getBigramAnalyserFactory(language)
	if !ok {
		return nil
	}

	analyser, err := factory()
	if err != nil {
		return nil
	}

	return analyser
}

// RegisteredBigramAnalyserFactories returns the names of languages that
// have bigram analyser factories registered.
//
// Returns []string which contains the registered language codes.
func RegisteredBigramAnalyserFactories() []string {
	return bigramAnalyserRegistry.registeredNames()
}

// getBigramAnalyserFactory looks up a factory by language code.
//
// Takes language (string) which is the language code.
//
// Returns BigramAnalyserFactory which is the factory.
// Returns bool which is true when found.
func getBigramAnalyserFactory(language string) (BigramAnalyserFactory, bool) {
	return bigramAnalyserRegistry.get(language)
}
