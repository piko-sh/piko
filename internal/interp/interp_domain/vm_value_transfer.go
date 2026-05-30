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
	"reflect"
)

// copySameKind copies a register value between frames when the source and destination
// kinds are identical.
//
// Takes destination (*Registers) which specifies the destination register banks.
// Takes source (*Registers) which specifies the source register banks.
// Takes kind (registerKind) which specifies which register bank to copy from.
// Takes destinationRegister (uint8) which specifies the destination register index.
// Takes sourceRegister (uint8) which specifies the source register index.
//
// copyRegisterSlot shares one register set; this crosses frames.
//
//nolint:dupl // 12-bank dispatch shape mirrors copyRegisterSlot
func copySameKind(destination, source *Registers, kind registerKind, destinationRegister, sourceRegister uint8) {
	switch kind {
	case registerInt:
		destination.ints[destinationRegister] = source.ints[sourceRegister]
	case registerFloat:
		destination.floats[destinationRegister] = source.floats[sourceRegister]
	case registerString:
		destination.strings[destinationRegister] = source.strings[sourceRegister]
	case registerGeneral:
		destination.general[destinationRegister] = source.general[sourceRegister]
	case registerBool:
		destination.bools[destinationRegister] = source.bools[sourceRegister]
	case registerUint:
		destination.uints[destinationRegister] = source.uints[sourceRegister]
	case registerComplex:
		destination.complex[destinationRegister] = source.complex[sourceRegister]
	case registerSliceInt:
		destination.slicesInt[destinationRegister] = source.slicesInt[sourceRegister]
	case registerSliceFloat:
		destination.slicesFloat[destinationRegister] = source.slicesFloat[sourceRegister]
	case registerSliceString:
		destination.slicesString[destinationRegister] = source.slicesString[sourceRegister]
	case registerSliceBool:
		destination.slicesBool[destinationRegister] = source.slicesBool[sourceRegister]
	case registerSliceUint:
		destination.slicesUint[destinationRegister] = source.slicesUint[sourceRegister]
	case registerSliceByte:
		destination.slicesByte[destinationRegister] = source.slicesByte[sourceRegister]
	default:
	}
}

// copyReturnFromGeneral unpacks a general-register value into the caller's typed register
// bank, handling interface unwrapping and zero-value defaults for generics compiled as
// any.
//
// Takes callerFrame (*callFrame) which specifies the frame to write the value into.
// Takes reflectValue (reflect.Value) which specifies the general-register value to
// unpack.
// Takes destination (varLocation) which specifies the destination typed register
// location.
func copyReturnFromGeneral(callerFrame *callFrame, reflectValue reflect.Value, destination varLocation) {
	if reflectValue.IsValid() && reflectValue.Kind() == reflect.Interface {
		reflectValue = reflectValue.Elem()
	}
	if !reflectValue.IsValid() {
		zeroTypedRegister(&callerFrame.registers, destination)
		return
	}
	unpackGeneralToTyped(&callerFrame.registers, reflectValue, destination)
}

// zeroTypedRegister writes a zero value into the destination register.
//
// Takes registers (*Registers) which specifies the register banks to write into.
// Takes destination (varLocation) which specifies the register location to zero.
func zeroTypedRegister(registers *Registers, destination varLocation) {
	switch destination.kind {
	case registerInt:
		registers.ints[destination.register] = 0
	case registerFloat:
		registers.floats[destination.register] = 0
	case registerString:
		registers.strings[destination.register] = ""
	case registerBool:
		registers.bools[destination.register] = false
	case registerUint:
		registers.uints[destination.register] = 0
	case registerComplex:
		registers.complex[destination.register] = 0
	case registerSliceInt:
		registers.slicesInt[destination.register] = nil
	case registerSliceFloat:
		registers.slicesFloat[destination.register] = nil
	case registerSliceString:
		registers.slicesString[destination.register] = nil
	case registerSliceBool:
		registers.slicesBool[destination.register] = nil
	case registerSliceUint:
		registers.slicesUint[destination.register] = nil
	case registerSliceByte:
		registers.slicesByte[destination.register] = nil
	default:
	}
}

