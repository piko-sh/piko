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
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		want string
		in   time.Duration
	}{
		{in: 0, want: "0ms"},
		{in: 500 * time.Millisecond, want: "500ms"},
		{in: 2 * time.Second, want: "2s"},
		{in: 90 * time.Second, want: "1m"},
		{in: 2 * time.Hour, want: "2h"},
		{in: 3 * 24 * time.Hour, want: "3d"},
		{in: -time.Minute, want: "0ms"},
	}
	for _, c := range cases {
		if got := FormatDuration(c.in); got != c.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatTimeSince(t *testing.T) {
	now := time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		past time.Time
		want string
	}{
		{now.Add(-5 * time.Second), "5s ago"},
		{now.Add(-2 * time.Hour), "2h ago"},
		{now.Add(time.Minute), hyphenGlyph},
		{time.Time{}, hyphenGlyph},
	}
	for _, c := range cases {
		if got := FormatTimeSince(now, c.past); got != c.want {
			t.Errorf("FormatTimeSince(now, %v) = %q, want %q", c.past, got, c.want)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		want string
		in   uint64
	}{
		{in: 0, want: "0 B"},
		{in: 500, want: "500 B"},
		{in: 1024, want: "1.0 KiB"},
		{in: 2 * 1024 * 1024, want: "2.0 MiB"},
		{in: 3 * 1024 * 1024 * 1024, want: "3.0 GiB"},
		{in: 4 * 1024 * 1024 * 1024 * 1024, want: "4.0 TiB"},
	}
	for _, c := range cases {
		if got := FormatBytes(c.in); got != c.want {
			t.Errorf("FormatBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDetailTime(t *testing.T) {
	cases := []struct {
		in   time.Time
		want string
	}{
		{time.Time{}, hyphenGlyph},
		{time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC), "2026-04-25 12:00:00"},
	}
	for _, c := range cases {
		if got := FormatDetailTime(c.in); got != c.want {
			t.Errorf("FormatDetailTime(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatDurationNanos(t *testing.T) {
	cases := []struct {
		want string
		in   int64
	}{
		{in: 0, want: "disabled"},
		{in: -1, want: "disabled"},
		{in: int64(time.Second), want: "1s"},
		{in: int64(2 * time.Minute), want: "2m0s"},
	}
	for _, c := range cases {
		if got := FormatDurationNanos(c.in); got != c.want {
			t.Errorf("FormatDurationNanos(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatOptionalTime(t *testing.T) {
	cases := []struct {
		want string
		in   int64
	}{
		{in: 0, want: "never"},
		{in: -1, want: "never"},
	}
	for _, c := range cases {
		if got := FormatOptionalTime(c.in); got != c.want {
			t.Errorf("FormatOptionalTime(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatUnixSeconds(t *testing.T) {
	if got := FormatUnixSeconds(0); got != hyphenGlyph {
		t.Errorf("FormatUnixSeconds(0) = %q, want %q", got, hyphenGlyph)
	}

	got := FormatUnixSeconds(time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC).Unix())
	if len(got) < 19 {
		t.Errorf("FormatUnixSeconds(non-zero) = %q, want a YYYY-MM-DD HH:MM:SS string", got)
	}
}

func TestFormatMilliseconds(t *testing.T) {
	cases := []struct {
		want string
		in   int64
	}{
		{in: 0, want: "0ms"},
		{in: 500, want: "500ms"},
		{in: 1500, want: "1.5s"},
		{in: 90_000, want: "1m30s"},
		{in: 3_660_000, want: "1h1m"},
	}
	for _, c := range cases {
		if got := FormatMilliseconds(c.in); got != c.want {
			t.Errorf("FormatMilliseconds(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMatchesFilter(t *testing.T) {
	cases := []struct {
		name   string
		filter string
		want   bool
	}{
		{"liveness", "", true},
		{"liveness", "live", true},
		{"liveness", "LIVE", true},
		{"liveness", "ready", false},
		{"liveness", "liveness", true},
	}
	for _, c := range cases {
		if got := matchesFilter(c.name, c.filter); got != c.want {
			t.Errorf("matchesFilter(%q, %q) = %v, want %v", c.name, c.filter, got, c.want)
		}
	}
}
