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

func TestAggregateFunctions(t *testing.T) {
	d1, d2, d3 := NewDecimalFromInt(10), NewDecimalFromInt(20), NewDecimalFromInt(30)
	dNeg := NewDecimalFromInt(-5)
	dInvalid := NewDecimalFromString("invalid")

	t.Run("SumDecimals", func(t *testing.T) {
		checkDecimal(t, SumDecimals(d1, d2, d3, dNeg), "55", false)
		checkDecimal(t, SumDecimals(), "0", false)
		checkDecimal(t, SumDecimals(d1, dInvalid, d3), "", true)
	})

	t.Run("AverageDecimals", func(t *testing.T) {
		checkDecimal(t, AverageDecimals(d1, d2, d3), "20", false)
		checkDecimal(t, AverageDecimals(dInvalid), "", true)
		checkDecimal(t, AverageDecimals(), "", true)
	})

	t.Run("MinDecimal", func(t *testing.T) {
		checkDecimal(t, MinDecimal(d1, d2, dNeg, d3), "-5", false)
		checkDecimal(t, MinDecimal(dInvalid, d1), "", true)
		checkDecimal(t, MinDecimal(d1, dInvalid), "", true)
	})

	t.Run("MaxDecimal", func(t *testing.T) {
		checkDecimal(t, MaxDecimal(d1, d2, dNeg, d3), "30", false)
		checkDecimal(t, MaxDecimal(dInvalid, d1), "", true)
		checkDecimal(t, MaxDecimal(d1, dInvalid), "", true)
	})

	t.Run("AbsSumDecimals", func(t *testing.T) {
		checkDecimal(t, AbsSumDecimals(d1, dNeg), "15", false)
		checkDecimal(t, AbsSumDecimals(d1, dInvalid), "", true)
	})
}

func TestSortFunctions(t *testing.T) {
	t.Run("SortDecimals", func(t *testing.T) {
		decs := []Decimal{
			NewDecimalFromInt(100),
			NewDecimalFromInt(-10),
			NewDecimalFromInt(20),
			NewDecimalFromInt(0),
		}
		err := SortDecimals(decs)
		if err != nil {
			t.Fatalf("SortDecimals failed: %v", err)
		}
		expectedOrder := []string{"-10", "0", "20", "100"}
		for i, d := range decs {
			s, _ := d.String()
			if s != expectedOrder[i] {
				t.Errorf("sort order incorrect at index %d: expected %s, got %s", i, expectedOrder[i], s)
			}
		}
	})

	t.Run("SortDecimalsReverse", func(t *testing.T) {
		decs := []Decimal{
			NewDecimalFromInt(100),
			NewDecimalFromInt(-10),
			NewDecimalFromInt(20),
			NewDecimalFromInt(0),
		}
		err := SortDecimalsReverse(decs)
		if err != nil {
			t.Fatalf("SortDecimalsReverse failed: %v", err)
		}
		expectedOrder := []string{"100", "20", "0", "-10"}
		for i, d := range decs {
			s, _ := d.String()
			if s != expectedOrder[i] {
				t.Errorf("reverse sort order incorrect at index %d: expected %s, got %s", i, expectedOrder[i], s)
			}
		}
	})

	t.Run("SortWithInvalid", func(t *testing.T) {
		decs := []Decimal{
			NewDecimalFromInt(100),
			NewDecimalFromString("invalid"),
			NewDecimalFromInt(20),
		}
		err := SortDecimals(decs)
		if err == nil {
			t.Error("expected SortDecimals to fail with an invalid decimal in the slice")
		}
	})
}

func TestAllocation(t *testing.T) {
	t.Run("AllocateEvenly", func(t *testing.T) {
		d100 := NewDecimalFromInt(100)
		parts, err := d100.Allocate(1, 1, 1, 1)
		if err != nil {
			t.Fatalf("Allocate failed: %v", err)
		}
		if len(parts) != 4 {
			t.Fatalf("expected 4 parts, got %d", len(parts))
		}
		for i := range 4 {
			checkDecimal(t, parts[i], "25", false)
		}
		sum := SumDecimals(parts...)
		checkDecimal(t, sum, "100", false)
	})

	t.Run("AllocateWithRemainder", func(t *testing.T) {
		d100 := NewDecimalFromString("10.01")
		parts, err := d100.Allocate(1, 1, 1)
		if err != nil {
			t.Fatalf("Allocate failed: %v", err)
		}
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(parts))
		}

		checkDecimal(t, parts[0], "3.336666666666666666666666666666667", false)
		checkDecimal(t, parts[1], "3.336666666666666666666666666666667", false)
		checkDecimal(t, parts[2], "3.336666666666666666666666666666666", false)

		sum := SumDecimals(parts...)
		checkDecimal(t, sum, "10.01", false)
	})

	t.Run("AllocateErrorCases", func(t *testing.T) {
		d100 := NewDecimalFromInt(100)
		_, err := d100.Allocate()
		if err != nil {
			t.Errorf("Allocate with no ratios should not error, got %v", err)
		}

		_, err = d100.Allocate(1, -1)
		if err == nil {
			t.Error("expected error for negative ratio")
		}

		_, err = d100.Allocate(0, 0)
		if err == nil {
			t.Error("expected error for zero sum of ratios")
		}

		dInvalid := NewDecimalFromString("invalid")
		_, err = dInvalid.Allocate(1, 1)
		if err == nil {
			t.Error("expected error when allocating an invalid decimal")
		}
	})
}
