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

package i18n_adapters

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestLoaderModeConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, LoaderMode("flatbuffer"), LoaderModeFlatBuffer)
	assert.Equal(t, LoaderMode("json"), LoaderModeJSON)
}

func TestNewLoader_DefaultMode(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{})
	require.NotNil(t, loader)
	assert.Equal(t, LoaderModeFlatBuffer, loader.Mode())
}

func TestNewLoader_DefaultLocale(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{})
	assert.Equal(t, "en-GB", loader.Config().DefaultLocale)
}

func TestNewLoader_CustomMode(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{Mode: LoaderModeJSON})
	assert.Equal(t, LoaderModeJSON, loader.Mode())
}

func TestNewLoader_CustomLocale(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{DefaultLocale: "fr-FR"})
	assert.Equal(t, "fr-FR", loader.Config().DefaultLocale)
}

func TestNewLoader_FullConfig(t *testing.T) {
	t.Parallel()

	config := LoaderConfig{
		Mode:           LoaderModeJSON,
		DefaultLocale:  "de-DE",
		FlatBufferPath: "dist/i18n.bin",
		JSONDirectory:  "i18n",
	}

	loader := NewLoader(config)
	assert.Equal(t, LoaderModeJSON, loader.Mode())
	assert.Equal(t, "de-DE", loader.Config().DefaultLocale)
	assert.Equal(t, "dist/i18n.bin", loader.Config().FlatBufferPath)
	assert.Equal(t, "i18n", loader.Config().JSONDirectory)
}

func TestLoader_Mode(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{Mode: LoaderModeFlatBuffer})
	assert.Equal(t, LoaderModeFlatBuffer, loader.Mode())

	loader2 := NewLoader(LoaderConfig{Mode: LoaderModeJSON})
	assert.Equal(t, LoaderModeJSON, loader2.Mode())
}

func TestLoader_Config(t *testing.T) {
	t.Parallel()

	config := LoaderConfig{
		Mode:          LoaderModeJSON,
		DefaultLocale: "es-ES",
		JSONDirectory: "translations",
	}
	loader := NewLoader(config)

	result := loader.Config()
	assert.Equal(t, LoaderModeJSON, result.Mode)
	assert.Equal(t, "es-ES", result.DefaultLocale)
	assert.Equal(t, "translations", result.JSONDirectory)
}

func TestLoader_Load_UnknownMode(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{Mode: "unknown"})
	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown loader mode")
}

func TestLoader_Load_FlatBufferMissingPath(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	loader := NewLoader(LoaderConfig{
		Mode:    LoaderModeFlatBuffer,
		Sandbox: sandbox,
	})

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "FlatBuffer path is required")
}

func TestLoader_Load_FlatBufferMissingSandbox(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{
		Mode:           LoaderModeFlatBuffer,
		FlatBufferPath: "dist/i18n.bin",
	})

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox is required")
}

func TestLoader_Load_FlatBufferFileNotFound(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	loader := NewLoader(LoaderConfig{
		Mode:           LoaderModeFlatBuffer,
		Sandbox:        sandbox,
		FlatBufferPath: "nonexistent/i18n.bin",
	})

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load FlatBuffer translations")
}

func TestLoader_Load_JSONMissingDirectory(t *testing.T) {
	t.Parallel()

	sandbox, _ := safedisk.NewNoOpSandbox(t.TempDir(), safedisk.ModeReadWrite)
	defer func() { _ = sandbox.Close() }()

	loader := NewLoader(LoaderConfig{
		Mode:    LoaderModeJSON,
		Sandbox: sandbox,
	})

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JSON directory is required")
}

func TestLoader_Load_JSONMissingSandbox(t *testing.T) {
	t.Parallel()

	loader := NewLoader(LoaderConfig{
		Mode:          LoaderModeJSON,
		JSONDirectory: "i18n",
	})

	_, err := loader.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sandbox is required")
}
