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
	"bytes"
	"context"
	"errors"
	"io/fs"
	"testing"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
	"piko.sh/piko/internal/inspector/inspector_schema/inspector_schema_gen"
	"piko.sh/piko/wdk/safedisk"
)

func testTypeData() *inspector_dto.TypeData {
	return &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"example.com/foo": {
				Path:    "example.com/foo",
				Name:    "foo",
				Version: "v1.2.3",
				FileImports: map[string]map[string]string{
					"foo.go": {
						"fmt":     "fmt",
						"myalias": "example.com/bar",
					},
				},
				NamedTypes: map[string]*inspector_dto.Type{
					"Widget": {
						Name:                 "Widget",
						PackagePath:          "example.com/foo",
						DefinedInFilePath:    "foo.go",
						TypeString:           "struct{ID int; Name string}",
						UnderlyingTypeString: "struct{ID int; Name string}",
						Fields: []*inspector_dto.Field{
							{
								Name:                     "ID",
								TypeString:               "int",
								UnderlyingTypeString:     "int",
								IsEmbedded:               false,
								RawTag:                   `json:"id"`,
								PackagePath:              "",
								IsGenericPlaceholder:     false,
								DeclaringPackagePath:     "example.com/foo",
								DeclaringTypeName:        "Widget",
								CompositeType:            inspector_dto.CompositeTypeNone,
								IsUnderlyingInternalType: false,
								IsUnderlyingPrimitive:    true,
								IsInternalType:           true,
								IsAlias:                  false,
								DefinitionFilePath:       "foo.go",
								DefinitionLine:           10,
								DefinitionColumn:         2,
							},
							{
								Name:                     "Items",
								TypeString:               "map[string]*Bar",
								UnderlyingTypeString:     "map[string]*Bar",
								IsEmbedded:               true,
								PackagePath:              "example.com/foo",
								DeclaringPackagePath:     "example.com/foo",
								DeclaringTypeName:        "Widget",
								CompositeType:            inspector_dto.CompositeTypeMap,
								IsUnderlyingInternalType: true,
								IsAlias:                  true,
								DefinitionFilePath:       "foo.go",
								DefinitionLine:           11,
								DefinitionColumn:         2,
								CompositeParts: []*inspector_dto.CompositePart{
									{
										Type:                     "string",
										TypeString:               "string",
										Role:                     "key",
										UnderlyingTypeString:     "string",
										CompositeType:            inspector_dto.CompositeTypeNone,
										Index:                    0,
										IsInternalType:           true,
										IsUnderlyingPrimitive:    true,
										IsUnderlyingInternalType: false,
									},
									{
										Type:                 "*Bar",
										TypeString:           "*Bar",
										Role:                 "value",
										UnderlyingTypeString: "*Bar",
										PackagePath:          "example.com/foo",
										CompositeType:        inspector_dto.CompositeTypePointer,
										Index:                1,
										IsInternalType:       false,
										IsGenericPlaceholder: true,
										IsAlias:              true,
										CompositeParts: []*inspector_dto.CompositePart{
											{
												Type:                 "Bar",
												TypeString:           "Bar",
												Role:                 "elem",
												UnderlyingTypeString: "Bar",
												PackagePath:          "example.com/foo",
												CompositeType:        inspector_dto.CompositeTypeNone,
												Index:                0,
												IsInternalType:       false,
											},
										},
									},
								},
							},
						},
						Methods: []*inspector_dto.Method{
							{
								Name:                 "String",
								TypeString:           "func() string",
								UnderlyingTypeString: "func() string",
								Signature: inspector_dto.FunctionSignature{
									Params:  []string{},
									Results: []string{"string"},
								},
								IsPointerReceiver:    false,
								DeclaringPackagePath: "example.com/foo",
								DeclaringTypeName:    "Widget",
								DefinitionFilePath:   "foo.go",
								DefinitionLine:       20,
								DefinitionColumn:     1,
							},
							{
								Name:                 "Update",
								TypeString:           "func(name string) error",
								UnderlyingTypeString: "func(name string) error",
								Signature: inspector_dto.FunctionSignature{
									Params:     []string{"string"},
									ParamNames: []string{"name"},
									Results:    []string{"error"},
								},
								IsPointerReceiver:    true,
								DeclaringPackagePath: "example.com/foo",
								DeclaringTypeName:    "Widget",
								DefinitionFilePath:   "foo.go",
								DefinitionLine:       25,
								DefinitionColumn:     1,
							},
						},
						TypeParams:       []string{"T", "U"},
						Stringability:    inspector_dto.StringableViaStringer,
						IsAlias:          false,
						DefinitionLine:   5,
						DefinitionColumn: 6,
					},
				},
				Funcs: map[string]*inspector_dto.Function{
					"NewWidget": {
						Name:                 "NewWidget",
						TypeString:           "func(id int) *Widget",
						UnderlyingTypeString: "func(id int) *Widget",
						Signature: inspector_dto.FunctionSignature{
							Params:  []string{"int"},
							Results: []string{"*Widget"},
						},
						DefinitionFilePath: "foo.go",
						DefinitionLine:     30,
						DefinitionColumn:   1,
					},
				},
			},
			"example.com/bar": {
				Path:    "example.com/bar",
				Name:    "bar",
				Version: "v0.1.0",
			},
		},
		FileToPackage: map[string]string{
			"foo.go": "example.com/foo",
			"bar.go": "example.com/bar",
		},
	}
}

