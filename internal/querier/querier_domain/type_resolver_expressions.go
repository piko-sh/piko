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

package querier_domain

import (
	"errors"
	"fmt"
	"strings"

	"piko.sh/piko/internal/querier/querier_dto"
)

// booleanNotNull is the pre-built SQL type descriptor for a non-nullable boolean column.
var booleanNotNull = querier_dto.SQLType{
	Category:   querier_dto.TypeCategoryBoolean,
	EngineName: querier_dto.CanonicalBoolean,
}

// resolveExpressionType infers the SQL type and
// nullability of a typed expression from the engine
// adapter. Dispatches on the concrete expression type
// via a type switch for compile-time safety.
//
// Takes expression (querier_dto.Expression) which
// specifies the expression to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for column and CTE lookups.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the expression modifies data.
//
// Returns querier_dto.SQLType which holds the inferred
// SQL type of the expression.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates a resolution failure.
func (r *typeResolver) resolveExpressionType(
	expression querier_dto.Expression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, nil
	}

	feature := expressionFeature(expression)
	if feature != 0 && !r.engine.SupportedExpressions().Has(feature) {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			fmt.Errorf("unsupported expression: %s", feature.String())
	}

	if isBooleanExpression(expression) {
		return booleanNotNull, false, nil
	}

	switch expr := expression.(type) {
	case *querier_dto.ColumnRefExpression:
		return r.resolveColumnRefExpression(expr, scope)
	case *querier_dto.FunctionCallExpression:
		return r.resolveFunctionCallExpression(expr, scope, dataModifying)
	case *querier_dto.CoalesceExpression:
		return r.resolveCoalesceExpression(expr, scope, dataModifying)
	case *querier_dto.CastExpression:
		return r.resolveCastExpression(expr, scope, dataModifying)
	case *querier_dto.LiteralExpression:
		return r.resolveLiteralExpression(expr)
	case *querier_dto.BinaryOpExpression:
		return r.resolveBinaryOpExpression(expr, scope, dataModifying)
	case *querier_dto.UnaryOpExpression:
		return r.resolveUnaryOpExpression(expr, scope, dataModifying)
	case *querier_dto.CaseWhenExpression:
		return r.resolveCaseWhenExpression(expr, scope, dataModifying)
	case *querier_dto.WindowFunctionExpression:
		return r.resolveWindowFunctionExpression(expr, scope, dataModifying)
	case *querier_dto.ScalarSubqueryExpression:
		return r.resolveScalarSubqueryExpression(expr, scope, dataModifying)
	case *querier_dto.ArraySubscriptExpression:
		return r.resolveArraySubscriptExpression(expr, scope, dataModifying)
	case *querier_dto.LambdaExpression:
		return r.resolveLambdaExpression(expr, scope, dataModifying)
	case *querier_dto.StructFieldAccessExpression:
		return r.resolveStructFieldAccessExpression(expr, scope, dataModifying)
	default:
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, nil
	}
}

// resolveColumnRefExpression resolves a column
// reference expression by looking up the column in the
// scope chain and returning its type and nullability.
//
// Takes expression (*querier_dto.ColumnRefExpression)
// which specifies the column reference to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for column lookups.
//
// Returns querier_dto.SQLType which holds the column's
// SQL type.
// Returns bool which indicates whether the column is
// nullable.
// Returns error which indicates Q030 for nil references
// or a scope resolution failure.
func (*typeResolver) resolveColumnRefExpression(
	expression *querier_dto.ColumnRefExpression,
	scope *scopeChain,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil column reference expression during type resolution")
	}
	column, _, err := scope.ResolveColumn(expression.TableAlias, expression.ColumnName)
	if err != nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, err
	}
	if column == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			fmt.Errorf("%s: nil column resolved for %s", querier_dto.CodeInternalNilGuard, expression.ColumnName)
	}
	return column.SQLType, column.Nullable, nil
}

