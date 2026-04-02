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
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"reflect"
	"testing"

	pikojson "piko.sh/piko/internal/json"
)

func TestMoneyValueRetrievers(t *testing.T) {
	mValid := NewMoneyFromString("123.456", "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("Amount", func(t *testing.T) {
		dec, err := mValid.Amount()
		if err != nil {
			t.Fatalf("Amount() failed: %v", err)
		}
		checkDecimal(t, dec, "123.456", false)

		_, err = mInvalid.Amount()
		if err == nil {
			t.Error("expected error from Amount() for invalid money")
		}
	})

	t.Run("CurrencyCode", func(t *testing.T) {
		code, err := mValid.CurrencyCode()
		if err != nil || code != "USD" {
			t.Errorf("expected currency code 'USD', got %q with error: %v", code, err)
		}
		_, err = mInvalid.CurrencyCode()
		if err == nil {
			t.Error("expected error from CurrencyCode() for invalid money")
		}
	})
}

func TestMoneyStringRepresentations(t *testing.T) {
	t.Run("Number (Full Precision)", func(t *testing.T) {
		m := NewMoneyFromString("123.4567", "USD")
		s, err := m.Number()
		if err != nil || s != "123.4567" {
			t.Errorf(`expected Number() to be "123.4567", got %q`, s)
		}
	})

	t.Run("RoundedNumber (Standard Precision, No Padding)", func(t *testing.T) {
		s, err := NewMoneyFromString("123.456", "USD").RoundedNumber()
		if err != nil || s != "123.46" {
			t.Errorf(`expected RoundedNumber() to be "123.46", got %q`, s)
		}
		s, err = NewMoneyFromInt(500, "EUR").RoundedNumber()
		if err != nil || s != "500" {
			t.Errorf(`expected RoundedNumber() for integer to be "500", got %q`, s)
		}
	})

	t.Run("FormattedNumber (Standard Precision, Padded)", func(t *testing.T) {
		s, err := NewMoneyFromString("123.4", "USD").FormattedNumber()
		if err != nil || s != "123.40" {
			t.Errorf(`expected FormattedNumber() to be "123.40", got %q`, s)
		}
		s, err = NewMoneyFromInt(500, "EUR").FormattedNumber()
		if err != nil || s != "500.00" {
			t.Errorf(`expected FormattedNumber() for integer to be "500.00", got %q`, s)
		}
		s, err = NewMoneyFromInt(556, "JPY").FormattedNumber()
		if err != nil || s != "556" {
			t.Errorf(`expected FormattedNumber() for JPY to be "556", got %q`, s)
		}
	})

	t.Run("String (Developer Facing)", func(t *testing.T) {
		s, err := NewMoneyFromString("123.45", "USD").String()
		if err != nil || s != "123.45 USD" {
			t.Errorf(`expected string "123.45 USD", got %q`, s)
		}
	})

	t.Run("Format (User Facing)", func(t *testing.T) {
		m := NewMoneyFromString("1234.567", "USD")
		formatted, err := m.Format("en-US")
		if err != nil || formatted != "$1,234.57" {
			t.Errorf("expected '$1,234.57', got %q", formatted)
		}
	})
}

func TestMoneyMustRetrievers(t *testing.T) {
	mValid := NewMoneyFromString("123.45", "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("MustNumber", func(t *testing.T) {
		if mValid.MustNumber() != "123.45" {
			t.Error("MustNumber failed for valid money")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustNumber to panic")
			}
		}()
		_ = mInvalid.MustNumber()
	})

	t.Run("MustString", func(t *testing.T) {
		if mValid.MustString() != "123.45 USD" {
			t.Error("MustString failed for valid money")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustString to panic")
			}
		}()
		_ = mInvalid.MustString()
	})

	t.Run("MustFormat", func(t *testing.T) {
		if mValid.MustFormat("en-US") != "$123.45" {
			t.Error("MustFormat failed for valid money")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustFormat to panic")
			}
		}()
		_ = mInvalid.MustFormat("en-US")
	})
}

