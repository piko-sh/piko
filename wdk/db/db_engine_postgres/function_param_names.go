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

package db_engine_postgres

const (
	// paramNameX is the canonical parameter name for a generic scalar input.
	paramNameX = "x"

	// paramNameY is the canonical parameter name for a second scalar input.
	paramNameY = "y"

	// paramNameA is the canonical parameter name for the first of two operands.
	paramNameA = "a"

	// paramNameB is the canonical parameter name for the second of two operands.
	paramNameB = "b"

	// paramNameN is the canonical parameter name for a count or index.
	paramNameN = "n"

	// paramNameValue is the canonical parameter name for a value argument.
	paramNameValue = "value"

	// paramNameLength is the canonical parameter name for a length argument.
	paramNameLength = "length"

	// paramNameString is the canonical parameter name for a string argument.
	paramNameString = "string"

	// paramNameStart is the canonical parameter name for a starting offset.
	paramNameStart = "start"

	// paramNameFormat is the canonical parameter name for a format string.
	paramNameFormat = "format"

	// paramNameField is the canonical parameter name for an extract field.
	paramNameField = "field"

	// paramNameSource is the canonical parameter name for a source value.
	paramNameSource = "source"

	// paramNameTarget is the canonical parameter name for a target value.
	paramNameTarget = "target"

	// paramNamePath is the canonical parameter name for a path argument.
	paramNamePath = "path"

	// paramNameNewValue is the canonical parameter name for a replacement value.
	paramNameNewValue = "new_value"

	// paramNameArray is the canonical parameter name for an array argument.
	paramNameArray = "array"

	// paramNameElement is the canonical parameter name for an element argument.
	paramNameElement = "element"

	// paramNameDelimiter is the canonical parameter name for a delimiter.
	paramNameDelimiter = "delimiter"

	// paramNameCount is the canonical parameter name for a count argument.
	paramNameCount = "count"

	// paramNameExpression is the canonical parameter name for an expression.
	paramNameExpression = "expression"

	// paramNameJSON is the canonical parameter name for a JSON or JSONB value.
	paramNameJSON = "json"

	// paramNameTimestamp is the canonical parameter name for a timestamp value.
	paramNameTimestamp = "timestamp"

	// paramNameSubstring is the canonical parameter name for a substring.
	paramNameSubstring = "substring"

	// paramNameYear is the canonical parameter name for a year component.
	paramNameYear = "year"

	// paramNameMonth is the canonical parameter name for a month component.
	paramNameMonth = "month"

	// paramNameDay is the canonical parameter name for a day component.
	paramNameDay = "day"

	// paramNameHour is the canonical parameter name for an hour component.
	paramNameHour = "hour"

	// paramNameMin is the canonical parameter name for a minute component.
	paramNameMin = "min"

	// paramNameSec is the canonical parameter name for a seconds component.
	paramNameSec = "sec"

	// paramNameText is the canonical parameter name for a text argument.
	paramNameText = "text"

	// funcNameToChar is the canonical name for the to_char conversion function.
	funcNameToChar = "to_char"
)
