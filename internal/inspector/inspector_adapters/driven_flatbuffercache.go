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

package inspector_adapters

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"sync"

	flatbuffers "github.com/google/flatbuffers/go"
	"piko.sh/piko/internal/fbs"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/inspector/inspector_schema"
	"piko.sh/piko/internal/inspector/inspector_schema/inspector_schema_gen"
	"piko.sh/piko/internal/mem"
	"piko.sh/piko/wdk/safeconv"
	"piko.sh/piko/wdk/safedisk"
)

const (
	// defaultDirPerm is the permission mode for creating cache directories.
	defaultDirPerm = 0755

	// defaultFilePerm is the permission mode for cache files.
	defaultFilePerm = 0644

	// initialBuilderSize is the starting buffer size in bytes for FlatBuffer
	// builders. Sized at 256 KiB to avoid repeated growByteBuffer reallocations
	// when serialising large type data.
	initialBuilderSize = 256 * 1024

	// flatbufferVectorAlignment is the byte alignment for FlatBuffer vectors.
	flatbufferVectorAlignment = 4
)

// builderPool reuses FlatBuffer Builder instances to reduce allocation pressure
// during type data serialisation.
var builderPool = sync.Pool{
	New: func() any {
		return flatbuffers.NewBuilder(initialBuilderSize)
	},
}

// FlatBufferCache provides high-performance binary caching of TypeData using
// FlatBuffers. It implements TypeDataProvider with parallel unpacking for large
// datasets and zero-copy string reading.
type FlatBufferCache struct {
	// sandbox provides sandboxed file system operations for cache access.
	sandbox safedisk.Sandbox
}

var _ inspector_domain.TypeDataProvider = (*FlatBufferCache)(nil)

// NewFlatBufferCache creates a new FlatBufferCache with the given sandbox.
// The sandbox root is the cache directory.
//
// Takes sandbox (safedisk.Sandbox) which provides sandboxed filesystem access
// to the cache directory.
//
// Returns *FlatBufferCache which is ready for use.
func NewFlatBufferCache(sandbox safedisk.Sandbox) *FlatBufferCache {
	return &FlatBufferCache{sandbox: sandbox}
}

// GetTypeData retrieves cached TypeData for the given cache key.
//
// Takes cacheKey (string) which identifies the cached type data to retrieve.
//
// Returns *inspector_dto.TypeData which contains the deserialised type data.
// Returns error when the cache is missing, corrupt, or has a schema version
// mismatch.
func (fc *FlatBufferCache) GetTypeData(_ context.Context, cacheKey string) (*inspector_dto.TypeData, error) {
	if fc.sandbox == nil || cacheKey == "" {
		return nil, errors.New("flatbuffer cache provider requires a sandbox and key")
	}

	fileName := fmt.Sprintf("typedata-%s.bin", cacheKey)
	data, err := fc.sandbox.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("cache miss or read error for key %s: %w", cacheKey, err)
	}

	payload, err := inspector_schema.Unpack(data)
	if err != nil {
		_ = fc.sandbox.Remove(fileName)
		if errors.Is(err, fbs.ErrSchemaVersionMismatch) {
			return nil, fmt.Errorf("cache schema version mismatch for key %s, invalidated", cacheKey)
		}
		return nil, fmt.Errorf("failed to unpack versioned cache for key %s: %w", cacheKey, err)
	}

	fbTypeData := inspector_schema_gen.GetRootAsTypeData(payload, 0)
	if fbTypeData == nil {
		_ = fc.sandbox.Remove(fileName)
		return nil, fmt.Errorf("failed to parse corrupt cache file for key %s", cacheKey)
	}

	return unpackTypeData(fbTypeData), nil
}

