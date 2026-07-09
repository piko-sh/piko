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

package log_collector_grpcfb

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"piko.sh/piko/wdk/telemetry/telemetry_grpcfb"
)

func messagesOf(crumbs []breadcrumb) []string {
	out := make([]string, 0, len(crumbs))
	for _, c := range crumbs {
		out = append(out, c.message)
	}
	return out
}

func TestBreadcrumbRing_AddRecentOrderingAndEviction(t *testing.T) {
	r := newBreadcrumbRing(3)
	for _, m := range []string{"a", "b", "c", "d", "e"} {
		r.add(breadcrumb{message: m})
	}

	got := r.recent(10, "")
	assert.Equal(t, []string{"c", "d", "e"}, messagesOf(got), "ring keeps the newest entries oldest-first")

	assert.Equal(t, []string{"d", "e"}, messagesOf(r.recent(2, "")), "limit returns the newest entries")

	assert.Empty(t, r.recent(0, ""), "a zero limit returns no entries")
}

func TestBreadcrumbRing_CapacityOne(t *testing.T) {
	r := newBreadcrumbRing(1)
	r.add(breadcrumb{message: "first"})
	r.add(breadcrumb{message: "second"})
	assert.Equal(t, []string{"second"}, messagesOf(r.recent(5, "")), "capacity-1 ring keeps only the newest")
}

func TestBreadcrumbRing_TraceIDFilter(t *testing.T) {
	r := newBreadcrumbRing(8)
	r.add(breadcrumb{message: "t1-a", traceID: "t1"})
	r.add(breadcrumb{message: "t2-a", traceID: "t2"})
	r.add(breadcrumb{message: "t1-b", traceID: "t1"})
	r.add(breadcrumb{message: "untraced"})

	assert.Equal(t, []string{"t1-a", "t1-b"}, messagesOf(r.recent(10, "t1")),
		"non-empty traceID returns only same-trace entries")

	assert.Equal(t, []string{"t1-a", "t2-a", "t1-b", "untraced"}, messagesOf(r.recent(10, "")),
		"empty traceID returns the global trail")

	r.add(breadcrumb{message: "t1-c", traceID: "t1"})
	assert.Equal(t, []string{"t1-b", "t1-c"}, messagesOf(r.recent(2, "t1")),
		"limit keeps the newest matching entries")
}

func TestBreadcrumbRing_NilAndEmpty(t *testing.T) {
	var nilRing *breadcrumbRing
	assert.NotPanics(t, func() { nilRing.add(breadcrumb{message: "x"}) }, "add on a nil ring is a no-op")
	assert.Nil(t, nilRing.recent(5, ""), "recent on a nil ring returns nil")

	empty := newBreadcrumbRing(4)
	assert.Empty(t, empty.recent(5, ""), "an empty ring returns no entries")
}

func TestBreadcrumbRing_MessageBounded(t *testing.T) {
	r := newBreadcrumbRing(2)
	long := strings.Repeat("é", maxBreadcrumbMsg)
	r.add(breadcrumb{message: long})

	got := r.recent(1, "")
	require.Len(t, got, 1)
	assert.LessOrEqual(t, len(got[0].message), maxBreadcrumbMsg, "retained message is byte-bounded")
	assert.True(t, utf8.ValidString(got[0].message), "truncated message stays valid UTF-8")
}

func TestBreadcrumbsJSON_ShapeAndEmpty(t *testing.T) {
	assert.Equal(t, "", breadcrumbsJSON(nil), "no crumbs serialises to the empty string")

	crumbs := []breadcrumb{
		{tsMs: time.Unix(0, 0).UnixMilli(), level: "INFO", logger: "store", message: "loading"},
		{tsMs: time.Unix(0, 0).UnixMilli(), level: "WARN", logger: "cache", message: "miss"},
	}
	out := breadcrumbsJSON(crumbs)
	require.NotEmpty(t, out)

	var decoded []map[string]string
	require.NoError(t, json.Unmarshal([]byte(out), &decoded), "breadcrumbs JSON must be a valid array of objects")
	require.Len(t, decoded, 2)
	assert.Equal(t, "store", decoded[0]["category"], "logger maps to category")
	assert.Equal(t, "loading", decoded[0]["message"])
	assert.Equal(t, "INFO", decoded[0]["level"])
	assert.NotEmpty(t, decoded[0]["ts"], "ts is rendered")
	assert.Equal(t, "cache", decoded[1]["category"])
}

