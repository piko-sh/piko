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

//go:build !js || !wasm

package interp_adapters

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"piko.sh/piko/wdk/safeconv"

	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/interp/interp_schema"
	"piko.sh/piko/internal/interp/interp_schema/interp_schema_gen"
	"piko.sh/piko/internal/mem"
)

// LoadCompiledFileSet reads and deserialises a previously saved
// compiled file set, reconstructing runtime types and values via
// the provided SymbolRegistry.
//
// Takes key (string) which identifies the cached bytecode to load.
// Takes registry (*interp_domain.SymbolRegistry) which provides
// symbol and type lookups for runtime reconstruction.
//
// Returns *interp_domain.CompiledFileSet which is the reconstructed
// compiled file set.
// Returns error when the sandbox is nil, the key is empty, the
// cache is missing, the schema version has changed, or
// reconstruction fails.
func (bytecodeStore *BytecodeStore) LoadCompiledFileSet(_ context.Context, key string, registry *interp_domain.SymbolRegistry) (*interp_domain.CompiledFileSet, error) {
	if bytecodeStore.sandbox == nil || key == "" {
		return nil, errors.New("bytecode store requires a sandbox and key")
	}

	fileName := fmt.Sprintf("bytecode-%s.bin", key)
	data, err := bytecodeStore.sandbox.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("bytecode cache miss or read error for key %s: %w", key, err)
	}

	payload, err := interp_schema.Unpack(data)
	if err != nil {
		_ = bytecodeStore.sandbox.Remove(fileName)
		if errors.Is(err, fbs.ErrSchemaVersionMismatch) {
			return nil, fmt.Errorf("bytecode schema version mismatch for key %s, invalidated", key)
		}
		return nil, fmt.Errorf("failed to unpack versioned bytecode for key %s: %w", key, err)
	}

	fbFileSet := interp_schema_gen.GetRootAsCompiledFileSet(payload, 0)
	if fbFileSet == nil {
		_ = bytecodeStore.sandbox.Remove(fileName)
		return nil, fmt.Errorf("failed to parse corrupt bytecode file for key %s", key)
	}

	return unpackCompiledFileSet(fbFileSet, registry)
}

// unpackCompiledFileSet reconstructs a CompiledFileSet from its
// FlatBuffer representation.
//
// Takes fbFileSet (*interp_schema_gen.CompiledFileSet) which is the
// serialised FlatBuffer file set.
// Takes registry (*interp_domain.SymbolRegistry) which provides
// symbol and type lookups for runtime reconstruction.
//
// Returns *interp_domain.CompiledFileSet which is the reconstructed
// compiled file set.
// Returns error when unpacking any function fails.
func unpackCompiledFileSet(fbFileSet *interp_schema_gen.CompiledFileSet, registry *interp_domain.SymbolRegistry) (*interp_domain.CompiledFileSet, error) {
	var root *interp_domain.CompiledFunction
	if fbRoot := fbFileSet.Root(nil); fbRoot != nil {
		var err error
		root, err = unpackCompiledFunction(fbRoot, registry)
		if err != nil {
			return nil, fmt.Errorf("unpacking root function: %w", err)
		}
	}

	var variableInitFunction *interp_domain.CompiledFunction
	if fbVarInit := fbFileSet.VariableInitFunction(nil); fbVarInit != nil {
		var err error
		variableInitFunction, err = unpackCompiledFunction(fbVarInit, registry)
		if err != nil {
			return nil, fmt.Errorf("unpacking variable init function: %w", err)
		}
	}

	entrypoints := make(map[string]uint16, fbFileSet.EntrypointsLength())
	var fbEntrypoint interp_schema_gen.EntrypointEntry
	for i := range fbFileSet.EntrypointsLength() {
		if fbFileSet.Entrypoints(&fbEntrypoint, i) {
			entrypoints[mem.String(fbEntrypoint.Name())] = fbEntrypoint.FunctionIndex()
		}
	}

	initFunctionIndices := make([]uint16, fbFileSet.InitialisationFunctionsLength())
	for i := range fbFileSet.InitialisationFunctionsLength() {
		initFunctionIndices[i] = fbFileSet.InitialisationFunctions(i)
	}

	return interp_domain.NewCompiledFileSetFromData(root, variableInitFunction, entrypoints, initFunctionIndices), nil
}

