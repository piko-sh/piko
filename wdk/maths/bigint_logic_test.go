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
	"fmt"
	"testing"
)

func TestBigIntComparisonMethods(t *testing.T) {
	b5_int := NewBigIntFromInt(5)
	b5_str := NewBigIntFromString("5")
	b6 := NewBigIntFromInt(6)
	bInvalid := NewBigIntFromString("abc")

	t.Run("Equals", func(t *testing.T) {
		eq, err := b5_int.Equals(b5_str)
		if err != nil || !eq {
			t.Errorf("expected 5 to equal '5'")
		}
		eq, err = b5_int.Equals(b6)
		if err != nil || eq {
			t.Errorf("expected 5 to not equal 6")
		}
		_, err = bInvalid.Equals(b5_int)
		if err == nil {
			t.Error("expected error on comparison with invalid bigint")
		}
		_, err = b5_int.Equals(bInvalid)
		if err == nil {
			t.Error("expected error on comparison with invalid bigint")
		}
	})

	t.Run("LessThan", func(t *testing.T) {
		lt, err := b5_int.LessThan(b6)
		if err != nil || !lt {
			t.Errorf("expected 5 to be less than 6")
		}
		lt, err = b6.LessThan(b5_int)
		if err != nil || lt {
			t.Errorf("expected 6 not to be less than 5")
		}
		lt, err = b5_int.LessThan(b5_str)
		if err != nil || lt {
			t.Errorf("expected 5 not to be less than '5'")
		}
	})

	t.Run("GreaterThan", func(t *testing.T) {
		gt, err := b6.GreaterThan(b5_int)
		if err != nil || !gt {
			t.Errorf("expected 6 to be greater than 5")
		}
		gt, err = b5_int.GreaterThan(b6)
		if err != nil || gt {
			t.Errorf("expected 5 not to be greater than 6")
		}
		gt, err = b5_int.GreaterThan(b5_str)
		if err != nil || gt {
			t.Errorf("expected 5 not to be greater than '5'")
		}
	})
}

func TestBigIntPropertyChecks(t *testing.T) {
	b5 := NewBigIntFromInt(5)
	bNeg1 := NewBigIntFromInt(-1)
	b4 := NewBigIntFromInt(4)
	bInvalid := NewBigIntFromString("abc")

	t.Run("IsZero", func(t *testing.T) {
		is, err := ZeroBigInt().IsZero()
		if err != nil || !is {
			t.Error("expected IsZero to be true for 0")
		}
		is, _ = b5.IsZero()
		if is {
			t.Error("expected IsZero to be false for 5")
		}
		_, err = bInvalid.IsZero()
		if err == nil {
			t.Error("expected error for IsZero on invalid bigint")
		}
	})

	t.Run("IsPositive/IsNegative", func(t *testing.T) {
		is, err := b5.IsPositive()
		if err != nil || !is {
			t.Error("expected IsPositive to be true for 5")
		}
		is, _ = bNeg1.IsPositive()
		if is {
			t.Error("expected IsPositive to be false for -1")
		}
		is, err = bNeg1.IsNegative()
		if err != nil || !is {
			t.Error("expected IsNegative to be true for -1")
		}
		is, _ = ZeroBigInt().IsPositive()
		if is {
			t.Error("expected IsPositive to be false for 0")
		}
	})

	t.Run("IsInteger", func(t *testing.T) {
		is, err := b5.IsInteger()
		if err != nil || !is {
			t.Error("expected IsInteger to always be true for a valid BigInt")
		}
	})

	t.Run("IsEven/IsOdd", func(t *testing.T) {
		is, err := b4.IsEven()
		if err != nil || !is {
			t.Error("expected 4 to be even")
		}
		is, _ = b5.IsEven()
		if is {
			t.Error("expected 5 not to be even")
		}
		is, err = b5.IsOdd()
		if err != nil || !is {
			t.Error("expected 5 to be odd")
		}
		is, _ = ZeroBigInt().IsEven()
		if !is {
			t.Error("expected 0 to be even")
		}
	})

	t.Run("IsMultipleOf", func(t *testing.T) {
		is, err := NewBigIntFromInt(10).IsMultipleOf(NewBigIntFromInt(2))
		if err != nil || !is {
			t.Error("expected 10 to be a multiple of 2")
		}
		is, _ = NewBigIntFromInt(10).IsMultipleOf(NewBigIntFromInt(3))
		if is {
			t.Error("expected 10 not to be a multiple of 3")
		}
		is, _ = ZeroBigInt().IsMultipleOf(b5)
		if !is {
			t.Error("expected 0 to be a multiple of 5")
		}
		is, _ = b5.IsMultipleOf(ZeroBigInt())
		if is {
			t.Error("expected 5 not to be a multiple of 0")
		}
	})

	t.Run("IsBetween", func(t *testing.T) {
		minVal, maxVal := NewBigIntFromInt(0), NewBigIntFromInt(10)
		is, err := b5.IsBetween(minVal, maxVal)
		if err != nil || !is {
			t.Error("expected 5 to be between 0 and 10")
		}
		is, _ = NewBigIntFromInt(11).IsBetween(minVal, maxVal)
		if is {
			t.Error("expected 11 not to be between 0 and 10")
		}
		is, _ = minVal.IsBetween(minVal, maxVal)
		if !is {
			t.Error("expected 0 to be between 0 and 10 (inclusive)")
		}
		_, err = b5.IsBetween(maxVal, minVal)
		if err == nil {
			t.Error("expected error when min > max for IsBetween")
		}
	})
}

