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

package telemetry_grpcfb

import (
	"context"
	"fmt"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"piko.sh/piko/internal/logger/logger_domain"
)

const (
	// ServiceName is the fully-qualified gRPC service name. FlatBuffers has no service
	// block, so the ServiceDesc is hand-written as a client-streaming ingest RPC.
	ServiceName = "piko.telemetry.v1.Telemetry"

	// IngestMethod is the client-streaming RPC method name.
	IngestMethod = "Ingest"

	// defaultMaxReceiveMessageSize bounds the largest single frame the server will accept
	// off the wire. It matches MaxMessageSize so the gRPC transport rejects oversized frames
	// before they reach the verifier.
	defaultMaxReceiveMessageSize = MaxMessageSize

	// defaultMaxConcurrentStreams bounds how many concurrent ingest streams one connection
	// may open.
	defaultMaxConcurrentStreams = 256

	// ingestFullMethod is the fully-qualified stream path used by the client.
	ingestFullMethod = "/" + ServiceName + "/" + IngestMethod
)

// IngestStream is the server's view of the inbound client stream.
type IngestStream interface {
	// Context returns the stream's context (deadline/cancellation, peer, metadata).
	//
	// Returns context.Context which carries the stream's deadline and metadata.
	Context() context.Context

	// Recv reads the next batch; it returns io.EOF once the client half-closes.
	//
	// Returns *Batch which is the decoded telemetry batch.
	// Returns error which is io.EOF on half-close, or non-nil on failure.
	Recv() (*Batch, error)

	// SendAndClose sends the terminal ack and closes the stream.
	//
	// Takes ack (*IngestAck) which is the terminal acknowledgement to send.
	//
	// Returns error which is non-nil when sending the ack fails.
	SendAndClose(*IngestAck) error
}

// Server is the telemetry ingestion handler.
type Server interface {
	// Ingest consumes the client stream of TelemetryBatch frames and returns an ack.
	//
	// Takes stream (IngestStream) which is the inbound client stream of batches.
	//
	// Returns error which is non-nil when ingestion fails.
	Ingest(stream IngestStream) error
}

var (
	// serviceDesc is the hand-written gRPC ServiceDesc registering the client-streaming
	// ingest RPC, since FlatBuffers schemas carry no service block.
	serviceDesc = grpc.ServiceDesc{
		ServiceName: ServiceName,
		HandlerType: (*Server)(nil),
		Methods:     []grpc.MethodDesc{},
		Streams: []grpc.StreamDesc{
			{
				StreamName:    IngestMethod,
				Handler:       ingestHandler,
				ClientStreams: true,
			},
		},
		Metadata: "piko/telemetry",
	}
)

// serverStream adapts a generic grpc.ServerStream to the typed IngestStream. The embedded
// ServerStream promotes Context(); RecvMsg/SendMsg run through the registered FlatBuffers
// Codec (the server is built with ForceServerCodec).
type serverStream struct {
	grpc.ServerStream
}

// Recv reads the next batch from the stream, returning io.EOF on client half-close.
//
// Returns *Batch which is the decoded telemetry batch.
// Returns error which is io.EOF on half-close, or non-nil on failure.
func (s *serverStream) Recv() (*Batch, error) {
	m := new(Batch)
	if err := s.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// SendAndClose sends the terminal ack and closes the stream.
//
// Takes a (*IngestAck) which is the terminal acknowledgement to send.
//
// Returns error which is non-nil when sending the ack fails.
func (s *serverStream) SendAndClose(a *IngestAck) error {
	return s.SendMsg(a)
}

// ServerOption tunes the gRPC server NewServer builds. Options apply ahead of any
// caller-supplied grpc.ServerOption so callers can still override the defaults.
type ServerOption func(*serverOptions)

// serverOptions holds the resolved server configuration.
type serverOptions struct {
	// maxReceiveMessageSize bounds the largest single frame the server accepts.
	maxReceiveMessageSize int

	// maxConcurrentStreams bounds concurrent ingest streams per connection.
	maxConcurrentStreams uint32
}

// ingestHandler adapts the raw gRPC stream to the typed Server.Ingest entry point.
//
// Takes server (any) which is the registered Server implementation.
// Takes stream (grpc.ServerStream) which is the raw inbound gRPC stream.
//
// Returns error which is whatever Server.Ingest returns.
func ingestHandler(server any, stream grpc.ServerStream) error {
	return server.(Server).Ingest(&serverStream{ServerStream: stream})
}

// recoverStreamInterceptor contains a panicking Ingest handler so one bad client cannot
// crash the sink process.
//
// Takes server (any) which is the registered Server implementation.
// Takes stream (grpc.ServerStream) which is the raw inbound gRPC stream.
// Takes info (*grpc.StreamServerInfo) which describes the invoked stream method.
// Takes handler (grpc.StreamHandler) which is the wrapped handler to invoke.
//
// Returns err (error) which is the handler's error, or codes.Internal on panic.
func recoverStreamInterceptor(server any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		if r := recover(); r != nil {
			_, l := logger_domain.From(stream.Context(), log)
			l.Error("recovered from ingest handler panic", logger_domain.String("panic", fmt.Sprint(r)),
				logger_domain.String("method", info.FullMethod), logger_domain.String("stack", string(debug.Stack())))
			err = status.Error(codes.Internal, "internal error")
		}
	}()
	return handler(server, stream)
}

