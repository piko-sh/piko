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
	"sync"
	"testing"
)

func checkMoney(t *testing.T, m Money, expectedValue string, expectedCode string, expectError bool) {
	t.Helper()

	err := m.Err()
	if expectError {
		if err == nil {
			t.Errorf("expected an error, but got nil for %s %s", expectedValue, expectedCode)
		}
		return
	}

	if err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
		return
	}

	amount, amountErr := m.Amount()
	if amountErr != nil {
		t.Errorf("unexpected error getting amount: %v", amountErr)
	}
	checkDecimal(t, amount, expectedValue, false)

	code, codeErr := m.CurrencyCode()
	if codeErr != nil {
		t.Errorf("unexpected error getting currency code: %v", codeErr)
	}
	if code != expectedCode {
		t.Errorf("expected currency code %q, but got %q", expectedCode, code)
	}
}

func TestMoneyConstructors(t *testing.T) {
	t.Run("NewMoneyFromString", func(t *testing.T) {
		checkMoney(t, NewMoneyFromString("123.45", "USD"), "123.45", "USD", false)
		checkMoney(t, NewMoneyFromString("100", "JPY"), "100", "JPY", false)
		checkMoney(t, NewMoneyFromString("", "USD"), "0", "USD", false)
	})

	t.Run("NewMoneyFromInt (Major Unit)", func(t *testing.T) {
		checkMoney(t, NewMoneyFromInt(500, "JPY"), "500", "JPY", false)
		checkMoney(t, NewMoneyFromInt(-50, "GBP"), "-50", "GBP", false)
	})

	t.Run("NewMoneyFromMinorInt", func(t *testing.T) {

		checkMoney(t, NewMoneyFromMinorInt(500, "USD"), "5", "USD", false)

		checkMoney(t, NewMoneyFromMinorInt(1234, "BHD"), "1.234", "BHD", false)
	})

	t.Run("NewMoneyFromFloat", func(t *testing.T) {
		checkMoney(t, NewMoneyFromFloat(99.99, "EUR"), "99.99", "EUR", false)
	})

	t.Run("NewMoneyFromDecimal", func(t *testing.T) {
		dec := NewDecimalFromString("1.23")
		checkMoney(t, NewMoneyFromDecimal(dec, "CHF"), "1.23", "CHF", false)
	})

	t.Run("Constructor Error Cases", func(t *testing.T) {
		checkMoney(t, NewMoneyFromDecimal(NewDecimalFromString("abc"), "USD"), "", "", true)
		checkMoney(t, NewMoneyFromString("100", "XXX"), "", "", true)
		checkMoney(t, NewMoneyFromString("abc", "USD"), "", "", true)
	})

	t.Run("Constant-like Constructors", func(t *testing.T) {
		checkMoney(t, ZeroMoney("USD"), "0", "USD", false)
		checkMoney(t, OneMoney("JPY"), "1", "JPY", false)
		checkMoney(t, HundredMoney("GBP"), "100", "GBP", false)
	})
}

func TestMoneyCoreMethods(t *testing.T) {
	mValid := NewMoneyFromInt(100, "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("Err", func(t *testing.T) {
		if mValid.Err() != nil {
			t.Errorf("Expected nil error for valid Money, got %v", mValid.Err())
		}
		if mInvalid.Err() == nil {
			t.Error("Expected a non-nil error for invalid Money")
		}
	})

	t.Run("Must", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected Must to panic on an error, but it did not")
			}
		}()
		mValid.Must()
		mInvalid.Must()
	})
}