// SaveTypeData serialises and stores TypeData to the cache with the given key.
//
// Takes cacheKey (string) which identifies the cache entry.
// Takes data (*inspector_dto.TypeData) which contains the type data to store.
//
// Returns error when the cache is not initialised, the key is empty,
// the directory cannot be created, or the file write fails.
//
// The write is atomic using a temporary file and rename. The output includes
// a 32-byte schema hash prefix for automatic cache invalidation.
func (fc *FlatBufferCache) SaveTypeData(_ context.Context, cacheKey string, data *inspector_dto.TypeData) error {
	if fc.sandbox == nil || cacheKey == "" {
		return errors.New("flatbuffer cache saver requires a sandbox and key")
	}

	if err := fc.sandbox.MkdirAll(".", defaultDirPerm); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	builder, ok := builderPool.Get().(*flatbuffers.Builder)
	if !ok {
		builder = flatbuffers.NewBuilder(initialBuilderSize)
	}
	builder.Reset()
	rootOffset := packTypeData(builder, data)
	builder.Finish(rootOffset)

	payload := builder.FinishedBytes()
	versionedData := make([]byte, fbs.PackedSize(len(payload)))
	inspector_schema.PackInto(versionedData, payload)

	builderPool.Put(builder)

	fileName := fmt.Sprintf("typedata-%s.bin", cacheKey)
	if err := fc.sandbox.WriteFileAtomic(fileName, versionedData, defaultFilePerm); err != nil {
		return fmt.Errorf("failed to write cache file atomically: %w", err)
	}

	return nil
}

// InvalidateCache removes the cached TypeData for the given key.
//
// Takes cacheKey (string) which identifies the cache entry to remove.
//
// Returns error when the cache directory or key is empty, or when the file
// cannot be removed.
func (fc *FlatBufferCache) InvalidateCache(_ context.Context, cacheKey string) error {
	if fc.sandbox == nil || cacheKey == "" {
		return errors.New("flatbuffer cache invalidator requires a sandbox and key")
	}

	fileName := fmt.Sprintf("typedata-%s.bin", cacheKey)
	err := fc.sandbox.Remove(fileName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove cache file %s: %w", fileName, err)
	}
	return nil
}

// ClearCache removes all cached data from the cache directory.
//
// Returns error when the sandbox is not set or the directory cannot be removed.
func (fc *FlatBufferCache) ClearCache(_ context.Context) error {
	if fc.sandbox == nil {
		return errors.New("flatbuffer cache requires a sandbox to clear")
	}

	if err := fc.sandbox.RemoveAll("."); err != nil {
		return fmt.Errorf("failed to clear cache directory: %w", err)
	}
	return nil
}

// EncodeTypeDataToFBS encodes TypeData to FlatBuffers binary format. Use it
// to generate pre-bundled stdlib data for WASM builds.
//
// Takes data (*inspector_dto.TypeData) which contains the type information to
// encode.
//
// Returns []byte which contains the FlatBuffers binary representation.
func EncodeTypeDataToFBS(data *inspector_dto.TypeData) []byte {
	builder, ok := builderPool.Get().(*flatbuffers.Builder)
	if !ok {
		builder = flatbuffers.NewBuilder(initialBuilderSize)
	}
	builder.Reset()
	rootOffset := packTypeData(builder, data)
	builder.Finish(rootOffset)
	result := make([]byte, len(builder.FinishedBytes()))
	copy(result, builder.FinishedBytes())
	builderPool.Put(builder)
	return result
}

// DecodeTypeDataFromFBS decodes FlatBuffers binary data into a TypeData
// struct. Use it to load pre-bundled standard library data in WASM builds.
//
// Takes data ([]byte) which contains the FlatBuffers binary data to parse.
//
// Returns *inspector_dto.TypeData which holds the decoded type information.
// Returns error when the data is empty or cannot be parsed as FlatBuffers.
func DecodeTypeDataFromFBS(data []byte) (*inspector_dto.TypeData, error) {
	if len(data) == 0 {
		return nil, errors.New("empty FlatBuffer data")
	}
	fbTypeData := inspector_schema_gen.GetRootAsTypeData(data, 0)
	if fbTypeData == nil {
		return nil, errors.New("failed to parse FlatBuffer data")
	}
	return unpackTypeData(fbTypeData), nil
}

// packTypeData writes type data into a FlatBuffers format.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes data (*inspector_dto.TypeData) which contains the type information.
//
// Returns flatbuffers.UOffsetT which is the offset of the written data.
func packTypeData(b *flatbuffers.Builder, data *inspector_dto.TypeData) flatbuffers.UOffsetT {
	packagesOffset := packMap(b, data.Packages, packPackageEntry)
	fileToPackageOffset := packMap(b, data.FileToPackage, packFileToPackageEntry)

	inspector_schema_gen.TypeDataStart(b)
	inspector_schema_gen.TypeDataAddPackages(b, packagesOffset)
	inspector_schema_gen.TypeDataAddFileToPackage(b, fileToPackageOffset)
	return inspector_schema_gen.TypeDataEnd(b)
}

