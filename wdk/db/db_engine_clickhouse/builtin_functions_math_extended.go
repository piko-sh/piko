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

// registerExtendedMathFunctions covers the hyperbolic, error-function, gamma, exponent
// base-2 and base-10, prime-test and conversion helpers that round out the math library.
//
// It delegates to topical helpers to keep each registration helper within the linter
// budget.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExtendedMathFunctions(b *FunctionCatalogueBuilder) {
	registerHyperbolicAndErrorFunctions(b)
	registerExponentAndConversionFunctions(b)
	registerPrimeTestFunctions(b)
}

// registerHyperbolicAndErrorFunctions covers the hyperbolic family, inverse hyperbolics,
// error functions and the gamma family.
//
// Each member takes a single Float64 and returns Float64.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerHyperbolicAndErrorFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{
		"sinh", "cosh", "tanh", "asinh", "acosh", "atanh",
		"erf", "erfc", "lgamma", "tgamma",
	} {
		b.Register(name, b.float64Type, b.float64Type)
	}
}

// registerExponentAndConversionFunctions covers the base-2 and base-10 exponentials, the
// natural log shifted by one, hypotenuse and the degree and radian conversion helpers,
// plus the integer exponent variants.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExponentAndConversionFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"exp2", "exp10", "log1p", "degrees", "radians"} {
		b.Register(name, b.float64Type, b.float64Type)
	}
	b.Register("intExp2", b.uint64Type, b.int64Type)
	b.Register("intExp10", b.uint64Type, b.int64Type)
	b.Register("hypot", b.float64Type, b.float64Type, b.float64Type)
}

// registerPrimeTestFunctions covers the deterministic primality test and the
// probabilistic Miller-Rabin variant.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerPrimeTestFunctions(b *FunctionCatalogueBuilder) {
	b.Register("isPrime", b.boolType, b.uint64Type)
	b.Register("isProbablePrime", b.boolType, b.uint64Type)
}

// registerExtendedRoundingFunctions covers the alias rounding names (ceiling, trunc) plus
// the rounding-to-bucket helpers (roundAge, roundDuration, roundDown, roundToExp2).
//
// Aliases share the canonical implementation's signature so the analyser routes them
// identically.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExtendedRoundingFunctions(b *FunctionCatalogueBuilder) {
	b.Register("ceiling", b.float64Type, b.float64Type)
	b.Register("trunc", b.float64Type, b.float64Type)
	b.Register("roundAge", b.uint64Type, b.uint64Type)
	b.Register("roundDuration", b.uint64Type, b.uint64Type)
	b.Register("roundDown", b.float64Type, b.float64Type, arrayOf(b.float64Type))
	b.Register("roundToExp2", b.uint64Type, b.uint64Type)
}

// registerExtendedBitFunctions covers the bit-slice helper.
//
// bitPositionsToArray is owned by registerCharAndBitmaskFunctions in
// builtin_functions_encoding_extended.go because it groups with the bitmask family, so it
// is not re-registered here.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExtendedBitFunctions(b *FunctionCatalogueBuilder) {
	b.Register("bitSlice", b.textType, b.textType, b.uint64Type)
	b.Register("bitSlice", b.textType, b.textType, b.uint64Type, b.uint64Type)
}
