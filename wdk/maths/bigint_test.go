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
	"math"
	"testing"
)

func checkBigInt(t *testing.T, b BigInt, expectedValue string, expectError bool) {
	t.Helper()

	err := b.Err()
	if expectError {
		if err == nil {
			t.Errorf("expected an error, but got nil for value %q", expectedValue)
		}
		return
	}

	if err != nil {
		t.Errorf("did not expect an error, but got: %v", err)
		return
	}

	s, strErr := b.String()
	if strErr != nil {
		t.Errorf("unexpected error converting to string: %v", strErr)
	}

	if s != expectedValue {
		t.Errorf("expected value %q, but got %q", expectedValue, s)
	}
}

func TestBigIntConstructors(t *testing.T) {
	t.Run("NewBigIntFromString", func(t *testing.T) {
		checkBigInt(t, NewBigIntFromString("123"), "123", false)
		checkBigInt(t, NewBigIntFromString("-456"), "-456", false)
		checkBigInt(t, NewBigIntFromString(""), "0", false)
		checkBigInt(t, NewBigIntFromString("abc"), "", true)
		largeNum := "987654321098765432109876543210"
		checkBigInt(t, NewBigIntFromString(largeNum), largeNum, false)
	})

	t.Run("NewBigIntFromInt", func(t *testing.T) {
		checkBigInt(t, NewBigIntFromInt(123), "123", false)
		checkBigInt(t, NewBigIntFromInt(-456), "-456", false)
		checkBigInt(t, NewBigIntFromInt(0), "0", false)
		checkBigInt(t, NewBigIntFromInt(math.MaxInt64), "9223372036854775807", false)
	})

	t.Run("ConstantConstructors", func(t *testing.T) {
		checkBigInt(t, ZeroBigInt(), "0", false)
		checkBigInt(t, OneBigInt(), "1", false)
		checkBigInt(t, TenBigInt(), "10", false)
		checkBigInt(t, HundredBigInt(), "100", false)
	})
}

func TestBigIntCoreMethods(t *testing.T) {
	bValid := NewBigIntFromInt(123)
	bInvalid := NewBigIntFromString("abc")

	t.Run("Err", func(t *testing.T) {
		if bValid.Err() != nil {
			t.Error("expected nil error for valid bigint")
		}
		if bInvalid.Err() == nil {
			t.Error("expected non-nil error for invalid bigint")
		}
	})

	t.Run("Must", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected Must to panic on an error, but it did not")
			}
		}()
		bValid.Must()
		bInvalid.Must()
	})
}

func TestBigIntArithmetic(t *testing.T) {
	b10 := NewBigIntFromInt(10)
	b5 := NewBigIntFromInt(5)
	bNeg2 := NewBigIntFromInt(-2)
	bInvalid := NewBigIntFromString("invalid")

	t.Run("Add", func(t *testing.T) {
		checkBigInt(t, b10.Add(b5), "15", false)
		checkBigInt(t, b10.Add(bNeg2), "8", false)
		checkBigInt(t, b10.AddInt(5), "15", false)
		checkBigInt(t, b10.AddString("5"), "15", false)
	})

	t.Run("Subtract", func(t *testing.T) {
		checkBigInt(t, b10.Subtract(b5), "5", false)
		checkBigInt(t, b5.Subtract(b10), "-5", false)
		checkBigInt(t, b10.SubtractInt(2), "8", false)
		checkBigInt(t, b10.SubtractString("1"), "9", false)
	})

	t.Run("Multiply", func(t *testing.T) {
		checkBigInt(t, b10.Multiply(b5), "50", false)
		checkBigInt(t, b10.Multiply(bNeg2), "-20", false)
		checkBigInt(t, b10.MultiplyInt(3), "30", false)
		checkBigInt(t, b10.MultiplyString("15"), "150", false)
	})

	t.Run("Divide", func(t *testing.T) {
		checkBigInt(t, b10.Divide(b5), "2", false)
		checkBigInt(t, b10.Divide(NewBigIntFromInt(3)), "3", false)
		checkBigInt(t, b10.DivideInt(4), "2", false)
		checkBigInt(t, b10.DivideString("2"), "5", false)
		checkBigInt(t, b10.Divide(ZeroBigInt()), "", true)
	})

	t.Run("Remainder", func(t *testing.T) {
		checkBigInt(t, NewBigIntFromInt(10).RemainderInt(3), "1", false)
		checkBigInt(t, NewBigIntFromString("11").Remainder(NewBigIntFromInt(3)), "2", false)
		checkBigInt(t, NewBigIntFromString("-10").RemainderInt(3), "-1", false)
		checkBigInt(t, NewBigIntFromInt(10).Remainder(ZeroBigInt()), "", true)
	})

	t.Run("Power", func(t *testing.T) {
		checkBigInt(t, NewBigIntFromInt(3).Power(NewBigIntFromInt(3)), "27", false)
		checkBigInt(t, NewBigIntFromInt(2).PowerInt(10), "1024", false)
		checkBigInt(t, NewBigIntFromInt(10).PowerString("0"), "1", false)
		checkBigInt(t, NewBigIntFromInt(5).Power(NewBigIntFromInt(-2)), "0", false)
	})

	t.Run("Chaining", func(t *testing.T) {
		result := b10.Add(b5).Multiply(bNeg2).Subtract(OneBigInt())
		checkBigInt(t, result, "-31", false)
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		result := b10.Add(bInvalid).Multiply(b5)
		checkBigInt(t, result, "", true)
		result = b10.Add(b5).Multiply(bInvalid)
		checkBigInt(t, result, "", true)
		result = bInvalid.Add(b5)
		checkBigInt(t, result, "", true)
	})

	t.Run("Immutability", func(t *testing.T) {
		original := NewBigIntFromInt(100)
		modified := original.AddInt(5)

		checkBigInt(t, modified, "105", false)
		checkBigInt(t, original, "100", false)

		if &original == &modified {
			t.Error("expected a new BigInt instance, but got the same one (not immutable)")
		}
	})
}

