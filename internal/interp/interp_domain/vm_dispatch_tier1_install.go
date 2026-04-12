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

//go:build !safe && !(js && wasm) && (amd64 || arm64)

package interp_domain

// handlerSubOpLenSliceIntDirect is the ASM body for the tier-1 sub-op that reads the
// length of an []int slice direct from its header.
//
//go:noescape
func handlerSubOpLenSliceIntDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpLenSliceFloatDirect is the ASM body for the tier-1 sub-op that reads the
// length of a []float64 slice direct.
//
//go:noescape
func handlerSubOpLenSliceFloatDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpLenSliceStringDirect is the ASM body for the tier-1 sub-op that reads the
// length of a []string slice direct.
//
//go:noescape
func handlerSubOpLenSliceStringDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpLenSliceBoolDirect is the ASM body for the tier-1 sub-op that reads the
// length of a []bool slice direct.
//
//go:noescape
func handlerSubOpLenSliceBoolDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpLenSliceUintDirect is the ASM body for the tier-1 sub-op that reads the
// length of a []uint slice direct.
//
//go:noescape
func handlerSubOpLenSliceUintDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceGetFloatDirect is the ASM body for the tier-1 indexed read of a
// []float64 element.
//
//go:noescape
func handlerSubOpSliceGetFloatDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceSetFloatDirect is the ASM body for the tier-1 indexed write of a
// []float64 element.
//
//go:noescape
func handlerSubOpSliceSetFloatDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceGetUintDirect is the ASM body for the tier-1 indexed read of a []uint
// element.
//
//go:noescape
func handlerSubOpSliceGetUintDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceSetUintDirect is the ASM body for the tier-1 indexed write of a []uint
// element.
//
//go:noescape
func handlerSubOpSliceSetUintDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceGetBoolDirect is the ASM body for the tier-1 indexed read of a []bool
// element.
//
//go:noescape
func handlerSubOpSliceGetBoolDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceSetBoolDirect is the ASM body for the tier-1 indexed write of a []bool
// element.
//
//go:noescape
func handlerSubOpSliceSetBoolDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceGetStringDirect is the ASM body for the tier-1 indexed read of a
// []string element.
//
//go:noescape
func handlerSubOpSliceGetStringDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceSetStringDirect is the ASM body for the tier-1 indexed write of a
// []string element.
//
//go:noescape
func handlerSubOpSliceSetStringDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpRealComplex is the ASM body for the tier-1 sub-op that extracts the real
// component of a complex128 value.
//
//go:noescape
func handlerSubOpRealComplex() //nolint:unused // ASM handler, JMP target only

// handlerSubOpImagComplex is the ASM body for the tier-1 sub-op that extracts the
// imaginary component of a complex128 value.
//
//go:noescape
func handlerSubOpImagComplex() //nolint:unused // ASM handler, JMP target only

// handlerSubOpMoveComplex is the ASM body for the tier-1 complex128 move (16-byte copy
// from source to destination register).
//
//go:noescape
func handlerSubOpMoveComplex() //nolint:unused // ASM handler, 16-byte copy

// handlerSubOpNegComplex is the ASM body for the tier-1 complex128 negate, implemented as
// a sign-bit XOR on both lanes.
//
//go:noescape
func handlerSubOpNegComplex() //nolint:unused // ASM handler, sign-bit XOR

// handlerSubOpMoveInt is the ASM body for the tier-1 int move.
//
//go:noescape
func handlerSubOpMoveInt() //nolint:unused // ASM handler, tier-1 move

// handlerSubOpMoveFloat is the ASM body for the tier-1 float64 move.
//
//go:noescape
func handlerSubOpMoveFloat() //nolint:unused // ASM handler, tier-1 float move

// handlerSubOpMoveBool is the ASM body for the tier-1 bool move.
//
//go:noescape
func handlerSubOpMoveBool() //nolint:unused // ASM handler, tier-1 bool move (context-base loaded)

