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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func validTypeData() *inspector_dto.TypeData {
	return &inspector_dto.TypeData{
		Packages: map[string]*inspector_dto.Package{
			"my/pkg": {
				Name: "pkg",
				Path: "my/pkg",
				FileImports: map[string]map[string]string{
					"/src/user.go": {"fmt": "fmt"},
				},
				NamedTypes: map[string]*inspector_dto.Type{
					"User": {
						Name:                 "User",
						TypeString:           "User",
						UnderlyingTypeString: "struct{Name string}",
						DefinedInFilePath:    "/src/user.go",
						DefinitionLine:       5,
						DefinitionColumn:     6,
						Fields: []*inspector_dto.Field{
							{
								Name:                     "Name",
								TypeString:               "string",
								UnderlyingTypeString:     "string",
								PackagePath:              "",
								DeclaringPackagePath:     "my/pkg",
								DeclaringTypeName:        "User",
								DefinitionFilePath:       "/src/user.go",
								DefinitionLine:           6,
								DefinitionColumn:         2,
								IsUnderlyingPrimitive:    true,
								IsUnderlyingInternalType: true,
								IsInternalType:           true,
							},
						},
						Methods: []*inspector_dto.Method{
							{
								Name:                 "GetName",
								DeclaringPackagePath: "my/pkg",
								DeclaringTypeName:    "User",
								Signature:            inspector_dto.FunctionSignature{},
							},
						},
					},
				},
				Funcs: map[string]*inspector_dto.Function{
					"NewUser": {
						Name:      "NewUser",
						Signature: inspector_dto.FunctionSignature{},
					},
				},
			},
		},
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	t.Run("should pass for valid TypeData", func(t *testing.T) {
		t.Parallel()
		err := validate(validTypeData())
		assert.NoError(t, err)
	})

	t.Run("should fail for nil TypeData", func(t *testing.T) {
		t.Parallel()
		err := validate(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TypeData artefact is nil")
	})

	t.Run("should fail for nil Packages map", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{Packages: nil}
		err := validate(td)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Packages map is nil")
	})

	t.Run("should collect multiple errors across packages", func(t *testing.T) {
		t.Parallel()
		td := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"a": nil,
				"b": nil,
			},
		}
		err := validate(td)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "2 error(s)")
	})
}

func TestValidatePackage(t *testing.T) {
	t.Parallel()
	t.Run("should fail for nil package", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validatePackage(nil, "my/pkg", collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "package data is nil")
	})

	t.Run("should fail for empty Name", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		pkg := &inspector_dto.Package{
			Name:        "",
			Path:        "my/pkg",
			FileImports: map[string]map[string]string{},
			NamedTypes:  map[string]*inspector_dto.Type{},
			Funcs:       map[string]*inspector_dto.Function{},
		}
		validatePackage(pkg, "my/pkg", collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "empty Name")
	})

	t.Run("should fail for mismatched Path", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		pkg := &inspector_dto.Package{
			Name:        "pkg",
			Path:        "wrong/path",
			FileImports: map[string]map[string]string{},
			NamedTypes:  map[string]*inspector_dto.Type{},
			Funcs:       map[string]*inspector_dto.Function{},
		}
		validatePackage(pkg, "my/pkg", collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "incorrect or empty Path")
	})

	t.Run("should fail for nil FileImports", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		pkg := &inspector_dto.Package{
			Name:        "pkg",
			Path:        "my/pkg",
			FileImports: nil,
			NamedTypes:  map[string]*inspector_dto.Type{},
			Funcs:       map[string]*inspector_dto.Function{},
		}
		validatePackage(pkg, "my/pkg", collector)
		require.NotEmpty(t, collector.errors)
		hasFileImportsErr := false
		for _, e := range collector.errors {
			if assert.ObjectsAreEqual("", "") {
				_ = e
			}
			if contains(e, "nil FileImports") {
				hasFileImportsErr = true
			}
		}
		assert.True(t, hasFileImportsErr)
	})

	t.Run("should fail for nil NamedTypes", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		pkg := &inspector_dto.Package{
			Name:        "pkg",
			Path:        "my/pkg",
			FileImports: map[string]map[string]string{},
			NamedTypes:  nil,
			Funcs:       map[string]*inspector_dto.Function{},
		}
		validatePackage(pkg, "my/pkg", collector)
		require.NotEmpty(t, collector.errors)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "nil NamedTypes") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for nil Funcs", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		pkg := &inspector_dto.Package{
			Name:        "pkg",
			Path:        "my/pkg",
			FileImports: map[string]map[string]string{},
			NamedTypes:  map[string]*inspector_dto.Type{},
			Funcs:       nil,
		}
		validatePackage(pkg, "my/pkg", collector)
		require.NotEmpty(t, collector.errors)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "nil Funcs") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})
}

