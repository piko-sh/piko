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
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"

	"piko.sh/piko/wdk/interp/interp_link"
)

var errTestEmptyKey = errors.New("empty key")

func linkedFixtureTypesPackage(t *testing.T, moduleRoot string) *packages.Package {
	t.Helper()
	config := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedName | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedFiles,
		Dir:  moduleRoot,
	}
	loaded, err := packages.Load(config, "./...")
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.Empty(t, loaded[0].Errors, "fixture package should type-check")
	return loaded[0]
}

func writeLinkedFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relative, content := range files {
		full := filepath.Join(root, relative)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	return root
}

func TestLinkedGenericDispatchesThroughSibling(t *testing.T) {
	t.Parallel()

	root := writeLinkedFixture(t, map[string]string{
		"go.mod": "module example.com/linked\n\ngo 1.22\n",
		"linked.go": `package linked

import "reflect"

//piko:link GetItemLink
func GetItem[T any](label string) T {
	var zero T
	return zero
}

func GetItemLink(tType reflect.Type, label string) reflect.Value {
	if tType.Kind() == reflect.String {
		return reflect.ValueOf(label + "!").Convert(tType)
	}
	return reflect.New(tType).Elem()
}
`,
	})
	pkg := linkedFixtureTypesPackage(t, root)

	getItemStub := func(label string) string { return "" }

	getItemLink := func(tType reflect.Type, label string) reflect.Value {
		if tType.Kind() == reflect.String {
			return reflect.ValueOf(label + "!").Convert(tType)
		}
		return reflect.New(tType).Elem()
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/linked": {
			"GetItem":     reflect.ValueOf(interp_link.Wrap(1, getItemLink)),
			"GetItemLink": reflect.ValueOf(getItemStub),
		},
	})
	symbols.RegisterTypesPackage("example.com/linked", pkg.Types)

	service := NewService()
	service.UseSymbols(symbols)

	source := `package main

import "example.com/linked"

func run() string {
	return linked.GetItem[string]("hello")
}

func main() {}
`

	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "hello!", result)
}

func TestLinkedGenericWithUserStructType(t *testing.T) {
	t.Parallel()

	root := writeLinkedFixture(t, map[string]string{
		"go.mod": "module example.com/userlinked\n\ngo 1.22\n",
		"linked.go": `package userlinked

import "reflect"

//piko:link BuildLink
func Build[T any](title string) T {
	var zero T
	return zero
}

func BuildLink(tType reflect.Type, title string) reflect.Value {
	instance := reflect.New(tType).Elem()
	if field := instance.FieldByName("Title"); field.IsValid() && field.CanSet() {
		field.SetString(title)
	}
	return instance
}
`,
	})
	pkg := linkedFixtureTypesPackage(t, root)

	buildStub := func(title string) any { return nil }
	buildLink := func(tType reflect.Type, title string) reflect.Value {
		instance := reflect.New(tType).Elem()
		if field := instance.FieldByName("Title"); field.IsValid() && field.CanSet() {
			field.SetString(title)
		}
		return instance
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/userlinked": {
			"Build":     reflect.ValueOf(interp_link.Wrap(1, buildLink)),
			"BuildLink": reflect.ValueOf(buildStub),
		},
	})
	symbols.RegisterTypesPackage("example.com/userlinked", pkg.Types)

	service := NewService()
	service.UseSymbols(symbols)

	source := `package main

import "example.com/userlinked"

type Post struct {
	Title string
	Views int
}

func run() string {
	item := userlinked.Build[Post]("hello-from-user-type")
	return item.Title
}

func main() {}
`

	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "hello-from-user-type", result)
}

func TestLinkedGenericErrorReturn(t *testing.T) {
	t.Parallel()

	root := writeLinkedFixture(t, map[string]string{
		"go.mod": "module example.com/linkederr\n\ngo 1.22\n",
		"linked.go": `package linkederr

import (
	"errors"
	"reflect"
)

//piko:link FetchLink
func Fetch[T any](key string) (T, error) {
	var zero T
	return zero, errors.New("placeholder")
}

func FetchLink(tType reflect.Type, key string) (reflect.Value, error) {
	if key == "" {
		return reflect.Zero(tType), errors.New("empty key")
	}
	return reflect.ValueOf(key).Convert(tType), nil
}
`,
	})
	pkg := linkedFixtureTypesPackage(t, root)

	fetchStub := func(key string) (string, error) { return "", nil }
	fetchLink := func(tType reflect.Type, key string) (reflect.Value, error) {
		if key == "" {
			return reflect.Zero(tType), errTestEmptyKey
		}
		return reflect.ValueOf(key).Convert(tType), nil
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/linkederr": {
			"Fetch":     reflect.ValueOf(interp_link.Wrap(1, fetchLink)),
			"FetchLink": reflect.ValueOf(fetchStub),
		},
	})
	symbols.RegisterTypesPackage("example.com/linkederr", pkg.Types)

	service := NewService()
	service.UseSymbols(symbols)

	source := `package main

import "example.com/linkederr"

func run() string {
	value, err := linkederr.Fetch[string]("present")
	if err != nil {
		return "err:" + err.Error()
	}
	return value
}

func main() {}
`

	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "present", result)
}

