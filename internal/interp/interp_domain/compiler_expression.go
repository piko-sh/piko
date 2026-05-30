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
	"reflect"

	"piko.sh/piko/wdk/safeconv"
)

const (
	// builtinPrint identifies the print builtin function.
	builtinPrint uint8 = 1

	// builtinPrintln identifies the println builtin function.
	builtinPrintln uint8 = 2

	// builtinClear identifies the clear builtin function.
	builtinClear uint8 = 3

	// selectDirectionReceive indicates a receive operation in a select case.
	selectDirectionReceive uint8 = 0

	// selectDirectionSend indicates a send operation in a select case.
	selectDirectionSend uint8 = 1

	// selectDirectionDefault indicates the default case in a select statement.
	selectDirectionDefault uint8 = 2
)

// compileExpression compiles expression and returns the register location holding the
// result. The compiler caps recursion at expressionDepthLimit (defaultMaxExpressionDepth,
// 1024) to defend against pathologically deep input.
//
// Takes expression which is the AST expression to compile.
//
// Returns the register location holding the result, or an error when the expression
// nesting limit is exceeded or the form is unsupported.
func (c *compiler) compileExpression(ctx context.Context, expression ast.Expr) (varLocation, error) {
	location, err := c.compileExpressionDispatch(ctx, expression)
	if err != nil {
		return location, err
	}
	if location.sourceType == nil && location.kind != registerGeneral {
		if tv, ok := c.info.Types[expression]; ok && tv.Type != nil {
			location.sourceType = c.exactReflectTypeForBoxing(c.substitutedType(tv.Type))
		}
	}
	return location, nil
}

// compileExpressionDispatch is the body of compileExpression: it dispatches on the AST
// node kind. compileExpression wraps it to stamp the result varLocation with the
// expression's exact source-level reflect.Type (used by emitBoxToGeneral to pick
// opPackTyped over opPackInterface).
//
// Takes expression (ast.Expr) which is the AST node to compile.
//
// Returns the result location and any compilation error.
func (c *compiler) compileExpressionDispatch(ctx context.Context, expression ast.Expr) (varLocation, error) {
	limit := c.expressionDepthLimit()
	if c.expressionDepth >= limit {
		return varLocation{}, fmt.Errorf("%w: expression nesting exceeded %d at %s", errCompileDepthLimit, limit, c.positionString(expression.Pos()))
	}
	c.expressionDepth++
	defer func() { c.expressionDepth-- }()

	c.setDebugPosition(ctx, expression.Pos())

	if tv, ok := c.info.Types[expression]; ok && tv.Value != nil {
		return c.compileConstant(ctx, tv)
	}

	switch e := expression.(type) {
	case *ast.BasicLit:
		return c.compileBasicLit(ctx, e)

	case *ast.Ident:
		return c.compileIdent(ctx, e)

	case *ast.BinaryExpr:
		return c.compileBinaryExpression(ctx, e)

	case *ast.UnaryExpr:
		return c.compileUnaryExpression(ctx, e)

	case *ast.ParenExpr:
		return c.compileExpression(ctx, e.X)

	case *ast.CallExpr:
		return c.compileCallExpression(ctx, e)

	case *ast.CompositeLit:
		return c.compileCompositeLit(ctx, e)

	case *ast.IndexExpr:
		return c.compileIndexExpression(ctx, e)

	case *ast.FuncLit:
		return c.compileFuncLit(ctx, e)

	case *ast.SliceExpr:
		return c.compileSliceExpression(ctx, e)

	case *ast.SelectorExpr:
		return c.compileSelectorExpression(ctx, e)

	case *ast.StarExpr:
		return c.compileStarExpression(ctx, e)

	case *ast.TypeAssertExpr:
		return c.compileTypeAssertExpression(ctx, e)

	default:
		return varLocation{}, fmt.Errorf("unsupported expression type: %T at %s", expression, c.positionString(expression.Pos()))
	}
}

var (
	// exactBasicReflectTypes maps each go/types basic kind to the exact Go reflect.Type.
	//
	// Deliberately does NOT collapse int->int64 etc. the way piko's register-bank type
	// converter does. emitBoxToGeneral consults the table so a boxed `int` survives as
	// reflect.Type `int`, distinct from a boxed `int64`.
	//
	//nolint:exhaustive // some BasicKinds have no scalar reflect.Type, so omit them
	exactBasicReflectTypes = map[types.BasicKind]reflect.Type{
		types.Bool:          reflect.TypeFor[bool](),
		types.Int:           reflect.TypeFor[int](),
		types.Int8:          reflect.TypeFor[int8](),
		types.Int16:         reflect.TypeFor[int16](),
		types.Int32:         reflect.TypeFor[int32](),
		types.Int64:         reflect.TypeFor[int64](),
		types.Uint:          reflect.TypeFor[uint](),
		types.Uint8:         reflect.TypeFor[uint8](),
		types.Uint16:        reflect.TypeFor[uint16](),
		types.Uint32:        reflect.TypeFor[uint32](),
		types.Uint64:        reflect.TypeFor[uint64](),
		types.Uintptr:       reflect.TypeFor[uintptr](),
		types.Float32:       reflect.TypeFor[float32](),
		types.Float64:       reflect.TypeFor[float64](),
		types.Complex64:     reflect.TypeFor[complex64](),
		types.Complex128:    reflect.TypeFor[complex128](),
		types.String:        reflect.TypeFor[string](),
		types.UntypedBool:   reflect.TypeFor[bool](),
		types.UntypedInt:    reflect.TypeFor[int](),
		types.UntypedRune:   reflect.TypeFor[int32](),
		types.UntypedFloat:  reflect.TypeFor[float64](),
		types.UntypedString: reflect.TypeFor[string](),
	}
)

