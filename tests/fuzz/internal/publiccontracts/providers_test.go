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

package publiccontracts

import (
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublicFamilyAndAdapterCoverage(t *testing.T) {
	repository := os.DirFS("../../../..")
	rows, err := Providers(repository)
	require.NoError(t, err)
	require.NotEmpty(t, rows)
	seen := map[string]bool{}
	modules := 0
	overlays := 0
	for _, row := range rows {
		if row.Execution == OverlayTest {
			overlays++
		} else {
			require.Equal(t, PackageTest, row.Execution, "%s: unknown execution kind", row.Family)
		}
		require.NotEmpty(t, row.Prerequisite, "empty family contract: %v", row)
		require.NotEmpty(t, row.Operation, "empty family contract: %v", row)
		for _, adapter := range row.Adapters {
			require.NotEmpty(t, adapter.Prerequisite, "%v", adapter)
			seen[adapter.Source] = true
			if adapter.SeparateModule {
				modules++
			}
		}
	}
	files, err := fs.Glob(repository, "wdk/*/*/go.mod")
	require.NoError(t, err)
	for _, file := range files {
		assert.True(t, seen[strings.TrimSuffix(file, "/go.mod")], "missing module %s", file)
	}
	assert.Equal(t, 8, overlays, "overlay tests")
	assert.Equal(t, len(files), modules)
	assert.True(t, seen["wdk/events/events_provider_gochannel"], "incomplete public contract inventory")
}

func TestRemovedPublicContractIsRejected(t *testing.T) {
	repository := fstest.MapFS{
		"wdk/events/events_provider_gochannel/doc.go":            {},
		"tests/fuzz/internal/publiccontracts/testdata/events.go": {Data: []byte("package noop\n// func TestProviderLifecycleContract(t *testing.T) {}\n")},
	}
	_, err := Providers(repository)
	require.ErrorIs(t, err, ErrInvalidRegistry)
	assert.Contains(t, err.Error(), "no longer defines TestProviderLifecycleContract")
}

func TestProvidersRequiresWDKDirectory(t *testing.T) {
	_, err := Providers(fstest.MapFS{})
	require.Error(t, err)
}

func TestAdapterPrerequisiteFallsBackToLocalPackage(t *testing.T) {
	assert.Equal(t, "local Go package, its build dependencies and in-process input fixtures", adapterPrerequisite("misc", "plain"))
	assert.Equal(t, "local Ollama server with a preloaded model for inference", adapterPrerequisite("llm", "llm_provider_ollama"))
}
