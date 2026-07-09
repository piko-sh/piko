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

package span_collector_grpcfb

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

type sink struct {
	batches []*telemetry_grpcfb.Batch
	mu      sync.Mutex
}

func (s *sink) Ingest(stream telemetry_grpcfb.IngestStream) error {
	for {
		b, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		s.mu.Lock()
		s.batches = append(s.batches, b)
		s.mu.Unlock()
	}
	return stream.SendAndClose(&telemetry_grpcfb.IngestAck{OK: true})
}

func (s *sink) spans() []telemetry_grpcfb.Span {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []telemetry_grpcfb.Span
	for _, b := range s.batches {
		out = append(out, b.Spans...)
	}
	return out
}

func newSinkClient(t *testing.T) (*telemetry_grpcfb.Client, *sink, func()) {
	t.Helper()

	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{
		SiteID:        "site-x",
		APIKey:        "key-x",
		FlushSize:     1,
		FlushInterval: time.Hour,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	client.Start(ctx)

	drain := func() {
		require.NoError(t, client.Close(ctx))
		cancel()
		_ = conn.Close()
		srv.Stop()
	}
	return client, snk, drain
}

func endSpanWithAttributes(t *testing.T, col *Collector, serviceName string, attrs ...attribute.KeyValue) {
	t.Helper()

	opts := []sdktrace.TracerProviderOption{sdktrace.WithSpanProcessor(col)}
	if serviceName != "" {
		opts = append(opts, sdktrace.WithResource(resource.NewSchemaless(
			attribute.String("service.name", serviceName),
		)))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	t.Cleanup(func() {
		_ = tp.Shutdown(context.Background())
	})

	_, span := tp.Tracer("span_collector_grpcfb_test").Start(context.Background(), "GET /widgets")
	span.SetAttributes(attrs...)
	span.End()
}

func findAttr(span telemetry_grpcfb.Span, key string) (string, bool) {
	for _, kv := range span.Attributes {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

func TestStatusToStringLowercase(t *testing.T) {
	cases := []struct {
		name string
		want string
		code codes.Code
	}{
		{name: "Unset", code: codes.Unset, want: "unset"},
		{name: "Ok", code: codes.Ok, want: "ok"},
		{name: "Error", code: codes.Error, want: "error"},
		{name: "unknown", code: codes.Code(99), want: "unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, statusToString(tc.code))
		})
	}
}

func TestNilCollectorIsSafe(t *testing.T) {
	c := New(nil)

	assert.NotPanics(t, func() { c.OnStart(context.Background(), nil) })
	assert.NotPanics(t, func() { c.OnEnd(nil) })
	assert.NotPanics(t, func() { _ = c.Shutdown(context.Background()) })
	assert.NotPanics(t, func() { _ = c.ForceFlush(context.Background()) })

	var nilCol *Collector
	assert.NotPanics(t, func() { nilCol.OnEnd(nil) })
	assert.NotPanics(t, func() { _ = nilCol.Shutdown(context.Background()) })
	assert.NotPanics(t, func() { _ = nilCol.ForceFlush(context.Background()) })
}

func TestOnEndNilClientDoesNotEnqueue(t *testing.T) {
	col := New(nil)
	assert.NotPanics(t, func() {
		endSpanWithAttributes(t, col, "", attribute.String("http.method", "GET"))
	})
}

func TestOnEndRedactsSensitiveAttributesByDefault(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client)

	longValue := strings.Repeat("£", defaultMaxAttrValueLen)

	endSpanWithAttributes(t, col, "checkout",
		attribute.String("url.query", "token=abc123&user=alice"),
		attribute.String("db.statement", "SELECT * FROM users WHERE ssn = '123'"),
		attribute.String("http.request.header.authorization", "Bearer sk-secret"),
		attribute.String("http.method", "GET"),
		attribute.String("long.value", longValue),
	)

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	got := spans[0]

	assert.Equal(t, "checkout", got.Service)
	assert.Equal(t, "GET /widgets", got.Operation)

	urlQuery, ok := findAttr(got, "url.query")
	require.True(t, ok, "url.query attribute should be present")
	assert.Equal(t, redactedPlaceholder, urlQuery)

	dbStmt, ok := findAttr(got, "db.statement")
	require.True(t, ok, "db.statement attribute should be present")
	assert.Equal(t, redactedPlaceholder, dbStmt)

	authHeader, ok := findAttr(got, "http.request.header.authorization")
	require.True(t, ok, "authorization attribute should be present")
	assert.Equal(t, redactedPlaceholder, authHeader)

	method, ok := findAttr(got, "http.method")
	require.True(t, ok, "http.method attribute should pass through")
	assert.Equal(t, "GET", method)

	long, ok := findAttr(got, "long.value")
	require.True(t, ok, "long.value attribute should pass through (truncated)")
	assert.LessOrEqual(t, len(long), defaultMaxAttrValueLen, "value must be truncated to the byte cap")
	assert.True(t, utf8.ValidString(long), "truncated value must remain valid UTF-8")
	assert.NotEmpty(t, long, "truncation must keep a non-empty prefix")
}

func TestOnEndFoldsStatusDescriptionIntoAttributes(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client)

	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(col))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	_, span := tp.Tracer("span_collector_grpcfb_test").Start(context.Background(), "work")
	span.SetStatus(codes.Error, "boom: downstream timed out")
	span.End()

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	got := spans[0]

	assert.Equal(t, "error", got.Status)
	desc, ok := findAttr(got, "otel.status_description")
	require.True(t, ok, "status description should be folded into attributes")
	assert.Equal(t, "boom: downstream timed out", desc)
}

