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

package gopls_bridge

import (
	"context"
	"sync"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func TestClientHandlerMarkReadyIsIdempotent(t *testing.T) {
	t.Parallel()

	handler := newClientHandler(t.TempDir())
	assert.False(t, isClosed(handler.ready), "a fresh handler is not ready")

	handler.markReady()
	handler.markReady()

	assert.True(t, isClosed(handler.ready), "markReady closes the ready channel")
}

func TestClientHandlerProgress(t *testing.T) {
	t.Parallel()

	t.Run("end marks the handler ready", func(t *testing.T) {
		t.Parallel()

		handler := newClientHandler(t.TempDir())
		require.NoError(t, handler.Progress(context.Background(), &protocol.ProgressParams{
			Value: map[string]any{"kind": "end"},
		}))
		assert.True(t, isClosed(handler.ready))
	})

	t.Run("begin does not mark the handler ready", func(t *testing.T) {
		t.Parallel()

		handler := newClientHandler(t.TempDir())
		require.NoError(t, handler.Progress(context.Background(), &protocol.ProgressParams{
			Value: map[string]any{"kind": "begin"},
		}))
		assert.False(t, isClosed(handler.ready))
	})

	t.Run("non-map value is ignored without panic", func(t *testing.T) {
		t.Parallel()

		handler := newClientHandler(t.TempDir())
		require.NoError(t, handler.Progress(context.Background(), &protocol.ProgressParams{Value: "not-a-map"}))
		assert.False(t, isClosed(handler.ready))
	})

	t.Run("non-string kind is ignored without panic", func(t *testing.T) {
		t.Parallel()

		handler := newClientHandler(t.TempDir())
		require.NoError(t, handler.Progress(context.Background(), &protocol.ProgressParams{
			Value: map[string]any{"kind": 42},
		}))
		assert.False(t, isClosed(handler.ready))
	})
}

func TestClientHandlerPublishDiagnostics(t *testing.T) {
	t.Parallel()

	t.Run("forwards to the installed sink", func(t *testing.T) {
		t.Parallel()

		handler := newClientHandler(t.TempDir())
		var (
			mu       sync.Mutex
			received *protocol.PublishDiagnosticsParams
		)
		handler.setSink(func(_ context.Context, params *protocol.PublishDiagnosticsParams) {
			mu.Lock()
			received = params
			mu.Unlock()
		})

		params := &protocol.PublishDiagnosticsParams{URI: "file:///x.pk.go"}
		require.NoError(t, handler.PublishDiagnostics(context.Background(), params))

		mu.Lock()
		defer mu.Unlock()
		require.NotNil(t, received)
		assert.Equal(t, protocol.DocumentURI("file:///x.pk.go"), received.URI)
	})

	t.Run("is a no-op without a sink", func(t *testing.T) {
		t.Parallel()

		handler := newClientHandler(t.TempDir())
		require.NoError(t, handler.PublishDiagnostics(context.Background(), &protocol.PublishDiagnosticsParams{}))
	})
}

func TestClientHandlerWorkspaceFolders(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	handler := newClientHandler(root)

	folders, err := handler.WorkspaceFolders(context.Background())
	require.NoError(t, err)
	require.Len(t, folders, 1)
	assert.Equal(t, string(fileURI(root)), folders[0].URI)
	assert.NotEmpty(t, folders[0].Name)
}

func TestClientHandlerConfiguration(t *testing.T) {
	t.Parallel()

	handler := newClientHandler(t.TempDir())
	result, err := handler.Configuration(context.Background(), &protocol.ConfigurationParams{
		Items: []protocol.ConfigurationItem{{}, {}, {}},
	})
	require.NoError(t, err)
	assert.Len(t, result, 3, "one nil entry per requested item")
	for _, item := range result {
		assert.Nil(t, item)
	}
}

func TestClientHandlerApplyEditRefused(t *testing.T) {
	t.Parallel()

	handler := newClientHandler(t.TempDir())
	response, err := handler.ApplyEdit(context.Background(), &protocol.ApplyWorkspaceEditParams{})
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.False(t, response.Applied, "gopls-originated edits are never applied to .pk files")
}

func TestClientHandlerShowMessageRequestDeclines(t *testing.T) {
	t.Parallel()

	handler := newClientHandler(t.TempDir())
	action, err := handler.ShowMessageRequest(context.Background(), &protocol.ShowMessageRequestParams{})
	require.NoError(t, err)
	assert.Nil(t, action, "declining a prompt is represented by a nil action")
}

func TestClientHandlerInertCallbacks(t *testing.T) {
	t.Parallel()

	handler := newClientHandler(t.TempDir())
	ctx := context.Background()

	require.NoError(t, handler.WorkDoneProgressCreate(ctx, &protocol.WorkDoneProgressCreateParams{}))
	require.NoError(t, handler.LogMessage(ctx, &protocol.LogMessageParams{Message: "hello"}))
	require.NoError(t, handler.ShowMessage(ctx, &protocol.ShowMessageParams{Message: "hello"}))
	require.NoError(t, handler.Telemetry(ctx, map[string]any{"k": "v"}))
	require.NoError(t, handler.RegisterCapability(ctx, &protocol.RegistrationParams{}))
	require.NoError(t, handler.UnregisterCapability(ctx, &protocol.UnregistrationParams{}))
}