// WithMaxReceiveMessageSize sets the largest single frame the server accepts off the
// wire. A non-positive size leaves the default (defaultMaxReceiveMessageSize) in place.
//
// Takes n (int) which is the maximum receive frame size in bytes.
//
// Returns ServerOption which applies the receive size limit.
func WithMaxReceiveMessageSize(n int) ServerOption {
	return func(o *serverOptions) {
		if n > 0 {
			o.maxReceiveMessageSize = n
		}
	}
}

// WithMaxConcurrentStreams bounds the concurrent ingest streams one connection may open.
// A zero value leaves the default (defaultMaxConcurrentStreams) in place.
//
// Takes n (uint32) which is the maximum concurrent ingest streams per connection.
//
// Returns ServerOption which applies the concurrent stream limit.
func WithMaxConcurrentStreams(n uint32) ServerOption {
	return func(o *serverOptions) {
		if n > 0 {
			o.maxConcurrentStreams = n
		}
	}
}

// RegisterServer registers a telemetry ingestion handler with a gRPC server (or any
// ServiceRegistrar). The server MUST use the telemetry FlatBuffers codec: build it with
// NewServer, or pass grpc.ForceServerCodec(Codec{}) to grpc.NewServer.
//
// Takes registrar (grpc.ServiceRegistrar) which is the registrar to attach the service
// to.
// Takes server (Server) which is the telemetry ingestion handler to register.
func RegisterServer(registrar grpc.ServiceRegistrar, server Server) {
	registrar.RegisterService(&serviceDesc, server)
}

// NewServer builds a gRPC server preconfigured with the telemetry FlatBuffers codec and a
// bounded receive message size. Telemetry options (opts) resolve the defaults.
//
// Takes opts (...ServerOption) which tune the resolved telemetry defaults.
//
// Returns *grpc.Server which is the configured telemetry server.
func NewServer(opts ...ServerOption) *grpc.Server {
	return newServer(opts, nil)
}

// NewServerWithGRPCOptions is NewServer extended with raw grpc.ServerOption values
// (transport credentials, interceptors, keepalive). They apply after the telemetry
// defaults, so a caller-supplied option wins on conflict.
//
// Takes opts ([]ServerOption) which tune the resolved telemetry defaults.
// Takes grpcOpts (...grpc.ServerOption) which apply last and win on conflict.
//
// Returns *grpc.Server which is the configured telemetry server.
func NewServerWithGRPCOptions(opts []ServerOption, grpcOpts ...grpc.ServerOption) *grpc.Server {
	return newServer(opts, grpcOpts)
}

// newServer resolves the telemetry options and grpc options into a configured server,
// applying the FlatBuffers codec, size and stream caps, and the recovery interceptor.
//
// Takes opts ([]ServerOption) which tune the resolved telemetry defaults.
// Takes grpcOpts ([]grpc.ServerOption) which apply last and win on conflict.
//
// Returns *grpc.Server which is the configured telemetry server.
func newServer(opts []ServerOption, grpcOpts []grpc.ServerOption) *grpc.Server {
	o := serverOptions{
		maxReceiveMessageSize: defaultMaxReceiveMessageSize,
		maxConcurrentStreams:  defaultMaxConcurrentStreams,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&o)
		}
	}
	base := make([]grpc.ServerOption, 0, 4+len(grpcOpts))
	base = append(base,
		grpc.ForceServerCodec(Codec{}),
		grpc.MaxRecvMsgSize(o.maxReceiveMessageSize),
		grpc.MaxConcurrentStreams(o.maxConcurrentStreams),
		grpc.ChainStreamInterceptor(recoverStreamInterceptor),
	)
	return grpc.NewServer(append(base, grpcOpts...)...)
}
