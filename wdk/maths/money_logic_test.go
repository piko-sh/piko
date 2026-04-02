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

func TestMoneyComparisonMethods(t *testing.T) {
	m5_00USD := NewMoneyFromString("5.00", "USD")
	m5USD := NewMoneyFromInt(5, "USD")
	m6USD := NewMoneyFromInt(6, "USD")
	m5EUR := NewMoneyFromInt(5, "EUR")
	mInvalid := NewMoneyFromString("abc", "USD")

	t.Run("Equals", func(t *testing.T) {

		eq, err := m5USD.Equals(m5_00USD)
		if err != nil || !eq {
			t.Errorf("expected 5 USD to equal 5.00 USD")
		}
		eq, err = m5USD.Equals(m6USD)
		if err != nil || eq {
			t.Errorf("expected 5 USD to not equal 6 USD")
		}

		eq, err = m5USD.Equals(m5EUR)
		if err != nil || eq {
			t.Errorf("expected 5 USD to not equal 5 EUR")
		}

		_, err = mInvalid.Equals(m5USD)
		if err == nil {
			t.Error("expected error on comparison with invalid decimal")
		}
	})

	t.Run("LessThan", func(t *testing.T) {
		lt, err := m5USD.LessThan(m6USD)
		if err != nil || !lt {
			t.Errorf("expected 5 USD to be less than 6 USD")
		}
		lt, err = m6USD.LessThan(m5USD)
		if err != nil || lt {
			t.Errorf("expected 6 USD not to be less than 5 USD")
		}

		_, err = m5USD.LessThan(m5EUR)
		if err == nil {
			t.Error("expected LessThan to error on currency mismatch")
		}
	})

	t.Run("GreaterThan", func(t *testing.T) {
		gt, err := m6USD.GreaterThan(m5USD)
		if err != nil || !gt {
			t.Errorf("expected 6 USD to be greater than 5 USD")
		}
		gt, err = m5USD.GreaterThan(m6USD)
		if err != nil || gt {
			t.Errorf("expected 5 USD not to be greater than 6 USD")
		}

		_, err = m5USD.GreaterThan(m5EUR)
		if err == nil {
			t.Error("expected GreaterThan to error on currency mismatch")
		}
	})
}

func TestMoneyPropertyChecks(t *testing.T) {
	mPos := NewMoneyFromInt(5, "USD")
	mNeg := NewMoneyFromInt(-1, "USD")
	mZero := ZeroMoney("USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("IsZero", func(t *testing.T) {
		is, err := mZero.IsZero()
		if err != nil || !is {
			t.Error("expected IsZero to be true for 0")
		}
		is, _ = mPos.IsZero()
		if is {
			t.Error("expected IsZero to be false for 5")
		}
		_, err = mInvalid.IsZero()
		if err == nil {
			t.Error("expected error for IsZero on invalid money")
		}
	})

	t.Run("IsPositive", func(t *testing.T) {
		is, err := mPos.IsPositive()
		if err != nil || !is {
			t.Error("expected IsPositive to be true for 5")
		}
		is, _ = mNeg.IsPositive()
		if is {
			t.Error("expected IsPositive to be false for -1")
		}
		is, _ = mZero.IsPositive()
		if is {
			t.Error("expected IsPositive to be false for 0")
		}
	})

	t.Run("IsNegative", func(t *testing.T) {
		is, err := mNeg.IsNegative()
		if err != nil || !is {
			t.Error("expected IsNegative to be true for -1")
		}
		is, _ = mPos.IsNegative()
		if is {
			t.Error("expected IsNegative to be false for 5")
		}
		is, _ = mZero.IsNegative()
		if is {
			t.Error("expected IsNegative to be false for 0")
		}
	})
}

