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

package interp_domain

import (
	"context"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// incDecWrapMsg is the fmt.Errorf wrapper used when forwarding a sub-compiler's
	// increment/decrement error so every dispatch branch returns a uniformly prefixed
	// diagnostic.
	incDecWrapMsg = "compiling increment/decrement: %w"
)

// compileIncDec compiles an increment or decrement statement (x++ or x--).
//
// Takes statement (*ast.IncDecStmt) which is the increment or decrement statement AST
// node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIncDec(ctx context.Context, statement *ast.IncDecStmt) (varLocation, error) {
	if selectorExpression, ok := statement.X.(*ast.SelectorExpr); ok {
		return wrapIncDecResult(c.compileIncDecSelector(ctx, statement, selectorExpression))
	}

	if indexExpression, ok := statement.X.(*ast.IndexExpr); ok {
		return wrapIncDecResult(c.compileIncDecIndex(ctx, statement, indexExpression))
	}

	if starExpression, ok := statement.X.(*ast.StarExpr); ok {
		return wrapIncDecResult(c.compileIncDecStar(ctx, statement, starExpression))
	}

	identifier, ok := statement.X.(*ast.Ident)
	if !ok {
		return varLocation{}, fmt.Errorf("unsupported inc/dec target: %T at %s", statement.X, c.positionString(statement.Pos()))
	}

	if ref, ok := c.upvalueMap[identifier.Name]; ok {
		return wrapIncDecResult(c.compileIncDecUpvalue(ctx, statement, ref))
	}

	if gv, ok := c.globalVariables[identifier.Name]; ok {
		return wrapIncDecResult(c.compileIncDecGlobal(ctx, statement, gv))
	}

	return c.compileIncDecLocal(ctx, statement, identifier)
}

// compileIncDecStar compiles `*p++` or `*p--` by reading through the pointer, applying
// inc/dec, and writing the result back through the same pointer.
//
// Takes statement (*ast.IncDecStmt) which is the inc/dec statement.
// Takes target (*ast.StarExpr) which is the dereference expression.
//
// Returns the location of the post-write value and any compilation error.
func (c *compiler) compileIncDecStar(ctx context.Context, statement *ast.IncDecStmt, target *ast.StarExpr) (varLocation, error) {
	currentLocation, err := c.compileStarExpression(ctx, target)
	if err != nil {
		return varLocation{}, err
	}
	if _, err := c.emitIncDec(ctx, statement.Tok, currentLocation); err != nil {
		return varLocation{}, err
	}
	c.emitNarrowIntegerTruncation(currentLocation, c.incDecStaticType(target))
	if err := c.compileStarAssign(ctx, target, currentLocation); err != nil {
		return varLocation{}, err
	}
	return currentLocation, nil
}

// incDecStaticType returns the static Go type the type-checker recorded for an inc/dec
// target expression, or nil when none is available. Used by inc/dec call sites to detect
// narrow numeric types that need post-write truncation.
//
// Takes targetExpression (ast.Expr) which is the inc/dec target.
//
// Returns the recorded types.Type, or nil when no type is available.
func (c *compiler) incDecStaticType(targetExpression ast.Expr) types.Type {
	if c.info == nil {
		return nil
	}
	if tv, ok := c.info.Types[targetExpression]; ok && tv.Type != nil {
		return tv.Type
	}
	if identifier, ok := targetExpression.(*ast.Ident); ok {
		if obj := c.info.Uses[identifier]; obj != nil {
			return obj.Type()
		}
		if obj := c.info.Defs[identifier]; obj != nil {
			return obj.Type()
		}
	}
	return nil
}

// compileIncDecLocal resolves the identifier against the current scope and emits the
// appropriate inc/dec sequence for a stack-local or spilled variable.
//
// Takes statement (*ast.IncDecStmt) which is the inc/dec statement.
// Takes identifier (*ast.Ident) which names the target variable.
//
// Returns the resulting location and any compilation error.
func (c *compiler) compileIncDecLocal(
	ctx context.Context,
	statement *ast.IncDecStmt,
	identifier *ast.Ident,
) (varLocation, error) {
	location, found := c.scopes.lookupVar(identifier.Name)
	if !found {
		return varLocation{}, fmt.Errorf("undefined variable: %s at %s", identifier.Name, c.positionString(identifier.Pos()))
	}

	staticType := c.incDecStaticType(identifier)

	if location.isSpilled {
		scratch := c.materialise(ctx, location)
		if _, err := c.emitIncDec(ctx, statement.Tok, scratch); err != nil {
			return varLocation{}, err
		}
		c.emitNarrowIntegerTruncation(scratch, staticType)
		c.emitSpillStore(ctx, scratch.register, location.kind, location.spillSlot)
		c.scopes.alloc.freeTemp(location.kind, scratch.register)
		c.emitSyncCaptured(ctx, location)
		return varLocation{}, nil
	}

	if location.isIndirect {
		scratch, err := c.emitIndirectRead(ctx, location)
		if err != nil {
			return varLocation{}, err
		}
		if _, err := c.emitIncDec(ctx, statement.Tok, scratch); err != nil {
			return varLocation{}, err
		}
		c.emitNarrowIntegerTruncation(scratch, staticType)
		c.emitIndirectWrite(ctx, location, scratch)
		c.scopes.alloc.freeTemp(scratch.kind, scratch.register)
		c.emitSyncCaptured(ctx, location)
		return varLocation{}, nil
	}

	result, err := c.emitIncDec(ctx, statement.Tok, location)
	if err != nil {
		return result, err
	}
	c.emitNarrowIntegerTruncation(location, staticType)
	c.emitSyncCaptured(ctx, location)
	return result, nil
}

// compileIncDecIndex compiles m[k]++ and s[i]++ (and their decrement forms) by desugaring
// to m[k] += 1 / m[k] -= 1 and dispatching through the existing compound-assign-index
// path. This covers both map and slice/array targets because the compound path already
// handles both.
//
// Takes statement (*ast.IncDecStmt) which holds the ++/-- token.
// Takes indexExpression (*ast.IndexExpr) which is the target expression.
//
// Returns the compiled location (always zero value) and any error.
func (c *compiler) compileIncDecIndex(ctx context.Context, statement *ast.IncDecStmt, indexExpression *ast.IndexExpr) (varLocation, error) {
	one := &ast.BasicLit{
		ValuePos: statement.Pos(),
		Kind:     token.INT,
		Value:    "1",
	}
	operatorToken := token.ADD
	if statement.Tok == token.DEC {
		operatorToken = token.SUB
	}
	c.populateIncDecLiteralType(indexExpression, one)
	return c.compileCompoundAssignIndex(ctx, indexExpression, one, operatorToken)
}