// packFileToPackageEntry creates a FlatBuffers entry that maps a file to its
// package.
//
// Takes b (*flatbuffers.Builder) which builds the FlatBuffers binary.
// Takes key (string) which is the file path.
// Takes value (string) which is the package name.
//
// Returns flatbuffers.UOffsetT which is the offset of the entry in the buffer.
func packFileToPackageEntry(b *flatbuffers.Builder, key string, value string) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := b.CreateString(value)
	inspector_schema_gen.FileToPackageEntryStart(b)
	inspector_schema_gen.FileToPackageEntryAddKey(b, keyOffset)
	inspector_schema_gen.FileToPackageEntryAddValue(b, valueOffset)
	return inspector_schema_gen.FileToPackageEntryEnd(b)
}

// packPackageEntry packs a package entry into the FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the buffer to write to.
// Takes key (string) which is the package path.
// Takes value (*inspector_dto.Package) which is the package data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed entry.
func packPackageEntry(b *flatbuffers.Builder, key string, value *inspector_dto.Package) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packPackage(b, value)
	inspector_schema_gen.PackageEntryStart(b)
	inspector_schema_gen.PackageEntryAddKey(b, keyOffset)
	inspector_schema_gen.PackageEntryAddValue(b, valueOffset)
	return inspector_schema_gen.PackageEntryEnd(b)
}

// packFileImportsEntry packs a file imports entry into the FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the buffer to write to.
// Takes key (string) which is the file path for this imports entry.
// Takes value (map[string]string) which maps import paths to their aliases.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed entry.
func packFileImportsEntry(b *flatbuffers.Builder, key string, value map[string]string) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packFileImportMap(b, value)
	inspector_schema_gen.FileImportsEntryStart(b)
	inspector_schema_gen.FileImportsEntryAddKey(b, keyOffset)
	inspector_schema_gen.FileImportsEntryAddValue(b, valueOffset)
	return inspector_schema_gen.FileImportsEntryEnd(b)
}

// packFileImportMap converts an import alias-to-path map into a FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the buffer to write to.
// Takes innerMap (map[string]string) which maps import aliases to their paths.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed map.
func packFileImportMap(b *flatbuffers.Builder, innerMap map[string]string) flatbuffers.UOffsetT {
	entriesOffset := packMap(b, innerMap, packAliasToPathEntry)
	inspector_schema_gen.FileImportMapStart(b)
	inspector_schema_gen.FileImportMapAddEntries(b, entriesOffset)
	return inspector_schema_gen.FileImportMapEnd(b)
}

// packAliasToPathEntry packs an alias-to-path mapping into a FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the builder to write the entry to.
// Takes key (string) which is the alias name.
// Takes value (string) which is the file path the alias points to.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed entry.
func packAliasToPathEntry(b *flatbuffers.Builder, key string, value string) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := b.CreateString(value)
	inspector_schema_gen.AliasToPathEntryStart(b)
	inspector_schema_gen.AliasToPathEntryAddKey(b, keyOffset)
	inspector_schema_gen.AliasToPathEntryAddValue(b, valueOffset)
	return inspector_schema_gen.AliasToPathEntryEnd(b)
}

// packPackage packs a Package into a FlatBuffer representation.
//
// Takes b (*flatbuffers.Builder) which builds the FlatBuffer output.
// Takes pkg (*inspector_dto.Package) which contains the package data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed Package.
func packPackage(b *flatbuffers.Builder, pkg *inspector_dto.Package) flatbuffers.UOffsetT {
	pathOffset := b.CreateString(pkg.Path)
	nameOffset := b.CreateString(pkg.Name)
	versionOffset := b.CreateString(pkg.Version)
	fileImportsOffset := packMap(b, pkg.FileImports, packFileImportsEntry)
	namedTypesOffset := packMap(b, pkg.NamedTypes, packNamedTypeEntry)
	funcsOffset := packMap(b, pkg.Funcs, packFuncEntry)
	variablesOffset := packMap(b, pkg.Variables, packVariableEntry)

	inspector_schema_gen.PackageStart(b)
	inspector_schema_gen.PackageAddPath(b, pathOffset)
	inspector_schema_gen.PackageAddName(b, nameOffset)
	inspector_schema_gen.PackageAddVersion(b, versionOffset)
	inspector_schema_gen.PackageAddFileImports(b, fileImportsOffset)
	inspector_schema_gen.PackageAddNamedTypes(b, namedTypesOffset)
	inspector_schema_gen.PackageAddFunctions(b, funcsOffset)
	inspector_schema_gen.PackageAddVariables(b, variablesOffset)
	return inspector_schema_gen.PackageEnd(b)
}