// handlerSubOpMoveUint is the ASM body for the tier-1 uint move.
//
//go:noescape
func handlerSubOpMoveUint() //nolint:unused // ASM handler, tier-1 uint move (context-base loaded)

// handlerSubOpMoveString is the ASM body for the tier-1 string move copying the 16-byte
// string header.
//
//go:noescape
func handlerSubOpMoveString() //nolint:unused // ASM handler, tier-1 string move (16-byte header copy)

// handlerSubOpMoveSliceInt is the ASM body for the tier-1 []int64 slice-header move
// (24-byte copy of {Data,Len,Cap}).
//
//go:noescape
func handlerSubOpMoveSliceInt() //nolint:unused // ASM handler, tier-1 typed-slice move

// handlerSubOpMoveSliceFloat is the ASM body for the tier-1 []float64 slice-header move
// (24-byte copy of {Data,Len,Cap}).
//
//go:noescape
func handlerSubOpMoveSliceFloat() //nolint:unused // ASM handler, tier-1 typed-slice move

// handlerSubOpMoveSliceString is the ASM body for the tier-1 []string slice-header move
// (24-byte copy of {Data,Len,Cap}).
//
//go:noescape
func handlerSubOpMoveSliceString() //nolint:unused // ASM handler, tier-1 typed-slice move

// handlerSubOpMoveSliceBool is the ASM body for the tier-1 []bool slice-header move
// (24-byte copy of {Data,Len,Cap}).
//
//go:noescape
func handlerSubOpMoveSliceBool() //nolint:unused // ASM handler, tier-1 typed-slice move

// handlerSubOpMoveSliceUint is the ASM body for the tier-1 []uint64 slice-header move
// (24-byte copy of {Data,Len,Cap}).
//
//go:noescape
func handlerSubOpMoveSliceUint() //nolint:unused // ASM handler, tier-1 typed-slice move

// handlerSubOpMoveSliceByte is the ASM body for the tier-1 []byte slice-header move
// (24-byte copy of {Data,Len,Cap}).
//
//go:noescape
func handlerSubOpMoveSliceByte() //nolint:unused // ASM handler, tier-1 typed-slice move

// handlerSubOpSliceSliceIntDirect is the ASM body for the tier-1 []int64 sub-slicing op
// (xs[lo:hi]) with both bounds in registers.
//
//go:noescape
func handlerSubOpSliceSliceIntDirect() //nolint:unused // ASM handler, tier-1 typed-slice sub-slice

// handlerSubOpSliceSliceFloatDirect is the ASM body for the tier-1 []float64 sub-slicing
// op (xs[lo:hi]) with both bounds in registers.
//
//go:noescape
func handlerSubOpSliceSliceFloatDirect() //nolint:unused // ASM handler, tier-1 typed-slice sub-slice

// handlerSubOpSliceSliceStringDirect is the ASM body for the tier-1 []string sub-slicing
// op (xs[lo:hi]) with both bounds in registers.
//
//go:noescape
func handlerSubOpSliceSliceStringDirect() //nolint:unused // ASM handler, tier-1 typed-slice sub-slice

// handlerSubOpSliceSliceBoolDirect is the ASM body for the tier-1 []bool sub-slicing op
// (xs[lo:hi]) with both bounds in registers.
//
//go:noescape
func handlerSubOpSliceSliceBoolDirect() //nolint:unused // ASM handler, tier-1 typed-slice sub-slice

// handlerSubOpSliceSliceUintDirect is the ASM body for the tier-1 []uint64 sub-slicing op
// (xs[lo:hi]) with both bounds in registers.
//
//go:noescape
func handlerSubOpSliceSliceUintDirect() //nolint:unused // ASM handler, tier-1 typed-slice sub-slice

// handlerSubOpTier2IncInt is the ASM body for the tier-2 in-place int increment,
// installed in tier-1 because it carries one operand.
//
//go:noescape
func handlerSubOpTier2IncInt() //nolint:unused // ASM handler, tier-2 in-place

