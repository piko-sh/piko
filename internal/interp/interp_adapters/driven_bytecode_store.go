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
	"io/fs"
	"maps"
	"reflect"
	"slices"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/interp/interp_domain"
	"piko.sh/piko/internal/interp/interp_schema"
	"piko.sh/piko/internal/interp/interp_schema/interp_schema_gen"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// bytecodeDefaultDirPerm is the default directory permission
	// for the bytecode cache directory.
	bytecodeDefaultDirPerm = 0755

	// bytecodeDefaultFilePerm is the default file permission for
	// cached bytecode files.
	bytecodeDefaultFilePerm = 0644

	// bytecodeInitialBuilder is the initial capacity in bytes for
	// the FlatBuffer builder used during serialisation.
	bytecodeInitialBuilder = 4096

	// bytecodeVectorAlignment is the byte alignment used when
	// building FlatBuffer offset vectors.
	bytecodeVectorAlignment = 4
)

// BytecodeStore provides FlatBuffer-based persistence for compiled
// bytecode. It implements interp_domain.BytecodeStorePort.
type BytecodeStore struct {
	// sandbox provides sandboxed filesystem access to the bytecode
	// cache directory.
	sandbox safedisk.Sandbox
}

var _ interp_domain.BytecodeStorePort = (*BytecodeStore)(nil)

// NewBytecodeStore creates a new BytecodeStore with the given
// sandbox. The sandbox root is the bytecode cache directory.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed
// filesystem access to the cache directory.
//
// Returns *BytecodeStore which is ready for use.
func NewBytecodeStore(sandbox safedisk.Sandbox) *BytecodeStore {
	return &BytecodeStore{sandbox: sandbox}
}

// SaveCompiledFileSet serialises and persists a compiled file set
// under the given key. The payload is wrapped with a schema version
// header and written atomically to prevent partial writes.
//
// Takes key (string) which identifies the compiled file set.
// Takes compiledFileSet (*interp_domain.CompiledFileSet) which is
// the compiled file set to persist.
//
// Returns error when the sandbox is nil, the key is empty, or
// filesystem operations fail.
func (bytecodeStore *BytecodeStore) SaveCompiledFileSet(_ context.Context, key string, compiledFileSet *interp_domain.CompiledFileSet) error {
	if bytecodeStore.sandbox == nil || key == "" {
		return errors.New("bytecode store requires a sandbox and key")
	}

	if err := bytecodeStore.sandbox.MkdirAll(".", bytecodeDefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create bytecode directory: %w", err)
	}

	builder := flatbuffers.NewBuilder(bytecodeInitialBuilder)
	rootOffset := packCompiledFileSet(builder, compiledFileSet)
	builder.Finish(rootOffset)

	payload := builder.FinishedBytes()
	versionedData := make([]byte, fbs.PackedSize(len(payload)))
	interp_schema.PackInto(versionedData, payload)

	fileName := fmt.Sprintf("bytecode-%s.bin", key)
	if err := bytecodeStore.sandbox.WriteFileAtomic(fileName, versionedData, bytecodeDefaultFilePerm); err != nil {
		return fmt.Errorf("failed to write bytecode file atomically: %w", err)
	}

	return nil
}

// InvalidateCache removes the cached bytecode for the given key.
// If the file does not exist, no error is returned.
//
// Takes key (string) which identifies the cached bytecode to
// remove.
//
// Returns error when the sandbox is nil, the key is empty, or the
// removal fails for a reason other than the file not existing.
func (bytecodeStore *BytecodeStore) InvalidateCache(_ context.Context, key string) error {
	if bytecodeStore.sandbox == nil || key == "" {
		return errors.New("bytecode store requires a sandbox and key")
	}

	fileName := fmt.Sprintf("bytecode-%s.bin", key)
	err := bytecodeStore.sandbox.Remove(fileName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove bytecode file %s: %w", fileName, err)
	}
	return nil
}

// PackCompiledFileSetToBytes serialises a CompiledFileSet into
// a versioned FlatBuffer byte slice ready for writing to disk.
//
// Takes compiledFileSet (*interp_domain.CompiledFileSet) which is
// the compiled file set to serialise.
//
// Returns []byte which is the versioned FlatBuffer payload.
func PackCompiledFileSetToBytes(compiledFileSet *interp_domain.CompiledFileSet) []byte {
	builder := flatbuffers.NewBuilder(bytecodeInitialBuilder)
	rootOffset := packCompiledFileSet(builder, compiledFileSet)
	builder.Finish(rootOffset)

	payload := builder.FinishedBytes()
	versionedData := make([]byte, fbs.PackedSize(len(payload)))
	interp_schema.PackInto(versionedData, payload)

	return versionedData
}

