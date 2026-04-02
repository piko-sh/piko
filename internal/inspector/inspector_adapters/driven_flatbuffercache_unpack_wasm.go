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

//go:build js && wasm

package inspector_adapters

import (
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/inspector/inspector_schema/inspector_schema_gen"
	"piko.sh/piko/internal/mem"
)

// unpackPackages unpacks packages from a FlatBuffer using WASM-safe patterns
// with arena allocation.
//
// Takes fb (*inspector_schema_gen.TypeData) which contains the
// serialised package data to unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Package which maps package paths to their
// unpacked package data, or nil when there are no packages.
func unpackPackages(fb *inspector_schema_gen.TypeData, arena *unpackArena) map[string]*inspector_dto.Package {
	length := fb.PackagesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Package, length)
	var entry inspector_schema_gen.PackageEntry
	var pkg inspector_schema_gen.Package
	for i := range length {
		if fb.Packages(&entry, i) {
			fbPackage := entry.Value(&pkg)
			if fbPackage != nil {
				key := mem.String(entry.Key())
				m[key] = unpackPackageWASM(fbPackage, arena)
			}
		}
	}
	return m
}

// unpackPackageWASM unpacks a single package using WASM-safe patterns with
// arena allocation.
//
// Takes fb (*inspector_schema_gen.Package) which is the FlatBuffer package to
// unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Package which contains the unpacked package data.
func unpackPackageWASM(fb *inspector_schema_gen.Package, arena *unpackArena) *inspector_dto.Package {
	p := arena.AllocPackage()
	p.Path = mem.String(fb.Path())
	p.Name = mem.String(fb.Name())
	p.Version = mem.String(fb.Version())
	p.FileImports = unpackFileImportsWASM(fb)
	p.NamedTypes = unpackNamedTypesWASM(fb, arena)
	p.Funcs = unpackFuncsWASM(fb, arena)
	p.Variables = unpackVariablesWASM(fb, arena)
	return p
}

// unpackFileImportsWASM extracts file imports using WASM-safe patterns.
//
// Takes fb (*inspector_schema_gen.Package) which provides the package data to
// extract imports from.
//
// Returns map[string]map[string]string which maps file paths to their import
// alias mappings, or nil when there are no imports.
func unpackFileImportsWASM(fb *inspector_schema_gen.Package) map[string]map[string]string {
	length := fb.FileImportsLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]map[string]string, length)
	var entry inspector_schema_gen.FileImportsEntry
	var importMap inspector_schema_gen.FileImportMap
	for i := range length {
		if fb.FileImports(&entry, i) {
			fbImportMap := entry.Value(&importMap)
			if fbImportMap != nil {
				key := mem.String(entry.Key())
				m[key] = unpackAliasMapWASM(fbImportMap)
			}
		}
	}
	return m
}

// unpackAliasMapWASM extracts alias-to-path mappings using WASM-safe patterns.
//
// Takes fb (*inspector_schema_gen.FileImportMap) which contains the FlatBuffer
// import map to unpack.
//
// Returns map[string]string which maps aliases to their paths, or nil if the
// input contains no entries.
func unpackAliasMapWASM(fb *inspector_schema_gen.FileImportMap) map[string]string {
	length := fb.EntriesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]string, length)
	var entry inspector_schema_gen.AliasToPathEntry
	for i := range length {
		if fb.Entries(&entry, i) {
			m[mem.String(entry.Key())] = mem.String(entry.Value())
		}
	}
	return m
}

// unpackNamedTypesWASM extracts named types using WASM-safe patterns with
// arena allocation.
//
// Takes fb (*inspector_schema_gen.Package) which provides the
// FlatBuffer package containing named type entries.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Type which maps type names to their
// unpacked type definitions, or nil when the package has no named types.
func unpackNamedTypesWASM(fb *inspector_schema_gen.Package, arena *unpackArena) map[string]*inspector_dto.Type {
	length := fb.NamedTypesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Type, length)
	var entry inspector_schema_gen.NamedTypeEntry
	var typ inspector_schema_gen.Type
	for i := range length {
		if fb.NamedTypes(&entry, i) {
			fbTyp := entry.Value(&typ)
			if fbTyp != nil {
				key := mem.String(entry.Key())
				m[key] = unpackTypeWASM(fbTyp, arena)
			}
		}
	}
	return m
}