// exactReflectTypeForBoxing returns the boxed scalar reflect.Type.
//
// Preserves interface type identity. Returns nil for non-scalar types (slices, maps,
// structs, pointers, interfaces), which box correctly via reflect.ValueOf already, and
// for types whose exact identity cannot be recovered, in which case the caller falls back
// to opPackInterface boxing.
//
// Named types resolve through their underlying basic kind: `type Celsius float64` returns
// reflect float64 (the width is preserved; the source-level name is recovered separately
// by the reflect.TypeOf interception path). Defined types over non-basic underlyings
// return nil.
//
// Takes t (types.Type) which is the go/types static type of the value.
//
// Returns the exact reflect.Type, or nil when legacy boxing applies.
func (*compiler) exactReflectTypeForBoxing(t types.Type) reflect.Type {
	if t == nil {
		return nil
	}
	underlying := t.Underlying()
	basic, ok := underlying.(*types.Basic)
	if !ok {
		return nil
	}
	return exactBasicReflectTypes[basic.Kind()]
}

// typeAssertReflectType returns the reflect.Type a type assertion should match against.
// For scalar targets it returns the exact reflect.Type (int, int32, float32, ...) so the
// assertion compares like-for-like against opPackTyped-boxed values; for everything else
// it falls back to the standard collapsing converter.
//
// This must agree with exactReflectTypeForBoxing: a value boxed as `int32` can only be
// asserted to `.(int32)` if the assertion target also resolves to reflect `int32` and not
// the collapsed `int64`.
//
// Takes ctx (context.Context) for the fallback converter.
// Takes targetType (types.Type) which is the assertion's target type.
//
// Returns the reflect.Type to match against; nil only when targetType itself is nil.
func (c *compiler) typeAssertReflectType(ctx context.Context, targetType types.Type) reflect.Type {
	if exact := c.exactReflectTypeForBoxing(targetType); exact != nil {
		return exact
	}
	return c.typeToReflect(ctx, targetType)
}

// compileConstant emits a load for a compile-time constant value already folded by
// go/types. For interface-typed constants the scalar value is boxed through
// opPackInterface into the general bank.
//
// Takes tv which is the folded type-and-value record from go/types.
//
// Returns the register location holding the loaded constant, or an error when the
// constant cannot be loaded into the target bank.
func (c *compiler) compileConstant(ctx context.Context, tv types.TypeAndValue) (varLocation, error) {
	kind := c.kindFor(tv.Type)
	if kind != registerGeneral {
		return c.emitScalarConstant(ctx, tv.Value, kind)
	}

	scalarKind := scalarKindForConstant(tv.Value)
	scalarLocation, err := c.emitScalarConstant(ctx, tv.Value, scalarKind)
	if err != nil {
		return varLocation{}, err
	}
	scalarLocation.sourceType = c.exactReflectTypeForBoxing(tv.Type)
	generalRegister := c.scopes.alloc.alloc(registerGeneral)
	if c.emitTypedBox(generalRegister, scalarLocation) {
		return varLocation{register: generalRegister, kind: registerGeneral}, nil
	}
	c.function.emit(opPackInterface, generalRegister, scalarLocation.register, uint8(scalarLocation.kind))
	return varLocation{register: generalRegister, kind: registerGeneral}, nil
}

// emitScalarConstant adds value to the matching typed constant pool and emits the wide
// load instruction into a freshly allocated register of kind.
//
// Takes value which is the folded constant.Value to load.
// Takes kind which is the target register bank.
//
// Returns the register location holding the loaded constant, or an error when value
// cannot be represented in the target bank.
func (c *compiler) emitScalarConstant(_ context.Context, value constant.Value, kind registerKind) (varLocation, error) {
	switch kind {
	case registerBool:
		return c.emitBoolScalarConstant(value)
	case registerInt:
		return c.emitIntScalarConstant(value)
	case registerUint:
		return c.emitUintScalarConstant(value)
	case registerFloat:
		return c.emitFloatScalarConstant(value)
	case registerString:
		return c.emitStringScalarConstant(value)
	case registerComplex:
		return c.emitComplexScalarConstant(value)
	default:
		return varLocation{}, fmt.Errorf("unsupported constant kind %v for register bank (value: %v)", kind, value)
	}
}

// emitBoolScalarConstant loads value into a freshly allocated bool register.
//
// Takes value (constant.Value) which is the folded bool constant to load.
//
// Returns varLocation which holds the loaded register's metadata.
// Returns error when the bool constant pool is exhausted.
func (c *compiler) emitBoolScalarConstant(value constant.Value) (varLocation, error) {
	register := c.scopes.alloc.alloc(registerBool)
	index, err := c.function.addBoolConstant(constant.BoolVal(value))
	if err != nil {
		return varLocation{}, err
	}
	c.function.emitWide(opLoadBoolConst, register, index)
	return varLocation{register: register, kind: registerBool}, nil
}

// emitIntScalarConstant loads value into a freshly allocated int register.
//
// Takes value (constant.Value) which is the folded integer constant to load.
//
// Returns varLocation which holds the loaded register's metadata.
// Returns error when the value cannot be represented as int64 or the constant pool is
// exhausted.
func (c *compiler) emitIntScalarConstant(value constant.Value) (varLocation, error) {
	v, ok := constant.Int64Val(value)
	if !ok {
		return varLocation{}, fmt.Errorf("cannot convert constant to int64: %v", value)
	}
	index, err := c.function.addIntConstant(v)
	if err != nil {
		return varLocation{}, err
	}
	register := c.scopes.alloc.alloc(registerInt)
	c.function.emitWide(opLoadIntConst, register, index)
	return varLocation{register: register, kind: registerInt}, nil
}

