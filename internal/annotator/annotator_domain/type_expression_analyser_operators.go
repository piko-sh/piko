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

// Analyses operator expressions in templates by validating operand types and determining result types for binary and unary operations.
// Handles arithmetic, comparison, logical, and string concatenation operators whilst ensuring type compatibility and correctness.

import (
	"context"
	"fmt"
	goast "go/ast"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/logger/logger_domain"
)

// resolveBinaryExpression finds the result type of a binary expression.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to resolve.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the result type, or a
// fallback value if either operand cannot be resolved.
func (a *typeExpressionAnalyser) resolveBinaryExpression(ctx context.Context, n *ast_domain.BinaryExpression) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveBinaryExpression", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))

	leftAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Left, a.location, a.depth+1)
	rightAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Right, a.location, a.depth+1)

	if leftAnn.ResolvedType == nil || rightAnn.ResolvedType == nil {
		return a.createBinaryFallbackAnnotation()
	}

	a.ctx.Logger.Trace("Binary operands resolved",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String("leftType", a.typeResolver.logAnn(leftAnn)),
		logger_domain.String("rightType", a.typeResolver.logAnn(rightAnn)))

	resultTypeExpr, resultPackageAlias, earlyReturn := a.resolveBinaryOperator(n, leftAnn, rightAnn)
	if earlyReturn != nil {
		return earlyReturn
	}

	ann := a.createBinaryResultAnnotation(resultTypeExpr, resultPackageAlias)
	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveBinaryExpression",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(ann)))
	return ann
}

// createBinaryFallbackAnnotation returns a fallback annotation for binary
// expressions when type resolution fails.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains an "any" type with
// no stringability.
func (a *typeExpressionAnalyser) createBinaryFallbackAnnotation() *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            newSimpleTypeInfo(goast.NewIdent(typeAny)),
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      &a.ctx.SFCSourcePath,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           int(inspector_dto.StringableNone),
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// resolveBinaryOperator resolves the result type of a binary expression.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to analyse.
// Takes leftAnn (*ast_domain.GoGeneratorAnnotation) which is the
// annotation for
// the left operand.
// Takes rightAnn (*ast_domain.GoGeneratorAnnotation) which is the annotation
// for the right operand.
//
// Returns goast.Expr which is the Go type expression for the result.
// Returns string which is an optional type name hint.
// Returns *ast_domain.GoGeneratorAnnotation which is the propagated annotation
// for coalesce operators.
func (a *typeExpressionAnalyser) resolveBinaryOperator(n *ast_domain.BinaryExpression, leftAnn, rightAnn *ast_domain.GoGeneratorAnnotation) (goast.Expr, string, *ast_domain.GoGeneratorAnnotation) {
	leftTypeInfo, rightTypeInfo := leftAnn.ResolvedType, rightAnn.ResolvedType

	switch n.Operator {
	case ast_domain.OpAnd:
		a.validateLogicalAndOperands(n, leftTypeInfo, rightTypeInfo)
		return goast.NewIdent(typeBool), "", nil

	case ast_domain.OpOr:
		a.validateOrOperator(n, leftTypeInfo, rightTypeInfo)
		return nil, "", leftAnn

	case ast_domain.OpGt, ast_domain.OpLt, ast_domain.OpGe, ast_domain.OpLe:
		a.validateOrderingComparison(n, leftTypeInfo, rightTypeInfo)
		return goast.NewIdent(typeBool), "", nil

	case ast_domain.OpEq, ast_domain.OpNe:
		a.validateEqualityComparison(n, leftTypeInfo, rightTypeInfo, true)
		return goast.NewIdent(typeBool), "", nil

	case ast_domain.OpLooseEq, ast_domain.OpLooseNe:
		a.validateEqualityComparison(n, leftTypeInfo, rightTypeInfo, false)
		return goast.NewIdent(typeBool), "", nil

	case ast_domain.OpPlus, ast_domain.OpMinus, ast_domain.OpMul, ast_domain.OpDiv, ast_domain.OpMod:
		return a.resolveArithmeticOperator(n, leftTypeInfo, rightTypeInfo)

	case ast_domain.OpCoalesce:
		a.validateCoalesceOperator(n, leftTypeInfo, rightTypeInfo)
		return nil, "", leftAnn
	}

	return goast.NewIdent(typeAny), "", nil
}

