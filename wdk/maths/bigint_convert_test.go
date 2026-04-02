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
	"reflect"
	"testing"

	"piko.sh/piko/internal/json"
)

func TestBigIntValueRetrievers(t *testing.T) {
	bValid := NewBigIntFromInt(123)
	bInvalid := NewBigIntFromString("invalid")

	t.Run("String", func(t *testing.T) {
		s, err := bValid.String()
		if err != nil || s != "123" {
			t.Errorf("expected String() to be '123', got %q with error %v", s, err)
		}
		_, err = bInvalid.String()
		if err == nil {
			t.Error("expected error from String() for invalid bigint")
		}
	})

	t.Run("Float64", func(t *testing.T) {
		f, err := bValid.Float64()
		if err != nil || f != 123.0 {
			t.Errorf("expected Float64() to be 123.0, got %f with error %v", f, err)
		}
		_, err = bInvalid.Float64()
		if err == nil {
			t.Error("expected error from Float64() for invalid bigint")
		}
	})

	t.Run("Int64", func(t *testing.T) {

		bFits := NewBigIntFromInt(math.MaxInt64)

		bOverflows := NewBigIntFromString("9223372036854775808")

		bInvalid := NewBigIntFromString("invalid")

		i, err := bFits.Int64()
		if err != nil {
			t.Errorf("did not expect an error for a value that fits in int64, but got %v", err)
		}
		if i != math.MaxInt64 {
			t.Errorf("expected Int64() to be %d, got %d", math.MaxInt64, i)
		}

		_, err = bOverflows.Int64()
		if err == nil {
			t.Error("expected error converting overflowing bigint to Int64, but got nil")
		}

		_, err = bInvalid.Int64()
		if err == nil {
			t.Error("expected error from Int64() for invalid bigint, but got nil")
		}
	})
}

func TestBigIntMustRetrievers(t *testing.T) {
	bValid := NewBigIntFromInt(789)
	bTooLarge := NewBigIntFromString("9223372036854775808")
	bInvalid := NewBigIntFromString("invalid")

	t.Run("MustString", func(t *testing.T) {
		if bValid.MustString() != "789" {
			t.Error("MustString failed for valid bigint")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustString to panic on an error")
			}
		}()
		_ = bInvalid.MustString()
	})

	t.Run("MustFloat64", func(t *testing.T) {
		if bValid.MustFloat64() != 789.0 {
			t.Error("MustFloat64 failed for valid bigint")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustFloat64 to panic on an error")
			}
		}()
		_ = bInvalid.MustFloat64()
	})

	t.Run("MustInt64", func(t *testing.T) {
		if bValid.MustInt64() != 789 {
			t.Error("MustInt64 failed for valid bigint")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustInt64 to panic on an overflow error")
			}
		}()
		_ = bTooLarge.MustInt64()
	})
}

func TestBigIntEncoding(t *testing.T) {
	t.Run("JSON Marshal", func(t *testing.T) {
		b := NewBigIntFromInt(123456789)
		bytes, err := json.Marshal(b)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}

		if string(bytes) != `"123456789"` {
			t.Errorf(`expected marshalled JSON to be '"123456789"', got %s`, string(bytes))
		}

		bInvalid := NewBigIntFromString("invalid")
		_, err = json.Marshal(bInvalid)
		if err == nil {
			t.Error("expected error marshalling an invalid bigint")
		}
	})

	t.Run("JSON Unmarshal", func(t *testing.T) {
		var b1 BigInt
		err := json.Unmarshal([]byte(`"54321"`), &b1)
		if err != nil {
			t.Fatalf("UnmarshalJSON from string failed: %v", err)
		}
		checkBigInt(t, b1, "54321", false)

		var b2 BigInt
		err = json.Unmarshal([]byte(`98765`), &b2)
		if err != nil {
			t.Fatalf("UnmarshalJSON from number failed: %v", err)
		}
		checkBigInt(t, b2, "98765", false)

		var b3 BigInt
		err = json.Unmarshal([]byte(`"invalid"`), &b3)
		if err == nil {
			t.Error("expected error unmarshalling an invalid string")
		}
		checkBigInt(t, b3, "", true)

		var b4 BigInt
		err = json.Unmarshal([]byte(`{"key":"value"}`), &b4)
		if err == nil {
			t.Error("expected error unmarshalling from an object")
		}
	})

	t.Run("SQL Scan and Value", func(t *testing.T) {
		b := NewBigIntFromInt(12345)
		v, err := b.Value()
		if err != nil {
			t.Fatalf("Value() failed: %v", err)
		}
		if !reflect.DeepEqual(v, "12345") {
			t.Errorf("expected Value() to be '12345', got %v", v)
		}

		var scannedB BigInt
		if err := scannedB.Scan(v); err != nil {
			t.Fatalf("Scan() from string value failed: %v", err)
		}
		checkBigInt(t, scannedB, "12345", false)
	})

	t.Run("SQL Scan Types", func(t *testing.T) {
		testCases := []struct {
			name     string
			input    any
			expected string
		}{
			{name: "string", input: "98765", expected: "98765"},
			{name: "[]byte", input: []byte("54321"), expected: "54321"},
			{name: "int64", input: int64(math.MaxInt64), expected: "9223372036854775807"},
			{name: "nil", input: nil, expected: "0"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var b BigInt
				err := b.Scan(tc.input)
				if err != nil {
					t.Fatalf("Scan failed for type %s: %v", tc.name, err)
				}
				checkBigInt(t, b, tc.expected, false)
			})
		}
	})

	t.Run("SQL Scan Invalid Type", func(t *testing.T) {
		var b BigInt
		err := b.Scan(true)
		if err == nil {
			t.Error("expected an error when scanning an unsupported type")
		}
	})

	t.Run("SQL Value with Error", func(t *testing.T) {
		bInvalid := NewBigIntFromString("invalid")
		_, err := bInvalid.Value()
		if err == nil {
			t.Error("expected error from Value() for an invalid bigint")
		}
	})
}