func TestLinkedGenericVariadicWithSliceReturn(t *testing.T) {
	t.Parallel()

	type Option func(*struct{ Weight float64 })
	resultReflectType := func(tType reflect.Type) reflect.Type {
		return reflect.StructOf([]reflect.StructField{
			{Name: "Item", Type: tType},
			{Name: "Score", Type: reflect.TypeFor[float64]()},
		})
	}

	searchStub := func(label string, _ ...Option) (any, error) { return nil, nil }
	searchLink := func(tType reflect.Type, label string, opts ...Option) (reflect.Value, error) {
		weight := 1.0
		for _, opt := range opts {
			cfg := struct{ Weight float64 }{}
			opt(&cfg)
			if cfg.Weight != 0 {
				weight = cfg.Weight
			}
		}
		resultType := resultReflectType(tType)
		slice := reflect.MakeSlice(reflect.SliceOf(resultType), 0, 1)
		entry := reflect.New(resultType).Elem()
		item := reflect.New(tType).Elem()
		if field := item.FieldByName("Title"); field.IsValid() && field.CanSet() {
			field.SetString(label)
		}
		entry.FieldByName("Item").Set(item)
		entry.FieldByName("Score").Set(reflect.ValueOf(weight))
		return reflect.Append(slice, entry), nil
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/searchpkg": {
			"Result": reflect.ValueOf(interp_link.WrapType("Result", 1, []interp_link.GenericField{
				{
					Name:     "Item",
					Exported: true,
					FieldType: interp_link.GenericFieldType{
						Kind:         interp_link.FieldKindTypeArg,
						TypeArgIndex: 0,
					},
				},
				{
					Name:     "Score",
					Exported: true,
					FieldType: interp_link.GenericFieldType{
						Kind:      interp_link.FieldKindBasic,
						BasicKind: reflect.Float64,
					},
				},
			})),
			"Option":     reflect.ValueOf((*Option)(nil)),
			"WithWeight": reflect.ValueOf(func(w float64) Option { return func(c *struct{ Weight float64 }) { c.Weight = w } }),
			"Search": reflect.ValueOf(interp_link.WrapFunc(1, searchLink,
				[]interp_link.GenericFieldType{
					{Kind: interp_link.FieldKindBasic, BasicKind: reflect.String},
					{
						Kind: interp_link.FieldKindSlice,
						Element: &interp_link.GenericFieldType{
							Kind:         interp_link.FieldKindNamed,
							NamedPackage: "example.com/searchpkg",
							NamedName:    "Option",
						},
					},
				},
				[]interp_link.GenericFieldType{
					{
						Kind: interp_link.FieldKindSlice,
						Element: &interp_link.GenericFieldType{
							Kind:         interp_link.FieldKindNamedGeneric,
							NamedPackage: "example.com/searchpkg",
							NamedName:    "Result",
							TypeArgs: []interp_link.GenericFieldType{
								{Kind: interp_link.FieldKindTypeArg, TypeArgIndex: 0},
							},
						},
					},
					{Kind: interp_link.FieldKindError},
				},
				true,
			)),
			"SearchLink": reflect.ValueOf(searchStub),
		},
	})
	symbols.SynthesiseAll()

	service := NewService()
	service.UseSymbols(symbols)

	source := `package main

import "example.com/searchpkg"

type Doc struct {
	Title string
}

func run() string {
	results, err := searchpkg.Search[Doc]("hello", searchpkg.WithWeight(2.5))
	if err != nil {
		return "err:" + err.Error()
	}
	if len(results) == 0 {
		return "empty"
	}
	return results[0].Item.Title
}

func main() {}
`

	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "hello", result)
}

func TestLinkedGenericWithSynthesisedTypesPackage(t *testing.T) {
	t.Parallel()

	buildStub := func(title string) any { return nil }
	buildLink := func(tType reflect.Type, title string) reflect.Value {
		instance := reflect.New(tType).Elem()
		if field := instance.FieldByName("Title"); field.IsValid() && field.CanSet() {
			field.SetString(title)
		}
		return instance
	}

	symbols := NewSymbolRegistry(SymbolExports{
		"example.com/synth": {
			"Build":     reflect.ValueOf(interp_link.Wrap(1, buildLink)),
			"BuildLink": reflect.ValueOf(buildStub),
		},
	})
	symbols.SynthesiseAll()

	service := NewService()
	service.UseSymbols(symbols)

	source := `package main

import "example.com/synth"

type Post struct {
	Title string
}

func run() string {
	item := synth.Build[Post]("from-synth")
	return item.Title
}

func main() {}
`

	result, err := service.EvalFile(context.Background(), source, "run")
	require.NoError(t, err)
	require.Equal(t, "from-synth", result)
}
