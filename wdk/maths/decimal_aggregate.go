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

// Allocate splits a Decimal value into parts based on the given ratios.
// Any remainder is added to the last part so the sum of parts equals the
// original value.
//
// Takes ratios (...int64) which specifies the proportions for splitting.
//
// Returns []Decimal which contains the split portions.
// Returns error when a ratio is negative or the sum of ratios is zero.
func (d Decimal) Allocate(ratios ...int64) ([]Decimal, error) {
	if d.err != nil {
		return nil, d.err
	}
	if len(ratios) == 0 {
		return []Decimal{}, nil
	}

	totalRatio, err := validateAndSumRatios(ratios)
	if err != nil {
		return nil, err
	}

	totalRatioDecimal := NewDecimalFromInt(totalRatio)
	if totalRatioDecimal.err != nil {
		return nil, totalRatioDecimal.err
	}

	return allocateDecimalPortions(d, ratios, totalRatioDecimal)
}

// SumDecimals returns the sum of all given decimal values.
//
// Takes decimals (...Decimal) which are the values to add together.
//
// Returns Decimal which is the total sum. Returns zero if no values are given.
func SumDecimals(decimals ...Decimal) Decimal {
	sum := ZeroDecimal()
	for _, d := range decimals {
		sum = sum.Add(d)
	}
	return sum
}

// AverageDecimals returns the average of all decimals in a slice.
//
// Takes decimals (...Decimal) which are the values to average.
//
// Returns Decimal which is the calculated average, or an error-state Decimal
// if the slice is empty.
func AverageDecimals(decimals ...Decimal) Decimal {
	if len(decimals) == 0 {
		return Decimal{err: errors.New("maths: cannot calculate average of an empty slice")}
	}
	sum := SumDecimals(decimals...)
	return sum.Divide(NewDecimalFromInt(int64(len(decimals))))
}

// MinDecimal returns the smallest value from the given decimals.
//
// Takes d1 (Decimal) which is the first value to compare.
// Takes others (...Decimal) which are extra values to compare.
//
// Returns Decimal which is the smallest value found. If any input has an
// error or the comparison fails, returns the first value that has an error.
func MinDecimal(d1 Decimal, others ...Decimal) Decimal {
	if d1.err != nil {
		return d1
	}
	minValue := d1
	for _, d := range others {
		if d.err != nil {
			return d
		}
		isLess, err := d.LessThan(minValue)
		if err != nil {
			return Decimal{err: fmt.Errorf("maths: Min comparison failed: %w", err)}
		}
		if isLess {
			minValue = d
		}
	}
	return minValue
}

// MaxDecimal returns the largest value from a list of Decimal values.
//
// Takes d1 (Decimal) which is the first value to compare.
// Takes others (...Decimal) which are extra values to compare.
//
// Returns Decimal which is the largest value found. If any input has an error
// or the comparison fails, returns the first value that contains an error.
func MaxDecimal(d1 Decimal, others ...Decimal) Decimal {
	if d1.err != nil {
		return d1
	}
	maxValue := d1
	for _, d := range others {
		if d.err != nil {
			return d
		}
		isGreater, err := d.GreaterThan(maxValue)
		if err != nil {
			return Decimal{err: fmt.Errorf("maths: Max comparison failed: %w", err)}
		}
		if isGreater {
			maxValue = d
		}
	}
	return maxValue
}

// AbsSumDecimals returns the sum of the absolute values of the given decimals.
//
// Takes decimals (...Decimal) which are the values to sum.
//
// Returns Decimal which is the sum of all absolute values.
func AbsSumDecimals(decimals ...Decimal) Decimal {
	sum := ZeroDecimal()
	for _, d := range decimals {
		sum = sum.Add(d.Abs())
	}
	return sum
}

// SortDecimals sorts a slice of Decimals in ascending order.
//
// Takes decimals ([]Decimal) which is the slice to sort in place.
//
// Returns error when any decimal in the slice is in an error state or when
// an unexpected comparison error occurs during sorting.
func SortDecimals(decimals []Decimal) error {
	for i, d := range decimals {
		if d.err != nil {
			return fmt.Errorf("maths: cannot sort slice with error at index %d: %w", i, d.err)
		}
	}
	var sortErr error
	slices.SortFunc(decimals, func(a, b Decimal) int {
		if sortErr != nil {
			return 0
		}
		isLess, err := a.LessThan(b)
		if err != nil {
			sortErr = fmt.Errorf("maths: unexpected error during sort comparison: %w", err)
			return 0
		}
		if isLess {
			return -1
		}
		return 1
	})
	return sortErr
}

// SortDecimalsReverse sorts a slice of Decimals in descending order.
//
// Takes decimals ([]Decimal) which is the slice to sort in place.
//
// Returns error when any decimal in the slice is in an error state or when
// an unexpected comparison error occurs during sorting.
func SortDecimalsReverse(decimals []Decimal) error {
	for i, d := range decimals {
		if d.err != nil {
			return fmt.Errorf("maths: cannot sort slice with error at index %d: %w", i, d.err)
		}
	}
	var sortErr error
	slices.SortFunc(decimals, func(a, b Decimal) int {
		if sortErr != nil {
			return 0
		}
		isGreater, err := a.GreaterThan(b)
		if err != nil {
			sortErr = fmt.Errorf("maths: unexpected error during sort comparison: %w", err)
			return 0
		}
		if isGreater {
			return -1
		}
		return 1
	})
	return sortErr
}

// allocateDecimalPortions distributes a Decimal across the given ratios. The
// last portion receives the remainder to ensure the sum equals the original
// value.
//
// Takes d (Decimal) which is the value to distribute.
// Takes ratios ([]int64) which specifies the distribution proportions.
// Takes totalRatioDecimal (Decimal) which is the sum of all ratios.
//
// Returns []Decimal which contains the distributed portions.
// Returns error when an arithmetic operation fails during allocation.
func allocateDecimalPortions(d Decimal, ratios []int64, totalRatioDecimal Decimal) ([]Decimal, error) {
	numRatios := len(ratios)
	portions := make([]Decimal, numRatios)
	sumAllocated := ZeroDecimal()
	lastIndex := numRatios - 1

	for i, ratio := range ratios {
		if i == lastIndex {
			rem := d.Subtract(sumAllocated)
			if rem.err != nil {
				return nil, rem.err
			}
			portions[i] = rem
			continue
		}
		ratioDec := NewDecimalFromInt(ratio)
		p := d.Multiply(ratioDec).Divide(totalRatioDecimal)
		if p.err != nil {
			return nil, fmt.Errorf("maths: allocation failed for ratio %d: %w", ratio, p.err)
		}
		portions[i] = p
		sumAllocated = sumAllocated.Add(p)
	}
	return portions, nil
}