// packCompiledFileSet serialises a CompiledFileSet into a
// FlatBuffer. The root and variable init functions are packed
// recursively.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes compiledFileSet (*interp_domain.CompiledFileSet) which is
// the compiled file set to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// CompiledFileSet table.
func packCompiledFileSet(builder *flatbuffers.Builder, compiledFileSet *interp_domain.CompiledFileSet) flatbuffers.UOffsetT {
	var rootOffset flatbuffers.UOffsetT
	if compiledFileSet.Root() != nil {
		rootOffset = packCompiledFunction(builder, compiledFileSet.Root())
	}

	var varInitOffset flatbuffers.UOffsetT
	if compiledFileSet.VariableInitFunction() != nil {
		varInitOffset = packCompiledFunction(builder, compiledFileSet.VariableInitFunction())
	}

	entrypointsOffset := packEntrypoints(builder, compiledFileSet.Entrypoints())
	initFunctionIndicesOffset := packUint16Slice(builder, compiledFileSet.InitFuncs())

	interp_schema_gen.CompiledFileSetStart(builder)
	if rootOffset != 0 {
		interp_schema_gen.CompiledFileSetAddRoot(builder, rootOffset)
	}
	if varInitOffset != 0 {
		interp_schema_gen.CompiledFileSetAddVariableInitFunction(builder, varInitOffset)
	}
	if entrypointsOffset != 0 {
		interp_schema_gen.CompiledFileSetAddEntrypoints(builder, entrypointsOffset)
	}
	if initFunctionIndicesOffset != 0 {
		interp_schema_gen.CompiledFileSetAddInitialisationFunctions(builder, initFunctionIndicesOffset)
	}
	return interp_schema_gen.CompiledFileSetEnd(builder)
}

