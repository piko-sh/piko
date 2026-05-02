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

package db_driver_d1

import (
	"database/sql/driver"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	// errNullParamUnsupported reports that a nil bound parameter was supplied.
	//
	// The D1 HTTP API binds parameters through a JSON array of strings, which cannot carry a
	// JSON null, so a genuine SQL NULL cannot be transmitted as a bound parameter. Callers
	// can match this with errors.Is and encode NULL directly in the statement text instead.
	errNullParamUnsupported = errors.New(
		"db_driver_d1: NULL bound parameters are not supported by the D1 string wire format",
	)
)

// stringifyNamedParams converts a slice of driver.NamedValue arguments to the []string
// format required by the D1 HTTP API.
//
// Takes args ([]driver.NamedValue) which are the named parameter values to convert.
//
// Returns []string which contains the stringified parameters.
// Returns error which wraps errNullParamUnsupported when any value is nil.
func stringifyNamedParams(args []driver.NamedValue) ([]string, error) {
	result := make([]string, len(args))
	for i, arg := range args {
		stringified, err := stringifyValue(arg.Value)
		if err != nil {
			return nil, fmt.Errorf("db_driver_d1: parameter %d: %w", arg.Ordinal, err)
		}
		result[i] = stringified
	}
	return result, nil
}

// stringifyValue converts a single value to its string representation for D1.
//
// The D1 HTTP API binds parameters through a JSON array of strings
// (cloudflare.QueryD1DatabaseParams.Parameters is []string), so every value must be
// rendered as a string. Each supported type maps to a stable, reversible textual form. A
// string is passed through unchanged. int64 and float64 use their canonical decimal
// forms. A bool maps to "1" or "0", matching SQLite's integer boolean storage. A []byte
// is base64-encoded with standard padding. A time.Time is rendered as
// v.UTC().Format(time.RFC3339Nano), preserving sub-second precision and normalising to
// UTC so reads can round-trip, matching SQLite's text date handling.
//
// The string-only wire format cannot carry a JSON null, so a nil value cannot be
// transmitted as a genuine SQL NULL. Rather than silently substituting the empty string
// (which SQLite would store as empty text, corrupting IS NULL and COALESCE semantics), a
// nil value returns errNullParamUnsupported. Callers that require true NULL semantics
// must encode it directly in the SQL statement text rather than as a bound parameter.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the stringified representation.
// Returns error which is errNullParamUnsupported when value is nil.
func stringifyValue(value any) (string, error) {
	if value == nil {
		return "", errNullParamUnsupported
	}

	switch v := value.(type) {
	case string:
		return v, nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), nil
	case bool:
		if v {
			return "1", nil
		}
		return "0", nil
	case []byte:
		return base64.StdEncoding.EncodeToString(v), nil
	case time.Time:
		return v.UTC().Format(time.RFC3339Nano), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}