// populateIncDecLiteralType records a TypeAndValue for the synthetic "1" literal used to
// desugar inc/dec into compound assignment. Without this the compound path reads an empty
// types.Type and mis-classifies the register kind on nested expressions.
//
// Takes indexExpression (*ast.IndexExpr) which identifies the element type.
// Takes literal (*ast.BasicLit) which is the synthetic "1" node.
func (c *compiler) populateIncDecLiteralType(indexExpression *ast.IndexExpr, literal *ast.BasicLit) {
	collectionType, ok := c.info.Types[indexExpression.X]
	if !ok || collectionType.Type == nil {
		return
	}
	var elementType types.Type
	switch collection := collectionType.Type.Underlying().(type) {
	case *types.Map:
		elementType = collection.Elem()
	case *types.Slice:
		elementType = collection.Elem()
	case *types.Array:
		elementType = collection.Elem()
	default:
		return
	}
	if c.info.Types == nil {
		c.info.Types = make(map[ast.Expr]types.TypeAndValue)
	}
	c.info.Types[literal] = types.TypeAndValue{
		Type:  elementType,
		Value: constant.MakeInt64(1),
	}
}

// compileIncDecGlobal compiles an inc/dec on a global variable.
//
// Takes statement (*ast.IncDecStmt) which is the increment or decrement statement AST
// node.
// Takes gv (globalVariableInfo) which is the global variable information for the target.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIncDecGlobal(ctx context.Context, statement *ast.IncDecStmt, gv globalVariableInfo) (varLocation, error) {
	currentLocation := c.emitGetGlobal(ctx, gv)
	if _, err := c.emitIncDec(ctx, statement.Tok, currentLocation); err != nil {
		return varLocation{}, err
	}
	c.emitNarrowIntegerTruncation(currentLocation, c.incDecStaticType(statement.X))
	c.emitSetGlobal(ctx, gv, currentLocation)
	return varLocation{}, nil
}

// compileIncDecSelector compiles s.Field++ or s.Field--.
//
// Takes statement (*ast.IncDecStmt) which is the increment or decrement statement AST
// node.
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the struct field.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIncDecSelector(ctx context.Context, statement *ast.IncDecStmt, selectorExpression *ast.SelectorExpr) (varLocation, error) {
	receiverLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &receiverLocation)

	selection := c.info.Selections[selectorExpression]
	if selection == nil {
		return varLocation{}, fmt.Errorf("unresolved selector: %s", selectorExpression.Sel.Name)
	}
	index := selection.Index()
	fieldIndex := safeconv.MustIntToUint8(index[len(index)-1])

	fieldKind := c.kindFor(selection.Type())
	if fieldKind != registerInt && fieldKind != registerFloat && fieldKind != registerUint {
		return varLocation{}, ErrCompileIncDecSelectorNumeric
	}

	if c.tryEmitFusedIncDecStructField(ctx, statement.Tok, selection, fieldKind, receiverLocation) {
		return varLocation{}, nil
	}

	if fieldKind == registerInt && len(index) == 1 {
		return c.emitTypedIntFieldIncDec(ctx, statement.Tok, selection, receiverLocation, fieldIndex)
	}

	return c.emitBoxedFieldIncDec(ctx, statement.Tok, selection, receiverLocation, fieldIndex, fieldKind)
}

// tryEmitFusedIncDecStructField emits the fused tier-1 subOpIncStructFieldInt/Uint (or
// Dec) super-instruction when the struct layout resolves at compile time AND the field is
// int/uint kind AND no narrow truncation is required AND the layout index fits in uint8.
//
// Saves the opGetFieldInt, tier-2 IncInt, opSetFieldInt three-trip and the int register
// temporary. Float is excluded because emitIncDec for floats already needs a constant
// pool entry and the gain wouldn't justify the second sub-op variant.
//
// Takes operatorToken (token.Token) which is the INC/DEC token.
// Takes selection (*types.Selection) which describes the field access.
// Takes fieldKind (registerKind) which is the field's register kind.
// Takes receiverLocation (varLocation) which is the boxed receiver.
//
// Returns true when the fused opcode was emitted.
func (c *compiler) tryEmitFusedIncDecStructField(ctx context.Context, operatorToken token.Token, selection *types.Selection, fieldKind registerKind, receiverLocation varLocation) bool {
	if fieldKind != registerInt && fieldKind != registerUint {
		return false
	}
	if narrowIntegerBitWidth(selection.Type()) != 0 {
		return false
	}
	layoutIdx, ok := c.tryResolveStructFieldLayout(ctx, selection)
	if !ok {
		return false
	}
	layout := c.function.structLayoutTable[layoutIdx]
	if registerKind(layout.RegisterKind) != fieldKind || !structFieldLayoutIndexFitsTier0(layoutIdx) {
		return false
	}
	sub := pickIncDecStructFieldSubOp(fieldKind, operatorToken)
	if sub == 0 {
		return false
	}
	c.function.emit(opDrillTier1, uint8(sub), receiverLocation.register, safeconv.MustUintToUint8(uint(layoutIdx)))
	return true
}

// emitTypedIntFieldIncDec emits inc/dec on a direct int-kind field.
//
// Emits opGetFieldInt, the inc/dec opcode, an optional narrow-truncation, and
// opSetFieldInt.
//
// Takes ctx (context.Context) which is observed for cancellation during sub-emission.
// Takes operatorToken (token.Token) which is the INC or DEC token.
// Takes selection (*types.Selection) which describes the field.
// Takes receiverLocation (varLocation) which is the boxed receiver.
// Takes fieldIndex (uint8) which is the direct field index.
//
// Returns varLocation which is the empty value location yielded by the statement.
// Returns error when emitIncDec surfaces a sub-emission failure.
func (c *compiler) emitTypedIntFieldIncDec(ctx context.Context, operatorToken token.Token, selection *types.Selection, receiverLocation varLocation, fieldIndex uint8) (varLocation, error) {
	valueRegister := c.scopes.alloc.allocTemp(registerInt)
	valueLocation := varLocation{register: valueRegister, kind: registerInt}
	c.emitTyped(ctx, opGetFieldInt, valueLocation, receiverLocation, rawOperand(fieldIndex))
	if _, err := c.emitIncDec(ctx, operatorToken, valueLocation); err != nil {
		return varLocation{}, err
	}
	c.emitNarrowIntegerTruncation(valueLocation, selection.Type())
	c.emitTyped(ctx, opSetFieldInt, receiverLocation, rawOperand(fieldIndex), valueLocation)
	return varLocation{}, nil
}