func TestMoneyArithmetic(t *testing.T) {
	m10USD := NewMoneyFromInt(10, "USD")
	m5USD := NewMoneyFromInt(5, "USD")
	m3EUR := NewMoneyFromInt(3, "EUR")
	mInvalid := NewMoneyFromString("abc", "USD")
	d2 := NewDecimalFromInt(2)

	t.Run("Add", func(t *testing.T) {
		checkMoney(t, m10USD.Add(m5USD), "15", "USD", false)
		checkMoney(t, m10USD.AddInt(2), "12", "USD", false)
		checkMoney(t, m10USD.AddMinorInt(50), "10.5", "USD", false)
		checkMoney(t, m10USD.AddFloat(1.25), "11.25", "USD", false)
		checkMoney(t, m10USD.AddString("0.75"), "10.75", "USD", false)
		checkMoney(t, m10USD.AddDecimal(NewDecimalFromInt(1)), "11", "USD", false)
	})

	t.Run("Subtract", func(t *testing.T) {
		checkMoney(t, m10USD.Subtract(m5USD), "5", "USD", false)
		checkMoney(t, m10USD.SubtractInt(2), "8", "USD", false)
		checkMoney(t, m10USD.SubtractMinorInt(50), "9.5", "USD", false)
		checkMoney(t, m10USD.SubtractFloat(1.25), "8.75", "USD", false)
		checkMoney(t, m10USD.SubtractString("0.75"), "9.25", "USD", false)
		checkMoney(t, m10USD.SubtractDecimal(OneDecimal()), "9", "USD", false)
	})

	t.Run("Multiply", func(t *testing.T) {
		checkMoney(t, m10USD.Multiply(d2), "20", "USD", false)
		checkMoney(t, m10USD.MultiplyInt(3), "30", "USD", false)
		checkMoney(t, m10USD.MultiplyFloat(1.5), "15", "USD", false)
		checkMoney(t, m10USD.MultiplyString("0.1"), "1", "USD", false)
	})

	t.Run("Divide", func(t *testing.T) {
		checkMoney(t, m10USD.Divide(d2), "5", "USD", false)
		checkMoney(t, m10USD.DivideInt(4), "2.5", "USD", false)
		checkMoney(t, m10USD.DivideFloat(0.5), "20", "USD", false)
		checkMoney(t, m10USD.DivideString("8"), "1.25", "USD", false)
	})

	t.Run("Arithmetic Error Handling", func(t *testing.T) {
		checkMoney(t, m10USD.Add(m3EUR), "", "", true)
		checkMoney(t, mInvalid.Add(m5USD), "", "", true)
		checkMoney(t, m5USD.Add(mInvalid), "", "", true)
		checkMoney(t, m10USD.Divide(ZeroDecimal()), "", "", true)
	})

	t.Run("Chaining", func(t *testing.T) {
		result := m10USD.Add(m5USD).Multiply(d2).SubtractInt(10)
		checkMoney(t, result, "20", "USD", false)
	})

	t.Run("Immutability", func(t *testing.T) {
		original := NewMoneyFromInt(100, "USD")
		_ = original.Add(NewMoneyFromInt(5, "USD"))
		checkMoney(t, original, "100", "USD", false)
	})

}

func TestMoneyModulusAndRemainder(t *testing.T) {
	mPos := NewMoneyFromString("10.5", "USD")
	mNeg := NewMoneyFromString("-10.5", "USD")
	dPos := NewDecimalFromInt(3)
	dNeg := NewDecimalFromInt(-3)

	t.Run("Remainder", func(t *testing.T) {

		checkMoney(t, mPos.Remainder(dPos), "1.5", "USD", false)
		checkMoney(t, mNeg.Remainder(dPos), "-1.5", "USD", false)
		checkMoney(t, mPos.Remainder(dNeg), "1.5", "USD", false)
		checkMoney(t, mNeg.Remainder(dNeg), "-1.5", "USD", false)
	})

	t.Run("Modulus", func(t *testing.T) {

		checkMoney(t, mPos.Modulus(dPos), "1.5", "USD", false)
		checkMoney(t, mNeg.Modulus(dPos), "1.5", "USD", false)
		checkMoney(t, mPos.Modulus(dNeg), "-1.5", "USD", false)
		checkMoney(t, mNeg.Modulus(dNeg), "-1.5", "USD", false)
	})

	t.Run("ErrorCases", func(t *testing.T) {
		checkMoney(t, mPos.Remainder(ZeroDecimal()), "", "", true)
		checkMoney(t, mPos.Modulus(ZeroDecimal()), "", "", true)
	})
}

func TestMoneyInPlaceArithmetic(t *testing.T) {
	t.Run("AddInPlace", func(t *testing.T) {
		m1 := NewMoneyFromInt(10, "USD")
		m2 := NewMoneyFromInt(5, "USD")
		m1.AddInPlace(m2)
		checkMoney(t, m1, "15", "USD", false)
		checkMoney(t, m2, "5", "USD", false)
	})

	t.Run("SubtractInPlace", func(t *testing.T) {
		m1 := NewMoneyFromInt(10, "USD")
		m2 := NewMoneyFromInt(5, "USD")
		m1.SubtractInPlace(m2)
		checkMoney(t, m1, "5", "USD", false)
	})

	t.Run("MultiplyInPlace", func(t *testing.T) {
		m1 := NewMoneyFromInt(10, "USD")
		factor := NewDecimalFromInt(3)
		m1.MultiplyInPlace(factor)
		checkMoney(t, m1, "30", "USD", false)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		m1 := NewMoneyFromInt(10, "USD")
		mEUR := NewMoneyFromInt(5, "EUR")
		m1.AddInPlace(mEUR)
		checkMoney(t, m1, "", "", true)

		m2 := NewMoneyFromInt(10, "USD")
		mInvalid := NewMoneyFromString("invalid", "USD")
		m2.SubtractInPlace(mInvalid)
		checkMoney(t, m2, "", "", true)
	})
}

