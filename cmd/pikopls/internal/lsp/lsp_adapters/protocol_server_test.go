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
	"context"
	"errors"
	"log/slog"
	"net"
	"testing"

	protocol "github.com/politepixels/golang-language-server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.lsp.dev/jsonrpc2"
	"piko.sh/piko/cmd/pikopls/internal/lsp/lsp_domain"
	"piko.sh/piko/wdk/goroutine"
)

var (
	errReplyFailed = errors.New("reply failed")
)

type panickingProtocolServer struct {
	protocol.Server
}

func (panickingProtocolServer) Hover(context.Context, *protocol.HoverParams) (*protocol.Hover, error) {
	panic("hover exploded")
}

func (panickingProtocolServer) Shutdown(context.Context) error {
	return nil
}

type initialisingProtocolServer struct {
	protocol.Server
}

func (initialisingProtocolServer) Initialize(context.Context, *protocol.InitializeParams) (*protocol.InitializeResult, error) {
	return &protocol.InitializeResult{
		Capabilities: protocol.ServerCapabilities{HoverProvider: true},
		ServerInfo:   &protocol.ServerInfo{Name: "pikopls", Version: "9.9.9"},
	}, nil
}

func (initialisingProtocolServer) Shutdown(context.Context) error {
	return nil
}

func TestRecoverHandlerPanics(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		handler     jsonrpc2.Handler
		wantErr     error
		name        string
		wantReplies int
		replyPanic  bool
		returnPanic bool
	}{
		{
			name: "a panic before replying is sent as the reply",
			handler: func(context.Context, jsonrpc2.Replier, jsonrpc2.Request) error {
				panic("boom")
			},
			wantReplies: 1,
			replyPanic:  true,
		},
		{
			name: "a panic after replying is returned without a second reply",
			handler: func(ctx context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
				_ = reply(ctx, "partial", nil)
				panic("boom")
			},
			wantReplies: 1,
			returnPanic: true,
		},
		{
			name: "a handler error passes through unchanged",
			handler: func(ctx context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
				_ = reply(ctx, nil, nil)
				return errReplyFailed
			},
			wantReplies: 1,
			wantErr:     errReplyFailed,
		},
		{
			name: "a successful handler returns nil",
			handler: func(ctx context.Context, reply jsonrpc2.Replier, _ jsonrpc2.Request) error {
				return reply(ctx, "ok", nil)
			},
			wantReplies: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(1), protocol.MethodTextDocumentHover, nil)
			require.NoError(t, err)
			var replyErrors []error
			reply := func(_ context.Context, _ any, replyErr error) error {
				replyErrors = append(replyErrors, replyErr)
				return nil
			}

			err = recoverHandlerPanics(testCase.handler)(context.Background(), reply, request)

			require.Len(t, replyErrors, testCase.wantReplies)
			switch {
			case testCase.replyPanic:
				require.NoError(t, err)
				requirePanicError(t, replyErrors[0])
			case testCase.returnPanic:
				requirePanicError(t, err)
			default:
				assert.Equal(t, testCase.wantErr, err)
			}
		})
	}
}

func TestServeProtocolAnswersPanickingRequestsWithErrors(t *testing.T) {
	t.Parallel()

	serverEnd, clientEnd := net.Pipe()
	serverConn, _ := serveProtocol(t.Context(), panickingProtocolServer{}, jsonrpc2.NewStream(serverEnd), slog.Default())
	clientConn := jsonrpc2.NewConn(jsonrpc2.NewStream(clientEnd))
	clientConn.Go(t.Context(), jsonrpc2.MethodNotFoundHandler)
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
		<-clientConn.Done()
		<-serverConn.Done()
	})

	params := &protocol.HoverParams{}
	params.TextDocument.URI = "untitled:Untitled-1"

	for _, method := range []string{protocol.MethodTextDocumentHover, protocol.MethodTextDocumentDefinition} {
		var result any
		_, err := clientConn.Call(t.Context(), method, params, &result)

		require.Error(t, err, "%s is answered with an error", method)
		assert.Contains(t, err.Error(), "panic in "+requestPanicComponentPrefix+method)
	}

	select {
	case <-serverConn.Done():
		t.Fatal("the connection must stay open after a handler panics")
	default:
	}
}

