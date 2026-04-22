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
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"

	"piko.sh/piko/wdk/safeconv"
)

// incDecWrapMsg is the fmt.Errorf wrapper used when forwarding a
// sub-compiler's increment/decrement error so every dispatch branch
// returns a uniformly prefixed diagnostic.
const incDecWrapMsg = "compiling increment/decrement: %w"

// compileIncDec compiles an increment or decrement statement (x++ or
// x--).
//
// Takes statement (*ast.IncDecStmt) which is the increment or decrement
// statement AST node to compile.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIncDec(ctx context.Context, statement *ast.IncDecStmt) (varLocation, error) {
	if selExpr, ok := statement.X.(*ast.SelectorExpr); ok {
		return wrapIncDecResult(c.compileIncDecSelector(ctx, statement, selExpr))
	}

	if indexExpr, ok := statement.X.(*ast.IndexExpr); ok {
		return wrapIncDecResult(c.compileIncDecIndex(ctx, statement, indexExpr))
	}

	identifier, ok := statement.X.(*ast.Ident)
	if !ok {
		return varLocation{}, fmt.Errorf("unsupported inc/dec target: %T at %s", statement.X, c.positionString(statement.Pos()))
	}

	if ref, ok := c.upvalueMap[identifier.Name]; ok {
		return wrapIncDecResult(c.compileIncDecUpvalue(ctx, statement, ref))
	}

	if gv, ok := c.globalVars[identifier.Name]; ok {
		return wrapIncDecResult(c.compileIncDecGlobal(ctx, statement, gv))
	}

	return c.compileIncDecLocal(ctx, statement, identifier)
}

// wrapIncDecResult forwards a sub-compiler's result pair, wrapping
// any error with the shared incDecWrapMsg.
//
// Takes result (varLocation) which is the sub-compiler's location.
// Takes err (error) which is the sub-compiler's error, possibly nil.
//
// Returns the result unchanged on success, or a zero location with
// the wrapped error.
func wrapIncDecResult(result varLocation, err error) (varLocation, error) {
	if err != nil {
		return varLocation{}, fmt.Errorf(incDecWrapMsg, err)
	}
	return result, nil
}

// compileIncDecLocal resolves the identifier against the current
// scope and emits the appropriate inc/dec sequence for a stack-local
// or spilled variable.
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

	if location.isSpilled {
		scratch := c.materialise(ctx, location)
		if _, err := c.emitIncDec(ctx, statement.Tok, scratch); err != nil {
			return varLocation{}, err
		}
		c.emitSpillStore(ctx, scratch.register, location.kind, location.spillSlot)
		c.scopes.alloc.freeTemp(location.kind, scratch.register)
		c.emitSyncCaptured(ctx, location)
		return varLocation{}, nil
	}

	result, err := c.emitIncDec(ctx, statement.Tok, location)
	if err != nil {
		return result, err
	}
	c.emitSyncCaptured(ctx, location)
	return result, nil
}

// compileIncDecIndex compiles m[k]++ and s[i]++ (and their -- forms)
// by desugaring to m[k] += 1 / m[k] -= 1 and dispatching through the
// existing compound-assign-index path. This covers both map and
// slice/array targets because the compound path already handles both.
//
// Takes statement (*ast.IncDecStmt) which holds the ++/-- token.
// Takes indexExpr (*ast.IndexExpr) which is the target expression.
//
// Returns the compiled location (always zero value) and any error.
func (c *compiler) compileIncDecIndex(ctx context.Context, statement *ast.IncDecStmt, indexExpr *ast.IndexExpr) (varLocation, error) {
	one := &ast.BasicLit{
		ValuePos: statement.Pos(),
		Kind:     token.INT,
		Value:    "1",
	}
	operatorToken := token.ADD
	if statement.Tok == token.DEC {
		operatorToken = token.SUB
	}
	c.populateIncDecLiteralType(indexExpr, one)
	return c.compileCompoundAssignIndex(ctx, indexExpr, one, operatorToken)
}

