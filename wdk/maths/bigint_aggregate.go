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

// Allocate splits a BigInt value into parts based on the given ratios.
// This uses integer division; any remainder goes to the last part so the sum
// of all parts equals the original value.
//
// Takes ratios (...int64) which specifies the proportional distribution.
//
// Returns []BigInt which contains the allocated parts.
// Returns error when any ratio is negative or the sum of ratios is zero.
func (b BigInt) Allocate(ratios ...int64) ([]BigInt, error) {
	if b.err != nil {
		return nil, b.err
	}
	if len(ratios) == 0 {
		return []BigInt{}, nil
	}

	totalRatio, err := validateAndSumRatios(ratios)
	if err != nil {
		return nil, err
	}

	totalRatioBigInt := NewBigIntFromInt(totalRatio)
	if totalRatioBigInt.err != nil {
		return nil, totalRatioBigInt.err
	}

	return allocateBigIntPortions(b, ratios, totalRatioBigInt)
}

// SumBigInts returns the sum of all bigints in a slice.
//
// Takes bigints (...BigInt) which are the values to sum.
//
// Returns BigInt which is the total sum, or zero if no values are provided.
func SumBigInts(bigints ...BigInt) BigInt {
	sum := ZeroBigInt()
	for _, b := range bigints {
		sum = sum.Add(b)
	}
	return sum
}

// AverageBigInts returns the average of all big integers in a slice.
// It returns a Decimal to preserve precision, as the average of integers
// is often fractional.
//
// Takes bigints (...BigInt) which are the values to average.
//
// Returns Decimal which contains the average, or an error-state Decimal
// if the slice is empty or any calculation fails.
func AverageBigInts(bigints ...BigInt) Decimal {
	if len(bigints) == 0 {
		return Decimal{err: errors.New("maths: cannot calculate average of an empty slice")}
	}

	sumBigInt := SumBigInts(bigints...)
	if sumBigInt.Err() != nil {
		return ZeroDecimalWithError(sumBigInt.Err())
	}
	sumString, err := sumBigInt.String()
	if err != nil {
		return ZeroDecimalWithError(err)
	}

	sumDecimal := NewDecimalFromString(sumString)
	countDecimal := NewDecimalFromInt(int64(len(bigints)))

	return sumDecimal.Divide(countDecimal)
}

// MinBigInt returns the smallest BigInt from the given values.
//
// Takes b1 (BigInt) which is the first value to compare.
// Takes others (...BigInt) which are extra values to compare.
//
// Returns BigInt which is the smallest value. If any input has an error or if
// comparison fails, returns a BigInt that holds the error.
func MinBigInt(b1 BigInt, others ...BigInt) BigInt {
	if b1.err != nil {
		return b1
	}
	minValue := b1
	for _, b := range others {
		if b.err != nil {
			return b
		}
		cmp, err := b.Cmp(minValue)
		if err != nil {
			return BigInt{err: fmt.Errorf("maths: Min comparison failed: %w", err)}
		}
		if cmp < 0 {
			minValue = b
		}
	}
	return minValue
}

// MaxBigInt returns the largest value from a set of BigInt values.
//
// Takes b1 (BigInt) which is the first value to compare.
// Takes others (...BigInt) which are additional values to compare.
//
// Returns BigInt which is the largest value found. If any input has an error
// or if a comparison fails, returns a BigInt carrying that error.
func MaxBigInt(b1 BigInt, others ...BigInt) BigInt {
	if b1.err != nil {
		return b1
	}
	maxValue := b1
	for _, b := range others {
		if b.err != nil {
			return b
		}
		cmp, err := b.Cmp(maxValue)
		if err != nil {
			return BigInt{err: fmt.Errorf("maths: Max comparison failed: %w", err)}
		}
		if cmp > 0 {
			maxValue = b
		}
	}
	return maxValue
}

