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

// StopWordsProviderFactory creates a stop words provider for a given language.
// Each language adapter registers its own factory under the language name.
type StopWordsProviderFactory = factoryFunc[StopWordsProviderPort]

// stopWordsProviderRegistry holds the registered stop words provider factories
// keyed by language.
var stopWordsProviderRegistry = newRegistry[StopWordsProviderPort]("stop words provider")

// RegisterStopWordsProviderFactory registers a stop words provider factory for
// a language. This should be called explicitly at application startup to
// register the providers needed for the application.
//
// Takes language (string) which is the language this factory provides
// (e.g., "english", "french").
// Takes factory (StopWordsProviderFactory) which creates providers.
func RegisterStopWordsProviderFactory(language string, factory StopWordsProviderFactory) {
	stopWordsProviderRegistry.register(language, factory)
}

// CreateStopWordsProvider creates a stop words provider for the specified
// language. If no factory is registered for the language or creation fails, a
// NoOpStopWordsProvider is returned instead.
//
// Takes language (string) which is the language to create a provider for
// (e.g., "english", "french").
//
// Returns StopWordsProviderPort which is the created provider, or a
// NoOpStopWordsProvider if no factory is registered for the language.
func CreateStopWordsProvider(language string) StopWordsProviderPort {
	factory, ok := getStopWordsProviderFactory(language)
	if !ok {
		return NewNoOpStopWordsProvider()
	}

	provider, err := factory()
	if err != nil {
		return NewNoOpStopWordsProvider()
	}

	return provider
}

// RegisteredStopWordsProviderFactories returns the names of all languages that
// have registered stop words provider factories. Use it for debugging and
// checking what is available.
//
// Returns []string which contains the language names of all registered
// factories.
func RegisteredStopWordsProviderFactories() []string {
	return stopWordsProviderRegistry.registeredNames()
}

// getStopWordsProviderFactory retrieves a registered stop words provider
// factory for the given language.
//
// Takes language (string) which is the language code to look up.
//
// Returns StopWordsProviderFactory which is the factory for the language.
// Returns bool which is false if no factory is registered for the language.
func getStopWordsProviderFactory(language string) (StopWordsProviderFactory, bool) {
	return stopWordsProviderRegistry.get(language)
}
