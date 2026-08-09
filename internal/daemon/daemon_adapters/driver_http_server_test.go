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

package daemon_adapters

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
)

func TestFormatServerURL_PortOnly(t *testing.T) {
	t.Parallel()

	result := formatServerURL(":8080", false)
	assert.Equal(t, "http://localhost:8080", result)
}

func TestFormatServerURL_HostAndPort(t *testing.T) {
	t.Parallel()

	result := formatServerURL("0.0.0.0:8080", false)
	assert.Equal(t, "http://0.0.0.0:8080", result)
}

func TestFormatServerURL_LocalhostAndPort(t *testing.T) {
	t.Parallel()

	result := formatServerURL("localhost:3000", false)
	assert.Equal(t, "http://localhost:3000", result)
}

func TestFormatServerURL_EmptyString(t *testing.T) {
	t.Parallel()

	result := formatServerURL("", false)
	assert.Equal(t, "http://", result)
}

func TestFormatServerURL_DomainAndPort(t *testing.T) {
	t.Parallel()

	result := formatServerURL("example.com:443", false)
	assert.Equal(t, "http://example.com:443", result)
}

func TestFormatServerURL_IPV4Address(t *testing.T) {
	t.Parallel()

	result := formatServerURL("192.168.1.1:9000", false)
	assert.Equal(t, "http://192.168.1.1:9000", result)
}

func TestFormatServerURL_PortOnlyZero(t *testing.T) {
	t.Parallel()

	result := formatServerURL(":0", false)
	assert.Equal(t, "http://localhost:0", result)
}

func TestNewDriverHTTPServerAdapter_ReturnsNonNil(t *testing.T) {
	t.Parallel()

	adapter := NewDriverHTTPServerAdapter()
	require.NotNil(t, adapter)
}

func TestNewDriverHTTPServerAdapter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	adapter := NewDriverHTTPServerAdapter()
	_, ok := adapter.(*driverHTTPServerAdapter)
	assert.True(t, ok)
}

func TestNewDriverHTTPServerAdapter_HasMainPurpose(t *testing.T) {
	t.Parallel()

	adapter, ok := NewDriverHTTPServerAdapter().(*driverHTTPServerAdapter)
	require.True(t, ok, "expected *driverHTTPServerAdapter")
	assert.Equal(t, serverPurposeMain, adapter.purpose)
}

func TestShutdown_NilServer_ReturnsNil(t *testing.T) {
	t.Parallel()

	adapter := &driverHTTPServerAdapter{purpose: serverPurposeMain}
	err := adapter.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestRecordServerCompletion_NilError_SetsOkStatus(t *testing.T) {
	t.Parallel()

	span := newNoopSpan()

	recordServerCompletion(context.Background(), span, nil)

	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Ok, span.statusCode)
	assert.Equal(t, "HTTP server started successfully", span.statusDesc)
}

func TestRecordServerCompletion_ServerClosedError_SetsOkStatus(t *testing.T) {
	t.Parallel()

	span := newNoopSpan()

	recordServerCompletion(context.Background(), span, http.ErrServerClosed)

	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Ok, span.statusCode)
	assert.Equal(t, "HTTP server closed gracefully", span.statusDesc)
}

func TestRecordServerCompletion_OtherError_SetsErrorStatus(t *testing.T) {
	t.Parallel()

	span := newNoopSpan()

	recordServerCompletion(context.Background(), span, errors.New("bind failed"))

	assert.True(t, span.statusSet)
	assert.Equal(t, codes.Error, span.statusCode)
	assert.Equal(t, "HTTP server failed", span.statusDesc)
	assert.True(t, span.errorRecorded)
}

func TestServerPurposeMain_Value(t *testing.T) {
	t.Parallel()
	assert.Equal(t, serverPurpose("main"), serverPurposeMain)
}

func TestServerPurposeHealth_Value(t *testing.T) {
	t.Parallel()
	assert.Equal(t, serverPurpose("health"), serverPurposeHealth)
}

func TestFormatServerURL_WithTLS_PortOnly(t *testing.T) {
	t.Parallel()
	result := formatServerURL(":8443", true)
	assert.Equal(t, "https://localhost:8443", result)
}

func TestFormatServerURL_WithTLS_HostAndPort(t *testing.T) {
	t.Parallel()
	result := formatServerURL("0.0.0.0:443", true)
	assert.Equal(t, "https://0.0.0.0:443", result)
}