// populateIncDecLiteralType records a TypeAndValue for the synthetic
// "1" literal used to desugar inc/dec into compound assignment. Without
// this the compound path reads an empty types.Type and mis-classifies
// the register kind on nested expressions.
//
// Takes indexExpr (*ast.IndexExpr) which identifies the element type.
// Takes literal (*ast.BasicLit) which is the synthetic "1" node.
func (c *compiler) populateIncDecLiteralType(indexExpr *ast.IndexExpr, literal *ast.BasicLit) {
	collectionType, ok := c.info.Types[indexExpr.X]
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
// Takes statement (*ast.IncDecStmt) which is the increment or decrement
// statement AST node.
// Takes gv (globalVariableInfo) which is the global variable information
// for the target.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIncDecGlobal(ctx context.Context, statement *ast.IncDecStmt, gv globalVariableInfo) (varLocation, error) {
	currentLocation := c.emitGetGlobal(ctx, gv)
	if _, err := c.emitIncDec(ctx, statement.Tok, currentLocation); err != nil {
		return varLocation{}, err
	}
	c.emitSetGlobal(ctx, gv, currentLocation)
	return varLocation{}, nil
}

// compileIncDecSelector compiles s.Field++ or s.Field--.
//
// Takes statement (*ast.IncDecStmt) which is the increment or decrement
// statement AST node.
// Takes selExpr (*ast.SelectorExpr) which is the selector expression
// identifying the struct field.
//
// Returns the compiled location and any error encountered.
func (c *compiler) compileIncDecSelector(ctx context.Context, statement *ast.IncDecStmt, selExpr *ast.SelectorExpr) (varLocation, error) {
	recvLocation, err := c.compileExpression(ctx, selExpr.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &recvLocation)

	selection := c.info.Selections[selExpr]
	if selection == nil {
		return varLocation{}, fmt.Errorf("unresolved selector: %s", selExpr.Sel.Name)
	}
	index := selection.Index()
	fieldIndex := safeconv.MustIntToUint8(index[len(index)-1])

	fieldKind := kindForType(selection.Type())
	if fieldKind != registerInt && fieldKind != registerFloat && fieldKind != registerUint {
		return varLocation{}, errors.New("inc/dec on selector requires numeric field")
	}

	if fieldKind == registerInt && len(index) == 1 {
		valReg := c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opGetFieldInt, valReg, recvLocation.register, fieldIndex)
		valLocation := varLocation{register: valReg, kind: registerInt}
		if _, err := c.emitIncDec(ctx, statement.Tok, valLocation); err != nil {
			return varLocation{}, err
		}
		c.function.emit(opSetFieldInt, recvLocation.register, fieldIndex, valReg)
		return varLocation{}, nil
	}

	currentRegister := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opGetField, currentRegister, recvLocation.register, fieldIndex)

	valReg := c.scopes.alloc.allocTemp(fieldKind)
	c.function.emit(opUnpackInterface, valReg, currentRegister, uint8(fieldKind))
	valLocation := varLocation{register: valReg, kind: fieldKind}
	if _, err := c.emitIncDec(ctx, statement.Tok, valLocation); err != nil {
		return varLocation{}, err
	}
	genResult := c.scopes.alloc.allocTemp(registerGeneral)
	c.function.emit(opPackInterface, genResult, valReg, uint8(fieldKind))
	c.function.emit(opSetField, recvLocation.register, fieldIndex, genResult)

	return varLocation{}, nil
}

// compileIncDecUpvalue compiles an inc/dec on a captured upvalue.
//
// Takes statement (*ast.IncDecStmt) which is the increment or decrement
// statement AST node.
// Takes ref (upvalueReference) which is the upvalue reference for the
// captured variable.
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

	c.function.emit(opSetUpvalue, currentRegister, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
	c.scopes.alloc.freeTemp(ref.kind, currentRegister)
	return varLocation{}, nil
}

// emitIncDec emits the actual increment or decrement instruction for a
// numeric register.
//
// Takes operatorToken (token.Token) which indicates whether this is an
// increment (token.INC) or decrement (token.DEC).
// Takes location (varLocation) which is the register location of the
// numeric value to modify.
//
// Returns the compiled location and any error encountered.
func (c *compiler) emitIncDec(_ context.Context, operatorToken token.Token, location varLocation) (varLocation, error) {
	switch location.kind {
	case registerInt:
		if operatorToken == token.INC {
			c.function.emit(opIncInt, location.register, 0, 0)
		} else {
			c.function.emit(opDecInt, location.register, 0, 0)
		}
	case registerFloat:
		oneIndex := c.function.addFloatConstant(1.0)
		tmpReg := c.scopes.alloc.allocTemp(registerFloat)
		c.function.emitWide(opLoadFloatConst, tmpReg, oneIndex)
		if operatorToken == token.INC {
			c.function.emit(opAddFloat, location.register, location.register, tmpReg)
		} else {
			c.function.emit(opSubFloat, location.register, location.register, tmpReg)
		}
		c.scopes.alloc.freeTemp(registerFloat, tmpReg)
	case registerUint:
		if operatorToken == token.INC {
			c.function.emit(opIncUint, location.register, 0, 0)
		} else {
			c.function.emit(opDecUint, location.register, 0, 0)
		}
	default:
		return varLocation{}, errors.New("inc/dec requires numeric variable")
	}
	return varLocation{}, nil
}