// unpackCompiledFunction reconstructs a CompiledFunction from its
// FlatBuffer representation. All constant pools are read, general
// constants and types are reconstructed via the SymbolRegistry,
// and child functions are unpacked recursively.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which is
// the serialised FlatBuffer function.
// Takes registry (*interp_domain.SymbolRegistry) which provides
// symbol and type lookups for runtime reconstruction.
//
// Returns *interp_domain.CompiledFunction which is the
// reconstructed compiled function.
// Returns error when reconstructing general constants, types, or
// child functions fails.
func unpackCompiledFunction(fbFunction *interp_schema_gen.CompiledFunction, registry *interp_domain.SymbolRegistry) (*interp_domain.CompiledFunction, error) { //nolint:revive // dispatch table
	name := mem.String(fbFunction.Name())
	sourceFile := mem.String(fbFunction.SourceFile())
	isVariadic := fbFunction.IsVariadic()

	var numRegisters [interp_domain.NumRegisterKinds]uint32
	for i := range fbFunction.RegisterCountsLength() {
		if i < interp_domain.NumRegisterKinds {
			numRegisters[i] = fbFunction.RegisterCounts(i)
		}
	}

	paramKinds := make([]interp_domain.RegisterKindValue, fbFunction.ParameterKindsLength())
	for i := range fbFunction.ParameterKindsLength() {
		paramKinds[i] = interp_domain.MakeRegisterKind(safeconv.MustInt8ToUint8(int8(fbFunction.ParameterKinds(i))))
	}

	resultKinds := make([]interp_domain.RegisterKindValue, fbFunction.ResultKindsLength())
	for i := range fbFunction.ResultKindsLength() {
		resultKinds[i] = interp_domain.MakeRegisterKind(safeconv.MustInt8ToUint8(int8(fbFunction.ResultKinds(i))))
	}

	body := make([]interp_domain.InstructionValue, fbFunction.BodyLength())
	var fbInstruction interp_schema_gen.Instruction
	for i := range fbFunction.BodyLength() {
		if fbFunction.Body(&fbInstruction, i) {
			body[i] = interp_domain.MakeInstruction(fbInstruction.Opcode(), fbInstruction.A(), fbInstruction.B(), fbInstruction.C())
		}
	}

	boolConstants := make([]bool, fbFunction.BoolConstantsLength())
	for i := range fbFunction.BoolConstantsLength() {
		boolConstants[i] = fbFunction.BoolConstants(i)
	}

	intConstants := make([]int64, fbFunction.IntConstantsLength())
	for i := range fbFunction.IntConstantsLength() {
		intConstants[i] = fbFunction.IntConstants(i)
	}

	floatConstants := make([]float64, fbFunction.FloatConstantsLength())
	for i := range fbFunction.FloatConstantsLength() {
		floatConstants[i] = fbFunction.FloatConstants(i)
	}

	uintConstants := make([]uint64, fbFunction.UintConstantsLength())
	for i := range fbFunction.UintConstantsLength() {
		uintConstants[i] = fbFunction.UintConstants(i)
	}

	complexConstants := make([]complex128, fbFunction.ComplexConstantsLength())
	var fbComplexValue interp_schema_gen.ComplexValue
	for i := range fbFunction.ComplexConstantsLength() {
		if fbFunction.ComplexConstants(&fbComplexValue, i) {
			complexConstants[i] = complex(fbComplexValue.Real(), fbComplexValue.Imaginary())
		}
	}

	stringConstants := make([]string, fbFunction.StringConstantsLength())
	for i := range fbFunction.StringConstantsLength() {
		stringConstants[i] = mem.String(fbFunction.StringConstants(i))
	}

	generalDescriptors, generalConstants, err := unpackGeneralConstants(fbFunction, registry)
	if err != nil {
		return nil, err
	}

	typeTableDescriptors, typeTable, err := unpackTypeTable(fbFunction, registry)
	if err != nil {
		return nil, err
	}

	typeNames := unpackTypeNames(fbFunction, registry, typeTable, typeTableDescriptors)

	callSites := unpackCallSites(fbFunction)
	upvalueDescriptors := unpackUpvalueDescriptors(fbFunction)

	functions := make([]*interp_domain.CompiledFunction, fbFunction.FunctionsLength())
	var fbChildFunction interp_schema_gen.CompiledFunction
	for i := range fbFunction.FunctionsLength() {
		if fbFunction.Functions(&fbChildFunction, i) {
			functions[i], err = unpackCompiledFunction(&fbChildFunction, registry)
			if err != nil {
				return nil, fmt.Errorf("unpacking child function %d: %w", i, err)
			}
		}
	}

	namedResultLocations := unpackVarLocations(fbFunction.NamedResultLocationsLength(), func(location *interp_schema_gen.VarLocation, index int) bool {
		return fbFunction.NamedResultLocations(location, index)
	})

	methodTable := make(map[string]uint16, fbFunction.MethodTableLength())
	var fbMethodEntry interp_schema_gen.MethodTableEntry
	for i := range fbFunction.MethodTableLength() {
		if fbFunction.MethodTable(&fbMethodEntry, i) {
			methodTable[mem.String(fbMethodEntry.Name())] = fbMethodEntry.FunctionIndex()
		}
	}

	var variableInitFunction *interp_domain.CompiledFunction
	if fbVarInit := fbFunction.VariableInitFunction(nil); fbVarInit != nil {
		variableInitFunction, err = unpackCompiledFunction(fbVarInit, registry)
		if err != nil {
			return nil, fmt.Errorf("unpacking variable init function: %w", err)
		}
	}

	return interp_domain.NewCompiledFunctionFromData(&interp_domain.CompiledFunctionData{
		Name:                       name,
		SourceFile:                 sourceFile,
		IsVariadic:                 isVariadic,
		NumRegisters:               numRegisters,
		ParamKinds:                 paramKinds,
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
		TypeNames:                  typeNames,
		CallSites:                  callSites,
		UpvalueDescriptors:         upvalueDescriptors,
		Functions:                  functions,
		NamedResultLocs:            namedResultLocations,
		MethodTable:                methodTable,
		VariableInitFunction:       variableInitFunction,
	}), nil
}