func TestOnEndPropagatesSpanIdentity(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client)

	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(col))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	tracer := tp.Tracer("span_collector_grpcfb_test")
	ctx, parent := tracer.Start(context.Background(), "parent", trace.WithSpanKind(trace.SpanKindServer))
	_, child := tracer.Start(ctx, "child", trace.WithSpanKind(trace.SpanKindClient))
	child.End()
	parent.End()

	drain()

	spans := snk.spans()
	require.Len(t, spans, 2)

	byOp := map[string]telemetry_grpcfb.Span{}
	for _, s := range spans {
		byOp[s.Operation] = s
	}

	parentSpan, ok := byOp["parent"]
	require.True(t, ok)
	assert.NotEmpty(t, parentSpan.TraceID)
	assert.NotEmpty(t, parentSpan.SpanID)
	assert.Empty(t, parentSpan.ParentID)
	assert.Equal(t, trace.SpanKindServer.String(), parentSpan.Kind)

	childSpan, ok := byOp["child"]
	require.True(t, ok)
	assert.Equal(t, parentSpan.TraceID, childSpan.TraceID)
	assert.Equal(t, parentSpan.SpanID, childSpan.ParentID)
	assert.Equal(t, trace.SpanKindClient.String(), childSpan.Kind)
}

func TestWithAttributeRedactorOverridesDefault(t *testing.T) {
	client, snk, drain := newSinkClient(t)

	passThrough := func(_, value string) (string, bool) { return value, true }
	col := New(client, WithAttributeRedactor(passThrough))

	endSpanWithAttributes(t, col, "",
		attribute.String("url.query", "token=abc123"),
		attribute.String("http.request.header.authorization", "Bearer sk-secret"),
	)

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	got := spans[0]

	urlQuery, ok := findAttr(got, "url.query")
	require.True(t, ok)
	assert.Equal(t, "token=abc123", urlQuery, "pass-through redactor must leave the value raw")

	authHeader, ok := findAttr(got, "http.request.header.authorization")
	require.True(t, ok)
	assert.Equal(t, "Bearer sk-secret", authHeader, "pass-through redactor must leave the value raw")
}

func TestWithAttributeRedactorCanDropAttributes(t *testing.T) {
	client, snk, drain := newSinkClient(t)

	dropTokens := func(key, value string) (string, bool) {
		if strings.Contains(key, "token") {
			return "", false
		}
		return value, true
	}
	col := New(client, WithAttributeRedactor(dropTokens))

	endSpanWithAttributes(t, col, "",
		attribute.String("session.token", "abc"),
		attribute.String("http.method", "POST"),
	)

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	got := spans[0]

	_, ok := findAttr(got, "session.token")
	assert.False(t, ok, "dropped attribute must not be streamed")

	method, ok := findAttr(got, "http.method")
	require.True(t, ok)
	assert.Equal(t, "POST", method)
}

func TestWithAttributeRedactorNilIsIgnored(t *testing.T) {
	col := New(nil, WithAttributeRedactor(nil))
	assert.NotNil(t, col.redact, "nil redactor option must not clear the default")
}

func TestServiceNameFromResource(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		client, snk, drain := newSinkClient(t)
		col := New(client)
		endSpanWithAttributes(t, col, "billing", attribute.String("http.method", "GET"))
		drain()

		spans := snk.spans()
		require.Len(t, spans, 1)
		assert.Equal(t, "billing", spans[0].Service)
	})

	t.Run("absent", func(t *testing.T) {
		client, snk, drain := newSinkClient(t)
		col := New(client)

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(col),
			sdktrace.WithResource(resource.Empty()),
		)
		t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })
		_, span := tp.Tracer("span_collector_grpcfb_test").Start(context.Background(), "work")
		span.End()
		drain()

		spans := snk.spans()
		require.Len(t, spans, 1)
		assert.Empty(t, spans[0].Service)
	})
}

func TestShutdownSharedClientIsNoOp(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client)

	require.NoError(t, col.Shutdown(context.Background()))

	endSpanWithAttributes(t, col, "", attribute.String("http.method", "GET"))
	drain()

	assert.Len(t, snk.spans(), 1, "shared client must remain usable after collector Shutdown")
}

