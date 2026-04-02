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

package i18n_domain

import "strings"

// pluralCategory represents a CLDR plural category used to select the correct
// plural form when formatting localised text.
//
// See https://cldr.unicode.org/index/cldr-spec/plural-rules
type pluralCategory uint8

const (
	// pluralZero is the plural category for zero quantities.
	pluralZero pluralCategory = iota

	// pluralOne represents the singular form category for count of one.
	pluralOne

	// pluralTwo is the plural category for counts of exactly two.
	pluralTwo

	// pluralFew is the plural category for small quantities in certain languages.
	pluralFew

	// pluralMany is the plural category for large quantities in certain languages.
	pluralMany

	// pluralOther is the default plural category for quantities that do not match
	// other specific plural rules.
	pluralOther
)

const (
	// mod10Divisor is used to get the last digit of a number.
	mod10Divisor = 10

	// mod100Divisor is the divisor used to get the last two digits of a number.
	mod100Divisor = 100

	// slavicMod11 is the mod100 value that excludes 11 from the singular form.
	slavicMod11 = 11

	// slavicFewStart is the lower bound for the Slavic "few" plural form range.
	slavicFewStart = 2

	// slavicFewEnd is the upper bound for the few plural category in Slavic rules.
	slavicFewEnd = 4

	// slavicMod12 is the lower bound for the mod 100 range that excludes the few
	// plural form in Slavic languages.
	slavicMod12 = 12

	// slavicMod14 is the upper bound for the mod100 range check in Slavic plurals.
	slavicMod14 = 14

	// arabicFewStart is the start of the Arabic few plural range.
	arabicFewStart = 3

	// arabicFewEnd is the upper limit for the Arabic few plural form.
	arabicFewEnd = 10

	// arabicManyEnd is the upper limit for the Arabic plural many category.
	arabicManyEnd = 99

	// formCount3 is three plural forms: zero/one, few, and other.
	formCount3 = 3

	// formCount4 is a plural form count for languages with four grammatical forms.
	formCount4 = 4

	// formCount6 is the full CLDR plural form count: zero, one, two, few, many, other.
	formCount6 = 6
)

// SplitPluralForms splits a template by pipe separator into plural forms.
//
// Takes template (string) which contains pipe-separated plural forms.
//
// Returns []string which contains the separated plural forms.
//
// For example: "one item|{count} items" returns ["one item", "{count} items"].
// Escaped pipes (||) are preserved as literal pipes.
func SplitPluralForms(template string) []string {
	if len(template) == 0 {
		return nil
	}

	var forms []string
	var current strings.Builder
	i := 0

	for i < len(template) {
		if template[i] == '|' {
			if i+1 < len(template) && template[i+1] == '|' {
				_ = current.WriteByte('|')
				i += 2
				continue
			}
			forms = append(forms, current.String())
			current.Reset()
			i++
		} else {
			_ = current.WriteByte(template[i])
			i++
		}
	}

	forms = append(forms, current.String())

	return forms
}

// HasPluralForms reports whether the template contains plural forms.
//
// Plural forms are indicated by unescaped pipe (|) separators. Escaped
// pipes (||) are not counted as plural form separators.
//
// Takes template (string) which is the template string to check.
//
// Returns bool which is true if the template contains plural forms.
func HasPluralForms(template string) bool {
	for i := 0; i < len(template); i++ {
		if template[i] == '|' {
			if i+1 < len(template) && template[i+1] == '|' {
				i++
				continue
			}
			return true
		}
	}
	return false
}

// SelectPluralForm selects the appropriate plural form based on count and
// locale. It uses simplified CLDR rules for common locales.
//
// Takes count (int) which specifies the number to determine plural form for.
// Takes locale (string) which identifies the language rules to apply.
// Takes forms ([]string) which provides the available plural form strings.
//
// Returns string which is the selected plural form, or empty if forms is empty.
func SelectPluralForm(count int, locale string, forms []string) string {
	if len(forms) == 0 {
		return ""
	}
	if len(forms) == 1 {
		return forms[0]
	}

	index := selectPluralFormIndex(count, locale, len(forms))
	return forms[index]
}

// selectPluralFormIndex returns the index of the correct plural form based on
// count and locale. This is used for zero-allocation access to pre-parsed
// plural forms.
//
// When formCount is zero or one, returns zero straight away.
//
// Takes count (int) which is the number to select the plural form for.
// Takes locale (string) which sets the language rules to apply.
// Takes formCount (int) which is the total number of plural forms available.
//
// Returns int which is the index into the plural forms array.
func selectPluralFormIndex(count int, locale string, formCount int) int {
	if formCount == 0 {
		return 0
	}
	if formCount == 1 {
		return 0
	}

	category := getPluralCategory(count, locale)
	index := categoryToIndex(category, formCount)

	if index >= formCount {
		return formCount - 1
	}
	return index
}

