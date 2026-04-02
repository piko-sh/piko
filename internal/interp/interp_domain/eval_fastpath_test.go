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
		name   string
		code   string
		expect any
	}{

		{"string_int", "import \"fp\"\nfp.StringInt(\"hello\")", int64(5)},

		{"string_bool_true", "import \"fp\"\nfp.StringBool(\"hi\")", true},
		{"string_bool_false", "import \"fp\"\nfp.StringBool(\"\")", false},

		{"string_rune_bool_true", "import \"fp\"\nfp.StringRuneBool(\"hello\", 'e')", true},
		{"string_rune_bool_false", "import \"fp\"\nfp.StringRuneBool(\"hello\", 'z')", false},

		{"string_rune_int_found", "import \"fp\"\nfp.StringRuneInt(\"hello\", 'l')", int64(2)},
		{"string_rune_int_missing", "import \"fp\"\nfp.StringRuneInt(\"hello\", 'z')", int64(-1)},

		{"string2_int_eq", "import \"fp\"\nfp.String2Int(\"a\", \"a\")", int64(0)},
		{"string2_int_ne", "import \"fp\"\nfp.String2Int(\"a\", \"b\")", int64(1)},

		{"string3_string", "import \"fp\"\nfp.String3String(\"a\", \"b\", \"c\")", "abc"},

		{"int_bool_true", "import \"fp\"\nfp.IntBool(5)", true},
		{"int_bool_false", "import \"fp\"\nfp.IntBool(-1)", false},

		{"int2_bool_true", "import \"fp\"\nfp.Int2Bool(5, 5)", true},
		{"int2_bool_false", "import \"fp\"\nfp.Int2Bool(5, 3)", false},

		{"int2_string_gt", "import \"fp\"\nfp.Int2String(5, 3)", "gt"},
		{"int2_string_le", "import \"fp\"\nfp.Int2String(3, 5)", "le"},

		{"float64_float64", "import \"fp\"\nfp.Float64Float64(2.5)", float64(5.0)},

		{"float642_float64", "import \"fp\"\nfp.Float642Float64(1.5, 2.5)", float64(4.0)},

		{"ret_float64", "import \"fp\"\nfp.RetFloat64()", float64(3.14)},

		{"ret_error_nil", "import \"fp\"\nerr := fp.RetError()\nerr == nil", true},

		{"void_int", "import \"fp\"\nfp.VoidInt(42)\ntrue", true},

		{"void_bool", "import \"fp\"\nfp.VoidBool(true)\ntrue", true},

		{"read_bool", "import \"fp\"\nfp.ReadBool(true)", false},
		{"read_float", "import \"fp\"\nfp.ReadFloat(2.5)", float64(2.5)},
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