// emitBoxedFieldIncDec emits inc/dec on a boxed field.
//
// Used for any field that cannot take the typed-int fast path: unpack the interface
// value, increment or decrement in the field-kind bank, then repack and store.
//
// Takes ctx (context.Context) which is observed for cancellation during sub-emission.
// Takes operatorToken (token.Token) which is the INC or DEC token.
// Takes selection (*types.Selection) which describes the field.
// Takes receiverLocation (varLocation) which is the boxed receiver.
// Takes fieldIndex (uint8) which is the direct field index.
// Takes fieldKind (registerKind) which is the field's register kind.
//
// Returns varLocation which is the empty value location yielded by the statement.
// Returns error when emitIncDec surfaces a sub-emission failure.
func (c *compiler) emitBoxedFieldIncDec(
	ctx context.Context,
	operatorToken token.Token,
	selection *types.Selection,
	receiverLocation varLocation,
	fieldIndex uint8,
	fieldKind registerKind,
) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opGetField, currentRegister, receiverLocation.register, fieldIndex)

	valueRegister := c.scopes.alloc.allocTemp(fieldKind)
	c.function.emit(opUnpackInterface, valueRegister, currentRegister, uint8(fieldKind))
	valueLocation := varLocation{register: valueRegister, kind: fieldKind}
	if _, err := c.emitIncDec(ctx, operatorToken, valueLocation); err != nil {
		return varLocation{}, err
	}
	c.emitNarrowIntegerTruncation(valueLocation, selection.Type())
	genResult := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opPackInterface, genResult, valueRegister, uint8(fieldKind))
	c.function.emit(opSetField, receiverLocation.register, fieldIndex, genResult)

	return varLocation{}, nil
}

// compileIncDecUpvalue compiles an inc/dec on a captured upvalue.
//
// Takes statement (*ast.IncDecStmt) which is the increment or decrement statement AST
// node.
// Takes ref (upvalueReference) which is the upvalue reference for the captured variable.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIncDecUpvalue(ctx context.Context, statement *ast.IncDecStmt, ref upvalueReference) (varLocation, error) {
	currentRegister := c.scopes.alloc.allocTemp(ref.kind)
	c.function.emit(opGetUpvalue, currentRegister, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
	currentLocation := varLocation{register: currentRegister, kind: ref.kind}

	if _, err := c.emitIncDec(ctx, statement.Tok, currentLocation); err != nil {
		c.scopes.alloc.freeTemp(ref.kind, currentRegister)
		return varLocation{}, err
	}
	c.emitNarrowIntegerTruncation(currentLocation, c.incDecStaticType(statement.X))

	c.function.emit(opSetUpvalue, currentRegister, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
	c.scopes.alloc.freeTemp(ref.kind, currentRegister)
	return varLocation{}, nil
}

// emitIncDec emits the actual increment or decrement instruction for a numeric register.
//
// Takes operatorToken (token.Token) which indicates whether this is an increment
// (token.INC) or decrement (token.DEC).
// Takes location (varLocation) which is the register location of the numeric value to
// modify.
//
// Returns the compiled location and any error encountered.
func (c *compiler) emitIncDec(_ context.Context, operatorToken token.Token, location varLocation) (varLocation, error) {
	switch location.kind {
	case registerInt:
		if operatorToken == token.INC {
			c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2IncInt), location.register)
		} else {
			c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DecInt), location.register)
		}
	case registerFloat:
		oneIndex, err := c.function.addFloatConstant(1.0)
		if err != nil {
			return varLocation{}, err
		}
		temporaryRegister := c.scopes.alloc.allocTemp(registerFloat)
		c.function.emitWide(opLoadFloatConst, temporaryRegister, oneIndex)
		if operatorToken == token.INC {
			c.function.emit(opAddFloat, location.register, location.register, temporaryRegister)
		} else {
			c.function.emit(opSubFloat, location.register, location.register, temporaryRegister)
		}
		c.scopes.alloc.freeTemp(registerFloat, temporaryRegister)
	case registerUint:
		if operatorToken == token.INC {
			c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2IncUint), location.register)
		} else {
			c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2DecUint), location.register)
		}
	default:
		return varLocation{}, ErrCompileIncDecRequiresNumeric
	}
	return varLocation{}, nil
}

// multiReturnDeferredStore records a non-direct LHS target whose store instruction must
// be emitted after the call completes.
type multiReturnDeferredStore struct {
	// target specifies the LHS expression that needs a deferred store.
	target ast.Expr

	// sourceLocation specifies the source register location holding the value to store.
	sourceLocation varLocation
}

// callResultKinds determines the register kinds for each result of a call expression.
//
// Takes callExpression (*ast.CallExpr) which is the call expression to determine result
// kinds for.
//
// Returns the slice of register kinds for each result and any error encountered.
func (c *compiler) callResultKinds(_ context.Context, callExpression *ast.CallExpr) ([]registerKind, error) {
	if identifier, ok := callExpression.Fun.(*ast.Ident); ok {
		if funcIndex, found := c.funcTable[identifier.Name]; found {
			return c.rootFunction.functions[funcIndex].resultKinds, nil
		}
	}

	tv := c.info.Types[callExpression.Fun]
	signature, ok := tv.Type.Underlying().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("cannot determine result types for call: %T", callExpression.Fun)
	}
	var kinds []registerKind
	for v := range signature.Results().Variables() {
		kinds = append(kinds, c.kindFor(v.Type()))
	}
	return kinds, nil
}

// compileMultiReturnAssign compiles an assignment from a multi-return function call.
//
// Takes leftHandSideList ([]ast.Expr) which is the left-hand side expressions to assign
// results to.
// Takes callExpression (*ast.CallExpr) which is the multi-return call expression to
// compile.
// Takes isDefine (bool) which indicates whether this is a := define or = assign.
//
// Returns the first result location and any error encountered.
func (c *compiler) compileMultiReturnAssign(ctx context.Context, leftHandSideList []ast.Expr, callExpression *ast.CallExpr, isDefine bool) (varLocation, error) {
	resultKinds, err := c.callResultKinds(ctx, callExpression)
	if err != nil {
		return varLocation{}, err
	}

	returnLocations := make([]varLocation, len(leftHandSideList))
	var deferred []multiReturnDeferredStore

	for i, leftHandSide := range leftHandSideList {
		location, ds, err := c.resolveMultiReturnTarget(ctx, leftHandSide, resultKinds[i], isDefine)
		if err != nil {
			return varLocation{}, err
		}
		returnLocations[i] = location
		if ds != nil {
			deferred = append(deferred, *ds)
		}
	}

	if err := c.emitMultiReturnCall(ctx, callExpression, returnLocations); err != nil {
		return varLocation{}, err
	}

	for i, leftHandSide := range leftHandSideList {
		if identifier, ok := leftHandSide.(*ast.Ident); ok && identifier.Name == blankIdentName {
			c.scopes.alloc.recycleRegister(returnLocations[i].kind, returnLocations[i].register)
		}
	}

	for _, ds := range deferred {
		if err := c.emitDeferredStore(ctx, ds); err != nil {
			return varLocation{}, err
		}
	}

	if len(returnLocations) > 0 {
		return returnLocations[0], nil
	}
	return varLocation{}, nil
}