// handlerSubOpTier2DecInt is the ASM body for the tier-2 in-place int decrement,
// installed in tier-1 because it carries one operand.
//
//go:noescape
func handlerSubOpTier2DecInt() //nolint:unused // ASM handler, tier-2 in-place

// handlerSubOpNegInt is the ASM body for the tier-1 int negate.
//
//go:noescape
func handlerSubOpNegInt() //nolint:unused // ASM handler, tier-1 unary

// handlerSubOpNegFloat is the ASM body for the tier-1 float64 negate.
//
//go:noescape
func handlerSubOpNegFloat() //nolint:unused // ASM handler, tier-1 unary

// handlerSubOpBitNot is the ASM body for the tier-1 bitwise NOT on integer registers.
//
//go:noescape
func handlerSubOpBitNot() //nolint:unused // ASM handler, tier-1 unary

// handlerSubOpIntToFloat is the ASM body for the tier-1 int to float64 conversion.
//
//go:noescape
func handlerSubOpIntToFloat() //nolint:unused // ASM handler, tier-1 conversion

// handlerSubOpFloatToInt is the ASM body for the tier-1 float64 to int conversion.
//
//go:noescape
func handlerSubOpFloatToInt() //nolint:unused // ASM handler, tier-1 conversion

// handlerSubOpUintToFloat is the ASM body for the tier-1 uint to float64 conversion.
//
//go:noescape
func handlerSubOpUintToFloat() //nolint:unused // ASM handler, tier-1 conversion

// handlerSubOpFloatToUint is the ASM body for the tier-1 float64 to uint conversion.
//
//go:noescape
func handlerSubOpFloatToUint() //nolint:unused // ASM handler, tier-1 conversion

// handlerSubOpMathSqrt is the ASM body for the tier-1 math.Sqrt unary.
//
//go:noescape
func handlerSubOpMathSqrt() //nolint:unused // ASM handler, tier-1 math unary

// handlerSubOpMathAbs is the ASM body for the tier-1 math.Abs unary.
//
//go:noescape
func handlerSubOpMathAbs() //nolint:unused // ASM handler, tier-1 math unary

// handlerSubOpMathFloor is the ASM body for the tier-1 math.Floor unary.
//
//go:noescape
func handlerSubOpMathFloor() //nolint:unused // ASM handler, tier-1 math unary

// handlerSubOpMathCeil is the ASM body for the tier-1 math.Ceil unary.
//
//go:noescape
func handlerSubOpMathCeil() //nolint:unused // ASM handler, tier-1 math unary

// handlerSubOpMathTrunc is the ASM body for the tier-1 math.Trunc unary.
//
//go:noescape
func handlerSubOpMathTrunc() //nolint:unused // ASM handler, tier-1 math unary

// handlerSubOpMathRound is the ASM body for the tier-1 math.Round unary.
//
//go:noescape
func handlerSubOpMathRound() //nolint:unused // ASM handler, tier-1 math unary

// handlerSubOpLenString is the ASM body for the tier-1 string length read, taking the
// length word from the 16-byte string header.
//
//go:noescape
func handlerSubOpLenString() //nolint:unused // ASM handler, tier-1 string length

// handlerSubOpMathSin is the NOFRAME shim installed in tier1JumpTable for math.Sin; it
// CALLs handlerSubOpMathSinReal then tail-jumps via DISPATCH_NEXT.
//
//go:noescape
func handlerSubOpMathSin() //nolint:unused // ASM shim, CALLs handlerSubOpMathSinReal

// handlerSubOpMathCos is the NOFRAME shim installed in tier1JumpTable for math.Cos; it
// CALLs handlerSubOpMathCosReal then tail-jumps via DISPATCH_NEXT.
//
//go:noescape
func handlerSubOpMathCos() //nolint:unused // ASM shim, CALLs handlerSubOpMathCosReal

