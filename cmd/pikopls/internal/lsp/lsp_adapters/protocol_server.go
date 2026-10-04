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
	"log/slog"
	"sync/atomic"

	protocol "github.com/politepixels/golang-language-server"
	"go.lsp.dev/jsonrpc2"
	"piko.sh/piko/wdk/goroutine"
)

const (
	// requestPanicComponentPrefix prefixes the method name in the component reported for a
	// request handler that panicked.
	requestPanicComponentPrefix = "lsp.request."
)

// serveProtocol starts serving the language server protocol for server over stream.
//
// Takes server (protocol.Server) which handles the decoded requests.
// Takes stream (jsonrpc2.Stream) which carries the framed JSON-RPC messages.
// Takes logger (*slog.Logger) which logs client-bound traffic.
//
// Returns jsonrpc2.Conn which is the running connection.
// Returns protocol.Client which sends requests and notifications to the client.
func serveProtocol(ctx context.Context, server protocol.Server, stream jsonrpc2.Stream, logger *slog.Logger) (jsonrpc2.Conn, protocol.Client) {
	conn := jsonrpc2.NewConn(stream)
	client := protocol.ClientDispatcher(conn, logger.With("component", "client"))
	ctx = protocol.WithClient(ctx, client)

	conn.Go(ctx, protocol.Handlers(recoverHandlerPanics(protocol.ServerHandler(server, jsonrpc2.MethodNotFoundHandler))))

	return conn, client
}

// recoverHandlerPanics wraps a JSON-RPC handler so that a panic while handling one
// message is logged once with its stack and answered with a stack-free error instead of
// crashing the process.
//
// Takes handler (jsonrpc2.Handler) which handles each message.
//
// Returns jsonrpc2.Handler which recovers panics raised by handler.
func recoverHandlerPanics(handler jsonrpc2.Handler) jsonrpc2.Handler {
	return func(ctx context.Context, reply jsonrpc2.Replier, request jsonrpc2.Request) (err error) {
		var replied atomic.Bool
		trackedReply := func(replyCtx context.Context, result any, replyErr error) error {
			replied.Store(true)
			return reply(replyCtx, result, replyErr)
		}

		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			panicErr := goroutine.HandlePanicRecovery(ctx, requestPanicComponentPrefix+request.Method(), recovered)
			if replied.Load() {
				err = panicErr
				return
			}
			err = reply(ctx, nil, panicErr)
		}()

		return handler(ctx, trackedReply, request)
	}
}
