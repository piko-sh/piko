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

//nolint:dupl // parallel typed API per registry.
package linguistics_domain

// PhoneticEncoderFactory creates a phonetic encoder for a given language.
// Each language adapter registers its own factory under the language name.
type PhoneticEncoderFactory = factoryFunc[PhoneticEncoderPort]

// phoneticEncoderRegistry holds the registered phonetic encoder factories keyed
// by language.
var phoneticEncoderRegistry = newRegistry[PhoneticEncoderPort]("phonetic encoder")

// RegisterPhoneticEncoderFactory registers a phonetic encoder factory for a
// language. This should be called explicitly at application startup to register
// the encoders needed for the application.
//
// Takes language (string) which is the language this factory provides
// (e.g., "english", "french").
// Takes factory (PhoneticEncoderFactory) which creates the encoder.
func RegisterPhoneticEncoderFactory(language string, factory PhoneticEncoderFactory) {
	phoneticEncoderRegistry.register(language, factory)
}

// CreatePhoneticEncoder creates a phonetic encoder for the specified language.
// If no factory is registered for the language or creation fails, a
// NoOpPhoneticEncoder is returned instead.
//
// Takes language (string) which specifies the language code for the encoder.
//
// Returns PhoneticEncoderPort which is the encoder for the given language, or
// a no-op encoder if creation fails.
func CreatePhoneticEncoder(language string) PhoneticEncoderPort {
	factory, ok := getPhoneticEncoderFactory(language)
	if !ok {
		return NewNoOpPhoneticEncoder(language)
	}

	encoder, err := factory()
	if err != nil {
		return NewNoOpPhoneticEncoder(language)
	}

	return encoder
}

// RegisteredPhoneticEncoderFactories returns the names of languages that have
// registered phonetic encoder factories.
//
// Returns []string which contains the names of all registered languages.
func RegisteredPhoneticEncoderFactories() []string {
	return phoneticEncoderRegistry.registeredNames()
}

// getPhoneticEncoderFactory retrieves a phonetic encoder factory for the given
// language.
//
// Takes language (string) which specifies the language code to look up.
//
// Returns PhoneticEncoderFactory which creates encoders for the language.
// Returns bool which indicates whether the language was found in the registry.
func getPhoneticEncoderFactory(language string) (PhoneticEncoderFactory, bool) {
	return phoneticEncoderRegistry.get(language)
}
