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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var testTime = time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)

func TestFormatDateTime_EnglishUK(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		style    DateTimeStyle
		dateOnly bool
		timeOnly bool
	}{
		{
			name:     "short date and time",
			style:    DateTimeStyleShort,
			expected: "15/01/2024 14:30",
		},
		{
			name:     "medium date and time",
			style:    DateTimeStyleMedium,
			expected: "15 Jan 2024 14:30:45",
		},
		{
			name:     "long date and time",
			style:    DateTimeStyleLong,
			expected: "15 January 2024 14:30:45 UTC",
		},
		{
			name:     "full date and time",
			style:    DateTimeStyleFull,
			expected: "Monday, 15 January 2024 14:30:45 UTC",
		},
		{
			name:     "short date only",
			style:    DateTimeStyleShort,
			dateOnly: true,
			expected: "15/01/2024",
		},
		{
			name:     "short time only",
			style:    DateTimeStyleShort,
			timeOnly: true,
			expected: "14:30",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDateTime(testTime, "en-GB", tc.style, tc.dateOnly, tc.timeOnly)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatDateTime_EnglishUS(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		style    DateTimeStyle
		dateOnly bool
	}{
		{
			name:     "short date",
			style:    DateTimeStyleShort,
			dateOnly: true,
			expected: "01/15/2024",
		},
		{
			name:     "medium date",
			style:    DateTimeStyleMedium,
			dateOnly: true,
			expected: "Jan 15, 2024",
		},
		{
			name:     "long date",
			style:    DateTimeStyleLong,
			dateOnly: true,
			expected: "January 15, 2024",
		},
		{
			name:     "full date",
			style:    DateTimeStyleFull,
			dateOnly: true,
			expected: "Monday, January 15, 2024",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDateTime(testTime, "en-US", tc.style, tc.dateOnly, false)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatDateTime_German(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		style    DateTimeStyle
		dateOnly bool
	}{
		{
			name:     "short date",
			style:    DateTimeStyleShort,
			dateOnly: true,
			expected: "15.01.2024",
		},
		{
			name:     "medium date",
			style:    DateTimeStyleMedium,
			dateOnly: true,
			expected: "15. Jan. 2024",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDateTime(testTime, "de-DE", tc.style, tc.dateOnly, false)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatDateTime_Japanese(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		style    DateTimeStyle
		dateOnly bool
	}{
		{
			name:     "short date",
			style:    DateTimeStyleShort,
			dateOnly: true,
			expected: "2024/01/15",
		},
		{
			name:     "medium date",
			style:    DateTimeStyleMedium,
			dateOnly: true,
			expected: "2024年1月15日",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatDateTime(testTime, "ja-JP", tc.style, tc.dateOnly, false)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatDateTime_LocaleFallback(t *testing.T) {

	result := FormatDateTime(testTime, "en-AU", DateTimeStyleShort, true, false)
	assert.Equal(t, "01/15/2024", result)

	result = FormatDateTime(testTime, "xx-XX", DateTimeStyleShort, true, false)
	assert.Equal(t, "2024-01-15", result)
}

func TestDateTime_FluentAPI(t *testing.T) {
	dt := NewDateTime(testTime)
	assert.Equal(t, DateTimeStyleMedium, dt.Style)

	dt = dt.Short()
	assert.Equal(t, DateTimeStyleShort, dt.Style)

	dt = dt.Long()
	assert.Equal(t, DateTimeStyleLong, dt.Style)

	dt = dt.Full()
	assert.Equal(t, DateTimeStyleFull, dt.Style)

	dt = dt.Medium()
	assert.Equal(t, DateTimeStyleMedium, dt.Style)

	dt = dt.DateOnly()
	assert.True(t, dt.OnlyDate)
	assert.False(t, dt.OnlyTime)

	dt = dt.TimeOnly()
	assert.False(t, dt.OnlyDate)
	assert.True(t, dt.OnlyTime)

	dt = dt.UTC()
	assert.True(t, dt.ConvertToUTC)
}

func TestDateTime_Format(t *testing.T) {
	dt := NewDateTime(testTime).Short().DateOnly()
	result := dt.Format("en-GB")
	assert.Equal(t, "15/01/2024", result)

	dt = NewDateTime(testTime).Short().DateOnly()
	result = dt.Format("en-US")
	assert.Equal(t, "01/15/2024", result)
}

func TestDateTime_String(t *testing.T) {
	dt := NewDateTime(testTime)
	result := dt.String()

	assert.Contains(t, result, "2024")
	assert.Contains(t, result, "15")
}

func TestNewDateTimeWithStyle(t *testing.T) {
	dt := newDateTimeWithStyle(testTime, DateTimeStyleLong)
	assert.Equal(t, DateTimeStyleLong, dt.Style)
	assert.Equal(t, testTime, dt.Time)
}

func TestDateTime_UTCConversion(t *testing.T) {

	loc, _ := time.LoadLocation("America/New_York")
	localTime := time.Date(2024, 1, 15, 9, 30, 45, 0, loc)

	dt := NewDateTime(localTime).Short().UTC().TimeOnly()
	result := dt.Format("en-GB")

	assert.Equal(t, "14:30", result)
}

func TestFormatDateTime_TimeFormats(t *testing.T) {

	afternoonTime := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	result := FormatDateTime(afternoonTime, "en-US", DateTimeStyleShort, false, true)
	assert.Equal(t, "2:30 PM", result)

	result = FormatDateTime(afternoonTime, "en-GB", DateTimeStyleShort, false, true)
	assert.Equal(t, "14:30", result)
}

func TestRender_DateTime(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"event": "Event on ${date}",
	})

	buffer := NewStrBuf(64)
	entry, _ := store.Get("en-GB", "event")

	vars := map[string]any{
		"date": testTime,
	}

	result := Render(entry, vars, nil, "en-GB", buffer)
	assert.Equal(t, "Event on 15 Jan 2024 14:30:45", result)
}

func TestRender_DateTimeWithStyle(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"event": "Event on ${date}",
	})

	buffer := NewStrBuf(64)
	entry, _ := store.Get("en-GB", "event")

	vars := map[string]any{
		"date": NewDateTime(testTime).Short().DateOnly(),
	}

	result := Render(entry, vars, nil, "en-GB", buffer)
	assert.Equal(t, "Event on 15/01/2024", result)
}

func TestRender_DateTimeUS(t *testing.T) {
	store := NewStore("en-US")
	store.AddTranslations("en-US", map[string]string{
		"event": "Event on ${date}",
	})

	buffer := NewStrBuf(64)
	entry, _ := store.Get("en-US", "event")

	vars := map[string]any{
		"date": NewDateTime(testTime).Short().DateOnly(),
	}

	result := Render(entry, vars, nil, "en-US", buffer)
	assert.Equal(t, "Event on 01/15/2024", result)
}

func TestTranslation_TimeVar(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"message": "Created at ${timestamp}",
	})
	pool := NewStrBufPool(64)

	entry, _ := store.Get("en-GB", "message")
	trans := NewTranslationWithLocale("message", entry, pool, "en-GB")
	trans.TimeVar("timestamp", testTime)
	result := trans.String()

	assert.Equal(t, "Created at 15 Jan 2024 14:30:45", result)
}

func TestTranslation_DateTimeVar(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"message": "Event: ${when}",
	})
	pool := NewStrBufPool(64)

	entry, _ := store.Get("en-GB", "message")
	trans := NewTranslationWithLocale("message", entry, pool, "en-GB")
	trans.DateTimeVar("when", NewDateTime(testTime).Long().DateOnly())
	result := trans.String()

	assert.Equal(t, "Event: 15 January 2024", result)
}