// resolveFunctionCallExpression resolves a function
// call by performing overload resolution against the
// function resolver and inferring nullability from the
// function's nullable behaviour and argument
// nullability.
//
// Takes expression (*querier_dto.FunctionCallExpression)
// which specifies the function call to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving argument expressions.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the function modifies data.
//
// Returns querier_dto.SQLType which holds the
// function's return type.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates a resolution or
// overload failure.
func (r *typeResolver) resolveFunctionCallExpression(
	expression *querier_dto.FunctionCallExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil function call expression during type resolution")
	}

	argumentTypes := make([]querier_dto.SQLType, 0, len(expression.Arguments))
	anyArgumentNullable := false

	for _, argument := range expression.Arguments {
		if argument == nil {
			argumentTypes = append(argumentTypes, querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown})
			anyArgumentNullable = true
			continue
		}
		argumentType, argumentNullable, _ := r.resolveExpressionType(argument, scope, dataModifying)
		argumentTypes = append(argumentTypes, argumentType)
		if argumentNullable {
			anyArgumentNullable = true
		}
	}

	match, resolveError := r.functionResolver.Resolve(expression.FunctionName, expression.Schema, argumentTypes)
	if resolveError != nil {
		*dataModifying = true
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			fmt.Errorf("%s: %s", resolveError.Code, resolveError.Message)
	}
	if match == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			fmt.Errorf("%s: nil function match for %s during type resolution", querier_dto.CodeInternalNilGuard, expression.FunctionName)
	}

	if match.dataAccess != querier_dto.DataAccessReadOnly {
		*dataModifying = true
	}

	nullable := true
	switch match.nullableBehaviour {
	case querier_dto.FunctionNullableNeverNull:
		nullable = false
	case querier_dto.FunctionNullableReturnsNullOnNull:
		nullable = anyArgumentNullable
	case querier_dto.FunctionNullableCalledOnNull:
		nullable = true
	}

	return match.returnType, nullable, nil
}

// resolveCoalesceExpression resolves a COALESCE
// expression by computing the common supertype of all
// arguments. The result is nullable only if every
// argument is nullable.
//
// Takes expression (*querier_dto.CoalesceExpression)
// which specifies the COALESCE to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving argument expressions.
// Takes dataModifying (*bool) which specifies a flag
// set to true if any argument modifies data.
//
// Returns querier_dto.SQLType which holds the common
// supertype of all arguments.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates Q030 for a nil
// expression.
func (r *typeResolver) resolveCoalesceExpression(
	expression *querier_dto.CoalesceExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil COALESCE expression during type resolution")
	}

	resultType := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	allNullable := true

	for _, argument := range expression.Arguments {
		if argument == nil {
			continue
		}
		argumentType, argumentNullable, _ := r.resolveExpressionType(argument, scope, dataModifying)
		resultType = r.commonSupertype(resultType, argumentType)
		if !argumentNullable {
			allNullable = false
		}
	}

	return resultType, allNullable, nil
}

// resolveCastExpression resolves a CAST expression by
// normalising the target type name through the engine
// adapter and preserving the inner expression's
// nullability.
//
// Takes expression (*querier_dto.CastExpression) which
// specifies the CAST to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving the inner expression.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the inner expression modifies data.
//
// Returns querier_dto.SQLType which holds the
// normalised target type.
// Returns bool which indicates whether the result is
// nullable (inherits from the inner expression).
// Returns error which indicates Q030 for a nil
// expression.
func (r *typeResolver) resolveCastExpression(
	expression *querier_dto.CastExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil cast expression during type resolution")
	}
	innerNullable := true
	if expression.Inner != nil {
		_, innerNullable, _ = r.resolveExpressionType(expression.Inner, scope, dataModifying)
	}

	if expression.TypeName != "" {
		return r.engine.NormaliseTypeName(expression.TypeName), innerNullable, nil
	}
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, innerNullable, nil
}

// resolveLiteralExpression resolves a literal expression by normalising its type
// name through the engine adapter. Literals are never nullable.
//
// Takes expression (*querier_dto.LiteralExpression) which specifies the literal to resolve.
//
// Returns querier_dto.SQLType which holds the normalised literal type.
// Returns bool which indicates whether the result is nullable (always false for literals).
// Returns error which indicates Q030 for a nil expression.
func (r *typeResolver) resolveLiteralExpression(
	expression *querier_dto.LiteralExpression,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil literal expression during type resolution")
	}
	if expression.TypeName != "" {
		return r.engine.NormaliseTypeName(expression.TypeName), false, nil
	}
	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, false, nil
}