// packCompiledFunction recursively serialises a CompiledFunction
// into a FlatBuffer. All constant pools, descriptors, call sites,
// and child functions are packed.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes compiledFunction (*interp_domain.CompiledFunction) which is
// the function to serialise.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// CompiledFunction table.
func packCompiledFunction(builder *flatbuffers.Builder, compiledFunction *interp_domain.CompiledFunction) flatbuffers.UOffsetT { //nolint:revive // serialisation dispatch
	nameOffset := builder.CreateString(compiledFunction.ExportName())
	sourceFileOffset := builder.CreateString(compiledFunction.ExportSourceFile())

	numRegistersOffset := packRegisterCounts(builder, compiledFunction.NumRegistersSlice())
	paramKindsOffset := packRegisterKinds(builder, compiledFunction.ParamKinds())
	resultKindsOffset := packRegisterKinds(builder, compiledFunction.ResultKinds())

	bodyOffset := packInstructions(builder, compiledFunction.Body())
	boolConstantsOffset := packBoolSlice(builder, compiledFunction.BoolConstants())
	intConstantsOffset := packInt64Slice(builder, compiledFunction.IntConstants())
	floatConstantsOffset := packFloat64Slice(builder, compiledFunction.FloatConstants())
	uintConstantsOffset := packUint64Slice(builder, compiledFunction.UintConstants())
	complexConstantsOffset := packComplexSlice(builder, compiledFunction.ComplexConstants())
	stringConstantsOffset := packStringSlice(builder, compiledFunction.StringConstants())

	generalDescriptorsOffset := packGeneralConstantDescriptors(builder, compiledFunction.GeneralConstantDescriptors())
	typeTableDescriptorsOffset := packTypeDescriptors(builder, compiledFunction.TypeTableDescriptors())
	typeNamesOffset := packTypeNames(builder, compiledFunction.TypeNames())

	callSitesOffset := packCallSites(builder, compiledFunction.CallSites())
	upvalueDescriptorsOffset := packUpvalueDescriptors(builder, compiledFunction.UpvalueDescriptors())

	childFunctions := compiledFunction.ExportFunctions()
	childOffsets := make([]flatbuffers.UOffsetT, len(childFunctions))
	for i, child := range childFunctions {
		childOffsets[i] = packCompiledFunction(builder, child)
	}
	functionsOffset := createVector(builder, childOffsets)

	namedResultLocsOffset := packVarLocations(builder, compiledFunction.NamedResultLocs())
	methodTableOffset := packMethodTable(builder, compiledFunction.MethodTable())

	var varInitFunctionOffset flatbuffers.UOffsetT
	if compiledFunction.VariableInitFunction() != nil {
		varInitFunctionOffset = packCompiledFunction(builder, compiledFunction.VariableInitFunction())
	}

	interp_schema_gen.CompiledFunctionStart(builder)
	interp_schema_gen.CompiledFunctionAddName(builder, nameOffset)
	interp_schema_gen.CompiledFunctionAddSourceFile(builder, sourceFileOffset)
	interp_schema_gen.CompiledFunctionAddIsVariadic(builder, compiledFunction.ExportIsVariadic())
	if numRegistersOffset != 0 {
		interp_schema_gen.CompiledFunctionAddRegisterCounts(builder, numRegistersOffset)
	}
	if paramKindsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddParameterKinds(builder, paramKindsOffset)
	}
	if resultKindsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddResultKinds(builder, resultKindsOffset)
	}
	if bodyOffset != 0 {
		interp_schema_gen.CompiledFunctionAddBody(builder, bodyOffset)
	}
	if boolConstantsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddBoolConstants(builder, boolConstantsOffset)
	}
	if intConstantsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddIntConstants(builder, intConstantsOffset)
	}
	if floatConstantsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddFloatConstants(builder, floatConstantsOffset)
	}
	if uintConstantsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddUintConstants(builder, uintConstantsOffset)
	}
	if complexConstantsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddComplexConstants(builder, complexConstantsOffset)
	}
	if stringConstantsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddStringConstants(builder, stringConstantsOffset)
	}
	if generalDescriptorsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddGeneralConstantDescriptors(builder, generalDescriptorsOffset)
	}
	if typeTableDescriptorsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddTypeTableDescriptors(builder, typeTableDescriptorsOffset)
	}
	if typeNamesOffset != 0 {
		interp_schema_gen.CompiledFunctionAddTypeNames(builder, typeNamesOffset)
	}
	if callSitesOffset != 0 {
		interp_schema_gen.CompiledFunctionAddCallSites(builder, callSitesOffset)
	}
	if upvalueDescriptorsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddUpvalueDescriptors(builder, upvalueDescriptorsOffset)
	}
	if functionsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddFunctions(builder, functionsOffset)
	}
	if namedResultLocsOffset != 0 {
		interp_schema_gen.CompiledFunctionAddNamedResultLocations(builder, namedResultLocsOffset)
	}
	if methodTableOffset != 0 {
		interp_schema_gen.CompiledFunctionAddMethodTable(builder, methodTableOffset)
	}
	if varInitFunctionOffset != 0 {
		interp_schema_gen.CompiledFunctionAddVariableInitFunction(builder, varInitFunctionOffset)
	}
	return interp_schema_gen.CompiledFunctionEnd(builder)
}

// packInstructions packs a slice of instructions as FlatBuffer
// structs. Instructions are 4-byte fixed-size structs enabling
// zero-copy reads.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes instructions ([]interp_domain.InstructionData) which holds
// the bytecode instructions to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// instruction vector, or 0 when the slice is empty.
func packInstructions(builder *flatbuffers.Builder, instructions []interp_domain.InstructionData) flatbuffers.UOffsetT {
	if len(instructions) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartBodyVector(builder, len(instructions))
	for i := len(instructions) - 1; i >= 0; i-- {
		interp_schema_gen.CreateInstruction(builder, instructions[i].Operation, instructions[i].A, instructions[i].B, instructions[i].C)
	}
	return builder.EndVector(len(instructions))
}

// packUpvalueDescriptors packs upvalue descriptors as FlatBuffer
// structs. Each descriptor is a 3-byte fixed-size struct.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes descriptors ([]interp_domain.UpvalueDescriptorData) which
// holds the upvalue descriptors to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packUpvalueDescriptors(builder *flatbuffers.Builder, descriptors []interp_domain.UpvalueDescriptorData) flatbuffers.UOffsetT {
	if len(descriptors) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartUpvalueDescriptorsVector(builder, len(descriptors))
	for i := len(descriptors) - 1; i >= 0; i-- {
		interp_schema_gen.CreateUpvalueDescriptor(builder, descriptors[i].Index, interp_schema_gen.RegisterKind(safeconv.MustUint8ToInt8(descriptors[i].Kind)), descriptors[i].IsLocal)
	}
	return builder.EndVector(len(descriptors))
}

