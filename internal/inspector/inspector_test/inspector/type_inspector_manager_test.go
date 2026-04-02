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

package inspector_test

import (
	"context"
	goast "go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func findMainFilePath(t *testing.T, projectDir string, sources map[string][]byte) string {
	t.Helper()
	absRoot, err := filepath.Abs(projectDir)
	require.NoError(t, err)
	for path := range sources {
		if strings.HasSuffix(path, string(os.PathSeparator)+"main.go") {
			if filepath.Dir(path) == absRoot {
				return path
			}
		}
	}

	for path := range sources {
		if strings.HasSuffix(path, string(os.PathSeparator)+"main.go") || strings.HasSuffix(path, "/main.go") || strings.HasSuffix(path, "\\main.go") {
			return path
		}
	}
	require.FailNow(t, "Could not locate main.go in project")
	return ""
}

func findFileContaining(t *testing.T, sources map[string][]byte, substr string) string {
	t.Helper()
	for path, content := range sources {
		if strings.Contains(string(content), substr) {
			return path
		}
	}
	require.FailNowf(t, "Could not find file containing substring", "substr=%s", substr)
	return ""
}

func findAnyFileInPackage(t *testing.T, sources map[string][]byte, packageName string) string {
	t.Helper()
	needle := "package " + packageName
	for path, content := range sources {
		if strings.HasPrefix(strings.TrimSpace(string(content)), needle) || strings.Contains(string(content), "\n"+needle) {
			return path
		}
	}
	require.FailNowf(t, "Could not find a file declaring package", "pkg=%s", packageName)
	return ""
}

func setupManagerForProject(t *testing.T, projectDir, moduleName string) *inspector_domain.TypeBuilder {
	absPath, err := filepath.Abs(projectDir)
	require.NoError(t, err, "Failed to get absolute path for test project: %s", projectDir)

	config := inspector_dto.Config{
		BaseDir:    absPath,
		ModuleName: moduleName,
	}

	provider := inspector_adapters.NewInMemoryProvider(nil)

	return inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(provider))
}

func getSourceContentsForTest(t *testing.T, root string) map[string][]byte {
	sourceContents := make(map[string][]byte)
	absRoot, err := filepath.Abs(root)
	require.NoError(t, err)

	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			content, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			sourceContents[path] = content
		}
		return nil
	})
	require.NoError(t, err)
	return sourceContents
}

