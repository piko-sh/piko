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

package db_catalogue_postgres

import (
	"testing"

	"piko.sh/piko/internal/querier/querier_dto"
)

type echoTypeNormaliser struct {
	names []string
}

func (normaliser *echoTypeNormaliser) NormaliseTypeName(
	name string,
	modifiers ...int,
) querier_dto.SQLType {
	normaliser.names = append(normaliser.names, name)
	return querier_dto.SQLType{EngineName: name}
}

func TestLooksLikeTypeName(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "known scalar lowercase", token: "integer", want: true},
		{name: "known scalar uppercase", token: "TEXT", want: true},
		{name: "known scalar mixed case", token: "TimestampTZ", want: true},
		{name: "array suffix on known type", token: "integer[]", want: true},
		{name: "array suffix on unknown type", token: "mytype[]", want: true},
		{name: "plain parameter name", token: "user_id", want: false},
		{name: "user-defined type name", token: "address", want: false},
		{name: "empty token", token: "", want: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := looksLikeTypeName(testCase.token); got != testCase.want {
				t.Fatalf("looksLikeTypeName(%q) = %t, want %t", testCase.token, got, testCase.want)
			}
		})
	}
}

func TestParseReturnType(t *testing.T) {
	tests := []struct {
		name           string
		returnType     string
		wantType       string
		wantReturnsSet bool
	}{
		{name: "scalar", returnType: "integer", wantType: "integer", wantReturnsSet: false},
		{name: "setof", returnType: "SETOF integer", wantType: "integer", wantReturnsSet: true},
		{
			name:           "setof composite with spaces",
			returnType:     "SETOF my_schema.record_type",
			wantType:       "my_schema.record_type",
			wantReturnsSet: true,
		},
		{
			name:           "leading and trailing whitespace trimmed",
			returnType:     "  numeric  ",
			wantType:       "numeric",
			wantReturnsSet: false,
		},
		{
			name:           "setof prefix is case sensitive",
			returnType:     "setof integer",
			wantType:       "setof integer",
			wantReturnsSet: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			normaliser := &echoTypeNormaliser{}
			sqlType, returnsSet := parseReturnType(testCase.returnType, normaliser)

			if sqlType.EngineName != testCase.wantType {
				t.Fatalf("parseReturnType(%q) type = %q, want %q",
					testCase.returnType, sqlType.EngineName, testCase.wantType)
			}
			if returnsSet != testCase.wantReturnsSet {
				t.Fatalf("parseReturnType(%q) returnsSet = %t, want %t",
					testCase.returnType, returnsSet, testCase.wantReturnsSet)
			}
		})
	}
}

