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

package interp_adapters

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"piko.sh/piko/wdk/safeconv"

	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/interp/interp_schema"
	"piko.sh/piko/internal/interp/interp_schema/interp_schema_gen"
	"piko.sh/piko/internal/mem"
)

const (
	// maxBytecodeTypeDescriptorDepth caps the recursion depth when reconstructing nested
	// type descriptors from a cached bytecode file. The cap defends against stack exhaustion
	// from a tampered or corrupted on-disk payload.
	maxBytecodeTypeDescriptorDepth = 256

	// maxBytecodeFunctionNestingDepth caps the recursion depth when reconstructing nested
	// compiled functions (closures within closures) from a cached bytecode file.
	maxBytecodeFunctionNestingDepth = 256

	// structFieldLayoutMaxPathDepth mirrors interp_domain's path-depth cap so reconstructed
	// StructFieldLayoutData arrays match the domain type's array size at compile time.
	structFieldLayoutMaxPathDepth = 4
)

const (
	// slotBankInt is the global-store bank holding int64 slots.
	slotBankInt = 0

	// slotBankFloat is the global-store bank holding float64 slots.
	slotBankFloat = 1

	// slotBankString is the global-store bank holding string slots.
	slotBankString = 2

	// slotBankBool is the global-store bank holding bool slots.
	slotBankBool = 3

	// slotBankUint is the global-store bank holding uint64 slots.
	slotBankUint = 4

	// slotBankComplex is the global-store bank holding complex128 slots.
	slotBankComplex = 5

	// slotBankGeneral is the global-store bank holding reflect.Value slots.
	slotBankGeneral = 6
)

var (
	// errBytecodeRecursionDepthExceeded indicates that the recursion depth bound was reached
	// while unpacking a cached bytecode file. The cache should be considered corrupt or
	// tampered and discarded.
	errBytecodeRecursionDepthExceeded = errors.New("bytecode unpack recursion depth exceeded")

	// errCorruptBytecodePayload signals a tampered bytecode payload.
	errCorruptBytecodePayload = errors.New("corrupt bytecode payload")
)

// boundedCount validates a FlatBuffer vector length.
//
// A vector of n elements occupies at least n bytes, so any length greater than the
// payload size proves the payload is truncated or tampered. Returning an error here stops
// a small crafted payload from forcing a multi-gigabyte make.
//
// Takes declared (int) which is the length the FlatBuffer accessor reported.
// Takes payloadLen (int) which is the total payload size in bytes.
// Takes what (string) which names the vector for the error message.
//
// Returns the validated length, or an error when it is implausible.
func boundedCount(declared, payloadLen int, what string) (int, error) {
	if declared < 0 || declared > payloadLen {
		return 0, fmt.Errorf("%w: %s count %d exceeds payload size %d bytes", errCorruptBytecodePayload, what, declared, payloadLen)
	}
	return declared, nil
}

// validatedRegisterKind converts a serialised register-kind byte to a uint8 after
// confirming it names a real register kind. A tampered payload can carry an out-of-range
// byte (including values that would panic safeconv.MustInt8ToUint8); rejecting it keeps
// unpacking from corrupting a register bank or crashing the host.
//
// Takes raw (int8) which is the register-kind value read from the FlatBuffer.
// Takes what (string) which names the field for the error message.
//
// Returns the validated kind byte, or an error when it is out of range.
func validatedRegisterKind(raw int8, what string) (uint8, error) {
	if raw < 0 || int(raw) >= interp_domain.NumRegisterKinds {
		return 0, fmt.Errorf("%w: %s register kind %d out of range [0,%d)", errCorruptBytecodePayload, what, raw, interp_domain.NumRegisterKinds)
	}
	return uint8(raw), nil
}

// LoadCompiledFromBytes deserialises a piko-packed bytecode payload directly from memory.
//
// Decodes the output of PackCompiledFileSetToBytes without a disk-backed store. Used by
// hosts that transport bytecode through a module-bundle field
// ([modules_domain.ModuleBundle.Bytecode]) rather than via the whole-program
// BytecodeStore disk cache.
//
// Signature matches [interp_domain.Service.LoadModule]'s bytecodeUnpacker parameter so
// hosts can pass the function directly. Cancellation uses context.Background internally:
// payloads are short and the unpack is CPU-bound.
//
// Takes data ([]byte) which carries the packed bytecode payload.
// Takes registry (*interp_domain.SymbolRegistry) which provides symbol and type lookups
// for runtime reconstruction.
//
// Returns *interp_domain.CompiledFileSet which is the reconstructed file set.
// Returns error when the schema header doesn't match the current binary or the FlatBuffer
// payload is corrupt.
func LoadCompiledFromBytes(data []byte, registry *interp_domain.SymbolRegistry) (*interp_domain.CompiledFileSet, error) {
	if len(data) == 0 {
		return nil, errors.New("bytecode payload is empty")
	}
	payload, err := interp_schema.Unpack(data)
	if err != nil {
		return nil, fmt.Errorf("unpacking bytecode header: %w", err)
	}
	fbFileSet := interp_schema_gen.GetRootAsCompiledFileSet(payload, 0)
	if fbFileSet == nil {
		return nil, errors.New("bytecode payload is corrupt: cannot parse FlatBuffer root")
	}
	return unpackCompiledFileSet(context.Background(), fbFileSet, registry, len(payload))
}

