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

package engine_shared_test

import (
	"testing"

	"piko.sh/piko/internal/querier/querier_adapters/engine_shared"
)

var (
	postgresLikeComments = engine_shared.CommentRules{NestedBlockComments: true}
	mysqlLikeComments    = engine_shared.CommentRules{
		DoubleDashRequiresWhitespace: true,
		HashLineComment:              true,
	}
	sqliteLikeComments = engine_shared.CommentRules{}
)

func TestSkipWhitespaceAndComments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		rules    engine_shared.CommentRules
		input    string
		position int
		want     int
	}{
		{name: "leading whitespace", rules: postgresLikeComments, input: "  \t\n x", want: 5},
		{name: "line comment to newline", rules: postgresLikeComments, input: "-- skip me\nx", want: 11},
		{name: "line comment to end", rules: postgresLikeComments, input: "-- skip me", want: 10},
		{name: "block comment", rules: postgresLikeComments, input: "/* hi */x", want: 8},
		{name: "nested block comment", rules: postgresLikeComments, input: "/* a /* b */ c */x", want: 17},
		{name: "non-nested stops at first close", rules: sqliteLikeComments, input: "/* a /* b */ c */x", want: 13},
		{name: "mysql hash comment", rules: mysqlLikeComments, input: "# skip\nx", want: 7},
		{name: "mysql double dash needs space", rules: mysqlLikeComments, input: "--x", want: 0},
		{name: "mysql double dash with space", rules: mysqlLikeComments, input: "-- x\ny", want: 5},
		{name: "mysql double dash before newline", rules: mysqlLikeComments, input: "--\nFROM", want: 3},
		{name: "mysql double dash before carriage return", rules: mysqlLikeComments, input: "--\rFROM", want: 3},
		{name: "mysql double dash at end of input", rules: mysqlLikeComments, input: "--", want: 2},
		{name: "mysql double dash before tab", rules: mysqlLikeComments, input: "--\tx", want: 4},
		{name: "no comment leaves position", rules: postgresLikeComments, input: "-x", want: 0},
		{name: "interleaved", rules: postgresLikeComments, input: "  -- a\n /* b */\tx", want: 16},
	}

	for index := range cases {
		testCase := cases[index]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := testCase.rules.SkipWhitespaceAndComments(testCase.input, testCase.position)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != testCase.want {
				t.Fatalf("position = %d, want %d", got, testCase.want)
			}
		})
	}
}

func TestSkipWhitespaceAndCommentsUnterminated(t *testing.T) {
	t.Parallel()

	for _, rules := range []engine_shared.CommentRules{postgresLikeComments, mysqlLikeComments, sqliteLikeComments} {
		_, err := rules.SkipWhitespaceAndComments("/* never closed", 0)
		if err == nil {
			t.Fatalf("expected unterminated block comment error for rules %+v", rules)
		}
	}
}

func TestTryReadPrefixedNumber(t *testing.T) {
	t.Parallel()

	allBases := engine_shared.NumberRules{HexPrefix: true, OctalPrefix: true, BinaryPrefix: true}
	hexOnly := engine_shared.NumberRules{HexPrefix: true}

	cases := []struct {
		name        string
		rules       engine_shared.NumberRules
		input       string
		wantEnd     int
		wantMatched bool
	}{
		{name: "hex", rules: allBases, input: "0xDEAD", wantEnd: 6, wantMatched: true},
		{name: "octal", rules: allBases, input: "0o777", wantEnd: 5, wantMatched: true},
		{name: "binary", rules: allBases, input: "0b1010", wantEnd: 6, wantMatched: true},
		{name: "octal disabled", rules: hexOnly, input: "0o777", wantEnd: 0, wantMatched: false},
		{name: "binary disabled", rules: hexOnly, input: "0b10", wantEnd: 0, wantMatched: false},
		{name: "plain decimal not prefixed", rules: allBases, input: "123", wantEnd: 0, wantMatched: false},
		{name: "zero alone", rules: allBases, input: "0", wantEnd: 0, wantMatched: false},
		{name: "hex stops at non-hex", rules: allBases, input: "0xFFg", wantEnd: 4, wantMatched: true},
	}

	for index := range cases {
		testCase := cases[index]
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			end, matched, err := testCase.rules.TryReadPrefixedNumber(testCase.input, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if matched != testCase.wantMatched {
				t.Fatalf("matched = %v, want %v", matched, testCase.wantMatched)
			}
			if end != testCase.wantEnd {
				t.Fatalf("end = %d, want %d", end, testCase.wantEnd)
			}
		})
	}
}

func TestTryReadPrefixedNumberRequireDigits(t *testing.T) {
	t.Parallel()

	lenient := engine_shared.NumberRules{HexPrefix: true}
	if _, matched, err := lenient.TryReadPrefixedNumber("0x", 0); err != nil || !matched {
		t.Fatalf("lenient bare prefix: matched=%v err=%v, want matched=true err=nil", matched, err)
	}

	strict := engine_shared.NumberRules{HexPrefix: true, RequireDigitsAfterPrefix: true}
	end, matched, err := strict.TryReadPrefixedNumber("0x", 0)
	if err == nil {
		t.Fatal("strict bare prefix: expected error, got nil")
	}
	if matched {
		t.Fatal("strict bare prefix: matched should be false on error")
	}
	if end != 0 {
		t.Fatalf("strict bare prefix: end = %d, want 0", end)
	}
	if _, matched, err := strict.TryReadPrefixedNumber("0xAB", 0); err != nil || !matched {
		t.Fatalf("strict with digits: matched=%v err=%v, want matched=true err=nil", matched, err)
	}
}

func TestPrefixValidatorDisabledBases(t *testing.T) {
	t.Parallel()

	hexOnly := engine_shared.NumberRules{HexPrefix: true}
	if hexOnly.PrefixValidator('x') == nil {
		t.Fatal("hex validator should be present when HexPrefix is set")
	}
	if hexOnly.PrefixValidator('o') != nil {
		t.Fatal("octal validator should be nil when OctalPrefix is unset")
	}
	if hexOnly.PrefixValidator('b') != nil {
		t.Fatal("binary validator should be nil when BinaryPrefix is unset")
	}
	if hexOnly.PrefixValidator('z') != nil {
		t.Fatal("unknown prefix should yield a nil validator")
	}
}