// handlerSubOpMathExp is the NOFRAME shim installed in tier1JumpTable for math.Exp; it
// CALLs handlerSubOpMathExpReal then tail-jumps via DISPATCH_NEXT.
//
//go:noescape
func handlerSubOpMathExp() //nolint:unused // ASM shim, CALLs handlerSubOpMathExpReal

// handlerSubOpMathTan is the NOFRAME shim installed in tier1JumpTable for math.Tan; it
// CALLs handlerSubOpMathTanReal then tail-jumps via DISPATCH_NEXT.
//
//go:noescape
func handlerSubOpMathTan() //nolint:unused // ASM shim, CALLs handlerSubOpMathTanReal

// handlerSubOpMathSinReal is the framed real handler for math.Sin, CALLed by its shim so
// Plan-9 manages the abi0-call frame.
//
//go:noescape
func handlerSubOpMathSinReal() //nolint:unused // ASM real handler, CALLed by shim

// handlerSubOpMathCosReal is the framed real handler for math.Cos, CALLed by its shim so
// Plan-9 manages the abi0-call frame.
//
//go:noescape
func handlerSubOpMathCosReal() //nolint:unused // ASM real handler, CALLed by shim

// handlerSubOpMathExpReal is the framed real handler for math.Exp, CALLed by its shim so
// Plan-9 manages the abi0-call frame.
//
//go:noescape
func handlerSubOpMathExpReal() //nolint:unused // ASM real handler, CALLed by shim

// handlerSubOpMathTanReal is the framed real handler for math.Tan, CALLed by its shim so
// Plan-9 manages the abi0-call frame.
//
//go:noescape
func handlerSubOpMathTanReal() //nolint:unused // ASM real handler, CALLed by shim

// handlerSubOpMathMod is the NOFRAME shim installed in tier1JumpTable for math.Mod; it
// CALLs handlerSubOpMathModReal then tail-jumps via DISPATCH_NEXT.
//
//go:noescape
func handlerSubOpMathMod() //nolint:unused // ASM shim, CALLs handlerSubOpMathModReal

// handlerSubOpMathModReal is the framed real handler for math.Mod, CALLed by its shim so
// Plan-9 manages the abi0-call frame.
//
//go:noescape
func handlerSubOpMathModReal() //nolint:unused // ASM real handler, CALLed by shim

// handlerSubOpStrconvFormatBool is the inlined ASM body for tier-1 strconv.FormatBool;
// the result is a small fixed string so no Go trampoline is required.
//
//go:noescape
func handlerSubOpStrconvFormatBool() //nolint:unused // ASM inlined handler (no Go trampoline)

// handlerSubOpStrconvItoa is the NOFRAME shim for tier-1 strconv.Itoa; CALLs
// handlerSubOpStrconvItoaReal then tail-jumps via DISPATCH_NEXT.
//
//go:noescape
func handlerSubOpStrconvItoa() //nolint:unused // ASM shim

// handlerSubOpStrconvItoaReal is the framed real handler for strconv.Itoa, CALLed by its
// shim.
//
//go:noescape
func handlerSubOpStrconvItoaReal() //nolint:unused // ASM real handler

// handlerSubOpStrconvFormatInt is the NOFRAME shim for tier-1 strconv.FormatInt; CALLs
// handlerSubOpStrconvFormatIntReal.
//
//go:noescape
func handlerSubOpStrconvFormatInt() //nolint:unused // ASM shim

// handlerSubOpStrconvFormatIntReal is the framed real handler for strconv.FormatInt,
// CALLed by its shim.
//
//go:noescape
func handlerSubOpStrconvFormatIntReal() //nolint:unused // ASM real handler

// handlerSubOpCap is the NOFRAME shim for tier-1 cap(); CALLs handlerSubOpCapReal.
//
//go:noescape
func handlerSubOpCap() //nolint:unused // ASM shim

// handlerSubOpCapReal is the framed real handler for cap(), CALLed by its shim.
//
//go:noescape
func handlerSubOpCapReal() //nolint:unused // ASM real handler

