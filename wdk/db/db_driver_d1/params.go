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
	"fmt"
	"strconv"
	"time"
)

// stringifyNamedParams converts a slice of driver.NamedValue arguments to the
// []string format required by the D1 HTTP API.
//
// Takes args ([]driver.NamedValue) which are the named parameter values to
// convert.
//
// Returns []string which contains the stringified parameters.
func stringifyNamedParams(args []driver.NamedValue) []string {
	result := make([]string, len(args))
	for i, arg := range args {
		result[i] = stringifyValue(arg.Value)
	}
	return result
}

// stringifyValue converts a single value to its string representation for D1.
//
// Takes value (any) which is the value to convert.
//
// Returns string which is the stringified representation.
func stringifyValue(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case string:
		return v
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case []byte:
		return base64.StdEncoding.EncodeToString(v)
	case time.Time:
		return strconv.FormatInt(v.Unix(), 10)
	default:
		return fmt.Sprintf("%v", v)
	}
}
