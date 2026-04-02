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
	"math"
	"testing"
)

func checkDecimal(t *testing.T, d Decimal, expectedValue string, expectError bool) {
	t.Helper()

	err := d.Err()
	if expectError {
		if err == nil {
			t.Errorf("expected an error, but got nil")
		}
		return
	}

	if err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
		return
	}

	s, strErr := d.String()
	if strErr != nil {
		t.Errorf("unexpected error converting to string: %v", strErr)
	}

	if s != expectedValue {
		t.Errorf("expected value %q, but got %q", expectedValue, s)
	}
}

func TestConstructors(t *testing.T) {
	t.Run("NewDecimalFromString", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromString("123"), "123", false)
		checkDecimal(t, NewDecimalFromString("123.45"), "123.45", false)
		checkDecimal(t, NewDecimalFromString("-50.5"), "-50.5", false)
		checkDecimal(t, NewDecimalFromString(""), "0", false)
		checkDecimal(t, NewDecimalFromString("abc"), "", true)
		checkDecimal(t, NewDecimalFromString("1.23e2"), "123", false)
		checkDecimal(t, NewDecimalFromString("1.23e-2"), "0.0123", false)
	})

	t.Run("NewDecimalFromInt", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromInt(123), "123", false)
		checkDecimal(t, NewDecimalFromInt(-456), "-456", false)
		checkDecimal(t, NewDecimalFromInt(0), "0", false)
	})

	t.Run("NewDecimalFromFloat", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromFloat(123.45), "123.45", false)
		checkDecimal(t, NewDecimalFromFloat(-0.5), "-0.5", false)
		checkDecimal(t, NewDecimalFromFloat(math.NaN()), "", true)
		checkDecimal(t, NewDecimalFromFloat(math.Inf(1)), "", true)
		checkDecimal(t, NewDecimalFromFloat(math.Inf(-1)), "", true)
	})

	t.Run("ConstantConstructors", func(t *testing.T) {
		checkDecimal(t, ZeroDecimal(), "0", false)
		checkDecimal(t, OneDecimal(), "1", false)
		checkDecimal(t, TenDecimal(), "10", false)
		checkDecimal(t, HundredDecimal(), "100", false)
	})
}

func TestCoreMethods(t *testing.T) {
	dValid := NewDecimalFromInt(123)
	dInvalid := NewDecimalFromString("abc")

	t.Run("Err", func(t *testing.T) {
		if dValid.Err() != nil {
			t.Error("expected nil error for valid decimal")
		}
		if dInvalid.Err() == nil {
			t.Error("expected non-nil error for invalid decimal")
		}
	})

	t.Run("Must", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected Must to panic on an error, but it did not")
			}
		}()
		dValid.Must()
		dInvalid.Must()
	})
}

func TestArithmetic(t *testing.T) {
	d10 := NewDecimalFromInt(10)
	d5 := NewDecimalFromInt(5)
	dNeg2 := NewDecimalFromInt(-2)
	dInvalid := NewDecimalFromString("invalid")

	t.Run("Add", func(t *testing.T) {
		checkDecimal(t, d10.Add(d5), "15", false)
		checkDecimal(t, d10.Add(dNeg2), "8", false)
		checkDecimal(t, d10.AddInt(5), "15", false)
		checkDecimal(t, d10.AddString("5.5"), "15.5", false)
		checkDecimal(t, d10.AddFloat(0.1), "10.1", false)
	})

	t.Run("Subtract", func(t *testing.T) {
		checkDecimal(t, d10.Subtract(d5), "5", false)
		checkDecimal(t, d5.Subtract(d10), "-5", false)
		checkDecimal(t, d10.SubtractInt(2), "8", false)
		checkDecimal(t, d10.SubtractString("1.5"), "8.5", false)
		checkDecimal(t, d10.SubtractFloat(0.5), "9.5", false)
	})

	t.Run("Multiply", func(t *testing.T) {
		checkDecimal(t, d10.Multiply(d5), "50", false)
		checkDecimal(t, d10.Multiply(dNeg2), "-20", false)
		checkDecimal(t, d10.MultiplyInt(3), "30", false)
		checkDecimal(t, d10.MultiplyString("1.5"), "15", false)
		checkDecimal(t, d10.MultiplyFloat(0.5), "5", false)
	})

	t.Run("Divide", func(t *testing.T) {
		checkDecimal(t, d10.Divide(d5), "2", false)
		checkDecimal(t, d10.Divide(NewDecimalFromInt(3)), "3.333333333333333333333333333333333", false)
		checkDecimal(t, d10.DivideInt(4), "2.5", false)
		checkDecimal(t, d10.DivideString("2.5"), "4", false)
		checkDecimal(t, d10.DivideFloat(0.25), "40", false)
		checkDecimal(t, d10.Divide(ZeroDecimal()), "", true)
	})

	t.Run("Remainder", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromInt(10).RemainderInt(3), "1", false)
		checkDecimal(t, NewDecimalFromString("10.5").Remainder(NewDecimalFromInt(3)), "1.5", false)
		checkDecimal(t, NewDecimalFromString("-10.5").RemainderInt(3), "-1.5", false)
		checkDecimal(t, NewDecimalFromInt(10).Remainder(ZeroDecimal()), "", true)
	})

	t.Run("Power", func(t *testing.T) {
		checkDecimal(t, NewDecimalFromInt(3).Power(NewDecimalFromInt(3)), "27", false)
		checkDecimal(t, NewDecimalFromInt(4).PowerFloat(0.5), "2", false)
		checkDecimal(t, NewDecimalFromInt(2).PowerString("-2"), "0.25", false)
	})

	t.Run("Chaining", func(t *testing.T) {
		result := d10.Add(d5).Multiply(dNeg2).Subtract(OneDecimal())
		checkDecimal(t, result, "-31", false)
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		checkDecimal(t, dInvalid.Add(d5), "", true)
		checkDecimal(t, d5.Add(dInvalid), "", true)
		checkDecimal(t, d10.Divide(dInvalid), "", true)
	})
}

