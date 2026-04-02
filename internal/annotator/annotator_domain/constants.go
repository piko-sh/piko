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

package annotator_domain

const (
	// logKeyDepth is the logging key for visitor nesting depth.
	logKeyDepth = "depth"

	// logKeyName is the log attribute key for identifier names.
	logKeyName = "name"

	// logKeyBase is the logger key for the base expression name.
	logKeyBase = "base"

	// logKeyProp is the log attribute key for property names.
	logKeyProp = "prop"

	// logKeyExpr is the log attribute key for expression strings.
	logKeyExpr = "expr"

	// logKeyMessage is the key for the message field in structured log entries.
	logKeyMessage = "message"

	// logKeyResolvedType is the logging key for the resolved type name.
	logKeyResolvedType = "resolvedType"

	// logKeyCallee is the log attribute key for callee expression strings.
	logKeyCallee = "callee"

	// logKeyTag is the log field key for HTML tag names.
	logKeyTag = "tag"

	// logKeyPath is the log key for file paths.
	logKeyPath = "path"

	// logKeyDiagnostic is the log key for tracing diagnostic messages.
	logKeyDiagnostic = "Diagnostic"

	// logKeySrc is the log field key for an image or asset source path.
	logKeySrc = "src"

	// logKeyProfile is the log key for the asset profile name.
	logKeyProfile = "profile"

	// logKeyDensities is the log key for image density values.
	logKeyDensities = "densities"

	// transformKeyDensity is the key for storing density values in the
	// TransformationParams map for image variants.
	transformKeyDensity = "_density"

	// transformKeyResponsive is the internal marker for responsive image processing.
	transformKeyResponsive = "_responsive"

	// transformKeyDensities is the parameter key for image density values.
	transformKeyDensities = "densities"

	// attributeSrc is the name of the HTML src attribute.
	attributeSrc = "src"

	// pkgStrconv is the package name for the strconv standard library.
	pkgStrconv = "strconv"

	// baseDecimal is the numeric base for converting integers to strings.
	baseDecimal = 10

	// bitSize32 is the bit size for 32-bit floating-point numbers.
	bitSize32 = 32

	// bitSize64 is the bit size for 64-bit floating-point values.
	bitSize64 = 64

	// typeNameString is the type name for string values.
	typeNameString = "string"

	// typeNameBool is the Go type name for boolean values.
	typeNameBool = "bool"

	// typeNameInt is the type name for int.
	typeNameInt = "int"

	// typeNameInt8 is the Go type name for int8.
	typeNameInt8 = "int8"

	// typeNameInt16 is the type name for int16.
	typeNameInt16 = "int16"

	// typeNameInt32 is the type name for a 32-bit signed integer.
	typeNameInt32 = "int32"

	// typeNameInt64 is the type name for int64.
	typeNameInt64 = "int64"

	// typeNameUint is the type name for uint.
	typeNameUint = "uint"

	// typeNameUint8 is the type name for uint8.
	typeNameUint8 = "uint8"

	// typeNameUint16 is the type name for uint16.
	typeNameUint16 = "uint16"

	// typeNameUint32 is the type name for uint32.
	typeNameUint32 = "uint32"

	// typeNameUint64 is the type name for uint64.
	typeNameUint64 = "uint64"

	// typeNameFloat32 is the type name for the float32 type.
	typeNameFloat32 = "float32"

	// typeNameFloat64 is the type name for float64 values.
	typeNameFloat64 = "float64"

	// typeNameByte is the type name for byte.
	typeNameByte = "byte"

	// typeNameRune is the type name for rune.
	typeNameRune = "rune"

	// packageAliasMaths is the import alias for the maths package.
	packageAliasMaths = "maths"

	// typeNameMathsDecimal is the qualified type name for maths.Decimal.
	typeNameMathsDecimal = "maths.Decimal"

	// typeNameMathsBigInt is the full type name for math/big.Int.
	typeNameMathsBigInt = "maths.BigInt"
)