// multiReturnDeferredStore records a non-direct LHS target whose
// store instruction must be emitted after the call completes.
type multiReturnDeferredStore struct {
	// target specifies the LHS expression that needs a deferred store.
	target ast.Expr

	// srcLocation specifies the source register location holding the value to store.
	srcLocation varLocation
}

// callResultKinds determines the register kinds for each result of a
// call expression.
//
// Takes callExpr (*ast.CallExpr) which is the call expression to
// determine result kinds for.
//
// Returns the slice of register kinds for each result and any error
// encountered.
func (c *compiler) callResultKinds(_ context.Context, callExpr *ast.CallExpr) ([]registerKind, error) {
	if identifier, ok := callExpr.Fun.(*ast.Ident); ok {
		if funcIndex, found := c.funcTable[identifier.Name]; found {
			return c.rootFunction.functions[funcIndex].resultKinds, nil
		}
	}

	tv := c.info.Types[callExpr.Fun]
	signature, ok := tv.Type.Underlying().(*types.Signature)
	if !ok {
		return nil, fmt.Errorf("cannot determine result types for call: %T", callExpr.Fun)
	}
	var kinds []registerKind
	for v := range signature.Results().Variables() {
		kinds = append(kinds, kindForType(v.Type()))
	}
	return kinds, nil
}

// compileMultiReturnAssign compiles an assignment from a multi-return
// function call.
//
// Takes lhsList ([]ast.Expr) which is the left-hand side expressions to
// assign results to.
// Takes callExpr (*ast.CallExpr) which is the multi-return call
// expression to compile.
// Takes isDefine (bool) which indicates whether this is a := define or
// = assign.
//
// Returns the first result location and any error encountered.
func (c *compiler) compileMultiReturnAssign(ctx context.Context, lhsList []ast.Expr, callExpr *ast.CallExpr, isDefine bool) (varLocation, error) {
	resultKinds, err := c.callResultKinds(ctx, callExpr)
	if err != nil {
		return varLocation{}, err
	}

	returnLocs := make([]varLocation, len(lhsList))
	var deferred []multiReturnDeferredStore

	for i, leftHandSide := range lhsList {
		location, ds, err := c.resolveMultiReturnTarget(ctx, leftHandSide, resultKinds[i], isDefine)
		if err != nil {
			return varLocation{}, err
		}
		returnLocs[i] = location
		if ds != nil {
			deferred = append(deferred, *ds)
		}
	}

	if err := c.emitMultiReturnCall(ctx, callExpr, returnLocs); err != nil {
		return varLocation{}, err
	}

	for i, leftHandSide := range lhsList {
		if identifier, ok := leftHandSide.(*ast.Ident); ok && identifier.Name == blankIdentName {
			c.scopes.alloc.recycleRegister(returnLocs[i].kind, returnLocs[i].register)
		}
	}

	for _, ds := range deferred {
		if err := c.emitDeferredStore(ctx, ds); err != nil {
			return varLocation{}, err
		}
	}

	if len(returnLocs) > 0 {
		return returnLocs[0], nil
	}
	return varLocation{}, nil
}

// resolveMultiReturnTarget resolves a single LHS target for a
// multi-return assignment.
//
// Takes leftHandSide (ast.Expr) which is the left-hand side expression to
// resolve.
// Takes kind (registerKind) which is the expected register kind for the
// result.
// Takes isDefine (bool) which indicates whether this is a := define or
// = assign.
//
// Returns the target location, an optional deferred store, and any
// error.
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
		ds := &multiReturnDeferredStore{srcLocation: location, target: leftHandSide}
		return location, ds, nil

	default:
		return varLocation{}, nil, fmt.Errorf("unsupported assignment target: %T at %s", leftHandSide, c.positionString(leftHandSide.Pos()))
	}
}