// unpackCompiledFileSet reconstructs a CompiledFileSet from its FlatBuffer
// representation.
//
// Takes fbFileSet (*interp_schema_gen.CompiledFileSet) which is the serialised FlatBuffer
// file set.
// Takes registry (*interp_domain.SymbolRegistry) which provides symbol and type lookups
// for runtime reconstruction.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound every declared vector length.
//
// Returns *interp_domain.CompiledFileSet which is the reconstructed compiled file set.
// Returns error when unpacking any function fails or a declared vector length exceeds the
// payload size.
func unpackCompiledFileSet(ctx context.Context, fbFileSet *interp_schema_gen.CompiledFileSet, registry *interp_domain.SymbolRegistry, payloadLen int) (*interp_domain.CompiledFileSet, error) {
	var root *interp_domain.CompiledFunction
	if fbRoot := fbFileSet.Root(nil); fbRoot != nil {
		var err error
		root, err = unpackCompiledFunction(ctx, fbRoot, registry, 0, payloadLen)
		if err != nil {
			return nil, fmt.Errorf("unpacking root function: %w", err)
		}
	}

	var variableInitFunction *interp_domain.CompiledFunction
	if fbVarInit := fbFileSet.VariableInitFunction(nil); fbVarInit != nil {
		var err error
		variableInitFunction, err = unpackCompiledFunction(ctx, fbVarInit, registry, 0, payloadLen)
		if err != nil {
			return nil, fmt.Errorf("unpacking variable init function: %w", err)
		}
	}

	entrypoints, err := unpackEntrypoints(fbFileSet, payloadLen)
	if err != nil {
		return nil, err
	}

	initCount, err := boundedCount(int(fbFileSet.InitialisationFunctionsLength()), payloadLen, "initialisation functions")
	if err != nil {
		return nil, err
	}
	initFunctionIndices := make([]uint16, initCount)
	for i := range initCount {
		initFunctionIndices[i] = fbFileSet.InitialisationFunctions(i)
	}

	slotAllocation := unpackSlotAllocation(fbFileSet)

	packageVariables, err := unpackPackageVariables(fbFileSet, payloadLen)
	if err != nil {
		return nil, err
	}

	return interp_domain.NewCompiledFileSetFromDataWithVars(root, variableInitFunction, entrypoints, initFunctionIndices, slotAllocation, packageVariables), nil
}

// unpackEntrypoints reconstructs the entrypoint name-to-index map from its FlatBuffer
// entries.
//
// Takes fbFileSet (*interp_schema_gen.CompiledFileSet) which contains the serialised
// entrypoint entries.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared entry count.
//
// Returns map[string]uint16 which maps entrypoint names to their function indices.
// Returns error when the declared entry count exceeds the payload size.
func unpackEntrypoints(fbFileSet *interp_schema_gen.CompiledFileSet, payloadLen int) (map[string]uint16, error) {
	count, err := boundedCount(int(fbFileSet.EntrypointsLength()), payloadLen, "entrypoints")
	if err != nil {
		return nil, err
	}
	entrypoints := make(map[string]uint16, count)
	var fbEntrypoint interp_schema_gen.EntrypointEntry
	for i := range count {
		if fbFileSet.Entrypoints(&fbEntrypoint, i) {
			entrypoints[mem.String(fbEntrypoint.Name())] = fbEntrypoint.FunctionIndex()
		}
	}
	return entrypoints, nil
}

// unpackSlotAllocation reconstructs the per-bank global-store slot reservation counts
// from their FlatBuffer struct. A nil struct (no reservations were serialised) yields the
// zero allocation.
//
// Takes fbFileSet (*interp_schema_gen.CompiledFileSet) which contains the serialised slot
// allocation.
//
// Returns interp_domain.SlotAllocation with one count per bank.
func unpackSlotAllocation(fbFileSet *interp_schema_gen.CompiledFileSet) interp_domain.SlotAllocation {
	var slotAllocation interp_domain.SlotAllocation
	if alloc := fbFileSet.SlotAllocation(nil); alloc != nil {
		slotAllocation[slotBankInt] = alloc.IntCount()
		slotAllocation[slotBankFloat] = alloc.FloatCount()
		slotAllocation[slotBankString] = alloc.StringCount()
		slotAllocation[slotBankBool] = alloc.BoolCount()
		slotAllocation[slotBankUint] = alloc.UintCount()
		slotAllocation[slotBankComplex] = alloc.ComplexCount()
		slotAllocation[slotBankGeneral] = alloc.GeneralCount()
	}
	return slotAllocation
}

// unpackPackageVariables reconstructs the exported package-variable metadata vector,
// recursively unpacking each entry's optional type descriptor.
//
// Takes fbFileSet (*interp_schema_gen.CompiledFileSet) which contains the serialised
// package-variable entries.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared entry count.
//
// Returns []interp_domain.PackageVariableMetadata which holds the reconstructed metadata.
// Returns error when a serialised type descriptor cannot be unpacked, the declared entry
// count exceeds the payload size, or a register kind is out of range.
func unpackPackageVariables(fbFileSet *interp_schema_gen.CompiledFileSet, payloadLen int) ([]interp_domain.PackageVariableMetadata, error) {
	count, err := boundedCount(int(fbFileSet.PackageVariablesLength()), payloadLen, "package variables")
	if err != nil {
		return nil, err
	}
	packageVariables := make([]interp_domain.PackageVariableMetadata, 0, count)
	var fbVar interp_schema_gen.PackageVariableEntry
	for i := range count {
		if !fbFileSet.PackageVariables(&fbVar, i) {
			continue
		}
		var typeData *interp_domain.TypeDescriptorData
		if td := fbVar.TypeDescriptor(nil); td != nil {
			value, err := unpackTypeDescriptor(td, 0, payloadLen)
			if err != nil {
				return nil, fmt.Errorf("unpacking package-variable type descriptor for %s.%s: %w", mem.String(fbVar.PackagePath()), mem.String(fbVar.Name()), err)
			}
			typeData = &value
		}
		registerKind, err := validatedRegisterKind(int8(fbVar.RegisterKind()), "package variable")
		if err != nil {
			return nil, fmt.Errorf("unpacking package variable %s.%s: %w", mem.String(fbVar.PackagePath()), mem.String(fbVar.Name()), err)
		}
		packageVariables = append(packageVariables, interp_domain.PackageVariableMetadata{
			Name:         mem.String(fbVar.Name()),
			PackagePath:  mem.String(fbVar.PackagePath()),
			Type:         typeData,
			RegisterKind: registerKind,
			RelativeSlot: fbVar.RelativeSlot(),
		})
	}
	return packageVariables, nil
}

