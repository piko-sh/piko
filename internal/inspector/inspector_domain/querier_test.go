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

package inspector_domain

import (
	"context"
	"go/ast"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestTypeQuerier_Unit(t *testing.T) {
	t.Parallel()

	t.Run("Structs and Fields", func(t *testing.T) {
		t.Parallel()
		t.Run("should handle basic structs with exported and unexported fields", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Response": {
								Name: "Response",
								Fields: []*inspector_dto.Field{
									{Name: "Name", TypeString: "string"},
								},
							},
						},
					},
				},
			}

			respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
			require.NotNil(t, respType)
			require.Len(t, respType.Fields, 1)
			assert.Equal(t, "Name", respType.Fields[0].Name)
			assert.Equal(t, "string", respType.Fields[0].TypeString)
		})

		t.Run("should correctly capture raw struct tags", func(t *testing.T) {
			t.Parallel()

			tag := `json:"name,omitempty" validate:"required" prop:"username"`
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Request": {
								Name: "Request",
								Fields: []*inspector_dto.Field{
									{Name: "Name", TypeString: "string", RawTag: tag},
								},
							},
						},
					},
				},
			}

			reqType := typeData.Packages["my-project/main"].NamedTypes["Request"]
			require.Len(t, reqType.Fields, 1)
			assert.Equal(t, tag, reqType.Fields[0].RawTag)
		})
	})

	t.Run("Methods", func(t *testing.T) {
		t.Parallel()

		t.Run("should handle methods on both value and pointer receivers", func(t *testing.T) {
			t.Parallel()
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name: "User",
								Methods: []*inspector_dto.Method{
									{
										Name:              "GetName",
										IsPointerReceiver: true,
										TypeString:        "string",
										Signature:         inspector_dto.FunctionSignature{Results: []string{"string"}},
									},
									{
										Name:              "GetAge",
										IsPointerReceiver: false,
										TypeString:        "int",
										Signature:         inspector_dto.FunctionSignature{Results: []string{"int"}},
									},
								},
							},
						},
					},
				},
			}

			userType := typeData.Packages["my-project/main"].NamedTypes["User"]
			require.Len(t, userType.Methods, 2)
			methodMap := make(map[string]*inspector_dto.Method)
			for _, m := range userType.Methods {
				methodMap[m.Name] = m
			}
			assert.Equal(t, "string", methodMap["GetName"].TypeString)
			assert.Equal(t, "int", methodMap["GetAge"].TypeString)
		})

		t.Run("should ignore unexported but include no-return-value methods", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name: "User",
								Methods: []*inspector_dto.Method{
									{
										Name:              "Good",
										IsPointerReceiver: true,
										TypeString:        "bool",
										Signature:         inspector_dto.FunctionSignature{Results: []string{"bool"}},
									},
									{
										Name:              "NoReturn",
										IsPointerReceiver: true,
										TypeString:        "",
										Signature:         inspector_dto.FunctionSignature{},
									},
								},
							},
						},
					},
				},
			}

			userType := typeData.Packages["my-project/main"].NamedTypes["User"]
			require.Len(t, userType.Methods, 2)

			methodMap := make(map[string]bool)
			for _, m := range userType.Methods {
				methodMap[m.Name] = true
			}

			assert.Contains(t, methodMap, "Good", "Should contain the method with a return value")
			assert.Contains(t, methodMap, "NoReturn", "Should contain the exported method with no return value")
			assert.NotContains(t, methodMap, "bad", "Should NOT contain the unexported method")
		})

		t.Run("should correctly encode method signatures with multiple arguments and returns", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name: "User",
								Methods: []*inspector_dto.Method{
									{
										Name:              "Update",
										IsPointerReceiver: true,
										TypeString:        "bool",
										Signature: inspector_dto.FunctionSignature{
											Params:  []string{"int", "string"},
											Results: []string{"bool", "error"},
										},
									},
								},
							},
						},
					},
				},
			}

			userType := typeData.Packages["my-project/main"].NamedTypes["User"]
			require.Len(t, userType.Methods, 1)
			methodSig := userType.Methods[0].Signature
			require.NotNil(t, methodSig)
			assert.Equal(t, []string{"int", "string"}, methodSig.Params)
			assert.Equal(t, []string{"bool", "error"}, methodSig.Results)
		})

		t.Run("should correctly flag pointer receivers", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name: "User",
								Methods: []*inspector_dto.Method{
									{
										Name:              "SetName",
										IsPointerReceiver: true,
										TypeString:        "",
										Signature:         inspector_dto.FunctionSignature{Params: []string{"string"}},
									},
									{
										Name:              "GetName",
										IsPointerReceiver: false,
										TypeString:        "string",
										Signature:         inspector_dto.FunctionSignature{Results: []string{"string"}},
									},
								},
							},
						},
					},
				},
			}

			userType := typeData.Packages["my-project/main"].NamedTypes["User"]
			require.Len(t, userType.Methods, 2)

			methodMap := make(map[string]*inspector_dto.Method)
			for _, m := range userType.Methods {
				methodMap[m.Name] = m
			}

			require.Contains(t, methodMap, "SetName")
			require.Contains(t, methodMap, "GetName")

			assert.True(t, methodMap["SetName"].IsPointerReceiver, "SetName should have a pointer receiver")
			assert.False(t, methodMap["GetName"].IsPointerReceiver, "GetName should have a value receiver")
		})
	})

	t.Run("Complex Types and Composition", func(t *testing.T) {
		t.Parallel()

		t.Run("should distinguish between type definition and type alias", func(t *testing.T) {
			t.Parallel()
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"UserID": {
								Name:                 "UserID",
								TypeString:           "my-project/main.UserID",
								UnderlyingTypeString: "string",
							},
							"RequestID": {
								Name:                 "RequestID",
								TypeString:           "string",
								UnderlyingTypeString: "string",
							},
							"Response": {
								Name:       "Response",
								TypeString: "my-project/main.Response",
								Fields: []*inspector_dto.Field{
									{
										Name:                 "UID",
										TypeString:           "my-project/main.UserID",
										UnderlyingTypeString: "string",
									},
									{
										Name:                 "RID",
										TypeString:           "string",
										UnderlyingTypeString: "string",
									},
								},
							},
						},
					},
				},
			}

			respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
			require.NotNil(t, respType, "Response type should be found in encoded data")
			require.Len(t, respType.Fields, 2)

			uidField := respType.Fields[0]
			assert.Equal(t, "UID", uidField.Name)
			assert.Equal(t, "my-project/main.UserID", uidField.TypeString)
			assert.Equal(t, "string", uidField.UnderlyingTypeString)

			ridField := respType.Fields[1]
			assert.Equal(t, "RID", ridField.Name)
			assert.Equal(t, "string", ridField.TypeString, "The type of a field using a type alias should resolve to the aliased type")
			assert.Equal(t, "string", ridField.UnderlyingTypeString, "The underlying type should also be the aliased type")
		})
	})

	t.Run("Packages and Naming", func(t *testing.T) {
		t.Parallel()

		t.Run("should respect aliased imports in type strings", func(t *testing.T) {
			t.Parallel()
			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/models": {
						Name: "models_v1",
						Path: "my-project/models",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:       "User",
								TypeString: "my-project/models.User",
							},
						},
					},
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						FileImports: map[string]map[string]string{
							"my-project/main/main.go": {"api": "my-project/models"},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Response": {
								Name:       "Response",
								TypeString: "my-project/main.Response",
								Fields: []*inspector_dto.Field{
									{
										Name:       "User",
										TypeString: "api.User",
									},
								},
							},
						},
					},
				},
			}

			respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
			require.Len(t, respType.Fields, 1)
			assert.Equal(t, "api.User", respType.Fields[0].TypeString)
		})

		t.Run("should handle dot imports correctly", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/utils": {
						Name: "utils",
						Path: "my-project/utils",
						NamedTypes: map[string]*inspector_dto.Type{
							"Helper": {
								Name:       "Helper",
								TypeString: "my-project/utils.Helper",
							},
						},
					},
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						FileImports: map[string]map[string]string{
							"my-project/main/main.go": {".": "my-project/utils"},
						},
						NamedTypes: map[string]*inspector_dto.Type{
							"Response": {
								Name:       "Response",
								TypeString: "my-project/main.Response",
								Fields: []*inspector_dto.Field{
									{
										Name:       "Helper",
										TypeString: "utils.Helper",
									},
								},
							},
						},
					},
				},
			}

			respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
			require.Len(t, respType.Fields, 1)
			assert.Equal(t, "utils.Helper", respType.Fields[0].TypeString)
		})
	})

	t.Run("Generics", func(t *testing.T) {
		t.Parallel()

		t.Run("should handle generic type with constraints", func(t *testing.T) {
			t.Parallel()
			pkg := types.NewPackage("my-project/main", "main")

			anyConstraint := types.NewInterfaceType(nil, nil)
			comparableLookup := types.Universe.Lookup("comparable").Type().Underlying()
			comparableConstraint, ok := comparableLookup.(*types.Interface)
			require.True(t, ok, "comparable underlying type should be *types.Interface")
			kParam := types.NewTypeParam(types.NewTypeName(0, pkg, "K", nil), comparableConstraint)
			vParam := types.NewTypeParam(types.NewTypeName(0, pkg, "V", nil), anyConstraint)
			tParams := []*types.TypeParam{kParam, vParam}

			structDef := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "Key", kParam, false),
				types.NewField(0, pkg, "Value", vParam, false),
			}, nil)

			typeName := types.NewTypeName(0, pkg, "KeyValue", nil)
			namedType := types.NewNamed(typeName, nil, nil)
			namedType.SetTypeParams(tParams)
			namedType.SetUnderlying(structDef)
			pkg.Scope().Insert(typeName)

			loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}
			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			kvType := typeData.Packages["my-project/main"].NamedTypes["KeyValue"]
			require.NotNil(t, kvType)
			assert.Equal(t, "main.KeyValue[K, V]", kvType.TypeString)
			assert.Equal(t, "struct{Key K; Value V}", kvType.UnderlyingTypeString)
			require.Len(t, kvType.Fields, 2)
			assert.Equal(t, "K", kvType.Fields[0].TypeString)
			assert.Equal(t, "V", kvType.Fields[1].TypeString)
		})

		t.Run("should handle package-level generic function", func(t *testing.T) {
			t.Parallel()

			pkg := types.NewPackage("my-project/main", "main")

			comparableLookup := types.Universe.Lookup("comparable").Type().Underlying()
			comparableConstraint, ok := comparableLookup.(*types.Interface)
			require.True(t, ok, "comparable underlying type should be *types.Interface")
			tParam := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), comparableConstraint)

			sig := types.NewSignatureType(
				nil, nil, []*types.TypeParam{tParam},
				types.NewTuple(
					types.NewVar(0, pkg, "slice", types.NewSlice(tParam)),
					types.NewVar(0, pkg, "value", tParam),
				),
				types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.Int])),
				false,
			)

			typeFunction := types.NewFunc(0, pkg, "Find", sig)
			pkg.Scope().Insert(typeFunction)

			loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}
			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			findFunc := typeData.Packages["my-project/main"].Funcs["Find"]
			require.NotNil(t, findFunc)
			assert.Equal(t, []string{"[]T", "T"}, findFunc.Signature.Params)
			assert.Equal(t, []string{"int"}, findFunc.Signature.Results)
		})

		t.Run("should handle methods using generic type parameter", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Set": {
								Name:       "Set",
								TypeString: "my-project/main.Set[T]",
								TypeParams: []string{"T"},
								Methods: []*inspector_dto.Method{
									{
										Name:              "Add",
										IsPointerReceiver: true,
										TypeString:        "",
										Signature:         inspector_dto.FunctionSignature{Params: []string{"T"}},
									},
								},
							},
						},
					},
				},
			}

			setType := typeData.Packages["my-project/main"].NamedTypes["Set"]
			require.NotNil(t, setType)
			require.Len(t, setType.Methods, 1)

			addMethod := setType.Methods[0]
			assert.Equal(t, "Add", addMethod.Name)
			assert.Equal(t, []string{"T"}, addMethod.Signature.Params)
			assert.Empty(t, addMethod.Signature.Results)
		})
	})

	t.Run("Interfaces Functions and Core Types", func(t *testing.T) {
		t.Parallel()

		t.Run("should handle interface embedding", func(t *testing.T) {
			t.Parallel()

			pkg := types.NewPackage("my-project/main", "main")

			readerMethods := []*types.Func{
				types.NewFunc(0, pkg, "Read", types.NewSignatureType(
					nil, nil, nil,

					types.NewTuple(types.NewVar(0, pkg, "p", types.NewSlice(types.Typ[types.Uint8]))),
					types.NewTuple(
						types.NewVar(0, pkg, "n", types.Typ[types.Int]),
						types.NewVar(0, pkg, "err", types.Universe.Lookup("error").Type()),
					),
					false,
				)),
			}
			readerIface := types.NewInterfaceType(readerMethods, nil)
			readerTypeName := types.NewTypeName(0, pkg, "Reader", nil)
			readerNamedType := types.NewNamed(readerTypeName, readerIface, nil)
			pkg.Scope().Insert(readerTypeName)

			closerMethods := []*types.Func{
				types.NewFunc(0, pkg, "Close", types.NewSignatureType(
					nil, nil, nil, nil,
					types.NewTuple(types.NewVar(0, pkg, "err", types.Universe.Lookup("error").Type())),
					false,
				)),
			}
			closerIface := types.NewInterfaceType(closerMethods, nil)
			closerTypeName := types.NewTypeName(0, pkg, "Closer", nil)
			closerNamedType := types.NewNamed(closerTypeName, closerIface, nil)
			pkg.Scope().Insert(closerTypeName)

			embeddedTypes := []types.Type{readerNamedType, closerNamedType}
			readCloserIface := types.NewInterfaceType(nil, embeddedTypes)
			readCloserTypeName := types.NewTypeName(0, pkg, "ReadCloser", nil)
			types.NewNamed(readCloserTypeName, readCloserIface, nil)
			pkg.Scope().Insert(readCloserTypeName)

			loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			rcType := typeData.Packages["my-project/main"].NamedTypes["ReadCloser"]
			require.NotNil(t, rcType)
			require.Len(t, rcType.Methods, 2, "Should have methods from both embedded interfaces")

			methodMap := make(map[string]*inspector_dto.Method)
			for _, m := range rcType.Methods {
				methodMap[m.Name] = m
			}
			assert.Contains(t, methodMap, "Read")
			assert.Contains(t, methodMap, "Close")

			assert.Equal(t, []string{"[]uint8"}, methodMap["Read"].Signature.Params)

			assert.Equal(t, []string{"int", "error"}, methodMap["Read"].Signature.Results)
			assert.Equal(t, []string{"error"}, methodMap["Close"].Signature.Results)
		})

		t.Run("should handle variadic functions", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Logger": {
								Name:       "Logger",
								TypeString: "my-project/main.Logger",
								Methods: []*inspector_dto.Method{
									{
										Name:              "Logf",
										IsPointerReceiver: false,
										TypeString:        "",
										Signature: inspector_dto.FunctionSignature{
											Params: []string{"string", "...any"},
										},
									},
								},
							},
						},
					},
				},
			}

			loggerType := typeData.Packages["my-project/main"].NamedTypes["Logger"]
			require.NotNil(t, loggerType)
			require.Len(t, loggerType.Methods, 1)

			method := loggerType.Methods[0]
			assert.Equal(t, "Logf", method.Name)
			assert.Equal(t, []string{"string", "...any"}, method.Signature.Params)
			assert.Empty(t, method.Signature.Results)
		})

		t.Run("should distinguish fixed-size arrays from slices", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Message": {
								Name:       "Message",
								TypeString: "my-project/main.Message",
								Fields: []*inspector_dto.Field{
									{
										Name:       "Header",
										TypeString: "[16]uint8",
									},
									{
										Name:       "Body",
										TypeString: "[]uint8",
									},
								},
							},
						},
					},
				},
			}

			msgType := typeData.Packages["my-project/main"].NamedTypes["Message"]
			require.Len(t, msgType.Fields, 2)

			assert.Equal(t, "[16]uint8", msgType.Fields[0].TypeString)
			assert.Equal(t, "[]uint8", msgType.Fields[1].TypeString)
		})

		t.Run("should resolve chained type aliases", func(t *testing.T) {
			t.Parallel()

			pkg := types.NewPackage("my-project/main", "main")

			localIDTypeName := types.NewTypeName(0, pkg, "LocalID", nil)
			types.NewAlias(localIDTypeName, types.Typ[types.String])
			pkg.Scope().Insert(localIDTypeName)

			userIDTypeName := types.NewTypeName(0, pkg, "UserID", nil)
			types.NewAlias(userIDTypeName, localIDTypeName.Type())
			pkg.Scope().Insert(userIDTypeName)

			reqStruct := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "ID", userIDTypeName.Type(), false),
			}, nil)
			reqTypeName := types.NewTypeName(0, pkg, "Request", nil)
			types.NewNamed(reqTypeName, reqStruct, nil)
			pkg.Scope().Insert(reqTypeName)

			loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			reqType := typeData.Packages["my-project/main"].NamedTypes["Request"]
			require.NotNil(t, reqType, "Request type should be found in encoded data")
			require.Len(t, reqType.Fields, 1)
			assert.Equal(t, "main.UserID", reqType.Fields[0].TypeString, "Chained alias should preserve the declared alias type")
		})
	})

	t.Run("Advanced Composition", func(t *testing.T) {
		t.Parallel()

		t.Run("should handle embedding a pointer", func(t *testing.T) {
			t.Parallel()

			pkg := types.NewPackage("my-project/main", "main")

			employeeStruct := types.NewStruct(nil, nil)
			employeeTypeName := types.NewTypeName(0, pkg, "Employee", nil)
			employeeNamed := types.NewNamed(employeeTypeName, employeeStruct, nil)
			pkg.Scope().Insert(employeeTypeName)

			promoteSig := types.NewSignatureType(
				types.NewVar(0, pkg, "e", types.NewPointer(employeeNamed)),
				nil, nil, nil, nil, false,
			)
			employeeNamed.AddMethod(types.NewFunc(0, pkg, "Promote", promoteSig))

			managerStruct := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "Employee", types.NewPointer(employeeNamed), true),
			}, nil)
			managerTypeName := types.NewTypeName(0, pkg, "Manager", nil)
			types.NewNamed(managerTypeName, managerStruct, nil)
			pkg.Scope().Insert(managerTypeName)

			loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			managerType := typeData.Packages["my-project/main"].NamedTypes["Manager"]
			require.NotNil(t, managerType)
			require.Len(t, managerType.Fields, 1)
			assert.True(t, managerType.Fields[0].IsEmbedded)
			assert.Equal(t, "*main.Employee", managerType.Fields[0].TypeString)
			require.Len(t, managerType.Methods, 1)
			assert.Equal(t, "Promote", managerType.Methods[0].Name)
		})

		t.Run("should promote methods from unexported embedded type", func(t *testing.T) {
			t.Parallel()

			pkg := types.NewPackage("my-project/main", "main")

			privateStruct := types.NewStruct(nil, nil)
			privateTypeName := types.NewTypeName(0, pkg, "private", nil)
			privateNamed := types.NewNamed(privateTypeName, privateStruct, nil)
			pkg.Scope().Insert(privateTypeName)

			methodSig := types.NewSignatureType(
				types.NewVar(0, pkg, "p", privateNamed), nil, nil, nil,
				types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.String])),
				false,
			)
			privateNamed.AddMethod(types.NewFunc(0, pkg, "ExportedMethod", methodSig))

			publicStruct := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "private", privateNamed, true),
			}, nil)
			publicTypeName := types.NewTypeName(0, pkg, "Public", nil)
			types.NewNamed(publicTypeName, publicStruct, nil)
			pkg.Scope().Insert(publicTypeName)

			loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			publicType := typeData.Packages["my-project/main"].NamedTypes["Public"]
			require.NotNil(t, publicType)

			assert.Empty(t, publicType.Fields)

			require.Len(t, publicType.Methods, 1)
			assert.Equal(t, "ExportedMethod", publicType.Methods[0].Name)
			assert.Equal(t, "string", publicType.Methods[0].TypeString)
		})

		t.Run("should handle field that is an interface to a generic type", func(t *testing.T) {
			t.Parallel()

			pkg := types.NewPackage("my-project/main", "main")

			anyConstraint := types.NewInterfaceType(nil, nil)
			tParam := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), anyConstraint)

			produceMethod := types.NewFunc(0, pkg, "Produce", types.NewSignatureType(
				nil, nil, nil, nil,
				types.NewTuple(types.NewVar(0, pkg, "", tParam)),
				false,
			))

			producerIface := types.NewInterfaceType([]*types.Func{produceMethod}, nil)
			producerTypeName := types.NewTypeName(0, pkg, "Producer", nil)
			producerGenericType := types.NewNamed(producerTypeName, producerIface, nil)
			producerGenericType.SetTypeParams([]*types.TypeParam{tParam})
			pkg.Scope().Insert(producerTypeName)

			producerStringInstance, err := types.Instantiate(nil, producerGenericType, []types.Type{types.Typ[types.String]}, false)
			require.NoError(t, err)

			factoryStruct := types.NewStruct([]*types.Var{
				types.NewField(0, pkg, "StringProducer", producerStringInstance, false),
			}, nil)
			factoryTypeName := types.NewTypeName(0, pkg, "WidgetFactory", nil)
			types.NewNamed(factoryTypeName, factoryStruct, nil)
			pkg.Scope().Insert(factoryTypeName)

			loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			factoryType := typeData.Packages["my-project/main"].NamedTypes["WidgetFactory"]
			require.NotNil(t, factoryType)
			require.Len(t, factoryType.Fields, 1)
			assert.Equal(t, "StringProducer", factoryType.Fields[0].Name)
			assert.Equal(t, "main.Producer[string]", factoryType.Fields[0].TypeString)
		})
	})
	t.Run("Core Language Constructs", func(t *testing.T) {
		t.Parallel()

		t.Run("should handle pointer types", func(t *testing.T) {
			t.Parallel()

			t.Run("field with pointer to primitive", func(t *testing.T) {
				t.Parallel()
				typeData := &inspector_dto.TypeData{
					Packages: map[string]*inspector_dto.Package{
						"my-project/main": {
							Name: "main",
							Path: "my-project/main",
							NamedTypes: map[string]*inspector_dto.Type{
								"Response": {
									Name:       "Response",
									TypeString: "my-project/main.Response",
									Fields: []*inspector_dto.Field{
										{Name: "Count", TypeString: "*int"},
									},
								},
							},
						},
					},
				}

				respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
				require.Len(t, respType.Fields, 1)
				assert.Equal(t, "*int", respType.Fields[0].TypeString)
			})

			t.Run("field with multi-level pointer", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")
				configStruct := types.NewStruct(nil, nil)
				configTypeName := types.NewTypeName(0, pkg, "MyConfig", nil)
				configNamed := types.NewNamed(configTypeName, configStruct, nil)
				pkg.Scope().Insert(configTypeName)

				responseStruct := types.NewStruct([]*types.Var{
					types.NewField(0, pkg, "Config", types.NewPointer(types.NewPointer(configNamed)), false),
				}, nil)
				responseTypeName := types.NewTypeName(0, pkg, "Response", nil)
				_ = types.NewNamed(responseTypeName, responseStruct, nil)
				pkg.Scope().Insert(responseTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}
				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
				require.Len(t, respType.Fields, 1)
				assert.Equal(t, "**main.MyConfig", respType.Fields[0].TypeString)
			})
		})

		t.Run("should handle complex slice and map types", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"User": {
								Name:       "User",
								TypeString: "my-project/main.User",
							},
							"Session": {
								Name:       "Session",
								TypeString: "my-project/main.Session",
							},
							"Complex": {
								Name:       "Complex",
								TypeString: "my-project/main.Complex",
								Fields: []*inspector_dto.Field{
									{Name: "Users", TypeString: "[]*main.User"},
									{Name: "Rights", TypeString: "map[main.User]string"},
									{Name: "Cache", TypeString: "map[string]*main.Session"},
									{Name: "Groups", TypeString: "map[string][]main.User"},
								},
							},
						},
					},
				},
			}

			complexType := typeData.Packages["my-project/main"].NamedTypes["Complex"]
			require.Len(t, complexType.Fields, 4)
			fieldMap := make(map[string]*inspector_dto.Field)
			for _, f := range complexType.Fields {
				fieldMap[f.Name] = f
			}

			assert.Equal(t, "[]*main.User", fieldMap["Users"].TypeString)
			assert.Equal(t, "map[main.User]string", fieldMap["Rights"].TypeString)
			assert.Equal(t, "map[string]*main.Session", fieldMap["Cache"].TypeString)
			assert.Equal(t, "map[string][]main.User", fieldMap["Groups"].TypeString)
		})

		t.Run("should handle channel types", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Event": {
								Name:       "Event",
								TypeString: "my-project/main.Event",
							},
							"Broker": {
								Name:       "Broker",
								TypeString: "my-project/main.Broker",
								Fields: []*inspector_dto.Field{
									{Name: "Events", TypeString: "chan main.Event"},
									{Name: "Input", TypeString: "chan<- int"},
									{Name: "Output", TypeString: "<-chan string"},
								},
							},
						},
					},
				},
			}

			brokerType := typeData.Packages["my-project/main"].NamedTypes["Broker"]
			require.Len(t, brokerType.Fields, 3)
			fieldMap := make(map[string]*inspector_dto.Field)
			for _, f := range brokerType.Fields {
				fieldMap[f.Name] = f
			}

			assert.Equal(t, "chan main.Event", fieldMap["Events"].TypeString)
			assert.Equal(t, "chan<- int", fieldMap["Input"].TypeString)
			assert.Equal(t, "<-chan string", fieldMap["Output"].TypeString)
		})

		t.Run("should handle function types", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Router": {
								Name:       "Router",
								TypeString: "my-project/main.Router",
								Fields: []*inspector_dto.Field{
									{Name: "Handler", TypeString: "func(id int, name string) error"},
									{Name: "Callback", TypeString: "func()"},
								},
							},
						},
					},
				},
			}

			routerType := typeData.Packages["my-project/main"].NamedTypes["Router"]
			require.Len(t, routerType.Fields, 2)
			fieldMap := make(map[string]*inspector_dto.Field)
			for _, f := range routerType.Fields {
				fieldMap[f.Name] = f
			}

			assert.Equal(t, "func(id int, name string) error", fieldMap["Handler"].TypeString)

			assert.Equal(t, "func()", fieldMap["Callback"].TypeString)
		})

		t.Run("should handle special interface types", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Container": {
								Name:       "Container",
								TypeString: "my-project/main.Container",
								Fields: []*inspector_dto.Field{
									{Name: "Data", TypeString: "interface{}"},
									{Name: "Value", TypeString: "any"},
								},
							},
						},
					},
				},
			}

			containerType := typeData.Packages["my-project/main"].NamedTypes["Container"]
			require.Len(t, containerType.Fields, 2)
			assert.Equal(t, "interface{}", containerType.Fields[0].TypeString)
			assert.Equal(t, "any", containerType.Fields[1].TypeString)
		})

		t.Run("should handle all built-in primitive types", func(t *testing.T) {
			t.Parallel()

			typeData := &inspector_dto.TypeData{
				Packages: map[string]*inspector_dto.Package{
					"my-project/main": {
						Name: "main",
						Path: "my-project/main",
						NamedTypes: map[string]*inspector_dto.Type{
							"Primitives": {
								Name:       "Primitives",
								TypeString: "my-project/main.Primitives",
								Fields: []*inspector_dto.Field{
									{Name: "IntVal", TypeString: "int"},
									{Name: "Int8Val", TypeString: "int8"},
									{Name: "Int16Val", TypeString: "int16"},
									{Name: "Int32Val", TypeString: "int32"},
									{Name: "Int64Val", TypeString: "int64"},
									{Name: "UintVal", TypeString: "uint"},
									{Name: "Uint8Val", TypeString: "uint8"},
									{Name: "Uint16Val", TypeString: "uint16"},
									{Name: "Uint32Val", TypeString: "uint32"},
									{Name: "Uint64Val", TypeString: "uint64"},
									{Name: "UintptrVal", TypeString: "uintptr"},
									{Name: "Float32Val", TypeString: "float32"},
									{Name: "Float64Val", TypeString: "float64"},
									{Name: "Complex64Val", TypeString: "complex64"},
									{Name: "Complex128Val", TypeString: "complex128"},
									{Name: "BoolVal", TypeString: "bool"},
									{Name: "StringVal", TypeString: "string"},
									{Name: "RuneVal", TypeString: "rune"},
								},
							},
						},
					},
				},
			}

			primitivesType := typeData.Packages["my-project/main"].NamedTypes["Primitives"]
			require.Len(t, primitivesType.Fields, 18)

			expected := map[string]string{
				"IntVal": "int", "Int8Val": "int8", "Int16Val": "int16", "Int32Val": "int32", "Int64Val": "int64",
				"UintVal": "uint", "Uint8Val": "uint8", "Uint16Val": "uint16", "Uint32Val": "uint32", "Uint64Val": "uint64",
				"UintptrVal": "uintptr", "Float32Val": "float32", "Float64Val": "float64",
				"Complex64Val": "complex64", "Complex128Val": "complex128",
				"BoolVal": "bool", "StringVal": "string", "RuneVal": "rune",
			}
			for _, field := range primitivesType.Fields {
				assert.Equal(t, expected[field.Name], field.TypeString)
			}
		})

		t.Run("should handle edge cases", func(t *testing.T) {
			t.Parallel()

			t.Run("struct with blank identifier field", func(t *testing.T) {
				t.Parallel()
				typeData := &inspector_dto.TypeData{
					Packages: map[string]*inspector_dto.Package{
						"my-project/main": {
							Name: "main",
							Path: "my-project/main",
							NamedTypes: map[string]*inspector_dto.Type{
								"Edge": {
									Name:       "Edge",
									TypeString: "my-project/main.Edge",
									Fields: []*inspector_dto.Field{
										{Name: "Exported", TypeString: "bool"},
									},
								},
							},
						},
					},
				}

				edgeType := typeData.Packages["my-project/main"].NamedTypes["Edge"]
				require.Len(t, edgeType.Fields, 1, "Should only find the exported field")
				assert.Equal(t, "Exported", edgeType.Fields[0].Name)
			})
			t.Run("struct with no exported fields", func(t *testing.T) {
				t.Parallel()

				typeData := &inspector_dto.TypeData{
					Packages: map[string]*inspector_dto.Package{
						"my-project/main": {
							Name: "main",
							Path: "my-project/main",
							NamedTypes: map[string]*inspector_dto.Type{
								"Secret": {
									Name:       "Secret",
									TypeString: "my-project/main.Secret",
									Fields:     []*inspector_dto.Field{},
								},
							},
						},
					},
				}

				secretType := typeData.Packages["my-project/main"].NamedTypes["Secret"]
				require.NotNil(t, secretType)
				assert.Empty(t, secretType.Fields, "Should find 0 exported fields")
			})
		})
	})

	t.Run("Advanced Edge Cases", func(t *testing.T) {
		t.Parallel()

		t.Run("Generics", func(t *testing.T) {
			t.Parallel()

			t.Run("should handle union type constraints", func(t *testing.T) {
				t.Parallel()
				pkg := types.NewPackage("my-project/main", "main")

				intTerm := types.NewTerm(true, types.Typ[types.Int])
				floatTerm := types.NewTerm(true, types.Typ[types.Float64])
				numberUnion := types.NewUnion([]*types.Term{intTerm, floatTerm})

				numberIface := types.NewInterfaceType(nil, []types.Type{numberUnion})

				tParam := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), numberIface)
				valueStruct := types.NewStruct([]*types.Var{
					types.NewField(0, pkg, "V", tParam, false),
				}, nil)
				valueTypeName := types.NewTypeName(0, pkg, "Value", nil)
				valueNamed := types.NewNamed(valueTypeName, nil, nil)
				valueNamed.SetTypeParams([]*types.TypeParam{tParam})
				valueNamed.SetUnderlying(valueStruct)
				pkg.Scope().Insert(valueTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}
				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				valType := typeData.Packages["my-project/main"].NamedTypes["Value"]
				require.NotNil(t, valType)
				assert.Equal(t, "main.Value[T]", valType.TypeString)
			})

			t.Run("should handle tilde constraints on structs", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")
				pointStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "X", types.Typ[types.Int], false)}, nil)
				pointTypeName := types.NewTypeName(0, pkg, "Point", nil)
				pointNamed := types.NewNamed(pointTypeName, pointStruct, nil)
				pkg.Scope().Insert(pointTypeName)

				pointTerm := types.NewTerm(true, pointNamed)
				anyPointUnion := types.NewUnion([]*types.Term{pointTerm})

				anyPointIface := types.NewInterfaceType(nil, []types.Type{anyPointUnion})

				pParam := types.NewTypeParam(types.NewTypeName(0, pkg, "P", nil), anyPointIface)
				drawSig := types.NewSignatureType(nil, nil, []*types.TypeParam{pParam}, types.NewTuple(types.NewVar(0, pkg, "p", pParam)), nil, false)
				drawFunc := types.NewFunc(0, pkg, "Draw", drawSig)
				pkg.Scope().Insert(drawFunc)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}
				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				inspectedFunction := typeData.Packages["my-project/main"].Funcs["Draw"]
				require.NotNil(t, inspectedFunction)
				assert.Equal(t, "Draw", inspectedFunction.Name)
			})

			t.Run("should handle recursive generic types", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				anyConstraint := types.NewInterfaceType(nil, nil)
				tParam := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), anyConstraint)
				nodeTypeName := types.NewTypeName(0, pkg, "Node", nil)
				nodeNamed := types.NewNamed(nodeTypeName, nil, nil)
				nodeNamed.SetTypeParams([]*types.TypeParam{tParam})

				nodeStruct := types.NewStruct([]*types.Var{
					types.NewField(0, pkg, "Value", tParam, false),
					types.NewField(0, pkg, "Next", types.NewPointer(nodeNamed), false),
				}, nil)
				nodeNamed.SetUnderlying(nodeStruct)
				pkg.Scope().Insert(nodeTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				nodeType := typeData.Packages["my-project/main"].NamedTypes["Node"]
				require.NotNil(t, nodeType)
				require.Len(t, nodeType.Fields, 2)
				assert.Equal(t, "T", nodeType.Fields[0].TypeString)
				assert.Equal(t, "*main.Node[T]", nodeType.Fields[1].TypeString)
			})

			t.Run("should handle generic methods on non-generic types", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				myStruct := types.NewStruct(nil, nil)
				myTypeName := types.NewTypeName(0, pkg, "MyStruct", nil)
				myNamed := types.NewNamed(myTypeName, myStruct, nil)
				pkg.Scope().Insert(myTypeName)

				anyConstraint := types.NewInterfaceType(nil, nil)
				tParam := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), anyConstraint)
				genericFuncSig := types.NewSignatureType(
					nil, nil, []*types.TypeParam{tParam},
					types.NewTuple(types.NewVar(0, pkg, "v", tParam)),
					types.NewTuple(types.NewVar(0, pkg, "", tParam)),
					false,
				)

				invalidGenericMethod := types.NewFunc(0, pkg, "GenericMethod", genericFuncSig)
				myNamed.AddMethod(invalidGenericMethod)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				msType := typeData.Packages["my-project/main"].NamedTypes["MyStruct"]
				require.NotNil(t, msType)

				assert.Empty(t, msType.Methods, "Generic methods are not part of a type's method set and should be ignored")
			})

			t.Run("should handle instantiated generic used as a type parameter", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")
				anyConstraint := types.NewInterfaceType(nil, nil)

				tParamBox := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), anyConstraint)
				boxStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Value", tParamBox, false)}, nil)
				boxTypeName := types.NewTypeName(0, pkg, "Box", nil)
				boxNamed := types.NewNamed(boxTypeName, boxStruct, nil)
				boxNamed.SetTypeParams([]*types.TypeParam{tParamBox})
				pkg.Scope().Insert(boxTypeName)

				tParamWrapper := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), anyConstraint)
				wrapperStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "V", tParamWrapper, false)}, nil)
				wrapperTypeName := types.NewTypeName(0, pkg, "Wrapper", nil)
				wrapperNamed := types.NewNamed(wrapperTypeName, wrapperStruct, nil)
				wrapperNamed.SetTypeParams([]*types.TypeParam{tParamWrapper})
				pkg.Scope().Insert(wrapperTypeName)

				boxStringInst, err := types.Instantiate(nil, boxNamed, []types.Type{types.Typ[types.String]}, false)
				require.NoError(t, err)

				wrapperBoxStringInst, err := types.Instantiate(nil, wrapperNamed, []types.Type{boxStringInst}, false)
				require.NoError(t, err)

				responseStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "BoxedWrapper", wrapperBoxStringInst, false)}, nil)
				responseTypeName := types.NewTypeName(0, pkg, "Response", nil)
				types.NewNamed(responseTypeName, responseStruct, nil)
				pkg.Scope().Insert(responseTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
				require.NotNil(t, respType)
				require.Len(t, respType.Fields, 1)
				assert.Equal(t, "main.Wrapper[main.Box[string]]", respType.Fields[0].TypeString)
			})
		})

		t.Run("Composition and Recursion", func(t *testing.T) {
			t.Parallel()

			t.Run("should handle mutually recursive struct types", func(t *testing.T) {
				t.Parallel()
				pkg := types.NewPackage("my-project/main", "main")

				aTypeName := types.NewTypeName(0, pkg, "A", nil)
				aNamed := types.NewNamed(aTypeName, nil, nil)

				bTypeName := types.NewTypeName(0, pkg, "B", nil)
				bNamed := types.NewNamed(bTypeName, nil, nil)

				aStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "B", types.NewPointer(bNamed), false)}, nil)
				bStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "A", types.NewPointer(aNamed), false)}, nil)

				aNamed.SetUnderlying(aStruct)
				bNamed.SetUnderlying(bStruct)

				pkg.Scope().Insert(aTypeName)
				pkg.Scope().Insert(bTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				aType := typeData.Packages["my-project/main"].NamedTypes["A"]
				bType := typeData.Packages["my-project/main"].NamedTypes["B"]
				require.NotNil(t, aType)
				require.NotNil(t, bType)
				require.Len(t, aType.Fields, 1)
				require.Len(t, bType.Fields, 1)
				assert.Equal(t, "*main.B", aType.Fields[0].TypeString)
				assert.Equal(t, "*main.A", bType.Fields[0].TypeString)
			})

			t.Run("should handle embedding an instantiated generic type", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")
				anyConstraint := types.NewInterfaceType(nil, nil)

				tParamBox := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), anyConstraint)
				boxStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Value", tParamBox, false)}, nil)
				boxTypeName := types.NewTypeName(0, pkg, "Box", nil)
				boxNamed := types.NewNamed(boxTypeName, boxStruct, nil)
				boxNamed.SetTypeParams([]*types.TypeParam{tParamBox})
				pkg.Scope().Insert(boxTypeName)

				boxStringInst, err := types.Instantiate(nil, boxNamed, []types.Type{types.Typ[types.String]}, false)
				require.NoError(t, err)

				stringBoxStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Box", boxStringInst, true)}, nil)
				stringBoxTypeName := types.NewTypeName(0, pkg, "StringBox", nil)
				types.NewNamed(stringBoxTypeName, stringBoxStruct, nil)
				pkg.Scope().Insert(stringBoxTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				sbType := typeData.Packages["my-project/main"].NamedTypes["StringBox"]
				require.NotNil(t, sbType)
				require.Len(t, sbType.Fields, 1)
				assert.True(t, sbType.Fields[0].IsEmbedded)
				assert.Equal(t, "main.Box[string]", sbType.Fields[0].TypeString)
			})

			t.Run("should handle embedding the built-in error type", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")
				errorType := types.Universe.Lookup("error").Type()

				myErrorStruct := types.NewStruct([]*types.Var{
					types.NewField(0, pkg, "error", errorType, true),
					types.NewField(0, pkg, "Code", types.Typ[types.Int], false),
				}, nil)
				myErrorTypeName := types.NewTypeName(0, pkg, "MyError", nil)
				types.NewNamed(myErrorTypeName, myErrorStruct, nil)
				pkg.Scope().Insert(myErrorTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				meType := typeData.Packages["my-project/main"].NamedTypes["MyError"]
				require.NotNil(t, meType)
				require.Len(t, meType.Methods, 1)
				assert.Equal(t, "Error", meType.Methods[0].Name)
			})

			t.Run("should handle embedding an anonymous struct", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				anonStruct := types.NewStruct([]*types.Var{
					types.NewField(0, pkg, "Status", types.Typ[types.String], false),
				}, nil)

				responseStruct := types.NewStruct([]*types.Var{
					types.NewField(0, pkg, "", anonStruct, true),
				}, nil)
				responseTypeName := types.NewTypeName(0, pkg, "Response", nil)
				types.NewNamed(responseTypeName, responseStruct, nil)
				pkg.Scope().Insert(responseTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
				require.NotNil(t, respType)

				assert.Empty(t, respType.Fields)
			})

			t.Run("should handle method shadowing from an embedded type", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")
				baseStruct := types.NewStruct(nil, nil)
				baseTypeName := types.NewTypeName(0, pkg, "Base", nil)
				baseNamed := types.NewNamed(baseTypeName, baseStruct, nil)
				baseNamed.AddMethod(types.NewFunc(0, pkg, "String", types.NewSignatureType(
					types.NewVar(0, pkg, "b", baseNamed), nil, nil, nil,
					types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.String])), false,
				)))
				pkg.Scope().Insert(baseTypeName)

				outerStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Base", baseNamed, true)}, nil)
				outerTypeName := types.NewTypeName(0, pkg, "Outer", nil)
				outerNamed := types.NewNamed(outerTypeName, outerStruct, nil)
				outerNamed.AddMethod(types.NewFunc(0, pkg, "String", types.NewSignatureType(
					types.NewVar(0, pkg, "o", outerNamed), nil, nil, nil,
					types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.String])), false,
				)))
				pkg.Scope().Insert(outerTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				outerType := typeData.Packages["my-project/main"].NamedTypes["Outer"]
				require.NotNil(t, outerType)
				require.Len(t, outerType.Methods, 1)

				assert.Equal(t, "String", outerType.Methods[0].Name)
			})
		})

		t.Run("Interfaces", func(t *testing.T) {
			t.Parallel()

			t.Run("should handle interfaces embedding instantiated generic interfaces", func(t *testing.T) {
				t.Parallel()
				pkg := types.NewPackage("my-project/main", "main")

				anyConstraint := types.NewInterfaceType(nil, nil)
				tParam := types.NewTypeParam(types.NewTypeName(0, pkg, "T", nil), anyConstraint)
				getterIface := types.NewInterfaceType([]*types.Func{
					types.NewFunc(0, pkg, "Get", types.NewSignatureType(nil, nil, nil, nil, types.NewTuple(types.NewVar(0, pkg, "", tParam)), false)),
				}, nil)
				getterTypeName := types.NewTypeName(0, pkg, "Getter", nil)
				getterNamed := types.NewNamed(getterTypeName, getterIface, nil)
				getterNamed.SetTypeParams([]*types.TypeParam{tParam})
				pkg.Scope().Insert(getterTypeName)

				getterStringInst, err := types.Instantiate(nil, getterNamed, []types.Type{types.Typ[types.String]}, false)
				require.NoError(t, err)

				stringGetterIface := types.NewInterfaceType(nil, []types.Type{getterStringInst})
				stringGetterTypeName := types.NewTypeName(0, pkg, "StringGetter", nil)
				types.NewNamed(stringGetterTypeName, stringGetterIface, nil)
				pkg.Scope().Insert(stringGetterTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				sgType := typeData.Packages["my-project/main"].NamedTypes["StringGetter"]
				require.NotNil(t, sgType)
				require.Len(t, sgType.Methods, 1)
				assert.Equal(t, "Get", sgType.Methods[0].Name)
				assert.Equal(t, "string", sgType.Methods[0].TypeString)
			})

			t.Run("should handle methods that return interface types", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				loggerIface := types.NewInterfaceType(nil, nil)
				loggerTypeName := types.NewTypeName(0, pkg, "Logger", nil)
				loggerNamed := types.NewNamed(loggerTypeName, loggerIface, nil)
				pkg.Scope().Insert(loggerTypeName)

				factoryIface := types.NewInterfaceType([]*types.Func{
					types.NewFunc(0, pkg, "NewLogger", types.NewSignatureType(
						nil, nil, nil, nil, types.NewTuple(types.NewVar(0, pkg, "", loggerNamed)), false,
					)),
				}, nil)
				factoryTypeName := types.NewTypeName(0, pkg, "LoggerFactory", nil)
				types.NewNamed(factoryTypeName, factoryIface, nil)
				pkg.Scope().Insert(factoryTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				factoryType := typeData.Packages["my-project/main"].NamedTypes["LoggerFactory"]
				require.NotNil(t, factoryType)
				require.Len(t, factoryType.Methods, 1)
				assert.Equal(t, "main.Logger", factoryType.Methods[0].TypeString)
			})

			t.Run("should handle an alias to an empty interface", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				dataTypeName := types.NewTypeName(0, pkg, "Data", nil)
				types.NewAlias(dataTypeName, types.NewInterfaceType(nil, nil))
				pkg.Scope().Insert(dataTypeName)

				containerStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Value", dataTypeName.Type(), false)}, nil)
				containerTypeName := types.NewTypeName(0, pkg, "Container", nil)
				types.NewNamed(containerTypeName, containerStruct, nil)
				pkg.Scope().Insert(containerTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				containerType := typeData.Packages["my-project/main"].NamedTypes["Container"]
				require.NotNil(t, containerType)
				require.Len(t, containerType.Fields, 1)
				assert.Equal(t, "main.Data", containerType.Fields[0].TypeString)
			})
		})

		t.Run("Type Definitions and Aliases", func(t *testing.T) {
			t.Parallel()

			t.Run("should handle type definition on a primitive with methods", func(t *testing.T) {
				t.Parallel()
				pkg := types.NewPackage("my-project/main", "main")

				myIntTypeName := types.NewTypeName(0, pkg, "MyInt", nil)
				myIntNamed := types.NewNamed(myIntTypeName, types.Typ[types.Int], nil)
				myIntNamed.AddMethod(types.NewFunc(0, pkg, "IsPositive", types.NewSignatureType(
					types.NewVar(0, pkg, "m", myIntNamed), nil, nil, nil,
					types.NewTuple(types.NewVar(0, pkg, "", types.Typ[types.Bool])), false,
				)))
				pkg.Scope().Insert(myIntTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				miType := typeData.Packages["my-project/main"].NamedTypes["MyInt"]
				require.NotNil(t, miType)
				assert.Equal(t, "int", miType.UnderlyingTypeString)
				require.Len(t, miType.Methods, 1)
				assert.Equal(t, "IsPositive", miType.Methods[0].Name)
			})

			t.Run("should handle an alias to a fixed-size array", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				macTypeName := types.NewTypeName(0, pkg, "MACAddress", nil)
				types.NewAlias(macTypeName, types.NewArray(types.Typ[types.Uint8], 6))
				pkg.Scope().Insert(macTypeName)

				deviceStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "MAC", macTypeName.Type(), false)}, nil)
				deviceTypeName := types.NewTypeName(0, pkg, "Device", nil)
				types.NewNamed(deviceTypeName, deviceStruct, nil)
				pkg.Scope().Insert(deviceTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				deviceType := typeData.Packages["my-project/main"].NamedTypes["Device"]
				require.NotNil(t, deviceType)
				require.Len(t, deviceType.Fields, 1)
				assert.Equal(t, "main.MACAddress", deviceType.Fields[0].TypeString)
			})

			t.Run("should handle an alias to a function type", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				handlerSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
				handlerTypeName := types.NewTypeName(0, pkg, "HandlerFunc", nil)
				types.NewAlias(handlerTypeName, handlerSig)
				pkg.Scope().Insert(handlerTypeName)

				routerStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Handler", handlerTypeName.Type(), false)}, nil)
				routerTypeName := types.NewTypeName(0, pkg, "Router", nil)
				types.NewNamed(routerTypeName, routerStruct, nil)
				pkg.Scope().Insert(routerTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				routerType := typeData.Packages["my-project/main"].NamedTypes["Router"]
				require.NotNil(t, routerType)
				require.Len(t, routerType.Fields, 1)
				assert.Equal(t, "main.HandlerFunc", routerType.Fields[0].TypeString)
			})

			t.Run("should handle an alias to a map type", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				headersTypeName := types.NewTypeName(0, pkg, "Headers", nil)
				types.NewAlias(headersTypeName, types.NewMap(types.Typ[types.String], types.Typ[types.String]))
				pkg.Scope().Insert(headersTypeName)

				reqStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "H", headersTypeName.Type(), false)}, nil)
				reqTypeName := types.NewTypeName(0, pkg, "Request", nil)
				types.NewNamed(reqTypeName, reqStruct, nil)
				pkg.Scope().Insert(reqTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				reqType := typeData.Packages["my-project/main"].NamedTypes["Request"]
				require.NotNil(t, reqType)
				require.Len(t, reqType.Fields, 1)
				assert.Equal(t, "main.Headers", reqType.Fields[0].TypeString)
			})
		})

		t.Run("Package and Built-in Corner Cases", func(t *testing.T) {
			t.Parallel()

			t.Run("should handle fields using types from the unsafe package", func(t *testing.T) {
				t.Parallel()
				pkg := types.NewPackage("my-project/main", "main")
				unsafePtr := types.Typ[types.UnsafePointer]

				bufferStruct := types.NewStruct([]*types.Var{types.NewField(0, pkg, "Ptr", unsafePtr, false)}, nil)
				bufferTypeName := types.NewTypeName(0, pkg, "Buffer", nil)
				types.NewNamed(bufferTypeName, bufferStruct, nil)
				pkg.Scope().Insert(bufferTypeName)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				bufType := typeData.Packages["my-project/main"].NamedTypes["Buffer"]
				require.NotNil(t, bufType)
				require.Len(t, bufType.Fields, 1)
				assert.Equal(t, "unsafe.Pointer", bufType.Fields[0].TypeString)
			})

			t.Run("should handle fields using instantiated generics from another package", func(t *testing.T) {
				t.Parallel()

				modelsPackage := types.NewPackage("my-project/models", "models")
				anyConstraint := types.NewInterfaceType(nil, nil)
				tParamBox := types.NewTypeParam(types.NewTypeName(0, modelsPackage, "T", nil), anyConstraint)
				boxStruct := types.NewStruct([]*types.Var{types.NewField(0, modelsPackage, "Value", tParamBox, false)}, nil)
				boxTypeName := types.NewTypeName(0, modelsPackage, "Box", nil)
				boxNamed := types.NewNamed(boxTypeName, boxStruct, nil)
				boxNamed.SetTypeParams([]*types.TypeParam{tParamBox})
				modelsPackage.Scope().Insert(boxTypeName)

				mainPackage := types.NewPackage("my-project/main", "main")
				boxStringInst, err := types.Instantiate(nil, boxNamed, []types.Type{types.Typ[types.String]}, false)
				require.NoError(t, err)

				containerStruct := types.NewStruct([]*types.Var{types.NewField(0, mainPackage, "StringBox", boxStringInst, false)}, nil)
				containerTypeName := types.NewTypeName(0, mainPackage, "Container", nil)
				types.NewNamed(containerTypeName, containerStruct, nil)
				mainPackage.Scope().Insert(containerTypeName)

				loadedPackages := []*packages.Package{
					{Name: "models", PkgPath: "my-project/models", Types: modelsPackage},
					{Name: "main", PkgPath: "my-project/main", Types: mainPackage},
				}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				containerType := typeData.Packages["my-project/main"].NamedTypes["Container"]
				require.NotNil(t, containerType)
				require.Len(t, containerType.Fields, 1)
				assert.Equal(t, "models.Box[string]", containerType.Fields[0].TypeString)
			})

			t.Run("should handle struct with no fields", func(t *testing.T) {
				t.Parallel()

				typeData := &inspector_dto.TypeData{
					Packages: map[string]*inspector_dto.Package{
						"my-project/main": {
							Name: "main",
							Path: "my-project/main",
							NamedTypes: map[string]*inspector_dto.Type{
								"Empty": {
									Name:       "Empty",
									TypeString: "my-project/main.Empty",
									Fields:     []*inspector_dto.Field{},
									Methods:    []*inspector_dto.Method{},
								},
							},
						},
					},
				}

				emptyType := typeData.Packages["my-project/main"].NamedTypes["Empty"]
				require.NotNil(t, emptyType)
				assert.Empty(t, emptyType.Fields)
				assert.Empty(t, emptyType.Methods)
			})

			t.Run("should handle variadic function with non-any type", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				sig := types.NewSignatureType(
					nil, nil, nil,
					types.NewTuple(types.NewVar(0, pkg, "messages", types.NewSlice(types.Typ[types.String]))),
					nil, true,
				)
				logFunc := types.NewFunc(0, pkg, "Log", sig)
				pkg.Scope().Insert(logFunc)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				inspectedFunction := typeData.Packages["my-project/main"].Funcs["Log"]
				require.NotNil(t, inspectedFunction)
				require.Len(t, inspectedFunction.Signature.Params, 1)
				assert.Equal(t, "...string", inspectedFunction.Signature.Params[0])
			})

			t.Run("should handle multiple named return values", func(t *testing.T) {
				t.Parallel()

				pkg := types.NewPackage("my-project/main", "main")

				sig := types.NewSignatureType(
					nil, nil, nil, nil,
					types.NewTuple(
						types.NewVar(0, pkg, "n", types.Typ[types.Int]),
						types.NewVar(0, pkg, "err", types.Universe.Lookup("error").Type()),
					),
					false,
				)
				myFunc := types.NewFunc(0, pkg, "MyFunc", sig)
				pkg.Scope().Insert(myFunc)

				loadedPackages := []*packages.Package{{Name: "main", PkgPath: "my-project/main", Types: pkg}}

				typeData, err := extractAndEncode(loadedPackages, "")
				require.NoError(t, err)

				inspectedFunction := typeData.Packages["my-project/main"].Funcs["MyFunc"]
				require.NotNil(t, inspectedFunction)
				require.Len(t, inspectedFunction.Signature.Results, 2)

				assert.Equal(t, "int", inspectedFunction.Signature.Results[0])
				assert.Equal(t, "error", inspectedFunction.Signature.Results[1])
			})
		})
	})

	t.Run("Advanced Scenarios and Interactions", func(t *testing.T) {
		t.Parallel()

		t.Run("should correctly encode a map whose definition uses multiple aliases", func(t *testing.T) {
			t.Parallel()
			modelsPackage := types.NewPackage("my-project/models", "models")
			mainPackage := types.NewPackage("my-project/main", "main")

			metadataStruct := types.NewStruct(nil, nil)
			metadataTypeName := types.NewTypeName(0, modelsPackage, "Metadata", nil)
			metadataNamed := types.NewNamed(metadataTypeName, metadataStruct, nil)
			modelsPackage.Scope().Insert(metadataTypeName)

			anyConstraint := types.NewInterfaceType(nil, nil)
			mParam := types.NewTypeParam(types.NewTypeName(0, modelsPackage, "M", nil), anyConstraint)
			productStruct := types.NewStruct([]*types.Var{
				types.NewField(0, modelsPackage, "Meta", mParam, false),
			}, nil)
			productTypeName := types.NewTypeName(0, modelsPackage, "Product", nil)
			productGenericNamed := types.NewNamed(productTypeName, productStruct, nil)
			productGenericNamed.SetTypeParams([]*types.TypeParam{mParam})
			modelsPackage.Scope().Insert(productTypeName)

			userIDTypeName := types.NewTypeName(0, mainPackage, "UserID", nil)
			types.NewAlias(userIDTypeName, types.Typ[types.String])
			mainPackage.Scope().Insert(userIDTypeName)

			productInstance, err := types.Instantiate(nil, productGenericNamed, []types.Type{metadataNamed}, false)
			require.NoError(t, err)

			productMapDef := types.NewMap(userIDTypeName.Type(), productInstance)
			productMapTypeName := types.NewTypeName(0, mainPackage, "ProductMap", nil)
			types.NewAlias(productMapTypeName, productMapDef)
			mainPackage.Scope().Insert(productMapTypeName)

			responseStruct := types.NewStruct([]*types.Var{
				types.NewField(0, mainPackage, "Products", productMapTypeName.Type(), false),
			}, nil)
			responseTypeName := types.NewTypeName(0, mainPackage, "Response", nil)
			types.NewNamed(responseTypeName, responseStruct, nil)
			mainPackage.Scope().Insert(responseTypeName)

			loadedPackages := []*packages.Package{
				{Name: "models", PkgPath: "my-project/models", Types: modelsPackage},
				{Name: "main", PkgPath: "my-project/main", Types: mainPackage, Syntax: []*ast.File{
					{Imports: []*ast.ImportSpec{
						{Path: &ast.BasicLit{Value: `"my-project/models"`}, Name: ast.NewIdent("models")},
					}},
				}},
			}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
			require.NotNil(t, respType)
			require.Len(t, respType.Fields, 1)
			productsField := respType.Fields[0]
			expectedTypeString := "main.ProductMap"
			assert.Equal(t, expectedTypeString, productsField.TypeString)
		})

		t.Run("should handle an alias to an instantiated generic type", func(t *testing.T) {
			t.Parallel()

			modelsPackage := types.NewPackage("my-project/models", "models")
			mainPackage := types.NewPackage("my-project/main", "main")

			anyConstraint := types.NewInterfaceType(nil, nil)
			tParam := types.NewTypeParam(types.NewTypeName(0, modelsPackage, "T", nil), anyConstraint)
			boxStruct := types.NewStruct([]*types.Var{types.NewField(0, modelsPackage, "Value", tParam, false)}, nil)
			boxTypeName := types.NewTypeName(0, modelsPackage, "Box", nil)
			boxNamed := types.NewNamed(boxTypeName, boxStruct, nil)
			boxNamed.SetTypeParams([]*types.TypeParam{tParam})
			modelsPackage.Scope().Insert(boxTypeName)

			boxStringInstance, err := types.Instantiate(nil, boxNamed, []types.Type{types.Typ[types.String]}, false)
			require.NoError(t, err)

			stringBoxTypeName := types.NewTypeName(0, mainPackage, "StringBox", nil)
			types.NewAlias(stringBoxTypeName, boxStringInstance)
			mainPackage.Scope().Insert(stringBoxTypeName)

			responseStruct := types.NewStruct([]*types.Var{
				types.NewField(0, mainPackage, "PrimaryBox", stringBoxTypeName.Type(), false),
			}, nil)
			responseTypeName := types.NewTypeName(0, mainPackage, "Response", nil)
			types.NewNamed(responseTypeName, responseStruct, nil)
			mainPackage.Scope().Insert(responseTypeName)

			loadedPackages := []*packages.Package{
				{Name: "models", PkgPath: "my-project/models", Types: modelsPackage},
				{Name: "main", PkgPath: "my-project/main", Types: mainPackage, Syntax: []*ast.File{
					{Imports: []*ast.ImportSpec{{Path: &ast.BasicLit{Value: `"my-project/models"`}, Name: ast.NewIdent("models")}}},
				}},
			}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
			require.NotNil(t, respType)
			require.Len(t, respType.Fields, 1)
			assert.Equal(t, "main.StringBox", respType.Fields[0].TypeString)
		})

		t.Run("should handle a generic type definition that uses an alias", func(t *testing.T) {
			t.Parallel()

			modelsPackage := types.NewPackage("my-project/models", "models")

			identifierTypeName := types.NewTypeName(0, modelsPackage, "Identifier", nil)
			types.NewAlias(identifierTypeName, types.Typ[types.String])
			modelsPackage.Scope().Insert(identifierTypeName)

			anyConstraint := types.NewInterfaceType(nil, nil)
			tParam := types.NewTypeParam(types.NewTypeName(0, modelsPackage, "T", nil), anyConstraint)
			itemStruct := types.NewStruct([]*types.Var{
				types.NewField(0, modelsPackage, "ID", identifierTypeName.Type(), false),
				types.NewField(0, modelsPackage, "Data", tParam, false),
			}, nil)
			itemTypeName := types.NewTypeName(0, modelsPackage, "Item", nil)
			itemNamed := types.NewNamed(itemTypeName, itemStruct, nil)
			itemNamed.SetTypeParams([]*types.TypeParam{tParam})
			modelsPackage.Scope().Insert(itemTypeName)

			loadedPackages := []*packages.Package{{Name: "models", PkgPath: "my-project/models", Types: modelsPackage}}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			itemType := typeData.Packages["my-project/models"].NamedTypes["Item"]
			require.NotNil(t, itemType)
			assert.Equal(t, "struct{ID string; Data T}", itemType.UnderlyingTypeString)
		})

		t.Run("should encode promoted method from embedded generic with substituted type", func(t *testing.T) {
			t.Parallel()

			modelsPackage := types.NewPackage("my-project/models", "models")
			mainPackage := types.NewPackage("my-project/main", "main")

			anyConstraint := types.NewInterfaceType(nil, nil)
			tParam := types.NewTypeParam(types.NewTypeName(0, modelsPackage, "T", nil), anyConstraint)
			boxStruct := types.NewStruct(nil, nil)
			boxTypeName := types.NewTypeName(0, modelsPackage, "Box", nil)
			boxNamed := types.NewNamed(boxTypeName, boxStruct, nil)
			boxNamed.SetTypeParams([]*types.TypeParam{tParam})
			boxNamed.AddMethod(types.NewFunc(0, modelsPackage, "Get", types.NewSignatureType(
				types.NewVar(0, modelsPackage, "b", boxNamed), nil, nil, nil,
				types.NewTuple(types.NewVar(0, modelsPackage, "", tParam)), false,
			)))
			modelsPackage.Scope().Insert(boxTypeName)

			boxStringInstance, err := types.Instantiate(nil, boxNamed, []types.Type{types.Typ[types.String]}, false)
			require.NoError(t, err)

			stringBoxStruct := types.NewStruct([]*types.Var{
				types.NewField(0, mainPackage, "Box", boxStringInstance, true),
			}, nil)
			stringBoxTypeName := types.NewTypeName(0, mainPackage, "StringBox", nil)
			types.NewNamed(stringBoxTypeName, stringBoxStruct, nil)
			mainPackage.Scope().Insert(stringBoxTypeName)

			loadedPackages := []*packages.Package{
				{Name: "models", PkgPath: "my-project/models", Types: modelsPackage},
				{Name: "main", PkgPath: "my-project/main", Types: mainPackage, Syntax: []*ast.File{
					{Imports: []*ast.ImportSpec{{Path: &ast.BasicLit{Value: `"my-project/models"`}, Name: ast.NewIdent("models")}}},
				}},
			}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			sbType := typeData.Packages["my-project/main"].NamedTypes["StringBox"]
			require.NotNil(t, sbType)
			require.Len(t, sbType.Methods, 1)
			getMethod := sbType.Methods[0]
			assert.Equal(t, "Get", getMethod.Name)
			assert.Equal(t, "string", getMethod.TypeString, "Return type should be substituted from T to string")
		})

		t.Run("should handle an alias to a generic function type", func(t *testing.T) {
			t.Parallel()

			modelsPackage := types.NewPackage("my-project/models", "models")
			mainPackage := types.NewPackage("my-project/main", "main")
			anyConstraint := types.NewInterfaceType(nil, nil)
			errType := types.Universe.Lookup("error").Type()

			tParam := types.NewTypeParam(types.NewTypeName(0, modelsPackage, "T", nil), anyConstraint)

			funcSig := types.NewSignatureType(
				nil, nil, []*types.TypeParam{tParam},
				types.NewTuple(types.NewVar(0, nil, "", tParam)),
				types.NewTuple(types.NewVar(0, nil, "", tParam), types.NewVar(0, nil, "", errType)),
				false,
			)

			processorTypeName := types.NewTypeName(0, modelsPackage, "Processor", nil)
			types.NewAlias(processorTypeName, funcSig)
			modelsPackage.Scope().Insert(processorTypeName)

			processorIntInstance, err := types.Instantiate(nil, funcSig, []types.Type{types.Typ[types.Int]}, false)
			require.NoError(t, err)

			intProcessorTypeName := types.NewTypeName(0, mainPackage, "IntProcessor", nil)
			types.NewAlias(intProcessorTypeName, processorIntInstance)
			mainPackage.Scope().Insert(intProcessorTypeName)

			responseStruct := types.NewStruct([]*types.Var{
				types.NewField(0, mainPackage, "Handler", intProcessorTypeName.Type(), false),
			}, nil)
			responseTypeName := types.NewTypeName(0, mainPackage, "Response", nil)
			types.NewNamed(responseTypeName, responseStruct, nil)
			mainPackage.Scope().Insert(responseTypeName)

			loadedPackages := []*packages.Package{
				{Name: "models", PkgPath: "my-project/models", Types: modelsPackage},
				{Name: "main", PkgPath: "my-project/main", Types: mainPackage, Syntax: []*ast.File{
					{Imports: []*ast.ImportSpec{{Path: &ast.BasicLit{Value: `"my-project/models"`}, Name: ast.NewIdent("models")}}},
				}},
			}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			respType := typeData.Packages["my-project/main"].NamedTypes["Response"]
			require.NotNil(t, respType)
			require.Len(t, respType.Fields, 1)

			assert.Equal(t, "main.IntProcessor", respType.Fields[0].TypeString)
		})

		t.Run("should handle a method returning an instantiated generic interface", func(t *testing.T) {
			t.Parallel()

			modelsPackage := types.NewPackage("my-project/models", "models")
			anyConstraint := types.NewInterfaceType(nil, nil)

			tParam := types.NewTypeParam(types.NewTypeName(0, modelsPackage, "T", nil), anyConstraint)
			producerIface := types.NewInterfaceType([]*types.Func{
				types.NewFunc(0, modelsPackage, "Produce", types.NewSignatureType(
					nil, nil, nil, nil, types.NewTuple(types.NewVar(0, nil, "", tParam)), false,
				)),
			}, nil)
			producerTypeName := types.NewTypeName(0, modelsPackage, "Producer", nil)
			producerNamed := types.NewNamed(producerTypeName, producerIface, nil)
			producerNamed.SetTypeParams([]*types.TypeParam{tParam})
			modelsPackage.Scope().Insert(producerTypeName)

			producerStringInstance, err := types.Instantiate(nil, producerNamed, []types.Type{types.Typ[types.String]}, false)
			require.NoError(t, err)

			factoryStruct := types.NewStruct(nil, nil)
			factoryTypeName := types.NewTypeName(0, modelsPackage, "Factory", nil)
			factoryNamed := types.NewNamed(factoryTypeName, factoryStruct, nil)
			factoryNamed.AddMethod(types.NewFunc(0, modelsPackage, "NewStringProducer", types.NewSignatureType(
				types.NewVar(0, modelsPackage, "f", factoryNamed), nil, nil, nil,
				types.NewTuple(types.NewVar(0, nil, "", producerStringInstance)), false,
			)))
			modelsPackage.Scope().Insert(factoryTypeName)

			loadedPackages := []*packages.Package{{Name: "models", PkgPath: "my-project/models", Types: modelsPackage}}

			typeData, err := extractAndEncode(loadedPackages, "")
			require.NoError(t, err)

			factoryType := typeData.Packages["my-project/models"].NamedTypes["Factory"]
			require.NotNil(t, factoryType)
			require.Len(t, factoryType.Methods, 1)
			method := factoryType.Methods[0]
			assert.Equal(t, "NewStringProducer", method.Name)
			assert.Equal(t, "models.Producer[string]", method.TypeString)
		})
	})
}