// packVarLocations packs variable locations as FlatBuffer tables.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes locations ([]interp_domain.VarLocationData) which holds the
// variable locations to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packVarLocations(builder *flatbuffers.Builder, locations []interp_domain.VarLocationData) flatbuffers.UOffsetT {
	if len(locations) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(locations))
	for i, location := range locations {
		offsets[i] = packVarLocation(builder, location)
	}
	return createVector(builder, offsets)
}

// packVarLocation packs a single variable location as a FlatBuffer
// table.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes location (interp_domain.VarLocationData) which holds the
// variable location fields.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// VarLocation table.
func packVarLocation(builder *flatbuffers.Builder, location interp_domain.VarLocationData) flatbuffers.UOffsetT {
	interp_schema_gen.VarLocationStart(builder)
	interp_schema_gen.VarLocationAddUpvalueIndex(builder, location.UpvalueIndex)
	interp_schema_gen.VarLocationAddRegister(builder, location.Register)
	interp_schema_gen.VarLocationAddKind(builder, interp_schema_gen.RegisterKind(safeconv.MustUint8ToInt8(location.Kind)))
	interp_schema_gen.VarLocationAddIsUpvalue(builder, location.IsUpvalue)
	interp_schema_gen.VarLocationAddIsIndirect(builder, location.IsIndirect)
	interp_schema_gen.VarLocationAddOriginalKind(builder, interp_schema_gen.RegisterKind(safeconv.MustUint8ToInt8(location.OriginalKind)))
	interp_schema_gen.VarLocationAddIsSpilled(builder, location.IsSpilled)
	interp_schema_gen.VarLocationAddSpillSlot(builder, location.SpillSlot)
	return interp_schema_gen.VarLocationEnd(builder)
}

// packCallSites packs call site descriptors as FlatBuffer tables.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes callSites ([]interp_domain.CallSiteData) which holds the
// call site descriptors to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packCallSites(builder *flatbuffers.Builder, callSites []interp_domain.CallSiteData) flatbuffers.UOffsetT {
	if len(callSites) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(callSites))
	for i, site := range callSites {
		offsets[i] = packCallSite(builder, site)
	}
	return createVector(builder, offsets)
}

// packCallSite packs a single call site as a FlatBuffer table.
// Only static metadata is serialised; runtime caches are
// reconstructed on load.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes site (interp_domain.CallSiteData) which holds the call
// site fields.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// CallSite table.
func packCallSite(builder *flatbuffers.Builder, site interp_domain.CallSiteData) flatbuffers.UOffsetT {
	argumentsOffset := packVarLocations(builder, site.Arguments)
	returnsOffset := packVarLocations(builder, site.Returns)

	interp_schema_gen.CallSiteStart(builder)
	interp_schema_gen.CallSiteAddFunctionIndex(builder, site.FuncIndex)
	interp_schema_gen.CallSiteAddClosureRegister(builder, site.ClosureRegister)
	interp_schema_gen.CallSiteAddNativeRegister(builder, site.NativeRegister)
	interp_schema_gen.CallSiteAddIsClosure(builder, site.IsClosure)
	interp_schema_gen.CallSiteAddIsNative(builder, site.IsNative)
	interp_schema_gen.CallSiteAddIsMethod(builder, site.IsMethod)
	interp_schema_gen.CallSiteAddMethodReceiverRegister(builder, site.MethodRecvReg)
	if argumentsOffset != 0 {
		interp_schema_gen.CallSiteAddArguments(builder, argumentsOffset)
	}
	if returnsOffset != 0 {
		interp_schema_gen.CallSiteAddReturns(builder, returnsOffset)
	}
	return interp_schema_gen.CallSiteEnd(builder)
}

