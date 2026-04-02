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

package generator_adapters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewFSWriter(t *testing.T) {
	t.Parallel()

	sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
	defer sandbox.Close()
	writer := NewFSWriter(sandbox)

	require.NotNil(t, writer)

	var _ generator_domain.FSWriterPort = writer
}

func TestFSWriter_WriteFile(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		writer := NewFSWriter(sandbox)

		err := writer.WriteFile(context.Background(), "output.go", []byte("package main"))

		require.NoError(t, err)
	})

	errorCases := []struct {
		setupMock func(*safedisk.MockSandbox)
		name      string
	}{
		{
			name: "MkdirAll error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.MkdirAllErr = errors.New("cannot create directory")
			},
		},
		{
			name: "CreateTemp error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.CreateTempErr = errors.New("disk full")
			},
		},
		{
			name: "Write error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileWriteErr = errors.New("write failed")
			},
		},
		{
			name: "Sync error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileSyncErr = errors.New("sync failed")
			},
		},
		{
			name: "Close error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.NextTempFileCloseErr = errors.New("close failed")
			},
		},
		{
			name: "Chmod error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.ChmodErr = errors.New("permission denied")
			},
		},
		{
			name: "Rename error",
			setupMock: func(builder *safedisk.MockSandbox) {
				builder.RenameErr = errors.New("rename failed")
			},
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
			defer sandbox.Close()
			tc.setupMock(sandbox)

			writer := NewFSWriter(sandbox)
			err := writer.WriteFile(context.Background(), "output.go", []byte("package main"))

			require.Error(t, err)
		})
	}
}

func TestFSWriter_ReadDir(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		writer := NewFSWriter(sandbox)

		entries, err := writer.ReadDir("somedir")

		require.NoError(t, err)
		assert.Nil(t, entries)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.ReadDirErr = errors.New("read directory failed")

		writer := NewFSWriter(sandbox)
		_, err := writer.ReadDir("somedir")

		require.Error(t, err)
	})
}

func TestFSWriter_RemoveAll(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		writer := NewFSWriter(sandbox)

		err := writer.RemoveAll("somedir")

		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		sandbox := safedisk.NewMockSandbox("/sandbox", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.RemoveAllErr = errors.New("remove all failed")

		writer := NewFSWriter(sandbox)
		err := writer.RemoveAll("somedir")

		require.Error(t, err)
	})
}