// validateOrderingComparison checks that an ordering comparison uses types
// that support ordering operators such as <, >, <=, and >=.
//
// Takes n (*ast_domain.BinaryExpression) which is the comparison
// expression to check.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which holds the type of the
// left operand.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which holds the type of
// the right operand.
func (a *typeExpressionAnalyser) validateOrderingComparison(n *ast_domain.BinaryExpression, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) {
	result := ValidateOrderingComparison(leftTypeInfo, rightTypeInfo, n.Operator)
	if !result.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Error, result.Message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeComparisonError)
	}
}

// validateEqualityComparison checks that two types can be compared for
// equality and reports an error if they cannot.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to check.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type of the
// left side.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type of the
// right side.
// Takes strict (bool) which sets whether to use strict comparison rules.
func (a *typeExpressionAnalyser) validateEqualityComparison(n *ast_domain.BinaryExpression, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo, strict bool) {
	result := ValidateEqualityComparison(n.Operator, leftTypeInfo, rightTypeInfo, strict)
	if !result.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Error, result.Message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeComparisonError)
	}
}

// resolveArithmeticOperator determines the result type for an arithmetic
// binary expression.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to analyse.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the left
// operand type.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the right
// operand type.
//
// Returns goast.Expr which is the result type expression.
// Returns string which is the package alias for the result type.
// Returns *ast_domain.GoGeneratorAnnotation which is always nil.
func (a *typeExpressionAnalyser) resolveArithmeticOperator(
	n *ast_domain.BinaryExpression,
	leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo,
) (goast.Expr, string, *ast_domain.GoGeneratorAnnotation) {
	if isMoneyType(leftTypeInfo) || isMoneyType(rightTypeInfo) {
		typeExpr, packageAlias := a.resolveMoneyArithmetic(n, leftTypeInfo, rightTypeInfo)
		return typeExpr, packageAlias, nil
	}

	if n.Operator == ast_domain.OpPlus && isStringType(leftTypeInfo) && isStringType(rightTypeInfo) {
		return goast.NewIdent(typeString), "", nil
	}

	if isArithmeticType(leftTypeInfo, rightTypeInfo) {
		promoted := promoteNumericTypes(leftTypeInfo, rightTypeInfo)
		return promoted.TypeExpression, promoted.PackageAlias, nil
	}

	leftType := goastutil.ASTToTypeString(leftTypeInfo.TypeExpression, leftTypeInfo.PackageAlias)
	rightType := goastutil.ASTToTypeString(rightTypeInfo.TypeExpression, rightTypeInfo.PackageAlias)
	message := fmt.Sprintf("Operator '%s' not defined for operand types '%s' and '%s'", n.Operator, leftType, rightType)
	a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeArithmeticError)
	return goast.NewIdent(typeAny), "", nil
}

// validateCoalesceOperator checks that the null coalescing operator has
// operand types that work together.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to check.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the left
// operand type.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the right
// operand type.
func (a *typeExpressionAnalyser) validateCoalesceOperator(n *ast_domain.BinaryExpression, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) {
	result := ValidateCoalesceOperator(leftTypeInfo, rightTypeInfo)
	if !result.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Warning, result.Message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeCoalesceError)
	}
}

// validateLogicalAndOperands checks that both sides of a logical AND
// expression have boolean types.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to check.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which holds the type of
// the left side.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which holds the type of
// the right side.
func (a *typeExpressionAnalyser) validateLogicalAndOperands(n *ast_domain.BinaryExpression, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) {
	leftResult := ValidateLogicalAndLeftOperand(leftTypeInfo)
	if !leftResult.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Error, leftResult.Message, n, a.location.Add(n.Left.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeLogicalOperatorError)
	}
	rightResult := ValidateLogicalAndRightOperand(rightTypeInfo)
	if !rightResult.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Error, rightResult.Message, n, a.location.Add(n.Right.GetRelativeLocation()), n.GoAnnotations, annotator_dto.CodeLogicalOperatorError)
	}
}

