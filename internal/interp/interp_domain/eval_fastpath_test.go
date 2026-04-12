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
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func fpStringInt(s string) int   { return len(s) }
func fpStringBool(s string) bool { return s != "" }
func fpStringRuneBool(s string, r int32) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
func fpStringRuneInt(s string, r int32) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}
func fpString2Int(a, b string) int {
	if a == b {
		return 0
	}
	return 1
}
func fpString3String(a, b, c string) string { return a + b + c }
func fpIntBool(n int) bool                  { return n > 0 }
func fpInt2Bool(a, b int) bool              { return a == b }
func fpInt2String(a, b int) string {
	if a > b {
		return "gt"
	}
	return "le"
}
func fpFloat64Float64(x float64) float64     { return x * 2 }
func fpFloat642Float64(x, y float64) float64 { return x + y }
func fpRetFloat64() float64                  { return 3.14 }
func fpRetError() error                      { return nil }
func fpRetErrorNonNil() error                { return errors.New("oops") }
func fpVoidInt(_ int)                        {}
func fpVoidBool(_ bool)                      {}
func fpReadFloat(x float64) float64          { return x }
func fpReadBool(b bool) bool                 { return !b }

func TestNativeFastpathDispatchers(t *testing.T) {
	t.Parallel()

	service := newTestServiceWithSymbols(t, SymbolExports{
		"fp": {
			"StringInt":       reflect.ValueOf(fpStringInt),
			"StringBool":      reflect.ValueOf(fpStringBool),
			"StringRuneBool":  reflect.ValueOf(fpStringRuneBool),
			"StringRuneInt":   reflect.ValueOf(fpStringRuneInt),
			"String2Int":      reflect.ValueOf(fpString2Int),
			"String3String":   reflect.ValueOf(fpString3String),
			"IntBool":         reflect.ValueOf(fpIntBool),
			"Int2Bool":        reflect.ValueOf(fpInt2Bool),
			"Int2String":      reflect.ValueOf(fpInt2String),
			"Float64Float64":  reflect.ValueOf(fpFloat64Float64),
			"Float642Float64": reflect.ValueOf(fpFloat642Float64),
			"RetFloat64":      reflect.ValueOf(fpRetFloat64),
			"RetError":        reflect.ValueOf(fpRetError),
			"RetErrorNonNil":  reflect.ValueOf(fpRetErrorNonNil),
			"VoidInt":         reflect.ValueOf(fpVoidInt),
			"VoidBool":        reflect.ValueOf(fpVoidBool),
			"ReadFloat":       reflect.ValueOf(fpReadFloat),
			"ReadBool":        reflect.ValueOf(fpReadBool),
		},
	})

	tests := []struct {
		expect any
		name   string
		code   string
	}{

		{name: "string_int", code: "import \"fp\"\nfp.StringInt(\"hello\")", expect: int64(5)},

		{name: "string_bool_true", code: "import \"fp\"\nfp.StringBool(\"hi\")", expect: true},
		{name: "string_bool_false", code: "import \"fp\"\nfp.StringBool(\"\")", expect: false},

		{name: "string_rune_bool_true", code: "import \"fp\"\nfp.StringRuneBool(\"hello\", 'e')", expect: true},
		{name: "string_rune_bool_false", code: "import \"fp\"\nfp.StringRuneBool(\"hello\", 'z')", expect: false},

		{name: "string_rune_int_found", code: "import \"fp\"\nfp.StringRuneInt(\"hello\", 'l')", expect: int64(2)},
		{name: "string_rune_int_missing", code: "import \"fp\"\nfp.StringRuneInt(\"hello\", 'z')", expect: int64(-1)},

		{name: "string2_int_eq", code: "import \"fp\"\nfp.String2Int(\"a\", \"a\")", expect: int64(0)},
		{name: "string2_int_ne", code: "import \"fp\"\nfp.String2Int(\"a\", \"b\")", expect: int64(1)},

		{name: "string3_string", code: "import \"fp\"\nfp.String3String(\"a\", \"b\", \"c\")", expect: "abc"},

		{name: "int_bool_true", code: "import \"fp\"\nfp.IntBool(5)", expect: true},
		{name: "int_bool_false", code: "import \"fp\"\nfp.IntBool(-1)", expect: false},

		{name: "int2_bool_true", code: "import \"fp\"\nfp.Int2Bool(5, 5)", expect: true},
		{name: "int2_bool_false", code: "import \"fp\"\nfp.Int2Bool(5, 3)", expect: false},

		{name: "int2_string_gt", code: "import \"fp\"\nfp.Int2String(5, 3)", expect: "gt"},
		{name: "int2_string_le", code: "import \"fp\"\nfp.Int2String(3, 5)", expect: "le"},

		{name: "float64_float64", code: "import \"fp\"\nfp.Float64Float64(2.5)", expect: float64(5.0)},

		{name: "float642_float64", code: "import \"fp\"\nfp.Float642Float64(1.5, 2.5)", expect: float64(4.0)},

		{name: "ret_float64", code: "import \"fp\"\nfp.RetFloat64()", expect: float64(3.14)},

		{name: "ret_error_nil", code: "import \"fp\"\nerr := fp.RetError()\nerr == nil", expect: true},

		{name: "void_int", code: "import \"fp\"\nfp.VoidInt(42)\ntrue", expect: true},

		{name: "void_bool", code: "import \"fp\"\nfp.VoidBool(true)\ntrue", expect: true},

		{name: "read_bool", code: "import \"fp\"\nfp.ReadBool(true)", expect: false},
		{name: "read_float", code: "import \"fp\"\nfp.ReadFloat(2.5)", expect: float64(2.5)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			localService := service.Clone()
			result, err := localService.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
