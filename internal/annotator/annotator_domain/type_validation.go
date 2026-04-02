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

package annotator_domain

// Pure validation functions for type checking in expressions. These functions
// are stateless and return validation results that can be unit tested without
// requiring full analysis context or TypeResolver infrastructure.

import (
	"fmt"

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
)

// ValidationResult holds the outcome of a type validation check.
type ValidationResult struct {
	// Message contains the error or warning message when Valid is false.
	Message string

	// Valid indicates whether the validation passed.
	Valid bool
}

// ValidateOrderingComparison checks if types support ordering operators
// (<, >, <=, >=).
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the left operand type.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the right operand type.
// Takes operator (ast_domain.BinaryOp) which is the comparison operator.
//
// Returns ValidationResult which indicates whether the comparison is valid.
func ValidateOrderingComparison(left, right *ast_domain.ResolvedTypeInfo, operator ast_domain.BinaryOp) ValidationResult {
	if areComparableForOrdering(left, right) {
		return ValidationResult{Valid: true}
	}
	leftString := goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	rightString := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Invalid operation: cannot compare type '%s' to '%s' with operator '%s'", leftString, rightString, operator),
	}
}

// ValidateEqualityComparison checks if types support equality operators (==,
// !=, ===, !==).
//
// Takes operator (ast_domain.BinaryOp) which is the equality operator.
// Takes left (*ast_domain.ResolvedTypeInfo) which is the type of the left
// operand.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the type of the right
// operand.
// Takes strict (bool) which indicates whether strict equality semantics apply.
//
// Returns ValidationResult which indicates whether the comparison is valid.
func ValidateEqualityComparison(operator ast_domain.BinaryOp, left, right *ast_domain.ResolvedTypeInfo, strict bool) ValidationResult {
	if areComparableForEquality(operator, left, right) {
		return ValidationResult{Valid: true}
	}
	leftString := goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	rightString := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	verb := "compare"
	if strict {
		verb = "strictly compare"
	}
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Invalid operation: cannot %s type '%s' to '%s'", verb, leftString, rightString),
	}
}

// ValidateLogicalAndLeftOperand checks if the left operand of a logical AND is
// boolean.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the type of the left
// operand.
//
// Returns ValidationResult which indicates whether the left operand is valid.
func ValidateLogicalAndLeftOperand(left *ast_domain.ResolvedTypeInfo) ValidationResult {
	if isBoolLike(left) {
		return ValidationResult{Valid: true}
	}
	leftString := goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Left operand of '&&' must be boolean, but got type '%s'; use the ~ operator for truthiness evaluation (e.g., ~value)", leftString),
	}
}

// ValidateLogicalAndRightOperand checks if the right operand of a logical AND
// is boolean.
//
// Takes right (*ast_domain.ResolvedTypeInfo) which is the type of the right
// operand.
//
// Returns ValidationResult which indicates whether the right operand is valid.
func ValidateLogicalAndRightOperand(right *ast_domain.ResolvedTypeInfo) ValidationResult {
	if isBoolLike(right) {
		return ValidationResult{Valid: true}
	}
	rightString := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Right operand of '&&' must be boolean, but got type '%s'; use the ~ operator for truthiness evaluation (e.g., ~value)", rightString),
	}
}

// ValidateOrOperator checks if OR operands are type-compatible.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the left operand type.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the right operand type.
//
// Returns ValidationResult which indicates whether the OR operation is valid.
func ValidateOrOperator(left, right *ast_domain.ResolvedTypeInfo) ValidationResult {
	if isAssignable(right, left) || isNilType(left) {
		return ValidationResult{Valid: true}
	}
	leftString := goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	rightString := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	suggestion := suggestCoercionFunction(leftString, rightString)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Logical OR operator '||' used with incompatible types '%s' and '%s'%s", leftString, rightString, suggestion),
	}
}

// ValidateCoalesceOperator checks if null coalescing operands are compatible.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the left operand type.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the right operand type.
//
// Returns ValidationResult which indicates whether the coalesce is valid.
func ValidateCoalesceOperator(left, right *ast_domain.ResolvedTypeInfo) ValidationResult {
	if isAssignable(right, left) || isNilType(left) {
		return ValidationResult{Valid: true}
	}
	leftString := goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	rightString := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Null coalescing operator '??' used with incompatible types '%s' and '%s'", leftString, rightString),
	}
}

// ValidateBooleanCondition checks if an expression has a boolean type.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which is the type to check.
//
// Returns ValidationResult which indicates whether the type is boolean.
func ValidateBooleanCondition(typeInfo *ast_domain.ResolvedTypeInfo) ValidationResult {
	if typeInfo == nil || isBoolLike(typeInfo) {
		return ValidationResult{Valid: true}
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Condition must be a boolean expression, but got type '%s'; use the ~ operator for truthiness evaluation (e.g., ~value)", typeString),
	}
}