// unpackCompiledFunction reconstructs a CompiledFunction from its FlatBuffer
// representation. All constant pools are read, general constants and types are
// reconstructed via the SymbolRegistry, and child functions are unpacked recursively.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which is the serialised
// FlatBuffer function.
// Takes registry (*interp_domain.SymbolRegistry) which provides symbol and type lookups
// for runtime reconstruction.
// Takes depth (int) which is the current function-nesting recursion depth.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound every declared vector length.
//
// Returns *interp_domain.CompiledFunction which is the reconstructed compiled function.
// Returns error when reconstructing general constants, types, or child functions fails, a
// declared vector length exceeds the payload size, or a register kind is out of range.
func unpackCompiledFunction(ctx context.Context, fbFunction *interp_schema_gen.CompiledFunction, registry *interp_domain.SymbolRegistry, depth, payloadLen int) (*interp_domain.CompiledFunction, error) { //nolint:revive // dispatch table
	if depth > maxBytecodeFunctionNestingDepth {
		return nil, fmt.Errorf("%w: function nesting exceeded %d", errBytecodeRecursionDepthExceeded, maxBytecodeFunctionNestingDepth)
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("bytecode unpack cancelled: %w", err)
	}
	name := mem.String(fbFunction.Name())
	sourceFile := mem.String(fbFunction.SourceFile())
	isVariadic := fbFunction.IsVariadic()
	isPointerReceiver := fbFunction.IsPointerReceiver()

	var numRegisters [interp_domain.NumRegisterKinds]uint32
	for i := range fbFunction.RegisterCountsLength() {
		if i < interp_domain.NumRegisterKinds {
			numRegisters[i] = fbFunction.RegisterCounts(i)
		}
	}

	parameterKindCount, err := boundedCount(int(fbFunction.ParameterKindsLength()), payloadLen, "parameter kinds")
	if err != nil {
		return nil, err
	}
	parameterKinds := make([]interp_domain.RegisterKindValue, parameterKindCount)
	for i := range parameterKindCount {
		kind, err := validatedRegisterKind(int8(fbFunction.ParameterKinds(i)), "parameter")
		if err != nil {
			return nil, err
		}
		parameterKinds[i] = interp_domain.MakeRegisterKind(kind)
	}

	var parameterRegisters []uint8
	if declared := fbFunction.ParameterRegistersLength(); declared > 0 {
		parameterRegisterCount, err := boundedCount(int(declared), payloadLen, "parameter registers")
		if err != nil {
			return nil, err
		}
		parameterRegisters = make([]uint8, parameterRegisterCount)
		for i := range parameterRegisterCount {
			parameterRegisters[i] = fbFunction.ParameterRegisters(i)
		}
	}

	resultKindCount, err := boundedCount(int(fbFunction.ResultKindsLength()), payloadLen, "result kinds")
	if err != nil {
		return nil, err
	}
	resultKinds := make([]interp_domain.RegisterKindValue, resultKindCount)
	for i := range resultKindCount {
		kind, err := validatedRegisterKind(int8(fbFunction.ResultKinds(i)), "result")
		if err != nil {
			return nil, err
		}
		resultKinds[i] = interp_domain.MakeRegisterKind(kind)
	}

	bodyCount, err := boundedCount(int(fbFunction.BodyLength()), payloadLen, "function body")
	if err != nil {
		return nil, err
	}
	body := make([]interp_domain.InstructionValue, bodyCount)
	var fbInstruction interp_schema_gen.Instruction
	for i := range bodyCount {
		if fbFunction.Body(&fbInstruction, i) {
			body[i] = interp_domain.MakeInstruction(fbInstruction.Opcode(), fbInstruction.A(), fbInstruction.B(), fbInstruction.C())
		}
	}

	boolConstantCount, err := boundedCount(int(fbFunction.BoolConstantsLength()), payloadLen, "bool constants")
	if err != nil {
		return nil, err
	}
	boolConstants := make([]bool, boolConstantCount)
	for i := range boolConstantCount {
		boolConstants[i] = fbFunction.BoolConstants(i)
	}

	intConstantCount, err := boundedCount(int(fbFunction.IntConstantsLength()), payloadLen, "int constants")
	if err != nil {
		return nil, err
	}
	intConstants := make([]int64, intConstantCount)
	for i := range intConstantCount {
		intConstants[i] = fbFunction.IntConstants(i)
	}

	floatConstantCount, err := boundedCount(int(fbFunction.FloatConstantsLength()), payloadLen, "float constants")
	if err != nil {
		return nil, err
	}
	floatConstants := make([]float64, floatConstantCount)
	for i := range floatConstantCount {
		floatConstants[i] = fbFunction.FloatConstants(i)
	}

	uintConstantCount, err := boundedCount(int(fbFunction.UintConstantsLength()), payloadLen, "uint constants")
	if err != nil {
		return nil, err
	}
	uintConstants := make([]uint64, uintConstantCount)
	for i := range uintConstantCount {
		uintConstants[i] = fbFunction.UintConstants(i)
	}

	complexConstantCount, err := boundedCount(int(fbFunction.ComplexConstantsLength()), payloadLen, "complex constants")
	if err != nil {
		return nil, err
	}
	complexConstants := make([]complex128, complexConstantCount)
	var fbComplexValue interp_schema_gen.ComplexValue
	for i := range complexConstantCount {
		if fbFunction.ComplexConstants(&fbComplexValue, i) {
			complexConstants[i] = complex(fbComplexValue.Real(), fbComplexValue.Imaginary())
		}
	}

	stringConstantCount, err := boundedCount(int(fbFunction.StringConstantsLength()), payloadLen, "string constants")
	if err != nil {
		return nil, err
	}
	stringConstants := make([]string, stringConstantCount)
	for i := range stringConstantCount {
		stringConstants[i] = mem.String(fbFunction.StringConstants(i))
	}

	generalDescriptors, generalConstants, err := unpackGeneralConstants(fbFunction, registry, payloadLen)
	if err != nil {
		return nil, err
	}

	typeTableDescriptors, typeTable, err := unpackTypeTable(fbFunction, registry, payloadLen)
	if err != nil {
		return nil, err
	}

	typeTableInterfaceMethods, err := unpackInterfaceMethodSets(fbFunction, payloadLen)
	if err != nil {
		return nil, err
	}

	typeNames, err := unpackTypeNames(fbFunction, registry, payloadLen)
	if err != nil {
		return nil, err
	}

	callSites, err := unpackCallSites(fbFunction, payloadLen)
	if err != nil {
		return nil, err
	}
	upvalueDescriptors, err := unpackUpvalueDescriptors(fbFunction, payloadLen)
	if err != nil {
		return nil, err
	}
	structLayoutTable, err := unpackStructLayoutTable(fbFunction, payloadLen)
	if err != nil {
		return nil, err
	}

	functionCount, err := boundedCount(int(fbFunction.FunctionsLength()), payloadLen, "child functions")
	if err != nil {
		return nil, err
	}
	functions := make([]*interp_domain.CompiledFunction, functionCount)
	var fbChildFunction interp_schema_gen.CompiledFunction
	for i := range functionCount {
		if fbFunction.Functions(&fbChildFunction, i) {
			functions[i], err = unpackCompiledFunction(ctx, &fbChildFunction, registry, depth+1, payloadLen)
			if err != nil {
				return nil, fmt.Errorf("unpacking child function %d: %w", i, err)
			}
		}
	}

	namedResultLocations, err := unpackVarLocations(fbFunction.NamedResultLocationsLength(), func(location *interp_schema_gen.VarLocation, index int) bool {
		return fbFunction.NamedResultLocations(location, index)
	}, payloadLen)
	if err != nil {
		return nil, fmt.Errorf("unpacking named result locations: %w", err)
	}

	methodTableCount, err := boundedCount(int(fbFunction.MethodTableLength()), payloadLen, "method table")
	if err != nil {
		return nil, err
	}
	methodTable := make(map[string]uint16, methodTableCount)
	var fbMethodEntry interp_schema_gen.MethodTableEntry
	for i := range methodTableCount {
		if fbFunction.MethodTable(&fbMethodEntry, i) {
			methodTable[mem.String(fbMethodEntry.Name())] = fbMethodEntry.FunctionIndex()
		}
	}

	var variableInitFunction *interp_domain.CompiledFunction
	if fbVarInit := fbFunction.VariableInitFunction(nil); fbVarInit != nil {
		variableInitFunction, err = unpackCompiledFunction(ctx, fbVarInit, registry, depth+1, payloadLen)
		if err != nil {
			return nil, fmt.Errorf("unpacking variable init function: %w", err)
		}
	}

	return interp_domain.NewCompiledFunctionFromData(&interp_domain.CompiledFunctionData{
		Name:                       name,
		SourceFile:                 sourceFile,
		IsVariadic:                 isVariadic,
		IsPointerReceiver:          isPointerReceiver,
		NumRegisters:               numRegisters,
		ParamKinds:                 parameterKinds,
		ParamRegisters:             parameterRegisters,
		ResultKinds:                resultKinds,
		Body:                       body,
		BoolConstants:              boolConstants,
		IntConstants:               intConstants,
		FloatConstants:             floatConstants,
		UintConstants:              uintConstants,
		ComplexConstants:           complexConstants,
		StringConstants:            stringConstants,
		GeneralConstants:           generalConstants,
		GeneralConstantDescriptors: generalDescriptors,
		TypeTable:                  typeTable,
		TypeTableDescriptors:       typeTableDescriptors,
		TypeTableInterfaceMethods:  typeTableInterfaceMethods,
		TypeNames:                  typeNames,
		CallSites:                  callSites,
		UpvalueDescriptors:         upvalueDescriptors,
		Functions:                  functions,
		NamedResultLocations:       namedResultLocations,
		MethodTable:                methodTable,
		VariableInitFunction:       variableInitFunction,
		StructLayoutTable:          structLayoutTable,
	}), nil
}