func TestBigIntCheckMethodsSystematically(t *testing.T) {
	bPos := NewBigIntFromInt(10)
	bNeg := NewBigIntFromInt(-10)
	bZero := ZeroBigInt()
	bEven := NewBigIntFromInt(4)
	bOdd := NewBigIntFromInt(5)
	bInvalid := NewBigIntFromString("invalid")

	testCases := []struct {
		name     string
		check    bool
		expected bool
	}{
		{name: "CheckIsZero on zero", check: bZero.CheckIsZero(), expected: true},
		{name: "CheckIsPositive on positive", check: bPos.CheckIsPositive(), expected: true},
		{name: "CheckIsNegative on negative", check: bNeg.CheckIsNegative(), expected: true},
		{name: "CheckIsInteger on integer", check: bPos.CheckIsInteger(), expected: true},
		{name: "CheckEquals on equal", check: bPos.CheckEquals(NewBigIntFromInt(10)), expected: true},
		{name: "CheckLessThan on lesser", check: bOdd.CheckLessThan(bPos), expected: true},
		{name: "CheckGreaterThan on greater", check: bPos.CheckGreaterThan(bOdd), expected: true},
		{name: "CheckIsBetween when inside", check: bOdd.CheckIsBetween(bEven, bPos), expected: true},
		{name: "CheckIsEven on even", check: bEven.CheckIsEven(), expected: true},
		{name: "CheckIsOdd on odd", check: bOdd.CheckIsOdd(), expected: true},
		{name: "CheckIsMultipleOf when true", check: bPos.CheckIsMultipleOf(bOdd), expected: true},
		{name: "CheckIsMultipleOf when false", check: bPos.CheckIsMultipleOf(NewBigIntFromInt(3)), expected: false},
		{name: "CheckIsMultipleOf on invalid", check: bInvalid.CheckIsMultipleOf(bPos), expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check; got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestBigIntMustMethodsSystematically(t *testing.T) {
	bValid := NewBigIntFromInt(5)
	bInvalid := NewBigIntFromString("abc")

	t.Run("Must methods should not panic on valid bigint", func(t *testing.T) {
		_ = bValid.MustIsZero()
		_ = bValid.MustIsPositive()
		_ = bValid.MustIsNegative()
		_ = bValid.MustIsInteger()
		_ = bValid.MustIsEven()
		_ = bValid.MustIsOdd()
		_ = bValid.MustEquals(NewBigIntFromInt(5))
		_ = bValid.MustLessThan(NewBigIntFromInt(10))
		_ = bValid.MustGreaterThan(NewBigIntFromInt(0))
		_ = bValid.MustIsBetween(ZeroBigInt(), TenBigInt())
		_ = bValid.MustIsMultipleOf(OneBigInt())
	})

	testCases := []struct {
		mustCall func()
		name     string
	}{
		{mustCall: func() { _ = bInvalid.MustIsZero() }, name: "MustIsZero"},
		{mustCall: func() { _ = bInvalid.MustIsPositive() }, name: "MustIsPositive"},
		{mustCall: func() { _ = bInvalid.MustIsNegative() }, name: "MustIsNegative"},
		{mustCall: func() { _ = bInvalid.MustIsInteger() }, name: "MustIsInteger"},
		{mustCall: func() { _ = bInvalid.MustEquals(bValid) }, name: "MustEquals"},
		{mustCall: func() { _ = bInvalid.MustLessThan(bValid) }, name: "MustLessThan"},
		{mustCall: func() { _ = bInvalid.MustGreaterThan(bValid) }, name: "MustGreaterThan"},
		{mustCall: func() { _ = bInvalid.MustIsBetween(ZeroBigInt(), TenBigInt()) }, name: "MustIsBetween"},
		{mustCall: func() { _ = bInvalid.MustIsEven() }, name: "MustIsEven"},
		{mustCall: func() { _ = bInvalid.MustIsOdd() }, name: "MustIsOdd"},
		{mustCall: func() { _ = bInvalid.MustIsMultipleOf(OneBigInt()) }, name: "MustIsMultipleOf"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s should panic on invalid bigint", tc.name), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected %s to panic, but it did not", tc.name)
				}
			}()
			tc.mustCall()
		})
	}
}

