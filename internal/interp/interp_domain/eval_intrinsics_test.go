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
		name   string
		code   string
		expect any
	}{

		{"ContainsRune_found", "import \"strings\"\nstrings.ContainsRune(\"hello\", 'e')", true},
		{"ContainsRune_not_found", "import \"strings\"\nstrings.ContainsRune(\"hello\", 'z')", false},
		{"ContainsRune_empty", "import \"strings\"\nstrings.ContainsRune(\"\", 'a')", false},

		{"EqualFold_true", "import \"strings\"\nstrings.EqualFold(\"Go\", \"go\")", true},
		{"EqualFold_false", "import \"strings\"\nstrings.EqualFold(\"Go\", \"py\")", false},
		{"EqualFold_empty", "import \"strings\"\nstrings.EqualFold(\"\", \"\")", true},

		{"Trim_spaces", "import \"strings\"\nstrings.Trim(\"  hello  \", \" \")", "hello"},
		{"Trim_chars", "import \"strings\"\nstrings.Trim(\"!!hello!!\", \"!\")", "hello"},
		{"Trim_noop", "import \"strings\"\nstrings.Trim(\"hello\", \"!\")", "hello"},

		{"IndexRune_found", "import \"strings\"\nstrings.IndexRune(\"hello\", 'l')", int64(2)},
		{"IndexRune_not_found", "import \"strings\"\nstrings.IndexRune(\"hello\", 'z')", int64(-1)},
		{"IndexRune_first", "import \"strings\"\nstrings.IndexRune(\"hello\", 'h')", int64(0)},

		{"LastIndex_found", "import \"strings\"\nstrings.LastIndex(\"go gopher\", \"go\")", int64(3)},
		{"LastIndex_not_found", "import \"strings\"\nstrings.LastIndex(\"hello\", \"xyz\")", int64(-1)},
		{"LastIndex_single", "import \"strings\"\nstrings.LastIndex(\"abcabc\", \"c\")", int64(5)},

		{"Join_comma", "import \"strings\"\nstrings.Join([]string{\"a\", \"b\", \"c\"}, \",\")", "a,b,c"},
		{"Join_empty_sep", "import \"strings\"\nstrings.Join([]string{\"a\", \"b\"}, \"\")", "ab"},
		{"Join_single", "import \"strings\"\nstrings.Join([]string{\"only\"}, \",\")", "only"},

		{"Split_comma", "import \"strings\"\nlen(strings.Split(\"a,b,c\", \",\"))", int64(3)},
		{"Split_no_sep", "import \"strings\"\nlen(strings.Split(\"hello\", \",\"))", int64(1)},

		{"ReplaceAll_basic", "import \"strings\"\nstrings.ReplaceAll(\"aaa\", \"a\", \"b\")", "bbb"},
		{"ReplaceAll_no_match", "import \"strings\"\nstrings.ReplaceAll(\"hello\", \"z\", \"x\")", "hello"},
		{"ReplaceAll_empty_old", "import \"strings\"\nstrings.ReplaceAll(\"hi\", \"\", \"-\")", "-h-i-"},

		{"Repeat_3", "import \"strings\"\nstrings.Repeat(\"ab\", 3)", "ababab"},
		{"Repeat_0", "import \"strings\"\nstrings.Repeat(\"ab\", 0)", ""},
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

		{"Abs_negative", "import \"math\"\nmath.Abs(-3.14)", 3.14, 0},
		{"Abs_positive", "import \"math\"\nmath.Abs(3.14)", 3.14, 0},
		{"Abs_zero", "import \"math\"\nmath.Abs(0.0)", 0.0, 0},

		{"Sqrt_16", "import \"math\"\nmath.Sqrt(16.0)", 4.0, 0},
		{"Sqrt_2", "import \"math\"\nmath.Sqrt(2.0)", math.Sqrt(2.0), 1e-10},

		{"Floor_positive", "import \"math\"\nmath.Floor(3.7)", 3.0, 0},
		{"Floor_negative", "import \"math\"\nmath.Floor(-3.2)", -4.0, 0},

		{"Ceil_positive", "import \"math\"\nmath.Ceil(3.2)", 4.0, 0},
		{"Ceil_negative", "import \"math\"\nmath.Ceil(-3.7)", -3.0, 0},

		{"Trunc_positive", "import \"math\"\nmath.Trunc(3.9)", 3.0, 0},
		{"Trunc_negative", "import \"math\"\nmath.Trunc(-3.9)", -3.0, 0},

		{"Round_up", "import \"math\"\nmath.Round(3.5)", 4.0, 0},
		{"Round_down", "import \"math\"\nmath.Round(3.4)", 3.0, 0},
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
		name   string
		code   string
		expect any
	}{
		{"ContainsRune", "import \"strings\"\nstrings.ContainsRune(\"hello\", 'e')", true},
		{"EqualFold", "import \"strings\"\nstrings.EqualFold(\"Go\", \"go\")", true},
		{"Trim", "import \"strings\"\nstrings.Trim(\"  hello  \", \" \")", "hello"},
		{"IndexRune", "import \"strings\"\nstrings.IndexRune(\"hello\", 'l')", int64(2)},
		{"LastIndex", "import \"strings\"\nstrings.LastIndex(\"go gopher\", \"go\")", int64(3)},
		{"Join", "import \"strings\"\nstrings.Join([]string{\"a\", \"b\"}, \",\")", "a,b"},
		{"Split", "import \"strings\"\nlen(strings.Split(\"a,b,c\", \",\"))", int64(3)},
		{"ReplaceAll", "import \"strings\"\nstrings.ReplaceAll(\"aaa\", \"a\", \"b\")", "bbb"},
		{"Repeat", "import \"strings\"\nstrings.Repeat(\"ab\", 3)", "ababab"},
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
		{"Abs", "import \"math\"\nmath.Abs(-3.14)", 3.14},
		{"Sqrt", "import \"math\"\nmath.Sqrt(16.0)", 4.0},
		{"Floor", "import \"math\"\nmath.Floor(3.7)", 3.0},
		{"Ceil", "import \"math\"\nmath.Ceil(3.2)", 4.0},
		{"Trunc", "import \"math\"\nmath.Trunc(3.9)", 3.0},
		{"Round", "import \"math\"\nmath.Round(3.5)", 4.0},
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
		name   string
		code   string
		expect any
	}{
		{"FormatBool", "import \"strconv\"\nstrconv.FormatBool(true)", "true"},
		{"FormatInt", "import \"strconv\"\nstrconv.FormatInt(42, 10)", "42"},
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
		name   string
		code   string
		expect any
	}{
		{"FormatBool_true", "import \"strconv\"\nstrconv.FormatBool(true)", "true"},
		{"FormatBool_false", "import \"strconv\"\nstrconv.FormatBool(false)", "false"},
		{"FormatInt_base10", "import \"strconv\"\nstrconv.FormatInt(42, 10)", "42"},
		{"FormatInt_base16", "import \"strconv\"\nstrconv.FormatInt(255, 16)", "ff"},
		{"FormatInt_base2", "import \"strconv\"\nstrconv.FormatInt(10, 2)", "1010"},
		{"FormatInt_negative", "import \"strconv\"\nstrconv.FormatInt(-42, 10)", "-42"},
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
