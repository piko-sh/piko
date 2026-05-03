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
	"database/sql/driver"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bojanz/currency"
	"piko.sh/piko/internal/json"
)

// Amount returns the underlying numeric value of the Money object as a
// Decimal. This is the way to get the value for non-monetary calculations.
//
// Returns Decimal which contains the numeric value.
// Returns error when the Money object holds an error from a prior operation.
func (m Money) Amount() (Decimal, error) {
	if m.err != nil {
		return Decimal{}, m.err
	}
	return NewDecimalFromString(m.amount.Number()), nil
}

// Number returns the raw, full-precision numeric value as a string, without
// currency symbols or formatting.
//
// For example, a Money object representing 123.456 EUR returns "123.456".
//
// Returns string which is the unformatted numeric value.
// Returns error when the Money object holds a prior error.
func (m Money) Number() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.amount.Number(), nil
}

// RoundedNumber returns the value as a string, rounded to the currency's
// standard decimal places, without trailing zeros for whole numbers.
// For example, 123.456 EUR becomes "123.46" and 500.00 EUR becomes "500".
//
// Returns string which is the formatted rounded value.
// Returns error when the Money has an existing error or conversion fails.
func (m Money) RoundedNumber() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	roundedAmount := m.amount.Round()

	decimalFromRounded := NewDecimalFromString(roundedAmount.Number())
	if decimalFromRounded.Err() != nil {
		return "", fmt.Errorf("failed to create decimal from rounded amount: %w", decimalFromRounded.Err())
	}
	return decimalFromRounded.String()
}

// FormattedNumber returns the mathematical value as a string, rounded and
// padded with zeros to the currency's standard number of decimal places.
//
// This is suitable for display where consistent formatting is required.
// Example: 123.456 EUR -> "123.46", 500 EUR -> "500.00", 500 JPY -> "500".
//
// Returns string which is the formatted number with correct decimal places.
// Returns error when the Money instance has a stored error or conversion fails.
//
// Safe for concurrent use. Uses a read lock on the currency registry.
func (m Money) FormattedNumber() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	currencyRegistryMutex.RLock()
	defer currencyRegistryMutex.RUnlock()
	roundedAmount := m.amount.Round()

	numberString := roundedAmount.Number()
	digits, ok := currency.GetDigits(roundedAmount.CurrencyCode())
	if !ok || digits == 0 {
		return NewDecimalFromString(numberString).String()
	}

	dec, err := NewDecimalFromString(numberString).Float64()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%.*f", digits, dec), nil
}

// CurrencyCode returns the ISO 4217 currency code (e.g., "USD", "EUR").
//
// Returns string which is the three-letter currency code.
// Returns error when the Money value was created with an invalid operation.
func (m Money) CurrencyCode() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.amount.CurrencyCode(), nil
}

// String returns a developer-friendly string representation of the money
// value, including the currency code.
//
// Example: "123.45 USD". This uses the full internal precision.
//
// Returns string which is the formatted money value.
// Returns error when the Money value contains an existing error.
func (m Money) String() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.amount.String(), nil
}

// Format returns a locale-aware, formatted string suitable for display to
// end-users. It includes the correct currency symbol, grouping, and decimal
// separators for the given locale.
//
// Takes localeID (string) which specifies the locale for formatting, such as
// "en-US" or "de-DE".
//
// Returns string which is the formatted currency value, for example
// "$1,234.57" for locale "en-US".
// Returns error when the Money value was created with an error.
//
// Safe for concurrent use; uses a read lock on the currency registry.
func (m Money) Format(localeID string) (string, error) {
	if m.err != nil {
		return "", m.err
	}

	currencyRegistryMutex.RLock()
	defer currencyRegistryMutex.RUnlock()

	locale := currency.NewLocale(localeID)
	formatter := currency.NewFormatter(locale)
	return formatter.Format(m.amount.Round()), nil
}

// DefaultFormat returns the money value as a string using en_GB locale settings.
//
// Returns string which is the formatted monetary value.
//
// Safe for concurrent use; protected by a read lock on the currency registry.
func (m Money) DefaultFormat() string {
	if m.err != nil {
		return ZeroMoney(DefaultCurrencyCode).DefaultFormat()
	}
	currencyRegistryMutex.RLock()
	defer currencyRegistryMutex.RUnlock()
	locale := currency.NewLocale("en_GB")
	formatter := currency.NewFormatter(locale)
	return formatter.Format(m.amount.Round())
}

// MustNumber returns the numeric string representation of the money value.
// It is the panicking version of Number.
//
// Returns string which is the numeric representation.
//
// Panics when Number returns an error.
func (m Money) MustNumber() string {
	s, err := m.Number()
	if err != nil {
		panic(err)
	}
	return s
}

// MustRoundedNumber is the panicking version of RoundedNumber.
//
// Returns string which is the rounded monetary value.
//
// Panics when RoundedNumber returns an error.
func (m Money) MustRoundedNumber() string {
	s, err := m.RoundedNumber()
	if err != nil {
		panic(err)
	}
	return s
}

// MustFormattedNumber is the panicking version of FormattedNumber.
//
// Returns string which is the formatted number without the currency symbol.
//
// Panics when FormattedNumber returns an error.
func (m Money) MustFormattedNumber() string {
	s, err := m.FormattedNumber()
	if err != nil {
		panic(err)
	}
	return s
}