// unpackFuncsWASM extracts functions using WASM-safe patterns with arena
// allocation.
//
// Takes fb (*inspector_schema_gen.Package) which contains the
// FlatBuffers package data to extract functions from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Function which maps function names to their
// definitions, or nil when the package contains no functions.
func unpackFuncsWASM(fb *inspector_schema_gen.Package, arena *unpackArena) map[string]*inspector_dto.Function {
	length := fb.FunctionsLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Function, length)
	var entry inspector_schema_gen.FunctionEntry
	var inspectedFunction inspector_schema_gen.Function
	for i := range length {
		if fb.Functions(&entry, i) {
			flatbufferFunction := entry.Value(&inspectedFunction)
			if flatbufferFunction != nil {
				key := mem.String(entry.Key())
				m[key] = unpackFunctionWASM(flatbufferFunction, arena)
			}
		}
	}
	return m
}

// unpackVariablesWASM extracts variables using WASM-safe patterns with arena
// allocation.
//
// Takes fb (*inspector_schema_gen.Package) which contains the serialised
// variable data to unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns map[string]*inspector_dto.Variable which maps variable names to
// their unpacked representations, or nil if the package has no variables.
func unpackVariablesWASM(fb *inspector_schema_gen.Package, arena *unpackArena) map[string]*inspector_dto.Variable {
	length := fb.VariablesLength()
	if length == 0 {
		return nil
	}

	m := make(map[string]*inspector_dto.Variable, length)
	var entry inspector_schema_gen.VariableEntry
	var v inspector_schema_gen.Variable
	for i := range length {
		if fb.Variables(&entry, i) {
			fbVar := entry.Value(&v)
			if fbVar != nil {
				key := mem.String(entry.Key())
				m[key] = unpackVariableWASM(fbVar, arena)
			}
		}
	}
	return m
}

// unpackVariableWASM converts a FlatBuffer variable using WASM-safe patterns
// with arena allocation.
//
// Takes fb (*inspector_schema_gen.Variable) which is the FlatBuffer variable
// to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Variable which contains the converted variable data.
func unpackVariableWASM(fb *inspector_schema_gen.Variable, arena *unpackArena) *inspector_dto.Variable {
	result := arena.AllocVariable()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.DefinedInFilePath = mem.String(fb.DefinedInFilePath())
	result.CompositeParts = unpackVariableCompositePartsWASM(fb, arena)
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	result.CompositeType = inspector_dto.CompositeType(fb.CompositeType())
	result.IsConst = fb.IsConst()
	return result
}

// unpackVariableCompositePartsWASM extracts composite parts from a variable
// using arena allocation.
//
// Takes fb (*inspector_schema_gen.Variable) which contains the composite parts
// to extract.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.CompositePart which contains the extracted parts,
// or nil when the variable has no composite parts.
func unpackVariableCompositePartsWASM(fb *inspector_schema_gen.Variable, arena *unpackArena) []*inspector_dto.CompositePart {
	length := fb.CompositePartsLength()
	if length == 0 {
		return nil
	}

	s := arena.CompositePartPtrSlice(length)
	var part inspector_schema_gen.CompositePart
	for i := range length {
		if fb.CompositeParts(&part, i) {
			s[i] = unpackCompositePartWASM(&part, arena)
		}
	}
	return s
}

