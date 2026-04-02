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

	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/goastutil"
)

// BinaryOpEmitter provides a way to emit binary operation expressions.
// This enables mocking and testing of binary operation logic.
type BinaryOpEmitter interface {
	// emit converts a binary expression to its Go AST representation.
	//
	// Takes n (*ast_domain.BinaryExpression) which is the binary
	// expression to convert.
	//
	// Returns goast.Expr which is the converted Go expression.
	// Returns []goast.Stmt which contains any auxiliary statements needed.
	// Returns []*ast_domain.Diagnostic which reports any conversion issues.
	emit(n *ast_domain.BinaryExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic)
}

// binaryOpEmitter handles code generation for binary expressions.
// It implements BinaryOpEmitter.
type binaryOpEmitter struct {
	// emitter provides access to the main code emitter.
	emitter *emitter

	// expressionEmitter handles emitting the left and right operands
	// of binary operations.
	expressionEmitter ExpressionEmitter
}

var _ BinaryOpEmitter = (*binaryOpEmitter)(nil)

// emit generates a Go binary expression from a domain binary expression.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to convert.
//
// Returns goast.Expr which is the generated Go expression.
// Returns []goast.Stmt which contains any statements needed for the expression.
// Returns []*ast_domain.Diagnostic which contains any diagnostics from emission.
func (be *binaryOpEmitter) emit(n *ast_domain.BinaryExpression) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	leftGoExpr, rightGoExpr, allStmts, allDiags := be.emitOperands(n)

	coerced := be.applyTypeCoercion(n, leftGoExpr, rightGoExpr, allStmts)

	if expression, handled := be.tryEmitLogicalOperator(n, coerced.leftExpr, coerced.rightExpr); handled {
		return expression, coerced.updatedStmts, allDiags
	}

	return be.emitNativeOrHelperOp(n, coerced.leftExpr, coerced.rightExpr, coerced.leftType, coerced.rightType, coerced.updatedStmts, allDiags)
}

// emitOperands emits both operands of a binary expression.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary
// expression to process.
//
// Returns leftExpr (goast.Expr) which is the emitted left operand.
// Returns rightExpr (goast.Expr) which is the emitted right operand.
// Returns statements ([]goast.Stmt) which contains statements from both operands.
// Returns diagnostics ([]*ast_domain.Diagnostic) which contains
// diagnostics from both operands.
func (be *binaryOpEmitter) emitOperands(n *ast_domain.BinaryExpression) (leftExpr, rightExpr goast.Expr, statements []goast.Stmt, diagnostics []*ast_domain.Diagnostic) {
	leftExpr, leftStmts, leftDiags := be.expressionEmitter.emit(n.Left)
	rightExpr, rightStmts, rightDiags := be.expressionEmitter.emit(n.Right)

	statements = leftStmts
	statements = append(statements, rightStmts...)
	diagnostics = leftDiags
	diagnostics = append(diagnostics, rightDiags...)

	return leftExpr, rightExpr, statements, diagnostics
}

// coercionResult holds the result of type coercion for binary operations.
type coercionResult struct {
	// leftExpr is the left operand expression after type coercion.
	leftExpr goast.Expr

	// rightExpr is the right operand expression after type coercion has been applied.
	rightExpr goast.Expr

	// leftType is the type of the left operand after coercion.
	leftType *ast_domain.ResolvedTypeInfo

	// rightType is the type of the right operand after coercion.
	rightType *ast_domain.ResolvedTypeInfo

	// updatedStmts contains any statements generated during type coercion.
	updatedStmts []goast.Stmt
}