// unpackInterfaceMethodSets reconstructs the per-type-table-entry interface method-name
// sets from their FlatBuffer representation.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which contains the serialised
// sets.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared set and method counts.
//
// Returns [][]string which holds the per-type-table-entry method name lists aligned with
// the type table, or nil when the function had no non-empty interface entries.
// Returns error when a declared count exceeds the payload size.
func unpackInterfaceMethodSets(fbFunction *interp_schema_gen.CompiledFunction, payloadLen int) ([][]string, error) {
	count, err := boundedCount(int(fbFunction.TypeTableInterfaceMethodsLength()), payloadLen, "interface method sets")
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	sets := make([][]string, count)
	var entry interp_schema_gen.InterfaceMethodSet
	for i := range count {
		if !fbFunction.TypeTableInterfaceMethods(&entry, i) {
			continue
		}
		methodCount, err := boundedCount(int(entry.MethodsLength()), payloadLen, "interface methods")
		if err != nil {
			return nil, err
		}
		if methodCount == 0 {
			continue
		}
		methods := make([]string, methodCount)
		for j := range methodCount {
			methods[j] = string(entry.Methods(j))
		}
		sets[i] = methods
	}
	return sets, nil
}

// unpackStructLayoutTable reconstructs the struct-field layout table from FlatBuffer-side
// StructFieldLayout entries. Each entry round trips byte-for-byte: offset, type index,
// path[0..3] + length, kind, register kind, flags.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which is the FlatBuffer-side
// compiled function.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared entry count.
//
// Returns []interp_domain.StructFieldLayoutData which holds one entry per table slot, or
// empty when the function had no fast-path field accesses.
// Returns error when the declared entry count exceeds the payload size.
func unpackStructLayoutTable(fbFunction *interp_schema_gen.CompiledFunction, payloadLen int) ([]interp_domain.StructFieldLayoutData, error) {
	count, err := boundedCount(int(fbFunction.StructLayoutTableLength()), payloadLen, "struct field layout")
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	layouts := make([]interp_domain.StructFieldLayoutData, count)
	var entry interp_schema_gen.StructFieldLayout
	for i := range count {
		if !fbFunction.StructLayoutTable(&entry, i) {
			continue
		}
		layouts[i] = interp_domain.StructFieldLayoutData{
			Offset:         entry.Offset(),
			TypeIndex:      entry.TypeIndex(),
			Path:           [structFieldLayoutMaxPathDepth]uint8{entry.Path0(), entry.Path1(), entry.Path2(), entry.Path3()},
			PathLength:     entry.PathLength(),
			Kind:           entry.Kind(),
			RegisterKind:   entry.RegisterKind(),
			Flags:          entry.Flags(),
			FieldTypeIndex: entry.FieldTypeIndex(),
		}
	}
	return layouts, nil
}