func TestDecimalCrossTypeArithmetic(t *testing.T) {
	t.Parallel()
	d10 := NewDecimalFromInt(10)
	b5 := NewBigIntFromInt(5)
	dInvalid := NewDecimalFromString("invalid")

	t.Run("AddDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.AddDecimal(NewDecimalFromString("2.5")), "12.5", false)
		checkDecimal(t, dInvalid.AddDecimal(d10), "", true)
	})

	t.Run("AddBigInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.AddBigInt(b5), "15", false)
		checkDecimal(t, dInvalid.AddBigInt(b5), "", true)
	})

	t.Run("SubtractDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.SubtractDecimal(NewDecimalFromString("2.5")), "7.5", false)
		checkDecimal(t, dInvalid.SubtractDecimal(d10), "", true)
	})

	t.Run("SubtractBigInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.SubtractBigInt(b5), "5", false)
		checkDecimal(t, dInvalid.SubtractBigInt(b5), "", true)
	})

	t.Run("MultiplyDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.MultiplyDecimal(NewDecimalFromString("1.5")), "15", false)
		checkDecimal(t, dInvalid.MultiplyDecimal(d10), "", true)
	})

	t.Run("MultiplyBigInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.MultiplyBigInt(b5), "50", false)
		checkDecimal(t, dInvalid.MultiplyBigInt(b5), "", true)
	})

	t.Run("DivideDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.DivideDecimal(NewDecimalFromString("4")), "2.5", false)
		checkDecimal(t, dInvalid.DivideDecimal(d10), "", true)
	})

	t.Run("DivideBigInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.DivideBigInt(b5), "2", false)
		checkDecimal(t, dInvalid.DivideBigInt(b5), "", true)
	})
}

func TestDecimalModulus(t *testing.T) {
	t.Parallel()

	t.Run("Modulus", func(t *testing.T) {
		t.Parallel()

		checkDecimal(t, NewDecimalFromString("10.5").Modulus(NewDecimalFromInt(3)), "1.5", false)
		checkDecimal(t, NewDecimalFromString("-10.5").Modulus(NewDecimalFromInt(3)), "1.5", false)
		checkDecimal(t, NewDecimalFromString("10.5").Modulus(NewDecimalFromInt(-3)), "-1.5", false)
		checkDecimal(t, NewDecimalFromString("-10.5").Modulus(NewDecimalFromInt(-3)), "-1.5", false)
		checkDecimal(t, NewDecimalFromInt(10).Modulus(ZeroDecimal()), "", true)
	})

	t.Run("ModulusDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(10).ModulusDecimal(NewDecimalFromInt(3)), "1", false)
		checkDecimal(t, NewDecimalFromString("invalid").ModulusDecimal(NewDecimalFromInt(3)), "", true)
	})

	t.Run("ModulusBigInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(10).ModulusBigInt(NewBigIntFromInt(3)), "1", false)
		checkDecimal(t, NewDecimalFromString("invalid").ModulusBigInt(NewBigIntFromInt(3)), "", true)
	})

	t.Run("ModulusInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(10).ModulusInt(3), "1", false)
		checkDecimal(t, NewDecimalFromString("invalid").ModulusInt(3), "", true)
	})

	t.Run("ModulusString", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(10).ModulusString("3"), "1", false)
		checkDecimal(t, NewDecimalFromString("invalid").ModulusString("3"), "", true)
	})

	t.Run("ModulusFloat", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(10).ModulusFloat(3), "1", false)
		checkDecimal(t, NewDecimalFromString("invalid").ModulusFloat(3), "", true)
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		t.Parallel()
		dInvalid := NewDecimalFromString("invalid")
		checkDecimal(t, dInvalid.Modulus(NewDecimalFromInt(3)), "", true)
		checkDecimal(t, NewDecimalFromInt(10).Modulus(dInvalid), "", true)
	})
}

