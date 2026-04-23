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

const (
	// builtinPrint identifies the print builtin function.
	builtinPrint uint8 = 1

	// builtinPrintln identifies the println builtin function.
	builtinPrintln uint8 = 2

	// builtinClear identifies the clear builtin function.
	builtinClear uint8 = 3

	// selectDirectionRecv indicates a receive operation in a select case.
	selectDirectionRecv uint8 = 0

	// selectDirectionSend indicates a send operation in a select case.
	selectDirectionSend uint8 = 1

	// selectDirectionDefault indicates the default case in a select statement.
	selectDirectionDefault uint8 = 2
)

// compileExpression compiles an expression and returns the register location
// where the result is stored.
//
// Takes expression (ast.Expr) which is the expression AST node to compile.
//
// Returns the register location of the result and any compilation error.
func (c *compiler) compileExpression(ctx context.Context, expression ast.Expr) (varLocation, error) {
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

// compileConstant compiles a compile-time constant value known from
// go/types. This handles constant folding -- the value has already
// been computed by the type checker.
//
// Takes tv (types.TypeAndValue) which is the type and constant value to
// compile.
//
// Returns the register location of the loaded constant and any error.
func (c *compiler) compileConstant(ctx context.Context, tv types.TypeAndValue) (varLocation, error) {
	kind := kindForType(tv.Type)
	if kind != registerGeneral {
		return c.emitScalarConstant(ctx, tv.Value, kind)
	}

	scalarKind := scalarKindForConstant(tv.Value)
	scalarLocation, err := c.emitScalarConstant(ctx, tv.Value, scalarKind)
	if err != nil {
		return varLocation{}, err
	}
	genReg := c.scopes.alloc.alloc(registerGeneral)
	c.function.emit(opPackInterface, genReg, scalarLocation.register, uint8(scalarLocation.kind))
	return varLocation{register: genReg, kind: registerGeneral}, nil
}

// emitScalarConstant emits bytecode to load a compile-time constant
// into a register of the given scalar kind.
//
// Takes value (constant.Value) which is the constant value to load.
// Takes kind (registerKind) which is the target register kind for the
// constant.
//
// Returns the register location of the loaded constant and any error.
func (c *compiler) emitScalarConstant(_ context.Context, value constant.Value, kind registerKind) (varLocation, error) {
	switch kind {
	case registerBool:
		register := c.scopes.alloc.alloc(registerBool)
		index := c.function.addBoolConstant(constant.BoolVal(value))
		c.function.emitWide(opLoadBoolConst, register, index)
		return varLocation{register: register, kind: registerBool}, nil
	case registerInt:
		v, ok := constant.Int64Val(value)
		if !ok {
			return varLocation{}, fmt.Errorf("cannot convert constant to int64: %v", value)
		}
		index := c.function.addIntConstant(v)
		register := c.scopes.alloc.alloc(registerInt)
		c.function.emitWide(opLoadIntConst, register, index)
		return varLocation{register: register, kind: registerInt}, nil
	case registerUint:
		u, ok := constant.Uint64Val(value)
		if !ok {
			v, vOk := constant.Int64Val(value)
			if vOk {
				u = uint64(v) //nolint:gosec // intentional int64->uint64
				ok = true
			}
		}
		if !ok {
			return varLocation{}, fmt.Errorf("cannot convert constant to uint64: %v", value)
		}
		index := c.function.addUintConstant(u)
		register := c.scopes.alloc.alloc(registerUint)
		c.function.emitWide(opLoadUintConst, register, index)
		return varLocation{register: register, kind: registerUint}, nil
	case registerFloat:
		v, _ := constant.Float64Val(value)
		index := c.function.addFloatConstant(v)
		register := c.scopes.alloc.alloc(registerFloat)
		c.function.emitWide(opLoadFloatConst, register, index)
		return varLocation{register: register, kind: registerFloat}, nil
	case registerString:
		v := constant.StringVal(value)
		index := c.function.addStringConstant(v)
		register := c.scopes.alloc.alloc(registerString)
		c.function.emitWide(opLoadStringConst, register, index)
		return varLocation{register: register, kind: registerString}, nil
	case registerComplex:
		realPart, _ := constant.Float64Val(constant.Real(value))
		imaginaryPart, _ := constant.Float64Val(constant.Imag(value))
		index := c.function.addComplexConstant(complex(realPart, imaginaryPart))
		register := c.scopes.alloc.alloc(registerComplex)
		c.function.emitWide(opLoadComplexConst, register, index)
		return varLocation{register: register, kind: registerComplex}, nil
	default:
		return varLocation{}, fmt.Errorf("unsupported constant kind %v for register bank (value: %v)", kind, value)
	}
}

// compileBasicLit compiles a basic literal (number, string, etc.).
// This is the fallback when go/types doesn't provide a constant value.
//
// Takes lit (*ast.BasicLit) which is the basic literal AST node to compile.
//
// Returns the register location of the loaded literal and any error.
func (c *compiler) compileBasicLit(ctx context.Context, lit *ast.BasicLit) (varLocation, error) {
	tv, ok := c.info.Types[lit]
	if ok && tv.Value != nil {
		return c.compileConstant(ctx, tv)
	}
	return varLocation{}, fmt.Errorf("basic literal without type info: %s", lit.Value)
}

// compileIdent compiles an identifier reference.
//
// Takes identifier (*ast.Ident) which is the identifier AST node to compile.
//
// Returns the register location of the resolved identifier and any error.
func (c *compiler) compileIdent(ctx context.Context, identifier *ast.Ident) (varLocation, error) {
	if identifier.Name == identTrue {
		register := c.scopes.alloc.alloc(registerBool)
		index := c.function.addBoolConstant(true)
		c.function.emitWide(opLoadBoolConst, register, index)
		return varLocation{register: register, kind: registerBool}, nil
	}
	if identifier.Name == identFalse {
		register := c.scopes.alloc.alloc(registerBool)
		index := c.function.addBoolConstant(false)
		c.function.emitWide(opLoadBoolConst, register, index)
		return varLocation{register: register, kind: registerBool}, nil
	}
	if identifier.Name == identNil {
		register := c.scopes.alloc.alloc(registerGeneral)
		c.function.emit(opLoadNil, register, 0, 0)
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

	if gv, ok := c.globalVars[identifier.Name]; ok {
		return c.emitGetGlobal(ctx, gv), nil
	}

	if funcIndex, found := c.funcTable[identifier.Name]; found {
		dest := c.scopes.alloc.alloc(registerGeneral)
		c.function.emitWide(opMakeClosure, dest, funcIndex)
		return varLocation{register: dest, kind: registerGeneral}, nil
	}

	return varLocation{}, fmt.Errorf("undefined: %s at %s", identifier.Name, c.positionString(identifier.Pos()))
}

// compileBinaryExpression compiles a binary expression (a op b).
//
// Takes expression (*ast.BinaryExpr) which is the binary expression AST
// node to compile.
//
// Returns the register location of the result and any compilation error.
func (c *compiler) compileBinaryExpression(ctx context.Context, expression *ast.BinaryExpr) (varLocation, error) {
	if expression.Op == token.LAND {
		return c.compileShortCircuitAnd(ctx, expression)
	}
	if expression.Op == token.LOR {
		return c.compileShortCircuitOr(ctx, expression)
	}

	left, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}

	right, err := c.compileExpression(ctx, expression.Y)
	if err != nil {
		return varLocation{}, err
	}

	return c.emitBinaryOp(ctx, expression.Op, left, right)
}

// compileShortCircuitAnd compiles a logical AND expression with short-circuit
// evaluation, skipping the right operand when the left is false.
//
// Takes expression (*ast.BinaryExpr) which is the binary AND expression to
// compile.
//
// Returns the int register location (0 or 1) and any
// compilation error.
func (c *compiler) compileShortCircuitAnd(ctx context.Context, expression *ast.BinaryExpr) (varLocation, error) {
	return c.compileShortCircuit(ctx, expression, opJumpIfFalse)
}

// compileShortCircuitOr compiles a logical OR expression with
// short-circuit evaluation, skipping the right operand when the
// left is true.
//
// Takes expression (*ast.BinaryExpr) which is the binary OR
// expression to compile.
//
// Returns the int register location (0 or 1) and any
// compilation error.
func (c *compiler) compileShortCircuitOr(ctx context.Context, expression *ast.BinaryExpr) (varLocation, error) {
	return c.compileShortCircuit(ctx, expression, opJumpIfTrue)
}

// compileShortCircuit compiles a short-circuit boolean
// expression using the given skip opcode to conditionally
// bypass the right operand.
//
// Takes expression (*ast.BinaryExpr) which is the binary
// expression to compile.
// Takes skipOp (opcode) which is the jump opcode used to skip
// the right operand.
//
// Returns the int register location (0 or 1) and any
// compilation error.
func (c *compiler) compileShortCircuit(ctx context.Context, expression *ast.BinaryExpr, skipOp opcode) (varLocation, error) {
	left, err := c.compileExpression(ctx, expression.X)
	if err != nil {
		return varLocation{}, err
	}
	left = c.ensureIntForBranch(ctx, left)
	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(opMoveInt, dest, left.register, 0)
	jumpToEnd := c.function.emitJump(skipOp, dest)
	right, err := c.compileExpression(ctx, expression.Y)
	if err != nil {
		return varLocation{}, err
	}
	right = c.ensureIntForBranch(ctx, right)
	c.function.emit(opMoveInt, dest, right.register, 0)
	c.function.patchJump(jumpToEnd)
	return varLocation{register: dest, kind: registerInt}, nil
}

// emitBinaryOp emits the appropriate instruction for a binary operation.
//
// Takes op (token.Token) which is the binary operator token.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns the register location of the result and any compilation error.
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

// emitArithOp emits a type-specialised arithmetic instruction.
//
// Takes intOp (opcode) which is the opcode for integer operands.
// Takes floatOp (opcode) which is the opcode for float operands.
// Takes strOp (opcode) which is the opcode for string operands.
// Takes genOp (opcode) which is the fallback opcode for general
// operands.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns the register location of the result and any compilation error.
func (c *compiler) emitArithOp(ctx context.Context, intOp, floatOp, strOp, genOp opcode, left, right varLocation) (varLocation, error) {
	switch left.kind {
	case registerInt:
		return c.emitTypedArith(ctx, intOp, registerInt, left, right)
	case registerFloat:
		if floatOp == 0 {
			return varLocation{}, errors.New("operation not supported for float")
		}
		return c.emitTypedArith(ctx, floatOp, registerFloat, left, right)
	case registerString:
		if strOp == 0 {
			return varLocation{}, errors.New("operation not supported for string")
		}
		return c.emitTypedArith(ctx, strOp, registerString, left, right)
	case registerUint:
		return c.emitArithUint(ctx, intOp, left, right)
	case registerComplex:
		return c.emitArithComplex(ctx, intOp, left, right)
	default:
		if genOp == 0 {
			return varLocation{}, errors.New("operation not supported for this type")
		}
		return c.emitTypedArith(ctx, genOp, registerGeneral, left, right)
	}
}

// emitTypedArith emits a single arithmetic instruction with the given
// opcode and result register kind.
//
// Takes op (opcode) which is the arithmetic opcode to emit.
// Takes kind (registerKind) which is the register kind for the
// result.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns the register location of the result and any compilation error.
func (c *compiler) emitTypedArith(_ context.Context, op opcode, kind registerKind, left, right varLocation) (varLocation, error) {
	dest := c.scopes.alloc.alloc(kind)
	c.function.emit(op, dest, left.register, right.register)
	return varLocation{register: dest, kind: kind}, nil
}

// emitArithUint resolves the uint-specific opcode for an arithmetic
// operation and emits it.
//
// Takes intOp (opcode) which is the integer opcode to map to its
// uint equivalent.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns the register location of the result and any compilation error.
func (c *compiler) emitArithUint(ctx context.Context, intOp opcode, left, right varLocation) (varLocation, error) {
	uintOp, ok := intToUintArithOp(intOp)
	if !ok {
		return varLocation{}, errors.New("operation not supported for uint")
	}
	return c.emitTypedArith(ctx, uintOp, registerUint, left, right)
}

// emitArithComplex resolves the complex-specific opcode for an
// arithmetic operation and emits it.
//
// Takes intOp (opcode) which is the integer opcode to map to its
// complex equivalent.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns the register location of the result and any compilation error.
func (c *compiler) emitArithComplex(ctx context.Context, intOp opcode, left, right varLocation) (varLocation, error) {
	complexOp, ok := intToComplexArithOp(intOp)
	if !ok {
		return varLocation{}, errors.New("operation not supported for complex")
	}
	return c.emitTypedArith(ctx, complexOp, registerComplex, left, right)
}

// emitCompareOp emits a type-specialised comparison instruction.
// The result is always in an int register (0 or 1).
//
// Takes intOp (opcode) which is the opcode for integer comparison.
// Takes floatOp (opcode) which is the opcode for float comparison.
// Takes strOp (opcode) which is the opcode for string comparison.
// Takes genOp (opcode) which is the fallback opcode for general
// comparison.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns the register location of the comparison result and any error.
func (c *compiler) emitCompareOp(ctx context.Context, intOp, floatOp, strOp, genOp opcode, left, right varLocation) (varLocation, error) {
	dest := c.scopes.alloc.alloc(registerInt)

	switch left.kind {
	case registerInt:
		c.function.emit(intOp, dest, left.register, right.register)
	case registerFloat:
		if floatOp == 0 {
			return varLocation{}, errors.New("comparison not supported for float")
		}
		c.function.emit(floatOp, dest, left.register, right.register)
	case registerString:
		if strOp == 0 {
			return varLocation{}, errors.New("comparison not supported for string")
		}
		c.function.emit(strOp, dest, left.register, right.register)
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
			return varLocation{}, errors.New("comparison not supported for this type")
		}
		c.function.emit(genOp, dest, left.register, right.register)
	}

	return varLocation{register: dest, kind: registerInt}, nil
}

