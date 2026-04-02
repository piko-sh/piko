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

	"piko.sh/piko/wdk/safeconv"
)

// Type aliases expose internal types to the adapter layer for
// bytecode serialisation without duplicating struct definitions.
type (
	// RegisterKindValue is the exported alias for registerKind.
	RegisterKindValue = registerKind

	// InstructionValue is the exported alias for instruction.
	InstructionValue = instruction

	// GeneralConstantDescriptorInternal is the exported alias for
	// generalConstantDescriptor.
	GeneralConstantDescriptorInternal = generalConstantDescriptor

	// TypeDescriptorInternal is the exported alias for
	// typeDescriptor.
	TypeDescriptorInternal = typeDescriptor

	// CallSiteInternal is the exported alias for callSite.
	CallSiteInternal = callSite

	// VarLocationInternal is the exported alias for varLocation.
	VarLocationInternal = varLocation
)

// InstructionData is a serialisation-safe representation of a single
// bytecode instruction.
type InstructionData struct {
	// Operation is the opcode byte.
	Operation uint8

	// A is the first operand (typically the destination register).
	A uint8

	// B is the second operand (source register or low immediate byte).
	B uint8

	// C is the third operand (source register or high immediate byte).
	C uint8
}

// UpvalueDescriptorData is a serialisation-safe representation of an
// upvalue descriptor.
type UpvalueDescriptorData struct {
	// Index is the register index in the enclosing scope.
	Index uint8

	// Kind is the register bank of the captured variable.
	Kind uint8

	// IsLocal is true when the upvalue captures directly from the
	// enclosing function.
	IsLocal bool
}

// VarLocationData is a serialisation-safe representation of a
// variable's storage location.
type VarLocationData struct {
	// UpvalueIndex is the index into the upvalue table when
	// IsUpvalue is true.
	UpvalueIndex int32

	// Register is the register index within the bank.
	Register uint8

	// Kind is the register bank identifier.
	Kind uint8

	// IsUpvalue is true when the variable lives on the heap.
	IsUpvalue bool

	// IsIndirect is true when the variable's address has been taken.
	IsIndirect bool

	// OriginalKind is the register bank before heap-escaping.
	OriginalKind uint8

	// IsSpilled is true when the variable lives in the spill area.
	IsSpilled bool

	// SpillSlot is the 0-based spill slot index when IsSpilled is true.
	SpillSlot uint16
}

// CallSiteData is a serialisation-safe representation of a function
// call site's static metadata.
type CallSiteData struct {
	// Arguments records where each argument lives in the caller's
	// frame.
	Arguments []VarLocationData

	// Returns records where to store each return value in the
	// caller's frame.
	Returns []VarLocationData

	// FuncIndex is the index into the enclosing function's child
	// functions slice for the callee.
	FuncIndex uint16

	// ClosureRegister is the general register holding the closure
	// value.
	ClosureRegister uint8

	// NativeRegister is the general register holding the native
	// function value.
	NativeRegister uint8

	// IsClosure is true when the callee is a closure stored in a
	// general register.
	IsClosure bool

	// IsNative is true when the callee is a native Go function.
	IsNative bool

	// IsMethod is true when the callee is a bound method.
	IsMethod bool

	// MethodRecvReg is the general register holding the method
	// receiver.
	MethodRecvReg uint8
}

// TypeDescriptorData is a serialisation-safe representation of a
// reflect.Type descriptor.
type TypeDescriptorData struct {
	// PackagePath is the import path for named types.
	PackagePath string

	// Name is the type name for named types.
	Name string

	// Elem is the element type for pointers, slices, arrays, and
	// channels.
	Elem *TypeDescriptorData

	// Key is the key type for maps.
	Key *TypeDescriptorData

	// Value is the value type for maps.
	Value *TypeDescriptorData

	// Fields holds struct field descriptors.
	Fields []TypeDescFieldData

	// Params holds function parameter type descriptors.
	Params []TypeDescriptorData

	// Results holds function result type descriptors.
	Results []TypeDescriptorData

	// Length is the array length.
	Length int32

	// Dir is the channel direction.
	Dir int32

	// BasicKind is the reflect.Kind for basic types.
	BasicKind uint8

	// Kind is the typeDescKind identifying the structural category.
	Kind uint8

	// IsVariadic is true for variadic function types.
	IsVariadic bool
}

