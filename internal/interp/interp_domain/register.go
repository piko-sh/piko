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
	"go/types"
	"reflect"
)

// registerKind identifies which typed register bank a value belongs to.
// Using separate banks for common primitive types avoids the overhead of
// boxing/unboxing values in reflect.Value for the majority of operations.
type registerKind uint8

const (
	// registerInt stores int64 values. All Go signed integer types (int, int8,
	// int16, int32, int64) and untyped int/rune are stored here, using
	// int64 as the common representation.
	registerInt registerKind = iota

	// registerFloat stores float64 values. Both float32 and float64 are stored
	// here, with float32 promoted to float64.
	registerFloat

	// registerString stores string values natively.
	registerString

	// registerGeneral stores reflect.Value for all other types: interfaces,
	// pointers, slices, maps, arrays, structs, channels, and functions.
	registerGeneral

	// registerBool stores bool values natively.
	registerBool

	// registerUint stores uint64 values. All Go unsigned integer types (uint,
	// uint8, uint16, uint32, uint64, uintptr) are stored here.
	registerUint

	// registerComplex stores complex128 values. Both complex64 and complex128
	// are stored here, with complex64 promoted to complex128.
	registerComplex
)

// NumRegisterKinds is the number of register bank types.
const NumRegisterKinds = 7

// String returns the human-readable name of the register kind.
//
// Returns the register kind name as a string.
func (k registerKind) String() string {
	switch k {
	case registerInt:
		return "int"
	case registerFloat:
		return "float"
	case registerString:
		return "string"
	case registerGeneral:
		return "general"
	case registerBool:
		return "bool"
	case registerUint:
		return "uint"
	case registerComplex:
		return "complex"
	default:
		return "unknown"
	}
}

// Registers holds the seven typed register banks for a call frame.
type Registers struct {
	// ints stores int64 values for the integer register bank.
	ints []int64

	// floats stores float64 values for the float register bank.
	floats []float64

	// strings stores string values for the string register bank.
	strings []string

	// general stores reflect.Value values for the general register bank.
	general []reflect.Value

	// bools stores bool values for the boolean register bank.
	bools []bool

	// uints stores uint64 values for the unsigned integer register bank.
	uints []uint64

	// complex stores complex128 values for the complex register bank.
	complex []complex128
}

// NewRegistersForBench is an exported wrapper for benchmarking direct
// allocation vs arena allocation.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of
// registers per bank.
//
// Returns a freshly allocated register file.
func NewRegistersForBench(numRegs [NumRegisterKinds]uint32) Registers {
	return newRegisters(numRegs)
}

// newRegisters creates a register file sized for a compiled function.
//
// Takes numRegs ([NumRegisterKinds]uint32) which is the number of
// registers per bank.
//
// Returns a freshly allocated register file.
func newRegisters(numRegs [NumRegisterKinds]uint32) Registers {
	return Registers{
		ints:    make([]int64, numRegs[registerInt]),
		floats:  make([]float64, numRegs[registerFloat]),
		strings: make([]string, numRegs[registerString]),
		general: make([]reflect.Value, numRegs[registerGeneral]),
		bools:   make([]bool, numRegs[registerBool]),
		uints:   make([]uint64, numRegs[registerUint]),
		complex: make([]complex128, numRegs[registerComplex]),
	}
}

// kindForType determines the register kind for a given Go type.
//
// Takes t (types.Type) which is the Go type to classify.
//
// Returns the register kind for the type's register bank.
func kindForType(t types.Type) registerKind {
	t = t.Underlying()

	if basic, ok := t.(*types.Basic); ok {
		return kindForBasic(basic.Kind())
	}

	return registerGeneral
}

// kindForBasic maps a types.BasicKind to a registerKind.
//
// Takes k (types.BasicKind) which is the basic type kind to map.
//
// Returns the corresponding register kind.
func kindForBasic(k types.BasicKind) registerKind {
	switch k {
	case types.Bool, types.UntypedBool:
		return registerBool

	case types.Int, types.Int8, types.Int16, types.Int32, types.Int64,
		types.UntypedInt, types.UntypedRune:
		return registerInt

	case types.Uint, types.Uint8, types.Uint16, types.Uint32, types.Uint64,
		types.Uintptr:
		return registerUint

	case types.Float32, types.Float64, types.UntypedFloat:
		return registerFloat

	case types.String, types.UntypedString:
		return registerString

	case types.Complex64, types.Complex128, types.UntypedComplex:
		return registerComplex

	default:
		return registerGeneral
	}
}
