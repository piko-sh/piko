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
	"reflect"
	"testing"

	"piko.sh/piko/internal/json"
)

func TestValueRetrievers(t *testing.T) {
	dValid := NewDecimalFromString("123.45")
	dInteger := NewDecimalFromString("789.0")
	dInvalid := NewDecimalFromString("invalid")

	t.Run("String", func(t *testing.T) {
		s, err := dValid.String()
		if err != nil || s != "123.45" {
			t.Errorf("expected String() to be '123.45', got %q with error %v", s, err)
		}
		_, err = dInvalid.String()
		if err == nil {
			t.Error("expected error from String() for invalid decimal")
		}
	})

	t.Run("Float64", func(t *testing.T) {
		f, err := dValid.Float64()
		if err != nil || f != 123.45 {
			t.Errorf("expected Float64() to be 123.45, got %f with error %v", f, err)
		}
		_, err = dInvalid.Float64()
		if err == nil {
			t.Error("expected error from Float64() for invalid decimal")
		}
	})

	t.Run("Int64", func(t *testing.T) {
		i, err := dInteger.Int64()
		if err != nil || i != 789 {
			t.Errorf("expected Int64() to be 789, got %d with error %v", i, err)
		}
		_, err = dValid.Int64()
		if err == nil {
			t.Error("expected error converting non-integer to Int64")
		}
		_, err = dInvalid.Int64()
		if err == nil {
			t.Error("expected error from Int64() for invalid decimal")
		}
	})
}

func TestMustRetrievers(t *testing.T) {
	dValid := NewDecimalFromString("123.45")
	dInteger := NewDecimalFromInt(789)
	dInvalid := NewDecimalFromString("invalid")

	t.Run("MustString", func(t *testing.T) {
		if dValid.MustString() != "123.45" {
			t.Error("MustString failed for valid decimal")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustString to panic on an error")
			}
		}()
		_ = dInvalid.MustString()
	})

	t.Run("MustFloat64", func(t *testing.T) {
		if dValid.MustFloat64() != 123.45 {
			t.Error("MustFloat64 failed for valid decimal")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustFloat64 to panic on an error")
			}
		}()
		_ = dInvalid.MustFloat64()
	})

	t.Run("MustInt64", func(t *testing.T) {
		if dInteger.MustInt64() != 789 {
			t.Error("MustInt64 failed for valid integer decimal")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustInt64 to panic on an error for non-integer")
			}
		}()
		_ = dValid.MustInt64()
	})
}

func TestEncoding(t *testing.T) {
	t.Run("JSON Marshal", func(t *testing.T) {
		d := NewDecimalFromString("123.45")
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}

		if string(b) != `"123.45"` {
			t.Errorf(`expected marshalled JSON to be '"123.45"', got %s`, string(b))
		}

		dInvalid := NewDecimalFromString("invalid")
		_, err = json.Marshal(dInvalid)
		if err == nil {
			t.Error("expected error marshalling an invalid decimal")
		}
	})

	t.Run("JSON Unmarshal", func(t *testing.T) {
		var d1 Decimal
		err := json.Unmarshal([]byte(`"543.21"`), &d1)
		if err != nil {
			t.Fatalf("UnmarshalJSON from string failed: %v", err)
		}
		checkDecimal(t, d1, "543.21", false)

		var d2 Decimal
		err = json.Unmarshal([]byte(`987.65`), &d2)
		if err != nil {
			t.Fatalf("UnmarshalJSON from number failed: %v", err)
		}
		checkDecimal(t, d2, "987.65", false)

		var d3 Decimal
		err = json.Unmarshal([]byte(`"invalid"`), &d3)
		if err == nil {
			t.Error("expected error unmarshalling an invalid string")
		}

		var d4 Decimal
		err = json.Unmarshal([]byte(`{"key":"value"}`), &d4)
		if err == nil {
			t.Error("expected error unmarshalling from an object")
		}
	})

	t.Run("SQL Scan and Value", func(t *testing.T) {
		d := NewDecimalFromString("123.45")
		v, err := d.Value()
		if err != nil {
			t.Fatalf("Value() failed: %v", err)
		}
		if !reflect.DeepEqual(v, "123.45") {
			t.Errorf("expected Value() to be '123.45', got %v", v)
		}

		var scannedD Decimal
		if err := scannedD.Scan(v); err != nil {
			t.Fatalf("Scan() from string value failed: %v", err)
		}
		checkDecimal(t, scannedD, "123.45", false)
	})

	t.Run("SQL Scan Types", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    any
			expected string
		}{
			{name: "string", input: "987.65", expected: "987.65"},
			{name: "[]byte", input: []byte("543.21"), expected: "543.21"},
			{name: "int64", input: int64(123), expected: "123"},
			{name: "float64", input: float64(45.67), expected: "45.67"},
			{name: "nil", input: nil, expected: "0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var d Decimal
				err := d.Scan(tc.input)
				if err != nil {
					t.Fatalf("Scan failed for type %s: %v", tc.name, err)
				}
				checkDecimal(t, d, tc.expected, false)
			})
		}
	})

	t.Run("SQL Scan Invalid Type", func(t *testing.T) {
		var d Decimal
		err := d.Scan(true)
		if err == nil {
			t.Error("expected an error when scanning an unsupported type")
		}
	})

	t.Run("SQL Value with Error", func(t *testing.T) {
		dInvalid := NewDecimalFromString("invalid")
		_, err := dInvalid.Value()
		if err == nil {
			t.Error("expected error from Value() for an invalid decimal")
		}
	})
}