// applyTypeCoercion applies numeric coercion for arithmetic operations.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary expression to coerce.
// Takes leftGoExpr (goast.Expr) which is the left operand expression.
// Takes rightGoExpr (goast.Expr) which is the right operand expression.
// Takes allStmts ([]goast.Stmt) which contains statements generated so far.
//
// Returns coercionResult which contains the coerced expressions and types.
func (be *binaryOpEmitter) applyTypeCoercion(
	n *ast_domain.BinaryExpression,
	leftGoExpr goast.Expr,
	rightGoExpr goast.Expr,
	allStmts []goast.Stmt,
) coercionResult {
	leftAnn := getAnnotationFromExpression(n.Left)
	rightAnn := getAnnotationFromExpression(n.Right)

	var finalLeftType *ast_domain.ResolvedTypeInfo
	if leftAnn != nil {
		finalLeftType = leftAnn.ResolvedType
	}
	var finalRightType *ast_domain.ResolvedTypeInfo
	if rightAnn != nil {
		finalRightType = rightAnn.ResolvedType
	}

	if !IsArithmeticOperator(n.Operator) {
		return coercionResult{
			leftExpr:     leftGoExpr,
			rightExpr:    rightGoExpr,
			leftType:     finalLeftType,
			rightType:    finalRightType,
			updatedStmts: allStmts,
		}
	}

	var newLeftStmts, newRightStmts []goast.Stmt
	leftGoExpr, newLeftStmts, finalLeftType = be.coerceToNumber(leftGoExpr, leftAnn)
	rightGoExpr, newRightStmts, finalRightType = be.coerceToNumber(rightGoExpr, rightAnn)
	allStmts = append(allStmts, newLeftStmts...)
	allStmts = append(allStmts, newRightStmts...)

	return coercionResult{
		leftExpr:     leftGoExpr,
		rightExpr:    rightGoExpr,
		leftType:     finalLeftType,
		rightType:    finalRightType,
		updatedStmts: allStmts,
	}
}

// tryEmitLogicalOperator handles logical AND/OR operators and emits the
// appropriate Go expression.
//
// For || (OpOr): Uses EvaluateOr helper which returns the first truthy value
// (JavaScript semantics).
// For && (OpAnd): Uses native Go && wrapped in truthiness checks (returns
// bool).
//
// Takes n (*ast_domain.BinaryExpression) which is the binary expression to emit.
// Takes leftGoExpr (goast.Expr) which is the already-emitted left operand.
// Takes rightGoExpr (goast.Expr) which is the already-emitted right operand.
//
// Returns goast.Expr which is the emitted Go expression, or nil if not
// handled.
// Returns bool which indicates whether the operator was a logical operator.
func (*binaryOpEmitter) tryEmitLogicalOperator(
	n *ast_domain.BinaryExpression,
	leftGoExpr goast.Expr,
	rightGoExpr goast.Expr,
) (goast.Expr, bool) {
	switch n.Operator {
	case ast_domain.OpOr:
		helperCall := callHelper(helperEvaluateOr, leftGoExpr, rightGoExpr)
		return wrapWithTypeAssertion(helperCall, n.GoAnnotations), true

	case ast_domain.OpAnd:
		leftGoExpr = wrapInTruthinessCallIfNeeded(leftGoExpr, n.Left)
		rightGoExpr = wrapInTruthinessCallIfNeeded(rightGoExpr, n.Right)
		return &goast.BinaryExpr{X: leftGoExpr, Op: token.LAND, Y: rightGoExpr}, true

	default:
		return nil, false
	}
}

// binaryOpContext holds the data needed to emit a binary operator.
type binaryOpContext struct {
	// expression is the original binary expression being emitted.
	expression *ast_domain.BinaryExpression

	// leftExpr is the Go expression for the left operand after it has been emitted.
	leftExpr goast.Expr

	// rightExpr is the Go expression for the right operand.
	rightExpr goast.Expr

	// leftType is the resolved type of the left operand.
	leftType *ast_domain.ResolvedTypeInfo

	// rightType is the resolved type of the right operand.
	rightType *ast_domain.ResolvedTypeInfo

	// statements holds statements to emit before the binary operation expression.
	statements []goast.Stmt

	// diagnostics holds any warnings or errors found while processing
	// the binary operation.
	diagnostics []*ast_domain.Diagnostic
}