// validateOrOperator checks that both sides of a logical OR expression have
// types that can work together and reports an error if they do not.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to check.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which holds the type of
// the left side.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which holds the type of
// the right side.
func (a *typeExpressionAnalyser) validateOrOperator(n *ast_domain.BinaryExpression, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) {
	result := ValidateOrOperator(leftTypeInfo, rightTypeInfo)
	if !result.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Error, result.Message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeLogicalOperatorError)
	}
}

// createBinaryResultAnnotation builds an annotation for a binary expression
// result type.
//
// Takes resultTypeExpr (goast.Expr) which is the AST expression for the result
// type.
// Takes resultPackageAlias (string) which is the package alias for the
// result type.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the annotation with
// stringability details worked out from the result type.
func (a *typeExpressionAnalyser) createBinaryResultAnnotation(resultTypeExpr goast.Expr, resultPackageAlias string) *ast_domain.GoGeneratorAnnotation {
	resultTypeInfo := newSimpleTypeInfoWithAlias(resultTypeExpr, resultPackageAlias)
	stringability, isPointer := a.typeResolver.determineStringability(a.ctx, resultTypeInfo)
	return &ast_domain.GoGeneratorAnnotation{
		EffectiveKeyExpression:  nil,
		DynamicCollectionInfo:   nil,
		StaticCollectionLiteral: nil,
		ParentTypeName:          nil,
		BaseCodeGenVarName:      nil,
		GeneratedSourcePath:     nil,
		DynamicAttributeOrigins: nil,
		ResolvedType:            resultTypeInfo,
		Symbol:                  nil,
		PartialInfo:             nil,
		PropDataSource:          nil,
		OriginalSourcePath:      &a.ctx.SFCSourcePath,
		OriginalPackageAlias:    nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		StaticCollectionData:    nil,
		Srcset:                  nil,
		Stringability:           stringability,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   isPointer,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// resolveMoneyArithmetic handles arithmetic operations on Money types.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to resolve.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type of the
// left operand.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type of the
// right operand.
//
// Returns goast.Expr which is the resolved Go AST expression.
// Returns string which is the result type name, or empty when there is an
// error.
func (a *typeExpressionAnalyser) resolveMoneyArithmetic(n *ast_domain.BinaryExpression, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) (goast.Expr, string) {
	isLeftMoney, isRightMoney := isMoneyType(leftTypeInfo), isMoneyType(rightTypeInfo)
	isLeftDecimal := getNumericFamily(leftTypeInfo) == familyDecimal
	isRightDecimal := getNumericFamily(rightTypeInfo) == familyDecimal

	switch n.Operator {
	case ast_domain.OpPlus, ast_domain.OpMinus:
		return a.resolveMoneyAddSub(n, leftTypeInfo, rightTypeInfo, isLeftMoney, isRightMoney, isLeftDecimal, isRightDecimal)
	case ast_domain.OpMul, ast_domain.OpDiv:
		return a.resolveMoneyMulDiv(n, leftTypeInfo, rightTypeInfo, isLeftMoney, isRightMoney)
	default:
		a.ctx.addDiagnosticForExpression(ast_domain.Error, "Operator '%' not defined for Money type", n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeArithmeticError)
	}
	return goast.NewIdent(typeAny), ""
}

// resolveMoneyAddSub resolves the result type for addition or subtraction
// involving Money types.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to analyse.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the left
// operand type.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the right
// operand type.
// Takes isLeftMoney (bool) which indicates if the left operand is Money.
// Takes isRightMoney (bool) which indicates if the right operand is Money.
// Takes isLeftDecimal (bool) which indicates if the left operand is Decimal.
// Takes isRightDecimal (bool) which indicates if the right operand is Decimal.
//
// Returns goast.Expr which is the resolved type expression.
// Returns string which is the package alias, or empty if the operation is
// invalid.
func (a *typeExpressionAnalyser) resolveMoneyAddSub(
	n *ast_domain.BinaryExpression,
	leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo,
	isLeftMoney, isRightMoney, isLeftDecimal, isRightDecimal bool,
) (goast.Expr, string) {
	if (isLeftMoney && isRightMoney) || (isLeftMoney && isRightDecimal) || (isLeftDecimal && isRightMoney) {
		return goastutil.TypeStringToAST("maths.Money"), "maths"
	}

	invalidType := goastutil.ASTToTypeString(rightTypeInfo.TypeExpression, rightTypeInfo.PackageAlias)
	if isRightMoney {
		invalidType = goastutil.ASTToTypeString(leftTypeInfo.TypeExpression, leftTypeInfo.PackageAlias)
	}
	message := fmt.Sprintf("Invalid operation: cannot add or subtract Money with type '%s'", invalidType)
	a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeArithmeticError)
	return goast.NewIdent(typeAny), ""
}

// resolveMoneyMulDiv finds the result type for Money multiply or divide
// operations.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to check.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type of the
// left operand.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type of the
// right operand.
// Takes isLeftMoney (bool) which shows if the left operand is Money.
// Takes isRightMoney (bool) which shows if the right operand is Money.
//
// Returns goast.Expr which is the result type expression.
// Returns string which is the package alias needed for the result type.
func (a *typeExpressionAnalyser) resolveMoneyMulDiv(n *ast_domain.BinaryExpression, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo, isLeftMoney, isRightMoney bool) (goast.Expr, string) {
	if isLeftMoney && isRightMoney {
		message := fmt.Sprintf("Invalid operation: Cannot multiply or divide Money by Money ('%s')", n.String())
		a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeArithmeticError)
		return goast.NewIdent(typeAny), ""
	}

	if (isLeftMoney && isNumericType(rightTypeInfo)) || (isNumericType(leftTypeInfo) && isRightMoney) {
		return goastutil.TypeStringToAST("maths.Money"), "maths"
	}

	invalidType := goastutil.ASTToTypeString(rightTypeInfo.TypeExpression, rightTypeInfo.PackageAlias)
	if isRightMoney {
		invalidType = goastutil.ASTToTypeString(leftTypeInfo.TypeExpression, leftTypeInfo.PackageAlias)
	}
	message := fmt.Sprintf("Invalid operation: can only multiply or divide Money by a standard number, not type '%s'", invalidType)
	a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeArithmeticError)
	return goast.NewIdent(typeAny), ""
}

