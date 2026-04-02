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

package binder

import (
	"piko.sh/piko/internal/ast/ast_domain"
)

// isPathExpression recursively validates that an AST expression tree only
// contains nodes that are valid for a struct path. A valid path consists of a
// chain of identifiers, member accesses, and index accesses with literal
// integer or string indices.
//
// This validates that form keys cannot contain arbitrary computations,
// operators, function calls, or other expressions that have no meaning as a
// destination for a value.
//
// Allowed AST node types:
//   - *ast_domain.Identifier: e.g., "Name"
//   - *ast_domain.MemberExpression: e.g., "User.Name"
//   - *ast_domain.IndexExpression: e.g., "Items[0]" or "Items[\"key\"]"
//
// All other node types (e.g., BinaryExpr, CallExpr, StringLiteral) are
// forbidden.
//
// Takes expression (ast_domain.Expression) which is the AST node to
// validate.
//
// Returns bool which is true if the expression represents a valid
// path, and false otherwise.
func isPathExpression(expression ast_domain.Expression) bool {
	if expression == nil {
		return false
	}

	switch node := expression.(type) {
	case *ast_domain.Identifier:
		return true

	case *ast_domain.MemberExpression:
		return isPathExpression(node.Base)

	case *ast_domain.IndexExpression:
		_, isIntLiteral := node.Index.(*ast_domain.IntegerLiteral)
		_, isStringLiteral := node.Index.(*ast_domain.StringLiteral)
		return isPathExpression(node.Base) && (isIntLiteral || isStringLiteral)

	default:
		return false
	}
}
