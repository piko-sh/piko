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

import (
	"strings"
	"time"
)

// DateTimeStyle defines the format used when showing date and time values.
type DateTimeStyle uint8

const (
	// DateTimeStyleShort is the compact format (e.g. "15/01/2024", "3:04 PM").
	DateTimeStyleShort DateTimeStyle = iota

	// DateTimeStyleMedium is a moderate format (e.g., "15 Jan 2024", "15:04:05").
	DateTimeStyleMedium

	// DateTimeStyleLong is a detailed format (e.g., "15 January 2024", "15:04:05
	// GMT").
	DateTimeStyleLong

	// DateTimeStyleFull is the most detailed format (e.g., "Monday, 15 January
	// 2024").
	DateTimeStyleFull
)

// DateTime wraps a time.Time value with formatting options for localised
// rendering. It implements fmt.Stringer.
type DateTime struct {
	// Time is the underlying timestamp value.
	Time time.Time

	// Style sets the formatting style (short, medium, long, or full).
	Style DateTimeStyle

	// OnlyDate indicates whether to show only the date part when formatting.
	OnlyDate bool

	// OnlyTime indicates whether to format only the time part, hiding the date.
	OnlyTime bool

	// ConvertToUTC indicates whether to convert the time to UTC before formatting.
	ConvertToUTC bool
}

// NewDateTime creates a DateTime with default medium style.
//
// Takes t (time.Time) which specifies the time value to wrap.
//
// Returns DateTime which is configured with medium style and all display
// options disabled.
func NewDateTime(t time.Time) DateTime {
	return DateTime{
		Time:         t,
		Style:        DateTimeStyleMedium,
		OnlyDate:     false,
		OnlyTime:     false,
		ConvertToUTC: false,
	}
}

// DateOnly returns a copy configured to format only the date.
//
// Returns DateTime which has date formatting enabled and time formatting
// disabled.
func (dt DateTime) DateOnly() DateTime {
	dt.OnlyDate = true
	dt.OnlyTime = false
	return dt
}

// TimeOnly returns a copy configured to format only the time.
//
// Returns DateTime which is a copy with time-only formatting enabled.
func (dt DateTime) TimeOnly() DateTime {
	dt.OnlyTime = true
	dt.OnlyDate = false
	return dt
}

// UTC returns a copy configured to use UTC timezone.
//
// Returns DateTime which is the modified copy with UTC conversion enabled.
func (dt DateTime) UTC() DateTime {
	dt.ConvertToUTC = true
	return dt
}

// Short returns a copy with the short date-time style applied.
//
// Returns DateTime which is the modified copy.
func (dt DateTime) Short() DateTime {
	dt.Style = DateTimeStyleShort
	return dt
}

// Medium returns a copy with medium style.
//
// Returns DateTime which is a copy of the receiver with medium style applied.
func (dt DateTime) Medium() DateTime {
	dt.Style = DateTimeStyleMedium
	return dt
}

// Long returns a copy of the receiver with the long style set.
//
// Returns DateTime which is a copy with the long date and time style.
func (dt DateTime) Long() DateTime {
	dt.Style = DateTimeStyleLong
	return dt
}

// Full returns a copy with full style.
//
// Returns DateTime which is a copy of the receiver with full style applied.
func (dt DateTime) Full() DateTime {
	dt.Style = DateTimeStyleFull
	return dt
}

// localePatterns holds date and time format patterns for a locale.
// Patterns use Go's time.Format reference time (Mon Jan 2 15:04:05 MST 2006).
type localePatterns struct {
	// DateShort is the Go time format pattern for short date style.
	DateShort string

	// DateMedium is the Go time format pattern for medium-style dates.
	DateMedium string

	// DateLong is the Go time format pattern for long date style.
	DateLong string

	// DateFull is the Go time layout for full date format in this locale.
	DateFull string

	// TimeShort is the Go time format pattern for short time display.
	TimeShort string

	// TimeMedium is the Go time format pattern for medium-length time display.
	TimeMedium string

	// TimeLong is the Go time format pattern for the long time style.
	TimeLong string

	// TimeFull is the Go time format pattern for full time display.
	TimeFull string
}