// handlerSubOpBytesToString is the NOFRAME shim for tier-1 []byte to string conversion;
// CALLs handlerSubOpBytesToStringReal.
//
//go:noescape
func handlerSubOpBytesToString() //nolint:unused // ASM shim

// handlerSubOpBytesToStringReal is the framed real handler for []byte-to-string, CALLed
// by its shim.
//
//go:noescape
func handlerSubOpBytesToStringReal() //nolint:unused // ASM real handler

// handlerSubOpBoxSliceInt is the NOFRAME shim for tier-1 boxing of []int into an any;
// CALLs handlerSubOpBoxSliceIntReal.
//
//go:noescape
func handlerSubOpBoxSliceInt() //nolint:unused // ASM shim

// handlerSubOpBoxSliceIntReal is the framed real handler for boxing []int, CALLed by its
// shim.
//
//go:noescape
func handlerSubOpBoxSliceIntReal() //nolint:unused // ASM real handler

// handlerSubOpUnboxSliceInt is the NOFRAME shim for tier-1 unboxing of an any back into
// []int; CALLs handlerSubOpUnboxSliceIntReal.
//
//go:noescape
func handlerSubOpUnboxSliceInt() //nolint:unused // ASM shim

// handlerSubOpUnboxSliceIntReal is the framed real handler for unboxing []int, CALLed by
// its shim.
//
//go:noescape
func handlerSubOpUnboxSliceIntReal() //nolint:unused // ASM real handler

// handlerSubOpMakeSliceInt is the NOFRAME shim for tier-1 make([]int, n); CALLs
// handlerSubOpMakeSliceIntReal.
//
//go:noescape
func handlerSubOpMakeSliceInt() //nolint:unused // ASM shim

// handlerSubOpMakeSliceIntReal is the framed real handler for make([]int, n), CALLed by
// its shim.
//
//go:noescape
func handlerSubOpMakeSliceIntReal() //nolint:unused // ASM real handler

// handlerSubOpMakeSliceFloat is the NOFRAME shim for tier-1 make([]float64, n); CALLs
// handlerSubOpMakeSliceFloatReal.
//
//go:noescape
func handlerSubOpMakeSliceFloat() //nolint:unused // ASM shim

// handlerSubOpMakeSliceFloatReal is the framed real handler for make([]float64, n),
// CALLed by its shim.
//
//go:noescape
func handlerSubOpMakeSliceFloatReal() //nolint:unused // ASM real handler

// handlerSubOpMakeSliceString is the NOFRAME shim for tier-1 make([]string, n); CALLs
// handlerSubOpMakeSliceStringReal.
//
//go:noescape
func handlerSubOpMakeSliceString() //nolint:unused // ASM shim

// handlerSubOpMakeSliceStringReal is the framed real handler for make([]string, n),
// CALLed by its shim.
//
//go:noescape
func handlerSubOpMakeSliceStringReal() //nolint:unused // ASM real handler

// handlerSubOpMakeSliceBool is the NOFRAME shim for tier-1 make([]bool, n); CALLs
// handlerSubOpMakeSliceBoolReal.
//
//go:noescape
func handlerSubOpMakeSliceBool() //nolint:unused // ASM shim

// handlerSubOpMakeSliceBoolReal is the framed real handler for make([]bool, n), CALLed by
// its shim.
//
//go:noescape
func handlerSubOpMakeSliceBoolReal() //nolint:unused // ASM real handler

// handlerSubOpMakeSliceUint is the NOFRAME shim for tier-1 make([]uint, n); CALLs
// handlerSubOpMakeSliceUintReal.
//
//go:noescape
func handlerSubOpMakeSliceUint() //nolint:unused // ASM shim

// handlerSubOpMakeSliceUintReal is the framed real handler for make([]uint, n), CALLed by
// its shim.
//
//go:noescape
func handlerSubOpMakeSliceUintReal() //nolint:unused // ASM real handler