// resolveBinaryOpExpression resolves a binary operator
// expression by inferring the result type from the
// operator and operand types. Special cases include
// string concatenation, JSON operators, and bitwise
// operators.
//
// Takes expression (*querier_dto.BinaryOpExpression)
// which specifies the binary operation to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving operand expressions.
// Takes dataModifying (*bool) which specifies a flag
// set to true if either operand modifies data.
//
// Returns querier_dto.SQLType which holds the inferred
// result type.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates Q030 for a nil
// expression.
func (r *typeResolver) resolveBinaryOpExpression(
	expression *querier_dto.BinaryOpExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil binary operator expression during type resolution")
	}

	leftType, leftNullable, _ := r.resolveExpressionType(expression.Left, scope, dataModifying)
	rightType, rightNullable, _ := r.resolveExpressionType(expression.Right, scope, dataModifying)
	nullable := leftNullable || rightNullable

	switch expression.Operator {
	case "||":
		return querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryText,
			EngineName: querier_dto.CanonicalText,
		}, nullable, nil
	case "->", "#>":
		if leftType.Category == querier_dto.TypeCategoryJSON {
			return leftType, true, nil
		}
		return querier_dto.SQLType{
			Category:   leftType.Category,
			EngineName: "json",
		}, true, nil
	case "->>", "#>>":
		return querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryText,
			EngineName: querier_dto.CanonicalText,
		}, true, nil
	case "&", "|", "<<", ">>":
		return querier_dto.SQLType{
			Category:   querier_dto.TypeCategoryInteger,
			EngineName: querier_dto.CanonicalInt8,
		}, nullable, nil
	default:
		return r.commonSupertype(leftType, rightType), nullable, nil
	}
}

// resolveUnaryOpExpression resolves a unary operator
// expression. NOT operators always produce a
// non-nullable boolean; other operators preserve the
// operand type.
//
// Takes expression (*querier_dto.UnaryOpExpression)
// which specifies the unary operation to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving the operand.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the operand modifies data.
//
// Returns querier_dto.SQLType which holds the inferred
// result type.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates Q030 for a nil
// expression.
func (r *typeResolver) resolveUnaryOpExpression(
	expression *querier_dto.UnaryOpExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil unary operator expression during type resolution")
	}

	if strings.EqualFold(expression.Operator, "NOT") {
		return booleanNotNull, false, nil
	}

	operandType, operandNullable, operandError := r.resolveExpressionType(expression.Operand, scope, dataModifying)
	return operandType, operandNullable, operandError
}

// resolveCaseWhenExpression resolves a CASE/WHEN
// expression by computing the common supertype across
// all branch results and the ELSE result. The result is
// nullable if any branch is nullable or the ELSE clause
// is absent.
//
// Takes expression (*querier_dto.CaseWhenExpression)
// which specifies the CASE expression to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving branch expressions.
// Takes dataModifying (*bool) which specifies a flag
// set to true if any branch modifies data.
//
// Returns querier_dto.SQLType which holds the common
// supertype of all branches.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates Q030 for a nil
// expression.
func (r *typeResolver) resolveCaseWhenExpression(
	expression *querier_dto.CaseWhenExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil CASE expression during type resolution")
	}

	resultType := querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
	nullable := expression.ElseResult == nil

	for _, branch := range expression.Branches {
		if branch.Result == nil {
			nullable = true
			continue
		}
		branchType, branchNullable, _ := r.resolveExpressionType(branch.Result, scope, dataModifying)
		resultType = r.commonSupertype(resultType, branchType)
		if branchNullable {
			nullable = true
		}
	}

	if expression.ElseResult != nil {
		elseType, elseNullable, _ := r.resolveExpressionType(expression.ElseResult, scope, dataModifying)
		resultType = r.commonSupertype(resultType, elseType)
		if elseNullable {
			nullable = true
		}
	}

	return resultType, nullable, nil
}

// resolveWindowFunctionExpression resolves a window
// function expression by delegating to the underlying
// function call resolution.
//
// Takes expression
// (*querier_dto.WindowFunctionExpression) which
// specifies the window function to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving function arguments.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the function modifies data.
//
// Returns querier_dto.SQLType which holds the
// function's return type.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates Q030 for a nil
// expression or function resolution failure.
func (r *typeResolver) resolveWindowFunctionExpression(
	expression *querier_dto.WindowFunctionExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression == nil || expression.Function == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true,
			errors.New(querier_dto.CodeInternalNilGuard + ": nil window function expression during type resolution")
	}
	return r.resolveFunctionCallExpression(expression.Function, scope, dataModifying)
}

