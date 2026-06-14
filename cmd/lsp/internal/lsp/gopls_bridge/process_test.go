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
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
)

type failingGopls struct {
	protocol.Server
}

func (*failingGopls) Initialize(_ context.Context, _ *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return nil, errors.New("gopls refused to initialise")
}

func (*failingGopls) Shutdown(_ context.Context) error { return nil }

func (*failingGopls) Exit(_ context.Context) error { return nil }

func TestChildServerAccessor(t *testing.T) {
	t.Parallel()

	server := &recordingServer{}
	child := newTestChild(server)
	assert.Same(t, server, child.Server())
}

func TestProcessStreamWriteAndClose(t *testing.T) {
	t.Parallel()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	stream := &processStream{ReadCloser: reader, writer: writer}

	count, writeErr := stream.Write([]byte("hello"))
	require.NoError(t, writeErr)
	assert.Equal(t, 5, count)

	require.NoError(t, stream.Close())
}

func TestProcessStreamWriteHonoursDeadline(t *testing.T) {
	t.Parallel()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = reader.Close(); _ = writer.Close() })

	stream := &processStream{ReadCloser: reader, writer: writer}
	payload := make([]byte, 4<<20)

	done := make(chan error, 1)
	go func() {
		_, writeErr := stream.Write(payload)
		done <- writeErr
	}()

	select {
	case writeErr := <-done:
		require.Error(t, writeErr, "a write to a non-draining pipe must fail under the deadline")
		assert.ErrorIs(t, writeErr, os.ErrDeadlineExceeded)
	case <-time.After(goplsWriteTimeout + 5*time.Second):
		t.Fatal("processStream.Write hung past its write deadline")
	}
}

func TestSpawnGoplsStartError(t *testing.T) {
	t.Parallel()

	_, _, err := spawnGopls(filepath.Join(t.TempDir(), "nonexistent-gopls"), t.TempDir())
	require.Error(t, err, "spawning a missing binary fails")
}

func TestDialChildInitializeError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	clientEnd, serverEnd := net.Pipe()

	_, serverConn, _ := protocol.NewServer(ctx, &failingGopls{}, jsonrpc2.NewStream(serverEnd), quietLogger())
	defer func() { _ = serverConn.Close() }()

	child, err := dialChild(ctx, clientEnd, nil, t.TempDir(), nil, nil)
	require.Error(t, err, "a refused handshake surfaces an error")
	assert.Nil(t, child)
}

func TestChildMonitorMarksDead(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("relies on a POSIX no-op binary")
	}
	binary, lookErr := exec.LookPath("true")
	if lookErr != nil {
		t.Skip("no `true` binary available")
	}

	command := exec.Command(binary)
	require.NoError(t, command.Start())

	child := &Child{command: command, done: make(chan struct{})}
	child.monitor(context.Background())

	assert.True(t, child.dead.Load(), "the child is marked dead once the process exits")
	select {
	case <-child.done:
	case <-time.After(time.Second):
		t.Fatal("done channel should be closed after the process exits")
	}
}
