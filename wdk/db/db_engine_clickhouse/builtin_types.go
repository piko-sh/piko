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

package db_engine_clickhouse

import (
	"errors"
	"maps"
	"strconv"
	"strings"
	"unicode/utf8"

	"piko.sh/piko/internal/querier/querier_dto"
)

const (
	// decimalRadix is the radix used by parseIntegerModifier.
	decimalRadix = 10

	// decimal32ImpliedPrecision is the precision implied by Decimal32(S).
	decimal32ImpliedPrecision = 9

	// decimal64ImpliedPrecision is the precision implied by Decimal64(S).
	decimal64ImpliedPrecision = 18

	// decimal128ImpliedPrecision is the precision implied by Decimal128(S).
	decimal128ImpliedPrecision = 38

	// decimal256ImpliedPrecision is the precision implied by Decimal256(S).
	decimal256ImpliedPrecision = 76

	// maxTypeParseDepth caps recursion through the type-name parser when descending into
	// nested wrappers (Array(Array(Tuple(...))), Nullable(LowCardinality(Array(...))),
	// etc.). The type-name parser is invoked outside the configurable maxParseDepth path
	// (type catalogue lookups, NormaliseTypeName) so it keeps its own fixed cap; it is
	// already safe and need not be configurable.
	maxTypeParseDepth = 64

	// maxTypeModifierValue bounds a type-modifier integer (FixedString length, Decimal
	// precision/scale, DateTime64 scale). It is generous enough for any realistic type while
	// rejecting absurd inputs and preventing int overflow during accumulation.
	maxTypeModifierValue = 1 << 30
)

var (
	// errTypeDepthExceeded is the sentinel returned when the type-name parser descends past
	// maxTypeParseDepth nested wrappers.
	errTypeDepthExceeded = errors.New("clickhouse: type-name recursion depth exceeded")

	// clickhousePrimitiveTypes maps ClickHouse type names to their structured SQLType.
	// Wrapper types (Nullable, Array, LowCardinality, Tuple, Map, Nested, Enum) are NOT in
	// this table; they are parsed by parseClickHouseType and built compositionally from the
	// inner type's catalogue entry.
	//
	// Keys are canonical (case-preserved) ClickHouse names; lookups go through
	// canonicaliseTypeName so case-insensitive matches resolve to the canonical form.
	// Catalogue entries are added by buildBuiltinTypeMap so callers can rebuild with extras
	// layered in.
	clickhousePrimitiveTypes = map[string]querier_dto.SQLType{}
)

func init() {
	registerPrimitiveNumericTypes()
	registerPrimitiveScalarTypes()
	registerPrimitiveSpecialisedTypes()
}

// registerPrimitive lower-cases the type name, stamps it onto the SQLType, and inserts it
// into the package-level primitives map. The helper is used by the registerPrimitive*
// group functions so the boilerplate stays out of the registration list itself.
//
// Takes name (string), the canonical ClickHouse type spelling.
// Takes sqlType (querier_dto.SQLType), the structured type without engine name.
func registerPrimitive(name string, sqlType querier_dto.SQLType) {
	sqlType.EngineName = name
	clickhousePrimitiveTypes[strings.ToLower(name)] = sqlType
}

// registerPrimitiveNumericTypes registers the integer and floating point primitive types.
//
// Unsigned integers (UInt8, UInt16, UInt32, UInt64, UInt128, UInt256), signed integers
// (Int8, Int16, Int32, Int64, Int128, Int256), and floats (Float32, Float64, BFloat16)
// all map to integer or float categories; the engine's type mapping table picks the
// Go-side representation.
func registerPrimitiveNumericTypes() {
	registerPrimitive("UInt8", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("UInt16", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("UInt32", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("UInt64", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("UInt128", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("UInt256", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})

	registerPrimitive("Int8", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("Int16", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("Int32", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("Int64", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("Int128", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})
	registerPrimitive("Int256", querier_dto.SQLType{Category: querier_dto.TypeCategoryInteger})

	registerPrimitive("Float32", querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat})
	registerPrimitive("Float64", querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat})
	registerPrimitive("BFloat16", querier_dto.SQLType{Category: querier_dto.TypeCategoryFloat})
}

// registerPrimitiveScalarTypes registers booleans, the canonical String type, and Date /
// DateTime variants. FixedString(N) and DateTime64(N) are not registered here; they
// require a precision modifier and are handled by parseClickHouseType.
func registerPrimitiveScalarTypes() {
	registerPrimitive("Bool", querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean})
	registerPrimitive("Boolean", querier_dto.SQLType{Category: querier_dto.TypeCategoryBoolean})

	registerPrimitive("String", querier_dto.SQLType{Category: querier_dto.TypeCategoryText})

	registerPrimitive("Date", querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal})
	registerPrimitive("Date32", querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal})
	registerPrimitive("DateTime", querier_dto.SQLType{Category: querier_dto.TypeCategoryTemporal})
}

// registerPrimitiveSpecialisedTypes registers UUID, network address types (IPv4 / IPv6),
// JSON, the catch-all Dynamic / Nothing types, and the geometric shapes (Point / Ring /
// Polygon / MultiPolygon). These are domain-specific categories whose Go-side
// representation is picked by the codegen target.
func registerPrimitiveSpecialisedTypes() {
	registerPrimitive("UUID", querier_dto.SQLType{Category: querier_dto.TypeCategoryUUID})

	registerPrimitive("IPv4", querier_dto.SQLType{Category: querier_dto.TypeCategoryNetwork})
	registerPrimitive("IPv6", querier_dto.SQLType{Category: querier_dto.TypeCategoryNetwork})

	registerPrimitive("JSON", querier_dto.SQLType{Category: querier_dto.TypeCategoryJSON})

	registerPrimitive("Dynamic", querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown})
	registerPrimitive("Nothing", querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown})

	registerPrimitive("Point", querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric})
	registerPrimitive("Ring", querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric})
	registerPrimitive("Polygon", querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric})
	registerPrimitive("MultiPolygon", querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric})
	registerPrimitive("LineString", querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric})
	registerPrimitive("MultiLineString", querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric})
	registerPrimitive("Geometry", querier_dto.SQLType{Category: querier_dto.TypeCategoryGeometric})

	registerPrimitive("Identifier", querier_dto.SQLType{Category: querier_dto.TypeCategoryText})
}