// unpackGeneralConstants reconstructs general constants from their FlatBuffer
// descriptors. Each descriptor is converted back to an internal descriptor and its
// runtime reflect.Value is reconstructed via the SymbolRegistry.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which contains the serialised
// general constant descriptors.
// Takes registry (*interp_domain.SymbolRegistry) which provides symbol lookups for
// runtime value reconstruction.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared descriptor count.
//
// Returns []interp_domain.GeneralConstantDescriptorInternal which holds the reconstructed
// internal descriptors.
// Returns []reflect.Value which holds the reconstructed runtime values.
// Returns error when any constant cannot be reconstructed or the declared count exceeds
// the payload size.
func unpackGeneralConstants( //nolint:dupl // mirrors unpackTypeTable
	fbFunction *interp_schema_gen.CompiledFunction,
	registry *interp_domain.SymbolRegistry,
	payloadLen int,
) ([]interp_domain.GeneralConstantDescriptorInternal, []reflect.Value, error) {
	count, err := boundedCount(int(fbFunction.GeneralConstantDescriptorsLength()), payloadLen, "general constants")
	if err != nil {
		return nil, nil, err
	}
	if count == 0 {
		return nil, nil, nil
	}
	descriptors := make([]interp_domain.GeneralConstantDescriptorInternal, count)
	values := make([]reflect.Value, count)
	var fbDescriptor interp_schema_gen.GeneralConstantDescriptor
	for i := range count {
		if !fbFunction.GeneralConstantDescriptors(&fbDescriptor, i) {
			continue
		}
		data, err := unpackGeneralConstantDescriptor(&fbDescriptor, payloadLen)
		if err != nil {
			return nil, nil, fmt.Errorf("unpacking general constant descriptor %d: %w", i, err)
		}
		descriptors[i] = interp_domain.ImportGeneralConstantDescriptor(data)
		value, err := interp_domain.ReconstructGeneralConstant(data, registry)
		if err != nil {
			return nil, nil, fmt.Errorf("reconstructing general constant %d: %w", i, err)
		}
		values[i] = value
	}
	return descriptors, values, nil
}

// unpackTypeTable reconstructs the type table from its FlatBuffer descriptors. Each
// descriptor is converted back to an internal descriptor and its runtime reflect.Type is
// reconstructed via the SymbolRegistry.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which contains the serialised
// type table descriptors.
// Takes registry (*interp_domain.SymbolRegistry) which provides named type lookups for
// runtime reconstruction.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared descriptor count.
//
// Returns []interp_domain.TypeDescriptorInternal which holds the reconstructed internal
// descriptors.
// Returns []reflect.Type which holds the reconstructed runtime types.
// Returns error when any type cannot be reconstructed or the declared count exceeds the
// payload size.
func unpackTypeTable( //nolint:dupl // mirrors unpackGeneralConstants
	fbFunction *interp_schema_gen.CompiledFunction,
	registry *interp_domain.SymbolRegistry,
	payloadLen int,
) ([]interp_domain.TypeDescriptorInternal, []reflect.Type, error) {
	count, err := boundedCount(int(fbFunction.TypeTableDescriptorsLength()), payloadLen, "type table")
	if err != nil {
		return nil, nil, err
	}
	if count == 0 {
		return nil, nil, nil
	}
	descriptors := make([]interp_domain.TypeDescriptorInternal, count)
	types := make([]reflect.Type, count)
	var fbDescriptor interp_schema_gen.TypeDescriptor
	for i := range count {
		if !fbFunction.TypeTableDescriptors(&fbDescriptor, i) {
			continue
		}
		data, err := unpackTypeDescriptor(&fbDescriptor, 0, payloadLen)
		if err != nil {
			return nil, nil, fmt.Errorf("unpacking type descriptor %d: %w", i, err)
		}
		descriptors[i] = interp_domain.ImportTypeDescriptor(data)
		reconstructedType, err := interp_domain.DescriptorToReflectType(data, registry)
		if err != nil {
			return nil, nil, fmt.Errorf("reconstructing type %d: %w", i, err)
		}
		types[i] = reconstructedType
	}
	return descriptors, types, nil
}