// resolveCustomType resolves an unknown SQL type
// against the catalogue's enums and composite types.
// For array types, recursively resolves the element
// type.
//
// Takes sqlType (querier_dto.SQLType) which specifies
// the type to resolve.
//
// Returns querier_dto.SQLType which holds the resolved
// type with category and metadata populated.
func (r *typeResolver) resolveCustomType(sqlType querier_dto.SQLType) querier_dto.SQLType {
	if sqlType.Category == querier_dto.TypeCategoryArray && sqlType.ElementType != nil {
		resolved := r.resolveCustomType(*sqlType.ElementType)
		if resolved.Category != sqlType.ElementType.Category {
			sqlType.ElementType = &resolved
		}
		return sqlType
	}
	if sqlType.Category != querier_dto.TypeCategoryUnknown || sqlType.EngineName == "" {
		return sqlType
	}
	for _, schema := range r.catalogue.Schemas {
		if enum, exists := schema.Enums[sqlType.EngineName]; exists {
			sqlType.Category = querier_dto.TypeCategoryEnum
			sqlType.EnumValues = enum.Values
			sqlType.Schema = enum.Schema
			return sqlType
		}
		if _, exists := schema.CompositeTypes[sqlType.EngineName]; exists {
			sqlType.Category = querier_dto.TypeCategoryComposite
			sqlType.Schema = schema.Name
			return sqlType
		}
	}
	return sqlType
}

// resolveScalarSubqueryExpression resolves a scalar
// subquery expression by creating an inner scope and
// resolving the first output column of the inner query.
// The result is always nullable since the subquery may
// return zero rows.
//
// Takes expression
// (*querier_dto.ScalarSubqueryExpression) which
// specifies the subquery to resolve.
// Takes outerScope (*scopeChain) which specifies the
// parent scope for correlated references.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the subquery modifies data.
//
// Returns querier_dto.SQLType which holds the type of
// the first output column.
// Returns bool which indicates whether the result is
// nullable (always true for subqueries).
// Returns error which indicates a resolution failure.
func (r *typeResolver) resolveScalarSubqueryExpression(
	expression *querier_dto.ScalarSubqueryExpression,
	outerScope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression.InnerQuery == nil || len(expression.InnerQuery.OutputColumns) == 0 {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, nil
	}

	innerScope := newScopeChain(querier_dto.ScopeKindQuery, outerScope)
	r.addRawTablesToScope(expression.InnerQuery, innerScope)

	firstColumn := expression.InnerQuery.OutputColumns[0]
	if firstColumn.Expression == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, nil
	}

	resolvedType, _, err := r.resolveExpressionType(firstColumn.Expression, innerScope, dataModifying)
	if err != nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, nil
	}
	return resolvedType, true, nil
}

// resolveArraySubscriptExpression resolves an array
// subscript expression by extracting the element type
// from the array type. The result is always nullable
// since the subscript index may be out of bounds.
//
// Takes expression
// (*querier_dto.ArraySubscriptExpression) which
// specifies the subscript to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving the array expression.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the array expression modifies data.
//
// Returns querier_dto.SQLType which holds the array
// element type.
// Returns bool which indicates whether the result is
// nullable (always true).
// Returns error which indicates a resolution failure
// for the array expression.
func (r *typeResolver) resolveArraySubscriptExpression(
	expression *querier_dto.ArraySubscriptExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	arrayType, _, err := r.resolveExpressionType(expression.Array, scope, dataModifying)
	if err != nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, err
	}
	if arrayType.ElementType != nil {
		return *arrayType.ElementType, true, nil
	}
	return arrayType, true, nil
}

// resolveLambdaExpression resolves a lambda expression by resolving its body expression.
//
// Takes expression (*querier_dto.LambdaExpression) which specifies the lambda to resolve.
// Takes scope (*scopeChain) which specifies the scope chain for resolving the body.
// Takes dataModifying (*bool) which specifies a flag set to true if the body modifies data.
//
// Returns querier_dto.SQLType which holds the type of the lambda body.
// Returns bool which indicates whether the result is nullable.
// Returns error which indicates a resolution failure for the body expression.
func (r *typeResolver) resolveLambdaExpression(
	expression *querier_dto.LambdaExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	if expression.Body == nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, nil
	}
	return r.resolveExpressionType(expression.Body, scope, dataModifying)
}