// ValidateUnaryNot checks if the operand of a logical NOT is boolean.
//
// Takes operand (*ast_domain.ResolvedTypeInfo) which is the type of the
// operand.
//
// Returns ValidationResult which indicates whether the NOT operation is valid.
func ValidateUnaryNot(operand *ast_domain.ResolvedTypeInfo) ValidationResult {
	if isBoolLike(operand) {
		return ValidationResult{Valid: true}
	}
	operandString := goastutil.ASTToTypeString(operand.TypeExpression, operand.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Unary operator '!' is not defined for non-boolean type '%s'", operandString),
	}
}

// ValidateUnaryNeg checks if the operand of a numeric negation is arithmetic.
//
// Takes operand (*ast_domain.ResolvedTypeInfo) which is the type of the
// operand.
//
// Returns ValidationResult which indicates whether the negation operation is
// valid.
func ValidateUnaryNeg(operand *ast_domain.ResolvedTypeInfo) ValidationResult {
	if getNumericFamily(operand) != familyNone {
		return ValidationResult{Valid: true}
	}
	operandString := goastutil.ASTToTypeString(operand.TypeExpression, operand.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Unary operator '-' is not defined for non-arithmetic type '%s'", operandString),
	}
}

// ValidateArithmeticOperator checks if types support arithmetic operators (+,
// -, *, /, %).
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the type of the left
// operand.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the type of the right
// operand.
// Takes operator (ast_domain.BinaryOp) which is the arithmetic operator.
//
// Returns ValidationResult which indicates whether the operation is valid.
func ValidateArithmeticOperator(left, right *ast_domain.ResolvedTypeInfo, operator ast_domain.BinaryOp) ValidationResult {
	if isMoneyType(left) || isMoneyType(right) {
		return ValidationResult{Valid: true}
	}
	if operator == ast_domain.OpPlus && isStringType(left) && isStringType(right) {
		return ValidationResult{Valid: true}
	}
	if isArithmeticType(left, right) {
		return ValidationResult{Valid: true}
	}
	leftString := goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	rightString := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Operator '%s' not defined for operand types '%s' and '%s'", operator, leftString, rightString),
	}
}

// ValidateTernaryBranches checks if ternary branches have compatible types.
//
// Takes consequent (*ast_domain.ResolvedTypeInfo) which is the true
// branch type.
// Takes alternate (*ast_domain.ResolvedTypeInfo) which is the false
// branch type.
//
// Returns ValidationResult which indicates whether the branches are compatible.
func ValidateTernaryBranches(consequent, alternate *ast_domain.ResolvedTypeInfo) ValidationResult {
	if isAssignable(alternate, consequent) {
		return ValidationResult{Valid: true}
	}
	consequentString := goastutil.ASTToTypeString(consequent.TypeExpression, consequent.PackageAlias)
	alternateString := goastutil.ASTToTypeString(alternate.TypeExpression, alternate.PackageAlias)
	suggestion := suggestCoercionFunction(consequentString, alternateString)
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Ternary expression has mismatched types: true branch is '%s', false branch is '%s'%s", consequentString, alternateString, suggestion),
	}
}

// ValidateMoneyAddSub checks if Money addition or subtraction operands are
// valid.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the type of the left
// operand.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the type of the right
// operand.
// Takes isLeftMoney (bool) which indicates if the left operand is Money.
// Takes isRightMoney (bool) which indicates if the right operand is Money.
//
// Returns ValidationResult which indicates whether the operation is valid.
func ValidateMoneyAddSub(left, right *ast_domain.ResolvedTypeInfo, isLeftMoney, isRightMoney bool) ValidationResult {
	isLeftDecimal := getNumericFamily(left) == familyDecimal
	isRightDecimal := getNumericFamily(right) == familyDecimal

	if (isLeftMoney && isRightMoney) || (isLeftMoney && isRightDecimal) || (isLeftDecimal && isRightMoney) {
		return ValidationResult{Valid: true}
	}

	invalidType := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	if isRightMoney {
		invalidType = goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	}
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Invalid operation: cannot add or subtract Money with type '%s'", invalidType),
	}
}

// ValidateMoneyMulDiv checks operands for Money multiplication or division.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the left operand type.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the right operand type.
// Takes isLeftMoney (bool) which indicates if the left operand is Money.
// Takes isRightMoney (bool) which indicates if the right operand is Money.
// Takes expressionString (string) which is the expression string for
// error messages.
//
// Returns ValidationResult which indicates whether the operation is valid.
func ValidateMoneyMulDiv(left, right *ast_domain.ResolvedTypeInfo, isLeftMoney, isRightMoney bool, expressionString string) ValidationResult {
	if isLeftMoney && isRightMoney {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Invalid operation: Cannot multiply or divide Money by Money ('%s')", expressionString),
		}
	}

	if (isLeftMoney && isNumericType(right)) || (isNumericType(left) && isRightMoney) {
		return ValidationResult{Valid: true}
	}

	invalidType := goastutil.ASTToTypeString(right.TypeExpression, right.PackageAlias)
	if isRightMoney {
		invalidType = goastutil.ASTToTypeString(left.TypeExpression, left.PackageAlias)
	}
	return ValidationResult{
		Valid:   false,
		Message: fmt.Sprintf("Invalid operation: can only multiply or divide Money by a standard number, not type '%s'", invalidType),
	}
}