// unpackGeneralToTyped extracts a concrete value from a reflect.Value into the
// appropriate typed register bank.
//
// Takes registers (*Registers) which specifies the register banks to write into.
// Takes v (reflect.Value) which specifies the reflect.Value to extract the concrete value
// from.
// Takes destination (varLocation) which specifies the destination typed register
// location.
//
// One clean switch over registerKind reads better than splitting per scalar/slice; Go
// compiles the dense enum switch to a jump table so the apparent complexity is constant
// runtime cost.
//
//nolint:revive // cyclomatic: dense registerKind switch
func unpackGeneralToTyped(registers *Registers, v reflect.Value, destination varLocation) {
	switch destination.kind {
	case registerInt:
		if v.Kind() == reflect.Bool {
			registers.ints[destination.register] = boolToInt64(v.Bool())
		} else {
			registers.ints[destination.register] = v.Int()
		}
	case registerFloat:
		registers.floats[destination.register] = v.Float()
	case registerString:
		registers.strings[destination.register] = v.String()
	case registerBool:
		registers.bools[destination.register] = v.Bool()
	case registerUint:
		registers.uints[destination.register] = v.Uint()
	case registerComplex:
		registers.complex[destination.register] = v.Complex()
	case registerSliceInt:
		if slice, ok := reflect.TypeAssert[[]int64](v); ok {
			registers.slicesInt[destination.register] = slice
		}
	case registerSliceFloat:
		if slice, ok := reflect.TypeAssert[[]float64](v); ok {
			registers.slicesFloat[destination.register] = slice
		}
	case registerSliceString:
		if slice, ok := reflect.TypeAssert[[]string](v); ok {
			registers.slicesString[destination.register] = slice
		}
	case registerSliceBool:
		if slice, ok := reflect.TypeAssert[[]bool](v); ok {
			registers.slicesBool[destination.register] = slice
		}
	case registerSliceUint:
		if slice, ok := reflect.TypeAssert[[]uint64](v); ok {
			registers.slicesUint[destination.register] = slice
		}
	case registerSliceByte:
		if slice, ok := reflect.TypeAssert[[]byte](v); ok {
			registers.slicesByte[destination.register] = slice
		}
	default:
	}
}

// copyReturnToGeneral boxes a typed register value into a reflect.Value and stores it in
// the caller's general register.
//
// Takes callerFrame (*callFrame) which specifies the frame whose general register
// receives the value.
// Takes sourceRegisters (*Registers) which specifies the source register banks containing
// the typed value.
// Takes kind (registerKind) which specifies which typed register bank to read from.
// Takes sourceRegister (uint8) which specifies the source register index.
// Takes destinationRegister (uint8) which specifies the destination general register
// index.
func copyReturnToGeneral(callerFrame *callFrame, sourceRegisters *Registers, kind registerKind, sourceRegister, destinationRegister uint8) {
	switch kind {
	case registerInt:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.ints[sourceRegister])
	case registerFloat:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.floats[sourceRegister])
	case registerString:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.strings[sourceRegister])
	case registerBool:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.bools[sourceRegister])
	case registerUint:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.uints[sourceRegister])
	case registerComplex:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.complex[sourceRegister])
	case registerSliceInt:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.slicesInt[sourceRegister])
	case registerSliceFloat:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.slicesFloat[sourceRegister])
	case registerSliceString:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.slicesString[sourceRegister])
	case registerSliceBool:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.slicesBool[sourceRegister])
	case registerSliceUint:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.slicesUint[sourceRegister])
	case registerSliceByte:
		callerFrame.registers.general[destinationRegister] = reflect.ValueOf(sourceRegisters.slicesByte[sourceRegister])
	default:
	}
}

// assignReflectParams writes reflect.Value arguments into the typed register banks
// according to the compiled parameter kinds.
//
// Takes registers (*Registers) which specifies the register banks to write into.
// Takes parameterKinds ([]registerKind) which specifies the register kind for each
// parameter position.
// Takes arguments ([]reflect.Value) which provides the reflect.Value arguments to assign.
func assignReflectParams(registers *Registers, parameterKinds []registerKind, arguments []reflect.Value) {
	var parameterIndex [NumRegisterKinds]uint8
	for i, argument := range arguments {
		if i >= len(parameterKinds) {
			break
		}
		if argument.Kind() == reflect.Interface && !argument.IsNil() {
			argument = argument.Elem()
		}
		kind := parameterKinds[i]
		register := parameterIndex[kind]
		parameterIndex[kind]++
		assignReflectArg(registers, kind, register, argument)
	}
}

