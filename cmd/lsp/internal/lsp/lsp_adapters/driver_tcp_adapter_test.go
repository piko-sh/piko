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

package lsp_adapters

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"piko.sh/piko/cmd/lsp/internal/lsp/lsp_domain"
	"piko.sh/piko/internal/config"
	"piko.sh/piko/internal/inspector/inspector_domain"
	"piko.sh/piko/internal/inspector/inspector_dto"
)

func validTCPDeps(t *testing.T) TCPAdapterDeps {
	t.Helper()
	return TCPAdapterDeps{
		Addr:                 "localhost:0",
		CoordinatorService:   &stdioStubCoordinatorService{},
		Resolver:             &stdioStubResolverPort{},
		TypeInspectorManager: inspector_domain.NewTypeBuilder(inspector_dto.Config{}),
		DocCache:             lsp_domain.NewDocumentCache(),
		LSPReader:            &stdioStubFSReader{},
		PathsConfig:          &config.PathsConfig{},
		FormattingEnabled:    false,
	}
}

func TestNewTCPAdapter(t *testing.T) {
	t.Parallel()

	t.Run("returns adapter when all dependencies are non-nil", func(t *testing.T) {
		t.Parallel()

		adapter, err := NewTCPAdapter(validTCPDeps(t))

		require.NoError(t, err)
		require.NotNil(t, adapter)
	})

	t.Run("returns error when CoordinatorService is nil", func(t *testing.T) {
		t.Parallel()

		deps := validTCPDeps(t)
		deps.CoordinatorService = nil

		adapter, err := NewTCPAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "coordinatorService cannot be nil")
	})

	t.Run("returns error when Resolver is nil", func(t *testing.T) {
		t.Parallel()

		deps := validTCPDeps(t)
		deps.Resolver = nil

		adapter, err := NewTCPAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "resolver cannot be nil")
	})

	t.Run("returns error when TypeInspectorManager is nil", func(t *testing.T) {
		t.Parallel()

		deps := validTCPDeps(t)
		deps.TypeInspectorManager = nil

		adapter, err := NewTCPAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "typeInspectorManager cannot be nil")
	})

	t.Run("returns error when DocCache is nil", func(t *testing.T) {
		t.Parallel()

		deps := validTCPDeps(t)
		deps.DocCache = nil

		adapter, err := NewTCPAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "docCache cannot be nil")
	})

	t.Run("returns error when LSPReader is nil", func(t *testing.T) {
		t.Parallel()

		deps := validTCPDeps(t)
		deps.LSPReader = nil

		adapter, err := NewTCPAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "lspReader cannot be nil")
	})

	t.Run("returns error when PathsConfig is nil", func(t *testing.T) {
		t.Parallel()

		deps := validTCPDeps(t)
		deps.PathsConfig = nil

		adapter, err := NewTCPAdapter(deps)

		require.Error(t, err)
		assert.Nil(t, adapter)
		assert.Contains(t, err.Error(), "pathsConfig cannot be nil")
	})

	t.Run("applies high defaults for optional knobs", func(t *testing.T) {
		t.Parallel()

		adapter, err := NewTCPAdapter(validTCPDeps(t))
		require.NoError(t, err)
		concrete, ok := adapter.(*tcpAdapter)
		require.True(t, ok)
		assert.Equal(t, int64(defaultMaxLSPMessageBytes), concrete.maxMessageBytes)
		assert.Equal(t, defaultLSPConnectionInactivityTimeout, concrete.connectionInactivityTimeout)
		assert.Equal(t, defaultMaxConcurrentLSPConnections, cap(concrete.connectionSemaphore))
	})

	t.Run("respects explicit knob overrides", func(t *testing.T) {
		t.Parallel()

		deps := validTCPDeps(t)
		deps.MaxMessageBytes = 4096
		deps.MaxConcurrentConnections = 8
		deps.ConnectionInactivityTimeout = time.Minute

		adapter, err := NewTCPAdapter(deps)
		require.NoError(t, err)
		concrete, ok := adapter.(*tcpAdapter)
		require.True(t, ok)
		assert.Equal(t, int64(4096), concrete.maxMessageBytes)
		assert.Equal(t, time.Minute, concrete.connectionInactivityTimeout)
		assert.Equal(t, 8, cap(concrete.connectionSemaphore))
	})
}

type stubReadWriteCloser struct {
	reader io.Reader
	writer bytes.Buffer
	closed bool
}

func (s *stubReadWriteCloser) Read(p []byte) (int, error)  { return s.reader.Read(p) }
func (s *stubReadWriteCloser) Write(p []byte) (int, error) { return s.writer.Write(p) }
func (s *stubReadWriteCloser) Close() error                { s.closed = true; return nil }