func TestMoneyCheckMethodsSystematically(t *testing.T) {
	mPos := NewMoneyFromInt(10, "USD")
	mNeg := NewMoneyFromInt(-10, "USD")
	mZero := ZeroMoney("USD")
	mOther := NewMoneyFromInt(5, "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	testCases := []struct {
		name     string
		check    bool
		expected bool
	}{

		{name: "CheckIsZero on zero", check: mZero.CheckIsZero(), expected: true},
		{name: "CheckIsZero on non-zero", check: mPos.CheckIsZero(), expected: false},
		{name: "CheckIsZero on invalid", check: mInvalid.CheckIsZero(), expected: false},

		{name: "CheckIsPositive on positive", check: mPos.CheckIsPositive(), expected: true},
		{name: "CheckIsPositive on negative", check: mNeg.CheckIsPositive(), expected: false},
		{name: "CheckIsPositive on zero", check: mZero.CheckIsPositive(), expected: false},
		{name: "CheckIsPositive on invalid", check: mInvalid.CheckIsPositive(), expected: false},

		{name: "CheckIsNegative on negative", check: mNeg.CheckIsNegative(), expected: true},
		{name: "CheckIsNegative on positive", check: mPos.CheckIsNegative(), expected: false},
		{name: "CheckIsNegative on zero", check: mZero.CheckIsNegative(), expected: false},
		{name: "CheckIsNegative on invalid", check: mInvalid.CheckIsNegative(), expected: false},

		{name: "CheckEquals on equal", check: mPos.CheckEquals(NewMoneyFromInt(10, "USD")), expected: true},
		{name: "CheckEquals on unequal", check: mPos.CheckEquals(mOther), expected: false},
		{name: "CheckEquals on currency mismatch", check: mPos.CheckEquals(NewMoneyFromInt(10, "EUR")), expected: false},
		{name: "CheckEquals on invalid", check: mInvalid.CheckEquals(mPos), expected: false},

		{name: "CheckLessThan on lesser", check: mOther.CheckLessThan(mPos), expected: true},
		{name: "CheckLessThan on greater", check: mPos.CheckLessThan(mOther), expected: false},
		{name: "CheckLessThan on invalid", check: mInvalid.CheckLessThan(mPos), expected: false},

		{name: "CheckGreaterThan on greater", check: mPos.CheckGreaterThan(mOther), expected: true},
		{name: "CheckGreaterThan on lesser", check: mOther.CheckGreaterThan(mPos), expected: false},
		{name: "CheckGreaterThan on invalid", check: mInvalid.CheckGreaterThan(mPos), expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check; got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestMoneyMustMethodsSystematically(t *testing.T) {
	mValid := NewMoneyFromInt(5, "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")
	mEUR := NewMoneyFromInt(5, "EUR")

	t.Run("Must methods should not panic on valid inputs", func(t *testing.T) {
		_ = mValid.MustIsZero()
		_ = mValid.MustIsPositive()
		_ = mValid.MustIsNegative()
		_ = mValid.MustEquals(NewMoneyFromInt(5, "USD"))
		_ = mValid.MustLessThan(NewMoneyFromInt(10, "USD"))
		_ = mValid.MustGreaterThan(NewMoneyFromInt(0, "USD"))
	})

	t.Run("MustEquals should not panic on currency mismatch", func(t *testing.T) {

		if mValid.MustEquals(mEUR) {
			t.Error("MustEquals should return false for currency mismatch, not panic")
		}
	})

	testCases := []struct {
		name         string
		mustCall     func()
		panicMessage string
	}{
		{name: "MustIsZero on invalid", mustCall: func() { _ = mInvalid.MustIsZero() }, panicMessage: "MustIsZero"},
		{name: "MustIsPositive on invalid", mustCall: func() { _ = mInvalid.MustIsPositive() }, panicMessage: "MustIsPositive"},
		{name: "MustIsNegative on invalid", mustCall: func() { _ = mInvalid.MustIsNegative() }, panicMessage: "MustIsNegative"},
		{name: "MustEquals on invalid", mustCall: func() { _ = mInvalid.MustEquals(mValid) }, panicMessage: "MustEquals"},
		{name: "MustLessThan on invalid", mustCall: func() { _ = mInvalid.MustLessThan(mValid) }, panicMessage: "MustLessThan"},
		{name: "MustGreaterThan on invalid", mustCall: func() { _ = mInvalid.MustGreaterThan(mValid) }, panicMessage: "MustGreaterThan"},
		{name: "MustLessThan on currency mismatch", mustCall: func() { _ = mValid.MustLessThan(mEUR) }, panicMessage: "MustLessThan"},
		{name: "MustGreaterThan on currency mismatch", mustCall: func() { _ = mValid.MustGreaterThan(mEUR) }, panicMessage: "MustGreaterThan"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s should panic", tc.name), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected %s to panic, but it did not", tc.panicMessage)
				}
			}()
			tc.mustCall()
		})
	}
}

