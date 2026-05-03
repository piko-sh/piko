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

package lsp_adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewLspFSReader(t *testing.T) {
	t.Parallel()

	t.Run("returns reader when both dependencies are non-nil", func(t *testing.T) {
		t.Parallel()

		docCache := lsp_domain.NewDocumentCache()
		fallback := NewOsFSReader()

		reader, err := NewLspFSReader(docCache, fallback)

		require.NoError(t, err)
		require.NotNil(t, reader)
	})

	t.Run("returns error when docCache is nil", func(t *testing.T) {
		t.Parallel()

		reader, err := NewLspFSReader(nil, NewOsFSReader())

		require.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "docCache cannot be nil")
	})

	t.Run("returns error when fallback is nil", func(t *testing.T) {
		t.Parallel()

		reader, err := NewLspFSReader(lsp_domain.NewDocumentCache(), nil)

		require.Error(t, err)
		assert.Nil(t, reader)
		assert.Contains(t, err.Error(), "fallback reader cannot be nil")
	})
}

func TestNewOsFSReader(t *testing.T) {
	t.Parallel()

	t.Run("creates reader with default sandbox behaviour", func(t *testing.T) {
		t.Parallel()

		reader := NewOsFSReader()

		require.NotNil(t, reader)

		osReader, ok := reader.(*osFSReader)
		require.True(t, ok, "expected *osFSReader")
		assert.Nil(t, osReader.sandbox)
	})

	t.Run("creates reader with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		reader := NewOsFSReader(WithOsFSReaderSandbox(sandbox))

		require.NotNil(t, reader)
		osReader, ok := reader.(*osFSReader)
		require.True(t, ok, "expected *osFSReader")
		assert.Equal(t, sandbox, osReader.sandbox)
	})
}

func TestOsFSReader_ReadFile(t *testing.T) {
	t.Parallel()

	t.Run("reads file using injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		require.NoError(t, sandbox.WriteFile("test.txt", []byte("hello world"), 0600))

		reader := NewOsFSReader(WithOsFSReaderSandbox(sandbox))

		content, err := reader.ReadFile(context.Background(), "/test/test.txt")

		require.NoError(t, err)
		assert.Equal(t, []byte("hello world"), content)
	})

	t.Run("returns error when ReadFile fails with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.ReadFileErr = errors.New("disk read error")

		reader := NewOsFSReader(WithOsFSReaderSandbox(sandbox))

		_, err := reader.ReadFile(context.Background(), "/test/missing.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "osFSReader failed to read file")
		assert.Contains(t, err.Error(), "disk read error")
	})

	t.Run("returns error when file not found with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/test", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		reader := NewOsFSReader(WithOsFSReaderSandbox(sandbox))

		_, err := reader.ReadFile(context.Background(), "/test/nonexistent.txt")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "osFSReader failed to read file")
	})
}

func TestSetupLogFile(t *testing.T) {
	t.Parallel()

	t.Run("creates log file with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()

		logSetup := setupLogFile(sandbox, nil)

		require.NotNil(t, logSetup)
		assert.NotNil(t, logSetup.logFile)

		assert.Nil(t, logSetup.sandbox)
		logSetup.close()
	})

	t.Run("returns nil when OpenFile fails with injected sandbox", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/tmp", safedisk.ModeReadWrite)
		defer func() { _ = sandbox.Close() }()
		sandbox.OpenFileErr = errors.New("permission denied")

		logSetup := setupLogFile(sandbox, nil)

		assert.Nil(t, logSetup)
	})
}