// resolveUnaryExpression resolves the type of a unary expression.
//
// Takes n (*ast_domain.UnaryExpression) which is the unary expression to analyse.
//
// Returns *ast_domain.GoGeneratorAnnotation which contains the resolved type
// information, or a fallback annotation when the operand cannot be resolved.
func (a *typeExpressionAnalyser) resolveUnaryExpression(ctx context.Context, n *ast_domain.UnaryExpression) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveUnaryExpression", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))

	rightAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Right, a.location, a.depth+1)

	if rightAnn == nil || rightAnn.ResolvedType == nil {
		fallback := newFallbackAnnotation()
		fallback.OriginalSourcePath = &a.ctx.SFCSourcePath
		return fallback
	}

	var resultTypeExpr goast.Expr
	var resultPackageAlias string

	switch n.Operator {
	case ast_domain.OpNot:
		if !isBoolLike(rightAnn.ResolvedType) {
			rightType := goastutil.ASTToTypeString(rightAnn.ResolvedType.TypeExpression, rightAnn.ResolvedType.PackageAlias)
			message := fmt.Sprintf("Unary operator '!' is not defined for non-boolean type '%s'", rightType)
			a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeLogicalOperatorError)
		}
		resultTypeExpr = goast.NewIdent(typeBool)

	case ast_domain.OpNeg:
		if getNumericFamily(rightAnn.ResolvedType) == familyNone {
			rightType := goastutil.ASTToTypeString(rightAnn.ResolvedType.TypeExpression, rightAnn.ResolvedType.PackageAlias)
			message := fmt.Sprintf("Unary operator '-' is not defined for non-arithmetic type '%s'", rightType)
			a.ctx.addDiagnosticForExpression(ast_domain.Error, message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeArithmeticError)
			resultTypeExpr = goast.NewIdent(typeAny)
		} else {
			resultTypeExpr = rightAnn.ResolvedType.TypeExpression
			resultPackageAlias = rightAnn.ResolvedType.PackageAlias
		}

	case ast_domain.OpTruthy:
		resultTypeExpr = goast.NewIdent(typeBool)
	}

	resultTypeInfo := newSimpleTypeInfoWithAlias(resultTypeExpr, resultPackageAlias)
	stringability, isPointer := a.typeResolver.determineStringability(a.ctx, resultTypeInfo)
	ann := newAnnotationFull(resultTypeInfo, &a.ctx.SFCSourcePath, stringability)
	ann.IsPointerToStringable = isPointer
	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveUnaryExpression",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(ann)))
	return ann
}

