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
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

package lsp_domain

import (
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"piko.sh/piko/cmd/lsp/internal/lsp/gopls_bridge"
)

func TestApplyGoplsInitOptions(t *testing.T) {
	t.Parallel()

	t.Run("leaves the flag default when no options are sent", func(t *testing.T) {
		t.Parallel()

		server := &Server{goplsBridgeEnabled: true}
		server.applyGoplsInitOptions(&protocol.InitializeParams{})
		assert.True(t, server.goplsBridgeEnabled)
	})

	t.Run("goBridge=false overrides an enabled default", func(t *testing.T) {
		t.Parallel()

		server := &Server{goplsBridgeEnabled: true}
		server.applyGoplsInitOptions(&protocol.InitializeParams{
			InitializationOptions: map[string]any{"goBridge": false},
		})
		assert.False(t, server.goplsBridgeEnabled)
	})

	t.Run("goBridge=true overrides a disabled default", func(t *testing.T) {
		t.Parallel()

		server := &Server{goplsBridgeEnabled: false}
		server.applyGoplsInitOptions(&protocol.InitializeParams{
			InitializationOptions: map[string]any{"goBridge": true},
		})
		assert.True(t, server.goplsBridgeEnabled)
	})
}

func TestGoplsBridgeActiveRequiresManager(t *testing.T) {
	t.Parallel()

	server := &Server{goplsBridgeEnabled: true, goplsManager: nil}
	assert.False(t, server.goplsBridgeActive(), "a nil manager means the bridge is inactive")
}

func TestMapLocationsClassifiesURIs(t *testing.T) {
	t.Parallel()

	mapper := gopls_bridge.NewMapper("file:///site/pages/cards.pk", "file:///site/piko-lsp/h/source.pk.go", 10, 1)
	request := &goplsRequest{mapper: mapper, virtualURI: mapper.VirtualURI()}

	locations := []protocol.Location{
		{URI: mapper.VirtualURI(), Range: protocol.Range{Start: protocol.Position{Line: 2}, End: protocol.Position{Line: 2}}},
		{URI: "file:///site/piko-lsp/other/source.pk.go"},
		{URI: "file:///go/pkg/mod/piko.sh/piko/facade.go", Range: protocol.Range{Start: protocol.Position{Line: 5}}},
	}
	mapped := request.mapLocations(locations)

	assert.Len(t, mapped, 2, "the primary overlay is rewritten, the satellite dropped, the dep passed through")
	assert.Equal(t, mapper.RealURI(), mapped[0].URI)
	assert.Equal(t, uint32(11), mapped[0].Range.Start.Line, "virtual line 2 maps to .pk line 11 (content starts at line 10)")
	assert.Equal(t, protocol.DocumentURI("file:///go/pkg/mod/piko.sh/piko/facade.go"), mapped[1].URI)
}