func TestDecimalRemainderVariants(t *testing.T) {
	t.Parallel()
	d10 := NewDecimalFromString("10.5")
	dInvalid := NewDecimalFromString("invalid")

	t.Run("RemainderDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.RemainderDecimal(NewDecimalFromInt(3)), "1.5", false)
		checkDecimal(t, dInvalid.RemainderDecimal(NewDecimalFromInt(3)), "", true)
	})

	t.Run("RemainderBigInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.RemainderBigInt(NewBigIntFromInt(3)), "1.5", false)
		checkDecimal(t, dInvalid.RemainderBigInt(NewBigIntFromInt(3)), "", true)
	})

	t.Run("RemainderString", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.RemainderString("3"), "1.5", false)
		checkDecimal(t, dInvalid.RemainderString("3"), "", true)
	})

	t.Run("RemainderFloat", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, d10.RemainderFloat(3), "1.5", false)
		checkDecimal(t, dInvalid.RemainderFloat(3), "", true)
	})
}

func TestDecimalPowerVariants(t *testing.T) {
	t.Parallel()
	dInvalid := NewDecimalFromString("invalid")

	t.Run("PowerDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(2).PowerDecimal(NewDecimalFromInt(3)), "8", false)
		checkDecimal(t, dInvalid.PowerDecimal(NewDecimalFromInt(2)), "", true)
	})

	t.Run("PowerBigInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(2).PowerBigInt(NewBigIntFromInt(3)), "8", false)
		checkDecimal(t, dInvalid.PowerBigInt(NewBigIntFromInt(2)), "", true)
	})

	t.Run("PowerInt", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewDecimalFromInt(2).PowerInt(3), "8", false)
		checkDecimal(t, dInvalid.PowerInt(2), "", true)
	})
}

func TestDecimalInPlace(t *testing.T) {
	t.Parallel()

	t.Run("AddInPlace", func(t *testing.T) {
		d := NewDecimalFromInt(10)
		d.AddInPlace(NewDecimalFromString("2.5"))
		checkDecimal(t, d, "12.5", false)
	})

	t.Run("SubtractInPlace", func(t *testing.T) {
		d := NewDecimalFromInt(10)
		d.SubtractInPlace(NewDecimalFromString("2.5"))
		checkDecimal(t, d, "7.5", false)
	})

	t.Run("MultiplyInPlace", func(t *testing.T) {
		d := NewDecimalFromInt(10)
		d.MultiplyInPlace(NewDecimalFromString("1.5"))
		checkDecimal(t, d, "15", false)
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		d := NewDecimalFromInt(10)
		dInvalid := NewDecimalFromString("invalid")
		d.AddInPlace(dInvalid)
		checkDecimal(t, d, "", true)
	})
}

func TestDecimalErrorPropagationEdgeCases(t *testing.T) {
	t.Parallel()
	dInvalid := NewDecimalFromString("invalid")

	checkDecimal(t, dInvalid.AddInt(5), "", true)
	checkDecimal(t, dInvalid.AddString("5"), "", true)
	checkDecimal(t, dInvalid.AddFloat(5.0), "", true)
	checkDecimal(t, dInvalid.SubtractInt(5), "", true)
	checkDecimal(t, dInvalid.SubtractString("5"), "", true)
	checkDecimal(t, dInvalid.SubtractFloat(5.0), "", true)
	checkDecimal(t, dInvalid.MultiplyInt(5), "", true)
	checkDecimal(t, dInvalid.MultiplyString("5"), "", true)
	checkDecimal(t, dInvalid.MultiplyFloat(5.0), "", true)
	checkDecimal(t, dInvalid.DivideInt(5), "", true)
	checkDecimal(t, dInvalid.DivideString("5"), "", true)
	checkDecimal(t, dInvalid.DivideFloat(5.0), "", true)
	checkDecimal(t, dInvalid.RemainderInt(3), "", true)
	checkDecimal(t, dInvalid.PowerFloat(2.0), "", true)
	checkDecimal(t, dInvalid.PowerString("2"), "", true)
}