func TestTypeBuilder_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping type builder integration tests in short mode")
	}

	t.Run("Happy Path Project", func(t *testing.T) {
		projectDir := "./testdata/01_happy_path"
		moduleName := "testproject_happy"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)

		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)

		inspector, ok := manager.GetQuerier()
		require.True(t, ok, "Inspector should have been created successfully")
		require.NotNil(t, inspector)

		responseType := goast.NewIdent("Response")

		importerPackagePath := moduleName

		mainPackageName := "main"

		importerFilePath := findMainFilePath(t, projectDir, sourceContents)
		propsFilePath := findFileContaining(t, sourceContents, "type Props struct")

		t.Run("should find top-level Props type from local script", func(t *testing.T) {

			propsType := inspector.FindPropsType(propsFilePath)
			require.NotNil(t, propsType)
			assert.Equal(t, "Props", propsType.(*goast.Ident).Name)
		})

		t.Run("should find function return type from local script", func(t *testing.T) {

			getUserFuncReturnType := inspector.FindFuncReturnType(mainPackageName, "GetUser", importerPackagePath, importerFilePath)
			require.NotNil(t, getUserFuncReturnType, "GetUser function should be found")

			star, ok := getUserFuncReturnType.(*goast.StarExpr)
			require.True(t, ok)
			selector, ok := star.X.(*goast.SelectorExpr)
			require.True(t, ok)
			assert.Equal(t, "models", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "User", selector.Sel.Name)
		})

		t.Run("should find field pointing to internal struct", func(t *testing.T) {

			fieldType := inspector.FindFieldType(responseType, "User", importerPackagePath, importerFilePath)
			require.NotNil(t, fieldType)

			star, ok := fieldType.(*goast.StarExpr)
			require.True(t, ok)
			selector, ok := star.X.(*goast.SelectorExpr)
			require.True(t, ok)
			assert.Equal(t, "models", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "User", selector.Sel.Name)
		})

		t.Run("should find field on a nested struct from another package", func(t *testing.T) {
			userType := &goast.SelectorExpr{X: goast.NewIdent("models"), Sel: goast.NewIdent("User")}

			ageFieldType := inspector.FindFieldType(userType, "Age", importerPackagePath, importerFilePath)
			require.NotNil(t, ageFieldType)
			assert.Equal(t, "int", ageFieldType.(*goast.Ident).Name)
		})

		t.Run("should find method on a pointer receiver", func(t *testing.T) {
			userPtrType := &goast.StarExpr{X: &goast.SelectorExpr{X: goast.NewIdent("models"), Sel: goast.NewIdent("User")}}

			returnType := inspector.FindMethodReturnType(userPtrType, "GetProfileURL", importerPackagePath, importerFilePath)
			require.NotNil(t, returnType)
			assert.Equal(t, "string", returnType.(*goast.Ident).Name)
		})
	})

	t.Run("Project with External Dependencies", func(t *testing.T) {
		projectDir := "./testdata/05_external_deps"
		moduleName := "testproject_external_deps"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)

		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)

		inspector, ok := manager.GetQuerier()
		require.True(t, ok)
		require.NotNil(t, inspector)

		responseType := goast.NewIdent("Response")
		importerPackagePath := moduleName

		t.Run("should resolve field with external struct type", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "RequestID", importerPackagePath, findMainFilePath(t, projectDir, sourceContents))
			require.NotNil(t, fieldType)

			typeString := goastutil.ASTToTypeString(fieldType)
			assert.Equal(t, "uuid.UUID", typeString, "Should return the named type, not its underlying array")
		})

		t.Run("should resolve field with external interface type", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "CurrentSpan", importerPackagePath, findMainFilePath(t, projectDir, sourceContents))
			require.NotNil(t, fieldType)
			typeString := goastutil.ASTToTypeString(fieldType)
			assert.Equal(t, "trace.Span", typeString)
		})

		t.Run("should resolve map with external type as value", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "SpanContexts", importerPackagePath, findMainFilePath(t, projectDir, sourceContents))
			require.NotNil(t, fieldType)
			mapType, ok := fieldType.(*goast.MapType)
			require.True(t, ok)
			mapKey, ok := mapType.Key.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "string", mapKey.Name)

			valueSelector, ok := mapType.Value.(*goast.SelectorExpr)
			require.True(t, ok)
			valueSelectorX, ok := valueSelector.X.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "trace", valueSelectorX.Name)
			assert.Equal(t, "SpanContext", valueSelector.Sel.Name)
		})
	})

	t.Run("Complex Project with Aliases and Name Collisions", func(t *testing.T) {
		projectDir := "./testdata/06_complex"
		moduleName := "testproject_complex"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)

		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)

		inspector, ok := manager.GetQuerier()
		require.True(t, ok)
		require.NotNil(t, inspector)

		responseType := goast.NewIdent("Response")
		importerPackagePath := moduleName

		t.Run("should resolve name-colliding type to correct imported package", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "CurrentUser", importerPackagePath, findMainFilePath(t, projectDir, sourceContents))
			require.NotNil(t, fieldType)
			selector, ok := fieldType.(*goast.SelectorExpr)
			require.True(t, ok)
			selectorX, ok := selector.X.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "api", selectorX.Name, "Should resolve to 'api.User', not 'db.User'")
			assert.Equal(t, "User", selector.Sel.Name)
		})

		t.Run("should respect aliased imports", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "Span", importerPackagePath, findMainFilePath(t, projectDir, sourceContents))
			require.NotNil(t, fieldType)
			selector, ok := fieldType.(*goast.SelectorExpr)
			require.True(t, ok)
			selectorX, ok := selector.X.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "oteltrace", selectorX.Name, "Should use the 'oteltrace' alias")
			assert.Equal(t, "Span", selector.Sel.Name)
		})

		t.Run("should resolve fields from embedded structs", func(t *testing.T) {
			loginEventType := &goast.SelectorExpr{X: goast.NewIdent("services"), Sel: goast.NewIdent("LoginEvent")}

			servicesPackagePath := moduleName + "/services"
			fieldType := inspector.FindFieldType(loginEventType, "Timestamp", servicesPackagePath, findAnyFileInPackage(t, sourceContents, "services"))
			require.NotNil(t, fieldType)
			selector, ok := fieldType.(*goast.SelectorExpr)
			require.True(t, ok)
			selectorX, ok := selector.X.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "time", selectorX.Name)
			assert.Equal(t, "Time", selector.Sel.Name)
		})
	})

	t.Run("Error Handling: Malformed Project (Syntax Error)", func(t *testing.T) {
		manager := setupManagerForProject(t, "./testdata/02_malformed", "testproject_malformed")
		sourceContents := getSourceContentsForTest(t, "./testdata/02_malformed")
		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse source contents into ASTs")
	})

	t.Run("Error Handling: Semantic Error Project (Type Check)", func(t *testing.T) {
		manager := setupManagerForProject(t, "./testdata/04_semantic_error", "testproject_semantic_error")
		sourceContents := getSourceContentsForTest(t, "./testdata/04_semantic_error")
		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "undefined: models")
	})

	t.Run("Edge Case: Project with no go.mod", func(t *testing.T) {
		projectDir := "./testdata/03_no_gomod"
		manager := setupManagerForProject(t, projectDir, "")
		sourceContents := getSourceContentsForTest(t, projectDir)
		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)
		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		absPath, err := filepath.Abs(projectDir)
		require.NoError(t, err)

		importerPackagePath := absPath

		responseType := goast.NewIdent("Response")
		greetMethodReturnType := inspector.FindMethodReturnType(responseType, "Greet", importerPackagePath, findMainFilePath(t, projectDir, sourceContents))
		require.NotNil(t, greetMethodReturnType)
		assert.Equal(t, "string", greetMethodReturnType.(*goast.Ident).Name)
	})

	t.Run("Advanced Types Project", func(t *testing.T) {
		projectDir := "./testdata/07_advanced_types"
		moduleName := "testproject_advanced_types"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)
		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)
		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		responseType := goast.NewIdent("Response")
		importerPackagePath := moduleName
		mainFilePath := findMainFilePath(t, projectDir, sourceContents)

		t.Run("should return declared alias and then resolve to underlying primitive", func(t *testing.T) {

			declaredFieldType := inspector.FindFieldType(responseType, "PrimaryID", importerPackagePath, mainFilePath)
			require.NotNil(t, declaredFieldType)
			assert.Equal(t, "main.UserID", goastutil.ASTToTypeString(declaredFieldType), "FindFieldType should return the declared alias")

			underlyingType := inspector.ResolveToUnderlyingAST(declaredFieldType, mainFilePath)
			require.NotNil(t, underlyingType)
			assert.Equal(t, "string", goastutil.ASTToTypeString(underlyingType), "ResolveToUnderlyingAST should resolve the alias to its primitive type")
		})

		t.Run("should resolve complex map with generic types", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "Products", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldType)

			mapType, ok := fieldType.(*goast.MapType)
			require.True(t, ok)
			keyIdent, ok := mapType.Key.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "string", keyIdent.Name)
			valueSelector, ok := mapType.Value.(*goast.IndexExpr)
			require.True(t, ok)
			expectedValueString := "models.Product[models.Metadata]"
			actualValueString := goastutil.ASTToTypeString(valueSelector)
			assert.Equal(t, expectedValueString, actualValueString)
		})

		t.Run("should find method on an instantiated generic type receiver", func(t *testing.T) {
			genericType := &goast.StarExpr{
				X: &goast.IndexExpr{
					X:     &goast.SelectorExpr{X: goast.NewIdent("models"), Sel: goast.NewIdent("Product")},
					Index: &goast.SelectorExpr{X: goast.NewIdent("models"), Sel: goast.NewIdent("Metadata")},
				},
			}
			returnType := inspector.FindMethodReturnType(genericType, "GetMeta", importerPackagePath, mainFilePath)
			require.NotNil(t, returnType)
			assert.Equal(t, "models.Metadata", goastutil.ASTToTypeString(returnType))
		})
	})

	t.Run("Complex Embedding and Shadowing Project", func(t *testing.T) {
		projectDir := "./testdata/08_embedding"
		moduleName := "testproject_embedding"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)
		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)
		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		fullPostType := &goast.SelectorExpr{X: goast.NewIdent("models"), Sel: goast.NewIdent("FullPost")}
		importerPackagePath := moduleName
		mainFilePath := findMainFilePath(t, projectDir, sourceContents)

		t.Run("should resolve shadowed embedded field to the outer type", func(t *testing.T) {
			fieldType := inspector.FindFieldType(fullPostType, "ID", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldType)
			assert.Equal(t, "string", goastutil.ASTToTypeString(fieldType))
		})

		t.Run("should resolve non-shadowed embedded field", func(t *testing.T) {
			fieldType := inspector.FindFieldType(fullPostType, "CreatedAt", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldType)
			selector, ok := fieldType.(*goast.SelectorExpr)
			require.True(t, ok)
			selectorX, ok := selector.X.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "time", selectorX.Name)
			assert.Equal(t, "Time", selector.Sel.Name)
		})

		t.Run("should resolve promoted method from embedded struct", func(t *testing.T) {
			returnType := inspector.FindMethodReturnType(fullPostType, "GetID", importerPackagePath, mainFilePath)
			require.NotNil(t, returnType)
			assert.Equal(t, "uint64", goastutil.ASTToTypeString(returnType))
		})
	})

	t.Run("Interfaces and Function Types Project", func(t *testing.T) {
		projectDir := "./testdata/09_interfaces"
		moduleName := "testproject_interfaces"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)
		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)
		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		responseType := goast.NewIdent("Response")
		importerPackagePath := moduleName
		servicesPackagePath := moduleName + "/services"
		mainFilePath := findMainFilePath(t, projectDir, sourceContents)
		servicesFilePath := findAnyFileInPackage(t, sourceContents, "services")

		t.Run("should correctly identify a field of interface type", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "Logger", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldType)
			selector, ok := fieldType.(*goast.SelectorExpr)
			require.True(t, ok)
			selectorX, ok := selector.X.(*goast.Ident)
			require.True(t, ok)
			assert.Equal(t, "services", selectorX.Name)
			assert.Equal(t, "Logger", selector.Sel.Name)
		})

		t.Run("should find method on an interface type", func(t *testing.T) {
			loggerInterfaceType := &goast.SelectorExpr{X: goast.NewIdent("services"), Sel: goast.NewIdent("Logger")}
			returnType := inspector.FindMethodReturnType(loggerInterfaceType, "Log", servicesPackagePath, servicesFilePath)
			require.NotNil(t, returnType)
			assert.Equal(t, "error", returnType.(*goast.Ident).Name)
		})

		t.Run("should correctly identify a field of function type", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "Handler", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldType)

			funcType, ok := fieldType.(*goast.FuncType)
			require.True(t, ok)

			require.Len(t, funcType.Params.List, 1)
			assert.Equal(t, "string", funcType.Params.List[0].Type.(*goast.Ident).Name)

			require.Len(t, funcType.Results.List, 1)
			assert.Equal(t, "error", funcType.Results.List[0].Type.(*goast.Ident).Name)
		})
	})

	t.Run("Build Tags Project", func(t *testing.T) {
		projectDir := "./testdata/10_buildtags"
		sourceContents := getSourceContentsForTest(t, projectDir)
		absPath, err := filepath.Abs(projectDir)
		require.NoError(t, err)

		config := inspector_dto.Config{
			BaseDir:    absPath,
			ModuleName: "testproject_buildtags",
			GOOS:       "js",
			GOARCH:     "wasm",
		}
		manager := inspector_domain.NewTypeBuilder(config, inspector_domain.WithProvider(inspector_adapters.NewInMemoryProvider(nil)))

		err = manager.Build(context.Background(), sourceContents, map[string]string{})
		require.Error(t, err, "Build should fail when no build tags match")
		assert.Contains(t, err.Error(), "undefined: MyConfig")
	})

	t.Run("Edge Cases Project (Dot Imports)", func(t *testing.T) {
		projectDir := "./testdata/11_edgecases"
		moduleName := "testproject_edgecases"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)
		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)
		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		responseType := goast.NewIdent("Response")
		importerPackagePath := moduleName
		mainFilePath := findMainFilePath(t, projectDir, sourceContents)

		t.Run("should resolve type from a dot-imported package", func(t *testing.T) {
			fieldType := inspector.FindFieldType(responseType, "Helper", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldType)

			selector, ok := fieldType.(*goast.SelectorExpr)
			require.True(t, ok)
			assert.Equal(t, "utils", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "Helper", selector.Sel.Name)
		})
	})

	t.Run("Type Definition vs. Type Alias Method Resolution", func(t *testing.T) {
		projectDir := "./testdata/15_aliased_type_method"
		moduleName := "testproject_aliased_method"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)

		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)

		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		responseType := goast.NewIdent("Response")
		importerPackagePath := moduleName
		mainFilePath := findMainFilePath(t, projectDir, sourceContents)

		t.Run("should NOT find method on a new type definition", func(t *testing.T) {
			fieldInfo := inspector.FindFieldInfo(context.Background(), responseType, "DefinedUUID", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldInfo)
			sig := inspector.FindMethodSignature(fieldInfo.Type, "String", importerPackagePath, mainFilePath)
			assert.Nil(t, sig, "A type definition (type T U) should NOT inherit methods from the underlying type U")
		})

		t.Run("should find method on a type alias", func(t *testing.T) {
			fieldInfo := inspector.FindFieldInfo(context.Background(), responseType, "AliasedUUID", importerPackagePath, mainFilePath)
			require.NotNil(t, fieldInfo)

			sig := inspector.FindMethodSignature(fieldInfo.Type, "String", importerPackagePath, mainFilePath)
			require.NotNil(t, sig, "A type alias (type T = U) should share the exact same method set as the original type U")

			require.Len(t, sig.Results, 1)
			assert.Equal(t, "string", sig.Results[0])
			assert.Empty(t, sig.Params)
		})
	})

	t.Run("Alias Context Bug Project", func(t *testing.T) {
		projectDir := "./testdata/17_alias_context_bug"
		moduleName := "testproject_alias_bug"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)

		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)

		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		mainPackagePath := moduleName
		servicesPackagePath := moduleName + "/services"
		layer1PackagePath := moduleName + "/layer1"
		layer2PackagePath := moduleName + "/layer2"

		mainFilePath := findMainFilePath(t, projectDir, sourceContents)
		servicesFilePath := findAnyFileInPackage(t, sourceContents, "services")
		layer1FilePath := findAnyFileInPackage(t, sourceContents, "layer1")
		layer2FilePath := findAnyFileInPackage(t, sourceContents, "layer2")

		t.Run("should resolve package function return type using its own package context", func(t *testing.T) {

			returnType := inspector.FindFuncReturnType("services", "GetUserFunc", mainPackagePath, mainFilePath)
			require.NotNil(t, returnType)

			star, ok := returnType.(*goast.StarExpr)
			require.True(t, ok)
			selector, ok := star.X.(*goast.SelectorExpr)
			require.True(t, ok)

			assert.Equal(t, "models", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "User", selector.Sel.Name)
		})

		t.Run("should resolve method return type using its receiver's package context", func(t *testing.T) {
			userServiceType := &goast.SelectorExpr{X: goast.NewIdent("services"), Sel: goast.NewIdent("UserService")}

			returnType := inspector.FindMethodReturnType(userServiceType, "GetUser", servicesPackagePath, servicesFilePath)
			require.NotNil(t, returnType)

			star, ok := returnType.(*goast.StarExpr)
			require.True(t, ok)
			selector, ok := star.X.(*goast.SelectorExpr)
			require.True(t, ok)

			assert.Equal(t, "models", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "User", selector.Sel.Name)
		})

		t.Run("should resolve deeply nested field type using correct intermediate contexts", func(t *testing.T) {
			responseType := goast.NewIdent("Response")

			l1InfoField := inspector.FindFieldInfo(context.Background(), responseType, "NestedInfo", mainPackagePath, mainFilePath)
			require.NotNil(t, l1InfoField)
			assert.Equal(t, "layer1.Layer1Response", goastutil.ASTToTypeString(l1InfoField.Type))

			l2DataField := inspector.FindFieldInfo(context.Background(), l1InfoField.Type, "Data", layer1PackagePath, layer1FilePath)
			require.NotNil(t, l2DataField)
			assert.Equal(t, "layer2.Layer2Data", goastutil.ASTToTypeString(l2DataField.Type))

			userField := inspector.FindFieldInfo(context.Background(), l2DataField.Type, "User", layer2PackagePath, layer2FilePath)
			require.NotNil(t, userField)

			selector, ok := userField.Type.(*goast.SelectorExpr)
			require.True(t, ok)
			assert.Equal(t, "models", selector.X.(*goast.Ident).Name)
			assert.Equal(t, "User", selector.Sel.Name)
		})

		t.Run("should resolve generic type argument using its own package context", func(t *testing.T) {

			returnType := inspector.FindFuncReturnType("generics", "GetUserModelBox", mainPackagePath, mainFilePath)
			require.NotNil(t, returnType)

			indexExpr, ok := returnType.(*goast.IndexExpr)
			require.True(t, ok)

			baseTypeSelector, ok := indexExpr.X.(*goast.SelectorExpr)
			require.True(t, ok)
			assert.Equal(t, "generics", baseTypeSelector.X.(*goast.Ident).Name)
			assert.Equal(t, "Box", baseTypeSelector.Sel.Name)

			typeArgSelector, ok := indexExpr.Index.(*goast.SelectorExpr)
			require.True(t, ok)

			assert.Equal(t, "models", typeArgSelector.X.(*goast.Ident).Name)
			assert.Equal(t, "User", typeArgSelector.Sel.Name)
		})
	})

	t.Run("Deep Method Resolution with Import Alias Bug", func(t *testing.T) {
		projectDir := "./testdata/18_deep_method_resolution_with_alias_bug"
		moduleName := "testcase_18_deep_method_resolution_with_alias_bug"
		manager := setupManagerForProject(t, projectDir, moduleName)
		sourceContents := getSourceContentsForTest(t, projectDir)

		err := manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err)

		inspector, ok := manager.GetQuerier()
		require.True(t, ok)

		mainPackagePath := moduleName
		servicesPackagePath := moduleName + "/services"
		dtosPackagePath := moduleName + "/dtos"
		mathsPackagePath := "piko.sh/piko/wdk/maths"

		mainFilePath := findMainFilePath(t, projectDir, sourceContents)
		servicesFilePath := findAnyFileInPackage(t, sourceContents, "services")
		dtosFilePath := findAnyFileInPackage(t, sourceContents, "dtos")

		baseType := goast.NewIdent("Response")

		t.Log("Step 1: Resolving Response.ServiceResponse")
		serviceResponseField := inspector.FindFieldInfo(context.Background(), baseType, "ServiceResponse", mainPackagePath, mainFilePath)
		require.NotNil(t, serviceResponseField, "Failed to find 'ServiceResponse' field")
		assert.Equal(t, "services.TransactionServiceResponse", goastutil.ASTToTypeString(serviceResponseField.Type))

		t.Log("Step 2: Resolving TransactionServiceResponse.Transaction")

		transactionField := inspector.FindFieldInfo(context.Background(), serviceResponseField.Type, "Transaction", servicesPackagePath, servicesFilePath)
		require.NotNil(t, transactionField, "Failed to find 'Transaction' field")

		assert.Equal(t, "dto_alias.TransactionDto", goastutil.ASTToTypeString(transactionField.Type))

		t.Log("Step 3: Resolving TransactionDto.Amount")

		amountField := inspector.FindFieldInfo(context.Background(), transactionField.Type, "Amount", dtosPackagePath, dtosFilePath)
		require.NotNil(t, amountField, "Failed to find 'Amount' field. THIS IS WHERE THE BUG OCCURS.")
		assert.Equal(t, "maths.Money", goastutil.ASTToTypeString(amountField.Type))

		t.Log("Step 4: Resolving Amount.MustNumber()")

		sig := inspector.FindMethodSignature(amountField.Type, "MustNumber", mathsPackagePath, dtosFilePath)
		require.NotNil(t, sig, "Failed to find 'MustNumber' method on the final resolved type")
		require.Len(t, sig.Results, 1, "MustNumber should have one return value")
		assert.Equal(t, "string", sig.Results[0], "The return type of MustNumber should be string")
	})
}

