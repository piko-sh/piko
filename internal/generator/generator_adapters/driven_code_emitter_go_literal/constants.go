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

package driven_code_emitter_go_literal

const (
	// HTMLAttributeTypeFmt is the format string for building HTMLAttribute type
	// names in generated Go AST code.
	HTMLAttributeTypeFmt = "%s.HTMLAttribute"

	// FieldNameName is the struct field name for the Name field of an HTML
	// attribute.
	FieldNameName = "Name"

	// FieldNameValue is the struct field name for the Value field when building
	// HTMLAttribute composite literals in the generated Go AST.
	FieldNameValue = "Value"

	// BlankIdentifier is the underscore character used to discard values in Go
	// assignments.
	BlankIdentifier = "_"

	// DiagnosticsVarName is the variable name used to store diagnostics during
	// code generation.
	DiagnosticsVarName = "diagnostics"

	// StringTypeName is the Go type name for string literals in generated code.
	StringTypeName = "string"

	// MakeFuncName is the name of the built-in make function in Go.
	MakeFuncName = "make"

	// Int64TypeName is the Go type name for 64-bit signed integers.
	Int64TypeName = "int64"

	// Int32TypeName is the name of the Go int32 type.
	Int32TypeName = "int32"

	// Int16TypeName is the type name for the Go int16 type.
	Int16TypeName = "int16"

	// Int8TypeName is the type name for the int8 type.
	Int8TypeName = "int8"

	// UintTypeName is the Go type name for uint.
	UintTypeName = "uint"

	// Uint64TypeName is the Go type name for unsigned 64-bit integers.
	Uint64TypeName = "uint64"

	// Uint32TypeName is the Go type name for uint32.
	Uint32TypeName = "uint32"

	// Uint16TypeName is the Go type name for uint16.
	Uint16TypeName = "uint16"

	// Uint8TypeName is the Go type name for uint8.
	Uint8TypeName = "uint8"

	// ByteTypeName is the Go type name for byte, which is an alias for uint8.
	ByteTypeName = "byte"

	// Float64TypeName is the Go type name for float64.
	Float64TypeName = "float64"

	// Float32TypeName is the Go type name for 32-bit floating-point numbers.
	Float32TypeName = "float32"

	// BoolTypeName is the Go type name for boolean values.
	BoolTypeName = "bool"

	// IntTypeName is the name of the int built-in type in Go.
	IntTypeName = "int"

	// NoPropsTypeName is the type name for components that do not accept props.
	NoPropsTypeName = "NoProps"

	// EmptyInterfaceTypeName is the Go type name for the empty interface.
	EmptyInterfaceTypeName = "interface{}"

	// RequestVarName is the variable name for the HTTP request parameter in
	// generated function signatures.
	RequestVarName = "r"

	// OkVarName is the variable name for the success flag in type assertions.
	OkVarName = "ok"

	// GoKeywordNil is the Go nil keyword string used for cached identifier lookups.
	GoKeywordNil = "nil"

	// PageMetaVarName is the variable name for page metadata in generated code.
	PageMetaVarName = "pageMeta"

	// MainCachePolicyVarName is the variable name for the main cache policy.
	MainCachePolicyVarName = "mainCachePolicy"

	// NumericRankFloat64 is the highest numeric rank, used for float64 types.
	NumericRankFloat64 = 10

	// NumericRankFloat32 is the numeric rank for float32 types in type promotion.
	NumericRankFloat32 = 9

	// NumericRankInt64 is the numeric rank for int64 and uint64 types.
	NumericRankInt64 = 6

	// NumericRankInt is the numeric rank for int, uint, and uintptr types.
	NumericRankInt = 5

	// NumericRankInt32 is the numeric rank for int32, uint32, and rune types.
	NumericRankInt32 = 4

	// NumericRankInt16 is the numeric rank for int16 and uint16 types.
	NumericRankInt16 = 3

	// NumericRankInt8 is the numeric rank for int8, uint8, and byte types.
	NumericRankInt8 = 2

	// NumericRankUnknown is the rank for types that are nil or not numeric.
	NumericRankUnknown = 0

	// IntValueZero represents the integer value zero.
	IntValueZero = 0

	// IntValueOne is the integer value one, used when converting boolean to number.
	IntValueOne = 1

	// ChildCountThreshold is the number of child elements below which static HTML
	// output is used instead of dynamic rendering.
	ChildCountThreshold = 3

	// AttributeCapacityBuffer is the extra space added to attribute slice capacity
	// estimates. It provides room for attributes that directives may add, such as
	// class, style, and event handlers.
	AttributeCapacityBuffer = 8

	// StatementSliceCapacity is the default capacity for statement slices created
	// during template node emission.
	StatementSliceCapacity = 16

	// PartialAttributeCapacity is the number of attributes added for public
	// partial invocations: "partial", "partial_name", and "partial_src".
	PartialAttributeCapacity = 3

	// PartialPropsAttributeCapacity is the extra capacity for the
	// "partial_props" attribute, added only when a public partial has
	// query-bound primitive props.
	PartialPropsAttributeCapacity = 1

	// SingleDirectiveAttrCount is the count for directive-based attributes
	// (DirClass, DirStyle, Key) that each contribute one attribute.
	SingleDirectiveAttrCount = 1

	// PikoImgSrcsetAttrCount is the number of attributes added for srcset
	// on piko:img elements.
	PikoImgSrcsetAttrCount = 1

	// PikoImgSizesAttrCount is the number of attributes added for the sizes
	// attribute on piko:img elements. This is only added when width descriptors
	// are present and sizes is specified.
	PikoImgSizesAttrCount = 1

	// MiscDirectiveCapacity is the starting slice capacity for miscellaneous
	// directive statements during code generation.
	MiscDirectiveCapacity = 4

	// defaultDiagnosticCapacity is the default pre-allocation capacity for
	// diagnostic slices during code emission.
	defaultDiagnosticCapacity = 4

	// largeDiagnosticCapacity is a larger pre-allocation capacity for diagnostic
	// slices when more diagnostics are expected.
	largeDiagnosticCapacity = 8

	// defaultEmissionStatementCapacity is the default pre-allocation capacity
	// for statement slices during code emission.
	defaultEmissionStatementCapacity = 8

	// smallEmissionStatementCapacity is a smaller pre-allocation capacity for
	// statement slices when fewer statements are expected.
	smallEmissionStatementCapacity = 4

	// directWriterStatementCapacity is the pre-allocation capacity for
	// statement slices used in direct writer emission contexts.
	directWriterStatementCapacity = 6

	// TokenKindInt is the token kind for integer literals, matching go/token.INT.
	TokenKindInt = 5

	// TokenKindFloat is the token kind for float literals (go/token.FLOAT = 6).
	TokenKindFloat = 6

	// TokenKindChar is the token kind for rune/character literals (go/token.CHAR = 7).
	TokenKindChar = 7

	// timePackagePath is the import path for the standard time package.
	timePackagePath = "time"

	// timeParseFunc is the function name for time.Parse.
	timeParseFunc = "Parse"

	// timeParseDuration is the function name for time.ParseDuration.
	timeParseDuration = "ParseDuration"

	// timeDateFormat is the Go reference date layout for date-only parsing.
	timeDateFormat = "2006-01-02"

	// timeTimeFormat is the Go reference time layout for time-only parsing.
	timeTimeFormat = "15:04:05"

	// varNameErr is the variable name for error values in generated code.
	varNameErr = "err"

	// varNameData is the variable name for data values in generated code.
	varNameData = "data"

	// varNameV is the variable name for temporary parsed values in generated code.
	varNameV = "v"

	// varNameQP is the variable name for query parameter values in generated code.
	varNameQP = "qp"

	// pkgStrconv is the strconv package name used when building AST nodes for
	// number parsing functions.
	pkgStrconv = "strconv"

	// pkgSafeconv is the import alias for the Piko safeconv package used for
	// clamped integer conversions in query parameter fallbacks.
	pkgSafeconv = "safeconv"

	// pkgJSON is the import name for the standard library JSON package.
	pkgJSON = "json"

	// pkgMaths is the import alias for the Piko maths package.
	pkgMaths = "maths"

	// mathsDecimalTypeName is the full type name for maths.Decimal.
	mathsDecimalTypeName = "maths.Decimal"

	// mathsBigIntTypeName is the type name for maths.BigInt.
	mathsBigIntTypeName = "maths.BigInt"

	// mathsPackagePath is the import path for the Piko maths package.
	mathsPackagePath = "piko.sh/piko/wdk/maths"

	// strconvParseInt is the function name for strconv.ParseInt.
	strconvParseInt = "ParseInt"

	// strconvParseFloat is the name of the strconv.ParseFloat function.
	strconvParseFloat = "ParseFloat"

	// strconvParseBool is the name of the strconv.ParseBool function.
	strconvParseBool = "ParseBool"

	// strconvItoa is the name of the strconv.Itoa function.
	strconvItoa = "Itoa"

	// strconvFormatInt is the name of the strconv.FormatInt function.
	strconvFormatInt = "FormatInt"

	// strconvFormatUint is the name of the strconv.FormatUint function.
	strconvFormatUint = "FormatUint"

	// strconvFormatFloat is the function name for strconv.FormatFloat.
	strconvFormatFloat = "FormatFloat"

	// strconvFormatBool is the name of the strconv.FormatBool function.
	strconvFormatBool = "FormatBool"

	// mathsMustString is the MustString method name used on maths types.
	mathsMustString = "MustString"

	// mathsMustInt64 is the method name for MustInt64 on maths types.
	mathsMustInt64 = "MustInt64"

	// mathsMustFloat64 is the name of the MustFloat64 method on maths types.
	mathsMustFloat64 = "MustFloat64"

	// mathsToDecimal is the name of the ToDecimal method on maths.BigInt.
	mathsToDecimal = "ToDecimal"

	// mathsToBigInt is the name of the ToBigInt method on maths.Decimal.
	mathsToBigInt = "ToBigInt"

	// mathsNewDecimalFromInt is the constructor name for creating a Decimal from
	// an int64.
	mathsNewDecimalFromInt = "NewDecimalFromInt"

	// mathsNewDecimalFromFloat is the constructor name for creating a Decimal from
	// a float64.
	mathsNewDecimalFromFloat = "NewDecimalFromFloat"

	// mathsNewDecimalFromString is the constructor name for creating a Decimal
	// from a string.
	mathsNewDecimalFromString = "NewDecimalFromString"

	// mathsNewBigIntFromInt is the constructor name for creating a BigInt from
	// an int64.
	mathsNewBigIntFromInt = "NewBigIntFromInt"

	// mathsNewBigIntFromString is the name of the constructor that creates a BigInt
	// from a string value.
	mathsNewBigIntFromString = "NewBigIntFromString"

	// helperValueToString names the runtime helper function that converts
	// values to strings.
	helperValueToString = "ValueToString"

	// numericBaseDecimal is the base ten value for decimal number parsing.
	numericBaseDecimal = 10

	// bitSize16 is the bit size for parsing 16-bit integers.
	bitSize16 = 16

	// bitSize32 is the bit size for 32-bit numbers used when parsing strings.
	bitSize32 = 32

	// bitSize64 is the bit size for parsing 64-bit integers and floats.
	bitSize64 = 64

	// actionArgsCount3 is the number of arguments for EncodeActionPayloadBytes3.
	actionArgsCount3 = 3

	// actionArgsCount4 is the argument count for EncodeActionPayloadBytes4.
	actionArgsCount4 = 4

	// stylePartsCount3 is the number of parts for BuildStyleStringBytes3.
	stylePartsCount3 = 3

	// stylePartsCount4 is the number of parts for BuildStyleStringBytes4.
	stylePartsCount4 = 4

	// classPartsCount4 is the number of parts used by BuildClassBytes4.
	classPartsCount4 = 4

	// classPartsCount6 is the part count for BuildClassBytes6.
	classPartsCount6 = 6

	// classPartsCount8 is the part count for BuildClassBytes8.
	classPartsCount8 = 8

	// actionArgTypeName is the type name for pikoruntime.ActionArgument.
	actionArgTypeName = "ActionArgument"

	// identRenderErr is the variable name for render errors in generated code.
	identRenderErr = "renderErr"

	// identInternalMetadata is the struct name for internal metadata
	// in generated code.
	identInternalMetadata = "InternalMetadata"

	// identCtx is the variable name for the context parameter in generated code.
	identCtx = "ctx"
)
