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
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

type registeredMetadata struct {
	Title       string
	Description string
}

type registeredNoProps struct{}

type registeredRequestData struct {
	Path string
}

func TestRegisteredType_StructLiteral_PreservesTypeAndFields(t *testing.T) {
	t.Parallel()

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/sdk": {
			"Metadata":    reflect.ValueOf((*registeredMetadata)(nil)),
			"NoProps":     reflect.ValueOf((*registeredNoProps)(nil)),
			"RequestData": reflect.ValueOf((*registeredRequestData)(nil)),
		},
	})

	source := `package main

import "example.com/sdk"

func entrypoint() sdk.Metadata {
	return sdk.Metadata{
		Title:       "My Title",
		Description: "My Description",
	}
}

func main() {}
`
	service := NewService()
	service.UseSymbols(symbols)
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)

	result, err := service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)

	meta, ok := result.(registeredMetadata)
	require.True(t, ok, "expected registeredMetadata, got %T", result)
	require.Equal(t, "My Title", meta.Title)
	require.Equal(t, "My Description", meta.Description)
}

func TestRegisteredType_MultiReturn_ViaClosureCallback(t *testing.T) {
	t.Parallel()

	type callResult struct {
		response any
		metadata registeredMetadata
		err      error
	}
	var captured callResult

	callRender := func(fn any) {
		fnValue := reflect.ValueOf(fn)
		results := fnValue.Call([]reflect.Value{
			reflect.ValueOf(&registeredRequestData{Path: "/test"}),
			reflect.ValueOf(registeredNoProps{}),
		})
		if len(results) >= 2 {
			captured.response = results[0].Interface()
			if meta, ok := results[1].Interface().(registeredMetadata); ok {
				captured.metadata = meta
			}
		}
		if len(results) >= 3 && !results[2].IsNil() {
			if e, ok := results[2].Interface().(error); ok {
				captured.err = e
			}
		}
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/sdk": {
			"Metadata":    reflect.ValueOf((*registeredMetadata)(nil)),
			"NoProps":     reflect.ValueOf((*registeredNoProps)(nil)),
			"RequestData": reflect.ValueOf((*registeredRequestData)(nil)),
		},
		"example.com/harness": {
			"CallRender": reflect.ValueOf(callRender),
		},
	})

	source := `package main

import (
	"example.com/sdk"
	"example.com/harness"
)

type Response struct {
	Heading string
	Count   int
}

func Render(r *sdk.RequestData, props sdk.NoProps) (Response, sdk.Metadata, error) {
	return Response{Heading: "Welcome", Count: 42}, sdk.Metadata{
		Title:       "Welcome Page",
		Description: "A welcome page",
	}, nil
}

func entrypoint() {
	harness.CallRender(Render)
}

func main() {}
`
	service := NewService()
	service.UseSymbols(symbols)
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)

	_, err = service.ExecuteEntrypoint(context.Background(), cfs, "entrypoint")
	require.NoError(t, err)

	require.Equal(t, "Welcome Page", captured.metadata.Title,
		"Metadata.Title must be preserved through closure multi-return")
	require.Equal(t, "A welcome page", captured.metadata.Description,
		"Metadata.Description must be preserved through closure multi-return")
	require.NoError(t, captured.err)
}

func TestRegisteredType_InitRegistersClosureMultiReturn(t *testing.T) {
	t.Parallel()

	var registeredFn any

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/sdk": {
			"Metadata":    reflect.ValueOf((*registeredMetadata)(nil)),
			"NoProps":     reflect.ValueOf((*registeredNoProps)(nil)),
			"RequestData": reflect.ValueOf((*registeredRequestData)(nil)),
		},
		"example.com/registry": {
			"RegisterFunc": reflect.ValueOf(func(name string, fn any) {
				registeredFn = fn
			}),
		},
	})

	source := `package main

import (
	"example.com/sdk"
	"example.com/registry"
)

type Response struct {
	Heading string
}

func init() {
	registry.RegisterFunc("Render", func(r *sdk.RequestData, props sdk.NoProps) (Response, sdk.Metadata, error) {
		return Response{Heading: "Welcome"}, sdk.Metadata{Title: "Welcome Page"}, nil
	})
}

func main() {}
`
	service := NewService()
	service.UseSymbols(symbols)
	cfs, err := service.CompileFileSet(context.Background(), map[string]string{"main.go": source})
	require.NoError(t, err)

	err = service.ExecuteInits(context.Background(), cfs)
	require.NoError(t, err)

	require.NotNil(t, registeredFn, "init() should have registered a function")

	fnValue := reflect.ValueOf(registeredFn)
	results := fnValue.Call([]reflect.Value{
		reflect.ValueOf(&registeredRequestData{Path: "/"}),
		reflect.ValueOf(registeredNoProps{}),
	})

	require.Len(t, results, 3, "expected 3 return values")

	meta, ok := results[1].Interface().(registeredMetadata)
	require.True(t, ok, "expected registeredMetadata at index 1, got %T", results[1].Interface())
	require.Equal(t, "Welcome Page", meta.Title,
		"Metadata.Title must be preserved when closure is called via reflect")
}