func TestTypeBuilder_InterPackageDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping type builder inter-package discovery tests in short mode")
	}

	t.Run("should consistently discover all local packages via discovery", func(t *testing.T) {
		projectDir := "./testdata/12_call_package_func"
		moduleName := "testcase_12_call_package_func"

		sourceContents := getSourceContentsForTest(t, projectDir)
		absProjectDir, err := filepath.Abs(projectDir)
		require.NoError(t, err)

		mainPkPath := filepath.Join(absProjectDir, "main.pk")
		pkContent, err := os.ReadFile(mainPkPath)
		require.NoError(t, err)

		contentString := string(pkContent)
		scriptTagStart := `<script type="application/x-go">`
		scriptStartIndex := strings.Index(contentString, scriptTagStart)
		require.NotEqual(t, -1, scriptStartIndex, ".pk file must contain a Go script tag")
		scriptEndIndex := strings.Index(contentString, `</script>`)
		require.NotEqual(t, -1, scriptEndIndex, ".pk file must contain a closing script tag")

		goCode := contentString[scriptStartIndex+len(scriptTagStart) : scriptEndIndex]

		virtualGoPath := filepath.Join(absProjectDir, "main_from_pk.go")
		sourceContents[virtualGoPath] = []byte(goCode)

		manager := setupManagerForProject(t, projectDir, moduleName)

		err = manager.Build(context.Background(), sourceContents, map[string]string{})
		require.NoError(t, err, "The build process should complete without errors")

		typeData, err := manager.GetTypeData()
		require.NoError(t, err)
		require.NotNil(t, typeData)

		helpersPackagePath := "testcase_12_call_package_func/helpers"
		mainPackagePath := moduleName

		mainPackageData, mainPackageExists := typeData.Packages[mainPackagePath]
		require.True(t, mainPackageExists, "Main package data must exist")

		require.NotNil(t, mainPackageData.FileImports, "The FileImports map should have been serialised.")
		mainFileImports, mainFileImportsExist := mainPackageData.FileImports[virtualGoPath]
		require.True(t, mainFileImportsExist, "An import map for the virtual go file ('%s') must exist.", virtualGoPath)

		foundPath, aliasExists := mainFileImports["helpers"]
		assert.True(t, aliasExists, "The 'helpers' package alias must be present in the main file's import map")
		assert.Equal(t, helpersPackagePath, foundPath)

		pkgData, pkgExists := typeData.Packages[helpersPackagePath]
		require.True(t, pkgExists, "The data for package '%s' must be present in the packages map", helpersPackagePath)

		_, funcExists := pkgData.Funcs["Format"]
		assert.True(t, funcExists, "The 'Format' function must be found within the 'helpers' package data")
	})
}

