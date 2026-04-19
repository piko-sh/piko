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

package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTempYAML(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}

func TestLoadConfig(t *testing.T) {
	t.Run("loads config from explicit path", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempYAML(t, dir, "tui.yaml",
			"endpoint: http://custom:9090\ntheme: dark\nrefreshInterval: 5s\ntitle: My App\n")

		cfg, err := LoadConfig(path, WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://custom:9090", cfg.Endpoint)
		assert.Equal(t, "dark", cfg.Theme)
		assert.Equal(t, "5s", cfg.RefreshInterval)
		assert.Equal(t, "My App", cfg.Title)
	})

	t.Run("returns defaults when no config found", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)

		cfg, err := LoadConfig("", WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080", cfg.Endpoint)
		assert.Equal(t, "2s", cfg.RefreshInterval)
		assert.Equal(t, "default", cfg.Theme)
	})

	t.Run("explicit config path takes precedence", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		writeTempYAML(t, dir, "tui.yaml", "endpoint: http://local:1111\n")
		customPath := writeTempYAML(t, dir, "custom.yaml", "endpoint: http://custom:2222\n")

		cfg, err := LoadConfig(customPath, WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://custom:2222", cfg.Endpoint)
	})

	t.Run("environment variables override file values", func(t *testing.T) {
		dir := t.TempDir()
		path := writeTempYAML(t, dir, "tui.yaml",
			"endpoint: http://file:3333\ntheme: light\n")

		t.Setenv("PIKO_TUI_ENDPOINT", "http://env:4444")
		t.Setenv("PIKO_TUI_THEME", "retro")

		cfg, err := LoadConfig(path, WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://env:4444", cfg.Endpoint)
		assert.Equal(t, "retro", cfg.Theme)
	})
}
