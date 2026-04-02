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

func TestComparisonMethods(t *testing.T) {
	d5_00 := NewDecimalFromString("5.00")
	d5 := NewDecimalFromInt(5)
	d6 := NewDecimalFromInt(6)
	dInvalid := NewDecimalFromString("abc")

	t.Run("Equals", func(t *testing.T) {
		eq, err := d5.Equals(d5_00)
		if err != nil || !eq {
			t.Errorf("expected 5 to equal 5.00")
		}
		eq, err = d5.Equals(d6)
		if err != nil || eq {
			t.Errorf("expected 5 to not equal 6")
		}
		_, err = dInvalid.Equals(d5)
		if err == nil {
			t.Error("expected error on comparison with invalid decimal")
		}
		_, err = d5.Equals(dInvalid)
		if err == nil {
			t.Error("expected error on comparison with invalid decimal")
		}
	})

	t.Run("LessThan", func(t *testing.T) {
		lt, err := d5.LessThan(d6)
		if err != nil || !lt {
			t.Errorf("expected 5 to be less than 6")
		}
		lt, err = d6.LessThan(d5)
		if err != nil || lt {
			t.Errorf("expected 6 not to be less than 5")
		}
		lt, err = d5.LessThan(d5_00)
		if err != nil || lt {
			t.Errorf("expected 5 not to be less than 5.00")
		}
	})

	t.Run("GreaterThan", func(t *testing.T) {
		gt, err := d6.GreaterThan(d5)
		if err != nil || !gt {
			t.Errorf("expected 6 to be greater than 5")
		}
		gt, err = d5.GreaterThan(d6)
		if err != nil || gt {
			t.Errorf("expected 5 not to be greater than 6")
		}
		gt, err = d5.GreaterThan(d5_00)
		if err != nil || gt {
			t.Errorf("expected 5 not to be greater than 5.00")
		}
	})
}

func TestPropertyChecks(t *testing.T) {
	d5 := NewDecimalFromInt(5)
	dNeg1 := NewDecimalFromInt(-1)
	dNonInt := NewDecimalFromString("123.45")
	dInt := NewDecimalFromString("123.0")
	d4 := NewDecimalFromInt(4)
	dInvalid := NewDecimalFromString("abc")

	t.Run("IsZero", func(t *testing.T) {
		is, err := ZeroDecimal().IsZero()
		if err != nil || !is {
			t.Error("expected IsZero to be true for 0")
		}
		is, _ = d5.IsZero()
		if is {
			t.Error("expected IsZero to be false for 5")
		}
		_, err = dInvalid.IsZero()
		if err == nil {
			t.Error("expected error for IsZero on invalid decimal")
		}
	})

	t.Run("IsPositive", func(t *testing.T) {
		is, err := d5.IsPositive()
		if err != nil || !is {
			t.Error("expected IsPositive to be true for 5")
		}
		is, _ = dNeg1.IsPositive()
		if is {
			t.Error("expected IsPositive to be false for -1")
		}
		is, _ = ZeroDecimal().IsPositive()
		if is {
			t.Error("expected IsPositive to be false for 0")
		}
	})

	t.Run("IsNegative", func(t *testing.T) {
		is, err := dNeg1.IsNegative()
		if err != nil || !is {
			t.Error("expected IsNegative to be true for -1")
		}
		is, _ = d5.IsNegative()
		if is {
			t.Error("expected IsNegative to be false for 5")
		}
		is, _ = ZeroDecimal().IsNegative()
		if is {
			t.Error("expected IsNegative to be false for 0")
		}
	})

	t.Run("IsInteger", func(t *testing.T) {
		is, err := dInt.IsInteger()
		if err != nil || !is {
			t.Error("expected IsInteger to be true for 123.0")
		}
		is, _ = dNonInt.IsInteger()
		if is {
			t.Error("expected IsInteger to be false for 123.45")
		}
	})

	t.Run("IsEven/IsOdd", func(t *testing.T) {
		is, err := d4.IsEven()
		if err != nil || !is {
			t.Error("expected 4 to be even")
		}
		is, _ = d5.IsEven()
		if is {
			t.Error("expected 5 not to be even")
		}
		is, err = d5.IsOdd()
		if err != nil || !is {
			t.Error("expected 5 to be odd")
		}
		is, _ = d4.IsOdd()
		if is {
			t.Error("expected 4 not to be odd")
		}
		is, _ = dNonInt.IsEven()
		if is {
			t.Error("expected non-integer to not be even")
		}
		is, _ = ZeroDecimal().IsEven()
		if !is {
			t.Error("expected 0 to be even")
		}
		is, _ = ZeroDecimal().IsOdd()
		if is {
			t.Error("expected 0 not to be odd")
		}
	})

	t.Run("IsMultipleOf", func(t *testing.T) {
		is, err := NewDecimalFromInt(10).IsMultipleOf(NewDecimalFromInt(2))
		if err != nil || !is {
			t.Error("expected 10 to be a multiple of 2")
		}
		is, _ = NewDecimalFromInt(10).IsMultipleOf(NewDecimalFromInt(3))
		if is {
			t.Error("expected 10 not to be a multiple of 3")
		}
		is, _ = ZeroDecimal().IsMultipleOf(d5)
		if !is {
			t.Error("expected 0 to be a multiple of 5")
		}
		is, _ = d5.IsMultipleOf(ZeroDecimal())
		if is {
			t.Error("expected 5 not to be a multiple of 0")
		}
	})

	t.Run("IsBetween", func(t *testing.T) {
		minVal, maxVal := NewDecimalFromInt(0), NewDecimalFromInt(10)
		is, err := d5.IsBetween(minVal, maxVal)
		if err != nil || !is {
			t.Error("expected 5 to be between 0 and 10")
		}
		is, _ = NewDecimalFromInt(11).IsBetween(minVal, maxVal)
		if is {
			t.Error("expected 11 not to be between 0 and 10")
		}
		is, _ = minVal.IsBetween(minVal, maxVal)
		if !is {
			t.Error("expected 0 to be between 0 and 10 (inclusive)")
		}
		_, err = d5.IsBetween(maxVal, minVal)
		if err == nil {
			t.Error("expected error when min > max for IsBetween")
		}
	})

	t.Run("IsCloseTo", func(t *testing.T) {
		target := NewDecimalFromString("4.99")
		tolerance := NewDecimalFromString("0.02")
		is, err := d5.IsCloseTo(target, tolerance)
		if err != nil || !is {
			t.Error("expected 5 to be close to 4.99 with tolerance 0.02")
		}
		is, _ = d5.IsCloseTo(target, NewDecimalFromString("0.001"))
		if is {
			t.Error("expected 5 not to be close to 4.99 with tolerance 0.001")
		}
		_, err = d5.IsCloseTo(target, NewDecimalFromInt(-1))
		if err == nil {
			t.Error("expected error for negative tolerance")
		}
	})
}