// emitBoolCompare converts bool operands to int and emits the
// comparison.
//
// Takes intOp (opcode) which is the integer comparison opcode to
// use.
// Takes dest (uint8) which is the destination register for the
// result.
// Takes left (varLocation) which is the left bool operand location.
// Takes right (varLocation) which is the right bool operand
// location.
func (c *compiler) emitBoolCompare(_ context.Context, intOp opcode, dest uint8, left, right varLocation) {
	leftInt := c.scopes.alloc.allocTemp(registerInt)
	rightInt := c.scopes.alloc.allocTemp(registerInt)
	c.function.emit(opBoolToInt, leftInt, left.register, 0)
	c.function.emit(opBoolToInt, rightInt, right.register, 0)
	c.function.emit(intOp, dest, leftInt, rightInt)
	c.scopes.alloc.freeTemp(registerInt, leftInt)
	c.scopes.alloc.freeTemp(registerInt, rightInt)
}

// emitUintCompare resolves the uint-specific comparison opcode and
// emits it, falling back to genOp when no mapping exists.
//
// Takes intOp (opcode) which is the integer comparison opcode to
// map.
// Takes genOp (opcode) which is the fallback general opcode.
// Takes dest (uint8) which is the destination register for the
// result.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
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

// emitComplexCompare resolves the complex-specific comparison opcode
// and emits it. Only == and != are supported for complex numbers.
//
// Takes intOp (opcode) which is the integer comparison opcode to map.
// Takes dest (uint8) which is the destination register for the result.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns an error if the comparison operator is not supported for
// complex numbers.
func (c *compiler) emitComplexCompare(_ context.Context, intOp opcode, dest uint8, left, right varLocation) error {
	switch intOp {
	case opEqInt:
		c.function.emit(opEqComplex, dest, left.register, right.register)
	case opNeInt:
		c.function.emit(opNeComplex, dest, left.register, right.register)
	default:
		return errors.New("comparison not supported for complex (only == and !=)")
	}
	return nil
}

