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

package layouter_domain

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// applyTextTransform returns the text with the CSS text-transform
// property applied.
//
// For uppercase and lowercase the entire string is converted using
// Unicode-aware case mapping. For capitalise, the first letter of each
// word is title-cased using language-undetermined rules per the CSS
// specification.
//
// Takes text (string) which is the source text.
// Takes transform (TextTransformType) which is the transform to apply.
//
// Returns string which is the transformed text, or the original text
// when the transform is none.
func applyTextTransform(text string, transform TextTransformType) string {
	switch transform {
	case TextTransformUppercase:
		return strings.ToUpper(text)
	case TextTransformLowercase:
		return strings.ToLower(text)
	case TextTransformCapitalise:
		return cases.Title(language.Und).String(text)
	default:
		return text
	}
}