// resolveMultiReturnIdent resolves an identifier LHS target for a
// multi-return assignment.
//
// Takes target (*ast.Ident) which is the identifier to resolve.
// Takes kind (registerKind) which is the expected register kind for the
// result.
// Takes isDefine (bool) which indicates whether this is a := define or
// = assign.
//
// Returns the target location, an optional deferred store, and any
// error.
func (c *compiler) resolveMultiReturnIdent(ctx context.Context,
	target *ast.Ident,
	kind registerKind,
	isDefine bool,
) (varLocation, *multiReturnDeferredStore, error) {
	if target.Name == blankIdentName {
		register := c.scopes.alloc.allocOrRecycle(kind, 0)
		return varLocation{register: register, kind: kind}, nil, nil
	}

	if isDefine {
		return c.resolveMultiReturnDefine(ctx, target, kind)
	}

	return c.resolveMultiReturnAssignIdent(ctx, target, kind)
}

// resolveMultiReturnDefine resolves a := target for a multi-return
// assignment.
//
// Takes target (*ast.Ident) which is the identifier to declare or look
// up.
// Takes kind (registerKind) which is the register kind for the new
// variable.
//
// Returns the target location, an optional deferred store, and any
// error.
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

// resolveMultiReturnAssignIdent resolves a plain = target for a
// multi-return assignment.
//
// Takes target (*ast.Ident) which is the identifier to look up.
// Takes kind (registerKind) which is the expected register kind for the
// result.
//
// Returns the target location, an optional deferred store, and any
// error.
func (c *compiler) resolveMultiReturnAssignIdent(_ context.Context,
	target *ast.Ident,
	kind registerKind,
) (varLocation, *multiReturnDeferredStore, error) {
	if _, ok := c.upvalueMap[target.Name]; ok {
		register := c.scopes.alloc.alloc(kind)
		location := varLocation{register: register, kind: kind}
		return location, &multiReturnDeferredStore{srcLocation: location, target: target}, nil
	}
	if _, ok := c.globalVars[target.Name]; ok {
		register := c.scopes.alloc.alloc(kind)
		location := varLocation{register: register, kind: kind}
		return location, &multiReturnDeferredStore{srcLocation: location, target: target}, nil
	}
	location, _ := c.scopes.lookupVar(target.Name)
	return location, nil, nil
}

// emitMultiReturnCall compiles and emits the call instruction for a
// multi-return call.
//
// Takes callExpr (*ast.CallExpr) which is the call expression to
// compile.
// Takes returnLocs ([]varLocation) which is the pre-allocated return
// locations for each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitMultiReturnCall(ctx context.Context, callExpr *ast.CallExpr, returnLocs []varLocation) error {
	callFun := c.unwrapGenericInstantiation(ctx, callExpr.Fun)
	switch fun := callFun.(type) {
	case *ast.Ident:
		if funcIndex, found := c.funcTable[fun.Name]; found {
			callee := c.rootFunction.functions[funcIndex]
			argLocs, err := c.compileCallArgs(ctx, callExpr, callee)
			if err != nil {
				return err
			}
			site := callSite{
				funcIndex: funcIndex,
				arguments: argLocs,
				returns:   returnLocs,
			}
			siteIndex := c.function.addCallSite(site)
			c.function.emitWide(opCall, 0, siteIndex)
			return nil
		}

		fnLocation, found := c.scopes.lookupVar(fun.Name)
		if !found {
			return fmt.Errorf("undefined function: %s at %s", fun.Name, c.positionString(fun.Pos()))
		}
		argLocs, err := c.compileArgExprs(ctx, callExpr)
		if err != nil {
			return err
		}
		site := callSite{
			isNative:       true,
			nativeRegister: fnLocation.register,
			arguments:      argLocs,
			returns:        returnLocs,
		}
		siteIndex := c.function.addCallSite(site)
		c.function.emitWide(opCallNative, 0, siteIndex)
		return nil

	case *ast.SelectorExpr:
		return c.compileMultiReturnSelectorCall(ctx, fun, callExpr, returnLocs)

	default:
		return fmt.Errorf("unsupported call target: %T at %s", callExpr.Fun, c.positionString(callExpr.Fun.Pos()))
	}
}

// compileArgExprs compiles all argument expressions for a call.
//
// Takes callExpr (*ast.CallExpr) which is the call expression whose
// arguments are compiled.
//
// Returns the compiled argument locations and any error encountered.
func (c *compiler) compileArgExprs(ctx context.Context, callExpr *ast.CallExpr) ([]varLocation, error) {
	argLocs := make([]varLocation, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return nil, err
		}
		argLocs[i] = location
	}
	return argLocs, nil
}

