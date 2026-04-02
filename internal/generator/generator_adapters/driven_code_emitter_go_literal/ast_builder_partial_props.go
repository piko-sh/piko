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

package driven_code_emitter_go_literal

import (
	goast "go/ast"
	"go/token"
)

// buildPropToStringExpr generates a Go AST expression that converts a prop
// field value to its string representation, suitable for use as a query
// parameter value.
//
// Takes fieldAccess (goast.Expr) which is the expression accessing the prop
// field (e.g. props_key.FieldName).
// Takes baseType (string) which is the Go base type name of the field.
//
// Returns goast.Expr which converts the field value to a string using the
// appropriate strconv function, or the field directly for string types.
func buildPropToStringExpr(fieldAccess goast.Expr, baseType string) goast.Expr {
	switch baseType {
	case "int":
		return callStrconv(strconvItoa, fieldAccess)
	case "int8", "int16", "int32":
		return callStrconv(strconvFormatInt,
			&goast.CallExpr{Fun: cachedIdent(Int64TypeName), Args: []goast.Expr{fieldAccess}},
			intLit(numericBaseDecimal))
	case "int64":
		return callStrconv(strconvFormatInt, fieldAccess, intLit(numericBaseDecimal))
	case "uint", "uint8", "uint16", "uint32":
		return callStrconv(strconvFormatUint,
			&goast.CallExpr{Fun: cachedIdent("uint64"), Args: []goast.Expr{fieldAccess}},
			intLit(numericBaseDecimal))
	case "uint64":
		return callStrconv(strconvFormatUint, fieldAccess, intLit(numericBaseDecimal))
	case "float32":
		return callStrconv(strconvFormatFloat,
			&goast.CallExpr{Fun: cachedIdent("float64"), Args: []goast.Expr{fieldAccess}},
			&goast.BasicLit{Kind: token.CHAR, Value: "'f'"},
			intLit(-1),
			intLit(bitSize32))
	case "float64":
		return callStrconv(strconvFormatFloat,
			fieldAccess,
			&goast.BasicLit{Kind: token.CHAR, Value: "'f'"},
			intLit(-1),
			intLit(bitSize64))
	case "bool":
		return callStrconv(strconvFormatBool, fieldAccess)
	default:
		return fieldAccess
	}
}
