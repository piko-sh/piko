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

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Normaliser handles text normalisation operations like lowercasing and
// diacritic removal. It implements NormaliserPort and is safe for concurrent
// use.
type Normaliser struct {
	// preserveCase indicates whether text case should be preserved during
	// normalisation; when false, text is converted to lowercase.
	preserveCase bool
}

// NewNormaliser creates a new text normaliser with the given settings.
//
// Takes preserveCase (bool) which controls whether to keep the original case.
//
// Returns *Normaliser which is the configured normaliser ready for use.
func NewNormaliser(preserveCase bool) *Normaliser {
	return &Normaliser{
		preserveCase: preserveCase,
	}
}

// Normalise applies all normalisation changes to the input text.
//
// Takes text (string) which is the input to normalise.
//
// Returns string which is the normalised version ready for indexing and
// searching.
func (n *Normaliser) Normalise(text string) string {
	transformer := getDiacriticRemover()

	normalised, _, err := transform.String(transformer, text)
	if err != nil {
		normalised = text
	}

	if !n.preserveCase {
		normalised = strings.ToLower(normalised)
	}

	return normalised
}

// NormaliseRune normalises a single rune.
//
// Takes r (rune) which is the character to normalise.
//
// Returns rune which is the normalised character, lowercased if case
// preservation is disabled.
func (n *Normaliser) NormaliseRune(r rune) rune {
	if !n.preserveCase {
		r = unicode.ToLower(r)
	}
	return r
}

// getDiacriticRemover creates a transformer that removes diacritics from text.
// A new transformer is created for each call to ensure thread safety.
//
// Returns transform.Transformer which removes diacritics from Unicode text.
func getDiacriticRemover() transform.Transformer {
	return transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)),
		norm.NFC,
	)
}

// isWordChar reports whether the rune is part of a word.
// Letters, numbers, underscores, and hyphens are considered word characters.
//
// Takes r (rune) which is the character to check.
//
// Returns bool which is true if the rune is a word character.
func isWordChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' || r == '-'
}

// isSeparator reports whether the rune is a word separator.
//
// Takes r (rune) which is the character to check.
//
// Returns bool which is true if the rune is whitespace or punctuation.
func isSeparator(r rune) bool {
	return unicode.IsSpace(r) || unicode.IsPunct(r)
}
