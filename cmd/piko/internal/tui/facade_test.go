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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/safedisk"
)

func TestLoadConfig(t *testing.T) {
	t.Run("loads config from sandbox", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("piko.yaml", []byte("tui:\n  endpoint: \"http://custom:9090\"\n  theme: \"dark\"\n  refreshInterval: \"5s\"\n  title: \"My App\"\n"))

		tuiConfig, err := LoadConfig("", WithConfigSandbox(sandbox), WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://custom:9090", tuiConfig.Endpoint)
		assert.Equal(t, "dark", tuiConfig.Theme)
		assert.Equal(t, "5s", tuiConfig.RefreshInterval)
		assert.Equal(t, "My App", tuiConfig.Title)
	})

	t.Run("returns defaults when no config found", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()

		tuiConfig, err := LoadConfig("", WithConfigSandbox(sandbox), WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://localhost:8080", tuiConfig.Endpoint)
		assert.Equal(t, "2s", tuiConfig.RefreshInterval)
		assert.Equal(t, "default", tuiConfig.Theme)
	})

	t.Run("explicit config path takes precedence", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("piko.yaml", []byte("tui:\n  endpoint: \"http://local:1111\"\n"))
		sandbox.AddFile("custom.yaml", []byte("tui:\n  endpoint: \"http://custom:2222\"\n"))

		tuiConfig, err := LoadConfig("custom.yaml", WithConfigSandbox(sandbox), WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://custom:2222", tuiConfig.Endpoint)
	})

	t.Run("environment variables override file values", func(t *testing.T) {
		sandbox := safedisk.NewMockSandbox("/project", safedisk.ModeReadOnly)
		defer func() { _ = sandbox.Close() }()
		sandbox.AddFile("piko.yaml", []byte("tui:\n  endpoint: \"http://file:3333\"\n  theme: \"light\"\n"))

		t.Setenv("PIKO_TUI_ENDPOINT", "http://env:4444")
		t.Setenv("PIKO_TUI_THEME", "retro")

		tuiConfig, err := LoadConfig("", WithConfigSandbox(sandbox), WithHomeDir("/nonexistent"))

		require.NoError(t, err)
		assert.Equal(t, "http://env:4444", tuiConfig.Endpoint)
		assert.Equal(t, "retro", tuiConfig.Theme)
	})
}