// packTypeDescriptor recursively packs a type descriptor as a
// FlatBuffer table. Recursive fields (elem, key, value, fields,
// params, results) are packed depth-first.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes descriptor (interp_domain.TypeDescriptorData) which holds
// the type descriptor to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// TypeDescriptor table.
func packTypeDescriptor(builder *flatbuffers.Builder, descriptor interp_domain.TypeDescriptorData) flatbuffers.UOffsetT {
	packagePathOffset := builder.CreateString(descriptor.PackagePath)
	nameOffset := builder.CreateString(descriptor.Name)

	var elemOffset, keyOffset, valueOffset flatbuffers.UOffsetT
	if descriptor.Elem != nil {
		elemOffset = packTypeDescriptor(builder, *descriptor.Elem)
	}
	if descriptor.Key != nil {
		keyOffset = packTypeDescriptor(builder, *descriptor.Key)
	}
	if descriptor.Value != nil {
		valueOffset = packTypeDescriptor(builder, *descriptor.Value)
	}

	fieldsOffset := packTypeDescFields(builder, descriptor.Fields)
	paramsOffset := packTypeDescriptors(builder, descriptor.Params)
	resultsOffset := packTypeDescriptors(builder, descriptor.Results)

	interp_schema_gen.TypeDescriptorStart(builder)
	interp_schema_gen.TypeDescriptorAddKind(builder, interp_schema_gen.TypeDescKind(safeconv.MustUint8ToInt8(descriptor.Kind)))
	interp_schema_gen.TypeDescriptorAddPackagePath(builder, packagePathOffset)
	interp_schema_gen.TypeDescriptorAddName(builder, nameOffset)
	interp_schema_gen.TypeDescriptorAddBasicKind(builder, descriptor.BasicKind)
	if elemOffset != 0 {
		interp_schema_gen.TypeDescriptorAddElement(builder, elemOffset)
	}
	if keyOffset != 0 {
		interp_schema_gen.TypeDescriptorAddKey(builder, keyOffset)
	}
	if valueOffset != 0 {
		interp_schema_gen.TypeDescriptorAddValue(builder, valueOffset)
	}
	interp_schema_gen.TypeDescriptorAddLength(builder, descriptor.Length)
	interp_schema_gen.TypeDescriptorAddDirection(builder, descriptor.Dir)
	if fieldsOffset != 0 {
		interp_schema_gen.TypeDescriptorAddFields(builder, fieldsOffset)
	}
	if paramsOffset != 0 {
		interp_schema_gen.TypeDescriptorAddParams(builder, paramsOffset)
	}
	if resultsOffset != 0 {
		interp_schema_gen.TypeDescriptorAddResults(builder, resultsOffset)
	}
	interp_schema_gen.TypeDescriptorAddIsVariadic(builder, descriptor.IsVariadic)
	return interp_schema_gen.TypeDescriptorEnd(builder)
}

// packTypeDescriptors packs a slice of type descriptors as a
// FlatBuffer vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes descriptors ([]interp_domain.TypeDescriptorData) which
// holds the type descriptors to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packTypeDescriptors(builder *flatbuffers.Builder, descriptors []interp_domain.TypeDescriptorData) flatbuffers.UOffsetT {
	if len(descriptors) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(descriptors))
	for i := range descriptors {
		offsets[i] = packTypeDescriptor(builder, descriptors[i])
	}
	return createVector(builder, offsets)
}

// packTypeDescFields packs struct field descriptors as FlatBuffer
// tables.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes fields ([]interp_domain.TypeDescFieldData) which holds the
// struct field descriptors to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packTypeDescFields(builder *flatbuffers.Builder, fields []interp_domain.TypeDescFieldData) flatbuffers.UOffsetT {
	if len(fields) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(fields))
	for i := range fields {
		nameOffset := builder.CreateString(fields[i].Name)
		tagOffset := builder.CreateString(fields[i].Tag)
		packagePathOffset := builder.CreateString(fields[i].PackagePath)
		typeOffset := packTypeDescriptor(builder, fields[i].Typ)

		interp_schema_gen.TypeDescFieldStart(builder)
		interp_schema_gen.TypeDescFieldAddName(builder, nameOffset)
		interp_schema_gen.TypeDescFieldAddTag(builder, tagOffset)
		interp_schema_gen.TypeDescFieldAddPackagePath(builder, packagePathOffset)
		interp_schema_gen.TypeDescFieldAddTypeDescriptor(builder, typeOffset)
		offsets[i] = interp_schema_gen.TypeDescFieldEnd(builder)
	}
	return createVector(builder, offsets)
}

