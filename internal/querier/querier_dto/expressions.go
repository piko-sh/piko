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

package querier_dto

// Expression is the typed representation of a SQL expression that flows
// between engine adapters and the domain type resolver. Each concrete type
// corresponds to a distinct SQL expression form (column reference, function
// call, binary operation, etc.).
//
// Engine adapters construct concrete Expression values during query analysis.
// The domain type resolver dispatches on the concrete type via a type switch
// to perform bottom-up type inference.
//
// A nil Expression represents SQL NULL, resolved as TypeCategoryUnknown with
// nullable=true.
type Expression interface {
	// expressionKind returns a string tag identifying the concrete expression form.
	expressionKind() string
}

// ColumnRefExpression represents a reference to a table column.
type ColumnRefExpression struct {
	// TableAlias holds the table alias or name qualifying this column reference.
	TableAlias string

	// ColumnName holds the name of the referenced column.
	ColumnName string
}

// expressionKind identifies this expression as a column
// reference.
//
// Returns string which holds the expression kind tag.
func (*ColumnRefExpression) expressionKind() string { return "column_ref" }

// FunctionCallExpression represents a SQL function invocation.
type FunctionCallExpression struct {
	// FilterExpression holds the optional FILTER (WHERE ...) clause for this function call.
	FilterExpression Expression

	// FunctionName holds the name of the SQL function being invoked.
	FunctionName string

	// Schema holds the schema qualifier for the function, if specified.
	Schema string

	// Arguments holds the list of expressions passed as arguments to the function.
	Arguments []Expression
}

// expressionKind identifies this expression as a
// function call.
//
// Returns string which holds the expression kind tag.
func (*FunctionCallExpression) expressionKind() string { return "function_call" }

// CoalesceExpression represents a COALESCE(...) call with special
// nullability semantics (not-null if any argument is not-null).
type CoalesceExpression struct {
	// Arguments holds the list of expressions passed to COALESCE.
	Arguments []Expression
}

// expressionKind identifies this expression as a
// coalesce call.
//
// Returns string which holds the expression kind tag.
func (*CoalesceExpression) expressionKind() string { return "coalesce" }

// CastExpression represents a CAST(expression AS type) expression.
type CastExpression struct {
	// Inner holds the expression being cast.
	Inner Expression

	// TypeName holds the target type name in the CAST.
	TypeName string
}

// expressionKind identifies this expression as a cast.
//
// Returns string which holds the expression kind tag.
func (*CastExpression) expressionKind() string { return "cast" }

// LiteralExpression represents a SQL literal value (integer, text, etc.).
type LiteralExpression struct {
	// TypeName holds the inferred type name of this literal.
	TypeName string
}

// expressionKind identifies this expression as a literal.
//
// Returns string which holds the expression kind tag.
func (*LiteralExpression) expressionKind() string { return "literal" }

// BinaryOpExpression represents an arithmetic or string concatenation
// operator: +, -, *, /, %, ||.
type BinaryOpExpression struct {
	// Left holds the left-hand operand of the binary operation.
	Left Expression

	// Right holds the right-hand operand of the binary operation.
	Right Expression

	// Operator holds the operator symbol (e.g. "+", "||").
	Operator string
}

// expressionKind identifies this expression as a binary
// operation.
//
// Returns string which holds the expression kind tag.
func (*BinaryOpExpression) expressionKind() string { return "binary_op" }

// ComparisonExpression represents a comparison operator that produces a
// boolean result: =, <>, !=, <, >, <=, >=, LIKE, GLOB, REGEXP, MATCH.
type ComparisonExpression struct {
	// Left holds the left-hand operand of the comparison.
	Left Expression

	// Right holds the right-hand operand of the comparison.
	Right Expression

	// Operator holds the comparison operator (e.g. "=", "LIKE").
	Operator string
}

// expressionKind identifies this expression as a
// comparison.
//
// Returns string which holds the expression kind tag.
func (*ComparisonExpression) expressionKind() string { return "comparison" }

// IsNullExpression represents IS NULL or IS NOT NULL.
type IsNullExpression struct {
	// Inner holds the expression being tested for nullness.
	Inner Expression

	// Negated indicates whether this is IS NOT NULL (true) rather than IS NULL (false).
	Negated bool
}

// expressionKind identifies this expression as an IS
// NULL test.
//
// Returns string which holds the expression kind tag.
func (*IsNullExpression) expressionKind() string { return "is_null" }

// InListExpression represents expression IN (value, ...).
type InListExpression struct {
	// Inner holds the expression being tested for membership.
	Inner Expression

	// Values holds the list of expressions in the IN clause.
	Values []Expression
}

// expressionKind identifies this expression as an IN
// list test.
//
// Returns string which holds the expression kind tag.
func (*InListExpression) expressionKind() string { return "in_list" }

// BetweenExpression represents expression BETWEEN low AND high.
type BetweenExpression struct {
	// Inner holds the expression being range-tested.
	Inner Expression

	// Low holds the lower bound of the BETWEEN range.
	Low Expression

	// High holds the upper bound of the BETWEEN range.
	High Expression
}