// unpackTypeNames reconstructs the type names map from FlatBuffer entries. Each entry's
// type descriptor is resolved to a reflect.Type and paired with its string name.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which contains the serialised
// type name entries.
// Takes registry (*interp_domain.SymbolRegistry) which provides named type lookups for
// runtime reconstruction.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared entry count.
//
// Returns map[reflect.Type]string which maps runtime types to their string names.
// Returns error when a serialised type descriptor exceeds the recursion depth bound or
// the declared entry count exceeds the payload size.
func unpackTypeNames(
	fbFunction *interp_schema_gen.CompiledFunction,
	registry *interp_domain.SymbolRegistry,
	payloadLen int,
) (map[reflect.Type]string, error) {
	count, err := boundedCount(int(fbFunction.TypeNamesLength()), payloadLen, "type names")
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	result := make(map[reflect.Type]string, count)
	var fbEntry interp_schema_gen.TypeNameEntry
	for i := range count {
		if !fbFunction.TypeNames(&fbEntry, i) {
			continue
		}
		name := mem.String(fbEntry.Name())
		fbTypeDescriptor := fbEntry.TypeDescriptor(nil)
		if fbTypeDescriptor == nil {
			continue
		}
		data, err := unpackTypeDescriptor(fbTypeDescriptor, 0, payloadLen)
		if err != nil {
			return nil, fmt.Errorf("unpacking type name descriptor %d: %w", i, err)
		}
		reconstructedType, err := interp_domain.DescriptorToReflectType(data, registry)
		if err == nil {
			result[reconstructedType] = name
		}
	}
	return result, nil
}

// unpackCallSites reconstructs call sites from their FlatBuffer representation. Only
// static metadata is deserialised; runtime caches are initialised to their zero values.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which contains the serialised
// call sites.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared call-site and argument counts.
//
// Returns []interp_domain.CallSiteInternal which holds the reconstructed call sites.
// Returns error when a declared count exceeds the payload size or a register kind is out
// of range.
func unpackCallSites(fbFunction *interp_schema_gen.CompiledFunction, payloadLen int) ([]interp_domain.CallSiteInternal, error) {
	count, err := boundedCount(int(fbFunction.CallSitesLength()), payloadLen, "call sites")
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	sites := make([]interp_domain.CallSiteInternal, count)
	var fbCallSite interp_schema_gen.CallSite
	for i := range count {
		if !fbFunction.CallSites(&fbCallSite, i) {
			continue
		}
		arguments, err := unpackVarLocationData(fbCallSite.ArgumentsLength(), func(location *interp_schema_gen.VarLocation, index int) bool {
			return fbCallSite.Arguments(location, index)
		}, payloadLen)
		if err != nil {
			return nil, fmt.Errorf("unpacking call site %d arguments: %w", i, err)
		}
		returns, err := unpackVarLocationData(fbCallSite.ReturnsLength(), func(location *interp_schema_gen.VarLocation, index int) bool {
			return fbCallSite.Returns(location, index)
		}, payloadLen)
		if err != nil {
			return nil, fmt.Errorf("unpacking call site %d returns: %w", i, err)
		}
		data := interp_domain.CallSiteData{
			FuncIndex:              fbCallSite.FunctionIndex(),
			ClosureRegister:        fbCallSite.ClosureRegister(),
			NativeRegister:         fbCallSite.NativeRegister(),
			IsClosure:              fbCallSite.IsClosure(),
			IsNative:               fbCallSite.IsNative(),
			IsMethod:               fbCallSite.IsMethod(),
			MethodReceiverRegister: fbCallSite.MethodReceiverRegister(),
			IsEllipsisSpread:       fbCallSite.IsEllipsisSpread(),
			Arguments:              arguments,
			Returns:                returns,
		}
		sites[i] = interp_domain.MakeCallSite(data)
	}
	return sites, nil
}

// unpackUpvalueDescriptors reconstructs upvalue descriptors from their FlatBuffer
// representation.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which contains the serialised
// upvalue descriptors.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared descriptor count.
//
// Returns []interp_domain.UpvalueDescriptor which holds the reconstructed upvalue
// descriptors.
// Returns error when the declared count exceeds the payload size or a register kind is
// out of range.
func unpackUpvalueDescriptors(fbFunction *interp_schema_gen.CompiledFunction, payloadLen int) ([]interp_domain.UpvalueDescriptor, error) {
	count, err := boundedCount(int(fbFunction.UpvalueDescriptorsLength()), payloadLen, "upvalue descriptors")
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	descriptors := make([]interp_domain.UpvalueDescriptor, count)
	var fbDescriptor interp_schema_gen.UpvalueDescriptor
	for i := range count {
		if !fbFunction.UpvalueDescriptors(&fbDescriptor, i) {
			continue
		}
		kind, err := validatedRegisterKind(int8(fbDescriptor.Kind()), "upvalue")
		if err != nil {
			return nil, err
		}
		originalKind, err := validatedRegisterKind(int8(fbDescriptor.OriginalKind()), "upvalue original")
		if err != nil {
			return nil, err
		}
		descriptors[i] = interp_domain.MakeUpvalueDescriptor(interp_domain.UpvalueDescriptorData{
			Index:        fbDescriptor.Index(),
			Kind:         kind,
			OriginalKind: originalKind,
			IsLocal:      fbDescriptor.IsLocal(),
			IsIndirect:   fbDescriptor.IsIndirect(),
		})
	}
	return descriptors, nil
}