// AbsSumBigInts returns the sum of the absolute values of all bigints.
//
// Takes bigints (...BigInt) which are the values to sum.
//
// Returns BigInt which is the sum of all absolute values.
func AbsSumBigInts(bigints ...BigInt) BigInt {
	sum := ZeroBigInt()
	for _, b := range bigints {
		sum = sum.Add(b.Abs())
	}
	return sum
}

// SortBigInts sorts a slice of BigInts in ascending order.
//
// Takes bigints ([]BigInt) which is the slice to sort in place.
//
// Returns error when any BigInt in the slice has an error or when an
// unexpected comparison error occurs during sorting.
func SortBigInts(bigints []BigInt) error {
	for i, b := range bigints {
		if b.err != nil {
			return fmt.Errorf("maths: cannot sort slice with error at index %d: %w", i, b.err)
		}
	}
	var sortErr error
	slices.SortFunc(bigints, func(a, b BigInt) int {
		if sortErr != nil {
			return 0
		}
		cmpResult, err := a.Cmp(b)
		if err != nil {
			sortErr = fmt.Errorf("maths: unexpected error during sort comparison: %w", err)
			return 0
		}
		return cmpResult
	})
	return sortErr
}

// SortBigIntsReverse sorts a slice of BigInts in descending order.
//
// Takes bigints ([]BigInt) which is the slice to sort in place.
//
// Returns error when any BigInt in the slice has an error or when an
// unexpected comparison error occurs during sorting.
func SortBigIntsReverse(bigints []BigInt) error {
	for i, b := range bigints {
		if b.err != nil {
			return fmt.Errorf("maths: cannot sort slice with error at index %d: %w", i, b.err)
		}
	}
	var sortErr error
	slices.SortFunc(bigints, func(a, b BigInt) int {
		if sortErr != nil {
			return 0
		}
		cmpResult, err := a.Cmp(b)
		if err != nil {
			sortErr = fmt.Errorf("maths: unexpected error during sort comparison: %w", err)
			return 0
		}
		return -cmpResult
	})
	return sortErr
}

// validateAndSumRatios checks that all ratios are non-negative and returns
// their sum.
//
// Takes ratios ([]int64) which contains the ratio values to validate and sum.
//
// Returns int64 which is the sum of all ratios.
// Returns error when any ratio is negative or the sum is zero.
func validateAndSumRatios(ratios []int64) (int64, error) {
	var totalRatio int64
	for _, ratio := range ratios {
		if ratio < 0 {
			return 0, errors.New("maths: cannot allocate to a negative ratio")
		}
		totalRatio += ratio
	}
	if totalRatio == 0 {
		return 0, errors.New("maths: sum of ratios cannot be zero")
	}
	return totalRatio, nil
}

// allocateBigIntPortions distributes a BigInt across the given ratios. The
// last portion receives the remainder to ensure the sum equals the original
// value.
//
// Takes b (BigInt) which is the value to distribute.
// Takes ratios ([]int64) which specifies the proportions for distribution.
// Takes totalRatioBigInt (BigInt) which is the sum of all ratios.
//
// Returns []BigInt which contains the allocated portions for each ratio.
// Returns error when an arithmetic operation fails during allocation.
func allocateBigIntPortions(b BigInt, ratios []int64, totalRatioBigInt BigInt) ([]BigInt, error) {
	results := make([]BigInt, len(ratios))
	allocatedSum := ZeroBigInt()
	lastIndex := len(ratios) - 1

	for i, ratio := range ratios {
		if i == lastIndex {
			remainder := b.Subtract(allocatedSum)
			if remainder.err != nil {
				return nil, remainder.err
			}
			results[i] = remainder
			continue
		}
		ratioBigInt := NewBigIntFromInt(ratio)
		portion := b.Multiply(ratioBigInt).Divide(totalRatioBigInt)
		if portion.err != nil {
			return nil, fmt.Errorf("maths: allocation failed for ratio %d: %w", ratio, portion.err)
		}
		results[i] = portion
		allocatedSum = allocatedSum.Add(portion)
	}
	return results, nil
}