// packGeneralConstantDescriptors packs general constant descriptors
// as FlatBuffer tables.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes descriptors ([]interp_domain.GeneralConstantDescriptorData)
// which holds the descriptors to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packGeneralConstantDescriptors(builder *flatbuffers.Builder, descriptors []interp_domain.GeneralConstantDescriptorData) flatbuffers.UOffsetT {
	if len(descriptors) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(descriptors))
	for i := range descriptors {
		packagePathOffset := builder.CreateString(descriptors[i].PackagePath)
		symbolNameOffset := builder.CreateString(descriptors[i].SymbolName)
		typeDescOffset := packTypeDescriptor(builder, descriptors[i].TypeDesc)

		interp_schema_gen.GeneralConstantDescriptorStart(builder)
		interp_schema_gen.GeneralConstantDescriptorAddKind(builder, interp_schema_gen.GeneralConstantKind(safeconv.MustUint8ToInt8(descriptors[i].Kind)))
		interp_schema_gen.GeneralConstantDescriptorAddPackagePath(builder, packagePathOffset)
		interp_schema_gen.GeneralConstantDescriptorAddSymbolName(builder, symbolNameOffset)
		interp_schema_gen.GeneralConstantDescriptorAddTypeDescriptor(builder, typeDescOffset)
		offsets[i] = interp_schema_gen.GeneralConstantDescriptorEnd(builder)
	}
	return createVector(builder, offsets)
}

// packTypeNames packs the type names map as a vector of entries.
// Each entry pairs a type descriptor with its string name.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes typeNames (map[reflect.Type]interp_domain.TypeNameData)
// which maps runtime types to their serialisable name entries.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the map is empty.
func packTypeNames(builder *flatbuffers.Builder, typeNames map[reflect.Type]interp_domain.TypeNameData) flatbuffers.UOffsetT {
	if len(typeNames) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, 0, len(typeNames))
	for _, data := range typeNames { //nolint:gocritic // map iteration copies values
		nameOffset := builder.CreateString(data.Name)
		typeDescOffset := packTypeDescriptor(builder, data.TypeDesc)

		interp_schema_gen.TypeNameEntryStart(builder)
		interp_schema_gen.TypeNameEntryAddName(builder, nameOffset)
		interp_schema_gen.TypeNameEntryAddTypeDescriptor(builder, typeDescOffset)
		offsets = append(offsets, interp_schema_gen.TypeNameEntryEnd(builder))
	}
	return createVector(builder, offsets)
}

// packEntrypoints packs the entrypoints map as a sorted vector of
// entrypoint entries.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes entrypoints (map[string]uint16) which maps function names
// to their indices.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the map is empty.
func packEntrypoints(builder *flatbuffers.Builder, entrypoints map[string]uint16) flatbuffers.UOffsetT {
	return packStringUint16Map(builder, entrypoints, func(mapBuilder *flatbuffers.Builder, nameOffset flatbuffers.UOffsetT, index uint16) flatbuffers.UOffsetT {
		interp_schema_gen.EntrypointEntryStart(mapBuilder)
		interp_schema_gen.EntrypointEntryAddName(mapBuilder, nameOffset)
		interp_schema_gen.EntrypointEntryAddFunctionIndex(mapBuilder, index)
		return interp_schema_gen.EntrypointEntryEnd(mapBuilder)
	})
}

// packMethodTable packs the method table map as a sorted vector of
// method table entries.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes methodTable (map[string]uint16) which maps method names to
// their function indices.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the map is empty.
func packMethodTable(builder *flatbuffers.Builder, methodTable map[string]uint16) flatbuffers.UOffsetT {
	return packStringUint16Map(builder, methodTable, func(mapBuilder *flatbuffers.Builder, nameOffset flatbuffers.UOffsetT, index uint16) flatbuffers.UOffsetT {
		interp_schema_gen.MethodTableEntryStart(mapBuilder)
		interp_schema_gen.MethodTableEntryAddName(mapBuilder, nameOffset)
		interp_schema_gen.MethodTableEntryAddFunctionIndex(mapBuilder, index)
		return interp_schema_gen.MethodTableEntryEnd(mapBuilder)
	})
}