func TestTypeBuilder_PackageMadness(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping type builder package madness tests in short mode")
	}

	projectDir := "./testdata/13_package_madness"
	moduleName := "testproject_madness"
	manager := setupManagerForProject(t, projectDir, moduleName)
	sourceContents := getSourceContentsForTest(t, projectDir)

	err := manager.Build(context.Background(), sourceContents, map[string]string{})
	require.NoError(t, err)

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Inspector should have been created successfully")

	typeData, err := manager.GetTypeData()
	require.NoError(t, err)
	require.NotNil(t, typeData)

	responseType := goast.NewIdent("Response")
	mainPackagePath := moduleName

	mainFilePath := findMainFilePath(t, projectDir, sourceContents)

	t.Run("Ground Truth: main package import map correctness", func(t *testing.T) {

		imports := inspector.GetImportsForFile(mainPackagePath, mainFilePath)
		require.NotNil(t, imports)

		expectedExternalPath := "testproject_madness/third_party/github.com/external/helpers"

		assert.Equal(t, "testproject_madness/pkg/api", imports["apiv1"])
		assert.Equal(t, "testproject_madness/pkg/api/v2", imports["api"])
		assert.Equal(t, "testproject_madness/pkg/utils", imports["localhelpers"])
		assert.Equal(t, expectedExternalPath, imports["helpers"])
		assert.Equal(t, "testproject_madness/pkg/http", imports["myhttp"])
		assert.Equal(t, "net/http", imports["http"])

		assert.Equal(t, "testproject_madness/pkg/dot", imports["."])
	})

	t.Run("Scenario 4 & 5: should distinguish aliased vs. default name from collided packages", func(t *testing.T) {
		fieldInfoUser := inspector.FindFieldInfo(context.Background(), responseType, "APIUser", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfoUser)
		assert.Equal(t, "apiv1.User", goastutil.ASTToTypeString(fieldInfoUser.Type))

		fieldInfoProduct := inspector.FindFieldInfo(context.Background(), responseType, "APIProduct", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfoProduct)
		assert.Equal(t, "api.Product", goastutil.ASTToTypeString(fieldInfoProduct.Type))
	})

	t.Run("Scenario 1 & 2: should distinguish local vs. external packages with same name", func(t *testing.T) {
		fieldInfoLocal := inspector.FindFieldInfo(context.Background(), responseType, "Local", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfoLocal)
		assert.Equal(t, "localhelpers.UtilHelper", goastutil.ASTToTypeString(fieldInfoLocal.Type))

		fieldInfoExternal := inspector.FindFieldInfo(context.Background(), responseType, "External", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfoExternal)
		assert.Equal(t, "helpers.ExternalHelper", goastutil.ASTToTypeString(fieldInfoExternal.Type))
	})

	t.Run("Scenario 7: should resolve dot-imported type correctly", func(t *testing.T) {
		fieldInfo := inspector.FindFieldInfo(context.Background(), responseType, "DotInfo", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfo)

		assert.Equal(t, "dot.DotImported", goastutil.ASTToTypeString(fieldInfo.Type))
		assert.Equal(t, "dot", fieldInfo.PackageAlias)
	})

	t.Run("Scenario 9: should resolve generic type from aliased, name-mismatched package", func(t *testing.T) {
		fieldInfo := inspector.FindFieldInfo(context.Background(), responseType, "Boxed", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfo)
		assert.Equal(t, "localhelpers.Box[apiv1.User]", goastutil.ASTToTypeString(fieldInfo.Type))
	})

	t.Run("Scenario 10: should find promoted method from embedded, aliased type", func(t *testing.T) {
		sig := inspector.FindMethodSignature(responseType, "Get", mainPackagePath, mainFilePath)
		require.NotNil(t, sig, "Should find promoted 'Get' method from embedded myhttp.Client")
		require.Len(t, sig.Results, 1)
		assert.Equal(t, "string", sig.Results[0])
	})

	t.Run("Scenario 8: should not crash on blank import", func(t *testing.T) {
		_, ok := typeData.Packages["testproject_madness/pkg/blank"]
		assert.True(t, ok, "Package from blank import should be loaded")
	})
}

