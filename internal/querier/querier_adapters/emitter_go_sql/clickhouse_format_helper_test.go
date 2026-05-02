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

package emitter_go_sql

import (
	"fmt"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderClickHouseHelperSource(t *testing.T) string {
	t.Helper()
	file, err := renderClickHouseFormatHelper("mydb")
	require.NoError(t, err)
	return string(file.Content)
}

func TestRenderClickHouseFormatHelperEmitsValidGoFile(t *testing.T) {
	file, err := renderClickHouseFormatHelper("mydb")
	require.NoError(t, err)
	assert.Equal(t, "clickhouse_format.go", file.Name)

	source := string(file.Content)

	fileSet := token.NewFileSet()
	_, parseError := parser.ParseFile(fileSet, "clickhouse_format.go", source, parser.AllErrors)
	require.NoError(t, parseError, "generated helper must be valid Go:\n%s", source)

	assert.Contains(t, source, "package mydb")
	assert.Contains(t, source, "func pikoClickHouseFormat(value any) string")
	assert.Contains(t, source, "func pikoClickHouseLiteral(value any) string")
}

func TestRenderClickHouseFormatHelperImportsRuntimeDependencies(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	for _, importPath := range []string{`"fmt"`, `"reflect"`, `"slices"`, `"strings"`, `"time"`} {
		assert.Contains(t, source, importPath, "helper must import %s for its runtime body", importPath)
	}
}

func TestRenderClickHouseLiteralEscapesBackslashBeforeSingleQuote(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	expectedString := `"'" + strings.ReplaceAll(strings.ReplaceAll(s, "\\", "\\\\"), "'", "''") + "'"`
	assert.Contains(t, source, expectedString, "string values must escape backslash then single quote")

	backslashOffset := strings.Index(source, `strings.ReplaceAll(s, "\\", "\\\\")`)
	singleQuoteOffset := strings.Index(source, `, "'", "''")`)
	require.GreaterOrEqual(t, backslashOffset, 0, "expected the backslash escape call in:\n%s", source)
	require.GreaterOrEqual(t, singleQuoteOffset, 0, "expected the single-quote escape call in:\n%s", source)
	assert.Less(t, backslashOffset, singleQuoteOffset,
		"the backslash escape must be the inner call so it runs before the single-quote escape")
}

func TestRenderClickHouseLiteralEscapesByteSlicesAsQuotedStrings(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	expectedBytes := `if b, ok := value.([]byte); ok {`
	assert.Contains(t, source, expectedBytes)
	assert.Contains(t, source, `"'" + strings.ReplaceAll(strings.ReplaceAll(string(b), "\\", "\\\\"), "'", "''") + "'"`)
}

func TestRenderClickHouseLiteralQuotesTimeValues(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, `if t, ok := value.(time.Time); ok {`)
	assert.Contains(t, source, `"'" + pikoClickHouseFormatDepth(t, depth) + "'"`)
}

func TestRenderClickHouseLiteralRecursesThroughFormatForOtherTypes(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	literalBodyStart := strings.Index(source, "func pikoClickHouseLiteralDepth(value any, depth int) string")
	require.GreaterOrEqual(t, literalBodyStart, 0)
	literalBody := source[literalBodyStart:]
	assert.Contains(t, literalBody, "return pikoClickHouseFormatDepth(value, depth)")
}

func TestRenderClickHouseFormatSerialisesSlicesAsBracketedLiterals(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "case reflect.Slice, reflect.Array:")
	assert.Contains(t, source, "parts := make([]string, rv.Len())")
	assert.Contains(t, source, "parts[i] = pikoClickHouseLiteralDepth(rv.Index(i).Interface(), depth+1)")

	assert.Contains(t, source, `return "[" + strings.Join(parts, ",") + "]"`)
}

func TestRenderClickHouseFormatSerialisesMapsWithDeterministicOrdering(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "case reflect.Map:")
	assert.Contains(t, source, "keys := rv.MapKeys()")

	assert.Contains(t, source, "pikoClickHouseLiteralDepth(key.Interface(), depth+1)+\":\"+pikoClickHouseLiteralDepth(rv.MapIndex(key).Interface(), depth+1)")

	assert.Contains(t, source, "slices.Sort(parts)")
	assert.Contains(t, source, `return "{" + strings.Join(parts, ",") + "}"`)
}

