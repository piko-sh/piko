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

package db_catalogue_sqlite

import (
	"testing"

	"piko.sh/piko/internal/querier/querier_dto"
)

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "plain identifier",
			identifier: "users",
			want:       `"users"`,
		},
		{
			name:       "empty identifier",
			identifier: "",
			want:       `""`,
		},
		{
			name:       "embedded space",
			identifier: "user table",
			want:       `"user table"`,
		},
		{
			name:       "single embedded double quote",
			identifier: `a"b`,
			want:       `"a""b"`,
		},
		{
			name:       "multiple embedded double quotes",
			identifier: `a"b"c`,
			want:       `"a""b""c"`,
		},
		{
			name:       "leading and trailing double quotes",
			identifier: `"x"`,
			want:       `"""x"""`,
		},
		{
			name:       "identifier already containing escaped quotes",
			identifier: `a""b`,
			want:       `"a""""b"`,
		},
		{
			name:       "unicode identifier preserved verbatim",
			identifier: "naïve_café",
			want:       `"naïve_café"`,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := quoteIdentifier(testCase.identifier)
			if got != testCase.want {
				t.Fatalf("quoteIdentifier(%q) = %q, want %q", testCase.identifier, got, testCase.want)
			}
		})
	}
}

func TestClassifyGeneratedColumn(t *testing.T) {
	tests := []struct {
		name              string
		hidden            int
		wantIsGenerated   bool
		wantGeneratedKind querier_dto.GeneratedKind
	}{
		{
			name:              "ordinary visible column",
			hidden:            0,
			wantIsGenerated:   false,
			wantGeneratedKind: querier_dto.GeneratedKindNone,
		},
		{
			name:              "hidden column that is not generated",
			hidden:            1,
			wantIsGenerated:   false,
			wantGeneratedKind: querier_dto.GeneratedKindNone,
		},
		{
			name:              "virtual generated column",
			hidden:            hiddenVirtualColumn,
			wantIsGenerated:   true,
			wantGeneratedKind: querier_dto.GeneratedKindVirtual,
		},
		{
			name:              "stored generated column",
			hidden:            hiddenStoredColumn,
			wantIsGenerated:   true,
			wantGeneratedKind: querier_dto.GeneratedKindStored,
		},
		{
			name:              "unknown flag treated as ordinary",
			hidden:            42,
			wantIsGenerated:   false,
			wantGeneratedKind: querier_dto.GeneratedKindNone,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			isGenerated, generatedKind := classifyGeneratedColumn(testCase.hidden)
			if isGenerated != testCase.wantIsGenerated {
				t.Fatalf("classifyGeneratedColumn(%d) isGenerated = %t, want %t",
					testCase.hidden, isGenerated, testCase.wantIsGenerated)
			}
			if generatedKind != testCase.wantGeneratedKind {
				t.Fatalf("classifyGeneratedColumn(%d) generatedKind = %d, want %d",
					testCase.hidden, generatedKind, testCase.wantGeneratedKind)
			}
		})
	}
}
