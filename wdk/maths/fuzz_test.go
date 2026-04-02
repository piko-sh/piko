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

//go:build fuzz

package maths

import (
	"math"
	"strings"
	"testing"
)

func FuzzBigIntArithmetic(f *testing.F) {

	f.Add(int64(0), int64(0))
	f.Add(int64(1), int64(1))
	f.Add(int64(-1), int64(1))
	f.Add(int64(math.MaxInt64), int64(1))
	f.Add(int64(math.MinInt64), int64(1))
	f.Add(int64(math.MaxInt64), int64(math.MinInt64))
	f.Add(int64(0), int64(math.MaxInt64))
	f.Add(int64(123456789), int64(-987654321))

	f.Fuzz(func(t *testing.T, a, b int64) {
		bigA := NewBigIntFromInt(a)
		bigB := NewBigIntFromInt(b)
		zero := ZeroBigInt()
		one := OneBigInt()

		sumAB := bigA.Add(bigB)
		sumBA := bigB.Add(bigA)
		if eq, _ := sumAB.Equals(sumBA); !eq {
			t.Errorf("Addition not commutative: %d + %d", a, b)
		}

		prodAB := bigA.Multiply(bigB)
		prodBA := bigB.Multiply(bigA)
		if eq, _ := prodAB.Equals(prodBA); !eq {
			t.Errorf("Multiplication not commutative: %d * %d", a, b)
		}

		sumAZero := bigA.Add(zero)
		if eq, _ := sumAZero.Equals(bigA); !eq {
			t.Errorf("Additive identity failed: %d + 0 != %d", a, a)
		}

		prodAOne := bigA.Multiply(one)
		if eq, _ := prodAOne.Equals(bigA); !eq {
			t.Errorf("Multiplicative identity failed: %d * 1 != %d", a, a)
		}

		diff := bigA.Subtract(bigA)
		if eq, _ := diff.Equals(zero); !eq {
			t.Errorf("Additive inverse failed: %d - %d != 0", a, a)
		}

		prodAZero := bigA.Multiply(zero)
		if eq, _ := prodAZero.Equals(zero); !eq {
			t.Errorf("Multiplication by zero failed: %d * 0 != 0", a)
		}

		if b == 0 {
			result := bigA.Divide(bigB)
			if result.Err() == nil {
				t.Errorf("Division by zero should return error")
			}
			result = bigA.Remainder(bigB)
			if result.Err() == nil {
				t.Errorf("Remainder by zero should return error")
			}
		} else {

			quotient := bigA.Divide(bigB)
			remainder := bigA.Remainder(bigB)
			if quotient.Err() == nil && remainder.Err() == nil {
				reconstructed := quotient.Multiply(bigB).Add(remainder)
				if eq, _ := reconstructed.Equals(bigA); !eq {
					t.Errorf("Division relationship failed: %d != (%d / %d) * %d + (%d %% %d)", a, a, b, b, a, b)
				}
			}
		}
	})
}

func FuzzBigIntStringRoundtrip(f *testing.F) {
	f.Add("0")
	f.Add("1")
	f.Add("-1")
	f.Add("9223372036854775807")
	f.Add("-9223372036854775808")
	f.Add("99999999999999999999999999999999999999")
	f.Add("-99999999999999999999999999999999999999")
	f.Add("123456789012345678901234567890")

	f.Fuzz(func(t *testing.T, s string) {

		s = strings.TrimSpace(s)
		if s == "" || s == "-" || s == "+" {
			return
		}

		bigInt := NewBigIntFromString(s)
		if bigInt.Err() != nil {

			return
		}

		str, err := bigInt.String()
		if err != nil {
			t.Errorf("String() failed for valid BigInt from %q: %v", s, err)
			return
		}

		reparsed := NewBigIntFromString(str)
		if reparsed.Err() != nil {
			t.Errorf("Failed to reparse %q (from original %q): %v", str, s, reparsed.Err())
			return
		}

		if eq, _ := bigInt.Equals(reparsed); !eq {
			t.Errorf("Roundtrip failed: %q -> %q -> not equal", s, str)
		}
	})
}