func TestCappedReadWriteCloser_RejectsOversizedMessage(t *testing.T) {
	t.Parallel()

	t.Run("returns errMessageTooLarge once cap is exhausted", func(t *testing.T) {
		t.Parallel()

		payload := bytes.Repeat([]byte{'x'}, 1024)
		stub := &stubReadWriteCloser{reader: bytes.NewReader(payload)}
		capped := newCappedReadWriteCloser(stub, 16)

		buf := make([]byte, 1024)
		n, err := capped.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, 16, n)

		_, err = capped.Read(buf)
		require.Error(t, err)
		assert.ErrorIs(t, err, errMessageTooLarge)
	})

	t.Run("non-positive limit disables the cap", func(t *testing.T) {
		t.Parallel()

		payload := bytes.Repeat([]byte{'y'}, 1024)
		stub := &stubReadWriteCloser{reader: bytes.NewReader(payload)}
		capped := newCappedReadWriteCloser(stub, 0)

		total := 0
		for {
			buf := make([]byte, 256)
			n, err := capped.Read(buf)
			total += n
			if err != nil {
				assert.ErrorIs(t, err, io.EOF)
				break
			}
		}
		assert.Equal(t, 1024, total)
	})

	t.Run("write and close pass through to underlying conn", func(t *testing.T) {
		t.Parallel()

		stub := &stubReadWriteCloser{reader: bytes.NewReader(nil)}
		capped := newCappedReadWriteCloser(stub, 1024)

		_, err := capped.Write([]byte("hello"))
		require.NoError(t, err)
		assert.Equal(t, "hello", stub.writer.String())

		require.NoError(t, capped.Close())
		assert.True(t, stub.closed)
	})
}

func TestLSPTCPAdapter_BoundsConcurrentConnections(t *testing.T) {
	t.Parallel()

	deps := validTCPDeps(t)
	deps.MaxConcurrentConnections = 2

	adapter, err := NewTCPAdapter(deps)
	require.NoError(t, err)
	concrete, ok := adapter.(*tcpAdapter)
	require.True(t, ok)

	assert.True(t, concrete.acquireConnectionSlot(), "first acquire should succeed")
	assert.True(t, concrete.acquireConnectionSlot(), "second acquire should succeed")
	assert.False(t, concrete.acquireConnectionSlot(), "third acquire should be rejected because the cap is reached")

	concrete.releaseConnectionSlot()
	assert.True(t, concrete.acquireConnectionSlot(), "acquire should succeed once a slot is freed")

	concrete.releaseConnectionSlot()
	concrete.releaseConnectionSlot()
}

func TestLSPTCPAdapter_DrainHandlersWaitsForGoroutines(t *testing.T) {
	t.Parallel()

	deps := validTCPDeps(t)
	adapter, err := NewTCPAdapter(deps)
	require.NoError(t, err)
	concrete, ok := adapter.(*tcpAdapter)
	require.True(t, ok)

	finished := make(chan struct{})
	concrete.goroutineWG.Go(func() {
		time.Sleep(50 * time.Millisecond)
		close(finished)
	})

	concrete.drainHandlers(t.Context())
	select {
	case <-finished:
	default:
		t.Fatal("drainHandlers returned before goroutine finished")
	}
}

type panickingReadWriteCloser struct{}

func (panickingReadWriteCloser) Read(_ []byte) (int, error)  { panic("forced panic in handler") }
func (panickingReadWriteCloser) Write(_ []byte) (int, error) { return 0, errors.New("write closed") }
func (panickingReadWriteCloser) Close() error                { return nil }

func TestLSPTCPAdapter_RecoversFromHandlerPanic(t *testing.T) {
	t.Parallel()

	recovered := make(chan struct{})
	go func() {
		defer close(recovered)
		defer func() {
			_ = recover()
		}()

		_, _ = newCappedReadWriteCloser(panickingReadWriteCloser{}, 1024).Read(make([]byte, 16))
	}()

	select {
	case <-recovered:
	case <-time.After(2 * time.Second):
		t.Fatal("handler did not unwind")
	}

	deps := validTCPDeps(t)
	adapter, err := NewTCPAdapter(deps)
	require.NoError(t, err)
	concrete, ok := adapter.(*tcpAdapter)
	require.True(t, ok)

	done := make(chan struct{})
	concrete.goroutineWG.Go(func() {
		defer func() {
			_ = recover()
			close(done)
		}()
		panic("simulated handler panic")
	})

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("recovery did not complete")
	}

	concrete.drainHandlers(t.Context())
}

func TestLSPTCPAdapter_RejectsOversizedMessage(t *testing.T) {
	t.Parallel()

	deps := validTCPDeps(t)
	deps.Addr = "127.0.0.1:0"
	deps.MaxMessageBytes = 64
	deps.MaxConcurrentConnections = 4
	deps.ConnectionInactivityTimeout = 5 * time.Second

	adapter, err := NewTCPAdapter(deps)
	require.NoError(t, err)
	concrete, ok := adapter.(*tcpAdapter)
	require.True(t, ok)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = listener.Close() }()

	serverErr := make(chan error, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			serverErr <- acceptErr
			return
		}
		defer func() { _ = conn.Close() }()
		capped := newCappedReadWriteCloser(conn, concrete.maxMessageBytes)
		buf := make([]byte, 256)
		_, readErr := io.ReadAtLeast(capped, buf, 200)
		serverErr <- readErr
	}()

	clientConn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer func() { _ = clientConn.Close() }()

	largeHeader := fmt.Sprintf("Content-Length: %d\r\n\r\n", 100*1024*1024*1024)
	_, _ = clientConn.Write([]byte(largeHeader))
	_, _ = clientConn.Write(bytes.Repeat([]byte{'x'}, 1024))

	select {
	case err := <-serverErr:
		require.Error(t, err)
		assert.ErrorIs(t, err, errMessageTooLarge)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not reject oversized message in time")
	}
}