func TestCheckMethodsSystematically(t *testing.T) {
	dPos := NewDecimalFromInt(10)
	dNeg := NewDecimalFromInt(-10)
	dZero := ZeroDecimal()
	dEven := NewDecimalFromInt(4)
	dOdd := NewDecimalFromInt(5)
	dNonInt := NewDecimalFromString("4.5")
	dInvalid := NewDecimalFromString("invalid")

	testCases := []struct {
		name     string
		check    bool
		expected bool
	}{

		{name: "CheckIsZero on zero", check: dZero.CheckIsZero(), expected: true},
		{name: "CheckIsZero on non-zero", check: dPos.CheckIsZero(), expected: false},
		{name: "CheckIsZero on invalid", check: dInvalid.CheckIsZero(), expected: false},

		{name: "CheckIsPositive on positive", check: dPos.CheckIsPositive(), expected: true},
		{name: "CheckIsPositive on negative", check: dNeg.CheckIsPositive(), expected: false},
		{name: "CheckIsPositive on zero", check: dZero.CheckIsPositive(), expected: false},
		{name: "CheckIsPositive on invalid", check: dInvalid.CheckIsPositive(), expected: false},

		{name: "CheckIsNegative on negative", check: dNeg.CheckIsNegative(), expected: true},
		{name: "CheckIsNegative on positive", check: dPos.CheckIsNegative(), expected: false},
		{name: "CheckIsNegative on zero", check: dZero.CheckIsNegative(), expected: false},
		{name: "CheckIsNegative on invalid", check: dInvalid.CheckIsNegative(), expected: false},

		{name: "CheckIsInteger on integer", check: dPos.CheckIsInteger(), expected: true},
		{name: "CheckIsInteger on non-integer", check: dNonInt.CheckIsInteger(), expected: false},
		{name: "CheckIsInteger on invalid", check: dInvalid.CheckIsInteger(), expected: false},

		{name: "CheckEquals on equal", check: dPos.CheckEquals(NewDecimalFromInt(10)), expected: true},
		{name: "CheckEquals on unequal", check: dPos.CheckEquals(dOdd), expected: false},
		{name: "CheckEquals on invalid", check: dInvalid.CheckEquals(dPos), expected: false},

		{name: "CheckLessThan on lesser", check: dOdd.CheckLessThan(dPos), expected: true},
		{name: "CheckLessThan on greater", check: dPos.CheckLessThan(dOdd), expected: false},
		{name: "CheckLessThan on invalid", check: dInvalid.CheckLessThan(dPos), expected: false},

		{name: "CheckGreaterThan on greater", check: dPos.CheckGreaterThan(dOdd), expected: true},
		{name: "CheckGreaterThan on lesser", check: dOdd.CheckGreaterThan(dPos), expected: false},
		{name: "CheckGreaterThan on invalid", check: dInvalid.CheckGreaterThan(dPos), expected: false},

		{name: "CheckIsBetween when inside", check: dOdd.CheckIsBetween(dEven, dPos), expected: true},
		{name: "CheckIsBetween when outside", check: dNeg.CheckIsBetween(dEven, dPos), expected: false},
		{name: "CheckIsBetween on invalid", check: dInvalid.CheckIsBetween(dEven, dPos), expected: false},

		{name: "CheckIsCloseTo when close", check: dEven.CheckIsCloseTo(dOdd, OneDecimal()), expected: true},
		{name: "CheckIsCloseTo when not close", check: dEven.CheckIsCloseTo(dPos, OneDecimal()), expected: false},
		{name: "CheckIsCloseTo on invalid", check: dInvalid.CheckIsCloseTo(dPos, OneDecimal()), expected: false},

		{name: "CheckIsEven on even", check: dEven.CheckIsEven(), expected: true},
		{name: "CheckIsEven on odd", check: dOdd.CheckIsEven(), expected: false},
		{name: "CheckIsEven on non-integer", check: dNonInt.CheckIsEven(), expected: false},
		{name: "CheckIsEven on invalid", check: dInvalid.CheckIsEven(), expected: false},

		{name: "CheckIsOdd on odd", check: dOdd.CheckIsOdd(), expected: true},
		{name: "CheckIsOdd on even", check: dEven.CheckIsOdd(), expected: false},
		{name: "CheckIsOdd on non-integer", check: dNonInt.CheckIsOdd(), expected: false},
		{name: "CheckIsOdd on invalid", check: dInvalid.CheckIsOdd(), expected: false},

		{name: "CheckIsMultipleOf when true", check: dPos.CheckIsMultipleOf(dOdd), expected: true},
		{name: "CheckIsMultipleOf when false", check: dPos.CheckIsMultipleOf(NewDecimalFromInt(3)), expected: false},
		{name: "CheckIsMultipleOf on invalid", check: dInvalid.CheckIsMultipleOf(dPos), expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check; got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestMustMethodsSystematically(t *testing.T) {
	dValid := NewDecimalFromInt(5)
	dInvalid := NewDecimalFromString("abc")

	t.Run("Must methods should not panic on valid decimal", func(t *testing.T) {
		_ = dValid.MustIsZero()
		_ = dValid.MustIsPositive()
		_ = dValid.MustIsNegative()
		_ = dValid.MustIsInteger()
		_ = dValid.MustIsEven()
		_ = dValid.MustIsOdd()
		_ = dValid.MustEquals(NewDecimalFromInt(5))
		_ = dValid.MustLessThan(NewDecimalFromInt(10))
		_ = dValid.MustGreaterThan(NewDecimalFromInt(0))
		_ = dValid.MustIsBetween(ZeroDecimal(), TenDecimal())
		_ = dValid.MustIsCloseTo(NewDecimalFromInt(5), OneDecimal())
		_ = dValid.MustIsMultipleOf(OneDecimal())
	})

	testCases := []struct {
		name         string
		mustCall     func()
		panicMessage string
	}{
		{name: "MustIsZero", mustCall: func() { _ = dInvalid.MustIsZero() }, panicMessage: "MustIsZero"},
		{name: "MustIsPositive", mustCall: func() { _ = dInvalid.MustIsPositive() }, panicMessage: "MustIsPositive"},
		{name: "MustIsNegative", mustCall: func() { _ = dInvalid.MustIsNegative() }, panicMessage: "MustIsNegative"},
		{name: "MustIsInteger", mustCall: func() { _ = dInvalid.MustIsInteger() }, panicMessage: "MustIsInteger"},
		{name: "MustEquals", mustCall: func() { _ = dInvalid.MustEquals(dValid) }, panicMessage: "MustEquals"},
		{name: "MustLessThan", mustCall: func() { _ = dInvalid.MustLessThan(dValid) }, panicMessage: "MustLessThan"},
		{name: "MustGreaterThan", mustCall: func() { _ = dInvalid.MustGreaterThan(dValid) }, panicMessage: "MustGreaterThan"},
		{name: "MustIsBetween", mustCall: func() { _ = dInvalid.MustIsBetween(ZeroDecimal(), TenDecimal()) }, panicMessage: "MustIsBetween"},
		{name: "MustIsCloseTo", mustCall: func() { _ = dInvalid.MustIsCloseTo(ZeroDecimal(), OneDecimal()) }, panicMessage: "MustIsCloseTo"},
		{name: "MustIsEven", mustCall: func() { _ = dInvalid.MustIsEven() }, panicMessage: "MustIsEven"},
		{name: "MustIsOdd", mustCall: func() { _ = dInvalid.MustIsOdd() }, panicMessage: "MustIsOdd"},
		{name: "MustIsMultipleOf", mustCall: func() { _ = dInvalid.MustIsMultipleOf(OneDecimal()) }, panicMessage: "MustIsMultipleOf"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s should panic on invalid decimal", tc.name), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected %s to panic, but it did not", tc.panicMessage)
				}
			}()
			tc.mustCall()
		})
	}
}

