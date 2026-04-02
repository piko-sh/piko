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
	benchBigIntResult maths.BigInt
	benchBoolResult   bool
	bigintStrA        = "98765432109876543210"
	bigintStrB        = "12345678901234567890"
	bigintA           = maths.NewBigIntFromString(bigintStrA)
	bigintB           = maths.NewBigIntFromString(bigintStrB)
	bigintC           = maths.NewBigIntFromInt(100)
)

func BenchmarkBigInt_FromString(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchBigIntResult = maths.NewBigIntFromString(bigintStrA)
	}
}

func BenchmarkBigInt_FromInt(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchBigIntResult = maths.NewBigIntFromInt(1234567890)
	}
}

func BenchmarkBigInt_Add(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchBigIntResult = bigintA.Add(bigintB)
	}
}

func BenchmarkBigInt_Subtract(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchBigIntResult = bigintA.Subtract(bigintB)
	}
}

func BenchmarkBigInt_Multiply(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchBigIntResult = bigintA.Multiply(bigintB)
	}
}

func BenchmarkBigInt_Divide(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchBigIntResult = bigintA.Divide(bigintC)
	}
}

func BenchmarkBigInt_Chain(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {

		benchBigIntResult = bigintA.Add(bigintB).Multiply(bigintC)
	}
}

func BenchmarkBigInt_AddInPlace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		value := maths.NewBigIntFromString(bigintStrA)
		value.AddInPlace(bigintB)
	}
}

func BenchmarkBigInt_MultiplyInPlace(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		value := maths.NewBigIntFromString(bigintStrA)
		value.MultiplyInPlace(bigintC)
	}
}

func BenchmarkBigInt_Add_Loop_Immutable(b *testing.B) {
	value := maths.ZeroBigInt()
	addend := maths.NewBigIntFromInt(1)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {

		value = value.Add(addend)
	}
}

func BenchmarkBigInt_Add_Loop_InPlace(b *testing.B) {
	value := maths.ZeroBigInt()
	addend := maths.NewBigIntFromInt(1)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {

		value.AddInPlace(addend)
	}
}

func BenchmarkBigInt_Multiply_Loop_Immutable(b *testing.B) {
	value := maths.OneBigInt()
	multiplier := maths.NewBigIntFromInt(2)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value = value.Multiply(multiplier)
	}
}

func BenchmarkBigInt_Multiply_Loop_InPlace(b *testing.B) {
	value := maths.OneBigInt()
	multiplier := maths.NewBigIntFromInt(2)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		value.MultiplyInPlace(multiplier)
	}
}

func BenchmarkBigInt_Equals(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		benchBoolResult, _ = bigintA.Equals(bigintB)
	}
}

func BenchmarkBigInt_IsZero(b *testing.B) {
	zero := maths.ZeroBigInt()
	b.ReportAllocs()
	for b.Loop() {
		benchBoolResult, _ = zero.IsZero()
	}
}