// TypeDescFieldData is a serialisation-safe representation of a
// struct field within a type descriptor.
type TypeDescFieldData struct {
	// Name is the field name.
	Name string

	// Tag is the struct tag string.
	Tag string

	// PackagePath is the package path for unexported fields.
	PackagePath string

	// Typ is the field's type descriptor.
	Typ TypeDescriptorData
}

// GeneralConstantDescriptorData is a serialisation-safe
// representation of a general constant descriptor.
type GeneralConstantDescriptorData struct {
	// PackagePath is the import path of the package containing the
	// symbol or type.
	PackagePath string

	// SymbolName is the name of the symbol or type within its
	// package.
	SymbolName string

	// TypeDesc describes the composite type for composite zero
	// values.
	TypeDesc TypeDescriptorData

	// Kind identifies the source of the general constant.
	Kind uint8
}

// TypeNameData holds a type name entry paired with its type
// descriptor for serialisation.
type TypeNameData struct {
	// Name is the string name associated with the type.
	Name string

	// TypeDesc is the serialisable type descriptor.
	TypeDesc TypeDescriptorData
}

// Root returns the root function container.
//
// Returns *CompiledFunction which holds all compiled functions from
// every source file.
func (cfs *CompiledFileSet) Root() *CompiledFunction { return cfs.root }

// VariableInitFunction returns the package-level variable
// initialisation function.
//
// Returns *CompiledFunction which holds the bytecode for
// package-level variable initialisers, or nil if none exist.
func (cfs *CompiledFileSet) VariableInitFunction() *CompiledFunction {
	return cfs.variableInitFunction
}

// Entrypoints returns the entrypoint name-to-index mapping.
//
// Returns map[string]uint16 which maps function names to their
// indices in the root function's child functions slice.
func (cfs *CompiledFileSet) Entrypoints() map[string]uint16 { return cfs.entrypoints }

// InitFuncs returns the init function indices in source order.
//
// Returns []uint16 which holds the indices of init() functions in
// the root function's child functions slice.
func (cfs *CompiledFileSet) InitFuncs() []uint16 { return cfs.initFunctionIndices }

// ExportName returns the function name for serialisation.
//
// Returns string which is the function's qualified name.
func (cf *CompiledFunction) ExportName() string { return cf.name }

// ExportSourceFile returns the source file name for serialisation.
//
// Returns string which is the source file path where this function
// was defined.
func (cf *CompiledFunction) ExportSourceFile() string { return cf.sourceFile }

// ExportIsVariadic returns whether the function accepts variadic
// arguments.
//
// Returns bool which is true when the function's last parameter is
// variadic.
func (cf *CompiledFunction) ExportIsVariadic() bool { return cf.isVariadic }

// NumRegistersSlice returns the per-bank register counts as a slice.
//
// Returns []uint32 which holds the peak register usage for each
// register bank.
func (cf *CompiledFunction) NumRegistersSlice() []uint32 { return cf.numRegisters[:] }

// ParamKinds returns the parameter register kinds as uint8 values.
//
// Returns []uint8 where each element is a registerKind cast to
// uint8.
func (cf *CompiledFunction) ParamKinds() []uint8 {
	result := make([]uint8, len(cf.paramKinds))
	for i, kind := range cf.paramKinds {
		result[i] = uint8(kind)
	}
	return result
}

// ResultKinds returns the result register kinds as uint8 values.
//
// Returns []uint8 where each element is a registerKind cast to
// uint8.
func (cf *CompiledFunction) ResultKinds() []uint8 {
	result := make([]uint8, len(cf.resultKinds))
	for i, kind := range cf.resultKinds {
		result[i] = uint8(kind)
	}
	return result
}

// Body returns the instruction body as serialisation-safe data.
//
// Returns []InstructionData which holds all bytecode instructions.
func (cf *CompiledFunction) Body() []InstructionData {
	result := make([]InstructionData, len(cf.body))
	for i, instr := range cf.body {
		result[i] = InstructionData{Operation: uint8(instr.op), A: instr.a, B: instr.b, C: instr.c}
	}
	return result
}

// BoolConstants returns the bool constant pool.
//
// Returns []bool which holds all boolean constants referenced by
// bytecode.
func (cf *CompiledFunction) BoolConstants() []bool { return cf.boolConstants }

