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

package linguistics_phonetic_english

import (
	"piko.sh/piko/internal/linguistics/linguistics_domain"
)

// Language is the language code for this encoder.
const Language = "english"

var _ linguistics_domain.PhoneticEncoderPort = (*Encoder)(nil)

// Encoder provides English phonetic encoding using the Double Metaphone
// algorithm. It implements the linguistics_domain.PhoneticEncoderPort interface.
type Encoder struct {
	// encoder converts text to phonetic representations for comparison.
	encoder *linguistics_domain.PhoneticEncoder
}

// NewWithMaxLength creates a new English Double Metaphone encoder with a custom
// maximum code length.
//
// Takes maxLength (int) which controls the maximum length of phonetic codes.
//
// Returns (*Encoder, error) where the error is always nil for this encoder.
func NewWithMaxLength(maxLength int) (*Encoder, error) {
	return &Encoder{
		encoder: linguistics_domain.NewPhoneticEncoder(maxLength),
	}, nil
}

// Encode returns the primary phonetic encoding of a word using the Double
// Metaphone algorithm.
//
// Takes word (string) which is the word to encode. The word should be in lower
// case and without accents for best results.
//
// Returns string which is the phonetic code, typically four characters.
func (e *Encoder) Encode(word string) string {
	return e.encoder.Encode(word)
}

// GetLanguage returns the language this encoder is configured for.
//
// Returns string which is the language code.
func (*Encoder) GetLanguage() string {
	return Language
}

// Factory creates a new English phonetic encoder instance. Use this with
// linguistics_domain.RegisterPhoneticEncoderFactory for explicit registration.
//
// Returns linguistics_domain.PhoneticEncoderPort which is the encoder instance.
// Returns error when the encoder cannot be created.
func Factory() (linguistics_domain.PhoneticEncoderPort, error) {
	return New()
}

// New creates a new English Double Metaphone encoder.
//
// Returns *Encoder which is ready for use.
// Returns error which is always nil for this encoder.
func New() (*Encoder, error) {
	return NewWithMaxLength(linguistics_domain.DefaultPhoneticCodeLength)
}

func init() {
	linguistics_domain.RegisterPhoneticEncoderFactory(Language, Factory)
}