// assignReflectArg writes a single reflect.Value argument into the register bank for the
// given kind.
//
// Takes registers (*Registers) which specifies the register banks to write into.
// Takes kind (registerKind) which specifies which register bank to target.
// Takes register (uint8) which specifies the register index within the bank.
// Takes argument (reflect.Value) which provides the reflect.Value to assign.
//
// One clean switch over registerKind reads better than splitting per scalar/slice; Go
// compiles the dense enum switch to a jump table so the apparent complexity is constant
// runtime cost.
//
//nolint:revive // cyclomatic: dense registerKind switch
func assignReflectArg(registers *Registers, kind registerKind, register uint8, argument reflect.Value) {
	switch kind {
	case registerInt:
		registers.ints[register] = argument.Int()
	case registerFloat:
		registers.floats[register] = argument.Float()
	case registerString:
		registers.strings[register] = argument.String()
	case registerBool:
		registers.bools[register] = argument.Bool()
	case registerUint:
		registers.uints[register] = argument.Uint()
	case registerComplex:
		registers.complex[register] = argument.Complex()
	case registerSliceInt:
		if slice, ok := reflect.TypeAssert[[]int64](argument); ok {
			registers.slicesInt[register] = slice
		} else if slice, ok := sliceFromReflectField[int64](argument); ok {
			registers.slicesInt[register] = slice
		}
	case registerSliceFloat:
		if slice, ok := reflect.TypeAssert[[]float64](argument); ok {
			registers.slicesFloat[register] = slice
		} else if slice, ok := sliceFromReflectField[float64](argument); ok {
			registers.slicesFloat[register] = slice
		}
	case registerSliceString:
		if slice, ok := reflect.TypeAssert[[]string](argument); ok {
			registers.slicesString[register] = slice
		} else if slice, ok := sliceFromReflectField[string](argument); ok {
			registers.slicesString[register] = slice
		}
	case registerSliceBool:
		if slice, ok := reflect.TypeAssert[[]bool](argument); ok {
			registers.slicesBool[register] = slice
		}
	case registerSliceUint:
		if slice, ok := reflect.TypeAssert[[]uint64](argument); ok {
			registers.slicesUint[register] = slice
		} else if slice, ok := sliceFromReflectField[uint64](argument); ok {
			registers.slicesUint[register] = slice
		}
	case registerSliceByte:
		if slice, ok := reflect.TypeAssert[[]byte](argument); ok {
			registers.slicesByte[register] = slice
		} else if argument.Kind() == reflect.Slice && argument.Type().Elem().Kind() == reflect.Uint8 {
			registers.slicesByte[register] = argument.Bytes()
		}
	default:

		registers.general[register] = valueCopyForBoundary(argument)
	}
}

// buildReflectResults packages the VM's results into a slice of reflect.Value matching
// the function's output signature.
//
// Takes allResults ([]any) which specifies all return values from the VM execution.
// Takes funcType (reflect.Type) which defines the function signature whose output types
// determine the result packaging.
//
// Returns a slice of reflect.Values with results converted to match funcType's output
// types.
func buildReflectResults(allResults []any, funcType reflect.Type) []reflect.Value {
	outputCount := funcType.NumOut()
	results := make([]reflect.Value, outputCount)
	for i := range outputCount {
		outputType := funcType.Out(i)
		if i < len(allResults) && allResults[i] != nil {
			reflectValue := reflect.ValueOf(allResults[i])
			if reflectValue.Type().ConvertibleTo(outputType) {
				results[i] = reflectValue.Convert(outputType)
			} else {
				results[i] = reflectValue
			}
		} else {
			results[i] = reflect.Zero(outputType)
		}
	}
	return results
}

// syncCellToRegister copies the value from an upvalue cell back into the register at the
// given location.
//
// Takes registers (*Registers) which specifies the register banks to write into.
// Takes cell (*upvalueCell) which specifies the upvalue cell to read from.
// Takes location (varLocation) which specifies the register location to write to.
//
//nolint:dupl // 12-bank shape mirrors writeRegisterToCell.
func syncCellToRegister(registers *Registers, cell *upvalueCell, location varLocation) {
	switch location.kind {
	case registerInt:
		registers.ints[location.register] = cell.intValue
	case registerFloat:
		registers.floats[location.register] = cell.floatValue
	case registerString:
		registers.strings[location.register] = cell.stringValue
	case registerGeneral:
		registers.general[location.register] = cell.generalValue
	case registerBool:
		registers.bools[location.register] = cell.boolValue
	case registerUint:
		registers.uints[location.register] = cell.uintValue
	case registerComplex:
		registers.complex[location.register] = cell.complexValue
	case registerSliceInt:
		registers.slicesInt[location.register] = cell.sliceIntValue
	case registerSliceFloat:
		registers.slicesFloat[location.register] = cell.sliceFloatValue
	case registerSliceString:
		registers.slicesString[location.register] = cell.sliceStringValue
	case registerSliceBool:
		registers.slicesBool[location.register] = cell.sliceBoolValue
	case registerSliceUint:
		registers.slicesUint[location.register] = cell.sliceUintValue
	case registerSliceByte:
		registers.slicesByte[location.register] = cell.sliceByteValue
	default:
	}
}

// copyRegisterSlot copies a value within the same register bank from sourceRegister to
// destinationRegister. Delegates to copySameKind passing the same register set as both
// source and destination.
//
// Takes registers (*Registers) which specifies the register banks to operate on.
// Takes kind (registerKind) which specifies which register bank to copy within.
// Takes destinationRegister (uint8) which specifies the destination register index.
// Takes sourceRegister (uint8) which specifies the source register index.
func copyRegisterSlot(registers *Registers, kind registerKind, destinationRegister, sourceRegister uint8) {
	copySameKind(registers, registers, kind, destinationRegister, sourceRegister)
}