// IntConstants returns the int64 constant pool.
//
// Returns []int64 which holds all integer constants referenced by
// bytecode.
func (cf *CompiledFunction) IntConstants() []int64 { return cf.intConstants }

// FloatConstants returns the float64 constant pool.
//
// Returns []float64 which holds all floating-point constants
// referenced by bytecode.
func (cf *CompiledFunction) FloatConstants() []float64 { return cf.floatConstants }

// UintConstants returns the uint64 constant pool.
//
// Returns []uint64 which holds all unsigned integer constants
// referenced by bytecode.
func (cf *CompiledFunction) UintConstants() []uint64 { return cf.uintConstants }

// ComplexConstants returns the complex128 constant pool.
//
// Returns []complex128 which holds all complex number constants
// referenced by bytecode.
func (cf *CompiledFunction) ComplexConstants() []complex128 { return cf.complexConstants }

// StringConstants returns the string constant pool.
//
// Returns []string which holds all string constants referenced by
// bytecode.
func (cf *CompiledFunction) StringConstants() []string { return cf.stringConstants }

// ExportFunctions returns the child functions for serialisation.
//
// Returns []*CompiledFunction which holds the nested function
// definitions.
func (cf *CompiledFunction) ExportFunctions() []*CompiledFunction { return cf.functions }

// VariableInitFunction returns the variable initialisation function.
//
// Returns *CompiledFunction which holds the bytecode for local
// variable initialisers, or nil if none exist.
func (cf *CompiledFunction) VariableInitFunction() *CompiledFunction {
	return cf.variableInitFunction
}

// GeneralConstantDescriptors returns the general constant
// descriptors as serialisation-safe data.
//
// Returns []GeneralConstantDescriptorData which describes how to
// reconstruct each general constant from its serialised form.
func (cf *CompiledFunction) GeneralConstantDescriptors() []GeneralConstantDescriptorData {
	result := make([]GeneralConstantDescriptorData, len(cf.generalConstantDescriptors))
	for i := range cf.generalConstantDescriptors {
		result[i] = exportGeneralConstantDescriptor(cf.generalConstantDescriptors[i])
	}
	return result
}

// TypeTableDescriptors returns the type table descriptors as
// serialisation-safe data.
//
// Returns []TypeDescriptorData which describes how to reconstruct
// each type in the type table.
func (cf *CompiledFunction) TypeTableDescriptors() []TypeDescriptorData {
	result := make([]TypeDescriptorData, len(cf.typeTableDescriptors))
	for i := range cf.typeTableDescriptors {
		result[i] = exportTypeDescriptor(cf.typeTableDescriptors[i])
	}
	return result
}

// TypeNames returns the type names map as serialisation-safe data.
//
// Returns map[reflect.Type]TypeNameData which pairs each type with
// its string name and serialisable descriptor.
func (cf *CompiledFunction) TypeNames() map[reflect.Type]TypeNameData {
	if len(cf.typeNames) == 0 {
		return nil
	}
	result := make(map[reflect.Type]TypeNameData, len(cf.typeNames))
	for i, reflectType := range cf.typeTable {
		if i < len(cf.typeTableDescriptors) {
			if name, ok := cf.typeNames[reflectType]; ok {
				result[reflectType] = TypeNameData{
					Name:     name,
					TypeDesc: exportTypeDescriptor(cf.typeTableDescriptors[i]),
				}
			}
		}
	}
	return result
}

// CallSites returns the call sites as serialisation-safe data.
//
// Returns []CallSiteData which holds the static metadata for each
// function call in the bytecode.
func (cf *CompiledFunction) CallSites() []CallSiteData {
	result := make([]CallSiteData, len(cf.callSites))
	for i := range cf.callSites {
		result[i] = exportCallSite(cf.callSites[i])
	}
	return result
}

// UpvalueDescriptors returns the upvalue descriptors as
// serialisation-safe data.
//
// Returns []UpvalueDescriptorData which describes how each upvalue
// is captured.
func (cf *CompiledFunction) UpvalueDescriptors() []UpvalueDescriptorData {
	result := make([]UpvalueDescriptorData, len(cf.upvalueDescriptors))
	for i, descriptor := range cf.upvalueDescriptors {
		result[i] = UpvalueDescriptorData{
			Index:   descriptor.index,
			Kind:    uint8(descriptor.kind),
			IsLocal: descriptor.isLocal,
		}
	}
	return result
}