// emitNativeOrHelperOp determines whether to use a native operator or runtime
// helper.
//
// Takes n (*ast_domain.BinaryExpression) which is the binary expression to emit.
// Takes leftGoExpr (goast.Expr) which is the left operand Go expression.
// Takes rightGoExpr (goast.Expr) which is the right operand Go expression.
// Takes finalLeftType (*ast_domain.ResolvedTypeInfo) which is the resolved
// type of the left operand.
// Takes finalRightType (*ast_domain.ResolvedTypeInfo) which is the resolved
// type of the right operand.
// Takes allStmts ([]goast.Stmt) which contains accumulated statements.
// Takes allDiags ([]*ast_domain.Diagnostic) which contains accumulated
// diagnostics.
//
// Returns goast.Expr which is the resulting Go expression for the operation.
// Returns []goast.Stmt which contains any additional statements needed.
// Returns []*ast_domain.Diagnostic which contains any diagnostics generated.
func (be *binaryOpEmitter) emitNativeOrHelperOp(
	n *ast_domain.BinaryExpression,
	leftGoExpr goast.Expr,
	rightGoExpr goast.Expr,
	finalLeftType *ast_domain.ResolvedTypeInfo,
	finalRightType *ast_domain.ResolvedTypeInfo,
	allStmts []goast.Stmt,
	allDiags []*ast_domain.Diagnostic,
) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	ctx := &binaryOpContext{
		expression:  n,
		leftExpr:    leftGoExpr,
		rightExpr:   rightGoExpr,
		leftType:    finalLeftType,
		rightType:   finalRightType,
		statements:  allStmts,
		diagnostics: allDiags,
	}
	return be.emitNativeOrHelperOpWithContext(ctx)
}

// emitNativeOrHelperOpWithContext handles a binary operation using the given
// context.
//
// Takes ctx (*binaryOpContext) which provides the operands and type details.
//
// Returns goast.Expr which is the resulting binary operation expression.
// Returns []goast.Stmt which holds any statements needed for the operation.
// Returns []*ast_domain.Diagnostic which holds any diagnostics found.
func (be *binaryOpEmitter) emitNativeOrHelperOpWithContext(ctx *binaryOpContext) (goast.Expr, []goast.Stmt, []*ast_domain.Diagnostic) {
	if expression, ok := tryNilComparison(ctx); ok {
		return expression, ctx.statements, ctx.diagnostics
	}

	tempLeftAnn := createTempAnnotation(ctx.leftType)
	tempRightAnn := createTempAnnotation(ctx.rightType)

	if opToken, canUseNative := getNativeBinaryOp(ctx.expression.Operator, tempLeftAnn, tempRightAnn); canUseNative {
		nativeExpr := be.emitNativeBinaryOp(opToken, ctx.leftExpr, ctx.leftType, ctx.rightExpr, ctx.rightType)
		return nativeExpr, ctx.statements, ctx.diagnostics
	}

	return be.emitHelperBinaryOp(ctx.expression, ctx.leftExpr, ctx.rightExpr), ctx.statements, ctx.diagnostics
}

// emitNativeBinaryOp generates a standard Go binary expression (e.g., `a + b`).
// It uses the final, potentially coerced types of its operands to perform
// correct type promotion and casting.
//
// Takes operator (token.Token) which specifies the binary operator to apply.
// Takes left (goast.Expr) which is the left operand expression.
// Takes leftType (*ast_domain.ResolvedTypeInfo) which is the resolved type of
// the left operand.
// Takes right (goast.Expr) which is the right operand expression.
// Takes rightType (*ast_domain.ResolvedTypeInfo) which is the resolved type of
// the right operand.
//
// Returns goast.Expr which is the resulting binary expression with any
// necessary type casts applied.
func (*binaryOpEmitter) emitNativeBinaryOp(operator token.Token, left goast.Expr, leftType *ast_domain.ResolvedTypeInfo, right goast.Expr, rightType *ast_domain.ResolvedTypeInfo) goast.Expr {
	if isNumeric(leftType) && isNumeric(rightType) {
		promotedTypeInfo := promoteNumericTypes(leftType, rightType)
		promotedTypeString := goastutil.ASTToTypeString(promotedTypeInfo.TypeExpression, promotedTypeInfo.PackageAlias)

		leftTypeString := goastutil.ASTToTypeString(leftType.TypeExpression, leftType.PackageAlias)
		rightTypeString := goastutil.ASTToTypeString(rightType.TypeExpression, rightType.PackageAlias)

		if leftTypeString != promotedTypeString {
			left = &goast.CallExpr{Fun: cachedIdent(promotedTypeString), Args: []goast.Expr{left}}
		}
		if rightTypeString != promotedTypeString {
			right = &goast.CallExpr{Fun: cachedIdent(promotedTypeString), Args: []goast.Expr{right}}
		}
	}

	return &goast.BinaryExpr{X: left, Op: operator, Y: right}
}