// emitUintScalarConstant loads value into a freshly allocated uint register.
//
// Takes value (constant.Value) which is the folded unsigned constant to load.
//
// Returns varLocation which holds the loaded register's metadata.
// Returns error when the value cannot be represented as uint64 or the constant pool is
// exhausted.
func (c *compiler) emitUintScalarConstant(value constant.Value) (varLocation, error) {
	u, ok := constant.Uint64Val(value)
	if !ok {
		v, valueOk := constant.Int64Val(value)
		if valueOk {
			u = safeconv.Int64ToUint64Reinterpret(v)
			ok = true
		}
	}
	if !ok {
		return varLocation{}, fmt.Errorf("cannot convert constant to uint64: %v", value)
	}
	index, err := c.function.addUintConstant(u)
	if err != nil {
		return varLocation{}, err
	}
	register := c.scopes.alloc.alloc(registerUint)
	c.function.emitWide(opLoadUintConst, register, index)
	return varLocation{register: register, kind: registerUint}, nil
}

// emitFloatScalarConstant loads value into a freshly allocated float register.
//
// Takes value (constant.Value) which is the folded floating-point constant.
//
// Returns varLocation which holds the loaded register's metadata.
// Returns error when the float constant pool is exhausted.
func (c *compiler) emitFloatScalarConstant(value constant.Value) (varLocation, error) {
	v, _ := constant.Float64Val(value)
	index, err := c.function.addFloatConstant(v)
	if err != nil {
		return varLocation{}, err
	}
	register := c.scopes.alloc.alloc(registerFloat)
	c.function.emitWide(opLoadFloatConst, register, index)
	return varLocation{register: register, kind: registerFloat}, nil
}

// emitStringScalarConstant loads value into a freshly allocated string register.
//
// Takes value (constant.Value) which is the folded string constant.
//
// Returns varLocation which holds the loaded register's metadata.
// Returns error when the string constant pool is exhausted.
func (c *compiler) emitStringScalarConstant(value constant.Value) (varLocation, error) {
	v := constant.StringVal(value)
	index, err := c.function.addStringConstant(v)
	if err != nil {
		return varLocation{}, err
	}
	register := c.scopes.alloc.alloc(registerString)
	c.function.emitWide(opLoadStringConst, register, index)
	return varLocation{register: register, kind: registerString}, nil
}

// emitComplexScalarConstant loads value into a freshly allocated complex register.
//
// Takes value (constant.Value) which is the folded complex constant.
//
// Returns varLocation which holds the loaded register's metadata.
// Returns error when the complex constant pool is exhausted.
func (c *compiler) emitComplexScalarConstant(value constant.Value) (varLocation, error) {
	realPart, _ := constant.Float64Val(constant.Real(value))
	imaginaryPart, _ := constant.Float64Val(constant.Imag(value))
	index, err := c.function.addComplexConstant(complex(realPart, imaginaryPart))
	if err != nil {
		return varLocation{}, err
	}
	register := c.scopes.alloc.alloc(registerComplex)
	c.function.emitWide(opLoadComplexConst, register, index)
	return varLocation{register: register, kind: registerComplex}, nil
}

// compileBasicLit compiles a basic literal (number, string, etc.). Reached only when
// compileExpression's pre-check did not find a folded constant for the literal in
// c.info.Types.
//
// Takes lit which is the basic literal AST node to compile.
//
// Returns the register location holding the literal, or an error when no type information
// is available for the literal.
func (c *compiler) compileBasicLit(ctx context.Context, lit *ast.BasicLit) (varLocation, error) {
	tv, ok := c.info.Types[lit]
	if ok && tv.Value != nil {
		return c.compileConstant(ctx, tv)
	}
	return varLocation{}, fmt.Errorf("basic literal without type info: %s", lit.Value)
}

// compileIdent compiles an identifier reference, resolving the predeclared literals
// (true, false, nil), local scope variables (including spilled and indirect locations),
// upvalues, globals, and top-level functions in that order.
//
// Takes identifier which is the identifier AST node to resolve and load.
//
// Returns the register location holding the identifier's value, or an error when the
// identifier is not defined in any scope.
func (c *compiler) compileIdent(ctx context.Context, identifier *ast.Ident) (varLocation, error) {
	if identifier.Name == identTrue {
		register := c.scopes.alloc.alloc(registerBool)
		index, err := c.function.addBoolConstant(true)
		if err != nil {
			return varLocation{}, err
		}
		c.function.emitWide(opLoadBoolConst, register, index)
		return varLocation{register: register, kind: registerBool}, nil
	}
	if identifier.Name == identFalse {
		register := c.scopes.alloc.alloc(registerBool)
		index, err := c.function.addBoolConstant(false)
		if err != nil {
			return varLocation{}, err
		}
		c.function.emitWide(opLoadBoolConst, register, index)
		return varLocation{register: register, kind: registerBool}, nil
	}
	if identifier.Name == identNil {
		register := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opDrillTier1, uint8(subOpDrillTier2), uint8(subOpTier2LoadNil), register)
		return varLocation{register: register, kind: registerGeneral}, nil
	}

	location, found := c.scopes.lookupVar(identifier.Name)
	if found {
		if location.isIndirect {
			return c.emitIndirectRead(ctx, location)
		}
		if location.isSpilled {
			return c.materialise(ctx, location), nil
		}
		return location, nil
	}

	if ref, ok := c.upvalueMap[identifier.Name]; ok {
		dest := c.scopes.alloc.alloc(ref.kind)
		c.function.emit(opGetUpvalue, dest, safeconv.MustIntToUint8(ref.index), uint8(ref.kind))
		return varLocation{register: dest, kind: ref.kind}, nil
	}

	if gv, ok := c.globalVariables[identifier.Name]; ok {
		return c.emitGetGlobal(ctx, gv), nil
	}

	if funcIndex, found := c.funcTable[identifier.Name]; found {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emitWide(opMakeClosure, dest, funcIndex)
		return varLocation{register: dest, kind: registerGeneral}, nil
	}

	return varLocation{}, fmt.Errorf("undefined: %s at %s", identifier.Name, c.positionString(identifier.Pos()))
}