// unpackGeneralConstants reconstructs general constants from their
// FlatBuffer descriptors. Each descriptor is converted back to an
// internal descriptor and its runtime reflect.Value is
// reconstructed via the SymbolRegistry.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which
// contains the serialised general constant descriptors.
// Takes registry (*interp_domain.SymbolRegistry) which provides
// symbol lookups for runtime value reconstruction.
//
// Returns []interp_domain.GeneralConstantDescriptorInternal which
// holds the reconstructed internal descriptors.
// Returns []reflect.Value which holds the reconstructed runtime
// values.
// Returns error when any constant cannot be reconstructed.
func unpackGeneralConstants( //nolint:dupl // mirrors unpackTypeTable
	fbFunction *interp_schema_gen.CompiledFunction,
	registry *interp_domain.SymbolRegistry,
) ([]interp_domain.GeneralConstantDescriptorInternal, []reflect.Value, error) {
	count := fbFunction.GeneralConstantDescriptorsLength()
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
		data := unpackGeneralConstantDescriptor(&fbDescriptor)
		descriptors[i] = interp_domain.ImportGeneralConstantDescriptor(data)
		value, err := interp_domain.ReconstructGeneralConstant(data, registry)
		if err != nil {
			return nil, nil, fmt.Errorf("reconstructing general constant %d: %w", i, err)
		}
		values[i] = value
	}
	return descriptors, values, nil
}

