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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"piko.sh/piko/wdk/maths"
	"piko.sh/piko/wdk/safeconv"
)

// defaultPrecision signals that no explicit precision has been set.
const defaultPrecision = -1

// formatBuilderPool reuses FormatBuilder instances to reduce allocations.
var formatBuilderPool = sync.Pool{
	New: func() any { return &FormatBuilder{} },
}

// FormatBuilder provides a fluent API for formatting numeric and temporal
// values. It implements fmt.Stringer so templates auto-stringify the result.
//
// Use F() for locale-free formatting and NewLF() (via r.LF()) for
// locale-aware formatting with thousand separators, currency symbols, and
// localised date patterns.
type FormatBuilder struct {
	// value is the data to format; may be any supported type.
	value any

	// locale is the locale code used for localised formatting.
	locale string

	// precision is the number of decimal places for numeric values.
	precision int

	// style is the date-time formatting style.
	style DateTimeStyle

	// dateOnly restricts temporal output to the date portion only.
	dateOnly bool

	// timeOnly restricts temporal output to the time portion only.
	timeOnly bool

	// utc converts temporal values to UTC before formatting.
	utc bool

	// hasLocale indicates whether a locale has been explicitly set.
	hasLocale bool
}

// NewLF creates a FormatBuilder with a locale pre-applied. This is called by
// RequestData.LF() which passes the current request locale.
//
// Takes value (any) which is the value to format.
// Takes locale (string) which is the locale code for formatting.
//
// Returns *FormatBuilder which is ready for optional method chaining.
func NewLF(value any, locale string) *FormatBuilder {
	fb, ok := formatBuilderPool.Get().(*FormatBuilder)
	if !ok {
		fb = &FormatBuilder{}
	}
	fb.reset()
	fb.value = value
	fb.locale = locale
	fb.hasLocale = true
	return fb
}

// Precision sets the number of decimal places for numeric values.
// For non-numeric types this is a no-op.
//
// Takes n (int) which is the number of decimal places.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) Precision(n int) *FormatBuilder {
	fb.precision = n
	return fb
}

// Short sets the short style for temporal values.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) Short() *FormatBuilder {
	fb.style = DateTimeStyleShort
	return fb
}

// Medium sets the medium style for temporal values.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) Medium() *FormatBuilder {
	fb.style = DateTimeStyleMedium
	return fb
}

// Long sets the long style for temporal values.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) Long() *FormatBuilder {
	fb.style = DateTimeStyleLong
	return fb
}

// Full sets the full style for temporal values.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) Full() *FormatBuilder {
	fb.style = DateTimeStyleFull
	return fb
}

// DateOnly configures temporal formatting to show only the date portion.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) DateOnly() *FormatBuilder {
	fb.dateOnly = true
	fb.timeOnly = false
	return fb
}

// TimeOnly configures temporal formatting to show only the time portion.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) TimeOnly() *FormatBuilder {
	fb.timeOnly = true
	fb.dateOnly = false
	return fb
}

// UTC configures temporal formatting to convert to UTC before formatting.
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) UTC() *FormatBuilder {
	fb.utc = true
	return fb
}

// Locale overrides the locale for formatting. This works on both F() and LF()
// builders.
//
// Takes code (string) which is the locale code (e.g. "de-DE", "en-GB").
//
// Returns *FormatBuilder for method chaining.
func (fb *FormatBuilder) Locale(code string) *FormatBuilder {
	fb.locale = code
	fb.hasLocale = true
	return fb
}

// String renders the formatted value and returns the FormatBuilder to the pool.
// Implements fmt.Stringer for automatic conversion in templates.
//
// Returns string which is the formatted representation of the value.
func (fb *FormatBuilder) String() string {
	result := fb.format()
	fb.release()
	return result
}

// release returns the FormatBuilder to the pool for reuse.
//
// Caller must not use the FormatBuilder after calling release.
func (fb *FormatBuilder) release() {
	formatBuilderPool.Put(fb)
}

// reset clears all fields to prepare for reuse.
func (fb *FormatBuilder) reset() {
	fb.value = nil
	fb.locale = ""
	fb.precision = defaultPrecision
	fb.style = DateTimeStyleMedium
	fb.dateOnly = false
	fb.timeOnly = false
	fb.utc = false
	fb.hasLocale = false
}