func TestDecimalLogicVariants(t *testing.T) {
	t.Parallel()
	d5 := NewDecimalFromInt(5)
	dInvalid := NewDecimalFromString("invalid")

	t.Run("EqualsInt", func(t *testing.T) {
		t.Parallel()
		eq, err := d5.EqualsInt(5)
		if err != nil || !eq {
			t.Error("expected 5 to equal int 5")
		}
		eq, err = d5.EqualsInt(6)
		if err != nil || eq {
			t.Error("expected 5 not to equal int 6")
		}
		_, err = dInvalid.EqualsInt(5)
		if err == nil {
			t.Error("expected error on invalid decimal")
		}
	})

	t.Run("EqualsString", func(t *testing.T) {
		t.Parallel()
		eq, err := d5.EqualsString("5")
		if err != nil || !eq {
			t.Error("expected 5 to equal string '5'")
		}
		eq, err = d5.EqualsString("6")
		if err != nil || eq {
			t.Error("expected 5 not to equal string '6'")
		}
	})

	t.Run("EqualsFloat", func(t *testing.T) {
		t.Parallel()
		eq, err := d5.EqualsFloat(5.0)
		if err != nil || !eq {
			t.Error("expected 5 to equal float 5.0")
		}
		eq, err = d5.EqualsFloat(6.0)
		if err != nil || eq {
			t.Error("expected 5 not to equal float 6.0")
		}
	})

	t.Run("LessThanInt", func(t *testing.T) {
		t.Parallel()
		lt, err := d5.LessThanInt(10)
		if err != nil || !lt {
			t.Error("expected 5 to be less than 10")
		}
		lt, err = d5.LessThanInt(3)
		if err != nil || lt {
			t.Error("expected 5 not to be less than 3")
		}
	})

	t.Run("LessThanString", func(t *testing.T) {
		t.Parallel()
		lt, err := d5.LessThanString("10")
		if err != nil || !lt {
			t.Error("expected 5 to be less than '10'")
		}
		lt, err = d5.LessThanString("3")
		if err != nil || lt {
			t.Error("expected 5 not to be less than '3'")
		}
	})

	t.Run("LessThanFloat", func(t *testing.T) {
		t.Parallel()
		lt, err := d5.LessThanFloat(10.0)
		if err != nil || !lt {
			t.Error("expected 5 to be less than 10.0")
		}
		lt, err = d5.LessThanFloat(3.0)
		if err != nil || lt {
			t.Error("expected 5 not to be less than 3.0")
		}
	})

	t.Run("GreaterThanInt", func(t *testing.T) {
		t.Parallel()
		gt, err := d5.GreaterThanInt(3)
		if err != nil || !gt {
			t.Error("expected 5 to be greater than 3")
		}
		gt, err = d5.GreaterThanInt(10)
		if err != nil || gt {
			t.Error("expected 5 not to be greater than 10")
		}
	})

	t.Run("GreaterThanString", func(t *testing.T) {
		t.Parallel()
		gt, err := d5.GreaterThanString("3")
		if err != nil || !gt {
			t.Error("expected 5 to be greater than '3'")
		}
		gt, err = d5.GreaterThanString("10")
		if err != nil || gt {
			t.Error("expected 5 not to be greater than '10'")
		}
	})

	t.Run("GreaterThanFloat", func(t *testing.T) {
		t.Parallel()
		gt, err := d5.GreaterThanFloat(3.0)
		if err != nil || !gt {
			t.Error("expected 5 to be greater than 3.0")
		}
		gt, err = d5.GreaterThanFloat(10.0)
		if err != nil || gt {
			t.Error("expected 5 not to be greater than 10.0")
		}
	})
}