// compileBinaryExpression compiles a binary expression, routing logical && and || through
// short-circuit emitters and matching the interface-vs-nil comparison fast path before
// the generic emit path. Applies opTruncateNarrow when the static type is a narrow
// integer.
//
// Takes expression which is the binary expression AST node to compile.
//
// Returns the register location holding the result, or an error when either operand or
// the operator cannot be compiled.
func (c *compiler) compileBinaryExpression(ctx context.Context, expression *ast.BinaryExpr) (varLocation, error) {
	if expression.Op == token.LAND {
		return c.compileShortCircuitAnd(ctx, expression)
	}
	if expression.Op == token.LOR {
		return c.compileShortCircuitOr(ctx, expression)
	}

	if location, ok, err := c.tryCompileInterfaceNilComparison(ctx, expression); ok || err != nil {
		return location, err
	}

	left, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	right, err := c.compileExpression(ctx, expression.Y)
	if err != nil {
		return varLocation{}, err
	}

	result, err := c.emitBinaryOp(ctx, expression.Op, left, right)
	if err != nil {
		return result, err
	}
	if t := c.staticTypeOf(expression); t != nil {
		c.emitNarrowIntegerTruncation(result, t)
	}
	return result, nil
}

// tryCompileInterfaceNilComparison emits opEqInterfaceNil or opNeInterfaceNil when
// expression is an interface-vs-nil equality comparison, preserving Go's distinction
// between a nil interface and an interface holding a typed nil. Returns applied=false
// when the pattern does not match.
//
// Takes expression which is the binary expression to inspect for the pattern.
//
// Returns the register location holding the comparison result, a flag set when the
// pattern matched, and an error when compiling the interface side failed.
func (c *compiler) tryCompileInterfaceNilComparison(ctx context.Context, expression *ast.BinaryExpr) (varLocation, bool, error) {
	if expression.Op != token.EQL && expression.Op != token.NEQ {
		return varLocation{}, false, nil
	}
	interfaceSide, nilSide, ok := c.classifyInterfaceNilOperands(expression.X, expression.Y)
	if !ok {
		return varLocation{}, false, nil
	}
	_ = nilSide
	location, err := c.compileExpression(ctx, interfaceSide)
	if err != nil {
		return varLocation{}, false, err
	}
	c.boxToGeneral(ctx, &location)
	dest := c.scopes.alloc.alloc(registerInt)
	op := opEqInterfaceNil
	if expression.Op == token.NEQ {
		op = opNeInterfaceNil
	}
	c.function.emit(op, dest, location.register, 0)
	return varLocation{register: dest, kind: registerInt}, true, nil
}

// classifyInterfaceNilOperands returns the interface-typed and the nil-literal operand
// from a binary equality expression, with matched set when either side ordering produces
// the pattern.
//
// Takes left which is the left-hand operand of the equality expression.
// Takes right which is the right-hand operand of the equality expression.
//
// Returns the interface-typed operand, the nil-literal operand, and a flag set when one
// side is interface-typed and the other is nil.
func (c *compiler) classifyInterfaceNilOperands(left, right ast.Expr) (interfaceExpr ast.Expr, nilExpr ast.Expr, matched bool) {
	if c.isNilLiteral(right) && c.expressionIsInterfaceTyped(left) {
		return left, right, true
	}
	if c.isNilLiteral(left) && c.expressionIsInterfaceTyped(right) {
		return right, left, true
	}
	return nil, nil, false
}

// expressionIsInterfaceTyped reports whether expression's static type underlies as an
// interface (empty or non-empty).
//
// Takes expression which is the AST expression to inspect.
//
// Returns true when the underlying static type is an interface.
func (c *compiler) expressionIsInterfaceTyped(expression ast.Expr) bool {
	staticType := c.staticTypeOf(expression)
	if staticType == nil {
		return false
	}
	_, ok := staticType.Underlying().(*types.Interface)
	return ok
}

// isNilLiteral reports whether expression resolves to the universe-scope nil identifier
// (as confirmed by go/types when type info is available).
//
// Takes expression which is the AST expression to test.
//
// Returns true when expression is the predeclared nil identifier.
func (c *compiler) isNilLiteral(expression ast.Expr) bool {
	identifier, ok := expression.(*ast.Ident)
	if !ok || identifier.Name != "nil" {
		return false
	}
	if c.info == nil {
		return true
	}
	if object := c.info.Uses[identifier]; object != nil {
		_, isNil := object.(*types.Nil)
		return isNil
	}
	return true
}