// resolveMultiReturnTarget resolves a single LHS target for a multi-return assignment.
//
// Takes leftHandSide (ast.Expr) which is the left-hand side expression to resolve.
// Takes kind (registerKind) which is the expected register kind for the result.
// Takes isDefine (bool) which indicates whether this is a := define or = assign.
//
// Returns the target location, an optional deferred store, and any error.
func (c *compiler) resolveMultiReturnTarget(ctx context.Context,
	leftHandSide ast.Expr,
	kind registerKind,
	isDefine bool,
) (varLocation, *multiReturnDeferredStore, error) {
	switch target := leftHandSide.(type) {
	case *ast.Ident:
		return c.resolveMultiReturnIdent(ctx, target, kind, isDefine)

	case *ast.IndexExpr, *ast.SelectorExpr, *ast.StarExpr:
		register := c.scopes.alloc.alloc(kind)
		location := varLocation{register: register, kind: kind}
		ds := &multiReturnDeferredStore{sourceLocation: location, target: leftHandSide}
		return location, ds, nil

	default:
		return varLocation{}, nil, fmt.Errorf("unsupported assignment target: %T at %s", leftHandSide, c.positionString(leftHandSide.Pos()))
	}
}

// resolveMultiReturnIdent resolves an identifier LHS target for a multi-return
// assignment.
//
// Takes target (*ast.Ident) which is the identifier to resolve.
// Takes kind (registerKind) which is the expected register kind for the result.
// Takes isDefine (bool) which indicates whether this is a := define or = assign.
//
// Returns the target location, an optional deferred store, and any error.
func (c *compiler) resolveMultiReturnIdent(ctx context.Context,
	target *ast.Ident,
	kind registerKind,
	isDefine bool,
) (varLocation, *multiReturnDeferredStore, error) {
	if target.Name == blankIdentName {
		register := c.scopes.alloc.alloc(kind)
		return varLocation{register: register, kind: kind}, nil, nil
	}

	if isDefine {
		return c.resolveMultiReturnDefine(ctx, target, kind)
	}

	return c.resolveMultiReturnAssignIdent(ctx, target, kind)
}

// resolveMultiReturnDefine resolves a := target for a multi-return assignment.
//
// Takes target (*ast.Ident) which is the identifier to declare or look up.
// Takes kind (registerKind) which is the register kind for the new variable.
//
// Returns the target location, an optional deferred store, and any error.
func (c *compiler) resolveMultiReturnDefine(_ context.Context,
	target *ast.Ident,
	kind registerKind,
) (varLocation, *multiReturnDeferredStore, error) {
	typeObject := c.info.Defs[target]
	if typeObject != nil {
		location := c.scopes.declareVar(target.Name, kind)
		return location, nil, nil
	}
	location, _ := c.scopes.lookupVar(target.Name)
	return location, nil, nil
}

// resolveMultiReturnAssignIdent resolves a plain = target for a multi-return assignment.
//
// Takes target (*ast.Ident) which is the identifier to look up.
// Takes kind (registerKind) which is the expected register kind for the result.
//
// Returns the target location, an optional deferred store, and any error.
func (c *compiler) resolveMultiReturnAssignIdent(_ context.Context,
	target *ast.Ident,
	kind registerKind,
) (varLocation, *multiReturnDeferredStore, error) {
	if _, ok := c.upvalueMap[target.Name]; ok {
		register := c.scopes.alloc.alloc(kind)
		location := varLocation{register: register, kind: kind}
		return location, &multiReturnDeferredStore{sourceLocation: location, target: target}, nil
	}
	if _, ok := c.globalVariables[target.Name]; ok {
		register := c.scopes.alloc.alloc(kind)
		location := varLocation{register: register, kind: kind}
		return location, &multiReturnDeferredStore{sourceLocation: location, target: target}, nil
	}
	location, _ := c.scopes.lookupVar(target.Name)
	return location, nil, nil
}

// emitMultiReturnCall compiles and emits the call instruction for a multi-return call.
//
// Takes callExpression (*ast.CallExpr) which is the call expression to compile.
// Takes returnLocations ([]varLocation) which is the pre-allocated return locations for
// each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitMultiReturnCall(ctx context.Context, callExpression *ast.CallExpr, returnLocations []varLocation) error {
	callFun := c.unwrapGenericInstantiation(ctx, callExpression.Fun)
	switch fun := callFun.(type) {
	case *ast.Ident:
		if funcIndex, found := c.funcTable[fun.Name]; found {
			callee := c.rootFunction.functions[funcIndex]
			argumentLocations, err := c.compileCallArguments(ctx, callExpression, callee)
			if err != nil {
				return err
			}
			site := callSite{
				funcIndex: funcIndex,
				arguments: argumentLocations,
				returns:   returnLocations,
			}
			siteIndex, addErr := c.function.addCallSite(&site)
			if addErr != nil {
				return addErr
			}
			c.function.emitWide(opCall, 0, siteIndex)
			return nil
		}

		functionLocation, found := c.scopes.lookupVar(fun.Name)
		if !found {
			return fmt.Errorf("undefined function: %s at %s", fun.Name, c.positionString(fun.Pos()))
		}
		argumentLocations, err := c.compileArgumentExpressions(ctx, callExpression)
		if err != nil {
			return err
		}
		argumentTypeNames, argumentTypeStrings := c.resolveArgumentStaticTypes(callExpression)
		site := callSite{
			isNative:                  true,
			nativeRegister:            functionLocation.register,
			arguments:                 argumentLocations,
			returns:                   returnLocations,
			argumentStaticTypeNames:   argumentTypeNames,
			argumentStaticTypeStrings: argumentTypeStrings,
		}
		siteIndex, addErr := c.function.addCallSite(&site)
		if addErr != nil {
			return addErr
		}
		c.function.emitWide(opCallNative, 0, siteIndex)
		return nil

	case *ast.SelectorExpr:
		return c.compileMultiReturnSelectorCall(ctx, fun, callExpression, returnLocations)

	default:
		return fmt.Errorf("unsupported call target: %T at %s", callExpression.Fun, c.positionString(callExpression.Fun.Pos()))
	}
}