// unpackTypeTable reconstructs the type table from its FlatBuffer
// descriptors. Each descriptor is converted back to an internal
// descriptor and its runtime reflect.Type is reconstructed via the
// SymbolRegistry.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which
// contains the serialised type table descriptors.
// Takes registry (*interp_domain.SymbolRegistry) which provides
// named type lookups for runtime reconstruction.
//
// Returns []interp_domain.TypeDescriptorInternal which holds the
// reconstructed internal descriptors.
// Returns []reflect.Type which holds the reconstructed runtime
// types.
// Returns error when any type cannot be reconstructed.
func unpackTypeTable( //nolint:dupl // mirrors unpackGeneralConstants
	fbFunction *interp_schema_gen.CompiledFunction,
	registry *interp_domain.SymbolRegistry,
) ([]interp_domain.TypeDescriptorInternal, []reflect.Type, error) {
	count := fbFunction.TypeTableDescriptorsLength()
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
		data := unpackTypeDescriptor(&fbDescriptor)
		descriptors[i] = interp_domain.ImportTypeDescriptor(data)
		reconstructedType, err := interp_domain.DescriptorToReflectType(data, registry)
		if err != nil {
			return nil, nil, fmt.Errorf("reconstructing type %d: %w", i, err)
		}
		types[i] = reconstructedType
	}
	return descriptors, types, nil
}

// unpackTypeNames reconstructs the type names map from FlatBuffer
// entries. Each entry's type descriptor is resolved to a
// reflect.Type and paired with its string name.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which
// contains the serialised type name entries.
// Takes registry (*interp_domain.SymbolRegistry) which provides
// named type lookups for runtime reconstruction.
// Takes typeTable ([]reflect.Type) which is the reconstructed type
// table (unused, reserved for future optimisation).
// Takes typeTableDescriptors
// ([]interp_domain.TypeDescriptorInternal) which is the
// reconstructed type table descriptors (unused, reserved for
// future optimisation).
//
// Returns map[reflect.Type]string which maps runtime types to their
// string names.
func unpackTypeNames(
	fbFunction *interp_schema_gen.CompiledFunction,
	registry *interp_domain.SymbolRegistry,
	typeTable []reflect.Type,
	typeTableDescriptors []interp_domain.TypeDescriptorInternal,
) map[reflect.Type]string {
	count := fbFunction.TypeNamesLength()
	if count == 0 {
		return nil
	}
	result := make(map[reflect.Type]string, count)
	var fbEntry interp_schema_gen.TypeNameEntry
	for i := range count {
		if !fbFunction.TypeNames(&fbEntry, i) {
			continue
		}
		name := mem.String(fbEntry.Name())
		if fbTypeDescriptor := fbEntry.TypeDescriptor(nil); fbTypeDescriptor != nil {
			data := unpackTypeDescriptor(fbTypeDescriptor)
			reconstructedType, err := interp_domain.DescriptorToReflectType(data, registry)
			if err == nil {
				result[reconstructedType] = name
			}
		}
	}
	_ = typeTable
	_ = typeTableDescriptors
	return result
}

// unpackCallSites reconstructs call sites from their FlatBuffer
// representation. Only static metadata is deserialised; runtime
// caches are initialised to their zero values.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which
// contains the serialised call sites.
//
// Returns []interp_domain.CallSiteInternal which holds the
// reconstructed call sites.
func unpackCallSites(fbFunction *interp_schema_gen.CompiledFunction) []interp_domain.CallSiteInternal {
	count := fbFunction.CallSitesLength()
	if count == 0 {
		return nil
	}
	sites := make([]interp_domain.CallSiteInternal, count)
	var fbCallSite interp_schema_gen.CallSite
	for i := range count {
		if !fbFunction.CallSites(&fbCallSite, i) {
			continue
		}
		data := interp_domain.CallSiteData{
			FuncIndex:       fbCallSite.FunctionIndex(),
			ClosureRegister: fbCallSite.ClosureRegister(),
			NativeRegister:  fbCallSite.NativeRegister(),
			IsClosure:       fbCallSite.IsClosure(),
			IsNative:        fbCallSite.IsNative(),
			IsMethod:        fbCallSite.IsMethod(),
			MethodRecvReg:   fbCallSite.MethodReceiverRegister(),
			Arguments: unpackVarLocationData(fbCallSite.ArgumentsLength(), func(location *interp_schema_gen.VarLocation, index int) bool {
				return fbCallSite.Arguments(location, index)
			}),
			Returns: unpackVarLocationData(fbCallSite.ReturnsLength(), func(location *interp_schema_gen.VarLocation, index int) bool {
				return fbCallSite.Returns(location, index)
			}),
		}
		sites[i] = interp_domain.MakeCallSite(data)
	}
	return sites
}

