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

package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/interp/interp_adapters/driver_symbols_extract"
	"piko.sh/piko/wdk/safedisk"
)

func TestWriteInitManifestWritesFreshFile(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/tmp/project", safedisk.ModeReadWrite)
	flags := extractInitFlags{
		output:       "piko-symbols.yaml",
		packageName:  "driven_project_symbols",
		generatedDir: "internal/symbols",
	}
	result := driver_symbols_extract.DiscoverResult{
		RequiredImports: []string{"github.com/a/one", "github.com/b/two"},
	}

	var stdout, stderr bytes.Buffer
	code := writeInitManifest(&stdout, &stderr, sandbox, "piko-symbols.yaml", flags, result)

	require.Equal(t, 0, code)
	require.Contains(t, stdout.String(), "Wrote piko-symbols.yaml with 2 package(s).")
	require.Empty(t, stderr.String())

	written, err := sandbox.ReadFile("piko-symbols.yaml")
	require.NoError(t, err)
	require.Contains(t, string(written), "package: driven_project_symbols")
	require.Contains(t, string(written), "output: internal/symbols")
	require.Contains(t, string(written), "  - github.com/a/one")
	require.Contains(t, string(written), "  - github.com/b/two")
}

func TestWriteInitManifestRefusesExistingFileWithoutForce(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/tmp/project", safedisk.ModeReadWrite)
	sandbox.AddFile("piko-symbols.yaml", []byte("existing"))

	flags := extractInitFlags{output: "piko-symbols.yaml"}
	result := driver_symbols_extract.DiscoverResult{RequiredImports: []string{"github.com/example/pkg"}}

	var stdout, stderr bytes.Buffer
	code := writeInitManifest(&stdout, &stderr, sandbox, "piko-symbols.yaml", flags, result)

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "already exists; use --force to overwrite")
	require.Empty(t, stdout.String())

	kept, err := sandbox.ReadFile("piko-symbols.yaml")
	require.NoError(t, err)
	require.Equal(t, "existing", string(kept), "existing manifest must not be overwritten")
}

func TestWriteInitManifestForceOverwrites(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/tmp/project", safedisk.ModeReadWrite)
	sandbox.AddFile("piko-symbols.yaml", []byte("old content"))

	flags := extractInitFlags{
		output:       "piko-symbols.yaml",
		packageName:  "driven_project_symbols",
		generatedDir: "internal/symbols",
		force:        true,
	}
	result := driver_symbols_extract.DiscoverResult{RequiredImports: []string{"github.com/example/pkg"}}

	var stdout, stderr bytes.Buffer
	code := writeInitManifest(&stdout, &stderr, sandbox, "piko-symbols.yaml", flags, result)

	require.Equal(t, 0, code)

	written, err := sandbox.ReadFile("piko-symbols.yaml")
	require.NoError(t, err)
	require.NotEqual(t, "old content", string(written))
	require.Contains(t, string(written), "github.com/example/pkg")
}

func TestWriteInitManifestReportsCgoAndGenericWarnings(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/tmp/project", safedisk.ModeReadWrite)
	flags := extractInitFlags{
		output:       "piko-symbols.yaml",
		packageName:  "driven_project_symbols",
		generatedDir: "internal/symbols",
	}
	result := driver_symbols_extract.DiscoverResult{
		RequiredImports:   []string{"github.com/example/ok"},
		SkippedCgo:        []string{"github.com/example/cgo"},
		GenericCandidates: []string{"github.com/example/generic"},
	}

	var stdout, stderr bytes.Buffer
	code := writeInitManifest(&stdout, &stderr, sandbox, "piko-symbols.yaml", flags, result)

	require.Equal(t, 0, code)
	require.Contains(t, stderr.String(), "github.com/example/cgo")
	require.Contains(t, stderr.String(), "uses cgo")
	require.Contains(t, stderr.String(), "github.com/example/generic")
	require.Contains(t, stderr.String(), "generic types")
}

func TestRenderInitManifestRoundTripsThroughLoadManifest(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/tmp/project", safedisk.ModeReadWrite)
	flags := extractInitFlags{
		output:       "piko-symbols.yaml",
		packageName:  "driven_project_symbols",
		generatedDir: "internal/symbols",
	}
	result := driver_symbols_extract.DiscoverResult{
		RequiredImports: []string{"github.com/a/one", "github.com/b/two", "example.com/c/three"},
	}

	var stdout, stderr bytes.Buffer
	code := writeInitManifest(&stdout, &stderr, sandbox, "piko-symbols.yaml", flags, result)
	require.Equal(t, 0, code)

	written, err := sandbox.ReadFile("piko-symbols.yaml")
	require.NoError(t, err)

	tmp := t.TempDir() + "/piko-symbols.yaml"
	require.NoError(t, os.WriteFile(tmp, written, 0o644))

	loaded, err := driver_symbols_extract.LoadManifest(tmp)
	require.NoError(t, err, "rendered YAML must parse back through LoadManifest")
	require.Equal(t, flags.packageName, loaded.Package)
	require.Equal(t, flags.generatedDir, loaded.Output)
	require.Len(t, loaded.Packages, len(result.RequiredImports))
}