// emitHelperBinaryOp creates a call to a runtime helper function for complex
// or loose comparisons.
//
// Takes n (*ast_domain.BinaryExpression) which provides the operator
// and annotations.
// Takes left (goast.Expr) which is the left operand expression.
// Takes right (goast.Expr) which is the right operand expression.
//
// Returns goast.Expr which is the helper function call expression.
func (*binaryOpEmitter) emitHelperBinaryOp(n *ast_domain.BinaryExpression, left, right goast.Expr) goast.Expr {
	switch n.Operator {
	case ast_domain.OpEq:
		return callHelper(helperEvaluateStrictEquality, left, right)
	case ast_domain.OpNe:
		return &goast.UnaryExpr{Op: token.NOT, X: callHelper(helperEvaluateStrictEquality, left, right)}
	case ast_domain.OpLooseEq:
		return callHelper(helperEvaluateLooseEquality, left, right)
	case ast_domain.OpLooseNe:
		return &goast.UnaryExpr{Op: token.NOT, X: callHelper(helperEvaluateLooseEquality, left, right)}
	case ast_domain.OpCoalesce:
		helperCall := callHelper(helperEvaluateCoalesce, left, right)
		return wrapWithTypeAssertion(helperCall, n.GoAnnotations)
	default:
		helperCall := callHelper(helperEvaluateBinary, left, strLit(string(n.Operator)), right)
		if isOrderedComparisonOperator(n.Operator) {
			return &goast.TypeAssertExpr{
				X:    helperCall,
				Type: cachedIdent("bool"),
			}
		}
		return wrapWithTypeAssertion(helperCall, resolveArithmeticResultAnnotation(n))
	}
}

// coerceToNumber generates code to convert a boolean expression to an integer
// (0 or 1).
//
// Takes expression (goast.Expr) which is the expression to convert.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides type info.
//
// Returns goast.Expr which is the converted expression.
// Returns []goast.Stmt which contains prerequisite statements for the
// conversion.
// Returns *ast_domain.ResolvedTypeInfo which describes the resulting type.
func (be *binaryOpEmitter) coerceToNumber(expression goast.Expr, ann *ast_domain.GoGeneratorAnnotation) (goast.Expr, []goast.Stmt, *ast_domain.ResolvedTypeInfo) {
	if ann == nil {
		return expression, nil, nil
	}
	if ann.ResolvedType == nil || !isBoolType(ann.ResolvedType.TypeExpression) {
		return expression, nil, ann.ResolvedType
	}
	tempVar := cachedIdent(be.emitter.nextTempName())
	prereqStmts := []goast.Stmt{
		&goast.DeclStmt{Decl: &goast.GenDecl{Tok: token.VAR, Specs: []goast.Spec{
			&goast.ValueSpec{Names: []*goast.Ident{tempVar}, Type: cachedIdent(Int64TypeName)},
		}}},
		&goast.IfStmt{
			Cond: expression,
			Body: &goast.BlockStmt{List: []goast.Stmt{assignExpression(tempVar.Name, intLit(IntValueOne))}},
			Else: &goast.BlockStmt{List: []goast.Stmt{assignExpression(tempVar.Name, intLit(IntValueZero))}},
		},
	}
	newTypeInfo := &ast_domain.ResolvedTypeInfo{
		TypeExpression:       cachedIdent(Int64TypeName),
		PackageAlias:         "",
		CanonicalPackagePath: "",
		IsSynthetic:          false,
	}
	return tempVar, prereqStmts, newTypeInfo
}

// IsArithmeticOperator determines whether a binary operator is an arithmetic
// operator (addition, subtraction, multiplication, division, or modulo).
//
// Takes operator (ast_domain.BinaryOp) which is the binary operator to check.
//
// Returns bool which is true if the operator is arithmetic, false otherwise.
func IsArithmeticOperator(operator ast_domain.BinaryOp) bool {
	switch operator {
	case ast_domain.OpPlus, ast_domain.OpMinus, ast_domain.OpMul, ast_domain.OpDiv, ast_domain.OpMod:
		return true
	default:
		return false
	}
}

// newBinaryOpEmitter creates a new emitter for binary operations.
//
// Takes emitter (*emitter) which provides the base output functions.
// Takes expressionEmitter (ExpressionEmitter) which writes expression output.
//
// Returns *binaryOpEmitter which is ready to write binary operation code.
func newBinaryOpEmitter(emitter *emitter, expressionEmitter ExpressionEmitter) *binaryOpEmitter {
	return &binaryOpEmitter{
		emitter:           emitter,
		expressionEmitter: expressionEmitter,
	}
}

