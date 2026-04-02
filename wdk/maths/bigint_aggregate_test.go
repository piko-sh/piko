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

package maths

import (
	"testing"
)

func TestBigIntAggregateFunctions(t *testing.T) {
	b10 := NewBigIntFromInt(10)
	b20 := NewBigIntFromInt(20)
	b30 := NewBigIntFromInt(30)
	bNeg5 := NewBigIntFromInt(-5)
	bInvalid := NewBigIntFromString("invalid")

	t.Run("SumBigInts", func(t *testing.T) {
		checkBigInt(t, SumBigInts(b10, b20, b30, bNeg5), "55", false)
		checkBigInt(t, SumBigInts(), "0", false)
		checkBigInt(t, SumBigInts(b10, bInvalid, b30), "", true)
	})

	t.Run("AverageBigInts", func(t *testing.T) {

		checkDecimal(t, AverageBigInts(b10, b20, b30), "20", false)
		checkDecimal(t, AverageBigInts(NewBigIntFromInt(1), NewBigIntFromInt(2)), "1.5", false)
		checkDecimal(t, AverageBigInts(bInvalid), "", true)
		checkDecimal(t, AverageBigInts(), "", true)
	})

	t.Run("MinBigInt", func(t *testing.T) {
		checkBigInt(t, MinBigInt(b10, b20, bNeg5, b30), "-5", false)
		checkBigInt(t, MinBigInt(bInvalid, b10), "", true)
		checkBigInt(t, MinBigInt(b10, bInvalid), "", true)
	})

	t.Run("MaxBigInt", func(t *testing.T) {
		checkBigInt(t, MaxBigInt(b10, b20, bNeg5, b30), "30", false)
		checkBigInt(t, MaxBigInt(bInvalid, b10), "", true)
		checkBigInt(t, MaxBigInt(b10, bInvalid), "", true)
	})

	t.Run("AbsSumBigInts", func(t *testing.T) {

		checkBigInt(t, AbsSumBigInts(b10, bNeg5), "15", false)
		checkBigInt(t, AbsSumBigInts(b10, bInvalid), "", true)
	})
}

func TestBigIntSortFunctions(t *testing.T) {
	t.Run("SortBigInts", func(t *testing.T) {
		bigints := []BigInt{
			NewBigIntFromInt(100),
			NewBigIntFromInt(-10),
			NewBigIntFromInt(20),
			NewBigIntFromInt(0),
		}
		err := SortBigInts(bigints)
		if err != nil {
			t.Fatalf("SortBigInts failed: %v", err)
		}
		expectedOrder := []string{"-10", "0", "20", "100"}
		for i, b := range bigints {
			s, _ := b.String()
			if s != expectedOrder[i] {
				t.Errorf("sort order incorrect at index %d: expected %s, got %s", i, expectedOrder[i], s)
			}
		}
	})

	t.Run("SortBigIntsReverse", func(t *testing.T) {
		bigints := []BigInt{
			NewBigIntFromInt(100),
			NewBigIntFromInt(-10),
			NewBigIntFromInt(20),
			NewBigIntFromInt(0),
		}
		err := SortBigIntsReverse(bigints)
		if err != nil {
			t.Fatalf("SortBigIntsReverse failed: %v", err)
		}
		expectedOrder := []string{"100", "20", "0", "-10"}
		for i, b := range bigints {
			s, _ := b.String()
			if s != expectedOrder[i] {
				t.Errorf("reverse sort order incorrect at index %d: expected %s, got %s", i, expectedOrder[i], s)
			}
		}
	})

	t.Run("SortWithInvalid", func(t *testing.T) {
		bigints := []BigInt{
			NewBigIntFromInt(100),
			NewBigIntFromString("invalid"),
			NewBigIntFromInt(20),
		}
		err := SortBigInts(bigints)
		if err == nil {
			t.Error("expected SortBigInts to fail with an invalid bigint in the slice")
		}
	})
}

func TestBigIntAllocation(t *testing.T) {
	t.Run("AllocateEvenly", func(t *testing.T) {
		b100 := NewBigIntFromInt(100)
		parts, err := b100.Allocate(1, 1, 1, 1)
		if err != nil {
			t.Fatalf("Allocate failed: %v", err)
		}
		if len(parts) != 4 {
			t.Fatalf("expected 4 parts, got %d", len(parts))
		}
		for i := range 4 {
			checkBigInt(t, parts[i], "25", false)
		}
		sum := SumBigInts(parts...)
		checkBigInt(t, sum, "100", false)
	})

	t.Run("AllocateWithRemainder", func(t *testing.T) {
		b101 := NewBigIntFromInt(101)
		parts, err := b101.Allocate(1, 1, 1)
		if err != nil {
			t.Fatalf("Allocate failed: %v", err)
		}
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(parts))
		}

		checkBigInt(t, parts[0], "33", false)
		checkBigInt(t, parts[1], "33", false)
		checkBigInt(t, parts[2], "35", false)

		sum := SumBigInts(parts...)
		checkBigInt(t, sum, "101", false)
	})

	t.Run("AllocateErrorCases", func(t *testing.T) {
		b100 := NewBigIntFromInt(100)
		_, err := b100.Allocate()
		if err != nil {
			t.Errorf("Allocate with no ratios should not error, got %v", err)
		}

		_, err = b100.Allocate(1, -1)
		if err == nil {
			t.Error("expected error for negative ratio")
		}

		_, err = b100.Allocate(0, 0)
		if err == nil {
			t.Error("expected error for zero sum of ratios")
		}

		bInvalid := NewBigIntFromString("invalid")
		_, err = bInvalid.Allocate(1, 1)
		if err == nil {
			t.Error("expected error when allocating an invalid bigint")
		}
	})
}