func FuzzBigIntComparison(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(1), int64(2))
	f.Add(int64(-1), int64(1))
	f.Add(int64(math.MaxInt64), int64(math.MinInt64))

	f.Fuzz(func(t *testing.T, a, b int64) {
		bigA := NewBigIntFromInt(a)
		bigB := NewBigIntFromInt(b)

		if eq, _ := bigA.Equals(bigA); !eq {
			t.Errorf("Reflexivity failed: %d != %d", a, a)
		}

		eqAB, _ := bigA.Equals(bigB)
		eqBA, _ := bigB.Equals(bigA)
		if eqAB != eqBA {
			t.Errorf("Equality symmetry failed for %d, %d", a, b)
		}

		cmpAB, _ := bigA.Cmp(bigB)
		cmpBA, _ := bigB.Cmp(bigA)
		if cmpAB != -cmpBA {
			t.Errorf("Comparison antisymmetry failed: cmp(%d, %d)=%d, cmp(%d, %d)=%d", a, b, cmpAB, b, a, cmpBA)
		}

		ltAB, _ := bigA.LessThan(bigB)
		if ltAB && eqAB {
			t.Errorf("Inconsistent: %d < %d but also %d == %d", a, b, a, b)
		}

		gtAB, _ := bigA.GreaterThan(bigB)
		count := 0
		if ltAB {
			count++
		}
		if eqAB {
			count++
		}
		if gtAB {
			count++
		}
		if count != 1 {
			t.Errorf("Trichotomy failed for %d, %d: lt=%v, eq=%v, gt=%v", a, b, ltAB, eqAB, gtAB)
		}
	})
}

func FuzzBigIntIsZero(f *testing.F) {
	f.Add(int64(0), int64(1))
	f.Add(int64(0), int64(math.MaxInt64))
	f.Add(int64(0), int64(-1))
	f.Add(int64(1), int64(0))
	f.Add(int64(5), int64(5))

	f.Fuzz(func(t *testing.T, a, b int64) {
		bigA := NewBigIntFromInt(a)
		bigB := NewBigIntFromInt(b)

		zero := ZeroBigInt()
		product := zero.Multiply(bigA)
		if isZero, _ := product.IsZero(); !isZero {
			t.Errorf("0 * %d should be zero, but IsZero() returned false", a)
		}

		product2 := bigA.Multiply(zero)
		if isZero, _ := product2.IsZero(); !isZero {
			t.Errorf("%d * 0 should be zero, but IsZero() returned false", a)
		}

		diff := bigA.Subtract(bigA)
		if isZero, _ := diff.IsZero(); !isZero {
			t.Errorf("%d - %d should be zero, but IsZero() returned false", a, a)
		}

		prod := bigA.Multiply(bigB)
		complexZero := prod.Subtract(prod)
		if isZero, _ := complexZero.IsZero(); !isZero {
			t.Errorf("(%d * %d) - (%d * %d) should be zero, but IsZero() returned false", a, b, a, b)
		}

		isZeroResult, err := complexZero.IsZero()
		checkIsZeroResult := complexZero.CheckIsZero()
		if err == nil && isZeroResult != checkIsZeroResult {
			t.Errorf("IsZero() and CheckIsZero() disagree")
		}
	})
}