func TestTypeQuerier_Memoization(t *testing.T) {
	t.Parallel()

	t.Run("should cache type resolutions across multiple calls", func(t *testing.T) {
		t.Parallel()

		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my-project/models": {
					Name: "models",
					Path: "my-project/models",
					FileImports: map[string]map[string]string{
						"/path/to/file.go": {
							"models": "my-project/models",
						},
					},
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {
							Name:       "User",
							TypeString: "models.User",
							Fields: []*inspector_dto.Field{
								{Name: "Name", TypeString: "string"},
								{Name: "Email", TypeString: "string"},
							},
						},
						"Product": {
							Name:       "Product",
							TypeString: "models.Product",
							Fields: []*inspector_dto.Field{
								{Name: "ID", TypeString: "int"},
								{Name: "Title", TypeString: "string"},
							},
						},
					},
				},
			},
		}

		querier := NewTypeQuerier(nil, typeData, inspector_dto.Config{})

		userExpr := &ast.SelectorExpr{
			X:   &ast.Ident{Name: "models"},
			Sel: &ast.Ident{Name: "User"},
		}

		result1, pkg1 := querier.ResolveExprToNamedTypeWithMemoization(
			context.Background(),
			userExpr,
			"my-project/models",
			"/path/to/file.go",
		)

		require.NotNil(t, result1, "First call should resolve the type")
		assert.Equal(t, "User", result1.Name)
		assert.Equal(t, "models", pkg1)

		result2, pkg2 := querier.ResolveExprToNamedTypeWithMemoization(
			context.Background(),
			userExpr,
			"my-project/models",
			"/path/to/file.go",
		)

		require.NotNil(t, result2, "Second call should also resolve the type")
		assert.Equal(t, "User", result2.Name)
		assert.Equal(t, "models", pkg2)

		assert.Same(t, result1, result2, "Cache should return the exact same Type pointer")

		productExpr := &ast.SelectorExpr{
			X:   &ast.Ident{Name: "models"},
			Sel: &ast.Ident{Name: "Product"},
		}

		result3, pkg3 := querier.ResolveExprToNamedTypeWithMemoization(
			context.Background(),
			productExpr,
			"my-project/models",
			"/path/to/file.go",
		)

		require.NotNil(t, result3, "Third call should resolve the Product type")
		assert.Equal(t, "Product", result3.Name)
		assert.Equal(t, "models", pkg3)
		assert.NotSame(t, result1, result3, "Different types should have different pointers")
	})

	t.Run("should use file-scoped cache keys", func(t *testing.T) {
		t.Parallel()

		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my-project/models": {
					Name: "models",
					Path: "my-project/models",
					FileImports: map[string]map[string]string{
						"/path/to/file1.go": {"models": "my-project/models"},
						"/path/to/file2.go": {"models": "my-project/models"},
					},
					NamedTypes: map[string]*inspector_dto.Type{
						"User": {
							Name:       "User",
							TypeString: "models.User",
						},
					},
				},
			},
		}

		querier := NewTypeQuerier(nil, typeData, inspector_dto.Config{})

		userExpr := &ast.SelectorExpr{
			X:   &ast.Ident{Name: "models"},
			Sel: &ast.Ident{Name: "User"},
		}

		result1, _ := querier.ResolveExprToNamedTypeWithMemoization(
			context.Background(),
			userExpr,
			"my-project/models",
			"/path/to/file1.go",
		)

		result2, _ := querier.ResolveExprToNamedTypeWithMemoization(
			context.Background(),
			userExpr,
			"my-project/models",
			"/path/to/file2.go",
		)

		require.NotNil(t, result1)
		require.NotNil(t, result2)

		assert.Equal(t, result1.Name, result2.Name)
	})

	t.Run("should bypass cache for non-deconstructable expressions", func(t *testing.T) {
		t.Parallel()

		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"my-project/main": {
					Name: "main",
					Path: "my-project/main",
					FileImports: map[string]map[string]string{
						"/path/to/file.go": {},
					},
					NamedTypes: map[string]*inspector_dto.Type{},
				},
			},
		}

		querier := NewTypeQuerier(nil, typeData, inspector_dto.Config{})

		complexExpr := &ast.StarExpr{
			X: &ast.ArrayType{
				Elt: &ast.Ident{Name: "int"},
			},
		}

		result, pkg := querier.ResolveExprToNamedTypeWithMemoization(
			context.Background(),
			complexExpr,
			"my-project/main",
			"/path/to/file.go",
		)

		_ = result
		_ = pkg
	})
}