func TestMoneyComparisonVariants(t *testing.T) {
	t.Parallel()
	m5USD := NewMoneyFromInt(5, "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("EqualsInt", func(t *testing.T) {
		t.Parallel()
		eq, err := m5USD.EqualsInt(5)
		if err != nil || !eq {
			t.Error("expected 5 USD to equal int 5")
		}
		eq, err = m5USD.EqualsInt(6)
		if err != nil || eq {
			t.Error("expected 5 USD not to equal int 6")
		}
		_, err = mInvalid.EqualsInt(5)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("EqualsString", func(t *testing.T) {
		t.Parallel()
		eq, err := m5USD.EqualsString("5")
		if err != nil || !eq {
			t.Error("expected 5 USD to equal string '5'")
		}
		eq, err = m5USD.EqualsString("6")
		if err != nil || eq {
			t.Error("expected 5 USD not to equal string '6'")
		}
		_, err = mInvalid.EqualsString("5")
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("EqualsFloat", func(t *testing.T) {
		t.Parallel()
		eq, err := m5USD.EqualsFloat(5.0)
		if err != nil || !eq {
			t.Error("expected 5 USD to equal float 5.0")
		}
		eq, err = m5USD.EqualsFloat(6.0)
		if err != nil || eq {
			t.Error("expected 5 USD not to equal float 6.0")
		}
		_, err = mInvalid.EqualsFloat(5.0)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("LessThanInt", func(t *testing.T) {
		t.Parallel()
		lt, err := m5USD.LessThanInt(10)
		if err != nil || !lt {
			t.Error("expected 5 USD to be less than 10")
		}
		lt, err = m5USD.LessThanInt(3)
		if err != nil || lt {
			t.Error("expected 5 USD not to be less than 3")
		}
		_, err = mInvalid.LessThanInt(5)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("LessThanString", func(t *testing.T) {
		t.Parallel()
		lt, err := m5USD.LessThanString("10")
		if err != nil || !lt {
			t.Error("expected 5 USD to be less than '10'")
		}
		lt, err = m5USD.LessThanString("3")
		if err != nil || lt {
			t.Error("expected 5 USD not to be less than '3'")
		}
		_, err = mInvalid.LessThanString("5")
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("LessThanFloat", func(t *testing.T) {
		t.Parallel()
		lt, err := m5USD.LessThanFloat(10.0)
		if err != nil || !lt {
			t.Error("expected 5 USD to be less than 10.0")
		}
		lt, err = m5USD.LessThanFloat(3.0)
		if err != nil || lt {
			t.Error("expected 5 USD not to be less than 3.0")
		}
		_, err = mInvalid.LessThanFloat(5.0)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("GreaterThanInt", func(t *testing.T) {
		t.Parallel()
		gt, err := m5USD.GreaterThanInt(3)
		if err != nil || !gt {
			t.Error("expected 5 USD to be greater than 3")
		}
		gt, err = m5USD.GreaterThanInt(10)
		if err != nil || gt {
			t.Error("expected 5 USD not to be greater than 10")
		}
		_, err = mInvalid.GreaterThanInt(5)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("GreaterThanString", func(t *testing.T) {
		t.Parallel()
		gt, err := m5USD.GreaterThanString("3")
		if err != nil || !gt {
			t.Error("expected 5 USD to be greater than '3'")
		}
		gt, err = m5USD.GreaterThanString("10")
		if err != nil || gt {
			t.Error("expected 5 USD not to be greater than '10'")
		}
		_, err = mInvalid.GreaterThanString("5")
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("GreaterThanFloat", func(t *testing.T) {
		t.Parallel()
		gt, err := m5USD.GreaterThanFloat(3.0)
		if err != nil || !gt {
			t.Error("expected 5 USD to be greater than 3.0")
		}
		gt, err = m5USD.GreaterThanFloat(10.0)
		if err != nil || gt {
			t.Error("expected 5 USD not to be greater than 10.0")
		}
		_, err = mInvalid.GreaterThanFloat(5.0)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})
}

func TestMoneyPredicateMethods(t *testing.T) {
	t.Parallel()
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("IsInteger", func(t *testing.T) {
		t.Parallel()
		is, err := NewMoneyFromInt(5, "USD").IsInteger()
		if err != nil || !is {
			t.Error("expected 5 to be an integer")
		}
		is, err = NewMoneyFromString("5.5", "USD").IsInteger()
		if err != nil || is {
			t.Error("expected 5.5 not to be an integer")
		}
		_, err = mInvalid.IsInteger()
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("IsBetween", func(t *testing.T) {
		t.Parallel()
		minVal := NewMoneyFromInt(1, "USD")
		maxVal := NewMoneyFromInt(10, "USD")
		is, err := NewMoneyFromInt(5, "USD").IsBetween(minVal, maxVal)
		if err != nil || !is {
			t.Error("expected 5 to be between 1 and 10")
		}
		is, err = NewMoneyFromInt(15, "USD").IsBetween(minVal, maxVal)
		if err != nil || is {
			t.Error("expected 15 not to be between 1 and 10")
		}
		_, err = NewMoneyFromInt(5, "USD").IsBetween(NewMoneyFromInt(5, "EUR"), maxVal)
		if err == nil {
			t.Error("expected error on currency mismatch")
		}
		_, err = mInvalid.IsBetween(minVal, maxVal)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("IsCloseTo", func(t *testing.T) {
		t.Parallel()
		target := NewMoneyFromInt(5, "USD")
		tolerance := NewMoneyFromString("0.5", "USD")
		is, err := NewMoneyFromString("5.3", "USD").IsCloseTo(target, tolerance)
		if err != nil || !is {
			t.Error("expected 5.3 to be close to 5 with tolerance 0.5")
		}
		is, err = NewMoneyFromInt(10, "USD").IsCloseTo(target, tolerance)
		if err != nil || is {
			t.Error("expected 10 not to be close to 5")
		}
		_, err = NewMoneyFromInt(5, "USD").IsCloseTo(target, NewMoneyFromInt(-1, "USD"))
		if err == nil {
			t.Error("expected error on negative tolerance")
		}
		_, err = NewMoneyFromInt(5, "USD").IsCloseTo(NewMoneyFromInt(5, "EUR"), tolerance)
		if err == nil {
			t.Error("expected error on currency mismatch")
		}
		_, err = mInvalid.IsCloseTo(target, tolerance)
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("IsEven", func(t *testing.T) {
		t.Parallel()
		is, err := NewMoneyFromInt(4, "USD").IsEven()
		if err != nil || !is {
			t.Error("expected 4 to be even")
		}
		is, err = NewMoneyFromInt(5, "USD").IsEven()
		if err != nil || is {
			t.Error("expected 5 not to be even")
		}
		is, _ = NewMoneyFromString("4.5", "USD").IsEven()
		if is {
			t.Error("expected 4.5 not to be even")
		}
		_, err = mInvalid.IsEven()
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("IsOdd", func(t *testing.T) {
		t.Parallel()
		is, err := NewMoneyFromInt(5, "USD").IsOdd()
		if err != nil || !is {
			t.Error("expected 5 to be odd")
		}
		is, err = NewMoneyFromInt(4, "USD").IsOdd()
		if err != nil || is {
			t.Error("expected 4 not to be odd")
		}
		is, _ = NewMoneyFromString("5.5", "USD").IsOdd()
		if is {
			t.Error("expected 5.5 not to be odd")
		}
		_, err = mInvalid.IsOdd()
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})

	t.Run("IsMultipleOf", func(t *testing.T) {
		t.Parallel()
		is, err := NewMoneyFromInt(10, "USD").IsMultipleOf(NewMoneyFromInt(5, "USD"))
		if err != nil || !is {
			t.Error("expected 10 to be a multiple of 5")
		}
		is, err = NewMoneyFromInt(10, "USD").IsMultipleOf(NewMoneyFromInt(3, "USD"))
		if err != nil || is {
			t.Error("expected 10 not to be a multiple of 3")
		}
		_, err = NewMoneyFromInt(10, "USD").IsMultipleOf(NewMoneyFromInt(5, "EUR"))
		if err == nil {
			t.Error("expected error on currency mismatch")
		}
		_, err = mInvalid.IsMultipleOf(NewMoneyFromInt(5, "USD"))
		if err == nil {
			t.Error("expected error on invalid money")
		}
	})
}

func TestMoneyCheckPredicateVariants(t *testing.T) {
	t.Parallel()
	mInvalid := NewMoneyFromString("invalid", "USD")

	testCases := []struct {
		name     string
		check    bool
		expected bool
	}{
		{name: "CheckIsInteger on integer", check: NewMoneyFromInt(5, "USD").CheckIsInteger(), expected: true},
		{name: "CheckIsInteger on non-integer", check: NewMoneyFromString("5.5", "USD").CheckIsInteger(), expected: false},
		{name: "CheckIsInteger on invalid", check: mInvalid.CheckIsInteger(), expected: false},

		{name: "CheckIsBetween in range", check: NewMoneyFromInt(5, "USD").CheckIsBetween(NewMoneyFromInt(1, "USD"), NewMoneyFromInt(10, "USD")), expected: true},
		{name: "CheckIsBetween out of range", check: NewMoneyFromInt(15, "USD").CheckIsBetween(NewMoneyFromInt(1, "USD"), NewMoneyFromInt(10, "USD")), expected: false},
		{name: "CheckIsBetween invalid", check: mInvalid.CheckIsBetween(NewMoneyFromInt(1, "USD"), NewMoneyFromInt(10, "USD")), expected: false},

		{name: "CheckIsCloseTo close", check: NewMoneyFromString("5.3", "USD").CheckIsCloseTo(NewMoneyFromInt(5, "USD"), NewMoneyFromString("0.5", "USD")), expected: true},
		{name: "CheckIsCloseTo far", check: NewMoneyFromInt(10, "USD").CheckIsCloseTo(NewMoneyFromInt(5, "USD"), NewMoneyFromString("0.5", "USD")), expected: false},
		{name: "CheckIsCloseTo invalid", check: mInvalid.CheckIsCloseTo(NewMoneyFromInt(5, "USD"), NewMoneyFromString("0.5", "USD")), expected: false},

		{name: "CheckIsEven on even", check: NewMoneyFromInt(4, "USD").CheckIsEven(), expected: true},
		{name: "CheckIsEven on odd", check: NewMoneyFromInt(5, "USD").CheckIsEven(), expected: false},
		{name: "CheckIsEven on invalid", check: mInvalid.CheckIsEven(), expected: false},

		{name: "CheckIsOdd on odd", check: NewMoneyFromInt(5, "USD").CheckIsOdd(), expected: true},
		{name: "CheckIsOdd on even", check: NewMoneyFromInt(4, "USD").CheckIsOdd(), expected: false},
		{name: "CheckIsOdd on invalid", check: mInvalid.CheckIsOdd(), expected: false},

		{name: "CheckIsMultipleOf true", check: NewMoneyFromInt(10, "USD").CheckIsMultipleOf(NewMoneyFromInt(5, "USD")), expected: true},
		{name: "CheckIsMultipleOf false", check: NewMoneyFromInt(10, "USD").CheckIsMultipleOf(NewMoneyFromInt(3, "USD")), expected: false},
		{name: "CheckIsMultipleOf invalid", check: mInvalid.CheckIsMultipleOf(NewMoneyFromInt(5, "USD")), expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check; got != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, got)
			}
		})
	}
}

func TestMoneyMustPredicateVariants(t *testing.T) {
	t.Parallel()
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("Must methods should not panic on valid inputs", func(t *testing.T) {
		t.Parallel()
		_ = NewMoneyFromInt(5, "USD").MustIsInteger()
		_ = NewMoneyFromInt(5, "USD").MustIsBetween(NewMoneyFromInt(1, "USD"), NewMoneyFromInt(10, "USD"))
		_ = NewMoneyFromInt(5, "USD").MustIsCloseTo(NewMoneyFromInt(5, "USD"), NewMoneyFromString("0.5", "USD"))
		_ = NewMoneyFromInt(4, "USD").MustIsEven()
		_ = NewMoneyFromInt(5, "USD").MustIsOdd()
		_ = NewMoneyFromInt(10, "USD").MustIsMultipleOf(NewMoneyFromInt(5, "USD"))
	})

	testCases := []struct {
		mustCall func()
		name     string
	}{
		{name: "MustIsInteger", mustCall: func() { _ = mInvalid.MustIsInteger() }},
		{name: "MustIsBetween", mustCall: func() { _ = mInvalid.MustIsBetween(NewMoneyFromInt(1, "USD"), NewMoneyFromInt(10, "USD")) }},
		{name: "MustIsCloseTo", mustCall: func() { _ = mInvalid.MustIsCloseTo(NewMoneyFromInt(5, "USD"), NewMoneyFromString("0.5", "USD")) }},
		{name: "MustIsEven", mustCall: func() { _ = mInvalid.MustIsEven() }},
		{name: "MustIsOdd", mustCall: func() { _ = mInvalid.MustIsOdd() }},
		{name: "MustIsMultipleOf", mustCall: func() { _ = mInvalid.MustIsMultipleOf(NewMoneyFromInt(5, "USD")) }},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s should panic on invalid money", tc.name), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected %s to panic, but it did not", tc.name)
				}
			}()
			tc.mustCall()
		})
	}
}