// getPluralCategory finds the CLDR plural category for a number and locale.
//
// Takes count (int) which is the number to find the plural form for.
// Takes locale (string) which is the language code (e.g. "en", "en-GB").
//
// Returns pluralCategory which is the correct plural form to use.
func getPluralCategory(count int, locale string) pluralCategory {
	baseLang := locale
	if lang, _, found := strings.Cut(locale, "-"); found {
		baseLang = lang
	}

	switch baseLang {
	case "fr":
		if count == 0 || count == 1 {
			return pluralOne
		}
		return pluralOther

	case "ru", "uk", "be", "sr", "hr", "bs":
		return getSlavicPlural(count)

	case "pl":
		return getPolishPlural(count)

	case "ar":
		return getArabicPlural(count)

	case "zh", "ja", "ko", "vi", "th", "id", "ms":
		return pluralOther

	default:
		return getSimplePlural(count)
	}
}

// getSimplePlural returns the plural category for languages with simple
// one/other rules. This covers English, German, Dutch, Swedish, Danish,
// Norwegian, Finnish, Estonian, Hungarian, Italian, Spanish, Portuguese,
// Greek, Hebrew, Bulgarian, Turkish, and many others.
//
// Takes count (int) which is the number to check for plurality.
//
// Returns pluralCategory which is pluralOne when count equals 1, or
// pluralOther for all other values.
func getSimplePlural(count int) pluralCategory {
	if count == 1 {
		return pluralOne
	}
	return pluralOther
}

// getSlavicPlural returns the plural category for Slavic languages such as
// Russian.
//
// Takes count (int) which is the number to find the plural form for.
//
// Returns pluralCategory which is one of pluralOne, pluralFew, or pluralMany
// based on Slavic plural rules.
func getSlavicPlural(count int) pluralCategory {
	abs := count
	if abs < 0 {
		abs = -abs
	}

	mod10 := abs % mod10Divisor
	mod100 := abs % mod100Divisor

	if mod10 == 1 && mod100 != slavicMod11 {
		return pluralOne
	}
	if mod10 >= slavicFewStart && mod10 <= slavicFewEnd && (mod100 < slavicMod12 || mod100 > slavicMod14) {
		return pluralFew
	}
	return pluralMany
}

// getPolishPlural returns the plural category for Polish language.
//
// Takes count (int) which is the number to find the plural form for.
//
// Returns pluralCategory which is pluralOne, pluralFew, or pluralMany.
func getPolishPlural(count int) pluralCategory {
	abs := count
	if abs < 0 {
		abs = -abs
	}

	if abs == 1 {
		return pluralOne
	}

	mod10 := abs % mod10Divisor
	mod100 := abs % mod100Divisor

	if mod10 >= slavicFewStart && mod10 <= slavicFewEnd && (mod100 < slavicMod12 || mod100 > slavicMod14) {
		return pluralFew
	}
	return pluralMany
}

// getArabicPlural returns the plural category for Arabic.
//
// Takes count (int) which is the number to categorise.
//
// Returns pluralCategory which indicates the plural form to use based on
// Arabic plural rules.
func getArabicPlural(count int) pluralCategory {
	abs := count
	if abs < 0 {
		abs = -abs
	}

	if abs == 0 {
		return pluralZero
	}
	if abs == 1 {
		return pluralOne
	}
	if abs == 2 {
		return pluralTwo
	}

	mod100 := abs % mod100Divisor
	if mod100 >= arabicFewStart && mod100 <= arabicFewEnd {
		return pluralFew
	}
	if mod100 >= slavicMod11 && mod100 <= arabicManyEnd {
		return pluralMany
	}
	return pluralOther
}

// categoryToIndex maps a plural category to an array index based on the
// number of available forms.
//
// Takes category (pluralCategory) which is the CLDR plural category to map.
// Takes formCount (int) which is the number of plural forms available.
//
// Returns int which is the array index for the given category.
func categoryToIndex(category pluralCategory, formCount int) int {
	switch formCount {
	case 2:
		if category == pluralOne {
			return 0
		}
		return 1

	case formCount3:
		if category == pluralZero || category == pluralOne {
			return 0
		}
		if category == pluralFew {
			return 1
		}
		return 2

	case formCount4:
		switch category {
		case pluralOne:
			return 0
		case pluralFew:
			return 1
		case pluralMany:
			return 2
		default:
			return formCount3
		}

	case formCount6:
		return int(category)

	default:
		return int(category) % formCount
	}
}