func TestGetNumberLocale(t *testing.T) {
	testCases := []struct {
		name     string
		locale   string
		expected NumberLocale
	}{

		{name: "en", locale: "en", expected: LocaleEnGB},
		{name: "en-GB", locale: "en-GB", expected: LocaleEnGB},
		{name: "en-US", locale: "en-US", expected: LocaleEnUS},
		{name: "en-AU", locale: "en-AU", expected: LocaleEnGB},
		{name: "ja-JP", locale: "ja-JP", expected: LocaleEnGB},
		{name: "zh-CN", locale: "zh-CN", expected: LocaleEnGB},
		{name: "ko-KR", locale: "ko-KR", expected: LocaleEnGB},

		{name: "de", locale: "de", expected: LocaleDeDe},
		{name: "de-DE", locale: "de-DE", expected: LocaleDeDe},
		{name: "es-ES", locale: "es-ES", expected: LocaleDeDe},
		{name: "it-IT", locale: "it-IT", expected: LocaleDeDe},
		{name: "pt-BR", locale: "pt-BR", expected: LocaleDeDe},
		{name: "nl-NL", locale: "nl-NL", expected: LocaleDeDe},

		{name: "fr", locale: "fr", expected: LocaleFrFr},
		{name: "fr-FR", locale: "fr-FR", expected: LocaleFrFr},
		{name: "sv-SE", locale: "sv-SE", expected: LocaleFrFr},
		{name: "pl-PL", locale: "pl-PL", expected: LocaleFrFr},
		{name: "ru-RU", locale: "ru-RU", expected: LocaleFrFr},

		{name: "de-CH", locale: "de-CH", expected: LocaleSwiss},
		{name: "fr-CH", locale: "fr-CH", expected: LocaleSwiss},
		{name: "it-CH", locale: "it-CH", expected: LocaleSwiss},

		{name: "en-IN", locale: "en-IN", expected: LocaleIndian},
		{name: "hi-IN", locale: "hi-IN", expected: LocaleIndian},

		{name: "unknown", locale: "xx-YY", expected: LocaleEnGB},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetNumberLocale(tc.locale)
			if got != tc.expected {
				t.Errorf("GetNumberLocale(%q) = %+v, want %+v", tc.locale, got, tc.expected)
			}
		})
	}
}