func TestValidateType(t *testing.T) {
	t.Parallel()
	t.Run("should fail for nil type", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(nil, "my/pkg", "User", collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "type data is nil")
	})

	t.Run("should fail for empty Name", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(&inspector_dto.Type{
			Name:                 "",
			TypeString:           "User",
			UnderlyingTypeString: "struct{}",
			DefinedInFilePath:    "/src/user.go",
		}, "my/pkg", "User", collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "incorrect or empty Name")
	})

	t.Run("should fail for mismatched Name", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(&inspector_dto.Type{
			Name:                 "Wrong",
			TypeString:           "Wrong",
			UnderlyingTypeString: "struct{}",
			DefinedInFilePath:    "/src/user.go",
		}, "my/pkg", "User", collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "incorrect or empty Name")
	})

	t.Run("should fail for empty DefinedInFilePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(&inspector_dto.Type{
			Name:                 "User",
			TypeString:           "User",
			UnderlyingTypeString: "struct{}",
			DefinedInFilePath:    "",
		}, "my/pkg", "User", collector)
		require.NotEmpty(t, collector.errors)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty DefinedInFilePath") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty TypeString", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(&inspector_dto.Type{
			Name:                 "User",
			TypeString:           "",
			UnderlyingTypeString: "struct{}",
			DefinedInFilePath:    "/src/user.go",
		}, "my/pkg", "User", collector)
		require.NotEmpty(t, collector.errors)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty TypeString") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty UnderlyingTypeString", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(&inspector_dto.Type{
			Name:                 "User",
			TypeString:           "User",
			UnderlyingTypeString: "",
			DefinedInFilePath:    "/src/user.go",
		}, "my/pkg", "User", collector)
		require.NotEmpty(t, collector.errors)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty UnderlyingTypeString") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should validate fields within type", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(&inspector_dto.Type{
			Name:                 "User",
			TypeString:           "User",
			UnderlyingTypeString: "struct{}",
			DefinedInFilePath:    "/src/user.go",
			Fields:               []*inspector_dto.Field{nil},
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "nil field") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should validate methods within type", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateType(&inspector_dto.Type{
			Name:                 "User",
			TypeString:           "User",
			UnderlyingTypeString: "struct{}",
			DefinedInFilePath:    "/src/user.go",
			Methods:              []*inspector_dto.Method{nil},
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "nil method") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})
}

