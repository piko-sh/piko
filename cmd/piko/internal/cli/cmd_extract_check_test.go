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
	"testing"

	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/interp/interp_adapters/driver_symbols_extract"
)

func TestReportCheckResultGreenPath(t *testing.T) {
	t.Parallel()

	manifest := &driver_symbols_extract.Manifest{
		Packages: []driver_symbols_extract.PackageConfig{
			{ImportPath: "github.com/a/one"},
			{ImportPath: "github.com/b/two"},
		},
	}
	result := driver_symbols_extract.DiscoverResult{
		RequiredImports: []string{"github.com/a/one", "github.com/b/two"},
	}

	var stdout, stderr bytes.Buffer
	code := reportCheckResult(&stdout, &stderr, "piko-symbols.yaml", manifest, result)

	require.Equal(t, 0, code)
	require.Contains(t, stdout.String(), "piko-symbols.yaml is up-to-date (2 package(s)).")
	require.Empty(t, stderr.String())
}

func TestReportCheckResultExitsOneOnMissing(t *testing.T) {
	t.Parallel()

	manifest := &driver_symbols_extract.Manifest{
		Packages: []driver_symbols_extract.PackageConfig{
			{ImportPath: "github.com/a/one"},
		},
	}
	result := driver_symbols_extract.DiscoverResult{
		RequiredImports: []string{"github.com/a/one", "github.com/new/gap", "github.com/extra/missing"},
	}

	var stdout, stderr bytes.Buffer
	code := reportCheckResult(&stdout, &stderr, "piko-symbols.yaml", manifest, result)

	require.Equal(t, 1, code)
	require.Contains(t, stderr.String(), "is missing 2 package(s)")
	require.Contains(t, stderr.String(), "  - github.com/extra/missing")
	require.Contains(t, stderr.String(), "  - github.com/new/gap")
	require.Contains(t, stderr.String(), "piko extract generate")
	require.Empty(t, stdout.String())
}

func TestReportCheckResultWarnsOnUnusedButExitsZero(t *testing.T) {
	t.Parallel()

	manifest := &driver_symbols_extract.Manifest{
		Packages: []driver_symbols_extract.PackageConfig{
			{ImportPath: "github.com/a/one"},
			{ImportPath: "github.com/stale/unused"},
		},
	}
	result := driver_symbols_extract.DiscoverResult{
		RequiredImports: []string{"github.com/a/one"},
	}

	var stdout, stderr bytes.Buffer
	code := reportCheckResult(&stdout, &stderr, "piko-symbols.yaml", manifest, result)

	require.Equal(t, 0, code, "unused entries must not fail the check")
	require.Contains(t, stdout.String(), "is up-to-date")
	require.Contains(t, stderr.String(), "github.com/stale/unused")
	require.Contains(t, stderr.String(), "not used by the project")
}

func TestReportCheckResultWarnsOnCgoAndGeneric(t *testing.T) {
	t.Parallel()

	manifest := &driver_symbols_extract.Manifest{
		Packages: []driver_symbols_extract.PackageConfig{
			{ImportPath: "github.com/a/one"},
		},
	}
	result := driver_symbols_extract.DiscoverResult{
		RequiredImports:   []string{"github.com/a/one"},
		SkippedCgo:        []string{"github.com/example/cgo"},
		GenericCandidates: []string{"github.com/example/generic"},
	}

	var stdout, stderr bytes.Buffer
	code := reportCheckResult(&stdout, &stderr, "piko-symbols.yaml", manifest, result)

	require.Equal(t, 0, code)
	require.Contains(t, stderr.String(), "github.com/example/cgo")
	require.Contains(t, stderr.String(), "uses cgo")
	require.Contains(t, stderr.String(), "github.com/example/generic")
	require.Contains(t, stderr.String(), "exports generic types")
}