// wrapWithTypeAssertion wraps an expression that returns 'any' with a type
// assertion when the expected type is known from annotations.
//
// Takes expression (goast.Expr) which is the expression to wrap.
// Takes ann (*ast_domain.GoGeneratorAnnotation) which provides the resolved
// type for the assertion.
//
// Returns goast.Expr which is the original expression if no type information
// is available, or a type assertion expression if the annotation contains a
// resolved type.
func wrapWithTypeAssertion(expression goast.Expr, ann *ast_domain.GoGeneratorAnnotation) goast.Expr {
	if ann == nil || ann.ResolvedType == nil || ann.ResolvedType.TypeExpression == nil {
		return expression
	}
	return &goast.TypeAssertExpr{
		X:    expression,
		Type: ann.ResolvedType.TypeExpression,
	}
}

// resolveArithmeticResultAnnotation determines the result type annotation
// for an arithmetic binary expression, using a three-step strategy to
// provide a concrete type assertion that EvaluateBinary (which returns
// 'any') cannot supply on its own.
//
// The strategy tries, in order:
//  1. The BinaryExpr's own annotation, if it has a concrete (non-any) type.
//  2. The left operand's annotation (e.g. from a state.Field reference).
//  3. The left operand's literal type (e.g. DecimalLiteral -> maths.Decimal).
//
// Takes n (*ast_domain.BinaryExpression) which is the arithmetic expression.
//
// Returns *ast_domain.GoGeneratorAnnotation which provides the result type for
// the type assertion, or nil if no type can be determined.
func resolveArithmeticResultAnnotation(n *ast_domain.BinaryExpression) *ast_domain.GoGeneratorAnnotation {
	if ann := n.GoAnnotations; ann != nil && ann.ResolvedType != nil && ann.ResolvedType.TypeExpression != nil {
		if !isAnyType(ann.ResolvedType.TypeExpression) {
			return ann
		}
	}

	if leftAnn := getAnnotationFromExpression(n.Left); leftAnn != nil && leftAnn.ResolvedType != nil && leftAnn.ResolvedType.TypeExpression != nil {
		if !isAnyType(leftAnn.ResolvedType.TypeExpression) {
			return leftAnn
		}
	}

	return inferMathsTypeAnnotation(n.Left)
}

// inferMathsTypeAnnotation creates a synthetic annotation for known maths
// literal types. For nested binary expressions it recurses into the left
// operand to find the underlying literal type.
//
// Takes expression (ast_domain.Expression) which is the expression to inspect.
//
// Returns *ast_domain.GoGeneratorAnnotation with the maths type, or nil if the
// expression is not a maths literal.
func inferMathsTypeAnnotation(expression ast_domain.Expression) *ast_domain.GoGeneratorAnnotation {
	switch e := expression.(type) {
	case *ast_domain.DecimalLiteral:
		_ = e
		return makeMathsAnnotation("Decimal")
	case *ast_domain.BigIntLiteral:
		_ = e
		return makeMathsAnnotation("BigInt")
	case *ast_domain.BinaryExpression:
		return inferMathsTypeAnnotation(e.Left)
	default:
		return nil
	}
}

// makeMathsAnnotation creates a GoGeneratorAnnotation for a maths package type.
//
// Takes typeName (string) which is the type name (e.g. "Decimal", "BigInt").
//
// Returns *ast_domain.GoGeneratorAnnotation with the resolved type set to the
// maths package type.
func makeMathsAnnotation(typeName string) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		ResolvedType: &ast_domain.ResolvedTypeInfo{
			TypeExpression: &goast.SelectorExpr{
				X:   cachedIdent("maths"),
				Sel: cachedIdent(typeName),
			},
			PackageAlias: "maths",
		},
	}
}

// isAnyType reports whether a Go AST type expression represents the untyped
// 'any' or 'interface{}' type.
//
// Takes expression (goast.Expr) which is the type expression to check.
//
// Returns bool which is true when the expression is the identifier 'any'.
func isAnyType(expression goast.Expr) bool {
	identifier, ok := expression.(*goast.Ident)
	return ok && identifier.Name == "any"
}