func TestTranslation_DateTimeVarUS(t *testing.T) {
	store := NewStore("en-US")
	store.AddTranslations("en-US", map[string]string{
		"message": "Event: ${when}",
	})
	pool := NewStrBufPool(64)

	entry, _ := store.Get("en-US", "message")
	trans := NewTranslationWithLocale("message", entry, pool, "en-US")
	trans.DateTimeVar("when", NewDateTime(testTime).Long().DateOnly())
	result := trans.String()

	assert.Equal(t, "Event: January 15, 2024", result)
}

func TestTranslation_DateTimeVarFull(t *testing.T) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"message": "Meeting: ${when}",
	})
	pool := NewStrBufPool(64)

	entry, _ := store.Get("en-GB", "message")
	trans := NewTranslationWithLocale("message", entry, pool, "en-GB")
	trans.DateTimeVar("when", NewDateTime(testTime).Full().DateOnly())
	result := trans.String()

	assert.Equal(t, "Meeting: Monday, 15 January 2024", result)
}

func BenchmarkFormatDateTime_Short(b *testing.B) {
	for b.Loop() {
		_ = FormatDateTime(testTime, "en-GB", DateTimeStyleShort, false, false)
	}
}

func BenchmarkFormatDateTime_Medium(b *testing.B) {
	for b.Loop() {
		_ = FormatDateTime(testTime, "en-GB", DateTimeStyleMedium, false, false)
	}
}

func BenchmarkFormatDateTime_Long(b *testing.B) {
	for b.Loop() {
		_ = FormatDateTime(testTime, "en-GB", DateTimeStyleLong, false, false)
	}
}

func BenchmarkDateTime_FluentAPI(b *testing.B) {
	for b.Loop() {
		dt := NewDateTime(testTime).Short().DateOnly()
		_ = dt.Format("en-GB")
	}
}

func BenchmarkRender_WithDateTime(b *testing.B) {
	store := NewStore("en-GB")
	store.AddTranslations("en-GB", map[string]string{
		"event": "Event on ${date}",
	})
	buffer := NewStrBuf(64)
	entry, _ := store.Get("en-GB", "event")
	vars := map[string]any{
		"date": NewDateTime(testTime).Short().DateOnly(),
	}
	b.ResetTimer()

	for b.Loop() {
		_ = Render(entry, vars, nil, "en-GB", buffer)
	}
}