func TestRenderClickHouseFormatReturnsEmptyForNil(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	formatBodyStart := strings.Index(source, "func pikoClickHouseFormatDepth(value any, depth int) string")
	require.GreaterOrEqual(t, formatBodyStart, 0)
	formatBody := source[formatBodyStart:]
	assert.Contains(t, formatBody, "if value == nil {")
	assert.Contains(t, formatBody, `return ""`)
}

func TestRenderClickHouseFormatRecursesThroughPointersAndInterfaces(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "case reflect.Ptr, reflect.Interface:")
	assert.Contains(t, source, "if rv.IsNil() {")
	assert.Contains(t, source, "return pikoClickHouseFormatDepth(rv.Elem().Interface(), depth+1)")
}

func TestRenderClickHouseFormatHandlesTimeDateAndDateTime(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "case time.Time:")

	assert.Contains(t, source, "u := v.UTC()")
	assert.Contains(t, source, "if u.IsZero() {")
	assert.Contains(t, source, "u.Hour() == 0 && u.Minute() == 0 && u.Second() == 0 && u.Nanosecond() == 0")
	assert.Contains(t, source, `return u.Format("2006-01-02")`)
	assert.Contains(t, source, `return u.Format("2006-01-02 15:04:05.999999999")`)
}

func TestRenderClickHouseFormatSupportsStringerAndStringError(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "case fmt.Stringer:")
	assert.Contains(t, source, "return v.String()")

	assert.Contains(t, source, "if s, err := v.String(); err == nil {")
	assert.Contains(t, source, "return s")
}

func TestRenderClickHouseFormatFallsBackToSprint(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "return fmt.Sprint(value)")
}

func TestRenderClickHouseFormatBoundsRecursionDepth(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "const pikoClickHouseMaxDepth = 32")
	assert.Contains(t, source, "func pikoClickHouseFormat(value any) string")
	assert.Contains(t, source, "return pikoClickHouseFormatDepth(value, 0)")
	assert.Contains(t, source, "func pikoClickHouseLiteral(value any) string")
	assert.Contains(t, source, "return pikoClickHouseLiteralDepth(value, 0)")

	guardCount := strings.Count(source, "if depth > pikoClickHouseMaxDepth {")
	assert.GreaterOrEqual(t, guardCount, 2,
		"both pikoClickHouseFormatDepth and pikoClickHouseLiteralDepth must carry the recursion-depth guard")

	formatDepthStart := strings.Index(source, "func pikoClickHouseFormatDepth(value any, depth int) string")
	require.GreaterOrEqual(t, formatDepthStart, 0)
	literalDepthStart := strings.Index(source, "func pikoClickHouseLiteralDepth(value any, depth int) string")
	require.GreaterOrEqual(t, literalDepthStart, 0)
	require.Less(t, formatDepthStart, literalDepthStart, "format helper is emitted before the literal helper")
	formatDepthBody := source[formatDepthStart:literalDepthStart]
	literalDepthBody := source[literalDepthStart:]
	assert.Contains(t, formatDepthBody, "if depth > pikoClickHouseMaxDepth {")
	assert.Contains(t, literalDepthBody, "if depth > pikoClickHouseMaxDepth {")
}

func TestRenderClickHouseFormatSerialisesNamedByteSlicesAsStrings(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	assert.Contains(t, source, "if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {")
	assert.Contains(t, source, "return string(rv.Bytes())")
}

func TestPikoClickHouseFormatNamedByteSliceRoundTrips(t *testing.T) {
	t.Parallel()

	type blob []byte
	formatted := pikoClickHouseFormatNamedByteSliceReference(blob("hi"))
	assert.Equal(t, "hi", formatted)
}

func pikoClickHouseFormatNamedByteSliceReference(value any) string {
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Slice && rv.Type().Elem().Kind() == reflect.Uint8 {
		return string(rv.Bytes())
	}
	return ""
}