func ExampleBigInt() {
	googol := NewBigIntFromString("1").Multiply(
		TenBigInt().PowerInt(100),
	)
	fmt.Println(googol.MustString())

}

func TestBigIntInPlaceArithmetic(t *testing.T) {
	t.Run("AddInPlace", func(t *testing.T) {
		b1 := NewBigIntFromInt(10)
		b2 := NewBigIntFromInt(5)
		b1.AddInPlace(b2)
		checkBigInt(t, b1, "15", false)
		checkBigInt(t, b2, "5", false)
	})

	t.Run("SubtractInPlace", func(t *testing.T) {
		b1 := NewBigIntFromInt(10)
		b2 := NewBigIntFromInt(5)
		b1.SubtractInPlace(b2)
		checkBigInt(t, b1, "5", false)
	})

	t.Run("MultiplyInPlace", func(t *testing.T) {
		b1 := NewBigIntFromInt(10)
		b2 := NewBigIntFromInt(5)
		b1.MultiplyInPlace(b2)
		checkBigInt(t, b1, "50", false)
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		b1 := NewBigIntFromInt(10)
		bInvalid := NewBigIntFromString("invalid")
		b1.AddInPlace(bInvalid)
		checkBigInt(t, b1, "", true)
	})
}

func TestZeroBigIntWithError(t *testing.T) {
	t.Parallel()
	b := ZeroBigIntWithError(errors.New("test error"))
	if b.Err() == nil {
		t.Error("expected error, got nil")
	}
	if b.Err().Error() != "test error" {
		t.Errorf("expected 'test error', got %q", b.Err().Error())
	}
}

func TestBigIntModulus(t *testing.T) {
	t.Parallel()

	t.Run("Modulus", func(t *testing.T) {
		t.Parallel()

		checkBigInt(t, NewBigIntFromInt(10).Modulus(NewBigIntFromInt(3)), "1", false)
		checkBigInt(t, NewBigIntFromInt(-10).Modulus(NewBigIntFromInt(3)), "2", false)
		checkBigInt(t, NewBigIntFromInt(10).Modulus(NewBigIntFromInt(-3)), "1", false)
		checkBigInt(t, NewBigIntFromInt(-10).Modulus(NewBigIntFromInt(-3)), "2", false)
		checkBigInt(t, NewBigIntFromInt(10).Modulus(ZeroBigInt()), "", true)
	})

	t.Run("ModulusInt", func(t *testing.T) {
		t.Parallel()
		checkBigInt(t, NewBigIntFromInt(10).ModulusInt(3), "1", false)
		checkBigInt(t, NewBigIntFromInt(-10).ModulusInt(3), "2", false)
	})

	t.Run("ModulusString", func(t *testing.T) {
		t.Parallel()
		checkBigInt(t, NewBigIntFromInt(10).ModulusString("3"), "1", false)
		checkBigInt(t, NewBigIntFromString("abc").ModulusString("3"), "", true)
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		t.Parallel()
		bInvalid := NewBigIntFromString("invalid")
		checkBigInt(t, bInvalid.Modulus(NewBigIntFromInt(3)), "", true)
		checkBigInt(t, NewBigIntFromInt(10).Modulus(bInvalid), "", true)
		checkBigInt(t, bInvalid.ModulusInt(3), "", true)
		checkBigInt(t, bInvalid.ModulusString("3"), "", true)
	})
}

