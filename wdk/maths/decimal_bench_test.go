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

//go:build bench

package maths_test

import (
	"testing"

	"piko.sh/piko/wdk/maths"
)

var (
	benchDecimalResult     maths.Decimal
	benchDecimalBoolResult bool
	decimalStrA            = "9876543210.123456789"
	decimalStrB            = "1234567890.987654321"
	decimalA               = maths.NewDecimalFromString(decimalStrA)
	decimalB               = maths.NewDecimalFromString(decimalStrB)
	decimalC               = maths.NewDecimalFromInt(150)
)

func BenchmarkDecimal_FromString(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalResult = maths.NewDecimalFromString(decimalStrA)
	}
}

func BenchmarkDecimal_FromFloat(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalResult = maths.NewDecimalFromFloat(12345.6789)
	}
}

func BenchmarkDecimal_Add(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalResult = decimalA.Add(decimalB)
	}
}

func BenchmarkDecimal_Subtract(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalResult = decimalA.Subtract(decimalB)
	}
}

func BenchmarkDecimal_Multiply(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalResult = decimalA.Multiply(decimalB)
	}
}

func BenchmarkDecimal_Divide(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalResult = decimalA.Divide(decimalC)
	}
}

func BenchmarkDecimal_Chain(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {

		benchDecimalResult = decimalA.Add(decimalB).Divide(decimalC)
	}
}

func BenchmarkDecimal_AddInPlace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		value := maths.NewDecimalFromString(decimalStrA)
		value.AddInPlace(decimalB)
	}
}

func BenchmarkDecimal_MultiplyInPlace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		value := maths.NewDecimalFromString(decimalStrA)
		value.MultiplyInPlace(decimalC)
	}
}

func BenchmarkDecimal_Add_Loop_Immutable(b *testing.B) {
	value := maths.ZeroDecimal()
	addend := maths.NewDecimalFromInt(1)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value = value.Add(addend)
	}
}

func BenchmarkDecimal_Add_Loop_InPlace(b *testing.B) {
	value := maths.ZeroDecimal()
	addend := maths.NewDecimalFromInt(1)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value.AddInPlace(addend)
	}
}

func BenchmarkDecimal_Multiply_Loop_Immutable(b *testing.B) {
	value := maths.OneDecimal()
	multiplier := maths.NewDecimalFromString("1.000001")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value = value.Multiply(multiplier)
	}
}

func BenchmarkDecimal_Multiply_Loop_InPlace(b *testing.B) {
	value := maths.OneDecimal()
	multiplier := maths.NewDecimalFromString("1.000001")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value.MultiplyInPlace(multiplier)
	}
}

func BenchmarkDecimal_Equals(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalBoolResult, _ = decimalA.Equals(decimalB)
	}
}

func BenchmarkDecimal_IsInteger(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchDecimalBoolResult, _ = decimalA.IsInteger()
	}
}