// NamedResultLocs returns the named result locations as
// serialisation-safe data.
//
// Returns []VarLocationData which describes where each named result
// is stored.
func (cf *CompiledFunction) NamedResultLocs() []VarLocationData {
	return exportVarLocations(cf.namedResultLocs)
}

// MethodTable returns the method table mapping method names to
// function indices.
//
// Returns map[string]uint16 which maps method names to their indices
// in the child functions slice.
func (cf *CompiledFunction) MethodTable() map[string]uint16 { return cf.methodTable }

// NewCompiledFileSetFromData constructs a CompiledFileSet from
// serialisation-safe data. Used by the unpack adapter to rebuild
// after deserialisation.
//
// Takes root (*CompiledFunction) which is the root function
// container.
// Takes variableInitFunction (*CompiledFunction) which is the
// variable initialiser.
// Takes entrypoints (map[string]uint16) which maps function names
// to indices.
// Takes initFunctionIndices ([]uint16) which holds init function indices.
//
// Returns *CompiledFileSet which is the reconstructed compiled file
// set.
func NewCompiledFileSetFromData(
	root *CompiledFunction,
	variableInitFunction *CompiledFunction,
	entrypoints map[string]uint16,
	initFunctionIndices []uint16,
) *CompiledFileSet {
	return &CompiledFileSet{
		root:                 root,
		variableInitFunction: variableInitFunction,
		entrypoints:          entrypoints,
		initFunctionIndices:  initFunctionIndices,
	}
}

// CompiledFunctionData holds serialisation-safe fields for
// constructing a CompiledFunction. Used by the unpack adapter
// to rebuild after deserialisation.
type CompiledFunctionData struct {
	// VariableInitFunction is the package-level variable
	// initialisation function, or nil if none exists.
	VariableInitFunction *CompiledFunction

	// MethodTable is the mapping of method names to their
	// indices in the child functions slice.
	MethodTable map[string]uint16

	// TypeNames is the mapping of reflect types to their
	// string names.
	TypeNames map[reflect.Type]string

	// Name is the qualified name of the function.
	Name string

	// SourceFile is the source file path where the function
	// was defined.
	SourceFile string

	// StringConstants is the string constant pool referenced
	// by bytecode.
	StringConstants []string

	// GeneralConstantDescriptors is the slice of descriptors
	// that describe how to reconstruct general constants.
	GeneralConstantDescriptors []generalConstantDescriptor

	// BoolConstants is the boolean constant pool referenced
	// by bytecode.
	BoolConstants []bool

	// IntConstants is the int64 constant pool referenced by
	// bytecode.
	IntConstants []int64

	// FloatConstants is the float64 constant pool referenced
	// by bytecode.
	FloatConstants []float64

	// UintConstants is the uint64 constant pool referenced by
	// bytecode.
	UintConstants []uint64

	// ComplexConstants is the complex128 constant pool
	// referenced by bytecode.
	ComplexConstants []complex128

	// ResultKinds is the register bank for each return value.
	ResultKinds []registerKind

	// GeneralConstants is the general constant pool holding
	// reflect values referenced by bytecode.
	GeneralConstants []reflect.Value

	// Body is the slice of bytecode instructions forming the
	// function body.
	Body []instruction

	// TypeTable is the slice of reflect types used by the
	// function.
	TypeTable []reflect.Type

	// TypeTableDescriptors is the slice of type descriptors
	// for serialising the type table.
	TypeTableDescriptors []typeDescriptor

	// ParamKinds is the register bank for each parameter.
	ParamKinds []registerKind

	// CallSites is the slice of call site metadata for each
	// function call in the bytecode.
	CallSites []callSite

	// UpvalueDescriptors is the slice of upvalue descriptors
	// that describe captured variables.
	UpvalueDescriptors []UpvalueDescriptor

	// Functions is the slice of child function definitions.
	Functions []*CompiledFunction

	// NamedResultLocs is the slice of variable locations for
	// named return values.
	NamedResultLocs []varLocation

	// NumRegisters is the per-bank peak register usage counts.
	NumRegisters [NumRegisterKinds]uint32

	// IsVariadic is true when the function's last parameter is
	// variadic.
	IsVariadic bool
}