// compileTypedNilOrExpression emits a typed-zero load for typed `nil` literals.
//
// Fires when expression is the predeclared `nil` identifier and expectedType is a
// concrete nil-bearing type (pointer, slice, map, chan, signature/func). Returns
// (location, true, nil) when the typed-nil path fired so the caller skips the regular
// compileExpression path; otherwise returns (zero, false, nil) to signal a fall-through.
//
// Without this, `nil` always compiles to opLoadNil which stores reflect.Value{} (truly
// zero, no type tag). For typed destinations like *Holder that reach interface-conversion
// or dynamic method dispatch, the lost type tag panics handleCallMethod's receiver.Type()
// and breaks `s == nil` semantics for typed-nil interfaces. Interface destinations
// intentionally keep the untyped nil so a == nil stays true.
//
// Takes ctx (context.Context) which drives reflect-type synthesis.
// Takes expression (ast.Expr) which is the source expression about to be compiled.
// Takes expectedType (types.Type) which is the static type at the consumption site
// (declared variable type, return slot type, parameter type, etc).
//
// Returns a location holding the typed-zero reflect.Value when the fast path fired, true
// to indicate the caller should skip compileExpression, and any compilation error from
// the constant pool addition.
func (c *compiler) compileTypedNilOrExpression(ctx context.Context, expression ast.Expr, expectedType types.Type) (varLocation, bool, error) {
	if expectedType == nil {
		return varLocation{}, false, nil
	}
	identifier, ok := expression.(*ast.Ident)
	if !ok || identifier.Name != identNil {
		return varLocation{}, false, nil
	}
	if c.info != nil {
		if object := c.info.Uses[identifier]; object != nil {
			if _, isNil := object.(*types.Nil); !isNil {
				return varLocation{}, false, nil
			}
		}
	}
	underlying := expectedType.Underlying()
	if underlying == nil {
		return varLocation{}, false, nil
	}
	switch underlying.(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan, *types.Signature:
	default:
		return varLocation{}, false, nil
	}
	reflectType := c.typeToReflect(ctx, expectedType)
	if reflectType == nil {
		return varLocation{}, false, nil
	}
	constIndex, err := c.function.addGeneralConstant(reflect.Zero(reflectType), generalConstantDescriptor{
		kind:     generalConstantCompositeZero,
		typeDesc: reflectTypeToDescriptor(reflectType),
	})
	if err != nil {
		c.recordStickyError(err)
		return varLocation{}, false, err
	}
	register := c.scopes.alloc.alloc(registerGeneral)
	c.function.emitWide(opLoadGeneralConst, register, constIndex)
	return varLocation{register: register, kind: registerGeneral}, true, nil
}

// emitNarrowIntegerTruncation emits opTruncateNarrow when staticType is a narrow integer
// (int8/int16/int32 or uint8/uint16/uint32) and location holds the value in the int or
// uint bank, preserving Go's modular wrap semantics across 64-bit storage.
//
// Takes location which is the register holding the value to potentially truncate.
// Takes staticType which is the static Go type that determines the bit width.
func (c *compiler) emitNarrowIntegerTruncation(location varLocation, staticType types.Type) {
	bitWidth := narrowIntegerBitWidth(staticType)
	if bitWidth == 0 {
		return
	}
	if location.kind != registerInt && location.kind != registerUint {
		return
	}
	c.function.emit(opTruncateNarrow, location.register, bitWidth, uint8(location.kind))
}

// compileShortCircuitAnd compiles a logical AND with short-circuit evaluation, skipping
// the right operand when the left is false. The result is an int register holding 0 or 1.
//
// Takes expression which is the logical AND binary expression to compile.
//
// Returns the int register location holding 0 or 1, or an error when either operand fails
// to compile.
func (c *compiler) compileShortCircuitAnd(ctx context.Context, expression *ast.BinaryExpr) (varLocation, error) {
	return c.compileShortCircuit(ctx, expression, opJumpIfFalse)
}

// compileShortCircuitOr compiles a logical OR with short-circuit evaluation, skipping the
// right operand when the left is true. The result is an int register holding 0 or 1.
//
// Takes expression which is the logical OR binary expression to compile.
//
// Returns the int register location holding 0 or 1, or an error when either operand fails
// to compile.
func (c *compiler) compileShortCircuitOr(ctx context.Context, expression *ast.BinaryExpr) (varLocation, error) {
	return c.compileShortCircuit(ctx, expression, opJumpIfTrue)
}

// compileShortCircuit compiles a boolean short-circuit expression using skipOp
// (opJumpIfFalse for AND, opJumpIfTrue for OR) to bypass the right operand. The result is
// an int register holding 0 or 1.
//
// Takes expression which is the binary expression to compile.
// Takes skipOp which is the conditional branch opcode used to skip the right operand.
//
// Returns the int register location holding 0 or 1, or an error when either operand fails
// to compile.
func (c *compiler) compileShortCircuit(ctx context.Context, expression *ast.BinaryExpr, skipOp opcode) (varLocation, error) {
	left, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}
	left = c.ensureIntForBranch(ctx, left)
	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opDrillTier1, uint8(subOpMoveInt), dest, left.register)
	jumpToEnd := c.function.emitJump(skipOp, dest)
	right, err := c.compileExpression(ctx, expression.Y)
	if err != nil {
		return varLocation{}, err
	}
	right = c.ensureIntForBranch(ctx, right)
	c.function.emit(opDrillTier1, uint8(subOpMoveInt), dest, right.register)
	c.function.patchJump(jumpToEnd)
	return varLocation{register: dest, kind: registerInt}, nil
}