func TestValidateField(t *testing.T) {
	t.Parallel()
	t.Run("should fail for nil field", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateField(nil, "my/pkg", "User", collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "nil field")
	})

	t.Run("should fail for empty Name", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateField(&inspector_dto.Field{
			Name:                     "",
			TypeString:               "string",
			UnderlyingTypeString:     "string",
			DeclaringPackagePath:     "my/pkg",
			DeclaringTypeName:        "User",
			IsUnderlyingPrimitive:    true,
			IsUnderlyingInternalType: true,
			IsInternalType:           true,
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty Name") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty TypeString", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateField(&inspector_dto.Field{
			Name:                     "Foo",
			TypeString:               "",
			UnderlyingTypeString:     "string",
			DeclaringPackagePath:     "my/pkg",
			DeclaringTypeName:        "User",
			IsUnderlyingPrimitive:    true,
			IsUnderlyingInternalType: true,
			IsInternalType:           true,
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty TypeString") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty UnderlyingTypeString", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateField(&inspector_dto.Field{
			Name:                     "Foo",
			TypeString:               "string",
			UnderlyingTypeString:     "",
			DeclaringPackagePath:     "my/pkg",
			DeclaringTypeName:        "User",
			IsUnderlyingPrimitive:    true,
			IsUnderlyingInternalType: true,
			IsInternalType:           true,
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty UnderlyingTypeString") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})
}

func TestValidateUnderlyingTypeConsistency(t *testing.T) {
	t.Parallel()
	t.Run("should fail when primitive but not internal", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateUnderlyingTypeConsistency(true, false, validationContext{kind: vctxField, name: "test field"}, collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "is_underlying_primitive=true but is_underlying_internal_type=false")
	})

	t.Run("should pass when both true", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateUnderlyingTypeConsistency(true, true, validationContext{kind: vctxField, name: "test field"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should pass when both false", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateUnderlyingTypeConsistency(false, false, validationContext{kind: vctxField, name: "test field"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should pass when not primitive but internal", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateUnderlyingTypeConsistency(false, true, validationContext{kind: vctxField, name: "test field"}, collector)
		assert.Empty(t, collector.errors)
	})
}

func TestValidateFieldDeclaringInfo(t *testing.T) {
	t.Parallel()
	t.Run("should fail for empty DeclaringPackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			DeclaringPackagePath: "",
			DeclaringTypeName:    "User",
		}
		validateFieldDeclaringInfo(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "empty DeclaringPackagePath")
	})

	t.Run("should fail for mismatched DeclaringPackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			DeclaringPackagePath: "other/pkg",
			DeclaringTypeName:    "User",
		}
		validateFieldDeclaringInfo(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "DeclaringPackagePath 'other/pkg' but should be 'my/pkg'")
	})

	t.Run("should fail for empty DeclaringTypeName", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			DeclaringPackagePath: "my/pkg",
			DeclaringTypeName:    "",
		}
		validateFieldDeclaringInfo(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "empty DeclaringTypeName")
	})

	t.Run("should fail for mismatched DeclaringTypeName", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			DeclaringPackagePath: "my/pkg",
			DeclaringTypeName:    "Admin",
		}
		validateFieldDeclaringInfo(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "DeclaringTypeName 'Admin' but should be 'User'")
	})

	t.Run("should pass for correct declaring info", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			DeclaringPackagePath: "my/pkg",
			DeclaringTypeName:    "User",
		}
		validateFieldDeclaringInfo(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})
}

func TestValidateFieldCompositeParts(t *testing.T) {
	t.Parallel()
	t.Run("should fail for parts with CompositeTypeNone", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			Name:          "Data",
			CompositeType: inspector_dto.CompositeTypeNone,
			CompositeParts: []*inspector_dto.CompositePart{
				{
					Role:                     "elem",
					TypeString:               "string",
					UnderlyingTypeString:     "string",
					IsUnderlyingPrimitive:    true,
					IsUnderlyingInternalType: true,
				},
			},
		}
		validateFieldCompositeParts(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "composite parts but CompositeType is None") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for composite type with no parts (non-signature)", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			Name:           "Data",
			CompositeType:  inspector_dto.CompositeTypeSlice,
			CompositeParts: nil,
		}
		validateFieldCompositeParts(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "composite but has no CompositeParts") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should allow signature type with no parts", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			Name:           "Handler",
			CompositeType:  inspector_dto.CompositeTypeSignature,
			CompositeParts: nil,
		}
		validateFieldCompositeParts(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should validate each composite part", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		field := &inspector_dto.Field{
			Name:          "Data",
			CompositeType: inspector_dto.CompositeTypeSlice,
			CompositeParts: []*inspector_dto.CompositePart{
				nil,
			},
		}
		validateFieldCompositeParts(field, "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "nil composite part") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})
}

func TestValidateCompositePart(t *testing.T) {
	t.Parallel()
	t.Run("should fail for nil part", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateCompositePart(nil, "Data", "my/pkg", "User", 0, collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "nil composite part at index 0")
	})

	t.Run("should fail for empty Role", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			Role:                     "",
			TypeString:               "string",
			UnderlyingTypeString:     "string",
			IsUnderlyingPrimitive:    true,
			IsUnderlyingInternalType: true,
		}
		validateCompositePart(part, "Data", "my/pkg", "User", 0, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty Role") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty TypeString", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			Role:                     "elem",
			TypeString:               "",
			UnderlyingTypeString:     "string",
			IsUnderlyingPrimitive:    true,
			IsUnderlyingInternalType: true,
		}
		validateCompositePart(part, "Data", "my/pkg", "User", 0, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty TypeString") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty UnderlyingTypeString", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			Role:                     "elem",
			TypeString:               "string",
			UnderlyingTypeString:     "",
			IsUnderlyingPrimitive:    true,
			IsUnderlyingInternalType: true,
		}
		validateCompositePart(part, "Data", "my/pkg", "User", 0, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty UnderlyingTypeString") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should check underlying type consistency in parts", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			Role:                     "elem",
			TypeString:               "string",
			UnderlyingTypeString:     "string",
			IsUnderlyingPrimitive:    true,
			IsUnderlyingInternalType: false,
		}
		validateCompositePart(part, "Data", "my/pkg", "User", 0, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "is_underlying_primitive=true but is_underlying_internal_type=false") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})
}

