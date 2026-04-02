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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/internal/annotator/annotator_dto"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewActionGeneratorAdapter(t *testing.T) {
	t.Parallel()

	adapter := NewActionGeneratorAdapter()

	require.NotNil(t, adapter)
	require.NotNil(t, adapter.registryEmitter)
	require.NotNil(t, adapter.wrapperEmitter)
	require.NotNil(t, adapter.typeScriptEmitter)

	var _ generator_domain.ActionGeneratorPort = adapter
}

func TestActionGeneratorAdapter_GenerateActions(t *testing.T) {
	t.Parallel()

	t.Run("nil manifest returns nil", func(t *testing.T) {
		t.Parallel()

		adapter := NewActionGeneratorAdapter()
		err := adapter.GenerateActions(context.Background(), nil, "mymod", t.TempDir())

		require.NoError(t, err)
	})

	t.Run("empty manifest returns nil", func(t *testing.T) {
		t.Parallel()

		adapter := NewActionGeneratorAdapter()
		manifest := &annotator_dto.ActionManifest{}
		err := adapter.GenerateActions(context.Background(), manifest, "mymod", t.TempDir())

		require.NoError(t, err)
	})

	t.Run("manifest with empty actions returns nil", func(t *testing.T) {
		t.Parallel()

		adapter := NewActionGeneratorAdapter()
		manifest := &annotator_dto.ActionManifest{
			Actions: []annotator_dto.ActionDefinition{},
		}
		err := adapter.GenerateActions(context.Background(), manifest, "mymod", t.TempDir())

		require.NoError(t, err)
	})

	t.Run("single action generates all files", func(t *testing.T) {
		t.Parallel()

		outputDir := t.TempDir()
		adapter := NewActionGeneratorAdapter()

		manifest := &annotator_dto.ActionManifest{
			Actions: []annotator_dto.ActionDefinition{
				{
					Name:        "user.create",
					PackagePath: "mymod/actions/user",
					PackageName: "user",
					StructName:  "CreateAction",
					HTTPMethod:  "POST",
					HasError:    true,
				},
			},
		}

		err := adapter.GenerateActions(context.Background(), manifest, "mymod", outputDir)
		require.NoError(t, err)

		registryPath := filepath.Join(outputDir, "dist", "actions", "registry.go")
		wrapperPath := filepath.Join(outputDir, "dist", "actions", "wrappers.go")
		tsPath := filepath.Join(outputDir, "dist", "ts", "actions.gen.ts")

		registryData, err := os.ReadFile(registryPath)
		require.NoError(t, err)
		assert.Contains(t, string(registryData), "package actions")
		assert.Contains(t, string(registryData), "user.create")

		wrapperData, err := os.ReadFile(wrapperPath)
		require.NoError(t, err)
		assert.Contains(t, string(wrapperData), "package actions")
		assert.Contains(t, string(wrapperData), "invokeUserCreate")

		tsData, err := os.ReadFile(tsPath)
		require.NoError(t, err)
		assert.NotEmpty(t, tsData)
		assert.Contains(t, string(tsData), "registerActionFunction('user.create',")
		assert.Contains(t, string(tsData), "registerActionFunction")
	})
}

func TestActionWriteFile(t *testing.T) {
	t.Parallel()

	t.Run("creates parent directories", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		path := filepath.Join(directory, "deep", "nested", "file.go")
		adapter := NewActionGeneratorAdapter()

		err := adapter.writeFile(path, []byte("package main"))

		require.NoError(t, err)

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "package main", string(data))
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		t.Parallel()

		directory := t.TempDir()
		path := filepath.Join(directory, "file.go")
		adapter := NewActionGeneratorAdapter()

		err := adapter.writeFile(path, []byte("version 1"))
		require.NoError(t, err)

		err = adapter.writeFile(path, []byte("version 2"))
		require.NoError(t, err)

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, "version 2", string(data))
	})
}

func TestActionGeneratorAdapter_Sandbox(t *testing.T) {
	t.Parallel()

	t.Run("writes files via sandbox", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer sandbox.Close()

		adapter := NewActionGeneratorAdapter(WithActionSandbox(sandbox))
		manifest := &annotator_dto.ActionManifest{
			Actions: []annotator_dto.ActionDefinition{
				{
					Name:        "user.create",
					PackagePath: "mymod/actions/user",
					PackageName: "user",
					StructName:  "CreateAction",
					HTTPMethod:  "POST",
					HasError:    true,
				},
			},
		}

		err := adapter.GenerateActions(context.Background(), manifest, "mymod", "/project")
		require.NoError(t, err)

		assert.GreaterOrEqual(t, sandbox.CallCounts["MkdirAll"], 1)
		assert.GreaterOrEqual(t, sandbox.CallCounts["WriteFile"], 3)
	})

	t.Run("MkdirAll error propagates", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.MkdirAllErr = errors.New("no space")

		adapter := NewActionGeneratorAdapter(WithActionSandbox(sandbox))
		manifest := &annotator_dto.ActionManifest{
			Actions: []annotator_dto.ActionDefinition{
				{
					Name:        "user.create",
					PackagePath: "mymod/actions/user",
					PackageName: "user",
					StructName:  "CreateAction",
					HTTPMethod:  "POST",
					HasError:    true,
				},
			},
		}

		err := adapter.GenerateActions(context.Background(), manifest, "mymod", "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "creating directory")
	})

	t.Run("WriteFile error propagates", func(t *testing.T) {
		t.Parallel()
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadWrite)
		defer sandbox.Close()
		sandbox.WriteFileErr = errors.New("permission denied")

		adapter := NewActionGeneratorAdapter(WithActionSandbox(sandbox))
		manifest := &annotator_dto.ActionManifest{
			Actions: []annotator_dto.ActionDefinition{
				{
					Name:        "user.create",
					PackagePath: "mymod/actions/user",
					PackageName: "user",
					StructName:  "CreateAction",
					HTTPMethod:  "POST",
					HasError:    true,
				},
			},
		}

		err := adapter.GenerateActions(context.Background(), manifest, "mymod", "/project")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "writing registry")
	})
}