// compileMultiReturnSelectorCall dispatches a multi-return selector
// call to the appropriate code path.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method or function.
// Takes callExpr (*ast.CallExpr) which is the call expression to
// compile.
// Takes returnLocs ([]varLocation) which is the pre-allocated return
// locations for each result.
//
// Returns any error encountered during compilation.
func (c *compiler) compileMultiReturnSelectorCall(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpr *ast.CallExpr,
	returnLocs []varLocation,
) error {
	if funcIndex, ok := c.resolveMethodFunc(ctx, selectorExpression); ok {
		fieldPath := c.resolveEmbeddedFieldPath(ctx, selectorExpression)
		return c.emitMethodCallWithReturns(ctx, selectorExpression, callExpr, funcIndex, fieldPath, returnLocs)
	}

	if c.isInterfaceMethodCall(ctx, selectorExpression) {
		return c.emitDynamicMethodCallWithReturns(ctx, selectorExpression, callExpr, returnLocs)
	}

	if handled, err := c.emitLinkedCallWithReturns(ctx, selectorExpression, callExpr, returnLocs); handled || err != nil {
		return err
	}

	return c.emitNativeSelectorCallWithReturns(ctx, selectorExpression, callExpr, returnLocs)
}

// resolveMethodFunc resolves a selector expression to a compiled method
// function index.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve.
//
// Returns the function index and true if found, or zero and false
// otherwise.
func (c *compiler) resolveMethodFunc(ctx context.Context, selectorExpression *ast.SelectorExpr) (uint16, bool) {
	tableName, ok := c.resolveMethodTableName(ctx, selectorExpression)
	if !ok {
		return 0, false
	}
	funcIndex, found := c.funcTable[tableName]
	return funcIndex, found
}

// resolveEmbeddedFieldPath returns the field path for an embedded
// method receiver.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression to
// resolve the embedded path for.
//
// Returns the field index path excluding the final method index, or nil
// for direct methods.
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

// emitMethodCallWithReturns compiles an interpreted method call with
// pre-allocated return locations.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the method.
// Takes callExpr (*ast.CallExpr) which is the call expression to
// compile.
// Takes funcIndex (uint16) which is the function index of the compiled
// method.
// Takes fieldPath ([]int) which is the embedded field path to the
// receiver, or nil.
// Takes returnLocs ([]varLocation) which is the pre-allocated return
// locations for each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitMethodCallWithReturns(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpr *ast.CallExpr,
	funcIndex uint16,
	fieldPath []int,
	returnLocs []varLocation,
) error {
	recvLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return err
	}
	c.boxToGeneral(ctx, &recvLocation)

	for _, fieldIndex := range fieldPath {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opGetField, dest, recvLocation.register, safeconv.MustIntToUint8(fieldIndex))
		recvLocation = varLocation{register: dest, kind: registerGeneral}
	}

	argLocs := make([]varLocation, 0, 1+len(callExpr.Args))
	argLocs = append(argLocs, recvLocation)
	for _, arg := range callExpr.Args {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return err
		}
		argLocs = append(argLocs, location)
	}

	site := callSite{
		funcIndex: funcIndex,
		arguments: argLocs,
		returns:   returnLocs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCall, 0, siteIndex)
	return nil
}

// emitDynamicMethodCallWithReturns compiles an interface method call
// with pre-allocated return locations.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the interface method.
// Takes callExpr (*ast.CallExpr) which is the call expression to
// compile.
// Takes returnLocs ([]varLocation) which is the pre-allocated return
// locations for each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitDynamicMethodCallWithReturns(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpr *ast.CallExpr,
	returnLocs []varLocation,
) error {
	recvLocation, err := c.compileExpression(ctx, selectorExpression.X)
	if err != nil {
		return err
	}
	c.boxToGeneral(ctx, &recvLocation)

	argLocs := make([]varLocation, 0, 1+len(callExpr.Args))
	argLocs = append(argLocs, recvLocation)
	for _, arg := range callExpr.Args {
		location, err := c.compileExpression(ctx, arg)
		if err != nil {
			return err
		}
		argLocs = append(argLocs, location)
	}

	methodIndex := c.function.addStringConstant(selectorExpression.Sel.Name)

	site := callSite{
		arguments: argLocs,
		returns:   returnLocs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCallMethod, 0, siteIndex)
	c.function.emitExtension(methodIndex, 0)
	return nil
}

