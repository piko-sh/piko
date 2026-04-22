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
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvalDivisionByZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		code    string
	}{
		{name: "integer division by zero literal", code: `1 / 0`, wantErr: errTypeCheck},
		{name: "integer remainder by zero literal", code: `1 % 0`, wantErr: errTypeCheck},
		{name: "integer division by zero variable", code: "x := 0\n10 / x", wantErr: errDivisionByZero},
		{name: "integer remainder by zero variable", code: "x := 0\n10 % x", wantErr: errDivisionByZero},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			_, err := service.Eval(context.Background(), tt.code)
			require.Error(t, err)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestEvalCompilationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		wantErr error
		name    string
		code    string
	}{
		{name: "syntax error", code: `func {{{`, wantErr: errParse},
		{name: "undefined variables", code: `x + y`, wantErr: errTypeCheck},
		{name: "type mismatch assignment", code: "var x int\nx = \"hello\"", wantErr: errTypeCheck},
		{name: "mismatched operand types", code: `1 + "hello"`, wantErr: errTypeCheck},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			_, err := service.Eval(context.Background(), tt.code)
			require.Error(t, err)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestEvalUnregisteredPackageImport(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{})

	_, err := service.Eval(context.Background(), `import "example.com/missing/pkg"
_ = 1`)
	require.Error(t, err)
	require.ErrorIs(t, err, errTypeCheck)
	require.ErrorIs(t, err, errPackageNotInRegistry)
	require.Contains(t, err.Error(), "piko extract discover")
	require.Contains(t, err.Error(), "example.com/missing/pkg")
}

func TestExtractMissingPackagePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "standard go/types shape",
			msg:  `could not import example.com/pkg (package "example.com/pkg" not registered with interpreter)`,
			want: "example.com/pkg",
		},
		{
			name: "missing prefix returns empty",
			msg:  `something unexpected about example.com/pkg`,
			want: "",
		},
		{
			name: "empty after prefix returns empty",
			msg:  "could not import ",
			want: "",
		},
		{
			name: "no space after path returns empty",
			msg:  "could not import example.com/only",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, extractMissingPackagePath(tt.msg))
		})
	}
}

func TestEvalIndexOutOfBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code string
	}{
		{name: "slice index out of bounds", code: "s := []int{1, 2, 3}\ns[5]"},
		{name: "string index out of bounds", code: "s := \"hello\"\ns[10]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService()
			_, err := service.Eval(context.Background(), tt.code)
			require.Error(t, err)
		})
	}
}

func TestEvalTypeAssertionFailure(t *testing.T) {
	t.Parallel()

	t.Run("failing type assertion produces error", func(t *testing.T) {
		t.Parallel()
		service := NewService()
		_, err := service.Eval(context.Background(), "var x interface{} = 42\nv := x.(string)\nv")
		require.Error(t, err)
	})
}
