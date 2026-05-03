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

// NoOpPhoneticEncoder is a phonetic encoder that returns empty strings.
// It implements the PhoneticEncoderPort interface and is used as the default
// when no phonetic encoder is configured.
//
// Use this when:
//   - No phonetic encoding is required for your use case
//   - A language is not supported by available phonetic encoders
//   - Testing tokenisation without phonetic encoding side effects
type NoOpPhoneticEncoder struct {
	// language is the language code for this encoder.
	language string
}

var _ PhoneticEncoderPort = (*NoOpPhoneticEncoder)(nil)

// NewNoOpPhoneticEncoder creates a no-op phonetic encoder for the specified
// language. The encoder will return empty strings for all words.
//
// Takes language (string) which is the language code to associate with this
// encoder. The language is normalised using ValidateLanguage.
//
// Returns *NoOpPhoneticEncoder which implements PhoneticEncoderPort but
// performs no encoding.
func NewNoOpPhoneticEncoder(language string) *NoOpPhoneticEncoder {
	return &NoOpPhoneticEncoder{
		language: ValidateLanguage(language),
	}
}

// Encode returns an empty string. Satisfies the PhoneticEncoderPort interface
// without performing any actual phonetic encoding.
//
// Returns string which is always empty.
func (*NoOpPhoneticEncoder) Encode(_ string) string {
	return ""
}

// GetLanguage returns the language this encoder is configured for.
//
// Returns string which is the language code.
func (e *NoOpPhoneticEncoder) GetLanguage() string {
	return e.language
}