func requirePanicError(t *testing.T, err error) {
	t.Helper()

	panicErr, ok := errors.AsType[*goroutine.PanicError](err)
	require.True(t, ok, "the panic is reported as a PanicError")
	assert.Equal(t, requestPanicComponentPrefix+protocol.MethodTextDocumentHover, panicErr.Component)
	assert.Equal(t, "boom", panicErr.Value)
	assert.NotContains(t, err.Error(), "goroutine ", "the error carries no stack trace")
}

func TestServeProtocolAdvertisesExtensionCapabilitiesOnInitialise(t *testing.T) {
	t.Parallel()

	serverEnd, clientEnd := net.Pipe()
	serverConn, _ := serveProtocol(t.Context(), initialisingProtocolServer{}, jsonrpc2.NewStream(serverEnd), slog.Default())
	clientConn := jsonrpc2.NewConn(jsonrpc2.NewStream(clientEnd))
	clientConn.Go(t.Context(), jsonrpc2.MethodNotFoundHandler)
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
		<-clientConn.Done()
		<-serverConn.Done()
	})

	var result struct {
		Capabilities map[string]any      `json:"capabilities"`
		ServerInfo   protocol.ServerInfo `json:"serverInfo"`
	}
	_, err := clientConn.Call(t.Context(), protocol.MethodInitialize, &protocol.InitializeParams{}, &result)
	require.NoError(t, err)

	assert.Equal(t, true, result.Capabilities["hoverProvider"])
	assert.Equal(t, true, result.Capabilities["inlayHintProvider"])
	assert.Equal(t, true, result.Capabilities["typeHierarchyProvider"])
	assert.Equal(t, "9.9.9", result.ServerInfo.Version)
}

func TestAdvertiseExtensionCapabilities(t *testing.T) {
	t.Parallel()

	initialiseResult := &protocol.InitializeResult{}
	testCases := []struct {
		result      any
		replyErr    error
		name        string
		method      string
		wantWrapped bool
	}{
		{name: "initialisation replies are decorated", method: protocol.MethodInitialize, result: initialiseResult, wantWrapped: true},
		{name: "initialisation errors pass through", method: protocol.MethodInitialize, result: nil, replyErr: errReplyFailed},
		{name: "other methods pass through", method: protocol.MethodTextDocumentHover, result: "hover"},
		{name: "unexpected initialisation results pass through", method: protocol.MethodInitialize, result: "unexpected"},
		{name: "nil initialisation results pass through", method: protocol.MethodInitialize, result: (*protocol.InitializeResult)(nil)},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request, err := jsonrpc2.NewCall(jsonrpc2.NewNumberID(1), testCase.method, nil)
			require.NoError(t, err)

			var replied any
			var repliedErr error
			reply := func(_ context.Context, result any, replyErr error) error {
				replied = result
				repliedErr = replyErr
				return nil
			}
			handler := func(ctx context.Context, innerReply jsonrpc2.Replier, _ jsonrpc2.Request) error {
				return innerReply(ctx, testCase.result, testCase.replyErr)
			}

			require.NoError(t, advertiseExtensionCapabilities(handler)(t.Context(), reply, request))

			assert.Equal(t, testCase.replyErr, repliedErr)
			if testCase.wantWrapped {
				extended, ok := replied.(*lsp_domain.ExtendedInitialiseResult)
				require.True(t, ok, "the initialisation result is replaced by the extended form")
				assert.NotNil(t, extended)
				return
			}
			assert.Equal(t, testCase.result, replied)
		})
	}
}