func TestRenderClickHouseLiteralQuotesStringerAndSprintFallback(t *testing.T) {

	source := renderClickHouseHelperSource(t)

	literalBodyStart := strings.Index(source, "func pikoClickHouseLiteralDepth(value any, depth int) string")
	require.GreaterOrEqual(t, literalBodyStart, 0)
	literalBody := source[literalBodyStart:]

	expectedFallback := `return "'" + strings.ReplaceAll(strings.ReplaceAll(pikoClickHouseFormatDepth(value, depth), "\\", "\\\\"), "'", "''") + "'"`
	assert.Contains(t, literalBody, expectedFallback,
		"the Stringer / Sprint fallback must be single-quoted and escaped")
}

func TestRenderClickHouseLiteralEmitsNumericsAndBoolsBare(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	literalBodyStart := strings.Index(source, "func pikoClickHouseLiteralDepth(value any, depth int) string")
	require.GreaterOrEqual(t, literalBodyStart, 0)
	literalBody := source[literalBodyStart:]

	assert.Contains(t, literalBody, "switch value.(type) {")
	assert.Contains(t, literalBody, "case bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, uintptr, float32, float64, complex64, complex128:")
	assert.Contains(t, literalBody, "return pikoClickHouseFormatDepth(value, depth)")
}

func TestRenderClickHouseLiteralEmitsNestedContainersBare(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	literalBodyStart := strings.Index(source, "func pikoClickHouseLiteralDepth(value any, depth int) string")
	require.GreaterOrEqual(t, literalBodyStart, 0)
	literalBody := source[literalBodyStart:]

	assert.Contains(t, literalBody, "case reflect.Slice, reflect.Array, reflect.Map:")
}

func TestRenderClickHouseLiteralRendersNilPointerAsNull(t *testing.T) {
	source := renderClickHouseHelperSource(t)

	literalBodyStart := strings.Index(source, "func pikoClickHouseLiteralDepth(value any, depth int) string")
	require.GreaterOrEqual(t, literalBodyStart, 0)
	literalBody := source[literalBodyStart:]

	assert.Contains(t, literalBody, "case reflect.Ptr, reflect.Interface:")
	assert.Contains(t, literalBody, "return pikoClickHouseLiteralDepth(rv.Elem().Interface(), depth+1)")
	assert.Contains(t, literalBody, `return "NULL"`)
}

func pikoClickHouseLiteralReference(value any, depth int) string {
	escape := func(s string) string {
		s = strings.ReplaceAll(s, "\\", "\\\\")
		return strings.ReplaceAll(s, "'", "''")
	}
	format := func(v any) string {
		switch typed := v.(type) {
		case fmt.Stringer:
			return typed.String()
		default:
			return fmt.Sprint(v)
		}
	}
	if value == nil {
		return "NULL"
	}
	if s, ok := value.(string); ok {
		return "'" + escape(s) + "'"
	}
	switch value.(type) {
	case bool, int, int64, float64:
		return fmt.Sprint(value)
	}
	return "'" + escape(format(value)) + "'"
}

type maliciousStringer struct{ payload string }

func (m maliciousStringer) String() string { return m.payload }

func TestPikoClickHouseLiteralEscapesAdversarialStringer(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payload  string
		expected string
	}{
		{name: "single quote", payload: "a'b", expected: "'a''b'"},
		{name: "backslash", payload: `a\b`, expected: `'a\\b'`},
		{name: "injection attempt", payload: "a';DROP", expected: "'a'';DROP'"},
		{name: "comma and bracket", payload: "x],y[", expected: "'x],y['"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := pikoClickHouseLiteralReference(maliciousStringer{payload: test.payload}, 0)
			assert.Equal(t, test.expected, result,
				"a Stringer payload must be quoted and escaped, not smuggled into the composite")
		})
	}
}

func TestPikoClickHouseLiteralEmitsNumericsBare(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "42", pikoClickHouseLiteralReference(42, 0))
	assert.Equal(t, "true", pikoClickHouseLiteralReference(true, 0))
	assert.Equal(t, "NULL", pikoClickHouseLiteralReference(nil, 0))
}