// canonicaliseTypeName lowercases the input. ClickHouse is case-insensitive on type-name
// matches for built-ins, so the catalogue key is the lowercased form.
//
// Takes name (string) which is the ClickHouse type-name spelling.
//
// Returns string which is the lowercased catalogue key.
func canonicaliseTypeName(name string) string {
	return strings.ToLower(name)
}

// buildTypeCatalogue assembles the ClickHouse built-in type catalogue.
//
// Extras supplied via WithExtraTypes are merged after the built-ins so the caller can
// override the engine's defaults for niche dialects (ClickHouse Cloud or Altinity).
//
// Takes extras (map[string]querier_dto.SQLType) which holds type overrides to layer in.
//
// Returns *querier_dto.TypeCatalogue which holds the assembled catalogue.
func buildTypeCatalogue(extras map[string]querier_dto.SQLType) *querier_dto.TypeCatalogue {
	types := make(map[string]querier_dto.SQLType, len(clickhousePrimitiveTypes)+len(extras))
	maps.Copy(types, clickhousePrimitiveTypes)
	for name := range extras {
		types[strings.ToLower(name)] = extras[name]
	}
	return &querier_dto.TypeCatalogue{Types: types}
}

// typeParseResult carries the structured result of parsing a ClickHouse type-name string.
// The outer wrappers (Nullable, LowCardinality) are stripped during parsing; the result
// reports what was stripped so the caller can apply nullability / cardinality flags to
// the consumer (column, parameter, function return).
type typeParseResult struct {
	// SQLType is the inner type after wrapper stripping.
	SQLType querier_dto.SQLType

	// Nullable is true when the outermost wrapper was Nullable(T). Inner Nullable wrappers
	// (inside Array(Nullable(T)) etc.) are preserved on the inner SQLType so the consumer
	// can introspect.
	Nullable bool

	// LowCardinality is true when LowCardinality(T) appeared as an outer wrapper. Currently
	// the analyser treats it as a no-op because LowCardinality is a storage hint with no
	// Go-side impact.
	LowCardinality bool
}

// normaliseTypeName parses a ClickHouse type-name string into the catalogue's structured
// SQLType form. Honours the dialect-supplied hook first; falls back to the parser when
// the hook returns nil.
//
// The function discards the Nullable and LowCardinality outer wrappers because
// EnginePort.NormaliseTypeName has no place to put them; callers that need the wrapper
// information (the DDL parser when reading a CREATE TABLE column) call
// parseClickHouseType directly.
//
// Takes name (string) which is the ClickHouse type-name spelling.
// Takes hook (func(string, []int) *querier_dto.SQLType) which is the dialect override
// consulted before the parser.
// Takes modifiers (...int) which are the type modifiers passed to the hook.
//
// Returns querier_dto.SQLType which is the normalised type, or an Unknown type when the
// name cannot be parsed.
func normaliseTypeName(name string, hook func(string, []int) *querier_dto.SQLType, modifiers ...int) querier_dto.SQLType {
	if hook != nil {
		if result := hook(name, modifiers); result != nil {
			return *result
		}
	}
	result, err := parseClickHouseType(name)
	if err != nil {
		return querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryUnknown,
			EngineName: name,
		}
	}
	return result.SQLType
}

// parseClickHouseType parses a ClickHouse type-name string into a typeParseResult. The
// parser is recursive-descent over the input characters; ClickHouse type names form a
// context-free grammar over the alphabet of identifiers, parens, commas, single-quoted
// literals, integers, and equals signs.
//
// Takes input (string) which is the ClickHouse type-name to parse.
//
// Returns typeParseResult which holds the parsed structure with wrapper flags.
// Returns error when the type-name is malformed (unterminated parens, missing comma,
// unknown wrapper).
func parseClickHouseType(input string) (typeParseResult, error) {
	parser := &typeNameParser{input: input}
	parser.skipWhitespace()
	result, err := parser.parseType()
	if err != nil {
		return typeParseResult{}, err
	}
	parser.skipWhitespace()
	if parser.position < len(parser.input) {
		return typeParseResult{}, &typeNameParseError{
			input:    input,
			position: parser.position,
			message:  "trailing characters after type",
		}
	}
	return result, nil
}