// unpackVarLocations reconstructs variable locations as internal varLocation values from
// a FlatBuffer vector accessed via the getter function.
//
// Takes length (int) which is the number of variable locations in the vector.
// Takes getter (func) which retrieves each VarLocation by index from the FlatBuffer.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared length.
//
// Returns []interp_domain.VarLocationInternal which holds the reconstructed variable
// locations.
// Returns error when the declared length exceeds the payload size or a register kind is
// out of range.
func unpackVarLocations(length int, getter func(*interp_schema_gen.VarLocation, int) bool, payloadLen int) ([]interp_domain.VarLocationInternal, error) {
	count, err := boundedCount(length, payloadLen, "variable locations")
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	locations := make([]interp_domain.VarLocationInternal, count)
	var fbLocation interp_schema_gen.VarLocation
	for i := range count {
		if !getter(&fbLocation, i) {
			continue
		}
		kind, err := validatedRegisterKind(int8(fbLocation.Kind()), "variable location")
		if err != nil {
			return nil, err
		}
		originalKind, err := validatedRegisterKind(int8(fbLocation.OriginalKind()), "variable location original")
		if err != nil {
			return nil, err
		}
		locations[i] = interp_domain.MakeVarLocation(interp_domain.VarLocationData{
			UpvalueIndex: fbLocation.UpvalueIndex(),
			Register:     fbLocation.Register(),
			Kind:         kind,
			IsUpvalue:    fbLocation.IsUpvalue(),
			IsIndirect:   fbLocation.IsIndirect(),
			OriginalKind: originalKind,
			IsSpilled:    fbLocation.IsSpilled(),
			SpillSlot:    fbLocation.SpillSlot(),
		})
	}
	return locations, nil
}

// unpackVarLocationData reconstructs variable locations as serialisation-safe
// VarLocationData values from a FlatBuffer vector accessed via the getter function.
//
// Takes length (int) which is the number of variable locations in the vector.
// Takes getter (func) which retrieves each VarLocation by index from the FlatBuffer.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared length.
//
// Returns []interp_domain.VarLocationData which holds the reconstructed variable location
// data.
// Returns error when the declared length exceeds the payload size or a register kind is
// out of range.
func unpackVarLocationData(length int, getter func(*interp_schema_gen.VarLocation, int) bool, payloadLen int) ([]interp_domain.VarLocationData, error) {
	count, err := boundedCount(length, payloadLen, "variable location data")
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	locations := make([]interp_domain.VarLocationData, count)
	var fbLocation interp_schema_gen.VarLocation
	for i := range count {
		if !getter(&fbLocation, i) {
			continue
		}
		kind, err := validatedRegisterKind(int8(fbLocation.Kind()), "variable location data")
		if err != nil {
			return nil, err
		}
		originalKind, err := validatedRegisterKind(int8(fbLocation.OriginalKind()), "variable location data original")
		if err != nil {
			return nil, err
		}
		locations[i] = interp_domain.VarLocationData{
			UpvalueIndex: fbLocation.UpvalueIndex(),
			Register:     fbLocation.Register(),
			Kind:         kind,
			IsUpvalue:    fbLocation.IsUpvalue(),
			IsIndirect:   fbLocation.IsIndirect(),
			OriginalKind: originalKind,
			IsSpilled:    fbLocation.IsSpilled(),
			SpillSlot:    fbLocation.SpillSlot(),
		}
	}
	return locations, nil
}

// unpackGeneralConstantDescriptor reconstructs a general constant descriptor from its
// FlatBuffer representation.
//
// Takes fbDescriptor (*interp_schema_gen.GeneralConstantDescriptor) which is the
// serialised FlatBuffer descriptor.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the embedded type descriptor's vector lengths.
//
// Returns interp_domain.GeneralConstantDescriptorData which holds the reconstructed
// descriptor data.
// Returns error when the embedded type descriptor exceeds the recursion depth bound or
// declares a vector length larger than the payload size.
func unpackGeneralConstantDescriptor(fbDescriptor *interp_schema_gen.GeneralConstantDescriptor, payloadLen int) (interp_domain.GeneralConstantDescriptorData, error) {
	data := interp_domain.GeneralConstantDescriptorData{
		Kind:        safeconv.MustInt8ToUint8(int8(fbDescriptor.Kind())),
		PackagePath: mem.String(fbDescriptor.PackagePath()),
		SymbolName:  mem.String(fbDescriptor.SymbolName()),
	}
	if fbTypeDescriptor := fbDescriptor.TypeDescriptor(nil); fbTypeDescriptor != nil {
		typeDesc, err := unpackTypeDescriptor(fbTypeDescriptor, 0, payloadLen)
		if err != nil {
			return interp_domain.GeneralConstantDescriptorData{}, err
		}
		data.TypeDesc = typeDesc
	}
	return data, nil
}