// NewCompiledFunctionFromData constructs a CompiledFunction from
// serialisation-safe data. Used by the unpack adapter to rebuild
// after deserialisation.
//
// Takes d (*CompiledFunctionData) which holds all the fields
// needed to reconstruct the compiled function.
//
// Returns *CompiledFunction which is the reconstructed compiled
// function.
func NewCompiledFunctionFromData(d *CompiledFunctionData) *CompiledFunction {
	return &CompiledFunction{
		name:                       d.Name,
		sourceFile:                 d.SourceFile,
		isVariadic:                 d.IsVariadic,
		numRegisters:               d.NumRegisters,
		paramKinds:                 d.ParamKinds,
		resultKinds:                d.ResultKinds,
		body:                       d.Body,
		boolConstants:              d.BoolConstants,
		intConstants:               d.IntConstants,
		floatConstants:             d.FloatConstants,
		uintConstants:              d.UintConstants,
		complexConstants:           d.ComplexConstants,
		stringConstants:            d.StringConstants,
		generalConstants:           d.GeneralConstants,
		generalConstantDescriptors: d.GeneralConstantDescriptors,
		typeTable:                  d.TypeTable,
		typeTableDescriptors:       d.TypeTableDescriptors,
		typeNames:                  d.TypeNames,
		callSites:                  d.CallSites,
		upvalueDescriptors:         d.UpvalueDescriptors,
		functions:                  d.Functions,
		namedResultLocs:            d.NamedResultLocs,
		methodTable:                d.MethodTable,
		variableInitFunction:       d.VariableInitFunction,
	}
}

// MakeInstruction creates an instruction from raw operand bytes.
//
// Takes operation (uint8) which is the opcode.
// Takes a, b, c (uint8) which are the three operands.
//
// Returns instruction which is the constructed bytecode instruction.
func MakeInstruction(operation, a, b, c uint8) instruction { //nolint:revive // bridges domain/adapter boundary
	return instruction{op: opcode(operation), a: a, b: b, c: c}
}

// MakeRegisterKind creates a registerKind from a uint8 value.
//
// Takes value (uint8) which is the register bank identifier.
//
// Returns registerKind which is the typed register bank value.
func MakeRegisterKind(value uint8) registerKind { return registerKind(value) } //nolint:revive // bridges domain/adapter boundary

// MakeVarLocation creates a varLocation from serialisation-safe
// data.
//
// Takes data (VarLocationData) which holds the variable location
// fields.
//
// Returns varLocation which is the internal variable location.
func MakeVarLocation(data VarLocationData) varLocation { //nolint:revive // bridges domain/adapter boundary
	return varLocation{
		upvalueIndex: int(data.UpvalueIndex),
		register:     data.Register,
		kind:         registerKind(data.Kind),
		isUpvalue:    data.IsUpvalue,
		isIndirect:   data.IsIndirect,
		originalKind: registerKind(data.OriginalKind),
		isSpilled:    data.IsSpilled,
		spillSlot:    data.SpillSlot,
	}
}

// MakeUpvalueDescriptor creates an UpvalueDescriptor from
// serialisation-safe data.
//
// Takes data (UpvalueDescriptorData) which holds the upvalue
// descriptor fields.
//
// Returns UpvalueDescriptor which is the internal upvalue
// descriptor.
func MakeUpvalueDescriptor(data UpvalueDescriptorData) UpvalueDescriptor { //nolint:revive // unexported fields
	return UpvalueDescriptor{
		index:   data.Index,
		kind:    registerKind(data.Kind),
		isLocal: data.IsLocal,
	}
}

// MakeCallSite creates a callSite from serialisation-safe data.
//
// Takes data (CallSiteData) which holds the call site fields.
//
// Returns callSite which is the internal call site.
func MakeCallSite(data CallSiteData) callSite { //nolint:revive // bridges domain/adapter boundary
	arguments := make([]varLocation, len(data.Arguments))
	for i, argument := range data.Arguments {
		arguments[i] = MakeVarLocation(argument)
	}
	returns := make([]varLocation, len(data.Returns))
	for i, returnLocation := range data.Returns {
		returns[i] = MakeVarLocation(returnLocation)
	}
	return callSite{
		arguments:       arguments,
		returns:         returns,
		funcIndex:       data.FuncIndex,
		closureRegister: data.ClosureRegister,
		nativeRegister:  data.NativeRegister,
		isClosure:       data.IsClosure,
		isNative:        data.IsNative,
		isMethod:        data.IsMethod,
		methodRecvReg:   data.MethodRecvReg,
	}
}