// emitBinaryOp emits the typed instruction for binary operator op, routing arithmetic,
// comparison, and bitwise/shift tokens to the matching specialised emitter.
//
// Takes op which is the binary operator token.
// Takes left which is the left operand register location.
// Takes right which is the right operand register location.
//
// Returns the register location holding the result, or an error when the operator is
// unsupported for the operand banks.
func (c *compiler) emitBinaryOp(ctx context.Context, op token.Token, left, right varLocation) (varLocation, error) {
	switch op {
	case token.ADD:
		return c.emitArithOp(ctx, opAddInt, opAddFloat, opConcatString, opAdd, left, right)
	case token.SUB:
		return c.emitArithOp(ctx, opSubInt, opSubFloat, 0, opSub, left, right)
	case token.MUL:
		return c.emitArithOp(ctx, opMulInt, opMulFloat, 0, opMul, left, right)
	case token.QUO:
		return c.emitArithOp(ctx, opDivInt, opDivFloat, 0, opDiv, left, right)
	case token.REM:
		return c.emitArithOp(ctx, opRemInt, 0, 0, opRem, left, right)

	case token.EQL:
		return c.emitCompareOp(ctx, opEqInt, opEqFloat, opEqString, opEqGeneral, left, right)
	case token.NEQ:
		return c.emitCompareOp(ctx, opNeInt, opNeFloat, opNeString, opNeGeneral, left, right)
	case token.LSS:
		return c.emitCompareOp(ctx, opLtInt, opLtFloat, opLtString, opLtGeneral, left, right)
	case token.LEQ:
		return c.emitCompareOp(ctx, opLeInt, opLeFloat, opLeString, opLeGeneral, left, right)
	case token.GTR:
		return c.emitCompareOp(ctx, opGtInt, opGtFloat, opGtString, opGtGeneral, left, right)
	case token.GEQ:
		return c.emitCompareOp(ctx, opGeInt, opGeFloat, opGeString, opGeGeneral, left, right)

	case token.AND:
		return c.emitIntOnlyOp(ctx, opBitAnd, left, right)
	case token.OR:
		return c.emitIntOnlyOp(ctx, opBitOr, left, right)
	case token.XOR:
		return c.emitIntOnlyOp(ctx, opBitXor, left, right)
	case token.AND_NOT:
		return c.emitIntOnlyOp(ctx, opBitAndNot, left, right)
	case token.SHL:
		return c.emitIntOnlyOp(ctx, opShiftLeft, left, right)
	case token.SHR:
		return c.emitIntOnlyOp(ctx, opShiftRight, left, right)

	default:
		return varLocation{}, fmt.Errorf("unsupported binary operator: %s (left=%v, right=%v)", op, left.kind, right.kind)
	}
}

// emitArithOp dispatches an arithmetic operation to the bank-specific emitter based on
// left.kind, using the supplied opcodes for int, float, string, uint, complex, and
// general operands. A zero opcode signals the operation is not supported for that bank.
//
// Takes intOp which is the opcode for int operands (also the seed for uint/complex
// mappings).
// Takes floatOp which is the opcode for float operands, or zero when unsupported.
// Takes strOp which is the opcode for string operands, or zero when unsupported.
// Takes genOp which is the opcode for general (interface) operands, or zero when
// unsupported.
// Takes left which is the left operand register location.
// Takes right which is the right operand register location.
//
// Returns the register location holding the result, or an error when the bank does not
// support the operation.
func (c *compiler) emitArithOp(ctx context.Context, intOp, floatOp, strOp, genOp opcode, left, right varLocation) (varLocation, error) {
	switch left.kind {
	case registerInt:
		return c.emitTypedArith(ctx, intOp, registerInt, left, right)
	case registerFloat:
		if floatOp == 0 {
			return varLocation{}, ErrCompileArithFloatUnsupported
		}
		return c.emitTypedArith(ctx, floatOp, registerFloat, left, right)
	case registerString:
		if strOp == 0 {
			return varLocation{}, ErrCompileArithStringUnsupported
		}
		return c.emitTypedArith(ctx, strOp, registerString, left, right)
	case registerUint:
		return c.emitArithUint(ctx, intOp, left, right)
	case registerComplex:
		return c.emitArithComplex(ctx, intOp, left, right)
	default:
		if genOp == 0 {
			return varLocation{}, ErrCompileArithGeneralUnsupported
		}
		return c.emitTypedArith(ctx, genOp, registerGeneral, left, right)
	}
}

// emitTypedArith allocates a destination register of kind and emits op through emitTyped
// so the operand-shape descriptor inserts a bank coercion when an operand arrives in the
// wrong register bank.
//
// Takes op which is the opcode to emit.
// Takes kind which is the destination register bank.
// Takes left which is the left operand register location.
// Takes right which is the right operand register location.
//
// Returns the register location holding the result, and a nil error on success.
func (c *compiler) emitTypedArith(ctx context.Context, op opcode, kind registerKind, left, right varLocation) (varLocation, error) {
	dest := c.scopes.alloc.alloc(kind)
	destinationLocation := varLocation{register: dest, kind: kind}
	c.emitTyped(ctx, op, destinationLocation, left, right)
	return destinationLocation, nil
}

// emitArithUint maps intOp to its uint counterpart and emits it via emitTypedArith.
//
// Takes intOp which is the int-bank opcode whose uint counterpart will be emitted.
// Takes left which is the left operand register location.
// Takes right which is the right operand register location.
//
// Returns the register location holding the result, or an error when no uint counterpart
// exists.
func (c *compiler) emitArithUint(ctx context.Context, intOp opcode, left, right varLocation) (varLocation, error) {
	uintOp, ok := intToUintArithOp(intOp)
	if !ok {
		return varLocation{}, ErrCompileArithUintUnsupported
	}
	return c.emitTypedArith(ctx, uintOp, registerUint, left, right)
}

// emitArithComplex maps intOp to its complex counterpart and emits it via emitTypedArith.
//
// Takes intOp which is the int-bank opcode whose complex counterpart will be emitted.
// Takes left which is the left operand register location.
// Takes right which is the right operand register location.
//
// Returns the register location holding the result, or an error when no complex
// counterpart exists.
func (c *compiler) emitArithComplex(ctx context.Context, intOp opcode, left, right varLocation) (varLocation, error) {
	complexOp, ok := intToComplexArithOp(intOp)
	if !ok {
		return varLocation{}, ErrCompileArithComplexUnsupported
	}
	return c.emitTypedArith(ctx, complexOp, registerComplex, left, right)
}