// typeNameParser is a small recursive-descent parser over a ClickHouse type-name string.
//
// The parser is deliberately lightweight: it does not tokenise via the SQL tokeniser
// because type names are a closed sub-grammar and reusing the SQL lexer would carry far
// more machinery than the few productions need. The depth field bounds recursion through
// nested wrappers so adversarial inputs cannot blow the Go stack.
type typeNameParser struct {
	// input is the type-name string being parsed.
	input string

	// position is the cursor offset into input.
	position int

	// depth is the current recursion depth through nested wrappers.
	depth int
}

// skipWhitespace advances the cursor past any spaces, tabs, and line breaks.
func (p *typeNameParser) skipWhitespace() {
	for p.position < len(p.input) {
		character := p.input[p.position]
		if character == ' ' || character == '\t' || character == '\n' || character == '\r' {
			p.position++
			continue
		}
		break
	}
}

// peek returns the character at the current cursor, or zero on EOF.
//
// Returns byte which is the current character, or zero at end of input.
func (p *typeNameParser) peek() byte {
	if p.position >= len(p.input) {
		return 0
	}
	return p.input[p.position]
}

// consume advances the cursor and returns the consumed character.
//
// Returns byte which is the consumed character.
func (p *typeNameParser) consume() byte {
	character := p.peek()
	p.position++
	return character
}

// skipSingleQuotedString consumes a single-quoted string literal.
//
// The literal is consumed including its quotes. It honours backslash escapes so a quote,
// comma, or parenthesis inside the literal does not prematurely terminate an enclosing
// scan. The cursor must be positioned on the opening quote. On an unterminated literal it
// consumes to end of input.
func (p *typeNameParser) skipSingleQuotedString() {
	if p.peek() != '\'' {
		return
	}
	p.consume()
	for p.position < len(p.input) {
		switch p.peek() {
		case '\\':
			p.consume()
			p.consume()
		case '\'':
			p.consume()
			return
		default:
			p.consume()
		}
	}
}

// match consumes one expected character and returns true on success.
//
// Takes c (byte) which is the character expected at the cursor.
//
// Returns bool which is true when the character matched and was consumed.
func (p *typeNameParser) match(c byte) bool {
	if p.peek() != c {
		return false
	}
	p.position++
	return true
}

// parseIdentifier reads a bare identifier, since the type-name production does not need
// quoted identifiers.
//
// The scan advances by UTF-8 rune width so multi-byte codepoints inside an identifier
// body are not truncated mid-codepoint.
//
// Returns string which is the identifier text, or empty when none is at the cursor.
func (p *typeNameParser) parseIdentifier() string {
	start := p.position
	for p.position < len(p.input) {
		character, width := utf8.DecodeRuneInString(p.input[p.position:])
		if !isIdentPart(character) {
			break
		}
		p.position += width
	}
	return p.input[start:p.position]
}

// parseType is the top-level production that dispatches on the leading identifier to the
// appropriate wrapper handler or primitive lookup.
//
// The depth counter is incremented on entry and decremented on exit so nested wrappers
// (Array(Array(Tuple(...))) and similar) cannot drive the parser past maxTypeParseDepth
// recursive frames.
//
// Returns typeParseResult which holds the parsed type.
// Returns error when the type-name is malformed or recursion exceeds maxTypeParseDepth.
func (p *typeNameParser) parseType() (typeParseResult, error) {
	if p.depth >= maxTypeParseDepth {
		return typeParseResult{}, errTypeDepthExceeded
	}
	p.depth++
	defer func() { p.depth-- }()

	p.skipWhitespace()
	identifier := p.parseIdentifier()
	if identifier == "" {
		return typeParseResult{}, &typeNameParseError{
			input:    p.input,
			position: p.position,
			message:  "expected type identifier",
		}
	}
	if result, dispatched, err := p.dispatchWrapperType(identifier); dispatched {
		return result, err
	}
	return p.dispatchScalarType(identifier)
}

