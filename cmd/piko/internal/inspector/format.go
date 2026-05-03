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

package inspector

import (
	"fmt"
	"strings"
	"time"
)

// hyphenGlyph is the placeholder used by the CLI Printer for
// missing values.
const hyphenGlyph = "-"

// hoursPerDay is the duration unit used when formatting durations of
// one day or more.
const hoursPerDay = 24

// secondsPerMinute is the multiplier used when expanding minute and
// hour components in the millisecond duration formatter.
const secondsPerMinute = 60

// FormatTimeSince returns a short human-readable string for the elapsed
// time between past and now.
//
// The output uses the largest unit that produces a non-zero value with
// a single decimal place where helpful: "5s ago", "12m ago", "2h ago",
// "3d ago".
//
// Takes now (time.Time) which is the reference time.
// Takes past (time.Time) which is the earlier instant.
//
// Returns string which is the relative-time label. When past is zero
// or in the future, hyphen is returned.
func FormatTimeSince(now, past time.Time) string {
	if past.IsZero() || past.After(now) {
		return hyphenGlyph
	}
	return FormatDuration(now.Sub(past)) + " ago"
}

// FormatDuration renders d in the largest sensible unit.
//
// Values below one second are represented in milliseconds. Values are
// non-negative; negative durations are treated as zero.
//
// Takes d (time.Duration) which is the duration to render.
//
// Returns string which is the formatted label.
func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Second:
		ms := d.Milliseconds()
		if ms <= 0 {
			return "0ms"
		}
		return fmt.Sprintf("%dms", ms)
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < hoursPerDay*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/hoursPerDay))
	}
}

// FormatBytes returns a human-readable byte count using IEC binary
// units (1024-based). Common UI choice for memory and file sizes.
//
// Takes bytes (uint64) which is the count to format.
//
// Returns string in the form "12.3 MiB" / "456 B".
func FormatBytes(bytes uint64) string {
	const (
		kib = 1024
		mib = 1024 * 1024
		gib = 1024 * 1024 * 1024
		tib = 1024 * 1024 * 1024 * 1024
	)
	switch {
	case bytes >= tib:
		return fmt.Sprintf("%.1f TiB", float64(bytes)/float64(tib))
	case bytes >= gib:
		return fmt.Sprintf("%.1f GiB", float64(bytes)/float64(gib))
	case bytes >= mib:
		return fmt.Sprintf("%.1f MiB", float64(bytes)/float64(mib))
	case bytes >= kib:
		return fmt.Sprintf("%.1f KiB", float64(bytes)/float64(kib))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatDetailTime renders a timestamp for a detail-pane row in the
// shared "YYYY-MM-DD HH:MM:SS" layout. Zero values render as a
// hyphen, so the row reads "Started -" instead of an old epoch.
//
// Takes t (time.Time) which is the timestamp to render.
//
// Returns string which is the formatted timestamp or hyphen.
func FormatDetailTime(t time.Time) string {
	if t.IsZero() {
		return hyphenGlyph
	}
	return t.Format("2006-01-02 15:04:05")
}

// FormatDurationNanos renders a proto-encoded nanosecond duration as a
// human-readable string. Returns "disabled" for non-positive values
// so callers that use 0 as "off" render a clean label.
//
// Takes nanos (int64) which is the proto-encoded nanosecond duration.
//
// Returns string which is "disabled" when nanos<=0, otherwise the
// formatted duration.
func FormatDurationNanos(nanos int64) string {
	if nanos <= 0 {
		return "disabled"
	}
	return time.Duration(nanos).String()
}

// FormatOptionalTime renders a unix-millisecond timestamp as RFC 3339
// time, or "never" when the value is zero.
//
// Takes ms (int64) which is the unix-millisecond timestamp.
//
// Returns string which is "never" for the zero timestamp, otherwise
// the formatted time.
func FormatOptionalTime(ms int64) string {
	if ms <= 0 {
		return "never"
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

// FormatUnixSeconds renders an integer second-precision unix timestamp
// using the same "YYYY-MM-DD HH:MM:SS" layout as the legacy CLI
// describe output. Zero values render as hyphenGlyph so empty rows
// match the historical "-" placeholder.
//
// Takes ts (int64) which is the unix timestamp in seconds.
//
// Returns string which is the formatted local-time stamp, or
// hyphenGlyph when ts is zero.
func FormatUnixSeconds(ts int64) string {
	if ts == 0 {
		return hyphenGlyph
	}
	return time.Unix(ts, 0).Format("2006-01-02 15:04:05")
}

// FormatMilliseconds renders a millisecond-precision duration in the
// historical CLI describe layout: "500ms" below one second, "1.5s"
// below one minute, "5m30s" below one hour, and "2h15m" otherwise.
//
// Takes ms (int64) which is the duration in milliseconds.
//
// Returns string which is the formatted duration label.
func FormatMilliseconds(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	totalSeconds := int64(d.Seconds())
	if d < time.Hour {
		m := totalSeconds / secondsPerMinute
		s := totalSeconds % secondsPerMinute
		return fmt.Sprintf("%dm%ds", m, s)
	}
	totalMinutes := totalSeconds / secondsPerMinute
	h := totalMinutes / secondsPerMinute
	m := totalMinutes % secondsPerMinute
	return fmt.Sprintf("%dh%dm", h, m)
}

// matchesFilter reports whether name matches filter using the CLI
// describe semantics: an empty filter matches everything, and a
// non-empty filter matches when name equals or has the filter as a
// case-insensitive prefix.
//
// Takes name (string) which is the value to test.
// Takes filter (string) which is the filter to compare against; an
// empty filter matches every name.
//
// Returns bool which reports whether name matches.
func matchesFilter(name, filter string) bool {
	if filter == "" {
		return true
	}
	lower := strings.ToLower(name)
	filterLower := strings.ToLower(filter)
	return lower == filterLower || strings.HasPrefix(lower, filterLower)
}
