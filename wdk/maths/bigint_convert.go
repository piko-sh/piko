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
	"strconv"

	"piko.sh/piko/internal/json"
)

// String returns the decimal text form of the big integer.
//
// Returns string which is the value as a decimal number.
// Returns error when the big integer is in an error state.
func (b BigInt) String() (string, error) {
	if b.err != nil {
		return "", b.err
	}
	return b.value.String(), nil
}

// Float64 returns the float64 value of the BigInt.
//
// This conversion may lose precision for very large numbers.
//
// Returns float64 which is the converted value.
// Returns error when the BigInt is in an error state or conversion fails.
func (b BigInt) Float64() (float64, error) {
	if b.err != nil {
		return 0, b.err
	}
	s, err := b.String()
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

// Int64 returns the value as an int64.
//
// Returns int64 which is the numeric value.
// Returns error when the BigInt is in an error state or the value is too large
// to fit in an int64.
func (b BigInt) Int64() (int64, error) {
	if b.err != nil {
		return 0, b.err
	}
	if !b.value.IsInt64() {
		return 0, fmt.Errorf("maths: bigint value %s overflows int64", b.value.String())
	}
	return b.value.Int64(), nil
}

// MustString returns the string representation of the BigInt.
//
// Returns string which is the decimal representation of the value.
//
// Panics if an error occurs during string conversion.
func (b BigInt) MustString() string {
	s, err := b.String()
	if err != nil {
		panic(err)
	}
	return s
}

// MustFloat64 returns the float64 representation of the BigInt.
//
// Be aware that this conversion can result in a loss of precision.
//
// Returns float64 which is the numeric value of the BigInt.
//
// Panics when the conversion fails.
func (b BigInt) MustFloat64() float64 {
	f, err := b.Float64()
	if err != nil {
		panic(err)
	}
	return f
}

// MustInt64 returns the int64 representation of the BigInt.
//
// Returns int64 which is the value of the BigInt.
//
// Panics when the value overflows int64.
func (b BigInt) MustInt64() int64 {
	i, err := b.Int64()
	if err != nil {
		panic(err)
	}
	return i
}

// MarshalJSON implements the json.Marshaler interface.
// It marshals the bigint as a JSON string to preserve full precision.
//
// Returns []byte which contains the JSON-encoded string representation.
// Returns error when the BigInt holds an error or string conversion fails.
func (b BigInt) MarshalJSON() ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}
	s, err := b.String()
	if err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It can unmarshal from a JSON string or a JSON number.
//
// Takes data ([]byte) which contains the JSON-encoded value to parse.
//
// Returns error when the receiver is nil or the data is not a valid string
// or number.
func (b *BigInt) UnmarshalJSON(data []byte) error {
	if b == nil {
		return errors.New("maths: UnmarshalJSON on nil BigInt pointer")
	}

	var strValue string
	if err := json.Unmarshal(data, &strValue); err == nil {
		*b = NewBigIntFromString(strValue)
		return b.err
	}

	var numValue stdjson.Number
	if err := json.Unmarshal(data, &numValue); err == nil {
		*b = NewBigIntFromString(numValue.String())
		return b.err
	}

	return errors.New("maths: bigint must be a JSON string or a whole number")
}

// Value implements the driver.Valuer interface for database serialisation.
// It returns the bigint as a string for storage in a high-precision database
// column.
//
// Returns driver.Value which contains the string representation.
// Returns error when the BigInt holds a previous error.
func (b BigInt) Value() (driver.Value, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.String()
}

// Scan implements the sql.Scanner interface for database deserialisation.
//
// Takes source (interface{}) which provides the database value to scan. Accepts
// string, []byte, int64, or nil types.
//
// Returns error when b is nil or source is an unsupported type.
func (b *BigInt) Scan(source any) error {
	if b == nil {
		return errors.New("maths: Scan on nil BigInt pointer")
	}

	var temp BigInt
	switch v := source.(type) {
	case string:
		temp = NewBigIntFromString(v)
	case []byte:
		temp = NewBigIntFromString(string(v))
	case int64:
		temp = NewBigIntFromInt(v)
	case nil:
		temp = ZeroBigInt()
	default:
		return fmt.Errorf("maths: cannot scan type %T into BigInt", source)
	}

	*b = temp
	return b.err
}
