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

func TestImplementationIndex(t *testing.T) {
	t.Parallel()
	t.Run("should find single implementor of an interface", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Reader": {
							Name:                 "Reader",
							UnderlyingTypeString: "interface{Read([]byte) (int, error)}",
							Methods: []*inspector_dto.Method{
								{
									Name: "Read",
									Signature: inspector_dto.FunctionSignature{
										Params:  []string{"[]byte"},
										Results: []string{"int", "error"},
									},
								},
							},
						},
						"FileReader": {
							Name:                 "FileReader",
							UnderlyingTypeString: "struct{path string}",
							DefinedInFilePath:    "/src/reader.go",
							DefinitionLine:       10,
							DefinitionColumn:     6,
							Methods: []*inspector_dto.Method{
								{
									Name: "Read",
									Signature: inspector_dto.FunctionSignature{
										Params:  []string{"[]byte"},
										Results: []string{"int", "error"},
									},
								},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/a", "Reader")

		require.Len(t, impls, 1)
		assert.Equal(t, "FileReader", impls[0].TypeName)
		assert.Equal(t, "pkg/a", impls[0].PackagePath)
		assert.Equal(t, "/src/reader.go", impls[0].DefinitionFile)
		assert.Equal(t, 10, impls[0].DefinitionLine)
		assert.Equal(t, 6, impls[0].DefinitionCol)
	})

	t.Run("should find multiple implementors of an interface", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/io": {
					Name: "io",
					Path: "pkg/io",
					NamedTypes: map[string]*inspector_dto.Type{
						"Writer": {
							Name:                 "Writer",
							UnderlyingTypeString: "interface{Write([]byte) (int, error)}",
							Methods: []*inspector_dto.Method{
								{
									Name: "Write",
									Signature: inspector_dto.FunctionSignature{
										Params:  []string{"[]byte"},
										Results: []string{"int", "error"},
									},
								},
							},
						},
						"FileWriter": {
							Name:                 "FileWriter",
							UnderlyingTypeString: "struct{}",
							Methods: []*inspector_dto.Method{
								{
									Name: "Write",
									Signature: inspector_dto.FunctionSignature{
										Params:  []string{"[]byte"},
										Results: []string{"int", "error"},
									},
								},
							},
						},
						"BufferWriter": {
							Name:                 "BufferWriter",
							UnderlyingTypeString: "struct{}",
							Methods: []*inspector_dto.Method{
								{
									Name: "Write",
									Signature: inspector_dto.FunctionSignature{
										Params:  []string{"[]byte"},
										Results: []string{"int", "error"},
									},
								},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/io", "Writer")

		require.Len(t, impls, 2)
		names := make(map[string]bool)
		for _, impl := range impls {
			names[impl.TypeName] = true
		}
		assert.True(t, names["FileWriter"])
		assert.True(t, names["BufferWriter"])
	})

	t.Run("should handle type implementing multiple interfaces", func(t *testing.T) {
		t.Parallel()
		readSig := inspector_dto.FunctionSignature{
			Params:  []string{"[]byte"},
			Results: []string{"int", "error"},
		}
		writeSig := inspector_dto.FunctionSignature{
			Params:  []string{"[]byte"},
			Results: []string{"int", "error"},
		}

		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/io": {
					Name: "io",
					Path: "pkg/io",
					NamedTypes: map[string]*inspector_dto.Type{
						"Reader": {
							Name:                 "Reader",
							UnderlyingTypeString: "interface{Read([]byte) (int, error)}",
							Methods:              []*inspector_dto.Method{{Name: "Read", Signature: readSig}},
						},
						"Writer": {
							Name:                 "Writer",
							UnderlyingTypeString: "interface{Write([]byte) (int, error)}",
							Methods:              []*inspector_dto.Method{{Name: "Write", Signature: writeSig}},
						},
						"ReadWriter": {
							Name:                 "ReadWriter",
							UnderlyingTypeString: "struct{}",
							Methods: []*inspector_dto.Method{
								{Name: "Read", Signature: readSig},
								{Name: "Write", Signature: writeSig},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)

		readerImpls := index.FindImplementations("pkg/io", "Reader")
		writerImpls := index.FindImplementations("pkg/io", "Writer")

		require.Len(t, readerImpls, 1)
		assert.Equal(t, "ReadWriter", readerImpls[0].TypeName)

		require.Len(t, writerImpls, 1)
		assert.Equal(t, "ReadWriter", writerImpls[0].TypeName)
	})

	t.Run("should not match empty interface", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Empty": {
							Name:                 "Empty",
							UnderlyingTypeString: "interface{}",
							Methods:              []*inspector_dto.Method{},
						},
						"Concrete": {
							Name:                 "Concrete",
							UnderlyingTypeString: "struct{}",
							Methods: []*inspector_dto.Method{
								{Name: "Foo", Signature: inspector_dto.FunctionSignature{Results: []string{"string"}}},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/a", "Empty")
		assert.Empty(t, impls)
	})

	t.Run("should not match type with missing method", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Doer": {
							Name:                 "Doer",
							UnderlyingTypeString: "interface{Do() error; Undo() error}",
							Methods: []*inspector_dto.Method{
								{Name: "Do", Signature: inspector_dto.FunctionSignature{Results: []string{"error"}}},
								{Name: "Undo", Signature: inspector_dto.FunctionSignature{Results: []string{"error"}}},
							},
						},
						"PartialDoer": {
							Name:                 "PartialDoer",
							UnderlyingTypeString: "struct{}",
							Methods: []*inspector_dto.Method{
								{Name: "Do", Signature: inspector_dto.FunctionSignature{Results: []string{"error"}}},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/a", "Doer")
		assert.Empty(t, impls)
	})

	t.Run("should not match type with wrong signature", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Stringer": {
							Name:                 "Stringer",
							UnderlyingTypeString: "interface{String() string}",
							Methods: []*inspector_dto.Method{
								{Name: "String", Signature: inspector_dto.FunctionSignature{Results: []string{"string"}}},
							},
						},
						"BadStringer": {
							Name:                 "BadStringer",
							UnderlyingTypeString: "struct{}",
							Methods: []*inspector_dto.Method{
								{Name: "String", Signature: inspector_dto.FunctionSignature{Results: []string{"int"}}},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/a", "Stringer")
		assert.Empty(t, impls)
	})

	t.Run("should handle cross-package implementation", func(t *testing.T) {
		t.Parallel()
		sig := inspector_dto.FunctionSignature{Results: []string{"string"}}
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/iface": {
					Name: "iface",
					Path: "pkg/iface",
					NamedTypes: map[string]*inspector_dto.Type{
						"Namer": {
							Name:                 "Namer",
							UnderlyingTypeString: "interface{Name() string}",
							Methods:              []*inspector_dto.Method{{Name: "Name", Signature: sig}},
						},
					},
				},
				"pkg/impl": {
					Name: "impl",
					Path: "pkg/impl",
					NamedTypes: map[string]*inspector_dto.Type{
						"Person": {
							Name:                 "Person",
							UnderlyingTypeString: "struct{}",
							Methods:              []*inspector_dto.Method{{Name: "Name", Signature: sig}},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/iface", "Namer")
		require.Len(t, impls, 1)
		assert.Equal(t, "Person", impls[0].TypeName)
		assert.Equal(t, "pkg/impl", impls[0].PackagePath)
	})

	t.Run("should handle nil TypeData", func(t *testing.T) {
		t.Parallel()
		index := NewImplementationIndex(nil)
		impls := index.FindImplementations("pkg/a", "Reader")
		assert.Empty(t, impls)
	})

	t.Run("should handle TypeData with nil Packages", func(t *testing.T) {
		t.Parallel()
		index := NewImplementationIndex(&inspector_dto.TypeData{Packages: nil})
		impls := index.FindImplementations("pkg/a", "Reader")
		assert.Empty(t, impls)
	})

	t.Run("should handle nil package in Packages map", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/nil": nil,
			},
		}
		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/nil", "Anything")
		assert.Empty(t, impls)
	})

	t.Run("should handle package with nil NamedTypes", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name:       "a",
					Path:       "pkg/a",
					NamedTypes: nil,
				},
			},
		}
		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/a", "Anything")
		assert.Empty(t, impls)
	})

	t.Run("should handle nil type info in NamedTypes", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"NilType": nil,
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		assert.NotNil(t, index)
	})

	t.Run("should handle nil method in methods slice", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Iface": {
							Name:                 "Iface",
							UnderlyingTypeString: "interface{Do()}",
							Methods: []*inspector_dto.Method{
								nil,
								{Name: "Do", Signature: inspector_dto.FunctionSignature{}},
							},
						},
						"Impl": {
							Name:                 "Impl",
							UnderlyingTypeString: "struct{}",
							Methods: []*inspector_dto.Method{
								nil,
								{Name: "Do", Signature: inspector_dto.FunctionSignature{}},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/a", "Iface")
		require.Len(t, impls, 1)
		assert.Equal(t, "Impl", impls[0].TypeName)
	})

	t.Run("should handle method with empty name", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name: "a",
					Path: "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{
						"Iface": {
							Name:                 "Iface",
							UnderlyingTypeString: "interface{Do()}",
							Methods: []*inspector_dto.Method{
								{Name: "", Signature: inspector_dto.FunctionSignature{}},
								{Name: "Do", Signature: inspector_dto.FunctionSignature{}},
							},
						},
					},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		assert.NotNil(t, index)
	})

	t.Run("should return empty for non-existent interface", func(t *testing.T) {
		t.Parallel()
		typeData := &inspector_dto.TypeData{
			Packages: map[string]*inspector_dto.Package{
				"pkg/a": {
					Name:       "a",
					Path:       "pkg/a",
					NamedTypes: map[string]*inspector_dto.Type{},
				},
			},
		}

		index := NewImplementationIndex(typeData)
		impls := index.FindImplementations("pkg/a", "NonExistent")
		assert.Empty(t, impls)
	})
}
