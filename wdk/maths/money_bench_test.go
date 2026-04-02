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
	benchMoneyResult     maths.Money
	benchMoneyBoolResult bool
	moneyStrA            = "9876543210.12"
	moneyStrB            = "1234567890.98"
	moneyA               = maths.NewMoneyFromString(moneyStrA, "USD")
	moneyB               = maths.NewMoneyFromString(moneyStrB, "USD")
	moneyEUR             = maths.NewMoneyFromString("500.00", "EUR")
	scalarDecimal        = maths.NewDecimalFromInt(150)
)

func BenchmarkMoney_FromString(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMoneyResult = maths.NewMoneyFromString(moneyStrA, "USD")
	}
}

func BenchmarkMoney_FromMinorInt(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMoneyResult = maths.NewMoneyFromMinorInt(12345, "GBP")
	}
}

func BenchmarkMoney_Add(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMoneyResult = moneyA.Add(moneyB)
	}
}

func BenchmarkMoney_Multiply(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMoneyResult = moneyA.Multiply(scalarDecimal)
	}
}

func BenchmarkMoney_Chain(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {

		benchMoneyResult = moneyA.Subtract(moneyB).Multiply(scalarDecimal)
	}
}

func BenchmarkMoney_AddInPlace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		value := maths.NewMoneyFromString(moneyStrA, "USD")
		value.AddInPlace(moneyB)
	}
}

func BenchmarkMoney_Add_Loop_Immutable(b *testing.B) {
	value := maths.ZeroMoney("USD")
	addend := maths.NewMoneyFromMinorInt(1, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value = value.Add(addend)
	}
}

func BenchmarkMoney_Add_Loop_InPlace(b *testing.B) {
	value := maths.ZeroMoney("USD")
	addend := maths.NewMoneyFromMinorInt(1, "USD")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value.AddInPlace(addend)
	}
}

func BenchmarkMoney_Multiply_Loop_Immutable(b *testing.B) {
	value := maths.NewMoneyFromMinorInt(1, "USD")
	multiplier := maths.NewDecimalFromString("1.000001")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value = value.Multiply(multiplier)
	}
}

func BenchmarkMoney_Multiply_Loop_InPlace(b *testing.B) {
	value := maths.NewMoneyFromMinorInt(1, "USD")
	multiplier := maths.NewDecimalFromString("1.000001")
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value.MultiplyInPlace(multiplier)
	}
}

func BenchmarkMoney_Equals_SameCurrency(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMoneyBoolResult, _ = moneyA.Equals(moneyB)
	}
}

func BenchmarkMoney_Equals_DiffCurrency(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchMoneyBoolResult, _ = moneyA.Equals(moneyEUR)
	}
}

func BenchmarkMoney_RoundToStandard(b *testing.B) {
	m := maths.NewMoneyFromString("123.456789", "USD")
	b.ReportAllocs()
	for b.Loop() {
		benchMoneyResult = m.RoundToStandard()
	}
}

func BenchmarkMoney_Convert(b *testing.B) {

	rates, err := maths.NewExchangeRates("USD", map[string]maths.Decimal{
		"EUR": maths.NewDecimalFromString("0.92"),
		"GBP": maths.NewDecimalFromString("0.80"),
	})
	if err != nil {
		b.Fatalf("failed to setup converter: %v", err)
	}
	converter := maths.NewConverter(rates)
	source := maths.NewMoneyFromInt(100, "USD")

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		benchMoneyResult = converter.Convert(source, "EUR")
	}
}
