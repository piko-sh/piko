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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/inspector/inspector_adapters"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func TestDebugGenericAlias(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
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
	}

	baseDir := t.TempDir()
	moduleName := "testproject"

	goModPath := filepath.Join(baseDir, "go.mod")
	goModContent := []byte("module " + moduleName + "\n\ngo 1.23\n")
	err := os.WriteFile(goModPath, goModContent, 0644)
	require.NoError(t, err)

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
	require.NoError(t, err)

	inspector, ok := manager.GetQuerier()
	require.True(t, ok)

	opts := inspector_domain.DumpOptions{
		SanitisePathPrefix:    baseDir,
		FilterPackagePrefixes: []string{"testproject"},
		Format:                inspector_domain.DumpFormatReadable,
	}
	dump, err := manager.DumpTypeData(opts)
	require.NoError(t, err)
	fmt.Println("=== TYPE DATA DUMP ===")
	fmt.Println(dump)
	fmt.Println("=== END TYPE DATA DUMP ===")

	mainFilePath := filepath.Join(baseDir, "main.go")
	startAST := goastutil.TypeStringToAST("facade.SearchResult[models.Doc]")
	resolvedAST := inspector.ResolveToUnderlyingAST(startAST, mainFilePath)
	fmt.Printf("Resolved: %s\n", goastutil.ASTToTypeString(resolvedAST))

	fmt.Println("\n=== DEBUG: Testing field lookup on resolved type ===")

	resolvedAST2, resolvedFilePath := inspector.ResolveToUnderlyingASTWithContext(context.Background(), startAST, mainFilePath)
	fmt.Printf("Resolved AST: %s\n", goastutil.ASTToTypeString(resolvedAST2))
	fmt.Printf("Resolved File Path: %s\n", resolvedFilePath)

	facadeFilePath := filepath.Join(baseDir, "facade/types.go")
	fmt.Printf("\n=== Checking paths ===\n")
	fmt.Printf("baseDir: %s\n", baseDir)
	fmt.Printf("mainFilePath: %s\n", mainFilePath)
	fmt.Printf("facadeFilePath: %s\n", facadeFilePath)

	fmt.Println("\n=== Trying field lookup with original type ===")
	fieldInfo := inspector.FindFieldInfo(context.Background(), startAST, "Item", "testproject", mainFilePath)
	if fieldInfo == nil {
		fmt.Println("Field lookup FAILED: nil")
	} else {
		fmt.Printf("Field lookup SUCCESS: Type=%s, CanonicalPackage=%s\n",
			goastutil.ASTToTypeString(fieldInfo.Type), fieldInfo.CanonicalPackagePath)
	}

	fmt.Println("\n=== Trying field lookup with resolved type directly ===")
	fieldInfo2 := inspector.FindFieldInfo(context.Background(), resolvedAST2, "Item", "testproject/facade", resolvedFilePath)
	if fieldInfo2 == nil {
		fmt.Println("Field lookup with resolved type FAILED: nil")
	} else {
		fmt.Printf("Field lookup with resolved type SUCCESS: Type=%s, CanonicalPackage=%s\n",
			goastutil.ASTToTypeString(fieldInfo2.Type), fieldInfo2.CanonicalPackagePath)
	}

	fmt.Println("\n=== Trying field lookup with runtime package context ===")
	runtimeFilePath := filepath.Join(baseDir, "runtime/types.go")
	fieldInfo3 := inspector.FindFieldInfo(context.Background(), resolvedAST2, "Item", "testproject/runtime", runtimeFilePath)
	if fieldInfo3 == nil {
		fmt.Println("Field lookup with runtime context FAILED: nil")
	} else {
		fmt.Printf("Field lookup with runtime context SUCCESS: Type=%s, CanonicalPackage=%s\n",
			goastutil.ASTToTypeString(fieldInfo3.Type), fieldInfo3.CanonicalPackagePath)
	}
}