// resolveStructFieldAccessExpression resolves a struct
// field access expression by resolving the struct
// expression and then looking up the named field within
// its struct fields.
//
// Takes expression
// (*querier_dto.StructFieldAccessExpression) which
// specifies the field access to resolve.
// Takes scope (*scopeChain) which specifies the scope
// chain for resolving the struct expression.
// Takes dataModifying (*bool) which specifies a flag
// set to true if the struct expression modifies data.
//
// Returns querier_dto.SQLType which holds the type of
// the accessed field.
// Returns bool which indicates whether the result is
// nullable.
// Returns error which indicates a resolution failure
// for the struct expression.
func (r *typeResolver) resolveStructFieldAccessExpression(
	expression *querier_dto.StructFieldAccessExpression,
	scope *scopeChain,
	dataModifying *bool,
) (querier_dto.SQLType, bool, error) {
	structType, nullable, err := r.resolveExpressionType(expression.Struct, scope, dataModifying)
	if err != nil {
		return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, err
	}

	for i := range structType.StructFields {
		if structType.StructFields[i].Name == expression.FieldName {
			return structType.StructFields[i].SQLType, nullable, nil
		}
	}

	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}, true, nil
}

// addRawTablesToScope registers tables from a raw query
// analysis into the given scope by looking them up in
// the catalogue. Both FROM tables and JOIN clause
// tables are registered.
//
// Takes raw (*querier_dto.RawQueryAnalysis) which
// specifies the raw query with table references.
// Takes scope (*scopeChain) which specifies the scope
// to populate with resolved tables.
func (r *typeResolver) addRawTablesToScope(raw *querier_dto.RawQueryAnalysis, scope *scopeChain) {
	for _, tableReference := range raw.FromTables {
		schemaName := tableReference.Schema
		if schemaName == "" {
			schemaName = r.catalogue.DefaultSchema
		}
		schema, exists := r.catalogue.Schemas[schemaName]
		if !exists {
			continue
		}
		if table, tableExists := schema.Tables[tableReference.Name]; tableExists {
			_ = scope.AddTable(tableReference, querier_dto.JoinInner, table)
		}
	}
	for _, joinClause := range raw.JoinClauses {
		schemaName := joinClause.Table.Schema
		if schemaName == "" {
			schemaName = r.catalogue.DefaultSchema
		}
		schema, exists := r.catalogue.Schemas[schemaName]
		if !exists {
			continue
		}
		if table, tableExists := schema.Tables[joinClause.Table.Name]; tableExists {
			_ = scope.AddTable(joinClause.Table, joinClause.Kind, table)
		}
	}
}

// commonSupertype returns the common supertype of two
// SQL types following the engine's promotion and
// implicit casting rules. Used for CASE/WHEN branches,
// UNION columns, and COALESCE arguments.
//
// Takes left (querier_dto.SQLType) which specifies the
// first type to merge.
// Takes right (querier_dto.SQLType) which specifies the
// second type to merge.
//
// Returns querier_dto.SQLType which holds the common
// supertype, or unknown if no promotion path exists.
func (r *typeResolver) commonSupertype(left querier_dto.SQLType, right querier_dto.SQLType) querier_dto.SQLType {
	if left.Category == querier_dto.TypeCategoryUnknown {
		return right
	}
	if right.Category == querier_dto.TypeCategoryUnknown {
		return left
	}

	if left.Category == right.Category && strings.EqualFold(left.EngineName, right.EngineName) {
		return left
	}

	if left.Category == right.Category {
		return r.engine.PromoteType(left, right)
	}

	if r.engine.CanImplicitCast(left.Category, right.Category) {
		return right
	}
	if r.engine.CanImplicitCast(right.Category, left.Category) {
		return left
	}

	return querier_dto.SQLType{Category: querier_dto.TypeCategoryUnknown}
}