// packNamedTypeEntry packs a key-value pair into a FlatBuffers table.
//
// Takes b (*flatbuffers.Builder) which is the buffer to write to.
// Takes key (string) which is the name of the entry.
// Takes value (*inspector_dto.Type) which is the type data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed entry.
func packNamedTypeEntry(b *flatbuffers.Builder, key string, value *inspector_dto.Type) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packType(b, value)
	inspector_schema_gen.NamedTypeEntryStart(b)
	inspector_schema_gen.NamedTypeEntryAddKey(b, keyOffset)
	inspector_schema_gen.NamedTypeEntryAddValue(b, valueOffset)
	return inspector_schema_gen.NamedTypeEntryEnd(b)
}

// packFuncEntry packs a function entry into the FlatBuffers builder.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes key (string) which is the lookup key for the entry.
// Takes value (*inspector_dto.Function) which is the function data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed entry.
func packFuncEntry(b *flatbuffers.Builder, key string, value *inspector_dto.Function) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packFunction(b, value)
	inspector_schema_gen.FunctionEntryStart(b)
	inspector_schema_gen.FunctionEntryAddKey(b, keyOffset)
	inspector_schema_gen.FunctionEntryAddValue(b, valueOffset)
	return inspector_schema_gen.FunctionEntryEnd(b)
}

// packVariableEntry packs a variable entry into the FlatBuffers builder.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes key (string) which is the lookup key for the entry.
// Takes value (*inspector_dto.Variable) which is the variable data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed entry.
func packVariableEntry(b *flatbuffers.Builder, key string, value *inspector_dto.Variable) flatbuffers.UOffsetT {
	keyOffset := b.CreateString(key)
	valueOffset := packVariable(b, value)
	inspector_schema_gen.VariableEntryStart(b)
	inspector_schema_gen.VariableEntryAddKey(b, keyOffset)
	inspector_schema_gen.VariableEntryAddValue(b, valueOffset)
	return inspector_schema_gen.VariableEntryEnd(b)
}

// packVariable packs a variable into the FlatBuffer format.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes v (*inspector_dto.Variable) which contains the variable data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed variable.
func packVariable(b *flatbuffers.Builder, v *inspector_dto.Variable) flatbuffers.UOffsetT {
	nameOffset := b.CreateString(v.Name)
	typeStringOffset := b.CreateString(v.TypeString)
	underlyingTypeStringOffset := b.CreateString(v.UnderlyingTypeString)
	definedInFilePathOffset := b.CreateString(v.DefinedInFilePath)
	compositePartsOffset := packSlice(b, v.CompositeParts, packCompositePart)

	inspector_schema_gen.VariableStart(b)
	inspector_schema_gen.VariableAddName(b, nameOffset)
	inspector_schema_gen.VariableAddTypeString(b, typeStringOffset)
	inspector_schema_gen.VariableAddUnderlyingTypeString(b, underlyingTypeStringOffset)
	inspector_schema_gen.VariableAddDefinedInFilePath(b, definedInFilePathOffset)
	inspector_schema_gen.VariableAddCompositeParts(b, compositePartsOffset)
	inspector_schema_gen.VariableAddDefinitionLine(b, safeconv.IntToInt32(v.DefinitionLine))
	inspector_schema_gen.VariableAddDefinitionColumn(b, safeconv.IntToInt32(v.DefinitionColumn))
	inspector_schema_gen.VariableAddCompositeType(b, inspector_schema_gen.CompositeType(safeconv.IntToUint8(int(v.CompositeType))))
	inspector_schema_gen.VariableAddIsConst(b, v.IsConst)
	return inspector_schema_gen.VariableEnd(b)
}