// dispatchWrapperType handles the wrapper-type identifiers (Nullable, LowCardinality,
// Array, Tuple, Map, Nested, Enum / Enum8 / Enum16, JSON, Dynamic, Object, Variant,
// AggregateFunction / SimpleAggregateFunction). Returns dispatched=false when the
// identifier is not a wrapper so the caller can fall through to the scalar dispatch.
//
// Splitting parseType into wrapper / scalar dispatch keeps each helper below the
// cyclomatic-complexity budget while preserving the identical dispatch order of the
// original parser.
//
// Takes identifier (string), the leading type identifier text.
//
// Returns result (typeParseResult), the parsed type when dispatched.
// Returns dispatched (bool), true when this helper handled the identifier.
// Returns err (error), any parse error from the wrapper body.
func (p *typeNameParser) dispatchWrapperType(identifier string) (result typeParseResult, dispatched bool, err error) {
	switch strings.ToLower(identifier) {
	case "nullable":
		result, err = p.parseNullableWrapper()
		return result, true, err
	case "lowcardinality":
		result, err = p.parseLowCardinalityWrapper()
		return result, true, err
	case "array":
		result, err = p.parseArrayType()
		return result, true, err
	case "tuple":
		result, err = p.parseTupleType()
		return result, true, err
	case "map":
		result, err = p.parseMapType()
		return result, true, err
	case "nested":
		result, err = p.parseNestedType()
		return result, true, err
	}
	return p.dispatchEnumOrSpecialType(identifier)
}

// dispatchEnumOrSpecialType continues the wrapper-type dispatch with the enum /
// parametrised / specialised wrapper identifiers (Enum, Enum8, Enum16, JSON, Dynamic,
// FixedString, Decimal variants, DateTime / DateTime64, Variant, Object,
// AggregateFunction, SimpleAggregateFunction). Returns dispatched=false on miss so
// dispatchWrapperType can fall through.
//
// Takes identifier (string), the leading type identifier text.
//
// Returns result, dispatched, err with the same semantics as dispatchWrapperType.
func (p *typeNameParser) dispatchEnumOrSpecialType(identifier string) (result typeParseResult, dispatched bool, err error) {
	switch strings.ToLower(identifier) {
	case "enum":

		result, err = p.parseEnumType("Enum8")
		return result, true, err
	case "enum8", "enum16":
		result, err = p.parseEnumType(identifier)
		return result, true, err
	case "json":
		result, err = p.parseJSONType()
		return result, true, err
	case "dynamic":
		result, err = p.parseDynamicType()
		return result, true, err
	case "fixedstring":
		result, err = p.parseFixedStringType()
		return result, true, err
	case "decimal", "decimal32", "decimal64", "decimal128", "decimal256":
		result, err = p.parseDecimalType(identifier)
		return result, true, err
	case "datetime":
		result, err = p.parseDateTimeType()
		return result, true, err
	case "datetime64":
		result, err = p.parseDateTime64Type()
		return result, true, err
	case "variant":
		result, err = p.parseVariantType()
		return result, true, err
	case "object":
		result, err = p.parseObjectType()
		return result, true, err
	case "aggregatefunction", "simpleaggregatefunction":
		result, err = p.parseAggregateFunctionType(identifier)
		return result, true, err
	}
	return typeParseResult{}, false, nil
}

// dispatchScalarType handles the residual scalar / primitive identifiers that did not
// match the wrapper dispatch. Currently every non-wrapper name is looked up through
// parsePrimitiveByName.
//
// Takes identifier (string), the leading type identifier text.
//
// Returns the parsed type and any error from the primitive lookup.
func (p *typeNameParser) dispatchScalarType(identifier string) (typeParseResult, error) {
	return p.parsePrimitiveByName(identifier)
}

// parseObjectType handles the legacy ClickHouse Object('json') declaration that predates
// the dedicated JSON type.
//
// The form takes a single string-literal argument naming the underlying object kind; the
// only value ClickHouse ever shipped is 'json', which is the alias of the JSON type. The
// parser treats any payload as the JSON type so other Object variants still resolve to
// the JSON surface; consumers that need to distinguish the literal argument can do so by
// inspecting the source SQL.
//
// Returns typeParseResult which holds the JSON type.
// Returns error when the Object body is not well-formed.
func (p *typeNameParser) parseObjectType() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after Object")
	}
	p.skipWhitespace()

	for p.position < len(p.input) && p.peek() != ')' {
		if p.peek() == '\'' {
			p.skipSingleQuotedString()
			continue
		}
		p.consume()
	}
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close Object")
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryJSON,
			EngineName: "JSON",
		},
	}, nil
}

// parseNullableWrapper handles Nullable(T). Sets the result's Nullable flag and returns
// the inner type unwrapped.
//
// Returns typeParseResult which holds the inner type with the Nullable flag set.
// Returns error when the Nullable body is not well-formed.
func (p *typeNameParser) parseNullableWrapper() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after Nullable")
	}
	inner, err := p.parseType()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close Nullable")
	}
	inner.Nullable = true
	return inner, nil
}

// parseLowCardinalityWrapper handles LowCardinality(T). The wrapper is a storage hint
// with no Go-side impact, so the parser strips it and propagates the inner type's flags
// through.
//
// Returns typeParseResult which holds the inner type with the LowCardinality flag set.
// Returns error when the LowCardinality body is not well-formed.
func (p *typeNameParser) parseLowCardinalityWrapper() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after LowCardinality")
	}
	inner, err := p.parseType()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close LowCardinality")
	}
	inner.LowCardinality = true
	return inner, nil
}