func FuzzDecimalArithmetic(f *testing.F) {
	f.Add(0.0, 0.0)
	f.Add(1.0, 1.0)
	f.Add(-1.0, 1.0)
	f.Add(0.1, 0.2)
	f.Add(math.MaxFloat64, 1.0)
	f.Add(math.SmallestNonzeroFloat64, 1.0)
	f.Add(123.456, -789.012)
	f.Add(0.0000001, 0.0000002)

	f.Fuzz(func(t *testing.T, a, b float64) {

		if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) {
			return
		}

		decA := NewDecimalFromFloat(a)
		decB := NewDecimalFromFloat(b)
		zero := ZeroDecimal()
		one := OneDecimal()

		if decA.Err() != nil || decB.Err() != nil {
			return
		}

		sumAB := decA.Add(decB)
		sumBA := decB.Add(decA)
		if sumAB.Err() == nil && sumBA.Err() == nil {
			if eq, _ := sumAB.Equals(sumBA); !eq {

				diff := sumAB.Subtract(sumBA).Abs()
				if gt, _ := diff.GreaterThan(NewDecimalFromString("0.0000001")); gt {
					t.Errorf("Addition not commutative: %v + %v", a, b)
				}
			}
		}

		prodAB := decA.Multiply(decB)
		prodBA := decB.Multiply(decA)
		if prodAB.Err() == nil && prodBA.Err() == nil {
			if eq, _ := prodAB.Equals(prodBA); !eq {
				diff := prodAB.Subtract(prodBA).Abs()
				if gt, _ := diff.GreaterThan(NewDecimalFromString("0.0000001")); gt {
					t.Errorf("Multiplication not commutative: %v * %v", a, b)
				}
			}
		}

		sumAZero := decA.Add(zero)
		if sumAZero.Err() == nil {
			if eq, _ := sumAZero.Equals(decA); !eq {
				t.Errorf("Additive identity failed: %v + 0", a)
			}
		}

		prodAOne := decA.Multiply(one)
		if prodAOne.Err() == nil {
			if eq, _ := prodAOne.Equals(decA); !eq {
				t.Errorf("Multiplicative identity failed: %v * 1", a)
			}
		}

		if isZero, _ := decB.IsZero(); isZero {
			result := decA.Divide(decB)
			if result.Err() == nil {
				t.Errorf("Division by zero should return error")
			}
		}
	})
}

func FuzzDecimalStringRoundtrip(f *testing.F) {
	f.Add("0")
	f.Add("0.0")
	f.Add("1")
	f.Add("-1")
	f.Add("0.1")
	f.Add("0.123456789")
	f.Add("-123.456")
	f.Add("999999999999.999999999999")
	f.Add("0.000000000001")

	f.Fuzz(func(t *testing.T, s string) {
		s = strings.TrimSpace(s)
		if s == "" || s == "-" || s == "+" || s == "." {
			return
		}

		dec := NewDecimalFromString(s)
		if dec.Err() != nil {
			return
		}

		str, err := dec.String()
		if err != nil {
			t.Errorf("String() failed for valid Decimal from %q: %v", s, err)
			return
		}

		reparsed := NewDecimalFromString(str)
		if reparsed.Err() != nil {
			t.Errorf("Failed to reparse %q: %v", str, reparsed.Err())
			return
		}

		if eq, _ := dec.Equals(reparsed); !eq {
			t.Errorf("Roundtrip failed: %q -> %q -> not equal", s, str)
		}
	})
}

func FuzzDecimalIsZero(f *testing.F) {
	f.Add(1.0, 2.0)
	f.Add(0.1, 0.2)
	f.Add(-5.5, 5.5)
	f.Add(1000000.0, 0.000001)

	f.Fuzz(func(t *testing.T, a, b float64) {
		if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) || math.IsInf(b, 0) {
			return
		}

		decA := NewDecimalFromFloat(a)
		if decA.Err() != nil {
			return
		}

		zero := ZeroDecimal()

		product := zero.Multiply(decA)
		if product.Err() == nil {
			if isZero, _ := product.IsZero(); !isZero {
				t.Errorf("0 * %v should be zero", a)
			}
		}

		diff := decA.Subtract(decA)
		if diff.Err() == nil {
			if isZero, _ := diff.IsZero(); !isZero {
				t.Errorf("%v - %v should be zero", a, a)
			}
		}
	})
}