// packType converts a Type into its FlatBuffers form.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes t (*inspector_dto.Type) which holds the type data to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed type.
func packType(b *flatbuffers.Builder, t *inspector_dto.Type) flatbuffers.UOffsetT {
	nameOffset := b.CreateString(t.Name)
	packagePathOffset := b.CreateString(t.PackagePath)
	definedInFilePathOffset := b.CreateString(t.DefinedInFilePath)
	typeStringOffset := b.CreateString(t.TypeString)
	underlyingTypeStringOffset := b.CreateString(t.UnderlyingTypeString)
	fieldsOffset := packSlice(b, t.Fields, packField)
	methodsOffset := packSlice(b, t.Methods, packMethod)
	typeParamsOffset := packStringSlice(b, t.TypeParams)

	inspector_schema_gen.TypeStart(b)
	inspector_schema_gen.TypeAddName(b, nameOffset)
	inspector_schema_gen.TypeAddPackagePath(b, packagePathOffset)
	inspector_schema_gen.TypeAddDefinedInFilePath(b, definedInFilePathOffset)
	inspector_schema_gen.TypeAddTypeString(b, typeStringOffset)
	inspector_schema_gen.TypeAddUnderlyingTypeString(b, underlyingTypeStringOffset)
	inspector_schema_gen.TypeAddFields(b, fieldsOffset)
	inspector_schema_gen.TypeAddMethods(b, methodsOffset)
	inspector_schema_gen.TypeAddTypeParams(b, typeParamsOffset)
	inspector_schema_gen.TypeAddStringability(b, inspector_schema_gen.StringabilityMethod(safeconv.IntToUint8(int(t.Stringability))))
	inspector_schema_gen.TypeAddIsAlias(b, t.IsAlias)
	inspector_schema_gen.TypeAddDefinitionLine(b, safeconv.IntToInt32(t.DefinitionLine))
	inspector_schema_gen.TypeAddDefinitionColumn(b, safeconv.IntToInt32(t.DefinitionColumn))
	return inspector_schema_gen.TypeEnd(b)
}

// packField writes a field to the FlatBuffers builder.
//
// Takes b (*flatbuffers.Builder) which collects the output data.
// Takes f (*inspector_dto.Field) which holds the field data to write.
//
// Returns flatbuffers.UOffsetT which is the offset of the written field.
func packField(b *flatbuffers.Builder, f *inspector_dto.Field) flatbuffers.UOffsetT {
	nameOffset := b.CreateString(f.Name)
	typeStringOffset := b.CreateString(f.TypeString)
	underlyingTypeStringOffset := b.CreateString(f.UnderlyingTypeString)
	rawTagOffset := b.CreateString(f.RawTag)
	pkgPathOffset := b.CreateString(f.PackagePath)
	declaringPackagePathOffset := b.CreateString(f.DeclaringPackagePath)
	declaringTypeNameOffset := b.CreateString(f.DeclaringTypeName)
	compositePartsOffset := packSlice(b, f.CompositeParts, packCompositePart)
	definitionFilePathOffset := b.CreateString(f.DefinitionFilePath)

	inspector_schema_gen.FieldStart(b)
	inspector_schema_gen.FieldAddName(b, nameOffset)
	inspector_schema_gen.FieldAddTypeString(b, typeStringOffset)
	inspector_schema_gen.FieldAddUnderlyingTypeString(b, underlyingTypeStringOffset)
	inspector_schema_gen.FieldAddIsEmbedded(b, f.IsEmbedded)
	inspector_schema_gen.FieldAddRawTag(b, rawTagOffset)
	inspector_schema_gen.FieldAddPackagePath(b, pkgPathOffset)
	inspector_schema_gen.FieldAddIsGenericPlaceholder(b, f.IsGenericPlaceholder)
	inspector_schema_gen.FieldAddDeclaringPackagePath(b, declaringPackagePathOffset)
	inspector_schema_gen.FieldAddDeclaringTypeName(b, declaringTypeNameOffset)
	inspector_schema_gen.FieldAddCompositeParts(b, compositePartsOffset)
	inspector_schema_gen.FieldAddCompositeType(b, inspector_schema_gen.CompositeType(safeconv.IntToUint8(int(f.CompositeType))))
	inspector_schema_gen.FieldAddIsUnderlyingInternalType(b, f.IsUnderlyingInternalType)
	inspector_schema_gen.FieldAddIsUnderlyingPrimitive(b, f.IsUnderlyingPrimitive)
	inspector_schema_gen.FieldAddIsInternalType(b, f.IsInternalType)
	inspector_schema_gen.FieldAddIsAlias(b, f.IsAlias)
	inspector_schema_gen.FieldAddDefinitionFilePath(b, definitionFilePathOffset)
	inspector_schema_gen.FieldAddDefinitionLine(b, safeconv.IntToInt32(f.DefinitionLine))
	inspector_schema_gen.FieldAddDefinitionColumn(b, safeconv.IntToInt32(f.DefinitionColumn))
	return inspector_schema_gen.FieldEnd(b)
}