// tryNilComparison checks if one operand is a nil literal and the other is a
// nillable type (pointer, map, slice, func, chan, interface). If so, it returns
// a direct Go binary expression (e.g., `ptr == nil` or `ptr != nil`).
//
// This optimisation avoids calling EvaluateStrictEquality at runtime, which
// would otherwise fail to correctly compare a typed nil (e.g., (*Image)(nil))
// with an untyped nil due to type information differences.
//
// Takes ctx (*binaryOpContext) which provides the original expression and type
// info.
//
// Returns goast.Expr which is the direct comparison expression if applicable.
// Returns bool which is true if the nil comparison optimisation was applied.
func tryNilComparison(ctx *binaryOpContext) (goast.Expr, bool) {
	operator := ctx.expression.Operator
	if operator != ast_domain.OpEq && operator != ast_domain.OpNe {
		return nil, false
	}

	leftIsNil := isNilLiteral(ctx.expression.Left)
	rightIsNil := isNilLiteral(ctx.expression.Right)

	if !leftIsNil && !rightIsNil {
		return nil, false
	}

	if leftIsNil && rightIsNil {
		goOp := token.EQL
		if operator == ast_domain.OpNe {
			goOp = token.NEQ
		}
		return &goast.BinaryExpr{X: ctx.leftExpr, Op: goOp, Y: ctx.rightExpr}, true
	}

	var nillableType *ast_domain.ResolvedTypeInfo
	var nillableExpr goast.Expr
	var nilExpr goast.Expr

	if leftIsNil {
		nillableType = ctx.rightType
		nillableExpr = ctx.rightExpr
		nilExpr = ctx.leftExpr
	} else {
		nillableType = ctx.leftType
		nillableExpr = ctx.leftExpr
		nilExpr = ctx.rightExpr
	}

	if nillableType == nil || nillableType.TypeExpression == nil {
		return nil, false
	}

	if !isNillableType(nillableType.TypeExpression) {
		return nil, false
	}

	goOp := token.EQL
	if operator == ast_domain.OpNe {
		goOp = token.NEQ
	}

	if leftIsNil {
		return &goast.BinaryExpr{X: nilExpr, Op: goOp, Y: nillableExpr}, true
	}
	return &goast.BinaryExpr{X: nillableExpr, Op: goOp, Y: nilExpr}, true
}

// isNilLiteral checks whether an expression is a nil literal.
//
// Takes expression (ast_domain.Expression) which is the expression to check.
//
// Returns bool which is true if the expression is a NilLiteral.
func isNilLiteral(expression ast_domain.Expression) bool {
	_, ok := expression.(*ast_domain.NilLiteral)
	return ok
}

// createTempAnnotation creates a simple annotation for type checking.
//
// Takes resolvedType (*ast_domain.ResolvedTypeInfo) which provides the type
// data to wrap.
//
// Returns *ast_domain.GoGeneratorAnnotation which holds only the resolved type
// with all other fields set to nil or zero values.
func createTempAnnotation(resolvedType *ast_domain.ResolvedTypeInfo) *ast_domain.GoGeneratorAnnotation {
	return &ast_domain.GoGeneratorAnnotation{
		ResolvedType:            resolvedType,
		EffectiveKeyExpression:  nil,
		PropDataSource:          nil,
		BaseCodeGenVarName:      nil,
		ParentTypeName:          nil,
		GeneratedSourcePath:     nil,
		FieldTag:                nil,
		SourceInvocationKey:     nil,
		OriginalPackageAlias:    nil,
		OriginalSourcePath:      nil,
		DynamicAttributeOrigins: nil,
		Symbol:                  nil,
		PartialInfo:             nil,
		Srcset:                  nil,
		Stringability:           0,
		IsStatic:                false,
		NeedsCSRF:               false,
		NeedsRuntimeSafetyCheck: false,
		IsStructurallyStatic:    false,
		IsPointerToStringable:   false,
		StaticCollectionLiteral: nil,
		StaticCollectionData:    nil,
		DynamicCollectionInfo:   nil,
		IsCollectionCall:        false,
		IsHybridCollection:      false,
		IsMapAccess:             false,
	}
}

