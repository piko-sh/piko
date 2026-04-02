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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/generator/generator_domain"
	"piko.sh/piko/wdk/safedisk"
)

func TestNewDrivenI18nEmitter(t *testing.T) {
	t.Parallel()

	config := I18nEmitterConfig{}
	sourceSandbox := safedisk.NewMockSandbox("/source", safedisk.ModeReadOnly)
	defer sourceSandbox.Close()
	outputSandbox := safedisk.NewMockSandbox("/output", safedisk.ModeReadWrite)
	defer outputSandbox.Close()

	emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)

	require.NotNil(t, emitter)

	var _ generator_domain.I18nEmitterPort = emitter
}

func TestDrivenI18nEmitter_EmitI18n(t *testing.T) {
	t.Parallel()

	t.Run("empty I18nSourceDir skips emission", func(t *testing.T) {
		t.Parallel()

		config := I18nEmitterConfig{
			I18nSourceDir: "",
		}
		sourceSandbox := safedisk.NewMockSandbox("/source", safedisk.ModeReadOnly)
		defer sourceSandbox.Close()
		outputSandbox := safedisk.NewMockSandbox("/output", safedisk.ModeReadWrite)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		err := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, err)
	})

	t.Run("directory does not exist skips emission", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		config := I18nEmitterConfig{
			I18nSourceDir: "nonexistent",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})

	t.Run("empty directory with no JSON files skips", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "locales")
		require.NoError(t, os.MkdirAll(localesDir, 0750))

		config := I18nEmitterConfig{
			I18nSourceDir: "locales",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})

	t.Run("directory with only non-JSON files skips", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "locales")
		require.NoError(t, os.MkdirAll(localesDir, 0750))
		require.NoError(t, os.WriteFile(filepath.Join(localesDir, "readme.txt"), []byte("not json"), 0640))

		config := I18nEmitterConfig{
			I18nSourceDir: "locales",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})

	t.Run("single locale file generates output", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "locales")
		require.NoError(t, os.MkdirAll(localesDir, 0750))
		require.NoError(t, os.WriteFile(
			filepath.Join(localesDir, "en.json"),
			[]byte(`{"greeting": "Hello"}`),
			0640,
		))

		config := I18nEmitterConfig{
			I18nSourceDir: "locales",
			DefaultLocale: "en",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)

		outputData, readErr := outputSandbox.ReadFile("i18n.bin")
		require.NoError(t, readErr)
		assert.NotEmpty(t, outputData)
	})

	t.Run("BaseDir set resolves path with prefix", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "myproject", "locales")
		require.NoError(t, os.MkdirAll(localesDir, 0750))
		require.NoError(t, os.WriteFile(
			filepath.Join(localesDir, "en.json"),
			[]byte(`{"hello": "world"}`),
			0640,
		))

		config := I18nEmitterConfig{
			BaseDir:       "myproject",
			I18nSourceDir: "locales",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})

	t.Run("BaseDir dot does not prefix path", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "locales")
		require.NoError(t, os.MkdirAll(localesDir, 0750))
		require.NoError(t, os.WriteFile(
			filepath.Join(localesDir, "en.json"),
			[]byte(`{"test": "value"}`),
			0640,
		))

		config := I18nEmitterConfig{
			BaseDir:       ".",
			I18nSourceDir: "locales",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})

	t.Run("empty default locale falls back to en", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "locales")
		require.NoError(t, os.MkdirAll(localesDir, 0750))
		require.NoError(t, os.WriteFile(
			filepath.Join(localesDir, "en.json"),
			[]byte(`{"key": "value"}`),
			0640,
		))

		config := I18nEmitterConfig{
			I18nSourceDir: "locales",
			DefaultLocale: "",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})

	t.Run("invalid JSON file is handled gracefully", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "locales")
		require.NoError(t, os.MkdirAll(localesDir, 0750))
		require.NoError(t, os.WriteFile(
			filepath.Join(localesDir, "en.json"),
			[]byte("not valid json{{{"),
			0640,
		))

		config := I18nEmitterConfig{
			I18nSourceDir: "locales",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})

	t.Run("subdirectories in locales directory are skipped", func(t *testing.T) {
		t.Parallel()

		sourceDir := t.TempDir()
		outputDir := t.TempDir()

		localesDir := filepath.Join(sourceDir, "locales")
		require.NoError(t, os.MkdirAll(filepath.Join(localesDir, "subdir"), 0750))
		require.NoError(t, os.WriteFile(
			filepath.Join(localesDir, "en.json"),
			[]byte(`{"message": "hi"}`),
			0640,
		))

		config := I18nEmitterConfig{
			I18nSourceDir: "locales",
		}

		sourceSandbox, err := safedisk.NewSandbox(sourceDir, safedisk.ModeReadOnly)
		require.NoError(t, err)
		defer sourceSandbox.Close()

		outputSandbox, err := safedisk.NewSandbox(outputDir, safedisk.ModeReadWrite)
		require.NoError(t, err)
		defer outputSandbox.Close()

		emitter := NewDrivenI18nEmitter(config, sourceSandbox, outputSandbox)
		emitErr := emitter.EmitI18n(context.Background(), "i18n.bin")

		require.NoError(t, emitErr)
	})
}