// packMethod converts a method DTO into a FlatBuffers Method table.
//
// Takes b (*flatbuffers.Builder) which builds the FlatBuffers output.
// Takes m (*inspector_dto.Method) which holds the method data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed Method.
func packMethod(b *flatbuffers.Builder, m *inspector_dto.Method) flatbuffers.UOffsetT {
	nameOffset := b.CreateString(m.Name)
	typeStringOffset := b.CreateString(m.TypeString)
	underlyingTypeStringOffset := b.CreateString(m.UnderlyingTypeString)
	sigOffset := packFunctionSignature(b, &m.Signature)
	declaringPackagePathOffset := b.CreateString(m.DeclaringPackagePath)
	declaringTypeNameOffset := b.CreateString(m.DeclaringTypeName)
	definitionFilePathOffset := b.CreateString(m.DefinitionFilePath)

	inspector_schema_gen.MethodStart(b)
	inspector_schema_gen.MethodAddName(b, nameOffset)
	inspector_schema_gen.MethodAddTypeString(b, typeStringOffset)
	inspector_schema_gen.MethodAddUnderlyingTypeString(b, underlyingTypeStringOffset)
	inspector_schema_gen.MethodAddSignature(b, sigOffset)
	inspector_schema_gen.MethodAddIsPointerReceiver(b, m.IsPointerReceiver)
	inspector_schema_gen.MethodAddDeclaringPackagePath(b, declaringPackagePathOffset)
	inspector_schema_gen.MethodAddDeclaringTypeName(b, declaringTypeNameOffset)
	inspector_schema_gen.MethodAddDefinitionFilePath(b, definitionFilePathOffset)
	inspector_schema_gen.MethodAddDefinitionLine(b, safeconv.IntToInt32(m.DefinitionLine))
	inspector_schema_gen.MethodAddDefinitionColumn(b, safeconv.IntToInt32(m.DefinitionColumn))
	return inspector_schema_gen.MethodEnd(b)
}

// packFunction packs a function into the FlatBuffer format.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes f (*inspector_dto.Function) which contains the function data to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed function.
func packFunction(b *flatbuffers.Builder, f *inspector_dto.Function) flatbuffers.UOffsetT {
	nameOffset := b.CreateString(f.Name)
	typeStringOffset := b.CreateString(f.TypeString)
	underlyingTypeStringOffset := b.CreateString(f.UnderlyingTypeString)
	sigOffset := packFunctionSignature(b, &f.Signature)
	definitionFilePathOffset := b.CreateString(f.DefinitionFilePath)

	inspector_schema_gen.FunctionStart(b)
	inspector_schema_gen.FunctionAddName(b, nameOffset)
	inspector_schema_gen.FunctionAddTypeString(b, typeStringOffset)
	inspector_schema_gen.FunctionAddUnderlyingTypeString(b, underlyingTypeStringOffset)
	inspector_schema_gen.FunctionAddSignature(b, sigOffset)
	inspector_schema_gen.FunctionAddDefinitionFilePath(b, definitionFilePathOffset)
	inspector_schema_gen.FunctionAddDefinitionLine(b, safeconv.IntToInt32(f.DefinitionLine))
	inspector_schema_gen.FunctionAddDefinitionColumn(b, safeconv.IntToInt32(f.DefinitionColumn))
	return inspector_schema_gen.FunctionEnd(b)
}

// packFunctionSignature writes a function signature to a FlatBuffer.
//
// Takes b (*flatbuffers.Builder) which is the buffer to write to.
// Takes sig (*inspector_dto.FunctionSignature) which holds the parameters and
// results to write.
//
// Returns flatbuffers.UOffsetT which is the offset of the written signature.
func packFunctionSignature(b *flatbuffers.Builder, sig *inspector_dto.FunctionSignature) flatbuffers.UOffsetT {
	paramsOffset := packStringSlice(b, sig.Params)
	resultsOffset := packStringSlice(b, sig.Results)
	paramNamesOffset := packStringSlice(b, sig.ParamNames)
	inspector_schema_gen.FunctionSignatureStart(b)
	inspector_schema_gen.FunctionSignatureAddParams(b, paramsOffset)
	inspector_schema_gen.FunctionSignatureAddResults(b, resultsOffset)
	inspector_schema_gen.FunctionSignatureAddParamNames(b, paramNamesOffset)
	return inspector_schema_gen.FunctionSignatureEnd(b)
}