func TestFlatBufferRoundTrip(t *testing.T) {
	original := testTypeData()

	encoded := EncodeTypeDataToFBS(original)
	require.NotEmpty(t, encoded, "encoded bytes should not be empty")

	decoded, err := DecodeTypeDataFromFBS(encoded)
	require.NoError(t, err)

	assert.Equal(t, original.FileToPackage, decoded.FileToPackage)
	require.Len(t, decoded.Packages, len(original.Packages))

	for packagePath, originalPackage := range original.Packages {
		decPackage, ok := decoded.Packages[packagePath]
		require.True(t, ok, "missing package: %s", packagePath)

		assert.Equal(t, originalPackage.Path, decPackage.Path)
		assert.Equal(t, originalPackage.Name, decPackage.Name)
		assert.Equal(t, originalPackage.Version, decPackage.Version)
		assert.Equal(t, originalPackage.FileImports, decPackage.FileImports)

		for typeName, originalType := range originalPackage.NamedTypes {
			decType, ok := decPackage.NamedTypes[typeName]
			require.True(t, ok, "missing type: %s", typeName)

			assert.Equal(t, originalType.Name, decType.Name)
			assert.Equal(t, originalType.PackagePath, decType.PackagePath)
			assert.Equal(t, originalType.DefinedInFilePath, decType.DefinedInFilePath)
			assert.Equal(t, originalType.TypeString, decType.TypeString)
			assert.Equal(t, originalType.UnderlyingTypeString, decType.UnderlyingTypeString)
			assert.Equal(t, originalType.TypeParams, decType.TypeParams)
			assert.Equal(t, originalType.Stringability, decType.Stringability)
			assert.Equal(t, originalType.IsAlias, decType.IsAlias)
			assert.Equal(t, originalType.DefinitionLine, decType.DefinitionLine)
			assert.Equal(t, originalType.DefinitionColumn, decType.DefinitionColumn)
			require.Len(t, decType.Fields, len(originalType.Fields))

			for i, originalField := range originalType.Fields {
				decField := decType.Fields[i]
				assert.Equal(t, originalField.Name, decField.Name, "field %d name", i)
				assert.Equal(t, originalField.TypeString, decField.TypeString)
				assert.Equal(t, originalField.UnderlyingTypeString, decField.UnderlyingTypeString)
				assert.Equal(t, originalField.IsEmbedded, decField.IsEmbedded)
				assert.Equal(t, originalField.RawTag, decField.RawTag)
				assert.Equal(t, originalField.PackagePath, decField.PackagePath)
				assert.Equal(t, originalField.IsGenericPlaceholder, decField.IsGenericPlaceholder)
				assert.Equal(t, originalField.DeclaringPackagePath, decField.DeclaringPackagePath)
				assert.Equal(t, originalField.DeclaringTypeName, decField.DeclaringTypeName)
				assert.Equal(t, originalField.CompositeType, decField.CompositeType)
				assert.Equal(t, originalField.IsUnderlyingInternalType, decField.IsUnderlyingInternalType)
				assert.Equal(t, originalField.IsUnderlyingPrimitive, decField.IsUnderlyingPrimitive)
				assert.Equal(t, originalField.IsInternalType, decField.IsInternalType)
				assert.Equal(t, originalField.IsAlias, decField.IsAlias)
				assert.Equal(t, originalField.DefinitionFilePath, decField.DefinitionFilePath)
				assert.Equal(t, originalField.DefinitionLine, decField.DefinitionLine)
				assert.Equal(t, originalField.DefinitionColumn, decField.DefinitionColumn)
				assertCompositePartsEqual(t, originalField.CompositeParts, decField.CompositeParts)
			}

			require.Len(t, decType.Methods, len(originalType.Methods))
			for i, origMethod := range originalType.Methods {
				decMethod := decType.Methods[i]
				assert.Equal(t, origMethod.Name, decMethod.Name, "method %d name", i)
				assert.Equal(t, origMethod.TypeString, decMethod.TypeString)
				assert.Equal(t, origMethod.UnderlyingTypeString, decMethod.UnderlyingTypeString)
				assert.Equal(t, origMethod.Signature, decMethod.Signature)
				assert.Equal(t, origMethod.IsPointerReceiver, decMethod.IsPointerReceiver)
				assert.Equal(t, origMethod.DeclaringPackagePath, decMethod.DeclaringPackagePath)
				assert.Equal(t, origMethod.DeclaringTypeName, decMethod.DeclaringTypeName)
				assert.Equal(t, origMethod.DefinitionFilePath, decMethod.DefinitionFilePath)
				assert.Equal(t, origMethod.DefinitionLine, decMethod.DefinitionLine)
				assert.Equal(t, origMethod.DefinitionColumn, decMethod.DefinitionColumn)
			}
		}

		for functionName, originalFunction := range originalPackage.Funcs {
			decFunc, ok := decPackage.Funcs[functionName]
			require.True(t, ok, "missing function: %s", functionName)
			assert.Equal(t, originalFunction.Name, decFunc.Name)
			assert.Equal(t, originalFunction.TypeString, decFunc.TypeString)
			assert.Equal(t, originalFunction.UnderlyingTypeString, decFunc.UnderlyingTypeString)
			assert.Equal(t, originalFunction.Signature, decFunc.Signature)
			assert.Equal(t, originalFunction.DefinitionFilePath, decFunc.DefinitionFilePath)
			assert.Equal(t, originalFunction.DefinitionLine, decFunc.DefinitionLine)
			assert.Equal(t, originalFunction.DefinitionColumn, decFunc.DefinitionColumn)
		}
	}
}