var (
	// defaultPatterns is the fallback for unknown locales (ISO-like).
	defaultPatterns = localePatterns{
		DateShort:  "2006-01-02",
		DateMedium: "2 Jan 2006",
		DateLong:   "2 January 2006",
		DateFull:   "Monday, 2 January 2006",
		TimeShort:  "15:04",
		TimeMedium: "15:04:05",
		TimeLong:   "15:04:05 MST",
		TimeFull:   "15:04:05 MST",
	}

	// localeFormats maps locale codes to their date/time patterns. Locales are
	// matched by prefix, so "en-GB" matches before falling back to "en".
	localeFormats = map[string]localePatterns{
		"en-GB": {
			DateShort:  "02/01/2006",
			DateMedium: "2 Jan 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday, 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"en-US": {
			DateShort:  "01/02/2006",
			DateMedium: "Jan 2, 2006",
			DateLong:   "January 2, 2006",
			DateFull:   "Monday, January 2, 2006",
			TimeShort:  "3:04 PM",
			TimeMedium: "3:04:05 PM",
			TimeLong:   "3:04:05 PM MST",
			TimeFull:   "3:04:05 PM MST",
		},
		"en": {
			DateShort:  "01/02/2006",
			DateMedium: "Jan 2, 2006",
			DateLong:   "January 2, 2006",
			DateFull:   "Monday, January 2, 2006",
			TimeShort:  "3:04 PM",
			TimeMedium: "3:04:05 PM",
			TimeLong:   "3:04:05 PM MST",
			TimeFull:   "3:04:05 PM MST",
		},
		"de-DE": {
			DateShort:  "02.01.2006",
			DateMedium: "2. Jan. 2006",
			DateLong:   "2. January 2006",
			DateFull:   "Monday, 2. January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"de": {
			DateShort:  "02.01.2006",
			DateMedium: "2. Jan. 2006",
			DateLong:   "2. January 2006",
			DateFull:   "Monday, 2. January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"fr-FR": {
			DateShort:  "02/01/2006",
			DateMedium: "2 janv. 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"fr": {
			DateShort:  "02/01/2006",
			DateMedium: "2 janv. 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"es-ES": {
			DateShort:  "02/01/2006",
			DateMedium: "2 ene 2006",
			DateLong:   "2 de January de 2006",
			DateFull:   "Monday, 2 de January de 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"es": {
			DateShort:  "02/01/2006",
			DateMedium: "2 ene 2006",
			DateLong:   "2 de January de 2006",
			DateFull:   "Monday, 2 de January de 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"it-IT": {
			DateShort:  "02/01/2006",
			DateMedium: "2 gen 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"it": {
			DateShort:  "02/01/2006",
			DateMedium: "2 gen 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"pt-BR": {
			DateShort:  "02/01/2006",
			DateMedium: "2 de jan. de 2006",
			DateLong:   "2 de January de 2006",
			DateFull:   "Monday, 2 de January de 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"pt": {
			DateShort:  "02/01/2006",
			DateMedium: "2 de jan. de 2006",
			DateLong:   "2 de January de 2006",
			DateFull:   "Monday, 2 de January de 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"ja-JP": {
			DateShort:  "2006/01/02",
			DateMedium: "2006年1月2日",
			DateLong:   "2006年1月2日",
			DateFull:   "2006年1月2日 Monday",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"ja": {
			DateShort:  "2006/01/02",
			DateMedium: "2006年1月2日",
			DateLong:   "2006年1月2日",
			DateFull:   "2006年1月2日 Monday",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"zh-CN": {
			DateShort:  "2006/01/02",
			DateMedium: "2006年1月2日",
			DateLong:   "2006年1月2日",
			DateFull:   "2006年1月2日 Monday",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"zh": {
			DateShort:  "2006/01/02",
			DateMedium: "2006年1月2日",
			DateLong:   "2006年1月2日",
			DateFull:   "2006年1月2日 Monday",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"ko-KR": {
			DateShort:  "2006. 01. 02.",
			DateMedium: "2006년 1월 2일",
			DateLong:   "2006년 1월 2일",
			DateFull:   "2006년 1월 2일 Monday",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"ko": {
			DateShort:  "2006. 01. 02.",
			DateMedium: "2006년 1월 2일",
			DateLong:   "2006년 1월 2일",
			DateFull:   "2006년 1월 2일 Monday",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"ru-RU": {
			DateShort:  "02.01.2006",
			DateMedium: "2 янв. 2006 г.",
			DateLong:   "2 January 2006 г.",
			DateFull:   "Monday, 2 January 2006 г.",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"ru": {
			DateShort:  "02.01.2006",
			DateMedium: "2 янв. 2006 г.",
			DateLong:   "2 January 2006 г.",
			DateFull:   "Monday, 2 January 2006 г.",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"nl-NL": {
			DateShort:  "02-01-2006",
			DateMedium: "2 jan. 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"nl": {
			DateShort:  "02-01-2006",
			DateMedium: "2 jan. 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"pl-PL": {
			DateShort:  "02.01.2006",
			DateMedium: "2 sty 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday, 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"pl": {
			DateShort:  "02.01.2006",
			DateMedium: "2 sty 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday, 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"sv-SE": {
			DateShort:  "2006-01-02",
			DateMedium: "2 jan. 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
		"sv": {
			DateShort:  "2006-01-02",
			DateMedium: "2 jan. 2006",
			DateLong:   "2 January 2006",
			DateFull:   "Monday 2 January 2006",
			TimeShort:  "15:04",
			TimeMedium: "15:04:05",
			TimeLong:   "15:04:05 MST",
			TimeFull:   "15:04:05 MST",
		},
	}
)

// Format formats the DateTime according to its configuration and the given
// locale.
//
// Takes locale (string) which specifies the locale for formatting.
//
// Returns string which is the formatted date and time representation.
func (dt DateTime) Format(locale string) string {
	t := dt.Time
	if dt.ConvertToUTC {
		t = t.UTC()
	}
	return FormatDateTime(t, locale, dt.Style, dt.OnlyDate, dt.OnlyTime)
}

// String implements fmt.Stringer using medium style and empty locale defaults.
//
// Returns string which is the formatted date and time.
func (dt DateTime) String() string {
	return dt.Format("")
}

// FormatDateTime formats a time.Time value according to the locale and style.
//
// Takes t (time.Time) which is the time value to format.
// Takes locale (string) which specifies the locale for formatting patterns.
// Takes style (DateTimeStyle) which controls the output format length.
// Takes dateOnly (bool) which when true returns only the date portion.
// Takes timeOnly (bool) which when true returns only the time portion.
//
// Returns string which is the formatted date, time, or combined datetime.
func FormatDateTime(t time.Time, locale string, style DateTimeStyle, dateOnly, timeOnly bool) string {
	patterns := getPatternsForLocale(locale)

	var datePart, timePart string

	if !timeOnly {
		switch style {
		case DateTimeStyleShort:
			datePart = t.Format(patterns.DateShort)
		case DateTimeStyleMedium:
			datePart = t.Format(patterns.DateMedium)
		case DateTimeStyleLong:
			datePart = t.Format(patterns.DateLong)
		case DateTimeStyleFull:
			datePart = t.Format(patterns.DateFull)
		}
	}

	if !dateOnly {
		switch style {
		case DateTimeStyleShort:
			timePart = t.Format(patterns.TimeShort)
		case DateTimeStyleMedium:
			timePart = t.Format(patterns.TimeMedium)
		case DateTimeStyleLong:
			timePart = t.Format(patterns.TimeLong)
		case DateTimeStyleFull:
			timePart = t.Format(patterns.TimeFull)
		}
	}

	if dateOnly {
		return datePart
	}
	if timeOnly {
		return timePart
	}
	if datePart != "" && timePart != "" {
		return datePart + " " + timePart
	}
	if datePart != "" {
		return datePart
	}
	return timePart
}

// newDateTimeWithStyle creates a DateTime with the specified style.
//
// Takes t (time.Time) which specifies the time value to wrap.
// Takes style (DateTimeStyle) which determines how the time is formatted.
//
// Returns DateTime which contains the time with the given style and default
// display options.
func newDateTimeWithStyle(t time.Time, style DateTimeStyle) DateTime {
	return DateTime{
		Time:         t,
		Style:        style,
		OnlyDate:     false,
		OnlyTime:     false,
		ConvertToUTC: false,
	}
}

// getPatternsForLocale returns the format patterns for a locale.
// It tries an exact match first, then falls back to the base language.
//
// Takes locale (string) which specifies the locale code to look up.
//
// Returns localePatterns which contains the format patterns for the locale.
func getPatternsForLocale(locale string) localePatterns {
	if patterns, ok := localeFormats[locale]; ok {
		return patterns
	}

	if baseLang, _, found := strings.Cut(locale, "-"); found && baseLang != "" {
		if patterns, ok := localeFormats[baseLang]; ok {
			return patterns
		}
	}

	return defaultPatterns
}