// ImportTypeDescriptor converts serialisation-safe TypeDescriptorData
// back to an internal typeDescriptor.
//
// Takes data (TypeDescriptorData) which holds the serialised type
// descriptor fields.
//
// Returns typeDescriptor which is the internal type descriptor.
func ImportTypeDescriptor(data TypeDescriptorData) typeDescriptor { //nolint:revive,dupl // bridges domain/adapter boundary; mirrors export
	descriptor := typeDescriptor{
		packagePath: data.PackagePath,
		name:        data.Name,
		basicKind:   data.BasicKind,
		length:      int(data.Length),
		dir:         int(data.Dir),
		kind:        typeDescKind(data.Kind),
		isVariadic:  data.IsVariadic,
	}
	if data.Elem != nil {
		descriptor.element = new(ImportTypeDescriptor(*data.Elem))
	}
	if data.Key != nil {
		descriptor.key = new(ImportTypeDescriptor(*data.Key))
	}
	if data.Value != nil {
		descriptor.value = new(ImportTypeDescriptor(*data.Value))
	}
	if len(data.Fields) > 0 {
		descriptor.fields = make([]typeDescField, len(data.Fields))
		for i := range data.Fields {
			descriptor.fields[i] = typeDescField{
				name:        data.Fields[i].Name,
				tag:         data.Fields[i].Tag,
				packagePath: data.Fields[i].PackagePath,
				typ:         ImportTypeDescriptor(data.Fields[i].Typ),
			}
		}
	}
	if len(data.Params) > 0 {
		descriptor.params = make([]typeDescriptor, len(data.Params))
		for i := range data.Params {
			descriptor.params[i] = ImportTypeDescriptor(data.Params[i])
		}
	}
	if len(data.Results) > 0 {
		descriptor.results = make([]typeDescriptor, len(data.Results))
		for i := range data.Results {
			descriptor.results[i] = ImportTypeDescriptor(data.Results[i])
		}
	}
	return descriptor
}

// ImportGeneralConstantDescriptor converts serialisation-safe data
// back to an internal generalConstantDescriptor.
//
// Takes data (GeneralConstantDescriptorData) which holds the
// serialised descriptor fields.
//
// Returns generalConstantDescriptor which is the internal
// descriptor.
func ImportGeneralConstantDescriptor(data GeneralConstantDescriptorData) generalConstantDescriptor { //nolint:revive // bridges domain/adapter boundary
	return generalConstantDescriptor{
		packagePath: data.PackagePath,
		symbolName:  data.SymbolName,
		typeDesc:    ImportTypeDescriptor(data.TypeDesc),
		kind:        generalConstantKind(data.Kind),
	}
}

// ReconstructGeneralConstant rebuilds a reflect.Value from a
// serialisation-safe descriptor using the SymbolRegistry.
//
// Takes descriptor (GeneralConstantDescriptorData) which describes
// the constant to reconstruct.
// Takes registry (*SymbolRegistry) which provides symbol lookups.
//
// Returns reflect.Value which is the reconstructed runtime value.
// Returns error when the symbol or type cannot be found.
func ReconstructGeneralConstant(descriptor GeneralConstantDescriptorData, registry *SymbolRegistry) (reflect.Value, error) {
	return reconstructGeneralConstant(ImportGeneralConstantDescriptor(descriptor), registry)
}

// DescriptorToReflectType reconstructs a reflect.Type from
// serialisation-safe descriptor data using the SymbolRegistry.
//
// Takes descriptor (TypeDescriptorData) which describes the type to
// reconstruct.
// Takes registry (*SymbolRegistry) which provides named type
// lookups.
//
// Returns reflect.Type which is the reconstructed runtime type.
// Returns error when a named type cannot be found.
func DescriptorToReflectType(descriptor TypeDescriptorData, registry *SymbolRegistry) (reflect.Type, error) {
	return descriptorToReflectType(ImportTypeDescriptor(descriptor), registry)
}

