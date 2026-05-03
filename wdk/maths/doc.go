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

// Package maths provides arbitrary-precision numeric types with fluent APIs.
//
// Offers three core types for precise numerical computation: BigInt for
// arbitrary-precision integers, Decimal for high-precision decimal numbers (34
// digits), and Money for currency-aware monetary values. All types use a fluent
// API pattern and propagate the first error encountered in a chain of
// operations.
//
// # Fluent API and error propagation
//
// All numeric types follow a fluent API pattern where operations return
// new values rather than modifying in place (except for explicit
// *InPlace methods). Errors are captured within the type and propagated
// through chains:
//
//	result := maths.NewDecimalFromString("100.50").
//		Add(maths.NewDecimalFromString("25.25")).
//		MultiplyFloat(1.5)
//	if err := result.Err(); err != nil {
//		// Handle error once at the end
//	}
//
// # Aggregate operations
//
// The package provides aggregate functions for collections:
//
//	sum := maths.SumDecimals(values...)
//	avg := maths.AverageDecimals(values...)
//	min := maths.MinDecimal(a, b, c)
//	max := maths.MaxDecimal(a, b, c)
//
// # Allocation
//
// Values can be split proportionally using the Allocate method. The
// last portion receives any remainder to ensure the sum of parts
// equals the original value:
//
//	parts, err := maths.NewDecimalFromString("100").Allocate(1, 1, 1)
//
// # Currency conversion
//
// Two converters are provided for exchanging monetary values between
// currencies. [Converter] uses rates relative to a single base
// currency, whilst [MatrixConverter] uses a full rate matrix
// supporting direct and triangulated conversions:
//
//	rates, _ := maths.NewExchangeRates("GBP", map[string]maths.Decimal{
//		"USD": maths.NewDecimalFromString("1.27"),
//		"EUR": maths.NewDecimalFromString("1.17"),
//	})
//	converter := maths.NewConverter(rates)
//	usdAmount := converter.Convert(gbpAmount, "USD")
//
// # Type conversions
//
// Types can be converted between each other:
//
//	bigint := maths.NewBigIntFromInt(100)
//	decimal := bigint.ToDecimal()
//	money := maths.NewMoneyFromDecimal(decimal, "GBP")
//
// # Thread safety
//
// Individual values are not safe for concurrent modification. However,
// Money operations that access the currency registry
// ([NewMoneyFromDecimal], [RegisterCurrency]) use appropriate locking
// and are safe for concurrent use.
package maths
