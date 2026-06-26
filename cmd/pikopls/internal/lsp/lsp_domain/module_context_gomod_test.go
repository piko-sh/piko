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

package lsp_domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestReadModuleNameFromGoMod(t *testing.T) {
	t.Parallel()

	t.Run("reads module name from valid go.mod", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("go.mod", []byte("module github.com/example/myapp\n\ngo 1.24\n"))

		name, err := readModuleNameFromGoMod(sandbox)

		require.NoError(t, err)
		assert.Equal(t, "github.com/example/myapp", name)
	})

	t.Run("reads module name with extra whitespace", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("go.mod", []byte("  module   github.com/spaced/mod  \n"))

		name, err := readModuleNameFromGoMod(sandbox)

		require.NoError(t, err)
		assert.Equal(t, "github.com/spaced/mod", name)
	})

	t.Run("returns error for missing module line", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("go.mod", []byte("go 1.24\n\nrequire (\n)\n"))

		_, err := readModuleNameFromGoMod(sandbox)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no 'module' line")
	})

	t.Run("returns error when go.mod cannot be read", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.ReadFileErr = errors.New("permission denied")
		sandbox.AddFile("go.mod", []byte("module foo"))

		_, err := readModuleNameFromGoMod(sandbox)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "reading go.mod")
	})

	t.Run("returns error when go.mod does not exist", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		_, err := readModuleNameFromGoMod(sandbox)

		require.Error(t, err)
	})
}
