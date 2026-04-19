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

package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestResolveSandboxPaths(t *testing.T) {
	t.Parallel()

	t.Run("current directory", func(t *testing.T) {
		t.Parallel()
		sandboxDir, checkPath := resolveSandboxPaths(".")
		assert.Equalf(t, ".", checkPath, "checkPath = %q, want %q", checkPath, ".")

		assert.NotEmpty(t, sandboxDir, "sandboxDir should not be empty for current directory")
	})

	t.Run("subdirectory", func(t *testing.T) {
		t.Parallel()
		sandboxDir, checkPath := resolveSandboxPaths("my-project")
		assert.Equalf(t, "my-project", checkPath, "checkPath = %q, want %q", checkPath, "my-project")
		assert.Equalf(t, ".", sandboxDir, "sandboxDir = %q, want %q", sandboxDir, ".")
	})

	t.Run("nested path", func(t *testing.T) {
		t.Parallel()
		sandboxDir, checkPath := resolveSandboxPaths("foo/bar")
		assert.Equalf(t, "bar", checkPath, "checkPath = %q, want %q", checkPath, "bar")
		assert.Equalf(t, "foo", sandboxDir, "sandboxDir = %q, want %q", sandboxDir, "foo")
	})
}

func testFactory(t *testing.T) safedisk.Factory {
	t.Helper()
	factory, err := safedisk.NewFactory(safedisk.FactoryConfig{
		AllowedPaths: []string{os.TempDir()},
		Enabled:      true,
	})
	require.NoErrorf(t, err, "failed to create test factory: %v", err)
	return factory
}

func TestValidateDestination_NewPath(t *testing.T) {
	t.Parallel()

	factory := testFactory(t)
	directory := t.TempDir()
	newPath := filepath.Join(directory, "new-project")

	err := validateDestination(factory, newPath)
	assert.NoErrorf(t, err, "validateDestination() = %v, want nil for non-existent path", err)
}

func TestValidateDestination_ExistingDir(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	existingDir := filepath.Join(directory, "existing")
	require.NoError(t, os.Mkdir(existingDir, 0750))

	err := validateDestination(testFactory(t), existingDir)
	assert.Error(t, err, "validateDestination() should return error for existing directory")
}

func TestValidateDestination_ExistingFile(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	filePath := filepath.Join(directory, "somefile")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0640))

	err := validateDestination(testFactory(t), filePath)
	assert.Error(t, err, "validateDestination() should return error for existing file")
}

func TestCreateDirs(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	dest := filepath.Join(directory, "project")
	require.NoError(t, os.Mkdir(dest, 0750))

	err := createDirs(testFactory(t), ScaffoldData{DestinationPath: dest, EnableInterpreted: true})
	require.NoErrorf(t, err, "createDirs() = %v", err)

	expectedDirs := []string{
		"actions/greeting",
		"cmd/generator",
		"cmd/main",
		"components",
		"pages",
		"partials",
		"lib/icons",
		"styles",
		"dist",
		"e2e",
		"internal/interpreted",
	}

	for _, d := range expectedDirs {
		fullPath := filepath.Join(dest, d)
		info, err := os.Stat(fullPath)
		if !assert.NoErrorf(t, err, "directory %q not created: %v", d, err) {
			continue
		}
		assert.Truef(t, info.IsDir(), "%q exists but is not a directory", d)
	}
}

func TestCreateProject_HappyPath(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	dest := filepath.Join(directory, "test-project")

	data := ScaffoldData{
		ProjectName:     "test-project",
		ModuleName:      "example.com/test-project",
		DestinationPath: dest,
	}

	err := CreateProject(data)
	require.NoErrorf(t, err, "CreateProject() = %v", err)

	expectedFiles := []string{
		"go.mod",
		"README.md",
		"internal/piko.go",
		"cmd/main/main.go",
		"cmd/generator/main.go",
		"actions/greeting/print.go",
		"actions/greeting/submit.go",
		"pages/index.pk",
		"pages/index_test.go",
		"partials/layout.pk",
		"partials/code-window.pk",
		"lib/icons/piko-logo-inline.svg",
		"lib/icons/book-open.svg",
		"lib/icons/bolt.svg",
		"styles/theme.css",
	}

	for _, f := range expectedFiles {
		fullPath := filepath.Join(dest, f)
		_, err := os.Stat(fullPath)
		assert.NoErrorf(t, err, "expected file %q not found: %v", f, err)
	}
}

func TestCreateProject_WithAgents(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	dest := filepath.Join(directory, "agent-project")

	data := ScaffoldData{
		ProjectName:     "agent-project",
		ModuleName:      "example.com/agent-project",
		DestinationPath: dest,
		EnableAgents:    true,
	}

	err := CreateProject(data)
	require.NoErrorf(t, err, "CreateProject() = %v", err)

	expectedFiles := []string{
		"AGENTS.md",
		"references/pk-file-format.md",
		"references/template-syntax.md",
		"references/server-actions.md",
	}

	for _, f := range expectedFiles {
		fullPath := filepath.Join(dest, f)
		_, err := os.Stat(fullPath)
		assert.NoErrorf(t, err, "expected agent file %q not found: %v", f, err)
	}

	excludedFiles := []string{
		"SKILL.md",
		".lsp.json",
		".claude-plugin/plugin.json",
	}

	for _, f := range excludedFiles {
		fullPath := filepath.Join(dest, f)
		_, err := os.Stat(fullPath)
		assert.Errorf(t, err, "Claude Code file %q should not be in project scaffold", f)
	}
}

func TestCreateProject_WithoutAgents(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	dest := filepath.Join(directory, "no-agents")

	data := ScaffoldData{
		ProjectName:     "no-agents",
		ModuleName:      "example.com/no-agents",
		DestinationPath: dest,
		EnableAgents:    false,
	}

	err := CreateProject(data)
	require.NoErrorf(t, err, "CreateProject() = %v", err)

	agentFiles := []string{
		"AGENTS.md",
		"SKILL.md",
		".lsp.json",
		"references",
	}

	for _, f := range agentFiles {
		fullPath := filepath.Join(dest, f)
		_, err := os.Stat(fullPath)
		assert.Errorf(t, err, "agent file %q should not exist when EnableAgents is false", f)
	}
}

func TestCreateProject_ExistingDir(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	dest := filepath.Join(directory, "existing")
	require.NoError(t, os.Mkdir(dest, 0750))

	data := ScaffoldData{
		ProjectName:     "existing",
		DestinationPath: dest,
	}

	err := CreateProject(data)
	assert.Error(t, err, "CreateProject() should return error for existing directory")
}