func TestValidateNestedCompositeParts(t *testing.T) {
	t.Parallel()
	t.Run("should skip when CompositeType is None", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			CompositeType: inspector_dto.CompositeTypeNone,
		}
		validateNestedCompositeParts(part, "Data", "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should fail when composite has no parts (non-signature)", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			CompositeType:  inspector_dto.CompositeTypeSlice,
			CompositeParts: nil,
		}
		validateNestedCompositeParts(part, "Data", "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "composite but has no CompositeParts")
	})

	t.Run("should allow signature type with no parts", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			CompositeType:  inspector_dto.CompositeTypeSignature,
			CompositeParts: nil,
		}
		validateNestedCompositeParts(part, "Data", "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should recurse into nested parts", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		part := &inspector_dto.CompositePart{
			Role:          "elem",
			CompositeType: inspector_dto.CompositeTypeSlice,
			CompositeParts: []*inspector_dto.CompositePart{
				nil,
			},
		}
		validateNestedCompositeParts(part, "Data", "my/pkg", "User", validationContext{kind: vctxField, name: "test"}, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "nil composite part") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})
}

func TestValidateMethod(t *testing.T) {
	t.Parallel()
	t.Run("should fail for nil method", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateMethod(nil, "my/pkg", "User", collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "nil method")
	})

	t.Run("should fail for empty Name", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateMethod(&inspector_dto.Method{
			Name:                 "",
			DeclaringPackagePath: "my/pkg",
			DeclaringTypeName:    "User",
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty Name") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty DeclaringPackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateMethod(&inspector_dto.Method{
			Name:                 "GetName",
			DeclaringPackagePath: "",
			DeclaringTypeName:    "User",
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty DeclaringPackagePath") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for empty DeclaringTypeName", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateMethod(&inspector_dto.Method{
			Name:                 "GetName",
			DeclaringPackagePath: "my/pkg",
			DeclaringTypeName:    "",
		}, "my/pkg", "User", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "empty DeclaringTypeName") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should pass for valid method", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateMethod(&inspector_dto.Method{
			Name:                 "GetName",
			DeclaringPackagePath: "my/pkg",
			DeclaringTypeName:    "User",
			Signature:            inspector_dto.FunctionSignature{},
		}, "my/pkg", "User", collector)
		assert.Empty(t, collector.errors)
	})
}

func TestValidateFunc(t *testing.T) {
	t.Parallel()
	t.Run("should fail for nil function", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateFunc(nil, "my/pkg", "NewUser", collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "function data is nil")
	})

	t.Run("should fail for empty Name", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateFunc(&inspector_dto.Function{
			Name: "",
		}, "my/pkg", "NewUser", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "incorrect or empty Name") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should fail for mismatched Name", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateFunc(&inspector_dto.Function{
			Name: "OtherFunc",
		}, "my/pkg", "NewUser", collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "incorrect or empty Name") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should pass for valid function", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateFunc(&inspector_dto.Function{
			Name:      "NewUser",
			Signature: inspector_dto.FunctionSignature{},
		}, "my/pkg", "NewUser", collector)
		assert.Empty(t, collector.errors)
	})
}

func TestValidateSignature(t *testing.T) {
	t.Parallel()
	t.Run("should fail for nil signature", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateSignature(nil, validationContext{kind: vctxFunc, name: "test func"}, collector)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "nil Signature")
	})

	t.Run("should pass for valid signature", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		validateSignature(&inspector_dto.FunctionSignature{}, validationContext{kind: vctxFunc, name: "test func"}, collector)
		assert.Empty(t, collector.errors)
	})
}