// unpackTypeDescriptor recursively reconstructs a type descriptor from FlatBuffer form.
//
// Recursive fields (elem, key, value, fields, params, results) are unpacked depth-first.
// The depth parameter caps recursion to defend against tampered or corrupted on-disk
// payloads.
//
// Takes fbDescriptor (*interp_schema_gen.TypeDescriptor) which is the serialised
// FlatBuffer type descriptor.
// Takes depth (int) which is the current recursion depth.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the declared field, parameter, and result counts.
//
// Returns interp_domain.TypeDescriptorData which holds the reconstructed type descriptor
// data.
// Returns error when the recursion depth bound is exceeded or a declared count exceeds
// the payload size.
func unpackTypeDescriptor(fbDescriptor *interp_schema_gen.TypeDescriptor, depth, payloadLen int) (interp_domain.TypeDescriptorData, error) { //nolint:revive // dispatch table
	if depth > maxBytecodeTypeDescriptorDepth {
		return interp_domain.TypeDescriptorData{}, fmt.Errorf("%w: type descriptor depth exceeded %d", errBytecodeRecursionDepthExceeded, maxBytecodeTypeDescriptorDepth)
	}
	data := interp_domain.TypeDescriptorData{
		Kind:        safeconv.MustInt8ToUint8(int8(fbDescriptor.Kind())),
		PackagePath: mem.String(fbDescriptor.PackagePath()),
		Name:        mem.String(fbDescriptor.Name()),
		BasicKind:   fbDescriptor.BasicKind(),
		Length:      fbDescriptor.Length(),
		Dir:         fbDescriptor.Direction(),
		IsVariadic:  fbDescriptor.IsVariadic(),
	}
	if fbElem := fbDescriptor.Element(nil); fbElem != nil {
		element, err := unpackTypeDescriptor(fbElem, depth+1, payloadLen)
		if err != nil {
			return interp_domain.TypeDescriptorData{}, err
		}
		data.Elem = new(element)
	}
	if fbKey := fbDescriptor.Key(nil); fbKey != nil {
		key, err := unpackTypeDescriptor(fbKey, depth+1, payloadLen)
		if err != nil {
			return interp_domain.TypeDescriptorData{}, err
		}
		data.Key = new(key)
	}
	if fbValue := fbDescriptor.Value(nil); fbValue != nil {
		value, err := unpackTypeDescriptor(fbValue, depth+1, payloadLen)
		if err != nil {
			return interp_domain.TypeDescriptorData{}, err
		}
		data.Value = new(value)
	}
	if declared := fbDescriptor.FieldsLength(); declared > 0 {
		fieldCount, err := boundedCount(int(declared), payloadLen, "type descriptor fields")
		if err != nil {
			return interp_domain.TypeDescriptorData{}, err
		}
		data.Fields = make([]interp_domain.TypeDescFieldData, fieldCount)
		var fbField interp_schema_gen.TypeDescField
		for i := range fieldCount {
			if !fbDescriptor.Fields(&fbField, i) {
				continue
			}
			fieldData, err := unpackTypeDescFieldData(&fbField, depth, payloadLen)
			if err != nil {
				return interp_domain.TypeDescriptorData{}, err
			}
			data.Fields[i] = fieldData
		}
	}
	if declared := fbDescriptor.ParamsLength(); declared > 0 {
		paramCount, err := boundedCount(int(declared), payloadLen, "type descriptor params")
		if err != nil {
			return interp_domain.TypeDescriptorData{}, err
		}
		data.Params = make([]interp_domain.TypeDescriptorData, paramCount)
		var fbParameter interp_schema_gen.TypeDescriptor
		for i := range paramCount {
			if fbDescriptor.Params(&fbParameter, i) {
				parameter, err := unpackTypeDescriptor(&fbParameter, depth+1, payloadLen)
				if err != nil {
					return interp_domain.TypeDescriptorData{}, err
				}
				data.Params[i] = parameter
			}
		}
	}
	if declared := fbDescriptor.ResultsLength(); declared > 0 {
		resultCount, err := boundedCount(int(declared), payloadLen, "type descriptor results")
		if err != nil {
			return interp_domain.TypeDescriptorData{}, err
		}
		data.Results = make([]interp_domain.TypeDescriptorData, resultCount)
		var fbResultType interp_schema_gen.TypeDescriptor
		for i := range resultCount {
			if fbDescriptor.Results(&fbResultType, i) {
				result, err := unpackTypeDescriptor(&fbResultType, depth+1, payloadLen)
				if err != nil {
					return interp_domain.TypeDescriptorData{}, err
				}
				data.Results[i] = result
			}
		}
	}
	return data, nil
}

// unpackTypeDescFieldData materialises a single TypeDescFieldData entry from a FlatBuffer
// field descriptor, recursively unpacking its type descriptor when present.
//
// Takes fbField (*interp_schema_gen.TypeDescField) which is the FlatBuffer field
// descriptor to materialise.
// Takes depth (int) which is the current recursion depth used to bound nested
// descriptors.
// Takes payloadLen (int) which is the total untrusted payload size in bytes, used to
// bound the nested descriptor's vector lengths.
//
// Returns the materialised TypeDescFieldData entry, and an error that is non-nil when
// descriptor unpacking fails.
func unpackTypeDescFieldData(fbField *interp_schema_gen.TypeDescField, depth, payloadLen int) (interp_domain.TypeDescFieldData, error) {
	fieldData := interp_domain.TypeDescFieldData{
		Name:        mem.String(fbField.Name()),
		Tag:         mem.String(fbField.Tag()),
		PackagePath: mem.String(fbField.PackagePath()),
	}
	fbFieldType := fbField.TypeDescriptor(nil)
	if fbFieldType == nil {
		return fieldData, nil
	}
	fieldType, err := unpackTypeDescriptor(fbFieldType, depth+1, payloadLen)
	if err != nil {
		return interp_domain.TypeDescFieldData{}, err
	}
	fieldData.Typ = fieldType
	return fieldData, nil
}
