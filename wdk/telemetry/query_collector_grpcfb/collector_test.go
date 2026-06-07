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

package query_collector_grpcfb

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"piko.sh/piko"
)

func TestRedactSQLStripsLiteralsKeepsIdentifiers(t *testing.T) {
	cases := []struct {
		name        string
		in          string
		want        string
		mustNotHave []string
		mustHave    []string
	}{
		{
			name:        "string and numeric literals are redacted",
			in:          "SELECT * FROM users WHERE email = 'alice@example.com' AND age = 42",
			want:        "SELECT * FROM users WHERE email = '?' AND age = ?",
			mustNotHave: []string{"alice@example.com", "42"},
		},
		{
			name:     "identifiers with embedded digits are preserved",
			in:       "SELECT col2 FROM t1",
			want:     "SELECT col2 FROM t1",
			mustHave: []string{"col2", "t1"},
		},
		{
			name:        "doubled single-quote escapes collapse to one literal",
			in:          "SELECT * FROM t WHERE n = 'O''Brien'",
			want:        "SELECT * FROM t WHERE n = '?'",
			mustNotHave: []string{"Brien"},
		},
		{
			name:        "decimal literals are redacted",
			in:          "SELECT * FROM t WHERE price > 3.14",
			want:        "SELECT * FROM t WHERE price > ?",
			mustNotHave: []string{"3.14"},
		},
		{
			name:        "the documented example redacts both literals",
			in:          "SELECT * FROM t WHERE name='alice' AND id=42",
			want:        "SELECT * FROM t WHERE name='?' AND id=?",
			mustNotHave: []string{"alice", "42"},
		},
		{
			name:        "hex literals are redacted",
			in:          "SELECT * FROM t WHERE flags = 0x1A",
			want:        "SELECT * FROM t WHERE flags = ?",
			mustNotHave: []string{"0x1A", "1A"},
		},
		{
			name:        "scientific notation literals are redacted",
			in:          "SELECT * FROM t WHERE big = 1e10",
			want:        "SELECT * FROM t WHERE big = ?",
			mustNotHave: []string{"1e10"},
		},
		{
			name:        "signed numeric literals are redacted",
			in:          "SELECT * FROM t WHERE n = -42",
			want:        "SELECT * FROM t WHERE n = ?",
			mustNotHave: []string{"-42", "42"},
		},
		{
			name:        "signed scientific literals are redacted",
			in:          "SELECT * FROM t WHERE n = +1.5E-3",
			want:        "SELECT * FROM t WHERE n = ?",
			mustNotHave: []string{"1.5E-3", "1.5"},
		},
		{
			name:        "tagged dollar-quoted strings are redacted",
			in:          "SELECT $tag$secret pii$tag$ AS x",
			want:        "SELECT '?' AS x",
			mustNotHave: []string{"secret pii"},
		},
		{
			name:        "empty-tag dollar-quoted strings are redacted",
			in:          "SELECT $$secret pii$$ AS x",
			want:        "SELECT '?' AS x",
			mustNotHave: []string{"secret pii"},
		},
		{
			name:        "escape-string bodies are redacted",
			in:          `INSERT INTO u VALUES (E'O\'Brien')`,
			want:        "INSERT INTO u VALUES ('?')",
			mustNotHave: []string{"Brien"},
		},
		{
			name:        "doubled-quote escape-string bodies are redacted",
			in:          "INSERT INTO u VALUES (E'O''Brien')",
			want:        "INSERT INTO u VALUES ('?')",
			mustNotHave: []string{"Brien"},
		},
		{
			name:        "adjacent numeric literals are each redacted",
			in:          "SELECT * FROM t WHERE id IN (1,2,3)",
			want:        "SELECT * FROM t WHERE id IN (?,?,?)",
			mustNotHave: []string{"1,2,3"},
		},
		{
			name:     "version-like identifiers are preserved",
			in:       "SELECT x2y FROM v1.2.3",
			want:     "SELECT x2y FROM v1.2.3",
			mustHave: []string{"x2y", "v1.2.3"},
		},
		{
			name: "empty input is unchanged",
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := redactSQL(tc.in)
			assert.Equal(t, tc.want, got)
			for _, frag := range tc.mustNotHave {
				assert.NotContains(t, got, frag)
			}
			for _, frag := range tc.mustHave {
				assert.Contains(t, got, frag)
			}
		})
	}
}

func TestObserveQueryNilReceiverIsNoOp(t *testing.T) {
	ctx := context.Background()

	assert.NotPanics(t, func() {
		var c *Collector
		c.ObserveQuery(ctx, &piko.QueryObservation{Statement: "SELECT 1"})
	})

	assert.NotPanics(t, func() {
		New(nil).ObserveQuery(ctx, &piko.QueryObservation{Statement: "SELECT 1"})
	})
}

func TestObserveQueryNilObservationIsNoOp(t *testing.T) {
	assert.NotPanics(t, func() {
		New(nil).ObserveQuery(context.Background(), nil)
	})
}

func TestRedactSQLDoesNotLeakLongString(t *testing.T) {
	secret := strings.Repeat("a", 9000)
	got := redactSQL("SELECT * FROM t WHERE c = '" + secret + "'")
	assert.NotContains(t, got, secret)
	assert.Equal(t, "SELECT * FROM t WHERE c = '?'", got)
}