func TestValidatePackagePathConsistency(t *testing.T) {
	t.Parallel()
	t.Run("should require PackagePath for named types", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "User",
			PackagePath: "",
		}
		validatePackagePathConsistency(input, validationContext{kind: vctxField, name: "test"}, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "named type") && contains(e, "non-empty PackagePath") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should pass for named type with PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "User",
			PackagePath: "my/pkg",
		}
		validatePackagePathConsistency(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should pass for primitive type without PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "string",
			PackagePath: "",
		}
		validatePackagePathConsistency(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should fail for primitive type with PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "string",
			PackagePath: "my/pkg",
		}
		validatePackagePathConsistency(input, validationContext{kind: vctxField, name: "test"}, collector)
		hasErr := false
		for _, e := range collector.errors {
			if contains(e, "primitive or literal") && contains(e, "empty PackagePath") {
				hasErr = true
			}
		}
		assert.True(t, hasErr)
	})

	t.Run("should allow internal type without PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:     "User",
			PackagePath:    "",
			IsInternalType: true,
		}
		validatePackagePathConsistency(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})
}

func TestValidateGenericPlaceholderPackagePath(t *testing.T) {
	t.Parallel()
	t.Run("should return false for non-generic field", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			IsGenericPlaceholder: false,
		}
		result := validateGenericPlaceholderPackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.False(t, result)
		assert.Empty(t, collector.errors)
	})

	t.Run("should return true and pass for generic with empty PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:           "T",
			PackagePath:          "",
			IsGenericPlaceholder: true,
		}
		result := validateGenericPlaceholderPackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.True(t, result)
		assert.Empty(t, collector.errors)
	})

	t.Run("should return true and fail for generic with PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:           "T",
			PackagePath:          "my/pkg",
			IsGenericPlaceholder: true,
		}
		result := validateGenericPlaceholderPackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.True(t, result)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "generic placeholder")
	})
}

func TestValidateCompositeWithGenericsPackagePath(t *testing.T) {
	t.Parallel()
	t.Run("should return false when TypeString contains dot", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString: "pkg.Type",
			CompositeParts: []*inspector_dto.CompositePart{
				{IsGenericPlaceholder: true},
			},
		}
		result := validateCompositeWithGenericsPackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.False(t, result)
	})

	t.Run("should return false when no generic placeholders in parts", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString: "[]string",
			CompositeParts: []*inspector_dto.CompositePart{
				{IsGenericPlaceholder: false},
			},
		}
		result := validateCompositeWithGenericsPackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.False(t, result)
	})

	t.Run("should return true and pass for composite generic with empty PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "[]T",
			PackagePath: "",
			CompositeParts: []*inspector_dto.CompositePart{
				{IsGenericPlaceholder: true},
			},
		}
		result := validateCompositeWithGenericsPackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.True(t, result)
		assert.Empty(t, collector.errors)
	})

	t.Run("should return true and fail for composite generic with PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "[]T",
			PackagePath: "my/pkg",
			CompositeParts: []*inspector_dto.CompositePart{
				{IsGenericPlaceholder: true},
			},
		}
		result := validateCompositeWithGenericsPackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.True(t, result)
		require.Len(t, collector.errors, 1)
		assert.Contains(t, collector.errors[0], "composite type containing generic placeholders")
	})
}

func TestValidatePrimitiveTypePackagePath(t *testing.T) {
	t.Parallel()
	t.Run("should pass when PackagePath is empty", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "string",
			PackagePath: "",
		}
		validatePrimitiveTypePackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should allow PackagePath for composite with named components (map)", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "map[string]pkg.Type",
			PackagePath: "my/pkg",
		}
		validatePrimitiveTypePackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should allow PackagePath for composite with named components (slice)", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "[]pkg.Type",
			PackagePath: "my/pkg",
		}
		validatePrimitiveTypePackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should fail for non-alias primitive with PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:  "int",
			PackagePath: "my/pkg",
			IsAlias:     false,
		}
		validatePrimitiveTypePackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "primitive or literal type")
	})

	t.Run("should pass for alias to non-primitive with PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:            "int",
			PackagePath:           "my/pkg",
			IsAlias:               true,
			IsUnderlyingPrimitive: false,
		}
		validatePrimitiveTypePackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		assert.Empty(t, collector.errors)
	})

	t.Run("should fail for alias to primitive with PackagePath", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		input := packagePathInput{
			TypeString:            "int",
			PackagePath:           "my/pkg",
			IsAlias:               true,
			IsUnderlyingPrimitive: true,
		}
		validatePrimitiveTypePackagePath(input, validationContext{kind: vctxField, name: "test"}, collector)
		require.NotEmpty(t, collector.errors)
		assert.Contains(t, collector.errors[0], "alias to a primitive type")
	})
}