// unpackTypeWASM converts a FlatBuffer type using WASM-safe patterns with
// arena allocation.
//
// Takes fb (*inspector_schema_gen.Type) which is the FlatBuffer type
// to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Type which is the converted data transfer object.
func unpackTypeWASM(fb *inspector_schema_gen.Type, arena *unpackArena) *inspector_dto.Type {
	result := arena.AllocType()
	result.Name = mem.String(fb.Name())
	result.PackagePath = mem.String(fb.PackagePath())
	result.DefinedInFilePath = mem.String(fb.DefinedInFilePath())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.Fields = unpackFieldsWASM(fb, arena)
	result.Methods = unpackMethodsWASM(fb, arena)
	result.TypeParams = unpackTypeParamsWASM(fb, arena)
	result.Stringability = inspector_dto.StringabilityMethod(fb.Stringability())
	result.IsAlias = fb.IsAlias()
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackFieldsWASM extracts fields using WASM-safe patterns with arena
// allocation.
//
// Takes fb (*inspector_schema_gen.Type) which provides the type to
// extract fields from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.Field which contains the extracted fields, or nil
// if the type has no fields.
func unpackFieldsWASM(fb *inspector_schema_gen.Type, arena *unpackArena) []*inspector_dto.Field {
	length := fb.FieldsLength()
	if length == 0 {
		return nil
	}

	s := arena.FieldPtrSlice(length)
	var field inspector_schema_gen.Field
	for i := range length {
		if fb.Fields(&field, i) {
			s[i] = unpackFieldWASM(&field, arena)
		}
	}
	return s
}

// unpackFieldWASM converts a single field using WASM-safe patterns with arena
// allocation.
//
// Takes fb (*inspector_schema_gen.Field) which is the FlatBuffers
// field to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Field which is the converted field with all string
// values safely copied from WASM memory.
func unpackFieldWASM(fb *inspector_schema_gen.Field, arena *unpackArena) *inspector_dto.Field {
	result := arena.AllocField()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.IsEmbedded = fb.IsEmbedded()
	result.RawTag = mem.String(fb.RawTag())
	result.PackagePath = mem.String(fb.PackagePath())
	result.IsGenericPlaceholder = fb.IsGenericPlaceholder()
	result.DeclaringPackagePath = mem.String(fb.DeclaringPackagePath())
	result.DeclaringTypeName = mem.String(fb.DeclaringTypeName())
	result.CompositeParts = unpackCompositePartsWASM(fb, arena)
	result.CompositeType = inspector_dto.CompositeType(fb.CompositeType())
	result.IsUnderlyingInternalType = fb.IsUnderlyingInternalType()
	result.IsUnderlyingPrimitive = fb.IsUnderlyingPrimitive()
	result.IsInternalType = fb.IsInternalType()
	result.IsAlias = fb.IsAlias()
	result.DefinitionFilePath = mem.String(fb.DefinitionFilePath())
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackCompositePartsWASM extracts composite parts from a field using arena
// allocation.
//
// Takes fb (*inspector_schema_gen.Field) which contains the composite parts to
// extract.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.CompositePart which contains the extracted parts,
// or nil when the field has no composite parts.
func unpackCompositePartsWASM(fb *inspector_schema_gen.Field, arena *unpackArena) []*inspector_dto.CompositePart {
	length := fb.CompositePartsLength()
	if length == 0 {
		return nil
	}

	s := arena.CompositePartPtrSlice(length)
	var part inspector_schema_gen.CompositePart
	for i := range length {
		if fb.CompositeParts(&part, i) {
			s[i] = unpackCompositePartWASM(&part, arena)
		}
	}
	return s
}

// unpackCompositePartWASM converts a single composite part using arena
// allocation.
//
// Takes fb (*inspector_schema_gen.CompositePart) which is the FlatBuffers
// composite part to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.CompositePart which is the converted DTO
// representation.
func unpackCompositePartWASM(fb *inspector_schema_gen.CompositePart, arena *unpackArena) *inspector_dto.CompositePart {
	result := arena.AllocCompositePart()
	result.Type = mem.String(fb.Type())
	result.TypeString = mem.String(fb.TypeString())
	result.Role = mem.String(fb.Role())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.PackagePath = mem.String(fb.PackagePath())
	result.CompositeParts = unpackNestedCompositePartsWASM(fb, arena)
	result.CompositeType = inspector_dto.CompositeType(fb.CompositeType())
	result.Index = int(fb.Index())
	result.IsInternalType = fb.IsInternalType()
	result.IsUnderlyingInternalType = fb.IsUnderlyingInternalType()
	result.IsGenericPlaceholder = fb.IsGenericPlaceholder()
	result.IsAlias = fb.IsAlias()
	result.IsUnderlyingPrimitive = fb.IsUnderlyingPrimitive()
	return result
}

// unpackNestedCompositePartsWASM extracts nested composite parts using arena
// allocation.
//
// Takes fb (*inspector_schema_gen.CompositePart) which is the parent
// composite part to extract nested parts from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.CompositePart which contains the extracted nested
// parts, or nil if there are no nested parts.
func unpackNestedCompositePartsWASM(fb *inspector_schema_gen.CompositePart, arena *unpackArena) []*inspector_dto.CompositePart {
	length := fb.CompositePartsLength()
	if length == 0 {
		return nil
	}

	s := arena.CompositePartPtrSlice(length)
	var part inspector_schema_gen.CompositePart
	for i := range length {
		if fb.CompositeParts(&part, i) {
			s[i] = unpackCompositePartWASM(&part, arena)
		}
	}
	return s
}

// unpackMethodsWASM extracts methods using WASM-safe patterns with arena
// allocation.
//
// Takes fb (*inspector_schema_gen.Type) which is the FlatBuffers type
// to extract methods from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []*inspector_dto.Method which contains the extracted methods, or nil
// if the type has no methods.
func unpackMethodsWASM(fb *inspector_schema_gen.Type, arena *unpackArena) []*inspector_dto.Method {
	length := fb.MethodsLength()
	if length == 0 {
		return nil
	}

	s := arena.MethodPtrSlice(length)
	var method inspector_schema_gen.Method
	for i := range length {
		if fb.Methods(&method, i) {
			s[i] = unpackMethodWASM(&method, arena)
		}
	}
	return s
}

// unpackMethodWASM converts a single method from FlatBuffers to DTO format
// using arena allocation.
//
// Takes fb (*inspector_schema_gen.Method) which is the FlatBuffers
// method to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Method which is the converted method representation.
func unpackMethodWASM(fb *inspector_schema_gen.Method, arena *unpackArena) *inspector_dto.Method {
	var sig inspector_schema_gen.FunctionSignature
	fbSig := fb.Signature(&sig)

	result := arena.AllocMethod()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.Signature = unpackFunctionSignatureWASM(fbSig, arena)
	result.IsPointerReceiver = fb.IsPointerReceiver()
	result.DeclaringPackagePath = mem.String(fb.DeclaringPackagePath())
	result.DeclaringTypeName = mem.String(fb.DeclaringTypeName())
	result.DefinitionFilePath = mem.String(fb.DefinitionFilePath())
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackFunctionWASM converts a function from the WASM schema to the DTO using
// arena allocation.
//
// Takes fb (*inspector_schema_gen.Function) which is the function to convert.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns *inspector_dto.Function which is the converted function data.
func unpackFunctionWASM(fb *inspector_schema_gen.Function, arena *unpackArena) *inspector_dto.Function {
	var sig inspector_schema_gen.FunctionSignature
	fbSig := fb.Signature(&sig)

	result := arena.AllocFunction()
	result.Name = mem.String(fb.Name())
	result.TypeString = mem.String(fb.TypeString())
	result.UnderlyingTypeString = mem.String(fb.UnderlyingTypeString())
	result.Signature = unpackFunctionSignatureWASM(fbSig, arena)
	result.DefinitionFilePath = mem.String(fb.DefinitionFilePath())
	result.DefinitionLine = int(fb.DefinitionLine())
	result.DefinitionColumn = int(fb.DefinitionColumn())
	return result
}

// unpackFunctionSignatureWASM extracts a function signature from a WASM buffer
// using arena allocation.
//
// Takes fb (*inspector_schema_gen.FunctionSignature) which is the flatbuffer to
// unpack.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns inspector_dto.FunctionSignature which contains the extracted
// parameter and result type names, or an empty signature if fb is nil or has
// no parameters or results.
func unpackFunctionSignatureWASM(fb *inspector_schema_gen.FunctionSignature, arena *unpackArena) inspector_dto.FunctionSignature {
	if fb == nil {
		return inspector_dto.FunctionSignature{}
	}

	paramsLen := fb.ParamsLength()
	resultsLen := fb.ResultsLength()
	paramNamesLen := fb.ParamNamesLength()

	if paramsLen == 0 && resultsLen == 0 && paramNamesLen == 0 {
		return inspector_dto.FunctionSignature{}
	}

	total := paramsLen + resultsLen + paramNamesLen
	backing := arena.StringSlice(total)

	for i := range paramsLen {
		backing[i] = mem.String(fb.Params(i))
	}
	for i := range resultsLen {
		backing[paramsLen+i] = mem.String(fb.Results(i))
	}
	for i := range paramNamesLen {
		backing[paramsLen+resultsLen+i] = mem.String(fb.ParamNames(i))
	}

	sig := inspector_dto.FunctionSignature{
		Params:  backing[:paramsLen:paramsLen],
		Results: backing[paramsLen : paramsLen+resultsLen : paramsLen+resultsLen],
	}
	if paramNamesLen > 0 {
		sig.ParamNames = backing[paramsLen+resultsLen : paramsLen+resultsLen+paramNamesLen : paramsLen+resultsLen+paramNamesLen]
	}
	return sig
}

// unpackTypeParamsWASM extracts type parameters from a FlatBuffer type using
// arena allocation.
//
// Takes fb (*inspector_schema_gen.Type) which is the FlatBuffer type to extract
// parameters from.
// Takes arena (*unpackArena) which provides pre-allocated memory slabs.
//
// Returns []string which contains the extracted type parameters, or nil if
// there are none.
func unpackTypeParamsWASM(fb *inspector_schema_gen.Type, arena *unpackArena) []string {
	length := fb.TypeParamsLength()
	if length == 0 {
		return nil
	}

	s := arena.StringSlice(length)
	for i := range length {
		s[i] = mem.String(fb.TypeParams(i))
	}
	return s
}