// parseArrayType handles Array(T). The result's SQLType is TypeCategoryArray with
// ElementType pointing at the inner.
//
// Returns typeParseResult which holds the array type with its element type.
// Returns error when the Array body is not well-formed.
func (p *typeNameParser) parseArrayType() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after Array")
	}
	inner, err := p.parseType()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close Array")
	}
	element := inner.SQLType
	element.Nullable = element.Nullable || inner.Nullable
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  "Array",
			ElementType: &element,
		},
	}, nil
}

// parseTupleType handles the anonymous Tuple(T1, T2, ...) form and the named Tuple(name1
// T1, name2 T2, ...) form.
//
// Anonymous fields are synthesised as _1, _2, and so on to keep the StructFields slice
// homogeneous so downstream consumers can index fields uniformly regardless of whether
// the tuple was declared with names.
//
// Returns typeParseResult which holds the struct type with its fields.
// Returns error when the Tuple body is not well-formed.
func (p *typeNameParser) parseTupleType() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after Tuple")
	}
	var fields []querier_dto.StructField
	fieldIndex := 1
	for {
		p.skipWhitespace()
		field, err := p.parseTupleField(fieldIndex)
		if err != nil {
			return typeParseResult{}, err
		}
		fields = append(fields, field)
		fieldIndex++
		p.skipWhitespace()
		if !p.match(',') {
			break
		}
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close Tuple")
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:     querier_dto.TypeCategoryStruct,
			EngineName:   "Tuple",
			StructFields: fields,
		},
	}, nil
}

// parseTupleField reads one field of a Tuple body.
//
// ClickHouse allows both anonymous (Tuple(String, UInt64)) and named (Tuple(s String, u
// UInt64)) fields; the parser looks two identifiers deep to disambiguate. The returned
// StructField.Name carries either the explicit name (named form) or the synthesised _N
// placeholder (anonymous form). For example, parseTupleField(1) on "name String" returns
// a field with Name="name"; on "String" alone it returns a field with Name="_1".
//
// Takes anonymousIndex (int) which is the 1-based position used to synthesise an
// anonymous field name.
//
// Returns querier_dto.StructField which is the parsed field with its name and type.
// Returns error when the field is not well-formed.
func (p *typeNameParser) parseTupleField(anonymousIndex int) (querier_dto.StructField, error) {
	savedPosition := p.position
	first := p.parseIdentifier()
	p.skipWhitespace()

	nextRune, _ := utf8.DecodeRuneInString(p.input[p.position:])
	if first != "" && p.position < len(p.input) && isIdentStart(nextRune) {
		fieldType, err := p.parseType()
		if err != nil {
			return querier_dto.StructField{}, err
		}
		merged := fieldType.SQLType
		merged.Nullable = merged.Nullable || fieldType.Nullable
		return querier_dto.StructField{
			Name:    first,
			SQLType: merged,
		}, nil
	}

	p.position = savedPosition
	fieldType, err := p.parseType()
	if err != nil {
		return querier_dto.StructField{}, err
	}
	merged := fieldType.SQLType
	merged.Nullable = merged.Nullable || fieldType.Nullable
	return querier_dto.StructField{
		Name:    synthesiseAnonymousFieldName(anonymousIndex),
		SQLType: merged,
	}, nil
}

// synthesiseAnonymousFieldName returns the conventional name for anonymous tuple fields,
// such as _1, _2, and _3.
//
// Takes index (int) which is the 1-based field position.
//
// Returns string which is the synthesised field name.
func synthesiseAnonymousFieldName(index int) string {
	return "_" + strconv.Itoa(index)
}

// parseMapType handles Map(K, V). The K type becomes KeyType, the V type becomes
// ElementType (matching the existing TypeCategoryMap convention used by duckdb).
//
// Returns typeParseResult which holds the map type with its key and value types.
// Returns error when the Map body is not well-formed.
func (p *typeNameParser) parseMapType() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after Map")
	}
	keyType, err := p.parseType()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	if !p.match(',') {
		return typeParseResult{}, p.errorAt("expected ',' between Map key and value types")
	}
	valueType, err := p.parseType()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close Map")
	}
	key := keyType.SQLType
	key.Nullable = key.Nullable || keyType.Nullable
	value := valueType.SQLType
	value.Nullable = value.Nullable || valueType.Nullable
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryMap,
			EngineName:  "Map",
			KeyType:     &key,
			ElementType: &value,
		},
	}, nil
}

// parseNestedType handles Nested(field1 T1, field2 T2, ...). The parser desugars to
// Array(Tuple(field1 T1, field2 T2, ...)) to match the underlying ClickHouse storage
// model and let downstream consumers treat nested columns uniformly with array-of-tuple.
//
// Returns typeParseResult which holds the array-of-tuple type.
// Returns error when the Nested body is not well-formed.
func (p *typeNameParser) parseNestedType() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after Nested")
	}
	var fields []querier_dto.StructField
	for {
		p.skipWhitespace()
		fieldName := p.parseIdentifier()
		if fieldName == "" {
			return typeParseResult{}, p.errorAt("expected field name in Nested(...)")
		}
		p.skipWhitespace()
		fieldType, err := p.parseType()
		if err != nil {
			return typeParseResult{}, err
		}
		merged := fieldType.SQLType
		merged.Nullable = merged.Nullable || fieldType.Nullable
		fields = append(fields, querier_dto.StructField{
			Name:    fieldName,
			SQLType: merged,
		})
		p.skipWhitespace()
		if !p.match(',') {
			break
		}
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close Nested")
	}
	tuple := querier_dto.SQLType{
		Category:     querier_dto.TypeCategoryStruct,
		EngineName:   "Tuple",
		StructFields: fields,
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryArray,
			EngineName:  "Array",
			ElementType: &tuple,
		},
	}, nil
}