func assertCompositePartsEqual(t *testing.T, orig, dec []*inspector_dto.CompositePart) {
	t.Helper()
	require.Len(t, dec, len(orig))
	for i, originalPart := range orig {
		decPart := dec[i]
		assert.Equal(t, originalPart.Type, decPart.Type, "composite part %d type", i)
		assert.Equal(t, originalPart.TypeString, decPart.TypeString)
		assert.Equal(t, originalPart.Role, decPart.Role)
		assert.Equal(t, originalPart.UnderlyingTypeString, decPart.UnderlyingTypeString)
		assert.Equal(t, originalPart.PackagePath, decPart.PackagePath)
		assert.Equal(t, originalPart.CompositeType, decPart.CompositeType)
		assert.Equal(t, originalPart.Index, decPart.Index)
		assert.Equal(t, originalPart.IsInternalType, decPart.IsInternalType)
		assert.Equal(t, originalPart.IsUnderlyingInternalType, decPart.IsUnderlyingInternalType)
		assert.Equal(t, originalPart.IsGenericPlaceholder, decPart.IsGenericPlaceholder)
		assert.Equal(t, originalPart.IsAlias, decPart.IsAlias)
		assert.Equal(t, originalPart.IsUnderlyingPrimitive, decPart.IsUnderlyingPrimitive)
		assertCompositePartsEqual(t, originalPart.CompositeParts, decPart.CompositeParts)
	}
}