func TestAddThousandSeparators(t *testing.T) {
	testCases := []struct {
		name               string
		input              string
		sep                string
		expected           string
		primaryGroupSize   int
		secondaryGroupSize int
	}{

		{
			name:               "standard 7 digits",
			input:              "1234567",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1,234,567",
		},
		{
			name:               "standard 6 digits",
			input:              "123456",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "123,456",
		},
		{
			name:               "standard 4 digits",
			input:              "1234",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1,234",
		},
		{
			name:               "standard 3 digits (no separator)",
			input:              "123",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "123",
		},
		{
			name:               "standard 10 digits",
			input:              "1234567890",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1,234,567,890",
		},

		{
			name:               "indian 7 digits",
			input:              "1234567",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 2,
			expected:           "12,34,567",
		},
		{
			name:               "indian 8 digits",
			input:              "12345678",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 2,
			expected:           "1,23,45,678",
		},
		{
			name:               "indian 9 digits",
			input:              "123456789",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 2,
			expected:           "12,34,56,789",
		},
		{
			name:               "indian 10 digits",
			input:              "1234567890",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 2,
			expected:           "1,23,45,67,890",
		},
		{
			name:               "indian 5 digits",
			input:              "12345",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 2,
			expected:           "12,345",
		},
		{
			name:               "indian 4 digits",
			input:              "1234",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 2,
			expected:           "1,234",
		},
		{
			name:               "indian 3 digits (no separator)",
			input:              "123",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 2,
			expected:           "123",
		},

		{
			name:               "period separator (German style)",
			input:              "1234567",
			sep:                ".",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1.234.567",
		},
		{
			name:               "space separator (French style)",
			input:              "1234567",
			sep:                " ",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1 234 567",
		},
		{
			name:               "apostrophe separator (Swiss style)",
			input:              "1234567",
			sep:                "'",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1'234'567",
		},
		{
			name:               "narrow no-break space (French CLDR)",
			input:              "1234567",
			sep:                "\u202f",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1\u202f234\u202f567",
		},

		{
			name:               "empty string",
			input:              "",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "",
		},
		{
			name:               "single digit",
			input:              "1",
			sep:                ",",
			primaryGroupSize:   3,
			secondaryGroupSize: 0,
			expected:           "1",
		},
		{
			name:               "zero group size",
			input:              "1234567",
			sep:                ",",
			primaryGroupSize:   0,
			secondaryGroupSize: 0,
			expected:           "1234567",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := addThousandSeparators(tc.input, tc.sep, tc.primaryGroupSize, tc.secondaryGroupSize)
			if got != tc.expected {
				t.Errorf("addThousandSeparators(%q, %q, %d, %d) = %q, want %q",
					tc.input, tc.sep, tc.primaryGroupSize, tc.secondaryGroupSize, got, tc.expected)
			}
		})
	}
}

func TestFormatNumberString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
		locale   NumberLocale
	}{

		{
			name:     "en-GB positive integer",
			input:    "1234567",
			locale:   LocaleEnGB,
			expected: "1,234,567",
		},
		{
			name:     "en-GB positive decimal",
			input:    "1234567.89",
			locale:   LocaleEnGB,
			expected: "1,234,567.89",
		},
		{
			name:     "en-GB negative",
			input:    "-1234567.89",
			locale:   LocaleEnGB,
			expected: "-1,234,567.89",
		},

		{
			name:     "de-DE positive integer",
			input:    "1234567",
			locale:   LocaleDeDe,
			expected: "1.234.567",
		},
		{
			name:     "de-DE positive decimal",
			input:    "1234567.89",
			locale:   LocaleDeDe,
			expected: "1.234.567,89",
		},
		{
			name:     "de-DE negative",
			input:    "-1234567.89",
			locale:   LocaleDeDe,
			expected: "-1.234.567,89",
		},

		{
			name:     "fr-FR positive integer",
			input:    "1234567",
			locale:   LocaleFrFr,
			expected: "1\u202f234\u202f567",
		},
		{
			name:     "fr-FR positive decimal",
			input:    "1234567.89",
			locale:   LocaleFrFr,
			expected: "1\u202f234\u202f567,89",
		},

		{
			name:     "de-CH positive integer",
			input:    "1234567",
			locale:   LocaleSwiss,
			expected: "1'234'567",
		},
		{
			name:     "de-CH positive decimal",
			input:    "1234567.89",
			locale:   LocaleSwiss,
			expected: "1'234'567.89",
		},

		{
			name:     "en-IN 7 digits",
			input:    "1234567",
			locale:   LocaleIndian,
			expected: "12,34,567",
		},
		{
			name:     "en-IN 8 digits",
			input:    "12345678",
			locale:   LocaleIndian,
			expected: "1,23,45,678",
		},
		{
			name:     "en-IN 10 digits",
			input:    "1234567890",
			locale:   LocaleIndian,
			expected: "1,23,45,67,890",
		},
		{
			name:     "en-IN with decimal",
			input:    "12345678.90",
			locale:   LocaleIndian,
			expected: "1,23,45,678.90",
		},
		{
			name:     "en-IN negative",
			input:    "-12345678.90",
			locale:   LocaleIndian,
			expected: "-1,23,45,678.90",
		},

		{
			name:     "raw no formatting",
			input:    "1234567.89",
			locale:   LocaleRaw,
			expected: "1234567.89",
		},

		{
			name:     "empty string",
			input:    "",
			locale:   LocaleEnGB,
			expected: "",
		},
		{
			name:     "small number no grouping",
			input:    "123.45",
			locale:   LocaleEnGB,
			expected: "123.45",
		},
		{
			name:     "high precision",
			input:    "1234567.123456789012345",
			locale:   LocaleEnGB,
			expected: "1,234,567.123456789012345",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatNumberString(tc.input, tc.locale)
			if got != tc.expected {
				t.Errorf("FormatNumberString(%q, %+v) = %q, want %q",
					tc.input, tc.locale, got, tc.expected)
			}
		})
	}
}

