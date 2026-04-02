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

func TestMoneyAggregateFunctions(t *testing.T) {
	m10USD := NewMoneyFromInt(10, "USD")
	m20USD := NewMoneyFromInt(20, "USD")
	m30USD := NewMoneyFromInt(30, "USD")
	mNeg5USD := NewMoneyFromInt(-5, "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")
	m5EUR := NewMoneyFromInt(5, "EUR")

	t.Run("SumMoney", func(t *testing.T) {
		checkMoney(t, SumMoney(m10USD, m20USD, m30USD), "60", "USD", false)
		checkMoney(t, SumMoney(m10USD, mNeg5USD), "5", "USD", false)
		checkMoney(t, SumMoney(m10USD, m20USD, mInvalid), "", "", true)
		checkMoney(t, SumMoney(m10USD, m5EUR), "", "", true)
		checkMoney(t, SumMoney(), "", "", true)
	})

	t.Run("AverageMoney", func(t *testing.T) {
		checkMoney(t, AverageMoney(m10USD, m20USD, m30USD), "20", "USD", false)
		checkMoney(t, AverageMoney(m10USD, m20USD), "15", "USD", false)
		checkMoney(t, AverageMoney(m10USD, mInvalid), "", "", true)
		checkMoney(t, AverageMoney(), "", "", true)
	})

	t.Run("AbsSumMoney", func(t *testing.T) {
		checkMoney(t, AbsSumMoney(m10USD, mNeg5USD), "15", "USD", false)
		checkMoney(t, AbsSumMoney(mNeg5USD, mNeg5USD), "10", "USD", false)
		checkMoney(t, AbsSumMoney(), "", "", true)
		checkMoney(t, AbsSumMoney(m10USD, mInvalid), "", "", true)
	})

	t.Run("MinMoney", func(t *testing.T) {
		checkMoney(t, MinMoney(m10USD, m20USD, mNeg5USD, m30USD), "-5", "USD", false)
		checkMoney(t, MinMoney(m20USD, m10USD), "10", "USD", false)
		checkMoney(t, MinMoney(m10USD, mInvalid), "", "", true)
		checkMoney(t, MinMoney(mInvalid, m10USD), "", "", true)
		checkMoney(t, MinMoney(m10USD, m5EUR), "", "", true)
	})

	t.Run("MaxMoney", func(t *testing.T) {
		checkMoney(t, MaxMoney(m10USD, m20USD, mNeg5USD, m30USD), "30", "USD", false)
		checkMoney(t, MaxMoney(mNeg5USD, m10USD), "10", "USD", false)
		checkMoney(t, MaxMoney(m10USD, mInvalid), "", "", true)
		checkMoney(t, MaxMoney(mInvalid, m10USD), "", "", true)
		checkMoney(t, MaxMoney(m10USD, m5EUR), "", "", true)
	})
}

func TestMoneySortFunctions(t *testing.T) {
	m100 := NewMoneyFromInt(100, "USD")
	mNeg10 := NewMoneyFromInt(-10, "USD")
	m20 := NewMoneyFromInt(20, "USD")
	mZero := ZeroMoney("USD")

	t.Run("SortMoney", func(t *testing.T) {
		monies := []Money{m100, mNeg10, m20, mZero}
		err := SortMoney(monies)
		if err != nil {
			t.Fatalf("SortMoney failed: %v", err)
		}

		expectedOrder := []Money{mNeg10, mZero, m20, m100}
		for i, m := range monies {
			eq, _ := m.Equals(expectedOrder[i])
			if !eq {
				t.Errorf("sort order incorrect at index %d: expected %s, got %s", i, expectedOrder[i].MustString(), m.MustString())
			}
		}
	})

	t.Run("SortMoneyReverse", func(t *testing.T) {
		monies := []Money{m100, mNeg10, m20, mZero}
		err := SortMoneyReverse(monies)
		if err != nil {
			t.Fatalf("SortMoneyReverse failed: %v", err)
		}

		expectedOrder := []Money{m100, m20, mZero, mNeg10}
		for i, m := range monies {
			eq, _ := m.Equals(expectedOrder[i])
			if !eq {
				t.Errorf("reverse sort order incorrect at index %d: expected %s, got %s", i, expectedOrder[i].MustString(), m.MustString())
			}
		}
	})

	t.Run("SortMoney Error Cases", func(t *testing.T) {
		invalidSlice := []Money{m100, NewMoneyFromString("invalid", "USD")}
		err := SortMoney(invalidSlice)
		if err == nil {
			t.Error("expected SortMoney to fail with an invalid money in the slice")
		}

		mismatchedSlice := []Money{m100, NewMoneyFromInt(5, "EUR")}
		err = SortMoney(mismatchedSlice)
		if err == nil {
			t.Error("expected SortMoney to fail with mismatched currencies")
		}
	})
}

func TestMoneyAllocation(t *testing.T) {
	t.Run("Allocate", func(t *testing.T) {
		original := NewMoneyFromString("10.00", "USD")
		parts, err := original.Allocate(1, 2, 3)
		if err != nil {
			t.Fatalf("Allocate failed: %v", err)
		}
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(parts))
		}
		sum := SumMoney(parts...)
		eq, _ := sum.Equals(original)
		if !eq {
			t.Errorf("sum of allocated parts (%s) must equal original value (%s)", sum.MustString(), original.MustString())
		}
	})

	t.Run("AllocateWithSubMinorUnits", func(t *testing.T) {
		m := NewMoneyFromString("0.01", "USD")
		parts, err := m.Allocate(1, 1)
		if err != nil {
			t.Fatal(err)
		}

		checkMoney(t, parts[0], "0.005", "USD", false)
		checkMoney(t, parts[1], "0.005", "USD", false)

		sum := SumMoney(parts...)
		checkMoney(t, sum, "0.01", "USD", false)
	})

	t.Run("Allocate On Negative Money", func(t *testing.T) {
		m := NewMoneyFromInt(-100, "USD")
		parts, err := m.Allocate(1, 1)
		if err != nil {
			t.Errorf("unexpected error allocating money: %v", err)
		}
		if len(parts) != 2 {
			t.Errorf("expected 2 parts, got %d", len(parts))
		}
		checkMoney(t, parts[0], "-50", "USD", false)
		checkMoney(t, parts[1], "-50", "USD", false)
	})

	t.Run("Allocate Error Cases", func(t *testing.T) {
		m100 := NewMoneyFromInt(100, "USD")
		mInvalid := NewMoneyFromString("invalid", "USD")

		_, err := m100.Allocate()
		if err != nil {
			t.Errorf("Allocate with no ratios should return empty slice, not error: %v", err)
		}

		_, err = m100.Allocate(1, -1)
		if err == nil {
			t.Error("expected error for negative ratio")
		}

		_, err = m100.Allocate(0, 0)
		if err == nil {
			t.Error("expected error for zero sum of ratios")
		}

		_, err = mInvalid.Allocate(1, 1)
		if err == nil {
			t.Error("Allocate should propagate errors")
		}
	})
}