// emitCompareOp emits the type-specialised comparison for left and right and returns the
// int register (0 or 1) holding the result. Bool operands are converted to int; uint and
// complex have their own bank-specific paths.
//
// Takes intOp which is the opcode for int operands (also the seed for uint mappings).
// Takes floatOp which is the opcode for float operands, or zero when unsupported.
// Takes strOp which is the opcode for string operands, or zero when unsupported.
// Takes genOp which is the opcode for general (interface) operands, or zero when
// unsupported.
// Takes left which is the left operand register location.
// Takes right which is the right operand register location.
//
// Returns the int register location holding 0 or 1, or an error when the bank does not
// support the comparison.
func (c *compiler) emitCompareOp(ctx context.Context, intOp, floatOp, strOp, genOp opcode, left, right varLocation) (varLocation, error) {
	dest := c.scopes.alloc.alloc(registerInt)
	destinationLocation := varLocation{register: dest, kind: registerInt}

	switch left.kind {
	case registerInt:
		c.emitTyped(ctx, intOp, destinationLocation, left, right)
	case registerFloat:
		if floatOp == 0 {
			return varLocation{}, ErrCompileCompareFloatUnsupported
		}
		c.emitTyped(ctx, floatOp, destinationLocation, left, right)
	case registerString:
		if strOp == 0 {
			return varLocation{}, ErrCompileCompareStringUnsupported
		}
		c.emitTyped(ctx, strOp, destinationLocation, left, right)
	case registerBool:
		c.emitBoolCompare(ctx, intOp, dest, left, right)
	case registerUint:
		c.emitUintCompare(ctx, intOp, genOp, dest, left, right)
	case registerComplex:
		if err := c.emitComplexCompare(ctx, intOp, dest, left, right); err != nil {
			return varLocation{}, err
		}
	default:
		if genOp == 0 {
			return varLocation{}, ErrCompileCompareGeneralUnsupported
		}
		c.emitTyped(ctx, genOp, destinationLocation, left, right)
	}

	return destinationLocation, nil
}

// emitBoolCompare converts both bool operands to int via opBoolToInt in temporary
// registers and emits the integer comparison into dest.
//
// Takes intOp which is the integer comparison opcode to emit.
// Takes dest which is the destination int register receiving the 0/1 result.
// Takes left which is the left bool operand register location.
// Takes right which is the right bool operand register location.
func (c *compiler) emitBoolCompare(_ context.Context, intOp opcode, dest uint8, left, right varLocation) {
	leftInt := c.scopes.alloc.allocTemp(registerInt)
	rightInt := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opDrillTier1, uint8(subOpBoolToInt), leftInt, left.register)
	c.function.emit(opDrillTier1, uint8(subOpBoolToInt), rightInt, right.register)
	c.function.emit(intOp, dest, leftInt, rightInt)
	c.scopes.alloc.freeTemp(registerInt, leftInt)
	c.scopes.alloc.freeTemp(registerInt, rightInt)
}

// emitUintCompare maps intOp to its uint comparison counterpart and emits it into dest,
// falling back to genOp when no uint mapping exists and genOp is non-zero.
//
// Takes intOp which is the int comparison opcode whose uint counterpart is sought.
// Takes genOp which is the general-bank fallback opcode, or zero when no fallback.
// Takes dest which is the destination int register receiving the 0/1 result.
// Takes left which is the left uint operand register location.
// Takes right which is the right uint operand register location.
func (c *compiler) emitUintCompare(_ context.Context, intOp, genOp opcode, dest uint8, left, right varLocation) {
	uintCmpOp, ok := intToUintCmpOp(intOp)
	if !ok {
		if genOp != 0 {
			c.function.emit(genOp, dest, left.register, right.register)
		}
		return
	}
	c.function.emit(uintCmpOp, dest, left.register, right.register)
}

// emitComplexCompare emits opEqComplex or opNeComplex into dest.
// Returns an error for ordering operators because complex numbers support only == and !=
// in Go.
//
// Takes intOp which is the int-bank comparison opcode determining equality or inequality.
// Takes dest which is the destination int register receiving the 0/1 result.
// Takes left which is the left complex operand register location.
// Takes right which is the right complex operand register location.
//
// Returns a nil error for == and !=, or an error for ordering operators.
func (c *compiler) emitComplexCompare(_ context.Context, intOp opcode, dest uint8, left, right varLocation) error {
	switch intOp {
	case opEqInt:
		c.function.emit(opEqComplex, dest, left.register, right.register)
	case opNeInt:
		c.function.emit(opNeComplex, dest, left.register, right.register)
	default:
		return ErrCompileCompareComplexOrdering
	}
	return nil
}

// emitIntOnlyOp emits a bitwise or shift operation that requires integer operands. uint
// operands are routed to their uint-specific opcode; mixed int/uint operand pairs are
// bridged through an IntToUint or UintToInt temporary so the dispatched instruction
// receives matching banks.
//
// Takes op which is the int-bank opcode to emit (or whose uint counterpart applies).
// Takes left which is the left operand register location.
// Takes right which is the right operand register location.
//
// Returns the register location holding the result, or an error when the operand banks
// are unsupported for the operation.
func (c *compiler) emitIntOnlyOp(_ context.Context, op opcode, left, right varLocation) (varLocation, error) {
	if left.kind == registerUint {
		var uintOp opcode
		switch op {
		case opBitAnd:
			uintOp = opBitAndUint
		case opBitOr:
			uintOp = opBitOrUint
		case opBitXor:
			uintOp = opBitXorUint
		case opBitAndNot:
			uintOp = opBitAndNotUint
		case opShiftLeft:
			uintOp = opShiftLeftUint
		case opShiftRight:
			uintOp = opShiftRightUint
		default:
			return varLocation{}, ErrCompileArithUintUnsupported
		}
		dest := c.scopes.alloc.alloc(registerUint)
		rightRegister := right.register
		if right.kind == registerInt {
			rightRegister = c.scopes.alloc.allocTemp(registerUint)
			c.function.emit(opDrillTier1, uint8(subOpIntToUint), rightRegister, right.register)
		}
		c.function.emit(uintOp, dest, left.register, rightRegister)
		if right.kind == registerInt {
			c.scopes.alloc.freeTemp(registerUint, rightRegister)
		}
		return varLocation{register: dest, kind: registerUint}, nil
	}
	if left.kind != registerInt {
		return varLocation{}, ErrCompileBitwiseRequiresInteger
	}
	dest := c.scopes.alloc.alloc(registerInt)
	rightRegister := right.register
	if right.kind == registerUint {
		rightRegister = c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpUintToInt), rightRegister, right.register)
	}
	c.function.emit(op, dest, left.register, rightRegister)
	if right.kind == registerUint {
		c.scopes.alloc.freeTemp(registerInt, rightRegister)
	}
	return varLocation{register: dest, kind: registerInt}, nil
}