// emitNativeSelectorCallWithReturns compiles a native selector call
// with pre-allocated return locations.
//
// Takes selectorExpression (*ast.SelectorExpr) which is the selector expression
// identifying the native function or method.
// Takes callExpr (*ast.CallExpr) which is the call expression to
// compile.
// Takes returnLocs ([]varLocation) which is the pre-allocated return
// locations for each result.
//
// Returns any error encountered during compilation.
func (c *compiler) emitNativeSelectorCallWithReturns(ctx context.Context,
	selectorExpression *ast.SelectorExpr,
	callExpr *ast.CallExpr,
	returnLocs []varLocation,
) error {
	fnLocation, err := c.compileSelectorExpression(ctx, selectorExpression)
	if err != nil {
		return err
	}

	argLocs, err := c.compileArgExprs(ctx, callExpr)
	if err != nil {
		return err
	}

	site := callSite{
		isNative:       true,
		nativeRegister: fnLocation.register,
		arguments:      argLocs,
		returns:        returnLocs,
	}
	siteIndex := c.function.addCallSite(site)
	c.function.emitWide(opCallNative, 0, siteIndex)
	return nil
}

// emitDeferredStore emits a store instruction for a multi-return LHS
// target that could not be written directly.
//
// Takes ds (multiReturnDeferredStore) which is the deferred store record
// containing the target and source location.
//
// Returns any error encountered during the store emission.
func (c *compiler) emitDeferredStore(ctx context.Context, ds multiReturnDeferredStore) error {
	switch target := ds.target.(type) {
	case *ast.Ident:
		if ref, ok := c.upvalueMap[target.Name]; ok {
			c.function.emit(opSetUpvalue, ds.srcLocation.register, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
			return nil
		}
		if gv, ok := c.globalVars[target.Name]; ok {
			c.emitSetGlobal(ctx, gv, ds.srcLocation)
			return nil
		}
		return fmt.Errorf("deferred store: variable %s not found at %s", target.Name, c.positionString(target.Pos()))
	case *ast.IndexExpr:
		return c.compileIndexAssign(ctx, target, ds.srcLocation)
	case *ast.SelectorExpr:
		return c.compileSelectorAssign(ctx, target, ds.srcLocation)
	case *ast.StarExpr:
		return c.compileStarAssign(ctx, target, ds.srcLocation)
	default:
		return fmt.Errorf("unsupported deferred store target: %T at %s", ds.target, c.positionString(ds.target.Pos()))
	}
}

// declareCommaOkTargets declares or looks up the value and ok
// destination variables for comma-ok assignments.
//
// Takes valIdent (*ast.Ident) which is the identifier for the value
// target.
// Takes okIdent (*ast.Ident) which is the identifier for the ok boolean
// target.
// Takes valKind (registerKind) which is the register kind for the value
// variable.
// Takes blankValKind (registerKind) which is the register kind to use
// when the value target is blank.
// Takes isDefine (bool) which indicates whether this is a := define or
// = assign.
//
// Returns the value destination location and the ok destination
// location.
func (c *compiler) declareCommaOkTargets(ctx context.Context, valIdent, okIdent *ast.Ident, valKind registerKind, blankValKind registerKind, isDefine bool) (valDest varLocation, okDest varLocation) {
	if isDefine {
		valDest = c.declareCommaOkVal(ctx, valIdent, valKind, blankValKind)
		okDest = c.declareCommaOkBool(ctx, okIdent)
	} else {
		valDest = c.lookupCommaOkVal(ctx, valIdent, blankValKind)
		okDest = c.lookupCommaOkBool(ctx, okIdent)
	}

	return valDest, okDest
}

// declareCommaOkVal declares or resolves the value target for a
// comma-ok define.
//
// Takes valIdent (*ast.Ident) which is the identifier for the value
// target.
// Takes valKind (registerKind) which is the register kind for the value
// variable.
// Takes blankValKind (registerKind) which is the register kind to use
// when the target is blank.
//
// Returns the resolved value location.
func (c *compiler) declareCommaOkVal(_ context.Context, valIdent *ast.Ident, valKind registerKind, blankValKind registerKind) varLocation {
	if valIdent.Name != blankIdentName {
		typeObject := c.info.Defs[valIdent]
		if typeObject != nil {
			return c.scopes.declareVar(valIdent.Name, valKind)
		}
		location, _ := c.scopes.lookupVar(valIdent.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(blankValKind), kind: blankValKind}
}

// declareCommaOkBool declares or resolves the ok target for a comma-ok
// define.
//
// Takes okIdent (*ast.Ident) which is the identifier for the ok boolean
// target.
//
// Returns the resolved ok location.
func (c *compiler) declareCommaOkBool(_ context.Context, okIdent *ast.Ident) varLocation {
	if okIdent.Name != blankIdentName {
		typeObject := c.info.Defs[okIdent]
		if typeObject != nil {
			return c.scopes.declareVar(okIdent.Name, registerInt)
		}
		location, _ := c.scopes.lookupVar(okIdent.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(registerInt), kind: registerInt}
}

// lookupCommaOkVal looks up the value target for a comma-ok assign.
//
// Takes valIdent (*ast.Ident) which is the identifier for the value
// target.
// Takes blankValKind (registerKind) which is the register kind to use
// when the target is blank.
//
// Returns the resolved value location.
func (c *compiler) lookupCommaOkVal(_ context.Context, valIdent *ast.Ident, blankValKind registerKind) varLocation {
	if valIdent.Name != blankIdentName {
		location, _ := c.scopes.lookupVar(valIdent.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(blankValKind), kind: blankValKind}
}

// lookupCommaOkBool looks up the ok target for a comma-ok assign.
//
// Takes okIdent (*ast.Ident) which is the identifier for the ok boolean
// target.
//
// Returns the resolved ok location.
func (c *compiler) lookupCommaOkBool(_ context.Context, okIdent *ast.Ident) varLocation {
	if okIdent.Name != blankIdentName {
		location, _ := c.scopes.lookupVar(okIdent.Name)
		return location
	}
	return varLocation{register: c.scopes.alloc.allocTemp(registerInt), kind: registerInt}
}

// compileMapCommaOk compiles v, ok := m[k] or v, ok = m[k].
//
// Takes lhsList ([]ast.Expr) which is the left-hand side expressions
// for value and ok targets.
// Takes indexExpr (*ast.IndexExpr) which is the map index expression to
// compile.
// Takes isDefine (bool) which indicates whether this is a := define or
// = assign.
//
// Returns the value destination location and any error encountered.
func (c *compiler) compileMapCommaOk(ctx context.Context, lhsList []ast.Expr, indexExpr *ast.IndexExpr, isDefine bool) (varLocation, error) {
	mapLocation, err := c.compileExpression(ctx, indexExpr.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &mapLocation)

	keyLocation, err := c.compileExpression(ctx, indexExpr.Index)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &keyLocation)

	valIdent, ok := lhsList[0].(*ast.Ident)
	if !ok {
		return varLocation{}, errors.New("map comma-ok value target is not an identifier")
	}
	okIdent, ok := lhsList[1].(*ast.Ident)
	if !ok {
		return varLocation{}, errors.New("map comma-ok ok target is not an identifier")
	}

	valKind := registerKind(registerGeneral)
	if isDefine {
		mapType, ok := c.info.Types[indexExpr.X].Type.Underlying().(*types.Map)
		if !ok {
			return varLocation{}, errors.New("map comma-ok source is not a map type")
		}
		valKind = kindForType(mapType.Elem())
	}

	valDest, okDest := c.declareCommaOkTargets(ctx, valIdent, okIdent, valKind, registerGeneral, isDefine)

	genDest := c.scopes.alloc.alloc(registerGeneral)
	okReg := c.scopes.alloc.alloc(registerInt)

	c.function.emit(opMapIndexOk, genDest, mapLocation.register, keyLocation.register)
	c.function.emit(opExt, okReg, 0, 0)

	if valDest.kind == registerGeneral {
		c.emitMove(ctx, valDest, varLocation{register: genDest, kind: registerGeneral})
	} else if valDest.isSpilled {
		scratch := c.scopes.alloc.allocTemp(valDest.kind)
		c.function.emit(opUnpackInterface, scratch, genDest, uint8(valDest.kind))
		c.emitSpillStore(ctx, scratch, valDest.kind, valDest.spillSlot)
		c.scopes.alloc.freeTemp(valDest.kind, scratch)
	} else {
		c.function.emit(opUnpackInterface, valDest.register, genDest, uint8(valDest.kind))
	}

	c.emitMove(ctx, okDest, varLocation{register: okReg, kind: registerInt})

	return valDest, nil
}

// compileChanRecvCommaOk compiles v, ok := <-ch or v, ok = <-ch.
//
// Takes lhsList ([]ast.Expr) which is the left-hand side expressions
// for value and ok targets.
// Takes unaryExpr (*ast.UnaryExpr) which is the channel receive
// expression to compile.
// Takes isDefine (bool) which indicates whether this is a := define or
// = assign.
//
// Returns the value destination location and any error encountered.
func (c *compiler) compileChanRecvCommaOk(ctx context.Context, lhsList []ast.Expr, unaryExpr *ast.UnaryExpr, isDefine bool) (varLocation, error) {
	chLocation, err := c.compileExpression(ctx, unaryExpr.X)
	if err != nil {
		return varLocation{}, err
	}

	tv := c.info.Types[unaryExpr.X]
	chanType, ok := tv.Type.Underlying().(*types.Chan)
	if !ok {
		return varLocation{}, errors.New("channel receive comma-ok source is not a channel type")
	}
	elemType := chanType.Elem()
	valKind := kindForType(elemType)

	valIdent, ok := lhsList[0].(*ast.Ident)
	if !ok {
		return varLocation{}, errors.New("channel receive comma-ok value target is not an identifier")
	}
	okIdent, ok := lhsList[1].(*ast.Ident)
	if !ok {
		return varLocation{}, errors.New("channel receive comma-ok ok target is not an identifier")
	}

	valDest, okDest := c.declareCommaOkTargets(ctx, valIdent, okIdent, valKind, valKind, isDefine)

	okReg := c.scopes.alloc.alloc(registerInt)
	destReg := c.scopes.alloc.alloc(valKind)
	c.function.emit(opChanRecv, chLocation.register, okReg, 0)
	c.function.emit(opExt, destReg, uint8(valKind), 0)

	c.emitMove(ctx, valDest, varLocation{register: destReg, kind: valKind})
	c.emitMove(ctx, okDest, varLocation{register: okReg, kind: registerInt})

	return valDest, nil
}

// compileTypeAssertCommaOk compiles v, ok := x.(T) or v, ok = x.(T).
//
// Takes lhsList ([]ast.Expr) which is the left-hand side expressions
// for value and ok targets.
// Takes assertExpr (*ast.TypeAssertExpr) which is the type assertion
// expression to compile.
// Takes isDefine (bool) which indicates whether this is a := define or
// = assign.
//
// Returns the value destination location and any error encountered.
func (c *compiler) compileTypeAssertCommaOk(ctx context.Context, lhsList []ast.Expr, assertExpr *ast.TypeAssertExpr, isDefine bool) (varLocation, error) {
	srcLocation, err := c.compileExpression(ctx, assertExpr.X)
	if err != nil {
		return varLocation{}, err
	}
	c.boxToGeneral(ctx, &srcLocation)

	targetType := c.info.Types[assertExpr.Type].Type
	reflectType := c.typeToReflect(ctx, targetType)
	typeIndex := c.function.addTypeRef(reflectType)
	valKind := kindForType(targetType)

	valIdent, ok := lhsList[0].(*ast.Ident)
	if !ok {
		return varLocation{}, errors.New("type assert comma-ok value target is not an identifier")
	}
	okIdent, ok := lhsList[1].(*ast.Ident)
	if !ok {
		return varLocation{}, errors.New("type assert comma-ok ok target is not an identifier")
	}

	valDest, okDest := c.declareCommaOkTargets(ctx, valIdent, okIdent, valKind, registerGeneral, isDefine)

	genDest := c.scopes.alloc.alloc(registerGeneral)
	okReg := c.scopes.alloc.alloc(registerInt)

	c.function.emit(opTypeAssert, genDest, srcLocation.register, okReg)
	c.function.emitExtension(typeIndex, 0)

	if valDest.kind == registerGeneral {
		c.emitMove(ctx, valDest, varLocation{register: genDest, kind: registerGeneral})
	} else if valDest.isSpilled {
		scratch := c.scopes.alloc.allocTemp(valDest.kind)
		c.function.emit(opUnpackInterface, scratch, genDest, uint8(valDest.kind))
		c.emitSpillStore(ctx, scratch, valDest.kind, valDest.spillSlot)
		c.scopes.alloc.freeTemp(valDest.kind, scratch)
	} else {
		c.function.emit(opUnpackInterface, valDest.register, genDest, uint8(valDest.kind))
	}

	c.emitMove(ctx, okDest, varLocation{register: okReg, kind: registerInt})

	return valDest, nil
}
