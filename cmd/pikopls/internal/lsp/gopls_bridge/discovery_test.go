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

package gopls_bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverGoplsPath(t *testing.T) {
	t.Parallel()

	t.Run("returns false for an explicit path that does not exist", func(t *testing.T) {
		t.Parallel()

		resolved, found := DiscoverGoplsPath(filepath.Join(t.TempDir(), "definitely-not-gopls"))

		assert.False(t, found)
		assert.Empty(t, resolved)
	})

	t.Run("resolves an explicit existing executable to its absolute path", func(t *testing.T) {
		t.Parallel()

		executable, err := os.Executable()
		require.NoError(t, err)

		resolved, found := DiscoverGoplsPath(executable)

		assert.True(t, found)
		assert.True(t, filepath.IsAbs(resolved))
	})

	t.Run("rejects a directory", func(t *testing.T) {
		t.Parallel()

		resolved, found := DiscoverGoplsPath(t.TempDir())

		assert.False(t, found)
		assert.Empty(t, resolved)
	})

	t.Run("resolves a bare name found on PATH", func(t *testing.T) {
		t.Parallel()

		resolved, found := DiscoverGoplsPath("go")

		assert.True(t, found)
		assert.NotEmpty(t, resolved)
	})

	t.Run("falls through to search directories for an absent bare name", func(t *testing.T) {
		t.Parallel()

		resolved, found := DiscoverGoplsPath("definitely-not-a-real-binary-xyz")

		assert.False(t, found)
		assert.Empty(t, resolved)
	})
}

func TestGoEnv(t *testing.T) {
	t.Parallel()

	t.Run("returns a value for a supported key", func(t *testing.T) {
		t.Parallel()

		assert.NotEmpty(t, goEnv(goEnvGOPATH))
	})

	t.Run("rejects an unsupported key via the allow-list", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, goEnv("NOT_A_REAL_GO_ENV_KEY"))
	})
}

func TestExecutableAtRejectsMissingAndDirectory(t *testing.T) {
	t.Parallel()

	_, found := executableAt(filepath.Join(t.TempDir(), "missing"))
	assert.False(t, found)

	_, foundDir := executableAt(t.TempDir())
	assert.False(t, foundDir, "a directory is not an executable")
}

func TestGoSubprocessEnvForcesAutoToolchain(t *testing.T) {
	t.Run("sets GOTOOLCHAIN=auto so a project's required toolchain resolves", func(t *testing.T) {
		assert.Equal(t, "auto", effectiveEnvValue(goSubprocessEnv(), "GOTOOLCHAIN"),
			"the gopls subprocess must let go resolve a newer-than-installed toolchain, matching piko's core package loading")
	})

	t.Run("overrides a GOTOOLCHAIN=local in the parent environment", func(t *testing.T) {
		t.Setenv("GOTOOLCHAIN", "local")
		assert.Equal(t, "auto", effectiveEnvValue(goSubprocessEnv(), "GOTOOLCHAIN"),
			"a deliberately local parent env must not re-break projects that require a newer toolchain")
	})
}

func effectiveEnvValue(env []string, key string) string {
	prefix := key + "="
	value := ""
	for _, entry := range env {
		if suffix, found := strings.CutPrefix(entry, prefix); found {
			value = suffix
		}
	}
	return value
}