func TestFlatBufferRoundTrip_ParamNames(t *testing.T) {
	t.Parallel()

	sig := inspector_dto.FunctionSignature{
		Params:     []string{"string"},
		ParamNames: []string{"name"},
		Results:    []string{"error"},
	}

	b := flatbuffers.NewBuilder(256)
	offset := packFunctionSignature(b, &sig)
	b.Finish(offset)
	buffer := b.FinishedBytes()

	fb := inspector_schema_gen.GetRootAsFunctionSignature(buffer, 0)
	require.NotNil(t, fb)

	t.Logf("Isolated: ParamsLength=%d ResultsLength=%d ParamNamesLength=%d",
		fb.ParamsLength(), fb.ResultsLength(), fb.ParamNamesLength())

	unpacked := unpackFunctionSignature(fb)
	assert.Equal(t, []string{"string"}, unpacked.Params)
	assert.Equal(t, []string{"error"}, unpacked.Results)
	assert.Equal(t, []string{"name"}, unpacked.ParamNames)

	method := &inspector_dto.Method{
		Name:                 "Call",
		TypeString:           "func(name string) error",
		UnderlyingTypeString: "func(name string) error",
		Signature:            sig,
		DeclaringPackagePath: "example.com/foo",
		DeclaringTypeName:    "Widget",
		DefinitionFilePath:   "foo.go",
		DefinitionLine:       10,
		DefinitionColumn:     1,
	}

	b2 := flatbuffers.NewBuilder(512)
	methodOffset := packMethod(b2, method)
	b2.Finish(methodOffset)
	buf2 := b2.FinishedBytes()

	fbMethod := inspector_schema_gen.GetRootAsMethod(buf2, 0)
	require.NotNil(t, fbMethod)

	var fbSig inspector_schema_gen.FunctionSignature
	sigResult := fbMethod.Signature(&fbSig)
	require.NotNil(t, sigResult)

	t.Logf("Nested: ParamsLength=%d ResultsLength=%d ParamNamesLength=%d",
		sigResult.ParamsLength(), sigResult.ResultsLength(), sigResult.ParamNamesLength())

	unpacked2 := unpackFunctionSignature(sigResult)
	assert.Equal(t, []string{"string"}, unpacked2.Params)
	assert.Equal(t, []string{"error"}, unpacked2.Results)
	assert.Equal(t, []string{"name"}, unpacked2.ParamNames, "ParamNames lost when nested inside Method")

	method2 := &inspector_dto.Method{
		Name:                 "String",
		TypeString:           "func() string",
		UnderlyingTypeString: "func() string",
		Signature: inspector_dto.FunctionSignature{
			Params:  []string{},
			Results: []string{"string"},
		},
		DeclaringPackagePath: "example.com/foo",
		DeclaringTypeName:    "Widget",
		DefinitionFilePath:   "foo.go",
		DefinitionLine:       20,
		DefinitionColumn:     1,
	}

	b3 := flatbuffers.NewBuilder(1024)

	methods := []*inspector_dto.Method{method2, method}
	methodOffsets := make([]flatbuffers.UOffsetT, len(methods))
	for i, m := range methods {
		methodOffsets[i] = packMethod(b3, m)
	}

	b3.StartVector(4, len(methodOffsets), 4)
	for i := len(methodOffsets) - 1; i >= 0; i-- {
		b3.PrependUOffsetT(methodOffsets[i])
	}
	methodsVecOffset := b3.EndVector(len(methodOffsets))

	nameOffset := b3.CreateString("Widget")
	inspector_schema_gen.TypeStart(b3)
	inspector_schema_gen.TypeAddName(b3, nameOffset)
	inspector_schema_gen.TypeAddMethods(b3, methodsVecOffset)
	typeOffset := inspector_schema_gen.TypeEnd(b3)
	b3.Finish(typeOffset)
	buf3 := b3.FinishedBytes()

	fbType := inspector_schema_gen.GetRootAsType(buf3, 0)
	require.Equal(t, 2, fbType.MethodsLength())

	var fbMethod2 inspector_schema_gen.Method
	fbType.Methods(&fbMethod2, 1)

	var fbSig2 inspector_schema_gen.FunctionSignature
	sigResult2 := fbMethod2.Signature(&fbSig2)
	require.NotNil(t, sigResult2)

	t.Logf("Multi-method: ParamsLength=%d ResultsLength=%d ParamNamesLength=%d",
		sigResult2.ParamsLength(), sigResult2.ResultsLength(), sigResult2.ParamNamesLength())

	unpacked3 := unpackFunctionSignature(sigResult2)
	assert.Equal(t, []string{"string"}, unpacked3.Params)
	assert.Equal(t, []string{"error"}, unpacked3.Results)
	assert.Equal(t, []string{"name"}, unpacked3.ParamNames, "ParamNames lost when multiple methods packed")
}

func TestFlatBufferRoundTrip_EmptyTypeData(t *testing.T) {
	original := &inspector_dto.TypeData{}

	encoded := EncodeTypeDataToFBS(original)
	require.NotEmpty(t, encoded)

	decoded, err := DecodeTypeDataFromFBS(encoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.Packages)
	assert.Nil(t, decoded.FileToPackage)
}

