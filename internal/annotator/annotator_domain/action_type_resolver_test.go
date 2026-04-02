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

package annotator_domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestIsPrimitive(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		typeString string
		expected   bool
	}{
		{name: "string is primitive", typeString: "string", expected: true},
		{name: "int is primitive", typeString: "int", expected: true},
		{name: "int8 is primitive", typeString: "int8", expected: true},
		{name: "int16 is primitive", typeString: "int16", expected: true},
		{name: "int32 is primitive", typeString: "int32", expected: true},
		{name: "int64 is primitive", typeString: "int64", expected: true},
		{name: "uint is primitive", typeString: "uint", expected: true},
		{name: "uint8 is primitive", typeString: "uint8", expected: true},
		{name: "uint16 is primitive", typeString: "uint16", expected: true},
		{name: "uint32 is primitive", typeString: "uint32", expected: true},
		{name: "uint64 is primitive", typeString: "uint64", expected: true},
		{name: "float32 is primitive", typeString: "float32", expected: true},
		{name: "float64 is primitive", typeString: "float64", expected: true},
		{name: "bool is primitive", typeString: "bool", expected: true},
		{name: "byte is primitive", typeString: "byte", expected: true},
		{name: "rune is primitive", typeString: "rune", expected: true},
		{name: "error is primitive", typeString: "error", expected: true},
		{name: "any is primitive", typeString: "any", expected: true},
		{name: "struct is not primitive", typeString: "MyStruct", expected: false},
		{name: "qualified type is not primitive", typeString: "pkg.Type", expected: false},
		{name: "slice is not primitive", typeString: "[]string", expected: false},
		{name: "map is not primitive", typeString: "map[string]int", expected: false},
		{name: "pointer is not primitive", typeString: "*int", expected: false},
		{name: "empty string is not primitive", typeString: "", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isPrimitive(tc.typeString)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractTypeName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		typeString string
		expected   string
	}{
		{
			name:       "simple type",
			typeString: "User",
			expected:   "User",
		},
		{
			name:       "qualified type",
			typeString: "pkg.User",
			expected:   "User",
		},
		{
			name:       "pointer type",
			typeString: "*User",
			expected:   "User",
		},
		{
			name:       "pointer to qualified type",
			typeString: "*pkg.User",
			expected:   "User",
		},
		{
			name:       "slice type",
			typeString: "[]User",
			expected:   "User",
		},
		{
			name:       "slice of qualified type",
			typeString: "[]pkg.User",
			expected:   "User",
		},
		{
			name:       "deeply nested qualified",
			typeString: "github.com/example/pkg.User",
			expected:   "User",
		},
		{
			name:       "primitive type",
			typeString: "string",
			expected:   "string",
		},
		{
			name:       "empty string",
			typeString: "",
			expected:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractTypeName(tc.typeString)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractJSONTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		rawTag   string
		expected string
	}{
		{
			name:     "simple json tag",
			rawTag:   "`json:\"username\"`",
			expected: "username",
		},
		{
			name:     "json tag with omitempty",
			rawTag:   "`json:\"email,omitempty\"`",
			expected: "email",
		},
		{
			name:     "json tag with hyphen (skip)",
			rawTag:   "`json:\"-\"`",
			expected: "-",
		},
		{
			name:     "multiple tags",
			rawTag:   "`json:\"name\" validate:\"required\"`",
			expected: "name",
		},
		{
			name:     "no json tag",
			rawTag:   "`validate:\"required\"`",
			expected: "",
		},
		{
			name:     "empty string",
			rawTag:   "",
			expected: "",
		},
		{
			name:     "tag without backticks",
			rawTag:   "json:\"value\"",
			expected: "value",
		},
		{
			name:     "json tag with multiple options",
			rawTag:   "`json:\"field,omitempty,string\"`",
			expected: "field",
		},
		{
			name:     "malformed tag - no closing quote",
			rawTag:   "`json:\"incomplete",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractJSONTag(tc.rawTag)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractValidateTag(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		rawTag   string
		expected string
	}{
		{
			name:     "simple validate tag",
			rawTag:   "`validate:\"required\"`",
			expected: "required",
		},
		{
			name:     "validate with multiple rules",
			rawTag:   "`validate:\"required,email\"`",
			expected: "required,email",
		},
		{
			name:     "validate with min/max",
			rawTag:   "`validate:\"min=1,max=100\"`",
			expected: "min=1,max=100",
		},
		{
			name:     "multiple tags with validate",
			rawTag:   "`json:\"email\" validate:\"required,email\"`",
			expected: "required,email",
		},
		{
			name:     "no validate tag",
			rawTag:   "`json:\"field\"`",
			expected: "",
		},
		{
			name:     "empty string",
			rawTag:   "",
			expected: "",
		},
		{
			name:     "tag without backticks",
			rawTag:   "validate:\"required\"",
			expected: "required",
		},
		{
			name:     "malformed tag - no closing quote",
			rawTag:   "`validate:\"incomplete",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := extractValidateTag(tc.rawTag)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGoTypeToTSType(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		goType   string
		expected string
	}{

		{name: "string", goType: "string", expected: "string"},

		{name: "int", goType: "int", expected: "number"},
		{name: "int8", goType: "int8", expected: "number"},
		{name: "int16", goType: "int16", expected: "number"},
		{name: "int32", goType: "int32", expected: "number"},
		{name: "int64", goType: "int64", expected: "number"},
		{name: "uint", goType: "uint", expected: "number"},
		{name: "uint8", goType: "uint8", expected: "number"},
		{name: "uint16", goType: "uint16", expected: "number"},
		{name: "uint32", goType: "uint32", expected: "number"},
		{name: "uint64", goType: "uint64", expected: "number"},

		{name: "float32", goType: "float32", expected: "number"},
		{name: "float64", goType: "float64", expected: "number"},

		{name: "bool", goType: "bool", expected: "boolean"},

		{name: "byte", goType: "byte", expected: "number"},
		{name: "rune", goType: "rune", expected: "number"},

		{name: "error", goType: "error", expected: "Error"},

		{name: "any", goType: "any", expected: "any"},
		{name: "interface{}", goType: "interface{}", expected: "any"},

		{name: "pointer to string", goType: "*string", expected: "string"},
		{name: "pointer to int", goType: "*int", expected: "number"},

		{name: "string slice", goType: "[]string", expected: "string[]"},
		{name: "int slice", goType: "[]int", expected: "number[]"},
		{name: "bool slice", goType: "[]bool", expected: "boolean[]"},

		{name: "map string to int", goType: "map[string]int", expected: "Record<string, any>"},
		{name: "map string to any", goType: "map[string]interface{}", expected: "Record<string, any>"},

		{name: "time.Time", goType: "time.Time", expected: "string"},
		{name: "qualified Time", goType: "custom.Time", expected: "string"},

		{name: "uuid.UUID", goType: "uuid.UUID", expected: "string"},
		{name: "google UUID", goType: "google.UUID", expected: "string"},

		{name: "custom struct", goType: "pkg.MyStruct", expected: "MyStruct"},
		{name: "unqualified struct", goType: "MyStruct", expected: "MyStruct"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := goTypeToTSType(tc.goType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsErrorTypeString(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		typeString string
		expected   bool
	}{
		{name: "error type", typeString: "error", expected: true},
		{name: "Error type", typeString: "Error", expected: false},
		{name: "string type", typeString: "string", expected: false},
		{name: "custom error", typeString: "MyError", expected: false},
		{name: "empty string", typeString: "", expected: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isErrorTypeString(tc.typeString)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestHasMethod(t *testing.T) {
	t.Parallel()

	typeWithMethods := &inspector_dto.Type{
		Name: "MyAction",
		Methods: []*inspector_dto.Method{
			{Name: "Call"},
			{Name: "Validate"},
			{Name: "Method"},
		},
	}

	typeWithNoMethods := &inspector_dto.Type{
		Name:    "EmptyType",
		Methods: []*inspector_dto.Method{},
	}

	testCases := []struct {
		name       string
		t          *inspector_dto.Type
		methodName string
		expected   bool
	}{
		{
			name:       "has Call method",
			t:          typeWithMethods,
			methodName: "Call",
			expected:   true,
		},
		{
			name:       "has Validate method",
			t:          typeWithMethods,
			methodName: "Validate",
			expected:   true,
		},
		{
			name:       "has Method method",
			t:          typeWithMethods,
			methodName: "Method",
			expected:   true,
		},
		{
			name:       "does not have Unknown method",
			t:          typeWithMethods,
			methodName: "Unknown",
			expected:   false,
		},
		{
			name:       "empty type has no methods",
			t:          typeWithNoMethods,
			methodName: "Call",
			expected:   false,
		},
		{
			name:       "case sensitive - lowercase",
			t:          typeWithMethods,
			methodName: "call",
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := hasMethod(tc.t, tc.methodName)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFindCallMethodInType(t *testing.T) {
	t.Parallel()

	callMethod := &inspector_dto.Method{
		Name: "Call",
		Signature: inspector_dto.FunctionSignature{
			Params:  []string{"Input"},
			Results: []string{"Output", "error"},
		},
	}

	typeWithCall := &inspector_dto.Type{
		Name: "MyAction",
		Methods: []*inspector_dto.Method{
			{Name: "Validate"},
			callMethod,
			{Name: "Method"},
		},
	}

	typeWithoutCall := &inspector_dto.Type{
		Name: "NoCallType",
		Methods: []*inspector_dto.Method{
			{Name: "Validate"},
			{Name: "Process"},
		},
	}

	testCases := []struct {
		t        *inspector_dto.Type
		expected *inspector_dto.Method
		name     string
	}{
		{
			name:     "finds Call method",
			t:        typeWithCall,
			expected: callMethod,
		},
		{
			name:     "returns nil when no Call method",
			t:        typeWithoutCall,
			expected: nil,
		},
		{
			name: "returns nil for empty methods",
			t: &inspector_dto.Type{
				Name:    "EmptyType",
				Methods: []*inspector_dto.Method{},
			},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := findCallMethodInType(tc.t)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsOptionalField(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		field    *inspector_dto.Field
		name     string
		expected bool
	}{
		{
			name: "pointer type is optional",
			field: &inspector_dto.Field{
				Name:       "Name",
				TypeString: "*string",
				RawTag:     "",
			},
			expected: true,
		},
		{
			name: "omitempty tag is optional",
			field: &inspector_dto.Field{
				Name:       "Email",
				TypeString: "string",
				RawTag:     "`json:\"email,omitempty\"`",
			},
			expected: true,
		},
		{
			name: "non-pointer non-omitempty is required",
			field: &inspector_dto.Field{
				Name:       "ID",
				TypeString: "int",
				RawTag:     "`json:\"id\"`",
			},
			expected: false,
		},
		{
			name: "no tag is required",
			field: &inspector_dto.Field{
				Name:       "Required",
				TypeString: "string",
				RawTag:     "",
			},
			expected: false,
		},
		{
			name: "pointer with omitempty is optional",
			field: &inspector_dto.Field{
				Name:       "Optional",
				TypeString: "*int",
				RawTag:     "`json:\"optional,omitempty\"`",
			},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := isOptionalField(tc.field)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractTypeInfoFromString(t *testing.T) {
	t.Parallel()

	t.Run("extracts primitive type info", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("string", nil)

		assert.Equal(t, "string", result.Name)
		assert.Equal(t, "string", result.TSType)
		assert.False(t, result.IsPointer)
		assert.Empty(t, result.PackagePath)
	})

	t.Run("extracts pointer type info", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("*string", nil)

		assert.Equal(t, "string", result.Name)
		assert.Equal(t, "string", result.TSType)
		assert.True(t, result.IsPointer)
	})

	t.Run("extracts qualified type without package lookup", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("pkg.User", nil)

		assert.Equal(t, "pkg.User", result.Name)
		assert.Equal(t, "User", result.TSType)
		assert.False(t, result.IsPointer)
	})

	t.Run("extracts int type", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("int", nil)

		assert.Equal(t, "int", result.Name)
		assert.Equal(t, "number", result.TSType)
		assert.False(t, result.IsPointer)
	})

	t.Run("extracts bool type", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("bool", nil)

		assert.Equal(t, "bool", result.Name)
		assert.Equal(t, "boolean", result.TSType)
		assert.False(t, result.IsPointer)
	})

	t.Run("extracts slice type", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("[]string", nil)

		assert.Equal(t, "[]string", result.Name)
		assert.Equal(t, "string[]", result.TSType)
		assert.False(t, result.IsPointer)
	})

	t.Run("extracts map type", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("map[string]int", nil)

		assert.Equal(t, "map[string]int", result.Name)
		assert.Equal(t, "Record<string, any>", result.TSType)
		assert.False(t, result.IsPointer)
	})

	t.Run("extracts type with package lookup", func(t *testing.T) {
		t.Parallel()

		packages := map[string]*inspector_dto.Package{
			"github.com/example/myapp/models": {
				Name: "models",
				NamedTypes: map[string]*inspector_dto.Type{
					"User": {
						Name:   "User",
						Fields: []*inspector_dto.Field{},
					},
				},
			},
		}

		result := extractTypeInfoFromString("models.User", packages)

		assert.Equal(t, "User", result.Name)
		assert.Equal(t, "github.com/example/myapp/models", result.PackagePath)
	})

	t.Run("extracts pointer to qualified type with package lookup", func(t *testing.T) {
		t.Parallel()

		packages := map[string]*inspector_dto.Package{
			"github.com/example/myapp/models": {
				Name: "models",
				NamedTypes: map[string]*inspector_dto.Type{
					"User": {
						Name:   "User",
						Fields: []*inspector_dto.Field{},
					},
				},
			},
		}

		result := extractTypeInfoFromString("*models.User", packages)

		assert.Equal(t, "User", result.Name)
		assert.True(t, result.IsPointer)
		assert.Equal(t, "github.com/example/myapp/models", result.PackagePath)
	})

	t.Run("handles empty packages map", func(t *testing.T) {
		t.Parallel()

		result := extractTypeInfoFromString("pkg.Custom", map[string]*inspector_dto.Package{})

		assert.Equal(t, "pkg.Custom", result.Name)
		assert.Equal(t, "Custom", result.TSType)
		assert.Empty(t, result.PackagePath)
	})
}

func TestDetectActionCapabilities(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		actionType *inspector_dto.Type
		expected   annotator_dto.ActionCapabilities
	}{
		{
			name: "has SSE capability",
			actionType: &inspector_dto.Type{
				Name: "StreamAction",
				Methods: []*inspector_dto.Method{
					{Name: "Call"},
					{Name: "StreamProgress"},
				},
			},
			expected: annotator_dto.ActionCapabilities{
				HasSSE: true,
			},
		},
		{
			name: "has multiple capabilities",
			actionType: &inspector_dto.Type{
				Name: "FullAction",
				Methods: []*inspector_dto.Method{
					{Name: "Call"},
					{Name: "Middlewares"},
					{Name: "RateLimit"},
					{Name: "CacheConfig"},
					{Name: "ResourceLimits"},
				},
			},
			expected: annotator_dto.ActionCapabilities{
				HasMiddlewares:    true,
				HasRateLimit:      true,
				HasCacheConfig:    true,
				HasResourceLimits: true,
			},
		},
		{
			name: "no capabilities",
			actionType: &inspector_dto.Type{
				Name: "BasicAction",
				Methods: []*inspector_dto.Method{
					{Name: "Call"},
				},
			},
			expected: annotator_dto.ActionCapabilities{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := detectActionCapabilities(tc.actionType)
			assert.Equal(t, tc.expected, result)
		})
	}
}