func TestIsCompositeTypeWithNamedComponents(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		typeString string
		expected   bool
	}{
		{
			name:       "map with named value type",
			typeString: "map[string]pkg.Type",
			expected:   true,
		},
		{
			name:       "map with named key type",
			typeString: "map[pkg.Key]string",
			expected:   true,
		},
		{
			name:       "map with only primitives",
			typeString: "map[string]int",
			expected:   false,
		},
		{
			name:       "slice with named type",
			typeString: "[]pkg.Type",
			expected:   true,
		},
		{
			name:       "slice with primitive",
			typeString: "[]string",
			expected:   false,
		},
		{
			name:       "func with named type",
			typeString: "func(pkg.Type)",
			expected:   true,
		},
		{
			name:       "plain named type",
			typeString: "User",
			expected:   false,
		},
		{
			name:       "malformed map without closing bracket",
			typeString: "map[string",
			expected:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isCompositeTypeWithNamedComponents(tc.typeString)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContainsNamedType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		typeString string
		expected   bool
	}{
		{
			name:       "named type with dot",
			typeString: "pkg.Type",
			expected:   true,
		},
		{
			name:       "no dot",
			typeString: "string",
			expected:   false,
		},
		{
			name:       "dotted type is named",
			typeString: "time.Time",
			expected:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := containsNamedType(tc.typeString)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCompositePartsContainGenericPlaceholder(t *testing.T) {
	t.Parallel()
	t.Run("should return false for nil parts", func(t *testing.T) {
		t.Parallel()
		assert.False(t, compositePartsContainGenericPlaceholder(nil))
	})

	t.Run("should return false for empty parts", func(t *testing.T) {
		t.Parallel()
		assert.False(t, compositePartsContainGenericPlaceholder([]*inspector_dto.CompositePart{}))
	})

	t.Run("should return true for direct generic placeholder", func(t *testing.T) {
		t.Parallel()
		parts := []*inspector_dto.CompositePart{
			{IsGenericPlaceholder: true},
		}
		assert.True(t, compositePartsContainGenericPlaceholder(parts))
	})

	t.Run("should return true for nested generic placeholder", func(t *testing.T) {
		t.Parallel()
		parts := []*inspector_dto.CompositePart{
			{
				IsGenericPlaceholder: false,
				CompositeParts: []*inspector_dto.CompositePart{
					{IsGenericPlaceholder: true},
				},
			},
		}
		assert.True(t, compositePartsContainGenericPlaceholder(parts))
	})

	t.Run("should return false when no placeholder anywhere", func(t *testing.T) {
		t.Parallel()
		parts := []*inspector_dto.CompositePart{
			{
				IsGenericPlaceholder: false,
				CompositeParts: []*inspector_dto.CompositePart{
					{IsGenericPlaceholder: false},
				},
			},
		}
		assert.False(t, compositePartsContainGenericPlaceholder(parts))
	})

	t.Run("should skip nil parts in slice", func(t *testing.T) {
		t.Parallel()
		parts := []*inspector_dto.CompositePart{
			nil,
			{IsGenericPlaceholder: true},
		}
		assert.True(t, compositePartsContainGenericPlaceholder(parts))
	})

	t.Run("should return false when all parts are nil", func(t *testing.T) {
		t.Parallel()
		parts := []*inspector_dto.CompositePart{nil, nil}
		assert.False(t, compositePartsContainGenericPlaceholder(parts))
	})
}

func TestErrorCollector(t *testing.T) {
	t.Parallel()
	t.Run("should format errors with context", func(t *testing.T) {
		t.Parallel()
		collector := &errorCollector{}
		collector.add(validationContext{kind: vctxPackage, name: "my context"}, "something went wrong: %s", "details")
		require.Len(t, collector.errors, 1)
		assert.Equal(t, "package 'my context': something went wrong: details", collector.errors[0])
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstring(s, substr)
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
