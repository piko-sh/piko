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

package maths

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	pikojson "piko.sh/piko/internal/json"
)

// String returns the decimal value as a text string.
//
// Returns string which is the decimal formatted as text.
// Returns error when the decimal is in an error state.
func (d Decimal) String() (string, error) {
	if d.err != nil {
		return "", d.err
	}
	return d.value.Text('f'), nil
}

// Float64 returns the decimal as a float64 value.
//
// This conversion may lose precision.
//
// Returns float64 which is the decimal value as a floating-point number.
// Returns error when the decimal is in an error state.
func (d Decimal) Float64() (float64, error) {
	if d.err != nil {
		return 0, d.err
	}
	s, err := d.String()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

// Int64 returns the whole number value of the decimal as an int64.
//
// Returns int64 which is the whole number value.
// Returns error when the decimal is in an error state or has a
// fractional part.
func (d Decimal) Int64() (int64, error) {
	if d.err != nil {
		return 0, d.err
	}
	isInteger, err := d.IsInteger()
	if err != nil {
		return 0, err
	}
	if !isInteger {
		return 0, errors.New("maths: cannot convert decimal with fractional part to int64")
	}
	return d.value.Int64()
}

// MustString returns the string representation of the decimal.
//
// Returns string which is the formatted decimal value.
//
// Panics when the string conversion fails.
func (d Decimal) MustString() string {
	s, err := d.String()
	if err != nil {
		panic(err)
	}
	return s
}

// MustFloat64 returns the float64 representation of the decimal.
//
// This conversion can result in a loss of precision.
//
// Returns float64 which is the decimal value as a floating-point number.
//
// Panics when the conversion fails.
func (d Decimal) MustFloat64() float64 {
	f, err := d.Float64()
	if err != nil {
		panic(err)
	}
	return f
}

// MustInt64 returns the int64 representation of the decimal.
//
// Returns int64 which is the whole number value of the decimal.
//
// Panics when an error occurs, such as when the decimal has a fractional part.
func (d Decimal) MustInt64() int64 {
	i, err := d.Int64()
	if err != nil {
		panic(err)
	}
	return i
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the decimal as a JSON string to preserve precision.
//
// Returns []byte which contains the JSON-encoded string representation.
// Returns error when the decimal is in an error state or string conversion
// fails.
func (d Decimal) MarshalJSON() ([]byte, error) {
	if d.err != nil {
		return nil, d.err
	}
	s, err := d.String()
	if err != nil {
		return nil, err
	}
	return pikojson.Marshal(s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It can unmarshal from a JSON string or a JSON number.
//
// Takes data ([]byte) which contains the JSON-encoded value to parse.
//
// Returns error when the receiver is nil or the data is not a valid JSON
// string or number.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	if d == nil {
		return errors.New("maths: UnmarshalJSON on nil Decimal pointer")
	}

	var numValue json.Number
	if err := pikojson.Unmarshal(data, &numValue); err == nil {
		*d = NewDecimalFromString(numValue.String())
		return d.err
	}

	var strValue string
	if err := pikojson.Unmarshal(data, &strValue); err == nil {
		*d = NewDecimalFromString(strValue)
		return d.err
	}

	var anyValue any
	if err := pikojson.Unmarshal(data, &anyValue); err == nil {
		switch v := anyValue.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			*d = NewDecimalFromString(fmt.Sprint(v))
			return d.err
		}
	}

	return errors.New("maths: decimal must be a JSON string or number")
}

// Value implements the driver.Valuer interface for database serialisation.
// It returns the decimal as a string for storage in a high-precision
// database column.
//
// Returns driver.Value which contains the decimal as a string.
// Returns error when the decimal holds a previous error.
func (d Decimal) Value() (driver.Value, error) {
	if d.err != nil {
		return nil, d.err
	}
	return d.String()
}

// Scan implements the sql.Scanner interface for database deserialisation.
//
// Takes source (interface{}) which is the database value to scan; accepts string,
// []byte, int64, float64, nil, or any type implementing fmt.Stringer
// (for example duckdb.Decimal).
//
// Returns error when d is nil or source is an unsupported type.
func (d *Decimal) Scan(source any) error {
	if d == nil {
		return errors.New("maths: Scan on nil Decimal pointer")
	}

	var temp Decimal
	switch v := source.(type) {
	case string:
		temp = NewDecimalFromString(v)
	case []byte:
		temp = NewDecimalFromString(string(v))
	case int64:
		temp = NewDecimalFromInt(v)
	case float64:
		temp = NewDecimalFromFloat(v)
	case nil:
		temp = ZeroDecimal()
	case fmt.Stringer:
		temp = NewDecimalFromString(v.String())
	default:
		s, ok := stringerFromReflect(source)
		if !ok {
			return fmt.Errorf("maths: cannot scan type %T into Decimal", source)
		}

		temp = NewDecimalFromString(s)
	}

	*d = temp
	return d.err
}

// stringerFromReflect attempts to call a String() string method on source,
// including pointer-receiver methods that a type switch cannot reach when
// the value is stored in an interface.
//
// Takes source (any) which is the value to try calling String() on.
//
// Returns string which is the result of the String() call.
// Returns bool which is true when the method was found and called successfully.
func stringerFromReflect(source any) (string, bool) {
	v := reflect.ValueOf(source)
	if !v.IsValid() {
		return "", false
	}

	m := v.MethodByName("String")
	if !m.IsValid() && v.CanAddr() {
		m = v.Addr().MethodByName("String")
	}

	if !m.IsValid() {
		ptr := reflect.New(v.Type())
		ptr.Elem().Set(v)
		m = ptr.MethodByName("String")
	}
	if !m.IsValid() {
		return "", false
	}

	mt := m.Type()
	if mt.NumIn() != 0 || mt.NumOut() != 1 || mt.Out(0).Kind() != reflect.String {
		return "", false
	}

	result := m.Call(nil)
	return result[0].String(), true
}

// NumberLocale defines the formatting rules for displaying numbers in a
// specific locale.
type NumberLocale struct {
	// DecimalSep is the character that separates the whole number from the
	// decimal part (e.g., "." or ",").
	DecimalSep string

	// ThousandSep is the character used to separate groups of digits
	// (e.g. ",", ".", "'", or " "); an empty string means no grouping.
	ThousandSep string

	// GroupSize is the number of digits in the primary (rightmost) group;
	// typically 3.
	GroupSize int

	// SecondaryGroupSize is the number of digits in groups after the first.
	// Set to 0 to use GroupSize for all groups (Western-style), or 2 for
	// Indian-style grouping (e.g., 1,23,45,678).
	SecondaryGroupSize int
}

var (
	// LocaleEnGB uses "." for decimal and "," for thousands (e.g., 1,234.56).
	// Used by: en-GB, en-AU, en-NZ, en-IE, en-ZA, ja, zh, ko, th.
	LocaleEnGB = NumberLocale{DecimalSep: ".", ThousandSep: ",", GroupSize: 3}

	// LocaleEnUS uses "." for decimal and "," for thousands (e.g., 1,234.56).
	// Identical to LocaleEnGB but kept separate for semantic clarity.
	LocaleEnUS = NumberLocale{DecimalSep: ".", ThousandSep: ",", GroupSize: 3}

	// LocaleDeDe uses "," for decimal and "." for thousands (e.g., 1.234,56).
	// Used by: de, es, it, pt, nl, el, tr, hr, ro, sr, sl, vi, id.
	LocaleDeDe = NumberLocale{DecimalSep: ",", ThousandSep: ".", GroupSize: 3}

	// LocaleFrFr uses "," for decimal and narrow no-break space (U+202F) for
	// thousands (e.g., 1 234,56).
	// Used by: fr, sv, nb, nn, da, fi, pl, ru, uk, be, cs, sk, bg, hu.
	LocaleFrFr = NumberLocale{DecimalSep: ",", ThousandSep: "\u202f", GroupSize: 3}

	// LocaleSwiss uses "." for decimal and apostrophe "'" for thousands
	// (e.g., 1'234.56).
	// Used by: de-CH, fr-CH, it-CH, rm-CH.
	LocaleSwiss = NumberLocale{DecimalSep: ".", ThousandSep: "'", GroupSize: 3}

	// LocaleIndian defines Indian number formatting with lakh and crore grouping.
	//
	// The first group from the right has 3 digits, subsequent groups have
	// 2 digits (e.g., 1,23,45,678.90). Uses "." for decimals and "," for
	// thousands. Supported locales: en-IN, hi, bn, ta, te, mr, gu, kn, ml,
	// pa, or, as, ne.
	LocaleIndian = NumberLocale{
		DecimalSep:         ".",
		ThousandSep:        ",",
		GroupSize:          3,
		SecondaryGroupSize: 2,
	}

	// LocaleRaw uses no thousand separator and "." for decimal.
	// Useful for machine-readable output or when no formatting is desired.
	LocaleRaw = NumberLocale{DecimalSep: ".", ThousandSep: "", GroupSize: 0}

	// localeMap maps locale strings to NumberLocale settings.
	// Organised by number format family for clarity.
	localeMap = map[string]NumberLocale{
		"en":    LocaleEnGB,
		"en-GB": LocaleEnGB, // English (Traditional)
		"en-US": LocaleEnUS, // English (Simplified)
		"en-AU": LocaleEnGB, // Australia
		"en-CA": LocaleEnGB, // Canada (English)
		"en-NZ": LocaleEnGB, // New Zealand
		"en-IE": LocaleEnGB, // Ireland
		"en-ZA": LocaleEnGB, // South Africa
		"en-SG": LocaleEnGB, // Singapore
		"en-HK": LocaleEnGB, // Hong Kong
		"en-PH": LocaleEnGB, // Philippines
		"ja":    LocaleEnGB, // Japanese
		"ja-JP": LocaleEnGB,
		"zh":    LocaleEnGB, // Chinese (Simplified)
		"zh-CN": LocaleEnGB,
		"zh-TW": LocaleEnGB, // Chinese (Traditional)
		"zh-HK": LocaleEnGB, // Chinese (Hong Kong)
		"ko":    LocaleEnGB, // Korean
		"ko-KR": LocaleEnGB,
		"th":    LocaleEnGB, // Thai
		"th-TH": LocaleEnGB,
		"ms":    LocaleEnGB, // Malay
		"ms-MY": LocaleEnGB,
		"de":    LocaleDeDe, // German
		"de-DE": LocaleDeDe,
		"de-AT": LocaleDeDe, // Austria
		"es":    LocaleDeDe, // Spanish
		"es-ES": LocaleDeDe,
		"es-MX": LocaleDeDe, // Mexico
		"es-AR": LocaleDeDe, // Argentina
		"es-CO": LocaleDeDe, // Colombia
		"es-CL": LocaleDeDe, // Chile
		"es-PE": LocaleDeDe, // Peru
		"es-VE": LocaleDeDe, // Venezuela
		"it":    LocaleDeDe, // Italian
		"it-IT": LocaleDeDe,
		"pt":    LocaleDeDe, // Portuguese
		"pt-BR": LocaleDeDe, // Brazil
		"pt-PT": LocaleDeDe, // Portugal
		"nl":    LocaleDeDe, // Dutch
		"nl-NL": LocaleDeDe, // Netherlands
		"nl-BE": LocaleDeDe, // Belgium (Dutch)
		"el":    LocaleDeDe, // Greek
		"el-GR": LocaleDeDe,
		"tr":    LocaleDeDe, // Turkish
		"tr-TR": LocaleDeDe,
		"hr":    LocaleDeDe, // Croatian
		"hr-HR": LocaleDeDe,
		"sr":    LocaleDeDe, // Serbian
		"sr-RS": LocaleDeDe,
		"sl":    LocaleDeDe, // Slovenian
		"sl-SI": LocaleDeDe,
		"bs":    LocaleDeDe, // Bosnian
		"bs-BA": LocaleDeDe,
		"mk":    LocaleDeDe, // Macedonian
		"mk-MK": LocaleDeDe,
		"ro":    LocaleDeDe, // Romanian
		"ro-RO": LocaleDeDe,
		"vi":    LocaleDeDe, // Vietnamese
		"vi-VN": LocaleDeDe,
		"id":    LocaleDeDe, // Indonesian
		"id-ID": LocaleDeDe,
		"fr":    LocaleFrFr, // French
		"fr-FR": LocaleFrFr,
		"fr-CA": LocaleFrFr, // Canada (French)
		"fr-BE": LocaleFrFr, // Belgium (French)
		"sv":    LocaleFrFr, // Swedish
		"sv-SE": LocaleFrFr,
		"nb":    LocaleFrFr, // Norwegian Bokmål
		"nb-NO": LocaleFrFr,
		"nn":    LocaleFrFr, // Norwegian Nynorsk
		"nn-NO": LocaleFrFr,
		"no":    LocaleFrFr, // Norwegian (generic)
		"no-NO": LocaleFrFr,
		"da":    LocaleFrFr, // Danish
		"da-DK": LocaleFrFr,
		"fi":    LocaleFrFr, // Finnish
		"fi-FI": LocaleFrFr,
		"is":    LocaleFrFr, // Icelandic
		"is-IS": LocaleFrFr,
		"pl":    LocaleFrFr, // Polish
		"pl-PL": LocaleFrFr,
		"ru":    LocaleFrFr, // Russian
		"ru-RU": LocaleFrFr,
		"uk":    LocaleFrFr, // Ukrainian
		"uk-UA": LocaleFrFr,
		"be":    LocaleFrFr, // Belarusian
		"be-BY": LocaleFrFr,
		"cs":    LocaleFrFr, // Czech
		"cs-CZ": LocaleFrFr,
		"sk":    LocaleFrFr, // Slovak
		"sk-SK": LocaleFrFr,
		"bg":    LocaleFrFr, // Bulgarian
		"bg-BG": LocaleFrFr,
		"hu":    LocaleFrFr, // Hungarian
		"hu-HU": LocaleFrFr,
		"lt":    LocaleFrFr, // Lithuanian
		"lt-LT": LocaleFrFr,
		"lv":    LocaleFrFr, // Latvian
		"lv-LV": LocaleFrFr,
		"et":    LocaleFrFr, // Estonian
		"et-EE": LocaleFrFr,
		"de-CH": LocaleSwiss, // Swiss German
		"fr-CH": LocaleSwiss, // Swiss French
		"it-CH": LocaleSwiss, // Swiss Italian
		"rm":    LocaleSwiss, // Romansh
		"rm-CH": LocaleSwiss,
		"en-IN": LocaleIndian, // Indian English
		"hi":    LocaleIndian, // Hindi
		"hi-IN": LocaleIndian,
		"bn":    LocaleIndian, // Bengali
		"bn-IN": LocaleIndian,
		"bn-BD": LocaleIndian, // Bangladesh
		"ta":    LocaleIndian, // Tamil
		"ta-IN": LocaleIndian,
		"te":    LocaleIndian, // Telugu
		"te-IN": LocaleIndian,
		"mr":    LocaleIndian, // Marathi
		"mr-IN": LocaleIndian,
		"gu":    LocaleIndian, // Gujarati
		"gu-IN": LocaleIndian,
		"kn":    LocaleIndian, // Kannada
		"kn-IN": LocaleIndian,
		"ml":    LocaleIndian, // Malayalam
		"ml-IN": LocaleIndian,
		"pa":    LocaleIndian, // Punjabi
		"pa-IN": LocaleIndian,
		"or":    LocaleIndian, // Odia
		"or-IN": LocaleIndian,
		"as":    LocaleIndian, // Assamese
		"as-IN": LocaleIndian,
		"ne":    LocaleIndian, // Nepali
		"ne-NP": LocaleIndian,
		"si":    LocaleIndian, // Sinhala
		"si-LK": LocaleIndian,
	}
)

// Format returns the decimal as a string with locale-specific formatting.
// This keeps full precision while using the correct separators for the locale.
//
// Takes locale (string) which specifies the locale code (e.g. "de-DE",
// "en-GB").
//
// Returns string which is the formatted decimal with locale-specific
// separators.
// Returns error when the decimal has a previous error or cannot be converted
// to a string.
//
// Example:
// d := NewDecimalFromString("1234567.89012345")
// d.Format("de-DE") // Returns "1.234.567,89012345"
// d.Format("en-GB") // Returns "1,234,567.89012345"
func (d Decimal) Format(locale string) (string, error) {
	if d.err != nil {
		return "", d.err
	}

	s, err := d.String()
	if err != nil {
		return "", err
	}

	return FormatNumberString(s, GetNumberLocale(locale)), nil
}

// MustFormat returns the locale-formatted string representation.
//
// Takes locale (string) which specifies the locale for formatting.
//
// Returns string which is the formatted decimal value.
//
// Panics when the locale is invalid or formatting fails.
func (d Decimal) MustFormat(locale string) string {
	s, err := d.Format(locale)
	if err != nil {
		panic(err)
	}
	return s
}

// GetNumberLocale returns the number formatting rules for a given locale.
// Falls back to LocaleEnGB if the locale is not found.
//
// Takes locale (string) which specifies the locale identifier to look up.
//
// Returns NumberLocale which contains the number formatting rules for the
// requested locale.
func GetNumberLocale(locale string) NumberLocale {
	if l, ok := localeMap[locale]; ok {
		return l
	}
	return LocaleEnGB
}

// FormatNumberString applies locale formatting to a number string. It
// preserves full precision while adding thousand separators and the correct
// decimal separator.
//
// Takes s (string) which is the number string to format.
// Takes locale (NumberLocale) which specifies the formatting rules.
//
// Returns string which is the formatted number with locale-specific
// separators.
func FormatNumberString(s string, locale NumberLocale) string {
	if s == "" {
		return s
	}

	negative := false
	if s[0] == '-' {
		negative = true
		s = s[1:]
	}

	intPart := s
	fracPart := ""
	if dotIndex := indexOf(s, '.'); dotIndex != -1 {
		intPart = s[:dotIndex]
		fracPart = s[dotIndex+1:]
	}

	if locale.GroupSize > 0 && locale.ThousandSep != "" && len(intPart) > locale.GroupSize {
		intPart = addThousandSeparators(intPart, locale.ThousandSep, locale.GroupSize, locale.SecondaryGroupSize)
	}

	var result string
	if negative {
		result = "-"
	}
	result += intPart
	if fracPart != "" {
		result += locale.DecimalSep + fracPart
	}

	return result
}

// indexOf returns the index of the first occurrence of char in s, or -1 if
// not found.
//
// Takes s (string) which is the string to search.
// Takes char (byte) which is the character to find.
//
// Returns int which is the index of the first match, or -1 if not found.
func indexOf(s string, char byte) int {
	for i := range len(s) {
		if s[i] == char {
			return i
		}
	}
	return -1
}

// addThousandSeparators inserts separators into a number string to improve
// readability. Supports variable group sizes for Indian-style lakh and crore
// grouping.
//
// Takes s (string) which is the number string to format.
// Takes sep (string) which is the separator to insert between groups.
// Takes primaryGroupSize (int) which is the number of digits in the rightmost
// group.
// Takes secondaryGroupSize (int) which is the number of digits in later
// groups. If 0, uses primaryGroupSize for all groups (standard Western style).
//
// Returns string which is the formatted string with separators added.
//
// Examples:
// addThousandSeparators("1234567", ",", 3, 0) returns "1,234,567"
// addThousandSeparators("1234567", ",", 3, 2) returns "12,34,567" (Indian)
func addThousandSeparators(s, sep string, primaryGroupSize, secondaryGroupSize int) string {
	n := len(s)
	if n <= primaryGroupSize || primaryGroupSize <= 0 {
		return s
	}

	if secondaryGroupSize <= 0 {
		secondaryGroupSize = primaryGroupSize
	}

	remaining := n - primaryGroupSize
	numSeps := (remaining + secondaryGroupSize - 1) / secondaryGroupSize

	result := make([]byte, n+numSeps*len(sep))

	j := len(result) - 1
	count := 0
	currentGroupSize := primaryGroupSize
	firstGroupDone := false

	for i := n - 1; i >= 0; i-- {
		result[j] = s[i]
		j--
		count++

		if count == currentGroupSize && i > 0 {
			for k := len(sep) - 1; k >= 0; k-- {
				result[j] = sep[k]
				j--
			}
			count = 0

			if !firstGroupDone {
				currentGroupSize = secondaryGroupSize
				firstGroupDone = true
			}
		}
	}

	return string(result)
}