// MustString returns the formatted money value as a string.
//
// Returns string which is the formatted representation of the money value.
//
// Panics if the underlying String method returns an error.
func (m Money) MustString() string {
	s, err := m.String()
	if err != nil {
		panic(err)
	}
	return s
}

// MustFormat is the panicking version of Format.
//
// Takes localeID (string) which specifies the locale for formatting.
//
// Returns string which is the formatted money value.
//
// Panics when formatting fails.
func (m Money) MustFormat(localeID string) string {
	s, err := m.Format(localeID)
	if err != nil {
		panic(err)
	}
	return s
}

// MarshalJSON implements the json.Marshaler interface.
//
// Returns []byte which contains the JSON-encoded amount.
// Returns error when the Money has a stored error or encoding fails.
func (m Money) MarshalJSON() ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.amount.MarshalJSON()
}

// moneyJSON is the struct used for JSON object unmarshalling.
type moneyJSON struct {
	// Number holds the numeric value from JSON; required field that cannot be nil.
	Number any `json:"number"`

	// Currency is the ISO 4217 currency code; must not be empty.
	Currency string `json:"currency"`
}

// UnmarshalJSON implements the json.Unmarshaler interface.
//
// It handles both object format ({"number": "10.99", "currency": "USD"}) and
// simple string format ("10.99") which assumes DefaultCurrencyCode. Currency
// codes are case-insensitive and the number field accepts strings or numbers.
//
// Takes data ([]byte) which contains the JSON-encoded money value.
//
// Returns error when the receiver is nil, required fields are missing, or the
// value cannot be parsed.
func (m *Money) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("money: UnmarshalJSON on nil Money pointer")
	}

	var parsedMoney moneyJSON
	if err := json.Unmarshal(data, &parsedMoney); err == nil {
		return m.unmarshalJSONObject(parsedMoney)
	}

	var strValue string
	if err := json.Unmarshal(data, &strValue); err == nil {
		return m.unmarshalJSONString(strValue)
	}

	m.err = errors.New("money value must be a JSON object or a string")
	return m.err
}

// unmarshalJSONObject handles unmarshalling from a JSON object with number and
// currency fields.
//
// Takes parsedMoney (moneyJSON) which contains the parsed JSON fields.
//
// Returns error when the currency code is missing, the number field is missing,
// or the amount cannot be parsed.
func (m *Money) unmarshalJSONObject(parsedMoney moneyJSON) error {
	if parsedMoney.Currency == "" {
		m.err = errors.New("money: missing currency code in JSON object")
		return m.err
	}
	if parsedMoney.Number == nil {
		m.err = errors.New("money: missing number field in JSON object")
		return m.err
	}

	numberString, err := extractNumberString(parsedMoney.Number)
	if err != nil {
		m.err = err
		return m.err
	}

	code := strings.ToUpper(parsedMoney.Currency)
	newAmount, parseErr := currency.NewAmount(numberString, code)
	if parseErr != nil {
		m.err = fmt.Errorf("failed to parse money object from number '%s' and currency '%s': %w", numberString, code, parseErr)
		return m.err
	}
	m.amount = newAmount
	m.err = nil
	return nil
}

// unmarshalJSONString handles unmarshalling from a plain JSON string value.
//
// Takes strValue (string) which is the JSON string to parse as a money amount.
//
// Returns error when the string cannot be parsed as a valid money amount.
func (m *Money) unmarshalJSONString(strValue string) error {
	if strValue == "" {
		*m = ZeroMoney(DefaultCurrencyCode)
		return nil
	}
	newAmount, parseErr := currency.NewAmount(strValue, DefaultCurrencyCode)
	if parseErr != nil {
		m.err = fmt.Errorf("failed to parse money string '%s': %w", strValue, parseErr)
		return m.err
	}
	m.amount = newAmount
	m.err = nil
	return nil
}

// Value implements the driver.Valuer interface for database serialisation.
//
// Returns driver.Value which contains the serialised monetary amount.
// Returns error when the Money instance has a prior error or serialisation
// fails.
func (m Money) Value() (driver.Value, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.amount.Value()
}

// Scan implements the sql.Scanner interface for database reads.
//
// Takes source (any) which is the value from the database driver.
//
// Returns error when the receiver is nil or the source cannot be parsed.
func (m *Money) Scan(source any) error {
	if m == nil {
		return errors.New("money: Scan on nil Money pointer")
	}

	var underlyingAmount currency.Amount
	err := underlyingAmount.Scan(source)

	if underlyingAmount.IsZero() && underlyingAmount.CurrencyCode() == "" {
		*m = ZeroMoney(DefaultCurrencyCode)
		return nil
	}

	if err != nil {
		m.err = err
		return err
	}

	m.amount = underlyingAmount
	m.err = nil
	return nil
}

// extractNumberString converts a number field from various types to a string.
//
// Takes number (any) which is the value to convert.
//
// Returns string which is the converted number as text.
// Returns error when the number type is not supported.
func extractNumberString(number any) (string, error) {
	switch v := number.(type) {
	case string:
		return v, nil
	case stdjson.Number:
		return v.String(), nil
	case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprint(v), nil
	default:
		return "", fmt.Errorf("money: unsupported type for 'number' field in JSON object: %T", number)
	}
}