func TestDecimalFormat(t *testing.T) {
	testCases := []struct {
		name     string
		locale   string
		expected string
		decimal  Decimal
	}{

		{
			name:     "en-GB",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "en-GB",
			expected: "1,234,567.89",
		},
		{
			name:     "en-US",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "en-US",
			expected: "1,234,567.89",
		},

		{
			name:     "de-DE",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "de-DE",
			expected: "1.234.567,89",
		},
		{
			name:     "nl-NL (Dutch uses German style)",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "nl-NL",
			expected: "1.234.567,89",
		},
		{
			name:     "es-ES (Spanish uses German style)",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "es-ES",
			expected: "1.234.567,89",
		},
		{
			name:     "it-IT (Italian uses German style)",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "it-IT",
			expected: "1.234.567,89",
		},
		{
			name:     "pt-BR (Portuguese uses German style)",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "pt-BR",
			expected: "1.234.567,89",
		},

		{
			name:     "fr-FR",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "fr-FR",
			expected: "1\u202f234\u202f567,89",
		},
		{
			name:     "sv-SE (Swedish uses French style)",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "sv-SE",
			expected: "1\u202f234\u202f567,89",
		},
		{
			name:     "pl-PL (Polish uses French style)",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "pl-PL",
			expected: "1\u202f234\u202f567,89",
		},
		{
			name:     "ru-RU (Russian uses French style)",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "ru-RU",
			expected: "1\u202f234\u202f567,89",
		},

		{
			name:     "de-CH",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "de-CH",
			expected: "1'234'567.89",
		},
		{
			name:     "fr-CH",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "fr-CH",
			expected: "1'234'567.89",
		},
		{
			name:     "it-CH",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "it-CH",
			expected: "1'234'567.89",
		},

		{
			name:     "en-IN",
			decimal:  NewDecimalFromString("12345678.90"),
			locale:   "en-IN",
			expected: "1,23,45,678.9",
		},
		{
			name:     "hi-IN",
			decimal:  NewDecimalFromString("12345678.90"),
			locale:   "hi-IN",
			expected: "1,23,45,678.9",
		},
		{
			name:     "en-IN large number (1 crore)",
			decimal:  NewDecimalFromString("10000000"),
			locale:   "en-IN",
			expected: "1,00,00,000",
		},
		{
			name:     "en-IN 1 lakh",
			decimal:  NewDecimalFromString("100000"),
			locale:   "en-IN",
			expected: "1,00,000",
		},

		{
			name:     "ja-JP",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "ja-JP",
			expected: "1,234,567.89",
		},
		{
			name:     "zh-CN",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "zh-CN",
			expected: "1,234,567.89",
		},
		{
			name:     "ko-KR",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "ko-KR",
			expected: "1,234,567.89",
		},

		{
			name:     "unknown locale falls back to en-GB",
			decimal:  NewDecimalFromString("1234567.89"),
			locale:   "xx-YY",
			expected: "1,234,567.89",
		},

		{
			name:     "negative number",
			decimal:  NewDecimalFromString("-1234567.89"),
			locale:   "en-GB",
			expected: "-1,234,567.89",
		},
		{
			name:     "small number no grouping needed",
			decimal:  NewDecimalFromString("123.45"),
			locale:   "en-GB",
			expected: "123.45",
		},
		{
			name:     "integer only",
			decimal:  NewDecimalFromString("1234567"),
			locale:   "en-GB",
			expected: "1,234,567",
		},
		{
			name:     "high precision preserved",
			decimal:  NewDecimalFromString("1234567.1234567890123456789"),
			locale:   "en-GB",
			expected: "1,234,567.1234567890123456789",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.decimal.Format(tc.locale)
			if err != nil {
				t.Fatalf("Format(%q) returned unexpected error: %v", tc.locale, err)
			}
			if got != tc.expected {
				t.Errorf("Format(%q) = %q, want %q", tc.locale, got, tc.expected)
			}
		})
	}
}