func TestTypeBuilder_GenericFieldResolution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping type builder generic field resolution tests in short mode")
	}

	projectDir := "./testdata/14_generic_field_resolution"
	moduleName := "testproject_generics"
	manager := setupManagerForProject(t, projectDir, moduleName)
	sourceContents := getSourceContentsForTest(t, projectDir)

	err := manager.Build(context.Background(), sourceContents, map[string]string{})
	require.NoError(t, err)

	inspector, ok := manager.GetQuerier()
	require.True(t, ok)

	mainPackagePath := moduleName
	modelsPackagePath := moduleName + "/models"
	mainFilePath := findMainFilePath(t, projectDir, sourceContents)
	modelsFilePath := findAnyFileInPackage(t, sourceContents, "models")

	responseType := goast.NewIdent("Response")

	boxedItemField := inspector.FindFieldInfo(context.Background(), responseType, "BoxedItem", mainPackagePath, mainFilePath)
	require.NotNil(t, boxedItemField, "Should find the 'BoxedItem' field")

	contentField := inspector.FindFieldInfo(context.Background(), boxedItemField.Type, "Content", mainPackagePath, mainFilePath)
	require.NotNil(t, contentField, "Should find the 'Content' field on the aliased generic type")

	require.NotNil(t, contentField.SubstMap, "FieldInfo must carry the substitution map from the generic base type")
	require.Contains(t, contentField.SubstMap, "T", "SubstMap must contain the type parameter 'T'")

	replacementType, ok := contentField.SubstMap["T"]
	require.True(t, ok)
	assert.Equal(t, "models.Item", goastutil.ASTToTypeString(replacementType), "The substitution map must map 'T' to 'models.Item'")

	expectedContentType := "models.Item"
	actualContentType := goastutil.ASTToTypeString(contentField.Type)
	assert.Equal(t, expectedContentType, actualContentType, "The type of the 'Content' field should be the substituted generic argument")

	valueField := inspector.FindFieldInfo(context.Background(), contentField.Type, "Value", modelsPackagePath, modelsFilePath)
	require.NotNil(t, valueField, "Should find the 'Value' field on the concrete Item type")
	assert.Equal(t, "string", goastutil.ASTToTypeString(valueField.Type))
}

