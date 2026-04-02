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

package inspector_domain_test

import (
	"context"
	goast "go/ast"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func setupInspectorWithCode(t *testing.T, sources map[string]string) *inspector_domain.TypeQuerier {
	t.Helper()

	baseDir, err := os.MkdirTemp("", "querier-test-")
	require.NoError(t, err, "Failed to create temp directory for test")
	t.Cleanup(func() {
		_ = os.RemoveAll(baseDir)
	})

	moduleName := "testproject"
	goModPath := filepath.Join(baseDir, "go.mod")
	goModContent := []byte("module " + moduleName + "\n\ngo 1.23\n")
	err = os.WriteFile(goModPath, goModContent, 0644)
	require.NoError(t, err, "Failed to write dummy go.mod")

	sourceContents := make(map[string][]byte, len(sources))
	for path, content := range sources {
		fullPath := filepath.Join(baseDir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0644)
		require.NoError(t, err)

		sourceContents[fullPath] = []byte(content)
	}

	config := inspector_dto.Config{
		BaseDir:    baseDir,
		ModuleName: moduleName,
	}

	provider := inspector_adapters.NewInMemoryProvider(nil)
	manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))

	err = manager.Build(context.Background(), sourceContents, map[string]string{})
	require.NoError(t, err, "Inspector manager failed to build")

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Failed to get querier from manager")
	require.NotNil(t, inspector, "Inspector should not be nil")

	return inspector
}