// resolveTernaryExpression finds the type of a ternary expression.
//
// Takes n (*ast_domain.TernaryExpression) which is the ternary
// expression to check.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds the type from the true
// branch.
func (a *typeExpressionAnalyser) resolveTernaryExpression(ctx context.Context, n *ast_domain.TernaryExpression) *ast_domain.GoGeneratorAnnotation {
	a.ctx.Logger.Trace("[TR-DEBUG] Enter resolveTernaryExpression", logger_domain.Int(logKeyDepth, a.depth), logger_domain.String(logKeyExpr, n.String()))
	conditionAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Condition, a.location, a.depth+1)
	a.validateBooleanCondition(conditionAnn, n.Condition, n, n.GoAnnotations)
	consequentAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Consequent, a.location, a.depth+1)
	alternateAnn := a.typeResolver.resolveRecursive(ctx, a.ctx, n.Alternate, a.location, a.depth+1)

	branchResult := ValidateTernaryBranches(consequentAnn.ResolvedType, alternateAnn.ResolvedType)
	if !branchResult.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Error, branchResult.Message, n, a.location.Add(n.RelativeLocation), n.GoAnnotations, annotator_dto.CodeTypeMismatch)
	}
	a.ctx.Logger.Trace("[TR-DEBUG] Exit resolveTernaryExpression",
		logger_domain.Int(logKeyDepth, a.depth),
		logger_domain.String(logKeyExpr, n.String()),
		logger_domain.String(logKeyResolvedType, a.typeResolver.logAnn(consequentAnn)))
	return consequentAnn
}

// validateBooleanCondition checks that a condition expression has a boolean
// type and reports an error if it does not.
//
// Takes conditionAnn (*ast_domain.GoGeneratorAnnotation) which is the resolved
// annotation for the condition expression.
// Takes conditionExpr (ast_domain.Expression) which is the condition expression
// used to show the error location.
// Takes parentExpr (ast_domain.Expression) which is the parent expression where
// the error is added.
// Takes parentAnn (*ast_domain.GoGeneratorAnnotation) which is the annotation
// to add the error to.
func (a *typeExpressionAnalyser) validateBooleanCondition(
	conditionAnn *ast_domain.GoGeneratorAnnotation,
	conditionExpr, parentExpr ast_domain.Expression,
	parentAnn *ast_domain.GoGeneratorAnnotation,
) {
	if conditionAnn == nil || conditionAnn.ResolvedType == nil {
		return
	}
	result := ValidateBooleanCondition(conditionAnn.ResolvedType)
	if !result.Valid {
		a.ctx.addDiagnosticForExpression(ast_domain.Error, result.Message, parentExpr, a.location.Add(conditionExpr.GetRelativeLocation()), parentAnn, annotator_dto.CodeLogicalOperatorError)
	}
}
