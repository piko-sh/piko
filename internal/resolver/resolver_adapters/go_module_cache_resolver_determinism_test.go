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

package resolver_adapters

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	moduleBoundaryDeterminismRuns = 32
)

func TestFindModuleBoundaryForAbsPath_SharedReplaceDirResolvesTheSameWay(t *testing.T) {
	t.Parallel()

	shared := filepath.Join(t.TempDir(), "shared")
	resolver := &GoModuleCacheResolver{
		replaceDirs: map[string]string{
			"example.com/d": shared,
			"example.com/b": shared,
			"example.com/a": shared,
			"example.com/c": shared,
			"example.com/e": shared,
			"example.com/f": shared,
		},
	}

	for range moduleBoundaryDeterminismRuns {
		modulePath, subpath, err := resolver.findModuleBoundaryForAbsPath(filepath.Join(shared, "pkg", "file.go"))

		require.NoError(t, err)
		require.Equal(t, "example.com/a", modulePath, "the same module must own a shared replace directory on every run")
		require.Equal(t, "pkg/file.go", subpath)
	}
}