func TestFindFieldInfo(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		sources               map[string]string
		name                  string
		baseTypeName          string
		fieldName             string
		importerFile          string
		expectedTypeString    string
		expectedCanonicalPath string
		expectedPropName      string
		expectedFound         bool
		expectedIsRequired    bool
	}{

		{
			name: "Basic: Simple primitive field",
			sources: map[string]string{
				"main.go": `package main; type User struct { Name string }`,
			},
			baseTypeName:          "User",
			fieldName:             "Name",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "string",
			expectedCanonicalPath: "",
		},
		{
			name: "Basic: Simple pointer field",
			sources: map[string]string{
				"main.go": `package main; type User struct { ID *int }`,
			},
			baseTypeName:          "User",
			fieldName:             "ID",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "*int",
			expectedCanonicalPath: "",
		},
		{
			name: "Basic: Non-existent field",
			sources: map[string]string{
				"main.go": `package main; type User struct { Name string }`,
			},
			baseTypeName:  "User",
			fieldName:     "Age",
			importerFile:  "main.go",
			expectedFound: false,
		},

		{
			name: "Embedding: Directly embedded field",
			sources: map[string]string{
				"main.go": `package main; type Base struct { ID int }; type User struct { Base }`,
			},
			baseTypeName:          "User",
			fieldName:             "ID",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "int",
			expectedCanonicalPath: "",
		},
		{
			name: "Embedding: Embedded pointer field",
			sources: map[string]string{
				"main.go": `package main; type Base struct { ID int }; type User struct { *Base }`,
			},
			baseTypeName:          "User",
			fieldName:             "ID",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "int",
			expectedCanonicalPath: "",
		},
		{
			name: "Embedding: Deeply embedded field",
			sources: map[string]string{
				"main.go": `package main; type A struct{ Name string }; type B struct{A}; type C struct{B}`,
			},
			baseTypeName:          "C",
			fieldName:             "Name",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "string",
			expectedCanonicalPath: "",
		},
		{
			name: "Embedding: Field shadowing",
			sources: map[string]string{
				"main.go": `package main; import "fmt"; type Base struct { Name fmt.Stringer }; type User struct { Base; Name string }`,
			},
			baseTypeName:          "User",
			fieldName:             "Name",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "string",
			expectedCanonicalPath: "",
		},

		{
			name: `Tags: validate:"required"`,
			sources: map[string]string{
				"main.go": `package main; type User struct { Name string ` + "`validate:\"required\"`" + `}`,
			},
			baseTypeName:       "User",
			fieldName:          "Name",
			importerFile:       "main.go",
			expectedFound:      true,
			expectedTypeString: "string",
			expectedIsRequired: true,
		},
		{
			name: `Tags: prop:"custom_name"`,
			sources: map[string]string{
				"main.go": `package main; type User struct { Name string ` + "`prop:\"username\"`" + `}`,
			},
			baseTypeName:       "User",
			fieldName:          "Name",
			importerFile:       "main.go",
			expectedFound:      true,
			expectedTypeString: "string",
			expectedPropName:   "username",
		},
		{
			name: `Tags: Combined and complex`,
			sources: map[string]string{
				"main.go": `package main; type User struct { Email string ` + "`prop:\"user_email,omitempty\" validate:\"required,email\"`" + `}`,
			},
			baseTypeName:       "User",
			fieldName:          "Email",
			importerFile:       "main.go",
			expectedFound:      true,
			expectedTypeString: "string",
			expectedPropName:   "user_email",
			expectedIsRequired: true,
		},

		{
			name: "Generics: Field is a generic type with a concrete type argument",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct { Value T }; type User struct { NameBox Box[string] }`,
			},
			baseTypeName:          "User",
			fieldName:             "NameBox",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "main.Box[string]",
			expectedCanonicalPath: "testproject",
		},
		{
			name: "Generics: Find field on a generic struct instance",
			sources: map[string]string{
				"main.go": `package main; type Box[T any] struct { Value T }`,
			},
			baseTypeName:          "Box[string]",
			fieldName:             "Value",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "string",
			expectedCanonicalPath: "",
		},

		{
			name: "Packages: Field type from same package",
			sources: map[string]string{
				"models/user.go": `package models; type User struct { Address Address }; type Address struct {}`,
				"main.go":        `package main; import "testproject/models"; var u models.User`,
			},
			baseTypeName:          "models.User",
			fieldName:             "Address",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "models.Address",
			expectedCanonicalPath: "testproject/models",
		},
		{
			name: "Packages: Field type from different package (aliased import)",
			sources: map[string]string{
				"pkg/types/types.go": `package types; type Time struct {}`,
				"main.go":            `package main; import t "testproject/pkg/types"; type Event struct { Timestamp t.Time }`,
			},
			baseTypeName:          "Event",
			fieldName:             "Timestamp",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "t.Time",
			expectedCanonicalPath: "testproject/pkg/types",
		},
		{
			name: "Packages: Field type from dot import",
			sources: map[string]string{
				"pkg/types/types.go": `package types; type Time struct {}`,
				"main.go":            `package main; import . "testproject/pkg/types"; type Event struct { Timestamp Time }`,
			},
			baseTypeName:          "Event",
			fieldName:             "Timestamp",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "types.Time",
			expectedCanonicalPath: "testproject/pkg/types",
		},
		{
			name: "Packages: Import alias collision (file-scoped context)",
			sources: map[string]string{
				"pkg/uuid_google/uuid.go": `package uuid_google; type UUID struct {}`,
				"pkg/uuid_custom/uuid.go": `package uuid_custom; type UUID struct {}`,
				"api/a.go":                `package api; import uuid "testproject/pkg/uuid_google"; type RequestA struct { ID uuid.UUID }`,
				"api/b.go":                `package api; import uuid "testproject/pkg/uuid_custom"; type RequestB struct { ID uuid.UUID }`,
			},
			baseTypeName:          "api.RequestA",
			fieldName:             "ID",
			importerFile:          "api/a.go",
			expectedFound:         true,
			expectedTypeString:    "uuid.UUID",
			expectedCanonicalPath: "testproject/pkg/uuid_google",
		},
		{
			name: "Aliases: Local type alias to primitive",
			sources: map[string]string{
				"main.go": `package main; type UserID = string; type User struct { ID UserID }`,
			},
			baseTypeName:          "User",
			fieldName:             "ID",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "main.UserID",
			expectedCanonicalPath: "",
		},
		{
			name: "Aliases: Local type alias to imported type",
			sources: map[string]string{
				"models/user.go": `package models; type User struct{ Name string }`,
				"main.go":        `package main; import "testproject/models"; type LocalUser = models.User; type Data struct { Owner LocalUser }`,
			},
			baseTypeName:          "Data",
			fieldName:             "Owner",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "main.LocalUser",
			expectedCanonicalPath: "testproject/models",
		},

		{
			name: "Composites: Pointer to slice of imported type",
			sources: map[string]string{
				"models/user.go": `package models; type User struct{}`,
				"main.go":        `package main; import "testproject/models"; type Response struct { Users *[]models.User }`,
			},
			baseTypeName:          "Response",
			fieldName:             "Users",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "*[]models.User",
			expectedCanonicalPath: "testproject/models",
		},
		{
			name: "Composites: Map with imported key and value types",
			sources: map[string]string{
				"models/user.go": `package models; type User struct{}`,
				"perms/perms.go": `package perms; type Permission struct{}`,
				"main.go":        `package main; import "testproject/models"; import "testproject/perms"; type ACL struct { Rules map[*models.User]perms.Permission }`,
			},
			baseTypeName:          "ACL",
			fieldName:             "Rules",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "map[*models.User]perms.Permission",
			expectedCanonicalPath: "testproject/perms",
		},
		{
			name: "Composites: Slice of pointers to generics with imported type argument",
			sources: map[string]string{
				"models/user.go": `package models; type User struct{}`,
				"generic/box.go": `package generic; type Box[T any] struct{ Value T }`,
				"main.go":        `package main; import "testproject/models"; import "testproject/generic"; type Response struct { UserBoxes []*generic.Box[models.User] }`,
			},
			baseTypeName:          "Response",
			fieldName:             "UserBoxes",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "[]*generic.Box[models.User]",
			expectedCanonicalPath: "testproject/generic",
		},

		{
			name: "Edge Case: nil base type",
			sources: map[string]string{
				"main.go": `package main;`,
			},
			baseTypeName:  "",
			fieldName:     "any",
			importerFile:  "main.go",
			expectedFound: false,
		},
		{
			name: "Edge Case: Unresolvable base type",
			sources: map[string]string{
				"main.go": `package main;`,
			},
			baseTypeName:  "nonexistent.Type",
			fieldName:     "any",
			importerFile:  "main.go",
			expectedFound: false,
		},
		{
			name: "Edge Case: Circular embedding",
			sources: map[string]string{
				"main.go": `package main; type A struct {*B}; type B struct {*A}`,
			},
			baseTypeName:  "A",
			fieldName:     "NonExistentField",
			importerFile:  "main.go",
			expectedFound: false,
		},

		{
			name: "Adv Generics: Embedding an instantiated generic type",
			sources: map[string]string{
				"generic/box.go": `package generic; type Box[T any] struct{ Value T }`,
				"main.go":        `package main; import "testproject/generic"; type StringBox generic.Box[string]; type Container struct{ StringBox }`,
			},
			baseTypeName:          "Container",
			fieldName:             "Value",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "string",
			expectedCanonicalPath: "",
		},
		{
			name: "Adv Generics: Embedding a generic type with a generic parameter",
			sources: map[string]string{
				"generic/box.go": `package generic; type Box[T any] struct{ Value T }`,
				"main.go":        `package main; import "testproject/generic"; type Wrapper[K any] struct{ generic.Box[K] }`,
			},
			baseTypeName:          "Wrapper[int]",
			fieldName:             "Value",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "int",
			expectedCanonicalPath: "",
		},
		{
			name: "Adv Generics: Field is a nested generic type",
			sources: map[string]string{
				"generic/types.go": `package generic; type Box[T any] struct{}; type Wrapper[K any] struct{}`,
				"main.go":          `package main; import "testproject/generic"; type Container struct { Nested generic.Wrapper[generic.Box[string]] }`,
			},
			baseTypeName:          "Container",
			fieldName:             "Nested",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "generic.Wrapper[generic.Box[string]]",
			expectedCanonicalPath: "testproject/generic",
		},

		{
			name: "Adv Aliases: Chained local type aliases",
			sources: map[string]string{
				"main.go": `package main; type UserID = int64; type AccountID = UserID; type Session struct{ ID AccountID }`,
			},
			baseTypeName:          "Session",
			fieldName:             "ID",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "main.AccountID",
			expectedCanonicalPath: "",
		},
		{
			name: "Adv Types: Field is an interface type",
			sources: map[string]string{
				"main.go": `package main; import "io"; type Task struct{ Source io.Reader }`,
			},
			baseTypeName:          "Task",
			fieldName:             "Source",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "io.Reader",
			expectedCanonicalPath: "io",
		},
		{
			name: "Adv Types: Field is a function type",
			sources: map[string]string{
				"main.go": `package main; import "net/http"; type Server struct{ Handler func(w http.ResponseWriter, r *http.Request) }`,
			},
			baseTypeName:          "Server",
			fieldName:             "Handler",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "func(w http.ResponseWriter, r *http.Request)",
			expectedCanonicalPath: "net/http",
		},

		{
			name: "Ambiguity: Ambiguous selector from multiple embeddings",
			sources: map[string]string{
				"main.go": `package main; type A struct{ ID int }; type B struct{ ID int }; type C struct{ A; B }`,
			},
			baseTypeName:  "C",
			fieldName:     "ID",
			importerFile:  "main.go",
			expectedFound: false,
		},
		{
			name: "Ambiguity: Field from a non-promoted embedded type",
			sources: map[string]string{
				"main.go": `package main; type Base struct{ ID int }; type User struct{ MyBase Base }`,
			},
			baseTypeName:  "User",
			fieldName:     "ID",
			importerFile:  "main.go",
			expectedFound: false,
		},

		{
			name: "Adv Interactions: Embedded type from a dot-imported package",
			sources: map[string]string{
				"models/base.go": `package models; type Base struct{ ID int }`,
				"main.go":        `package main; import . "testproject/models"; type User struct{ Base }`,
			},
			baseTypeName:          "User",
			fieldName:             "ID",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "int",
			expectedCanonicalPath: "",
		},
		{
			name: "Adv Interactions: Requalification of dot-imported type inside a composite",
			sources: map[string]string{
				"models/user.go": `package models; type User struct{}`,
				"main.go":        `package main; import . "testproject/models"; type Response struct{ Users []*User }`,
			},
			baseTypeName:          "Response",
			fieldName:             "Users",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "[]*models.User",
			expectedCanonicalPath: "testproject/models",
		},
		{
			name: "Deep Resolution: Multi-package field lookup through struct embedding",
			sources: map[string]string{
				"main.go": `
            package main
            import "testproject/layer1"
            type Response struct { L1Data layer1.Layer1Response }
        `,
				"layer1/types.go": `
            package layer1
            import "testproject/layer2"
            type Layer1Response struct { L2Data layer2.Layer2Response }
        `,
				"layer2/types.go": `
            package layer2
            import "testproject/layer3"
            type Layer2Response struct { L3Data layer3.Layer3Response }
        `,
				"layer3/types.go": `
            package layer3
            import "testproject/models"
            type Layer3Response struct { FinalItem models.Data }
        `,
				"models/data.go": `
            package models
            type Data struct { Name string }
        `,
			},

			baseTypeName:          "Response",
			fieldName:             "L1Data.L2Data.L3Data.FinalItem.Name",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "string",
			expectedCanonicalPath: "",
		},
		{
			name: "Deep Resolution: Lookup through a type alias in another package",
			sources: map[string]string{
				"main.go": `
            package main
            import "testproject/layer2"
            type Response struct { L2Alias layer2.L3Alias }
        `,
				"layer2/types.go": `
            package layer2
            import "testproject/layer3"
            // The alias is defined here, in a different file and package
            type L3Alias = layer3.Layer3Response
        `,
				"layer3/types.go": `
            package layer3
            import "testproject/models"
            type Layer3Response struct { FinalItem models.Data }
        `,
				"models/data.go": `
            package models
            type Data struct { Name string }
        `,
			},
			baseTypeName:       "Response",
			fieldName:          "L2Alias.FinalItem.Name",
			importerFile:       "main.go",
			expectedFound:      true,
			expectedTypeString: "string",
		},

		{
			name: "Generic Alias: Field lookup on generic type alias with concrete type argument",
			sources: map[string]string{
				"main.go": `
package main
import "testproject/facade"
import "testproject/models"
type Response struct {
	Results []facade.SearchResult[models.Doc]
}`,
				"facade/types.go": `
package facade
import "testproject/runtime"
type SearchResult[T any] = runtime.SearchResult[T]`,
				"runtime/types.go": `
package runtime
type SearchResult[T any] struct {
	Item  T
	Score float64
}`,
				"models/doc.go": `
package models
type Doc struct {
	Title string
	URL   string
}`,
			},
			baseTypeName:          "facade.SearchResult[models.Doc]",
			fieldName:             "Item",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "models.Doc",
			expectedCanonicalPath: "testproject/models",
		},
		{
			name: "Generic Alias: Nested field lookup through generic type alias",
			sources: map[string]string{
				"main.go": `
package main
import "testproject/facade"
import "testproject/models"
type Response struct {
	Results []facade.SearchResult[models.Doc]
}`,
				"facade/types.go": `
package facade
import "testproject/runtime"
type SearchResult[T any] = runtime.SearchResult[T]`,
				"runtime/types.go": `
package runtime
type SearchResult[T any] struct {
	Item  T
	Score float64
}`,
				"models/doc.go": `
package models
type Doc struct {
	Title string
	URL   string
}`,
			},
			baseTypeName:          "facade.SearchResult[models.Doc]",
			fieldName:             "Item.Title",
			importerFile:          "main.go",
			expectedFound:         true,
			expectedTypeString:    "string",
			expectedCanonicalPath: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			inspector := setupInspectorWithCode(t, tc.sources)

			importerPackagePath := "testproject"
			if directory := filepath.Dir(tc.importerFile); directory != "." {
				importerPackagePath = "testproject/" + directory
			}
			importerFilePath := filepath.Join(inspector.Config.BaseDir, tc.importerFile)

			var baseTypeAST goast.Expr
			if tc.baseTypeName != "" {
				baseTypeAST = goastutil.TypeStringToAST(tc.baseTypeName)
			}

			fieldInfo := inspector.FindFieldInfo(context.Background(), baseTypeAST, tc.fieldName, importerPackagePath, importerFilePath)

			if !tc.expectedFound {
				assert.Nil(t, fieldInfo, "Expected field to not be found, but it was")
				return
			}

			require.NotNil(t, fieldInfo, "Expected field to be found, but it was not")

			actualTypeString := goastutil.ASTToTypeString(fieldInfo.Type)
			assert.Equal(t, tc.expectedTypeString, actualTypeString, "FieldInfo.Type does not match expected")

			if tc.expectedCanonicalPath != "" {
				assert.Equal(t, tc.expectedCanonicalPath, fieldInfo.CanonicalPackagePath, "FieldInfo.CanonicalPackagePath does not match")
			}

			if tc.expectedPropName != "" {
				assert.Equal(t, tc.expectedPropName, fieldInfo.PropName, "FieldInfo.PropName does not match")
			} else if tc.expectedFound {
				assert.Equal(t, tc.fieldName, fieldInfo.PropName, "FieldInfo.PropName should default to field name")
			}

			assert.Equal(t, tc.expectedIsRequired, fieldInfo.IsRequired, "FieldInfo.IsRequired does not match")
		})
	}
}