// ensureIntRegister converts location to the int bank in place.
//
// Coerces from every numeric-shaped bank that Go would accept as a slice index: uint,
// float, bool, and the general bank (where boxed values arrive after a method-receiver
// pass-through). Other kinds are left unchanged for callers that gate the operation
// themselves.
//
// Go method receivers of integer underlying type are declared on the general bank by
// compileFuncParams (the receiver kind defaults to registerGeneral irrespective of the
// declared type), so a method body that indexes a slice with the receiver relies on this
// path to unbox the receiver into int before the index op emits.
//
// Takes location which is the register location to coerce in place.
func (c *compiler) ensureIntRegister(_ context.Context, location *varLocation) {
	if location.kind == registerInt {
		return
	}
	switch location.kind {
	case registerUint:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpUintToInt), dest, location.register)
		location.register = dest
		location.kind = registerInt
	case registerGeneral:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpMoveGeneralToInt), dest, location.register)
		location.register = dest
		location.kind = registerInt
	case registerBool:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpBoolToInt), dest, location.register)
		location.register = dest
		location.kind = registerInt
	case registerFloat:
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpFloatToInt), dest, location.register)
		location.register = dest
		location.kind = registerInt
	default:
	}
}

// ensureIntForBranch returns location as an int register suitable for opJumpIfFalse /
// opJumpIfTrue. Bool locations are converted via opBoolToInt; general locations are first
// unpacked through opUnpackInterface to a bool and then converted to int.
//
// Takes location which is the register location to coerce to the int bank.
//
// Returns an int-bank register location holding the branch-ready value.
func (c *compiler) ensureIntForBranch(_ context.Context, location varLocation) varLocation {
	if location.kind == registerInt {
		return location
	}
	if location.kind == registerBool {
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpBoolToInt), dest, location.register)
		return varLocation{register: dest, kind: registerInt}
	}
	if location.kind == registerGeneral {
		booleanRegister := c.scopes.alloc.alloc(registerBool)
		c.function.emit(opUnpackInterface, booleanRegister, location.register, uint8(registerBool))
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opDrillTier1, uint8(subOpBoolToInt), dest, booleanRegister)
		return varLocation{register: dest, kind: registerInt}
	}
	return location
}

// scalarKindForConstant maps a go/constant kind to the scalar registerKind used for
// loading the value, returning registerGeneral for unknown kinds.
//
// Takes value which is the constant.Value whose kind selects the bank.
//
// Returns the matching register bank, or registerGeneral for unknown kinds.
func scalarKindForConstant(value constant.Value) registerKind {
	switch value.Kind() {
	case constant.Bool:
		return registerBool
	case constant.Int:
		return registerInt
	case constant.Float:
		return registerFloat
	case constant.String:
		return registerString
	case constant.Complex:
		return registerComplex
	default:
		return registerGeneral
	}
}

// intToUintArithOp maps an int arithmetic opcode to its uint counterpart, returning (0,
// false) when no mapping exists.
//
// Takes intOp which is the int-bank arithmetic opcode to translate.
//
// Returns the matching uint opcode and true, or (0, false) when no mapping exists.
func intToUintArithOp(intOp opcode) (opcode, bool) {
	switch intOp {
	case opAddInt:
		return opAddUint, true
	case opSubInt:
		return opSubUint, true
	case opMulInt:
		return opMulUint, true
	case opDivInt:
		return opDivUint, true
	case opRemInt:
		return opRemUint, true
	default:
		return 0, false
	}
}

// intToComplexArithOp maps an int arithmetic opcode to its complex counterpart, returning
// (0, false) when no mapping exists.
//
// Takes intOp which is the int-bank arithmetic opcode to translate.
//
// Returns the matching complex opcode and true, or (0, false) when no mapping exists.
func intToComplexArithOp(intOp opcode) (opcode, bool) {
	switch intOp {
	case opAddInt:
		return opAddComplex, true
	case opSubInt:
		return opSubComplex, true
	case opMulInt:
		return opMulComplex, true
	case opDivInt:
		return opDivComplex, true
	default:
		return 0, false
	}
}

// intToUintCmpOp maps an int comparison opcode to its uint counterpart, returning (0,
// false) when no mapping exists.
//
// Takes intOp which is the int-bank comparison opcode to translate.
//
// Returns the matching uint opcode and true, or (0, false) when no mapping exists.
func intToUintCmpOp(intOp opcode) (opcode, bool) {
	switch intOp {
	case opEqInt:
		return opEqUint, true
	case opNeInt:
		return opNeUint, true
	case opLtInt:
		return opLtUint, true
	case opLeInt:
		return opLeUint, true
	case opGtInt:
		return opGtUint, true
	case opGeInt:
		return opGeUint, true
	default:
		return 0, false
	}
}