func TestAlpnProtocols(t *testing.T) {
	t.Parallel()

	t.Run("keeps a configured preference order", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []string{"http/1.1"}, alpnProtocols([]string{"http/1.1"}),
			"an explicit list is the caller's choice, including one that omits h2")
	})

	t.Run("defaults to HTTP/2 then HTTP/1.1", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, []string{alpnProtocolHTTP2, alpnProtocolHTTP11}, alpnProtocols(nil),
			"the server declares HTTP/2, so the listener must be able to negotiate it")
		assert.Equal(t, []string{alpnProtocolHTTP2, alpnProtocolHTTP11}, alpnProtocols([]string{}))
	})
}

func TestNewDriverHTTPServerAdapter_NoTLSConfig(t *testing.T) {
	t.Parallel()

	adapter := NewDriverHTTPServerAdapter()
	a, ok := adapter.(*driverHTTPServerAdapter)
	require.True(t, ok)
	assert.Nil(t, a.tlsConfig)
}

func TestFormatTLSVersion_Known(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "TLS 1.2", formatTLSVersion(0x0303))
	assert.Equal(t, "TLS 1.3", formatTLSVersion(0x0304))
}

func TestFormatTLSVersion_Unknown(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "unknown", formatTLSVersion(0x0000))
}

func TestDriverHTTPServerAdapter_BuildServerConfiguresHTTP2(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	t.Run("without TLS accepts cleartext HTTP/2", func(t *testing.T) {
		t.Parallel()

		adapter := &driverHTTPServerAdapter{}
		server := adapter.buildServer(context.Background(), ":0", handler)

		require.NotNil(t, server.Protocols, "protocols must be configured")
		assert.True(t, server.Protocols.UnencryptedHTTP2(),
			"cleartext HTTP/2 replaces the h2c handler wrapper when TLS is off")
		assert.True(t, server.Protocols.HTTP1(), "HTTP/1.1 must remain available")
		assert.True(t, server.Protocols.HTTP2(), "HTTP/2 must remain available")
	})

	t.Run("with TLS negotiates HTTP/2 over ALPN only", func(t *testing.T) {
		t.Parallel()

		adapter := &driverHTTPServerAdapter{tlsConfig: &TLSAdapterConfig{}}
		server := adapter.buildServer(context.Background(), ":0", handler)

		require.NotNil(t, server.Protocols, "protocols must be configured")
		assert.False(t, server.Protocols.UnencryptedHTTP2(),
			"a TLS listener carries no cleartext connections to upgrade")
		assert.True(t, server.Protocols.HTTP2(), "HTTP/2 must be offered over ALPN")
	})

	t.Run("carries the standard timeouts", func(t *testing.T) {
		t.Parallel()

		adapter := &driverHTTPServerAdapter{}
		server := adapter.buildServer(context.Background(), ":0", handler)

		assert.Equal(t, defaultReadTimeout, server.ReadTimeout)
		assert.Equal(t, defaultWriteTimeout, server.WriteTimeout)
		assert.Equal(t, defaultIdleTimeout, server.IdleTimeout,
			"HTTP/2 has no idle setting of its own, so it inherits this one")
		assert.Equal(t, defaultReadHeaderTimeout, server.ReadHeaderTimeout)
		assert.Equal(t, defaultMaxHeaderBytes, server.MaxHeaderBytes)
	})

	t.Run("preserves HTTP/2 tuning", func(t *testing.T) {
		t.Parallel()

		adapter := &driverHTTPServerAdapter{}
		server := adapter.buildServer(context.Background(), ":0", handler)

		require.NotNil(t, server.HTTP2, "HTTP/2 configuration must be set")
		assert.Equal(t, http2MaxConcurrentStreams, server.HTTP2.MaxConcurrentStreams)
		assert.Equal(t, http2SendPingTimeout, server.HTTP2.SendPingTimeout)
		assert.Equal(t, http2PingTimeout, server.HTTP2.PingTimeout)
	})

	t.Run("counts protocol errors after the starting context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		adapter := &driverHTTPServerAdapter{}
		server := adapter.buildServer(ctx, ":0", handler)
		cancel()

		require.NotNil(t, server.HTTP2.CountError, "protocol errors must still be counted")
		assert.NotPanics(t, func() { server.HTTP2.CountError("frame_headers_bad_path") },
			"the server outlives the context that started it, so counting must survive cancellation")
	})
}
