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

// registerExtendedRandomFunctions covers string-shaped random helpers, fuzz operations
// and the negative-binomial distribution beyond the base registerRandomFunctions set.
//
// The negative-binomial helper returns the number of failures before the configured count
// of successes, so the return type is a non-negative integer modelled as UInt64.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExtendedRandomFunctions(b *FunctionCatalogueBuilder) {
	b.Register("randomString", b.textType, b.uint64Type)
	b.Register("randomStringUTF8", b.textType, b.uint64Type)
	b.Register("randomPrintableASCII", b.textType, b.uint64Type)
	b.Register("randomFixedString", b.textType, b.uint64Type)
	b.Register("fuzzBits", b.textType, b.textType, b.float64Type)
	b.Register("randNegativeBinomial", b.uint64Type, b.uint64Type, b.float64Type)
}

// registerInOperatorFunctions covers the IN and NOT IN function-call equivalents along
// with the distributed-query globalIn and globalNotIn spellings.
//
// ClickHouse accepts these as ordinary functions in addition to the keyword syntax, so "a
// IN (b, c)" may also be written as "in(a, (b, c))", and recording them here keeps the
// analyser aligned when users prefer the call form. The right-hand argument is Dynamic
// because the set expression may be an array, tuple or subquery result, which the
// catalogue cannot express cleanly without per-argument variadic support.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerInOperatorFunctions(b *FunctionCatalogueBuilder) {
	for _, name := range []string{"in", "notIn", "globalIn", "globalNotIn"} {
		b.Register(name, b.boolType, b.unknownType, b.unknownType)
	}
}

// registerExtendedWindowFunctions covers window functions added beyond the base lag,
// lead, rank, first_value, last_value and nth_value set.
//
// nonNegativeDerivative computes the discrete derivative across the window, clamping
// negative results to zero. ClickHouse accepts an optional interval third argument for
// time-axis scaling, so both arities are registered.
//
// Takes b (*FunctionCatalogueBuilder) which is the catalogue to register into.
func registerExtendedWindowFunctions(b *FunctionCatalogueBuilder) {
	b.Register("nonNegativeDerivative", b.float64Type, b.float64Type, b.dateTimeType)
	b.Register("nonNegativeDerivative", b.float64Type, b.float64Type, b.dateTimeType, b.textType)
}