func TestBigIntRemainderString(t *testing.T) {
	t.Parallel()
	checkBigInt(t, NewBigIntFromString("11").RemainderString("3"), "2", false)
	checkBigInt(t, NewBigIntFromString("-11").RemainderString("3"), "-2", false)
	checkBigInt(t, NewBigIntFromString("abc").RemainderString("3"), "", true)
}

func TestBigIntCrossTypeArithmetic(t *testing.T) {
	t.Parallel()
	b10 := NewBigIntFromInt(10)
	d3 := NewDecimalFromString("3.5")
	bInvalid := NewBigIntFromString("invalid")

	t.Run("AddDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.AddDecimal(d3), "13.5", false)
		checkDecimal(t, bInvalid.AddDecimal(d3), "", true)
	})

	t.Run("AddFloat", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.AddFloat(0.5), "10.5", false)
		checkDecimal(t, bInvalid.AddFloat(0.5), "", true)
	})

	t.Run("SubtractDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.SubtractDecimal(d3), "6.5", false)
		checkDecimal(t, bInvalid.SubtractDecimal(d3), "", true)
	})

	t.Run("SubtractFloat", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.SubtractFloat(0.5), "9.5", false)
		checkDecimal(t, bInvalid.SubtractFloat(0.5), "", true)
	})

	t.Run("MultiplyDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.MultiplyDecimal(d3), "35", false)
		checkDecimal(t, bInvalid.MultiplyDecimal(d3), "", true)
	})

	t.Run("MultiplyFloat", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.MultiplyFloat(1.5), "15", false)
		checkDecimal(t, bInvalid.MultiplyFloat(1.5), "", true)
	})

	t.Run("DivideDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.DivideDecimal(NewDecimalFromInt(4)), "2.5", false)
		checkDecimal(t, bInvalid.DivideDecimal(d3), "", true)
	})

	t.Run("DivideFloat", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, b10.DivideFloat(4), "2.5", false)
		checkDecimal(t, bInvalid.DivideFloat(4), "", true)
	})

	t.Run("PowerDecimal", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewBigIntFromInt(2).PowerDecimal(NewDecimalFromInt(3)), "8", false)
		checkDecimal(t, bInvalid.PowerDecimal(NewDecimalFromInt(2)), "", true)
	})

	t.Run("PowerFloat", func(t *testing.T) {
		t.Parallel()
		checkDecimal(t, NewBigIntFromInt(4).PowerFloat(0.5), "2", false)
		checkDecimal(t, bInvalid.PowerFloat(0.5), "", true)
	})
}

func TestBigIntErrorPropagationEdgeCases(t *testing.T) {
	t.Parallel()
	bInvalid := NewBigIntFromString("invalid")

	checkBigInt(t, bInvalid.AddInt(5), "", true)
	checkBigInt(t, bInvalid.AddString("5"), "", true)
	checkBigInt(t, bInvalid.SubtractInt(5), "", true)
	checkBigInt(t, bInvalid.SubtractString("5"), "", true)
	checkBigInt(t, bInvalid.MultiplyInt(5), "", true)
	checkBigInt(t, bInvalid.MultiplyString("5"), "", true)
	checkBigInt(t, bInvalid.DivideInt(5), "", true)
	checkBigInt(t, bInvalid.DivideString("5"), "", true)
	checkBigInt(t, bInvalid.PowerInt(2), "", true)
	checkBigInt(t, bInvalid.PowerString("2"), "", true)
	checkBigInt(t, bInvalid.RemainderInt(3), "", true)
	checkBigInt(t, bInvalid.RemainderString("3"), "", true)
}