// packCompositePart converts a composite part into FlatBuffers format.
//
// Takes b (*flatbuffers.Builder) which builds the output buffer.
// Takes cp (*inspector_dto.CompositePart) which holds the part data to convert.
//
// Returns flatbuffers.UOffsetT which is the offset of the converted part.
func packCompositePart(b *flatbuffers.Builder, cp *inspector_dto.CompositePart) flatbuffers.UOffsetT {
	typeOffset := b.CreateString(cp.Type)
	typeStringOffset := b.CreateString(cp.TypeString)
	roleOffset := b.CreateString(cp.Role)
	underlyingTypeStringOffset := b.CreateString(cp.UnderlyingTypeString)
	pkgPathOffset := b.CreateString(cp.PackagePath)
	compositePartsOffset := packSlice(b, cp.CompositeParts, packCompositePart)

	inspector_schema_gen.CompositePartStart(b)
	inspector_schema_gen.CompositePartAddType(b, typeOffset)
	inspector_schema_gen.CompositePartAddTypeString(b, typeStringOffset)
	inspector_schema_gen.CompositePartAddRole(b, roleOffset)
	inspector_schema_gen.CompositePartAddUnderlyingTypeString(b, underlyingTypeStringOffset)
	inspector_schema_gen.CompositePartAddPackagePath(b, pkgPathOffset)
	inspector_schema_gen.CompositePartAddCompositeParts(b, compositePartsOffset)
	inspector_schema_gen.CompositePartAddCompositeType(b, inspector_schema_gen.CompositeType(safeconv.IntToUint8(int(cp.CompositeType))))
	inspector_schema_gen.CompositePartAddIndex(b, safeconv.IntToInt32(cp.Index))
	inspector_schema_gen.CompositePartAddIsInternalType(b, cp.IsInternalType)
	inspector_schema_gen.CompositePartAddIsUnderlyingInternalType(b, cp.IsUnderlyingInternalType)
	inspector_schema_gen.CompositePartAddIsGenericPlaceholder(b, cp.IsGenericPlaceholder)
	inspector_schema_gen.CompositePartAddIsAlias(b, cp.IsAlias)
	inspector_schema_gen.CompositePartAddIsUnderlyingPrimitive(b, cp.IsUnderlyingPrimitive)
	return inspector_schema_gen.CompositePartEnd(b)
}

// unpackTypeData converts a FlatBuffer TypeData into its DTO form.
//
// Takes fb (*inspector_schema_gen_gen.TypeData) which is the serialised type data.
//
// Returns *inspector_dto.TypeData which contains the unpacked packages and
// file-to-package mappings.
func unpackTypeData(fb *inspector_schema_gen.TypeData) *inspector_dto.TypeData {
	counts := countEntities(fb)
	arena := newUnpackArena(counts)

	return &inspector_dto.TypeData{
		Packages:      unpackPackages(fb, arena),
		FileToPackage: unpackMap(fb.FileToPackageLength(), fb.FileToPackage, unpackFileToPackageEntry),
	}
}

// unpackFileToPackageEntry extracts the key and value strings from a file to
// package entry.
//
// Takes fb (*inspector_schema_gen.FileToPackageEntry) which holds the packed
// entry data.
//
// Returns key (string) which is the file path.
// Returns value (string) which is the package name.
func unpackFileToPackageEntry(fb *inspector_schema_gen.FileToPackageEntry) (key, value string) {
	return mem.String(fb.Key()), mem.String(fb.Value())
}