func TestTypeBuilder_UUIDCollisionNightmare(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping type builder UUID collision tests in short mode")
	}

	projectDir := "./testdata/16_uuid_collision_nightmare"
	moduleName := "testproject_uuid_nightmare"
	manager := setupManagerForProject(t, projectDir, moduleName)
	sourceContents := getSourceContentsForTest(t, projectDir)

	err := manager.Build(context.Background(), sourceContents, map[string]string{})
	require.NoError(t, err)

	inspector, ok := manager.GetQuerier()
	require.True(t, ok, "Inspector should have been created successfully")

	typeData, err := manager.GetTypeData()
	require.NoError(t, err)
	require.NotNil(t, typeData)

	baseResponseType := goast.NewIdent("Response")
	mainPackagePath := moduleName

	mainFilePath := findMainFilePath(t, projectDir, sourceContents)
	localUUIDFilePath := findAnyFileInPackage(t, sourceContents, "uuid")

	t.Run("Ground Truth: main package import map correctness", func(t *testing.T) {

		imports := inspector.GetImportsForFile(mainPackagePath, mainFilePath)
		assert.Equal(t, "github.com/google/uuid", imports["uuid"], "The unaliased 'uuid' should map to google/uuid")
		assert.Equal(t, "github.com/gofrs/uuid/v5", imports["gofrsuuid"])
		assert.Equal(t, "modernc.org/libc/uuid/uuid", imports["moderncuuid"])
		assert.Equal(t, "testproject_uuid_nightmare/pkg/uuid", imports["localuuid"])
		assert.Equal(t, "testproject_uuid_nightmare/pkg/models", imports["models"])
	})

	t.Run("Direct Fields in Response", func(t *testing.T) {
		testCases := []struct {
			fieldName           string
			expectedPackagePath string
			expectedTypeName    string
		}{
			{fieldName: "RealGoogleUUID", expectedPackagePath: "github.com/google/uuid", expectedTypeName: "UUID"},
			{fieldName: "LocalFakeUUID", expectedPackagePath: "testproject_uuid_nightmare/pkg/uuid", expectedTypeName: "UUID"},
			{fieldName: "GofrsUUID", expectedPackagePath: "github.com/gofrs/uuid/v5", expectedTypeName: "UUID"},
			{fieldName: "ModerncUUID", expectedPackagePath: "modernc.org/libc/uuid/uuid", expectedTypeName: "Uuid_t"},
		}

		for _, tc := range testCases {
			t.Run(tc.fieldName, func(t *testing.T) {

				fieldInfo := inspector.FindFieldInfo(context.Background(), baseResponseType, tc.fieldName, mainPackagePath, mainFilePath)
				require.NotNil(t, fieldInfo)

				assert.Equal(t, tc.expectedPackagePath, fieldInfo.CanonicalPackagePath)

				unqualifiedType := goastutil.UnqualifyTypeExpr(fieldInfo.Type)
				assert.Equal(t, tc.expectedTypeName, goastutil.ASTToTypeString(unqualifiedType))
			})
		}
	})

	t.Run("Transitive Fields in Models", func(t *testing.T) {
		testCases := []struct {
			modelFieldName      string
			modelFileName       string
			uuidFieldName       string
			expectedPackagePath string
			expectedTypeName    string
		}{
			{modelFieldName: "ModelA", modelFileName: "a.go", uuidFieldName: "UUID", expectedPackagePath: "github.com/gofrs/uuid/v5", expectedTypeName: "UUID"},
			{modelFieldName: "ModelB", modelFileName: "b.go", uuidFieldName: "UUID", expectedPackagePath: "testproject_uuid_nightmare/pkg/uuid", expectedTypeName: "UUID"},
			{modelFieldName: "ModelC", modelFileName: "c.go", uuidFieldName: "UUID", expectedPackagePath: "github.com/google/uuid", expectedTypeName: "UUID"},
			{modelFieldName: "ModelD", modelFileName: "d.go", uuidFieldName: "UUID", expectedPackagePath: "modernc.org/libc/uuid/uuid", expectedTypeName: "Uuid_t"},
			{modelFieldName: "ModelE", modelFileName: "e.go", uuidFieldName: "UUID", expectedPackagePath: "testproject_uuid_nightmare/pkg/models", expectedTypeName: "UUID"},
		}

		for _, tc := range testCases {
			t.Run(tc.modelFieldName, func(t *testing.T) {
				modelsPackagePath := "testproject_uuid_nightmare/pkg/models"

				modelFilePath := findFileContaining(t, sourceContents, "type "+tc.modelFieldName+" struct")

				modelFieldInfo := inspector.FindFieldInfo(context.Background(), baseResponseType, tc.modelFieldName, mainPackagePath, mainFilePath)
				require.NotNil(t, modelFieldInfo, "Could not find field %s on Response", tc.modelFieldName)

				uuidFieldInfo := inspector.FindFieldInfo(context.Background(), modelFieldInfo.Type, tc.uuidFieldName, modelsPackagePath, modelFilePath)
				require.NotNil(t, uuidFieldInfo, "Could not find field %s on %s", tc.uuidFieldName, tc.modelFieldName)

				assert.Equal(t, tc.expectedPackagePath, uuidFieldInfo.CanonicalPackagePath)
				unqualifiedType := goastutil.UnqualifyTypeExpr(uuidFieldInfo.Type)
				assert.Equal(t, tc.expectedTypeName, goastutil.ASTToTypeString(unqualifiedType))
			})
		}
	})

	t.Run("Method Resolution on Correct Type", func(t *testing.T) {
		fieldInfo := inspector.FindFieldInfo(context.Background(), baseResponseType, "RealGoogleUUID", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfo)

		sig := inspector.FindMethodSignature(fieldInfo.Type, "String", mainPackagePath, mainFilePath)
		require.NotNil(t, sig, "Should find the .String() method on the google/uuid type")

		assert.Empty(t, sig.Params)
		require.Len(t, sig.Results, 1)
		assert.Equal(t, "string", sig.Results[0])
	})

	t.Run("Method Resolution on Incorrect Type", func(t *testing.T) {
		fieldInfo := inspector.FindFieldInfo(context.Background(), baseResponseType, "LocalFakeUUID", mainPackagePath, mainFilePath)
		require.NotNil(t, fieldInfo)

		localUUIDPackagePath := "testproject_uuid_nightmare/pkg/uuid"
		sig := inspector.FindMethodSignature(fieldInfo.Type, "String", localUUIDPackagePath, localUUIDFilePath)
		assert.Nil(t, sig, "Should NOT find the .String() method on our local fake uuid type")
	})
}