// unpackUpvalueDescriptors reconstructs upvalue descriptors from
// their FlatBuffer representation.
//
// Takes fbFunction (*interp_schema_gen.CompiledFunction) which
// contains the serialised upvalue descriptors.
//
// Returns []interp_domain.UpvalueDescriptor which holds the
// reconstructed upvalue descriptors.
func unpackUpvalueDescriptors(fbFunction *interp_schema_gen.CompiledFunction) []interp_domain.UpvalueDescriptor {
	count := fbFunction.UpvalueDescriptorsLength()
	if count == 0 {
		return nil
	}
	descriptors := make([]interp_domain.UpvalueDescriptor, count)
	var fbDescriptor interp_schema_gen.UpvalueDescriptor
	for i := range count {
		if fbFunction.UpvalueDescriptors(&fbDescriptor, i) {
			descriptors[i] = interp_domain.MakeUpvalueDescriptor(interp_domain.UpvalueDescriptorData{
				Index:   fbDescriptor.Index(),
				Kind:    safeconv.MustInt8ToUint8(int8(fbDescriptor.Kind())),
				IsLocal: fbDescriptor.IsLocal(),
			})
		}
	}
	return descriptors
}

// unpackVarLocations reconstructs variable locations as internal
// varLocation values from a FlatBuffer vector accessed via the
// getter function.
//
// Takes length (int) which is the number of variable locations in
// the vector.
// Takes getter (func) which retrieves each VarLocation by index
// from the FlatBuffer.
//
// Returns []interp_domain.VarLocationInternal which holds the
// reconstructed variable locations.
func unpackVarLocations(length int, getter func(*interp_schema_gen.VarLocation, int) bool) []interp_domain.VarLocationInternal {
	if length == 0 {
		return nil
	}
	locations := make([]interp_domain.VarLocationInternal, length)
	var fbLocation interp_schema_gen.VarLocation
	for i := range length {
		if getter(&fbLocation, i) {
			locations[i] = interp_domain.MakeVarLocation(interp_domain.VarLocationData{
				UpvalueIndex: fbLocation.UpvalueIndex(),
				Register:     fbLocation.Register(),
				Kind:         safeconv.MustInt8ToUint8(int8(fbLocation.Kind())),
				IsUpvalue:    fbLocation.IsUpvalue(),
				IsIndirect:   fbLocation.IsIndirect(),
				OriginalKind: safeconv.MustInt8ToUint8(int8(fbLocation.OriginalKind())),
				IsSpilled:    fbLocation.IsSpilled(),
				SpillSlot:    fbLocation.SpillSlot(),
			})
		}
	}
	return locations
}

// unpackVarLocationData reconstructs variable locations as
// serialisation-safe VarLocationData values from a FlatBuffer
// vector accessed via the getter function.
//
// Takes length (int) which is the number of variable locations in
// the vector.
// Takes getter (func) which retrieves each VarLocation by index
// from the FlatBuffer.
//
// Returns []interp_domain.VarLocationData which holds the
// reconstructed variable location data.
func unpackVarLocationData(length int, getter func(*interp_schema_gen.VarLocation, int) bool) []interp_domain.VarLocationData {
	if length == 0 {
		return nil
	}
	locations := make([]interp_domain.VarLocationData, length)
	var fbLocation interp_schema_gen.VarLocation
	for i := range length {
		if getter(&fbLocation, i) {
			locations[i] = interp_domain.VarLocationData{
				UpvalueIndex: fbLocation.UpvalueIndex(),
				Register:     fbLocation.Register(),
				Kind:         safeconv.MustInt8ToUint8(int8(fbLocation.Kind())),
				IsUpvalue:    fbLocation.IsUpvalue(),
				IsIndirect:   fbLocation.IsIndirect(),
				OriginalKind: safeconv.MustInt8ToUint8(int8(fbLocation.OriginalKind())),
				IsSpilled:    fbLocation.IsSpilled(),
				SpillSlot:    fbLocation.SpillSlot(),
			}
		}
	}
	return locations
}