// exportTypeDescriptor converts an internal typeDescriptor to
// serialisation-safe TypeDescriptorData.
//
// Takes descriptor (typeDescriptor) which is the internal type
// descriptor to convert.
//
// Returns TypeDescriptorData which is the serialisation-safe
// representation.
func exportTypeDescriptor(descriptor typeDescriptor) TypeDescriptorData { //nolint:revive,dupl // dispatch table; mirrors import
	data := TypeDescriptorData{
		PackagePath: descriptor.packagePath,
		Name:        descriptor.name,
		BasicKind:   descriptor.basicKind,
		Length:      safeconv.IntToInt32(descriptor.length),
		Dir:         safeconv.IntToInt32(descriptor.dir),
		Kind:        uint8(descriptor.kind),
		IsVariadic:  descriptor.isVariadic,
	}
	if descriptor.element != nil {
		data.Elem = new(exportTypeDescriptor(*descriptor.element))
	}
	if descriptor.key != nil {
		data.Key = new(exportTypeDescriptor(*descriptor.key))
	}
	if descriptor.value != nil {
		data.Value = new(exportTypeDescriptor(*descriptor.value))
	}
	if len(descriptor.fields) > 0 {
		data.Fields = make([]TypeDescFieldData, len(descriptor.fields))
		for i := range descriptor.fields {
			data.Fields[i] = TypeDescFieldData{
				Name:        descriptor.fields[i].name,
				Tag:         descriptor.fields[i].tag,
				PackagePath: descriptor.fields[i].packagePath,
				Typ:         exportTypeDescriptor(descriptor.fields[i].typ),
			}
		}
	}
	if len(descriptor.params) > 0 {
		data.Params = make([]TypeDescriptorData, len(descriptor.params))
		for i := range descriptor.params {
			data.Params[i] = exportTypeDescriptor(descriptor.params[i])
		}
	}
	if len(descriptor.results) > 0 {
		data.Results = make([]TypeDescriptorData, len(descriptor.results))
		for i := range descriptor.results {
			data.Results[i] = exportTypeDescriptor(descriptor.results[i])
		}
	}
	return data
}

// exportGeneralConstantDescriptor converts an internal
// generalConstantDescriptor to serialisation-safe data.
//
// Takes descriptor (generalConstantDescriptor) which is the
// internal descriptor to convert.
//
// Returns GeneralConstantDescriptorData which is the
// serialisation-safe representation.
func exportGeneralConstantDescriptor(descriptor generalConstantDescriptor) GeneralConstantDescriptorData {
	return GeneralConstantDescriptorData{
		PackagePath: descriptor.packagePath,
		SymbolName:  descriptor.symbolName,
		TypeDesc:    exportTypeDescriptor(descriptor.typeDesc),
		Kind:        uint8(descriptor.kind),
	}
}

// exportCallSite converts an internal callSite to
// serialisation-safe data.
//
// Takes site (callSite) which is the internal call site to
// convert.
//
// Returns CallSiteData which is the serialisation-safe
// representation.
func exportCallSite(site callSite) CallSiteData {
	return CallSiteData{
		Arguments:       exportVarLocations(site.arguments),
		Returns:         exportVarLocations(site.returns),
		FuncIndex:       site.funcIndex,
		ClosureRegister: site.closureRegister,
		NativeRegister:  site.nativeRegister,
		IsClosure:       site.isClosure,
		IsNative:        site.isNative,
		IsMethod:        site.isMethod,
		MethodRecvReg:   site.methodRecvReg,
	}
}

// exportVarLocations converts a slice of internal varLocation to
// serialisation-safe data.
//
// Takes locations ([]varLocation) which is the slice of internal
// variable locations to convert.
//
// Returns []VarLocationData which holds the serialisation-safe
// representations.
func exportVarLocations(locations []varLocation) []VarLocationData {
	if len(locations) == 0 {
		return nil
	}
	result := make([]VarLocationData, len(locations))
	for i, location := range locations {
		result[i] = VarLocationData{
			UpvalueIndex: safeconv.IntToInt32(location.upvalueIndex),
			Register:     location.register,
			Kind:         uint8(location.kind),
			IsUpvalue:    location.isUpvalue,
			IsIndirect:   location.isIndirect,
			OriginalKind: uint8(location.originalKind),
			IsSpilled:    location.isSpilled,
			SpillSlot:    location.spillSlot,
		}
	}
	return result
}
