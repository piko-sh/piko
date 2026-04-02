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

package component_linker_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/001_prop_validation"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/002_required_and_default_props"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/003_invocation_canonicalisation"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/004_prop_tag_and_coercion"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/005_complex_prop_type"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/006_request_overrides"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/007_component_with_no_props"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/008_prop_type_is_local_alias"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/009_class_attributes_and_canonicalisation"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/010_multifile_go_package_for_props"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/011_coercion_failure_and_cascade"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/012_duplicate_prop_tag_error"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/013_prop_from_embedded_struct"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/014_nil_vs_omitted_prop_canonicalisation"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/015_complex_default_value_parsing"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/016_interaction_with_pfor"
	"piko.sh/piko/internal/annotator/annotator_test/component_linker/testdata/017_server_prop_typo_suggestion"
	"piko.sh/piko/internal/ast/ast_domain"
	"piko.sh/piko/internal/compiler/compiler_domain"
	"piko.sh/piko/internal/goastutil"
	"piko.sh/piko/internal/resolver/resolver_adapters"
)

var testRegistry = map[string]TestCaseDef{
	"001_prop_validation": {
		CreateExpansionResult: testcase_01.CreateExpansionResult,
	},
	"002_required_and_default_props": {
		CreateExpansionResult: testcase_02.CreateExpansionResult,
	},
	"003_invocation_canonicalisation": {
		CreateExpansionResult: testcase_03.CreateExpansionResult,
	},
	"004_prop_tag_and_coercion": {
		CreateExpansionResult: testcase_04.CreateExpansionResult,
	},
	"005_complex_prop_type": {
		CreateExpansionResult: testcase_05.CreateExpansionResult,
	},
	"006_request_overrides": {
		CreateExpansionResult: testcase_06.CreateExpansionResult,
	},
	"007_component_with_no_props": {
		CreateExpansionResult: testcase_07.CreateExpansionResult,
	},
	"008_prop_type_is_local_alias": {
		CreateExpansionResult: testcase_08.CreateExpansionResult,
	},
	"009_class_attributes_and_canonicalisation": {
		CreateExpansionResult: testcase_09.CreateExpansionResult,
	},
	"010_multifile_go_package_for_props": {
		CreateExpansionResult: testcase_10.CreateExpansionResult,
	},
	"011_coercion_failure_and_cascade": {
		CreateExpansionResult: testcase_11.CreateExpansionResult,
	},
	"012_duplicate_prop_tag_error": {
		CreateExpansionResult: testcase_12.CreateExpansionResult,
	},
	"013_prop_from_embedded_struct": {
		CreateExpansionResult: testcase_13.CreateExpansionResult,
	},
	"014_nil_vs_omitted_prop_canonicalisation": {
		CreateExpansionResult: testcase_14.CreateExpansionResult,
	},
	"015_complex_default_value_parsing": {
		CreateExpansionResult: testcase_15.CreateExpansionResult,
	},
	"016_interaction_with_pfor": {
		CreateExpansionResult: testcase_16.CreateExpansionResult,
	},
	"017_server_prop_typo_suggestion": {
		CreateExpansionResult: testcase_17.CreateExpansionResult,
	},
}

func TestComponentLinker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping component linker integration tests in short mode")
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
		if entry.IsDir() {
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

			entryFileName := "main.pk"

			entryPointModulePath := filepath.ToSlash(filepath.Join(moduleName, entryFileName))

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
}