// emitIntOnlyOp emits an integer-only operation.
//
// Takes op (opcode) which is the integer opcode to emit.
// Takes left (varLocation) which is the left operand location.
// Takes right (varLocation) which is the right operand location.
//
// Returns the register location of the result and any compilation error.
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
			return varLocation{}, errors.New("operation not supported for uint")
		}
		rightRegister := right.register
		if right.kind == registerInt {
			rightRegister = c.scopes.alloc.allocTemp(registerUint)
			c.function.emit(opIntToUint, rightRegister, right.register, 0)
		}
		dest := c.scopes.alloc.alloc(registerUint)
		c.function.emit(uintOp, dest, left.register, rightRegister)
		if right.kind == registerInt {
			c.scopes.alloc.freeTemp(registerUint, rightRegister)
		}
		return varLocation{register: dest, kind: registerUint}, nil
	}
	if left.kind != registerInt {
		return varLocation{}, errors.New("operation requires integer operands")
	}
	rightRegister := right.register
	if right.kind == registerUint {
		rightRegister = c.scopes.alloc.allocTemp(registerInt)
		c.function.emit(opUintToInt, rightRegister, right.register, 0)
	}
	dest := c.scopes.alloc.alloc(registerInt)
	c.function.emit(op, dest, left.register, rightRegister)
	if right.kind == registerUint {
		c.scopes.alloc.freeTemp(registerInt, rightRegister)
	}
	return varLocation{register: dest, kind: registerInt}, nil
}