// parseEnumType handles Enum8('a' = 1, 'b' = 2, ...) and the Enum16 variant. The numeric
// tag is parsed but discarded because the framework treats Enum8 and Enum16 as
// user-defined enums and only the value list matters for Go-side codegen.
//
// Takes name (string) which is the original enum type name (Enum8 or Enum16).
//
// Returns typeParseResult which holds the parsed Enum type with its value list.
// Returns error when the enum body is not well-formed.
func (p *typeNameParser) parseEnumType(name string) (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after " + name)
	}
	var values []string
	for {
		value, parseErr := p.parseEnumEntry()
		if parseErr != nil {
			return typeParseResult{}, parseErr
		}
		values = append(values, value)
		p.skipWhitespace()
		if !p.match(',') {
			break
		}
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close " + name)
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryEnum,
			EngineName: name,
			EnumValues: values,
		},
	}, nil
}

// parseEnumEntry consumes a single 'value' [ = N ] enum entry from the input cursor.
//
// The numeric tag is parsed but discarded so the caller need only collect the string
// values.
//
// Returns string which is the enum value text without the quotes.
// Returns error when the entry is malformed, such as an unterminated quote or a missing
// leading apostrophe.
func (p *typeNameParser) parseEnumEntry() (string, error) {
	p.skipWhitespace()
	if !p.match('\'') {
		return "", p.errorAt("expected single-quoted enum value")
	}
	start := p.position
	for p.position < len(p.input) && p.input[p.position] != '\'' {
		if p.input[p.position] == '\\' && p.position+1 < len(p.input) {
			p.position += 2
			continue
		}
		p.position++
	}
	if p.position >= len(p.input) {
		return "", p.errorAt("unterminated enum value")
	}
	value := p.input[start:p.position]
	p.consume()
	p.skipWhitespace()
	if p.match('=') {
		p.skipNumericTag()
	}
	return value, nil
}

// skipNumericTag advances over the numeric tag that follows the `=` separator in an enum
// entry. The tag may be negative.
func (p *typeNameParser) skipNumericTag() {
	p.skipWhitespace()
	if p.peek() == '-' {
		p.consume()
	}
	for p.position < len(p.input) && isDigit(p.input[p.position]) {
		p.position++
	}
}

// parseFixedStringType handles FixedString(N). The N modifier becomes the Length field on
// the resulting SQLType.
//
// Returns typeParseResult which holds the text type with its length.
// Returns error when the FixedString body is not well-formed.
func (p *typeNameParser) parseFixedStringType() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after FixedString")
	}
	length, err := p.parseIntegerModifier()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close FixedString")
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryText,
			EngineName: "FixedString",
			Length:     &length,
		},
	}, nil
}

// parseDecimalType handles Decimal(P, S), Decimal32(S), Decimal64(S), Decimal128(S), and
// Decimal256(S). The Px variants hard-code precision via the type name; only S is
// parameterised.
//
// Takes name (string) which is the original Decimal type name.
//
// Returns typeParseResult which holds the decimal type with its precision and scale.
// Returns error when the Decimal body is not well-formed.
func (p *typeNameParser) parseDecimalType(name string) (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after " + name)
	}
	first, err := p.parseIntegerModifier()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	var precision *int
	var scale *int
	switch strings.ToLower(name) {
	case "decimal":
		if !p.match(',') {
			return typeParseResult{}, p.errorAt("expected ',' between Decimal precision and scale")
		}
		second, err := p.parseIntegerModifier()
		if err != nil {
			return typeParseResult{}, err
		}
		precision = &first
		scale = &second
	case "decimal32":
		precision = new(decimal32ImpliedPrecision)
		scale = &first
	case "decimal64":
		precision = new(decimal64ImpliedPrecision)
		scale = &first
	case "decimal128":
		precision = new(decimal128ImpliedPrecision)
		scale = &first
	case "decimal256":
		precision = new(decimal256ImpliedPrecision)
		scale = &first
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close " + name)
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryDecimal,
			EngineName: name,
			Precision:  precision,
			Scale:      scale,
		},
	}, nil
}

// parseDateTimeType handles DateTime (no modifier) and DateTime('TZ') (with timezone).
//
// The timezone is captured but not retained on the SQLType because the codegen target
// (time.Time) carries its own timezone awareness.
//
// Returns typeParseResult which holds the temporal type.
// Returns error when the DateTime body is not well-formed.
func (p *typeNameParser) parseDateTimeType() (typeParseResult, error) {
	result := typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryTemporal,
			EngineName: "DateTime",
		},
	}
	if p.peek() != '(' {
		return result, nil
	}
	p.consume()
	p.skipWhitespace()
	if p.match('\'') {
		for p.position < len(p.input) && p.input[p.position] != '\'' {
			p.position++
		}
		if p.position < len(p.input) {
			p.consume()
		}
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close DateTime")
	}
	return result, nil
}