// unpackGeneralConstantDescriptor reconstructs a general constant
// descriptor from its FlatBuffer representation.
//
// Takes fbDescriptor (*interp_schema_gen.GeneralConstantDescriptor)
// which is the serialised FlatBuffer descriptor.
//
// Returns interp_domain.GeneralConstantDescriptorData which holds
// the reconstructed descriptor data.
func unpackGeneralConstantDescriptor(fbDescriptor *interp_schema_gen.GeneralConstantDescriptor) interp_domain.GeneralConstantDescriptorData {
	data := interp_domain.GeneralConstantDescriptorData{
		Kind:        safeconv.MustInt8ToUint8(int8(fbDescriptor.Kind())),
		PackagePath: mem.String(fbDescriptor.PackagePath()),
		SymbolName:  mem.String(fbDescriptor.SymbolName()),
	}
	if fbTypeDescriptor := fbDescriptor.TypeDescriptor(nil); fbTypeDescriptor != nil {
		data.TypeDesc = unpackTypeDescriptor(fbTypeDescriptor)
	}
	return data
}

// unpackTypeDescriptor recursively reconstructs a type descriptor
// from its FlatBuffer representation. Recursive fields (elem, key,
// value, fields, params, results) are unpacked depth-first.
//
// Takes fbDescriptor (*interp_schema_gen.TypeDescriptor) which is
// the serialised FlatBuffer type descriptor.
//
// Returns interp_domain.TypeDescriptorData which holds the
// reconstructed type descriptor data.
func unpackTypeDescriptor(fbDescriptor *interp_schema_gen.TypeDescriptor) interp_domain.TypeDescriptorData { //nolint:revive // dispatch table
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
		data.Elem = new(unpackTypeDescriptor(fbElem))
	}
	if fbKey := fbDescriptor.Key(nil); fbKey != nil {
		data.Key = new(unpackTypeDescriptor(fbKey))
	}
	if fbValue := fbDescriptor.Value(nil); fbValue != nil {
		data.Value = new(unpackTypeDescriptor(fbValue))
	}
	if fieldCount := fbDescriptor.FieldsLength(); fieldCount > 0 {
		data.Fields = make([]interp_domain.TypeDescFieldData, fieldCount)
		var fbField interp_schema_gen.TypeDescField
		for i := range fieldCount {
			if fbDescriptor.Fields(&fbField, i) {
				data.Fields[i] = interp_domain.TypeDescFieldData{
					Name:        mem.String(fbField.Name()),
					Tag:         mem.String(fbField.Tag()),
					PackagePath: mem.String(fbField.PackagePath()),
				}
				if fbFieldType := fbField.TypeDescriptor(nil); fbFieldType != nil {
					data.Fields[i].Typ = unpackTypeDescriptor(fbFieldType)
				}
			}
		}
	}
	if paramCount := fbDescriptor.ParamsLength(); paramCount > 0 {
		data.Params = make([]interp_domain.TypeDescriptorData, paramCount)
		var fbParameter interp_schema_gen.TypeDescriptor
		for i := range paramCount {
			if fbDescriptor.Params(&fbParameter, i) {
				data.Params[i] = unpackTypeDescriptor(&fbParameter)
			}
		}
	}
	if resultCount := fbDescriptor.ResultsLength(); resultCount > 0 {
		data.Results = make([]interp_domain.TypeDescriptorData, resultCount)
		var fbResultType interp_schema_gen.TypeDescriptor
		for i := range resultCount {
			if fbDescriptor.Results(&fbResultType, i) {
				data.Results[i] = unpackTypeDescriptor(&fbResultType)
			}
		}
	}
	return data
}