func TestDecodeTypeDataFromFBS_EmptyData(t *testing.T) {
	_, err := DecodeTypeDataFromFBS(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty FlatBuffer data")

	_, err = DecodeTypeDataFromFBS([]byte{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty FlatBuffer data")
}

func TestFlatBufferCache_SaveAndGet(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	require.NoError(t, err)
	cache := NewFlatBufferCache(sandbox)
	ctx := context.Background()
	original := testTypeData()

	err = cache.SaveTypeData(ctx, "test", original)
	require.NoError(t, err)

	decoded, err := cache.GetTypeData(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, original.FileToPackage, decoded.FileToPackage)
	assert.Len(t, decoded.Packages, len(original.Packages))
}

func TestFlatBufferCache_NilSandboxOrEmptyKey(t *testing.T) {
	ctx := context.Background()
	td := testTypeData()
	mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)

	t.Run("nil sandbox on GetTypeData", func(t *testing.T) {
		cache := &FlatBufferCache{}
		_, err := cache.GetTypeData(ctx, "key")
		require.Error(t, err)
	})

	t.Run("empty key on GetTypeData", func(t *testing.T) {
		cache := NewFlatBufferCache(mockSandbox)
		_, err := cache.GetTypeData(ctx, "")
		require.Error(t, err)
	})

	t.Run("nil sandbox on SaveTypeData", func(t *testing.T) {
		cache := &FlatBufferCache{}
		err := cache.SaveTypeData(ctx, "key", td)
		require.Error(t, err)
	})

	t.Run("empty key on SaveTypeData", func(t *testing.T) {
		cache := NewFlatBufferCache(mockSandbox)
		err := cache.SaveTypeData(ctx, "", td)
		require.Error(t, err)
	})

	t.Run("nil sandbox on InvalidateCache", func(t *testing.T) {
		cache := &FlatBufferCache{}
		err := cache.InvalidateCache(ctx, "key")
		require.Error(t, err)
	})

	t.Run("empty key on InvalidateCache", func(t *testing.T) {
		cache := NewFlatBufferCache(mockSandbox)
		err := cache.InvalidateCache(ctx, "")
		require.Error(t, err)
	})

	t.Run("nil sandbox on ClearCache", func(t *testing.T) {
		cache := &FlatBufferCache{}
		err := cache.ClearCache(ctx)
		require.Error(t, err)
	})
}

func TestFlatBufferCache_InvalidateCache(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	require.NoError(t, err)
	cache := NewFlatBufferCache(sandbox)
	ctx := context.Background()

	err = cache.SaveTypeData(ctx, "key", testTypeData())
	require.NoError(t, err)

	err = cache.InvalidateCache(ctx, "key")
	require.NoError(t, err)

	_, err = cache.GetTypeData(ctx, "key")
	require.Error(t, err, "should get cache miss after invalidation")
}

func TestFlatBufferCache_InvalidateCache_NonExistent(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	require.NoError(t, err)
	cache := NewFlatBufferCache(sandbox)
	ctx := context.Background()

	err = cache.InvalidateCache(ctx, "missing")
	require.NoError(t, err)
}

func TestFlatBufferCache_ClearCache(t *testing.T) {
	sandbox, err := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	require.NoError(t, err)
	cache := NewFlatBufferCache(sandbox)
	ctx := context.Background()

	err = cache.SaveTypeData(ctx, "key1", testTypeData())
	require.NoError(t, err)

	err = cache.ClearCache(ctx)
	require.NoError(t, err)

	_, err = cache.GetTypeData(ctx, "key1")
	require.Error(t, err, "should get cache miss after clear")
}

func TestFlatBufferCache_CacheMiss(t *testing.T) {
	mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
	cache := NewFlatBufferCache(mockSandbox)
	ctx := context.Background()

	_, err := cache.GetTypeData(ctx, "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestFlatBufferCache_CorruptData(t *testing.T) {
	mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
	cache := NewFlatBufferCache(mockSandbox)
	ctx := context.Background()

	mockSandbox.AddFile("typedata-corrupt.bin", []byte("this is not valid flatbuffer data"))

	_, err := cache.GetTypeData(ctx, "corrupt")
	require.Error(t, err)

	_, statErr := mockSandbox.Stat("typedata-corrupt.bin")
	assert.ErrorIs(t, statErr, fs.ErrNotExist)
}

func TestFlatBufferCache_SchemaVersionMismatch(t *testing.T) {
	mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
	cache := NewFlatBufferCache(mockSandbox)
	ctx := context.Background()

	wrongPrefix := bytes.Repeat([]byte{0xFF}, 32)
	data := append(wrongPrefix, []byte("dummy payload data here")...)

	mockSandbox.AddFile("typedata-mismatch.bin", data)

	_, err := cache.GetTypeData(ctx, "mismatch")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "version mismatch")

	_, statErr := mockSandbox.Stat("typedata-mismatch.bin")
	assert.ErrorIs(t, statErr, fs.ErrNotExist)
}

func TestFlatBufferCache_SaveWriteFailure(t *testing.T) {
	mockSandbox := safedisk.NewMockSandbox("/cache", safedisk.ModeReadWrite)
	mockSandbox.WriteFileAtomicErr = errors.New("mock write failure")
	cache := NewFlatBufferCache(mockSandbox)
	ctx := context.Background()

	err := cache.SaveTypeData(ctx, "key", testTypeData())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write cache file atomically")
}