func TestMoneyEncoding(t *testing.T) {
	t.Run("JSON Marshal", func(t *testing.T) {
		m := NewMoneyFromMinorInt(12345, "USD")
		b, err := pikojson.Marshal(m)
		if err != nil {
			t.Fatalf("MarshalJSON failed: %v", err)
		}
		expectedJSON := `{"number":"123.45","currency":"USD"}`
		if string(b) != expectedJSON {
			t.Errorf("expected marshalled JSON to be %s, got %s", expectedJSON, string(b))
		}

		mInvalid := NewMoneyFromString("invalid", "USD")
		_, err = pikojson.Marshal(mInvalid)
		if err == nil {
			t.Error("expected error marshalling an invalid money object")
		}
	})

	t.Run("JSON Unmarshal", func(t *testing.T) {
		testCases := []struct {
			name          string
			jsonInput     string
			expectedValue string
			expectedCode  string
			expectError   bool
		}{
			{name: "Object with string number", jsonInput: `{"number":"543.21","currency":"GBP"}`, expectedValue: "543.21", expectedCode: "GBP", expectError: false},
			{name: "Object with number", jsonInput: `{"number":987.65,"currency":"EUR"}`, expectedValue: "987.65", expectedCode: "EUR", expectError: false},
			{name: "Object with integer", jsonInput: `{"number":123,"currency":"JPY"}`, expectedValue: "123", expectedCode: "JPY", expectError: false},
			{name: "Object with case-insensitive currency", jsonInput: `{"number":"100","currency":"usd"}`, expectedValue: "100", expectedCode: "USD", expectError: false},
			{name: "Simple string", jsonInput: `"42.42"`, expectedValue: "42.42", expectedCode: "GBP", expectError: false},
			{name: "Empty string", jsonInput: `""`, expectedValue: "0", expectedCode: "GBP", expectError: false},
			{name: "Missing currency", jsonInput: `{"number":"100"}`, expectedValue: "", expectedCode: "", expectError: true},
			{name: "Missing number", jsonInput: `{"currency":"USD"}`, expectedValue: "", expectedCode: "", expectError: true},
			{name: "Unsupported number type", jsonInput: `{"number":true,"currency":"USD"}`, expectedValue: "", expectedCode: "", expectError: true},
			{name: "Invalid currency code", jsonInput: `{"number":"100","currency":"XXX"}`, expectedValue: "", expectedCode: "", expectError: true},
			{name: "Invalid JSON object", jsonInput: `{"number":"100","currency":`, expectedValue: "", expectedCode: "", expectError: true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var m Money
				err := pikojson.Unmarshal([]byte(tc.jsonInput), &m)
				if tc.expectError {
					if err == nil {
						t.Error("expected an error but got none")
					}
				} else {
					if err != nil {
						t.Errorf("did not expect an error but got: %v", err)
					}
					checkMoney(t, m, tc.expectedValue, tc.expectedCode, false)
				}
			})
		}
	})

	t.Run("SQL Scan and Value", func(t *testing.T) {
		m := NewMoneyFromMinorInt(999, "GBP")
		v, err := m.Value()
		if err != nil {
			t.Fatalf("Value() failed: %v", err)
		}
		expectedValue := "(9.99,GBP)"
		if !reflect.DeepEqual(v, expectedValue) {
			t.Errorf("expected Value() to be %q, got %v", expectedValue, v)
		}

		var scannedM Money
		if err := scannedM.Scan(v); err != nil {
			t.Fatalf("Scan() failed: %v", err)
		}
		checkMoney(t, scannedM, "9.99", "GBP", false)
	})

	t.Run("SQL Scan Lenient Zero", func(t *testing.T) {
		var m Money

		err := m.Scan("(0,)")
		if err != nil {
			t.Fatalf("expected Scan to leniently handle invalid zero, but got error: %v", err)
		}
		checkMoney(t, m, "0", "GBP", false)
	})
}

var (
	_ json.Marshaler   = (*Money)(nil)
	_ json.Unmarshaler = (*Money)(nil)
	_ driver.Valuer    = Money{}
	_ sql.Scanner      = (*Money)(nil)
)

func TestMoneyDefaultFormat(t *testing.T) {
	t.Parallel()

	t.Run("ValidMoney", func(t *testing.T) {
		t.Parallel()
		m := NewMoneyFromString("1234.56", "USD")
		result := m.DefaultFormat()
		if result == "" {
			t.Error("expected non-empty default format")
		}
	})

	t.Run("InvalidMoney", func(t *testing.T) {
		t.Parallel()
		mInvalid := NewMoneyFromString("invalid", "USD")
		result := mInvalid.DefaultFormat()
		if result == "" {
			t.Error("expected non-empty format even for invalid money (falls back to zero)")
		}
	})
}

func TestMoneyRoundedDefaultFormat(t *testing.T) {
	t.Parallel()
	m := NewMoneyFromString("1234.567", "USD")
	result := m.RoundedDefaultFormat()
	if result == "" {
		t.Error("expected non-empty rounded default format")
	}
}

func TestMoneyMustFormats(t *testing.T) {
	t.Parallel()
	mValid := NewMoneyFromString("123.456", "USD")
	mInvalid := NewMoneyFromString("invalid", "USD")

	t.Run("MustRoundedNumber valid", func(t *testing.T) {
		t.Parallel()
		s := mValid.MustRoundedNumber()
		if s != "123.46" {
			t.Errorf("expected '123.46', got %q", s)
		}
	})

	t.Run("MustRoundedNumber panic", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustRoundedNumber to panic on invalid money")
			}
		}()
		_ = mInvalid.MustRoundedNumber()
	})

	t.Run("MustFormattedNumber valid", func(t *testing.T) {
		t.Parallel()
		s := mValid.MustFormattedNumber()
		if s != "123.46" {
			t.Errorf("expected '123.46', got %q", s)
		}
	})

	t.Run("MustFormattedNumber panic", func(t *testing.T) {
		t.Parallel()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MustFormattedNumber to panic on invalid money")
			}
		}()
		_ = mInvalid.MustFormattedNumber()
	})
}
