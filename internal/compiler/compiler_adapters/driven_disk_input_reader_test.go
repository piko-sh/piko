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

package compiler_adapters

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

type directoryFileInfo struct{}

func (directoryFileInfo) Name() string       { return "somedir" }
func (directoryFileInfo) Size() int64        { return 0 }
func (directoryFileInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o755 }
func (directoryFileInfo) ModTime() time.Time { return time.Time{} }
func (directoryFileInfo) IsDir() bool        { return true }
func (directoryFileInfo) Sys() any           { return nil }

func TestDiskInputReader_ReadSFC(t *testing.T) {
	t.Parallel()

	t.Run("reads file successfully", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		expectedContent := []byte("<template><h1>Hello</h1></template>")
		sandbox.AddFile("pages/home.pk", expectedContent)

		reader := NewDiskInputReader(sandbox)

		content, err := reader.ReadSFC(context.Background(), "pages/home.pk")

		require.NoError(t, err)
		assert.Equal(t, expectedContent, content)
	})

	t.Run("returns error when file not found", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		reader := NewDiskInputReader(sandbox)

		content, err := reader.ReadSFC(context.Background(), "pages/missing.pk")

		require.Error(t, err)
		assert.Nil(t, content)
		assert.Contains(t, err.Error(), "cannot stat file")
	})

	t.Run("returns error when stat fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		sandbox.StatErr = errors.New("permission denied")

		reader := NewDiskInputReader(sandbox)

		content, err := reader.ReadSFC(context.Background(), "pages/home.pk")

		require.Error(t, err)
		assert.Nil(t, content)
		assert.Contains(t, err.Error(), "cannot stat file")
	})

	t.Run("returns error when read fails", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		sandbox.AddFile("pages/home.pk", []byte("content"))
		sandbox.ReadFileErr = errors.New("disk read error")

		reader := NewDiskInputReader(sandbox)

		content, err := reader.ReadSFC(context.Background(), "pages/home.pk")

		require.Error(t, err)
		assert.Nil(t, content)
		assert.Contains(t, err.Error(), "failed reading file")
	})

	t.Run("returns error when path is a directory", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		fileHandle := sandbox.AddFile("pages", nil)
		fileHandle.StatInfo = directoryFileInfo{}

		reader := NewDiskInputReader(sandbox)

		content, err := reader.ReadSFC(context.Background(), "pages")

		require.Error(t, err)
		assert.Nil(t, content)
		assert.Contains(t, err.Error(), "source path is a directory")
	})
}
