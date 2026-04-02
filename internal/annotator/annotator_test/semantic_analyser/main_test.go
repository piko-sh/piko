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

package semantic_analyser_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"piko.sh/piko/internal/annotator/annotator_test/semantic_analyser/testdata/001_basic_resolution"
	"piko.sh/piko/internal/annotator/annotator_test/semantic_analyser/testdata/002_pfor_scoping"
	"piko.sh/piko/internal/annotator/annotator_test/semantic_analyser/testdata/003_slotted_content_context_switch"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/resolver/resolver_adapters"

	"github.com/stretchr/testify/require"
)

var testRegistry = map[string]TestCaseDef{
	"001_basic_resolution": {
		CreateLinkingResult: testcase_01.CreateLinkingResult,
	},
	"002_pfor_scoping": {
		CreateLinkingResult: testcase_02.CreateLinkingResult,
	},
	"003_slotted_content_context_switch": {
		CreateLinkingResult: testcase_03.CreateLinkingResult,
	},
}

func TestSemanticAnnotator(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping semantic annotator integration tests in short mode")
	}

	goastutil.ResetDynamicCaches()
	ast_domain.ClearExpressionCache()
	ast_domain.ClearSelectorCache()
	ast_domain.ResetAllPools()
	compiler_domain.ClearIdentifierRegistry()
	compiler_domain.ClearBindingRegistry()
	compiler_domain.ClearLocRefRegistry()

	testdataRoot := "./testdata"

	entries, err := os.ReadDir(testdataRoot)
	if err != nil {
		t.Fatalf("Critical test setup error: Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		testCaseName := entry.Name()
		testDef, ok := testRegistry[testCaseName]
		if !ok {
			t.Logf("Skipping test case '%s': no TestCaseDef registered in main_test.go", testCaseName)
			continue
		}

		testCasePath := filepath.Join(testdataRoot, testCaseName)
		srcPath := filepath.Join(testCasePath, "src")

		tempResolver := resolver_adapters.NewLocalModuleResolver(srcPath)
		err := tempResolver.DetectLocalModule(context.Background())
		require.NoError(t, err, "Failed to detect module for test case %s", testCaseName)
		moduleName := tempResolver.GetModuleName()

		entryPointModulePath := filepath.ToSlash(filepath.Join(moduleName, "main.pk"))

		tc := testCase{
			Name:      testCaseName,
			Path:      testCasePath,
			EntryFile: entryPointModulePath,
			TestDef:   testDef,
		}

		t.Run(tc.Name, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}