// compileArgumentExpressions compiles all argument expressions for a native call.
//
// Bool-typed expressions whose results land in an int register (for example a comparison
// such as `a != ""`) are coerced to a bool register so downstream boxing into an
// interface{} parameter reflects the source type rather than int64. Scalar arguments
// destined for a fixed interface{} parameter are pre-boxed via
// preboxNativeInterfaceArgument so they reach the native callee clothed in their exact
// source-level type (int, int32, ...) rather than the canonical int64 the runtime boxing
// path produces; without this, `v, ok := m.Load(7)` passes int64(7) while `m.Store(7, x)`
// passes int(7), so the sync.Map key lookup misses. The single-return path applies the
// same pre-boxing in compileNativeArguments.
//
// Takes callExpression (*ast.CallExpr) which is the call expression whose arguments are
// compiled.
//
// Returns the compiled argument locations and any error encountered.
func (c *compiler) compileArgumentExpressions(ctx context.Context, callExpression *ast.CallExpr) ([]varLocation, error) {
	nativeSignature := c.nativeCallSignature(callExpression)
	argumentLocations := make([]varLocation, len(callExpression.Args))
	for i, argument := range callExpression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return nil, err
		}
		location = c.coerceEvalBoolResult(ctx, c.info, argument, location)
		argumentLocations[i] = c.preboxNativeInterfaceArgument(nativeSignature, i, location)
	}
	return argumentLocations, nil
}

// compileMultiReturnSelectorCall dispatches a multi-return selector call to the
// appropriate code path.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method or function.
// Takes callExpression (*ast.CallExpr) which is the call expression to compile.
// Takes returnLocations ([]varLocation) which is the pre-allocated return locations for
// each result.
//
// Returns any error encountered during compilation.
func (c *compiler) compileMultiReturnSelectorCall(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpression *ast.CallExpr,
	returnLocations []varLocation,
) error {
	if funcIndex, ok := c.resolveMethodFunc(ctx, selectorExpression); ok {
		fieldPath := c.resolveEmbeddedFieldPath(ctx, selectorExpression)
		return c.emitMethodCallWithReturns(ctx, selectorExpression, callExpression, funcIndex, fieldPath, returnLocations)
	}

	if c.isInterfaceMethodCall(ctx, selectorExpression) {
		return c.emitDynamicMethodCallWithReturns(ctx, selectorExpression, callExpression, returnLocations)
	}

	if handled, err := c.emitLinkedCallWithReturns(ctx, selectorExpression, callExpression, returnLocations); handled || err != nil {
		return err
	}

	return c.emitNativeSelectorCallWithReturns(ctx, selectorExpression, callExpression, returnLocations)
}

// resolveMethodFunc resolves a selector expression to a compiled method function index.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve.
//
// Returns the function index and true if found, or zero and false otherwise.
func (c *compiler) resolveMethodFunc(ctx context.Context, selectorExpression *ast.SelectorExpr) (uint16, bool) {
	tableName, ok := c.resolveMethodTableName(ctx, selectorExpression)
	if !ok {
		return 0, false
	}
	funcIndex, found := c.funcTable[tableName]
	return funcIndex, found
}

// resolveEmbeddedFieldPath returns the field path for an embedded method receiver.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve the embedded path for.
//
// Returns the field index path excluding the final method index, or nil for direct
// methods.
func (c *compiler) resolveEmbeddedFieldPath(_ context.Context, selectorExpression *ast.SelectorExpr) []int {
	selection, ok := c.info.Selections[selectorExpression]
	if !ok {
		return nil
	}
	index := selection.Index()
	if len(index) <= 1 {
		return nil
	}
	return index[:len(index)-1]
}

// emitMethodCallWithReturns compiles an interpreted method call with pre-allocated return
// locations.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes callExpression (*ast.CallExpr) which is the call expression to compile.
// Takes funcIndex (uint16) which is the function index of the compiled method.
// Takes fieldPath ([]int) which is the embedded field path to the receiver, or nil.
// Takes returnLocations ([]varLocation) which is the pre-allocated return locations for
// each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitMethodCallWithReturns(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpression *ast.CallExpr,
	funcIndex uint16,
	fieldPath []int,
	returnLocations []varLocation,
) error {
	receiverLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return err
	}
	c.boxToGeneral(ctx, &receiverLocation)

	for _, fieldIndex := range fieldPath {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, receiverLocation.register, safeconv.MustIntToUint8(fieldIndex))
		receiverLocation = varLocation{register: dest, kind: registerGeneral}
	}

	argumentLocations := make([]varLocation, 0, 1+len(callExpression.Args))
	argumentLocations = append(argumentLocations, receiverLocation)
	for _, argument := range callExpression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return err
		}
		argumentLocations = append(argumentLocations, location)
	}

	site := callSite{
		funcIndex: funcIndex,
		arguments: argumentLocations,
		returns:   returnLocations,
	}
	siteIndex, addErr := c.function.addCallSite(&site)
	if addErr != nil {
		return addErr
	}
	c.function.emitWide(opCall, 0, siteIndex)
	return nil
}

// emitDynamicMethodCallWithReturns compiles an interface method call with pre-allocated
// return locations.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the interface method.
// Takes callExpression (*ast.CallExpr) which is the call expression to compile.
// Takes returnLocations ([]varLocation) which is the pre-allocated return locations for
// each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitDynamicMethodCallWithReturns(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpression *ast.CallExpr,
	returnLocations []varLocation,
) error {
	receiverLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return err
	}
	c.boxToGeneral(ctx, &receiverLocation)

	argumentLocations := make([]varLocation, 0, 1+len(callExpression.Args))
	argumentLocations = append(argumentLocations, receiverLocation)
	for _, argument := range callExpression.Args {
		location, err := c.compileExpression(ctx, argument)
		if err != nil {
			return err
		}
		argumentLocations = append(argumentLocations, location)
	}

	methodIndex, methodErr := c.function.addStringConstant(selectorExpression.Sel.Name)
	if methodErr != nil {
		return methodErr
	}

	site := callSite{
		arguments: argumentLocations,
		returns:   returnLocations,
	}
	siteIndex, addErr := c.function.addCallSite(&site)
	if addErr != nil {
		return addErr
	}
	c.function.emitWide(opCallMethod, 0, siteIndex)
	c.function.emitExtension(methodIndex, 0)
	return nil
}