// parseDateTime64Type handles DateTime64(precision) and DateTime64(precision, 'TZ').
//
// The precision becomes the Precision field; any timezone is parsed and discarded for the
// reason described on parseDateTimeType.
//
// Returns typeParseResult which holds the temporal type with its precision.
// Returns error when the DateTime64 body is not well-formed.
func (p *typeNameParser) parseDateTime64Type() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after DateTime64")
	}
	precision, err := p.parseIntegerModifier()
	if err != nil {
		return typeParseResult{}, err
	}
	p.skipWhitespace()
	if p.match(',') {
		p.skipWhitespace()
		if p.match('\'') {
			for p.position < len(p.input) && p.input[p.position] != '\'' {
				p.position++
			}
			if p.position < len(p.input) {
				p.consume()
			}
		}
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close DateTime64")
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryTemporal,
			EngineName: "DateTime64",
			Precision:  &precision,
		},
	}, nil
}

// parseJSONType handles the bare JSON form and the parametrised form.
//
// The parametrised JSON(max_dynamic_paths=N, path.to TypeName, SKIP path) body is
// consumed opaquely, since its contents do not affect the Go-side type, which is always
// any-like, but the parser must still detect and balance the parens so any trailing input
// can be surfaced as a trailing-junk error by the top-level driver.
//
// Returns typeParseResult which holds the JSON type.
// Returns error when the parametrised body is not well-formed.
func (p *typeNameParser) parseJSONType() (typeParseResult, error) {
	result := typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryJSON,
			EngineName: "JSON",
		},
	}
	if p.peek() != '(' {
		return result, nil
	}
	if err := p.skipBalancedParens(); err != nil {
		return typeParseResult{}, err
	}
	return result, nil
}

// parseDynamicType handles both the bare Dynamic form and the parametrised
// Dynamic(max_types=N) form. Like JSON, the body is consumed opaquely; only the wrapper's
// presence matters at the catalogue layer because Dynamic is always type-erased on the Go
// side.
//
// Returns typeParseResult which holds the Dynamic type.
// Returns error when the parametrised body is not well-formed.
func (p *typeNameParser) parseDynamicType() (typeParseResult, error) {
	result := typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryUnknown,
			EngineName: "Dynamic",
		},
	}
	if p.peek() != '(' {
		return result, nil
	}
	if err := p.skipBalancedParens(); err != nil {
		return typeParseResult{}, err
	}
	return result, nil
}

// parseVariantType handles Variant(T1, T2, ...). Modelled as a union over the member
// types so the emitter can generate a tagged-variant Go destination where wanted
// (defaults to any).
//
// Returns typeParseResult which holds the union type with its members.
// Returns error when the Variant body is not well-formed.
func (p *typeNameParser) parseVariantType() (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after Variant")
	}
	var members []querier_dto.UnionMember
	memberIndex := 1
	for {
		p.skipWhitespace()
		memberType, err := p.parseType()
		if err != nil {
			return typeParseResult{}, err
		}
		merged := memberType.SQLType
		merged.Nullable = merged.Nullable || memberType.Nullable
		members = append(members, querier_dto.UnionMember{
			Tag:     synthesiseAnonymousFieldName(memberIndex),
			SQLType: merged,
		})
		memberIndex++
		p.skipWhitespace()
		if !p.match(',') {
			break
		}
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close Variant")
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:     querier_dto.TypeCategoryUnion,
			EngineName:   "Variant",
			UnionMembers: members,
		},
	}, nil
}

// parseAggregateFunctionType handles `AggregateFunction(name, argType1, ...)` and the
// SimpleAggregateFunction variant.
//
// The aggregate's name and argument-type list are captured into a reconstructed
// EngineName ("AggregateFunction(uniq, UInt64)") so downstream codegen can distinguish
// aggregate-state columns from finalised aggregate results; the carried EngineName also
// feeds the resolver when it needs to know which aggregate produced the state. The result
// Category is TypeCategoryAggregateState so consumers can branch on the category without
// parsing the EngineName again.
//
// The inner argument type remains accessible via ElementType for the (common) case where
// only the first argument is meaningful for codegen.
//
// Takes name (string), the wrapper name (AggregateFunction or SimpleAggregateFunction).
//
// Returns the reconstructed typeParseResult; returns error when the body is malformed
// (missing comma, unterminated paren).
func (p *typeNameParser) parseAggregateFunctionType(name string) (typeParseResult, error) {
	if !p.match('(') {
		return typeParseResult{}, p.errorAt("expected '(' after " + name)
	}
	p.skipWhitespace()
	aggregateName := p.parseIdentifier()
	if aggregateName == "" {
		return typeParseResult{}, p.errorAt("expected aggregate name")
	}

	p.skipWhitespace()
	if p.peek() == '(' {
		if err := p.skipBalancedParens(); err != nil {
			return typeParseResult{}, err
		}
	}
	p.skipWhitespace()
	if !p.match(',') {
		return typeParseResult{}, p.errorAt("expected ',' between aggregate name and arg type")
	}
	innerType, err := p.parseType()
	if err != nil {
		return typeParseResult{}, err
	}
	argTypeNames := []string{strings.TrimSpace(innerType.SQLType.EngineName)}
	for {
		p.skipWhitespace()
		if !p.match(',') {
			break
		}
		extraType, extraErr := p.parseType()
		if extraErr != nil {
			return typeParseResult{}, extraErr
		}
		argTypeNames = append(argTypeNames, strings.TrimSpace(extraType.SQLType.EngineName))
	}
	p.skipWhitespace()
	if !p.match(')') {
		return typeParseResult{}, p.errorAt("expected ')' to close " + name)
	}
	innerCopy := innerType.SQLType
	innerCopy.Nullable = innerCopy.Nullable || innerType.Nullable
	reconstructedName := name + "(" + aggregateName + ", " + strings.Join(argTypeNames, ", ") + ")"
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:    querier_dto.TypeCategoryAggregateState,
			EngineName:  reconstructedName,
			ElementType: &innerCopy,
		},
	}, nil
}