// expressionKind identifies this expression as a BETWEEN
// range test.
//
// Returns string which holds the expression kind tag.
func (*BetweenExpression) expressionKind() string { return "between" }

// LogicalOpExpression represents AND or OR with two or more operands.
type LogicalOpExpression struct {
	// Operator holds the logical operator ("AND" or "OR").
	Operator string

	// Operands holds the list of boolean sub-expressions joined by the operator.
	Operands []Expression
}

// expressionKind identifies this expression as a logical
// operation.
//
// Returns string which holds the expression kind tag.
func (*LogicalOpExpression) expressionKind() string { return "logical_op" }

// UnaryOpExpression represents a unary operator: -, +, ~, NOT.
type UnaryOpExpression struct {
	// Operand holds the expression the unary operator is applied to.
	Operand Expression

	// Operator holds the unary operator symbol (e.g. "-", "NOT").
	Operator string
}

// expressionKind identifies this expression as a unary
// operation.
//
// Returns string which holds the expression kind tag.
func (*UnaryOpExpression) expressionKind() string { return "unary_op" }

// CaseWhenExpression represents a CASE WHEN conditional
// expression.
type CaseWhenExpression struct {
	// ElseResult holds the optional ELSE branch expression, or nil if absent.
	ElseResult Expression

	// Branches holds the ordered list of WHEN/THEN pairs.
	Branches []CaseWhenBranch
}

// expressionKind identifies this expression as a CASE
// WHEN.
//
// Returns string which holds the expression kind tag.
func (*CaseWhenExpression) expressionKind() string { return "case_when" }

// CaseWhenBranch holds a single WHEN condition THEN result pair.
type CaseWhenBranch struct {
	// Condition holds the boolean expression for this WHEN clause.
	Condition Expression

	// Result holds the expression returned when Condition is true.
	Result Expression
}

// ExistsExpression represents an EXISTS(...) subquery test.
type ExistsExpression struct {
	// InnerQuery holds the parsed subquery inside the EXISTS clause.
	InnerQuery *RawQueryAnalysis
}

// expressionKind identifies this expression as an EXISTS
// test.
//
// Returns string which holds the expression kind tag.
func (*ExistsExpression) expressionKind() string { return "exists" }

// WindowFunctionExpression wraps a function call with an OVER clause.
type WindowFunctionExpression struct {
	// Function holds the underlying function call that the window applies to.
	Function *FunctionCallExpression
}

// expressionKind identifies this expression as a window
// function.
//
// Returns string which holds the expression kind tag.
func (*WindowFunctionExpression) expressionKind() string { return "window_function" }

// ScalarSubqueryExpression represents a scalar subquery in expression position,
// e.g. (SELECT max(x) FROM t).
//
// The result is always nullable because the subquery might return zero rows.
type ScalarSubqueryExpression struct {
	// InnerQuery holds the parsed subquery that produces the scalar value.
	InnerQuery *RawQueryAnalysis
}

// expressionKind identifies this expression as a scalar
// subquery.
//
// Returns string which holds the expression kind tag.
func (*ScalarSubqueryExpression) expressionKind() string { return "scalar_subquery" }

// ArraySubscriptExpression represents an array element access: arr[n].
// Resolved as the element type of the array, always nullable.
type ArraySubscriptExpression struct {
	// Array holds the expression producing the array value.
	Array Expression

	// Index holds the expression producing the subscript index.
	Index Expression
}

// expressionKind identifies this expression as an array
// subscript.
//
// Returns string which holds the expression kind tag.
func (*ArraySubscriptExpression) expressionKind() string { return "array_subscript" }

// LambdaExpression represents a DuckDB lambda expression used as a
// function argument (e.g. x -> x + 1).
//
// The type of the lambda is the type of its body expression.
type LambdaExpression struct {
	// Body holds the expression that forms the lambda body.
	Body Expression

	// Parameters holds the names of the lambda parameters.
	Parameters []string
}

// expressionKind identifies this expression as a lambda.
//
// Returns string which holds the expression kind tag.
func (*LambdaExpression) expressionKind() string { return "lambda" }

// StructFieldAccessExpression represents accessing a named field on a STRUCT
// expression (e.g. s.field_name in DuckDB).
type StructFieldAccessExpression struct {
	// Struct holds the expression producing the struct value.
	Struct Expression

	// FieldName holds the name of the field being accessed.
	FieldName string
}

// expressionKind identifies this expression as a struct
// field access.
//
// Returns string which holds the expression kind tag.
func (*StructFieldAccessExpression) expressionKind() string { return "struct_field_access" }

// UnknownExpression is the fallback for expression forms that the engine
// adapter cannot structurally represent. Resolved as TypeCategoryUnknown
// with nullable=true.
type UnknownExpression struct{}

// expressionKind identifies this expression as unknown.
//
// Returns string which holds the expression kind tag.
func (*UnknownExpression) expressionKind() string { return "unknown" }
