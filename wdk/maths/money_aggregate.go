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
	"errors"
	"fmt"
	"slices"
)

// Allocate splits a Money value into parts based on the given ratios.
// Any remainder is added to the last part so the sum of all parts equals
// the original value.
//
// Takes ratios (...int64) which specifies how to divide the value.
//
// Returns []Money which contains the allocated parts.
// Returns error when the Money has an existing error or allocation fails.
func (m Money) Allocate(ratios ...int64) ([]Money, error) {
	if m.err != nil {
		return nil, m.err
	}
	decAmount, err := m.Amount()
	if err != nil {
		return nil, err
	}
	allocatedDecs, err := decAmount.Allocate(ratios...)
	if err != nil {
		return nil, err
	}

	results := make([]Money, len(allocatedDecs))
	code, _ := m.CurrencyCode()
	for i, dec := range allocatedDecs {
		results[i] = NewMoneyFromDecimal(dec, code)
		if results[i].err != nil {
			return nil, results[i].err
		}
	}
	return results, nil
}

// AverageMoney returns the average of the given money values.
// All values must have the same currency.
//
// Takes monies (...Money) which provides the values to average.
//
// Returns Money which contains the calculated average, or an error-state Money
// if the slice is empty or currencies do not match.
func AverageMoney(monies ...Money) Money {
	if len(monies) == 0 {
		return Money{err: errors.New("money: cannot calculate average of an empty slice")}
	}

	sum := SumMoney(monies...)
	if sum.err != nil {
		return sum
	}

	count := NewDecimalFromInt(int64(len(monies)))
	return sum.Divide(count)
}

// SumMoney returns the sum of all money values in a slice.
// All values must have the same currency.
//
// When the slice is empty, returns an error-state Money.
//
// Takes monies (...Money) which are the values to sum.
//
// Returns Money which is the total sum of all provided values.
func SumMoney(monies ...Money) Money {
	if len(monies) == 0 {
		return Money{err: errors.New("money: cannot sum an empty slice")}
	}
	sum := monies[0]
	for i := 1; i < len(monies); i++ {
		sum = sum.Add(monies[i])
	}
	return sum
}

// AbsSumMoney returns the sum of the absolute values of all money in a slice.
// All values must have the same currency.
//
// When the slice is empty, returns an error-state Money.
//
// Takes monies (...Money) which are the monetary values to sum.
//
// Returns Money which is the absolute sum of all values.
func AbsSumMoney(monies ...Money) Money {
	if len(monies) == 0 {
		return Money{err: errors.New("money: cannot calculate absolute sum of an empty slice")}
	}

	first := monies[0]
	if first.err != nil {
		return first
	}
	code, _ := first.CurrencyCode()

	sum := ZeroMoney(code)
	for _, m := range monies {
		sum = sum.Add(m.Abs())
	}
	return sum
}

// MinMoney returns the smallest Money value from a list of arguments.
// All values must have the same currency.
//
// Takes m1 (Money) which is the first value to compare.
// Takes others (...Money) which are extra values to compare.
//
// Returns Money which is the smallest value. Returns an error Money if any
// value has an error or if currencies do not match.
func MinMoney(m1 Money, others ...Money) Money {
	if m1.err != nil {
		return m1
	}
	minValue := m1
	for _, m := range others {
		if m.err != nil {
			return m
		}
		cmp, err := m.amount.Cmp(minValue.amount)
		if err != nil {
			return Money{err: err}
		}
		if cmp < 0 {
			minValue = m
		}
	}
	return minValue
}

// MaxMoney returns the largest money value from a list of arguments.
// All values must have the same currency.
//
// Takes m1 (Money) which is the first value to compare.
// Takes others (...Money) which are extra values to compare.
//
// Returns Money which is the largest value. If any value has an error or the
// currencies do not match, returns a Money with an error instead.
func MaxMoney(m1 Money, others ...Money) Money {
	if m1.err != nil {
		return m1
	}
	maxValue := m1
	for _, m := range others {
		if m.err != nil {
			return m
		}
		cmp, err := m.amount.Cmp(maxValue.amount)
		if err != nil {
			return Money{err: err}
		}
		if cmp > 0 {
			maxValue = m
		}
	}
	return maxValue
}

// SortMoney sorts a slice of Money values in ascending order.
//
// Takes monies ([]Money) which is the slice to sort in place.
//
// Returns error when any value has an error, when currencies are mismatched,
// or when an unexpected comparison error occurs during sorting.
func SortMoney(monies []Money) error {
	return sortMoneyInternal(monies, Money.LessThan)
}

// SortMoneyReverse sorts a slice of Money values in descending order.
//
// Takes monies ([]Money) which contains the values to sort in place.
//
// Returns error when any value has an error, when currencies are mismatched,
// or when an unexpected comparison error occurs during sorting.
func SortMoneyReverse(monies []Money) error {
	return sortMoneyInternal(monies, Money.GreaterThan)
}

// sortMoneyInternal validates currency consistency and sorts using the given
// comparison function.
//
// Takes monies ([]Money) which is the slice to sort in place.
// Takes compare (func(Money, Money) (bool, error)) which returns true when
// the first argument should come before the second.
//
// Returns error when any value has an error, when currencies are mismatched,
// or when an unexpected comparison error occurs during sorting.
func sortMoneyInternal(monies []Money, compare func(Money, Money) (bool, error)) error {
	if len(monies) <= 1 {
		return nil
	}

	firstCurrency, err := monies[0].CurrencyCode()
	if err != nil {
		return fmt.Errorf("money: cannot sort slice with error at index 0: %w", err)
	}
	for i := 1; i < len(monies); i++ {
		if monies[i].err != nil {
			return fmt.Errorf("money: cannot sort slice with error at index %d: %w", i, monies[i].err)
		}
		currentCurrency, _ := monies[i].CurrencyCode()
		if currentCurrency != firstCurrency {
			return fmt.Errorf("money: currency mismatch in slice for sorting ('%s' vs '%s')", firstCurrency, currentCurrency)
		}
	}

	var sortErr error
	slices.SortFunc(monies, func(a, b Money) int {
		if sortErr != nil {
			return 0
		}
		result, cmpErr := compare(a, b)
		if cmpErr != nil {
			sortErr = fmt.Errorf("money: unexpected error during sort comparison: %w", cmpErr)
			return 0
		}
		if result {
			return -1
		}
		return 1
	})
	return sortErr
}