// ensureIntRegister converts a variable location to an int register in place,
// performing a uint-to-int conversion if needed.
//
// Takes location (*varLocation) which is the variable location to convert.
// It is modified in place if conversion is performed.
func (c *compiler) ensureIntRegister(_ context.Context, location *varLocation) {
	if location.kind == registerInt {
		return
	}
	if location.kind == registerUint {
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opUintToInt, dest, location.register, 0)
		location.register = dest
		location.kind = registerInt
	}
}

// ensureIntForBranch converts a variable location to an int register for
// use in branch instructions. Bool registers are converted via
// opBoolToInt; general (interface) registers are first unpacked to a
// bool register via opUnpackInterface and then converted to int. The
// general path is required because type-assert results (e.g.
// v.(bool)) land in the general bank but opJumpIfFalse reads from int.
//
// Takes location (varLocation) which is the variable location to
// convert.
//
// Returns the location converted to an int register, or the original
// location if no conversion is needed.
func (c *compiler) ensureIntForBranch(_ context.Context, location varLocation) varLocation {
	if location.kind == registerInt {
		return location
	}
	if location.kind == registerBool {
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opBoolToInt, dest, location.register, 0)
		return varLocation{register: dest, kind: registerInt}
	}
	if location.kind == registerGeneral {
		boolReg := c.scopes.alloc.alloc(registerBool)
		c.function.emit(opUnpackInterface, boolReg, location.register, uint8(registerBool))
		dest := c.scopes.alloc.alloc(registerInt)
		c.function.emit(opBoolToInt, dest, boolReg, 0)
		return varLocation{register: dest, kind: registerInt}
	}
	return location
}

// scalarKindForConstant maps a go/constant kind to the scalar
// registerKind used for loading the value.
//
// Takes value (constant.Value) which is the constant value whose kind
// determines the register kind mapping.
//
// Returns the scalar registerKind corresponding to the constant's kind.
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

// intToUintArithOp maps an int arithmetic opcode to its uint
// equivalent.
//
// Takes intOp (opcode) which is the integer arithmetic opcode to map.
//
// Returns the uint opcode and true if a mapping exists, or zero and false
// if no mapping exists.
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

// intToComplexArithOp maps an int arithmetic opcode to its complex
// equivalent.
//
// Takes intOp (opcode) which is the integer arithmetic opcode to map.
//
// Returns the complex opcode and true if a mapping exists, or zero and
// false if no mapping exists.
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

// intToUintCmpOp maps an int comparison opcode to its uint equivalent.
//
// Takes intOp (opcode) which is the integer comparison opcode to map.
//
// Returns the uint comparison opcode and true if a mapping exists, or
// zero and false if no mapping exists.
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