// handlerSubOpMakeSliceByte is the NOFRAME shim for tier-1 make([]byte, n); CALLs
// handlerSubOpMakeSliceByteReal.
//
//go:noescape
func handlerSubOpMakeSliceByte() //nolint:unused // ASM shim

// handlerSubOpMakeSliceByteReal is the framed real handler for make([]byte, n), CALLed by
// its shim.
//
//go:noescape
func handlerSubOpMakeSliceByteReal() //nolint:unused // ASM real handler

// handlerSubOpLenSliceByteDirect is the ASM body for the tier-1 read of the length of a
// []byte slice direct from its header.
//
//go:noescape
func handlerSubOpLenSliceByteDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceGetByteDirect is the ASM body for the tier-1 indexed read of a []byte
// element.
//
//go:noescape
func handlerSubOpSliceGetByteDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceSetByteDirect is the ASM body for the tier-1 indexed write of a []byte
// element.
//
//go:noescape
func handlerSubOpSliceSetByteDirect() //nolint:unused // ASM handler, JMP target only

// handlerSubOpSliceByteSlice is the ASM body for the tier-1 sub-slice operation on a
// []byte producing a new header.
//
//go:noescape
func handlerSubOpSliceByteSlice() //nolint:unused // ASM handler, JMP target only

// handlerSubOpRangeNextSliceByte is the ASM body for the tier-1 range-next step over a
// []byte, advancing the loop iterator.
//
//go:noescape
func handlerSubOpRangeNextSliceByte() //nolint:unused // ASM handler, JMP target only

// handlerSubOpIncStructFieldInt is the NOFRAME shim for tier-1 in-place increment of an
// int struct field; CALLs the framed real handler.
//
//go:noescape
func handlerSubOpIncStructFieldInt() //nolint:unused // ASM shim

// handlerSubOpIncStructFieldIntReal is the framed real handler for in-place increment of
// an int struct field, CALLed by its shim.
//
//go:noescape
func handlerSubOpIncStructFieldIntReal() //nolint:unused // ASM real handler

// handlerSubOpDecStructFieldInt is the NOFRAME shim for tier-1 in-place decrement of an
// int struct field; CALLs the framed real handler.
//
//go:noescape
func handlerSubOpDecStructFieldInt() //nolint:unused // ASM shim

// handlerSubOpDecStructFieldIntReal is the framed real handler for in-place decrement of
// an int struct field, CALLed by its shim.
//
//go:noescape
func handlerSubOpDecStructFieldIntReal() //nolint:unused // ASM real handler

// handlerSubOpIncStructFieldUint is the NOFRAME shim for tier-1 in-place increment of a
// uint struct field; CALLs the framed real handler.
//
//go:noescape
func handlerSubOpIncStructFieldUint() //nolint:unused // ASM shim

// handlerSubOpIncStructFieldUintReal is the framed real handler for in-place increment of
// a uint struct field, CALLed by its shim.
//
//go:noescape
func handlerSubOpIncStructFieldUintReal() //nolint:unused // ASM real handler

// handlerSubOpDecStructFieldUint is the NOFRAME shim for tier-1 in-place decrement of a
// uint struct field; CALLs the framed real handler.
//
//go:noescape
func handlerSubOpDecStructFieldUint() //nolint:unused // ASM shim

// handlerSubOpDecStructFieldUintReal is the framed real handler for in-place decrement of
// a uint struct field, CALLed by its shim.
//
//go:noescape
func handlerSubOpDecStructFieldUintReal() //nolint:unused // ASM real handler

// handlerSubOpRangeCheckUintJumpFalse is the ASM body for the tier-1 fused uint
// range-check + conditional jump used by range loops.
//
//go:noescape
func handlerSubOpRangeCheckUintJumpFalse() //nolint:unused // ASM handler, JMP target only

// handlerSubOpTier2IncUint is the ASM body for the tier-2 in-place uint increment,
// installed in tier-1 because it carries one operand.
//
//go:noescape
func handlerSubOpTier2IncUint() //nolint:unused // ASM handler, tier-2 in-place