// unpackFunctionSignature converts a FlatBuffer function signature to a DTO.
//
// Takes fb (*inspector_schema_gen_gen.FunctionSignature) which is the FlatBuffer
// representation to unpack.
//
// Returns inspector_dto.FunctionSignature which contains the extracted
// parameter and result type strings.
func unpackFunctionSignature(fb *inspector_schema_gen.FunctionSignature) inspector_dto.FunctionSignature {
	paramsLen := fb.ParamsLength()
	resultsLen := fb.ResultsLength()
	paramNamesLen := fb.ParamNamesLength()

	if paramsLen == 0 && resultsLen == 0 && paramNamesLen == 0 {
		return inspector_dto.FunctionSignature{}
	}

	total := paramsLen + resultsLen + paramNamesLen
	backing := make([]string, total)

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

// packMap writes a map to a FlatBuffers vector with sorted keys.
//
// When the map is empty, returns 0 without writing to the builder.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes m (map[K]V) which is the map to write.
// Takes packer (func(...)) which converts each key-value pair to an offset.
//
// Returns flatbuffers.UOffsetT which is the offset of the packed vector.
func packMap[K comparable, V any](b *flatbuffers.Builder, m map[K]V, packer func(*flatbuffers.Builder, K, V) flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(m) == 0 {
		return 0
	}

	if len(m) == 1 {
		for k, v := range m {
			offset := packer(b, k, v)
			return createVector(b, []flatbuffers.UOffsetT{offset})
		}
	}

	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b K) int {
		switch va := any(a).(type) {
		case string:
			return cmp.Compare(va, any(b).(string))
		case int:
			return cmp.Compare(va, any(b).(int))
		default:
			return cmp.Compare(fmt.Sprintf("%v", a), fmt.Sprintf("%v", b))
		}
	})

	offsets := make([]flatbuffers.UOffsetT, len(keys))
	for i, k := range keys {
		offsets[i] = packer(b, k, m[k])
	}
	return createVector(b, offsets)
}

// packSlice packs a slice of items into a FlatBuffers vector.
//
// When the slice is empty, returns 0 without creating a vector.
//
// Takes b (*flatbuffers.Builder) which builds the FlatBuffers buffer.
// Takes s ([]T) which contains the items to pack.
// Takes packer (func(...)) which converts each item to an offset.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector.
func packSlice[T any](b *flatbuffers.Builder, s []T, packer func(*flatbuffers.Builder, T) flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	if len(s) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(s))
	for i, item := range s {
		offsets[i] = packer(b, item)
	}
	return createVector(b, offsets)
}

// packStringSlice packs a slice of strings into a FlatBuffers vector.
//
// When the slice is empty, returns 0 without creating any buffer entries.
//
// Takes b (*flatbuffers.Builder) which is the builder to write strings into.
// Takes s ([]string) which contains the strings to pack.
//
// Returns flatbuffers.UOffsetT which is the offset of the created vector.
func packStringSlice(b *flatbuffers.Builder, s []string) flatbuffers.UOffsetT {
	if len(s) == 0 {
		return 0
	}
	offsets := make([]flatbuffers.UOffsetT, len(s))
	for i, str := range s {
		offsets[i] = b.CreateString(str)
	}
	return createVector(b, offsets)
}

// createVector builds a FlatBuffers vector from the given offsets.
//
// Takes b (*flatbuffers.Builder) which is the builder to write to.
// Takes offsets ([]flatbuffers.UOffsetT) which holds the element offsets.
//
// Returns flatbuffers.UOffsetT which is the offset of the new vector.
func createVector(b *flatbuffers.Builder, offsets []flatbuffers.UOffsetT) flatbuffers.UOffsetT {
	b.StartVector(flatbufferVectorAlignment, len(offsets), flatbufferVectorAlignment)
	for i := len(offsets) - 1; i >= 0; i-- {
		b.PrependUOffsetT(offsets[i])
	}
	return b.EndVector(len(offsets))
}

// unpackMap builds a map from a list of items using the given functions.
//
// Takes length (int) which is the number of items to process.
// Takes getItem (func(*T, int) bool) which gets an item at the given index
// into the pointer and returns true if it worked.
// Takes unpacker (func(*T) (K, V)) which gets a key and value from an item.
//
// Returns map[K]V which holds the key-value pairs, or nil if length is zero.
func unpackMap[T any, K comparable, V any](length int, getItem func(*T, int) bool, unpacker func(*T) (K, V)) map[K]V {
	if length == 0 {
		return nil
	}
	m := make(map[K]V, length)
	var item T
	for i := range length {
		if getItem(&item, i) {
			k, v := unpacker(&item)
			m[k] = v
		}
	}
	return m
}
