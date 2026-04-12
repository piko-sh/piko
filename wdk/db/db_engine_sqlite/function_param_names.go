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

package db_engine_sqlite

const (
	// paramNameX is the canonical name for a first numeric or generic argument in SQLite
	// built-in function signatures.
	paramNameX = "x"

	// paramNameY is the canonical name for a second numeric or generic argument in SQLite
	// built-in function signatures.
	paramNameY = "y"

	// paramNameStr is the canonical name for a string argument.
	paramNameStr = "str"

	// paramNameJSON is the canonical name for a JSON-text argument.
	paramNameJSON = "json"

	// paramNameExpression is the canonical name for a window function expression argument.
	paramNameExpression = "expression"

	// paramNameTable is the canonical name for a table-reference argument used by FTS5 and
	// R-Tree auxiliary functions.
	paramNameTable = "table"

	// paramNameFormat is the canonical name for a format-string argument.
	paramNameFormat = "format"

	// paramNameValue is the canonical name for a generic value argument.
	paramNameValue = "value"
)