// handlerSubOpTier2DecUint is the ASM body for the tier-2 in-place uint decrement,
// installed in tier-1 because it carries one operand.
//
//go:noescape
func handlerSubOpTier2DecUint() //nolint:unused // ASM handler, tier-2 in-place

// installTier1SliceTypedASM is a no-op retained for call-site symmetry.
//
// Tier-1 typed-slice sub-op handlers are installed by the asmgen-driven
// initSubOpJumpTables (in the generated init ASM file), which uses LEAQ to take the .abi0
// address directly instead of reflect.ValueOf (which would store the ABIInternal wrapper
// and accumulate stack frames per hot-path dispatch).
func installTier1SliceTypedASM() {}

// installTier1ComplexASM is a no-op retained for call-site symmetry.
//
// Tier-1 complex sub-ops are installed by initSubOpJumpTables.
func installTier1ComplexASM() {}

// installTier1MathASM is a no-op retained for call-site symmetry.
//
// Tier-1 math sub-ops (Sin/Cos/Exp/Tan/Mod) are installed by initSubOpJumpTables. Each is
// a NOSPLIT|NOFRAME shim in tier1JumpTable that CALLs a paired "real" handler ($48
// auto-managed frame for marshalling args to the abi0 wrapper), then tail-jumps via
// DISPATCH_NEXT after the real handler RETs. Trampolines in asm_call_trampolines.go
// bridge to math.Sin / Cos / Exp / Tan / Mod via Plan-9's autogenerated abi0 wrapper.
//
// The shim+real split is mandated by a Plan-9 ASM / stack-walker conflict. Plan-9 ASM
// resolves cross-package CALLs to a Go function as funcname.abi0 (the <ABIInternal>
// selector is restricted to the runtime package), and calling that wrapper writes args at
// FP-relative offsets so the caller needs a non-zero frame. Plan-9 with a non-zero frame
// auto-emits PUSHQ BP + SUBQ $N, SP at the prologue and balances it on RET, but these
// handlers do not RET: they tail-jump via DISPATCH_NEXT, the auto-epilogue never fires,
// and a manual ADDQ + POPQ BP triggers "unbalanced PUSH/POP" from the assembler. With
// NOFRAME + manual SUBQ/ADDQ the Go stack walker still uses the declared TEXT frame size,
// so any preemption-driven stack walk between manual ADDQ and tail JMP reads garbage.
//
// The shim (NOFRAME) and real (framed) split avoids the walker mismatch: the framed
// callee handles prologue/epilogue around the abi0 CALL; the shim then performs the JMP
// tail with SP equal to its declared (zero) frame.
func installTier1MathASM() {}

// installTier1StrconvASM is a no-op retained for call-site symmetry.
//
// Tier-1 strconv sub-ops are installed by initSubOpJumpTables.
func installTier1StrconvASM() {}

// installTier1RuntimeASM is a no-op retained for call-site symmetry.
//
// Tier-1 runtime sub-ops are installed by initSubOpJumpTables.
func installTier1RuntimeASM() {}

// installTier1ControlASM is a no-op retained for call-site symmetry.
//
// Tier-1 control-flow handlers (handlerJump, handlerLoadIntConstSmall, handlerLoadBool,
// handlerIncIntJumpLt, handlerLenStringLtJumpFalse, handlerReturnInline) are installed by
// initSubOpJumpTables in the generated init ASM file, which uses LEAQ to take the .abi0
// address directly. The init() routines call initSubOpJumpTables after this stub.
//
// handlerJump is reused unchanged from its tier-0 installation because the operand bytes
// it reads (B|C as a 16-bit signed jump offset) sit at the same byte positions in both
// the tier-0 and tier-1 instruction encodings. handlerLoadIntConstSmall and
// handlerLoadBool extract operands from byte positions B and C; the asmgen factories in
// handlers_arithmetic.go emit ExtractB+ExtractC.
func installTier1ControlASM() {}