// emitNativeSelectorCallWithReturns compiles a native selector call with pre-allocated
// return locations.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the native function or method.
// Takes callExpression (*ast.CallExpr) which is the call expression to compile.
// Takes returnLocations ([]varLocation) which is the pre-allocated return locations for
// each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitNativeSelectorCallWithReturns(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpression *ast.CallExpr,
	returnLocations []varLocation,
) error {
	functionLocation, err := c.compileSelectorExpression(ctx, selectorExpression)
	if err != nil {
		return err
	}

	argumentLocations, err := c.compileArgumentExpressions(ctx, callExpression)
	if err != nil {
		return err
	}

	argumentTypeNames, argumentTypeStrings := c.resolveArgumentStaticTypes(callExpression)
	site := callSite{
		isNative:                  true,
		nativeRegister:            functionLocation.register,
		arguments:                 argumentLocations,
		returns:                   returnLocations,
		argumentStaticTypeNames:   argumentTypeNames,
		argumentStaticTypeStrings: argumentTypeStrings,
	}
	siteIndex, addErr := c.function.addCallSite(&site)
	if addErr != nil {
		return addErr
	}
	c.function.emitWide(opCallNative, 0, siteIndex)
	return nil
}

// emitDeferredStore emits a store instruction for a multi-return LHS target that could
// not be written directly.
//
// Takes ds (multiReturnDeferredStore) which is the deferred store record containing the
// target and source location.
//
// Returns any error encountered during the store emission.
func (c *compiler) emitDeferredStore(ctx context.Context, ds multiReturnDeferredStore) error {
	switch target := ds.target.(type) {
	case *ast.Ident:
		if ref, ok := c.upvalueMap[target.Name]; ok {
			c.function.emit(opSetUpvalue, ds.sourceLocation.register, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
			return nil
		}
		if gv, ok := c.globalVariables[target.Name]; ok {
			c.emitSetGlobal(ctx, gv, ds.sourceLocation)
			return nil
		}
		return fmt.Errorf("deferred store: variable %s not found at %s", target.Name, c.positionString(target.Pos()))
	case *ast.IndexExpr:
		return c.compileIndexAssign(ctx, target, ds.sourceLocation)
	case *ast.SelectorExpr:
		return c.compileSelectorAssign(ctx, target, ds.sourceLocation)
	case *ast.StarExpr:
		return c.compileStarAssign(ctx, target, ds.sourceLocation)
	default:
		return fmt.Errorf("unsupported deferred store target: %T at %s", ds.target, c.positionString(ds.target.Pos()))
	}
}

// declareCommaOkTargets declares or looks up the value and ok destination variables for
// comma-ok assignments.
//
// Takes valueIdentifier (*ast.Ident) which is the identifier for the value target.
// Takes okIdentifier (*ast.Ident) which is the identifier for the ok boolean target.
// Takes valueKind (registerKind) which is the register kind for the value variable.
// Takes blankValueKind (registerKind) which is the register kind to use when the value
// target is blank.
// Takes isDefine (bool) which indicates whether this is a := define or = assign.
//
// Returns the value destination location and the ok destination location.
func (c *compiler) declareCommaOkTargets(
	ctx context.Context,
	valueIdentifier, okIdentifier *ast.Ident,
	valueKind registerKind,
	blankValueKind registerKind,
	isDefine bool,
) (valueDestination varLocation, okDestination varLocation) {
	if isDefine {
		valueDestination = c.declareCommaOkValue(ctx, valueIdentifier, valueKind, blankValueKind)
		okDestination = c.declareCommaOkBool(ctx, okIdentifier)
	} else {
		valueDestination = c.lookupCommaOkValue(ctx, valueIdentifier, blankValueKind)
		okDestination = c.lookupCommaOkBool(ctx, okIdentifier)
	}

	return valueDestination, okDestination
}

// declareCommaOkValue declares or resolves the value target for a comma-ok define.
//
// Takes valueIdentifier (*ast.Ident) which is the identifier for the value target.
// Takes valueKind (registerKind) which is the register kind for the value variable.
// Takes blankValueKind (registerKind) which is the register kind to use when the target
// is blank.
//
// Returns the resolved value location.
func (c *compiler) declareCommaOkValue(_ context.Context, valueIdentifier *ast.Ident, valueKind registerKind, blankValueKind registerKind) varLocation {
	if valueIdentifier.Name != blankIdentName {
		typeObject := c.info.Defs[valueIdentifier]
		if typeObject != nil {
			return c.scopes.declareVar(valueIdentifier.Name, valueKind)
		}
		location, _ := c.scopes.lookupVar(valueIdentifier.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(blankValueKind), kind: blankValueKind}
}

// declareCommaOkBool declares or resolves the ok target for a comma-ok define.
//
// Takes okIdentifier (*ast.Ident) which is the identifier for the ok boolean target.
//
// Returns the resolved ok location.
func (c *compiler) declareCommaOkBool(_ context.Context, okIdentifier *ast.Ident) varLocation {
	if okIdentifier.Name != blankIdentName {
		typeObject := c.info.Defs[okIdentifier]
		if typeObject != nil {
			return c.scopes.declareVar(okIdentifier.Name, registerInt)
		}
		location, _ := c.scopes.lookupVar(okIdentifier.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(registerInt), kind: registerInt}
}

// lookupCommaOkValue looks up the value target for a comma-ok assign.
//
// Takes valueIdentifier (*ast.Ident) which is the identifier for the value target.
// Takes blankValueKind (registerKind) which is the register kind to use when the target
// is blank.
//
// Returns the resolved value location.
func (c *compiler) lookupCommaOkValue(_ context.Context, valueIdentifier *ast.Ident, blankValueKind registerKind) varLocation {
	if valueIdentifier.Name != blankIdentName {
		location, _ := c.scopes.lookupVar(valueIdentifier.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(blankValueKind), kind: blankValueKind}
}

// lookupCommaOkBool looks up the ok target for a comma-ok assign.
//
// Takes okIdentifier (*ast.Ident) which is the identifier for the ok boolean target.
//
// Returns the resolved ok location.
func (c *compiler) lookupCommaOkBool(_ context.Context, okIdentifier *ast.Ident) varLocation {
	if okIdentifier.Name != blankIdentName {
		location, _ := c.scopes.lookupVar(okIdentifier.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(registerInt), kind: registerInt}
}

// compileMapCommaOk compiles v, ok := m[k] or v, ok = m[k].
//
// Takes leftHandSideList ([]ast.Expr) which is the left-hand side expressions for value
// and ok targets.
// Takes indexExpression (*ast.IndexExpr) which is the map index expression to compile.
// Takes isDefine (bool) which indicates whether this is a := define or = assign.
//
// Returns the value destination location and any error encountered.
func (c *compiler) compileMapCommaOk(ctx context.Context, leftHandSideList []ast.Expr, indexExpression *ast.IndexExpr, isDefine bool) (varLocation, error) {
	mapLocation, err := c.compileExpression(ctx, indexExpression.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &mapLocation)
	keyLocation, err := c.compileExpression(ctx, indexExpression.Index)
	if err != nil {
		return varLocation{}, err
	}
	valueIdentifier, okIdentifier, err := mapCommaOkTargetIdents(leftHandSideList)
	if err != nil {
		return varLocation{}, err
	}
	valueKind, keyKind, elementType, err := c.mapCommaOkKinds(indexExpression, isDefine)
	if err != nil {
		return varLocation{}, err
	}
	if op, ok := selectTypedMapIndexOkOpcode(keyKind, valueKind, keyLocation.kind); ok {
		return c.emitTypedMapCommaOk(ctx, typedMapCommaOkRequest{
			valueIdentifier: valueIdentifier,
			okIdentifier:    okIdentifier,
			mapLocation:     mapLocation,
			keyLocation:     keyLocation,
			op:              op,
			valueKind:       valueKind,
			isDefine:        isDefine,
		}), nil
	}
	c.boxToGeneral(ctx, &keyLocation)
	valueDestination, okDestination := c.declareCommaOkTargets(ctx, valueIdentifier, okIdentifier, valueKind, registerGeneral, isDefine)
	genDest := c.scopes.alloc.alloc(registerGeneral)
	okRegister := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opMapIndexOk, genDest, mapLocation.register, keyLocation.register)
	c.function.emit(opExt, okRegister, 0, 0)
	c.storeCommaOkResult(ctx, valueDestination, genDest, elementType)
	c.emitMove(ctx, okDestination, varLocation{register: okRegister, kind: registerInt})
	return valueDestination, nil
}

// mapCommaOkKinds resolves the value kind, key kind, and value element type from a map
// index expression. For define-form (`v, ok := m[k]`) the source must be a map type; the
// assign-form falls back to general-bank kinds when the type checker hasn't recorded a
// map.
//
// Takes indexExpression (*ast.IndexExpr) which is the map index.
// Takes isDefine (bool) which differentiates `:=` from `=`.
//
// Returns the value kind, key kind, value element type, and any validation error.
func (c *compiler) mapCommaOkKinds(indexExpression *ast.IndexExpr, isDefine bool) (valueKind registerKind, keyKind registerKind, elementType types.Type, err error) {
	mapType, ok := c.info.Types[indexExpression.X].Type.Underlying().(*types.Map)
	if !ok {
		if isDefine {
			return registerGeneral, registerGeneral, nil, ErrCompileMapCommaOkSourceNotMap
		}
		return registerGeneral, registerGeneral, nil, nil
	}
	elementType = mapType.Elem()
	return c.kindFor(elementType), c.kindFor(mapType.Key()), elementType, nil
}

// typedMapCommaOkRequest bundles the parameters of emitTypedMapCommaOk so the helper
// signature stays under the argument-count cap and the call site reads each field by
// name.
type typedMapCommaOkRequest struct {
	// valueIdentifier is the LHS identifier receiving the matched value.
	valueIdentifier *ast.Ident

	// okIdentifier is the LHS identifier receiving the presence bool.
	okIdentifier *ast.Ident

	// mapLocation is the compiled location of the source map register.
	mapLocation varLocation

	// keyLocation is the compiled location of the lookup key.
	keyLocation varLocation

	// op is the typed comma-ok opcode to emit.
	op opcode

	// valueKind is the register kind for the value destination.
	valueKind registerKind

	// isDefine reports whether the assignment uses := (declare) rather than = (assign).
	isDefine bool
}

// emitTypedMapCommaOk emits the typed comma-ok bytecode (typed key, typed value, ok flag)
// for the matched primitive (key, value) kinds. Allocates the typed destination, emits
// the op + extension word, and moves the result into the declared targets.
//
// Takes request (typedMapCommaOkRequest) which carries the matched opcode, identifiers,
// register locations, value kind, and the declare-vs-assign flag.
//
// Returns the value destination location.
func (c *compiler) emitTypedMapCommaOk(ctx context.Context, request typedMapCommaOkRequest) varLocation {
	valueDestination, okDestination := c.declareCommaOkTargets(ctx, request.valueIdentifier, request.okIdentifier, request.valueKind, registerGeneral, request.isDefine)
	typedDest := c.scopes.alloc.alloc(request.valueKind)
	okRegister := c.scopes.alloc.alloc(registerInt)
	c.function.emit(request.op, typedDest, request.mapLocation.register, request.keyLocation.register)
	c.function.emit(opExt, okRegister, 0, 0)
	if valueDestination.isSpilled {
		c.emitSpillStore(ctx, typedDest, request.valueKind, valueDestination.spillSlot)
	} else if valueDestination.register != typedDest || valueDestination.kind != request.valueKind {
		c.emitMove(ctx, valueDestination, varLocation{register: typedDest, kind: request.valueKind})
	}
	c.emitMove(ctx, okDestination, varLocation{register: okRegister, kind: registerInt})
	return valueDestination
}

// storeCommaOkResult writes a comma-ok value into its declared target.
//
// Moves the boxed general-bank value at genDest into valueDestination, choosing
// emitMoveTyped for general-bank destinations, opUnpackInterface for typed-bank
// destinations, and the spill-store pathway when valueDestination is spilled. Used by the
// `v, ok := ...` shapes (map index, type assertion) that dispatch through opMapIndexOk /
// opTypeAssert and write to a typed bank destination via the general scratch.
//
// Takes valueDestination (varLocation) which is the destination register location of the
// comma-ok value.
// Takes genDest (uint8) which is the general scratch register holding the boxed value
// emitted by the producing opcode.
// Takes elementType (types.Type) which carries the static type of the comma-ok value,
// used by emitMoveTyped to pick the right opMoveGeneral snapshot mode. May be nil.
func (c *compiler) storeCommaOkResult(ctx context.Context, valueDestination varLocation, genDest uint8, elementType types.Type) {
	switch {
	case valueDestination.kind == registerGeneral:
		c.emitMoveTyped(ctx, valueDestination, varLocation{register: genDest, kind: registerGeneral}, elementType)
	case valueDestination.isSpilled:
		scratch := c.scopes.alloc.allocTemp(valueDestination.kind)
		c.function.emit(opUnpackInterface, scratch, genDest, uint8(valueDestination.kind))
		c.emitSpillStore(ctx, scratch, valueDestination.kind, valueDestination.spillSlot)
		c.scopes.alloc.freeTemp(valueDestination.kind, scratch)
	default:
		c.function.emit(opUnpackInterface, valueDestination.register, genDest, uint8(valueDestination.kind))
	}
}

// compileChannelReceiveCommaOk compiles v, ok := <-ch or v, ok = <-ch.
//
// Takes leftHandSideList ([]ast.Expr) which is the left-hand side expressions for value
// and ok targets.
// Takes unaryExpression (*ast.UnaryExpr) which is the channel receive expression to
// compile.
// Takes isDefine (bool) which indicates whether this is a := define or = assign.
//
// Returns the value destination location and any error encountered.
func (c *compiler) compileChannelReceiveCommaOk(ctx context.Context, leftHandSideList []ast.Expr, unaryExpression *ast.UnaryExpr, isDefine bool) (varLocation, error) {
	channelLocation, err := c.compileExpression(ctx, unaryExpression.X)
	if err != nil {
		return varLocation{}, err
	}

	tv := c.info.Types[unaryExpression.X]
	channelType, ok := tv.Type.Underlying().(*types.Chan)
	if !ok {
		return varLocation{}, ErrCompileChanRecvCommaOkSourceNotChan
	}
	elementType := channelType.Elem()
	valueKind := c.kindFor(elementType)

	valueIdentifier, ok := leftHandSideList[0].(*ast.Ident)
	if !ok {
		return varLocation{}, ErrCompileChanRecvCommaOkValueNotIdent
	}
	okIdentifier, ok := leftHandSideList[1].(*ast.Ident)
	if !ok {
		return varLocation{}, ErrCompileChanRecvCommaOkOkNotIdent
	}

	valueDestination, okDestination := c.declareCommaOkTargets(ctx, valueIdentifier, okIdentifier, valueKind, valueKind, isDefine)

	okRegister := c.scopes.alloc.alloc(registerInt)
	destinationRegister := c.scopes.alloc.alloc(valueKind)
	c.function.emit(opDrillTier1, uint8(subOpChannelReceive), channelLocation.register, okRegister)
	c.function.emit(opExt, destinationRegister, uint8(valueKind), 0)

	c.emitMoveTyped(ctx, valueDestination, varLocation{register: destinationRegister, kind: valueKind}, elementType)
	c.emitMove(ctx, okDestination, varLocation{register: okRegister, kind: registerInt})

	return valueDestination, nil
}

// compileTypeAssertCommaOk compiles v, ok := x.(T) or v, ok = x.(T).
//
// Takes leftHandSideList ([]ast.Expr) which is the left-hand side expressions for value
// and ok targets.
// Takes assertExpr (*ast.TypeAssertExpr) which is the type assertion expression to
// compile.
// Takes isDefine (bool) which indicates whether this is a := define or = assign.
//
// Returns the value destination location and any error encountered.
func (c *compiler) compileTypeAssertCommaOk(ctx context.Context, leftHandSideList []ast.Expr, assertExpr *ast.TypeAssertExpr, isDefine bool) (varLocation, error) {
	sourceLocation, err := c.compileExpression(ctx, assertExpr.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &sourceLocation)

	targetType := c.info.Types[assertExpr.Type].Type
	reflectType := c.typeAssertReflectType(ctx, targetType)
	methodNames := interfaceTargetMethodNames(targetType)
	typeIndex, typeErr := c.function.addTypeRefWithMethods(reflectType, methodNames)
	if typeErr != nil {
		return varLocation{}, typeErr
	}
	valueKind := c.kindFor(targetType)

	valueIdentifier, ok := leftHandSideList[0].(*ast.Ident)
	if !ok {
		return varLocation{}, ErrCompileTypeAssertCommaOkValueNotIdent
	}
	okIdentifier, ok := leftHandSideList[1].(*ast.Ident)
	if !ok {
		return varLocation{}, ErrCompileTypeAssertCommaOkOkNotIdent
	}

	valueDestination, okDestination := c.declareCommaOkTargets(ctx, valueIdentifier, okIdentifier, valueKind, registerGeneral, isDefine)

	genDest := c.scopes.alloc.alloc(registerGeneral)
	okRegister := c.scopes.alloc.alloc(registerInt)

	c.function.emit(opTypeAssert, genDest, sourceLocation.register, okRegister)
	c.function.emitExtension(typeIndex, 0)

	c.storeCommaOkResult(ctx, valueDestination, genDest, targetType)
	c.emitMove(ctx, okDestination, varLocation{register: okRegister, kind: registerInt})

	return valueDestination, nil
}

// wrapIncDecResult forwards a sub-compiler's result pair, wrapping any error with the
// shared incDecWrapMsg.
//
// Takes result (varLocation) which is the sub-compiler's location.
// Takes err (error) which is the sub-compiler's error, possibly nil.
//
// Returns the result unchanged on success, or a zero location with the wrapped error.
func wrapIncDecResult(result varLocation, err error) (varLocation, error) {
	if err != nil {
		return varLocation{}, fmt.Errorf(incDecWrapMsg, err)
	}
	return result, nil
}

// pickIncDecStructFieldSubOp selects the fused inc/dec sub-opcode for the given field
// kind and INC/DEC token. Returns 0 when no fused variant exists.
//
// Takes fieldKind (registerKind) which is the field's register kind.
// Takes operatorToken (token.Token) which is the INC or DEC token.
//
// Returns the chosen sub-opcode, or 0 when there is no fused variant.
func pickIncDecStructFieldSubOp(fieldKind registerKind, operatorToken token.Token) subOpcode {
	switch {
	case fieldKind == registerInt && operatorToken == token.INC:
		return subOpIncStructFieldInt
	case fieldKind == registerInt && operatorToken == token.DEC:
		return subOpDecStructFieldInt
	case fieldKind == registerUint && operatorToken == token.INC:
		return subOpIncStructFieldUint
	case fieldKind == registerUint && operatorToken == token.DEC:
		return subOpDecStructFieldUint
	}
	return 0
}

// mapCommaOkTargetIdents extracts the value and ok identifier targets from a comma-ok
// left-hand side, returning a descriptive error when either target is not a bare
// identifier.
//
// Takes leftHandSideList ([]ast.Expr) which is the LHS of the comma-ok assignment.
//
// Returns the value identifier, the ok identifier, and any validation error.
func mapCommaOkTargetIdents(leftHandSideList []ast.Expr) (valueIdentifier *ast.Ident, okIdentifier *ast.Ident, err error) {
	value, ok := leftHandSideList[0].(*ast.Ident)
	if !ok {
		return nil, nil, ErrCompileMapCommaOkValueNotIdent
	}
	okIdent, ok := leftHandSideList[1].(*ast.Ident)
	if !ok {
		return nil, nil, ErrCompileMapCommaOkOkNotIdent
	}
	return value, okIdent, nil
}
