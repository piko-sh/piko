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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
)

type fakeGopls struct {
	protocol.Server
	initialised  chan struct{}
	hover        *protocol.Hover
	capabilities protocol.ServerCapabilities
}

func (f *fakeGopls) Initialize(_ context.Context, _ *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{Capabilities: f.capabilities}, nil
}

func (f *fakeGopls) Initialized(_ context.Context, _ *protocol.InitializedParams) error {
	close(f.initialised)
	return nil
}

func (f *fakeGopls) Hover(_ context.Context, _ *protocol.HoverParams) (*protocol.Hover, error) {
	return f.hover, nil
}

func (*fakeGopls) Shutdown(_ context.Context) error { return nil }

func (*fakeGopls) Exit(_ context.Context) error { return nil }

func TestIsBenignCloseError(t *testing.T) {
	t.Parallel()

	benign := []error{
		net.ErrClosed,
		io.ErrClosedPipe,
		os.ErrClosed,
		os.ErrProcessDone,
		fmt.Errorf("close conn: %w", net.ErrClosed),
	}
	for _, candidate := range benign {
		assert.Truef(t, isBenignCloseError(candidate), "%v should be filtered as a benign close", candidate)
	}

	notBenign := []error{
		context.DeadlineExceeded,
		errGoplsShutdownTimeout,
		errors.New("gopls refused shutdown"),
	}
	for _, candidate := range notBenign {
		assert.Falsef(t, isBenignCloseError(candidate), "%v should surface as a real teardown error", candidate)
	}
}

func TestDialChildHandshake(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clientEnd, serverEnd := net.Pipe()

	fake := &fakeGopls{
		initialised: make(chan struct{}),
		capabilities: protocol.ServerCapabilities{
			HoverProvider:      true,
			CompletionProvider: &protocol.CompletionOptions{TriggerCharacters: []string{"."}},
		},
	}
	_, serverConn, _ := protocol.NewServer(ctx, fake, jsonrpc2.NewStream(serverEnd), quietLogger())
	defer func() { _ = serverConn.Close() }()

	child, err := dialChild(ctx, clientEnd, nil, t.TempDir(), nil, nil)
	require.NoError(t, err)
	require.NotNil(t, child)

	assert.Equal(t, true, child.Capabilities().HoverProvider, "gopls capabilities should be captured")
	assert.NotNil(t, child.Capabilities().CompletionProvider)
	assert.True(t, child.IsAlive(), "a freshly dialled child is alive")

	select {
	case <-fake.initialised:
	case <-time.After(2 * time.Second):
		t.Fatal("gopls did not receive the initialized notification")
	}

	require.NoError(t, child.Close(ctx))
}

func TestDialChildReadLoopOutlivesSpawningContext(t *testing.T) {
	t.Parallel()

	spawnCtx, cancelSpawn := context.WithCancel(context.Background())
	clientEnd, serverEnd := net.Pipe()

	fake := &fakeGopls{
		initialised:  make(chan struct{}),
		capabilities: protocol.ServerCapabilities{HoverProvider: true},
		hover:        &protocol.Hover{Contents: protocol.MarkupContent{Value: "alive"}},
	}
	_, serverConn, _ := protocol.NewServer(context.Background(), fake, jsonrpc2.NewStream(serverEnd), quietLogger())
	defer func() { _ = serverConn.Close() }()

	child, err := dialChild(spawnCtx, clientEnd, nil, t.TempDir(), nil, nil)
	require.NoError(t, err)
	defer func() { _ = child.Close(context.Background()) }()

	cancelSpawn()

	for attempt := range 3 {
		callCtx, cancelCall := context.WithTimeout(context.Background(), 2*time.Second)
		hover, hoverErr := child.Server().Hover(callCtx, &protocol.HoverParams{})
		cancelCall()
		require.NoErrorf(t, hoverErr, "call %d: the read loop must survive the spawning context being cancelled", attempt)
		require.NotNil(t, hover)
		assert.Equal(t, "alive", hover.Contents.Value)
	}
	assert.True(t, child.IsAlive())
}
