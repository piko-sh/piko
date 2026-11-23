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

package scenarios

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/json"
	"piko.sh/piko/tests/fuzz/internal/casefile"
	"piko.sh/piko/tests/fuzz/internal/publiccontracts"
)

func TestPublicOverlayReceiptHashesCompiledBytes(t *testing.T) {
	rows, err := publiccontracts.Providers(os.DirFS("../../../.."))
	require.NoError(t, err)
	repository, work := t.TempDir(), t.TempDir()
	write := func(name, body string) {
		t.Helper()
		name = filepath.Join(repository, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(name), 0o700))
		require.NoError(t, os.WriteFile(name, []byte(body), 0o600))
	}
	var indices []int
	for index, row := range rows {
		require.NoError(t, os.MkdirAll(filepath.Join(repository, "wdk", row.Family), 0o700))
		write(row.Source, "package local\nimport \"testing\"\nfunc "+row.Test+"(t *testing.T){}\n")
		if row.Family == "events" || row.Family == "binder" {
			indices = append(indices, index)
			write(filepath.Join(row.Package, "doc.go"), "package local\n")
			write(filepath.Join(row.Package, "go.mod"), "module example.org/fixture\n\ngo 1.27\n")
		}
	}
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	for _, index := range indices {
		receipt, err := RunPublic(t.Context(), repository, work, &casefile.Case{Target: "provider-contracts", Index: uint64(index)})
		require.NoError(t, err)
		row := rows[index]
		original, err := os.ReadFile(filepath.Join(repository, row.Source))
		require.NoError(t, err)
		ran := original
		if row.Execution == publiccontracts.OverlayTest {
			overlays, err := filepath.Glob(filepath.Join(work, "go-suite-*", "overlay.json"))
			require.NoError(t, err)
			require.Len(t, overlays, 1)
			contents, err := os.ReadFile(overlays[0])
			require.NoError(t, err)
			var overlay overlaySpec
			require.NoError(t, json.Unmarshal(contents, &overlay))
			ran, err = os.ReadFile(overlay.Replace[filepath.Join(repository, row.Package, "piko_fuzz_scenario_test.go")])
			require.NoError(t, err)
			require.False(t, bytes.Equal(original, ran), "test fixture did not distinguish raw input and actual rendered bytes")
		}
		assert.Equal(t, 1, receipt.Steps, row.Test)
		assert.Equal(t, 1, receipt.Assertions, row.Test)
		assert.Equal(t, fmt.Sprintf("%x", sha256.Sum256(ran)), receipt.SourceSHA256, row.Test)
	}
}

func TestRunPackageContractRequiresAPositivePass(t *testing.T) {
	t.Setenv("GOWORK", "off")
	t.Setenv("GOPROXY", "off")
	testCases := []struct {
		name    string
		body    string
		message string
	}{
		{name: "pass", body: "func TestContract(t *testing.T) {}\n", message: ""},
		{name: "failure", body: "func TestContract(t *testing.T) { t.Fatal(\"contract broken\") }\n", message: "provider/fixture/TestContract: "},
		{name: "missing test", body: "func TestOther(t *testing.T) {}\n", message: "infrastructure: provider/fixture missing pass for TestContract"},
		{name: "compile failure", body: "func TestContract(t *testing.T) { undefined() }\n", message: "infrastructure: preparing provider/fixture/TestContract"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			repository := t.TempDir()
			directory := filepath.Join(repository, "wdk", "fixture")
			require.NoError(t, os.MkdirAll(directory, 0o700))
			require.NoError(t, os.WriteFile(filepath.Join(directory, "go.mod"), []byte("module example.org/fixture\n\ngo 1.27\n"), 0o600))
			source := "package fixture\n\nimport \"testing\"\n\n" + testCase.body
			require.NoError(t, os.WriteFile(filepath.Join(directory, "contract_test.go"), []byte(source), 0o600))
			provider := publiccontracts.Provider{Family: "fixture", Package: "wdk/fixture", Source: "wdk/fixture/contract_test.go", Test: "TestContract"}

			err := runPackageContract(t.Context(), repository, &provider)

			if testCase.message == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, testCase.message)
			if testCase.name == "failure" {
				assert.NotContains(t, err.Error(), "infrastructure:")
			}
		})
	}
}