// packStringUint16Map packs a map[string]uint16 as a sorted vector
// of FlatBuffer entries using the provided entry builder function.
// Keys are sorted for deterministic output.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes entries (map[string]uint16) which is the map to serialise.
// Takes buildEntry (func) which creates each entry from a name
// offset and uint16 value.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the map is empty.
func packStringUint16Map(
	builder *flatbuffers.Builder,
	entries map[string]uint16,
	buildEntry func(*flatbuffers.Builder, flatbuffers.UOffsetT, uint16) flatbuffers.UOffsetT,
) flatbuffers.UOffsetT {
	if len(entries) == 0 {
		return 0
	}
	keys := slices.Sorted(maps.Keys(entries))
	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, key := range keys {
		nameOffset := builder.CreateString(key)
		offsets[i] = buildEntry(builder, nameOffset, entries[key])
	}
	return createVector(builder, offsets)
}

// packComplexSlice packs complex128 constants as FlatBuffer
// ComplexValue structs. Each struct is 16 bytes (two float64).
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]complex128) which holds the complex constants to
// pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packComplexSlice(builder *flatbuffers.Builder, values []complex128) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartComplexConstantsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		interp_schema_gen.CreateComplexValue(builder, real(values[i]), imag(values[i]))
	}
	return builder.EndVector(len(values))
}

// packStringSlice packs string constants as a FlatBuffer string
// vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]string) which holds the string constants to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packStringSlice(builder *flatbuffers.Builder, values []string) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(values))
	for i, value := range values {
		offsets[i] = builder.CreateString(value)
	}
	return createVector(builder, offsets)
}

// packBoolSlice packs bool constants as a FlatBuffer boolean vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]bool) which holds the boolean constants to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packBoolSlice(builder *flatbuffers.Builder, values []bool) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartBoolConstantsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		builder.PrependBool(values[i])
	}
	return builder.EndVector(len(values))
}

// packInt64Slice packs int64 constants as a FlatBuffer int64 vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]int64) which holds the integer constants to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packInt64Slice(builder *flatbuffers.Builder, values []int64) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartIntConstantsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		builder.PrependInt64(values[i])
	}
	return builder.EndVector(len(values))
}

// packFloat64Slice packs float64 constants as a FlatBuffer float64
// vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]float64) which holds the floating-point constants
// to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packFloat64Slice(builder *flatbuffers.Builder, values []float64) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartFloatConstantsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		builder.PrependFloat64(values[i])
	}
	return builder.EndVector(len(values))
}

// packUint64Slice packs uint64 constants as a FlatBuffer uint64
// vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]uint64) which holds the unsigned integer constants
// to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packUint64Slice(builder *flatbuffers.Builder, values []uint64) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartUintConstantsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		builder.PrependUint64(values[i])
	}
	return builder.EndVector(len(values))
}

// packUint16Slice packs uint16 init function indices as a FlatBuffer
// uint16 vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]uint16) which holds the init function indices to
// pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packUint16Slice(builder *flatbuffers.Builder, values []uint16) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFileSetStartInitialisationFunctionsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		builder.PrependUint16(values[i])
	}
	return builder.EndVector(len(values))
}

// packRegisterCounts packs per-bank register counts as a uint32
// FlatBuffer vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]uint32) which holds the register counts to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packRegisterCounts(builder *flatbuffers.Builder, values []uint32) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartRegisterCountsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		builder.PrependUint32(values[i])
	}
	return builder.EndVector(len(values))
}

// packRegisterKinds packs register kind values as a FlatBuffer int8
// vector.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes values ([]uint8) which holds the register kind values to
// pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func packRegisterKinds(builder *flatbuffers.Builder, values []uint8) flatbuffers.UOffsetT {
	if len(values) == 0 {
		return 0
	}
	interp_schema_gen.CompiledFunctionStartParameterKindsVector(builder, len(values))
	for i := len(values) - 1; i >= 0; i-- {
		builder.PrependInt8(safeconv.MustUint8ToInt8(values[i]))
	}
	return builder.EndVector(len(values))
}

// createVector builds a FlatBuffers vector from pre-built table
// offsets.
//
// Takes builder (*flatbuffers.Builder) which is the FlatBuffer
// builder to write into.
// Takes offsets ([]flatbuffers.UOffsetT) which holds the table
// offsets to include in the vector.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed
// vector, or 0 when the slice is empty.
func createVector(builder *flatbuffers.Builder, offsets []flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(offsets) == 0 {
		return 0
	}
	builder.StartVector(bytecodeVectorAlignment, len(offsets), bytecodeVectorAlignment)
	for i := len(offsets) - 1; i >= 0; i-- {
		builder.PrependUOffsetT(offsets[i])
	}
	return builder.EndVector(len(offsets))
}
