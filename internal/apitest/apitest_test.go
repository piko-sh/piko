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

package apitest

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type testStruct struct {
	Name    string
	Count   int
	private bool
}

func (t testStruct) Public() string    { return t.Name }
func (t *testStruct) SetName(n string) { t.Name = n }

type testInterface interface {
	DoSomething(x int) (string, error)
	Simple()
}

func TestGetEntityType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entity   any
		expected string
	}{
		{name: "nil", entity: nil, expected: "Nil"},
		{name: "struct", entity: testStruct{}, expected: "Struct"},
		{name: "func", entity: func() {}, expected: "Func"},
		{name: "interface pointer", entity: (*testInterface)(nil), expected: "Interface"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getEntityType(tt.entity))
		})
	}
}

func TestFormatType(t *testing.T) {
	t.Parallel()

	t.Run("nil type", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "nil", formatType(nil))
	})

	t.Run("simple type", func(t *testing.T) {
		t.Parallel()
		typ := reflect.TypeFor[int]()
		assert.Equal(t, "int", formatType(typ))
	})

	t.Run("func type", func(t *testing.T) {
		t.Parallel()
		typ := reflect.TypeFor[func(int) string]()
		result := formatType(typ)
		assert.Contains(t, result, "func")
		assert.Contains(t, result, "int")
		assert.Contains(t, result, "string")
	})
}

func TestFormatMethodSignature(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expected   string
		params     []string
		results    []string
		isVariadic bool
	}{
		{name: "no params or results", expected: "()", params: nil, results: nil, isVariadic: false},
		{name: "params only", expected: "(int, string)", params: []string{"int", "string"}, results: nil, isVariadic: false},
		{name: "results only", expected: "() error", params: nil, results: []string{"error"}, isVariadic: false},
		{name: "params and result", expected: "(int) string", params: []string{"int"}, results: []string{"string"}, isVariadic: false},
		{name: "multiple results", expected: "(int) (string, error)", params: []string{"int"}, results: []string{"string", "error"}, isVariadic: false},
		{name: "variadic", expected: "(...string)", params: []string{"[]string"}, results: nil, isVariadic: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := formatMethodSignature(tt.params, tt.results, tt.isVariadic)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFunction(t *testing.T) {
	t.Parallel()

	t.Run("non-func type", func(t *testing.T) {
		t.Parallel()
		typ := reflect.TypeFor[int]()
		result := formatFunction(typ)
		assert.Equal(t, "int", result)
	})

	t.Run("func no params no results", func(t *testing.T) {
		t.Parallel()
		typ := reflect.TypeFor[func()]()
		result := formatFunction(typ)
		assert.Equal(t, "()", result)
	})

	t.Run("func with params and results", func(t *testing.T) {
		t.Parallel()
		typ := reflect.TypeFor[func(int, string) error]()
		result := formatFunction(typ)
		assert.Equal(t, "(int, string) error", result)
	})
}

func TestGenerateSignature(t *testing.T) {
	t.Parallel()

	t.Run("nil entity", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "nil", generateSignature("Nil", nil))
	})

	t.Run("struct entity", func(t *testing.T) {
		t.Parallel()
		sig := generateSignature("TestStruct", testStruct{})
		assert.Contains(t, sig, "struct {")
		assert.Contains(t, sig, "Name")
		assert.Contains(t, sig, "Count")
		assert.Contains(t, sig, "Public")
	})

	t.Run("func entity", func(t *testing.T) {
		t.Parallel()
		sig := generateSignature("MyFunc", func(int) string { return "" })
		assert.Contains(t, sig, "func")
		assert.Contains(t, sig, "int")
		assert.Contains(t, sig, "string")
	})

	t.Run("interface entity", func(t *testing.T) {
		t.Parallel()
		sig := generateSignature("MyIface", (*testInterface)(nil))
		assert.Contains(t, sig, "interface {")
		assert.Contains(t, sig, "DoSomething")
		assert.Contains(t, sig, "Simple")
	})
}

func TestGenerateStructSignature(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[testStruct]()
	ptrType := reflect.PointerTo(typ)
	sig := generateStructSignature(typ, ptrType)

	assert.True(t, strings.HasPrefix(sig, "struct {"))
	assert.True(t, strings.HasSuffix(sig, "}"))
	assert.Contains(t, sig, "Name string")
	assert.Contains(t, sig, "Count int")
	assert.Contains(t, sig, "unexported fields")
	assert.Contains(t, sig, "Public")
	assert.Contains(t, sig, "SetName")
}

func TestGenerateInterfaceSignature(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeFor[testInterface]()
	sig := generateInterfaceSignature(typ)

	assert.True(t, strings.HasPrefix(sig, "interface {"))
	assert.True(t, strings.HasSuffix(sig, "}"))
	assert.Contains(t, sig, "DoSomething")
	assert.Contains(t, sig, "Simple")
}

func TestGenerateAPISnapshotYAML_Empty(t *testing.T) {
	t.Parallel()

	node, err := generateAPISnapshotYAML(Surface{})

	require.NoError(t, err)
	require.NotNil(t, node)
	assert.Equal(t, yaml.MappingNode, node.Kind)
	assert.Empty(t, node.Content)
}

func TestGenerateAPISnapshotYAML_SortedKeys(t *testing.T) {
	t.Parallel()

	surface := Surface{
		"Zebra": 42,
		"Apple": "hello",
	}

	node, err := generateAPISnapshotYAML(surface)

	require.NoError(t, err)
	require.Len(t, node.Content, 4)

	assert.Equal(t, "Apple", node.Content[0].Value)
	assert.Equal(t, "Zebra", node.Content[2].Value)
}

func TestGenerateAPISnapshotYAML_FuncEntry(t *testing.T) {
	t.Parallel()

	surface := Surface{
		"MyFunc": func(int) string { return "" },
	}

	node, err := generateAPISnapshotYAML(surface)

	require.NoError(t, err)
	require.Len(t, node.Content, 2)

	assert.Equal(t, "MyFunc", node.Content[0].Value)
	assert.Contains(t, node.Content[0].LineComment, "Func")
	assert.Contains(t, node.Content[1].Value, "func")
}