// isBooleanExpression returns true for expression types that always resolve to
// a non-nullable boolean result (comparisons, IS NULL, IN, BETWEEN, logical
// operators, EXISTS).
//
// Takes expression (querier_dto.Expression) which specifies the expression to classify.
//
// Returns bool which indicates true if the expression always produces a boolean result.
func isBooleanExpression(expression querier_dto.Expression) bool {
	switch expression.(type) {
	case *querier_dto.ComparisonExpression,
		*querier_dto.IsNullExpression,
		*querier_dto.InListExpression,
		*querier_dto.BetweenExpression,
		*querier_dto.LogicalOpExpression,
		*querier_dto.ExistsExpression:
		return true
	default:
		return false
	}
}

// expressionFeature maps a concrete expression type to
// its corresponding SQLExpressionFeature flag. Returns
// 0 for expression kinds that do not need feature
// validation (column refs, function calls, literals,
// casts, coalesce).
//
// Takes expression (querier_dto.Expression) which
// specifies the expression to map.
//
// Returns querier_dto.SQLExpressionFeature which holds
// the feature flag, or 0 if no validation is needed.
func expressionFeature(expression querier_dto.Expression) querier_dto.SQLExpressionFeature {
	switch expr := expression.(type) {
	case *querier_dto.BinaryOpExpression:
		return binaryOpFeature(expr.Operator)
	case *querier_dto.ComparisonExpression:
		return comparisonFeature(expr.Operator)
	case *querier_dto.IsNullExpression:
		return querier_dto.SQLFeatureIsNull
	case *querier_dto.InListExpression:
		return querier_dto.SQLFeatureInList
	case *querier_dto.BetweenExpression:
		return querier_dto.SQLFeatureBetween
	case *querier_dto.LogicalOpExpression:
		return querier_dto.SQLFeatureLogicalOp
	case *querier_dto.UnaryOpExpression:
		return querier_dto.SQLFeatureUnaryOp
	case *querier_dto.CaseWhenExpression:
		return querier_dto.SQLFeatureCaseWhen
	case *querier_dto.ExistsExpression:
		return querier_dto.SQLFeatureExists
	case *querier_dto.WindowFunctionExpression:
		return querier_dto.SQLFeatureWindowFunction
	case *querier_dto.ScalarSubqueryExpression:
		return querier_dto.SQLFeatureScalarSubquery
	case *querier_dto.ArraySubscriptExpression:
		return querier_dto.SQLFeatureArraySubscript
	case *querier_dto.LambdaExpression:
		return querier_dto.SQLFeatureLambda
	case *querier_dto.StructFieldAccessExpression:
		return querier_dto.SQLFeatureStructFieldAccess
	default:
		return 0
	}
}

// binaryOpFeature maps a binary operator string to its SQLExpressionFeature flag.
//
// Takes operator (string) which specifies the operator symbol (e.g. "||", "->", "&").
//
// Returns querier_dto.SQLExpressionFeature which holds the corresponding feature flag.
func binaryOpFeature(operator string) querier_dto.SQLExpressionFeature {
	switch operator {
	case "||":
		return querier_dto.SQLFeatureStringConcat
	case "->", "->>":
		return querier_dto.SQLFeatureJSONOp
	case "&", "|", "<<", ">>":
		return querier_dto.SQLFeatureBitwiseOp
	default:
		return querier_dto.SQLFeatureBinaryArithmetic
	}
}

// comparisonFeature maps a comparison operator string
// to its SQLExpressionFeature flag.
//
// Takes operator (string) which specifies the
// comparison operator (e.g. "LIKE", "GLOB", "=").
//
// Returns querier_dto.SQLExpressionFeature which holds
// the corresponding feature flag.
func comparisonFeature(operator string) querier_dto.SQLExpressionFeature {
	switch operator {
	case "LIKE", "GLOB", "REGEXP", "MATCH":
		return querier_dto.SQLFeaturePatternMatch
	default:
		return querier_dto.SQLFeatureBinaryComparison
	}
}

// extractErrorCode extracts a Q-code prefix from an
// error message. Error messages follow the format
// "Q0NN: description".
//
// Takes err (error) which specifies the error whose
// message is parsed for a Q-code prefix.
//
// Returns string which holds the extracted Q-code
// (e.g. "Q001"), defaulting to "Q001" if no code is
// found.
func extractErrorCode(err error) string {
	message := err.Error()
	if len(message) >= errorCodeMinLength && message[0] == 'Q' && message[errorCodePrefixLength] == ':' {
		return message[:errorCodePrefixLength]
	}
	return querier_dto.CodeUnknownColumn
}