func TestRegisterCurrencyConcurrency(t *testing.T) {
	var wg sync.WaitGroup
	const numGoroutines = 100

	wg.Add(numGoroutines)
	for i := range numGoroutines {
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				code := fmt.Sprintf("C%d", i)
				RegisterCurrency(code, CurrencyDefinition{Digits: 2})
			} else {
				_ = NewMoneyFromInt(100, "USD")
				_ = NewMoneyFromInt(100, "EUR")
			}
		}(i)
	}

	wg.Wait()
}

func TestMoneyBigIntArithmetic(t *testing.T) {
	t.Parallel()
	m10USD := NewMoneyFromInt(10, "USD")
	b5 := NewBigIntFromInt(5)
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("AddBigInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, m10USD.AddBigInt(b5), "15", "USD", false)
		checkMoney(t, mInvalid.AddBigInt(b5), "", "", true)
	})

	t.Run("SubtractBigInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, m10USD.SubtractBigInt(b5), "5", "USD", false)
		checkMoney(t, mInvalid.SubtractBigInt(b5), "", "", true)
	})

	t.Run("MultiplyBigInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, m10USD.MultiplyBigInt(b5), "50", "USD", false)
		checkMoney(t, mInvalid.MultiplyBigInt(b5), "", "", true)
	})

	t.Run("DivideBigInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, m10USD.DivideBigInt(b5), "2", "USD", false)
		checkMoney(t, mInvalid.DivideBigInt(b5), "", "", true)
	})
}

func TestMoneyModulusAndRemainderVariants(t *testing.T) {
	t.Parallel()
	mPos := NewMoneyFromString("10.5", "USD")
	dPos := NewDecimalFromInt(3)
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("RemainderBigInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.RemainderBigInt(NewBigIntFromInt(3)), "1.5", "USD", false)
		checkMoney(t, mInvalid.RemainderBigInt(NewBigIntFromInt(3)), "", "", true)
	})

	t.Run("RemainderInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.RemainderInt(3), "1.5", "USD", false)
		checkMoney(t, mInvalid.RemainderInt(3), "", "", true)
	})

	t.Run("RemainderFloat", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.RemainderFloat(3), "1.5", "USD", false)
		checkMoney(t, mInvalid.RemainderFloat(3), "", "", true)
	})

	t.Run("RemainderString", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.RemainderString("3"), "1.5", "USD", false)
		checkMoney(t, mInvalid.RemainderString("3"), "", "", true)
	})

	t.Run("ModulusBigInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.ModulusBigInt(NewBigIntFromInt(3)), "1.5", "USD", false)
		checkMoney(t, mInvalid.ModulusBigInt(NewBigIntFromInt(3)), "", "", true)
	})

	t.Run("ModulusInt", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.ModulusInt(3), "1.5", "USD", false)
		checkMoney(t, mInvalid.ModulusInt(3), "", "", true)
	})

	t.Run("ModulusFloat", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.ModulusFloat(3), "1.5", "USD", false)
		checkMoney(t, mInvalid.ModulusFloat(3), "", "", true)
	})

	t.Run("ModulusString", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.ModulusString("3"), "1.5", "USD", false)
		checkMoney(t, mInvalid.ModulusString("3"), "", "", true)
	})

	t.Run("ErrorFromOperand", func(t *testing.T) {
		t.Parallel()
		checkMoney(t, mPos.Remainder(dPos), "1.5", "USD", false)
		checkMoney(t, mPos.Modulus(dPos), "1.5", "USD", false)
	})
}

func TestMoneyErrorPropagationEdgeCases(t *testing.T) {
	t.Parallel()
	mInvalid := NewMoneyFromString("invalid", "USD")

	checkMoney(t, mInvalid.AddInt(5), "", "", true)
	checkMoney(t, mInvalid.AddMinorInt(500), "", "", true)
	checkMoney(t, mInvalid.AddFloat(5.0), "", "", true)
	checkMoney(t, mInvalid.AddString("5"), "", "", true)
	checkMoney(t, mInvalid.AddDecimal(OneDecimal()), "", "", true)
	checkMoney(t, mInvalid.SubtractInt(5), "", "", true)
	checkMoney(t, mInvalid.SubtractMinorInt(500), "", "", true)
	checkMoney(t, mInvalid.SubtractFloat(5.0), "", "", true)
	checkMoney(t, mInvalid.SubtractString("5"), "", "", true)
	checkMoney(t, mInvalid.SubtractDecimal(OneDecimal()), "", "", true)
	checkMoney(t, mInvalid.MultiplyInt(5), "", "", true)
	checkMoney(t, mInvalid.MultiplyFloat(5.0), "", "", true)
	checkMoney(t, mInvalid.MultiplyString("5"), "", "", true)
	checkMoney(t, mInvalid.DivideInt(5), "", "", true)
	checkMoney(t, mInvalid.DivideFloat(5.0), "", "", true)
	checkMoney(t, mInvalid.DivideString("5"), "", "", true)
}