// getNativeBinaryOp checks whether a binary operation can use a native Go
// operator instead of a custom implementation.
//
// Takes operator (ast_domain.BinaryOp) which is the binary operation to check.
// Takes leftAnn (*ast_domain.GoGeneratorAnnotation) which provides type
// information for the left operand.
// Takes rightAnn (*ast_domain.GoGeneratorAnnotation) which provides type
// information for the right operand.
//
// Returns token.Token which is the Go token for the native operation, or
// token.ILLEGAL if no native operator applies.
// Returns bool which is true if a native operation can be used.
func getNativeBinaryOp(operator ast_domain.BinaryOp, leftAnn, rightAnn *ast_domain.GoGeneratorAnnotation) (token.Token, bool) {
	if isLogicalOperator(operator) {
		return token.ILLEGAL, true
	}

	if !hasTypeInformation(leftAnn, rightAnn) {
		return token.ILLEGAL, false
	}

	leftTypeInfo, rightTypeInfo := leftAnn.ResolvedType, rightAnn.ResolvedType

	if hasBoolOperand(leftTypeInfo, rightTypeInfo) && !isEqualityOperator(operator) {
		return token.ILLEGAL, false
	}

	if operatorToken, ok := tryEqualityOperation(operator, leftTypeInfo, rightTypeInfo); ok {
		return operatorToken, true
	}

	if operatorToken, ok := tryNumericOperation(operator, leftTypeInfo, rightTypeInfo); ok {
		return operatorToken, true
	}

	if operatorToken, ok := tryStringConcatenation(operator, leftTypeInfo, rightTypeInfo); ok {
		return operatorToken, true
	}

	return token.ILLEGAL, false
}

// isLogicalOperator reports whether the operator is a logical AND or OR.
//
// Takes operator (ast_domain.BinaryOp) which is the binary operator to check.
//
// Returns bool which is true if operator is OpAnd or OpOr.
func isLogicalOperator(operator ast_domain.BinaryOp) bool {
	return operator == ast_domain.OpAnd || operator == ast_domain.OpOr
}

// hasTypeInformation checks if both annotations have full type information.
//
// Takes leftAnn (*ast_domain.GoGeneratorAnnotation) which is the left operand
// annotation.
// Takes rightAnn (*ast_domain.GoGeneratorAnnotation) which is the right
// operand annotation.
//
// Returns bool which is true when both annotations are not nil and have
// resolved types.
func hasTypeInformation(leftAnn, rightAnn *ast_domain.GoGeneratorAnnotation) bool {
	return leftAnn != nil && rightAnn != nil &&
		leftAnn.ResolvedType != nil && rightAnn.ResolvedType != nil
}

// hasBoolOperand checks if either operand is a boolean type.
//
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type info for
// the left operand.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which is the type info for
// the right operand.
//
// Returns bool which is true if either operand has a boolean type.
func hasBoolOperand(leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) bool {
	return isBoolType(leftTypeInfo.TypeExpression) || isBoolType(rightTypeInfo.TypeExpression)
}

// isOrderedComparisonOperator reports whether operator is an ordered comparison
// operator (greater than, less than, or their equal variants).
//
// Takes operator (ast_domain.BinaryOp) which is the binary operator to check.
//
// Returns bool which is true if operator is >, <, >=, or <=.
func isOrderedComparisonOperator(operator ast_domain.BinaryOp) bool {
	switch operator {
	case ast_domain.OpGt, ast_domain.OpLt, ast_domain.OpGe, ast_domain.OpLe:
		return true
	default:
		return false
	}
}

// isEqualityOperator reports whether operator is an equality or inequality
// operator.
//
// Takes operator (ast_domain.BinaryOp) which is the binary operator to check.
//
// Returns bool which is true if operator is an equality or inequality operator.
func isEqualityOperator(operator ast_domain.BinaryOp) bool {
	return operator == ast_domain.OpEq || operator == ast_domain.OpNe ||
		operator == ast_domain.OpLooseEq || operator == ast_domain.OpLooseNe
}

// typesAreComparable checks if two types can be compared for equality.
//
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which is the first type.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which is the second type.
//
// Returns bool which is true when both types are numeric, both are strings,
// or both are booleans.
func typesAreComparable(leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) bool {
	return (isNumeric(leftTypeInfo) && isNumeric(rightTypeInfo)) ||
		(isExpressionStringType(leftTypeInfo) && isExpressionStringType(rightTypeInfo)) ||
		(isBoolType(leftTypeInfo.TypeExpression) && isBoolType(rightTypeInfo.TypeExpression))
}

