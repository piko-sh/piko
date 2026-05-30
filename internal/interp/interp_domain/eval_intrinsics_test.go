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
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func newIntrinsicsService(t *testing.T) *Service {
	t.Helper()
	return newTestServiceWithSymbols(t, SymbolExports{
		"strings": {
			"Contains":     reflect.ValueOf(strings.Contains),
			"ContainsRune": reflect.ValueOf(strings.ContainsRune),
			"Count":        reflect.ValueOf(strings.Count),
			"EqualFold":    reflect.ValueOf(strings.EqualFold),
			"HasPrefix":    reflect.ValueOf(strings.HasPrefix),
			"HasSuffix":    reflect.ValueOf(strings.HasSuffix),
			"Index":        reflect.ValueOf(strings.Index),
			"IndexRune":    reflect.ValueOf(strings.IndexRune),
			"Join":         reflect.ValueOf(strings.Join),
			"LastIndex":    reflect.ValueOf(strings.LastIndex),
			"Repeat":       reflect.ValueOf(strings.Repeat),
			"ReplaceAll":   reflect.ValueOf(strings.ReplaceAll),
			"Split":        reflect.ValueOf(strings.Split),
			"ToLower":      reflect.ValueOf(strings.ToLower),
			"ToUpper":      reflect.ValueOf(strings.ToUpper),
			"Trim":         reflect.ValueOf(strings.Trim),
			"TrimPrefix":   reflect.ValueOf(strings.TrimPrefix),
			"TrimSpace":    reflect.ValueOf(strings.TrimSpace),
			"TrimSuffix":   reflect.ValueOf(strings.TrimSuffix),
		},
		"math": {
			"Abs":   reflect.ValueOf(math.Abs),
			"Ceil":  reflect.ValueOf(math.Ceil),
			"Cos":   reflect.ValueOf(math.Cos),
			"Exp":   reflect.ValueOf(math.Exp),
			"Floor": reflect.ValueOf(math.Floor),
			"Max":   reflect.ValueOf(math.Max),
			"Min":   reflect.ValueOf(math.Min),
			"Mod":   reflect.ValueOf(math.Mod),
			"Pow":   reflect.ValueOf(math.Pow),
			"Round": reflect.ValueOf(math.Round),
			"Sin":   reflect.ValueOf(math.Sin),
			"Sqrt":  reflect.ValueOf(math.Sqrt),
			"Tan":   reflect.ValueOf(math.Tan),
			"Trunc": reflect.ValueOf(math.Trunc),
		},
		"strconv": {
			"FormatBool": reflect.ValueOf(strconv.FormatBool),
			"FormatInt":  reflect.ValueOf(strconv.FormatInt),
			"Itoa":       reflect.ValueOf(strconv.Itoa),
		},
	})
}

func TestIntrinsicStringFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{

		{name: "ContainsRune_found", code: "import \"strings\"\nstrings.ContainsRune(\"hello\", 'e')", expect: true},
		{name: "ContainsRune_not_found", code: "import \"strings\"\nstrings.ContainsRune(\"hello\", 'z')", expect: false},
		{name: "ContainsRune_empty", code: "import \"strings\"\nstrings.ContainsRune(\"\", 'a')", expect: false},

		{name: "EqualFold_true", code: "import \"strings\"\nstrings.EqualFold(\"Go\", \"go\")", expect: true},
		{name: "EqualFold_false", code: "import \"strings\"\nstrings.EqualFold(\"Go\", \"py\")", expect: false},
		{name: "EqualFold_empty", code: "import \"strings\"\nstrings.EqualFold(\"\", \"\")", expect: true},

		{name: "Trim_spaces", code: "import \"strings\"\nstrings.Trim(\"  hello  \", \" \")", expect: "hello"},
		{name: "Trim_chars", code: "import \"strings\"\nstrings.Trim(\"!!hello!!\", \"!\")", expect: "hello"},
		{name: "Trim_noop", code: "import \"strings\"\nstrings.Trim(\"hello\", \"!\")", expect: "hello"},

		{name: "IndexRune_found", code: "import \"strings\"\nstrings.IndexRune(\"hello\", 'l')", expect: int64(2)},
		{name: "IndexRune_not_found", code: "import \"strings\"\nstrings.IndexRune(\"hello\", 'z')", expect: int64(-1)},
		{name: "IndexRune_first", code: "import \"strings\"\nstrings.IndexRune(\"hello\", 'h')", expect: int64(0)},

		{name: "LastIndex_found", code: "import \"strings\"\nstrings.LastIndex(\"go gopher\", \"go\")", expect: int64(3)},
		{name: "LastIndex_not_found", code: "import \"strings\"\nstrings.LastIndex(\"hello\", \"xyz\")", expect: int64(-1)},
		{name: "LastIndex_single", code: "import \"strings\"\nstrings.LastIndex(\"abcabc\", \"c\")", expect: int64(5)},

		{name: "Join_comma", code: "import \"strings\"\nstrings.Join([]string{\"a\", \"b\", \"c\"}, \",\")", expect: "a,b,c"},
		{name: "Join_empty_sep", code: "import \"strings\"\nstrings.Join([]string{\"a\", \"b\"}, \"\")", expect: "ab"},
		{name: "Join_single", code: "import \"strings\"\nstrings.Join([]string{\"only\"}, \",\")", expect: "only"},

		{name: "Split_comma", code: "import \"strings\"\nlen(strings.Split(\"a,b,c\", \",\"))", expect: int64(3)},
		{name: "Split_no_sep", code: "import \"strings\"\nlen(strings.Split(\"hello\", \",\"))", expect: int64(1)},

		{name: "ReplaceAll_basic", code: "import \"strings\"\nstrings.ReplaceAll(\"aaa\", \"a\", \"b\")", expect: "bbb"},
		{name: "ReplaceAll_no_match", code: "import \"strings\"\nstrings.ReplaceAll(\"hello\", \"z\", \"x\")", expect: "hello"},
		{name: "ReplaceAll_empty_old", code: "import \"strings\"\nstrings.ReplaceAll(\"hi\", \"\", \"-\")", expect: "-h-i-"},

		{name: "Repeat_3", code: "import \"strings\"\nstrings.Repeat(\"ab\", 3)", expect: "ababab"},
		{name: "Repeat_0", code: "import \"strings\"\nstrings.Repeat(\"ab\", 0)", expect: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := newIntrinsicsService(t)
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestIntrinsicMathFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect float64
		delta  float64
	}{

		{name: "Abs_negative", code: "import \"math\"\nmath.Abs(-3.14)", expect: 3.14, delta: 0},
		{name: "Abs_positive", code: "import \"math\"\nmath.Abs(3.14)", expect: 3.14, delta: 0},
		{name: "Abs_zero", code: "import \"math\"\nmath.Abs(0.0)", expect: 0.0, delta: 0},

		{name: "Sqrt_16", code: "import \"math\"\nmath.Sqrt(16.0)", expect: 4.0, delta: 0},
		{name: "Sqrt_2", code: "import \"math\"\nmath.Sqrt(2.0)", expect: math.Sqrt(2.0), delta: 1e-10},

		{name: "Floor_positive", code: "import \"math\"\nmath.Floor(3.7)", expect: 3.0, delta: 0},
		{name: "Floor_negative", code: "import \"math\"\nmath.Floor(-3.2)", expect: -4.0, delta: 0},

		{name: "Ceil_positive", code: "import \"math\"\nmath.Ceil(3.2)", expect: 4.0, delta: 0},
		{name: "Ceil_negative", code: "import \"math\"\nmath.Ceil(-3.7)", expect: -3.0, delta: 0},

		{name: "Trunc_positive", code: "import \"math\"\nmath.Trunc(3.9)", expect: 3.0, delta: 0},
		{name: "Trunc_negative", code: "import \"math\"\nmath.Trunc(-3.9)", expect: -3.0, delta: 0},

		{name: "Round_up", code: "import \"math\"\nmath.Round(3.5)", expect: 4.0, delta: 0},
		{name: "Round_down", code: "import \"math\"\nmath.Round(3.4)", expect: 3.0, delta: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := newIntrinsicsService(t)
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			if tt.delta > 0 {
				require.InDelta(t, tt.expect, result, tt.delta)
			} else {
				require.Equal(t, tt.expect, result)
			}
		})
	}
}

func TestGoDispatchIntrinsicStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "ContainsRune", code: "import \"strings\"\nstrings.ContainsRune(\"hello\", 'e')", expect: true},
		{name: "EqualFold", code: "import \"strings\"\nstrings.EqualFold(\"Go\", \"go\")", expect: true},
		{name: "Trim", code: "import \"strings\"\nstrings.Trim(\"  hello  \", \" \")", expect: "hello"},
		{name: "IndexRune", code: "import \"strings\"\nstrings.IndexRune(\"hello\", 'l')", expect: int64(2)},
		{name: "LastIndex", code: "import \"strings\"\nstrings.LastIndex(\"go gopher\", \"go\")", expect: int64(3)},
		{name: "Join", code: "import \"strings\"\nstrings.Join([]string{\"a\", \"b\"}, \",\")", expect: "a,b"},
		{name: "Split", code: "import \"strings\"\nlen(strings.Split(\"a,b,c\", \",\"))", expect: int64(3)},
		{name: "ReplaceAll", code: "import \"strings\"\nstrings.ReplaceAll(\"aaa\", \"a\", \"b\")", expect: "bbb"},
		{name: "Repeat", code: "import \"strings\"\nstrings.Repeat(\"ab\", 3)", expect: "ababab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := newIntrinsicsService(t)
			service = NewService(WithForceGoDispatch())
			service.UseSymbols(NewSymbolRegistry(SymbolExports{
				"strings": {
					"Contains":     reflect.ValueOf(strings.Contains),
					"ContainsRune": reflect.ValueOf(strings.ContainsRune),
					"EqualFold":    reflect.ValueOf(strings.EqualFold),
					"Index":        reflect.ValueOf(strings.Index),
					"IndexRune":    reflect.ValueOf(strings.IndexRune),
					"Join":         reflect.ValueOf(strings.Join),
					"LastIndex":    reflect.ValueOf(strings.LastIndex),
					"Repeat":       reflect.ValueOf(strings.Repeat),
					"ReplaceAll":   reflect.ValueOf(strings.ReplaceAll),
					"Split":        reflect.ValueOf(strings.Split),
					"Trim":         reflect.ValueOf(strings.Trim),
				},
			}))
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestGoDispatchIntrinsicMath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   string
		expect float64
	}{
		{name: "Abs", code: "import \"math\"\nmath.Abs(-3.14)", expect: 3.14},
		{name: "Sqrt", code: "import \"math\"\nmath.Sqrt(16.0)", expect: 4.0},
		{name: "Floor", code: "import \"math\"\nmath.Floor(3.7)", expect: 3.0},
		{name: "Ceil", code: "import \"math\"\nmath.Ceil(3.2)", expect: 4.0},
		{name: "Trunc", code: "import \"math\"\nmath.Trunc(3.9)", expect: 3.0},
		{name: "Round", code: "import \"math\"\nmath.Round(3.5)", expect: 4.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(WithForceGoDispatch())
			service.UseSymbols(NewSymbolRegistry(SymbolExports{
				"math": {
					"Abs":   reflect.ValueOf(math.Abs),
					"Ceil":  reflect.ValueOf(math.Ceil),
					"Floor": reflect.ValueOf(math.Floor),
					"Round": reflect.ValueOf(math.Round),
					"Sqrt":  reflect.ValueOf(math.Sqrt),
					"Trunc": reflect.ValueOf(math.Trunc),
				},
			}))
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestGoDispatchIntrinsicStrconv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "FormatBool", code: "import \"strconv\"\nstrconv.FormatBool(true)", expect: "true"},
		{name: "FormatInt", code: "import \"strconv\"\nstrconv.FormatInt(42, 10)", expect: "42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := NewService(WithForceGoDispatch())
			service.UseSymbols(NewSymbolRegistry(SymbolExports{
				"strconv": {
					"FormatBool": reflect.ValueOf(strconv.FormatBool),
					"FormatInt":  reflect.ValueOf(strconv.FormatInt),
				},
			}))
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}

func TestIntrinsicStrconvFunctions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		expect any
		name   string
		code   string
	}{
		{name: "FormatBool_true", code: "import \"strconv\"\nstrconv.FormatBool(true)", expect: "true"},
		{name: "FormatBool_false", code: "import \"strconv\"\nstrconv.FormatBool(false)", expect: "false"},
		{name: "FormatInt_base10", code: "import \"strconv\"\nstrconv.FormatInt(42, 10)", expect: "42"},
		{name: "FormatInt_base16", code: "import \"strconv\"\nstrconv.FormatInt(255, 16)", expect: "ff"},
		{name: "FormatInt_base2", code: "import \"strconv\"\nstrconv.FormatInt(10, 2)", expect: "1010"},
		{name: "FormatInt_negative", code: "import \"strconv\"\nstrconv.FormatInt(-42, 10)", expect: "-42"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			service := newIntrinsicsService(t)
			result, err := service.Eval(context.Background(), tt.code)
			require.NoError(t, err)
			require.Equal(t, tt.expect, result)
		})
	}
}
