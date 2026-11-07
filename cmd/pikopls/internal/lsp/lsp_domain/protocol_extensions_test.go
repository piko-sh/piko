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

package lsp_domain

import (
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/internal/json"
)

func TestExtendInitialiseResult_JSONRoundTrip(t *testing.T) {
	result := &protocol.InitializeResult{
		Capabilities: buildServerCapabilities(false),
		ServerInfo:   &protocol.ServerInfo{Name: serverName, Version: "1.2.3"},
	}

	encoded, err := json.Marshal(ExtendInitialiseResult(result))
	require.NoError(t, err)

	var decoded struct {
		Capabilities map[string]any `json:"capabilities"`
		ServerInfo   struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	assert.Equal(t, serverName, decoded.ServerInfo.Name)
	assert.Equal(t, "1.2.3", decoded.ServerInfo.Version)
	assert.Equal(t, true, decoded.Capabilities["inlayHintProvider"])
	assert.Equal(t, true, decoded.Capabilities["typeHierarchyProvider"])
	assert.Equal(t, true, decoded.Capabilities["hoverProvider"])
	assert.Equal(t, true, decoded.Capabilities["referencesProvider"])
	assert.Equal(t, true, decoded.Capabilities["workspaceSymbolProvider"])
	assert.Contains(t, decoded.Capabilities, "documentLinkProvider")
	assert.NotContains(t, decoded.Capabilities, "semanticTokensProvider")

	executeCommand, ok := decoded.Capabilities["executeCommandProvider"].(map[string]any)
	require.True(t, ok, "executeCommandProvider should be an object")
	assert.Equal(t, []any{commandRefreshDiagnostics}, executeCommand["commands"])

	var roundTripped protocol.InitializeResult
	require.NoError(t, json.Unmarshal(encoded, &roundTripped))
	assert.Equal(t, "1.2.3", roundTripped.ServerInfo.Version)
	assert.Equal(t, true, roundTripped.Capabilities.HoverProvider)
}

func TestExtendInitialiseResult_NilResult(t *testing.T) {
	assert.Nil(t, ExtendInitialiseResult(nil))
}