func TestShutdownOwnedClientClosesIt(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	col, err := Dial("passthrough:///bufnet",
		telemetry_grpcfb.Config{SiteID: "site-x", APIKey: "key-x", FlushSize: 1, FlushInterval: time.Hour},
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)
	assert.True(t, col.ownsClient, "Dial-built collector must own its client")

	require.NoError(t, col.Shutdown(context.Background()))
	require.NoError(t, col.Shutdown(context.Background()))
}

func TestWithMaxAttributesCapsAndMarksDropped(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithMaxAttributes(2))

	endSpanWithAttributes(t, col, "",
		attribute.String("a", "1"),
		attribute.String("b", "2"),
		attribute.String("c", "3"),
		attribute.String("d", "4"),
	)

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	got := spans[0]

	kept := 0
	for _, key := range []string{"a", "b", "c", "d"} {
		if _, ok := findAttr(got, key); ok {
			kept++
		}
	}
	assert.Equal(t, 2, kept, "only maxAttributes attributes may be copied")

	dropped, ok := findAttr(got, attrsDroppedKey)
	require.True(t, ok, "dropped marker must be present when the cap is hit")
	assert.Equal(t, "2", dropped, "dropped marker must record the dropped count")
}

func TestWithMaxAttributesNoMarkerWhenUnderCap(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithMaxAttributes(8))

	endSpanWithAttributes(t, col,
		"",
		attribute.String("a", "1"),
		attribute.String("b", "2"),
	)

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	_, ok := findAttr(spans[0], attrsDroppedKey)
	assert.False(t, ok, "no dropped marker when the cap is not reached")
}

func TestWithMaxAttributesNonPositiveIsIgnored(t *testing.T) {
	col := New(nil, WithMaxAttributes(0), WithMaxAttributes(-5))
	assert.Equal(t, defaultMaxAttributes, col.maxAttributes, "non-positive cap must keep the default")
}

func TestWithMaxAttributeValueLenCapsAndMarksTruncated(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client, WithMaxAttributeValueLen(4))

	endSpanWithAttributes(t, col, "",
		attribute.String("note", "abcdefghij"),
	)

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	got := spans[0]

	note, ok := findAttr(got, "note")
	require.True(t, ok)
	assert.LessOrEqual(t, len(note), 4, "value must be capped to the configured length")
	assert.True(t, utf8.ValidString(note), "truncated value must stay valid UTF-8")

	truncated, ok := findAttr(got, attrTruncatedKey)
	require.True(t, ok, "truncation marker must be present when a value is clipped")
	assert.Equal(t, "1", truncated, "truncation marker must record the clipped count")
}

func TestOnEndReCapsCustomRedactorValue(t *testing.T) {
	client, snk, drain := newSinkClient(t)

	balloon := func(_, _ string) (string, bool) { return strings.Repeat("x", 100), true }
	col := New(client, WithAttributeRedactor(balloon), WithMaxAttributeValueLen(8))

	endSpanWithAttributes(t, col, "", attribute.String("k", "v"))

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	got := spans[0]

	value, ok := findAttr(got, "k")
	require.True(t, ok)
	assert.LessOrEqual(t, len(value), 8, "custom redactor output must be re-capped in OnEnd")

	_, ok = findAttr(got, attrTruncatedKey)
	assert.True(t, ok, "re-capping a redactor value must be observable")
}

func TestOnEndNoTruncationMarkerWhenValuesFit(t *testing.T) {
	client, snk, drain := newSinkClient(t)
	col := New(client)

	endSpanWithAttributes(t, col, "", attribute.String("http.method", "GET"))

	drain()

	spans := snk.spans()
	require.Len(t, spans, 1)
	_, ok := findAttr(spans[0], attrTruncatedKey)
	assert.False(t, ok, "no truncation marker when every value fits")
}

func TestWithMaxAttributeValueLenNonPositiveIsIgnored(t *testing.T) {
	col := New(nil, WithMaxAttributeValueLen(0), WithMaxAttributeValueLen(-1))
	assert.Equal(t, defaultMaxAttrValueLen, col.maxAttrValueLen, "non-positive length must keep the default")
}

func TestForceFlushQueuesPartialBatch(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := telemetry_grpcfb.NewServer()
	snk := &sink{}
	telemetry_grpcfb.RegisterServer(srv, snk)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(telemetry_grpcfb.Codec{})),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{
		SiteID: "site-x", APIKey: "key-x", FlushSize: 1000, FlushInterval: time.Hour,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client.Start(ctx)
	col := New(client)

	endSpanWithAttributes(t, col, "", attribute.String("http.method", "GET"))

	require.NoError(t, col.ForceFlush(ctx))
	require.NoError(t, client.Close(ctx))

	assert.Len(t, snk.spans(), 1, "ForceFlush must queue the buffered span for sending")
}
