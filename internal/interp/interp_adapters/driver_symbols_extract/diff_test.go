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

package driver_symbols_extract

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiffAllRequiredDeclared(t *testing.T) {
	t.Parallel()

	manifest := &Manifest{
		Packages: []PackageConfig{
			{ImportPath: "github.com/example/a"},
			{ImportPath: "github.com/example/b"},
		},
	}
	discovered := DiscoverResult{
		RequiredImports: []string{"github.com/example/a", "github.com/example/b"},
	}

	got := Diff(manifest, discovered)

	require.Empty(t, got.Missing)
	require.Empty(t, got.Unused)
}

func TestDiffReportsMissing(t *testing.T) {
	t.Parallel()

	manifest := &Manifest{
		Packages: []PackageConfig{
			{ImportPath: "github.com/example/a"},
		},
	}
	discovered := DiscoverResult{
		RequiredImports: []string{"github.com/example/a", "github.com/example/b", "github.com/example/c"},
	}

	got := Diff(manifest, discovered)

	require.Equal(t, []string{"github.com/example/b", "github.com/example/c"}, got.Missing)
	require.Empty(t, got.Unused)
}

func TestDiffReportsUnused(t *testing.T) {
	t.Parallel()

	manifest := &Manifest{
		Packages: []PackageConfig{
			{ImportPath: "github.com/example/a"},
			{ImportPath: "github.com/example/stale"},
		},
	}
	discovered := DiscoverResult{
		RequiredImports: []string{"github.com/example/a"},
	}

	got := Diff(manifest, discovered)

	require.Empty(t, got.Missing)
	require.Equal(t, []string{"github.com/example/stale"}, got.Unused)
}

func TestDiffNilManifestAllMissing(t *testing.T) {
	t.Parallel()

	discovered := DiscoverResult{
		RequiredImports: []string{"github.com/example/a", "github.com/example/b"},
	}

	got := Diff(nil, discovered)

	require.Equal(t, []string{"github.com/example/a", "github.com/example/b"}, got.Missing)
	require.Empty(t, got.Unused)
}