func TestDecimalFormatError(t *testing.T) {
	dInvalid := NewDecimalFromString("invalid")
	_, err := dInvalid.Format("en-GB")
	if err == nil {
		t.Error("expected error from Format() for invalid decimal")
	}
}

func TestDecimalMustFormat(t *testing.T) {
	d := NewDecimalFromString("1234567.89")
	got := d.MustFormat("en-GB")
	expected := "1,234,567.89"
	if got != expected {
		t.Errorf("MustFormat(\"en-GB\") = %q, want %q", got, expected)
	}

	dInvalid := NewDecimalFromString("invalid")
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected MustFormat to panic on invalid decimal")
		}
	}()
	_ = dInvalid.MustFormat("en-GB")
}

func TestLocaleConsistency(t *testing.T) {

	datetimeLocales := []string{
		"en", "en-GB", "en-US",
		"de", "de-DE",
		"fr", "fr-FR",
		"es", "es-ES",
		"it", "it-IT",
		"pt", "pt-BR",
		"ja", "ja-JP",
		"zh", "zh-CN",
		"ko", "ko-KR",
		"ru", "ru-RU",
		"nl", "nl-NL",
		"pl", "pl-PL",
		"sv", "sv-SE",
	}

	for _, locale := range datetimeLocales {
		t.Run(locale, func(t *testing.T) {
			got := GetNumberLocale(locale)

			if got.GroupSize == 0 && got.ThousandSep == "" {
				t.Errorf("locale %q returned LocaleRaw, expected a proper locale", locale)
			}
		})
	}
}

func TestIndianNumberingExamples(t *testing.T) {

	testCases := []struct {
		name     string
		value    string
		expected string
	}{
		{name: "1 thousand", value: "1000", expected: "1,000"},
		{name: "10 thousand", value: "10000", expected: "10,000"},
		{name: "1 lakh", value: "100000", expected: "1,00,000"},
		{name: "10 lakh", value: "1000000", expected: "10,00,000"},
		{name: "1 crore", value: "10000000", expected: "1,00,00,000"},
		{name: "10 crore", value: "100000000", expected: "10,00,00,000"},
		{name: "1 arab (100 crore)", value: "1000000000", expected: "1,00,00,00,000"},
		{name: "with decimals", value: "12345678.50", expected: "1,23,45,678.5"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDecimalFromString(tc.value)
			got, err := d.Format("en-IN")
			if err != nil {
				t.Fatalf("Format failed: %v", err)
			}
			if got != tc.expected {
				t.Errorf("Format(\"en-IN\") = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestSwissNumberingExamples(t *testing.T) {

	testCases := []struct {
		name     string
		value    string
		locale   string
		expected string
	}{
		{name: "de-CH thousand", value: "1000", locale: "de-CH", expected: "1'000"},
		{name: "de-CH million", value: "1000000", locale: "de-CH", expected: "1'000'000"},
		{name: "de-CH with decimals", value: "1234.56", locale: "de-CH", expected: "1'234.56"},
		{name: "fr-CH thousand", value: "1000", locale: "fr-CH", expected: "1'000"},
		{name: "it-CH thousand", value: "1000", locale: "it-CH", expected: "1'000"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDecimalFromString(tc.value)
			got, err := d.Format(tc.locale)
			if err != nil {
				t.Fatalf("Format failed: %v", err)
			}
			if got != tc.expected {
				t.Errorf("Format(%q) = %q, want %q", tc.locale, got, tc.expected)
			}
		})
	}
}