type stackValuer struct {
	frames []string
}

func (s stackValuer) LogValue() slog.Value {
	return slog.AnyValue(s.frames)
}

func TestResolveStack_Variants(t *testing.T) {
	frames := []string{"\tpkg.Func file.go:10", "\tpkg.Other other.go:20"}

	fromValuer := resolveStack(slog.AnyValue(stackValuer{frames: frames}), "fallback")
	assert.Equal(t, "\tpkg.Func file.go:10\n\tpkg.Other other.go:20", fromValuer)
	assert.NotContains(t, fromValuer, "{0x", "resolved frames must not be a struct dump")

	fromAny := resolveStack(slog.AnyValue([]any{"one", 2}), "fallback")
	assert.Equal(t, "one\n2", fromAny)

	assert.Equal(t, "raw stack text", resolveStack(slog.StringValue("raw stack text"), "raw stack text"))

	assert.Equal(t, "fallback", resolveStack(slog.IntValue(7), "fallback"))
}

func TestAddAttr_StackTraceFieldResolved(t *testing.T) {
	h := &Handler{}
	r := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "boom", 0)
	r.AddAttrs(slog.Any("stack_trace", stackValuer{frames: []string{"\tpkg.Func file.go:10"}}))

	var line telemetry_grpcfb.LogLine
	fields, extracted := h.collectFields(&r, &line)

	assert.Equal(t, "\tpkg.Func file.go:10", extracted.stack, "dedicated stack is the resolved frames")

	var stackField string
	var found bool
	for _, f := range fields {
		if f.Key == "stack_trace" {
			stackField, found = f.Value, true
		}
	}
	require.True(t, found, "stack_trace is also kept as a Context field")
	assert.Equal(t, extracted.stack, stackField, "Context field matches the resolved stack, not a struct dump")
	assert.NotContains(t, stackField, "{0x", "Context field must not leak a heap-pointer struct dump")
}

func TestHandle_AttachesBreadcrumbsToError(t *testing.T) {
	snk, lis, stop := startSink(t)
	defer stop()
	conn := dialSink(t, lis)
	defer conn.Close()

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{SiteID: "s", FlushInterval: time.Hour})
	client.Start(context.Background())
	h := New(client)

	for _, m := range []string{"step one", "step two"} {
		rec := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, m, 0)
		require.NoError(t, h.Handle(context.Background(), rec))
	}
	errRec := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "boom", 0)
	require.NoError(t, h.Handle(context.Background(), errRec))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	_, errs := snk.snapshot()
	require.Len(t, errs, 1)
	crumbs := errs[0].BreadcrumbsJSON
	require.NotEmpty(t, crumbs, "an error carries the trail of lines leading up to it")
	assert.Contains(t, crumbs, "step one")
	assert.Contains(t, crumbs, "step two")
	assert.NotContains(t, crumbs, "boom", "an error is not its own breadcrumb")
}

func TestHandle_BreadcrumbsAreTraceScoped(t *testing.T) {
	snk, lis, stop := startSink(t)
	defer stop()
	conn := dialSink(t, lis)
	defer conn.Close()

	client := telemetry_grpcfb.New(conn, telemetry_grpcfb.Config{SiteID: "s", FlushInterval: time.Hour})
	client.Start(context.Background())
	h := New(client)

	mine := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "my-request-line", 0)
	mine.AddAttrs(slog.String("trace_id", "trace-a"))
	require.NoError(t, h.Handle(context.Background(), mine))

	other := slog.NewRecord(time.Unix(0, 0), slog.LevelInfo, "other-request-line", 0)
	other.AddAttrs(slog.String("trace_id", "trace-b"))
	require.NoError(t, h.Handle(context.Background(), other))

	errRec := slog.NewRecord(time.Unix(0, 0), slog.LevelError, "failed", 0)
	errRec.AddAttrs(slog.String("trace_id", "trace-a"))
	require.NoError(t, h.Handle(context.Background(), errRec))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, client.Close(ctx))

	_, errs := snk.snapshot()
	require.Len(t, errs, 1)
	crumbs := errs[0].BreadcrumbsJSON
	assert.Contains(t, crumbs, "my-request-line", "a traced error carries its own trace's trail")
	assert.NotContains(t, crumbs, "other-request-line", "breadcrumbs from other traces are excluded")
}