func TestParseSingleArgument(t *testing.T) {
	tests := []struct {
		name           string
		raw            string
		wantName       string
		wantType       string
		wantIsOptional bool
	}{
		{
			name:     "named scalar argument",
			raw:      "user_id integer",
			wantName: "user_id",
			wantType: "integer",
		},
		{
			name:     "nameless scalar argument",
			raw:      "integer",
			wantName: "",
			wantType: "integer",
		},
		{
			name:     "nameless multi-word type",
			raw:      "double precision",
			wantName: "",
			wantType: "double precision",
		},
		{
			name:     "named multi-word type",
			raw:      "amount double precision",
			wantName: "amount",
			wantType: "double precision",
		},
		{
			name:     "in mode prefix stripped",
			raw:      "IN user_id integer",
			wantName: "user_id",
			wantType: "integer",
		},
		{
			name:     "out mode prefix stripped",
			raw:      "OUT result_count bigint",
			wantName: "result_count",
			wantType: "bigint",
		},
		{
			name:     "inout mode prefix stripped",
			raw:      "INOUT counter integer",
			wantName: "counter",
			wantType: "integer",
		},
		{
			name:     "variadic mode prefix stripped",
			raw:      "VARIADIC items text[]",
			wantName: "items",
			wantType: "text[]",
		},
		{
			name:           "default clause sets optional",
			raw:            "limit_count integer DEFAULT 10",
			wantName:       "limit_count",
			wantType:       "integer",
			wantIsOptional: true,
		},
		{
			name:           "mode prefix with default clause",
			raw:            "IN flag boolean DEFAULT false",
			wantName:       "flag",
			wantType:       "boolean",
			wantIsOptional: true,
		},
		{
			name:     "nameless type with leading whitespace",
			raw:      "  text  ",
			wantName: "",
			wantType: "text",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			normaliser := &echoTypeNormaliser{}
			argument := parseSingleArgument(testCase.raw, normaliser)

			if argument.Name != testCase.wantName {
				t.Fatalf("parseSingleArgument(%q) name = %q, want %q",
					testCase.raw, argument.Name, testCase.wantName)
			}
			if argument.Type.EngineName != testCase.wantType {
				t.Fatalf("parseSingleArgument(%q) type = %q, want %q",
					testCase.raw, argument.Type.EngineName, testCase.wantType)
			}
			if argument.IsOptional != testCase.wantIsOptional {
				t.Fatalf("parseSingleArgument(%q) isOptional = %t, want %t",
					testCase.raw, argument.IsOptional, testCase.wantIsOptional)
			}
		})
	}
}

func TestParseFunctionArguments(t *testing.T) {
	type wantArgument struct {
		name       string
		typeName   string
		isOptional bool
	}

	tests := []struct {
		name      string
		arguments string
		want      []wantArgument
	}{
		{
			name:      "empty argument list",
			arguments: "",
			want:      nil,
		},
		{
			name:      "whitespace-only argument list",
			arguments: "   ",
			want:      nil,
		},
		{
			name:      "single named argument",
			arguments: "user_id integer",
			want: []wantArgument{
				{name: "user_id", typeName: "integer"},
			},
		},
		{
			name:      "multiple named arguments",
			arguments: "user_id integer, label text",
			want: []wantArgument{
				{name: "user_id", typeName: "integer"},
				{name: "label", typeName: "text"},
			},
		},
		{
			name:      "parenthesised type is not split on inner comma",
			arguments: "amount numeric(10,2), label text",
			want: []wantArgument{
				{name: "amount", typeName: "numeric(10,2)"},
				{name: "label", typeName: "text"},
			},
		},
		{
			name:      "default clauses mark optional arguments",
			arguments: "user_id integer, page_size integer DEFAULT 20",
			want: []wantArgument{
				{name: "user_id", typeName: "integer"},
				{name: "page_size", typeName: "integer", isOptional: true},
			},
		},
		{
			name:      "mixed modes and multi-word types",
			arguments: "IN user_id integer, OUT total double precision",
			want: []wantArgument{
				{name: "user_id", typeName: "integer"},
				{name: "total", typeName: "double precision"},
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			normaliser := &echoTypeNormaliser{}
			arguments := parseFunctionArguments(testCase.arguments, normaliser)

			if len(arguments) != len(testCase.want) {
				t.Fatalf("parseFunctionArguments(%q) length = %d, want %d",
					testCase.arguments, len(arguments), len(testCase.want))
			}

			for index, want := range testCase.want {
				got := arguments[index]
				if got.Name != want.name {
					t.Fatalf("parseFunctionArguments(%q)[%d] name = %q, want %q",
						testCase.arguments, index, got.Name, want.name)
				}
				if got.Type.EngineName != want.typeName {
					t.Fatalf("parseFunctionArguments(%q)[%d] type = %q, want %q",
						testCase.arguments, index, got.Type.EngineName, want.typeName)
				}
				if got.IsOptional != want.isOptional {
					t.Fatalf("parseFunctionArguments(%q)[%d] isOptional = %t, want %t",
						testCase.arguments, index, got.IsOptional, want.isOptional)
				}
			}
		})
	}
}