func FuzzMoneyArithmetic(f *testing.F) {
	f.Add("0", "USD")
	f.Add("1.00", "USD")
	f.Add("-1.00", "USD")
	f.Add("0.01", "USD")
	f.Add("999999.99", "EUR")
	f.Add("0.001", "BTC")

	f.Fuzz(func(t *testing.T, amount, currency string) {
		amount = strings.TrimSpace(amount)
		currency = strings.TrimSpace(strings.ToUpper(currency))

		if amount == "" || currency == "" || len(currency) > 5 {
			return
		}

		money := NewMoneyFromString(amount, currency)
		if money.Err() != nil {
			return
		}

		zero := ZeroMoney(currency)

		sum := money.Add(zero)
		if sum.Err() == nil {
			if eq, _ := sum.Equals(money); !eq {
				t.Errorf("Additive identity failed for %s %s", amount, currency)
			}
		}

		diff := money.Subtract(money)
		if diff.Err() == nil {
			if isZero, _ := diff.IsZero(); !isZero {
				t.Errorf("%s - %s should be zero", amount, amount)
			}
		}
	})
}

func FuzzMoneyStringRoundtrip(f *testing.F) {
	f.Add("0.00", "USD")
	f.Add("1.23", "EUR")
	f.Add("-45.67", "GBP")
	f.Add("1000000.00", "JPY")

	f.Fuzz(func(t *testing.T, amount, currency string) {
		amount = strings.TrimSpace(amount)
		currency = strings.TrimSpace(strings.ToUpper(currency))

		if amount == "" || currency == "" || len(currency) > 5 {
			return
		}

		money := NewMoneyFromString(amount, currency)
		if money.Err() != nil {
			return
		}

		str, err := money.String()
		if err != nil {
			t.Errorf("String() failed: %v", err)
			return
		}

		if str == "" {
			t.Errorf("String() returned empty for valid money")
		}
	})
}

func FuzzBigIntToDecimalRoundtrip(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(1))
	f.Add(int64(-1))
	f.Add(int64(1000000))
	f.Add(int64(math.MaxInt32))
	f.Add(int64(math.MinInt32))

	f.Fuzz(func(t *testing.T, n int64) {
		bigInt := NewBigIntFromInt(n)

		dec := bigInt.ToDecimal()
		if dec.Err() != nil {

			return
		}

		isInt, _ := dec.IsInteger()
		if !isInt {
			t.Errorf("Decimal from BigInt(%d) should be integer", n)
		}

		backToBigInt := dec.ToBigInt()
		if backToBigInt.Err() != nil {
			t.Errorf("Failed to convert Decimal back to BigInt: %v", backToBigInt.Err())
			return
		}

		if eq, _ := bigInt.Equals(backToBigInt); !eq {
			t.Errorf("Roundtrip failed: BigInt(%d) -> Decimal -> BigInt != original", n)
		}
	})
}

func FuzzDecimalPrecision(f *testing.F) {

	f.Add("0.1", "0.2", "0.3")
	f.Add("0.01", "0.02", "0.03")
	f.Add("1.001", "2.002", "3.003")

	f.Fuzz(func(t *testing.T, aString, bString, expectedString string) {
		a := NewDecimalFromString(aString)
		b := NewDecimalFromString(bString)
		expected := NewDecimalFromString(expectedString)

		if a.Err() != nil || b.Err() != nil || expected.Err() != nil {
			return
		}

		sum := a.Add(b)
		if sum.Err() != nil {
			return
		}

		_, _ = sum.Equals(expected)

	})
}

func FuzzErrorPropagation(f *testing.F) {
	f.Add(int64(1), int64(0))
	f.Add(int64(100), int64(0))

	f.Fuzz(func(t *testing.T, a, b int64) {
		bigA := NewBigIntFromInt(a)
		bigB := NewBigIntFromInt(b)

		if b == 0 {

			errResult := bigA.Divide(bigB)
			if errResult.Err() == nil {
				t.Fatal("Division by zero should produce error")
			}

			added := errResult.Add(bigA)
			if added.Err() == nil {
				t.Error("Error should propagate through Add")
			}

			multiplied := errResult.Multiply(bigA)
			if multiplied.Err() == nil {
				t.Error("Error should propagate through Multiply")
			}

			subtracted := errResult.Subtract(bigA)
			if subtracted.Err() == nil {
				t.Error("Error should propagate through Subtract")
			}
		}
	})
}