// tryEqualityOperation finds the Go token for an equality operator.
//
// Takes operator (ast_domain.BinaryOp) which is the equality operator to
// match.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the left
// operand type.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which describes the right
// operand type.
//
// Returns token.Token which is the Go token for the operator, or token.ILLEGAL
// if the types cannot be compared or the operator is not a strict equality
// operator.
// Returns bool which is true when a valid token was found.
func tryEqualityOperation(operator ast_domain.BinaryOp, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) (token.Token, bool) {
	if !typesAreComparable(leftTypeInfo, rightTypeInfo) {
		return token.ILLEGAL, false
	}

	if operator == ast_domain.OpEq {
		return token.EQL, true
	}
	if operator == ast_domain.OpNe {
		return token.NEQ, true
	}

	return token.ILLEGAL, false
}

// tryNumericOperation maps a numeric binary operator to its Go token.
//
// Takes operator (BinaryOp) which specifies the binary operation to map.
// Takes leftTypeInfo (*ResolvedTypeInfo) which describes the left operand type.
// Takes rightTypeInfo (*ResolvedTypeInfo) which describes the right operand
// type.
//
// Returns token.Token which is the Go token for the operation.
// Returns bool which is true when both operands are numeric and the operation
// is supported.
func tryNumericOperation(operator ast_domain.BinaryOp, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) (token.Token, bool) {
	if !isNumeric(leftTypeInfo) || !isNumeric(rightTypeInfo) {
		return token.ILLEGAL, false
	}

	numericOps := map[ast_domain.BinaryOp]token.Token{
		ast_domain.OpPlus:  token.ADD,
		ast_domain.OpMinus: token.SUB,
		ast_domain.OpMul:   token.MUL,
		ast_domain.OpDiv:   token.QUO,
		ast_domain.OpMod:   token.REM,
		ast_domain.OpGt:    token.GTR,
		ast_domain.OpLt:    token.LSS,
		ast_domain.OpGe:    token.GEQ,
		ast_domain.OpLe:    token.LEQ,
	}

	if operatorToken, ok := numericOps[operator]; ok {
		return operatorToken, true
	}

	return token.ILLEGAL, false
}

// tryStringConcatenation attempts string concatenation using the + operator.
//
// Takes operator (ast_domain.BinaryOp) which specifies the binary operation to
// check.
// Takes leftTypeInfo (*ast_domain.ResolvedTypeInfo) which provides the left
// operand type.
// Takes rightTypeInfo (*ast_domain.ResolvedTypeInfo) which provides the right
// operand type.
//
// Returns token.Token which is token.ADD for string concatenation or
// token.ILLEGAL otherwise.
// Returns bool which indicates whether string concatenation applies.
func tryStringConcatenation(operator ast_domain.BinaryOp, leftTypeInfo, rightTypeInfo *ast_domain.ResolvedTypeInfo) (token.Token, bool) {
	if operator == ast_domain.OpPlus && isExpressionStringType(leftTypeInfo) && isExpressionStringType(rightTypeInfo) {
		return token.ADD, true
	}
	return token.ILLEGAL, false
}

// getNumericRank returns a rank value for numeric type promotion.
//
// Takes typeInfo (*ast_domain.ResolvedTypeInfo) which provides the type to
// rank.
//
// Returns int which is the numeric rank for the given type, or
// NumericRankUnknown if the type is nil or not a known numeric type.
func getNumericRank(typeInfo *ast_domain.ResolvedTypeInfo) int {
	if typeInfo == nil || typeInfo.TypeExpression == nil {
		return NumericRankUnknown
	}
	typeString := goastutil.ASTToTypeString(typeInfo.TypeExpression, typeInfo.PackageAlias)
	switch typeString {
	case "float64":
		return NumericRankFloat64
	case "float32":
		return NumericRankFloat32
	case "int64", "uint64":
		return NumericRankInt64
	case "int", "uint", "uintptr":
		return NumericRankInt
	case "int32", "uint32", "rune":
		return NumericRankInt32
	case "int16", "uint16":
		return NumericRankInt16
	case "int8", "uint8", "byte":
		return NumericRankInt8
	default:
		return NumericRankUnknown
	}
}

// promoteNumericTypes returns the type with the higher rank for casting.
//
// Takes left (*ast_domain.ResolvedTypeInfo) which is the first type to compare.
// Takes right (*ast_domain.ResolvedTypeInfo) which is the second type to compare.
//
// Returns *ast_domain.ResolvedTypeInfo which is the type with the higher numeric
// rank, used to find the target type for numeric conversions.
func promoteNumericTypes(left, right *ast_domain.ResolvedTypeInfo) *ast_domain.ResolvedTypeInfo {
	if getNumericRank(left) >= getNumericRank(right) {
		return left
	}
	return right
}
