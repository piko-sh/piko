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

package config_domain

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type flatConfig struct {
	Name  string
	Value int
}

type nestedTestConfig struct {
	Host   string
	Nested struct {
		Deep struct{ Secret string }
		Port int
	}
}

type pointerTestConfig struct {
	Nested *struct{ Port int }
	Host   string
}

type noInitTestConfig struct {
	Nested *struct{ Port int } `noinit:"true"`
	Host   string
}

func TestWalkerFlatStruct(t *testing.T) {
	config := &flatConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	var visited []string
	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		visited = append(visited, keyPath)
		return nil
	}

	state := &walkState{processor: processor}
	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
	assert.Equal(t, []string{"Name", "Value"}, visited)
}

func TestWalkerNestedStruct(t *testing.T) {
	config := &nestedTestConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	var visited []string
	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		visited = append(visited, keyPath)
		return nil
	}

	state := &walkState{processor: processor}
	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
	assert.Contains(t, visited, "Host")
	assert.Contains(t, visited, "Nested.Port")
	assert.Contains(t, visited, "Nested.Deep.Secret")
}

func TestWalkerPointerStruct(t *testing.T) {
	config := &pointerTestConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	var visited []string
	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		visited = append(visited, keyPath)
		return nil
	}

	state := &walkState{processor: processor}
	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
	assert.Contains(t, visited, "Host")

	assert.Contains(t, visited, "Nested.Port")
	assert.NotNil(t, config.Nested)
}

func TestWalkerNoInit(t *testing.T) {
	config := &noInitTestConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	var visited []string
	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		visited = append(visited, keyPath)
		return nil
	}

	state := &walkState{processor: processor}
	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
	assert.Contains(t, visited, "Host")

	assert.Nil(t, config.Nested)
}

func TestWalkerNonStruct(t *testing.T) {
	loader := &Loader{opts: LoaderOptions{}}

	var visited []string
	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		visited = append(visited, keyPath)
		return nil
	}

	state := &walkState{processor: processor}
	err := loader.walk(reflect.ValueOf(new("not a struct")).Elem(), state)

	require.NoError(t, err)
	assert.Empty(t, visited)
}

func TestWalkerNilProcessor(t *testing.T) {
	config := &flatConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	state := &walkState{processor: nil}
	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
}

func TestWalkerProcessorError(t *testing.T) {
	config := &flatConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		if keyPath == "Value" {
			return assert.AnError
		}
		return nil
	}

	state := &walkState{processor: processor}
	err := loader.walk(reflect.ValueOf(config), state)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Value")
}

func TestWalkerUnexportedFields(t *testing.T) {
	type configWithUnexported struct {
		Public string
	}

	config := &configWithUnexported{}
	loader := &Loader{opts: LoaderOptions{}}

	var visited []string
	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		visited = append(visited, keyPath)
		return nil
	}

	state := &walkState{processor: processor}
	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
	assert.Equal(t, []string{"Public"}, visited)

}

func TestWalkerKeyPrefix(t *testing.T) {
	config := &flatConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	var visited []string
	processor := func(field *reflect.StructField, _ reflect.Value, _, keyPath string) error {
		visited = append(visited, keyPath)
		return nil
	}

	state := &walkState{processor: processor, keyPrefix: "Config"}
	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
	assert.Equal(t, []string{"Config.Name", "Config.Value"}, visited)
}

func TestWalkerSourceAttribution(t *testing.T) {
	config := &flatConfig{}
	loader := &Loader{opts: LoaderOptions{}}

	ctx := &LoadContext{
		FieldSources: make(map[string]string),
	}

	processor := func(field *reflect.StructField, value reflect.Value, _, keyPath string) error {
		if keyPath == "Name" {
			value.SetString("changed")
		}
		return nil
	}

	state := &walkState{
		processor: processor,
		ctx:       ctx,
		source:    "test-source",
	}

	err := loader.walk(reflect.ValueOf(config), state)

	require.NoError(t, err)
	assert.Equal(t, "test-source", ctx.FieldSources["Name"])

	_, hasValue := ctx.FieldSources["Value"]
	assert.False(t, hasValue)
}

func TestProcessorFuncSignature(t *testing.T) {

	var p processorFunc = func(field *reflect.StructField, value reflect.Value, prefix, keyPath string) error {
		assert.NotNil(t, field)
		assert.True(t, value.IsValid())
		assert.NotEmpty(t, keyPath)
		return nil
	}

	config := &flatConfig{Name: "test"}
	fieldValue := reflect.ValueOf(config).Elem().Field(0)

	err := p(new(reflect.TypeFor[flatConfig]().Field(0)), fieldValue, "", "Name")
	require.NoError(t, err)
}

func TestWalkState(t *testing.T) {
	ctx := &LoadContext{
		FieldSources: make(map[string]string),
	}

	state := &walkState{
		processor: func(field *reflect.StructField, value reflect.Value, prefix, keyPath string) error {
			return nil
		},
		ctx:       ctx,
		keyPrefix: "Config",
		source:    "env: TEST",
	}

	assert.NotNil(t, state.processor)
	assert.Equal(t, ctx, state.ctx)
	assert.Equal(t, "Config", state.keyPrefix)
	assert.Equal(t, "env: TEST", state.source)
}