func TestBigIntLogicVariants(t *testing.T) {
	t.Parallel()
	b5 := NewBigIntFromInt(5)
	bInvalid := NewBigIntFromString("invalid")

	t.Run("EqualsInt", func(t *testing.T) {
		t.Parallel()
		eq, err := b5.EqualsInt(5)
		if err != nil || !eq {
			t.Error("expected 5 to equal int 5")
		}
		eq, err = b5.EqualsInt(6)
		if err != nil || eq {
			t.Error("expected 5 not to equal int 6")
		}
		_, err = bInvalid.EqualsInt(5)
		if err == nil {
			t.Error("expected error on invalid bigint")
		}
	})

	t.Run("EqualsString", func(t *testing.T) {
		t.Parallel()
		eq, err := b5.EqualsString("5")
		if err != nil || !eq {
			t.Error("expected 5 to equal string '5'")
		}
		eq, err = b5.EqualsString("6")
		if err != nil || eq {
			t.Error("expected 5 not to equal string '6'")
		}
		_, err = bInvalid.EqualsString("5")
		if err == nil {
			t.Error("expected error on invalid bigint")
		}
	})

	t.Run("LessThanInt", func(t *testing.T) {
		t.Parallel()
		lt, err := b5.LessThanInt(10)
		if err != nil || !lt {
			t.Error("expected 5 to be less than 10")
		}
		lt, err = b5.LessThanInt(3)
		if err != nil || lt {
			t.Error("expected 5 not to be less than 3")
		}
		_, err = bInvalid.LessThanInt(5)
		if err == nil {
			t.Error("expected error on invalid bigint")
		}
	})

	t.Run("LessThanString", func(t *testing.T) {
		t.Parallel()
		lt, err := b5.LessThanString("10")
		if err != nil || !lt {
			t.Error("expected 5 to be less than '10'")
		}
		lt, err = b5.LessThanString("3")
		if err != nil || lt {
			t.Error("expected 5 not to be less than '3'")
		}
	})

	t.Run("GreaterThanInt", func(t *testing.T) {
		t.Parallel()
		gt, err := b5.GreaterThanInt(3)
		if err != nil || !gt {
			t.Error("expected 5 to be greater than 3")
		}
		gt, err = b5.GreaterThanInt(10)
		if err != nil || gt {
			t.Error("expected 5 not to be greater than 10")
		}
		_, err = bInvalid.GreaterThanInt(5)
		if err == nil {
			t.Error("expected error on invalid bigint")
		}
	})

	t.Run("GreaterThanString", func(t *testing.T) {
		t.Parallel()
		gt, err := b5.GreaterThanString("3")
		if err != nil || !gt {
			t.Error("expected 5 to be greater than '3'")
		}
		gt, err = b5.GreaterThanString("10")
		if err != nil || gt {
			t.Error("expected 5 not to be greater than '10'")
		}
	})
}