// skipBalancedParens consumes a parenthesised body starting at the current cursor
// position, where the cursor must point at the opening paren.
//
// All inner characters are discarded; the cursor lands just past the matching close
// paren. parseAggregateFunctionType uses it to swallow the aggregate's call-parameter
// list, such as quantile(0.5), when present.
//
// Returns error when the parens are unbalanced or the input ends before the close paren
// is reached.
func (p *typeNameParser) skipBalancedParens() error {
	if !p.match('(') {
		return p.errorAt("expected '(' for balanced skip")
	}
	depth := 1
	for depth > 0 && p.position < len(p.input) {
		character := p.consume()
		switch character {
		case '(':
			depth++
		case ')':
			depth--
		default:
		}
	}
	if depth != 0 {
		return p.errorAt("unbalanced parens in aggregate call-parameter list")
	}
	return nil
}

// parsePrimitiveByName looks up a bare type name in the primitive catalogue.
//
// An unrecognised name yields an Unknown-category SQLType so downstream consumers can
// still proceed with a placeholder; the analyser surfaces the unknown type via a
// diagnostic later.
//
// Takes identifier (string) which is the bare type name to look up.
//
// Returns typeParseResult which holds the matched type, or an Unknown type when no match.
// Returns error which is always nil.
func (*typeNameParser) parsePrimitiveByName(identifier string) (typeParseResult, error) {
	canonical := canonicaliseTypeName(identifier)
	if sqlType, found := clickhousePrimitiveTypes[canonical]; found {
		return typeParseResult{SQLType: sqlType}, nil
	}
	return typeParseResult{
		SQLType: querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryUnknown,
			EngineName: identifier,
		},
	}, nil
}

// parseIntegerModifier consumes a non-negative decimal integer and returns its value.
//
// FixedString(N), Decimal(P, S), and DateTime64(N) use it. A leading minus sign is
// rejected because none of these modifiers may be negative, and the magnitude is bounded
// by maxTypeModifierValue so an overlong run of digits cannot overflow.
//
// Returns int which is the parsed modifier value.
// Returns error when no digits are present, the value is negative, or it exceeds
// maxTypeModifierValue.
func (p *typeNameParser) parseIntegerModifier() (int, error) {
	p.skipWhitespace()
	if p.peek() == '-' {
		return 0, p.errorAt("integer modifier must not be negative")
	}
	start := p.position
	for p.position < len(p.input) && isDigit(p.input[p.position]) {
		p.position++
	}
	if start == p.position {
		return 0, p.errorAt("expected integer modifier")
	}
	value := 0
	for index := start; index < p.position; index++ {
		digit := int(p.input[index] - '0')
		if value > (maxTypeModifierValue-digit)/decimalRadix {
			return 0, p.errorAt("integer modifier out of range")
		}
		value = value*decimalRadix + digit
	}
	return value, nil
}

// errorAt is a shorthand for constructing a typeNameParseError at the current cursor
// position.
//
// Takes message (string) which is the diagnostic text.
//
// Returns error which is the typeNameParseError at the current position.
func (p *typeNameParser) errorAt(message string) error {
	return &typeNameParseError{
		input:    p.input,
		position: p.position,
		message:  message,
	}
}

// typeNameParseError is the error returned for malformed type-name strings. Carries the
// offending position so diagnostics can highlight it.
type typeNameParseError struct {
	// input is the type-name string that failed to parse.
	input string

	// message is the diagnostic describing the failure.
	message string

	// position is the cursor offset where the failure was detected.
	position int
}

// Error returns the formatted diagnostic for the parse failure.
//
// Returns string which is the diagnostic including the message, type name, and position.
func (e *typeNameParseError) Error() string {
	return "clickhouse: " + e.message + " in type name " + strconv.Quote(e.input) + " at position " + strconv.Itoa(e.position)
}
