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

package agents_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pluginManifestPath = ".claude-plugin/plugin.json"
	lspConfigPath = ".lsp.json"
	marketplaceManifestPath = "../../.claude-plugin/marketplace.json"
	pikoLanguageID = "piko"
)

var (
	kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	semanticVersion = regexp.MustCompile(`^\d+\.\d+\.\d+([-+][0-9A-Za-z.-]+)?$`)
)

type manifestAuthor struct {
	Name string `json:"name"`
	Email string `json:"email,omitempty"`
	URL string `json:"url,omitempty"`
}

type pluginManifest struct {
	Author *manifestAuthor `json:"author"`
	Name string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Version string `json:"version"`
	Homepage string `json:"homepage"`
	Repository string `json:"repository"`
	Licence string `json:"license"`
	Keywords []string `json:"keywords"`
}

type lspServer struct {
	ExtensionToLanguage map[string]string `json:"extensionToLanguage"`
	InitialisationOptions map[string]any `json:"initializationOptions"`
	Env map[string]string `json:"env"`
	Command string `json:"command"`
	Args []string `json:"args"`
}

type marketplacePlugin struct {
	Name string `json:"name"`
	Source string `json:"source"`
	Description string `json:"description"`
}

type marketplaceManifest struct {
	Owner *manifestAuthor `json:"owner"`
	Name string `json:"name"`
	Description string `json:"description"`
	Plugins []marketplacePlugin `json:"plugins"`
}

func TestPluginManifestSchema(t *testing.T) {
	t.Parallel()

	manifest := decodeStrict[pluginManifest](t, pluginManifestPath)

	assert.Regexp(t, kebabCase, manifest.Name)
	assert.NotEmpty(t, manifest.Description)
	assert.Regexp(t, semanticVersion, manifest.Version)
	require.NotNil(t, manifest.Author)
	assert.NotEmpty(t, manifest.Author.Name)
}

func TestLSPConfigSchema(t *testing.T) {
	t.Parallel()

	servers := decodeStrict[map[string]lspServer](t, lspConfigPath)

	server, ok := servers[pikoLanguageID]
	require.True(t, ok, "%s must define the %q server", lspConfigPath, pikoLanguageID)
	assert.Equal(t, "pikopls", server.Command)
	assert.Equal(t, map[string]string{".pk": pikoLanguageID, ".pkc": pikoLanguageID}, server.ExtensionToLanguage)
	assert.Equal(t, true, server.InitialisationOptions["goBridge"])

	for extension := range server.ExtensionToLanguage {
		assert.Regexp(t, `^\.[a-z0-9]+$`, extension)
	}
}

func TestMarketplaceManifestSchema(t *testing.T) {
	t.Parallel()

	marketplace := decodeStrict[marketplaceManifest](t, marketplaceManifestPath)

	manifest := decodeStrict[pluginManifest](t, pluginManifestPath)

	assert.Regexp(t, kebabCase, marketplace.Name)
	require.NotNil(t, marketplace.Owner)
	assert.NotEmpty(t, marketplace.Owner.Name)
	require.Len(t, marketplace.Plugins, 1)

	entry := marketplace.Plugins[0]
	assert.Equal(t, manifest.Name, entry.Name)
	assert.NotEmpty(t, entry.Description)

	repositoryRoot := filepath.Dir(filepath.Dir(marketplaceManifestPath))
	sourceDirectory, err := filepath.Abs(filepath.Join(repositoryRoot, entry.Source))
	require.NoError(t, err)
	pluginDirectory, err := filepath.Abs(".")
	require.NoError(t, err)
	assert.Equal(t, pluginDirectory, sourceDirectory, "marketplace source must point at plugins/agents")
}

func decodeStrict[T any](t *testing.T, path string) T {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var target T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	require.NoError(t, decoder.Decode(&target), "%s does not match the expected schema", path)
	assert.False(t, decoder.More(), "%s has trailing data", path)
	return target
}