// format dispatches formatting based on the value's type.
//
// Returns string which is the formatted representation of the stored value.
//
//nolint:revive // type dispatch
func (fb *FormatBuilder) format() string {
	switch v := fb.value.(type) {
	case nil:
		return ""
	case string:
		return v
	case maths.Decimal:
		return fb.formatDecimal(v)
	case *maths.Decimal:
		if v == nil {
			return ""
		}
		return fb.formatDecimal(*v)
	case maths.Money:
		return fb.formatMoney(v)
	case *maths.Money:
		if v == nil {
			return ""
		}
		return fb.formatMoney(*v)
	case maths.BigInt:
		return fb.formatBigInt(v)
	case *maths.BigInt:
		if v == nil {
			return ""
		}
		return fb.formatBigInt(*v)
	case time.Time:
		return fb.formatTime(v)
	case *time.Time:
		if v == nil {
			return ""
		}
		return fb.formatTime(*v)
	case DateTime:
		return fb.formatDateTime(v)
	case *DateTime:
		if v == nil {
			return ""
		}
		return fb.formatDateTime(*v)
	case float64:
		return fb.formatFloat64(v)
	case float32:
		return fb.formatFloat64(float64(v))
	case int:
		return fb.formatInt64(int64(v))
	case int64:
		return fb.formatInt64(v)
	case int32:
		return fb.formatInt64(int64(v))
	case int16:
		return fb.formatInt64(int64(v))
	case int8:
		return fb.formatInt64(int64(v))
	case uint:
		return fb.formatUint64(uint64(v))
	case uint64:
		return fb.formatUint64(v)
	case uint32:
		return fb.formatUint64(uint64(v))
	case uint16:
		return fb.formatUint64(uint64(v))
	case uint8:
		return fb.formatUint64(uint64(v))
	case time.Duration:
		return v.String()
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatDecimal formats a maths.Decimal value.
//
// Takes d (maths.Decimal) which is the decimal value to format.
//
// Returns string which is the formatted decimal, or empty on error.
func (fb *FormatBuilder) formatDecimal(d maths.Decimal) string {
	if fb.precision >= 0 {
		d = d.Round(safeconv.IntToInt32(fb.precision))
	}
	if fb.hasLocale && fb.locale != "" {
		s, err := d.Format(fb.locale)
		if err != nil {
			return ""
		}
		return s
	}
	s, err := d.String()
	if err != nil {
		return ""
	}
	return s
}

// formatMoney formats a maths.Money value.
//
// Takes m (maths.Money) which is the monetary value to format.
//
// Returns string which is the formatted money value, or empty on error.
func (fb *FormatBuilder) formatMoney(m maths.Money) string {
	if fb.hasLocale && fb.locale != "" {
		s, err := m.Format(moneyLocale(fb.locale))
		if err != nil {
			return ""
		}
		return s
	}
	s, err := m.FormattedNumber()
	if err != nil {
		return ""
	}
	return s
}

// formatBigInt formats a maths.BigInt value.
//
// Takes b (maths.BigInt) which is the big integer to format.
//
// Returns string which is the formatted integer, or empty on error.
func (fb *FormatBuilder) formatBigInt(b maths.BigInt) string {
	if fb.hasLocale && fb.locale != "" {
		s, err := b.ToDecimal().Format(fb.locale)
		if err != nil {
			return ""
		}
		return s
	}
	s, err := b.String()
	if err != nil {
		return ""
	}
	return s
}

// formatFloat64 formats a float64 value.
//
// Takes f (float64) which is the floating-point number to format.
//
// Returns string which is the formatted number.
func (fb *FormatBuilder) formatFloat64(f float64) string {
	var s string
	if fb.precision >= 0 {
		s = strconv.FormatFloat(f, 'f', fb.precision, 64)
	} else {
		s = strconv.FormatFloat(f, 'f', -1, 64)
	}
	if fb.hasLocale && fb.locale != "" {
		return maths.FormatNumberString(s, maths.GetNumberLocale(fb.locale))
	}
	return s
}

// formatInt64 formats a signed integer value.
//
// Takes i (int64) which is the integer to format.
//
// Returns string which is the formatted integer.
func (fb *FormatBuilder) formatInt64(i int64) string {
	s := strconv.FormatInt(i, 10)
	if fb.hasLocale && fb.locale != "" {
		return maths.FormatNumberString(s, maths.GetNumberLocale(fb.locale))
	}
	return s
}

// formatUint64 formats an unsigned integer value.
//
// Takes u (uint64) which is the unsigned integer to format.
//
// Returns string which is the formatted unsigned integer.
func (fb *FormatBuilder) formatUint64(u uint64) string {
	s := strconv.FormatUint(u, 10)
	if fb.hasLocale && fb.locale != "" {
		return maths.FormatNumberString(s, maths.GetNumberLocale(fb.locale))
	}
	return s
}

// formatTime formats a time.Time value.
//
// Takes t (time.Time) which is the time to format.
//
// Returns string which is the formatted time.
func (fb *FormatBuilder) formatTime(t time.Time) string {
	if fb.utc {
		t = t.UTC()
	}
	if fb.hasLocale && fb.locale != "" {
		return FormatDateTime(t, fb.locale, fb.style, fb.dateOnly, fb.timeOnly)
	}
	return FormatDateTime(t, "", fb.style, fb.dateOnly, fb.timeOnly)
}

// formatDateTime formats a DateTime value.
//
// Takes dt (DateTime) which is the date-time to format.
//
// Returns string which is the formatted date-time.
func (fb *FormatBuilder) formatDateTime(dt DateTime) string {
	if fb.utc {
		dt.ConvertToUTC = true
	}
	dt.Style = fb.style
	dt.OnlyDate = fb.dateOnly
	dt.OnlyTime = fb.timeOnly

	if fb.hasLocale && fb.locale != "" {
		return dt.Format(fb.locale)
	}
	return dt.Format("")
}

var _ fmt.Stringer = (*FormatBuilder)(nil)

// F creates a FormatBuilder with no locale applied. The value is formatted
// using its default string representation, with optional precision control
// via method chaining.
//
// Takes value (any) which is the value to format.
//
// Returns *FormatBuilder which is ready for optional method chaining and
// implements fmt.Stringer.
func F(value any) *FormatBuilder {
	fb, ok := formatBuilderPool.Get().(*FormatBuilder)
	if !ok {
		fb = &FormatBuilder{}
	}
	fb.reset()
	fb.value = value
	return fb
}

// moneyLocale converts a dash-separated locale code (such as "en-GB")
// to the underscore format expected by the currency library ("en_GB").
//
// Takes locale (string) which is the dash-separated locale code.
//
// Returns string which is the underscore-separated locale code.
func moneyLocale(locale string) string {
	return strings.ReplaceAll(locale, "-", "_")
}
