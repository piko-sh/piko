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

package interp_domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSkipVerbFlagChars(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  int
	}{
		{input: "+0v", want: 2},
		{input: "-#5v", want: 2},
		{input: "  d", want: 2},
		{input: "v", want: 0},
		{input: "", want: 0},
		{input: "#+", want: 2},
	}
	for _, tc := range cases {
		got := skipVerbFlagChars([]rune(tc.input), 0)
		require.Equal(t, tc.want, got, "skipVerbFlagChars(%q)", tc.input)
	}
}

func TestSkipVerbWidth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  int
	}{
		{input: "10v", want: 2},
		{input: "*v", want: 1},
		{input: "123*v", want: 4},
		{input: "v", want: 0},
		{input: "", want: 0},
	}
	for _, tc := range cases {
		got := skipVerbWidth([]rune(tc.input), 0)
		require.Equal(t, tc.want, got, "skipVerbWidth(%q)", tc.input)
	}
}

func TestSkipVerbPrecision(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  int
	}{
		{input: "3v", want: 1},
		{input: "*v", want: 1},
		{input: "v", want: 0},
		{input: "", want: 0},
	}
	for _, tc := range cases {
		got := skipVerbPrecision([]rune(tc.input), 0)
		require.Equal(t, tc.want, got, "skipVerbPrecision(%q)", tc.input)
	}
}

func TestSkipVerbFlagsAndWidth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  int
	}{
		{input: "v", want: 0},
		{input: "10v", want: 2},
		{input: "-10.3f", want: 5},
		{input: "+#0 v", want: 4},
		{input: ".3v", want: 2},
		{input: "*.*v", want: 3},
		{input: "", want: 0},
	}
	for _, tc := range cases {
		got := skipVerbFlagsAndWidth([]rune(tc.input), 0)
		require.Equal(t, tc.want, got, "skipVerbFlagsAndWidth(%q)", tc.input)
	}
}

func TestIndexOfRune(t *testing.T) {
	t.Parallel()
	require.Equal(t, 0, indexOfRune([]rune("hello"), 0, 'h'))
	require.Equal(t, 4, indexOfRune([]rune("hello"), 0, 'o'))
	require.Equal(t, 4, indexOfRune([]rune("hello"), 4, 'o'))
	require.Equal(t, -1, indexOfRune([]rune("hello"), 0, 'z'))
	require.Equal(t, -1, indexOfRune([]rune("hello"), 5, 'o'))
}

func TestParsePositiveInt(t *testing.T) {
	t.Parallel()

	value, ok := parsePositiveInt("42")
	require.True(t, ok)
	require.Equal(t, 42, value)

	_, ok = parsePositiveInt("")
	require.False(t, ok)

	_, ok = parsePositiveInt("abc")
	require.False(t, ok)

	_, ok = parsePositiveInt("0")
	require.False(t, ok, "zero should not count as positive")

	_, ok = parsePositiveInt("12a3")
	require.False(t, ok, "non-numeric character should fail")
}

func TestStripPikoSentinelFromTypeString(t *testing.T) {
	t.Parallel()

	noSentinel := "main.Person"
	require.Equal(t, noSentinel, stripPikoSentinelFromTypeString(noSentinel),
		"strings without the sentinel pass through")

	withSentinel := "struct { Name string; " + pikoIDFieldPrefix + " any }"
	got := stripPikoSentinelFromTypeString(withSentinel)
	require.NotContains(t, got, pikoIDFieldPrefix,
		"sentinel segment should be removed")
}

func TestStripPikoSentinelFromTypeString_FirstFieldDropsTrailingFields(t *testing.T) {
	t.Parallel()
	first := "struct { " + pikoIDFieldPrefix + " any; Name string }"
	got := stripPikoSentinelFromTypeString(first)
	require.NotContains(t, got, pikoIDFieldPrefix,
		"sentinel must be removed regardless of position")
}

func TestStripPikoSentinelFromTypeString_NoBraceReturnsUnchanged(t *testing.T) {
	t.Parallel()
	weird := "prefix " + pikoIDFieldPrefix + " trailing"
	got := stripPikoSentinelFromTypeString(weird)
	require.Equal(t, weird, got, "missing closing brace leaves input unchanged")
}

func TestWrapPikoSynthesisedFmtArg_IdentityForNonSynth(t *testing.T) {
	t.Parallel()
	require.Equal(t, "hello", wrapPikoSynthesisedFmtArg(nil, "hello"))
	require.Equal(t, 42, wrapPikoSynthesisedFmtArg(nil, 42))
	require.Nil(t, wrapPikoSynthesisedFmtArg(nil, nil))
}
